package fyneui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The status line under the drawer's charts (G12, owner 2026-10-12, evening): one line per metrics
// provider, Prometheus first, then the Kubernetes Metrics Server, so the user sees why a chart is
// empty or stale. Each line has an icon in a status colour (answering, not answering, not allowed,
// not found, not looked for yet) and says what the provider serves now (history, live values,
// network, filesystem) and when it last answered; the tooltip over the line has the whole state:
// the service, the last good answer and the last failure (a class, never a response body). The
// data is the engine's (ObjectMetrics.Providers), which agents read in metrics.get and ui.state.
// Lens shows one source line; the wording and the icons here are Cogpit's (inferred).

// providerStatusLine is the status line: a row per provider, updated in place, and the button that
// sets the cluster's Prometheus (metricssettings.go).
type providerStatusLine struct {
	sh       *shell
	root     *fyne.Container
	rowsBox  *fyne.Container
	rows     []*providerRow
	settings *tipButton
}

func newProviderStatusLine(sh *shell) *providerStatusLine {
	l := &providerStatusLine{sh: sh, rowsBox: container.NewVBox(), settings: newPrometheusButton(sh)}
	l.root = container.NewVBox(l.rowsBox, container.NewHBox(l.settings))
	return l
}

// set shows the providers' states as of now; no providers hides the line.
func (l *providerStatusLine) set(providers []metrics.ProviderStatus, now time.Time) {
	shown := make([]metrics.ProviderStatus, 0, len(providers))
	for _, ps := range providers {
		if ps.Label != "" {
			shown = append(shown, ps)
		}
	}
	for len(l.rows) < len(shown) {
		r := newProviderRow(l.sh)
		l.rows = append(l.rows, r)
		l.rowsBox.Add(r)
	}
	if hasPrometheusSlot(shown) {
		l.settings.Show()
	} else {
		l.settings.Hide()
	}
	for i, r := range l.rows {
		if i >= len(shown) {
			r.Hide()
			continue
		}
		r.set(providerLine(shown[i], now, l.sh.timeFormat()))
		r.Show()
	}
	if len(shown) == 0 {
		l.root.Hide()
	} else {
		l.root.Show()
	}
	l.root.Refresh()
}

// providerLineText is one row's content.
type providerLineText struct {
	icon  fyne.Resource
	color fyne.ThemeColorName
	text  string
	tip   string
}

// servesText names what a provider serves: "history, live values, network, filesystem".
func servesText(serves []string) string {
	words := make([]string, 0, len(serves))
	for _, s := range serves {
		if s == metrics.ServesLive {
			s = "live values"
		}
		words = append(words, s)
	}
	return strings.Join(words, ", ")
}

