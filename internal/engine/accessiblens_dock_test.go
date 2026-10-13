package engine

import (
	"context"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// dockConnector connects "test" over any source, with access reviews over a fake reviewer for the
// context's namespace "default", the given profiles (the local kinds added, as the cluster
// connector adds them) and actions; its connection is a stand-in that is never dialled.
type dockConnector struct {
	src      cache.Source
	rev      *fakeReviewer
	profiles []*model.KindProfile
	actions  Actions
}

func (c dockConnector) Contexts() ([]ContextInfo, error) {
	return []ContextInfo{{Name: "test", Current: true}}, nil
}

func (c dockConnector) Connect(context.Context, string) (*Backend, error) {
	ch := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	a := newAccessReviews(c.rev, "default")
	return &Backend{
		Context: "test", Namespace: "default", Cache: ch, Profiles: append(slices.Clone(c.profiles), kinds.Local()...),
		Actions: c.actions, Conn: &cluster.Conn{}, Access: a, Close: func() { a.stop(); ch.Close() },
	}, nil
}

// dockActions is the backend's writer: it dials port forwards to a tunnel that carries nothing.
type dockActions struct {
	mu    sync.Mutex
	dials []string
}

func (*dockActions) RestartDeployment(context.Context, string, string, string) error { return nil }
func (*dockActions) DeletePod(context.Context, string, string, string) error         { return nil }

func (d *dockActions) DialPortForward(_ context.Context, namespace, pod string) (cluster.PortTunnel, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.dials = append(d.dials, namespace+"/"+pod)
	return &idleTunnel{done: make(chan struct{})}, nil
}

func (d *dockActions) dialed() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return slices.Clone(d.dials)
}

type idleTunnel struct {
	once sync.Once
	done chan struct{}
}

func (t *idleTunnel) Forward(conn net.Conn, _ uint16) error { return conn.Close() }
func (t *idleTunnel) Done() <-chan struct{}                 { return t.done }
func (t *idleTunnel) Close() error                          { t.once.Do(func() { close(t.done) }); return nil }

// shopOnly is a cluster whose user may read pods, services, deployments and replica sets in the
// namespace shop only: not in all namespaces, not in the context's namespace default. No
// accessible namespaces are set, so the namespaces the plan names (the context's own) miss shop.
// It holds the Deployment web with its replica set and running pod web-1-a (container app).
func shopOnly(t *testing.T) (*testkit.NamespacedOnlySource, *fakeReviewer) {
	t.Helper()
	mem := testkit.NewMemorySource()
	mem.Set(testkit.DeploymentsGVR, testkit.Object("apps/v1", "Deployment", "shop", "web", nil))
	mem.Set(replicaSetsGVR, owned(testkit.Object("apps/v1", "ReplicaSet", "shop", "web-1", nil), "apps/v1", "Deployment", "web", "uid-shop-web"))
	mem.Set(testkit.PodsGVR, owned(logPod("shop", "web-1-a", "app"), "apps/v1", "ReplicaSet", "web-1", "uid-shop-web-1"))
	src := &testkit.NamespacedOnlySource{Inner: mem, Denied: []string{"default"}, GVRs: []schema.GroupVersionResource{
		testkit.PodsGVR, testkit.ServicesGVR, testkit.DeploymentsGVR, replicaSetsGVR,
	}}
	rev := newFakeReviewer(map[string][]cluster.AccessRule{"shop": {
		rule("get,list,watch", "", "pods,services"), rule("get,list,watch", "apps", "deployments,replicasets"),
	}})
	return src, rev
}

func shopSession(t *testing.T, src cache.Source, rev *fakeReviewer, actions Actions) *Session {
	t.Helper()
	profiles := testProfiles()
	rs, ok := kinds.ByKey("replicasets")
	if !ok {
		t.Fatal("no replica sets profile")
	}
	s := New(Options{Connector: dockConnector{src: src, rev: rev, profiles: append(profiles, rs), actions: actions},
		SnapshotInterval: 5 * time.Millisecond})
	t.Cleanup(s.Close)
	connectTest(t, s)
	return s
}

