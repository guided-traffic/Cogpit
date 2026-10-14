package engine

import (
	"slices"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// A46: the drawer tells the metrics hub every visible chart's range, the default hour only where a
// chart shows it, so that charts far apart get query sets of their own and an hour nobody shows
// costs nothing.
func TestDrawerPassesEveryChartsRange(t *testing.T) {
	pods, _ := kinds.ByKey("pods")
	nodes, _ := kinds.ByKey("nodes")
	const week = 7 * 24 * time.Hour
	for _, tc := range []struct {
		name    string
		profile *model.KindProfile
		windows map[string]time.Duration
		names   []string
		known   bool
		hidden  MetricSet
		want    []time.Duration
	}{
		{"a node's only chart at a week", nodes, map[string]time.Duration{"": week}, []string{""}, true, 0, []time.Duration{week}},
		{"a node at the default", nodes, nil, []string{""}, true, 0, []time.Duration{time.Hour}},
		{"a pod at a week, its container at the default", pods, map[string]time.Duration{"": week}, []string{"", "app"}, true, 0, []time.Duration{time.Hour, week}},
		{"every chart at a week", pods, map[string]time.Duration{"": week, "app": week}, []string{"", "app"}, true, 0, []time.Duration{week}},
		{"the containers' charts hidden", pods, map[string]time.Duration{"": week}, []string{"", "app"}, true, HideContainers, []time.Duration{week}},
		{"a pod the cache does not hold yet", pods, map[string]time.Duration{"": week}, []string{""}, false, 0, []time.Duration{time.Hour, week}},
		{"a range set for a container not listed yet", pods, map[string]time.Duration{"": week, "late": 5 * time.Minute}, []string{"", "app"}, true, 0, []time.Duration{5 * time.Minute, time.Hour, week}},
	} {
		d := &drawerWatch{backend: &Backend{}, profile: tc.profile, windows: tc.windows, hidden: tc.hidden}
		if got := d.windowListLocked(tc.names, tc.known); !slices.Equal(got, tc.want) {
			t.Errorf("%s: %v, want %v", tc.name, got, tc.want)
		}
	}
}
