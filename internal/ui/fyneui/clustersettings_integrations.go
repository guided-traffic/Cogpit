package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// A cluster's Settings › Integrations (D-76; engine/integrations.go): each integration of the
// cluster switched on or off, and at once whether Cogpit found it automatically or it needs a
// manual configuration (the owner's requirement of 2026-10-17, I1): one row per integration,
// its switch with its label, a status line in the status colours ("Found automatically:
// monitoring/kube-prometheus-alertmanager:9093", "Needs manual configuration: none found by name",
// "Not in this cluster: …", "No permission: …", "Checking…", "Off", "Not connected"), and the line
// of what it adds. Lens has no such section: its place (before Metrics), the wording and the rows
// are Cogpit's (the lead's decisions, 2026-10-17). A switch dispatches cluster.settings.set's
// integrations at once and makes the choice the cluster's own; the status follows
// EventIntegrations and EventMetrics. The window's question about a new cluster's integrations
// (integrationsask.go) shows the same rows.

const (
	integrationsSetting = "integrations"
	integrationsLabel   = "Integrations"
	integrationsHelp    = "Tools this cluster runs that Cogpit reads to show more. Off: Cogpit asks the cluster nothing for it and shows nothing of it."
	// integrationsUndecided is the note while the user made no choice for the cluster.
	integrationsUndecided = "Not chosen for this cluster yet: the default for new clusters applies (Preferences › Kubernetes)."
)

func init() { clusterSectionLabels[engine.SectionIntegrations] = integrationsLabel }

// integrationRowExtras are what an integration's own file puts under its row in the section, for
// the shown context (Alertmanager's address fields, filled in an init of its file); none: the row
// alone. Built with the section; the extra follows the state by itself.
var integrationRowExtras = map[engine.Integration]func(p *settingsPage, ctx string) fyne.CanvasObject{}

// integrationRow is one integration's row: its switch with its label, the status line, and the line
// of what it adds; on the settings page a link to where a manual address is set (Prometheus's).
type integrationRow struct {
	name   engine.Integration
	check  *widget.Check
	status *textLine
	slot   *fyne.Container // holds the status line, and the link beside it
	link   *linkText       // "Set its address…" for Prometheus while it needs one; nil elsewhere
	adds   *textLine
	box    *fyne.Container
	state  engine.IntegrationState  // the status shown, for the link and tests
	last   engine.IntegrationStatus // the status as the engine gave it
}

