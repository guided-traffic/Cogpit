# Cogpit roadmap

**Stage: pre-alpha** · a local MVP runs on macOS; every alpha-gate workflow exists in part · next: alpha on macOS, ~110–320 agent-hours away (seven core workflows end to end, walked by a person)

> **Last reviewed:** 2026-10-13 · **Last updated:** 2026-10-13 · **Change:** minor (night build of 2026-10-13/14 in progress: Create Secret and the node shell built, 39 of 39 actions, breadth ~81%; the CI rework of 2026-10-13 in the progress log: parallel jobs, sharded race tests, semantic-release live; the day run of 2026-10-13 in the progress log: D-29, the lists' copy icons, the decoded certificates (D-30) and local times (D-31) built, Upcoming 2 done, breadth unchanged at ~79%, beyond Lens 20 of 40 done; earlier the same day, major: the night run in the progress log; breadth counted by `make parity`, ~56% → ~79%; the hours re-estimated roughly by the lead from the gap estimates the run lowered, not re-assessed one by one; the upcoming work after it) · **Target:** Lens Desktop 2026.10.51501

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
| **Feature breadth** | **~81%** (was ~79%) | measured by `make parity`: 64 of 64 views (ten of them the Gateway API views, generic), 39 of 39 actions, 113 of 120 interface rows (pasting kubeconfigs is not a target, D-21); 9 rows done, the rest partial |
| **Ready for real work** | **~30%** (was ~26%) | estimated (lead, 2026-10-13: breadth jumped, nothing walked by a person yet) |
| Mainstream practitioner | ~29% (was ~25%) | estimated |
| Essentials user | ~36% (was ~34%) | estimated |
| Remaining to **alpha** (macOS) | **~110–320 h** | estimated (was ~135–370) |
| Remaining to **beta** (macOS, with the Flux CD and Argo CD views) | **~380–900 h** | estimated (was ~425–995) |
| Remaining to **full parity** (free features, macOS) | **~360–920 h** | estimated (was ~420–1,040) |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~560–1,330 h** | estimated (was ~620–1,450) |
| **Beyond Lens** (MCP control of UI and cluster, built-in agent) | **~50–110 h** | estimated, unchanged; not part of the parity numbers; 20 of 40 rows done, 6 partial (the CLI and its stdio bridge, view_rows, related objects, the agents allowance, the audit log and diagnostics in the window, editor drafts, names copyable in the lists, certificates in Secrets decoded, 2026-10-13) |

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
| Foundation: workspace, cache, engine, CI | built in part | 29–74 | [architecture](docs/developer/architecture.md#current-state-2026-10-13) |

## Upcoming

| Rank | Milestone | Estimate |
|---|---|---:|
| 1 | The owner walks the night run's and the day run's features in a real window (none was seen outside offscreen renders): hotbar, navigator sections, tab bar, palette, Preferences and Cluster settings, welcome page, create dialogs, Helm install and upgrade from the repositories of Helm's own `repositories.yaml` (D-29, an authenticated one too), the copy icons in the lists, a TLS Secret's decoded certificates, the cluster Overview, the metrics status line, kubectl download, selection and Delete; then the owner's open questions (devlog and the docs' "Cogpit's default" notes) | human |
| 2 | Walk the alpha-gate workflows in the GUI on kind and a real cluster: change a workload, maintain a node, reach a service, Helm releases, adding and syncing kubeconfigs | human |
| 3 | Credential plugins verified on EKS, GKE and AKS (M1); the node shell (H-7) once the owner decides | 15–40 h + human |
| 4 | Finish M0 and M2: the generated pages and scorecard, captures of inferred rows; packaging and updates for macOS (M10); the built-in chat after the alpha (G21, D-10) | 80–200 h + human |

Detail: [docs/parity/roadmap.md](docs/parity/roadmap.md).

## Progress log

| Date | Entry |
|---|---|
| 2026-10-13 | CI rework (owner's ticket, the evening): every check a job of its own with its own make target on the organisation's self-hosted runners, as in `valkey-operator`: lint, parity, GoSec (new; 43 findings triaged, one real one fixed: `servicesOf` wrapped a Service port from the cluster), govulncheck, cyclomatic complexity (new; threshold 56, main's maximum), unit tests, the UI's race tests in eight shards by a hash of the test name (`hack/shard`), the other race tests, the kind tests and the drain cluster (in CI for the first time; `make kind-up` adds the Metrics Server), the macOS-only tests and app, the Windows compile, and a combined coverage of 83.6 % (`hack/coverage`). The old macOS job took 40 min; no job of the new set takes more than 14 min on the shared self-hosted node (one 16-CPU machine for all Linux jobs, owner's call to keep it). Found on the way: tests read the runner pod's in-cluster variables and missing shell, two tests waited for the wrong thing (fixed in the tests, 10 runs with `-race`), and a commit body line starting with `BREAKING CHANGE:` that would have made the first release 1.0.0 (`releaseRules` keep breaking changes minor under v0). Semantic-release is live (D-16 revised): each release carries `cogpit-cli` for six platforms, the unsigned app for Apple silicon and `SHA256SUMS` ([releasing](docs/developer/releasing.md)) |
| 2026-10-13 | Day run (the lead with three agents in worktrees, merged by cherry-pick). Built the owner's requirements of the morning: the chart repositories are Helm's own `repositories.yaml` (D-29), resolved as the helm CLI does, looked at every 2 s and taken over within about 2 s (measured in the tests), credentials included (user and password, CA file, client certificate, `insecure_skip_tls_verify`, `pass_credentials_all`) and kept in memory only, sent per request to the repository's own scheme, host and port; Cogpit writes nothing into Helm's files, `helm.repo.add` and `helm.repo.remove` are gone, the Repositories dialog is read-only; checked on kind with an install from a password-protected repository behind its own CA. Names, IPs, messages and the other text worth copying in the lists copyable with the drawer's copy icon, right after the text (D-25; after the owner looked at it: a blacklist of columns where copying makes no sense, by column type and id, custom resource columns kept). Certificates in Secrets decoded in the drawer, one field each, and for agents in `object_get` (D-30, owner): any key holding PEM certificates, chains in order, validity coloured; never the PEM, a value or a private key; after the owner's answers also DER certificates, `tls.key` checked against `tls.crt` in memory and the chain's order, and the decoded certificates readable by the built-in agent (H-45 closed); checked on kind. After the owner looked at the running app: every timestamp in the local time zone and the region's format, `10.09.2026 17:41:01` on the owner's Mac, read from macOS's region settings (D-31); the drawer's copy icon right after pills and chips; the macOS window buttons centred in Cogpit's 40-point bar (measured with a native harness). Later: no scaled text while the window is resized; every byte amount in binary units with one decimal (D-32: "15.6 GiB", not "16360780Ki"); a node drawer's Pods table with CPU and Memory from the metrics hub, sortable by every column as engine state (`ui.sortDrawerTable`, 153 commands), checked on kind. Fixed: Disconnect greyed during a reconnect, removed clusters' tabs left in the tab bar, `cogpit-cli contexts` without the insecure mark, nine kinds without reference links (Pod Security Policies among them), the Overviews, workload logs and port forwards reading with handles taken before the access reviews answered, a CI flake of `view_rows`. Measured: breadth ~79% (unchanged), beyond Lens 20 of 40 done; 152 commands and MCP tools (two removed). Nothing tried in a real window |
| 2026-10-13 | Night run (2026-10-12 22:12 to 2026-10-13 ~04:30; the lead with up to four agents at a time in worktrees, merged by cherry-pick, 21 scopes). Built: the hotbar (G13); the metrics providers together (Prometheus history, Metrics Server live values, a status line per provider, the fallback both ways, the cluster's Prometheus setting, kube-prometheus-stack checked on kind; P13 16 → 0 Prometheus requests per 90 s with CPU or Memory shown); the cluster Overview with Issues; seven create dialogs; the navigator's header, sections, keyboard and tooltips; a tab bar with preview and fixed tabs and history; the command palette; Preferences and Cluster settings with cluster identity, proxies, hidden metrics, accessible namespaces (namespaced watches when cluster-wide listing is denied), Helm's storage driver and kubectl for terminals (D-3, a verified download); a welcome page, Help and Window menus, Disconnect and Remove; title-bar history and notifications; Helm install from repositories, upgrade to repository versions, the ConfigMaps driver; the Applications, Device Resources, Vertical Pod Autoscalers and Pod Security Policies views; log selectors, jump to a time, older lines; row checkboxes, Delete of a selection, column reorder, search toggles; editor drafts and Revert; `cogpit-cli` (contexts, get, the MCP stdio bridge); `view_rows`, related objects, the agents allowance, the audit log and diagnostics in the window. Security fixes found on the way: Helm previews and errors that could quote Secrets read by `lookup` (H-31), a `proxy-url` password in connect errors. Measured: breadth ~79% (64 of 64 views, 37 of 39 actions, 112 of 120 interface rows); 154 commands and MCP tools (was 79). Nothing was tried in a real window |
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
| 2026-10-13 | minor | The day run in the progress log (D-29, copy icons in the lists, decoded certificates D-30 with key match and DER, local times D-31, binary byte units D-32, the node drawer's Pods table, the window buttons and resizing, leftovers); Upcoming 2 done; breadth unchanged, beyond Lens 20 of 40, hours not re-estimated |
| 2026-10-13 | minor | Upcoming: the owner's requirements of 2026-10-13 (D-29, D-25) |
| 2026-10-13 | major | The night run in the progress log; breadth ~79%; hours re-estimated roughly; upcoming after it |
| 2026-10-12 | minor | The fifth run in the progress log; upcoming after it; Helm releases and metrics re-estimated; breadth unchanged |
| 2026-10-12 | major | Fourth measurement after the fourth implementation run, counted by `make parity` |
| 2026-10-12 | minor | Upcoming: the owner's decisions D-13 (revised), D-14, D-15, D-16 |
| 2026-10-11 | major | Third measurement after the third implementation run |
| 2026-10-11 | major | Second measurement after the second implementation run |
| 2026-10-11 | major | First measurement after the local MVP |
| 2026-10-11 | major | Go plan, macOS first, beyond-Lens targets separated |
| 2026-10-11 | major | First version |
