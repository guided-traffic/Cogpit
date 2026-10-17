package fyneui

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// integrationsFixture is what Cogpit found of kind-dev's integrations, a mix of every kind of
// answer: the Metrics Server found, Prometheus not found by name (its address to be set),
// Alertmanager found, no Trivy Operator, Kyverno's reports not readable. Synthetic, shaped like a
// real cluster's.
var integrationsFixture = []engine.IntegrationStatus{
	{Name: engine.IntegrationMetricsServer, State: engine.IntegrationFound},
	{Name: engine.IntegrationPrometheus, State: engine.IntegrationManual, Detail: "none found by name"},
	{Name: engine.IntegrationAlertmanager, State: engine.IntegrationFound, Where: "monitoring/kube-prometheus-alertmanager:9093"},
	{Name: engine.IntegrationTrivy, State: engine.IntegrationMissing, Detail: "aquasecurity.github.io is not served (no Trivy Operator)"},
	{Name: engine.IntegrationKyverno, State: engine.IntegrationForbidden, Detail: "the current user may not list policyreports"},
}

// integrationsFake is a fake connected to kind-dev that answers the integration commands as the
// engine does: cluster.settings.get with the fixture's statuses (a switch off says "off"), the
// switches applied to the UI state, an answer and Later ending the question.
type integrationsFake struct {
	*fakeEngine
	mu       sync.Mutex
	statuses []engine.IntegrationStatus
}

func newIntegrationsFake(t *testing.T) *integrationsFake {
	t.Helper()
	f := connectedFake(t, 20)
	f.ui.Context, f.ui.Connection = "kind-dev", engine.Connection{State: engine.Connected}
	f.ui.Preferences = &engine.Preferences{}
	f.contexts = append(f.contexts, engine.ContextInfo{Name: "kind-dev"})
	inf := &integrationsFake{fakeEngine: f, statuses: slices.Clone(integrationsFixture)}
	f.handlers["cluster.settings.get"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p clusterSettingsContext
		_ = json.Unmarshal(normalize(inv.Params), &p)
		u := f.UIState()
		inf.mu.Lock()
		defer inf.mu.Unlock()
		v := engine.ClusterSettingsView{Context: p.Context, Name: p.Context, ContextName: p.Context, Connected: true,
			IntegrationsDecided: u.IntegrationsDecided(p.Context), MetricsSource: u.MetricsSourceOf(p.Context)}
		for _, st := range inf.statuses {
			in, _ := engine.IntegrationOf(st.Name)
			st.Label, st.Adds, st.On = in.Label, in.Adds, u.IntegrationOn(p.Context, st.Name)
			if !st.On {
				st.State, st.Where, st.Detail = engine.IntegrationOff, "", ""
			}
			v.Integrations = append(v.Integrations, st)
		}
		return v, nil
	}
	f.handlers["cluster.settings.set"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p struct {
			Context      string          `json:"context"`
			Integrations map[string]bool `json:"integrations"`
		}
		if err := json.Unmarshal(normalize(inv.Params), &p); err != nil {
			return nil, err
		}
		inf.setIntegrations(p.Context, p.Integrations)
		return nil, nil
	}
	f.handlers["cluster.integrations.answer"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p struct {
			Context       string          `json:"context"`
			Integrations  map[string]bool `json:"integrations"`
			SaveAsDefault bool            `json:"saveAsDefault"`
		}
		if err := json.Unmarshal(normalize(inv.Params), &p); err != nil {
			return nil, err
		}
		inf.setIntegrations(p.Context, p.Integrations)
		return nil, nil
	}
	f.handlers["cluster.integrations.later"] = func(context.Context, engine.Invocation) (any, error) {
		f.update(func(u *engine.UIState) { u.IntegrationsQuestion = "" })
		return nil, nil
	}
	f.handlers["cluster.settings.open"] = func(context.Context, engine.Invocation) (any, error) { return nil, nil }
	return inf
}

