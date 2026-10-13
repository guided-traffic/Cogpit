package engine

import (
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Proxies (G13; ui-parity › Cluster settings › Proxy and Preferences › HTTP proxy for non-cluster
// traffic): an HTTP(S) or SOCKS5 proxy for one cluster's API traffic (ClusterSettings.Proxy, applied
// through rest.Config.Proxy at the next connect, connector.go and internal/cluster), and one for
// the traffic that is no cluster's (Preferences.HTTPProxy: the Helm repositories' downloads and kubectl's). Both
// are kept in Cogpit's own files, never in a kubeconfig; a cluster's replaces its kubeconfig's
// proxy-url for Cogpit only (Cogpit's reading, inferred: Lens's precedence is not captured).
//
// A proxy URL can carry a user and a password, so it is a credential (docs/security/credentials.md):
// it is kept in the owner-only settings file as typed, and everywhere else it is shown with the
// password replaced by "•••" (RedactProxyURL): the UI state and the settings commands' results
// (what agents read), the audit log's parameters, the connection log; and no error quotes it. The
// window's field shows the redacted form too, and a value whose password is still "•••" keeps the
// password set before (MergeProxyPassword), so the user can change the host without typing it again.
//
// Only the user sets a proxy, in Cogpit's window (ErrProxyGUIOnly): a proxy decides where Cogpit
// sends the user's cluster traffic, with the user's credentials in it, and an agent that could set
// one and then connect (cluster.connect is a UI command) would route the user's identity through a
// host of its choosing; behind TLS that host sees only the server's name, but with
// insecure-skip-tls-verify or a plain-http server it reads the bearer token. Agents read the setting
// with its password left out, and may clear it. Cogpit's default (the lead proposed letting agents
// set it), not the owner's decision.

// Bounds and marks of a proxy URL.
const (
	maxProxyURL = 2048
	// ProxyRedacted stands for a proxy's password wherever the URL is shown.
	ProxyRedacted = "•••"
	// proxyNotURL is what a value that is not a proxy URL is shown as: never the value itself.
	proxyNotURL = "(not a valid proxy URL)"
)

// ProxyPlaceholder is the proxy field's placeholder (captured from Lens's cluster settings).
const ProxyPlaceholder = "http://<address>:<port>"

// ErrProxyGUIOnly refuses an agent's or the CLI's proxy: only the user sets one, in the window.
var ErrProxyGUIOnly = errors.New("only the user, in Cogpit's own window, can set a proxy: it decides where Cogpit " +
	"sends the user's traffic and may carry a password; agents and the CLI can read it (without its password) and clear it")

// errProxyNotURL is the refusal of a value that does not parse; it never quotes the value.
var errProxyNotURL = errors.New("the proxy is not a URL such as http://proxy.example.com:3128")

// proxySchemes are the proxies Go's HTTP client and client-go's streaming connections (exec,
// attach, port forward) speak.
var proxySchemes = []string{"http", "https", "socks5"}

// CheckProxyURL says why a value cannot be a proxy: "" (none) passes; otherwise an http://,
// https:// or socks5:// URL with a host, an optional port, user and password, and no path, query
// or fragment. Its errors never quote the value, which may hold a password.
func CheckProxyURL(raw string) error {
	_, err := ParseProxyURL(raw)
	return err
}

// ParseProxyURL returns a checked proxy URL (CheckProxyURL); nil for "".
func ParseProxyURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	if len(raw) > maxProxyURL || !utf8.ValidString(raw) || strings.IndexFunc(raw, func(r rune) bool {
		return unicode.IsControl(r) || unicode.IsSpace(r)
	}) >= 0 {
		return nil, fmt.Errorf("the proxy is one URL of at most %d bytes, without spaces", maxProxyURL)
	}
	u, err := url.Parse(raw)
	if err != nil || u.Opaque != "" {
		return nil, errProxyNotURL
	}
	scheme := strings.ToLower(u.Scheme)
	switch {
	case !slices.Contains(proxySchemes, scheme):
		return nil, errors.New("a proxy URL starts with http://, https:// or socks5://")
	case u.Hostname() == "":
		return nil, errors.New("the proxy URL names no host")
	case u.Path != "" && u.Path != "/", u.RawQuery != "", u.Fragment != "", u.ForceQuery:
		return nil, errors.New("a proxy URL has no path, query or fragment: http://<address>:<port>")
	}
	if p := u.Port(); p != "" {
		if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
			return nil, errors.New("the proxy's port is a number from 1 to 65535")
		}
	}
	u.Scheme = scheme
	return u, nil
}

