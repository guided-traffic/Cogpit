package terminal

import (
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	widget2 "github.com/fyne-io/terminal/internal/widget"
)

// Cogpit patch: scrolling through the scrollback (history.go). The rows are numbered from the
// first the scrollback held: the scrollback's rows, then the screen's. A row keeps its number when
// it scrolls off the screen into the scrollback, so a selection stays on its text. The view draws
// the rows from viewTop on: the screen while it follows the output, or the rows the user
// scrolled back to. It holds copies of these rows only; the content holds the screen and is never
// drawn, and a grid of the whole scrollback drew all of its rows on every refresh.

// screenStart is the number of the screen's first row.
func (t *Terminal) screenStart() int {
	return t.history().starts[len(t.hist.starts)-1]
}

// topRow is the number of the first row there is: the notice of dropped lines, if there were
// any, else the scrollback's first.
func (t *Terminal) topRow() int {
	h := t.history()
	if h.dropped > 0 {
		return h.starts[0] - 1
	}
	return h.starts[0]
}

func (t *Terminal) history() *history {
	if t.hist == nil {
		t.hist = newHistory(int(t.config.Columns))
	}
	return t.hist
}

// rowCells returns the cells of a row, and whether the line on it goes on in the next row.
func (t *Terminal) rowCells(row int) (cells []widget.TextGridCell, continues bool) {
	h := t.history()
	start := t.screenStart()
	switch {
	case row >= start:
		r := row - start
		if r >= len(t.content.Rows) {
			return nil, false
		}
		return t.content.Rows[r].Cells, t.content.Rows[r].Style == wrapMark
	case row >= h.starts[0]:
		line, seg := h.locate(row - h.starts[0])
		last := seg == h.rowsOf(h.lines[line].n)-1
		return h.row(line, seg), !last || (line == len(h.lines)-1 && h.open)
	case row == h.starts[0]-1 && h.dropped > 0:
		return t.droppedNotice(), false
	}
	return nil, false
}

// droppedNotice is the row above the scrollback once the oldest lines were dropped.
func (t *Terminal) droppedNotice() []widget.TextGridCell {
	text := "… " + strconv.FormatInt(t.history().dropped, 10) + " older lines dropped (scrollback limit)"
	style := &widget.CustomTextGridStyle{FGColor: theme.Color(theme.ColorNameDisabled)}
	cells := make([]widget.TextGridCell, 0, len(text))
	for _, r := range text {
		cells = append(cells, widget.TextGridCell{Rune: r, Style: style})
	}
	return cells
}

// refreshView fills the view with the rows from viewTop on, the selection highlighted.
func (t *Terminal) refreshView() {
	if t.view == nil || t.content == nil {
		return
	}
	start := t.screenStart()
	if !t.scrolledBack || t.viewTop > start {
		t.viewTop = start
	}
	t.viewTop = max(t.viewTop, t.topRow())
	t.scrolledBack = t.viewTop < start

	selected := t.hasSelectedText()
	rows := t.viewRows[:0]
	for y := 0; y < int(t.config.Rows); y++ {
		cells, _ := t.rowCells(t.viewTop + y)
		if selected && cells != nil {
			cells = append([]widget.TextGridCell(nil), cells...) // the highlight changes the copy
		}
		rows = append(rows, widget.TextGridRow{Cells: cells})
	}
	t.viewRows = rows
	t.view.Rows = rows
	if selected {
		sr, sc, er, ec := t.getSelectedRange()
		sr, er = sr-t.viewTop, er-t.viewTop
		if er >= 0 && sr < len(rows) {
			if sr < 0 {
				sr, sc = 0, 0
			}
			if er >= len(rows) {
				er, ec = len(rows)-1, int(t.config.Columns)-1
			}
			widget2.HighlightRange(t.view, t.blockMode, sr, sc, er, ec, highlightBitMask)
		}
	}
	t.view.Refresh()
	if t.bar != nil {
		t.bar.Refresh()
	}
}

// scrollBy moves the view up by lines rows, down for a negative count.
func (t *Terminal) scrollBy(lines int) {
	t.scrollTo(t.viewTop - lines)
}

