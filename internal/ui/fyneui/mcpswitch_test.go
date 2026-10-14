package fyneui

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// fakeMCP is an MCPControl that records its calls and hands out a new command on every start.
type fakeMCP struct {
	mu       sync.Mutex
	calls    []string
	runs     int
	startErr error
	stdio    string // the stdio registration an app bundle with cogpit-cli offers; "" for HTTP
}

func (m *fakeMCP) Start(fromUser bool) (MCPEndpoint, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, fmt.Sprintf("start(user=%v)", fromUser))
	if m.startErr != nil {
		return MCPEndpoint{}, m.startErr
	}
	m.runs++
	url := fmt.Sprintf("http://127.0.0.1:%d/mcp", 40000+m.runs)
	if m.stdio != "" {
		return MCPEndpoint{URL: url, Command: m.stdio, Stdio: true}, nil
	}
	return MCPEndpoint{URL: url, Command: fmt.Sprintf("claude mcp add cogpit %s --header token-%d", url, m.runs)}, nil
}

func (m *fakeMCP) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, "stop")
	return nil
}

func (m *fakeMCP) log() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return strings.Join(m.calls, ",")
}

func mcpShell(t *testing.T, ctl *fakeMCP) (*shell, fyne.Window, *mainQueue) {
	t.Helper()
	sh, w, q := newTestShell(t, connectedFake(t, 20), Options{MCP: ctl})
	sh.startMCP()
	settle(w)
	return sh, w, q
}

// TestMCPItemSwitchesTheServer: clicking the status bar item stops the server and starts it again;
// the item, its tooltip, the top bar's copy button and the Help menu follow, and the command copied
// after a restart is the new one, never the old.
func TestMCPItemSwitchesTheServer(t *testing.T) {
	ctl := &fakeMCP{}
	sh, w, q := mcpShell(t, ctl)
	item := sh.status.mcp
	if item == nil || sh.top.mcp == nil {
		t.Fatal("no MCP item or button with an MCP control")
	}
	if ctl.log() != "start(user=false)" || !sh.mcp.on || !item.on {
		t.Fatalf("after launch: calls %s, on %v", ctl.log(), sh.mcp.on)
	}
	if !strings.Contains(item.tip, "External MCP server") || !strings.Contains(item.tip, "http://127.0.0.1:40001/mcp") ||
		!strings.Contains(item.tip, "click to stop") {
		t.Errorf("tooltip while on: %q", item.tip)
	}
	if sh.top.mcp.Disabled() {
		t.Error("the copy button is disabled while the server runs")
	}

	test.Tap(item)
	q.until(t, "the server off", func() bool { return !sh.mcp.on && !sh.mcp.busy })
	if ctl.log() != "start(user=false),stop" {
		t.Errorf("calls %s, want a stop", ctl.log())
	}
	if !strings.Contains(item.tip, "off — click to start") || item.on {
		t.Errorf("after stop: on %v, tooltip %q", item.on, item.tip)
	}
	if !sh.top.mcp.Disabled() {
		t.Error("the copy button still works while the server is off")
	}
	sh.mcpMenu = fyne.NewMenuItem("x", nil) // the menu item exists only after mainMenu
	sh.refreshMCP()
	if !sh.mcpMenu.Disabled {
		t.Error("Help › Copy MCP command is enabled while the server is off")
	}
	sh.app.Clipboard().SetContent("")
	sh.copyMCPCommand()
	settle(w)
	if got := sh.app.Clipboard().Content(); got != "" {
		t.Errorf("a command was copied while the server is off: %q", got)
	}
	if sh.app.Preferences().BoolWithFallback(prefMCPEnabled, true) {
		t.Error("the off state was not stored")
	}

	test.Tap(item)
	q.until(t, "the server on", func() bool { return sh.mcp.on && !sh.mcp.busy })
	if ctl.log() != "start(user=false),stop,start(user=true)" {
		t.Errorf("calls %s, want a start by the user", ctl.log())
	}
	if !strings.Contains(item.tip, "40002") || sh.top.mcp.Disabled() {
		t.Errorf("after restart: tooltip %q, button disabled %v", item.tip, sh.top.mcp.Disabled())
	}
	sh.copyMCPCommand()
	if got := sh.app.Clipboard().Content(); !strings.Contains(got, "token-2") || strings.Contains(got, "token-1") {
		t.Errorf("copied %q, want the new command", got)
	}
	if !sh.app.Preferences().BoolWithFallback(prefMCPEnabled, true) {
		t.Error("the on state was not stored")
	}
}

