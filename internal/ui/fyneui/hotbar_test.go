package fyneui

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// fakeHotbar gives a fake engine the hotbar.* commands. Changes run through the engine's own bank
// operations (engine.HotbarState) on f.ui.Hotbar, as Session.ChangeHotbar does; hotbar.list adds
// what each tile shows, as Session.HotbarView does, with missing and exec set by the test.
type fakeHotbar struct {
	f       *fakeEngine
	missing map[string]bool
	exec    map[string]string
}

func withHotbar(f *fakeEngine, current int, banks ...engine.HotbarBank) *fakeHotbar {
	fh := &fakeHotbar{f: f, missing: map[string]bool{}, exec: map[string]string{}}
	for i := range banks {
		if banks[i].Clusters == nil {
			banks[i].Clusters = []engine.HotbarTile{}
		}
	}
	f.ui.Hotbar = &engine.HotbarState{Banks: banks, Current: current}
	change := func(op func(h *engine.HotbarState, inv engine.Invocation) error) func(context.Context, engine.Invocation) (any, error) {
		return func(_ context.Context, inv engine.Invocation) (any, error) {
			f.mu.Lock()
			next := f.ui.Hotbar.Clone()
			f.mu.Unlock()
			if err := op(&next, inv); err != nil {
				return nil, err
			}
			f.update(func(u *engine.UIState) { u.Hotbar = &next })
			return map[string]any{"hotbar": fh.view()}, nil
		}
	}
	params := func(inv engine.Invocation, p any) error { return json.Unmarshal(normalize(inv.Params), p) }
	f.handlers["hotbar.list"] = func(context.Context, engine.Invocation) (any, error) {
		return map[string]any{"hotbar": fh.view()}, nil
	}
	f.handlers["hotbar.addCluster"] = change(func(h *engine.HotbarState, inv engine.Invocation) error {
		var p hotbarClusterParams
		if err := params(inv, &p); err != nil {
			return err
		}
		if p.Context == "" {
			p.Context = f.UIState().Context
		}
		return h.AddCluster(p.Bank, p.Context)
	})
	f.handlers["hotbar.removeCluster"] = change(func(h *engine.HotbarState, inv engine.Invocation) error {
		var p hotbarClusterParams
		if err := params(inv, &p); err != nil {
			return err
		}
		return h.RemoveCluster(p.Bank, p.Context)
	})
	f.handlers["hotbar.moveCluster"] = change(func(h *engine.HotbarState, inv engine.Invocation) error {
		var p hotbarMoveParams
		if err := params(inv, &p); err != nil {
			return err
		}
		return h.MoveCluster(p.Bank, p.Context, p.Position)
	})
	f.handlers["hotbar.switchBank"] = change(func(h *engine.HotbarState, inv engine.Invocation) error {
		var p hotbarSwitchParams
		if err := params(inv, &p); err != nil {
			return err
		}
		return h.SwitchBank(p.Bank)
	})
	f.handlers["hotbar.createBank"] = change(func(h *engine.HotbarState, inv engine.Invocation) error {
		var p hotbarCreateParams
		if err := params(inv, &p); err != nil {
			return err
		}
		_, err := h.CreateBank(p.Name)
		return err
	})
	f.handlers["hotbar.renameBank"] = change(func(h *engine.HotbarState, inv engine.Invocation) error {
		var p hotbarRenameParams
		if err := params(inv, &p); err != nil {
			return err
		}
		return h.RenameBank(p.Bank, p.Name)
	})
	f.handlers["hotbar.removeBank"] = change(func(h *engine.HotbarState, inv engine.Invocation) error {
		var p hotbarBankParams
		if err := params(inv, &p); err != nil {
			return err
		}
		_, err := h.RemoveBank(p.Bank)
		return err
	})
	f.handlers["hotbar.openCluster"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p hotbarSlotParams
		if err := params(inv, &p); err != nil {
			return nil, err
		}
		f.mu.Lock()
		t, err := f.ui.Hotbar.Tile(p.Bank, p.Slot)
		f.mu.Unlock()
		if err != nil {
			return nil, err
		}
		f.update(func(u *engine.UIState) {
			u.Context, u.Connection, u.View = t.Context, engine.Connection{State: engine.Connected}, "pods"
		})
		return map[string]any{"context": t.Context, "state": engine.Connected}, nil
	}
	return fh
}

