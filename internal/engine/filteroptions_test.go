package engine

import (
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

func rowOf(cells ...string) model.Row {
	r := model.Row{Cells: make([]model.Cell, len(cells))}
	for i, c := range cells {
		r.Cells[i] = model.Cell{Text: c}
	}
	return r
}

// TestRowMatcher: the search's four modes on a few rows. Terms must each occur in a cell (any
// cell), ignoring case unless match case is on; a regular expression must match in one cell,
// ignoring case unless match case is on, and it is taken as typed.
func TestRowMatcher(t *testing.T) {
	web := rowOf("Web-0", "shop", "Running")
	api := rowOf("api-7d9f", "shop", "CrashLoopBackOff")
	tests := []struct {
		name   string
		text   string
		opts   FilterOptions
		want   []bool // web, api
		errSub string
	}{
		{"empty", "", FilterOptions{}, []bool{true, true}, ""},
		{"empty regex", "", FilterOptions{Regex: true}, []bool{true, true}, ""},
		{"terms ignore case", "web RUNNING", FilterOptions{}, []bool{true, false}, ""},
		{"terms in different cells", "shop crash", FilterOptions{}, []bool{false, true}, ""},
		{"match case keeps case", "web", FilterOptions{MatchCase: true}, []bool{false, false}, ""},
		{"match case found", "Web shop", FilterOptions{MatchCase: true}, []bool{true, false}, ""},
		{"regex ignores case", "^WEB-\\d$", FilterOptions{Regex: true}, []bool{true, false}, ""},
		{"regex with match case", "^web-", FilterOptions{Regex: true, MatchCase: true}, []bool{false, false}, ""},
		{"regex alternation", "^(web|api)-", FilterOptions{Regex: true}, []bool{true, true}, ""},
		{"regex is one pattern, not terms", "web shop", FilterOptions{Regex: true}, []bool{false, false}, ""},
		{"regex within one cell", "Running|Crash", FilterOptions{Regex: true}, []bool{true, true}, ""},
		{"regex spaces kept", "^shop $", FilterOptions{Regex: true}, []bool{false, false}, ""},
		{"term text is not a pattern", "web-.", FilterOptions{}, []bool{false, false}, ""},
		{"invalid regex filters nothing out", "web(", FilterOptions{Regex: true}, []bool{true, true}, "missing closing ) in `web(`"},
		{"invalid regex quotes the text as typed", "[z-a]", FilterOptions{Regex: true}, []bool{true, true}, "invalid character class range in `z-a`"},
		{"invalid text without regex is a term", "web(", FilterOptions{}, []bool{false, false}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewRowMatcher(tt.text, tt.opts)
			switch {
			case tt.errSub == "" && err != nil:
				t.Fatalf("error %v", err)
			case tt.errSub != "" && (err == nil || !strings.Contains(err.Error(), tt.errSub)):
				t.Fatalf("error %v, want one with %q", err, tt.errSub)
			}
			if got := FilterError(tt.text, tt.opts); (got != "") != (tt.errSub != "") || strings.Contains(got, "(?i)") {
				t.Errorf("FilterError %q", got)
			}
			for i, row := range []model.Row{web, api} {
				if got := m.Match(row); got != tt.want[i] {
					t.Errorf("row %q: %v, want %v", row.Cells[0].Text, got, tt.want[i])
				}
				if got := MatchesFilter(row, tt.text, tt.opts); got != tt.want[i] {
					t.Errorf("MatchesFilter row %q: %v, want %v", row.Cells[0].Text, got, tt.want[i])
				}
			}
		})
	}
}

// TestRowMatcherBounds: a pattern longer than MaxFilterPattern is refused without compiling, and
// one whose program would be huge is refused by regexp itself, quickly; neither filters anything
// out. RE2 matching is linear, so a pathological backtracking pattern runs in time too.
func TestRowMatcherBounds(t *testing.T) {
	long := strings.Repeat("a", MaxFilterPattern+1)
	if _, err := NewRowMatcher(long, FilterOptions{Regex: true}); err == nil || !strings.Contains(err.Error(), "1000 bytes") {
		t.Errorf("a long pattern: %v", err)
	}
	if _, err := NewRowMatcher(long, FilterOptions{}); err != nil {
		t.Errorf("a long text without regex is terms: %v", err)
	}
	start := time.Now()
	if _, err := NewRowMatcher("((((a{100}){100}){100}){100})", FilterOptions{Regex: true}); err == nil {
		t.Error("a pattern too large to compile was taken")
	}
	m, err := NewRowMatcher("(a+)+$", FilterOptions{Regex: true}) // exponential with backtracking
	if err != nil {
		t.Fatal(err)
	}
	if m.Match(rowOf(strings.Repeat("a", 10_000) + "!")) {
		t.Error("(a+)+$ matched a text ending in !")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Errorf("bounded patterns took %v", d)
	}
}

// TestSnapshotFiltersWithOptions: a view's snapshot applies the search with its toggles, carries
// them, and with an invalid pattern shows every row and says why.
func TestSnapshotFiltersWithOptions(t *testing.T) {
	src := testkit.NewMemorySource()
	for _, n := range []string{"Web-0", "web-1", "api-0"} {
		src.Set(testkit.PodsGVR, testkit.Pod("shop", n))
	}
	s, _ := newTestSession(t, src, 5*time.Millisecond)
	connectTest(t, s)
	names := func(snap *TableSnapshot) string {
		out := make([]string, len(snap.Rows))
		for i, r := range snap.Rows {
			out[i] = r.Ref.Name
		}
		return strings.Join(out, ",")
	}
	check := func(filter string, o FilterOptions, want, wantErr string) {
		t.Helper()
		s.UpdateUI(func(u *UIState) { u.Filter, u.FilterOptions = filter, o })
		snap := waitTable(t, s, "pods", func(sn *TableSnapshot) bool {
			return sn.Synced && sn.Filter == filter && sn.FilterOptions == o
		})
		if got := names(snap); got != want || !strings.Contains(snap.FilterError, wantErr) || (wantErr == "") != (snap.FilterError == "") {
			t.Errorf("%q %+v: rows %s, error %q; want %s, %q", filter, o, got, snap.FilterError, want, wantErr)
		}
	}
	check("web", FilterOptions{}, "Web-0,web-1", "")
	check("web", FilterOptions{MatchCase: true}, "web-1", "")
	check("^w", FilterOptions{Regex: true}, "Web-0,web-1", "")
	check("^w", FilterOptions{Regex: true, MatchCase: true}, "web-1", "")
	check("^(web|api)-0$", FilterOptions{Regex: true}, "api-0,Web-0", "")
	check("web(", FilterOptions{Regex: true}, "api-0,Web-0,web-1", "missing closing )")
}