// providerLine is the text, icon and tooltip of one provider's row; the tooltip's times of day are
// written in tf, the user's zone and format (D-31).
func providerLine(ps metrics.ProviderStatus, now time.Time, tf model.TimeFormat) providerLineText {
	name := ps.Label
	if ps.Service != "" {
		name += " " + ps.Service
	}
	since := func(t time.Time) string { return agoText(now.Sub(t)) }
	var l providerLineText
	var state string
	switch ps.Status {
	case metrics.StatusAvailable:
		l.icon, l.color = theme.ConfirmIcon(), colorStatusOK
		what := servesText(ps.Serves)
		if what == "" {
			what = "answering, not needed now"
		}
		l.text = name + ": " + what
		if !ps.LastOK.IsZero() {
			l.text += " · " + since(ps.LastOK)
		}
		state = "Found and answering."
	case metrics.StatusUnavailable:
		l.icon, l.color = theme.WarningIcon(), colorStatusWarning
		l.text = name + ": not answering"
		if ps.LastError != "" {
			l.text += ", " + ps.LastError
		}
		if !ps.LastOK.IsZero() {
			l.text += " · last answer " + since(ps.LastOK)
		}
		state = "Found, but it does not answer: its charts keep what they have."
	case metrics.StatusForbidden:
		l.icon, l.color = theme.ErrorIcon(), colorStatusError
		l.text = name + ": not allowed"
		if ps.LastError != "" {
			l.text += ", " + ps.LastError
		}
		state = "The current user may not read it."
	case metrics.StatusNotServed:
		l.icon, l.color = theme.InfoIcon(), colorMuted
		l.text = name + ": not found"
		state = "Not found in this cluster."
	default:
		l.icon, l.color = theme.QuestionIcon(), colorMuted
		l.text = name + ": not looked for yet"
		state = "Not looked for yet: Cogpit looks for it when a chart needs it."
	}
	tip := []string{name, state}
	if what := servesText(ps.Serves); what != "" {
		tip = append(tip, "Serves: "+what+".")
	}
	if !ps.LastOK.IsZero() {
		tip = append(tip, "Last good answer: "+tf.FormatTime(ps.LastOK)+" ("+since(ps.LastOK)+")")
	}
	if ps.LastError != "" {
		line := "Last error: " + ps.LastError
		if !ps.LastErrorAt.IsZero() {
			line += " at " + tf.FormatTime(ps.LastErrorAt) + " (" + since(ps.LastErrorAt) + ")"
		}
		tip = append(tip, line)
	}
	l.tip = strings.Join(tip, "\n")
	return l
}

// providerRow is one line of the status line: an icon in a status colour and a text that is cut
// to the drawer's width, with the provider's whole state as the tooltip over either.
type providerRow struct {
	widget.BaseWidget
	sh   *shell
	line providerLineText
	text *textLine
}

func newProviderRow(sh *shell) *providerRow {
	r := &providerRow{sh: sh, text: newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})}
	r.text.truncate = true
	r.ExtendBaseWidget(r)
	return r
}

func (r *providerRow) set(l providerLineText) {
	r.line = l
	r.text.SetText(l.text)
	r.text.SetColor(colorMuted)
	r.Refresh()
}

// MouseIn implements desktop.Hoverable.
func (r *providerRow) MouseIn(ev *desktop.MouseEvent) {
	r.sh.tip.hover(r, r.line.tip, ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (r *providerRow) MouseMoved(ev *desktop.MouseEvent) { r.sh.tip.moved(r, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (r *providerRow) MouseOut() { r.sh.tip.hideFor(r) }

// CreateRenderer implements fyne.Widget.
func (r *providerRow) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromResource(nil)
	img.FillMode = canvas.ImageFillContain
	rr := &providerRowRenderer{r: r, img: img}
	rr.Refresh()
	return rr
}

type providerRowRenderer struct {
	r   *providerRow
	img *canvas.Image
}

func (p *providerRowRenderer) side() float32 {
	return fyne.MeasureText("Mg", p.r.Theme().Size(theme.SizeNameCaptionText), fyne.TextStyle{}).Height
}

func (p *providerRowRenderer) Layout(size fyne.Size) {
	s := p.side()
	gap := p.r.Theme().Size(theme.SizeNameInnerPadding) / 2
	p.img.Move(fyne.NewPos(0, (size.Height-s)/2))
	p.img.Resize(fyne.NewSquareSize(s))
	p.r.text.Move(fyne.NewPos(s+gap, 0))
	p.r.text.Resize(fyne.NewSize(fyne.Max(size.Width-s-gap, 0), size.Height))
}

func (p *providerRowRenderer) MinSize() fyne.Size {
	return fyne.NewSize(p.side(), fyne.Max(p.side(), p.r.text.MinSize().Height))
}

func (p *providerRowRenderer) Refresh() {
	if p.r.line.icon != nil {
		p.img.Resource = themedIcon(p.r.line.icon, p.r.line.color)
	}
	p.img.Refresh()
	p.r.text.Refresh()
	p.Layout(p.r.Size())
}

func (p *providerRowRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{p.img, p.r.text}
}

func (p *providerRowRenderer) Destroy() {}
