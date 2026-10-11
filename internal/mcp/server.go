// Package mcp serves the engine's command registry as MCP tools over Streamable HTTP on
// 127.0.0.1, behind a random bearer token and Host and Origin checks (D-2,
// docs/security/agents-and-mcp.md). Every tool call dispatches through engine.Session with origin
// ExternalAgent, so the engine's confirmation rules and audit log apply.
//
// Layer L4.
package mcp

import (
	"context"
	"errors"

	_ "github.com/modelcontextprotocol/go-sdk/mcp" // pinned; Phase 1 uses it

	"github.com/guided-traffic/cogpit/internal/engine"
)

// ErrNotImplemented marks a Phase 0 stub.
var ErrNotImplemented = errors.New("not implemented")

// Options configure the server.
type Options struct {
	// Addr is the listen address; it must be a loopback address. Empty means 127.0.0.1:0
	// (a free port).
	Addr string
	// TokenFile is where the bearer token is written, owner-only (0600). Empty means
	// <user config dir>/Cogpit/mcp-token.
	TokenFile string
	// Version is reported to clients.
	Version string
}

// Server is the running MCP endpoint.
type Server struct {
	session *engine.Session
	opts    Options
}

// New prepares a server for the session's registry. Nothing listens until Start.
func New(session *engine.Session, opts Options) (*Server, error) {
	return &Server{session: session, opts: opts}, nil
}

// Start writes a fresh token and starts listening. It returns once the listener is bound.
func (s *Server) Start(ctx context.Context) error {
	_ = ctx
	return ErrNotImplemented
}

// URL returns the endpoint URL, "http://127.0.0.1:<port>/mcp".
func (s *Server) URL() string { return "" }

// TokenPath returns the path of the token file.
func (s *Server) TokenPath() string { return "" }

// ClaudeAddCommand returns a ready-to-paste `claude mcp add` command for this endpoint.
func (s *Server) ClaudeAddCommand() string { return "" }

// Close stops the server and removes the token file.
func (s *Server) Close() error { return nil }
