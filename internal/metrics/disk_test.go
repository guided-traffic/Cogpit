package metrics_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The nodes' disks (G12, the Nodes list's Disk column): node-exporter's root filesystem through
// Prometheus, one instant query per list poll, matched to nodes by their node label or instance;
// never the kubelet's summary API (nodes/proxy).

const gib = float64(1 << 30)

// diskSeries is node-exporter's answer to the disk query: a node by its node label (the prometheus
// chart's and kube-prometheus-stack's service discovery), one by its instance (the node's name with
// the exporter's port), one by its address, and a size without its available bytes (left out).
func diskSeries(p *fakeProm) {
	part := func(name string, l map[string]string) map[string]string {
		out := map[string]string{"part": name}
		for k, v := range l {
			out[k] = v
		}
		return out
	}
	byLabel := map[string]string{"node": "n1", "instance": "10.0.0.1:9100"}
	byName := map[string]string{"instance": "n2:9100"}
	byAddr := map[string]string{"instance": "10.0.0.3:9100"}
	half := map[string]string{"node": "n4", "instance": "10.0.0.4:9100"}
	p.onRel(`node_filesystem_size_bytes{mountpoint="/",fstype!="rootfs"}`,
		s(part("size", byLabel), 0, 100*gib), s(part("avail", byLabel), 0, 25*gib),
		s(part("size", byName), 0, 50*gib), s(part("avail", byName), 0, 45*gib),
		s(part("size", byAddr), 0, 200*gib), s(part("avail", byAddr), 0, 10*gib),
		s(part("size", half), 0, 80*gib),
	)
}

// dualHub is a hub over a fake cluster's Metrics Server and its Prometheus behind the service proxy.
func dualHub(t *testing.T, c *promCluster, opts metrics.Options) *metrics.Hub {
	t.Helper()
	c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
	src, _, err := metrics.SourcesForConn(c.conn, servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	h := metrics.NewHubFor(src, opts)
	t.Cleanup(h.Close)
	return h
}

func TestNodeDisksFromNodeExporter(t *testing.T) {
	c := newPromCluster(t, true)
	diskSeries(c.prom)
	h := dualHub(t, c, metrics.Options{ListInterval: time.Hour})
	onChange, polls := pollCounter()
	sub := h.Subscribe(metrics.DiskScope, onChange)
	defer sub.Release()
	waitPoll(t, polls, 1)

	if st := h.DiskState(); !st.Usable() || st.Label != "Prometheus (monitoring/prometheus-server)" {
		t.Fatalf("disk state = %+v", st)
	}
	for _, tc := range []struct {
		keys       []string
		used, size float64
	}{
		{[]string{"n1"}, 75 * gib, 100 * gib},
		{[]string{"n2", "10.0.0.2"}, 5 * gib, 50 * gib},
		{[]string{"n3", "10.0.0.3"}, 190 * gib, 200 * gib}, // by its address
	} {
		d, ok := h.NodeDisk(tc.keys...)
		if !ok || d.UsedBytes != int64(tc.used) || d.SizeBytes != int64(tc.size) || d.Timestamp.IsZero() {
			t.Errorf("NodeDisk(%v) = %+v, %v; want %v of %v", tc.keys, d, ok, tc.used, tc.size)
		}
	}
	if d, ok := h.NodeDisk("n4", "10.0.0.4"); ok {
		t.Errorf("a node with a size and no available bytes has a disk: %+v", d)
	}
	if _, ok := h.NodeDisk("n9"); ok {
		t.Error("an unknown node has a disk")
	}
	var queries []string
	for _, r := range c.prom.requests() {
		if r.query != "" {
			queries = append(queries, r.query)
		}
	}
	want := `label_replace(max by (node, instance) (node_filesystem_size_bytes{mountpoint="/",fstype!="rootfs"}), "part", "size", "", "") or ` +
		`label_replace(max by (node, instance) (node_filesystem_avail_bytes{mountpoint="/",fstype!="rootfs"}), "part", "avail", "", "")`
	if len(queries) != 1 || queries[0] != want {
		t.Errorf("queries = %q, want the one disk query %q", queries, want)
	}
}

// P12 and P13 for the Nodes list with its Disk column: the Metrics Server's node list per list poll
// as before, and of Prometheus one probe (detection) and one instant query per poll; nothing once
// released. Never nodes/proxy.
func TestNodeDiskBudget(t *testing.T) {
	c := newPromCluster(t, true)
	c.srv.Set(nodeMetricsGVR, &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "metrics.k8s.io/v1beta1", "kind": "NodeMetrics", "metadata": map[string]any{"name": "n1"},
		"timestamp": "2026-10-12T10:00:00Z", "window": "15s", "usage": map[string]any{"cpu": "1", "memory": "2Gi"},
	}})
	diskSeries(c.prom)
	h := dualHub(t, c, metrics.Options{ListInterval: 150 * time.Millisecond})
	count := func(key string) int64 { return c.conn.Counter().Snapshot().ByVerbResource[key] }

	nodesChange, nodePolls := pollCounter()
	diskChange, diskPolls := pollCounter()
	nodes := h.Subscribe(metrics.Scope{Kind: metrics.KindNodes}, nodesChange)
	disk := h.Subscribe(metrics.DiskScope, diskChange)
	waitPoll(t, diskPolls, 3)
	waitPoll(t, nodePolls, 3)
	nodes.Release()
	disk.Release()
	time.Sleep(50 * time.Millisecond) // a poll in flight at the release ends
	settled := c.conn.Counter().Snapshot()
	nDisk, nNodes := 3+drained(diskPolls), 3+drained(nodePolls) // the polls that answered; one more may have been in flight

	probes, diskQueries := 0, 0
	for _, r := range c.prom.requests() {
		switch {
		case strings.HasSuffix(r.path, "/buildinfo"):
			probes++
		case strings.Contains(r.query, "node_filesystem_"):
			diskQueries++
		default:
			t.Errorf("unexpected Prometheus request %+v", r)
		}
	}
	if probes != 1 || diskQueries < nDisk || diskQueries > nDisk+1 {
		t.Errorf("Prometheus: %d probes and %d disk queries for %d disk polls, want 1 and one per poll", probes, diskQueries, nDisk)
	}
	if n := int(count("list nodes.metrics.k8s.io")); n < nNodes || n > nNodes+1 {
		t.Errorf("%d node lists for %d list polls", n, nNodes)
	}
	for key := range settled.ByVerbResource {
		if strings.Contains(key, "nodes/proxy") {
			t.Errorf("the kubelet's proxy was asked: %s", key)
		}
	}
	time.Sleep(400 * time.Millisecond)
	if got := c.conn.Counter().Snapshot().Total; got != settled.Total {
		t.Errorf("%d requests after the Nodes list closed", got-settled.Total)
	}
}

