package fyneui

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// startShell starts a shell on eng (the fake engine f, or a wrapper of it) in a 1440×900
// offscreen window, as newTestShell does for f itself.
func startShell(t testing.TB, eng Engine, f *fakeEngine, opts Options) (*shell, fyne.Window, *mainQueue) {
	t.Helper()
	a := test.NewTempApp(t)
	a.Settings().SetTheme(newTheme(opts.Theme))
	w := test.NewTempWindow(t, nil)
	w.SetPadded(false)
	q := &mainQueue{}
	sh := newShell(eng, a, w, opts, q.do)
	sh.now = func() time.Time { return fixedNow }
	sh.confirmer.armDelay = 0
	f.SetConfirmer(sh.confirmer)
	w.SetContent(sh.root)
	w.Resize(fyne.NewSize(1440, 900))
	sh.start()
	t.Cleanup(func() {
		sh.stop()
		q.drain()
	})
	return sh, w, q
}

// logsEngine is the fake engine whose log streams are what the real engine streams for the tab:
// a pod's container, or a deployment's pods with "<pod>/<container> " before every line, with
// the server's timestamps when the tab asks for them. Streams stay open until closed.
type logsEngine struct{ *fakeEngine }

var demoMerged = []struct{ source, ts, msg string }{
	{"checkout-api-7d9f8b6c4-x2k9p/api", "2026-10-11T14:28:01.204118340Z", "starting checkout-api version=2.14.1 commit=9f3c2e1"},
	{"checkout-api-7d9f8b6c4-q8w3n/api", "2026-10-11T14:28:01.311902118Z", "starting checkout-api version=2.14.1 commit=9f3c2e1"},
	{"checkout-api-7d9f8b6c4-x2k9p/envoy", "2026-10-11T14:28:01.402551730Z", "envoy ready; upstream api:8080 healthy"},
	{"checkout-api-7d9f8b6c4-x2k9p/api", "2026-10-11T14:28:09.023772001Z", "POST /api/v1/checkout 201 18.2ms cart=c_8f2k1"},
	{"checkout-api-7d9f8b6c4-q8w3n/api", "2026-10-11T14:28:12.774310925Z", "ERROR payment provider unreachable: dial tcp 10.96.14.2:443: i/o timeout; retrying with exponential backoff, attempt=1 next=2s"},
	{"checkout-api-7d9f8b6c4-q8w3n/api", "2026-10-11T14:28:14.801233417Z", "payment provider reachable again after 2.03s"},
	{"checkout-api-7d9f8b6c4-x2k9p/envoy", "2026-10-11T14:28:15.112908733Z", "upstream connect error or disconnect/reset before headers. retried and the latest reset reason: connection timeout"},
	{"checkout-api-7d9f8b6c4-x2k9p/api", "2026-10-11T14:28:19.023109876Z", "GET /healthz 200 0.4ms"},
	{"checkout-api-7d9f8b6c4-q8w3n/api", "2026-10-11T14:28:22.640022190Z", "error budget burn rate 0.4 (ok)"},
}

func (e logsEngine) OpenLogs(_ context.Context, tabID string) (io.ReadCloser, error) {
	e.mu.Lock()
	e.opened[tabID]++
	tab, ok := e.ui.DockTab(tabID)
	e.mu.Unlock()
	if !ok {
		return nil, io.ErrClosedPipe
	}
	var b strings.Builder
	for _, l := range demoMerged {
		if tab.Ref.Kind != kindPods {
			b.WriteString(l.source + " ")
		} else if !strings.HasSuffix(l.source, "x2k9p/api") {
			continue
		}
		if tab.Logs.Timestamps {
			b.WriteString(l.ts + " ")
		}
		b.WriteString(l.msg + "\n")
	}
	return newOpenStream(e.fakeEngine, tabID, b.String()), nil
}

