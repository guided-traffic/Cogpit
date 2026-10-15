package termview

import (
	"strings"
	"unicode"

	fyneterm "github.com/fyne-io/terminal"
)

// What the stream widgets show, as text for a reader outside them: an agent's dock.read (G17,
// D-44; engine/dockoutput.go pages it). Both are read on the main goroutine, like the widgets.

// LogOutput is what a log view shows, line by line, oldest first: the lines its rows show (the
// sources it narrows to and its filter applied), each as shown (resource names and timestamps as
// the view shows them, one line unwrapped), numbered by their place in the stream, which stays
// while the view holds them (numbers go below zero for older lines loaded later). It indexes the
// view's lines at the moment it is made: use it on the main goroutine before the view changes.
type LogOutput struct {
	l    *Logs
	nums []int64
}

// Output returns what the view shows now. Main goroutine only.
func (l *Logs) Output() LogOutput {
	out := LogOutput{l: l, nums: make([]int64, 0, l.rowLines)}
	for i, r := range l.rows {
		if i == 0 || r.line != l.rows[i-1].line { // a wrapped line has several rows
			out.nums = append(out.nums, r.line)
		}
	}
	return out
}

// Len is the number of lines the view shows.
func (o LogOutput) Len() int { return len(o.nums) }

// Number is line i's number in the stream.
func (o LogOutput) Number(i int) int64 { return o.nums[i] }

// Text is line i as the view shows it.
func (o LogOutput) Text(i int) string {
	text, _ := o.l.view.shown(o.l.lines.at(int(o.nums[i] - o.l.lines.evicted)))
	return text
}

// HeldLines is the number of lines the view holds, before its filter. Main goroutine only.
func (l *Logs) HeldLines() int { return l.lines.len() }

// DroppedLines is the number of older lines the view no longer holds (its scrollback's bound), as
// its status counts them. Main goroutine only.
func (l *Logs) DroppedLines() int64 {
	return max(l.stReceived+l.prepended-int64(l.lines.len()), 0)
}

// Ended reports whether the stream ended, as the status shows it. Main goroutine only.
func (l *Logs) Ended() bool { return l.stEnded }

// Output returns what the terminal holds as a numbered list for a reader (D-44): its scrollback's
// lines, then its screen's, without the empty lines below the last one with text. A line is what
// the output wrote between two line feeds, one wrapped over several rows is one line, as text: the
// characters of its cells without colours or escape sequences (the widget holds none), a cell
// never written as a space, any other control character left out, no trailing spaces. Lines are
// numbered from the first the terminal held and keep their numbers when they scroll into the
// scrollback; the scrollback drops its oldest lines past its bound (256 MiB). Main goroutine only.
func (t *Terminal) Output() Output {
	first, end := t.term.Lines()
	for end > first && (Output{term: t.term}).text(end-1) == "" {
		end--
	}
	return Output{term: t.term, first: first, n: int(end - first)}
}

// screenRune is a cell's character as text: a cell never written (0) is a space, other control
// characters are left out.
func screenRune(r rune) rune {
	switch {
	case r == 0:
		return ' '
	case r == '\t':
		return r
	case unicode.IsControl(r):
		return -1
	}
	return r
}

// Output is a terminal's lines, read from the widget when asked for one (Terminal.Output).
type Output struct {
	term  *fyneterm.Terminal
	first int64
	n     int
}

// Len is the number of lines.
func (o Output) Len() int { return o.n }

// Number is line i's number.
func (o Output) Number(i int) int64 { return o.first + int64(i) }

// Text is line i.
func (o Output) Text(i int) string { return o.text(o.Number(i)) }

// Dropped is the number of older lines the terminal no longer holds.
func (o Output) Dropped() int64 { return max(o.first-1, 0) }

func (o Output) text(number int64) string {
	texts := o.term.LineTexts(number, number+1)
	if len(texts) == 0 {
		return ""
	}
	return strings.TrimRight(strings.Map(screenRune, texts[0]), " ")
}
