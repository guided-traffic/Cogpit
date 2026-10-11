// Package cache is the only place Cogpit lists and watches. It keeps one shared client-go
// informer per (kind, namespace scope) of a cluster, reference-counted by handles and kept warm for
// a grace period after the last handle is released; it drops managedFields before an object
// enters the store; views, commands and MCP tools read only from it (architecture §5, D-13).
//
// Layer L1.
package cache

import (
	"context"
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
	GVR       schema.GroupVersionResource
	Namespace string
}

// Options configure a Cache.
type Options struct {
	// WarmPeriod is how long an informer stays running after its last handle is released.
	// Zero means DefaultWarmPeriod.
	WarmPeriod time.Duration
}

// InformerStats describe one informer for the diagnostics view.
type InformerStats struct {
	Scope     Scope     `json:"scope"`
	Objects   int       `json:"objects"`
	Handles   int       `json:"handles"`
	Synced    bool      `json:"synced"`
	Events    int64     `json:"events"`              // watch events received since start
	WarmUntil time.Time `json:"warmUntil,omitempty"` // set while no handle holds the informer
}

// Cache holds the informers of one cluster. It is safe for concurrent use.
type Cache struct {
	src  Source
	opts Options
}

// New returns a cache over src. No informer runs until the first Acquire.
func New(src Source, opts Options) *Cache {
	return &Cache{src: src, opts: opts}
}

// Acquire returns a handle on the informer for scope, starting it if it is not running. The
// caller must Release the handle when it stops showing or using the data.
func (c *Cache) Acquire(scope Scope) *Handle {
	return &Handle{scope: scope}
}

// Stats returns one entry per running informer.
func (c *Cache) Stats() []InformerStats { return nil }

// Close stops every informer immediately.
func (c *Cache) Close() {}

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

// Handle is one consumer's claim on an informer.
type Handle struct {
	scope Scope
}

// Scope returns the handle's scope.
func (h *Handle) Scope() Scope { return h.scope }

// HasSynced reports whether the informer has finished its initial list.
func (h *Handle) HasSynced() bool { return false }

// WaitForSync blocks until the informer has synced or ctx ends; it reports whether it synced.
func (h *Handle) WaitForSync(ctx context.Context) bool {
	<-ctx.Done()
	return false
}

// List returns every object in the store. The objects are shared and must not be modified.
func (h *Handle) List() []*unstructured.Unstructured { return nil }

// Get returns one object by namespace and name.
func (h *Handle) Get(namespace, name string) (*unstructured.Unstructured, bool) {
	_, _ = namespace, name
	return nil, false
}

// Subscribe calls fn for every change after it returns. fn runs on the informer's goroutine and
// must not block. The returned function removes the subscription.
func (h *Handle) Subscribe(fn func(Event)) (cancel func()) {
	_ = fn
	return func() {}
}

// Err returns the informer's last list or watch error, or nil (a forbidden kind, a lost
// connection). Errors are states the UI shows, not failures.
func (h *Handle) Err() error { return nil }

// Release gives the handle back. It is idempotent. After the last release the informer stays warm
// for the warm period, then stops.
func (h *Handle) Release() {}
