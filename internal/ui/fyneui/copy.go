package fyneui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"sigs.k8s.io/yaml"
)

// Copying values out of the detail drawer (D-25). Every value of the drawer has a copy icon that
// shows when the pointer is over it; a tap puts the value on the clipboard and says so in a toast.
// A value made of several entries (labels, annotations, ports, images, environment) copies the one
// entry the pointer is over, in YAML syntax: `app.kubernetes.io/name: grafana`. A Secret's value
// is copyable only once revealed, and the clipboard is cleared again 60 seconds after a copy of
// such a value if it still holds it.
//
// It is a GUI behaviour and no engine command: an agent reads values through object_get, so a
// command would add a way to put a Secret on the user's clipboard, nothing else.

// secretClipboardTTL is how long a copied Secret value stays on the clipboard.
const secretClipboardTTL = 60 * time.Second

// copyText puts text on the clipboard. A Secret's value (secret) is cleared again after the
// shell's secretClipTTL if the clipboard still holds it; a later copy of anything replaces it and
// cancels the clearing.
func (sh *shell) copyText(text string, secret bool) {
	if !secret || sh.setConcealed == nil || !sh.setConcealed(text) {
		sh.app.Clipboard().SetContent(text)
	}
	if sh.clipTimer != nil {
		sh.clipTimer.Stop()
	}
	sh.clipTimer, sh.clipSecret = nil, ""
	if !secret || text == "" {
		return
	}
	sh.clipSecret = text
	sh.clipTimer = time.AfterFunc(sh.secretClipTTL, func() { sh.do(sh.clearSecretClipboard) })
}

// clearSecretClipboard empties the clipboard if it still holds the Secret value copied last, and
// forgets the value. The shell runs it on stop too, so that a value does not outlive Cogpit's
// window on the clipboard.
func (sh *shell) clearSecretClipboard() {
	if sh.clipTimer != nil {
		sh.clipTimer.Stop()
	}
	if sh.clipSecret != "" && sh.app.Clipboard().Content() == sh.clipSecret {
		sh.app.Clipboard().SetContent("")
	}
	sh.clipTimer, sh.clipSecret = nil, ""
}

// copyHook says what the copy icon copies when the pointer is over a value.
type copyHook struct {
	d      *drawer
	text   string
	secret bool
}

func (h *copyHook) enter(w fyne.CanvasObject, natural float32) { h.d.showCopy(w, h, natural) }
func (h *copyHook) leave(w fyne.CanvasObject)                  { h.d.hideCopy(w) }

// copyFor returns the hook for a value, or nil for text that has none.
func (d *drawer) copyFor(text string, secret bool) *copyHook {
	if text == "" {
		return nil
	}
	return &copyHook{d: d, text: text, secret: secret}
}

// line returns a selectable text line of the drawer. copyText is what its icon copies; "" for none.
func (d *drawer) line(text string, fg fyne.ThemeColorName, size fyne.ThemeSizeName, style fyne.TextStyle, truncate bool, copyText string) *selLine {
	l := newTextLine(text, fg, size, style)
	l.truncate = truncate
	return newSelLine(l, d.copyFor(copyText, false), func(sel string) { d.sh.copyText(sel, false) })
}

// showCopy puts the copy icon by the value w, which the pointer entered. natural is the width of
// the value's text: the icon sits right after it, or at the value's right end when it is wider.
func (d *drawer) showCopy(w fyne.CanvasObject, h *copyHook, natural float32) {
	drv := fyne.CurrentApp().Driver()
	host := drv.AbsolutePositionForObject(d.host)
	at := drv.AbsolutePositionForObject(w).Subtract(host)
	ms := d.copyBtn.MinSize()
	x := fyne.Min(natural+d.sh.app.Settings().Theme().Size(theme.SizeNamePadding), w.Size().Width-ms.Width)
	y := float32(0)
	if lineH := fyne.MeasureText("Mg", d.sh.app.Settings().Theme().Size(theme.SizeNameText), fyne.TextStyle{}).Height; w.Size().Height <= 1.6*lineH {
		y = (w.Size().Height - ms.Height) / 2
	}
	d.copyOwner, d.copyHook = w, h
	d.hostLayout.over = at.AddXY(fyne.Max(x, 0), y)
	d.copyBtn.Show()
	d.host.Refresh()
}

// hideCopy hides the copy icon when w is the value it is shown for.
func (d *drawer) hideCopy(w fyne.CanvasObject) {
	if d.copyOwner == w && d.copyBtn.Visible() {
		d.copyBtn.Hide()
	}
}

// resetCopy forgets the value the icon was shown for: the drawer rebuilt or moved.
func (d *drawer) resetCopy() {
	d.copyOwner, d.copyHook = nil, nil
	if d.copyBtn != nil && d.copyBtn.Visible() {
		d.copyBtn.Hide()
	}
}

// copyHovered runs the copy icon: the value's text on the clipboard, and a toast that says so.
func (d *drawer) copyHovered() {
	h := d.copyHook
	if h == nil {
		return
	}
	d.sh.copyText(h.text, h.secret)
	if h.secret {
		d.sh.toast("info", fmt.Sprintf("Copied. The clipboard is cleared in %s.", ttlText(d.sh.secretClipTTL)))
		return
	}
	d.sh.toast("info", "Copied to the clipboard")
}