// withLogCommands answers logs.setOptions and logs.controller as the engine does.
func withLogCommands(t testing.TB, f *fakeEngine, deployment model.ObjectRef) {
	t.Helper()
	f.handlers["logs.setOptions"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p logOptionsParams
		if err := json.Unmarshal(inv.Params, &p); err != nil {
			return nil, err
		}
		f.update(func(u *engine.UIState) {
			for i := range u.Dock {
				o := &u.Dock[i].Logs
				if u.Dock[i].ID != p.ID {
					continue
				}
				for dst, v := range map[*bool]*bool{&o.Previous: p.Previous, &o.Timestamps: p.Timestamps, &o.ResourceNames: p.ResourceNames, &o.Wrap: p.Wrap, &o.Filter: p.Filter} {
					if v != nil {
						*dst = *v
					}
				}
				if p.Search != nil {
					o.Search = *p.Search
				}
			}
		})
		return nil, nil
	}
	f.handlers["logs.controller"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p dockParams
		if err := json.Unmarshal(inv.Params, &p); err != nil {
			return nil, err
		}
		f.update(func(u *engine.UIState) {
			for i := range u.Dock {
				if u.Dock[i].ID == p.ID {
					r := deployment
					u.Dock[i].Ref, u.Dock[i].Container, u.Dock[i].Title = &r, "", "Logs: Deployment "+r.Name
					u.Dock[i].Logs.ResourceNames = true
				}
			}
		})
		return nil, nil
	}
	if err := engine.Register(f.reg, engine.Spec[dockParams]{ID: "logs.controller", Label: "Logs of the Controlling Resource", Danger: engine.DangerRead,
		Enabled: func(*engine.Session, dockParams) error { return f.reason("logs.controller") },
		Run:     func(context.Context, *engine.Session, dockParams) (any, error) { return nil, nil }}); err != nil {
		t.Fatal(err)
	}
}

// logsScene is the pods view with a logs tab in the dock: on a pod, or on its deployment.
func logsScene(t *testing.T, mode ThemeMode, workload bool, opts engine.LogOptions) (*shell, fyne.Window, *mainQueue, *fakeEngine) {
	t.Helper()
	f := connectedFake(t, 200)
	pod := model.ObjectRef{Context: "demo", Kind: kindPods, Namespace: "checkout", Name: "checkout-api-7d9f8b6c4-x2k9p"}
	dep := model.ObjectRef{Context: "demo", Kind: "deployments", Namespace: "checkout", Name: "checkout-api"}
	withLogCommands(t, f, dep)
	tab := engine.DockTab{ID: "logs-1", Type: engine.DockLogs, Title: "Logs: " + pod.Name + "/api", Context: "demo", Ref: &pod, Container: "api", Logs: opts}
	if workload {
		tab = engine.DockTab{ID: "logs-1", Type: engine.DockLogs, Title: "Logs: Deployment checkout-api", Context: "demo", Ref: &dep, Logs: opts}
	}
	f.ui.Dock, f.ui.ActiveDock = []engine.DockTab{tab}, "logs-1"
	sh, w, q := startShell(t, logsEngine{f}, f, Options{Theme: mode})
	q.until(t, "the log lines", func() bool {
		lt := logTabOf(sh)
		return len(sh.kinds) > 0 && lt != nil && lt.view != nil && len(lt.view.ShownLines()) > 0
	})
	return sh, w, q, f
}

func logTabOf(sh *shell) *logTab {
	if c, ok := sh.dock.contents["logs-1"]; ok {
		return c.logs
	}
	return nil
}

