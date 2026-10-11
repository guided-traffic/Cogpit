# Roadmap

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (Go plan: Fyne spike gate before M2; alpha and beta on macOS; MCP basics in M2; Windows and Linux, then the built-in agent after the macOS beta) · **Target:** Lens Desktop 2026.10.51501

Forward-looking plan: the alpha gate, the milestones, the **current focus** and what's next with
estimates. Where Cogpit stands is in [`target-app-parity.md`](target-app-parity.md); the ranked
work list is [`gaps.md`](gaps.md); the one-page summary is [`ROADMAP.md`](../../ROADMAP.md).

Status legend: ✅ done · 🟡 in progress · ⬜ not started. Hours are Opus 5.5 agent-hours for one
agent working sequentially ([calibration](target-app-parity.md#remaining-effort-and-calibration)).

**Stage: pre-alpha.** Alpha needs the gate below on macOS: ~400–950 h. Cogpit is a desktop client
for macOS first; Windows and Linux follow after the macOS beta
([D-7](../developer/decisions.md#d-7-language-and-platforms)).

## Alpha gate

The core workflows a daily Lens user runs, checked end to end on macOS against a real cluster. Any
**no**, or a **partial** that blocks the workflow, keeps Cogpit pre-alpha. Every workflow needs the
foundation (G1–G4, 125–250 h) and the Fyne spike (16–24 h); the hours below come on top.

| Core workflow | Works end to end? | Needs | Hours to pass |
|---|---|---|---:|
| **Connect:** add a kubeconfig (file, synced folder or pasted text); connect with a client certificate, a token, or an exec plugin (EKS, GKE, AKS); switch between clusters | no | M1, M2 | 30–60 |
| **Triage a failing pod:** Pods list with status, the drawer with containers, conditions and events, logs (follow, previous container), a shell in the container | no | M2, M3 | 110–250 |
| **Change a workload:** edit YAML and apply it after a diff and a dry-run; scale and restart a Deployment; delete a pod after a confirmation | no | M4 | 60–145 |
| **Reach a service:** forward a pod or service port and open it in the browser | no | M5 | 15–35 |
| **Manage Helm releases in the cluster** (Lens's Helm view): list releases, read history and values, upgrade, roll back | no | M6 | 40–85 |
| **Maintain a node:** cordon, drain and uncordon with confirmations; node details | no | M4 | 10–35 |
| **Use kubectl:** a terminal bound to the cluster's context and namespace | no | M3 | 15–40 |
| A downloadable macOS build (unsigned is enough) | no | G14 | 5–10 |

**Result: fails.** Nothing is built.

## Milestones

| M | Status | Scope | Definition of done | Estimate |
|---|---|---|---|---:|
| **M0** Skeleton | ⬜ | `go.mod`, `Makefile`, golangci-lint with `depguard` layering, CI on macOS with compile checks for Windows and Linux, envtest and `kind` jobs, semantic-release, the parity and scorecard generators (G1) | `make lint test` green in CI; a forbidden import fails the lint; the generated parity pages match the hand-written checklists | 20–40 |
| **Spike** Fyne gate | ⬜ | Table, YAML editor, terminal and MCP control on Fyne ([spike plan](../developer/spike-fyne.md)) | Gate criteria S1–S7 recorded; [D-1](../developer/decisions.md#d-1-ui-toolkit) confirmed or the fallback chosen | 16–24 |
| **M1** Connect | ⬜ | `kubeconfig` and `cluster`: discovery, merge, pasting; every authentication form; cached discovery; access reviews; `cogpit-cli contexts` and `cogpit-cli get <kind>` (G2) | Connects to `kind` and envtest in CI with certificate and token auth; exec plugins verified on EKS, GKE and AKS (human); fuzzed kubeconfig parsing never panics | 25–50 |
| **M2** Browse | ⬜ | The cache (G3), the engine and registry with UI state (G4), the Fyne shell: layout, sidebar from discovery, tables, drawer, namespace filter (part of G5); about 20 core views (part of G6); MCP `ui_state`, `resources_find`, `ui_navigate` (part of G17) | A 10,000-pod list meets budgets P2–P4; an idle view costs no list requests (P9); every built-in kind the cluster serves appears; an MCP client navigates the GUI to a Service by its cluster IP | 180–420 |
| **M3** Inspect | ⬜ | Logs, terminal sessions, pod shell, attach, node shell (G8) | Budget P5 met; a terminal's `kubectl` talks to its own cluster; the node shell meets the rules in [privilege footprint](../security/privilege-footprint.md#node-shell) | 45–90 |
| **M4** Change | ⬜ | Editor with diff and server-side dry-run, create from templates, delete with confirmation, workload and node actions with the kubectl libraries (G9, part of G7) | Every Lens Desktop action on workloads and nodes; the command fuzz target green; confirmation enforced for the CLI and agents too; an interrupted drain is detected on the next start | 65–155 |
| **M5** Reach | ⬜ | Port forwarding: manager, dialog, view (G10) | Forward, open in browser, stop; listeners bound to `127.0.0.1` only | 20–55 |
| **M6** Helm | ⬜ | Helm SDK in-process: releases from the cache, install, upgrade, rollback, uninstall as crash-safe jobs; repositories and charts (G11) | The Helm workflow of the alpha gate passes on `kind`; a release left pending by a crash is offered for completion or rollback | 40–95 |
| **Alpha (macOS)** | ⬜ | M0–M6 and an unsigned macOS build | The alpha gate passes | ~400–950 total |
| **M7** Metrics | ⬜ | Provider detection, cluster overview, charts (G12) | Metrics shown on `kind` with metrics-server and with Prometheus; no provider is a state, not an error; polling stops when nothing visible needs it | 40–80 |
| **M8** Interface parity | ⬜ | Clusters, hotbar, palette, menus, shortcuts, preferences, cluster settings, themes, deep links (rest of G5 and G13) | Every free Lens Desktop row in [ui-parity](ui-parity.md) done | 85–210 |
| **M9** MCP control | ⬜ | The rest of [beyond-lens › MCP](beyond-lens.md#mcp-control-of-ui-and-cluster): every command as a tool, dock tools, the GUI confirmation flow, the audit log, the stdio bridge (rest of G17) | An external agent completes the triage and change workflows through MCP, with every mutating step confirmed in the GUI | 20–40 |
| **M10** macOS distribution | ⬜ | Signed universal app in a notarized disk image, Homebrew cask, Sparkle 2 with an EdDSA-signed appcast, update notice and one-click update (rest of G14; [D-4](../developer/decisions.md#d-4-packaging-and-updates)) | Installs from the disk image and updates itself on a clean Mac without warnings | 20–40 |
| **GitOps** Flux CD and Argo CD | ⬜ | Kind profiles, dashboards, Argo CD Sync and Refresh (G23, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | A Flux and an Argo CD installation on `kind` show their objects with status; Sync and Refresh work with confirmation | 55–115 |
| **Beta (macOS)** | ⬜ | M7–M10, the GitOps views, ~75% of views and actions, budgets P1–P11 met (G15), the first user bug backlog (G16) | Ready for real work ≥ ~75% on macOS; no gap marked *blocks beta* | ~840–1,870 total |
| **M11** Windows and Linux | ⬜ | ConPTY, keychain backends, X11 and Wayland, packages and updates per format, Ubuntu first (G22) | The alpha gate passes on Windows and on Ubuntu | 40–80 |
| **M12** Built-in agent | ⬜ | Chat tab, OpenAI-compatible endpoint (LiteLLM, Open WebUI), tool loop with D-2's confirmation rules (G21) | The agent answers questions about the selected object, navigates the UI on request, and runs a confirmed action, against a LiteLLM and an Open WebUI endpoint | 40–80 |
| **M13** Paid-plan features | ⬜ | The remaining paid Lens features, built into the code: cloud account integrations, Security Center, cost monitoring (G18, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features), [D-5](../developer/decisions.md#d-5-extensions)) | Each matches its Lens Desktop rows | 110–220 |

M11 and M12 do not depend on each other; their order is the owner's call.

## Current focus

Ranked; each item links to its gap. Pick from the top unless an issue is assigned to you.

1. **The owner's go-ahead to start implementation.** Until then, sessions change documentation
   only ([AGENTS.md](../../AGENTS.md)).
2. **M0 Skeleton** ([G1](gaps.md#g1-workspace-ci-and-progress-generators)). 20–40 h.
3. **The Fyne spike** ([plan](../developer/spike-fyne.md)), which decides
   [D-1](../developer/decisions.md#d-1-ui-toolkit) before M2. 16–24 h.
4. **M1 Connect** ([G2](gaps.md#g2-cluster-connection-and-authentication)). 25–50 h plus cloud
   accounts.

## What's next, with estimates

| Next | Scope | Estimate |
|---|---|---:|
| 0.1 "connects" | M0, the spike and M1: the CLI lists contexts and objects of any cluster; the UI toolkit is settled | 60–115 h |
| 0.2 "browses" | M2: the macOS app shows any cluster's resources from the cache, read-only; agents can navigate it | 180–420 h |
| Alpha (macOS) | M3–M6 and an unsigned build: the seven core workflows | ~400–950 h total |
| Beta (macOS) | M7–M10, the GitOps views, performance, stability | ~840–1,870 h total |
| After the macOS beta | M11 Windows and Linux, M12 built-in agent, M13 paid-plan features; Teamwork only if a relay service is wanted (G20) | 190–380 h, Teamwork 150–300 h |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | Go plan: spike gate, macOS-first alpha and beta, MCP basics in M2 and full control in M9, one-click updates in M10, Windows and Linux in M11, built-in agent in M12 |
| 2026-10-11 | major | First version: alpha gate, M0–M10, current focus |
