package commands_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// integrationsHarness connects "test" with cluster settings and preferences kept in the folders
// given (the same folders are the next start's).
func integrationsHarness(t *testing.T, settingsDir, prefsDir string) *harness {
	t.Helper()
	return newHarness(t, seed(), engine.Options{ClusterSettings: engine.NewClusterSettingsStore(settingsDir),
		Preferences: engine.NewPreferencesStore(prefsDir)})
}

// question is ui_state's integrationsQuestion as an agent reads it.
func question(t *testing.T, h *harness) string {
	t.Helper()
	var u struct {
		Question string `json:"integrationsQuestion"`
	}
	h.must(t, engine.OriginExternalAgent, "ui.state", struct{}{}, &u)
	return u.Question
}

// integrationsOf reads cluster.settings.get's integrations as an agent: name → on, and whether
// the user chose them.
func integrationsOf(t *testing.T, h *harness, ctx string) (map[string]bool, bool) {
	t.Helper()
	var v engine.ClusterSettingsView
	h.must(t, engine.OriginExternalAgent, "cluster.settings.get", map[string]any{"context": ctx}, &v)
	on := map[string]bool{}
	for _, in := range v.Integrations {
		on[string(in.Name)] = in.On
	}
	if len(on) != len(engine.Integrations) {
		t.Fatalf("%d integrations listed", len(v.Integrations))
	}
	return on, v.IntegrationsDecided
}

// The question's lifecycle (D-76): asked at the first connect of a cluster without a choice, as
// ui_state shows it; Later puts it off for the run (a reconnect does not ask again), the next start
// asks again; an answer decides the cluster, with saveAsDefault the default for new clusters too,
// which the next new cluster starts from; a decided cluster is not asked. Agents run every step:
// they change only Cogpit's own settings.
func TestIntegrationsQuestionLifecycle(t *testing.T) {
	settings, prefs := t.TempDir(), t.TempDir()
	h := integrationsHarness(t, settings, prefs)
	if q := question(t, h); q != "test" {
		t.Fatalf("question after the first connect: %q", q)
	}
	on, decided := integrationsOf(t, h, "test")
	if decided || slices.Contains(boolValues(on), false) {
		t.Fatalf("before an answer: decided %v, %v", decided, on)
	}

	// Later: gone for this run, a reconnect included; a second Later has nothing to put off.
	h.must(t, engine.OriginExternalAgent, "cluster.integrations.later", map[string]any{}, nil)
	if q := question(t, h); q != "" {
		t.Fatalf("after Later: %q", q)
	}
	if err := h.run(t, engine.OriginExternalAgent, "cluster.integrations.later", map[string]any{}, nil); !errors.Is(err, engine.ErrNoIntegrationsQuestion) &&
		!strings.Contains(errText(err), engine.ErrNoIntegrationsQuestion.Error()) {
		t.Errorf("Later twice: %v", err)
	}
	if err := h.s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if err := h.s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	if q := question(t, h); q != "" {
		t.Errorf("asked again in the same run: %q", q)
	}

	// The next start asks again, until answered.
	h2 := integrationsHarness(t, settings, prefs)
	if q := question(t, h2); q != "test" {
		t.Fatalf("the next start: %q", q)
	}
	h2.must(t, engine.OriginExternalAgent, "cluster.integrations.answer", map[string]any{"integrations": map[string]bool{"alertmanager": false}}, nil)
	if q := question(t, h2); q != "" {
		t.Errorf("after the answer: %q", q)
	}
	on, decided = integrationsOf(t, h2, "test")
	if !decided || on["alertmanager"] || !on["kyverno"] {
		t.Errorf("answered: decided %v, %v", decided, on)
	}
	if p, _ := h2.s.Preferences(); p.IntegrationsOff != 0 {
		t.Errorf("the default changed without saveAsDefault: %v", engine.IntegrationNames(p.IntegrationsOff))
	}

	// Another cluster, new: asked, prefilled with the default; answered with saveAsDefault.
	if err := h2.s.Connect(context.Background(), "other"); err != nil {
		t.Fatal(err)
	}
	if q := question(t, h2); q != "other" {
		t.Fatalf("a second new cluster: %q", q)
	}
	h2.must(t, engine.OriginGUI, "cluster.integrations.answer", map[string]any{"context": "other",
		"integrations": map[string]bool{"trivy": false, "kyverno": false}, "saveAsDefault": true}, nil)
	var pr commands.PreferencesResult
	h2.must(t, engine.OriginExternalAgent, "preferences.get", struct{}{}, &pr)
	if got := engine.IntegrationNames(pr.Preferences.IntegrationsOff); !slices.Equal(got, []string{"trivy", "kyverno"}) {
		t.Errorf("the default saved: %v", got)
	}

	// The next new cluster starts from it (the cluster settings of a third start are new, its
	// preferences the same): asked, Trivy and Kyverno off until answered.
	h3 := integrationsHarness(t, t.TempDir(), prefs)
	if q := question(t, h3); q != "test" {
		t.Fatalf("a new cluster after the default was saved: %q", q)
	}
	on, decided = integrationsOf(t, h3, "test")
	if decided || on["trivy"] || on["kyverno"] || !on["alertmanager"] || !on["prometheus"] {
		t.Errorf("the default applied: decided %v, %v", decided, on)
	}
	// A decided cluster is not asked (the first start's folder: test decided by h2).
	h4 := integrationsHarness(t, settings, prefs)
	if q := question(t, h4); q != "" {
		t.Errorf("a decided cluster asked: %q", q)
	}
}

