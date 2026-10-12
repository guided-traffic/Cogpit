package termview

import (
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var olderT0 = time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)

// stampedLines are "<prefix><timestamp> line <i>" for i in [from, to), one a second.
func stampedLines(prefix string, from, to int) []string {
	var out []string
	for i := from; i < to; i++ {
		out = append(out, prefix+olderT0.Add(time.Duration(i)*time.Second).Format(time.RFC3339Nano)+fmt.Sprintf(" line %d", i))
	}
	return out
}

// streamLines shows a view of v whose stream the test writes to; it waits for want lines.
func streamLines(t *testing.T, v LogView, size fyne.Size, lines []string) (*Logs, func([]string)) {
	t.Helper()
	pr, pw := io.Pipe()
	l, _ := showLogs(t, pr, size)
	l.SetView(v)
	waitApplied := appliedWaiter(t, l)
	total := 0
	write := func(lines []string) {
		go func() { _, _ = io.WriteString(pw, strings.Join(lines, "\n")+"\n") }()
		total += len(lines)
		waitApplied(total)
	}
	t.Cleanup(func() { _ = pw.Close() })
	write(lines)
	return l, write
}

// TestLogsPrependOlderLines: the view says what it holds, takes older lines on top without the
// ones it holds (at its oldest timestamp too), keeps the rows in view in view, and counts them.
func TestLogsPrependOlderLines(t *testing.T) {
	l, _ := streamLines(t, LogView{Stamped: true}, fyne.NewSize(600, 300), stampedLines("", 100, 200))
	held := l.Held()
	if len(held) != 1 || held[0].Source != "" || held[0].Lines != 100 || !held[0].Oldest.Equal(olderT0.Add(100*time.Second)) {
		t.Fatalf("held %+v", held)
	}
	l.SetFollow(false)
	l.list.ScrollToTop()
	before := l.list.GetScrollOffset()
	if n := l.Prepend(stampedLines("", 80, 101)); n != 20 {
		t.Fatalf("prepended %d lines, want the 20 older ones", n)
	}
	rows := rowTexts(l)
	if len(rows) != 120 || rows[0] != "line 80" || rows[20] != "line 100" || rows[21] != "line 101" {
		t.Errorf("rows %q … %q", rows[:2], rows[19:22])
	}
	if got := l.list.GetScrollOffset(); got < before+19*l.stride() {
		t.Errorf("offset %v after prepending 20 rows at %v: the rows in view moved", got, before)
	}
	if st := l.StatusText(); !strings.HasPrefix(st, "120 lines") || strings.Contains(st, "dropped") {
		t.Errorf("status %q", st)
	}
	if n := l.Prepend(stampedLines("", 90, 100)); n != 0 {
		t.Errorf("lines it holds were prepended again: %d", n)
	}
	// The same timestamp twice: the one held stays single, the other goes in.
	at := olderT0.Add(80 * time.Second).Format(time.RFC3339Nano)
	if n := l.Prepend([]string{at + " line 80", at + " another at 80"}); n != 1 || rowTexts(l)[0] != "another at 80" {
		t.Errorf("prepended %d at the oldest timestamp, first row %q", n, rowTexts(l)[0])
	}
}

// TestLogsPrependBoundedAndFenced: the view takes only as many older lines as it keeps, the
// newest of them; lines of a source it held nothing of go in, and the stream's own copy of them
// is dropped until that source's lines are newer.
func TestLogsPrependBoundedAndFenced(t *testing.T) {
	pr, pw := io.Pipe()
	l, _ := showLogs(t, pr, fyne.NewSize(600, 300))
	l.lines = newLineRing(30, logMaxBytes) // before any line arrives
	l.SetView(LogView{Prefixed: true, Names: true, Stamped: true})
	waitApplied := appliedWaiter(t, l)
	go func() { _, _ = io.WriteString(pw, strings.Join(stampedLines("a/x ", 50, 70), "\n")+"\n") }()
	waitApplied(20)
	t.Cleanup(func() { _ = pw.Close() })
	if l.Room() != 10 || l.Capacity() != 30 {
		t.Fatalf("room %d of %d", l.Room(), l.Capacity())
	}
	older := append(stampedLines("a/x ", 30, 50), stampedLines("b/y ", 40, 45)...)
	if n := l.Prepend(older); n != 10 {
		t.Fatalf("prepended %d, want the room's 10", n)
	}
	// The newest 10 of a/x's 20 older and b/y's 5, in the order given: a/x 45–49 and b/y 40–44.
	rows := rowTexts(l)
	if !strings.HasPrefix(rows[0], "a/x ") || !strings.HasSuffix(rows[0], " line 45") || !strings.HasSuffix(rows[9], " line 44") {
		t.Errorf("rows %q", rows[:10])
	}
	if len(l.fences) != 1 || l.fences["b/y"] == nil {
		t.Fatalf("fences %v", l.fences)
	}
	// b/y's stream brings its history (40–44, held now) and a newer line.
	go func() {
		_, _ = io.WriteString(pw, strings.Join(stampedLines("b/y ", 40, 46), "\n")+"\n")
	}()
	waitApplied(26)
	count := map[string]int{}
	for _, r := range rowTexts(l) {
		count[r]++
	}
	for r, n := range count {
		if n > 1 {
			t.Errorf("%q shows %d times", r, n)
		}
	}
	if !strings.HasSuffix(rowTexts(l)[len(rowTexts(l))-1], " line 45") || len(l.fences) != 0 {
		t.Errorf("last row %q, fences %v", rowTexts(l)[len(rowTexts(l))-1], l.fences)
	}
}

// TestLogsHeadRow: the row above the lines shows in its tone, is the list's first item, and
// selecting it acts (and does not stay selected); without text there is none.
func TestLogsHeadRow(t *testing.T) {
	l, _ := streamLines(t, LogView{Stamped: true}, fyne.NewSize(600, 300), stampedLines("", 0, 5))
	if l.list.Length() != 5 || l.Head() != "" {
		t.Fatalf("length %d without a head", l.list.Length())
	}
	tapped := 0
	l.SetHead("Load older lines", HeadAction, func() { tapped++ })
	if l.list.Length() != 6 {
		t.Errorf("length %d with a head", l.list.Length())
	}
	if runs := l.itemRuns(0); len(runs) != 1 || runs[0].text != "Load older lines" || runs[0].color != theme.ColorNamePrimary {
		t.Errorf("head runs %+v", runs)
	}
	if runs := l.itemRuns(1); runs[0].text != "line 0" {
		t.Errorf("first row %+v", runs)
	}
	l.list.Select(0)
	l.list.Select(2)
	if tapped != 1 {
		t.Errorf("the head acted %d times", tapped)
	}
	l.SetHead("Beginning of the log", HeadInfo, nil)
	l.list.Select(0)
	if runs := l.itemRuns(0); runs[0].color != theme.ColorNamePlaceHolder || tapped != 1 {
		t.Errorf("info head %+v, acted %d", runs, tapped)
	}
	l.SetHead("", HeadInfo, nil)
	if l.list.Length() != 5 {
		t.Errorf("length %d after the head went", l.list.Length())
	}
}
