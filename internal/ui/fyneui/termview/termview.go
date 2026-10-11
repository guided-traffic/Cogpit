// Package termview holds the dock's stream widgets: a terminal on fyne-io/terminal connected to a
// terminal.Session with RunWithConnection (local shell or pod shell, with resize), and a log view
// that follows a log stream with a bounded scrollback.
//
// Both show untrusted output (docs/security/trust-boundaries.md): the terminal passes it through
// an escape-sequence filter (sanitize.go), the log view strips escape sequences entirely.
//
// Layer L4.
package termview

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"runtime/debug"
	"sync"

	fyneterm "github.com/fyne-io/terminal"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/terminal"
)

// sessionEndedNotice is written into the terminal after the session's last output.
const sessionEndedNotice = "\r\n\x1b[90m[session ended]\x1b[0m\r\n"

// Terminal shows one terminal session.
//
// Copy and paste are the widget's (fyne-io/terminal): dragging selects (with Alt: a block),
// double-click selects a word; on macOS ⌘C copies the selection and ⌘V pastes, elsewhere
// Ctrl+Shift+C and Ctrl+Shift+V; a right click pastes the last selection. Bracketed paste is
// honoured when the program asks for it.
type Terminal struct {
	widget.BaseWidget
	sess  terminal.Session
	term  *fyneterm.Terminal
	sizes chan fyneterm.Config
	ended chan struct{}

	mu        sync.Mutex
	started   bool
	closeOnce sync.Once
	closeErr  error
}

// NewTerminal connects a terminal widget to sess. The connection starts when the widget is first
// laid out with a usable size. Closing the widget closes the session.
func NewTerminal(sess terminal.Session) *Terminal {
	t := &Terminal{
		sess:  sess,
		term:  fyneterm.New(),
		sizes: make(chan fyneterm.Config, 8),
		ended: make(chan struct{}),
	}
	t.ExtendBaseWidget(t)
	// The widget reports its size in cells here whenever a resize changes it; it drops a report
	// when the channel is full, so the forwarder drains to the latest.
	t.term.AddListener(t.sizes)
	return t
}

// Done is closed when the session's output has ended and "[session ended]" is shown, or after
// Close.
func (t *Terminal) Done() <-chan struct{} { return t.ended }

// Text returns the terminal's screen as text. Call it on the main goroutine.
func (t *Terminal) Text() string { return t.term.Text() }

// Close ends the session. It is idempotent and does not block on the session.
func (t *Terminal) Close() error {
	t.closeOnce.Do(func() {
		t.mu.Lock()
		notStarted := !t.started
		t.started = true // a later layout must not start the connection
		t.mu.Unlock()

		t.term.Close()
		if t.sess != nil {
			t.closeErr = t.sess.Close() // the pending Read in run returns io.EOF and the loop ends
		}
		if notStarted {
			close(t.ended)
		}
	})
	return t.closeErr
}

// CreateRenderer implements fyne.Widget.
func (t *Terminal) CreateRenderer() fyne.WidgetRenderer {
	return &terminalRenderer{t: t}
}

// start connects the widget to the session once it has a size in cells; the widget otherwise
// polls for one, racing the main goroutine.
func (t *Terminal) start() {
	t.mu.Lock()
	if t.started || t.sess == nil {
		t.mu.Unlock()
		return
	}
	t.started = true
	t.mu.Unlock()

	go t.forwardSizes()
	go t.run()
}

// run feeds the session's filtered output to the widget until it ends.
func (t *Terminal) run() {
	defer close(t.ended)
	defer func() {
		if r := recover(); r != nil {
			// The widget's parser panicked on its reading goroutine (see escFilter for the known
			// cases). Its state is unknown: end the session.
			slog.Error("termview: terminal widget panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
		_ = t.sess.Close()
	}()
	out := &endNotice{src: newEscFilter(t.sess), notice: []byte(sessionEndedNotice)}
	if err := t.term.RunWithConnection(sessionInput{t.sess}, out); err != nil {
		slog.Warn("termview: terminal connection ended", "error", err.Error())
	}
}

// forwardSizes tells the session about the widget's size in cells.
func (t *Terminal) forwardSizes() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("termview: resize forwarder panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	var lastCols, lastRows uint16
	for {
		var cfg fyneterm.Config
		select {
		case cfg = <-t.sizes:
		case <-t.ended:
			return
		}
	drain:
		for {
			select {
			case cfg = <-t.sizes:
			default:
				break drain
			}
		}
		cols, rows := cells(cfg.Columns), cells(cfg.Rows)
		if cols == 0 || rows == 0 || (cols == lastCols && rows == lastRows) {
			continue
		}
		if err := t.sess.Resize(cols, rows); err == nil {
			lastCols, lastRows = cols, rows
		}
	}
}

func cells(n uint) uint16 {
	if n > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(n)
}

type terminalRenderer struct {
	t *Terminal
}

func (r *terminalRenderer) Layout(size fyne.Size) {
	r.t.term.Move(fyne.NewPos(0, 0))
	r.t.term.Resize(size)
	if minSize := r.t.term.MinSize(); size.Width >= minSize.Width && size.Height >= minSize.Height {
		r.t.start()
	}
}

func (r *terminalRenderer) MinSize() fyne.Size           { return r.t.term.MinSize() }
func (r *terminalRenderer) Refresh()                     { r.t.term.Refresh() }
func (r *terminalRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.t.term} }
func (r *terminalRenderer) Destroy()                     {}

// sessionInput is the widget's input side: keystrokes go to the session. Closing it does nothing;
// Terminal.Close and the end of run close the session.
type sessionInput struct {
	sess terminal.Session
}

func (s sessionInput) Write(p []byte) (int, error) { return s.sess.Write(p) }
func (s sessionInput) Close() error                { return nil }

// endNotice passes the session's output on and, when it ends, appends a notice and reports
// io.EOF. Any error ends it: the widget's loop treats errors other than EOF as transient and
// would spin on them.
type endNotice struct {
	src    io.Reader
	notice []byte
	ended  bool
}

func (e *endNotice) Read(p []byte) (int, error) {
	if !e.ended {
		n, err := e.src.Read(p)
		if err == nil {
			return n, nil
		}
		e.ended = true
		if !errors.Is(err, io.EOF) {
			slog.Debug("termview: session output ended", "error", err.Error())
		}
		if n > 0 {
			return n, nil
		}
	}
	if len(e.notice) > 0 {
		n := copy(p, e.notice)
		e.notice = e.notice[n:]
		return n, nil
	}
	return 0, io.EOF
}