// view is the hotbar as hotbar.list returns it.
func (fh *fakeHotbar) view() engine.HotbarState {
	u := fh.f.UIState()
	v := u.Hotbar.Clone()
	for i := range v.Banks {
		for j := range v.Banks[i].Clusters {
			t := &v.Banks[i].Clusters[j]
			t.Label, t.Connection = engine.ContextLabel(t.Context), engine.Disconnected
			if t.Context == u.Context {
				t.Connection = u.Connection.State
			}
			t.Missing, t.Exec = fh.missing[t.Context], fh.exec[t.Context]
		}
	}
	return v
}

func tilesOf(h *hotbar) []*hotbarTile { return findAll[*hotbarTile](h.tiles) }

// hotbarScene is the shell connected to kind-cogpit with a hotbar of two banks: Default with five
// clusters (kind-cogpit connected, legacy-onprem missing), Production with one.
func hotbarScene(t *testing.T, mode ThemeMode) (*shell, fyne.Window, *mainQueue, *fakeEngine, *fakeHotbar) {
	f := connectedFake(t, 10000)
	f.ui.Context = "kind-cogpit"
	f.ui.Connection = engine.Connection{State: engine.Connected}
	tiles := func(ids ...string) []engine.HotbarTile {
		out := make([]engine.HotbarTile, 0, len(ids))
		for _, id := range ids {
			out = append(out, engine.HotbarTile{Context: id})
		}
		return out
	}
	fh := withHotbar(f, 1,
		engine.HotbarBank{Name: "Default", Clusters: tiles("kind-cogpit", "prod-eu-central", "staging-us-east", "legacy-onprem [.kube/old.yaml]", "dev-hfi", "demo")},
		engine.HotbarBank{Name: "Production", Clusters: tiles("prod-eu-central")})
	fh.missing["legacy-onprem [.kube/old.yaml]"] = true
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the hotbar", func() bool { return sh.hot.root.Visible() && len(tilesOf(sh.hot)) == 6 && len(sh.kinds) > 0 })
	return sh, w, q, f, fh
}

// Without a hotbar in the engine the bar is hidden and nothing asks for one: the shell looks as it
// did before the hotbar.
func TestHotbarHiddenWithoutAHotbar(t *testing.T) {
	f := connectedFake(t, 10)
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	if sh.hot.root.Visible() || len(f.callsOf("hotbar.list")) != 0 {
		t.Errorf("visible %v, hotbar.list called %d times", sh.hot.root.Visible(), len(f.callsOf("hotbar.list")))
	}
	if sh.hot.contextItem("demo") != nil {
		t.Error("a hotbar item without a hotbar")
	}
}

