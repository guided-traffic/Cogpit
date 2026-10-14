package fyneui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"runtime"
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
	dockOffset       = 0.66                   // the workspace's share of the height while the dock is open
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
	// do runs a function on the main goroutine (fyne.Do, a queue in tests); once the shell stopped,
	// what it queued before and what arrives after is dropped (stopped).
	do      func(func())
	stopped atomic.Bool
	now     func() time.Time
	// saveFile asks where to save a file and writes it (a file save dialog; tests record it).
	saveFile func(name string, rows int, data []byte)

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
	metricsQueued atomic.Bool
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
	palette   *cmdPalette   // the command palette, ⌘⇧P (palette.go)
	hot       *hotbar       // the pinned clusters at the window's left edge (hotbar.go)
	tabs      *tabBar       // the tabs of cluster views above the workspace (tabbar.go)
	kc        *kubeconfigUI // the kubeconfig sources' dialogs (kubeconfigs.go)
	settings  *settingsPage // Preferences and Cluster settings (preferences_page.go)
	confirmer *guiConfirmer
	drains    *drainBanner // unfinished drains, below the top bar (drains.go)
	helm      *helmBanner  // unfinished Helm release actions, below the drains (helmbanner.go)
	titlebar  *titlebar    // Cogpit's own title bar on macOS (titlebar.go)

	drafts *draftsBanner // unsaved editor text kept from an earlier run, below the Helm banner (drafts.go)

	helmHTTP *helmHTTPBanner // Helm repositories waiting to send credentials over http, below the Helm banner (helmhttp.go, H-40)

	hsplit    *fyne.Container // table | drawer, the drawer's width set by the user (panelwidths.go)
	mainRow   *fyne.Container // sidebar | body, the sidebar's width set by the user
	center    *fyne.Container // mainRow, or the open settings page in its place
	sidebarW  float32         // the user's widths of the two panels, stored in the preferences
	drawerW   float32
	sideLay   *panelLayout
	drawerLay *panelLayout
	chat      *chatPanel       // the agent's chat column at the window's right edge (chat.go)
	chatW     float32          // the user's width of the chat column, stored in the preferences
	chatLay   *panelLayout     // the chat column beside chatRow's work area
	chatSplit *fyne.Container  // the work area | the chat column, while the chat is shown
	chatRow   *fyne.Container  // the work area alone, or chatSplit
	vsplit    *container.Split // workspace / dock
	workspace *fyne.Container  // the table view, or hsplit while the drawer is open
	body      *fyne.Container  // the workspace, with the dock below it
	root      fyne.CanvasObject

	workArea fyne.CanvasObject // the workspace with the welcome page over it, not over the dock (auditlog.go)

	// A Secret's value copied from the drawer is cleared from the clipboard after secretClipTTL
	// (copy.go); main goroutine only.
	clipTimer     *time.Timer
	clipSecret    string
	secretClipTTL time.Duration
	// copyConfirm is how long the copy icon shows a checkmark after a copy (a test shortens it).
	copyConfirm time.Duration
	// toastTime, when set, replaces every toast's lifetime (a test shortens it).
	toastTime time.Duration
	// setConcealed puts a Secret's value on the system clipboard marked for clipboard managers to
	// skip (clipboard_darwin.go); nil in tests, which use Fyne's test clipboard and must not touch
	// the system's. It reports false when it could not, and Fyne's clipboard is used.
	setConcealed func(string) bool

	unsubscribe func()
	// early are the notices the engine gave before the window subscribed (a restored tab bar's,
	// D-41), shown as toasts with the start's warning once the window is up (showStartNotices).
	early       []engine.Notification
	stopTickers chan struct{}
	perf        *perfMeter

	mcp     mcpState       // the MCP server as the window knows it (mcp.go)
	mcpMenu *fyne.MenuItem // Help › Copy MCP command, disabled while the server is off

	helmRepos *helmReposDialog // the open Helm repositories dialog (helmcharts.go)
	// zone is the Preferences' time zone for the charts; nil: the clock's (preferences.go).
	zone *time.Location
	// icons are the clusters' uploaded icons by file, nil for one that cannot be read;
	// iconLoading the files being read (clusteridentity.go).
	icons       map[string]fyne.Resource
	iconLoading map[string]bool
	// charts are the Helm charts' icons the window fetched (helmcharticons.go, D-46).
	charts chartIconCache
}

