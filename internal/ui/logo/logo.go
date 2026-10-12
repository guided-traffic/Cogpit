// Package logo draws Cogpit's logo, the "C helm" (D-24): an open C around a ship's wheel with
// seven spokes and a needle in the C's opening, a hexagon hub, an inner ring, bar handles outside
// the C and a thin outer ring. The geometry is the one of the reference SVGs
// (docs/developer/assets/cogpit-*.svg), in a 100×100 box; the package rasterises it with
// supersampling and imports neither Fyne nor any other Cogpit package, so that the app icon
// generator (hack/icon) and the header's mark (internal/ui/fyneui) draw the same logo.
//
// Layer L4 (internal/ui).
package logo

import (
	"image"
	"image/color"
	"math"
)

// The geometry, in the reference SVGs' units: a 100×100 box, the wheel's centre at (50, 50).
const (
	Box = 100.0 // the box's side

	centre    = 50.0
	outerR    = 41.0 // the thin outer ring
	cR        = 28.0 // the C's radius
	cOpening  = 75.0 // the C's opening, in degrees, to the right
	innerR    = 12.0 // the inner ring
	spokeBeg  = 12.0 // a spoke runs from the inner ring
	spokeEnd  = 27.0 // to just inside the C
	handleBeg = 34.0 // a bar handle stands outside the C, from here
	handleEnd = 40.0 // to here
	needleEnd = 89.0 // the needle runs from the hub, through the C's opening, to here
	spokes    = 8    // spokes in all; the needle takes the place of the one pointing right

	// The plate of the app icon: a rounded square, as in the reference.
	plateInset  = 2.0
	plateRadius = 22.0
	plateEdge   = 1.0 // width of its faint edge line
)

// Style is the line weights of one drawing of the logo, in box units.
type Style struct {
	OuterRing        float64 // stroke of the thin outer ring
	OuterRingOpacity float64
	C                float64 // stroke of the C
	InnerRing        float64
	Spoke            float64
	Handle           float64
	Needle           float64
	HubRadius        float64 // circumradius of the hexagon hub
	Plate            color.Color
}

// Icon is the app icon's drawing: thin lines on a plate of the given colour.
func Icon(plate color.Color) Style {
	return Style{OuterRing: 0.9, OuterRingOpacity: .6, C: 5.6, InnerRing: 2, Spoke: 1.6, Handle: 2.6,
		Needle: 2.2, HubRadius: 5.13, Plate: plate}
}

// Mark is the drawing for a bar, without a plate: the same logo with heavier lines, so that it
// reads at 20 pixels.
func Mark() Style {
	return Style{OuterRing: 2.25, OuterRingOpacity: .6, C: 14, InnerRing: 5, Spoke: 4, Handle: 6.5,
		Needle: 5.5, HubRadius: 6.75}
}

// Colors are the logo's two inks: Frame draws the C, the handles and the outer ring, Accent the
// inner ring, the spokes, the needle and the hub.
type Colors struct{ Frame, Accent color.Color }

const samples = 4 // per axis: 16 samples a pixel

// premultiplied colour accumulators, in 0..1
type pm struct{ r, g, b, a float64 }

func (d *pm) over(c pm) {
	k := 1 - c.a
	d.r, d.g, d.b, d.a = c.r+d.r*k, c.g+d.g*k, c.b+d.b*k, c.a+d.a*k
}

func ink(c color.Color, opacity float64) pm {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	a := float64(n.A) / 255 * opacity
	return pm{float64(n.R) / 255 * a, float64(n.G) / 255 * a, float64(n.B) / 255 * a, a}
}

// layer is one painted shape: where it covers, in box units, and what it paints.
type layer struct {
	covers func(x, y float64) bool
	paint  pm
}

// layers returns the logo's shapes back to front, as the reference SVGs paint them.
func layers(st Style, c Colors) []layer {
	var ls []layer
	add := func(covers func(x, y float64) bool, paint pm) { ls = append(ls, layer{covers, paint}) }

	if st.Plate != nil {
		size := Box - 2*plateInset
		add(func(x, y float64) bool { return roundRect(x, y, plateInset, size, plateRadius) <= 0 }, ink(st.Plate, 1))
		add(func(x, y float64) bool {
			return math.Abs(roundRect(x, y, plateInset, size, plateRadius)) <= plateEdge/2
		},
			ink(color.White, .06))
	}
	add(func(x, y float64) bool { return math.Abs(math.Hypot(x-centre, y-centre)-outerR) <= st.OuterRing/2 },
		ink(c.Frame, st.OuterRingOpacity))

	// The C: an arc from -(opening/2) around the left to +(opening/2), the opening to the right.
	half := cOpening / 2 * math.Pi / 180
	add(func(x, y float64) bool { return nearArc(x, y, half, st.C/2) }, ink(c.Frame, 1))

	add(func(x, y float64) bool { return math.Abs(math.Hypot(x-centre, y-centre)-innerR) <= st.InnerRing/2 },
		ink(c.Accent, 1))

	// The seven spokes, each an inner bar in the accent ink and a handle outside the C in the frame
	// ink, every 45° but to the right, where the needle is.
	for k := 1; k < spokes; k++ {
		a := float64(k) * 2 * math.Pi / spokes
		dx, dy := math.Cos(a), math.Sin(a)
		add(segment(centre+spokeBeg*dx, centre+spokeBeg*dy, centre+spokeEnd*dx, centre+spokeEnd*dy, st.Spoke/2), ink(c.Accent, 1))
		add(segment(centre+handleBeg*dx, centre+handleBeg*dy, centre+handleEnd*dx, centre+handleEnd*dy, st.Handle/2),
			ink(c.Frame, 1))
	}
	add(segment(centre, centre, needleEnd, centre, st.Needle/2), ink(c.Accent, 1))

	// The hub: a hexagon with a vertex up and one down, i.e. vertices at 30° + k·60°.
	var hex [6][2]float64
	for k := range hex {
		a := (30 + float64(k)*60) * math.Pi / 180
		hex[k] = [2]float64{centre + st.HubRadius*math.Cos(a), centre + st.HubRadius*math.Sin(a)}
	}
	add(func(x, y float64) bool { return inConvex(hex[:], x, y) }, ink(c.Accent, 1))
	return ls
}