// ttlText renders a duration the toast names: "60 seconds".
func ttlText(t time.Duration) string {
	if t >= time.Minute && t%time.Minute == 0 {
		if t == time.Minute {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", int(t/time.Minute))
	}
	s := t.Seconds()
	if s == 1 {
		return "1 second"
	}
	if s == float64(int(s)) {
		return fmt.Sprintf("%d seconds", int(s))
	}
	return t.String()
}

// copyButton is the copy icon: a small button on a chip-coloured plate, hidden until the pointer is
// over a value. The drawer owns one and moves it from value to value (the table of a node has
// hundreds of values; a button each would cost).
type copyButton struct {
	widget.BaseWidget
	d *drawer
}

var (
	_ desktop.Hoverable  = (*copyButton)(nil)
	_ desktop.Cursorable = (*copyButton)(nil)
)

func newCopyButton(d *drawer) *copyButton {
	b := &copyButton{d: d}
	b.ExtendBaseWidget(b)
	b.Hide()
	return b
}

// Tapped implements fyne.Tappable.
func (b *copyButton) Tapped(*fyne.PointEvent) { b.d.copyHovered() }

// Cursor implements desktop.Cursorable.
func (b *copyButton) Cursor() desktop.Cursor { return desktop.PointerCursor }

// MouseIn implements desktop.Hoverable: the pointer moved from the value onto the icon, which the
// value's MouseOut has just hidden.
func (b *copyButton) MouseIn(ev *desktop.MouseEvent) {
	b.Show()
	b.d.sh.tip.hover(b, "Copy", ev.AbsolutePosition)
}

// MouseMoved implements desktop.Hoverable.
func (b *copyButton) MouseMoved(ev *desktop.MouseEvent) { b.d.sh.tip.moved(b, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (b *copyButton) MouseOut() {
	b.Hide()
	b.d.sh.tip.hideFor(b)
}

// CreateRenderer implements fyne.Widget.
func (b *copyButton) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(nil)
	img := canvas.NewImageFromResource(nil)
	img.FillMode = canvas.ImageFillContain
	r := &copyButtonRenderer{b: b, bg: bg, img: img}
	r.Refresh()
	return r
}

type copyButtonRenderer struct {
	b   *copyButton
	bg  *canvas.Rectangle
	img *canvas.Image
}

func (r *copyButtonRenderer) MinSize() fyne.Size {
	s := r.b.Theme().Size(theme.SizeNameInlineIcon) + 2*r.b.Theme().Size(theme.SizeNamePadding)
	return fyne.NewSquareSize(s)
}

func (r *copyButtonRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	pad := r.b.Theme().Size(theme.SizeNamePadding)
	r.img.Move(fyne.NewPos(pad, pad))
	r.img.Resize(size.SubtractWidthHeight(2*pad, 2*pad))
}

func (r *copyButtonRenderer) Refresh() {
	th := r.b.Theme()
	r.bg.FillColor = themeColor(th, colorChip)
	r.bg.StrokeColor = themeColor(th, theme.ColorNameInputBorder)
	r.bg.StrokeWidth = th.Size(theme.SizeNameInputBorder)
	r.bg.CornerRadius = th.Size(theme.SizeNameInputRadius)
	r.img.Resource = themedIcon(theme.ContentCopyIcon(), theme.ColorNameForeground)
	r.Layout(r.b.Size())
	r.bg.Refresh()
	r.img.Refresh()
}

func (r *copyButtonRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.img} }
func (r *copyButtonRenderer) Destroy()                     {}

// overlayLayout fills the space with its first object and keeps the second where the drawer put it
// (over), at its own size: the copy icon above the details.
type overlayLayout struct{ over fyne.Position }

func (l *overlayLayout) MinSize(objs []fyne.CanvasObject) fyne.Size { return objs[0].MinSize() }

func (l *overlayLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	objs[0].Move(fyne.Position{})
	objs[0].Resize(size)
	objs[1].Move(l.over)
	objs[1].Resize(objs[1].MinSize())
}

// hoverWrap makes a value that is no hoverable itself (a chip, a pill) show the copy icon.
type hoverWrap struct {
	widget.BaseWidget
	content fyne.CanvasObject
	hook    *copyHook
}

var _ desktop.Hoverable = (*hoverWrap)(nil)

// wrapCopy returns content as it is when it has no hook, else wrapped to show the copy icon.
func wrapCopy(content fyne.CanvasObject, hook *copyHook) fyne.CanvasObject {
	if hook == nil {
		return content
	}
	w := &hoverWrap{content: content, hook: hook}
	w.ExtendBaseWidget(w)
	return w
}

// MouseIn implements desktop.Hoverable.
func (w *hoverWrap) MouseIn(*desktop.MouseEvent) { w.hook.enter(w, w.content.MinSize().Width) }

// MouseMoved implements desktop.Hoverable.
func (w *hoverWrap) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (w *hoverWrap) MouseOut() { w.hook.leave(w) }

// CreateRenderer implements fyne.Widget.
func (w *hoverWrap) CreateRenderer() fyne.WidgetRenderer { return widget.NewSimpleRenderer(w.content) }

var entryKey = regexp.MustCompile(`^[A-Za-z0-9_./-]+$`)

// yamlEntry renders one entry of a value with several entries in YAML syntax. An entry of the form
// key=value (a label, an annotation, a selector, an environment variable) becomes `key: value`;
// any other entry is a YAML scalar: plain when YAML allows, else quoted.
func yamlEntry(entry string) string {
	var v any = entry
	if k, val, ok := strings.Cut(entry, "="); ok && entryKey.MatchString(k) {
		v = map[string]string{k: val}
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		return entry
	}
	return strings.TrimRight(string(b), "\n")
}

// entryList reports whether an item's value is a list of entries: a folded list, several chips or
// several lines. Each entry of it has its own icon, and copies in YAML syntax.
func entryList(chips, lines int, folded bool) bool { return folded || chips > 1 || lines > 1 }
