package fyneui

import (
	"context"
	"encoding/json"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The status line under the drawer's charts and each chart's variant as engine state (G12, owner
// 2026-10-12, evening). The providers' states are canned as the engine reports them.

// bothAnswering is the normal case: Prometheus gives the history (and the network and the
// filesystem), the Metrics Server the live values.
func bothAnswering() []metrics.ProviderStatus {
	return []metrics.ProviderStatus{
		{Provider: metrics.PrometheusID, Label: metrics.PrometheusLabel, Service: "monitoring/prometheus-server:http", Status: metrics.StatusAvailable,
			LastOK: fixedNow.Add(-40 * time.Second), Serves: []string{metrics.ServesHistory, metrics.ServesNetwork, metrics.ServesFilesystem}},
		{Provider: metrics.ServerID, Label: metrics.ServerLabel, Status: metrics.StatusAvailable,
			LastOK: fixedNow.Add(-5 * time.Second), Serves: []string{metrics.ServesLive}},
	}
}

// serverDown is the fallback: the Metrics Server stopped answering, Prometheus serves the live
// values too.
func serverDown() []metrics.ProviderStatus {
	ps := bothAnswering()
	ps[0].Serves = []string{metrics.ServesHistory, metrics.ServesLive, metrics.ServesNetwork, metrics.ServesFilesystem}
	ps[0].LastOK = fixedNow.Add(-10 * time.Second)
	ps[1] = metrics.ProviderStatus{Provider: metrics.ServerID, Label: metrics.ServerLabel, Status: metrics.StatusUnavailable,
		LastOK: fixedNow.Add(-3 * time.Minute), LastError: "unavailable (HTTP 503)", LastErrorAt: fixedNow.Add(-25 * time.Second)}
	return ps
}

// prometheusGone: Prometheus was not found; the Metrics Server alone, without history.
func prometheusGone() []metrics.ProviderStatus {
	ps := bothAnswering()
	ps[0] = metrics.ProviderStatus{Provider: metrics.PrometheusID, Label: metrics.PrometheusLabel, Status: metrics.StatusNotServed,
		LastError: "not found (HTTP 404)", LastErrorAt: fixedNow.Add(-2 * time.Minute)}
	return ps
}

func withProviders(ps []metrics.ProviderStatus) func(*engine.ObjectMetrics) {
	return func(m *engine.ObjectMetrics) { m.Providers = ps }
}

// withVariantCommand answers ui.setMetricsVariant as the engine does: the chart's variant changes
// in what ObjectMetrics returns, and EventMetrics follows.
func withVariantCommand(f *fakeEngine) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers["ui.setMetricsVariant"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p struct {
			Ref       model.ObjectRef `json:"ref"`
			Container string          `json:"container"`
			Variant   string          `json:"variant"`
		}
		if err := json.Unmarshal(normalize(inv.Params), &p); err != nil {
			return nil, err
		}
		f.setVariant(p.Ref, p.Container, p.Variant)
		return nil, nil
	}
}

// setVariant changes a chart's variant as the engine holds it (the GUI's command or an agent's) and
// reports it.
func (f *fakeEngine) setVariant(ref model.ObjectRef, container, variant string) {
	f.mu.Lock()
	old := f.metrics[ref.String()]
	if old == nil {
		f.mu.Unlock()
		return
	}
	m := *old
	m.Targets = slices.Clone(old.Targets)
	for i := range m.Targets {
		if m.Targets[i].Name == container {
			m.Targets[i].Variant = variant
		}
	}
	f.metrics[ref.String()] = &m
	f.mu.Unlock()
	f.emit(engine.Event{Type: engine.EventMetrics, View: ref.Kind})
}

func statusTexts(p *metricsPanel) []string {
	var out []string
	for _, r := range p.status.rows {
		if r.Visible() {
			out = append(out, r.text.text)
		}
	}
	return out
}

func TestProviderStatusLine(t *testing.T) {
	sh, f, q, p := promDrawer(t, ThemeDark, withProviders(bothAnswering()))
	got := statusTexts(p)
	want := []string{
		"Prometheus monitoring/prometheus-server:http: history, network, filesystem · 40 s ago",
		"Kubernetes Metrics Server: live values · 5 s ago",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("status line = %q, want %q", got, want)
	}
	if r := p.status.rows[0]; r.line.color != colorStatusOK || !strings.Contains(r.line.tip, "Found and answering.") || !strings.Contains(r.line.tip, "Last good answer: 14:29:20 (40 s ago)") {
		t.Errorf("Prometheus's row: %q in %q", r.line.tip, r.line.color)
	}
	// Hovering a row shows its whole state.
	row := p.status.rows[1]
	row.MouseIn(&desktop.MouseEvent{PointEvent: fyne.PointEvent{AbsolutePosition: fyne.NewPos(10, 10)}})
	q.until(t, "the tooltip", func() bool { return sh.tip.visibleText() != "" })
	if tip := sh.tip.visibleText(); !strings.Contains(tip, "Kubernetes Metrics Server") || !strings.Contains(tip, "Serves: live values.") {
		t.Errorf("tooltip = %q", tip)
	}
	row.MouseOut()

	// The Metrics Server stops answering: the line says so, with the failure's class, and that
	// Prometheus serves the live values now; the tooltip has the last error's time.
	f.mu.Lock()
	m := *f.metrics[p.ref.String()]
	m.Providers = serverDown()
	f.metrics[p.ref.String()] = &m
	f.mu.Unlock()
	f.emit(engine.Event{Type: engine.EventMetrics, View: "pods"})
	q.until(t, "the fallback on the status line", func() bool {
		texts := statusTexts(p)
		return len(texts) == 2 && strings.Contains(texts[1], "not answering")
	})
	got = statusTexts(p)
	if got[0] != "Prometheus monitoring/prometheus-server:http: history, live values, network, filesystem · 10 s ago" ||
		got[1] != "Kubernetes Metrics Server: not answering, unavailable (HTTP 503) · last answer 3 min ago" {
		t.Errorf("status line with the Metrics Server down = %q", got)
	}
	if r := p.status.rows[1]; r.line.color != colorStatusWarning || !strings.Contains(r.line.tip, "Last error: unavailable (HTTP 503) at 14:29:35 (25 s ago)") {
		t.Errorf("the Metrics Server's row: %q in %q", r.line.tip, r.line.color)
	}

	// Prometheus not found: the Metrics Server alone, the line says Prometheus is not found.
	f.mu.Lock()
	m.Providers = prometheusGone()
	f.metrics[p.ref.String()] = &m
	f.mu.Unlock()
	f.emit(engine.Event{Type: engine.EventMetrics, View: "pods"})
	q.until(t, "Prometheus not found", func() bool {
		texts := statusTexts(p)
		return len(texts) == 2 && texts[0] == "Prometheus: not found"
	})
	if r := p.status.rows[0]; r.line.color != colorMuted || !strings.Contains(r.line.tip, "Not found in this cluster.") {
		t.Errorf("Prometheus's row: %q in %q", r.line.tip, r.line.color)
	}
}

// Every state has its icon and wording, and a status line never shows more than the class of a
// failure (the engine hands it nothing else; the line adds nothing either).
func TestProviderLineStates(t *testing.T) {
	for _, tc := range []struct {
		ps    metrics.ProviderStatus
		text  string
		color fyne.ThemeColorName
	}{
		{metrics.ProviderStatus{Label: "Prometheus", Status: metrics.StatusForbidden, LastError: "forbidden (HTTP 403)"}, "Prometheus: not allowed, forbidden (HTTP 403)", colorStatusError},
		{metrics.ProviderStatus{Label: "Prometheus", Status: metrics.StatusUnknown}, "Prometheus: not looked for yet", colorMuted},
		{metrics.ProviderStatus{Label: "Prometheus", Service: "monitoring/prometheus-server:http", Status: metrics.StatusAvailable}, "Prometheus monitoring/prometheus-server:http: answering, not needed now", colorStatusOK},
		{metrics.ProviderStatus{Label: "Kubernetes Metrics Server", Status: metrics.StatusUnavailable, LastError: "timeout"}, "Kubernetes Metrics Server: not answering, timeout", colorStatusWarning},
	} {
		l := providerLine(tc.ps, fixedNow, germanTime)
		if l.text != tc.text || l.color != tc.color || l.icon == nil || !strings.HasPrefix(l.tip, tc.ps.Label) {
			t.Errorf("%s %s: %q in %q (tip %q), want %q in %q", tc.ps.Label, tc.ps.Status, l.text, l.color, l.tip, tc.text, tc.color)
		}
	}
}

// A chart's variant is engine state: the button asks the engine (ui.setMetricsVariant), and the
// chart follows what the engine reports, whoever set it.
func TestMetricsVariantIsEngineState(t *testing.T) {
	sh, f, q, p := promDrawer(t, ThemeDark, withProviders(bothAnswering()))
	p.variants[variantNetwork].OnTapped()
	q.until(t, "the command", func() bool { return len(f.callsOf("ui.setMetricsVariant")) == 1 })
	var params struct {
		Container string `json:"container"`
		Variant   string `json:"variant"`
	}
	if err := json.Unmarshal(normalize(f.callsOf("ui.setMetricsVariant")[0].Params), &params); err != nil || params.Variant != variantNetwork || params.Container != "" {
		t.Fatalf("ui.setMetricsVariant params %+v (%v)", params, err)
	}
	q.until(t, "the engine's answer", func() bool { return !p.variantPending && p.data.Targets[0].Variant == variantNetwork })
	if p.variant != variantNetwork {
		t.Errorf("the pod's chart shows %q", p.variant)
	}
	// An agent switches the app container's chart to Memory: the window follows.
	f.setVariant(p.ref, "app", variantMemory)
	q.until(t, "the agent's variant", func() bool { c := p.perBox["app"]; return c != nil && c.variant == variantMemory })
	if c := p.perBox["app"]; c.plot.data.Series[0].Name != "Memory usage" {
		t.Errorf("the app chart draws %q", c.plot.data.Series[0].Name)
	}
	_ = sh
}

// A sample without the part a chart shows (Prometheus had the memory but no CPU rate yet) is left
// out, not drawn as 0.
func TestMissingPartIsLeftOutOfTheChart(t *testing.T) {
	tg := &engine.MetricsTarget{Samples: []model.UsageSample{
		{At: fixedNow.Add(-time.Minute), Usage: model.Usage{CPUMilli: 120, MemoryBytes: 64 << 20}},
		{At: fixedNow.Add(-30 * time.Second), Usage: model.Usage{MemoryBytes: 65 << 20, Missing: model.UsageCPU}},
		{At: fixedNow, Usage: model.Usage{CPUMilli: 130, MemoryBytes: 66 << 20}},
	}}
	cpu, _, _, _ := usageSeries(tg, true, false)
	mem, _, _, _ := usageSeries(tg, false, false)
	if len(cpu[0].Points) != 2 || len(mem[0].Points) != 3 {
		t.Errorf("cpu %d points, memory %d: the sample without CPU belongs to the memory chart only", len(cpu[0].Points), len(mem[0].Points))
	}
	for _, pt := range cpu[0].Points {
		if pt.Value == 0 {
			t.Errorf("a missing CPU value drawn as 0 at %v", pt.At)
		}
	}
}

// TestRenderMetricsStatusLine draws the status line under a pod's charts in both themes: both
// providers answering, the Metrics Server down (Prometheus serves the live values), Prometheus not
// found, and the narrowest drawer. To COGPIT_PNG_DIR or a temporary directory, never to testdata.
func TestRenderMetricsStatusLine(t *testing.T) {
	looks := []struct {
		name   string
		ps     []metrics.ProviderStatus
		narrow bool
		hover  bool
	}{
		{"both", bothAnswering(), false, false},
		{"server_down", serverDown(), false, true},
		{"prometheus_gone", prometheusGone(), false, false},
		{"narrow", serverDown(), true, false},
	}
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		for _, l := range looks {
			name := "metrics_status_" + l.name + "_" + string(mode) + ".png"
			t.Run(name, func(t *testing.T) {
				sh, _, q, p := promDrawer(t, mode, withProviders(l.ps))
				if l.narrow {
					sh.setDrawerWidth(minDrawerWidth)
					q.until(t, "the drawer to take the width", func() bool { return sh.drawer.width == minDrawerWidth })
				}
				settle(sh.win)
				if l.hover {
					r := p.status.rows[1]
					pos := fyne.CurrentApp().Driver().AbsolutePositionForObject(r)
					r.MouseIn(&desktop.MouseEvent{PointEvent: fyne.PointEvent{AbsolutePosition: pos.AddXY(40, 8)}})
					q.until(t, "the tooltip", func() bool { return sh.tip.visibleText() != "" })
					settle(sh.win)
				}
				dir := os.Getenv("COGPIT_PNG_DIR")
				if dir == "" {
					dir = t.TempDir()
				}
				f, err := os.Create(filepath.Join(dir, name))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				if err := png.Encode(f, sh.win.Canvas().Capture()); err != nil {
					t.Fatal(err)
				}
				t.Log("rendered", f.Name())
			})
		}
	}
}

