package testkit

import (
	"context"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// GatedSource wraps a cache.Source and holds every list of one resource until Open: a test decides
// when an informer's initial list arrives, so that it can subscribe, or hold something else,
// before it does.
type GatedSource struct {
	inner cache.Source
	gvr   schema.GroupVersionResource
	gate  chan struct{}
	once  sync.Once
}

// NewGatedSource returns a source that holds the lists of gvr until Open.
func NewGatedSource(inner cache.Source, gvr schema.GroupVersionResource) *GatedSource {
	return &GatedSource{inner: inner, gvr: gvr, gate: make(chan struct{})}
}

// Open lets the held lists and every later one through.
func (g *GatedSource) Open() { g.once.Do(func() { close(g.gate) }) }

// ListerWatcher implements cache.Source.
func (g *GatedSource) ListerWatcher(scope cache.Scope) toolscache.ListerWatcher {
	lw := g.inner.ListerWatcher(scope)
	if scope.GVR != g.gvr {
		return lw
	}
	return &gatedLW{gate: g.gate, inner: lw}
}

type gatedLW struct {
	gate  <-chan struct{}
	inner toolscache.ListerWatcher
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch.
func (l *gatedLW) IsWatchListSemanticsUnSupported() bool { return true }

func (l *gatedLW) List(opts metav1.ListOptions) (runtime.Object, error) {
	return l.ListWithContext(context.Background(), opts)
}

func (l *gatedLW) ListWithContext(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
	select {
	case <-l.gate:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return toolscache.ToListerWithContext(l.inner).ListWithContext(ctx, opts)
}

func (l *gatedLW) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return l.WatchWithContext(context.Background(), opts)
}

func (l *gatedLW) WatchWithContext(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	return toolscache.ToWatcherWithContext(l.inner).WatchWithContext(ctx, opts)
}
