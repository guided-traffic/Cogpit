package fyneui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Timings of the shell. They are behaviour, not styling, so they live here and not in the theme.
const (
	searchDebounce   = 150 * time.Millisecond // typing pause before the filter is sent
	ageRefresh       = 10 * time.Second       // how often age cells are re-rendered
	toastLifetime    = 5 * time.Second
	drawerOffset     = 0.55 // the table's share of the width while the drawer is open (drawer ≈ 45%)
	dockOffset       = 0.66 // the workspace's share of the height while the dock is open
	diagnosticsEvery = 250 * time.Millisecond
)

// shell is the window's content and the glue between engine state and widgets.
//
// Threading: every field without a comment saying otherwise is touched only on Fyne's main
// goroutine. Engine events arrive on engine goroutines and are handed over with sh.do; commands
// are dispatched from goroutines, never from the main goroutine, because a command may wait for
// the network or for a confirmation that itself needs the main goroutine.
type shell struct {
	eng  Engine
	sess *engine.Session // the real session, for Command.Enabled; nil with test fakes
	app  fyne.App
	win  fyne.Window
	opts Options
	do   func(func()) // runs a function on the main goroutine: fyne.Do, a queue in tests
	now  func() time.Time

	ctx    context.Context // ends when the window closes; bounds every dispatch
	cancel context.CancelFunc

	ui         engine.UIState // the state the widgets show
	uiApplied  bool
	kinds      []kindInfo
	kindsFor   string // the context kinds were loaded for
	kindsSeq   int    // the latest kinds.list request; older results are dropped
	namespaces []string
	diag       diagView

	// Coalescing of engine events: the latest UI state wins, and at most one table, detail and
	// diagnostics refresh is queued at a time. Guarded by pendingMu or atomic.
	pendingMu     sync.Mutex
	pendingUI     *engine.UIState
	tableQueued   atomic.Bool
	detailQueued  atomic.Bool
	kindsQueued   atomic.Bool
	diagQueued    atomic.Bool
	lastDiagFetch atomic.Int64 // UnixNano of the last diagnostics fetch

	top       *topBar
	side      *sidebar
	view      *tableView
	drawer    *drawer
	dock      *dock
	status    *statusBar
	toasts    *fyne.Container
	tip       *tooltip // the one tooltip: cut-off cells, warning icons, drawer actions
	picker    *picker
	confirmer *guiConfirmer
	drains    *drainBanner // unfinished drains, below the top bar (drains.go)
	helm      *helmBanner  // unfinished Helm release actions, below the drains (helmbanner.go)

	hsplit    *container.Split // table | drawer
	vsplit    *container.Split // workspace / dock
	workspace *fyne.Container  // the table view, or hsplit while the drawer is open
	body      *fyne.Container  // the workspace, with the dock below it
	root      fyne.CanvasObject

	unsubscribe func()
	stopTickers chan struct{}
	perf        *perfMeter
}

// newShell builds every widget. Nothing talks to the engine until start.
func newShell(eng Engine, a fyne.App, w fyne.Window, opts Options, do func(func())) *shell {
	sh := &shell{eng: eng, app: a, win: w, opts: opts, do: do, now: time.Now, stopTickers: make(chan struct{})}
	sh.sess, _ = eng.(*engine.Session)
	sh.ctx, sh.cancel = context.WithCancel(context.Background())

	sh.top = newTopBar(sh)
	sh.side = newSidebar(sh)
	sh.view = newTableView(sh)
	sh.drawer = newDrawer(sh)
	sh.dock = newDock(sh)
	sh.status = newStatusBar(sh)
	sh.picker = newPicker(sh)
	sh.confirmer = newGUIConfirmer(sh)
	sh.drains = newDrainBanner(sh)
	sh.helm = newHelmBanner(sh)
	sh.toasts = box(&toastLayout{})
	sh.tip = newTooltip(sh)

	sh.hsplit = container.NewHSplit(sh.view.root, sh.drawer.root)
	sh.hsplit.SetOffset(drawerOffset)
	sh.vsplit = container.NewVSplit(widget.NewLabel(""), widget.NewLabel(""))
	sh.vsplit.SetOffset(dockOffset)
	sh.workspace = container.NewStack(sh.view.root)
	sh.body = container.NewStack()
	sh.layoutBody()

	sidebar := sized(sizeSidebarWidth, "", newSurface(colorSidebar), sh.side.root)
	main := container.NewBorder(nil, nil, container.NewHBox(sidebar, newDivider(true)), nil, sh.body)
	frame := container.NewBorder(
		container.NewVBox(sh.top.root, newDivider(false), sh.drains.root, sh.helm.root),
		container.NewVBox(newDivider(false), sh.status.root),
		nil, nil, main)
	sh.root = container.NewStack(frame, sh.tip.layer, sh.toasts)
	return sh
}

