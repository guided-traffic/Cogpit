package engine

import (
	"slices"
	"strconv"
	"strings"
	"testing"
)

// testLines are lines numbered as a logs tab numbers them: ascending, with gaps where a filter hides
// lines.
type testLines struct {
	nums []int64
	text []string
}

func (l testLines) Len() int           { return len(l.text) }
func (l testLines) Number(i int) int64 { return l.nums[i] }
func (l testLines) Text(i int) string  { return l.text[i] }
func ptr64(n int64) *int64             { return &n }
func page(src DockLines, q DockOutputQuery) DockPage {
	q, err := CheckDockQuery(q)
	if err != nil {
		panic(err)
	}
	return PageDockLines(src, q)
}

// numbered returns n lines numbered from first, every step-th number (a filter's gaps).
func numbered(first int64, n, step int) testLines {
	var l testLines
	for i := range n {
		num := first + int64(i*step)
		l.nums = append(l.nums, num)
		l.text = append(l.text, "line "+strconv.FormatInt(num, 10))
	}
	return l
}

// TestPageDockLines: the newest lines by default, after and before page forward and back across a
// filter's gaps, more says whether lines lie beyond, the limit is bounded.
func TestPageDockLines(t *testing.T) {
	src := numbered(-4, 10, 3) // -4, -1, 2, ..., 23: lines prepended before 0 and a filter's gaps
	for _, tt := range []struct {
		name        string
		q           DockOutputQuery
		first, last int64
		n           int
		more        bool
	}{
		{"newest", DockOutputQuery{Limit: 3}, 17, 23, 3, true},
		{"all", DockOutputQuery{}, -4, 23, 10, false},
		{"after", DockOutputQuery{After: ptr64(2), Limit: 3}, 5, 11, 3, true},
		{"after a gap", DockOutputQuery{After: ptr64(3), Limit: 100}, 5, 23, 7, false},
		{"after the last", DockOutputQuery{After: ptr64(23)}, 0, 0, 0, false},
		{"before", DockOutputQuery{Before: ptr64(5), Limit: 2}, -1, 2, 2, true},
		{"before the first", DockOutputQuery{Before: ptr64(-4)}, 0, 0, 0, false},
		{"before a gap", DockOutputQuery{Before: ptr64(4), Limit: 100}, -4, 2, 3, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := page(src, tt.q)
			if len(p.Lines) != tt.n || p.First != tt.first || p.Last != tt.last || p.More != tt.more {
				t.Fatalf("lines %q first %d last %d more %v; want %d lines %d..%d more %v", p.Lines, p.First, p.Last, p.More, tt.n, tt.first, tt.last, tt.more)
			}
			if tt.n > 0 && (p.Lines[0] != "line "+strconv.FormatInt(tt.first, 10) || p.Lines[tt.n-1] != "line "+strconv.FormatInt(tt.last, 10)) {
				t.Errorf("lines %q are not %d..%d in order", p.Lines, tt.first, tt.last)
			}
		})
	}
	if p := page(numbered(1, 3000, 1), DockOutputQuery{Limit: 5000}); len(p.Lines) != MaxDockLines || p.Last != 3000 || !p.More {
		t.Errorf("limit 5000: %d lines up to %d, more %v; want the newest %d", len(p.Lines), p.Last, p.More, MaxDockLines)
	}
	if _, err := CheckDockQuery(DockOutputQuery{After: ptr64(1), Before: ptr64(2)}); err == nil {
		t.Error("after and before together accepted")
	}
	if _, err := CheckDockQuery(DockOutputQuery{Limit: -1}); err == nil {
		t.Error("a negative limit accepted")
	}
}

// TestPageDockLinesBounds: a line longer than MaxDockLineBytes is cut on a rune's start and counted;
// a page stops before MaxDockPageBytes.
func TestPageDockLinesBounds(t *testing.T) {
	long := strings.Repeat("é", MaxDockLineBytes) // two bytes a rune
	src := testLines{nums: []int64{1, 2}, text: []string{"short", long}}
	p := page(src, DockOutputQuery{})
	if p.Cut != 1 || len(p.Lines[1]) > MaxDockLineBytes || !strings.HasSuffix(p.Lines[1], DockLineCut) ||
		!strings.HasPrefix(strings.TrimSuffix(p.Lines[1], DockLineCut), "éé") || strings.ContainsRune(p.Lines[1], '�') {
		t.Errorf("cut %d, line of %d bytes ending %q", p.Cut, len(p.Lines[1]), p.Lines[1][len(p.Lines[1])-8:])
	}
	big := numbered(1, 100, 1)
	for i := range big.text {
		big.text[i] = strings.Repeat("x", 8<<10)
	}
	p = page(big, DockOutputQuery{Limit: 100})
	if total := len(p.Lines) * (8 << 10); total > MaxDockPageBytes || len(p.Lines) != MaxDockPageBytes/(8<<10) || !p.More || p.Last != 100 {
		t.Errorf("%d lines of 8 KiB (%d bytes), more %v, last %d", len(p.Lines), total, p.More, p.Last)
	}
	p = page(big, DockOutputQuery{After: ptr64(0), Limit: 100})
	if len(p.Lines) != MaxDockPageBytes/(8<<10) || p.First != 1 || !p.More {
		t.Errorf("after: %d lines from %d, more %v", len(p.Lines), p.First, p.More)
	}
}

func TestIsTerminalTab(t *testing.T) {
	var got []DockType
	for _, d := range []DockType{DockLogs, DockTerminal, DockShell, DockAttach, DockEdit, DockCreate, DockHelmUpgrade,
		DockHelmInstall, DockAudit, DockDiagnostics, DockNodeShell} {
		if IsTerminalTab(d) {
			got = append(got, d)
		}
	}
	if want := []DockType{DockTerminal, DockShell, DockAttach, DockNodeShell}; !slices.Equal(got, want) {
		t.Errorf("terminal tabs %v, want %v", got, want)
	}
}
