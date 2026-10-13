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
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The namespaces of the accessible-namespaces test on the shared kind cluster: it creates them,
// acts only inside them, and deletes them (with the ServiceAccount, Roles and RoleBindings in them).
const (
	accessibleA = "cogpit-an-a"
	accessibleB = "cogpit-an-b"
	accessibleC = "cogpit-an-c" // the ServiceAccount may read nothing here
)

// TestKindAccessibleNamespaces connects Cogpit's engine as a ServiceAccount that may get, list and
// watch pods only in cogpit-an-a and cogpit-an-b, through a temporary kubeconfig with a token from
// the TokenRequest API, with the three cogpit-an-* namespaces as the cluster's accessible
// namespaces, against a real API server: the Pods view shows the two namespaces' pods from one
// watch per namespace, names cogpit-an-c as denied, and sends no cluster-wide list or watch of
// pods (two initial reads of pods in all, each in its namespace: the cache holds no cluster-wide
// pods scope).
func TestKindAccessibleNamespaces(t *testing.T) {
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
	kube := admin.Kube()
	for _, ns := range []string{accessibleA, accessibleB, accessibleC} {
		freshNamespace(ctx, t, kube, ns)
		if _, err := kube.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web"}, Spec: podTemplate("web", nil).Spec,
		}, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	const sa = "an-reader"
	if _, err := kube.CoreV1().ServiceAccounts(accessibleA).Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: sa}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	for _, ns := range []string{accessibleA, accessibleB} {
		if _, err := kube.RbacV1().Roles(ns).Create(ctx, &rbacv1.Role{
			ObjectMeta: metav1.ObjectMeta{Name: sa},
			Rules:      []rbacv1.PolicyRule{{Verbs: []string{"get", "list", "watch"}, APIGroups: []string{""}, Resources: []string{"pods"}}},
		}, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
		if _, err := kube.RbacV1().RoleBindings(ns).Create(ctx, &rbacv1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{Name: sa},
			Subjects:   []rbacv1.Subject{{Kind: rbacv1.ServiceAccountKind, Name: sa, Namespace: accessibleA}},
			RoleRef:    rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: sa},
		}, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	expiry := int64(600)
	tr, err := kube.CoreV1().ServiceAccounts(accessibleA).CreateToken(ctx, sa, &authenticationv1.TokenRequest{
		Spec: authenticationv1.TokenRequestSpec{ExpirationSeconds: &expiry},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatal(err)
	}
	// The token lives only in this temporary kubeconfig; it is never logged.
	path := writeTokenKubeconfig(t, admin, sa, accessibleA, tr.Status.Token)

	s := engine.New(engine.Options{
		Connector:       &engine.ClusterConnector{KubeconfigPaths: []string{path}},
		ClusterSettings: engine.NewClusterSettingsStore(t.TempDir()),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeClusterSettings(sa, func(c *engine.ClusterSettings) error {
		c.AccessibleNamespaces = []string{accessibleA, accessibleB, accessibleC}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, sa); err != nil {
		t.Fatal(err)
	}

	var ts *engine.TableSnapshot
	waitFor(ctx, t, "the Pods view of the two namespaces", func() bool {
		ts = s.Table("pods")
		return ts != nil && ts.Synced && len(ts.Rows) == 2
	})
	var spaces []string
	for _, r := range ts.Rows {
		spaces = append(spaces, r.Ref.Namespace)
	}
	slices.Sort(spaces)
	if !slices.Equal(spaces, []string{accessibleA, accessibleB}) || !slices.Equal(ts.Watched, []string{accessibleA, accessibleB}) ||
		!slices.Equal(ts.Denied, []string{accessibleC}) || ts.Error != "" {
		t.Errorf("rows in %q, watched %q, denied %q, error %q", spaces, ts.Watched, ts.Denied, ts.Error)
	}
	b := s.Backend()
	waitFor(ctx, t, "both watches settled", func() bool {
		st := b.Conn.Counter().Snapshot()
		return st.ByVerbResource["watchlist pods"]+st.ByVerbResource["watch pods"] >= 2
	})
	st := b.Conn.Counter().Snapshot()
	initial := st.ByVerbResource["watchlist pods"] + st.ByVerbResource["list pods"]
	if initial != 2 {
		t.Errorf("initial reads of pods: %d, want 2 (one per namespace; a cluster-wide attempt would be a third): %v", initial, st.ByVerbResource)
	}
	for _, sc := range b.Cache.Stats() {
		if sc.Scope.GVR == testkit.PodsGVR && sc.Scope.Namespace == "" {
			t.Errorf("a cluster-wide pods scope: %+v", sc)
		}
	}
	r, a := reviews(st)
	t.Logf("pod reads: watchlist %d, list %d, watch %d; access reviews: %d rules, %d access; all: %v",
		st.ByVerbResource["watchlist pods"], st.ByVerbResource["list pods"], st.ByVerbResource["watch pods"], r, a, st.ByVerbResource)
	if a != 0 {
		t.Errorf("access reviews: %d, want none (cogpit-an-c's rules deny pods, so no cluster-wide binding grants them)", a)
	}

	// An agent's read goes through the same watches.
	objs, err := s.Objects(ctx, "pods", nil)
	if err != nil || len(objs) != 2 {
		t.Errorf("Objects: %d, %v", len(objs), err)
	}
	if after := b.Conn.Counter().Snapshot(); after.ByVerbResource["watchlist pods"]+after.ByVerbResource["list pods"] != 2 {
		t.Errorf("the agent's read cost a list: %v", after.ByVerbResource)
	}
}
