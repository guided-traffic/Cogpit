package fyneui

import (
	"fmt"
	"image/color"
	"log/slog"
	"runtime/debug"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// MCPEndpoint is what a running MCP server tells the window: where agents connect and the command
// that registers it with Claude Code.
type MCPEndpoint struct {
	// URL is the endpoint, shown in the status bar item's tooltip.
	URL string
	// Command is the `claude mcp add` command the window copies: the HTTP registration for this
	// run, which carries the bearer token and changes on every start, or with Stdio the
	// registration of the cogpit-cli in the app's bundle.
	Command string
	// Stdio marks a Command that registers the bundled cogpit-cli's stdio bridge: run once, it
	// follows every restart of Cogpit (D-2).
	Stdio bool
}

// MCPControl lets the user switch the external MCP server, the loopback HTTP endpoint for other
// programs, off and on from the window (D-23). It touches nothing else of the engine: an
// in-process agent keeps working while the endpoint is off. The window
// calls it from a goroutine of its own, never from the main goroutine, and it is the only way to
// start the server: the switch is not an engine command and not an MCP tool, so no agent can reach
// it. The desktop app implements it over internal/mcp (cmd/cogpit), which internal/ui may not
// import.
type MCPControl interface {
	// Start brings the server up, with a new token, and returns its endpoint. fromUser is true for
	// a click on the status bar item and false for the start at launch, so that the implementation
	// can record the user's switch in the audit log.
	Start(fromUser bool) (MCPEndpoint, error)
	// Stop takes the server down: it stops listening, ends every agent connection and pending tool
	// call, and removes the token file. Only the user's click calls it; quitting is the caller's own
	// business.
	Stop() error
}

// prefMCPEnabled is the preference that remembers the user's switch across restarts. Absent means
// on, as before the switch existed; only a click writes it.
const prefMCPEnabled = "mcp.enabled"

// mcpState is what the window knows of the MCP server. Main goroutine only.
type mcpState struct {
	on      bool   // the server runs
	busy    bool   // a start or stop is under way; clicks are ignored
	command string // the registration command of the current run; "" while off
	stdio   bool   // command registers the bundled cogpit-cli (MCPEndpoint.Stdio)
	url     string
	err     string // why the last start failed; "" otherwise
}

// hasMCP reports whether the window has an MCP server at all: --no-mcp leaves neither the item nor
// the buttons.
func (sh *shell) hasMCP() bool { return sh.opts.MCP != nil || sh.opts.MCPCommand != "" }

// mcpTip is the status bar item's tooltip.
func (sh *shell) mcpTip() string {
	m := sh.mcp
	switch {
	case m.busy && m.on:
		return mcpTipName + ": stopping…"
	case m.busy:
		return mcpTipName + ": starting…"
	case m.on && m.url != "":
		return mcpTipName + ": on — " + m.url + " — click to stop"
	case m.on:
		return mcpTipName + ": on — click to stop"
	case m.err != "":
		return mcpTipName + ": off — click to start\nThe last start failed: " + m.err
	}
	return mcpTipName + ": off — click to start"
}

// mcpTipName names what the switch controls: the HTTP endpoint other programs connect to with the
// token. A built-in chat (D-10) calls the engine in-process and does not depend on it.
const mcpTipName = "External MCP server (for other apps on this computer)"

// startMCP starts the server at launch unless the user switched it off in an earlier run. A server
// that cannot start leaves the item in its off state with the reason in the tooltip; the stored
// choice stays as it was. Call it once, before the window shows.
func (sh *shell) startMCP() {
	ctl := sh.opts.MCP
	if ctl == nil {
		return
	}
	if sh.app != nil && !sh.app.Preferences().BoolWithFallback(prefMCPEnabled, true) {
		sh.applyMCP(false, MCPEndpoint{}, "")
		return
	}
	ep, err := ctl.Start(false)
	if err != nil {
		slog.Warn("MCP server not started", "err", err)
		sh.applyMCP(false, MCPEndpoint{}, err.Error())
		return
	}
	sh.applyMCP(true, ep, "")
}

// toggleMCP is a click on the status bar item: it stops a running server and starts a stopped one.
// The work runs off the main goroutine (stopping waits for open requests); a click while it runs
// is ignored. The choice is stored once the switch worked.
func (sh *shell) toggleMCP() {
	ctl := sh.opts.MCP
	if ctl == nil || sh.mcp.busy {
		return
	}
	want := !sh.mcp.on
	sh.mcp.busy = true
	sh.refreshMCP()
	go func() {
		var (
			ep  MCPEndpoint
			err error
		)
		func() {
			defer func() {
				if r := recover(); r != nil {
					slog.Error("MCP switch panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
					err = fmt.Errorf("internal error: %v", r)
				}
			}()
			if want {
				ep, err = ctl.Start(true)
			} else {
				err = ctl.Stop()
			}
		}()
		sh.do(func() {
			sh.mcp.busy = false
			switch {
			case want && err != nil:
				sh.applyMCP(false, MCPEndpoint{}, err.Error())
				sh.toast("error", "cannot start the MCP server: "+err.Error())
				return
			case want:
				sh.applyMCP(true, ep, "")
			default:
				// A failed stop (the token file could not be removed) still leaves nothing
				// listening: the server is off either way.
				sh.applyMCP(false, MCPEndpoint{}, "")
				if err != nil {
					sh.toast("error", "the MCP server is off, but: "+err.Error())
				}
			}
			if sh.app != nil {
				sh.app.Preferences().SetBool(prefMCPEnabled, want)
			}
		})
	}()
}

// applyMCP records the server's state and brings the item, the buttons and the menu in line, so
// that no control offers a command of a server that is not running.
func (sh *shell) applyMCP(on bool, ep MCPEndpoint, errText string) {
	sh.mcp.on, sh.mcp.err = on, errText
	sh.mcp.command, sh.mcp.url, sh.mcp.stdio = "", "", false
	if on {
		sh.mcp.command, sh.mcp.url, sh.mcp.stdio = ep.Command, ep.URL, ep.Stdio
	}
	sh.refreshMCP()
}

// refreshMCP shows sh.mcp in the status bar item, the top bar button and the Help menu.
func (sh *shell) refreshMCP() {
	if sh.status != nil && sh.status.mcp != nil {
		sh.status.mcp.set(sh.mcp.on, sh.mcpTip())
	}
	if sh.top != nil && sh.top.mcp != nil {
		if sh.mcp.on {
			sh.top.mcp.Enable()
		} else {
			sh.top.mcp.Disable()
		}
	}
	if sh.mcpMenu != nil {
		sh.mcpMenu.Disabled = !sh.mcp.on
		if sh.win != nil {
			if m := sh.win.MainMenu(); m != nil {
				m.Refresh()
			}
		}
	}
}

// mcpItem is the status bar's MCP indicator and its switch: the accent pill while the server runs,
// a muted pill with a line through it while it is off. Clicking it asks the shell to switch the
// server; the tooltip says what a click does.
type mcpItem struct {
	widget.BaseWidget
	sh   *shell
	pill *pill
	on   bool
	tip  string
}

func newMCPItem(sh *shell, on bool) *mcpItem {
	it := &mcpItem{sh: sh, pill: newPill("MCP", colorAccent, true), on: on}
	it.ExtendBaseWidget(it)
	it.set(on, sh.mcpTip())
	return it
}

// set shows the state and the tooltip text.
func (it *mcpItem) set(on bool, tip string) {
	it.on, it.tip = on, tip
	it.pill.tinted = on
	if on {
		it.pill.Set("MCP", colorAccent)
	} else {
		it.pill.Set("MCP", colorMuted)
	}
	it.pill.Refresh()
	it.Refresh()
	if it.sh.tip != nil && it.sh.tip.owner == it { // a shown tooltip follows the state
		it.sh.tip.hide()
	}
}

// Tapped implements fyne.Tappable.
func (it *mcpItem) Tapped(*fyne.PointEvent) {
	it.sh.tip.hideFor(it)
	it.sh.toggleMCP()
}

// Cursor implements desktop.Cursorable.
func (it *mcpItem) Cursor() desktop.Cursor { return desktop.PointerCursor }

// MouseIn implements desktop.Hoverable.
func (it *mcpItem) MouseIn(ev *desktop.MouseEvent) { it.sh.tip.hover(it, it.tip, ev.AbsolutePosition) }

// MouseMoved implements desktop.Hoverable.
func (it *mcpItem) MouseMoved(ev *desktop.MouseEvent) { it.sh.tip.moved(it, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (it *mcpItem) MouseOut() { it.sh.tip.hideFor(it) }

// CreateRenderer implements fyne.Widget.
func (it *mcpItem) CreateRenderer() fyne.WidgetRenderer {
	r := &mcpItemRenderer{it: it, strike: canvas.NewLine(color.Transparent)}
	r.strike.StrokeWidth = 1.5
	r.Refresh()
	return r
}

type mcpItemRenderer struct {
	it     *mcpItem
	strike *canvas.Line
}

func (r *mcpItemRenderer) MinSize() fyne.Size { return r.it.pill.MinSize() }

func (r *mcpItemRenderer) Layout(size fyne.Size) {
	r.it.pill.Move(fyne.NewPos(0, 0))
	r.it.pill.Resize(size)
	// The line runs through the pill's text, a little inside its ends.
	ms := r.it.pill.MinSize()
	x0 := (size.Width - ms.Width) / 2
	y := size.Height / 2
	r.strike.Position1 = fyne.NewPos(x0+3, y)
	r.strike.Position2 = fyne.NewPos(x0+ms.Width-3, y)
}

func (r *mcpItemRenderer) Refresh() {
	r.strike.StrokeColor = themeColor(r.it.Theme(), colorMuted)
	if r.it.on {
		r.strike.Hide()
	} else {
		r.strike.Show()
	}
	r.Layout(r.it.Size())
	r.it.pill.Refresh()
	r.strike.Refresh()
}

func (r *mcpItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.it.pill, r.strike}
}

func (r *mcpItemRenderer) Destroy() {}
