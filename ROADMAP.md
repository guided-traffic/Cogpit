# Cogpit roadmap

**Stage: pre-alpha** · a local MVP runs on macOS; every alpha-gate workflow exists in part · next: alpha on macOS, ~135–370 agent-hours away (seven core workflows end to end)

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-12 · **Change:** minor (the fifth run in the progress log and the upcoming work after it; breadth counted by `make parity`, unchanged at ~56%; Helm releases and metrics re-estimated, the other estimates not re-assessed) · **Target:** Lens Desktop 2026.10.51501

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
| **Feature breadth** | **~76%** (was ~56%) | measured by `make parity`: 63 of 64 views (ten of them the Gateway API views, generic), 37 of 39 actions, 109 of 120 interface rows (pasting kubeconfigs is not a target, D-21); 9 rows done, the rest partial |
| **Ready for real work** | **~26%** (was ~23%) | estimated |
| Mainstream practitioner | ~25% (was ~24%) | estimated |
| Essentials user | ~34% (was ~32%) | estimated |
| Remaining to **alpha** (macOS) | **~135–370 h** | estimated (was ~150–400) |
| Remaining to **beta** (macOS, with the Flux CD and Argo CD views) | **~425–995 h** | estimated (was ~440–1,020) |
| Remaining to **full parity** (free features, macOS) | **~420–1,040 h** | estimated (was ~430–1,060) |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~620–1,450 h** | estimated (was ~640–1,480) |
| **Beyond Lens** (MCP control of UI and cluster, built-in agent) | **~50–110 h** | estimated, unchanged; not part of the parity numbers; 13 of 38 rows done, 9 partial (the CLI and its stdio bridge, 2026-10-13) |

### Readiness by audience

| Audience | Ready % | Agent-hours to ~95% | Work that dominates |
|---|---:|---:|---|
| Full Lens target | ~17% | 620–1,450 | Everything below, plus paid-plan features, Windows and Linux |
| Mainstream practitioner (macOS) | ~25% | 420–1,040 | The interface, stability, metrics, the depth of views and actions, Helm install and charts |
| Essentials user (macOS) | ~34% | 90–280 | Cloud credential plugins verified on EKS, GKE and AKS, end-to-end checks on real clusters |

