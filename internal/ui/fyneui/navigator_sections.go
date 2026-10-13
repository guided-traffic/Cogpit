package fyneui

import (
	"image/color"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The navigator's section level (ui-parity › Layout and Clusters, G5, G13): the level above a
// cluster's tree. Lens's documentation lists the navigator's sections, Kubernetes Clusters (Local
// Kubeconfigs and the cloud sections), Teamwork and Extensions; its captures show the scope
// dropdown naming a section such as "Local Kubeconfigs", a cluster's root with a › and a ‹ to go
// back. The scope is engine state (UIState.NavigatorScope, engine/navscope.go): this level shows
// while it is the section, the tree (navigator.go) while it is a cluster.
//
// Cogpit's defaults (inferred, not the owner's decision, 2026-10-13):
//   - one section, "Local Kubeconfigs" under "Kubernetes Clusters", as the level's root; the cloud
//     sections (after the beta, G18), Teamwork (D-9) and Extensions (D-5) are not built, so the
//     root has no ‹ and nothing above it;
//   - every context the kubeconfig sources offer, the context picker's list, grouped under a line
//     naming the folder or file they come from (".kube/", "dev.yaml"), the demo cluster first
//     without one; each row the cluster's badge (its identity), its display name, a connection
//     dot (the connected one's state; red after a failed connect; grey otherwise) and a ›; the
//     source as the tooltip, "<name> - <source>"; the active cluster's row marked;
//   - a click on a row or its ›, → or ↵ drills into the cluster (ui.setNavigatorScope): a cluster
//     not connected is connected through the engine as the picker connects it, its login program
//     shown for approval first; ↑↓, Home and End move, ← goes to the root, Escape clears the
//     search or closes the drawer, a letter starts the search, F1–F12 reach the hotbar;
//   - the header's search filters the clusters by name, context and source, ↵ drills into the
//     first match;
//   - a row's right-click menu is the cluster's (Connect, Settings…, the hotbar's item,
//     Disconnect, Remove from Cogpit), the root's has Add Kubeconfigs….

// The section's names (Lens's, from its documentation and captures).
const (
	navSectionTitle  = "Local Kubeconfigs"
	navSectionParent = "Kubernetes Clusters"
	// navSectionRoot is the keyboard's id of the section's root row; context ids never take it.
	navSectionRoot = "\x00section"
)

// navScopeParams are ui.setNavigatorScope's parameters, mirroring the engine's schema.
type navScopeParams struct {
	Scope string `json:"scope"`
}

// navSections is the navigator's section level: its root and the clusters below it.
type navSections struct {
	widget.BaseWidget
	s *sidebar

	root    *navSectionRootRow
	list    *fyne.Container
	scroll  *container.Scroll
	hint    *textLine
	hintRow fyne.CanvasObject

	contexts []engine.ContextInfo // as the kubeconfig sources offer them, read off the main goroutine
	loaded   bool
	loadSeq  int
	loadErr  string

	rows    map[string]*navContextRow // the rows made, by context id
	heads   map[string]*textLine      // the groups' heading lines made, by group
	visible []string                  // the keyboard's rows in order: navSectionRoot, then the shown contexts

	focus   string // the row the keyboard is on
	focused bool   // the list has the window's keyboard focus
	shift   bool   // Shift is held (⇧F1–⇧F12 for the hotbar)

	menuModel *fyne.Menu // a row's open menu, for tests
	subOnce   sync.Once
}

var (
	_ fyne.Focusable  = (*navSections)(nil)
	_ desktop.Keyable = (*navSections)(nil)
)

func newNavSections(s *sidebar) *navSections {
	n := &navSections{s: s, rows: map[string]*navContextRow{}, heads: map[string]*textLine{}, visible: []string{navSectionRoot}}
	n.root = newNavSectionRootRow(n)
	n.hint = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	n.hint.truncate = true
	n.hintRow = inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, n.hint)
	n.list = container.New(layout.NewVBoxLayout(), n.root)
	n.scroll = container.NewVScroll(inset(theme.SizeNamePadding, theme.SizeNamePadding, n.list))
	n.ExtendBaseWidget(n)
	return n
}

// CreateRenderer implements fyne.Widget.
func (n *navSections) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(n.scroll) }

