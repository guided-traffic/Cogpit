package fyneui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The drawer's Metrics section (G12, architecture §9), after Lens's (capture, 2026-10-12): above the
// Properties of a pod or node, a line naming the source ("Displaying metrics from Kubernetes
// Metrics Server"), a time range (1h at first), a button for each variant (CPU, Memory, Network,
// Filesystem), a flag button that marks the object's events on the chart, and the chart with the
// requests and limits as dashed lines. Network and Filesystem need Prometheus and are disabled
// with that reason. Each container of a pod gets a compact chart of its own under its section's
// title, following the same variant and range.
//
// The panel keeps its widgets across the drawer's rebuilds (drawer.kept) and updates them in place
// when the engine reports new numbers (engine.EventMetrics, about every 15 seconds while the drawer
// is open), so a new measurement redraws the chart and nothing else.

const (
	metricsKey = "metrics"

	variantCPU        = "cpu"
	variantMemory     = "memory"
	variantNetwork    = "network"
	variantFilesystem = "filesystem"

	needsPrometheus = "needs Prometheus"
)

// metricsRanges are the time ranges the panel offers, by label. The Metrics Server keeps no
// history, so the hub's series (an hour at most) is what a range shows; Prometheus will add the
// longer ranges.
var metricsRanges = []struct {
	label string
	d     time.Duration
}{
	{"5m", 5 * time.Minute}, {"15m", 15 * time.Minute}, {"30m", 30 * time.Minute}, {"1h", time.Hour},
}

// defaultMetricsRange is Lens's default time range.
const defaultMetricsRange = "1h"

type metricsPanel struct {
	*chartCtl // the pod's or node's own chart and controls

	d   *drawer
	ref model.ObjectRef

	data   *engine.ObjectMetrics
	events []engine.EventItem
	loads  int
	closed bool

	root    *fyne.Container
	caption *textLine
	note    *textLine
	perBox  map[string]*chartCtl // each container's compact chart and controls, by container name
}

// chartCtl is one chart with its own row of controls: the time range, the variant, the events
// flag. Its selection is its own, so changing one chart changes only that chart; the panel's data
// (one poll per drawer) feeds them all. It lives as long as the panel, so its selection survives
// the drawer's rebuilds and the polls.
type chartCtl struct {
	p       *metricsPanel
	name    string // the container's name; "" for the pod or node
	compact bool

	variant    string
	window     time.Duration
	showEvents bool

	root     *fyne.Container // a container's controls and chart; nil for the pod's, which the panel's root holds
	controls *fyne.Container
	rangeSel *widget.Select // the pod's; a container's chart has rangeBtn
	rangeBtn *tipButton
	variants map[string]*tipButton
	flag     *tipButton
	plot     *chart
}

// metricsVariants are the buttons of every chart; short is the label on a compact chart's small
// buttons (the tooltip has the full name).
var metricsVariants = []struct{ id, label, short string }{
	{variantCPU, "CPU", "CPU"}, {variantMemory, "Memory", "Mem"}, {variantNetwork, "Network", "Net"}, {variantFilesystem, "Filesystem", "FS"},
}

