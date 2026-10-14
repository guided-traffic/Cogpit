package fyneui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/png"
	"math"

	"github.com/fyne-io/oksvg"
	"github.com/srwiley/rasterx"
)

// An SVG chart icon is rasterised here, in the goroutine that fetched it, and the window draws the
// PNG (H-88). Fyne's renderer would parse it again on the main goroutine, size its rasteriser by
// the icon's viewBox (a number the icon's host chooses: a viewBox of a million units a side asks
// for a buffer of a trillion pixels, which no recover catches), and its recover covers the drawing
// only, not the parsing. Here the rasteriser has the icon's own size, chartIconRasterSide, whatever
// the viewBox says, and a panic anywhere fails this icon, which then shows the glyph.

// chartIconRasterSide is the side, in pixels, an SVG icon is drawn at: the drawer's 28 points at
// up to four times the screen's scale.
const chartIconRasterSide = 128

// drawSVG draws a parsed icon; a variable so that a test can make it panic.
var drawSVG = func(icon *oksvg.SvgIcon, d *rasterx.Dasher) { icon.Draw(d, 1) }

// rasteriseSVG returns an SVG icon as a PNG of at most chartIconRasterSide pixels a side, keeping
// its aspect.
func rasteriseSVG(data []byte) (out []byte, err error) {
	const side = chartIconRasterSide
	defer func() {
		if r := recover(); r != nil {
			out, err = nil, fmt.Errorf("the SVG renderer failed: %v", r)
		}
	}()
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data), oksvg.IgnoreErrorMode)
	if err != nil {
		return nil, err
	}
	vw, vh := icon.ViewBox.W, icon.ViewBox.H
	if !(vw > 0 && vh > 0) || math.IsInf(vw, 0) || math.IsInf(vh, 0) {
		return nil, errors.New("the SVG has no usable size")
	}
	w, h := side, side
	switch {
	case vw > vh:
		h = max(1, int(math.Round(float64(side)*vh/vw)))
	case vh > vw:
		w = max(1, int(math.Round(float64(side)*vw/vh)))
	}
	y := 0.0
	if icon.ViewBox.Y < 0 { // as Fyne's own renderer places a viewBox that starts above 0
		y = icon.ViewBox.Y + (-icon.ViewBox.Y/vh)*float64(h)
	}
	icon.SetTarget(0, y, float64(w), float64(h))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	drawSVG(icon, rasterx.NewDasher(w, h, rasterx.NewScannerGV(w, h, img, img.Bounds())))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
