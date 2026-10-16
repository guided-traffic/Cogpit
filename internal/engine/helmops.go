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
// a notice (finish: run the action again; or dismiss; an install is offered nothing, only its
// release's state, A17), agents read them with helm.pendingOperations. An entry whose action
// evidently finished (the release is gone after an uninstall, or deployed at a newer revision
// after a rollback) is dropped with a note.
//
// Create it with NewHelmOps and pass it as Options.Helm; it belongs to one session.
type HelmOps struct {
	journal HelmJournal

	// jmu serialises the journal's changes with the running state they depend on: Begin's check
	// and record, End's record and release, Forget's and dropFinished's check and removal. Without
	// it a Begin between another operation's check and its write loses its entry (D-11). Taken
	// before mu; never held while emitting events.
	jmu sync.Mutex

	mu       sync.Mutex
	s        *Session
	running  map[helmKey]bool
	lastRev  uint64
	lastConn ConnectionState
	lastCtx  string // the shown cluster the notice was last offered for (D-69)
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
	prev, prevCtx := h.lastConn, h.lastCtx
	h.lastConn, h.lastCtx = u.Connection.State, u.Context
	s := h.s
	h.mu.Unlock()
	// A connect, a reconnect, or another connected cluster shown (D-69): the notice is the shown
	// cluster's.
	if s == nil || (u.Connection.State == prev && u.Context == prevCtx) {
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
		if e.Context == ctxName && !h.Running(e.Context, e.Namespace, e.Release) && !e.RunsElsewhere() { // H-22
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
	b := s.BackendOf(ctxName) // its cluster, shown or not (D-69)
	if b == nil {
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
		if !s.state.holds(b) || time.Now().After(deadline) || s.isClosed() {
			s.log.Warn("engine: unfinished Helm actions offered without checking their releases", "context", ctxName, "err", hd.Err())
			return pending
		}
		time.Sleep(20 * time.Millisecond)
	}
	kept := pending[:0]
	for _, e := range pending {
		obj, found := KindGet(hd, p, e.Namespace, e.Release)
		done := helmOpFinished(e, obj, found)
		if done == "" {
			kept = append(kept, e)
			continue
		}
		if err := h.forget(e.Context, e.Namespace, e.Release); err != nil {
			// Unreadable journal, or the user started the action again while the cache synced.
			s.log.Error("engine: dropping a finished Helm action", "release", e.Release, "err", err)
			kept = append(kept, e)
			continue
		}
		s.log.Info("engine: dropped a Helm action that finished", "context", e.Context, "namespace", e.Namespace, "release", e.Release, "action", e.Action)
		s.emit(Event{Type: EventToast, Level: LevelInfo, Message: fmt.Sprintf(
			"Cogpit dropped the unfinished %s of release %s (context %s): %s.", e.Action, e.Release, e.Context, done)})
	}
	return kept
}

// helmOpFinished says why the cache shows a recorded action needs nothing more, or "" when it is
// still to be offered: a release that is gone (uninstalled, or nothing left to roll back), a
// rollback or an upgrade whose release is deployed at a revision above the one it started from,
// and an upgrade whose release is still at that revision, which never started: Cogpit ended (or
// the Helm SDK refused) before the SDK made its pending revision, and nothing changed. An upgrade
// left pending-upgrade or failed above it is offered: finishing it rolls back to FromRevision. An
// install (FromRevision 0) whose release is deployed finished; one whose release does not exist
// never stored it; one left pending-install or failed stays, shown with its release's state and
// offered nothing (A17).
func helmOpFinished(e PendingHelmOp, obj *unstructured.Unstructured, found bool) string {
	if !found && e.Action == HelmInstall {
		return "the release does not exist: the install never stored it, or it was uninstalled since"
	}
	if !found {
		return "the release no longer exists"
	}
	rev, deployed := helm.ReleaseRevision(obj), helm.ReleaseStatus(obj) == "deployed"
	switch {
	case e.Action == HelmUpgrade && rev == e.FromRevision:
		return "it never reached the cluster, the release is unchanged"
	case deployed && rev > e.FromRevision && (e.Action == HelmRollback || e.Action == HelmUpgrade || e.Action == HelmInstall):
		return "the cluster shows it finished"
	}
	return ""
}

// Begin records an action before the Helm SDK runs. It fails, and the action must not start, when
// an action on the release runs already or the journal cannot be written.
func (h *HelmOps) Begin(p PendingHelmOp) error {
	if h == nil {
		return ErrNoHelmJournal
	}
	if err := h.begin(p); err != nil {
		return err
	}
	h.changed("")
	return nil
}

func (h *HelmOps) begin(p PendingHelmOp) error {
	h.jmu.Lock()
	defer h.jmu.Unlock()
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
	return nil
}

// End records an action's outcome: a finished action's entry is removed; a failed one is kept as
// failed with its error. It returns the journal's error, if any.
func (h *HelmOps) End(p PendingHelmOp, opErr error) error {
	if h == nil {
		return ErrNoHelmJournal
	}
	err := h.end(p, opErr)
	h.changed("")
	return err
}

// end records the outcome while the action still counts as running, so that no Begin on the
// release can record a new action that this write would then remove or overwrite.
func (h *HelmOps) end(p PendingHelmOp, opErr error) error {
	h.jmu.Lock()
	defer h.jmu.Unlock()
	var err error
	if opErr == nil {
		err = h.journal.Remove(p.Context, p.Namespace, p.Release)
	} else {
		p.State, p.Error = HelmOpFailed, opErr.Error()
		err = h.journal.Put(p)
	}
	h.mu.Lock()
	delete(h.running, p.key())
	h.mu.Unlock()
	return err
}

// Forget removes a pending action without touching the cluster: Dismiss. An action running in
// this process is not forgotten.
func (h *HelmOps) Forget(context, namespace, release string) error {
	if h == nil {
		return ErrNoHelmJournal
	}
	if err := h.forget(context, namespace, release); err != nil {
		return err
	}
	h.changed("")
	return nil
}

// forget removes an entry unless its action runs, checked and removed as one step.
func (h *HelmOps) forget(context, namespace, release string) error {
	h.jmu.Lock()
	defer h.jmu.Unlock()
	if h.Running(context, namespace, release) {
		return fmt.Errorf("the Helm action on release %s is running", release)
	}
	return h.journal.Remove(context, namespace, release)
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
	// Elsewhere: status running, in another live Cogpit process (Owner names its pid; H-22).
	Elsewhere bool `json:"elsewhere,omitempty"`
}

// Ref returns the release the entry is about.
func (p PendingHelmOp) Ref() model.ObjectRef {
	return model.ObjectRef{Context: p.Context, Kind: helm.ReleaseKind, Namespace: p.Namespace, Name: p.Release}
}

// PendingHelmOps returns the pending release actions of the shown cluster's context (all: of every
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
		case e.RunsElsewhere():
			info.Status, info.Elsewhere = HelmOpStatusRunning, true
		case e.State == HelmOpFailed:
			info.Status = HelmOpStatusFailed
		default:
			info.Status = HelmOpStatusInterrupted
		}
		if s.BackendOf(e.Context) != nil { // every connected cluster's (D-69)
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
