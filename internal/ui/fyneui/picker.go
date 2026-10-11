package fyneui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// demoLabel is how the picker names the synthetic demo cluster.
const demoLabel = "Demo cluster (synthetic, 10,000 pods)"

// picker is the context picker: a modal listing the kubeconfig contexts (and the demo cluster)
// with a filter, connecting the chosen one through cluster.connect.
type picker struct {
	sh *shell

	popup      *widget.PopUp
	filter     *widget.Entry
	list       *fyne.Container
	status     *textLine
	connectB   *widget.Button
	cancelB    *widget.Button
	contexts   []engine.ContextInfo
	selected   string
	connecting string
	content    fyne.CanvasObject
}

func newPicker(sh *shell) *picker {
	p := &picker{sh: sh}
	title := newTextLine("Connect to a cluster", theme.ColorNameForeground, theme.SizeNameHeadingText, fyne.TextStyle{Bold: true})
	intro := newTextLine("Pick a kubeconfig context. Cogpit browses read-only and asks before it changes anything.",
		colorMuted, theme.SizeNameText, fyne.TextStyle{})
	intro.truncate = true
	p.filter = widget.NewEntry()
	p.filter.SetPlaceHolder("Filter contexts…")
	p.filter.SetIcon(theme.SearchIcon())
	p.filter.OnChanged = func(string) { p.render() }
	p.filter.OnSubmitted = func(string) { p.connect() }
	p.list = container.NewVBox()
	p.status = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	p.status.truncate = true
	p.connectB = widget.NewButtonWithIcon("Connect", theme.LoginIcon(), p.connect)
	p.connectB.Importance = widget.HighImportance
	p.cancelB = widget.NewButton("Not now", p.hide)

	listBg := newSurface(theme.ColorNameInputBackground)
	listBg.radius, listBg.stroke = theme.SizeNameInputRadius, theme.ColorNameInputBorder
	list := container.NewStack(listBg, inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewVScroll(p.list)))
	top := container.NewVBox(title, intro, newSpace(theme.SizeNamePadding), p.filter, newSpace(theme.SizeNamePadding))
	bottom := container.NewBorder(newSpace(theme.SizeNameInnerPadding), nil, nil,
		container.NewHBox(p.cancelB, p.connectB), p.status)
	p.content = sized(sizePickerWidth, sizePickerHeight,
		inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, container.NewBorder(top, bottom, nil, nil, list)))
	return p
}

// show opens the picker and loads the contexts off the main goroutine.
func (p *picker) show() {
	if p.popup == nil {
		p.popup = widget.NewModalPopUp(p.content, p.sh.win.Canvas())
	}
	p.popup.Show()
	p.sh.win.Canvas().Focus(p.filter)
	if p.connecting == "" {
		p.status.SetColor(colorMuted)
		p.status.SetText("")
	}
	go func() {
		defer p.sh.recoverTo("contexts")
		ctxs, err := p.sh.eng.Contexts()
		p.sh.do(func() {
			if err != nil {
				p.status.SetColor(colorStatusError)
				p.status.SetText("Could not read the kubeconfig: " + err.Error())
				return
			}
			p.contexts = ctxs
			if p.selected == "" || !p.has(p.selected) {
				p.selected = p.defaultContext()
			}
			p.render()
		})
	}()
}

func (p *picker) hide() {
	if p.popup != nil {
		p.popup.Hide()
	}
}

func (p *picker) visible() bool { return p.popup != nil && p.popup.Visible() }

// connected closes the picker once the engine reports a connection.
func (p *picker) connected(ctxName string) {
	if p.connecting == "" || p.connecting == ctxName {
		p.connecting = ""
		p.connectB.Enable()
		p.hide()
	}
}

// setError shows why connecting failed.
func (p *picker) setError(name string, err error) {
	p.connecting = ""
	p.connectB.Enable()
	p.status.SetColor(colorStatusError)
	p.status.SetText("Could not connect to " + name + ": " + err.Error())
}

func (p *picker) has(name string) bool {
	for _, c := range p.contexts {
		if c.Name == name {
			return true
		}
	}
	return false
}

// defaultContext preselects the active context, else the kubeconfig's current one, else the
// first.
func (p *picker) defaultContext() string {
	if p.sh.ui.Context != "" && p.has(p.sh.ui.Context) {
		return p.sh.ui.Context
	}
	for _, c := range p.contexts {
		if c.Current {
			return c.Name
		}
	}
	if len(p.contexts) > 0 {
		return p.contexts[0].Name
	}
	return ""
}

