package fyneui

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// selectionRefs are object.deleteMany's parameters.
type selectionRefs struct {
	Refs []model.ObjectRef `json:"refs"`
}

// withDeleteMany registers Delete of a selection in the fake as the engine has it, and answers it
// as the engine does: one confirmation naming every object, then nothing more (the fake deletes
// nothing).
func withDeleteMany(t testing.TB, f *fakeEngine) {
	t.Helper()
	err := engine.Register(f.reg, engine.Spec[selectionRefs]{
		ID: "object.deleteMany", Label: "Delete", Order: engine.OrderDelete, Danger: engine.DangerDestructive,
		Enabled: func(_ *engine.Session, p selectionRefs) error {
			for _, r := range p.Refs {
				if r.Kind == kindPortForwards {
					return engine.ErrNotApplicable
				}
			}
			return f.reason("object.deleteMany")
		},
		Targets: func(_ *engine.Session, p selectionRefs) ([]model.ObjectRef, error) { return p.Refs, nil },
		Run:     func(context.Context, *engine.Session, selectionRefs) (any, error) { return nil, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	f.handlers["object.deleteMany"] = func(ctx context.Context, inv engine.Invocation) (any, error) {
		var p selectionRefs
		if err := json.Unmarshal(inv.Params, &p); err != nil {
			return nil, err
		}
		f.mu.Lock()
		c, uiCtx := f.confirmer, f.ui.Context
		f.mu.Unlock()
		req := engine.ConfirmRequest{Command: inv.ID, Label: "Delete", Danger: engine.DangerDestructive, Origin: inv.Origin, Context: uiCtx, Objects: p.Refs}
		for _, r := range p.Refs {
			if !slices.Contains(req.Namespaces, r.Namespace) {
				req.Namespaces = append(req.Namespaces, r.Namespace)
			}
		}
		ok, err := c.Confirm(ctx, req)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, engine.ErrDeclined
		}
		return nil, nil
	}
}

// heldKeys makes keyModifiers report mods until the test ends or the returned function runs.
func heldKeys(t *testing.T, mods fyne.KeyModifier) (release func()) {
	t.Helper()
	old := keyModifiers
	keyModifiers = func() fyne.KeyModifier { return mods }
	release = func() { keyModifiers = old }
	t.Cleanup(release)
	return release
}

// lastSelect is the last ui.select's parameters.
func lastSelect(t *testing.T, f *fakeEngine) selectModeParams {
	t.Helper()
	calls := f.callsOf("ui.select")
	if len(calls) == 0 {
		t.Fatal("no ui.select")
	}
	return decodeParams[selectModeParams](t, calls[len(calls)-1])
}

func refKeys(refs []model.ObjectRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Key()
	}
	return out
}

// TestRowCheckboxes: every row starts with a checkbox; a checkbox toggles its row without the
// drawer, ⇧ adds the rows from the last one checked, ⌘-click on a row toggles it, a plain click
// opens the drawer and checks nothing (A73); from two selected objects the header shows the
// selection's count, Delete and a clear button.
func TestRowCheckboxes(t *testing.T) {
	f := connectedFake(t, 200)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil })
	settle(w)
	rows := f.tables["pods"].Rows
	tv := sh.view
	if _, cols := tv.length(); cols != len(tv.cols)+2 {
		t.Fatalf("%d table columns for %d view columns: want the checkbox and the ⋮ besides", cols, len(tv.cols))
	}

	box := cellAt(t, sh, 2, checkColumn)
	if box.content.kind != cellCheck || box.iconRes != tv.cellIcon(cellContent{kind: cellCheck}, false) {
		t.Fatalf("row 2's first cell: kind %v, icon %v", box.content.kind, box.iconRes)
	}
	test.Tap(box)
	q.until(t, "the row checked", func() bool { return len(sh.ui.Selected) == 1 })
	if p := lastSelect(t, f); p.Mode != "toggle" || !p.NoDrawer || len(p.Refs) != 1 || p.Refs[0] != rows[2].Ref {
		t.Fatalf("checkbox sent %+v", p)
	}
	settle(w)
	if c := cellAt(t, sh, 2, checkColumn); !c.selected || c.iconRes != tv.cellIcon(cellContent{kind: cellCheck}, true) || sh.ui.Drawer != nil {
		t.Fatalf("checked row: selected %v, checked icon %v, drawer %v", c.selected, c.iconRes == tv.cellIcon(cellContent{kind: cellCheck}, true), sh.ui.Drawer)
	}
	if tv.check.bar.root.Visible() {
		t.Error("the selection bar shows for one object")
	}

	// ⇧ on another checkbox adds the rows between.
	release := heldKeys(t, fyne.KeyModifierShift)
	test.Tap(cellAt(t, sh, 5, checkColumn))
	release()
	q.until(t, "the range", func() bool { return len(sh.ui.Selected) == 4 })
	between := []model.ObjectRef{rows[2].Ref, rows[3].Ref, rows[4].Ref, rows[5].Ref}
	if p := lastSelect(t, f); p.Mode != "add" || !slices.Equal(refKeys(p.Refs), refKeys(between)) {
		t.Fatalf("⇧-click sent %+v", p)
	}
	settle(w)
	if !tv.check.bar.root.Visible() || tv.check.bar.count.text != "4 selected" {
		t.Fatalf("selection bar visible %v: %q", tv.check.bar.root.Visible(), tv.check.bar.count.text)
	}

	// ⌘-click on a row toggles it, without the drawer.
	release = heldKeys(t, fyne.KeyModifierShortcutDefault)
	test.Tap(cellAt(t, sh, 3, firstDataColumn+1))
	release()
	q.until(t, "the row toggled off", func() bool { return len(sh.ui.Selected) == 3 })
	if p := lastSelect(t, f); p.Mode != "toggle" || p.Refs[0] != rows[3].Ref || sh.ui.Drawer != nil {
		t.Fatalf("⌘-click sent %+v, drawer %v", p, sh.ui.Drawer)
	}

	// The bar's clear button empties the selection.
	test.Tap(tv.check.bar.clear)
	q.until(t, "the selection cleared", func() bool { return len(sh.ui.Selected) == 0 })
	settle(w)
	if tv.check.bar.root.Visible() {
		t.Error("the selection bar stays without a selection")
	}

	// A plain click opens the drawer and checks nothing (A73: ui.select's mode open).
	test.Tap(cellAt(t, sh, 7, firstDataColumn+1))
	q.until(t, "the drawer", func() bool { return sh.ui.Drawer != nil })
	if p := lastSelect(t, f); p.Mode != "open" || p.NoDrawer || len(p.Refs) != 1 || p.Refs[0] != rows[7].Ref {
		t.Fatalf("a click sent %+v", p)
	}
	if len(sh.ui.Selected) != 0 {
		t.Errorf("a click checked %v", sh.ui.Selected)
	}
}

