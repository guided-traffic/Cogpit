package terminal

import (
	"image/color"
	"sort"
	"unicode/utf8"

	"fyne.io/fyne/v2/widget"
)

// Cogpit patch: the scrollback. The lines that scroll off the top of the main screen are kept
// here, compactly: a line's text as UTF-8 and its styles as runs, about a hundred bytes for a
// line of 80 characters where a row of the grid holds a style object per cell. A line is what
// the output wrote between two line feeds; one wrapped over several rows is one line, so it
// wraps again at another width. Lines keep their numbers while held, and the oldest are dropped
// once the lines take more than the limit.

// defaultScrollbackLimit is the most the scrollback of one terminal holds, in bytes.
const defaultScrollbackLimit = 256 << 20

// wrapMark marks a screen row that the output continued on the next row (the right margin's
// automatic wrap); see markWrapped. The content is never drawn, so the mark is never seen.
var wrapMark widget.TextGridStyle = &widget.CustomTextGridStyle{}

// histLine is a line of the scrollback.
type histLine struct {
	text  []byte     // the cells' runes, a cell without one as a space
	runs  []styleRun // the cells' styles, by the cell each one starts at; none for no style
	n     int        // the number of cells
	ascii bool       // every rune is one byte, so a cell's offset in text is its index
}

// styleRun is a style that applies from a cell to the next run.
type styleRun struct {
	from  int
	style widget.TextGridStyle
}

// history is the scrollback. Main goroutine only.
type history struct {
	lines   []histLine
	starts  []int // the row each line starts on at width cols, then the row after the last line
	cols    int
	first   int64 // the number of lines[0]
	bytes   int
	limit   int
	open    bool  // the last line goes on in the screen's first row
	dropped int64 // lines dropped to keep within limit
}

func newHistory(cols int) *history {
	return &history{starts: []int{0}, cols: cols, limit: defaultScrollbackLimit, first: 1}
}

// next is the number the next line will get.
func (h *history) next() int64 { return h.first + int64(len(h.lines)) }

// rowCount is the number of rows the lines take at width cols.
func (h *history) rowCount() int { return h.starts[len(h.starts)-1] - h.starts[0] }

func (h *history) rowsOf(n int) int {
	if h.cols <= 0 || n <= h.cols {
		return 1
	}
	return (n + h.cols - 1) / h.cols
}

// push adds a screen row that scrolled off: to the last line if that one went on in this row,
// as a new line otherwise. A row that does not go on drops its trailing blanks. A line that grew
// to a quarter of the limit (output without line feeds) ends there, so that trim can drop it.
func (h *history) push(row widget.TextGridRow, wrapped bool) {
	cells := row.Cells
	if !wrapped {
		cells = trimBlanks(cells)
	}
	if h.open && len(h.lines) > 0 && h.lines[len(h.lines)-1].size() < h.limit/4 {
		last := &h.lines[len(h.lines)-1]
		h.bytes -= last.size()
		last.appendCells(cells)
		h.bytes += last.size()
		h.starts[len(h.starts)-1] = h.starts[len(h.starts)-2] + h.rowsOf(last.n)
	} else {
		l := histLine{ascii: true}
		l.appendCells(cells)
		h.lines = append(h.lines, l)
		h.starts = append(h.starts, h.starts[len(h.starts)-1]+h.rowsOf(l.n))
		h.bytes += l.size()
	}
	h.open = wrapped
	if !wrapped {
		h.lines[len(h.lines)-1].clip()
	}
	h.trim()
}

// trim drops the oldest lines while the lines take more than the limit, keeping the newest.
func (h *history) trim() {
	if h.bytes <= h.limit || len(h.lines) <= 1 {
		return
	}
	drop := 0
	for h.bytes > h.limit && drop < len(h.lines)-1 {
		h.bytes -= h.lines[drop].size()
		h.lines[drop] = histLine{}
		drop++
	}
	h.lines = h.lines[drop:]
	h.starts = h.starts[drop:]
	h.first += int64(drop)
	h.dropped += int64(drop)
}

