package engine

import (
	"context"
	"strings"
	"testing"
)

// TestDrawerOnlyCommands: a drawer-only command is among a kind's actions (the drawer's toolbar)
// and says so, so that the row menu can leave it out; it needs an Order to be anywhere.
func TestDrawerOnlyCommands(t *testing.T) {
	r := menuRegistry(t)
	run := func(context.Context, *Session, menuRef) (any, error) { return nil, nil }
	if err := Register(r, Spec[menuRef]{ID: "probe.copy", Label: "Copy", Order: 10, DrawerOnly: true, Danger: DangerUI, Run: run}); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, c := range r.Actions("nodes") {
		ids = append(ids, c.ID)
	}
	if got := strings.Join(ids, ","); got != "probe.copy,probe.cordon,probe.uncordon,probe.drain,probe.edit,probe.delete" {
		t.Errorf("node actions %s", got)
	}
	c, _ := r.Get("probe.copy")
	if !c.DrawerOnly {
		t.Error("the command does not say it is drawer-only")
	}
	if e, _ := r.Get("probe.edit"); e.DrawerOnly {
		t.Error("Edit is in the row menu too")
	}
	err := Register(r, Spec[menuRef]{ID: "probe.nowhere", Label: "Nowhere", DrawerOnly: true, Danger: DangerUI, Run: run})
	if err == nil || !strings.Contains(err.Error(), "needs an order") {
		t.Errorf("a drawer-only command without an order: %v", err)
	}
}
