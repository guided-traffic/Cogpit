package engine

import (
	"cmp"
	"fmt"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
)

// viewWorker keeps the snapshot of the open view current. It holds a cache handle on the view's
// kind in all namespaces (the namespace filter is applied client-side in the MVP), marks the view
// dirty on every watch event, and rebuilds the snapshot at most once per SnapshotInterval.
type viewWorker struct {
	s       *Session
	backend *Backend
	profile *model.KindProfile
	rows    *rowCache
	handle  *cache.Handle
	plan    nsPlan // how the kind is read: in all namespaces, one by one, or not yet (accessiblens_plan.go)
	unsub   func()
	related *relatedSet // the indexes of the profile's Needs; nil without
	usage   *usageFeed  // the metrics of a kind with CPU and Memory columns; nil without or while they are hidden (guarded by mu)

	mu     sync.Mutex
	params viewParams

	snap     atomic.Pointer[TableSnapshot]
	wake     chan struct{}
	quit     chan struct{}
	stopped  atomic.Bool
	panicked atomic.Bool
}

// startViewWorker acquires the view's handle and starts its loop. The first snapshot is built at
// once; an informer that is warm already serves it without a list request. The kind is read as
// plan says; a pending plan reads nothing until the session starts the view again with its plan.
func (s *Session) startViewWorker(b *Backend, p *model.KindProfile, rc *rowCache, params viewParams, plan nsPlan) *viewWorker {
	w := &viewWorker{
		s: s, backend: b, profile: p, rows: rc, params: params, plan: plan,
		handle: b.acquireForView(p, plan),
		wake:   make(chan struct{}, 1),
		quit:   make(chan struct{}),
	}
	w.unsub = w.handle.Subscribe(func(cache.Event) { signal(w.wake) })
	w.related = newRelatedSet(b, p.Needs, func() { signal(w.wake) })
	w.followHidden(params.metrics) // the metrics feed, unless the cluster's settings hide both columns
	signal(w.wake)
	s.goSafe("table view "+p.Key, func() {
		runCoalesced(w.quit, w.wake, s.opts.SnapshotInterval, w.unsettled, w.rebuild)
	})
	return w
}

// setParams hands the view new namespaces, filter or sort; a change produces a snapshot within
// one interval.
func (w *viewWorker) setParams(p viewParams) {
	w.mu.Lock()
	changed := !w.params.equal(p)
	metricsChanged := w.params.metrics != p.metrics
	w.params = p
	w.mu.Unlock()
	if metricsChanged {
		w.followHidden(p.metrics)
	}
	if changed {
		signal(w.wake)
	}
}

// stop ends the loop and releases the handle; the cache keeps the informer warm. It does not wait
// for a snapshot being built: that one is dropped instead of emitted.
func (w *viewWorker) stop() {
	if !w.stopped.CompareAndSwap(false, true) {
		return
	}
	close(w.quit)
	w.unsub()
	w.handle.Release()
	w.related.release()
	w.mu.Lock()
	usage := w.usage
	w.usage = nil
	w.mu.Unlock()
	usage.release()
}

// unsettled reports whether the informer's sync state or error changed since the last snapshot:
// neither comes with a watch event.
func (w *viewWorker) unsettled() bool {
	last := w.snap.Load()
	if last == nil {
		return true
	}
	return last.Synced != w.synced() || last.Error != errText(w.handle.Err())
}

// synced reports whether the view's kind and the kinds of its related-objects indexes finished
// their initial lists.
func (w *viewWorker) synced() bool { return w.handle.HasSynced() && w.related.synced() }

