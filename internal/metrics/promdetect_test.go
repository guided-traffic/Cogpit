package metrics_test

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

func TestCandidatesRankKnownServices(t *testing.T) {
	svc := func(ns, name string, ports ...metrics.ServicePort) metrics.Service {
		return metrics.Service{Namespace: ns, Name: name, Ports: ports}
	}
	p := func(name string, port int32) metrics.ServicePort { return metrics.ServicePort{Name: name, Port: port} }
	got := metrics.Candidates([]metrics.Service{
		svc("default", "kubernetes", p("https", 443)),
		svc("monitoring", "prometheus-alertmanager", p("http", 9093)),
		svc("monitoring", "prometheus-node-exporter", p("metrics", 9100)),
		svc("monitoring", "prometheus-server", p("http", 80)),
		svc("monitoring", "prometheus-operated", p("web", 9090)),
		svc("monitoring", "stack-kube-prometheus-prometheus", p("reloader-web", 8080), p("http-web", 9090)),
		svc("monitoring", "stack-kube-prometheus-operator", p("https", 443)),
		svc("apps", "prometheus", p("", 9090)),
		svc("lens-metrics", "prometheus", p("web", 80)),
		svc("monitoring", "prometheus-k8s", p("web", 9090)),
		svc("monitoring", "thanos-query", p("http", 9090)),
		svc("monitoring", "prometheus-pushgateway", p("http", 9091)),
		svc("x", "prometheus-server-no-ports"),
	})
	var names []string
	for _, c := range got {
		names = append(names, c.Target.String()+":"+c.Target.Port)
	}
	want := []string{
		"monitoring/stack-kube-prometheus-prometheus:http-web",
		"monitoring/prometheus-k8s:web",
		"monitoring/prometheus-operated:web",
		"monitoring/prometheus-server:http",
		"monitoring/thanos-query:http", // since 2026-10-16 night, Thanos by its name (D-71)
		"lens-metrics/prometheus:web",
		"apps/prometheus:9090",
	}
	if strings.Join(names, " ") != strings.Join(want, " ") {
		t.Errorf("candidates:\n got %v\nwant %v", names, want)
	}

	// kube-prometheus-stack's names by release: the release alone when it contains the chart's
	// name, else <release>-kube-prometheus-stack cut to 26 characters, so a short release keeps
	// "-stack" (kps, 2026-10-12) and a longer one loses it.
	for _, name := range []string{"kube-prometheus-stack-prometheus", "kps-kube-prometheus-stack-prometheus", "monitoring-kube-prometheus-prometheus"} {
		if c := metrics.Candidates([]metrics.Service{svc("cogpit-kps", name, p("http-web", 9090))}); len(c) != 1 || c[0].Score != 100 {
			t.Errorf("%s: candidates %+v, want it first", name, c)
		}
	}
	if c := metrics.Candidates([]metrics.Service{svc("x", "kube-prometheus-stack-operator", p("https", 443))}); len(c) != 0 {
		t.Errorf("the operator's service is a candidate: %+v", c)
	}
}

func TestDetectFoundThroughTheServiceProxy(t *testing.T) {
	c := newPromCluster(t, false)
	c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
	det := metrics.NewPrometheusDetector(c.conn.Kube(), servicesOf(web("monitoring", "prometheus-server", 80), web("default", "web", 80)), metrics.PrometheusOptions{})

	prov, st := det.Detect(context.Background())
	if prov == nil || !st.Usable() || st.Provider != "prometheus" {
		t.Fatalf("state = %+v", st)
	}
	if st.Label != "Prometheus (monitoring/prometheus-server)" {
		t.Errorf("label = %q", st.Label)
	}
	if caps := prov.Capabilities(); !caps.History || !caps.Network || !caps.Filesystem {
		t.Errorf("capabilities = %+v", caps)
	}
	if n := c.proxyCount(); n != 1 {
		t.Errorf("%d service proxy requests, want 1 buildinfo probe", n)
	}
	if reqs := c.prom.requests(); len(reqs) != 1 || reqs[0].path != "/api/v1/status/buildinfo" {
		t.Errorf("requests = %+v", reqs)
	}
	if n := c.srv.Requests()["list services"]; n != 0 {
		t.Errorf("detection listed services itself (%d)", n)
	}
}

