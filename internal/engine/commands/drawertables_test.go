package commands_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// waitUntil fails the test when cond does not hold within ten seconds.
func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, cond) {
		t.Fatalf("timed out waiting for %s", what)
	}
}

// drawerPodsResult is object.get's result as far as the Pods table goes.
type drawerPodsResult struct {
	Sections []model.DetailSection `json:"sections"`
}

func (r drawerPodsResult) pods(t *testing.T) *model.DetailTable {
	t.Helper()
	for _, s := range r.Sections {
		if s.Title == "Pods" && s.Table != nil {
			return s.Table
		}
	}
	t.Fatalf("no Pods table among %d sections", len(r.Sections))
	return nil
}

// ui.sortDrawerTable sorts the drawer's Pods by a column's id or title, for agents as for the
// window; it needs the drawer open on an object with a sortable table and a column the table
// has; the order is UI state (drawerSort) that object.get's rows follow.
func TestSortDrawerTable(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.NodesGVR, testkit.Object("v1", "Node", "", "node-1", nil))
	src.Set(testkit.PodsGVR, onNode(testkit.Pod("shop", "web-0"), "node-1"))
	src.Set(testkit.PodsGVR, onNode(testkit.Pod("argo", "api-0"), "node-1"))
	src.Set(testkit.PodsGVR, onNode(testkit.Pod("jobs", "batch-0"), "node-1"))
	h := relatedHarness(t, src)
	agent := engine.OriginExternalAgent
	node := model.ObjectRef{Kind: "nodes", Name: "node-1"}

	if err := h.run(t, agent, "ui.sortDrawerTable", map[string]any{"column": "namespace"}, nil); err == nil || !strings.Contains(err.Error(), "drawer is closed") {
		t.Fatalf("with the drawer closed: %v", err)
	}
	h.must(t, agent, "ui.navigate", map[string]any{"ref": node}, nil)
	var got drawerPodsResult
	waitUntil(t, "the node's Pods", func() bool {
		got = drawerPodsResult{}
		if err := h.run(t, agent, "object.get", map[string]any{"ref": node, "noYaml": true}, &got); err != nil {
			return false
		}
		for _, s := range got.Sections {
			if s.Title == "Pods" && s.Table != nil && len(s.Table.Rows) == 3 {
				return true
			}
		}
		return false
	})
	if tbl := got.pods(t); !tbl.Sortable || tbl.Sort == nil || tbl.Sort.Column != "name" || !slices.Contains(tbl.IDs, model.ColumnMemory) {
		t.Fatalf("the Pods table: sortable %v, sort %+v, ids %q", tbl.Sortable, tbl.Sort, tbl.IDs)
	}

	for _, bad := range []map[string]any{
		{"column": "nope"},
		{"section": "Containers", "column": "name"},
	} {
		if err := h.run(t, agent, "ui.sortDrawerTable", bad, nil); err == nil {
			t.Errorf("%v: no error", bad)
		} else if bad["column"] == "nope" && !strings.Contains(err.Error(), "memory") {
			t.Errorf("the error does not list the columns: %v", err)
		}
	}

	var u engine.UIState
	h.must(t, agent, "ui.sortDrawerTable", map[string]any{"column": "Namespace", "desc": true}, &u) // a title, the only table
	if got := u.DrawerSort["Pods"]; got != (engine.SortSpec{Column: "namespace", Desc: true}) {
		t.Fatalf("drawerSort %+v", u.DrawerSort)
	}
	h.must(t, agent, "object.get", map[string]any{"ref": node, "noYaml": true}, &got)
	tbl := got.pods(t)
	var names []string
	for _, r := range tbl.Rows {
		names = append(names, r[0])
	}
	if !slices.Equal(names, []string{"web-0", "batch-0", "api-0"}) || tbl.Sort == nil || *tbl.Sort != (model.TableSort{Column: "namespace", Desc: true}) {
		t.Errorf("by namespace, descending: %q, sort %+v", names, tbl.Sort)
	}

	h.must(t, engine.OriginGUI, "ui.sortDrawerTable", map[string]any{"section": "pods", "column": "memory"}, &u)
	if got := u.DrawerSort["Pods"]; got != (engine.SortSpec{Column: model.ColumnMemory}) {
		t.Errorf("drawerSort after the window's click %+v", u.DrawerSort)
	}

	// An object whose drawer has no sortable table.
	h.must(t, agent, "ui.navigate", map[string]any{"ref": model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}}, nil)
	if err := h.run(t, agent, "ui.sortDrawerTable", map[string]any{"column": "name"}, nil); err == nil || !strings.Contains(err.Error(), "no sortable table") {
		t.Errorf("a pod's drawer: %v", err)
	}
}
