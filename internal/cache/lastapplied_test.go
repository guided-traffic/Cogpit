package cache_test

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

var configMapsGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

// applied returns obj with the annotation kubectl apply writes: the applied manifest, a Secret's
// values included.
func applied(obj *unstructured.Unstructured, manifest string) *unstructured.Unstructured {
	obj.SetAnnotations(map[string]string{cache.LastAppliedAnnotation: manifest, "team": "db"})
	return obj
}

// A Secret applied with kubectl carries its values in the last-applied annotation: the cache drops
// that annotation from Secrets in every scope (metadata only, label-selected, full), keeps the
// other annotations, and leaves other kinds' last-applied annotation alone.
func TestSecretsLoseTheLastAppliedAnnotation(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.SecretsGVR, applied(testkit.Secret("apps", "db", nil, map[string]string{"password": "hunter2"}),
		`{"apiVersion":"v1","data":{"password":"aHVudGVyMg=="},"kind":"Secret"}`))
	src.Set(testkit.SecretsGVR, applied(testkit.Secret("apps", "sh.helm.release.v1.web.v1", map[string]string{"owner": "helm"}, map[string]string{"release": "x"}),
		`{"stringData":{"release":"x"}}`))
	src.Set(configMapsGVR, applied(testkit.Object("v1", "ConfigMap", "apps", "settings", nil), `{"data":{"mode":"fast"}}`))
	c := cache.New(src, cache.Options{WarmPeriod: time.Minute})
	defer c.Close()

	for _, scope := range []cache.Scope{
		{GVR: testkit.SecretsGVR, MetadataOnly: true},
		{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm"},
		{GVR: testkit.SecretsGVR},
	} {
		h := c.Acquire(scope)
		waitSync(t, h)
		if len(h.List()) == 0 {
			t.Fatalf("%s: empty", scope)
		}
		for _, o := range h.List() {
			ann := o.GetAnnotations()
			if _, has := ann[cache.LastAppliedAnnotation]; has {
				t.Errorf("%s: %s kept the last-applied annotation", scope, o.GetName())
			}
			if ann["team"] != "db" {
				t.Errorf("%s: %s lost its other annotations: %v", scope, o.GetName(), ann)
			}
		}
		h.Release()
	}

	// A watch event is transformed like the initial list.
	meta := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, MetadataOnly: true})
	defer meta.Release()
	src.Set(testkit.SecretsGVR, applied(testkit.Secret("apps", "tls", nil, map[string]string{"tls.key": "k"}), `{"data":{"tls.key":"aw=="}}`))
	waitFor(t, func() bool { _, ok := meta.Get("apps", "tls"); return ok })
	if o, _ := meta.Get("apps", "tls"); o.GetAnnotations()[cache.LastAppliedAnnotation] != "" {
		t.Error("a watched Secret kept the last-applied annotation")
	}

	cm := c.Acquire(cache.Scope{GVR: configMapsGVR})
	defer cm.Release()
	waitSync(t, cm)
	if o, _ := cm.Get("apps", "settings"); o == nil || o.GetAnnotations()[cache.LastAppliedAnnotation] == "" {
		t.Error("a Config Map lost its last-applied annotation; only Secrets drop it")
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("timed out")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
