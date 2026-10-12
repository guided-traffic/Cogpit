// Command icon draws Cogpit's app icon, the "C helm" (D-24), into a 1024×1024 PNG. The drawing is
// internal/ui/logo's, the geometry of docs/developer/assets/cogpit-icon.svg: Solarized cyan and
// base1 on Solarized Dark's base03, on a rounded-square plate. No third-party assets.
//
// The PNG carries its own plate, with transparent corners and the margin macOS puts around an icon
// (100 of 1024 pixels, the plate's radius as `fyne package` would mask it), so `make app` passes
// `--use-raw-icon`: without it fyne package would shrink and mask the icon a second time.
// Regenerate it with:
//
//	go run ./hack/icon -o packaging/icon.png
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"

	"github.com/guided-traffic/cogpit/internal/ui/logo"
)

const (
	size   = 1024
	border = 100.0 // transparent margin around the plate, as macOS icons have
)

// Solarized: base03 (the plate), base1 (the C, the handles, the outer ring) and cyan (the rest).
var (
	plate  = color.NRGBA{R: 0x00, G: 0x2b, B: 0x36, A: 0xff}
	frame  = color.NRGBA{R: 0x93, G: 0xa1, B: 0xa1, A: 0xff}
	accent = color.NRGBA{R: 0x2a, G: 0xa1, B: 0x98, A: 0xff}
)

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

// render draws the plate (the reference's 2…98 of its 100-unit box) into the square inside the
// margin.
func render() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	scale := (size - 2*border) / (logo.Box - 4) // 96 box units are the plate's side
	off := border - 2*scale
	logo.Draw(img, logo.Icon(plate), logo.Colors{Frame: frame, Accent: accent}, scale, off, off)
	return img
}
