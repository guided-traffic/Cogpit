package engine

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
)

// ErrNoHelmJournal is returned when the session keeps no Helm journal (Options.Helm is nil):
// Cogpit does not start a release action it could not offer to finish after a crash.
var ErrNoHelmJournal = errors.New("this Cogpit session keeps no Helm journal, so it cannot roll back or uninstall releases")

// HelmOps tracks a session's release actions (D-11), as Drains tracks drains: the journal that
// keeps them across restarts, and the actions running in this process. An action is recorded
// before the Helm SDK runs and removed when it returns without an error; one that fails keeps its
// entry, and so does one a crash interrupts, which leaves the release pending-rollback or
// uninstalling. When a context is connected, its entries are offered with EventHelm: the GUI shows
// a notice (finish: run the action again; or dismiss), agents read them with
// helm.pendingOperations. An entry whose action evidently finished (the release is gone after an
// uninstall, or deployed at a newer revision after a rollback) is dropped with a note.
//
// Create it with NewHelmOps and pass it as Options.Helm; it belongs to one session.
type HelmOps struct {
	journal HelmJournal

	mu       sync.Mutex
	s        *Session
	running  map[helmKey]bool
	lastRev  uint64
	lastConn ConnectionState
}

// NewHelmOps returns the tracker for a journal.
func NewHelmOps(j HelmJournal) *HelmOps {
	return &HelmOps{journal: j, running: map[helmKey]bool{}}
}

// HelmOps returns the session's release action tracker, or nil when it keeps none.
func (s *Session) HelmOps() *HelmOps { return s.opts.Helm }

// attach binds the tracker to its session and follows its connections. It is nil-safe.
func (h *HelmOps) attach(s *Session) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.s = s
	h.mu.Unlock()
	s.Subscribe(h.onEvent)
}

// onEvent runs on the goroutine that changed the UI state: it must not block.
func (h *HelmOps) onEvent(ev Event) {
	if ev.Type != EventUIState || ev.UIState == nil {
		return
	}
	u := ev.UIState
	h.mu.Lock()
	if u.Revision <= h.lastRev {
		h.mu.Unlock()
		return
	}
	h.lastRev = u.Revision
	prev := h.lastConn
	h.lastConn = u.Connection.State
	s := h.s
	h.mu.Unlock()
	if s == nil || u.Connection.State == prev {
		return
	}
	if u.Connection.State == Connected && !u.Connection.Demo {
		ctxName := u.Context
		s.goSafe("unfinished Helm actions", func() { h.offer(s, ctxName) })
		return
	}
	h.changed("") // the previous cluster's actions no longer apply
}

// offer reads the journal for a freshly connected context, drops the entries whose action
// evidently finished, and announces the rest.
func (h *HelmOps) offer(s *Session, ctxName string) {
	entries, err := h.journal.List()
	if err != nil {
		s.log.Error("engine: the Helm journal is unreadable", "err", err)
		s.emit(Event{Type: EventToast, Level: LevelError, Message: "Unfinished Helm actions cannot be offered: " + err.Error()})
		h.changed("")
		return
	}
	var pending []PendingHelmOp
	for _, e := range entries {
		if e.Context == ctxName && !h.Running(e.Context, e.Namespace, e.Release) {
			pending = append(pending, e)
		}
	}
	if len(pending) > 0 {
		pending = h.dropFinished(s, ctxName, pending)
	}
	msg := ""
	switch n := len(pending); {
	case n == 1:
		msg = fmt.Sprintf("The %s of release %s in context %s did not finish", pending[0].Action, pending[0].Release, ctxName)
	case n > 1:
		msg = fmt.Sprintf("%d Helm actions in context %s did not finish", n, ctxName)
	}
	if b := s.Backend(); b != nil && b.Context == ctxName {
		h.changed(msg)
	}
}