// render rebuilds the rows that match the filter.
func (p *picker) render() {
	f := strings.ToLower(strings.TrimSpace(p.filter.Text))
	rows := make([]fyne.CanvasObject, 0, len(p.contexts))
	for _, c := range p.contexts {
		title, sub := contextLabels(c)
		if f != "" && !strings.Contains(strings.ToLower(c.Name+" "+title+" "+sub), f) {
			continue
		}
		name := c.Name
		rows = append(rows, newPickerRow(c, title, sub, name == p.selected,
			func() { p.selected = name; p.render() },
			func() { p.selected = name; p.connect() }))
	}
	if len(rows) == 0 {
		rows = append(rows, inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
			newTextLine("No contexts match.", colorMuted, theme.SizeNameText, fyne.TextStyle{})))
	}
	p.list.Objects = rows
	p.list.Refresh()
}

// connect connects the selected context.
func (p *picker) connect() {
	name := p.selected
	if name == "" || p.connecting != "" {
		return
	}
	p.connecting = name
	p.connectB.Disable()
	p.status.SetColor(colorMuted)
	p.status.SetText("Connecting to " + name + "…")
	p.sh.dispatch("cluster.connect", contextParams{Context: name}, func(_ any, err error) {
		if err != nil {
			p.setError(name, err)
			return
		}
		p.connected(name)
	})
}

// contextLabels returns a context's title and subtitle in the picker.
func contextLabels(c engine.ContextInfo) (title, sub string) {
	if c.Demo {
		return demoLabel, "context " + c.Name + " · generated in memory, no credentials needed"
	}
	var parts []string
	if c.Cluster != "" {
		parts = append(parts, c.Cluster)
	}
	if c.Server != "" {
		parts = append(parts, c.Server)
	}
	if c.Namespace != "" {
		parts = append(parts, "namespace "+c.Namespace)
	}
	return c.Name, strings.Join(parts, " · ")
}

// pickerRow is one context in the picker: title, badges and a subtitle. A tap selects it, a
// double tap connects.
type pickerRow struct {
	widget.BaseWidget
	selected, hovered bool
	onTap, onDouble   func()

	bg     *canvas.Rectangle
	title  *textLine
	sub    *textLine
	badges *fyne.Container
}

func newPickerRow(c engine.ContextInfo, title, sub string, selected bool, onTap, onDouble func()) *pickerRow {
	r := &pickerRow{selected: selected, onTap: onTap, onDouble: onDouble, bg: canvas.NewRectangle(color.Transparent)}
	r.title = newTextLine(title, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	r.sub = newTextLine(sub, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	r.sub.truncate = true
	r.badges = container.NewHBox()
	if c.Current {
		r.badges.Add(newPill("current", colorAccent, true))
	}
	if c.Demo {
		r.badges.Add(newPill("demo", statusColorName(model.StatusSucceeded), true))
	}
	r.ExtendBaseWidget(r)
	return r
}

// Tapped implements fyne.Tappable.
func (r *pickerRow) Tapped(*fyne.PointEvent) { r.onTap() }

// DoubleTapped implements fyne.DoubleTappable.
func (r *pickerRow) DoubleTapped(*fyne.PointEvent) { r.onDouble() }

// MouseIn implements desktop.Hoverable.
func (r *pickerRow) MouseIn(*desktop.MouseEvent) { r.hovered = true; r.Refresh() }

// MouseMoved implements desktop.Hoverable.
func (r *pickerRow) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (r *pickerRow) MouseOut() { r.hovered = false; r.Refresh() }

// CreateRenderer implements fyne.Widget.
func (r *pickerRow) CreateRenderer() fyne.WidgetRenderer {
	content := inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
		container.NewVBox(container.NewHBox(r.title, r.badges), r.sub))
	rr := &pickerRowRenderer{r: r, content: content}
	rr.Refresh()
	return rr
}

type pickerRowRenderer struct {
	r       *pickerRow
	content *fyne.Container
}

func (rr *pickerRowRenderer) Layout(size fyne.Size) {
	rr.r.bg.Resize(size)
	rr.content.Resize(size)
}

func (rr *pickerRowRenderer) MinSize() fyne.Size { return rr.content.MinSize() }

func (rr *pickerRowRenderer) Refresh() {
	th := rr.r.Theme()
	rr.r.bg.CornerRadius = th.Size(theme.SizeNameSelectionRadius)
	switch {
	case rr.r.selected:
		rr.r.bg.FillColor = themeColor(th, theme.ColorNameSelection)
	case rr.r.hovered:
		rr.r.bg.FillColor = themeColor(th, theme.ColorNameHover)
	default:
		rr.r.bg.FillColor = color.Transparent
	}
	rr.r.bg.Refresh()
}

func (rr *pickerRowRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{rr.r.bg, rr.content}
}

func (rr *pickerRowRenderer) Destroy() {}
