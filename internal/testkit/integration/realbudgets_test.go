package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// realIdle is COGPIT_REAL_IDLE (a Go duration), else def.
func realIdle(def time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv("COGPIT_REAL_IDLE")); err == nil && d > 0 {
		return d
	}
	return def
}

// realViews are the built-in views the real-cluster measurement opens, one after the other, as a
// user browsing would; the Pods view is the one the connection opens itself.
var realViews = []string{"deployments", "replicasets", "services", "configmaps", "events", "nodes", "namespaces"}

// heapNow returns the live heap after two collections.
func heapNow() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapAlloc
}

// requestDelta is what a connection asked between two counter snapshots, by verb.
func requestDelta(before, after cluster.RequestStats) map[string]int64 {
	out := map[string]int64{}
	for verb, n := range after.ByVerb {
		if d := n - before.ByVerb[verb]; d != 0 {
			out[verb] = d
		}
	}
	return out
}

// deltaByTarget is what a connection asked between two snapshots, by verb and resource (kinds,
// never objects' names).
func deltaByTarget(before, after cluster.RequestStats) map[string]int64 {
	out := map[string]int64{}
	for k, n := range after.ByVerbResource {
		if d := n - before.ByVerbResource[k]; d != 0 {
			out[k] = d
		}
	}
	return out
}

// builtinResource reports whether a counted resource is one of Kubernetes' own (core or a
// *.k8s.io group, apps, batch, ...), which a measurement may name; a custom kind's is not named.
func builtinResource(res string) bool {
	res, _, _ = strings.Cut(res, "/") // a subresource
	_, group, grouped := strings.Cut(res, ".")
	switch {
	case !grouped:
		return true
	case strings.HasSuffix(group, ".k8s.io"):
		return true
	}
	switch group {
	case "apps", "batch", "autoscaling", "policy":
		return true
	}
	return false
}

// realBudgetSession connects a session with every command to the real cluster and opens views, each
// until its table synced.
type realBudgetSession struct {
	t   *testing.T
	ctx context.Context
	s   *engine.Session
}

func (r realBudgetSession) open(view string) {
	r.t.Helper()
	if _, err := r.s.Dispatch(r.ctx, engine.Invocation{ID: "ui.openView", Params: json.RawMessage(`{"view":"` + view + `"}`), Origin: engine.OriginGUI}); err != nil {
		r.t.Fatalf("open %s: %v", view, err)
	}
	if !testkit.WaitFor(r.ctx, func() bool { tbl := r.s.Table(view); return tbl != nil && tbl.Synced }) {
		r.t.Fatalf("the %s view never synced", view)
	}
}

// settled reports whether every informer the cache holds has synced.
func (r realBudgetSession) settled() bool {
	for _, st := range r.s.Backend().Cache.Stats() {
		if !st.Synced {
			return false
		}
	}
	return true
}

func (r realBudgetSession) waitSettled() {
	r.t.Helper()
	if !testkit.WaitFor(r.ctx, r.settled) {
		r.t.Fatal("the cache never settled")
	}
}

