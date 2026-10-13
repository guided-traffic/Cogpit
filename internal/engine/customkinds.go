package engine

import (
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// crdsGVR is the resource the engine watches to learn a cluster's custom resource kinds.
var crdsGVR = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}

// customKindsCoalesce stretches the snapshot interval for rebuilding the custom kinds: installing
// a chart creates dozens of definitions within a second, and each rebuild redraws the sidebar.
const customKindsCoalesce = 5

// customKinds keeps a backend's custom resource kinds current: it holds a cache handle on the
// cluster's custom resource definitions (one watch, shared through the cache, released when the
// backend closes), rebuilds the kinds' profiles after changes, coalesced, and publishes them as
// one immutable customSet that views, commands, agents and the UI read concurrently.
type customKinds struct {
	s       *Session
	b       *Backend
	handle  *cache.Handle
	unsub   func()
	wake    chan struct{}
	quit    chan struct{}
	stopped atomic.Bool
	set     atomic.Pointer[customSet]

	// entries are the kinds of the last build by kind key. Only the rebuild loop touches them.
	entries map[string]customEntry
}

type customEntry struct {
	kind    kinds.CustomKind
	profile *model.KindProfile
}

// customSet is one state of a backend's custom resource kinds. It is published whole and never
// changed; a definition update that changes nothing a view shows keeps its profile pointer, so the
// open view keeps running.
type customSet struct {
	profiles []*model.KindProfile // sorted by API group, then Kind
	byKey    map[string]*model.KindProfile
	synced   bool   // the definitions' initial list is done
	err      string // the watch's last error: without list and watch on definitions there are no custom kinds
}

// attachCustomKinds starts the custom kinds watch of a backend that asks for it. It runs before
// the backend is published (st.mu held), so every reader of the backend sees b.custom.
func (s *Session) attachCustomKinds(b *Backend) {
	if !b.CustomResources || b.Cache == nil || b.custom != nil {
		return
	}
	ck := &customKinds{
		s: s, b: b,
		wake:    make(chan struct{}, 1),
		quit:    make(chan struct{}),
		entries: map[string]customEntry{},
	}
	ck.set.Store(&customSet{byKey: map[string]*model.KindProfile{}})
	ck.handle = b.Cache.Acquire(cache.Scope{GVR: crdsGVR})
	ck.unsub = ck.handle.Subscribe(func(cache.Event) { signal(ck.wake) })
	signal(ck.wake)
	b.custom = ck
	interval := s.opts.SnapshotInterval * customKindsCoalesce
	s.goSafe("custom resource kinds", func() {
		runCoalesced(ck.quit, ck.wake, interval, ck.unsettled, ck.rebuild)
	})
}

// stop ends the loop and releases the handle on the definitions.
func (ck *customKinds) stop() {
	if !ck.stopped.CompareAndSwap(false, true) {
		return
	}
	close(ck.quit)
	ck.unsub()
	ck.handle.Release()
}

// unsettled reports whether the watch's sync state or error changed since the last build: neither
// comes with a watch event.
func (ck *customKinds) unsettled() bool {
	last := ck.set.Load()
	return last.synced != ck.handle.HasSynced() || last.err != errText(ck.handle.Err())
}

