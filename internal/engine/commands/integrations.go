package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The question of a new cluster's integrations (D-76; engine/integrationsask.go): the window asks,
// for a connected cluster the user made no choice of integrations for, which ones Cogpit reads;
// ui_state carries it as integrationsQuestion. cluster.integrations.answer answers it (the choice
// becomes the cluster's, and with saveAsDefault the default for new clusters too),
// cluster.integrations.later puts it off for this run. Both change only Cogpit's own settings, so
// they are UI commands, and agents may run them as they may set the same with cluster.settings.set
// and preferences.set (the lead's decision, 2026-10-17): a switch only changes what Cogpit reads
// with the user's identity, never a write.

type integrationsAnswerParams struct {
	Context       string          `json:"context,omitempty" jsonschema:"context id as cluster_contexts lists it (a bare context name works when one file only defines it); empty or omitted: the connected context, which is the one the window asks about (ui_state's integrationsQuestion)"`
	Integrations  map[string]bool `json:"integrations,omitempty" jsonschema:"the integrations switched on (true) or off (false), by name: metrics-server, prometheus, alertmanager, trivy, kyverno; those left out stay as the question offers them (the default for new clusters)"`
	SaveAsDefault bool            `json:"saveAsDefault,omitempty" jsonschema:"also keep the answer as the default for new clusters (the question's checkbox Save as the default for new clusters)"`
}

type integrationsLaterParams struct {
	Context string `json:"context,omitempty" jsonschema:"context id as ui_state's integrationsQuestion names it; empty or omitted: the connected context"`
}

func registerIntegrations(g *registrar) {
	add(g, engine.Spec[integrationsAnswerParams]{
		ID: "cluster.integrations.answer", Label: "Choose Integrations", Danger: engine.DangerUI,
		Description: "Answers the window's question about a cluster's integrations (ui_state's integrationsQuestion), as its " +
			"Apply button does: starting from what applies now (the default for new clusters), the integrations named are " +
			"switched on or off and the result is the cluster's own choice; with saveAsDefault it is also the default for " +
			"new clusters. Works for a cluster not asked about too. Off, an integration is asked nothing and shows nothing; " +
			"on, Cogpit reads what the cluster runs with the user's identity and never writes to it; cluster_settings_get " +
			"lists what Cogpit found of each." + clusterSettingsOnly + " Returns the cluster's settings.",
		Enabled: func(s *engine.Session, p integrationsAnswerParams) error {
			_, err := settingsContext(s, p.Context)
			return err
		},
		Run: func(_ context.Context, s *engine.Session, p integrationsAnswerParams) (any, error) {
			on, off, err := engine.ParseIntegrationSwitches(p.Integrations)
			if err != nil {
				return nil, err
			}
			id, err := settingsContext(s, p.Context)
			if err != nil {
				return nil, err
			}
			if err := offered(s, id); err != nil {
				return nil, err
			}
			if _, err := s.AnswerIntegrations(id, on, off, p.SaveAsDefault); err != nil {
				return nil, err
			}
			return settingsView(s, id)
		},
	})

	add(g, engine.Spec[integrationsLaterParams]{
		ID: "cluster.integrations.later", Label: "Ask Later", Danger: engine.DangerUI,
		Description: "Puts off the window's question about a cluster's integrations, as its Later button does: it is not " +
			"asked again this run, and asked again at the next start's connect until it is answered. Meanwhile the default " +
			"for new clusters applies, and the cluster's Settings › Integrations (cluster_settings_set's integrations) " +
			"chooses them. Changes nothing but the window. Returns the new UI state.",
		Enabled: func(s *engine.Session, p integrationsLaterParams) error {
			id, err := settingsContext(s, p.Context)
			if err != nil {
				return err
			}
			if s.UIState().IntegrationsQuestion != id {
				return engine.ErrNoIntegrationsQuestion
			}
			return nil
		},
		Run: func(_ context.Context, s *engine.Session, p integrationsLaterParams) (any, error) {
			id, err := settingsContext(s, p.Context)
			if err != nil {
				return nil, err
			}
			if err := s.PutOffIntegrationsQuestion(id); err != nil {
				return nil, err
			}
			return s.UIState(), nil
		},
	})
}
