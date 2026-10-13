package cache_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	toolscache "k8s.io/client-go/tools/cache"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// forbidSource refuses every list and watch of one resource, as an API server refuses a user
// without the verb.
type forbidSource struct {
	*testkit.MemorySource
	forbid string
}

func (f *forbidSource) ListerWatcher(scope cache.Scope) toolscache.ListerWatcher {
	if scope.GVR.Resource == f.forbid {
		return forbiddenLW{}
	}
	return f.MemorySource.ListerWatcher(scope)
}

type forbiddenLW struct{}

var errForbidden = errors.New(`configmaps is forbidden: User "dev" cannot list resource "configmaps" (test)`)

func (forbiddenLW) IsWatchListSemanticsUnSupported() bool             { return true }
func (forbiddenLW) List(metav1.ListOptions) (runtime.Object, error)   { return nil, errForbidden }
func (forbiddenLW) Watch(metav1.ListOptions) (watch.Interface, error) { return nil, errForbidden }

// TestJoinReadsSeveralScopesAsOne: a joined handle lists, gets, syncs and delivers the events of
// both its parts (Helm's records in Secrets and in ConfigMaps), keeps each part's own label
// selection, and releases both parts; a part the user may not list leaves the other one showing,
// with its error.
func TestJoinReadsSeveralScopesAsOne(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	src := testkit.NewMemorySource()
	owner := map[string]string{"owner": "helm"}
	record := func(gvr schema.GroupVersionResource, kind, ns, name string, labels map[string]string) {
		o := testkit.Object("v1", kind, ns, name, nil)
		o.SetUID(o.GetUID() + types.UID("-"+kind))
		o.SetLabels(labels)
		src.Set(gvr, o)
	}
	record(testkit.SecretsGVR, "Secret", "apps", "sh.helm.release.v1.web.v1", owner)
	record(configMapsGVR, "ConfigMap", "apps", "sh.helm.release.v1.web.v1", owner) // the same name in the other part
	record(configMapsGVR, "ConfigMap", "jobs", "sh.helm.release.v1.cron.v1", owner)
	record(configMapsGVR, "ConfigMap", "apps", "settings", nil) // not selected
	c := cache.New(src, cache.Options{WarmPeriod: 20 * time.Millisecond})
	defer c.Close()

	secrets := cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm"}
	configMaps := cache.Scope{GVR: configMapsGVR, LabelSelector: "owner=helm"}
	h := cache.Join(c.Acquire(secrets), c.Acquire(configMaps))
	if h.Scope() != secrets || len(h.Parts()) != 2 {
		t.Fatalf("scope %v, %d parts", h.Scope(), len(h.Parts()))
	}
	var events atomic.Int64
	var inside sync.Mutex // the joined subscription delivers one event at a time
	unsub := h.Subscribe(func(cache.Event) {
		if !inside.TryLock() {
			t.Error("two events delivered at once")
			return
		}
		defer inside.Unlock()
		events.Add(1)
	})
	defer unsub()
	if !h.WaitForSync(ctx) {
		t.Fatal("the joined handle did not sync")
	}
	if n := len(h.List()); n != 3 {
		t.Errorf("List: %d objects, want 3", n)
	}
	if n := len(h.ListNamespace("apps")); n != 2 {
		t.Errorf("ListNamespace(apps): %d objects, want 2", n)
	}
	if o, ok := h.Get("apps", "sh.helm.release.v1.web.v1"); !ok || o.GetKind() != "Secret" {
		t.Errorf("Get returns the first part's object: %v %v", ok, o)
	}
	if o, ok := h.Parts()[1].Get("apps", "sh.helm.release.v1.web.v1"); !ok || o.GetKind() != "ConfigMap" {
		t.Errorf("the second part's object: %v %v", ok, o)
	}
	if _, ok := h.Get("jobs", "sh.helm.release.v1.cron.v1"); !ok {
		t.Error("Get misses the second part's object")
	}
	before := events.Load()
	record(configMapsGVR, "ConfigMap", "apps", "sh.helm.release.v1.web.v2", owner)
	record(testkit.SecretsGVR, "Secret", "apps", "sh.helm.release.v1.api.v1", owner)
	waitFor(t, func() bool { return events.Load() >= before+2 && len(h.List()) == 5 })
	if !c.Running(secrets) || !c.Running(configMaps) {
		t.Error("a part is not running")
	}

	h.Release()
	h.Release() // idempotent
	waitFor(t, func() bool { return !c.Running(secrets) && !c.Running(configMaps) })

	// ConfigMaps forbidden: the Secrets part shows, and the joined handle says why the other is missing.
	fs := &forbidSource{MemorySource: src, forbid: "configmaps"}
	c2 := cache.New(fs, cache.Options{})
	defer c2.Close()
	h2 := cache.Join(c2.Acquire(secrets), c2.Acquire(configMaps))
	defer h2.Release()
	if !h2.WaitForSync(ctx) {
		t.Fatalf("a forbidden part keeps the joined handle from syncing: %v", h2.Err())
	}
	if n := len(h2.List()); n != 2 || h2.Err() == nil {
		t.Errorf("with ConfigMaps forbidden: %d objects, err %v", n, h2.Err())
	}

	// Every part forbidden: never synced, the error shown.
	c3 := cache.New(&forbidSource{MemorySource: src, forbid: "configmaps"}, cache.Options{})
	defer c3.Close()
	h3 := cache.Join(c3.Acquire(configMaps), c3.Acquire(configMaps))
	defer h3.Release()
	waitFor(t, func() bool { return h3.Err() != nil })
	if h3.HasSynced() {
		t.Error("synced although no part listed")
	}

	if one := c.Acquire(secrets); cache.Join(one) != one {
		t.Error("Join of one handle is not that handle")
	} else {
		one.Release()
	}
	if (&cache.Handle{}).Parts() != nil {
		t.Error("a plain handle has parts")
	}
}
