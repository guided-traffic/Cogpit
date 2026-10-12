package engine

import "fmt"

// Disconnect closes the connection to the connected cluster, or ends a connect under way
// (cluster.disconnect, the Cluster menu's Disconnect). Nothing is sent to the cluster: the cache's
// informers stop, the cluster's port forwards stop (as on switching to another cluster), and the
// UI is left Disconnected without an error. Context keeps naming the cluster; its tabs stay in the
// tab bar and connect to it again when one is selected, and the workspace shows the welcome page
// (UIState.Welcome). The cluster's connect log notes the disconnect.
//
// A connect started after Disconnect wins: Disconnect then leaves the new connection alone.
func (s *Session) Disconnect() error {
	u := s.UIState()
	if u.Connection.State != Connected && u.Connection.State != Connecting {
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
	})
	return nil
}
