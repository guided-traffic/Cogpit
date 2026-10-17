package yamlview

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
)

// clockOf gives v's history a clock the test moves; it starts at a fixed time.
func clockOf(v *View) *time.Time {
	now := time.Date(2026, 10, 17, 20, 0, 0, 0, time.UTC)
	v.hist.now = func() time.Time { return now }
	return &now
}

func undo(v *View) { v.TypedShortcut(&fyne.ShortcutUndo{}) }
func redo(v *View) { v.TypedShortcut(&fyne.ShortcutRedo{}) }
func redoAlt(v *View) {
	v.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyZ, Modifier: primaryModifier | fyne.KeyModifierShift})
}
func undoAll(v *View) {
	for v.Undo() {
	}
}
func redoAll(v *View) {
	for v.Redo() {
	}
}
func wantSel(t *testing.T, v *View, want string) {
	t.Helper()
	if got := v.SelectedText(); got != want {
		t.Fatalf("selection %q, want %q", got, want)
	}
}

func TestUndoTypingRun(t *testing.T) {
	v, _ := editable(t, "name: web", 0, 9)
	clockOf(v)
	changes := 0
	v.OnChanged = func() { changes++ }
	typeText(v, "-api")
	wantText(t, v, "name: web-api")
	undo(v)
	wantText(t, v, "name: web") // the run is one step
	wantCursor(t, v, 0, 9)
	redo(v)
	wantText(t, v, "name: web-api")
	wantCursor(t, v, 0, 13)
	undo(v)
	redoAlt(v) // ⇧⌘Z (Ctrl+Shift+Z elsewhere) redoes too
	wantText(t, v, "name: web-api")
	if changes != 4+4 {
		t.Fatalf("OnChanged called %d times, want 8 (4 typed, 2 undos, 2 redos)", changes)
	}
	undo(v)
	undo(v) // nothing more to undo
	wantText(t, v, "name: web")
}

func TestTypingRunBreaks(t *testing.T) {
	t.Run("a pause", func(t *testing.T) {
		v, _ := editable(t, "", 0, 0)
		now := clockOf(v)
		typeText(v, "ab")
		*now = now.Add(typingPause)
		typeText(v, "cd")
		undo(v)
		wantText(t, v, "ab")
		undo(v)
		wantText(t, v, "")
	})
	t.Run("a caret move", func(t *testing.T) {
		v, _ := editable(t, "", 0, 0)
		clockOf(v)
		typeText(v, "ab")
		key(v, fyne.KeyLeft, fyne.KeyRight) // back where the run ended
		typeText(v, "cd")
		undo(v)
		wantText(t, v, "ab")
	})
	t.Run("a new line", func(t *testing.T) {
		v, _ := editable(t, "spec:", 0, 5)
		clockOf(v)
		typeText(v, " {}")
		key(v, fyne.KeyReturn)
		typeText(v, "x: 1")
		wantText(t, v, "spec: {}\nx: 1")
		undo(v) // the new line and what was typed on it
		wantText(t, v, "spec: {}")
		wantCursor(t, v, 0, 8)
		undo(v)
		wantText(t, v, "spec:")
	})
	t.Run("another kind of edit", func(t *testing.T) {
		v, _ := editable(t, "", 0, 0)
		clockOf(v)
		typeText(v, "abc")
		key(v, fyne.KeyBackspace)
		typeText(v, "d")
		wantText(t, v, "abd")
		undo(v)
		wantText(t, v, "ab")
		undo(v)
		wantText(t, v, "abc")
		undo(v)
		wantText(t, v, "")
	})
	t.Run("the focus leaving", func(t *testing.T) {
		v, _ := editable(t, "", 0, 0)
		clockOf(v)
		typeText(v, "ab")
		v.FocusLost()
		v.FocusGained()
		typeText(v, "cd")
		undo(v)
		wantText(t, v, "ab")
	})
}

func TestUndoDeletions(t *testing.T) {
	v, _ := editable(t, "replicas: 3\nimage: web", 0, 11)
	clockOf(v)
	key(v, fyne.KeyBackspace, fyne.KeyBackspace, fyne.KeyBackspace) // "3", " ", ":"
	wantText(t, v, "replicas\nimage: web")
	undo(v) // consecutive Backspaces are one step
	wantText(t, v, "replicas: 3\nimage: web")
	wantCursor(t, v, 0, 11)

	v.SetCursorPosition(1, 0)
	key(v, fyne.KeyDelete, fyne.KeyDelete) // forward Deletes are one step
	wantText(t, v, "replicas: 3\nage: web")
	undo(v)
	wantText(t, v, "replicas: 3\nimage: web")

	// Backspace across a line end joins the run.
	v.SetCursorPosition(1, 2)
	key(v, fyne.KeyBackspace, fyne.KeyBackspace, fyne.KeyBackspace)
	wantText(t, v, "replicas: 3age: web")
	undo(v)
	wantText(t, v, "replicas: 3\nimage: web")

	// ⌥⌫ deletes a word as a Backspace does.
	v.SetCursorPosition(1, 10)
	word := &desktop.CustomShortcut{KeyName: fyne.KeyBackspace, Modifier: fyne.KeyModifierAlt}
	v.TypedShortcut(word)
	v.TypedShortcut(word)
	wantText(t, v, "replicas: 3\nimage")
	undo(v)
	wantText(t, v, "replicas: 3\nimage: web")
}