// start subscribes to the engine, shows the current state and connects or opens the picker.
func (sh *shell) start() {
	sh.unsubscribe = sh.eng.Subscribe(sh.onEvent)
	sh.applyUIState(sh.eng.UIState())
	sh.requestDiagnostics()
	sh.installShortcuts()
	go sh.tick()

	switch {
	case sh.opts.AutoConnect != "":
		name := sh.opts.AutoConnect
		sh.dispatch("cluster.connect", contextParams{Context: name}, func(_ any, err error) {
			if err != nil {
				sh.picker.show()
				sh.picker.setError(name, err)
			}
		})
	case sh.ui.Connection.State != engine.Connected:
		sh.picker.show()
	}
}

// stop ends everything start began: the subscription, tickers, dispatches and dock streams.
func (sh *shell) stop() {
	if sh.unsubscribe != nil {
		sh.unsubscribe()
	}
	select {
	case <-sh.stopTickers:
	default:
		close(sh.stopTickers)
	}
	sh.cancel()
	sh.dock.closeAll()
}

// tick re-renders age cells every ageRefresh, so "5m" becomes "6m" without an engine event.
func (sh *shell) tick() {
	t := time.NewTicker(ageRefresh)
	defer t.Stop()
	for {
		select {
		case <-sh.stopTickers:
			return
		case <-t.C:
			sh.do(sh.view.refreshAges)
		}
	}
}

// onEvent runs on engine goroutines. It must not block and must not touch widgets: it coalesces
// and hands over to the main goroutine.
func (sh *shell) onEvent(ev engine.Event) {
	switch ev.Type {
	case engine.EventUIState:
		if ev.UIState == nil {
			return
		}
		st := ev.UIState.Clone()
		sh.pendingMu.Lock()
		queued := sh.pendingUI != nil
		if sh.pendingUI == nil || st.Revision >= sh.pendingUI.Revision {
			sh.pendingUI = &st
		}
		sh.pendingMu.Unlock()
		if !queued {
			sh.do(sh.flushUIState)
		}
	case engine.EventTable:
		if sh.tableQueued.CompareAndSwap(false, true) {
			sh.do(func() {
				sh.tableQueued.Store(false)
				sh.refreshTable()
			})
		}
	case engine.EventDetail:
		if sh.detailQueued.CompareAndSwap(false, true) {
			sh.do(func() {
				sh.detailQueued.Store(false)
				sh.drawer.request()
			})
		}
	case engine.EventDiagnostics:
		sh.requestDiagnostics()
	case engine.EventKinds:
		// A custom resource definition changed: reload the kinds for the sidebar and the View menu.
		if sh.kindsQueued.CompareAndSwap(false, true) {
			sh.do(func() {
				sh.kindsQueued.Store(false)
				if sh.ui.Connection.State == engine.Connected {
					sh.loadKinds()
				}
			})
		}
	case engine.EventToast:
		level, msg := ev.Level, ev.Message
		sh.do(func() { sh.toast(level, msg) })
	case engine.EventDrains:
		sh.drains.request()
	case engine.EventHelm:
		sh.helm.request()
	case engine.EventEditor:
		id, rev := ev.View, ev.Revision
		sh.do(func() { sh.dock.editorEvent(id, rev) })
	}
}

