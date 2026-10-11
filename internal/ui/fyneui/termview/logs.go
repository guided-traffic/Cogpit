package termview

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime/debug"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Bounds of the log view (budget P5 in docs/developer/testing.md: follow 10,000 lines per second,
// memory bounded by the scrollback).
const (
	logScrollback   = 10_000                 // lines kept
	logMaxBytes     = 32 << 20               // bytes kept over all lines; the oldest go first
	logMaxLineBytes = 64 << 10               // longer lines are cut here
	logFlushEvery   = 100 * time.Millisecond // at most one UI update per interval
	logTabWidth     = 8
	// logMaxShownBytes bounds the text one row renders. It is wider than any screen (the row is
	// clipped anyway), and measuring a 64 KiB line on every refresh would cost the main goroutine.
	logMaxShownBytes = 2048
)

// truncatedMark ends a line cut at logMaxLineBytes.
const truncatedMark = " …"

// Logs shows a log stream, following it by default.
//
// A reader goroutine splits the stream into lines, strips escape sequences and control
// characters (log lines are untrusted text, never markup) and queues them; a flusher hands the
// queue to the main goroutine at most every logFlushEvery. Only the main goroutine touches the
// widgets and the scrollback, and the list renders only the visible rows.
type Logs struct {
	widget.BaseWidget
	stream io.ReadCloser

	// Reader side, guarded by mu.
	mu       sync.Mutex
	pending  *lineRing
	received int64
	ended    bool
	endErr   error

	// Main goroutine only.
	lines       *lineRing
	follow      bool
	list        *widget.List
	followCheck *widget.Check
	status      *widget.Label
	batch       []string

	stop       chan struct{}
	flushed    chan struct{} // closed when the flusher has published the end of the stream or stopped
	closeOnce  sync.Once
	closeErr   error
	onFlush    func(d time.Duration, lines int) // test hook: time spent on the main goroutine per batch
	lineStride float32
}

// NewLogs shows stream; it reads until the stream ends or Close is called.
func NewLogs(stream io.ReadCloser) *Logs {
	l := &Logs{
		stream:  stream,
		pending: newLineRing(logScrollback, logMaxBytes),
		lines:   newLineRing(logScrollback, logMaxBytes),
		follow:  true,
		stop:    make(chan struct{}),
		flushed: make(chan struct{}),
	}
	l.list = widget.NewList(
		func() int { return l.lines.len() },
		func() fyne.CanvasObject {
			t := canvas.NewText("", theme.Color(theme.ColorNameForeground))
			t.TextStyle = fyne.TextStyle{Monospace: true}
			return t
		},
		func(id widget.ListItemID, o fyne.CanvasObject) {
			t := o.(*canvas.Text)
			t.Text = shownText(l.lines.at(id))
			t.Color = theme.Color(theme.ColorNameForeground)
			t.Refresh()
		},
	)
	l.list.HideSeparators = true
	l.followCheck = widget.NewCheck("Follow", nil)
	l.followCheck.Checked = true
	l.followCheck.OnChanged = l.setFollow
	l.status = widget.NewLabel("0 lines")
	l.ExtendBaseWidget(l)
	l.startReading()
	return l
}

// Close stops reading and closes the stream. It is idempotent.
func (l *Logs) Close() error {
	l.closeOnce.Do(func() {
		close(l.stop)
		if l.stream != nil {
			l.closeErr = l.stream.Close()
		}
	})
	return l.closeErr
}

// CreateRenderer implements fyne.Widget.
func (l *Logs) CreateRenderer() fyne.WidgetRenderer {
	bar := container.NewHBox(l.followCheck, layout.NewSpacer(), l.status)
	return widget.NewSimpleRenderer(container.NewBorder(bar, nil, nil, nil, l.list))
}

// setFollow is the Follow toggle: on, the view jumps to the newest line and stays there.
func (l *Logs) setFollow(on bool) {
	l.follow = on
	if on {
		l.list.ScrollToBottom()
	}
}

func (l *Logs) startReading() {
	if l.stream == nil {
		l.ended = true
		close(l.flushed)
		return
	}
	go l.read()
	go l.flush()
}

