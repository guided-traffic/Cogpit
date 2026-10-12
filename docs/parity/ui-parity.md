# UI parity: layout, navigation, tables, dock and feel

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-12 · **Change:** minor (drawer charts, G12 third slice: Prometheus, Network and Filesystem, ranges to 30 days held by the engine; no count moved) · **Target:** Lens Desktop 2026.10.51501

How Cogpit's interface compares with Lens Desktop 2026.10.51501: the window layout, clusters and
hotbar, the cluster sidebar, list tables, the detail drawer, the bottom dock with its editor, log
and terminal tabs, the command palette, menus, keyboard shortcuts, preferences and cluster
settings. Resource views (columns and actions per kind) are in
[`resource-parity.md`](resource-parity.md); platforms, packaging and authentication in
[`platform-parity.md`](platform-parity.md); what Cogpit adds beyond Lens in
[`beyond-lens.md`](beyond-lens.md). Method: [`README.md`](README.md).

**Sources.** Lens Desktop's documentation under
[Using Lens](https://docs.lenshq.io/k8slens/using-lens/layout/) (layout, navigator, hotbar,
details panel, log options, terminal, advanced editor, shortcuts, preferences, overview), its
cluster pages ([cluster settings](https://docs.lenshq.io/k8slens/cluster/cluster-settings/),
[metrics](https://docs.lenshq.io/k8slens/cluster/cluster-metrics/),
[create resource](https://docs.lenshq.io/k8slens/cluster/create-resource/),
[view logs](https://docs.lenshq.io/k8slens/cluster/view-logs/)) and its release notes to
2026.10.51501. Lens Desktop was not run for this page: rows marked *capture needed* need a look at
an activated Lens Desktop ([D-12](../developer/decisions.md#d-12-parity-sources)).

**Plan column:** *free* (Lens Desktop's Personal plan) or *paid* (Plus, Pro or Enterprise).
Modifier names are macOS first (⌘ ⌥ ⇧ ⌃); Windows and Linux use Ctrl and Alt. **Status:** as
marked per row; rows without evidence are missing.

**Summary (measured, 2026-10-12, after the fourth implementation run, counted by `make parity`):**
56 of 133 rows exist (was 55): 55 of the 120 free-plan rows (47 partial, 8 done: the cell tooltip, right-click on a
row, the dock's "New tab" menu, keyboard navigation in tables and its shortcut row, the container
port's link, the forward dialog and the drawer's Forward…, each matching its captured row) and one
paid row (the diff view, partial); 7 more rows are not a target. The fourth run added Kubeconfig
Syncs and the Kubeconfig syncs preference (partial, their dialogs inferred); adding a kubeconfig
by pasting is not a target since the owner's decision D-21 (2026-10-12).
Remaining (estimated): see gaps [G5](gaps.md#g5-app-shell-navigation-tables-and-drawer) (layout,
sidebar, tables, drawer, dock), [G13](gaps.md#g13-clusters-hotbar-palette-menus-shortcuts-and-preferences)
(clusters, hotbar, palette, menus, shortcuts, preferences, cluster settings),
[G8](gaps.md#g8-logs-terminal-and-shells) and [G9](gaps.md#g9-yaml-editor-and-create-resource).

## Layout

Lens Desktop's regions ([layout](https://docs.lenshq.io/k8slens/using-lens/layout/)): title bar,
hotbar, navigator, tab bar, workspace, dock, status bar.

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Title bar with history back and forward, notifications and a context menu with Preferences; captured (2026-10-11): back and forward arrows, an "Update" button with a menu when an update is available, the paid "AI" button with a menu, a notifications bell, and on the right the vendor's "UPGRADE" and account avatar (not targets, D-9). Lens draws its bar into the window's title bar, beside the macOS window buttons (captured). Owner requirement (2026-10-11): Cogpit takes over the title-bar area too: no standard macOS title bar above Cogpit's top bar, the window draggable by Cogpit's own bar, and Cogpit's logo beside the window buttons, not below them. Fyne 2.8.1 hands out the `NSWindow` (`driver.NativeWindow.RunNative`, `MacWindowContext`), so a full-size content view with a transparent, hidden title is possible from `internal/ui` (verified in Fyne's source); whether dragging works without extra code is not verified | free | partial: built 2026-10-11, not yet seen in a real window (an offscreen render cannot show AppKit's title bar; the owner's smoke launch decides); history, notifications and the menu are missing | the AppKit bridge ([`titlebar_darwin.go`](../../internal/ui/fyneui/titlebar_darwin.go), [`.m`](../../internal/ui/fyneui/titlebar_darwin.m)) gives the window a full-size content view under a transparent title bar without a title, once the window exists (queued from the app's start); Cogpit's top bar keeps room for the window buttons from their frames (`standardWindowButton`: the logo starts as far right of the zoom button as the close button starts right of the window's edge), grows to the title bar's height if that is taller, and gives the room back in full screen (AppKit's full-screen, resize and screen notifications); a mouse-down on the bar where no button is starts a window drag (`performWindowDragWithEvent:`, then a posted mouse-up so that GLFW and Fyne do not keep the button pressed), the second click of a double click zooms, minimises or does nothing as `AppleActionOnDoubleClick` says ("Fill" zooms, inferred; the older `AppleMiniaturizeOnDoubleClick` when unset); Linux and Windows keep their native frames ([`titlebar_other.go`](../../internal/ui/fyneui/titlebar_other.go)); `COGPIT_SYSTEM_TITLEBAR=1` keeps the standard title bar. Tested offscreen: the setting's parsing, the room and height from a geometry, full screen, the drag area under the bar (`TestDoubleClickAction`, `TestTitlebarLayout`, `TestTopBarMakesRoomForWindowButtons`, `TestBarDragArea`); [PNG](../../internal/ui/fyneui/testdata/titlebar_room_dark.png) (the room is empty offscreen), [light](../../internal/ui/fyneui/testdata/titlebar_room_light.png) |
| Hotbar down the left edge ([Hotbar](#hotbar)); captured: one tile per cluster with its icon or image and a short label, and a bank switcher ("‹ 1 ›") at the bottom | paid | missing | – |
| Navigator with the cluster's resource tree ([Cluster sidebar](#cluster-sidebar)) | free | partial: its width is the user's (owner requirement, built 2026-10-12): drag the edge, 160 to 480 points, kept in the preferences; Lens's behaviour here is not captured | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go), [`panelwidths.go`](../../internal/ui/fyneui/panelwidths.go); `TestSidebarWidthByDraggingItsEdge`, `TestPanelWidthsSurviveARestart` |
| Navigator header: a dropdown of the scope the tree shows, one cluster or a section such as "Local Kubeconfigs", and a search button; the tree's root is the cluster, with its icon, a connection dot and a ›, and a ‹ to go back; hovering an item shows "<item> - <context>" (captures, 2026-10-11) | free | missing | – |
| Tab bar of cluster views: single click opens a preview tab, double-click (or ⇧↵) a fixed tab; a tab reads "<view> - <context>" with the cluster's icon and a close button; the preview tab's title is italic; hovering a tab shows its full title (captures, 2026-10-11; the italic-means-preview reading is inferred) | free | missing | – |
| Workspace showing the active view | free | partial | [`tableview.go`](../../internal/ui/fyneui/tableview.go); [PNG](../../internal/ui/fyneui/testdata/demo_pods.png) |
| Detail drawer on the right ([Detail drawer](#detail-drawer)) | free | partial: its width never follows its content and is the user's (owner requirement, built 2026-10-12): drag the left edge, 320 to 1000 points, kept in the preferences; long values wrap or are cut, the full text stays in the copy icon and the selection | [`drawer.go`](../../internal/ui/fyneui/drawer.go), [`panelwidths.go`](../../internal/ui/fyneui/panelwidths.go); `TestDrawerWidthNeverFollowsItsContent`, `TestDrawerWidthByDraggingItsEdge`, `TestPanelWidthsSurviveARestart`; render `TestRenderDrawerLongStatus` (PNG not committed) |
| Dock along the bottom ([Dock](#dock)) | free | partial | [`dock.go`](../../internal/ui/fyneui/dock.go) |
| Status bar | free | partial | [`bars.go`](../../internal/ui/fyneui/bars.go): context, API requests, informers, snapshots, demo events/s; the link to the owner ([D-23](../developer/decisions.md#d-23-a-link-to-the-owner-in-the-status-bar)) at the right, left of the MCP item, which switches the external MCP server (owner, 2026-10-12); its text at 13 points in a 26-point bar, as VS Code's reads (owner, 2026-10-12: the caption size was too small); `TestStatusBarOwnerLink`, `TestStatusBarOwnerLinkGivesWayFirst` |
| Welcome page (Help › Welcome) | free | missing | – |

## Clusters

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Kubeconfig files in `~/.kube/` (`%USERPROFILE%\.kube` on Windows) are found without setup; Lens lists `~/.kube` as an ordinary, removable synced folder in Preferences › Kubernetes (capture, 2026-10-12) | free | partial (2026-10-12): `~/.kube` is a synced folder from the first start (listed in the Kubeconfig Sources dialog as a sync folder with a "default" badge while it exists, removable, the removal remembered); its files, `config` included, are offered as `<context> [.kube/<file>]`, each read on its own and nothing merged, so removing the folder also removes `config`'s contexts, as in Lens (D-28, built 2026-10-12; the files `$KUBECONFIG` names, or `--kubeconfig`, are offered as sources of their own for the run, badge "this run", removable until the next start; a context name two files share gives two contexts); a credential plugin of any of them, `config`'s too, runs only after the user allowed it, asked once per file and identical plugin in one dialog that lists every context with its server and trust, `TestApprovalDialogListsEveryContextOfThePlugin` (Cogpit's own, H-1, D-28; Lens asks for none) | [`engine/kubeconfigs.go`](../../internal/engine/kubeconfigs.go) `WithDefaultFolder`, [`cmd/cogpit/main.go`](../../cmd/cogpit/main.go) `openKubeconfigSources`, [`cluster.go`](../../internal/cluster/cluster.go) `NamedKubeconfigFiles`; `TestDefaultKubeconfigIsOneFileOfTheSyncedFolder`, `TestSessionFilesAreSourcesOfTheirOwn`, `TestSameNamedContextsInTwoFilesAreBothReachable`, `TestConfigWithAPluginAsksForApprovalOnce`, `TestRunFlagConnectsABareContextName`, `TestDefaultFolderIsSyncedUntilRemoved`, `TestDefaultFolderOnlyWhileItExists`, `TestNoDefaultFolderWithoutTheOption`, `TestDefaultFolderPluginsNeedApproval`, `TestDefaultFolderIDDoesNotCollide`, `TestNamedKubeconfigFiles` |
| Kubeconfig Syncs: add files and folders to watch | free | partial (2026-10-12): one dialog, "Sync Files and Folders" (Cluster menu, the picker's "Add Kubeconfigs", the Kubeconfig Sources dialog), takes a file or a folder: select one and press Sync, or press Sync with nothing selected for the folder shown (`kubeconfig.addFile`, `kubeconfig.addFolder`); a folder's own files are picked up and dropped as they come and go (polled every 2 s), subfolders not (as Lens), dot files too (as Lens; since 2026-10-12): every regular file of at most 1 MiB that parses and defines a context, a FIFO or device never opened; each file is read on its own and offered as `<context> [<file>]`; when the connected context's file is deleted or its source removed, Cogpit closes the connection (the workspace says why, a warning toast, the picker marks the context); a connect that failed or was closed that way has its log behind a warning icon beside the context in the picker, "Connection Updates: <context>", INFO and ERROR lines; removing a synced item asks first, naming its path and saying that only Cogpit's link goes and the files stay on disk (owner requirement), and closes a connection made from it. Still differs from Lens (capture, 2026-10-12): Lens opens the system's file dialog, Cogpit its own browser, because Fyne's file dialogs take a file or a folder, not both, and it selects one item at a time; Lens runs a synced file's credential plugin on connect without asking (Cogpit asks: H-1, deliberate); the log holds what Cogpit knows ("Connecting to", the error, the reason a connection was closed), not the plugin's standard error, which client-go writes to Cogpit's own, and there is no ↻ beside the ⚠; a file edited so that it no longer defines the connected context leaves the connection open, and so does a file that is only rewritten or unreadable for a moment (Lens's behaviour not captured); the closed connection of a file that is the default kubeconfig is not handled; Lens's paste item is not a target, D-21. Cogpit's own dialogs are inferred | [`engine/kubeconfigs.go`](../../internal/engine/kubeconfigs.go), [`engine/kubeconfigs_connect.go`](../../internal/engine/kubeconfigs_connect.go), [`engine/connectlog.go`](../../internal/engine/connectlog.go), [`fyneui/kubeconfigs.go`](../../internal/ui/fyneui/kubeconfigs.go), [`fyneui/syncpicker.go`](../../internal/ui/fyneui/syncpicker.go), [`fyneui/connectlog.go`](../../internal/ui/fyneui/connectlog.go); `TestSyncFolderPicksUpAndDropsFiles`, `TestSyncFolderReadsDotFiles`, `TestSyncFolderNeverOpensAFIFO`, `TestConnectionClosesWhenItsFileLeavesASyncFolder`, `TestConnectionClosesWhenAnAddedFileIsDeleted`, `TestConnectionStaysWhileItsFileIsOnlyRewritten`, `TestConnectionClosesWhenItsSourceIsRemoved`, `TestConnectionToASourceClosesWhenItsFileIsDeleted`, `TestConnectLogOfAFailedConnect`, `TestPickerMarksAFailedConnectAndOpensItsLog`, `TestSyncPickerTakesFilesAndFoldersInOneDialog`, `TestSyncPickerNavigates`, `TestRemoveConfirmationNamesThePathAndSaysOnlyTheLinkGoes`, `TestAddedFilesAreNeverWritten`, `TestExecPluginRunsOnlyOnceAllowed`, `TestSyncFromTheMenuAndEventsReloadThePicker` |
| Add clusters by pasting a kubeconfig; Lens stores the text itself, not in `~/.kube/config`; its "Add Kubeconfigs" menu offers "Paste from clipboard" (capture, 2026-10-12) | free | not a target ([D-21](../developer/decisions.md#d-21-pasting-a-kubeconfig), owner 2026-10-12; the paste dialog built the same day is removed) | – |
| Navigator sections: Kubernetes Clusters (Local Kubeconfigs and the cloud sections), Teamwork, Extensions | free | missing | – |
| Search clusters by name (⌘P) | free | partial: filter in the context picker, no ⌘P | the context picker's filter field, [`picker.go`](../../internal/ui/fyneui/picker.go) (the picker closes on a click outside it and on Escape, owner 2026-10-12: `TestPickerClosesOnAClickOutside`, [`overlay.go`](../../internal/ui/fyneui/overlay.go)); no ⌘P |
| Cluster menu: connect, disconnect, settings, add to hotbar, remove (labels: capture needed) | free | missing | – |
| Cluster identity: upload an icon (.jpeg, .png, .svg) or pick a colour | free | missing | – |
| Drill into a section or cluster with its chevron to filter the tabs to that scope | free | missing | – |
| Navigator keyboard: ↑↓ move, → opens, ← closes, Tab and ⇧Tab, ↵ preview tab, ⇧↵ fixed tab | free | missing | – |
| Cloud sections list clusters from AWS, Azure and Google accounts | paid | missing | – |

## Hotbar

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Hotbar of up to 12 clusters per bank | paid | missing | – |
| Banks: create, rename, remove, switch from the command palette | paid | missing | – |
| F1–F12 switch to a cluster in the bank, ⇧F1–⇧F12 switch bank | paid | missing | – |
| Add a cluster from its menu ("Add to hotbar") | paid | missing | – |

## Cluster sidebar

The tree as the running Lens Desktop shows it: captures by the owner on 2026-10-11 of two clusters
with Flux CD installed, one on Kubernetes 1.35.x and one on 1.36.2, every group expanded (Lens
version not recorded with the captures). Groups marked *NEW* carry that badge in Lens. Items in
brackets were in neither capture; their place follows Lens's documentation.

```text
Overview
Applications
Flux (NEW, paid)    Dashboard, Helm Releases, Kustomizations,
                    Image Automation › Image Repositories, Image Policies, Image Update Automations,
                    Notifications › Alerts, Providers, Receivers,
                    Sources › Git Repositories, OCI Repositories, Helm Repositories, Buckets,
                              External Artifacts
Nodes
Workloads           Overview, Pods, Deployments, Daemon Sets, Stateful Sets, Replica Sets,
                    Replication Controllers, Jobs, Cron Jobs
Config              Config Maps, Secrets, Resource Quotas, Limit Ranges, Horizontal Pod
                    Autoscalers, [Vertical Pod Autoscalers], Pod Disruption Budgets, Priority
                    Classes, Runtime Classes, Leases, Mutating Webhook Configurations, Validating
                    Webhook Configurations,
                    Admission Policies (NEW) › Mutating Admission Policies, Mutating Admission
                              Policy Bindings, Validating Admission Policies, Validating
                              Admission Policy Bindings
Network             Services, Endpoint Slices, Endpoints, Ingresses, Ingress Classes, Network
                    Policies,
                    Gateway API (NEW) › Gateway Classes, Gateways, HTTP Routes, Backend TLS
                              Policies, gRPC Routes, Listener Sets, Reference Grants, TCP Routes,
                              TLS Routes, UDP Routes
                    Port Forwarding
Storage             Persistent Volume Claims, Persistent Volumes, Storage Classes
Device Resources    (NEW) Resource Claims, Resource Claim Templates, Device Classes, Resource Slices
Namespaces
Events
Helm                Charts, Releases
Access Control      Service Accounts, Cluster Roles, Roles, Cluster Role Bindings, Role Bindings,
                    [Pod Security Policies]
Custom Resources    Definitions, then one collapsed group per API group, sorted by name; inside a
                    group its kinds by Kind name, singular and sorted (acme.cert-manager.io ›
                    Challenge, Order)
[Paid sections]     [Argo CD, Cost Monitoring, Security Center]
```

**What decides which items appear** is what the cluster serves, two ways:

- **The Kubernetes version** for built-in APIs. Mutating Admission Policies and their bindings were
  in the 1.36.2 capture and not in the 1.35.x one: `admissionregistration.k8s.io/v1`
  `MutatingAdmissionPolicy` exists from 1.36 (the `prerelease-lifecycle-gen:introduced=1.36` tag in
  k8s.io/api v0.37.1); before, it is `v1beta1` only, and beta APIs are off by default.
- **Installed CRDs** for everything else. Gateway API was in the 1.36.2 capture only; it is a set of
  CRDs (`gateway.networking.k8s.io`, and `gateway.networking.x-k8s.io` for Listener Sets), so its
  absence on the other cluster is a matter of what is installed there, not of the version (that
  capture's custom resource list is cut off, so it is not verified). Flux CD and Vertical Pod
  Autoscalers work the same way.

A curated section does not take its groups out of Custom Resources: the Flux toolkit groups
(`helm.toolkit.fluxcd.io`, `kustomize.toolkit.fluxcd.io`, …) and `gateway.networking.k8s.io` are
listed there too.

Where the documentation said otherwise, the captures win: Flux sits between Applications and
Nodes; Endpoint Slices come before Endpoints; Gateway API is a sub-group of Network after Network
Policies; the webhook views read "Configurations"; the admission policies are a sub-group of Config,
mutating first; Custom Resources starts with Definitions.

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| The tree above | free | partial: the kinds Cogpit has profiles for, in the captured order; Custom Resources as captured (Definitions, then one closed group per API group, sorted, its kinds by singular Kind name, sorted; the groups of curated sections listed too, selecting either entry opens the same view); Network › Gateway API and Config › Admission Policies as sub-groups, Config › Vertical Pod Autoscalers (a CRD kind); Workloads › Overview first in its group (a page, 2026-10-11, `TestWorkloadsOverviewPage`); Port Forwarding as the Network group's last entry, after Gateway API, as captured (a local kind, so a kind view: 2026-10-11, `TestPortForwardingView`); Helm › Releases between Events and Access Control (2026-10-11; Helm › Charts missing); the other pages that are not kinds (the cluster Overview, Applications, Flux) are missing | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go) `navLayout`; `TestSidebarCustomResources`, `TestReleasesViewAndDrawer` (the Helm group); PNGs [dark](../../internal/ui/fyneui/testdata/sidebar_custom_dark.png), [light](../../internal/ui/fyneui/testdata/sidebar_custom_light.png) |
| A group or item appears only when the cluster serves its kinds (Admission Policies, Gateway API, Device Resources) | free | partial: built-in kinds by discovery at connect, a kind's older API versions included where its profile lists them (2026-10-11: Validating Admission Policies and their bindings at v1, else v1beta1, else v1alpha1; Mutating Admission Policies and their bindings at v1 only, so they appear from Kubernetes 1.36 as captured: `TestBuiltinAdmissionPoliciesFollowTheServedVersion`); custom resource kinds (Gateway API, Vertical Pod Autoscalers, every API group of Custom Resources) follow the cluster's definitions while connected, where Lens needs a reconnect; kinds the user may not list are left out once the access reviews know it, kinds not known yet are shown (2026-10-11), and the View menu offers the same kinds (2026-10-12) | [`connector.go`](../../internal/engine/connector.go), [`customkinds.go`](../../internal/engine/customkinds.go), [`accessreviews.go`](../../internal/engine/accessreviews.go), [`sidebar.go`](../../internal/ui/fyneui/sidebar.go); `TestSidebarFollowsKindsEvent`, `TestCustomKindLifecycle`, `TestBuiltinAdmissionPoliciesFollowTheServedVersion`, `TestSidebarHidesKindsNotListable`, `TestViewMenuHidesKindsNotListable`, `TestKindAccessReviews`; [PNG](../../internal/ui/fyneui/testdata/access_dark.png) |
| Groups expand and collapse | free | partial: every group starts closed (owner, 2026-10-12; Lens opens its groups, captured for Custom Resources' API groups only) except the ones holding the open view (`TestSidebarCustomResources`); open and closed states survive a change of the cluster's kinds, not a reconnect; the group of a view opened elsewhere (an agent, a link) opens to show it (inferred) | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go); `TestSidebarFollowsKindsEvent` |

## Tables

Every list view is one table ([resource views](resource-parity.md)).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Sort by a column header, ascending and descending arrows, empty values last | free | partial | `TestHeaderTapDispatchesSort`; engine `SortRows` with empty text and age cells last |
| Resize columns by dragging | free | partial | `TestColumnResizeByDraggingHeaderSeparator`; widths kept in memory only |
| Double-click a column separator to fit its content | free | partial: whether Lens fits to every row or to the rows on screen is not documented (Cogpit: every row, inferred); that a double click on the separator reaches the table in a real window is not verified offscreen (Fyne's test driver calls `DoubleTapped` directly) | 2026-10-11: a double click on the separator right of a column in the header row (where the resize cursor shows) sizes the column to its title (with the sort arrow) and the widest text of all rows, kept as the user's width like a dragged one; between body cells it does nothing (`TestDoubleClickSeparatorFitsColumn`; [`tableheader.go`](../../internal/ui/fyneui/tableheader.go) `fitToContent`); fitting 10,000 distinct names costs 6.4 ms on the main goroutine (estimated from per-rune advances, the widest 16 measured exactly; measuring every name was 154 ms) |
| Reorder columns by dragging | free | missing | – |
| Show and hide columns (⋮ at the end of the header row) | free | partial: the ⋮ as captured; what it opens is inferred (Cogpit: one check per column, toggling without closing, as the namespace filter does); the choice lasts for the session, not across restarts (whether Lens keeps it: capture needed) | 2026-10-11: hidden columns are UI state per view (`ui.setColumns`, `UIState.HiddenColumns`, in `ui_state`), the snapshot's columns carry them, a kind's default-hidden columns (a custom resource's printer columns of a higher priority, the Events and Definitions name) start hidden; the last visible column cannot be hidden (`TestSetColumns`, `TestColumnsMenuShowsAndHides`, `TestColumnsMenuKeepsOneColumn`; [`tableheader.go`](../../internal/ui/fyneui/tableheader.go), [`commands/tables.go`](../../internal/engine/commands/tables.go)); [PNG](../../internal/ui/fyneui/testdata/columns_menu_dark.png), [light](../../internal/ui/fyneui/testdata/columns_menu_light.png) |
| Search field per view, with toggles for match case ("Aa") and regular expressions (".*") (capture, 2026-10-11) | free | partial: no toggles | `TestSearchIsDebounced`; terms matched in any cell |
| A cell too narrow for its text shows the full text in a tooltip on hover (capture of an event message, 2026-10-11) | free | done | `TestTooltips`: after a hover delay (450 ms, Cogpit's choice) a cut-off cell shows its full text near the pointer; leaving the cell or the table, scrolling and changing the view close it ([`tooltip.go`](../../internal/ui/fyneui/tooltip.go)); Fyne 2.8 has no tooltip widget, Cogpit draws its own; [PNG](../../internal/ui/fyneui/testdata/tooltip_dark.png) |
| Namespace filter: multi-select, default "All namespaces", Space toggles without closing; none on cluster-scoped views (captures of Nodes, Ingress Classes, Cluster Roles and Cluster Role Bindings, 2026-10-11) | free | partial | multi-select popup, default all ([`tableview.go`](../../internal/ui/fyneui/tableview.go)); hidden on cluster-scoped views since 2026-10-11 (`TestNamespaceFilterHiddenOnClusterScopedViews`, [PNG](../../internal/ui/fyneui/testdata/nodes_list_dark.png)); Space not verified |
| Actions menu per row (⋮ at the row's end), starting with Show Details and ending with Edit and Delete, the kind's own actions between; items with a › open a submenu (captures of a custom resource, a pod and a cron job, 2026-10-11) | free | partial: the items are only as complete as the registry's commands (Edit and Delete on every kind, node and workload actions come with their commands; node shell is not built); acts on its row, not on a multi-selection; a disabled item says why after its label | `TestRowMenuNodes`, `TestRowMenuCronJobs`, `TestRowMenuPodsContainerSubmenu`, `TestMenuColumnOpensMenuWithoutSelecting`: Show Details, then `Registry.Actions` of the kind by `Order`, hidden when not applicable (Uncordon replaces Cordon), greyed when disabled; a command with a `container` parameter opens a submenu of the containers and init containers, one that needs more parameters a schema-generated dialog (`TestScaleDialogPrefilled`) ([`rowmenu.go`](../../internal/ui/fyneui/rowmenu.go), [`paramdialog.go`](../../internal/ui/fyneui/paramdialog.go)); [PNG](../../internal/ui/fyneui/testdata/row_menu_dark.png), [dialog](../../internal/ui/fyneui/testdata/scale_dialog_dark.png) |
| Right-click on a row opens the actions menu (since 2025.10.230725) | free | done | `TestRowMenuNodes` opens the menu with a secondary tap on a row; the menu is the ⋮'s |
| Keyboard: ↑↓ rows, Tab columns, ↵ or Space opens the details | free | done | 2026-10-11: with the table focused (a click on a row, or Tab into it), ↑↓ move the row band, Tab and ⇧Tab move between columns, ↵ and Space open the row's details through `ui.select` as a click does, Escape closes them; a ring in the accent colour marks the keyboard's cell (Cogpit's drawing: Fyne's own cell marker is hidden with the separators); a newly opened view starts at its first cell (`TestTableKeyboard`; [`tablekeys.go`](../../internal/ui/fyneui/tablekeys.go)); [PNG](../../internal/ui/fyneui/testdata/keyboard_dark.png), [light](../../internal/ui/fyneui/testdata/keyboard_light.png) |
| Export the table as CSV (a download button beside the item count, "18 items"; the format is inferred) | free | partial: the button and the count as captured; the format is inferred (a header row with the visible columns' titles, then the rows as the table shows them: filtered, sorted, ages relative), as are the file name ("<view>-<context>.csv") and "18 of 10,000 items" while a filter hides rows (Cogpit's) | 2026-10-11: `table.exportCsv` (Read) writes the open view's snapshot in the engine, so agents export the same text; the GUI saves it with a file save dialog ([`tableheader.go`](../../internal/ui/fyneui/tableheader.go), [`engine/tables.go`](../../internal/engine/tables.go) `WriteCSV`); cells a spreadsheet would run as a formula (=, +, -, @ first) get a leading apostrophe (Cogpit's); a metadata-only kind (Secrets) exports no values (`TestWriteCSV`, `TestCSVOfMetadataOnlyKind`, `TestExportCSV`, `TestCSVButtonSavesTheView`, `TestItemCount`; on kind 1.36.1 `TestKindWorkloadsOverviewAndCSV`); [PNG](../../internal/ui/fyneui/testdata/pods_list_dark.png) |
| Select rows with a checkbox column, all rows with the header's checkbox (capture, 2026-10-11) | free | missing | – |
| A warning column right after Name in built-in views (Pods, Pod Disruption Budgets, Cron Jobs): an orange triangle on rows with a problem (seen on a pod with many restarts and a container not ready; the rule is inferred) | free | partial: the rules are inferred; Cogpit's icon is Fyne's warning triangle in the warning colour; Pod Disruption Budgets and Cron Jobs come with their views | Pods and Nodes (`TestPodWarningTexts`, `TestNodeWarnings`): the icon centred in a narrow column without a header title, its reason as the tooltip (`TestTooltips`); inferred rules in [resource parity](resource-parity.md#cluster) ([`pods.go`](../../internal/model/kinds/pods.go) `podWarning`, [`nodes.go`](../../internal/model/kinds/nodes.go)); [PNG](../../internal/ui/fyneui/testdata/pods_list_dark.png). Cluster Roles, Roles and Cluster Role Bindings (2026-10-11): a role or binding that grants nothing, inferred ([resource parity](resource-parity.md#access-control); `TestClusterRoleTexts`, `TestRoleTexts`, `TestClusterRoleBindingTexts`; [PNG](../../internal/ui/fyneui/testdata/clusterrolebindings_list_dark.png)) |
| A floating "+" button at the bottom right of every list opens a Create resource tab in the dock (capture, 2026-10-11) | free | partial: what Lens fills the tab with is not captured; Cogpit starts from the template of the view's kind, else the example Pod (Cogpit's choice); none on the views of local and virtual kinds, Port Forwarding and Helm Releases, whose objects no manifest creates (owner, 2026-10-12; `kinds.list` marks them `local` and `virtual`) | [`createbutton.go`](../../internal/ui/fyneui/createbutton.go), `TestCreateTabTemplatesAndCreate`, `TestCreateButtonHiddenWithoutCluster`, `TestCreateButtonHiddenOnLocalAndVirtualKinds`; [PNG](../../internal/ui/fyneui/testdata/create_dark.png) |
| The sorted column shows an arrow beside its title | free | partial | `TestHeaderTapDispatchesSort` |
| Cells that name another object are links: Namespace, Controlled By, Node, Involved Object, a definition's resource (captures, 2026-10-11); a Namespace link narrows the open view's namespace filter to that namespace (owner, 2026-10-11); where the others lead: capture needed | free | partial: Controlled By and Involved Object link built-in kinds only (a custom kind's key needs its plural, which a reference does not carry); a definition's resource comes with the Definitions view; Cogpit's links open the object (inferred); further links of Cogpit's (2026-10-11, third run, inferred): a claim's and a volume's Storage Class, a volume's Claim, and a claim's pod when it has one (a cell holds one link; several pods stay text) (`TestClaimPodsCellLinks`, `TestPersistentVolumeLinks`) | `TestLinkClicks`: in the link colour, underlined and a hand pointer while hovered; a click on the text follows the link without selecting the row, beside it selects the row; the Namespace link sets `ui.setNamespace` to that namespace, others `ui.navigate`, a link without a name `ui.openView`; a link to a kind the cluster does not serve, or that Cogpit has no view for yet (Secrets), shows as plain text, and one that fails anyway shows an error toast ([`links.go`](../../internal/ui/fyneui/links.go)); Service Accounts have a view since 2026-10-11, so links to them lead there |
| Row line height (preference) | free | missing | – |
| Status colours per state (palette: capture needed; seen: "Running" green, "Evicted" red) | free | partial | status tokens mapped by `StatusColor` ([`theme.go`](../../internal/ui/fyneui/theme.go)); Solarized's accents ([D-22](../developer/decisions.md#d-22-colour-theme), 2026-10-12), contrast held by `TestPaletteContrast`; Lens's own palette is not captured |

## Detail drawer

[Details panel](https://docs.lenshq.io/k8slens/using-lens/details-panel/).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Opens from the right when a row is selected, beside the narrowed table; titled "<Kind>: <name>" (capture, 2026-10-11) | free | partial: the drawer's width is the user's and stays when its content changes (2026-10-12, owner requirement; see Layout) | `TestRowTapDispatchesSelect`; `ui.select` opens the drawer |
| Actions at the top: Copy, Edit, Delete and the kind's own actions; captured as icons in the drawer's title bar: for a pod seven icons ending with Edit and Delete, for a node Shell, Cordon (a pause icon; Uncordon, a play icon, on a cordoned node), Drain, Edit, Delete (which pod icon is which action: inferred, tooltips needed) | free | partial: Copy (`object.copyName`, 2026-10-11) is the first icon of every drawer and puts the object's name on the clipboard ("Copies the resource name", documented; the toast saying so is Cogpit's); the node capture shows no Copy among its five icons, so where Lens puts it is inferred; the icons are Fyne's, chosen per command ([`cmdicons.go`](../../internal/ui/fyneui/cmdicons.go)), the pause and play for Cordon and Uncordon as captured | `TestDrawerToolbarFromRegistry`, `TestDrawerActionsDisabledWithReason`: the row menu's actions as icons in the title bar, a tooltip names each and says why a disabled one cannot run; [PNG](../../internal/ui/fyneui/testdata/node_drawer_dark.png); Copy: `TestCopyPutsTheNameOnTheClipboard`, `TestCopyIsFirstInEveryDrawer`, `TestDrawerOnlyActionsStayOutOfRowMenus`, [PNG](../../internal/ui/fyneui/testdata/drawer_copy_dark.png) |
| Drawer sections in order: Metrics, Properties, the kind's own sections, Events ("No events found" when none) (capture of a pod and a node) | free | partial: the Metrics section is first for a pod and a node since 2026-10-12 (see Cluster overview and charts); the YAML stays a second tab (Cogpit's own) | `TestDrawerSectionsAsLens`, `TestDrawerNoEventsFound`, `TestBuildDetailPodAsLens`: Properties first (metadata and the kind's state merged, no UID), the kind's sections with group headings (Init Containers, Containers), Events last ([`drawer.go`](../../internal/ui/fyneui/drawer.go), [`engine/drawer.go`](../../internal/engine/drawer.go)); [PNG](../../internal/ui/fyneui/testdata/pod_drawer_dark.png) |
| Long values fold: Labels, Annotations, Tolerations, environment variables, mounts and volumes show a count ("6 Labels") and expand on click; lists of short values show as chips (pod IPs, conditions, images) (capture) | free | partial: images are not chips; an unfolded list shows chips (labels, volumes) or one line per entry (annotations, tolerations, environment, mounts), which is inferred | `TestDrawerSectionsAsLens` folds and unfolds the labels; pod IPs, conditions and secrets are chips (`TestPodDetailSections`) |
| Object references in the drawer are links (namespace, controller, node, service account, secrets); container ports have a "Forward…" button (capture) | free | done (2026-10-11): Forward… beside each TCP container port, right after the port's link ([Port forwarding](#port-forwarding)) | `TestDrawerSectionsAsLens` (the node link navigates), `TestPropertiesSection`, `TestPodDetailSections`: namespace, controller (built-in kinds), node, service account, secrets and a volume's ConfigMap, Secret or claim link to their objects; the service account link leads to the Service Accounts view (2026-10-11, `TestPodServiceAccountLinkLive`); a binding's role and its ServiceAccount subjects, in a table, link to them (`TestClusterRoleBindingDrawerLinks`; table cells link, `TestDrawerTableLinks`); a claim's class, volume and pods, a volume's class and claim, an endpoint slice's Service, and the cells of drawer tables (a node's pods and their namespaces, a Service's endpoint slices, an endpoint's node and target) link to theirs, a table's warning column shows the icon with its reason as the tooltip (`TestPersistentVolumeClaimPods`, `TestDrawerCellTable`, `TestNodeDrawerPodsTable`, `TestEndpointSliceDrawer`; [PNG](../../internal/ui/fyneui/testdata/node_pods_drawer_dark.png)); Forward… beside each TCP container port (`TestForwardDialogFromPodDrawer`, `TestDrawerItemLinkAndRowMenuDialog`, [PNG](../../internal/ui/fyneui/testdata/pod_drawer_ports_dark.png)) |

## Dock

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Bottom dock with tabs: Terminal session, Create resource, Edit, logs, Helm Install, Helm Upgrade; tabs named after their object ("Pod <name>"), with a close button; on the right, maximize and collapse (capture, 2026-10-11) | free | partial: terminal, shell, logs, Edit ("ConfigMap demo"), Create resource and Helm Upgrade ("Helm Upgrade: <name>", 2026-10-12, as captured that day: save and search icons, "/<chart>" with "version:" and a picker of the release's own version, "(no repository)", the info line and the values in a YAML editor; a status line is Cogpit's); no Helm Install tab, no maximize | `TestDockOpensAndClosesStreams`, `TestEditTabSaveReviewsThenApplies`, `TestCreateTabTemplatesAndCreate`, `TestHelmUpgradeTabSaves`, PNGs [dark](../../internal/ui/fyneui/testdata/helm_upgrade_tab_dark.png), [light](../../internal/ui/fyneui/testdata/helm_upgrade_tab_light.png) ([resource parity](resource-parity.md#helm)) |
| "New tab" menu: Terminal session, Create resource | free | done | [`dock.go`](../../internal/ui/fyneui/dock.go) `addMenu`, `TestDockAddMenuOffersCreateResource`; an item the engine disables names its reason |
| Tab icons take the cluster's colour | free | missing | – |

## Editor and create resource

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Edit tab: YAML editor with syntax highlighting and Save | free | partial: Cogpit's Save first shows the changes against the object as the cache holds it and the server's dry run, and writes only on Apply (Cogpit's safety step; that Lens's Save writes at once is inferred); "Save & Close" is inferred; no unsaved-text drafts | `object.edit`, `editor.diff`, `object.dryRun`, `object.apply` ([`commands/edit.go`](../../internal/engine/commands/edit.go), [`engine/editor.go`](../../internal/engine/editor.go)); `TestEditDiffDryRunApply`, `TestApplyConflictNeverOverwrites`, `TestEditTabSaveReviewsThenApplies`, `TestEditTabConflictAsksAgain`, `TestKindEditCreateConflictDelete`; [PNG](../../internal/ui/fyneui/testdata/edit_diff_dark.png) |
| Diff view: "Show changes", "Revert to original", "Exit diff view" | paid | partial: every Save shows the changes (line diff, three lines of context) and "Back to Editor" leaves them; no "Revert to original" | [`diffview.go`](../../internal/ui/fyneui/diffview.go), `TestEditTabSaveReviewsThenApplies`; [PNG](../../internal/ui/fyneui/testdata/edit_diff_dark.png) |
| Create resource tab: template picker, "Create & Close"; captured: a dock tab with save and search buttons, a "Select Template …" picker whose templates sit under a "LENS" heading, one per kind in alphabetical order (ClusterRole, ClusterRoleBinding, ConfigMap, CronJob, CustomResourceDefinition, DaemonSet, Deployment, Endpoint, EndpointSlice, HorizontalPodAutoscaler, Ingress, IngressClass, Job, …), and an example Pod manifest to start from; Cogpit writes its own templates (clean-room) | free | partial: no save and no search button (what they do is not captured); Cogpit's own 30 templates under a "Cogpit" heading, without Secret (H-8) and the deprecated Endpoints; Create as well as "Create & Close" | [`createtab.go`](../../internal/ui/fyneui/createtab.go), [`engine/templates/`](../../internal/engine/templates/), `object.create`, `resource.createTab`, `resource.templates`; `TestCreateObjects`, `TestCreateTabAndTemplates`, `TestCreateTabTemplatesAndCreate`; [PNG](../../internal/ui/fyneui/testdata/create_dark.png) |
| Bundled templates, device-resource templates included (2026.9) | free | partial: Cogpit's own templates include DeviceClass, ResourceClaim and ResourceClaimTemplate (`resource.k8s.io/v1`); which device templates Lens bundles is not captured | every template passes a server-side dry run on Kubernetes 1.36.1 (`TestKindEditCreateConflictDelete`) |
| Dialogs instead of YAML to create Namespaces, Secrets, Resource Quotas, Service Accounts and RBAC objects | free | missing | – |

## Logs

[Log options](https://docs.lenshq.io/k8slens/using-lens/log-options/) and
[view logs](https://docs.lenshq.io/k8slens/cluster/view-logs/).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Pod and container selectors | free | missing (2026-10-11: a controlling resource's tab shows every pod and container, each line named, with no selector to narrow it; a pod's tab shows the container picked in the row menu's submenu) | – |
| Follow; jump to the latest line | free | partial | [`termview/logs.go`](../../internal/ui/fyneui/termview/logs.go), follow toggle tested |
| Show timestamps | free | partial (2026-10-11): an option of the tab in the engine (`logs.setOptions`, `ui_state` shows it), sent as `timestamps=true`, so the stream opens again; the API server's RFC 3339 timestamp shows muted before the message (its format and colour are inferred: the documentation says only "Display timestamps") | `TestPodLogOptions`, `TestWorkloadLogsPreviousAndTimestamps`, `TestLogTabOptionsAreEngineState`, `TestRunsColourNamesTimestampsAndMatches`, `TestKindWorkloadLogsAttachRestartScale`, [PNG](../../internal/ui/fyneui/testdata/logs_pod_dark.png) |
| Word wrap | free | partial (2026-10-11): an option of the tab in the engine; long lines wrap at the tab's width, after a space where the row has one (inferred), again when the width changes | `TestLogsViewOptions`, `TestLogsWrapFollowsTheWidth`, [PNG](../../internal/ui/fyneui/testdata/logs_wrap_filter_dark.png), [light](../../internal/ui/fyneui/testdata/logs_wrap_filter_light.png) |
| Show the previous terminated container | free | partial (2026-10-11): an option of the tab in the engine, sent as `previous=true` without following; in a controlling resource's tab every container with a previous instance, after which the stream ends | `TestPodLogOptions`, `TestWorkloadLogsPreviousAndTimestamps`, `TestLogTabOptionsAreEngineState`, `TestKindWorkloadLogsAttachRestartScale` (a crashing pod's previous container on kind) |
| Logs of a controlling resource (all its pods), with resource names shown | free | partial (2026-10-11): `workload.logs` from the row menus of Deployments, Daemon Sets, Stateful Sets, Replica Sets and Jobs, and, in a pod's tab, a button that moves the tab to the logs of the pod's controller (`logs.controller`; the documentation's "move to the logs of a controlling resource"; that a Deployment's pod moves to the Deployment, not its replica set, is inferred); every line starts with `<pod>/<container>`, in the accent colour, hidden with "Resource names" off (the documentation's "Show resource names"; the format is inferred); pods and restarted containers join while the tab follows | `TestWorkloadLogsMergeEveryContainer`, `TestWorkloadLogsFollowNewPodsAndRestarts`, `TestLogsControllerMovesTheTab`, `TestKindWorkloadLogsAttachRestartScale`, `TestLogTabMovesToTheController`, [PNG](../../internal/ui/fyneui/testdata/logs_workload_dark.png), [light](../../internal/ui/fyneui/testdata/logs_workload_light.png) |
| Filter matching lines on or off (`/`) | free | partial (2026-10-11): the tab's search (an option in the engine) highlights matches, case-insensitively; "Filter" on shows only the matching lines and the count ("2 of 9 lines"), as the documentation describes; `/` focuses the search when nothing else, or the log list, has the keyboard focus (whether Lens's "/" is a key or the control's label is not captured) | `TestLogsViewOptions`, `TestLogsFilterAppliesToNewLines`, `TestLogsSlashFocusesSearch`, `TestLogTabSearchIsDebouncedAndSlashFocusesIt`, [PNG](../../internal/ui/fyneui/testdata/logs_wrap_filter_dark.png) |
| Download logs | free | partial (2026-10-11): a save dialog proposes a file named after the tab; the GUI writes every line the tab holds as it shows them (resource names and timestamps as toggled, not only the filtered lines; inferred) | `TestLogDownloadWritesTheShownLines`, `TestLogsViewOptions` |
| Jump to a time and date | free | missing | – |
| Load older lines when scrolling up | free | missing | – |

## Terminal and shells

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Terminal session for a cluster, with the working directory and default namespace from cluster settings | free | partial: context and namespace, no cluster settings | `TestStartLocalBindsKubectlToContextAndNamespace`, `TestKindEngineWorkflows`; namespace from the filter, no cluster settings |
| Pod shell, attach and node shell open as dock tabs | free | partial: pod shell and attach (2026-10-11: stdin and a TTY only as the container has them; the attachment icon on the tab is Cogpit's); no node shell | `TestKindPodExec`, `TestKindEngineWorkflows`; attach: `TestPodAttach`, `TestAttachWithoutTTY`, `TestAttachTabShowsATerminal`, `TestKindWorkloadLogsAttachRestartScale`, [PNG](../../internal/ui/fyneui/testdata/attach_dark.png) |
| Copy and paste with the right mouse button (preference) | free | missing | – |
| Copy and paste keys: ⌘C / ⌘V; Windows and Linux ⌃C, ⌃⇧C or ⌃Ins, and ⌃V, ⌃⇧V or ⇧Ins | free | partial: macOS keys | fyne-io/terminal: ⌘C/⌘V on macOS (offscreen, synthetic events: `TestTerminalCopyAndPaste` in [`termview/terminal_test.go`](../../internal/ui/fyneui/termview/terminal_test.go)); Ctrl+Shift+C/V on Linux and Windows |
| Theme, font size and font family (preferences) | free | missing | – |
| Shell path and shell arguments (2026.10: one argument per line) | free | missing | – |

## Port forwarding

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| A container port link ("<name>: <port>/<protocol>" in the drawer, captured) forwards to a random local port and opens the browser | free | done (2026-10-11): each TCP port of a container, init containers too, is its own drawer item whose value is a link (an item action marked `link`) that runs `pod.portForward` without a local port and with `openInBrowser`; the GUI opens the forward's URL with Fyne's `OpenURL` (offscreen tests stop at that call; no real window checked). UDP and SCTP ports have no link (Cogpit's: the port-forward protocol carries TCP only) | `TestForwardPortLink`, `TestContainerPortsForward`, `TestDrawerPortActionsDispatch`; [`kinds/portforwards.go`](../../internal/model/kinds/portforwards.go) `containerPortItems`; PNG [dark](../../internal/ui/fyneui/testdata/pod_drawer_ports_dark.png), [light](../../internal/ui/fyneui/testdata/pod_drawer_ports_light.png) |
| Forward dialog, from the "Forward…" button beside a container port (init containers too): titled "Port Forwarding for <pod>", a local port field ("Random" when empty), an "https" checkbox (off), an "Open in Browser" checkbox (on), Cancel and Start; the window behind is dimmed (capture, 2026-10-11) | free | done (2026-10-11): as captured, with Fyne's dimmed dialog; the field's label "Local port" is Cogpit's (the capture names a local port field); Start dispatches `pod.portForward` and opens the browser when asked; a service's ports have the same button (`service.portForward`, inferred: Lens's service drawer is not captured). a local port outside 1 to 65535 disables Start once the field loses focus (Cogpit's) | `TestForwardDialogFromPodDrawer`, `TestParseLocalPort`; [`fyneui/portforward.go`](../../internal/ui/fyneui/portforward.go); PNG [dark](../../internal/ui/fyneui/testdata/forward_dialog_dark.png), [light](../../internal/ui/fyneui/testdata/forward_dialog_light.png) |
| Port Forwarding view to open, edit, stop and delete forwards ([resource parity](resource-parity.md#network)) | free | partial (2026-10-11): the view of the local kind `portforwards` with the captured columns and row menu (Open, Edit, Stop, Delete), the same actions as icons in the drawer; Open opens the forward's URL, `http(s)://127.0.0.1:<port>` (the GUI opens no other host; not `localhost` as Lens shows it, which resolves to `::1` first, where another process may listen: review 2026-10-11, `TestForwardURLReachesTheForward`), Edit opens the forward dialog started from the forward's settings and starts it again (inferred: Lens's Edit dialog is not captured), Stop keeps the forward stopped in the list, Delete removes it; the status texts (Active, Starting, Waiting, Reconnecting, Stopped, Failed) and their colours are inferred; the view has no "+" button (owner, 2026-10-12; Lens's view not captured) | `TestPortForwardingView`, `TestPortForwardLifecycle`, `TestPortForwardsForAgentsAndMenus`, `TestKindPortForward`; PNGs [view dark](../../internal/ui/fyneui/testdata/portforwards_view_dark.png), [view light](../../internal/ui/fyneui/testdata/portforwards_view_light.png), [drawer dark](../../internal/ui/fyneui/testdata/portforward_drawer_dark.png), [drawer light](../../internal/ui/fyneui/testdata/portforward_drawer_light.png) |

## Command palette

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| ⌘⇧P opens it | free | missing | – |
| "Switch to *resource*" from a cluster view | free | missing | – |
| Hotbar bank commands (`hotbar: create bank`, rename, remove) | paid | missing | – |

## Application menu

Lens Desktop documents its menus only in part.

| Menu and items | Plan | Status | Evidence |
|---|---|---|---|
| Application menu with Preferences (items: capture needed) | free | missing | – |
| Window: Minimize; on macOS Zoom, Bring All to Front and the list of windows; on Windows and Linux Close Window (2026.10.51501) | free | missing | – |
| Help › Welcome, with "Open Onboarding Wizard..." | free | missing | – |

## Keyboard shortcuts

Lens Desktop's [shortcuts page](https://docs.lenshq.io/k8slens/using-lens/shortcuts/) (dated
2026-07-09). Its terminal page gives ⌘⇧, for a terminal session, which contradicts this table;
the table is taken as correct until a capture decides.

| Action | Windows / Linux | macOS | Plan | Status | Evidence |
|---|---|---|---|---|---|
| Command palette | Ctrl+Shift+P | ⌘⇧P | free | missing | – |
| Search a cluster by name | Ctrl+P | ⌘P | free | missing | – |
| Search in the tab | Ctrl+F | ⌘F | free | missing | – |
| Preferences | Ctrl+, | ⌘, | free | missing | – |
| Cluster settings | Ctrl+Shift+, | ⌘⇧, | free | missing | – |
| Ask AI (cluster tab) | Ctrl+I | ⌘I | paid | missing (opens Cogpit's agent chat, [D-10](../developer/decisions.md#d-10-built-in-agent)) | – |
| Terminal session (cluster tab) | Ctrl+Shift+T | ⌘⇧T | free | missing | – |
| Close the tab | Ctrl+W | ⌘W | free | missing | – |
| Reload | Ctrl+R | ⌘R | free | missing | – |
| Back / forward in the tab's history | Alt+← / Alt+→ | ⌃- / ⌃⇧- | free | missing | – |
| Next tab | Ctrl+PageUp (as documented) | ⌘⌥→ | free | missing | – |
| Previous tab | Ctrl+PageDown (as documented) | ⌘⌥← | free | missing | – |
| Tab *n* | Ctrl+*n* | ⌘*n* | free | missing | – |
| Cluster in the hotbar bank / switch bank | F1–F12 / Shift+F1–F12 | same | paid | missing | – |
| New space (Teamwork) | Ctrl+N | ⌘N | free | not a target ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | – |
| Quit | Alt+F4 | ⌘Q | free | missing | – |
| Table: rows / columns / details | ↑↓ / Tab / ↵ or Space | same | free | done (2026-10-11; ⇧Tab goes back a column) | `TestTableKeyboard` |

## Preferences

[Preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/) (page dated 2026-10-05).

| Setting | Plan | Status | Evidence |
|---|---|---|---|
| Theme: Sync with computer, Dark, Light | free | missing | – |
| Start at login | free | missing | – |
| Update channel: Stable, Beta, Alpha | free | missing | – |
| Locale timezone | free | missing | – |
| Command-line tool on or off ("Lens CLI") | paid | missing | – |
| HTTP proxy for non-cluster traffic | free | missing | – |
| Allow untrusted certificate authorities for non-cluster traffic | free | missing | – |
| kubectl: download binaries, download mirror (Google or Azure), directory for binaries, path to kubectl | free | missing | – |
| Helm repositories: pick public ones, add custom (name, URL, key, CA and certificate files, username and password; OCI registries) | free | missing | – |
| `HELM_DRIVER`: secret (default), configmap, sql | free | missing | – |
| Kubeconfig syncs | free | partial (2026-10-12): the Kubeconfig Sources dialog (Cluster menu › Kubeconfig Sources…, `kubeconfig.sources`) lists the added files and sync folders with their contexts and errors and removes them; Cogpit has no Preferences page yet. Lens (capture, 2026-10-12): Preferences › Kubernetes › MANAGE KUBECONFIGS, a "Sync Files and Folders" button and SYNCED ITEMS, one row per path with an icon (a folder, a file, a question mark for a file that no longer exists) and a trash icon that removes the entry without asking. Owner requirement (2026-10-12, built): Cogpit asks before removing, naming the path and saying that only the link is removed, never the file; the dialog lists `~/.kube` as the default sync folder, with its own Remove, and its "Sync Files and Folders…" button opens the same one dialog as the Cluster menu | [`fyneui/kubeconfigs.go`](../../internal/ui/fyneui/kubeconfigs.go); `TestSourcesDialogRemovesAfterConfirming`, `TestRemoveConfirmationNamesThePathAndSaysOnlyTheLinkGoes`, `TestRenderKubeconfigSources`, `TestRenderRemoveConfirmation` |
| Editor: show minimap and its position, line numbers, tab size, font size, font family | free | missing | – |
| Tables: row line height | free | missing | – |
| Terminal: shell path, shell arguments, copy and paste behaviour, theme, font size, font family | free | missing | – |
| Integrations: AWS EKS, Azure AKS, Google GKE | paid | missing | – |
| External tools: the AI command-line tool for Ask AI | paid | not a target (superseded by [D-10](../developer/decisions.md#d-10-built-in-agent)) | – |
| Extensions: install from file system, auto-update | paid | not a target ([D-5](../developer/decisions.md#d-5-extensions)) | – |
| Telemetry, surveys, error reports; Teamwork | free / paid | not a target ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | – |

## Cluster settings

[Cluster settings](https://docs.lenshq.io/k8slens/cluster/cluster-settings/). Captured (2026-10-11): the
settings open as a tab "Settings - <context>" with the cluster's icon; a page list on the left
holds General, Proxy, Terminal, Namespaces, Metrics, Lens Metrics, Cost and Node Shell, in that
order, and each page shows its settings under upper-case headings with a help line below each.

| Setting | Plan | Status | Evidence |
|---|---|---|---|
| General: cluster name (a text field, the context name until changed), icon (the image beside it, with a "…" menu: Upload Icon, Pick Icon Color, Reset Icon), kubeconfig path (in a box, as a link that opens the file manager at the file: Finder on macOS, owner; on Linux and Windows: inferred) (captures, 2026-10-11) | free | missing | – |
| Proxy: HTTP proxy for this cluster's API traffic, one field (placeholder "http://<address>:<port>") (capture) | free | missing | – |
| Terminal: starting working directory (a field with a folder picker, "$HOME" when unset), default namespace for kubectl ("default" when unset) (capture) | free | missing | – |
| Accessible namespaces (for users who may not list namespaces): a field that adds one namespace at a time (capture) | free | missing | – |
| Metrics provider: auto-detect or Helm 14.x, Helm, Lens, Prometheus Operator, StackLight, VictoriaMetrics, VictoriaMetrics single; service address `namespace/service:port`. Captured: a "Metrics source" dropdown (Automatic, Prometheus, Kubernetes Metrics Server, No metrics; under it a line naming the source in use), a "Prometheus" dropdown for the query format ("Auto Detect Prometheus"), a checkbox to filter empty containers out of the queries (on), the auto-detected provider and path (`<namespace>/services/<service>:<port>`), a checkbox for HTTPS requests (off), a custom path prefix | free | missing | – |
| Hide metrics from the UI: a multi-select of metrics, "Hide all metrics" and "Reset" buttons, and a line saying all metrics are visible when none is hidden (capture) | free | missing | – |
| Lens Metrics: install a bundled Prometheus, kube-state-metrics and node-exporter into the cluster on request; captured as three switches (all off) and an Apply button, greyed while nothing changed (inferred) | free | missing | – |
| Node shell: image (the default, shown as the field's placeholder, is the tag `docker.io/alpine:3.19`), an optional image pull secret in `kube-system` (capture) | free | missing | – |
| Cost: the source as a choice of Automatic (showing which source it uses), Kubecost, OpenCost, an estimate from node types and list prices (nothing to install), or Off, each with its state (Not found, Ready); rates: the hourly and monthly total for the cluster's nodes, an hourly rate per node type (prefilled, editable, "default" until changed), and fallback rates per core-hour, GiB-hour and GPU-hour for node types the price catalogue does not know (capture) | paid | missing | – |

## Cluster overview and charts

[Overview](https://docs.lenshq.io/k8slens/using-lens/overview/).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| CPU, memory and pod usage against requests, limits, allocatable and capacity | free | missing | – |
| Master and worker node charts | free | missing | – |
| Time range from 1 hour to 2 months | free | missing | – |
| Issues | free | missing | – |
| Metrics charts in detail drawers: pods, each container of a pod, nodes (captured); a time range (default "1h"), one tab per metric (CPU, memory, network, filesystem; a container has no network tab), a line naming the provider ("Displaying metrics from Prometheus: <namespace>/<service>:<port>"); legends: CPU usage, requests, limits, and for nodes allocatable capacity and capacity; a node's toolbar has four metric icons, then after a divider a flag icon with the tooltip "Show events" (presumably marks the events on the chart: inferred), and the section a ⋮ menu; the namespace and the service in the provider line are links (capture of a node) | free | partial (2026-10-12, G12 first and third slice): Prometheus as the source when one is found (through the API server's service proxy), else the Metrics Server. Built: the section above the Properties of a pod and a node, a source line "Displaying metrics from Prometheus (<namespace>/<service>)" or "Displaying metrics from Kubernetes Metrics Server" (or "Metrics are not available: <why>" without a provider, forbidden, or when it does not answer), a range dropdown (default 1h as captured; 5m, 15m, 30m, 1h, 3h, 6h, 12h, 1d, 2d, 7d, 14d, 30d with Prometheus, up to 1h with the Metrics Server: values beyond the default inferred), each chart's range held by the engine (`ui.setMetricsRange`, `ui.state`'s `drawerCharts`), the buttons CPU and Memory, Network and Filesystem enabled with Prometheus (else disabled with the tooltip "needs Prometheus"); Network as Receive and Transmit in bytes per second, Filesystem as Usage in bytes or Read and Write per second, whichever the cluster's Prometheus has; a container's Network chart shows the pod's with the line "Network is per pod" (Lens has no network tab on a container: Cogpit's, owner's requirement) (`TestPrometheusSourceAndVariants`, `TestPrometheusFilesystemIO`, `TestVariantsFollowTheProvider`, `TestMetricsRangeIsEngineState`, `TestHoverNetworkAndFilesystem`, `TestRenderPrometheusMetrics`); a chart (canvas lines, a value axis from zero, a time axis, a legend) of the series the engine gathers while the drawer is open (one point per 15 s, at most an hour), the requests and limits as dashed lines (a pod: the sums, the limit only when every container sets one; a node: allocatable capacity and capacity), a compact chart under each container of a pod, each chart (the pod's and every container's) with its own row of controls (range, the variant buttons, the flag; 2026-10-12, owner: "no main button"; the container's row is small: a range button with a menu, CPU, Mem, Net, FS), the flag button ("Show events", inferred) marking the object's events as yellow vertical lines with a diamond (`TestPodDrawerMetricsSection`, `TestMetricsVariantsRangeAndEvents`, `TestMetricsSectionStates`, `TestNodeDrawerMetricsLines`, `TestMetricsEventUpdatesInPlace`, `TestEveryChartHasItsOwnControls`, `TestLateContainerStartsWithCPU`, `TestChartDrawsWhatItIsGiven`, `TestRenderMetricsDrawer`, `TestRenderDemoMetrics`; hovering a chart (2026-10-12, owner): a crosshair on the nearest data point, a dot per series and a panel with the exact time (local, seconds, "N s ago"), each series' value in the axis's units (cores or millicores, binary bytes), the request and limit, and the event near a marker (`TestHoverSnapsToTheNearestPoint`, `TestHoverNamesTheEventNearAMarker`, `TestHoverOutHidesAndDataRefreshKeepsTheCrosshair`, `TestRenderChartHover`; [`chart_hover.go`](../../internal/ui/fyneui/chart_hover.go)); [`metricsview.go`](../../internal/ui/fyneui/metricsview.go), [`chart.go`](../../internal/ui/fyneui/chart.go), [`engine/metrics.go`](../../internal/engine/metrics.go)). Missing: history from before the drawer opened with the Metrics Server (it keeps none; a closed drawer loses its series), the port in the provider line and its namespace and service as links, the ⋮ menu, the chart's fill |

## Appearance and application

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Dark and light themes following the system | free | partial: command-line flag | `--theme dark\|light\|system` ([`main.go`](../../cmd/cogpit/main.go), [`theme.go`](../../internal/ui/fyneui/theme.go), `TestThemeModeIgnoresSystemVariantUnlessSystem`); no preference UI; owner requirement (2026-10-12, [D-22](../developer/decisions.md#d-22-colour-theme)): Solarized Dark and Light, not built |
| Zoom in, out and reset; full screen (capture needed) | free | missing | – |
| Several windows (Window menu) | free | missing | – |
| Deep links open the right view (Lens: `lens://`; Cogpit: `cogpit://`) | free | missing | – |
| Notifications in the title bar | free | missing | – |
| Safe mode (`--safe-mode`, extensions off) | free | not a target (D-5) | – |
| Ask AI: title-bar button and row actions launch the configured AI command-line tool | paid | not a target (superseded by [D-10](../developer/decisions.md#d-10-built-in-agent)) | – |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-12 | minor | Drawer charts (G12, third slice): Prometheus as the source, Network and Filesystem, ranges to 30 days held by the engine; the values beyond 1h inferred; the row stays partial, no count moved |
| 2026-10-12 | minor | Owner's smoke test: the sidebar's groups start closed, the status bar's text grows to 13 points; no count moved |
| 2026-10-12 | minor | Recount after the fifth run (lead): 56 of 133 rows exist, 55 of the 120 free rows (47 partial, 8 done) |
| 2026-10-12 | minor | Scope U (G5): the drawer's and the sidebar's widths are the user's, dragged and stored; no row changes status |
| 2026-10-12 | minor | Scope C2 (G2), following the Kubeconfig Syncs capture: dot files read, a connection closed when its file goes, one dialog for files and folders, `~/.kube` synced by default, a connect's log behind a warning icon, a Remove confirmation naming the path (owner requirement); the two rows stay partial with their remaining differences named; no count moved |
| 2026-10-12 | trivial | Status colours row: Solarized's accents, contrast held by a test (D-22 built); no count moved |
| 2026-10-12 | minor | Owner requirements noted on the status bar (D-23) and the themes (D-22) |
| 2026-10-12 | minor | Kubeconfig Syncs captured (owner): the file dialog for files and folders, no subfolders but dot files, new files within 1–2 s, a removed file's cluster closed, plugins run without asking, "Connection Updates"; the Preferences list; owner requirement: confirm removing a synced item |
| 2026-10-12 | minor | Adding a kubeconfig by pasting: not a target (owner, D-21); 54 of 120 free rows |
| 2026-10-12 | minor | The paste dialog and its menu items removed (D-21); the Kubeconfig Sources row names files and sync folders only; no row changed state |
| 2026-10-12 | major | Recount after the fourth run (`make parity`): 56 of 134 rows, 55 of the 121 free rows (47 partial, 8 done) |
| 2026-10-12 | minor | Fourth run, the third run's leftovers: no "+" on Port Forwarding and Helm Releases; the View menu hides the kinds the sidebar hides |
| 2026-10-12 | trivial | Dock: the Helm Upgrade tab built (fourth run) |
| 2026-10-12 | minor | Fourth run (agent D, G2): Kubeconfig Syncs, adding a kubeconfig by pasting and the Kubeconfig syncs preference partial; Lens's dialogs not captured, their layout inferred |
| 2026-10-12 | minor | `make parity`: Keyboard shortcuts with Plan and Evidence columns, so that the two paid shortcuts no longer read as free (the 121 free rows, reproduced); evidence named for kubeconfig loading, terminal copy and paste, and the theme flag; the theme row's pipes escaped |
| 2026-10-12 | trivial | Dock: Lens's Helm Upgrade tab captured |
| 2026-10-11 | minor | Third run, tables and window (G5): the macOS title bar partial (built, not seen in a real window yet); Tables: no namespace filter on cluster-scoped views, show and hide columns, fit a column, CSV export partial, keyboard done (and its shortcut row); the sidebar's Workloads › Overview |
| 2026-10-11 | minor | Logs (G8): timestamps, word wrap, previous container, logs of a controlling resource with resource names, filter and `/`, download partial, as engine-held options of the tab; the drawer's Copy |
| 2026-10-11 | minor | Terminal and shells: attach to a container as a dock tab (G8) |
| 2026-10-11 | minor | Cluster sidebar: Helm › Releases (Charts missing); the dock's Helm tabs stay missing |
| 2026-10-11 | minor | Port forwarding (G10): the port link and the Forward dialog done, the Port Forwarding view partial (status texts and Edit's dialog inferred); the drawer's Forward… done; Port Forwarding in the sidebar |
| 2026-10-11 | minor | Sidebar: kinds the user may not list are left out (access reviews); the row stays partial |
| 2026-10-11 | minor | Third run (agent D): drawer tables of cells (links, status colours, the warning icon), the first in a node's Pods table; links to claims, volumes, storage classes, pods and endpoint slices |
| 2026-10-11 | minor | Access Control views (G6): the warning column on Cluster Roles, Roles and Cluster Role Bindings; links to Service Accounts lead to their view; drawer tables link their cells |
| 2026-10-11 | minor | Cluster sidebar: the admission policy views follow the served API version (validating ones fall back to v1beta1 and v1alpha1, mutating ones need v1) |
| 2026-10-11 | minor | Title bar: the owner's requirement that Cogpit draws into the macOS title bar, logo beside the window buttons |
| 2026-10-11 | major | Recount after the second implementation run: 37 of 121 free rows (3 done), the paid diff view partial |
| 2026-10-11 | minor | Gap G5: cell tooltips and right-click done; row menus from the registry, the warning column, links in cells, the drawer's Properties, folds, links, toolbar icons and events order partial |
| 2026-10-11 | minor | G9: Edit tab with review, dry run and apply; Create resource tab with Cogpit's templates; the dock's "New tab" menu and the list's "+" button (8 rows) |
| 2026-10-11 | minor | Cluster sidebar: Custom Resources with its API groups, Network › Gateway API and Config › Admission Policies as sub-groups; custom kinds follow the cluster's definitions while connected |
| 2026-10-11 | minor | Cluster settings: the metrics source choices, the icon menu, the kubeconfig link opens the file manager (owner) |
| 2026-10-11 | minor | Captures of the Forward dialog and the metrics toolbar's "Show events" tooltip |
| 2026-10-11 | minor | Captures of every cluster settings page: the page list and each page's settings |
| 2026-10-11 | minor | Captures of a cordoned node: Uncordon in the drawer, tab tooltips, the node metrics toolbar |
| 2026-10-11 | minor | Namespace links narrow the view's namespace filter (owner); the navigator's scope dropdown and item tooltips |
| 2026-10-11 | minor | Captures of row menus, the warning icon and statuses: menu layout with submenus, Evicted in red, the navigator's cluster root |
| 2026-10-11 | minor | Captures of the Events view and of Create resource: cell tooltips, the + button, the template picker |
| 2026-10-11 | minor | Captures of the pod and node drawers and the window frame: drawer sections, folding and chips, links and Forward buttons, metrics in drawers, title bar, hotbar |
| 2026-10-11 | minor | Captures of Pods, Pod Disruption Budgets, Definitions and the Workloads Overview: object links, the warning column, the "+" button, the sort arrow |
| 2026-10-11 | minor | From a capture of a custom resource view: navigator header, tab titles, search toggles, row checkboxes, row and column menus, CSV button, namespace links, dock controls |
| 2026-10-11 | minor | Cluster sidebar from a second capture on Kubernetes 1.36.2: Mutating Admission Policies (version-dependent), the Gateway API sub-group (CRD-dependent), custom resource kinds by singular Kind name |
| 2026-10-11 | minor | Cluster sidebar from a capture of the running Lens Desktop (owner): Flux placement and structure, Endpoint Slices before Endpoints, Admission Policies sub-group, Custom Resources › Definitions and groups |
| 2026-10-11 | minor | The MVP: layout, sidebar, table, drawer, dock, follow, terminal and shell rows partial (24) |
| 2026-10-11 | major | Sources narrowed to Lens Desktop (D-12): Freelens rows and details removed, sidebar tree in documentation order; Scope column replaced by Plan; Ask AI superseded by the built-in agent |
| 2026-10-11 | major | First version, from Lens Desktop's documentation and release notes to 2026.10.51501 and Freelens 1.10.3 source |
