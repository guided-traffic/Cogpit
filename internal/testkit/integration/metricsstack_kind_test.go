//go:build kind

package integration_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestKindMetricsStack installs Cogpit Metrics (D-61) into the kind cluster through the engine, as
// an agent would (confirmed): Prometheus and kube-state-metrics from the bundled charts, with the
// kind cluster's default StorageClass for the history. node-exporter is left out: the kind
// cluster's own Prometheus runs one on the node's port 9100. It checks that detection finds the new
// Prometheus first and that it serves a node's and a pod's history, then removes everything and
// checks that nothing of it is left, cluster-scoped objects included. It changes which Prometheus
// detection finds while it runs, so it runs only with COGPIT_KIND_METRICS_STACK=1, never in make
// test-kind or CI.
func TestKindMetricsStack(t *testing.T) {
	if os.Getenv("COGPIT_KIND_METRICS_STACK") != "1" {
		t.Skip("installs a Prometheus that detection prefers to the kind cluster's: set COGPIT_KIND_METRICS_STACK=1")
	}
	kubeconfig, contextName := kindEnv(t)
	metricsStackTrial(t, kubeconfig, contextName)
}

// TestRealClusterMetricsStack is the same trial on a real cluster lent for tests
// (COGPIT_REAL_KUBECONFIG), beside the cluster's own Prometheus: Prometheus and kube-state-metrics
// in cogpit-metrics, node-exporter left out (the cluster's own holds port 9100 on every node), and
// everything removed after. It writes to the cluster, so it also needs COGPIT_REAL_INSTALL=1.
func TestRealClusterMetricsStack(t *testing.T) {
	kubeconfig, contextName := realCluster(t)
	if os.Getenv("COGPIT_REAL_INSTALL") != "1" {
		t.Skip("installs into the lent cluster: set COGPIT_REAL_INSTALL=1")
	}
	metricsStackTrial(t, kubeconfig, contextName)
}

