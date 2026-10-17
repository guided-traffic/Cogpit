package fyneui

import (
	"slices"

	"fyne.io/fyne/v2"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Several clusters connected in the window (D-69 stage 2, built 2026-10-16): the engine keeps
// several clusters connected and the window shows one (UIState.Context; every connected one in
// UIState.Connections). This file holds what every place that shows a cluster shares: the marks
// of its dot and the switch between connected clusters.
//
// The marks (D-69 item 5, the owner's decision of 2026-10-16): the dot's shape says live or
// resting, its colour the state, on every dot of a cluster (its hotbar tile, its navigator section
// row and the tree's root, its welcome-page row, the top bar's and the status bar's for the shown
// one):
//
//	filled green            live: the shown cluster, or a connected one whose views are warm
//	filled green, blinking  (re)connecting, a background cluster's reconnect too (conndot.go, F13)
//	green ring              connected, cold: its views released, showing it lists them again
//	grey ring               its login lapsed and Cogpit did not renew it (the cluster's setting)
//	filled red              needs the user: a connect failed, a renewal failed (Login expired)
//	filled grey             not connected
//
// The tooltip names the state; no banner and no toast for the rings. A click on any dot opens its
// cluster's connection log (D-66).
//
// Switching (D-69 item 1): a click on a connected cluster's hotbar tile, navigator section row,
// welcome-page row or one of its tabs shows it at once (cluster.connect of a connected context only
// shows it, from its cache): no login program is asked for again, nothing is sent. A cluster that
// is not connected connects and is shown; the others stay connected. A connect that fails while
// another cluster was shown shows that cluster again (clusterSwitch): the failed one's dots turn
// red and a toast says why.

// clusterMark is how a cluster's dot is drawn: its connection (the blink and the tooltip follow
// it), its colour when it does not blink, a ring for a resting state, and the state its tooltip
// names ("" for connectionText's).
type clusterMark struct {
	conn  engine.Connection
	color fyne.ThemeColorName
	ring  bool
	state string
}

// The tooltip's states of the marks that connectionText does not name.
const (
	markWarm         = "Connected, in the background: its views kept warm, showing it is instant"
	markCold         = "Connected, in the background: its views released, showing it lists them again"
	markLapsed       = "Connected, its login lapsed: Cogpit did not renew it while the cluster was not shown; showing it or Log in again renews it"
	markFailed       = "Connection failed: its last connect failed"
	markReconnecting = "Reconnecting: a new connection from the kubeconfig file, the old one shown until it stands"
	markBackground   = ", in the background"
)

// markOf is a cluster's mark in ui: the shown cluster's connection (a failed connect too), a
// connected cluster's entry of Connections, else a failed last connect (problem, the connect log's)
// or not connected.
func markOf(ui engine.UIState, ctx string, problem bool) clusterMark {
	entry, connected := ui.ConnectionOf(ctx)
	shown := ctx != "" && ctx == ui.Context
	conn := engine.Connection{State: engine.Disconnected}
	switch {
	case ctx == "":
	case shown:
		conn = ui.Connection
	case connected:
		conn = entry.Connection
	case problem:
		conn = engine.Connection{State: engine.ConnectError}
	}
	reconnect := conn.Reconnecting && conn.State.Live()
	if reconnect { // a new connection from the kubeconfig file (D-81): its dots blink as at a connect
		conn = engine.Connection{State: engine.Connecting}
	}
	m := clusterMark{conn: conn, color: connectionColor(conn)}
	background := connected && !shown
	if reconnect {
		m.state = markReconnecting
		if background {
			m.state += markBackground
		}
		return m
	}
	switch {
	case conn.State == engine.ConnectError || conn.State == engine.LoginExpired:
		if !shown && !connected {
			m.state = markFailed
		} else if background {
			m.state = connectionText(conn) + markBackground
		}
	case connected && entry.LoginLapsed:
		m.color, m.ring, m.state = colorStatusUnknown, true, markLapsed
	case blinkingState(conn.State) && background:
		m.state = connectionText(conn) + markBackground
	case blinkingState(conn.State):
	case background && entry.Cold:
		m.ring, m.state = true, markCold
	case background:
		m.state = markWarm
	}
	return m
}

// mark is a cluster's mark as the window shows it now; problem says its last connect failed.
func (sh *shell) mark(ctx string, problem bool) clusterMark { return markOf(sh.ui, ctx, problem) }

// connectProblem reports whether a context's last connect failed, as the navigator's list of the
// contexts says (ContextInfo.ConnectProblem, read again after every connect: clustersChanged).
func (sh *shell) connectProblem(ctx string) bool {
	if sh.side == nil || sh.side.sec == nil {
		return false
	}
	c, ok := sh.side.sec.context(ctx)
	return ok && c.ConnectProblem
}

// clusterConnected reports whether the engine holds a live connection to a context: the shown one
// connected (interrupted, its login expired included), or a connected one not shown.
func clusterConnected(ui engine.UIState, ctx string) bool {
	if ctx == "" {
		return false
	}
	if ctx == ui.Context && ui.Connection.State.Live() {
		return true
	}
	c, ok := ui.ConnectionOf(ctx)
	return ok && c.Connection.State.Live()
}

// switchable reports whether a click on a cluster only shows it: it is connected and not the
// shown one. No login program is shown for approval then, and nothing connects.
func switchable(ui engine.UIState, ctx string) bool {
	return ctx != ui.Context && clusterConnected(ui, ctx)
}

// connectItemLabel is the menus' item for a cluster: "Show" for a connected cluster the window does not
// show (D-69: it connects nothing), else "Connect".
func connectItemLabel(ui engine.UIState, ctx string) string {
	if switchable(ui, ctx) {
		return "Show"
	}
	return "Connect"
}

// clusterStates is every cluster's connection state the window knows: each connected one's and the
// shown one's (its Connection, a connect under way or failed too). The blinker follows them.
func clusterStates(ui engine.UIState) map[string]engine.ConnectionState {
	out := make(map[string]engine.ConnectionState, len(ui.Connections)+1)
	for _, c := range ui.Connections {
		out[c.Context] = blinkState(c.Connection)
	}
	if ui.Context != "" {
		out[ui.Context] = blinkState(ui.Connection)
	}
	return out
}

// blinkState is a connection's state for the blink: a reconnect blinks as a connect does (D-81).
func blinkState(c engine.Connection) engine.ConnectionState {
	if c.Reconnecting && c.State.Live() {
		return engine.Connecting
	}
	return c.State
}

// connectionsChanged reports whether the connected clusters or their states changed.
func connectionsChanged(st, old engine.UIState) bool {
	return !slices.EqualFunc(st.Connections, old.Connections, func(a, b engine.ConnectedCluster) bool {
		return a.Context == b.Context && a.Connection == b.Connection && a.Shown == b.Shown && a.Cold == b.Cold &&
			a.LoginLapsed == b.LoginLapsed
	})
}

// clusterSwitch is the window's memory of the cluster it showed, for a connect that fails while
// another cluster was shown (D-69 stage 2, item 7): the window shows that cluster again.
type clusterSwitch struct {
	shown string // the last cluster the window showed connected
	back  string // the cluster whose failed connect the window went back from, until the next connect
}

// clustersChanged follows the clusters after a change of the UI state: every dot's mark and blink,
// the bars, the navigator and the hotbar's tiles when only a connected cluster's state changed (the
// tab bar and the welcome page follow through tabsChanged), the contexts' failed connects after a
// connect ended, and a failed connect while another cluster was shown. applyUIState calls it after
// every change.
func (sh *shell) clustersChanged(st, old engine.UIState, first bool) {
	shownChanged := first || st.Context != old.Context || st.Connection != old.Connection
	if connectionsChanged(st, old) && !shownChanged { // the shown cluster's change refreshes them already (connectionChanged, hot.stateChanged)
		sh.blink.observeAll(clusterStates(st))
		sh.top.update()
		sh.status.update()
		sh.side.updateHeader()
		sh.hot.stateChanged()
		sh.view.checksChanged() // the selection bar's Delete follows a lapsed login too (tablecheck.go)
	}
	if first || shownChanged || connectionsChanged(st, old) {
		sh.kc.followReconnect(sh.kc.connectLog) // the open Connection Updates' Reconnect (connectlog.go, D-81)
		sh.view.cluster.followReconnect()       // the cluster Overview's (clusteroverview.go)
	}
	if !first && old.Connection.State == engine.Connecting && st.Connection.State != engine.Connecting {
		sh.side.sec.load() // a connect ended: its contexts' failed connects (ContextInfo.ConnectProblem)
		sh.tabs.welcome.reload()
	}
	sh.followSwitch(st, old)
}

// followSwitch remembers the cluster shown connected, and shows it again when a connect of another
// cluster failed while it was shown and is still connected: the failed cluster's dots turn red and
// a toast says why (the hotbar's own toast stays away then, hotbar.open).
func (sh *shell) followSwitch(st, old engine.UIState) {
	sw := &sh.switcher
	if st.Context != "" && st.Connection.State.Live() {
		sw.shown = st.Context
		return
	}
	if st.Connection.State == engine.Connecting {
		sw.back = ""
	}
	failed := st.Connection.State == engine.ConnectError && old.Connection.State != engine.ConnectError
	prev := sw.shown
	if !failed || prev == "" || prev == st.Context || !clusterConnected(st, prev) {
		return
	}
	sw.back = st.Context
	name := sh.identity(st.Context).name
	sh.toast("error", "Could not connect to "+name+": "+st.Connection.Error)
	sh.dispatch("cluster.connect", contextParams{Context: prev}, nil) // a connected cluster: only shown again
}

// wentBack reports whether the window showed the cluster before again after ctx's connect failed
// (followSwitch), which said why in its own toast.
func (sh *shell) wentBack(ctx string) bool { return ctx != "" && sh.switcher.back == ctx }
