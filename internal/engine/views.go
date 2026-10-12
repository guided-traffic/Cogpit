package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// ErrClosed is returned by a Session that was closed.
var ErrClosed = errors.New("the session is closed")

// state holds the connected backend, the open view with its cache handle and coalesced snapshots,
// the drawer's watch, and the diagnostics loop.
//
// Lock order: state.mu before Session.mu. Session.mu is never held while state.mu is taken
// (UpdateUI releases it before calling uiChanged); state.mu may be held while Session.mu is taken
// briefly to read the UI state (customKindsChanged reconciles against the latest one). state.mu is
// never held while emitting events or building snapshots.
type state struct {
	s *Session

	// connMu serialises Connect calls, so that the UI state follows one connection at a time.
	connMu sync.Mutex

	mu            sync.Mutex
	closed        bool
	connectSeq    uint64
	connectCancel context.CancelFunc
	backend       *Backend
	appliedRev    uint64
	view          *viewWorker
	page          *overviewWorker // the open page's worker (overview.go); nil unless a page is open
	vanished      *TableSnapshot  // the open view after the cluster stopped serving its kind (customkinds.go)
	drawer        *drawerWatch
	diagStop      chan struct{}
	sorts         map[string]SortSpec  // the sort order remembered per view
	rowCaches     map[string]*rowCache // rows per view of the current backend, reused across visits
	dockSeq       map[DockType]int

	logs connectLogs // the last connect attempt of each context (connectlog.go); has its own lock

	snapshots atomic.Uint64 // table snapshots built since start; also the snapshots' revisions
	wg        sync.WaitGroup
}

func newState(s *Session) *state {
	return &state{
		s:         s,
		sorts:     map[string]SortSpec{},
		rowCaches: map[string]*rowCache{},
		dockSeq:   map[DockType]int{},
	}
}

// uiChanged follows a UI state change: start the view worker of a newly opened view and stop the
// old one, hand namespaces, filter and sort to the worker, and follow the drawer. It reconciles
// against the new state rather than the difference, so calls that arrive out of order (two
// goroutines updating the UI at once) are harmless: an older revision is ignored.
func (st *state) uiChanged(_, now UIState) {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.closed || now.Revision < st.appliedRev {
		return
	}
	st.appliedRev = now.Revision
	st.reconcileLocked(now)
}

func (st *state) reconcileLocked(now UIState) {
	b := st.backend

	// The view: one worker for the open kind of the current backend.
	var prof *model.KindProfile
	if b != nil {
		prof, _ = b.Profile(now.View)
	}
	st.noteVanishedLocked(b, prof, now.View)
	if st.view != nil && (st.view.backend != b || st.view.profile != prof) {
		st.view.stop()
		st.view = nil
	}
	if st.view == nil && prof != nil {
		rc := st.rowCaches[prof.Key]
		if rc == nil {
			rc = &rowCache{}
			st.rowCaches[prof.Key] = rc
		}
		st.view = st.s.startViewWorker(b, prof, rc, paramsOf(now))
	} else if st.view != nil {
		st.view.setParams(paramsOf(now))
	}
	if prof != nil && prof.ColumnIndex(now.Sort.Column) >= 0 {
		st.sorts[prof.Key] = now.Sort
	}
	st.reconcilePageLocked(b, now)

	// The drawer: a watch on its object and on the events. A custom kind's profile changes with its
	// definition (another version, or gone), and the drawer follows it.
	var drawerProf *model.KindProfile
	if b != nil && now.Drawer != nil {
		drawerProf, _ = b.Profile(now.Drawer.Kind)
	}
	if st.drawer != nil && (now.Drawer == nil || st.drawer.backend != b || st.drawer.ref != *now.Drawer || st.drawer.profile != drawerProf) {
		st.drawer.stop()
		st.drawer = nil
	}
	if st.drawer == nil && drawerProf != nil {
		st.drawer = st.s.startDrawerWatch(b, drawerProf, *now.Drawer)
	}
}

