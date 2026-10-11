# Developer documentation

For contributors and agents. Start with [`AGENTS.md`](../../AGENTS.md) for the rules in short, then
this page.

## What has to be in your head first

1. **Nothing is built yet.** As of 2026-10-11 the repository holds documentation only.
   Implementation starts with milestone M0 when the owner gives the go-ahead
   ([roadmap](../parity/roadmap.md#milestones)).
2. **Go, Fyne behind a gate.** The engine is Go on client-go, the kubectl libraries and the Helm
   SDK. The UI is Fyne if the [spike](spike-fyne.md) before M2 passes, a Rust/egui process if it
   fails ([decisions](decisions.md)).
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
| [`releasing.md`](releasing.md) | Release flow, artifacts per platform, signed updates |
| [`decisions.md`](decisions.md) | Decisions taken and open |
| [`spike-fyne.md`](spike-fyne.md) | The UI toolkit gate before M2 |

Where Cogpit stands against Lens: [`../parity/`](../parity/README.md). Security architecture:
[`../security/`](../security/README.md).

## Planned repository layout

```text
cogpit/
├─ AGENTS.md, CLAUDE.md, ATTRIBUTION.md, README.md, ROADMAP.md, SECURITY.md, LICENSE
├─ go.mod                     module github.com/guided-traffic/cogpit
├─ Makefile                   build, test, test-unit-race, lint, parity, scorecard, package
├─ .golangci.yml              linters and the depguard layering rules
├─ cmd/cogpit/                desktop app (Fyne, cgo)
├─ cmd/cogpit-cli/            headless CLI and MCP stdio bridge (pure Go)
├─ internal/                  packages, layered L0–L4 (architecture.md §2–§3)
├─ tools/                     generators for the parity checklist and the scorecard
├─ packaging/                 per-OS packaging scripts and manifests
└─ docs/
   ├─ developer/              this directory
   ├─ parity/                 progress against Lens
   └─ security/               security architecture, one page per perspective
```

## Planned packages

| Package | Layer | Responsibility |
|---|---|---|
| `internal/model` | L0 | Kind profiles, object refs, rows, status tokens; pure |
| `internal/kubeconfig` | L0 | Kubeconfig discovery, merge and validation from bytes |
| `internal/cluster` | L1 | Connections: client-go config, auth, cached discovery, access reviews |
| `internal/cache` | L1 | Shared informers, reference counting, warm period, transforms, request counters |
| `internal/terminal` | L1 | PTY sessions: local shell with its own kubeconfig, pod exec and attach |
| `internal/helm` | L2 | Helm SDK, crash-safe release actions |
| `internal/metrics` | L2 | Metrics providers, polling while visible |
| `internal/engine` | L3 | Session, command registry, UI state, jobs, events, confirmation, audit log |
| `internal/mcp` | L4 | MCP server over the registry |
| `internal/agent` | L4 | Built-in agent with an OpenAI-compatible endpoint |
| `internal/update` | L4 | Update check, verification and install |
| `internal/ui/fyneui` | L4 | The Fyne shell |
| `internal/testkit` | – | Fixtures, fake clients, kind and envtest helpers |

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

Nothing builds yet. Once M0 lands, through the `Makefile` as in the team's other repositories:

| Task | Command |
|---|---|
| Build | `make build` |
| Run the app | `make run` |
| Unit tests with the race detector | `make test-unit-race` |
| All test tiers you touched | `make test` |
| Lint, layering included | `make lint` |
| Regenerate parity checklist and scorecard | `make parity`, `make scorecard` |
| Package for this platform | `make package` |
