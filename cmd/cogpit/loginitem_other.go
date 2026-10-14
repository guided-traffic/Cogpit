//go:build !darwin

package main

import "github.com/guided-traffic/cogpit/internal/engine"

// newLoginItem returns none: start at login is built for the macOS app only (D-48). Windows (a
// Run key or the Startup folder) and Linux (an XDG autostart file) come with their packages (G22);
// until then the Preferences row is not offered, rather than a switch that does nothing.
func newLoginItem() engine.LoginItem { return nil }
