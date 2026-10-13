package testkit

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"sync"

	authorizationv1 "k8s.io/api/authorization/v1"
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

// refuseRead counts a resource read by namespace (RequestsIn) and, with FakeAccess.Enforce,
// answers 403 Forbidden to one the rules do not allow, as RBAC does: a watch with
// sendInitialEvents needs list and watch. It reports whether it answered.
func (s *FakeAPIServer) refuseRead(w http.ResponseWriter, r *http.Request, req fakeRequest, resource string, watching, initial bool) bool {
	var verbs []string
	verb := "list"
	switch {
	case r.Method != http.MethodGet:
		return false
	case watching && initial:
		verb, verbs = "watchlist", []string{"list", "watch"}
	case watching:
		verb, verbs = "watch", []string{"watch"}
	case req.name != "":
		verb, verbs = "get", []string{"get"}
	default:
		verbs = []string{"list"}
	}
	key := verb + " " + resource
	if req.namespace != "" {
		key += " in " + req.namespace
	}
	s.mu.Lock()
	if s.scoped == nil {
		s.scoped = map[string]int64{}
	}
	s.scoped[key]++
	s.mu.Unlock()
	a := s.opts.Access
	if a == nil || !a.Enforce {
		return false
	}
	for _, v := range verbs {
		attrs := &authorizationv1.ResourceAttributes{Namespace: req.namespace, Verb: v, Group: req.res.GVR.Group, Resource: req.res.GVR.Resource}
		if !a.allows(attrs) {
			s.count(verb, resource)
			scope := "at the cluster scope"
			if req.namespace != "" {
				scope = "in the namespace \"" + req.namespace + "\""
			}
			writeStatus(w, http.StatusForbidden, "Forbidden", fmt.Sprintf("%s is forbidden: User \"fake\" cannot %s resource %q in API group %q %s",
				req.res.GVR.Resource, v, req.res.GVR.Resource, req.res.GVR.Group, scope))
			return true
		}
	}
	return false
}

// RequestsIn returns the resource reads the server was sent, refused ones included, by "verb
// resource in namespace" ("watchlist pods in team-a"), or "verb resource" for a read in all
// namespaces or of a cluster-scoped resource.
func (s *FakeAPIServer) RequestsIn() map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int64, len(s.scoped))
	for k, v := range s.scoped {
		out[k] = v
	}
	return out
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