// rebuild turns the cached definitions into profiles and publishes them. Established definitions
// only; a kind a built-in (curated) profile covers keeps the built-in one. A panic is logged and
// leaves the last set in place.
func (ck *customKinds) rebuild() {
	defer func() {
		if r := recover(); r != nil {
			ck.s.log.Error("engine: building the custom resource kinds failed", "context", ck.b.Context,
				"panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	synced := ck.handle.HasSynced() // before List: a set never claims a partial list is complete
	errMsg := errText(ck.handle.Err())
	crds := ck.handle.List()
	slices.SortFunc(crds, func(x, y *unstructured.Unstructured) int { return strings.Compare(x.GetName(), y.GetName()) })

	next := make(map[string]customEntry, len(crds))
	for _, crd := range crds {
		k, err := kinds.ParseCRD(crd)
		if err != nil {
			continue // not established (yet), or not a kind Cogpit can name
		}
		key := k.Key()
		if _, dup := next[key]; dup || ck.b.builtinProfile(key) != nil {
			continue
		}
		if old, ok := ck.entries[key]; ok && old.kind.Equal(k) {
			next[key] = old
			continue
		}
		next[key] = customEntry{kind: k, profile: kinds.ProfileFor(k)} // a curated profile, else the generic one
	}
	addArgoCompanions(ck.b, next) // argocd.go: Argo CD's settings views come with its Application kind
	changed := len(next) != len(ck.entries)
	for key, e := range next {
		if old, ok := ck.entries[key]; !ok || old.profile != e.profile {
			changed = true
		}
	}
	ck.entries = next

	set := &customSet{
		profiles: make([]*model.KindProfile, 0, len(next)),
		byKey:    make(map[string]*model.KindProfile, len(next)),
		synced:   synced,
		err:      errMsg,
	}
	for key, e := range next {
		set.profiles = append(set.profiles, e.profile)
		set.byKey[key] = e.profile
	}
	slices.SortFunc(set.profiles, func(x, y *model.KindProfile) int {
		if c := strings.Compare(x.Group, y.Group); c != 0 {
			return c
		}
		if c := strings.Compare(x.Kind, y.Kind); c != 0 {
			return c
		}
		return strings.Compare(x.Key, y.Key)
	})
	if ck.stopped.Load() {
		return
	}
	ck.set.Store(set)
	if changed {
		ck.s.customKindsChanged(ck.b)
	}
}

// customKindsChanged follows a change of the connected backend's custom kinds: the open view and
// the drawer follow their profiles (a kind that is gone leaves a view that says so, a kind whose
// version changed moves its view to the new version), discovery is refreshed, and EventKinds tells
// the UI to read kinds.list again.
func (s *Session) customKindsChanged(b *Backend) {
	if b.Conn != nil {
		b.Conn.InvalidateServedResources()
	}
	st := s.state
	st.mu.Lock()
	if st.closed || st.backend != b {
		st.mu.Unlock()
		return
	}
	// The UI state is read under state.mu: read before it, it could be older than one an UpdateUI
	// applied in between, and reconciling against it would close the drawer or view that update
	// opened. Read here, it is the latest; an UpdateUI still on its way to uiChanged reconciles the
	// same state again.
	ui := s.UIState()
	if ui.Revision > st.appliedRev {
		st.appliedRev = ui.Revision
	}
	st.reconcileLocked(ui)
	gone := st.vanished
	st.mu.Unlock()

	s.emit(Event{Type: EventKinds})
	if gone != nil {
		s.emit(Event{Type: EventTable, View: gone.View, Revision: gone.Revision})
	}
}

// noteVanishedLocked keeps what the open view showed when the cluster stops serving its kind (its
// custom resource definition was deleted or is no longer established): Table then returns a
// snapshot that says so, instead of nothing. reconcileLocked calls it before it stops the view.
func (st *state) noteVanishedLocked(b *Backend, prof *model.KindProfile, view string) {
	switch {
	case prof != nil || b == nil || view == "":
		st.vanished = nil
	case st.view != nil && st.view.backend == b && st.view.profile.Key == view:
		p := st.view.profile
		st.view.mu.Lock()
		params := st.view.params
		st.view.mu.Unlock()
		st.vanished = &TableSnapshot{
			View: p.Key, Context: b.Context, Title: p.Title, Namespaced: p.Namespaced,
			Columns: p.Columns, Rows: []model.Row{}, Sort: params.sort,
			Namespaces: slices.Clone(params.namespaces), Filter: params.filter,
			Synced: true, Revision: st.snapshots.Add(1),
			Error: "The cluster no longer serves " + p.Title + " (" + p.Key + "): its custom resource definition was deleted or is not established.",
		}
	case st.vanished != nil && (st.vanished.View != view || st.vanished.Context != b.Context):
		st.vanished = nil
	}
}

// builtinProfile returns the backend's built-in profile for a kind key, or nil.
func (b *Backend) builtinProfile(key string) *model.KindProfile {
	for _, p := range b.Profiles {
		if p.Key == key {
			return p
		}
	}
	return nil
}

// customSet returns the backend's current custom kinds, or nil when it does not watch them.
func (b *Backend) customSet() *customSet {
	if b == nil || b.custom == nil {
		return nil
	}
	return b.custom.set.Load()
}

// CustomProfiles returns the kinds of the cluster's established custom resource definitions,
// sorted by API group and Kind, without the kinds a built-in profile covers. The slice is an
// immutable snapshot: callers must not modify it.
func (b *Backend) CustomProfiles() []*model.KindProfile {
	if set := b.customSet(); set != nil {
		return set.profiles
	}
	return nil
}

// Cached reports whether the backend's cache already watches a kind in all namespaces, in the
// scope its profile gives it (a view, a drawer or a lookup uses it, or used it within the warm
// period), so that reading it costs no request. A local kind is always cached: its store asks no
// API server.
func (b *Backend) Cached(p *model.KindProfile) bool {
	if p == nil {
		return false
	}
	if p.Local {
		return true
	}
	if isApplications(p) {
		return b.apps.running(b.Scope(p))
	}
	if isHelmReleases(p) { // both drivers' records (helmdriver.go)
		return b.releasesCached(p)
	}
	return b.Cache != nil && (b.Cache.Running(b.Scope(p)) || b.namespacesRunning(p))
}

// customKindsNote explains why a custom resource kind key is unknown: the definitions are still
// being listed, or the user may not list them. It is empty for other keys.
func (b *Backend) customKindsNote(key string) string {
	set := b.customSet()
	if set == nil || !strings.Contains(key, ".") {
		return ""
	}
	switch {
	case set.err != "":
		return " (custom resource kinds are unavailable: " + set.err + ")"
	case !set.synced:
		return " (the cluster's custom resource definitions are still being listed; try again shortly)"
	}
	return ""
}
