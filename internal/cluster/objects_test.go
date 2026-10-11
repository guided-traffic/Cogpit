package cluster

import (
	"context"
	"errors"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

var configMapsGVR = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}

// objectsConn is a Conn on client-go's fake dynamic client and fake discovery: the core group
// serves configmaps and secrets, apps serves deployments, rbac serves clusterroles.
func objectsConn(t *testing.T, objs ...runtime.Object) (*Conn, *dynamicfake.FakeDynamicClient, *fakediscovery.FakeDiscovery) {
	t.Helper()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), map[schema.GroupVersionResource]string{
		configMapsGVR: "ConfigMapList",
		{Group: "apps", Version: "v1", Resource: "deployments"}: "DeploymentList",
	}, objs...)
	disco := &fakediscovery.FakeDiscovery{Fake: &k8stesting.Fake{Resources: []*metav1.APIResourceList{
		{GroupVersion: "v1", APIResources: []metav1.APIResource{
			{Name: "configmaps", Kind: "ConfigMap", Namespaced: true, Verbs: metav1.Verbs{"get", "list", "watch", "create", "update", "delete"}},
			{Name: "secrets", Kind: "Secret", Namespaced: true, Verbs: metav1.Verbs{"get", "list"}},
			{Name: "namespaces", Kind: "Namespace", Verbs: metav1.Verbs{"get", "list"}},
		}},
		{GroupVersion: "apps/v1", APIResources: []metav1.APIResource{
			{Name: "deployments", Kind: "Deployment", Namespaced: true, Verbs: metav1.Verbs{"get", "list"}},
			{Name: "deployments/scale", Kind: "Scale", Namespaced: true, Verbs: metav1.Verbs{"get", "update"}},
		}},
		{GroupVersion: "rbac.authorization.k8s.io/v1", APIResources: []metav1.APIResource{
			{Name: "clusterroles", Kind: "ClusterRole", Verbs: metav1.Verbs{"get", "list"}},
		}},
	}}}
	// client-go's object tracker ignores dryRun; a dry run must store nothing, as on a server.
	dyn.PrependReactor("*", "*", func(a k8stesting.Action) (bool, runtime.Object, error) {
		switch a := a.(type) {
		case k8stesting.CreateActionImpl:
			if len(a.CreateOptions.DryRun) > 0 {
				return true, a.GetObject(), nil
			}
		case k8stesting.UpdateActionImpl:
			if len(a.UpdateOptions.DryRun) > 0 {
				return true, a.GetObject(), nil
			}
		}
		return false, nil, nil
	})
	return &Conn{contextName: "test", namespace: "default", dyn: dyn, disco: disco, counter: NewRequestCounter()}, dyn, disco
}

func configMap(namespace, name, rv string, data map[string]any) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"namespace": namespace, "name": name, "uid": "uid-" + name},
		"data":     data,
	}}
	if rv != "" {
		u.SetResourceVersion(rv)
	}
	return u
}

