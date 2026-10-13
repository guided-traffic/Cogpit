package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// A namespace-limited user against a fake API server that enforces RBAC, measured at the HTTP
// level (G3, G12; architecture §5 and §9): a user whose one RoleBinding is in the context's
// namespace team-a, and a cluster-admin, through the same steps: the Pods view with its CPU and
// Memory, an agent's object_get of a Deployment, resources_list of the Deployments, metrics.get of
// every pod. The limited user's reads end in team-a after one refused read per kind in all
// namespaces, and the pods' metrics are listed in team-a alone; the cluster-admin's counts are the
// ones of before: one read per kind in all namespaces and one list of every pod's metrics.

var podMetricsGVR = schema.GroupVersionResource{Group: "metrics.k8s.io", Version: "v1beta1", Resource: "pods"}

func podMetrics(ns, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "metrics.k8s.io/v1beta1", "kind": "PodMetrics",
		"metadata":  map[string]any{"namespace": ns, "name": name},
		"timestamp": time.Now().UTC().Format(time.RFC3339), "window": "15s",
		"containers": []any{map[string]any{"name": "app", "usage": map[string]any{"cpu": "250m", "memory": "64Mi"}}},
	}}
}

// connectLimited connects an engine to a fake API server that serves pods, deployments, events and
// the Metrics API, enforcing access's rules, with team-a as the context's namespace. team-a holds
// three pods and the Deployment web, team-b two pods; every pod is measured.
func connectLimited(ctx context.Context, t *testing.T, access *testkit.FakeAccess) (*engine.Session, *testkit.FakeAPIServer) {
	t.Helper()
	resources := append(append([]testkit.FakeResource(nil), testkit.DefaultFakeResources...),
		testkit.FakeResource{GVR: testkit.EventsGVR, Kind: "Event", Namespaced: true},
		testkit.FakeResource{GVR: podMetricsGVR, Kind: "PodMetrics", Namespaced: true})
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: "limited-token", Access: access, Resources: resources})
	t.Cleanup(srv.Close)
	for ns, n := range map[string]int{"team-a": 3, "team-b": 2} {
		for i := range n {
			name := fmt.Sprintf("web-%d", i)
			srv.Set(testkit.PodsGVR, testkit.Pod(ns, name))
			srv.Set(podMetricsGVR, podMetrics(ns, name))
		}
	}
	srv.Set(testkit.DeploymentsGVR, testkit.Object("apps/v1", "Deployment", "team-a", "web", nil))
	s := engine.New(engine.Options{
		Connector:       &engine.ClusterConnector{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "team-a")}},
		ClusterSettings: engine.NewClusterSettingsStore(t.TempDir()),
	})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, "fake"); err != nil {
		t.Fatal(err)
	}
	return s, srv
}

// readsOf are the server's reads of one resource, by "verb resource [in namespace]", refused ones
// included.
func readsOf(srv *testkit.FakeAPIServer, resource string) map[string]int64 {
	out := map[string]int64{}
	for k, v := range srv.RequestsIn() {
		if verb, rest, _ := strings.Cut(k, " "); verb != "" && (rest == resource || strings.HasPrefix(rest, resource+" in ")) {
			out[k] = v
		}
	}
	return out
}

func agentDispatch(ctx context.Context, t *testing.T, s *engine.Session, id string, params any) any {
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

// steps runs the user's steps and returns the pods' CPU column as the view shows it once it has
// a measurement for every row.
func limitedSteps(ctx context.Context, t *testing.T, s *engine.Session, rows int) {
	t.Helper()
	if !testkit.WaitFor(ctx, func() bool {
		ts := s.Table("pods")
		if ts == nil || !ts.Synced || len(ts.Rows) != rows {
			return false
		}
		ci := -1
		for i, c := range ts.Columns {
			if c.ID == model.ColumnCPU {
				ci = i
			}
		}
		for _, r := range ts.Rows {
			if ci < 0 || ci >= len(r.Cells) || r.Cells[ci].Text == "" {
				return false
			}
		}
		return true
	}) {
		t.Fatalf("the Pods view never showed %d rows with their CPU: %+v", rows, s.Table("pods"))
	}
	start := time.Now()
	ref := model.ObjectRef{Kind: "deployments", Namespace: "team-a", Name: "web"}
	m, ok := agentDispatch(ctx, t, s, "object.get", map[string]any{"ref": ref}).(*engine.DetailModel)
	if !ok || m.Missing || m.Loading {
		t.Fatalf("object.get: %+v", m)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("object.get took %s", d)
	}
	list, ok := agentDispatch(ctx, t, s, "resources.list", map[string]any{"kind": "deployments"}).(commands.ListResult)
	if !ok || list.Total != 1 {
		t.Fatalf("resources.list: %+v", list)
	}
	res, ok := agentDispatch(ctx, t, s, "metrics.get", map[string]any{"kind": "pods"}).(*engine.MetricsResult)
	if !ok || res.Status != "available" || res.Total != rows {
		t.Fatalf("metrics.get: %+v", res)
	}
}

func TestBudgetNamespaceLimitedUser(t *testing.T) {
	t.Run("a RoleBinding in the context's namespace alone", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		rule := testkit.FakeRule{Verbs: []string{"get", "list", "watch"}, APIGroups: []string{"", "apps", "metrics.k8s.io"},
			Resources: []string{"pods", "deployments", "events"}}
		access := &testkit.FakeAccess{Enforce: true, Rules: map[string][]testkit.FakeRule{"team-a": {rule}}}
		s, srv := connectLimited(ctx, t, access)
		limitedSteps(ctx, t, s, 3)

		// The first read of each kind in all namespaces is refused (a streaming watch, and the plain
		// list client-go's reflector falls back to at once: one or two requests), then team-a's alone;
		// the pods' metrics are listed in team-a alone, never in all namespaces.
		for _, res := range []string{"pods", "deployments.apps", "events"} {
			reads := readsOf(srv, res)
			wide := reads["watchlist "+res] + reads["list "+res]
			if wide < 1 || wide > 2 || reads["watchlist "+res+" in team-a"] != 1 || len(reads) > 3 {
				t.Errorf("%s: reads %v, want one refused read in all namespaces (and its list), then team-a's", res, reads)
			}
		}
		if got, want := readsOf(srv, "pods.metrics.k8s.io"), map[string]int64{"list pods.metrics.k8s.io in team-a": 1}; !maps.Equal(got, want) {
			t.Errorf("pod metrics: reads %v, want %v", got, want)
		}
		total := s.Backend().Conn.Counter().Snapshot().Total
		time.Sleep(idle)
		if after := s.Backend().Conn.Counter().Snapshot().Total; after != total {
			t.Errorf("idle cost %d requests: %v", after-total, srv.RequestsIn())
		}
	})

	t.Run("a cluster-admin", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
		defer cancel()
		all := testkit.FakeRule{Verbs: []string{"*"}, APIGroups: []string{"*"}, Resources: []string{"*"}}
		s, srv := connectLimited(ctx, t, &testkit.FakeAccess{Enforce: true, Rules: map[string][]testkit.FakeRule{"": {all}}})
		limitedSteps(ctx, t, s, 5)
		for res, want := range map[string]map[string]int64{
			"pods":                {"watchlist pods": 1},
			"deployments.apps":    {"watchlist deployments.apps": 1},
			"events":              {"watchlist events": 1},
			"pods.metrics.k8s.io": {"list pods.metrics.k8s.io": 1},
		} {
			if got := readsOf(srv, res); !maps.Equal(got, want) {
				t.Errorf("%s: reads %v, want %v (as before)", res, got, want)
			}
		}
	})
}
