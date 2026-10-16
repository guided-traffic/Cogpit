package fyneui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The metrics' source line under the drawer's and the Overview's charts (G12): "Displaying metrics
// from Prometheus (monitoring/prometheus-k8s) and Kubernetes Metrics Server", or the reason there
// are none. Lens's line names the provider's namespace and service as links (captured on a node's
// drawer: "Displaying metrics from Prometheus: <namespace>/<service>:<port>"); Cogpit's does since
// the night of 2026-10-16/17: the namespace opens the Namespace, the service the Service, through
// ui.navigate as every link of a drawer does (followLink), in the drawer's or the Overview's
// cluster. The line is cut to its width with an ellipsis, as a truncating textLine is.

// sourceLine is one line of text in pieces, some of them links.
type sourceLine struct {
	widget.BaseWidget
	text  string // the whole line, as a textLine's (tests read it)
	color fyne.ThemeColorName
	size  fyne.ThemeSizeName
	segs  []lineSeg
	hover int // the link under the pointer; -1 for none
}

// lineSeg is a piece of a sourceLine: plain text, or a link when onTap is set.
type lineSeg struct {
	text  string
	onTap func()
}

var (
	_ fyne.Tappable      = (*sourceLine)(nil)
	_ desktop.Hoverable  = (*sourceLine)(nil)
	_ desktop.Cursorable = (*sourceLine)(nil)
)

func newSourceLine(text string, c fyne.ThemeColorName, size fyne.ThemeSizeName) *sourceLine {
	l := &sourceLine{color: c, size: size, hover: -1}
	l.ExtendBaseWidget(l)
	l.SetText(text)
	return l
}

// SetText shows plain text, no links.
func (l *sourceLine) SetText(s string) { l.setSegs([]lineSeg{{text: s}}) }

// SetColor changes the plain text's colour token; the links keep the link colour.
func (l *sourceLine) SetColor(c fyne.ThemeColorName) {
	if c != l.color {
		l.color = c
		l.Refresh()
	}
}

// setSegs shows the pieces; the same pieces cost nothing (a poll redraws the line every 15 s).
func (l *sourceLine) setSegs(segs []lineSeg) {
	var b strings.Builder
	same := len(segs) == len(l.segs)
	for i, s := range segs {
		b.WriteString(s.text)
		if same && (s.text != l.segs[i].text || (s.onTap == nil) != (l.segs[i].onTap == nil)) {
			same = false
		}
	}
	l.segs = segs // new taps (another object's drawer) even when the text is the same
	if same {
		return
	}
	l.text, l.hover = b.String(), -1
	l.Refresh()
}

// links are the line's links by their text.
func (l *sourceLine) links() map[string]func() {
	out := map[string]func(){}
	for _, s := range l.segs {
		if s.onTap != nil {
			out[s.text] = s.onTap
		}
	}
	return out
}

// sourcePrefix begins the source line before the providers' names.
const sourcePrefix = "Displaying metrics from "

// sourceSegments splits "Displaying metrics from <source>" where the source names the Prometheus
// in use as "namespace/service": the namespace and the service become links that open the
// Namespace and the Service in context's cluster. Without such a provider, or a source that does
// not name it, the line is plain.
func (sh *shell) sourceSegments(source string, providers []metrics.ProviderStatus, context string) []lineSeg {
	text := sourcePrefix + source
	var ns, svc string
	for _, ps := range providers {
		if ps.Provider == metrics.PrometheusID && ps.Namespace != "" && ps.ServiceName != "" {
			ns, svc = ps.Namespace, ps.ServiceName
		}
	}
	i := -1
	if ns != "" {
		i = strings.Index(text, "("+ns+"/"+svc+")")
	}
	if i < 0 {
		return []lineSeg{{text: text}}
	}
	open := func(ref model.ObjectRef) func() {
		return func() { sh.followLink(ref, context, false) }
	}
	i++ // after "("
	return []lineSeg{
		{text: text[:i]},
		{text: ns, onTap: open(model.ObjectRef{Kind: kindNamespaces, Name: ns})},
		{text: "/"},
		{text: svc, onTap: open(model.ObjectRef{Kind: kindServices, Namespace: ns, Name: svc})},
		{text: text[i+len(ns)+1+len(svc):]},
	}
}

