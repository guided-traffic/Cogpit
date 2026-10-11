// Package commands registers Cogpit's engine commands, one file per area (ui.go, resources.go,
// dock.go, ...). Every command becomes a menu item, a palette entry and an MCP tool through the
// registry; the GUI and agents dispatch the same commands.
//
// Layer L3, with package engine.
package commands

import "github.com/guided-traffic/cogpit/internal/engine"

// RegisterAll registers every command. It fails on the first registration error (a duplicate id
// is a programming error found at start-up).
func RegisterAll(r *engine.Registry) error {
	_ = r
	return nil
}