// scrollTo moves the view's top to a row, clamped to the rows there are.
func (t *Terminal) scrollTo(top int) {
	start := t.screenStart()
	t.viewTop = max(t.topRow(), min(top, start))
	t.scrolledBack = t.viewTop < start
	t.Refresh()
}

// isCopy reports whether a shortcut copies the selection: ⌘C on macOS, ctrl+shift+C and ctrl+Insert
// elsewhere, where ctrl+C goes to the shell.
func isCopy(s fyne.Shortcut) bool {
	switch s := s.(type) {
	case *fyne.ShortcutCopy:
		return goos == "darwin" || s.Secondary
	case *desktop.CustomShortcut:
		return s.KeyName == fyne.KeyC && s.Modifier == fyne.KeyModifierShift|fyne.KeyModifierShortcutDefault
	}
	return false
}

// isModifier reports whether a key only modifies others, which sends nothing to the shell.
func isModifier(name fyne.KeyName) bool {
	switch name {
	case desktop.KeyShiftLeft, desktop.KeyShiftRight, desktop.KeyControlLeft, desktop.KeyControlRight,
		desktop.KeyAltLeft, desktop.KeyAltRight, desktop.KeySuperLeft, desktop.KeySuperRight,
		desktop.KeyCapsLock, desktop.KeyMenu:
		return true
	}
	return false
}

// followOutput returns the view to the screen, where it follows the output.
func (t *Terminal) followOutput() {
	if !t.scrolledBack {
		return
	}
	t.scrolledBack = false
	t.Refresh()
}

// Scrolled is the mouse wheel or the trackpad: a program that asked for the mouse gets it as its
// buttons 4 and 5, a full-screen program (the alternate screen) as the arrow keys, and otherwise
// it scrolls through the scrollback.
func (t *Terminal) Scrolled(ev *fyne.ScrollEvent) {
	cell := t.guessCellSize()
	t.scrollRemainder += ev.Scrolled.DY
	lines := int(t.scrollRemainder / cell.Height)
	if lines == 0 {
		return
	}
	t.scrollRemainder -= float32(lines) * cell.Height

	count := max(lines, -lines)
	switch {
	case t.onMouseDown != nil:
		btn := byte(64) // wheel up
		if lines < 0 {
			btn = 65
		}
		p := t.screenPosition(ev.Position)
		report := []byte{asciiEscape, '[', 'M', 32 + btn, 32 + byte(p.Col), 32 + byte(p.Row)}
		for i := 0; i < count; i++ {
			_, _ = t.Write(report)
		}
	case t.altBufferActive:
		key := fyne.KeyUp
		if lines < 0 {
			key = fyne.KeyDown
		}
		for i := 0; i < count; i++ {
			t.typeCursorKey(key) // as the arrow keys: ESC O A in a full-screen program's mode
		}
	default:
		t.scrollBy(lines)
	}
}

// screenPosition returns the screen's cell at a position, for a program that asked for the
// mouse; a position over the scrollback is on the screen's first row.
func (t *Terminal) screenPosition(pos fyne.Position) position {
	p := t.getTermPosition(pos)
	p.Row = max(p.Row-t.screenStart(), 1)
	return p
}

// markWrapped marks a screen row that the output goes on from in the next row.
func (t *Terminal) markWrapped(row int) {
	if row >= 0 && row < len(t.content.Rows) {
		t.content.Rows[row].Style = wrapMark
	}
}

// unwrapPadding takes back the wrap into the cursor's row when that row holds nothing but blanks
// at a carriage return: a shell's padding to the right margin, as zsh's PROMPT_SP writes after a
// line without a line feed so that its prompt starts on a row of its own, not a line that goes on.
// Joined, the padding pushed the prompt into the middle of a row at another width.
func (t *Terminal) unwrapPadding() {
	if t.cursorRow < len(t.content.Rows) && len(trimBlanks(t.content.Rows[t.cursorRow].Cells)) > 0 {
		return
	}
	switch {
	case t.cursorRow > 0 && t.cursorRow-1 < len(t.content.Rows):
		if t.content.Rows[t.cursorRow-1].Style == wrapMark {
			t.content.Rows[t.cursorRow-1].Style = nil
		}
	case t.cursorRow == 0 && t.hist != nil:
		t.hist.open = false
	}
}

