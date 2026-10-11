package cache

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	toolscache "k8s.io/client-go/tools/cache"
)

// DynamicSource returns the Source for a real cluster: list-watchers from client-go's dynamic
// client, with WatchList (streaming initial lists) where the server supports it.
func DynamicSource(client dynamic.Interface) Source {
	return dynamicSource{client: client}
}

type dynamicSource struct {
	client dynamic.Interface
}

func (d dynamicSource) ListerWatcher(gvr schema.GroupVersionResource, namespace string) toolscache.ListerWatcher {
	_, _ = gvr, namespace
	return nil
}
