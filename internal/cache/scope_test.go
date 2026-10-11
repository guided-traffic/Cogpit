package cache_test

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

func seedSecrets(src *testkit.MemorySource) {
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", map[string]string{"app": "db"}, map[string]string{"password": "hunter2"}))
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "sh.helm.release.v1.web.v1", map[string]string{"owner": "helm", "name": "web"}, map[string]string{"release": "payload"}))
}

func waitSync(t *testing.T, h *cache.Handle) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !h.WaitForSync(ctx) {
		t.Fatalf("%s did not sync: %v", h.Scope(), h.Err())
	}
}

// Each distinct scope is one informer: the full Secrets, their metadata, and the ones labelled
// owner=helm are three watches; the same selector written another way shares one.
func TestScopesAreOneInformerEach(t *testing.T) {
	src := testkit.NewMemorySource()
	seedSecrets(src)
	c := cache.New(src, cache.Options{WarmPeriod: time.Minute})
	defer c.Close()

	full := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR})
	meta := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, MetadataOnly: true})
	helm := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm,name=web"})
	again := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "name = web, owner = helm"})
	for _, h := range []*cache.Handle{full, meta, helm, again} {
		defer h.Release()
		waitSync(t, h)
	}
	if n := len(c.Stats()); n != 3 {
		t.Fatalf("%d informers, want 3: %+v", n, c.Stats())
	}
	if src.Lists() != 3 {
		t.Errorf("%d lists, want 3", src.Lists())
	}
	if helm.Scope() != again.Scope() {
		t.Errorf("selectors not normalised: %v vs %v", helm.Scope(), again.Scope())
	}
	if !c.Running(cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "name=web,owner=helm"}) {
		t.Error("Running does not normalise the selector")
	}

	if got := len(full.List()); got != 2 {
		t.Errorf("full scope holds %d Secrets, want 2", got)
	}
	if got := helm.List(); len(got) != 1 || got[0].GetName() != "sh.helm.release.v1.web.v1" {
		t.Errorf("owner=helm holds %v", names(got))
	}
	if _, has, _ := unstructured.NestedMap(helm.List()[0].Object, "data"); !has {
		t.Error("the label-selected scope holds full objects, data included")
	}
	for _, o := range meta.List() {
		if !cache.IsMetadataObject(o) {
			t.Errorf("%s: apiVersion %s kind %s, want PartialObjectMetadata", o.GetName(), o.GetAPIVersion(), o.GetKind())
		}
		if _, has := o.Object["data"]; has {
			t.Errorf("%s: the metadata-only scope holds data", o.GetName())
		}
		if o.GetUID() == "" || o.GetLabels() == nil {
			t.Errorf("%s: metadata lost: %v", o.GetName(), o.Object["metadata"])
		}
	}

	// Watch events honour the scope too.
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "sh.helm.release.v1.web.v2", map[string]string{"owner": "helm", "name": "web"}, map[string]string{"release": "p2"}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool { return len(helm.List()) == 2 && len(meta.List()) == 3 }) {
		t.Fatalf("helm %v meta %v", names(helm.List()), names(meta.List()))
	}
	if _, has := meta.List()[0].Object["data"]; has {
		t.Error("a watched object reached the metadata-only scope with data")
	}
}

// FilterListerWatcher turns label changes into adds and deletes, as the API server does.
func TestFilterListerWatcherFollowsLabelChanges(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.SecretsGVR, testkit.Secret("a", "x", map[string]string{"owner": "helm"}, nil))
	src.Set(testkit.SecretsGVR, testkit.Secret("a", "y", nil, nil))
	scope := cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm", MetadataOnly: true}
	lw := toolscache.ToListerWatcherWithContext(cache.FilterListerWatcher(scope, src.ListerWatcher(cache.Scope{GVR: testkit.SecretsGVR})))
	list, err := lw.ListWithContext(context.Background(), metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	items := list.(*unstructured.UnstructuredList).Items
	if len(items) != 1 || items[0].GetName() != "x" || !cache.IsMetadataObject(&items[0]) {
		t.Fatalf("list: %+v", items)
	}
	w, err := lw.WatchWithContext(context.Background(), metav1.ListOptions{ResourceVersion: list.(*unstructured.UnstructuredList).GetResourceVersion()})
	if err != nil {
		t.Fatal(err)
	}
	defer w.Stop()
	src.Set(testkit.SecretsGVR, testkit.Secret("a", "y", map[string]string{"owner": "helm"}, nil)) // moves in
	src.Set(testkit.SecretsGVR, testkit.Secret("a", "y", nil, nil))                                // moves out
	want := []watch.EventType{watch.Added, watch.Deleted}
	for i, typ := range want {
		select {
		case ev := <-w.ResultChan():
			u := ev.Object.(*unstructured.Unstructured)
			if ev.Type != typ || u.GetName() != "y" || !cache.IsMetadataObject(u) {
				t.Fatalf("event %d: %s %s (%s), want %s y", i, ev.Type, u.GetName(), u.GetKind(), typ)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("event %d: none", i)
		}
	}
}

// Memory serves the local kinds: informers over it see sets and deletes like watch events, and
// a watch from a resourceVersion it no longer keeps gets 410 Gone.
func TestMemorySource(t *testing.T) {
	m := cache.NewMemory()
	gvr := testkit.SecretsGVR
	m.Set(gvr, testkit.Object("v1", "Thing", "a", "one", nil))
	c := cache.New(m, cache.Options{WarmPeriod: time.Minute})
	defer c.Close()
	h := c.Acquire(cache.Scope{GVR: gvr})
	defer h.Release()
	waitSync(t, h)
	if len(h.List()) != 1 {
		t.Fatalf("list: %v", names(h.List()))
	}
	m.Set(gvr, testkit.Object("v1", "Thing", "a", "two", nil))
	m.Delete(gvr, "a", "one")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool { l := h.List(); return len(l) == 1 && l[0].GetName() == "two" }) {
		t.Fatalf("after set and delete: %v", names(h.List()))
	}
	if got, ok := m.Get(gvr, "a", "two"); !ok || got.GetResourceVersion() == "" {
		t.Errorf("Get: %v %v", got, ok)
	}

	for i := range 1100 {
		m.Set(gvr, testkit.Object("v1", "Thing", "b", "n", map[string]any{"spec": map[string]any{"i": int64(i)}}))
	}
	if _, err := toolscache.ToWatcherWithContext(m.ListerWatcher(cache.Scope{GVR: gvr})).WatchWithContext(context.Background(), metav1.ListOptions{ResourceVersion: "1"}); err == nil {
		t.Error("a watch from a compacted resourceVersion did not fail")
	}
}
