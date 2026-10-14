package metrics

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// History that outlives a closed drawer (G12): the hub keeps a closed drawer's CPU and memory
// series, bounded by age, by charts and by points, and hands them back to the next drawer of the
// object.

// liveOnly is a Metrics Server: one pod and one node, measured at the clock's time.
type liveOnly struct {
	now func() time.Time
}

func (*liveOnly) ID() string                 { return ServerID }
func (*liveOnly) Label() string              { return ServerLabel }
func (*liveOnly) Capabilities() Capabilities { return Capabilities{} }
func (l *liveOnly) ListPods(context.Context, string) ([]model.PodUsage, error) {
	return nil, nil
}
func (l *liveOnly) GetPod(_ context.Context, ns, name string) (model.PodUsage, error) {
	return model.PodUsage{Namespace: ns, Name: name, Timestamp: l.now(), Containers: []model.ContainerUsage{
		{Name: "app", Usage: model.Usage{CPUMilli: 100, MemoryBytes: 64 << 20}},
	}}, nil
}
func (l *liveOnly) ListNodes(context.Context) ([]model.NodeUsage, error) { return nil, nil }
func (l *liveOnly) GetNode(context.Context, string) (model.NodeUsage, error) {
	return model.NodeUsage{}, ErrNoData
}

// series makes n points of a target at step, ending at end.
func series(n int, end time.Time, step time.Duration) []model.UsageSample {
	out := make([]model.UsageSample, n)
	for i := range out {
		out[i] = model.UsageSample{At: end.Add(-time.Duration(n-1-i) * step), Usage: model.Usage{CPUMilli: int64(i), MemoryBytes: int64(i) << 20}}
	}
	return out
}

// testChart is a closed drawer's chart: the pod and three containers, n points each.
func testChart(n int, end time.Time) *chart {
	c := newChart()
	for _, t := range []string{"", "a", "b", "c"} {
		c.usage[t] = series(n, end, 15*time.Second)
	}
	c.network = []model.NetworkSample{{At: end}}
	c.at = end
	return c
}

func podScope(i int) Scope { return Scope{Kind: KindPods, Namespace: "web", Name: fmt.Sprint("p", i)} }

// Reopening a pod's drawer shows its last minutes again: the Metrics Server keeps no history, the
// hub keeps the closed chart's series and gives them to the next subscription of the object.
func TestHistoryOutlivesAClosedDrawer(t *testing.T) {
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	advance := func(d time.Duration) { mu.Lock(); now = now.Add(d); mu.Unlock() }
	h := NewHub(Fixed(&liveOnly{now: clock}), Options{Now: clock, ObjectInterval: time.Hour})
	defer h.Close()
	sc := Scope{Kind: KindPods, Namespace: "web", Name: "a"}

	polled := make(chan struct{}, 8)
	sub := h.Subscribe(sc, func() { polled <- struct{}{} })
	<-polled
	for range 3 { // three more measurements, 15 s apart
		advance(15 * time.Second)
		Kick(h, sc)
		<-polled
	}
	if n := len(h.History(sc).Targets[""].Usage); n != 4 {
		t.Fatalf("%d points while the drawer is open, want 4", n)
	}
	sub.Release()
	if charts, points := h.Retained(); charts != 1 || points != 8 {
		t.Fatalf("kept %d charts, %d points after the drawer closed, want 1 and 8 (the pod and its container)", charts, points)
	}

	// Ten minutes later the drawer opens again: the chart starts with the kept points, before its
	// first poll answers, and continues after a gap.
	advance(10 * time.Minute)
	sub = h.Subscribe(sc, func() { polled <- struct{}{} })
	defer sub.Release()
	if hs := h.History(sc); hs == nil || len(hs.Targets[""].Usage) != 4 || len(hs.Targets["app"].Usage) != 4 {
		t.Fatalf("the reopened drawer starts with %+v, want the 4 kept points", hs)
	}
	<-polled
	if pts := h.History(sc).Targets[""].Usage; len(pts) != 5 || pts[4].At.Sub(pts[3].At) != 10*time.Minute {
		t.Errorf("after the first poll: %d points, the last two %v apart, want 5 and the 10 minutes the drawer was closed", len(pts), pts[len(pts)-1].At.Sub(pts[len(pts)-2].At))
	}
	if charts, _ := h.Retained(); charts != 0 {
		t.Errorf("%d charts kept while the drawer is open again, want none (the chart is the scope's)", charts)
	}
}

// A kept chart is good for an hour: older, the drawer starts empty.
func TestRetainedHistoryExpires(t *testing.T) {
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	h := NewHub(Fixed(&liveOnly{now: func() time.Time { return now }}), Options{Now: func() time.Time { return now }, ObjectInterval: time.Hour})
	defer h.Close()
	h.mu.Lock()
	h.retainLocked(podScope(1), testChart(240, now))
	now = now.Add(61 * time.Minute)
	c := h.restoreLocked(podScope(1))
	h.mu.Unlock()
	if len(c.usage) != 0 || !c.at.IsZero() {
		t.Errorf("a chart kept 61 minutes ago came back: %d series", len(c.usage))
	}
	if charts, points := h.Retained(); charts != 0 || points != 0 {
		t.Errorf("the expired chart is still kept: %d, %d", charts, points)
	}
}

