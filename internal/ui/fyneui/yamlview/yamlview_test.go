package yamlview

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// editable returns an editable, focused view holding text, with the caret at line, col.
func editable(t *testing.T, text string, line, col int) (*View, *fakeApp) {
	t.Helper()
	v, app := newLogicView(600, 400)
	v.SetText(text)
	v.SetReadOnly(false)
	v.FocusGained()
	v.SetCursorPosition(line, col)
	return v, app
}

func key(v *View, names ...fyne.KeyName) {
	for _, n := range names {
		v.TypedKey(&fyne.KeyEvent{Name: n})
	}
}

func typeText(v *View, s string) {
	for _, r := range s {
		v.TypedRune(r)
	}
}

func shift(v *View, names ...fyne.KeyName) {
	v.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
	key(v, names...)
	v.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
}

func wantText(t *testing.T, v *View, want string) {
	t.Helper()
	if got := v.Text(); got != want {
		t.Fatalf("text\n got %q\nwant %q", got, want)
	}
}

func wantCursor(t *testing.T, v *View, line, col int) {
	t.Helper()
	if l, c := v.CursorPosition(); l != line || c != col {
		t.Fatalf("cursor at %d:%d, want %d:%d", l, c, line, col)
	}
}

func TestNewIsEmptyReadOnly(t *testing.T) {
	v, _ := newLogicView(100, 100)
	if !v.ReadOnly() || v.Text() != "" || v.LineCount() != 1 || !v.Highlighting() {
		t.Fatalf("new view: readOnly %v, text %q, lines %d", v.ReadOnly(), v.Text(), v.LineCount())
	}
}

func TestTypingAndOnChanged(t *testing.T) {
	v, _ := editable(t, "a: 1", 0, 1)
	changes := 0
	v.OnChanged = func() { changes++ }
	typeText(v, "bc")
	wantText(t, v, "abc: 1")
	wantCursor(t, v, 0, 3)
	if changes != 2 {
		t.Fatalf("OnChanged called %d times, want 2", changes)
	}
	v.SetText("x")
	if changes != 2 {
		t.Fatal("SetText must not call OnChanged")
	}
	v.TypedRune('\n') // control runes are not text; Enter arrives as a key
	v.TypedRune(0x7f)
	wantText(t, v, "x")
}

func TestEnterKeepsIndentation(t *testing.T) {
	v, _ := editable(t, "spec:\n  replicas: 3", 1, 13)
	key(v, fyne.KeyReturn)
	wantText(t, v, "spec:\n  replicas: 3\n  ")
	wantCursor(t, v, 2, 2)

	v, _ = editable(t, "spec:", 0, 5)
	key(v, fyne.KeyEnter)
	wantText(t, v, "spec:\n  ")

	v, _ = editable(t, "  script: |", 0, 11)
	key(v, fyne.KeyReturn)
	wantText(t, v, "  script: |\n    ")

	// In the middle of the indentation only the part before the caret is repeated.
	v, _ = editable(t, "    x", 0, 2)
	key(v, fyne.KeyReturn)
	wantText(t, v, "  \n    x")
}

func TestLineBoundaryEdits(t *testing.T) {
	v, _ := editable(t, "ab\ncd", 1, 0)
	key(v, fyne.KeyBackspace) // joins with the previous line
	wantText(t, v, "abcd")
	wantCursor(t, v, 0, 2)

	v, _ = editable(t, "ab\ncd", 0, 2)
	key(v, fyne.KeyDelete) // at end of line joins the next one
	wantText(t, v, "abcd")
	wantCursor(t, v, 0, 2)

	v, _ = editable(t, "ab\ncd", 0, 0)
	key(v, fyne.KeyBackspace) // at document start: nothing
	wantText(t, v, "ab\ncd")

	v, _ = editable(t, "ab\ncd", 1, 2)
	key(v, fyne.KeyDelete) // at document end: nothing
	wantText(t, v, "ab\ncd")

	v, _ = editable(t, "ab", 0, 2)
	key(v, fyne.KeyReturn)
	typeText(v, "c")
	wantText(t, v, "ab\nc")
	key(v, fyne.KeyLeft, fyne.KeyLeft) // left at column 0 crosses to the previous line's end
	wantCursor(t, v, 0, 2)
	key(v, fyne.KeyRight)
	wantCursor(t, v, 1, 0)
}

