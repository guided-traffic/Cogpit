package engine

import (
	"errors"
	"slices"
	"testing"
	"time"
)

// The dock per cluster (D-69 item 9, dockscope.go).

// dockIDs are the ids of tabs.
func dockIDs(tabs []DockTab) []string {
	out := make([]string, 0, len(tabs))
	for _, t := range tabs {
		out = append(out, t.ID)
	}
	return out
}

// addDockTab opens a tab of a cluster (none for "") and makes it active, as the commands that open
// tabs do.
func addDockTab(s *Session, id, contextID string) UIState {
	return s.UpdateUI(func(u *UIState) {
		u.Dock = append(u.Dock, DockTab{ID: id, Type: DockTerminal, Title: id, Context: contextID})
		u.ActiveDock = id
	})
}

// The dock shows the shown cluster's tabs and those of no cluster; showing another cluster keeps
// the left one's active tab and open state and gives them back when it is shown again, with every
// tab still in Dock (its stream running). A cluster shown the first time keeps a tab of no cluster
// that was active, collapsed only when it shows none of its own.
func TestDockFollowsTheShownCluster(t *testing.T) {
	s, _ := poolSession(t, time.Hour, "alpha", "beta")
	mustConnect(t, s, "alpha")
	addDockTab(s, "terminal-1", "alpha")
	addDockTab(s, "logs-2", "alpha")
	u := addDockTab(s, "audit-3", "") // the Audit Log belongs to no cluster
	if u.ActiveDock != "audit-3" || !u.DockOpen {
		t.Fatalf("alpha's dock: active %q, open %v", u.ActiveDock, u.DockOpen)
	}
	s.UpdateUI(func(u *UIState) { u.ActiveDock = "logs-2" })

	mustConnect(t, s, "beta")
	u = s.UIState()
	if got := dockIDs(u.DockInScope()); !slices.Equal(got, []string{"audit-3"}) {
		t.Errorf("beta's dock shows %q", got)
	}
	if len(u.Dock) != 3 {
		t.Errorf("Dock lost alpha's tabs: %q", dockIDs(u.Dock))
	}
	if u.ActiveDock != "audit-3" || u.DockOpen {
		t.Errorf("beta shown the first time: active %q, open %v; want the Audit Log, collapsed", u.ActiveDock, u.DockOpen)
	}
	if err := s.DockToggleUnavailable(); err != nil {
		t.Errorf("the Audit Log is beta's too, the dock can open: %v", err)
	}
	u = addDockTab(s, "terminal-4", "beta")
	if u.ActiveDock != "terminal-4" || !u.DockOpen {
		t.Errorf("beta's own tab: active %q, open %v", u.ActiveDock, u.DockOpen)
	}

	mustConnect(t, s, "alpha") // only shown: alpha is connected
	u = s.UIState()
	if got := dockIDs(u.DockInScope()); !slices.Equal(got, []string{"terminal-1", "logs-2", "audit-3"}) {
		t.Errorf("alpha's dock shows %q", got)
	}
	if u.ActiveDock != "logs-2" || !u.DockOpen {
		t.Errorf("alpha back: active %q, open %v; want logs-2, open", u.ActiveDock, u.DockOpen)
	}
	if active, open := u.dockMemo["beta"].active, u.dockMemo["beta"].open; active != "terminal-4" || !open {
		t.Errorf("beta's dock kept as %q, open %v", active, open)
	}

	collapse := false
	if _, err := s.SetLayout(LayoutDock, &collapse); err != nil {
		t.Fatal(err)
	}
	mustConnect(t, s, "beta")
	if u = s.UIState(); u.ActiveDock != "terminal-4" || !u.DockOpen {
		t.Errorf("beta back: active %q, open %v", u.ActiveDock, u.DockOpen)
	}
	mustConnect(t, s, "alpha")
	if u = s.UIState(); u.ActiveDock != "logs-2" || u.DockOpen {
		t.Errorf("alpha's collapsed dock came back as %q, open %v", u.ActiveDock, u.DockOpen)
	}
}

