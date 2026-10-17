package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Preferences › Kubernetes › Integrations of new clusters (D-76; engine/integrations.go): the
// default for every cluster the user made no choice of integrations for, which the question at its
// first connect offers first and the question's "Save as the default for new clusters" sets. Every
// integration is on by default (the owner, 2026-10-17). Cogpit's own: Lens has no integrations;
// its place beside the other cluster-wide defaults of the Kubernetes section is the lead's
// decision (2026-10-17).

const (
	integrationDefaultsSetting = "integrationDefaults"
	integrationDefaultsTitle   = "Integrations of new clusters"
	integrationDefaultsHelp    = "Used by every cluster you have not chosen for; asked at its first connect. A cluster's own choice is in its Settings › Integrations."
)

// newClusterIntegrationsSetting returns the row and the function that shows the preference in it.
func (p *settingsPage) newClusterIntegrationsSetting() (fyne.CanvasObject, func()) {
	checks := make([]*widget.Check, 0, len(engine.Integrations))
	objs := make([]fyne.CanvasObject, 0, len(engine.Integrations))
	for _, in := range engine.Integrations {
		name := in.Name
		c := widget.NewCheck(in.Label, func(on bool) {
			p.setPreference(integrationDefaultsSetting, map[string]bool{string(name): on}) // nothing while syncing
		})
		checks = append(checks, c)
		objs = append(objs, c)
	}
	sync := func() {
		off := p.prefs().IntegrationsOff
		for i, in := range engine.Integrations {
			checks[i].SetChecked(off&in.Set == 0)
		}
	}
	return p.setting(integrationDefaultsSetting, integrationDefaultsTitle, integrationDefaultsHelp, container.NewVBox(objs...)), sync
}
