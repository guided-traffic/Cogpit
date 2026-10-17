package commands_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

type hotbarOut struct {
	Hotbar engine.HotbarState `json:"hotbar"`
}

func hotbarContexts(h engine.HotbarState) string {
	var out []string
	for _, t := range h.Clusters {
		out = append(out, t.Context)
	}
	return strings.Join(out, ",")
}

// disabledReason returns the reason of a DisabledError, failing the test on any other outcome.
func disabledReason(t *testing.T, err error) string {
	t.Helper()
	var dis *engine.DisabledError
	if !errors.As(err, &dis) {
		t.Fatalf("want a disabled command, got %v", err)
	}
	return dis.Reason
}

// The hotbar commands end to end: pin the connected cluster and another, reorder them, open one by
// its place (connecting through cluster.connect's path), unpin one; ui.state shows the same hotbar,
// one list (D-83), with each tile's connection; nothing reaches the cluster.
// Agents run them without a confirmation (D-2: UI operations), and the audit log records them.
func TestHotbarCommands(t *testing.T) {
	var audit bytes.Buffer
	dir := t.TempDir()
	h := newHarness(t, seed(), engine.Options{Hotbar: engine.NewHotbar(dir), Audit: &audit})
	gui, agent := engine.OriginGUI, engine.OriginExternalAgent

	var out hotbarOut
	h.must(t, gui, "hotbar.addCluster", map[string]any{}, &out) // the connected context
	if got := hotbarContexts(out.Hotbar); got != "test" || out.Hotbar.Clusters[0].Connection != engine.Connected {
		t.Fatalf("after adding the connected cluster: %+v", out.Hotbar)
	}
	err := h.run(t, gui, "hotbar.addCluster", map[string]any{"context": "test"}, nil)
	if r := disabledReason(t, err); !strings.Contains(r, "test is on the hotbar already") {
		t.Errorf("adding it twice: %q", r)
	}
	if err := h.run(t, gui, "hotbar.addCluster", map[string]any{"context": "nope"}, nil); err == nil || !strings.Contains(err.Error(), "no context nope") {
		t.Errorf("adding a context nobody offers: %v", err)
	}
	h.must(t, agent, "hotbar.addCluster", map[string]any{"context": "other"}, nil) // no confirmer: agents run it freely
	h.must(t, gui, "hotbar.moveCluster", map[string]any{"context": "other", "position": 1}, &out)
	if got := hotbarContexts(out.Hotbar); got != "other,test" {
		t.Errorf("after the move: %s", got)
	}
	err = h.run(t, gui, "hotbar.moveCluster", map[string]any{"context": "other", "position": 3}, nil)
	if r := disabledReason(t, err); r != "position 3 is outside the hotbar's 1 to 2" {
		t.Errorf("a move past the end: %q", r)
	}

	// F1 opens the top tile: a connection to "other", as cluster.connect makes it.
	var conn commands.ConnectResult
	h.must(t, agent, "hotbar.openCluster", map[string]any{"slot": 1}, &conn)
	if conn.Context != "other" || conn.State != engine.Connected || h.s.UIState().Context != "other" {
		t.Fatalf("hotbar.openCluster: %+v, UI context %q", conn, h.s.UIState().Context)
	}
	// The same backend afterwards: no reconnect. (Not the UI state's revision: the integrations of a
	// new connection, the Alertmanager's first answer among them, change it on their own.)
	before := h.s.BackendOf("other")
	h.must(t, gui, "hotbar.openCluster", map[string]any{"slot": 1}, &conn)
	if after := h.s.BackendOf("other"); conn.State != engine.Connected || before == nil || after != before {
		t.Errorf("opening the connected cluster again reconnected: %+v, backend %p → %p", conn, before, after)
	}
	if r := disabledReason(t, h.run(t, gui, "hotbar.openCluster", map[string]any{"slot": 5}, nil)); r != "the hotbar has no cluster in slot 5 (it has 2)" {
		t.Errorf("an empty slot: %q", r)
	}

	// ui.state carries the hotbar as the user sees it.
	var state commands.StateResult
	h.must(t, agent, "ui.state", struct{}{}, &state)
	if state.Hotbar == nil || len(state.Hotbar.Clusters) != 2 {
		t.Fatalf("ui.state's hotbar: %+v", state.Hotbar)
	}
	tiles := state.Hotbar.Clusters
	// test stayed connected when other was opened (D-69): both tiles say connected.
	if tiles[0].Context != "other" || tiles[0].Connection != engine.Connected || tiles[1].Connection != engine.Connected || tiles[1].Label != "test" {
		t.Errorf("ui.state's tiles: %+v", tiles)
	}

	// ui.state's hotbar is one list: no banks, no shown bank (D-83).
	var raw map[string]any
	h.must(t, agent, "ui.state", struct{}{}, &raw)
	if hb, _ := raw["hotbar"].(map[string]any); hb == nil || len(hb) != 1 || hb["clusters"] == nil {
		t.Errorf("ui.state's hotbar has more than its clusters: %v", raw["hotbar"])
	}
	h.must(t, gui, "hotbar.removeCluster", map[string]any{"context": "test"}, &out)
	if got := hotbarContexts(out.Hotbar); got != "other" {
		t.Errorf("after removing test: %s", got)
	}
	if r := disabledReason(t, h.run(t, gui, "hotbar.removeCluster", map[string]any{"context": "test"}, nil)); !strings.Contains(r, "test is not on the hotbar") {
		t.Errorf("removing it again: %q", r)
	}

	// Nothing reached the cluster; the file holds what the user sees.
	if calls := h.writer.Calls(); len(calls) != 0 {
		t.Errorf("hotbar commands wrote to the cluster: %v", calls)
	}
	file, err := os.ReadFile(filepath.Join(dir, engine.HotbarFile))
	if err != nil || !strings.Contains(string(file), `"context": "other"`) || strings.Contains(string(file), `"test"`) {
		t.Errorf("hotbar.json: %s %v", file, err)
	}
	for _, id := range []string{"hotbar.addCluster", "hotbar.openCluster"} {
		if !strings.Contains(audit.String(), `"`+id+`"`) {
			t.Errorf("the audit log misses the agent's %s", id)
		}
	}
}

