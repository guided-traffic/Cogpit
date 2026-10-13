package fyneui

import (
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// The Argo CD Dashboard's upper part (resource-parity › Paid Lens Desktop views › Argo CD): Lens's
// documented health strip (Applications, Synced, Healthy, Out of sync, Degraded, Missing,
// Progressing), a card per Argo CD kind with its count and health bar, the Applications card with
// Sync and Refresh of several Applications, and the recent activity. Needs attention below is the
// page's table. The model is the engine's (argocd.dashboard); the look is Cogpit's own (inferred:
// Lens's dashboard is not captured).

// argoActivityLines bounds the activity the panel lists; agents read 20.
const argoActivityLines = 8

// argoPanel shows the Dashboard's model.
type argoPanel struct {
	sh       *shell
	data     *engine.ArgoCDDashboardModel
	strip    *fyne.Container
	cards    *fyne.Container
	activity *fyne.Container
	filter   *widget.Entry // the activity's filter: the window's, as the navigator's search is
	openUI   *widget.Button
	sync     *widget.Button
	refresh  *widget.Button
	root     fyne.CanvasObject

	requests atomic.Int32 // coalesced as the Workloads Overview's (overview.go)
}

func newArgoPanel(sh *shell) *argoPanel {
	p := &argoPanel{sh: sh}
	p.strip = container.NewGridWithColumns(7)
	p.cards = container.NewGridWithColumns(5)
	p.activity = container.NewVBox()
	p.sync = widget.NewButtonWithIcon("Sync Out of Sync…", theme.UploadIcon(), func() {
		if p.data != nil {
			sh.openArgoSyncDialog(p.data.OutOfSync)
		}
	})
	p.refresh = widget.NewButtonWithIcon("Refresh All…", theme.ViewRefreshIcon(), func() {
		if p.data != nil {
			sh.openArgoRefreshDialog(p.data.All)
		}
	})
	// Lens's "Open Argo UI": a port forward to argocd-server, opened in the browser.
	p.openUI = widget.NewButtonWithIcon("Open Argo CD UI", theme.ComputerIcon(), func() {
		sh.dispatch("argocd.openUI", struct{}{}, func(res any, err error) { sh.forwardDone(res, err, true) })
	})
	p.sync.Disable()
	p.refresh.Disable()
	actions := container.NewHBox(newTextLine("Applications", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(), p.openUI, p.sync, p.refresh)
	title := newTextLine("Recent activity", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{Bold: true})
	p.filter = widget.NewEntry()
	p.filter.SetPlaceHolder("Filter activity…")
	p.filter.OnChanged = func(string) { p.showActivity() }
	head := container.NewBorder(nil, nil, title, sized(sizeSearchWidth, "", p.filter))
	p.root = inset(theme.SizeNameInnerPadding, theme.SizeNamePadding,
		container.NewVBox(p.strip, newSpace(theme.SizeNamePadding), actions, p.cards, newSpace(theme.SizeNamePadding), head, p.activity))
	return p
}

// request reads the model off the main goroutine and shows it, coalesced.
func (p *argoPanel) request() {
	for {
		n := p.requests.Load()
		if n >= 2 {
			return
		}
		if p.requests.CompareAndSwap(n, n+1) {
			if n == 1 {
				return
			}
			break
		}
	}
	p.sh.dispatch("argocd.dashboard", struct{}{}, func(res any, err error) {
		again := p.requests.Swap(0) == 2
		if err == nil {
			var m engine.ArgoCDDashboardModel
			if derr := decode(res, &m); derr == nil {
				p.show(&m)
			}
		}
		if again && p.sh.ui.View == engine.ArgoCDDashboard {
			p.request()
		}
	})
}

// show draws a model.
func (p *argoPanel) show(m *engine.ArgoCDDashboardModel) {
	p.data = m
	tiles := make([]fyne.CanvasObject, 0, len(m.Strip))
	for _, t := range m.Strip {
		tiles = append(tiles, newArgoTile(p.sh, t, m.Synced))
	}
	p.strip.Objects = tiles
	p.strip.Refresh()
	cards := make([]fyne.CanvasObject, 0, len(m.Kinds))
	for _, k := range m.Kinds {
		cards = append(cards, newArgoCard(p.sh, k))
	}
	p.cards.Objects = cards
	p.cards.Refresh()
	p.enable(p.sync, len(m.OutOfSync) > 0)
	p.enable(p.refresh, len(m.All) > 0)
	p.showActivity()
}

// showActivity lists the newest activities that match the filter (the Application's namespace and
// name, what happened, the revision; every term, ignoring case), at most argoActivityLines.
func (p *argoPanel) showActivity() {
	if p.data == nil {
		return
	}
	terms := strings.Fields(strings.ToLower(p.filter.Text))
	lines := make([]fyne.CanvasObject, 0, argoActivityLines)
	now := p.sh.now()
	for _, a := range p.data.Activity {
		if len(lines) == argoActivityLines {
			break
		}
		text := strings.ToLower(a.Ref.Key() + " " + a.Text + " " + a.Revision)
		if !slices.ContainsFunc(terms, func(t string) bool { return !strings.Contains(text, t) }) {
			lines = append(lines, newArgoActivityLine(p.sh, a, now))
		}
	}
	switch {
	case len(lines) > 0:
	case len(terms) > 0:
		lines = append(lines, newTextLine("No activity matches the filter.", colorMuted, theme.SizeNameText, fyne.TextStyle{}))
	default:
		lines = append(lines, newTextLine("No operations yet.", colorMuted, theme.SizeNameText, fyne.TextStyle{}))
	}
	p.activity.Objects = lines
	p.activity.Refresh()
}

func (p *argoPanel) enable(b *widget.Button, on bool) {
	if on {
		b.Enable()
	} else {
		b.Disable()
	}
}

// newArgoTile is one count of the strip: the count large, in its status colour when not zero, the
// label below; a click opens the Applications view searched for the state (the view has no status
// filter).
func newArgoTile(sh *shell, t engine.ArgoTile, synced bool) fyne.CanvasObject {
	color := theme.ColorNameForeground
	if t.Count > 0 && t.Status != model.StatusNone {
		color = statusColorName(t.Status)
	}
	count := formatCount(int64(t.Count))
	if !synced {
		count += " …"
	}
	num := newTextLine(count, color, theme.SizeNameHeadingText, fyne.TextStyle{Bold: true})
	label := newLinkText(t.Label, func() { sh.openArgoApplications(t.Filter) })
	return container.NewStack(newSurface(theme.ColorNameInputBackground),
		inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewVBox(num, label)))
}

// openArgoApplications opens Argo CD's Applications view, searched for filter ("" unfiltered).
func (sh *shell) openArgoApplications(filter string) {
	sh.dispatch("ui.openView", openViewParams{View: kinds.ArgoApplicationsKey}, func(_ any, err error) {
		if err != nil {
			sh.showError(err)
			return
		}
		if filter != "" {
			sh.dispatch("ui.setFilter", filterParams{Filter: filter}, nil)
		}
	})
}

// newArgoCard is one kind's card: its title as a link to its view, its count, its health bar.
func newArgoCard(sh *shell, k engine.OverviewKind) fyne.CanvasObject {
	t := newOverviewTile(sh, k.Kind)
	t.set(k)
	return t
}

// newArgoActivityLine is one entry of the recent activity: how long ago, the Application (a link
// to it), what happened in its status colour, the revision.
func newArgoActivityLine(sh *shell, a engine.ArgoActivityItem, now time.Time) fyne.CanvasObject {
	ago := newTextLine(model.FormatAge(now.Sub(a.At))+" ago", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	ref := a.Ref
	app := newLinkText(ref.Name, func() { sh.dispatch("ui.navigate", refParams{Ref: ref}, nil) })
	text := a.Text
	if a.Revision != "" {
		text += " · " + a.Revision
	}
	what := newTextLine(text, statusColorName(a.Status), theme.SizeNameText, fyne.TextStyle{})
	what.truncate = true
	return container.NewBorder(nil, nil, container.NewHBox(ago, box(naturalWidth{text: ref.Name}, app)), nil, what)
}

// naturalWidth lays out a link (which takes any width, shortening its text) as wide as its text.
type naturalWidth struct{ text string }

func (n naturalWidth) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.MeasureText(n.text, fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameText), fyne.TextStyle{})
}

func (n naturalWidth) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}
