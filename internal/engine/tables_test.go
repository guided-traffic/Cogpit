package engine

import (
	"slices"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestDefinitionsGroupFilterAndColumns: the Definitions view lists the definitions of the groups in
// UIState.Groups only, other views ignore the group filter, and a view's snapshot hides the columns
// UIState.HiddenColumns lists for it, or its kind's defaults without an entry.
func TestDefinitionsGroupFilterAndColumns(t *testing.T) {
	defs, ok := kinds.ByKey(kinds.KeyCustomResourceDefinitions)
	if !ok {
		t.Fatal("no Definitions profile")
	}
	src := testkit.NewMemorySource()
	src.Set(crdsGVR, orderCRD())
	src.Set(crdsGVR, testkit.CRD("example.com", "widgets", "Widget", true,
		testkit.CRDVersion{Name: "v1", Served: true, Storage: true}))
	src.Set(testkit.PodsGVR, testkit.Pod("default", "web-0"))
	s := newCRDSession(t, src, defs)
	openView(s, kinds.KeyCustomResourceDefinitions)
	snap := waitTable(t, s, kinds.KeyCustomResourceDefinitions, func(t *TableSnapshot) bool { return len(t.Rows) == 2 })
	if got := hiddenOf(snap); !slices.Equal(got, []string{"name"}) {
		t.Fatalf("default hidden columns %v, want the Definitions view's hidden name", got)
	}

	s.UpdateUI(func(u *UIState) { u.Groups = []string{orderGroup} })
	snap = waitTable(t, s, kinds.KeyCustomResourceDefinitions, func(t *TableSnapshot) bool { return len(t.Rows) == 1 })
	if snap.Rows[0].Ref.Name != "orders."+orderGroup || snap.Total != 2 {
		t.Fatalf("filtered by group: %s, total %d", summary(snap), snap.Total)
	}

	s.UpdateUI(func(u *UIState) {
		u.HiddenColumns = map[string][]string{kinds.KeyCustomResourceDefinitions: {"scope", "age"}}
	})
	snap = waitTable(t, s, kinds.KeyCustomResourceDefinitions, func(t *TableSnapshot) bool {
		return slices.Equal(hiddenOf(t), []string{"scope", "age"})
	})
	if len(snap.Rows) != 1 || len(snap.Rows[0].Cells) != len(snap.Columns) {
		t.Fatalf("rows no longer fit the columns: %s", summary(snap))
	}

	// The group filter is the Definitions view's only.
	openView(s, "pods")
	waitTable(t, s, "pods", func(t *TableSnapshot) bool { return len(t.Rows) == 1 })
}

func hiddenOf(t *TableSnapshot) []string {
	var out []string
	for _, c := range t.Columns {
		if c.Hidden {
			out = append(out, c.ID)
		}
	}
	return out
}
