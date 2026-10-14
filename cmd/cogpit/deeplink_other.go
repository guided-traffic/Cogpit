//go:build !darwin

package main

import (
	"log/slog"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// installDeepLinks returns no queue: only the macOS app registers the cogpit URL scheme so far
// (D-47). Windows and Linux packages do not exist yet (G22); registering the scheme there (a
// registry key, a .desktop file's MimeType) comes with them.
func installDeepLinks(*slog.Logger) *engine.DeepLinkQueue { return nil }
