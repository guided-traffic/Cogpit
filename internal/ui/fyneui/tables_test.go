package fyneui

import (
	"context"
	"encoding/json"
	"slices"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The tests of the table's header and keyboard features (ui-parity › Tables).

// TestNamespaceFilterHiddenOnClusterScopedViews: Lens shows no namespace filter on a
// cluster-scoped view; a namespaced view shows it again.
func TestNamespaceFilterHiddenOnClusterScopedViews(t *testing.T) {
	f := kindsFake(t, "nodes")
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the nodes view", func() bool { return sh.view.snap != nil && sh.view.snap.View == "nodes" })
	if sh.view.nsBox.Visible() {
		t.Error("the namespace filter shows on Nodes")
	}
	sh.dispatch("ui.openView", openViewParams{View: "pods"}, nil)
	q.until(t, "the pods view", func() bool { return sh.view.snap != nil && sh.view.snap.View == "pods" })
	if !sh.view.nsBox.Visible() {
		t.Error("the namespace filter is hidden on Pods")
	}
}

// handleColumns makes the fake answer ui.setColumns as the engine does: the view's snapshot hides
// the listed columns and the UI state records them.
func handleColumns(f *fakeEngine) {
	f.handlers["ui.setColumns"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p columnsParams
		if err := json.Unmarshal(inv.Params, &p); err != nil {
			return nil, err
		}
		f.mu.Lock()
		s := *f.tables[p.View]
		s.Columns = engine.ColumnsShown(s.Columns, p.Hidden, true)
		s.Revision++
		f.tables[p.View] = &s
		f.mu.Unlock()
		f.update(func(u *engine.UIState) {
			if u.HiddenColumns == nil {
				u.HiddenColumns = map[string][]string{}
			}
			u.HiddenColumns[p.View] = p.Hidden
		})
		f.emit(engine.Event{Type: engine.EventTable, View: p.View})
		return nil, nil
	}
}

func columnCheck(t *testing.T, sh *shell, title string) *widget.Check {
	t.Helper()
	for _, c := range sh.view.colChecks {
		if c.Text == title {
			return c
		}
	}
	t.Fatalf("no column %q in the list", title)
	return nil
}

// TestColumnsMenuShowsAndHides: the ⋮ at the end of the header row lists every column of the view
// with a check; unchecking one sends ui.setColumns and the table drops the column, checking it
// again brings it back. The list stays open while toggling.
func TestColumnsMenuShowsAndHides(t *testing.T) {
	f := kindsFake(t, "pods")
	handleColumns(f)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the pods view", func() bool { return sh.view.snap != nil && sh.view.snap.View == "pods" })
	tv := sh.view

	menu := headerFor(t, sh, tv.menuColumn())
	if !menu.menu {
		t.Fatal("the header of the ⋮ column is not the column menu")
	}
	test.Tap(menu)
	if !tv.columnsOpen() || len(tv.colChecks) != len(tv.snap.Columns) {
		t.Fatalf("column list open %v with %d checks for %d columns", tv.columnsOpen(), len(tv.colChecks), len(tv.snap.Columns))
	}
	ns := columnCheck(t, sh, "Namespace")
	if !ns.Checked {
		t.Fatal("Namespace is shown but unchecked")
	}
	visible := len(tv.cols)
	test.Tap(ns)
	q.until(t, "the namespace column to go", func() bool { return len(tv.cols) == visible-1 })
	if got := decodeParams[columnsParams](t, f.callsOf("ui.setColumns")[0]); got.View != "pods" || !slices.Equal(got.Hidden, []string{"namespace"}) {
		t.Fatalf("ui.setColumns %+v", got)
	}
	for _, ci := range tv.cols {
		if tv.snap.Columns[ci].ID == "namespace" {
			t.Fatal("the hidden namespace column is still in the table")
		}
	}
	if ns.Checked || !tv.columnsOpen() {
		t.Fatalf("after hiding: checked %v, list open %v", ns.Checked, tv.columnsOpen())
	}
	test.Tap(ns)
	q.until(t, "the namespace column to come back", func() bool { return len(tv.cols) == visible })
	if !ns.Checked {
		t.Error("Namespace is shown again but unchecked")
	}
}

// TestColumnsMenuKeepsOneColumn: the list does not hide the last visible column.
func TestColumnsMenuKeepsOneColumn(t *testing.T) {
	f := kindsFake(t, "nodes")
	handleColumns(f)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the nodes view", func() bool { return sh.view.snap != nil && sh.view.snap.View == "nodes" })
	var hidden []string
	for _, c := range sh.view.snap.Columns[1:] {
		hidden = append(hidden, c.ID)
	}
	sh.dispatch("ui.setColumns", columnsParams{View: "nodes", Hidden: hidden}, nil)
	q.until(t, "one column left", func() bool { return len(sh.view.cols) == 1 })
	sh.view.openColumns()
	name := columnCheck(t, sh, "Name")
	test.Tap(name)
	q.drain()
	if !name.Checked || len(f.callsOf("ui.setColumns")) != 1 {
		t.Fatalf("the last column was hidden: checked %v, calls %d", name.Checked, len(f.callsOf("ui.setColumns")))
	}
}

func renderColumnsMenu(t *testing.T, mode ThemeMode, name string) {
	f := kindsFake(t, "pods")
	handleColumns(f)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the pods view", func() bool {
		return sh.view.snap != nil && sh.view.snap.View == "pods" && sh.diag.total > 0
	})
	sh.dispatch("ui.setColumns", columnsParams{View: "pods", Hidden: []string{"qos"}}, nil)
	q.until(t, "QoS hidden", func() bool { return columnHidden(sh.view.snap, "qos") })
	settle(w)
	test.Tap(headerFor(t, sh, sh.view.menuColumn()))
	savePNG(t, w, name)
}