func TestBackspaceInIndentation(t *testing.T) {
	v, _ := editable(t, "      x", 0, 6)
	key(v, fyne.KeyBackspace)
	wantText(t, v, "    x")
	v, _ = editable(t, "     x", 0, 5)
	key(v, fyne.KeyBackspace)
	wantText(t, v, "    x")
}

func TestMultiByteEditing(t *testing.T) {
	v, _ := editable(t, "a世é", 0, 3)
	key(v, fyne.KeyBackspace)
	wantText(t, v, "a世")
	key(v, fyne.KeyLeft)
	wantCursor(t, v, 0, 1)
	key(v, fyne.KeyDelete)
	wantText(t, v, "a")
}

func TestVerticalMovesKeepColumn(t *testing.T) {
	v, _ := editable(t, "abcdef\nab\nabcdef", 0, 5)
	key(v, fyne.KeyDown)
	wantCursor(t, v, 1, 2)
	key(v, fyne.KeyDown)
	wantCursor(t, v, 2, 5)
	key(v, fyne.KeyDown) // past the last line: end of document
	wantCursor(t, v, 2, 6)
	v.SetCursorPosition(0, 3)
	key(v, fyne.KeyUp) // past the first line: start of document
	wantCursor(t, v, 0, 0)
}

func TestHomeEnd(t *testing.T) {
	v, _ := editable(t, "    key: v", 0, 7)
	key(v, fyne.KeyHome)
	wantCursor(t, v, 0, 4)
	key(v, fyne.KeyHome)
	wantCursor(t, v, 0, 0)
	key(v, fyne.KeyEnd)
	wantCursor(t, v, 0, 10)
}

func TestShiftSelectionAndClipboard(t *testing.T) {
	v, app := editable(t, "one\ntwo\nthree", 0, 1)
	shift(v, fyne.KeyRight, fyne.KeyDown)
	if got := v.SelectedText(); got != "ne\ntw" {
		t.Fatalf("selected %q", got)
	}
	v.TypedShortcut(&fyne.ShortcutCopy{Clipboard: app.clip})
	if app.clip.content != "ne\ntw" {
		t.Fatalf("copied %q", app.clip.content)
	}
	v.TypedShortcut(&fyne.ShortcutCut{}) // no clipboard in the shortcut: the app's
	wantText(t, v, "oo\nthree")
	wantCursor(t, v, 0, 1)
	v.SetCursorPosition(1, 5)
	v.TypedShortcut(&fyne.ShortcutPaste{Clipboard: app.clip})
	wantText(t, v, "oo\nthreene\ntw")
	wantCursor(t, v, 2, 2)

	// Typing replaces the selection; an arrow without Shift collapses it to its edge.
	shift(v, fyne.KeyHome)
	typeText(v, "X")
	wantText(t, v, "oo\nthreene\nX")
	shift(v, fyne.KeyUp)
	key(v, fyne.KeyLeft)
	wantCursor(t, v, 1, 1)
	if v.SelectedText() != "" {
		t.Fatal("arrow did not collapse the selection")
	}
}

func TestSelectAllAndEscape(t *testing.T) {
	v, app := editable(t, "a\nb", 0, 0)
	v.TypedShortcut(&fyne.ShortcutSelectAll{})
	v.TypedShortcut(&fyne.ShortcutCopy{})
	if app.clip.content != "a\nb" {
		t.Fatalf("copied %q", app.clip.content)
	}
	key(v, fyne.KeyEscape)
	if v.SelectedText() != "" {
		t.Fatal("Escape kept the selection")
	}
}

func TestReadOnlyRefusesEditsAllowsCopy(t *testing.T) {
	v, app := newLogicView(600, 400)
	v.SetText("a: 1\nb: 2")
	v.FocusGained()
	changes := 0
	v.OnChanged = func() { changes++ }
	typeText(v, "x")
	key(v, fyne.KeyReturn, fyne.KeyBackspace, fyne.KeyDelete, fyne.KeyTab)
	app.clip.content = "pasted"
	v.TypedShortcut(&fyne.ShortcutPaste{Clipboard: app.clip})
	v.TypedShortcut(&fyne.ShortcutSelectAll{})
	v.TypedShortcut(&fyne.ShortcutCut{Clipboard: app.clip})
	v.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyBackspace, Modifier: fyne.KeyModifierAlt})
	wantText(t, v, "a: 1\nb: 2")
	if changes != 0 || app.clip.content != "pasted" {
		t.Fatalf("read-only view changed: %d changes, clipboard %q", changes, app.clip.content)
	}
	v.TypedShortcut(&fyne.ShortcutCopy{Clipboard: app.clip})
	if app.clip.content != "a: 1\nb: 2" {
		t.Fatalf("read-only copy gave %q", app.clip.content)
	}
	if v.AcceptsTab() {
		t.Fatal("a read-only view must let Tab move the focus")
	}
	// Keyboard selection still works.
	v.SetCursorPosition(1, 0)
	shift(v, fyne.KeyEnd)
	if v.SelectedText() != "b: 2" {
		t.Fatalf("selected %q", v.SelectedText())
	}
}