// The bar shows the shown bank: a tile per cluster with its initials and label, the connected one
// active with a green dot, the missing one marked, the full id with state and key as tooltip, and
// the switcher's number with the bank's name as its tooltip.
func TestHotbarShowsTheShownBank(t *testing.T) {
	sh, _, q, f, _ := hotbarScene(t, ThemeDark)
	tl := tilesOf(sh.hot)
	if tl[0].tile.Label != "kind-cogpit" || !tl[0].active() || tl[1].active() || tl[0].slot != 1 {
		t.Errorf("first tiles: %+v %+v", tl[0].tile, tl[1].tile)
	}
	if !tl[3].tile.Missing || tl[3].tile.Label != "legacy-onprem" {
		t.Errorf("the missing tile: %+v", tl[3].tile)
	}
	if got := tl[3].tipText(); got != "legacy-onprem [.kube/old.yaml]\nNo kubeconfig source offers this context any more · F4" {
		t.Errorf("tooltip %q", got)
	}
	if got := tl[0].tipText(); got != "kind-cogpit\nConnected · F1" {
		t.Errorf("tooltip %q", got)
	}
	r := test.WidgetRenderer(tl[0]).(*hotbarTileRenderer)
	if r.initials.Text != "KC" {
		t.Errorf("initials %q", r.initials.Text)
	}
	if sh.hot.number.text != "1" || sh.hot.number.tip != "Bank 1 of 2: Default" || sh.hot.prev.disabled {
		t.Errorf("switcher %q %q", sh.hot.number.text, sh.hot.number.tip)
	}
	// The "+" pins the connected cluster; it is in the bank already.
	if !sh.hot.add.disabled || sh.hot.add.tip != "kind-cogpit is in this bank already" {
		t.Errorf("+: %v %q", sh.hot.add.disabled, sh.hot.add.tip)
	}
	// Another bank shown by an agent: the bar follows the UI state.
	f.update(func(u *engine.UIState) { h := u.Hotbar.Clone(); h.Current = 2; u.Hotbar = &h })
	q.until(t, "bank 2", func() bool { return len(tilesOf(sh.hot)) == 1 && sh.hot.number.text == "2" })
	if sh.hot.add.disabled || sh.hot.add.tip != "Add kind-cogpit to bank 2 (Production)" {
		t.Errorf("+ on bank 2: %v %q", sh.hot.add.disabled, sh.hot.add.tip)
	}
	test.Tap(sh.hot.add)
	q.until(t, "the cluster pinned", func() bool { return len(tilesOf(sh.hot)) == 2 })
	if p := decodeParams[hotbarClusterParams](t, f.callsOf("hotbar.addCluster")[0]); p.Bank != 2 || p.Context != "" {
		t.Errorf("hotbar.addCluster %+v", p)
	}
}

// A click opens the tile's cluster through hotbar.openCluster, and the bar shows the new connection;
// the connected cluster's tile and a missing one dispatch nothing (the missing one says why).
func TestHotbarTileClickOpensTheCluster(t *testing.T) {
	sh, _, q, f, _ := hotbarScene(t, ThemeDark)
	test.Tap(tilesOf(sh.hot)[0]) // connected already
	test.Tap(tilesOf(sh.hot)[3]) // missing
	q.drain()
	if n := len(f.callsOf("hotbar.openCluster")); n != 0 {
		t.Errorf("hotbar.openCluster dispatched %d times", n)
	}
	if !slices.ContainsFunc(sh.toasts.Objects, func(o fyne.CanvasObject) bool { return o != nil }) {
		t.Error("no toast for the missing tile")
	}
	test.Tap(tilesOf(sh.hot)[1])
	q.until(t, "the connection", func() bool { return sh.ui.Context == "prod-eu-central" && tilesOf(sh.hot)[1].active() })
	p := decodeParams[hotbarSlotParams](t, f.callsOf("hotbar.openCluster")[0])
	if p.Slot != 2 || p.Bank != 1 {
		t.Errorf("hotbar.openCluster %+v", p)
	}
	if tilesOf(sh.hot)[0].active() {
		t.Error("the old cluster's tile is still active")
	}
}

// A tile whose login program waits for the user's approval shows it first, as the section does:
// nothing connects before the user allowed it.
func TestHotbarTileAsksForTheLoginProgramFirst(t *testing.T) {
	f := connectedFake(t, 10)
	fh := withHotbar(f, 1, engine.HotbarBank{Name: "Default", Clusters: []engine.HotbarTile{{Context: "eks [eks.yaml]"}}})
	fh.exec["eks [eks.yaml]"] = engine.ExecRequired
	f.handlers["kubeconfig.pendingApprovals"] = func(context.Context, engine.Invocation) (any, error) {
		return nil, errors.New("no plugin in this fake")
	}
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the hotbar", func() bool { return len(tilesOf(sh.hot)) == 1 })
	test.Tap(tilesOf(sh.hot)[0])
	q.until(t, "the approval asked", func() bool { return len(f.callsOf("kubeconfig.pendingApprovals")) == 1 })
	q.drain()
	if n := len(f.callsOf("hotbar.openCluster")); n != 0 {
		t.Errorf("connected before the approval: %d", n)
	}
}

