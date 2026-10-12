package engine

import (
	"context"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The cluster Overview's charts: CPU, Memory and Pods for the chosen node group, each with the
// lines Lens's documentation lists (CPU and Memory: usage, requests, limits, allocatable capacity,
// capacity; Pods: usage, allocatable capacity, capacity). The lines come from the cache; the usage
// series of CPU and Memory from the metrics hub (the cluster scope), the Pods' usage from the
// cache, a point per sample while the page is open.

// The charts' metrics (OverviewChart.Metric).
const (
	OverviewCPU    = "cpu"
	OverviewMemory = "memory"
	OverviewPods   = "pods"
)

// OverviewChart is one chart of the Overview, in its own unit: cores, bytes or pods.
type OverviewChart struct {
	Metric string `json:"metric"` // "cpu", "memory", "pods"
	Title  string `json:"title"`  // "CPU", "Memory", "Pods"
	Unit   string `json:"unit"`   // "cores", "bytes", "pods"
	// Current is the newest usage and At its time; nil without one.
	Current *float64  `json:"current,omitempty"`
	At      time.Time `json:"at,omitzero"`
	// Requests and Limits are the sums over the pods that have not finished on the group's nodes
	// (a limit counts where a container sets one, as kubectl's "Allocated resources" does); the
	// Pods chart has neither. Allocatable and Capacity are the nodes' sums.
	Requests    *float64 `json:"requests,omitempty"`
	Limits      *float64 `json:"limits,omitempty"`
	Allocatable float64  `json:"allocatable"`
	Capacity    float64  `json:"capacity"`
	// Samples is the usage over the chart's range, oldest first; only with series.
	Samples []OverviewSample `json:"samples,omitempty"`
}

// OverviewSample is one point of a chart.
type OverviewSample struct {
	At    time.Time `json:"at"`
	Value float64   `json:"value"`
}

// overviewSampleEvery is how often the open page samples its pods count (the Metrics Server's
// resolution, as the drawer's polls); a test shortens it.
var overviewSampleEvery = 15 * time.Second

// overviewMaxPoints bounds a pods series: a day of samples.
const overviewMaxPoints = 5760

// overviewCharts is the open page's chart data: the pods series of each node group, sampled from
// the cache while the page is open.
type overviewCharts struct {
	s *Session
	b *Backend

	mu     sync.Mutex
	choice OverviewChoice
	last   *overviewBuild
	pods   map[string][]OverviewSample // by node group
	quit   chan struct{}
	once   sync.Once
}

func newOverviewCharts(s *Session, b *Backend, choice OverviewChoice) *overviewCharts {
	c := &overviewCharts{s: s, b: b, choice: choice, pods: map[string][]OverviewSample{}, quit: make(chan struct{})}
	s.goSafe("cluster overview samples", c.sampleLoop)
	return c
}

// sampleLoop records the pods count every overviewSampleEvery and tells the window.
func (c *overviewCharts) sampleLoop() {
	t := time.NewTicker(overviewSampleEvery)
	defer t.Stop()
	for {
		select {
		case <-c.quit:
			return
		case now := <-t.C:
			c.mu.Lock()
			if c.last != nil {
				c.sampleLocked(c.last, now)
			}
			c.mu.Unlock()
			c.s.emit(Event{Type: EventMetrics, View: ClusterOverview})
		}
	}
}

func (c *overviewCharts) setChoice(choice OverviewChoice) {
	c.mu.Lock()
	c.choice = choice
	c.mu.Unlock()
	c.s.emit(Event{Type: EventMetrics, View: ClusterOverview})
}

func (c *overviewCharts) release() {
	c.once.Do(func() { close(c.quit) })
}

// follow takes a new build: the pods series gets a point when the count changed.
func (c *overviewCharts) follow(ob *overviewBuild) {
	c.mu.Lock()
	defer c.mu.Unlock()
	changed := c.last == nil
	for _, g := range []string{NodesMaster, NodesWorker} {
		if c.last != nil && c.last.groups[g].pods != ob.groups[g].pods {
			changed = true
		}
	}
	c.last = ob
	if changed {
		c.sampleLocked(ob, ob.at)
	}
}

// sampleLocked adds a pods point per group at, bounded to the longest range and a day of samples.
func (c *overviewCharts) sampleLocked(ob *overviewBuild, at time.Time) {
	keep := max(chartWindowOf(c.choice), DefaultOverviewWindow)
	for _, g := range []string{NodesMaster, NodesWorker} {
		pts := c.pods[g]
		if n := len(pts); n > 0 && !at.After(pts[n-1].At) {
			continue
		}
		pts = append(pts, OverviewSample{At: at, Value: float64(ob.groups[g].pods)})
		cut := 0
		for cut < len(pts) && pts[cut].At.Before(at.Add(-keep)) {
			cut++
		}
		if over := len(pts) - cut - overviewMaxPoints; over > 0 {
			cut += over
		}
		c.pods[g] = pts[cut:]
	}
}

// fill adds the charts of the open page to m: the lines, the current values, and with series the
// samples within the range.
func (c *overviewCharts) fill(m *ClusterOverviewModel, ob *overviewBuild, series bool) {
	m.Charts = chartLines(ob.groups[m.Nodes])
	c.mu.Lock()
	pods := c.pods[m.Nodes]
	c.mu.Unlock()
	if n := len(pods); n > 0 {
		last := pods[n-1]
		v := last.Value
		m.Charts[2].Current, m.Charts[2].At = &v, last.At
		if series {
			m.Charts[2].Samples = samplesWithin(pods, ob.at.Add(-m.Window))
		}
	}
}

// samplesWithin returns the points at or after from; shared, not to be modified.
func samplesWithin(pts []OverviewSample, from time.Time) []OverviewSample {
	return tail(pts, from, func(p OverviewSample) time.Time { return p.At })
}

// currentCharts fills the charts of a model built without the page: the lines and the current pods
// count from the cache.
func currentCharts(_ context.Context, _ *Backend, m *ClusterOverviewModel, ob *overviewBuild) {
	m.Charts = chartLines(ob.groups[m.Nodes])
	v := float64(ob.groups[m.Nodes].pods)
	m.Charts[2].Current, m.Charts[2].At = &v, ob.at
}

// chartLines are a node group's three charts with their lines from the cache.
func chartLines(t groupTotals) []OverviewChart {
	cores := func(milli int64) float64 { return float64(milli) / 1000 }
	f := func(v float64) *float64 { return &v }
	return []OverviewChart{
		{
			Metric: OverviewCPU, Title: "CPU", Unit: "cores",
			Requests: f(cores(t.requests.CPUMilli)), Limits: f(cores(t.limits.CPUMilli)),
			Allocatable: cores(t.allocatable.CPUMilli), Capacity: cores(t.capacity.CPUMilli),
		},
		{
			Metric: OverviewMemory, Title: "Memory", Unit: "bytes",
			Requests: f(float64(t.requests.MemoryBytes)), Limits: f(float64(t.limits.MemoryBytes)),
			Allocatable: float64(t.allocatable.MemoryBytes), Capacity: float64(t.capacity.MemoryBytes),
		},
		{Metric: OverviewPods, Title: "Pods", Unit: "pods", Allocatable: float64(t.allocPods), Capacity: float64(t.capaPods)},
	}
}

// usageValue is a chart's value of a usage: cores or bytes.
func usageValue(metric string, u model.Usage) (float64, bool) {
	if metric == OverviewCPU {
		return float64(u.CPUMilli) / 1000, u.HasCPU()
	}
	return float64(u.MemoryBytes), u.HasMemory()
}
