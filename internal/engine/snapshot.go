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
	unsub   func()

	mu     sync.Mutex
	params viewParams

	snap     atomic.Pointer[TableSnapshot]
	wake     chan struct{}
	quit     chan struct{}
	stopped  atomic.Bool
	panicked atomic.Bool
}

// startViewWorker acquires the view's handle and starts its loop. The first snapshot is built at
// once; an informer that is warm already serves it without a list request.
func (s *Session) startViewWorker(b *Backend, p *model.KindProfile, rc *rowCache, params viewParams) *viewWorker {
	w := &viewWorker{
		s: s, backend: b, profile: p, rows: rc, params: params,
		handle: b.Cache.Acquire(cache.Scope{GVR: p.GVR()}),
		wake:   make(chan struct{}, 1),
		quit:   make(chan struct{}),
	}
	w.unsub = w.handle.Subscribe(func(cache.Event) { signal(w.wake) })
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
	w.params = p
	w.mu.Unlock()
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
}

// unsettled reports whether the informer's sync state or error changed since the last snapshot:
// neither comes with a watch event.
func (w *viewWorker) unsettled() bool {
	last := w.snap.Load()
	if last == nil {
		return true
	}
	return last.Synced != w.handle.HasSynced() || last.Error != errText(w.handle.Err())
}

// rebuild builds, stores and announces a snapshot. A panic is reported once per view as a toast
// and does not end the loop.
func (w *viewWorker) rebuild() {
	defer func() {
		if r := recover(); r != nil {
			w.s.log.Error("engine: building a table snapshot failed", "view", w.profile.Key, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			if w.panicked.CompareAndSwap(false, true) {
				w.s.emit(Event{Type: EventToast, Level: "error", Message: "the " + w.profile.Title + " view failed to update: internal error"})
			}
		}
	}()
	w.mu.Lock()
	p := w.params
	w.mu.Unlock()

	synced := w.handle.HasSynced() // before List: a snapshot never claims a partial list is complete
	errMsg := errText(w.handle.Err())
	snap := w.rows.build(w.profile, w.backend.Context, w.handle.List(), p)
	snap.Synced = synced
	snap.Error = errMsg
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
	gen     uint32
	rows    map[string]*rowEntry
}

type rowEntry struct {
	rv  string
	gen uint32
	row model.Row
}

// build turns the objects into a snapshot: rows from the cache (rebuilt when the resourceVersion
// changed), filtered by namespaces and search text, sorted. Total counts the objects before
// filtering.
func (rc *rowCache) build(p *model.KindProfile, context string, objs []*unstructured.Unstructured, params viewParams) *TableSnapshot {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	if rc.rows == nil || rc.profile != p || rc.context != context {
		rc.rows = make(map[string]*rowEntry, len(objs))
		rc.profile, rc.context = p, context
	}
	rc.gen++

	var nsSet map[string]struct{}
	if p.Namespaced && len(params.namespaces) > 0 {
		nsSet = make(map[string]struct{}, len(params.namespaces))
		for _, ns := range params.namespaces {
			nsSet[ns] = struct{}{}
		}
	}
	filter := strings.TrimSpace(params.filter)

	kept := make([]*rowEntry, 0, len(objs))
	for _, obj := range objs {
		key := string(obj.GetUID())
		if key == "" {
			key = obj.GetNamespace() + "/" + obj.GetName()
		}
		rv := obj.GetResourceVersion()
		e := rc.rows[key]
		if e == nil || e.rv != rv || rv == "" {
			e = &rowEntry{rv: rv, row: model.BuildRow(p, context, obj)}
			rc.rows[key] = e
		}
		e.gen = rc.gen
		if nsSet != nil {
			if _, ok := nsSet[e.row.Ref.Namespace]; !ok {
				continue
			}
		}
		if filter != "" && !model.MatchesFilter(e.row, filter) {
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
	return &TableSnapshot{
		View: p.Key, Context: context, Title: p.Title, Namespaced: p.Namespaced,
		Columns: p.Columns, Rows: rows, Sort: params.sort,
		Namespaces: slices.Clone(params.namespaces), Filter: params.filter,
		Total: len(objs),
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
		if typ == model.ColumnNumber || typ == model.ColumnAge {
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
// age cell without a timestamp. A number cell always shows its number.
func emptyCell(typ model.ColumnType, c model.Cell) bool {
	switch typ {
	case model.ColumnNumber:
		return false
	case model.ColumnAge:
		return c.Num == 0 && c.Text == ""
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
