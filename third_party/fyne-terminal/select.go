package terminal

import (
	"fyne.io/fyne/v2"
)

// getSelectedRange returns the current selection range, start row, start col, end row, end col
// It always returns a positive selection
func (t *Terminal) getSelectedRange() (int, int, int, int) {
	if t.selStart == nil || t.selEnd == nil {
		return 0, 0, 0, 0
	}

	startRow := t.selStart.Row
	startCol := t.selStart.Col
	endRow := t.selEnd.Row
	endCol := t.selEnd.Col

	if t.blockMode {
		if startCol > endCol {
			// Swap the start and end colums
			startCol, endCol = endCol, startCol
		}

		if startRow > endRow {
			// Swap the start and end rows
			startRow, endRow = endRow, startRow
		}

		return startRow - 1, startCol - 1, endRow - 1, endCol - 1
	}
	// Check if the user has selected in reverse
	if startRow > endRow || (startRow == endRow && startCol > endCol) {
		// Swap the start and end rows and columns
		startRow, endRow = endRow, startRow
		startCol, endCol = endCol, startCol
	}

	return startRow - 1, startCol - 1, endRow - 1, endCol - 1
}

// Cogpit patch: the selection is by the rows of the scrollback and the screen (scrollback.go),
// and the view highlights it in its copies of the rows.
func (t *Terminal) highlightSelectedText() {
	t.Refresh()
}

func (t *Terminal) clearSelectedText() {
	t.clearSelection()
	t.Refresh()
}

// SelectedText gets the text that is currently selected.
func (t *Terminal) SelectedText() string {
	if !t.hasSelectedText() {
		return ""
	}
	sr, sc, er, ec := t.getSelectedRange()
	return t.selectedText(sr, sc, er, ec)
}

func (t *Terminal) copySelectedText(clipboard fyne.Clipboard) {
	// copy start and end sel to clipboard and clear the sel style
	text := t.SelectedText()
	clipboard.SetContent(text)
	t.clearSelectedText()
}

func (t *Terminal) pasteText(clipboard fyne.Clipboard) {
	content := clipboard.Content()

	if t.bracketedPasteMode {
		_, _ = t.in.Write(
			append(
				append(
					[]byte{asciiEscape, '[', '2', '0', '0', '~'},
					[]byte(content)...,
				),
				[]byte{asciiEscape, '[', '2', '0', '1', '~'}...,
			),
		)
		return
	}
	_, _ = t.in.Write([]byte(content))
}

func (t *Terminal) hasSelectedText() bool {
	return t.selStart != nil && t.selEnd != nil
}

func (t *Terminal) selectClipboard() *selectClipboard {
	if t.selectClipSource == nil {
		t.selectClipSource = &selectClipboard{t: t}
	}

	return t.selectClipSource
}

type selectClipboard struct {
	t *Terminal

	content string
}

func (s *selectClipboard) Content() string {
	return s.content
}

func (s *selectClipboard) SetContent(c string) {
	s.content = c
}
