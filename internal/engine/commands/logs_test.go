package commands_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// TestWorkloadLogsOpensOneTabPerResource: Logs of a Deployment, Daemon Set, Stateful Set, Replica
// Set or Job opens a logs tab on the resource with its pods' names shown, or activates the one
// open on it; it is at Lens's place (after the kind's other actions), a read, and not offered for
// other kinds.
func TestWorkloadLogsOpensOneTabPerResource(t *testing.T) {
	h := newWorkloadHarness(t, nil, nil, engine.Options{})
	cmd, _ := h.s.Registry().Get("workload.logs")
	if cmd.Order != 300 || cmd.Danger != engine.DangerRead || cmd.Label != "Logs" || cmd.TakesContainer() || len(cmd.DialogFields()) != 0 {
		t.Errorf("workload.logs: %+v", cmd)
	}
	for kind, name := range map[string]string{
		"deployments": "web", "daemonsets": "agent", "statefulsets": "db", "replicasets": "orphan", "jobs": "migrate",
	} {
		ns := "shop"
		if kind == "daemonsets" || kind == "jobs" {
			ns = "ops"
		}
		ref := actionRef(kind, ns, name)
		if err := cmd.Enabled(h.s, cmd.ObjectParams(ref, "")); err != nil {
			t.Errorf("%s: %v", kind, err)
		}
		var res commands.DockResult
		h.must(t, engine.OriginExternalAgent, "workload.logs", map[string]any{"ref": ref}, &res)
		if !res.Created || res.Tab.Type != engine.DockLogs || res.Tab.Ref.Kind != kind || res.Tab.Container != "" ||
			!res.Tab.Logs.ResourceNames || !strings.HasPrefix(res.Tab.Title, "Logs: ") || !strings.HasSuffix(res.Tab.Title, " "+name) {
			t.Errorf("%s: tab %+v", kind, res)
		}
	}
	var webTab string
	for _, tab := range h.s.UIState().Dock {
		if tab.Ref.Kind == "deployments" {
			webTab = tab.ID
		}
	}
	var again commands.DockResult
	h.must(t, engine.OriginGUI, "workload.logs", map[string]any{"ref": actionRef("deployments", "shop", "web")}, &again)
	if again.Created || again.Tab.ID != webTab || again.Tab.Title != "Logs: Deployment web" {
		t.Errorf("second Logs of web: %+v", again)
	}
	if u := h.s.UIState(); len(u.Dock) != 5 || u.ActiveDock != webTab {
		t.Errorf("dock %+v active %s", u.Dock, u.ActiveDock)
	}
	if err := cmd.Enabled(h.s, cmd.ObjectParams(actionRef("pods", "shop", "web-0"), "")); !errors.Is(err, engine.ErrNotApplicable) {
		t.Errorf("workload.logs on a pod: %v", err)
	}
	if err := h.run(t, engine.OriginGUI, "workload.logs", map[string]any{"ref": actionRef("deployments", "shop", "gone")}, nil); err == nil {
		t.Error("logs of a missing deployment")
	}
}

