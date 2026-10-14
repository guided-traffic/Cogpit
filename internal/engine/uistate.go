package engine

import (
	"maps"
	"slices"
	"time"

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
	// View is the open view's kind key ("pods") or page key ("workloads/overview", overview.go;
	// "cluster/overview", clusteroverview.go).
	View string `json:"view"`
	// Namespaces filter namespaced views; empty means all namespaces.
	Namespaces []string `json:"namespaces"`
	// Groups filter the Definitions view by API group; empty means all groups.
	Groups []string `json:"groups,omitempty"`
	// HiddenColumns are the ids of the columns each view hides, by view key; a view without an
	// entry hides the columns its kind hides by default (model.Column.Hidden).
	HiddenColumns map[string][]string `json:"hiddenColumns,omitempty"`
	// ColumnOrder are the column ids of each view in the order the user arranged them (dragging a
	// header, ui.setColumnOrder), by view key; a view without an entry shows its kind's order.
	ColumnOrder map[string][]string `json:"columnOrder,omitempty"`
	// Filter is the view's search text.
	Filter string `json:"filter"`
	// FilterOptions are the search field's toggles, match case and regular expression
	// (filteroptions.go).
	FilterOptions FilterOptions `json:"filterOptions"`
	// States is the open view's status filter: the states its rows must carry, as the view's
	// snapshot offers them (TableSnapshot.StateChoices; statusfilter.go); empty: every row. It
	// belongs to the view it was set for: opening another view clears it.
	States []string `json:"states,omitempty"`
	// statesView is the view States was set for (syncStates).
	statesView string
	// Sort is the open view's sort order.
	Sort SortSpec `json:"sort"`
	// Selected are the selected objects of the open view.
	Selected []model.ObjectRef `json:"selected"`
	// Drawer is the object the detail drawer shows; nil when it is closed.
	Drawer *model.ObjectRef `json:"drawer,omitempty"`
	// DrawerSort is the order of the drawer's sortable tables (a node's or a workload's Pods), by
	// the table's section title ("Pods"), as ui.sortDrawerTable sets it (drawersort.go). It stays
	// when the drawer shows another object and is the same in every tab; a table without an entry
	// is sorted by name.
	DrawerSort map[string]SortSpec `json:"drawerSort,omitempty"`
	// DrawerChartVersion is the version a Helm chart's drawer shows, as ui.setChartVersion sets it
	// (helmchartdrawer.go): its properties and README are that version's. Empty: the version the
	// chart is listed with. It ends when the drawer closes or shows another object.
	DrawerChartVersion string `json:"drawerChartVersion,omitempty"`
	// Dock lists the dock's tabs; ActiveDock is the id of the shown one.
	Dock       []DockTab `json:"dock"`
	ActiveDock string    `json:"activeDock,omitempty"`
	// Hotbar is the hotbar's banks and the shown one (hotbar.go); nil in a session without one.
	// Its tiles carry their contexts only; ui.state adds what each tile shows (Session.HotbarView).
	Hotbar *HotbarState `json:"hotbar,omitempty"`
	// Overview is what the cluster Overview's charts show: their time range and node group
	// (clusteroverview.go); zero: the defaults.
	Overview OverviewChoice `json:"overview,omitzero"`
	// Palette is set while the command palette is open (ui.palette.open, ⌘⇧P; palette.go). What
	// the user types in it stays in the window, as the context picker's filter does.
	Palette bool `json:"palette,omitempty"`
	// Chat is set while the built-in agent's chat column is shown (chat.toggle, ⌘I; chat.go).
	// What is said in it stays in the window: agents see only that it is open.
	Chat bool `json:"chat,omitempty"`
	// NavigatorScope is what the navigator shows (navscope.go): "section:local", the Local
	// Kubeconfigs section with every cluster Cogpit offers, or "cluster:<context id>", the
	// connected cluster's tree. The tab bar shows the tabs of the scope. The engine keeps it in step
	// with the connection; ui.setNavigatorScope sets it.
	NavigatorScope string `json:"navigatorScope"`
	// Welcome is set while the workspace shows the welcome page (welcome.go): no cluster is
	// connected and none is connecting, or one is and no tab is active. The engine derives it
	// after every change (syncWelcome); ui.welcome.open shows the page over a connected cluster.
	Welcome bool `json:"welcome,omitempty"`
	// Preferences are Cogpit's own settings (preferences.go); nil in a session that keeps none.
	Preferences *Preferences `json:"preferences,omitempty"`
	// LoginItem is Cogpit's start at login as the operating system reports it (loginitem.go); nil
	// where the platform offers none.
	LoginItem *LoginItemState `json:"loginItem,omitempty"`
	// TimeFormat is how the window writes absolute times (D-31): Go layouts of a date and a time
	// of day, from the computer's region settings ("02.01.2006", "15:04:05"), in the Preferences'
	// time zone. Agents read the drawn texts (view rows, drawer items) in it; structured times of
	// results are RFC 3339.
	TimeFormat model.TimeLayout `json:"timeFormat"`
	// ClusterSettings are the settings of every context that has some, by context id
	// (clustersettings.go): display names and icons the hotbar, the picker and the navigator show.
	ClusterSettings map[string]ClusterSettings `json:"clusterSettings,omitempty"`
	// Settings is the open settings page, Preferences (⌘,) or a context's Cluster settings (⌘⇧,);
	// nil while none is open.
	Settings *SettingsPage `json:"settings,omitempty"`
	// Notifications sums up the notification log, the toasts the user was shown (notifications.go):
	// how many, how many unread, the most severe unread level; notifications.list reads them.
	Notifications NotificationSummary `json:"notifications,omitzero"`
	// Tabs are the tab bar's tabs in their order, each a view of one cluster with its own state
	// (tabs.go); ActiveTab is the shown one's id, empty when none is shown. View, Namespaces,
	// Groups, Filter, FilterOptions, States, Sort, Selected and Drawer above are the active tab's.
	Tabs      []ViewTab `json:"tabs,omitempty"`
	ActiveTab string    `json:"activeTab,omitempty"`
	tabSeq    uint64    // the number of the last tab id handed out
	tabClock  uint64    // counts tab activations, for ViewTab.used
}

