package engine

import (
	"context"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The pods' CPU and Memory for a user who may list pods in some namespaces only (D-39): the Pods
// list, a node drawer's Pods table and agents' metrics.get read the pods' metrics with one list
// per namespace the Pods view reads, never in all namespaces, which the Metrics API refuses such a
// user as RBAC refuses the pods; a user who may list pods everywhere keeps the one list.

// nsMetricsConnector connects "test" to a cache over src, with access reviews over rev for the
// context's namespace, the given profiles and a metrics hub over prov with short intervals.
type nsMetricsConnector struct {
	src       cache.Source
	rev       *fakeReviewer
	prov      metrics.Provider
	profiles  []*model.KindProfile
	namespace string
}

func (c nsMetricsConnector) Contexts() ([]ContextInfo, error) {
	return []ContextInfo{{Name: "test", Current: true}}, nil
}

func (c nsMetricsConnector) Connect(context.Context, string) (*Backend, error) {
	ch := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	a := newAccessReviews(c.rev, c.namespace)
	hub := metrics.NewHub(metrics.Fixed(c.prov), metrics.Options{
		ListInterval: 40 * time.Millisecond, ObjectInterval: 40 * time.Millisecond, TTL: 20 * time.Millisecond,
	})
	return &Backend{
		Context: "test", Namespace: c.namespace, Cache: ch, Profiles: c.profiles, Access: a, Metrics: hub,
		Close: func() { a.stop(); ch.Close() },
	}, nil
}

// limitedMetricsSession connects a user who may read the nodes (a cluster-wide binding) and, with
// rules, pods: the drawer fixture's pods in apps and jobs on node-1 and node-2, and one in infra,
// measured by the provider. accessible are the cluster's accessible namespaces.
func limitedMetricsSession(t *testing.T, rules map[string][]cluster.AccessRule, refuse bool, accessible ...string) (*Session, *podTableProvider) {
	t.Helper()
	mem := testkit.NewMemorySource()
	mem.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-1", nil))
	mem.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-2", nil))
	mem.Set(testkit.PodsGVR, podOnNode("apps", "web-0", "node-1"))
	mem.Set(testkit.PodsGVR, podOnNode("apps", "api-0", "node-1"))
	mem.Set(testkit.PodsGVR, podOnNode("jobs", "batch-0", "node-1"))
	mem.Set(testkit.PodsGVR, podOnNode("apps", "far-0", "node-2"))
	mem.Set(testkit.PodsGVR, podOnNode("infra", "db-0", "node-1"))
	src := &testkit.NamespacedOnlySource{Inner: mem}
	if refuse {
		src.GVRs, src.Denied = []schema.GroupVersionResource{testkit.PodsGVR}, []string{"infra"}
	}
	usage := map[string]model.Usage{"infra/db-0": {CPUMilli: 77, MemoryBytes: 7 << 20}}
	for k, v := range drawerPodUsage {
		usage[k] = v
	}
	prov := &podTableProvider{usage: usage}
	s := New(Options{
		Connector: nsMetricsConnector{src: src, rev: newFakeReviewer(rules), prov: prov, namespace: "apps",
			profiles: []*model.KindProfile{builtinProfile(t, "pods"), builtinProfile(t, "nodes"), testEvents}},
		ClusterSettings:  NewClusterSettingsStore(filepath.Join(t.TempDir(), "Cogpit")),
		SnapshotInterval: 5 * time.Millisecond,
	})
	t.Cleanup(s.Close)
	if len(accessible) > 0 {
		if _, err := s.ChangeClusterSettings("test", func(c *ClusterSettings) error { c.AccessibleNamespaces = accessible; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	connectTest(t, s)
	return s, prov
}

// cpuOf is the CPU cell of a pod's row in a snapshot; empty without the row or a measurement.
func cpuOf(snap *TableSnapshot, namespace, name string) string {
	ci := slices.IndexFunc(snap.Columns, func(c model.Column) bool { return c.ID == model.ColumnCPU })
	for i := range snap.Rows {
		if r := &snap.Rows[i]; r.Ref.Namespace == namespace && r.Ref.Name == name {
			return cellAt(r, ci).Text
		}
	}
	return ""
}

func TestPodMetricsPerNamespaceOfThePlan(t *testing.T) {
	pods := rule("get,list,watch", "", "pods")
	nodes := rule("get,list,watch", "", "nodes")
	s, prov := limitedMetricsSession(t, map[string][]cluster.AccessRule{"": {nodes}, "apps": {pods}, "jobs": {pods}}, true, "apps", "jobs")

	// The Pods list: apps and jobs, each with its own list of the pods' metrics.
	waitFor(t, "the Pods list's CPU in apps and jobs", func() bool {
		snap := s.Table("pods")
		return snap != nil && snap.Synced && slices.Equal(snap.Watched, []string{"apps", "jobs"}) &&
			cpuOf(snap, "apps", "web-0") != "" && cpuOf(snap, "jobs", "batch-0") != ""
	})
	if got := prov.listed(); slices.Contains(got, "") || !slices.Contains(got, "apps") || !slices.Contains(got, "jobs") {
		t.Errorf("the pod lists asked for %q, want apps and jobs, never all namespaces", got)
	}

	// A node's drawer shares those lists for its Pods table: nothing more is polled for it than
	// the node's own chart.
	ref := model.ObjectRef{Context: "test", Kind: "nodes", Name: "node-1"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &ref })
	waitFor(t, "the node's Pods table with web-0's CPU", func() bool {
		tbl := podsTable(t, s, ref)
		if tbl == nil {
			return false
		}
		for r, cells := range tbl.Cells {
			if cells[0].Text == "web-0" {
				return tbl.Rows[r][tbl.ColumnIndex(model.ColumnCPU)] != ""
			}
		}
		return false
	})
	hub := s.Backend().Metrics
	if n := hub.Subscribed(); n != 3 {
		t.Errorf("%d scopes polled with the Pods list and a node's drawer, want 3 (apps, jobs, node-1)", n)
	}

	// Agents: metrics.get of every pod reads the same namespaces.
	res, err := s.MetricsGet(context.Background(), "", "pods", "", "", 0)
	if err != nil || res.Status != metrics.StatusAvailable {
		t.Fatalf("metrics.get: %+v, %v", res, err)
	}
	var got []string
	for _, p := range res.Pods {
		got = append(got, p.Namespace+"/"+p.Name)
	}
	if want := []string{"apps/far-0", "apps/web-0", "jobs/batch-0"}; !slices.Equal(got, want) {
		t.Errorf("metrics.get's pods %q, want %q", got, want)
	}
	if got := prov.listed(); slices.Contains(got, "") || slices.Contains(got, "infra") {
		t.Errorf("the pod lists asked for %q, never all namespaces or infra", got)
	}
}

// A cluster-admin's Pods list keeps the one list of every pod's metrics, shared with a node's
// drawer and metrics.get: nothing per namespace.
func TestPodMetricsClusterWideForAUserWhoMayList(t *testing.T) {
	s, prov := limitedMetricsSession(t, map[string][]cluster.AccessRule{"": {rule("*", "*", "*")}}, false)
	waitFor(t, "the Pods list's CPU", func() bool {
		snap := s.Table("pods")
		return snap != nil && snap.Synced && cpuOf(snap, "infra", "db-0") != "" && cpuOf(snap, "apps", "web-0") != ""
	})
	ref := model.ObjectRef{Context: "test", Kind: "nodes", Name: "node-1"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &ref })
	waitFor(t, "the node's Pods table", func() bool { tbl := podsTable(t, s, ref); return tbl != nil && len(tbl.Rows) == 4 })
	if n := s.Backend().Metrics.Subscribed(); n != 2 {
		t.Errorf("%d scopes polled, want 2 (every pod, node-1)", n)
	}
	if _, err := s.MetricsGet(context.Background(), "", "pods", "", "", 0); err != nil {
		t.Fatal(err)
	}
	if got := prov.listed(); slices.ContainsFunc(got, func(ns string) bool { return ns != "" }) {
		t.Errorf("the pod lists asked for %q, want every namespace at once", got)
	}
}
