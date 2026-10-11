package commands_test

import (
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// TestWorkloadsOverviewCommands: an agent opens the Workloads Overview as a page, reads it, and
// works its events table with the commands of any view: sort, select (the drawer opens on the
// event), columns and CSV.
func TestWorkloadsOverviewCommands(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	agent := engine.OriginExternalAgent
	h.must(t, agent, "ui.openView", map[string]any{"view": engine.WorkloadsOverview}, nil)
	snap := waitTable(t, h.s, engine.WorkloadsOverview, func(s *engine.TableSnapshot) bool { return s.Synced && len(s.Rows) == 1 })
	var st commands.StateResult
	h.must(t, agent, "ui.state", nil, &st)
	if st.View != engine.WorkloadsOverview || st.ViewTitle != "Overview" || st.Rows != 1 {
		t.Fatalf("state %+v", st)
	}

	var res commands.OverviewResult
	h.must(t, agent, "workloads.overview", map[string]any{"events": true}, &res)
	if len(res.Kinds) != 2 || res.Kinds[0].Kind != "pods" || res.Kinds[0].Count != 4 || res.Kinds[1].Kind != "deployments" || res.Kinds[1].Count != 2 {
		t.Fatalf("overview kinds %+v", res.Kinds)
	}
	if res.Events == nil || len(res.Events.Rows) != 1 {
		t.Fatalf("overview events %+v", res.Events)
	}
	var infra commands.OverviewResult
	h.must(t, agent, "workloads.overview", map[string]any{"namespaces": []string{"infra"}}, &infra)
	if infra.Kinds[0].Count != 2 || infra.Events != nil {
		t.Fatalf("overview of infra: %+v, events %v", infra.Kinds[0], infra.Events)
	}

	h.must(t, agent, "ui.sort", map[string]any{"column": "name", "desc": true}, nil)
	if err := h.run(t, agent, "ui.sort", map[string]any{"column": "restarts"}, nil); err == nil {
		t.Fatal("sorted the page's events by a column they do not have")
	}
	ev := snap.Rows[0].Ref
	h.must(t, agent, "ui.select", map[string]any{"refs": []model.ObjectRef{ev}}, nil)
	if u := h.s.UIState(); u.View != engine.WorkloadsOverview || u.Drawer == nil || !u.Drawer.Same(ev) {
		t.Fatalf("selecting an event of the page: %+v", u)
	}
	h.must(t, agent, "ui.setColumns", map[string]any{"hidden": []string{"lastSeen"}}, nil)
	waitTable(t, h.s, engine.WorkloadsOverview, func(s *engine.TableSnapshot) bool { return len(hiddenIDs(s)) == 1 })
	var csv commands.CSVResult
	h.must(t, agent, "table.exportCsv", nil, &csv)
	if !strings.HasPrefix(csv.CSV, "Name,Namespace\n") || csv.Rows != 1 || csv.Filename != "workloads_overview-test.csv" {
		t.Fatalf("csv of the page %+v", csv)
	}

	if err := h.run(t, agent, "ui.openView", map[string]any{"view": "workloads/nothing"}, nil); err == nil {
		t.Fatal("opened a page that does not exist")
	}
}
