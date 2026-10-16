package fyneui

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// alertsButton is the top bar's alerts pill, left of the notifications' bell (the owner, 2026-10-16
// night): a warning sign and the number of alerts on a rounded background, as the status bar's MCP
// pill, its corners the same. Its colour is the bell's while there is none, the warning colour while there are warnings
// only, the error colour once one is critical (the theme's status tokens). Prepared only: it has no
// source of alerts and no action yet (the owner: "Der Button hat noch keine Funktion"), so it shows
// 0 and its tooltip says so. Cogpit's own; Lens's title bar has no such pill.
type alertsButton struct {
	widget.BaseWidget
	sh       *shell
	critical int
	warning  int
}

var _ desktop.Hoverable = (*alertsButton)(nil)

func newAlertsButton(sh *shell) *alertsButton {
	b := &alertsButton{sh: sh}
	b.ExtendBaseWidget(b)
	return b
}

// set shows the counts of critical and warning alerts; unchanged counts cost nothing.
func (b *alertsButton) set(critical, warning int) {
	critical, warning = max(critical, 0), max(warning, 0)
	if critical == b.critical && warning == b.warning {
		return
	}
	b.critical, b.warning = critical, warning
	b.Refresh()
}

// colour is the pill's colour token: the bell's without alerts, else the worst alert's.
func (b *alertsButton) colour() fyne.ThemeColorName {
	switch {
	case b.critical > 0:
		return colorStatusError
	case b.warning > 0:
		return colorStatusWarning
	}
	return theme.ColorNameForeground
}

// countText is the number in the pill: every alert, "99+" beyond 99.
func (b *alertsButton) countText() string {
	n := b.critical + b.warning
	if n > 99 {
		return "99+"
	}
	return strconv.Itoa(n)
}

// tipText is the pill's tooltip.
func (b *alertsButton) tipText() string {
	return "Alerts: " + strconv.Itoa(b.critical) + " critical, " + strconv.Itoa(b.warning) +
		" warnings (no source of alerts yet)"
}

// MouseIn implements desktop.Hoverable.
func (b *alertsButton) MouseIn(ev *desktop.MouseEvent) {
	b.sh.tip.hover(b, b.tipText(), ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (b *alertsButton) MouseMoved(ev *desktop.MouseEvent) { b.sh.tip.moved(b, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (b *alertsButton) MouseOut() { b.sh.tip.hideFor(b) }

// CreateRenderer implements fyne.Widget.
func (b *alertsButton) CreateRenderer() fyne.WidgetRenderer {
	r := &alertsRenderer{b: b, bg: canvas.NewRectangle(color.Transparent), img: newIconImage(),
		count: canvas.NewText("", color.Transparent)}
	r.count.TextStyle = fyne.TextStyle{Bold: true}
	r.Refresh()
	return r
}

type alertsRenderer struct {
	b     *alertsButton
	bg    *canvas.Rectangle
	img   *iconImage
	count *canvas.Text
}

// metrics are the pill's inner sizes: the icon's side, the padding at its ends and between the
// icon and the number, and the pill's height (a line of caption text with the pill's padding, as
// the status bar's pills).
func (r *alertsRenderer) metrics() (icon, pad, height float32) {
	th := r.b.Theme()
	pad = th.Size(theme.SizeNamePadding)
	text := fyne.MeasureText("Mg", th.Size(theme.SizeNameCaptionText), r.count.TextStyle).Height
	icon = text
	return icon, pad, text + pad
}

func (r *alertsRenderer) MinSize() fyne.Size {
	icon, pad, h := r.metrics()
	return fyne.NewSize(2*pad+icon+pad/2+r.count.MinSize().Width, h)
}

func (r *alertsRenderer) Layout(size fyne.Size) {
	icon, pad, h := r.metrics()
	ms := r.MinSize()
	y := (size.Height - h) / 2 // the pill keeps its height and sits centred in the bar's row
	x := (size.Width - ms.Width) / 2
	r.bg.Move(fyne.NewPos(x, y))
	r.bg.Resize(fyne.NewSize(ms.Width, h))
	r.bg.CornerRadius = r.b.Theme().Size(sizeChipRadius) // the status bar's MCP pill's
	r.img.img.Move(fyne.NewPos(x+pad, y+(h-icon)/2))
	r.img.img.Resize(fyne.NewSize(icon, icon))
	ts := r.count.MinSize()
	r.count.Move(fyne.NewPos(x+pad+icon+pad/2, y+(h-ts.Height)/2))
	r.count.Resize(ts)
}

func (r *alertsRenderer) Refresh() {
	th := r.b.Theme()
	c := r.b.colour()
	fg := themeColor(th, c)
	r.img.set(th, theme.WarningIcon(), c)
	r.count.Text = r.b.countText()
	r.count.TextSize = th.Size(theme.SizeNameCaptionText)
	r.count.Color = fg
	r.bg.FillColor = withAlpha(fg, 0x2c) // as a tinted pill's (widgets.go)
	r.Layout(r.b.Size())
	r.bg.Refresh()
	r.count.Refresh()
}

func (r *alertsRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.img.img, r.count}
}

func (r *alertsRenderer) Destroy() {}
