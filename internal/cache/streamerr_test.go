package cache_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// OnStreamError hears of a list that broke, with its scope and error, also when the hook panics;
// a refusal (403) is OnRefused's, not a broken stream.
func TestOnStreamErrorHearsABrokenList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	lost := errors.New("http2: client connection lost")
	src := testkit.RefusingSource{Inner: testkit.NewMemorySource(), Refuse: map[schema.GroupVersionResource]error{
		testkit.PodsGVR:     lost,
		testkit.ServicesGVR: apierrors.NewForbidden(schema.GroupResource{Resource: "services"}, "", errors.New("no")),
	}}
	c := cache.New(src, cache.Options{})
	defer c.Close()
	var mu sync.Mutex
	heard := map[cache.Scope]error{}
	c.OnStreamError(func(sc cache.Scope, err error) {
		mu.Lock()
		heard[sc] = err
		mu.Unlock()
		panic("a hook bug")
	})
	pods := c.Acquire(cache.Scope{GVR: testkit.PodsGVR})
	defer pods.Release()
	svcs := c.Acquire(cache.Scope{GVR: testkit.ServicesGVR})
	defer svcs.Release()
	if !testkit.WaitFor(ctx, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return heard[cache.Scope{GVR: testkit.PodsGVR}] != nil
	}) {
		t.Fatal("the broken list was not reported")
	}
	if !testkit.WaitFor(ctx, func() bool { return cache.IsRefused(svcs.Err()) }) {
		t.Fatalf("services: %v", svcs.Err())
	}
	mu.Lock()
	defer mu.Unlock()
	if err := heard[cache.Scope{GVR: testkit.PodsGVR}]; !errors.Is(err, lost) {
		t.Errorf("reported %v, want %v", err, lost)
	}
	if _, ok := heard[cache.Scope{GVR: testkit.ServicesGVR}]; ok {
		t.Error("a refusal was reported as a broken stream")
	}
}
