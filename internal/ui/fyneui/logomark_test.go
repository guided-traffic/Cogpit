package fyneui

import (
	"image"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

// TestHeaderMark: the header's mark is the C helm of D-24, 20 pixels, drawn in the theme's logo
// inks: base1 and cyan on the dark theme, base01 and cyan on the light one.
func TestHeaderMark(t *testing.T) {
	for mode, frame := range map[ThemeMode]color.NRGBA{
		ThemeDark:  {R: 0x93, G: 0xa1, B: 0xa1, A: 0xff},
		ThemeLight: {R: 0x58, G: 0x6e, B: 0x75, A: 0xff},
	} {
		t.Run(string(mode), func(t *testing.T) {
			a := test.NewTempApp(t)
			a.Settings().SetTheme(newTheme(mode))
			l := newLogo()
			w := test.NewTempWindow(t, l)
			w.Resize(fyne.NewSize(40, 40))

			if got := l.MinSize(); got.Width != 20 || got.Height != 20 {
				t.Fatalf("the mark is %v, want 20×20", got)
			}
			img, ok := test.WidgetRenderer(l.(fyne.Widget)).Objects()[0].(*canvas.Image).Image.(*image.NRGBA)
			if !ok || img.Bounds().Dx() != 20 {
				t.Fatalf("the mark's picture is %T, want a 20-pixel NRGBA", img)
			}
			at := func(x, y float64) color.NRGBA { return img.NRGBAAt(int(math.Floor(x/5)), int(math.Floor(y/5))) }
			accent := color.NRGBA{R: 0x2a, G: 0xa1, B: 0x98, A: 0xff}
			if got := at(50, 50); !closeTo(got, accent) { // the hub
				t.Errorf("hub %v, want the accent %v", got, accent)
			}
			if got := at(50, 22); !closeTo(got, frame) && !closeTo(got, accent) { // the C's top, or the spoke below it
				t.Errorf("top of the C %v, want %v", got, frame)
			}
			if got := at(22, 62); !closeTo(got, frame) { // the C, left and low, off every spoke
				t.Errorf("the C %v, want %v", got, frame)
			}
			if got := at(2, 2); got.A != 0 {
				t.Errorf("the corner is %v, want transparent: the mark has no plate", got)
			}
			if got := at(82, 57); got.A > 0x10 { // the opening below the needle
				t.Errorf("the C's opening is %v, want empty", got)
			}
		})
	}
}

func closeTo(a, b color.NRGBA) bool {
	d := func(x, y uint8) int { return max(int(x)-int(y), int(y)-int(x)) }
	return d(a.R, b.R) < 40 && d(a.G, b.G) < 40 && d(a.B, b.B) < 40 && a.A > 0xc0
}
