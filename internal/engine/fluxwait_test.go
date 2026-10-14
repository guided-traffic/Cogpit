package engine

import (
	"context"
	"errors"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

var waitKsGVR = schema.GroupVersionResource{Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Resource: "kustomizations"}

const waitKsKey = "kustomizations.kustomize.toolkit.fluxcd.io"

// kustomization is a Kustomization with a status: the token its controller handled last, its
// conditions (type to status, the Ready one with message), its generation and the one observed.
func kustomization(name, uid, handled string, generation, observed int64, conds map[string]string, message string) *unstructured.Unstructured {
	obj := testkit.Object("kustomize.toolkit.fluxcd.io/v1", "Kustomization", "flux-system", name, map[string]any{"spec": map[string]any{}})
	obj.SetUID(types.UID(uid))
	obj.SetGeneration(generation)
	var cs []any
	for typ, st := range conds {
		cs = append(cs, map[string]any{"type": typ, "status": st, "message": message})
	}
	status := map[string]any{"conditions": cs, "observedGeneration": observed}
	if handled != "" {
		status["lastHandledReconcileAt"] = handled
	}
	obj.Object["status"] = status
	return obj
}

// fluxWaitSession connects a session whose kinds include Flux's Kustomizations.
func fluxWaitSession(t *testing.T, src *testkit.MemorySource) *Session {
	t.Helper()
	var ks *model.KindProfile
	for _, p := range kinds.Curated() {
		if p.Key == waitKsKey {
			ks = p
		}
	}
	fc := &fakeConnector{src: src, profiles: append(testProfiles(), ks)}
	s := New(Options{Connector: fc, SnapshotInterval: 5 * time.Millisecond})
	t.Cleanup(s.Close)
	connectTest(t, s)
	return s
}

// TestFluxHandledAndSettled: a handled token answers a request sent at since when it is that time
// or later, a token that is no time when it differs from the one before; a status counts once
// nothing reconciles and the generation is observed.
func TestFluxHandledAndSettled(t *testing.T) {
	since := time.Date(2026, 10, 15, 1, 0, 0, 500, time.UTC)
	for _, tc := range []struct {
		token, prev string
		want        bool
	}{
		{"", "", false},
		{"2026-10-15T01:00:00.0000005Z", "", true},
		{"2026-10-15T01:00:01Z", "2026-10-15T00:00:00Z", true},
		{"2026-10-15T00:59:59.999Z", "", false},
		{"manual-1", "manual-1", false},
		{"manual-2", "manual-1", true},
	} {
		if got := fluxHandled(tc.token, since, tc.prev); got != tc.want {
			t.Errorf("fluxHandled(%q, prev %q) = %v", tc.token, tc.prev, got)
		}
	}
	for _, tc := range []struct {
		obj  *unstructured.Unstructured
		want bool
	}{
		{kustomization("a", "u", "", 2, 2, map[string]string{"Ready": "True"}, ""), true},
		{kustomization("a", "u", "", 3, 2, map[string]string{"Ready": "False"}, ""), false},
		{kustomization("a", "u", "", 2, 2, map[string]string{"Ready": "False", "Reconciling": "True"}, ""), false},
		{kustomization("a", "u", "", 2, 2, map[string]string{"Ready": "False", "Reconciling": "False"}, ""), true},
	} {
		if got := fluxSettled(tc.obj); got != tc.want {
			t.Errorf("fluxSettled(%v) = %v", tc.obj.Object["status"], got)
		}
	}
}

// TestWaitFluxReconciled (A91): the wait follows the object through the cache: a token handled
// before the request does not end it, nor does a reconciliation still under way; the controller's
// verdict on the request does, ready or not, with the Ready message. A deleted, replaced or
// suspended object ends it with an error, and so does the time running out.
func TestWaitFluxReconciled(t *testing.T) {
	src := testkit.NewMemorySource()
	old := "2026-01-01T00:00:00Z"
	src.Set(waitKsGVR, kustomization("apps", "uid-apps", old, 1, 1, map[string]string{"Ready": "True"}, "Applied revision: main@sha1:old"))
	src.Set(waitKsGVR, kustomization("broken", "uid-broken", old, 1, 1, map[string]string{"Ready": "True"}, "fine"))
	s := fluxWaitSession(t, src)
	ref := func(name string) model.ObjectRef {
		return model.ObjectRef{Context: "test", Kind: waitKsKey, Namespace: "flux-system", Name: name}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	since := time.Now()
	type result struct {
		res FluxReconciled
		err error
	}
	wait := func(name string) (<-chan result, <-chan string) {
		out, progress := make(chan result, 1), make(chan string, 10)
		go func() {
			res, err := s.WaitFluxReconciled(ctx, ref(name), "uid-"+name, since, old, func(p string) {
				select {
				case progress <- p:
				default:
				}
			})
			out <- result{res, err}
		}()
		return out, progress
	}

	got, progress := wait("apps")
	if p := <-progress; p != "Ready: Applied revision: main@sha1:old" {
		t.Errorf("first progress %q", p)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	src.Set(waitKsGVR, kustomization("apps", "uid-apps", now, 1, 1, map[string]string{"Ready": "Unknown", "Reconciling": "True"}, "reconciliation in progress"))
	if p := <-progress; p != "Reconciling: reconciliation in progress" {
		t.Errorf("progress %q", p)
	}
	select {
	case r := <-got:
		t.Fatalf("ended while reconciling: %+v", r)
	case <-time.After(50 * time.Millisecond):
	}
	src.Set(waitKsGVR, kustomization("apps", "uid-apps", now, 1, 1, map[string]string{"Ready": "True", "Reconciling": "False"}, "Applied revision: main@sha1:new"))
	if r := <-got; r.err != nil || !r.res.Ready || r.res.Message != "Applied revision: main@sha1:new" {
		t.Errorf("ready: %+v", r)
	}

	got, _ = wait("broken")
	src.Set(waitKsGVR, kustomization("broken", "uid-broken", time.Now().UTC().Format(time.RFC3339Nano), 1, 1, map[string]string{"Ready": "False"}, "kustomize build failed"))
	if r := <-got; r.err != nil || r.res.Ready || r.res.Message != "kustomize build failed" {
		t.Errorf("not ready: %+v", r)
	}

	// Replaced, suspended, deleted: a request sent now, which no token answers yet.
	since = time.Now()
	got, _ = wait("apps")
	src.Set(waitKsGVR, kustomization("apps", "uid-other", old, 1, 1, map[string]string{"Ready": "True"}, ""))
	if r := <-got; !errors.Is(r.err, ErrFluxObjectGone) {
		t.Errorf("replaced: %+v", r)
	}
	src.Set(waitKsGVR, kustomization("broken", "uid-broken", old, 1, 1, map[string]string{"Ready": "True"}, ""))
	waitFor(t, "the reset object", func() bool {
		obj, err := s.Lookup(ref("broken"))
		return err == nil && LastHandledReconcile(obj) == old
	})
	got, _ = wait("broken")
	suspended := kustomization("broken", "uid-broken", old, 1, 1, map[string]string{"Ready": "True"}, "")
	suspended.Object["spec"] = map[string]any{"suspend": true}
	src.Set(waitKsGVR, suspended)
	if r := <-got; !errors.Is(r.err, ErrFluxSuspended) {
		t.Errorf("suspended: %+v", r)
	}
	src.Set(waitKsGVR, kustomization("broken", "uid-broken", old, 1, 1, map[string]string{"Ready": "True"}, ""))
	waitFor(t, "the resumed object", func() bool {
		obj, err := s.Lookup(ref("broken"))
		return err == nil && !kinds.IsFluxSuspended(obj)
	})
	got, _ = wait("broken")
	src.Delete(waitKsGVR, "flux-system", "broken")
	if r := <-got; !errors.Is(r.err, ErrFluxObjectGone) {
		t.Errorf("deleted: %+v", r)
	}

	// The time runs out.
	src.Set(waitKsGVR, kustomization("idle", "uid-idle", old, 1, 1, map[string]string{"Ready": "True"}, ""))
	short, stop := context.WithTimeout(ctx, 100*time.Millisecond)
	defer stop()
	waitFor(t, "the idle object", func() bool { _, err := s.Lookup(ref("idle")); return err == nil })
	if _, err := s.WaitFluxReconciled(short, ref("idle"), "uid-idle", since, old, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("no controller: %v", err)
	}
}