// newShell builds every widget. Nothing talks to the engine until start.
func newShell(eng Engine, a fyne.App, w fyne.Window, opts Options, do func(func())) *shell {
	sh := &shell{eng: eng, app: a, win: w, opts: opts, now: time.Now, stopTickers: make(chan struct{}), secretClipTTL: secretClipboardTTL, copyConfirm: copyConfirmTime}
	// Work that reaches the main goroutine after stop would apply engine state to a window that
	// is gone, possibly under another app and theme (tests run shells one after another).
	sh.do = func(fn func()) {
		do(func() {
			if !sh.stopped.Load() {
				fn()
			}
		})
	}
	sh.mcp = mcpState{on: opts.MCP == nil && opts.MCPCommand != "", command: opts.MCPCommand}
	sh.sess, _ = eng.(*engine.Session)
	sh.ctx, sh.cancel = context.WithCancel(context.Background())
	sh.saveFile = func(name string, rows int, data []byte) { sh.saveWithDialog(name, rows, data, nil) }
	sh.titlebar = newTitlebar(sh)
	sh.loadWidths() // before the drawer, which wraps its values to the width it will have

	sh.top = newTopBar(sh)
	sh.side = newSidebar(sh)
	sh.view = newTableView(sh)
	sh.drawer = newDrawer(sh)
	sh.dock = newDock(sh)
	sh.status = newStatusBar(sh)
	sh.kc = newKubeconfigUI(sh)
	sh.picker = newPicker(sh)
	sh.palette = newCmdPalette(sh)
	sh.settings = newSettingsPage(sh)
	sh.confirmer = newGUIConfirmer(sh)
	sh.drains = newDrainBanner(sh)
	sh.helm = newHelmBanner(sh)
	sh.helmHTTP = newHelmHTTPBanner(sh)
	sh.drafts = newDraftsBanner(sh)
	sh.toasts = box(&toastLayout{})
	sh.tip = newTooltip(sh)
	sh.hot = newHotbar(sh)
	sh.tabs = newTabBar(sh)

	sh.drawerLay = &panelLayout{want: func() float32 { return sh.drawerW }, lo: minDrawerWidth, hi: maxDrawerWidth, restMin: minTableRoom,
		onGot: func(float32) { sh.do(sh.drawer.widthChanged) }}
	sh.hsplit = newPanel(sh.drawerLay, sh.drawer.root, sh.view.root, newResizeHandle(-1,
		func() float32 { return sh.drawerLay.got },
		func(w float32) float32 { return sh.setDrawerWidth(w) },
		sh.saveWidths))
	sh.vsplit = container.NewVSplit(widget.NewLabel(""), widget.NewLabel(""))
	sh.vsplit.SetOffset(dockOffset)
	sh.workspace = container.NewStack(sh.view.root)
	// The welcome page lies over the workspace, not the dock: a dock tab of no cluster (the Audit
	// Log) shows while no cluster is connected.
	sh.workArea = sh.tabs.workspace(sh.workspace)
	sh.body = container.NewStack()
	sh.layoutBody()

	sidebar := box(lockedWidth{}, newSurface(colorSidebar), sh.side.root)
	sh.sideLay = &panelLayout{first: true, want: func() float32 { return sh.sidebarW }, lo: minSidebarWidth, hi: maxSidebarWidth, restMin: minTableRoom}
	main := newPanel(sh.sideLay, sidebar, container.NewBorder(sh.tabs.root, nil, nil, nil, sh.body), newResizeHandle(1,
		func() float32 { return sh.sideLay.got },
		func(w float32) float32 { return sh.setSidebarWidth(w) },
		sh.saveWidths))
	sh.mainRow = main
	sh.center = container.NewStack(main)
	// The agent's chat column, the full height beside the work area, a settings page included
	// (chat.go); its width is the user's, as the sidebar's and the drawer's.
	sh.chat = newChatPanel(sh)
	sh.chatLay = &panelLayout{want: func() float32 { return sh.chatW }, lo: minChatWidth, hi: maxChatWidth,
		restMin: minSidebarWidth + handleWidth + minTableRoom}
	sh.chatSplit = newPanel(sh.chatLay, sh.chat.root, sh.center, newResizeHandle(-1,
		func() float32 { return sh.chatLay.got },
		func(w float32) float32 { return sh.setChatWidth(w) },
		sh.saveWidths))
	sh.chatRow = container.NewStack(sh.center)
	frame := container.NewBorder(
		container.NewVBox(sh.top.root, newDivider(false), sh.drains.root, sh.helm.root, sh.helmHTTP.root, sh.drafts.root),
		container.NewVBox(newDivider(false), sh.status.root),
		sh.hot.root, nil, sh.chatRow)
	sh.root = container.NewStack(frame, sh.tip.layer, sh.toasts)
	return sh
}

