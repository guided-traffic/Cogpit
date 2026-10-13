package fyneui

import (
	"image/color"
	"slices"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The navigator's header and tree (ui-parity › Layout and Clusters). Captured from Lens Desktop
// (2026-10-11): above the tree a dropdown of the scope the tree shows (one cluster, or a section
// such as "Local Kubeconfigs") and a search button; the tree's root is the cluster, with its icon,
// a connection dot and a ›, and a ‹ to go back; hovering an item shows "<item> - <context>". From
// Lens's documentation: ↑↓ move, → opens, ← closes, Tab and ⇧Tab, ↵ a preview tab, ⇧↵ a fixed tab.
//
// The level above the tree is the section, Local Kubeconfigs (navigator_sections.go); which one
// shows is the navigator's scope, engine state (UIState.NavigatorScope). The rest is Cogpit's
// (inferred):
//   - the dropdown offers the section and its clusters (navigator_header.go); the root's ‹ and ← on
//     the root go up to the section (ui.setNavigatorScope), the context picker stays on ⌘P; the ›
//     is drawn as captured and does nothing, the tree being drilled into its cluster already;
//   - the search filters the items by name, case-insensitively: a matching item shows with the
//     groups above it, a matching group with everything in it, and groups show open while a search
//     is on (their own open state comes back after it); Escape clears and closes the field, ↓ moves
//     into the tree, ↵ opens the first match, and a letter typed in the tree starts a search;
//   - ↵ and a click open the item's view in the preview tab (ui.openView), ⇧↵ and a double click in
//     a fixed tab (ui.tab.open; tabbar.go); on a group they open or close it, on the root they do
//     nothing; Space does what ↵ does; Home and End go to
//     the first and the last row; Escape clears the search, else closes the detail drawer;
//   - a click focuses the tree, and the row the keyboard is on has a ring in the accent colour
//     while the tree has the focus, as the table's keyboard cell has (tablekeys.go).
//
// The search text, the keyboard's row and which groups are open are the window's, not engine
// state: they change what the navigator lists, not what a view shows (architecture §4.3). Opening a
// view is engine state and goes through ui.openView or ui.tab.open, so agents see it.

// rootNode is the node id of the tree's root, the connected cluster. Kind keys, page keys, groups
// and aliases never take this form.
const rootNode = "cluster:"

// Cogpit's chevrons ‹ and ›, drawn for Cogpit like the open group's chevron in theme.go: Fyne's
// NavigateBack and NavigateNext icons are arrows, and the capture shows chevrons.
var (
	iconChevronRight = fyne.NewStaticResource("cogpit-chevron-right.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">`+
			`<path d="M8.6 7.4 10 6l6 6-6 6-1.4-1.4 4.6-4.6z"/></svg>`))
	iconChevronLeft = fyne.NewStaticResource("cogpit-chevron-left.svg", []byte(
		`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">`+
			`<path d="M15.4 7.4 14 6l-6 6 6 6 1.4-1.4-4.6-4.6z"/></svg>`))
)

// navTree is the navigator's tree: Cogpit's own rows in a scroll, not Fyne's widget.Tree, whose
// rows keep the hover and the keyboard's highlight private (a row's tooltip needs its pointer, the
// focus ring needs the keyboard's row) and cannot draw the cluster as the root.
type navTree struct {
	widget.BaseWidget
	s *sidebar

	rows    map[string]*navRow // the item rows made, by node id; rows of hidden nodes are dropped
	visible []string           // the shown node ids in order, rootNode first
	list    *fyne.Container
	scroll  *container.Scroll

	focus   string // the row the keyboard is on
	focused bool   // the tree has the window's keyboard focus
	shift   bool   // Shift is held (⇧F1–⇧F12 for the hotbar)
}

var (
	_ fyne.Focusable  = (*navTree)(nil)
	_ desktop.Keyable = (*navTree)(nil)
)

func newNavTree(s *sidebar) *navTree {
	t := &navTree{s: s, rows: map[string]*navRow{}, visible: []string{rootNode}}
	t.list = container.New(layout.NewVBoxLayout())
	t.scroll = container.NewVScroll(inset(theme.SizeNamePadding, theme.SizeNamePadding, t.list))
	t.ExtendBaseWidget(t)
	return t
}

// CreateRenderer implements fyne.Widget.
func (t *navTree) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(t.scroll) }

// IsBranchOpen reports whether the user opened a group (a search shows every group open without
// changing this).
func (t *navTree) IsBranchOpen(id string) bool { return t.s.open[id] }