// flushUIState applies the newest queued UI state.
func (sh *shell) flushUIState() {
	sh.pendingMu.Lock()
	st := sh.pendingUI
	sh.pendingUI = nil
	sh.pendingMu.Unlock()
	if st != nil {
		sh.applyUIState(*st)
	}
}

// requestDiagnostics reads the counters off the main goroutine, at most every diagnosticsEvery,
// and shows them. It may be called from any goroutine.
func (sh *shell) requestDiagnostics() {
	if !sh.diagQueued.CompareAndSwap(false, true) {
		return
	}
	go func() {
		defer sh.recoverTo("diagnostics")
		wait := time.Duration(sh.lastDiagFetch.Load()+int64(diagnosticsEvery)) - time.Duration(time.Now().UnixNano())
		if wait > 0 {
			select {
			case <-time.After(wait):
			case <-sh.ctx.Done():
				return
			}
		}
		sh.lastDiagFetch.Store(time.Now().UnixNano())
		sh.diagQueued.Store(false)
		dv := newDiagView(sh.eng.Diagnostics())
		sh.do(func() {
			sh.diag = dv
			sh.status.update()
		})
	}()
}

// applyUIState makes the widgets show st. Revisions only move forward: an older state that
// arrives late from another engine goroutine is dropped.
func (sh *shell) applyUIState(st engine.UIState) {
	if sh.uiApplied && st.Revision < sh.ui.Revision {
		return
	}
	old := sh.ui
	first := !sh.uiApplied
	sh.ui, sh.uiApplied = st, true

	if first || st.Context != old.Context || st.Connection != old.Connection {
		sh.top.update()
		sh.status.update()
		sh.side.updateHeader()
		sh.view.updatePlaceholder()
		if st.Connection.State == engine.Connected && (sh.kindsFor != st.Context) {
			sh.loadKinds()
			sh.loadNamespaces()
			sh.picker.connected(st.Context)
		}
		if st.Connection.State != engine.Connected && old.Connection.State == engine.Connected {
			sh.kinds, sh.kindsFor, sh.namespaces = nil, "", nil
			sh.side.setKinds(nil)
			sh.win.SetMainMenu(sh.mainMenu())
		}
	}
	if first || st.View != old.View {
		sh.side.selectView(st.View)
		sh.view.viewChanged()
		sh.refreshTable()
	}
	if first || !slices.Equal(st.Namespaces, old.Namespaces) || st.Filter != old.Filter || st.Sort != old.Sort {
		sh.view.stateChanged()
	}
	if first || !slices.Equal(st.Selected, old.Selected) {
		sh.view.selectionChanged(st.Selected)
	}
	if first || !sameRef(st.Drawer, old.Drawer) {
		sh.drawerChanged()
	}
	if first || !dockEqual(st, old) {
		sh.dock.sync(st.Dock, st.ActiveDock)
		sh.layoutBody()
	}
}

// refreshTable shows the newest snapshot of the open view.
func (sh *shell) refreshTable() {
	if sh.ui.View == "" {
		sh.view.setSnapshot(nil)
		return
	}
	sh.view.setSnapshot(sh.eng.Table(sh.ui.View))
}

// drawerChanged opens, closes or retargets the detail drawer.
func (sh *shell) drawerChanged() {
	if sh.ui.Drawer == nil {
		sh.drawer.clear()
		sh.workspace.Objects = []fyne.CanvasObject{sh.view.root}
	} else {
		sh.hsplit.Leading, sh.hsplit.Trailing = sh.view.root, sh.drawer.root
		sh.workspace.Objects = []fyne.CanvasObject{sh.hsplit}
		sh.hsplit.Refresh()
		sh.drawer.request()
	}
	sh.workspace.Refresh()
}

// layoutBody puts the dock below the workspace: a resizable split while the dock is open, only
// its tab strip while it is collapsed.
func (sh *shell) layoutBody() {
	if sh.dock.open() {
		sh.vsplit.Leading, sh.vsplit.Trailing = sh.workspace, sh.dock.panel
		sh.dock.attachStrip(true)
		sh.body.Objects = []fyne.CanvasObject{sh.vsplit}
		sh.vsplit.Refresh()
	} else {
		sh.dock.attachStrip(false)
		sh.body.Objects = []fyne.CanvasObject{container.NewBorder(nil, sh.dock.collapsedBar, nil, nil, sh.workspace)}
	}
	sh.body.Refresh()
}