// TestLogTabOptionsAreEngineState: a toggle in the toolbar asks the engine (logs.setOptions) and
// the view follows the engine's state, whoever changed it; an option that changes the request
// (timestamps) opens the stream again, the others do not.
func TestLogTabOptionsAreEngineState(t *testing.T) {
	sh, _, q, f := logsScene(t, ThemeDark, false, engine.LogOptions{})
	lt := logTabOf(sh)
	if lt.names.Visible() || !lt.controller.Visible() {
		t.Error("a pod's tab shows resource names, or not the controller button")
	}
	test.Tap(lt.wrap)
	q.until(t, "wrap from the engine", func() bool { return lt.view.View().Wrap })
	if calls := f.callsOf("logs.setOptions"); len(calls) != 1 || !strings.Contains(string(calls[0].Params), `"wrap":true`) {
		t.Errorf("calls %v", calls)
	}

	// An agent turns timestamps on and searches: the checks follow, the stream opens again.
	go func() {
		_, _ = f.Dispatch(context.Background(), engine.Invocation{ID: "logs.setOptions", Origin: engine.OriginExternalAgent,
			Params: mustJSON(t, map[string]any{"id": "logs-1", "timestamps": true, "search": "POST", "filter": true})})
	}()
	q.until(t, "the agent's options", func() bool {
		return lt.timestamps.Checked && lt.filter.Checked && lt.search.Text == "POST" && lt.view.View().Search == "POST"
	})
	q.until(t, "the stream opened again", func() bool {
		f.mu.Lock()
		defer f.mu.Unlock()
		return f.opened["logs-1"] == 2
	})
	q.until(t, "the timestamped lines", func() bool {
		lines := lt.view.ShownLines()
		return len(lines) > 0 && strings.HasPrefix(lines[0], "2026-10-11T14:28:01.204118340Z ")
	})
	if n := len(f.callsOf("logs.setOptions")); n != 2 { // the agent's; following it dispatched nothing
		t.Errorf("%d setOptions calls", n)
	}
	f.mu.Lock()
	closed := f.closed["logs-1"]
	f.mu.Unlock()
	if closed < 1 {
		t.Error("the first stream was not closed")
	}
}

// TestLogTabSearchIsDebouncedAndSlashFocusesIt: typing in the search sends it once the user
// pauses; "/" with nothing focused puts the focus into the search.
func TestLogTabSearchIsDebouncedAndSlashFocusesIt(t *testing.T) {
	sh, w, q, f := logsScene(t, ThemeDark, false, engine.LogOptions{})
	lt := logTabOf(sh)
	w.Canvas().Unfocus()
	test.TypeOnCanvas(w.Canvas(), "/")
	if w.Canvas().Focused() != lt.search {
		t.Fatalf("focused %v, want the search", w.Canvas().Focused())
	}
	test.Type(lt.search, "health")
	q.until(t, "the search in the engine", func() bool { return lt.view.View().Search == "health" })
	if calls := f.callsOf("logs.setOptions"); len(calls) != 1 || !strings.Contains(string(calls[0].Params), `"search":"health"`) {
		t.Errorf("search calls %d: %v", len(calls), calls)
	}
	if lt.search.Text != "health" {
		t.Errorf("search entry %q", lt.search.Text)
	}
}

// TestLogTabMovesToTheController: the pod's tab moves to its deployment's logs (logs.controller):
// the stream opens again for all pods, with their names; the button greys when the pod has no
// controller.
func TestLogTabMovesToTheController(t *testing.T) {
	sh, _, q, f := logsScene(t, ThemeDark, false, engine.LogOptions{})
	lt := logTabOf(sh)
	q.until(t, "the controller button", func() bool { return !lt.controller.Disabled() })
	test.Tap(lt.controller)
	q.until(t, "logs.controller", func() bool { return len(f.callsOf("logs.controller")) == 1 })
	q.until(t, "the deployment's logs", func() bool {
		lines := lt.view.ShownLines()
		return lt.view.View().Prefixed && len(lines) == len(demoMerged) && strings.HasPrefix(lines[0], "checkout-api-7d9f8b6c4-x2k9p/api ")
	})
	if !lt.names.Visible() || !lt.names.Checked || lt.controller.Visible() {
		t.Error("the deployment's tab hides its names option or shows the controller button")
	}
	if b := sh.dock.tabRow.Objects[0].(*tabButton); b.title != "Logs: Deployment checkout-api" {
		t.Errorf("tab title %q", b.title)
	}

	f2 := connectedFake(t, 10)
	withLogCommands(t, f2, model.ObjectRef{})
	f2.disabled["logs.controller"] = "pod web-0 has no controlling resource"
	pod := f2.tables["pods"].Rows[0].Ref
	f2.ui.Dock, f2.ui.ActiveDock = []engine.DockTab{{ID: "logs-1", Type: engine.DockLogs, Title: "Logs", Context: "demo", Ref: &pod, Container: "api"}}, "logs-1"
	sh2, _, q2 := startShell(t, logsEngine{f2}, f2, Options{})
	q2.until(t, "the greyed controller button", func() bool {
		lt := logTabOf(sh2)
		return lt != nil && lt.controller.Disabled() && strings.Contains(lt.controller.tip, "no controlling resource")
	})
}