// OpenBranch opens a group.
func (t *navTree) OpenBranch(id string) { t.s.setOpen(id, true) }

// CloseBranch closes a group.
func (t *navTree) CloseBranch(id string) { t.s.setOpen(id, false) }

// Select does what a click on the node does, without moving the focus: it opens the node's view,
// or opens or closes a group.
func (t *navTree) Select(id string) { t.s.activate(id) }

// rebuild lays the rows out again from the sidebar's nodes, the open groups and the search.
func (t *navTree) rebuild() {
	s := t.s
	ids := s.shownNodes()
	t.visible = append(append(t.visible[:0], rootNode), ids...)
	objs := make([]fyne.CanvasObject, 0, len(ids)+2)
	objs = append(objs, s.rootRow)
	if hint := s.hint(); hint != "" {
		s.emptyHint.SetText(hint)
		objs = append(objs, s.hintRow)
	}
	rows := make(map[string]*navRow, len(ids))
	for _, id := range ids {
		r := t.rows[id]
		if r == nil {
			r = newNavRow(t, id)
		}
		r.update(s.titles[id], navIcon(id), s.isBranch(id), s.depth(id))
		rows[id] = r
		objs = append(objs, r)
	}
	if tip := s.sh.tip; tip != nil { // nil while the shell is being built
		if owner, ok := tip.owner.(*navRow); ok && rows[owner.id] != owner {
			tip.hide() // its row is gone and would never say the pointer left
		}
	}
	t.rows = rows
	if !slices.Contains(t.visible, t.focus) {
		t.focus = t.shownAncestor(t.focus)
	}
	t.list.Objects = objs
	// The scroll fits its content to the rows' new height, then lays out and refreshes every row:
	// selection, focus and chevrons.
	t.scroll.Refresh()
}

// shownAncestor is the nearest shown node above id, else the root.
func (t *navTree) shownAncestor(id string) string {
	for p := t.s.parents[id]; p != ""; p = t.s.parents[p] {
		if slices.Contains(t.visible, p) {
			return p
		}
	}
	return rootNode
}

// refreshRow redraws one row: after the focus or the mark moved.
func (t *navTree) refreshRow(id string) {
	if id == rootNode {
		t.s.rootRow.Refresh()
		return
	}
	if r := t.rows[id]; r != nil {
		r.Refresh()
	}
}

// rowOf returns the row object of a shown node.
func (t *navTree) rowOf(id string) fyne.CanvasObject {
	if id == rootNode {
		return t.s.rootRow
	}
	if r := t.rows[id]; r != nil {
		return r
	}
	return nil
}

// setFocus moves the keyboard to a shown node and scrolls it into view.
func (t *navTree) setFocus(id string) {
	if id != t.focus {
		old := t.focus
		t.focus = id
		t.refreshRow(old)
		t.refreshRow(id)
	}
	t.reveal(id)
}

// reveal scrolls a shown node's row into view.
func (t *navTree) reveal(id string) {
	row := t.rowOf(id)
	if row == nil || !row.Visible() {
		return
	}
	y := t.list.Position().Y + row.Position().Y
	h := row.Size().Height
	view := t.scroll.Size().Height
	if view <= 0 {
		return
	}
	off := t.scroll.Offset.Y
	switch {
	case y < off:
		t.scroll.ScrollToOffset(fyne.NewPos(0, y))
	case y+h > off+view:
		t.scroll.ScrollToOffset(fyne.NewPos(0, y+h-view))
	}
}

// tapped is a click on a row: the tree takes the keyboard there, and the row does what it does.
func (t *navTree) tapped(id string) {
	t.setFocus(id)
	if c := t.s.sh.win.Canvas(); c.Focused() != t {
		c.Focus(t)
	}
	t.s.activate(id)
}

// enter moves the keyboard into the tree from the search field: to the first match.
func (t *navTree) enter() {
	if len(t.visible) > 1 {
		t.focus = t.visible[1]
	}
	t.s.sh.win.Canvas().Focus(t)
	t.setFocus(t.focus)
}

// step moves the keyboard d rows down (up for a negative d), stopping at the ends.
func (t *navTree) step(d int) {
	i := slices.Index(t.visible, t.focus)
	switch {
	case i < 0:
		i = 0
	case i+d >= 0 && i+d < len(t.visible):
		i += d
	}
	t.setFocus(t.visible[i])
}

