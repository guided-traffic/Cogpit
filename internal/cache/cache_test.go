package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

func TestSharedInformerWarmPeriodAndTransform(t *testing.T) {
	src := testkit.NewMemorySource()
	pod := testkit.Pod("default", "web-0")
	pod.Object["metadata"].(map[string]any)["managedFields"] = []any{map[string]any{"manager": "kubectl"}}
	src.Set(testkit.PodsGVR, pod)

	c := cache.New(src, cache.Options{WarmPeriod: 200 * time.Millisecond})
	defer c.Close()
	scope := cache.Scope{GVR: testkit.PodsGVR}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	h1 := c.Acquire(scope)
	h2 := c.Acquire(scope)
	if !h1.WaitForSync(ctx) {
		t.Fatal("not synced")
	}
	got, ok := h2.Get("default", "web-0")
	if !ok {
		t.Fatal("pod not in cache")
	}
	if _, found := got.Object["metadata"].(map[string]any)["managedFields"]; found {
		t.Error("managedFields not stripped")
	}
	if src.Lists() != 1 {
		t.Errorf("two handles cost %d lists, want 1", src.Lists())
	}

	events := make(chan cache.Event, 4)
	unsub := h1.Subscribe(func(ev cache.Event) { events <- ev })
	src.Set(testkit.PodsGVR, testkit.Pod("default", "web-1"))
	select {
	case ev := <-events:
		if ev.Type != cache.Added || ev.Object.GetName() != "web-1" {
			t.Errorf("event %+v", ev)
		}
	case <-ctx.Done():
		t.Fatal("no event")
	}
	unsub()

	h1.Release()
	h1.Release() // idempotent
	h2.Release()
	h3 := c.Acquire(scope) // within the warm period: no new list
	if !h3.HasSynced() || src.Lists() != 1 {
		t.Errorf("reacquire within warm period: synced=%v lists=%d", h3.HasSynced(), src.Lists())
	}
	h3.Release()
	if !testkit.WaitFor(ctx, func() bool { return len(c.Stats()) == 0 }) {
		t.Fatal("informer still running after the warm period")
	}
	h4 := c.Acquire(scope)
	defer h4.Release()
	if !h4.WaitForSync(ctx) || src.Lists() != 2 {
		t.Errorf("after expiry: lists=%d, want 2", src.Lists())
	}
}
