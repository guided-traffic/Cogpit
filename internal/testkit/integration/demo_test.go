package integration_test

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestDemoClusterThroughTheEngine runs `cogpit --demo` without the window: the real demo cluster
// (10,000 pods, 1,000 watch events per second) through the real cache and engine. The Pods view
// fills, the churn reaches the view as at most ten snapshots per second (architecture §5), and the
// agent showcase finds the demo's Service with ClusterIP 10.96.100.100.
func TestDemoClusterThroughTheEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the 10,000-pod demo cluster")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{IncludeDemo: true}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, engine.DemoContext); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool {
		tbl := s.Table("pods")
		return tbl != nil && tbl.Synced && tbl.Total >= demo.DefaultPods*9/10
	}) {
		t.Fatalf("the Pods view never filled: %+v", s.Table("pods"))
	}
	t.Logf("pods view: %d rows", s.Table("pods").Total)

	var snapshots atomic.Int64
	cancelSub := s.Subscribe(func(ev engine.Event) {
		if ev.Type == engine.EventTable && ev.View == "pods" {
			snapshots.Add(1)
		}
	})
	const window = 3 * time.Second
	time.Sleep(window)
	cancelSub()
	n := snapshots.Load()
	t.Logf("%d pod snapshots in %s under the demo's churn", n, window)
	if n > int64(window/(100*time.Millisecond))+2 {
		t.Errorf("%d snapshots in %s: coalescing allows at most ten per second", n, window)
	}
	if n < int64(window/time.Second) {
		t.Errorf("only %d snapshots in %s: the churn does not reach the view", n, window)
	}

	res, err := s.Dispatch(ctx, engine.Invocation{
		ID:     "resources.find",
		Params: json.RawMessage(`{"kind":"services","field":"clusterIP","value":"10.96.100.100"}`),
		Origin: engine.OriginExternalAgent,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := res.(commands.FindResult)
	want := model.ObjectRef{Kind: "services", Namespace: demo.MCPNamespace, Name: demo.MCPService}
	if found.Count != 1 || !found.Matches[0].Ref.Same(want) {
		t.Fatalf("resources.find: %+v", found)
	}
}
