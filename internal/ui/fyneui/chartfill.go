package fyneui

import (
	"image/color"
	"sort"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

// The area under a chart's series (G12, ui-parity › Cluster overview and charts, night run
// 2026-10-16/17): Lens's metrics charts fill it (captured with the drawer's charts, "the chart's
// fill"); Cogpit's drew lines only. Each run of a series' points (a line between gaps,
// chartgap.go) is filled down to zero in the series' colour at chartFillAlpha, under the grid's
// reference lines, the event markers and the lines themselves, so those stay readable. The
// alpha and the fill of every series (the network's receive and transmit both) are Cogpit's: the
// lead's decisions for the owner to confirm in a real window (architecture §9).

// chartFillAlpha is the fill's opacity: a fifth of the line's colour.
const chartFillAlpha = 0x33

// chartFill is the area under one run of points, xs and ys in the chart's coordinates (the plot's
// x and y of each point, left to right), down to base (the plot's y of zero); nil when it has no
// area. It is one raster the size of the run: a pixel is filled where it lies under the line
// between the points around it.
func chartFill(xs, ys []float32, base float32, c color.Color) *canvas.Raster {
	if len(xs) < 2 || len(xs) != len(ys) {
		return nil
	}
	x0, x1 := xs[0], xs[len(xs)-1]
	top := base
	for _, y := range ys {
		top = min(top, y)
	}
	w, h := x1-x0, base-top
	if w <= 0 || h <= 0 {
		return nil
	}
	fill := withAlpha(c, chartFillAlpha)
	r := canvas.NewRasterWithPixels(func(px, py, pw, ph int) color.Color {
		if pw <= 0 || ph <= 0 {
			return color.NRGBA{}
		}
		ux := x0 + (float32(px)+0.5)*w/float32(pw)
		uy := top + (float32(py)+0.5)*h/float32(ph)
		if uy >= lineAt(xs, ys, ux) {
			return fill
		}
		return color.NRGBA{} // the same colour type as the fill: the raster's image takes the first pixel's type
	})
	r.Move(fyne.NewPos(x0, top))
	r.Resize(fyne.NewSize(w, h))
	return r
}

// lineAt is the y of the line through the points at x, between the two points around it.
func lineAt(xs, ys []float32, x float32) float32 {
	i := sort.Search(len(xs), func(i int) bool { return xs[i] >= x })
	switch {
	case i <= 0:
		return ys[0]
	case i >= len(xs):
		return ys[len(ys)-1]
	case xs[i] == xs[i-1]:
		return min(ys[i], ys[i-1])
	}
	t := (x - xs[i-1]) / (xs[i] - xs[i-1])
	return ys[i-1] + t*(ys[i]-ys[i-1])
}
