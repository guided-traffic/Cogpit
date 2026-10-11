//go:build fynetest

// Tests that paint need Fyne's software painter, which imports github.com/anthonynsimon/bild, and
// Fyne's test package, which imports github.com/stretchr/testify; go.mod lists neither yet. Run
// them with -tags fynetest once `go mod tidy` has added both (harness_test.go explains).

package yamlview

import (
	"flag"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// updateRenders rewrites the PNGs in testdata: go test ./internal/ui/fyneui/yamlview -run Render -update
var updateRenders = flag.Bool("update", false, "write the offscreen renders into testdata")

// TestRenderSample renders the sample object in the dark and light variants and with a selection,
// checks that highlighting produced distinct colours, and with -update writes the PNGs that the
// pull request shows (docs/developer/conventions.md, UI).
func TestRenderSample(t *testing.T) {
	cases := []struct {
		name    string
		variant fyne.ThemeVariant
		setup   func(v *View, w fyne.Window)
	}{
		{"yaml_dark.png", theme.VariantDark, nil},
		{"yaml_light.png", theme.VariantLight, nil},
		{"yaml_selection.png", theme.VariantDark, func(v *View, w fyne.Window) {
			w.Canvas().Focus(v)
			containers := strings.Index(sampleYAML, "      containers:")
			v.SetCursorPosition(strings.Count(sampleYAML[:containers], "\n"), 6)
			v.shiftDown = true
			for range 4 {
				v.TypedKey(&fyne.KeyEvent{Name: fyne.KeyDown})
			}
			v.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnd})
			v.shiftDown = false
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, w := newRenderView(t, tc.variant, 900, 700)
			v.SetText(sampleYAML)
			if tc.setup != nil {
				tc.setup(v, w)
			}
			img := w.Canvas().Capture()
			if n := countColours(img); n < 6 {
				t.Errorf("render has %d distinct colours, want highlighting (≥ 6)", n)
			}
			if *updateRenders {
				writePNG(t, filepath.Join("testdata", tc.name), img)
			}
		})
	}
}

// TestRenderOnlyVisibleLines checks the core of S4: the canvas objects depend on the viewport, not
// on the document.
func TestRenderOnlyVisibleLines(t *testing.T) {
	v, w := newRenderView(t, theme.VariantDark, 900, 700)
	v.SetText(syntheticYAML(1 << 20))
	w.Canvas().Capture()
	r := newRenderer(v) // builds the same objects as the cached renderer for this state
	texts := 0
	for _, o := range r.Objects() {
		if _, ok := o.(*canvas.Text); ok {
			texts++
		}
	}
	rows := int(700/v.metrics().lh) + 2
	if texts > rows*12 {
		t.Errorf("%d text objects for %d visible rows", texts, rows)
	}
	if len(v.doc.states) > rows+1 {
		t.Errorf("tokenizer states for %d lines, want only the visible prefix (%d)", len(v.doc.states), rows)
	}
}

// newRenderView puts a fresh view into an unpadded offscreen window of the given size, with Fyne's
// default theme in the given variant. The test window never reaches the screen.
func newRenderView(tb testing.TB, variant fyne.ThemeVariant, w, h float32) (*View, fyne.Window) {
	tb.Helper()
	app := test.NewTempApp(tb)
	app.Settings().SetTheme(forcedVariant{Theme: theme.DefaultTheme(), variant: variant})
	v := New()
	win := test.NewWindow(v)
	win.SetPadded(false)
	win.Resize(fyne.NewSize(w, h))
	tb.Cleanup(win.Close)
	return v, win
}

func countColours(img image.Image) int {
	seen := map[color.NRGBA]bool{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y += 2 {
		for x := b.Min.X; x < b.Max.X; x += 2 {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.A == 0xff && (c.R != c.G || c.G != c.B) { // chromatic pixels only
				seen[color.NRGBA{R: c.R &^ 0x1f, G: c.G &^ 0x1f, B: c.B &^ 0x1f, A: 0xff}] = true
			}
		}
	}
	return len(seen)
}

func writePNG(t *testing.T, path string, img image.Image) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	}()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
