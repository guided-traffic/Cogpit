package model

import (
	"slices"
	"testing"
)

func stateTexts(states []RowState) []string {
	var out []string
	for _, s := range states {
		out = append(out, s.State)
	}
	return out
}

// TestStatesOfColumns: a kind's states come from its state columns: a Status cell's whole text, a
// Conditions cell's words, a cell's parts with their colours; an empty cell gives none.
func TestStatesOfColumns(t *testing.T) {
	pods := &KindProfile{Key: "pods", Columns: []Column{{ID: "name"}, {ID: "status", Title: "Status"}}}
	s := StatesOf(pods)
	if got := s.Columns(); !slices.Equal(got, []string{"Status"}) {
		t.Fatalf("columns %v", got)
	}
	row := Row{Cells: []Cell{{Text: "web"}, {Text: "CrashLoopBackOff", Status: StatusError}}}
	if got := s.Of(&row); len(got) != 1 || got[0] != (RowState{State: "CrashLoopBackOff", Text: "CrashLoopBackOff", Status: StatusError}) {
		t.Errorf("pod states %+v", got)
	}
	if got := s.Of(&Row{Cells: []Cell{{Text: "web"}, {}}}); got != nil {
		t.Errorf("an empty cell has states %+v", got)
	}
	if got := s.Of(&Row{Cells: []Cell{{Text: "web"}}}); got != nil {
		t.Errorf("a short row has states %+v", got)
	}

	deploys := &KindProfile{Columns: []Column{{ID: "name"}, {ID: "conditions", Title: "Conditions"}}}
	row = Row{Cells: []Cell{{Text: "web"}, {Text: "Available  Progressing", Status: StatusOK}}}
	got := StatesOf(deploys).Of(&row)
	if !slices.Equal(stateTexts(got), []string{"Available", "Progressing"}) || got[0].Status != StatusNone {
		t.Errorf("conditions states %+v", got)
	}

	nodes := &KindProfile{Columns: []Column{{ID: "name"}, {ID: "conditions", Title: "Conditions"}}}
	row = Row{Cells: []Cell{{Text: "n1"}, PartsCell(StatusOK, CellPart{Text: "SchedulingDisabled", Status: StatusWarning}, CellPart{Text: "Ready", Status: StatusOK})}}
	got = StatesOf(nodes).Of(&row)
	if !slices.Equal(stateTexts(got), []string{"SchedulingDisabled", "Ready"}) || got[0].Status != StatusWarning || got[1].Status != StatusOK {
		t.Errorf("parts states %+v", got)
	}

	// A kind that names its state columns: an event's Type; a Words Status.
	events := &KindProfile{Columns: []Column{{ID: "name"}, {ID: "type", Title: "Type"}, {ID: "status", Title: "Status"}},
		States: []StateColumn{{ID: "type"}}}
	row = Row{Cells: []Cell{{Text: "e"}, {Text: "Warning", Status: StatusWarning}, {Text: "ignored"}}}
	if got := stateTexts(StatesOf(events).Of(&row)); !slices.Equal(got, []string{"Warning"}) {
		t.Errorf("declared states %v", got)
	}
}

// TestStatesOfTokens: a kind without a state column offers its rows' status tokens by name; a row
// without a token has no state.
func TestStatesOfTokens(t *testing.T) {
	sets := &KindProfile{Columns: []Column{{ID: "name"}, {ID: "pods", Title: "Pods"}}}
	s := StatesOf(sets)
	if s.Columns() != nil || s.Facets() != 1 {
		t.Fatalf("columns %v facets %d", s.Columns(), s.Facets())
	}
	if got := s.Of(&Row{Status: StatusPending}); len(got) != 1 || got[0].State != "Pending" || got[0].Status != StatusPending {
		t.Errorf("token states %+v", got)
	}
	if got := s.Of(&Row{}); got != nil {
		t.Errorf("no token: %+v", got)
	}
	for _, st := range []Status{StatusOK, StatusSucceeded, StatusPending, StatusWarning, StatusError, StatusTerminating, StatusUnknown, StatusStopped} {
		if StatusTitle(st) == "" {
			t.Errorf("token %q has no title", st)
		}
	}
	if (States{}).Of(&Row{Status: StatusOK}) != nil {
		t.Error("the zero States found a state")
	}
}

