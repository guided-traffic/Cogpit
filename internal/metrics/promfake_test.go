package metrics_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// fakeProm is a Prometheus HTTP API behind the fake API server's service proxy: buildinfo, query
// and query_range. A rule answers a query that contains its text with the series it holds; a query
// no rule matches gets an empty result, as a Prometheus without that metric does.
type fakeProm struct {
	mu      sync.Mutex
	rules   []promRule
	queries []promRequest
	prefix  string
	status  int // when set, every request gets this status
}

type promRule struct {
	contains string
	series   []promTestSeries
}

type promTestSeries struct {
	labels map[string]string
	values [][2]float64 // unix seconds, value
}

type promRequest struct {
	path  string
	query string
	start string
	end   string
	step  string
}

func (f *fakeProm) on(contains string, series ...promTestSeries) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.rules = append(f.rules, promRule{contains: contains, series: series})
}

func (f *fakeProm) requests() []promRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]promRequest(nil), f.queries...)
}

// queryCount counts the queries (not the buildinfo probes) that contain text.
func (f *fakeProm) queryCount(text string) int {
	n := 0
	for _, r := range f.requests() {
		if r.query != "" && strings.Contains(r.query, text) {
			n++
		}
	}
	return n
}

func (f *fakeProm) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	status, prefix := f.status, f.prefix
	f.mu.Unlock()
	if status != 0 {
		http.Error(w, "refused", status)
		return
	}
	path := r.URL.Path
	if prefix != "" {
		if !strings.HasPrefix(path, prefix+"/") {
			http.NotFound(w, r)
			return
		}
		path = strings.TrimPrefix(path, prefix)
	}
	q := r.URL.Query()
	f.mu.Lock()
	f.queries = append(f.queries, promRequest{path: r.URL.Path, query: q.Get("query"), start: q.Get("start"), end: q.Get("end"), step: q.Get("step")})
	rules := append([]promRule(nil), f.rules...)
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch path {
	case "/api/v1/status/buildinfo":
		_, _ = w.Write([]byte(`{"status":"success","data":{"version":"2.55.1","revision":"x"}}`))
	case "/api/v1/query_range", "/api/v1/query":
		result := []any{}
		for _, rule := range rules {
			if !strings.Contains(q.Get("query"), rule.contains) {
				continue
			}
			for _, s := range rule.series {
				item := map[string]any{"metric": s.labels}
				var vals [][]any
				for _, v := range s.values {
					vals = append(vals, []any{v[0], formatValue(v[1])})
				}
				if path == "/api/v1/query" {
					if len(vals) > 0 {
						item["value"] = vals[len(vals)-1]
					}
				} else {
					item["values"] = vals
				}
				result = append(result, item)
			}
			break
		}
		typ := "matrix"
		if path == "/api/v1/query" {
			typ = "vector"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"status": "success", "data": map[string]any{"resultType": typ, "result": result}})
	default:
		http.NotFound(w, r)
	}
}

func formatValue(v float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

// promCluster is a fake API server with a connection, services given to detection, and the fake
// Prometheus behind namespace/service:port.
type promCluster struct {
	srv  *testkit.FakeAPIServer
	conn *cluster.Conn
	prom *fakeProm
}

func newPromCluster(t *testing.T, withMetricsAPI bool) *promCluster {
	t.Helper()
	opts := testkit.FakeAPIOptions{}
	if withMetricsAPI {
		opts.Resources = metricsResources()
	}
	srv, conn := connectFake(t, opts)
	return &promCluster{srv: srv, conn: conn, prom: &fakeProm{}}
}

func (c *promCluster) proxyCount() int64 {
	return c.conn.Counter().Snapshot().ByVerbResource["get services/proxy"]
}

// servicesOf returns a ServiceSource for the given services.
func servicesOf(svcs ...metrics.Service) metrics.ServiceSource {
	return metrics.ServiceSourceFunc(func(context.Context) ([]metrics.Service, error) { return svcs, nil })
}

func web(ns, name string, port int32) metrics.Service {
	return metrics.Service{Namespace: ns, Name: name, Ports: []metrics.ServicePort{{Name: "http", Port: port}}}
}
