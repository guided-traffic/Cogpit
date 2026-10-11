package cache

import (
	"context"
	"fmt"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"
)

// The apiVersion and kind of an object a metadata-only scope holds: what the API server answers
// for PartialObjectMetadata.
const (
	metadataAPIVersion = "meta.k8s.io/v1"
	metadataKind       = "PartialObjectMetadata"
)

// listObject returns an empty UnstructuredList of kind List.
func listObject() *unstructured.UnstructuredList {
	l := &unstructured.UnstructuredList{Object: map[string]any{}}
	l.SetAPIVersion("v1")
	l.SetKind("List")
	return l
}

// MetadataObject returns obj as a metadata-only scope holds it: apiVersion meta.k8s.io/v1, kind
// PartialObjectMetadata, and a copy of obj's metadata; no spec, status, data or any other field.
// A kind profile reads such an object through the usual metadata accessors (name, namespace,
// labels, annotations, owner references, creation and deletion timestamps).
func MetadataObject(obj *unstructured.Unstructured) *unstructured.Unstructured {
	out := &unstructured.Unstructured{Object: map[string]any{}}
	out.SetAPIVersion(metadataAPIVersion)
	out.SetKind(metadataKind)
	if md, ok := obj.Object["metadata"].(map[string]any); ok {
		out.Object["metadata"] = runtimeDeepCopyMap(md)
	}
	return out
}

// IsMetadataObject reports whether obj is a metadata-only object (MetadataObject).
func IsMetadataObject(obj *unstructured.Unstructured) bool {
	return obj != nil && obj.GetAPIVersion() == metadataAPIVersion && obj.GetKind() == metadataKind
}

// FilterScope applies the label selector and MetadataOnly of a scope to an object an in-memory
// source holds (the demo cluster, test sources): it returns nil when the object does not match the
// selector, the metadata-only copy for a metadata-only scope, and obj itself otherwise. The
// namespace is the source's to apply. A selector that does not parse matches nothing.
func FilterScope(scope Scope, obj *unstructured.Unstructured) *unstructured.Unstructured {
	if obj == nil {
		return nil
	}
	if scope.LabelSelector != "" {
		sel, err := labels.Parse(scope.LabelSelector)
		if err != nil || !sel.Matches(labels.Set(obj.GetLabels())) {
			return nil
		}
	}
	if scope.MetadataOnly {
		return MetadataObject(obj)
	}
	return obj
}

// runtimeDeepCopyMap deep-copies JSON-shaped content; values of other types are kept as they are
// (unstructured's DeepCopyJSONValue panics on them).
func runtimeDeepCopyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return runtimeDeepCopyMap(x)
	case []any:
		out := make([]any, len(x))
		for i, e := range x {
			out[i] = deepCopyValue(e)
		}
		return out
	}
	return v
}

// FilterListerWatcher applies a scope's label selector and MetadataOnly to a list-watcher over
// unfiltered objects, for in-memory sources with a list and watch path of their own (the demo
// cluster). Like the API server, its watch turns a Modified that moves an object out of the
// selection into Deleted and one that moves it in into Added: it remembers what its last list and
// the watches since handed out. A plain scope returns lw unchanged.
func FilterListerWatcher(scope Scope, lw toolscache.ListerWatcher) toolscache.ListerWatcher {
	if scope.LabelSelector == "" && !scope.MetadataOnly {
		return lw
	}
	return &filterLW{scope: scope, inner: lw, in: map[string]bool{}}
}

// filterLW serves one informer: its reflector lists, then watches, one call at a time.
type filterLW struct {
	scope Scope
	inner toolscache.ListerWatcher

	mu sync.Mutex
	in map[string]bool // namespace/name of the objects the consumer holds
}

// IsWatchListSemanticsUnSupported tells client-go's reflector to list, then watch: the filter
// does not rewrite streamed initial lists.
func (f *filterLW) IsWatchListSemanticsUnSupported() bool { return true }

func filterKey(u *unstructured.Unstructured) string { return u.GetNamespace() + "/" + u.GetName() }

func (f *filterLW) List(opts metav1.ListOptions) (runtime.Object, error) {
	return f.ListWithContext(context.Background(), opts)
}

func (f *filterLW) Watch(opts metav1.ListOptions) (watch.Interface, error) {
	return f.WatchWithContext(context.Background(), opts)
}

func (f *filterLW) ListWithContext(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
	obj, err := toolscache.ToListerWithContext(f.inner).ListWithContext(ctx, opts)
	if err != nil {
		return nil, err
	}
	list, ok := obj.(*unstructured.UnstructuredList)
	if !ok {
		return nil, fmt.Errorf("cannot filter a list of type %T", obj)
	}
	out := list.DeepCopy()
	out.Items = out.Items[:0]
	in := map[string]bool{}
	for i := range list.Items {
		if o := FilterScope(f.scope, &list.Items[i]); o != nil {
			out.Items = append(out.Items, *o.DeepCopy())
			in[filterKey(o)] = true
		}
	}
	f.mu.Lock()
	f.in = in
	f.mu.Unlock()
	return out, nil
}

func (f *filterLW) WatchWithContext(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	w, err := toolscache.ToWatcherWithContext(f.inner).WatchWithContext(ctx, opts)
	if err != nil {
		return nil, err
	}
	return watch.Filter(w, f.filterEvent), nil
}

func (f *filterLW) filterEvent(ev watch.Event) (watch.Event, bool) {
	u, ok := ev.Object.(*unstructured.Unstructured)
	if !ok || ev.Type == watch.Bookmark || ev.Type == watch.Error {
		return ev, true // errors and bookmarks pass
	}
	key := filterKey(u)
	match := FilterScope(f.scope, u)
	f.mu.Lock()
	defer f.mu.Unlock()
	held := f.in[key]
	shaped := func() *unstructured.Unstructured {
		if f.scope.MetadataOnly {
			return MetadataObject(u)
		}
		return u
	}
	switch ev.Type {
	case watch.Deleted:
		if !held {
			return ev, false
		}
		delete(f.in, key)
		return watch.Event{Type: watch.Deleted, Object: shaped()}, true
	case watch.Added, watch.Modified:
		if match == nil {
			if !held {
				return ev, false
			}
			delete(f.in, key)
			return watch.Event{Type: watch.Deleted, Object: shaped()}, true
		}
		typ := watch.Modified
		if !held {
			typ = watch.Added
		}
		f.in[key] = true
		return watch.Event{Type: typ, Object: match}, true
	}
	return ev, true
}