// A port forward started while the access reviews are still out (a permanent one starts at
// connect) waits for them instead of taking pods in all namespaces, which the API server refuses
// and which never syncs; once they answer that the user may not list pods everywhere, it reads its
// target's namespace alone, though the namespaces the plan names miss it, and connects.
func TestPortForwardWaitsForThePlanAndReadsItsNamespace(t *testing.T) {
	src, rev := shopOnly(t)
	rev.set(func(f *fakeReviewer) { f.gate = make(chan struct{}) })
	actions := &dockActions{}
	s := shopSession(t, src, rev, actions)

	type outcome struct {
		f   Forward
		err error
	}
	done := make(chan outcome, 1)
	go func() {
		f, err := s.StartForward(context.Background(), ForwardSpec{Target: model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-1-a"}, Port: 8080})
		done <- outcome{f, err}
	}()
	time.Sleep(50 * time.Millisecond)
	if calls := podCalls(src); len(calls) != 0 {
		t.Errorf("pods read before the access reviews answered: %v", calls)
	}
	rev.set(func(f *fakeReviewer) { close(f.gate); f.gate = nil })

	var o outcome
	select {
	case o = <-done:
	case <-time.After(10 * time.Second):
		t.Fatalf("the forward never started; pod calls %v", podCalls(src))
	}
	if o.err != nil || o.f.Phase != ForwardActive || !slices.Equal(actions.dialed(), []string{"shop/web-1-a"}) {
		t.Fatalf("forward %+v, %v; dials %v; pod calls %v", o.f, o.err, actions.dialed(), podCalls(src))
	}
	if calls := podCalls(src); calls["list pods"] != 0 || calls["list pods in shop"] != 1 {
		t.Errorf("pod calls %v, want shop's alone", calls)
	}
	if _, err := s.StopForward(o.f.Ref); err != nil {
		t.Error(err)
	}

	// A namespace the user may not list pods in either: the forward fails with the refusal instead
	// of waiting for a pod it will never see, and reading one object there says why too.
	f, err := s.StartForward(context.Background(), ForwardSpec{Target: model.ObjectRef{Kind: "pods", Namespace: "default", Name: "db-0"}, Port: 5432})
	if !apierrors.IsForbidden(err) || f.Phase != ForwardFailed {
		t.Errorf("a forward in a refused namespace: %+v, %v", f, err)
	}
	if _, err := s.Object(context.Background(), model.ObjectRef{Kind: "pods", Namespace: "default", Name: "db-0"}); !apierrors.IsForbidden(err) {
		t.Errorf("an object in a refused namespace: %v", err)
	}
}

// A Deployment's logs tab opened while the access reviews are still out waits for them, then
// reads the Deployment, its replica sets and its pods in its namespace alone, though the
// namespaces the plan names miss it: the tab streams the pod's log, and nothing is read in all
// namespaces.
func TestWorkloadLogsWaitForThePlanAndReadTheirNamespace(t *testing.T) {
	src, rev := shopOnly(t)
	rev.set(func(f *fakeReviewer) { f.gate = make(chan struct{}) })
	s := shopSession(t, src, rev, nil)
	f := newFakeLogs()
	f.content["web-1-a/app"] = "hello\n"

	type opened struct {
		tab DockTab
		err error
	}
	done := make(chan opened, 1)
	go func() {
		tab, _, err := s.OpenWorkloadLogsTab(context.Background(), model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"})
		done <- opened{tab, err}
	}()
	time.Sleep(50 * time.Millisecond)
	rev.set(func(f *fakeReviewer) { close(f.gate); f.gate = nil })
	var o opened
	select {
	case o = <-done:
	case <-time.After(15 * time.Second):
		t.Fatal("the tab never opened")
	}
	if o.err != nil {
		t.Fatalf("open the tab: %v (calls %v)", o.err, src.Calls())
	}
	lr := readLines(openTabLogs(t, s, o.tab.ID, f))
	lr.wait(t, "web-1-a/app hello")
	for k, n := range src.Calls() {
		if n > 0 && (k == "list pods" || k == "list replicasets.apps" || k == "list deployments.apps") {
			t.Errorf("read in all namespaces: %v", src.Calls())
		}
	}
}