// pushToHistory moves the screen's top rows into the scrollback, as scrolling the whole screen
// up does; the alternate screen keeps none.
func (t *Terminal) pushToHistory(rows int) {
	if t.altBufferActive || t.scrollTop != 0 {
		return
	}
	h := t.history()
	for i := 0; i < rows; i++ {
		if i < len(t.content.Rows) {
			row := t.content.Rows[i]
			h.push(row, row.Style == wrapMark)
		} else {
			h.push(widget.TextGridRow{}, false)
		}
	}
}

// clearHistory drops the scrollback, as erase saved lines (CSI 3 J) asks.
func (t *Terminal) clearHistory() {
	t.history().clear()
	t.clearSelection()
}

func (t *Terminal) clearSelection() {
	t.blockMode = false
	t.selecting = false
	t.selStart = nil
	t.selEnd = nil
}

// selectionOnScreen reports whether the selection reaches the screen, whose rows output changes.
func (t *Terminal) selectionOnScreen() bool {
	if !t.hasSelectedText() {
		return false
	}
	_, _, er, _ := t.getSelectedRange()
	return er >= t.screenStart()
}

// reflow wraps the screen at a new size, the main screen's lines and the scrollback's together:
// the screen goes into the scrollback, which wraps its lines at the new width, and the last
// rows come back as the screen, the cursor on the cell it was on. Rows below the cursor and the
// last with text are left out; the view keeps the line at its top.
func (t *Terminal) reflow(rows, cols int) {
	h := t.history()
	anchorLine, anchorCell := int64(-1), 0
	if t.scrolledBack && t.viewTop < t.screenStart() && t.viewTop >= h.starts[0] {
		line, seg := h.locate(t.viewTop - h.starts[0])
		anchorLine, anchorCell = h.first+int64(line), seg*h.cols
	}

	last := t.cursorRow
	for i := len(t.content.Rows) - 1; i > last; i-- {
		if len(trimBlanks(t.content.Rows[i].Cells)) > 0 {
			last = i
			break
		}
	}
	cursorLine, cursorCell := t.lineAt(t.cursorRow)
	cursorCell += t.cursorCol
	for i := 0; i <= last; i++ {
		row := widget.TextGridRow{}
		if i < len(t.content.Rows) {
			row = t.content.Rows[i]
		}
		h.push(row, row.Style == wrapMark && i < last)
	}
	h.open = false
	if i := int(cursorLine - h.first); i >= 0 && i < len(h.lines) && h.lines[i].n < cursorCell {
		h.padLine(i, cursorCell) // its trailing blanks went, the cursor stays where it was
	}
	h.setCols(cols)

	cursorRow, cursorCol := 0, 0
	if i := int(cursorLine - h.first); i >= 0 && i < len(h.lines) {
		cursorRow, cursorCol = h.rowOf(i, cursorCell)
	}
	start := max(0, h.rowCount()-rows)
	start = max(min(start, cursorRow), cursorRow-rows+1) // the cursor on the screen
	screen := h.popRows(start)
	if len(screen) > rows {
		screen = screen[:rows]
	}
	t.content.Rows = screen
	t.cursorRow, t.cursorCol = cursorRow-start, cursorCol
	t.layoutCols, t.layoutRows = cols, rows

	if i := int(anchorLine - h.first); anchorLine >= 0 && i >= 0 && i < len(h.lines) {
		p, _ := h.rowOf(i, anchorCell)
		t.viewTop = h.starts[0] + p
	}
}

// lineAt returns the scrollback line a screen row belongs to, by its index in the scrollback
// once the screen went there, and the cell the row starts at in it.
func (t *Terminal) lineAt(row int) (line int64, cell int) {
	h := t.history()
	line = h.next()
	if h.open && len(h.lines) > 0 {
		line--
		cell = h.lines[len(h.lines)-1].n
	}
	for i := 0; i < row; i++ {
		if i < len(t.content.Rows) && t.content.Rows[i].Style == wrapMark {
			cell += len(t.content.Rows[i].Cells)
			continue
		}
		line++
		cell = 0
	}
	return line, cell
}