// load reads the contexts off the main goroutine, then shows them; a newer load supersedes an
// older one. It also starts following the contexts (a kubeconfig synced or removed).
func (n *navSections) load() {
	n.subscribe()
	n.loadSeq++
	seq := n.loadSeq
	sh := n.s.sh
	go func() {
		defer sh.recoverTo("the navigator's clusters")
		ctxs, err := sh.eng.Contexts() // a partial list is shown with what failed
		sh.do(func() {
			if seq != n.loadSeq {
				return
			}
			n.contexts, n.loaded, n.loadErr = ctxs, true, ""
			if err != nil {
				n.loadErr = err.Error()
			}
			n.render()
		})
	}()
}

// subscribe reloads the contexts when the engine reports that they changed, for as long as the
// shell runs. It never touches a widget on the engine's goroutine.
func (n *navSections) subscribe() {
	n.subOnce.Do(func() {
		sh := n.s.sh
		cancel := sh.eng.Subscribe(func(ev engine.Event) {
			if ev.Type == engine.EventContexts {
				sh.do(n.load)
			}
		})
		go func() {
			<-sh.ctx.Done()
			cancel()
		}()
	})
}

// context returns a listed context.
func (n *navSections) context(id string) (engine.ContextInfo, bool) {
	i := slices.IndexFunc(n.contexts, func(c engine.ContextInfo) bool { return c.Name == id })
	if i < 0 {
		return engine.ContextInfo{}, false
	}
	return n.contexts[i], true
}

// sourceGroup is the group a context is listed under: the synced folder its file is in (".kube/"),
// or the file the user added ("dev.yaml"); "" for a context without a kubeconfig source (the demo
// cluster).
func sourceGroup(c engine.ContextInfo) string {
	if i := strings.LastIndex(c.Source, "/"); i >= 0 {
		return c.Source[:i+1]
	}
	return c.Source
}

// rowTitle is a context's name in the section: its display name (cluster settings), else the
// context's name without its file; the demo cluster's is short.
func (n *navSections) rowTitle(c engine.ContextInfo) string {
	if c.Demo && n.s.sh.ui.ClusterSettings[c.Name].Name == "" {
		return "Demo cluster"
	}
	return kubeconfig.DisplayText(n.s.sh.identity(c.Name).name, maxShownPath)
}

// matches reports whether a context matches the search, ignoring case: its name, display name or
// source.
func (n *navSections) matches(c engine.ContextInfo, f string) bool {
	if f == "" {
		return true
	}
	return strings.Contains(strings.ToLower(c.Name+" "+n.rowTitle(c)+" "+c.Source), strings.ToLower(f))
}

// groups are the contexts matching filter by group, in order: the contexts without a source first,
// then the groups by name; within a group as the engine lists them (by id).
func (n *navSections) groups(filter string) (keys []string, byGroup map[string][]engine.ContextInfo) {
	byGroup = map[string][]engine.ContextInfo{}
	for _, c := range n.contexts {
		if !n.matches(c, filter) {
			continue
		}
		g := sourceGroup(c)
		if _, ok := byGroup[g]; !ok {
			keys = append(keys, g)
		}
		byGroup[g] = append(byGroup[g], c)
	}
	slices.Sort(keys) // "" first
	return keys, byGroup
}

// ordered are every context in the order the section lists them (the scope dropdown's order).
func (n *navSections) ordered() []engine.ContextInfo {
	keys, byGroup := n.groups("")
	var out []engine.ContextInfo
	for _, g := range keys {
		out = append(out, byGroup[g]...)
	}
	return out
}

// render lays the rows out again: after a load, a search, a connection or an identity changed.
func (n *navSections) render() {
	keys, byGroup := n.groups(n.s.filter)
	objs := []fyne.CanvasObject{n.root}
	visible := []string{navSectionRoot}
	rows := make(map[string]*navContextRow, len(n.rows))
	heads := make(map[string]*textLine, len(n.heads))
	count := 0
	for _, g := range keys {
		if g != "" {
			h := n.heads[g]
			if h == nil {
				h = newTextLine(kubeconfig.DisplayText(g, maxShownPath), colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{Bold: true})
				h.truncate = true
			}
			heads[g] = h
			objs = append(objs, inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, h))
		}
		for _, c := range byGroup[g] {
			r := n.rows[c.Name]
			if r == nil {
				r = newNavContextRow(n, c.Name)
			}
			r.set(c)
			rows[c.Name] = r
			objs = append(objs, r)
			visible = append(visible, c.Name)
			count++
		}
	}
	if hint := n.hintText(count); hint != "" {
		n.hint.SetText(hint)
		objs = append(objs, n.hintRow)
	}
	if tip := n.s.sh.tip; tip != nil {
		if owner, ok := tip.owner.(*navContextRow); ok && rows[owner.id] != owner {
			tip.hide() // its row is gone and would never say the pointer left
		}
	}
	n.rows, n.heads, n.visible = rows, heads, visible
	if n.loaded && !slices.Contains(visible, n.focus) { // before the first read, the cluster it is to go to
		n.focus = navSectionRoot
	}
	n.root.update(len(n.contexts))
	n.list.Objects = objs
	n.scroll.Refresh()
}

