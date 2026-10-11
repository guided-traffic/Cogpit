// Package fyneui is Cogpit's desktop shell on Fyne: the window layout after Lens (sidebar, view
// header with namespace filter and search, table, detail drawer, dock, status bar), Cogpit's theme
// with its status tokens, the context picker and the confirmation dialog.
//
// Layer L4, and the only package tree that imports Fyne. The shell renders engine state and
// changes it only by dispatching ui.* commands; engine events reach widgets through fyne.Do.
package fyneui

import (
	"errors"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// ErrNotImplemented marks a Phase 0 stub.
var ErrNotImplemented = errors.New("not implemented")

// Options configure the shell.
type Options struct {
	// Title is the window title; empty means "Cogpit".
	Title string
	// MCPCommand is the `claude mcp add` command shown in the Help menu; empty hides it.
	MCPCommand string
	// AutoConnect connects this context at start instead of showing the picker first.
	AutoConnect string
	// PerfReport, when set, measures main-goroutine latency (a jank meter) and reports it.
	PerfReport func(PerfStats)
}

// PerfStats is what the jank meter measured: how late callbacks posted to the main goroutine
// every frame ran, a proxy for frame time.
type PerfStats struct {
	Samples   int     `json:"samples"`
	P50Ms     float64 `json:"p50Ms"`
	P99Ms     float64 `json:"p99Ms"`
	MaxMs     float64 `json:"maxMs"`
	Over16Ms  int     `json:"over16Ms"`
	DurationS float64 `json:"durationS"`
}

// Run builds the window for the session, installs the GUI confirmer and runs Fyne's event loop
// until the window closes.
func Run(s *engine.Session, opts Options) error {
	_, _ = s, opts
	return ErrNotImplemented
}
