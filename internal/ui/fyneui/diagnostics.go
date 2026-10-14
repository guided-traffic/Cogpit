package fyneui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The Diagnostics dock tab (D-13 cache transparency; engine/diagnostics_view.go): what the status
// bar sums up, in full: the connected cluster's API requests by verb and resource, its watches
// with their state, and the objects per kind, read with diagnostics.get from Cogpit's own counters
// whenever they move (EventDiagnostics, at most once a second). Its first line also counts the
// table snapshots built since start, in every state (connected, the demo cluster, no cluster):
// the only place the GUI shows them since the status bar dropped them (owner, 2026-10-14).
// Memory per kind is not measured. Its layout and wording are Cogpit's (Lens has no such view).

// diagnosticsTab is the tab's content; it is the dock content's editor (dockEditor), so the dock
// closes it.
type diagnosticsTab struct {
	sh *shell

	root     *fyne.Container
	summary  *textLine
	requests *fyne.Container
	watches  *fyne.Container
	kinds    *fyne.Container

	report engine.DiagnosticsReport
	seq    int
	closed bool
}

var _ dockEditor = (*diagnosticsTab)(nil)

func newDiagnosticsTab(sh *shell) *diagnosticsTab {
	d := &diagnosticsTab{sh: sh, requests: container.NewVBox(), watches: container.NewVBox(), kinds: container.NewVBox()}
	d.summary = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	d.summary.truncate = true
	column := func(title string, body *fyne.Container) fyne.CanvasObject {
		return container.NewBorder(inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, heading(title)), nil, nil, nil,
			container.NewVScroll(inset(theme.SizeNameInnerPadding, "", body)))
	}
	cols := container.NewGridWithColumns(3,
		column("API requests by verb and resource", d.requests),
		column("Watches", d.watches),
		column("Objects per kind", d.kinds))
	d.root = container.NewBorder(container.NewVBox(inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, d.summary), newDivider(false)),
		nil, nil, nil, cols)
	d.reload()
	return d
}

// remote implements dockEditor; the tab has no text.
func (d *diagnosticsTab) remote(uint64) {}

// close implements dockEditor: answers that come later are dropped.
func (d *diagnosticsTab) close() { d.closed = true }

// reload reads the counters again.
func (d *diagnosticsTab) reload() {
	d.seq++
	seq := d.seq
	d.sh.dispatch("diagnostics.get", struct{}{}, func(res any, err error) {
		if d.closed || seq != d.seq {
			return
		}
		var r engine.DiagnosticsReport
		if err == nil {
			err = decode(res, &r)
		}
		if err != nil {
			d.summary.SetText("The counters cannot be read: " + errorText(err))
			return
		}
		d.show(r)
	})
}

// show draws a report.
func (d *diagnosticsTab) show(r engine.DiagnosticsReport) {
	d.report = r
	d.summary.SetText(diagnosticsSummary(r))
	line := func(text string, c fyne.ThemeColorName, mono bool) fyne.CanvasObject {
		l := newTextLine(text, c, theme.SizeNameText, fyne.TextStyle{Monospace: mono})
		l.truncate = true
		return l
	}
	var reqs []fyne.CanvasObject
	for _, q := range r.RequestsByTarget {
		reqs = append(reqs, line(fmt.Sprintf("%6s  %s %s", formatCount(q.Count), q.Verb, q.Resource), theme.ColorNameForeground, true))
	}
	if len(reqs) == 0 {
		reqs = append(reqs, line(noneText(r, "No requests since connecting."), colorMuted, false))
	}
	var watches []fyne.CanvasObject
	for _, w := range r.Watches {
		color, text := theme.ColorNameForeground, fmt.Sprintf("%s · %s · %s objects · %s events", w.Scope, w.State, formatCount(int64(w.Objects)), formatCount(w.Events))
		switch {
		case w.Error != "":
			color, text = colorStatusError, text+" · "+w.Error
		case !w.Synced:
			color, text = colorStatusPending, text+" · listing"
		case w.State == engine.WatchCovered:
			color, text = colorMuted, fmt.Sprintf("%s · served by the %s watch · %s objects", w.Scope, w.CoveredBy, formatCount(int64(w.Objects)))
		case w.State == engine.WatchWarm:
			color = colorMuted
		}
		watches = append(watches, line(text, color, false))
	}
	if len(watches) == 0 {
		watches = append(watches, line(noneText(r, "No watches."), colorMuted, false))
	}
	var kinds []fyne.CanvasObject
	for _, k := range r.Kinds {
		kinds = append(kinds, line(fmt.Sprintf("%7s  %s", formatCount(int64(k.Objects)), k.Kind), theme.ColorNameForeground, true))
	}
	if len(kinds) == 0 {
		kinds = append(kinds, line(noneText(r, "No objects held."), colorMuted, false))
	}
	d.requests.Objects, d.watches.Objects, d.kinds.Objects = reqs, watches, kinds
	d.requests.Refresh()
	d.watches.Refresh()
	d.kinds.Refresh()
}

// noneText is what an empty column says: that no cluster is connected, or text.
func noneText(r engine.DiagnosticsReport, text string) string {
	if !r.Connected {
		return "No cluster connected."
	}
	return text
}

// diagnosticsSummary is the tab's first line.
func diagnosticsSummary(r engine.DiagnosticsReport) string {
	if !r.Connected {
		return fmt.Sprintf("No cluster connected · %s table snapshots built", formatCount(int64(r.Snapshots)))
	}
	parts := []string{r.Context}
	if r.Demo {
		parts = append(parts, "the demo cluster: no API requests")
	} else {
		parts = append(parts, formatCount(r.Requests)+" API requests since connecting")
	}
	parts = append(parts,
		fmt.Sprintf("%d open watches, %d warm, %d covered", r.Open, r.Warm, r.Covered),
		formatCount(int64(r.Objects))+" objects held",
		formatCount(int64(r.Snapshots))+" table snapshots")
	return strings.Join(parts, " · ")
}

// diagnosticsChanged reads the counters again for an open Diagnostics tab (EventDiagnostics).
func (d *dock) diagnosticsChanged() {
	for _, c := range d.contents {
		if t, ok := c.editor.(*diagnosticsTab); ok && !c.closed {
			t.reload()
		}
	}
}

// diagnosticsMenuItem is Help › Diagnostics (diagnostics.open); nil when the build has no such
// command.
func (sh *shell) diagnosticsMenuItem() *fyne.MenuItem {
	c, ok := sh.eng.Registry().Get("diagnostics.open")
	if !ok {
		return nil
	}
	return fyne.NewMenuItem(c.Label, func() { sh.dispatch(c.ID, struct{}{}, nil) })
}