func TestMapKindAndResource(t *testing.T) {
	c, _, _ := objectsConn(t)
	ctx := context.Background()
	for _, tc := range []struct {
		apiVersion, kind string
		want             schema.GroupVersionResource
		namespaced       bool
	}{
		{"v1", "ConfigMap", configMapsGVR, true},
		{"apps/v1", "Deployment", schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}, true},
		{"rbac.authorization.k8s.io/v1", "ClusterRole", schema.GroupVersionResource{Group: "rbac.authorization.k8s.io", Version: "v1", Resource: "clusterroles"}, false},
		{"v1", "Namespace", schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, false},
	} {
		m, err := c.MapKind(ctx, tc.apiVersion, tc.kind)
		if err != nil {
			t.Errorf("MapKind(%s, %s): %v", tc.apiVersion, tc.kind, err)
			continue
		}
		if m.GVR != tc.want || m.Namespaced != tc.namespaced || m.GVK.Kind != tc.kind {
			t.Errorf("MapKind(%s, %s) = %+v", tc.apiVersion, tc.kind, m)
		}
	}
	for _, bad := range [][2]string{{"v1", "Widget"}, {"example.io/v1", "Widget"}, {"", "ConfigMap"}, {"a/b/c", "X"}, {"v1", ""}} {
		if _, err := c.MapKind(ctx, bad[0], bad[1]); err == nil {
			t.Errorf("MapKind(%q, %q) succeeded", bad[0], bad[1])
		}
	}

	m, err := c.MapResource(ctx, "", "configmaps")
	if err != nil || m.GVR != configMapsGVR || !m.Namespaced || m.GVK.Kind != "ConfigMap" {
		t.Errorf("MapResource(configmaps) = %+v, %v", m, err)
	}
	m, err = c.MapResource(ctx, "rbac.authorization.k8s.io", "clusterroles")
	if err != nil || m.Namespaced || m.GVK.Kind != "ClusterRole" || m.GVR.Version != "v1" {
		t.Errorf("MapResource(clusterroles) = %+v, %v", m, err)
	}
	if _, err := c.MapResource(ctx, "", "widgets"); err == nil {
		t.Error("MapResource(widgets) succeeded")
	}
}

// A kind the cached discovery does not know is looked up once more after a fresh discovery: a
// custom resource definition created after the first mapping is found.
func TestMapKindRefreshesDiscoveryOnce(t *testing.T) {
	c, _, disco := objectsConn(t)
	ctx := context.Background()
	if _, err := c.MapKind(ctx, "v1", "ConfigMap"); err != nil {
		t.Fatal(err)
	}
	disco.Resources = append(disco.Resources, &metav1.APIResourceList{GroupVersion: "example.cogpit.io/v1", APIResources: []metav1.APIResource{
		{Name: "widgets", Kind: "Widget", Namespaced: true, Verbs: metav1.Verbs{"get", "list"}},
	}})
	m, err := c.MapKind(ctx, "example.cogpit.io/v1", "Widget")
	if err != nil || m.GVR.Resource != "widgets" || !m.Namespaced {
		t.Errorf("MapKind(Widget) after the CRD appeared = %+v, %v", m, err)
	}
}

func lastAction[T k8stesting.Action](dyn *dynamicfake.FakeDynamicClient) (T, bool) {
	var zero T
	acts := dyn.Actions()
	for i := len(acts) - 1; i >= 0; i-- {
		if a, ok := acts[i].(T); ok {
			return a, true
		}
	}
	return zero, false
}

func TestUpdateObjectSendsPreconditionAndDryRun(t *testing.T) {
	c, dyn, _ := objectsConn(t, configMap("team-a", "web", "41", map[string]any{"a": "1"}))
	ctx := context.Background()

	edited := configMap("team-a", "web", "41", map[string]any{"a": "2"})
	if _, err := c.UpdateObject(ctx, configMapsGVR, edited, true); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	upd, ok := lastAction[k8stesting.UpdateAction](dyn)
	if !ok {
		t.Fatal("no update sent")
	}
	opts := upd.(k8stesting.UpdateActionImpl).UpdateOptions
	if len(opts.DryRun) != 1 || opts.DryRun[0] != metav1.DryRunAll || opts.FieldManager != FieldManager {
		t.Errorf("dry-run options %+v", opts)
	}
	sent := upd.GetObject().(*unstructured.Unstructured)
	if sent.GetResourceVersion() != "41" || upd.GetNamespace() != "team-a" || upd.GetResource() != configMapsGVR {
		t.Errorf("sent %s %s rv %q", upd.GetResource(), upd.GetNamespace(), sent.GetResourceVersion())
	}

	if _, err := c.UpdateObject(ctx, configMapsGVR, edited, false); err != nil {
		t.Fatalf("update: %v", err)
	}
	upd, _ = lastAction[k8stesting.UpdateAction](dyn)
	if opts := upd.(k8stesting.UpdateActionImpl).UpdateOptions; len(opts.DryRun) != 0 || opts.FieldManager != FieldManager {
		t.Errorf("update options %+v", opts)
	}

	// Without the base version the update is refused before a request is made.
	n := len(dyn.Actions())
	if _, err := c.UpdateObject(ctx, configMapsGVR, configMap("team-a", "web", "", nil), false); !errors.Is(err, ErrNoPrecondition) {
		t.Errorf("update without resourceVersion: %v", err)
	}
	if len(dyn.Actions()) != n {
		t.Error("an update without precondition reached the client")
	}
}

