package termview

import "time"

// JumpTo scrolls the view to the first line written at or after t among the rows it shows: the
// line with the earliest such timestamp, the first row of equal ones, at the top of the view and
// selected. The view stops following. It reports whether there is such a line; without one the
// view goes to its newest line. Main goroutine only.
func (l *Logs) JumpTo(t time.Time) bool {
	best, bestAt := -1, time.Time{}
	for i, r := range l.rows {
		if i > 0 && r.line == l.rows[i-1].line {
			continue // a wrapped line's other rows (line numbers go below zero once lines are prepended)
		}
		_, ts, ok := l.view.source(l.lines.at(int(r.line - l.lines.evicted)))
		if !ok || ts.Before(t) {
			continue
		}
		if best < 0 || ts.Before(bestAt) {
			best, bestAt = i, ts
		}
	}
	l.SetFollow(false)
	if best < 0 {
		l.hasJumped = false
		l.list.UnselectAll()
		l.list.ScrollToBottom()
		return false
	}
	l.jumped, l.hasJumped = l.rows[best].line, true
	id := best + l.headRows()
	l.list.Select(id)
	l.list.ScrollToOffset(float32(id) * l.stride())
	return true
}

// JumpedLine returns the line the last jump found, as the view shows it, or "" when it found
// none or the line is gone. Main goroutine only.
func (l *Logs) JumpedLine() string {
	i := int(l.jumped - l.lines.evicted)
	if !l.hasJumped || i < 0 || i >= l.lines.len() {
		return ""
	}
	text, _ := l.view.shown(l.lines.at(i))
	return text
}