// start subscribes to the engine, shows the current state and connects the --context cluster, if
// any. Without one the workspace shows the welcome page and the picker stays closed until the user
// opens it (⌘P, All Clusters…; D-42): nothing connects by itself, a restored tab included (D-41).
func (sh *shell) start() {
	sh.early = sh.earlyNotices() // before subscribing: what comes after arrives as events
	sh.unsubscribe = sh.eng.Subscribe(sh.onEvent)
	sh.applyUIState(sh.eng.UIState())
	sh.requestDiagnostics()
	sh.installShortcuts()
	go sh.tick()

	if name := sh.opts.AutoConnect; name != "" {
		// A --context that fails opens the picker with the error beside the context, where a login
		// program not allowed yet can be allowed (Cogpit's default).
		sh.dispatch("cluster.connect", contextParams{Context: name}, func(_ any, err error) {
			if err != nil {
				sh.picker.show()
				sh.picker.setError(name, err)
			}
		})
	}
}

// earlyNotices returns the unread notifications the engine logged before the window subscribed,
// oldest first; none from an engine that keeps no log (the tests' fake).
func (sh *shell) earlyNotices() []engine.Notification {
	if sh.sess == nil {
		return nil
	}
	var out []engine.Notification
	for _, n := range sh.sess.Notifications(true).Notifications { // newest first
		if !n.Read {
			out = append([]engine.Notification{n}, out...)
		}
	}
	return out
}

// showStartNotices shows, once the window is up, the warning the start brought (a crash report the
// last run left) and the notices the engine gave before the window subscribed (the tabs of the last
// run not restored, an unreadable tabs file), each closing marks its notification read.
func (sh *shell) showStartNotices() {
	if sh.opts.StartWarning != "" {
		sh.toast("warning", sh.opts.StartWarning)
	}
	for _, n := range sh.early {
		sh.showToast(n.Level, n.Message, n.ID)
	}
	sh.early = nil
}

