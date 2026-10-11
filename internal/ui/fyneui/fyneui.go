// Package fyneui is Cogpit's desktop shell on Fyne: the window layout after Lens (sidebar, view
// header with namespace filter and search, table, detail drawer, dock, status bar), Cogpit's theme
// with its status tokens, the context picker and the confirmation dialog.
//
// Layer L4, and the only package tree that imports Fyne. The shell renders engine state and
// changes it only by dispatching ui.* commands; engine events reach widgets through fyne.Do.
package fyneui

import (
	"context"
	"io"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/terminal"
)

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
	// PerfDuration, with PerfReport, scrolls the table through its rows for this long, then
	// reports and quits. Zero measures until the window closes, without scrolling.
	PerfDuration time.Duration
	// Theme picks the palette; empty means ThemeDark.
	Theme ThemeMode
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

// Engine is the part of *engine.Session the shell uses. It exists so that the shell can be tested
// against a fake with canned data; the desktop app passes the real session to [Run].
type Engine interface {
	Dispatch(ctx context.Context, inv engine.Invocation) (any, error)
	UIState() engine.UIState
	Table(view string) *engine.TableSnapshot
	Detail(ref model.ObjectRef) (*engine.DetailModel, error)
	Subscribe(fn func(engine.Event)) (cancel func())
	Diagnostics() engine.Diagnostics
	Registry() *engine.Registry
	Contexts() ([]engine.ContextInfo, error)
	OpenLogs(ctx context.Context, tabID string) (io.ReadCloser, error)
	OpenTerminal(ctx context.Context, tabID string) (terminal.Session, error)
	SetConfirmer(c engine.Confirmer)
}

var _ Engine = (*engine.Session)(nil)

// Run builds the window for the session, installs the GUI confirmer and runs Fyne's event loop
// until the window closes.
func Run(s *engine.Session, opts Options) error {
	a := app.NewWithID("io.guided-traffic.cogpit")
	a.Settings().SetTheme(newTheme(opts.Theme))
	title := opts.Title
	if title == "" {
		title = "Cogpit"
	}
	w := a.NewWindow(title)
	w.SetPadded(false) // the bars and the sidebar reach the window's edges
	sh := newShell(s, a, w, opts, fyne.Do)
	s.SetConfirmer(sh.confirmer)
	w.SetContent(sh.root)
	w.SetMainMenu(sh.mainMenu())
	w.Resize(fyne.NewSize(1440, 900))
	sh.start()
	if opts.PerfReport != nil {
		sh.startPerf()
	}
	w.ShowAndRun()
	if sh.perf != nil {
		sh.perf.finish() // the window closed before PerfDuration ended, or there is none
	}
	sh.stop()
	s.SetConfirmer(nil)
	return nil
}