// right opens a closed group, and moves into an open one or from the root to the first item.
func (t *navTree) right() {
	s, id := t.s, t.focus
	if id != rootNode && s.isBranch(id) && !s.shownOpen(id) {
		s.setOpen(id, true)
		return
	}
	if id != rootNode && !s.isBranch(id) {
		return
	}
	if i := slices.Index(t.visible, id); i >= 0 && i+1 < len(t.visible) && s.parentOf(t.visible[i+1]) == id {
		t.setFocus(t.visible[i+1])
	}
}

// left closes an open group, else moves to the row above in the tree: an item's group, a top-level
// item's root, and from the root up to the section level. While a search is on, groups stay open
// and ← only moves.
func (t *navTree) left() {
	s, id := t.s, t.focus
	switch {
	case id == rootNode:
		s.goUp()
	case id == "":
	case s.isBranch(id) && s.open[id] && s.filter == "":
		s.setOpen(id, false)
	default:
		t.setFocus(s.parentOf(id))
	}
}

// TypedKey implements fyne.Focusable: the navigator's keyboard.
func (t *navTree) TypedKey(ev *fyne.KeyEvent) {
	if t.s.sh.hot.key(ev.Name, t.shift) { // F1–F12 and ⇧F1–⇧F12 belong to the hotbar
		return
	}
	switch ev.Name {
	case fyne.KeyUp:
		t.step(-1)
	case fyne.KeyDown:
		t.step(1)
	case fyne.KeyHome:
		t.setFocus(t.visible[0])
	case fyne.KeyEnd:
		t.setFocus(t.visible[len(t.visible)-1])
	case fyne.KeyRight:
		t.right()
	case fyne.KeyLeft:
		t.left()
	case fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace:
		if t.shift && ev.Name != fyne.KeySpace { // ⇧↵: a fixed tab (tabbar.go)
			t.s.openFixed(t.focus)
			return
		}
		t.s.activate(t.focus) // the preview tab
	case fyne.KeyEscape:
		if t.s.head.searching() {
			t.s.head.closeSearch()
			return
		}
		t.s.sh.closeDrawer()
	}
}

// TypedRune implements fyne.Focusable: a letter typed in the tree starts a search with it.
func (t *navTree) TypedRune(r rune) {
	if unicode.IsPrint(r) && !unicode.IsSpace(r) {
		t.s.head.openSearch(t.s.filter + string(r))
	}
}

// KeyDown implements desktop.Keyable: it tracks Shift, which F-keys reach TypedKey without.
func (t *navTree) KeyDown(ev *fyne.KeyEvent) {
	if ev.Name == desktop.KeyShiftLeft || ev.Name == desktop.KeyShiftRight {
		t.shift = true
	}
}

// KeyUp implements desktop.Keyable.
func (t *navTree) KeyUp(ev *fyne.KeyEvent) {
	if ev.Name == desktop.KeyShiftLeft || ev.Name == desktop.KeyShiftRight {
		t.shift = false
	}
}

// FocusGained implements fyne.Focusable: the keyboard starts on the open view's row, else on the
// first item.
func (t *navTree) FocusGained() {
	t.focused = true
	if !slices.Contains(t.visible, t.focus) {
		switch {
		case slices.Contains(t.visible, t.s.marked):
			t.focus = t.s.marked
		case len(t.visible) > 1:
			t.focus = t.visible[1]
		default:
			t.focus = rootNode
		}
	}
	t.setFocus(t.focus)
	t.refreshRow(t.focus)
}

// FocusLost implements fyne.Focusable.
func (t *navTree) FocusLost() {
	t.focused, t.shift = false, false
	t.refreshRow(t.focus)
}

// keyboardOn reports whether the ring is on a node: the tree has the focus and the keyboard is there.
func (t *navTree) keyboardOn(id string) bool { return t.focused && t.focus == id }

// navTip is the tooltip of a navigator item: "<item> - <context>" (captured), the context by its
// name without the file it comes from.
func navTip(item, contextID string) string {
	if contextID == "" {
		return item
	}
	return item + " - " + engine.ContextLabel(contextID)
}

// navRow is one item of the tree: a group's chevron, an icon for top-level items and groups, and
// the title, indented by its depth.
type navRow struct {
	widget.BaseWidget
	t       *navTree
	id      string
	title   string
	icon    fyne.Resource
	branch  bool
	depth   int
	hovered bool
}

var (
	_ fyne.Tappable     = (*navRow)(nil)
	_ desktop.Hoverable = (*navRow)(nil)
)

