package fyneui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The Prometheus a cluster's metrics come from, as the user sets it (G12). Lens keeps it in the
// cluster settings' Metrics section ("Auto Detect Prometheus", the service address
// namespace/service:port and a path prefix; captured); Cogpit has no cluster settings page yet,
// so the button under the drawer's status line opens a dialog with the same fields (Cogpit's
// placement, inferred). The engine keeps the setting in Cogpit's preferences for the context
// (cluster.setPrometheus), never in the kubeconfig. Since 2026-10-16 the dialog also has Lens's
// other provider settings (captured: a query format dropdown, "filter empty containers" on, HTTPS
// off): the query format as Cogpit's presets (metrics.QueryFormats, Cogpit's labels), the two
// checks with Lens's defaults.

// prometheusButton is the status line's button that opens the dialog: shown when the cluster has
// Prometheus detection (a real cluster, not the demo).
func newPrometheusButton(sh *shell) *tipButton {
	b := newTipButton(sh, theme.SettingsIcon(), "Set which Prometheus this cluster's metrics come from", nil)
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

// openPrometheusSettings reads the cluster's setting from the engine and shows the dialog with it.
func (sh *shell) openPrometheusSettings() {
	sh.dispatch("cluster.prometheus", struct{}{}, func(res any, err error) {
		if err != nil {
			sh.showError(err)
			return
		}
		set, _ := res.(engine.PrometheusSetting)
		sh.showPrometheusDialog(set)
	})
}

// showPrometheusDialog asks for the Prometheus: detected automatically, or a service address and a
// path prefix. Saving dispatches cluster.setPrometheus; the engine checks the address and refuses
// what is not one (shown as a toast).
func (sh *shell) showPrometheusDialog(set engine.PrometheusSetting) {
	address := widget.NewEntry()
	address.SetPlaceHolder("namespace/service:port")
	address.SetText(set.Address)
	prefix := widget.NewEntry()
	prefix.SetPlaceHolder("/ (none)")
	prefix.SetText(set.Prefix)
	auto := widget.NewCheck("Auto detect Prometheus", func(on bool) {
		if on {
			address.Disable()
			prefix.Disable()
		} else {
			address.Enable()
			prefix.Enable()
		}
	})
	auto.SetChecked(set.Auto)
	formats := make([]string, len(metrics.QueryFormats))
	for i, o := range metrics.QueryFormats {
		formats[i] = o.Label
	}
	format := widget.NewSelect(formats, nil)
	format.SetSelected(metrics.QueryFormats[0].Label)
	for _, o := range metrics.QueryFormats {
		if o.Name == set.Format {
			format.SetSelected(o.Label)
		}
	}
	filter := widget.NewCheck("Filter empty containers", nil)
	filter.SetChecked(set.FilterEmptyContainers || set.Format == "") // a setting read before 2026-10-16: Lens's default
	https := widget.NewCheck("Use HTTPS", nil)
	https.SetChecked(set.HTTPS)
	note := "Kept in Cogpit's settings for this cluster; the kubeconfig is not changed."
	if !set.Kept {
		note = "Kept until Cogpit closes (no settings folder); the kubeconfig is not changed."
	}
	items := []*widget.FormItem{
		{Text: "", Widget: auto},
		{Text: "Service address", Widget: address, HintText: "read through the API server's service proxy"},
		{Text: "Path prefix", Widget: prefix},
		{Text: "Query format", Widget: format, HintText: "how this Prometheus names a node's series"},
		{Text: "", Widget: filter, HintText: "leave the series without a container name out of the pods' queries"},
		{Text: "", Widget: https, HintText: "reach the Prometheus service over https through the service proxy"},
		{Text: "", Widget: newTextLine(note, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})},
	}
	title := "Prometheus for " + set.Context
	d := dialog.NewForm(title, "Save", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		params := struct {
			Address               string `json:"address"`
			Prefix                string `json:"prefix,omitempty"`
			Format                string `json:"format"`
			FilterEmptyContainers bool   `json:"filterEmptyContainers"`
			HTTPS                 bool   `json:"https"`
		}{Address: autoSetting, Format: autoSetting, FilterEmptyContainers: filter.Checked, HTTPS: https.Checked}
		for _, o := range metrics.QueryFormats {
			if o.Label == format.Selected {
				params.Format = o.Name
			}
		}
		if !auto.Checked {
			params.Address, params.Prefix = strings.TrimSpace(address.Text), strings.TrimSpace(prefix.Text)
		}
		sh.dispatch("cluster.setPrometheus", params, func(_ any, err error) { sh.showError(err) })
	}, sh.win)
	th := sh.app.Settings().Theme()
	d.Resize(fyne.NewSize(th.Size(sizeDialogWidth), d.MinSize().Height))
	d.Show()
}

// autoSetting is cluster.setPrometheus's word for detection (address) and the detected query format.
const autoSetting = "auto"

// queryLine says how the Prometheus is asked when that is not Lens's default, for the settings
// page's line: " Query format: …", " Empty containers kept.", " Over https.".
func queryLine(set engine.PrometheusSetting) string {
	var out string
	if set.Format != "" && set.Format != autoSetting {
		out += " Query format: " + set.FormatLabel + "."
	}
	if set.Format != "" && !set.FilterEmptyContainers {
		out += " Empty containers kept."
	}
	if set.HTTPS {
		out += " Over https."
	}
	return out
}
