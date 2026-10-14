package engine

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Reading the audit log back (G17; D-2: "an audit log the user can read"): the window's Audit Log
// dock tab and agents (audit.list) read the entries of Options.AuditPath, the file the app and
// cogpit-cli append to, newest first, from its last MaxAuditRead bytes. What is read is what was
// written: commands, their parameters with Secret values, Helm values and proxy passwords already
// redacted (auditParams), contexts and outcomes; the parameters are redacted once more on the way
// out, in case a line came from an older Cogpit or another hand. Nothing is read from a cluster.

// Bounds of the audit log's reading.
const (
	// MaxAuditRead is how much of the file's end is read: older entries are not listed.
	MaxAuditRead = 4 << 20
	// DefaultAuditEntries and MaxAuditEntries bound one page of entries.
	DefaultAuditEntries = 100
	MaxAuditEntries     = 1000
	// maxAuditParams bounds the parameters an entry carries; larger ones are summed up.
	maxAuditParams = 2048
	// maxAuditObjects bounds the objects an entry names; ObjectCount counts them all.
	maxAuditObjects = 20
)

// Origin filters of the audit log, besides the origins themselves.
const (
	AuditAllOrigins = ""       // every entry
	AuditAgents     = "agents" // external agents and the built-in agent
)

// EventAudit: a line was written to the audit log; re-read it with audit.list. Coalesced to at
// most one per auditEventDelay.
const EventAudit EventType = "audit"

// DockAudit: the Audit Log, the audit log's entries newest first (audit.open).
const DockAudit DockType = "audit"

// AuditTab is the Audit Log tab's state: which origins it lists.
type AuditTab struct {
	// Origin is "" (all), "agents", or one origin: gui, cli, externalAgent, builtInAgent, deeplink.
	Origin string `json:"origin,omitempty"`
}

// ErrNoAuditLog says the session keeps no audit log to read.
var ErrNoAuditLog = errors.New("this Cogpit keeps no audit log file to read")

