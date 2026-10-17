package engine

import (
	"encoding/json"
	"testing"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The integration settings (D-76): names and JSON, the default for new clusters, the user's choice
// over it, and the Metrics source kept before D-76 moved into the two metrics switches.
func TestIntegrationSettings(t *testing.T) {
	if set, err := ParseIntegrationSet([]string{" Kyverno ", "prometheus"}); err != nil || set != IntKyverno|IntPrometheus {
		t.Errorf("ParseIntegrationSet = %v, %v", set, err)
	}
	if _, err := ParseIntegrationSet([]string{"falco"}); err == nil {
		t.Error("an unknown integration was accepted")
	}
	b, _ := json.Marshal(IntTrivy | IntMetricsServer)
	if string(b) != `["metrics-server","trivy"]` {
		t.Errorf("JSON %s", b)
	}
	var back IntegrationSet
	if err := json.Unmarshal([]byte(`["trivy","falco"]`), &back); err != nil || back != IntTrivy {
		t.Errorf("unmarshal kept %v, %v", back, err)
	}

	// Undecided: the default for new clusters applies; every integration is on without one.
	u := &UIState{}
	for _, in := range Integrations {
		if !u.IntegrationOn("ctx", in.Name) {
			t.Errorf("%s is off by default", in.Name)
		}
	}
	u.Preferences = &Preferences{IntegrationsOff: IntAlertmanager}
	if u.IntegrationOn("ctx", IntegrationAlertmanager) || u.IntegrationsDecided("ctx") {
		t.Error("the default for new clusters does not apply to an undecided cluster")
	}
	// Decided: the cluster's own choice, even all on.
	none := IntegrationSet(0)
	u.ClusterSettings = map[string]ClusterSettings{"ctx": {IntegrationsOff: &none}}
	if !u.IntegrationOn("ctx", IntegrationAlertmanager) || !u.IntegrationsDecided("ctx") {
		t.Error("the cluster's choice does not win over the default")
	}
	if (ClusterSettings{IntegrationsOff: &none}).IsZero() {
		t.Error("a choice of all on is kept as no choice")
	}

	// SetIntegrations starts from what applies now.
	var cs ClusterSettings
	cs.SetIntegrations(IntTrivy|IntKyverno, IntKyverno, IntPrometheus)
	if *cs.IntegrationsOff != IntTrivy|IntPrometheus {
		t.Errorf("SetIntegrations from the default: %v", IntegrationNames(*cs.IntegrationsOff))
	}
	cs.SetMetricsSource(0, "")
	if *cs.IntegrationsOff != IntTrivy {
		t.Errorf("Automatic: %v", IntegrationNames(*cs.IntegrationsOff))
	}

	// The Metrics source kept before D-76.
	for source, want := range map[string]IntegrationSet{"prometheus": IntMetricsServer, "metrics-server": IntPrometheus, "none": IntMetrics} {
		old := ClusterSettings{MetricsSource: source}
		old.normalize()
		if old.MetricsSource != "" || old.IntegrationsOff == nil || *old.IntegrationsOff != want || old.Check() != nil {
			t.Errorf("%s moved to %+v", source, old)
		}
		u := &UIState{ClusterSettings: map[string]ClusterSettings{"ctx": old}}
		if got := metricsSourceOf(u, "ctx"); got != metrics.Source(source) {
			t.Errorf("%s reads back as %q", source, got)
		}
	}
}

// IntegrationStatuses: a context not connected says so, with its switches.
func TestIntegrationStatusesNotConnected(t *testing.T) {
	s := New(Options{})
	defer s.Close()
	got := s.IntegrationStatuses("ctx")
	if len(got) != len(Integrations) {
		t.Fatalf("%d statuses", len(got))
	}
	for _, st := range got {
		if !st.On || st.State != IntegrationNotConnected || st.Label == "" || st.Adds == "" {
			t.Errorf("%+v", st)
		}
	}
}
