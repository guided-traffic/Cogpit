# Roadmap

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (the local MVP: alpha gate and milestones re-assessed, new current focus) · **Target:** Lens Desktop 2026.10.51501

Forward-looking plan: the alpha gate, the milestones, the **current focus** and what's next with
estimates. Where Cogpit stands is in [`target-app-parity.md`](target-app-parity.md); the ranked
work list is [`gaps.md`](gaps.md); the one-page summary is [`ROADMAP.md`](../../ROADMAP.md).

Status legend: ✅ done · 🟡 in progress · ⬜ not started. Hours are Opus 5.5 agent-hours for one
agent working sequentially ([calibration](target-app-parity.md#remaining-effort-and-calibration)).

**Stage: pre-alpha.** Alpha needs the gate below on macOS: ~250–700 h (was ~400–950 before the
local MVP of 2026-10-11). Cogpit is a desktop client
for macOS first; Windows and Linux follow after the macOS beta
([D-7](../developer/decisions.md#d-7-language-and-platforms)).

## Alpha gate

The core workflows a daily Lens user runs, checked end to end on macOS against a real cluster. Any
**no**, or a **partial** that blocks the workflow, keeps Cogpit pre-alpha. Every workflow needs the
foundation (G1–G4, 125–250 h) and the Fyne spike (16–24 h); the hours below come on top.

| Core workflow | Works end to end? | Needs | Hours to pass |
|---|---|---|---:|
| **Connect:** add a kubeconfig (file, synced folder or pasted text); connect with a client certificate, a token, or an exec plugin (EKS, GKE, AKS); switch between clusters | partial: contexts from `$KUBECONFIG` or `~/.kube/config` in a picker; certificates (kind) and tokens (fake server) work; no adding, syncing or pasting; plugins unverified | M1, M2 | 22–45 |
| **Triage a failing pod:** Pods list with status, the drawer with containers, conditions and events, logs (follow, previous container), a shell in the container | partial: all of it except the previous container's logs, on kind (`TestKindEngineWorkflows`) | M2, M3 | 70–180 |
| **Change a workload:** edit YAML and apply it after a diff and a dry-run; scale and restart a Deployment; delete a pod after a confirmation | no: restart and delete with confirmation exist; editing and scaling do not | M4 | 50–130 |
| **Reach a service:** forward a pod or service port and open it in the browser | no | M5 | 15–35 |
| **Manage Helm releases in the cluster** (Lens's Helm view): list releases, read history and values, upgrade, roll back | no | M6 | 40–85 |
| **Maintain a node:** cordon, drain and uncordon with confirmations; node details | no | M4 | 10–35 |
| **Use kubectl:** a terminal bound to the cluster's context and namespace | partial: works on kind (`TestKindEngineWorkflows`); a shell profile can override it ([H-20](../security/credentials.md#h-20-shell-profiles-can-override-the-terminals-kubeconfig)) | M3 | 5–20 |
| A downloadable macOS build (unsigned is enough) | partial: `make app` builds an unsigned app bundle locally; nothing to download | G14 | 3–8 |

**Result: fails.** Editing, port forwarding, Helm and node maintenance do not exist; connecting and
triaging work only in part.

## Milestones

| M | Status | Scope | Definition of done | Estimate |
|---|---|---|---|---:|
| **M0** Skeleton | 🟡 (go.mod, Makefile, lint with layering done; CI, generators, semantic-release open) | `go.mod`, `Makefile`, golangci-lint with `depguard` layering, CI on macOS with compile checks for Windows and Linux, envtest and `kind` jobs, semantic-release, the parity and scorecard generators (G1) | `make lint test` green in CI; a forbidden import fails the lint; the generated parity pages match the hand-written checklists | 20–40 |
| **Spike** Fyne gate | 🟡 (ran as the local MVP; [results](../developer/spike-fyne.md#results) recorded, decision pending) | Table, YAML editor, terminal and MCP control on Fyne ([spike plan](../developer/spike-fyne.md)) | Gate criteria S1–S7 recorded; [D-1](../developer/decisions.md#d-1-ui-toolkit) confirmed or the fallback chosen | 16–24 |
| **M1** Connect | 🟡 (contexts, connections, discovery, credential-safe errors; no CLI, no pasting) | `kubeconfig` and `cluster`: discovery, merge, pasting; every authentication form; cached discovery; access reviews; `cogpit-cli contexts` and `cogpit-cli get <kind>` (G2) | Connects to `kind` and envtest in CI with certificate and token auth; exec plugins verified on EKS, GKE and AKS (human); fuzzed kubeconfig parsing never panics | 25–50 |
| **M2** Browse | 🟡 (cache, engine, shell, six views, MCP navigation; budgets P2 and P9–P11 met in tests) | The cache (G3), the engine and registry with UI state (G4), the Fyne shell: layout, sidebar from discovery, tables, drawer, namespace filter (part of G5); about 20 core views (part of G6); MCP `ui_state`, `resources_find`, `ui_navigate` (part of G17) | A 10,000-pod list meets budgets P2–P4; an idle view costs no list requests (P9); every built-in kind the cluster serves appears; an MCP client navigates the GUI to a Service by its cluster IP | 180–420 |
| **M3** Inspect | 🟡 (logs with follow, cluster-bound terminal, pod shell; no attach or node shell) | Logs, terminal sessions, pod shell, attach, node shell (G8) | Budget P5 met; a terminal's `kubectl` talks to its own cluster; the node shell meets the rules in [privilege footprint](../security/privilege-footprint.md#node-shell) | 45–90 |
| **M4** Change | 🟡 (Restart and Delete with confirmation only) | Editor with diff and server-side dry-run, create from templates, delete with confirmation, workload and node actions with the kubectl libraries (G9, part of G7) | Every Lens Desktop action on workloads and nodes; the command fuzz target green; confirmation enforced for the CLI and agents too; an interrupted drain is detected on the next start | 65–155 |
| **M5** Reach | ⬜ | Port forwarding: manager, dialog, view (G10) | Forward, open in browser, stop; listeners bound to `127.0.0.1` only | 20–55 |
| **M6** Helm | ⬜ | Helm SDK in-process: releases from the cache, install, upgrade, rollback, uninstall as crash-safe jobs; repositories and charts (G11) | The Helm workflow of the alpha gate passes on `kind`; a release left pending by a crash is offered for completion or rollback | 40–95 |
| **Alpha (macOS)** | ⬜ | M0–M6 and an unsigned macOS build | The alpha gate passes | ~400–950 total |
| **M7** Metrics | ⬜ | Provider detection, cluster overview, charts (G12) | Metrics shown on `kind` with metrics-server and with Prometheus; no provider is a state, not an error; polling stops when nothing visible needs it | 40–80 |
| **M8** Interface parity | ⬜ | Clusters, hotbar, palette, menus, shortcuts, preferences, cluster settings, themes, deep links (rest of G5 and G13) | Every free Lens Desktop row in [ui-parity](ui-parity.md) done | 85–210 |
| **M9** MCP control | 🟡 (every command a tool, confirmation in the GUI, audit log file; no stdio bridge) | The rest of [beyond-lens › MCP](beyond-lens.md#mcp-control-of-ui-and-cluster): every command as a tool, dock tools, the GUI confirmation flow, the audit log, the stdio bridge (rest of G17) | An external agent completes the triage and change workflows through MCP, with every mutating step confirmed in the GUI | 20–40 |
| **M10** macOS distribution | ⬜ | Signed universal app in a notarized disk image, Homebrew cask, Sparkle 2 with an EdDSA-signed appcast, update notice and one-click update (rest of G14; [D-4](../developer/decisions.md#d-4-packaging-and-updates)) | Installs from the disk image and updates itself on a clean Mac without warnings | 20–40 |
| **GitOps** Flux CD and Argo CD | ⬜ | Kind profiles, dashboards, Argo CD Sync and Refresh (G23, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | A Flux and an Argo CD installation on `kind` show their objects with status; Sync and Refresh work with confirmation | 55–115 |
| **Beta (macOS)** | ⬜ | M7–M10, the GitOps views, ~75% of views and actions, budgets P1–P11 met (G15), the first user bug backlog (G16) | Ready for real work ≥ ~75% on macOS; no gap marked *blocks beta* | ~840–1,870 total |
| **M11** Windows and Linux | ⬜ | ConPTY, keychain backends, X11 and Wayland, packages and updates per format, Ubuntu first (G22) | The alpha gate passes on Windows and on Ubuntu | 40–80 |
| **M12** Built-in agent | ⬜ | Chat tab, OpenAI-compatible endpoint (LiteLLM, Open WebUI), tool loop with D-2's confirmation rules (G21) | The agent answers questions about the selected object, navigates the UI on request, and runs a confirmed action, against a LiteLLM and an Open WebUI endpoint | 40–80 |
| **M13** Paid-plan features | ⬜ | The remaining paid Lens features, built into the code: cloud account integrations, Security Center, cost monitoring (G18, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features), [D-5](../developer/decisions.md#d-5-extensions)) | Each matches its Lens Desktop rows | 110–220 |

M11 and M12 do not depend on each other; their order is the owner's call.

## Current focus

Ranked; each item links to its gap. Pick from the top unless an issue is assigned to you.

1. **The owner's gate decision on the Fyne spike** ([results](../developer/spike-fyne.md#results),
   [D-1](../developer/decisions.md#d-1-ui-toolkit)); S1 passed with a reservation (owner,
   2026-10-11); open: D-1 itself and keeping the patched terminal widget ([H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)).
2. **Finish M0** ([G1](gaps.md#g1-workspace-ci-and-progress-generators)): CI with the race tests,
   lint and the kind tests, so that the MVP's tests guard `main`. 12–28 h.
3. **Editing** ([G9](gaps.md#g9-yaml-editor-and-create-resource)): wire the YAML editor to an Edit
   tab with diff, server-side dry-run and apply; the largest hole in the alpha gate's "change a
   workload". 27–60 h.
4. **Breadth of M2** ([G6](gaps.md#g6-resource-views), [G7](gaps.md#g7-resource-actions)): the
   remaining core views (Daemon Sets, Stateful Sets, Replica Sets, Jobs, Cron Jobs, Ingresses,
   Config Maps, Secrets, PVCs) and the generic custom-resource view, one kind profile per agent.

## What's next, with estimates

| Next | Scope | Estimate |
|---|---|---:|
| 0.1 "connects" | the rest of M0 and M1: CI, the CLI lists contexts and objects of any cluster; the UI toolkit is settled | 30–70 h |
| 0.2 "browses" | the rest of M2: about 20 core views, the generic custom-resource view, access reviews | 130–330 h |
| Alpha (macOS) | M3–M6 and an unsigned build: the seven core workflows | ~250–700 h total |
| Beta (macOS) | M7–M10, the GitOps views, performance, stability | ~680–1,610 h total |
| After the macOS beta | M11 Windows and Linux, M12 built-in agent, M13 paid-plan features; Teamwork only if a relay service is wanted (G20) | 190–380 h, Teamwork 150–300 h |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | The local MVP: alpha gate rows partial where they work; M0, the spike, M1–M4 and M9 in progress; current focus: the gate decision, CI, editing, breadth |
| 2026-10-11 | major | Go plan: spike gate, macOS-first alpha and beta, MCP basics in M2 and full control in M9, one-click updates in M10, Windows and Linux in M11, built-in agent in M12 |
| 2026-10-11 | major | First version: alpha gate, M0–M10, current focus |
