package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

type menuRef struct {
	Ref model.ObjectRef `json:"ref"`
}

type menuContainer struct {
	Ref       model.ObjectRef `json:"ref"`
	Container string          `json:"container,omitempty"`
}

type menuScale struct {
	Ref      model.ObjectRef `json:"ref"`
	Replicas int32           `json:"replicas" jsonschema:"the number of replicas"`
	Note     string          `json:"note,omitempty"`
}

// menuRegistry registers probe commands shaped like the row-menu commands the parallel work adds:
// a cordon/uncordon pair that hides one of them, a scale command with a dialog and Prefill, a
// container command, and Edit and Delete for every kind.
func menuRegistry(t *testing.T) *Registry {
	t.Helper()
	r := NewRegistry()
	run := func(context.Context, *Session, menuRef) (any, error) { return nil, nil }
	targets := func(_ *Session, p menuRef) ([]model.ObjectRef, error) { return []model.ObjectRef{p.Ref}, nil }
	cordoned := func(ref model.ObjectRef) bool { return strings.HasSuffix(ref.Name, "-cordoned") }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(Register(r, Spec[menuRef]{ID: "probe.cordon", Label: "Cordon", Kinds: []string{"nodes"}, Order: 200, Danger: DangerMutating, Targets: targets, Run: run,
		Enabled: func(_ *Session, p menuRef) error {
			if cordoned(p.Ref) {
				return fmt.Errorf("already cordoned: %w", ErrNotApplicable)
			}
			return nil
		}}))
	must(Register(r, Spec[menuRef]{ID: "probe.uncordon", Label: "Uncordon", Kinds: []string{"nodes"}, Order: 200, Danger: DangerMutating, Targets: targets, Run: run,
		Enabled: func(_ *Session, p menuRef) error {
			if !cordoned(p.Ref) {
				return fmt.Errorf("not cordoned: %w", ErrNotApplicable)
			}
			return nil
		}}))
	must(Register(r, Spec[menuRef]{ID: "probe.drain", Label: "Drain", Kinds: []string{"nodes"}, Order: 300, Danger: DangerDestructive, Targets: targets, Run: run,
		Enabled: func(*Session, menuRef) error { return errors.New("the demo cluster cannot be changed") }}))
	must(Register(r, Spec[menuScale]{ID: "probe.scale", Label: "Scale", Kinds: []string{"deployments"}, Order: 100, Danger: DangerMutating,
		Targets:  func(_ *Session, p menuScale) ([]model.ObjectRef, error) { return []model.ObjectRef{p.Ref}, nil },
		Escalate: func(_ *Session, p menuScale) bool { return p.Replicas == 0 },
		Prefill:  func(_ *Session, p menuScale) (menuScale, error) { p.Replicas = 3; return p, nil },
		Run:      func(context.Context, *Session, menuScale) (any, error) { return nil, nil }}))
	must(Register(r, Spec[menuContainer]{ID: "probe.logs", Label: "Logs", Kinds: []string{"pods"}, Order: 500, Danger: DangerRead,
		Run: func(context.Context, *Session, menuContainer) (any, error) { return nil, nil }}))
	must(Register(r, Spec[menuRef]{ID: "probe.edit", Label: "Edit", Order: OrderEdit, Danger: DangerRead, Run: run}))
	must(Register(r, Spec[menuRef]{ID: "probe.delete", Label: "Delete", Order: OrderDelete, Danger: DangerDestructive, Targets: targets, Run: run}))
	must(Register(r, Spec[menuRef]{ID: "probe.services", Label: "Forward", Kinds: []string{"services"}, Order: 100, Danger: DangerRead, Run: run}))
	must(Register(r, Spec[menuRef]{ID: "probe.unordered", Label: "Not in menus", Kinds: []string{"nodes"}, Danger: DangerRead, Run: run}))
	return r
}

func TestActionsOrderAndKinds(t *testing.T) {
	r := menuRegistry(t)
	ids := func(kind string) string {
		var out []string
		for _, c := range r.Actions(kind) {
			out = append(out, c.ID)
		}
		return strings.Join(out, ",")
	}
	// Order ascending, ties by id; Edit and Delete (no Kinds) on every kind; Order 0 never.
	if got := ids("nodes"); got != "probe.cordon,probe.uncordon,probe.drain,probe.edit,probe.delete" {
		t.Errorf("nodes: %s", got)
	}
	if got := ids("deployments"); got != "probe.scale,probe.edit,probe.delete" {
		t.Errorf("deployments: %s", got)
	}
	if got := ids("configmaps"); got != "probe.edit,probe.delete" {
		t.Errorf("a kind without actions of its own: %s", got)
	}
}

