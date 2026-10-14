package fyneui

import (
	"bytes"
	"image"
	"image/png"
	"runtime"
	"strings"
	"testing"

	"github.com/fyne-io/oksvg"
	"github.com/srwiley/rasterx"
)

// An SVG chart icon becomes a PNG of the raster side, in its aspect, whatever its viewBox says; a
// broken or panicking one fails without taking the window along (H-88).

func decodePNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("not a PNG: %v", err)
	}
	return img
}

func TestRasteriseSVGDrawsTheIcon(t *testing.T) {
	out, err := rasteriseSVG([]byte(fixtureIconSVG))
	if err != nil {
		t.Fatal(err)
	}
	img := decodePNG(t, out)
	if b := img.Bounds(); b.Dx() != 128 || b.Dy() != 128 {
		t.Fatalf("size %v, want 128×128", b)
	}
	if _, _, _, a := img.At(64, 64).RGBA(); a == 0 {
		t.Error("the icon's centre is transparent: nothing was drawn")
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a != 0 {
		t.Error("the corner outside the circle is drawn")
	}

	wide := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 40 20"><rect width="40" height="20" fill="#2a9d8f"/></svg>`
	out, err = rasteriseSVG([]byte(wide))
	if err != nil {
		t.Fatal(err)
	}
	if b := decodePNG(t, out).Bounds(); b.Dx() != 128 || b.Dy() != 64 {
		t.Errorf("a 2:1 icon is %v, want 128×64", b)
	}
}

// A viewBox of a billion units a side costs what a 128-pixel icon costs: Fyne's renderer would
// size its rasteriser by it.
func TestRasteriseSVGIgnoresAHugeViewBox(t *testing.T) {
	huge := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1000000000 1000000000">` +
		`<rect width="1000000000" height="1000000000" fill="#2a9d8f"/></svg>`
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	out, err := rasteriseSVG([]byte(huge))
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodePNG(t, out).Bounds(); b.Dx() != 128 || b.Dy() != 128 {
		t.Errorf("size %v", b)
	}
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 64<<20 { // generous: other tests' goroutines allocate too
		t.Errorf("allocated %d bytes for a 128-pixel icon", alloc)
	} else {
		t.Logf("allocated %d bytes", alloc)
	}
}

func TestRasteriseSVGFailsWithoutTheWindow(t *testing.T) {
	for name, svg := range map[string]string{
		"no size":  `<svg xmlns="http://www.w3.org/2000/svg"><rect width="1" height="1"/></svg>`,
		"not XML":  `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><rect`,
		"negative": `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 -5 10"/>`,
	} {
		if out, err := rasteriseSVG([]byte(svg)); err == nil {
			t.Errorf("%s: drew %d bytes, want an error", name, len(out))
		}
	}

	drawn := drawSVG
	t.Cleanup(func() { drawSVG = drawn })
	drawSVG = func(*oksvg.SvgIcon, *rasterx.Dasher) { panic("renderer bug") }
	if _, err := rasteriseSVG([]byte(fixtureIconSVG)); err == nil || !strings.Contains(err.Error(), "renderer bug") {
		t.Errorf("a panicking renderer: %v, want its error", err)
	}
}