// The hotbar takes more than twelve clusters (D-83), the ones past F12 opened by their slot; a tile
// whose context no source offers any more stays, marked missing, and cannot be opened.
func TestHotbarPastTwelveAndMissingTile(t *testing.T) {
	dir := t.TempDir()
	clusters := []map[string]string{{"context": "gone [old.yaml]"}}
	for i := 2; i <= 14; i++ {
		clusters = append(clusters, map[string]string{"context": fmt.Sprintf("c%d [x.yaml]", i)})
	}
	clusters = append(clusters, map[string]string{"context": "other"})
	doc, _ := json.Marshal(map[string]any{"version": 2, "clusters": clusters})
	if err := os.WriteFile(filepath.Join(dir, engine.HotbarFile), doc, 0o600); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, seed(), engine.Options{Hotbar: engine.NewHotbar(dir)})
	h.must(t, engine.OriginGUI, "hotbar.addCluster", map[string]any{"context": "test"}, nil) // the sixteenth
	var out hotbarOut
	h.must(t, engine.OriginExternalAgent, "hotbar.list", struct{}{}, &out)
	if n := len(out.Hotbar.Clusters); n != 16 || out.Hotbar.Clusters[15].Context != "test" {
		t.Fatalf("%d tiles: %+v", n, out.Hotbar)
	}
	if tl := out.Hotbar.Clusters[0]; !tl.Missing || tl.Label != "gone" {
		t.Errorf("the gone context's tile: %+v", tl)
	}
	r := disabledReason(t, h.run(t, engine.OriginGUI, "hotbar.openCluster", map[string]any{"slot": 1}, nil))
	if !strings.Contains(r, "gone [old.yaml] is no longer offered by any kubeconfig source") {
		t.Errorf("opening a missing tile: %q", r)
	}
	var conn commands.ConnectResult // slot 15, past F12: opened by its place
	h.must(t, engine.OriginExternalAgent, "hotbar.openCluster", map[string]any{"slot": 15}, &conn)
	if conn.Context != "other" || conn.State != engine.Connected {
		t.Errorf("slot 15: %+v", conn)
	}
	h.must(t, engine.OriginGUI, "hotbar.moveCluster", map[string]any{"context": "test", "position": 1}, &out)
	if out.Hotbar.Clusters[0].Context != "test" || out.Hotbar.Clusters[1].Context != "gone [old.yaml]" {
		t.Errorf("after moving the sixteenth to the top: %+v", out.Hotbar.Clusters[:2])
	}
	// Removing it is how it goes; the others keep their order.
	h.must(t, engine.OriginGUI, "hotbar.removeCluster", map[string]any{"context": "gone [old.yaml]"}, &out)
	if out.Hotbar.Clusters[1].Context != "c2 [x.yaml]" || len(out.Hotbar.Clusters) != 15 {
		t.Errorf("after removing the missing tile: %+v", out.Hotbar)
	}
	// A bank parameter of the banks' time is refused: the tools take none (D-83).
	if err := h.run(t, engine.OriginExternalAgent, "hotbar.openCluster", map[string]any{"slot": 1, "bank": 1}, nil); err == nil {
		t.Error("hotbar.openCluster took a bank")
	}
}