// TestMCPCopyNamesWhatTheCommandIs: the copy dialog says that the HTTP registration must be run
// again after a restart, and that the bundled cogpit-cli's stdio registration is run once; the
// clipboard gets the command either way.
func TestMCPCopyNamesWhatTheCommandIs(t *testing.T) {
	for _, tc := range []struct {
		name, stdio, says string
	}{
		{"http", "", "run it again after a restart"},
		{"stdio", "claude mcp add --scope user cogpit -- /Applications/Cogpit.app/Contents/MacOS/cogpit-cli mcp", "Run it once"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sh, w, _ := mcpShell(t, &fakeMCP{stdio: tc.stdio})
			sh.copyMCPCommand()
			settle(w)
			if got := sh.app.Clipboard().Content(); got != sh.mcp.command || (tc.stdio != "" && got != tc.stdio) {
				t.Errorf("copied %q", got)
			}
			texts := strings.Join(overlayTexts(w), "\n")
			if !strings.Contains(texts, tc.says) {
				t.Errorf("dialog %q, want %q", texts, tc.says)
			}
		})
	}
}

// TestMCPOffIsRemembered: a server the user switched off stays off at the next launch, and a
// launch without a stored choice starts it.
func TestMCPOffIsRemembered(t *testing.T) {
	ctl := &fakeMCP{}
	sh, _, _ := newTestShell(t, connectedFake(t, 20), Options{MCP: ctl})
	sh.app.Preferences().SetBool(prefMCPEnabled, false)
	sh.startMCP()
	if ctl.log() != "" || sh.mcp.on || sh.status.mcp.on {
		t.Errorf("a stored off was ignored: calls %q, on %v", ctl.log(), sh.mcp.on)
	}
	if !sh.top.mcp.Disabled() {
		t.Error("the copy button is enabled while the server is off")
	}
}

// storingMCP is a fakeMCP that keeps the switch where cogpit-cli reads it (MCPSwitchStore, D-49).
type storingMCP struct {
	fakeMCP
	stored, set bool
	writes      []bool
}

func (m *storingMCP) StoredMCP() (on, set bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.stored, m.set
}

func (m *storingMCP) StoreMCP(on bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stored, m.set, m.writes = on, true, append(m.writes, on)
	return nil
}

func (m *storingMCP) written() []bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]bool(nil), m.writes...)
}

// TestMCPSwitchStore (D-49, A15): the switch store decides the launch over the window's
// preference; a store never set takes the preference once (a switch set before D-49); every click
// that worked is stored there too, for cogpit-cli mcp.
func TestMCPSwitchStore(t *testing.T) {
	off := &storingMCP{stored: false, set: true}
	sh, _, _ := newTestShell(t, connectedFake(t, 20), Options{MCP: off})
	sh.startMCP()
	if off.log() != "" || sh.mcp.on {
		t.Errorf("a stored off was ignored: calls %q", off.log())
	}

	migrate := &storingMCP{}
	sh, _, _ = newTestShell(t, connectedFake(t, 20), Options{MCP: migrate})
	sh.app.Preferences().SetBool(prefMCPEnabled, false)
	sh.startMCP()
	if migrate.log() != "" || sh.mcp.on || len(migrate.written()) != 1 || migrate.written()[0] {
		t.Errorf("the preference's off was not taken and copied: calls %q, writes %v", migrate.log(), migrate.written())
	}

	on := &storingMCP{stored: true, set: true}
	sh, w, q := newTestShell(t, connectedFake(t, 20), Options{MCP: on})
	sh.startMCP()
	settle(w)
	q.until(t, "the server on", func() bool { return sh.mcp.on })
	test.Tap(sh.status.mcp)
	q.until(t, "the switch stored off", func() bool { got := on.written(); return len(got) == 1 && !got[0] })
	if sh.mcp.on {
		t.Error("the server is still on after the click")
	}
}

