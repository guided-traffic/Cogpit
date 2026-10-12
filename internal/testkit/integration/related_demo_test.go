package integration_test

import (
	"context"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestNodeDrawerUnderDemoChurn opens a node's drawer on the demo cluster (10,000 pods on 50 nodes,
// 1,000 watch events per second) and reads it the way the window does: on every EventDetail it
// fetches the detail and redraws only when the sections changed (drawer.show compares them). The
// node's Pods table comes from the related-objects index over all pods, which the drawer keeps
// current and shares with its fetches; it wakes for its own node's pods only, which under the
// demo's churn still change several times a second. The engine must announce it at most ten
// times a second, and the redraws can be no more than those announcements. The test logs what a
// fetch costs and the process's CPU with and without the drawer (architecture §4.4).
func TestNodeDrawerUnderDemoChurn(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the 10,000-pod demo cluster")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{IncludeDemo: true}})
	defer s.Close()
	if err := s.Connect(ctx, engine.DemoContext); err != nil {
		t.Fatal(err)
	}
	nodes, err := s.Objects(ctx, "nodes", nil)
	if err != nil || len(nodes) == 0 {
		t.Fatalf("demo nodes: %d, %v", len(nodes), err)
	}
	// The node running the most pods: the demo has not-ready and cordoned nodes with few, and the
	// cache lists nodes in no particular order.
	pods, err := s.Objects(ctx, "pods", nil)
	if err != nil {
		t.Fatal(err)
	}
	perNode := map[string]int{}
	busiest := nodes[0].GetName()
	for _, p := range pods {
		n, _, _ := unstructured.NestedString(p.Object, "spec", "nodeName")
		if perNode[n]++; n != "" && perNode[n] > perNode[busiest] {
			busiest = n
		}
	}
	ref := model.ObjectRef{Context: engine.DemoContext, Kind: "nodes", Name: busiest}

	// The baseline: the same cluster and churn without the drawer, its informers warm.
	const window = 3 * time.Second
	cpu0, measured := processCPU()
	time.Sleep(window)
	cpu1, _ := processCPU()
	baseline := cpu1 - cpu0

	s.UpdateUI(func(u *engine.UIState) { u.Drawer = &ref })
	podsTable := func(m *engine.DetailModel) *model.DetailTable {
		for _, sec := range m.Sections {
			if sec.Title == "Pods" {
				return sec.Table
			}
		}
		return nil
	}
	if !testkit.WaitFor(ctx, func() bool {
		m, err := s.DetailWait(ctx, ref)
		return err == nil && podsTable(m) != nil && len(podsTable(m).Rows) >= demo.DefaultPods/demo.DefaultNodes/2
	}) {
		t.Fatal("the node's Pods table never filled")
	}

	// The window's reader: a wake per EventDetail, coalesced, one fetch per wake.
	var (
		events, fetches, redraws atomic.Int64
		mu                       sync.Mutex
		took                     []time.Duration
		rows                     int
	)
	wake := make(chan struct{}, 1)
	stop := make(chan struct{})
	cancelSub := s.Subscribe(func(ev engine.Event) {
		if ev.Type == engine.EventDetail {
			events.Add(1)
			select {
			case wake <- struct{}{}:
			default:
			}
		}
	})
	done := make(chan struct{})
	go func() {
		defer close(done)
		var last []model.DetailSection
		for {
			select {
			case <-stop:
				return
			case <-wake:
			}
			start := time.Now()
			m, err := s.Detail(ref)
			d := time.Since(start)
			if err != nil {
				continue
			}
			fetches.Add(1)
			mu.Lock()
			took = append(took, d)
			if tbl := podsTable(m); tbl != nil {
				rows = len(tbl.Rows)
			}
			mu.Unlock()
			if last == nil || !reflect.DeepEqual(last, m.Sections) {
				redraws.Add(1)
			}
			last = m.Sections
		}
	}()

	cpu2, _ := processCPU()
	time.Sleep(window)
	cpu3, _ := processCPU()
	cancelSub()
	close(stop)
	<-done

	n, f, r := events.Load(), fetches.Load(), redraws.Load()
	mu.Lock()
	slices.Sort(took)
	var p50, maxTook, inFetch time.Duration
	if len(took) > 0 {
		p50, maxTook = took[len(took)/2], took[len(took)-1]
	}
	for _, d := range took {
		inFetch += d
	}
	mu.Unlock()
	t.Logf("node %s with %d pods, %s of churn: %d detail events, %d fetches, %d redraws; detail built in %v (p50), %v (max); fetching took %.1f%% of a core",
		ref.Name, rows, window, n, f, r, p50, maxTook, 100*inFetch.Seconds()/window.Seconds())
	if measured {
		// What the open drawer and its reader cost the process beyond the churn alone.
		t.Logf("process CPU: %.1f%% of a core without the drawer, %.1f%% with it: the drawer costs %.1f%%",
			100*baseline.Seconds()/window.Seconds(), 100*(cpu3-cpu2).Seconds()/window.Seconds(),
			100*((cpu3-cpu2)-baseline).Seconds()/window.Seconds())
	}
	if limit := int64(window/(100*time.Millisecond)) + 2; n > limit {
		t.Errorf("%d detail events in %s: coalescing allows at most %d", n, window, limit)
	}
	if n < int64(window/time.Second) {
		t.Errorf("only %d detail events in %s: the churn does not reach the drawer", n, window)
	}
	if r > n {
		t.Errorf("%d redraws for %d detail events", r, n)
	}
}
