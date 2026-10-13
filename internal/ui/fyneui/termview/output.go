package termview

import (
	"strings"
	"unicode"
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

// ScreenLines returns the terminal's screen as lines of text, top to bottom: the characters of its
// cells without colours or escape sequences (the widget holds none), a cell never written as a
// space, any other control character left out, each line without its trailing spaces, and without
// the empty lines below the last one with text. The widget keeps no scrollback: what scrolled off
// the screen is gone. Main goroutine only.
func (t *Terminal) ScreenLines() []string {
	text := t.term.Text()
	if text == "" {
		return []string{}
	}
	rows := strings.Split(text, "\n")
	for i, row := range rows {
		rows[i] = strings.TrimRight(strings.Map(screenRune, row), " ")
	}
	end := len(rows)
	for end > 0 && rows[end-1] == "" {
		end--
	}
	return rows[:end]
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

// ScreenOutput is a terminal's screen lines as a numbered list for a reader (rows from 1).
type ScreenOutput []string

// Len is the number of lines.
func (s ScreenOutput) Len() int { return len(s) }

// Number is line i's row on the screen, from 1.
func (s ScreenOutput) Number(i int) int64 { return int64(i + 1) }

// Text is line i.
func (s ScreenOutput) Text(i int) string { return s[i] }