func columnHidden(s *engine.TableSnapshot, id string) bool {
	for _, c := range s.Columns {
		if c.ID == id {
			return c.Hidden
		}
	}
	return false
}

func TestRenderColumnsMenuDark(t *testing.T) {
	renderColumnsMenu(t, ThemeDark, "columns_menu_dark.png")
}
func TestRenderColumnsMenuLight(t *testing.T) {
	renderColumnsMenu(t, ThemeLight, "columns_menu_light.png")
}

// definitionsFake is a fake on the Definitions view with definitions of three API groups, and
// the custom kinds they define in kinds.list. It answers ui.setGroups as the engine does.
func definitionsFake(t *testing.T) *fakeEngine {
	f := connectedFake(t, 1)
	crds := []*unstructured.Unstructured{
		testkit.CRD("acme.cert-manager.io", "orders", "Order", true),
		testkit.CRD("acme.cert-manager.io", "challenges", "Challenge", true),
		testkit.CRD("cert-manager.io", "certificates", "Certificate", true),
		testkit.CRD("cert-manager.io", "clusterissuers", "ClusterIssuer", false),
		testkit.CRD("gateway.networking.k8s.io", "gateways", "Gateway", true),
	}
	f.kinds = append(f.kinds, kindInfo{Kind: kinds.KeyCustomResourceDefinitions, Title: "Definitions", APIKind: "CustomResourceDefinition", Group: "apiextensions.k8s.io", Version: "v1"})
	for _, c := range crds {
		k, err := kinds.ParseCRD(c)
		if err != nil {
			t.Fatal(err)
		}
		f.kinds = append(f.kinds, kindInfo{Kind: k.Key(), Title: k.Kind, APIKind: k.Kind, Group: k.Group, Version: "v1", Namespaced: k.Namespaced})
	}
	all := kindSnapshot(t, kinds.KeyCustomResourceDefinitions, crds)
	f.tables[kinds.KeyCustomResourceDefinitions] = all
	f.ui.View = kinds.KeyCustomResourceDefinitions
	f.handlers["ui.setGroups"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p groupsParams
		if err := json.Unmarshal(inv.Params, &p); err != nil {
			return nil, err
		}
		s := *all
		s.Rows = nil
		gi := slices.IndexFunc(s.Columns, func(c model.Column) bool { return c.ID == "group" })
		for _, r := range all.Rows {
			if len(p.Groups) == 0 || slices.Contains(p.Groups, r.Cells[gi].Text) {
				s.Rows = append(s.Rows, r)
			}
		}
		s.Revision++
		f.mu.Lock()
		f.tables[kinds.KeyCustomResourceDefinitions] = &s
		f.mu.Unlock()
		f.update(func(u *engine.UIState) { u.Groups = p.Groups })
		f.emit(engine.Event{Type: engine.EventTable, View: kinds.KeyCustomResourceDefinitions})
		return nil, nil
	}
	return f
}