func TestApplicableHidesOneOfAPair(t *testing.T) {
	r := menuRegistry(t)
	cordon, _ := r.Get("probe.cordon")
	uncordon, _ := r.Get("probe.uncordon")
	drain, _ := r.Get("probe.drain")
	for _, tt := range []struct {
		node                 string
		cordonOn, uncordonOn bool
	}{{"worker-1", true, false}, {"worker-1-cordoned", false, true}} {
		ref := model.ObjectRef{Kind: "nodes", Name: tt.node}
		if got := cordon.Applicable(nil, cordon.ObjectParams(ref, "")); got != tt.cordonOn {
			t.Errorf("%s: cordon applicable %v", tt.node, got)
		}
		if got := uncordon.Applicable(nil, uncordon.ObjectParams(ref, "")); got != tt.uncordonOn {
			t.Errorf("%s: uncordon applicable %v", tt.node, got)
		}
		// Disabled is not hidden: Drain applies, it just cannot run now.
		if p := drain.ObjectParams(ref, ""); !drain.Applicable(nil, p) || drain.Enabled(nil, p) == nil {
			t.Errorf("%s: drain should be shown greyed", tt.node)
		}
	}
}

func TestDialogFieldsAndObjectParams(t *testing.T) {
	r := menuRegistry(t)
	scale, _ := r.Get("probe.scale")
	logs, _ := r.Get("probe.logs")
	del, _ := r.Get("probe.delete")
	// A command with a Prefill shows its optional fields too (Note); one without, its required ones.
	if f := scale.DialogFields(); strings.Join(f, ",") != "replicas,note" {
		t.Errorf("scale dialog fields %v, want replicas and note", f)
	}
	if f := logs.DialogFields(); len(f) != 0 || !logs.TakesContainer() || del.TakesContainer() || len(del.DialogFields()) != 0 {
		t.Errorf("logs fields %v container %v; delete container %v", f, logs.TakesContainer(), del.TakesContainer())
	}
	ref := model.ObjectRef{Context: "test", Kind: "deployments", Namespace: "shop", Name: "web"}
	params := scale.ObjectParams(ref, "ignored")
	// The stub validates: Enabled runs instead of failing on the missing replicas.
	if err := scale.Enabled(nil, params); err != nil {
		t.Fatalf("enabled with stub params: %v", err)
	}
	pre, err := scale.Prefill(nil, params)
	if err != nil {
		t.Fatal(err)
	}
	var got menuScale
	if err := json.Unmarshal(pre, &got); err != nil || got.Replicas != 3 || got.Ref != ref {
		t.Errorf("prefill %s %v", pre, err)
	}
	var withContainer map[string]any
	_ = json.Unmarshal(logs.ObjectParams(model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}, "envoy"), &withContainer)
	if withContainer["container"] != "envoy" {
		t.Errorf("container params %v", withContainer)
	}
}

type menuScaleOptional struct {
	Ref      model.ObjectRef `json:"ref"`
	Replicas *uint16         `json:"replicas,omitempty"`
}

// TestDialogFieldsOfAPrefilledOptionalParameter: a command whose dialog parameter is optional in
// its schema but filled by Prefill (the real Scale) still gets the dialog, and its stub leaves the
// parameter out so that Prefill fills it from the cache.
func TestDialogFieldsOfAPrefilledOptionalParameter(t *testing.T) {
	r := NewRegistry()
	if err := Register(r, Spec[menuScaleOptional]{ID: "probe.scale", Label: "Scale", Kinds: []string{"deployments"}, Order: 100,
		Danger:  DangerMutating,
		Targets: func(_ *Session, p menuScaleOptional) ([]model.ObjectRef, error) { return []model.ObjectRef{p.Ref}, nil },
		Prefill: func(_ *Session, p menuScaleOptional) (menuScaleOptional, error) {
			if p.Replicas == nil {
				n := uint16(4)
				p.Replicas = &n
			}
			return p, nil
		},
		Run: func(context.Context, *Session, menuScaleOptional) (any, error) { return nil, nil }}); err != nil {
		t.Fatal(err)
	}
	c, _ := r.Get("probe.scale")
	if f := c.DialogFields(); len(f) != 1 || f[0] != "replicas" || c.Requires("replicas") {
		t.Fatalf("dialog fields %v, requires %v", f, c.Requires("replicas"))
	}
	params := c.ObjectParams(model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"}, "")
	if strings.Contains(string(params), "replicas") {
		t.Errorf("stub %s sets the optional replicas", params)
	}
	pre, err := c.Prefill(nil, params)
	if err != nil || !strings.Contains(string(pre), `"replicas":4`) {
		t.Errorf("prefill %s %v", pre, err)
	}
}
