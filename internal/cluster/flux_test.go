package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

// TestFluxPatch: the bodies of Reconcile (forced too), Suspend and Resume, as the flux CLI sends
// them, with Cogpit's UID precondition.
func TestFluxPatch(t *testing.T) {
	at := time.Date(2026, 10, 13, 22, 1, 2, 345000000, time.UTC)
	yes, no := true, false
	for name, tc := range map[string]struct {
		uid  string
		req  FluxRequest
		want string
	}{
		"reconcile":      {"uid-1", FluxRequest{Reconcile: true}, `{"metadata":{"annotations":{"reconcile.fluxcd.io/requestedAt":"2026-10-13T22:01:02.345Z"},"uid":"uid-1"}}`},
		"force":          {"uid-1", FluxRequest{Reconcile: true, Force: true}, `{"metadata":{"annotations":{"reconcile.fluxcd.io/forceAt":"2026-10-13T22:01:02.345Z","reconcile.fluxcd.io/requestedAt":"2026-10-13T22:01:02.345Z"},"uid":"uid-1"}}`},
		"suspend":        {"uid-1", FluxRequest{Suspend: &yes}, `{"metadata":{"uid":"uid-1"},"spec":{"suspend":true}}`},
		"resume":         {"uid-1", FluxRequest{Reconcile: true, Suspend: &no}, `{"metadata":{"annotations":{"reconcile.fluxcd.io/requestedAt":"2026-10-13T22:01:02.345Z"},"uid":"uid-1"},"spec":{"suspend":false}}`},
		"without a uid":  {"", FluxRequest{Suspend: &yes}, `{"spec":{"suspend":true}}`},
		"reconcile only": {"", FluxRequest{Reconcile: true}, `{"metadata":{"annotations":{"reconcile.fluxcd.io/requestedAt":"2026-10-13T22:01:02.345Z"}}}`},
	} {
		got, err := FluxPatch(at, tc.uid, tc.req)
		if err != nil || string(got) != tc.want {
			t.Errorf("%s: %s, %v\nwant %s", name, got, err, tc.want)
		}
	}
	if _, err := FluxPatch(at, "uid-1", FluxRequest{}); err == nil {
		t.Error("an empty request patched")
	}
}

// TestPatchFluxSendsAMergePatch: one JSON merge patch on the object through the dynamic client, as
// Cogpit's field manager; the API server's refusal of the UID becomes ErrReplaced.
func TestPatchFluxSendsAMergePatch(t *testing.T) {
	fixed := time.Date(2026, 10, 13, 22, 0, 0, 0, time.UTC)
	now = func() time.Time { return fixed }
	defer func() { now = time.Now }()
	gvr := schema.GroupVersionResource{Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Resource: "kustomizations"}
	ks := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "kustomize.toolkit.fluxcd.io/v1", "kind": "Kustomization",
		"metadata": map[string]any{"namespace": "flux-system", "name": "apps", "uid": "uid-apps"},
		"spec":     map[string]any{"path": "./apps"},
	}}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{gvr: "KustomizationList"}, ks)
	c := &Conn{contextName: "test", dyn: dyn, counter: NewRequestCounter()}
	ctx := context.Background()
	yes := true
	if err := c.PatchFlux(ctx, gvr, "flux-system", "apps", "uid-apps", FluxRequest{Suspend: &yes}); err != nil {
		t.Fatal(err)
	}
	p, ok := lastAction[k8stesting.PatchActionImpl](dyn)
	if !ok || p.GetPatchType() != types.MergePatchType || p.GetName() != "apps" || p.GetNamespace() != "flux-system" ||
		string(p.GetPatch()) != `{"metadata":{"uid":"uid-apps"},"spec":{"suspend":true}}` {
		t.Fatalf("patch %+v %s", p, p.GetPatch())
	}
	got, err := dyn.Resource(gvr).Namespace("flux-system").Get(ctx, "apps", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if v, _, _ := unstructured.NestedBool(got.Object, "spec", "suspend"); !v {
		t.Errorf("spec.suspend not set: %v", got.Object["spec"])
	}

	invalid := apierrors.NewInvalid(schema.GroupKind{Group: gvr.Group, Kind: "Kustomization"}, "apps", field.ErrorList{
		field.Invalid(field.NewPath("metadata", "uid"), "uid-apps", "field is immutable"),
	})
	dyn.PrependReactor("patch", "kustomizations", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, invalid })
	if err := c.PatchFlux(ctx, gvr, "flux-system", "apps", "uid-apps", FluxRequest{Reconcile: true}); !errors.Is(err, ErrReplaced) {
		t.Errorf("a replaced object: %v", err)
	}
	if err := c.PatchFlux(ctx, gvr, "flux-system", "apps", "uid-apps", FluxRequest{}); err == nil {
		t.Error("an empty request was sent")
	}
}