// TestStateMatcher: one state column keeps a row with any selected state; several are each a
// condition of their own, a state qualified by its column's title; an unqualified state matches its
// text in any column; nothing selected keeps every row.
func TestStateMatcher(t *testing.T) {
	pods := StatesOf(&KindProfile{Columns: []Column{{ID: "name"}, {ID: "status", Title: "Status"}}})
	row := func(cells ...string) Row {
		r := Row{Cells: []Cell{{Text: "x"}}}
		for _, c := range cells {
			r.Cells = append(r.Cells, Cell{Text: c})
		}
		return r
	}
	running, pending := row("Running"), row("Pending")
	m := pods.Matcher([]string{"Running", "Failed"})
	if !m.Match(pods.Of(&running)) || m.Match(pods.Of(&pending)) {
		t.Error("one column: any selected state")
	}
	if !(StateMatcher{}).All() || !pods.Matcher(nil).Match(pods.Of(&pending)) {
		t.Error("nothing selected filters")
	}
	if m := pods.Matcher([]string{"Gone"}); m.Match(pods.Of(&running)) || m.All() {
		t.Error("a state no row carries keeps rows")
	}

	argo := &KindProfile{Columns: []Column{{ID: "name"}, {ID: "sync", Title: "Sync"}, {ID: "health", Title: "Health"}}}
	s := StatesOf(argo)
	if got := s.Columns(); !slices.Equal(got, []string{"Sync", "Health"}) {
		t.Fatalf("columns %v", got)
	}
	a := row("OutOfSync", "Degraded")
	b := row("OutOfSync", "Healthy")
	c := row("Synced", "Unknown")
	d := row("Unknown", "Healthy")
	if got := stateTexts(s.Of(&a)); !slices.Equal(got, []string{"Sync: OutOfSync", "Health: Degraded"}) {
		t.Fatalf("qualified %v", got)
	}
	if q := s.Qualified(argo, "health", "Degraded"); q != "Health: Degraded" {
		t.Errorf("qualified %q", q)
	}
	if q := pods.Qualified(&KindProfile{Columns: []Column{{ID: "name"}, {ID: "status"}}}, "status", "Running"); q != "Running" {
		t.Errorf("one column qualified %q", q)
	}
	cases := []struct {
		sel  []string
		want []bool // a, b, c, d
	}{
		{[]string{"Sync: OutOfSync"}, []bool{true, true, false, false}},
		{[]string{"Sync: OutOfSync", "Health: Degraded"}, []bool{true, false, false, false}},
		{[]string{"Sync: OutOfSync", "Sync: Synced"}, []bool{true, true, true, false}},
		{[]string{"Health: Degraded", "Health: Unknown"}, []bool{true, false, true, false}},
		{[]string{"Unknown"}, []bool{false, false, true, true}},
		{[]string{"Unknown", "Health: Healthy"}, []bool{false, false, false, true}},
	}
	for _, tc := range cases {
		m := s.Matcher(tc.sel)
		for i, r := range []Row{a, b, c, d} {
			if got := m.Match(s.Of(&r)); got != tc.want[i] {
				t.Errorf("%v on row %d: %v, want %v", tc.sel, i, got, tc.want[i])
			}
		}
	}
	if !RowMatchesStates(argo, a, []string{"Health: Degraded"}) || RowMatchesStates(argo, b, []string{"Health: Degraded"}) || !RowMatchesStates(argo, b, nil) {
		t.Error("RowMatchesStates")
	}
}
