//go:build kind

package integration_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// accessNamespace is the shared kind cluster's namespace of the access-review test; it creates
// it, acts only inside it, and deletes it.
const accessNamespace = "cogpit-f"

// TestKindAccessReviews connects Cogpit's engine as a ServiceAccount whose only permissions are
// get, list and watch on pods in its own namespace, through a temporary kubeconfig with a token
// from the TokenRequest API, and checks what the access reviews make of it against a real API
// server: Pods listable, Deployments, Nodes and Namespaces hidden from the sidebar, Delete, Edit
// and Logs greyed "not permitted", for one rules review.
func TestKindAccessReviews(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	admin, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	kube, ns := admin.Kube(), accessNamespace
	freshNamespace(ctx, t, kube, ns)

	const sa = "pod-reader"
	if _, err := kube.CoreV1().ServiceAccounts(ns).Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: sa}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.RbacV1().Roles(ns).Create(ctx, &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{Name: sa},
		Rules:      []rbacv1.PolicyRule{{Verbs: []string{"get", "list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods"}}},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.RbacV1().RoleBindings(ns).Create(ctx, &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: sa},
		Subjects:   []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, Name: sa, Namespace: ns}},
		RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: sa},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web"}, Spec: podTemplate("web", nil).Spec,
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	expiry := int64(600)
	tr, err := kube.CoreV1().ServiceAccounts(ns).CreateToken(ctx, sa, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{ExpirationSeconds: &expiry},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// The token lives only in this temporary kubeconfig; it is never logged.
	path := writeTokenKubeconfig(t, admin, "pod-reader", ns, tr.Status.Token)

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{path}}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, "pod-reader"); err != nil {
		t.Fatal(err)
	}

	got := settledKinds(ctx, t, s, "pods", "deployments", "nodes", "namespaces")
	for kind, want := range map[string]bool{"pods": true, "deployments": false, "nodes": false, "namespaces": false} {
		if got[kind] != want {
			t.Errorf("%s: listable %v, want %v", kind, got[kind], want)
		}
	}

	ref := model.ObjectRef{Kind: "pods", Namespace: ns, Name: "web"}
	for id, reason := range map[string]string{
		"object.delete": "not permitted: you may not delete pods in " + ns,
		"object.edit":   "not permitted: you may not update pods in " + ns,
		"pod.logs":      "not permitted: you may not get pods/log in " + ns,
	} {
		cmd, ok := s.Registry().Get(id)
		if !ok {
			t.Fatalf("no command %s", id)
		}
		var err error
		waitFor(ctx, t, id+" greyed", func() bool {
			err = cmd.Enabled(s, cmd.ObjectParams(ref, "web"))
			return errors.Is(err, engine.ErrNotPermitted)
		})
		if err.Error() != reason {
			t.Errorf("%s: %q, want %q", id, err, reason)
		}
	}
	st := s.Backend().Conn.Counter().Snapshot()
	if r, a := reviews(st); r != 1 || a != 0 {
		t.Errorf("reviews: %d rules, %d access; want one rules review of %s and no access review", r, a, ns)
	}
	t.Logf("requests: %v", st.ByVerbResource)
}

// writeTokenKubeconfig writes a kubeconfig file in t.TempDir() with one context, name, for the
// admin connection's server and CA, the given default namespace and a bearer token.
func writeTokenKubeconfig(t *testing.T, admin *cluster.Conn, name, namespace, token string) string {
	t.Helper()
	cfg := admin.RESTConfig()
	kc := clientcmdapi.NewConfig()
	kc.Clusters[name] = &clientcmdapi.Cluster{Server: cfg.Host, CertificateAuthorityData: cfg.CAData}
	kc.AuthInfos[name] = &clientcmdapi.AuthInfo{Token: token}
	kc.Contexts[name] = &clientcmdapi.Context{Cluster: name, AuthInfo: name, Namespace: namespace}
	kc.CurrentContext = name
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := clientcmd.WriteToFile(*kc, path); err != nil {
		t.Fatal("writing the temporary kubeconfig failed") // the error could quote the file
	}
	return path
}
