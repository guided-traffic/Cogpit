// Command icon draws Cogpit's app icon, a cog around a cockpit gauge, into a 1024×1024 PNG. It
// uses only simple vector shapes and no third-party assets.
//
// The icon is square and full-bleed: `fyne package` adds the macOS rounded-square mask and
// margin itself. Regenerate it with:
//
//	go run ./hack/icon -o packaging/icon.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

const (
	size    = 1024
	samples = 4 // per axis: 16 samples a pixel for anti-aliasing
	centre  = size / 2.0

	teeth     = 10
	rTip      = 430.0 // gear tooth tips
	rRoot     = 365.0 // gear body
	rFace     = 288.0 // gauge face, inside the gear ring
	rimWidth  = 9.0   // dark line between the ring and the face
	toothRoot = 66.0  // half-width of a tooth at its root, in pixels of arc
	toothTip  = 44.0  // half-width at its tip

	sweepStart = 225.0 // gauge sweep, degrees counter-clockwise from +x: lower left …
	sweepEnd   = -45.0 // … over the top to lower right
	needleAt   = 0.62  // needle position along the sweep
	redFrom    = 0.74  // the amber band starts here
)

// Palette.
var (
	bgTop     = rgb(0x0b, 0x1d, 0x33)
	bgBottom  = rgb(0x06, 0x12, 0x22)
	gearLight = rgb(0x45, 0xd9, 0xc8)
	gearDark  = rgb(0x17, 0x93, 0x88)
	rim       = rgb(0x0a, 0x4a, 0x45)
	faceIn    = rgb(0x14, 0x2b, 0x46)
	faceOut   = rgb(0x0a, 0x18, 0x2b)
	tick      = rgb(0xd6, 0xea, 0xff)
	band      = rgb(0xff, 0xb6, 0x27)
	needle    = rgb(0xff, 0x7a, 0x3d)
	hub       = rgb(0xe8, 0xf1, 0xfa)
	hubCore   = rgb(0x0d, 0x1b, 0x2e)
)

type rgba struct{ r, g, b float64 }

func rgb(r, g, b uint8) rgba { return rgba{float64(r), float64(g), float64(b)} }

func mix(a, b rgba, t float64) rgba {
	t = math.Max(0, math.Min(1, t))
	return rgba{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

func main() {
	out := flag.String("o", "packaging/icon.png", "output PNG file")
	flag.Parse()
	if err := write(*out); err != nil {
		fmt.Fprintln(os.Stderr, "icon:", err)
		os.Exit(1)
	}
}

func write(path string) (err error) {
	img := render()
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	return png.Encode(f, img)
}

func render() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	const n = samples * samples
	for py := range size {
		for px := range size {
			var sum rgba
			for sy := range samples {
				for sx := range samples {
					c := shade(float64(px)+(float64(sx)+0.5)/samples, float64(py)+(float64(sy)+0.5)/samples)
					sum.r, sum.g, sum.b = sum.r+c.r, sum.g+c.g, sum.b+c.b
				}
			}
			img.SetNRGBA(px, py, color.NRGBA{R: uint8(sum.r/n + 0.5), G: uint8(sum.g/n + 0.5), B: uint8(sum.b/n + 0.5), A: 255})
		}
	}
	return img
}

// shade returns the colour at a point, painting the layers back to front.
func shade(x, y float64) rgba {
	dx, dy := x-centre, centre-y // y up
	r := math.Hypot(dx, dy)
	theta := math.Atan2(dy, dx)

	c := mix(bgTop, bgBottom, y/size)
	if inGear(r, theta) {
		c = mix(gearLight, gearDark, (y-(centre-rTip))/(2*rTip))
	}
	if r <= rFace+rimWidth {
		c = rim
	}
	if r > rFace {
		return c
	}
	c = mix(faceIn, faceOut, r/rFace)

	// The sweep position of this point: 0 at sweepStart, 1 at sweepEnd, clockwise.
	deg := theta * 180 / math.Pi
	pos := math.Mod(sweepStart-deg+720, 360) / (sweepStart - sweepEnd)
	if pos <= 1 {
		if pos >= redFrom && r >= 262 && r <= 276 {
			c = band
		}
		if onTick(r, pos) {
			c = tick
		}
	}
	if onNeedle(dx, dy) {
		c = needle
	}
	switch {
	case r <= 18:
		c = hubCore
	case r <= 40:
		c = hub
	}
	return c
}

// inGear reports whether a point lies on the cog: its body or one of its trapezoid teeth.
func inGear(r, theta float64) bool {
	if r > rTip {
		return false
	}
	if r <= rRoot {
		return true
	}
	pitch := 2 * math.Pi / teeth
	a := math.Mod(theta-math.Pi/2+pitch/2+4*math.Pi, pitch) - pitch/2 // angle from the nearest tooth's centre; a tooth points up
	half := toothRoot + (toothTip-toothRoot)*(r-rRoot)/(rTip-rRoot)
	return math.Abs(a)*r <= half
}

// onTick reports whether a point lies on a gauge tick: eleven major ticks and minor ones between.
func onTick(r, pos float64) bool {
	const steps = 20
	k := math.Round(pos * steps)
	arc := math.Abs(pos-k/steps) * (sweepStart - sweepEnd) * math.Pi / 180 * r // distance along the arc
	if int(k)%2 == 0 {
		return r >= 210 && r <= 256 && arc <= 7
	}
	return r >= 232 && r <= 256 && arc <= 3.5
}

// onNeedle reports whether a point lies on the needle: a tapered bar from the hub to the dial.
func onNeedle(dx, dy float64) bool {
	angle := (sweepStart - needleAt*(sweepStart-sweepEnd)) * math.Pi / 180
	ux, uy := math.Cos(angle), math.Sin(angle)
	along := dx*ux + dy*uy
	across := math.Abs(-dx*uy + dy*ux)
	const length, tail = 236.0, 52.0
	if along < -tail || along > length {
		return false
	}
	half := 15.0
	if along > 0 {
		half = 15 - 12*along/length
	}
	return across <= half
}
