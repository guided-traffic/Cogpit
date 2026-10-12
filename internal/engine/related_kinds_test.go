package engine

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The related-objects index with Cogpit's own profiles (architecture §4.4): the Persistent Volume
// Claims view's Pods column and the node drawer's Pods table follow the pods, though neither the
// claim nor the node changes.

func builtinProfile(t testing.TB, key string) *model.KindProfile {
	t.Helper()
	p, ok := kinds.ByKey(key)
	if !ok {
		t.Fatalf("no built-in profile %s", key)
	}
	return p
}

// podOnNode is a running pod on node that mounts the claims.
func podOnNode(ns, name, node string, claims ...string) *unstructured.Unstructured {
	p := testkit.Pod(ns, name)
	_ = unstructured.SetNestedField(p.Object, node, "spec", "nodeName")
	vols := []any{}
	for _, c := range claims {
		vols = append(vols, map[string]any{"name": c, "persistentVolumeClaim": map[string]any{"claimName": c}})
	}
	_ = unstructured.SetNestedSlice(p.Object, vols, "spec", "volumes")
	return p
}

// nodePods returns the names in the Pods table of a drawer model, nil without one.
func nodePods(m *DetailModel) []string {
	for _, sec := range m.Sections {
		if sec.Title == "Pods" && sec.Table != nil {
			names := make([]string, 0, len(sec.Table.Rows))
			for _, r := range sec.Table.Rows {
				names = append(names, r[0])
			}
			return names
		}
	}
	return nil
}

func TestClaimPodsColumnFollowsPods(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(claimsGVR, testkit.Object("v1", "PersistentVolumeClaim", "apps", "data", map[string]any{"status": map[string]any{"phase": "Bound"}}))
	src.Set(testkit.PodsGVR, podOnNode("apps", "db-0", "node-1", "data"))
	src.Set(testkit.PodsGVR, podOnNode("other", "db-0", "node-1", "data")) // same claim name, other namespace
	s, fc := newTestSession(t, src, 5*time.Millisecond)
	pvc := builtinProfile(t, "persistentvolumeclaims")
	fc.profiles = append(testProfiles(), pvc)
	connectTest(t, s)
	s.UpdateUI(func(u *UIState) { u.View = pvc.Key })

	col := pvc.ColumnIndex("pods")
	pods := func(ts *TableSnapshot) model.Cell {
		if len(ts.Rows) != 1 {
			return model.Cell{Text: "?"}
		}
		return ts.Rows[0].Cells[col]
	}
	link := func(name string) *model.ObjectRef {
		return &model.ObjectRef{Kind: "pods", Namespace: "apps", Name: name}
	}
	sameLink := func(a, b *model.ObjectRef) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

	waitTable(t, s, pvc.Key, func(ts *TableSnapshot) bool {
		c := pods(ts)
		return ts.Synced && c.Text == "db-0" && sameLink(c.Link, link("db-0"))
	})
	src.Set(testkit.PodsGVR, podOnNode("apps", "db-1", "node-2", "data"))
	waitTable(t, s, pvc.Key, func(ts *TableSnapshot) bool { c := pods(ts); return c.Text == "db-0, db-1" && c.Link == nil })
	src.Delete(testkit.PodsGVR, "apps", "db-0")
	waitTable(t, s, pvc.Key, func(ts *TableSnapshot) bool {
		c := pods(ts)
		return c.Text == "db-1" && sameLink(c.Link, link("db-1"))
	})

	m, err := s.DetailWait(context.Background(), model.ObjectRef{Kind: pvc.Key, Namespace: "apps", Name: "data"})
	if err != nil {
		t.Fatal(err)
	}
	var chips []model.Chip
	for _, sec := range m.Sections {
		for _, it := range sec.Items {
			if it.Name == "Pods" {
				chips = it.Chips
			}
		}
	}
	if len(chips) != 1 || chips[0].Text != "db-1" || !sameLink(chips[0].Link, link("db-1")) {
		t.Errorf("drawer's Pods %+v", chips)
	}
}

