package fyneui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"math"
	"net/url"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	logoart "github.com/guided-traffic/cogpit/internal/ui/logo"
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

// topBar is the window's top bar: the Cogpit wordmark, the history arrows, the active context with
// its connection state, the context picker button, when an MCP command is configured an MCP button,
// and the notifications' bell. On macOS
// it is also the window's title bar (titlebar.go): room for the window buttons before the logo,
// and a drag area under its contents.
type topBar struct {
	sh          *shell
	dot         *dot
	context     *textLine
	file        *textLine // the kubeconfig the context comes from, muted
	state       *textLine
	switchB     *widget.Button
	mcp         *widget.Button
	lead        *canvas.Rectangle // the window buttons' room; hidden unless Cogpit draws the title bar
	history     *titleHistory     // the history arrows and the bar's context menu (titlehistory.go)
	notes       *notificationsUI  // the bell and its panel (notifications.go)
	left        *fyne.Container
	titleHeight float32 // the title bar's height the bar covers, in Fyne units; 0 without
	titleLead   float32 // the window buttons' room last made (lead's width); 0 without
	titleSet    bool    // setTitlebar ran: titleLead and titleHeight are what it applied
	root        fyne.CanvasObject
}

func newTopBar(sh *shell) *topBar {
	t := &topBar{sh: sh}
	t.dot = newDot(statusColorName(model.StatusUnknown))
	t.context = newTextLine("", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	t.file = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	t.file.Hide()
	t.state = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	t.switchB = widget.NewButtonWithIcon("Switch context", theme.MenuDropDownIcon(), func() { sh.picker.show() })
	t.switchB.IconPlacement = widget.ButtonIconTrailingText
	t.switchB.Importance = widget.LowImportance

	wordmark := newTextLine("Cogpit", theme.ColorNameForeground, theme.SizeNameSubHeadingText, fyne.TextStyle{Bold: true})
	t.lead = canvas.NewRectangle(color.Transparent)
	t.lead.Hide()
	t.history = newTitleHistory(sh)
	t.notes = newNotificationsUI(sh)
	t.left = container.NewHBox(t.lead, newLogo(), wordmark, t.history.root, newHGap(theme.SizeNameInnerPadding), newDivider(true),
		newHGap(theme.SizeNameInnerPadding), t.dot, t.context, t.file, t.state, t.switchB)
	right := container.NewHBox()
	if sh.hasMCP() {
		t.mcp = widget.NewButtonWithIcon("MCP", theme.ContentCopyIcon(), sh.copyMCPCommand)
		t.mcp.Importance = widget.LowImportance
		if !sh.mcp.on {
			t.mcp.Disable()
		}
		right.Add(t.mcp)
	}
	right.Add(t.notes.bell)
	drag := newBarDragArea(func() { sh.titlebar.mouseDown() })
	drag.secondary = t.history.showMenu
	t.root = box(&topBarLayout{t: t}, newSurface(colorTopBar), drag,
		inset(theme.SizeNameInnerPadding, "", container.NewBorder(nil, nil, t.left, right)))
	return t
}

func (t *topBar) update() {
	ui := t.sh.ui
	t.dot.SetColor(connectionColor(ui.Connection))
	name, file := splitContextID(ui.Context)
	if name == "" {
		name = "No cluster"
	}
	t.context.SetText(name)
	t.file.SetText(file)
	if file == "" {
		t.file.Hide()
	} else {
		t.file.Show()
	}
	t.state.SetText(connectionText(ui.Connection))
}

// splitContextID splits a source context's id, "<context> [<file>]" (engine.SourceContextID), into
// the context's name and the file it comes from, so that the bars show the name and the file apart;
// an id without a file (demo) is a name alone.
func splitContextID(id string) (name, file string) {
	if i := strings.LastIndex(id, " ["); i > 0 && strings.HasSuffix(id, "]") {
		return id[:i], id[i+2 : len(id)-1]
	}
	return id, ""
}

// newLogo draws Cogpit's mark, the "C helm" of D-24: an open C around a ship's wheel, in the heavy
// line weights for a bar (internal/ui/logo, the geometry of docs/developer/assets/cogpit-mark-*.svg).
// The inks are theme tokens, so the mark follows the theme; the picture is drawn at the pixel size
// the canvas shows it at.
func newLogo() fyne.CanvasObject {
	l := &logo{}
	l.ExtendBaseWidget(l)
	return l
}

type logo struct {
	widget.BaseWidget
	px         int         // the size the picture was drawn at, in pixels
	frame, acc color.Color // and with these inks
}

// CreateRenderer implements fyne.Widget.
func (l *logo) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromImage(nil)
	img.FillMode, img.ScaleMode = canvas.ImageFillContain, canvas.ImageScaleSmooth
	r := &logoRenderer{l: l, img: img}
	r.Refresh()
	return r
}

type logoRenderer struct {
	l   *logo
	img *canvas.Image
}

func (r *logoRenderer) MinSize() fyne.Size {
	s := r.l.Theme().Size(sizeLogo)
	return fyne.NewSize(s, s)
}

func (r *logoRenderer) Layout(size fyne.Size) {
	s := r.MinSize().Width
	r.img.Move(fyne.NewPos((size.Width-s)/2, (size.Height-s)/2))
	r.img.Resize(fyne.NewSize(s, s))
}