// TestLogDownloadWritesTheShownLines: the download writes every line the tab holds as shown, to
// the file the save dialog names, proposed after the tab.
func TestLogDownloadWritesTheShownLines(t *testing.T) {
	sh, _, q, _ := logsScene(t, ThemeDark, true, engine.LogOptions{ResourceNames: true, Timestamps: true, Search: "error", Filter: true})
	lt := logTabOf(sh)
	q.until(t, "every line", func() bool { return len(lt.view.ShownLines()) == len(demoMerged) })
	path := filepath.Join(t.TempDir(), "logs.log")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveLogs(f, lt.view.ShownLines()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	if len(lines) != len(demoMerged) || lines[4] != demoMerged[4].source+" "+demoMerged[4].ts+" "+demoMerged[4].msg {
		t.Errorf("saved %q", lines)
	}
	for title, want := range map[string]string{
		"Logs: Deployment checkout-api": "Deployment_checkout-api.log",
		"Logs: web-0/app":               "web-0_app.log",
		"Logs: \x1b]0;x\a":              "]0;x.log",
		"":                              "logs.log",
	} {
		if got := logFileName(title); got != want {
			t.Errorf("logFileName(%q) = %q, want %q", title, got, want)
		}
	}
	// The button opens the save dialog over the window.
	test.Tap(lt.download)
	if len(sh.win.Canvas().Overlays().List()) == 0 {
		t.Error("no save dialog")
	}
}

func TestRenderLogsWorkloadDark(t *testing.T) {
	_, w, _, _ := logsScene(t, ThemeDark, true, engine.LogOptions{ResourceNames: true, Timestamps: true, Search: "error"})
	savePNG(t, w, "logs_workload_dark.png")
}

func TestRenderLogsWorkloadLight(t *testing.T) {
	_, w, _, _ := logsScene(t, ThemeLight, true, engine.LogOptions{ResourceNames: true, Timestamps: true, Search: "error"})
	savePNG(t, w, "logs_workload_light.png")
}

// TestRenderLogsWrapFilter: a pod's tab with timestamps, wrapped lines and a filter.
func TestRenderLogsWrapFilter(t *testing.T) {
	for _, mode := range []ThemeMode{ThemeDark, ThemeLight} {
		sh, w, q, _ := logsScene(t, mode, true, engine.LogOptions{Timestamps: true, Wrap: true, Search: "error", Filter: true})
		lt := logTabOf(sh)
		q.until(t, "the filtered rows", func() bool { return lt.view.View().Filter && len(lt.view.ShownLines()) == len(demoMerged) })
		savePNG(t, w, "logs_wrap_filter_"+string(mode)+".png")
	}
}

// TestRenderLogsPod: a pod's tab, with the button that moves it to its controller's logs.
func TestRenderLogsPod(t *testing.T) {
	sh, w, q, _ := logsScene(t, ThemeDark, false, engine.LogOptions{Timestamps: true})
	lt := logTabOf(sh)
	q.until(t, "the controller button", func() bool { return lt.controller.Visible() && !lt.controller.Disabled() })
	savePNG(t, w, "logs_pod_dark.png")
}
