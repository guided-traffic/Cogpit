package fyneui

import (
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The Prometheus a cluster's metrics come from, as the user sets it (G12). Lens keeps it in the
// cluster settings' Metrics section (captured); so does Cogpit since 2026-10-16 night: the fields
// are on Cluster Settings › Metrics (clustersettings_prometheus.go), and the button under the
// drawer's and the Overview's status line opens that page. Until then the button opened a dialog
// with the same fields (Cogpit's placement, from before the settings page existed).

// newPrometheusButton is the status line's button that opens Cluster Settings › Metrics: shown when
// the cluster has Prometheus detection (a real cluster, not the demo).
func newPrometheusButton(sh *shell) *tipButton {
	b := newTipButton(sh, theme.SettingsIcon(), "Set which Prometheus this cluster's metrics come from (Cluster Settings › Metrics)", nil)
	b.Text, b.Importance = "Prometheus…", widget.LowImportance
	b.OnTapped = sh.openPrometheusSettings
	b.Hide()
	return b
}

// hasPrometheusSlot reports whether the providers include Prometheus, found or not.
func hasPrometheusSlot(ps []metrics.ProviderStatus) bool {
	for _, p := range ps {
		if p.Provider == metrics.PrometheusID {
			return true
		}
	}
	return false
}

// openPrometheusSettings opens the shown cluster's Settings › Metrics, where its Prometheus is set.
func (sh *shell) openPrometheusSettings() {
	sh.dispatch("cluster.settings.open", clusterSettingsOpenParams{Section: engine.SectionMetrics}, nil)
}
