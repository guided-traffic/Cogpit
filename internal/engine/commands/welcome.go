package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The welcome page (engine/welcome.go; Help › Welcome, ui-parity › Layout and Application menu).
// Which page the workspace shows is UI state, so agents open it as the user does (D-2: a UI
// command, run freely).

func registerWelcome(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "ui.welcome.open", Label: "Welcome", MenuPath: "Help/Welcome", Danger: engine.DangerUI,
		Description: "Shows Cogpit's welcome page in the workspace, as Help › Welcome does: the clusters to connect with one " +
			"click, Add Kubeconfigs and the documentation. Over the shown cluster it leaves the active tab, which keeps its " +
			"view, filter, selection and drawer and shows them again when it is selected (ui_tab_select) or a view is opened. " +
			"The page shows by itself while no cluster is connected and while no tab is active; ui_state reports it as welcome. " +
			"Refused while a connect runs. Returns the new UI state.",
		Enabled: func(s *engine.Session, _ struct{}) error { return s.CheckOpenWelcome() },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.OpenWelcome()
		},
	})
}
