package yamlview

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The S4 / P8 benchmarks with painting: each operation is followed by a full offscreen paint of a
// 900×700 window (Fyne's software painter via the test driver), the offscreen stand-in for a
// repaint. The software painter rasterises every text object on every capture, while the GL
// painter caches text textures by content, so these numbers overstate the painting part of a GPU
// frame; they leave out the GPU upload and vsync. Tagged like render_test.go.

// BenchmarkKeystroke5MB is keystroke to repaint: a rune typed in the middle of the 5 MB document,
// highlighting on, then the window painted.
func BenchmarkKeystroke5MB(b *testing.B) {
	v, w := newRenderView(b, theme.VariantDark, 900, 700)
	v.SetText(doc5MB())
	v.SetReadOnly(false)
	w.Canvas().Focus(v)
	line := v.LineCount() / 2
	v.SetCursorPosition(line, 6)
	w.Canvas().Capture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.TypedRune('x')
		w.Canvas().Capture()
		if i%64 == 63 { // keep the line short: start on the next one
			line++
			v.SetCursorPosition(line, 6)
		}
	}
}

// BenchmarkSetText5MB loads the 5 MB document and paints the first screen.
func BenchmarkSetText5MB(b *testing.B) {
	v, w := newRenderView(b, theme.VariantDark, 900, 700)
	text := doc5MB()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.SetText(text)
		w.Canvas().Capture()
	}
}

// BenchmarkScrollPage5MB presses Page Down through the 5 MB document and paints each page; at the
// end it starts again from the top (outside the timing).
func BenchmarkScrollPage5MB(b *testing.B) {
	v, w := newRenderView(b, theme.VariantDark, 900, 700)
	v.SetText(doc5MB())
	w.Canvas().Focus(v)
	last := v.LineCount() - 1
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.TypedKey(&fyne.KeyEvent{Name: fyne.KeyPageDown})
		w.Canvas().Capture()
		if l, _ := v.CursorPosition(); l >= last {
			b.StopTimer()
			v.SetCursorPosition(0, 0)
			b.StartTimer()
		}
	}
}

// BenchmarkPaintOnly is the offscreen paint of the same window with nothing changed, to separate the
// painter's share from the widget's.
func BenchmarkPaintOnly(b *testing.B) {
	v, w := newRenderView(b, theme.VariantDark, 900, 700)
	v.SetText(doc5MB())
	v.SetCursorPosition(v.LineCount()/2, 6)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Canvas().Capture()
	}
}

// BenchmarkPaintFloor paints the same window holding one full-size rectangle: the software
// painter's fixed cost for a 900×700 frame, which no widget can avoid.
func BenchmarkPaintFloor(b *testing.B) {
	_, w := newRenderView(b, theme.VariantDark, 900, 700)
	w.SetContent(canvas.NewRectangle(theme.Color(theme.ColorNameInputBackground)))
	w.Resize(fyne.NewSize(900, 700))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Canvas().Capture()
	}
}

// BenchmarkPaintTextGridViewport paints a widget.TextGrid holding only the visible lines with line
// numbers, the design this package did not take: two canvas objects per character cell.
func BenchmarkPaintTextGridViewport(b *testing.B) {
	_, w := newRenderView(b, theme.VariantDark, 900, 700)
	v := New()
	v.SetText(doc5MB())
	rows := int(700/v.metrics().lh) + 1
	grid := widget.NewTextGrid()
	grid.ShowLineNumbers = true
	grid.SetText(strings.Join(v.doc.lines[:rows], "\n"))
	w.SetContent(grid)
	w.Resize(fyne.NewSize(900, 700))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w.Canvas().Capture()
	}
}