**Why pre-alpha.** The [alpha gate](docs/parity/roadmap.md#alpha-gate) needs seven core workflows end
to end on macOS: connect, triage a failing pod, change a workload, reach a service, manage Helm
releases in the cluster, maintain a node, use kubectl in a terminal. Since the fourth run every
one exists, Helm upgrade and adding and syncing kubeconfigs included, on kind through the
engine and offscreen in the GUI; none has been walked through by a person on a real cluster, and
no cloud credential plugin is verified.

## By dimension

| Dimension | % ready | Hours to full | Doc |
|---|---:|---:|---|
| Resource views and actions | 29% | 50–170 | [resource-parity](docs/parity/resource-parity.md) |
| Interface fidelity and feel | 23% | 100–275 | [ui-parity](docs/parity/ui-parity.md) |
| Cluster connectivity and authentication | 28% | 10–28 + human | [platform-parity](docs/parity/platform-parity.md#cluster-authentication-and-connectivity) |
| Logs, terminal, shells, port forwarding | 47% | 18–46 | [ui-parity](docs/parity/ui-parity.md#logs) |
| Performance and API load | 25% (P2, P9–P11 measured) | 28–66 | [testing](docs/developer/testing.md#performance-budgets-proposed-not-measured) |
| Stability | 11% | 100–200 | [gaps](docs/parity/gaps.md#g16-stability-and-the-user-bug-backlog) |
| Helm releases and metrics | 32% (was 25%; metrics in the lists and the drawer from the Metrics Server and Prometheus) | 42–95 | [gaps](docs/parity/gaps.md#g11-helm) |
| Platforms and distribution (macOS, then Windows and Linux) | 5% | 62–125 + human | [platform-parity](docs/parity/platform-parity.md#operating-systems-and-packages) |
| Paid-plan integrations (Flux CD and Argo CD in the beta) | 0% | 165–335 + human | [resource-parity](docs/parity/resource-parity.md#paid-lens-desktop-views) |
| Foundation: workspace, cache, engine, CI | built in part | 29–74 | [architecture](docs/developer/architecture.md#current-state-2026-10-12) |

## Upcoming

| Rank | Milestone | Estimate |
|---|---|---:|
| 1 | The owner's open requirements: the hotbar (G13); metrics (G12): a status line per provider, history from Prometheus then live values from the Metrics Server, the override's settings page, kube-prometheus-stack checked, the cluster overview with its charts | 30–70 h |
| 2 | Walk the workflows in the GUI on kind and a real cluster: change a workload, maintain a node, reach a service, Helm upgrade, rollback and uninstall, adding and syncing kubeconfigs, the title bar | human |
| 3 | Helm install and charts (M6); credential plugins verified on EKS, GKE and AKS (M1) | 25–55 h + human |
| 4 | Finish M0 and M2: the generated pages and scorecard, the create dialogs, captures of inferred rows; the built-in chat after the alpha (G21, D-10) | 80–200 h + human |

Detail: [docs/parity/roadmap.md](docs/parity/roadmap.md).

## Progress log

| Date | Entry |
|---|---|
| 2026-10-12 | Fifth implementation run (the lead with at most two agents at a time after the fourth run's memory incident, then the owner's smoke test in the evening). Finished the fourth run: its review (13 findings, each verified with a failing test, then fixed), pasting kubeconfigs removed (D-21), the owner's decisions D-17 to D-20 and D-22 to D-28 built (token files and plugin programs bound by their content, Secrets from YAML in the GUI only, port forwards that fail at once, Solarized colours, the C helm logo, copying from the drawer, the default kubeconfig as sources, one plugin approval per file), Lens's Helm and Kubeconfig Syncs captures followed, the terminal widget's fixes offered upstream (fyne-io/terminal #155–#158). New: metrics from the Metrics Server and Prometheus (CPU and Memory columns; the drawer's charts per pod and container in four variants, ranges to 30 days, a hover panel), checked against a real Prometheus on kind; the drawer's and the sidebar's widths set by the user; a pod's containers as squares; crash reports; the external MCP server switched in the status bar. The smoke test found a crash at every quit on macOS (fixed). Measured: breadth ~56% (55 of 64 views, 29 of 39 actions, 55 of 120 interface rows); 79 commands and MCP tools (was 76) |
| 2026-10-12 | Fourth implementation run (Phase 0 by the lead, five agents, merged one branch at a time; stopped once when the Mac ran out of memory, then at most two agents at a time with throttled builds). Before it: the dependencies brought up to date (44 Renovate pull requests, held back by flaky tests, closed; three updates that do not compile held back by rules) and ten flaky tests fixed, four of them product bugs. Built: `make parity` in CI (it reproduced every breadth number and found six partial rows without evidence and a wrong platform count); Helm upgrade with Lens's drawer, Rollback dialog and Helm Upgrade tab, crash-safe; adding, pasting and syncing kubeconfigs, each read on its own, credential plugins run only once allowed; the Secrets view on a full watch with Helm's releases served from it; the journals locked between Cogpit processes; the terminal widget fuzzed alone with twelve fixes; live related-objects indexes; Restart and Scale bound to the UID in the patch. Measured: breadth ~56% (55 of 64 views, 29 of 39 actions, 55 of 121 interface rows); 76 commands and MCP tools (was 65) |
| 2026-10-11 | Third implementation run (Phase 0 contracts by the lead, eight agents in parallel, merged one branch at a time, a review). Built: port forwarding (pod and service, 127.0.0.1 only, the captured dialog, the view, reconnects) and Helm releases from the `owner=helm` Secrets (history, values, manifest; rollback and uninstall, crash-safe through a journal), the two alpha-gate workflows that did not exist; the Secrets view from metadata only with one GET per opened Secret, values masked and revealed in the GUI only (H-8); the Config, Access Control, Network and Storage views (28 views); access reviews that grey commands and hide kinds; attach, Restart of Daemon and Stateful Sets, Replica Set scale, logs of a workload's pods and the log options; Cogpit's own title bar on macOS, column choice, keyboard navigation, CSV export and the Workloads Overview. Measured: breadth ~54% (55 of 64 views, 28 of 39 actions, 52 of 121 interface rows); 65 commands and MCP tools (was 41) |
| 2026-10-11 | Second implementation run (seven agents in parallel, then merge, CI, review). CI is green on GitHub on every push: macOS lint, race tests and build, Windows and Linux compile checks, kind on 1.37, govulncheck, a coverage badge (81.7%), semantic-release as a dry run (no tag, no release). Built: 10 more built-in views (workloads, Ingresses, Ingress Classes, Config Maps, PVCs, Definitions) and a view per custom resource kind from a watch on the definitions, with the sidebar's Custom Resources and Gateway API groups; editing with diff, dry run and a resourceVersion precondition, create from Cogpit's templates, Delete for every kind; Cordon, Uncordon, a crash-safe Drain (tested on a three-node kind cluster), Scale, Trigger, Suspend, Resume, Evict; row menus and the drawer from the registry, links, warning icons, tooltips, the drawer's Properties. Measured: breadth ~30% (27 of 64 views, 14 of 39 actions, 37 of 121 interface rows); 41 commands and MCP tools |
| 2026-10-11 | Gate S1 and S4 measured again in a real window with view switches, typing and a CPU profile: start-up stall is Fyne's per-window GL set-up (0.31 s) plus the first table fill (151–193 ms); view switches ≤ 42 ms; typing into 5 MB fits the frame. Owner: S1 passes with a reservation; S4 passes; D-1 decided for Fyne, with re-checks of S7 (CI) and S5 (Windows, Linux) |
| 2026-10-11 | Implementation started. A local MVP runs on macOS: the context picker (kubeconfig contexts and a synthetic demo cluster with 10,000 pods), six views from shared informers with snapshots coalesced to ten per second, the detail drawer with YAML, logs with follow, a cluster-bound terminal, a pod shell, Restart and Delete behind confirmation, and an MCP server whose `ui_navigate` drives the GUI. The Fyne spike ran with it ([results](docs/developer/spike-fyne.md#results)). Measured: breadth ~12%; sort of 10,000 rows 10 ms (P2); 0 API requests for an idle view, a warm reopen and the agent showcase (P9–P11). S1 in a real window: no frame over 16 ms after the first second, but a start-up stall of 43–460 ms. Found and fixed in a patched copy: the terminal widget froze or crashed on container output (H-19). Claude Code drove the GUI to the Service with ClusterIP 10.96.100.100 over MCP |
| 2026-10-11 | Plan revised: Go instead of Rust, Fyne behind a spike gate, macOS first, cache-first data flow, MCP control of UI and cluster, a built-in agent on OpenAI-compatible endpoints, one-click updates. Nothing built |
| 2026-10-11 | Repository started. Documentation method adopted from PhotoCraft; target fixed at Lens Desktop 2026.10.51501; baseline 0% |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-12 | minor | The fifth run in the progress log; upcoming after it; Helm releases and metrics re-estimated; breadth unchanged |
| 2026-10-12 | major | Fourth measurement after the fourth implementation run, counted by `make parity` |
| 2026-10-12 | minor | Upcoming: the owner's decisions D-13 (revised), D-14, D-15, D-16 |
| 2026-10-11 | major | Third measurement after the third implementation run |
| 2026-10-11 | major | Second measurement after the second implementation run |
| 2026-10-11 | major | First measurement after the local MVP |
| 2026-10-11 | major | Go plan, macOS first, beyond-Lens targets separated |
| 2026-10-11 | major | First version |
