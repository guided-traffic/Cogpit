package commands

import (
	"context"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// refuseProxy refuses a proxy set by the CLI or a link (engine.ErrProxyGUIOnly, engine/proxy.go):
// they may clear a proxy (an empty value), not set one. The user's window sets one; an agent's
// proxy waits for the user's acceptance whatever the allowance of agent writes (setsProxy, A20,
// owner 2026-10-14 evening). A proxy left out passes.
func refuseProxy(ctx context.Context, proxy *string) error {
	if o := engine.OriginOf(ctx); setsProxy(proxy) && o != engine.OriginGUI && !o.IsAgent() {
		return engine.ErrProxyGUIOnly
	}
	return nil
}

// setsProxy reports whether a settings change sets a proxy (not only clears it or leaves it).
func setsProxy(proxy *string) bool { return proxy != nil && strings.TrimSpace(*proxy) != "" }

// proxyDetail is the confirmation's line for an agent's proxy: where the traffic goes, its
// password never shown.
func proxyDetail(what, proxy string) string {
	return "It sends " + what + " through the proxy " + engine.RedactProxyURL(proxy) +
		"; the proxy sees the hosts Cogpit talks to, and wherever TLS is not checked, the credentials too."
}
