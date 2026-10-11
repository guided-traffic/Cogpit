# UI parity: layout, navigation, tables, dock and feel

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** minor (the MVP: 24 rows partial) · **Target:** Lens Desktop 2026.10.51501

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

**Summary (measured, 2026-10-11):** 24 of 124 rows exist, all **partial** and all free-plan rows
(24 of 111); 6 more rows are not a target. No row is done: none has been compared with a running
Lens Desktop.
Remaining (estimated): see gaps [G5](gaps.md#g5-app-shell-navigation-tables-and-drawer) (layout,
sidebar, tables, drawer, dock), [G13](gaps.md#g13-clusters-hotbar-palette-menus-shortcuts-and-preferences)
(clusters, hotbar, palette, menus, shortcuts, preferences, cluster settings),
[G8](gaps.md#g8-logs-terminal-and-shells) and [G9](gaps.md#g9-yaml-editor-and-create-resource).

## Layout

Lens Desktop's regions ([layout](https://docs.lenshq.io/k8slens/using-lens/layout/)): title bar,
hotbar, navigator, tab bar, workspace, dock, status bar.

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Title bar with history back and forward, notifications and a context menu with Preferences | free | missing | – |
| Hotbar down the left edge ([Hotbar](#hotbar)) | paid | missing | – |
| Navigator with the cluster's resource tree ([Cluster sidebar](#cluster-sidebar)) | free | partial | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go) |
| Navigator header: a dropdown of the cluster shown in the tree and a search button (capture, 2026-10-11) | free | missing | – |
| Tab bar of cluster views: single click opens a preview tab, double-click (or ⇧↵) a fixed tab; a tab reads "<view> - <context>" with the cluster's icon and a close button; the preview tab's title is italic (capture, 2026-10-11; the italic-means-preview reading is inferred) | free | missing | – |
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
| The tree above | free | partial: six kinds, in the captured order | six kinds: Nodes; Workloads › Pods, Deployments; Network › Services; Namespaces; Events; the rest of the order is laid out for coming profiles ([`sidebar.go`](../../internal/ui/fyneui/sidebar.go) `navLayout`) |
| A group or item appears only when the cluster serves its kinds (Admission Policies, Gateway API, Device Resources) | free | partial | [`connector.go`](../../internal/engine/connector.go) filters profiles by discovery |
| Groups expand and collapse | free | partial | [`sidebar.go`](../../internal/ui/fyneui/sidebar.go) |

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
| Namespace filter: multi-select, default "All namespaces", Space toggles without closing | free | partial | multi-select popup, default all ([`tableview.go`](../../internal/ui/fyneui/tableview.go)); Space not verified |
| Actions menu per row (⋮ at the row's end); for a custom resource: Show Details, Edit, Delete (capture, 2026-10-11) | free | missing | – |
| Right-click on a row opens the actions menu (since 2025.10.230725) | free | missing | – |
| Keyboard: ↑↓ rows, Tab columns, ↵ or Space opens the details | free | missing | – |
| Export the table as CSV (a download button beside the item count, "18 items"; the format is inferred) | free | missing | – |
| Select rows with a checkbox column, all rows with the header's checkbox (capture, 2026-10-11) | free | missing | – |
| Namespace cells are links (capture, 2026-10-11; where they lead: capture needed) | free | missing | – |
| Row line height (preference) | free | missing | – |
| Status colours per state (palette: capture needed) | free | partial | status tokens mapped by `StatusColor` ([`theme.go`](../../internal/ui/fyneui/theme.go)); Cogpit's own palette |

## Detail drawer

[Details panel](https://docs.lenshq.io/k8slens/using-lens/details-panel/).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Opens from the right when a row is selected | free | partial | `TestRowTapDispatchesSelect`; `ui.select` opens the drawer |
| Actions at the top: Copy, Edit, Delete and the kind's own actions | free | partial: Delete and kind actions | Delete and the kind's actions (Logs, Shell, Restart), disabled with the reason; no Copy, no Edit |

## Dock

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Bottom dock with tabs: Terminal session, Create resource, Edit, logs, Helm Install, Helm Upgrade; tabs named after their object ("Pod <name>"), with a close button; on the right, maximize and collapse (capture, 2026-10-11) | free | partial: terminal, shell, logs; no maximize | `TestDockOpensAndClosesStreams`; terminal, shell and logs tabs only |
| "New tab" menu: Terminal session, Create resource | free | partial: Terminal session | Terminal session only |
| Tab icons take the cluster's colour | free | missing | – |

## Editor and create resource

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| Edit tab: YAML editor with syntax highlighting and Save | free | missing | – |
| Diff view: "Show changes", "Revert to original", "Exit diff view" | paid | missing | – |
| Create resource tab: template picker, "Create & Close" | free | missing | – |
| Bundled templates, device-resource templates included (2026.9) | free | missing | – |
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
| A container port link forwards to a random local port and opens the browser | free | missing | – |
| Forward dialog (fields: capture needed) | free | missing | – |
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

[Cluster settings](https://docs.lenshq.io/k8slens/cluster/cluster-settings/).

| Setting | Plan | Status | Evidence |
|---|---|---|---|
| General: cluster name, icon, kubeconfig path | free | missing | – |
| Proxy: HTTP proxy for this cluster | free | missing | – |
| Terminal: working directory, default namespace | free | missing | – |
| Accessible namespaces (for users who may not list namespaces) | free | missing | – |
| Metrics provider: auto-detect or Helm 14.x, Helm, Lens, Prometheus Operator, StackLight, VictoriaMetrics, VictoriaMetrics single; service address `namespace/service:port` | free | missing | – |
| Hide metrics from the UI | free | missing | – |
| Lens Metrics: install a bundled Prometheus, kube-state-metrics and node-exporter into the cluster on request | free | missing | – |
| Node shell: image, image pull secret | free | missing | – |
| Cost | paid | missing | – |

## Cluster overview and charts

[Overview](https://docs.lenshq.io/k8slens/using-lens/overview/).

| Behaviour | Plan | Status | Evidence |
|---|---|---|---|
| CPU, memory and pod usage against requests, limits, allocatable and capacity | free | missing | – |
| Master and worker node charts | free | missing | – |
| Time range from 1 hour to 2 months | free | missing | – |
| Issues | free | missing | – |
| Metrics charts in detail drawers (which kinds: capture needed) | free | missing | – |

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
| 2026-10-11 | minor | From a capture of a custom resource view: navigator header, tab titles, search toggles, row checkboxes, row and column menus, CSV button, namespace links, dock controls |
| 2026-10-11 | minor | Cluster sidebar from a second capture on Kubernetes 1.36.2: Mutating Admission Policies (version-dependent), the Gateway API sub-group (CRD-dependent), custom resource kinds by singular Kind name |
| 2026-10-11 | minor | Cluster sidebar from a capture of the running Lens Desktop (owner): Flux placement and structure, Endpoint Slices before Endpoints, Admission Policies sub-group, Custom Resources › Definitions and groups |
| 2026-10-11 | minor | The MVP: layout, sidebar, table, drawer, dock, follow, terminal and shell rows partial (24) |
| 2026-10-11 | major | Sources narrowed to Lens Desktop (D-12): Freelens rows and details removed, sidebar tree in documentation order; Scope column replaced by Plan; Ask AI superseded by the built-in agent |
| 2026-10-11 | major | First version, from Lens Desktop's documentation and release notes to 2026.10.51501 and Freelens 1.10.3 source |
