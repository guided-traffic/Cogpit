package engine

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

var secretsGVR = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}

func TestDetailWithEventsNewestFirst(t *testing.T) {
	src := testkit.NewMemorySource()
	pod := testkit.Pod("shop", "web-0")
	src.Set(testkit.PodsGVR, pod)
	base := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	uid := "uid-shop-web-0"
	src.Set(testkit.EventsGVR, event("shop", "e1", "Pod", "web-0", uid, "Scheduled", "Normal", base))
	src.Set(testkit.EventsGVR, event("shop", "e2", "Pod", "web-0", uid, "BackOff", "Warning", base.Add(2*time.Minute)))
	src.Set(testkit.EventsGVR, event("shop", "e3", "Pod", "web-0", "", "Pulled", "Normal", base.Add(time.Minute)))   // matched by name
	src.Set(testkit.EventsGVR, event("shop", "e4", "Pod", "web-1", "", "Pulled", "Normal", base.Add(3*time.Minute))) // another pod
	src.Set(testkit.EventsGVR, event("shop", "e5", "Pod", "web-0", "uid-old", "Killing", "Normal", base))            // an older pod of the same name
	s, _ := newTestSession(t, src, 10*time.Millisecond)
	connectTest(t, s)

	ref := model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}
	m, err := s.DetailWait(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	if m.Title != "Pod: web-0" || m.Missing || m.Loading || m.Status != model.StatusOK || m.Ref.Context != "test" {
		t.Errorf("detail header %+v", m)
	}
	var reasons []string
	for _, e := range m.Events {
		reasons = append(reasons, e.Reason)
	}
	if !slices.Equal(reasons, []string{"BackOff", "Pulled", "Scheduled"}) {
		t.Errorf("events %v, want newest first and only this pod's", reasons)
	}
	if e := m.Events[0]; e.Status != model.StatusWarning || e.Count != 2 || e.Source != "kubelet node-1" || e.LastSeen != base.Add(2*time.Minute).Unix() {
		t.Errorf("event item %+v", e)
	}
	// Properties first (the metadata, no UID), then the kind's sections; events after them.
	if len(m.Sections) != 2 || m.Sections[0].Title != model.TitleProperties || m.Sections[1].Title != "Containers" {
		t.Errorf("sections %+v", m.Sections)
	}
	for _, it := range m.Sections[0].Items {
		if it.Name == "UID" {
			t.Errorf("the drawer shows the UID: %+v", it)
		}
	}
	if !slices.Equal(m.Containers, []string{"app"}) {
		t.Errorf("containers %v", m.Containers)
	}
	if !strings.Contains(m.YAML, "name: web-0") || !strings.Contains(m.YAML, "podIP: 10.244.0.10") {
		t.Errorf("yaml %q", m.YAML)
	}

	missing, err := s.Detail(model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "gone"})
	if err != nil || !missing.Missing {
		t.Errorf("missing object: %+v %v", missing, err)
	}
	if _, err := s.Detail(model.ObjectRef{Context: "other", Kind: "pods", Namespace: "shop", Name: "web-0"}); !errors.Is(err, ErrWrongCluster) {
		t.Errorf("foreign context: %v", err)
	}
	if _, err := s.Detail(model.ObjectRef{Kind: "pods", Name: "web-0"}); err == nil {
		t.Error("namespaced kind without a namespace accepted")
	}
	if _, err := s.Detail(model.ObjectRef{Kind: "widgets", Namespace: "shop", Name: "w"}); !errors.Is(err, ErrUnknownKind) {
		t.Errorf("unknown kind: %v", err)
	}
}

func TestDetailRedactsSecrets(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(secretsGVR, testkit.Object("v1", "Secret", "shop", "db-creds", map[string]any{
		"data": map[string]any{"password": "aHVudGVyMg=="},
		"metadata": map[string]any{"annotations": map[string]any{
			"kubectl.kubernetes.io/last-applied-configuration": `{"data":{"password":"aHVudGVyMg=="}}`,
		}},
	}))
	s, _ := newTestSession(t, src, 10*time.Millisecond)
	connectTest(t, s)
	m, err := s.DetailWait(context.Background(), model.ObjectRef{Kind: "secrets", Namespace: "shop", Name: "db-creds"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(m.YAML, "aHVudGVyMg") || !strings.Contains(m.YAML, model.Redacted) {
		t.Errorf("secret value in the YAML: %s", m.YAML)
	}
	// The cache still holds the value: redaction copies, it does not modify the shared object.
	obj, err := s.Object(context.Background(), model.ObjectRef{Kind: "secrets", Namespace: "shop", Name: "db-creds"})
	if err != nil || obj.Object["data"].(map[string]any)["password"] != "aHVudGVyMg==" {
		t.Errorf("cached secret modified: %v", err)
	}
}

func TestDrawerEmitsDetailEvents(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("shop", "web-0"))
	src.Set(testkit.PodsGVR, testkit.Pod("shop", "web-1"))
	s, _ := newTestSession(t, src, 10*time.Millisecond)
	connectTest(t, s)
	ref := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "shop", Name: "web-0"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &ref })
	rec := record(s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// An event about another pod does not touch the drawer; one about the drawer's pod does.
	src.Set(testkit.EventsGVR, event("shop", "other", "Pod", "web-1", "uid-shop-web-1", "Pulled", "Normal", time.Now()))
	time.Sleep(100 * time.Millisecond)
	before := rec.count(EventDetail, "")
	src.Set(testkit.EventsGVR, event("shop", "mine", "Pod", "web-0", "uid-shop-web-0", "BackOff", "Warning", time.Now()))
	if !testkit.WaitFor(ctx, func() bool { return rec.count(EventDetail, "") > before }) {
		t.Fatal("no detail event for the drawer's event")
	}
	m, err := s.Detail(ref)
	if err != nil || len(m.Events) != 1 || m.Events[0].Reason != "BackOff" {
		t.Errorf("drawer detail %+v %v", m, err)
	}

	before = rec.count(EventDetail, "")
	p := testkit.Pod("shop", "web-0")
	p.SetLabels(map[string]string{"v": "2"})
	src.Set(testkit.PodsGVR, p)
	if !testkit.WaitFor(ctx, func() bool { return rec.count(EventDetail, "") > before }) {
		t.Fatal("no detail event for the object's change")
	}

	s.UpdateUI(func(u *UIState) { u.Drawer = nil })
	s.state.mu.Lock()
	open := s.state.drawer != nil
	s.state.mu.Unlock()
	if open {
		t.Error("drawer watch survived closing the drawer")
	}
}