// detachLocked stops everything bound to the current backend and returns it for closing outside
// the lock.
func (st *state) detachLocked() *Backend {
	if st.view != nil {
		st.view.stop()
		st.view = nil
	}
	if st.page != nil {
		st.page.stop()
		st.page = nil
	}
	if st.drawer != nil {
		st.drawer.stop()
		st.drawer = nil
	}
	if st.diagStop != nil {
		close(st.diagStop)
		st.diagStop = nil
	}
	old := st.backend
	st.backend = nil
	clear(st.rowCaches)
	return old
}

// close disconnects and stops every loop. It waits a bounded time for the engine's goroutines.
func (st *state) close() {
	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return
	}
	st.closed = true
	if st.connectCancel != nil {
		st.connectCancel()
	}
	old := st.detachLocked()
	st.mu.Unlock()
	old.close(st.s.log)

	done := make(chan struct{})
	go func() {
		st.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		st.s.log.Warn("engine: goroutines still running after close")
	}
}

// Backend returns the connected backend, or nil. It is shared: callers must not modify it.
func (s *Session) Backend() *Backend {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.backend
}

// Contexts lists the contexts the connector offers. With a partial result (the kubeconfig failed
// to load but the demo cluster is offered), both the list and the error are returned.
func (s *Session) Contexts() ([]ContextInfo, error) {
	if s.opts.Connector == nil {
		return nil, errors.New("no connector configured")
	}
	var (
		out []ContextInfo
		err error
	)
	if pErr := s.safeCall("cluster.contexts", func() error { out, err = s.opts.Connector.Contexts(); return nil }); pErr != nil {
		return nil, pErr
	}
	s.markConnectProblems(out)
	return out, err
}

// Connect connects a context through the connector and opens the default view: Pods if the
// cluster serves them, else its first kind. The UI state follows the connection's lifecycle:
// Connecting, then Connected or ConnectError with the message.
//
// The previous backend is released when the connection starts, not when it succeeds, so that
// UIState.Context always names the cluster the engine acts on: a command can never act on the old
// cluster while the UI already names the new one. A later Connect supersedes a pending one.
func (s *Session) Connect(ctx context.Context, name string) error {
	st := s.state
	connector := s.opts.Connector
	if connector == nil {
		return errors.New("no connector configured")
	}
	if r, ok := connector.(contextResolver); ok {
		// A bare context name (--context kind-dev) names the one source context that has it (D-28).
		resolved, err := r.ResolveContext(name)
		if err != nil {
			return fmt.Errorf("connect %s: %w", name, err)
		}
		name = resolved
	}
	cctx, cancel := context.WithCancel(ctx)
	defer cancel()

	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return ErrClosed
	}
	if st.connectCancel != nil {
		st.connectCancel() // supersede a pending connect: it returns soon and leaves the UI alone
	}
	st.connectSeq++
	seq := st.connectSeq
	st.connectCancel = cancel
	st.mu.Unlock()

	st.connMu.Lock()
	defer st.connMu.Unlock()
	if !st.current(seq) {
		return fmt.Errorf("connect %s: superseded by a newer connect", name)
	}

	st.mu.Lock()
	old := st.detachLocked()
	st.mu.Unlock()
	old.close(s.log)
	s.UpdateUI(func(u *UIState) {
		u.Context = name
		u.Connection = Connection{State: Connecting}
		u.View = ""
		u.Selected = nil
		u.Drawer = nil
		u.Filter = ""
	})

	st.logs.restart(name, "Connecting to "+name)
	var b *Backend
	err := s.safeCall("connect", func() (err error) { b, err = connector.Connect(cctx, name); return err })
	if err == nil && b == nil {
		err = errors.New("the connector returned no cluster")
	}
	if err != nil {
		if st.current(seq) {
			st.logs.add(name, LogError, err.Error())
			msg := err.Error()
			s.UpdateUI(func(u *UIState) { u.Connection = Connection{State: ConnectError, Error: msg} })
		}
		return fmt.Errorf("connect %s: %w", name, err)
	}

	st.mu.Lock()
	if st.closed || st.connectSeq != seq {
		st.mu.Unlock()
		b.close(s.log)
		return fmt.Errorf("connect %s: superseded by a newer connect", name)
	}
	s.attachCustomKinds(b)
	s.attachAccess(b)
	st.logs.add(name, LogInfo, "Connected to "+name)
	st.backend = b
	st.diagStop = make(chan struct{})
	diagStop := st.diagStop
	st.mu.Unlock()
	s.goSafe("diagnostics", func() { s.diagnosticsLoop(diagStop) })

	view := defaultView(b)
	sort := s.ViewSort(view)
	s.UpdateUI(func(u *UIState) {
		u.Context = b.Context
		u.Connection = Connection{State: Connected, Demo: b.Demo}
		u.View = view
		u.Sort = sort
		u.Selected = nil
		u.Drawer = nil
		u.Filter = ""
	})
	return nil
}

