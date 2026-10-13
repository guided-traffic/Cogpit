package commands_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// lineList is what a fake window shows in a tab: lines numbered from 1.
type lineList []string

func (l lineList) Len() int           { return len(l) }
func (l lineList) Number(i int) int64 { return int64(i + 1) }
func (l lineList) Text(i int) string  { return l[i] }

// fakeWindow is a DockOutputReader that shows fixed lines per tab, as the window would: a page of
// them with PageDockLines. during runs inside a read, while the window holds the tab.
type fakeWindow struct {
	mu     sync.Mutex
	lines  map[string]lineList
	reads  int
	during func()
}

func (w *fakeWindow) show(id string, lines ...string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lines == nil {
		w.lines = map[string]lineList{}
	}
	w.lines[id] = lines
}

func (w *fakeWindow) ReadDockOutput(_ context.Context, tab engine.DockTab, q engine.DockOutputQuery) (engine.DockOutput, error) {
	w.mu.Lock()
	src, ok := w.lines[tab.ID]
	w.reads++
	during := w.during
	w.mu.Unlock()
	if during != nil {
		during()
	}
	if !ok {
		return engine.DockOutput{Opening: true, DockPage: engine.DockPage{Lines: []string{}}}, nil
	}
	return engine.DockOutput{DockPage: engine.PageDockLines(src, q), Shown: src.Len(), Held: src.Len()}, nil
}

// openDockTab opens a dock tab with a command and returns its id.
func openDockTab(t *testing.T, h *harness, id string, params any) string {
	t.Helper()
	var res struct {
		Tab engine.DockTab `json:"tab"`
	}
	h.must(t, engine.OriginGUI, id, params, &res)
	return res.Tab.ID
}

func readDock(t *testing.T, h *harness, origin engine.Origin, params map[string]any) (engine.DockOutput, error) {
	t.Helper()
	var out engine.DockOutput
	err := h.run(t, origin, "dock.read", params, &out)
	return out, err
}

// TestDockReadLogsTab: an agent reads a logs tab's lines as the window shows them, the newest by
// default, paging with after and before; its call is audited without the lines.
func TestDockReadLogsTab(t *testing.T) {
	audit := &syncBuffer{}
	h := newHarness(t, seed(), engine.Options{Audit: audit})
	w := &fakeWindow{}
	h.s.SetDockOutputReader(w)
	pod := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "shop", Name: "web-0"}
	waitTable(t, h.s, "pods", func(*engine.TableSnapshot) bool { return true })
	h.must(t, engine.OriginGUI, "ui.openView", map[string]any{"view": "pods"}, nil)
	id := openDockTab(t, h, "pod.logs", map[string]any{"ref": pod})
	lines := make([]string, 0, 50)
	for i := range 50 {
		lines = append(lines, "GET /healthz 200 #"+string(rune('a'+i%26)))
	}
	w.show(id, lines...)

	out, err := readDock(t, h, engine.OriginExternalAgent, map[string]any{"id": id, "limit": 10})
	if err != nil {
		t.Fatal(err)
	}
	if out.Tab.ID != id || out.Tab.Type != engine.DockLogs || len(out.Lines) != 10 || out.First != 41 || out.Last != 50 || !out.More || out.Shown != 50 {
		t.Fatalf("newest page: tab %s %s, %d lines %d..%d, more %v, shown %d", out.Tab.ID, out.Tab.Type, len(out.Lines), out.First, out.Last, out.More, out.Shown)
	}
	back, err := readDock(t, h, engine.OriginExternalAgent, map[string]any{"id": id, "before": out.First, "limit": 15})
	if err != nil || back.First != 26 || back.Last != 40 || !back.More {
		t.Fatalf("before %d: %d..%d more %v, %v", out.First, back.First, back.Last, back.More, err)
	}
	w.show(id, append(lines, "POST /orders 201")...)
	next, err := readDock(t, h, engine.OriginBuiltInAgent, map[string]any{"id": id, "after": out.Last})
	if err != nil || len(next.Lines) != 1 || next.Lines[0] != "POST /orders 201" || next.More {
		t.Fatalf("after %d: %q more %v, %v", out.Last, next.Lines, next.More, err)
	}
	if _, err := readDock(t, h, engine.OriginExternalAgent, map[string]any{"id": id, "after": 1, "before": 5}); err == nil {
		t.Error("after and before together accepted")
	}
	n := 0
	for _, e := range audit.Lines() {
		if e.Command == "dock.read" {
			n++
			if strings.Contains(string(e.Params), "healthz") || strings.Contains(string(e.Params), "orders") {
				t.Errorf("the audit line carries the output: %s", e.Params)
			}
		}
	}
	if n != 4 {
		t.Errorf("%d audit lines of dock.read, want 4", n)
	}
}