// A tab of a cluster the window does not show, opened by an agent, goes to that cluster's dock:
// the shown dock keeps its tab and its open state; the tab is active, its dock open, when its
// cluster is shown. Closing the shown active tab gives way to a tab the dock shows, never to the
// hidden cluster's; a hidden cluster whose tabs all close keeps nothing.
func TestAHiddenClustersTabStaysInItsDock(t *testing.T) {
	s, _ := poolSession(t, time.Hour, "alpha", "beta")
	mustConnect(t, s, "alpha")
	mustConnect(t, s, "beta")
	addDockTab(s, "terminal-1", "beta")
	collapse := false
	if _, err := s.SetLayout(LayoutDock, &collapse); err != nil {
		t.Fatal(err)
	}
	u := addDockTab(s, "logs-2", "alpha") // an agent's pod.logs on alpha
	if u.ActiveDock != "terminal-1" || u.DockOpen {
		t.Errorf("the shown dock: active %q, open %v; want terminal-1, collapsed", u.ActiveDock, u.DockOpen)
	}
	if m := u.dockMemo["alpha"]; m.active != "logs-2" || !m.open {
		t.Errorf("alpha's dock: %+v", m)
	}
	addDockTab(s, "terminal-3", "beta")
	u = s.UpdateUI(func(u *UIState) { // dock.close of the active tab picks its neighbour, here alpha's
		u.Dock = slices.DeleteFunc(u.Dock, func(t DockTab) bool { return t.ID == "terminal-3" })
		u.ActiveDock = "logs-2"
	})
	if u.ActiveDock != "terminal-1" {
		t.Errorf("after closing the active tab: %q, want beta's terminal-1", u.ActiveDock)
	}
	if m := u.dockMemo["alpha"]; m.active != "logs-2" {
		t.Errorf("the neighbour pick changed alpha's dock: %+v", m)
	}

	mustConnect(t, s, "alpha")
	if u = s.UIState(); u.ActiveDock != "logs-2" || !u.DockOpen {
		t.Errorf("alpha shown: active %q, open %v", u.ActiveDock, u.DockOpen)
	}
	u = s.UpdateUI(func(u *UIState) { // beta's last tab closes while alpha is shown
		u.Dock = slices.DeleteFunc(u.Dock, func(t DockTab) bool { return t.ID == "terminal-1" })
	})
	if _, kept := u.dockMemo["beta"]; kept {
		t.Errorf("beta keeps a dock without tabs: %+v", u.dockMemo)
	}
	mustConnect(t, s, "beta")
	if u = s.UIState(); u.ActiveDock != "" || u.DockOpen {
		t.Errorf("beta without tabs: active %q, open %v", u.ActiveDock, u.DockOpen)
	}
	if err := s.DockToggleUnavailable(); !errors.Is(err, ErrNoDockTabs) {
		t.Errorf("beta's empty dock, alpha's logs hidden: %v, want ErrNoDockTabs", err)
	}
}

// The memo of the docks lives in UIState: a clone does not share it, so an old state stays as it
// was.
func TestDockMemoIsCloned(t *testing.T) {
	u := UIState{Context: "alpha", Dock: []DockTab{{ID: "a", Context: "alpha"}, {ID: "b", Context: "beta"}}, ActiveDock: "a", DockOpen: true}
	old := u.Clone()
	u.Context = "beta"
	u.SyncDock(old)
	if old.dockMemo != nil {
		t.Errorf("the old state got a memo: %+v", old.dockMemo)
	}
	c := u.Clone()
	c.dockMemo["alpha"] = dockMemo{}
	if u.dockMemo["alpha"].active != "a" {
		t.Errorf("a clone's change reached the state: %+v", u.dockMemo)
	}
	if u.ActiveDock != "b" || u.DockOpen {
		t.Errorf("beta shown: active %q, open %v", u.ActiveDock, u.DockOpen)
	}
}
