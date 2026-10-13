package terminal

import "fyne.io/fyne/v2"

// Cogpit patch (addition): what the secondary mouse button does is the application's choice. The
// widget pasted the text last selected in it on every right click; an application that offers
// "copy and paste with the right mouse button" as a preference, or none at all, could not change
// that.

// RightClickAction is what a click of the secondary (right) mouse button does in the terminal.
type RightClickAction int

const (
	// RightClickPasteSelection pastes the text last selected in this terminal, as before. It is the
	// default.
	RightClickPasteSelection RightClickAction = iota
	// RightClickNone neither copies nor pastes.
	RightClickNone
	// RightClickCopyPaste copies the selection to the clipboard when there is one, and pastes the
	// clipboard otherwise (bracketed when the program asks for it).
	RightClickCopyPaste
)

// SetRightClickAction chooses what the secondary mouse button does. A program that asked for mouse
// events still receives the click. Call it on Fyne's main goroutine.
func (t *Terminal) SetRightClickAction(a RightClickAction) {
	t.rightClick = a
}

// rightClicked runs the right-click action. selected is the selection the click found, which
// MouseDown clears before it calls this.
func (t *Terminal) rightClicked(selected string) {
	switch t.rightClick {
	case RightClickPasteSelection:
		t.pasteText(t.selectClipboard())
	case RightClickCopyPaste:
		clip := fyne.CurrentApp().Clipboard()
		if selected != "" {
			clip.SetContent(selected)
			return
		}
		t.pasteText(clip)
	}
}
