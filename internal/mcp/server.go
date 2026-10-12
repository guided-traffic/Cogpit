// Package mcp serves the engine's command registry as MCP tools over Streamable HTTP on
// 127.0.0.1, behind a random bearer token and Host and Origin checks (D-2,
// docs/security/agents-and-mcp.md). Every tool call dispatches through engine.Session with origin
// ExternalAgent, so the engine's confirmation rules and audit log apply.
//
// Layer L4.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/guided-traffic/cogpit/internal/engine"
)

const (
	// defaultAddr binds a free port on the IPv4 loopback address.
	defaultAddr = "127.0.0.1:0"
	// endpointPath is the one path the server answers on.
	endpointPath = "/mcp"
	// serverName is the MCP server name and the name the Claude Code registration uses.
	serverName = "cogpit"
	// shutdownGrace bounds how long Close waits for open requests before cutting connections.
	shutdownGrace = 2 * time.Second
)

// Options configure the server.
type Options struct {
	// Addr is the listen address; it must be a loopback address. Empty means 127.0.0.1:0
	// (a free port).
	Addr string
	// TokenFile is where the bearer token is written, owner-only (0600) in an owner-only (0700)
	// directory. Empty means <user config dir>/Cogpit/mcp-token.
	TokenFile string
	// Version is reported to clients; empty means "dev".
	Version string
	// Logger receives the server's logs; nil means slog.Default(). The token is never logged.
	Logger *slog.Logger
}

type serverState int

const (
	stateNew serverState = iota
	stateRunning
	stateClosed
)

// Server is the MCP endpoint for one engine session. Create it with New, call Start, and Close it
// on exit. Stop ends the endpoint like Close does but lets Start bring it back, with a new token
// and a new port (the user's switch in the status bar); a closed server cannot be started again.
type Server struct {
	session *engine.Session
	opts    Options
	log     *slog.Logger

	// ctl serialises Start, Stop and Close as wholes, so that a Start right after a Stop finds the
	// port free again and the old run's token file gone.
	ctl sync.Mutex

	mu sync.Mutex
	// lifetime ends with Stop or Close. Every tool call's context ends with it, so that a call
	// waiting for the user's confirmation does not outlive the endpoint. Each Start after a Stop
	// gets a new one. Guarded by mu.
	lifetime context.Context
	stop     context.CancelFunc

	state     serverState
	token     string
	tokenPath string
	url       string
	httpSrv   *http.Server
	served    chan struct{} // closed when http.Serve has returned
}

// New prepares a server for the session's registry. Nothing listens until Start. It refuses a
// listen address that is not a loopback address.
func New(session *engine.Session, opts Options) (*Server, error) {
	if session == nil {
		return nil, errors.New("mcp: a session is required")
	}
	if opts.Addr == "" {
		opts.Addr = defaultAddr
	}
	if _, err := listenAddr(opts.Addr); err != nil {
		return nil, err
	}
	if opts.Version == "" {
		opts.Version = "dev"
	}
	log := opts.Logger
	if log == nil {
		log = slog.Default()
	}
	lifetime, stop := context.WithCancel(context.Background())
	return &Server{
		session: session, opts: opts, log: log.With("component", "mcp"),
		lifetime: lifetime, stop: stop, tokenPath: opts.TokenFile,
	}, nil
}

// Start generates a fresh token, writes it to the token file, builds one MCP tool per registered
// command and starts listening. It returns once the listener is bound; ctx bounds only that
// start-up, and Close stops the server. Commands registered after Start are not tools until the
// next start.
func (s *Server) Start(ctx context.Context) error {
	s.ctl.Lock()
	defer s.ctl.Unlock()
	s.mu.Lock()
	defer s.mu.Unlock()
	switch s.state {
	case stateRunning:
		return errors.New("mcp: server already started")
	case stateClosed:
		return errors.New("mcp: server closed; create a new one")
	}

	tokenPath, ownDir := s.opts.TokenFile, false
	if tokenPath == "" {
		p, err := defaultTokenPath()
		if err != nil {
			return err
		}
		// The default directory is Cogpit's own, so Start may tighten its mode (writeTokenFile).
		tokenPath, ownDir = p, true
	}
	token, err := newToken()
	if err != nil {
		return err
	}
	addr, err := listenAddr(s.opts.Addr)
	if err != nil {
		return err
	}
	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("mcp: listen on %s: %w", addr, err)
	}
	tcp, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return fmt.Errorf("mcp: unexpected listener address %v", ln.Addr())
	}
	if err := writeTokenFile(tokenPath, token, ownDir); err != nil {
		_ = ln.Close()
		return err
	}

	life := s.lifetime
	srv, tools := s.newMCPServer()
	sdkHandler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return srv }, &sdk.StreamableHTTPOptions{
		// Stateless: every POST is its own session. Clients of both protocol generations work
		// (the 2026-07-28 protocol runs only stateless over HTTP), no session table grows, and a
		// call's lifetime is the lifetime of the HTTP request that carries it, which callContext
		// relies on to end a pending confirmation when the client goes away.
		Stateless:                    true,
		PropagateRequestCancellation: true,
		Logger:                       sdkLogger(s.log),
		// The SDK's own DNS-rebinding check (DisableLocalhostProtection false) stays on behind
		// the guard.
	})
	mux := http.NewServeMux()
	mux.Handle(endpointPath, sdkHandler)

	s.httpSrv = &http.Server{
		Handler:           newGuard(tcp, token, s.log, mux),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       2 * time.Minute,
		ErrorLog:          slog.NewLogLogger(s.log.Handler(), slog.LevelWarn),
		// Request contexts end with Close, so streams and pending calls unwind at once.
		BaseContext: func(net.Listener) context.Context { return life },
	}
	s.served = make(chan struct{})
	go s.serve(s.httpSrv, ln, s.served)

	s.token = token
	s.tokenPath = tokenPath
	s.url = "http://" + net.JoinHostPort(tcp.IP.String(), strconv.Itoa(tcp.Port)) + endpointPath
	s.state = stateRunning
	s.log.Info("MCP server listening", "url", s.url, "tokenFile", tokenPath, "tools", tools)
	return nil
}

