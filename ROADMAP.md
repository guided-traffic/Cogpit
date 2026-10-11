# Cogpit roadmap

**Stage: pre-alpha** · a local MVP runs on macOS; every alpha-gate workflow exists in part · next: alpha on macOS, ~150–400 agent-hours away (seven core workflows end to end)

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (third measurement, after the third implementation run) · **Target:** Lens Desktop 2026.10.51501

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
| **Feature breadth** | **~54%** (was ~30%) | measured: 55 of 64 views (ten of them the Gateway API views, generic), 28 of 39 actions, 52 of 121 interface rows; 9 rows done, the rest partial |
| **Ready for real work** | **~23%** (was ~14%) | estimated |
| Mainstream practitioner | ~24% (was ~14%) | estimated |
| Essentials user | ~32% (was ~28%) | estimated |
| Remaining to **alpha** (macOS) | **~150–400 h** | estimated (was ~210–520) |
| Remaining to **beta** (macOS, with the Flux CD and Argo CD views) | **~440–1,020 h** | estimated (was ~580–1,360) |
| Remaining to **full parity** (free features, macOS) | **~430–1,060 h** | estimated (was ~570–1,400) |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~640–1,480 h** | estimated (was ~780–1,820) |
| **Beyond Lens** (MCP control of UI and cluster, built-in agent) | **~50–110 h** | estimated, unchanged; not part of the parity numbers; 10 of 34 rows done, 7 partial |

### Readiness by audience

| Audience | Ready % | Agent-hours to ~95% | Work that dominates |
|---|---:|---:|---|
| Full Lens target | ~16% | 640–1,480 | Everything below, plus paid-plan features, Windows and Linux |
| Mainstream practitioner (macOS) | ~24% | 430–1,060 | The interface, stability, metrics, the depth of views and actions, Helm upgrade and charts |
| Essentials user (macOS) | ~32% | 100–300 | Cloud credential plugins, adding and pasting kubeconfigs, end-to-end checks on real clusters |

