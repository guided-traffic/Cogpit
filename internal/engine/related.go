package engine

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"slices"
	"sync"
	"sync/atomic"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
)

// relatedSet serves the related-objects indexes a profile declares in Needs (architecture §4.4):
// it holds a cache handle on each index's kind for as long as its owner (a view, the drawer, one
// read) runs, counts the changes of those kinds, and builds an index from the cache when a lookup
// needs it and the kind changed since the last build. Nothing here lists or watches by itself: the
// handles share the cache's informers with every other consumer.
type relatedSet struct {
	needs    []model.RelatedIndex
	profiles map[string]*model.KindProfile // by kind key; a kind the cluster does not serve is absent
	handles  map[string]*cache.Handle      // by kind key
	unsubs   []func()

	gen atomic.Uint64 // bumped by every event of an indexed kind

	mu    sync.Mutex
	built map[string]*builtIndex // by index ID
}

type builtIndex struct {
	gen   uint64
	byKey map[string][]*unstructured.Unstructured
}

// newRelatedSet acquires the handles of needs' kinds on b; onChange, if set, runs (on an informer
// goroutine; it must not block) after every change of one of them. It returns nil for no needs.
func newRelatedSet(b *Backend, needs []model.RelatedIndex, onChange func()) *relatedSet {
	if len(needs) == 0 {
		return nil
	}
	r := &relatedSet{
		needs: needs, profiles: map[string]*model.KindProfile{}, handles: map[string]*cache.Handle{},
		built: map[string]*builtIndex{},
	}
	for _, ix := range needs {
		if _, done := r.handles[ix.Kind]; done {
			continue
		}
		p, ok := b.Profile(ix.Kind)
		if !ok {
			continue
		}
		h := b.Acquire(p)
		r.profiles[ix.Kind], r.handles[ix.Kind] = p, h
		r.unsubs = append(r.unsubs, h.Subscribe(func(cache.Event) {
			r.gen.Add(1)
			if onChange != nil {
				onChange()
			}
		}))
	}
	return r
}

// release gives the handles back; the informers stay warm.
func (r *relatedSet) release() {
	if r == nil {
		return
	}
	for _, u := range r.unsubs {
		u()
	}
	for _, h := range r.handles {
		h.Release()
	}
}

// synced reports whether every served kind of the set finished its initial list.
func (r *relatedSet) synced() bool {
	if r == nil {
		return true
	}
	for _, h := range r.handles {
		if !h.HasSynced() {
			return false
		}
	}
	return true
}

// waitSynced waits, bounded by ctx and syncTimeout, for the set's kinds. A kind that does not
// sync leaves its lookups empty, which is a state, not an error.
func (r *relatedSet) waitSynced(ctx context.Context) {
	if r == nil {
		return
	}
	wctx, cancel := context.WithTimeout(ctx, syncTimeout)
	defer cancel()
	for _, h := range r.handles {
		h.WaitForSync(wctx)
	}
}

// view returns the set's lookups at the current generation: one snapshot of rows reads every
// index at most once, however many rows it builds. A nil set yields model.NoRelated.
//
// The generation also says whether every kind had synced: an index is empty until its kind
// syncs, and an informer counts its initial list's last event (bumping gen) before it reports
// synced, with no event after. Rows built in between would otherwise stay cached at a generation
// that never changes, empty related columns and all. The events are counted before synced is
// read, so a view that says synced has every index full.
func (r *relatedSet) view() (model.Related, uint64) {
	if r == nil {
		return model.NoRelated, 0
	}
	g := r.gen.Load() << 1
	if r.synced() {
		g |= 1
	}
	return &relatedView{r: r, gen: g, idx: map[string]map[string][]*unstructured.Unstructured{}}, g
}

// index returns the index with id at generation gen, building it if the last build is older.
func (r *relatedSet) index(ix model.RelatedIndex, gen uint64) map[string][]*unstructured.Unstructured {
	h, ok := r.handles[ix.Kind]
	if !ok || !h.HasSynced() || ix.Keys == nil {
		return nil
	}
	id := ix.ID()
	r.mu.Lock()
	defer r.mu.Unlock()
	if b := r.built[id]; b != nil && b.gen == gen {
		return b.byKey
	}
	byKey := buildIndex(ix, KindList(h, r.profiles[ix.Kind]))
	r.built[id] = &builtIndex{gen: gen, byKey: byKey}
	return byKey
}

// buildIndex maps every object to its keys; an index function that panics on an object skips it.
func buildIndex(ix model.RelatedIndex, objs []*unstructured.Unstructured) map[string][]*unstructured.Unstructured {
	out := map[string][]*unstructured.Unstructured{}
	for _, o := range objs {
		for _, k := range indexKeys(ix, o) {
			out[k] = append(out[k], o)
		}
	}
	return out
}

func indexKeys(ix model.RelatedIndex, o *unstructured.Unstructured) (keys []string) {
	defer func() {
		if r := recover(); r != nil {
			slog.Default().Error("engine: related-objects index panicked", "index", ix.ID(), "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			keys = nil
		}
	}()
	return ix.Keys(o)
}

// relatedView is a relatedSet's lookups at one generation.
type relatedView struct {
	r   *relatedSet
	gen uint64
	idx map[string]map[string][]*unstructured.Unstructured // not shared between goroutines
}

// Objects implements model.Related. Only the indexes the profile declared are served.
func (v *relatedView) Objects(ix model.RelatedIndex, key string) []*unstructured.Unstructured {
	id := ix.ID()
	byKey, ok := v.idx[id]
	if !ok {
		declared := false
		for _, n := range v.r.needs {
			if n.ID() == id {
				ix, declared = n, true
				break
			}
		}
		if declared {
			byKey = v.r.index(ix, v.gen)
		}
		v.idx[id] = byKey
	}
	return byKey[key]
}

// Listed implements model.Related: the index is declared, its kind served and listed.
func (v *relatedView) Listed(ix model.RelatedIndex) bool {
	id := ix.ID()
	if !slices.ContainsFunc(v.r.needs, func(n model.RelatedIndex) bool { return n.ID() == id }) {
		return false
	}
	h, ok := v.r.handles[ix.Kind]
	return ok && h.HasSynced()
}

// BuildRows builds table rows for objects of a kind as its view would, related objects included:
// for a profile with Needs it holds the indexed kinds' handles for the call and waits (bounded by
// ctx and 15 seconds) for their initial lists. Commands that list rows outside a view use it.
func (s *Session) BuildRows(ctx context.Context, b *Backend, p *model.KindProfile, objs []*unstructured.Unstructured) []model.Row {
	rs := newRelatedSet(b, p.Needs, nil)
	defer rs.release()
	rs.waitSynced(ctx)
	rel, _ := rs.view()
	rows := make([]model.Row, 0, len(objs))
	for _, o := range objs {
		rows = append(rows, model.BuildRowWith(p, b.Context, o, rel))
	}
	return rows
}
