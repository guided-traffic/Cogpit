package mcp

import (
	"context"
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// requestContextKey carries the HTTP request's own context into the tool handler (callContext).
type requestContextKey struct{}

// guard admits a request only when it names this endpoint by a loopback name (Host), comes from
// no web page other than this endpoint's own origin (Origin), and carries the bearer token, in
// that order: a page attempting DNS rebinding gets 403 before it learns anything about
// authentication. The SDK's own Host check runs again behind it.
type guard struct {
	hosts   map[string]bool // allowed Host values: "127.0.0.1:<port>", "localhost:<port>", "[::1]:<port>"
	origins map[string]bool // allowed Origin values: "http://" + each allowed host
	log     *slog.Logger
	next    http.Handler

	mu    sync.Mutex
	token []byte
	// epoch ends when the token is replaced (rotate) or the server stops (close): every request
	// the token admitted runs under it and ends with it. Guarded by mu.
	epoch context.Context
	end   context.CancelFunc
}

func newGuard(addr *net.TCPAddr, token string, log *slog.Logger, next http.Handler) *guard {
	g := &guard{hosts: map[string]bool{}, origins: map[string]bool{}, token: []byte(token), log: log, next: next}
	g.epoch, g.end = context.WithCancel(context.Background())
	port := strconv.Itoa(addr.Port)
	for _, name := range []string{"127.0.0.1", "localhost", "::1", addr.IP.String()} {
		hp := net.JoinHostPort(name, port)
		g.hosts[hp] = true
		g.origins["http://"+hp] = true
	}
	return g
}

// rotate replaces the token (the user's Regenerate token, F6): from now on only the new one is
// admitted, and every request the old one admitted ends, so a tool call waiting for the user's
// confirmation fails and its command does not run. The listener stays.
func (g *guard) rotate(token string) {
	g.mu.Lock()
	end := g.end
	g.token = []byte(token)
	g.epoch, g.end = context.WithCancel(context.Background())
	g.mu.Unlock()
	end()
}

// close ends every request the guard admitted; the server is stopping.
func (g *guard) close() {
	g.mu.Lock()
	end := g.end
	g.mu.Unlock()
	end()
}

func (g *guard) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !g.hosts[strings.ToLower(r.Host)] {
		g.reject(w, r, http.StatusForbidden, "Host not allowed", "forbidden: the Host header must name this endpoint by a loopback address")
		return
	}
	// No Origin: a non-browser client. Any Origin but our own: a web page, which may not drive
	// Cogpit; "null" (a sandboxed or file page) included.
	if origins := r.Header.Values("Origin"); len(origins) > 0 && (len(origins) > 1 || !g.origins[strings.ToLower(origins[0])]) {
		g.reject(w, r, http.StatusForbidden, "Origin not allowed", "forbidden: requests from web pages are not allowed")
		return
	}
	presented, epoch, ok := g.authorized(r)
	if !ok {
		reason := "missing bearer token"
		if presented {
			reason = "wrong bearer token"
		}
		g.reject(w, r, http.StatusUnauthorized, reason,
			"unauthorized: a valid bearer token is required. The user may have regenerated Cogpit's MCP token; "+
				"copy the registration command again from the MCP details (the gear of the MCP item in Cogpit's status bar)")
		return
	}
	// The token has done its job; nothing behind the guard needs it, so it cannot leak from there.
	r.Header.Del("Authorization")
	// The request runs under the token that admitted it: a new token ends it (rotate).
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	defer context.AfterFunc(epoch, cancel)()
	g.next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, requestContextKey{}, ctx)))
}

// authorized reports whether the request carries a bearer token (presented) and whether it is
// this server's token, compared in constant time, and the epoch of the token it checked against.
func (g *guard) authorized(r *http.Request) (presented bool, epoch context.Context, ok bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return false, nil, false
	}
	token = strings.TrimSpace(token)
	g.mu.Lock()
	want, epoch := g.token, g.epoch
	g.mu.Unlock()
	return token != "", epoch, subtle.ConstantTimeCompare([]byte(token), want) == 1
}

// reject answers with status and logs why, never with a header value: the Authorization header
// may hold a token, and Host and Origin are the caller's to choose.
func (g *guard) reject(w http.ResponseWriter, r *http.Request, status int, reason, body string) {
	level := slog.LevelWarn // a foreign Host or Origin is a web page probing the endpoint
	if status == http.StatusUnauthorized {
		level = slog.LevelInfo // usually a client holding a token the user regenerated since
		w.Header().Set("WWW-Authenticate", `Bearer realm="cogpit"`)
	}
	g.log.Log(r.Context(), level, "MCP request rejected", "status", status, "reason", reason,
		"remote", r.RemoteAddr, "method", r.Method, "path", r.URL.Path)
	http.Error(w, body, status)
}

// listenAddr validates a listen address and returns it normalised: the host must be a loopback IP
// or "localhost" (bound as 127.0.0.1, so the address does not depend on the resolver).
func listenAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("mcp: listen address %q: %w", addr, err)
	}
	if strings.EqualFold(host, "localhost") {
		host = "127.0.0.1"
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return "", fmt.Errorf("mcp: listen address %q is not a loopback address; the MCP endpoint listens on 127.0.0.1 only", addr)
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "", fmt.Errorf("mcp: listen address %q: invalid port", addr)
	}
	return net.JoinHostPort(ip.String(), port), nil
}