// stop ends everything start began: the subscription, tickers, dispatches and dock streams, and
// drops the main-goroutine work still queued.
func (sh *shell) stop() {
	sh.stopped.Store(true)
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
	sh.titlebar.stop()
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
	case engine.EventMetrics:
		if ev.View == engine.ClusterOverview { // the cluster Overview's charts, not a drawer's
			sh.do(func() {
				if sh.ui.View == engine.ClusterOverview {
					sh.view.cluster.request()
				}
			})
			return
		}
		if sh.metricsQueued.CompareAndSwap(false, true) {
			sh.do(func() {
				sh.metricsQueued.Store(false)
				sh.drawer.metricsChanged()
				sh.settings.metricsChanged() // the Prometheus setting may be new
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
		level, msg, id := ev.Level, ev.Message, ev.Notification
		sh.do(func() { sh.showToast(level, msg, id) }) // the engine keeps it in its notification log
	case engine.EventDrains:
		sh.drains.request()
	case engine.EventHelm:
		sh.helm.request()
	case engine.EventHelmRepos:
		sh.helmHTTP.request() // a repository waiting to send credentials over http (H-40)
	case engine.EventDrafts:
		sh.drafts.request()
	case engine.EventEditor:
		id, rev := ev.View, ev.Revision
		sh.do(func() { sh.dock.editorEvent(id, rev) })
	case engine.EventContexts:
		sh.do(func() {
			sh.kc.contextsChanged()
			sh.hot.request() // a tile's context may have gone, or come back
		})
	case engine.EventAudit: // a line in the audit log: the Audit Log tab reads it (auditlog.go)
		sh.do(sh.dock.auditChanged)
	case engine.EventDockRead: // an agent read a terminal tab: it flashes (dockagentread.go)
		id := ev.View
		sh.do(func() { sh.dock.flashAgentRead(id) })
	case engine.EventChat: // the agent chat: a conversation or the lists changed (chat.go)
		sh.chat.event(ev)
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
			sh.dock.diagnosticsChanged() // an open Diagnostics tab (diagnostics.go)
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
			sh.side.setKinds(nil) // the View menu follows below, with the connection's other items
		}
	}
	if !first && (st.ActiveTab != old.ActiveTab || st.View != old.View) {
		sh.view.dropSearch() // a search typed for one tab or view stays out of the next (tableview.go)
	}
	if first || st.View != old.View {
		sh.side.selectView(st.View)
		sh.view.viewChanged()
		sh.refreshTable()
	}
	if first || tableStateChanged(st, old) {
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
	if first || !sameHotbar(st.Hotbar, old.Hotbar) || st.Context != old.Context || st.Connection != old.Connection {
		sh.hot.stateChanged()
	}
	if !first && st.Overview != old.Overview && st.View == engine.ClusterOverview {
		sh.view.cluster.showControls() // the charts' range or node group, whoever set it
		sh.view.cluster.request()
	}
	if first || st.Palette || old.Palette { // while open, what it lists follows the state
		sh.palette.stateChanged()
	}
	if first || !reflect.DeepEqual(st.Preferences, old.Preferences) {
		sh.preferencesChanged()
	}
	if !first && (!maps.EqualFunc(st.ClusterSettings, old.ClusterSettings, engine.ClusterSettings.Equal) || st.Context != old.Context) {
		sh.identitiesChanged()
	}
	if first || !reflect.DeepEqual(st.Settings, old.Settings) || (st.Settings != nil && (!reflect.DeepEqual(st.Preferences, old.Preferences) ||
		!reflect.DeepEqual(st.LoginItem, old.LoginItem) ||
		!maps.EqualFunc(st.ClusterSettings, old.ClusterSettings, engine.ClusterSettings.Equal) || st.Context != old.Context || st.Connection != old.Connection)) {
		sh.settings.stateChanged()
	}
	if first || (st.Context == "") != (old.Context == "") || st.Connection != old.Connection || st.Chat != old.Chat {
		// Cluster Settings… is enabled while a cluster is chosen, Disconnect while one is connected or
		// connecting (a reconnect too: menus_cluster.go), Refresh Access Reviews while one is connected
		// (menuaccess.go), the View menu's kinds while connected (loadKinds rebuilds it once they come).
		sh.win.SetMainMenu(sh.mainMenu())
	}
	if first || st.Scope() != old.Scope() {
		sh.side.scopeChanged() // the section's clusters or the cluster's tree (navigator_sections.go)
	}
	if first || tabsChanged(st, old) {
		sh.tabs.sync()
	}
	if first || helmChartsShown(st) != helmChartsShown(old) {
		sh.helmHTTP.request() // shown with the Charts view and Helm Install tabs (helmhttp.go)
	}
	sh.chat.stateChanged(old, first) // the chat column: shown, its cluster, the input's state (chat.go)
	sh.top.stateChanged()            // the history arrows and the bell (titlehistory.go)
}

// refreshTable shows the newest snapshot of the open view.
func (sh *shell) refreshTable() {
	if sh.ui.View == "" {
		sh.view.setSnapshot(nil)
		return
	}
	sh.view.setSnapshot(sh.eng.Table(sh.ui.View))
	if engine.IsPage(sh.ui.View) {
		sh.view.requestPage() // the page's kinds or totals changed with its table
	}
}

// drawerChanged opens, closes or retargets the detail drawer.
func (sh *shell) drawerChanged() {
	if sh.ui.Drawer == nil {
		sh.drawer.clear()
		sh.workspace.Objects = []fyne.CanvasObject{sh.view.root}
	} else {
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
		sh.vsplit.Leading, sh.vsplit.Trailing = sh.workArea, sh.dock.panel
		sh.dock.attachStrip(true)
		sh.body.Objects = []fyne.CanvasObject{sh.vsplit}
		sh.vsplit.Refresh()
	} else {
		sh.dock.attachStrip(false)
		sh.body.Objects = []fyne.CanvasObject{container.NewBorder(nil, sh.dock.collapsedBar, nil, nil, sh.workArea)}
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
		sh.view.updateNamespaceButton()
		sh.view.create.update() // local and virtual kinds have no "+"
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

// kindTitle returns the display title of a kind key (from kinds.list, else the key itself) or of
// a page.
func (sh *shell) kindTitle(key string) string {
	if t, ok := engine.PageTitle(key); ok {
		return t
	}
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

// toastLifetimes is how long a toast stays per level; an error stays longest, because it asks for
// a reaction. The pointer over a toast pauses its time.
var toastLifetimes = map[string]time.Duration{"info": 6 * time.Second, "warning": 9 * time.Second, "error": 14 * time.Second}

// toast shows a message the window raised itself and keeps it in the engine's notification log
// (notifications.go), for the window's eyes only.
func (sh *shell) toast(level, msg string) {
	if msg == "" {
		return
	}
	sh.showToast(level, msg, sh.recordToast(level, msg))
}

// showToast shows a message in the bottom-right corner: a card on a strong surface with a bar,
// border and icon in the level's colour, a bold title, the text and a close button, which marks
// its notification (id, 0 for none) read.
func (sh *shell) showToast(level, msg string, id uint64) {
	if msg == "" {
		return
	}
	c, title, icon := toastStyle(level)
	life := toastLifetimes[level]
	if life == 0 {
		life = toastLifetimes["info"]
	}
	if sh.toastTime > 0 {
		life = sh.toastTime
	}
	th := sh.app.Settings().Theme()
	pad := th.Size(theme.SizeNamePadding)
	text := newParagraph(msg, colorToastText,
		th.Size(sizeToastWidth)-2*th.Size(theme.SizeNameInlineIcon)-8*pad-toastBar)
	head := newTextLine(title, colorToastText, theme.SizeNameText, fyne.TextStyle{Bold: true})
	bg := newSurface(colorToast)
	bg.stroke = c
	bar := newSurface(c)
	bar.min = fyne.NewSize(toastBar, 0)
	card := &toastCard{}
	card.ExtendBaseWidget(card)
	closeBtn := newToastClose(func() {
		sh.dismissToast(card)
		sh.markToastRead(id)
	})
	card.life = life
	card.expire = func() { sh.do(func() { sh.dismissToast(card) }) }
	card.content = container.NewStack(
		bg,
		container.NewBorder(nil, nil, bar, nil,
			inset(theme.SizeNamePadding, theme.SizeNamePadding,
				container.NewBorder(nil, nil,
					container.NewVBox(container.NewPadded(widget.NewIcon(themedIcon(icon, c)))),
					container.NewVBox(closeBtn),
					container.NewVBox(head, text)))),
	)
	sh.toasts.Objects = append(sh.toasts.Objects, card)
	sh.toasts.Refresh()
	card.arm()
}

// toastBar is the width of the level's colour bar at a toast's left edge.
const toastBar = 5

// dismissToast removes a toast, on expiry or by its close button.
func (sh *shell) dismissToast(card *toastCard) {
	card.disarm()
	sh.toasts.Objects = slices.DeleteFunc(sh.toasts.Objects, func(o fyne.CanvasObject) bool { return o == card })
	sh.toasts.Refresh()
}

// toastCard is one toast. Its time runs while the pointer is away: entering stops it, leaving
// starts the full lifetime again.
type toastCard struct {
	widget.BaseWidget
	content fyne.CanvasObject
	life    time.Duration
	expire  func() // called on a timer goroutine
	mu      sync.Mutex
	timer   *time.Timer
	hovered bool
}

var _ desktop.Hoverable = (*toastCard)(nil)

func (c *toastCard) arm() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
	}
	if !c.hovered {
		c.timer = time.AfterFunc(c.life, c.expire)
	}
}

func (c *toastCard) disarm() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}
}

// MouseIn implements desktop.Hoverable.
func (c *toastCard) MouseIn(*desktop.MouseEvent) {
	c.mu.Lock()
	c.hovered = true
	c.mu.Unlock()
	c.disarm()
}

// MouseMoved implements desktop.Hoverable.
func (c *toastCard) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (c *toastCard) MouseOut() {
	c.mu.Lock()
	c.hovered = false
	c.mu.Unlock()
	c.arm()
}

// CreateRenderer implements fyne.Widget.
func (c *toastCard) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(c.content) }