func newMetricsPanel(d *drawer, ref model.ObjectRef) *metricsPanel {
	p := &metricsPanel{d: d, ref: ref, perBox: map[string]*chartCtl{}}
	p.chartCtl = newChartCtl(p, "", false)
	p.caption = newTextLine("Loading metrics…", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	p.caption.truncate = true
	p.note = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	p.note.truncate = true
	p.root = container.NewVBox(p.controls, p.plot, p.caption, p.note)
	p.render()
	return p
}

// newChartCtl makes a chart with its controls: CPU selected, the default range, events off.
func newChartCtl(p *metricsPanel, name string, compact bool) *chartCtl {
	c := &chartCtl{p: p, name: name, compact: compact, variant: variantCPU, window: time.Hour, variants: map[string]*tipButton{}}
	sh := p.d.sh

	// The pod's chart picks its range in a select; a container's compact chart in a small button
	// that opens a menu, so that its whole row of controls fits the drawer's minimum width.
	ctl := make([]fyne.CanvasObject, 0, 3+len(metricsVariants))
	if compact {
		c.rangeBtn = newTipButton(sh, nil, "Time range", nil)
		c.rangeBtn.Text, c.rangeBtn.Importance = defaultMetricsRange, widget.MediumImportance
		c.rangeBtn.OnTapped = func() { c.rangeMenu() }
		ctl = append(ctl, c.rangeBtn)
	} else {
		labels := make([]string, len(metricsRanges))
		for i, r := range metricsRanges {
			labels[i] = r.label
		}
		c.rangeSel = widget.NewSelect(labels, c.setRange)
		c.rangeSel.Selected = defaultMetricsRange
		ctl = append(ctl, c.rangeSel)
	}
	for _, v := range metricsVariants {
		id := v.id
		b := newTipButton(sh, nil, v.label, func() { c.setVariant(id) })
		b.Text = v.label
		if compact {
			b.Text = v.short
		}
		c.variants[id] = b
		ctl = append(ctl, b)
	}
	c.variants[variantNetwork].tip = "Network: " + needsPrometheus
	c.variants[variantFilesystem].tip = "Filesystem: " + needsPrometheus
	c.variants[variantNetwork].Disable()
	c.variants[variantFilesystem].Disable()
	c.variants[variantCPU].tip = "CPU usage"
	c.variants[variantMemory].tip = "Memory usage"

	c.flag = newTipButton(sh, flagIcon, "Show events", func() {
		c.showEvents = !c.showEvents
		c.p.render()
	})
	if !compact {
		ctl = append(ctl, newDivider(true))
	}
	ctl = append(ctl, c.flag)
	c.controls = box(&flowLayout{avail: p.d.contentWidth}, ctl...)
	c.plot = newChart()
	return c
}

// close stops the panel's loads from landing (keptCloser).
func (p *metricsPanel) close() { p.closed = true }

// setRange sets this chart's time range by its label.
func (c *chartCtl) setRange(label string) {
	for _, r := range metricsRanges {
		if r.label == label {
			c.window = r.d
		}
	}
	if c.rangeSel != nil && c.rangeSel.Selected != label {
		c.rangeSel.Selected = label
		c.rangeSel.Refresh()
	}
	if c.rangeBtn != nil {
		c.rangeBtn.Text = label
		c.rangeBtn.Refresh()
	}
	c.p.render()
}

// rangeMenu opens the menu of ranges under a compact chart's range button.
func (c *chartCtl) rangeMenu() {
	items := make([]*fyne.MenuItem, len(metricsRanges))
	for i, r := range metricsRanges {
		label := r.label
		items[i] = fyne.NewMenuItem(label, func() { c.setRange(label) })
		items[i].Checked = r.d == c.window
	}
	if cnv := fyne.CurrentApp().Driver().CanvasForObject(c.rangeBtn); cnv != nil {
		widget.ShowPopUpMenuAtRelativePosition(fyne.NewMenu("", items...), cnv, fyne.NewPos(0, c.rangeBtn.Size().Height), c.rangeBtn)
	}
}

// setVariant switches this chart to CPU or Memory.
func (c *chartCtl) setVariant(id string) {
	if id == c.variant {
		return
	}
	c.variant = id
	c.p.render()
}

// setEvents hands the panel the object's events, whose times the flag button marks.
func (p *metricsPanel) setEvents(ev []engine.EventItem) {
	if slices.Equal(ev, p.events) {
		return
	}
	p.events = ev
	for _, c := range p.charts() {
		if c.showEvents {
			c.update()
		}
	}
}

// charts are the pod's or node's chart and the containers'.
func (p *metricsPanel) charts() []*chartCtl {
	out := make([]*chartCtl, 1, 1+len(p.perBox))
	out[0] = p.chartCtl
	for _, c := range p.perBox {
		out = append(out, c)
	}
	return out
}

// refresh reads the engine's current numbers off the main goroutine and renders them.
func (p *metricsPanel) refresh() {
	p.loads++
	n, ref, sh := p.loads, p.ref, p.d.sh
	go func() {
		defer sh.recoverTo("metrics")
		m, err := sh.eng.ObjectMetrics(ref)
		sh.do(func() {
			if p.closed || n != p.loads {
				return
			}
			p.apply(m, err)
		})
	}()
}

func (p *metricsPanel) apply(m *engine.ObjectMetrics, err error) {
	if err != nil || m == nil {
		m = &engine.ObjectMetrics{Ref: p.ref, Status: metrics.StatusUnavailable, Message: "could not read the metrics"}
		if err != nil {
			m.Message = err.Error()
		}
	}
	hadBoxes := len(p.containerNames())
	p.data = m
	p.render()
	if len(p.containerNames()) != hadBoxes {
		p.d.rebuild() // the containers' charts appear with the first numbers
	}
}

// containerNames are the containers the panel has numbers for.
func (p *metricsPanel) containerNames() []string {
	var names []string
	if p.data != nil && p.data.Status == metrics.StatusAvailable {
		for _, t := range p.data.Targets {
			if t.Name != "" {
				names = append(names, t.Name)
			}
		}
	}
	return names
}

// containerChart returns the chart of a container with its own controls, or nil without numbers
// for it.
func (p *metricsPanel) containerChart(name string) fyne.CanvasObject {
	if !slices.Contains(p.containerNames(), name) {
		return nil
	}
	c := p.perBox[name]
	if c == nil {
		c = newChartCtl(p, name, true)
		c.root = container.NewVBox(c.controls, c.plot)
		p.perBox[name] = c
		c.update()
	}
	return c.root
}

// render draws the state: the source line or the reason there is none, and the charts.
func (p *metricsPanel) render() {
	m := p.data
	switch {
	case m == nil:
		p.setState(true, "Loading metrics…")
	case m.Status == metrics.StatusAvailable:
		p.setState(true, "Displaying metrics from "+m.Source)
	case m.Status == metrics.StatusUnknown:
		p.setState(true, "Waiting for the first measurement…")
	case m.Status == metrics.StatusNotServed:
		// A cluster without metrics-server or Prometheus is a state, not a problem: said quietly.
		p.setState(false, "No metrics: "+m.Message)
		p.caption.SetColor(colorMuted)
	default:
		p.setState(false, "Metrics are not available: "+m.Message)
	}
	if m != nil && m.Status == metrics.StatusAvailable {
		var note string
		if t := p.target(""); t != nil && t.Current == nil {
			note = "No metrics for this " + singular(p.d.sh.kindTitle(p.ref.Kind)) + " yet."
		} else if t != nil && !p.coversWindow(t) {
			note = "The Metrics Server keeps no history: the chart starts when this drawer opened."
		}
		p.note.SetText(note)
	}
	for _, c := range p.charts() {
		c.update()
	}
	p.root.Refresh()
}

// setState shows or hides the controls and the chart, and sets the caption; without charts there is
// no note either.
func (p *metricsPanel) setState(charts bool, caption string) {
	for _, o := range []fyne.CanvasObject{p.controls, p.plot} {
		if charts {
			o.Show()
		} else {
			o.Hide()
		}
	}
	if !charts {
		p.caption.SetColor(colorStatusWarning)
	} else {
		p.caption.SetColor(colorMuted)
	}
	p.caption.SetText(caption)
	if !charts {
		p.note.SetText("")
	}
}

func (p *metricsPanel) target(name string) *engine.MetricsTarget {
	if p.data == nil {
		return nil
	}
	for i := range p.data.Targets {
		if p.data.Targets[i].Name == name {
			return &p.data.Targets[i]
		}
	}
	return nil
}

// coversWindow reports whether the series reaches back over the chosen range.
func (p *metricsPanel) coversWindow(t *engine.MetricsTarget) bool {
	if len(t.Samples) == 0 {
		return false
	}
	return p.d.sh.now().Sub(t.Samples[0].At) >= p.window
}

// update shows the chart's selection on its buttons and fills the chart.
func (c *chartCtl) update() {
	for id, b := range c.variants {
		b.Importance = widget.MediumImportance
		if id == c.variant && id != variantNetwork && id != variantFilesystem {
			b.Importance = widget.HighImportance
		}
		b.Refresh()
	}
	c.flag.Importance = widget.LowImportance
	c.flag.tip = "Show events"
	if c.showEvents {
		c.flag.Importance = widget.HighImportance
		c.flag.tip = "Hide events"
	}
	c.flag.Refresh()
	c.renderTarget()
	if c.root != nil {
		c.root.Refresh()
	}
}

// renderTarget fills the chart with its target's series, lines and the events.
func (c *chartCtl) renderTarget() {
	p := c.p
	to := p.d.sh.now()
	d := chartData{From: to.Add(-c.window), To: to, Compact: c.compact, Loc: p.d.sh.loc(), Empty: "No data yet"}
	cpu := c.variant == variantCPU
	if cpu {
		d.Format = formatCores
	} else {
		d.Format = formatBytesShort
	}
	value := func(u model.Usage) float64 {
		if cpu {
			return float64(u.CPUMilli) / 1000
		}
		return float64(u.MemoryBytes)
	}
	t := p.target(c.name)
	if t == nil || p.data == nil || p.data.Status != metrics.StatusAvailable {
		c.plot.Set(d)
		return
	}
	title := "Memory usage"
	if cpu {
		title = "CPU usage"
	}
	s := chartSeries{Name: title, Color: colorAccent}
	for _, sm := range t.Samples {
		s.Points = append(s.Points, chartPoint{At: sm.At, Value: value(sm.Usage)})
	}
	if len(s.Points) == 0 && t.Current != nil {
		s.Points = append(s.Points, chartPoint{At: t.At, Value: value(*t.Current)})
	}
	d.Series = []chartSeries{s}
	if p.ref.Kind == string(model.UsageNodes) {
		d.Lines = []chartLine{
			{Name: "Allocatable capacity", Color: colorStatusWarning, Value: value(t.Allocatable)},
			{Name: "Capacity", Color: colorStatusError, Value: value(t.Capacity)},
		}
	} else {
		d.Lines = []chartLine{
			{Name: "Requests", Color: colorStatusWarning, Value: value(t.Requests)},
			{Name: "Limits", Color: colorStatusError, Value: value(t.Limits)},
		}
	}
	if c.showEvents {
		for _, e := range p.events {
			if e.LastSeen > 0 {
				d.Markers = append(d.Markers, chartMarker{At: time.Unix(e.LastSeen, 0), Name: "Events", Color: colorStatusPending})
			}
		}
	}
	c.plot.Set(d)
}

// flagIcon is the events toggle's icon: a flag, as Lens's "Show events" button has (a Material
// glyph, drawn in the theme's foreground like Fyne's own icons).
var flagIcon = theme.NewThemedResource(fyne.NewStaticResource("flag.svg", []byte(
	`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M14.4 6 14 4H5v17h2v-7h5.6l.4 2h7V6z"/></svg>`)))

// loc is the time zone of the chart's time axis: the clock's.
func (sh *shell) loc() *time.Location { return sh.now().Location() }

// formatCores formats cores for the value axis: "0.25", "1.5", "2".
func formatCores(v float64) string { return fmt.Sprintf("%.3g", v) }

// formatBytesShort formats bytes for the value axis in binary units without the "B": "64Mi", "1.5Gi".
func formatBytesShort(v float64) string {
	if v < 1024 {
		return fmt.Sprintf("%.3g", v)
	}
	units := []string{"Ki", "Mi", "Gi", "Ti", "Pi"}
	i := -1
	for v >= 1024 && i < len(units)-1 {
		v /= 1024
		i++
	}
	return fmt.Sprintf("%.3g%s", v, units[i])
}

// metricsPanel returns the drawer's Metrics panel for the shown object, made on first use; nil for
// a kind without a Metrics section.
func (d *drawer) metricsPanel(m *engine.DetailModel) *metricsPanel {
	if !m.Metrics {
		return nil
	}
	p, ok := d.kept[metricsKey].(*metricsPanel)
	if !ok || p.ref != m.Ref {
		p = newMetricsPanel(d, m.Ref)
		d.kept[metricsKey] = p
		p.refresh()
	}
	p.setEvents(m.Events)
	return p
}

// metricsChanged runs when the engine reports new metrics for the open drawer.
func (d *drawer) metricsChanged() {
	if p, ok := d.kept[metricsKey].(*metricsPanel); ok {
		p.refresh()
	}
}

// metricsSection returns the objects of the Metrics section: its title, the divider and the panel.
func (d *drawer) metricsSection(p *metricsPanel) []fyne.CanvasObject {
	return []fyne.CanvasObject{
		d.line("Metrics", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true}, false, ""),
		newDivider(false), p.root,
	}
}

