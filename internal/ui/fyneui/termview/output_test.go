package termview

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"github.com/guided-traffic/cogpit/internal/engine"
)

// outputLines returns every line of an output, in order.
func outputLines(o interface {
	Len() int
	Text(int) string
}) []string {
	out := make([]string, 0, o.Len())
	for i := range o.Len() {
		out = append(out, o.Text(i))
	}
	return out
}

// TestLogOutputIsWhatTheViewShows: the output is the lines the view shows, as it shows them: the
// filter and the selected pod applied, names and timestamps as toggled, a wrapped line once, numbered
// ascending by its place in the stream, so that a page of them can be found again.
func TestLogOutputIsWhatTheViewShows(t *testing.T) {
	l, _ := showMerged(t, mergedLogs, LogView{Prefixed: true, Stamped: true}, fyne.NewSize(900, 400))

	all := l.Output()
	if all.Len() != len(mergedLogs) || all.Text(0) != "starting order-service version=1.42.0" {
		t.Fatalf("%d lines, the first %q", all.Len(), all.Text(0))
	}
	for i := 1; i < all.Len(); i++ {
		if all.Number(i) <= all.Number(i-1) {
			t.Fatalf("numbers not ascending: %d after %d", all.Number(i), all.Number(i-1))
		}
	}
	if l.HeldLines() != len(mergedLogs) || l.DroppedLines() != 0 {
		t.Errorf("held %d, dropped %d", l.HeldLines(), l.DroppedLines())
	}

	l.SetView(LogView{Prefixed: true, Stamped: true, Names: true, Timestamps: true, Search: "payment", Filter: true, Wrap: true})
	got := outputLines(l.Output())
	want := []string{
		"web-7d9f-b2/app " + localStamps("2026-10-11T12:00:12.774310925Z") + " ERROR payment provider unreachable: dial tcp 10.96.14.2:443: i/o timeout",
		"web-7d9f-b2/app " + localStamps("2026-10-11T12:00:14.801233417Z") + " payment provider reachable again after 2.03s; queued 4 orders for capture and released the connection pool's idle sockets",
	}
	if !slices.Equal(got, want) {
		t.Errorf("filtered, with names and timestamps, wrapped:\n got %q\nwant %q", got, want)
	}

	l.SetView(LogView{Prefixed: true, Stamped: true, Pod: "web-7d9f-a1"})
	got = outputLines(l.Output())
	// The pod's four lines (two containers), and Cogpit's own note, which the view shows whatever it narrows to.
	if len(got) != 5 || slices.ContainsFunc(got, func(s string) bool { return strings.Contains(s, "attempt=1") }) {
		t.Errorf("one pod: %q", got)
	}
}

// TestScreenLinesHaveNoEscapes: a terminal's lines as text has the characters of its cells, no
// colours, no escape sequences, no control characters, no trailing blanks.
func TestScreenLinesHaveNoEscapes(t *testing.T) {
	sess := newFakeSession()
	term, _ := showTerminal(t, sess, fyne.NewSize(640, 320))
	if got := outputLines(term.Output()); len(got) != 0 {
		t.Errorf("an empty screen: %q", got)
	}
	sess.feed(t, "$ \x1b[1;31mkubectl\x1b[0m get pods\r\n\x1b]0;title\x07NAME   READY\r\nweb-0  1/1\x1b[K\r\n\x1b[5Cmoved\r\n$ ")
	got := outputLines(term.Output())
	want := []string{"$ kubectl get pods", "NAME   READY", "web-0  1/1", "     moved", "$"}
	if !slices.Equal(got, want) {
		t.Errorf("screen lines\n got %q\nwant %q", got, want)
	}
	for _, line := range got {
		if strings.ContainsFunc(line, func(r rune) bool { return r < 0x20 || r == 0x7f }) {
			t.Errorf("a control character in %q", line)
		}
	}
	out := term.Output()
	if out.Len() != 5 || out.Number(0) != 1 || out.Number(4) != 5 || out.Text(1) != "NAME   READY" {
		t.Errorf("numbered output: %d lines, %d..%d", out.Len(), out.Number(0), out.Number(4))
	}
	sess.end()
	waitDone(t, term.Done())
}

// TestOutputNumbersStayWithTheirLines: a terminal's lines come from its scrollback and its screen,
// numbered from its first line, and a line keeps its number as later output scrolls it up, so an
// agent can ask for the lines above one it read (before: n, limit: 2).
func TestOutputNumbersStayWithTheirLines(t *testing.T) {
	sess := newFakeSession()
	term, _ := showTerminal(t, sess, fyne.NewSize(640, 320))
	var b strings.Builder
	for i := 1; i <= 300; i++ {
		fmt.Fprintf(&b, "line %d\r\n", i)
	}
	sess.feed(t, b.String())
	out := term.Output()
	if out.Len() != 300 || out.Number(0) != 1 || out.Text(0) != "line 1" || out.Text(299) != "line 300" {
		t.Fatalf("%d lines from %d: %q .. %q", out.Len(), out.Number(0), out.Text(0), out.Text(out.Len()-1))
	}

	sess.feed(t, "line 301\r\nline 302\r\n")
	out = term.Output()
	before := int64(300)
	page := engine.PageDockLines(out, engine.DockOutputQuery{Before: &before, Limit: 2})
	if !slices.Equal(page.Lines, []string{"line 298", "line 299"}) || page.First != 298 {
		t.Errorf("the two lines above line 300: %q from %d", page.Lines, page.First)
	}
	sess.end()
	waitDone(t, term.Done())
}
