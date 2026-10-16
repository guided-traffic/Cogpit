package fyneui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/helm"
)

// A cluster's Settings › Cogpit Metrics (G12, D-61; engine/metricsstack.go,
// commands/metricsstack.go): Lens's Lens Metrics page (captured: three switches, Prometheus,
// kube-state-metrics and node-exporter, all off, and an Apply button), for the metrics stack Cogpit
// bundles. Each switch shows whether its release is installed and in which state; Apply is enabled
// while the switches differ from what is installed (Lens greys it while nothing changed, inferred).
// Apply asks the engine for the plan first: an install shows its sentence (what is created
// cluster-wide, where the history goes) in the window's own dialog, since the engine confirms an
// install only for agents; a removal is confirmed by the engine, for the window too. The page's
// name, the checks in place of Lens's toggles and the lines are Cogpit's (inferred).

const (
	metricsStackSectionLabel = "Cogpit Metrics"
	metricsStackSetting      = "metricsStack"
)

func init() { clusterSectionLabels[engine.SectionMetricsStack] = metricsStackSectionLabel }

// metricsStackParamNames are cluster.setMetricsStack's parameters, by component id.
var metricsStackParamNames = map[string]string{string(helm.MetricsPrometheus): "prometheus",
	string(helm.MetricsKubeStateMetrics): "kubeStateMetrics", string(helm.MetricsNodeExporter): "nodeExporter"}

// metricsStackView is cluster.metricsStack's answer as the page reads it.
type metricsStackView struct {
	engine.MetricsStackState
	Plan *struct {
		Install     []string `json:"install"`
		Remove      []string `json:"remove"`
		Destructive bool     `json:"destructive"`
		Empty       bool     `json:"empty"`
		Detail      string   `json:"detail"`
	} `json:"plan"`
}

// stackSection is the shown Cogpit Metrics section's controls and the state they show.
type stackSection struct {
	p      *settingsPage
	checks map[string]*widget.Check // by component id
	lines  map[string]*textLine
	apply  *widget.Button
	note   *textLine // why nothing can be applied, or the bundle and the namespace
	store  *textLine // where Prometheus keeps its history
	state  *engine.MetricsStackState
	// shown is what each switch was last set to from the state: a switch the user changed keeps
	// the change until the installed state itself changes (the page's sync runs at every change
	// of the window's state).
	shown map[string]bool
	busy  bool // an Apply in flight
	seq   int
}

// clusterMetricsStackSection builds a cluster's Settings › Cogpit Metrics.
func (p *settingsPage) clusterMetricsStackSection() fyne.CanvasObject {
	sec := &stackSection{p: p, checks: map[string]*widget.Check{}, lines: map[string]*textLine{}, shown: map[string]bool{}}
	p.stack = sec
	rows := container.New(layout.NewFormLayout()) // a switch, and its release's state beside it
	for _, comp := range helm.MetricsComponents {
		c := struct{ id, label string }{string(comp), comp.Label()}
		check := widget.NewCheck(c.label, func(bool) { sec.follow() })
		line := newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
		line.truncate = true
		sec.checks[c.id], sec.lines[c.id] = check, line
		p.controls["metricsStack."+c.id] = check
		rows.Add(check)
		rows.Add(container.NewVBox(layout.NewSpacer(), line, layout.NewSpacer()))
	}
	sec.apply = widget.NewButton("Apply", sec.applyTapped)
	sec.apply.Importance = widget.HighImportance
	sec.apply.Disable()
	sec.note = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	sec.note.truncate = true
	sec.store = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	sec.store.truncate = true
	p.controls["metricsStackApply"], p.controls["metricsStackNote"], p.controls["metricsStackStorage"] = sec.apply, sec.note, sec.store
	p.sync = sec.request
	return p.setting(metricsStackSetting, "Bundled metrics stack",
		"Charts Cogpit bundles, installed with your identity as one Helm release each; Cogpit's charts read from this Prometheus.",
		container.NewVBox(rows, sec.store, sec.note, container.NewHBox(sec.apply)))
}

// request reads the stack's state for the shown cluster, when it is the connected one.
func (sec *stackSection) request() {
	p := sec.p
	v := p.clusterView()
	switch {
	case engine.IsDemoContext(p.shown.Context):
		sec.disable("The demo cluster's metrics are its own: nothing is installed into it.")
		return
	case !v.Connected:
		sec.disable("Connect to this cluster to see and change what Cogpit installed into it.")
		return
	}
	sec.seq++
	seq, ctx := sec.seq, p.shown.Context
	p.sh.dispatch("cluster.metricsStack", struct{}{}, func(res any, err error) {
		if p.stack != sec || seq != sec.seq || p.shown.Context != ctx {
			return
		}
		var view metricsStackView
		switch {
		case err != nil:
			sec.disable(err.Error())
			return
		case decode(res, &view) != nil:
			return
		}
		st := view.MetricsStackState
		sec.state = &st
		sec.show()
	})
}

// disable shows a section that cannot apply anything, and why.
func (sec *stackSection) disable(why string) {
	sec.state = nil
	clear(sec.shown)
	for id, c := range sec.checks {
		sec.p.syncing = true
		c.SetChecked(false)
		sec.p.syncing = false
		c.Disable()
		sec.lines[id].SetText("")
	}
	sec.apply.Disable()
	sec.store.SetText("")
	sec.note.SetText(why)
}

