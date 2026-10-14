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
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
	"github.com/guided-traffic/cogpit/internal/model"
)

// demoLabel is how the picker names the synthetic demo cluster.
const demoLabel = "Demo cluster (synthetic, 10,000 pods)"

// picker is the context picker: a modal listing the kubeconfig contexts (and the demo cluster)
// with a filter, connecting the chosen one through cluster.connect.
type picker struct {
	sh *shell

	popup      *dismissOverlay // a click outside or Escape closes it (owner, 2026-10-12)
	filter     *keyEntry       // takes Escape itself: a focused field gets every key
	list       *fyne.Container
	status     *textLine
	connectB   *widget.Button
	cancelB    *widget.Button
	addB       *widget.Button // "Add Kubeconfigs": sync a file or a folder (syncpicker.go)
	contexts   []engine.ContextInfo
	selected   string
	connecting string
	content    fyne.CanvasObject
	menuModel  *fyne.Menu // a row's open menu (right-click or ⋮), for tests
}

func newPicker(sh *shell) *picker {
	p := &picker{sh: sh}
	title := newTextLine("Connect to a cluster", theme.ColorNameForeground, theme.SizeNameHeadingText, fyne.TextStyle{Bold: true})
	intro := newTextLine("Pick a kubeconfig context. Cogpit browses read-only and asks before it changes anything.",
		colorMuted, theme.SizeNameText, fyne.TextStyle{})
	intro.truncate = true
	p.filter = newKeyEntry(func(ev *fyne.KeyEvent) bool {
		if ev.Name != fyne.KeyEscape || !p.visible() {
			return false
		}
		p.popup.dismiss() // the window's Escape never sees a key the focused filter gets
		return true
	})
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
	p.addB = widget.NewButtonWithIcon("Add Kubeconfigs", theme.ContentAddIcon(), nil)
	p.addB.OnTapped = sh.kc.addKubeconfigs

	listBg := newSurface(theme.ColorNameInputBackground)
	listBg.radius, listBg.stroke = theme.SizeNameInputRadius, theme.ColorNameInputBorder
	list := container.NewStack(listBg, inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewVScroll(p.list)))
	top := container.NewVBox(title, intro, newSpace(theme.SizeNamePadding), p.filter, newSpace(theme.SizeNamePadding))
	bottom := container.NewBorder(newSpace(theme.SizeNameInnerPadding), nil, p.addB,
		container.NewHBox(p.cancelB, p.connectB), p.status)
	p.content = sized(sizePickerWidth, sizePickerHeight,
		inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, container.NewBorder(top, bottom, nil, nil, list)))
	return p
}

// show opens the picker and loads the contexts off the main goroutine.
func (p *picker) show() {
	if p.sh.palette.visible() { // ⌘P from the command palette: the picker takes its place
		p.sh.palette.dismissThen(nil)
	}
	if p.popup == nil {
		p.popup = newDismissOverlay(p.sh.win.Canvas(), p.content)
	}
	p.popup.Show()
	p.sh.win.Canvas().Focus(p.filter)
	if p.connecting == "" {
		p.status.SetColor(colorMuted)
		p.status.SetText("")
	}
	p.load()
}

// load reads the contexts off the main goroutine and shows them: when the picker opens, and when
// the engine reports that they changed (a kubeconfig source was added, a sync folder changed).
func (p *picker) load() {
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
		if cs := p.sh.ui.ClusterSettings[c.Name]; cs.Name != "" { // the display name, the context below it
			sub = strings.TrimSuffix("context "+title+" · "+sub, " · ")
			title = kubeconfig.DisplayText(cs.Name, maxShownPath)
		}
		if f != "" && !strings.Contains(strings.ToLower(c.Name+" "+title+" "+sub+" "+c.Source), f) {
			continue
		}
		name := c.Name
		row := newPickerRow(c, title, sub, name == p.selected,
			func() { p.selected = name; p.render() },
			func() { p.selected = name; p.connect() },
			func() { p.sh.kc.showConnectLog(name) })
		row.badge.setIdentity(p.sh.identity(name))
		row.onMenu = func(pos fyne.Position) { p.menu(name, pos) }
		if p.sh.ui.Hotbar != nil { // the ⋮ offers the hotbar's item; without a hotbar right-click is enough
			row.addMenuButton()
		}
		rows = append(rows, row)
	}
	if len(rows) == 0 {
		rows = append(rows, inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
			newTextLine("No contexts match.", colorMuted, theme.SizeNameText, fyne.TextStyle{})))
	}
	p.list.Objects = rows
	p.list.Refresh()
}