// newToastClose is the small × of a toast.
func newToastClose(fn func()) *widget.Button {
	b := widget.NewButtonWithIcon("", themedIcon(theme.CancelIcon(), colorToastText), fn)
	b.Importance = widget.LowImportance
	return b
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

// mainMenu builds the window menu: Cluster, View (one item per kind), Helm, on Windows and Linux
// Window, and Help (menus.go).
func (sh *shell) mainMenu() *fyne.MainMenu {
	// ⌘P (search a cluster by name) is cluster.connect's shortcut in the registry: the picker asks
	// for the context.
	cluster := fyne.NewMenu("Cluster", sh.withShortcut(fyne.NewMenuItem("Switch Context…", sh.picker.show), "cluster.connect"))
	if item := sh.hot.contextItem(sh.ui.Context); item != nil { // Lens's cluster menu: "Add to Hotbar"
		cluster.Items = append(cluster.Items, item)
	}
	cluster.Items = append(cluster.Items, sh.clusterMenuItems()...) // Disconnect, Remove from Cogpit… (menus_cluster.go)
	cluster.Items = append(cluster.Items, sh.kc.menuItems()...)
	cluster.Items = append(cluster.Items, sh.refreshAccessItem())
	cluster.Items = append(cluster.Items, sh.settingsMenuItems()...) // ⌘⇧, and ⌘, from any focus
	var items []*fyne.MenuItem
	for _, k := range listableKinds(sh.kinds) { // as the sidebar: no kind the user may not list
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
	view := fyne.NewMenu("View", items...)
	// The tab bar's ⌘W, ⌘⌥→ and ⌘⌥←, ⌘1–9, ⌃- and ⌃⇧-, from any focus (tabbar.go).
	view.Items = append(sh.tabs.menuItems(), view.Items...)
	if chat := sh.chatMenuItems(); len(chat) > 0 { // ⌘I and ⌘., from any focus (chat.go)
		view.Items = append(append(chat, fyne.NewMenuItemSeparator()), view.Items...)
	}
	if pal := sh.paletteMenuItem(); pal != nil { // ⌘⇧P, from any focus: menu shortcuts come first
		view.Items = append([]*fyne.MenuItem{pal, fyne.NewMenuItemSeparator()}, view.Items...)
	}
	menus := []*fyne.Menu{cluster, view, sh.helmMenu()}
	if window := sh.windowMenu(runtime.GOOS); window != nil { // macOS keeps GLFW's Window menu (menus.go)
		menus = append(menus, window)
	}
	return fyne.NewMainMenu(append(menus, sh.helpMenu())...)
}

// copyMCPCommand puts the `claude mcp add` command of the running server on the clipboard and shows
// it. While the server is off there is no command: the old one names a token that no longer works,
// and the bundled cogpit-cli would serve an engine of its own rather than this window.
func (sh *shell) copyMCPCommand() {
	cmd := sh.mcp.command
	if !sh.mcp.on || cmd == "" {
		return
	}
	sh.app.Clipboard().SetContent(cmd)
	entry := widget.NewEntry()
	entry.SetText(cmd)
	entry.TextStyle = fyne.TextStyle{Monospace: true}
	text := "Copied to the clipboard. Run it in a terminal to let an MCP client drive this Cogpit window. " +
		"Its port and token change at every start of Cogpit: run it again after a restart."
	if sh.mcp.stdio {
		text = "Copied to the clipboard. Run it once in a terminal: Claude Code then starts the cogpit-cli inside " +
			"Cogpit.app, which drives this Cogpit window whenever it runs, across restarts and updates. " +
			"Run it again if you move Cogpit.app."
	}
	intro := widget.NewLabel(text)
	intro.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustom("MCP command", "Close", container.NewVBox(intro, entry), sh.win)
	d.Resize(fyne.NewSize(sh.win.Canvas().Size().Width*0.5, 0))
	d.Show()
}

// installShortcuts adds Lens's keyboard shortcuts that Cogpit supports, as the registry declares
// them: ⌘F (ui.setFilter: the search), ⌘⇧T (terminal.open), Escape (close the palette or the
// picker, else the drawer), "/" (the logs tab's search), and F1–F12 and ⇧F1–⇧F12 for the hotbar
// while no text field has the focus (the table passes them on). ⌘⇧P and ⌘P are the main menu's
// (mainMenu), which take them from any focus.
func (sh *shell) installShortcuts() {
	c := sh.win.Canvas()
	if sc := sh.registryShortcut("ui.setFilter"); sc != nil {
		c.AddShortcut(sc, func(fyne.Shortcut) { c.Focus(sh.view.search) })
	}
	if sc := sh.registryShortcut("terminal.open"); sc != nil {
		c.AddShortcut(sc, func(fyne.Shortcut) { sh.dispatch("terminal.open", struct{}{}, nil) })
	}
	c.SetOnTypedKey(func(ev *fyne.KeyEvent) {
		if sh.hot.key(ev.Name, shiftHeld()) { // F1–F12, ⇧F1–⇧F12: the hotbar (Fyne has no shortcut for them)
			return
		}
		if ev.Name == fyne.KeyEscape {
			if sh.palette.visible() { // the palette lies over everything
				sh.palette.dismissThen(nil)
				return
			}
			if sh.top.notes.visible() { // the notifications' panel (notifications.go)
				sh.top.notes.close()
				return
			}
			if sh.picker.visible() { // the picker first: it lies over the drawer
				sh.picker.popup.dismiss()
				return
			}
			if sh.settings.visible() { // a settings page lies over the views
				sh.dispatch("settings.close", struct{}{}, nil)
				return
			}
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
		View   string   `json:"view"`
		States []string `json:"states,omitempty"` // opens the view filtered to these states (a dashboard's state link)
	}
	namespacesParams struct {
		Namespaces []string `json:"namespaces"`
	}
	groupsParams struct {
		Groups []string `json:"groups"`
	}
	columnsParams struct {
		View   string   `json:"view"`
		Hidden []string `json:"hidden"`
	}
	filterParams struct {
		Filter    string `json:"filter"`
		MatchCase *bool  `json:"matchCase,omitempty"` // the "Aa" toggle; nil: unchanged
		Regex     *bool  `json:"regex,omitempty"`     // the ".*" toggle; nil: unchanged
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
	Local      bool           `json:"local,omitempty"`    // port forwards: Cogpit's own objects
	Virtual    bool           `json:"virtual,omitempty"`  // Helm releases: made from other objects
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
		if x.ID != y.ID || x.Title != y.Title || x.Logs != y.Logs || x.Container != y.Container || !sameRef(x.Ref, y.Ref) ||
			x.Audit != y.Audit || // the Audit Log's filter (auditlog.go)
			x.AgentRead != y.AgentRead || x.AgentReadExpired != y.AgentReadExpired || x.AgentReadCapped != y.AgentReadCapped { // a terminal's "Agents may read" (dockoutput.go, dockagentread.go)
			return false
		}
		// A Helm tab's chart, version and digest change in place (a version picked, its chart
		// loaded): the tab's editor updates from them (dock.sync).
		if !reflect.DeepEqual(x.Helm, y.Helm) || !reflect.DeepEqual(x.HelmInstall, y.HelmInstall) {
			return false
		}
		// A terminal's kubectl check and its download change in place (kubectloffer.go).
		if !reflect.DeepEqual(x.Kubectl, y.Kubectl) {
			return false
		}
	}
	return true
}
