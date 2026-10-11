package commands_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// hiddenIDs returns the ids of a snapshot's hidden columns.
func hiddenIDs(t *engine.TableSnapshot) []string {
	var out []string
	for _, c := range t.Columns {
		if c.Hidden {
			out = append(out, c.ID)
		}
	}
	return out
}

// TestSetColumns: an agent hides and shows columns as the header's ⋮ does; the state is per view,
// visible in ui_state, applied to the view's snapshot, and validated.
func TestSetColumns(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	agent := engine.OriginExternalAgent
	h.must(t, agent, "ui.openView", map[string]any{"view": "pods"}, nil)

	h.must(t, agent, "ui.setColumns", map[string]any{"hidden": []string{"node", "namespace"}}, nil)
	var st commands.StateResult
	h.must(t, agent, "ui.state", nil, &st)
	if got := st.HiddenColumns["pods"]; !slices.Equal(got, []string{"namespace", "node"}) {
		t.Fatalf("hiddenColumns %v, want the ids in the columns' order", st.HiddenColumns)
	}
	waitTable(t, h.s, "pods", func(s *engine.TableSnapshot) bool {
		return slices.Equal(hiddenIDs(s), []string{"namespace", "node"}) && len(s.Rows) == 4
	})

	// Another view keeps its own columns.
	h.must(t, agent, "ui.setColumns", map[string]any{"view": "services", "hidden": []string{"clusterIP"}}, nil)
	if u := h.s.UIState(); !slices.Equal(u.HiddenColumns["services"], []string{"clusterIP"}) || len(u.HiddenColumns["pods"]) != 2 {
		t.Fatalf("hiddenColumns %v", u.HiddenColumns)
	}

	for _, bad := range []struct {
		params map[string]any
		want   string
	}{
		{map[string]any{"hidden": []string{"ip"}}, `no column "ip"`},
		{map[string]any{"hidden": []string{"name", "namespace", "node"}}, "at least one column"},
		{map[string]any{"view": "widgets"}, "unknown kind"},
	} {
		if err := h.run(t, agent, "ui.setColumns", bad.params, nil); err == nil || !strings.Contains(err.Error(), bad.want) {
			t.Errorf("%v: %v, want %q", bad.params, err, bad.want)
		}
	}

	// Showing every column is the default for pods: the entry goes; so does a reset.
	h.must(t, agent, "ui.setColumns", map[string]any{"hidden": []string{}}, nil)
	h.must(t, agent, "ui.setColumns", map[string]any{"view": "services", "reset": true}, nil)
	if u := h.s.UIState(); len(u.HiddenColumns) != 0 {
		t.Fatalf("hiddenColumns %v after showing every column", u.HiddenColumns)
	}
	waitTable(t, h.s, "pods", func(s *engine.TableSnapshot) bool { return len(hiddenIDs(s)) == 0 })
}

// TestExportCSV: an agent exports the open view as the table shows it: visible columns, filtered
// and sorted rows; only the open view can be exported.
func TestExportCSV(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	agent := engine.OriginExternalAgent
	h.must(t, agent, "ui.openView", map[string]any{"view": "services"}, nil)
	h.must(t, agent, "ui.setNamespace", map[string]any{"namespaces": []string{"shop"}}, nil)
	h.must(t, agent, "ui.sort", map[string]any{"column": "name", "desc": true}, nil)
	h.must(t, agent, "ui.setColumns", map[string]any{"hidden": []string{"namespace"}}, nil)
	waitTable(t, h.s, "services", func(s *engine.TableSnapshot) bool {
		return len(s.Rows) == 2 && s.Sort.Desc && len(hiddenIDs(s)) == 1
	})

	var res commands.CSVResult
	h.must(t, agent, "table.exportCsv", nil, &res)
	want := "Name,Cluster IP\ncheckout,10.96.100.100\ncart,10.96.100.101\n"
	if res.CSV != want || res.Rows != 2 || res.View != "services" || res.Filename != "services-test.csv" {
		t.Fatalf("export %+v, want csv %q", res, want)
	}
	if err := h.run(t, agent, "table.exportCsv", map[string]any{"view": "pods"}, nil); err == nil || !strings.Contains(err.Error(), "only the open view") {
		t.Fatalf("exporting a view that is not open: %v", err)
	}
}

// TestSetGroups: the Definitions view's group filter is UI state, trimmed and without duplicates.
func TestSetGroups(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	h.must(t, engine.OriginExternalAgent, "ui.setGroups", map[string]any{"groups": []string{" cert-manager.io", "", "cert-manager.io", "example.com"}}, nil)
	var st commands.StateResult
	h.must(t, engine.OriginExternalAgent, "ui.state", nil, &st)
	if !slices.Equal(st.Groups, []string{"cert-manager.io", "example.com"}) {
		t.Fatalf("groups %v", st.Groups)
	}
	h.must(t, engine.OriginGUI, "ui.setGroups", map[string]any{}, nil)
	if g := h.s.UIState().Groups; len(g) != 0 {
		t.Fatalf("groups %v after clearing", g)
	}
}
