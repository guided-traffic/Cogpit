package fyneui

import (
	"slices"
	"time"
)

// A chart's line breaks at a gap (G12): since a closed drawer's chart is kept (metrics, retain.go),
// a reopened drawer shows the minutes it measured before, then nothing for the time it was closed,
// then the new measurements. A line drawn straight across that time would show values nobody
// measured. A gap is a step between two points longer than four times the series' usual step (the
// median of its steps) and two minutes: a missed measurement or two of the Metrics Server's 15
// seconds still joins, a week's history at its ten-minute step joins, a drawer closed for ten
// minutes does not (Cogpit's rule).

// chartGapFloor is the shortest step between two points that can be a gap.
const chartGapFloor = 2 * time.Minute

// gapLimit is the longest step between two points of a series that still joins them; 0 when the
// series is too short to know its step (every step joins).
func gapLimit(pts []chartPoint) time.Duration {
	if len(pts) < 3 {
		return 0
	}
	steps := make([]time.Duration, 0, len(pts)-1)
	for i := 1; i < len(pts); i++ {
		if d := pts[i].At.Sub(pts[i-1].At); d > 0 {
			steps = append(steps, d)
		}
	}
	if len(steps) == 0 {
		return 0
	}
	slices.Sort(steps)
	return max(4*steps[len(steps)/2], chartGapFloor)
}

// seriesRuns are the runs of a series' points drawn as one line, as [start, end) indexes: the points
// within from and to, split where a step is a gap (gapLimit).
func seriesRuns(pts []chartPoint, from, to time.Time) [][2]int {
	limit := gapLimit(pts)
	var runs [][2]int
	start := -1
	flush := func(end int) {
		if start >= 0 {
			runs = append(runs, [2]int{start, end})
			start = -1
		}
	}
	for i, p := range pts {
		if p.At.Before(from) || p.At.After(to) {
			flush(i)
			continue
		}
		if start >= 0 && limit > 0 && p.At.Sub(pts[i-1].At) > limit {
			flush(i)
		}
		if start < 0 {
			start = i
		}
	}
	flush(len(pts))
	return runs
}
