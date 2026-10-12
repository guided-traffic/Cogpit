package metrics_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

// promHub is a hub over the detector chain of a fake cluster that has the fake Prometheus.
func promHub(t *testing.T, c *promCluster, opts metrics.Options) *metrics.Hub {
	t.Helper()
	c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
	det, _, err := metrics.ChainForConn(c.conn, servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
	if err != nil {
		t.Fatal(err)
	}
	h := metrics.NewHub(det, opts)
	t.Cleanup(h.Close)
	return h
}

var webPodA = metrics.Scope{Kind: metrics.KindPods, Namespace: "web", Name: "a"}

// polls returns a callback that counts polls and a channel that gets each count.
func pollCounter() (func(), chan int) {
	ch := make(chan int, 64)
	n := 0
	return func() { n++; ch <- n }, ch
}

func waitPoll(t *testing.T, ch chan int, n int) {
	t.Helper()
	for {
		select {
		case got := <-ch:
			if got >= n {
				return
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("poll %d never came", n)
		}
	}
}

// P13: a drawer on a cluster with Prometheus costs one query set per step, however many charts and
// consumers share it, and nothing is sent when nothing is open.
func TestPrometheusBudget(t *testing.T) {
	c := newPromCluster(t, false)
	podData(c.prom)
	c.prom.on("container_fs_usage_bytes", s(lc("app"), 1000, 5000, 1030, 6000))
	h := promHub(t, c, metrics.Options{ObjectInterval: 300 * time.Millisecond, MinStep: 5 * time.Millisecond, RangeWindow: 600 * time.Millisecond})

	idle := c.conn.Counter().Snapshot().Total
	time.Sleep(120 * time.Millisecond)
	if got := c.conn.Counter().Snapshot().Total; got != idle || c.proxyCount() != 0 {
		t.Fatalf("an idle hub sent %d requests (%d through the proxy)", got-idle, c.proxyCount())
	}

	onChange, polls := pollCounter()
	a := h.Subscribe(webPodA, onChange)
	b := h.Subscribe(webPodA, nil) // a second consumer of the same drawer
	waitPoll(t, polls, 3)

	// Right after poll 3 (the next one is 300 ms away): the detection probe, then five queries per
	// poll: cpu, memory, rx, tx and the filesystem usage.
	if got := c.proxyCount(); got != 1+3*5 {
		t.Errorf("%d proxy requests after 3 polls, want 16 (1 probe + 3 x 5)", got)
	}
	for _, m := range []string{"container_cpu_usage_seconds_total", "container_memory_working_set_bytes", "container_network_receive_bytes_total", "container_network_transmit_bytes_total", "container_fs_usage_bytes"} {
		if n := c.prom.queryCount(m); n != 3 {
			t.Errorf("%s asked for %d times in 3 polls", m, n)
		}
	}
	snap := c.conn.Counter().Snapshot()
	if snap.ByVerb["watch"] != 0 || snap.ByVerbResource["list services"] != 0 {
		t.Errorf("unexpected requests: %v", snap.ByVerbResource)
	}
	for _, r := range c.prom.requests() {
		if strings.Contains(r.path, "query_range") && (r.step != "0.005" || r.end == "" || r.start == "") {
			t.Errorf("range request %+v", r)
			break
		}
	}

	if st := h.State(webPodA); !st.Usable() || st.Label != "Prometheus (monitoring/prometheus-server)" {
		t.Errorf("state = %+v", st)
	}
	hist := h.History(webPodA)
	if hist == nil || len(hist.Targets) != 3 || hist.FilesystemSource != "usage" || len(hist.Targets[""].Network) == 0 {
		t.Fatalf("history = %+v", hist)
	}
	if p, ok := h.Pod("web", "a"); !ok || p.Total().CPUMilli != 520 || len(p.Containers) != 2 {
		t.Errorf("pod = %+v (%v): the current values come from the last point", p, ok)
	}
	if got := h.Series(webPodA); len(got) != 0 {
		t.Errorf("the hub kept its own series for a provider with history: %+v", got)
	}

	a.Release()
	b.Release()
	settled := c.conn.Counter().Snapshot().Total
	time.Sleep(500 * time.Millisecond)
	if got := c.conn.Counter().Snapshot().Total; got != settled {
		t.Errorf("%d requests after everything was released", got-settled)
	}
}

func TestPrometheusRangeFollowsTheChartsWindows(t *testing.T) {
	c := newPromCluster(t, false)
	podData(c.prom)
	h := promHub(t, c, metrics.Options{}) // the defaults: one hour at 30 s
	onChange, polls := pollCounter()
	sub := h.Subscribe(webPodA, onChange)
	defer sub.Release()
	waitPoll(t, polls, 1)

	// A chart switched to five minutes: one poll at once, at the finer step, still covering the hour.
	sub.SetWindows(5 * time.Minute)
	waitPoll(t, polls, 2)
	var steps, spans []string
	for _, r := range c.prom.requests() {
		if strings.Contains(r.query, "container_cpu") {
			steps = append(steps, r.step)
			spans = append(spans, r.start+".."+r.end)
		}
	}
	if len(steps) != 2 || steps[0] != "30" || steps[1] != "15" {
		t.Errorf("steps = %v, want 30 then 15", steps)
	}
	for _, sp := range spans {
		from, to, _ := strings.Cut(sp, "..")
		if from == "" || to == "" || from >= to {
			t.Errorf("span %s", sp)
		}
	}
	// The same windows again change nothing.
	sub.SetWindows(5 * time.Minute)
	time.Sleep(60 * time.Millisecond)
	if n := c.prom.queryCount("container_cpu"); n != 2 {
		t.Errorf("%d cpu queries; an unchanged window must not poll", n)
	}
}

func TestPrometheusListsUseInstantQueries(t *testing.T) {
	c := newPromCluster(t, false)
	c.prom.on("container_cpu_usage_seconds_total{container", s(map[string]string{"namespace": "web", "pod": "a", "container": "app"}, 1000, 0.25))
	c.prom.on("container_memory_working_set_bytes{container", s(map[string]string{"namespace": "web", "pod": "a", "container": "app"}, 1000, 64*mib))
	h := promHub(t, c, metrics.Options{ListInterval: 300 * time.Millisecond})
	onChange, polls := pollCounter()
	sub := h.Subscribe(metrics.Scope{Kind: metrics.KindPods}, onChange)
	defer sub.Release()
	waitPoll(t, polls, 2)
	if got := c.proxyCount(); got != 1+2*2 {
		t.Errorf("%d proxy requests after 2 list polls, want 5 (1 probe + 2 x 2)", got)
	}
	for _, r := range c.prom.requests() {
		if strings.Contains(r.path, "query_range") {
			t.Errorf("a list asked for a range: %+v", r)
		}
	}
	if p, ok := h.Pod("web", "a"); !ok || p.Total().CPUMilli != 250 || p.Total().MemoryBytes != 64*mib {
		t.Errorf("pod = %+v (%v)", p, ok)
	}
}

func TestPrometheusForbiddenIsTheScopesState(t *testing.T) {
	c := newPromCluster(t, false)
	podData(c.prom)
	c.prom.status = http.StatusForbidden
	h := promHub(t, c, metrics.Options{ObjectInterval: 300 * time.Millisecond})
	onChange, polls := pollCounter()
	sub := h.Subscribe(webPodA, onChange)
	defer sub.Release()
	waitPoll(t, polls, 1)
	if st := h.State(webPodA); st.Status != metrics.StatusForbidden || !strings.Contains(st.Message, "services/proxy") {
		t.Errorf("state = %+v", st)
	}
	if n := c.proxyCount(); n != 1 {
		t.Errorf("%d proxy requests: a forbidden proxy is probed once, not queried", n)
	}
}

// A Prometheus that goes away while a drawer is open is a state; detection runs again.
func TestPrometheusGoneIsAStateAndDetectedAgain(t *testing.T) {
	c := newPromCluster(t, false)
	podData(c.prom)
	h := promHub(t, c, metrics.Options{ObjectInterval: 40 * time.Millisecond, MinStep: 5 * time.Millisecond, RangeWindow: 100 * time.Millisecond, RetryDetectTTL: 10 * time.Millisecond})
	onChange, polls := pollCounter()
	sub := h.Subscribe(webPodA, onChange)
	defer sub.Release()
	waitPoll(t, polls, 1)
	c.prom.mu.Lock()
	c.prom.status = http.StatusNotFound
	c.prom.mu.Unlock()
	until := func(what string, ok func(metrics.State) bool) {
		t.Helper()
		deadline := time.Now().Add(20 * time.Second)
		for !ok(h.State(webPodA)) {
			if time.Now().After(deadline) {
				t.Fatalf("%s: state = %+v", what, h.State(webPodA))
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	until("Prometheus answers 404", func(st metrics.State) bool { return st.Status == metrics.StatusNotServed })
	c.prom.mu.Lock()
	c.prom.status = 0
	c.prom.mu.Unlock()
	until("Prometheus came back", func(st metrics.State) bool { return st.Usable() })
}
