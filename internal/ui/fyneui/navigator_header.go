package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// navHeader is the navigator's header (navigator.go): the scope dropdown and the search button,
// and below them the search field while it is open.
type navHeader struct {
	s       *sidebar
	scope   *navScopeButton
	searchB *tipButton
	entry   *navSearchEntry
	field   *fyne.Container // the search field's row, hidden until the search button is pressed
	root    *fyne.Container

	menuModel *fyne.Menu // the open scope menu, for tests
}

func newNavHeader(s *sidebar) *navHeader {
	h := &navHeader{s: s}
	h.scope = newNavScopeButton(h)
	h.searchB = newTipButton(s.sh, theme.SearchIcon(), "Search the navigator", h.toggleSearch)
	h.entry = newNavSearchEntry(h)
	h.field = inset(theme.SizeNamePadding, "", container.NewVBox(h.entry, newSpace(theme.SizeNamePadding)))
	h.field.Hide()
	top := inset(theme.SizeNamePadding, theme.SizeNamePadding,
		container.NewBorder(nil, nil, nil, container.NewCenter(h.searchB), h.scope))
	h.root = container.NewVBox(top, h.field, newDivider(false))
	return h
}

// update follows the navigator's scope: the dropdown names the section, or the cluster the tree
// shows.
func (h *navHeader) update() {
	ui := h.s.sh.ui
	if ui.ScopeCluster() == "" {
		h.scope.setText(navSectionTitle)
		h.setPlaceHolder("Search clusters…")
		return
	}
	name, _ := splitContextID(ui.Context)
	h.scope.setText(name)
	h.setPlaceHolder("Search items…")
}

// setPlaceHolder names what the search filters at the level shown.
func (h *navHeader) setPlaceHolder(text string) {
	if h.entry.PlaceHolder != text {
		h.entry.SetPlaceHolder(text)
	}
}

// showScopeMenu opens the scope dropdown below its button: the section, Local Kubeconfigs, below it
// the hotbar's clusters and the connected one (scopeClusters), the scope shown checked, and All
// Clusters… (the context picker, ⌘P) for every other one (the owner's A52, 2026-10-14 evening: so
// the list needs no cap). Choosing a cluster sets the scope (ui.setNavigatorScope), one not
// connected through the section's drill, which shows its login program for approval first.
func (h *navHeader) showScopeMenu() {
	sh := h.s.sh
	cluster := sh.ui.ScopeCluster()
	local := fyne.NewMenuItem(navSectionTitle, h.s.goUp)
	local.Checked = cluster == ""
	items := []*fyne.MenuItem{local, fyne.NewMenuItemSeparator()}
	ctxs := h.scopeClusters()
	titles := map[string]int{}
	for _, c := range ctxs {
		titles[h.s.sec.rowTitle(c)]++
	}
	for _, c := range ctxs {
		id, label := c.Name, h.s.sec.rowTitle(c)
		if titles[label] > 1 && c.Source != "" { // a name two files share: say which file
			label += " (" + kubeconfig.DisplayText(c.Source, maxShownArg) + ")"
		}
		item := fyne.NewMenuItem(label, func() { h.s.sec.drill(id) })
		item.Checked = id == cluster
		items = append(items, item)
	}
	if len(ctxs) > 0 {
		items = append(items, fyne.NewMenuItemSeparator())
	}
	items = append(items, sh.withShortcut(fyne.NewMenuItem("All Clusters…", sh.picker.show), "cluster.connect"))
	h.menuModel = fyne.NewMenu("", items...)
	sh.tip.hide()
	pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(h.scope).AddXY(0, h.scope.Size().Height)
	widget.NewPopUpMenu(h.menuModel, sh.win.Canvas()).ShowAtPosition(pos)
}

// scopeClusters are the clusters the scope dropdown lists (A52): the tiles of the hotbar's shown bank
// in their order, then the connected (or connecting) cluster when no tile holds it. A tile whose
// context no kubeconfig source offers any more is left out once the section has read them. Which
// bank: the shown one, as the hotbar beside the dropdown shows it (the lead's night decision,
// 2026-10-14, for the owner to confirm; D-51).
func (h *navHeader) scopeClusters() []engine.ContextInfo {
	sh, sec := h.s.sh, h.s.sec
	var out []engine.ContextInfo
	seen := map[string]bool{}
	add := func(id string, always bool) {
		if id == "" || seen[id] {
			return
		}
		c, ok := sec.context(id)
		if !ok && sec.loaded && !always {
			return
		}
		if !ok {
			c = engine.ContextInfo{Name: id}
		}
		seen[id] = true
		out = append(out, c)
	}
	if hb := sh.ui.Hotbar; hb != nil && hb.Current >= 1 && hb.Current <= len(hb.Banks) {
		for _, t := range hb.Banks[hb.Current-1].Clusters {
			add(t.Context, false)
		}
	}
	if st := sh.ui.Connection.State; st == engine.Connected || st == engine.Connecting {
		add(sh.ui.Context, true)
	}
	return out
}

// searching reports whether the search field is open.
func (h *navHeader) searching() bool { return h.field.Visible() }

// toggleSearch opens the search field, or closes and clears it.
func (h *navHeader) toggleSearch() {
	if h.searching() {
		h.closeSearch()
		return
	}
	h.openSearch("")
}