// The tile's menu: Connect, Move Up (not for the first), Move Down (not for the last), Remove from
// Hotbar, each through its command.
func TestHotbarTileMenu(t *testing.T) {
	sh, _, q, f, _ := hotbarScene(t, ThemeDark)
	tl := tilesOf(sh.hot)
	tl[0].TappedSecondary(&fyne.PointEvent{AbsolutePosition: fyne.NewPos(30, 100)})
	items := sh.hot.menuModel.Items
	labels := []string{}
	for _, it := range items {
		labels = append(labels, it.Label)
	}
	if strings.Join(labels, ",") != "Connect,,Move Up,Move Down,,Remove from Hotbar" {
		t.Fatalf("menu %v", labels)
	}
	if !items[0].Disabled || !items[2].Disabled || items[3].Disabled {
		t.Errorf("connected tile: Connect %v, Move Up %v, Move Down %v", items[0].Disabled, items[2].Disabled, items[3].Disabled)
	}
	items[3].Action() // Move Down
	q.until(t, "the move", func() bool { return tilesOf(sh.hot)[1].tile.Context == "kind-cogpit" })
	if p := decodeParams[hotbarMoveParams](t, f.callsOf("hotbar.moveCluster")[0]); p.Context != "kind-cogpit" || p.Position != 2 || p.Bank != 1 {
		t.Errorf("hotbar.moveCluster %+v", p)
	}
	last := tilesOf(sh.hot)[5]
	last.TappedSecondary(&fyne.PointEvent{})
	if !sh.hot.menuModel.Items[3].Disabled {
		t.Error("Move Down on the last tile")
	}
	sh.hot.menuModel.Items[5].Action() // Remove from Hotbar
	q.until(t, "the removal", func() bool { return len(tilesOf(sh.hot)) == 5 })
	if p := decodeParams[hotbarClusterParams](t, f.callsOf("hotbar.removeCluster")[0]); p.Context != "demo" || p.Bank != 1 {
		t.Errorf("hotbar.removeCluster %+v", p)
	}
}

// F1–F12 open a tile of the shown bank and ⇧F1–⇧F12 show a bank: from the window when nothing has
// the focus, and from the table when it has; a key without a tile or bank does nothing.
func TestHotbarFunctionKeys(t *testing.T) {
	sh, w, q, f, _ := hotbarScene(t, ThemeDark)
	w.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyF3})
	q.until(t, "F3", func() bool { return sh.ui.Context == "staging-us-east" })
	sh.view.table.TypedKey(&fyne.KeyEvent{Name: fyne.KeyF2})
	q.until(t, "F2 from the table", func() bool { return sh.ui.Context == "prod-eu-central" })
	if !sh.hot.key(fyne.KeyF12, false) || !sh.hot.key(fyne.KeyF9, true) {
		t.Error("an F-key without a tile or bank was passed on")
	}
	if sh.hot.key(fyne.KeyA, false) {
		t.Error("a letter taken by the hotbar")
	}
	sh.hot.key(fyne.KeyF2, true)
	q.until(t, "⇧F2", func() bool { return sh.hot.number.text == "2" })
	q.drain()
	if n := len(f.callsOf("hotbar.openCluster")); n != 2 {
		t.Errorf("%d opens, want 2 (F12 has no tile)", n)
	}
	if p := decodeParams[hotbarSwitchParams](t, f.callsOf("hotbar.switchBank")[0]); p.Bank != 2 || len(f.callsOf("hotbar.switchBank")) != 1 {
		t.Errorf("hotbar.switchBank %+v (⇧F9 has no bank)", p)
	}
}