// setIntegrations applies switches to a context's settings as the engine does, which ends the
// question about it.
func (inf *integrationsFake) setIntegrations(ctx string, switches map[string]bool) {
	on, off, _ := engine.ParseIntegrationSwitches(switches)
	inf.update(func(u *engine.UIState) {
		if u.ClusterSettings == nil {
			u.ClusterSettings = map[string]engine.ClusterSettings{}
		}
		cs := u.ClusterSettings[ctx]
		cs.SetIntegrations(u.Preferences.IntegrationsOff, on, off)
		u.ClusterSettings[ctx] = cs
		if u.IntegrationsQuestion == ctx {
			u.IntegrationsQuestion = ""
		}
	})
}

// integrationRowOf is the section's row of an integration.
func integrationRowOf(t *testing.T, rows []*integrationRow, name engine.Integration) *integrationRow {
	t.Helper()
	for _, r := range rows {
		if r.name == name {
			return r
		}
	}
	t.Fatalf("no row for %s", name)
	return nil
}

// openIntegrationsSection shows kind-dev's Settings › Integrations and waits for the statuses.
func openIntegrationsSection(t *testing.T, mode ThemeMode) (*shell, fyne.Window, *mainQueue, *integrationsFake) {
	t.Helper()
	inf := newIntegrationsFake(t)
	sh, w, q := newTestShell(t, inf.fakeEngine, Options{Theme: mode})
	inf.update(func(u *engine.UIState) {
		u.Settings = &engine.SettingsPage{Page: engine.SettingsCluster, Context: "kind-dev", Section: engine.SectionIntegrations}
	})
	q.until(t, "the integrations' statuses", func() bool {
		rows := sh.settings.integrationRows
		return len(rows) == len(engine.Integrations) && rows[len(rows)-1].state == engine.IntegrationForbidden
	})
	return sh, w, q, inf
}

// Cluster Settings › Integrations (D-76, I1): a row per integration, its switch, whether Cogpit
// found it automatically (and where), needs a manual configuration, or it is not in the cluster, in
// the status colours; Prometheus's link to where its address is set; a switch sets the cluster's
// own choice at once, and the note about the default goes; a detection that settled reads the
// statuses again.
func TestIntegrationsSection(t *testing.T) {
	sh, _, q, inf := openIntegrationsSection(t, ThemeDark)
	p := sh.settings
	want := map[engine.Integration]struct {
		text string
		c    fyne.ThemeColorName
	}{
		engine.IntegrationMetricsServer: {"Found automatically", colorStatusOK},
		engine.IntegrationPrometheus:    {"Needs manual configuration: none found by name", colorStatusWarning},
		engine.IntegrationAlertmanager:  {"Found automatically: monitoring/kube-prometheus-alertmanager:9093", colorStatusOK},
		engine.IntegrationTrivy:         {"Not in this cluster: aquasecurity.github.io is not served (no Trivy Operator)", colorMuted},
		engine.IntegrationKyverno:       {"No permission: the current user may not list policyreports", colorStatusError},
	}
	for name, w := range want {
		r := integrationRowOf(t, p.integrationRows, name)
		if r.status.text != w.text || r.status.color != w.c || !r.check.Checked {
			t.Errorf("%s: %q in %s, on %v", name, r.status.text, w.c, r.check.Checked)
		}
		if in, _ := engine.IntegrationOf(name); r.check.Text != in.Label || r.adds.text != in.Adds {
			t.Errorf("%s: label %q, adds %q", name, r.check.Text, r.adds.text)
		}
	}
	prom := integrationRowOf(t, p.integrationRows, engine.IntegrationPrometheus)
	if prom.link == nil || !prom.link.Visible() {
		t.Fatal("no link to the Prometheus address on a manual Prometheus")
	}
	if note := p.controls["integrationsNote"]; !note.Visible() {
		t.Error("the note about the default is hidden before a choice")
	}

	// The user switches Alertmanager off: cluster.settings.set at once, the cluster decided.
	test.Tap(integrationRowOf(t, p.integrationRows, engine.IntegrationAlertmanager).check)
	q.until(t, "Alertmanager off", func() bool {
		r := integrationRowOf(t, p.integrationRows, engine.IntegrationAlertmanager)
		return !r.check.Checked && r.state == engine.IntegrationOff && !p.controls["integrationsNote"].Visible()
	})
	calls := inf.callsOf("cluster.settings.set")
	if len(calls) != 1 || string(normalize(calls[0].Params)) != `{"context":"kind-dev","integrations":{"alertmanager":false}}` {
		t.Fatalf("set: %d calls, %s", len(calls), calls[0].Params)
	}

	// The link opens the Metrics section, where the Prometheus's address is set.
	test.Tap(prom.link)
	q.until(t, "the Metrics section asked for", func() bool { return len(inf.callsOf("cluster.settings.open")) == 1 })
	if c := inf.callsOf("cluster.settings.open")[0]; string(normalize(c.Params)) != `{"context":"kind-dev","section":"metrics"}` {
		t.Errorf("open: %s", c.Params)
	}

	// A detection settled (EventIntegrations): read again, the new status shows.
	inf.mu.Lock()
	inf.statuses[1] = engine.IntegrationStatus{Name: engine.IntegrationPrometheus, State: engine.IntegrationFound, Where: "monitoring/prometheus-operated:9090"}
	inf.mu.Unlock()
	inf.emit(engine.Event{Type: engine.EventIntegrations, View: "kind-dev"})
	q.until(t, "Prometheus found", func() bool {
		r := integrationRowOf(t, p.integrationRows, engine.IntegrationPrometheus)
		return r.status.text == "Found automatically: monitoring/prometheus-operated:9090" && !r.link.Visible()
	})
}