// drained counts the polls a counter's channel still holds.
func drained(ch chan int) int {
	n := 0
	for {
		select {
		case <-ch:
			n++
		default:
			return n
		}
	}
}

// Without Prometheus the column's state says why, and nothing is sent but the detection's look at
// the cached Services (none here: no request at all).
func TestNodeDisksWithoutPrometheus(t *testing.T) {
	c := newPromCluster(t, true)
	src, _, err := metrics.SourcesForConn(c.conn, servicesOf(), metrics.PrometheusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	h := metrics.NewHubFor(src, metrics.Options{ListInterval: time.Hour})
	t.Cleanup(h.Close)
	if st := h.DiskState(); st.Status != metrics.StatusUnknown {
		t.Errorf("state before the first poll = %+v", st)
	}
	onChange, polls := pollCounter()
	sub := h.Subscribe(metrics.DiskScope, onChange)
	defer sub.Release()
	waitPoll(t, polls, 1)
	st := h.DiskState()
	if st.Status != metrics.StatusNotServed || !strings.Contains(st.Message, "no Prometheus was found") {
		t.Errorf("state = %+v", st)
	}
	if n := c.proxyCount(); n != 0 {
		t.Errorf("%d proxy requests without a Prometheus", n)
	}
	if _, ok := h.NodeDisk("n1"); ok {
		t.Error("a disk without a provider")
	}
}

// A Prometheus without node-exporter answers: the column's state names it, and Prometheus stays
// found for the charts (its slot is not marked failing).
func TestNodeDisksWithoutNodeExporter(t *testing.T) {
	c := newPromCluster(t, true)
	h := dualHub(t, c, metrics.Options{ListInterval: time.Hour})
	onChange, polls := pollCounter()
	sub := h.Subscribe(metrics.DiskScope, onChange)
	defer sub.Release()
	waitPoll(t, polls, 1)
	st := h.DiskState()
	if st.Status != metrics.StatusNotServed || !strings.Contains(st.Message, "Prometheus (monitoring/prometheus-server) has no node-exporter series") {
		t.Errorf("state = %+v", st)
	}
	if ps := h.Providers(); ps[0].Provider != metrics.PrometheusID || ps[0].Status != metrics.StatusAvailable || ps[0].LastError != "" {
		t.Errorf("Prometheus after an empty disk answer = %+v", ps[0])
	}
}

// A user refused the service proxy: the column says so (forbidden), with the class, never a body.
func TestNodeDisksForbidden(t *testing.T) {
	c := newPromCluster(t, true)
	c.prom.status = http.StatusForbidden
	h := dualHub(t, c, metrics.Options{ListInterval: time.Hour})
	onChange, polls := pollCounter()
	sub := h.Subscribe(metrics.DiskScope, onChange)
	defer sub.Release()
	waitPoll(t, polls, 1)
	if st := h.DiskState(); st.Status != metrics.StatusForbidden && st.Status != metrics.StatusUnavailable {
		t.Errorf("state with the proxy refused = %+v", st)
	}
	if st := h.DiskState(); strings.Contains(st.Message, "refused") {
		t.Errorf("the state quotes the answer's body: %+v", st)
	}
}

// The disk's share and cells: model.NodeDisk.
func TestNodeDiskShare(t *testing.T) {
	d := model.NodeDisk{UsedBytes: 75, SizeBytes: 100}
	if d.Share() != 0.75 || (model.NodeDisk{}).Share() != 0 {
		t.Errorf("shares = %v, %v", d.Share(), (model.NodeDisk{}).Share())
	}
}
