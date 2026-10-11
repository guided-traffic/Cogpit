// Package cache is the only place Cogpit lists and watches. It keeps one shared client-go
// informer per (kind, namespace scope) of a cluster, reference-counted by handles and kept warm for
// a grace period after the last handle is released; it drops managedFields before an object
// enters the store; views, commands and MCP tools read only from it (architecture §5, D-13).
//
// Layer L1.
//
// # Errors are states
//
// A list or watch that fails (a forbidden kind, a lost connection) is not a failure of the
// cache: the informer retries with client-go's backoff, and Handle.Err reports the last error
// until a list or a watch succeeds again.
//
// # Panics: what is and is not recovered
//
// A client-go informer runs on several goroutines: the one that calls Run (it also processes the
// queue and updates the store), the reflector (list, watch, decoding, the transform), and one
// pair per event handler. client-go guards its goroutines with utilruntime.HandleCrash, which logs
// and then re-panics while utilruntime.ReallyCrash is true, the default. In this package:
//
//   - Code the cache runs on client-go's goroutines recovers on its own: the Source's List and
//     Watch (a panic becomes a list or watch error), the managedFields transform (the object is
//     kept unchanged), and the event dispatch to subscribers (a panicking subscriber is logged and
//     skipped).
//   - The goroutine that runs the informer recovers a panic and reports it through Handle.Err,
//     which covers the queue processing and store updates on that goroutine.
//   - Not recovered: a panic inside client-go's own reflector or event-handler goroutines, or in
//     a Source's watch.Interface producer goroutine. With ReallyCrash true they end the process.
//     Whether the app sets utilruntime.ReallyCrash = false at start-up (then client-go logs and
//     carries on; an informer that stops that way is reported through Handle.Err) is a process-wide
//     decision for cmd/, not for this package.
package cache

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"
)

// DefaultWarmPeriod is how long an informer keeps running after its last handle is released.
const DefaultWarmPeriod = 5 * time.Minute

// closeTimeout bounds how long Close waits for the informers' goroutines to end.
const closeTimeout = 5 * time.Second

// Source creates the list-watchers informers run on: client-go's dynamic client for a real
// cluster (DynamicSource), a synthetic generator in demo mode, an in-memory fake in tests.
//
// A ListerWatcher that does not support WatchList semantics (streaming initial lists) must
// implement `IsWatchListSemanticsUnSupported() bool` returning true; client-go 0.37's reflector
// otherwise expects a WatchList-capable server.
type Source interface {
	ListerWatcher(gvr schema.GroupVersionResource, namespace string) toolscache.ListerWatcher
}

// Scope is what one informer watches: a resource, in one namespace or (empty) in all.
type Scope struct {
	GVR       schema.GroupVersionResource `json:"gvr"`
	Namespace string                      `json:"namespace,omitempty"`
}

// String names the scope the way kubectl names resources: "pods", "deployments.apps",
// "pods in team-a".
func (s Scope) String() string {
	name := s.GVR.Resource
	if s.GVR.Group != "" {
		name += "." + s.GVR.Group
	}
	if s.Namespace != "" {
		name += " in " + s.Namespace
	}
	return name
}

// Options configure a Cache.
type Options struct {
	// WarmPeriod is how long an informer stays running after its last handle is released.
	// Zero means DefaultWarmPeriod.
	WarmPeriod time.Duration
	// Logger receives informer errors; nil means slog.Default().
	Logger *slog.Logger
}

// InformerStats describe one informer for the diagnostics view.
type InformerStats struct {
	Scope     Scope     `json:"scope"`
	Objects   int       `json:"objects"`
	Handles   int       `json:"handles"`
	Synced    bool      `json:"synced"`
	Events    int64     `json:"events"`              // events delivered since start, initial list included
	WarmUntil time.Time `json:"warmUntil,omitempty"` // set while no handle holds the informer
	Error     string    `json:"error,omitempty"`
}

// Cache holds the informers of one cluster. It is safe for concurrent use.
type Cache struct {
	src  Source
	opts Options
	log  *slog.Logger

	mu        sync.Mutex
	informers map[Scope]*entry
	closed    bool
}