func TestUndoRestoresTheSelection(t *testing.T) {
	v, _ := editable(t, "hello world", 0, 1)
	clockOf(v)
	v.SelectRange(0, 1, 0, 4)
	wantSel(t, v, "ell")
	key(v, fyne.KeyBackspace) // a deleted selection is a step of its own
	key(v, fyne.KeyBackspace) // and the next Backspace another
	wantText(t, v, "o world")
	undo(v)
	wantText(t, v, "ho world")
	undo(v)
	wantText(t, v, "hello world")
	wantSel(t, v, "ell")

	typeText(v, "ipp") // typing over a selection: one step with what it replaced
	wantText(t, v, "hippo world")
	undo(v)
	wantText(t, v, "hello world")
	wantSel(t, v, "ell")
	redo(v)
	wantText(t, v, "hippo world")
	wantSel(t, v, "")
	wantCursor(t, v, 0, 4)
}

func TestUndoPasteCutAndIndent(t *testing.T) {
	v, app := editable(t, "a: 1\nb: 2", 0, 4)
	clockOf(v)
	app.clip.content = "\r\nc: 3" // CRLF from the clipboard
	typeText(v, "0")
	v.TypedShortcut(&fyne.ShortcutPaste{Clipboard: app.clip})
	typeText(v, "9")
	wantText(t, v, "a: 10\nc: 39\nb: 2")
	undo(v)
	wantText(t, v, "a: 10\nc: 3\nb: 2") // the typing after the paste
	undo(v)
	wantText(t, v, "a: 10\nb: 2") // the paste, one step
	undo(v)
	wantText(t, v, "a: 1\nb: 2")
	redoAll(v)
	wantText(t, v, "a: 10\nc: 39\nb: 2")

	v.SetText("x: 1\ny: 2\nz: 3")
	v.SelectRange(0, 0, 1, 4)
	v.TypedShortcut(&fyne.ShortcutCut{Clipboard: app.clip})
	wantText(t, v, "\nz: 3")
	undo(v)
	wantText(t, v, "x: 1\ny: 2\nz: 3")
	wantSel(t, v, "x: 1\ny: 2")

	key(v, fyne.KeyTab) // indents both lines: one step
	wantText(t, v, "  x: 1\n  y: 2\nz: 3")
	shift(v, fyne.KeyTab)
	shift(v, fyne.KeyTab)
	wantText(t, v, "x: 1\ny: 2\nz: 3")
	undo(v)
	wantText(t, v, "  x: 1\n  y: 2\nz: 3")
	undo(v)
	undo(v)
	wantText(t, v, "x: 1\ny: 2\nz: 3")
	wantSel(t, v, "x: 1\ny: 2")
	redo(v) // the caret and the selection as the indent left them
	wantText(t, v, "  x: 1\n  y: 2\nz: 3")
	wantSel(t, v, "x: 1\n  y: 2")

	// Replace All (the find widget) is one step.
	v.SelectAll()
	v.ReplaceSelection("everything else")
	undo(v)
	wantText(t, v, "  x: 1\n  y: 2\nz: 3")
}

func TestAnEditAfterUndoDropsTheRedo(t *testing.T) {
	v, _ := editable(t, "", 0, 0)
	clockOf(v)
	typeText(v, "one")
	v.SetCursorPosition(0, 3)
	typeText(v, " two")
	undo(v)
	typeText(v, " three")
	wantText(t, v, "one three")
	redo(v) // " two" is gone for good
	wantText(t, v, "one three")
	undo(v)
	wantText(t, v, "one")
}

// TestTextFromOutsideClearsTheHistory: a load, a Revert, a draft restored or an agent's text
// replace the document; the history of the old text does not apply to the new one (D-84).
func TestTextFromOutsideClearsTheHistory(t *testing.T) {
	v, _ := editable(t, "a: 1", 0, 4)
	clockOf(v)
	typeText(v, "0")
	v.UpdateText("a: 10") // the same text: the history stays
	undo(v)
	wantText(t, v, "a: 1")
	redo(v)

	v.UpdateText("a: 2") // another text: the history goes
	if v.Undo() || v.Redo() {
		t.Fatal("undo or redo after UpdateText with another text")
	}
	wantText(t, v, "a: 2")

	typeText(v, "x")
	v.SetText("b: 1")
	if v.Undo() {
		t.Fatal("undo after SetText")
	}
	typeText(v, "y")
	v.ClearHistory()
	if v.Undo() || len(v.hist.steps) != 0 || v.hist.bytes != 0 {
		t.Fatalf("ClearHistory kept %d steps, %d bytes", len(v.hist.steps), v.hist.bytes)
	}
}

