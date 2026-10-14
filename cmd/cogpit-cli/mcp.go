package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/guided-traffic/cogpit/internal/buildinfo"
	"github.com/guided-traffic/cogpit/internal/mcp"
)

const mcpAbout = `Serves Cogpit's MCP tools over stdin and stdout, for agents that speak MCP only over stdio:
register "cogpit-cli mcp" as an MCP server once, instead of the URL and token that change with
every start of Cogpit.

While Cogpit runs with its MCP server switched on (its token and endpoint files in Cogpit's
configuration directory), every call is forwarded to it: the agent drives the window the user
sees, and mutating and destructive commands wait for the user's confirmation there. When Cogpit
restarts, the bridge follows it to its new port and token; a call is sent again only if it never
reached Cogpit.

Without a running Cogpit (or with --headless) the bridge runs an engine of its own, without a
window: read and UI commands run (connect a context with cluster_connect first), and commands
that need the user's confirmation are refused, since there is no window to ask in. It reads the
same kubeconfig sources and runs only the credential plugins the user allowed in Cogpit's window.
Every command an agent runs through it is written to Cogpit's audit log.

stdout carries the protocol only; logs go to stderr.`

func runMCP(ctx context.Context, args []string, e *env) int {
	fs := newFlags("mcp", "[flags]", mcpAbout)
	var c common
	c.register(fs)
	headless := fs.Bool("headless", false, "serve an engine of its own even when Cogpit runs (--kubeconfig applies to it only)")
	if code, done := fs.parse(args, e); done {
		return code
	}
	if len(fs.positional) > 0 {
		return fs.usageError(e, "mcp takes no arguments")
	}
	c.apply(e)
	t := e.mcpTransport
	if t == nil {
		t = &sdk.StdioTransport{}
	}

	if !*headless && e.dirsErr == nil && e.dirs.Config != "" {
		br, err := mcp.DialApp(ctx, mcp.BridgeOptions{
			TokenFile: filepath.Join(e.dirs.Config, mcp.TokenFileName), Version: buildinfo.Version(), Logger: e.logger,
		})
		switch {
		case err == nil:
			defer func() { _ = br.Close() }()
			return served(e, br.Serve(ctx, t))
		case !errors.Is(err, mcp.ErrAppNotRunning):
			return fail(e, err)
		}
		e.logger.Info("no running Cogpit serves MCP; serving an engine of the bridge's own", "reason", err.Error())
	}

	s, closeSession, err := e.newSession(sessionOptions{kubeconfig: c.kubeconfig, audit: true})
	if err != nil {
		return fail(e, err)
	}
	defer closeSession()
	defer s.WatchKubeconfigs(0)() // a sync folder's files that come or go, as in the app
	srv, err := mcp.New(s, mcp.Options{Version: buildinfo.Version(), Logger: e.logger})
	if err != nil {
		return fail(e, err)
	}
	defer func() { _ = srv.Close() }()
	return served(e, srv.ServeTransport(ctx, t))
}

// served turns the end of serving into an exit code: the client closing the stream, or a signal,
// is a normal end.
func served(e *env, err error) int {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, sdk.ErrConnectionClosed) {
		return exitOK
	}
	return fail(e, fmt.Errorf("mcp: %w", err))
}