// hintText is the line below the clusters when there is something to say.
func (n *navSections) hintText(shown int) string {
	switch {
	case !n.loaded:
		return "Reading the kubeconfigs…"
	case shown > 0 && n.loadErr != "":
		return "Some kubeconfigs could not be read: " + n.loadErr
	case shown > 0:
		return ""
	case n.loadErr != "":
		return "Could not read the kubeconfigs: " + n.loadErr
	case n.s.filter != "":
		return "No clusters match “" + n.s.filter + "”."
	}
	return "No clusters yet: right-click Local Kubeconfigs to add a kubeconfig."
}

// marked reports whether a context is the active cluster: connected or connecting.
func (n *navSections) marked(id string) bool {
	ui := n.s.sh.ui
	return id == ui.Context && (ui.Connection.State == engine.Connected || ui.Connection.State == engine.Connecting)
}

// dotColor is a context's connection dot: the active one's state, red after a failed connect, the
// unknown state's grey otherwise.
func (n *navSections) dotColor(c engine.ContextInfo) fyne.ThemeColorName {
	ui := n.s.sh.ui
	switch {
	case c.Name == ui.Context:
		return connectionColor(ui.Connection)
	case c.ConnectProblem:
		return statusColorName(model.StatusError)
	}
	return connectionColor(engine.Connection{})
}

// drill shows a cluster's tree (ui.setNavigatorScope). A cluster not connected is connected through
// the engine as the picker connects it: a login program the user has not allowed yet, or that
// changed, is shown for approval first (kubeconfigs.go), and nothing runs without it.
func (n *navSections) drill(id string) {
	sh := n.s.sh
	if !n.marked(id) {
		if c, ok := n.context(id); ok && (c.Exec == engine.ExecRequired || c.Exec == engine.ExecChanged) {
			sh.kc.reviewPlugin(id, func(allowed bool) {
				if allowed {
					n.dispatchScope(id)
				}
			})
			return
		}
	}
	n.dispatchScope(id)
}

func (n *navSections) dispatchScope(id string) {
	n.s.setScope(engine.NavScopeCluster(id))
}

// openFirstMatch drills into the first cluster the search shows (↵ in the search field).
func (n *navSections) openFirstMatch() {
	if len(n.visible) > 1 {
		n.setFocus(n.visible[1])
		n.drill(n.visible[1])
	}
}

// shownFrom is called when the section level shows: the keyboard goes to the cluster the navigator
// came from, and the contexts are read again (unless the first read is still under way).
func (n *navSections) shownFrom(cluster string) {
	if cluster != "" {
		n.focus = cluster
	}
	if n.loaded {
		n.load()
	}
	n.render()
}

// rowOf returns the row object of a shown id.
func (n *navSections) rowOf(id string) fyne.CanvasObject {
	if id == navSectionRoot {
		return n.root
	}
	if r := n.rows[id]; r != nil {
		return r
	}
	return nil
}

func (n *navSections) refreshRow(id string) {
	if r := n.rowOf(id); r != nil {
		r.Refresh()
	}
}

// setFocus moves the keyboard to a shown row and scrolls it into view.
func (n *navSections) setFocus(id string) {
	if id != n.focus {
		old := n.focus
		n.focus = id
		n.refreshRow(old)
		n.refreshRow(id)
	}
	row := n.rowOf(id)
	if row == nil || !row.Visible() {
		return
	}
	y, h, view, off := n.list.Position().Y+row.Position().Y, row.Size().Height, n.scroll.Size().Height, n.scroll.Offset.Y
	switch {
	case view <= 0:
	case y < off:
		n.scroll.ScrollToOffset(fyne.NewPos(0, y))
	case y+h > off+view:
		n.scroll.ScrollToOffset(fyne.NewPos(0, y+h-view))
	}
}