// serve runs the HTTP server until Close. A panic is logged rather than ending the app.
func (s *Server) serve(srv *http.Server, ln net.Listener, done chan<- struct{}) {
	defer close(done)
	defer func() {
		if r := recover(); r != nil {
			s.log.Error("MCP server panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.log.Error("MCP server stopped", "err", err)
	}
}

// ServeTransport serves the registry's tools over one MCP transport until the client disconnects
// or ctx ends: the in-process path for tests and for the stdio bridge (architecture §7). It needs
// no Start and uses no token, because whoever holds the transport is already trusted; the
// engine's confirmation rules and audit log apply as over HTTP.
func (s *Server) ServeTransport(ctx context.Context, t sdk.Transport) error {
	s.mu.Lock()
	closed := s.state == stateClosed
	s.mu.Unlock()
	if closed {
		return errors.New("mcp: server closed")
	}
	srv, _ := s.newMCPServer()
	return srv.Run(ctx, t)
}

// URL returns the endpoint URL, "http://127.0.0.1:<port>/mcp", or "" when the server is not
// running.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != stateRunning {
		return ""
	}
	return s.url
}

// TokenPath returns the path of the token file: Options.TokenFile, or the default path once Start
// has resolved it.
func (s *Server) TokenPath() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tokenPath
}

// ClaudeAddCommand returns a ready-to-paste shell command that registers this endpoint with
// Claude Code for every directory (user scope), replacing an earlier registration:
//
//	claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user --transport http cogpit http://127.0.0.1:<port>/mcp --header "Authorization: Bearer <token>"
//
// It returns "" when the server is not running. The command contains the token: print it to the
// user's terminal, never to a log. The port and the token change on every start, so the
// registration must be repeated after Cogpit restarts; the stdio bridge (architecture §7) is the
// later fix for that friction. The syntax is POSIX shell (macOS, Linux).
func (s *Server) ClaudeAddCommand() string {
	url, token := s.endpoint()
	if url == "" {
		return ""
	}
	// --header takes several values, so it comes last, after the name and the URL.
	return fmt.Sprintf(`claude mcp remove --scope user %[1]s >/dev/null 2>&1; `+
		`claude mcp add --scope user --transport http %[1]s %[2]s --header "Authorization: Bearer %[3]s"`,
		serverName, url, token)
}

// ClientConfigJSON returns an MCP client configuration for this endpoint, for
// `claude -p --strict-mcp-config --mcp-config <file>`; it leaves the user's Claude Code
// configuration untouched:
//
//	{"mcpServers": {"cogpit": {"type": "http", "url": "...", "headers": {"Authorization": "Bearer ..."}}}}
//
// It returns "" when the server is not running. Like ClaudeAddCommand it contains the token,
// which changes on every start: write it to an owner-only file and never log it.
func (s *Server) ClientConfigJSON() string {
	url, token := s.endpoint()
	if url == "" {
		return ""
	}
	type entry struct {
		Type    string            `json:"type"`
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	cfg := struct {
		MCPServers map[string]entry `json:"mcpServers"`
	}{MCPServers: map[string]entry{serverName: {
		Type: "http", URL: url, Headers: map[string]string{"Authorization": "Bearer " + token},
	}}}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil { // cannot happen for maps of strings
		return ""
	}
	return string(data)
}

func (s *Server) endpoint() (url, token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.state != stateRunning {
		return "", ""
	}
	return s.url, s.token
}

// life returns the context that ends when the endpoint stops.
func (s *Server) life() context.Context {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lifetime
}

// Close stops the server for good, ends pending tool calls (a call waiting for a confirmation fails
// and its command does not run) and removes the token file if it still holds this server's token.
// It is safe to call more than once.
func (s *Server) Close() error { return s.halt(true) }

// Stop ends the endpoint as Close does: it stops listening, ends every connection and pending tool
// call, and removes the token file. Unlike Close it leaves the server startable: Start brings the
// endpoint back with a new token and a new listener. Stopping a server that is not running does
// nothing. The user's switch in the status bar is the one caller; no tool can reach it.
func (s *Server) Stop() error { return s.halt(false) }

func (s *Server) halt(final bool) error {
	s.ctl.Lock()
	defer s.ctl.Unlock()
	s.mu.Lock()
	if s.state == stateClosed {
		s.mu.Unlock()
		return nil
	}
	wasRunning := s.state == stateRunning
	srv, served, path, token, stop := s.httpSrv, s.served, s.tokenPath, s.token, s.stop
	s.token, s.httpSrv, s.url = "", nil, ""
	switch {
	case final:
		s.state = stateClosed
	case wasRunning:
		s.state = stateNew
		s.lifetime, s.stop = context.WithCancel(context.Background())
	}
	s.mu.Unlock()

	if !wasRunning && !final {
		return nil
	}
	stop()
	if !wasRunning {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownGrace)
	defer cancel()
	var errs []error
	if err := srv.Shutdown(ctx); err != nil {
		if err := srv.Close(); err != nil {
			errs = append(errs, fmt.Errorf("mcp: close: %w", err))
		}
	}
	<-served
	if err := removeTokenFile(path, token); err != nil {
		errs = append(errs, err)
	}
	s.log.Info("MCP server stopped")
	return errors.Join(errs...)
}
