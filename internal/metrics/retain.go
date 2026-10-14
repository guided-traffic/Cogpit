package metrics

import (
	"slices"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// History that outlives a closed drawer (G12): the Metrics Server keeps no history, so a chart
// that only it feeds started empty every time its drawer opened. When the last subscription of an
// object scope is released, the hub keeps the chart's CPU and memory series (the object's and each
// container's) for a while, and a subscription of the same scope takes them back: reopening a
// pod's drawer shows its last minutes again, then the new measurements after a gap (which a
// Prometheus, where there is one, fills once). The network and the filesystem are not kept (only
// Prometheus has them, and it is asked again), nor what history the chart held (asked again too).
//
// Bounded three ways (Cogpit's defaults, not the owner's decision): each series keeps what lies
// within RetainAge of its newest point; a kept chart whose newest point is older than RetainAge
// when its drawer opens again is dropped; and the store holds at most RetainObjects charts and
// RetainPoints points in all, the least recently closed dropped first (LRU). At the defaults the
// store's samples take at most 50,000 × 48 bytes, about 2.3 MiB (TestRetainedHistoryIsBounded
// measures the heap).

// Defaults of the retention (Options.RetainObjects, RetainAge, RetainPoints).
const (
	// DefaultRetainObjects is how many closed charts the hub keeps: a user's working set of pods
	// and nodes over an hour.
	DefaultRetainObjects = 50
	// DefaultRetainAge is how far back a kept series reaches, and how long a kept chart is good
	// for: the default range of a chart.
	DefaultRetainAge = time.Hour
	// DefaultRetainPoints bounds the points of all kept charts together: an hour of the Metrics
	// Server's 15-second measurements is 240 points per series, so a pod with three containers
	// (four series) takes 960 and 50 such pods 48,000.
	DefaultRetainPoints = 50_000
)

// retainedChart is a closed drawer's chart: its CPU and memory series by target, and the time of
// its newest point.
type retainedChart struct {
	usage  map[string][]model.UsageSample
	newest time.Time
	points int
}

// retainLocked keeps a released chart's usage series, each cut to RetainAge back from its own
// newest point and copied (a long-open drawer's series would otherwise keep its whole array), and
// drops the least recently closed charts beyond the bounds. A chart without points is not kept.
func (h *Hub) retainLocked(sc Scope, c *chart) {
	if c == nil || h.opts.RetainObjects < 0 {
		return
	}
	h.forgetRetainedLocked(sc)
	r := &retainedChart{usage: make(map[string][]model.UsageSample, len(c.usage))}
	for name, pts := range c.usage {
		pts = bound(pts, h.opts.RetainAge, 0, usageAt)
		if len(pts) == 0 {
			continue
		}
		r.usage[name] = slices.Clone(pts)
		r.points += len(pts)
		if last := pts[len(pts)-1].At; last.After(r.newest) {
			r.newest = last
		}
	}
	if r.points == 0 || r.points > h.opts.RetainPoints {
		return
	}
	if h.retained == nil {
		h.retained = map[Scope]*retainedChart{}
	}
	h.retained[sc] = r
	h.retainOrder = append(h.retainOrder, sc)
	h.retainPoints += r.points
	for len(h.retainOrder) > 0 && (len(h.retainOrder) > h.opts.RetainObjects || h.retainPoints > h.opts.RetainPoints) {
		h.forgetRetainedLocked(h.retainOrder[0])
	}
}

// forgetRetainedLocked drops a scope's kept chart, if there is one.
func (h *Hub) forgetRetainedLocked(sc Scope) *retainedChart {
	r := h.retained[sc]
	if r == nil {
		return nil
	}
	delete(h.retained, sc)
	h.retainPoints -= r.points
	if i := slices.Index(h.retainOrder, sc); i >= 0 {
		h.retainOrder = slices.Delete(h.retainOrder, i, i+1)
	}
	return r
}

// restoreLocked is the chart a new subscription of an object scope starts with: the kept one,
// when its newest point is within RetainAge of now, else a new chart. The kept chart leaves the
// store either way: it is the scope's own again.
func (h *Hub) restoreLocked(sc Scope) *chart {
	c := newChart()
	r := h.forgetRetainedLocked(sc)
	if r == nil || h.opts.Now().Sub(r.newest) > h.opts.RetainAge {
		return c
	}
	c.usage = r.usage
	c.at = r.newest
	return c
}

// Retained reports how many closed charts the hub keeps and their points in all.
func (h *Hub) Retained() (charts, points int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.retained), h.retainPoints
}
