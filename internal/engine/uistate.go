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
	// View is the open view's kind key ("pods") or page key ("workloads/overview", overview.go).
	View string `json:"view"`
	// Namespaces filter namespaced views; empty means all namespaces.
	Namespaces []string `json:"namespaces"`
	// Groups filter the Definitions view by API group; empty means all groups.
	Groups []string `json:"groups,omitempty"`
	// HiddenColumns are the ids of the columns each view hides, by view key; a view without an
	// entry hides the columns its kind hides by default (model.Column.Hidden).
	HiddenColumns map[string][]string `json:"hiddenColumns,omitempty"`
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
	Logs      LogOptions       `json:"logs,omitzero"`       // a logs tab's options (logs.go)
	Helm      *HelmUpgradeTab  `json:"helm,omitempty"`      // a Helm Upgrade tab's chart and version
}

// HelmUpgradeTab is what a Helm Upgrade tab shows beside its values editor (Lens's "Helm Upgrade:
// <name>" tab, captured 2026-10-12): the release's chart, the version the upgrade installs and the
// versions its picker offers. The tab's Ref is the release (kind helmreleases); its values are the
// tab's editor text (editor.get, editor.setText), which agents read withheld (model.Withheld).
type HelmUpgradeTab struct {
	Chart   string `json:"chart"`
	Version string `json:"version"`
	// Versions are the picker's choices: without chart repositories, the release's own chart
	// version only.
	Versions []string `json:"versions"`
	// UserSuppliedOnly: the editor holds the user-supplied values only, not the chart's defaults
	// merged in (Lens's "User-supplied values only").
	UserSuppliedOnly bool `json:"userSuppliedOnly"`
}

// DockType is a dock tab's kind.
type DockType string

// Dock tab types.
const (
	DockLogs     DockType = "logs"     // a container's log stream
	DockTerminal DockType = "terminal" // a local shell bound to the context
	DockShell    DockType = "shell"    // a shell in a container
	DockAttach   DockType = "attach"   // attached to a container's main process (pod.attach)
	DockEdit     DockType = "edit"     // a YAML editor on one object (object.edit)
	DockCreate   DockType = "create"   // a YAML editor for new objects (resource.createTab)
	// DockHelmUpgrade: a release's values and chart version, saved by helm.release.upgrade.
	DockHelmUpgrade DockType = "helmUpgrade"
)

// Clone returns a deep copy.
func (u UIState) Clone() UIState {
	out := u
	out.Namespaces = slices.Clone(u.Namespaces)
	out.Groups = slices.Clone(u.Groups)
	if u.HiddenColumns != nil {
		out.HiddenColumns = make(map[string][]string, len(u.HiddenColumns))
		for view, ids := range u.HiddenColumns {
			out.HiddenColumns[view] = slices.Clone(ids)
		}
	}
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
		if t.Helm != nil {
			h := *t.Helm
			h.Versions = slices.Clone(t.Helm.Versions)
			out.Dock[i].Helm = &h
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