// tableScroll is the open view's table's scroller: its offset is how far the table is scrolled.
func tableScroll(t *testing.T, sh *shell) *container.Scroll {
	t.Helper()
	for _, s := range findAll[*container.Scroll](sh.view.table) {
		if s.OnScrolled != nil { // the body's, which resourceTable hooks
			return s
		}
	}
	t.Fatal("no scroller in the table")
	return nil
}

// TestPlainClickKeepsTheCheckedRows: the owner's decision A73: a plain click on a row opens its
// drawer and keeps the checked rows checked. The clicked row is highlighted as the drawer's, its
// checkbox as it was; the selection bar and the header's checkbox still count the checked rows; a
// click on a checked row keeps it checked; ⌘-click still toggles, without moving the drawer or
// scrolling back to the first checked row.
func TestPlainClickKeepsTheCheckedRows(t *testing.T) {
	f := connectedFake(t, 200)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil })
	settle(w)
	rows := f.tables["pods"].Rows
	tv := sh.view
	checked, unchecked := tv.cellIcon(cellContent{kind: cellCheck}, true), tv.cellIcon(cellContent{kind: cellCheck}, false)

	test.Tap(cellAt(t, sh, 2, checkColumn))
	q.until(t, "row 2 checked", func() bool { return len(sh.ui.Selected) == 1 })
	test.Tap(cellAt(t, sh, 4, checkColumn))
	q.until(t, "row 4 checked", func() bool { return len(sh.ui.Selected) == 2 })
	want := refKeys([]model.ObjectRef{rows[2].Ref, rows[4].Ref})

	test.Tap(cellAt(t, sh, 7, firstDataColumn+1))
	q.until(t, "the drawer on row 7", func() bool { return sh.ui.Drawer != nil && *sh.ui.Drawer == rows[7].Ref })
	settle(w)
	if got := refKeys(sh.ui.Selected); !slices.Equal(got, want) {
		t.Fatalf("after a click on row 7 the selection is %v, want %v", got, want)
	}
	if p := lastSelect(t, f); p.Mode != "open" || p.NoDrawer || len(p.Refs) != 1 || p.Refs[0] != rows[7].Ref {
		t.Fatalf("the click sent %+v", p)
	}
	for _, r := range []struct {
		row             int
		selected, check bool
	}{{2, true, true}, {4, true, true}, {7, true, false}, {5, false, false}} {
		box, name := cellAt(t, sh, r.row, checkColumn), cellAt(t, sh, r.row, firstDataColumn+1)
		icon := unchecked
		if r.check {
			icon = checked
		}
		if box.iconRes != icon || name.selected != r.selected || box.selected != r.selected {
			t.Errorf("row %d: checked icon %v (want %v), highlighted %v (want %v)", r.row, box.iconRes == checked, r.check, name.selected, r.selected)
		}
	}
	if !tv.check.bar.root.Visible() || tv.check.bar.count.text != "2 selected" || tv.check.state != checkSome {
		t.Errorf("selection bar %v %q, header state %d", tv.check.bar.root.Visible(), tv.check.bar.count.text, tv.check.state)
	}

	// A click on a checked row: its drawer, still checked; row 7 is no longer highlighted.
	test.Tap(cellAt(t, sh, 4, firstDataColumn+1))
	q.until(t, "the drawer on row 4", func() bool { return sh.ui.Drawer != nil && *sh.ui.Drawer == rows[4].Ref })
	settle(w)
	if got := refKeys(sh.ui.Selected); !slices.Equal(got, want) {
		t.Fatalf("after a click on the checked row 4 the selection is %v, want %v", got, want)
	}
	if c := cellAt(t, sh, 7, firstDataColumn+1); c.selected {
		t.Error("row 7 stays highlighted after the drawer moved on")
	}
	if c := cellAt(t, sh, 4, checkColumn); c.iconRes != checked {
		t.Error("row 4 lost its check with the click")
	}

	// ⌘-click far down toggles that row; the drawer stays, and the table stays where it is.
	release := heldKeys(t, fyne.KeyModifierShortcutDefault)
	far := cellAt(t, sh, 150, firstDataColumn+1) // scrolls there
	scroll := tableScroll(t, sh)
	at := scroll.Offset
	test.Tap(far)
	release()
	q.until(t, "row 150 checked", func() bool { return len(sh.ui.Selected) == 3 })
	settle(w)
	if *sh.ui.Drawer != rows[4].Ref {
		t.Errorf("⌘-click moved the drawer to %v", sh.ui.Drawer)
	}
	if scroll.Offset != at {
		t.Errorf("⌘-click scrolled the table from %v to %v (to the first checked row)", at, scroll.Offset)
	}
}

