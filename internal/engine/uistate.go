package engine

import (
	"slices"

	"github.com/guided-traffic/cogpit/internal/model"
)

// UIState is the state of the user interface, held by the engine so that agents see and drive the
// same state as the user (architecture §4.3). The GUI changes it only through ui.* commands.
type UIState struct {
	// Revision increases with every change.
	Revision uint64 `json:"revision"`
	// Context is the kubeconfig context of the active cluster, empty before connecting.
	Context string `json:"context"`
	// Connection is the state of the connection to Context.
	Connection Connection `json:"connection"`
	// View is the open view's kind key ("pods") or page ("overview").
	View string `json:"view"`
	// Namespaces filter namespaced views; empty means all namespaces.
	Namespaces []string `json:"namespaces"`
	// Filter is the view's search text.
	Filter string `json:"filter"`
	// Sort is the open view's sort order.
	Sort SortSpec `json:"sort"`
	// Selected are the selected objects of the open view.
	Selected []model.ObjectRef `json:"selected"`
	// Drawer is the object the detail drawer shows; nil when it is closed.
	Drawer *model.ObjectRef `json:"drawer,omitempty"`
	// Dock lists the dock's tabs; ActiveDock is the id of the shown one.
	Dock       []DockTab `json:"dock"`
	ActiveDock string    `json:"activeDock,omitempty"`
}

// Connection is the state of the active cluster connection.
type Connection struct {
	State ConnectionState `json:"state"`
	Error string          `json:"error,omitempty"` // set in ConnectionError; never contains a credential
	Demo  bool            `json:"demo,omitempty"`  // the synthetic demo cluster
}

// ConnectionState is a connection's lifecycle state.
type ConnectionState string

// Connection states.
const (
	Disconnected ConnectionState = "disconnected"
	Connecting   ConnectionState = "connecting"
	Connected    ConnectionState = "connected"
	ConnectError ConnectionState = "error"
)

// SortSpec is a table's sort order.
type SortSpec struct {
	Column string `json:"column" jsonschema:"column id, e.g. name, age, restarts"`
	Desc   bool   `json:"desc,omitempty" jsonschema:"descending order"`
}

// DockTab is one tab of the bottom dock.
type DockTab struct {
	ID        string           `json:"id"`
	Type      DockType         `json:"type"`
	Title     string           `json:"title"`
	Context   string           `json:"context,omitempty"`
	Ref       *model.ObjectRef `json:"ref,omitempty"`       // the pod for logs and shells
	Container string           `json:"container,omitempty"` // the container for logs and shells
}

// DockType is a dock tab's kind.
type DockType string

// Dock tab types.
const (
	DockLogs     DockType = "logs"     // a container's log stream
	DockTerminal DockType = "terminal" // a local shell bound to the context
	DockShell    DockType = "shell"    // a shell in a container
	DockEdit     DockType = "edit"     // a YAML editor on one object (object.edit)
	DockCreate   DockType = "create"   // a YAML editor for new objects (resource.createTab)
)

// Clone returns a deep copy.
func (u UIState) Clone() UIState {
	out := u
	out.Namespaces = slices.Clone(u.Namespaces)
	out.Selected = slices.Clone(u.Selected)
	if u.Drawer != nil {
		d := *u.Drawer
		out.Drawer = &d
	}
	out.Dock = make([]DockTab, len(u.Dock))
	for i, t := range u.Dock {
		out.Dock[i] = t
		if t.Ref != nil {
			r := *t.Ref
			out.Dock[i].Ref = &r
		}
	}
	return out
}

// DockTab returns the dock tab with the given id.
func (u UIState) DockTab(id string) (DockTab, bool) {
	for _, t := range u.Dock {
		if t.ID == id {
			return t, true
		}
	}
	return DockTab{}, false
}
