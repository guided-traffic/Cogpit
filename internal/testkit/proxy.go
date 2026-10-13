package testkit

import (
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// ForwardProxy is an HTTP forward proxy for tests: it tunnels CONNECT requests (how a client
// reaches an https server through a proxy) and forwards plain-http requests, counts both, and
// records the Proxy-Authorization it was sent. With a user set it refuses a request without
// that user's Basic credentials (407). It listens on loopback.
type ForwardProxy struct {
	srv *httptest.Server
	// URL is the proxy's URL, http://127.0.0.1:<port>, without credentials.
	URL *url.URL

	user, password string

	mu       sync.Mutex
	connects map[string]int // CONNECT target host:port → count
	plain    int
	auths    []string // the Proxy-Authorization users seen (decoded user names)
	refused  int
}

// NewForwardProxy starts a forward proxy; with a user and password it asks for them. It closes
// with the test.
func NewForwardProxy(t testing.TB, user, password string) *ForwardProxy {
	t.Helper()
	p := &ForwardProxy{user: user, password: password, connects: map[string]int{}}
	p.srv = httptest.NewServer(http.HandlerFunc(p.serve))
	t.Cleanup(p.srv.Close)
	u, err := url.Parse(p.srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	p.URL = u
	return p
}

// URLWithUser returns the proxy's URL with a user and password in it.
func (p *ForwardProxy) URLWithUser(user, password string) *url.URL {
	u := *p.URL
	u.User = url.UserPassword(user, password)
	return &u
}

// Connects returns how many CONNECT tunnels the proxy opened, by target.
func (p *ForwardProxy) Connects() map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]int, len(p.connects))
	for k, v := range p.connects {
		out[k] = v
	}
	return out
}

// Total returns the requests the proxy served: tunnels and plain requests.
func (p *ForwardProxy) Total() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	n := p.plain
	for _, v := range p.connects {
		n += v
	}
	return n
}

// Users returns the user names of the Proxy-Authorization headers the proxy was sent.
func (p *ForwardProxy) Users() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]string(nil), p.auths...)
}

// Refused returns how many requests the proxy refused for missing or wrong credentials.
func (p *ForwardProxy) Refused() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.refused
}

func (p *ForwardProxy) authorized(r *http.Request) bool {
	h := r.Header.Get("Proxy-Authorization")
	user, pass := "", ""
	if b, ok := strings.CutPrefix(h, "Basic "); ok {
		if raw, err := base64.StdEncoding.DecodeString(b); err == nil {
			user, pass, _ = strings.Cut(string(raw), ":")
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if h != "" {
		p.auths = append(p.auths, user)
	}
	if p.user == "" {
		return true
	}
	if user != p.user || pass != p.password {
		p.refused++
		return false
	}
	return true
}

func (p *ForwardProxy) serve(w http.ResponseWriter, r *http.Request) {
	if !p.authorized(r) {
		w.Header().Set("Proxy-Authenticate", `Basic realm="test"`)
		http.Error(w, "proxy authentication required", http.StatusProxyAuthRequired)
		return
	}
	if r.Method == http.MethodConnect {
		p.tunnel(w, r)
		return
	}
	p.mu.Lock()
	p.plain++
	p.mu.Unlock()
	out, err := http.NewRequestWithContext(r.Context(), r.Method, r.URL.String(), r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	out.Header = r.Header.Clone()
	out.Header.Del("Proxy-Authorization")
	resp, err := (&http.Transport{Proxy: nil}).RoundTrip(out)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	for k, vs := range resp.Header {
		for _, v := range vs {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

// tunnel connects the client to the target and copies both ways until either side closes.
func (p *ForwardProxy) tunnel(w http.ResponseWriter, r *http.Request) {
	target, err := net.DialTimeout("tcp", r.Host, 5*time.Second)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		_ = target.Close()
		http.Error(w, "no hijacking", http.StatusInternalServerError)
		return
	}
	p.mu.Lock()
	p.connects[r.Host]++
	p.mu.Unlock()
	client, buf, err := hj.Hijack()
	if err != nil {
		_ = target.Close()
		return
	}
	if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		_ = client.Close()
		_ = target.Close()
		return
	}
	if n := buf.Reader.Buffered(); n > 0 {
		pending, _ := buf.Reader.Peek(n)
		_, _ = target.Write(pending)
	}
	var once sync.Once
	done := func() {
		once.Do(func() {
			_ = client.Close()
			_ = target.Close()
		})
	}
	go func() {
		defer done()
		_, _ = io.Copy(target, client)
	}()
	go func() {
		defer done()
		_, _ = io.Copy(client, target)
	}()
}
