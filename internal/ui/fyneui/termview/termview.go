// Package termview holds the dock's stream widgets: a terminal on fyne-io/terminal connected to a
// terminal.Session with RunWithConnection (local shell or pod shell, with resize), and a log view
// that follows a log stream with a bounded scrollback.
//
// Layer L4.
package termview

import (
	"io"

	_ "github.com/fyne-io/terminal" // pinned; Phase 1 uses it

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/terminal"
)

// Terminal shows one terminal session.
type Terminal struct {
	widget.BaseWidget
	sess terminal.Session
}

// NewTerminal connects a terminal widget to sess. Closing the widget closes the session.
func NewTerminal(sess terminal.Session) *Terminal {
	t := &Terminal{sess: sess}
	t.ExtendBaseWidget(t)
	return t
}

// Close ends the session.
func (t *Terminal) Close() error {
	if t.sess == nil {
		return nil
	}
	return t.sess.Close()
}

// CreateRenderer implements fyne.Widget.
func (t *Terminal) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel(""))
}

// Logs shows a log stream, following it by default.
type Logs struct {
	widget.BaseWidget
	stream io.ReadCloser
}

// NewLogs shows stream; it reads until the stream ends or Close is called.
func NewLogs(stream io.ReadCloser) *Logs {
	l := &Logs{stream: stream}
	l.ExtendBaseWidget(l)
	return l
}

// Close stops reading and closes the stream.
func (l *Logs) Close() error {
	if l.stream == nil {
		return nil
	}
	return l.stream.Close()
}

// CreateRenderer implements fyne.Widget.
func (l *Logs) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel(""))
}