func TestObjectsAndObject(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("shop", "web-0"))
	src.Set(testkit.PodsGVR, testkit.Pod("infra", "dns-0"))
	s, _ := newTestSession(t, src, 10*time.Millisecond)
	ctx := context.Background()
	if _, err := s.Objects(ctx, "pods", nil); !errors.Is(err, ErrNotConnected) {
		t.Errorf("objects before connect: %v", err)
	}
	connectTest(t, s)
	all, err := s.Objects(ctx, "pods", nil)
	if err != nil || len(all) != 2 {
		t.Fatalf("objects: %d %v", len(all), err)
	}
	shop, err := s.Objects(ctx, "pods", []string{"shop"})
	if err != nil || len(shop) != 1 || shop[0].GetName() != "web-0" {
		t.Errorf("objects in shop: %v %v", shop, err)
	}
	if _, err := s.Object(ctx, model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "nope"}); !errors.Is(err, ErrNotFound) {
		t.Errorf("missing object: %v", err)
	}
	if _, err := s.Objects(ctx, "widgets", nil); !errors.Is(err, ErrUnknownKind) || !strings.Contains(err.Error(), "pods") {
		t.Errorf("unknown kind: %v", err)
	}
	if obj, err := s.Lookup(model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}); err != nil || obj.GetName() != "web-0" {
		t.Errorf("lookup: %v", err)
	}
	ref, _, err := s.ResolveRef(model.ObjectRef{Kind: "namespaces", Namespace: "ignored", Name: "shop"})
	if err != nil || ref.Namespace != "" || ref.Context != "test" {
		t.Errorf("resolve cluster-scoped ref: %+v %v", ref, err)
	}
}

func TestStreamsNeedAConnection(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("shop", "web-0"))
	s, _ := newTestSession(t, src, 10*time.Millisecond)
	connectTest(t, s) // the fake backend has no Conn, like the demo cluster
	ref := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "shop", Name: "web-0"}
	logs := DockTab{ID: s.NewDockID(DockLogs), Type: DockLogs, Context: "test", Ref: &ref, Container: "app"}
	term := DockTab{ID: s.NewDockID(DockTerminal), Type: DockTerminal, Context: "test"}
	s.UpdateUI(func(u *UIState) { u.Dock = append(u.Dock, logs, term) })
	if logs.ID != "logs-1" || term.ID != "terminal-1" {
		t.Errorf("dock ids %s %s", logs.ID, term.ID)
	}
	ctx := context.Background()
	if _, err := s.OpenLogs(ctx, logs.ID); !errors.Is(err, ErrDemoCluster) {
		t.Errorf("logs without a connection: %v", err)
	}
	if _, err := s.OpenTerminal(ctx, term.ID); !errors.Is(err, ErrDemoCluster) {
		t.Errorf("terminal without a connection: %v", err)
	}
	if _, err := s.OpenLogs(ctx, term.ID); err == nil {
		t.Error("a terminal tab opened as logs")
	}
	if _, err := s.OpenLogs(ctx, "logs-99"); err == nil {
		t.Error("an unknown tab opened")
	}
}

func TestClusterConnectorContextsAndDemo(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "no-such-kubeconfig")
	c := &ClusterConnector{KubeconfigPaths: []string{missing}, IncludeDemo: true, DemoOptions: demo.Options{Pods: 20, Services: 3, Namespaces: 2, Nodes: 2, NoChurn: true}}
	list, err := c.Contexts()
	if len(list) == 0 || !list[len(list)-1].Demo || list[len(list)-1].Name != DemoContext {
		t.Errorf("demo entry missing: %+v (err %v)", list, err)
	}
	if _, statErr := os.Stat(missing); statErr == nil {
		t.Error("the connector created a kubeconfig file")
	}

	b, err := c.Connect(context.Background(), DemoContext)
	if errors.Is(err, errDemoUnavailable) {
		t.Skip("the demo package is a stub in this tree")
	}
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if !b.Demo || b.Context != DemoContext || b.Conn != nil || b.CanWrite() || len(b.Profiles) == 0 || b.DemoStats == nil {
		t.Errorf("demo backend %+v", b)
	}
	// Only the kinds the demo serves get a view, as discovery decides on a real cluster, and the
	// Applications made of its deployments (applications.go).
	if len(b.Profiles) != len(demo.Resources())+1 {
		t.Errorf("demo offers %d views for %d served kinds and Applications", len(b.Profiles), len(demo.Resources()))
	}
}