func TestUndoReadOnlyAndNothingToUndo(t *testing.T) {
	v, _ := editable(t, "a: 1", 0, 4)
	clockOf(v)
	nothing := 0
	v.OnNothingToUndo = func() { nothing++ }
	typeText(v, "0")
	v.SetReadOnly(true)
	undo(v)
	wantText(t, v, "a: 10") // a read-only view neither undoes nor hands the shortcut on
	if nothing != 0 {
		t.Fatal("OnNothingToUndo called for a read-only view")
	}
	v.SetReadOnly(false)
	undo(v)
	wantText(t, v, "a: 1")
	if nothing != 0 {
		t.Fatal("OnNothingToUndo called although the view undid an edit")
	}
	undo(v)
	if nothing != 1 {
		t.Fatalf("OnNothingToUndo called %d times, want 1", nothing)
	}
}

func TestHistoryBounds(t *testing.T) {
	v, _ := editable(t, "", 0, 0)
	clockOf(v)
	for range historySteps + 5 {
		typeText(v, "x")
		v.hist.seal() // a step each
	}
	if len(v.hist.steps) != historySteps {
		t.Fatalf("%d steps kept, want %d", len(v.hist.steps), historySteps)
	}
	undoAll(v)
	wantText(t, v, strings.Repeat("x", 5)) // the five oldest steps are gone

	v.SetText("")
	typeText(v, strings.Repeat("r", runBytes+10)) // a run without a pause ends at runBytes
	undo(v)
	wantText(t, v, strings.Repeat("r", runBytes))

	var h history
	big := strings.Repeat("y", 12<<20)
	for i := range 3 {
		h.record(step{start: pos{0, i}, new: big, kind: editOther})
	}
	if len(h.steps) != 2 || h.bytes != 2*len(big) || h.index != 2 {
		t.Fatalf("bytes cap: %d steps, %d bytes, index %d", len(h.steps), h.bytes, h.index)
	}
	huge := strings.Repeat("z", historyBytes+1)
	h.record(step{old: huge, kind: editOther})
	if len(h.steps) != 1 || h.bytes != len(huge) { // the newest step is kept, however large
		t.Fatalf("a step over the cap: %d steps, %d bytes", len(h.steps), h.bytes)
	}
}

// TestUndoRedoRoundTrip runs random edits, undos and redos, then undoes everything back to the text
// it started with and redoes everything to the text of its newest edit.
func TestUndoRedoRoundTrip(t *testing.T) {
	for seed := range uint64(50) {
		v, app := editable(t, sampleYAML[:400], 3, 2)
		now := clockOf(v)
		rng := rand.New(rand.NewPCG(seed, 84))
		start := v.Text()
		for range 300 {
			*now = now.Add(time.Duration(rng.IntN(1500)) * time.Millisecond)
			switch rng.IntN(11) {
			case 0, 1, 2:
				typeText(v, string(rune('a'+rng.IntN(26))))
			case 3:
				key(v, fyne.KeyReturn)
			case 4:
				key(v, fyne.KeyBackspace)
			case 5:
				key(v, fyne.KeyDelete)
			case 6:
				key(v, fyne.KeyTab)
			case 7:
				shift(v, fyne.KeyTab)
			case 8:
				app.clip.content = []string{"k: v", "\n- a\n- b\n", "é"}[rng.IntN(3)]
				v.TypedShortcut(&fyne.ShortcutPaste{Clipboard: app.clip})
			case 9:
				l := rng.IntN(v.LineCount())
				v.SelectRange(l, rng.IntN(6), min(l+rng.IntN(3), v.LineCount()-1), rng.IntN(6))
			case 10:
				if rng.IntN(2) == 0 {
					v.Undo()
				} else {
					v.Redo()
				}
			}
		}
		redoAll(v) // a run that ended on an undo: its newest edit's text
		end := v.Text()
		undoAll(v)
		if got := v.Text(); got != start {
			t.Fatalf("seed %d: undoing everything gives\n%q\nwant\n%q", seed, got, start)
		}
		redoAll(v)
		if got := v.Text(); got != end {
			t.Fatalf("seed %d: redoing everything gives\n%q\nwant\n%q", seed, got, end)
		}
	}
}