func TestTabAndOutdent(t *testing.T) {
	v, _ := editable(t, "a:\nb: 1", 1, 0)
	key(v, fyne.KeyTab)
	wantText(t, v, "a:\n  b: 1")
	wantCursor(t, v, 1, 2)
	v.TypedRune('x')
	key(v, fyne.KeyTab) // odd column: to the next two-space stop
	wantText(t, v, "a:\n  x b: 1")

	v, _ = editable(t, "a: 1\nb: 2\nc: 3", 0, 1)
	shift(v, fyne.KeyDown, fyne.KeyDown)
	key(v, fyne.KeyTab) // a selection across lines indents them
	wantText(t, v, "  a: 1\n  b: 2\n  c: 3")
	if got := v.SelectedText(); got != ": 1\n  b: 2\n  c" {
		t.Fatalf("selection after indent %q", got)
	}
	shift(v, fyne.KeyTab) // Shift+Tab outdents
	wantText(t, v, "a: 1\nb: 2\nc: 3")
}

func TestWordShortcuts(t *testing.T) {
	v, _ := editable(t, "name: kube-system  # ns", 0, 0)
	word := func(k fyne.KeyName, mod fyne.KeyModifier) {
		v.TypedShortcut(&desktop.CustomShortcut{KeyName: k, Modifier: mod})
	}
	word(fyne.KeyRight, fyne.KeyModifierAlt)
	wantCursor(t, v, 0, 4)
	word(fyne.KeyRight, fyne.KeyModifierAlt)
	wantCursor(t, v, 0, 5)
	word(fyne.KeyRight, fyne.KeyModifierControl)
	wantCursor(t, v, 0, 17)
	word(fyne.KeyLeft, fyne.KeyModifierAlt|fyne.KeyModifierShift)
	if v.SelectedText() != "kube-system" {
		t.Fatalf("word selection %q", v.SelectedText())
	}
	word(fyne.KeyEnd, fyne.KeyModifierControl)
	wantCursor(t, v, 0, 23)
	word(fyne.KeyUp, fyne.KeyModifierSuper)
	wantCursor(t, v, 0, 0)
	word(fyne.KeyRight, fyne.KeyModifierSuper)
	wantCursor(t, v, 0, 23)
	word(fyne.KeyBackspace, fyne.KeyModifierAlt)
	wantText(t, v, "name: kube-system  # ")
}

func TestMouseSelection(t *testing.T) {
	v, _ := newLogicView(600, 400)
	v.SetText("alpha: 1\nbeta: 2\ngamma: 3")
	m := v.metrics()
	at := func(line, col int) fyne.Position {
		return fyne.NewPos(float32(m.textX()+float64(col)*m.cw)+1, m.pad+float32(float64(line)*m.lh+m.lh/2))
	}
	v.MouseDown(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at(0, 2)}, Button: desktop.MouseButtonPrimary})
	v.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: at(1, 3)}})
	v.DragEnd()
	v.MouseUp(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at(1, 3)}})
	if got := v.SelectedText(); got != "pha: 1\nbet" {
		t.Fatalf("drag selected %q", got)
	}
	// Shift+click extends.
	v.MouseDown(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: at(2, 5)}, Button: desktop.MouseButtonPrimary, Modifier: fyne.KeyModifierShift})
	if got := v.SelectedText(); got != "pha: 1\nbeta: 2\ngamma" {
		t.Fatalf("shift-click selected %q", got)
	}
	v.DoubleTapped(&fyne.PointEvent{Position: at(1, 1)})
	if got := v.SelectedText(); got != "beta" {
		t.Fatalf("double click selected %q", got)
	}
	// A click in the gutter goes to column 0.
	v.MouseDown(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(2, at(2, 0).Y)}, Button: desktop.MouseButtonPrimary})
	wantCursor(t, v, 2, 0)
}

