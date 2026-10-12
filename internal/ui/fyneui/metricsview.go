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
	d   *drawer
	ref model.ObjectRef

	data       *engine.ObjectMetrics
	events     []engine.EventItem
	variant    string
	window     time.Duration
	showEvents bool
	loads      int
	closed     bool

	root     *fyne.Container
	controls *fyne.Container
	rangeSel *widget.Select
	variants map[string]*tipButton
	flag     *tipButton
	plot     *chart
	caption  *textLine
	note     *textLine
	perBox   map[string]*chart // each container's compact chart, by container name
}

var metricsVariants = []struct{ id, label string }{
	{variantCPU, "CPU"}, {variantMemory, "Memory"}, {variantNetwork, "Network"}, {variantFilesystem, "Filesystem"},
}

func newMetricsPanel(d *drawer, ref model.ObjectRef) *metricsPanel {
	p := &metricsPanel{d: d, ref: ref, variant: variantCPU, window: time.Hour, variants: map[string]*tipButton{}, perBox: map[string]*chart{}}
	sh := d.sh

	labels := make([]string, len(metricsRanges))
	for i, r := range metricsRanges {
		labels[i] = r.label
	}
	p.rangeSel = widget.NewSelect(labels, func(l string) {
		for _, r := range metricsRanges {
			if r.label == l {
				p.window = r.d
			}
		}
		p.render()
	})
	p.rangeSel.Selected = defaultMetricsRange

	ctl := make([]fyne.CanvasObject, 1, 3+len(metricsVariants))
	ctl[0] = p.rangeSel
	for _, v := range metricsVariants {
		id := v.id
		b := newTipButton(sh, nil, v.label, func() { p.setVariant(id) })
		b.Text = v.label
		p.variants[id] = b
		ctl = append(ctl, b)
	}
	p.variants[variantNetwork].tip = "Network: " + needsPrometheus
	p.variants[variantFilesystem].tip = "Filesystem: " + needsPrometheus
	p.variants[variantNetwork].Disable()
	p.variants[variantFilesystem].Disable()
	p.variants[variantCPU].tip = "CPU usage"
	p.variants[variantMemory].tip = "Memory usage"

	p.flag = newTipButton(sh, flagIcon, "Show events", func() {
		p.showEvents = !p.showEvents
		p.render()
	})
	ctl = append(ctl, newDivider(true), p.flag)
	p.controls = box(&flowLayout{avail: d.contentWidth}, ctl...)

	p.plot = newChart()
	p.caption = newTextLine("Loading metrics…", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	p.caption.truncate = true
	p.note = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	p.note.truncate = true
	p.root = container.NewVBox(p.controls, p.plot, p.caption, p.note)
	p.render()
	return p
}

// close stops the panel's loads from landing (keptCloser).
func (p *metricsPanel) close() { p.closed = true }

// setVariant switches the chart to CPU or Memory.
func (p *metricsPanel) setVariant(id string) {
	if id == p.variant {
		return
	}
	p.variant = id
	p.render()
}

// setEvents hands the panel the object's events, whose times the flag button marks.
func (p *metricsPanel) setEvents(ev []engine.EventItem) {
	if slices.Equal(ev, p.events) {
		return
	}
	p.events = ev
	if p.showEvents {
		p.render()
	}
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

// containerChart returns the compact chart of a container, or nil without numbers for it.
func (p *metricsPanel) containerChart(name string) fyne.CanvasObject {
	if !slices.Contains(p.containerNames(), name) {
		return nil
	}
	c := p.perBox[name]
	if c == nil {
		c = newChart()
		p.perBox[name] = c
		p.renderTarget(c, name, true)
	}
	return c
}

// render draws the state: the source line or the reason there is none, and the charts.
func (p *metricsPanel) render() {
	for id, b := range p.variants {
		b.Importance = widget.MediumImportance
		if id == p.variant && id != variantNetwork && id != variantFilesystem {
			b.Importance = widget.HighImportance
		}
		b.Refresh()
	}
	if p.showEvents {
		p.flag.Importance = widget.HighImportance
	} else {
		p.flag.Importance = widget.LowImportance
	}
	p.flag.tip = "Show events"
	if p.showEvents {
		p.flag.tip = "Hide events"
	}
	p.flag.Refresh()

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
	p.renderTarget(p.plot, "", false)
	for name, c := range p.perBox {
		p.renderTarget(c, name, true)
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

// renderTarget fills a chart with one target's series, lines and the events.
func (p *metricsPanel) renderTarget(c *chart, name string, compact bool) {
	to := p.d.sh.now()
	d := chartData{From: to.Add(-p.window), To: to, Compact: compact, Loc: p.d.sh.loc(), Empty: "No data yet"}
	cpu := p.variant == variantCPU
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
	t := p.target(name)
	if t == nil || p.data == nil || p.data.Status != metrics.StatusAvailable {
		c.Set(d)
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
	if p.showEvents {
		for _, e := range p.events {
			if e.LastSeen > 0 {
				d.Markers = append(d.Markers, chartMarker{At: time.Unix(e.LastSeen, 0), Name: "Events", Color: colorStatusPending})
			}
		}
	}
	c.Set(d)
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