// TestDefinitionsGroupFilter: the Definitions view shows "All groups" where other views have the
// namespace filter; it offers the cluster's API groups, and a check narrows the list through
// ui.setGroups without closing the popup.
func TestDefinitionsGroupFilter(t *testing.T) {
	f := definitionsFake(t)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the definitions", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil && len(sh.view.snap.Rows) == 5 })
	tv := sh.view
	if !tv.nsBox.Visible() || tv.nsButton.Text != "All groups" {
		t.Fatalf("filter visible %v, text %q", tv.nsBox.Visible(), tv.nsButton.Text)
	}
	test.Tap(tv.nsButton)
	var labels []string
	for _, c := range tv.nsChecks {
		labels = append(labels, c.Text)
	}
	if want := []string{"All groups", "acme.cert-manager.io", "cert-manager.io", "gateway.networking.k8s.io"}; !slices.Equal(labels, want) {
		t.Fatalf("group filter offers %v, want %v", labels, want)
	}
	test.Tap(tv.nsChecks[2]) // cert-manager.io
	q.until(t, "two definitions", func() bool { return len(tv.snap.Rows) == 2 })
	if tv.nsButton.Text != "cert-manager.io" || !tv.nsPopup.Visible() || tv.nsChecks[0].Checked {
		t.Fatalf("after choosing a group: text %q, popup %v, all checked %v", tv.nsButton.Text, tv.nsPopup.Visible(), tv.nsChecks[0].Checked)
	}
	if got := decodeParams[groupsParams](t, f.callsOf("ui.setGroups")[0]).Groups; !slices.Equal(got, []string{"cert-manager.io"}) {
		t.Fatalf("ui.setGroups %v", got)
	}
	if len(f.callsOf("ui.setNamespace")) != 0 {
		t.Fatal("the group filter changed the namespace filter")
	}
	test.Tap(tv.nsChecks[0]) // All groups
	q.until(t, "every definition", func() bool { return len(tv.snap.Rows) == 5 })
	if tv.nsButton.Text != "All groups" {
		t.Fatalf("text %q after All groups", tv.nsButton.Text)
	}
}

func renderDefinitionsGroups(t *testing.T, mode ThemeMode, name string) {
	f := definitionsFake(t)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the definitions", func() bool {
		return len(sh.kinds) > 0 && sh.view.snap != nil && len(sh.view.snap.Rows) == 5 && sh.diag.total > 0
	})
	sh.dispatch("ui.setGroups", groupsParams{Groups: []string{"acme.cert-manager.io", "cert-manager.io"}}, nil)
	q.until(t, "two groups", func() bool { return len(sh.view.snap.Rows) == 4 })
	settle(w)
	test.Tap(sh.view.nsButton)
	savePNG(t, w, name)
}

func TestRenderDefinitionsGroupsDark(t *testing.T) {
	renderDefinitionsGroups(t, ThemeDark, "definitions_groups_dark.png")
}

func TestRenderDefinitionsGroupsLight(t *testing.T) {
	renderDefinitionsGroups(t, ThemeLight, "definitions_groups_light.png")
}