func TestScrollingAndCaretVisibility(t *testing.T) {
	v, _ := newLogicView(600, 400)
	v.SetText(strings.Repeat("key: value\n", 1000))
	m := v.metrics()
	rows := int(400/m.lh) - 1

	key(v, fyne.KeyPageDown)
	if l, _ := v.CursorPosition(); l != rows {
		t.Fatalf("PageDown moved to line %d, want %d", l, rows)
	}
	if v.scrollY != float64(rows)*m.lh {
		t.Fatalf("PageDown scrolled to %v, want %v", v.scrollY, float64(rows)*m.lh)
	}

	v.SetCursorPosition(900, 0) // far below: scrolls just enough to show the caret
	top := float64(900) * m.lh
	if v.scrollY > top || v.scrollY+400 < top+m.lh {
		t.Fatalf("caret line %v not visible at scroll %v", top, v.scrollY)
	}

	v.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, 1e9)})
	if v.scrollY != 0 {
		t.Fatalf("scrolling up past the top left offset %v", v.scrollY)
	}
	v.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -1e9)})
	if want := v.maxScrollY(v.Size(), m); v.scrollY != want {
		t.Fatalf("scrolling down past the end: %v, want %v", v.scrollY, want)
	}

	v.ScrollToLine(10)
	if v.scrollY != 10*m.lh {
		t.Fatalf("ScrollToLine(10): %v", v.scrollY)
	}
	if l, _ := v.CursorPosition(); l != 900 {
		t.Fatal("ScrollToLine moved the caret")
	}
}

func TestUpdateTextKeepsPosition(t *testing.T) {
	v, _ := newLogicView(600, 400)
	v.SetText(strings.Repeat("status: Running\n", 500))
	v.SetCursorPosition(300, 3)
	scroll := v.scrollY
	changes := 0
	v.OnChanged = func() { changes++ }

	v.UpdateText(strings.Repeat("status: Pending\n", 500))
	if l, c := v.CursorPosition(); l != 300 || c != 3 || v.scrollY != scroll {
		t.Fatalf("after update: cursor %d:%d scroll %v, want 300:3 scroll %v", l, c, v.scrollY, scroll)
	}
	v.UpdateText("short")
	if l, c := v.CursorPosition(); l != 0 || c != 5 || v.scrollY != 0 {
		t.Fatalf("after shrinking: cursor %d:%d scroll %v", l, c, v.scrollY)
	}
	if changes != 0 {
		t.Fatal("UpdateText called OnChanged")
	}
}

func TestHorizontalReveal(t *testing.T) {
	v, _ := newLogicView(300, 200)
	v.SetText("k: " + strings.Repeat("x", 500))
	v.SetCursorPosition(0, 400)
	if v.scrollX <= 0 {
		t.Fatal("caret far right did not scroll horizontally")
	}
	m := v.metrics()
	x := m.textX() + 400*m.cw - v.scrollX
	if x < m.textX() || x > 300 {
		t.Fatalf("caret at x %v outside the view", x)
	}
}

func TestScrollBarDrag(t *testing.T) {
	v, _ := newLogicView(600, 400)
	v.SetText(strings.Repeat("line\n", 2000))
	m := v.metrics()
	size := v.Size()
	off, length, track := v.vThumb(size, m)
	// Grab the thumb and drag it to the bottom of the track.
	v.MouseDown(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(size.Width-2, off+length/2)}, Button: desktop.MouseButtonPrimary})
	v.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, track)})
	v.DragEnd()
	if want := v.maxScrollY(size, m); v.scrollY != want {
		t.Fatalf("thumb drag scrolled to %v, want %v", v.scrollY, want)
	}
	if l, _ := v.CursorPosition(); l != 0 {
		t.Fatal("scroll bar drag moved the caret")
	}
}

func TestHighlightingSwitch(t *testing.T) {
	v, _ := newLogicView(600, 400)
	v.SetText("a: 1")
	if len(v.lineTokens(0)) == 0 {
		t.Fatal("no tokens with highlighting on")
	}
	v.SetHighlighting(false)
	if v.lineTokens(0) != nil {
		t.Fatal("tokens with highlighting off")
	}
}