func newNavRow(t *navTree, id string) *navRow {
	r := &navRow{t: t, id: id}
	r.ExtendBaseWidget(r)
	return r
}

// update sets what the row shows; an unchanged row costs nothing (the tree's rebuild refreshes it).
func (r *navRow) update(title string, icon fyne.Resource, branch bool, depth int) {
	r.title, r.icon, r.branch, r.depth = title, icon, branch, depth
}

// tipText is the row's tooltip.
func (r *navRow) tipText() string { return navTip(r.title, r.t.s.sh.ui.Context) }

// Tapped implements fyne.Tappable.
func (r *navRow) Tapped(*fyne.PointEvent) {
	r.t.s.sh.tip.hideFor(r)
	if r.t.s.navClicked(r.id) { // a double click's second: a fixed tab (tabbar.go)
		r.t.s.openFixed(r.id)
		return
	}
	r.t.tapped(r.id)
}

// MouseIn implements desktop.Hoverable.
func (r *navRow) MouseIn(ev *desktop.MouseEvent) {
	r.hovered = true
	r.Refresh()
	r.t.s.sh.tip.hover(r, r.tipText(), ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (r *navRow) MouseMoved(ev *desktop.MouseEvent) { r.t.s.sh.tip.moved(r, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (r *navRow) MouseOut() {
	r.hovered = false
	r.Refresh()
	r.t.s.sh.tip.hideFor(r)
}

// CreateRenderer implements fyne.Widget.
func (r *navRow) CreateRenderer() fyne.WidgetRenderer {
	rr := &navRowRenderer{r: r, bg: canvas.NewRectangle(color.Transparent), ring: canvas.NewRectangle(color.Transparent),
		chevron: newIconImage(), icon: newIconImage(), txt: canvas.NewText("", color.Transparent)}
	rr.Refresh()
	return rr
}

type navRowRenderer struct {
	r             *navRow
	bg, ring      *canvas.Rectangle
	chevron, icon *iconImage
	txt           *canvas.Text
}

// rowMinSize is the size of an item row: a line of text with the inner padding above and below,
// and room for an icon.
func rowMinSize(th fyne.Theme) fyne.Size {
	is := th.Size(theme.SizeNameInlineIcon)
	h := fyne.MeasureText("Mg", th.Size(theme.SizeNameText), fyne.TextStyle{}).Height + 2*th.Size(theme.SizeNameInnerPadding)
	return fyne.NewSize(2*is, fyne.Max(h, is))
}

func (rr *navRowRenderer) MinSize() fyne.Size { return rowMinSize(rr.r.Theme()) }

func (rr *navRowRenderer) Layout(size fyne.Size) {
	r := rr.r
	th := r.Theme()
	pad, is := th.Size(theme.SizeNamePadding), th.Size(theme.SizeNameInlineIcon)
	rr.bg.Move(fyne.NewPos(0, 0))
	rr.bg.Resize(size)
	rr.ring.Move(fyne.NewPos(0, 0))
	rr.ring.Resize(size)
	x := pad + float32(r.depth)*(is+pad)
	rr.chevron.img.Move(fyne.NewPos(x, (size.Height-is)/2))
	rr.chevron.img.Resize(fyne.NewSize(is, is))
	x += is + pad
	if r.icon != nil {
		rr.icon.img.Move(fyne.NewPos(x, (size.Height-is)/2))
		rr.icon.img.Resize(fyne.NewSize(is, is))
		x += is + 2*pad
	}
	rr.txt.Text = ellipsize(r.title, size.Width-x-pad, rr.txt.TextSize, rr.txt.TextStyle)
	ts := rr.txt.MinSize()
	rr.txt.Move(fyne.NewPos(x, (size.Height-ts.Height)/2))
	rr.txt.Resize(ts)
}

func (rr *navRowRenderer) Refresh() {
	r := rr.r
	th := r.Theme()
	paintRow(th, rr.bg, rr.ring, r.t.s.marked == r.id, r.hovered, r.t.keyboardOn(r.id))
	var chevron fyne.Resource
	if r.branch {
		chevron = iconChevronRight
		if r.t.s.shownOpen(r.id) {
			chevron = theme.MoveDownIcon() // Cogpit's chevron (theme.go)
		}
	}
	rr.chevron.set(th, chevron, theme.ColorNameForeground)
	rr.icon.set(th, r.icon, colorMuted)
	rr.txt.TextSize = th.Size(theme.SizeNameText)
	rr.txt.Color = themeColor(th, theme.ColorNameForeground)
	rr.Layout(r.Size())
	rr.bg.Refresh()
	rr.ring.Refresh()
	rr.txt.Refresh()
}

func (rr *navRowRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{rr.bg, rr.chevron.img, rr.icon.img, rr.txt, rr.ring}
}

func (rr *navRowRenderer) Destroy() {}

// paintRow colours a row's background (the open view's row, else the hovered one) and its focus
// ring.
func paintRow(th fyne.Theme, bg, ring *canvas.Rectangle, selected, hovered, focused bool) {
	bg.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	switch {
	case selected:
		bg.FillColor = themeColor(th, theme.ColorNameSelection)
	case hovered:
		bg.FillColor = themeColor(th, theme.ColorNameHover)
	default:
		bg.FillColor = color.Transparent
	}
	ring.FillColor = color.Transparent
	ring.StrokeColor = themeColor(th, colorAccent)
	ring.StrokeWidth = th.Size(theme.SizeNameInputBorder)
	ring.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	if focused {
		ring.Show()
	} else {
		ring.Hide()
	}
}

// navRootRow is the tree's root: the connected cluster with a ‹ up to the section, its badge,
// name and kubeconfig, a connection dot and a ›. Its menu (right-click) has the hotbar's item for
// the cluster, Add to Hotbar or Remove from Hotbar, as the picker's rows have (inferred: Lens's
// cluster menu has "Add to Hotbar", its other items are not captured).
type navRootRow struct {
	widget.BaseWidget
	s         *sidebar
	hovered   bool
	back      *navIconButton
	next      *widget.Icon // the ›
	body      fyne.CanvasObject
	bg, ring  *canvas.Rectangle
	menuModel *fyne.Menu // the open menu, for tests
}

// showCluster shows or hides what only a cluster has: the dot, the › and the subtitle's line.
func (r *navRootRow) showCluster(has bool) {
	for _, o := range []fyne.CanvasObject{r.s.dot, r.next, r.s.subtitle} {
		if has {
			o.Show()
		} else {
			o.Hide()
		}
	}
}

var (
	_ fyne.Tappable          = (*navRootRow)(nil)
	_ fyne.SecondaryTappable = (*navRootRow)(nil)
	_ desktop.Hoverable      = (*navRootRow)(nil)
)

// TappedSecondary implements fyne.SecondaryTappable: the cluster's menu (Settings, the hotbar's
// item, Disconnect, Remove from Cogpit), while it has an item.
func (r *navRootRow) TappedSecondary(ev *fyne.PointEvent) {
	sh := r.s.sh
	var items []*fyne.MenuItem
	for _, item := range append([]*fyne.MenuItem{sh.settingsItem(sh.ui.Context), sh.hot.contextItem(sh.ui.Context)},
		sh.clusterItems(sh.ui.Context)...) { // Disconnect, Remove from Cogpit (menus_cluster.go)
		if item != nil {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return
	}
	sh.tip.hide()
	r.menuModel = fyne.NewMenu("", items...)
	widget.NewPopUpMenu(r.menuModel, sh.win.Canvas()).ShowAtPosition(ev.AbsolutePosition)
}

func newNavRootRow(s *sidebar) *navRootRow {
	r := &navRootRow{s: s, bg: canvas.NewRectangle(color.Transparent), ring: canvas.NewRectangle(color.Transparent)}
	r.back = newNavIconButton(s.sh, iconChevronLeft, "Back to "+navSectionTitle, s.goUp)
	r.next = widget.NewIcon(themedIcon(iconChevronRight, colorMuted))
	r.body = inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewBorder(nil, nil,
		container.NewHBox(container.NewCenter(r.back), container.NewCenter(s.badge)),
		container.NewHBox(container.NewCenter(s.dot), container.NewCenter(r.next)),
		container.NewVBox(layout.NewSpacer(), s.name, s.subtitle, layout.NewSpacer())))
	r.ExtendBaseWidget(r)
	return r
}

// tipText is the root's tooltip: the context's full id and the connection's state.
func (r *navRootRow) tipText() string {
	ui := r.s.sh.ui
	if ui.Context == "" {
		return ""
	}
	return ui.Context + " - " + connectionText(ui.Connection)
}

// Tapped implements fyne.Tappable: the keyboard goes to the root.
func (r *navRootRow) Tapped(*fyne.PointEvent) {
	r.s.sh.tip.hideFor(r)
	r.s.tree.tapped(rootNode)
}

// MouseIn implements desktop.Hoverable.
func (r *navRootRow) MouseIn(ev *desktop.MouseEvent) {
	r.hovered = true
	r.Refresh()
	r.s.sh.tip.hover(r, r.tipText(), ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (r *navRootRow) MouseMoved(ev *desktop.MouseEvent) { r.s.sh.tip.moved(r, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (r *navRootRow) MouseOut() {
	r.hovered = false
	r.Refresh()
	r.s.sh.tip.hideFor(r)
}

// CreateRenderer implements fyne.Widget.
func (r *navRootRow) CreateRenderer() fyne.WidgetRenderer {
	rr := &navRootRenderer{r: r}
	rr.Refresh()
	return rr
}

type navRootRenderer struct{ r *navRootRow }

func (rr *navRootRenderer) MinSize() fyne.Size { return rr.r.body.MinSize() }

func (rr *navRootRenderer) Layout(size fyne.Size) {
	for _, o := range []fyne.CanvasObject{rr.r.bg, rr.r.ring, rr.r.body} {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

func (rr *navRootRenderer) Refresh() {
	r := rr.r
	paintRow(r.Theme(), r.bg, r.ring, false, r.hovered, r.s.tree.keyboardOn(rootNode))
	rr.Layout(r.Size())
	r.bg.Refresh()
	r.ring.Refresh()
}

func (rr *navRootRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{rr.r.bg, rr.r.body, rr.r.ring}
}

func (rr *navRootRenderer) Destroy() {}

// clusterBadge, the cluster's identity at the root, is drawn in clusteridentity.go.

// navIconButton is a small icon with a tooltip and a hover background: the root's ‹. It takes no
// keyboard focus, so Tab leaves the tree at once (the top bar's "Switch context" does what it does).
type navIconButton struct {
	widget.BaseWidget
	sh      *shell
	icon    fyne.Resource
	tip     string
	onTap   func()
	hovered bool
}

var (
	_ fyne.Tappable     = (*navIconButton)(nil)
	_ desktop.Hoverable = (*navIconButton)(nil)
)

func newNavIconButton(sh *shell, icon fyne.Resource, tip string, tap func()) *navIconButton {
	b := &navIconButton{sh: sh, icon: icon, tip: tip, onTap: tap}
	b.ExtendBaseWidget(b)
	return b
}

// Tapped implements fyne.Tappable.
func (b *navIconButton) Tapped(*fyne.PointEvent) {
	b.sh.tip.hideFor(b)
	if b.onTap != nil {
		b.onTap()
	}
}

// MouseIn implements desktop.Hoverable.
func (b *navIconButton) MouseIn(ev *desktop.MouseEvent) {
	b.hovered = true
	b.Refresh()
	b.sh.tip.hover(b, b.tip, ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (b *navIconButton) MouseMoved(ev *desktop.MouseEvent) { b.sh.tip.moved(b, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (b *navIconButton) MouseOut() {
	b.hovered = false
	b.Refresh()
	b.sh.tip.hideFor(b)
}

// CreateRenderer implements fyne.Widget.
func (b *navIconButton) CreateRenderer() fyne.WidgetRenderer {
	r := &navIconButtonRenderer{b: b, bg: canvas.NewRectangle(color.Transparent), img: newIconImage()}
	r.Refresh()
	return r
}

type navIconButtonRenderer struct {
	b   *navIconButton
	bg  *canvas.Rectangle
	img *iconImage
}

func (r *navIconButtonRenderer) MinSize() fyne.Size {
	th := r.b.Theme()
	side := th.Size(theme.SizeNameInlineIcon) + 2*th.Size(theme.SizeNamePadding)
	return fyne.NewSize(side, side)
}

func (r *navIconButtonRenderer) Layout(size fyne.Size) {
	is := r.b.Theme().Size(theme.SizeNameInlineIcon)
	r.bg.Resize(size)
	r.img.img.Move(fyne.NewPos((size.Width-is)/2, (size.Height-is)/2))
	r.img.img.Resize(fyne.NewSize(is, is))
}

func (r *navIconButtonRenderer) Refresh() {
	th := r.b.Theme()
	r.bg.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	r.bg.FillColor = color.Transparent
	if r.b.hovered {
		r.bg.FillColor = themeColor(th, theme.ColorNameHover)
	}
	r.img.set(th, r.b.icon, theme.ColorNameForeground)
	r.Layout(r.b.Size())
	r.bg.Refresh()
}

func (r *navIconButtonRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.img.img}
}

func (r *navIconButtonRenderer) Destroy() {}
