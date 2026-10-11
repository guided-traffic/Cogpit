# Cogpit roadmap

**Stage: pre-alpha** · a local MVP runs on macOS · next: alpha on macOS, ~250–700 agent-hours away (seven core workflows end to end)

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (first measurement after the local MVP) · **Target:** Lens Desktop 2026.10.51501

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
| **Feature breadth** | **~12%** | measured: 6 of 64 views, 4 of 40 actions, 25 of 117 interface rows, all partial |
| **Ready for real work** | **~10%** | estimated |
| Mainstream practitioner | ~8% | estimated |
| Essentials user | ~23% | estimated |
| Remaining to **alpha** (macOS) | **~250–700 h** | estimated (was ~400–950) |
| Remaining to **beta** (macOS, with the Flux CD and Argo CD views) | **~680–1,610 h** | estimated (was ~840–1,870) |
| Remaining to **full parity** (free features, macOS) | **~670–1,650 h** | estimated (was ~800–1,850) |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~870–2,060 h** | estimated (was ~1,000–2,300) |
| **Beyond Lens** (MCP control of UI and cluster, built-in agent) | **~50–110 h** | estimated (was ~70–140); not part of the parity numbers; 9 of 34 rows done |

### Readiness by audience

| Audience | Ready % | Agent-hours to ~95% | Work that dominates |
|---|---:|---:|---|
| Full Lens target | ~7% | 870–2,060 | Everything below, plus paid-plan features, Windows and Linux |
| Mainstream practitioner (macOS) | ~8% | 670–1,650 | Resource views and actions, the interface, logs and terminal |
| Essentials user (macOS) | ~23% | 200–520 | Editing YAML, more core views and actions, log options, cloud credential plugins |

**Why pre-alpha.** The [alpha gate](docs/parity/roadmap.md#alpha-gate) needs seven core workflows end
to end on macOS: connect, triage a failing pod, change a workload, reach a service, manage Helm
releases in the cluster, maintain a node, use kubectl in a terminal. The MVP covers connecting,
triaging and kubectl in part; editing, port forwarding, Helm and node maintenance do not exist.

## By dimension

| Dimension | % ready | Hours to full | Doc |
|---|---:|---:|---|
| Resource views and actions | 5% | 182–555 | [resource-parity](docs/parity/resource-parity.md) |
| Interface fidelity and feel | 8% | 152–390 | [ui-parity](docs/parity/ui-parity.md) |
| Cluster connectivity and authentication | 15% | 18–40 + human | [platform-parity](docs/parity/platform-parity.md#cluster-authentication-and-connectivity) |
| Logs, terminal, shells, port forwarding | 20% | 40–90 | [ui-parity](docs/parity/ui-parity.md#logs) |
| Performance and API load | 25% (P2, P9–P11 measured) | 30–70 | [testing](docs/developer/testing.md#performance-budgets-proposed-not-measured) |
| Stability | 5% | 100–200 | [gaps](docs/parity/gaps.md#g16-stability-and-the-user-bug-backlog) |
| Helm releases and metrics | 0% | 70–140 | [gaps](docs/parity/gaps.md#g11-helm) |
| Platforms and distribution (macOS, then Windows and Linux) | 5% | 62–125 + human | [platform-parity](docs/parity/platform-parity.md#operating-systems-and-packages) |
| Paid-plan integrations (Flux CD and Argo CD in the beta) | 0% | 165–335 + human | [resource-parity](docs/parity/resource-parity.md#paid-lens-desktop-views) |
| Foundation: workspace, cache, engine | built in part | 52–118 | [architecture](docs/developer/architecture.md#current-state-2026-10-11) |

## Upcoming

| Rank | Milestone | Estimate |
|---|---|---:|
| 1 | The owner's decision on keeping the patched terminal widget ([H-19](docs/security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)) | – |
| 2 | Finish M0: CI, generators, semantic-release | 12–28 h |
| 3 | Editing: the YAML editor in an Edit tab with diff, dry-run and apply | 27–60 h |
| 4 | Finish M1: pasting and syncing kubeconfigs, access reviews, first CLI commands | 18–40 h + human |
| 5 | Finish M2: about 20 core views, the generic custom-resource view | 130–330 h |

Detail: [docs/parity/roadmap.md](docs/parity/roadmap.md).

## Progress log

| Date | Entry |
|---|---|
| 2026-10-11 | Gate S1 and S4 measured again in a real window with view switches, typing and a CPU profile: start-up stall is Fyne's per-window GL set-up (0.31 s) plus the first table fill (151–193 ms); view switches ≤ 42 ms; typing into 5 MB fits the frame. Owner: S1 passes with a reservation; S4 passes; D-1 decided for Fyne, with re-checks of S7 (CI) and S5 (Windows, Linux) |
| 2026-10-11 | Implementation started. A local MVP runs on macOS: the context picker (kubeconfig contexts and a synthetic demo cluster with 10,000 pods), six views from shared informers with snapshots coalesced to ten per second, the detail drawer with YAML, logs with follow, a cluster-bound terminal, a pod shell, Restart and Delete behind confirmation, and an MCP server whose `ui_navigate` drives the GUI. The Fyne spike ran with it ([results](docs/developer/spike-fyne.md#results)). Measured: breadth ~12%; sort of 10,000 rows 10 ms (P2); 0 API requests for an idle view, a warm reopen and the agent showcase (P9–P11). S1 in a real window: no frame over 16 ms after the first second, but a start-up stall of 43–460 ms. Found and fixed in a patched copy: the terminal widget froze or crashed on container output (H-19). Claude Code drove the GUI to the Service with ClusterIP 10.96.100.100 over MCP |
| 2026-10-11 | Plan revised: Go instead of Rust, Fyne behind a spike gate, macOS first, cache-first data flow, MCP control of UI and cluster, a built-in agent on OpenAI-compatible endpoints, one-click updates. Nothing built |
| 2026-10-11 | Repository started. Documentation method adopted from PhotoCraft; target fixed at Lens Desktop 2026.10.51501; baseline 0% |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | First measurement after the local MVP |
| 2026-10-11 | major | Go plan, macOS first, beyond-Lens targets separated |
| 2026-10-11 | major | First version |