// TestMCPStartFailureShowsOff: a server that cannot start leaves the item off with the reason, and
// the stored choice alone decides the next launch.
func TestMCPStartFailureShowsOff(t *testing.T) {
	ctl := &fakeMCP{startErr: errors.New("address in use")}
	sh, _, _ := newTestShell(t, connectedFake(t, 20), Options{MCP: ctl})
	sh.startMCP()
	if sh.mcp.on || !strings.Contains(sh.status.mcp.tip, "address in use") {
		t.Errorf("on %v, tooltip %q", sh.mcp.on, sh.status.mcp.tip)
	}
	if !sh.app.Preferences().BoolWithFallback(prefMCPEnabled, true) {
		t.Error("a failed start changed the stored choice")
	}
}

// TestNoMCPControlShowsNoItem: --no-mcp leaves no item, no button and no menu entry, and the link
// ends the bar.
func TestNoMCPControlShowsNoItem(t *testing.T) {
	sh, w, _ := newTestShell(t, connectedFake(t, 20), Options{})
	settle(w)
	if sh.status.mcp != nil || sh.top.mcp != nil || sh.hasMCP() {
		t.Error("an MCP item or button without a server")
	}
	statusSizes(sh, 2000)
	if sh.status.layout.sep.Visible() {
		t.Error("a divider after the link with nothing right of it")
	}
}

// TestMCPSwitchIsNoCommand: the switch is a GUI control, not an engine command, so no MCP tool
// can start the server (D-2, agents and MCP).
func TestMCPSwitchIsNoCommand(t *testing.T) {
	r := engine.NewRegistry()
	if err := commands.RegisterAll(r); err != nil {
		t.Fatal(err)
	}
	for _, c := range r.Commands() {
		text := strings.ToLower(c.ID + " " + c.Label + " " + c.Description)
		if strings.Contains(text, "mcp") {
			t.Errorf("command %q mentions the MCP server: %q", c.ID, text)
		}
	}
}

// TestMCPSwitchKeepsTheLinkLeftAndWidths: the item keeps its width on and off, so that the link and
// the status items do not move when the user clicks it; the link still gives way first.
func TestMCPSwitchKeepsTheLinkLeftAndWidths(t *testing.T) {
	ctl := &fakeMCP{}
	sh, w, q := mcpShell(t, ctl)
	q.until(t, "the diagnostics", func() bool { return sh.diag.total > 0 })
	settle(w)
	s := sh.status
	th := fyne.CurrentApp().Settings().Theme()
	width := s.layout.left.MinSize().Width + s.layout.right.MinSize().Width + 2*th.Size(theme.SizeNameInnerPadding) + 400
	link, _, linkW, _ := statusSizes(sh, width)
	onW := s.layout.right.Size().Width
	test.Tap(s.mcp)
	q.until(t, "off", func() bool { return !sh.mcp.on && !sh.mcp.busy })
	settle(w)
	link2, _, linkW2, _ := statusSizes(sh, width)
	if s.layout.right.Size().Width != onW || link != link2 || linkW != linkW2 {
		t.Errorf("the layout moved: item %v→%v, link %v→%v, link width %v→%v", onW, s.layout.right.Size().Width, link, link2, linkW, linkW2)
	}
}
