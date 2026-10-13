package testkit

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// SlowLists wraps a source so that every list of one resource in all namespaces answers after
// Delay, as a slow API server's would: the answers of the other scopes come first.
type SlowLists struct {
	Inner cache.Source
	GVR   schema.GroupVersionResource
	Delay time.Duration
}

// ListerWatcher implements cache.Source.
func (s SlowLists) ListerWatcher(scope cache.Scope) toolscache.ListerWatcher {
	lw := s.Inner.ListerWatcher(scope)
	if scope.GVR != s.GVR || scope.Namespace != "" {
		return lw
	}
	return slowLW{inner: lw, delay: s.Delay}
}

type slowLW struct {
	inner toolscache.ListerWatcher
	delay time.Duration
}

func (l slowLW) List(opts metav1.ListOptions) (runtime.Object, error) {
	time.Sleep(l.delay)
	return toolscache.ToListerWithContext(l.inner).ListWithContext(context.Background(), opts)
}

func (l slowLW) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return toolscache.ToWatcherWithContext(l.inner).WatchWithContext(context.Background(), opts)
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch.
func (slowLW) IsWatchListSemanticsUnSupported() bool { return true }
