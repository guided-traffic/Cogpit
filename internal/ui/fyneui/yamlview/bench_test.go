package yamlview

import (
	"runtime"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// Gate S4 / budget P8 (docs/developer/spike-fyne.md, docs/developer/testing.md): a 5 MB YAML
// document, keystroke to repaint ≤ 16 ms with highlighting, memory proportional to the document.
// The benchmarks here measure the widget's own work for one keystroke (edit, tokenizer, placing
// the canvas objects) without painting; bench_paint_test.go adds the offscreen paint.

const fiveMB = 5 << 20

// doc5MB is the synthetic document, built once per test binary.
var doc5MB = sync.OnceValue(func() string { return syntheticYAML(fiveMB) })

// keystrokeView returns a 900×700 editable view on the 5 MB document with the caret in the middle.
func keystrokeView(tb testing.TB) *View {
	tb.Helper()
	v, _ := newLogicView(900, 700)
	v.SetText(doc5MB())
	v.SetReadOnly(false)
	v.FocusGained()
	v.SetCursorPosition(v.LineCount()/2, 6)
	return v
}

// BenchmarkKeystrokeLayout5MB inserts a rune in the middle of the 5 MB document and rebuilds the
// visible canvas objects: everything a keystroke costs before the painter runs.
func BenchmarkKeystrokeLayout5MB(b *testing.B) {
	v := keystrokeView(b)
	line, _ := v.CursorPosition()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.TypedRune('x')
		if i%64 == 63 { // keep the line short: start on the next one
			line++
			v.SetCursorPosition(line, 6)
		}
	}
}

// BenchmarkNewlineLayout5MB is the most expensive single-key edit: Enter in the middle of the
// document shifts every later line header.
func BenchmarkNewlineLayout5MB(b *testing.B) {
	v := keystrokeView(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		v.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	}
}

// BenchmarkJumpToEnd5MB measures the worst scroll: from the top straight to the last line, which
// scans the tokenizer state of every line once.
func BenchmarkJumpToEnd5MB(b *testing.B) {
	v, _ := newLogicView(900, 700)
	text := doc5MB()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		v.SetText(text)
		b.StartTimer()
		v.ScrollToLine(v.LineCount())
	}
}

// TestMemoryProportionalToDocument measures the heap the view adds when it loads the 5 MB
// document, against the document's size, and the same for a widget.TextGrid holding it (its cell
// model only; the grid is not rendered).
func TestMemoryProportionalToDocument(t *testing.T) {
	if testing.Short() {
		t.Skip("allocates a TextGrid of the 5 MB document")
	}
	text := doc5MB()
	v, _ := newLogicView(900, 700)

	viewBytes := heapGrowth(func() { v.SetText(text) })
	runtime.KeepAlive(v)
	ratio := float64(viewBytes) / float64(len(text))
	t.Logf("view: document %.2f MB in %d lines; heap +%.2f MB = %.2f bytes per document byte (the text itself is shared, not copied)",
		mb(len(text)), v.LineCount(), mb(int(viewBytes)), ratio)
	if ratio > 1 {
		t.Errorf("view adds %.2f bytes per document byte, want ≤ 1", ratio)
	}

	// After edits on every visible line and a scroll through the whole document (all tokenizer
	// states computed) the overhead stays proportional.
	editBytes := heapGrowth(func() {
		for i := range 40 {
			v.SetReadOnly(false)
			v.SetCursorPosition(v.LineCount()/2+i, 4)
			v.TypedRune('x')
		}
		v.ScrollToLine(v.LineCount())
	})
	t.Logf("view after 40 edits and a scroll to the end: heap +%.2f MB more", mb(int(editBytes)))

	grid := &widget.TextGrid{} // not extended: SetText builds the cells without a renderer
	gridBytes := heapGrowth(func() { grid.SetText(text) })
	runtime.KeepAlive(grid)
	t.Logf("TextGrid: heap +%.2f MB = %.1f bytes per document byte (%d rows of cells)",
		mb(int(gridBytes)), float64(gridBytes)/float64(len(text)), len(grid.Rows))
}

// heapGrowth returns the live heap f leaves behind, after a full collection on either side.
func heapGrowth(f func()) uint64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	f()
	runtime.GC()
	runtime.ReadMemStats(&after)
	if after.HeapAlloc < before.HeapAlloc {
		return 0
	}
	return after.HeapAlloc - before.HeapAlloc
}

func mb(n int) float64 { return float64(n) / (1 << 20) }
