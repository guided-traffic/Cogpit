package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// DrainSettings are a drain's options as node.drain takes them. Every zero value is the safe
// default, except IgnoreDaemonSets, which is a pointer so that leaving it out means true.
// Lens's drain defaults are not captured; these are Cogpit's (inferred).
type DrainSettings struct {
	IgnoreDaemonSets   *bool  `json:"ignoreDaemonSets,omitempty" jsonschema:"leave DaemonSet-managed pods running (their DaemonSet would start them again at once); default true. false refuses to drain a node that runs any, as kubectl drain does without --ignore-daemonsets"`
	DeleteEmptyDirData bool   `json:"deleteEmptyDirData,omitempty" jsonschema:"also evict pods with emptyDir volumes, whose emptyDir data is lost; default false refuses to drain a node that runs any (kubectl drain --delete-emptydir-data)"`
	Force              bool   `json:"force,omitempty" jsonschema:"also evict pods that no controller manages, which nothing starts again; default false refuses to drain a node that runs any (kubectl drain --force)"`
	GracePeriodSeconds uint16 `json:"gracePeriodSeconds,omitempty" jsonschema:"seconds each evicted pod gets to shut down; 0 (default) uses each pod's own terminationGracePeriodSeconds. Unlike kubectl, 0 never ends pods at once"`
	TimeoutSeconds     uint16 `json:"timeoutSeconds,omitempty" jsonschema:"give up after this many seconds, leaving the node cordoned and partly drained; 0 (default) means 300"`
}

// DefaultDrainTimeout bounds a drain whose settings name no timeout: a PodDisruptionBudget that
// never allows an eviction would otherwise hold the drain forever. Cogpit's choice (inferred).
const DefaultDrainTimeout = 300 * time.Second

// WithDefaults returns the settings with every default filled in, as a dialog shows them.
func (d DrainSettings) WithDefaults() DrainSettings {
	if d.IgnoreDaemonSets == nil {
		yes := true
		d.IgnoreDaemonSets = &yes
	}
	if d.TimeoutSeconds == 0 {
		d.TimeoutSeconds = uint16(DefaultDrainTimeout / time.Second)
	}
	return d
}

// ClusterOptions translates the settings into the drain library's options.
func (d DrainSettings) ClusterOptions() cluster.DrainOptions {
	d = d.WithDefaults()
	grace := -1 // the pod's own
	if d.GracePeriodSeconds > 0 {
		grace = int(d.GracePeriodSeconds)
	}
	return cluster.DrainOptions{
		IgnoreDaemonSets:   *d.IgnoreDaemonSets,
		DeleteEmptyDirData: d.DeleteEmptyDirData,
		Force:              d.Force,
		GracePeriodSeconds: grace,
		Timeout:            time.Duration(d.TimeoutSeconds) * time.Second,
	}
}

// DrainState is a journal entry's state.
type DrainState string

// Drain states.
const (
	// DrainRunning: the drain was started. An entry that a later start of Cogpit finds in this
	// state was interrupted: Cogpit ended (crashed, was killed, lost power) before the drain did.
	DrainRunning DrainState = "running"
	// DrainFailed: the drain ended with an error (a refused pod, a blocked eviction, a timeout);
	// the node stays cordoned and partly drained.
	DrainFailed DrainState = "failed"
)

// PendingDrain is a drain Cogpit started and has not seen finish: recorded before the node is
// cordoned, removed when every pod is evicted (D-11).
type PendingDrain struct {
	Context   string        `json:"context"`
	Node      string        `json:"node"`
	Started   time.Time     `json:"started"`
	StartedBy Origin        `json:"startedBy"`
	State     DrainState    `json:"state"`
	Error     string        `json:"error,omitempty"`
	Settings  DrainSettings `json:"settings"`
}

func (p PendingDrain) key() drainKey { return drainKey{p.Context, p.Node} }

type drainKey struct{ context, node string }

// DrainJournal keeps the pending drains across restarts of Cogpit. Implementations are safe for
// concurrent use.
type DrainJournal interface {
	// List returns every entry. A journal that cannot be read returns an error, never a partial
	// list.
	List() ([]PendingDrain, error)
	// Put adds an entry or replaces the one for the same context and node.
	Put(PendingDrain) error
	// Remove deletes the entry for a context and node; a missing entry is not an error.
	Remove(context, node string) error
}

// MemoryDrainJournal is a DrainJournal that lives as long as the process: for tests and for a
// session without a configuration directory.
type MemoryDrainJournal struct {
	mu      sync.Mutex
	entries []PendingDrain
}

// NewMemoryDrainJournal returns an empty in-memory journal.
func NewMemoryDrainJournal() *MemoryDrainJournal { return &MemoryDrainJournal{} }

// List implements DrainJournal.
func (m *MemoryDrainJournal) List() ([]PendingDrain, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.entries), nil
}

// Put implements DrainJournal.
func (m *MemoryDrainJournal) Put(p PendingDrain) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = putEntry(m.entries, p)
	return nil
}