// Connection is the state of the active cluster connection.
type Connection struct {
	State ConnectionState `json:"state"`
	Error string          `json:"error,omitempty"` // set in ConnectError, and in Disconnected when Cogpit closed the connection (its kubeconfig file went away); never contains a credential
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
	// HelmInstall is a Helm Install tab's chart, version and defaults (helminstall.go).
	HelmInstall *HelmInstallTab `json:"helmInstall,omitempty"`
	// Kubectl is a terminal tab's kubectl: the check made when it opened, the offer of a matching
	// release and its download (kubectl.go).
	Kubectl *KubectlCheck `json:"kubectl,omitempty"`
	// Audit is the Audit Log tab's filter (auditlist.go).
	Audit AuditTab `json:"audit,omitzero"`
	// NodeShell is a node shell tab's node and pod (nodeshell.go).
	NodeShell *NodeShellTab `json:"nodeShell,omitempty"`
	// AgentRead is set on a terminal tab (terminal, shell, attach, node shell) whose output the
	// user lets agents read (dock.allowAgentRead, the tab's switch; D-44); off when it opens.
	// Agents read logs tabs without it (dockoutput.go).
	AgentRead bool `json:"agentRead,omitempty"`
	// AgentReadOn is when the switch went on; zero while it is off. AgentLastRead is when an agent
	// last read the tab since then; zero before the first read. AgentReadUntil is when the switch
	// turns itself off: AgentReadIdle after it went on or after the last read, at the latest
	// AgentReadMax after it went on; zero while it is off. AgentReadExpired is set once it turned
	// itself off, until the user turns it on again, and AgentReadCapped when that was at its
	// AgentReadMax rather than for want of a read (agentread.go, H-83).
	AgentReadOn      time.Time `json:"agentReadOn,omitzero"`
	AgentLastRead    time.Time `json:"agentLastRead,omitzero"`
	AgentReadUntil   time.Time `json:"agentReadUntil,omitzero"`
	AgentReadExpired bool      `json:"agentReadExpired,omitempty"`
	AgentReadCapped  bool      `json:"agentReadCapped,omitempty"`
}

// HelmUpgradeTab is what a Helm Upgrade tab shows beside its values editor (Lens's "Helm Upgrade:
// <name>" tab, captured 2026-10-12): the release's chart, the version the upgrade installs and the
// versions its picker offers. The tab's Ref is the release (kind helmreleases); its values are the
// tab's editor text (editor.get, editor.setText), which agents read withheld (model.Withheld).
// helm.release.upgradeTab with a repository or a version changes the choice (helmupgrade.go).
type HelmUpgradeTab struct {
	Chart string `json:"chart"`
	// Version is the version the upgrade installs: the release's own (its stored chart), or one of
	// Repository's.
	Version string `json:"version"`
	// ReleaseVersion is the release's own chart version, which the picker marks as the current one.
	ReleaseVersion string `json:"releaseVersion,omitempty"`
	// Repository is the chart repository whose versions the picker offers, "" when no repository
	// of Helm's repositories file lists the chart; Repositories are every one that does, in the file's
	// order.
	Repository   string   `json:"repository,omitempty"`
	Repositories []string `json:"repositories,omitempty"`
	// Versions are the picker's choices: the release's own chart version, and Repository's versions
	// newest first, prereleases after the others.
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
	// DockHelmInstall: a chart's values and version, installed by helm.chart.install (helminstall.go).
	DockHelmInstall DockType = "helmInstall"
)

// Clone returns a deep copy.
func (u UIState) Clone() UIState {
	out := u
	out.Namespaces = slices.Clone(u.Namespaces)
	out.Groups = slices.Clone(u.Groups)
	out.States = slices.Clone(u.States)
	if u.HiddenColumns != nil {
		out.HiddenColumns = make(map[string][]string, len(u.HiddenColumns))
		for view, ids := range u.HiddenColumns {
			out.HiddenColumns[view] = slices.Clone(ids)
		}
	}
	if u.ColumnOrder != nil {
		out.ColumnOrder = make(map[string][]string, len(u.ColumnOrder))
		for view, ids := range u.ColumnOrder {
			out.ColumnOrder[view] = slices.Clone(ids)
		}
	}
	out.Selected = slices.Clone(u.Selected)
	if u.Drawer != nil {
		d := *u.Drawer
		out.Drawer = &d
	}
	out.DrawerSort = maps.Clone(u.DrawerSort)
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
			h.Repositories = slices.Clone(t.Helm.Repositories)
			out.Dock[i].Helm = &h
		}
		if t.Kubectl != nil {
			k := t.Kubectl.clone()
			out.Dock[i].Kubectl = &k
		}
	}
	if u.Hotbar != nil {
		h := u.Hotbar.Clone()
		out.Hotbar = &h
	}
	if u.Preferences != nil {
		p := u.Preferences.Clone()
		out.Preferences = &p
	}
	if u.LoginItem != nil {
		l := *u.LoginItem
		out.LoginItem = &l
	}
	out.ClusterSettings = maps.Clone(u.ClusterSettings)
	if u.Settings != nil {
		p := *u.Settings
		out.Settings = &p
	}
	out.Tabs = cloneTabs(u.Tabs)
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