func TestDetectNotFoundCostsNothing(t *testing.T) {
	c := newPromCluster(t, false)
	for name, svcs := range map[string][]metrics.Service{
		"no services":  nil,
		"no candidate": {web("default", "web", 80), web("monitoring", "prometheus-alertmanager", 9093)},
	} {
		det := metrics.NewPrometheusDetector(c.conn.Kube(), servicesOf(svcs...), metrics.PrometheusOptions{})
		prov, st := det.Detect(context.Background())
		if prov != nil || st.Status != metrics.StatusNotServed {
			t.Errorf("%s: state = %+v", name, st)
		}
	}
	if n := c.proxyCount(); n != 0 {
		t.Errorf("%d proxy requests with no candidate", n)
	}
}

func TestDetectCandidateThatIsNotPrometheus(t *testing.T) {
	c := newPromCluster(t, false)
	c.srv.SetProxy("monitoring", "prometheus-server:http", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>hello</html>"))
	}))
	det := metrics.NewPrometheusDetector(c.conn.Kube(), servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
	prov, st := det.Detect(context.Background())
	if prov != nil || st.Status != metrics.StatusNotServed {
		t.Errorf("state = %+v, want notServed", st)
	}
}

func TestDetectForbiddenIsAStateAndStopsProbing(t *testing.T) {
	c := newPromCluster(t, false)
	c.prom.status = http.StatusForbidden
	c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
	c.srv.SetProxy("monitoring", "prometheus-operated:http", c.prom)
	det := metrics.NewPrometheusDetector(c.conn.Kube(), servicesOf(web("monitoring", "prometheus-server", 80), web("monitoring", "prometheus-operated", 9090)), metrics.PrometheusOptions{})
	prov, st := det.Detect(context.Background())
	if prov != nil || st.Status != metrics.StatusForbidden || !strings.Contains(st.Message, "services/proxy") {
		t.Fatalf("state = %+v", st)
	}
	if n := c.proxyCount(); n != 1 {
		t.Errorf("%d probes after a 403, want 1", n)
	}
}

