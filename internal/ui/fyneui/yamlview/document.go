package yamlview

import (
	"slices"
	"strings"
	"unicode/utf8"
)

// pos is a place in the document: a line index and a byte offset into that line. Offsets that
// reach the document through clamp sit on a rune boundary.
type pos struct {
	line, off int
}

// before reports whether p comes before q.
func (p pos) before(q pos) bool {
	return p.line < q.line || (p.line == q.line && p.off < q.off)
}

// order returns p and q sorted.
func order(p, q pos) (pos, pos) {
	if q.before(p) {
		return q, p
	}
	return p, q
}

// tabWidth is the distance between tab stops when a line contains a tab. YAML indents with spaces
// only, so a tab appears inside scalars at most.
const tabWidth = 4

// document holds the text as lines. SetText keeps the caller's string and slices it into lines
// without copying, so a freshly loaded document costs one string header per line on top of the
// text itself; an edited line gets its own small allocation. Nothing here is proportional to the
// number of characters beyond the text, unlike widget.TextGrid's cell per character.
type document struct {
	// lines is never empty: an empty document is one empty line.
	lines []string
	// states holds the tokenizer's context at the start of lines[0:len(states)]. It is a prefix
	// computed on demand and cut back to the edited line by every edit, so it grows only as far as
	// the view has scrolled.
	states []lineState
	// extent is an upper bound of the widest line in columns: bytes plus the extra width of tabs.
	// It grows with edits and is exact again after setText.
	extent int
	// text caches the joined document; textOK says whether it is current.
	text   string
	textOK bool
}

func newDocument() document {
	return document{lines: []string{""}, states: []lineState{0}, textOK: true}
}

// setText replaces the document. "\r\n" and a lone "\r" end lines as "\n" does, as in YAML; the
// document returns them as "\n". Text without "\r" is kept as it is, not copied.
func (d *document) setText(s string) {
	s = normaliseNewlines(s)
	lines := make([]string, 0, strings.Count(s, "\n")+1)
	extent := 0
	rest := s
	for {
		i := strings.IndexByte(rest, '\n')
		line := rest
		if i >= 0 {
			line = rest[:i]
		}
		lines = append(lines, line)
		extent = max(extent, lineExtent(line))
		if i < 0 {
			break
		}
		rest = rest[i+1:]
	}
	clear(d.lines) // drop references to the old text before the slice is replaced
	d.lines = lines
	d.states = append(d.states[:0], 0)
	d.extent = extent
	d.text, d.textOK = s, true
}

// lineExtent is an upper bound of the line's width in columns. Every rune is at least as many
// bytes long as it is wide, except a tab.
func lineExtent(line string) int {
	n := len(line)
	if strings.IndexByte(line, '\t') >= 0 {
		n += (tabWidth - 1) * strings.Count(line, "\t")
	}
	return n
}

// joined returns the whole document with "\n" between lines.
func (d *document) joined() string {
	if !d.textOK {
		d.text = strings.Join(d.lines, "\n")
		d.textOK = true
	}
	return d.text
}

// end is the position after the last character.
func (d *document) end() pos {
	last := len(d.lines) - 1
	return pos{last, len(d.lines[last])}
}

// clamp moves p into the document and onto a rune boundary.
func (d *document) clamp(p pos) pos {
	if p.line < 0 {
		return pos{}
	}
	if p.line >= len(d.lines) {
		return d.end()
	}
	line := d.lines[p.line]
	switch {
	case p.off <= 0:
		p.off = 0
	case p.off >= len(line):
		p.off = len(line)
	default:
		p.off = alignOffset(line, p.off)
	}
	return p
}

// alignOffset moves off back to the start of the rune it falls into. Invalid bytes count as runes of
// one byte, as utf8.DecodeRuneInString sees them, so every offset reachable by decoding forwards is
// kept.
func alignOffset(line string, off int) int {
	if off <= 0 || off >= len(line) {
		return min(max(off, 0), len(line))
	}
	for start := off; start >= 0 && start > off-utf8.UTFMax; start-- {
		if utf8.RuneStart(line[start]) {
			if _, size := utf8.DecodeRuneInString(line[start:]); start+size > off {
				return start
			}
			return off
		}
	}
	return off
}

// touch records that line i changed: the joined text and every tokenizer state after it are stale.
func (d *document) touch(i int) {
	d.textOK = false
	d.text = ""
	if len(d.states) > i+1 {
		d.states = d.states[:i+1]
	}
}

// insert puts s at p and returns the position after it. "\r\n" and "\r" in s become "\n".
func (d *document) insert(p pos, s string) pos {
	p = d.clamp(p)
	if s == "" {
		return p
	}
	s = normaliseNewlines(s)
	line := d.lines[p.line]
	head, tail := line[:p.off], line[p.off:]
	d.touch(p.line)
	if strings.IndexByte(s, '\n') < 0 {
		d.lines[p.line] = head + s + tail
		d.extent = max(d.extent, lineExtent(d.lines[p.line]))
		return pos{p.line, p.off + len(s)}
	}
	parts := strings.Split(s, "\n")
	last := len(parts) - 1
	endOff := len(parts[last])
	parts[0] = head + parts[0]
	parts[last] += tail
	for _, part := range parts {
		d.extent = max(d.extent, lineExtent(part))
	}
	d.lines[p.line] = parts[0]
	d.lines = slices.Insert(d.lines, p.line+1, parts[1:]...)
	return pos{p.line + last, endOff}
}

