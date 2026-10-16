package fyneui

import (
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

// The area under a series is filled (Lens's charts, captured: "the chart's fill"): one raster per
// run of points, between the run's first and last point and down to zero, in the series' colour at
// chartFillAlpha; a gap splits it as it splits the line, a point alone has none; the fill lies under
// the lines.
func TestChartFillsUnderTheSeries(t *testing.T) {
	a := test.NewTempApp(t)
	a.Settings().SetTheme(newTheme(ThemeDark))
	at := time.Date(2026, 10, 16, 21, 0, 0, 0, time.UTC)
	var pts []chartPoint
	for i := range 20 {
		if i >= 8 && i < 14 {
			continue // a gap: two runs
		}
		pts = append(pts, chartPoint{At: at.Add(time.Duration(i) * 30 * time.Second), Value: float64(1 + i%3)})
	}
	pts = append(pts, chartPoint{At: at.Add(30 * time.Minute), Value: 2}) // alone after a long gap: a dot, no fill
	c := newChart()
	c.Set(chartData{From: at, To: at.Add(31 * time.Minute), Series: []chartSeries{{Name: "CPU usage", Color: colorAccent, Points: pts}}})
	w := test.NewTempWindow(t, c)
	w.Resize(fyne.NewSize(400, 200))
	objs := test.WidgetRenderer(c).Objects()
	var fills []*canvas.Raster
	firstLine := -1
	for i, o := range objs {
		switch x := o.(type) {
		case *canvas.Raster:
			fills = append(fills, x)
		case *canvas.Line:
			if x.StrokeWidth == chartLineWidth && firstLine < 0 {
				firstLine = i
			}
		}
	}
	if len(fills) != 2 {
		t.Fatalf("%d fills, want one per run of points (2)", len(fills))
	}
	g := c.geom
	if f := fills[0]; f.Position().X != g.x(pts[0].At) || f.Position().Y+f.Size().Height != g.y(0) {
		t.Errorf("the first fill at %v size %v, want from x %v down to the axis at %v", f.Position(), f.Size(), g.x(pts[0].At), g.y(0))
	}
	for i, o := range objs {
		if r, ok := o.(*canvas.Raster); ok && r == fills[len(fills)-1] && i > firstLine {
			t.Error("a fill is drawn over a line")
		}
	}
	// The raster: filled under the line, transparent above, the series' colour at the fill's alpha.
	img := fills[0].Generator(100, 100)
	under, above := img.At(50, 99), img.At(50, 0)
	if _, _, _, a := under.RGBA(); a == 0 {
		t.Error("no fill under the line")
	}
	if _, _, _, a := above.RGBA(); a != 0 {
		t.Error("filled above the line")
	}
	if n, ok := under.(color.NRGBA); !ok || n.A != chartFillAlpha {
		t.Errorf("fill colour %#v, want the accent at alpha %#x", under, chartFillAlpha)
	}
}

func TestLineAt(t *testing.T) {
	xs, ys := []float32{0, 10, 20}, []float32{100, 50, 80}
	for _, c := range []struct{ x, want float32 }{{-5, 100}, {0, 100}, {5, 75}, {10, 50}, {15, 65}, {25, 80}} {
		if got := lineAt(xs, ys, c.x); got != c.want {
			t.Errorf("lineAt(%v) = %v, want %v", c.x, got, c.want)
		}
	}
	if chartFill([]float32{1}, []float32{1}, 5, color.White) != nil || chartFill([]float32{1, 1}, []float32{1, 1}, 5, color.White) != nil {
		t.Error("a fill without width")
	}
}
