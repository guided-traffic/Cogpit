package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// F1–F12 and ⇧F1–⇧F12 open the hotbar's clusters and banks in the plain search fields too (the
// owner's A64, 2026-10-14 evening): the view's search, the navigator's, the logs tab's, the Argo CD
// dashboard's activity filter, the editor's Find field, the context picker's and the command
// palette's filters. A text field takes every key while it has the focus, so the window's key handler
// never sees them there; these fields hand them to the hotbar first. The terminal and the YAML
// editor keep their F-keys: a shell or a program in it may use them, and the editor is a text area,
// not a search field. A field that is no search (a name, a path, a value in a dialog) keeps them too.

// hotbarKey hands an F-key typed in a search field to the hotbar: true when it took it. Shift comes
// from the desktop driver, as F-keys arrive without their modifiers.
func hotbarKey(sh *shell, ev *fyne.KeyEvent) bool {
	if functionKey(ev.Name) == 0 || sh.hot == nil {
		return false
	}
	return sh.hot.key(ev.Name, shiftHeld())
}

// searchEntry is a plain search field whose F-keys are the hotbar's.
type searchEntry struct {
	widget.Entry
	sh *shell
}

func newSearchEntry(sh *shell) *searchEntry {
	e := &searchEntry{sh: sh}
	e.ExtendBaseWidget(e)
	return e
}

// TypedKey implements fyne.Focusable.
func (e *searchEntry) TypedKey(ev *fyne.KeyEvent) {
	if hotbarKey(e.sh, ev) {
		return
	}
	e.Entry.TypedKey(ev)
}
