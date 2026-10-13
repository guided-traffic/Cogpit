package fyneui

import (
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The drawer's copy icon right after every kind of value (D-25, the owner in a real window,
// 2026-10-13: "the copy button at Not After has wandered to the far right again"): a padding after
// a pill, a line, a link, an event's message; at the value's end when the text leaves no room,
// the text then shortened with "…" before it; in an action row before the buttons.

// iconAt returns the copy icon's left edge and width, in window coordinates.
func iconAt(t *testing.T, d *drawer) (float32, float32) {
	t.Helper()
	if !d.copyBtn.Visible() {
		t.Fatal("no copy icon")
	}
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(d.copyBtn).X, d.copyBtn.Size().Width
}

// near fails unless got is want within half a point.
func near(t *testing.T, what string, got, want float32) {
	t.Helper()
	if d := got - want; d < -0.5 || d > 0.5 {
		t.Errorf("%s: %v, want %v", what, got, want)
	}
}

func padding() float32 { return fyne.CurrentApp().Settings().Theme().Size(theme.SizeNamePadding) }

// hostRight is the right edge of the drawer's details, in window coordinates.
func hostRight(d *drawer) float32 {
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(d.host).X + d.host.Size().Width
}

// TestDrawerCopyIconAfterThePill: the owner's case, a TLS Secret's Not After drawn as a warning
// pill: the icon sits a padding after the pill, far from the drawer's right edge.
func TestDrawerCopyIconAfterThePill(t *testing.T) {
	f, ref, _ := certsSecretsFake(t)
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && sh.drawer.last.Ref == ref })
	settle(w)
	d := sh.drawer
	hw := chipWrapWith(t, d, func(s string) bool { return strings.HasSuffix(s, "(in 20 days)") })
	p := findAll[*pill](hw.content)[0]
	hw.MouseIn(mouse())
	x, side := iconAt(t, d)
	drv := fyne.CurrentApp().Driver()
	pillEnd := drv.AbsolutePositionForObject(p).X + p.Size().Width
	near(t, "the icon after the Not After pill", x, pillEnd+padding())
	if right := hostRight(d); right-(x+side) < 50 {
		t.Errorf("the icon ends %v before the drawer's right edge: it wandered to the far right", right-(x+side))
	}
	if !hw.hook.chip {
		t.Error("the pill is no chip")
	}
	// The pointer's way to the icon stays inside the pill's value, which spans the row.
	hx := drv.AbsolutePositionForObject(hw).X
	if x+side > hx+hw.Size().Width {
		t.Errorf("the icon ends at %v, beyond the value's hover area %v", x+side, hx+hw.Size().Width)
	}
	hw.MouseOut()
}

// TestDrawerCopyIconAfterTheText: a short line right after its text, the text whole; a long one
// (a SHA-256 fingerprint) at the value's end with its text shortened before the icon, kept while
// the pointer is on the icon and given back after; a link after its text.
func TestDrawerCopyIconAfterTheText(t *testing.T) {
	f, ref, _ := certsSecretsFake(t)
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && sh.drawer.last.Ref == ref })
	settle(w)
	d := sh.drawer
	drv := fyne.CurrentApp().Driver()

	serial := selLineWith(t, d, "4F:2A:9C:11:E0")
	serial.MouseIn(mouse())
	x, _ := iconAt(t, d)
	text := fyne.MeasureText(serial.line.text, serial.Theme().Size(serial.line.size), serial.line.style).Width
	near(t, "the icon after the serial number", x, drv.AbsolutePositionForObject(serial).X+text+padding())
	if shown, cut := serial.shown(); cut || shown != serial.line.text {
		t.Errorf("the serial number shows %q beside its icon", shown)
	}
	serial.MouseOut()

	var long *selLine
	for _, l := range findAll[*selLine](d.details) {
		if l.hook != nil && strings.Count(l.line.text, ":") > 20 {
			long = l
			break
		}
	}
	if long == nil {
		t.Fatal("no fingerprint line")
	}
	whole, _ := long.shown()
	long.MouseIn(mouse())
	x, side := iconAt(t, d)
	lx := drv.AbsolutePositionForObject(long).X
	near(t, "the icon of a long value at its end", x+side, lx+long.Size().Width)
	shown, cut := long.shown()
	if !cut || !strings.HasSuffix(shown, "…") || len(shown) >= len(whole) {
		t.Errorf("beside the icon the fingerprint shows %q, without it %q", shown, whole)
	}
	if end := lx + fyne.MeasureText(shown, long.Theme().Size(long.line.size), long.line.style).Width; end > x-padding()+0.5 {
		t.Errorf("the shortened text ends at %v, under the icon at %v", end, x)
	}
	// Onto the icon: the value's MouseOut, the icon's MouseIn; the text stays short.
	long.MouseOut()
	d.copyBtn.MouseIn(mouse())
	if s, _ := long.shown(); s != shown {
		t.Errorf("with the pointer on the icon the fingerprint shows %q, want %q", s, shown)
	}
	d.copyBtn.MouseOut()
	if s, _ := long.shown(); s != whole || long.room != 0 {
		t.Errorf("without the icon the fingerprint shows %q (room %v), want %q", s, long.room, whole)
	}
	sh.app.Clipboard().SetContent("")
	long.MouseIn(mouse())
	d.copyBtn.Tapped(nil)
	if got := sh.app.Clipboard().Content(); got != long.line.text {
		t.Errorf("the icon copied %q, want the whole fingerprint", got)
	}
	long.MouseOut()

	links := 0
	for _, l := range findAll[*linkText](d.details) {
		if l.hook == nil {
			continue
		}
		links++
		l.MouseIn(mouse())
		x, _ := iconAt(t, d)
		text := fyne.MeasureText(l.text, l.Theme().Size(l.size), fyne.TextStyle{}).Width
		near(t, "the icon after the link "+l.text, x, drv.AbsolutePositionForObject(l).X+text+padding())
		l.MouseOut()
	}
	if links == 0 {
		t.Error("no link value checked")
	}
}