// AuditItem is one entry of the audit log as audit.list returns it.
type AuditItem struct {
	Time    time.Time `json:"time"`
	Origin  Origin    `json:"origin"`
	Command string    `json:"command"`
	// Label is the command's label in this Cogpit's registry; empty for an action that is no
	// command (the MCP switch) or a command it does not know.
	Label   string `json:"label,omitempty"`
	Context string `json:"context,omitempty"`
	// Phase is "requested" for the line a write gets before it runs, "done" for the outcome.
	Phase string `json:"phase"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
	// Objects are the objects the parameters name (ref, refs), at most 20 of ObjectCount.
	Objects     []model.ObjectRef `json:"objects,omitempty"`
	ObjectCount int               `json:"objectCount,omitempty"`
	// Params are the parameters as logged (redacted); a summary when larger than 2 KiB.
	Params json.RawMessage `json:"params,omitempty"`
	// ReadsLikeUser marks a call of the built-in agent over a private connection, whose reads
	// returned what the window shows (AuditEntry.ReadsLikeUser).
	ReadsLikeUser bool `json:"readsLikeUser,omitempty"`
}

// AuditList is a page of the audit log's entries, newest first.
type AuditList struct {
	// Origin is the filter applied.
	Origin string `json:"origin,omitempty"`
	// Matched counts the entries read that pass the filter; Offset and Returned place the page.
	Matched  int         `json:"matched"`
	Offset   int         `json:"offset"`
	Returned int         `json:"returned"`
	Entries  []AuditItem `json:"entries"`
	// Partial says the file is longer than what was read (its last 4 MiB): older entries are not
	// listed. Skipped counts lines that are no entry (a line being written, a damaged one).
	Partial bool `json:"partial,omitempty"`
	Skipped int  `json:"skipped,omitempty"`
}

// CheckAuditOrigin checks an origin filter.
func CheckAuditOrigin(origin string) error {
	switch Origin(origin) {
	case AuditAllOrigins, AuditAgents, OriginGUI, OriginCLI, OriginExternalAgent, OriginBuiltInAgent, OriginDeepLink:
		return nil
	}
	return fmt.Errorf("origin %q: one of agents, %s, %s, %s, %s, %s, or empty for all", origin, OriginGUI, OriginCLI, OriginExternalAgent, OriginBuiltInAgent, OriginDeepLink)
}

func auditOriginMatches(filter string, o Origin) bool {
	switch filter {
	case AuditAllOrigins:
		return true
	case AuditAgents:
		return o.IsAgent()
	}
	return string(o) == filter
}

// HasAuditLog reports whether the session can read its audit log.
func (s *Session) HasAuditLog() bool { return s.opts.AuditPath != "" }

// AuditList reads a page of the audit log's entries, newest first, of the origins the filter
// keeps. A file that does not exist yet has no entries.
func (s *Session) AuditList(origin string, offset, limit int) (AuditList, error) {
	if err := CheckAuditOrigin(origin); err != nil {
		return AuditList{}, err
	}
	switch {
	case offset < 0:
		return AuditList{}, fmt.Errorf("offset %d is negative", offset)
	case limit < 0:
		return AuditList{}, fmt.Errorf("limit %d is negative", limit)
	case limit == 0:
		limit = DefaultAuditEntries
	}
	limit = min(limit, MaxAuditEntries)
	if s.opts.AuditPath == "" {
		return AuditList{}, ErrNoAuditLog
	}
	data, partial, err := readTail(s.opts.AuditPath, MaxAuditRead)
	if err != nil {
		return AuditList{}, fmt.Errorf("the audit log: %w", err)
	}
	out := AuditList{Origin: origin, Offset: offset, Partial: partial, Entries: []AuditItem{}}
	lines := bytes.Split(data, []byte{'\n'})
	for i := len(lines) - 1; i >= 0; i-- { // the file is in time order: the newest last
		line := bytes.TrimSpace(lines[i])
		if len(line) == 0 {
			continue
		}
		var e AuditEntry
		if json.Unmarshal(line, &e) != nil || e.Command == "" {
			out.Skipped++
			continue
		}
		if !auditOriginMatches(origin, e.Origin) {
			continue
		}
		out.Matched++
		if out.Matched <= offset || len(out.Entries) >= limit {
			continue
		}
		out.Entries = append(out.Entries, s.auditItem(e))
	}
	out.Returned = len(out.Entries)
	return out, nil
}

// readTail reads at most limit bytes from the end of the file, from the start of a line; partial
// says it left the file's beginning out.
func readTail(path string, limit int64) (data []byte, partial bool, err error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, false, err
	}
	start := max(fi.Size()-limit, 0)
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return nil, false, err
	}
	data, err = io.ReadAll(io.LimitReader(f, limit))
	if err != nil {
		return nil, false, err
	}
	if start > 0 { // began inside a line: drop it
		if i := bytes.IndexByte(data, '\n'); i >= 0 {
			data = data[i+1:]
		} else {
			data = nil
		}
	}
	return data, start > 0, nil
}

// auditItem is an entry as listed: its objects from the parameters, its parameters redacted again
// and bounded, the command's label.
func (s *Session) auditItem(e AuditEntry) AuditItem {
	it := AuditItem{Time: e.Time, Origin: e.Origin, Command: e.Command, Context: e.Context, Phase: e.Phase, OK: e.OK, Error: e.Error,
		ReadsLikeUser: e.ReadsLikeUser}
	if c, ok := s.reg.Get(e.Command); ok {
		it.Label = c.Label
	}
	if len(e.Params) > 0 && json.Valid(e.Params) {
		p := auditParams(e.Params)
		if len(p) > maxAuditParams {
			p, _ = json.Marshal("<" + model.FormatBytes(int64(len(p))) + " of parameters; not shown>")
		}
		it.Params = p
		it.Objects, it.ObjectCount = auditObjects(e.Params)
	}
	return it
}

// auditObjects returns the objects a command's parameters name: its ref, its refs.
func auditObjects(raw json.RawMessage) ([]model.ObjectRef, int) {
	var p struct {
		Ref  *model.ObjectRef  `json:"ref"`
		Refs []model.ObjectRef `json:"refs"`
	}
	if json.Unmarshal(raw, &p) != nil {
		return nil, 0
	}
	var all []model.ObjectRef
	if p.Ref != nil && p.Ref.Name != "" {
		all = append(all, *p.Ref)
	}
	for _, r := range p.Refs {
		if r.Name != "" {
			all = append(all, r)
		}
	}
	return slices.Clip(all[:min(len(all), maxAuditObjects)]), len(all)
}

// ObjectText names an object as the Audit Log shows it: "pods shop/web-0", "nodes worker-1".
func ObjectText(r model.ObjectRef) string {
	name := r.Name
	if r.Namespace != "" {
		name = r.Namespace + "/" + name
	}
	return strings.TrimSpace(r.Kind + " " + name)
}

// auditEventDelay coalesces EventAudit: the window reads the log at most this often while agents
// work.
const auditEventDelay = 250 * time.Millisecond

// auditNotifier is the session's audit writer: each line written also schedules one EventAudit.
type auditNotifier struct {
	w       io.Writer
	s       *Session
	pending atomic.Bool
}

// notifyingAudit wraps the session's audit writer so that the window learns of new lines; nil stays
// nil.
func (s *Session) notifyingAudit(w io.Writer) io.Writer {
	if w == nil {
		return nil
	}
	return &auditNotifier{w: w, s: s}
}

func (n *auditNotifier) Write(p []byte) (int, error) {
	written, err := n.w.Write(p)
	if n.pending.CompareAndSwap(false, true) {
		// Not here: Write runs under the audit lock, and a subscriber must never wait for it.
		time.AfterFunc(auditEventDelay, func() {
			n.pending.Store(false)
			n.s.emit(Event{Type: EventAudit})
		})
	}
	return written, err
}