// The switcher: the arrows step through the banks and wrap around; the number's menu lists them
// and creates, renames and removes; removing a bank with clusters asks first.
func TestHotbarBankSwitcher(t *testing.T) {
	sh, w, q, f, _ := hotbarScene(t, ThemeDark)
	test.Tap(sh.hot.prev) // from bank 1 to the last, 2
	q.until(t, "bank 2", func() bool { return sh.hot.number.text == "2" })
	test.Tap(sh.hot.next) // and around to 1
	q.until(t, "bank 1", func() bool { return sh.hot.number.text == "1" })

	test.Tap(sh.hot.number)
	labels := []string{}
	for _, it := range sh.hot.menuModel.Items {
		labels = append(labels, it.Label)
	}
	if strings.Join(labels, "|") != "1  Default|2  Production||New Bank…|Rename Bank…|Remove Bank…" || !sh.hot.menuModel.Items[0].Checked {
		t.Fatalf("bank menu %q", labels)
	}
	sh.hot.menuModel.Items[3].Action() // New Bank…
	entry := findAll[*widget.Entry](w.Canvas().Overlays().Top())
	if len(entry) != 1 || entry[0].PlaceHolder != "Bank 3" {
		t.Fatalf("the name dialog: %d entries", len(entry))
	}
	test.Type(entry[0], "Staging")
	test.Tap(findButton(w, "Create"))
	q.until(t, "bank 3", func() bool { return sh.hot.number.text == "3" && sh.hot.number.tip == "Bank 3 of 3: Staging" })

	sh.hot.renameBank()
	entry = findAll[*widget.Entry](w.Canvas().Overlays().Top())
	entry[0].SetText("QA")
	test.Tap(findButton(w, "Rename"))
	q.until(t, "the rename", func() bool { return sh.hot.number.tip == "Bank 3 of 3: QA" })

	sh.hot.removeBank() // empty: no question
	q.until(t, "bank 3 gone", func() bool { return len(sh.hot.view.Banks) == 2 })
	sh.hot.switchBank(1)
	q.until(t, "bank 1", func() bool { return sh.hot.number.text == "1" })
	sh.hot.removeBank() // six clusters: asks
	if findButton(w, "Remove") == nil {
		t.Fatal("no confirmation for a bank with clusters")
	}
	test.Tap(findButton(w, "Cancel"))
	q.drain()
	if n := len(f.callsOf("hotbar.removeBank")); n != 1 {
		t.Fatalf("Cancel removed the bank: %d removals", n)
	}
	sh.hot.removeBank()
	test.Tap(findButton(w, "Remove"))
	q.until(t, "bank 1 gone", func() bool { return len(sh.hot.view.Banks) == 1 && sh.hot.view.Banks[0].Name == "Production" })
	if !sh.hot.prev.disabled {
		t.Error("the arrows stay enabled with one bank")
	}
	if title, msg := removeBankText(1, engine.HotbarBank{Name: "Default", Clusters: make([]engine.HotbarTile, 6)}); title != "Remove bank 1?" ||
		!strings.Contains(msg, `"Default" and its 6 tiles`) || !strings.Contains(msg, "stay as they are") {
		t.Errorf("confirmation %q %q", title, msg)
	}
}

// "Add to Hotbar" is in a context's menu: the navigator's section rows (right-click) and the
// Cluster menu, as "Remove from Hotbar" when the shown bank holds the context.
func TestHotbarAddFromTheContextMenus(t *testing.T) {
	sh, _, q, f, _ := hotbarScene(t, ThemeDark)
	cluster := sh.mainMenu().Items[0]
	if cluster.Label != "Cluster" || cluster.Items[1].Label != "Remove from Hotbar" {
		t.Errorf("Cluster menu: %q %q", cluster.Label, cluster.Items[1].Label)
	}

	n := sh.side.sec
	q.until(t, "the section's rows", func() bool { return n.loaded && len(n.rows) == 5 })
	n.rows["demo"].TappedSecondary(&fyne.PointEvent{AbsolutePosition: fyne.NewPos(400, 300)}) // in the bank already
	if got := n.menuModel.Items[1].Label; got != "Remove from Hotbar" {
		t.Errorf("demo's menu: %q", got)
	}
	f.update(func(u *engine.UIState) { h := u.Hotbar.Clone(); h.Current = 2; u.Hotbar = &h })
	q.until(t, "bank 2", func() bool { return sh.hot.number.text == "2" })
	n.rows["staging-us-east"].TappedSecondary(&fyne.PointEvent{AbsolutePosition: fyne.NewPos(400, 300)})
	items := n.menuModel.Items
	if items[0].Label != "Connect" || items[1].Label != "Add to Hotbar" {
		t.Fatalf("staging's menu: %q %q", items[0].Label, items[1].Label)
	}
	items[1].Action()
	q.until(t, "pinned", func() bool { return len(tilesOf(sh.hot)) == 2 })
	if p := decodeParams[hotbarClusterParams](t, f.callsOf("hotbar.addCluster")[0]); p.Context != "staging-us-east" || p.Bank != 2 {
		t.Errorf("hotbar.addCluster %+v", p)
	}
	// The Cluster menu follows: kind-cogpit is not in bank 2.
	if got := sh.mainMenu().Items[0].Items[1].Label; got != "Add to Hotbar" {
		t.Errorf("Cluster menu on bank 2: %q", got)
	}
}