// openSearch shows the search field with text, which filters the tree at once, and gives it the
// keyboard.
func (h *navHeader) openSearch(text string) {
	if !h.searching() {
		h.field.Show()
		h.s.root.Refresh() // the header grew: the tree below it moves down
	}
	h.entry.SetText(text)
	h.entry.CursorColumn = len([]rune(text))
	h.entry.Refresh()
	h.s.sh.win.Canvas().Focus(h.entry)
}

// closeSearch clears the search and hides its field; the keyboard goes back to the tree if the
// field had it.
func (h *navHeader) closeSearch() {
	c := h.s.sh.win.Canvas()
	hadFocus := c.Focused() == h.entry
	h.entry.SetText("")
	h.field.Hide()
	h.s.root.Refresh()
	if hadFocus {
		c.Focus(h.s.list())
	}
}

// navSearchEntry is the navigator's search field: typing filters the tree, Escape clears and
// closes it, ↓ moves into the tree, ↵ opens the first match.
type navSearchEntry struct {
	widget.Entry
	h *navHeader
}

func newNavSearchEntry(h *navHeader) *navSearchEntry {
	e := &navSearchEntry{h: h}
	e.ExtendBaseWidget(e)
	e.SetPlaceHolder("Search items…")
	e.OnChanged = h.s.searchTyped
	e.OnSubmitted = func(string) { h.s.openFirstMatch() }
	return e
}

// TypedKey adds Escape and ↓ to the entry's keys; F1–F12 and ⇧F1–⇧F12 are the hotbar's (A64).
func (e *navSearchEntry) TypedKey(ev *fyne.KeyEvent) {
	if hotbarKey(e.h.s.sh, ev) {
		return
	}
	switch ev.Name {
	case fyne.KeyEscape:
		e.h.closeSearch()
		return
	case fyne.KeyDown:
		if e.h.s.section {
			e.h.s.sec.enter()
		} else {
			e.h.s.tree.enter()
		}
		return
	}
	e.Entry.TypedKey(ev)
}

// navScopeButton is the scope dropdown's button: the scope's name, cut to the navigator's width,
// and a ▾.
type navScopeButton struct {
	widget.BaseWidget
	h       *navHeader
	text    *textLine
	hovered bool
}

var (
	_ fyne.Tappable     = (*navScopeButton)(nil)
	_ desktop.Hoverable = (*navScopeButton)(nil)
)

func newNavScopeButton(h *navHeader) *navScopeButton {
	b := &navScopeButton{h: h, text: newTextLine("No cluster", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})}
	b.text.truncate = true
	b.ExtendBaseWidget(b)
	return b
}

func (b *navScopeButton) setText(s string) { b.text.SetText(s) }

// Tapped implements fyne.Tappable: the scope menu.
func (b *navScopeButton) Tapped(*fyne.PointEvent) {
	b.h.s.sh.tip.hideFor(b)
	b.h.showScopeMenu()
}

// MouseIn implements desktop.Hoverable.
func (b *navScopeButton) MouseIn(ev *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
	b.h.s.sh.tip.hover(b, "The scope the navigator shows", ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (b *navScopeButton) MouseMoved(ev *desktop.MouseEvent) {
	b.h.s.sh.tip.moved(b, ev.AbsolutePosition)
}

// MouseOut implements desktop.Hoverable.
func (b *navScopeButton) MouseOut() {
	b.hovered = false
	b.Refresh()
	b.h.s.sh.tip.hideFor(b)
}

// CreateRenderer implements fyne.Widget.
func (b *navScopeButton) CreateRenderer() fyne.WidgetRenderer {
	r := &navScopeRenderer{b: b, bg: canvas.NewRectangle(color.Transparent), arrow: newIconImage()}
	r.Refresh()
	return r
}

type navScopeRenderer struct {
	b     *navScopeButton
	bg    *canvas.Rectangle
	arrow *iconImage
}

func (r *navScopeRenderer) MinSize() fyne.Size {
	th := r.b.Theme()
	inner, is := th.Size(theme.SizeNameInnerPadding), th.Size(theme.SizeNameInlineIcon)
	return fyne.NewSize(2*inner+is, fyne.Max(r.b.text.MinSize().Height, is)+2*inner)
}

func (r *navScopeRenderer) Layout(size fyne.Size) {
	th := r.b.Theme()
	inner, pad, is := th.Size(theme.SizeNameInnerPadding), th.Size(theme.SizeNamePadding), th.Size(theme.SizeNameInlineIcon)
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	r.arrow.img.Move(fyne.NewPos(size.Width-inner-is, (size.Height-is)/2))
	r.arrow.img.Resize(fyne.NewSize(is, is))
	r.b.text.Move(fyne.NewPos(inner, 0))
	r.b.text.Resize(fyne.NewSize(fyne.Max(0, size.Width-2*inner-is-pad), size.Height))
}

func (r *navScopeRenderer) Refresh() {
	th := r.b.Theme()
	r.bg.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	r.bg.FillColor = color.Transparent
	if r.b.hovered {
		r.bg.FillColor = themeColor(th, theme.ColorNameHover)
	}
	r.arrow.set(th, theme.MenuDropDownIcon(), theme.ColorNameForeground)
	r.Layout(r.b.Size())
	r.bg.Refresh()
	r.b.text.Refresh()
}

func (r *navScopeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.b.text, r.arrow.img}
}

func (r *navScopeRenderer) Destroy() {}
