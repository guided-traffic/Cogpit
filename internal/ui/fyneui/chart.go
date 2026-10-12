package fyneui

import (
	"fmt"
	"image/color"
	"math"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// chart is a small line chart for the drawer's Metrics section (architecture §9): time on the
// horizontal axis, a value axis from zero, lines for series (usage), dashed horizontal lines for
// fixed values (requests, limits, allocatable) and vertical markers with a diamond for moments
// (events). It draws what it is given and knows nothing of where the numbers came from, so the
// Prometheus provider's series will use it as they are.
//
// Everything is drawn with canvas lines and text in theme colours; the renderer rebuilds its
// objects on every Refresh and Layout (a chart holds some hundred segments at most).
type chart struct {
	widget.BaseWidget
	data chartData
}

// chartData is what a chart draws.
type chartData struct {
	// From and To are the time axis. A series' points outside it are not drawn.
	From, To time.Time
	// Loc is the time zone of the axis labels; nil means local time.
	Loc *time.Location
	// Series are the lines of measured values over time.
	Series []chartSeries
	// Lines are horizontal lines at a fixed value; Markers are vertical lines at a moment.
	Lines   []chartLine
	Markers []chartMarker
	// Format formats a value for the axis; nil prints the number.
	Format func(float64) string
	// Empty is the text shown in place of the plot while there is no point to draw.
	Empty string
	// Compact leaves out the legend and the horizontal grid's labels get a smaller plot (a
	// container's chart under its section title).
	Compact bool
}

type chartSeries struct {
	Name   string
	Color  fyne.ThemeColorName
	Points []chartPoint
}

type chartPoint struct {
	At    time.Time
	Value float64
}

type chartLine struct {
	Name  string
	Color fyne.ThemeColorName
	Value float64
}

type chartMarker struct {
	At    time.Time
	Name  string
	Color fyne.ThemeColorName
}

// Chart geometry, in Fyne units.
const (
	chartPlotHeight        = 120   // the plot's height
	chartCompactHeight     = 72    // a compact chart's plot
	chartMinWidth          = 120   // the narrowest the chart asks for; the drawer's width is the user's
	chartLineWidth         = 1.5   // a series
	chartRefWidth          = 1     // requests and limits
	chartDash, chartGap    = 6, 4  // a reference line's dashes
	chartDiamond           = 3.5   // half the diamond of an event marker
	chartGridLines         = 3     // the grid's horizontal lines above the axis
	chartPointRadius       = 2.5   // a single point's dot
	chartLegendSwatch      = 8     // the legend's colour square
	chartMaxSegmentsPerSer = 2_000 // never draw more than this per series
)

func newChart() *chart {
	c := &chart{}
	c.ExtendBaseWidget(c)
	return c
}

// Set replaces the chart's data and redraws it.
func (c *chart) Set(d chartData) {
	c.data = d
	c.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (c *chart) CreateRenderer() fyne.WidgetRenderer {
	r := &chartRenderer{c: c}
	r.Refresh()
	return r
}

type chartRenderer struct {
	c    *chart
	objs []fyne.CanvasObject
}

func (r *chartRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *chartRenderer) Destroy()                     {}

func (r *chartRenderer) Refresh() {
	r.Layout(r.c.Size())
	canvas.Refresh(r.c)
}

// metrics are the sizes the layout works from.
type chartMetrics struct {
	pad, text float32
	textH     float32
}

func (r *chartRenderer) sizes() chartMetrics {
	th := r.c.Theme()
	text := th.Size(theme.SizeNameCaptionText)
	return chartMetrics{
		pad: th.Size(theme.SizeNamePadding), text: text,
		textH: fyne.MeasureText("Mg", text, fyne.TextStyle{}).Height,
	}
}

func (r *chartRenderer) plotHeight() float32 {
	if r.c.data.Compact {
		return chartCompactHeight
	}
	return chartPlotHeight
}

func (r *chartRenderer) legendHeight() float32 {
	if r.c.data.Compact || len(r.c.data.Series)+len(r.c.data.Lines)+len(r.c.data.Markers) == 0 {
		return 0
	}
	m := r.sizes()
	return m.textH + m.pad
}

func (r *chartRenderer) MinSize() fyne.Size {
	m := r.sizes()
	return fyne.NewSize(chartMinWidth, m.pad+r.plotHeight()+m.pad+m.textH+r.legendHeight())
}

// valueMax is the top of the value axis: the largest value drawn, rounded up to a tick.
func (d chartData) valueMax() float64 {
	top := 0.0
	for _, s := range d.Series {
		for _, p := range s.Points {
			if !p.At.Before(d.From) && !p.At.After(d.To) {
				top = math.Max(top, p.Value)
			}
		}
	}
	for _, l := range d.Lines {
		top = math.Max(top, l.Value)
	}
	if top <= 0 {
		return 1
	}
	return top
}

// niceStep returns a round step (1, 2, 2.5, 5 times a power of ten) so that about n steps cover top.
func niceStep(top float64, n int) float64 {
	raw := top / float64(n)
	if raw <= 0 {
		return 1
	}
	exp := math.Pow(10, math.Floor(math.Log10(raw)))
	for _, m := range []float64{1, 2, 2.5, 5, 10} {
		if raw <= m*exp {
			return m * exp
		}
	}
	return 10 * exp
}

func (d chartData) format(v float64) string {
	if d.Format != nil {
		return d.Format(v)
	}
	return fmt.Sprintf("%g", v)
}

// timeLabel formats an axis time: with seconds on a short range.
func (d chartData) timeLabel(t time.Time) string {
	loc := d.Loc
	if loc == nil {
		loc = time.Local
	}
	if d.To.Sub(d.From) < 10*time.Minute {
		return t.In(loc).Format("15:04:05")
	}
	return t.In(loc).Format("15:04")
}

func (r *chartRenderer) Layout(size fyne.Size) {
	th := r.c.Theme()
	d := r.c.data
	m := r.sizes()
	r.objs = r.objs[:0]
	add := func(o fyne.CanvasObject) { r.objs = append(r.objs, o) }
	col := func(n fyne.ThemeColorName) color.Color {
		if n == "" {
			n = colorAccent
		}
		return themeColor(th, n)
	}
	text := func(s string, c fyne.ThemeColorName) *canvas.Text {
		t := canvas.NewText(s, themeColor(th, c))
		t.TextSize = m.text
		return t
	}

	top := m.pad
	plotH := r.plotHeight()
	legendH := r.legendHeight()
	// The plot's own height grows with the widget when it is given more than it asked for.
	if extra := size.Height - r.MinSize().Height; extra > 0 {
		plotH += extra
	}
	hasData := false
	for _, s := range d.Series {
		hasData = hasData || len(s.Points) > 0
	}
	ymax := d.valueMax()
	step := niceStep(ymax, chartGridLines)
	ticks := int(math.Ceil(ymax / step))
	top2 := step * float64(ticks) // the top of the axis, a whole number of steps
	if top2 <= 0 {
		top2 = 1
	}

	// The value labels' column.
	labelW := float32(0)
	labels := make([]string, 0, ticks+1)
	for i := 0; i <= ticks; i++ {
		l := d.format(step * float64(i))
		labels = append(labels, l)
		labelW = fyne.Max(labelW, fyne.MeasureText(l, m.text, fyne.TextStyle{}).Width)
	}
	left := labelW + m.pad*2
	plotW := fyne.Max(size.Width-left-m.pad, 1)
	y := func(v float64) float32 { return top + plotH - float32(v/top2)*plotH }
	span := d.To.Sub(d.From)
	x := func(t time.Time) float32 {
		if span <= 0 {
			return left
		}
		return left + float32(float64(t.Sub(d.From))/float64(span))*plotW
	}

	// Plot background and grid.
	bg := canvas.NewRectangle(col(theme.ColorNameInputBackground))
	bg.Move(fyne.NewPos(left, top))
	bg.Resize(fyne.NewSize(plotW, plotH))
	add(bg)
	for i, l := range labels {
		gy := y(step * float64(i))
		line := canvas.NewLine(col(theme.ColorNameSeparator))
		line.StrokeWidth = 1
		line.Position1, line.Position2 = fyne.NewPos(left, gy), fyne.NewPos(left+plotW, gy)
		add(line)
		t := text(l, colorMuted)
		ts := t.MinSize()
		t.Move(fyne.NewPos(left-m.pad-ts.Width, gy-ts.Height/2))
		t.Resize(ts)
		add(t)
	}

	// Time labels: start, middle and end of the axis.
	if span > 0 {
		for i, at := range []time.Time{d.From, d.From.Add(span / 2), d.To} {
			t := text(d.timeLabel(at), colorMuted)
			ts := t.MinSize()
			tx := x(at) - ts.Width/2
			switch i {
			case 0:
				tx = left
			case 2:
				tx = left + plotW - ts.Width
			}
			t.Move(fyne.NewPos(tx, top+plotH+m.pad/2))
			t.Resize(ts)
			add(t)
		}
	}

	if !hasData {
		msg := d.Empty
		if msg == "" {
			msg = "No data yet"
		}
		t := text(msg, colorMuted)
		ts := t.MinSize()
		t.Move(fyne.NewPos(left+(plotW-ts.Width)/2, top+(plotH-ts.Height)/2))
		t.Resize(ts)
		add(t)
	}

	// Event markers: a vertical line with a diamond at its top.
	for _, mk := range d.Markers {
		if mk.At.Before(d.From) || mk.At.After(d.To) {
			continue
		}
		c := col(fallback(mk.Color, colorStatusPending))
		mx := x(mk.At)
		line := canvas.NewLine(c)
		line.StrokeWidth = 1
		line.Position1, line.Position2 = fyne.NewPos(mx, top), fyne.NewPos(mx, top+plotH)
		add(line)
		cy := top + chartDiamond
		pts := [4]fyne.Position{
			fyne.NewPos(mx, cy-chartDiamond), fyne.NewPos(mx+chartDiamond, cy),
			fyne.NewPos(mx, cy+chartDiamond), fyne.NewPos(mx-chartDiamond, cy),
		}
		for i := range pts {
			e := canvas.NewLine(c)
			e.StrokeWidth = 1.5
			e.Position1, e.Position2 = pts[i], pts[(i+1)%4]
			add(e)
		}
	}

	// Reference lines, dashed.
	for _, l := range d.Lines {
		if l.Value <= 0 {
			continue
		}
		c := col(l.Color)
		ly := y(l.Value)
		for dx := float32(0); dx < plotW; dx += chartDash + chartGap {
			seg := canvas.NewLine(c)
			seg.StrokeWidth = chartRefWidth
			seg.Position1 = fyne.NewPos(left+dx, ly)
			seg.Position2 = fyne.NewPos(left+fyne.Min(dx+chartDash, plotW), ly)
			add(seg)
		}
	}

	// Series.
	for _, s := range d.Series {
		c := col(s.Color)
		var prev *chartPoint
		segs := 0
		for i := range s.Points {
			p := s.Points[i]
			if p.At.Before(d.From) || p.At.After(d.To) {
				prev = nil
				continue
			}
			if prev != nil && segs < chartMaxSegmentsPerSer {
				seg := canvas.NewLine(c)
				seg.StrokeWidth = chartLineWidth
				seg.Position1, seg.Position2 = fyne.NewPos(x(prev.At), y(prev.Value)), fyne.NewPos(x(p.At), y(p.Value))
				add(seg)
				segs++
			}
			prev = &s.Points[i]
		}
		if len(s.Points) == 1 || (segs == 0 && prev != nil) {
			dot := canvas.NewCircle(c)
			dot.Move(fyne.NewPos(x(prev.At)-chartPointRadius, y(prev.Value)-chartPointRadius))
			dot.Resize(fyne.NewSize(2*chartPointRadius, 2*chartPointRadius))
			add(dot)
		}
	}

	// Legend: a colour square and a name for each series, line and marker kind.
	if legendH > 0 {
		lx := left
		ly := top + plotH + m.pad/2 + m.textH + m.pad/2
		entry := func(name string, c fyne.ThemeColorName) {
			sw := canvas.NewRectangle(col(c))
			sw.Move(fyne.NewPos(lx, ly+(m.textH-chartLegendSwatch)/2))
			sw.Resize(fyne.NewSize(chartLegendSwatch, chartLegendSwatch))
			add(sw)
			t := text(name, colorMuted)
			ts := t.MinSize()
			t.Move(fyne.NewPos(lx+chartLegendSwatch+m.pad, ly))
			t.Resize(ts)
			add(t)
			lx += chartLegendSwatch + m.pad + ts.Width + 2*m.pad
		}
		for _, s := range d.Series {
			entry(s.Name, s.Color)
		}
		for _, l := range d.Lines {
			if l.Value > 0 {
				entry(l.Name, l.Color)
			}
		}
		if len(d.Markers) > 0 {
			entry(d.Markers[0].Name, fallback(d.Markers[0].Color, colorStatusPending))
		}
	}
}

func fallback(n, def fyne.ThemeColorName) fyne.ThemeColorName {
	if n == "" {
		return def
	}
	return n
}
