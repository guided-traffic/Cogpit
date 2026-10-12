//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestKindPrometheus checks the drawer's charts against a real Prometheus: the prometheus-community
// `prometheus` chart that `make kind-prometheus` installs into the kind cluster (monitoring/
// prometheus-server, scraping the kubelets' cAdvisor every minute). Detection finds it among the
// cached Services, a node's drawer and then a running pod's drawer get CPU, memory, network and
// filesystem history through the API server's service proxy, a chart's range changes through
// ui.setMetricsRange, and metrics.get answers from Prometheus. The node comes first on purpose: on
// containerd the kubelet's cAdvisor has container_fs_usage_bytes for the node's root cgroup but
// not for containers, so the filesystem metric must be found per kind. It only reads. Skipped
// when the cluster has no Prometheus.
func TestKindPrometheus(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	svc := model.ObjectRef{Kind: "services", Namespace: "monitoring", Name: "prometheus-server"}
	if _, err := s.Object(ctx, svc); errors.Is(err, engine.ErrNotFound) {
		t.Skip("the kind cluster has no Prometheus: run make kind-prometheus")
	} else if err != nil {
		t.Fatal(err)
	}
	dispatch := func(id string, params any) any {
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
	const source = "Prometheus (monitoring/prometheus-server)"

	// A node's drawer.
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
	dispatch("ui.navigate", map[string]any{"ref": node})
	m := waitHistory(ctx, t, s, node, func(m *engine.ObjectMetrics) bool {
		w := m.Targets[0]
		return len(w.Samples) > 0 && len(w.Network) > 0 && len(w.Filesystem) > 0
	})
	if m.Source != source || !m.Capabilities.History || !m.Capabilities.Network || !m.Capabilities.Filesystem {
		t.Errorf("node: source %q capabilities %+v", m.Source, m.Capabilities)
	}
	logTarget(t, "node "+node.Name, m.FilesystemSource, m.Targets[0])

	// A running pod's drawer: the seed's nginx.
	dispatch("ui.openView", map[string]any{"view": "pods"})
	var pod model.ObjectRef
	if !testkit.WaitFor(ctx, func() bool {
		tbl := s.Table("pods")
		if tbl == nil || !tbl.Synced {
			return false
		}
		for _, r := range tbl.Rows {
			if r.Ref.Namespace == "cogpit-demo" && strings.HasPrefix(r.Ref.Name, "web-") && r.Status == model.StatusOK {
				pod = r.Ref
				return true
			}
		}
		return false
	}) {
		t.Fatal("no running web pod in cogpit-demo (make kind-seed)")
	}
	dispatch("ui.navigate", map[string]any{"ref": pod})
	m = waitHistory(ctx, t, s, pod, func(m *engine.ObjectMetrics) bool {
		if len(m.Targets) < 2 {
			return false
		}
		w, c := m.Targets[0], m.Targets[1]
		return len(w.Samples) > 0 && len(c.Samples) > 0 && len(w.Network) > 0 && len(c.Network) > 0 &&
			len(w.Filesystem) > 0 && len(c.Filesystem) > 0
	})
	whole, ctr := m.Targets[0], m.Targets[1]
	if m.Source != source || whole.Current == nil || whole.Current.MemoryBytes <= 0 || !ctr.NetworkOfPod || whole.NetworkOfPod {
		t.Errorf("pod: source %q, current %+v, network of pod %v/%v", m.Source, whole.Current, whole.NetworkOfPod, ctr.NetworkOfPod)
	}
	if m.FilesystemSource != metrics.FilesystemUsage && m.FilesystemSource != metrics.FilesystemIO {
		t.Errorf("pod filesystem source %q", m.FilesystemSource)
	}
	logTarget(t, "pod "+pod.Name, m.FilesystemSource, whole)
	logTarget(t, "container "+ctr.Name, m.FilesystemSource, ctr)

	// The pod's chart over six hours: the engine holds the range, ui.state shows it, the history
	// keeps coming.
	dispatch("ui.setMetricsRange", map[string]any{"ref": pod, "range": "6h"})
	m = waitHistory(ctx, t, s, pod, func(m *engine.ObjectMetrics) bool {
		return m.Targets[0].Range == "6h" && len(m.Targets[0].Samples) > 0
	})
	st := dispatch("ui.state", struct{}{}).(commands.StateResult)
	if len(st.DrawerCharts) < 2 || st.DrawerCharts[0].Range != "6h" || st.DrawerCharts[1].Range != "1h" {
		t.Errorf("ui.state's drawer charts = %+v", st.DrawerCharts)
	}
	t.Logf("pod chart at 6h: %d samples from %s", len(m.Targets[0].Samples), m.Targets[0].Samples[0].At.Format(time.RFC3339))

	// metrics.get reads the instant values from Prometheus too.
	pods := dispatch("metrics.get", map[string]any{"kind": "pods", "namespace": "cogpit-demo"}).(*engine.MetricsResult)
	if pods.Status != metrics.StatusAvailable || pods.Source != source || len(pods.Pods) == 0 {
		t.Fatalf("metrics.get pods: %+v", pods)
	}
	for _, p := range pods.Pods {
		if p.MemoryBytes <= 0 {
			t.Errorf("metrics.get: %s has no memory", p.Name)
		}
		t.Logf("metrics.get %s: cpu %dm memory %s", p.Name, p.CPUMilli, model.FormatBytes(p.MemoryBytes))
	}
	nodes := dispatch("metrics.get", map[string]any{"kind": "nodes"}).(*engine.MetricsResult)
	if nodes.Source != source || len(nodes.Nodes) == 0 || nodes.Nodes[0].MemoryBytes <= 0 {
		t.Errorf("metrics.get nodes: %+v", nodes)
	} else {
		t.Logf("metrics.get node %s: cpu %dm memory %s", nodes.Nodes[0].Name, nodes.Nodes[0].CPUMilli, model.FormatBytes(nodes.Nodes[0].MemoryBytes))
	}

	// Budget P13 against the real Prometheus: 90 seconds with the Pods list and the pod's drawer
	// open cost the drawer's polls (one query set per 30 s step: CPU, memory, receive, transmit and
	// the filesystem's one or two queries; three sets, give or take one by the phase) and the list's
	// two instant queries per minute, all GETs through the service proxy.
	dispatch("ui.setMetricsRange", map[string]any{"ref": pod, "range": "default"})
	time.Sleep(2 * time.Second)
	proxy := func() int64 { return s.Diagnostics().Requests.ByVerbResource["get services/proxy"] }
	before, writesBefore := proxy(), writes(s)
	time.Sleep(90 * time.Second)
	n := proxy() - before
	perSet := int64(5)
	if m.FilesystemSource == metrics.FilesystemIO {
		perSet = 6
	}
	if n < 2*perSet+2 || n > 4*perSet+4 || writes(s) != writesBefore {
		t.Errorf("90 s cost %d proxy requests (query sets of %d) and %d writes", n, perSet, writes(s)-writesBefore)
	}
	t.Logf("budget P13 on kind: %d proxy GETs in 90 s (filesystem %s, %d per query set)", n, m.FilesystemSource, perSet)
}

// writes counts the connection's requests that would change something; the access reviews (a
// create of a self-subject review, which stores nothing) are not among them.
func writes(s *engine.Session) int64 {
	var n int64
	for key, c := range s.Diagnostics().Requests.ByVerbResource {
		verb, resource, _ := strings.Cut(key, " ")
		switch verb {
		case "create", "update", "patch", "delete", "deletecollection":
			if !strings.HasPrefix(resource, "selfsubject") {
				n += c
			}
		}
	}
	return n
}

// waitHistory waits until the drawer's metrics of ref are Prometheus's and ok accepts them.
func waitHistory(ctx context.Context, t *testing.T, s *engine.Session, ref model.ObjectRef, ok func(*engine.ObjectMetrics) bool) *engine.ObjectMetrics {
	t.Helper()
	var m *engine.ObjectMetrics
	if !testkit.WaitFor(ctx, func() bool {
		m, _ = s.ObjectMetrics(ref)
		return m != nil && m.Status == metrics.StatusAvailable && len(m.Targets) > 0 && ok(m)
	}) {
		if m != nil {
			for _, tg := range m.Targets {
				t.Logf("%s %q: %d samples, %d network, %d filesystem", ref.Name, tg.Name, len(tg.Samples), len(tg.Network), len(tg.Filesystem))
			}
			t.Fatalf("%s: status %q source %q message %q filesystem %q", ref.Name, m.Status, m.Source, m.Message, m.FilesystemSource)
		}
		t.Fatalf("%s: no metrics", ref.Name)
	}
	return m
}

// logTarget logs a chart's latest numbers, which the test's report quotes.
func logTarget(t *testing.T, what, fsSource string, tg engine.MetricsTarget) {
	t.Helper()
	u := tg.Samples[len(tg.Samples)-1]
	n := tg.Network[len(tg.Network)-1]
	f := tg.Filesystem[len(tg.Filesystem)-1]
	t.Logf("%s: %d points from %s; cpu %dm, memory %s; network rx %.0f B/s tx %.0f B/s; filesystem (%s) usage %.0f B, read %.1f B/s, write %.1f B/s",
		what, len(tg.Samples), tg.Samples[0].At.Format(time.RFC3339), u.CPUMilli, model.FormatBytes(u.MemoryBytes),
		n.ReceiveBytesPerSec, n.TransmitBytesPerSec, fsSource, f.UsageBytes, f.ReadBytesPerSec, f.WriteBytesPerSec)
}