// TestHeaderCheckbox: the header's checkbox selects every row the view shows, shows a dash while
// some are selected, and clears the selection when all of them are.
func TestHeaderCheckbox(t *testing.T) {
	f := connectedFake(t, 20)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil })
	settle(w)
	head := func() *headerCell { return headerFor(t, sh, checkColumn) }
	if h := head(); !h.box || h.boxed != checkNone {
		t.Fatalf("the first header: checkbox %v, state %d", h.box, h.boxed)
	}
	test.Tap(head())
	q.until(t, "every row", func() bool { return len(sh.ui.Selected) == 20 })
	if p := lastSelect(t, f); !p.All || !p.NoDrawer {
		t.Fatalf("the header's checkbox sent %+v", p)
	}
	settle(w)
	if head().boxed != checkAll {
		t.Fatalf("header state %d with every row selected", head().boxed)
	}

	rows := f.tables["pods"].Rows
	f.update(func(u *engine.UIState) { u.Selected = []model.ObjectRef{rows[1].Ref, rows[4].Ref} })
	q.until(t, "two selected", func() bool { return len(sh.ui.Selected) == 2 })
	settle(w)
	if head().boxed != checkSome {
		t.Fatalf("header state %d with two rows selected", head().boxed)
	}
	test.Tap(head())
	q.until(t, "every row again", func() bool { return len(sh.ui.Selected) == 20 })
	settle(w)
	test.Tap(head())
	q.until(t, "none", func() bool { return len(sh.ui.Selected) == 0 })
	if p := lastSelect(t, f); p.All || len(p.Refs) != 0 || !p.NoDrawer || p.Mode != "replace" {
		t.Fatalf("clearing sent %+v", p)
	}
}

