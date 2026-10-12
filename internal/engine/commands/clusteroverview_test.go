package commands_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// kindsConnector connects "test" to a cache over src with the real profiles of the given kinds.
type kindsConnector struct {
	src  *testkit.MemorySource
	keys []string
}

func (c *kindsConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: "test", Current: true}}, nil
}

func (c *kindsConnector) Connect(context.Context, string) (*engine.Backend, error) {
	var profiles []*model.KindProfile
	for _, k := range c.keys {
		if p, ok := kinds.ByKey(k); ok {
			profiles = append(profiles, p)
		}
	}
	ca := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	return &engine.Backend{Context: "test", Namespace: "default", Cache: ca, Profiles: profiles, Close: ca.Close}, nil
}

// overviewCluster is a node that is not Ready and a warning event about a pod a minute ago.
func overviewCluster() *testkit.MemorySource {
	src := testkit.NewMemorySource()
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "worker-2", map[string]any{
		"status": map[string]any{
			"capacity":   map[string]any{"cpu": "2", "memory": "4Gi", "pods": "110"},
			"conditions": []any{map[string]any{"type": "Ready", "status": "False"}},
		},
	}))
	src.Set(testkit.PodsGVR, testkit.Pod("shop", "web-0"))
	src.Set(testkit.EventsGVR, testkit.Object("v1", "Event", "shop", "web-0.1", map[string]any{
		"involvedObject": map[string]any{"kind": "Pod", "namespace": "shop", "name": "web-0"},
		"reason":         "Unhealthy", "type": "Warning", "message": "Readiness probe failed",
		"lastTimestamp": time.Now().Add(-time.Minute).UTC().Format(time.RFC3339),
	}))
	return src
}

// TestClusterOverviewCommands: an agent opens the cluster Overview, reads it, and works its Issues
// with the commands of any view; ui.select there takes an issue's object of any kind and opens
// its drawer over the page, where a kind's view refuses another kind.
func TestClusterOverviewCommands(t *testing.T) {
	src := overviewCluster()
	s := engine.New(engine.Options{Connector: &kindsConnector{src: src, keys: []string{"nodes", "pods", "deployments", "events", "namespaces"}}, SnapshotInterval: 10 * time.Millisecond})
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	if err := s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	h := &harness{s: s, src: src}
	agent := engine.OriginExternalAgent

	h.must(t, agent, "ui.openView", map[string]any{"view": engine.ClusterOverview}, nil)
	snap := waitTable(t, s, engine.ClusterOverview, func(s *engine.TableSnapshot) bool { return s.Synced && len(s.Rows) == 2 })
	var st commands.StateResult
	h.must(t, agent, "ui.state", nil, &st)
	if st.View != engine.ClusterOverview || st.ViewTitle != "Overview" || st.Rows != 2 || st.Sort.Column != "age" || !st.Sort.Desc {
		t.Fatalf("state %+v", st)
	}

	var m engine.ClusterOverviewModel
	h.must(t, agent, "cluster.overview", nil, &m)
	if len(m.Issues) != 2 || m.Issues[0].Source != engine.IssueEvent || m.Issues[0].Message != "Unhealthy: Readiness probe failed" ||
		m.Issues[1].Ref.Kind != "nodes" || m.Nodes != engine.NodesWorker || len(m.Charts) != 3 || m.Charts[0].Capacity != 2 {
		t.Fatalf("cluster.overview %+v", m)
	}

	node := snap.Rows[snap.IndexOf(model.ObjectRef{Context: "test", Kind: "nodes", Name: "worker-2"})].Ref
	h.must(t, agent, "ui.select", map[string]any{"refs": []model.ObjectRef{node}}, nil)
	if u := s.UIState(); u.View != engine.ClusterOverview || u.Drawer == nil || !u.Drawer.Same(node) {
		t.Fatalf("selecting an issue's node: %+v", u)
	}
	h.must(t, agent, "ui.sort", map[string]any{"column": "kind"}, nil)
	if err := h.run(t, agent, "ui.sort", map[string]any{"column": "restarts"}, nil); err == nil {
		t.Fatal("sorted the Issues by a column they do not have")
	}
	h.must(t, agent, "ui.setColumns", map[string]any{"hidden": []string{"age"}}, nil)
	waitTable(t, s, engine.ClusterOverview, func(s *engine.TableSnapshot) bool { return len(hiddenIDs(s)) == 1 && s.Sort.Column == "kind" })
	var csv commands.CSVResult
	h.must(t, agent, "table.exportCsv", nil, &csv)
	if !strings.HasPrefix(csv.CSV, "Name,Kind,Namespace,Message\n") || csv.Rows != 2 {
		t.Fatalf("csv of the Issues %+v", csv)
	}

	// The charts' range and node group: UI state, set by the page's controls and by agents.
	h.must(t, agent, "ui.setOverviewRange", map[string]any{"range": "7d"}, nil)
	h.must(t, agent, "ui.setOverviewNodes", map[string]any{"nodes": "master"}, nil)
	for _, bad := range []struct {
		id     string
		params map[string]any
	}{
		{"ui.setOverviewRange", map[string]any{"range": "90d"}}, {"ui.setOverviewRange", map[string]any{"range": "soon"}},
		{"ui.setOverviewNodes", map[string]any{"nodes": "gpu"}},
	} {
		if err := h.run(t, agent, bad.id, bad.params, nil); err == nil {
			t.Errorf("%s %v was taken", bad.id, bad.params)
		}
	}
	h.must(t, agent, "ui.state", nil, &st)
	if st.Overview != (engine.OverviewChoice{Range: "7d", Nodes: engine.NodesMaster}) {
		t.Fatalf("ui.state's overview %+v", st.Overview)
	}
	m = engine.ClusterOverviewModel{}
	h.must(t, agent, "cluster.overview", map[string]any{"noIssues": true}, &m)
	if m.Range != "7d" || m.Nodes != engine.NodesMaster || len(m.Issues) != 0 || len(m.Charts) != 3 {
		t.Fatalf("cluster.overview after the choice: %+v", m)
	}
	h.must(t, agent, "ui.setOverviewRange", map[string]any{"range": "default"}, nil)
	h.must(t, agent, "ui.setOverviewNodes", nil, nil)
	if o := s.UIState().Overview; o != (engine.OverviewChoice{}) {
		t.Fatalf("back to the defaults: %+v", o)
	}

	h.must(t, agent, "ui.openView", map[string]any{"view": "pods"}, nil)
	if err := h.run(t, agent, "ui.select", map[string]any{"refs": []model.ObjectRef{node}}, nil); err == nil {
		t.Fatal("the Pods view selected a node")
	}
}
