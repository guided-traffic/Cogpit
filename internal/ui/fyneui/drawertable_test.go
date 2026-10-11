package fyneui

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
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

// TestDrawerTableLines: a cell of several lines shows them one under the other from the row's top,
// and the next row starts below the tallest cell.
func TestDrawerTableLines(t *testing.T) {
	f := connectedFake(t, 3)
	ref := f.tables["pods"].Rows[0].Ref
	f.details[ref.String()] = &engine.DetailModel{
		Ref: ref, Title: "Pod: " + ref.Name, Events: []engine.EventItem{},
		Sections: []model.DetailSection{{Title: "Rules", Table: &model.DetailTable{
			Columns: []string{"Resources", "Verbs"},
			Rows:    [][]string{{"pods\npods/log", "get\nlist\nwatch"}, {"configmaps", "update"}},
		}}},
	}
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && len(sh.drawer.details.Objects) > 2 })

	pos := map[string]float32{}
	for _, l := range findAll[*textLine](sh.drawer.details) {
		pos[l.text] = fyne.CurrentApp().Driver().AbsolutePositionForObject(l).Y
	}
	for _, s := range []string{"pods", "pods/log", "get", "list", "watch", "configmaps", "update"} {
		if _, ok := pos[s]; !ok {
			t.Fatalf("no line %q in %q", s, drawerTexts(sh))
		}
	}
	if pos["pods"] != pos["get"] || pos["pods/log"] <= pos["pods"] || pos["watch"] <= pos["list"] {
		t.Errorf("lines not one under the other from the row's top: %v", pos)
	}
	if pos["configmaps"] <= pos["watch"] || pos["configmaps"] != pos["update"] {
		t.Errorf("the next row must start below the tallest cell: %v", pos)
	}
}

// TestCapWidths: columns that do not fit give up width from the widest down; the narrow ones keep
// theirs, and columns that fit are left alone.
func TestCapWidths(t *testing.T) {
	w := []float32{150, 45, 110, 125, 165}
	capWidths(w, 485)
	if want := []float32{110, 45, 110, 110, 110}; !slices.Equal(w, want) {
		t.Errorf("capped %v, want %v", w, want)
	}
	fits := []float32{80, 40}
	capWidths(fits, 200)
	if !slices.Equal(fits, []float32{80, 40}) {
		t.Errorf("columns that fit changed: %v", fits)
	}
	capWidths(nil, 10)
}
