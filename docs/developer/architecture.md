# Architecture

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-12 · **Change:** minor (related rows rebuilt when the indexed kinds finish listing; a stopped shell drops its queued work; fourth run: kubeconfig sources and the approval of their credential plugins, §6; an upgrade from a tab bound to the tab's text revision, values scrubbed in their quoted forms too, upgrade and rollback checked against the server's newest revision, §8) · **Target:** Lens Desktop 2026.10.51501

## Current state (2026-10-11)

The local MVP is built and was extended by a second and a third implementation run (2026-10-11)
and a fourth (2026-10-12):
a macOS desktop app that connects to a kubeconfig context or a synthetic demo cluster, shows 41
built-in views, the Port Forwarding view (a local kind), the Helm Releases view (a virtual kind),
the Workloads Overview page and one view per custom resource kind from a watch-fed cache, greys
what the user's access reviews deny, has a detail drawer and row menus from the command registry,
a dock with logs (with their options and the logs of a workload's pods), a local terminal, pod
shells, attach and Edit and Create resource tabs, writes through 76 commands (editing, delete,
node and workload actions, a crash-safe drain, port forwards, Helm rollback, uninstall and
upgrade with a crash journal locked between Cogpit processes), adds, pastes and syncs kubeconfigs
whose credential plugins run only once the user allowed them, holds Secrets in a full watch that
Helm's releases share, draws its own title bar on macOS, and an MCP server lets agents drive all
of it.
CI runs on every push ([releasing › CI](releasing.md#ci)). The MVP doubled as the Fyne spike
([results](spike-fyne.md#results)). Where the code differs from the plan below, this section wins.

| Package | State | Differs from the plan |
|---|---|---|
| [`internal/model`](../../internal/model/), [`model/kinds`](../../internal/model/kinds/) | Built: refs, rows (links, coloured parts, warning and until columns), status tokens, panic-free accessors, Secret redaction, the drawer's Properties; 16 built-in profiles (Nodes, Pods, Deployments, Daemon Sets, Stateful Sets, Replica Sets, Jobs, Cron Jobs, Services, Ingresses, Ingress Classes, Config Maps, Persistent Volume Claims, Namespaces, Events, Custom Resource Definitions) and a generic profile built from a custom resource definition and its printer columns, with golden and fuzz tests; the third run added 25 built-in profiles (the Config, Access Control, Network and Storage views, Secrets (in full since the fourth run, with Keys and Type), Replication Controllers; 41 in all), the local kind Port Forwarding, the profile fields of §5 (scopes, `Derive`, `Local`, `FallbackVersions`) and the related-objects indexes of §4.4, drawer tables of cells, item actions and private items | No CPU and memory columns; no generic view for built-in kinds without a profile or for aggregated APIs; Cron Jobs' Next execution uses `time.LoadLocation` (the zone database is embedded in `cmd/cogpit` for Windows) |
| [`internal/kubeconfig`](../../internal/kubeconfig/) | Built: contexts from bytes, error text without credentials, the per-terminal kubeconfig; fourth run (2026-10-12): the exec credential plugin of a context with its fingerprint, the checks of pasted text, untrusted kubeconfig text made safe to show ([`exec.go`](../../internal/kubeconfig/exec.go), [`source.go`](../../internal/kubeconfig/source.go)) | – |
| [`internal/cluster`](../../internal/cluster/) | Built: connections through client-go's loading rules, the counting transport, discovery, log streams, restart and delete; for every kind, a REST mapper over in-memory cached discovery and update (with a resourceVersion precondition), create (both with dryRun=All) and delete (with a UID precondition) through the dynamic client ([`objects.go`](../../internal/cluster/objects.go)); an update sent without `managedFields` (the cache drops them) keeps every manager's entry on the server and records Cogpit's fields under the manager `cogpit` (verified on kind 1.36.1, `TestKindEditCreateConflictDelete`); access reviews (`ReviewRules`, `ReviewAccess`, [`access.go`](../../internal/cluster/access.go)); third run: the metadata client for metadata-only informers, port-forward tunnels on loopback listeners Cogpit owns across reconnects ([`portforward.go`](../../internal/cluster/portforward.go)), restarts of Daemon and Stateful Sets, Replica Set scale, the API server's warnings logged once per connection; fourth run (2026-10-12): one kubeconfig file read on its own (`InspectFile`), sync folder listings, and `Options.ExecPolicy`, asked before a context's exec plugin can run ([`sources.go`](../../internal/cluster/sources.go), §6) | Discovery is cached in memory for 10 minutes, not on disk; one rate limiter per cluster (QPS 5, burst 10) shared by every client, discovery with its own budget (50/300), access reviews with theirs (10/20, so that they never delay a view's list); implicit kubeconfig writes are switched off (the legacy `.kubeconfig` migration and the oidc persister, which keeps refreshed tokens in memory) |
| [`internal/cache`](../../internal/cache/) | Built: one shared informer per scope, handles, warm period, `managedFields` transform, no resync, streaming initial lists where the server supports them, error state, stats; a handle reports synced only once the initial list reached its subscribers; third run: scopes with label selectors and metadata only (§5), `cache.Memory` for local kinds, the `last-applied-configuration` annotation dropped from Secrets before they enter the store; fourth run: covering scopes (a label-selected scope served from its running unselected scope, with handovers both ways, §5) | Views watch all namespaces and filter client-side; no polling sources yet |
| [`internal/terminal`](../../internal/terminal/) | Built: local shell on Cogpit's own PTY with the per-terminal kubeconfig; pod shell over exec (WebSocket, SPDY fallback); attach to a container's main process (third run), with stdin and a TTY as the container has them | No node shell; Windows has no local shell yet |
| [`internal/demo`](../../internal/demo/) (L2, new) | Built: the synthetic cluster for `--demo` behind a `cache.Source`: 10,000 pods, 300 services, 1,000 watch events per second, watch resume and 410 Gone | A package the plan did not list |
| [`internal/engine`](../../internal/engine/), [`engine/commands`](../../internal/engine/commands/) | Built: Session, registry with JSON schemas, `Order`, `Escalate`, `Prefill`, `Detail` and `ErrNotApplicable` (§4.2), D-2 confirmation per origin bound to its context, objects and UIDs, audit log with Secrets redacted, UI state, coalesced snapshots (≤ 10 per second per view), the drawer model, streams, editor tabs, the custom kinds from a watch on the definitions, the drain journal, 41 commands; the access reviews of the connected user ([`accessreviews.go`](../../internal/engine/accessreviews.go), §4.2); third run: `Backend.Acquire`, `KindList` and `KindGet` for every read of a kind, the local store, the related-objects index, the port-forward manager ([`portforward.go`](../../internal/engine/portforward.go)), the Secrets drawer and its GUI-only reveal ([`secretdata.go`](../../internal/engine/secretdata.go); from the cache since the fourth run, no GET), Helm's actions and journal ([`helmops.go`](../../internal/engine/helmops.go)), log tabs with options and workload logs, attach, hidden columns and the group filter as UI state, the Workloads Overview page, CSV export, drawer-only commands; 65 commands; fourth run (2026-10-12): kubeconfig sources (added files, pasted kubeconfigs, sync folders) and the approvals of their credential plugins ([`kubeconfigs.go`](../../internal/engine/kubeconfigs.go), §6), seven `kubeconfig.*` commands | No jobs or cancellation UI (a drain runs in its dispatch call); a write command refuses to run if the user switched clusters while its confirmation was open; access reviews have no time-to-live (§5): they are kept until `cluster.refreshAccess` or the next connect |
| [`internal/mcp`](../../internal/mcp/) | Built: every command as a tool, Streamable HTTP on loopback, bearer token, Host and Origin checks | Stateless HTTP (one session per request); the token lives in `<user config dir>/Cogpit/mcp-token` and changes with the port on every start; no stdio bridge |
| [`internal/ui/fyneui`](../../internal/ui/fyneui/) and its [`yamlview`](../../internal/ui/fyneui/yamlview/) and [`termview`](../../internal/ui/fyneui/termview/) | Built: shell after Lens's layout, theme with status tokens, context picker, confirmation dialog, the YAML view, terminal and log widgets, a jank meter (`--perf`); row menus (⋮ and right-click) and the drawer's toolbar from the registry, schema-generated parameter dialogs, links and coloured parts in cells, Cogpit's own tooltips (Fyne 2.8 has none); third run: Cogpit's own title bar on macOS through a cgo bridge to the `NSWindow` ([`titlebar_darwin.go`](../../internal/ui/fyneui/titlebar_darwin.go); `COGPIT_SYSTEM_TITLEBAR=1` keeps the system one), the column menu, keyboard navigation, fit-to-content, CSV download, the Workloads Overview, item actions in the drawer (Forward…, Show), the port-forward dialog, the Helm banner, log tabs with their toolbar; fourth run (2026-10-12): the kubeconfig dialogs (paste, sources, the credential plugin's approval) and the picker's "Add Kubeconfigs" ([`kubeconfigs.go`](../../internal/ui/fyneui/kubeconfigs.go)) | The YAML view is editable in Edit and Create resource tabs, whose text lives in the engine (`editor.get`, `editor.setText`); edits are applied as updates (PUT), not patches; the terminal widget is a patched copy of fyne-io/terminal ([`third_party/fyne-terminal`](../../third_party/fyne-terminal/PATCHES.md)) behind an escape filter, because the pinned upstream froze or crashed on container output |
| [`cmd/cogpit`](../../cmd/cogpit/) | Built: wires the packages, the drain journal (`<user config dir>/Cogpit/drains.json`) and the Helm journal (`helm.json`); the kubeconfig sources (`kubeconfigs.json`, `kubeconfigs/`, `exec-approvals.json`, fourth run) and their watcher; flags `--kubeconfig`, `--context`, `--demo`, `--mcp-addr`, `--no-mcp`, `--perf`, `--theme` | client-go goroutine panics are logged instead of ending the app |
| [`internal/helm`](../../internal/helm/) (L2) | Built in the third run: release payloads decoded (bounded, fuzzed), the Releases profile as a virtual kind over the `owner=helm` Secrets with a bounded memo, rollback and uninstall through the Helm SDK on the connection's own client configuration; fourth run (2026-10-12): Lens's release drawer (Properties, values and notes as private items, resources by kind), the upgrade with the release's own chart, values read from the cached Secret (`ReleaseValues`), values scrubbed from errors and the SDK's log (`ScrubValues`, `RedactSDKLog`) | Install, repositories and charts not built, so an upgrade installs the release's own chart version only; no action can be cancelled: rollback and uninstall take no context, and the upgrade's context is not passed on (§8) |
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
so an agent's Scale the user accepted is bound like Scale to zero (`TestRestartAndScaleCarryTheUID`). A command can ask
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
  For that a follow stream always asks for timestamps and strips them unless the tab shows them.
  Not covered: a resumption names the container, not its instance, so if the cache shows a
  restart only after the backoff, the new instance's first lines come twice (inferred, not
  observed).
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
`TestRevealIsGUIOnly`); agents read the drawer, the YAML (`model.Redact`), the list's cells, the
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
at most `Options.HandoverTimeout` (one minute), then stops; a covered scope that is only warm ends
with its cover instead of starting a watch to stay warm. A selected scope with its own informer moves
onto a cover that starts later, once the cover has caught up, and its own informer stops.

As built: a selected scope's handles read a view, the objects its subscribers have been told
about, which an event handler on the serving informer keeps (client-go's `AddEventHandler`: a
consistent snapshot of the store, then every change). A handover registers a handler on the new
informer that fills a staging copy, and moves the view onto it once that handler has synced and has
delivered a resourceVersion at least as new as every object the view holds; the subscribers then
get the difference as events, and the old informer's events are ignored. With comparable
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
- `object.edit` and `object.delete` do not apply to local and virtual kinds
  (`KindProfile.APIObject`); such kinds bring their own commands. `kinds.list` marks them `local`
  and `virtual` (2026-10-12), and the GUI shows no "+" (Create resource) on their views.

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

### Kubeconfig sources (built 2026-10-12)

Besides the default kubeconfig files (`--kubeconfig`, else `$KUBECONFIG`, else `~/.kube/config`,
merged as kubectl merges them), the user adds sources in Cogpit
([`engine/kubeconfigs.go`](../../internal/engine/kubeconfigs.go), the commands in
[`commands/kubeconfigs.go`](../../internal/engine/commands/kubeconfigs.go)):

- **A kubeconfig file** (`kubeconfig.addFile`): read, re-read when it changes, never written.
- **A pasted kubeconfig** (`kubeconfig.paste`): checked (at most 1 MiB, a valid document with a
  context; errors never echo the text) and stored as Cogpit's own file
  `<user config dir>/Cogpit/kubeconfigs/pasted-<16 hex>.yaml`, mode 0600 in a 0700 folder inside
  the 0700 `Cogpit` folder (a folder found more open is tightened), written atomically, never
  through a symlink. Removing the source deletes the copy, and only a regular file in that folder.
  It is never written into the user's kubeconfig files.
- **A sync folder** (`kubeconfig.addFolder`): every file in it (not in subfolders; hidden files,
  `~` backups, `.swp` and `.tmp` skipped; at most 500 files of at most 1 MiB) that parses and
  defines a context. `Session.WatchKubeconfigs` compares the sources' file sizes and modification
  times every 2 seconds (polling, no file-system notifications: `fsnotify` is not a direct
  dependency) and emits `EventContexts` when they change; the picker lists again.

The list of sources is `<user config dir>/Cogpit/kubeconfigs.json` (0600). Each source file is
read **on its own**, never merged with another file: a context in a dropped file can then never
borrow the user entry of the user's own kubeconfig and send its credential to the dropped file's
server. Its contexts are offered as `<context> [<label>]`: the label is the file's name
(`dev.yaml`, `dev.yaml (2)` for a second file of that name), `pasted <n>`, or the folder's name and
the file's (`kubeconfigs/team.yaml`). That id is `UIState.Context` and the context of every
confirmation, audit line and journal entry; the connection reads the one file
(`cluster.Options.KubeconfigPaths`), and a terminal binds kubectl to the context's name in that
file (`Conn.Context()`). A source id is resolved before the default files' names, so a default
context that happens to be named like one is hidden, with a problem saying so.

**Credential plugins of sources run only after the user allowed them** (trust-boundaries H-1). The
connector passes `KubeconfigSources.Policy` as `cluster.Options.ExecPolicy`; `cluster.Connect`
asks it with the plugin described from the very configuration the client is built from
(`rest.Config.ExecProvider`), before any client exists, so nothing runs when it refuses and a file
changed between listing and connecting cannot slip another command past it. An approval
(`<user config dir>/Cogpit/exec-approvals.json`, 0600) is recorded per file and context with a
SHA-256 fingerprint of the file, context, user, server URL (with any user information), CA data or
file, `insecure-skip-tls-verify`, TLS server name, proxy URL, API version, command, arguments,
environment names and values, interactive mode and `provideClusterInfo`
([`kubeconfig/exec.go`](../../internal/kubeconfig/exec.go)); any change asks again. The plugins
of the default kubeconfig files run as kubectl runs them, without an approval.

Adding a file or folder, pasting, removing a source and allowing a plugin are refused for every
origin but the GUI (`engine.ErrGUIOnly`, checked in each command's `run`), as `secret.reveal` is:
a kubeconfig can name a program to run on the user's computer, so an agent must not be able to
put one there or let one run. Agents may list the sources (`kubeconfig.sources`) and read a
plugin's description (`kubeconfig.execPlugin`, without its arguments); the audit log records an
agent's attempt to paste by the size of its parameters only.

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
"Release: <name>" (`KindProfile.DetailTitle`), Properties with Chart (an Upgrade button beside it,
an item action), Updated ("68s ago (12 Oct 2026, 8:30:16 CEST)"), Namespace, Version (the chart's
version, inferred) and Status as Lens writes it ("Deployed"; the other states capitalized the same
way, inferred), then the user-supplied values, the chart's notes (both `Private`: agents read
`model.Withheld`; cut for the drawer to 16 KiB or 200 lines) and the resources of the newest
revision's manifest, one table of Name and Namespace per kind with links for built-in kinds. As
for every virtual kind, the engine adds no generic metadata and no events, and the drawer opens no
events watch (§4.4). The History table and the redacted manifest of the third run are gone: Lens's
drawer has neither; the history is `helm.release.history`, the YAML tab Cogpit's summary.
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
  `revision` they are based on (the drawer's Save, agents), and `version` (only the release's own).
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
- **The Rollback dialog is the confirmation** (GUI, [`helmrollback.go`](../../internal/ui/fyneui/helmrollback.go)).
  Rollback stays destructive and the engine asks the GUI's confirmer for every origin. Lens's dialog
  (a REVISION dropdown, newest first, the current one selected) names the release, its namespace and
  its context, and when the user picks Rollback hands the confirmer the newest release Secret's UID
  it showed; the engine's confirmation of exactly that command, from the GUI origin, on that release
  and context, within ten seconds, takes that answer instead of a second dialog, and one whose UID
  differs (the release moved while the dialog was open) is refused. Agents' requests carry their own
  origin and never match. The banner's and the agents' rollbacks are confirmed as before.

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
| 2026-10-12 | minor | §10: the patched terminal widget's two goroutines (parser on the reading one, screen and state on the main one) and its fuzz targets |
| 2026-10-12 | minor | Fourth run, the third run's leftovers: the journals shared by Cogpit processes (§8: a directory lock, an owner per process, entries running elsewhere); `kinds.list` marks local and virtual kinds (§5); the View menu and Refresh Access Reviews in the GUI (§4.2); live related-objects indexes shared with the drawer's fetches, the cluster-role rule on `Listed`, `ui.navigate`'s filter check with related cells (§4.4); workload logs resume dropped streams and forget ended instances (§4.4); Restart's and Scale's patches carry the UID, mutating confirmations record cached UIDs (§4.2) |
| 2026-10-12 | minor | Fourth run (agent D): kubeconfig sources (added files, pasted kubeconfigs, sync folders), each file read on its own, and credential plugins of sources run only after the user allowed them (§6; current state rows) |
| 2026-10-12 | minor | Fourth run, Helm: the upgrade as built (§8: what binds, the SDK's context, `ErrUnchanged`, values scrubbed, the Rollback dialog as the confirmation); Lens's release drawer; virtual kinds' drawers without metadata and events, `DetailTitle` (§4.4) |
| 2026-10-12 | minor | Fourth run, agent A: covering scopes built (§5, how a handover stays exact, the measured budgets); the Secrets view on a full watch, its drawer from the cache without a GET; `resources.find` without a kind skips uncached kinds that hold Secret values; the last-applied annotation re-checked and kept dropped |
| 2026-10-12 | minor | §4.4: the related objects' generation changes when the indexed kinds finish their initial list; §12: a stopped shell drops the work it queued for the main goroutine |
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
