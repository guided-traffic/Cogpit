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
}

var _ fyneui.MCPControl = (*mcpControl)(nil)

// Start implements fyneui.MCPControl. The server gets a new token and listener on every start. The
// command carries the token: it goes to the terminal that started Cogpit and to the window, never
// to the logger. A switch by the user is recorded in the audit log, the start at launch is not.
func (c *mcpControl) Start(fromUser bool) (fyneui.MCPEndpoint, error) {
	err := c.srv.Start(context.Background())
	if fromUser {
		c.session.AuditUser(auditMCPSwitch, map[string]bool{"enabled": true}, err)
	}
	if err != nil {
		return fyneui.MCPEndpoint{}, err
	}
	ep := fyneui.MCPEndpoint{URL: c.srv.URL(), Command: c.srv.ClaudeAddCommand()}
	if c.out != nil {
		_, _ = fmt.Fprintf(c.out, "Cogpit MCP server: %s\nRegister it with Claude Code (the token changes on every start):\n\n  %s\n\n", ep.URL, ep.Command)
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