// An integration's own file puts its controls under its row (Alertmanager's address fields).
func TestIntegrationRowExtras(t *testing.T) {
	prev, had := integrationRowExtras[engine.IntegrationAlertmanager]
	t.Cleanup(func() {
		if had {
			integrationRowExtras[engine.IntegrationAlertmanager] = prev
		} else {
			delete(integrationRowExtras, engine.IntegrationAlertmanager)
		}
	})
	extra := widget.NewEntry()
	var gotCtx string
	integrationRowExtras[engine.IntegrationAlertmanager] = func(_ *settingsPage, ctx string) fyne.CanvasObject {
		gotCtx = ctx
		return extra
	}
	sh, _, _, _ := openIntegrationsSection(t, ThemeDark)
	r := integrationRowOf(t, sh.settings.integrationRows, engine.IntegrationAlertmanager)
	if gotCtx != "kind-dev" || len(r.box.Objects) != 3 || !within(r.box.Objects[2], extra) {
		t.Errorf("the extra under Alertmanager's row: context %q, %d objects", gotCtx, len(r.box.Objects))
	}
}

// within reports whether o is in the tree of containers under root.
func within(root, o fyne.CanvasObject) bool {
	if root == o {
		return true
	}
	if c, ok := root.(*fyne.Container); ok {
		return slices.ContainsFunc(c.Objects, func(x fyne.CanvasObject) bool { return within(x, o) })
	}
	return false
}

func TestRenderIntegrationsSection(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		t.Run(string(mode), func(t *testing.T) {
			_, w, _, _ := openIntegrationsSection(t, mode)
			savePNG(t, w, "cluster_settings_integrations_"+string(mode)+".png")
		})
	}
}

