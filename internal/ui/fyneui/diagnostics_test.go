package fyneui

import (
	"context"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/engine"
)

// diagnosticsFake is a fake connected to a real cluster's counters: requests by verb and resource,
// open, warm and covered watches; it answers diagnostics.get as the engine does (engine.ReportOf).
func diagnosticsFake(t *testing.T) *fakeEngine {
	f := connectedFake(t, 20)
	f.ui.Context, f.ui.Connection = "kind-dev", engine.Connection{State: engine.Connected}
	gvr := func(group, resource string) schema.GroupVersionResource {
		return schema.GroupVersionResource{Group: group, Version: "v1", Resource: resource}
	}
	f.diag = engine.Diagnostics{Context: "kind-dev", Snapshots: 318}
	f.diag.Requests.Total = 41
	f.diag.Requests.ByVerb = map[string]int64{"list": 6, "watch": 7, "get": 22, "discovery": 4, "create": 2}
	f.diag.Requests.ByVerbResource = map[string]int64{"get pods/log": 14, "get nodes": 8, "list pods": 2, "watch pods": 3,
		"list services": 1, "watch services": 1, "list deployments.apps": 1, "watch deployments.apps": 1, "list secrets": 1,
		"watch secrets": 1, "list nodes": 1, "watch nodes": 1, "discovery": 4, "create selfsubjectrulesreviews.authorization.k8s.io": 2}
	f.diag.Informers = []cache.InformerStats{
		{Scope: cache.Scope{GVR: gvr("", "pods")}, Objects: 214, Handles: 2, Synced: true, Events: 1840},
		{Scope: cache.Scope{GVR: gvr("", "nodes")}, Objects: 6, Handles: 1, Synced: true, Events: 412},
		{Scope: cache.Scope{GVR: gvr("", "secrets")}, Objects: 57, Handles: 1, Synced: true, Events: 57},
		{Scope: cache.Scope{GVR: gvr("", "secrets"), LabelSelector: "owner=helm"}, Objects: 9, Handles: 1, Synced: true, CoveredBy: "secrets"},
		{Scope: cache.Scope{GVR: gvr("apps", "deployments")}, Objects: 31, Synced: true, Events: 31, WarmUntil: fixedNow.Add(4 * time.Minute)},
		{Scope: cache.Scope{GVR: gvr("", "services")}, Objects: 44, Synced: true, Events: 44, WarmUntil: fixedNow.Add(2 * time.Minute)},
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
// the watches open, then warm, then covered, and the objects per kind; it follows the counters
// when they move (EventDiagnostics).
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
	if got := text(d.kinds.Objects[0]); !strings.Contains(got, "214  pods") || len(d.kinds.Objects) != 5 {
		t.Errorf("kinds: first %q, %d kinds", got, len(d.kinds.Objects))
	}
	if !strings.Contains(d.summary.text, "41 API requests since connecting") || !strings.Contains(d.summary.text, "3 open watches, 2 warm, 1 covered") {
		t.Errorf("summary %q", d.summary.text)
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
