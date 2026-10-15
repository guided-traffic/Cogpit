package commands_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// stackWriter is the cluster side of Cogpit Metrics: the fake installer's releases, and the
// namespace Cogpit creates and deletes kept in the source, so the cache sees them.
type stackWriter struct {
	*fakeInstaller
	nmu    sync.Mutex
	nsOps  []string
	uidSeq int
}

func (w *stackWriter) CreateObject(_ context.Context, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured, _ bool) (*unstructured.Unstructured, error) {
	w.nmu.Lock()
	w.uidSeq++
	w.nsOps = append(w.nsOps, "create "+gvr.Resource+" "+obj.GetName())
	o := obj.DeepCopy()
	o.SetUID(types.UID("ns-uid-" + string(rune('0'+w.uidSeq))))
	w.nmu.Unlock()
	w.src.Set(testkit.NamespacesGVR, o)
	return o, nil
}

func (w *stackWriter) DeleteObject(_ context.Context, gvr schema.GroupVersionResource, namespace, name, uid string) error {
	w.nmu.Lock()
	w.nsOps = append(w.nsOps, "delete "+gvr.Resource+" "+name+" "+uid)
	w.nmu.Unlock()
	w.src.Delete(testkit.NamespacesGVR, "", name)
	return nil
}

func (w *stackWriter) namespaceOps() []string {
	w.nmu.Lock()
	defer w.nmu.Unlock()
	return slices.Clone(w.nsOps)
}

// stackConnector connects "test" with the test profiles, the releases and the StorageClasses.
type stackConnector struct {
	src     *testkit.MemorySource
	actions engine.Actions
	access  engine.Access
}

func (c *stackConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: "test", Current: true}}, nil
}

func (c *stackConnector) Connect(_ context.Context, name string) (*engine.Backend, error) {
	ch := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	classes := &model.KindProfile{Key: "storageclasses", Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses",
		Kind: "StorageClass", Title: "Storage Classes", Columns: []model.Column{{ID: "name", Title: "Name", Type: model.ColumnText}}}
	profiles := append(append(testProfiles(), helm.NewReleaseProfile(), classes), kinds.Local()...)
	return &engine.Backend{Context: name, Namespace: "default", Cache: ch, Close: ch.Close, Profiles: profiles,
		Conn: &cluster.Conn{}, Actions: c.actions, Access: c.access}, nil
}

// noClusterRoles is access reviews that know the user may not create cluster roles.
type noClusterRoles struct{}

func (noClusterRoles) Allowed(_ string, v engine.Verb) (bool, bool) {
	if v.Resource == "clusterroles" || v.Resource == "clusterrolebindings" {
		return false, true
	}
	return true, true
}

type stackHarness struct {
	s      *engine.Session
	src    *testkit.MemorySource
	w      *stackWriter
	mu     sync.Mutex
	asked  []engine.ConfirmRequest
	answer bool
}

var storageClassesGVR = schema.GroupVersionResource{Group: "storage.k8s.io", Version: "v1", Resource: "storageclasses"}

func newStackHarness(t *testing.T, access ...engine.Access) *stackHarness {
	t.Helper()
	src := testkit.NewMemorySource()
	h := &stackHarness{src: src, answer: true, w: &stackWriter{fakeInstaller: newFakeInstaller(t, src)}}
	var acc engine.Access
	if len(access) > 0 {
		acc = access[0]
	}
	h.s = engine.New(engine.Options{
		Connector: &stackConnector{src: src, actions: h.w, access: acc}, SnapshotInterval: 10 * time.Millisecond,
		Audit: &syncBuffer{}, Helm: engine.NewHelmOps(engine.NewMemoryHelmJournal()),
		Logger: slog.New(slog.NewTextHandler(&syncBuffer{}, nil)),
		Confirmer: engine.ConfirmFunc(func(_ context.Context, req engine.ConfirmRequest) (bool, error) {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.asked = append(h.asked, req)
			return h.answer, nil
		}),
	})
	t.Cleanup(h.s.Close)
	if err := commands.RegisterAll(h.s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := h.s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *stackHarness) dispatch(origin engine.Origin, id string, params any) (any, error) {
	raw, _ := json.Marshal(params)
	return h.s.Dispatch(context.Background(), engine.Invocation{ID: id, Params: raw, Origin: origin})
}

func (h *stackHarness) confirmations() []engine.ConfirmRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.asked)
}

func (h *stackHarness) state(t *testing.T) engine.MetricsStackState {
	t.Helper()
	return h.view(t, struct{}{}).MetricsStackState
}

func (h *stackHarness) view(t *testing.T, params any) commands.MetricsStackView {
	t.Helper()
	res, err := h.dispatch(engine.OriginExternalAgent, "cluster.metricsStack", params)
	if err != nil {
		t.Fatal(err)
	}
	return res.(commands.MetricsStackView)
}