// TestSelectionMenu: the menu of a row inside a selection of several acts on the selection: its
// count, Delete of every selected object (one confirmation listing each) and Clear Selection; a
// row outside it keeps its own menu.
func TestSelectionMenu(t *testing.T) {
	f := connectedFake(t, 20)
	withDeleteMany(t, f)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil })
	settle(w)
	rows := f.tables["pods"].Rows
	sel := []model.ObjectRef{rows[1].Ref, rows[2].Ref, rows[3].Ref}
	f.update(func(u *engine.UIState) { u.Selected = slices.Clone(sel) })
	q.until(t, "three selected", func() bool { return len(sh.ui.Selected) == 3 })
	settle(w)

	m := openMenu(t, sh, q, 2)
	if got := labels(m); !slices.Equal(got, []string{"3 objects selected", "", "Delete", "", "Clear Selection"}) || !m.Items[0].Disabled {
		t.Fatalf("selection menu %v", got)
	}
	if got := labels(openMenu(t, sh, q, 6)); len(got) == 0 || got[0] != "Show Details" {
		t.Fatalf("an unselected row's menu %v", got)
	}

	m = openMenu(t, sh, q, 3)
	item(t, m, "Delete").Action()
	q.until(t, "the confirmation", func() bool { return sh.confirmer.current != nil })
	call := f.callsOf("object.deleteMany")
	if len(call) != 1 || !slices.Equal(refKeys(decodeParams[selectionRefs](t, call[0]).Refs), refKeys(sel)) {
		t.Fatalf("object.deleteMany %+v", call)
	}
	var texts []string
	for _, c := range findAll[*canvas.Text](sh.win.Canvas().Overlays().Top()) {
		texts = append(texts, c.Text)
	}
	all := strings.Join(texts, "\n")
	for _, r := range sel {
		if !strings.Contains(all, r.Name) {
			t.Errorf("the confirmation does not name %s:\n%s", r.Name, all)
		}
	}
	if !strings.Contains(all, "Objects (3)") || !strings.Contains(all, "Delete 3 objects: ") {
		t.Errorf("the confirmation's count:\n%s", all)
	}
	test.Tap(sh.confirmer.current.cancel)
	q.until(t, "the dialog gone", func() bool { return sh.confirmer.current == nil })

	// Delete is greyed with its reason when it cannot run; Clear Selection empties the selection.
	f.disabled["object.deleteMany"] = "not permitted: you may not delete pods in default"
	m = openMenu(t, sh, q, 1)
	if d := item(t, m, "Delete"); !d.Disabled || !strings.Contains(d.Label, "not permitted") {
		t.Fatalf("a denied Delete: %q disabled %v", d.Label, d.Disabled)
	}
	item(t, m, "Clear Selection").Action()
	q.until(t, "the selection cleared", func() bool { return len(sh.ui.Selected) == 0 })
}

// TestSelectionBarDeletes: the header's Delete sends the whole selection.
func TestSelectionBarDeletes(t *testing.T) {
	f := connectedFake(t, 20)
	withDeleteMany(t, f)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil })
	rows := f.tables["pods"].Rows
	f.update(func(u *engine.UIState) { u.Selected = []model.ObjectRef{rows[0].Ref, rows[9].Ref} })
	q.until(t, "two selected", func() bool { return len(sh.ui.Selected) == 2 })
	settle(w)
	test.Tap(sh.view.check.bar.del)
	q.until(t, "the confirmation", func() bool { return sh.confirmer.current != nil })
	if call := f.callsOf("object.deleteMany"); len(call) != 1 || len(decodeParams[selectionRefs](t, call[0]).Refs) != 2 {
		t.Fatalf("object.deleteMany %+v", call)
	}
	test.Tap(sh.confirmer.current.confirm)
	q.until(t, "the dialog gone", func() bool { return sh.confirmer.current == nil })
}

