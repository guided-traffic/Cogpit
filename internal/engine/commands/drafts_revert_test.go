package commands_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestRevertToOriginal: Revert to original puts the object as the cache holds it now into an Edit
// tab and bases the edit on it; Undo brings the text back with the version it was based on, so
// that an apply of it is still refused over a change it has not shown.
func TestRevertToOriginal(t *testing.T) {
	gui, agent := engine.OriginGUI, engine.OriginExternalAgent
	h, _ := editHarness(t, engine.Options{})
	var opened commands.EditResult
	h.must(t, gui, "object.edit", map[string]any{"ref": checkout}, &opened)
	id := opened.Tab.ID
	edited := strings.Replace(opened.Text, "type: ClusterIP", "type: NodePort", 1)
	h.must(t, gui, "editor.setText", map[string]any{"id": id, "yaml": edited}, nil)

	var rr engine.RevertResult
	h.must(t, agent, "editor.revert", map[string]any{"id": id}, &rr)
	if !rr.Changed || rr.Text != opened.Text || !rr.CanUndo || rr.BaseResourceVersion != opened.BaseResourceVersion {
		t.Fatalf("revert %+v", rr)
	}
	var st engine.EditorState
	h.must(t, gui, "editor.get", map[string]any{"id": id}, &st)
	if st.Text != opened.Text || !st.CanUndoRevert {
		t.Errorf("after the revert %+v", st)
	}
	h.must(t, gui, "editor.revert", map[string]any{"id": id}, &rr) // nothing more to revert; the undo stays
	if rr.Changed || !rr.CanUndo {
		t.Errorf("second revert %+v", rr)
	}
	h.must(t, gui, "editor.undoRevert", map[string]any{"id": id}, &rr)
	if rr.Text != edited || !rr.Changed {
		t.Fatalf("undo %+v", rr)
	}
	if err := h.run(t, gui, "editor.undoRevert", map[string]any{"id": id}, nil); !errors.Is(err, engine.ErrNothingToUndo) {
		t.Errorf("a second undo: %v", err)
	}
	h.must(t, gui, "editor.revert", map[string]any{"id": id}, nil)
	h.must(t, gui, "editor.setText", map[string]any{"id": id, "yaml": opened.Text + "# typed\n"}, nil)
	if err := h.run(t, gui, "editor.undoRevert", map[string]any{"id": id}, nil); !errors.Is(err, engine.ErrNothingToUndo) {
		t.Errorf("an undo after typing: %v", err)
	}

	// The cluster moved on: the revert takes the newer version and bases the edit on it.
	h.must(t, gui, "editor.setText", map[string]any{"id": id, "yaml": edited}, nil)
	obj, err := h.s.Lookup(checkout)
	if err != nil {
		t.Fatal(err)
	}
	bump(h.src, testkit.ServicesGVR, obj, "changed-by", "someone-else")
	newer := cachedRV(t, h, checkout, opened.BaseResourceVersion)
	h.must(t, gui, "editor.revert", map[string]any{"id": id}, &rr)
	if rr.BaseResourceVersion != newer || !strings.Contains(rr.Text, "changed-by: someone-else") {
		t.Fatalf("revert to the newer version %+v", rr)
	}
	h.must(t, gui, "editor.get", map[string]any{"id": id}, &st)
	if st.Stale {
		t.Errorf("a reverted tab is stale: %+v", st)
	}
	h.must(t, gui, "editor.undoRevert", map[string]any{"id": id}, &rr)
	if rr.BaseResourceVersion != opened.BaseResourceVersion || rr.Text != edited {
		t.Fatalf("undo to the older base %+v", rr)
	}
	err = h.run(t, gui, "object.apply", map[string]any{"id": id, "yaml": edited}, nil)
	var ce *engine.ConflictError
	if !errors.As(err, &ce) {
		t.Errorf("an apply of the undone text over the newer version: %v", err)
	}

	// A Secret's tab is the user's: agents cannot revert it (D-19).
	h.src.Set(secretsGVR, testkit.Object("v1", "Secret", "shop", "db", map[string]any{"data": map[string]any{"password": "aHVudGVyMg=="}}))
	var sec commands.EditResult
	h.must(t, gui, "object.edit", map[string]any{"ref": dbSecret}, &sec)
	if err := h.run(t, agent, "editor.revert", map[string]any{"id": sec.Tab.ID}, nil); !errors.Is(err, engine.ErrSecretGUIOnly) {
		t.Errorf("an agent's revert of a Secret: %v", err)
	}
	h.must(t, gui, "editor.revert", map[string]any{"id": sec.Tab.ID}, nil)

	// Only Edit tabs revert.
	var create commands.EditResult
	h.must(t, gui, "resource.createTab", map[string]any{}, &create)
	if err := h.run(t, gui, "editor.revert", map[string]any{"id": create.Tab.ID}, nil); err == nil {
		t.Error("a Create tab reverted")
	}
}
