package engine

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The palette's search: case-insensitive, every term must match, the label's start ranks first,
// the category and a fuzzy match count too, and greyed entries come after those that can run.
func TestFilterPalette(t *testing.T) {
	entries := []PaletteEntry{
		{Command: "pod.logs", Label: "Logs", Category: "web-0"},
		{Command: "object.delete", Label: "Delete", Category: "web-0"},
		{Command: "ui.openView", Label: "Switch to Deployments", Category: "Workloads"},
		{Command: "ui.openView", Label: "Switch to Pods", Category: "Workloads"},
		{Command: "ui.openView", Label: "Switch to Daemon Sets", Category: "Workloads", Reason: "no cluster connected"},
		{Command: "ui.openView", Label: "Switch to Deployment Controllers", Category: "Custom"},
		{Command: "hotbar.addCluster", Label: "Add to Hotbar", Category: "Hotbar"},
		{Command: "terminal.open", Label: "New Terminal", Category: "View"},
	}
	for _, tt := range []struct {
		query string
		want  []string
	}{
		{"", labels(entries)},
		{"   ", labels(entries)},
		{"DEP", []string{"Switch to Deployments", "Switch to Deployment Controllers"}},
		{"de", []string{"Delete", "Switch to Deployments", "Switch to Deployment Controllers", "Switch to Daemon Sets"}},
		{"swdep", []string{"Switch to Deployments", "Switch to Deployment Controllers"}},
		{"hotbar add", []string{"Add to Hotbar"}},
		{"web", []string{"Logs", "Delete"}},
		{"term", []string{"New Terminal"}},
		{"zzz", nil},
		{"switch zzz", nil},
	} {
		if got := labels(FilterPalette(entries, tt.query)); !slices.Equal(got, tt.want) {
			t.Errorf("%q: %q, want %q", tt.query, got, tt.want)
		}
	}
}

func labels(entries []PaletteEntry) []string {
	var out []string
	for _, e := range entries {
		out = append(out, e.Label)
	}
	return out
}

// The palette's object is the drawer's, else the one selected object; several selected objects
// are none, as for the row menu.
func TestPaletteObject(t *testing.T) {
	a := model.ObjectRef{Kind: "pods", Namespace: "x", Name: "a"}
	b := model.ObjectRef{Kind: "pods", Namespace: "x", Name: "b"}
	for _, tt := range []struct {
		st   UIState
		want string
	}{
		{UIState{}, ""},
		{UIState{Selected: []model.ObjectRef{a}}, "a"},
		{UIState{Selected: []model.ObjectRef{a, b}}, ""},
		{UIState{Selected: []model.ObjectRef{a, b}, Drawer: &b}, "b"},
	} {
		got, ok := PaletteObject(tt.st)
		if (tt.want == "") == ok || got.Name != tt.want {
			t.Errorf("%+v: %v %v, want %q", tt.st, got, ok, tt.want)
		}
	}
}

// A test registry's palette: a command that does not apply is left out, one that cannot run keeps
// its reason, a panic in Enabled becomes a reason, a container command gets an entry per container,
// and parameters with a required field come only with the caller's dialog.
func TestPaletteRules(t *testing.T) {
	r := NewRegistry()
	type refP struct {
		Ref model.ObjectRef `json:"ref"`
	}
	type contP struct {
		Ref       model.ObjectRef `json:"ref"`
		Container string          `json:"container,omitempty"`
	}
	type nameP struct {
		Name string `json:"name"`
	}
	type optP struct {
		Count int `json:"count,omitempty"`
	}
	nop := func(context.Context, *Session, refP) (any, error) { return nil, nil }
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(Register(r, Spec[refP]{ID: "x.hidden", Label: "Hidden", Kinds: []string{"pods"}, Order: 10, Danger: DangerUI, Run: nop,
		Enabled: func(*Session, refP) error { return ErrNotApplicable }}))
	must(Register(r, Spec[refP]{ID: "x.greyed", Label: "Greyed", Kinds: []string{"pods"}, Order: 20, Danger: DangerUI, Run: nop,
		Enabled: func(*Session, refP) error { return errors.New("not now") }}))
	must(Register(r, Spec[refP]{ID: "x.panics", Label: "Panics", Kinds: []string{"pods"}, Order: 30, Danger: DangerUI, Run: nop,
		Enabled: func(*Session, refP) error { panic("boom") }}))
	must(Register(r, Spec[contP]{ID: "x.logs", Label: "Logs", Kinds: []string{"pods"}, Order: 40, Danger: DangerRead,
		Run: func(context.Context, *Session, contP) (any, error) { return nil, nil }}))
	must(Register(r, Spec[nameP]{ID: "x.named", Label: "Named", MenuPath: "X/Named", Danger: DangerUI,
		Run: func(context.Context, *Session, nameP) (any, error) { return nil, nil }}))
	must(Register(r, Spec[optP]{ID: "x.optional", Label: "Optional", MenuPath: "X/Optional", Danger: DangerUI,
		Run: func(context.Context, *Session, optP) (any, error) { return nil, nil }}))
	must(Register(r, Spec[struct{}]{ID: "x.agentOnly", Label: "Agent Only", Danger: DangerUI,
		Run: func(context.Context, *Session, struct{}) (any, error) { return nil, nil }}))

	pod := model.ObjectRef{Kind: "pods", Namespace: "x", Name: "web-0"}
	in := PaletteInput{State: UIState{Drawer: &pod}, Containers: []string{"app", "sidecar"}}
	got := r.Palette(nil, in)
	var rows []string
	for _, e := range got {
		rows = append(rows, e.Command+":"+e.Label+":"+e.Reason)
	}
	want := []string{"x.greyed:Greyed:not now", "x.panics:Panics:unavailable: boom", "x.logs:Logs · app:", "x.logs:Logs · sidecar:",
		"x.optional:Optional:"}
	if !slices.Equal(rows, want) {
		t.Errorf("entries\n%s\nwant\n%s", strings.Join(rows, "\n"), strings.Join(want, "\n"))
	}
	if got[2].Container != "app" || !strings.Contains(string(got[2].Params), `"container":"app"`) {
		t.Errorf("container entry %+v", got[2])
	}

	in.Runs = func(id string) bool { return id == "x.named" }
	in.Containers = []string{"app"}
	got = r.Palette(nil, in)
	named := slices.IndexFunc(got, func(e PaletteEntry) bool { return e.Command == "x.named" })
	if named < 0 || !got[named].Own || got[named].Category != "X" {
		t.Errorf("x.named with a dialog: %+v", got)
	}
	if i := slices.IndexFunc(got, func(e PaletteEntry) bool { return e.Command == "x.logs" }); i < 0 || got[i].Label != "Logs" || got[i].Container != "" {
		t.Errorf("one container: %+v", got)
	}
}