func TestUpdateObjectConflictIsTyped(t *testing.T) {
	c, dyn, _ := objectsConn(t, configMap("team-a", "web", "57", nil))
	dyn.PrependReactor("update", "configmaps", func(a k8stesting.Action) (bool, runtime.Object, error) {
		obj := a.(k8stesting.UpdateAction).GetObject().(*unstructured.Unstructured)
		if obj.GetResourceVersion() != "57" {
			return true, nil, apierrors.NewConflict(schema.GroupResource{Resource: "configmaps"}, obj.GetName(),
				errors.New("the object has been modified; please apply your changes to the latest version and try again"))
		}
		return false, nil, nil
	})
	_, err := c.UpdateObject(context.Background(), configMapsGVR, configMap("team-a", "web", "41", map[string]any{"a": "2"}), false)
	if !apierrors.IsConflict(err) || !strings.Contains(err.Error(), "configmaps team-a/web") {
		t.Errorf("stale update: %v", err)
	}
}

func TestCreateObject(t *testing.T) {
	c, dyn, _ := objectsConn(t)
	ctx := context.Background()
	obj := configMap("", "settings", "", map[string]any{"k": "v"})
	if _, err := c.CreateObject(ctx, configMapsGVR, "team-a", obj, true); err != nil {
		t.Fatalf("dry run: %v", err)
	}
	cr, ok := lastAction[k8stesting.CreateAction](dyn)
	if !ok {
		t.Fatal("no create sent")
	}
	if opts := cr.(k8stesting.CreateActionImpl).CreateOptions; len(opts.DryRun) != 1 || opts.FieldManager != FieldManager {
		t.Errorf("dry-run create options %+v", opts)
	}
	if cr.GetNamespace() != "team-a" {
		t.Errorf("created in %q", cr.GetNamespace())
	}
	obj.SetNamespace("team-a")
	out, err := c.CreateObject(ctx, configMapsGVR, "team-a", obj, false)
	if err != nil || out.GetName() != "settings" {
		t.Fatalf("create: %v, %v", out, err)
	}
	if _, err := c.CreateObject(ctx, configMapsGVR, "team-a", obj, false); !apierrors.IsAlreadyExists(err) || !strings.Contains(err.Error(), "team-a/settings") {
		t.Errorf("second create: %v", err)
	}
}

func TestDeleteObjectWithUIDPrecondition(t *testing.T) {
	c, dyn, _ := objectsConn(t, configMap("team-a", "web", "3", nil))
	ctx := context.Background()
	if err := c.DeleteObject(ctx, configMapsGVR, "team-a", "web", "uid-web"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	del, ok := lastAction[k8stesting.DeleteAction](dyn)
	if !ok {
		t.Fatal("no delete sent")
	}
	pre := del.(k8stesting.DeleteActionImpl).DeleteOptions.Preconditions
	if pre == nil || pre.UID == nil || *pre.UID != types.UID("uid-web") || del.GetName() != "web" || del.GetNamespace() != "team-a" {
		t.Errorf("delete %s/%s preconditions %+v", del.GetNamespace(), del.GetName(), pre)
	}
	if err := c.DeleteObject(ctx, configMapsGVR, "team-a", "web", ""); !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "configmaps team-a/web") {
		t.Errorf("second delete: %v", err)
	}
}
