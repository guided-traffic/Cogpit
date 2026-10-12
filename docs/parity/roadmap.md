# Roadmap

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-12 · **Change:** major (fourth run: Connect and Helm workflows exist in full, not walked through; alpha ~135–370 h; new current focus) · **Target:** Lens Desktop 2026.10.51501

Forward-looking plan: the alpha gate, the milestones, the **current focus** and what's next with
estimates. Where Cogpit stands is in [`target-app-parity.md`](target-app-parity.md); the ranked
work list is [`gaps.md`](gaps.md); the one-page summary is [`ROADMAP.md`](../../ROADMAP.md).

Status legend: ✅ done · 🟡 in progress · ⬜ not started. Hours are Opus 5.5 agent-hours for one
agent working sequentially ([calibration](target-app-parity.md#remaining-effort-and-calibration)).

**Stage: pre-alpha.** Alpha needs the gate below on macOS: ~135–370 h (2026-10-12; ~150–400 after
the third run, ~210–520 after the second run, ~250–700 after the local MVP and ~400–950 before it, 2026-10-11). Cogpit is a desktop client
for macOS first; Windows and Linux follow after the macOS beta
([D-7](../developer/decisions.md#d-7-language-and-platforms)).

## Alpha gate

The core workflows a daily Lens user runs, checked end to end on macOS against a real cluster. Any
**no**, or a **partial** that blocks the workflow, keeps Cogpit pre-alpha. Every workflow needs the
foundation (G1–G4: 39–102 h left after the fourth implementation run; the Fyne spike has run);
the hours below come on top.

| Core workflow | Works end to end? | Needs | Hours to pass |
|---|---|---|---:|
| **Connect:** add a kubeconfig (file or synced folder; pasted text is not a target, D-21); connect with a client certificate, a token, or an exec plugin (EKS, GKE, AKS); switch between clusters | partial (2026-10-12): contexts from `$KUBECONFIG` or `~/.kube/config`, and from kubeconfig files, and sync folders the user adds, each read on its own; a credential plugin of an added file runs only after the user allowed it in the GUI (`TestKindAddedKubeconfig`, `TestKindExecPluginAfterApproval`); certificates (kind) and tokens (fake server) work; access reviews grey what the user may not do; no cloud plugin verified; Lens's dialogs not captured | M1, M2 | 12–30 |
| **Triage a failing pod:** Pods list with status, the drawer with containers, conditions and events, logs (follow, previous container), a shell in the container | partial: all of it on kind, the previous container's logs, timestamps and the logs of a workload's pods included (`TestKindEngineWorkflows`, `TestKindWorkloadLogsAttachRestartScale`); not walked through by a person on a real cluster | M2, M3 | 55–150 |
| **Change a workload:** edit YAML and apply it after a diff and a dry-run; scale and restart a Deployment; delete a pod after a confirmation | partial: every step exists and runs on kind through the engine (edit, diff, dry run, apply and a conflict: `TestKindEditCreateConflictDelete`; scale: `TestKindNodeAndWorkloadActions`; restart and delete confirmed); the GUI path (row menu, Edit tab, scale dialog) is tested offscreen only, not yet walked through by a person on a real cluster | M4 | 10–30 |
| **Reach a service:** forward a pod or service port and open it in the browser | partial (2026-10-11): pod and service forwards on 127.0.0.1, the captured dialog, the Port Forwarding view, reconnects after a pod restart, on kind (`TestKindPortForward`: an HTTP GET through the forward reaches the service, also after its pod was replaced); the browser opening is tested up to Fyne's `OpenURL` call, not yet seen in a real window | M5 | 2–6 |
| **Manage Helm releases in the cluster** (Lens's Helm view): list releases, read history and values, upgrade, roll back | partial (2026-10-12): releases from the cache with Lens's drawer (values, notes, resources), rollback with Lens's dialog, uninstall, and upgrade from the Helm Upgrade tab with the release's own chart, crash-safe through a journal, on kind (`TestKindHelmReleases`, `TestKindHelmUpgrade`); not walked through by a person on a real cluster; some of Lens's behaviour still to capture | M6 | 5–15 |
| **Maintain a node:** cordon, drain and uncordon with confirmations; node details | partial: cordon, a crash-safe drain and uncordon, confirmed, on a three-node kind cluster through the engine (`TestKindDrainCrashFinishUncordon`); the node drawer as captured with its Pods table, without metrics; the GUI path tested offscreen only | M4 | 2–8 |
| **Use kubectl:** a terminal bound to the cluster's context and namespace | partial: works on kind (`TestKindEngineWorkflows`); a shell profile can override it ([H-20](../security/credentials.md#h-20-shell-profiles-can-override-the-terminals-kubeconfig)) | M3 | 5–20 |
| A downloadable macOS build (unsigned is enough) | partial: `make app` builds an unsigned app bundle locally; nothing to download | G14 | 3–8 |

**Result: fails.** Every workflow exists since the fourth run (2026-10-12): Helm upgrade and adding
and syncing kubeconfigs were built. None has been walked through end to end in the GUI by
a person on a real cluster, and no cloud credential plugin is verified (human).

## Milestones

| M | Status | Scope | Definition of done | Estimate |
|---|---|---|---|---:|
| **M0** Skeleton | 🟡 (go.mod, Makefile, lint with layering done; CI green on GitHub since 2026-10-11 ([releasing › CI](../developer/releasing.md#ci)): lint, race tests and build on macOS, compile checks for Windows and Linux, kind on 1.37, govulncheck; semantic-release runs as a dry run; `make parity` checks the checklists in CI since 2026-10-12; the generated pages, the scorecard and the envtest job open) | `go.mod`, `Makefile`, golangci-lint with `depguard` layering, CI on macOS with compile checks for Windows and Linux, envtest and `kind` jobs, semantic-release, the parity and scorecard generators (G1) | `make lint test` green in CI; a forbidden import fails the lint; the generated parity pages match the hand-written checklists | 20–40 |
| **Spike** Fyne gate | ✅ (ran as the local MVP; [results](../developer/spike-fyne.md#results); D-1 decided for Fyne, with re-checks of S7 and S5) | Table, YAML editor, terminal and MCP control on Fyne ([spike plan](../developer/spike-fyne.md)) | Gate criteria S1–S7 recorded; [D-1](../developer/decisions.md#d-1-ui-toolkit) confirmed or the fallback chosen | 16–24 |
| **M1** Connect | 🟡 (contexts, connections, discovery, credential-safe errors; kubeconfig files and sync folders with credential plugins run only once allowed (2026-10-12); no CLI, no cloud plugin verified) | `kubeconfig` and `cluster`: discovery, merge; every authentication form; cached discovery; access reviews; `cogpit-cli contexts` and `cogpit-cli get <kind>` (G2) | Connects to `kind` and envtest in CI with certificate and token auth; exec plugins verified on EKS, GKE and AKS (human); fuzzed kubeconfig parsing never panics | 25–50 |
| **M2** Browse | 🟡 (cache with scoped, metadata-only, local and virtual kinds, engine, shell, 41 built-in profiles, the Port Forwarding (local) and Helm Releases (virtual) views, the Workloads Overview page and the custom resource views, row menus from the registry, MCP navigation, access reviews; budgets P2 and P9–P11 met in tests; the Secrets view from metadata only (H-8); 9 views and the cluster overview missing) | The cache (G3), the engine and registry with UI state (G4), the Fyne shell: layout, sidebar from discovery, tables, drawer, namespace filter (part of G5); about 20 core views (part of G6); MCP `ui_state`, `resources_find`, `ui_navigate` (part of G17) | A 10,000-pod list meets budgets P2–P4; an idle view costs no list requests (P9); every built-in kind the cluster serves appears; an MCP client navigates the GUI to a Service by its cluster IP | 180–420 |
| **M3** Inspect | 🟡 (logs with follow, previous container, timestamps, wrap, filter, download and the logs of a workload's pods; cluster-bound terminal, pod shell, attach; no node shell) | Logs, terminal sessions, pod shell, attach, node shell (G8) | Budget P5 met; a terminal's `kubectl` talks to its own cluster; the node shell meets the rules in [privilege footprint](../security/privilege-footprint.md#node-shell) | 45–90 |
| **M4** Change | 🟡 (Edit with diff, dry run and a resourceVersion precondition; create from Cogpit's templates; Delete for every kind; Restart (Deployments, Daemon Sets, Stateful Sets), Scale (Deployments, Stateful Sets, Replica Sets), Cordon, Uncordon, Drain, Trigger, Suspend, Resume, Evict, Attach with confirmations; an interrupted drain is offered on the next connect; Edit and Delete follow the access reviews; missing: the create dialogs, drafts, node shell) | Editor with diff and server-side dry-run, create from templates, delete with confirmation, workload and node actions with the kubectl libraries (G9, part of G7) | Every Lens Desktop action on workloads and nodes; the command fuzz target green; confirmation enforced for the CLI and agents too; an interrupted drain is detected on the next start | 65–155 |
| **M5** Reach | 🟡 (built 2026-10-11: pod and service forwards on 127.0.0.1, the captured dialog, the Port Forwarding view, reconnects after a pod restart, on kind; the browser not yet seen opening in a real window) | Port forwarding: manager, dialog, view (G10) | Forward, open in browser, stop; listeners bound to `127.0.0.1` only | 2–6 |
| **M6** Helm | 🟡 (read side, rollback and uninstall 2026-10-11: the Releases view from the `owner=helm` Secrets, history, values and manifest, rollback and uninstall through the Helm SDK, a release left pending by Cogpit offered on the next connect (journal), on kind; upgrade from the Helm Upgrade tab with Lens's drawer and Rollback dialog (2026-10-12); missing: install, repositories and charts) | Helm SDK in-process: releases from the cache, install, upgrade, rollback, uninstall as crash-safe jobs; repositories and charts (G11) | The Helm workflow of the alpha gate passes on `kind`; a release left pending by a crash is offered for completion or rollback | 25–60 |
| **Alpha (macOS)** | ⬜ | M0–M6 and an unsigned macOS build | The alpha gate passes | ~135–370 left |
| **M7** Metrics | ⬜ | Provider detection, cluster overview, charts (G12) | Metrics shown on `kind` with metrics-server and with Prometheus; no provider is a state, not an error; polling stops when nothing visible needs it | 40–80 |
| **M8** Interface parity | ⬜ | Clusters, hotbar, palette, menus, shortcuts, preferences, cluster settings, themes, deep links (rest of G5 and G13) | Every free Lens Desktop row in [ui-parity](ui-parity.md) done | 85–210 |
| **M9** MCP control | 🟡 (every command a tool, confirmation in the GUI, audit log file; no stdio bridge) | The rest of [beyond-lens › MCP](beyond-lens.md#mcp-control-of-ui-and-cluster): every command as a tool, dock tools, the GUI confirmation flow, the audit log, the stdio bridge (rest of G17) | An external agent completes the triage and change workflows through MCP, with every mutating step confirmed in the GUI | 20–40 |
| **M10** macOS distribution | ⬜ | Signed universal app in a notarized disk image, Homebrew cask, Sparkle 2 with an EdDSA-signed appcast, update notice and one-click update (rest of G14; [D-4](../developer/decisions.md#d-4-packaging-and-updates)) | Installs from the disk image and updates itself on a clean Mac without warnings | 20–40 |
| **GitOps** Flux CD and Argo CD | ⬜ | Kind profiles, dashboards, Argo CD Sync and Refresh (G23, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | A Flux and an Argo CD installation on `kind` show their objects with status; Sync and Refresh work with confirmation | 55–115 |
| **Beta (macOS)** | ⬜ | M7–M10, the GitOps views, ~75% of views and actions, budgets P1–P11 met (G15), the first user bug backlog (G16) | Ready for real work ≥ ~75% on macOS; no gap marked *blocks beta* | ~425–995 left |
| **M11** Windows and Linux | ⬜ | ConPTY, keychain backends, X11 and Wayland, packages and updates per format, Ubuntu first (G22) | The alpha gate passes on Windows and on Ubuntu | 40–80 |
| **M12** Built-in agent | ⬜ | Chat tab, OpenAI-compatible endpoint (LiteLLM, Open WebUI), tool loop with D-2's confirmation rules (G21) | The agent answers questions about the selected object, navigates the UI on request, and runs a confirmed action, against a LiteLLM and an Open WebUI endpoint | 40–80 |
| **M13** Paid-plan features | ⬜ | The remaining paid Lens features, built into the code: cloud account integrations, Security Center, cost monitoring (G18, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features), [D-5](../developer/decisions.md#d-5-extensions)) | Each matches its Lens Desktop rows | 110–220 |

M11 and M12 do not depend on each other; their order is the owner's call.

## Current focus

Ranked; each item links to its gap. Pick from the top unless an issue is assigned to you.

1. **The owner's next requirements (2026-10-12, "do not forget"):** pod and node metrics from
   metrics-server, and the drawer's resource graphs from Prometheus (pod and per container; CPU,
   memory, network, filesystem; requests and limits as lines; events as yellow markers)
   ([G12](gaps.md#g12-metrics-and-cluster-overview)); the hotbar
   ([G13](gaps.md#g13-clusters-hotbar-palette-menus-shortcuts-and-preferences)); the drawer's width
   set by the user only, never by its content, and a resizable sidebar
   ([G5](gaps.md#g5-app-shell-navigation-tables-and-drawer)).
2. **Finish the fifth run:** the default kubeconfig as sources
   ([D-28](../developer/decisions.md#d-28-the-default-kubeconfig-as-sources-as-lens-has-it)), the
   UI PNGs regenerated, the announced smoke launch with the owner, the upstream pull requests of the
   terminal widget ([G8](gaps.md#g8-logs-terminal-and-shells)).
3. **Walk the workflows in the GUI** on kind and a real cluster (owner): change a workload,
   maintain a node, reach a service, Helm upgrade, rollback and uninstall, adding and syncing
   kubeconfigs, Cogpit's own title bar; record what differs in the parity rows.
4. **Helm install and charts** ([G11](gaps.md#g11-helm)), then the credential plugins verified on
   EKS, GKE and AKS ([G2](gaps.md#g2-cluster-connection-and-authentication), human).
5. **Breadth and depth of M2 and M4** ([G6](gaps.md#g6-resource-views),
   [G7](gaps.md#g7-resource-actions)): the cluster overview (M7), the create dialogs, the node
   shell, captures of the rows marked inferred; the generated parity pages and the scorecard
   ([G1](gaps.md#g1-workspace-ci-and-progress-generators)).

## What's next, with estimates

| Next | Scope | Estimate |
|---|---|---:|
| 0.1 "connects" | the rest of M0 and M1: the generators, the CLI lists contexts and objects of any cluster | 24–54 h |
| 0.2 "browses" | the rest of M2: the cluster overview, the remaining views, captures of the inferred rows | 40–120 h |
| Alpha (macOS) | M3–M6 and an unsigned build: the seven core workflows | ~150–400 h total |
| Beta (macOS) | M7–M10, the GitOps views, performance, stability | ~440–1,020 h total |
| After the macOS beta | M11 Windows and Linux, M12 built-in agent, M13 paid-plan features; Teamwork only if a relay service is wanted (G20) | 190–380 h, Teamwork 150–300 h |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-12 | major | Fourth implementation run: Helm upgrade and adding, pasting and syncing kubeconfigs built, every alpha-gate workflow exists; M1 and M6 re-assessed; alpha ~135–370 h; new current focus |
| 2026-10-12 | minor | M0: `make parity` checks the checklists in CI (D-15); current focus item 1 notes it built |
| 2026-10-12 | minor | The owner's decisions (D-13 revised, D-14, D-15, D-16) in the current focus |
| 2026-10-11 | major | Third implementation run: "reach a service" and "manage Helm releases" partial (no Helm upgrade), every alpha-gate workflow exists in part; M2–M6 re-assessed; alpha ~150–400 h; new current focus |
| 2026-10-11 | minor | H-8 decided (owner): Helm and the Secrets view unblocked in the current focus |
| 2026-10-11 | major | Second implementation run: CI green (M0), editing and node and workload actions (M4), 16 built-in views and custom resources (M2); "change a workload" and "maintain a node" partial; alpha ~210–520 h; new current focus |
| 2026-10-11 | minor | M0: CI in workflow files (macOS gate, Windows and Linux compile checks, kind job, govulncheck, coverage badge), semantic-release as a dry run; not yet observed green |
| 2026-10-11 | major | The local MVP: alpha gate rows partial where they work; M0, the spike, M1–M4 and M9 in progress; current focus: the gate decision, CI, editing, breadth |
| 2026-10-11 | major | Go plan: spike gate, macOS-first alpha and beta, MCP basics in M2 and full control in M9, one-click updates in M10, Windows and Linux in M11, built-in agent in M12 |
| 2026-10-11 | major | First version: alpha gate, M0–M10, current focus |
