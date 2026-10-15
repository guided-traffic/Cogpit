//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// realCluster returns the kubeconfig and context of a real cluster someone lends for a test run,
// or skips: COGPIT_REAL_KUBECONFIG names the file (passed to Cogpit alone, never merged into the
// user's), COGPIT_REAL_CONTEXT the context (default: the file's current context). Never set in CI.
func realCluster(t *testing.T) (kubeconfig, contextName string) {
	t.Helper()
	kubeconfig = os.Getenv("COGPIT_REAL_KUBECONFIG")
	if kubeconfig == "" {
		t.Skip("COGPIT_REAL_KUBECONFIG is not set: these tests run against a real cluster lent for them")
	}
	contextName = os.Getenv("COGPIT_REAL_CONTEXT")
	if contextName == "" {
		cfg, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			t.Fatal(err)
		}
		contextName = cfg.CurrentContext
	}
	return kubeconfig, contextName
}

// realSession connects the engine to the real cluster, its metrics tuned by opts.
func realSession(ctx context.Context, t *testing.T, opts metrics.Options) (*engine.Session, func(id string, params any) any) {
	t.Helper()
	kubeconfig, contextName := realCluster(t)
	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}, MetricsOptions: opts}})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	return s, func(id string, params any) any {
		t.Helper()
		raw, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		return res
	}
}

// TestRealClusterNodeDrawer checks a node's drawer against a real cluster's Prometheus and Metrics
// Server: its CPU and memory history, and since 2026-10-16 its Network and Filesystem history from
// node-exporter, which on kube-prometheus-stack (whose kubelet ServiceMonitor drops cAdvisor's root
// cgroup) had none; then budget P13 with the drawer open, 90 seconds each with CPU shown and with
// the network shown. It only reads.
func TestRealClusterNodeDrawer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	s, dispatch := realSession(ctx, t, metrics.Options{})
	proxy := func() int64 { return s.Diagnostics().Requests.ByVerbResource["get services/proxy"] }

	dispatch("ui.openView", map[string]any{"view": "nodes"})
	var node model.ObjectRef
	if !testkit.WaitFor(ctx, func() bool {
		tbl := s.Table("nodes")
		if tbl == nil || !tbl.Synced || len(tbl.Rows) == 0 {
			return false
		}
		node = tbl.Rows[0].Ref
		return true
	}) {
		t.Fatal("no node")
	}
	opened := time.Now()
	dispatch("ui.navigate", map[string]any{"ref": node})
	m := waitHistory(ctx, t, s, node, func(m *engine.ObjectMetrics) bool {
		return len(m.Targets[0].Samples) > 0 && m.HistorySource != ""
	})
	t.Logf("source %q, history from %q", m.Source, m.HistorySource)
	logProviders(t, "node", m.Providers)
	if m.HistorySource != metrics.UsageNone {
		checkPast(t, "node CPU and memory", m.Targets[0].Samples, opened)
	}

	dispatch("ui.setMetricsVariant", map[string]any{"ref": node, "variant": "network"})
	m = waitHistory(ctx, t, s, node, func(m *engine.ObjectMetrics) bool { return len(m.Targets[0].Network) > 0 })
	net := m.Targets[0].Network
	pastNet := 0
	for _, p := range net {
		if p.At.Before(opened.Add(-time.Minute)) {
			pastNet++
		}
	}
	if pastNet == 0 {
		t.Errorf("node network: no point from before the drawer opened (%d points)", len(net))
	}
	dispatch("ui.setMetricsVariant", map[string]any{"ref": node, "variant": "filesystem"})
	m = waitHistory(ctx, t, s, node, func(m *engine.ObjectMetrics) bool { return len(m.Targets[0].Filesystem) > 0 })
	m.Targets[0].Network = net
	t.Logf("node network: %d points, %d from before the drawer opened; filesystem: %d points (%s)", len(net), pastNet, len(m.Targets[0].Filesystem), m.FilesystemSource)
	logTarget(t, "node "+node.Name, m.FilesystemSource, m.Targets[0])

	// Budget P13 with the node's drawer at 1 h: CPU shown, then the network shown.
	measure := func(what string) int64 {
		p0, w0 := proxy(), writes(s)
		time.Sleep(90 * time.Second)
		n := proxy() - p0
		if w := writes(s) - w0; w != 0 {
			t.Errorf("%s: %d writes", what, w)
		}
		t.Logf("budget P13 on a real cluster, a node's drawer at 1 h, %s: %d Prometheus requests in 90 s; requests %v", what, n, s.Diagnostics().Requests.ByVerbResource)
		return n
	}
	dispatch("ui.setMetricsVariant", map[string]any{"ref": node, "variant": "cpu"})
	time.Sleep(3 * time.Second)
	measure("CPU shown")
	dispatch("ui.setMetricsVariant", map[string]any{"ref": node, "variant": "network"})
	measure("the network shown")
}