// RedactProxyURL returns a proxy URL as it may be shown: its password replaced by "•••"
// ("http://alice:•••@proxy:3128"), the user name kept. A URL without a password is shown as it
// is; a value that is not a proxy URL is never shown.
func RedactProxyURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := ParseProxyURL(raw)
	if err != nil || u == nil {
		return proxyNotURL
	}
	if u.User == nil {
		return raw
	}
	if _, has := u.User.Password(); !has {
		return raw
	}
	user := url.User(u.User.Username()).String()
	return u.Scheme + "://" + user + ":" + ProxyRedacted + "@" + u.Host + u.Path
}

// proxyPasswordMark is how a shown proxy URL carries the redacted password.
const proxyPasswordMark = ":" + ProxyRedacted + "@"

// MergeProxyPassword returns the proxy value to keep when the user changes it to next: a value
// that still carries the redacted password ("•••", as the field shows it) keeps the password of
// old; any other value is next itself. A redacted password with no password before is refused.
func MergeProxyPassword(old, next string) (string, error) {
	next = strings.TrimSpace(next)
	if !strings.Contains(next, proxyPasswordMark) {
		return next, nil
	}
	ou, err := ParseProxyURL(old)
	if err != nil || ou == nil || ou.User == nil {
		return "", errors.New("the proxy's password is shown as " + ProxyRedacted + ": type it again")
	}
	if _, has := ou.User.Password(); !has {
		return "", errors.New("the proxy's password is shown as " + ProxyRedacted + ": type it again")
	}
	_, escaped, _ := strings.Cut(ou.User.String(), ":") // the password as it stands in a URL
	return strings.Replace(next, proxyPasswordMark, ":"+escaped+"@", 1), nil
}

// shownClusterSettings returns the settings as the UI state shows them: each proxy's password
// redacted. The map is a copy.
func shownClusterSettings(all map[string]ClusterSettings) map[string]ClusterSettings {
	out := maps.Clone(all)
	for id, cs := range out {
		if cs.Proxy != "" {
			cs.Proxy = RedactProxyURL(cs.Proxy)
			out[id] = cs
		}
	}
	return out
}

// proxyParams are the command parameters that carry a proxy URL: the audit log shows them redacted.
var proxyParams = []string{"proxy", "httpProxy"}

// redactProxyParams replaces the proxy URLs among a command's parameters by their redacted form
// (a value that is not a string by its type) and reports whether it changed any.
func redactProxyParams(params map[string]any) bool {
	changed := false
	for _, k := range proxyParams {
		v, ok := params[k]
		if !ok || v == nil {
			continue
		}
		if s, isString := v.(string); isString {
			if r := RedactProxyURL(s); r != s {
				params[k], changed = r, true
			}
			continue
		}
		params[k], changed = fmt.Sprintf("<a %T, not a string; not logged>", v), true
	}
	return changed
}

// proxyOf returns a context's proxy as kept, read from the file once; "" without one.
func (c *ClusterSettingsStore) proxyOf(context string) (string, error) {
	if c == nil {
		return "", nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.load()
	return c.by[context].Proxy, err
}

// clusterProxy returns the parsed proxy a context's connections go through; nil without one.
func (c *ClusterSettingsStore) clusterProxy(context string) (*url.URL, error) {
	raw, _ := c.proxyOf(context) // an unreadable file is reported where it loads: no proxy then
	u, err := ParseProxyURL(raw)
	if err != nil {
		return nil, fmt.Errorf("the proxy in this cluster's settings is not valid (%w); fix it in Cluster Settings › Proxy", err)
	}
	return u, nil
}

// proxyLogLine is the connection log's line about the proxy a connect goes through; "" without.
func (s *Session) proxyLogLine(context string) string {
	raw, _ := s.opts.ClusterSettings.proxyOf(context)
	if raw == "" {
		return ""
	}
	return "Through the proxy " + RedactProxyURL(raw) + " (Cluster Settings › Proxy)"
}

// NonClusterProxy picks the proxy of a request that goes to no cluster (a Helm repository's index
// or chart, a kubectl download, kubectl.go): the Preferences' HTTP proxy, else the environment's (HTTPS_PROXY, NO_PROXY), as
// before the preference existed. Its signature is http.Transport.Proxy's; the preference is read
// for every request, so a change applies to the next download.
func (s *Session) NonClusterProxy(req *http.Request) (*url.URL, error) {
	if p := s.opts.Preferences; p != nil {
		p.mu.Lock()
		_ = p.load() // an unreadable file is reported where it loads: the defaults then
		raw := p.prefs.HTTPProxy
		p.mu.Unlock()
		if raw != "" {
			u, err := ParseProxyURL(raw)
			if err != nil {
				return nil, fmt.Errorf("the HTTP proxy in the Preferences is not valid (%w); fix it in Preferences › Proxy", err)
			}
			return u, nil
		}
	}
	return envProxy(req)
}

// envProxy is the environment's proxy for traffic that is no cluster's; tests replace it.
var envProxy = http.ProxyFromEnvironment
