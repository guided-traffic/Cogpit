# Architecture

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-13 · **Change:** minor (§4.3 the notification log of every toast, `UIState.Notifications`, what stays in the window; §4.3 `UIState.ColumnOrder` and the snapshot's `Order`; the window's row checkboxes, §4.3 and Current state; the table's selection, G5, G7, G9: §4.2 actions on a selection and `object.deleteMany`, §4.3 `ui.select`'s modes; the welcome page and Disconnect, G5 and G13, §4.3: `UIState.Welcome`, `ui.welcome.open`, `cluster.disconnect`, the window's page and its Help and Window menus; the welcome page and Disconnect, G5 and G13, §4.3: `UIState.Welcome`, `ui.welcome.open`, `cluster.disconnect`; §4.4 a logs tab's source, narrowed in the merged stream, timestamps on every stream, its older lines and a jump to a time; §4.4 a logs tab's source, narrowed in the merged stream, timestamps on every stream, and its older lines; §4.4 a logs tab's source, narrowed in the merged stream, and timestamps on every stream; §8: a release upgraded to a chart version from the user's repositories, the Upgrade tab's choice of repository and version, G11; the tab bar of cluster views, G5, §4.3: `UIState.Tabs` and `ActiveTab`, kept in step by `syncTabs`, the `ui.tab.*` commands, `ui.openView` in the preview tab, the window's bar (`tabbar.go`) and its View menu keys, §4.2; the tab bar of cluster views, G5, §4.3: `UIState.Tabs` and `ActiveTab`, kept in step by `syncTabs`, the `ui.tab.*` commands, `ui.openView` in the preview tab; Preferences and cluster settings, G13, §4.3: engine state, stores, commands, the GUI that applies them; the cluster Overview, G12: §4.3 the page and `UIState.Overview`, §9 the cluster scope of the metrics hub; §5 a virtual kind made of several kinds, the Applications; §4.5 curated profiles of CRD kinds, the Vertical Pod Autoscalers; Current state: the four Device Resources views, 45 built-in views; §8: Helm chart repositories kept in Cogpit's own folders, the Charts view and the install, G11; the command palette, G13, §4.2: its rules from the registry's metadata, the shortcuts bound from the registry, §4.3: `UIState.Palette`; `cogpit-cli` contexts, get and the MCP stdio bridge, §7, and `internal/app`, the wiring both binaries share: Current state, §3; the navigator, §4.3: its search, keyboard row and open groups are the window's, its tree Cogpit's own rows; one line in Current state; the hotbar, G13, §4.3: `UIState.Hotbar`, `hotbar.json`, the `hotbar.*` commands and the bar left of the navigator; Current state after the fifth run's evening: crash reports, the external MCP server switched in the status bar, the window's metrics and chart hover, the quit crash fixed; G12 third slice, §9: Prometheus in the window, ranges as engine state, checked against a real Prometheus on kind; G12 second slice, §9: Prometheus through the service proxy, backend; G12 first slice, §9 as built: the Metrics Server provider, the hub that polls only while a view or drawer shows metrics, CPU and Memory columns, the drawer's Metrics section, `metrics.get`, the demo's synthetic Metrics Server; D-28 built, §6: every context from a source file read alone, `~/.kube/config` a file of the default folder, `$KUBECONFIG` and `--kubeconfig` files session sources, bare names resolved; Kubeconfig Syncs second pass, §6: dot files, `~/.kube` by default, a connection closed with its file, the connect log; related rows rebuilt when the indexed kinds finish listing; a stopped shell drops its queued work; fourth run: kubeconfig sources and the approval of their credential plugins, §6; an upgrade from a tab bound to the tab's text revision, values scrubbed in their quoted forms too, upgrade and rollback checked against the server's newest revision, the Rollback dialog's answer bound to its revision, §8; a revealed Secret value bound to the Secret's version, §5; pasting a kubeconfig removed again, D-21, §6) · **Target:** Lens Desktop 2026.10.51501

## Current state (2026-10-12)

The local MVP is built and was extended by a second and a third implementation run (2026-10-11),
a fourth and a fifth (2026-10-12):
a macOS desktop app that connects to a kubeconfig context or a synthetic demo cluster, shows 45
built-in views (41 until the four Device Resources views of 2026-10-13), the Port Forwarding view (a local kind), the Helm Releases view (a virtual kind), the Applications view (a virtual kind made of several kinds, 2026-10-13),
the Workloads Overview page and one view per custom resource kind from a watch-fed cache, greys
what the user's access reviews deny, has a detail drawer and row menus from the command registry,
a dock with logs (with their options, pod and container selectors, older lines, a jump to a time, and the logs of a workload's pods), a local terminal, pod
shells, attach and Edit and Create resource tabs, writes through 138 commands (reads included) (editing, Secrets
from YAML in the GUI only, delete, node and workload actions, a crash-safe drain, port forwards
that fail at once on permanent errors, Helm rollback, uninstall and upgrade with a crash journal
locked between Cogpit processes, each bound to the revision the server holds), adds and syncs
kubeconfigs (`~/.kube` by default, `config` included, and the files `$KUBECONFIG` names) whose credential plugins and outside token files run or are
read only once the user allowed them, bound to the programs' content inside the folder, holds
Secrets in a full watch that Helm's releases share, draws its own title bar on macOS with the C
helm logo in Solarized colours, lets the user copy any value from the drawer (a Secret's marked
concealed on the clipboard), and an MCP server lets agents drive all of it; what an agent asks the
user to accept is bound to what runs (text revisions, inline values shown). Since the fifth run's
evening (2026-10-12) it draws the drawer's resource charts from the Metrics Server or a Prometheus
it finds in the cluster (CPU, Memory, Network, Filesystem; ranges to 30 days held by the engine; a
hover panel with the exact values), keeps a crash report when it crashes, and lets the user switch
the external MCP server off in the status bar.
Since the night run (2026-10-12) a hotbar of pinned clusters in banks of twelve sits at the window's left edge (G13): engine state (`UIState.Hotbar`, nine `hotbar.*` commands, `<user config dir>/Cogpit/hotbar.json`), F1–F12 and ⇧F1–⇧F12 ([§4.3](#43-ui-state)).
Since 2026-10-13 Helm charts can be installed (G11, §8): public https repositories the user adds, kept in Cogpit's own folders (`helm.repo.*`, Helm › Repositories…), their charts as the local kind `helmcharts` (Helm › Charts), and a Helm Install dock tab whose install is previewed with the SDK's server-side dry run and runs crash-safe through the Helm journal; a release whose chart those repositories list is upgraded to a version of theirs, downloaded and checked against the index's digest (`helm.release.upgrade` with `version` and `repository`).
Since 2026-10-13 the navigator has Lens's header (a scope dropdown and a search), the cluster as the tree's root, "<item> - <context>" tooltips and its keyboard, in Cogpit's own tree rows ([`navigator.go`](../../internal/ui/fyneui/navigator.go); its search and keyboard are the window's, [§4.3](#43-ui-state)).
Since 2026-10-13 a command palette (⌘⇧P, G13) lists what the menus offer by the registry's metadata and runs it as the menus do; its open state is `UIState.Palette`, and the registry's shortcuts ⌘⇧P and ⌘P sit on main-menu items ([§4.2](#42-the-command-registry)).
Since 2026-10-13 the cluster Overview (`cluster/overview`, the navigator's first item, G12) shows the cluster's CPU, Memory and Pods for its master or worker nodes over a range, through a cluster scope of the metrics hub, and the Issues from the cache; its range and node group are `UIState.Overview` ([§4.3](#43-ui-state), [§9](#9-metrics-internalmetrics)).
Since 2026-10-13 Preferences (⌘,) and a cluster's Settings (⌘⇧,) are engine state and commands kept in Cogpit's own `preferences.json`, `cluster-settings.json` and `cluster-icons/` (G13): the theme switched live, the time zone, the tables' row height, the editor's and the terminal's settings, a cluster's display name and icon in the hotbar, the picker and the navigator, its terminals' directory and namespace ([§4.3](#43-ui-state)).
Since 2026-10-13 a welcome page fills the workspace while it has nothing else to show (no cluster connected, no tab active) and from Help › Welcome (`UIState.Welcome`, derived; `ui.welcome.open`); the Help menu (Welcome, Documentation, About Cogpit) and, on Windows and Linux, a Window menu are the window's, macOS keeping GLFW's own Window menu; a cluster's menus have Disconnect (`cluster.disconnect`) and Remove from Cogpit (`cluster.remove`, which unlinks a kubeconfig file the user added, never the file) (G5, G13; [§4.3](#43-ui-state), [§6](#kubeconfig-sources-built-2026-10-12)).
Since 2026-10-13 every list has row checkboxes and a header checkbox (G5): the selection is `UIState.Selected`, set through `ui.select` with modes, and Delete of a selection (`object.deleteMany`, G9) asks once, naming every object ([§4.2](#42-the-command-registry), [§4.3](#43-ui-state), [`tablecheck.go`](../../internal/ui/fyneui/tablecheck.go)).
Since 2026-10-13 a tab bar of cluster views sits above the workspace (G5): a click in the navigator opens a preview tab, a double click or ⇧↵ a fixed one, each tab keeps its own filter, selection and history, and the tabs are engine state (`UIState.Tabs`, the `ui.tab.*` commands; [§4.3](#43-ui-state)); a tab of another cluster connects to it when shown, since Cogpit holds one connection.
Since 2026-10-13 the top bar has the active tab's history arrows (`ui.tab.back`, `ui.tab.forward`), a context menu with Preferences… on a right-click, and a notifications bell whose panel lists the session's toasts: a bounded log of every toast, the engine's and the window's own, that is engine state (`UIState.Notifications`, the `notifications.*` commands), which agents read with the window's own texts and a Helm action's error withheld ([§4.3](#43-ui-state), [`titlehistory.go`](../../internal/ui/fyneui/titlehistory.go), [`notifications.go`](../../internal/ui/fyneui/notifications.go)).
CI runs on every push ([releasing › CI](releasing.md#ci)). The MVP doubled as the Fyne spike
([results](spike-fyne.md#results)). Where the code differs from the plan below, this section wins.

| Package | State | Differs from the plan |
|---|---|---|
| [`internal/model`](../../internal/model/), [`model/kinds`](../../internal/model/kinds/) | Built: refs, rows (links, coloured parts, marks (a pod's containers as squares, 2026-10-12), warning and until columns), status tokens (`stopped` added 2026-10-12 for a stopped container), panic-free accessors, Secret redaction, the drawer's Properties; 16 built-in profiles (Nodes, Pods, Deployments, Daemon Sets, Stateful Sets, Replica Sets, Jobs, Cron Jobs, Services, Ingresses, Ingress Classes, Config Maps, Persistent Volume Claims, Namespaces, Events, Custom Resource Definitions) and a generic profile built from a custom resource definition and its printer columns, with golden and fuzz tests; the third run added 25 built-in profiles (the Config, Access Control, Network and Storage views, Secrets (in full since the fourth run, with Keys and Type), Replication Controllers; 41 in all; the four Device Resources kinds at `resource.k8s.io/v1` on 2026-10-13, 45), the local kind Port Forwarding, the profile fields of §5 (scopes, `Derive`, `Local`, `FallbackVersions`) and the related-objects indexes of §4.4, drawer tables of cells, item actions and private items | No CPU and memory columns; no generic view for built-in kinds without a profile or for aggregated APIs; Cron Jobs' Next execution uses `time.LoadLocation` (the zone database is embedded in `cmd/cogpit` for Windows) |
| [`internal/kubeconfig`](../../internal/kubeconfig/) | Built: contexts from bytes, error text without credentials, the per-terminal kubeconfig; fourth run (2026-10-12): the exec credential plugin of a context with its fingerprint, untrusted kubeconfig text made safe to show ([`exec.go`](../../internal/kubeconfig/exec.go), [`source.go`](../../internal/kubeconfig/source.go)) | – |
| [`internal/cluster`](../../internal/cluster/) | Built: connections through client-go's loading rules, the counting transport, discovery, log streams, restart and delete; for every kind, a REST mapper over in-memory cached discovery and update (with a resourceVersion precondition), create (both with dryRun=All) and delete (with a UID precondition) through the dynamic client ([`objects.go`](../../internal/cluster/objects.go)); an update sent without `managedFields` (the cache drops them) keeps every manager's entry on the server and records Cogpit's fields under the manager `cogpit` (verified on kind 1.36.1, `TestKindEditCreateConflictDelete`); access reviews (`ReviewRules`, `ReviewAccess`, [`access.go`](../../internal/cluster/access.go)); third run: the metadata client for metadata-only informers, port-forward tunnels on loopback listeners Cogpit owns across reconnects ([`portforward.go`](../../internal/cluster/portforward.go)), restarts of Daemon and Stateful Sets, Replica Set scale, the API server's warnings logged once per connection; fourth run (2026-10-12): one kubeconfig file read on its own (`InspectFile`), sync folder listings, and `Options.ExecPolicy`, asked before a context's exec plugin can run ([`sources.go`](../../internal/cluster/sources.go), §6) | Discovery is cached in memory for 10 minutes, not on disk; one rate limiter per cluster (QPS 5, burst 10) shared by every client, discovery with its own budget (50/300), access reviews with theirs (10/20, so that they never delay a view's list); implicit kubeconfig writes are switched off (the legacy `.kubeconfig` migration and the oidc persister, which keeps refreshed tokens in memory) |
| [`internal/cache`](../../internal/cache/) | Built: one shared informer per scope, handles, warm period, `managedFields` transform, no resync, streaming initial lists where the server supports them, error state, stats; a handle reports synced only once the initial list reached its subscribers; third run: scopes with label selectors and metadata only (§5), `cache.Memory` for local kinds, the `last-applied-configuration` annotation dropped from Secrets before they enter the store; fourth run: covering scopes (a label-selected scope served from its running unselected scope, with handovers both ways, §5) | Views watch all namespaces and filter client-side; no polling sources yet |
| [`internal/terminal`](../../internal/terminal/) | Built: local shell on Cogpit's own PTY with the per-terminal kubeconfig; pod shell over exec (WebSocket, SPDY fallback); attach to a container's main process (third run), with stdin and a TTY as the container has them | No node shell; Windows has no local shell yet |
| [`internal/demo`](../../internal/demo/) (L2, new) | Built: the synthetic cluster for `--demo` behind a `cache.Source`: 10,000 pods, 300 services, 1,000 watch events per second, watch resume and 410 Gone; since 2026-10-12 also a synthetic Metrics Server (`demo/metrics.go`: usage of running pods and ready nodes from a hash, waves and noise, a measurement per 15 s) | A package the plan did not list |
| [`internal/engine`](../../internal/engine/), [`engine/commands`](../../internal/engine/commands/) | Built: Session, registry with JSON schemas, `Order`, `Escalate`, `Prefill`, `Detail` and `ErrNotApplicable` (§4.2), D-2 confirmation per origin bound to its context, objects and UIDs, audit log with Secrets redacted, UI state, coalesced snapshots (≤ 10 per second per view), the drawer model, streams, editor tabs, the custom kinds from a watch on the definitions, the drain journal, 41 commands; the access reviews of the connected user ([`accessreviews.go`](../../internal/engine/accessreviews.go), §4.2); third run: `Backend.Acquire`, `KindList` and `KindGet` for every read of a kind, the local store, the related-objects index, the port-forward manager ([`portforward.go`](../../internal/engine/portforward.go)), the Secrets drawer and its GUI-only reveal ([`secretdata.go`](../../internal/engine/secretdata.go); from the cache since the fourth run, no GET), Helm's actions and journal ([`helmops.go`](../../internal/engine/helmops.go)), log tabs with options and workload logs, attach, hidden columns and the group filter as UI state, the Workloads Overview page, CSV export, drawer-only commands; 65 commands; fourth run (2026-10-12): kubeconfig sources (added files, sync folders) and the approvals of their credential plugins ([`kubeconfigs.go`](../../internal/engine/kubeconfigs.go), §6), six `kubeconfig.*` commands | No jobs or cancellation UI (a drain runs in its dispatch call); a write command refuses to run if the user switched clusters while its confirmation was open; access reviews have no time-to-live (§5): they are kept until `cluster.refreshAccess` or the next connect |
| [`internal/mcp`](../../internal/mcp/) | Built: every command as a tool, Streamable HTTP on loopback, bearer token, Host and Origin checks; since 2026-10-12 `Server.Stop` and `Start` end and restart it with a new token (the status bar's MCP item) | Stateless HTTP (one session per request); the token lives in `<user config dir>/Cogpit/mcp-token` and changes with the port on every start; since 2026-10-13 the URL is written beside it (`mcp-endpoint`) and `cogpit-cli mcp` bridges stdio to it ([§7](#7-mcp-and-the-built-in-agent)) |
| [`internal/ui/fyneui`](../../internal/ui/fyneui/) and its [`yamlview`](../../internal/ui/fyneui/yamlview/) and [`termview`](../../internal/ui/fyneui/termview/) | Built: shell after Lens's layout, theme with status tokens, context picker, confirmation dialog, the YAML view, terminal and log widgets, a jank meter (`--perf`); row menus (⋮ and right-click) and the drawer's toolbar from the registry, schema-generated parameter dialogs, links and coloured parts in cells, Cogpit's own tooltips (Fyne 2.8 has none); third run: Cogpit's own title bar on macOS through a cgo bridge to the `NSWindow` ([`titlebar_darwin.go`](../../internal/ui/fyneui/titlebar_darwin.go); `COGPIT_SYSTEM_TITLEBAR=1` keeps the system one), the column menu, keyboard navigation, fit-to-content, CSV download, the Workloads Overview, item actions in the drawer (Forward…, Show), the port-forward dialog, the Helm banner, log tabs with their toolbar; fourth run (2026-10-12): the kubeconfig dialogs (paste, sources, the credential plugin's approval) and the picker's "Add Kubeconfigs" ([`kubeconfigs.go`](../../internal/ui/fyneui/kubeconfigs.go)); fifth run (2026-10-12): the Solarized Dark and Light colours as theme tokens, with the terminal's sixteen colours ([D-22](decisions.md#d-22-colour-theme), contrast held by `TestPaletteContrast`), and the C helm logo drawn by [`internal/ui/logo`](../../internal/ui/logo/logo.go) for the app icon and the header's mark ([D-24](decisions.md#d-24-logo)), copying values from the drawer (selectable lines, one copy icon moved to the value under the pointer, entries copied in YAML, a copied Secret value cleared from the clipboard after 60 seconds; [D-25](decisions.md#d-25-copying-values), [`selection.go`](../../internal/ui/fyneui/selection.go), [`copy.go`](../../internal/ui/fyneui/copy.go)), and the status bar's link to the owner, a GUI link outside the registry that gives way first when the bar is narrow ([D-23](decisions.md#d-23-a-link-to-the-owner-in-the-status-bar), `statusLayout` in [`bars.go`](../../internal/ui/fyneui/bars.go)); the fifth run's evening (the owner's smoke test): a pod's containers as squares, the drawer's Metrics section with each chart's own controls and a hover panel ([`chart_hover.go`](../../internal/ui/fyneui/chart_hover.go)), toasts as cards, a checkmark after a copy, sidebar groups closed at first, the status bar's text at 13 points with the link left of the MCP item that switches the external server, the context picker closed by a click outside, and a quit that no longer asks the terminated GLFW for the window ([`titlebar_darwin.go`](../../internal/ui/fyneui/titlebar_darwin.go)) | The YAML view is editable in Edit and Create resource tabs, whose text lives in the engine (`editor.get`, `editor.setText`); edits are applied as updates (PUT), not patches; the terminal widget is a patched copy of fyne-io/terminal ([`third_party/fyne-terminal`](../../third_party/fyne-terminal/PATCHES.md)) behind an escape filter, because the pinned upstream froze or crashed on container output |
| [`cmd/cogpit`](../../cmd/cogpit/) | Built: wires the packages, the drain journal (`<user config dir>/Cogpit/drains.json`) and the Helm journal (`helm.json`); the Helm chart repositories (`helm/repositories.json`, their indexes in `<user cache dir>/Cogpit/helm`, night run 2026-10-13); since 2026-10-13 the preferences (`preferences.json`) and cluster settings (`cluster-settings.json`, `cluster-icons/`), `--theme` an override of the theme preference for the run; the kubeconfig sources (`kubeconfigs.json`, `kubeconfigs/`, `exec-approvals.json`, fourth run) and their watcher; flags `--kubeconfig`, `--context`, `--demo`, `--mcp-addr`, `--no-mcp`, `--perf`, `--theme`; fifth run (2026-10-12): crash reports (`runtime/debug.SetCrashOutput` into `crash.log`, moved to `crashes/` at the next start and named in a toast, [`crash.go`](../../cmd/cogpit/crash.go), [D-11](decisions.md#d-11-crash-stance)), the external MCP server switched from the status bar and remembered (Fyne preference `mcp.enabled`, `--no-mcp` wins, [`mcpcontrol.go`](../../cmd/cogpit/mcpcontrol.go)) | client-go goroutine panics are logged instead of ending the app |
| [`internal/helm`](../../internal/helm/) (L2) | Built in the third run: release payloads decoded (bounded, fuzzed), the Releases profile as a virtual kind over the `owner=helm` Secrets with a bounded memo, rollback and uninstall through the Helm SDK on the connection's own client configuration; fourth run (2026-10-12): Lens's release drawer (Properties, values and notes as private items, resources by kind), the upgrade with the release's own chart, values read from the cached Secret (`ReleaseValues`), values scrubbed from errors and the SDK's log (`ScrubValues`, `RedactSDKLog`); night run (2026-10-13): public https chart repositories, their indexes downloaded bounded and parsed with the SDK, chart archives checked against the index's digest, and the install with a server-side dry run as its preview, and the upgrade to a chart version a repository lists ([`repo.go`](../../internal/helm/repo.go), [`install.go`](../../internal/helm/install.go), [`upgrade.go`](../../internal/helm/upgrade.go), §8) | No authenticated or OCI repositories (H-29); the upgrade has no preview; no action can be cancelled: rollback and uninstall take no context, and the upgrade's and install's contexts are not passed on (§8) |
| [`internal/metrics`](../../internal/metrics/) (L2) | Built 2026-10-12 (G12): the Metrics Server provider and its detection, the hub that polls while subscribed; Prometheus through the API server's service proxy, found among the cached Services by name, range queries per chart range; checked against a real Prometheus on kind. Fourth slice (the owner's evening requirements): both providers together, history from Prometheus once and the live values from the Metrics Server, the network and the filesystem only while a chart shows them, one provider alone only while the other fails or is missing, a status line per provider (`Hub.Providers`), a missing value as an empty cell, not 0 ([§9](#9-metrics-internalmetrics)); since 2026-10-13 the cluster's totals for the cluster Overview (`ClusterScope`: history as sums over the nodes, live sums of the node list) | Prometheus is found by name only (H-26) unless the user names it (`cluster.setPrometheus`, a dialog under the drawer's status line: Lens's cluster settings page is not built) |
| [`cmd/cogpit-cli`](../../cmd/cogpit-cli/) | Built 2026-10-13 (milestone 0.1, G2): a headless engine (no confirmer, no journals, no hotbar) over the same kubeconfig sources and credential plugin approvals as the app (`internal/app`), commands dispatched with origin CLI; `contexts` (`cluster.contexts`: table, JSON or YAML; context names and other untrusted text escaped for the terminal), `get <kind> [name]` (`cluster.connect`, then `kinds.list` and `resources.list` for a list or `object.get` for one object: the cache's first list awaited, the kind's columns as the view shows them, `resources_list`'s result or the manifest as JSON or YAML; a key that is not built in waits for the custom resource definitions' first list, `Backend.CustomKindsSettled`), `mcp` (the stdio bridge of [§7](#7-mcp-and-the-built-in-agent): forwarded to the running app, else a headless engine of its own), `version`, `--help` per command, exit codes 0 (done), 1 (failed), 2 (usage) | Without a configuration directory it refuses to connect, where the app merges the kubeconfig files as kubectl does and so runs their plugins unasked (Cogpit's default, not the owner's decision); `get` connects with `cluster.connect`, which opens the Pods view as in the app, so a `get` of another kind lists the pods once too (inferred cost, not measured); `-o json` of a list prints rows, not manifests (kubectl prints manifests) |
| [`internal/app`](../../internal/app/) (L4, new) | Built 2026-10-13: the wiring both binaries share: `UserDirs`, `Dirs.KubeconfigSources`, `Dirs.OpenAuditLog`; `cmd/cogpit` calls it from `openKubeconfigSources` and `openAuditLog` | A package the plan did not list |
| `internal/agent`, `internal/update` | Not built | – |

Layering is enforced by `depguard` in `make lint`, as planned in §3, with the demo package in L2.

Decided on 2026-10-11 for the next implementation run: the kind key of custom resources
([§4.5](#45-kind-keys)), and the contracts the parallel work shares: links in cells and the warning
column ([§4.4](#44-tables-details-editors-and-streams)), and the command fields `Order`,
`Escalate`, `Prefill` and `ErrNotApplicable` ([§4.2](#42-the-command-registry)).

Decided and built on 2026-10-11 for the third implementation run (its Phase 0, before the parallel
work): cache scopes with a label selector and a metadata-only flag, taken from the kind's profile,
and the local and virtual kinds ([§5 › Scopes](#scopes-local-and-virtual-kinds)); the
related-objects index ([§4.4](#44-tables-details-editors-and-streams)); how access reviews reach
`Command.Enabled` ([§4.2](#42-the-command-registry)); `helm.sh/helm/v4` v4.3.0 in `go.mod` with
a first `internal/helm` (release decoding, the SDK's action configuration on the connection's own
client configuration).

Decided on 2026-10-12 for the fourth implementation run (its Phase 0): `model.Related.Listed`
([§4.4](#44-tables-details-editors-and-streams), built); how the full Secrets scope and Helm's
`owner=helm` scope share one watch ([§5 › Scopes](#scopes-local-and-virtual-kinds); built
2026-10-12 as the covering scopes, with the Secrets view on a full watch); the Helm
upgrade seam, its journal entry and the Helm Upgrade dock tab ([§8](#8-helm-internalhelm); the
journal action `upgrade` and the tab's type `helmUpgrade` built, the rest by the run).

---

# Design plan (2026-10-11)

## 1. Goals and principles

**Goals**

- A native, open-source reimplementation of the current Lens Desktop for macOS, Linux and
  Windows ([parity](../parity/README.md); sources: [D-12](decisions.md#d-12-parity-sources)).
- Written in **Go** ([D-7](decisions.md#d-7-language-and-platforms)) on the libraries `kubectl`
  and `helm` are built from ([D-6](decisions.md#d-6-kubernetes-libraries)).
- **Fast for the user, light on the API server:** the UI reads only from a cache that watches keep
  current ([§5](#5-cache)).
- **Everything controllable over MCP**, the UI included
  ([D-2](decisions.md#d-2-agent-interface-scope)); a built-in agent uses the same tools
  ([D-10](decisions.md#d-10-built-in-agent)).
- **The UI toolkit can be replaced** without touching the engine: Fyne first, a Rust/egui
  process as the fallback ([D-1](decisions.md#d-1-ui-toolkit)).

**Principles**

1. **Engine-first, headless-first.** Every action works without the GUI: from tests, the CLI and
   agents.
2. **Everything is a command.** Each action has a stable id (`deployment.restart`, `ui.navigate`)
   with typed, JSON-serializable parameters. Menus, context menus, the palette, shortcuts, the CLI,
   MCP and the built-in agent dispatch the same commands.
3. **UI state is engine state.** The open view, namespace, filter, selection, drawer object and
   dock tabs live in the engine. The toolkit renders them; MCP reads and sets them.
4. **Data describes the UI.** Lists are table snapshots, details are section models, menus are the
   registry filtered by kind and state. A toolkit implements one table, one detail renderer, one
   editor, one terminal and one chart.
5. **Cache first.** Only the cache layer lists and watches. The UI and agents read from the cache
   and never call the API server for data the cache holds.
6. **Discover, don't assume.** Kinds, versions, permissions and metrics providers come from the
   cluster at runtime.
7. **Cluster-safe by construction.** Every mutating command declares its verbs and a danger level;
   confirmations are enforced in the engine for every origin (GUI, CLI, external agent, built-in
   agent).
8. **Crash-tolerant UI, crash-safe cluster operations**
   ([D-11](decisions.md#d-11-crash-stance)).
9. **Libraries over binaries.** client-go, the kubectl libraries and the Helm SDK run in-process.
   External programs only where the user expects them: credential plugins, the user's shell, and
   `kubectl` typed into a terminal ([D-3](decisions.md#d-3-helm-and-kubectl-binaries)).
10. **Clean-room against the current Lens Desktop**, without code from Lens, OpenLens or Freelens
    ([conventions](conventions.md#clean-room)).

### 1.1 Rules adopted from day one

| Property | Rule | Where |
|---|---|---|
| **API versions** | Kinds resolve through discovery per cluster. Built-in kinds get curated *kind profiles*; every other kind, CRDs included, gets a generic profile from discovery and `additionalPrinterColumns`. Objects are handled as unstructured content, so fields newer than the client libraries survive an edit | `model`, `cluster` |
| **API server load** | One watch per (cluster, kind, namespace scope), shared by every consumer; no polling of anything that can be watched; reads served from the cache | `cache` |
| **Scale** | Lists are virtualized; sort and filter run off the UI goroutine; the UI takes snapshots at a capped rate | `engine`, `ui` |
| **Permissions** | Access reviews decide what is shown and offered before the user tries | `cluster`, `engine` |
| **Multiple clusters** | One connection and one cache per kubeconfig context; nothing global assumes one active cluster | `engine` |
| **Credentials** | Credentials stay in the process that uses them; no local proxy that serves the user's identity to other processes | `cluster`, [security](../security/credentials.md) |
| **UI state** | Lives in the engine and is JSON-serializable | `engine` |
| **UI toolkit** | Only `internal/ui/...` imports it | layering (§3) |

The test: **adding a kind, an auth method, an agent or a second UI toolkit touches a leaf package,
not the foundation.**

---

## 2. Repository layout (planned)

Following the conventions of the team's other Go repositories (`cmd/`, `internal/`, a `Makefile`,
golangci-lint v2).

```text
cogpit/
├─ go.mod                        module github.com/guided-traffic/cogpit
├─ Makefile                      build, test, test-unit-race, lint, parity, scorecard, package
├─ .golangci.yml                 linters, plus depguard rules that enforce the layering (§3)
├─ cmd/
│  ├─ cogpit/                    desktop app (Fyne; cgo)
│  └─ cogpit-cli/                headless CLI and the MCP stdio bridge (pure Go)
├─ internal/
│  ├─ model/          L0  kind profiles, object refs, rows, status tokens; pure
│  ├─ kubeconfig/     L0  kubeconfig discovery, merge and validation from bytes; no I/O
│  ├─ cluster/        L1  connections: client-go config, auth, cached discovery, access reviews
│  ├─ cache/          L1  shared informers, reference counting, warm period, transforms, snapshots
│  ├─ terminal/       L1  PTY sessions: local shell with a per-terminal kubeconfig, pod exec and attach
│  ├─ helm/           L2  Helm SDK: releases from the cache, crash-safe actions
│  ├─ metrics/        L2  metrics provider detection, polling while visible, TTL cache
│  ├─ engine/         L3  Session, command registry, UI state, jobs, events, confirmation, audit log
│  ├─ mcp/            L4  MCP server over the registry (Streamable HTTP on loopback, stdio bridge)
│  ├─ agent/          L4  built-in agent: OpenAI-compatible client, tool loop over the registry
│  ├─ update/         L4  update check, download, verification, install
│  ├─ app/            L4  wiring both binaries share: directories, kubeconfig sources, audit log (added 2026-10-13)
│  ├─ ui/fyneui/      L4  the Fyne shell; the only package that imports Fyne
│  ├─ ui/logo/        L4  the C helm logo, rasterised for the app icon and the header (no Fyne)
│  └─ testkit/        –   fixtures, fake clients, kind and envtest helpers
├─ tools/                        generators for the parity checklist and the scorecard
├─ packaging/                    per-OS packaging scripts and manifests
└─ docs/
```

**Two binaries.** `cogpit` is the desktop app and needs cgo (Fyne renders through OpenGL).
`cogpit-cli` is pure Go (`CGO_ENABLED=0`), cross-compiles everywhere, and serves the CLI and the
MCP stdio bridge, so neither needs a display.

---

## 3. Dependency layering (enforced)

```text
 L5  cmd/*
 L4  ui/fyneui · mcp · agent · update · app
 L3  engine
 L2  helm · metrics
 L1  cluster · cache · terminal
 L0  model · kubeconfig
```

Rules, enforced by golangci-lint's `depguard` in `make lint` (milestone M0):

1. A package imports only packages in lower layers.
2. **Only `internal/ui/...` imports Fyne** (`fyne.io/fyne/v2`, `github.com/fyne-io/...`).
3. **`model` and `kubeconfig` do no I/O and import no client-go transport packages.** They take
   bytes and unstructured content and return values, which keeps them fuzzable.
4. **Only `cluster`, `cache`, `terminal`, `helm` and `metrics` talk to the API server.** The engine
   and everything above it reach clusters only through them.
5. A new package gets its layer and its `depguard` rule in the same change.

`internal/app` (L4, added 2026-10-13) is the wiring `cmd/cogpit` and `cmd/cogpit-cli` share:
Cogpit's directories (`app.UserDirs`), the kubeconfig sources a process offers
(`Dirs.KubeconfigSources`: `~/.kube`, the files named for the run, the sources the user added) and
the audit log. Its `depguard` rule (`l4-app`) keeps it off the other L4 packages and off client-go.

---

## 4. The UI seam (the most important interface)

Everything that crosses the seam is JSON-serializable: command parameters and results, table
snapshots, detail models, UI state and events. MCP and the CLI need that anyway, and it is what
lets the fallback UI of [D-1](decisions.md#d-1-ui-toolkit) run as a separate process over a pipe
without an engine rewrite. The signatures below are a sketch of intent, not an API.

### 4.1 `Session`: commands in, events out

```go
// package engine (sketch)
type Session interface {
	Dispatch(ctx context.Context, inv Invocation) (Result, error) // validates, checks permissions, confirms, runs
	Registry() *Registry                                         // menus, context menus, palette, shortcuts, MCP tools
	Table(view ViewID) *TableSnapshot                             // immutable; replaced, never mutated
	Detail(ref model.ObjectRef) (*DetailModel, error)
	UIState() UIState
	Subscribe(func(Event)) (cancel func())                        // ViewChanged, UIStateChanged, JobProgress, ConfirmRequested, Toast…
}

type Invocation struct {
	ID     string          // "ui.navigate"
	Params json.RawMessage // validated against the command's schema
	Origin Origin          // GUI, CLI, ExternalAgent, BuiltInAgent
}
```

### 4.2 The command registry

Each command registers:

- `id`, `label`, `menu path`, `default shortcut`, the kinds it applies to;
- a parameter type, from which the JSON schema is generated (`github.com/google/jsonschema-go`);
- `enabled(ctx) error`: a disabled item always says why;
- `verbs`: the API verbs and resources it uses, for access reviews and the
  [privilege footprint](../security/privilege-footprint.md);
- `danger`: `Read`, `UI`, `Mutating` or `Destructive`;
- `run`, returning a JSON-serializable result.

Fields added on 2026-10-11 for row menus, the drawer's toolbar and dialogs
([`registry.go`](../../internal/engine/registry.go)):

- `Order`: the command's place among its kinds' actions in the row menu (⋮) and the drawer,
  ascending; the kind's own actions in Lens's order below `OrderEdit` (900), then Edit and
  `OrderDelete` (1000). Zero keeps a command out of row menus and drawers. Lens's captured orders:
  Pod 100 Attach, 200 Shell, 300 Evict, 500 Logs; Node 100 Shell, 200 Cordon or Uncordon, 300
  Drain; Deployment 100 Scale, 200 Restart, 300 Logs; Cron Job 100 Trigger, 200 Suspend or Resume.
- `ErrNotApplicable`: an `enabled` error wrapping it hides the command for that object instead of
  greying it out, so Uncordon takes Cordon's place on a cordoned node and Resume takes Suspend's on
  a suspended cron job.
- `Escalate`: a mutating command whose invocation is destructive for some parameters (Scale to
  zero replicas) is confirmed and audited as destructive for those; its MCP tool carries
  `destructiveHint`.
- `Prefill`: parameters a schema-generated dialog starts with, read from the cache (the current
  replica count for Scale).
- `Detail`: a sentence the confirmation adds for parameters its objects don't show (the drain
  options that lose data).
- `DrawerOnly` (added in the third run, 2026-10-11): a command with an `Order` that the drawer's
  toolbar shows and the row menu leaves out: Copy (`object.copyName`), which Lens's drawer has and
  its captured row menus do not. Registering one without an `Order` is an error.

What a confirmation binds (2026-10-11, from the review): the context (`CheckConfirmedContext`), the
objects (`CheckConfirmedTargets`: a create whose namespace came from the filter is refused when the
filter moved while the dialog was open) and, for destructive commands, the objects' UIDs as the
cache held them when the user was asked (`ConfirmRequest.UIDs`, `ActUID`): Delete and Evict send
those as preconditions, never the UID of an object created again under the name; Restart (every
workload kind) and Scale to zero refuse when the cache holds the object under another UID by then
(review 2026-10-11, `TestRestartRefusesARecreatedObject`, `TestScaleToZeroRefusesARecreatedObject`).
Since 2026-10-12 (fourth run) their patches carry the UID too, which kubectl's do not: Restart's
pod-template patch and every Scale add `metadata.uid`, the confirmed UID, else the one the cache
holds when the command runs, and the API server refuses the patch on an object created again
under the name (the template patch because `metadata.uid` is immutable, 422; the scale
subresource by its UID precondition, 409; both verified on Kubernetes 1.36.1,
`TestKindPatchesBoundToTheUID`), which the command reports as `ErrTargetsChanged`. That closes the
gap between the cache check and the write, and covers a recreate the cache has not seen yet. A
mutating command's confirmation records the UIDs of its targets the cache holds already (no
watch started, no wait; a target it does not hold stays unbound and the command reads the cache),
so an agent's Scale the user accepted is bound like Scale to zero (`TestRestartAndScaleCarryTheUID`). A
command that writes a dock tab's text binds the text too, since `editor.setText` is a UI command
agents run freely: `object.apply` without `yaml` and `helm.release.upgrade` with `tab` name the
tab's text revision (`textRevision`, required) and are refused with `ErrTargetsChanged` when the
text changed since, as while the user's confirmation was open (review of the fourth run, R-2 and
R-10: `TestHelmUpgradeTabBoundToTheReviewedText`, `TestApplyFromTabBoundToTheReviewedText`). A
command can ask who dispatched it (`OriginOf`); `editor.get` uses that to redact Secrets for agents, and
`Spec.Refuse` refuses an origin before any confirmation (an agent's apply or create of a Secret, D-19), so the user is never
asked for what the command refuses anyway.

How menus use them ([`menu.go`](../../internal/engine/menu.go), 2026-10-11): `Registry.Actions(kind)`
is a kind's drawer toolbar and, without the drawer-only commands, its row menu: the commands whose
`Kinds` contain the kind or are empty, with an `Order`, sorted by `Order` then id. A command whose parameters have a `container` property
opens a submenu of the pod's containers and init containers (`TakesContainer`); one whose schema
requires more than `ref` and `container`, or that has a `Prefill` (then its optional fields too:
Scale's replicas, Drain's options), opens a dialog generated from the schema of those fields
(`DialogFields`: integer, number, string and boolean fields with their descriptions), started from
`Prefill`. `ObjectParams(ref, container)` fills the other required fields with zero values and
leaves the optional ones to `Prefill`, so that `Enabled`, `Applicable` and `Prefill` can be asked
before the dialog has the real ones. The UI
evaluates them off its main goroutine.

**Actions on a selection** (built 2026-10-13, G5, G7, G9, [`engine/selection.go`](../../internal/engine/selection.go),
[`commands/selection.go`](../../internal/engine/commands/selection.go)). A command that acts on
several objects takes them in a required `refs` array (`ParamRefs`, `TakesSelection`) and has an
`Order`; `Registry.SelectionActions(kinds)` lists those that apply to every kind of a selection
(no `Kinds`: every kind), and `Registry.Actions` leaves them out of one object's menus.
`SelectionParams(refs)` fills their parameters for `Enabled`. The access check of `Enabled` asks
for every object of `refs` as for a `ref` ([`access.go`](../../internal/engine/access.go)
`paramsRefs`): one denial greys the whole action. The only one is `object.deleteMany` (Delete of
the table's selection; the other actions stay on one object, Cogpit's default): at most
`MaxSelection` (500) objects; its `Targets` resolve every object, so the one confirmation names all
of them with their namespaces and the context, and records each UID (`ConfirmRequest.Message`
names the first `ConfirmNamed`, 10, and counts the rest; the GUI's dialog lists every object); it
checks the confirmed context and targets, then deletes the objects one after the other through
`Session.DeleteObject`, each bound to its own confirmed UID, stops only when the cluster changed or
the call ended, and reports what it did not delete in a `DeleteManyError` (deleted and failed
objects, each failure with its reason; nothing is retried). The deleted objects leave
`UIState.Selected`. The audit log's line carries `refs`, so it names every object
(`TestDeleteManyOneConfirmation`, `TestDeleteManyPartialFailure`,
`TestDeleteManyNeedsTheAgentsAcceptance`, `TestDeleteManyRefusals`).

**Access reviews in `Enabled`** (decided 2026-10-11, [`access.go`](../../internal/engine/access.go)).
The connector sets `Backend.Access`, an `engine.Access` that answers `Allowed(namespace, verb)` from
the access reviews cached per context and namespace, without blocking and without a request (an
answer it does not hold yet is `known == false`, and may start a review in the background). The
registry asks it in `Command.Enabled`, for every `Verb` the command declares, whenever
the command's parameters name an object in a `ref` field: a verb on `*` is resolved to the ref's
kind (its group and resource; the namespace dropped for cluster-scoped kinds), a namespaced verb is
asked in the ref's namespace. A known denial makes `Enabled` return an error wrapping
`ErrNotPermitted` ("not permitted: you may not patch deployments.apps in team-a"), so every menu
greys the command with that reason and `Dispatch` refuses it for every origin; menus only read
`Enabled`. The denial comes before the command's own reason, except `ErrNotApplicable`, which
keeps the command hidden (amended 2026-10-11: a cordoned node offers Uncordon, greyed, not Cordon
and Uncordon both). An unknown answer, a kind without a profile, a local kind, a verb on `*` for a
virtual kind (no API resource; Edit and Delete do not apply to it), and a command without a
`ref` pass: the API server decides. Nil `Access` allows everything.

As built (2026-10-11, [`accessreviews.go`](../../internal/engine/accessreviews.go),
[`cluster/access.go`](../../internal/cluster/access.go)). The cluster connector gives every real
cluster's backend an `Access` over the connection's reviews; the demo cluster has none. It keeps
the answers of one connection: a `SelfSubjectRulesReview` per namespace and a
`SelfSubjectAccessReview` per question, until `cluster.refreshAccess` drops them or the context is
connected again (the GUI's Cluster menu has it as Refresh Access Reviews since 2026-10-12,
disabled on the demo cluster); nothing runs on a timer. A missing answer starts its review in the background,
once (deduplicated), at most four at a time, on goroutines that recover from a panic and record a
failed review instead of leaving it pending; at most 512 namespaces' rules and 2,048 questions are
kept until a refresh, and past them an answer stays unknown, since a command asks in whatever
namespace its parameters name (review 2026-10-11, `TestAccessReviewsAreBounded`); arriving answers emit `EventKinds` (the sidebar reads
`kinds.list` again) and `EventDetail` (the drawer asks its commands' `Enabled` again), coalesced
to one pair per 200 ms. Which review answers what:

- A verb in a namespace: that namespace's rules. A rule naming it allows it; no rule denies it,
  unless the rules are incomplete (an authorizer that cannot list its rules, such as a cloud
  provider's IAM webhook, or an evaluation error), and then an access review asks.
- A verb on a cluster-scoped resource or in every namespace: the default namespace's rules deny it
  when complete and naming it nowhere, because every namespace's rules carry the cluster-wide
  bindings' rules. When they name it, a binding in that namespace may be why (Kubernetes' `view`
  role names namespaces), and an access review asks.
- Rules limited to named objects answer nothing (Allowed names no object); wildcard verbs, groups
  and resources and a failed rules review go to an access review.
- After an access review fails (a server that does not answer them, or forbids them), no more are
  sent until a refresh: unanswered questions stay unknown, so commands are offered and the API
  server decides. Measured: one failing rules review and at most the four access reviews in flight
  (`TestBudgetWithoutAccessReviews`).

Connecting prefetches the default namespace's rules, and a change of the namespace filter its
namespaces' rules. `kinds.list` reports `listable` per kind for the sidebar and the View menu
(the same filter since 2026-10-12), which hide a kind known not listable and show one not known
yet: a cluster-scoped kind is listable when the default
namespace's rules name it (no access review per kind; a RoleBinding naming a cluster-scoped kind
is told apart only for namespaces, below); a namespaced kind is listable when any namespace
reviewed so far names it, and not listable only when every namespace the user can see was
reviewed, completely, and none names it. The namespaces the user can see are the default namespace
and the namespace filter's; if an access review says the user may list namespaces, also the
namespaces the cache lists (the reviews hold one handle on them while connected, shared with the
namespace filter), at most 32: with more, such a kind stays shown. Measured on the fake API
server (`TestBudgetAccessReviews`): connecting costs one rules review; a user limited to their
namespace settles the sidebar with one rules review and one access review; a user who may list
namespaces with one rules review per namespace (five for four namespaces) and one access review;
browsing, menus and the sidebar asking again cost no request once answered.

**The command palette** (built 2026-10-13, G13, [`engine/palette.go`](../../internal/engine/palette.go),
[`fyneui/palette.go`](../../internal/ui/fyneui/palette.go)) lists what the window's menus offer in
the current UI state, decided by the registry's metadata, so a command appears in it without
palette code (`Registry.Palette`; default rule, not the owner's, 2026-10-13):

1. The palette's object (the drawer's, else the one selected object) brings its kind's actions as
   the drawer's toolbar lists them (`Registry.Actions`), without those that do not apply; one entry
   per container for a command that takes one when the pod has more than one.
2. Every other command needs a `MenuPath`: one without is a tool for agents (`ui.state`,
   `kinds.list`, `resources.list`, `editor.get`, ...). The palette's own commands are left out.
3. A read without an `Order` returns data to its caller: it is offered only where the window
   presents its result (the CSV download, the kubeconfig sources); `hotbar.list` and
   `workloads.overview` are not.
4. A command with a `ref` and no `Order` is left out: its object is picked elsewhere
   (`pod.portForward`'s port in the drawer, `ui.navigate`'s object by a search).
5. A kind's command without a `ref` (the create commands) comes with its kind's view.
6. The palette fills three parameters itself: `ui.openView` becomes Switch to *view* per view the
   navigator offers but the open one, `hotbar.switchBank` an entry per other bank (⇧F1–⇧F12),
   `hotbar.openCluster` one per tile of the shown bank (F1–F12).
7. A command with other parameters is offered when the window has a dialog of its own for them
   (`PaletteInput.Runs`, the GUI's `paletteRunners`: the picker for `cluster.connect`, a bank's
   name, the create dialogs, the search field for `ui.setFilter`); else only when none is
   required, and it runs with its schema's defaults. One with a required parameter and no dialog
   (`hotbar.moveCluster`, `object.apply`, `object.create`) is left out.

A command that cannot run now stays, greyed with its reason; running an entry dispatches its
command with origin GUI as its menu item would (an object's action through the row menu's
`runAction`), so confirmations, dialogs and the audit log apply unchanged. `FilterPalette` matches
every term of the query against the label and the category (label start, word start, substring,
category, then fuzzy from a word's start). The registry's shortcuts are bound from the registry:
⌘⇧P (`ui.palette.open`) and ⌘P (`cluster.connect`, the picker) on main-menu items, which Fyne asks
before the focused widget (on macOS the native menu's key equivalents), ⌘F (`ui.setFilter`) and
⌘⇧T (`terminal.open`) as window shortcuts; F1–F12 stay typed keys of the hotbar. Since 2026-10-13 the
tab bar's ⌘W, ⌘⌥→ and ⌘⌥←, ⌃- and ⌃⇧- are View menu items with the registry's shortcuts too, and
`ui.tab.select`'s range ⌘1–9 is bound key by key (⌘1–⌘8 the n-th tab, ⌘9 the last), as F1–F12 are
(`tabBar.menuItems`). ⌘Q is the
platform's (GLFW's macOS app menu), and nothing binds ⌘R.

Every command is also an MCP tool, generated from the registry (`ui.navigate` → `ui_navigate`).
Nothing is exposed to agents that the registry does not describe.

### 4.3 UI state

```go
type UIState struct {
	Context   string            // kubeconfig context of the active cluster tab
	View      string            // kind key or page: "pods", "services", "overview"
	Namespace []string          // empty: all namespaces
	Filter    string
	Selected  []model.ObjectRef
	Drawer    *model.ObjectRef  // object shown in the detail drawer
	Dock      []DockTab         // terminal, logs, editor, chat tabs and the active one
	Hotbar    *HotbarState      // the hotbar's banks and the shown one (2026-10-12)
	Palette   bool              // the command palette is open (2026-10-13)
	Overview  OverviewChoice    // the cluster Overview's chart range and node group (2026-10-13)
	Preferences     *Preferences               // Cogpit's own settings (2026-10-13)
	ClusterSettings map[string]ClusterSettings // display name, icon, terminal, per context (2026-10-13)
	Settings        *SettingsPage              // the open Preferences or Cluster settings page (2026-10-13)
	Tabs      []ViewTab         // the tab bar: a view of one cluster each, with its own state (2026-10-13)
	ActiveTab string            // the shown tab; View, Namespace, Filter, Selected, Drawer are its
	Welcome   bool              // the workspace shows the welcome page; derived (2026-10-13)
	ColumnOrder map[string][]string // each view's column ids as the user arranged them (2026-10-13)
	Notifications NotificationSummary // the toasts' log: total, unread, its worst level (2026-10-13)
}
```

The GUI changes state only through commands (`ui.openView`, `ui.select`, `ui.navigate`,
`ui.setFilter`), so a selection made by the user and one made by an agent are the same event.
`Selected` is a list (each tab keeps its own): since 2026-10-13 `ui.select` takes a `mode`
(`replace`, the default, as a click on a row; `add`, `toggle`, `remove`, as the row checkboxes,
⌘-click and ⇧-click) and `all` (every row the open view's snapshot shows, once the snapshot has
caught up with the namespace filter and the search: the header's checkbox), applied by
`ApplySelection` ([`selection.go`](../../internal/engine/selection.go); the selection keeps its
order, holds no object twice); only `replace` opens or closes the drawer (Cogpit's default,
inferred). The selection stays when the namespace filter or the search hides some of its rows
(inferred), and `ui.openView` clears it as before (`TestSelectModes`). The window's table
([`tablecheck.go`](../../internal/ui/fyneui/tablecheck.go)) has a checkbox column first (table
column 0; the view's columns follow from `firstDataColumn`, the ⋮ last) and the header's checkbox;
a checkbox, ⌘-click and ⇧-click dispatch `ui.select` with a mode and `noDrawer`, the header's
checkbox with `all`; it keeps the selection as a set for its cells and asks the registry's
`SelectionActions` for the menu of a selected row and the header's selection bar.
Added 2026-10-11 ([`uistate.go`](../../internal/engine/uistate.go)): `Groups`, the Definitions
view's API group filter (`ui.setGroups`), and `HiddenColumns`, the columns each view hides
(`ui.setColumns`; a view without an entry hides its kind's default-hidden columns). The view's
snapshot applies both: its `Columns` carry `Hidden` as the view hides them, at the kind's indexes.
Added 2026-10-13: `ColumnOrder`, each view's column ids in the order the user arranged them
(`ui.setColumnOrder`, dragging a header; kept for the session as the hidden columns are). The
snapshot carries it as `Order`, indexes into `Columns` (`engine.ColumnOrder`; nil for the kind's
order), so that `Columns` and the rows keep the kind's indexes; `DisplayOrder` gives the order
the window's table, its column list and `WriteCSV` follow
([`columnorder.go`](../../internal/ui/fyneui/columnorder.go)).

**Pages** (decided 2026-10-11, [`overview.go`](../../internal/engine/overview.go)): a view that is
not one kind's list has a page key with a slash, `<sidebar group>/<page>`, which no kind key has.
The first is `workloads/overview`, the Workloads Overview; the cluster's own Overview is
`cluster/overview` (built 2026-10-13: the plan's free key `overview` has no slash, so it would not be
a page; "cluster" because the item sits at the top of the cluster's tree). `ui.openView` opens a page like a kind. While it is open the engine runs the page's
worker in place of a view worker: it holds handles on the kinds the page reads (the seven workload
kinds and the events), released when another view opens, and builds the page's model (each kind's
count and status tokens, `workloads.overview`) and its table (the events, `Session.Table` under the
page key). The table's rows keep their kind's key (`events`), so `ui.select`, row menus and the
drawer work on them; `Session.ViewProfile` returns a page's table profile, so `ui.sort`,
`ui.setColumns` and `table.exportCsv` work on a page as on a kind's view.
The cluster Overview ([`clusteroverview.go`](../../internal/engine/clusteroverview.go), G12) runs a
worker of its own (`state.cluster`) with handles on nodes, pods, events, Deployments, Stateful Sets
and Daemon Sets (the kinds the user may list, as far as the access reviews know), and its table is
the Issues: rows of several kinds (`engine.MixedKinds`), each row's `Ref` the object the issue is
about, so `ui.select` there takes a row of any served kind and opens its drawer over the page, and
row menus act on each row's own kind. Its charts' range and node group are `UIState.Overview`
(`ui.setOverviewRange`, `ui.setOverviewNodes`), kept for the session whichever view is open.
Agents read it with `ui_state`: this is how the built-in agent knows which object the user means
by "this pod".

**The hotbar is engine state** (built 2026-10-12, G13, [`engine/hotbar.go`](../../internal/engine/hotbar.go),
[`commands/hotbar.go`](../../internal/engine/commands/hotbar.go), [`fyneui/hotbar.go`](../../internal/ui/fyneui/hotbar.go)).
`UIState.Hotbar` holds the banks (numbered from 1, as the switcher and ⇧F1–⇧F12 number them), each a
name and up to 12 tiles that store their context id only, and the shown bank; nil in a session
without a hotbar (`Options.Hotbar` unset, a headless session), which disables the commands with
`ErrNoHotbar` and hides the bar. The store (`engine.Hotbar`) keeps it in
`<user config dir>/Cogpit/hotbar.json` (0600 in a 0700 directory, written whole through
`writePrivateFile`); a file that does not load leaves an empty hotbar and is moved to
`hotbar.json.unreadable` before the first change writes, never overwritten. `Session.ChangeHotbar`
applies a pure operation of `HotbarState` (add, remove, move, create, rename, remove or switch a
bank; their `Check*` twins are the commands' `Enabled`), writes the file, and only then shows the
result through `UpdateUI`; a failed operation or write changes nothing. One lock orders the changes
from the write to the UI state, another guards the state alone, so a subscriber of the UI state may
read the hotbar while a change is shown. What a tile shows is derived, never stored:
`Session.HotbarView` (`hotbar.list`, and `ui.state`'s `hotbar`) adds each tile's label, the state of
a connection to it (Cogpit holds one), whether no kubeconfig source offers its context any more
(`missing`: the tile stays, inferred; counted only when the contexts list without an error) and its
credential plugin's approval. `hotbar.openCluster` connects through `Session.Connect`, as
`cluster.connect` does. The GUI reads `hotbar.list` again whenever `UIState.Hotbar`, the connection
or the contexts (`EventContexts`) change and changes the hotbar only through the commands. Fyne makes
no shortcut of an unmodified or shifted key, so F1–F12 and ⇧F1–⇧F12 reach the window's typed-key
handler (Shift read from the desktop driver) and the table's `TypedKey` (which tracks Shift), both
of which hand them to the hotbar; a focused text field or terminal keeps them.

**Preferences and cluster settings are engine state** (built 2026-10-13, G13,
[`engine/preferences.go`](../../internal/engine/preferences.go),
[`engine/clustersettings.go`](../../internal/engine/clustersettings.go),
[`commands/preferences.go`](../../internal/engine/commands/preferences.go),
[`commands/clustersettings.go`](../../internal/engine/commands/clustersettings.go)). `UIState.Preferences`
holds Cogpit's own settings (theme, time zone, the tables' row height, the editor's font size, line
numbers and tab size, the terminal's font size, the local shell and its arguments), nil in a
session that keeps none (`Options.Preferences` unset: `cogpit-cli`); `UIState.ClusterSettings` holds,
per context id, what the user set for that cluster (display name, icon colour or uploaded image,
the terminals' working directory and kubectl namespace), only contexts with settings; and
`UIState.Settings` the open settings page (`preferences` or `cluster` with its context, and the
section shown). The stores keep them as the hotbar's does: `<user config dir>/Cogpit/preferences.json`
and `cluster-settings.json`, read once, written whole through `writePrivateFile` (0600 in a 0700
directory), a file that does not load moved to `<file>.unreadable` before the first change, a field
out of bounds dropped on load without costing the others; uploaded icons are kept in
`cluster-icons/<sha256>.png|svg` and removed when no context uses them. `Session.ChangePreferences`
and `Session.ChangeClusterSettings` check the result, write, and only then show it through
`UpdateUI`. Commands: `preferences.get`, `preferences.set` (partial: the fields given),
`preferences.open` (⌘,), `cluster.settings.get`, `cluster.settings.set`, `cluster.settings.uploadIcon`,
`cluster.settings.resetIcon`, `cluster.settings.open` (⌘⇧,, the connected context unless one is named)
and `settings.close`; all UI or read commands that agents run freely (D-2), except two refused for
every origin but the GUI: the local terminal's shell and arguments in `preferences.set`
(`ErrShellGUIOnly`: they name a program Cogpit runs, and an agent can open a terminal) and the icon
upload (`ErrIconGUIOnly`: it reads a file on this computer). Where they take effect: the time zone
in the engine's clock for the drawer (`displayNow`: Created is written in that zone) and in the
GUI's charts; the shell, its arguments and the cluster's working directory and namespace when a
local terminal starts (`Session.LocalTerminalOptions`: the cluster's namespace, when set, replaces
the namespace filter's, Cogpit's default); the rest in the GUI. The Prometheus a cluster's metrics
come from stays in `metrics.json` (§9), unchanged.

The GUI ([`fyneui/preferences.go`](../../internal/ui/fyneui/preferences.go),
[`preferences_page.go`](../../internal/ui/fyneui/preferences_page.go),
[`clustersettings.go`](../../internal/ui/fyneui/clustersettings.go),
[`clusteridentity.go`](../../internal/ui/fyneui/clusteridentity.go)) applies `UIState.Preferences`
whenever it changes, whoever changed it: the theme and the row height by setting Cogpit's Fyne
theme again (`newThemeFor`; Fyne refreshes every widget; the window's `--theme` wins for the run),
the time zone in the charts' `loc`, the editor's settings on every open editor view, the
terminal's font size through a `container.ThemeOverride` around each terminal opened afterwards.
`UIState.Settings` puts the settings page in the window's centre in place of the navigator, the
views and the dock (the hotbar, the bars stay); its section list and close button dispatch
`preferences.open`, `cluster.settings.open` and `settings.close`, and a field's change dispatches
`preferences.set` or `cluster.settings.set` (a text field after a typing pause), the refusal shown
under the field. A cluster's identity (display name, initials, icon colour token, uploaded image) is
derived from `UIState.ClusterSettings` (`shell.identity`); the image's bytes come from the engine's
checked copy (`Session.ClusterIcon`, read off the main goroutine once per image). ⌘, and ⌘⇧, are
main-menu items (Preferences… moves into macOS's application menu); Settings… is in the picker's,
the hotbar tile's and the navigator root's menus.

**The tab bar is engine state** (built 2026-10-13, G5, [`engine/tabs.go`](../../internal/engine/tabs.go),
[`commands/tabs.go`](../../internal/engine/commands/tabs.go)). `UIState.Tabs` lists the tabs in their
order, each a `ViewTab`: an id (`tab-1`), the context it belongs to, the view, the title
"<view> - <context>" (the context without its kubeconfig file), whether it is the preview tab, its
own namespace filter, API groups, search, sort, selection and drawer, and its history (`Back`,
`Forward`: view keys, at most 50 each way). `ActiveTab` names the shown one, and the familiar fields
(`View`, `Namespaces`, `Filter`, `Sort`, `Selected`, `Drawer`) are that tab's live state, so every
command that reads or sets them works unchanged. One function keeps both in step:
`UIState.syncTabs`, which `UpdateUI` runs after every change, whichever command made it. While the
active tab's cluster is connected the tab's saved state follows the live one; a view changed by a
command that knows nothing of tabs (`ui.navigate`, a link) goes into the tab's history; a view opened
while no tab is active goes to the preview tab; a connect leaves no tab active until it succeeded,
then shows the cluster's active or most recently used tab with its own state, or else the connected
view in the preview tab. The tab commands (`ui.tab.open`, `.pin`, `.close`, `.select`, `.next`,
`.previous`, `.back`, `.forward`; all `DangerUI`) call `Session.OpenTab`, `PinTab`, `CloseTab`,
`SelectTab`, `CycleTab` and `TabHistory`; `ui.openView` is `OpenTab` in preview, so agents and every
existing caller open views as before. The titles' view part comes from a map the commands that
resolve a profile fill (`Session.NoteViewTitle`), so `UpdateUI` takes no lock but the map's; a view
never resolved is named by its key with a capital letter (`pods` → `Pods`, the default view).
Cogpit holds one connection (§6): the tabs of other clusters stay in the bar, and selecting one
connects to its cluster as `cluster.connect` does and then restores the tab. What is inferred there
(one preview tab in all, the neighbour shown on close, the history's contents) is listed in
`tabs.go` and in [ui-parity › Layout](../parity/ui-parity.md#layout). The window draws the bar
([`fyneui/tabbar.go`](../../internal/ui/fyneui/tabbar.go)) above the workspace from `UIState.Tabs`
and changes the tabs only through the commands: a click selects, the × closes, a double click on
the preview tab pins; the keys are View menu items (§4.2). Each tab draws its cluster's initials
badge as the navigator's root does, muted while that cluster is not connected; a cluster icon or
colour (cluster identity) is to replace it in both places.

**The welcome page is engine state** (built 2026-10-13, G5 and G13,
[`engine/welcome.go`](../../internal/engine/welcome.go),
[`commands/welcome.go`](../../internal/engine/commands/welcome.go)). `UIState.Welcome` says whether the
workspace shows the welcome page. It is derived, never set by a command: `UIState.syncWelcome`,
which `UpdateUI` runs after `syncTabs` (and `New` once), sets it while no cluster is connected and
none is connecting (at start, after a disconnect, a failed connect or a connection Cogpit closed),
and while a cluster is connected and no tab is active (`syncTabs` leaves no tab active only while
no view is open: the last tab closed). While a connect runs it is off, so that switching to a tab of
another cluster shows "Connecting to …" and not the page for a moment. `ui.welcome.open` (Help ›
Welcome, `DangerUI`) shows it over a connected cluster by leaving the active tab: `ActiveTab`
empty and the live view closed as closing the last tab closes it, the tab keeping the state
`syncTabs` saved, so selecting it (`ui.tab.select`) brings its view, filter, selection and drawer
back; opening a view ends the page as well. It also closes a settings page and the palette, and is
refused while a connect runs. Agents read the flag with `ui_state` (`welcome`). Cogpit's defaults,
inferred, not the owner's decision (Lens's welcome page is not captured). The window
([`fyneui/welcome.go`](../../internal/ui/fyneui/welcome.go)) draws the page over the workspace and
the dock, below the tab bar: the tab bar owns it as the workspace's state without an active tab
(`tabBar.workspace` wraps the body in `newShell`, `tabBar.sync` shows or hides it from
`UIState.Welcome`), and the page follows the contexts (`EventContexts`) and the hotbar through a
subscription of its own that only queues work for the main goroutine. A click on a cluster
dispatches `cluster.connect` as the picker does (a login program not allowed yet is shown first).
Documentation and About Cogpit are window actions outside the registry, as the status bar's link
(D-23): no agent opens a browser.

**Disconnect** (built 2026-10-13, G13, [`engine/disconnect.go`](../../internal/engine/disconnect.go),
`cluster.disconnect`, `DangerUI`): `Session.Disconnect` cancels a connect under way, waits for the
connect lock as a superseding connect does, detaches and closes the backend (the informers stop,
the cluster's port forwards stop, as on switching clusters; no request is sent) and leaves the UI
`Disconnected` without an error. `Context` keeps naming the cluster and its tabs stay, connecting
again when selected; the workspace shows the welcome page; the cluster's connect log gets an INFO
line. A connect started after the disconnect wins.

**Panel widths are GUI-only** (decided 2026-10-12, [`panelwidths.go`](../../internal/ui/fyneui/panelwidths.go)):
the width of the primary sidebar and of the detail drawer is a window-layout preference, not UI
state. What an agent drives is what the views show (view, filter, selection, which object the
drawer shows); how many points wide a panel is changes none of it, so no command sets the widths
and `ui_state` does not report them. The user drags a panel's edge (a 5 point grab area with a
one point line, the resize cursor and a highlight under the pointer); the width is kept in Fyne's
preferences (`cogpit.ui.sidebarWidth`, `cogpit.ui.drawerWidth`, in points) when the drag ends and
read again when the shell is built. Bounds: sidebar 160 to 480 (default 216), drawer 320 to 1000
(default 600), and the room beside a panel (the workspace beside the sidebar, the table beside the
drawer) keeps 480 points, whichever comes first; a stored value outside the bounds, zero or not a
number is clamped or replaced by the default, a drag is anchored to the pointer (the width is the one at the drag's start plus the pointer's total travel, clamped, so after overshooting a limit the edge stays until the pointer is back over it), and a window too narrow for the stored width shows
the panel narrower without overwriting the stored value.

**The navigator's search, keyboard row and open groups are GUI-only** (Cogpit's default,
2026-10-13, not the owner's decision; [`navigator.go`](../../internal/ui/fyneui/navigator.go)). They
change which views the navigator lists, not what a view shows, so, as with the panel widths, no
command sets them and `ui_state` does not report them; an agent lists the views with `kinds_list`.
What the navigator opens is engine state: a click, ↵ and the search's ↵ dispatch `ui.openView` (the
preview tab), a double click and ⇧↵ `ui.tab.open` with `fixed` (since 2026-10-13). The navigator and
the tab bar tell a double click themselves (two clicks within 500 ms, macOS's default): Fyne's own
double tap delays every single tap by 300 ms. The
tree is Cogpit's own rows (`navTree`), not Fyne's `widget.Tree`, whose rows keep their hover and
keyboard highlight private: an item's tooltip needs the pointer, the focus ring the keyboard's row.

**Why the content used to drive the drawer's width, and the rule now.** The drawer sat in a
`container.Split` whose offset is a fraction, and a Split clamps it by the minimum sizes of its
children: the drawer's minimum width was the widest thing in it (a status pill, an event's
header line, an unwrapped line), so a long value, such as a container's
`Waiting: CrashLoopBackOff (back-off …)` that the demo changes every few seconds, pushed the
divider and, through the window's minimum size, the table's columns and the toolbar with it. Now a
panel is a fixed number of points wide (`panelLayout`), its container's minimum width is a constant
whatever it holds, and the drawer builds its widgets to the width it has
(`drawer.contentWidth`, rebuilt by `widthChanged` when a layout gives it another): pills for
statuses, properties and chips wrap at any character (`wrapHard`) within a value's width, the
title bar's status pill is cut at 40% of the drawer, an event's header line cuts the reason and the
source (`eventHead`), links, text lines and table cells were already cut with an ellipsis, and the
event message is a wrapping label. The full text of a cut value stays available through the copy
icon and the selection ([D-25](decisions.md#d-25-copying-values)). A new drawer widget must not ask
for width beyond `contentWidth`; `TestDrawerWidthNeverFollowsItsContent` checks this on a model
with a 400-character value of every kind.

Toasts (`shell.toast`) are cards in the bottom-right corner: a surface of their own
(`colorToast`, `colorToastText`, 4.5:1 checked in `TestPaletteContrast`), a bar, border and icon
in the level's colour, a bold title, the text and a close button. They last 6 s (info), 9 s
(warning) or 14 s (error), and not while the pointer is over them.

**Every toast goes into the notification log** (2026-10-13, Cogpit's default, not the owner's
decision; [`engine/notifications.go`](../../internal/engine/notifications.go)), which the title
bar's bell lists ([`notifications.go`](../../internal/ui/fyneui/notifications.go)). The session
keeps the newest 100 in memory, numbered from 1, each with its level, text, time and whether the
user read it; a restart starts empty. `Session.emit` records an `EventToast` before any subscriber
sees it and sets the event's `Notification` id; the window records the toasts it raises itself
(a command's error, "Copied to the clipboard", the crash report notice at start) with
`RecordWindowToast`, an optional method the shell looks for, so a test fake keeps none. The log
sits under `Session.mu` with the UI state, and `UIState.Notifications` sums it up (total, unread,
the most severe unread level, the newest id): a change bumps the revision and emits `EventUIState`
directly, without `UpdateUI`, because toasts come from view workers and recover handlers that must
not wait for the views to follow a change. `notifications.list` returns it newest first,
`notifications.markRead` (named ids, or all) and `notifications.clear` change it; the window opens
the panel by reading the list and marking what it shows read, and a toast closed by its × marks its
own id. **What stays in the window:** a message the window raised itself, and an engine toast
emitted with `WindowOnly` (the warning `runHelmOp` shows with a failed Helm action's error, which a
chart's templates can fill from a Secret, [H-31](../security/privilege-footprint.md#h-31-an-agents-helm-values-can-copy-what-lookup-reads)),
read `model.Withheld` for every origin but the GUI. A new toast that quotes what the cluster
returned to an action whose error is withheld from agents must be emitted `WindowOnly` too. Whether
the panel is open is the window's, as the context picker's is.

### 4.4 Tables, details, editors and streams

- **Tables:** `TableSnapshot{Columns, Rows, Sort, Filter, Total, Revision}`; rows carry display
  text, sort keys and a status token that the toolkit maps to a theme colour.
- **Details:** section models: metadata, labels and annotations, conditions, related objects,
  events, containers, a metrics slot. As built (2026-10-11, [`detail.go`](../../internal/engine/detail.go),
  [`model/detail.go`](../../internal/model/detail.go)), in Lens's order: the metrics slot (not built,
  M7), one Properties section (the object's metadata without the UID, then the kind's own
  properties: a profile's first section titled "Properties" merges into it), the kind's other
  sections (a section may carry a `group` heading, "Containers", or a `table`, a node's
  Capacity), then the events. A virtual kind's drawer (2026-10-12, fourth run: Helm releases) is
  its profile's sections alone, without the generic metadata and without events, which never
  involve Cogpit's own objects (`DetailModel.NoEvents`; the drawer acquires no events watch for
  it), and a profile's `DetailTitle` names the kind in the title ("Release: web"). The GUI may draw
  a section of a kind itself (`drawerSections` in [`drawer.go`](../../internal/ui/fyneui/drawer.go):
  a release's values in an editor); agents read the section as items. A table cell may hold several lines (a role's resources and
  verbs). A table of cells (`model.CellTable`, decided 2026-10-11 in the third run: a node's Pods,
  a binding's subjects, an endpoint slice's endpoints) adds column `types` and `cells` beside the
  plain `rows`, so that the drawer draws links, status colours and the warning icon as the lists
  do; agents keep reading `rows`, and `DetailTable.Link` reads a cell's link. An item's `value` is
  always its plain text; `link` names the object
  it shows, `chips` are a list's entries, and `fold` folds a long list behind its count ("6
  Labels"). `object_get` returns the same model, except that an item marked `Private` (Helm's
  user-supplied values, which can hold credentials) reads `model.Withheld` for agents
  (`model.WithholdPrivate`, applied in `DetailWait` by the dispatching origin; added 2026-10-11,
  not part of the JSON).
- **Editors, logs, terminals and chats** are engine sessions shown as dock tabs, so agents can open,
  read and close them too.
- **Logs tabs** (third run, 2026-10-11, [`engine/logs.go`](../../internal/engine/logs.go)): a logs
  tab's options (previous container, timestamps, resource names, wrap, search, filter) are engine
  state on its dock tab (`DockTab.Logs`), changed with `logs.setOptions` by the GUI and agents
  alike. The GUI opens the stream again when its request changes (object, container, previous,
  timestamps) and shows the lines it holds the new way for the others
  ([`logtab.go`](../../internal/ui/fyneui/logtab.go)). The logs of a controlling resource
  (`workload.logs`) are one stream the engine merges
  ([`workloadlogs.go`](../../internal/engine/workloadlogs.go)): the resource's pods from the cache
  by owner reference (a Deployment's through its replica sets), each container's log started when
  the cache shows it running, so pods and restarted containers join while it follows; each line
  starts with `<pod>/<container>`; at most 40 containers at once. Since 2026-10-12 (fourth run)
  the set of started streams forgets the pods and container instances the cache no longer shows
  (a pod's UID is never reused and a restart count only grows, so they never come back), and a
  follow stream that the server ends while the cache still shows that container instance running
  is opened again: after 0.5 s, doubling up to 8 s, at most five times in a row without a new line
  (then a `[cogpit]` line says it stopped following), from `sinceTime`, the newest line's
  timestamp. The API carries `sinceTime` in whole seconds, so the resumed stream repeats that
  second, and Cogpit drops the lines before the newest timestamp and, at it, as many as it has.
  For that a follow stream always asks for timestamps; since 2026-10-13 every stream does (a
  pod's too) and keeps them, and the GUI hides them unless the tab shows them, so that the tab
  can jump to a time and load older lines by them, and the Timestamps toggle no longer opens the
  stream again. Not covered: a resumption names the container, not its instance, so if the cache
  shows a restart only after the backoff, the new instance's first lines come twice (inferred,
  not observed).
- **A logs tab's source** (night run, 2026-10-13, [`engine/logsources.go`](../../internal/engine/logsources.go)):
  which pod and container a tab shows is engine state too (`LogOptions.Pod`, `DockTab.Container`),
  changed with `logs.setSource` and offered by `logs.sources` from the cache. A pod's tab opens
  its stream again for another container or pod. A controlling resource's merged stream follows
  its tab's selection itself, by subscribing to the engine's UI state events: the streams the
  selection leaves out are cancelled (each container's stream has its own context) and keep their
  cursor; a rescan starts the selected ones that do not stream, and one selected again resumes
  from its cursor with `sinceTime` and at most 1,000 lines (`tailLines` as the bound), a `[cogpit]`
  line saying so when the bound cut lines off (its first line is newer than the cursor). The
  streams that stay are never opened again. The GUI narrows the lines it already holds by their
  `<pod>/<container>` prefix. With Previous, whose stream ends, a selection change opens it again.
  Init containers count only when named, sidecars (`restartPolicy: Always`) always.
- **Older lines of a logs tab** (night run, 2026-10-13, [`engine/logsolder.go`](../../internal/engine/logsolder.go),
  [`logtab_older.go`](../../internal/ui/fyneui/logtab_older.go)): the lines live in the GUI's
  view (termview, at most 10,000), not in the engine, so a request for older lines is engine
  state that the window carries out: `logs.loadOlder` counts it on the tab (`LogOptions.LoadOlder`),
  and the tab, following its state, sends what it holds of each container (the lines and the
  oldest timestamp, by prefix) and its room to `Session.OlderLogs`, which the GUI reaches through
  an interface beside `OpenLogs`. The API has no "before" cursor: the engine asks each container
  the tab shows and does not hold whole for its tail again with 1,000 more lines than it holds
  (shared, at least 100 each; at most 8 requests at once), keeps the lines older than the oldest
  held and those at that timestamp (at most 32 MiB of them, the view's own byte bound, so that
  long lines cannot make a request grow beyond it), merges them in time order and keeps the newest
  that fit the room. A container the server sent fewer lines of than asked is held whole and not asked again.
  The view drops the lines at its oldest timestamp it holds (timestamp and text), puts the rest
  before its lines with their rows, keeps the rows in view in place and stops following. Lines of
  a container it held nothing of go in as well, and the stream's own copy of them is dropped when
  it arrives (a fence per container, until its lines are newer). Agents never receive the lines:
  the command returns the tab.
- **A jump to a time** (night run, 2026-10-13, [`engine/logsjump.go`](../../internal/engine/logsjump.go),
  [`logtab_jump.go`](../../internal/ui/fyneui/logtab_jump.go)): `logs.jumpTo` reads the time in
  the engine (`ParseLogTime`: dates with times, times of day, durations ago, RFC 3339; zone-less
  times in UTC, the zone of the API server's timestamps the tab shows) and keeps it on the tab
  with a sequence number (`LogOptions.Jump`), so that the same time jumps again. The window
  follows it: when it holds lines at or before that time of every container it shows (or holds a
  container whole), it searches them; otherwise it first asks `OlderLogs` with `Since`, which asks
  each such container for its lines since then (`sinceTime` in whole seconds, the room as the
  `tailLines` bound) and is complete when the server sent fewer than that. The view then selects
  the line with the earliest timestamp at or after the time among the rows it shows (a merged
  tab's lines are in arrival order, not time order) and stops following. A view's line numbers go
  below zero once lines are prepended; nothing may use -1 as "none" (`trimRows` did, fixed with
  `TestLogsEvictPrependedLines`).
- **Links and warnings in cells** (2026-10-11, [`row.go`](../../internal/model/row.go)): a cell
  that names another object carries a `Link` (an object reference with a kind key; an empty context
  means the row's), which the UI renders as a link. A Namespace link in a list narrows the view's
  namespace filter (`ui.setNamespace`, owner 2026-10-11); other links navigate (`ui.navigate`).
  Lens's warning-icon column right after Name is a column of type `warning`: a cell with a status
  shows the icon in that status's colour with its text as the tooltip; an empty cell shows nothing.
  A cell with `parts` is drawn in coloured runs ("SchedulingDisabled" plain, "Ready" green on a
  cordoned node); its text stays the runs joined by spaces, for sorting and search.
- **Item actions** (decided 2026-10-11, [`model/detail.go`](../../internal/model/detail.go)
  `ItemAction`, [`itemactions.go`](../../internal/ui/fyneui/itemactions.go)): a drawer item can
  carry buttons beside its value, each a command with the parameters that name what the item
  shows ("Forward…" beside a container port: the pod, the container, the port). Agents see them in
  `object_get` and dispatch the same command. The GUI runs one as the row menu runs a command
  (a schema-generated dialog, prefilled with the params, when the command asks for more), unless a
  feature registers a dialog of its own for the command in `itemActionDialogs` from its own file;
  the row menu and the drawer's toolbar open that dialog too (the port forward's Edit). An action
  marked `link` (added 2026-10-11 by the port-forwarding run) is the value itself: the drawer draws
  the value as a link that runs it (a container port forwards at once and opens the browser),
  the other actions as buttons right after the value.
- **Related objects** (decided 2026-10-11, [`model/related.go`](../../internal/model/related.go),
  [`engine/related.go`](../../internal/engine/related.go)): a cell or drawer section that shows
  objects of another kind (a PVC's Pods column, a node drawer's Pods table) reads them through an
  index over that kind's cached objects, never through its own list. Indexes are declared once in
  [`kinds/related.go`](../../internal/model/kinds/related.go) as a `model.RelatedIndex` (the
  indexed kind, a name, and the keys an object is found under: `PodsByNode`, `PodsByClaim`); a
  profile names the ones it reads in `Needs` and reads them in `CellsWith` and `DetailWith`
  (which replace `Cells` and `Detail`) through `rel.Objects(index, key)`. While a view or drawer of
  the kind is open, the engine holds a cache handle on each index's kind (shared with every other
  consumer), rebuilds the view's rows when those objects change (the row cache keys on the related
  objects' generation as well as the resourceVersion; the generation also changes when the
  indexed kinds finish their initial list, which comes with no event of its own, so rows built
  while they were listing are rebuilt, 2026-10-12), and builds each index once per
  view or drawer (2026-10-12, below). A lookup on a kind that is still listing or not served returns
  nothing. Rows built outside a view (`resources.list`) use `Session.BuildRows`. A rule that warns
  about an object that does not exist (a binding's cluster role) must tell that from a kind still
  listing: it asks `rel.Listed(index)` (2026-10-12), true once the index is declared and its kind
  served and listed, and warns only then. The cluster-role rule does since 2026-10-12;
  `ClusterRolesByName` files a cluster role under its name only.
- **Live indexes** (2026-10-12, fourth run, [`related.go`](../../internal/engine/related.go)): an
  index is built from the cache on its first lookup and then kept current by its kind's events,
  each of which moves one object between keys (its old keys are remembered per object), so a
  lookup copies one key's objects and never rebuilds the index. The build runs outside the set's
  lock; events that arrive meanwhile are applied after it, in order. A virtual kind's index
  (none yet) is built again after each change, since its events are about the source objects.
  The drawer's set is shared with the fetches of the drawer's model (`Detail`, `object.get` while
  the drawer shows the object): they read the drawer's indexes instead of building their own,
  and their lookups tell the drawer which keys to wake for: a node's drawer wakes for its own
  pods (entering, changing in or leaving the node), not for every pod. Views still wake for
  every change of an indexed kind, since their rows read any key. Measured on the demo cluster
  (10,000 pods on 50 nodes, 1,000 events per second, a node with 261 pods; Apple M1 Ultra with
  other agents' builds running, `TestNodeDrawerUnderDemoChurn`): a fetch of the node's drawer took
  13–33 ms (p50, three runs) and the fetches 14–43 % of a core before, 2.6 ms and 2.3–2.5 % after;
  the drawer still fetches about nine times a second, because the node's own pods change that
  often under the demo's churn. The process-wide CPU difference with and without the drawer
  (6–14 % before, −6 to 11 % after) is within the churn's own variance and shows no trend.
  `ui.navigate` checks the view's filter against the row as the view shows it, related cells
  included (a claim found by its pod's name keeps the filter).

### 4.5 Kind keys

A kind key names a kind in views, object references, `ui.*` commands and MCP tools (decided
2026-10-11):

- **Built-in kinds keep the bare plural resource name:** `pods`, `deployments`, `ingresses`,
  `customresourcedefinitions`. These are the kinds the Kubernetes API server serves itself, which
  Cogpit ships a curated profile for in [`model/kinds`](../../internal/model/kinds/).
- **Kinds from custom resource definitions are `<plural>.<group>`**, as kubectl spells a resource
  with its group: `gateways.gateway.networking.k8s.io`, `certificates.cert-manager.io`. Bare plurals
  are not unique across groups (Gateway API's `gateways` and Istio's `gateways.networking.istio.io`
  can both exist), and a group is part of what a CRD is.
- The two never collide: a CRD's group always contains a dot, and a built-in key never does.
- A curated profile for a CRD kind (Gateway API, Flux CD, a Vertical Pod Autoscaler) keeps the
  `<plural>.<group>` key, so a key never changes when Cogpit learns to show a kind better. Built
  2026-10-13 for the Vertical Pod Autoscalers: `kinds.Curated()` lists such profiles, and the custom
  kinds watch takes `kinds.ProfileFor(kind)`, the curated profile while the definition's scope and
  Kind match and it serves the profile's version or one of its `FallbackVersions`, else the generic
  one ([`verticalpodautoscalers.go`](../../internal/model/kinds/verticalpodautoscalers.go)). They are
  not built-in kinds: they come and go with their definitions.
- The key carries no version: the version is the profile's preferred one (for CRDs, the highest
  served version by Kubernetes version priority, as discovery orders them). A built-in kind that
  moved through API versions of the same shape lists the older ones as `FallbackVersions`
  (Validating Admission Policies: v1, then v1beta1, v1alpha1); the connector keeps its profile,
  key unchanged, at the first version the cluster serves (2026-10-11,
  [`connector.go`](../../internal/engine/connector.go) `servedProfiles`).
- **Virtual and local kinds take Cogpit's own bare plural** (2026-10-11): Helm's releases are
  `helmreleases` ([`helm.ReleaseKind`](../../internal/helm/releases.go)), their objects apiVersion
  `cogpit.io/v1`, kind `HelmRelease`. Without a dot they never collide with a CRD kind (Flux's
  `helmreleases.helm.toolkit.fluxcd.io`); they must not reuse a built-in resource's plural. Their
  profile still names the resource they are made from (`secrets` for releases), so code that maps
  a resource to a kind key (`KindKeyFor`) skips them.

**Litmus test:** `cogpit-cli` and MCP can do everything the GUI does, except draw it.

---

## 5. Cache

The UI never waits for the API server. Cluster state flows one way: API server → watches → cache
→ snapshots → UI and agents.

```text
 API server ──list + watch──► cache (one informer per cluster/kind/namespace scope)
                                 │  transform: drop managedFields
                                 ▼
                         engine snapshots (coalesced, ≤ 10 per second per view)
                                 ▼
                    UI views · detail drawer · MCP tools · built-in agent
```

**Watches, shared and reference-counted.** `cache` keeps one client-go shared informer per
(cluster, kind, namespace scope), from `dynamicinformer` for full objects or `metadatainformer` when
a view needs only metadata. Every consumer (a view, an open drawer, an agent query) acquires a
handle and releases it. The informer starts with the first handle and stays **warm** for a grace
period after the last one is released (default five minutes, a preference), so switching back to a
recent view costs no list request. After the grace period it stops, which frees the API server's
watch and Cogpit's memory.

**Light on the API server.**

| Mechanism | Effect | client-go 0.37.1 |
|---|---|---|
| No resync period | No periodic relists; only watch events after the initial list | informer resync `0` |
| Watch bookmarks | Reconnecting resumes from a recent resource version instead of relisting | reflector sets `AllowWatchBookmarks: true` |
| Streaming initial lists | The initial list arrives as a stream instead of one large response | `WatchListClient`, beta and on by default since 1.35 |
| Relist only on `410 Gone` | A watch that fell too far behind recovers once | reflector behaviour |
| Namespace-scoped informers | Users limited to some namespaces, or a namespace filter, watch only those | per-namespace factories |
| Metadata-only informers | Views that show only metadata transfer and store only metadata | `metadatainformer` |
| Discovery cached on disk with a TTL, refreshed when CRDs change | Connecting again costs no discovery round trips | `discovery/cached/disk` |
| Access reviews cached per context and namespace (built without a TTL: kept until refreshed on demand or on reconnect, §4.2) | Menus and sidebars don't ask the API server on every render | engine cache |
| Client-side rate limit per cluster | Bounds lists, gets and writes; established watches are not affected | `rest.Config` QPS and Burst (defaults 5 and 10; a preference) |

**Current, but cheap.** Watch events land in the cache as they arrive; the engine rebuilds the
snapshots of the views that use a changed kind, coalesced to at most ten per second per view, so a
burst of a thousand events costs the UI at most ten redraws. Data that cannot be watched (metrics,
Helm chart repository indexes) is polled only while something visible needs it, at an interval per
source, and cached with a time-to-live.

**Small in memory.** A transform drops `metadata.managedFields` before an object enters the cache
(the editor's "managed fields" toggle fetches them on demand). Memory per cluster and object counts
per kind are visible in a diagnostics view. From core Secrets, in every scope, the same transform
also drops the `kubectl.kubernetes.io/last-applied-configuration` annotation (2026-10-11), which
`kubectl apply` fills with the whole manifest, values included (`TestSecretsLoseTheLastAppliedAnnotation`).
Re-checked with the full Secrets watch (2026-10-12) and kept: the annotation would hold every
applied Secret's values a second time, in plain text, and it would reach the drawer's Annotations,
which agents read through `object_get` (the YAML redacts it, the Properties would not). The drawer
builds a Secret's Properties from `model.Redact` as well, so an annotation that arrives another way
reads `<redacted>` (`TestSecretDrawerRedactsTheLastAppliedAnnotation`). The annotation still
crosses the wire with the object and is decoded before the transform drops it.

**Measurable.** A counting HTTP round-tripper per cluster records requests by verb and resource;
the diagnostics view shows them, and tests assert the budgets P9–P11 in
[`testing.md`](testing.md#performance-budgets-proposed-not-measured).

**Reads from the cache.** The detail drawer, the YAML editor's initial content, `ui_state`,
`resources_find` and every other read are served from the cache. A direct `GET` happens only for
data the cache does not hold (subresources, a kind nobody watches), and its result does not bypass
the cache's view of the world.

### Scopes, local and virtual kinds

Decided and built on 2026-10-11 ([`cache.Scope`](../../internal/cache/cache.go),
[`engine/kindscope.go`](../../internal/engine/kindscope.go)). A scope is a resource, a namespace
(empty: all), a **label selector** and a **metadata-only** flag; each distinct scope has its own
handles and is one informer and one watch unless a cover serves it (below), and `Acquire`
normalises the selector, so `a=1,b=2` and `b=2,a=1` share one. Every
`Source` honours the whole scope: the real cluster's source (`DynamicSource` over the dynamic and
the metadata client) sends the selector to the server and asks for `PartialObjectMetadata` for a
metadata-only scope, so object content never crosses the wire; in-memory sources apply
`cache.FilterScope` or `cache.FilterListerWatcher`. A metadata-only object reaches a kind profile
as an `unstructured.Unstructured` with apiVersion `meta.k8s.io/v1`, kind `PartialObjectMetadata`
and the object's `metadata`, nothing else (`cache.MetadataObject`, `cache.IsMetadataObject`).

A kind's profile decides its scope (`MetadataOnly`, `LabelSelector`), and every engine read of a
kind goes through `Backend.Acquire(profile)` and `KindList`/`KindGet`, never through a scope built
by hand, so the view, the drawer, `Lookup` and agents' reads of a kind share one informer.
Measured at the HTTP level (revised 2026-10-12 for the full Secrets watch): the Secrets view is one
initial list or streaming watch of every Secret in full, and opening a Secret's drawer or revealing
a value costs no request (`TestBudgetSecretsViewAndDrawer`); the Releases view alone is one
`secrets with owner=helm` request and nothing else (`TestBudgetHelmReleasesView`); both together are
one watch, the full one (`TestBudgetSecretsAndReleasesShareOneWatch`, below); a metadata-only scope
asks for `PartialObjectMetadata` and gets no value (`TestBudgetSecretScopes`; on kind,
`TestKindSecretScopes`, `TestKindSecretsView`). `resources.find` without a kind searches a virtual
kind, and a kind whose scope holds Secret values (`engine.HoldsSecretValues`: the Secrets and Helm's
releases), only when the cache holds it already (`Backend.Cached`, in the profile's scope or its
cover), so an agent's search by name never starts a watch of every Secret or release Secret
(review 2026-10-11, extended 2026-10-12, `TestBudgetFindWithoutKindStartsNoReleaseWatch`).
The Secrets view's drawer ([`secretdata.go`](../../internal/engine/secretdata.go); rebuilt
2026-10-12, before it made one `get secrets` per open) reads the Secret it shows from the cache like
any object and keeps nothing of its own: `BuildDetailWith` adds a Data section that lists each key
(sorted, at most 100) with its value masked and its size, a "Show" action beside a text value and a
binary value by its size only; it decodes each value to measure it and zeroes the bytes at once
(best effort: the cached object's base64 text and a revealed value's string are Go strings, which
cannot be overwritten). `secret.reveal` reads one value of the Secret the drawer shows from the
cache, cut to 4 KiB and 40 lines, for the GUI origin only (`TestSecretDrawerReadsTheCacheAndMasks`,
`TestRevealIsGUIOnly`). Its action names the Secret's `resourceVersion` the drawer listed the key
at: a Secret changed or created again since is refused, and the drawer masks a value revealed at an
older version and forgets it (`TestRevealedValueDroppedWhenSecretChanges`; R-6 of the fourth run's
review). Agents read the drawer, the YAML (`model.Redact`), the list's cells, the
CSV export and every other result without a value (`TestAgentsReadSecretsRedacted`).

**Covering scopes** (decided 2026-10-12 with [D-13](decisions.md#d-13-cache)'s revision, built
2026-10-12 by the fourth run, [`cache/cover.go`](../../internal/cache/cover.go)). A label-selected
scope is a subset of the unselected scope of the same resource, namespace and metadata flag, its
cover. While the cover runs (a handle holds it, or it is warm), the cache serves the selected scope
from it: the selected scope's handles list and get the covering informer's objects that match the
selector and see its events filtered by it (an object whose labels leave the selection arrives as a
delete, with the state that left it), and no second watch starts; `Cache.Stats` names the cover in
`coveredBy`, and `Cache.Running` reports a covered scope as running. A covered scope never keeps
its cover alive: when the cover's warm period ends while covered handles are open, the selected
scope starts its own informer, and the cover's informer keeps running until that one has caught up,
at most `Options.HandoverTimeout` (one minute), then stops: an own informer that cannot list does
not keep every Secret in memory. An own informer that has not finished its first list by then is
moved onto anyway, and the scope keeps what it showed (its own objects only), `HasSynced` false,
telling its subscribers nothing until that list arrives; then what the list does not hold is
reported deleted (`view.settle`). Before 2026-10-12 the subscribers were told every object was
deleted and got it added again in the same version (R-7 of the fourth run's review,
`TestCoverStopsAfterTheHandoverTimeout`). A covered scope that is only warm ends
with its cover instead of starting a watch to stay warm. A selected scope with its own informer moves
onto a cover that starts later, once the cover has caught up, and its own informer stops.

As built: a selected scope's handles read a view, the objects its subscribers have been told
about, which an event handler on the serving informer keeps (client-go's `AddEventHandler`: a
consistent snapshot of the store, then every change). A handover registers a handler on the new
informer that fills a staging copy, and moves the view onto it once that handler has synced and has
delivered a resourceVersion at least as new as every object the view holds; the subscribers then
get the difference as events, and the old informer's events are ignored. An object the new
informer's store holds but its handler has not delivered yet is kept, not reported deleted: a
relist notifies in key order, not resourceVersion order, so the newest version delivered says
nothing about the others (R-8 of the fourth run's review,
`TestExactHandoverDuringRelistKeepsUndeliveredObject`). With comparable
resourceVersions (KEP-5504: well-formed integers, which every Kubernetes API server's built-in
resources have; `k8s.io/apimachinery/pkg/util/resourceversion`) the events stay exact across a
handover: never a version older than or equal to one already sent, never an object sent as deleted
brought back in an older version (the view remembers the last 1,024 removals). Proven by
`TestCoverHandoversUnderChurn` (15 handovers in each direction under churn, 150 runs with the race
detector: no event twice, none missing, List equal to the events and to the server),
`TestViewApplyRules`, `TestViewHandoverRules`, `TestConcurrentCoversAndCoveredScopes`,
`TestCoverStopsAfterTheHandoverTimeout`. Where resourceVersions are not comparable, a version that
differs counts as newer (inferred to be enough; no such resource is covered today). Helm's releases
(`owner=helm`) are the case it is built for: with the Secrets view open (a full watch since the
revision of D-13) they are derived from the full Secrets watch, and alone they cost the
label-selected watch only, so that the Releases view alone never holds every Secret
(`TestBudgetSecretsAndReleasesShareOneWatch`: Secrets view then Releases view, no `owner=helm`
request; after the Secrets scope's warm period, one `owner=helm` watch opens and the full one
closes; opening the Secrets view again moves the releases back and closes theirs). Helm's profile and
`Derive` did not change.

- **Local kinds** (`KindProfile.Local`, group `cogpit.io`): objects Cogpit makes itself, the port
  forwards. They live in the backend's local store, a `cache.Memory` with a cache over it, made on
  first use and closed with the backend; writers call `Backend.PutLocal` and `DeleteLocal`; views,
  the drawer, row menus and agents read them like cluster objects, and no API server is asked.
  Every real cluster's backend offers the profiles in `kinds.Local()`.
- **Virtual kinds** (`KindProfile.Derive`): a kind whose objects are made from the cached objects
  of its scope, the Helm releases (one per release, from the release Secrets of each revision).
  `Derive` runs on every snapshot and lookup and memoizes its own expensive work; the derived
  objects carry a UID and resourceVersion that change with them, which the row cache and the
  confirmation's UID binding use.
- **A virtual kind made of several kinds** (built 2026-10-13): the Applications, the cached
  Deployments, Stateful Sets, Daemon Sets, Cron Jobs and Jobs grouped by
  `app.kubernetes.io/instance` ([`kinds.GroupApplications`](../../internal/model/kinds/applications.go)).
  `Derive` sees one scope's objects only, so the kind gets a cache of its own, as the local kinds
  do: `Backend.Acquire` routes its profile to an informer over a source
  ([`engine/applications.go`](../../internal/engine/applications.go)) that serves the grouped
  objects from a `cache.Memory`. The source holds handles on the workload kinds on the cluster's
  cache from its informer's first list until its last watch stops (the view closed and the warm
  period over) plus two seconds, regroups after their events (coalesced to 100 ms, an object
  set again only when its content changed), and a list waits, bounded, until every workload kind
  listed or failed; `Cached` and `Listable` follow it (listable when a workload kind is). The
  profile's `Derive` passes the objects through, which marks the kind virtual: no Edit or Delete,
  a flat drawer without events. Measured: one watch per workload kind, shared with their views,
  and no other request (`TestBudgetApplicationsView`); the handles go back after the view closes
  (`TestApplicationsReleaseTheWorkloads`).
- `object.edit` and `object.delete` do not apply to local and virtual kinds
  (`KindProfile.APIObject`); such kinds bring their own commands. `kinds.list` marks them `local`
  and `virtual` (2026-10-12), and the GUI shows no "+" (Create resource) on their views.

**Custom resource kinds** (2026-10-11, [`customkinds.go`](../../internal/engine/customkinds.go)).
A real cluster's backend holds one cache handle on `customresourcedefinitions` while it is
connected (`Backend.CustomResources`; the demo cluster has none). Its watch events, coalesced,
rebuild the generic profiles of the established definitions
([`kinds.ParseCRD`](../../internal/model/kinds/customresources.go)), or a curated profile where
Cogpit has one ([§4.5](#45-kind-keys), 2026-10-13), published as one immutable set
behind an atomic pointer: `Backend.Profile` and `Backend.AllProfiles` read it without a lock, an
update that changes nothing a view shows keeps the profile, and a change emits `EventKinds`, upon
which the UI re-reads `kinds.list`. The open view and drawer follow their profile: a new served
version moves them, a deleted definition leaves a view whose snapshot says the cluster no longer
serves the kind. A custom kind costs one watch when opened, like any kind
(`TestBudgetCustomResourceView`); the cache holds the definitions in full, schemas included.

**Not across restarts** (decided, [D-13](decisions.md#d-13-cache)). The cache lives in memory
only. Persisting it to disk would make start-up faster but would show stale state and put cluster
data, Secrets included, at rest on the disk.

---

## 6. Cluster connections (`internal/cluster`)

- One connection per kubeconfig context the user opens: a `rest.Config` from client-go's
  `clientcmd`, its discovery client, its access-review cache, its cache (§5).
- **Auth** as `kubectl` does it, because it is `kubectl`'s code: client certificates, tokens, exec
  credential plugins, the legacy `oidc` auth provider (refreshed tokens go through a persister
  Cogpit supplies, [H-4](../security/credentials.md#h-4-auth-provider-token-refresh)), `proxy-url`.
  client-go 0.37.1 rejects the removed `azure` provider and points to kubelogin.
- **Streams:** logs (follow, previous container, timestamps), exec and attach, port-forward. Local
  port-forward listeners bind to `127.0.0.1` only.
- **The API server's warnings** (built 2026-10-11, [`warnings.go`](../../internal/cluster/warnings.go)):
  every client of a connection hands them to one handler that logs each distinct warning once per
  connection and keeps them (`Conn.Warnings`: at most 64, and how many arrived). client-go's
  default logged every one, and the API server attaches a deprecation warning to every request
  for a deprecated kind: v1 Endpoints since Kubernetes 1.33 warn on each list, each watch an
  informer reconnects and each get (`TestConnectHandlesWarnings`). Measured on kind 1.36.1
  (`TestKindStorageAndNetworkViews`): the Endpoints view's streaming list brought one warning, each
  further list one more, and one line was logged.

### Kubeconfig sources (built 2026-10-12)

Every context comes from a source file read on its own; nothing is merged (D-28, built 2026-10-12;
until then the default kubeconfig files were merged as kubectl merges them). The sources are
([`engine/kubeconfigs.go`](../../internal/engine/kubeconfigs.go), the commands in
[`commands/kubeconfigs.go`](../../internal/engine/commands/kubeconfigs.go)):

- **`~/.kube`**, the default sync folder (below), `config` an ordinary file of it.
- **The files named for this run**, `--kubeconfig` (what `make run` passes) or else `$KUBECONFIG`
  (`cluster.NamedKubeconfigFiles`, passed to `engine.WithSessionFiles`; never `~/.kube/config` as a
  fallback): each a file source with `Session` set, shown with a "this run" badge, removable, but
  only for the session (kept in memory, back at the next start, never written to
  `kubeconfigs.json`, its approvals kept). A missing file, a non-regular file (`/dev/null`) and a
  file the sync folder owns already (directly in `~/.kube`, even after the folder was removed) are
  not offered. Adding such a file by hand stores it.
- **What the user adds:**

  - **A kubeconfig file** (`kubeconfig.addFile`): read, re-read when it changes, never written.
  - **A sync folder** (`kubeconfig.addFolder`): every regular file in it (not in subfolders; dot
  files are read since 2026-10-12, as Lens reads them; `~` backups, `.swp`, `.tmp`, `._*` and `.DS_Store`
  skipped as noise; at most 500 files of at most 1 MiB) that parses and defines a context. What
  makes a hidden file harmless is not its name: `cluster.ListSyncFolder` takes only regular files
  (`os.Stat` follows a link, so a FIFO or device is never listed, let alone opened), of at most 1 MiB,
  and `InspectFile` looks at the file before it opens it (opening a FIFO for reading blocks) and
  again after (`TestSyncFolderNeverOpensAFIFO`).
- **`~/.kube`, the default sync folder** (2026-10-12; Lens syncs it by default and lists it as an
  ordinary, removable item, captured). `cmd/cogpit` passes `<home>/.kube` to
  `engine.WithDefaultFolder`; the engine never looks up the home directory or the environment itself, so
  tests give it temporary ones (`TestNoDefaultFolderWithoutTheOption`, `kubeconfigs_session_test.go`). It is listed first, as a
  sync folder with `Default` set, while the directory exists; it is not written to the list of
  sources, and removing it is (`defaultFolderRemoved` in `kubeconfigs.json`), so that it does not
  come back; adding `~/.kube` again by hand makes it an ordinary source. `config` is one of its
  files, listed as `<context> [.kube/config]` like the others, so removing the folder hides it
  (Lens does the same). A start without `~/.kube` lists no folder; it appears when the directory
  does. The trust model is below.
- **One dialog for both** (`internal/ui/fyneui/syncpicker.go`): Lens opens the system's file dialog,
  which takes files and folders together; Fyne's two file dialogs take one kind each, so Cogpit
  has a small browser of its own, "Sync Files and Folders": select a file or a folder, or nothing for
  the folder shown, and press Sync; a path typed or pasted into the path field and not opened
  with Enter wins over both (since 2026-10-12: Sync added the folder shown, the home directory,
  `TestSyncPickerSyncsATypedPath`); the status line names the path that will be synced, hidden
  files are listed, and only directory listings are read. A sync that finds no context says so in a
  dialog ("No clusters found") with the skipped files and a fixed reason each (`SourceStatus.SkippedFiles`:
  "defines no context", "not a kubeconfig, or unreadable"; never a parser's message, which could quote
  the file), and offers to remove the source again; a folder's subfolders are not read. `Session.WatchKubeconfigs` compares the sources' file sizes and modification
  times every 2 seconds (polling, no file-system notifications: `fsnotify` is not a direct
  dependency) and emits `EventContexts` when they change; the picker lists again.

The list of sources is `<user config dir>/Cogpit/kubeconfigs.json` (0600). Each source file is
read **on its own**, never merged with another file: a context in a dropped file can then never
borrow the user entry of the user's own kubeconfig and send its credential to the dropped file's
server. Its contexts are offered as `<context> [<label>]`: the label is the file's name
(`dev.yaml`, `dev.yaml (2)` for a second file of that name), or the folder's name and
the file's (`kubeconfigs/team.yaml`, `.kube/config`). A context name two files share therefore
gives two contexts (`prod [a.yaml]`, `prod [b.yaml]`), both reachable
(`TestSameNamedContextsInTwoFilesAreBothReachable`). That id is `UIState.Context` and the context of every
confirmation, audit line and journal entry; the connection reads the one file
(`cluster.Options.KubeconfigPaths`), and a terminal binds kubectl to the context's name in that
file (`Conn.Context()`). The terminal's `KUBECONFIG` is its own file first, then the context's own file
alone (no other file of the user's), as it is for every source context.

`cluster.contexts`, the CLI and MCP list exactly these ids, with `source`, `kubeContext`, `file`
and `exec`; no context carries a bare name any more, and "current" is not set (the files'
`current-context` fields are not merged, so the picker preselects the first). A bare name typed
for `--context` or `cluster.connect` goes through `ClusterConnector.ResolveContext`
(called by `Session.Connect`): an id or `demo` is kept, a name that exactly one file defines
becomes that file's id (so `make run`'s `--kubeconfig <kind file> --context kind-cogpit` still
reaches kind, `TestRunFlagConnectsABareContextName`), a name several files define is refused with
their ids. The demo cluster stays the bare context `demo`; a file's own `demo` is `demo [file]`.
Without `Sources` (tests, and a start without a configuration directory) the connector still merges
`KubeconfigPaths` as before.

**Credential plugins of sources run only after the user allowed them** (trust-boundaries H-1). The
connector passes `KubeconfigSources.Policy` as `cluster.Options.ExecPolicy`; `cluster.Connect`
asks it with the plugin described from the very configuration the client is built from
(`rest.Config.ExecProvider`), before any client exists, so nothing runs when it refuses and a file
changed between listing and connecting cannot slip another command past it. A `tokenFile` outside
the kubeconfig's folder goes through the same policy and the same dialog (D-17, H-24): it is asked
**before** client-go builds the `rest.Config`, which is what reads the file (`checkTokenFilePolicy`
in [`cluster/sources.go`](../../internal/cluster/sources.go)). An approval
(`<user config dir>/Cogpit/exec-approvals.json`, 0600) is recorded per file and context with a
SHA-256 fingerprint (version 2) of the file, context, user, server URL (with any user information),
CA data or file, `insecure-skip-tls-verify`, TLS server name, proxy URL, API version, command,
arguments, environment names and values, interactive mode, `provideClusterInfo`, the token file
when it needs approval, and the SHA-256 of the program and of the CA file when they resolve to a
file inside the kubeconfig's own folder (R-5; a program found elsewhere, on `PATH`, is bound by
its path)
([`kubeconfig/exec.go`](../../internal/kubeconfig/exec.go) for the document and what it hashes,
[`cluster/pluginfiles.go`](../../internal/cluster/pluginfiles.go) for the file I/O: `kubeconfig`
does none, so `cluster.BindPluginFiles` reads the files, bounded and regular files only, when a
file is described and again right before the policy decides at connect time); any change asks
again. `kubeconfig.execPlugin` (one context) and `kubeconfig.pendingApprovals` (a group) return how the server is reached and trusted (`tls`: proxy URL
without user information, insecure, TLS server name, own CA) and the file facts, which the
approval dialog shows (R-11). Every file is a source now, `~/.kube/config` and the `$KUBECONFIG`
files included, so every plugin waits for the user, remembered across restarts in
`exec-approvals.json`, asked again when the plugin changes
(`TestConfigWithAPluginAsksForApprovalOnce`).

**Asking once per file and plugin** (D-28, owner 2026-10-12, built in scope C4;
[`engine/kubeconfigs_approvals.go`](../../internal/engine/kubeconfigs_approvals.go)). Only the
asking is bundled, the approvals are not. `Session.PendingExecApprovals` (the command
`kubeconfig.pendingApprovals`, read-only, open to every origin, arguments withheld from agents as in
`kubeconfig.execPlugin`) lists the logins that wait, grouped by **(file, plugin identity)**.
`kubeconfig.ExecPlugin.Identity` is the SHA-256 over the API version, command, arguments,
environment names and values, interactive mode, `provideClusterInfo`, the program's binding (resolved
path; the content digest or note for a program inside the folder, D-26) and the token file when it
needs approval (D-17); it leaves out the context, user, server, CA, proxy and TLS settings, which
differ per context and which each context's group entry lists itself (`ExecApprovalContext`: server,
`tls`, CA file facts, state, fingerprint). Only contexts that are not allowed yet are listed, so a
context added to the file later is a group of its own, or of the other new ones; a context whose
plugin changed is listed with state `changed`, and a file whose contexts changed together stays one
group. The dialog shows the shared plugin once and each context's server and R-11 warnings under its
name ([`kubeconfigs_approval.go`](../../internal/ui/fyneui/kubeconfigs_approval.go); it scrolls as a
whole when it would not fit the window). `kubeconfig.approveExec` (GUI only) takes the list of
`(context, fingerprint)` the dialog showed. `Session.ApproveExecAll` reads the files once, compares
every fingerprint with the file as it is now and, if any differs or its context left, records
nothing and returns `*ExecChangedError` naming those contexts (`errors.Is` `ErrExecChangedSinceShown`);
otherwise `KubeconfigSources.ApproveAll` writes one approval per context, each with its own
fingerprint, in one write. Because the fingerprint still binds the server and its trust, a changed
server in one context asks again for it even when its siblings are allowed
(`TestPendingApprovalsGroupContextsByFileAndPlugin`, `TestApprovingAGroupApprovesExactlyItsContexts`,
`TestApprovalIsRefusedAsAWholeWhenOneContextChanged`, `TestAContextAddedLaterAsksOnItsOwn`,
`TestTokenFileApprovalsBundleToo`, `TestPendingApprovalsAreBundledAndApprovedByTheGUIOnly`,
`TestApprovalDialogListsEveryContextOfThePlugin`, `TestApprovalDialogOneConfirmApprovesAll`).

**What `~/.kube` as a default folder means for the trust model** (2026-10-12; the same text is in
[trust-boundaries H-1](../security/trust-boundaries.md#h-1-kubeconfig-files-added-by-sync-folders)):
the folder is the user's own, and nothing it holds is written or run by Cogpit unasked. Since D-28
`config` is no exception: kubectl would run its plugin unasked, Cogpit asks, as for every file of a
folder that tools and people drop kubeconfigs into. Each is read **on its own**, never merged, and a
credential plugin of any of them runs only after the user allowed it. A process that can write to
`~/.kube` could replace `config` and so get a plugin run by kubectl; in Cogpit it gets a dialog
instead. A file that arrives there by accident (a kubeconfig saved or generated into the folder)
becomes contexts, which is where H-24's token-file exposure reaches too.

**A connection ends with its file** (2026-10-12, `Session.CloseGoneSourceConnection`, called by
`WatchKubeconfigs` after each change and by `kubeconfig.remove`): `Backend.SourceFile` is the file
the connection was made from; when `KubeconfigSources.Fate` finds it gone, or no source covers it any
more, the session closes the backend and leaves the UI `Disconnected` with the reason in
`Connection.Error`, sends a warning toast and logs the reason. Only a plain fact closes a
connection: a file that is being rewritten, does not parse for a moment, or lost the context
stays connected (it was built from the file as it was), and an unreadable list of sources or a
permission error counts as "still there" (`TestConnectionStaysWhileItsFileIsOnlyRewritten`). A
connect that started in the meantime is never touched (`disconnectBackend` closes only the backend
it was given). Removing a source (Cogpit's link only; the files stay on disk) ends a connection made
from it the same way, and the GUI's confirmation says so.

**The connect log** (2026-10-12, [`connectlog.go`](../../internal/engine/connectlog.go)): the session
keeps the last attempt to connect to each context, in memory, as INFO and ERROR lines (at most 100
lines of 600 bytes for 64 contexts; `cluster.connectLog`, a read command, returns them to every
origin). It holds "Connecting to", the connect error (which never carries a credential,
`kubeconfig.SafeError`), "Connected" and the reason a connection was closed; `Contexts()` marks
a context whose last line is an error (`ContextInfo.ConnectProblem`), and the picker shows a warning
icon that opens "Connection Updates: <context>". A credential plugin's standard error is not in it:
client-go sends it to Cogpit's own.

Adding a file or folder, removing a source and allowing a plugin are refused for every
origin but the GUI (`engine.ErrGUIOnly`, checked in each command's `run`), as `secret.reveal` is:
a kubeconfig can name a program to run on the user's computer, so an agent must not be able to
put one there or let one run. Agents may list the sources (`kubeconfig.sources`) and read a
plugin's description (`kubeconfig.execPlugin`, `kubeconfig.pendingApprovals`, without its arguments).

**A cluster's link, and removing it from Cogpit** (built 2026-10-13, G13,
[`engine/contextlink.go`](../../internal/engine/contextlink.go), the commands in
[`commands/clustermenu.go`](../../internal/engine/commands/clustermenu.go)). `cluster.links` (a read,
every origin) tells for each context its source, its file, every context that file defines, and
whether `cluster.remove` can unlink it; `cluster.remove` (GUI only, `ErrGUIOnly` in its `run`, as
`kubeconfig.remove`) removes the source of a file the user added, as `kubeconfig.remove` does: every
context of the file goes, its approvals are forgotten, a connection made from it closes, and the
file stays on disk. Refused, with the reason in `Enabled`, for a context of a sync folder (the
default `~/.kube` included: the link is the folder's, and unlinking it would take every cluster in
it), of a file named for this run (it comes back at the next start) and for the demo cluster.
Cogpit's rule, inferred, not the owner's decision (Lens's is not captured). The window asks first,
naming the path and every context that leaves (`TestClusterRemove`).

**Pasting is not a target** ([D-21](decisions.md#d-21-pasting-a-kubeconfig), owner, 2026-10-12): the
paste command, dialog and owner-only store built earlier the same day were removed. A build that
had been used in between left `kubeconfigs.json` entries of kind `pasted` and files
`<user config dir>/Cogpit/kubeconfigs/pasted-<16 hex>.yaml`. Cogpit no longer lists or reads such
an entry (`loadSources` skips it; the next save of the list drops it) and never deletes the files,
which are the user's: they can be added again as files, or deleted by hand
(`TestPastedSourcesOfEarlierBuildsAreNotOffered`).

## 7. MCP and the built-in agent

**MCP server** (`internal/mcp`, `github.com/modelcontextprotocol/go-sdk` v1.8.0):

- tools generated from the registry, with the registry's schemas and MCP tool annotations
  (`readOnlyHint` for `Read` and `UI`, `destructiveHint` for `Destructive`);
- Streamable HTTP on `127.0.0.1`, a random bearer token written owner-only to the app's runtime
  directory, `Host` and `Origin` checks against DNS rebinding;
- the user can switch this *external* endpoint off and on from the status bar item
  ([D-23](decisions.md#d-23-a-link-to-the-owner-in-the-status-bar)): `Server.Stop` ends the
  listener, every connection and pending call and removes the token file, and `Start` brings it
  back with a new token and port. The window holds the switch as `fyneui.MCPControl`, implemented in
  [`cmd/cogpit/mcpcontrol.go`](../../cmd/cogpit/mcpcontrol.go) (internal/ui may not import
  internal/mcp); the choice is kept in Fyne's preferences and `--no-mcp` overrides it. The switch
  stops only the HTTP listener: the engine, and the built-in agent below, which calls
  `Session.Dispatch` in-process, do not depend on it;
- `cogpit-cli mcp` as a stdio bridge for agents that only speak stdio (built 2026-10-13,
  [`mcp/bridge.go`](../../internal/mcp/bridge.go), [`cogpit-cli/mcp.go`](../../cmd/cogpit-cli/mcp.go)):
  `Server.Start` writes the endpoint's URL to `mcp-endpoint` beside the token file (same rules,
  removed with it); the bridge reads both (`ReadEndpoint`: a loopback `http` URL with the path
  `/mcp` and a 64-hex token, nothing else), connects as an MCP client over Streamable HTTP with the
  token (no redirects, no proxy), and serves the app's tools, instructions and annotations under
  the name `cogpit` on stdio, each call forwarded as it came, so the app dispatches it with origin
  `ExternalAgent`. When the app restarts, a call that never reached it (nothing listened, or 401
  before any command) reads the files again and is sent once more; one that reached it is never
  repeated. Without an endpoint file, with one nothing listens at (a crash), or with `--headless`,
  it serves a headless engine of its own instead (`Server.ServeTransport`; no confirmer, so
  commands that need the user's confirmation are refused; the app's kubeconfig sources and
  approvals; the audit log). Decided by Cogpit, not the owner: the mode is chosen once, at the
  bridge's start; a bridge forwarding to an app that quits does not fall back to a headless engine
  ([H-28](../security/agents-and-mcp.md#h-28-the-status-bars-switch-does-not-reach-a-headless-bridge)).

Example: "show me the Service with ClusterIP 1.2.3.4" becomes `resources_find {kind: services,
field: clusterIP, value: 1.2.3.4}` → one reference → `ui_navigate {ref}` → the GUI opens Services,
selects the row, scrolls to it and opens the drawer. All from the cache, without an API request.

**Built-in agent** (`internal/agent`, after the alpha, [D-10](decisions.md#d-10-built-in-agent)):
a chat tab in the dock. Each turn sends the conversation, a compact `ui_state`, and the registry's
tools to an OpenAI-compatible Chat Completions endpoint (LiteLLM, Open WebUI, or any compatible
server), streams the answer, and runs the tool calls through `Session.Dispatch` with origin
`BuiltInAgent`, so the confirmation rules of D-2 apply. Limits: tool calls per turn, tokens per
turn, and what cluster data may leave for the endpoint ([agents and MCP](../security/agents-and-mcp.md)).

**Confirmation flow.** A command from an agent whose danger needs confirmation raises
`ConfirmRequested`; the GUI shows the request (command, objects, namespace, context, origin); the
command runs only after the user accepts. Without a GUI (headless), such commands are refused.
Every agent-origin command lands in the audit log.

## 8. Helm (`internal/helm`)

The Helm SDK (`helm.sh/helm/v4`) runs in-process: releases, history, values and manifests come from
the release Secrets in the cache; install, upgrade, rollback and uninstall run as engine jobs.
These are the operations that leave a release `pending-*` when the process dies halfway
([D-11](decisions.md#d-11-crash-stance)), so they run in a worker process (`cogpit-cli`), or the
next start detects the pending release and offers to finish or roll it back. Which of the two is
decided when M6 is designed.

**Built (2026-10-11, the read side and two actions).** The releases are the virtual kind
`helmreleases` ([`releases.go`](../../internal/helm/releases.go), §4.5): `Derive` groups the
cached Secrets labelled `owner=helm` (served by the full Secrets watch while that runs, by their own
label-selected watch otherwise: §5 › Covering scopes, 2026-10-12) per namespace and release name, the newest revision wins,
and the object carries that revision's Secret UID and resourceVersion and a summary of the
history, never values or the manifest (the YAML tab shows this summary). Decoded payloads are
memoized per Secret UID and resourceVersion, values and manifests for the newest revision of each
release only; entries for Secrets that are gone leave the memo after two generations of 64 calls,
and it holds at most 4,096. Five minutes after the last `Derive` (nobody reads the releases, and
their watch ends after the cache's warm period) the memo forgets everything it decoded, so the
values leave memory with the Secrets they came from (review 2026-10-11,
`TestDeriveMemoForgetsWhenIdle`); before, they stayed until the cluster was disconnected. The drawer
is Lens's since the fourth run (captured 2026-10-12; [`detail.go`](../../internal/helm/detail.go)):
"Release: <name>" (`KindProfile.DetailTitle`), Properties with Chart ("<chart>-<version>", an Upgrade
button beside it, an item action), Updated ("68s ago (12 Oct 2026, 8:30:16 CEST)"), Namespace
(plain text), Version (the chart's version, inferred) and Status as Lens writes it ("Deployed"; the
other states capitalized the same way, inferred) in a neutral grey pill (`DetailItem.Pill`; the
list's column keeps its status colours), then the user-supplied values, the chart's notes (both `Private`: agents read
`model.Withheld`; cut for the drawer to 16 KiB or 200 lines) and the resources of the newest
revision's manifest, one table of Name and Namespace per kind with links for built-in kinds. As
for every virtual kind, the engine adds no generic metadata and no events, and the drawer opens no
events watch (§4.4). The History table and the redacted manifest of the third run are gone: Lens's
drawer has neither; the history is `helm.release.history`. The model is `Plain` (set for every
virtual kind with `NoEvents`): the GUI draws the title on one line, hides the status pill beside
the icons and the Details and YAML tabs (the details scroll alone; no YAML is set), as the capture
shows ([`drawer.go`](../../internal/ui/fyneui/drawer.go) `setPlain`). In the GUI the values section
([`helmdrawer.go`](../../internal/ui/fyneui/helmdrawer.go)) is a "User-supplied values only"
checkbox, unchecked at first (the editor reads every value, `helm.release.values` with `all`), the
editor, and Save as a bar the drawer's width under it.
`helm.release.rollback` and `helm.release.uninstall` run `action.NewRollback` and
`action.NewUninstall` in-process with the helm CLI's defaults (hooks waited for, 5 minutes,
rollback history kept to ten) on the connection's own client configuration, behind the
`HelmWriter` seam ([`helmreleases.go`](../../internal/engine/helmreleases.go)). Crash safety took
the second road, like the drain: a journal (`<user config dir>/Cogpit/helm.json`, 0600,
[`helmjournal.go`](../../internal/engine/helmjournal.go)) records each action before the SDK runs
and drops it when the SDK returns without an error; on connect, the context's entries are offered
(EventHelm, `helm.pendingOperations`, a banner in the GUI) to finish (run the action again) or
dismiss, and one the cache shows finished (the release gone after an uninstall, deployed at a
newer revision after a rollback) is dropped with a note ([`helmops.go`](../../internal/engine/helmops.go)).
The Helm SDK takes no context, so a running action cannot be cancelled; the SDK's own reads
(the release history, discovery) are part of the action, not cache reads.

**Upgrade (decided 2026-10-12, Phase 0 of the fourth run; G11, M6).** Lens's Upgrade opens a
"Helm Upgrade: <name>" dock tab (captured 2026-10-12, [resource parity](../parity/resource-parity.md#helm)):

- **Commands.** `helm.release.upgradeTab` (UI) opens or focuses a release's tab: a `DockTab` of
  type `helmUpgrade` whose `Ref` is the release and whose `Helm` holds the chart, the version the
  upgrade installs, the versions the picker offers (the release's own chart version only until
  chart repositories exist) and Lens's "User-supplied values only"; its values are the tab's editor
  text (`editor.get`, `editor.setText`), the release's user-supplied values, which agents read as
  `model.Withheld` as in the drawer. `helm.release.upgrade` (Mutating, as the
  [privilege footprint](../security/privilege-footprint.md) will rate it) upgrades the release with
  the tab's values and version, or with a `values` parameter; agents need the user's acceptance as
  for every mutating command (D-2), and the audit log records it without the values. The
  confirmation binds the newest release Secret's UID, as rollback's does: a release that moved to
  another revision since is refused.
- **Seam.** `HelmWriter` gains `UpgradeRelease(ctx, namespace, name string, up HelmUpgradeRequest)
  (revision int, err error)` with the chart version and the values; the SDK side takes the chart
  from the release's newest revision (no repository) and runs `action.Upgrade.RunWithContext`
  (the upgrade takes a context, unlike rollback and uninstall) with the helm CLI's defaults (hooks
  waited for, 5 minutes) and history kept to ten revisions, as Lens does (captured 2026-10-12).
- **Crash safety.** The journal records an `upgrade` (built: the action, `FromRevision`, `Chart`,
  `ChartVersion`, never the values, which can hold credentials and which the release's pending
  revision holds anyway). On the next connect an entry is dropped as finished when the cache shows
  the release deployed at a revision above `FromRevision`; otherwise it is offered, and finishing
  it rolls the release back to `FromRevision`: the Helm SDK's upgrade refuses a release left
  `pending-upgrade`, its rollback does not.
- **The journal files** (`drains.json`, `helm.json`) get a lock across Cogpit processes and an
  owning process per entry ([H-22](../security/privilege-footprint.md#h-22-two-cogpit-processes-share-the-journals)),
  a change of the journals' storage that the upgrade's entries ride on. Built 2026-10-12
  ([`journallock.go`](../../internal/engine/journallock.go)), below.

**Journals shared by processes** (built 2026-10-12, fourth run). Every Cogpit process of the user
reads and writes the same `drains.json` and `helm.json`; two mechanisms keep them apart:

- **One lock per directory.** Every read and every read-modify-write of either journal runs under
  an exclusive lock on `<dir>/journals.lock` (0600): `flock(2)` on macOS and Linux, `LockFileEx` on
  Windows (built and vetted by CI there, not run), no lock on other platforms. Both lock the open
  file, not the process, so the kernel drops them when the process ends, however it ends: there is
  no stale lock to clean up. A writer waits in the kernel for the lock, at most 10 seconds, then
  fails with `ErrJournalBusy` (a drain or Helm action is then not started). The lock file is never
  removed. Within a process the two journals take it one at a time.
- **An owner per process.** A process that writes an entry registers once per directory: it
  creates `<dir>/owners/<id>.lock` (a random 128-bit id, created and locked under the directory
  lock) and holds an exclusive lock on it until it ends. Every entry it writes names it (`owner`:
  `{id, pid}`). An owner lives while its file is locked; the PID is shown, never trusted, so a
  reused PID cannot make an ended process look alive. Files are opened close-on-exec, so a shell or
  `kubectl` that Cogpit starts does not inherit the lock.
- **What follows.** An entry in state running whose owner is another live process belongs to it:
  `node.pendingDrains` and `helm.pendingOperations` report it as running with `elsewhere` (no
  progress), the connect's offer leaves it out, and the journal refuses to replace it (Begin of the
  same drain or release action) or remove it (Dismiss) with `ErrRunsElsewhere`. A failed entry is
  anyone's to finish or dismiss. Entries of an owner that ended, and entries without an owner
  (written before owners existed; the file format stays version 1, `owner` is optional), are
  interrupted as before; a process that registers removes the owner files of ended processes (at
  most 256 per registration). An owner id read from the file is used only if it is 32 hex digits.
- **Measured** (2026-10-12, Apple M1 Ultra): four processes writing both journals at once keep
  every entry (`TestJournalsKeepEveryProcessesEntries`; with the lock taken out the same test kept
  85 of 120); a killed process's running drain is interrupted at once (`TestJournalEntryOfAKilledProcess`).
  A journal write (lock, read, write, sync, rename) costs 13.4 ms on macOS, of which the
  `F_FULLFSYNC` that makes it durable is all but 0.35 ms (a 20-entry journal, 200 writes).

**Upgrade as built (2026-10-12, the fourth run; [`helmupgrade.go`](../../internal/engine/helmupgrade.go),
[`commands/helmupgrade.go`](../../internal/engine/commands/helmupgrade.go),
[`helm/actions.go`](../../internal/helm/actions.go), [`helm/values.go`](../../internal/helm/values.go)).**
Where it differs from the contract above, this paragraph wins.

- **Commands.** `helm.release.upgradeTab` (UI, the drawer's and row menu's Upgrade at order 300,
  between Rollback and Delete as Lens orders its icons) opens the tab with the user-supplied values
  decoded from the newest release Secret in the cache; agents read its result and `editor.get`
  withheld. `helm.release.upgrade` (Mutating) takes `ref` with `tab`, or `values` with the
  `revision` they are based on (the drawer's Save, agents), and `version` (only the release's own;
  since 2026-10-13 a repository's too, [below](#8-helm-internalhelm)).
  Two reads were added for the GUI: `helm.release.values` (the user-supplied or all values from the
  cache, the drawer's editor; withheld from agents) and `helm.release.history` (the revisions from
  the cache, newest first, with the newest Secret's UID: the Rollback dialog).
- **What binds.** The engine records UIDs only in destructive confirmations, so an upgrade does not
  bind through its confirmation: it binds to the revision its values are based on, the tab's (its
  revision and release Secret UID when it read the values, moved by a successful upgrade from the
  tab) or the `revision` parameter, checked when the command runs, after any confirmation. A
  release that moved meanwhile is refused with `ErrTargetsChanged`, also while an agent's
  confirmation was open; the tab shows itself stale (`editor.get`: `baseRevision`,
  `cachedRevision`). The cache can lag behind the server, so the upgrade (and a rollback, with the
  cached newest revision) also hands that revision to the SDK action (`HelmUpgradeRequest.FromRevision`,
  `Actions.Rollback`'s `from`), which compares it with the newest revision it reads from the
  server and refuses with `helm.ErrReleaseMoved` and `helm.ErrUnchanged` before anything changes;
  the command reports `ErrTargetsChanged` (R-1 of the fourth run's review; the SDK reads the
  revision once more itself, a window of milliseconds). An upgrade from a tab also names the tab's text revision it takes
  (`textRevision`, required with `tab`; the GUI's Save passes the one its own `editor.setText`
  returned): a text changed since, by an agent's `editor.setText` while the confirmation is open
  or between Save's two dispatches, is refused with `ErrTargetsChanged` (review of the fourth run,
  R-2, 2026-10-12). A release pending another action is refused before anything runs.
- **SDK.** The chart is the newest revision's own (read by the SDK, part of the action); the values
  replace the user-supplied ones whole (`ResetValues`: an empty editor is no values, where the helm
  CLI would keep the old ones); history ten, hooks waited for, 5 minutes. `RunWithContext` gets
  `context.WithoutCancel`: the SDK's upgrade marks a release failed when its context ends while its
  own goroutine goes on changing the cluster, so an upgrade runs to its end or timeout like
  rollback and uninstall. An error the SDK returns before it made a revision (a template or schema
  that refuses the values, a pending release, another chart version) is `helm.ErrUnchanged`: its
  journal entry is removed, since nothing is left to finish.
- **Values never leave the user's window.** Errors, toasts, the journal's error text and the SDK's
  own log lines during an upgrade are scrubbed of every value of four or more characters, also in
  base64, in the escaped forms error texts quote strings in (Go's `%q` and `%+q`, JSON) and numbers
  as a template renders them (`1e+06`) (`helm.ScrubValues`, R-4 of the fourth run's review; values
  a template transformed otherwise are not found); the audit log
  records `values` and a Helm Upgrade tab's `editor.setText` text by size only; YAML parse errors
  name the line, never a value. A revision's description, which Helm writes from a failed
  upgrade's error (seen on a real cluster, 2026-10-12: the API server's message), reaches the
  release object scrubbed of that revision's values. The SDK's value coalescing writes conflicting values with the
  standard library's `log`, which reaches Cogpit's log through `slog.SetDefault`: `cmd/cogpit`
  wraps its handler in `helm.RedactSDKLog`, which drops them.
- **Crash safety.** On connect an upgrade entry is dropped when the release is gone, deployed above
  `FromRevision` (finished), or still at `FromRevision` (it never reached the cluster); otherwise
  (pending-upgrade or failed above it) the banner offers "Roll back to revision N"
  (`helm.release.rollback` to `FromRevision`), and `helm.release.upgrade` stays refused while the
  release is pending.
- **The tab in the GUI** ([`helmupgradetab.go`](../../internal/ui/fyneui/helmupgradetab.go)): save and
  find icons, the chart and its version as text beside a separate version picker (placeholder
  "Select…"; without a repository that lists the chart it offers the release's own version; with
  one, since 2026-10-13, below), the line "Release
  <name> values.yaml, version …, on <namespace> namespace." with the release and the namespace as
  links (`ui.navigate`) and a spinner in the version's place until the values are loaded, and the
  values editor. The find icon opens the editor's find widget at its bottom
  ([`findbar.go`](../../internal/ui/fyneui/findbar.go)): search text with match case, whole word and
  regular expression, replace with Replace and Replace all, previous and next, "1 of 3". It works
  on the editor's text through `yamlview.View` (`SelectRange` for the match, `ReplaceSelection` for
  an edit, which reaches the engine like typing does); matches are per line and capped, nothing
  here touches the engine. Save is unchanged: it sends the editor's text, then upgrades bound to the
  text revision its own `editor.setText` returned.
- **The Rollback dialog is the confirmation** (GUI, [`helmrollback.go`](../../internal/ui/fyneui/helmrollback.go)).
  Rollback stays destructive and the engine asks the GUI's confirmer for every origin. Lens's dialog
  (a REVISION dropdown, newest first, the current one selected, Rollback enabled with it, as
  captured; the engine accepts the newest revision as a target and helm writes a new revision equal
  to it, a reading of helm's behaviour, not of Lens's, which the dialog says) names the release, its namespace and
  its context, and when the user picks Rollback hands the confirmer the newest release Secret's UID
  it showed; the engine's confirmation of exactly that command, from the GUI origin, on that release
  and context, for the revision the user picked (the confirmation's detail names it,
  `commands.RollbackDetail`), within ten seconds, takes that answer instead of a second dialog, and
  one whose UID differs (the release moved while the dialog was open) is refused. The answer is
  dropped when its dispatch ends without taking it (refused before the confirmation), so it confirms
  no other rollback in its window (R-3 of the fourth run's review). Agents' requests carry their own
  origin and never match. The banner's and the agents' rollbacks are confirmed as before.

**Chart repositories and the Charts view (built 2026-10-13, night run; G11;
[`helm/repo.go`](../../internal/helm/repo.go), [`engine/helmrepos.go`](../../internal/engine/helmrepos.go),
[`commands/helmrepos.go`](../../internal/engine/commands/helmrepos.go),
[`kinds/helmcharts.go`](../../internal/model/kinds/helmcharts.go)).**

- **Where they are kept: the lead's default (2026-10-13), not the owner's decision** (the owner's
  open question "where Helm chart repositories are kept"). The list is Cogpit's own:
  `<user config dir>/Cogpit/helm/repositories.json` (0600 in a 0700 directory, replaced atomically
  through the same writer as the other private files, at most 64 repositories); each index lives in
  Cogpit's own cache folder, `<user cache dir>/Cogpit/helm/<name>-index.yaml` (0600, replaced only
  after the Helm SDK parsed the new download). Cogpit never reads or writes Helm's
  `repositories.yaml`, `$HELM_CONFIG_HOME` or `$HELM_CACHE_HOME`: no implicit writes outside
  Cogpit's folders (the golden rules), and Helm's file keeps an authenticated repository's
  password in plain text, which Cogpit would then hold. So the helm CLI's repositories do not
  appear in Cogpit until the user adds them here; whether Lens reads Helm's own repositories is not
  verified.
- **Public https repositories only.** A name is 1 to 63 letters, digits, `-` or `_` (no dot: a
  chart's object is named `<repository>.<chart>`); a URL is https with a host and without a user,
  password, query or fragment; redirects are followed to https only, at most 10. No credentials,
  client certificates, CA files or plain http (an authenticated repository is
  [H-29](../security/credentials.md#h-29-authenticated-helm-repositories)), no OCI registries.
- **Downloads.** On `helm.repo.add` (a repository whose index does not download and parse is not
  added), on `helm.repo.update`, and, as §5 has it for data that cannot be watched, while the
  Charts view is the open view or a Helm Install tab is in the dock: the indexes older than their
  time-to-live (10 minutes, Cogpit's default) are downloaded at once and the age is checked every
  minute; with neither open nothing is downloaded (`TestHelmIndexRefreshedOnlyWhileShown`). One
  download at a time, each bounded (Cogpit's defaults): an index to 32 MiB, a chart archive to
  16 MiB, two minutes. The index is parsed with the SDK (`repo.LoadIndexFile`, which also drops
  entries it finds invalid); Cogpit keeps of each version its version, app version, description
  (cut to 512 bytes), home, icon URL (never fetched), Kubernetes version, type, keywords,
  maintainers' names, sources, deprecation, creation time, digest and URLs, at most 1,000 versions
  per chart (`FuzzLoadIndex`).
- **The charts are the local kind `helmcharts`** (`cogpit.io/v1` `HelmChart`, cluster-scoped,
  named `<repository>.<chart>`): written into the connected backend's local store on connect and
  after every change of an index, so that the Charts view, its search, the drawer and agents'
  `resources.*` read them like any kind; the demo cluster has none. A chart is listed with its
  newest version that is not a semantic-version prerelease (helm search's default; Lens's choice is
  inferred), its object carries every version newest first. The navigator's Helm group lists
  Charts before Releases, as captured.
- **Commands.** `helm.repo.list` (read), `helm.repo.add` and `helm.repo.remove` (UI; refused for
  agents before anyone is asked: like the kubeconfig sources, where Cogpit fetches charts from and
  which https servers it reaches from the user's computer is the user's choice; the lead's default),
  `helm.repo.update` (UI, every origin).

**Install (built 2026-10-13, night run; G11; [`helm/install.go`](../../internal/helm/install.go),
[`engine/helminstall.go`](../../internal/engine/helminstall.go),
[`commands/helminstall.go`](../../internal/engine/commands/helminstall.go)).** Lens's Install
opens a "Helm Install" dock tab (resource parity; the tab itself is not captured, so its layout
is inferred).

- **The tab.** `helm.chart.installTab` (UI; Install in a chart's row menu and drawer, `ref` the
  chart, `version` optional: the listed version) downloads the chart's archive (bounded, checked
  against the index's digest when the index lists one, `ErrDigestMismatch` otherwise; the archive
  must hold the chart and version the index names) and opens a `DockTab` of type `helmInstall`,
  titled "Helm Install: <chart>", whose `HelmInstall` holds the repository, chart, version, the
  versions the index lists, the archive's digest and whether it was verified, the namespace the
  tab starts with (the namespace filter's single one, else the context's, else `default`) and a
  release name made from the chart's with five random characters (Cogpit's default; Lens's is not
  captured). Its editor holds the chart's `values.yaml` as shipped, comments included; agents read
  it withheld, the audit log records a `helmInstall-…` tab's `editor.setText` by size only, as for
  an Upgrade tab. The archive stays in memory with the tab, so that the install installs exactly
  what the tab showed; another version reloads the tab and replaces the values with that
  version's.
- **Preview.** `helm.chart.preview` (read) runs the SDK's install with `DryRunServer`: the
  templates render against the cluster's capabilities, every rendered object is read to find one
  that exists already and is not the release's, nothing is written. It returns the objects, the
  hooks and the custom resource definitions of the chart's `crds/` folder by kind, namespace and
  name, never their content (a rendered manifest holds the values). Only the window's preview
  renders against the cluster: an agent's or the CLI's renders offline (`DryRunClient`, helm
  template: default capabilities, no `lookup`, no check for existing objects, `offline` in the
  result), because charts reach `lookup` from values through `tpl`, and the names or the error a
  preview returns would otherwise carry what the cluster holds, Secrets included, to a caller that
  reads Secrets redacted everywhere else (`TestKindHelmInstall`: a probe template named from a
  `lookup`).
- **Install.** `helm.chart.install` (mutating) takes a tab with its text revision (an install from
  a tab is bound to the text the user reviewed, as an upgrade is), or a repository, chart, version
  and values (agents); the namespace, the release name and `createNamespace` (off by default) are
  parameters. Refused when the cache holds a release of that name in the namespace; the SDK side
  refuses one whose history exists on the server before anything runs. The SDK runs with the helm
  CLI's defaults: server-side apply, hooks waited for, 5 minutes, no `--atomic`, no dependency
  update (a chart whose `charts/` lacks a dependency is refused, as is a library chart), no
  post-renderer, no DNS lookups; `RunWithContext` gets `context.WithoutCancel` as the upgrade does.
  An error the SDK returns before it stored revision 1 (a template or schema that refuses the
  values, an object that exists, a missing namespace) is `helm.ErrUnchanged`: the journal entry is
  removed. Custom resource definitions from `crds/` and a namespace created with
  `createNamespace` stay after such an error, as helm leaves them.
- **What binds and who is asked.** The GUI's install is not confirmed by the engine (mutating, D-2);
  the Install tab shows the preview first and installs only from that dialog (Cogpit's choice).
  An agent's install waits for the user's acceptance: the confirmation names the release, its
  namespace and the context (its object), the chart, version and how the archive was checked
  (its detail), and shows values given inline ([D-27](decisions.md#d-27-an-agents-helm-values-in-the-confirmation)).
- **Values never leave the user's window.** Errors, toasts, the journal and the SDK's log lines
  during a preview or an install are scrubbed (`ScrubValues`) of the values that differ from the
  chart's defaults (`Chart.UserValues`): the defaults are public, and scrubbing them would hide an
  image or a service type from every message.
- **Crash safety** ([D-11](decisions.md#d-11-crash-stance)). The journal records an `install`
  (`Chart` as `<repository>/<chart>`, `ChartVersion`, never the values) before the SDK runs. On the
  next connect the entry is dropped when the release is deployed (finished) or does not exist (the
  install never stored it); otherwise (pending-install, failed) it is offered and finishing it
  uninstalls the release (`TestHelmInstallCrashJournal`). A journal holding an `install` entry is
  unreadable for a Cogpit from before 2026-10-13 (the file format stays version 1).
- **On kind** (`TestKindHelmInstall`, 2026-10-13): a chart served by an httptest repository (the
  SDK runs in Cogpit's process; the cluster never reaches the repository) added in temporary
  folders, previewed (no request that writes), installed from its tab with `createNamespace`, an
  agent's install with inline values confirmed and audited without them, both uninstalled; the
  requests are in the [privilege footprint](../security/privilege-footprint.md#actions-that-write).

**Upgrade to a repository's version (built 2026-10-13, night run; G11;
[`engine/helmupgrade_repos.go`](../../internal/engine/helmupgrade_repos.go),
[`commands/helmupgrade.go`](../../internal/engine/commands/helmupgrade.go),
[`helm/upgrade.go`](../../internal/helm/upgrade.go)).** Where it differs from the upgrade as built
above, this paragraph wins.

- **Where the versions come from.** The release's chart is looked up by name in the index of every
  repository added to Cogpit, as Cogpit holds it: the lookup downloads nothing and does not wait for
  a download that runs. Without a repository that lists the chart the Upgrade tab is as before: the
  release's own version, its stored chart. With one, the tab (`HelmUpgradeTab`) carries
  `Repository`, whose versions the picker offers, `Repositories`, every one that lists the chart in
  the order they were added, `ReleaseVersion`, the release's own, and `Versions`: the repository's
  newest first (the index's order), the prereleases after the others, and the release's own version
  in its place by version when the repository does not list it (`helm.CompareVersions`, semantic
  versions with a leading `v` allowed). `helm.release.upgradeTab` with `repository` or `version`
  changes an open tab's choice (UI state is engine state: the GUI's pickers dispatch it), checked
  against what the indexes list; a refocus without either keeps the choice.
- **Cogpit's defaults (inferred: Lens's choices are not captured; not the owner's decisions).**
  Several repositories list the chart: the first whose index lists the release's own version (the
  likely source of the release), else the first added. A new tab starts with the repository's
  newest version that is not a prerelease when it is newer than the release's own, so the drawer's
  Upgrade button and the row menu's open on it; otherwise with the release's own. The release's own
  version always upgrades with the release's stored chart, nothing downloaded, as before; only
  another version is downloaded. A chart is matched by its name only: a chart of the same name in
  another repository is offered too, and the tab and the confirmation name the repository.
- **The upgrade.** `helm.release.upgrade` takes `version` and `repository`, both optional (absent:
  from a tab its choice, else the release's own, the behaviour before). A version other than the
  release's own comes from `repository`, else the tab's, else the one repository that lists it
  (several: refused, "give repository"; none: `helm.ErrNoRepository`); `repository` with an empty
  `version` takes its newest version that is not a prerelease. These refusals happen in `Enabled`,
  before anyone is asked. When it runs, after the checks of the upgrade as built (the tab's text
  revision and base, the release's revision, a pending release), the chart is downloaded through
  the install's path (`HelmRepos.FetchChart`: https only, at most 16 MiB and two minutes, checked
  against the SHA-256 the index lists, the archive holding the chart and version the index names)
  before the journal records anything, so a download that fails or does not match leaves nothing to
  finish (`TestHelmUpgradeRepositoryRefusals`). The SDK side (`Actions.UpgradeChart`) refuses a
  chart of another name (`helm.ErrOtherChart`), a library chart and one whose dependencies are not
  packaged with it, and otherwise runs as the upgrade as built (ten revisions, hooks waited for,
  5 minutes, `ResetValues`, `context.WithoutCancel`, `ErrUnchanged` before a revision, values
  scrubbed; `TestUpgradeToARepositoryChart`, `TestUpgradeChartScrubsValues`). An older version (a
  downgrade) is allowed, as helm allows it, and the confirmation says so.
- **What the user accepts is what runs.** An agent's upgrade waits for the user (mutating, D-2).
  The confirmation's detail (`upgradeDetail`) names the chart version from and to, the repository,
  whether the archive is checked against the index's digest and an older version; inline values
  are shown as before (D-27). An agent's upgrade from a tab without `version` takes the tab's
  choice, which the agent could change with `helm.release.upgradeTab` while the confirmation is
  open: when the command runs it builds the detail again and refuses with `ErrTargetsChanged` when
  it reads differently from the one the user accepted (the tab's choice or an index changed), and a
  download whose index no longer lists the digest the detail promised is refused too
  (`TestHelmUpgradeFromARepositoryAgents`). The GUI's Save names the version and the repository the
  tab shows.
- **The tab in the GUI** ([`helmupgradetab.go`](../../internal/ui/fyneui/helmupgradetab.go); inferred:
  Lens with a repository is not captured). With a repository the chart reads
  "<repository>/<chart>" (the captured "/<chart>" read as an empty repository) and the version text
  drops "(no repository)"; the picker lists the versions, the release's own marked "(current)",
  on the engine's choice; a second picker chooses the repository when several list the chart; the
  info line's version is the chosen one, and the status line names a change of chart version
  ("from chart version 4.12.1 to 4.13.0, downloaded from …"). A pick dispatches
  `helm.release.upgradeTab` and the pickers follow the engine's tab, which the GUI reads from its UI
  state when the tab's EventEditor arrives: the dock syncs its tabs when they come, go or move, not
  when a tab's choice changes (`dockEqual` in [`shell.go`](../../internal/ui/fyneui/shell.go)), so the
  engine emits the EventEditor after the UI state (`TestHelmUpgradeTabRepositoryVersions`,
  `TestHelmUpgradeTabRepositoryChoice`, PNGs `helm_upgrade_picker_{dark,light}.png`).
- **No preview.** The upgrade has no preview or dry run, so nothing renders an agent's values
  against the cluster before the user accepts the upgrade itself (the install's preview renders
  offline for agents, above, for the `lookup` through `tpl` reason). What the accepted upgrade
  renders with `lookup` reaches the release's objects, and a template that fails on it can quote it
  in the error the agent receives: the [privilege footprint](../security/privilege-footprint.md#actions-that-write)
  says so for the install and the upgrade.
- **Crash safety** ([D-11](decisions.md#d-11-crash-stance)). The journal records the `upgrade` with
  `Chart` as `<repository>/<chart>` (the install's form) and `ChartVersion` the version it upgrades
  to, never the values; an interrupted one is offered as before, as a rollback to `FromRevision`
  (`TestHelmUpgradeRepositoryCrashJournal`). The upgrade's success moves the tab's base and its
  `ReleaseVersion` to the new revision and version.
- **On kind** (`TestKindHelmUpgradeFromRepository`, 2026-10-13): a chart served by an httptest
  repository (the cluster never reaches it), 0.1.0 installed through the engine into its own
  namespace, the tab starting on 0.2.0, the upgrade from the tab to 0.2.0 (the archive downloaded
  once, the ConfigMap 0.2.0 adds created), an agent's upgrade back to 0.1.0 with inline values
  confirmed naming both versions and "an older version" (the added ConfigMap deleted), the release
  uninstalled and the namespace deleted; the requests are in the
  [privilege footprint](../security/privilege-footprint.md#actions-that-write).

## 9. Metrics (`internal/metrics`)

Metrics cannot be watched, so they are the one documented exception to "never poll what can be
watched" ([§5](#5-cache)): they are polled, and only while something visible shows them. No provider,
a missing permission and a metrics-server that does not answer are **states** the UI shows, not
errors. Cogpit never installs anything into a cluster unless the user starts that action.

**Built 2026-10-12 (G12, first slice): the Kubernetes Metrics Server.** What exists:

| Piece | File | Responsibility |
|---|---|---|
| `Provider`, `Detector`, `State`, `Capabilities` | [`provider.go`](../../internal/metrics/provider.go) | The seam. A `Provider` answers "what do these pods and nodes use now" (`ListPods`, `GetPod`, `ListNodes`, `GetNode`; model types `PodUsage`, `NodeUsage` in [`model/usage.go`](../../internal/model/usage.go)). A `Detector` finds a provider or says why none: `State{Status, Provider, Label, Message}` with the statuses `available`, `notServed`, `forbidden`, `unavailable`, `unknown`. `Capabilities` (`History`, `Network`, `Filesystem`) say which chart variants a provider can draw: the Metrics Server none of them |
| `Server` | [`server.go`](../../internal/metrics/server.go), [`conn.go`](../../internal/metrics/conn.go) | The `metrics.k8s.io/v1beta1` provider on `k8s.io/metrics`' typed client, built on the connection's own HTTP client (`cluster.Conn.HTTPClient`), so the counting transport and the cluster's rate limit see it. Detection is one discovery `get` of the group version: not found is `notServed`, 401/403 `forbidden`, 503 (an `APIService` with no backend) `unavailable`. A pod or node with no measurement yet is `ErrNoData`, not a state |
| `Hub` | [`hub.go`](../../internal/metrics/hub.go) | Polling while subscribed, the result cache and the series, below |

**Subscription and polling.** A consumer calls `Hub.Subscribe(scope, onChange)` and `Release()`s
when its view or drawer goes away. A `Scope` is a kind (`pods`, `nodes`), a namespace (`""` = all) and
a name (`""` = a list). Everything that subscribes to the same scope shares one poll goroutine and
hears every poll through `onChange`; the goroutine ends with the last release. With nothing
subscribed the hub sends nothing, not even discovery (`TestNothingOpenNothingRequested`).

| Scope | Interval | Why |
|---|---|---|
| a list (`{pods, "", ""}` for a Pods view, `{nodes}` for the Nodes view) | 60 s (`DefaultListInterval`; Lens's list refresh) | A list of thousands of pods is a large response, and a column of current values does not need every measurement |
| one object (a pod or node drawer) | 15 s (`DefaultObjectInterval`) | The Metrics Server's own resolution (`--metric-resolution`, 15 s by default): every measurement reaches the chart, none is asked for twice; the response is one small object |

*Results are cached with a TTL* (`DefaultTTL`, 15 s). A result is current for the TTL, and while its
scope is polled for twice its interval, so a late poll does not blank a column. A subscription that
starts on a result younger than the TTL waits until it goes stale before its first poll, so closing
and reopening a view costs nothing (`TestSubscribeOnAFreshResultWaitsForItToGoStale`). `Hub.Fetch`
is the one-off read for agents: it serves a fresh result, or a fresh list that covers the object,
else sends one request and starts no poll (`TestFetchCachesForTheTTL`). *Detection* is kept for 10
minutes when a provider was found and for 2 minutes when not, so a metrics-server installed
meanwhile appears without a restart, and a missing API costs one discovery request per two minutes
while a view is open, nothing otherwise.

*A series per watched object.* While an object scope is subscribed, the hub keeps a bounded time
series of the object (a pod's total and each of its containers, by the measurement's own
timestamp): none more than an hour older than the newest, so 240 points at 15 s (`DefaultSeriesAge`;
`TestSeriesBounds`; since the fourth slice the series is the drawer's chart, which keeps its longest
range and at most 5,760 points, below). A new drawer starts with the point
a fresh list already holds. The series ends with the last subscription: Cogpit holds no history
across drawers; that is what Prometheus is for (below).

*Failures.* A poll that fails keeps the previous result (it expires by the TTL rules above) and sets
the scope's state; the next poll tries again at the interval. A `404` on a list means the API went
away and makes detection run again. A panicking provider is recovered at the goroutine's boundary
(`TestProviderPanicIsContained`).

**In the engine** ([`engine/metrics.go`](../../internal/engine/metrics.go)). `Backend.Metrics` is the
cluster's hub (a real cluster: the Metrics Server through `metrics.ForConn`; the demo cluster:
`metrics.Fixed(demo.Metrics)`); `ClusterConnector.MetricsOptions` tunes it, and it closes with the
backend. Three things subscribe, and each releases when it goes away:

| Subscriber | Scope | Where |
|---|---|---|
| a view of a kind with `KindProfile.Usage` (Pods, Nodes) | the whole list, all namespaces (the namespace filter is applied to the rows, as for every view) | `viewWorker.usage`, a `usageFeed` that bumps a generation and wakes the worker after each poll |
| the open drawer of a pod or node | that object | `drawerWatch.usage`; after each poll the engine emits `EventMetrics`, which the window answers by redrawing the chart only, not the drawer |
| nobody (an agent's `metrics.get`) | a one-off `Hub.Fetch` | no subscription, no poll |

*CPU and Memory columns.* The profiles of Pods and Nodes carry the columns (type
`model.ColumnMetric`: `Text` the formatted value, `Num` the sort key, an empty `Text` is "no
measurement" and sorts last in both directions) and a `UsageCells` function; `Cells` leaves them
empty. The row cache keeps the profile's row as `base` and patches the two cells when the feed's
generation moved (`rowCache.buildWith`), so a new measurement costs a copy of the cells of the
rows that have one, never a rebuild of 10,000 rows from objects, and the sort sees the patched
cells. Pods show cores with three decimals and memory in binary units ("0.012", "19.5MiB"); Nodes show
the share of allocatable ("7.5%"), the text of Lens's usage bars. Without a provider (no
metrics-server, forbidden, down) the columns stay empty; `Session.ObjectMetrics` and `metrics.get`
carry the state.

*The drawer's data.* `Session.ObjectMetrics(ref)` returns `ObjectMetrics`: the status, the source
line's label, the capabilities, and one `MetricsTarget` per chart: the pod or node as a whole
(a pod: the sum of its containers' usage, requests and limits), then each container from the pod
spec in order (`model.PodResources`, `model.PodTotals`), or a node's allocatable and capacity;
each with its latest measurement and the series the hub kept since the drawer opened. A pod's limit
line exists only where every container sets that limit (a container without one makes the pod
unbounded).

*The agents' read.* `metrics.get {kind, namespace?, name?, limit?}` (`DangerRead`; verbs `get` and
`list` on `pods.metrics.k8s.io` and `nodes.metrics.k8s.io`) returns the status, the source and the
current usage, per container for pods, at most `limit` items (default 500, `truncated` says when
more matched). A cluster without the Metrics API answers with the status `notServed` and no items.

**In the window** ([`metricsview.go`](../../internal/ui/fyneui/metricsview.go),
[`chart.go`](../../internal/ui/fyneui/chart.go)). The drawer model says `Metrics: true` for a kind with a
Metrics section; the drawer then builds a `metricsPanel` above the Properties and keeps it across
rebuilds (`drawer.kept`). The panel reads `Session.ObjectMetrics` off the main goroutine, and again
on every `EventMetrics`, and updates its widgets in place: a new measurement redraws the chart and
nothing else (`TestMetricsEventUpdatesInPlace`). It shows the source line ("Displaying metrics from
Kubernetes Metrics Server") or, without metrics, the reason in place of the chart. **Every chart has
its own row of controls** (`chartCtl`; there is no main button): the pod's or node's chart, and the
compact chart under each container's section title. A row is a time range (5m, 15m, 30m, 1h with the
Metrics Server; 1h at first; longer with Prometheus, and held by the engine since the third slice,
below), the variant buttons as icons with their names as tooltips (CPU a chip, Memory a memory
module, Network arrows both ways, Filesystem a stack of drives: Cogpit's own drawings; Network and
Filesystem disabled with the tooltip "needs Prometheus" unless the provider has them) and a flag button that marks the object's events from the drawer model. Range, variant
and flag are per chart, so changing one chart changes only that chart; the container's row is
smaller (a "1h" button that opens a menu of the ranges, no divider),
so it stays on one line at the drawer's minimum width. The data stays one poll per drawer: the
panel's `ObjectMetrics` feeds every chart, each reading its target by name and drawing it in its own
variant, with that container's requests and limits (`TestEveryChartHasItsOwnControls`). The
`chartCtl`s live in the panel (`perBox` for the containers), which `drawer.kept` keeps across
rebuilds, so a selection survives watch events and polls; a container that appears later gets a new
`chartCtl` with the defaults (CPU, 1h, events off: `TestLateContainerStartsWithCPU`). A selection is
not kept when the drawer shows another object or is closed. The chart widget takes plain series, dashed reference lines and markers and
draws them with canvas lines in theme tokens: usage in the accent colour, requests and allocatable
capacity in the warning colour, limits and capacity in the error colour, events in the pending
yellow; its width is never more than the drawer's.

**Hovering a chart** (2026-10-12, owner: "hover and get information about the exact time and value";
[`chart_hover.go`](../../internal/ui/fyneui/chart_hover.go)). Over the plot, a vertical crosshair stands
on the data point nearest to the pointer, a dot sits on each series at that point, and a panel inside
the chart (not the shell's delayed tooltip, so it needs no shell, has no delay and flips to the
crosshair's other side before it leaves the chart, which keeps it readable at the drawer's 320-unit
minimum) lists the local time with seconds and "N s ago", each series' value and the dashed lines'
values (request and limit, or allocatable capacity and capacity). The values use `chartData.Detail`
(CPU in millicores below one core, memory in binary units with one decimal; the axis's `Format` when
unset); within 5 units of an event marker the panel also names the event (reason, count, time, the
message cut to one line), taken from the marker's own `Reason`, `Message` and `Count`. The chart
implements only `desktop.Hoverable`, so clicks and scrolls pass through; `MouseOut`, `Hide` and no
data clear the panel. The pointer is kept across `Set`: a new poll recomputes the crosshair at the
same x, and only the overlay's objects are replaced, never the chart's own, so nothing flickers
(`TestHoverSnapsToTheNearestPoint`, `TestHoverNamesTheEventNearAMarker`,
`TestHoverOutHidesAndDataRefreshKeepsTheCrosshair`, `TestHoverNoDataNoPanelAndClicksPassThrough`,
`TestDrawerChartsHoverPerChartAndVariant`). The chart still knows nothing of its data's source, so the
Prometheus provider's series hover the same way.

*Limits worth knowing.* The Metrics Server keeps no history: the chart starts when the drawer opens
(the hub seeds it with the list's last measurement) and the panel says so. The series is per
drawer, not per object: closing the drawer drops it. The Pods and Nodes lists fit a 1,440-pixel window
with the new columns only because the default widths of several columns were trimmed
(`defaultColumnWidths`); a user's dragged widths still win.

**Built 2026-10-12 (G12, second slice, backend): Prometheus through the API server's service
proxy.** The Provider/Detector seam of the first slice took it without a change of shape; the window
draws it since the third slice (below).

| Piece | File | Responsibility |
|---|---|---|
| `Prometheus` | [`prom.go`](../../internal/metrics/prom.go), [`promrange.go`](../../internal/metrics/promrange.go) | The provider, `Capabilities{History, Network, Filesystem}` all true, label `Prometheus (<namespace>/<service>)`. Requests are client-go's `Services(ns).ProxyGet` on the connection's own clients: counted, rate limited (5 per second, burst 10 for lists, gets and the proxy), authenticated as the user. Only `GET`. `ListPods`, `GetPod`, `ListNodes`, `GetNode` are instant queries (the lists, `metrics.get`); `ObjectRange` (interface `RangeProvider`) is the drawer's history |
| PromQL | [`promql.go`](../../internal/metrics/promql.go) | Every query is built here. Pod, namespace and node names are user data and go through `quoteLabel` (a double-quoted literal with `\\`, `\"`, newlines and control characters escaped; `FuzzQuoteLabel` checks that it reads back as the name), node names also through `regexp.QuoteMeta` for the instance regular expression |
| `PrometheusDetector`, `Candidates`, `Chain` | [`promdetect.go`](../../internal/metrics/promdetect.go), [`chain.go`](../../internal/metrics/chain.go) | Detection and the chain "Prometheus if found, else Metrics Server, else not served"; since the fourth slice the engine holds both providers apart (below) and the chain serves a hub over one detector only |
| `Hub` additions | [`hub.go`](../../internal/metrics/hub.go) | Range queries for watched objects, `Subscription.SetWindows`, `Hub.History`, `Hub.Capabilities`, `Hub.Forget` |

*Detection.* The engine hands the detector the cluster's **cached Services** (`Backend.promServices`:
the Services kind's informer, shared with the Services view, no list of its own; it waits up to 3 s
for the access reviews and does not watch Services for a user known not to be allowed to list them,
and up to 8 s for the informer's first list). Candidates are picked by name only, so no unrelated
Service is probed (clean-room, from the charts' and Lens's documented defaults):

| Score | Service name | From |
|---|---|---|
| 100 | `<release>-kube-prometheus-prometheus`, `<release>-kube-prometheus-stack-prometheus` (a release of up to five characters), `kube-prometheus-stack-prometheus` | kube-prometheus-stack |
| 95 | `prometheus-k8s` | kube-prometheus manifests |
| 90 | `prometheus-operated` | Prometheus Operator's governing service |
| 80 | `prometheus-server` | the `prometheus` Helm chart |
| 70 | `prometheus` in `monitoring`, `prometheus` or `lens-metrics` | Lens's own install, hand-written installs |
| 40 | `prometheus` in any other namespace | |

Ties go to those namespaces in that order, then by name. The port is the one named `http-web`, `web`,
`http`, `http-prometheus`, `prometheus` or `tcp-prometheus` (in that order; ports with `reloader`,
`grpc` or `thanos` in their name are skipped), else 9090, else 80, else the first port; a named
port is passed by name, which the service proxy resolves. At most `MaxProbes` (3) candidates are
probed, best first, each with one `GET /api/v1/status/buildinfo` through
`/api/v1/namespaces/<ns>/services/<name>:<port>/proxy/`; a 200 whose body is Prometheus's
`{"status":"success","data":{"version":…}}` wins. A 403 or 401 stops the search with the state
**forbidden** ("may not use the service proxy of …, get services/proxy"). A 404 or a body that is not
buildinfo moves on to the next candidate and counts as not found. Anything else (a 503 for a Service
without endpoints, a timeout) moves on too, but if nothing answers the state is **unavailable** and
names the candidate. No candidate is **notServed** and costs no request. The hub keeps the answer like
the Metrics Server's: 10 minutes when found, 2 minutes otherwise. **Override:**
`PrometheusDetector.SetOverride(&metrics.Override{Namespace, Service, Port, Prefix})` (Port a number or
a port name, Prefix the route prefix); with an override only that service is probed, no Service list
is read, and `Backend.PrometheusDetector` plus `Hub.Forget()` apply it at once. Since the fourth
slice the user sets it (`cluster.setPrometheus`, a dialog from the drawer's status line; below). The chain (`metrics.ChainForConn`) returns the first provider
found; if none, the most telling state (forbidden, then unavailable, then notServed), so a Prometheus
the user may not read is reported instead of "no metrics-server". (Fourth slice: the engine holds both
providers apart and looks for Prometheus only when a drawer opens or the Metrics Server fails; the
most telling state is the hub's when neither answers.)

*Queries* (`rate` over `[R]`; `C` = `container!="",container!="POD"`, which leaves out the pod's own
cgroup and the pause container; `P` = `namespace="<ns>",pod="<pod>"`, each value quoted). **The rate
window `R`** is 4 minutes for the instant queries (the lists, `metrics.get`) and `max(4m, step + 1m)` for
a chart's range queries, so that every sample between two points counts (a week's chart, at a step of
about ten minutes, rates over 665 s and shows a spike between two of its points). Four minutes is
Prometheus's guidance of four scrape intervals for the `prometheus` chart's one-minute scrape, and
measured (2026-10-12, kind with that chart): cAdvisor stamps each sample with its own collection time,
up to a minute before the scrape, and with `[2m]` a half-hour chart lost 5 (CPU) to 7 (network) of
its 52 points and its newest one or two minutes; with `[4m]` none (`rateWindow`, `rangeRate` in
[`promql.go`](../../internal/metrics/promql.go)). The price is smoothing: a 5-minute chart shows
4-minute averages.

| Variant | Pod chart and container charts | Node chart |
|---|---|---|
| CPU | `sum by (container) (rate(container_cpu_usage_seconds_total{P,C}[R]))`; the pod's line is the sum of its containers | `sum(rate(container_cpu_usage_seconds_total{id="/",node="<n>"}[R])) or sum(rate(…{id="/",instance=~"<n>(:[0-9]+)?"}[R]))` |
| Memory | `sum by (container) (container_memory_working_set_bytes{P,C})` | the same on the root cgroup |
| Network | `sum(rate(container_network_receive_bytes_total{P}[R]))` and `…transmit…`: per pod (the network belongs to the pod's network namespace); a container's target carries the pod's samples and `NetworkOfPod` | the same on the root cgroup (`id="/"`, summed over interfaces) |
| Filesystem | `sum by (container) (container_fs_usage_bytes{P,C})`; the pod's line is the sum. **Discovered by trying, per kind:** if that has no series, `rate(container_fs_reads_bytes_total…)` and `…writes…` (`FilesystemSource` is `"usage"` or `"io"`; the provider remembers the answer for pods and for nodes apart, for as long as the detection lives, 10 minutes). The kinds differ in practice: on containerd the kubelet's cAdvisor reports the usage for the node's root cgroup but not for containers (kind, 2026-10-12), and one answer for both had left every pod's Filesystem chart empty after a node's drawer | the usage of the node's **block devices** only, `device=~"/dev/.+",device!="/dev/shm"` on the root cgroup: cAdvisor lists every mount there and the containers' overlays repeat the disk (kind: nine overlays of `/dev/vda1` summed to 503 GiB on a 56 GiB disk); the read and write rates on the root cgroup |
| Lists | `sum by (namespace, pod, container) (rate(…cpu…{C}[4m]))` and the memory gauge, one instant query each, optionally `namespace="<ns>"` | `sum by (node, instance) (…{id="/"})` for CPU and memory; the name is the `node` label, else the `instance` without its port |

Nodes use **cAdvisor's root cgroup, not node-exporter**: node-exporter series are labelled with the
exporter's own address, which has no reliable link to the node's name, while cAdvisor's carry the
`node` label (kube-prometheus-stack) or the node's name as the instance (the `prometheus` chart's
cAdvisor job). A Prometheus that labels neither shows empty node charts and columns; this is
**not verified against a real Prometheus**, only against the fake HTTP API in the tests.

*Ranges and polling.* A drawer on a provider with history does not use the hub's own series. Its
poll is one range query set: `ObjectRange` with `from = alignDown(now − window, step)`, `to = now`,
`step = max(15 s, shortest window / 120, longest window / 1000)` rounded up to whole seconds; the
window is the longest any of the drawer's charts shows, and the default hour is always among them, so
a chart nobody set has data (1 h alone: 30 s steps, 120 points). The grid is aligned to the epoch, so
successive polls give the same points. The poll interval is the step, so there is one new point per
poll. A pod costs **five queries per poll, issued at once**: CPU, memory, receive, transmit and the
filesystem usage, or six with the read and write rates instead (seven while the filesystem metric is
still being discovered, once per kind, or neither exists).
Only the CPU and memory queries must succeed; a failing network or filesystem query leaves its
series out (debug log). One poll serves every chart and every consumer of the scope; the Pods and
Nodes lists keep their 60 s and the instant queries (two per poll). The hub stores the latest
point as the pod's current usage, so the lists, `Hub.Pod` and the drawer's `Current` need nothing
else. `Subscription.SetWindows` (from `Session.SetMetricsRange`) wakes the poll early when the
ranges change. **Superseded by the fourth slice** (below): the history is asked for once, the live
values come from the Metrics Server, the network and the filesystem only while a chart shows them,
and the grid, steps and rate windows above apply to each of those queries.

Bounds: a response is cut at 16 MiB (`DefaultMaxResponseBytes`; larger is the state unavailable), a
query at 15 s, a poll at 20 s, a series at 11,000 points (Prometheus's own limit), range steps keep
the longest window within 1,000 points. NaN and infinite samples are dropped. A Prometheus failure
is a state with its own text (`StateError`): 403 forbidden, 404 notServed (and the next poll detects
again), 503 and timeouts unavailable. Query strings carry label values only; no credential is ever
part of a query or a log line (the credentials travel in the transport's headers, which the
logger never sees).

*In the engine.* `Session.ObjectMetrics(ref)` returns, with Prometheus, the targets' `Samples`
(CPU and memory, the old field), `Network` and `Filesystem` cut to each chart's range,
`ObjectMetrics.FilesystemSource`, the capabilities of the detected provider (so the window can
enable the two buttons) and the source line `Prometheus (<namespace>/<service>)`. A target's range
defaults to the hub's hour; `Session.SetMetricsRange(ref, target, window)` (target "" for the pod's
or node's chart, or a container's name; 1 minute to 30 days; 0 for the default) sets one chart's,
wakes the poll and emits `EventMetrics`; it applies to the open drawer only and `ErrDrawerNotOpen`
refuses it otherwise.

**A chart's range is UI state the engine holds** (2026-10-12, rule 2): the drawer's `drawerWatch`
keeps it per chart, every `MetricsTarget` reports it (`Window`, and `Range` as text: `"1h"`, `"7d"`;
the default for a chart nobody set), and the command **`ui.setMetricsRange {ref, container?, range?}`**
([`commands/metricsrange.go`](../../internal/engine/commands/metricsrange.go); `DangerUI`, no verbs,
it sends nothing itself) sets it for the GUI and agents alike: `range` is a Go duration or whole days
(`15m`, `6h`, `7d`; `ParseChartRange`, `FormatChartRange`), `default` or empty puts the chart back to
the hour; a container the drawer's object does not have, a kind without charts and a range outside
1 minute to 30 days are refused, a drawer showing another object is `ErrDrawerNotOpen`. It returns
every chart's range, and **`ui.state` lists them as `drawerCharts`** while a pod's or node's drawer is
open, so an agent sees what the user sees (`TestMetricsRangeCommand`; with Prometheus behind the
fake service proxy, `TestPrometheusInTheDrawer` checks the targets' `Range`). `metrics.get` serves the instant values from whichever provider is active
(`verbs` now include `get services/proxy`). **Detection puts a watch on Services** into the cache
for as long as a metrics view is open (and the warm period after): the Pods view that opens at
connect is such a view, so a connection holds one more watch than before (`TestBudgetCustomResourceView`
counts four). Since the fourth slice `metrics.get` and the lists ask the Metrics Server first, and
Prometheus is looked for (and Services watched) only when a drawer opens or the Metrics Server
fails or is missing; `TestBudgetCustomResourceView`'s fake API server has no Metrics API, so it
still counts four.

**Built 2026-10-12 (G12, third slice): the window draws Prometheus**
([`metricsview.go`](../../internal/ui/fyneui/metricsview.go), [`chart.go`](../../internal/ui/fyneui/chart.go),
[`chart_hover.go`](../../internal/ui/fyneui/chart_hover.go)). The chart widget did not change shape:
it still draws the plain series, lines and markers it is given; what changed is what the panel gives
it per variant.

| Variant | Series (hover names) | Lines | Units: axis, hover |
|---|---|---|---|
| CPU | "CPU usage" | requests and limits (a node: allocatable capacity and capacity) | cores: `0.25`; `250m`, `1.25` |
| Memory | "Memory usage" | the same | binary bytes: `192Mi`; `192.4Mi` |
| Network | "Receive" (accent), "Transmit" (`colorChartSecond`) | none | bytes per second: `48KiB/s`; `30.6KiB/s` |
| Filesystem, source `usage` | "Usage" | none | binary bytes, as Memory |
| Filesystem, source `io` | "Read" (accent), "Write" (`colorChartSecond`) | none | bytes per second, as Network |

- **The buttons follow the provider** (`chartCtl.follow`, `enableVariants`): Network and Filesystem
  are enabled from `ObjectMetrics.Capabilities` while the status is available, else disabled with the
  tooltip "Network: needs Prometheus"; the buttons stay icons, the names are their tooltips (owner).
  A chart showing a variant the provider no longer has goes back to CPU; a failed poll (a moment
  without metrics) keeps the choice.
- **A container's Network is the pod's** (`NetworkOfPod`): its chart says so in a line under the plot,
  "Network is per pod: its containers share it." Lens has no network tab on a container (capture);
  Cogpit shows the pod's because the owner asked for every variant on every chart. A compact chart
  draws its legend when it has two series, which only the legend tells apart.
- **Byte axes step in their unit** (`chartData.Binary`, `binaryStep`): 25KiB/s, 50KiB/s, not 24.4KiB/s;
  the Memory axis follows too. **Time axes** carry the date from a range of a day ("Oct 11 14:30") and
  the date alone from a week ("Oct 4"); the hover's "ago" counts days ("2 d 2 h ago").
- **Colours from tokens only**: the second series is the new `colorChartSecond` (Solarized violet,
  lightened to 4:1 on the plot in the dark theme, darkened to 3.9:1 in the light one); both series
  colours keep 3:1 on the plot and the hover panel (`TestPaletteContrast`) and a CIE76 difference of
  at least 40 from each other and from the requests', limits' and events' colours in both themes
  (`TestChartColoursAreDistinct`).
- **Ranges** (`metricsRanges`): 5m, 15m, 30m, 1h, 3h, 6h, 12h, 1d, 2d, 7d, 14d, 30d with a provider
  that has history, up to 1h with the Metrics Server, which keeps none. Lens's capture shows the
  default "1h" only; the other values are Cogpit's choice (inferred). The pod's chart offers them in a
  select, a container's in its small button's menu. **Choosing one asks the engine**
  (`ui.setMetricsRange`): the chart shows the new range at once, and until the engine answered a
  load in flight does not put the old one back (`chartCtl.pending`); then the range control follows
  what the engine reports, whoever set it, so an agent's change shows in the window. A drawer that
  shows another object by then (`ErrDrawerNotOpen`) is refused quietly, any other refusal is a toast,
  and either way the chart follows the engine's range again (`TestMetricsRangeIsEngineState`).
- **The source line** reads "Displaying metrics from Prometheus (<namespace>/<service>)"; the note
  "The Metrics Server keeps no history" shows only for a provider without history
  (`TestPrometheusSourceAndVariants`).
- **Hover** needs nothing new: each series is named in the panel with the variant's hover format
  (`TestHoverNetworkAndFilesystem`).

Tests on canned data: `TestPrometheusSourceAndVariants`, `TestPrometheusFilesystemIO`,
`TestVariantsFollowTheProvider`, `TestMetricsRangeIsEngineState`, `TestHoverNetworkAndFilesystem`,
`TestRateFormatsAndLongRanges`; renders of each variant in both themes, a container's network, a
week and the hover at the narrowest drawer (`TestRenderPrometheusMetrics`, to `COGPIT_PNG_DIR`).
Against a real Prometheus: below.

**Checked against a real Prometheus on 2026-10-12** (`make kind-prometheus`: the prometheus-community
`prometheus` chart 29.36.1, Prometheus 3.15.0, the server alone, scraping the kubelet's cAdvisor every
minute, on the kind cluster: Kubernetes 1.36.1, containerd 2.3.1, one node;
[`prometheus_kind_test.go`](../../internal/testkit/integration/prometheus_kind_test.go),
[`metricsview_kind_test.go`](../../internal/ui/fyneui/metricsview_kind_test.go), both skipped where the
cluster has no Prometheus, so not in CI). What held, with the numbers seen:

| Checked | Seen |
|---|---|
| Detection | `monitoring/prometheus-server` among the cached Services, port `http`, buildinfo 3.15.0; source line "Prometheus (monitoring/prometheus-server)" |
| Node drawer (`cogpit-control-plane`) | CPU 219–225m against the Metrics Server's 206m at about the same time; memory 1,600,798,720 bytes, the Metrics Server's 1,563,280Ki to the byte; network 0.1 to 4 KiB/s; filesystem `usage` 55.9 GiB, the node's disk (after the fix below) |
| Pod drawer (`cogpit-demo/web-…`, nginx) | memory 15.8MiB, the Metrics Server's 16,136Ki to the byte; CPU 0 at rest (0.1 millicores), 13–17m under a load of about 435 requests per second over the three web pods, network then 52–65 KiB/s received and 76–95 KiB/s sent; the container's chart carried the pod's network (`NetworkOfPod`); filesystem `io`, 0 bytes per second (nginx writes nothing) |
| The window | the source line, Network and Filesystem enabled, the ranges to 30d offered, 54 to 56 points per series on the default hour, the hover "Receive: 41.8KiB/s, Transmit: 60.6KiB/s" over the network chart |
| `ui.setMetricsRange`, `ui.state` | a pod chart at 6h, the container's at 1h, both listed in `drawerCharts` |
| `metrics.get` | the six seed pods from Prometheus (memory 10.7 to 19.4MiB), the node at 222m and 1.5GiB |
| Budget P13 | 16 proxy GETs in 90 s with the Pods list and a pod's drawer at 1h open: two query sets of six (the drawer polled about every 33 s: the 30 s step after the poll's own time) and two list polls of two; no write |

It found three faults the fake did not show, fixed the same day: the filesystem metric discovered once
for both kinds (a node's `usage` left every pod's chart empty, since containerd's cAdvisor reports
usage for the root cgroup only), a node's usage summed over every mount (503 GiB for a 56 GiB disk),
and `rate(…[2m])` leaving gaps at a one-minute scrape (the queries above).

**Not checked against a real Prometheus**, only against the fake: kube-prometheus-stack (the `node`
label, a 30 s scrape), `container_fs_usage_bytes` for containers (runtimes that report it), a user
refused the service proxy, a route prefix, the override, Thanos and VictoriaMetrics, and history
longer than the hour the kind Prometheus had (the 7d and 30d charts drew it at their right edge).
(Fourth slice: kube-prometheus-stack, its 30 s scrape and the override with a route prefix (against
the fake service proxy) are checked since, below; the `node` label is not, since the chart drops the
node series.)
Seen once and not again in four runs: right after connecting, a Nodes list poll whose instant
query had memory but no CPU, which the list showed as 0%: fixed in the fourth slice (a part
without a series is *missing*, below).

**Built 2026-10-12 (G12, fourth slice): both providers together, the status line** (the owner's
requirements of the evening). The chain of the second slice (Prometheus if found, else the Metrics
Server) is gone from the engine: the hub holds **both providers at once**, each with its own
detection, state, last good answer and last failure, and gives each its part.

| Piece | File | Responsibility |
|---|---|---|
| `NewHubFor(Sources{Live, History})`, the slots | [`hub.go`](../../internal/metrics/hub.go) | One slot per provider: the Metrics Server (`Live`) and Prometheus (`History`), from `metrics.SourcesForConn` in the engine's connector. `NewHub(det)` stays for one detector (the demo cluster's fixed provider, the chain in tests) |
| The drawer's poll | [`hubobject.go`](../../internal/metrics/hubobject.go) | History once, then live values; the network and the filesystem while shown; the fallback; the gap |
| The chart | [`chart.go`](../../internal/metrics/chart.go) | An object's series while a drawer shows it: history and live values merged per target, spliced by time, bounded |
| `ProviderStatus`, error classes | [`status.go`](../../internal/metrics/status.go) | The status line's data (`Hub.Providers`); a failure as a class (`errorClass`), never a response body |
| `RangeQuery.Want` | [`provider.go`](../../internal/metrics/provider.go), [`promrange.go`](../../internal/metrics/promrange.go) | Which series a range query asks for: CPU and memory, the network, the filesystem |

*Who serves what.* The history role goes to the first provider that answers range queries
(Prometheus), the live role to the first one without (the Metrics Server), else to the history
provider. Prometheus is **looked for only when it is needed**: when a drawer opens (its history) or
when the Metrics Server fails or is missing (the fallback). A Pods or Nodes list on a cluster whose
Metrics Server answers sends Prometheus nothing and puts no watch on Services
(`TestListsUseTheMetricsServerAndFallBack`, `TestBothProvidersBudget`).

*A drawer's poll* ([`hubobject.go`](../../internal/metrics/hubobject.go)), every 15 s
(`ObjectInterval`), or at the range's step while Prometheus serves the live values:

| What | When | Requests |
|---|---|---|
| History: CPU and memory over the longest chart range, at the step of the shortest | the drawer opens; a range changes (window or step); Prometheus answers again after it had not given the history; `Hub.Forget` | 2 range queries, once |
| Live values: the Metrics Server's measurement of the object | every poll | 1 `get pods.metrics.k8s.io` (or nodes) |
| Network, filesystem: over the chart range, then from their last point on | a chart shows them (`ui.setMetricsVariant`, `Subscription.SetWants`): at once, then once per step (30 s at 1 h); a series no chart shows any more is dropped and asked for over the range when shown again | 2 (receive, transmit) and 1 (usage) or 2 (read, write) range queries |
| Live values from Prometheus: CPU and memory from the chart's last point on | the Metrics Server failed in this poll (at once, the same poll), then once per step while it is left alone | 2 range queries |
| The gap: CPU and memory between two points more than two resolutions and a step apart (a live value after both providers were away) | once, when Prometheus answers | 2 range queries |

Asking for the network and the filesystem only while a chart shows them is **the lead's proposal,
not the owner's decision** (G12): the alternative is asking for them with the history at every
open, three or four more queries per drawer.

*The seam.* The chart keeps one point per time per target, oldest first: a range query's answer
replaces what the chart held from its start on (`splice`), a live value is added only when it is
newer than the target's last point, so the history's last point and the first measurement never
make a point twice. A live value more than `2 × 15 s + step` after the last point records a gap,
which Prometheus fills once (`TestGapIsFilledOnceFromPrometheus`). A series keeps what lies within its
longest chart range back from its own newest point (at least an hour; the cluster's clock, not the
desktop's, decides) and at most `DefaultSeriesPoints`, 5,760 (a day of 15 s measurements), the
oldest dropped first, so a long-open drawer loses its left edge, never a middle.

*Falling back* (owner): a provider that fails a request while in use gets the failure's state
(unavailable, forbidden; a 404 makes detection run again) and is left alone for `ProviderRetry`
(**one minute, Cogpit's default**, not the owner's) while the other serves its part; then it is
asked again and takes its part back when it answers. A provider nothing can stand in for is asked
at every poll. The Metrics Server down: Prometheus gives the live values in the same poll
(`TestMetricsServerFailsPrometheusTakesOver`), and the lists take Prometheus's instant queries.
Prometheus down: the Metrics Server's live values go on, the chart keeps the history it has, and
the history (or the network) is asked again once Prometheus answers
(`TestPrometheusGoneLeavesTheLiveValues`). Neither: the most telling state (forbidden, then
unavailable, then not served), as the chain had it. A provider that is found but fails keeps its
chart variants (the buttons stay enabled, the status line says why nothing moves); one that is not
found any more takes them away.

*The status line* (`Hub.Providers`, `ObjectMetrics.Providers`, `metrics.get`'s `providers`,
`ui.state`'s `drawerMetricsProviders`): per provider, Prometheus first, its id and label, the
service where Prometheus was found (`monitoring/prometheus-server:http`, the port by name or
number, with the route prefix), its status (`available`, `notServed`, `forbidden`, `unavailable`,
`unknown`: not looked for yet), the last good answer, the last error with its time, and what it
serves now (`history`, `live`, `network`, `filesystem`). The last error is a class and a short
reason (`unavailable (HTTP 503)`, `forbidden (HTTP 403)`, `timeout`, `unreadable answer`;
`errorClass`); the states' messages quote the API server's own Status text only, never a body the
proxied service or the metrics-server sent (client-go quotes such bodies in its errors; `errorText`
and Prometheus's `parseResult` leave them out: `TestErrorClassesCarryNoBodies`). A provider that
nothing asked since it last answered (Prometheus while only CPU and Memory are shown) shows that
answer's state and age: Cogpit sends no probe to keep the line current.

*The window* ([`metricsstatus.go`](../../internal/ui/fyneui/metricsstatus.go)): under the source
line, one row per provider: an icon in a status token (a check in `status.ok`, a warning in
`status.warning`, an error in `status.error`, an info or a question mark in `muted`), then
"Prometheus monitoring/prometheus-server:http: history, network, filesystem · 40 s ago" or
"Kubernetes Metrics Server: not answering, unavailable (HTTP 503) · last answer 3 min ago", cut to
the drawer's width; the tooltip over a row has the whole state, with the times
(`TestProviderStatusLine`, `TestProviderLineStates`, `TestRenderMetricsStatusLine`). The wording and
the icons are Cogpit's (inferred); Lens shows a source line only. The source line names both:
"Displaying metrics from Prometheus (monitoring/prometheus-server) and Kubernetes Metrics Server".

*A chart's variant is engine state* (rule 2): `ui.setMetricsVariant {ref, container?, variant}`
([`commands/metricsvariant.go`](../../internal/engine/commands/metricsvariant.go); `DangerUI`, no
verbs) sets what a chart of the open drawer shows (`cpu`, `memory`, `network`, `filesystem`; the
last two refused without Prometheus), `MetricsTarget.Variant` and `ui.state`'s `drawerCharts` report
it, and the window's variant buttons dispatch it and follow it (`TestMetricsVariantIsEngineState`).
The variants decide what the hub asks Prometheus for (`Session.SetMetricsVariant` →
`Subscription.SetWants`).

*The cluster's Prometheus, set by the user* ([`engine/metricssettings.go`](../../internal/engine/metricssettings.go),
[`commands/metricsprometheus.go`](../../internal/engine/commands/metricsprometheus.go),
[`fyneui/metricssettings.go`](../../internal/ui/fyneui/metricssettings.go)). `cluster.setPrometheus
{address, prefix?}` takes Lens's form of the address, `namespace/service:port` (the port by number
or name), and a route prefix, or `auto`; `CheckPrometheusOverride` refuses what is not one (the
namespace and service DNS labels, a port 1 to 65535 or a port name, a prefix of letters, digits,
`-`, `_`, `.` and `/` without `.` or `..` segments, so it can only name a path on that service
behind the proxy). The engine sets the override on the detector, calls `Hub.Forget` (an open drawer
asks the named service for its history at once) and keeps it per context in Cogpit's own
preferences, `<user config dir>/Cogpit/metrics.json` (0600, atomic writes; never the kubeconfig);
the connector sets it before anything is detected on the next connection, and a file entry that is
not a service address is left out when read (`TestMetricsSettingsKeepTheOverride`,
`TestMetricsSettingsReadOnlyValidEntries`, `TestPrometheusOverrideFromTheSettings`: the named
`observability/thanos-query:9090/prometheus` served the drawer, the service found by name got no
request on the next connection, `auto` brought detection back). `cluster.prometheus` reads it. It is
`DangerUI` and open to agents (Cogpit's default: it writes no cluster object and the user's own
preferences only; the owner may want it GUI-only, H-26). The GUI: a "Prometheus…" button under
the status line, shown when the cluster has Prometheus detection (not the demo cluster), opens
"Prometheus for <context>" with "Auto detect Prometheus", the service address and the path prefix
(`TestPrometheusSettingsDialog`, `TestRenderPrometheusSettings`); Lens's cluster settings page is not
built, so this placement is Cogpit's.

*Missing, not 0.* `model.Usage.Missing` names the parts a provider had no value for: a Prometheus
whose CPU rate has no series for a node or container it has memory for (a counter needs two samples
in the rate window), and a range answer's time that only one of CPU and memory has (`mergeUsage`).
The Pods and Nodes lists show such a part as an empty cell, sorted last
(`TestUsageCellsLeaveAMissingPartEmpty`), a pod's total lacks it only when every container does, and
a chart leaves the point out of that variant (`TestMissingPartIsLeftOutOfTheChart`). The Metrics
Server always measures both.

**Checked on kind on 2026-10-12** (Kubernetes 1.36.1, the cluster's metrics-server and the
`prometheus` chart of the third slice; `TestKindPrometheus`, `TestKindPrometheusDrawer`, and
`TestKindMetricsFallback`, which scales both deployments to zero and back and restores them, run only
with `COGPIT_KIND_FALLBACK=1`):

| Checked | Seen |
|---|---|
| The Pods list alone | the Metrics Server's lists, 0 Prometheus requests, no watch on Services |
| A node's and a pod's drawer | source "Prometheus (monitoring/prometheus-server) and Kubernetes Metrics Server"; the hour's history from Prometheus (121 points for the node, 120 for the pod at 30 s), the Metrics Server's measurements after it, one point per time, no step longer than 30 s at the seam; the network and the filesystem only once a chart showed them; the status line "Prometheus monitoring/prometheus-server:http: history, network, filesystem" and "Kubernetes Metrics Server: live values" |
| Budgets, 90 s with the Pods list and a pod's drawer at 1 h | CPU and Memory shown: 0 Prometheus requests, 6 Metrics Server gets, 2 lists (before this slice: 16 Prometheus requests); the network shown: 6 Prometheus requests (2 at once, 2 per 30 s step), 6 gets, 1 list |
| The Metrics Server scaled to zero | seen at the drawer's next poll (15 s), which took Prometheus's newest points in the same poll; the chart went on at 30 s; `metrics.get nodes` answered from Prometheus; the status line "Kubernetes Metrics Server: not answering, unavailable (HTTP 503)" |
| The Metrics Server back | serving the live values again 9 s after its pod was ready (the test's `ProviderRetry` of 20 s; a minute by default) |
| Prometheus scaled to zero | seen at the next request it got (a range change to 3 h): "unavailable (HTTP 503)"; the Metrics Server's live values went on |
| Prometheus back | the 3 h history asked for again and served 28 s after its pod was ready; no step longer than 30 s in the chart's last 15 minutes (the kind Prometheus's own history has an 11-minute hole from a Docker restart) |

**Checked against a real kube-prometheus-stack on 2026-10-12** (chart 92.3.0, operator v0.94.1, the
operator and one Prometheus only, scraping the kubelet's cAdvisor every 30 s, release `kps` in its
own namespace `cogpit-kps`: `make kind-kps`, `TestKindKubePrometheusStack`, then `make
kind-kps-down`, which also removes the operator's CRDs and its kubelet Service in `kube-system`):

| Checked | Seen |
|---|---|
| Detection | `cogpit-kps/kps-kube-prometheus-stack-prometheus:http-web`, preferred to the `prometheus` chart's `prometheus-server`. Reading the chart's naming for this check found a gap, fixed: the chart cuts `<release>-kube-prometheus-stack` to 26 characters, so a release of up to five characters keeps `-stack` in the Service's name, which the name pattern had missed (`prometheus-operated`, score 90, would have been found instead) |
| A pod's drawer | history at the 30 s scrape from the Prometheus's start, CPU, memory, network (rx 101 B/s, tx 142 B/s for nginx at rest), filesystem as read and write rates (containerd), the Metrics Server's live values after it, no step longer than 30 s |
| A node's drawer | **no history from Prometheus**: the chart's default kubelet ServiceMonitor drops cAdvisor's series without a pod ("Drop cgroup metrics with no pod", `[id, pod]` matching `.+;`), and the root cgroup the node queries read is such a series. The node's chart has the Metrics Server's live values only (a reason for both providers together); node history on kube-prometheus-stack would need node-exporter's series, which Cogpit does not query (node-exporter's labels name its own address, not the node) |

**Built 2026-10-13 (G12, M7): the cluster's totals for the cluster Overview**
([`hubcluster.go`](../../internal/metrics/hubcluster.go), [`promcluster.go`](../../internal/metrics/promcluster.go),
[`engine/clusteroverview_charts.go`](../../internal/engine/clusteroverview_charts.go)). A third scope
kind, `KindCluster`, whose object is the cluster: the sum over its nodes. It is polled as a drawer's
object is, through the same loop, roles and fallback (no second polling path): `IsList` is false for
it, so `Subscribe` gives it a chart and `pollObject` plans its history and live values.

| What | Where | How |
|---|---|---|
| The scope | `metrics.ClusterScope(masters)` | `Scope{Kind: cluster, Name: "m1,m2"}`: the control-plane nodes, sorted and joined (node names have no comma). Which nodes are masters is the cache's to know (the Nodes view's roles: `control-plane` or `master`), so the engine names them, and another control plane is another scope with a history of its own |
| The chart's targets | `TargetMaster`, `TargetWorker` | `""` every node, `master` the named nodes, `worker` every other node |
| Live values | `clusterInstant`, `addClusterLocked` | one `list nodes.metrics.k8s.io` per poll (15 s, the object interval), summed per target at the newest measurement's time; a part is missing from a sum only when every node lacks it; a target without a node (no masters) gets no point. A fresh Nodes-list result seeds the chart (`seedClusterLocked`) |
| History | `Prometheus.clusterRange` | the sums over the nodes' root cgroups, the series the node charts read, so the history and the Metrics Server's node usage mean the same: `sum(rate(container_cpu_usage_seconds_total{id="/"}[R]))` and `sum(container_memory_working_set_bytes{id="/"})` for every node, the same with `node=~"(m1\|m2)"` or else `instance=~"(m1\|m2)(:[0-9]+)?"` for the masters (names through `regexp.QuoteMeta` and `quoteLabel`); the workers are every node less the masters at the times both have a point, so no query names the workers and a cluster of thousands of nodes sends a short query. Four range queries with masters, two without; the network and the filesystem do not exist for the cluster (asked for alone, nothing is sent) |
| The fallback | `pollObject` | unchanged: a Metrics Server that fails hands the live values to Prometheus in the same poll (the range query carries the masters in its `Name`), Prometheus gone leaves the live sums |

The engine subscribes the scope only while the page is open (`overviewCharts.follow`, once the nodes
are listed), sets its window to the page's range (`SetWindows`), and reads `Hub.History` for the
chosen group's target. CPU and Memory's requests and limits are the sums over the pods that have not
finished on the group's nodes (a limit counts where a container sets one, as kubectl's "Allocated
resources" does; Cogpit's, inferred), allocatable and capacity the nodes' sums, all from the cache.
**The Pods chart comes from the cache**: the pods that have not finished on the group's nodes,
against the nodes' `allocatable.pods` and `capacity.pods`, a point per poll or per 15 s and whenever
the count changes, kept while the page is open (the longest range, at most a day of points): no
history from before the page opened (Prometheus's pod counts are not asked: Cogpit's default).
`cluster.overview` without the page reads the current values from one `Hub.Fetch` of the nodes (a
result younger than 15 s is served without a request) and starts no poll.

**Checked on kind on 2026-10-13** (Kubernetes 1.36.1, one control-plane node, the cluster's
metrics-server and the `prometheus` chart of `make kind-prometheus`;
[`TestKindClusterOverview`](../../internal/testkit/integration/clusteroverview_kind_test.go), read-only,
skipped without that Prometheus):

| Checked | Seen |
|---|---|
| Groups | master 1 (`cogpit-control-plane`), worker 0: the charts show the masters (the default without workers) over 1h; source "Prometheus (monitoring/prometheus-server) and Kubernetes Metrics Server" |
| CPU | the hour's history from Prometheus, 120 points at 30 s (23:40:30–00:40:00), no step longer than 30 s at the seam; current 0.204 cores against metrics.get's node at 206m; requests 1.11, limits 0.1 (the few containers that set one), allocatable and capacity 20 cores |
| Memory | 2,577,469,440 bytes (metrics.get: 2.4GiB); requests 586MiB, limits 758MiB, allocatable and capacity 15.8GiB |
| Pods | 18 of 110 (allocatable and capacity), from the cache |
| Budget P14 | 5 Prometheus requests in the page's first 45 s (1 probe + 4 range queries: CPU and memory of every node and of the master), then none; 5 node lists in 45 s (one per 15 s poll); a range change to 3h: 4 requests, 333 points; closed (the Nodes view open): no Prometheus request and no node list in 20 s; no write |
| Issues | 3: the `cogpit-demo/crashloop` pod (status "Error; Containers not ready: app; 417 restarts"), its Deployment ("0 of 1 pods ready"), its BackOff event of the last hour; every issue's object resolves |

*On kube-prometheus-stack* the cluster has no history, as a node has none there (its kubelet
ServiceMonitor drops the root cgroup's series): the charts show the Metrics Server's live sums from
the page's opening. A Prometheus that scrapes cAdvisor twice (two jobs) would count every node
twice in the "every node" sum: not seen, not guarded.

## 10. Terminal (`internal/terminal`)

The terminal widget is `github.com/fyne-io/terminal`. Its `RunLocalShell` starts the shell with
the process's own environment (verified in its source), so a terminal-specific `KUBECONFIG` is not
possible that way. Cogpit starts the shell itself on a PTY (`creack/pty` on macOS and Linux, ConPTY
on Windows) with the per-terminal kubeconfig of
[security › credentials](../security/credentials.md#the-terminals-kubeconfig), and connects the
widget with `RunWithConnection`. Pod exec and attach streams from client-go connect the same way.
Terminal output is untrusted: escape sequences that write the clipboard or change window state are
filtered ([trust boundaries](../security/trust-boundaries.md)). The widget is a patched copy
([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md), [D-14](decisions.md#d-14-terminal-widget)):
it parses output on the goroutine that reads the connection and changes the screen, the
configuration, the selection and the character sets only on Fyne's main goroutine, through
`fyne.Do`; media copy, which changes how the following output is parsed, stays on the reading
goroutine. It is fuzzed on its own, without the filter (`FuzzHandleOutput`, `FuzzRawWidget`).

## 11. Updates (`internal/update`)

Cogpit checks the release channel, shows when an update is available, and installs it with one
click ([D-4](decisions.md#d-4-packaging-and-updates)): download, verify the signature and checksum,
replace or upgrade, relaunch. On macOS this is Sparkle 2, embedded in the app bundle and driven from
Go through a small cgo bridge; its appcast is the signed update manifest. Installs owned by a
package manager hand over to the manager.
Security perspective: [updates](../security/updates.md).

## 12. Runtime model

```text
 Fyne main goroutine ──Dispatch──► engine ──► cache, cluster, helm, metrics (goroutines)
        ▲                            │
        └──── fyne.Do(...) ◄──── events (ViewChanged, UIStateChanged, ConfirmRequested, …)
 MCP server goroutines ──Dispatch──► engine (same path, origin ExternalAgent)
 agent goroutine ───────Dispatch──► engine (origin BuiltInAgent)
```

- The UI goroutine never waits on the network.
- Fyne requires widget updates on its main goroutine; engine events reach the UI through
  `fyne.Do` (Fyne 2.6 and later). Once the shell stopped, what it queued there is dropped
  (2026-10-12): it would apply engine state to a window that is gone.
- Every goroutine that does cluster I/O recovers from a panic at its boundary and reports it
  as an error ([D-11](decisions.md#d-11-crash-stance)).
- Jobs take a `context.Context`; the UI can cancel any job with progress.

## 13. Testing and quality

Tiers, fixtures and budgets: [`testing.md`](testing.md).

## 14. Key dependencies (observed 2026-10-11, revised 2026-10-12)

Versions as the Go module proxy listed them; pinned when M0 creates `go.mod`. Revised 2026-10-12
after the dependency update that cleared Renovate's backlog ([releasing › Renovate](releasing.md#renovate)).

| Need | Module | Version | Note |
|---|---|---|---|
| UI toolkit | `fyne.io/fyne/v2` | v2.8.1 | decided after the spike gate ([D-1](decisions.md#d-1-ui-toolkit)) |
| Terminal widget | `github.com/fyne-io/terminal` | commit `46cd12724ff3` (2026-10-09), as a patched copy | untagged; its `go.mod` asks for an unreleased Fyne, its code builds on v2.8.1, and the copy's `go.mod` requires v2.8.1 ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)) |
| Packaging | `fyne.io/tools` | v1.7.3 | `fyne package` |
| Kubernetes API | `k8s.io/client-go`, `k8s.io/apimachinery`, `k8s.io/api` | v0.37.1 | Kubernetes 1.37 |
| kubectl behaviour | `k8s.io/kubectl` | v0.37.1 | drain, rollout, describe |
| Metrics API | `k8s.io/metrics` | v0.37.1 | |
| Helm | `helm.sh/helm/v4` | v4.3.0 | in-process ([D-3](decisions.md#d-3-helm-and-kubectl-binaries)); in `go.mod` since 2026-10-11, built against k8s.io v0.37.0 and resolved to v0.37.1 with the rest (it also brings `k8s.io/apiserver`, `apiextensions-apiserver` v0.37.1, `controller-runtime` v0.25.2 since 2026-10-12, and `oras-go` as indirect dependencies) |
| MCP | `github.com/modelcontextprotocol/go-sdk` | v1.8.0 | MCP specification 2026-07-28 |
| LLM client | `github.com/openai/openai-go/v3` | v3.76.0 | OpenAI-compatible endpoints |
| Updates (macOS) | Sparkle | 2.10.0 | embedded framework, EdDSA-signed appcast ([D-4](decisions.md#d-4-packaging-and-updates)) |
| Updates (Windows, Linux) | `github.com/creativeprojects/go-selfupdate` | v1.6.0 | candidate for M11 |
| PTY (macOS, Linux) | `github.com/creack/pty` | v1.1.24 | |
| YAML | `sigs.k8s.io/yaml` | v1.6.0 | the Kubernetes YAML library |

Toolchain: the team's repositories declare `go 1.27.2`; this machine had Go 1.26.5 on 2026-10-11.

## 15. Open decisions

D-4 for Windows and Linux, decided with M11: see [`decisions.md`](decisions.md).

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-13 | minor | §4.3: the notification log (every toast, the engine's and the window's own, 100 per session, `UIState.Notifications`, the `notifications.*` commands, window-only texts withheld from agents); Current state: the title bar's history arrows, bell and context menu (G5) |
| 2026-10-13 | minor | §4.3 the welcome page as engine state (`UIState.Welcome`, derived by `syncWelcome`, `ui.welcome.open`), the window's page under the tab bar, and Disconnect (`cluster.disconnect`); §6 a cluster's link and Remove from Cogpit (`cluster.links`, `cluster.remove`, GUI only); Current state: the welcome page, the Help and Window menus, 132 commands (G5, G13, night run) |
| 2026-10-13 | minor | §4.3 the welcome page as engine state (`UIState.Welcome`, derived by `syncWelcome`, `ui.welcome.open`) and Disconnect (`cluster.disconnect`); §6 a cluster's link and Remove from Cogpit (`cluster.links`, `cluster.remove`, GUI only) (G5, G13, night run) |
| 2026-10-13 | minor | §4.4: a jump to a time in a logs tab, read by the engine, searched in the held lines first; Current state |
| 2026-10-13 | minor | §4.4: older lines of a logs tab, engine state carried out by the window, asked as a longer tail and put on top without repeating a line |
| 2026-10-13 | minor | §4.4: a logs tab's source (pod and container selectors), narrowed in the merged stream without opening the streams that stay; every stream carries the server's timestamps; Current state |
| 2026-10-13 | minor | §8 upgrade to a repository's version (G11, night run): the Upgrade tab's repository and versions from the indexes Cogpit holds, Cogpit's defaults for the choice, `helm.release.upgrade` with `version` and `repository`, the chart downloaded and checked before the journal records anything, the confirmation's from and to bound to what runs; Current state |
| 2026-10-13 | minor | §4.3 Preferences and cluster settings as engine state (G13, night run): `UIState.Preferences`, `ClusterSettings`, `Settings`, their stores and commands, the GUI that applies them, the settings pages in the window's centre, cluster identity; Current state and the `cmd/cogpit` row |
| 2026-10-13 | minor | The cluster Overview (G12, M7, night run): §4.3 the page `cluster/overview` (not the plan's `overview`), its worker, the Issues as rows of several kinds (`MixedKinds`), `UIState.Overview`; §9 the cluster scope (`KindCluster`, `ClusterScope`, the masters in the scope's name, the workers every node less the masters), its live sums, Prometheus's queries, the Pods from the cache; one line in Current state |
| 2026-10-13 | minor | §5: a virtual kind made of several kinds (the Applications), served from its own cache whose source holds the workloads' handles while read; Current state |
| 2026-10-13 | minor | §4.5: curated profiles of CRD kinds built (`kinds.ProfileFor`), the Vertical Pod Autoscalers the first |
| 2026-10-13 | minor | Current state: the four Device Resources kinds (`resource.k8s.io/v1`), 45 built-in views |
| 2026-10-13 | minor | §8 install (G11, night run): the Helm Install tab with the chart downloaded and checked against the index digest, the server-side dry run as the preview, the install as a crash-safe job offered as an uninstall, values scrubbed of what differs from the chart's defaults |
| 2026-10-13 | minor | §8 chart repositories (G11, night run): kept in Cogpit's own folders (the lead's default), public https only, indexes downloaded on add, on update and while the Charts view or a Helm Install tab is open past a 10-minute TTL, bounded and parsed with the SDK; the charts as the local kind `helmcharts` |
| 2026-10-13 | minor | §4.2 the command palette (G13, night run): which commands it lists by the registry's metadata, how it runs them, the shortcuts bound from the registry; §4.3 `UIState.Palette`; one line in Current state, 97 commands |
| 2026-10-13 | minor | §7 the stdio bridge as built: the endpoint file beside the token, forwarding with origin `ExternalAgent`, following a restart, the headless fallback; Current state rows of `internal/mcp` and `cmd/cogpit-cli` |
| 2026-10-13 | minor | `cogpit-cli contexts` and `get` (Current state); `internal/app`, the wiring both binaries share, in §2 and §3 with its `depguard` rule |
| 2026-10-13 | minor | §4.3 the navigator (night run): its search, keyboard row and open groups GUI-only (Cogpit's default), its tree Cogpit's own rows; one line in Current state |
| 2026-10-12 | minor | §4.3 the hotbar as engine state (G13, night run): `UIState.Hotbar`, the store in `hotbar.json`, `ChangeHotbar` and `HotbarView`, the GUI's bar and its F-keys; one line in Current state |
| 2026-10-12 | minor | Current state after the fifth run's evening: crash reports, the MCP switch, the window's metrics and chart hover, the quit crash fixed |
| 2026-10-12 | minor | §9 Prometheus in the window (G12, third slice): the variants per provider, Network and Filesystem drawn, the second series colour, byte axes in binary steps, ranges to 30 days held by the engine (`ui.setMetricsRange`, `MetricsTarget.Range`, `ui.state`'s `drawerCharts`); checked against a real Prometheus on kind, and the three faults it found fixed (the filesystem metric per kind, a node's disks only, the rate windows) |
| 2026-10-12 | minor | §9 Prometheus (G12, second slice, backend): detection among the cached Services and the override, the provider with range and instant queries, the detector chain, `RangeProvider`, per-chart ranges (`SetMetricsRange`), `ObjectMetrics` with network and filesystem series |
| 2026-10-12 | minor | §9 as built (G12 first slice): `internal/metrics` (the Metrics Server provider, detection as states, a hub with per-scope polling at 60 s and 15 s, a TTL cache, a bounded series per watched object), `KindProfile.Usage` and the usage columns, `Session.ObjectMetrics`, `EventMetrics`, `metrics.get`, the drawer's Metrics section and chart widget, the demo's synthetic Metrics Server |
| 2026-10-12 | minor | §4.3: panel widths (sidebar, drawer) are user-dragged, stored in the preferences and GUI-only; why the drawer followed its content and the rule that ends it |
| 2026-10-12 | minor | §6 (D-28 built, scope C3): the default kubeconfig no longer merged; `~/.kube/config` a file of the default folder, `$KUBECONFIG` and `--kubeconfig` files session sources, a bare context name resolved to the one file that defines it; plugins of `config` asked for like any file |
| 2026-10-12 | minor | §6: sync folders read dot files (regular files only); `~/.kube` a default sync folder and what that means for the trust model; one dialog for files and folders; a connection closed when its file goes; the connect log |
| 2026-10-12 | minor | Fifth run, look and use (D-22 to D-25): Solarized theme tokens and the terminal palette, the status bar link, the C helm logo (`internal/ui/logo`), copying values from the drawer (selectable lines, one copy icon, entries in YAML, a copied Secret value cleared from the clipboard); current state row |
| 2026-10-12 | minor | Review of the fourth run, fixed (§4: an apply from a tab's text bound to its text revision, R-10; §5 Secrets: a revealed value bound to the Secret's version, R-6; §5 covering scopes: a handover that timed out before the own informer's first list keeps the view instead of reporting every object deleted, R-7, and an exact handover during a relist keeps what the new informer holds, R-8; §8): an upgrade from a tab bound to the tab's text revision (R-2); upgrade and rollback checked against the server's newest revision, not only the cache's (R-1); values scrubbed in their quoted forms (R-4); the Rollback dialog's answer bound to the picked revision and dropped after its dispatch (R-3) |
| 2026-10-12 | minor | §10: the patched terminal widget's two goroutines (parser on the reading one, screen and state on the main one) and its fuzz targets |
| 2026-10-12 | minor | Fourth run, the third run's leftovers: the journals shared by Cogpit processes (§8: a directory lock, an owner per process, entries running elsewhere); `kinds.list` marks local and virtual kinds (§5); the View menu and Refresh Access Reviews in the GUI (§4.2); live related-objects indexes shared with the drawer's fetches, the cluster-role rule on `Listed`, `ui.navigate`'s filter check with related cells (§4.4); workload logs resume dropped streams and forget ended instances (§4.4); Restart's and Scale's patches carry the UID, mutating confirmations record cached UIDs (§4.2) |
| 2026-10-12 | minor | §6: pasting a kubeconfig removed (D-21): the command, the dialog, the owner-only store; sources of kind `pasted` from earlier builds are no longer listed, their files stay; 75 commands |
| 2026-10-12 | minor | Fourth run (agent D): kubeconfig sources (added files, pasted kubeconfigs, sync folders), each file read on its own, and credential plugins of sources run only after the user allowed them (§6; current state rows) |
| 2026-10-12 | minor | Fourth run, Helm: the upgrade as built (§8: what binds, the SDK's context, `ErrUnchanged`, values scrubbed, the Rollback dialog as the confirmation); Lens's release drawer; virtual kinds' drawers without metadata and events, `DetailTitle` (§4.4) |
| 2026-10-12 | minor | Fourth run, agent A: covering scopes built (§5, how a handover stays exact, the measured budgets); the Secrets view on a full watch, its drawer from the cache without a GET; `resources.find` without a kind skips uncached kinds that hold Secret values; the last-applied annotation re-checked and kept dropped |
| 2026-10-12 | minor | §4.4: the related objects' generation changes when the indexed kinds finish their initial list; §12: a stopped shell drops the work it queued for the main goroutine |
| 2026-10-12 | minor | §8: Lens's release drawer as captured the second time (`DetailModel.Plain`, `DetailItem.Pill`, Chart "<chart>-<version>", Namespace plain, the checkbox unchecked at first, Save a bar), the Rollback dialog enabled with the current revision (the engine accepts the newest revision as a target), the Upgrade tab's picker, links and find widget (C1) |
| 2026-10-12 | minor | Phase 0 of the fourth run: `model.Related.Listed`; covering scopes for the full Secrets watch and Helm's releases; the Helm upgrade's commands, seam, journal entry and dock tab |
| 2026-10-11 | minor | Access reviews built (§4.2): which review answers what, the namespaces the sidebar reviews, bounds and events, measured request counts; no TTL (§5); `ErrNotApplicable` before a denial |
| 2026-10-11 | minor | §4.4: drawer tables of cells (`model.CellTable`: links, status colours, the warning icon) and cells of several lines; telling a missing related object from a kind still listing |
| 2026-10-11 | minor | §4.5: `FallbackVersions`, a built-in kind's older API versions, picked by discovery at connect |
| 2026-10-11 | major | Current state after the third implementation run: 41 built-in profiles, local and virtual kinds, port forwarding, the Secrets view, Helm releases, access reviews, the title bar, 65 commands |
| 2026-10-11 | minor | Third run's Phase 0: cache scopes with label selector and metadata only, local and virtual kinds (§5), the related-objects index and item actions in the drawer (§4.4), access reviews in `Enabled` (§4.2), Helm v4.3.0 in `go.mod` |
| 2026-10-11 | major | Current state after the second implementation run: 16 built-in profiles and custom resources, editing, node and workload actions, the drain journal, 41 commands, CI |
| 2026-10-11 | minor | §4.2: what a confirmation binds (context, objects, UIDs), `Detail`, `OriginOf` |
| 2026-10-11 | minor | Row menus from the registry (`Actions`, submenus, schema dialogs); the drawer model's Properties section, links, chips, folds and tables; cells in coloured parts |
| 2026-10-11 | minor | Current state: editing and creating objects of every kind (G9) in cluster, engine and UI |
| 2026-10-11 | minor | §5: the watch on custom resource definitions and how custom kinds' profiles are published and read |
| 2026-10-11 | minor | Kind keys of custom resources (§4.5); links and the warning column in cells; the registry's Order, ErrNotApplicable, Escalate and Prefill |
| 2026-10-11 | major | Current state: the local MVP and where it differs from the plan |
| 2026-10-11 | major | Design plan v2: Go; cache-first data flow (§5); UI state in the engine; MCP and the built-in agent (§7); Helm in-process with crash-safe actions; terminal via own PTY; updates |
| 2026-10-11 | major | Design plan v1 (Rust), adapted from PhotoCraft's architecture |
