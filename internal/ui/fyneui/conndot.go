package fyneui

import (
	"image/color"
	"strings"
	"time"
	"weak"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// A cluster's connection shows as a dot, never a word (F13, the owner, 2026-10-16, amending D-58's
// S3): Connected in the Running containers' green, Connection failed and Login expired red, Not
// connected grey; while the cluster is (re)connecting its dots blink in that green, 500 ms full and
// 500 ms dimmed, for at least three intervals even when the reconnect succeeds at once, so that the
// user sees it (the minimum is the window's: agents read the true state in ui_state). Every dot of
// that cluster blinks in step: the top bar's, the status bar's, its hotbar tile's, its navigator
// rows' and its welcome-page row's. The dot's tooltip names the state, and a click on it opens the
// cluster's connection log (connectlog.go), never also its row or tile.

const (
	blinkPhase   = 500 * time.Millisecond // each phase, full and dimmed (the owner)
	blinkMinimum = 3 * time.Second        // three intervals at least (the owner)
	blinkDim     = 0.3                    // the dimmed phase's alpha: the dot stays (the lead's)
	blinkStill   = 0.5                    // the steady dot's alpha under Reduce motion (the lead's)
)

// blinkingState reports whether a state is a (re)connect: Connecting or Interrupted.
func blinkingState(s engine.ConnectionState) bool {
	return s == engine.Connecting || s == engine.Interrupted
}

// connectionColor maps a connection state to a colour token: Connected and a (re)connect in the
// Running containers' green (the owner, F13), a failed connect and an expired login red, Not
// connected grey.
func connectionColor(c engine.Connection) fyne.ThemeColorName {
	switch c.State {
	case engine.Connected, engine.Connecting, engine.Interrupted:
		return colorContainerRunning
	case engine.ConnectError, engine.LoginExpired:
		return colorStatusError
	}
	return colorStatusUnknown
}

// blinker is the one clock of the blinking dots, on the main goroutine. It follows the window's
// connection (observe): a (re)connect of a cluster starts a blink of its dots that lasts while the
// engine says it (re)connects and at least blinkMinimum; Login expired, Connection failed, a
// disconnect or another cluster end it at once. A ticker runs only while a blink lasts and
// refreshes the cluster's dots at each phase; without a blink nothing runs.
type blinker struct {
	sh     *shell
	now    func() time.Time
	reduce func() bool // Reduce motion (reducemotion_darwin.go), read when a blink starts
	ticker func(on bool)

	ctx   string    // the cluster that blinks; "" while none does
	live  bool      // the engine still says it (re)connects
	start time.Time // the blink's start: its phases count from here, so its dots blink in step
	until time.Time // it blinks at least until then
	still bool      // Reduce motion: a steady half-alpha dot instead

	dots []weak.Pointer[connDot] // every dot made, to refresh the blinking cluster's
	stop chan struct{}           // the running ticker's; nil while none runs
}

func newBlinker(sh *shell) *blinker {
	b := &blinker{sh: sh, now: time.Now, reduce: reduceMotion}
	b.ticker = b.runTicker
	return b
}

// observe follows the window's cluster and its connection state.
func (b *blinker) observe(ctx string, s engine.ConnectionState) {
	now := b.now()
	switch {
	case ctx != "" && blinkingState(s):
		if ctx != b.ctx || !b.on(now) {
			b.ctx, b.start, b.until, b.still = ctx, now, now.Add(blinkMinimum), b.reduce()
		}
		b.live = true
	case ctx != "" && ctx == b.ctx && s == engine.Connected:
		b.live = false // the minimum may still run
	default:
		b.ctx, b.live = "", false
	}
	b.tick()
}

// on reports whether a blink lasts at now.
func (b *blinker) on(now time.Time) bool {
	return b.ctx != "" && (b.live || now.Before(b.until))
}

// look is how a dot of ctx in state s shows now: its colour token and alpha.
func (b *blinker) look(ctx string, s engine.ConnectionState, base fyne.ThemeColorName) (fyne.ThemeColorName, float32) {
	now := b.now()
	if ctx == "" || ctx != b.ctx || !b.on(now) || (!blinkingState(s) && s != engine.Connected) {
		return base, 1
	}
	switch {
	case b.still:
		return colorContainerRunning, blinkStill
	case (now.Sub(b.start)/blinkPhase)%2 == 1:
		return colorContainerRunning, blinkDim
	}
	return colorContainerRunning, 1
}

// tick refreshes the blinking cluster's dots and runs the ticker while a blink lasts; the last
// tick of a blink refreshes them once more, steady.
func (b *blinker) tick() {
	now := b.now()
	on := b.on(now)
	ctx := b.ctx
	if !on {
		b.ctx = ""
	}
	live := b.dots[:0]
	for _, w := range b.dots {
		d := w.Value()
		if d == nil {
			continue
		}
		live = append(live, w)
		if d.context != "" && (d.context == ctx || d.blinked) {
			d.Refresh()
		}
	}
	clear(b.dots[len(live):])
	b.dots = live
	b.ticker(on)
}

// add keeps a dot to refresh while its cluster blinks.
func (b *blinker) add(d *connDot) { b.dots = append(b.dots, weak.Make(d)) }

// runTicker starts the ticker when on and none runs, and stops it otherwise: a goroutine that
// posts tick to the main goroutine every phase.
func (b *blinker) runTicker(on bool) {
	switch {
	case on && b.stop == nil:
		stop := make(chan struct{})
		b.stop = stop
		go func() {
			t := time.NewTicker(blinkPhase)
			defer t.Stop()
			for {
				select {
				case <-t.C:
					b.sh.do(b.tick)
				case <-stop:
					return
				case <-b.sh.stopTickers:
					return
				}
			}
		}()
	case !on && b.stop != nil:
		close(b.stop)
		b.stop = nil
	}
}

// connDot is a cluster's connection dot: its colour from the state (connectionColor) or the
// blinker, its state in the tooltip, and a click opening the cluster's connection log.
type connDot struct {
	widget.BaseWidget
	sh       *shell
	context  string // the cluster; "" for none, which has no log
	conn     engine.Connection
	color    fyne.ThemeColorName // the colour when it does not blink
	blinked  bool                // drawn blinking last time: refreshed once more when the blink ends
	tipExtra string              // a line the site adds to the tooltip
}

var (
	_ fyne.Tappable      = (*connDot)(nil)
	_ desktop.Hoverable  = (*connDot)(nil)
	_ desktop.Cursorable = (*connDot)(nil)
)

func newConnDot(sh *shell) *connDot {
	d := &connDot{sh: sh, color: colorStatusUnknown}
	d.ExtendBaseWidget(d)
	if sh != nil && sh.blink != nil {
		sh.blink.add(d)
	}
	return d
}

// set shows a cluster's connection; base is the colour when it does not blink, "" for
// connectionColor's.
func (d *connDot) set(ctx string, c engine.Connection, base fyne.ThemeColorName) {
	if base == "" {
		base = connectionColor(c)
	}
	if ctx == d.context && c == d.conn && base == d.color {
		return
	}
	d.context, d.conn, d.color = ctx, c, base
	d.Refresh()
}

// look is the dot's colour token and alpha now.
func (d *connDot) look() (fyne.ThemeColorName, float32) {
	if d.sh == nil || d.sh.blink == nil {
		return d.color, 1
	}
	return d.sh.blink.look(d.context, d.conn.State, d.color)
}

// tipText names the state: why and since when where the engine says so, the next attempt, and
// what a click does.
func (d *connDot) tipText() string {
	c := d.conn
	parts := []string{connectionText(c)}
	tf := d.sh.timeFormat()
	if !c.Since.IsZero() && (c.State == engine.Interrupted || c.State == engine.LoginExpired) {
		parts = append(parts, "since "+tf.FormatClock(c.Since))
	}
	if c.State == engine.Interrupted {
		next := "trying now"
		if !c.NextAttempt.IsZero() {
			next = "next attempt " + attemptIn(c.NextAttempt.Sub(d.sh.now()))
		}
		parts = append(parts, next)
	}
	text := strings.Join(parts, " · ")
	switch {
	case strings.TrimSpace(c.Reason) != "" && (c.State == engine.Interrupted || c.State == engine.LoginExpired):
		text += "\n" + upperFirst(strings.TrimSpace(c.Reason))
	case c.Error != "" && (c.State == engine.ConnectError || c.State == engine.Disconnected):
		text += "\n" + kubeconfig.DisplayText(c.Error, maxShownPath)
	}
	if d.tipExtra != "" {
		text += "\n" + d.tipExtra
	}
	if d.context != "" {
		text += "\nClick for its connection updates"
	}
	return text
}

// Tapped implements fyne.Tappable: the cluster's connection log; the row or tile under the dot
// gets no tap.
func (d *connDot) Tapped(*fyne.PointEvent) {
	if d.sh == nil {
		return
	}
	d.sh.tip.hideFor(d)
	if d.context != "" {
		d.sh.kc.showConnectLog(d.context)
	}
}

// Cursor implements desktop.Cursorable: a pointer over a dot that opens a log.
func (d *connDot) Cursor() desktop.Cursor {
	if d.context != "" {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn implements desktop.Hoverable.
func (d *connDot) MouseIn(ev *desktop.MouseEvent) {
	if d.sh != nil {
		d.sh.tip.hover(d, d.tipText(), ev.AbsolutePosition)
	}
}

// MouseMoved implements desktop.Hoverable.
func (d *connDot) MouseMoved(ev *desktop.MouseEvent) {
	if d.sh != nil {
		d.sh.tip.moved(d, ev.AbsolutePosition)
	}
}

// MouseOut implements desktop.Hoverable.
func (d *connDot) MouseOut() {
	if d.sh != nil {
		d.sh.tip.hideFor(d)
	}
}

// CreateRenderer implements fyne.Widget.
func (d *connDot) CreateRenderer() fyne.WidgetRenderer {
	r := &connDotRenderer{d: d, c: canvas.NewCircle(color.Transparent)}
	r.Refresh()
	return r
}

type connDotRenderer struct {
	d *connDot
	c *canvas.Circle
}

func (r *connDotRenderer) diameter() float32 { return r.d.Theme().Size(sizeDotDiameter) }

func (r *connDotRenderer) Layout(size fyne.Size) {
	d := min(r.diameter(), size.Width, size.Height)
	r.c.Move(fyne.NewPos((size.Width-d)/2, (size.Height-d)/2))
	r.c.Resize(fyne.NewSize(d, d))
}

func (r *connDotRenderer) MinSize() fyne.Size {
	d := r.diameter()
	return fyne.NewSize(d, d)
}

func (r *connDotRenderer) Refresh() {
	name, a := r.d.look()
	r.d.blinked = name != r.d.color || a != 1
	c := themeColor(r.d.Theme(), name)
	if a < 1 {
		c = withAlpha(c, scaleAlpha(color.NRGBAModel.Convert(c).(color.NRGBA).A, a))
	}
	r.c.FillColor = c
	r.Layout(r.d.Size())
	r.c.Refresh()
}

func (r *connDotRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.c} }
func (r *connDotRenderer) Destroy()                     {}

// scaleAlpha is an alpha of 255 times a: a dimmed or steady dot's.
func scaleAlpha(alpha uint8, a float32) uint8 { return uint8(float32(alpha) * a) }