// withPrometheusCommands answers cluster.prometheus with set and records cluster.setPrometheus.
func withPrometheusCommands(f *fakeEngine, set engine.PrometheusSetting) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.handlers["cluster.prometheus"] = func(context.Context, engine.Invocation) (any, error) { return set, nil }
	f.handlers["cluster.setPrometheus"] = func(context.Context, engine.Invocation) (any, error) { return set, nil }
}

// The button under the status line opens the Prometheus dialog with the engine's setting; saving
// sends the address and prefix (or auto) to cluster.setPrometheus. The demo cluster, which has no
// Prometheus to set, shows no button.
func TestPrometheusSettingsDialog(t *testing.T) {
	sh, f, q, p := promDrawer(t, ThemeDark, withProviders(bothAnswering()))
	withPrometheusCommands(f, engine.PrometheusSetting{Context: "demo", Auto: true, Kept: true})
	if !p.status.settings.Visible() {
		t.Fatal("no Prometheus button under the status line")
	}
	p.status.settings.OnTapped()
	q.until(t, "the dialog", func() bool { return findButton(sh.win, "Save") != nil })
	overlay := sh.win.Canvas().Overlays().Top()
	checks, entries := findAll[*widget.Check](overlay), findAll[*widget.Entry](overlay)
	if len(checks) != 1 || len(entries) != 2 || !checks[0].Checked || !entries[0].Disabled() {
		t.Fatalf("dialog: %d checks (auto %v), %d entries", len(checks), len(checks) > 0 && checks[0].Checked, len(entries))
	}
	if texts := overlayTexts(sh.win); !slices.Contains(texts, "Prometheus for demo") {
		t.Errorf("dialog texts %q", texts)
	}
	checks[0].SetChecked(false)
	entries[0].SetText("observability/thanos-query:9090")
	entries[1].SetText("/prometheus")
	settle(sh.win)
	findButton(sh.win, "Save").OnTapped()
	q.until(t, "cluster.setPrometheus", func() bool { return len(f.callsOf("cluster.setPrometheus")) == 1 })
	var params struct{ Address, Prefix string }
	if err := json.Unmarshal(normalize(f.callsOf("cluster.setPrometheus")[0].Params), &params); err != nil ||
		params.Address != "observability/thanos-query:9090" || params.Prefix != "/prometheus" {
		t.Errorf("params %+v (%v)", params, err)
	}

	// Auto again sends auto.
	p.status.settings.OnTapped()
	q.until(t, "the dialog again", func() bool { return findButton(sh.win, "Save") != nil })
	findButton(sh.win, "Save").OnTapped()
	q.until(t, "the second set", func() bool { return len(f.callsOf("cluster.setPrometheus")) == 2 })
	if s := string(f.callsOf("cluster.setPrometheus")[1].Params); !strings.Contains(s, `"address":"auto"`) {
		t.Errorf("auto sent %s", s)
	}

	// Without a Prometheus among the providers (the demo cluster): no button.
	p.status.set([]metrics.ProviderStatus{{Provider: "demo", Label: metrics.ServerLabel, Status: metrics.StatusAvailable}}, fixedNow)
	if p.status.settings.Visible() {
		t.Error("a Prometheus button without Prometheus")
	}
}

// TestRenderPrometheusSettings draws the dialog in both themes, to COGPIT_PNG_DIR or a temporary
// directory, never to testdata.
func TestRenderPrometheusSettings(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		name := "metrics_prometheus_settings_" + string(mode) + ".png"
		t.Run(name, func(t *testing.T) {
			sh, f, q, p := promDrawer(t, mode, withProviders(bothAnswering()))
			withPrometheusCommands(f, engine.PrometheusSetting{Context: "kind-cogpit", Address: "observability/thanos-query:9090", Prefix: "/prometheus", Kept: true})
			p.status.settings.OnTapped()
			q.until(t, "the dialog", func() bool { return findButton(sh.win, "Save") != nil })
			settle(sh.win)
			dir := os.Getenv("COGPIT_PNG_DIR")
			if dir == "" {
				dir = t.TempDir()
			}
			out, err := os.Create(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			if err := png.Encode(out, sh.win.Canvas().Capture()); err != nil {
				t.Fatal(err)
			}
			t.Log("rendered", out.Name())
		})
	}
}