// openIntegrationsQuestion shows the question about kind-dev, with Trivy off by default, and waits
// for the statuses.
func openIntegrationsQuestion(t *testing.T, mode ThemeMode) (*shell, fyne.Window, *mainQueue, *integrationsFake) {
	t.Helper()
	inf := newIntegrationsFake(t)
	inf.ui.Preferences = &engine.Preferences{IntegrationsOff: engine.IntTrivy}
	inf.mu.Lock()
	inf.statuses[4] = engine.IntegrationStatus{Name: engine.IntegrationKyverno, State: engine.IntegrationChecking}
	inf.mu.Unlock()
	sh, w, q := newTestShell(t, inf.fakeEngine, Options{Theme: mode})
	inf.update(func(u *engine.UIState) { u.IntegrationsQuestion = "kind-dev" })
	q.until(t, "the question with the statuses", func() bool {
		rows := sh.intq.rows
		return sh.intq.d != nil && len(rows) == len(engine.Integrations) && rows[2].state == engine.IntegrationFound
	})
	return sh, w, q, inf
}

// The question of a new cluster's integrations (D-76, I2): the same rows, prefilled from the
// default for new clusters, the statuses live; Apply answers with the switches and the checkbox,
// and the dialog closes when the engine drops the question.
func TestIntegrationsQuestionApply(t *testing.T) {
	sh, _, q, inf := openIntegrationsQuestion(t, ThemeDark)
	iq := sh.intq
	for _, r := range iq.rows {
		if want := r.name != engine.IntegrationTrivy; r.check.Checked != want {
			t.Errorf("%s prefilled %v", r.name, r.check.Checked)
		}
	}
	if kyv := integrationRowOf(t, iq.rows, engine.IntegrationKyverno); kyv.status.text != "Checking…" || kyv.status.color != colorStatusPending {
		t.Errorf("Kyverno: %q", kyv.status.text)
	}
	if iq.save.Checked || iq.save.Text != integrationsAskSave {
		t.Errorf("the checkbox: %q, %v", iq.save.Text, iq.save.Checked)
	}
	// Kyverno's detection settles while the dialog is open.
	inf.mu.Lock()
	inf.statuses[4] = engine.IntegrationStatus{Name: engine.IntegrationKyverno, State: engine.IntegrationFound, Where: "wgpolicyk8s.io/v1alpha2"}
	inf.mu.Unlock()
	inf.emit(engine.Event{Type: engine.EventIntegrations, View: "kind-dev"})
	q.until(t, "Kyverno found", func() bool {
		return integrationRowOf(t, iq.rows, engine.IntegrationKyverno).status.text == "Found automatically: wgpolicyk8s.io/v1alpha2"
	})
	if n := len(inf.callsOf("cluster.settings.set")); n != 0 {
		t.Fatalf("%d settings changed before Apply", n)
	}

	trivy := integrationRowOf(t, iq.rows, engine.IntegrationTrivy)
	if trivy.status.text != "Off: nothing is asked of the cluster" {
		t.Errorf("Trivy, off by default: %q", trivy.status.text)
	}
	test.Tap(trivy.check)
	if trivy.status.text != integrationsAskOff {
		t.Errorf("Trivy switched on in the dialog: %q", trivy.status.text)
	}
	test.Tap(trivy.check)
	test.Tap(integrationRowOf(t, iq.rows, engine.IntegrationAlertmanager).check)
	test.Tap(iq.save)
	test.Tap(iq.apply)
	q.until(t, "the dialog closed", func() bool { return iq.d == nil && iq.ctx == "" })
	calls := inf.callsOf("cluster.integrations.answer")
	want := `{"context":"kind-dev","integrations":{"alertmanager":false,"kyverno":true,"metrics-server":true,"prometheus":true,"trivy":false},"saveAsDefault":true}`
	if len(calls) != 1 || string(normalize(calls[0].Params)) != want {
		t.Fatalf("answer: %d calls, %s", len(calls), calls[0].Params)
	}
	if len(inf.callsOf("cluster.integrations.later")) != 0 {
		t.Error("closing after the answer put it off")
	}
}

