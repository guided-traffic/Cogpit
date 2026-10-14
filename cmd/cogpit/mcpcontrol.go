package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/mcp"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui"
)

// auditMCPSwitch is the audit log's name for the user's switch of the MCP server; the line's origin
// is "gui" and its parameters are {"enabled": true|false}.
const auditMCPSwitch = "gui.mcpServer"

// mcpControl is the window's handle on the MCP server (fyneui.MCPControl): the status bar item
// stops and starts the server through it. It lives in the app's main package because internal/ui
// may not import internal/mcp. Only the window holds it; no engine command or MCP tool reaches it,
// so an agent cannot switch the server on.
type mcpControl struct {
	srv     *mcp.Server
	session *engine.Session
	logger  *slog.Logger
	out     io.Writer // where the registration command is printed: the terminal that started Cogpit
	// cli is the cogpit-cli in the running app's bundle (bundledCLI), "" without one. With it the
	// window offers the stdio registration, made once, instead of the HTTP one with this run's port
	// and token (D-2).
	cli string
	// config is Cogpit's configuration directory, where the switch is kept for cogpit-cli (D-49);
	// "" keeps it in the window's preferences only.
	config string
}

var (
	_ fyneui.MCPControl     = (*mcpControl)(nil)
	_ fyneui.MCPSwitchStore = (*mcpControl)(nil)
)

// StoredMCP implements fyneui.MCPSwitchStore. A switch file that cannot be read counts as set and
// off, as cogpit-cli reads it: the server stays off until the user clicks.
func (c *mcpControl) StoredMCP() (on, set bool) {
	if c.config == "" {
		return true, false
	}
	on, set, err := mcp.ReadSwitch(c.config)
	if err != nil {
		c.logger.Warn("MCP switch unreadable; the server stays off", "err", err)
	}
	return on, set
}

// StoreMCP implements fyneui.MCPSwitchStore: the switch file cogpit-cli mcp reads (D-49).
func (c *mcpControl) StoreMCP(on bool) error {
	if c.config == "" {
		return nil
	}
	return mcp.WriteSwitch(c.config, on)
}

// Start implements fyneui.MCPControl. The server gets a new token and listener on every start. The
// command for the window is the stdio registration of the bundled cogpit-cli when there is one,
// else the HTTP registration, which carries the token: either goes to the terminal that started
// Cogpit and to the window, never to the logger. A switch by the user is recorded in the audit
// log, the start at launch is not.
func (c *mcpControl) Start(fromUser bool) (fyneui.MCPEndpoint, error) {
	err := c.srv.Start(context.Background())
	if fromUser {
		c.session.AuditUser(auditMCPSwitch, map[string]bool{"enabled": true}, err)
	}
	if err != nil {
		return fyneui.MCPEndpoint{}, err
	}
	ep := fyneui.MCPEndpoint{URL: c.srv.URL(), Command: c.srv.ClaudeAddCommand()}
	if cmd := mcp.ClaudeAddStdioCommand(c.cli); cmd != "" { // "" for no cli
		ep.Command, ep.Stdio = cmd, true
	}
	if c.out != nil {
		how := "the token changes on every start"
		if ep.Stdio {
			how = "once: Cogpit's own cogpit-cli follows every restart"
		}
		_, _ = fmt.Fprintf(c.out, "Cogpit MCP server: %s\nRegister it with Claude Code (%s):\n\n  %s\n\n", ep.URL, how, ep.Command)
	}
	return ep, nil
}

// Stop implements fyneui.MCPControl: the server stops listening, every agent connection and
// pending confirmation ends, and the token file is removed. The user's switch is recorded in the
// audit log.
func (c *mcpControl) Stop() error {
	err := c.srv.Stop()
	c.session.AuditUser(auditMCPSwitch, map[string]bool{"enabled": false}, err)
	if err != nil {
		c.logger.Warn("MCP server stopped with an error", "err", err)
	}
	return err
}
