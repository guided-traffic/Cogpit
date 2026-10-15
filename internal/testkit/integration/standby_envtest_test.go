//go:build envtest && unix

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
	"github.com/guided-traffic/cogpit/internal/testkit/envtest"
)

// envtestWatch are D-58's intervals shortened for the test: a heartbeat every 1.5 s with 1 s to
// answer (a real API server on a loaded runner answers /version within milliseconds; the second
// keeps a slow answer from counting as a failed check), the second check 0.2 s after a failed one,
// the reconnect attempts from 0.1 s doubling to 0.8 s.
var envtestWatch = engine.ConnWatchOptions{
	Heartbeat: 1500 * time.Millisecond, ProbeTimeout: time.Second, Confirm: 200 * time.Millisecond,
	LoginWait: 3 * time.Second, BackoffFirst: 100 * time.Millisecond, BackoffMax: 800 * time.Millisecond,
}

// TestEnvtestStandbyAPIServerStopped is D-58's (e) against a real kube-apiserver: envtest's etcd
// and kube-apiserver (Kubernetes 1.37.0), the engine connected through cluster.Connect with a
// cluster-admin's client certificate, the real client-go stack in between, and no proxy. The API
// server's process is stopped with SIGSTOP, a peer gone silent as a cluster is after the computer
// slept: its port still accepts connections and takes requests, nothing answers. The connection
// is interrupted within the heartbeat window, the view keeps its rows, a write is refused at once
// for the window and for agents; after SIGCONT the connection is back within the backoff, the
// cache's informers list or watch again (the gate held them) and follow what changes from then on.
func TestEnvtestStandbyAPIServerStopped(t *testing.T) {
	cp := envtest.Start(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "shop"}}
	if _, err := cp.Clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	createPod := func(name string) {
		t.Helper()
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: name},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "web", Image: "registry.example/web:1"}}},
		}
		if _, err := cp.Clientset.CoreV1().Pods("shop").Create(ctx, pod, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	createPod("web-0")
	createPod("web-1")

	s := engine.New(engine.Options{
		Connector:        &engine.ClusterConnector{KubeconfigPaths: []string{cp.WriteKubeconfig(t)}},
		ClusterSettings:  engine.NewClusterSettingsStore(t.TempDir()),
		SnapshotInterval: 20 * time.Millisecond, ConnWatch: envtestWatch,
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, envtest.Context); err != nil {
		t.Fatal(err)
	}
	rows := func() int {
		if tbl := s.Table("pods"); tbl != nil && tbl.Synced {
			return len(tbl.Rows)
		}
		return -1
	}
	connected := func() bool { return s.UIState().Connection.State == engine.Connected }
	if !testkit.WaitFor(ctx, func() bool { return rows() == 2 && connected() }) {
		t.Fatalf("pods: %d, connection %s", rows(), s.UIState().Connection.State)
	}

	cp.FreezeAPIServer(t)
	frozen := time.Now()
	if !testkit.WaitFor(ctx, func() bool { return s.UIState().Connection.State == engine.Interrupted }) {
		t.Fatalf("still %s", s.UIState().Connection.State)
	}
	interrupted := time.Since(frozen)
	// The next heartbeat and its timeout, the second check after Confirm and its timeout.
	window := envtestWatch.Heartbeat + 2*envtestWatch.ProbeTimeout + envtestWatch.Confirm + 2*time.Second
	if interrupted > window {
		t.Errorf("interrupted after %v, want within %v", interrupted, window)
	}
	if c := s.UIState().Connection; c.Since.IsZero() || c.Reason == "" {
		t.Errorf("connection %+v", c)
	}
	if rows() != 2 || s.Table("pods").Error != "" {
		t.Errorf("the view while interrupted: %d rows, error %q", rows(), s.Table("pods").Error)
	}
	// Writes are refused at once, for every origin, without a request to the stopped server.
	pod := model.ObjectRef{Context: envtest.Context, Kind: "pods", Namespace: "shop", Name: "web-0"}
	raw, _ := json.Marshal(map[string]any{"ref": pod})
	for _, o := range []engine.Origin{engine.OriginGUI, engine.OriginExternalAgent} {
		start := time.Now()
		_, err := s.Dispatch(ctx, engine.Invocation{ID: "object.delete", Params: raw, Origin: o})
		var de *engine.DisabledError
		if !errors.As(err, &de) || de.Reason != "connection interrupted" || time.Since(start) > 500*time.Millisecond {
			t.Errorf("%s's delete while interrupted: %v after %v", o, err, time.Since(start))
		}
	}
	// The connection stays interrupted while the server is stopped: the reconnect attempts dial
	// and the stopped process answers none of them, each failed attempt schedules the next one.
	attempts := map[time.Time]bool{}
	if !testkit.WaitFor(ctx, func() bool {
		c := s.UIState().Connection
		if c.State != engine.Interrupted {
			return true
		}
		if !c.NextAttempt.IsZero() {
			attempts[c.NextAttempt] = true
		}
		return len(attempts) >= 3
	}) {
		t.Fatalf("reconnect attempts scheduled: %d, connection %+v", len(attempts), s.UIState().Connection)
	}
	if st := s.UIState().Connection.State; st != engine.Interrupted {
		t.Fatalf("against the stopped server after %d scheduled attempts: %s", len(attempts), st)
	}

	cp.ThawAPIServer(t)
	thawed := time.Now()
	if !testkit.WaitFor(ctx, connected) {
		t.Fatalf("still %s after SIGCONT", s.UIState().Connection.State)
	}
	reconnected := time.Since(thawed)
	if bound := envtestWatch.BackoffMax + 2*envtestWatch.ProbeTimeout + 2*time.Second; reconnected > bound {
		t.Errorf("reconnected %v after SIGCONT, want within %v", reconnected, bound)
	}
	// The cache syncs again and follows the server: a pod created and one deleted after the
	// reconnect reach the view through the informers' new list or watch.
	createPod("web-2")
	if !testkit.WaitFor(ctx, func() bool { return rows() == 3 }) {
		t.Fatalf("pods after the reconnect and a create: %d", rows())
	}
	synced := time.Since(thawed)
	if err := cp.Clientset.CoreV1().Pods("shop").Delete(ctx, "web-1", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool { return rows() == 2 && s.Table("pods").IndexOf(pod) >= 0 }) {
		t.Fatalf("pods after a delete: %d", rows())
	}
	if _, err := s.Dispatch(ctx, engine.Invocation{ID: "object.delete", Params: raw, Origin: engine.OriginCLI}); err != nil &&
		errors.As(err, new(*engine.DisabledError)) {
		t.Errorf("a write after the reconnect is still disabled: %v", err)
	}
	t.Logf("interrupted %v after SIGSTOP (window %v); connected %v after SIGCONT, a new pod in the view after %v",
		interrupted.Round(time.Millisecond), window, reconnected.Round(time.Millisecond), synced.Round(time.Millisecond))
}