// TestLogOptionsAreEngineState: logs.setOptions changes the options a tab holds (and ui_state
// shows), only those given; bad tab ids and over-long searches are refused.
func TestLogOptionsAreEngineState(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	pod := model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}
	var res commands.DockResult
	h.must(t, engine.OriginGUI, "pod.logs", map[string]any{"ref": pod}, &res)
	if res.Tab.Logs != (engine.LogOptions{}) {
		t.Errorf("a new pod tab's options %+v", res.Tab.Logs)
	}
	var tab engine.DockTab
	h.must(t, engine.OriginExternalAgent, "logs.setOptions", map[string]any{"id": "logs-1", "timestamps": true, "wrap": true, "search": "error"}, &tab)
	want := engine.LogOptions{Timestamps: true, Wrap: true, Search: "error"}
	if tab.Logs != want {
		t.Errorf("options %+v, want %+v", tab.Logs, want)
	}
	tab = engine.DockTab{} // the result omits options that are off
	h.must(t, engine.OriginGUI, "logs.setOptions", map[string]any{"id": "logs-1", "filter": true, "previous": true, "timestamps": false}, &tab)
	want = engine.LogOptions{Previous: true, Wrap: true, Search: "error", Filter: true}
	if tab.Logs != want {
		t.Errorf("options %+v, want %+v", tab.Logs, want)
	}
	var st commands.StateResult
	h.must(t, engine.OriginExternalAgent, "ui.state", nil, &st)
	if len(st.Dock) != 1 || st.Dock[0].Logs != want {
		t.Errorf("ui_state's dock %+v", st.Dock)
	}

	h.must(t, engine.OriginGUI, "terminal.open", nil, nil)
	for _, params := range []map[string]any{
		{"id": "logs-9", "wrap": true},
		{"id": "terminal-1", "wrap": true},
	} {
		if err := h.run(t, engine.OriginGUI, "logs.setOptions", params, nil); err == nil {
			t.Errorf("%v: accepted", params)
		}
	}
	if err := h.run(t, engine.OriginGUI, "logs.setOptions", map[string]any{"id": "logs-1", "search": strings.Repeat("x", engine.MaxLogSearch+1)}, nil); err == nil {
		t.Error("an over-long search was accepted")
	}
	if err := h.run(t, engine.OriginGUI, "logs.setOptions", map[string]any{"id": "logs-1", "wrap": "yes"}, nil); err == nil {
		t.Error("a string for a boolean option was accepted")
	}
}

// TestLogsControllerMovesTheTab: a pod's tab moves to the resource that controls the pod (a
// Deployment through its replica set), keeping its options and showing the pods' names; a pod
// without a controller cannot, and a resource's tab does not offer it.
func TestLogsControllerMovesTheTab(t *testing.T) {
	h := newWorkloadHarness(t, nil, nil, engine.Options{})
	open := func(pod model.ObjectRef) string {
		var res commands.DockResult
		h.must(t, engine.OriginGUI, "pod.logs", map[string]any{"ref": pod}, &res)
		return res.Tab.ID
	}
	web := open(actionRef("pods", "shop", "web-7d9f-a1"))
	h.must(t, engine.OriginGUI, "logs.setOptions", map[string]any{"id": web, "timestamps": true}, nil)
	cmd, _ := h.s.Registry().Get("logs.controller")
	if err := cmd.Enabled(h.s, mustJSON(t, map[string]any{"id": web})); err != nil {
		t.Fatalf("logs.controller: %v", err)
	}
	var tab engine.DockTab
	h.must(t, engine.OriginGUI, "logs.controller", map[string]any{"id": web}, &tab)
	if tab.ID != web || tab.Ref.Kind != "deployments" || tab.Ref.Name != "web" || tab.Container != "" || tab.Title != "Logs: Deployment web" ||
		!tab.Logs.ResourceNames || !tab.Logs.Timestamps {
		t.Errorf("moved tab %+v (ref %+v)", tab, tab.Ref)
	}
	if err := cmd.Enabled(h.s, mustJSON(t, map[string]any{"id": web})); !errors.Is(err, engine.ErrNotApplicable) {
		t.Errorf("logs.controller on a resource's tab: %v", err)
	}

	db := open(actionRef("pods", "shop", "db-0"))
	h.must(t, engine.OriginExternalAgent, "logs.controller", map[string]any{"id": db}, &tab)
	if tab.Ref.Kind != "statefulsets" || tab.Ref.Name != "db" {
		t.Errorf("db-0's tab moved to %+v", tab.Ref)
	}
	loner := open(actionRef("pods", "shop", "web-0"))
	if err := cmd.Enabled(h.s, mustJSON(t, map[string]any{"id": loner})); !errors.Is(err, engine.ErrNoController) {
		t.Errorf("logs.controller for a pod without a controller: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := h.s.MoveLogsTab(ctx, "logs-77"); err == nil {
		t.Error("moved a tab that does not exist")
	}
}