// clear drops every line, as erase saved lines (CSI 3 J) asks; the numbers go on.
func (h *history) clear() {
	h.first = h.next()
	h.lines = nil
	h.starts = []int{0}
	h.bytes = 0
	h.open = false
	h.dropped = 0
}

// setCols wraps the lines at another width.
func (h *history) setCols(cols int) {
	if cols == h.cols {
		return
	}
	h.cols = cols
	start := 0
	for i := range h.lines {
		h.starts[i] = start
		start += h.rowsOf(h.lines[i].n)
	}
	h.starts[len(h.lines)] = start
}

// locate returns the line and the line's row that row p of the lines is.
func (h *history) locate(p int) (line, seg int) {
	abs := h.starts[0] + p
	line = sort.Search(len(h.lines), func(i int) bool { return h.starts[i+1] > abs }) // first line ending after abs
	return line, abs - h.starts[line]
}

// row returns the cells of a line's row at width cols.
func (h *history) row(line, seg int) []widget.TextGridCell {
	l := &h.lines[line]
	from := seg * h.cols
	if h.cols <= 0 {
		from = 0
	}
	to := l.n
	if h.cols > 0 {
		to = min(l.n, from+h.cols)
	}
	return l.cells(from, to)
}

// rowOf returns the row of the lines, and its column, that a line's cell is on.
func (h *history) rowOf(line, offset int) (p, col int) {
	p = h.starts[line] - h.starts[0]
	if h.cols <= 0 {
		return p, offset
	}
	n := h.lines[line].n
	if offset > 0 && offset == n && n%h.cols == 0 {
		return p + n/h.cols - 1, h.cols // after the last cell of a full row
	}
	return p + offset/h.cols, offset % h.cols
}

// padLine adds spaces to a line up to n cells.
func (h *history) padLine(line, n int) {
	l := &h.lines[line]
	if l.n >= n {
		return
	}
	h.bytes -= l.size()
	pad := make([]widget.TextGridCell, n-l.n)
	for i := range pad {
		pad[i].Rune = ' '
	}
	l.appendCells(pad)
	h.bytes += l.size()
	h.starts[line+1] = h.starts[line] + h.rowsOf(l.n)
	for i := line + 1; i < len(h.lines); i++ {
		h.starts[i+1] = h.starts[i] + h.rowsOf(h.lines[i].n)
	}
}

// popRows removes the rows of the lines from row p on and returns them as screen rows, each
// row a line goes on from marked; a line cut in two keeps its first part, which then goes on in
// the screen's first row.
func (h *history) popRows(p int) []widget.TextGridRow {
	if p >= h.rowCount() {
		return nil
	}
	line, seg := h.locate(p)
	var rows []widget.TextGridRow
	for i := line; i < len(h.lines); i++ {
		first := 0
		if i == line {
			first = seg
		}
		count := h.rowsOf(h.lines[i].n)
		for s := first; s < count; s++ {
			row := widget.TextGridRow{Cells: h.row(i, s)}
			if s < count-1 {
				row.Style = wrapMark
			}
			rows = append(rows, row)
		}
	}

	for i := line; i < len(h.lines); i++ {
		h.bytes -= h.lines[i].size()
	}
	if seg > 0 {
		h.lines[line].cut(seg * h.cols)
		h.bytes += h.lines[line].size()
		h.lines = h.lines[:line+1]
		h.starts = h.starts[:line+2]
		h.starts[line+1] = h.starts[line] + seg
		h.open = true
	} else {
		for i := line; i < len(h.lines); i++ {
			h.lines[i] = histLine{}
		}
		h.lines = h.lines[:line]
		h.starts = h.starts[:line+1]
		h.open = false
	}
	return rows
}