// current reports whether the connect with sequence number seq is still the latest.
func (st *state) current(seq uint64) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	return !st.closed && st.connectSeq == seq
}

// defaultView is the view a fresh connection opens: Pods if served, else the first kind.
func defaultView(b *Backend) string {
	if _, ok := b.Profile("pods"); ok {
		return "pods"
	}
	if len(b.Profiles) > 0 {
		return b.Profiles[0].Key
	}
	return ""
}

// ViewSort returns the sort order for a view: the one last used for it, else the default (name
// ascending; events by last seen, newest first).
func (s *Session) ViewSort(view string) SortSpec {
	st := s.state
	st.mu.Lock()
	defer st.mu.Unlock()
	p := st.backend.viewProfile(view) // a kind's profile, or a page's table profile
	if sp, ok := st.sorts[view]; ok && (p == nil || p.ColumnIndex(sp.Column) >= 0) {
		return sp
	}
	return defaultSort(p)
}

// DefaultSort is the sort order of a view nobody sorted yet: by name, ascending.
var DefaultSort = SortSpec{Column: "name"}

// lastSeenColumn is the events column the Events view sorts by by default, newest first.
const lastSeenColumn = "lastSeen"

func defaultSort(p *model.KindProfile) SortSpec {
	if p != nil && p.Key == eventsKind && p.ColumnIndex(lastSeenColumn) >= 0 {
		return SortSpec{Column: lastSeenColumn, Desc: true}
	}
	return DefaultSort
}

// Table returns the latest snapshot of a view, or nil if the view is not open or has no snapshot
// yet. An open view whose kind the cluster stopped serving returns a snapshot without rows whose
// Error says so. Snapshots are immutable and shared: callers must not modify them.
func (s *Session) Table(view string) *TableSnapshot {
	s.state.mu.Lock()
	w := s.state.view
	page := s.state.page
	gone := s.state.vanished
	s.state.mu.Unlock()
	if page != nil && view == WorkloadsOverview {
		return page.snap.Load()
	}
	if w == nil || w.profile.Key != view {
		if gone != nil && gone.View == view {
			return gone
		}
		return nil
	}
	return w.snap.Load()
}

// NewDockID returns a fresh dock tab id for a tab type: "logs-1", "terminal-2".
func (s *Session) NewDockID(t DockType) string {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	s.state.dockSeq[t]++
	return string(t) + "-" + strconv.Itoa(s.state.dockSeq[t])
}

// viewParams are the UI state a view's snapshot depends on.
type viewParams struct {
	namespaces []string
	groups     []string // the Definitions view's API group filter
	hidden     []string // the view's hidden column ids; only when hiddenSet
	hiddenSet  bool     // false: the kind's default columns
	filter     string
	sort       SortSpec
}

func paramsOf(u UIState) viewParams {
	hidden, set := u.HiddenColumns[u.View]
	return viewParams{
		namespaces: slices.Clone(u.Namespaces), groups: slices.Clone(u.Groups),
		hidden: slices.Clone(hidden), hiddenSet: set,
		filter: u.Filter, sort: u.Sort,
	}
}

func (p viewParams) equal(o viewParams) bool {
	return p.filter == o.filter && p.sort == o.sort && slices.Equal(p.namespaces, o.namespaces) &&
		slices.Equal(p.groups, o.groups) && p.hiddenSet == o.hiddenSet && slices.Equal(p.hidden, o.hidden)
}
