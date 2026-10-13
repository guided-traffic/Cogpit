package commands_test

import (
	"context"
	"slices"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// The Diagnostics view arranges the counters: requests by verb and by verb and resource, most
// first; watches open, then warm, then covered, a covered scope's objects not counted twice;
// objects per kind, most first.
func TestDiagnosticsReport(t *testing.T) {
	pods := schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	secrets := schema.GroupVersionResource{Version: "v1", Resource: "secrets"}
	deploys := schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
	warm := time.Date(2026, 10, 13, 2, 0, 0, 0, time.UTC)
	d := engine.Diagnostics{Context: "kind-dev", Snapshots: 42, Requests: cluster.RequestStats{Total: 23,
		ByVerb:         map[string]int64{"list": 4, "watch": 4, "get": 12, "discovery": 3},
		ByVerbResource: map[string]int64{"get pods/log": 12, "list pods": 2, "watch pods": 2, "list secrets": 1, "watch secrets": 1, "list deployments.apps": 1, "watch deployments.apps": 1, "discovery": 3}},
		Informers: []cache.InformerStats{
			{Scope: cache.Scope{GVR: secrets, LabelSelector: "owner=helm"}, Objects: 3, Handles: 1, Synced: true, CoveredBy: "secrets"},
			{Scope: cache.Scope{GVR: deploys}, Objects: 5, Synced: true, Events: 5, WarmUntil: warm},
			{Scope: cache.Scope{GVR: pods}, Objects: 120, Handles: 2, Synced: true, Events: 300},
			{Scope: cache.Scope{GVR: secrets}, Objects: 9, Handles: 1, Synced: true, Events: 9},
		}}
	r := engine.ReportOf(d, true)
	if r.Requests != 23 || r.RequestsByVerb[0] != (engine.DiagCount{Name: "get", Count: 12}) || len(r.RequestsByVerb) != 4 {
		t.Errorf("by verb: %+v", r.RequestsByVerb)
	}
	if first := r.RequestsByTarget[0]; first.Verb != "get" || first.Resource != "pods/log" || first.Count != 12 {
		t.Errorf("by verb and resource: %+v", r.RequestsByTarget)
	}
	if !slices.ContainsFunc(r.RequestsByTarget, func(x engine.DiagRequest) bool { return x.Verb == "discovery" && x.Resource == "" && x.Count == 3 }) {
		t.Errorf("a request without a resource: %+v", r.RequestsByTarget)
	}
	var states []string
	for _, w := range r.Watches {
		states = append(states, w.State+" "+w.Scope)
	}
	want := []string{"open pods", "open secrets", "warm deployments.apps", "covered secrets with owner=helm"}
	if !slices.Equal(states, want) || r.Open != 2 || r.Warm != 1 || r.Covered != 1 {
		t.Errorf("watches %v (open %d, warm %d, covered %d), want %v", states, r.Open, r.Warm, r.Covered, want)
	}
	if !r.Watches[2].WarmUntil.Equal(warm) || r.Watches[3].CoveredBy != "secrets" {
		t.Errorf("warm until %v, covered by %q", r.Watches[2].WarmUntil, r.Watches[3].CoveredBy)
	}
	if r.Objects != 134 || len(r.Kinds) != 3 || r.Kinds[0] != (engine.DiagKind{Kind: "pods", Objects: 120, Watches: 1}) ||
		r.Kinds[1] != (engine.DiagKind{Kind: "secrets", Objects: 9, Watches: 1}) {
		t.Errorf("objects %d, kinds %+v", r.Objects, r.Kinds)
	}
}

// diagnostics.get reads the connected cluster's watches from the cache, a view's open and a
// released one warm, and sends no request of its own; diagnostics.open opens one dock tab.
func TestDiagnosticsCommands(t *testing.T) {
	src := seed()
	h := newHarness(t, src, engine.Options{})
	gui := engine.OriginGUI
	h.must(t, gui, "ui.openView", map[string]any{"view": "pods"}, nil)
	waitTable(t, h.s, "pods", func(tb *engine.TableSnapshot) bool { return tb.Synced })
	if _, err := h.s.Objects(context.Background(), "services", nil); err != nil { // a lookup: released, warm
		t.Fatal(err)
	}
	h.waitSettled(t)
	lists, watches := src.Lists(), src.Watches()
	var r engine.DiagnosticsReport
	h.must(t, engine.OriginExternalAgent, "diagnostics.get", struct{}{}, &r)
	if src.Lists() != lists || src.Watches() != watches {
		t.Errorf("diagnostics.get sent requests: lists %d → %d, watches %d → %d", lists, src.Lists(), watches, src.Watches())
	}
	state := map[string]string{}
	for _, w := range r.Watches {
		state[w.Scope] = w.State
	}
	if state["pods"] != engine.WatchOpen || state["services"] != engine.WatchWarm || !r.Connected || r.Context != "test" {
		t.Errorf("watches %+v (connected %v, context %q)", state, r.Connected, r.Context)
	}
	if !slices.ContainsFunc(r.Kinds, func(k engine.DiagKind) bool { return k.Kind == "pods" && k.Objects == 4 }) {
		t.Errorf("objects per kind: %+v", r.Kinds)
	}

	var first, again commands.DockResult
	h.must(t, gui, "diagnostics.open", struct{}{}, &first)
	h.must(t, engine.OriginExternalAgent, "diagnostics.open", struct{}{}, &again)
	if !first.Created || again.Created || first.Tab.ID != again.Tab.ID || first.Tab.Type != engine.DockDiagnostics {
		t.Errorf("the tab: %+v, then %+v", first, again)
	}
}