// Draw paints the logo onto img, which it does not clear: the box's point (x, y) lands on the
// pixel (ox + x·scale, oy + y·scale).
func Draw(img *image.NRGBA, st Style, c Colors, scale, ox, oy float64) {
	ls := layers(st, c)
	b := img.Bounds()
	// Only the box's pixels (and a margin for the plate's edge line) can be touched.
	x0, x1 := max(b.Min.X, int(math.Floor(ox))-1), min(b.Max.X, int(math.Ceil(ox+Box*scale))+1)
	y0, y1 := max(b.Min.Y, int(math.Floor(oy))-1), min(b.Max.Y, int(math.Ceil(oy+Box*scale))+1)
	const n = samples * samples
	for py := y0; py < y1; py++ {
		for px := x0; px < x1; px++ {
			var sum pm
			for sy := range samples {
				for sx := range samples {
					x := (float64(px) + (float64(sx)+.5)/samples - ox) / scale
					y := (float64(py) + (float64(sy)+.5)/samples - oy) / scale
					var s pm
					for _, l := range ls {
						if l.covers(x, y) {
							s.over(l.paint)
						}
					}
					sum.r, sum.g, sum.b, sum.a = sum.r+s.r, sum.g+s.g, sum.b+s.b, sum.a+s.a
				}
			}
			if sum.a == 0 {
				continue
			}
			// The pixel's coverage composited over what the image holds.
			src := pm{sum.r / n, sum.g / n, sum.b / n, sum.a / n}
			old := img.NRGBAAt(px, py)
			dst := pm{float64(old.R) / 255 * float64(old.A) / 255, float64(old.G) / 255 * float64(old.A) / 255,
				float64(old.B) / 255 * float64(old.A) / 255, float64(old.A) / 255}
			dst.over(src)
			img.SetNRGBA(px, py, color.NRGBA{R: unit(dst.r / dst.a), G: unit(dst.g / dst.a), B: unit(dst.b / dst.a), A: unit(dst.a)})
		}
	}
}

func unit(v float64) uint8 { return uint8(math.Max(0, math.Min(1, v))*255 + .5) }

// roundRect is the signed distance of a point to a rounded square at (inset, inset) with the
// given side and corner radius: negative inside.
func roundRect(x, y, inset, side, r float64) float64 {
	cx, cy := inset+side/2, inset+side/2
	qx, qy := math.Abs(x-cx)-(side/2-r), math.Abs(y-cy)-(side/2-r)
	return math.Hypot(math.Max(qx, 0), math.Max(qy, 0)) + math.Min(math.Max(qx, qy), 0) - r
}

// nearArc reports whether a point lies within w of the C's arc, round caps included. The arc is
// centred on the wheel with radius cR and runs the long way from +half to -half (angles in screen
// direction, 0 pointing right).
func nearArc(x, y, half, w float64) bool {
	dx, dy := x-centre, y-centre
	a := math.Atan2(dy, dx)
	if math.Abs(a) >= half { // on the arc's side of the opening
		return math.Abs(math.Hypot(dx, dy)-cR) <= w
	}
	// In the opening: the nearest part of the arc is one of its two ends.
	for _, s := range [2]float64{-1, 1} {
		ex, ey := centre+cR*math.Cos(half), centre+s*cR*math.Sin(half)
		if math.Hypot(x-ex, y-ey) <= w {
			return true
		}
	}
	return false
}

// segment returns the cover of a line with round caps and the given half width.
func segment(x1, y1, x2, y2, w float64) func(x, y float64) bool {
	return func(x, y float64) bool {
		vx, vy := x2-x1, y2-y1
		t := ((x-x1)*vx + (y-y1)*vy) / (vx*vx + vy*vy)
		t = math.Max(0, math.Min(1, t))
		return math.Hypot(x-(x1+t*vx), y-(y1+t*vy)) <= w
	}
}

// inConvex reports whether a point lies inside a convex polygon.
func inConvex(poly [][2]float64, x, y float64) bool {
	sign := 0.0
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		cross := (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
		if cross == 0 {
			continue
		}
		if sign == 0 {
			sign = cross
		} else if sign*cross < 0 {
			return false
		}
	}
	return true
}
