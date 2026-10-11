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

// TestPodAttach: Attach Pod opens an attach tab on a container (the first one by default), at
// Lens's place (first in the pod menu, a submenu of the containers); it is mutating like a shell:
// from the GUI it runs, an agent's request needs the user's acceptance unless agent writes are
// allowed.
func TestPodAttach(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	pod := model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}
	cmd, _ := h.s.Registry().Get("pod.attach")
	if cmd.Order != 100 || cmd.Label != "Attach Pod" || !cmd.TakesContainer() || cmd.Danger != engine.DangerMutating ||
		len(cmd.Verbs) != 1 || cmd.Verbs[0] != (engine.Verb{Verb: "create", Resource: "pods", Subresource: "attach"}) {
		t.Errorf("pod.attach: %+v", cmd)
	}

	var res commands.DockResult
	h.must(t, engine.OriginGUI, "pod.attach", map[string]any{"ref": pod}, &res)
	if res.Tab.Type != engine.DockAttach || res.Tab.ID != "attach-1" || res.Tab.Container != "app" || res.Tab.Title != "Attach: web-0/app" ||
		res.Tab.Context != "test" || res.Tab.Ref == nil || res.Tab.Ref.Name != "web-0" || !res.Created {
		t.Errorf("attach tab %+v", res)
	}
	if u := h.s.UIState(); u.ActiveDock != "attach-1" || len(u.Dock) != 1 {
		t.Errorf("dock %+v", u.Dock)
	}
	if err := h.run(t, engine.OriginGUI, "pod.attach", map[string]any{"ref": pod, "container": "sidecar"}, nil); err == nil ||
		!strings.Contains(err.Error(), "no container \"sidecar\"") {
		t.Errorf("unknown container: %v", err)
	}

	if err := h.run(t, engine.OriginExternalAgent, "pod.attach", map[string]any{"ref": pod}, nil); !errors.Is(err, engine.ErrConfirmationUnavailable) {
		t.Errorf("agent attach without a GUI: %v", err)
	}
	c := &confirmations{answer: true}
	h.s.SetConfirmer(c)
	h.must(t, engine.OriginBuiltInAgent, "pod.attach", map[string]any{"ref": pod, "container": "app"}, nil)
	if req := c.last(t); req.Command != "pod.attach" || req.Danger != engine.DangerMutating || req.Objects[0].Name != "web-0" || req.Context != "test" {
		t.Errorf("confirmation %+v", req)
	}
	h.s.SetAgentWrites("test", true)
	asked := len(c.all())
	h.must(t, engine.OriginExternalAgent, "pod.attach", map[string]any{"ref": pod}, nil)
	if len(c.all()) != asked {
		t.Error("an allowed agent attach asked again")
	}

	// Not on another kind, not on the demo cluster.
	if err := cmd.Enabled(h.s, cmd.ObjectParams(model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"}, "")); err == nil {
		t.Error("attach applies to deployments")
	}
	s := engine.New(engine.Options{Connector: &fakeConnector{src: seed(), writer: &fakeWriter{}, demo: true}, SnapshotInterval: 10 * time.Millisecond})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(context.Background(), "demo"); err != nil {
		t.Fatal(err)
	}
	demo, _ := s.Registry().Get("pod.attach")
	if err := demo.Enabled(s, demo.ObjectParams(pod, "app")); err == nil || !strings.Contains(err.Error(), "attach sessions are not available on the demo cluster") {
		t.Errorf("attach on the demo cluster: %v", err)
	}
	if _, err := s.OpenTerminal(context.Background(), "attach-1"); err == nil {
		t.Error("an attach tab opened that the demo cluster does not have")
	}
}