// newIntegrationRow builds a row; onSwitch receives the user's switch (not the state's).
func newIntegrationRow(in engine.IntegrationInfo, onSwitch func(on bool)) *integrationRow {
	r := &integrationRow{name: in.Name}
	r.check = widget.NewCheck(in.Label, onSwitch)
	r.status = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	r.status.truncate = true
	r.adds = newTextLine(in.Adds, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	r.adds.truncate = true
	r.slot = container.NewStack(r.status)
	r.box = box(layout.NewCustomPaddedVBoxLayout(0), r.check,
		box(checkIndent{}, box(layout.NewCustomPaddedVBoxLayout(2), r.slot, r.adds)))
	return r
}

// withLink adds the link to where its address is set, shown while the status asks for one.
func (r *integrationRow) withLink(text string, onTap func()) {
	r.link = newLinkText(text, onTap)
	r.link.size = theme.SizeNameCaptionText
	r.link.Hide()
	r.slot.Objects = []fyne.CanvasObject{box(textThenLink{}, r.status, r.link)}
	r.slot.Refresh()
}

// withExtra puts an integration's own controls under its row.
func (r *integrationRow) withExtra(o fyne.CanvasObject) {
	if o != nil {
		r.box.Add(box(checkIndent{}, inset("", theme.SizeNamePadding, o)))
	}
}

// setStatus shows what Cogpit knows of the integration.
func (r *integrationRow) setStatus(st engine.IntegrationStatus) {
	text, c := integrationStatusLine(st)
	r.state, r.last = st.State, st
	r.status.SetText(text)
	r.status.SetColor(c)
	if r.link != nil {
		if st.State == engine.IntegrationManual || st.State == engine.IntegrationUnavailable || st.State == engine.IntegrationConfigured {
			r.link.Show()
		} else {
			r.link.Hide()
		}
	}
}

// integrationStatusLine is an integration's status line and its status colour.
func integrationStatusLine(st engine.IntegrationStatus) (string, fyne.ThemeColorName) {
	with := func(prefix, more string) string {
		if more == "" {
			return prefix
		}
		return prefix + ": " + more
	}
	where := func() string {
		if st.Where != "" && st.Detail != "" {
			return st.Where + " (" + st.Detail + ")"
		}
		return st.Where + st.Detail
	}
	switch st.State {
	case engine.IntegrationFound:
		return with("Found automatically", st.Where), colorStatusOK
	case engine.IntegrationConfigured:
		return with("Configured", st.Where), colorStatusOK
	case engine.IntegrationManual:
		return with("Needs manual configuration", st.Detail), colorStatusWarning
	case engine.IntegrationMissing:
		return with("Not in this cluster", st.Detail), colorMuted
	case engine.IntegrationForbidden:
		return with("No permission", where()), colorStatusError
	case engine.IntegrationUnavailable:
		return with("Not answering", where()), colorStatusError
	case engine.IntegrationOff:
		return "Off: nothing is asked of the cluster", colorMuted
	case engine.IntegrationNotConnected:
		return "Not connected: found when the cluster is connected", colorMuted
	}
	return "Checking…", colorStatusPending
}

// clusterIntegrationsSection builds a cluster's Settings › Integrations.
func (p *settingsPage) clusterIntegrationsSection() fyne.CanvasObject {
	ctx := p.shown.Context
	help := newTextLine(integrationsHelp, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	help.truncate = true
	note := newTextLine(integrationsUndecided, colorStatusPending, theme.SizeNameCaptionText, fyne.TextStyle{})
	note.truncate = true
	p.controls["integrationsNote"] = note
	rows := make([]*integrationRow, 0, len(engine.Integrations))
	objs := make([]fyne.CanvasObject, 0, 3+len(engine.Integrations))
	objs = append(objs, help, note, newSpace(theme.SizeNamePadding))
	for _, in := range engine.Integrations {
		name := in.Name
		r := newIntegrationRow(in, func(on bool) {
			p.setCluster(integrationsSetting, map[string]bool{string(name): on}) // nothing while syncing
		})
		if name == engine.IntegrationPrometheus {
			r.withLink("Set its address…", func() {
				p.sh.dispatch("cluster.settings.open", clusterSettingsOpenParams{Context: ctx, Section: engine.SectionMetrics}, nil)
			})
		}
		if extra := integrationRowExtras[name]; extra != nil {
			r.withExtra(extra(p, ctx))
		}
		p.controls["integration:"+string(name)] = r.box
		rows = append(rows, r)
		objs = append(objs, r.box)
	}
	p.integrationRows = rows
	p.sync = func() {
		v := p.clusterView()
		statuses := map[engine.Integration]engine.IntegrationStatus{}
		if p.view != nil {
			for _, st := range v.Integrations {
				statuses[st.Name] = st
			}
		}
		for _, r := range rows {
			r.check.SetChecked(p.sh.ui.IntegrationOn(ctx, r.name)) // the UI state's, so the switch follows at once
			if st, ok := statuses[r.name]; ok {
				r.setStatus(st)
			}
		}
		if p.sh.ui.IntegrationsDecided(ctx) {
			note.Hide()
		} else {
			note.Show()
		}
	}
	return container.NewVBox(p.setting(integrationsSetting, integrationsLabel, "", box(layout.NewCustomPaddedVBoxLayout(8), objs...)))
}

// integrationsChanged follows a change of what Cogpit knows of a cluster's integrations
// (EventIntegrations for ctx; "" for any cluster, EventMetrics): the Integrations section and the
// question read them again. Main goroutine.
func (sh *shell) integrationsChanged(ctx string) {
	p := sh.settings
	if p.visible() && p.shown.Page == engine.SettingsCluster && p.shown.Section == engine.SectionIntegrations &&
		(ctx == "" || ctx == p.shown.Context) {
		p.requestView()
	}
	sh.intq.statusesChanged(ctx)
}

// checkIndent lays its objects out from where a check's label starts, so that the lines under a
// switch line up with its label (widget.Check's layout: the focus square, the borders, the
// label's own padding).
type checkIndent struct{}

func (checkIndent) offset() float32 {
	th := fyne.CurrentApp().Settings().Theme()
	return th.Size(theme.SizeNameInlineIcon) + th.Size(theme.SizeNameInnerPadding) + 2*th.Size(theme.SizeNameInputBorder)
}

// textThenLink lays a text line out at its natural width (shortened when the room is short) with a
// link right after it, on one line.
type textThenLink struct{}

func (textThenLink) gap() float32 {
	return fyne.CurrentApp().Settings().Theme().Size(theme.SizeNameInnerPadding)
}

func (l textThenLink) MinSize(objs []fyne.CanvasObject) fyne.Size {
	ms := objs[0].MinSize()
	if objs[1].Visible() {
		lm := objs[1].MinSize()
		ms = fyne.NewSize(ms.Width+l.gap()+lm.Width, max(ms.Height, lm.Height))
	}
	return ms
}

func (l textThenLink) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	text, link := objs[0], objs[1]
	lw := float32(0)
	if link.Visible() {
		lw = link.MinSize().Width + l.gap()
	}
	tw := size.Width - lw
	if t, ok := text.(*textLine); ok {
		th := t.Theme()
		tw = min(tw, fyne.MeasureText(t.text, th.Size(t.size), t.style).Width+1)
	}
	text.Move(fyne.NewPos(0, 0))
	text.Resize(fyne.NewSize(max(tw, 0), size.Height))
	link.Move(fyne.NewPos(max(tw, 0)+l.gap(), 0))
	link.Resize(fyne.NewSize(link.MinSize().Width, size.Height))
}

func (l checkIndent) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var ms fyne.Size
	for _, o := range objs {
		if o.Visible() {
			ms = ms.Max(o.MinSize())
		}
	}
	return ms.AddWidthHeight(l.offset(), 0)
}

func (l checkIndent) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	x := l.offset()
	for _, o := range objs {
		o.Move(fyne.NewPos(x, 0))
		o.Resize(fyne.NewSize(max(size.Width-x, 0), size.Height))
	}
}