// text returns a line's text from a cell up to, not including, another.
func (h *history) text(line, from, to int) string {
	l := &h.lines[line]
	to = min(to, l.n)
	if from >= to {
		return ""
	}
	if l.ascii {
		return string(l.text[from:to])
	}
	start, end := l.byteOffset(from), l.byteOffset(to)
	return string(l.text[start:end])
}

func (l *histLine) size() int {
	return cap(l.text) + 32*cap(l.runs) + 96 // the line, its runs, and its share of the slices
}

func (l *histLine) appendCells(cells []widget.TextGridCell) {
	for _, c := range cells {
		r := c.Rune
		if r == 0 {
			r = ' '
		}
		if r >= utf8.RuneSelf {
			l.ascii = false
		}
		l.text = utf8.AppendRune(l.text, r)
		if !sameStyle(l.styleAtEnd(), c.Style) {
			l.runs = append(l.runs, styleRun{from: l.n, style: c.Style})
		}
		l.n++
	}
}

func (l *histLine) styleAtEnd() widget.TextGridStyle {
	if len(l.runs) == 0 {
		return nil
	}
	return l.runs[len(l.runs)-1].style
}

// clip drops the spare capacity a growing line may hold.
func (l *histLine) clip() {
	if cap(l.text) > len(l.text)+len(l.text)/8+16 {
		l.text = append([]byte(nil), l.text...)
	}
	if cap(l.runs) > len(l.runs) {
		l.runs = append([]styleRun(nil), l.runs...)
	}
}

// cut keeps the first n cells.
func (l *histLine) cut(n int) {
	if n >= l.n {
		return
	}
	end := n
	if !l.ascii {
		end = l.byteOffset(n)
	}
	l.text = append([]byte(nil), l.text[:end]...)
	keep := sort.Search(len(l.runs), func(i int) bool { return l.runs[i].from >= n })
	l.runs = append([]styleRun(nil), l.runs[:keep]...)
	l.n = n
}

func (l *histLine) byteOffset(cell int) int {
	if l.ascii {
		return cell
	}
	off := 0
	for i := 0; i < cell && off < len(l.text); i++ {
		_, size := utf8.DecodeRune(l.text[off:])
		off += size
	}
	return off
}

// cells returns the cells from one up to, not including, another.
func (l *histLine) cells(from, to int) []widget.TextGridCell {
	if from >= to {
		return nil
	}
	cells := make([]widget.TextGridCell, 0, to-from)
	run := sort.Search(len(l.runs), func(i int) bool { return l.runs[i].from > from }) - 1
	off := l.byteOffset(from)
	for i := from; i < to; i++ {
		for run+1 < len(l.runs) && l.runs[run+1].from <= i {
			run++
		}
		var style widget.TextGridStyle
		if run >= 0 {
			style = l.runs[run].style
		}
		r, size := rune(l.text[off]), 1
		if !l.ascii {
			r, size = utf8.DecodeRune(l.text[off:])
		}
		off += size
		cells = append(cells, widget.TextGridCell{Rune: r, Style: style})
	}
	return cells
}

// trimBlanks drops the trailing cells that show nothing: spaces without a background.
func trimBlanks(cells []widget.TextGridCell) []widget.TextGridCell {
	end := len(cells)
	for end > 0 {
		c := cells[end-1]
		if (c.Rune != ' ' && c.Rune != 0) || (c.Style != nil && c.Style.BackgroundColor() != nil) {
			break
		}
		end--
	}
	return cells[:end]
}

// sameStyle reports whether two cell styles look the same, so that one run can hold both.
func sameStyle(a, b widget.TextGridStyle) bool {
	if a == b {
		return true
	}
	ca, okA := a.(*widget.CustomTextGridStyle)
	cb, okB := b.(*widget.CustomTextGridStyle)
	if !okA || !okB || ca == nil || cb == nil {
		return false
	}
	return ca.TextStyle == cb.TextStyle && sameColor(ca.FGColor, cb.FGColor) && sameColor(ca.BGColor, cb.BGColor)
}

func sameColor(a, b color.Color) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}
