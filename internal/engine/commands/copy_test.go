package commands_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// TestCopyName: Copy returns the object's name, for an agent as for the GUI (which puts it on the
// clipboard); it is the first of every kind's drawer actions and in no row menu; it works on the
// demo cluster, which cannot be changed.
func TestCopyName(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	cmd, _ := h.s.Registry().Get("object.copyName")
	if cmd.Danger != engine.DangerUI || !cmd.DrawerOnly || cmd.Order == 0 || len(cmd.Kinds) != 0 || cmd.Label != "Copy" {
		t.Errorf("object.copyName: %+v", cmd)
	}
	for _, kind := range []string{"pods", "deployments", "services", "nodes", "configmaps"} {
		if acts := h.s.Registry().Actions(kind); len(acts) == 0 || acts[0].ID != "object.copyName" {
			t.Errorf("%s: drawer actions start with %v", kind, acts)
		}
	}
	var res commands.CopyResult
	h.must(t, engine.OriginExternalAgent, "object.copyName", map[string]any{"ref": model.ObjectRef{Kind: "services", Namespace: "shop", Name: "checkout"}}, &res)
	if res.Name != "checkout" || res.Ref.Context != "test" || res.Ref.Kind != "services" {
		t.Errorf("result %+v", res)
	}
	if err := h.run(t, engine.OriginGUI, "object.copyName", map[string]any{"ref": model.ObjectRef{Kind: "services", Namespace: "shop", Name: "gone"}}, nil); !errors.Is(err, engine.ErrNotFound) {
		t.Errorf("copy of a missing object: %v", err)
	}
	if err := h.run(t, engine.OriginGUI, "object.copyName", map[string]any{"ref": model.ObjectRef{Kind: "services", Name: "checkout"}}, nil); err == nil {
		t.Error("copy of a namespaced object without its namespace")
	}

	s := engine.New(engine.Options{Connector: &fakeConnector{src: seed(), writer: &fakeWriter{}, demo: true}, SnapshotInterval: 10 * time.Millisecond})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	pod := model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}
	if _, err := s.Dispatch(context.Background(), engine.Invocation{ID: "object.copyName", Params: mustJSON(t, map[string]any{"ref": pod}), Origin: engine.OriginGUI}); err != nil {
		t.Errorf("copy on the demo cluster: %v", err)
	}
}
