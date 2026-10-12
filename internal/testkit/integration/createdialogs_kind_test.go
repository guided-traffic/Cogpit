//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The names the create-dialog test owns on the shared kind cluster: a namespace, and one cluster
// role and one cluster role binding, all deleted at the end.
const (
	createNamespace          = "cogpit-b-create"
	createClusterRole        = "cogpit-b-health-reader"
	createClusterRoleBinding = "cogpit-b-health-readers"
)

// TestKindCreateDialogs creates one object of each create dialog's kind through the engine's
// commands against a real API server (G7): a namespace, a service account in it, a role and a
// role binding that grants it to the account, a cluster role with a non-resource URL and a
// cluster role binding (confirmed, as every cluster-wide grant is), and a resource quota. The
// account's permissions are then asked of the API server (SubjectAccessReview): the bindings
// work. A quota with a resource name the API server does not take is refused by the dry run and
// not created.
func TestKindCreateDialogs(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	var asked []engine.ConfirmRequest
	s := engine.New(engine.Options{
		Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}},
		Confirmer: engine.ConfirmFunc(func(_ context.Context, req engine.ConfirmRequest) (bool, error) {
			asked = append(asked, req)
			return true, nil
		}),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	kube := s.Backend().Conn.Kube()

	// What an earlier run left must be gone before the test creates it again.
	if !testkit.WaitFor(ctx, func() bool {
		_ = kube.RbacV1().ClusterRoleBindings().Delete(ctx, createClusterRoleBinding, metav1.DeleteOptions{})
		_ = kube.RbacV1().ClusterRoles().Delete(ctx, createClusterRole, metav1.DeleteOptions{})
		_, err := kube.CoreV1().Namespaces().Get(ctx, createNamespace, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true
		}
		_ = kube.CoreV1().Namespaces().Delete(ctx, createNamespace, metav1.DeleteOptions{})
		return false
	}) {
		t.Fatalf("namespace %s from an earlier run is still terminating", createNamespace)
	}
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		for name, del := range map[string]func() error{
			"cluster role binding": func() error {
				return kube.RbacV1().ClusterRoleBindings().Delete(cctx, createClusterRoleBinding, metav1.DeleteOptions{})
			},
			"cluster role": func() error {
				return kube.RbacV1().ClusterRoles().Delete(cctx, createClusterRole, metav1.DeleteOptions{})
			},
			"namespace": func() error { return kube.CoreV1().Namespaces().Delete(cctx, createNamespace, metav1.DeleteOptions{}) },
		} {
			if err := del(); err != nil && !apierrors.IsNotFound(err) {
				t.Errorf("cleanup of the %s: %v", name, err)
			}
		}
	})

	run := func(origin engine.Origin, id string, params any) (engine.CreateItem, error) {
		t.Helper()
		raw, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: origin})
		item, _ := res.(engine.CreateItem)
		return item, err
	}
	must := func(origin engine.Origin, id string, params any) {
		t.Helper()
		item, err := run(origin, id, params)
		if err != nil || !item.Created {
			t.Fatalf("%s: %+v, %v", id, item, err)
		}
	}
	gui, agent := engine.OriginGUI, engine.OriginExternalAgent
	ns := createNamespace
	must(gui, "namespace.create", map[string]any{"name": ns})
	must(gui, "serviceAccount.create", map[string]any{"name": "ci", "namespace": ns})
	must(agent, "role.create", map[string]any{"name": "pod-reader", "namespace": ns, "rules": []any{
		map[string]any{"apiGroups": []string{""}, "resources": []string{"pods"}, "verbs": []string{"get", "list", "watch"}}}})
	must(gui, "roleBinding.create", map[string]any{"name": "ci-reads-pods", "namespace": ns,
		"roleRef": map[string]any{"kind": "Role", "name": "pod-reader"}, "subjects": []any{map[string]any{"kind": "ServiceAccount", "name": "ci"}}})
	must(gui, "clusterRole.create", map[string]any{"name": createClusterRole, "rules": []any{
		map[string]any{"nonResourceURLs": []string{"/healthz"}, "verbs": []string{"get"}}}})
	must(gui, "clusterRoleBinding.create", map[string]any{"name": createClusterRoleBinding, "clusterRole": createClusterRole,
		"subjects": []any{map[string]any{"kind": "ServiceAccount", "name": "ci", "namespace": ns}}})
	must(gui, "resourceQuota.create", map[string]any{"name": "small", "namespace": ns, "hard": map[string]string{"pods": "10", "requests.cpu": "1"}})

	// The agent's role and the cluster-wide binding were confirmed; the GUI's other creates were not.
	var confirmed []string
	for _, r := range asked {
		confirmed = append(confirmed, r.Command+" "+string(r.Danger))
	}
	if !slices.Equal(confirmed, []string{"role.create mutating", "clusterRoleBinding.create destructive"}) {
		t.Errorf("confirmations %v", confirmed)
	}

	// The objects as the API server stored them.
	if _, err := kube.CoreV1().ServiceAccounts(ns).Get(ctx, "ci", metav1.GetOptions{}); err != nil {
		t.Error(err)
	}
	rb, err := kube.RbacV1().RoleBindings(ns).Get(ctx, "ci-reads-pods", metav1.GetOptions{})
	if err != nil || rb.RoleRef.Kind != "Role" || len(rb.Subjects) != 1 || rb.Subjects[0].Namespace != ns {
		t.Errorf("role binding %+v, %v", rb, err)
	}
	q, err := kube.CoreV1().ResourceQuotas(ns).Get(ctx, "small", metav1.GetOptions{})
	cpu := q.Spec.Hard[corev1.ResourceRequestsCPU]
	if err != nil || q.Spec.Hard.Pods().String() != "10" || cpu.String() != "1" {
		t.Errorf("quota %+v, %v", q, err)
	}

	// The bindings work: the account may list pods in its namespace and get /healthz, nothing more.
	account := "system:serviceaccount:" + ns + ":ci"
	for _, tt := range []struct {
		attrs *authorizationv1.ResourceAttributes
		path  string
		want  bool
	}{
		{attrs: &authorizationv1.ResourceAttributes{Namespace: ns, Verb: "list", Resource: "pods"}, want: true},
		{attrs: &authorizationv1.ResourceAttributes{Namespace: ns, Verb: "delete", Resource: "pods"}, want: false},
		{attrs: &authorizationv1.ResourceAttributes{Namespace: "default", Verb: "list", Resource: "pods"}, want: false},
		{path: "/healthz", want: true},
	} {
		review := &authorizationv1.SubjectAccessReview{Spec: authorizationv1.SubjectAccessReviewSpec{User: account, ResourceAttributes: tt.attrs}}
		if tt.path != "" {
			review.Spec.NonResourceAttributes = &authorizationv1.NonResourceAttributes{Path: tt.path, Verb: "get"}
		}
		got, err := kube.AuthorizationV1().SubjectAccessReviews().Create(ctx, review, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		if got.Status.Allowed != tt.want {
			t.Errorf("%+v %s: allowed %v, want %v", tt.attrs, tt.path, got.Status.Allowed, tt.want)
		}
	}

	// A resource the API server does not take in a quota stops at the dry run.
	if item, err := run(gui, "resourceQuota.create", map[string]any{"name": "gadgets", "namespace": ns, "hard": map[string]string{"gadgets": "1"}}); err == nil ||
		item.Created || !strings.Contains(err.Error(), "was not created") {
		t.Errorf("a quota on gadgets: %+v, %v", item, err)
	}
	if _, err := kube.CoreV1().ResourceQuotas(ns).Get(ctx, "gadgets", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Errorf("the refused quota exists: %v", err)
	}
}
