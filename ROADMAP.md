# Cogpit roadmap

**Stage: pre-alpha** · next: alpha on macOS, ~400–950 agent-hours away (seven core workflows end to end; nothing is built yet)

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (Go plan: Fyne spike gate, macOS first, cache-first data flow, MCP control and built-in agent as targets beyond Lens) · **Target:** Lens Desktop 2026.10.51501

Cogpit's one-page summary of how close it is to Lens Desktop 2026.10.51501, the Kubernetes IDE it
reimplements as a native desktop client in Go, for macOS first and Windows and Linux after the
macOS beta. Detail: [target-app parity](docs/parity/target-app-parity.md) (target, scope, weights,
evidence), [gaps](docs/parity/gaps.md) (the ranked work list),
[roadmap](docs/parity/roadmap.md) (alpha gate, milestones, current focus), the checklists for the
[interface](docs/parity/ui-parity.md), [resources](docs/parity/resource-parity.md),
[platforms](docs/parity/platform-parity.md) and what Cogpit does [beyond Lens](docs/parity/beyond-lens.md),
the [decisions](docs/developer/decisions.md) and the [architecture](docs/developer/architecture.md).

Hours are Opus 5.5 agent wall-clock hours for one agent working sequentially, estimated with
PhotoCraft's calibration until Cogpit has its own
([calibration](docs/parity/target-app-parity.md#remaining-effort-and-calibration)). After the
serial foundation, about 80% of the work parallelizes.

## Headline numbers

| | Value | Kind |
|---|---|---|
| **Feature breadth** | **0%** | measured: 0 of 63 views, 40 actions, 111 interface rows |
| **Ready for real work** | **0%** | measured: nothing runs |
| Mainstream practitioner | 0% | measured |
| Essentials user | 0% | measured |
| Remaining to **alpha** (macOS) | **~400–950 h** | estimated |
| Remaining to **beta** (macOS, with the Flux CD and Argo CD views) | **~840–1,870 h** | estimated |
| Remaining to **full parity** (free features, macOS) | **~800–1,850 h** | estimated |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~1,000–2,300 h** | estimated |
| **Beyond Lens** (MCP control of UI and cluster, built-in agent) | **~70–140 h** | estimated; not part of the parity numbers |

### Readiness by audience

| Audience | Ready % | Agent-hours to ~95% | Work that dominates |
|---|---:|---:|---|
| Full Lens target | 0% | 1,000–2,300 | Everything below, plus paid-plan features, Windows and Linux |
| Mainstream practitioner (macOS) | 0% | 800–1,850 | Resource views and actions, the interface, logs and terminal |
| Essentials user (macOS) | 0% | 300–700 | Foundation, the spike, a dozen core views and actions, logs and shell, the editor |

**Why pre-alpha.** The repository holds documentation only, and implementation waits for the
owner's go-ahead. The [alpha gate](docs/parity/roadmap.md#alpha-gate) needs seven core workflows end
to end on macOS: connect, triage a failing pod, change a workload, reach a service, manage Helm
releases in the cluster, maintain a node, use kubectl in a terminal.

## By dimension

| Dimension | % ready | Hours to full | Doc |
|---|---:|---:|---|
| Resource views and actions | 0% | 200–590 | [resource-parity](docs/parity/resource-parity.md) |
| Interface fidelity and feel | 0% | 180–435 | [ui-parity](docs/parity/ui-parity.md) |
| Cluster connectivity and authentication | 0% | 25–50 + human | [platform-parity](docs/parity/platform-parity.md#cluster-authentication-and-connectivity) |
| Logs, terminal, shells, port forwarding | 0% | 55–110 | [ui-parity](docs/parity/ui-parity.md#logs) |
| Performance and API load | 0% (no budget measured) | 40–80 | [testing](docs/developer/testing.md#performance-budgets-proposed-not-measured) |
| Stability | 0% | 100–200 | [gaps](docs/parity/gaps.md#g16-stability-and-the-user-bug-backlog) |
| Helm releases and metrics | 0% | 70–140 | [gaps](docs/parity/gaps.md#g11-helm) |
| Platforms and distribution (macOS, then Windows and Linux) | 0% | 65–130 + human | [platform-parity](docs/parity/platform-parity.md#operating-systems-and-packages) |
| Paid-plan integrations (Flux CD and Argo CD in the beta) | 0% | 165–335 + human | [resource-parity](docs/parity/resource-parity.md#paid-lens-desktop-views) |
| Foundation: workspace, cache, engine | 0% | 100–200 | [architecture](docs/developer/architecture.md) |

## Upcoming

| Rank | Milestone | Estimate |
|---|---|---:|
| 1 | The owner's go-ahead to start implementation | – |
| 2 | M0 Skeleton: `go.mod`, Makefile, lint with layering, CI, generators | 20–40 h |
| 3 | Fyne spike: table, YAML editor, terminal, MCP control; decides the UI toolkit | 16–24 h |
| 4 | M1 Connect: kubeconfigs, authentication, discovery, access reviews, first CLI commands | 25–50 h + human |
| 5 | M2 Browse: cache, engine, UI shell, about 20 core views, MCP navigation | 180–420 h |

Detail: [docs/parity/roadmap.md](docs/parity/roadmap.md).

## Progress log

| Date | Entry |
|---|---|
| 2026-10-11 | Plan revised: Go instead of Rust, Fyne behind a spike gate, macOS first, cache-first data flow, MCP control of UI and cluster, a built-in agent on OpenAI-compatible endpoints, one-click updates. Nothing built |
| 2026-10-11 | Repository started. Documentation method adopted from PhotoCraft; target fixed at Lens Desktop 2026.10.51501; baseline 0% |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | Go plan, macOS first, beyond-Lens targets separated |
| 2026-10-11 | major | First version |
