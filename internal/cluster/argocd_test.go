package cluster

import (
	"context"
	"errors"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation/field"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

// argoConn is a Conn on client-go's fake dynamic client serving Argo CD's Applications; every
// patch is recorded and answered by reply (nil: the tracker applies it).
func argoConn(t *testing.T, reply func(k8stesting.PatchActionImpl) error) (*Conn, *[]k8stesting.PatchActionImpl) {
	t.Helper()
	gvr := argoApplications("v1alpha1")
	app := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "argoproj.io/v1alpha1", "kind": "Application",
		"metadata": map[string]any{"name": "shop", "namespace": "argocd", "uid": "uid-1", "resourceVersion": "7"},
	}}
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gvr: "ApplicationList"}, app)
	var patches []k8stesting.PatchActionImpl
	dyn.PrependReactor("patch", "applications", func(a k8stesting.Action) (bool, runtime.Object, error) {
		p := a.(k8stesting.PatchActionImpl)
		patches = append(patches, p)
		if reply != nil {
			if err := reply(p); err != nil {
				return true, nil, err
			}
		}
		return true, app, nil
	})
	return &Conn{contextName: "test", namespace: "default", dyn: dyn, counter: NewRequestCounter()}, &patches
}

// TestArgoRefreshPatches: a merge patch of the refresh annotation, normal or hard, bound to the
// UID, as Cogpit's field manager.
func TestArgoRefreshPatches(t *testing.T) {
	c, patches := argoConn(t, nil)
	if err := c.RefreshArgoApplication(context.Background(), "v1alpha1", "argocd", "shop", "uid-1", false); err != nil {
		t.Fatal(err)
	}
	if err := c.RefreshArgoApplication(context.Background(), "v1alpha1", "argocd", "shop", "uid-1", true); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"normal"},"uid":"uid-1"}}`,
		`{"metadata":{"annotations":{"argocd.argoproj.io/refresh":"hard"},"uid":"uid-1"}}`,
	}
	if len(*patches) != 2 {
		t.Fatalf("%d patches", len(*patches))
	}
	for i, p := range *patches {
		if p.GetPatchType() != types.MergePatchType || string(p.GetPatch()) != want[i] || p.GetNamespace() != "argocd" || p.GetName() != "shop" ||
			p.GetResource().Group != "argoproj.io" || p.PatchOptions.FieldManager != FieldManager {
			t.Errorf("patch %d: %s %s %+v", i, p.GetPatchType(), p.GetPatch(), p.PatchOptions)
		}
	}
}

// TestArgoSyncPatch: the operation with the UID and resourceVersion preconditions; both are
// required.
func TestArgoSyncPatch(t *testing.T) {
	c, patches := argoConn(t, nil)
	op := map[string]any{"initiatedBy": map[string]any{"username": "cogpit"}, "sync": map[string]any{"revision": "main", "prune": true}}
	if err := c.SyncArgoApplication(context.Background(), "v1alpha1", "argocd", "shop", "uid-1", "7", op); err != nil {
		t.Fatal(err)
	}
	want := `{"metadata":{"resourceVersion":"7","uid":"uid-1"},"operation":{"initiatedBy":{"username":"cogpit"},"sync":{"prune":true,"revision":"main"}}}`
	if len(*patches) != 1 || string((*patches)[0].GetPatch()) != want {
		t.Fatalf("patches %v", *patches)
	}
	for _, pre := range [][2]string{{"", "7"}, {"uid-1", ""}} {
		if err := c.SyncArgoApplication(context.Background(), "v1alpha1", "argocd", "shop", pre[0], pre[1], op); !errors.Is(err, ErrNoPrecondition) {
			t.Errorf("without %q/%q: %v", pre[0], pre[1], err)
		}
	}
	if len(*patches) != 1 {
		t.Errorf("a sync without preconditions was sent")
	}
}

// TestArgoSyncConflicts: a newer version is ErrArgoChanged, another object under the name
// ErrReplaced (metadata.uid is immutable: 422 Invalid).
func TestArgoSyncConflicts(t *testing.T) {
	gr := schema.GroupResource{Group: "argoproj.io", Resource: "applications"}
	op := map[string]any{"sync": map[string]any{}}
	changed, _ := argoConn(t, func(k8stesting.PatchActionImpl) error {
		return apierrors.NewConflict(gr, "shop", errors.New("the object has been modified; please apply your changes to the latest version and try again"))
	})
	if err := changed.SyncArgoApplication(context.Background(), "v1alpha1", "argocd", "shop", "uid-1", "7", op); !errors.Is(err, ErrArgoChanged) {
		t.Errorf("conflict: %v", err)
	}
	invalid := apierrors.NewInvalid(schema.GroupKind{Group: "argoproj.io", Kind: "Application"}, "shop",
		field.ErrorList{field.Invalid(field.NewPath("metadata", "uid"), "uid-1", "field is immutable")})
	replacedConn, _ := argoConn(t, func(k8stesting.PatchActionImpl) error { return invalid })
	if err := replacedConn.SyncArgoApplication(context.Background(), "v1alpha1", "argocd", "shop", "uid-1", "7", op); !errors.Is(err, ErrReplaced) {
		t.Errorf("another object: %v", err)
	}
	if err := replacedConn.RefreshArgoApplication(context.Background(), "v1alpha1", "argocd", "shop", "uid-1", false); !errors.Is(err, ErrReplaced) {
		t.Errorf("refresh of another object: %v", err)
	}
}
