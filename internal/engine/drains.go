package engine

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
)

// ErrNoDrainJournal is returned when the session keeps no drain journal (Options.Drains is nil):
// Cogpit does not start a drain it could not offer to finish after a crash.
var ErrNoDrainJournal = errors.New("this Cogpit session keeps no drain journal, so it cannot drain nodes")

// Drains tracks a session's drains (D-11): the journal that keeps them across restarts, and the
// drains running in this process with their latest progress. A drain is recorded before its node
// is cordoned and removed once every pod is evicted; one that fails keeps its entry, and so does
// one a crash interrupts. When a context is connected, its pending drains are offered with
// EventDrains: the GUI shows a notice (finish the drain, uncordon the node, or dismiss), agents
// read them with node.pendingDrains. An entry whose node no longer exists is dropped with a note.
//
// Create it with NewDrains and pass it as Options.Drains; it belongs to one session.
type Drains struct {
	journal DrainJournal

	mu       sync.Mutex
	s        *Session
	running  map[drainKey]*cluster.DrainProgress
	lastRev  uint64
	lastConn ConnectionState
}

// NewDrains returns the tracker for a journal.
func NewDrains(j DrainJournal) *Drains {
	return &Drains{journal: j, running: map[drainKey]*cluster.DrainProgress{}}
}

// Drains returns the session's drain tracker, or nil when it keeps none.
func (s *Session) Drains() *Drains { return s.opts.Drains }

// attach binds the tracker to its session and follows the session's connections, to offer the
// pending drains of each context it connects. It is nil-safe.
func (d *Drains) attach(s *Session) {
	if d == nil {
		return
	}
	d.mu.Lock()
	d.s = s
	d.mu.Unlock()
	s.Subscribe(d.onEvent)
}

// onEvent runs on the goroutine that changed the UI state: it must not block.
func (d *Drains) onEvent(ev Event) {
	if ev.Type != EventUIState || ev.UIState == nil {
		return
	}
	u := ev.UIState
	d.mu.Lock()
	if u.Revision <= d.lastRev { // an older state delivered late
		d.mu.Unlock()
		return
	}
	d.lastRev = u.Revision
	prev := d.lastConn
	d.lastConn = u.Connection.State
	s := d.s
	d.mu.Unlock()
	if s == nil || u.Connection.State == prev {
		return
	}
	switch u.Connection.State {
	case Connected:
		if u.Connection.Demo {
			d.changed("")
			return
		}
		ctxName := u.Context
		s.goSafe("unfinished drains", func() { d.offer(s, ctxName) })
	default:
		d.changed("") // the previous cluster's drains no longer apply
	}
}

// offer reads the journal for a freshly connected context, drops the entries whose node no longer
// exists, and announces the rest.
func (d *Drains) offer(s *Session, ctxName string) {
	entries, err := d.journal.List()
	if err != nil {
		s.log.Error("engine: the drain journal is unreadable", "err", err)
		s.emit(Event{Type: EventToast, Level: drainsLevelError, Message: "Unfinished drains cannot be offered: " + err.Error()})
		d.changed("")
		return
	}
	var pending []PendingDrain
	for _, e := range entries {
		if e.Context == ctxName && !d.Running(e.Context, e.Node) {
			pending = append(pending, e)
		}
	}
	if len(pending) > 0 {
		pending = d.dropMissingNodes(s, ctxName, pending)
	}
	msg := ""
	switch n := len(pending); {
	case n == 1:
		msg = fmt.Sprintf("The drain of node %s in context %s did not finish", pending[0].Node, ctxName)
	case n > 1:
		msg = fmt.Sprintf("%d drains in context %s did not finish", n, ctxName)
	}
	if b := s.Backend(); b != nil && b.Context == ctxName {
		d.changed(msg)
	}
}

// offerWait bounds how long the offer waits for the node list it checks entries against.
const offerWait = syncTimeout