// show puts the state into the controls: each switch at what is installed, its line the release's
// state, Apply off until a switch differs.
func (sec *stackSection) show() {
	st := sec.state
	running := st.Running()
	for _, x := range st.Components {
		c := sec.checks[x.ID]
		if c == nil {
			continue
		}
		if was, ok := sec.shown[x.ID]; !ok || was != x.Installed {
			sec.p.syncing = true
			c.SetChecked(x.Installed)
			sec.p.syncing = false
			sec.shown[x.ID] = x.Installed
		}
		if running || sec.busy || !st.Known {
			c.Disable()
		} else {
			c.Enable()
		}
		sec.lines[x.ID].SetText(componentLine(x))
	}
	sec.store.SetText(storageLine(st.Storage))
	switch {
	case !st.Known:
		sec.note.SetText("Cogpit cannot read what is installed: " + st.Reason)
	case running || sec.busy:
		sec.note.SetText("A Helm action on Cogpit Metrics is running.")
	default:
		sec.note.SetText(fmt.Sprintf("Chart %s, namespace %s%s.", st.Chart, st.Namespace, namespaceNote(*st)))
	}
	sec.follow()
}

// storageLine says where an install's Prometheus keeps its history (the confirmation says it in
// full, with the bounds).
func storageLine(s helm.MetricsStorage) string {
	if s.Persistent() {
		return "Prometheus keeps its history on a volume of StorageClass " + s.StorageClass + ", the cluster's default."
	}
	return "Prometheus keeps its history in its pod (no default StorageClass): lost when the pod is replaced."
}

// componentLine is a switch's line: its release and state.
func componentLine(x engine.MetricsStackComponent) string {
	switch {
	case x.Running:
		return "release " + x.Release + ": a Helm action runs"
	case !x.Installed:
		return "not installed (release " + x.Release + ")"
	}
	line := "installed: release " + x.Release + ", " + x.Status
	if x.ChartVersion != "" {
		line += ", chart " + x.ChartVersion
	}
	if x.Revision > 1 {
		line += fmt.Sprintf(", revision %d", x.Revision)
	}
	return line
}

// namespaceNote says whose the namespace is.
func namespaceNote(st engine.MetricsStackState) string {
	switch {
	case !st.NamespaceExists:
		return " (created with the first install)"
	case st.NamespaceCogpits:
		return " (Cogpit's: deleted with the last component)"
	}
	return " (not Cogpit's: it stays)"
}

// changes are the switches that differ from what is installed, as cluster.setMetricsStack's
// parameters.
func (sec *stackSection) changes() map[string]bool {
	out := map[string]bool{}
	if sec.state == nil {
		return out
	}
	for _, x := range sec.state.Components {
		if c := sec.checks[x.ID]; c != nil && c.Checked != x.Installed {
			out[metricsStackParamNames[x.ID]] = c.Checked
		}
	}
	return out
}

// follow enables Apply while a switch differs from what is installed.
func (sec *stackSection) follow() {
	if sec.p.syncing {
		return
	}
	if len(sec.changes()) > 0 && sec.state != nil && sec.state.Known && !sec.state.Running() && !sec.busy {
		sec.apply.Enable()
	} else {
		sec.apply.Disable()
	}
}

// applyTapped plans the change, shows an install's sentence in a dialog, and applies it.
func (sec *stackSection) applyTapped() {
	params := sec.changes()
	if len(params) == 0 {
		return
	}
	p := sec.p
	p.sh.dispatch("cluster.metricsStack", params, func(res any, err error) {
		if p.stack != sec {
			return
		}
		var view metricsStackView
		if err == nil {
			err = decode(res, &view)
		}
		if err != nil {
			p.showResult(metricsStackSetting, err)
			return
		}
		switch {
		case view.Plan == nil || view.Plan.Empty:
			sec.request()
		case view.Plan.Destructive: // the engine asks, naming what goes
			sec.run(params)
		default:
			sec.confirmInstall(view.Plan.Detail, params)
		}
	})
}

// confirmInstall shows an install's sentence before it runs.
func (sec *stackSection) confirmInstall(detail string, params map[string]bool) {
	sh := sec.p.sh
	width := sh.app.Settings().Theme().Size(sizeConfirmWidth)
	title := "Install Cogpit Metrics in " + sec.p.shown.Context + "?"
	d := dialog.NewCustomConfirm(title, "Install", "Cancel", newParagraph(detail, theme.ColorNameForeground, width), func(ok bool) {
		if ok {
			sec.run(params)
		}
	}, sh.win)
	d.Resize(fyne.NewSize(width, d.MinSize().Height))
	d.Show()
}

// run dispatches cluster.setMetricsStack and shows the state after it.
func (sec *stackSection) run(params map[string]bool) {
	p := sec.p
	sec.busy = true
	if sec.state != nil {
		sec.show()
	}
	p.sh.dispatch("cluster.setMetricsStack", params, func(_ any, err error) {
		sec.busy = false
		if p.stack != sec {
			return
		}
		p.showResult(metricsStackSetting, err)
		sec.request()
	})
}

// metricsStackChanged follows the Helm journal (EventHelm): a release action on the stack started
// or ended.
func (p *settingsPage) metricsStackChanged() {
	if p.stack != nil && p.visible() && p.shown.Page == engine.SettingsCluster && p.shown.Section == engine.SectionMetricsStack && !p.stack.busy {
		p.stack.request()
	}
}
