# Architecture

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (current state of the local MVP) · **Target:** Lens Desktop 2026.10.51501

## Current state (2026-10-11)

The local MVP is built: a macOS desktop app that connects to a kubeconfig context or a synthetic
demo cluster, shows six views from a watch-fed cache, has a detail drawer, a dock with logs, a
local terminal and a pod shell, and an MCP server that lets agents drive the UI. It doubles as the
Fyne spike ([results](spike-fyne.md#results)). Where the code differs from the plan below, this
section wins.

| Package | State | Differs from the plan |
|---|---|---|
| [`internal/model`](../../internal/model/), [`model/kinds`](../../internal/model/kinds/) | Built: refs, rows, status tokens, panic-free accessors, Secret redaction; profiles for Nodes, Pods, Deployments, Services, Namespaces and Events with golden and fuzz tests | No generic profile for custom resources yet; no CPU and memory columns |
| [`internal/kubeconfig`](../../internal/kubeconfig/) | Built: contexts from bytes, error text without credentials, the per-terminal kubeconfig | – |
| [`internal/cluster`](../../internal/cluster/) | Built: connections through client-go's loading rules, the counting transport, discovery, log streams, restart and delete; for every kind, a REST mapper over in-memory cached discovery and update (with a resourceVersion precondition), create (both with dryRun=All) and delete (with a UID precondition) through the dynamic client ([`objects.go`](../../internal/cluster/objects.go)); an update sent without `managedFields` (the cache drops them) keeps every manager's entry on the server and records Cogpit's fields under the manager `cogpit` (verified on kind 1.36.1, `TestKindEditCreateConflictDelete`) | Discovery is cached in memory for 10 minutes, not on disk; one rate limiter per cluster (QPS 5, burst 10) shared by every client, discovery with its own budget (50/300); no access reviews; implicit kubeconfig writes are switched off (the legacy `.kubeconfig` migration and the oidc persister, which keeps refreshed tokens in memory) |
| [`internal/cache`](../../internal/cache/) | Built: one shared informer per scope, handles, warm period, `managedFields` transform, no resync, streaming initial lists where the server supports them, error state, stats | Views watch all namespaces and filter client-side; no metadata-only informers; no polling sources yet |
| [`internal/terminal`](../../internal/terminal/) | Built: local shell on Cogpit's own PTY with the per-terminal kubeconfig; pod shell over exec (WebSocket, SPDY fallback) | No attach, no node shell; Windows has no local shell yet |
| [`internal/demo`](../../internal/demo/) (L2, new) | Built: the synthetic cluster for `--demo` behind a `cache.Source`: 10,000 pods, 300 services, 1,000 watch events per second, watch resume and 410 Gone | A package the plan did not list |
| [`internal/engine`](../../internal/engine/), [`engine/commands`](../../internal/engine/commands/) | Built: Session, registry with JSON schemas, D-2 confirmation per origin, audit log, UI state, coalesced snapshots (≤ 10 per second per view), drawer, streams, 21 commands | No jobs or cancellation UI, no access reviews; a write command refuses to run if the user switched clusters while its confirmation was open |
| [`internal/mcp`](../../internal/mcp/) | Built: every command as a tool, Streamable HTTP on loopback, bearer token, Host and Origin checks | Stateless HTTP (one session per request); the token lives in `<user config dir>/Cogpit/mcp-token` and changes with the port on every start; no stdio bridge |
| [`internal/ui/fyneui`](../../internal/ui/fyneui/) and its [`yamlview`](../../internal/ui/fyneui/yamlview/) and [`termview`](../../internal/ui/fyneui/termview/) | Built: shell after Lens's layout, theme with status tokens, context picker, confirmation dialog, read-only YAML view, terminal and log widgets, a jank meter (`--perf`) | The YAML view is editable in Edit and Create resource tabs, whose text lives in the engine (`editor.get`, `editor.setText`); edits are applied as updates (PUT), not patches; the terminal widget is a patched copy of fyne-io/terminal ([`third_party/fyne-terminal`](../../third_party/fyne-terminal/PATCHES.md)) behind an escape filter, because the pinned upstream froze or crashed on container output |
| [`cmd/cogpit`](../../cmd/cogpit/) | Built: wires the packages; flags `--kubeconfig`, `--context`, `--demo`, `--mcp-addr`, `--no-mcp`, `--perf`, `--theme` | client-go goroutine panics are logged instead of ending the app |
| `cmd/cogpit-cli`, `internal/helm`, `internal/metrics`, `internal/agent`, `internal/update` | Not built (`cogpit-cli` is a stub) | – |

Layering is enforced by `depguard` in `make lint`, as planned in §3, with the demo package in L2.

Decided on 2026-10-11 for the next implementation run: the kind key of custom resources
([§4.5](#45-kind-keys)), and the contracts the parallel work shares: links in cells and the warning
column ([§4.4](#44-tables-details-editors-and-streams)), and the command fields `Order`,
`Escalate`, `Prefill` and `ErrNotApplicable` ([§4.2](#42-the-command-registry)).

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
Agents read it with `ui_state`: this is how the built-in agent knows which object the user means
by "this pod".

### 4.4 Tables, details, editors and streams

- **Tables:** `TableSnapshot{Columns, Rows, Sort, Filter, Total, Revision}`; rows carry display
  text, sort keys and a status token that the toolkit maps to a theme colour.
- **Details:** section models: metadata, labels and annotations, conditions, related objects,
  events, containers, a metrics slot.
- **Editors, logs, terminals and chats** are engine sessions shown as dock tabs, so agents can open,
  read and close them too.
- **Links and warnings in cells** (2026-10-11, [`row.go`](../../internal/model/row.go)): a cell
  that names another object carries a `Link` (an object reference with a kind key; an empty context
  means the row's), which the UI renders as a link. A Namespace link in a list narrows the view's
  namespace filter (`ui.setNamespace`, owner 2026-10-11); other links navigate (`ui.navigate`).
  Lens's warning-icon column right after Name is a column of type `warning`: a cell with a status
  shows the icon in that status's colour with its text as the tooltip; an empty cell shows nothing.

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
  served version by Kubernetes version priority, as discovery orders them).

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
| Access reviews cached per context and namespace with a TTL | Menus and sidebars don't ask the API server on every render | engine cache |
| Client-side rate limit per cluster | Bounds lists, gets and writes; established watches are not affected | `rest.Config` QPS and Burst (defaults 5 and 10; a preference) |

**Current, but cheap.** Watch events land in the cache as they arrive; the engine rebuilds the
snapshots of the views that use a changed kind, coalesced to at most ten per second per view, so a
burst of a thousand events costs the UI at most ten redraws. Data that cannot be watched (metrics,
Helm chart repository indexes) is polled only while something visible needs it, at an interval per
source, and cached with a time-to-live.

**Small in memory.** A transform drops `metadata.managedFields` before an object enters the cache
(the editor's "managed fields" toggle fetches them on demand). Memory per cluster and object counts
per kind are visible in a diagnostics view.

**Measurable.** A counting HTTP round-tripper per cluster records requests by verb and resource;
the diagnostics view shows them, and tests assert the budgets P9–P11 in
[`testing.md`](testing.md#performance-budgets-proposed-not-measured).

**Reads from the cache.** The detail drawer, the YAML editor's initial content, `ui_state`,
`resources_find` and every other read are served from the cache. A direct `GET` happens only for
data the cache does not hold (subresources, a kind nobody watches), and its result does not bypass
the cache's view of the world.

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
| Helm | `helm.sh/helm/v4` | v4.3.0 | in-process ([D-3](decisions.md#d-3-helm-and-kubectl-binaries)) |
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
| 2026-10-11 | minor | Current state: editing and creating objects of every kind (G9) in cluster, engine and UI |
| 2026-10-11 | minor | Kind keys of custom resources (§4.5); links and the warning column in cells; the registry's Order, ErrNotApplicable, Escalate and Prefill |
| 2026-10-11 | major | Current state: the local MVP and where it differs from the plan |
| 2026-10-11 | major | Design plan v2: Go; cache-first data flow (§5); UI state in the engine; MCP and the built-in agent (§7); Helm in-process with crash-safe actions; terminal via own PTY; updates |
| 2026-10-11 | major | Design plan v1 (Rust), adapted from PhotoCraft's architecture |