// TestDrawerCopyIconAfterTheMessage: an event's short message has the icon right after its text,
// not at the drawer's right edge; a message that wraps keeps it at the top right, inside.
func TestDrawerCopyIconAfterTheMessage(t *testing.T) {
	f := kindsFake(t, "pods")
	p := richPod(t)
	ref := model.ObjectRef{Context: "demo", Kind: "pods", Namespace: p.ns, Name: p.name}
	m := *f.details[ref.String()]
	m.Events = append([]engine.EventItem{{Type: "Normal", Reason: "Pulled", Message: "Image pulled", Count: 1}}, m.Events...)
	f.details[ref.String()] = &m
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && len(sh.drawer.details.Objects) > 2 && sh.diag.total > 0 })
	settle(w)
	d := sh.drawer
	drv := fyne.CurrentApp().Driver()
	short, wrapped := 0, 0
	for _, hw := range findAll[*hoverWrap](d.details) {
		l, ok := hw.content.(*widget.Label)
		if !ok {
			continue
		}
		th := l.Theme()
		inner := th.Size(theme.SizeNameInnerPadding)
		text := fyne.MeasureText(l.Text, th.Size(theme.SizeNameText), l.TextStyle).Width
		hw.MouseIn(mouse())
		x, side := iconAt(t, d)
		hx := drv.AbsolutePositionForObject(hw).X
		if inner+text+padding()+side <= hw.Size().Width {
			short++
			near(t, "the icon after the message "+l.Text, x, hx+inner+text+padding())
		} else {
			wrapped++
			near(t, "the icon of the wrapped message "+l.Text[:20], x+side, hostRight(d))
		}
		hw.MouseOut()
	}
	if short == 0 || wrapped == 0 {
		t.Fatalf("%d short and %d wrapped event messages checked, want both", short, wrapped)
	}
}

// TestDrawerCopyIconInAnActionRow: a container port is a link with Forward… beside it; the port
// owns the room for its icon, which sits after the port's text and before the button, over
// neither.
func TestDrawerCopyIconInAnActionRow(t *testing.T) {
	f := forwardsFake(t, "pods")
	sh, _, _ := openPodDrawer(t, f, healthyPodRef(t), ThemeDark)
	d := sh.drawer
	drv := fyne.CurrentApp().Driver()
	button := forwardButtons(sh)[0]
	var port *linkText
	bx := drv.AbsolutePositionForObject(button)
	for _, l := range findAll[*linkText](d.details) {
		ly := drv.AbsolutePositionForObject(l).Y
		if l.hook != nil && ly <= bx.Y+button.Size().Height && ly+l.Size().Height >= bx.Y {
			port = l
		}
	}
	if port == nil {
		t.Fatal("no port link on the Forward… button's row")
	}
	port.MouseIn(mouse())
	x, side := iconAt(t, d)
	text := fyne.MeasureText(port.text, port.Theme().Size(port.size), fyne.TextStyle{}).Width
	near(t, "the icon after the port", x, drv.AbsolutePositionForObject(port).X+text+padding())
	if x+side > bx.X {
		t.Errorf("the icon ends at %v, over the Forward… button at %v", x+side, bx.X)
	}
	if port.room != 0 {
		t.Errorf("the port %q is shortened for its icon (room %v)", port.text, port.room)
	}
	port.MouseOut()
}

// renderDrawerCopyValue shows the TLS Secret's certificates with the pointer on the leaf's Not
// After pill, the icon right after it; or (long) on its SHA-256 fingerprint, the icon at the
// value's end and the fingerprint shortened with "…" before it.
func renderDrawerCopyValue(t *testing.T, mode ThemeMode, long bool, name string) {
	f, ref, _ := certsSecretsFake(t)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the drawer", func() bool {
		return len(sh.kinds) > 0 && sh.view.snap != nil && sh.drawer.last != nil && sh.drawer.last.Ref == ref && sh.diag.total > 0
	})
	settle(w)
	scrollDrawerTo(t, sh, "Certificates · tls.crt")
	settle(w)
	if long {
		for _, l := range findAll[*selLine](sh.drawer.details) {
			if l.hook != nil && strings.Count(l.line.text, ":") > 20 {
				l.MouseIn(mouse())
				break
			}
		}
	} else {
		chipWrapWith(t, sh.drawer, func(s string) bool { return strings.HasSuffix(s, "(in 20 days)") }).MouseIn(mouse())
	}
	if !sh.drawer.copyBtn.Visible() {
		t.Fatal("no copy icon")
	}
	savePNG(t, w, name)
}

func TestRenderDrawerCopyIconAfterTheValue(t *testing.T) {
	renderDrawerCopyValue(t, ThemeDark, false, "drawer_copy_notafter_dark.png")
	renderDrawerCopyValue(t, ThemeLight, false, "drawer_copy_notafter_light.png")
	renderDrawerCopyValue(t, ThemeDark, true, "drawer_copy_long_dark.png")
	renderDrawerCopyValue(t, ThemeLight, true, "drawer_copy_long_light.png")
}