func metricsStackTrial(t *testing.T, kubeconfig, contextName string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kc := conn.Kube()
	if _, err := kc.CoreV1().Namespaces().Get(ctx, helm.MetricsNamespace, metav1.GetOptions{}); err == nil {
		t.Fatalf("namespace %s exists already: remove it first", helm.MetricsNamespace)
	}
	k := newKindSession(ctx, t, kubeconfig, contextName, engine.NewMemoryHelmJournal())
	defer k.s.Close()
	// Whatever happens, nothing of the stack stays.
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer ccancel()
		_, _ = k.dispatch(cctx, engine.OriginGUI, "cluster.setMetricsStack", map[string]any{"prometheus": false, "kubeStateMetrics": false, "nodeExporter": false})
		checkNothingLeft(cctx, t, kc)
	})

	res, err := k.dispatch(ctx, engine.OriginExternalAgent, "cluster.metricsStack", map[string]any{"prometheus": true, "kubeStateMetrics": true})
	if err != nil {
		t.Fatal(err)
	}
	view := res.(commands.MetricsStackView)
	t.Logf("storage: %s", view.StorageText)
	t.Logf("plan: %+v", *view.Plan)
	if view.Plan == nil || len(view.Plan.Install) != 2 || !strings.Contains(view.Plan.Detail, "ClusterRole cogpit-prometheus-server") {
		t.Fatalf("plan %+v", view.Plan)
	}

	counter := k.s.Backend().Conn.Counter()
	before := counter.Snapshot()
	start := time.Now()
	res, err = k.dispatch(ctx, engine.OriginExternalAgent, "cluster.setMetricsStack", map[string]any{"prometheus": true, "kubeStateMetrics": true})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	r := res.(commands.MetricsStackResult)
	t.Logf("installed %v in %s, namespace created %v; requests %v", r.Installed, time.Since(start).Round(time.Millisecond), r.NamespaceCreated, delta(before, counter.Snapshot()))
	k.mu.Lock()
	asked := append([]engine.ConfirmRequest(nil), k.asked...)
	k.mu.Unlock()
	if len(asked) != 1 || asked[0].Danger != engine.DangerMutating || len(asked[0].Objects) != 3 {
		t.Fatalf("confirmations %+v", asked)
	}
	t.Logf("the confirmation: %s", asked[0].Message())
	ns, err := kc.CoreV1().Namespaces().Get(ctx, helm.MetricsNamespace, metav1.GetOptions{})
	if err != nil || ns.Labels["app.kubernetes.io/managed-by"] != "cogpit" {
		t.Fatalf("the namespace: %+v %v", ns, err)
	}

	// The Deployments become ready (the cluster pulls the images).
	for _, name := range []string{"cogpit-prometheus-server", "cogpit-kube-state-metrics"} {
		if !waitEvery(ctx, 2*time.Second, func() bool {
			d, err := kc.AppsV1().Deployments(helm.MetricsNamespace).Get(ctx, name, metav1.GetOptions{})
			return err == nil && d.Status.ReadyReplicas >= 1
		}) {
			t.Fatalf("deployment %s never became ready", name)
		}
		t.Logf("deployment %s ready after %s", name, time.Since(start).Round(time.Second))
	}
	if pvcs, err := kc.CoreV1().PersistentVolumeClaims(helm.MetricsNamespace).List(ctx, metav1.ListOptions{}); err == nil {
		for _, c := range pvcs.Items {
			t.Logf("claim %s: %s, volume %s", c.Name, c.Status.Phase, c.Spec.VolumeName)
		}
	}

	if _, err := k.dispatch(ctx, engine.OriginExternalAgent, "ui.openView", map[string]any{"view": "nodes"}); err != nil {
		t.Fatal(err)
	}
	var node model.ObjectRef
	if !testkit.WaitFor(ctx, func() bool {
		tbl := k.s.Table("nodes")
		if tbl == nil || !tbl.Synced || len(tbl.Rows) == 0 {
			return false
		}
		node = tbl.Rows[0].Ref
		return true
	}) {
		t.Fatal("no node")
	}
	// The new Prometheus has a node's CPU and memory once it scraped for a few minutes (a rate needs
	// two samples a minute apart in its window): asked directly, every 15 s.
	prom := metrics.NewPrometheus(kc, metrics.Target{Namespace: metrics.CogpitPrometheusNamespace, Service: metrics.CogpitPrometheusService, Port: "http"}, metrics.PrometheusOptions{})
	if !waitEvery(ctx, 15*time.Second, func() bool {
		to := time.Now()
		got, err := prom.ObjectRange(ctx, metrics.RangeQuery{Kind: metrics.KindNodes, Name: node.Name, From: to.Add(-10 * time.Minute), To: to, Step: 30 * time.Second, Want: metrics.WantUsage})
		return err == nil && got.Targets[""] != nil && len(got.Targets[""].Usage) > 1
	}) {
		t.Fatal("Cogpit's Prometheus never had the node's usage")
	}
	t.Logf("the node's usage in Cogpit's Prometheus after %s", time.Since(start).Round(time.Second))
	// kube-state-metrics, found by its service's annotation, gives the cluster's pod counts (the
	// Overview's Pods history).
	if !waitEvery(ctx, 15*time.Second, func() bool {
		to := time.Now()
		got, err := prom.ObjectRange(ctx, metrics.RangeQuery{Kind: metrics.KindCluster, From: to.Add(-10 * time.Minute), To: to, Step: 30 * time.Second, Want: metrics.WantPods})
		return err == nil && got.PodsSource == metrics.PodsFromKubeStateMetrics
	}) {
		t.Fatal("Cogpit's Prometheus never had kube-state-metrics' pods")
	}
	t.Logf("kube-state-metrics' pods in Cogpit's Prometheus after %s", time.Since(start).Round(time.Second))

	// Detection finds Cogpit's own Prometheus first, and a node's drawer reads its history.
	k.s.Backend().Metrics.Forget()
	if _, err := k.dispatch(ctx, engine.OriginExternalAgent, "ui.navigate", map[string]any{"ref": node}); err != nil {
		t.Fatal(err)
	}
	var m *engine.ObjectMetrics
	if !testkit.WaitFor(ctx, func() bool {
		m, _ = k.s.ObjectMetrics(node)
		return m != nil && len(m.Providers) > 0 && m.Providers[0].Service == metrics.CogpitPrometheusNamespace+"/"+metrics.CogpitPrometheusService+":http" &&
			m.HistorySource != "" && len(m.Targets) > 0 && len(m.Targets[0].Samples) > 1
	}) {
		t.Fatalf("the node's drawer never read Cogpit's Prometheus: %+v", m)
	}
	t.Logf("node %s: source %q, history from %q, %d points, ready after %s", node.Name, m.Source, m.HistorySource, len(m.Targets[0].Samples), time.Since(start).Round(time.Second))
	logProviders(t, "node", m.Providers)

	// Remove it all: destructive, confirmed; nothing is left.
	start = time.Now()
	res, err = k.dispatch(ctx, engine.OriginExternalAgent, "cluster.setMetricsStack", map[string]any{"prometheus": false, "kubeStateMetrics": false})
	if err != nil {
		t.Fatalf("removal: %v", err)
	}
	r = res.(commands.MetricsStackResult)
	k.mu.Lock()
	last := k.asked[len(k.asked)-1]
	k.mu.Unlock()
	if last.Danger != engine.DangerDestructive || len(r.Removed) != 2 || !r.NamespaceDeleted {
		t.Errorf("removal %+v, confirmation %s", r, last.Message())
	}
	t.Logf("removed %v in %s, namespace deleted %v", r.Removed, time.Since(start).Round(time.Millisecond), r.NamespaceDeleted)
	checkNothingLeft(ctx, t, kc)
}

// waitEvery polls cond every d until it holds or ctx ends: for conditions that ask the API server.
func waitEvery(ctx context.Context, d time.Duration, cond func() bool) bool {
	for {
		if cond() {
			return true
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(d):
		}
	}
}

// checkNothingLeft waits until the stack's namespace is gone and fails on any cluster role, binding
// or persistent volume of it that is left.
func checkNothingLeft(ctx context.Context, t *testing.T, kc kubernetes.Interface) {
	t.Helper()
	if !waitEvery(ctx, 2*time.Second, func() bool {
		_, err := kc.CoreV1().Namespaces().Get(ctx, helm.MetricsNamespace, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}) {
		t.Errorf("namespace %s is still there", helm.MetricsNamespace)
	}
	roles, err := kc.RbacV1().ClusterRoles().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range roles.Items {
		if strings.HasPrefix(r.Name, "cogpit-") {
			t.Errorf("cluster role %s is left", r.Name)
		}
	}
	bindings, err := kc.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bindings.Items {
		if strings.HasPrefix(b.Name, "cogpit-") {
			t.Errorf("cluster role binding %s is left", b.Name)
		}
	}
	if !waitEvery(ctx, 2*time.Second, func() bool {
		pvs, err := kc.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return false
		}
		for _, pv := range pvs.Items {
			if pv.Spec.ClaimRef != nil && pv.Spec.ClaimRef.Namespace == helm.MetricsNamespace {
				return false
			}
		}
		return true
	}) {
		t.Errorf("a persistent volume of %s is left", helm.MetricsNamespace)
	}
	t.Logf("nothing of the stack is left: no namespace %s, no cogpit-* cluster role or binding, no volume of it", helm.MetricsNamespace)
}