// Later puts it off; an agent's answer closes the open dialog without a Later.
func TestIntegrationsQuestionLaterAndAgent(t *testing.T) {
	sh, _, q, inf := openIntegrationsQuestion(t, ThemeLight)
	test.Tap(sh.intq.later)
	q.until(t, "put off", func() bool { return sh.intq.d == nil })
	if n := len(inf.callsOf("cluster.integrations.later")); n != 1 {
		t.Fatalf("%d Later calls", n)
	}

	inf.update(func(u *engine.UIState) { u.IntegrationsQuestion = "kind-dev" })
	q.until(t, "asked again", func() bool { return sh.intq.d != nil })
	inf.setIntegrations("kind-dev", map[string]bool{"trivy": true}) // an agent answered
	q.until(t, "closed by the agent's answer", func() bool { return sh.intq.d == nil })
	if n := len(inf.callsOf("cluster.integrations.later")); n != 1 {
		t.Errorf("%d Later calls after an agent's answer", n)
	}
}

func TestRenderIntegrationsQuestion(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		t.Run(string(mode), func(t *testing.T) {
			_, w, _, _ := openIntegrationsQuestion(t, mode)
			savePNG(t, w, "integrations_question_"+string(mode)+".png")
		})
	}
}

// Preferences › Kubernetes › Integrations of new clusters: the default, switched by the user and
// shown as an agent sets it.
func TestNewClusterIntegrationsPreference(t *testing.T) {
	s, _ := settingsEngine(t)
	sh, _, q := newEngineShell(t, s, Options{})
	mustDispatch(t, s, "preferences.open", map[string]any{"section": "kubernetes"})
	q.until(t, "the Kubernetes section", func() bool { return sh.settings.controls[integrationDefaultsSetting] != nil })
	checks := sh.settings.controls[integrationDefaultsSetting].(*fyne.Container).Objects
	if len(checks) != len(engine.Integrations) {
		t.Fatalf("%d checks", len(checks))
	}
	for _, o := range checks {
		if !o.(*widget.Check).Checked {
			t.Errorf("%s off by default", o.(*widget.Check).Text)
		}
	}
	test.Tap(checks[2].(*widget.Check)) // Alertmanager
	q.until(t, "Alertmanager off by default", func() bool {
		p, _ := s.Preferences()
		return p.IntegrationsOff == engine.IntAlertmanager
	})
	mustDispatch(t, s, "preferences.set", map[string]any{"integrationDefaults": map[string]bool{"kyverno": false}})
	q.until(t, "the agent's change shown", func() bool { return !checks[4].(*widget.Check).Checked && !checks[2].(*widget.Check).Checked })
}

// The section over the real engine and the demo cluster: the metrics integrations' statuses from
// the hub (the demo's metrics its own: no Prometheus), the demo never asked; a switch is the
// cluster's choice and the Metrics source follows.
func TestIntegrationsSectionOnTheDemo(t *testing.T) {
	s, sh, _, until := demoSettingsShell(t)
	mustDispatch(t, s, "cluster.settings.open", map[string]any{"context": engine.DemoContext, "section": "integrations"})
	until("the metrics integrations' statuses", func() bool {
		rows := sh.settings.integrationRows
		return len(rows) == len(engine.Integrations) &&
			integrationRowOf(t, rows, engine.IntegrationMetricsServer).status.text == "Found automatically" &&
			integrationRowOf(t, rows, engine.IntegrationPrometheus).status.text == "Not in this cluster: the demo cluster's metrics are its own"
	})
	if q := s.UIState().IntegrationsQuestion; q != "" || sh.intq.d != nil {
		t.Errorf("the demo cluster asked: %q", q)
	}
	test.Tap(integrationRowOf(t, sh.settings.integrationRows, engine.IntegrationMetricsServer).check)
	until("the Metrics Server off", func() bool {
		u := s.UIState()
		return u.IntegrationsDecided(engine.DemoContext) && u.MetricsSourceOf(engine.DemoContext) == "prometheus" &&
			integrationRowOf(t, sh.settings.integrationRows, engine.IntegrationMetricsServer).state == engine.IntegrationOff
	})
}