// flowLayout lays its objects out left to right and starts a new line when the next one does not
// fit in the available width, so that a row of buttons never asks for more than the drawer has.
type flowLayout struct {
	avail func() float32
}

func (l *flowLayout) width(size fyne.Size) float32 {
	if size.Width > 0 {
		return size.Width
	}
	if l.avail != nil {
		return l.avail()
	}
	return 0
}

// place returns each visible object's position for a width, and the height used.
func (l *flowLayout) place(objs []fyne.CanvasObject, width float32) ([]fyne.Position, float32) {
	gap := fyne.CurrentApp().Settings().Theme().Size(theme.SizeNamePadding)
	pos := make([]fyne.Position, len(objs))
	var x, y, lineH float32
	for i, o := range objs {
		if !o.Visible() {
			continue
		}
		ms := o.MinSize()
		if x > 0 && x+ms.Width > width {
			x, y, lineH = 0, y+lineH+gap, 0
		}
		pos[i] = fyne.NewPos(x, y)
		x += ms.Width + gap
		lineH = fyne.Max(lineH, ms.Height)
	}
	return pos, y + lineH
}

// MinSize implements fyne.Layout: the widest object, and the height of the lines that fit the
// available width.
func (l *flowLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var w float32
	for _, o := range objs {
		if o.Visible() {
			w = fyne.Max(w, o.MinSize().Width)
		}
	}
	_, h := l.place(objs, fyne.Max(l.width(fyne.Size{}), w))
	return fyne.NewSize(w, h)
}

// Layout implements fyne.Layout.
func (l *flowLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	pos, _ := l.place(objs, l.width(size))
	for i, o := range objs {
		if o.Visible() {
			o.Move(pos[i])
			o.Resize(o.MinSize())
		}
	}
}

// containerTitleHasChart reports whether a drawer section is a container whose metrics the panel
// has: the sections of the group "Containers" are titled by the container's name.
func containerTitleHasChart(sec model.DetailSection) bool {
	return sec.Group == "Containers" && strings.TrimSpace(sec.Title) != ""
}