// Remove implements DrainJournal.
func (m *MemoryDrainJournal) Remove(context, node string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = removeEntry(m.entries, drainKey{context, node})
	return nil
}

func putEntry(entries []PendingDrain, p PendingDrain) []PendingDrain {
	out := removeEntry(entries, p.key())
	return append(out, p)
}

func removeEntry(entries []PendingDrain, k drainKey) []PendingDrain {
	return slices.DeleteFunc(slices.Clone(entries), func(e PendingDrain) bool { return e.key() == k })
}

// DrainJournalFile is the journal's file name in Cogpit's configuration directory.
const DrainJournalFile = "drains.json"

// drainJournalVersion is the file format's version; a file with another one is not read.
const drainJournalVersion = 1

// drainJournalDoc is the file: {"version":1,"drains":[{"context":…,"node":…},…]}.
type drainJournalDoc struct {
	Version int            `json:"version"`
	Drains  []PendingDrain `json:"drains"`
}

// FileDrainJournal is the DrainJournal of the desktop app: one JSON file, owner-only (0600) in an
// owner-only directory (0700), replaced atomically on every change (write a temporary file in
// the same directory, sync it, rename it over the old one), so that a crash leaves the old or the
// new journal, never a torn one. A file that cannot be parsed is reported, not overwritten: it may
// be the only record of a half-drained node.
//
// The file is not locked: two Cogpit processes changing it at once may lose one of their entries.
type FileDrainJournal struct {
	path string
	mu   sync.Mutex
}

// NewFileDrainJournal returns the journal at <dir>/drains.json. Nothing is read or written until
// it is used.
func NewFileDrainJournal(dir string) *FileDrainJournal {
	return &FileDrainJournal{path: filepath.Join(dir, DrainJournalFile)}
}

// Path returns the journal file's path.
func (j *FileDrainJournal) Path() string { return j.path }

// JournalError reports a journal file that cannot be read.
type JournalError struct {
	Path string
	Err  error
}

func (e *JournalError) Error() string {
	return fmt.Sprintf("the drain journal %s is unreadable (%v); Cogpit leaves it as it is: move it aside to start a new one", e.Path, e.Err)
}

// Unwrap returns the cause.
func (e *JournalError) Unwrap() error { return e.Err }

// List implements DrainJournal.
func (j *FileDrainJournal) List() ([]PendingDrain, error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.load()
}

// Put implements DrainJournal.
func (j *FileDrainJournal) Put(p PendingDrain) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	entries, err := j.load()
	if err != nil {
		return err
	}
	return j.save(putEntry(entries, p))
}

// Remove implements DrainJournal.
func (j *FileDrainJournal) Remove(context, node string) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	entries, err := j.load()
	if err != nil {
		return err
	}
	k := drainKey{context, node}
	if !slices.ContainsFunc(entries, func(e PendingDrain) bool { return e.key() == k }) {
		return nil
	}
	return j.save(removeEntry(entries, k))
}

// maxJournalSize bounds what List reads: a journal holds a few entries, a large file is not one.
const maxJournalSize = 1 << 20

func (j *FileDrainJournal) load() ([]PendingDrain, error) {
	info, err := os.Stat(j.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, &JournalError{Path: j.path, Err: err}
	}
	if info.Size() > maxJournalSize {
		return nil, &JournalError{Path: j.path, Err: fmt.Errorf("%d bytes is not a drain journal", info.Size())}
	}
	data, err := os.ReadFile(j.path)
	if err != nil {
		return nil, &JournalError{Path: j.path, Err: err}
	}
	var doc drainJournalDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, &JournalError{Path: j.path, Err: err}
	}
	if doc.Version != drainJournalVersion {
		return nil, &JournalError{Path: j.path, Err: fmt.Errorf("format version %d, this Cogpit reads %d", doc.Version, drainJournalVersion)}
	}
	for _, e := range doc.Drains {
		if e.Context == "" || e.Node == "" {
			return nil, &JournalError{Path: j.path, Err: errors.New("an entry without a context or node")}
		}
	}
	return doc.Drains, nil
}

func (j *FileDrainJournal) save(entries []PendingDrain) error {
	if entries == nil {
		entries = []PendingDrain{}
	}
	data, err := json.MarshalIndent(drainJournalDoc{Version: drainJournalVersion, Drains: entries}, "", "  ")
	if err != nil {
		return fmt.Errorf("drain journal: %w", err)
	}
	dir := filepath.Dir(j.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("drain journal: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".drains-*.tmp") // created 0600
	if err != nil {
		return fmt.Errorf("drain journal: %w", err)
	}
	written := false
	defer func() {
		if !written {
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("drain journal: %w", err)
	}
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("drain journal: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("drain journal: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("drain journal: %w", err)
	}
	if err := os.Rename(tmp.Name(), j.path); err != nil {
		return fmt.Errorf("drain journal: %w", err)
	}
	written = true
	syncDir(dir)
	return nil
}

// syncDir makes a rename in dir durable where the platform supports it; failing is harmless (the
// rename itself is atomic).
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}
