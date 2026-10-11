# UI parity: layout, navigation, tables, dock and feel

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (second implementation run: 38 of 134 rows) · **Target:** Lens Desktop 2026.10.51501

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

**Summary (measured, 2026-10-11, after the second implementation run):** 38 of 134 rows exist
(was 26): 37 of the 121 free-plan rows (34 partial, 3 done: the cell tooltip, right-click on a
row, the dock's "New tab" menu, each matching its captured row) and one paid row (the diff view,
partial); 6 more rows are not a target.
Remaining (estimated): see gaps [G5](gaps.md#g5-app-shell-navigation-tables-and-drawer) (layout,
sidebar, tables, drawer, dock), [G13](gaps.md#g13-clusters-hotbar-palette-menus-shortcuts-and-preferences)
(clusters, hotbar, palette, menus, shortcuts, preferences, cluster settings),
[G8](gaps.md#g8-logs-terminal-and-shells) and [G9](gaps.md#g9-yaml-editor-and-create-resource).

## Layout

Lens Desktop's regions ([layout](https://docs.lenshq.io/k8slens/using-lens/layout/)): title bar,
hotbar, navigator, tab bar, workspace, dock, status bar.

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Title bar with history back and forward, notifications and a context menu with Preferences; captured (2026-10-11): back and forward arrows, an "Update" button with a menu when an update is available, the paid "AI" button with a menu, a notifications bell, and on the right the vendor's "UPGRADE" and account avatar (not targets, D-9). Lens draws its bar into the window's title bar, beside the macOS window buttons (captured). Owner requirement (2026-10-11): Cogpit takes over the title-bar area too: no standard macOS title bar above Cogpit's top bar, the window draggable by Cogpit's own bar, and Cogpit's logo beside the window buttons, not below them. Fyne 2.8.1 hands out the `NSWindow` (`driver.NativeWindow.RunNative`, `MacWindowContext`), so a full-size content view with a transparent, hidden title is possible from `internal/ui` (verified in Fyne's source); whether dragging works without extra code is not verified | free | missing: Cogpit shows the standard macOS title bar above its own top bar | – |
| Hotbar down the left edge ([Hotbar](#hotbar)); captured: one tile per cluster with its icon or image and a short label, and a bank switcher ("‹ 1 ›") at the bottom | paid | missing | – |
| Navigator with the cluster's resource tree ([Cluster sidebar](#cluster-sidebar)) | free | partial | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go) |
| Navigator header: a dropdown of the scope the tree shows, one cluster or a section such as "Local Kubeconfigs", and a search button; the tree's root is the cluster, with its icon, a connection dot and a ›, and a ‹ to go back; hovering an item shows "<item> - <context>" (captures, 2026-10-11) | free | missing | – |
| Tab bar of cluster views: single click opens a preview tab, double-click (or ⇧↵) a fixed tab; a tab reads "<view> - <context>" with the cluster's icon and a close button; the preview tab's title is italic; hovering a tab shows its full title (captures, 2026-10-11; the italic-means-preview reading is inferred) | free | missing | – |
| Workspace showing the active view | free | partial | [`tableview.go`](../../internal/ui/fyneui/tableview.go); [PNG](../../internal/ui/fyneui/testdata/demo_pods.png) |
| Detail drawer on the right ([Detail drawer](#detail-drawer)) | free | partial | [`drawer.go`](../../internal/ui/fyneui/drawer.go) |
| Dock along the bottom ([Dock](#dock)) | free | partial | [`dock.go`](../../internal/ui/fyneui/dock.go) |
| Status bar | free | partial | [`bars.go`](../../internal/ui/fyneui/bars.go): context, API requests, informers, snapshots, demo events/s |
| Welcome page (Help › Welcome) | free | missing | – |

## Clusters

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Kubeconfig files in `~/.kube/` (`%USERPROFILE%\.kube` on Windows) are found without setup | free | partial: `$KUBECONFIG` or `~/.kube/config` only | `$KUBECONFIG` or `~/.kube/config` only (client-go's loading rules), not every file in `~/.kube/` |
| Kubeconfig Syncs: add files and folders to watch | free | missing | – |
| Add clusters by pasting a kubeconfig; Lens stores the text itself, not in `~/.kube/config` | free | missing | – |
| Navigator sections: Kubernetes Clusters (Local Kubeconfigs and the cloud sections), Teamwork, Extensions | free | missing | – |
| Search clusters by name (⌘P) | free | partial: filter in the context picker, no ⌘P | the context picker's filter field, [`picker.go`](../../internal/ui/fyneui/picker.go); no ⌘P |
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
| The tree above | free | partial: the kinds Cogpit has profiles for, in the captured order; Custom Resources as captured (Definitions, then one closed group per API group, sorted, its kinds by singular Kind name, sorted; the groups of curated sections listed too, selecting either entry opens the same view); Network › Gateway API and Config › Admission Policies as sub-groups, Config › Vertical Pod Autoscalers (a CRD kind); pages that are not kinds (Overview, Applications, Flux, Port Forwarding, Helm) are missing | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go) `navLayout`; `TestSidebarCustomResources`; PNGs [dark](../../internal/ui/fyneui/testdata/sidebar_custom_dark.png), [light](../../internal/ui/fyneui/testdata/sidebar_custom_light.png) |
| A group or item appears only when the cluster serves its kinds (Admission Policies, Gateway API, Device Resources) | free | partial: built-in kinds by discovery at connect, a kind's older API versions included where its profile lists them (2026-10-11: Validating Admission Policies and their bindings at v1, else v1beta1, else v1alpha1; Mutating Admission Policies and their bindings at v1 only, so they appear from Kubernetes 1.36 as captured: `TestBuiltinAdmissionPoliciesFollowTheServedVersion`); custom resource kinds (Gateway API, Vertical Pod Autoscalers, every API group of Custom Resources) follow the cluster's definitions while connected, where Lens needs a reconnect; no permission check | [`connector.go`](../../internal/engine/connector.go), [`customkinds.go`](../../internal/engine/customkinds.go); `TestSidebarFollowsKindsEvent`, `TestCustomKindLifecycle` |
| Groups expand and collapse | free | partial: the API groups of Custom Resources start collapsed (captured), the other groups open; open and closed states survive a change of the cluster's kinds, not a reconnect; the group of a view opened elsewhere (an agent, a link) opens to show it (inferred) | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go); `TestSidebarFollowsKindsEvent` |

## Tables

Every list view is one table ([resource views](resource-parity.md)).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Sort by a column header, ascending and descending arrows, empty values last | free | partial | `TestHeaderTapDispatchesSort`; engine `SortRows` with empty text and age cells last |
| Resize columns by dragging | free | partial | `TestColumnResizeByDraggingHeaderSeparator`; widths kept in memory only |
| Double-click a column separator to fit its content | free | missing | – |
| Reorder columns by dragging | free | missing | – |
| Show and hide columns (⋮ at the end of the header row) | free | missing | – |
| Search field per view, with toggles for match case ("Aa") and regular expressions (".*") (capture, 2026-10-11) | free | partial: no toggles | `TestSearchIsDebounced`; terms matched in any cell |
| A cell too narrow for its text shows the full text in a tooltip on hover (capture of an event message, 2026-10-11) | free | done | `TestTooltips`: after a hover delay (450 ms, Cogpit's choice) a cut-off cell shows its full text near the pointer; leaving the cell or the table, scrolling and changing the view close it ([`tooltip.go`](../../internal/ui/fyneui/tooltip.go)); Fyne 2.8 has no tooltip widget, Cogpit draws its own; [PNG](../../internal/ui/fyneui/testdata/tooltip_dark.png) |
| Namespace filter: multi-select, default "All namespaces", Space toggles without closing | free | partial | multi-select popup, default all ([`tableview.go`](../../internal/ui/fyneui/tableview.go)); Space not verified |
| Actions menu per row (⋮ at the row's end), starting with Show Details and ending with Edit and Delete, the kind's own actions between; items with a › open a submenu (captures of a custom resource, a pod and a cron job, 2026-10-11) | free | partial: the items are only as complete as the registry's commands (Edit and Delete on every kind, node and workload actions come with their commands; node shell is not built); acts on its row, not on a multi-selection; a disabled item says why after its label | `TestRowMenuNodes`, `TestRowMenuCronJobs`, `TestRowMenuPodsContainerSubmenu`, `TestMenuColumnOpensMenuWithoutSelecting`: Show Details, then `Registry.Actions` of the kind by `Order`, hidden when not applicable (Uncordon replaces Cordon), greyed when disabled; a command with a `container` parameter opens a submenu of the containers and init containers, one that needs more parameters a schema-generated dialog (`TestScaleDialogPrefilled`) ([`rowmenu.go`](../../internal/ui/fyneui/rowmenu.go), [`paramdialog.go`](../../internal/ui/fyneui/paramdialog.go)); [PNG](../../internal/ui/fyneui/testdata/row_menu_dark.png), [dialog](../../internal/ui/fyneui/testdata/scale_dialog_dark.png) |
| Right-click on a row opens the actions menu (since 2025.10.230725) | free | done | `TestRowMenuNodes` opens the menu with a secondary tap on a row; the menu is the ⋮'s |
| Keyboard: ↑↓ rows, Tab columns, ↵ or Space opens the details | free | missing | – |
| Export the table as CSV (a download button beside the item count, "18 items"; the format is inferred) | free | missing | – |
| Select rows with a checkbox column, all rows with the header's checkbox (capture, 2026-10-11) | free | missing | – |
| A warning column right after Name in built-in views (Pods, Pod Disruption Budgets, Cron Jobs): an orange triangle on rows with a problem (seen on a pod with many restarts and a container not ready; the rule is inferred) | free | partial: the rules are inferred; Cogpit's icon is Fyne's warning triangle in the warning colour; Pod Disruption Budgets and Cron Jobs come with their views | Pods and Nodes (`TestPodWarningTexts`, `TestNodeWarnings`): the icon centred in a narrow column without a header title, its reason as the tooltip (`TestTooltips`); inferred rules in [resource parity](resource-parity.md#cluster) ([`pods.go`](../../internal/model/kinds/pods.go) `podWarning`, [`nodes.go`](../../internal/model/kinds/nodes.go)); [PNG](../../internal/ui/fyneui/testdata/pods_list_dark.png). Cluster Roles, Roles and Cluster Role Bindings (2026-10-11): a role or binding that grants nothing, inferred ([resource parity](resource-parity.md#access-control); `TestClusterRoleTexts`, `TestRoleTexts`, `TestClusterRoleBindingTexts`; [PNG](../../internal/ui/fyneui/testdata/clusterrolebindings_list_dark.png)) |
| A floating "+" button at the bottom right of every list opens a Create resource tab in the dock (capture, 2026-10-11) | free | partial: what Lens fills the tab with is not captured; Cogpit starts from the template of the view's kind, else the example Pod (Cogpit's choice) | [`createbutton.go`](../../internal/ui/fyneui/createbutton.go), `TestCreateTabTemplatesAndCreate`, `TestCreateButtonHiddenWithoutCluster`; [PNG](../../internal/ui/fyneui/testdata/create_dark.png) |
| The sorted column shows an arrow beside its title | free | partial | `TestHeaderTapDispatchesSort` |
| Cells that name another object are links: Namespace, Controlled By, Node, Involved Object, a definition's resource (captures, 2026-10-11); a Namespace link narrows the open view's namespace filter to that namespace (owner, 2026-10-11); where the others lead: capture needed | free | partial: Controlled By and Involved Object link built-in kinds only (a custom kind's key needs its plural, which a reference does not carry); a definition's resource comes with the Definitions view; Cogpit's links open the object (inferred) | `TestLinkClicks`: in the link colour, underlined and a hand pointer while hovered; a click on the text follows the link without selecting the row, beside it selects the row; the Namespace link sets `ui.setNamespace` to that namespace, others `ui.navigate`, a link without a name `ui.openView`; a link to a kind the cluster does not serve, or that Cogpit has no view for yet (Secrets), shows as plain text, and one that fails anyway shows an error toast ([`links.go`](../../internal/ui/fyneui/links.go)); Service Accounts have a view since 2026-10-11, so links to them lead there |
| Row line height (preference) | free | missing | – |
| Status colours per state (palette: capture needed; seen: "Running" green, "Evicted" red) | free | partial | status tokens mapped by `StatusColor` ([`theme.go`](../../internal/ui/fyneui/theme.go)); Cogpit's own palette |

## Detail drawer

[Details panel](https://docs.lenshq.io/k8slens/using-lens/details-panel/).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Opens from the right when a row is selected, beside the narrowed table; titled "<Kind>: <name>" (capture, 2026-10-11) | free | partial | `TestRowTapDispatchesSelect`; `ui.select` opens the drawer |
| Actions at the top: Copy, Edit, Delete and the kind's own actions; captured as icons in the drawer's title bar: for a pod seven icons ending with Edit and Delete, for a node Shell, Cordon (a pause icon; Uncordon, a play icon, on a cordoned node), Drain, Edit, Delete (which pod icon is which action: inferred, tooltips needed) | free | partial: Copy (`object.copyName`) is missing; the icons are Fyne's, chosen per command ([`cmdicons.go`](../../internal/ui/fyneui/cmdicons.go)), the pause and play for Cordon and Uncordon as captured | `TestDrawerToolbarFromRegistry`, `TestDrawerActionsDisabledWithReason`: the row menu's actions as icons in the title bar, a tooltip names each and says why a disabled one cannot run; [PNG](../../internal/ui/fyneui/testdata/node_drawer_dark.png) |
| Drawer sections in order: Metrics, Properties, the kind's own sections, Events ("No events found" when none) (capture of a pod and a node) | free | partial: no metrics (M7); the YAML stays a second tab (Cogpit's own) | `TestDrawerSectionsAsLens`, `TestDrawerNoEventsFound`, `TestBuildDetailPodAsLens`: Properties first (metadata and the kind's state merged, no UID), the kind's sections with group headings (Init Containers, Containers), Events last ([`drawer.go`](../../internal/ui/fyneui/drawer.go), [`engine/drawer.go`](../../internal/engine/drawer.go)); [PNG](../../internal/ui/fyneui/testdata/pod_drawer_dark.png) |
| Long values fold: Labels, Annotations, Tolerations, environment variables, mounts and volumes show a count ("6 Labels") and expand on click; lists of short values show as chips (pod IPs, conditions, images) (capture) | free | partial: images are not chips; an unfolded list shows chips (labels, volumes) or one line per entry (annotations, tolerations, environment, mounts), which is inferred | `TestDrawerSectionsAsLens` folds and unfolds the labels; pod IPs, conditions and secrets are chips (`TestPodDetailSections`) |
| Object references in the drawer are links (namespace, controller, node, service account, secrets); container ports have a "Forward…" button (capture) | free | partial: no Forward… (port forwarding is not built) | `TestDrawerSectionsAsLens` (the node link navigates), `TestPropertiesSection`, `TestPodDetailSections`: namespace, controller (built-in kinds), node, service account, secrets and a volume's ConfigMap, Secret or claim link to their objects; the service account link leads to the Service Accounts view (2026-10-11, `TestPodServiceAccountLinkLive`); a binding's role and its ServiceAccount subjects, in a table, link to them (`TestClusterRoleBindingDrawerLinks`; table cells link, `TestDrawerTableLinks`) |

## Dock

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Bottom dock with tabs: Terminal session, Create resource, Edit, logs, Helm Install, Helm Upgrade; tabs named after their object ("Pod <name>"), with a close button; on the right, maximize and collapse (capture, 2026-10-11) | free | partial: terminal, shell, logs, Edit ("ConfigMap demo") and Create resource; no Helm tabs, no maximize | `TestDockOpensAndClosesStreams`, `TestEditTabSaveReviewsThenApplies`, `TestCreateTabTemplatesAndCreate` |
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
| Pod and container selectors | free | missing | – |
| Follow; jump to the latest line | free | partial | [`termview/logs.go`](../../internal/ui/fyneui/termview/logs.go), follow toggle tested |
| Show timestamps | free | missing | – |
| Word wrap | free | missing | – |
| Show the previous terminated container | free | missing | – |
| Logs of a controlling resource (all its pods), with resource names shown | free | missing | – |
| Filter matching lines on or off (`/`) | free | missing | – |
| Download logs | free | missing | – |
| Jump to a time and date | free | missing | – |
| Load older lines when scrolling up | free | missing | – |

## Terminal and shells

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Terminal session for a cluster, with the working directory and default namespace from cluster settings | free | partial: context and namespace, no cluster settings | `TestStartLocalBindsKubectlToContextAndNamespace`, `TestKindEngineWorkflows`; namespace from the filter, no cluster settings |
| Pod shell, attach and node shell open as dock tabs | free | partial: pod shell only | `TestKindPodExec`, `TestKindEngineWorkflows`; no attach, no node shell |
| Copy and paste with the right mouse button (preference) | free | missing | – |
| Copy and paste keys: ⌘C / ⌘V; Windows and Linux ⌃C, ⌃⇧C or ⌃Ins, and ⌃V, ⌃⇧V or ⇧Ins | free | partial: macOS keys | fyne-io/terminal: ⌘C/⌘V on macOS (offscreen test, synthetic events); Ctrl+Shift+C/V on Linux and Windows |
| Theme, font size and font family (preferences) | free | missing | – |
| Shell path and shell arguments (2026.10: one argument per line) | free | missing | – |

## Port forwarding

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| A container port link ("<name>: <port>/<protocol>" in the drawer, captured) forwards to a random local port and opens the browser | free | missing | – |
| Forward dialog, from the "Forward…" button beside a container port (init containers too): titled "Port Forwarding for <pod>", a local port field ("Random" when empty), an "https" checkbox (off), an "Open in Browser" checkbox (on), Cancel and Start; the window behind is dimmed (capture, 2026-10-11) | free | missing | – |
| Port Forwarding view to open, edit, stop and delete forwards ([resource parity](resource-parity.md#network)) | free | missing | – |

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

| Action | Windows / Linux | macOS | Status |
|---|---|---|---|
| Command palette | Ctrl+Shift+P | ⌘⇧P | missing |
| Search a cluster by name | Ctrl+P | ⌘P | missing |
| Search in the tab | Ctrl+F | ⌘F | missing |
| Preferences | Ctrl+, | ⌘, | missing |
| Cluster settings | Ctrl+Shift+, | ⌘⇧, | missing |
| Ask AI (cluster tab, paid) | Ctrl+I | ⌘I | missing (opens Cogpit's agent chat, [D-10](../developer/decisions.md#d-10-built-in-agent)) |
| Terminal session (cluster tab) | Ctrl+Shift+T | ⌘⇧T | missing |
| Close the tab | Ctrl+W | ⌘W | missing |
| Reload | Ctrl+R | ⌘R | missing |
| Back / forward in the tab's history | Alt+← / Alt+→ | ⌃- / ⌃⇧- | missing |
| Next tab | Ctrl+PageUp (as documented) | ⌘⌥→ | missing |
| Previous tab | Ctrl+PageDown (as documented) | ⌘⌥← | missing |
| Tab *n* | Ctrl+*n* | ⌘*n* | missing |
| Cluster in the hotbar bank / switch bank (paid) | F1–F12 / Shift+F1–F12 | same | missing |
| New space (Teamwork) | Ctrl+N | ⌘N | not a target ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) |
| Quit | Alt+F4 | ⌘Q | missing |
| Table: rows / columns / details | ↑↓ / Tab / ↵ or Space | same | missing |

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
| Kubeconfig syncs | free | missing | – |
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
| Metrics charts in detail drawers: pods, each container of a pod, nodes (captured); a time range (default "1h"), one tab per metric (CPU, memory, network, filesystem; a container has no network tab), a line naming the provider ("Displaying metrics from Prometheus: <namespace>/<service>:<port>"); legends: CPU usage, requests, limits, and for nodes allocatable capacity and capacity; a node's toolbar has four metric icons, then after a divider a flag icon with the tooltip "Show events" (presumably marks the events on the chart: inferred), and the section a ⋮ menu; the namespace and the service in the provider line are links (capture of a node) | free | missing | – |

## Appearance and application

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Dark and light themes following the system | free | partial: command-line flag | `--theme dark|light|system`; no preference UI |
| Zoom in, out and reset; full screen (capture needed) | free | missing | – |
| Several windows (Window menu) | free | missing | – |
| Deep links open the right view (Lens: `lens://`; Cogpit: `cogpit://`) | free | missing | – |
| Notifications in the title bar | free | missing | – |
| Safe mode (`--safe-mode`, extensions off) | free | not a target (D-5) | – |
| Ask AI: title-bar button and row actions launch the configured AI command-line tool | paid | not a target (superseded by [D-10](../developer/decisions.md#d-10-built-in-agent)) | – |

## Revision history

| Date | Change | Summary |
|---|---|---|
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
