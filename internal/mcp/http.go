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
	token   []byte
	log     *slog.Logger
	next    http.Handler
}

func newGuard(addr *net.TCPAddr, token string, log *slog.Logger, next http.Handler) *guard {
	g := &guard{hosts: map[string]bool{}, origins: map[string]bool{}, token: []byte(token), log: log, next: next}
	port := strconv.Itoa(addr.Port)
	for _, name := range []string{"127.0.0.1", "localhost", "::1", addr.IP.String()} {
		hp := net.JoinHostPort(name, port)
		g.hosts[hp] = true
		g.origins["http://"+hp] = true
	}
	return g
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
	presented, ok := g.authorized(r)
	if !ok {
		reason := "missing bearer token"
		if presented {
			reason = "wrong bearer token"
		}
		g.reject(w, r, http.StatusUnauthorized, reason,
			"unauthorized: a valid bearer token is required. Cogpit makes a new token on every start; "+
				"register the endpoint again with the command Cogpit prints at start-up")
		return
	}
	// The token has done its job; nothing behind the guard needs it, so it cannot leak from there.
	r.Header.Del("Authorization")
	g.next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestContextKey{}, r.Context())))
}

// authorized reports whether the request carries a bearer token (presented) and whether it is
// this server's token, compared in constant time.
func (g *guard) authorized(r *http.Request) (presented, ok bool) {
	scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return false, false
	}
	token = strings.TrimSpace(token)
	return token != "", subtle.ConstantTimeCompare([]byte(token), g.token) == 1
}

// reject answers with status and logs why, never with a header value: the Authorization header
// may hold a token, and Host and Origin are the caller's to choose.
func (g *guard) reject(w http.ResponseWriter, r *http.Request, status int, reason, body string) {
	level := slog.LevelWarn // a foreign Host or Origin is a web page probing the endpoint
	if status == http.StatusUnauthorized {
		level = slog.LevelInfo // usually a client holding the token of an earlier start
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
