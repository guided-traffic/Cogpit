//go:build kind

package integration_test

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// whoAmINamespace is the shared kind cluster's namespace of the user test; it creates it, acts only
// inside it, and deletes it.
const whoAmINamespace = "cogpit-who"

// TestKindWhoAmI (D-68) connects Cogpit's engine as a ServiceAccount with no permission of its own,
// through a temporary kubeconfig with a token from the TokenRequest API, and checks that the
// SelfSubjectReview of the connect names it as the API server sees it: its username and its groups,
// confirmed by the API server, for one review; the default RBAC lets every authenticated user ask.
// Disconnected, the cluster shows the username as its last known user.
func TestKindWhoAmI(t *testing.T) {
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
	if u := admin.ReviewUser(ctx); !u.Confirmed || u.Username == "" || len(u.Groups) == 0 {
		t.Errorf("the admin's review: %+v", u)
	}
	kube, ns := admin.Kube(), whoAmINamespace
	freshNamespace(ctx, t, kube, ns)

	const sa = "nobody"
	if _, err := kube.CoreV1().ServiceAccounts(ns).Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: sa}}, metav1.CreateOptions{}); err != nil {
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
	path := writeTokenKubeconfig(t, admin, "sa-entry@kind-who", ns, tr.Status.Token)

	dir := t.TempDir()
	s := engine.New(engine.Options{
		Connector:       &engine.ClusterConnector{KubeconfigPaths: []string{path}},
		ClusterSettings: engine.NewClusterSettingsStore(dir),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, "sa-entry@kind-who"); err != nil {
		t.Fatal(err)
	}
	u := s.UIState().User
	wantUser := "system:serviceaccount:" + ns + ":" + sa
	if u == nil || u.Source != engine.UserFromAPIServer || u.Username != wantUser {
		t.Fatalf("the user: %+v, want %s from the API server", u, wantUser)
	}
	for _, g := range []string{"system:serviceaccounts", "system:serviceaccounts:" + ns, "system:authenticated"} {
		if !slices.Contains(u.Groups, g) {
			t.Errorf("the groups %v lack %s", u.Groups, g)
		}
	}
	if name := engine.ClusterName(s.UIState().Context, engine.ClusterSettings{}); name != "kind-who" {
		t.Errorf("the short name %q", name)
	}
	st := s.Backend().Conn.Counter().Snapshot()
	t.Logf("the user %s, groups %v; requests: %v", u.Username, u.Groups, st.ByVerbResource)
	if n := st.ByVerbResource["create selfsubjectreviews.authentication.k8s.io"]; n != 1 {
		t.Errorf("%d SelfSubjectReviews at the connect, want 1", n)
	}

	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if u := s.UIState().User; u == nil || u.Source != engine.UserLastSeen || u.Username != wantUser || u.LastSeen.IsZero() {
		t.Errorf("not connected: %+v", u)
	}
}