// loadKinds fetches the connected cluster's kinds for the sidebar and the View menu.
func (sh *shell) loadKinds() {
	ctxName := sh.ui.Context
	sh.kindsFor = ctxName
	sh.kindsSeq++
	seq := sh.kindsSeq
	sh.dispatch("kinds.list", struct{}{}, func(res any, err error) {
		if sh.ui.Context != ctxName || seq != sh.kindsSeq {
			return
		}
		if err != nil {
			sh.toast("error", "Could not list the cluster's kinds: "+err.Error())
			return
		}
		kinds, derr := decodeKinds(res)
		if derr != nil {
			sh.toast("error", "Unexpected kinds.list result: "+derr.Error())
			return
		}
		sh.kinds = kinds
		sh.side.setKinds(kinds)
		sh.side.selectView(sh.ui.View)
		sh.view.updateTitle()
		sh.win.SetMainMenu(sh.mainMenu())
	})
}

// loadNamespaces fetches the namespace names for the namespace filter.
func (sh *shell) loadNamespaces() {
	ctxName := sh.ui.Context
	sh.dispatch("resources.list", listParams{Kind: kindNamespaces}, func(res any, err error) {
		if err != nil || sh.ui.Context != ctxName {
			return // the filter then offers "All namespaces" only; nothing to tell the user
		}
		names, derr := decodeNames(res)
		if derr == nil {
			sh.namespaces = names
			sh.view.namespacesLoaded()
		}
	})
}

// kindTitle returns the display title of a kind key: from kinds.list, else the key itself.
func (sh *shell) kindTitle(key string) string {
	for _, k := range sh.kinds {
		if k.Kind == key {
			return k.Title
		}
	}
	if key == "" {
		return ""
	}
	return strings.ToUpper(key[:1]) + key[1:]
}

// dispatch runs a command off the main goroutine, then calls done with its result on the main
// goroutine. Without done, an error is shown as a toast; a declined confirmation is not an error.
func (sh *shell) dispatch(id string, params any, done func(any, error)) {
	raw, err := json.Marshal(params)
	if err != nil {
		sh.toast("error", fmt.Sprintf("%s: %v", id, err))
		return
	}
	go func() {
		defer sh.recoverTo(id)
		res, err := sh.eng.Dispatch(sh.ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginGUI})
		if done == nil && (err == nil || errors.Is(err, engine.ErrDeclined) || sh.ctx.Err() != nil) {
			return
		}
		sh.do(func() {
			if done != nil {
				done(res, err)
				return
			}
			sh.showError(err)
		})
	}()
}

// showError shows a dispatch error as a toast; a declined confirmation is the user's answer.
func (sh *shell) showError(err error) {
	if err == nil || errors.Is(err, engine.ErrDeclined) {
		return
	}
	var dis *engine.DisabledError
	if errors.As(err, &dis) {
		sh.toast("warning", dis.Reason)
		return
	}
	sh.toast("error", err.Error())
}

// recoverTo turns a panic in a shell goroutine into a toast instead of ending the process (D-11).
func (sh *shell) recoverTo(what string) {
	if r := recover(); r != nil {
		msg := fmt.Sprintf("internal error in %s: %v", what, r)
		sh.do(func() { sh.toast("error", msg) })
	}
}

// commandEnabled evaluates a command's Enabled for params: nil when it can run, an error whose
// text is the reason otherwise. Call it off the main goroutine.
func (sh *shell) commandEnabled(id string, params any) (err error) {
	cmd, ok := sh.eng.Registry().Get(id)
	if !ok {
		return errUnavailable
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("unavailable: %v", r)
		}
	}()
	return cmd.Enabled(sh.sess, raw)
}

// errUnavailable marks a command this build does not register; its button is hidden.
var errUnavailable = errors.New("not available in this build")