// spans are where the pieces stand in a line as wide as width: each piece's start and end, the
// one that does not fit cut at the line's end and the ones after it empty, as Layout draws them.
func (l *sourceLine) spans(width float32) [][2]float32 {
	size := l.Theme().Size(l.size)
	out := make([][2]float32, len(l.segs))
	x := float32(0)
	for i, s := range l.segs {
		w := min(fyne.MeasureText(s.text, size, fyne.TextStyle{}).Width, max(width-x, 0))
		out[i] = [2]float32{x, x + w}
		x += w
	}
	return out
}

// segAt is the index of the link at x; -1 for none.
func (l *sourceLine) segAt(x float32) int {
	for i, sp := range l.spans(l.Size().Width) {
		if l.segs[i].onTap != nil && x >= sp[0] && x < sp[1] {
			return i
		}
	}
	return -1
}

// Tapped implements fyne.Tappable: a link runs.
func (l *sourceLine) Tapped(ev *fyne.PointEvent) {
	if i := l.segAt(ev.Position.X); i >= 0 {
		l.segs[i].onTap()
	}
}

// Cursor implements desktop.Cursorable: a hand over a link.
func (l *sourceLine) Cursor() desktop.Cursor {
	if l.hover >= 0 {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// MouseIn implements desktop.Hoverable.
func (l *sourceLine) MouseIn(ev *desktop.MouseEvent) { l.MouseMoved(ev) }

// MouseMoved implements desktop.Hoverable: the link under the pointer is underlined.
func (l *sourceLine) MouseMoved(ev *desktop.MouseEvent) {
	if i := l.segAt(ev.Position.X); i != l.hover {
		l.hover = i
		l.Refresh()
	}
}

// MouseOut implements desktop.Hoverable.
func (l *sourceLine) MouseOut() {
	if l.hover >= 0 {
		l.hover = -1
		l.Refresh()
	}
}

// CreateRenderer implements fyne.Widget.
func (l *sourceLine) CreateRenderer() fyne.WidgetRenderer {
	r := &sourceLineRenderer{l: l, line: canvas.NewRectangle(color.Transparent)}
	r.Refresh()
	return r
}

type sourceLineRenderer struct {
	l     *sourceLine
	texts []*canvas.Text
	line  *canvas.Rectangle // the hovered link's underline
	objs  []fyne.CanvasObject
}

func (r *sourceLineRenderer) Layout(size fyne.Size) {
	x := float32(0)
	r.line.Hide()
	for i, t := range r.texts {
		want := r.l.segs[i].text
		room := size.Width - x
		w := fyne.MeasureText(want, t.TextSize, t.TextStyle).Width
		if room <= 0 {
			t.Hide()
			continue
		}
		if w > room {
			want = ellipsize(want, room, t.TextSize, t.TextStyle)
		}
		if t.Text != want {
			t.Text = want
			t.Refresh()
		}
		t.Show()
		ms := t.MinSize()
		y := (size.Height - ms.Height) / 2
		t.Move(fyne.NewPos(x, y))
		t.Resize(ms)
		if i == r.l.hover {
			r.line.Move(fyne.NewPos(x, y+ms.Height-2))
			r.line.Resize(fyne.NewSize(ms.Width, 1))
			r.line.Show()
		}
		x += ms.Width
		if w > room {
			x = size.Width // cut here: nothing after it
		}
	}
}

func (r *sourceLineRenderer) MinSize() fyne.Size {
	// Like a truncating textLine: any width will do, the text is shortened to fit.
	return fyne.NewSize(0, fyne.MeasureText("Mg", r.l.Theme().Size(r.l.size), fyne.TextStyle{}).Height)
}

func (r *sourceLineRenderer) Refresh() {
	th := r.l.Theme()
	for len(r.texts) < len(r.l.segs) {
		r.texts = append(r.texts, canvas.NewText("", color.Transparent))
	}
	r.texts = r.texts[:len(r.l.segs)]
	r.objs = r.objs[:0]
	for i, t := range r.texts {
		s := r.l.segs[i]
		t.Text, t.TextSize = s.text, th.Size(r.l.size)
		t.Color = themeColor(th, r.l.color)
		if s.onTap != nil {
			t.Color = themeColor(th, colorLink)
		}
		r.objs = append(r.objs, t)
	}
	r.line.FillColor = themeColor(th, colorLink)
	r.objs = append(r.objs, r.line)
	r.Layout(r.l.Size())
	for _, t := range r.texts {
		t.Refresh()
	}
	r.line.Refresh()
}

func (r *sourceLineRenderer) Objects() []fyne.CanvasObject { return r.objs }
func (r *sourceLineRenderer) Destroy()                     {}
