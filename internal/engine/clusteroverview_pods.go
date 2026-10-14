package engine

import (
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The cluster Overview's Pods chart has history (G12) where Prometheus has kube-state-metrics: the
// hub asks for the pods on the group's nodes over the page's range once (metrics.WantPods,
// promclusterpods.go), and the cache's counts continue it from the page's first sample on. The two
// count the same pods (scheduled to a node of the group, not finished), so the seam does not jump.
// Without kube-state-metrics the chart is the cache's alone, as before, and the model says so
// (ClusterOverviewModel.PodsHistory "none").

// addPodsHistory puts Prometheus's pod counts before the cache's first sample in front of the
// chart's samples (within the range when series), and makes the newest count the chart's current
// value while the cache has none yet.
func addPodsHistory(ch *OverviewChart, hist []model.CountSample, cache []OverviewSample, from time.Time, series bool) {
	var cut time.Time // the cache's first sample: the history stops there
	if len(cache) > 0 {
		cut = cache[0].At
	}
	var before []OverviewSample
	for _, s := range hist {
		if !cut.IsZero() && !s.At.Before(cut) {
			break
		}
		before = append(before, OverviewSample{At: s.At, Value: s.Value})
	}
	if len(before) == 0 {
		return
	}
	if ch.Current == nil {
		last := before[len(before)-1]
		v := last.Value
		ch.Current, ch.At = &v, last.At
	}
	if series {
		ch.Samples = append(samplesWithin(before, from), ch.Samples...)
	}
}