// TestDockReadTerminalNeedsTheUsersSwitch (D-44): a terminal tab is refused to agents and the CLI
// until the user lets agents read it with the tab's switch, which no agent can turn on; the user's
// own window reads it either way; turning the switch off refuses again, even a read under way.
func TestDockReadTerminalNeedsTheUsersSwitch(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{ClusterSettings: engine.NewClusterSettingsStore("")})
	w := &fakeWindow{}
	h.s.SetDockOutputReader(w)
	h.must(t, engine.OriginGUI, "cluster.settings.setAgentWrites", map[string]any{"allowed": true}, nil) // no bearing on it
	id := openDockTab(t, h, "terminal.open", struct{}{})
	const secret = "export TOKEN=s3cr3t-value"
	w.show(id, "$ "+secret, "$ ")

	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		out, err := readDock(t, h, origin, map[string]any{"id": id})
		if !errors.Is(err, engine.ErrTerminalOutputWithheld) || len(out.Lines) != 0 {
			t.Errorf("%s before the switch: %v, lines %q", origin, err, out.Lines)
		}
		if err != nil && strings.Contains(err.Error(), "s3cr3t") {
			t.Errorf("%s: the refusal quotes the output", origin)
		}
		var ge *engine.GUIOnlyError
		if err := h.run(t, origin, "dock.allowAgentRead", map[string]any{"id": id, "allowed": true}, nil); !errors.As(err, &ge) {
			t.Errorf("%s turned the switch on: %v", origin, err)
		}
	}
	if w.reads != 0 {
		t.Errorf("the window was read %d times for refused calls", w.reads)
	}
	if out, err := readDock(t, h, engine.OriginGUI, map[string]any{"id": id}); err != nil || len(out.Lines) != 2 {
		t.Errorf("the window's own read: %q, %v", out.Lines, err)
	}

	var tab engine.DockTab
	h.must(t, engine.OriginGUI, "dock.allowAgentRead", map[string]any{"id": id, "allowed": true}, &tab)
	if !tab.AgentRead {
		t.Fatal("the switch did not turn on")
	}
	if st := h.s.UIState(); !st.Dock[len(st.Dock)-1].AgentRead {
		t.Error("ui_state does not show the switch")
	}
	out, err := readDock(t, h, engine.OriginExternalAgent, map[string]any{"id": id})
	if err != nil || len(out.Lines) != 2 || out.Lines[0] != "$ "+secret || !out.Tab.AgentRead {
		t.Fatalf("after the switch: %q, %v", out.Lines, err)
	}

	// The user switches off while the window reads: nothing of that read is returned.
	w.during = func() { _, _ = h.s.AllowAgentRead(id, false) }
	if out, err := readDock(t, h, engine.OriginExternalAgent, map[string]any{"id": id}); !errors.Is(err, engine.ErrTerminalOutputWithheld) || len(out.Lines) != 0 {
		t.Errorf("switched off during the read: %q, %v", out.Lines, err)
	}
	w.during = nil
	if _, err := readDock(t, h, engine.OriginExternalAgent, map[string]any{"id": id}); !errors.Is(err, engine.ErrTerminalOutputWithheld) {
		t.Errorf("after switching off: %v", err)
	}

	logs := openDockTab(t, h, "pod.logs", map[string]any{"ref": model.ObjectRef{Context: "test", Kind: "pods", Namespace: "shop", Name: "web-0"}})
	if err := h.run(t, engine.OriginGUI, "dock.allowAgentRead", map[string]any{"id": logs, "allowed": true}, nil); err == nil {
		t.Error("the switch was set on a logs tab")
	}
}

// TestDockReadRefusals: an unknown or closed tab, a tab dock.read does not read, a tab the window
// has not opened yet, and a session without a window.
func TestDockReadRefusals(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	w := &fakeWindow{}
	h.s.SetDockOutputReader(w)
	agent := engine.OriginExternalAgent
	if _, err := readDock(t, h, agent, map[string]any{"id": "logs-99"}); err == nil || !strings.Contains(err.Error(), "ui_state") {
		t.Errorf("unknown tab: %v", err)
	}
	id := openDockTab(t, h, "pod.logs", map[string]any{"ref": model.ObjectRef{Context: "test", Kind: "pods", Namespace: "shop", Name: "web-1"}})
	if out, err := readDock(t, h, agent, map[string]any{"id": id}); err != nil || !out.Opening || len(out.Lines) != 0 {
		t.Errorf("a tab the window has not opened: %+v, %v", out, err)
	}
	h.must(t, engine.OriginGUI, "dock.close", map[string]any{"id": id}, nil)
	if _, err := readDock(t, h, agent, map[string]any{"id": id}); err == nil || !strings.Contains(err.Error(), "no dock tab") {
		t.Errorf("closed tab: %v", err)
	}
	edit := openDockTab(t, h, "resource.createTab", map[string]any{})
	if _, err := readDock(t, h, agent, map[string]any{"id": edit}); err == nil || !strings.Contains(err.Error(), "editor_get") {
		t.Errorf("editor tab: %v", err)
	}

	h.s.SetDockOutputReader(nil)
	var dis *engine.DisabledError
	if _, err := readDock(t, h, agent, map[string]any{"id": edit}); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "without a window") {
		t.Errorf("headless: %v", err)
	}
}