// connect connects the selected context. A context whose login program (an exec credential
// plugin of a kubeconfig source) the user has not allowed, or that changed since, first shows that
// program (kubeconfigs.go); connecting goes on only once the user allowed it.
func (p *picker) connect() {
	name := p.selected
	if name == "" || p.connecting != "" {
		return
	}
	p.connecting = name
	p.connectB.Disable()
	p.status.SetColor(colorMuted)
	if c, ok := p.context(name); ok && (c.Exec == engine.ExecRequired || c.Exec == engine.ExecChanged) {
		refused := "Not connected: the program that logs in to " + name + " was not allowed."
		switch c.ExecKind {
		case kubeconfig.LoginTokenFile:
			p.status.SetText("Review the token file " + name + " reads to log in…")
		case kubeconfig.LoginInsecure: // A28: asked once before the first connect
			p.status.SetText("Review the connection to " + name + ", whose certificate is not checked…")
			refused = "Not connected: connecting to " + name + " without the certificate check was not allowed."
		default:
			p.status.SetText("Review the program " + name + " runs to log in…")
		}
		p.sh.kc.reviewPlugin(name, func(allowed bool) {
			if !allowed {
				p.connecting = ""
				p.connectB.Enable()
				p.status.SetColor(colorStatusWarning)
				p.status.SetText(refused)
				return
			}
			p.dispatchConnect(name)
		})
		return
	}
	p.dispatchConnect(name)
}

// menu is a context's menu in the picker (right-click or ⋮ on its row): Connect, Settings, and Add
// to Hotbar or Remove from Hotbar for the shown bank (Lens's cluster menu has "Add to Hotbar" and
// "Settings"; the other labels inferred).
func (p *picker) menu(name string, pos fyne.Position) {
	items := []*fyne.MenuItem{fyne.NewMenuItem("Connect", func() { p.selected = name; p.connect() })}
	if item := p.sh.settingsItem(name); item != nil {
		items = append(items, item)
	}
	if item := p.sh.hot.contextItem(name); item != nil {
		items = append(items, item)
	}
	items = append(items, p.sh.clusterItems(name)...) // Disconnect, Remove from Cogpit (menus_cluster.go)
	p.sh.tip.hide()
	p.menuModel = fyne.NewMenu("", items...)
	widget.NewPopUpMenu(p.menuModel, p.sh.win.Canvas()).ShowAtPosition(pos)
}

// context returns the listed context with this id.
func (p *picker) context(name string) (engine.ContextInfo, bool) {
	for _, c := range p.contexts {
		if c.Name == name {
			return c, true
		}
	}
	return engine.ContextInfo{}, false
}