// dropFinished removes, with a toast each, the entries whose action evidently finished once the
// cache has the releases: an uninstall whose release is gone, a rollback whose release is deployed
// at a newer revision than the one it started from (or is gone). Without a synced release list (no
// permission, a slow cluster, another cluster connected meanwhile) nothing is dropped.
func (h *HelmOps) dropFinished(s *Session, ctxName string, pending []PendingHelmOp) []PendingHelmOp {
	b := s.Backend()
	if b == nil || b.Context != ctxName {
		return pending
	}
	p, ok := b.Profile(helm.ReleaseKind)
	if !ok {
		return pending
	}
	hd := b.Acquire(p)
	defer hd.Release()
	deadline := time.Now().Add(offerWait)
	for !hd.HasSynced() {
		if s.Backend() != b || time.Now().After(deadline) || s.isClosed() {
			s.log.Warn("engine: unfinished Helm actions offered without checking their releases", "context", ctxName, "err", hd.Err())
			return pending
		}
		time.Sleep(20 * time.Millisecond)
	}
	kept := pending[:0]
	for _, e := range pending {
		obj, found := KindGet(hd, p, e.Namespace, e.Release)
		if !helmOpFinished(e, obj, found) {
			kept = append(kept, e)
			continue
		}
		if err := h.journal.Remove(e.Context, e.Namespace, e.Release); err != nil {
			s.log.Error("engine: dropping a finished Helm action", "release", e.Release, "err", err)
			kept = append(kept, e)
			continue
		}
		s.log.Info("engine: dropped a Helm action that finished", "context", e.Context, "namespace", e.Namespace, "release", e.Release, "action", e.Action)
		s.emit(Event{Type: EventToast, Level: LevelInfo, Message: fmt.Sprintf(
			"Cogpit dropped the unfinished %s of release %s (context %s): the cluster shows it finished.", e.Action, e.Release, e.Context)})
	}
	return kept
}

// helmOpFinished reports whether the cache shows a recorded action done.
func helmOpFinished(e PendingHelmOp, obj *unstructured.Unstructured, found bool) bool {
	if !found {
		return true // uninstalled, or nothing left to roll back
	}
	return e.Action == HelmRollback && helm.ReleaseRevision(obj) > e.FromRevision &&
		helm.ReleaseStatus(obj) == "deployed"
}

// Begin records an action before the Helm SDK runs. It fails, and the action must not start, when
// an action on the release runs already or the journal cannot be written.
func (h *HelmOps) Begin(p PendingHelmOp) error {
	if h == nil {
		return ErrNoHelmJournal
	}
	k := p.key()
	h.mu.Lock()
	if h.running[k] {
		h.mu.Unlock()
		return fmt.Errorf("a Helm action on release %s is running already", p.Release)
	}
	h.running[k] = true
	h.mu.Unlock()

	p.State, p.Error = HelmOpRunning, ""
	if err := h.journal.Put(p); err != nil {
		h.mu.Lock()
		delete(h.running, k)
		h.mu.Unlock()
		return fmt.Errorf("the %s was not started, because Cogpit could not record it first: %w", p.Action, err)
	}
	h.changed("")
	return nil
}

// End records an action's outcome: a finished action's entry is removed; a failed one is kept as
// failed with its error. It returns the journal's error, if any.
func (h *HelmOps) End(p PendingHelmOp, opErr error) error {
	if h == nil {
		return ErrNoHelmJournal
	}
	h.mu.Lock()
	delete(h.running, p.key())
	h.mu.Unlock()
	var err error
	if opErr == nil {
		err = h.journal.Remove(p.Context, p.Namespace, p.Release)
	} else {
		p.State, p.Error = HelmOpFailed, opErr.Error()
		err = h.journal.Put(p)
	}
	h.changed("")
	return err
}

// Forget removes a pending action without touching the cluster: Dismiss. An action running in
// this process is not forgotten.
func (h *HelmOps) Forget(context, namespace, release string) error {
	if h == nil {
		return ErrNoHelmJournal
	}
	if h.Running(context, namespace, release) {
		return fmt.Errorf("the Helm action on release %s is running", release)
	}
	if err := h.journal.Remove(context, namespace, release); err != nil {
		return err
	}
	h.changed("")
	return nil
}

