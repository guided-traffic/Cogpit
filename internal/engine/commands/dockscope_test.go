package commands_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// TestDockActivateShowsTheTabsCluster: the dock shows the shown cluster's tabs (D-69 item 9); an
// agent's terminal on the cluster not shown goes to that cluster's dock and dock.read's id finds it,
// and dock.activate of it shows its cluster first, without a connect.
func TestDockActivateShowsTheTabsCluster(t *testing.T) {
	h, pc := poolHarness(t, engine.Options{})
	var tab struct {
		Tab engine.DockTab `json:"tab"`
	}
	h.must(t, engine.OriginExternalAgent, "terminal.open", map[string]any{"context": "alpha"}, &tab)
	u := h.s.UIState()
	if tab.Tab.Context != "alpha" || u.Context != "beta" || u.ActiveDock == tab.Tab.ID || len(u.DockInScope()) != 0 {
		t.Fatalf("alpha's terminal %+v; shown %q, active %q, the dock shows %d tabs", tab.Tab, u.Context, u.ActiveDock, len(u.DockInScope()))
	}
	if _, ok := u.DockTab(tab.Tab.ID); !ok {
		t.Fatalf("the hidden tab %s is not in ui_state's dock", tab.Tab.ID)
	}
	count := func() int {
		pc.mu.Lock()
		defer pc.mu.Unlock()
		return pc.connects
	}
	connects := count()
	h.must(t, engine.OriginExternalAgent, "dock.activate", map[string]any{"id": tab.Tab.ID}, &u)
	if u.Context != "alpha" || u.ActiveDock != tab.Tab.ID || !u.DockOpen {
		t.Errorf("after dock.activate: shown %q, active %q, open %v", u.Context, u.ActiveDock, u.DockOpen)
	}
	if n := count(); n != connects {
		t.Errorf("dock.activate connected again: %d connects, %d before", n, connects)
	}
}