// renderSelection checks three pods, opens the selection's menu on one of them.
func renderSelection(t *testing.T, mode ThemeMode, name string) {
	f := kindsFake(t, "pods")
	withDeleteMany(t, f)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the pods view", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil && sh.diag.total > 0 })
	settle(w)
	for _, row := range []int{1, 2, 4} {
		test.Tap(cellAt(t, sh, row, checkColumn))
		q.until(t, "the row checked", func() bool { return sh.view.check.has(sh.view.snap.Rows[row].Ref) })
	}
	settle(w)
	c := cellAt(t, sh, 2, firstDataColumn)
	test.TapSecondaryAt(c, fyne.NewPos(60, c.Size().Height/2))
	q.until(t, "the selection menu", func() bool { return sh.view.menu != nil })
	settle(w)
	savePNG(t, w, name)
}

// renderClickKeepsChecks checks three pods, then clicks a fourth (A73): its drawer opens, the three
// stay checked, and the clicked row is highlighted with its checkbox empty.
func renderClickKeepsChecks(t *testing.T, mode ThemeMode, name string) {
	f := kindsFake(t, "pods")
	withDeleteMany(t, f)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the pods view", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil && sh.diag.total > 0 })
	settle(w)
	for _, row := range []int{1, 2, 4} {
		test.Tap(cellAt(t, sh, row, checkColumn))
		q.until(t, "the row checked", func() bool { return sh.view.check.has(sh.view.snap.Rows[row].Ref) })
	}
	clicked := sh.view.snap.Rows[6].Ref
	test.Tap(cellAt(t, sh, 6, firstDataColumn+1))
	q.until(t, "the drawer", func() bool { return sh.drawer.shown != nil && *sh.drawer.shown == clicked })
	if len(sh.ui.Selected) != 3 {
		t.Fatalf("the click left %d rows checked, want 3", len(sh.ui.Selected))
	}
	settle(w)
	savePNG(t, w, name)
}

func TestRenderClickKeepsTheCheckedRows(t *testing.T) {
	renderClickKeepsChecks(t, ThemeDark, "selection_click_dark.png")
	renderClickKeepsChecks(t, ThemeLight, "selection_click_light.png")
}

func TestRenderSelectionDark(t *testing.T) { renderSelection(t, ThemeDark, "selection_menu_dark.png") }
func TestRenderSelectionLight(t *testing.T) {
	renderSelection(t, ThemeLight, "selection_menu_light.png")
}

// renderDeleteSelection shows the confirmation of Delete for twelve pods: the count, the first ten
// named in the message, every pod in the scrolling list.
func renderDeleteSelection(t *testing.T, mode ThemeMode, name string) {
	f := kindsFake(t, "pods")
	withDeleteMany(t, f)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the pods view", func() bool { return len(sh.kinds) > 0 && sh.view.snap != nil && sh.diag.total > 0 })
	rows := f.tables["pods"].Rows
	var sel []model.ObjectRef
	for _, r := range rows[:12] {
		sel = append(sel, r.Ref)
	}
	f.update(func(u *engine.UIState) { u.Selected = sel })
	q.until(t, "twelve selected", func() bool { return len(sh.ui.Selected) == 12 })
	settle(w)
	test.Tap(sh.view.check.bar.del)
	q.until(t, "the confirmation", func() bool { return sh.confirmer.current != nil })
	settle(w)
	savePNG(t, w, name)
	test.Tap(sh.confirmer.current.cancel)
}

func TestRenderDeleteSelectionDark(t *testing.T) {
	renderDeleteSelection(t, ThemeDark, "delete_selection_dark.png")
}

func TestRenderDeleteSelectionLight(t *testing.T) {
	renderDeleteSelection(t, ThemeLight, "delete_selection_light.png")
}