// The demo cluster, and a session that keeps no cluster settings (the CLI's), are never asked.
func TestIntegrationsQuestionNotAsked(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	if q := question(t, h); q != "" {
		t.Errorf("asked without cluster settings: %q", q)
	}
	d := newDemoHarness(t, engine.Options{ClusterSettings: engine.NewClusterSettingsStore(t.TempDir())})
	if q := question(t, d); q != "" {
		t.Errorf("the demo cluster asked: %q", q)
	}
}

// cluster.settings.set's integrations and preferences.set's integrationDefaults: by name, unknown
// names refused with the known ones, the metrics switches shown as the Metrics source; not
// together with a metricsSource that names the same providers. Agents switch both ways.
func TestIntegrationSwitchCommands(t *testing.T) {
	h := integrationsHarness(t, t.TempDir(), t.TempDir())
	var v engine.ClusterSettingsView
	h.must(t, engine.OriginExternalAgent, "cluster.settings.set", map[string]any{"integrations": map[string]bool{"prometheus": false, "trivy": false}}, &v)
	if v.MetricsSource != "metrics-server" || !v.IntegrationsDecided {
		t.Errorf("after switching Prometheus off: source %q, decided %v", v.MetricsSource, v.IntegrationsDecided)
	}
	h.must(t, engine.OriginExternalAgent, "cluster.settings.set", map[string]any{"integrations": map[string]bool{"prometheus": true, "trivy": true}}, &v)
	if v.MetricsSource != "auto" {
		t.Errorf("an agent switched Prometheus on again: source %q", v.MetricsSource)
	}
	for _, in := range v.Integrations {
		if !in.On {
			t.Errorf("%s off", in.Name)
		}
	}
	err := h.run(t, engine.OriginExternalAgent, "cluster.settings.set", map[string]any{"integrations": map[string]bool{"falco": true}}, nil)
	if err == nil || !strings.Contains(err.Error(), "alertmanager") {
		t.Errorf("an unknown integration: %v", err)
	}
	err = h.run(t, engine.OriginExternalAgent, "cluster.settings.set", map[string]any{"metricsSource": "none", "integrations": map[string]bool{"prometheus": true}}, nil)
	if err == nil {
		t.Error("metricsSource and the metrics integrations together were accepted")
	}

	var pr commands.PreferencesResult
	h.must(t, engine.OriginExternalAgent, "preferences.set", map[string]any{"integrationDefaults": map[string]bool{"alertmanager": false, "kyverno": false}}, &pr)
	h.must(t, engine.OriginExternalAgent, "preferences.set", map[string]any{"integrationDefaults": map[string]bool{"kyverno": true}}, &pr)
	if got := engine.IntegrationNames(pr.Preferences.IntegrationsOff); !slices.Equal(got, []string{"alertmanager"}) {
		t.Errorf("the default for new clusters: %v", got)
	}
	if err := h.run(t, engine.OriginExternalAgent, "preferences.set", map[string]any{"integrationDefaults": map[string]bool{"falco": false}}, nil); err == nil {
		t.Error("an unknown integration in the default was accepted")
	}
}

func boolValues(m map[string]bool) []bool {
	out := make([]bool, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

func errText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// newDemoHarness connects "test" as a demo cluster (Backend.Demo, Connection.Demo).
func newDemoHarness(t *testing.T, opts engine.Options) *harness {
	t.Helper()
	h := &harness{src: seed(), writer: &fakeWriter{}}
	opts.Connector = &fakeConnector{src: h.src, writer: h.writer, demo: true}
	h.s = engine.New(opts)
	if err := commands.RegisterAll(h.s.Registry()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.s.Close)
	if err := h.s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	return h
}
