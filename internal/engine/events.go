package engine

// EventType says what an Event reports.
type EventType string

// Event types.
const (
	// EventUIState: the UI state changed; Event.UIState holds the new state.
	EventUIState EventType = "uiState"
	// EventTable: a new table snapshot for Event.View is available from Session.Table.
	EventTable EventType = "table"
	// EventDetail: the drawer object changed in the cache; re-read Session.Detail.
	EventDetail EventType = "detail"
	// EventToast: a message for the user (Event.Level, Event.Message).
	EventToast EventType = "toast"
	// EventDiagnostics: request counters or cache stats changed; re-read Session.Diagnostics.
	EventDiagnostics EventType = "diagnostics"
	// EventEditor: an editor tab's text or base version changed (editor.setText, a conflict);
	// Event.View holds the tab id, Event.Revision the text's revision. Re-read with editor.get.
	EventEditor EventType = "editor"
)

// Event is something the UI or an agent may react to. Events carry little data: the receiver
// reads the current state from the Session.
type Event struct {
	Type     EventType `json:"type"`
	View     string    `json:"view,omitempty"`
	Revision uint64    `json:"revision,omitempty"`
	UIState  *UIState  `json:"uiState,omitempty"`
	Level    string    `json:"level,omitempty"` // toast: "info", "warning", "error"
	Message  string    `json:"message,omitempty"`
}