// dropMissingNodes removes the entries whose node the connected cluster no longer has, with a
// toast for each, and returns the others. Without a synced node list (no permission to list
// nodes, a slow cluster, another cluster connected meanwhile) nothing is dropped.
func (d *Drains) dropMissingNodes(s *Session, ctxName string, pending []PendingDrain) []PendingDrain {
	b := s.Backend()
	if b == nil || b.Context != ctxName {
		return pending
	}
	p, ok := b.Profile(nodeKind)
	if !ok {
		return pending
	}
	h := b.Cache.Acquire(cache.Scope{GVR: p.GVR()})
	defer h.Release()
	deadline := time.Now().Add(offerWait)
	for !h.HasSynced() {
		if s.Backend() != b || time.Now().After(deadline) || s.isClosed() {
			s.log.Warn("engine: unfinished drains offered without checking their nodes", "context", ctxName, "err", h.Err())
			return pending
		}
		time.Sleep(20 * time.Millisecond)
	}
	if s.Backend() != b {
		return pending
	}
	exists := map[string]bool{}
	for _, n := range h.List() {
		exists[n.GetName()] = true
	}
	kept := pending[:0]
	for _, e := range pending {
		if exists[e.Node] {
			kept = append(kept, e)
			continue
		}
		if err := d.journal.Remove(e.Context, e.Node); err != nil {
			s.log.Error("engine: dropping a drain of a missing node", "node", e.Node, "err", err)
			kept = append(kept, e)
			continue
		}
		s.log.Info("engine: dropped the unfinished drain of a node that no longer exists", "context", e.Context, "node", e.Node)
		s.emit(Event{Type: EventToast, Level: "info", Message: fmt.Sprintf(
			"Cogpit dropped the unfinished drain of node %s (context %s): the node no longer exists.", e.Node, e.Context)})
	}
	return kept
}

// isClosed reports whether the session was closed.
func (s *Session) isClosed() bool {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.closed
}

// nodeKind is the kind key of nodes.
const nodeKind = "nodes"

// drainsLevelError is the toast level of a journal that cannot be read.
const drainsLevelError = "error"

// Begin records a drain before its node is cordoned. It fails, and the drain must not start, when
// a drain of the node runs already or the journal cannot be written.
func (d *Drains) Begin(p PendingDrain) error {
	if d == nil {
		return ErrNoDrainJournal
	}
	k := p.key()
	d.mu.Lock()
	if _, ok := d.running[k]; ok {
		d.mu.Unlock()
		return fmt.Errorf("a drain of node %s is running already", p.Node)
	}
	d.running[k] = &cluster.DrainProgress{Node: p.Node}
	d.mu.Unlock()

	p.State, p.Error = DrainRunning, ""
	if err := d.journal.Put(p); err != nil {
		d.mu.Lock()
		delete(d.running, k)
		d.mu.Unlock()
		return fmt.Errorf("the drain was not started, because Cogpit could not record it first: %w", err)
	}
	d.changed("")
	return nil
}

// Progress records a running drain's latest step and announces it.
func (d *Drains) Progress(context string, p cluster.DrainProgress) {
	if d == nil {
		return
	}
	k := drainKey{context, p.Node}
	d.mu.Lock()
	if _, ok := d.running[k]; ok {
		cp := p
		d.running[k] = &cp
	}
	d.mu.Unlock()
	d.changed("")
}

// End records a drain's outcome: a finished drain's entry is removed; a failed one is kept as
// failed with its error, since the node stays cordoned and partly drained. It returns the
// journal's error, if any.
func (d *Drains) End(p PendingDrain, drainErr error) error {
	if d == nil {
		return ErrNoDrainJournal
	}
	k := p.key()
	d.mu.Lock()
	delete(d.running, k)
	d.mu.Unlock()
	var err error
	if drainErr == nil {
		err = d.journal.Remove(p.Context, p.Node)
	} else {
		p.State, p.Error = DrainFailed, drainErr.Error()
		err = d.journal.Put(p)
	}
	d.changed("")
	return err
}

// Forget removes a pending drain without touching the cluster: Dismiss, or after an uncordon.
// A drain running in this process is not forgotten.
func (d *Drains) Forget(context, node string) error {
	if d == nil {
		return ErrNoDrainJournal
	}
	if d.Running(context, node) {
		return fmt.Errorf("the drain of node %s is running", node)
	}
	if err := d.journal.Remove(context, node); err != nil {
		return err
	}
	d.changed("")
	return nil
}