func installedOf(st engine.MetricsStackState) []string {
	var out []string
	for _, c := range st.Components {
		if c.Installed {
			out = append(out, c.ID)
		}
	}
	return out
}

// Cluster Settings › Cogpit Metrics (D-61): the plan of the switches and its sentence are read
// first (the window's own dialog before an install); Apply installs the switched-on components from
// the bundled charts as one release each in cogpit-metrics, creating the namespace with Cogpit's
// labels; an agent's Apply always asks; switching every component off uninstalls them (a
// destructive confirmation, for the window too) and deletes the namespace Cogpit created; an Apply
// that changes nothing asks nobody.
func TestMetricsStackApply(t *testing.T) {
	h := newStackHarness(t)
	class := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "storage.k8s.io/v1", "kind": "StorageClass", "reclaimPolicy": "Delete",
		"metadata": map[string]any{"name": "standard", "annotations": map[string]any{"storageclass.kubernetes.io/is-default-class": "true"}}}}
	h.src.Set(storageClassesGVR, class)

	st := h.state(t)
	if len(st.Components) != 3 || len(installedOf(st)) != 0 || st.NamespaceExists || !st.Known || st.Storage.StorageClass != "standard" ||
		st.Chart != "prometheus "+helm.MetricsChartVersion {
		t.Fatalf("a cluster without the stack: %+v", st)
	}

	// The plan of Prometheus and kube-state-metrics, read without changing anything.
	switches := map[string]any{"prometheus": true, "kubeStateMetrics": true}
	plan := h.view(t, switches).Plan
	if plan == nil || !slices.Equal(plan.Install, []string{"cogpit-prometheus", "cogpit-kube-state-metrics"}) || !plan.CreateNamespace || plan.Destructive {
		t.Fatalf("plan %+v", plan)
	}
	if d := plan.Detail; !strings.Contains(d, "chart prometheus "+helm.MetricsChartVersion+" that Cogpit bundles") ||
		!strings.Contains(d, "StorageClass standard") || !strings.Contains(d, "Secrets left out") || !strings.Contains(d, "creating the namespace") {
		t.Errorf("the plan's sentence: %s", d)
	}
	if n := len(h.w.recordedInstalls()); n != 0 {
		t.Fatalf("reading the plan installed %d", n)
	}

	// The window's Apply: an install, which the engine does not confirm for the window (its own
	// dialog showed the sentence).
	res, err := h.dispatch(engine.OriginGUI, "cluster.setMetricsStack", switches)
	if err != nil {
		t.Fatal(err)
	}
	if asked := h.confirmations(); len(asked) != 0 {
		t.Fatalf("confirmations %+v", asked)
	}
	r := res.(commands.MetricsStackResult)
	if !slices.Equal(r.Installed, []string{"cogpit-prometheus", "cogpit-kube-state-metrics"}) || !r.NamespaceCreated ||
		!slices.Equal(installedOf(r.State), []string{"prometheus", "kube-state-metrics"}) || !r.State.NamespaceCogpits {
		t.Errorf("result %+v", r)
	}
	installs := h.w.recordedInstalls()
	if len(installs) != 2 || installs[0].Namespace != helm.MetricsNamespace || installs[0].Chart.Name != "prometheus" ||
		installs[1].Chart.Name != "kube-state-metrics" || installs[0].CreateNamespace {
		t.Fatalf("installs %+v", installs)
	}
	if server, _ := installs[0].Values["server"].(map[string]any); server["persistentVolume"].(map[string]any)["enabled"] != true {
		t.Errorf("Prometheus's values with a default StorageClass: %+v", installs[0].Values)
	}
	if ops := h.w.namespaceOps(); !slices.Equal(ops, []string{"create namespaces cogpit-metrics"}) {
		t.Errorf("namespace writes %v", ops)
	}
	if pending, _ := h.s.PendingHelmOps(false); len(pending) != 0 {
		t.Errorf("journal left %+v", pending)
	}

	// An agent's Apply always waits for the user, also where agents may write; the confirmation
	// names the release and says what it means.
	if _, err := h.dispatch(engine.OriginExternalAgent, "cluster.setMetricsStack", map[string]any{"nodeExporter": true}); err != nil {
		t.Fatal(err)
	}
	asked := h.confirmations()
	if len(asked) != 1 || asked[0].Origin != engine.OriginExternalAgent || !asked[0].AgentsAsk || asked[0].Danger != engine.DangerMutating ||
		!strings.Contains(asked[0].Detail, "port 9100") || len(asked[0].Objects) != 1 || asked[0].Objects[0].Name != "cogpit-node-exporter" {
		t.Errorf("the agent's confirmation %+v", asked)
	}

	// Nothing to apply: refused before anyone is asked.
	if _, err := h.dispatch(engine.OriginExternalAgent, "cluster.setMetricsStack", map[string]any{"prometheus": true}); err == nil ||
		!strings.Contains(err.Error(), "nothing to apply") || len(h.confirmations()) != 1 {
		t.Errorf("an Apply that changes nothing: %v, %d confirmations", err, len(h.confirmations()))
	}
	if p := h.view(t, map[string]any{"prometheus": true}).Plan; p == nil || !p.Empty {
		t.Errorf("the plan of the installed state: %+v", p)
	}

	// Everything off: destructive, every release uninstalled, the namespace deleted by its UID.
	res, err = h.dispatch(engine.OriginGUI, "cluster.setMetricsStack", map[string]any{"prometheus": false, "kubeStateMetrics": false, "nodeExporter": false})
	if err != nil {
		t.Fatal(err)
	}
	asked = h.confirmations()
	if last := asked[len(asked)-1]; last.Danger != engine.DangerDestructive || !strings.Contains(last.Detail, "the history Prometheus collected") ||
		!strings.Contains(last.Detail, "Deletes namespace cogpit-metrics") {
		t.Errorf("the removal's confirmation %+v", last)
	}
	r = res.(commands.MetricsStackResult)
	if len(r.Removed) != 3 || !r.NamespaceDeleted || len(installedOf(r.State)) != 0 {
		t.Errorf("removal %+v", r)
	}
	if ops := h.w.namespaceOps(); len(ops) != 2 || !strings.HasPrefix(ops[1], "delete namespaces cogpit-metrics ns-uid-") {
		t.Errorf("namespace writes %v", ops)
	}
}

