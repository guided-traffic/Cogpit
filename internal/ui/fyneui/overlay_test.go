package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// The "Connect to a cluster" mask closes when the user clicks outside it on Cogpit, and on
// Escape; a click on the mask's own panel keeps it (owner, 2026-10-12).
func TestPickerClosesOnAClickOutside(t *testing.T) {
	f := connectedFake(t, 3)
	sh, w, q := newTestShell(t, f, Options{})
	sh.picker.show()
	q.until(t, "the picker", func() bool { return sh.picker.visible() })
	settle(w)

	panel := sh.picker.popup.panel
	inside := panel.Position().AddXY(4, 4) // the panel's padding, outside every widget of it
	test.TapCanvas(w.Canvas(), inside)
	if !sh.picker.visible() {
		t.Fatal("a click on the picker's own panel closed it")
	}
	test.TapCanvas(w.Canvas(), fyne.NewPos(5, w.Canvas().Size().Height-5))
	if sh.picker.visible() {
		t.Fatal("a click outside the picker left it open")
	}

	sh.picker.show()
	q.until(t, "the picker again", func() bool { return sh.picker.visible() })
	w.Canvas().Unfocus()
	sh.win.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if sh.picker.visible() {
		t.Error("Escape left the picker open")
	}
}