// New returns a cache over src. No informer runs until the first Acquire.
func New(src Source, opts Options) *Cache {
	if opts.WarmPeriod <= 0 {
		opts.WarmPeriod = DefaultWarmPeriod
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	return &Cache{src: src, opts: opts, log: opts.Logger, informers: map[Scope]*entry{}}
}

// entry is one running informer and its bookkeeping.
type entry struct {
	scope  Scope
	inf    toolscache.SharedIndexInformer
	cancel context.CancelFunc
	done   chan struct{} // closed when the informer's goroutine has returned
	log    *slog.Logger

	refs      int         // guarded by Cache.mu
	warmTimer *time.Timer // guarded by Cache.mu
	warmUntil time.Time   // guarded by Cache.mu
	warmGen   uint64      // guarded by Cache.mu; tells a stale timer from the current one

	events atomic.Int64

	errMu   sync.Mutex
	lastErr error

	// subs is replaced, never mutated (copy on write): dispatch reads it without a lock, so a
	// subscriber may subscribe or cancel from inside its callback.
	subMu sync.Mutex // serialises writers of subs
	subs  atomic.Pointer[[]*subscription]
}

type subscription struct {
	fn     func(Event)
	active atomic.Bool
}

// Acquire returns a handle on the informer for scope, starting it if it is not running. The
// caller must Release the handle when it stops showing or using the data. After Close, Acquire
// returns a handle on an empty, never-syncing store.
func (c *Cache) Acquire(scope Scope) *Handle {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return &Handle{scope: scope}
	}
	e := c.informers[scope]
	if e == nil {
		e = c.start(scope)
		c.informers[scope] = e
	}
	e.refs++
	if e.warmTimer != nil {
		e.warmTimer.Stop()
		e.warmTimer = nil
		e.warmUntil = time.Time{}
	}
	e.warmGen++ // a timer that already fired and waits for c.mu must not expire the entry
	return &Handle{scope: scope, e: e, c: c}
}