// toast shows a short message in the bottom-right corner for toastLifetime.
func (sh *shell) toast(level, msg string) {
	if msg == "" {
		return
	}
	c := colorAccent
	icon := theme.InfoIcon()
	switch level {
	case "error":
		c, icon = colorStatusError, theme.ErrorIcon()
	case "warning":
		c, icon = colorStatusWarning, theme.WarningIcon()
	}
	th := sh.app.Settings().Theme()
	pad := th.Size(theme.SizeNamePadding)
	text := newParagraph(msg, theme.ColorNameForeground,
		th.Size(sizeToastWidth)-th.Size(theme.SizeNameInlineIcon)-6*pad)
	bg := newSurface(colorToast)
	bg.radius, bg.stroke = theme.SizeNameInputRadius, theme.ColorNameInputBorder
	t := container.NewStack(
		bg,
		inset(theme.SizeNamePadding, theme.SizeNamePadding,
			container.NewBorder(nil, nil, container.NewCenter(widget.NewIcon(themedIcon(icon, c))), nil, text)),
	)
	sh.toasts.Objects = append(sh.toasts.Objects, t)
	sh.toasts.Refresh()
	time.AfterFunc(toastLifetime, func() {
		sh.do(func() {
			sh.toasts.Objects = slices.DeleteFunc(sh.toasts.Objects, func(o fyne.CanvasObject) bool { return o == t })
			sh.toasts.Refresh()
		})
	})
}

// toastLayout stacks toasts upwards from the bottom-right corner, above the status bar.
type toastLayout struct{}

func (toastLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (toastLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	th := fyne.CurrentApp().Settings().Theme()
	pad := th.Size(theme.SizeNamePadding) * 3
	w := th.Size(sizeToastWidth)
	y := size.Height - th.Size(sizeStatusHeight) - pad
	for i := len(objs) - 1; i >= 0; i-- {
		o := objs[i]
		h := o.MinSize().Height
		y -= h
		o.Move(fyne.NewPos(size.Width-w-pad, y))
		o.Resize(fyne.NewSize(w, h))
		y -= pad / 2
	}
}

// mainMenu builds the window menu: Cluster, View (one item per kind) and Help.
func (sh *shell) mainMenu() *fyne.MainMenu {
	cluster := fyne.NewMenu("Cluster", fyne.NewMenuItem("Switch Context…", sh.picker.show))
	var items []*fyne.MenuItem
	for _, k := range sh.kinds {
		key := k.Kind
		items = append(items, fyne.NewMenuItem(k.Title, func() {
			sh.dispatch("ui.openView", openViewParams{View: key}, nil)
		}))
	}
	if len(items) == 0 {
		none := fyne.NewMenuItem("No cluster connected", nil)
		none.Disabled = true
		items = append(items, none)
	}
	menus := []*fyne.Menu{cluster, fyne.NewMenu("View", items...)}
	if sh.opts.MCPCommand != "" {
		menus = append(menus, fyne.NewMenu("Help", fyne.NewMenuItem("Copy MCP command", sh.copyMCPCommand)))
	}
	return fyne.NewMainMenu(menus...)
}

// copyMCPCommand puts the `claude mcp add` command on the clipboard and shows it.
func (sh *shell) copyMCPCommand() {
	cmd := sh.opts.MCPCommand
	sh.app.Clipboard().SetContent(cmd)
	entry := widget.NewEntry()
	entry.SetText(cmd)
	entry.TextStyle = fyne.TextStyle{Monospace: true}
	intro := widget.NewLabel("Copied to the clipboard. Run it in a terminal to let an MCP client drive this Cogpit window:")
	intro.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom("MCP command", "Close", container.NewVBox(intro, entry), sh.win)
	d.Resize(fyne.NewSize(sh.win.Canvas().Size().Width*0.5, 0))
	d.Show()
}

// installShortcuts adds Lens's keyboard shortcuts that the MVP supports: ⌘F (search), ⌘⇧T
// (terminal session), Escape (close the drawer) and "/" (the logs tab's search).
func (sh *shell) installShortcuts() {
	c := sh.win.Canvas()
	c.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierShortcutDefault},
		func(fyne.Shortcut) { c.Focus(sh.view.search) })
	c.AddShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyT, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
		func(fyne.Shortcut) { sh.dispatch("terminal.open", struct{}{}, nil) })
	c.SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if ev.Name == fyne.KeyEscape {
			sh.closeDrawer()
		}
	})
	c.SetOnTypedRune(func(r rune) {
		if r == '/' {
			sh.dock.focusLogSearch() // "/" focuses the active logs tab's search
		}
	})
}