// A namespace cogpit-metrics the user made is used, never deleted; a refused confirmation does
// nothing; without a default StorageClass Prometheus keeps its history in an emptyDir.
func TestMetricsStackLeavesTheUsersNamespace(t *testing.T) {
	h := newStackHarness(t)
	ns := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Namespace",
		"metadata": map[string]any{"name": helm.MetricsNamespace, "uid": "users-ns", "labels": map[string]any{"team": "ops"}}}}
	h.src.Set(testkit.NamespacesGVR, ns)
	h.answer = false
	if _, err := h.dispatch(engine.OriginExternalAgent, "cluster.setMetricsStack", map[string]any{"prometheus": true}); !errors.Is(err, engine.ErrDeclined) {
		t.Fatalf("declined: %v", err)
	}
	if n := len(h.w.recordedInstalls()); n != 0 {
		t.Fatalf("%d installs after a refusal", n)
	}
	h.answer = true
	if _, err := h.dispatch(engine.OriginGUI, "cluster.setMetricsStack", map[string]any{"prometheus": true}); err != nil {
		t.Fatal(err)
	}
	installs := h.w.recordedInstalls()
	if server, _ := installs[0].Values["server"].(map[string]any); server["persistentVolume"].(map[string]any)["enabled"] != false || server["emptyDir"] == nil {
		t.Errorf("Prometheus's values without a default StorageClass: %+v", installs[0].Values)
	}
	if _, err := h.dispatch(engine.OriginGUI, "cluster.setMetricsStack", map[string]any{"prometheus": false}); err != nil {
		t.Fatal(err)
	}
	if ops := h.w.namespaceOps(); len(ops) != 0 {
		t.Errorf("the user's namespace was written: %v", ops)
	}
	if st := h.state(t); !st.NamespaceExists || st.NamespaceCogpits || len(installedOf(st)) != 0 {
		t.Errorf("after the removal: %+v", st)
	}
}

// Where the access reviews know the user may not create cluster roles, installing Prometheus or
// kube-state-metrics is greyed with the reason; node-exporter, whose chart creates none, is not.
func TestMetricsStackNotPermitted(t *testing.T) {
	h := newStackHarness(t, noClusterRoles{})
	c, _ := h.s.Registry().Get("cluster.setMetricsStack")
	if err := c.Enabled(h.s, json.RawMessage(`{"prometheus":true}`)); !errors.Is(err, engine.ErrNotPermitted) ||
		!strings.Contains(err.Error(), "create clusterroles.rbac.authorization.k8s.io") {
		t.Errorf("Prometheus without the right to create cluster roles: %v", err)
	}
	if err := c.Enabled(h.s, json.RawMessage(`{"nodeExporter":true}`)); err != nil {
		t.Errorf("node-exporter: %v", err)
	}
	if err := c.Enabled(h.s, json.RawMessage(`{}`)); err == nil {
		t.Error("an Apply without a switch is enabled")
	}
}
