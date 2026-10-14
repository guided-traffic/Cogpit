package fyneui

import (
	"image/color"
	"math"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// The Nodes list's usage bars (G12; resource-parity › Nodes, captured: "CPU, Memory, Disk (usage
// bars)"): the share as text, a bar filled to it, coloured at the thresholds; a node without a
// measurement has an empty cell, never 0%.

// barNodeUsage are the loads the bar tests give the fixture's nodes (allocatable: 7.8 cores and
// 30.2 GiB): a quiet node, one at the warning threshold, one past the error threshold, one without
// a measurement; the others in between.
var barNodeUsage = map[string]model.Usage{
	"control-plane-1": {CPUMilli: 936, MemoryBytes: 11 << 30},           // 12%, 36%
	"worker-a-01":     {CPUMilli: 6474, MemoryBytes: 25 << 30},          // 83%, 83%
	"worker-a-02":     {CPUMilli: 7410, MemoryBytes: 29 << 30},          // 95%, 96%
	"worker-a-04":     {CPUMilli: 2340, MemoryBytes: 8 << 30},           // 30%, 26%
	"worker-b-01":     {CPUMilli: 39, MemoryBytes: 3 << 30},             // 0.5%, 9.9%
	"worker-b-03":     {CPUMilli: 4290, MemoryBytes: 18 << 30},          // 55%, 60%
	"worker-c-01":     {CPUMilli: 5850, MemoryBytes: 2 << 30},           // 75%, 6.6%
	"worker-b-02":     {MemoryBytes: 30 << 30, Missing: model.UsageCPU}, // CPU not measured, 99%
	// worker-a-03: no measurement at all
}

// withNodeBars fills the fixture Nodes view's CPU and Memory cells as the engine's row cache does,
// from barNodeUsage.
func withNodeBars(t testing.TB, f *fakeEngine) {
	t.Helper()
	p, _ := kinds.ByKey("nodes")
	objs := map[string]*unstructured.Unstructured{}
	for _, o := range fixtureNodeObjects() {
		objs[o.GetName()] = o
	}
	snap := *f.tables["nodes"]
	snap.Rows = slices.Clone(snap.Rows)
	cpu, mem := p.ColumnIndex(model.ColumnCPU), p.ColumnIndex(model.ColumnMemory)
	for i, r := range snap.Rows {
		u, ok := barNodeUsage[r.Ref.Name]
		if !ok {
			continue
		}
		c, m := p.UsageCells(objs[r.Ref.Name], u)
		cells := slices.Clone(r.Cells)
		cells[cpu], cells[mem] = c, m
		snap.Rows[i].Cells = cells
	}
	f.tables["nodes"] = &snap
}

func TestNodesListDrawsUsageBars(t *testing.T) {
	f := kindsFake(t, "nodes")
	withNodeBars(t, f)
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the nodes view", func() bool { return sh.view.snap != nil && sh.view.snap.View == "nodes" && sh.diag.total > 0 })
	settle(w)
	p, _ := kinds.ByKey("nodes")
	cpuCol := firstDataColumn + p.ColumnIndex(model.ColumnCPU)
	memCol := firstDataColumn + p.ColumnIndex(model.ColumnMemory)
	row := func(name string) int { return rowOf(t, f, "nodes", name) }
	fillOf := func(c *tableCell) color.Color { return c.bar.fill.FillColor }
	check := func(name string, col int, share float64, want fyne.ThemeColorName) {
		t.Helper()
		c := sh.view.cellAt(t, row(name), col)
		if c.bar == nil || !c.bar.shown || !c.bar.track.Visible() {
			t.Fatalf("%s: no bar shown in %q", name, c.full)
		}
		tw, fw := c.bar.track.Size().Width, c.bar.fill.Size().Width
		if math.Abs(float64(fw)-float64(tw)*math.Min(share, 1)) > 1.5 {
			t.Errorf("%s: fill %v of a track of %v, want the share %.3f", name, fw, tw, share)
		}
		if got := fillOf(c); got != themeColor(c.Theme(), want) {
			t.Errorf("%s: bar colour %v, want %s", name, got, want)
		}
		// The share's text sits right-aligned before the bar.
		if tx := c.text.Position().X + c.text.Size().Width; tx > c.bar.track.Position().X {
			t.Errorf("%s: the text ends at %v, after the bar's start %v", name, tx, c.bar.track.Position().X)
		}
	}
	check("control-plane-1", cpuCol, 936.0/7800, colorAccent)
	check("worker-a-01", cpuCol, 6474.0/7800, statusColorName(model.StatusWarning))
	check("worker-a-02", cpuCol, 7410.0/7800, statusColorName(model.StatusError))
	check("worker-b-02", memCol, float64(30<<30)/float64(31715736<<10), statusColorName(model.StatusError))

	// The bars of a column line up: every share's slot is as wide.
	a, b := sh.view.cellAt(t, row("control-plane-1"), cpuCol), sh.view.cellAt(t, row("worker-a-02"), cpuCol)
	if a.bar.track.Position().X != b.bar.track.Position().X {
		t.Errorf("bars start at %v and %v, want one x", a.bar.track.Position().X, b.bar.track.Position().X)
	}
	// No measurement, or a part not measured: an empty cell and no bar, never 0%.
	for _, c := range []*tableCell{sh.view.cellAt(t, row("worker-a-03"), cpuCol), sh.view.cellAt(t, row("worker-b-02"), cpuCol)} {
		if c.full != "" || (c.bar != nil && (c.bar.track.Visible() || c.bar.fill.Visible())) {
			t.Errorf("a cell without a measurement shows %q and a bar", c.full)
		}
	}
	// The tooltip names the amounts behind the bar.
	if got := sh.view.cellAt(t, row("worker-a-02"), cpuCol).tipText(); got != "7.410 of 7.800 cores (95%)" {
		t.Errorf("tooltip = %q", got)
	}
}

func renderNodeBars(t *testing.T, mode ThemeMode, name string) {
	f := kindsFake(t, "nodes")
	withNodeBars(t, f)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the nodes view", func() bool {
		return len(sh.kinds) > 0 && sh.view.snap != nil && sh.view.snap.View == "nodes" && sh.diag.total > 0
	})
	settle(w)
	savePNG(t, w, name)
}

func TestRenderNodesUsageBarsDark(t *testing.T) {
	renderNodeBars(t, ThemeDark, "nodes_usage_bars_dark.png")
}

func TestRenderNodesUsageBarsLight(t *testing.T) {
	renderNodeBars(t, ThemeLight, "nodes_usage_bars_light.png")
}
