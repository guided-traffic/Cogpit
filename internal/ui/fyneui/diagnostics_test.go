package fyneui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// diagnosticsFake is a fake connected to a real cluster's counters: requests by verb and resource,
// open, warm and covered watches; it answers diagnostics.get as the engine does (engine.ReportOf).
func diagnosticsFake(t *testing.T) *fakeEngine {
	f := connectedFake(t, 20)
	f.ui.Context, f.ui.Connection = "kind-dev", engine.Connection{State: engine.Connected}
	// The counters cross the seam as JSON: the UI may not import package cache to name its types.
	diag := `{"context": "kind-dev", "snapshots": 318, "requests": {"total": 41,
	  "byVerb": {"list": 6, "watch": 7, "get": 22, "discovery": 4, "create": 2},
	  "byVerbResource": {"get pods/log": 14, "get nodes": 8, "list pods": 2, "watch pods": 3, "list services": 1,
	    "watch services": 1, "list deployments.apps": 1, "watch deployments.apps": 1, "list secrets": 1, "watch secrets": 1,
	    "list nodes": 1, "watch nodes": 1, "discovery": 4, "create selfsubjectrulesreviews.authorization.k8s.io": 2}},
	  "discovery": {"source": "disk", "checked": true, "groups": 31, "resources": 212, "persisted": true},
	  "informers": [
	    {"scope": {"gvr": {"Version": "v1", "Resource": "pods"}}, "objects": 214, "handles": 2, "synced": true, "events": 1840, "bytes": 1363149},
	    {"scope": {"gvr": {"Version": "v1", "Resource": "nodes"}}, "objects": 6, "handles": 1, "synced": true, "events": 412, "bytes": 30000},
	    {"scope": {"gvr": {"Version": "v1", "Resource": "secrets"}}, "objects": 57, "handles": 1, "synced": true, "events": 57, "bytes": 120000},
	    {"scope": {"gvr": {"Version": "v1", "Resource": "secrets"}, "labelSelector": "owner=helm"}, "objects": 9, "handles": 1, "synced": true, "coveredBy": "secrets"},
	    {"scope": {"gvr": {"Group": "apps", "Version": "v1", "Resource": "deployments"}}, "objects": 31, "synced": true, "events": 31, "warmUntil": "2026-10-11T14:34:00Z", "bytes": 160000},
	    {"scope": {"gvr": {"Version": "v1", "Resource": "services"}}, "objects": 44, "synced": true, "events": 44, "warmUntil": "2026-10-11T14:32:00Z", "bytes": 130000}]}`
	f.diag = engine.Diagnostics{} // not the demo cluster's
	if err := json.Unmarshal([]byte(diag), &f.diag); err != nil {
		t.Fatal(err)
	}
	f.handlers["diagnostics.get"] = func(context.Context, engine.Invocation) (any, error) {
		return engine.ReportOf(f.Diagnostics(), true), nil
	}
	return f
}

// openDiagnostics adds the Diagnostics tab to the fake's dock, as diagnostics.open does.
func openDiagnostics(f *fakeEngine) {
	f.update(func(u *engine.UIState) {
		u.Dock = append(u.Dock, engine.DockTab{ID: "diagnostics-1", Type: engine.DockDiagnostics, Title: "Diagnostics"})
		u.ActiveDock = "diagnostics-1"
	})
}

func diagnosticsTabOf(sh *shell) *diagnosticsTab {
	for _, c := range sh.dock.contents {
		if d, ok := c.editor.(*diagnosticsTab); ok {
			return d
		}
	}
	return nil
}

// The Diagnostics tab shows the counters in full: the requests by verb and resource, most first,
// the watches open, then warm, then covered, and the objects and memory per kind, the most memory
// first, their sums and where discovery came from; it follows the counters when they move
// (EventDiagnostics).
func TestDiagnosticsTab(t *testing.T) {
	f := diagnosticsFake(t)
	sh, _, q := newTestShell(t, f, Options{})
	openDiagnostics(f)
	q.until(t, "the Diagnostics", func() bool { d := diagnosticsTabOf(sh); return d != nil && len(d.watches.Objects) == 6 })
	d := diagnosticsTabOf(sh)
	text := func(o any) string { return o.(*textLine).text }
	if got := text(d.requests.Objects[0]); !strings.Contains(got, "14  get pods/log") {
		t.Errorf("first request row %q", got)
	}
	var watches []string
	for _, o := range d.watches.Objects {
		watches = append(watches, text(o))
	}
	if !strings.HasPrefix(watches[0], "nodes · open") || !strings.HasPrefix(watches[2], "secrets · open") ||
		!strings.HasPrefix(watches[3], "deployments.apps · warm") || !strings.Contains(watches[5], "served by the secrets watch") {
		t.Errorf("watches %q", watches)
	}
	if got := text(d.kinds.Objects[0]); !strings.Contains(got, "214     1.3 MiB  pods") || len(d.kinds.Objects) != 5 {
		t.Errorf("kinds: first %q, %d kinds", got, len(d.kinds.Objects))
	}
	if got := text(d.kinds.Objects[1]); !strings.Contains(got, "31   156.2 KiB  deployments.apps") {
		t.Errorf("kinds: second %q, want the next most memory", got)
	}
	for _, want := range []string{"41 API requests since connecting", "3 open watches, 2 warm, 1 covered",
		"352 objects held (1.7 MiB estimated)", "discovery from the disk cache"} {
		if !strings.Contains(d.summary.text, want) {
			t.Errorf("summary %q, want %q", d.summary.text, want)
		}
	}

	f.mu.Lock()
	f.diag.Requests.Total = 99
	f.mu.Unlock()
	f.emit(engine.Event{Type: engine.EventDiagnostics})
	q.until(t, "the new counters", func() bool { return strings.Contains(d.summary.text, "99 API requests") })
}

// The Diagnostics tab, dark and light: the PNGs of the pull request (AGENTS.md rule 7).
func TestRenderDiagnosticsDark(t *testing.T) { renderDiagnostics(t, ThemeDark, "diagnostics_dark.png") }
func TestRenderDiagnosticsLight(t *testing.T) {
	renderDiagnostics(t, ThemeLight, "diagnostics_light.png")
}

func renderDiagnostics(t *testing.T, mode ThemeMode, name string) {
	f := diagnosticsFake(t)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	openDiagnostics(f)
	q.until(t, "the Diagnostics", func() bool {
		d := diagnosticsTabOf(sh)
		return d != nil && len(d.watches.Objects) == 6 && sh.dock.open()
	})
	q.drain()
	savePNG(t, w, name)
}
