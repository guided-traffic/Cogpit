# Architecture

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (current state after the third implementation run) · **Target:** Lens Desktop 2026.10.51501

## Current state (2026-10-11)

The local MVP is built and was extended by a second and a third implementation run (2026-10-11):
a macOS desktop app that connects to a kubeconfig context or a synthetic demo cluster, shows 41
built-in views, the Port Forwarding view (a local kind), the Helm Releases view (a virtual kind),
the Workloads Overview page and one view per custom resource kind from a watch-fed cache, greys
what the user's access reviews deny, has a detail drawer and row menus from the command registry,
a dock with logs (with their options and the logs of a workload's pods), a local terminal, pod
shells, attach and Edit and Create resource tabs, writes through 65 commands (editing, delete,
node and workload actions, a crash-safe drain, port forwards, Helm rollback and uninstall with a
crash journal), draws its own title bar on macOS, and an MCP server lets agents drive all of it.
CI runs on every push ([releasing › CI](releasing.md#ci)). The MVP doubled as the Fyne spike
([results](spike-fyne.md#results)). Where the code differs from the plan below, this section wins.

| Package | State | Differs from the plan |
|---|---|---|
| [`internal/model`](../../internal/model/), [`model/kinds`](../../internal/model/kinds/) | Built: refs, rows (links, coloured parts, warning and until columns), status tokens, panic-free accessors, Secret redaction, the drawer's Properties; 16 built-in profiles (Nodes, Pods, Deployments, Daemon Sets, Stateful Sets, Replica Sets, Jobs, Cron Jobs, Services, Ingresses, Ingress Classes, Config Maps, Persistent Volume Claims, Namespaces, Events, Custom Resource Definitions) and a generic profile built from a custom resource definition and its printer columns, with golden and fuzz tests; the third run added 25 built-in profiles (the Config, Access Control, Network and Storage views, Secrets from metadata only, Replication Controllers; 41 in all), the local kind Port Forwarding, the profile fields of §5 (scopes, `Derive`, `Local`, `FallbackVersions`) and the related-objects indexes of §4.4, drawer tables of cells, item actions and private items | No CPU and memory columns; no generic view for built-in kinds without a profile or for aggregated APIs; Cron Jobs' Next execution uses `time.LoadLocation` (the zone database is embedded in `cmd/cogpit` for Windows) |
| [`internal/kubeconfig`](../../internal/kubeconfig/) | Built: contexts from bytes, error text without credentials, the per-terminal kubeconfig | – |
| [`internal/cluster`](../../internal/cluster/) | Built: connections through client-go's loading rules, the counting transport, discovery, log streams, restart and delete; for every kind, a REST mapper over in-memory cached discovery and update (with a resourceVersion precondition), create (both with dryRun=All) and delete (with a UID precondition) through the dynamic client ([`objects.go`](../../internal/cluster/objects.go)); an update sent without `managedFields` (the cache drops them) keeps every manager's entry on the server and records Cogpit's fields under the manager `cogpit` (verified on kind 1.36.1, `TestKindEditCreateConflictDelete`); access reviews (`ReviewRules`, `ReviewAccess`, [`access.go`](../../internal/cluster/access.go)); third run: the metadata client for metadata-only informers, port-forward tunnels on loopback listeners Cogpit owns across reconnects ([`portforward.go`](../../internal/cluster/portforward.go)), one Secret's GET, restarts of Daemon and Stateful Sets, Replica Set scale, the API server's warnings logged once per connection | Discovery is cached in memory for 10 minutes, not on disk; one rate limiter per cluster (QPS 5, burst 10) shared by every client, discovery with its own budget (50/300), access reviews with theirs (10/20, so that they never delay a view's list); implicit kubeconfig writes are switched off (the legacy `.kubeconfig` migration and the oidc persister, which keeps refreshed tokens in memory) |
| [`internal/cache`](../../internal/cache/) | Built: one shared informer per scope, handles, warm period, `managedFields` transform, no resync, streaming initial lists where the server supports them, error state, stats; a handle reports synced only once the initial list reached its subscribers; third run: scopes with label selectors and metadata only (§5), `cache.Memory` for local kinds, the `last-applied-configuration` annotation dropped from Secrets before they enter the store | Views watch all namespaces and filter client-side; no polling sources yet |
| [`internal/terminal`](../../internal/terminal/) | Built: local shell on Cogpit's own PTY with the per-terminal kubeconfig; pod shell over exec (WebSocket, SPDY fallback); attach to a container's main process (third run), with stdin and a TTY as the container has them | No node shell; Windows has no local shell yet |
| [`internal/demo`](../../internal/demo/) (L2, new) | Built: the synthetic cluster for `--demo` behind a `cache.Source`: 10,000 pods, 300 services, 1,000 watch events per second, watch resume and 410 Gone | A package the plan did not list |
| [`internal/engine`](../../internal/engine/), [`engine/commands`](../../internal/engine/commands/) | Built: Session, registry with JSON schemas, `Order`, `Escalate`, `Prefill`, `Detail` and `ErrNotApplicable` (§4.2), D-2 confirmation per origin bound to its context, objects and UIDs, audit log with Secrets redacted, UI state, coalesced snapshots (≤ 10 per second per view), the drawer model, streams, editor tabs, the custom kinds from a watch on the definitions, the drain journal, 41 commands; the access reviews of the connected user ([`accessreviews.go`](../../internal/engine/accessreviews.go), §4.2); third run: `Backend.Acquire`, `KindList` and `KindGet` for every read of a kind, the local store, the related-objects index, the port-forward manager ([`portforward.go`](../../internal/engine/portforward.go)), the Secrets drawer's GET and GUI-only reveal ([`secretdata.go`](../../internal/engine/secretdata.go)), Helm's actions and journal ([`helmops.go`](../../internal/engine/helmops.go)), log tabs with options and workload logs, attach, hidden columns and the group filter as UI state, the Workloads Overview page, CSV export, drawer-only commands; 65 commands | No jobs or cancellation UI (a drain runs in its dispatch call); a write command refuses to run if the user switched clusters while its confirmation was open; access reviews have no time-to-live (§5): they are kept until `cluster.refreshAccess` or the next connect |
| [`internal/mcp`](../../internal/mcp/) | Built: every command as a tool, Streamable HTTP on loopback, bearer token, Host and Origin checks | Stateless HTTP (one session per request); the token lives in `<user config dir>/Cogpit/mcp-token` and changes with the port on every start; no stdio bridge |
| [`internal/ui/fyneui`](../../internal/ui/fyneui/) and its [`yamlview`](../../internal/ui/fyneui/yamlview/) and [`termview`](../../internal/ui/fyneui/termview/) | Built: shell after Lens's layout, theme with status tokens, context picker, confirmation dialog, the YAML view, terminal and log widgets, a jank meter (`--perf`); row menus (⋮ and right-click) and the drawer's toolbar from the registry, schema-generated parameter dialogs, links and coloured parts in cells, Cogpit's own tooltips (Fyne 2.8 has none); third run: Cogpit's own title bar on macOS through a cgo bridge to the `NSWindow` ([`titlebar_darwin.go`](../../internal/ui/fyneui/titlebar_darwin.go); `COGPIT_SYSTEM_TITLEBAR=1` keeps the system one), the column menu, keyboard navigation, fit-to-content, CSV download, the Workloads Overview, item actions in the drawer (Forward…, Show), the port-forward dialog, the Helm banner, log tabs with their toolbar | The YAML view is editable in Edit and Create resource tabs, whose text lives in the engine (`editor.get`, `editor.setText`); edits are applied as updates (PUT), not patches; the terminal widget is a patched copy of fyne-io/terminal ([`third_party/fyne-terminal`](../../third_party/fyne-terminal/PATCHES.md)) behind an escape filter, because the pinned upstream froze or crashed on container output |
| [`cmd/cogpit`](../../cmd/cogpit/) | Built: wires the packages, the drain journal (`<user config dir>/Cogpit/drains.json`) and the Helm journal (`helm.json`); flags `--kubeconfig`, `--context`, `--demo`, `--mcp-addr`, `--no-mcp`, `--perf`, `--theme` | client-go goroutine panics are logged instead of ending the app |
| [`internal/helm`](../../internal/helm/) (L2) | Built in the third run: release payloads decoded (bounded, fuzzed), the Releases profile as a virtual kind over the `owner=helm` Secrets with a bounded memo, the drawer (history, values as private items, the manifest with Secret data redacted), rollback and uninstall through the Helm SDK on the connection's own client configuration | Install, upgrade, repositories and charts not built; the SDK takes no context, so an action cannot be cancelled (it runs up to its 5-minute timeout) |
| `cmd/cogpit-cli`, `internal/metrics`, `internal/agent`, `internal/update` | Not built (`cogpit-cli` is a stub) | – |

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
│  ├─ ui/fyneui/      L4  the Fyne shell; the only package that imports Fyne
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
 L4  ui/fyneui · mcp · agent · update
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
those as preconditions, never the UID of an object created again under the name. A command can ask
who dispatched it (`OriginOf`); `editor.get` uses that to redact Secrets for agents.

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
connected again; nothing runs on a timer. A missing answer starts its review in the background,
once (deduplicated), at most four at a time, on goroutines that recover from a panic and record a
failed review instead of leaving it pending; arriving answers emit `EventKinds` (the sidebar reads
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
namespaces' rules. `kinds.list` reports `listable` per kind for the sidebar, which hides a kind
known not listable and shows one not known yet: a cluster-scoped kind is listable when the default
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
}
```

The GUI changes state only through commands (`ui.openView`, `ui.select`, `ui.navigate`,
`ui.setFilter`), so a selection made by the user and one made by an agent are the same event.
Added 2026-10-11 ([`uistate.go`](../../internal/engine/uistate.go)): `Groups`, the Definitions
view's API group filter (`ui.setGroups`), and `HiddenColumns`, the columns each view hides
(`ui.setColumns`; a view without an entry hides its kind's default-hidden columns). The view's
snapshot applies both: its `Columns` carry `Hidden` as the view hides them, at the kind's indexes.

**Pages** (decided 2026-10-11, [`overview.go`](../../internal/engine/overview.go)): a view that is
not one kind's list has a page key with a slash, `<sidebar group>/<page>`, which no kind key has.
The first is `workloads/overview`, the Workloads Overview; the cluster's own Overview keeps the free
key `overview`. `ui.openView` opens a page like a kind. While it is open the engine runs the page's
worker in place of a view worker: it holds handles on the kinds the page reads (the seven workload
kinds and the events), released when another view opens, and builds the page's model (each kind's
count and status tokens, `workloads.overview`) and its table (the events, `Session.Table` under the
page key). The table's rows keep their kind's key (`events`), so `ui.select`, row menus and the
drawer work on them; `Session.ViewProfile` returns a page's table profile, so `ui.sort`,
`ui.setColumns` and `table.exportCsv` work on a page as on a kind's view.
Agents read it with `ui_state`: this is how the built-in agent knows which object the user means
by "this pod".

### 4.4 Tables, details, editors and streams

- **Tables:** `TableSnapshot{Columns, Rows, Sort, Filter, Total, Revision}`; rows carry display
  text, sort keys and a status token that the toolkit maps to a theme colour.
- **Details:** section models: metadata, labels and annotations, conditions, related objects,
  events, containers, a metrics slot. As built (2026-10-11, [`detail.go`](../../internal/engine/detail.go),
  [`model/detail.go`](../../internal/model/detail.go)), in Lens's order: the metrics slot (not built,
  M7), one Properties section (the object's metadata without the UID, then the kind's own
  properties: a profile's first section titled "Properties" merges into it), the kind's other
  sections (a section may carry a `group` heading, "Containers", or a `table`, a node's
  Capacity), then the events. A table cell may hold several lines (a role's resources and
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
  starts with `<pod>/<container>`; at most 40 containers at once.
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
  objects' generation as well as the resourceVersion), and builds each index at most once per
  snapshot. A lookup on a kind that is still listing or not served returns nothing. Rows built
  outside a view (`resources.list`) use `Session.BuildRows`. A rule that warns about an object
  that does not exist (a binding's cluster role) must tell that from a kind still listing: its
  index also files every object under a key no object has (`ClusterRolesByName` under `""`), and
  the rule warns only once that key returns objects (2026-10-11).

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
  `<plural>.<group>` key, so a key never changes when Cogpit learns to show a kind better.
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
`kubectl apply` fills with the whole manifest, values included: without it a metadata-only scope
would still hold the values of every applied Secret (`TestSecretsLoseTheLastAppliedAnnotation`).
The annotation still crosses the wire with the metadata.

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
(empty: all), a **label selector** and a **metadata-only** flag; each distinct scope is one informer
and one watch, and `Acquire` normalises the selector, so `a=1,b=2` and `b=2,a=1` share one. Every
`Source` honours the whole scope: the real cluster's source (`DynamicSource` over the dynamic and
the metadata client) sends the selector to the server and asks for `PartialObjectMetadata` for a
metadata-only scope, so object content never crosses the wire; in-memory sources apply
`cache.FilterScope` or `cache.FilterListerWatcher`. A metadata-only object reaches a kind profile
as an `unstructured.Unstructured` with apiVersion `meta.k8s.io/v1`, kind `PartialObjectMetadata`
and the object's `metadata`, nothing else (`cache.MetadataObject`, `cache.IsMetadataObject`).

A kind's profile decides its scope (`MetadataOnly`, `LabelSelector`), and every engine read of a
kind goes through `Backend.Acquire(profile)` and `KindList`/`KindGet`, never through a scope built
by hand, so the view, the drawer, `Lookup` and agents' reads of a kind share one informer.
Measured at the HTTP level: the Secrets of D-13 as metadata (`secrets (metadata)`) and Helm's
release Secrets (`secrets with owner=helm`) are one initial list or streaming watch each, and no
full or unselected Secrets request is sent (`TestBudgetSecretScopes`; on kind, `TestKindSecretScopes`).
The Secrets view's drawer (built 2026-10-11, [`secretdata.go`](../../internal/engine/secretdata.go))
makes the one direct read of Secret values: a `get secrets` of the Secret it opens on, once per
open, through `SecretReader` on `Backend.Writer` (`cluster.Conn.GetSecret`), skipped when the access
reviews deny it. The engine keeps the type and, per key, the size and the first 4 KiB of a text
value while the drawer shows that Secret and wipes them when it closes or moves; the drawer model
lists the keys masked, and `secret.reveal` hands one value to the GUI origin only
(`TestBudgetSecretsViewAndDrawer`: one metadata watch and one GET).

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
- `object.edit` and `object.delete` do not apply to local and virtual kinds
  (`KindProfile.APIObject`); such kinds bring their own commands.

**Custom resource kinds** (2026-10-11, [`customkinds.go`](../../internal/engine/customkinds.go)).
A real cluster's backend holds one cache handle on `customresourcedefinitions` while it is
connected (`Backend.CustomResources`; the demo cluster has none). Its watch events, coalesced,
rebuild the generic profiles of the established definitions
([`kinds.ParseCRD`](../../internal/model/kinds/customresources.go)), published as one immutable set
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

## 7. MCP and the built-in agent

**MCP server** (`internal/mcp`, `github.com/modelcontextprotocol/go-sdk` v1.8.0):

- tools generated from the registry, with the registry's schemas and MCP tool annotations
  (`readOnlyHint` for `Read` and `UI`, `destructiveHint` for `Destructive`);
- Streamable HTTP on `127.0.0.1`, a random bearer token written owner-only to the app's runtime
  directory, `Host` and `Origin` checks against DNS rebinding;
- `cogpit-cli mcp` as a stdio bridge for agents that only speak stdio: it reads the token and
  forwards to the running app, or runs a headless engine when no app is running.

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
cached Secrets labelled `owner=helm` per namespace and release name, the newest revision wins,
and the object carries that revision's Secret UID and resourceVersion and a summary of the
history, never values or the manifest (the YAML tab shows this summary). Decoded payloads are
memoized per Secret UID and resourceVersion, values and manifests for the newest revision of each
release only; entries for Secrets that are gone leave the memo after two generations of 64 calls,
and it holds at most 4,096. The drawer shows Properties, the History table, the user-supplied
values (marked `Private`: agents read `model.Withheld`) and the manifest with every Secret's data
redacted for everyone, both cut for the drawer (16 KiB or 200 lines, 32 KiB or 400 lines).
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

## 9. Metrics (`internal/metrics`)

Detect a provider per cluster (metrics-server, Prometheus through the API server's service proxy,
the provider variants Lens lists), allow an override in cluster settings, poll only while a visible
view shows metrics, cache results with a time-to-live. No provider is a state, not an error.
Cogpit never installs anything into a cluster unless the user starts that action.

## 10. Terminal (`internal/terminal`)

The terminal widget is `github.com/fyne-io/terminal`. Its `RunLocalShell` starts the shell with
the process's own environment (verified in its source), so a terminal-specific `KUBECONFIG` is not
possible that way. Cogpit starts the shell itself on a PTY (`creack/pty` on macOS and Linux, ConPTY
on Windows) with the per-terminal kubeconfig of
[security › credentials](../security/credentials.md#the-terminals-kubeconfig), and connects the
widget with `RunWithConnection`. Pod exec and attach streams from client-go connect the same way.
Terminal output is untrusted: escape sequences that write the clipboard or change window state are
filtered ([trust boundaries](../security/trust-boundaries.md)).

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
  `fyne.Do` (Fyne 2.6 and later).
- Every goroutine that does cluster I/O recovers from a panic at its boundary and reports it
  as an error ([D-11](decisions.md#d-11-crash-stance)).
- Jobs take a `context.Context`; the UI can cancel any job with progress.

## 13. Testing and quality

Tiers, fixtures and budgets: [`testing.md`](testing.md).

## 14. Key dependencies (observed 2026-10-11)

Versions as the Go module proxy listed them; pinned when M0 creates `go.mod`.

| Need | Module | Version | Note |
|---|---|---|---|
| UI toolkit | `fyne.io/fyne/v2` | v2.8.1 | decided after the spike gate ([D-1](decisions.md#d-1-ui-toolkit)) |
| Terminal widget | `github.com/fyne-io/terminal` | commit `91063bb29d69` (2026-09-27) | untagged; newer commits require an unreleased Fyne |
| Packaging | `fyne.io/tools` | v1.7.3 | `fyne package` |
| Kubernetes API | `k8s.io/client-go`, `k8s.io/apimachinery`, `k8s.io/api` | v0.37.1 | Kubernetes 1.37 |
| kubectl behaviour | `k8s.io/kubectl` | v0.37.1 | drain, rollout, describe |
| Metrics API | `k8s.io/metrics` | v0.37.1 | |
| Helm | `helm.sh/helm/v4` | v4.3.0 | in-process ([D-3](decisions.md#d-3-helm-and-kubectl-binaries)); in `go.mod` since 2026-10-11, built against k8s.io v0.37.0 and resolved to v0.37.1 with the rest (it also brings `k8s.io/apiserver`, `apiextensions-apiserver` v0.37.1, `controller-runtime` v0.24.1 and `oras-go` as indirect dependencies) |
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