func (r *logoRenderer) Refresh() {
	l, th := r.l, r.l.Theme()
	frame, accent := themeColor(th, colorLogo), themeColor(th, colorLogoAccent)
	scale := float32(1)
	if c := fyne.CurrentApp().Driver().CanvasForObject(l); c != nil {
		scale = c.Scale()
	}
	px := int(math.Ceil(float64(r.MinSize().Width * scale)))
	if px != l.px || frame != l.frame || accent != l.acc || r.img.Image == nil {
		img := image.NewNRGBA(image.Rect(0, 0, px, px))
		logoart.Draw(img, logoart.Mark(), logoart.Colors{Frame: frame, Accent: accent}, float64(px)/logoart.Box, 0, 0)
		r.img.Image = img
		l.px, l.frame, l.acc = px, frame, accent
	}
	r.Layout(l.Size())
	r.img.Refresh()
}

func (r *logoRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.img} }
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
	link      *linkText // the link to the owner (D-23), left of the MCP item
	mcp       *mcpItem  // the MCP indicator and its switch, at the right end; nil without an MCP server
	layout    *statusLayout
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
		return newTextLine("", colorMuted, sizeStatusText, fyne.TextStyle{})
	}
	s.dot = newDot(statusColorName(model.StatusUnknown))
	s.context, s.requests, s.informers, s.snapshots, s.demo = caption(), caption(), caption(), caption(), caption()
	s.demoSep = newDivider(true)
	left := container.NewHBox(s.dot, s.context, newDivider(true), s.requests, newDivider(true),
		s.informers, newDivider(true), s.snapshots, s.demoSep, s.demo)
	right := container.NewHBox()
	if sh.hasMCP() {
		s.mcp = newMCPItem(sh, sh.mcp.on)
		right.Add(s.mcp)
	}
	s.link = newLinkText(ownerLinkText, func() { sh.openOwnerLink() })
	s.link.size = sizeStatusText
	sep := newDivider(true)
	s.layout = &statusLayout{link: s.link, sep: sep, left: left, right: right}
	s.root = sized("", sizeStatusHeight, newSurface(colorStatusBar),
		inset(theme.SizeNameInnerPadding, "", box(s.layout, s.link, sep, left, right)))
	return s
}

// The status bar's link to the owner (D-23): shown always, a GUI link and not an engine command, so
// that no agent can open the page. Only a click opens it; nothing is sent before.
const (
	ownerLinkText = "Need a Kubernetes Expert? Visit: hans-fischer.com"
	ownerLinkURL  = "https://hans-fischer.com"
)

// openOwnerLink opens the owner's page in the user's browser.
func (sh *shell) openOwnerLink() {
	u, err := url.Parse(ownerLinkURL)
	if err == nil {
		err = openBrowser(sh.app, u)
	}
	if err != nil {
		sh.toast("error", "cannot open "+ownerLinkURL+": "+err.Error())
	}
}

// statusLayout lays the status bar out: the status items at the left end, the MCP item at the right
// end, and the link to the owner and its divider directly left of the MCP item. The items keep
// their minimum widths; the link takes what is left between them, is shortened with an ellipsis
// when that is less than its text, and is hidden, with its divider, when it is less than a few
// characters. The bar's minimum width is the items' alone, so the link never costs them room (D-23).
type statusLayout struct {
	link, sep   fyne.CanvasObject
	left, right fyne.CanvasObject
}

// linkMinChars is the shortest the link is shown: its first characters and an ellipsis.
const linkMinChars = "Need a…"

func (l *statusLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	lm, rm := l.left.MinSize(), l.right.MinSize()
	return fyne.NewSize(lm.Width+rm.Width, max(lm.Height, rm.Height, l.link.MinSize().Height))
}

func (l *statusLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	th := fyne.CurrentApp().Settings().Theme()
	lm, rm := l.left.MinSize(), l.right.MinSize()
	gap := th.Size(theme.SizeNameInnerPadding)
	textSize := th.Size(sizeStatusText)
	if lt, ok := l.link.(*linkText); ok {
		textSize = th.Size(lt.size)
	}
	full := fyne.MeasureText(ownerLinkText, textSize, fyne.TextStyle{}).Width
	least := fyne.MeasureText(linkMinChars, textSize, fyne.TextStyle{}).Width
	// What is left between the items for the link; its divider and the gaps around it come off. The
	// divider sits between the link and the MCP item, and a gap keeps the link off the left items.
	// Without an MCP item (--no-mcp) the link ends the bar and has no divider.
	withSep := rm.Width > 0
	room := size.Width - lm.Width - rm.Width - 3*gap - 1
	if !withSep {
		room += gap + 1
	}
	if room >= least {
		w := fyne.Min(room, full+1)
		l.show(true)
		if withSep {
			sepX := size.Width - rm.Width - gap - 1
			l.link.Move(fyne.NewPos(sepX-gap-w, 0))
			l.sep.Move(fyne.NewPos(sepX, 0))
			l.sep.Resize(fyne.NewSize(1, size.Height))
		} else {
			l.link.Move(fyne.NewPos(size.Width-w, 0))
			l.sep.Hide()
		}
		l.link.Resize(fyne.NewSize(w, size.Height))
	} else {
		l.show(false)
	}
	l.left.Move(fyne.NewPos(0, 0))
	l.left.Resize(fyne.NewSize(lm.Width, size.Height))
	l.right.Resize(fyne.NewSize(rm.Width, size.Height))
	l.right.Move(fyne.NewPos(size.Width-rm.Width, 0))
}

func (l *statusLayout) show(on bool) {
	for _, o := range []fyne.CanvasObject{l.link, l.sep} {
		if o.Visible() != on {
			if on {
				o.Show()
			} else {
				o.Hide()
			}
		}
	}
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