// Lines reports the numbers of the lines the terminal holds, from first up to, not including,
// end: the scrollback's, then the screen's. A line is what the output wrote between two line
// feeds, one wrapped over several rows is one line. A line keeps its number; the numbers of
// dropped lines are gone. Main goroutine only.
func (t *Terminal) Lines() (first, end int64) {
	if t.content == nil {
		return 1, 1
	}
	h := t.history()
	last, _ := t.lineAt(len(t.content.Rows))
	if n := len(t.content.Rows); n > 0 && t.content.Rows[n-1].Style == wrapMark {
		last++
	}
	return h.first, max(last, h.next())
}

// LineTexts returns the text of the lines numbered from up to, not including, end, without
// trailing spaces. Main goroutine only.
func (t *Terminal) LineTexts(from, end int64) []string {
	if t.content == nil {
		return nil
	}
	h := t.history()
	from = max(from, h.first)
	var screen []string
	screenFirst := h.next()
	if end > screenFirst || (h.open && end > screenFirst-1) {
		screen, screenFirst = t.screenLines()
	}
	texts := make([]string, 0, max(end-from, 0))
	for n := from; n < end; n++ {
		var text string
		if i := int(n - h.first); i < len(h.lines) && n < screenFirst {
			text = h.text(i, 0, h.lines[i].n)
		} else if j := int(n - screenFirst); j >= 0 && j < len(screen) {
			text = screen[j]
		} else {
			break
		}
		texts = append(texts, strings.TrimRight(text, " "))
	}
	return texts
}

// screenLines returns the lines on the screen, joined across wrapped rows, and the number of the
// first; a line the scrollback's last goes on in comes whole.
func (t *Terminal) screenLines() (lines []string, first int64) {
	h := t.history()
	first = h.next()
	var b strings.Builder
	if h.open && len(h.lines) > 0 {
		first--
		last := len(h.lines) - 1
		b.WriteString(h.text(last, 0, h.lines[last].n))
	}
	for i, row := range t.content.Rows {
		for _, c := range row.Cells {
			r := c.Rune
			if r == 0 {
				r = ' '
			}
			b.WriteRune(r)
		}
		if row.Style == wrapMark && i < len(t.content.Rows)-1 {
			continue
		}
		lines = append(lines, b.String())
		b.Reset()
	}
	return lines, first
}

// SetScrollbackLimit sets the most the scrollback holds, in bytes; 256 MiB by default.
func (t *Terminal) SetScrollbackLimit(bytes int) {
	t.history().limit = bytes
	t.hist.trim()
}

// selectedText returns the text from row sr, column sc to row er, column ec, the end included,
// over the scrollback's rows and the screen's: a row per line, except that a line wrapped over
// rows comes as one.
func (t *Terminal) selectedText(sr, sc, er, ec int) string {
	var b strings.Builder
	for row := sr; row <= er; row++ {
		cells, continues := t.rowCells(row)
		from, to := 0, len(cells)-1
		if t.blockMode {
			from, to = sc, ec
		} else {
			if row == sr {
				from = sc
			}
			if row == er {
				to = ec
			}
		}
		for col := max(from, 0); col <= to && col < len(cells); col++ {
			b.WriteRune(cells[col].Rune)
		}
		if row < er && (t.blockMode || !continues) {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// setCell sets a cell of the screen, adding the rows and cells before it; TextGrid.SetCell needs
// a renderer, which the content never gets.
func (t *Terminal) setCell(row, col int, cell widget.TextGridCell) {
	if row < 0 || col < 0 {
		return
	}
	for len(t.content.Rows) <= row {
		t.content.Rows = append(t.content.Rows, widget.TextGridRow{})
	}
	cells := t.content.Rows[row].Cells
	for len(cells) <= col {
		cells = append(cells, widget.TextGridCell{})
	}
	cells[col] = cell
	t.content.Rows[row].Cells = cells
}
