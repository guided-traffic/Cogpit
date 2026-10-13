package engine

import (
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// contextOnly is a cluster whose user has a RoleBinding in the context's own namespace team-a and
// nothing else: pods, services, deployments and replica sets there, refused in all namespaces and
// in team-b. The rules review of team-a allows them and no accessible namespaces are named, so a
// plan reads them in all namespaces first, and the API server refuses that. It holds the
// Deployment web with its replica set and its ready pod web-1-a (label app=web, port http 8080),
// and the Service web in front of it.
func contextOnly(t *testing.T) (*testkit.NamespacedOnlySource, *fakeReviewer) {
	t.Helper()
	mem := testkit.NewMemorySource()
	mem.Set(testkit.DeploymentsGVR, testkit.Object("apps/v1", "Deployment", "team-a", "web", nil))
	mem.Set(replicaSetsGVR, owned(testkit.Object("apps/v1", "ReplicaSet", "team-a", "web-1", nil), "apps/v1", "Deployment", "web", "uid-team-a-web"))
	mem.Set(testkit.PodsGVR, owned(readyWebPod("team-a", "web-1-a"), "apps/v1", "ReplicaSet", "web-1", "uid-team-a-web-1"))
	mem.Set(testkit.ServicesGVR, testkit.Object("v1", "Service", "team-a", "web", map[string]any{
		"spec": map[string]any{"selector": map[string]any{"app": "web"}, "ports": []any{map[string]any{"port": int64(80), "targetPort": "http"}}},
	}))
	src := &testkit.NamespacedOnlySource{Inner: mem, Denied: []string{"team-b"}, GVRs: []schema.GroupVersionResource{
		testkit.PodsGVR, testkit.ServicesGVR, testkit.DeploymentsGVR, replicaSetsGVR,
	}}
	rev := newFakeReviewer(map[string][]cluster.AccessRule{"team-a": {
		rule("get,list,watch", "", "pods,services"), rule("get,list,watch", "apps", "deployments,replicasets"),
	}})
	return src, rev
}

// readyWebPod is a running, ready pod labelled app=web with the container app and its port http
// 8080: a target of the Service web.
func readyWebPod(namespace, name string) *unstructured.Unstructured {
	return testkit.Object("v1", "Pod", namespace, name, map[string]any{
		"metadata": map[string]any{"labels": map[string]any{"app": "web"}},
		"spec": map[string]any{"containers": []any{map[string]any{
			"name": "app", "image": "registry.example/app:1",
			"ports": []any{map[string]any{"name": "http", "containerPort": int64(8080)}},
		}}},
		"status": map[string]any{
			"phase":             "Running",
			"conditions":        []any{map[string]any{"type": "Ready", "status": "True"}},
			"containerStatuses": []any{map[string]any{"name": "app", "restartCount": int64(0), "state": map[string]any{"running": map[string]any{}}}},
		},
	})
}

func contextOnlySession(t *testing.T, src *testkit.NamespacedOnlySource, rev *fakeReviewer, actions Actions) *Session {
	t.Helper()
	rs, ok := kinds.ByKey("replicasets")
	if !ok {
		t.Fatal("no replica sets profile")
	}
	s := New(Options{Connector: dockConnector{src: src, rev: rev, profiles: append(testProfiles(), rs), actions: actions, namespace: "team-a"},
		SnapshotInterval: 5 * time.Millisecond})
	t.Cleanup(s.Close)
	connectTest(t, s)
	return s
}

// callsOf are the source's list and watch calls of one resource, by scope ("list services",
// "watch services in team-a").
func callsOf(src *testkit.NamespacedOnlySource, resource string) map[string]int {
	out := map[string]int{}
	for k, v := range src.Calls() {
		if strings.Contains(k, " "+resource) {
			out[k] = v
		}
	}
	return out
}

// A user whose RoleBinding is in the context's own namespace only: the plan reads a kind in all
// namespaces first (the rules allow it in team-a, nothing else is reviewed), and the API server
// refuses. The reads that wait for their target (one object, a Deployment's logs, a Service's port
// forward) follow that refusal as the views do: they read the target's namespace alone at once,
// instead of waiting 15 seconds for a list that never comes and failing. The refusal is kept, so
// the next read and the kind's view go to the namespace without trying all namespaces again.
func TestTargetReadsFollowARefusedClusterWideList(t *testing.T) {
	src, rev := contextOnly(t)
	actions := &dockActions{}
	s := contextOnlySession(t, src, rev, actions)

	// One object: the Deployment, through the read that agents and the CLI use.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	obj, err := s.Object(ctx, model.ObjectRef{Kind: "deployments", Namespace: "team-a", Name: "web"})
	if err != nil || obj.GetName() != "web" {
		t.Fatalf("Object: %v, %v (calls %v)", obj, err, callsOf(src, "deployments"))
	}
	if calls := callsOf(src, "deployments"); calls["list deployments.apps"] != 1 || calls["list deployments.apps in team-a"] != 1 {
		t.Errorf("deployment calls %v: want the refused list in all namespaces once, then team-a's", calls)
	}

	// The Deployment's logs: its replica sets are read in team-a after one refusal.
	f := newFakeLogs()
	f.content["web-1-a/app"] = "hello\n"
	tab, _, err := s.OpenWorkloadLogsTab(ctx, model.ObjectRef{Kind: "deployments", Namespace: "team-a", Name: "web"})
	if err != nil {
		t.Fatalf("open the tab: %v", err)
	}
	type stream struct {
		rc  io.ReadCloser
		err error
	}
	opened := make(chan stream, 1)
	go func() {
		dt, b, err := s.streamTab(tab.ID, DockLogs)
		if err != nil {
			opened <- stream{err: err}
			return
		}
		rc, err := s.openLogs(ctx, dt, b, f)
		opened <- stream{rc, err}
	}()
	select {
	case o := <-opened:
		if o.err != nil {
			t.Fatalf("the logs: %v (calls %v)", o.err, callsOf(src, "replicasets"))
		}
		t.Cleanup(func() { _ = o.rc.Close() })
		readLines(o.rc).wait(t, "web-1-a/app hello")
	case <-time.After(10 * time.Second):
		t.Fatalf("the logs never opened; replica set calls %v", callsOf(src, "replicasets"))
	}
	if calls := callsOf(src, "replicasets"); calls["list replicasets.apps"] != 1 || calls["list replicasets.apps in team-a"] != 1 {
		t.Errorf("replica set calls %v: want one refused list in all namespaces, then team-a's", calls)
	}

	// A Service's port forward: the services are read in team-a after one refusal.
	fw, err := s.StartForward(ctx, ForwardSpec{Target: model.ObjectRef{Kind: "services", Namespace: "team-a", Name: "web"}, Port: 80})
	if err != nil || fw.Phase != ForwardActive || !slices.Equal(actions.dialed(), []string{"team-a/web-1-a"}) {
		t.Fatalf("forward %+v, %v; dials %v; service calls %v", fw, err, actions.dialed(), callsOf(src, "services"))
	}
	if calls := callsOf(src, "services"); calls["list services"] != 1 || calls["list services in team-a"] != 1 {
		t.Errorf("service calls %v: want one refused list in all namespaces, then team-a's", calls)
	}
	if _, err := s.StopForward(fw.Ref); err != nil {
		t.Error(err)
	}

	// The refusal is kept: the Deployments view reads team-a at once, without a second try in all
	// namespaces.
	s.UpdateUI(func(u *UIState) { u.View = "deployments" })
	ts := waitTable(t, s, "deployments", func(ts *TableSnapshot) bool { return ts.Synced && len(ts.Rows) == 1 })
	if !slices.Equal(ts.Watched, []string{"team-a"}) || ts.Error != "" {
		t.Errorf("the Deployments view: watched %q, error %q", ts.Watched, ts.Error)
	}
	if calls := callsOf(src, "deployments"); calls["list deployments.apps"] != 1 {
		t.Errorf("deployment calls %v: the view tried all namespaces again", calls)
	}
}