// A session without a hotbar (headless) disables every hotbar command with the reason, and ui.state
// carries no hotbar.
func TestHotbarCommandsWithoutAHotbar(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	for id, params := range map[string]any{
		"hotbar.list": struct{}{}, "hotbar.addCluster": map[string]any{}, "hotbar.removeCluster": map[string]any{},
		"hotbar.moveCluster": map[string]any{"context": "test", "position": 1}, "hotbar.openCluster": map[string]any{"slot": 1},
	} {
		if r := disabledReason(t, h.run(t, engine.OriginGUI, id, params, nil)); r != engine.ErrNoHotbar.Error() {
			t.Errorf("%s: %q", id, r)
		}
	}
	var raw map[string]any
	h.must(t, engine.OriginExternalAgent, "ui.state", struct{}{}, &raw)
	if _, ok := raw["hotbar"]; ok {
		t.Error("ui.state reports a hotbar")
	}
}

// The hotbar's commands write no cluster: no verbs, and danger read or ui only; the F-key shortcut
// is registered with the command the keys run. One hotbar: no bank commands (D-83).
func TestHotbarCommandsDeclareNoVerbs(t *testing.T) {
	r := engine.NewRegistry()
	if err := commands.RegisterAll(r); err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range r.Commands() {
		if !strings.HasPrefix(c.ID, "hotbar.") {
			continue
		}
		n++
		if len(c.Verbs) != 0 || (c.Danger != engine.DangerUI && c.Danger != engine.DangerRead) || c.Order != 0 {
			t.Errorf("%s: verbs %v, danger %s, order %d", c.ID, c.Verbs, c.Danger, c.Order)
		}
	}
	if n != 5 {
		t.Errorf("%d hotbar commands, want 5", n)
	}
	if c, _ := r.Get("hotbar.openCluster"); c.Shortcut != "F1–F12" {
		t.Errorf("hotbar.openCluster: shortcut %q", c.Shortcut)
	}
	for _, id := range []string{"hotbar.switchBank", "hotbar.createBank", "hotbar.renameBank", "hotbar.removeBank"} {
		if _, ok := r.Get(id); ok {
			t.Errorf("%s is still registered", id)
		}
	}
}
