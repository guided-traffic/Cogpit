package fyneui

import (
	"encoding/json"
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// connectionColor maps a connection state to a status colour token.
func connectionColor(c engine.Connection) fyne.ThemeColorName {
	switch c.State {
	case engine.Connected:
		return statusColorName(model.StatusOK)
	case engine.Connecting:
		return statusColorName(model.StatusPending)
	case engine.ConnectError:
		return statusColorName(model.StatusError)
	}
	return statusColorName(model.StatusUnknown)
}

// connectionText describes a connection state in a word or two.
func connectionText(c engine.Connection) string {
	switch c.State {
	case engine.Connected:
		return "Connected"
	case engine.Connecting:
		return "Connecting…"
	case engine.ConnectError:
		return "Connection failed"
	}
	return "Not connected"
}

// topBar is the window's top bar: the Cogpit wordmark, the active context with its connection
// state, the context picker button and, when an MCP command is configured, an MCP button.
type topBar struct {
	sh      *shell
	dot     *dot
	context *textLine
	state   *textLine
	switchB *widget.Button
	mcp     *widget.Button
	root    fyne.CanvasObject
}

func newTopBar(sh *shell) *topBar {
	t := &topBar{sh: sh}
	t.dot = newDot(statusColorName(model.StatusUnknown))
	t.context = newTextLine("", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	t.state = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	t.switchB = widget.NewButtonWithIcon("Switch context", theme.MenuDropDownIcon(), func() { sh.picker.show() })
	t.switchB.IconPlacement = widget.ButtonIconTrailingText
	t.switchB.Importance = widget.LowImportance

	wordmark := newTextLine("Cogpit", theme.ColorNameForeground, theme.SizeNameSubHeadingText, fyne.TextStyle{Bold: true})
	left := container.NewHBox(newLogo(), wordmark, newHGap(theme.SizeNameInnerPadding), newDivider(true),
		newHGap(theme.SizeNameInnerPadding), t.dot, t.context, t.state, t.switchB)
	right := container.NewHBox()
	if sh.opts.MCPCommand != "" {
		t.mcp = widget.NewButtonWithIcon("MCP", theme.ContentCopyIcon(), sh.copyMCPCommand)
		t.mcp.Importance = widget.LowImportance
		right.Add(t.mcp)
	}
	t.root = sized("", sizeBarHeight, newSurface(colorTopBar),
		inset(theme.SizeNameInnerPadding, "", container.NewBorder(nil, nil, left, right)))
	return t
}

func (t *topBar) update() {
	ui := t.sh.ui
	t.dot.SetColor(connectionColor(ui.Connection))
	if ui.Context == "" {
		t.context.SetText("No cluster")
	} else {
		t.context.SetText(ui.Context)
	}
	t.state.SetText(connectionText(ui.Connection))
}

// newLogo draws Cogpit's mark: a ring with a filled centre, like a gauge in a cockpit. Drawn from
// canvas primitives in the accent colour; no image asset.
func newLogo() fyne.CanvasObject { return &logo{} }

type logo struct{ widget.BaseWidget }

// CreateRenderer implements fyne.Widget.
func (l *logo) CreateRenderer() fyne.WidgetRenderer {
	l.ExtendBaseWidget(l)
	r := &logoRenderer{l: l, ring: canvas.NewCircle(nil), hub: canvas.NewCircle(nil)}
	r.Refresh()
	return r
}

type logoRenderer struct {
	l         *logo
	ring, hub *canvas.Circle
}

func (r *logoRenderer) MinSize() fyne.Size {
	s := r.l.Theme().Size(theme.SizeNameInlineIcon) + 2
	return fyne.NewSize(s, s)
}

func (r *logoRenderer) Layout(size fyne.Size) {
	s := r.MinSize().Width
	off := fyne.NewPos((size.Width-s)/2, (size.Height-s)/2)
	r.ring.Move(off)
	r.ring.Resize(fyne.NewSize(s, s))
	h := s * 0.36
	r.hub.Move(off.AddXY((s-h)/2, (s-h)/2))
	r.hub.Resize(fyne.NewSize(h, h))
}

func (r *logoRenderer) Refresh() {
	th := r.l.Theme()
	accent := themeColor(th, colorAccent)
	r.ring.FillColor = nil
	r.ring.StrokeColor = accent
	r.ring.StrokeWidth = r.MinSize().Width * 0.14 // proportional to the theme's icon size
	r.hub.FillColor = accent
	r.Layout(r.l.Size())
	r.ring.Refresh()
	r.hub.Refresh()
}

func (r *logoRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.ring, r.hub} }
func (r *logoRenderer) Destroy()                     {}

// diagView is what the status bar shows from engine.Diagnostics, prepared off the main
// goroutine.
type diagView struct {
	total, list, watch int64
	informers          int
	snapshots          uint64
	demo               bool
	demoRate           float64
	demoObjects        int64
}

func newDiagView(d engine.Diagnostics) diagView {
	v := diagView{
		total: d.Requests.Total, informers: len(d.Informers), snapshots: d.Snapshots, demo: d.Demo,
	}
	if d.Requests.ByVerb != nil {
		v.list, v.watch = d.Requests.ByVerb["list"], d.Requests.ByVerb["watch"]
	}
	if d.DemoStats != nil {
		// DemoStats crosses the seam as a JSON-shaped value; the UI may not import package demo.
		var s struct {
			Objects      int64   `json:"objects"`
			EventsPerSec float64 `json:"eventsPerSec"`
		}
		if b, err := json.Marshal(d.DemoStats); err == nil && json.Unmarshal(b, &s) == nil {
			v.demo, v.demoRate, v.demoObjects = true, s.EventsPerSec, s.Objects
		}
	}
	return v
}

// statusBar is the bottom line: context and connection, API requests, informers, snapshots,
// the demo cluster's event rate and the MCP indicator.
type statusBar struct {
	sh        *shell
	dot       *dot
	context   *textLine
	requests  *textLine
	informers *textLine
	snapshots *textLine
	demo      *textLine
	demoSep   fyne.CanvasObject
	root      fyne.CanvasObject
}

func newStatusBar(sh *shell) *statusBar {
	s := &statusBar{sh: sh}
	caption := func() *textLine {
		return newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	}
	s.dot = newDot(statusColorName(model.StatusUnknown))
	s.context, s.requests, s.informers, s.snapshots, s.demo = caption(), caption(), caption(), caption(), caption()
	s.demoSep = newDivider(true)
	left := container.NewHBox(s.dot, s.context, newDivider(true), s.requests, newDivider(true),
		s.informers, newDivider(true), s.snapshots, s.demoSep, s.demo)
	right := container.NewHBox()
	if sh.opts.MCPCommand != "" {
		right.Add(newPill("MCP", colorAccent, true))
	}
	s.root = sized("", sizeStatusHeight, newSurface(colorStatusBar),
		inset(theme.SizeNameInnerPadding, "", container.NewBorder(nil, nil, left, right)))
	return s
}

func (s *statusBar) update() {
	ui, d := s.sh.ui, s.sh.diag
	s.dot.SetColor(connectionColor(ui.Connection))
	ctx := ui.Context
	if ctx == "" {
		ctx = "no cluster"
	}
	s.context.SetText(ctx + " · " + connectionText(ui.Connection))
	s.requests.SetText(fmt.Sprintf("API %s req · list %s · watch %s",
		formatCount(d.total), formatCount(d.list), formatCount(d.watch)))
	s.informers.SetText(strconv.Itoa(d.informers) + " informers")
	s.snapshots.SetText(formatCount(int64(d.snapshots)) + " snapshots")
	if d.demo {
		s.demo.SetText(fmt.Sprintf("demo %s objects · %s events/s",
			formatCount(d.demoObjects), formatCount(int64(d.demoRate+0.5))))
		s.demo.Show()
		s.demoSep.Show()
	} else {
		s.demo.Hide()
		s.demoSep.Hide()
	}
}