// The parameters the bar sends are what the real registry accepts (it refuses unknown fields).
func TestHotbarParamsMatchTheRegistry(t *testing.T) {
	s := engine.New(engine.Options{Hotbar: engine.NewHotbar(t.TempDir())})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	for id, params := range map[string]any{
		"hotbar.list":          struct{}{},
		"hotbar.addCluster":    hotbarClusterParams{Context: "a", Bank: 1},
		"hotbar.removeCluster": hotbarClusterParams{Context: "a", Bank: 1},
		"hotbar.moveCluster":   hotbarMoveParams{Context: "a", Position: 2, Bank: 1},
		"hotbar.openCluster":   hotbarSlotParams{Slot: 1, Bank: 1},
		"hotbar.switchBank":    hotbarSwitchParams{Bank: 1},
		"hotbar.createBank":    hotbarCreateParams{Name: "x"},
		"hotbar.renameBank":    hotbarRenameParams{Bank: 1, Name: "x"},
		"hotbar.removeBank":    hotbarBankParams{Bank: 1},
	} {
		cmd, ok := s.Registry().Get(id)
		if !ok {
			t.Errorf("no command %s", id)
			continue
		}
		raw, _ := json.Marshal(params)
		var pe *engine.ParamsError
		if err := cmd.Enabled(s, raw); errors.As(err, &pe) {
			t.Errorf("%s %s: %v", id, raw, err)
		}
	}
}

func TestRenderHotbarEmptyDark(t *testing.T) {
	renderHotbarEmpty(t, ThemeDark, "hotbar_empty_dark.png")
}
func TestRenderHotbarEmptyLight(t *testing.T) {
	renderHotbarEmpty(t, ThemeLight, "hotbar_empty_light.png")
}

// renderHotbarEmpty: a first start's hotbar, one empty bank, the "+" to pin the connected cluster.
func renderHotbarEmpty(t *testing.T, mode ThemeMode, name string) {
	f := connectedFake(t, 10000)
	withHotbar(f, 1, engine.HotbarBank{Name: engine.DefaultHotbarBank})
	sh, w, q := newTestShell(t, f, Options{Theme: mode, MCPCommand: "claude mcp add cogpit -- cogpit-cli mcp"})
	q.until(t, "the hotbar", func() bool { return sh.hot.root.Visible() && len(sh.kinds) > 0 && sh.diag.total > 0 })
	savePNG(t, w, name)
}

func TestRenderHotbarBankDark(t *testing.T) { renderHotbarBank(t, ThemeDark, "hotbar_bank_dark.png") }
func TestRenderHotbarBankLight(t *testing.T) {
	renderHotbarBank(t, ThemeLight, "hotbar_bank_light.png")
}

// renderHotbarBank: six clusters in bank 1 of 2, kind-cogpit connected, legacy-onprem missing with
// its tooltip shown.
func renderHotbarBank(t *testing.T, mode ThemeMode, name string) {
	sh, w, q, _, _ := hotbarScene(t, mode)
	q.until(t, "diagnostics", func() bool { return sh.diag.total > 0 })
	settle(w)
	missing := tilesOf(sh.hot)[3]
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(missing).AddXY(missing.Size().Width/2, 20)
	sh.tip.delay = time.Millisecond
	sh.tip.hover(missing, missing.tipText(), pos)
	q.until(t, "the tooltip", func() bool { return sh.tip.visibleText() != "" })
	savePNG(t, w, name)
}

// TestRenderHotbarBankMenuDark: the switcher's menu, with the banks to switch to and the bank
// commands.
func TestRenderHotbarBankMenuDark(t *testing.T) {
	sh, w, q, _, _ := hotbarScene(t, ThemeDark)
	q.until(t, "diagnostics", func() bool { return sh.diag.total > 0 })
	settle(w)
	test.Tap(sh.hot.number)
	savePNG(t, w, "hotbar_bank_menu_dark.png")
}
