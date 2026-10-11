package testkit

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// RefusingSource wraps a cache.Source and refuses to list and watch some resources with an error,
// as the API server refuses a user without the permission.
type RefusingSource struct {
	Inner  cache.Source
	Refuse map[schema.GroupVersionResource]error
}

// ListerWatcher implements cache.Source.
func (r RefusingSource) ListerWatcher(scope cache.Scope) toolscache.ListerWatcher {
	if err, ok := r.Refuse[scope.GVR]; ok {
		return refusingLW{err: err}
	}
	return r.Inner.ListerWatcher(scope)
}

type refusingLW struct{ err error }

func (l refusingLW) List(metav1.ListOptions) (runtime.Object, error)   { return nil, l.err }
func (l refusingLW) Watch(metav1.ListOptions) (watch.Interface, error) { return nil, l.err }

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch.
func (l refusingLW) IsWatchListSemanticsUnSupported() bool { return true }
