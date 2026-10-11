package cache

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	toolscache "k8s.io/client-go/tools/cache"
)

// DynamicSource returns the Source for a real cluster: list-watchers from client-go's dynamic
// client, with WatchList (streaming initial lists) where the server supports it.
//
// The list-watchers carry the client's WatchList capability (toolscache's
// ToListWatcherWithWatchListSemantics): with a real client, client-go 0.37's reflector opens one
// watch with sendInitialEvents=true instead of a list followed by a watch, and falls back to list
// and watch when the server refuses; with client-go's fake dynamic client, which cannot stream
// initial lists, it lists and watches.
func DynamicSource(client dynamic.Interface) Source {
	return dynamicSource{client: client}
}

type dynamicSource struct {
	client dynamic.Interface
}

func (d dynamicSource) ListerWatcher(gvr schema.GroupVersionResource, namespace string) toolscache.ListerWatcher {
	// Namespace("") addresses the resource across all namespaces, and is what a cluster-scoped
	// resource needs.
	res := d.client.Resource(gvr).Namespace(namespace)
	return toolscache.ToListWatcherWithWatchListSemantics(&toolscache.ListWatch{
		ListWithContextFunc: func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
			return res.List(ctx, opts)
		},
		WatchFuncWithContext: func(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
			return res.Watch(ctx, opts)
		},
	}, d.client)
}
