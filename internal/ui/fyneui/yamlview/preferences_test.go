package yamlview

import (
	"testing"

	"fyne.io/fyne/v2"
)

// The Preferences' editor settings: the tab size is the indentation step of Tab, Shift+Tab,
// Backspace in the indentation and a new line after a key; hiding the line numbers narrows the
// gutter to a margin; the text size replaces the theme's.
func TestEditorPreferences(t *testing.T) {
	v, _ := editable(t, "a:\nb: 1", 1, 0)
	v.SetTabSize(4)
	key(v, fyne.KeyTab)
	wantText(t, v, "a:\n    b: 1")
	key(v, fyne.KeyBackspace)
	wantText(t, v, "a:\nb: 1")
	v, _ = editable(t, "a:", 0, 2)
	v.SetTabSize(4)
	key(v, fyne.KeyReturn)
	wantText(t, v, "a:\n    ")
	v, _ = editable(t, "a: 1\nb: 2", 0, 0)
	v.SetTabSize(3)
	shift(v, fyne.KeyDown, fyne.KeyEnd)
	key(v, fyne.KeyTab)
	wantText(t, v, "   a: 1\n   b: 2")
	shift(v, fyne.KeyTab)
	wantText(t, v, "a: 1\nb: 2")

	before := v.metrics()
	v.SetLineNumbers(false)
	after := v.metrics()
	if after.numW != 0 || after.gutterW >= before.gutterW {
		t.Errorf("without line numbers the gutter is %v (numbers %v), was %v", after.gutterW, after.numW, before.gutterW)
	}
	v.SetLineNumbers(true)
	if m := v.metrics(); m.gutterW != before.gutterW {
		t.Errorf("line numbers back: gutter %v, want %v", m.gutterW, before.gutterW)
	}
	v.SetTextSize(20)
	if m := v.metrics(); m.textSize != 20 || m.lh <= before.lh {
		t.Errorf("text size 20: %v, line height %v (was %v)", m.textSize, m.lh, before.lh)
	}
	v.SetTextSize(0)
	if m := v.metrics(); m.textSize != before.textSize {
		t.Errorf("text size back to the theme's: %v, want %v", m.textSize, before.textSize)
	}
}