func TestMinSizeIndependentOfDocument(t *testing.T) {
	v, _ := newLogicView(600, 400)
	small := newRenderer(v).MinSize()
	v.SetText(strings.Repeat(strings.Repeat("x", 300)+"\n", 5000))
	big := newRenderer(v).MinSize()
	if big.Height != small.Height || big.Width > small.Width*2 {
		t.Fatalf("MinSize grew with the document: %v -> %v", small, big)
	}
}

// FuzzViewKeys drives an editable view with random keys, runes, shortcuts and clicks. It must never
// panic, and the caret, the selection and the lines must stay consistent.
func FuzzViewKeys(f *testing.F) {
	f.Add("a: 1\n  b: |\n    x\n", []byte{0, 1, 2, 3, 10, 11, 12, 13, 20, 'q', 21})
	f.Add("", []byte{9, 9, 9, 4, 5, 6, 7, 8})
	f.Add("é世\t\x80\r\nz", []byte{3, 3, 3, 14, 15, 16, 17, 18, 19})
	keys := []fyne.KeyName{
		fyne.KeyLeft, fyne.KeyRight, fyne.KeyUp, fyne.KeyDown, fyne.KeyHome, fyne.KeyEnd,
		fyne.KeyPageUp, fyne.KeyPageDown, fyne.KeyReturn, fyne.KeyBackspace, fyne.KeyDelete,
		fyne.KeyTab, fyne.KeyEscape,
	}
	f.Fuzz(func(t *testing.T, text string, ops []byte) {
		v, app := newLogicView(300, 200)
		v.SetText(text)
		v.SetReadOnly(false)
		v.FocusGained()
		for i := 0; i < len(ops) && i < 400; i++ {
			op := ops[i]
			switch {
			case op < 26:
				v.TypedKey(&fyne.KeyEvent{Name: keys[int(op)%len(keys)]})
			case op < 30:
				v.KeyDown(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
			case op < 34:
				v.KeyUp(&fyne.KeyEvent{Name: desktop.KeyShiftLeft})
			case op < 38:
				v.TypedShortcut(&fyne.ShortcutCopy{Clipboard: app.clip})
			case op < 40:
				v.TypedShortcut(&fyne.ShortcutCut{Clipboard: app.clip})
			case op < 44:
				v.TypedShortcut(&fyne.ShortcutPaste{Clipboard: app.clip})
			case op < 46:
				v.TypedShortcut(&fyne.ShortcutSelectAll{})
			case op < 50:
				mods := []fyne.KeyModifier{fyne.KeyModifierAlt, fyne.KeyModifierSuper, fyne.KeyModifierControl | fyne.KeyModifierShift}
				v.TypedShortcut(&desktop.CustomShortcut{KeyName: keys[int(op)%6], Modifier: mods[int(op)%3]})
			case op < 54:
				p := fyne.NewPos(float32(op)*7-300, float32(op)*3-100)
				v.MouseDown(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: p}, Button: desktop.MouseButtonPrimary})
			case op < 58:
				v.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(float32(op)*5, float32(op)*9)}, Dragged: fyne.NewDelta(3, 40)})
			case op < 60:
				v.DragEnd()
			case op < 62:
				v.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(-20, float32(op)-61)})
			case op < 64:
				v.DoubleTapped(&fyne.PointEvent{Position: fyne.NewPos(float32(op)*4, 30)})
			default:
				v.TypedRune(rune(op))
			}
			checkView(t, v)
		}
	})
}

func checkView(t *testing.T, v *View) {
	t.Helper()
	d := &v.doc
	if len(d.lines) == 0 {
		t.Fatal("no lines")
	}
	for _, p := range []pos{v.caret, v.anchor} {
		if p.line < 0 || p.line >= len(d.lines) || p.off < 0 || p.off > len(d.lines[p.line]) {
			t.Fatalf("position %v outside the document (%d lines)", p, len(d.lines))
		}
	}
	for i, l := range d.lines {
		if strings.IndexByte(l, '\n') >= 0 {
			t.Fatalf("line %d contains a newline", i)
		}
	}
	if got := d.joined(); got != strings.Join(d.lines, "\n") {
		t.Fatal("cached text out of date")
	}
	if v.scrollX < 0 || v.scrollY < 0 {
		t.Fatalf("negative scroll %v %v", v.scrollX, v.scrollY)
	}
	if len(d.states) > len(d.lines) {
		t.Fatalf("%d states for %d lines", len(d.states), len(d.lines))
	}
}