// step moves the keyboard d rows down (up for a negative d), stopping at the ends.
func (n *navSections) step(d int) {
	i := slices.Index(n.visible, n.focus)
	switch {
	case i < 0:
		i = 0
	case i+d >= 0 && i+d < len(n.visible):
		i += d
	}
	n.setFocus(n.visible[i])
}

// tapped is a click on a row: the list takes the keyboard there, and the cluster's tree shows.
func (n *navSections) tapped(id string) {
	n.setFocus(id)
	if c := n.s.sh.win.Canvas(); c.Focused() != n {
		c.Focus(n)
	}
	if id != navSectionRoot {
		n.drill(id)
	}
}

// enter moves the keyboard into the list from the search field: to the first match.
func (n *navSections) enter() {
	if len(n.visible) > 1 {
		n.focus = n.visible[1]
	}
	n.s.sh.win.Canvas().Focus(n)
	n.setFocus(n.focus)
}

// TypedKey implements fyne.Focusable: the section's keyboard.
func (n *navSections) TypedKey(ev *fyne.KeyEvent) {
	if n.s.sh.hot.key(ev.Name, n.shift) { // F1–F12 and ⇧F1–⇧F12 belong to the hotbar
		return
	}
	switch ev.Name {
	case fyne.KeyUp:
		n.step(-1)
	case fyne.KeyDown:
		n.step(1)
	case fyne.KeyHome:
		n.setFocus(n.visible[0])
	case fyne.KeyEnd:
		n.setFocus(n.visible[len(n.visible)-1])
	case fyne.KeyRight, fyne.KeyReturn, fyne.KeyEnter, fyne.KeySpace:
		switch {
		case n.focus != navSectionRoot:
			n.drill(n.focus)
		case ev.Name == fyne.KeyRight:
			n.step(1) // from the root to the first cluster, as the tree's → does
		}
	case fyne.KeyLeft:
		if n.focus != navSectionRoot {
			n.setFocus(navSectionRoot) // nothing above the section: the root is as far up as it goes
		}
	case fyne.KeyEscape:
		if n.s.head.searching() {
			n.s.head.closeSearch()
			return
		}
		n.s.sh.closeDrawer()
	}
}

// TypedRune implements fyne.Focusable: a letter typed in the list starts a search with it.
func (n *navSections) TypedRune(r rune) {
	if unicode.IsPrint(r) && !unicode.IsSpace(r) {
		n.s.head.openSearch(n.s.filter + string(r))
	}
}

// KeyDown implements desktop.Keyable: it tracks Shift, which F-keys reach TypedKey without.
func (n *navSections) KeyDown(ev *fyne.KeyEvent) {
	if ev.Name == desktop.KeyShiftLeft || ev.Name == desktop.KeyShiftRight {
		n.shift = true
	}
}

// KeyUp implements desktop.Keyable.
func (n *navSections) KeyUp(ev *fyne.KeyEvent) {
	if ev.Name == desktop.KeyShiftLeft || ev.Name == desktop.KeyShiftRight {
		n.shift = false
	}
}

// FocusGained implements fyne.Focusable: the keyboard starts on the active cluster's row, else on
// the first cluster.
func (n *navSections) FocusGained() {
	n.focused = true
	if !slices.Contains(n.visible, n.focus) || n.focus == navSectionRoot {
		n.focus = navSectionRoot
		for _, id := range n.visible[1:] {
			if n.marked(id) {
				n.focus = id
			}
		}
		if n.focus == navSectionRoot && len(n.visible) > 1 {
			n.focus = n.visible[1]
		}
	}
	n.setFocus(n.focus)
	n.refreshRow(n.focus)
}

// FocusLost implements fyne.Focusable.
func (n *navSections) FocusLost() {
	n.focused, n.shift = false, false
	n.refreshRow(n.focus)
}

// keyboardOn reports whether the ring is on a row.
func (n *navSections) keyboardOn(id string) bool { return n.focused && n.focus == id }

// rowMenu is a cluster's menu (right-click on its row): Connect while it is not the active one,
// Settings…, the hotbar's item, Disconnect, Remove from Cogpit, as its menus elsewhere have them.
func (n *navSections) rowMenu(id string, pos fyne.Position) {
	sh := n.s.sh
	var items []*fyne.MenuItem
	if !n.marked(id) {
		items = append(items, fyne.NewMenuItem("Connect", func() { n.drill(id) }))
	}
	for _, item := range append([]*fyne.MenuItem{sh.settingsItem(id), sh.hot.contextItem(id)}, sh.clusterItems(id)...) {
		if item != nil {
			items = append(items, item)
		}
	}
	if len(items) == 0 {
		return
	}
	if items[0].IsSeparator {
		items = items[1:]
	}
	n.showMenu(items, pos)
}

