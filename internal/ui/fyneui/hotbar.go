package fyneui

import (
	"fmt"
	"image/color"
	"reflect"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The hotbar (G13; engine/hotbar.go): a narrow bar at the window's left edge, left of the
// navigator, with one tile per pinned cluster of the shown bank and the bank switcher "‹ 1 ›" at
// the bottom, as captured from Lens (2026-10-11). A tile is the cluster's identity
// (clusteridentity.go: its initials in its icon colour, the accent by default, or its uploaded
// image), its name under it (the display name its Settings give it, else the context's) and a dot
// for the connection's state; its tooltip is the full context id. A click opens the cluster (hotbar.openCluster); the
// tile's menu (right-click) connects, moves it up or down and removes it; a tile dragged within the
// bar moves to the gap it is dropped into (R2, hotbardrag.go); the switcher's arrows and its number
// (a menu of the banks) switch, create, rename and remove banks. F1–F12 open a tile of the shown
// bank, ⇧F1–⇧F12 show a bank. Inferred, Cogpit's own: the "+" below the tiles that pins the
// connected cluster, the menus' items, the confirmation before removing a bank that holds clusters,
// and that the arrows wrap around. Move Up and Move Down stay beside the drag, for whoever does not
// drag.
//
// The bar shows what hotbar.list returns, read again whenever UIState.Hotbar, the connection or
// the contexts change; every change goes through the hotbar.* commands (UI state is engine state).
// Without a hotbar in the engine (UIState.Hotbar nil) the bar is hidden.
type hotbar struct {
	sh *shell

	view *engine.HotbarState // the last hotbar.list; nil while there is none
	seq  int                 // the latest hotbar.list request; older answers are dropped

	tiles  *fyne.Container   // the shown bank's tiles and the "+"
	scroll *container.Scroll // the tiles' area: a tile dropped outside it does not move
	drag   *tileDrag         // a tile being dragged to another place (hotbardrag.go)
	add    *hotbarButton
	prev   *hotbarButton
	number *hotbarButton
	next   *hotbarButton
	root   *fyne.Container

	menuModel *fyne.Menu // the open tile or bank menu, for tests
	menuFor   string     // what the Cluster menu's hotbar item last offered, to rebuild it when that changes
}

func newHotbar(sh *shell) *hotbar {
	h := &hotbar{sh: sh}
	h.tiles = container.NewVBox()
	h.add = newHotbarButton(h, "+", theme.SizeNameHeadingText, h.addConnected)
	h.add.square = true
	h.prev = newHotbarButton(h, "‹", theme.SizeNameSubHeadingText, func() { h.step(-1) })
	h.next = newHotbarButton(h, "›", theme.SizeNameSubHeadingText, func() { h.step(1) })
	h.number = newHotbarButton(h, "1", theme.SizeNameText, nil)
	h.number.bold = true
	h.number.onTap = func() { h.bankMenu(h.number.menuPos()) } // above the switcher (showMenu)
	h.number.onSecondary = h.bankMenu
	h.prev.tip, h.next.tip = "Previous bank", "Next bank"
	switcher := container.NewVBox(newDivider(false),
		container.NewCenter(container.New(layout.NewCustomPaddedHBoxLayout(0), h.prev, h.number, h.next)))
	h.drag = newTileDrag(h)
	h.scroll = container.NewVScroll(inset("", theme.SizeNamePadding, container.NewStack(h.tiles, h.drag.layer)))
	content := container.NewBorder(nil, inset("", theme.SizeNamePadding, switcher), nil, nil, h.scroll)
	h.root = sized(sizeHotbarWidth, "", newSurface(theme.ColorNameHeaderBackground),
		container.NewBorder(nil, nil, nil, newDivider(true), content))
	h.root.Hide()
	return h
}

// stateChanged follows the UI state: hidden without a hotbar, else read again.
func (h *hotbar) stateChanged() {
	if h.sh.ui.Hotbar == nil {
		h.seq++
		h.setView(nil)
		return
	}
	h.request()
}

// request reads the hotbar as the user sees it (hotbar.list) and shows it. Main goroutine.
func (h *hotbar) request() {
	if h.sh.ui.Hotbar == nil {
		return
	}
	h.seq++
	seq := h.seq
	h.sh.dispatch("hotbar.list", struct{}{}, func(res any, err error) {
		if seq != h.seq || h.sh.ui.Hotbar == nil {
			return
		}
		var out struct {
			Hotbar engine.HotbarState `json:"hotbar"`
		}
		if err != nil || decode(res, &out) != nil || len(out.Hotbar.Banks) == 0 {
			return // the bar keeps what it shows; the next change reads it again
		}
		h.setView(&out.Hotbar)
	})
}

// bank returns the shown bank; false while there is none.
func (h *hotbar) bank() (engine.HotbarBank, bool) {
	if h.view == nil || h.view.Current < 1 || h.view.Current > len(h.view.Banks) {
		return engine.HotbarBank{}, false
	}
	return h.view.Banks[h.view.Current-1], true
}

// setView shows a hotbar, or hides the bar for nil.
func (h *hotbar) setView(v *engine.HotbarState) {
	h.view = v
	b, ok := h.bank()
	if !ok {
		h.root.Hide()
		h.syncMenu()
		return
	}
	if _, ok := h.sh.tip.owner.(*hotbarTile); ok {
		h.sh.tip.hide() // its tile is replaced, and would never say the pointer left
	}
	objs := make([]fyne.CanvasObject, 0, len(b.Clusters)+1)
	for i, t := range b.Clusters {
		objs = append(objs, newHotbarTile(h, t, i+1))
	}
	if len(b.Clusters) < engine.HotbarBankSize {
		objs = append(objs, h.add)
	}
	h.tiles.Objects = objs
	h.tiles.Refresh()
	h.drag.follow()
	h.updateAdd(b)
	h.number.setText(strconv.Itoa(h.view.Current))
	h.number.tip = fmt.Sprintf("Bank %d of %d: %s", h.view.Current, len(h.view.Banks), b.Name)
	many := len(h.view.Banks) > 1
	h.prev.setDisabled(!many)
	h.next.setDisabled(!many)
	h.root.Show()
	h.root.Refresh()
	h.syncMenu()
}

// updateAdd says what the "+" would pin, or why it cannot.
func (h *hotbar) updateAdd(b engine.HotbarBank) {
	ui := h.sh.ui
	switch {
	case ui.Context == "":
		h.add.tip, h.add.disabled = "Connect a cluster to pin it to this bank", true
	case b.Position(ui.Context) > 0:
		h.add.tip, h.add.disabled = engine.ContextLabel(ui.Context)+" is in this bank already", true
	default:
		h.add.tip, h.add.disabled = "Add "+engine.ContextLabel(ui.Context)+" to bank "+strconv.Itoa(h.view.Current)+" ("+b.Name+")", false
	}
	h.add.Refresh()
}

// addConnected pins the connected cluster to the shown bank.
func (h *hotbar) addConnected() {
	if h.view == nil || h.add.disabled {
		return
	}
	h.sh.dispatch("hotbar.addCluster", hotbarClusterParams{Bank: h.view.Current}, nil)
}

// step shows the previous or next bank, wrapping around (inferred).
func (h *hotbar) step(d int) {
	if h.view == nil || len(h.view.Banks) < 2 {
		return
	}
	n := len(h.view.Banks)
	h.switchBank((h.view.Current-1+d+n)%n + 1)
}

func (h *hotbar) switchBank(n int) {
	if h.view == nil || n < 1 || n > len(h.view.Banks) || n == h.view.Current {
		return
	}
	h.sh.dispatch("hotbar.switchBank", hotbarSwitchParams{Bank: n}, nil)
}

// open opens the tile in a slot of the shown bank: a missing context is explained, a credential
// plugin waiting for the user's approval is shown first (as the navigator's section does), and the connected
// cluster's tile does nothing.
func (h *hotbar) open(slot int) {
	b, ok := h.bank()
	if !ok || slot < 1 || slot > len(b.Clusters) {
		return
	}
	t := b.Clusters[slot-1]
	if t.Missing {
		h.sh.toast("warning", t.Context+" is no longer offered by any kubeconfig source. Remove its tile, or sync its kubeconfig again.")
		return
	}
	if t.Context == h.sh.ui.Context && (t.Connection == engine.Connected || t.Connection == engine.Connecting) {
		return
	}
	bank := h.view.Current
	run := func() {
		h.sh.dispatch("hotbar.openCluster", hotbarSlotParams{Slot: slot, Bank: bank}, func(res any, err error) {
			if err != nil {
				h.sh.showError(err)
				return
			}
			var out struct {
				Context string `json:"context"`
				State   string `json:"state"`
				Error   string `json:"error"`
			}
			if decode(res, &out) == nil && out.State == string(engine.ConnectError) {
				h.sh.toast("error", "Could not connect to "+t.Label+": "+out.Error)
			}
		})
	}
	if t.Exec == engine.ExecRequired || t.Exec == engine.ExecChanged {
		h.sh.kc.reviewPlugin(t.Context, func(allowed bool) {
			if !allowed {
				h.sh.toast("warning", "Not connected: the login of "+t.Label+" was not allowed.")
				return
			}
			run()
		})
		return
	}
	run()
}

// key handles F1–F12 (open a tile of the shown bank) and ⇧F1–⇧F12 (show a bank). It reports
// whether the key was one of them; a key without a tile or bank does nothing.
func (h *hotbar) key(name fyne.KeyName, shift bool) bool {
	n := functionKey(name)
	if n == 0 {
		return false
	}
	if shift {
		h.switchBank(n)
	} else {
		h.open(n)
	}
	return true
}

// functionKey is n for the key Fn (1 to 12), else 0.
func functionKey(name fyne.KeyName) int {
	for i, k := range []fyne.KeyName{fyne.KeyF1, fyne.KeyF2, fyne.KeyF3, fyne.KeyF4, fyne.KeyF5, fyne.KeyF6,
		fyne.KeyF7, fyne.KeyF8, fyne.KeyF9, fyne.KeyF10, fyne.KeyF11, fyne.KeyF12} {
		if name == k {
			return i + 1
		}
	}
	return 0
}

// shiftHeld reports whether Shift is down. Fyne delivers F-keys without their modifiers: unmodified
// and shifted keys are no shortcuts to it, so they reach TypedKey; the desktop driver knows the
// modifiers held.
func shiftHeld() bool {
	if d, ok := fyne.CurrentApp().Driver().(desktop.Driver); ok {
		return d.CurrentKeyModifiers()&fyne.KeyModifierShift != 0
	}
	return false
}

// showMenu pops a menu up at pos (window coordinates); one that would reach into the status bar
// opens upwards from pos instead, as the switcher's menu at the bar's bottom does.
func (h *hotbar) showMenu(m *fyne.Menu, pos fyne.Position) {
	h.sh.tip.hide()
	h.menuModel = m
	pm := widget.NewPopUpMenu(m, h.sh.win.Canvas())
	room := h.sh.win.Canvas().Size().Height - h.sh.app.Settings().Theme().Size(sizeStatusHeight)
	if ms := pm.MinSize(); pos.Y+ms.Height > room && pos.Y > ms.Height {
		pos.Y -= ms.Height
	}
	pm.ShowAtPosition(pos)
}

// tileMenu is a tile's menu: Connect, Settings, Move Up, Move Down, Remove from Hotbar (Cogpit's,
// inferred).
func (h *hotbar) tileMenu(slot int, pos fyne.Position) {
	b, ok := h.bank()
	if !ok || slot < 1 || slot > len(b.Clusters) {
		return
	}
	t, bank := b.Clusters[slot-1], h.view.Current
	connect := fyne.NewMenuItem("Connect", func() { h.open(slot) })
	connect.Disabled = t.Missing || (t.Context == h.sh.ui.Context && t.Connection == engine.Connected)
	move := func(to int) func() {
		return func() {
			h.sh.dispatch("hotbar.moveCluster", hotbarMoveParams{Context: t.Context, Position: to, Bank: bank}, nil)
		}
	}
	up := fyne.NewMenuItem("Move Up", move(slot-1))
	up.Disabled = slot == 1
	down := fyne.NewMenuItem("Move Down", move(slot+1))
	down.Disabled = slot == len(b.Clusters)
	remove := fyne.NewMenuItem("Remove from Hotbar", func() {
		h.sh.dispatch("hotbar.removeCluster", hotbarClusterParams{Context: t.Context, Bank: bank}, nil)
	})
	items := []*fyne.MenuItem{connect}
	if settings := h.sh.settingsItem(t.Context); settings != nil { // the cluster's Settings page
		items = append(items, settings)
	}
	items = append(items, h.sh.clusterItems(t.Context)...) // Disconnect, Remove from Cogpit (menus_cluster.go)
	items = append(items, fyne.NewMenuItemSeparator(), up, down, fyne.NewMenuItemSeparator(), remove)
	h.showMenu(fyne.NewMenu("", items...), pos)
}

// bankMenu is the switcher's menu: the banks to switch to, then New, Rename and Remove Bank.
func (h *hotbar) bankMenu(pos fyne.Position) {
	if h.view == nil {
		return
	}
	var items []*fyne.MenuItem
	for i, b := range h.view.Banks {
		n := i + 1
		item := fyne.NewMenuItem(fmt.Sprintf("%d  %s", n, b.Name), func() { h.switchBank(n) })
		item.Checked = n == h.view.Current
		items = append(items, item)
	}
	create := fyne.NewMenuItem("New Bank…", h.createBank)
	create.Disabled = len(h.view.Banks) >= engine.HotbarMaxBanks
	rename := fyne.NewMenuItem("Rename Bank…", h.renameBank)
	remove := fyne.NewMenuItem("Remove Bank…", h.removeBank)
	remove.Disabled = len(h.view.Banks) < 2
	items = append(items, fyne.NewMenuItemSeparator(), create, rename, remove)
	h.showMenu(fyne.NewMenu("", items...), pos)
}

// nameDialog asks for a bank's name.
func (h *hotbar) nameDialog(title, confirm, name, placeholder string, done func(string)) {
	entry := widget.NewEntry()
	entry.SetText(name)
	entry.SetPlaceHolder(placeholder)
	d := dialog.NewForm(title, confirm, "Cancel", []*widget.FormItem{widget.NewFormItem("Name", entry)}, func(ok bool) {
		if ok {
			done(entry.Text)
		}
	}, h.sh.win)
	d.Resize(fyne.NewSize(h.sh.app.Settings().Theme().Size(sizeDialogWidth), 0))
	d.Show()
	h.sh.win.Canvas().Focus(entry)
}

func (h *hotbar) createBank() {
	if h.view == nil {
		return
	}
	h.nameDialog("New Bank", "Create", "", fmt.Sprintf("Bank %d", len(h.view.Banks)+1), func(name string) {
		h.sh.dispatch("hotbar.createBank", hotbarCreateParams{Name: name}, nil)
	})
}

func (h *hotbar) renameBank() {
	b, ok := h.bank()
	if !ok {
		return
	}
	bank := h.view.Current
	h.nameDialog(fmt.Sprintf("Rename Bank %d", bank), "Rename", b.Name, b.Name, func(name string) {
		h.sh.dispatch("hotbar.renameBank", hotbarRenameParams{Bank: bank, Name: name}, nil)
	})
}

// removeBank removes the shown bank; one that holds clusters asks first (Cogpit's, inferred).
func (h *hotbar) removeBank() {
	b, ok := h.bank()
	if !ok || len(h.view.Banks) < 2 {
		return
	}
	bank := h.view.Current
	run := func() { h.sh.dispatch("hotbar.removeBank", hotbarBankParams{Bank: bank}, nil) }
	if len(b.Clusters) == 0 {
		run()
		return
	}
	title, msg := removeBankText(bank, b)
	text := widget.NewLabel(msg)
	text.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm(title, "Remove", "Cancel", text, func(ok bool) {
		if ok {
			run()
		}
	}, h.sh.win)
	d.Resize(fyne.NewSize(h.sh.app.Settings().Theme().Size(sizeConfirmWidth), 0))
	d.Show()
}

// removeBankText is the confirmation of removing a bank that holds clusters: what goes and what
// stays.
func removeBankText(n int, b engine.HotbarBank) (title, msg string) {
	return fmt.Sprintf("Remove bank %d?", n), fmt.Sprintf("The bank %q and its %d %s leave the hotbar. The clusters, "+
		"their kubeconfigs and an open connection stay as they are.", b.Name, len(b.Clusters), nounFor(len(b.Clusters), "tile", "tiles"))
}

// contextItem is the hotbar's item in a context's menu (the navigator's and the welcome page's rows, the Cluster menu):
// "Add to Hotbar", or "Remove from Hotbar" when the shown bank holds it; nil without a hotbar.
func (h *hotbar) contextItem(id string) *fyne.MenuItem {
	b, ok := h.bank()
	if !ok || id == "" {
		return nil
	}
	bank := h.view.Current
	if b.Position(id) > 0 {
		return fyne.NewMenuItem("Remove from Hotbar", func() {
			h.sh.dispatch("hotbar.removeCluster", hotbarClusterParams{Context: id, Bank: bank}, nil)
		})
	}
	item := fyne.NewMenuItem("Add to Hotbar", func() {
		h.sh.dispatch("hotbar.addCluster", hotbarClusterParams{Context: id, Bank: bank}, nil)
	})
	item.Disabled = len(b.Clusters) >= engine.HotbarBankSize
	return item
}

// syncMenu rebuilds the window's menu when the Cluster menu's hotbar item would change.
func (h *hotbar) syncMenu() {
	state := ""
	if item := h.contextItem(h.sh.ui.Context); item != nil {
		state = item.Label + fmt.Sprint(item.Disabled)
	}
	if state != h.menuFor {
		h.menuFor = state
		h.sh.win.SetMainMenu(h.sh.mainMenu())
	}
}

// sameHotbar reports whether two UI states show the same hotbar.
func sameHotbar(a, b *engine.HotbarState) bool { return reflect.DeepEqual(a, b) }

// Command parameters, mirroring the engine's (commands/hotbar.go).
type (
	hotbarClusterParams struct {
		Context string `json:"context,omitempty"`
		Bank    int    `json:"bank,omitempty"`
	}
	hotbarMoveParams struct {
		Context  string `json:"context"`
		Position int    `json:"position"`
		Bank     int    `json:"bank,omitempty"`
	}
	hotbarSlotParams struct {
		Slot int `json:"slot"`
		Bank int `json:"bank,omitempty"`
	}
	hotbarSwitchParams struct {
		Bank int `json:"bank"`
	}
	hotbarBankParams struct {
		Bank int `json:"bank,omitempty"`
	}
	hotbarCreateParams struct {
		Name string `json:"name,omitempty"`
	}
	hotbarRenameParams struct {
		Name string `json:"name"`
		Bank int    `json:"bank,omitempty"`
	}
)

// hotbarTile is one cluster on the hotbar.
type hotbarTile struct {
	widget.BaseWidget
	h       *hotbar
	tile    engine.HotbarTile
	slot    int
	hovered bool
}

var (
	_ fyne.Tappable          = (*hotbarTile)(nil)
	_ fyne.SecondaryTappable = (*hotbarTile)(nil)
	_ desktop.Hoverable      = (*hotbarTile)(nil)
)

func newHotbarTile(h *hotbar, t engine.HotbarTile, slot int) *hotbarTile {
	ht := &hotbarTile{h: h, tile: t, slot: slot}
	ht.ExtendBaseWidget(ht)
	return ht
}

// active reports whether the tile is the cluster the window is connected or connecting to.
func (t *hotbarTile) active() bool {
	return t.tile.Connection != "" && t.tile.Connection != engine.Disconnected
}

// tipText is the tooltip: the full context id, its state and its key.
func (t *hotbarTile) tipText() string {
	state := connectionText(engine.Connection{State: t.tile.Connection})
	if t.tile.Missing {
		state = "No kubeconfig source offers this context any more"
	}
	return t.tile.Context + "\n" + state + " · F" + strconv.Itoa(t.slot)
}

// Tapped implements fyne.Tappable: open the cluster.
func (t *hotbarTile) Tapped(*fyne.PointEvent) {
	t.h.sh.tip.hideFor(t)
	t.h.open(t.slot)
}

// TappedSecondary implements fyne.SecondaryTappable: the tile's menu.
func (t *hotbarTile) TappedSecondary(ev *fyne.PointEvent) { t.h.tileMenu(t.slot, ev.AbsolutePosition) }

// MouseIn implements desktop.Hoverable. While a tile is dragged over the others they neither
// light up nor show their tooltip: the line says where it drops.
func (t *hotbarTile) MouseIn(ev *desktop.MouseEvent) {
	if t.h.drag.active {
		return
	}
	t.hovered = true
	t.Refresh()
	t.h.sh.tip.hover(t, t.tipText(), ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (t *hotbarTile) MouseMoved(ev *desktop.MouseEvent) { t.h.sh.tip.moved(t, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (t *hotbarTile) MouseOut() {
	t.hovered = false
	t.Refresh()
	t.h.sh.tip.hideFor(t)
}

// CreateRenderer implements fyne.Widget.
func (t *hotbarTile) CreateRenderer() fyne.WidgetRenderer {
	r := &hotbarTileRenderer{t: t,
		bg: canvas.NewRectangle(color.Transparent), initials: canvas.NewText("", color.Transparent),
		ring: canvas.NewCircle(color.Transparent), dot: canvas.NewCircle(color.Transparent),
		label: canvas.NewText("", color.Transparent),
		img:   &canvas.Image{FillMode: canvas.ImageFillContain, ScaleMode: canvas.ImageScaleSmooth}}
	r.initials.TextStyle = fyne.TextStyle{Bold: true}
	r.Refresh()
	return r
}

type hotbarTileRenderer struct {
	t         *hotbarTile
	bg        *canvas.Rectangle
	initials  *canvas.Text
	ring, dot *canvas.Circle
	label     *canvas.Text
	img       *canvas.Image   // the cluster's uploaded icon (clusteridentity.go), over the square
	id        clusterIdentity // the cluster's name and icon, from its settings
}

func (r *hotbarTileRenderer) sizes() (side, pad, dot float32) {
	th := r.t.Theme()
	return th.Size(sizeHotbarTile), th.Size(theme.SizeNamePadding), th.Size(sizeDotDiameter)
}

func (r *hotbarTileRenderer) MinSize() fyne.Size {
	side, pad, _ := r.sizes()
	lh := fyne.MeasureText("Mg", r.label.TextSize, r.label.TextStyle).Height
	return fyne.NewSize(side+2*pad, pad+side+pad/2+lh+pad)
}

func (r *hotbarTileRenderer) Layout(size fyne.Size) {
	side, pad, d := r.sizes()
	x := (size.Width - side) / 2
	r.bg.Move(fyne.NewPos(x, pad))
	r.bg.Resize(fyne.NewSize(side, side))
	r.img.Move(fyne.NewPos(x+2, pad+2))
	r.img.Resize(fyne.NewSize(side-4, side-4))
	is := r.initials.MinSize()
	r.initials.Move(fyne.NewPos(x+(side-is.Width)/2, pad+(side-is.Height)/2))
	r.initials.Resize(is)
	ring := d + 4
	r.ring.Move(fyne.NewPos(x+side-ring+2, pad+side-ring+2))
	r.ring.Resize(fyne.NewSize(ring, ring))
	r.dot.Move(fyne.NewPos(x+side-d, pad+side-d))
	r.dot.Resize(fyne.NewSize(d, d))
	r.label.Text = ellipsize(r.id.name, size.Width-pad, r.label.TextSize, r.label.TextStyle)
	ls := r.label.MinSize()
	r.label.Move(fyne.NewPos((size.Width-ls.Width)/2, pad+side+pad/2))
	r.label.Resize(ls)
}

func (r *hotbarTileRenderer) Refresh() {
	t := r.t
	th := t.Theme()
	r.id = clusterIdentity{name: t.tile.Label, initials: nameInitials(t.tile.Label), ink: colorAccent}
	if sh := t.h.sh; sh != nil {
		r.id = sh.identity(t.tile.Context)
	}
	r.initials.Text = r.id.initials
	accent := themeColor(th, r.id.ink) // the cluster's icon colour, the accent by default
	r.bg.CornerRadius = th.Size(theme.SizeNameInputRadius)
	r.bg.StrokeWidth = 0
	r.initials.TextSize = th.Size(theme.SizeNameSubHeadingText)
	r.label.TextSize = th.Size(theme.SizeNameCaptionText)
	r.label.Color = themeColor(th, colorMuted)
	r.initials.Color = accent
	switch {
	case t.tile.Missing: // an empty outline in the warning colour: the context is gone
		r.bg.FillColor = color.Transparent
		r.bg.StrokeColor, r.bg.StrokeWidth = themeColor(th, colorStatusWarning), th.Size(theme.SizeNameInputBorder)
		r.initials.Color = themeColor(th, colorMuted)
	case t.active():
		r.bg.FillColor = withAlpha(accent, 0x4c)
		r.bg.StrokeColor, r.bg.StrokeWidth = accent, th.Size(theme.SizeNameInputBorder)
		r.label.Color = themeColor(th, theme.ColorNameForeground)
	default:
		r.bg.FillColor = withAlpha(accent, 0x2c) // the initials badge's tint (pill)
	}
	switch {
	case t.hovered && t.tile.Missing:
		r.bg.FillColor = themeColor(th, theme.ColorNameHover)
	case t.hovered:
		r.bg.FillColor = withAlpha(accent, 0x66)
	}
	r.img.Translucency = 0
	if t.lifted() {
		r.lift(th, accent)
	}
	dot := connectionColor(engine.Connection{State: t.tile.Connection})
	if t.tile.Missing {
		dot = colorStatusWarning
	}
	r.dot.FillColor = themeColor(th, dot)
	r.ring.FillColor = themeColor(th, theme.ColorNameHeaderBackground) // sets the dot off the tile
	if r.id.image != nil && !t.tile.Missing {                          // the uploaded icon in place of the initials
		r.img.Resource = r.id.image
		r.img.Show()
		r.initials.Hide()
	} else {
		r.img.Resource = nil
		r.img.Hide()
		r.initials.Show()
	}
	r.Layout(t.Size())
	for _, o := range r.Objects() {
		o.Refresh()
	}
}

// lift draws the tile being dragged out of its place (hotbardrag.go): an outline where it was, its
// initials, icon and name faded.
func (r *hotbarTileRenderer) lift(th fyne.Theme, accent color.Color) {
	r.bg.FillColor = color.Transparent
	r.bg.StrokeColor, r.bg.StrokeWidth = withAlpha(accent, 0x80), th.Size(theme.SizeNameInputBorder)
	r.initials.Color = withAlpha(accent, 0x80)
	r.label.Color = withAlpha(themeColor(th, colorMuted), 0x80)
	r.img.Translucency = 0.6
}

func (r *hotbarTileRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.img, r.initials, r.ring, r.dot, r.label}
}

func (r *hotbarTileRenderer) Destroy() {}

// hotbarButton is a small text button of the hotbar with a tooltip: the "+" (drawn as an empty
// tile), the switcher's arrows and its number.
type hotbarButton struct {
	widget.BaseWidget
	h           *hotbar
	text        string
	size        fyne.ThemeSizeName
	bold        bool
	square      bool // drawn as an empty tile with an outline
	tip         string
	disabled    bool
	hovered     bool
	onTap       func()
	onSecondary func(fyne.Position)
}

var (
	_ fyne.Tappable          = (*hotbarButton)(nil)
	_ fyne.SecondaryTappable = (*hotbarButton)(nil)
	_ desktop.Hoverable      = (*hotbarButton)(nil)
)

func newHotbarButton(h *hotbar, text string, size fyne.ThemeSizeName, tap func()) *hotbarButton {
	b := &hotbarButton{h: h, text: text, size: size, onTap: tap}
	b.ExtendBaseWidget(b)
	return b
}

func (b *hotbarButton) setText(s string) {
	if s != b.text {
		b.text = s
		b.Refresh()
	}
}

func (b *hotbarButton) setDisabled(d bool) {
	if d != b.disabled {
		b.disabled = d
		b.Refresh()
	}
}

// menuPos is where a menu opened from the button starts: its top-left corner, in window
// coordinates.
func (b *hotbarButton) menuPos() fyne.Position {
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(b)
}

// Tapped implements fyne.Tappable.
func (b *hotbarButton) Tapped(*fyne.PointEvent) {
	b.h.sh.tip.hideFor(b)
	if !b.disabled && b.onTap != nil {
		b.onTap()
	}
}

// TappedSecondary implements fyne.SecondaryTappable.
func (b *hotbarButton) TappedSecondary(ev *fyne.PointEvent) {
	if b.onSecondary != nil {
		b.onSecondary(ev.AbsolutePosition)
	}
}

// MouseIn implements desktop.Hoverable.
func (b *hotbarButton) MouseIn(ev *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
	b.h.sh.tip.hover(b, b.tip, ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (b *hotbarButton) MouseMoved(ev *desktop.MouseEvent) { b.h.sh.tip.moved(b, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (b *hotbarButton) MouseOut() {
	b.hovered = false
	b.Refresh()
	b.h.sh.tip.hideFor(b)
}

// CreateRenderer implements fyne.Widget.
func (b *hotbarButton) CreateRenderer() fyne.WidgetRenderer {
	r := &hotbarButtonRenderer{b: b, bg: canvas.NewRectangle(color.Transparent), txt: canvas.NewText(b.text, color.Transparent)}
	r.Refresh()
	return r
}

type hotbarButtonRenderer struct {
	b   *hotbarButton
	bg  *canvas.Rectangle
	txt *canvas.Text
}

func (r *hotbarButtonRenderer) MinSize() fyne.Size {
	th := r.b.Theme()
	pad := th.Size(theme.SizeNamePadding)
	if r.b.square {
		side := th.Size(sizeHotbarTile)
		return fyne.NewSize(side+2*pad, side+2*pad)
	}
	ts := r.txt.MinSize()
	return fyne.NewSize(ts.Width+2*pad, ts.Height+pad)
}

func (r *hotbarButtonRenderer) Layout(size fyne.Size) {
	th := r.b.Theme()
	if r.b.square {
		side, pad := th.Size(sizeHotbarTile), th.Size(theme.SizeNamePadding)
		r.bg.Move(fyne.NewPos((size.Width-side)/2, pad))
		r.bg.Resize(fyne.NewSize(side, side))
		ts := r.txt.MinSize()
		r.txt.Move(fyne.NewPos((size.Width-ts.Width)/2, pad+(side-ts.Height)/2))
		r.txt.Resize(ts)
		return
	}
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	ts := r.txt.MinSize()
	r.txt.Move(fyne.NewPos((size.Width-ts.Width)/2, (size.Height-ts.Height)/2))
	r.txt.Resize(ts)
}

func (r *hotbarButtonRenderer) Refresh() {
	b := r.b
	th := b.Theme()
	r.txt.Text = b.text
	r.txt.TextSize = th.Size(b.size)
	r.txt.TextStyle = fyne.TextStyle{Bold: b.bold}
	fg := theme.ColorNameForeground
	if b.disabled || b.square {
		fg = colorMuted
	}
	r.txt.Color = themeColor(th, fg)
	r.bg.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	r.bg.FillColor = color.Transparent
	r.bg.StrokeWidth = 0
	if b.square {
		r.bg.CornerRadius = th.Size(theme.SizeNameInputRadius)
		r.bg.StrokeColor, r.bg.StrokeWidth = themeColor(th, theme.ColorNameInputBorder), th.Size(theme.SizeNameInputBorder)
	}
	if b.hovered && !b.disabled {
		r.bg.FillColor = themeColor(th, theme.ColorNameHover)
	}
	r.Layout(b.Size())
	r.txt.Refresh()
	r.bg.Refresh()
}

func (r *hotbarButtonRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.txt} }
func (r *hotbarButtonRenderer) Destroy()                     {}
