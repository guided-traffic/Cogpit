package helm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Chart icons (D-46, HELM2): a repository's index names each chart version's icon as a URL on
// any host. Cogpit fetches an icon only when the window shows it (a row of the Charts view or a
// chart's drawer) and the Preferences' "Load chart icons" is on; never for agents. The fetch is
// none of the repository's business: https only, a redirect too; no credentials (not the
// repository's, whatever pass_credentials_all says), no cookies, no referrer; at most
// MaxIconBytes; through the Preferences' proxy like every download that is no cluster's. A host
// that resolves to a loopback, private, link-local or otherwise non-public address is refused,
// so that a chart's index cannot make Cogpit send requests into the user's own network, unless
// the host is one of the repositories' own (an internal repository's icons beside its charts).
// Each request dials the address it checked, so a name that resolves elsewhere a moment later
// changes nothing; through a proxy the proxy resolves the name and the check does not apply.
// What an icon is (PNG, JPEG or SVG) is checked by the engine (engine.CheckIcon).

// MaxIconBytes bounds a chart icon's download (Cogpit's default, D-46).
const MaxIconBytes int64 = 256 << 10

const (
	iconTimeout      = 20 * time.Second // one icon's download, connection and body
	maxIconRedirects = 5
)

// ErrIconAddress refuses an icon whose host is not at a public address (and is none of the
// repositories' own).
var ErrIconAddress = errors.New("the icon's host is not at a public address, and Cogpit fetches icons from the internet or from a chart repository's own host only")

// IconFetcher downloads chart icons. Its zero value is ready to use; it must not be copied once
// used.
type IconFetcher struct {
	// Base is the client whose transport (its TLS settings) icons are fetched with; nil: Go's
	// default transport. Its timeout, redirects and cookie jar are not used.
	Base *http.Client
	// Proxy picks each request's proxy, as http.Transport.Proxy does; nil: the environment's.
	Proxy func(*http.Request) (*url.URL, error)
	// Max bounds a download; zero means MaxIconBytes.
	Max int64

	mu         sync.Mutex
	transports map[string]*http.Transport // by the host allowed at any address ("": public only)
	proxies    sync.Map                   // the proxies' addresses the transports were told to use
}

// HostPort is a URL's host (lower case) and port (the scheme's when it names none), "" when it
// does not parse: what an icon's host is compared with a repository's by.
func HostPort(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return ""
	}
	port := u.Port()
	if port == "" {
		port = map[string]string{schemeHTTPS: "443", schemeHTTP: "80", "socks5": "1080"}[u.Scheme]
	}
	return net.JoinHostPort(strings.ToLower(u.Hostname()), port)
}

// FetchIcon downloads the icon at rawURL. own reports whether a host (HostPort's form) is one of
// the chart repositories' own, which may be at any address.
func (f *IconFetcher) FetchIcon(ctx context.Context, rawURL string, own func(hostPort string) bool) ([]byte, error) {
	u, err := url.Parse(rawURL)
	switch {
	case err != nil || u.Host == "":
		return nil, errors.New("the icon's URL does not parse")
	case u.Scheme != schemeHTTPS:
		return nil, fmt.Errorf("%s: Cogpit fetches icons over https only", RedactURL(rawURL))
	case u.User != nil:
		return nil, errors.New("an icon URL with a user or password is not fetched")
	}
	allowed := ""
	if hp := HostPort(rawURL); own != nil && own(hp) {
		allowed = hp
	}
	shown := RedactURL(rawURL)
	c := &http.Client{Transport: f.transport(allowed), Timeout: iconTimeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		switch {
		case req.URL.Scheme != schemeHTTPS:
			return fmt.Errorf("a redirect: %w", ErrNotHTTPS)
		case req.URL.User != nil:
			return errors.New("a redirect to a URL with a user or password is not followed")
		case len(via) >= maxIconRedirects:
			return errors.New("too many redirects")
		}
		req.Header.Del("Referer")
		return nil
	}}
	ctx, cancel := context.WithTimeout(ctx, iconTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, errors.New("the icon's URL does not make a request")
	}
	req.Header.Set("User-Agent", "Cogpit")
	req.Header.Set("Accept", "image/png, image/jpeg, image/svg+xml")
	resp, err := c.Do(req)
	if err != nil {
		var ue *url.Error
		if errors.As(err, &ue) {
			return nil, &url.Error{Op: ue.Op, URL: RedactURL(ue.URL), Err: ue.Err}
		}
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	limit := f.Max
	if limit <= 0 {
		limit = MaxIconBytes
	}
	switch {
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("%s: %s", shown, resp.Status)
	case resp.ContentLength > limit:
		return nil, fmt.Errorf("%s: larger than the %s Cogpit downloads of an icon", shown, model.FormatBytes(limit))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s: larger than the %s Cogpit downloads of an icon", shown, model.FormatBytes(limit))
	}
	return data, nil
}

// transport returns the transport that dials public addresses and, at any address, allowed (a
// repository's own host:port; "" none): one per allowed host, so that a connection kept open to
// a repository's host is never reused for another repository's icon.
func (f *IconFetcher) transport(allowed string) *http.Transport {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t := f.transports[allowed]; t != nil {
		return t
	}
	var base *http.Transport
	if f.Base != nil {
		base, _ = f.Base.Transport.(*http.Transport)
	}
	if base == nil {
		base = http.DefaultTransport.(*http.Transport)
	}
	t := base.Clone()
	t.Proxy = func(req *http.Request) (*url.URL, error) {
		pick := f.Proxy
		if pick == nil {
			pick = http.ProxyFromEnvironment
		}
		u, err := pick(req)
		if u != nil {
			f.proxies.Store(HostPort(u.String()), true) // the dial to it is the proxy's, not the icon host's
		}
		return u, err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return f.dial(ctx, dialer, network, addr, allowed)
	}
	if f.transports == nil {
		f.transports = map[string]*http.Transport{}
	}
	f.transports[allowed] = t
	return t
}

// dial connects to addr: to a proxy as it is; to the allowed host at any address; else to the
// first of its addresses that is public, by that address, so that the name is resolved once.
func (f *IconFetcher) dial(ctx context.Context, d *net.Dialer, network, addr, allowed string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if _, ok := f.proxies.Load(net.JoinHostPort(strings.ToLower(host), port)); ok || strings.EqualFold(addr, allowed) {
		return d.DialContext(ctx, network, addr)
	}
	ips, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	last := ErrIconAddress
	for _, ip := range ips {
		if !PublicAddress(ip) {
			continue
		}
		c, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return c, nil
		}
		last = err
	}
	return nil, last
}

// PublicAddress reports whether an address is on the internet: not loopback, private (RFC 1918,
// unique local), link-local, carrier-grade NAT (100.64.0.0/10), multicast, unspecified or reserved
// for documentation and benchmarks.
func PublicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	for _, p := range nonPublicPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}

// nonPublicPrefixes are the ranges PublicAddress refuses beyond the standard library's tests.
var nonPublicPrefixes = func() []netip.Prefix {
	ranges := []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "192.0.2.0/24", "198.18.0.0/15",
		"198.51.100.0/24", "203.0.113.0/24", "240.0.0.0/4", "64:ff9b:1::/48", "2001:db8::/32"}
	out := make([]netip.Prefix, 0, len(ranges))
	for _, s := range ranges {
		out = append(out, netip.MustParsePrefix(s))
	}
	return out
}()