func (n *navSections) showMenu(items []*fyne.MenuItem, pos fyne.Position) {
	sh := n.s.sh
	sh.tip.hide()
	n.menuModel = fyne.NewMenu("", items...)
	widget.NewPopUpMenu(n.menuModel, sh.win.Canvas()).ShowAtPosition(pos)
}

// navSectionRootRow is the section's root: "Local Kubeconfigs" with "Kubernetes Clusters" and the
// number of clusters below it. Its menu (right-click) adds kubeconfigs.
type navSectionRootRow struct {
	widget.BaseWidget
	n        *navSections
	icon     *widget.Icon
	title    *textLine
	sub      *textLine
	body     fyne.CanvasObject
	bg, ring *canvas.Rectangle
	hovered  bool
}

var (
	_ fyne.Tappable          = (*navSectionRootRow)(nil)
	_ fyne.SecondaryTappable = (*navSectionRootRow)(nil)
	_ desktop.Hoverable      = (*navSectionRootRow)(nil)
)

func newNavSectionRootRow(n *navSections) *navSectionRootRow {
	r := &navSectionRootRow{n: n, bg: canvas.NewRectangle(color.Transparent), ring: canvas.NewRectangle(color.Transparent)}
	r.icon = widget.NewIcon(themedIcon(theme.FolderOpenIcon(), theme.ColorNameForeground))
	r.title = newTextLine(navSectionTitle, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	r.sub = newTextLine(navSectionParent, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	r.title.truncate, r.sub.truncate = true, true
	side := sized(sizeNavBadge, sizeNavBadge, container.NewCenter(r.icon)) // the clusters' badges' column
	r.body = inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewBorder(nil, nil,
		container.NewCenter(side), nil, container.NewVBox(layout.NewSpacer(), r.title, r.sub, layout.NewSpacer())))
	r.ExtendBaseWidget(r)
	return r
}

// update shows the number of clusters the section holds.
func (r *navSectionRootRow) update(clusters int) {
	text := navSectionParent
	if r.n.loaded {
		text += " · " + strconv.Itoa(clusters) + " cluster" + plural(clusters)
	}
	r.sub.SetText(text)
}

// Tapped implements fyne.Tappable: the keyboard goes to the root.
func (r *navSectionRootRow) Tapped(*fyne.PointEvent) {
	r.n.s.sh.tip.hideFor(r)
	r.n.tapped(navSectionRoot)
}

// TappedSecondary implements fyne.SecondaryTappable: Add Kubeconfigs… (the Sync Files and Folders
// dialog; Cogpit's, inferred).
func (r *navSectionRootRow) TappedSecondary(ev *fyne.PointEvent) {
	sh := r.n.s.sh
	r.n.showMenu([]*fyne.MenuItem{fyne.NewMenuItem("Add Kubeconfigs…", sh.kc.addKubeconfigs)}, ev.AbsolutePosition)
}

// MouseIn implements desktop.Hoverable.
func (r *navSectionRootRow) MouseIn(ev *desktop.MouseEvent) {
	r.hovered = true
	r.Refresh()
	r.n.s.sh.tip.hover(r, navSectionTitle+" - "+navSectionParent, ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (r *navSectionRootRow) MouseMoved(ev *desktop.MouseEvent) {
	r.n.s.sh.tip.moved(r, ev.AbsolutePosition)
}

// MouseOut implements desktop.Hoverable.
func (r *navSectionRootRow) MouseOut() {
	r.hovered = false
	r.Refresh()
	r.n.s.sh.tip.hideFor(r)
}

// CreateRenderer implements fyne.Widget.
func (r *navSectionRootRow) CreateRenderer() fyne.WidgetRenderer {
	rr := &rowBodyRenderer{w: r, body: r.body, bg: r.bg, ring: r.ring, paint: func() {
		paintRow(r.Theme(), r.bg, r.ring, false, r.hovered, r.n.keyboardOn(navSectionRoot))
	}}
	rr.Refresh()
	return rr
}

// navContextRow is one cluster of the section: its badge, display name, connection dot and ›.
type navContextRow struct {
	widget.BaseWidget
	n        *navSections
	id       string
	info     engine.ContextInfo
	badge    *clusterBadge
	title    *textLine
	dot      *dot
	next     *navIconButton
	body     fyne.CanvasObject
	bg, ring *canvas.Rectangle
	hovered  bool
}

var (
	_ fyne.Tappable          = (*navContextRow)(nil)
	_ fyne.SecondaryTappable = (*navContextRow)(nil)
	_ desktop.Hoverable      = (*navContextRow)(nil)
)

func newNavContextRow(n *navSections, id string) *navContextRow {
	r := &navContextRow{n: n, id: id, badge: newClusterBadge(), dot: newDot(connectionColor(engine.Connection{})),
		bg: canvas.NewRectangle(color.Transparent), ring: canvas.NewRectangle(color.Transparent)}
	r.title = newTextLine("", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{})
	r.title.truncate = true
	r.next = newNavIconButton(n.s.sh, iconChevronRight, "", func() { n.tapped(id) })
	r.body = inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewBorder(nil, nil,
		container.NewHBox(container.NewCenter(r.badge), newSpace(theme.SizeNamePadding)),
		container.NewHBox(container.NewCenter(r.dot), container.NewCenter(r.next)),
		container.NewVBox(layout.NewSpacer(), r.title, layout.NewSpacer())))
	r.ExtendBaseWidget(r)
	return r
}

// set shows a context as listed now.
func (r *navContextRow) set(c engine.ContextInfo) {
	n := r.n
	r.info = c
	r.badge.setIdentity(n.s.sh.identity(c.Name))
	title := n.rowTitle(c)
	r.title.SetText(title)
	if bold := n.marked(c.Name); bold != r.title.style.Bold {
		r.title.style.Bold = bold
		r.title.Refresh()
	}
	r.dot.SetColor(n.dotColor(c))
	r.next.tip = "Show " + title
	r.Refresh()
}

// tipText is the row's tooltip: its name and the kubeconfig source it comes from.
func (r *navContextRow) tipText() string {
	title := r.n.rowTitle(r.info)
	switch {
	case r.info.Source != "":
		return title + " - " + kubeconfig.DisplayText(r.info.Source, maxShownPath)
	case r.info.Demo:
		return demoLabel
	}
	return title
}

// Tapped implements fyne.Tappable: the cluster's tree.
func (r *navContextRow) Tapped(*fyne.PointEvent) {
	r.n.s.sh.tip.hideFor(r)
	r.n.tapped(r.id)
}

// TappedSecondary implements fyne.SecondaryTappable: the cluster's menu.
func (r *navContextRow) TappedSecondary(ev *fyne.PointEvent) { r.n.rowMenu(r.id, ev.AbsolutePosition) }

// MouseIn implements desktop.Hoverable.
func (r *navContextRow) MouseIn(ev *desktop.MouseEvent) {
	r.hovered = true
	r.Refresh()
	r.n.s.sh.tip.hover(r, r.tipText(), ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (r *navContextRow) MouseMoved(ev *desktop.MouseEvent) {
	r.n.s.sh.tip.moved(r, ev.AbsolutePosition)
}

// MouseOut implements desktop.Hoverable.
func (r *navContextRow) MouseOut() {
	r.hovered = false
	r.Refresh()
	r.n.s.sh.tip.hideFor(r)
}

// CreateRenderer implements fyne.Widget.
func (r *navContextRow) CreateRenderer() fyne.WidgetRenderer {
	rr := &rowBodyRenderer{w: r, body: r.body, bg: r.bg, ring: r.ring, paint: func() {
		paintRow(r.Theme(), r.bg, r.ring, r.n.marked(r.id), r.hovered, r.n.keyboardOn(r.id))
	}}
	rr.Refresh()
	return rr
}

// rowBodyRenderer draws a row's background, its body and its focus ring, painted by paint.
type rowBodyRenderer struct {
	w        fyne.Widget
	body     fyne.CanvasObject
	bg, ring *canvas.Rectangle
	paint    func()
}

func (rr *rowBodyRenderer) MinSize() fyne.Size { return rr.body.MinSize() }

func (rr *rowBodyRenderer) Layout(size fyne.Size) {
	for _, o := range []fyne.CanvasObject{rr.bg, rr.ring, rr.body} {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

func (rr *rowBodyRenderer) Refresh() {
	rr.paint()
	rr.Layout(rr.w.Size())
	rr.bg.Refresh()
	rr.ring.Refresh()
	rr.body.Refresh()
}

func (rr *rowBodyRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{rr.bg, rr.body, rr.ring}
}

func (rr *rowBodyRenderer) Destroy() {}