// Running reports whether a drain of the node runs in this process.
func (d *Drains) Running(context, node string) bool {
	if d == nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	_, ok := d.running[drainKey{context, node}]
	return ok
}

// Pending returns the journal entry for a context and node, if there is one.
func (d *Drains) Pending(context, node string) (PendingDrain, bool, error) {
	if d == nil {
		return PendingDrain{}, false, ErrNoDrainJournal
	}
	entries, err := d.journal.List()
	if err != nil {
		return PendingDrain{}, false, err
	}
	for _, e := range entries {
		if e.Context == context && e.Node == node {
			return e, true, nil
		}
	}
	return PendingDrain{}, false, nil
}

// changed emits EventDrains.
func (d *Drains) changed(msg string) {
	d.mu.Lock()
	s := d.s
	d.mu.Unlock()
	if s != nil {
		s.emit(Event{Type: EventDrains, Message: msg})
	}
}

func (d *Drains) progressOf(k drainKey) *cluster.DrainProgress {
	d.mu.Lock()
	defer d.mu.Unlock()
	if p, ok := d.running[k]; ok {
		cp := *p
		return &cp
	}
	return nil
}

// Statuses of a pending drain, as PendingDrains reports them.
const (
	DrainStatusRunning     = "running"     // runs in this Cogpit; Progress has its latest step
	DrainStatusInterrupted = "interrupted" // an earlier run of Cogpit ended before the drain did
	DrainStatusFailed      = "failed"      // ended with an error; the node stays cordoned
)

// Node states of a pending drain's node, from the cache.
const (
	DrainNodeCordoned    = "cordoned"
	DrainNodeSchedulable = "schedulable"
	DrainNodeMissing     = "missing" // not in the cache's node list
	DrainNodeUnknown     = "unknown" // another context, or the node list is not in the cache
)

// PendingDrainInfo is a pending drain as node.pendingDrains reports it.
type PendingDrainInfo struct {
	PendingDrain
	Status    string                 `json:"status"`
	NodeState string                 `json:"nodeState"`
	Progress  *cluster.DrainProgress `json:"progress,omitempty"`
}

// PendingDrains returns the pending drains of the connected context (all: of every context),
// oldest first, with their status and their node's state from the cache. It reads the journal
// and the cache, never the API server.
func (s *Session) PendingDrains(all bool) ([]PendingDrainInfo, error) {
	d := s.opts.Drains
	if d == nil {
		return nil, ErrNoDrainJournal
	}
	b := s.Backend()
	if b == nil && !all {
		return nil, ErrNotConnected
	}
	entries, err := d.journal.List()
	if err != nil {
		return nil, err
	}
	out := make([]PendingDrainInfo, 0, len(entries))
	for _, e := range entries {
		if !all && e.Context != b.Context {
			continue
		}
		info := PendingDrainInfo{PendingDrain: e, NodeState: DrainNodeUnknown}
		switch {
		case d.Running(e.Context, e.Node):
			info.Status, info.Progress = DrainStatusRunning, d.progressOf(e.key())
		case e.State == DrainFailed:
			info.Status = DrainStatusFailed
		default:
			info.Status = DrainStatusInterrupted
		}
		if b != nil && e.Context == b.Context {
			info.NodeState = s.nodeState(e.Node)
		}
		out = append(out, info)
	}
	slices.SortFunc(out, func(a, b PendingDrainInfo) int {
		if c := a.Started.Compare(b.Started); c != 0 {
			return c
		}
		return cmp.Compare(a.Context+"/"+a.Node, b.Context+"/"+b.Node)
	})
	return out, nil
}

// nodeState reads a node's schedulability from the cache, without waiting.
func (s *Session) nodeState(node string) string {
	obj, err := s.Lookup(model.ObjectRef{Kind: nodeKind, Name: node})
	switch {
	case errors.Is(err, ErrNotFound):
		return DrainNodeMissing
	case err != nil:
		return DrainNodeUnknown
	}
	if v, _ := model.NestedBool(obj.Object, "spec", "unschedulable"); v {
		return DrainNodeCordoned
	}
	return DrainNodeSchedulable
}
