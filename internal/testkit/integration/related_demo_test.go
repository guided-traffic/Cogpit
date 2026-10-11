package integration_test

import (
	"context"
	"reflect"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestNodeDrawerUnderDemoChurn opens a node's drawer on the demo cluster (10,000 pods on 50 nodes,
// 1,000 watch events per second) and reads it the way the window does: on every EventDetail it
// fetches the detail and redraws only when the sections changed (drawer.show compares them). The
// node's Pods table comes from the related-objects index over all pods, so every pod event in the
// cluster wakes the drawer; the engine must still announce it at most ten times a second, and the
// redraws can be no more than those announcements.
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
	ref := model.ObjectRef{Context: engine.DemoContext, Kind: "nodes", Name: nodes[0].GetName()}
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

	const window = 3 * time.Second
	time.Sleep(window)
	cancelSub()
	close(stop)
	<-done

	n, f, r := events.Load(), fetches.Load(), redraws.Load()
	mu.Lock()
	slices.Sort(took)
	var p50, maxTook time.Duration
	if len(took) > 0 {
		p50, maxTook = took[len(took)/2], took[len(took)-1]
	}
	mu.Unlock()
	t.Logf("node %s with %d pods, %s of churn: %d detail events, %d fetches, %d redraws; detail built in %v (p50), %v (max)",
		ref.Name, rows, window, n, f, r, p50, maxTook)
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
