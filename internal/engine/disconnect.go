package engine

import (
	"fmt"
	"slices"
)

// Disconnect closes the connection to the connected cluster, or ends a connect under way
// (cluster.disconnect, the Cluster menu's Disconnect). Nothing is sent to the cluster: the cache's
// informers stop, the cluster's port forwards stop (as on switching to another cluster), and the
// UI is left Disconnected without an error. Context keeps naming the cluster; its tabs close (the
// owner's A61, 2026-10-14 evening; until then they stayed and connected again when selected), and
// the workspace shows the welcome page (UIState.Welcome). The cluster's connect log notes the
// disconnect.
//
// A connect started after Disconnect wins: Disconnect then leaves the new connection alone.
func (s *Session) Disconnect() error {
	u := s.UIState()
	if !activeConnection(u.Connection.State) {
		return ErrNotConnected
	}
	st := s.state
	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return ErrClosed
	}
	if st.connectCancel != nil {
		st.connectCancel() // a connect under way returns soon and, superseded, leaves the UI alone
	}
	st.connectSeq++
	seq := st.connectSeq
	st.mu.Unlock()

	st.connMu.Lock() // after a pending connect gave up, as a superseding connect waits
	defer st.connMu.Unlock()
	if !st.current(seq) {
		return fmt.Errorf("disconnect: superseded by a newer connect")
	}
	st.mu.Lock()
	b := st.detachLocked()
	st.mu.Unlock()
	b.close(s.log)
	name := u.Context
	if b != nil {
		name = b.Context
	}
	if name != "" {
		st.logs.add(name, LogInfo, "Disconnected from "+name+" (Cluster › Disconnect)")
	}
	s.UpdateUI(func(u *UIState) {
		u.Connection = Connection{State: Disconnected}
		u.View, u.Filter = "", ""
		u.Selected, u.Drawer = nil, nil
		u.closeTabsOf(name)
	})
	return nil
}

// closeTabsOf closes the tab bar's tabs of a cluster (Disconnect, A61), the active one too: no tab
// is active afterwards. The dock's tabs stay: their streams end with the connection, and an
// editor's text is the user's until its tab is closed.
func (u *UIState) closeTabsOf(contextID string) {
	if contextID == "" {
		return
	}
	if i := u.activeIndex(); i >= 0 && u.Tabs[i].Context == contextID {
		u.ActiveTab = ""
	}
	u.Tabs = slices.DeleteFunc(u.Tabs, func(t ViewTab) bool { return t.Context == contextID })
	if len(u.Tabs) == 0 {
		u.Tabs = nil
	}
}
