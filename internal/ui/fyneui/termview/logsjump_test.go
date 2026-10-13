package termview

import (
	"io"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
)

// TestLogsJumpTo: a jump finds the line with the earliest timestamp at or after the time among
// the rows the view shows, whatever their order (a merged stream's are in arrival order), selects
// it and stops following; a time after every line finds none; a filter's hidden lines are not
// found; prepended lines (numbered below zero) are found too.
func TestLogsJumpTo(t *testing.T) {
	l, _ := showMerged(t, mergedLogs, LogView{Prefixed: true, Names: true, Stamped: true, Timestamps: true}, fyne.NewSize(900, 300))
	at := func(s string) time.Time {
		ts, err := time.Parse(time.RFC3339Nano, "2026-10-11T"+s+"Z")
		if err != nil {
			t.Fatal(err)
		}
		return ts
	}
	if !l.JumpTo(at("12:00:12")) || !strings.Contains(l.JumpedLine(), "ERROR payment provider unreachable") || l.Following() {
		t.Errorf("jumped to %q, following %v", l.JumpedLine(), l.Following())
	}
	if !l.JumpTo(at("12:00:00")) || !strings.Contains(l.JumpedLine(), "12:00:01.204 starting") {
		t.Errorf("before every line: %q", l.JumpedLine())
	}
	if l.JumpTo(at("12:01:00")) || l.JumpedLine() != "" {
		t.Errorf("after every line: %q", l.JumpedLine())
	}
	l.SetView(LogView{Prefixed: true, Names: true, Stamped: true, Search: "budget", Filter: true})
	if !l.JumpTo(at("12:00:12")) || !strings.Contains(l.JumpedLine(), "error budget burn rate") {
		t.Errorf("filtered: %q", l.JumpedLine())
	}
	l.SetView(LogView{Prefixed: true, Names: true, Stamped: true, Wrap: true})
	older := "web-7d9f-a1/app 2026-10-11T11:59:00Z " + strings.Repeat("a long older line ", 20)
	if n := l.Prepend([]string{older}); n != 1 {
		t.Fatalf("prepended %d", n)
	}
	if !l.JumpTo(at("11:58:00")) || !strings.HasPrefix(l.JumpedLine(), "web-7d9f-a1/app a long older line") {
		t.Errorf("a prepended, wrapped line: %q", l.JumpedLine())
	}
}

// TestLogsEvictPrependedLines: lines prepended below line number zero are counted out right when
// the stream pushes them out.
func TestLogsEvictPrependedLines(t *testing.T) {
	pr, pw := io.Pipe()
	l, _ := showLogs(t, pr, fyne.NewSize(600, 300))
	l.lines = newLineRing(6, logMaxBytes)
	l.SetView(LogView{Stamped: true, Search: "line", Filter: true})
	waitApplied := appliedWaiter(t, l)
	go func() { _, _ = io.WriteString(pw, strings.Join(stampedLines("", 10, 15), "\n")+"\n") }()
	waitApplied(5)
	t.Cleanup(func() { _ = pw.Close() })
	if n := l.Prepend(stampedLines("", 9, 10)); n != 1 || l.rowLines != 6 || l.lines.evicted != -1 {
		t.Fatalf("prepended %d, %d lines shown, evicted %d", n, l.rowLines, l.lines.evicted)
	}
	go func() { _, _ = io.WriteString(pw, strings.Join(stampedLines("", 15, 17), "\n")+"\n") }()
	waitApplied(7) // pushes out line -1, then line 0
	if l.rowLines != 6 || len(l.rows) != 6 || rowTexts(l)[0] != "line 11" {
		t.Errorf("%d lines in %d rows from %q after two pushed out", l.rowLines, len(l.rows), rowTexts(l)[0])
	}
	if st := l.StatusText(); !strings.HasPrefix(st, "6 of 6 lines (2 older dropped)") {
		t.Errorf("status %q", st)
	}
}