func TestDetectUnreachableIsUnavailableAndTriesTheNext(t *testing.T) {
	c := newPromCluster(t, false)
	// prometheus-operated has no endpoints (no handler: 503); prometheus-server works.
	c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
	det := metrics.NewPrometheusDetector(c.conn.Kube(), servicesOf(web("monitoring", "prometheus-operated", 9090), web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
	prov, st := det.Detect(context.Background())
	if prov == nil || !st.Usable() || !strings.Contains(st.Label, "prometheus-server") {
		t.Fatalf("state = %+v", st)
	}
	if n := c.proxyCount(); n != 2 {
		t.Errorf("%d probes, want 2 (the dead one, then the live one)", n)
	}

	c2 := newPromCluster(t, false)
	det = metrics.NewPrometheusDetector(c2.conn.Kube(), servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
	prov, st = det.Detect(context.Background())
	if prov != nil || st.Status != metrics.StatusUnavailable || !strings.Contains(st.Message, "prometheus-server") {
		t.Errorf("only a dead candidate: state = %+v", st)
	}
}

func TestDetectBoundsTheProbes(t *testing.T) {
	c := newPromCluster(t, false)
	var svcs []metrics.Service
	for _, ns := range []string{"a", "b", "c", "d", "e"} {
		svcs = append(svcs, web(ns, "prometheus-server", 80))
	}
	det := metrics.NewPrometheusDetector(c.conn.Kube(), servicesOf(svcs...), metrics.PrometheusOptions{})
	_, st := det.Detect(context.Background())
	if st.Status != metrics.StatusUnavailable {
		t.Errorf("state = %+v", st)
	}
	if n := c.proxyCount(); n != metrics.MaxProbes {
		t.Errorf("%d probes, want at most %d", n, metrics.MaxProbes)
	}
}

func TestDetectOverride(t *testing.T) {
	c := newPromCluster(t, false)
	c.prom.prefix = "/prom"
	c.srv.SetProxy("obs", "thanos:9090", c.prom)
	det := metrics.NewPrometheusDetector(c.conn.Kube(), servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
	det.SetOverride(&metrics.Override{Namespace: "obs", Service: "thanos", Port: "9090", Prefix: "/prom/"})
	prov, st := det.Detect(context.Background())
	if prov == nil || st.Label != "Prometheus (obs/thanos)" {
		t.Fatalf("state = %+v", st)
	}
	if n := c.proxyCount(); n != 1 {
		t.Errorf("%d requests: an override is probed alone", n)
	}
	if reqs := c.prom.requests(); len(reqs) != 1 || reqs[0].path != "/prom/api/v1/status/buildinfo" {
		t.Errorf("requests = %+v", reqs)
	}
	det.SetOverride(&metrics.Override{Namespace: "obs"}) // not valid: cleared, detection is back
	if prov, _ := det.Detect(context.Background()); prov != nil {
		t.Error("an invalid override still applied")
	}
}

func TestDetectServicesFailureIsAState(t *testing.T) {
	c := newPromCluster(t, false)
	det := metrics.NewPrometheusDetector(c.conn.Kube(), metrics.ServiceSourceFunc(func(context.Context) ([]metrics.Service, error) {
		return nil, errors.New("services are not synced")
	}), metrics.PrometheusOptions{})
	prov, st := det.Detect(context.Background())
	if prov != nil || st.Status != metrics.StatusUnavailable {
		t.Errorf("state = %+v", st)
	}
}

func TestChainPrefersPrometheusAndFallsBackToTheMetricsServer(t *testing.T) {
	t.Run("prometheus found", func(t *testing.T) {
		c := newPromCluster(t, true)
		c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
		det, _, err := metrics.ChainForConn(c.conn, servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
		if err != nil {
			t.Fatal(err)
		}
		prov, st := det.Detect(context.Background())
		if prov == nil || prov.ID() != "prometheus" || !st.Usable() {
			t.Errorf("state = %+v", st)
		}
		if n := c.conn.Counter().Snapshot().ByVerbResource["discovery"]; n > 4 {
			t.Logf("discovery requests: %d", n) // the Metrics Server is not asked once Prometheus answers
		}
	})
	t.Run("not found falls back", func(t *testing.T) {
		c := newPromCluster(t, true)
		det, _, err := metrics.ChainForConn(c.conn, servicesOf(), metrics.PrometheusOptions{})
		if err != nil {
			t.Fatal(err)
		}
		prov, st := det.Detect(context.Background())
		if prov == nil || prov.ID() != "metrics-server" || !st.Usable() {
			t.Errorf("state = %+v", st)
		}
		if n := c.proxyCount(); n != 0 {
			t.Errorf("%d proxy requests without a candidate", n)
		}
	})
	t.Run("forbidden falls back", func(t *testing.T) {
		c := newPromCluster(t, true)
		c.prom.status = http.StatusForbidden
		c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
		det, _, _ := metrics.ChainForConn(c.conn, servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
		prov, st := det.Detect(context.Background())
		if prov == nil || prov.ID() != "metrics-server" || !st.Usable() {
			t.Errorf("state = %+v", st)
		}
	})
	t.Run("neither: the telling state wins", func(t *testing.T) {
		c := newPromCluster(t, false)
		c.prom.status = http.StatusForbidden
		c.srv.SetProxy("monitoring", "prometheus-server:http", c.prom)
		det, _, _ := metrics.ChainForConn(c.conn, servicesOf(web("monitoring", "prometheus-server", 80)), metrics.PrometheusOptions{})
		prov, st := det.Detect(context.Background())
		if prov != nil || st.Status != metrics.StatusForbidden {
			t.Errorf("state = %+v, want the forbidden proxy, not 'no metrics-server'", st)
		}
	})
	t.Run("neither and nothing wrong", func(t *testing.T) {
		c := newPromCluster(t, false)
		det, _, _ := metrics.ChainForConn(c.conn, servicesOf(), metrics.PrometheusOptions{})
		prov, st := det.Detect(context.Background())
		if prov != nil || st.Status != metrics.StatusNotServed {
			t.Errorf("state = %+v", st)
		}
	})
}
