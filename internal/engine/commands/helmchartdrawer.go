package commands

import (
	"context"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// ui.setChartVersion: the version a Helm chart's drawer shows (G11, HELM2; engine/helmchartdrawer.go).
// Like a metrics chart's variant it is UI state of the open drawer, so the drawer's version picker
// and an agent set the same thing; it ends when the drawer closes or shows another object. The
// drawer then shows that version's properties and README, which Cogpit downloads from the chart's
// repository (its archive, as an install does) when it does not keep it yet.

type chartVersionParams struct {
	Ref     model.ObjectRef `json:"ref" jsonschema:"the Helm chart the detail drawer shows (ui_state's drawer): kind helmcharts, name the chart's id as resources_list lists it (<repository>.<chart>, every dot of the repository's name doubled)"`
	Version string          `json:"version,omitempty" jsonschema:"the chart version to show, one of the chart's spec.versions (object_get's helmChart.versions); empty: the version the chart is listed with"`
}

func registerChartDrawer(g *registrar) {
	add(g, engine.Spec[chartVersionParams]{
		ID: "ui.setChartVersion", Label: "Chart Version", Danger: engine.DangerUI,
		Description: "Sets the version a Helm chart's detail drawer shows, as the drawer's version picker does: its properties " +
			"and README are then that version's (object_get returns them, helmChart.version and the README section), and " +
			"Install from the drawer installs it. The drawer must show the chart (ui_navigate opens it); the version ends " +
			"when the drawer closes or shows another object. Cogpit downloads the version's chart archive from its " +
			"repository to read its README when it does not keep it yet, with the repository's credentials for the " +
			"repository's own host only. Writes nothing to a cluster. Returns the new UI state (drawerChartVersion).",
		Enabled: func(s *engine.Session, _ chartVersionParams) error { _, err := connected(s); return err },
		Run: func(_ context.Context, s *engine.Session, p chartVersionParams) (any, error) {
			return s.SetDrawerChartVersion(p.Ref, strings.TrimSpace(p.Version))
		},
	})
}
