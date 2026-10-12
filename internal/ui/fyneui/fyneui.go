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
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
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
	// PerfMode, with PerfReport, picks the measurement: PerfScroll (the default) or PerfGate.
	PerfMode string
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
	// Stalls are the distinct stalls, each with its start (seconds after the scroll phase started)
	// and how long the first callback behind it waited; at most 20.
	Stalls []PerfStall `json:"stalls,omitempty"`
	// Phases covers the whole measurement, start-up included.
	Phases []PerfPhase `json:"phases,omitempty"`
}

// PerfPhase is one phase of a measurement: "startup", "scroll", "views", "yaml". Times are seconds
// since the meter started; stall times are relative to the phase's start.
type PerfPhase struct {
	Name     string  `json:"name"`
	FromS    float64 `json:"fromS"`
	ToS      float64 `json:"toS"`
	Samples  int     `json:"samples"`
	P50Ms    float64 `json:"p50Ms"`
	P99Ms    float64 `json:"p99Ms"`
	MaxMs    float64 `json:"maxMs"`
	Over16Ms int     `json:"over16Ms"`
	// Stalls of the phase, at most 20.
	Stalls []PerfStall `json:"stalls,omitempty"`
	// Switches are the view switches of the "views" phase.
	Switches []PerfSwitch `json:"switches,omitempty"`
	// Keystrokes is the typing of the "yaml" phase.
	Keystrokes *PerfKeys `json:"keystrokes,omitempty"`
}

// PerfSwitch is one view switch: when it was dispatched (seconds since the meter started), how
// long until the table showed the view's rows (-1: never within the dwell time), and the longest
// wait of a main-goroutine callback within a second after it.
type PerfSwitch struct {
	View        string  `json:"view"`
	AtS         float64 `json:"atS"`
	RowsAfterMs float64 `json:"rowsAfterMs"`
	Rows        int     `json:"rows"`
	MaxLateMs   float64 `json:"maxLateMs"`
}

// PerfKeys is the typing into the YAML editor: one character per frame into the middle of the
// document. Work is the widget's own time per keystroke (edit, tokenize, place); the phase's
// callback latencies show whether keystroke and repaint fit a frame.
type PerfKeys struct {
	Count     int     `json:"count"`
	DocBytes  int     `json:"docBytes"`
	SetTextMs float64 `json:"setTextMs"`
	WorkP50Ms float64 `json:"workP50Ms"`
	WorkP99Ms float64 `json:"workP99Ms"`
	WorkMaxMs float64 `json:"workMaxMs"`
}

// PerfStall is one stall of the main goroutine the jank meter saw.
type PerfStall struct {
	AtS float64 `json:"atS"`
	Ms  float64 `json:"ms"`
}

// Engine is the part of *engine.Session the shell uses. It exists so that the shell can be tested
// against a fake with canned data; the desktop app passes the real session to [Run].
type Engine interface {
	Dispatch(ctx context.Context, inv engine.Invocation) (any, error)
	UIState() engine.UIState
	Table(view string) *engine.TableSnapshot
	Detail(ref model.ObjectRef) (*engine.DetailModel, error)
	ObjectMetrics(ref model.ObjectRef) (*engine.ObjectMetrics, error)
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
	// After the app stops, engine events and tickers may still post work for a moment until the
	// shell unsubscribes; Fyne logs an error for each fyne.Do into a stopped app, so drop them.
	var stopped atomic.Bool
	a.Lifecycle().SetOnStopped(func() { stopped.Store(true) })
	do := func(fn func()) {
		if !stopped.Load() {
			fyne.Do(fn)
		}
	}
	sh := newShell(s, a, w, opts, do)
	sh.setConcealed = setConcealedClipboard // the real app's system clipboard (H-25)
	s.SetConfirmer(sh.confirmer)
	w.SetContent(sh.root)
	w.SetOnClosed(sh.clearSecretClipboard) // a copied Secret value does not outlive the window on the clipboard
	w.SetMainMenu(sh.mainMenu())
	w.Resize(fyne.NewSize(1440, 900))
	// Cogpit's own title bar on macOS needs the window, which exists once the app started: queued
	// from OnStarted, the install runs after the window's creation (titlebar.go).
	a.Lifecycle().SetOnStarted(func() { do(sh.titlebar.install) })
	sh.start()
	if opts.PerfReport != nil {
		sh.startPerf()
	}
	// Ctrl+C or SIGTERM quits the way closing the window does, so the caller's cleanup runs (the
	// MCP token file is removed, the informers stop) instead of the process ending mid-way.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sig)
	ran := make(chan struct{})
	defer close(ran)
	go func() {
		select {
		case <-sig:
			do(a.Quit)
		case <-ran:
		}
	}()
	w.ShowAndRun()
	stopped.Store(true)
	if sh.perf != nil {
		sh.perf.finish() // the window closed before PerfDuration ended, or there is none
	}
	sh.stop()
	s.SetConfirmer(nil)
	return nil
}