// Running reports whether an action on the release runs in this process.
func (h *HelmOps) Running(context, namespace, release string) bool {
	if h == nil {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running[helmKey{context, namespace, release}]
}

// Pending returns the journal entry for a release, if there is one.
func (h *HelmOps) Pending(context, namespace, release string) (PendingHelmOp, bool, error) {
	if h == nil {
		return PendingHelmOp{}, false, ErrNoHelmJournal
	}
	entries, err := h.journal.List()
	if err != nil {
		return PendingHelmOp{}, false, err
	}
	for _, e := range entries {
		if e.key() == (helmKey{context, namespace, release}) {
			return e, true, nil
		}
	}
	return PendingHelmOp{}, false, nil
}

// changed emits EventHelm.
func (h *HelmOps) changed(msg string) {
	h.mu.Lock()
	s := h.s
	h.mu.Unlock()
	if s != nil {
		s.emit(Event{Type: EventHelm, Message: msg})
	}
}

// Statuses of a pending release action, as PendingHelmOps reports them.
const (
	HelmOpStatusRunning     = "running"     // runs in this Cogpit
	HelmOpStatusInterrupted = "interrupted" // an earlier run of Cogpit ended before the action did
	HelmOpStatusFailed      = "failed"      // the Helm SDK returned an error
)

// Release states of a pending action's release, beyond its Helm status.
const (
	HelmReleaseMissing = "missing" // not in the cache's releases
	HelmReleaseUnknown = "unknown" // another context, or the releases are not in the cache
)

// PendingHelmOpInfo is a pending release action as helm.pendingOperations reports it.
type PendingHelmOpInfo struct {
	PendingHelmOp
	Status string `json:"status"`
	// ReleaseStatus is the release's newest revision's status in the cache ("pending-rollback",
	// "uninstalling", "deployed", …), or HelmReleaseMissing or HelmReleaseUnknown.
	ReleaseStatus   string `json:"releaseStatus"`
	ReleaseRevision int    `json:"releaseRevision,omitempty"`
}

// Ref returns the release the entry is about.
func (p PendingHelmOp) Ref() model.ObjectRef {
	return model.ObjectRef{Context: p.Context, Kind: helm.ReleaseKind, Namespace: p.Namespace, Name: p.Release}
}

// PendingHelmOps returns the pending release actions of the connected context (all: of every
// context), oldest first, with their status and their release's state from the cache. It reads
// the journal and the cache, never the API server.
func (s *Session) PendingHelmOps(all bool) ([]PendingHelmOpInfo, error) {
	h := s.opts.Helm
	if h == nil {
		return nil, ErrNoHelmJournal
	}
	b := s.Backend()
	if b == nil && !all {
		return nil, ErrNotConnected
	}
	entries, err := h.journal.List()
	if err != nil {
		return nil, err
	}
	out := make([]PendingHelmOpInfo, 0, len(entries))
	for _, e := range entries {
		if !all && e.Context != b.Context {
			continue
		}
		info := PendingHelmOpInfo{PendingHelmOp: e, ReleaseStatus: HelmReleaseUnknown}
		switch {
		case h.Running(e.Context, e.Namespace, e.Release):
			info.Status = HelmOpStatusRunning
		case e.State == HelmOpFailed:
			info.Status = HelmOpStatusFailed
		default:
			info.Status = HelmOpStatusInterrupted
		}
		if b != nil && e.Context == b.Context {
			info.ReleaseStatus, info.ReleaseRevision = s.releaseState(e.Ref())
		}
		out = append(out, info)
	}
	slices.SortFunc(out, func(a, b PendingHelmOpInfo) int {
		if c := a.Started.Compare(b.Started); c != 0 {
			return c
		}
		return cmp.Compare(a.Namespace+"/"+a.Release, b.Namespace+"/"+b.Release)
	})
	return out, nil
}

// releaseState reads a release's status and newest revision from the cache, without waiting.
func (s *Session) releaseState(ref model.ObjectRef) (string, int) {
	obj, err := s.Lookup(ref)
	switch {
	case errors.Is(err, ErrNotFound):
		return HelmReleaseMissing, 0
	case err != nil:
		return HelmReleaseUnknown, 0
	}
	return helm.ReleaseStatus(obj), helm.ReleaseRevision(obj)
}

