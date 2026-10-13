package terminal

import (
	"bytes"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	widget2 "github.com/fyne-io/terminal/internal/widget"
	"github.com/stretchr/testify/assert"
)

// selectionTerm is a terminal showing "Hello World" whose input goes to the returned buffer.
func selectionTerm() (*Terminal, *bytes.Buffer) {
	grid := widget2.NewTermGrid()
	row := widget.TextGridRow{}
	for _, r := range "Hello World" {
		row.Cells = append(row.Cells, widget.TextGridCell{Rune: r})
	}
	grid.Rows = []widget.TextGridRow{row}
	in := &bytes.Buffer{}
	return &Terminal{content: grid, in: NopCloser(in)}, in
}

// selectHello selects "Hello" as a drag does, which also keeps it as the terminal's own selection.
func selectHello(term *Terminal) {
	term.selStart, term.selEnd = &position{Row: 1, Col: 1}, &position{Row: 1, Col: 5}
	term.selectClipboard().SetContent(term.SelectedText())
}

func rightClick(term *Terminal) {
	term.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonSecondary})
}

// Cogpit patch: TestRightClickAction checks the three right-click actions. The default pastes the
// last selection (the widget's behaviour before the patch); none does nothing; copy and paste copies
// a selection to the clipboard, and pastes the clipboard when nothing is selected. Every click
// clears the selection, and the primary button never copies or pastes.
func TestRightClickAction(t *testing.T) {
	app := test.NewTempApp(t)
	clip := app.Clipboard()

	t.Run("paste the selection (default)", func(t *testing.T) {
		clip.SetContent("from the clipboard")
		term, in := selectionTerm()
		selectHello(term)
		rightClick(term)
		assert.Equal(t, "Hello", in.String())
		assert.False(t, term.hasSelectedText())
		assert.Equal(t, "from the clipboard", clip.Content())
	})

	t.Run("none", func(t *testing.T) {
		clip.SetContent("from the clipboard")
		term, in := selectionTerm()
		term.SetRightClickAction(RightClickNone)
		selectHello(term)
		rightClick(term)
		rightClick(term)
		assert.Empty(t, in.String())
		assert.False(t, term.hasSelectedText())
		assert.Equal(t, "from the clipboard", clip.Content())
	})

	t.Run("copy, else paste", func(t *testing.T) {
		clip.SetContent("kubectl get pods")
		term, in := selectionTerm()
		term.SetRightClickAction(RightClickCopyPaste)
		term.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary}) // nothing selected
		assert.Empty(t, in.String())
		rightClick(term) // nothing selected: paste
		assert.Equal(t, "kubectl get pods", in.String())

		in.Reset()
		selectHello(term)
		rightClick(term) // a selection: copy, type nothing
		assert.Equal(t, "Hello", clip.Content())
		assert.Empty(t, in.String())
		assert.False(t, term.hasSelectedText())

		term.bracketedPasteMode = true
		rightClick(term)
		assert.Equal(t, "\x1b[200~Hello\x1b[201~", in.String())

		in.Reset()
		selectHello(term)
		term.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary}) // clears, copies nothing
		clip.SetContent("unchanged")
		assert.Empty(t, in.String())
		assert.False(t, term.hasSelectedText())
		assert.Equal(t, "unchanged", clip.Content())
	})
}

// Cogpit patch: TestCopyPasteKeys_WindowsAndLinux checks the copy and paste keys off macOS, on any
// system the tests run on: Ctrl+Shift+C and Ctrl+Insert copy the selection to the clipboard,
// Ctrl+Shift+V and Shift+Insert paste it; Ctrl+C and Ctrl+V stay the interrupt and literal-next
// characters a shell needs.
func TestCopyPasteKeys_WindowsAndLinux(t *testing.T) {
	saved := goos
	goos = "linux"
	defer func() { goos = saved }()
	app := test.NewTempApp(t)
	clip := app.Clipboard()
	term, in := selectionTerm()
	term.setupShortcuts()
	ctrlShift := func(k fyne.KeyName) fyne.Shortcut {
		return &desktop.CustomShortcut{KeyName: k, Modifier: fyne.KeyModifierShift | fyne.KeyModifierShortcutDefault}
	}

	selectHello(term)
	clip.SetContent("")
	term.TypedShortcut(ctrlShift(fyne.KeyC))
	assert.Equal(t, "Hello", clip.Content(), "Ctrl+Shift+C")

	selectHello(term)
	clip.SetContent("")
	term.TypedShortcut(&fyne.ShortcutCopy{Secondary: true})
	assert.Equal(t, "Hello", clip.Content(), "Ctrl+Insert")
	assert.False(t, term.hasSelectedText())

	clip.SetContent("ls -l")
	term.TypedShortcut(ctrlShift(fyne.KeyV))
	assert.Equal(t, "ls -l", in.String(), "Ctrl+Shift+V")

	in.Reset()
	clip.SetContent("pwd")
	term.TypedShortcut(&fyne.ShortcutPaste{Secondary: true, Clipboard: clip})
	assert.Equal(t, "pwd", in.String(), "Shift+Insert")

	in.Reset()
	term.TypedShortcut(&fyne.ShortcutCopy{})
	term.TypedShortcut(&fyne.ShortcutPaste{})
	assert.Equal(t, []byte{0x03, 0x16}, in.Bytes(), "Ctrl+C and Ctrl+V")
}
