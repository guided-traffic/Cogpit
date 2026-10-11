package cache

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/metadata"
	toolscache "k8s.io/client-go/tools/cache"
)

// DynamicSource returns the Source for a real cluster: list-watchers from client-go's dynamic
// client for full objects and from its metadata client for metadata-only scopes, with WatchList
// (streaming initial lists) where the server supports it. A nil meta refuses metadata-only scopes
// rather than fetching full objects in their place.
//
// The list-watchers carry the client's WatchList capability (toolscache's
// ToListWatcherWithWatchListSemantics): with a real client, client-go 0.37's reflector opens one
// watch with sendInitialEvents=true instead of a list followed by a watch, and falls back to list
// and watch when the server refuses; with client-go's fake dynamic client, which cannot stream
// initial lists, it lists and watches.
func DynamicSource(client dynamic.Interface, meta metadata.Interface) Source {
	return dynamicSource{client: client, meta: meta}
}

type dynamicSource struct {
	client dynamic.Interface
	meta   metadata.Interface
}

// errNoMetadataClient refuses a metadata-only scope on a source without a metadata client.
var errNoMetadataClient = errors.New("no metadata client: refusing to list full objects for a metadata-only scope")

func (d dynamicSource) ListerWatcher(scope Scope) toolscache.ListerWatcher {
	if scope.MetadataOnly {
		return d.metadataLW(scope)
	}
	// Namespace("") addresses the resource across all namespaces, and is what a cluster-scoped
	// resource needs.
	res := d.client.Resource(scope.GVR).Namespace(scope.Namespace)
	return toolscache.ToListWatcherWithWatchListSemantics(&toolscache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			opts.LabelSelector = scope.LabelSelector
			return res.List(ctx, opts)
		},
		WatchFuncWithContext: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			opts.LabelSelector = scope.LabelSelector
			return res.Watch(ctx, opts)
		},
	}, d.client)
}

// metadataLW lists and watches PartialObjectMetadata and hands the informer unstructured objects
// of the shape MetadataObject describes, so that kind profiles read them like any object.
func (d dynamicSource) metadataLW(scope Scope) toolscache.ListerWatcher {
	if d.meta == nil {
		return failingLW{err: fmt.Errorf("cannot watch %s: %w", scope, errNoMetadataClient)}
	}
	res := d.meta.Resource(scope.GVR).Namespace(scope.Namespace)
	return toolscache.ToListWatcherWithWatchListSemantics(&toolscache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			opts.LabelSelector = scope.LabelSelector
			list, err := res.List(ctx, opts)
			if err != nil {
				return nil, err
			}
			return metadataList(list)
		},
		WatchFuncWithContext: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			opts.LabelSelector = scope.LabelSelector
			w, err := res.Watch(ctx, opts)
			if err != nil {
				return nil, err
			}
			return watch.Filter(w, metadataEvent), nil
		},
	}, d.meta)
}

// metadataList converts a PartialObjectMetadataList into an UnstructuredList.
func metadataList(list *metav1.PartialObjectMetadataList) (*unstructured.UnstructuredList, error) {
	out := listObject()
	out.SetResourceVersion(list.ResourceVersion)
	out.SetContinue(list.Continue)
	out.SetRemainingItemCount(list.RemainingItemCount)
	out.Items = make([]unstructured.Unstructured, 0, len(list.Items))
	for i := range list.Items {
		u, err := metadataToUnstructured(&list.Items[i])
		if err != nil {
			return nil, err
		}
		out.Items = append(out.Items, *u)
	}
	return out, nil
}

// metadataEvent converts one watch event's PartialObjectMetadata (bookmarks included, which carry
// the initial-events-end annotation the reflector looks for); a Status (an error event) passes as
// it is.
func metadataEvent(in watch.Event) (watch.Event, bool) {
	if pm, ok := in.Object.(*metav1.PartialObjectMetadata); ok {
		u, err := metadataToUnstructured(pm)
		if err != nil {
			return watch.Event{Type: watch.Error, Object: &metav1.Status{
				Status: metav1.StatusFailure, Message: "cannot convert object metadata: " + err.Error(),
			}}, true
		}
		in.Object = u
	}
	return in, true
}

func metadataToUnstructured(pm *metav1.PartialObjectMetadata) (*unstructured.Unstructured, error) {
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(pm)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{Object: m}
	u.SetAPIVersion(metadataAPIVersion)
	u.SetKind(metadataKind)
	return u, nil
}
