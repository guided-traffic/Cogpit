package fyneui

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/metrics"
)

// Cluster settings › Metrics › Metrics source (G12; Lens's dropdown, captured): Automatic,
// Prometheus, Kubernetes Metrics Server, No metrics, a line naming the source in use; a choice
// applies to the connected cluster at once: No metrics empties the Pods list's CPU and agents
// read the state, Automatic brings the numbers back.
func TestMetricsSourceFromTheSettings(t *testing.T) {
	s, sh, _, until := demoSettingsShell(t)
	mustDispatch(t, s, "cluster.settings.open", map[string]any{"context": engine.DemoContext, "section": "metrics"})
	until("the Metrics source", func() bool {
		l, ok := sh.settings.controls["metricsSourceLine"].(*textLine)
		return ok && sh.settings.view != nil && l.text == "In use: "+metrics.ServerLabel+"."
	})
	sel := sh.settings.controls["metricsSourceSelect"].(*widget.Select)
	if want := []string{"Automatic", "Prometheus", "Kubernetes Metrics Server", "No metrics"}; !slices.Equal(sel.Options, want) || sel.Selected != "Automatic" {
		t.Fatalf("options %q, selected %q", sel.Options, sel.Selected)
	}

	sel.SetSelected("No metrics")
	until("metrics off", func() bool {
		cs, _ := s.ClusterSettingsOf(engine.DemoContext)
		l := sh.settings.controls["metricsSourceLine"].(*textLine)
		return cs.MetricsSource == "none" && l.text == "No metrics are asked of this cluster."
	})
	mustDispatch(t, s, "settings.close", struct{}{})
	until("the CPU column empty", func() bool { return sh.view.snap != nil && usageRow(sh.view.snap) < 0 })
	// No request with metrics off is counted at the hub (TestMetricsSourceNone); here the window
	// and agents see the state.
	if m, _ := s.MetricsGet(t.Context(), "pods", "", "", 1); m.Status != metrics.StatusNotServed {
		t.Errorf("metrics.get with metrics off: %+v", m.Status)
	}

	// An agent sets Automatic again: the select follows, the numbers come back.
	mustDispatch(t, s, "cluster.settings.set", map[string]any{"context": engine.DemoContext, "metricsSource": "auto"})
	until("the CPU column back", func() bool { return sh.view.snap != nil && usageRow(sh.view.snap) >= 0 })
	mustDispatch(t, s, "cluster.settings.open", map[string]any{"context": engine.DemoContext, "section": "metrics"})
	until("Automatic selected", func() bool {
		sel, ok := sh.settings.controls["metricsSourceSelect"].(*widget.Select)
		return ok && sel.Selected == "Automatic"
	})
	if _, err := s.Dispatch(t.Context(), engine.Invocation{ID: "cluster.settings.set", Params: []byte(`{"metricsSource":"thanos"}`), Origin: engine.OriginExternalAgent}); err == nil {
		t.Error("an unknown metrics source was accepted")
	}
}