// agent dispatches a command as an external agent and decodes its result into out.
func (r realBudgetSession) agent(id string, params, out any) {
	r.t.Helper()
	raw, _ := json.Marshal(params)
	res, err := r.s.Dispatch(r.ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
	if err != nil {
		r.t.Fatalf("%s: %v", id, err)
	}
	if out != nil {
		data, _ := json.Marshal(res)
		if err := json.Unmarshal(data, out); err != nil {
			r.t.Fatal(err)
		}
	}
}

// TestRealBudgetsAndMemory measures on the real cluster, through the engine as the window drives
// it (headless, no window): the memory per kind the Diagnostics view estimates, against the heap
// the process grew by, and what the estimate costs; P10 (a view reopened within the warm period:
// list requests and the time to its rows); P11 (an agent's resources_find, ui_navigate and
// ui_state on a warm cache: API requests); P9 (the Pods view open for COGPIT_REAL_IDLE, ten minutes
// by default: requests by verb, the connection's heartbeat included).
func TestRealBudgetsAndMemory(t *testing.T) {
	kubeconfig, contextName := realCluster(t)
	idle := realIdle(10 * time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), idle+10*time.Minute)
	defer cancel()

	heapBefore := heapNow()
	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatalf("connect: %v", err)
	}
	r := realBudgetSession{t: t, ctx: ctx, s: s}
	r.open("pods")
	for _, v := range realViews {
		r.open(v)
	}
	r.open("pods")
	r.waitSettled()
	conn := s.Backend().Conn

	// Memory: the Diagnostics view's estimate, what it costs, and the heap the process grew by.
	report := s.DiagnosticsReport()
	var costs durations
	for range 20 {
		start := time.Now()
		_ = s.DiagnosticsReport()
		costs = append(costs, time.Since(start))
	}
	heapAfter := heapNow()
	named := map[string]bool{"customresourcedefinitions.apiextensions.k8s.io": true, "pods": true,
		"deployments.apps": true, "replicasets.apps": true, "events.events.k8s.io": true}
	for _, v := range realViews {
		named[v] = true
	}
	var otherObjects int
	var otherBytes int64
	for _, k := range report.Kinds {
		if !named[k.Kind] { // custom kinds and what the views need besides: summed, not named here
			otherObjects, otherBytes = otherObjects+k.Objects, otherBytes+k.Bytes
			continue
		}
		each := int64(0)
		if k.Objects > 0 {
			each = k.Bytes / int64(k.Objects)
		}
		t.Logf("memory: %-48s %6d objects %10s (%s each)", k.Kind, k.Objects, model.FormatBytes(k.Bytes), model.FormatBytes(each))
	}
	t.Logf("memory: %-48s %6d objects %10s", "other kinds", otherObjects, model.FormatBytes(otherBytes))
	t.Logf("memory: the cluster's cache %d objects in %d watches, %s estimated; the heap grew by %s from before connecting "+
		"(the engine, client-go and the cache together); diagnostics_get with the estimate took %v",
		report.Objects, len(report.Watches), model.FormatBytes(report.Bytes), model.FormatBytes(int64(heapAfter)-int64(heapBefore)), costs)
	if report.Bytes <= 0 {
		t.Error("no memory estimated")
	}

	// P10: a view reopened within the warm period.
	var reopen durations
	var p10 map[string]int64
	for range 5 {
		r.open("services")
		before := conn.Counter().Snapshot()
		if s.Table("pods") != nil {
			t.Fatal("the Pods view's table is there while the Services view is open")
		}
		start := time.Now()
		if _, err := s.Dispatch(ctx, engine.Invocation{ID: "ui.openView", Params: json.RawMessage(`{"view":"pods"}`), Origin: engine.OriginGUI}); err != nil {
			t.Fatal(err)
		}
		if !testkit.WaitFor(ctx, func() bool {
			tbl := s.Table("pods")
			return tbl != nil && tbl.Synced && tbl.Total > 0
		}) {
			t.Fatal("the Pods view never showed its rows again")
		}
		reopen = append(reopen, time.Since(start))
		p10 = requestDelta(before, conn.Counter().Snapshot())
		if p10[cluster.VerbList] != 0 {
			t.Errorf("P10: reopening the Pods view cost %d list requests", p10[cluster.VerbList])
		}
	}
	t.Logf("P10: the Pods view reopened within the warm period: rows after %v (WaitFor polls every 5 ms); requests %v (the last round)", reopen, p10)

	// P11: an agent finds a Service by its cluster IP, goes to it and reads the UI state.
	svcs, err := s.Objects(ctx, "services", nil)
	if err != nil || len(svcs) == 0 {
		t.Fatalf("services: %d, %v", len(svcs), err)
	}
	var ip string
	for _, o := range svcs {
		if v, _, _ := unstructured.NestedString(o.Object, "spec", "clusterIP"); v != "" && v != "None" {
			ip = v
			break
		}
	}
	if ip == "" {
		t.Fatal("no Service with a cluster IP")
	}
	r.waitSettled()
	before := conn.Counter().Snapshot()
	start := time.Now()
	var found commands.FindResult
	r.agent("resources.find", map[string]any{"kind": "services", "field": "clusterIP", "value": ip}, &found)
	if found.Count == 0 {
		t.Fatal("resources_find found nothing")
	}
	r.agent("ui.navigate", map[string]any{"ref": found.Matches[0].Ref}, nil)
	r.agent("ui.state", map[string]any{}, nil)
	took := time.Since(start)
	r.waitSettled()
	p11 := deltaByTarget(before, conn.Counter().Snapshot())
	t.Logf("P11: resources_find, ui_navigate and ui_state on a warm cache: %v; requests by verb and resource %v",
		took.Round(10*time.Microsecond), p11)
	for k, n := range p11 {
		if verb, _, _ := strings.Cut(k, " "); verb == cluster.VerbList || verb == cluster.VerbGet {
			t.Errorf("P11: %d %s", n, k)
		}
	}

	// P9: the Pods view open on the cluster, idle.
	if _, err := s.Dispatch(ctx, engine.Invocation{ID: "ui.closeDrawer", Params: json.RawMessage(`{}`), Origin: engine.OriginGUI}); err != nil {
		t.Logf("closing the drawer: %v", err)
	}
	r.open("pods")
	r.waitSettled()
	before = conn.Counter().Snapshot()
	t.Logf("P9: idle for %v with the Pods view open (%d scopes held, open or warm)", idle, len(s.Backend().Cache.Stats()))
	select {
	case <-time.After(idle): // the measurement is the period itself
	case <-ctx.Done():
		t.Fatal("the context ended before the idle period")
	}
	after := conn.Counter().Snapshot()
	byTarget := map[string]int64{}
	for k, n := range deltaByTarget(before, after) { // built-in kinds by name; others (custom kinds) only summed
		if verb, res, ok := strings.Cut(k, " "); ok && !builtinResource(res) {
			k = verb + " (a custom kind)"
		}
		byTarget[k] += n
	}
	t.Logf("P9: requests in %v idle: %v; by verb and resource: %v", idle, requestDelta(before, after), byTarget)
	// The cache lists nothing while idle; the Pods view's metrics are polled (P12, one list of
	// pods.metrics.k8s.io a minute) and the heartbeat asks for /version (D-58).
	for k, n := range byTarget {
		if verb, res, _ := strings.Cut(k, " "); verb == cluster.VerbList && !strings.HasSuffix(res, ".metrics.k8s.io") {
			t.Errorf("P9: %d %s while idle", n, k)
		}
	}
}
