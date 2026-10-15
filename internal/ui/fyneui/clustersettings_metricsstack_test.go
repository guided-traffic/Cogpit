package fyneui

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"

	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/helm"
)

// stackFake is a fake engine connected to a real-looking cluster whose Cogpit Metrics the test
// holds: cluster.metricsStack answers the state (and the plan of switches given), and
// cluster.setMetricsStack installs and removes in it.
type stackFake struct {
	*fakeEngine
	mu      sync.Mutex
	st      engine.MetricsStackState
	applied []map[string]bool
}

func newStackFake(t *testing.T) *stackFake {
	t.Helper()
	f := &stackFake{fakeEngine: connectedFake(t, 20)}
	f.ui.Context = "kind-dev"
	f.ui.Connection = engine.Connection{State: engine.Connected}
	f.st = engine.MetricsStackState{Context: "kind-dev", Namespace: helm.MetricsNamespace, Known: true,
		Chart: "prometheus " + helm.MetricsChartVersion, Repository: helm.MetricsChartRepository,
		Storage: helm.MetricsStorage{StorageClass: "standard", ReclaimPolicy: "Delete"}}
	f.st.StorageText = f.st.Storage.String()
	for _, c := range helm.MetricsComponents {
		f.st.Components = append(f.st.Components, engine.MetricsStackComponent{ID: string(c), Label: c.Label(), Release: c.Release()})
	}
	f.handlers["cluster.settings.get"] = func(context.Context, engine.Invocation) (any, error) {
		return engine.ClusterSettingsView{Context: "kind-dev", Name: "kind-dev", ContextName: "kind-dev", Connected: true}, nil
	}
	f.handlers["cluster.metricsStack"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		want := map[string]bool{}
		_ = json.Unmarshal(normalize(inv.Params), &want)
		f.mu.Lock()
		defer f.mu.Unlock()
		view := map[string]any{}
		raw, _ := json.Marshal(f.st)
		_ = json.Unmarshal(raw, &view)
		if len(want) > 0 { // the engine's sentence, shortened (commands.planDetail)
			plan := map[string]any{}
			detail := "Installs Prometheus (release cogpit-prometheus) in namespace cogpit-metrics from chart prometheus " +
				helm.MetricsChartVersion + " that Cogpit bundles, with your identity. Cluster-wide it creates ClusterRole " +
				"cogpit-prometheus-server, ClusterRoleBinding cogpit-prometheus-server."
			if want["nodeExporter"] {
				detail = "Installs node-exporter (release cogpit-node-exporter) in namespace cogpit-metrics from chart prometheus " +
					helm.MetricsChartVersion + " that Cogpit bundles, with your identity. node-exporter runs a pod on every node with " +
					"the host's network and process namespace and its root filesystem read-only, on port 9100 of each node."
			}
			plan["detail"] = detail
			for _, on := range want {
				if !on {
					plan["destructive"] = true
				}
			}
			view["plan"] = plan
		}
		return view, nil
	}
	f.handlers["cluster.setMetricsStack"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		want := map[string]bool{}
		_ = json.Unmarshal(normalize(inv.Params), &want)
		f.mu.Lock()
		defer f.mu.Unlock()
		f.applied = append(f.applied, want)
		for i, c := range f.st.Components {
			if on, ok := want[metricsStackParamNames[c.ID]]; ok {
				f.st.Components[i].Installed = on
				f.st.Components[i].Status, f.st.Components[i].ChartVersion, f.st.Components[i].Revision = "", "", 0
				if on {
					f.st.Components[i].Status, f.st.Components[i].ChartVersion, f.st.Components[i].Revision = "deployed", helm.MetricsChartVersion, 1
				}
			}
		}
		f.st.NamespaceExists, f.st.NamespaceCogpits = true, true
		return nil, nil
	}
	return f
}

func (f *stackFake) appliedSoFar() []map[string]bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.applied)
}

func (f *stackFake) openSection() {
	f.update(func(u *engine.UIState) {
		u.Settings = &engine.SettingsPage{Page: engine.SettingsCluster, Context: "kind-dev", Section: engine.SectionMetricsStack}
	})
}

func stackCheck(sh *shell, id string) *widget.Check {
	c, _ := sh.settings.controls["metricsStack."+id].(*widget.Check)
	return c
}

