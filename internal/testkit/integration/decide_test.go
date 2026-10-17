package integration_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// decideIntegrations makes a context's choice of integrations "all on", as a cluster the user
// answered the question for: a session that keeps cluster settings reads none of an undecided
// cluster's integrations (D-82), so the tests that read its metrics, alerts, image ratings or
// policy marks choose them first.
func decideIntegrations(t testing.TB, s *engine.Session, context string) {
	t.Helper()
	if _, err := s.ChangeClusterSettings(context, func(cs *engine.ClusterSettings) error {
		none := engine.IntegrationSet(0)
		cs.IntegrationsOff = &none
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}
