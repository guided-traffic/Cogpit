package cache_test

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestTrimmedScope: a scope that names a registered trim holds its objects trimmed, the kind's own
// scope beside it holds them whole (two informers), a change reaches the trimmed scope trimmed, a
// scope naming no registered trim holds whole objects, and the trim's name shows in the scope's.
func TestTrimmedScope(t *testing.T) {
	const name = "test-drop-data"
	cache.RegisterTrim(name, func(u *unstructured.Unstructured) { delete(u.Object, "data") })
	defer cache.RegisterTrim(name, nil)

	src := testkit.NewMemorySource()
	cm := testkit.Object("v1", "ConfigMap", "default", "big", map[string]any{"data": map[string]any{"k": "v"}})
	src.Set(configMapsGVR, cm)
	c := cache.New(src, cache.Options{WarmPeriod: time.Minute})
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	trimmed := c.Acquire(cache.Scope{GVR: configMapsGVR, Trim: name})
	defer trimmed.Release()
	whole := c.Acquire(cache.Scope{GVR: configMapsGVR})
	defer whole.Release()
	unknown := c.Acquire(cache.Scope{GVR: configMapsGVR, Trim: "no-such-trim"})
	defer unknown.Release()
	for _, h := range []*cache.Handle{trimmed, whole, unknown} {
		if !h.WaitForSync(ctx) {
			t.Fatalf("%s not synced", h.Scope())
		}
	}
	hasData := func(h *cache.Handle, obj string) bool {
		o, ok := h.Get("default", obj)
		if !ok {
			t.Fatalf("%s: %s not cached", h.Scope(), obj)
		}
		_, has := o.Object["data"]
		return has
	}
	if hasData(trimmed, "big") {
		t.Error("the trimmed scope kept the data")
	}
	if !hasData(whole, "big") || !hasData(unknown, "big") {
		t.Error("the whole scope, or one naming no registered trim, lost the data")
	}
	if got := len(cache.Runners(c)); got != 3 {
		t.Errorf("%d informers, want 3 (the trimmed scope is one of its own): %v", got, cache.Runners(c))
	}

	src.Set(configMapsGVR, testkit.Object("v1", "ConfigMap", "default", "next", map[string]any{"data": map[string]any{"a": "b"}}))
	if !testkit.WaitFor(ctx, func() bool { _, ok := trimmed.Get("default", "next"); return ok }) {
		t.Fatal("the change did not reach the trimmed scope")
	}
	if hasData(trimmed, "next") {
		t.Error("a changed object entered the trimmed scope whole")
	}
	if got := (cache.Scope{GVR: configMapsGVR, Trim: name}).String(); got != "configmaps ("+name+")" {
		t.Errorf("scope name %q", got)
	}
}