// TestRelatedRowsRebuiltWhenTheKindSyncs: a row built while a related kind had delivered its
// initial list but did not report synced yet is rebuilt once it does, though no event follows. The
// informer counts the initial list's last event before it reports synced; the generation rows were
// cached at did not change between the two, and the claims view kept an empty Pods column in a
// synced snapshot (TestClaimPodsColumnFollowsPods failed on slow machines). A second consumer of
// the pods holds the informer on that last event to make the window deterministic.
func TestRelatedRowsRebuiltWhenTheKindSyncs(t *testing.T) {
	mem := testkit.NewMemorySource()
	mem.Set(claimsGVR, testkit.Object("v1", "PersistentVolumeClaim", "apps", "data", map[string]any{"status": map[string]any{"phase": "Bound"}}))
	mem.Set(testkit.PodsGVR, podOnNode("apps", "db-0", "node-1", "data"))
	src := testkit.NewGatedSource(mem, testkit.PodsGVR)
	c := cache.New(src, cache.Options{WarmPeriod: time.Minute})
	defer c.Close()
	pvc := builtinProfile(t, "persistentvolumeclaims")
	b := &Backend{Context: "test", Cache: c, Profiles: append(testProfiles(), pvc)}

	rs := newRelatedSet(b, pvc.Needs, nil) // subscribes to the pods first
	defer rs.release()
	pods := b.Acquire(testPods)
	defer pods.Release()
	held, resume := make(chan struct{}), make(chan struct{})
	var resumeOnce sync.Once
	release := func() { resumeOnce.Do(func() { close(resume) }) }
	defer release()
	var holdOnce sync.Once
	unsub := pods.Subscribe(func(ev cache.Event) {
		if ev.Object.GetName() == "db-0" {
			holdOnce.Do(func() { close(held); <-resume })
		}
	})
	defer unsub()
	src.Open()
	select {
	case <-held:
	case <-time.After(10 * time.Second):
		t.Fatal("the pods' initial list never arrived")
	}

	claims := b.Acquire(pvc)
	defer claims.Release()
	waitFor(t, "the claims", func() bool { return claims.HasSynced() })
	col := pvc.ColumnIndex("pods")
	cell := func(snap *TableSnapshot) string {
		if len(snap.Rows) != 1 {
			return "?"
		}
		return snap.Rows[0].Cells[col].Text
	}
	rc := &rowCache{}
	rel, gen := rs.view()
	if rs.synced() {
		t.Fatal("the pods report synced while their handler still runs")
	}
	if got := cell(rc.build(pvc, b.Context, KindList(claims, pvc), viewParams{}, rel, gen)); got != "" {
		t.Fatalf("Pods column before the pods synced: %q, want empty", got)
	}

	release()
	waitFor(t, "the pods to sync", rs.synced)
	rel, synced := rs.view()
	if synced == gen {
		t.Errorf("the related generation stayed %d when the pods synced", gen)
	}
	if got := cell(rc.build(pvc, b.Context, KindList(claims, pvc), viewParams{}, rel, synced)); got != "db-0" {
		t.Errorf("Pods column after the pods synced: %q, want db-0", got)
	}
}

func TestNodeDrawerPodsFollowPods(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-1", nil))
	src.Set(testkit.PodsGVR, podOnNode("apps", "web-0", "node-1"))
	src.Set(testkit.PodsGVR, podOnNode("apps", "api-0", "node-2"))
	s, fc := newTestSession(t, src, 5*time.Millisecond)
	nodes := builtinProfile(t, "nodes")
	fc.profiles = append(testProfiles(), nodes)
	connectTest(t, s)
	ref := model.ObjectRef{Context: "test", Kind: nodes.Key, Name: "node-1"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &ref })
	rec := record(s)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	table := func() []string {
		m, err := s.Detail(ref)
		if err != nil {
			t.Fatal(err)
		}
		return nodePods(m)
	}
	waitPods := func(want ...string) {
		t.Helper()
		if !testkit.WaitFor(ctx, func() bool { return slices.Equal(table(), want) }) {
			t.Fatalf("node's Pods %q, want %q", table(), want)
		}
	}
	waitPods("web-0")

	// A pod scheduled to the node reaches the drawer: an EventDetail, then the row.
	before := rec.count(EventDetail, "")
	src.Set(testkit.PodsGVR, podOnNode("jobs", "batch-0", "node-1"))
	if !testkit.WaitFor(ctx, func() bool { return rec.count(EventDetail, "") > before }) {
		t.Fatal("no detail event for a pod scheduled to the node")
	}
	waitPods("batch-0", "web-0")
	src.Delete(testkit.PodsGVR, "apps", "web-0")
	waitPods("batch-0")

	// The status cell follows the pod.
	failing := podOnNode("jobs", "batch-0", "node-1")
	_ = unstructured.SetNestedField(failing.Object, "Failed", "status", "phase")
	_ = unstructured.SetNestedSlice(failing.Object, []any{map[string]any{
		"name": "app", "ready": false, "restartCount": int64(0),
		"state": map[string]any{"terminated": map[string]any{"exitCode": int64(1), "reason": "Error"}},
	}}, "status", "containerStatuses")
	src.Set(testkit.PodsGVR, failing)
	if !testkit.WaitFor(ctx, func() bool {
		m, _ := s.Detail(ref)
		for _, sec := range m.Sections {
			if sec.Title == "Pods" && sec.Table != nil && len(sec.Table.Cells) == 1 {
				c := sec.Table.Cells[0][len(sec.Table.Columns)-1]
				return c.Text == "Error" && c.Status == model.StatusError
			}
		}
		return false
	}) {
		t.Errorf("the status cell did not follow the pod")
	}
}