// The store's bounds: each series an hour back from its newest point, at most RetainObjects charts
// and RetainPoints points, the least recently closed dropped first; the network and the filesystem
// are not kept; the heap the kept charts take is measured.
func TestRetainedHistoryIsBounded(t *testing.T) {
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	h := NewHub(Fixed(&liveOnly{now: func() time.Time { return now }}), Options{Now: func() time.Time { return now }})
	defer h.Close()

	// A day of points in each series: only the last hour is kept (241 points at 15 s).
	h.mu.Lock()
	h.retainLocked(podScope(0), testChart(5760, now))
	r := h.retained[podScope(0)]
	h.mu.Unlock()
	if n := len(r.usage[""]); n != 241 || r.points != 4*241 {
		t.Fatalf("a day's series kept as %d points (%d in all), want the last hour's 241", n, r.points)
	}
	if c := cap(r.usage[""]); c > 2*241 {
		t.Errorf("the kept series holds an array of %d: the day's array, not a copy of the hour", c)
	}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)
	// 1,000 drawers closed, each an hour of four series: 964 points each.
	for i := range 1000 {
		c := testChart(241, now)
		h.mu.Lock()
		h.retainLocked(podScope(i), c)
		h.mu.Unlock()
	}
	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)
	charts, points := h.Retained()
	if charts > DefaultRetainObjects || points > DefaultRetainPoints {
		t.Fatalf("kept %d charts and %d points, bounds %d and %d", charts, points, DefaultRetainObjects, DefaultRetainPoints)
	}
	// 51 charts of 964 points would pass the points' bound: 50,000 / 964 = 51; the charts' bound
	// (50) holds first.
	if charts != DefaultRetainObjects {
		t.Errorf("kept %d charts, want %d", charts, DefaultRetainObjects)
	}
	h.mu.Lock()
	oldest := h.retainOrder[0]
	_, first := h.retained[podScope(0)]
	h.mu.Unlock()
	if oldest != podScope(950) || first {
		t.Errorf("the oldest kept chart is %v (the first closed still kept: %v), want p950 (LRU)", oldest, first)
	}
	grown := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	const sampleBytes = 48 // model.UsageSample: a time.Time and a usage
	limit := int64(DefaultRetainPoints*sampleBytes) * 2
	t.Logf("kept %d charts, %d points: the heap grew by %d bytes (samples alone %d)", charts, points, grown, points*sampleBytes)
	if grown > limit {
		t.Errorf("the heap grew by %d bytes with %d points kept, more than %d", grown, points, limit)
	}

	// A chart of more points than the store may hold is not kept; a negative RetainObjects keeps none.
	h.mu.Lock()
	h.opts.RetainPoints = 100
	h.retainLocked(podScope(5000), testChart(241, now))
	_, kept := h.retained[podScope(5000)]
	h.mu.Unlock()
	if kept {
		t.Error("a chart larger than the store was kept")
	}
	off := NewHub(Fixed(&liveOnly{now: func() time.Time { return now }}), Options{Now: func() time.Time { return now }, RetainObjects: -1})
	defer off.Close()
	off.mu.Lock()
	off.retainLocked(podScope(1), testChart(10, now))
	off.mu.Unlock()
	if charts, _ := off.Retained(); charts != 0 {
		t.Error("RetainObjects -1 kept a chart")
	}
}

// Keeping a chart and taking it back move it in the LRU order: a chart taken back and closed
// again is the newest.
func TestRetainedHistoryLRU(t *testing.T) {
	now := time.Date(2026, 10, 14, 10, 0, 0, 0, time.UTC)
	h := NewHub(Fixed(&liveOnly{now: func() time.Time { return now }}), Options{Now: func() time.Time { return now }, RetainObjects: 2})
	defer h.Close()
	h.mu.Lock()
	defer h.mu.Unlock()
	h.retainLocked(podScope(1), testChart(4, now))
	h.retainLocked(podScope(2), testChart(4, now))
	c := h.restoreLocked(podScope(1)) // reopened
	if len(c.usage) != 4 || len(c.network) != 0 {
		t.Errorf("restored %d series and %d network points, want 4 usage series and no network", len(c.usage), len(c.network))
	}
	h.retainLocked(podScope(1), c) // closed again: the newest
	h.retainLocked(podScope(3), testChart(4, now))
	if _, ok := h.retained[podScope(2)]; ok || len(h.retained) != 2 {
		t.Errorf("kept %v, want p1 and p3 (p2 the least recently closed)", h.retainOrder)
	}
}
