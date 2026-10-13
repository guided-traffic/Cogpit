package commands

import (
	"context"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// refuseProxy refuses a proxy set by any origin but the user's window (engine.ErrProxyGUIOnly,
// engine/proxy.go): agents and the CLI may clear a proxy (an empty value), not set one. A proxy
// left out passes.
func refuseProxy(ctx context.Context, proxy *string) error {
	if proxy != nil && strings.TrimSpace(*proxy) != "" && engine.OriginOf(ctx) != engine.OriginGUI {
		return engine.ErrProxyGUIOnly
	}
	return nil
}
