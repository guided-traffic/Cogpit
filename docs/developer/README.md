# Developer documentation

For contributors and agents. Start with [`AGENTS.md`](../../AGENTS.md) for the rules in short, then
this page.

## What has to be in your head first

1. **A local MVP is built** (2026-10-11) and was extended the same day: 16 built-in views and the
   custom resource views from the cache, drawer and row menus, editing, node and workload actions,
   logs, terminal, pod shell, CI, MCP control (41 commands); it doubled as the
   [Fyne spike](spike-fyne.md#results). What exists and
   where it differs from the plan: [architecture › Current state](architecture.md#current-state-2026-10-12).
2. **Go and Fyne.** The engine is Go on client-go, the kubectl libraries and the Helm SDK. The UI
   is Fyne: the [spike](spike-fyne.md) passed and D-1 is decided; re-checks on Windows and Linux
   can still move one platform's UI to a Rust/egui process ([decisions](decisions.md#d-1-ui-toolkit)).
3. **Cache first.** Only the cache layer lists and watches; the UI and agents read from the cache
   ([architecture §5](architecture.md#5-cache)).
4. **Everything is a command, and every command is an MCP tool.** The GUI, the CLI, external agents
   and the built-in agent dispatch the same commands; UI state lives in the engine, so agents can
   see and drive the UI.
5. **Never surprise a cluster.** Confirmations and verbs are enforced in the engine for every
   origin ([conventions](conventions.md#never-surprise-a-cluster)).

## Pages

| Page | Topic |
|---|---|
| [`architecture.md`](architecture.md) | Goals, principles, packages, layering, the UI seam, the cache, MCP and the agent, runtime model |
| [`conventions.md`](conventions.md) | Clean-room, never surprise a cluster, cache first, crash stance, secrets, style, commits |
| [`extending.md`](extending.md) | Checklists: a command, a kind profile, a cache consumer, a package |
| [`testing.md`](testing.md) | Test tiers, fixtures, performance and API-load budgets |
| [`releasing.md`](releasing.md) | CI jobs, coverage badge, Renovate, the semantic-release dry run, release flow, artifacts per platform, signed updates |
| [`decisions.md`](decisions.md) | Decisions taken and open |
| [`spike-fyne.md`](spike-fyne.md) | The UI toolkit gate before M2 |

Where Cogpit stands against Lens: [`../parity/`](../parity/README.md). Security architecture:
[`../security/`](../security/README.md).

## Repository layout

```text
cogpit/
├─ AGENTS.md, CLAUDE.md, ATTRIBUTION.md, README.md, ROADMAP.md, SECURITY.md, LICENSE
├─ go.mod                     module github.com/guided-traffic/cogpit
├─ Makefile                   build, run, run-demo, test, test-unit-race, test-kind, bench, lint, parity, kind-up, kind-down, kind-prometheus, app
├─ .golangci.yml              linters and the depguard layering rules
├─ .github/                   CI workflows (build, release, renovate), renovate.json, the coverage badge
├─ .releaserc.json            semantic-release config; package.json pins its npm tooling (CI only)
├─ cmd/cogpit/                desktop app (Fyne, cgo)
├─ cmd/cogpit-cli/            headless CLI and MCP stdio bridge (pure Go)
├─ internal/                  packages, layered L0–L4 (architecture.md §2–§3)
├─ hack/kind/                 kind cluster config, seed manifests (make kind-up), Prometheus values (make kind-prometheus)
├─ hack/icon/                 generator of the app icon (make icon; internal/ui/logo draws it)
├─ hack/parity/               make parity: checks the parity checklists and counts what they report
├─ hack/readmecmds/           the README's table of commands, from the registry
├─ packaging/                 icon.png; per-OS packaging later
├─ tools/                     (planned) generators of the parity checklist page and the scorecard
└─ docs/
   ├─ developer/              this directory
   ├─ parity/                 progress against Lens
   └─ security/               security architecture, one page per perspective
```

## Packages

| Package | Layer | Responsibility |
|---|---|---|
| `internal/model` | L0 | Kind profiles, object refs, rows, status tokens; pure |
| `internal/kubeconfig` | L0 | Kubeconfig discovery, merge and validation from bytes |
| `internal/cluster` | L1 | Connections: client-go config, auth, cached discovery, access reviews |
| `internal/cache` | L1 | Shared informers, reference counting, warm period, transforms, request counters |
| `internal/terminal` | L1 | PTY sessions: local shell with its own kubeconfig, pod exec and attach |
| `internal/helm` | L2 | Helm SDK, crash-safe release actions (not built) |
| `internal/metrics` | L2 | Metrics providers (the Metrics Server built, Prometheus next), the hub that polls while subscribed, its TTL cache and series ([architecture §9](architecture.md#9-metrics-internalmetrics)) |
| `internal/demo` | L2 | The synthetic demo cluster behind a `cache.Source`, and its synthetic Metrics Server (a `metrics.Provider`: demo and metrics are both L2, so the import is allowed) |
| `internal/engine`, `internal/engine/commands` | L3 | Session, command registry, UI state, snapshots, drawer, streams, confirmation, audit log; the commands one file per area |
| `internal/mcp` | L4 | MCP server over the registry |
| `internal/agent` | L4 | Built-in agent with an OpenAI-compatible endpoint (not built) |
| `internal/update` | L4 | Update check, verification and install (not built) |
| `internal/ui/fyneui` | L4 | The Fyne shell; `yamlview` the YAML viewer and editor, `termview` the terminal and log widgets |
| `internal/ui/logo` | L4 | The C helm logo ([D-24](decisions.md#d-24-logo)) rasterised for the app icon (`hack/icon`) and the header's mark; imports no Fyne |
| `internal/testkit` | – | In-memory source, fake API server, object builders; `integration/` holds cross-package and kind tests |

## Core flows, one fact each

- **Connect:** the user picks a context; `cluster` builds a client-go config from the kubeconfig,
  runs cached discovery and access reviews, and the sidebar shows the kinds the cluster serves and
  the user may list.
- **Browse:** a view acquires a cache handle; the informer lists once and then watches; the engine
  turns cache changes into snapshots, at most ten per second per view.
- **Act:** a menu item, shortcut, CLI call or agent tool call dispatches a command; the engine
  checks `enabled`, permissions and the origin's confirmation rule, then runs it.
- **Agent:** an agent reads `ui_state` to know what the user sees, finds objects in the cache and
  calls `ui_navigate` to show them.

## Build, test, lint

Through the `Makefile`, as in the team's other repositories (`make help` lists every target):

| Task | Command |
|---|---|
| Build | `make build` |
| Run the app | `make run` (kind cluster), `make run-demo` (demo cluster), `bin/cogpit` (your kubeconfig) |
| Unit tests with the race detector | `make test-unit-race` |
| Cluster tests | `make kind-up && make test-kind` |
| Benchmarks of the gate | `make bench` |
| Lint, layering included | `make lint` (golangci-lint v2.14.0, installed into `bin/`) |
| Package for macOS | `make app` |
| Check the parity checklists and their numbers | `make parity` ([rules](../parity/README.md#make-parity)) |
| Regenerate the parity checklist page and the scorecard | not built yet (gap G1) |
