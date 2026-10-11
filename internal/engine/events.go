package engine

// EventType says what an Event reports.
type EventType string

// Event types.
const (
	// EventUIState: the UI state changed; Event.UIState holds the new state.
	EventUIState EventType = "uiState"
	// EventTable: a new table snapshot for Event.View is available from Session.Table.
	EventTable EventType = "table"
	// EventDetail: the drawer object changed in the cache, or access reviews answered (its actions'
	// Enabled may differ); re-read Session.Detail and the actions.
	EventDetail EventType = "detail"
	// EventToast: a message for the user (Event.Level, Event.Message).
	EventToast EventType = "toast"
	// EventDiagnostics: request counters or cache stats changed; re-read Session.Diagnostics.
	EventDiagnostics EventType = "diagnostics"
	// EventDrains: the connected context's pending drains changed (offered on connect, one
	// started, progressed, ended or was dismissed); re-read them with node.pendingDrains.
	// Event.Message, if set, says what to offer.
	EventDrains EventType = "drains"
	// EventKinds: the kinds the connected cluster serves changed (a custom resource definition was
	// added, changed or deleted), or access reviews answered whether the user may list them;
	// re-read kinds.list.
	EventKinds EventType = "kinds"
	// EventEditor: an editor tab's text or base version changed (editor.setText, a conflict);
	// Event.View holds the tab id, Event.Revision the text's revision. Re-read with editor.get.
	EventEditor EventType = "editor"
)

// Toast levels (Event.Level).
const (
	LevelInfo    = "info"
	LevelWarning = "warning"
	LevelError   = "error"
)

// Event is something the UI or an agent may react to. Events carry little data: the receiver
// reads the current state from the Session.
type Event struct {
	Type     EventType `json:"type"`
	View     string    `json:"view,omitempty"`
	Revision uint64    `json:"revision,omitempty"`
	UIState  *UIState  `json:"uiState,omitempty"`
	Level    string    `json:"level,omitempty"` // toast: LevelInfo, LevelWarning, LevelError
	Message  string    `json:"message,omitempty"`
}