// start creates and runs the informer for scope. Called with c.mu held.
func (c *Cache) start(scope Scope) *entry {
	e := &entry{scope: scope, done: make(chan struct{}), log: c.log.With("scope", scope.String())}
	e.subs.Store(&[]*subscription{})
	lw := &trackedLW{e: e, inner: c.listerWatcher(e, scope)}
	inf := toolscache.NewSharedIndexInformerWithOptions(lw, &unstructured.Unstructured{}, toolscache.SharedIndexInformerOptions{
		ResyncPeriod:      0, // no periodic resyncs: only watch events after the initial list
		Indexers:          toolscache.Indexers{toolscache.NamespaceIndex: toolscache.MetaNamespaceIndexFunc},
		ObjectDescription: scope.String(),
	})
	e.inf = inf
	if err := inf.SetTransform(e.stripManagedFields); err != nil {
		e.log.Warn("cache: transform not installed", "err", err)
	}
	if err := inf.SetWatchErrorHandlerWithContext(func(_ context.Context, _ *toolscache.Reflector, err error) {
		e.watchError(err)
	}); err != nil {
		e.log.Warn("cache: watch error handler not installed", "err", err)
	}
	if _, err := inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { e.dispatch(Added, obj) },
		UpdateFunc: func(_, obj any) { e.dispatch(Updated, obj) },
		DeleteFunc: func(obj any) { e.dispatch(Deleted, obj) },
	}); err != nil {
		e.log.Warn("cache: event handler not installed", "err", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go func() {
		defer close(e.done)
		defer func() {
			if r := recover(); r != nil {
				e.setErr(fmt.Errorf("the informer for %s stopped: internal error: %v", scope, r))
				e.log.Error("cache: informer panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			}
		}()
		inf.RunWithContext(ctx)
		if ctx.Err() == nil {
			// Run returned on its own: client-go swallowed a panic (utilruntime.ReallyCrash is
			// false) or refused to run twice. The store is frozen from here on.
			e.setErr(fmt.Errorf("the informer for %s stopped unexpectedly", scope))
			e.log.Error("cache: informer stopped unexpectedly")
		}
	}()
	return e
}

// listerWatcher asks the source for a list-watcher, turning a panic or a nil result into a
// list-watcher that fails, so the error shows on the handle instead of ending the process.
func (c *Cache) listerWatcher(e *entry, scope Scope) (lw toolscache.ListerWatcher) {
	defer func() {
		if r := recover(); r != nil {
			e.log.Error("cache: source panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			lw = failingLW{err: fmt.Errorf("cannot watch %s: internal error: %v", scope, r)}
		}
	}()
	lw = c.src.ListerWatcher(scope.GVR, scope.Namespace)
	if lw == nil {
		lw = failingLW{err: fmt.Errorf("cannot watch %s: the source has no list-watcher for it", scope)}
	}
	return lw
}

// release drops one reference; the last one starts the warm period.
func (c *Cache) release(e *entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e.refs--
	if e.refs > 0 || c.closed {
		return
	}
	e.warmGen++
	gen := e.warmGen
	e.warmUntil = time.Now().Add(c.opts.WarmPeriod)
	e.warmTimer = time.AfterFunc(c.opts.WarmPeriod, func() { c.expire(e, gen) })
}

// expire stops an informer whose warm period ended without a new handle.
func (c *Cache) expire(e *entry, gen uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e.refs > 0 || e.warmGen != gen || c.informers[e.scope] != e {
		return
	}
	delete(c.informers, e.scope)
	e.warmTimer = nil
	e.cancel()
}

// Stats returns one entry per running informer.
func (c *Cache) Stats() []InformerStats {
	type row struct {
		e         *entry
		refs      int
		warmUntil time.Time
	}
	c.mu.Lock()
	rows := make([]row, 0, len(c.informers))
	for _, e := range c.informers {
		rows = append(rows, row{e: e, refs: e.refs, warmUntil: e.warmUntil})
	}
	c.mu.Unlock()

	// Counting the store's keys walks the store; do it without holding c.mu.
	out := make([]InformerStats, 0, len(rows))
	for _, r := range rows {
		st := InformerStats{
			Scope:     r.e.scope,
			Objects:   len(r.e.inf.GetStore().ListKeys()),
			Handles:   r.refs,
			Synced:    r.e.inf.HasSynced(),
			Events:    r.e.events.Load(),
			WarmUntil: r.warmUntil,
		}
		if err := r.e.err(); err != nil {
			st.Error = err.Error()
		}
		out = append(out, st)
	}
	return out
}

// Close stops every informer and waits (up to a few seconds) for their goroutines to end, so no
// request is in flight when the caller closes the connection. Handles keep the data they had.
func (c *Cache) Close() {
	c.mu.Lock()
	c.closed = true
	stopped := make([]*entry, 0, len(c.informers))
	for scope, e := range c.informers {
		if e.warmTimer != nil {
			e.warmTimer.Stop()
			e.warmTimer = nil
		}
		e.cancel()
		delete(c.informers, scope)
		stopped = append(stopped, e)
	}
	c.mu.Unlock()

	// Wait without c.mu: a subscriber still running on an informer goroutine may call into the
	// cache.
	deadline := time.NewTimer(closeTimeout)
	defer deadline.Stop()
	for _, e := range stopped {
		select {
		case <-e.done:
		case <-deadline.C:
			c.log.Warn("cache: informers did not stop in time", "timeout", closeTimeout)
			return
		}
	}
}

// EventType says what happened to an object.
type EventType string

// Event types.
const (
	Added   EventType = "added"
	Updated EventType = "updated"
	Deleted EventType = "deleted"
)

// Event is one change seen by an informer. Object is the new state (for Deleted, the last known
// state). Objects are shared with the store and must not be modified.
type Event struct {
	Type   EventType
	Object *unstructured.Unstructured
}

// dispatch runs on the informer's event-handler goroutine, one event at a time and in order.
func (e *entry) dispatch(typ EventType, obj any) {
	defer func() {
		// Nothing in here should panic; if it does, it must not reach client-go's HandleCrash.
		if r := recover(); r != nil {
			e.log.Error("cache: event dispatch panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	e.events.Add(1)
	if d, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
		obj = d.Obj
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	ev := Event{Type: typ, Object: u}
	for _, s := range *e.subs.Load() {
		if s.active.Load() {
			e.callSafely(s.fn, ev)
		}
	}
}

// callSafely runs a subscriber; a panicking subscriber must not stop the informer.
func (e *entry) callSafely(fn func(Event), ev Event) {
	defer func() {
		if r := recover(); r != nil {
			e.log.Error("cache: subscriber panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	fn(ev)
}

func (e *entry) subscribe(fn func(Event)) func() {
	s := &subscription{fn: fn}
	s.active.Store(true)
	e.subMu.Lock()
	old := *e.subs.Load()
	next := make([]*subscription, 0, len(old)+1)
	next = append(next, old...)
	next = append(next, s)
	e.subs.Store(&next)
	e.subMu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			s.active.Store(false)
			e.subMu.Lock()
			old := *e.subs.Load()
			next := make([]*subscription, 0, len(old))
			for _, x := range old {
				if x != s {
					next = append(next, x)
				}
			}
			e.subs.Store(&next)
			e.subMu.Unlock()
		})
	}
}

func (e *entry) setErr(err error) {
	e.errMu.Lock()
	e.lastErr = err
	e.errMu.Unlock()
}

func (e *entry) clearErr() {
	e.errMu.Lock()
	e.lastErr = nil
	e.errMu.Unlock()
}

func (e *entry) err() error {
	e.errMu.Lock()
	defer e.errMu.Unlock()
	return e.lastErr
}

// watchError receives the errors that end a list-and-watch cycle (client-go retries after it).
// The routine ones (a watch that expired, a closed stream) are not errors to show.
func (e *entry) watchError(err error) {
	if err == nil || apierrors.IsResourceExpired(err) || apierrors.IsGone(err) ||
		errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return
	}
	e.setErr(err)
	e.log.Warn("cache: list or watch failed; retrying", "err", err)
}

// stripManagedFields is the informer transform: managedFields are large and only the editor's
// "managed fields" toggle needs them, which fetches them on demand. It runs on the reflector's
// goroutine; if it panicked, the object is kept unchanged rather than lost.
func (e *entry) stripManagedFields(obj any) (out any, err error) {
	defer func() {
		if r := recover(); r != nil {
			e.log.Error("cache: transform panicked", "panic", fmt.Sprint(r))
			out, err = obj, nil
		}
	}()
	if u, ok := obj.(*unstructured.Unstructured); ok {
		unstructured.RemoveNestedField(u.Object, "metadata", "managedFields")
	}
	return obj, nil
}

// trackedLW wraps a Source's list-watcher: it records list and watch errors on the entry and
// clears them when a list or a watch succeeds, turns a panic in the source into an error, and
// passes the WatchList capability through to the reflector.
type trackedLW struct {
	e     *entry
	inner toolscache.ListerWatcher
}

var _ toolscache.ListerWatcherWithContext = (*trackedLW)(nil)

// IsWatchListSemanticsUnSupported reports what the wrapped list-watcher reports (false when it
// does not say), so the reflector streams initial lists exactly when the source supports it.
func (t *trackedLW) IsWatchListSemanticsUnSupported() bool {
	u, ok := t.inner.(interface{ IsWatchListSemanticsUnSupported() bool })
	return ok && u.IsWatchListSemanticsUnSupported()
}

func (t *trackedLW) List(opts metav1.ListOptions) (runtime.Object, error) {
	return t.ListWithContext(context.Background(), opts)
}

func (t *trackedLW) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return t.WatchWithContext(context.Background(), opts)
}

func (t *trackedLW) ListWithContext(ctx context.Context, opts metav1.ListOptions) (obj runtime.Object, err error) {
	defer t.recoverInto(&err, "list")
	obj, err = toolscache.ToListerWithContext(t.inner).ListWithContext(ctx, opts)
	t.observe(ctx, err)
	return obj, err
}

func (t *trackedLW) WatchWithContext(ctx context.Context, opts metav1.ListOptions) (w watch.Interface, err error) {
	defer t.recoverInto(&err, "watch")
	w, err = toolscache.ToWatcherWithContext(t.inner).WatchWithContext(ctx, opts)
	if err == nil && w == nil {
		err = errors.New("the source returned no watch")
	}
	t.observe(ctx, err)
	return w, err
}

// observe records the outcome of a list or a watch request. A request cut short because the
// informer is stopping says nothing about the cluster.
func (t *trackedLW) observe(ctx context.Context, err error) {
	switch {
	case ctx.Err() != nil:
	case err != nil:
		t.e.setErr(err)
	default:
		t.e.clearErr()
	}
}

func (t *trackedLW) recoverInto(err *error, op string) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("cannot %s %s: internal error: %v", op, t.e.scope, r)
		t.e.setErr(*err)
		t.e.log.Error("cache: source panicked", "op", op, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
	}
}

// failingLW stands in for a list-watcher the source could not provide.
type failingLW struct{ err error }

func (f failingLW) List(metav1.ListOptions) (runtime.Object, error)   { return nil, f.err }
func (f failingLW) Watch(metav1.ListOptions) (watch.Interface, error) { return nil, f.err }
func (f failingLW) IsWatchListSemanticsUnSupported() bool             { return true }

// Handle is one consumer's claim on an informer. Its reads stay valid after Release, but the data
// stops changing once the informer has expired.
type Handle struct {
	scope    Scope
	e        *entry
	c        *Cache
	released atomic.Bool
}

// Scope returns the handle's scope.
func (h *Handle) Scope() Scope { return h.scope }

// HasSynced reports whether the informer has finished its initial list.
func (h *Handle) HasSynced() bool { return h.e != nil && h.e.inf.HasSynced() }

// WaitForSync blocks until the informer has synced or ctx ends; it reports whether it synced.
func (h *Handle) WaitForSync(ctx context.Context) bool {
	if h.e == nil {
		<-ctx.Done()
		return false
	}
	if h.e.inf.HasSynced() {
		return true
	}
	t := time.NewTicker(10 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return h.e.inf.HasSynced()
		case <-t.C:
			if h.e.inf.HasSynced() {
				return true
			}
		}
	}
}

// List returns every object in the store. The objects are shared and must not be modified.
func (h *Handle) List() []*unstructured.Unstructured {
	if h.e == nil {
		return nil
	}
	return toUnstructured(h.e.inf.GetStore().List())
}

// ListNamespace returns the objects in one namespace, from the store's namespace index; for an
// informer scoped to one namespace it equals List. The objects are shared and must not be
// modified.
func (h *Handle) ListNamespace(namespace string) []*unstructured.Unstructured {
	if h.e == nil {
		return nil
	}
	items, err := h.e.inf.GetIndexer().ByIndex(toolscache.NamespaceIndex, namespace)
	if err != nil {
		return nil
	}
	return toUnstructured(items)
}

func toUnstructured(items []any) []*unstructured.Unstructured {
	out := make([]*unstructured.Unstructured, 0, len(items))
	for _, it := range items {
		if u, ok := it.(*unstructured.Unstructured); ok {
			out = append(out, u)
		}
	}
	return out
}

// Get returns one object by namespace and name.
func (h *Handle) Get(namespace, name string) (*unstructured.Unstructured, bool) {
	if h.e == nil {
		return nil, false
	}
	key := name
	if namespace != "" {
		key = namespace + "/" + name
	}
	it, ok, err := h.e.inf.GetStore().GetByKey(key)
	if err != nil || !ok {
		return nil, false
	}
	u, ok := it.(*unstructured.Unstructured)
	return u, ok
}

// Subscribe calls fn for every change after it returns; subscribe before reading List to miss
// nothing. fn runs on the informer's event goroutine, one event at a time in order, and must not
// block. The returned function removes the subscription; it is idempotent and may be called from
// inside fn. A delivery already running on the event goroutine may finish after it returns.
func (h *Handle) Subscribe(fn func(Event)) (cancel func()) {
	if h.e == nil || fn == nil {
		return func() {}
	}
	return h.e.subscribe(fn)
}

// Err returns the informer's last list or watch error, or nil (a forbidden kind, a lost
// connection). It clears when the next list or watch succeeds. Errors are states the UI shows,
// not failures.
func (h *Handle) Err() error {
	if h.e == nil {
		return nil
	}
	return h.e.err()
}

// Release gives the handle back. It is idempotent. After the last release the informer stays warm
// for the warm period, then stops.
func (h *Handle) Release() {
	if h.e == nil || !h.released.CompareAndSwap(false, true) {
		return
	}
	h.c.release(h.e)
}
