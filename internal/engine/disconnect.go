package engine

import "slices"

// Disconnect closes the connection to the cluster the window shows, or ends a connect under way
// (cluster.disconnect, the Cluster menu's Disconnect): DisconnectContext of the shown cluster.
func (s *Session) Disconnect() error { return s.DisconnectContext("") }

// DisconnectContext closes the connection to one connected cluster, "" the one the window shows
// (D-69: the others stay connected). Nothing is sent to the cluster: the cache's informers stop,
// the cluster's port forwards, logs and shells end, and its jobs stop. Its tabs close (the owner's
// A61, 2026-10-14 evening; until then they stayed and connected again when selected); the dock's
// tabs stay, their streams ended. For the shown cluster, or one whose connect is under way, the UI
// is left Disconnected without an error, Context keeps naming the cluster, and the workspace
// shows the welcome page (UIState.Welcome); for another one only its entry of Connections goes.
// The cluster's connect log notes the disconnect. A context that is not connected is a
// NotConnectedError.
//
// A connect started after Disconnect wins: Disconnect then leaves the new connection alone.
func (s *Session) DisconnectContext(contextID string) error {
	u := s.UIState()
	if contextID == "" {
		if !activeConnection(u.Connection.State) {
			return ErrNotConnected
		}
		contextID = u.Context
	}
	st := s.state
	st.mu.Lock()
	if st.closed {
		st.mu.Unlock()
		return ErrClosed
	}
	e := st.pool[contextID]
	shown := (u.Context == contextID && activeConnection(u.Connection.State)) || (e != nil && st.shown == e.b)
	if e == nil && !shown {
		st.mu.Unlock()
		return &NotConnectedError{Context: contextID}
	}
	if !shown {
		// Another connected cluster: the shown one and a connect under way are left alone. Decided
		// under the lock, so that a connect that shows it meanwhile makes this the shown case.
		b := st.removeLocked(contextID)
		st.mu.Unlock()
		b.close(s.log)
		s.noteDisconnect(contextID)
		s.UpdateUI(func(u *UIState) {
			u.dropConnection(contextID, e.gen)
			u.closeTabsOf(contextID)
		})
		return nil
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
		return errDisconnectSuperseded
	}
	st.mu.Lock()
	b := st.removeLocked(contextID) // nil for a connect under way, which has no backend yet
	st.mu.Unlock()
	b.close(s.log)
	s.noteDisconnect(contextID)
	s.UpdateUI(func(u *UIState) {
		u.dropConnection(contextID, 0)
		if u.Context == contextID {
			u.Connection = Connection{State: Disconnected}
			u.View, u.Filter = "", ""
			u.Selected, u.Drawer = nil, nil
		}
		u.closeTabsOf(contextID)
	})
	return nil
}

// noteDisconnect writes the user's disconnect into the context's connection log (D-66).
func (s *Session) noteDisconnect(contextID string) {
	a := s.state.logs.begin(contextID, AttemptDisconnect, LogInfo, "Disconnected from "+contextID+" (Cluster › Disconnect)")
	s.state.logs.finish(a, OutcomeOK, LogInfo, "")
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
