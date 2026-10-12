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

func hotbarContexts(b engine.HotbarBank) string {
	var out []string
	for _, t := range b.Clusters {
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
// its place (connecting through cluster.connect's path), banks created, renamed, switched and
// removed; ui.state shows the same hotbar with each tile's connection; nothing reaches the cluster.
// Agents run them without a confirmation (D-2: UI operations), and the audit log records them.
func TestHotbarCommands(t *testing.T) {
	var audit bytes.Buffer
	dir := t.TempDir()
	h := newHarness(t, seed(), engine.Options{Hotbar: engine.NewHotbar(dir), Audit: &audit})
	gui, agent := engine.OriginGUI, engine.OriginExternalAgent

	var out hotbarOut
	h.must(t, gui, "hotbar.addCluster", map[string]any{}, &out) // the connected context
	if got := hotbarContexts(out.Hotbar.Banks[0]); got != "test" || out.Hotbar.Banks[0].Clusters[0].Connection != engine.Connected {
		t.Fatalf("after adding the connected cluster: %+v", out.Hotbar)
	}
	err := h.run(t, gui, "hotbar.addCluster", map[string]any{"context": "test"}, nil)
	if r := disabledReason(t, err); !strings.Contains(r, "test is in bank 1 already") {
		t.Errorf("adding it twice: %q", r)
	}
	if err := h.run(t, gui, "hotbar.addCluster", map[string]any{"context": "nope"}, nil); err == nil || !strings.Contains(err.Error(), "no context nope") {
		t.Errorf("adding a context nobody offers: %v", err)
	}
	h.must(t, agent, "hotbar.addCluster", map[string]any{"context": "other"}, nil) // no confirmer: agents run it freely
	h.must(t, gui, "hotbar.moveCluster", map[string]any{"context": "other", "position": 1}, &out)
	if got := hotbarContexts(out.Hotbar.Banks[0]); got != "other,test" {
		t.Errorf("after the move: %s", got)
	}
	err = h.run(t, gui, "hotbar.moveCluster", map[string]any{"context": "other", "position": 3}, nil)
	if r := disabledReason(t, err); r != "position 3 is outside the bank's 1 to 2" {
		t.Errorf("a move past the end: %q", r)
	}

	// F1 opens the top tile: a connection to "other", as cluster.connect makes it.
	var conn commands.ConnectResult
	h.must(t, agent, "hotbar.openCluster", map[string]any{"slot": 1}, &conn)
	if conn.Context != "other" || conn.State != engine.Connected || h.s.UIState().Context != "other" {
		t.Fatalf("hotbar.openCluster: %+v, UI context %q", conn, h.s.UIState().Context)
	}
	rev := h.s.UIState().Revision
	h.must(t, gui, "hotbar.openCluster", map[string]any{"slot": 1}, &conn)
	if conn.State != engine.Connected || h.s.UIState().Revision != rev {
		t.Errorf("opening the connected cluster again reconnected: %+v, revision %d → %d", conn, rev, h.s.UIState().Revision)
	}
	if r := disabledReason(t, h.run(t, gui, "hotbar.openCluster", map[string]any{"slot": 5}, nil)); r != "bank 1 has no cluster in slot 5 (it has 2)" {
		t.Errorf("an empty slot: %q", r)
	}

	// ui.state carries the hotbar as the user sees it.
	var state commands.StateResult
	h.must(t, agent, "ui.state", struct{}{}, &state)
	if state.Hotbar == nil || len(state.Hotbar.Banks) != 1 || state.Hotbar.Current != 1 {
		t.Fatalf("ui.state's hotbar: %+v", state.Hotbar)
	}
	tiles := state.Hotbar.Banks[0].Clusters
	if tiles[0].Context != "other" || tiles[0].Connection != engine.Connected || tiles[1].Connection != engine.Disconnected || tiles[1].Label != "test" {
		t.Errorf("ui.state's tiles: %+v", tiles)
	}

	// Banks: create (shown at once), add to it, rename, switch, remove.
	h.must(t, gui, "hotbar.createBank", map[string]any{"name": "Prod"}, &out)
	if out.Hotbar.Current != 2 || out.Hotbar.Banks[1].Name != "Prod" {
		t.Fatalf("hotbar.createBank: %+v", out.Hotbar)
	}
	h.must(t, gui, "hotbar.addCluster", map[string]any{"context": "test"}, nil) // to the shown bank, 2
	h.must(t, agent, "hotbar.renameBank", map[string]any{"bank": 2, "name": "Production"}, nil)
	h.must(t, gui, "hotbar.switchBank", map[string]any{"bank": 1}, &out)
	if out.Hotbar.Current != 1 || out.Hotbar.Banks[1].Name != "Production" || hotbarContexts(out.Hotbar.Banks[1]) != "test" {
		t.Errorf("after rename and switch: %+v", out.Hotbar)
	}
	if r := disabledReason(t, h.run(t, gui, "hotbar.switchBank", map[string]any{"bank": 3}, nil)); r != "the hotbar has no bank 3 (it has 2 banks)" {
		t.Errorf("switching to bank 3: %q", r)
	}
	var removed commands.BankRemoved
	h.must(t, agent, "hotbar.removeBank", map[string]any{"bank": 2}, &removed)
	if removed.Removed.Name != "Production" || hotbarContexts(removed.Removed) != "test" || len(removed.Hotbar.Banks) != 1 {
		t.Errorf("hotbar.removeBank: %+v", removed)
	}
	if r := disabledReason(t, h.run(t, gui, "hotbar.removeBank", map[string]any{}, nil)); r != "the hotbar keeps at least one bank" {
		t.Errorf("removing the last bank: %q", r)
	}
	h.must(t, gui, "hotbar.removeCluster", map[string]any{"context": "test"}, &out)
	if got := hotbarContexts(out.Hotbar.Banks[0]); got != "other" {
		t.Errorf("after removing test: %s", got)
	}
	if r := disabledReason(t, h.run(t, gui, "hotbar.removeCluster", map[string]any{"context": "test"}, nil)); !strings.Contains(r, "test is not in bank 1") {
		t.Errorf("removing it again: %q", r)
	}

	// Nothing reached the cluster; the file holds what the user sees.
	if calls := h.writer.Calls(); len(calls) != 0 {
		t.Errorf("hotbar commands wrote to the cluster: %v", calls)
	}
	raw, err := os.ReadFile(filepath.Join(dir, engine.HotbarFile))
	if err != nil || !strings.Contains(string(raw), `"context": "other"`) || strings.Contains(string(raw), "Production") {
		t.Errorf("hotbar.json: %s %v", raw, err)
	}
	for _, id := range []string{"hotbar.addCluster", "hotbar.openCluster", "hotbar.renameBank", "hotbar.removeBank"} {
		if !strings.Contains(audit.String(), `"`+id+`"`) {
			t.Errorf("the audit log misses the agent's %s", id)
		}
	}
}

// A full bank refuses a thirteenth cluster with the reason; a tile whose context no source offers
// any more stays, marked missing, and cannot be opened; F-keys address the shown bank.
func TestHotbarFullBankAndMissingTile(t *testing.T) {
	dir := t.TempDir()
	clusters := []map[string]string{{"context": "gone [old.yaml]"}}
	for i := 2; i <= engine.HotbarBankSize; i++ {
		clusters = append(clusters, map[string]string{"context": fmt.Sprintf("c%d [x.yaml]", i)})
	}
	doc, _ := json.Marshal(map[string]any{"version": 1, "current": 1, "banks": []any{map[string]any{"name": "Default", "clusters": clusters}}})
	if err := os.WriteFile(filepath.Join(dir, engine.HotbarFile), doc, 0o600); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t, seed(), engine.Options{Hotbar: engine.NewHotbar(dir)})
	if r := disabledReason(t, h.run(t, engine.OriginGUI, "hotbar.addCluster", map[string]any{"context": "other"}, nil)); r != "bank full: 12 clusters" {
		t.Errorf("a thirteenth cluster: %q", r)
	}
	var out hotbarOut
	h.must(t, engine.OriginExternalAgent, "hotbar.list", struct{}{}, &out)
	if tl := out.Hotbar.Banks[0].Clusters[0]; !tl.Missing || tl.Label != "gone" {
		t.Errorf("the gone context's tile: %+v", tl)
	}
	r := disabledReason(t, h.run(t, engine.OriginGUI, "hotbar.openCluster", map[string]any{"slot": 1}, nil))
	if !strings.Contains(r, "gone [old.yaml] is no longer offered by any kubeconfig source") {
		t.Errorf("opening a missing tile: %q", r)
	}
	// Removing it is how it goes; the others keep their order.
	h.must(t, engine.OriginGUI, "hotbar.removeCluster", map[string]any{"context": "gone [old.yaml]"}, &out)
	if out.Hotbar.Banks[0].Clusters[0].Context != "c2 [x.yaml]" || len(out.Hotbar.Banks[0].Clusters) != engine.HotbarBankSize-1 {
		t.Errorf("after removing the missing tile: %+v", out.Hotbar.Banks[0])
	}
}

// A session without a hotbar (headless) disables every hotbar command with the reason, and ui.state
// carries no hotbar.
func TestHotbarCommandsWithoutAHotbar(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	for id, params := range map[string]any{
		"hotbar.list": struct{}{}, "hotbar.addCluster": map[string]any{}, "hotbar.removeCluster": map[string]any{},
		"hotbar.moveCluster": map[string]any{"context": "test", "position": 1}, "hotbar.openCluster": map[string]any{"slot": 1},
		"hotbar.switchBank": map[string]any{"bank": 1}, "hotbar.createBank": map[string]any{},
		"hotbar.renameBank": map[string]any{"name": "x"}, "hotbar.removeBank": map[string]any{},
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

// The hotbar's commands write no cluster: no verbs, and danger read or ui only; the F-key shortcuts
// are registered with the two that the keys run.
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
	if n != 9 {
		t.Errorf("%d hotbar commands, want 9", n)
	}
	for id, want := range map[string]string{"hotbar.openCluster": "F1–F12", "hotbar.switchBank": "Shift+F1–F12"} {
		if c, _ := r.Get(id); c.Shortcut != want {
			t.Errorf("%s: shortcut %q, want %q", id, c.Shortcut, want)
		}
	}
}
