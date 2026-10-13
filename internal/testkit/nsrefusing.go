package testkit

import (
	"errors"
	"slices"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// NamespacedOnlySource wraps a MemorySource as the API server answers a user whose Roles grant
// some resources in some namespaces only: it refuses, with 403 Forbidden, to list or watch the
// resources in GVRs in all namespaces, and in the namespaces Denied names; every other scope is
// the inner source's. It counts the list and watch calls per scope ("list pods", "watch pods in
// team-a"), refused ones included.
type NamespacedOnlySource struct {
	Inner  *MemorySource
	GVRs   []schema.GroupVersionResource
	Denied []string

	mu    sync.Mutex
	calls map[string]int
}

// ListerWatcher implements cache.Source.
func (s *NamespacedOnlySource) ListerWatcher(scope cache.Scope) toolscache.ListerWatcher {
	inner := s.Inner.ListerWatcher(scope)
	if slices.Contains(s.GVRs, scope.GVR) && (scope.Namespace == "" || slices.Contains(s.Denied, scope.Namespace)) {
		inner = refusingLW{err: apierrors.NewForbidden(schema.GroupResource{Group: scope.GVR.Group, Resource: scope.GVR.Resource},
			"", errors.New("the user may not list it in "+where(scope.Namespace)))}
	}
	return &countedLW{s: s, scope: scope.String(), inner: inner}
}

func where(namespace string) string {
	if namespace == "" {
		return "all namespaces"
	}
	return "namespace " + namespace
}

// Calls returns the list and watch calls so far, by "verb scope".
func (s *NamespacedOnlySource) Calls() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.calls))
	for k, v := range s.calls {
		out[k] = v
	}
	return out
}

func (s *NamespacedOnlySource) count(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.calls == nil {
		s.calls = map[string]int{}
	}
	s.calls[key]++
}

type countedLW struct {
	s     *NamespacedOnlySource
	scope string
	inner toolscache.ListerWatcher
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch.
func (lw *countedLW) IsWatchListSemanticsUnSupported() bool { return true }

func (lw *countedLW) List(opts metav1.ListOptions) (runtime.Object, error) {
	lw.s.count("list " + lw.scope)
	return lw.inner.List(opts)
}

func (lw *countedLW) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	lw.s.count("watch " + lw.scope)
	return lw.inner.Watch(opts)
}