// remove deletes the text between p and q and returns it.
func (d *document) remove(p, q pos) string {
	a, b := order(d.clamp(p), d.clamp(q))
	if a == b {
		return ""
	}
	removed := d.slice(a, b)
	d.touch(a.line)
	d.lines[a.line] = d.lines[a.line][:a.off] + d.lines[b.line][b.off:]
	if b.line > a.line {
		d.lines = slices.Delete(d.lines, a.line+1, b.line+1)
		d.extent = max(d.extent, lineExtent(d.lines[a.line])) // joined lines can be wider
	}
	return removed
}

// slice returns the text between p and q.
func (d *document) slice(p, q pos) string {
	a, b := order(d.clamp(p), d.clamp(q))
	if a.line == b.line {
		return d.lines[a.line][a.off:b.off]
	}
	n := len(d.lines[a.line]) - a.off + b.off + (b.line - a.line)
	for i := a.line + 1; i < b.line; i++ {
		n += len(d.lines[i])
	}
	var sb strings.Builder
	sb.Grow(n)
	sb.WriteString(d.lines[a.line][a.off:])
	for i := a.line + 1; i < b.line; i++ {
		sb.WriteByte('\n')
		sb.WriteString(d.lines[i])
	}
	sb.WriteByte('\n')
	sb.WriteString(d.lines[b.line][:b.off])
	return sb.String()
}

// stateAt returns the tokenizer context at the start of line i, scanning forward from the last
// known state when needed. The scan reads no further than line i.
func (d *document) stateAt(i int) lineState {
	if i <= 0 {
		return 0
	}
	if i >= len(d.lines) {
		i = len(d.lines) - 1
	}
	for len(d.states) <= i {
		n := len(d.states)
		d.states = append(d.states, nextState(d.lines[n-1], d.states[n-1]))
	}
	return d.states[i]
}

// prev returns the position one rune before p, crossing to the end of the previous line.
func (d *document) prev(p pos) pos {
	if p.off > 0 {
		_, size := utf8.DecodeLastRuneInString(d.lines[p.line][:p.off])
		return pos{p.line, p.off - size}
	}
	if p.line > 0 {
		return pos{p.line - 1, len(d.lines[p.line-1])}
	}
	return p
}

// next returns the position one rune after p, crossing to the start of the next line.
func (d *document) next(p pos) pos {
	line := d.lines[p.line]
	if p.off < len(line) {
		_, size := utf8.DecodeRuneInString(line[p.off:])
		return pos{p.line, p.off + size}
	}
	if p.line < len(d.lines)-1 {
		return pos{p.line + 1, 0}
	}
	return p
}

// normaliseNewlines turns "\r\n" and lone "\r" into "\n".
func normaliseNewlines(s string) string {
	if strings.IndexByte(s, '\r') < 0 {
		return s
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\r", "\n")
}

// runeWidth is the number of columns a rune takes: two for East Asian wide and fullwidth runes and
// most emoji, one for everything else. It is a small table rather than a full Unicode width
// database; Kubernetes YAML is almost always ASCII, and a wrong guess only shifts the rest of one
// line.
func runeWidth(r rune) int {
	switch {
	case r < 0x1100:
		return 1
	case r <= 0x115f, // Hangul Jamo
		r >= 0x2e80 && r <= 0xa4cf && r != 0x303f, // CJK radicals to Yi
		r >= 0xac00 && r <= 0xd7a3,                // Hangul syllables
		r >= 0xf900 && r <= 0xfaff,                // CJK compatibility ideographs
		r >= 0xfe30 && r <= 0xfe4f,                // CJK compatibility forms
		r >= 0xff00 && r <= 0xff60,                // fullwidth forms
		r >= 0xffe0 && r <= 0xffe6,
		r >= 0x1f300 && r <= 0x1f64f, // emoji
		r >= 0x1f900 && r <= 0x1f9ff,
		r >= 0x20000 && r <= 0x3fffd: // CJK extensions
		return 2
	}
	return 1
}

// nextTabStop returns the column after a tab at col.
func nextTabStop(col int) int {
	return (col/tabWidth + 1) * tabWidth
}

// displayCol returns the column at which byte offset off of line starts.
func displayCol(line string, off int) int {
	off = min(max(off, 0), len(line))
	col := 0
	for i := 0; i < off; {
		c := line[i]
		switch {
		case c == '\t':
			col = nextTabStop(col)
			i++
		case c < utf8.RuneSelf:
			col++
			i++
		default:
			r, size := utf8.DecodeRuneInString(line[i:])
			col += runeWidth(r)
			i += size
		}
	}
	return col
}

// offsetForCol returns the byte offset of the rune boundary nearest to column col.
func offsetForCol(line string, col int) int {
	if col <= 0 {
		return 0
	}
	c := 0
	for i := 0; i < len(line); {
		var w, size int
		switch b := line[i]; {
		case b == '\t':
			w, size = nextTabStop(c)-c, 1
		case b < utf8.RuneSelf:
			w, size = 1, 1
		default:
			var r rune
			r, size = utf8.DecodeRuneInString(line[i:])
			w = runeWidth(r)
		}
		if c+w > col {
			// col falls inside this rune: pick the nearer edge.
			if col-c > c+w-col {
				return i + size
			}
			return i
		}
		c += w
		i += size
	}
	return len(line)
}
