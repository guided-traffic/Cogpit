package fyneui

import (
	"context"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// TestDrawerOnlyActionsStayOutOfRowMenus: a drawer-only command (Copy) is the drawer's first icon
// and not in the row menu, which Lens's captured row menus show without it.
func TestDrawerOnlyActionsStayOutOfRowMenus(t *testing.T) {
	f := kindsFake(t, "nodes")
	registerProbes(t, f)
	if err := engine.Register(f.reg, engine.Spec[probeRef]{ID: "object.copyName", Label: "Copy", Order: 10, DrawerOnly: true,
		Danger: engine.DangerUI, Run: func(context.Context, *engine.Session, probeRef) (any, error) { return nil, nil }}); err != nil {
		t.Fatal(err)
	}
	ref := model.ObjectRef{Context: "demo", Kind: "nodes", Name: cordonedNode}
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "the toolbar", func() bool { return len(sh.drawer.actions.Objects) == 5 })
	var tips []string
	for _, o := range sh.drawer.actions.Objects {
		tips = append(tips, o.(*tipButton).tip)
	}
	if want := "Copy|Uncordon|Drain|Edit|Delete"; strings.Join(tips, "|") != want {
		t.Errorf("toolbar %q, want %q", tips, want)
	}
	got := labels(sh.rowMenu(ref, sh.evaluateActions(ref, nil)))
	if want := "Show Details|Uncordon|Drain|Edit|Delete"; strings.Join(got, "|") != want {
		t.Errorf("row menu %q, want %q", got, want)
	}
}
