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

// HoldList wraps a source so that every list of gvr waits until release is called: a cluster
// whose list of one resource takes a while after connecting (hundreds of custom resource
// definitions on a real cluster). Other resources list at once; watches are not held. release may
// be called more than once.
func HoldList(src cache.Source, gvr schema.GroupVersionResource) (held cache.Source, release func()) {
	ch := make(chan struct{})
	var once sync.Once
	return heldSource{Source: src, gvr: gvr, release: ch}, func() { once.Do(func() { close(ch) }) }
}

type heldSource struct {
	cache.Source
	gvr     schema.GroupVersionResource
	release <-chan struct{}
}

func (h heldSource) ListerWatcher(scope cache.Scope) toolscache.ListerWatcher {
	lw := h.Source.ListerWatcher(scope)
	if scope.GVR != h.gvr {
		return lw
	}
	return heldLW{lw: toolscache.ToListerWatcherWithContext(lw), release: h.release}
}

// heldLW lists after release; like the memory source, it tells client-go's reflector to list,
// then watch.
type heldLW struct {
	lw      toolscache.ListerWatcherWithContext
	release <-chan struct{}
}

func (l heldLW) List(o metav1.ListOptions) (runtime.Object, error) {
	return l.ListWithContext(context.Background(), o)
}

func (l heldLW) Watch(o metav1.ListOptions) (watch.Interface, error) {
	return l.WatchWithContext(context.Background(), o)
}

func (l heldLW) ListWithContext(ctx context.Context, o metav1.ListOptions) (runtime.Object, error) {
	select {
	case <-l.release:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return l.lw.ListWithContext(ctx, o)
}

func (l heldLW) WatchWithContext(ctx context.Context, o metav1.ListOptions) (watch.Interface, error) {
	return l.lw.WatchWithContext(ctx, o)
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch.
func (l heldLW) IsWatchListSemanticsUnSupported() bool { return true }
