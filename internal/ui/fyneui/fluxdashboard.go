package fyneui

import (
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The Flux Dashboard's upper part (resource-parity › Paid Lens Desktop views › Flux CD, G23,
// D-34): a tile per Flux kind the cluster serves, in the navigator's order and sub-groups, with its
// count, a bar of its objects by state and each state's count as a link to the kind's view. The
// objects that are not ready are the page's table below, which the table view shows as it shows any
// view's. The data is the engine's (flux.dashboard). Lens's Flux dashboard is not captured: the
// look is Cogpit's (inferred). The views have no status filter (Cogpit's), so a state's link opens
// the kind's view unfiltered.

// fluxTableTitle is the heading of the dashboard's table.
const fluxTableTitle = "Not Ready"

// fluxColumns is how many tiles a row of the dashboard holds.
const fluxColumns = 3

// fluxPanel shows the Flux dashboard's tiles.
type fluxPanel struct {
	sh    *shell
	box   *fyne.Container // the sections, each a heading (but the first) and a grid of tiles
	tiles map[string]*fluxTile
	root  fyne.CanvasObject

	// Requests of the model are coalesced, as the Workloads Overview's are (overview.go).
	requests atomic.Int32
}

func newFluxPanel(sh *shell) *fluxPanel {
	p := &fluxPanel{sh: sh, tiles: map[string]*fluxTile{}}
	p.box = container.NewVBox()
	p.root = inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, p.box)
	return p
}

// request reads the model off the main goroutine and shows it. It may be called from the main
// goroutine at any rate: requests made while one is in flight become one more.
func (p *fluxPanel) request() {
	for {
		n := p.requests.Load()
		if n >= 2 {
			return
		}
		if p.requests.CompareAndSwap(n, n+1) {
			if n == 1 {
				return // the one in flight runs again when it is done
			}
			break
		}
	}
	p.sh.dispatch("flux.dashboard", struct{}{}, func(res any, err error) {
		again := p.requests.Swap(0) == 2
		if err == nil {
			var m engine.FluxDashboardModel
			if derr := decode(res, &m); derr == nil {
				p.show(m)
			}
		}
		// An error leaves the last counts; the next change of a Flux object asks again.
		if again && p.sh.ui.View == engine.FluxDashboard {
			p.request()
		}
	})
}

// show makes the tiles show m: one per kind the cluster serves, under its sub-group's heading.
func (p *fluxPanel) show(m engine.FluxDashboardModel) {
	var objs []fyne.CanvasObject
	var grid []fyne.CanvasObject
	section := ""
	flush := func() {
		if len(grid) == 0 {
			return
		}
		for len(grid)%fluxColumns != 0 { // a short last row keeps the tiles' width
			grid = append(grid, container.NewStack())
		}
		objs = append(objs, container.NewGridWithColumns(fluxColumns, grid...))
		grid = nil
	}
	seen := map[string]bool{}
	for i, k := range m.Kinds {
		if i == 0 || k.Section != section {
			flush()
			section = k.Section
			if section != "" {
				objs = append(objs, inset("", theme.SizeNameInnerPadding,
					newTextLine(section, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{Bold: true})))
			}
		}
		t := p.tiles[k.Kind]
		if t == nil {
			t = newFluxTile(p.sh, k.Kind)
			p.tiles[k.Kind] = t
		}
		t.set(k)
		seen[k.Kind] = true
		grid = append(grid, t)
	}
	flush()
	if len(m.Kinds) == 0 {
		objs = append(objs, newTextLine("The cluster serves no Flux kinds.", colorMuted, theme.SizeNameText, fyne.TextStyle{}))
	}
	for key := range p.tiles {
		if !seen[key] {
			delete(p.tiles, key)
		}
	}
	p.box.Objects = objs
	p.box.Refresh()
}

// fluxTile is one Flux kind of the dashboard: its title as a link to its view, its count, the bar
// of its states, and each state's count as a link.
type fluxTile struct {
	widget.BaseWidget
	sh     *shell
	kind   string
	link   *linkText
	count  *textLine
	bar    *healthBar
	states *fyne.Container
	// links are the states' links by state ("notReady"): a dot in the state's colour and the
	// count with the state's name, each opening the kind's view.
	links map[string]*tapTarget
	shown []engine.FluxStateCount
	drawn bool
}

func newFluxTile(sh *shell, kind string) *fluxTile {
	t := &fluxTile{sh: sh, kind: kind}
	t.link = newLinkText("", t.open)
	t.count = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	t.bar = newHealthBar(sh)
	t.states = container.NewHBox()
	t.ExtendBaseWidget(t)
	return t
}

// open opens the kind's view.
func (t *fluxTile) open() { t.sh.dispatch("ui.openView", openViewParams{View: t.kind}, nil) }

func (t *fluxTile) set(k engine.FluxDashboardKind) {
	if t.link.text != k.Title {
		t.link.text = k.Title
		t.link.Refresh()
	}
	count := formatCount(int64(k.Count))
	if !k.Synced {
		count += " …"
	}
	t.count.SetText(count)
	statuses := make([]engine.StatusCount, len(k.States))
	names := make(map[model.Status]string, len(k.States))
	for i, st := range k.States {
		statuses[i] = engine.StatusCount{Status: st.Status, Count: st.Count}
		names[st.Status] = st.Title
	}
	t.bar.names = names
	t.bar.set(statuses, k.Count)
	if t.drawn && statesEqual(t.shown, k.States) {
		return
	}
	t.shown, t.drawn = k.States, true
	t.links = make(map[string]*tapTarget, len(k.States))
	objs := make([]fyne.CanvasObject, 0, len(k.States))
	for _, st := range k.States {
		link := newTapTarget(container.NewHBox(newDot(statusColorName(st.Status)),
			newTextLine(formatCount(int64(st.Count))+" "+st.Title, colorLink, theme.SizeNameText, fyne.TextStyle{})), t.open)
		t.links[string(st.State)] = link
		objs = append(objs, link)
	}
	if len(objs) == 0 {
		objs = append(objs, newTextLine("No objects", colorMuted, theme.SizeNameText, fyne.TextStyle{}))
	}
	t.states.Objects = objs
	t.states.Refresh()
}

func statesEqual(a, b []engine.FluxStateCount) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CreateRenderer implements fyne.Widget.
func (t *fluxTile) CreateRenderer() fyne.WidgetRenderer {
	line := container.NewBorder(nil, nil, t.link, t.count)
	return widget.NewSimpleRenderer(container.NewStack(newSurface(theme.ColorNameInputBackground),
		inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, container.NewVBox(line, t.bar, t.states))))
}