// read splits the stream into cleaned lines.
func (l *Logs) read() {
	var err error
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("log view: internal error: %v", r)
			slog.Error("termview: log reader panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
		l.mu.Lock()
		l.ended, l.endErr = true, err
		l.mu.Unlock()
	}()
	br := bufio.NewReaderSize(l.stream, logMaxLineBytes)
	for {
		line, rerr := br.ReadSlice('\n')
		if errors.Is(rerr, bufio.ErrBufferFull) {
			l.add(cleanLogLine(line) + truncatedMark)
			for errors.Is(rerr, bufio.ErrBufferFull) { // drop the rest of the line
				_, rerr = br.ReadSlice('\n')
			}
			if rerr != nil {
				err = rerr
				return
			}
			continue
		}
		if len(line) > 0 {
			l.add(cleanLogLine(line))
		}
		if rerr != nil {
			err = rerr
			return
		}
	}
}

func (l *Logs) add(line string) {
	l.mu.Lock()
	l.pending.push(line)
	l.received++
	l.mu.Unlock()
}

// flush publishes queued lines to the main goroutine at most every logFlushEvery. It waits for
// each update to finish, so a slow main goroutine slows the flusher down instead of queueing
// updates; the reader keeps reading and the bounded queue drops the oldest lines.
func (l *Logs) flush() {
	defer close(l.flushed)
	defer func() {
		if r := recover(); r != nil {
			slog.Error("termview: log flusher panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		}
	}()
	tick := time.NewTicker(logFlushEvery)
	defer tick.Stop()
	var shownReceived int64
	for {
		select {
		case <-l.stop:
			return
		case <-tick.C:
		}
		l.mu.Lock()
		batch := l.pending.drain(l.batch[:0])
		received, ended, endErr := l.received, l.ended, l.endErr
		l.mu.Unlock()
		l.batch = batch[:0] // reused once the main goroutine is done with it (DoAndWait)

		if len(batch) > 0 || received != shownReceived || ended {
			fyne.DoAndWait(func() {
				start := time.Now()
				l.apply(batch, received, ended, endErr)
				if l.onFlush != nil {
					l.onFlush(time.Since(start), len(batch))
				}
			})
			shownReceived = received
		}
		clear(batch)
		if ended {
			return
		}
	}
}

// apply adds a batch to the scrollback and updates the widgets. Main goroutine only.
func (l *Logs) apply(batch []string, received int64, ended bool, endErr error) {
	evictedBefore := l.lines.evicted
	for _, s := range batch {
		l.lines.push(s)
	}
	evicted := l.lines.evicted - evictedBefore

	l.status.SetText(statusText(l.lines.len(), received, ended, endErr))
	if l.follow {
		l.list.Refresh()
		l.list.ScrollToBottom()
		return
	}
	// Not following: keep the lines in view where they are. Evicted lines shift every index, so
	// the offset moves up by their height.
	offset := l.list.GetScrollOffset()
	l.list.Refresh()
	if evicted > 0 {
		l.list.ScrollToOffset(offset - float32(evicted)*l.stride())
	}
}

// stride is the height of one list row.
func (l *Logs) stride() float32 {
	if l.lineStride == 0 {
		t := canvas.NewText("M", theme.Color(theme.ColorNameForeground))
		t.TextStyle = fyne.TextStyle{Monospace: true}
		l.lineStride = t.MinSize().Height + l.list.Theme().Size(theme.SizeNamePadding)
	}
	return l.lineStride
}

func statusText(kept int, received int64, ended bool, endErr error) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d lines", kept)
	if dropped := received - int64(kept); dropped > 0 {
		fmt.Fprintf(&b, " (%d older dropped)", dropped)
	}
	switch {
	case ended && (endErr == nil || errors.Is(endErr, io.EOF)):
		b.WriteString(" · stream ended")
	case ended:
		b.WriteString(" · stream ended: " + cleanLogLine([]byte(endErr.Error())))
	}
	return b.String()
}

// cleanLogLine turns one raw log line into display text: no line ending, no escape sequences,
// no control characters, tabs expanded, valid UTF-8.
func cleanLogLine(line []byte) string {
	line = trimEOL(line)
	if isPlainText(line) {
		return string(line)
	}
	var b strings.Builder
	b.Grow(len(line))
	col := 0
	for i := 0; i < len(line); {
		c := line[i]
		switch {
		case c == esc:
			i = skipEscape(line, i)
			continue
		case c == '\t':
			n := logTabWidth - col%logTabWidth
			b.WriteString(strings.Repeat(" ", n))
			col += n
			i++
			continue
		case c < 0x20 || c == 0x7f:
			i++
			continue
		case c < utf8.RuneSelf:
			b.WriteByte(c)
			col++
			i++
			continue
		}
		r, size := utf8.DecodeRune(line[i:])
		i += size
		switch {
		case r == utf8.RuneError && size == 1:
			b.WriteRune(utf8.RuneError)
		case r >= 0x80 && r < 0xa0: // C1 controls
			continue
		default:
			b.WriteRune(r)
		}
		col++
	}
	return b.String()
}

// shownText is the part of a line a row renders.
func shownText(s string) string {
	if len(s) <= logMaxShownBytes {
		return s
	}
	cut := logMaxShownBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + truncatedMark
}

func trimEOL(line []byte) []byte {
	for len(line) > 0 && (line[len(line)-1] == '\n' || line[len(line)-1] == '\r') {
		line = line[:len(line)-1]
	}
	return line
}

// isPlainText reports whether line needs no cleaning: printable ASCII only, the common case.
func isPlainText(line []byte) bool {
	for _, c := range line {
		if c < 0x20 || c >= 0x7f {
			return false
		}
	}
	return true
}

// skipEscape returns the index after the escape sequence starting at line[i] (an ESC): a control
// sequence (ESC [ … final), a string (OSC, DCS, APC, PM, SOS: up to BEL or ST), or ESC and one
// more character. A sequence cut off by the end of the line ends there.
func skipEscape(line []byte, i int) int {
	i++ // ESC
	if i >= len(line) {
		return i
	}
	switch line[i] {
	case '[':
		for i++; i < len(line); i++ {
			if line[i] >= 0x40 && line[i] <= 0x7e {
				return i + 1
			}
		}
		return i
	case ']', 'P', '_', '^', 'X':
		for i++; i < len(line); i++ {
			if line[i] == 0x07 {
				return i + 1
			}
			if line[i] == esc && i+1 < len(line) && line[i+1] == '\\' {
				return i + 2
			}
		}
		return i
	default:
		return i + 1
	}
}

// lineRing is a bounded scrollback: at most maxLines lines and maxBytes bytes; the oldest lines
// go first.
type lineRing struct {
	buf      []string
	head     int
	n        int
	bytes    int
	maxBytes int
	evicted  int64 // lines dropped from the front, in total
}

func newLineRing(maxLines, maxBytes int) *lineRing {
	return &lineRing{buf: make([]string, maxLines), maxBytes: maxBytes}
}

func (r *lineRing) len() int { return r.n }

// at returns line i, 0 being the oldest kept; "" when out of range.
func (r *lineRing) at(i int) string {
	if i < 0 || i >= r.n {
		return ""
	}
	return r.buf[(r.head+i)%len(r.buf)]
}

func (r *lineRing) push(s string) {
	if r.n == len(r.buf) {
		r.evict()
	}
	r.buf[(r.head+r.n)%len(r.buf)] = s
	r.n++
	r.bytes += len(s)
	for r.bytes > r.maxBytes && r.n > 1 {
		r.evict()
	}
}

func (r *lineRing) evict() {
	r.bytes -= len(r.buf[r.head])
	r.buf[r.head] = ""
	r.head = (r.head + 1) % len(r.buf)
	r.n--
	r.evicted++
}

// drain appends all lines to dst, oldest first, and empties the ring.
func (r *lineRing) drain(dst []string) []string {
	for i := 0; i < r.n; i++ {
		j := (r.head + i) % len(r.buf)
		dst = append(dst, r.buf[j])
		r.buf[j] = ""
	}
	r.head, r.n, r.bytes = 0, 0, 0
	return dst
}
