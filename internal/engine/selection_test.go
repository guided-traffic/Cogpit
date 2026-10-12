package engine

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func cm(name string) model.ObjectRef {
	return model.ObjectRef{Context: "test", Kind: "configmaps", Namespace: "shop", Name: name}
}

func names(refs []model.ObjectRef) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.Name
	}
	return out
}

// TestApplySelection: the four modes keep the selection's order, add in the given order, never hold
// an object twice, and leave their inputs alone.
func TestApplySelection(t *testing.T) {
	cur := []model.ObjectRef{cm("a"), cm("b")}
	for _, tc := range []struct {
		mode SelectMode
		refs []model.ObjectRef
		want []string
	}{
		{SelectReplace, []model.ObjectRef{cm("c"), cm("a"), cm("c")}, []string{"c", "a"}},
		{SelectReplace, nil, []string{}},
		{SelectAdd, []model.ObjectRef{cm("c"), cm("a")}, []string{"a", "b", "c"}},
		{SelectToggle, []model.ObjectRef{cm("a"), cm("c")}, []string{"b", "c"}},
		{SelectToggle, []model.ObjectRef{cm("a"), cm("a")}, []string{"b"}},
		{SelectRemove, []model.ObjectRef{cm("b"), cm("x")}, []string{"a"}},
	} {
		got := ApplySelection(cur, tc.refs, tc.mode)
		if !slices.Equal(names(got), tc.want) {
			t.Errorf("%s %v: %v, want %v", tc.mode, names(tc.refs), names(got), tc.want)
		}
	}
	// A reference without a context is the same object as the connected cluster's.
	bare := cm("a")
	bare.Context = ""
	if got := ApplySelection(cur, []model.ObjectRef{bare}, SelectToggle); !slices.Equal(names(got), []string{"b"}) {
		t.Errorf("toggle without a context: %v", names(got))
	}
	if !slices.Equal(names(cur), []string{"a", "b"}) {
		t.Errorf("the input changed: %v", names(cur))
	}
	for in, want := range map[string]SelectMode{"": SelectReplace, "add": SelectAdd, "toggle": SelectToggle, "remove": SelectRemove} {
		if got, err := ParseSelectMode(in); err != nil || got != want {
			t.Errorf("ParseSelectMode(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := ParseSelectMode("extend"); err == nil {
		t.Error("an unknown mode was accepted")
	}
}

// TestSelectionActionsLeaveTheRowMenus: a command that takes a selection ("refs", required, with an
// Order) is in a selection's menu for the kinds it applies to, never in one object's menu.
func TestSelectionActionsLeaveTheRowMenus(t *testing.T) {
	r := NewRegistry()
	type refs struct {
		Refs []model.ObjectRef `json:"refs"`
	}
	type one struct {
		Ref model.ObjectRef `json:"ref"`
	}
	targets := func(*Session, refs) ([]model.ObjectRef, error) { return nil, nil }
	run := func(context.Context, *Session, refs) (any, error) { return nil, nil }
	for _, err := range []error{
		Register(r, Spec[refs]{ID: "object.deleteMany", Order: OrderDelete, Danger: DangerDestructive, Targets: targets, Run: run}),
		Register(r, Spec[refs]{ID: "deployment.restartMany", Kinds: []string{"deployments"}, Order: 200, Danger: DangerDestructive, Targets: targets, Run: run}),
		Register(r, Spec[one]{ID: "object.delete", Order: OrderDelete, Danger: DangerDestructive,
			Targets: func(*Session, one) ([]model.ObjectRef, error) { return nil, nil },
			Run:     func(context.Context, *Session, one) (any, error) { return nil, nil }}),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}
	ids := func(cs []*Command) []string {
		out := make([]string, len(cs))
		for i, c := range cs {
			out[i] = c.ID
		}
		return out
	}
	if got := ids(r.Actions("deployments")); !slices.Equal(got, []string{"object.delete"}) {
		t.Errorf("one object's actions %v", got)
	}
	if got := ids(r.SelectionActions([]string{"deployments"})); !slices.Equal(got, []string{"deployment.restartMany", "object.deleteMany"}) {
		t.Errorf("a selection of deployments: %v", got)
	}
	if got := ids(r.SelectionActions([]string{"deployments", "pods"})); !slices.Equal(got, []string{"object.deleteMany"}) {
		t.Errorf("a selection of deployments and pods: %v", got)
	}
	if got := string(SelectionParams(nil)); got != `{"refs":[]}` {
		t.Errorf("SelectionParams(nil) = %s", got)
	}
	if got := SelectionKinds([]model.ObjectRef{cm("a"), {Kind: "pods"}, cm("b")}); !slices.Equal(got, []string{"configmaps", "pods"}) {
		t.Errorf("SelectionKinds %v", got)
	}
}

// TestConfirmMessageOfManyObjects: a confirmation of several objects counts them, names the first
// ConfirmNamed and the count of the rest, and every namespace.
func TestConfirmMessageOfManyObjects(t *testing.T) {
	req := ConfirmRequest{Label: "Delete", Context: "kind-cogpit", Namespaces: []string{"shop"}}
	for i := range 3 {
		req.Objects = append(req.Objects, cm(string(rune('a'+i))))
	}
	if got, want := req.Message(), "Delete 3 objects: configmaps a, configmaps b, configmaps c in namespace shop (context kind-cogpit)?"; got != want {
		t.Errorf("message %q, want %q", got, want)
	}
	req.Objects = nil
	for i := range 25 {
		req.Objects = append(req.Objects, cm("cm-"+string(rune('a'+i))))
	}
	req.Namespaces = []string{"shop", "infra"}
	msg := req.Message()
	if !strings.HasPrefix(msg, "Delete 25 objects: configmaps cm-a, ") || !strings.Contains(msg, "configmaps cm-j and 15 more in namespaces shop, infra (context kind-cogpit)?") ||
		strings.Contains(msg, "cm-k") {
		t.Errorf("message %q", msg)
	}
	one := ConfirmRequest{Label: "Delete", Context: "c", Objects: []model.ObjectRef{cm("a")}, Namespaces: []string{"shop"}}
	if got := one.Message(); got != "Delete configmaps a in namespace shop (context c)?" {
		t.Errorf("one object: %q", got)
	}
}
