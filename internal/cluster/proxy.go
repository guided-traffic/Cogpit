package cluster

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"k8s.io/client-go/rest"
)

// A connection's proxy (Options.Proxy): the proxy a cluster's settings name in Cogpit
// (engine/proxy.go), applied to the client configuration every client of the connection is built
// from, so that lists, watches, discovery, access reviews, the Helm SDK's clients and the
// streaming connections (exec, attach, port forward: client-go's SPDY and WebSocket dialers read
// rest.Config.Proxy) all go through it. It replaces the kubeconfig's proxy-url, and the
// environment's HTTPS_PROXY, for this connection only; the kubeconfig file is not touched.

// applyProxy routes the configuration's requests through proxy; nil leaves client-go's choice (the
// kubeconfig's proxy-url, else the environment's).
func applyProxy(cfg *rest.Config, proxy *url.URL) {
	if proxy == nil {
		return
	}
	p := *proxy
	cfg.Proxy = http.ProxyURL(&p)
}

// scrubProxySecret keeps a proxy's password out of a connect error: client-go's and net/http's
// errors name the proxy's address, not its credentials, so this is a guard for an error that
// someday quotes the URL. The password is replaced as typed and as it stands escaped in a URL.
func scrubProxySecret(err error, proxy *url.URL) error {
	if err == nil || proxy == nil || proxy.User == nil {
		return err
	}
	pw, has := proxy.User.Password()
	if !has || pw == "" {
		return err
	}
	_, escaped, _ := strings.Cut(proxy.User.String(), ":")
	scrub := func(s string) string {
		s = strings.ReplaceAll(s, pw, "•••")
		if escaped != pw {
			s = strings.ReplaceAll(s, escaped, "•••")
		}
		return s
	}
	var ce *ConnectError
	if errors.As(err, &ce) {
		out := *ce
		out.Problem = scrub(ce.Problem)
		out.Context = scrub(ce.Context)
		if out.Problem == ce.Problem && out.Context == ce.Context {
			return err
		}
		return &out
	}
	if text := scrub(err.Error()); text != err.Error() {
		return errors.New(text)
	}
	return err
}
