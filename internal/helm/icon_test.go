package helm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// iconServer serves icons over TLS on loopback and records what each request carried.
type iconServer struct {
	srv *httptest.Server
	mu  sync.Mutex
	got []*http.Request
}

func newIconServer(t *testing.T, h http.HandlerFunc) *iconServer {
	t.Helper()
	s := &iconServer{}
	s.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		s.mu.Lock()
		s.got = append(s.got, req.Clone(context.Background()))
		s.mu.Unlock()
		h(w, req)
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *iconServer) requests() []*http.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]*http.Request(nil), s.got...)
}

// ownHost allows the hosts of the given URLs at any address, as a repository's own.
func ownHost(urls ...string) func(string) bool {
	return func(hp string) bool {
		for _, u := range urls {
			if HostPort(u) == hp {
				return true
			}
		}
		return false
	}
}

// An icon on a repository's own host downloads at any address, over https only, without
// credentials, cookies or a referrer, at most MaxIconBytes; an http URL, a URL with a user, a
// redirect to http, a body or a declared length over the bound and an error status are refused.
func TestFetchIcon(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\nnot really a PNG")
	big := make([]byte, MaxIconBytes+1)
	s := newIconServer(t, func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/icon.png":
			_, _ = w.Write(png)
		case "/moved.png":
			http.Redirect(w, req, "/icon.png", http.StatusFound)
		case "/plain.png":
			http.Redirect(w, req, "http://"+req.Host+"/icon.png", http.StatusFound)
		case "/big.png":
			_, _ = w.Write(big)
		case "/declared.png":
			w.Header().Set("Content-Length", "9999999")
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, req)
		}
	})
	f := &IconFetcher{Base: s.srv.Client()}
	own := ownHost(s.srv.URL)
	ctx := context.Background()
	data, err := f.FetchIcon(ctx, s.srv.URL+"/moved.png", own)
	if err != nil || string(data) != string(png) {
		t.Fatalf("icon %q %v", data, err)
	}
	for _, req := range s.requests() {
		for _, h := range []string{"Authorization", "Cookie", "Referer"} {
			if req.Header.Get(h) != "" {
				t.Errorf("%s carried %s", req.URL.Path, h)
			}
		}
		if req.Header.Get("User-Agent") != "Cogpit" {
			t.Errorf("%s: user agent %q", req.URL.Path, req.Header.Get("User-Agent"))
		}
	}
	before := len(s.requests())
	for what, tc := range map[string]struct {
		url, want string
	}{
		"http":            {strings.Replace(s.srv.URL, "https://", "http://", 1) + "/icon.png", "https only"},
		"a user":          {strings.Replace(s.srv.URL, "https://", "https://me:pw@", 1) + "/icon.png", "user or password"},
		"not a URL":       {"::not a url", "does not parse"},
		"a relative path": {"icons/web.png", "does not parse"},
	} {
		if _, err := f.FetchIcon(ctx, tc.url, own); err == nil || !strings.Contains(err.Error(), tc.want) || strings.Contains(err.Error(), "pw") {
			t.Errorf("%s: %v", what, err)
		}
	}
	if n := len(s.requests()); n != before {
		t.Errorf("refused URLs made %d requests", n-before)
	}
	for what, tc := range map[string]struct {
		path, want string
	}{
		"a redirect to http":    {"/plain.png", "https repository's downloads stay on https"},
		"a body over the bound": {"/big.png", "larger than the 256.0 KiB"},
		"a declared length":     {"/declared.png", "larger than the 256.0 KiB"},
		"not found":             {"/gone.png", "404"},
	} {
		if _, err := f.FetchIcon(ctx, s.srv.URL+tc.path, own); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v", what, err)
		}
	}
	if _, err := f.FetchIcon(ctx, s.srv.URL+"/plain.png", own); !errors.Is(err, ErrNotHTTPS) {
		t.Errorf("a redirect to http is not ErrNotHTTPS: %v", err)
	}
}

// An icon whose host is no repository's own and not at a public address is refused before any
// request: the loopback, the user's network, link-local addresses. A redirect from a repository's
// own host to another such host is refused too.
func TestFetchIconRefusesPrivateAddresses(t *testing.T) {
	s := newIconServer(t, func(w http.ResponseWriter, req *http.Request) { _, _ = w.Write([]byte("x")) })
	other := newIconServer(t, func(w http.ResponseWriter, req *http.Request) {
		http.Redirect(w, req, s.srv.URL+"/icon.png", http.StatusFound)
	})
	f := &IconFetcher{Base: s.srv.Client()}
	if _, err := f.FetchIcon(context.Background(), s.srv.URL+"/icon.png", nil); !errors.Is(err, ErrIconAddress) {
		t.Errorf("an icon on the loopback: %v", err)
	}
	if _, err := f.FetchIcon(context.Background(), other.srv.URL+"/icon.png", ownHost(other.srv.URL)); !errors.Is(err, ErrIconAddress) {
		t.Errorf("a redirect from a repository's host to another loopback host: %v", err)
	}
	if n := len(s.requests()); n != 0 {
		t.Errorf("the refused host saw %d requests", n)
	}
	for addr, public := range map[string]bool{
		"127.0.0.1": false, "10.1.2.3": false, "172.16.0.1": false, "192.168.1.1": false, "169.254.169.254": false,
		"100.64.0.1": false, "0.0.0.0": false, "224.0.0.1": false, "192.0.2.7": false, "::1": false, "fd00::1": false,
		"fe80::1": false, "::ffff:10.0.0.1": false, "2001:db8::1": false,
		"8.8.8.8": true, "140.82.112.3": true, "2606:4700:4700::1111": true, "::ffff:8.8.8.8": true,
	} {
		if got := PublicAddress(netip.MustParseAddr(addr)); got != public {
			t.Errorf("%s: public %v", addr, got)
		}
	}
}

// Through a proxy the proxy resolves the icon's host: the proxy itself may be at a private
// address, and the icon goes through it.
func TestFetchIconThroughTheProxy(t *testing.T) {
	s := newIconServer(t, func(w http.ResponseWriter, req *http.Request) { _, _ = w.Write([]byte("icon")) })
	proxy := testkit.NewForwardProxy(t, "", "")
	f := &IconFetcher{Base: s.srv.Client(), Proxy: func(*http.Request) (*url.URL, error) { return proxy.URL, nil }}
	data, err := f.FetchIcon(context.Background(), s.srv.URL+"/icon.png", nil)
	if err != nil || string(data) != "icon" {
		t.Fatalf("through the proxy: %q %v", data, err)
	}
	u, _ := url.Parse(s.srv.URL)
	if proxy.Connects()[u.Host] != 1 {
		t.Errorf("the proxy tunnelled %v", proxy.Connects())
	}
}