**Why pre-alpha.** The [alpha gate](docs/parity/roadmap.md#alpha-gate) needs seven core workflows end
to end on macOS: connect, triage a failing pod, change a workload, reach a service, manage Helm
releases in the cluster, maintain a node, use kubectl in a terminal. Since the third run every one
works in part, on kind through the engine and offscreen in the GUI; none has been walked through
by a person on a real cluster, and Helm upgrade, adding kubeconfigs and the cloud credential
plugins are missing.

## By dimension

| Dimension | % ready | Hours to full | Doc |
|---|---:|---:|---|
| Resource views and actions | 28% | 50–170 | [resource-parity](docs/parity/resource-parity.md) |
| Interface fidelity and feel | 22% | 100–275 | [ui-parity](docs/parity/ui-parity.md) |
| Cluster connectivity and authentication | 18% | 14–34 + human | [platform-parity](docs/parity/platform-parity.md#cluster-authentication-and-connectivity) |
| Logs, terminal, shells, port forwarding | 45% | 18–46 | [ui-parity](docs/parity/ui-parity.md#logs) |
| Performance and API load | 25% (P2, P9–P11 measured) | 28–66 | [testing](docs/developer/testing.md#performance-budgets-proposed-not-measured) |
| Stability | 9% | 100–200 | [gaps](docs/parity/gaps.md#g16-stability-and-the-user-bug-backlog) |
| Helm releases and metrics | 15% | 60–125 | [gaps](docs/parity/gaps.md#g11-helm) |
| Platforms and distribution (macOS, then Windows and Linux) | 5% | 62–125 + human | [platform-parity](docs/parity/platform-parity.md#operating-systems-and-packages) |
| Paid-plan integrations (Flux CD and Argo CD in the beta) | 0% | 165–335 + human | [resource-parity](docs/parity/resource-parity.md#paid-lens-desktop-views) |
| Foundation: workspace, cache, engine, CI | built in part | 32–80 | [architecture](docs/developer/architecture.md#current-state-2026-10-11) |

## Upcoming

| Rank | Milestone | Estimate |
|---|---|---:|
| 1 | The owner's decisions: the patched terminal widget ([H-19](docs/security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)), the Secrets view's Type column ([H-8](docs/security/privilege-footprint.md#h-8-secret-data-arrives-with-the-list)), the progress generators, the first real release | – |
| 2 | Walk the workflows in the GUI on kind and a real cluster: change a workload, maintain a node, reach a service, Helm rollback and uninstall, the title bar | human |
| 3 | Helm upgrade, then install and charts (M6) | 20–45 h |
| 4 | Connect: adding, pasting and syncing kubeconfigs; credential plugins (M1) | 14–34 h + human |
| 5 | Finish M0 and M2: the generators, the cluster overview with metrics, the create dialogs, captures of inferred rows | 80–200 h + human |

Detail: [docs/parity/roadmap.md](docs/parity/roadmap.md).

## Progress log

| Date | Entry |
|---|---|
| 2026-10-11 | Third implementation run (Phase 0 contracts by the lead, eight agents in parallel, merged one branch at a time, a review). Built: port forwarding (pod and service, 127.0.0.1 only, the captured dialog, the view, reconnects) and Helm releases from the `owner=helm` Secrets (history, values, manifest; rollback and uninstall, crash-safe through a journal), the two alpha-gate workflows that did not exist; the Secrets view from metadata only with one GET per opened Secret, values masked and revealed in the GUI only (H-8); the Config, Access Control, Network and Storage views (28 views); access reviews that grey commands and hide kinds; attach, Restart of Daemon and Stateful Sets, Replica Set scale, logs of a workload's pods and the log options; Cogpit's own title bar on macOS, column choice, keyboard navigation, CSV export and the Workloads Overview. Measured: breadth ~54% (55 of 64 views, 28 of 39 actions, 52 of 121 interface rows); 65 commands and MCP tools (was 41) |
| 2026-10-11 | Second implementation run (seven agents in parallel, then merge, CI, review). CI is green on GitHub on every push: macOS lint, race tests and build, Windows and Linux compile checks, kind on 1.37, govulncheck, a coverage badge (81.7%), semantic-release as a dry run (no tag, no release). Built: 10 more built-in views (workloads, Ingresses, Ingress Classes, Config Maps, PVCs, Definitions) and a view per custom resource kind from a watch on the definitions, with the sidebar's Custom Resources and Gateway API groups; editing with diff, dry run and a resourceVersion precondition, create from Cogpit's templates, Delete for every kind; Cordon, Uncordon, a crash-safe Drain (tested on a three-node kind cluster), Scale, Trigger, Suspend, Resume, Evict; row menus and the drawer from the registry, links, warning icons, tooltips, the drawer's Properties. Measured: breadth ~30% (27 of 64 views, 14 of 39 actions, 37 of 121 interface rows); 41 commands and MCP tools |
| 2026-10-11 | Gate S1 and S4 measured again in a real window with view switches, typing and a CPU profile: start-up stall is Fyne's per-window GL set-up (0.31 s) plus the first table fill (151–193 ms); view switches ≤ 42 ms; typing into 5 MB fits the frame. Owner: S1 passes with a reservation; S4 passes; D-1 decided for Fyne, with re-checks of S7 (CI) and S5 (Windows, Linux) |
| 2026-10-11 | Implementation started. A local MVP runs on macOS: the context picker (kubeconfig contexts and a synthetic demo cluster with 10,000 pods), six views from shared informers with snapshots coalesced to ten per second, the detail drawer with YAML, logs with follow, a cluster-bound terminal, a pod shell, Restart and Delete behind confirmation, and an MCP server whose `ui_navigate` drives the GUI. The Fyne spike ran with it ([results](docs/developer/spike-fyne.md#results)). Measured: breadth ~12%; sort of 10,000 rows 10 ms (P2); 0 API requests for an idle view, a warm reopen and the agent showcase (P9–P11). S1 in a real window: no frame over 16 ms after the first second, but a start-up stall of 43–460 ms. Found and fixed in a patched copy: the terminal widget froze or crashed on container output (H-19). Claude Code drove the GUI to the Service with ClusterIP 10.96.100.100 over MCP |
| 2026-10-11 | Plan revised: Go instead of Rust, Fyne behind a spike gate, macOS first, cache-first data flow, MCP control of UI and cluster, a built-in agent on OpenAI-compatible endpoints, one-click updates. Nothing built |
| 2026-10-11 | Repository started. Documentation method adopted from PhotoCraft; target fixed at Lens Desktop 2026.10.51501; baseline 0% |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | Third measurement after the third implementation run |
| 2026-10-11 | major | Second measurement after the second implementation run |
| 2026-10-11 | major | First measurement after the local MVP |
| 2026-10-11 | major | Go plan, macOS first, beyond-Lens targets separated |
| 2026-10-11 | major | First version |
