// Package cache is the only place Cogpit lists and watches. It keeps one shared client-go
// informer per (kind, namespace scope) of a cluster, reference-counted by handles and kept warm for
// a grace period after the last handle is released; it drops managedFields before an object
// enters the store; views, commands and MCP tools read only from it (architecture §5, D-13).
//
// Layer L1.
package cache

import (
	"context"
	"fmt"
	"log/slog"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	toolscache "k8s.io/client-go/tools/cache"
)

// DefaultWarmPeriod is how long an informer keeps running after its last handle is released.
const DefaultWarmPeriod = 5 * time.Minute

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
	Events    int64     `json:"events"`              // watch events received since start
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

	refs      int         // guarded by Cache.mu
	warmTimer *time.Timer // guarded by Cache.mu
	warmUntil time.Time   // guarded by Cache.mu

	events atomic.Int64

	errMu   sync.Mutex
	lastErr error

	subMu   sync.RWMutex
	subs    map[int]func(Event)
	nextSub int
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
	return &Handle{scope: scope, e: e, c: c}
}

// start creates and runs the informer for scope. Called with c.mu held.
func (c *Cache) start(scope Scope) *entry {
	lw := c.src.ListerWatcher(scope.GVR, scope.Namespace)
	inf := toolscache.NewSharedIndexInformerWithOptions(lw, &unstructured.Unstructured{}, toolscache.SharedIndexInformerOptions{
		ResyncPeriod: 0, // no periodic resyncs: only watch events after the initial list
		Indexers:     toolscache.Indexers{toolscache.NamespaceIndex: toolscache.MetaNamespaceIndexFunc},
	})
	e := &entry{scope: scope, inf: inf, subs: map[int]func(Event){}}
	if err := inf.SetTransform(stripManagedFields); err != nil {
		c.log.Warn("cache: transform not installed", "resource", scope.GVR.String(), "err", err)
	}
	if err := inf.SetWatchErrorHandlerWithContext(func(ctx context.Context, r *toolscache.Reflector, err error) {
		e.setErr(err)
		toolscache.DefaultWatchErrorHandler(ctx, r, err)
	}); err != nil {
		c.log.Warn("cache: watch error handler not installed", "resource", scope.GVR.String(), "err", err)
	}
	if _, err := inf.AddEventHandler(toolscache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj any) { e.dispatch(Added, obj) },
		UpdateFunc: func(_, obj any) { e.dispatch(Updated, obj) },
		DeleteFunc: func(obj any) { e.dispatch(Deleted, obj) },
	}); err != nil {
		c.log.Warn("cache: event handler not installed", "resource", scope.GVR.String(), "err", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	go func() {
		defer func() {
			if r := recover(); r != nil {
				e.setErr(fmt.Errorf("informer for %s stopped: internal error: %v", scope.GVR.Resource, r))
				c.log.Error("cache: informer panicked", "resource", scope.GVR.String(), "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			}
		}()
		inf.RunWithContext(ctx)
	}()
	return e
}

// release drops one reference; the last one starts the warm period.
func (c *Cache) release(e *entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e.refs--
	if e.refs > 0 || c.closed {
		return
	}
	e.warmUntil = time.Now().Add(c.opts.WarmPeriod)
	e.warmTimer = time.AfterFunc(c.opts.WarmPeriod, func() { c.expire(e) })
}

// expire stops an informer whose warm period ended without a new handle.
func (c *Cache) expire(e *entry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e.refs > 0 || c.informers[e.scope] != e {
		return
	}
	delete(c.informers, e.scope)
	e.cancel()
}

// Stats returns one entry per running informer.
func (c *Cache) Stats() []InformerStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]InformerStats, 0, len(c.informers))
	for _, e := range c.informers {
		st := InformerStats{
			Scope:     e.scope,
			Objects:   len(e.inf.GetStore().ListKeys()),
			Handles:   e.refs,
			Synced:    e.inf.HasSynced(),
			Events:    e.events.Load(),
			WarmUntil: e.warmUntil,
		}
		if err := e.err(); err != nil {
			st.Error = err.Error()
		}
		out = append(out, st)
	}
	return out
}

// Close stops every informer immediately.
func (c *Cache) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	for scope, e := range c.informers {
		if e.warmTimer != nil {
			e.warmTimer.Stop()
		}
		e.cancel()
		delete(c.informers, scope)
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

func (e *entry) dispatch(typ EventType, obj any) {
	e.events.Add(1)
	e.clearErr()
	if d, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
		obj = d.Obj
	}
	u, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return
	}
	e.subMu.RLock()
	defer e.subMu.RUnlock()
	for _, fn := range e.subs {
		callSafely(fn, Event{Type: typ, Object: u})
	}
}

// callSafely runs a subscriber; a panicking subscriber must not stop the informer.
func callSafely(fn func(Event), ev Event) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("cache: subscriber panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	fn(ev)
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

// stripManagedFields is the informer transform: managedFields are large and only the editor's
// "managed fields" toggle needs them, which fetches them on demand.
func stripManagedFields(obj any) (any, error) {
	if u, ok := obj.(*unstructured.Unstructured); ok {
		unstructured.RemoveNestedField(u.Object, "metadata", "managedFields")
	}
	return obj, nil
}

// Handle is one consumer's claim on an informer.
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
	items := h.e.inf.GetStore().List()
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

// Subscribe calls fn for every change after it returns. fn runs on the informer's goroutine and
// must not block. The returned function removes the subscription.
func (h *Handle) Subscribe(fn func(Event)) (cancel func()) {
	if h.e == nil {
		return func() {}
	}
	e := h.e
	e.subMu.Lock()
	id := e.nextSub
	e.nextSub++
	e.subs[id] = fn
	e.subMu.Unlock()
	return func() {
		e.subMu.Lock()
		delete(e.subs, id)
		e.subMu.Unlock()
	}
}

// Err returns the informer's last list or watch error, or nil (a forbidden kind, a lost
// connection). It clears when the next event arrives. Errors are states the UI shows, not
// failures.
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