// rebuild builds, stores and announces a snapshot. A panic is reported once per view as a toast
// and does not end the loop.
func (w *viewWorker) rebuild() {
	defer func() {
		if r := recover(); r != nil {
			w.s.log.Error("engine: building a table snapshot failed", "view", w.profile.Key, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			if w.panicked.CompareAndSwap(false, true) {
				w.s.emit(Event{Type: EventToast, Level: LevelError, Message: "the " + w.profile.Title + " view failed to update: internal error"})
			}
		}
	}()
	w.mu.Lock()
	p := w.params
	usage := w.usage
	w.mu.Unlock()

	synced := w.synced() // before List: a snapshot never claims a partial list is complete
	herr := w.handle.Err()
	errMsg := errText(herr)
	w.followRefusal(herr) // a cluster-wide watch refused: read namespace by namespace (accessiblens_plan.go)
	rel, relGen := w.related.view()
	snap := w.rows.buildWith(w.profile, w.backend.Context, KindList(w.handle, w.profile), p, rel, relGen, usage)
	snap.Synced = synced
	snap.Error = errMsg
	snap.Watched, snap.Denied = w.plan.watch, w.plan.denied
	snap.Revision = w.s.state.snapshots.Add(1)
	if w.stopped.Load() {
		return
	}
	w.snap.Store(snap)
	w.s.emit(Event{Type: EventTable, View: w.profile.Key, Revision: snap.Revision})
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// rowCache holds the rows of one view, keyed by UID, so that a snapshot rebuilds only the rows
// whose object changed (resourceVersion). Its mutex is held for a whole build: a stopped worker
// and its successor never build from the same cache at once.
type rowCache struct {
	mu      sync.Mutex
	profile *model.KindProfile
	context string
	relGen  uint64 // the related objects' generation the rows were built at (profiles with Needs)
	gen     uint32
	rows    map[string]*rowEntry
}

type rowEntry struct {
	rv  string
	gen uint32
	// row is the row as shown: what the profile built, or with the CPU and Memory cells filled from
	// the metrics (profiles with Usage). base is the profile's row while row is patched.
	row     model.Row
	base    model.Row
	patched bool   // row is base with metrics in it
	ugen    uint64 // the feed's generation + 1 that row was patched at; 0 for never
}

// build turns the objects into a snapshot: rows from the cache (rebuilt when the resourceVersion
// changed, and all of them when the related objects of a profile with Needs changed), filtered by
// namespaces and search text, sorted. Total counts the objects before filtering.
func (rc *rowCache) build(p *model.KindProfile, context string, objs []*unstructured.Unstructured, params viewParams, rel model.Related, relGen uint64) *TableSnapshot {
	return rc.buildWith(p, context, objs, params, rel, relGen, nil)
}

// buildWith is build with the metrics of a profile that has Usage: rows of objects that have a
// measurement show it in their CPU and Memory cells. The base rows stay as the profile built them,
// so a new measurement patches the cells without building the rows again.
func (rc *rowCache) buildWith(p *model.KindProfile, context string, objs []*unstructured.Unstructured, params viewParams, rel model.Related, relGen uint64, usage *usageFeed) *TableSnapshot {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.rows == nil || rc.profile != p || rc.context != context || (len(p.Needs) > 0 && rc.relGen != relGen) {
		rc.rows = make(map[string]*rowEntry, len(objs))
		rc.profile, rc.context, rc.relGen = p, context, relGen
	}
	rc.gen++

	var nsSet map[string]struct{}
	if p.Namespaced && len(params.namespaces) > 0 {
		nsSet = make(map[string]struct{}, len(params.namespaces))
		for _, ns := range params.namespaces {
			nsSet[ns] = struct{}{}
		}
	}
	match, filterErr := params.matcher()
	groupCol, groupSet := groupFilter(p, params.groups)

	cpuCol, memCol := -1, -1
	var ugen uint64
	if usage != nil && p.UsageCells != nil {
		cpuCol, memCol = p.ColumnIndex(model.ColumnCPU), p.ColumnIndex(model.ColumnMemory)
		ugen = usage.generation()
	}
	patching := cpuCol >= 0 || memCol >= 0
	ugen++ // 0 means "never patched"

	kept := make([]*rowEntry, 0, len(objs))
	for _, obj := range objs {
		key := string(obj.GetUID())
		if key == "" {
			key = obj.GetNamespace() + "/" + obj.GetName()
		}
		rv := obj.GetResourceVersion()
		e := rc.rows[key]
		if e == nil || e.rv != rv || rv == "" {
			e = &rowEntry{rv: rv, row: model.BuildRowWith(p, context, obj, rel)}
			rc.rows[key] = e
		}
		e.gen = rc.gen
		switch {
		case patching && e.ugen != ugen:
			e.ugen = ugen
			base := e.row
			if e.patched {
				base = e.base
			}
			if u, ok := usage.lookup(obj.GetNamespace(), obj.GetName()); ok {
				e.base, e.row, e.patched = base, withUsage(p, base, obj, u, cpuCol, memCol), true
			} else {
				e.row, e.patched = base, false
			}
		case !patching && e.patched: // the feed is gone: back to the profile's row
			e.row, e.patched, e.ugen = e.base, false, 0
		}
		if nsSet != nil {
			if _, ok := nsSet[e.row.Ref.Namespace]; !ok {
				continue
			}
		}
		if groupSet != nil {
			if _, ok := groupSet[cellAt(&e.row, groupCol).Text]; !ok {
				continue
			}
		}
		if !match.All() && !match.Match(e.row) {
			continue
		}
		kept = append(kept, e)
	}
	if len(rc.rows) > len(objs) { // objects were deleted: drop their rows
		for k, e := range rc.rows {
			if e.gen != rc.gen {
				delete(rc.rows, k)
			}
		}
	}

	sortEntries(kept, p.Columns, params.sort)
	rows := make([]model.Row, len(kept))
	for i, e := range kept {
		rows[i] = e.row
	}
	cols, metricsHidden := hideMetricColumns(p, ColumnsShown(p.Columns, params.hidden, params.hiddenSet), params.metrics)
	return &TableSnapshot{
		View: p.Key, Context: context, Title: p.Title, Namespaced: p.Namespaced,
		Columns: cols,
		Order:   ColumnOrder(p.Columns, params.order),
		Rows:    rows, Sort: params.sort,
		Namespaces: slices.Clone(params.namespaces), Filter: params.filter,
		FilterOptions: params.filterOpts, FilterError: filterErr,
		Total: len(objs), MetricsHidden: metricsHidden,
	}
}

// sortEntries sorts rows by a column: text case-insensitively, numbers and ages by Cell.Num. Empty
// cells (nothing to show) sort last in both directions; ties fall back to namespace, then name, so
// the order is total and stable across snapshots. An unknown column sorts by namespace and name.
func sortEntries(es []*rowEntry, cols []model.Column, spec SortSpec) {
	ci := -1
	for i, c := range cols {
		if c.ID == spec.Column {
			ci = i
			break
		}
	}
	if ci < 0 {
		slices.SortFunc(es, func(a, b *rowEntry) int { return compareRefs(&a.row, &b.row) })
		return
	}
	typ := cols[ci].Type
	slices.SortFunc(es, func(a, b *rowEntry) int {
		return compareRows(&a.row, &b.row, ci, typ, spec.Desc)
	})
}

// SortRows sorts rows in place the way table snapshots are sorted. Commands use it for results
// that are not a view's snapshot (resources.list).
func SortRows(rows []model.Row, cols []model.Column, spec SortSpec) {
	ci := slices.IndexFunc(cols, func(c model.Column) bool { return c.ID == spec.Column })
	if ci < 0 {
		slices.SortFunc(rows, func(a, b model.Row) int { return compareRefs(&a, &b) })
		return
	}
	typ := cols[ci].Type
	slices.SortFunc(rows, func(a, b model.Row) int { return compareRows(&a, &b, ci, typ, spec.Desc) })
}

func compareRows(a, b *model.Row, ci int, typ model.ColumnType, desc bool) int {
	ca, cb := cellAt(a, ci), cellAt(b, ci)
	ea, eb := emptyCell(typ, ca), emptyCell(typ, cb)
	if ea != eb {
		if ea {
			return 1
		}
		return -1
	}
	if !ea {
		var c int
		if typ == model.ColumnNumber || typ == model.ColumnAge || typ == model.ColumnUntil || typ == model.ColumnTime || typ == model.ColumnWarning || typ == model.ColumnMetric {
			c = cmp.Compare(ca.Num, cb.Num)
		} else {
			c = compareFold(ca.Text, cb.Text)
		}
		if desc {
			c = -c
		}
		if c != 0 {
			return c
		}
	}
	return compareRefs(a, b)
}

func compareRefs(a, b *model.Row) int {
	if c := strings.Compare(a.Ref.Namespace, b.Ref.Namespace); c != 0 {
		return c
	}
	return strings.Compare(a.Ref.Name, b.Ref.Name)
}

func cellAt(r *model.Row, i int) model.Cell {
	if i < len(r.Cells) {
		return r.Cells[i]
	}
	return model.Cell{}
}

// emptyCell reports whether a cell shows nothing (model.CellText): a text cell without text, an
// age cell without a timestamp, a warning cell without a warning. A number cell always shows its
// number.
func emptyCell(typ model.ColumnType, c model.Cell) bool {
	switch typ {
	case model.ColumnNumber:
		return false
	case model.ColumnAge, model.ColumnUntil, model.ColumnTime:
		return c.Num == 0 && c.Text == ""
	case model.ColumnMetric:
		return c.Text == "" // no measurement
	case model.ColumnWarning:
		return c.Num == 0 && c.Status == model.StatusNone
	}
	return c.Text == ""
}

// compareFold compares two strings case-insensitively without allocating for ASCII, which is
// what names, namespaces and statuses are.
func compareFold(a, b string) int {
	n := min(len(a), len(b))
	for i := range n {
		ca, cb := a[i], b[i]
		if ca >= utf8.RuneSelf || cb >= utf8.RuneSelf {
			return strings.Compare(strings.ToLower(a[i:]), strings.ToLower(b[i:]))
		}
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return cmp.Compare(ca, cb)
		}
	}
	return cmp.Compare(len(a), len(b))
}