// Cluster Settings › Cogpit Metrics: three switches at what is installed, Apply greyed until one
// differs; Apply shows an install's sentence in a dialog first, then installs; the switch follows
// the release; switching it off removes it without the window's dialog (the engine confirms a
// removal); a change the user has not applied survives a refresh of the state.
func TestCogpitMetricsSection(t *testing.T) {
	f := newStackFake(t)
	sh, w, q := newTestShell(t, f.fakeEngine, Options{})
	f.openSection()
	q.until(t, "the section", func() bool { return stackCheck(sh, "prometheus") != nil && sh.settings.stack.state != nil })
	var labels []string
	for _, o := range sh.settings.nav.Objects {
		labels = append(labels, o.(*widget.Button).Text)
	}
	if !slices.Equal(labels, []string{"General", "Proxy", "Terminal", "Namespaces", "Metrics", "Cogpit Metrics", "Node Shell", "Agents"}) {
		t.Errorf("sections %q", labels)
	}
	apply := sh.settings.controls["metricsStackApply"].(*widget.Button)
	for _, id := range []string{"prometheus", "kube-state-metrics", "node-exporter"} {
		if c := stackCheck(sh, id); c.Checked || c.Disabled() {
			t.Errorf("%s: checked %v, disabled %v", id, c.Checked, c.Disabled())
		}
	}
	if !apply.Disabled() || !strings.Contains(sh.settings.controls["metricsStackStorage"].(*textLine).text, "StorageClass standard") {
		t.Fatalf("apply disabled %v, storage %q", apply.Disabled(), sh.settings.controls["metricsStackStorage"].(*textLine).text)
	}

	stackCheck(sh, "prometheus").SetChecked(true)
	if apply.Disabled() {
		t.Fatal("Apply stays greyed with a switch changed")
	}
	// A refresh of the state (any change of the window's) keeps the change.
	sh.settings.stack.request()
	q.until(t, "the state again", func() bool { return len(f.callsOf("cluster.metricsStack")) >= 2 })
	q.drain()
	if !stackCheck(sh, "prometheus").Checked || apply.Disabled() {
		t.Error("a refresh lost the change")
	}

	test.Tap(apply)
	q.until(t, "the install's dialog", func() bool { return findButton(w, "Install") != nil })
	if texts := overlayTexts(w); !slices.ContainsFunc(texts, func(s string) bool { return strings.Contains(s, "ClusterRole cogpit-prometheus-server") }) {
		t.Errorf("the dialog does not say what is created cluster-wide: %q", texts)
	}
	test.Tap(findButton(w, "Install"))
	q.until(t, "the install", func() bool { return len(f.appliedSoFar()) == 1 })
	if a := f.appliedSoFar()[0]; len(a) != 1 || !a["prometheus"] {
		t.Errorf("applied %v", a)
	}
	q.until(t, "the installed state", func() bool {
		l := sh.settings.stack.lines["prometheus"]
		return strings.HasPrefix(l.text, "installed: release cogpit-prometheus, deployed, chart "+helm.MetricsChartVersion)
	})
	if !apply.Disabled() || !stackCheck(sh, "prometheus").Checked {
		t.Error("after the install: Apply enabled or the switch off")
	}

	// Off: the removal goes to the engine, which confirms it; no dialog of the window's.
	stackCheck(sh, "prometheus").SetChecked(false)
	test.Tap(apply)
	q.until(t, "the removal", func() bool { return len(f.appliedSoFar()) == 2 })
	if a := f.appliedSoFar()[1]; len(a) != 1 || a["prometheus"] {
		t.Errorf("applied %v", a)
	}
	if findButton(w, "Install") != nil {
		t.Error("the window asked before a removal the engine confirms")
	}
}

// Cogpit Metrics on the demo cluster: nothing to install into, said so.
func TestCogpitMetricsSectionOnTheDemo(t *testing.T) {
	f := connectedFake(t, 20)
	f.handlers["cluster.settings.get"] = func(context.Context, engine.Invocation) (any, error) {
		return engine.ClusterSettingsView{Context: "demo", Name: "demo", ContextName: "demo", Connected: true}, nil
	}
	sh, _, q := newTestShell(t, f, Options{})
	f.update(func(u *engine.UIState) {
		u.Settings = &engine.SettingsPage{Page: engine.SettingsCluster, Context: "demo", Section: engine.SectionMetricsStack}
	})
	q.until(t, "the section", func() bool {
		n, ok := sh.settings.controls["metricsStackNote"].(*textLine)
		return ok && strings.Contains(n.text, "demo cluster")
	})
	if !stackCheck(sh, "prometheus").Disabled() || len(f.callsOf("cluster.metricsStack")) != 0 {
		t.Error("the demo's switches are enabled, or its stack was asked for")
	}
}

// The section with Prometheus installed and node-exporter switched on, and the install's dialog,
// dark and light: the PNGs of the pull request (AGENTS.md rule 7).
func TestRenderCogpitMetrics(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		t.Run(string(mode), func(t *testing.T) {
			f := newStackFake(t)
			f.st.Components[0].Installed, f.st.Components[0].Status, f.st.Components[0].ChartVersion = true, "deployed", helm.MetricsChartVersion
			f.st.NamespaceExists, f.st.NamespaceCogpits = true, true
			sh, w, q := newTestShell(t, f.fakeEngine, Options{Theme: mode})
			f.openSection()
			q.until(t, "the section", func() bool {
				l := sh.settings.stack
				return l != nil && l.state != nil && stackCheck(sh, "prometheus").Checked
			})
			stackCheck(sh, "node-exporter").SetChecked(true)
			q.drain()
			savePNG(t, w, "cluster_settings_cogpit_metrics_"+string(mode)+".png")
			test.Tap(sh.settings.controls["metricsStackApply"].(*widget.Button))
			q.until(t, "the dialog", func() bool { return findButton(w, "Install") != nil })
			q.drain()
			savePNG(t, w, "cluster_settings_cogpit_metrics_install_"+string(mode)+".png")
		})
	}
}
