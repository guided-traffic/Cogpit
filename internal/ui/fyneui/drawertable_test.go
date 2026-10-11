package fyneui

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// TestDrawerTableLinks: a table cell that names an object of a kind the view can show is a link
// that navigates to it; one whose kind Cogpit has no view for stays plain text, as drawer values
// do.
func TestDrawerTableLinks(t *testing.T) {
	f := connectedFake(t, 3)
	ref := f.tables["pods"].Rows[0].Ref
	node := &model.ObjectRef{Kind: "nodes", Name: "worker-a-01"}
	secret := &model.ObjectRef{Kind: "secrets", Namespace: ref.Namespace, Name: "db-password"}
	f.details[ref.String()] = &engine.DetailModel{
		Ref: ref, Title: "Pod: " + ref.Name, Events: []engine.EventItem{},
		Sections: []model.DetailSection{{Title: "Placement", Table: &model.DetailTable{
			Columns: []string{"Kind", "Name"},
			Rows:    [][]string{{"Node", "worker-a-01"}, {"Secret", "db-password"}},
			Links:   [][]*model.ObjectRef{{nil, node}, {nil, secret}},
		}}},
	}
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && len(sh.drawer.details.Objects) > 2 })

	links := drawerLinks(sh)
	if links["worker-a-01"] == nil {
		t.Fatalf("the node cell is not a link: links %v, texts %q", keys(links), drawerTexts(sh))
	}
	if links["db-password"] != nil || !slices.Contains(drawerTexts(sh), "db-password") {
		t.Errorf("a cell naming a kind without a view must be plain text: links %v", keys(links))
	}
	test.Tap(links["worker-a-01"])
	q.until(t, "ui.navigate", func() bool { return len(f.callsOf("ui.navigate")) == 1 })
	if got := decodeParams[refParams](t, f.callsOf("ui.navigate")[0]).Ref; got != (model.ObjectRef{Context: "demo", Kind: "nodes", Name: "worker-a-01"}) {
		t.Errorf("node link %+v", got)
	}
}
