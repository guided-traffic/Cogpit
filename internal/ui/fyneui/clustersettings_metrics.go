package fyneui

import (
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/metrics"
)

// Cluster settings › Metrics › Hide metrics from the UI (G12, G13; engine/hidemetrics.go). Lens
// has a multi-select of metrics, "Hide all metrics" and "Reset" buttons and a line that says all
// metrics are visible when none is hidden (captured). Cogpit's control: two rows of checks, the
// places metrics show in (Cluster: the Overview's charts; Nodes; Pods; Containers) and what they
// measure (CPU, Memory, Network, Filesystem), the two buttons and the line. The list's entries
// and the rows are Cogpit's (inferred: Lens's list is not captured). A change is dispatched at
// once (cluster.settings.set with hiddenMetrics) and the window follows the engine's state: the
// lists' columns, the drawer's chart variants and the Overview's charts leave, and nothing is
// polled for them.

// hiddenMetricsSetting is the setting's name on the page (its control and its refusal line).
const hiddenMetricsSetting = "hiddenMetrics"

// hideMetricsSetting builds the Hide metrics setting of the shown cluster and returns it with the
// function that shows the state in it.
func (p *settingsPage) hideMetricsSetting() (fyne.CanvasObject, func()) {
	ctx := p.shown.Context
	labels := func(place bool) []string {
		var out []string
		for _, h := range engine.HideableMetrics {
			if h.Place == place {
				out = append(out, h.Label)
			}
		}
		return out
	}
	places := widget.NewCheckGroup(labels(true), nil)
	places.Horizontal = true
	measures := widget.NewCheckGroup(labels(false), nil)
	measures.Horizontal = true
	selected := func() engine.MetricSet {
		var m engine.MetricSet
		for _, h := range engine.HideableMetrics {
			if slices.Contains(places.Selected, h.Label) || slices.Contains(measures.Selected, h.Label) {
				m |= h.Set
			}
		}
		return m
	}
	set := func(m engine.MetricSet) { p.setCluster(hiddenMetricsSetting, engine.MetricNames(m)) }
	changed := func([]string) {
		if !p.syncing {
			set(selected())
		}
	}
	places.OnChanged, measures.OnChanged = changed, changed
	hideAll := widget.NewButton("Hide all metrics", func() { set(engine.HideAll) })
	reset := widget.NewButton("Reset", func() { set(0) })
	line := newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	line.truncate = true
	p.controls["hiddenPlaces"], p.controls["hiddenMeasures"] = places, measures
	p.controls["hideAllMetrics"], p.controls["resetMetrics"], p.controls["hiddenMetricsLine"] = hideAll, reset, line

	caption := func(text string) fyne.CanvasObject {
		return container.NewCenter(newTextLine(text, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{}))
	}
	control := container.NewVBox(
		container.New(layout.NewFormLayout(), caption("Places"), places, caption("Measures"), measures),
		container.NewHBox(hideAll, reset),
		line,
	)
	sync := func() {
		m := p.sh.ui.ClusterSettings[ctx].HiddenMetrics // the UI state's, so an agent's change shows at once
		pick := func(place bool) []string {
			var out []string
			for _, h := range engine.HideableMetrics {
				if h.Place == place && m&h.Set != 0 {
					out = append(out, h.Label)
				}
			}
			return out
		}
		if want := pick(true); !slices.Equal(want, places.Selected) {
			places.SetSelected(want)
		}
		if want := pick(false); !slices.Equal(want, measures.Selected) {
			measures.SetSelected(want)
		}
		line.SetText(hiddenMetricsLine(m))
		if m == 0 {
			reset.Disable()
		} else {
			reset.Enable()
		}
		if m == engine.HideAll {
			hideAll.Disable()
		} else {
			hideAll.Enable()
		}
	}
	return p.setting(hiddenMetricsSetting, "Hide metrics from the UI",
		"Hidden metrics leave the lists, the drawers and the Overview, and Cogpit stops asking the cluster for them.", control), sync
}

// metricsSourceSetting is the name of the Metrics source setting on the page.
const metricsSourceSetting = "metricsSource"

// metricsSourceSetting builds Cluster settings › Metrics › Metrics source (G12; Lens's dropdown,
// captured: Automatic, Prometheus, Kubernetes Metrics Server, No metrics, and under it a line naming
// the source in use) and returns it with the function that shows the state in it. A choice is
// dispatched at once (cluster.settings.set with metricsSource); the select follows the UI state, so
// an agent's change shows too.
func (p *settingsPage) metricsSourceSetting() (fyne.CanvasObject, func()) {
	ctx := p.shown.Context
	labels := make([]string, 0, len(engine.MetricsSources))
	for _, o := range engine.MetricsSources {
		labels = append(labels, o.Label)
	}
	sel := widget.NewSelect(labels, func(label string) {
		for _, o := range engine.MetricsSources {
			if o.Label == label && !p.syncing {
				p.setCluster(metricsSourceSetting, o.Name)
			}
		}
	})
	line := newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	line.truncate = true
	p.controls["metricsSourceLine"], p.controls["metricsSourceSelect"] = line, sel
	sync := func() {
		name := p.sh.ui.MetricsSourceOf(ctx)
		for _, o := range engine.MetricsSources {
			if o.Name == name && sel.Selected != o.Label {
				sel.SetSelected(o.Label)
			}
		}
		text := metricsSourceLine(p.clusterView(), name)
		if u := p.sh.ui; name == string(metrics.SourceNone) && u.IntegrationsAskFirst && !u.IntegrationsDecided(ctx) {
			text = metricsSourceUndecided // nothing is asked before the user chose (D-82)
		}
		line.SetText(text)
	}
	return p.setting(metricsSourceSetting, "Metrics source",
		"Automatic: history from Prometheus, current values from the Metrics Server. The others turn one or both off.",
		container.NewVBox(sel, line)), sync
}

// metricsSourceUndecided is the line under the setting while the user chose no integrations for the
// cluster: nothing of them is asked until then (D-82), its metrics included.
const metricsSourceUndecided = "Not chosen for this cluster yet: no metrics are asked until you choose (here, in Integrations, or in its question)."

// metricsSourceLine names the source in use under the setting.
func metricsSourceLine(v engine.ClusterSettingsView, name string) string {
	switch {
	case name == string(metrics.SourceNone):
		return "No metrics are asked of this cluster."
	case !v.Connected || engine.IsDemoContext(v.Context) && v.MetricsInUse == "":
		return "Applies while this cluster is connected."
	case v.MetricsInUse == "":
		return "In use: looking for the providers…"
	}
	return "In use: " + v.MetricsInUse + "."
}

// hiddenMetricsLine says what is hidden: "All metrics are visible.", or the hidden ones.
func hiddenMetricsLine(m engine.MetricSet) string {
	switch m {
	case 0:
		return "All metrics are visible."
	case engine.HideAll:
		return "All metrics are hidden."
	}
	return "Hidden: " + strings.Join(m.Labels(), ", ") + "."
}

// showVariants shows a chart's variant buttons the cluster's settings leave visible (all of them
// for a panel without the engine's list yet) and hides the others; the row closes up around them.
func (c *chartCtl) showVariants(visible []string) {
	for id, b := range c.variants {
		if visible == nil || slices.Contains(visible, id) {
			b.Show()
		} else {
			b.Hide()
		}
	}
	c.controls.Refresh()
}

// variantsShown are the variants the panel's charts offer: the engine's list, nil (all) before it
// answered.
func (p *metricsPanel) variantsShown() []string {
	if p.data == nil || p.data.Variants == nil {
		return nil
	}
	return p.data.Variants
}