// closeDrawer asks the engine to close the drawer.
func (sh *shell) closeDrawer() {
	if sh.ui.Drawer != nil {
		sh.dispatch("ui.closeDrawer", struct{}{}, nil)
	}
}

// Command parameters, mirroring the engine's command schemas. Slices are never nil: a JSON null
// would not validate against an array schema.
type (
	openViewParams struct {
		View string `json:"view"`
	}
	namespacesParams struct {
		Namespaces []string `json:"namespaces"`
	}
	filterParams struct {
		Filter string `json:"filter"`
	}
	sortParams struct {
		Column string `json:"column"`
		Desc   bool   `json:"desc"`
	}
	selectParams struct {
		Refs     []model.ObjectRef `json:"refs"`
		NoDrawer bool              `json:"noDrawer"`
	}
	refParams struct {
		Ref model.ObjectRef `json:"ref"`
	}
	containerParams struct {
		Ref       model.ObjectRef `json:"ref"`
		Container string          `json:"container"`
	}
	contextParams struct {
		Context string `json:"context"`
	}
	dockParams struct {
		ID string `json:"id"`
	}
	listParams struct {
		Kind string `json:"kind"`
	}
)

// kindInfo is one entry of kinds.list.
type kindInfo struct {
	Kind       string         `json:"kind"`
	Title      string         `json:"title"`
	APIKind    string         `json:"apiKind"`
	Group      string         `json:"group"`
	Version    string         `json:"version"`
	Namespaced bool           `json:"namespaced"`
	Columns    []model.Column `json:"columns"`
	Listable   *bool          `json:"listable,omitempty"` // nil: the access reviews have not answered
}

// decode turns a command result into out through JSON: results cross the engine seam as
// JSON-serializable values, whatever Go type the engine used for them.
func decode(res any, out any) error {
	var raw []byte
	switch r := res.(type) {
	case json.RawMessage:
		raw = r
	case []byte:
		raw = r
	default:
		b, err := json.Marshal(res)
		if err != nil {
			return err
		}
		raw = b
	}
	return json.Unmarshal(raw, out)
}

// decodeKinds reads kinds.list: an array, or an object with a "kinds" array.
func decodeKinds(res any) ([]kindInfo, error) {
	var list []kindInfo
	if err := decode(res, &list); err == nil {
		return list, nil
	}
	var wrapped struct {
		Kinds []kindInfo `json:"kinds"`
	}
	if err := decode(res, &wrapped); err != nil {
		return nil, err
	}
	return wrapped.Kinds, nil
}

// decodeNames reads the object names from a resources.list result.
func decodeNames(res any) ([]string, error) {
	var out struct {
		Rows []struct {
			Ref model.ObjectRef `json:"ref"`
		} `json:"rows"`
	}
	if err := decode(res, &out); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out.Rows))
	for _, r := range out.Rows {
		names = append(names, r.Ref.Name)
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}

// formatCount formats n with thousands separators: 10000 → "10,000".
func formatCount(n int64) string {
	s := strconv.FormatInt(n, 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func sameRef(a, b *model.ObjectRef) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func dockEqual(a, b engine.UIState) bool {
	if a.ActiveDock != b.ActiveDock || len(a.Dock) != len(b.Dock) {
		return false
	}
	for i := range a.Dock {
		x, y := a.Dock[i], b.Dock[i]
		if x.ID != y.ID || x.Title != y.Title || x.Logs != y.Logs || x.Container != y.Container || !sameRef(x.Ref, y.Ref) {
			return false
		}
	}
	return true
}