// dispatchConnect runs cluster.connect for name.
func (p *picker) dispatchConnect(name string) {
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
	title = c.Name
	if c.KubeContext != "" { // a kubeconfig source's context: its name in the file, the source as a badge
		title = c.KubeContext
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
	return kubeconfig.DisplayText(title, maxShownPath), kubeconfig.DisplayText(strings.Join(parts, " · "), maxShownPath)
}

// pickerRow is one context in the picker: title, badges and a subtitle. A tap selects it, a
// double tap connects.
type pickerRow struct {
	widget.BaseWidget
	selected, hovered bool
	onTap, onDouble   func()
	onMenu            func(pos fyne.Position) // the context's menu at pos (window coordinates)
	more              *widget.Button          // ⋮, which opens it too; nil without a hotbar
	badge             *clusterBadge           // the cluster's icon (clusteridentity.go)

	bg     *canvas.Rectangle
	title  *textLine
	sub    *textLine
	badges *fyne.Container
	warn   *widget.Button // opens the connect log; present when the last connect failed
	// note is a warning line under the subtitle: a context whose kubeconfig skips the server's
	// certificate check (insecureNote); nil otherwise. A line, not a tooltip: the window's tooltips
	// are drawn below dialogs, and the picker is one.
	note *textLine
}

func newPickerRow(c engine.ContextInfo, title, sub string, selected bool, onTap, onDouble, onWarn func()) *pickerRow {
	r := &pickerRow{selected: selected, onTap: onTap, onDouble: onDouble, bg: canvas.NewRectangle(color.Transparent),
		badge: newClusterBadge()}
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
	if c.Source != "" {
		r.badges.Add(newPill(kubeconfig.DisplayText(c.Source, maxShownArg), colorMuted, true))
	}
	what := "login program"
	switch c.ExecKind {
	case kubeconfig.LoginTokenFile:
		what = "token file"
	case kubeconfig.LoginInsecure:
		what = "unchecked connection" // A28; the "certificate not checked" pill below says why
	}
	switch c.Exec {
	case engine.ExecRequired:
		r.badges.Add(newPill(what+" not allowed yet", colorStatusWarning, true))
	case engine.ExecChanged:
		r.badges.Add(newPill(what+" changed", colorStatusWarning, true))
	}
	if c.Insecure { // the kubeconfig skips the server's certificate check (engine/tlswarning.go)
		r.badges.Add(newPill("certificate not checked", colorStatusWarning, true))
		r.note = newTextLine(insecureNote(c), colorStatusWarning, theme.SizeNameCaptionText, fyne.TextStyle{})
		r.note.truncate = true
	}
	if c.ConnectProblem { // Lens: a warning icon beside the context opens the log of the connect
		r.warn = widget.NewButtonWithIcon("", theme.NewColoredResource(theme.WarningIcon(), colorStatusWarning), onWarn)
		r.warn.Importance = widget.LowImportance
		r.badges.Add(r.warn)
	}
	r.ExtendBaseWidget(r)
	return r
}

// insecureNote is the warning line of a context whose kubeconfig switches the check of the
// server's certificate off (Cogpit's own text and default, 2026-10-13; Lens documents nothing for
// it). The connect log says the same at length (engine/tlswarning.go).
func insecureNote(c engine.ContextInfo) string {
	server := "its server"
	if c.Server != "" {
		server = kubeconfig.DisplayText(c.Server, maxShownPath)
	}
	return "insecure-skip-tls-verify: the credential goes to whoever answers at " + server
}

// addMenuButton gives the row a ⋮ at its end that opens its menu.
func (r *pickerRow) addMenuButton() {
	r.more = widget.NewButtonWithIcon("", theme.MoreVerticalIcon(), nil)
	r.more.Importance = widget.LowImportance
	r.more.OnTapped = func() {
		pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(r.more)
		r.onMenu(pos.AddXY(0, r.more.Size().Height))
	}
}

// Tapped implements fyne.Tappable.
func (r *pickerRow) Tapped(*fyne.PointEvent) { r.onTap() }

// TappedSecondary implements fyne.SecondaryTappable: the context's menu.
func (r *pickerRow) TappedSecondary(ev *fyne.PointEvent) {
	if r.onMenu != nil {
		r.onMenu(ev.AbsolutePosition)
	}
}

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
	lines := container.NewVBox(container.NewHBox(r.title, r.badges), r.sub)
	if r.note != nil {
		lines.Add(r.note)
	}
	var body fyne.CanvasObject = lines
	var more fyne.CanvasObject
	if r.more != nil {
		more = container.NewCenter(r.more)
	}
	body = container.NewBorder(nil, nil, container.NewHBox(container.NewCenter(r.badge), newSpace(theme.SizeNamePadding)), more, body)
	content := inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, body)
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
