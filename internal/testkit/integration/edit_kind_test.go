//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// editNamespace is the namespace the editing test owns on the shared kind cluster.
const editNamespace = "cogpit-edit"

var (
	kindConfigMaps = schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}
	kindNamespaces = schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
)

// TestKindEditCreateConflictDelete walks G9 through the engine against the kind cluster, in its
// own namespace: object.create makes the namespace and a ConfigMap (two documents, in order);
// another client changes the ConfigMap with its own field manager; object.edit opens it from
// the cache, object.apply writes an edit (dry run, then update with the base resourceVersion);
// another client changes it again and a stale apply is refused with a conflict, nothing
// overwritten; object.delete deletes it with its UID. It also dry-runs every Cogpit template
// against the API server. The namespace is deleted at the end.
func TestKindEditCreateConflictDelete(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	s := engine.New(engine.Options{
		Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}},
		Confirmer: engine.ConfirmFunc(func(context.Context, engine.ConfirmRequest) (bool, error) { return true, nil }),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	conn := s.Backend().Conn
	dyn := conn.Dynamic()
	run := func(id string, params any) (any, error) {
		t.Helper()
		raw, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		return s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginGUI})
	}
	must := func(id string, params any) any {
		t.Helper()
		res, err := run(id, params)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		return res
	}

	// A namespace left terminating by an earlier run must be gone before it can be created again.
	if !testkit.WaitFor(ctx, func() bool {
		_, err := dyn.Resource(kindNamespaces).Get(ctx, editNamespace, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}) {
		t.Fatalf("namespace %s still exists from an earlier run", editNamespace)
	}
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		if err := conn.DeleteObject(cctx, kindNamespaces, "", editNamespace, ""); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("cleanup: %v", err)
		}
	})

	// Create: the namespace first, then a ConfigMap in it, from one text.
	created := must("object.create", map[string]any{"yaml": `apiVersion: v1
kind: Namespace
metadata:
  name: ` + editNamespace + `
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: demo
  namespace: ` + editNamespace + `
data:
  a: "1"
`}).(engine.CreateResult)
	if created.Created != 2 || created.Items[1].Ref.Kind != "configmaps" || created.Items[0].Ref.Kind != "namespaces" {
		t.Fatalf("create: %+v", created)
	}
	ref := model.ObjectRef{Kind: "configmaps", Namespace: editNamespace, Name: "demo"}

	// Another client labels it with its own field manager.
	other := func(change func(*unstructured.Unstructured)) string {
		t.Helper()
		cur, err := dyn.Resource(kindConfigMaps).Namespace(editNamespace).Get(ctx, "demo", metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		change(cur)
		out, err := dyn.Resource(kindConfigMaps).Namespace(editNamespace).Update(ctx, cur, metav1.UpdateOptions{FieldManager: "other-client"})
		if err != nil {
			t.Fatal(err)
		}
		return out.GetResourceVersion()
	}
	labelled := other(func(u *unstructured.Unstructured) { u.SetLabels(map[string]string{"owner": "other-client"}) })

	// Edit from the cache, once it has the labelled version.
	var opened commands.EditResult
	if !testkit.WaitFor(ctx, func() bool {
		res, err := run("object.edit", map[string]any{"ref": ref})
		if err != nil {
			return false
		}
		opened = res.(commands.EditResult)
		if opened.BaseResourceVersion != labelled {
			_, _ = run("dock.close", map[string]any{"id": opened.Tab.ID})
			return false
		}
		return true
	}) {
		t.Fatalf("object.edit never showed version %s: %+v", labelled, opened)
	}
	if opened.Tab.Title != "ConfigMap demo" || strings.Contains(opened.Text, "managedFields") || !strings.Contains(opened.Text, "owner: other-client") {
		t.Fatalf("edit tab %+v:\n%s", opened.Tab, opened.Text)
	}
	edited := strings.Replace(opened.Text, `a: "1"`, "a: \"1\"\n  b: \"2\"", 1)
	diff := must("editor.diff", map[string]any{"id": opened.Tab.ID, "yaml": edited}).(engine.EditDiff)
	if !diff.Changed || diff.Added != 1 || diff.Removed != 0 || diff.Stale {
		t.Fatalf("diff %+v", diff)
	}
	if dry := must("object.dryRun", map[string]any{"id": opened.Tab.ID, "yaml": edited}).(engine.DryRunResult); !dry.Accepted {
		t.Fatalf("dry run %+v", dry)
	}
	applied := must("object.apply", map[string]any{"id": opened.Tab.ID, "yaml": edited}).(engine.ApplyResult)
	stored, err := dyn.Resource(kindConfigMaps).Namespace(editNamespace).Get(ctx, "demo", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if b, _, _ := unstructured.NestedString(stored.Object, "data", "b"); b != "2" || stored.GetResourceVersion() != applied.ResourceVersion {
		t.Fatalf("after apply: data %v, version %s (apply said %s)", stored.Object["data"], stored.GetResourceVersion(), applied.ResourceVersion)
	}
	// The update carried no managedFields: the server kept every manager's entry and recorded
	// Cogpit's change under its own manager.
	var managers []string
	for _, mf := range stored.GetManagedFields() {
		managers = append(managers, mf.Manager+"/"+string(mf.Operation))
	}
	if !slices.Contains(managers, "other-client/Update") || !slices.Contains(managers, "cogpit/Update") {
		t.Errorf("managedFields after an update without them: %v", managers)
	}
	t.Logf("managedFields after the apply: %v", managers)

	// A stale apply: another client changes the object after the edit's base version.
	changed := other(func(u *unstructured.Unstructured) {
		_ = unstructured.SetNestedField(u.Object, "3", "data", "c")
	})
	stale := strings.Replace(applied.Text, `b: "2"`, `b: "4"`, 1)
	_, err = run("object.apply", map[string]any{"id": opened.Tab.ID, "yaml": stale})
	var ce *engine.ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("stale apply: %v", err)
	}
	if ce.BaseResourceVersion != applied.ResourceVersion || ce.CurrentResourceVersion != changed {
		t.Errorf("conflict %+v, want base %s current %s", ce, applied.ResourceVersion, changed)
	}
	if !slices.ContainsFunc(ce.Diff, func(l engine.DiffLine) bool { return l.Op == engine.DiffRemoved && strings.Contains(l.Text, `c: "3"`) }) {
		t.Errorf("the conflict's diff does not show the other change: %+v", ce.Diff)
	}
	now, err := dyn.Resource(kindConfigMaps).Namespace(editNamespace).Get(ctx, "demo", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	b, _, _ := unstructured.NestedString(now.Object, "data", "b")
	c, _, _ := unstructured.NestedString(now.Object, "data", "c")
	if b != "2" || c != "3" || now.GetResourceVersion() != changed {
		t.Fatalf("the stale apply overwrote the other change: data %v version %s", now.Object["data"], now.GetResourceVersion())
	}

	// Every template passes the API server's validation (dry run, nothing stored).
	tpls, err := engine.Templates()
	if err != nil {
		t.Fatal(err)
	}
	for _, tp := range tpls {
		obj, err := engine.ParseManifest(tp.Text)
		if err != nil {
			t.Fatal(err)
		}
		m, err := conn.MapKind(ctx, obj.GetAPIVersion(), obj.GetKind())
		if err != nil {
			t.Errorf("template %s: %v", tp.Name, err)
			continue
		}
		ns := ""
		if m.Namespaced {
			ns = editNamespace
			obj.SetNamespace(ns)
		}
		if _, err := conn.CreateObject(ctx, m.GVR, ns, obj, true); err != nil {
			t.Errorf("template %s does not validate: %v", tp.Name, err)
		}
		if m.Namespaced == !tp.Namespaced {
			t.Errorf("template %s: namespaced %v, discovery says %v", tp.Name, tp.Namespaced, m.Namespaced)
		}
	}

	// Delete with the cached UID.
	must("dock.close", map[string]any{"id": opened.Tab.ID})
	if !testkit.WaitFor(ctx, func() bool {
		obj, err := s.Object(ctx, model.ObjectRef{Kind: "namespaces", Name: editNamespace})
		return err == nil && obj != nil
	}) {
		t.Fatal("the namespace never reached the cache")
	}
	must("object.delete", map[string]any{"ref": ref})
	if _, err := dyn.Resource(kindConfigMaps).Namespace(editNamespace).Get(ctx, "demo", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Errorf("after object.delete: %v", err)
	}
}
