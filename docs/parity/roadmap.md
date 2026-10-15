# Roadmap

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-15 · **Change:** minor (Current focus: the chat's API keys in an owner-only file on every platform, no keychain (the owner, 2026-10-15), the keychain on Windows and Linux no longer open; M11 without keychain backends); minor (Current focus: the chat's H-96 built, a private connection's conversations not kept); minor (Current focus: H-96 decided, R3 postponed by the owner); minor (Current focus: the owner's audit of the night run's decisions first; the chat and the evening's decisions built but R3 and A10); minor (Current focus: the About dialog of item 1 fixed, A60 built with it); minor (Current focus: the owner's findings in a real window of 2026-10-14 evening first (node history from Prometheus, the About dialog), his decision round of the same evening as item 8 after the agent chat; the open questions answered; the terminal fork's two newer patches offered upstream later); minor (M10: `cogpit-cli` inside the app and the disk image the release's only download, D-16 revised by the owner, built, no tag run yet); minor (M10 in progress: the signed, notarized disk image built, no tag run yet; Current focus: the node shell as built confirmed by the owner, D-37; GitOps: Flux CD's section built, the milestone in progress, G23; M3, M4: the node shell built, D-37, a night decision for the owner to confirm; Current focus: its confirmation replaces the owner's decision on H-7; alpha gate: the downloadable macOS build exists since v0.1.0, for Apple silicon; Current focus after the day run: D-29 and the copy icons built; M6: a release upgraded to a version from the user's chart repositories; M7: the cluster Overview built; M1 and 0.1: `cogpit-cli contexts` and `get` built; M9: the stdio bridge built; current focus after the fifth run and the owner's smoke test; M12 in progress: the built-in agent's engine side built, 20–40 h left; Current focus item 1: the node history from Prometheus built in the night run (G12, A43)) · **Target:** Lens Desktop 2026.10.51501

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
| **Manage Helm releases in the cluster** (Lens's Helm view): list releases, read history and values, upgrade, roll back | partial (2026-10-12): releases from the cache with Lens's drawer (values, notes, resources), rollback with Lens's dialog, uninstall, and upgrade from the Helm Upgrade tab with the release's own chart or, since 2026-10-13, a version from the chart repositories the user added, crash-safe through a journal, on kind (`TestKindHelmReleases`, `TestKindHelmUpgrade`, `TestKindHelmUpgradeFromRepository`); not walked through by a person on a real cluster; some of Lens's behaviour still to capture | M6 | 5–15 |
| **Maintain a node:** cordon, drain and uncordon with confirmations; node details | partial: cordon, a crash-safe drain and uncordon, confirmed, on a three-node kind cluster through the engine (`TestKindDrainCrashFinishUncordon`); the node drawer as captured with its Pods table, without metrics; the GUI path tested offscreen only | M4 | 2–8 |
| **Use kubectl:** a terminal bound to the cluster's context and namespace | partial: works on kind (`TestKindEngineWorkflows`); a shell profile can override it ([H-20](../security/credentials.md#h-20-shell-profiles-can-override-the-terminals-kubeconfig)) | M3 | 5–20 |
| A downloadable macOS build (unsigned is enough) | yes for Apple silicon (2026-10-13): releases v0.1.0 to v0.4.0 carry `Cogpit_<version>_darwin_arm64.zip`, ad-hoc signed and not notarized, beside `SHA256SUMS` ([releasing › the release pipeline](../developer/releasing.md#the-release-pipeline)); v0.1.1's download checked (checksums, the app's version). Since 2026-10-14 the pipeline ships the app in a signed, notarized disk image instead (no tag run yet; [`build.yml`](../../.github/workflows/build.yml)). Intel Macs get no app yet, and the binary needs the build runner's macOS (26.0) (G14) | G14 | 0 (Intel: G14) |

**Result: fails.** Every workflow exists since the fourth run (2026-10-12): Helm upgrade and adding
and syncing kubeconfigs were built. None has been walked through end to end in the GUI by
a person on a real cluster, and no cloud credential plugin is verified (human).

## Milestones

| M | Status | Scope | Definition of done | Estimate |
|---|---|---|---|---:|
| **M0** Skeleton | 🟡 (go.mod, Makefile, lint with layering done; CI on GitHub since 2026-10-11, rebuilt 2026-10-13 as parallel jobs of at most ten minutes ([releasing › CI](../developer/releasing.md#ci)): lint, parity, GoSec, govulncheck, cyclomatic complexity, unit tests, the race tests in shards, kind and the drain cluster on 1.37, macOS-only tests and the app, the Windows compile, combined coverage; semantic-release live since 2026-10-13, each release with `cogpit-cli`, the unsigned app and checksums; the generated pages, the scorecard and the envtest job open) | `go.mod`, `Makefile`, golangci-lint with `depguard` layering, CI on macOS with compile checks for Windows and Linux, envtest and `kind` jobs, semantic-release, the parity and scorecard generators (G1) | `make lint test` green in CI; a forbidden import fails the lint; the generated parity pages match the hand-written checklists | 20–40 |
| **Spike** Fyne gate | ✅ (ran as the local MVP; [results](../developer/spike-fyne.md#results); D-1 decided for Fyne, with re-checks of S7 and S5) | Table, YAML editor, terminal and MCP control on Fyne ([spike plan](../developer/spike-fyne.md)) | Gate criteria S1–S7 recorded; [D-1](../developer/decisions.md#d-1-ui-toolkit) confirmed or the fallback chosen | 16–24 |
| **M1** Connect | 🟡 (contexts, connections, discovery, credential-safe errors; kubeconfig files and sync folders with credential plugins run only once allowed (2026-10-12); `cogpit-cli contexts` and `cogpit-cli get` built 2026-10-13 on the app's sources and approvals, tested on the fake API server, the demo and kind (`TestKindCLIContextsAndGet`); no cloud plugin verified) | `kubeconfig` and `cluster`: discovery, merge; every authentication form; cached discovery; access reviews; `cogpit-cli contexts` and `cogpit-cli get <kind>` (G2) | Connects to `kind` and envtest in CI with certificate and token auth; exec plugins verified on EKS, GKE and AKS (human); fuzzed kubeconfig parsing never panics | 25–50 |
| **M2** Browse | 🟡 (cache with scoped, metadata-only, local and virtual kinds, engine, shell, 41 built-in profiles, the Port Forwarding (local) and Helm Releases (virtual) views, the Workloads Overview page and the custom resource views, row menus from the registry, MCP navigation, access reviews; budgets P2 and P9–P11 met in tests; the Secrets view from metadata only (H-8); 9 views missing; the cluster overview built 2026-10-13) | The cache (G3), the engine and registry with UI state (G4), the Fyne shell: layout, sidebar from discovery, tables, drawer, namespace filter (part of G5); about 20 core views (part of G6); MCP `ui_state`, `resources_find`, `ui_navigate` (part of G17) | A 10,000-pod list meets budgets P2–P4; an idle view costs no list requests (P9); every built-in kind the cluster serves appears; an MCP client navigates the GUI to a Service by its cluster IP | 180–420 |
| **M3** Inspect | 🟡 (logs with follow, previous container, timestamps, wrap, filter, download and the logs of a workload's pods; cluster-bound terminal, pod shell, attach; the node shell since 2026-10-13, D-37) | Logs, terminal sessions, pod shell, attach, node shell (G8) | Budget P5 met; a terminal's `kubectl` talks to its own cluster; the node shell meets the rules in [privilege footprint](../security/privilege-footprint.md#node-shell) | 45–90 |
| **M4** Change | 🟡 (Edit with diff, dry run and a resourceVersion precondition; create from Cogpit's templates; Delete for every kind; Restart (Deployments, Daemon Sets, Stateful Sets), Scale (Deployments, Stateful Sets, Replica Sets), Cordon, Uncordon, Drain, Trigger, Suspend, Resume, Evict, Attach with confirmations; an interrupted drain is offered on the next connect; Edit and Delete follow the access reviews; the create dialogs, drafts and the node shell since, 2026-10-12 and 2026-10-13) | Editor with diff and server-side dry-run, create from templates, delete with confirmation, workload and node actions with the kubectl libraries (G9, part of G7) | Every Lens Desktop action on workloads and nodes; the command fuzz target green; confirmation enforced for the CLI and agents too; an interrupted drain is detected on the next start | 65–155 |
| **M5** Reach | 🟡 (built 2026-10-11: pod and service forwards on 127.0.0.1, the captured dialog, the Port Forwarding view, reconnects after a pod restart, on kind; the browser not yet seen opening in a real window) | Port forwarding: manager, dialog, view (G10) | Forward, open in browser, stop; listeners bound to `127.0.0.1` only | 2–6 |
| **M6** Helm | 🟡 (read side, rollback and uninstall 2026-10-11: the Releases view from the `owner=helm` Secrets, history, values and manifest, rollback and uninstall through the Helm SDK, a release left pending by Cogpit offered on the next connect (journal), on kind; upgrade from the Helm Upgrade tab with Lens's drawer and Rollback dialog (2026-10-12); missing: install, repositories and charts) | Helm SDK in-process: releases from the cache, install, upgrade, rollback, uninstall as crash-safe jobs; repositories and charts (G11) | The Helm workflow of the alpha gate passes on `kind`; a release left pending by a crash is offered for completion or rollback | 25–60 |
| **Alpha (macOS)** | ⬜ | M0–M6 and an unsigned macOS build | The alpha gate passes | ~135–370 left |
| **M7** Metrics | 🟡 (the Metrics Server and Prometheus together with a fallback, the drawers' charts, the CPU and Memory columns, a status line per provider; the cluster Overview with its charts and Issues, 2026-10-13; checked on kind with both providers) | Provider detection, cluster overview, charts (G12) | Metrics shown on `kind` with metrics-server and with Prometheus; no provider is a state, not an error; polling stops when nothing visible needs it | 40–80 |
| **M8** Interface parity | ⬜ | Clusters, hotbar, palette, menus, shortcuts, preferences, cluster settings, themes, deep links (rest of G5 and G13) | Every free Lens Desktop row in [ui-parity](ui-parity.md) done | 85–210 |
| **M9** MCP control | 🟡 (every command a tool, confirmation in the GUI, audit log file; the stdio bridge `cogpit-cli mcp` built 2026-10-13) | The rest of [beyond-lens › MCP](beyond-lens.md#mcp-control-of-ui-and-cluster): every command as a tool, dock tools, the GUI confirmation flow, the audit log, the stdio bridge (rest of G17) | An external agent completes the triage and change workflows through MCP, with every mutating step confirmed in the GUI | 20–40 |
| **M10** macOS distribution | 🟡 (2026-10-14: the app for Apple silicon signed with the Developer ID in a notarized, stapled disk image, `cogpit-cli` inside the app, built in the release pipeline; the disk image and `SHA256SUMS` the release's only files ([D-16](../developer/decisions.md#d-16-the-first-real-release) revised by the owner); no tag run yet) | Signed universal app in a notarized disk image, Homebrew cask, Sparkle 2 with an EdDSA-signed appcast, update notice and one-click update (rest of G14; [D-4](../developer/decisions.md#d-4-packaging-and-updates)) | Installs from the disk image and updates itself on a clean Mac without warnings | 20–40 |
| **GitOps** Flux CD and Argo CD | 🟡 (Flux CD built 2026-10-13: the navigator's section, curated profiles of its thirteen kinds, its dashboard, Reconcile, Suspend and Resume on objects and selections, D-33, D-34; checked on `kind` with Flux's definitions and no controllers, `TestKindFlux`; Argo CD not started) | Kind profiles, dashboards, Argo CD Sync and Refresh (G23, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | A Flux and an Argo CD installation on `kind` show their objects with status; Sync and Refresh work with confirmation | 55–115 |
| **Beta (macOS)** | ⬜ | M7–M10, the GitOps views, ~75% of views and actions, budgets P1–P11 met (G15), the first user bug backlog (G16) | Ready for real work ≥ ~75% on macOS; no gap marked *blocks beta* | ~425–995 left |
| **M11** Windows and Linux | ⬜ | ConPTY, X11 and Wayland, packages and updates per format, Ubuntu first (G22) | The alpha gate passes on Windows and on Ubuntu | 40–80 |
| **M12** Built-in agent | 🟡 (2026-10-14 late evening: the engine side built, `internal/agent`; the chat column being built) | Chat column, OpenAI-compatible endpoint (LM Studio, LiteLLM, Open WebUI), tool loop with D-2's confirmation rules (G21) | The agent answers questions about the selected object, navigates the UI on request, and runs a confirmed action, against a LiteLLM and an Open WebUI endpoint | 20–40 left |
| **M13** Paid-plan features | ⬜ | The remaining paid Lens features, built into the code: cloud account integrations, Security Center, cost monitoring (G18, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features), [D-5](../developer/decisions.md#d-5-extensions)) | Each matches its Lens Desktop rows | 110–220 |

M11 and M12 do not depend on each other; their order is the owner's call.

## Current focus

Ranked; each item links to its gap. Pick from the top unless an issue is assigned to you. Items 1
to 8 are **the path to alpha**: the gate above, and what the owner's use of Cogpit in a real
window found on 2026-10-15; the items after them run beside it. Updated 2026-10-15 evening, after
the owner's audit of the night run's decisions (done that morning) and two interviews with the owner that
evening.

1. **A release with files, and the first signed appcast** ([G14](gaps.md#g14-packaging-signing-and-updates-macos)).
   The owner set up Sparkle's update key and the environments `sparkle` and `github-pages` on
   2026-10-15, and the releases carry the key since v0.11.0; but v0.9.0 to v0.12.0 have no files,
   because Apple kept each notarization submission beyond the job's 45 minutes. Since 163664f2 a
   job of its own waits up to 330 minutes for the one submission; v0.13.0's was still running on
   2026-10-15 evening. When it passes, the owner approves its appcast jobs in `sparkle` and the
   lead checks the feed against [releasing › Setting up Sparkle](../developer/releasing.md#setting-up-sparkle)
   ("What the first signed appcast must show"); the release after it is the first real update. If
   Apple holds it again: re-run the failed job, never the signing job
   ([when it fails](../developer/releasing.md#when-signing-or-notarization-fails)). The gate's
   download row is met by v0.8.0; this item matters because the walks below should run on an
   installed release that has the features of 2026-10-15.
2. **Standby and lost connections** ([G2](gaps.md#g2-cluster-connection-and-authentication),
   [G16](gaps.md#g16-stability-and-the-user-bug-backlog)): the owner's finding of 2026-10-15
   evening, decided in an interview the same evening, not built (its D-entry comes with the
   build). Closing the MacBook drops every connection, and on waking the window hangs; nothing
   detects it today (the state stays connected), nothing reconnects, and requests other than
   watches have no timeout. Decided: reproduce the hang on kind (the control plane frozen with
   `docker pause`, a goroutine dump) and fix its cause; detect an interruption through macOS's wake
   notification, a heartbeat (`GET /version` every 15 s, two failures) and a broken watch; show it
   (an amber dot, "Reconnecting…", a banner with Reconnect now) while the views keep the cache's
   last state and every mutating command is disabled with the reason "Connection interrupted", for
   every origin; reconnect by itself with a backoff up to 30 s; on a lasting 401 run an approved
   credential plugin once, then "Login expired" with Log in again; logs resume with a marker line,
   port forwards dial again, shells end with a Reconnect button; a drain or a Helm operation caught
   by the break is handled as after a crash. Also in the owner's log, cause not known: `services`
   alone answering 401 every 5 minutes for two hours while other kinds worked. It blocks "Connect"
   on a real cluster.
3. **Choosing a cluster: one list** ([G13](gaps.md#g13-clusters-hotbar-palette-menus-shortcuts-and-preferences)):
   the owner's interview of 2026-10-15 evening, not built (its D-entry comes with the build). The
   context picker, the top bar's "Switch context ▾" and the navigator's scope dropdown go; the
   navigator's section (every cluster, grouped by kubeconfig file or folder) is the one list. ⌘P,
   the Cluster menu's item (renamed "Connect to Cluster…"), the palette's Connect to Cluster and a
   failing `--context` open it with its search focused, Escape on an empty search returns to the
   cluster it came from; at a cluster the header shows "‹ All Clusters"; a "+" in its header adds
   kubeconfigs; the welcome page keeps its cluster groups. Before the owner walks "Connect".
4. **The owner's audit answers of 2026-10-15, to build** ([G5](gaps.md#g5-app-shell-navigation-tables-and-drawer),
   [G9](gaps.md#g9-yaml-editor-and-create-resource), [G21](gaps.md#g21-built-in-agent)): a
   Secret's values edited in the drawer like a Config Map's, in the window only (no engine path an
   agent reaches), with a masked diff and a dry run; Save rebases by itself when none of the
   edited keys changed underneath, otherwise the conflict as built, never a silent overwrite; a
   Config Map's unsaved drawer edits kept as disk drafts (0600), a Secret's values never on disk;
   the selection bar from one checked row; `ui.navigate` opens view and drawer and leaves the
   checked rows alone (selection only through `ui.select`). The chat: "send all tools" per
   connection (off by default); a warning for an `http://` connection to another machine and a CA
   file per connection; a message edited and sent again with everything after it dropped (F9; its
   gap to document with it: the cut part's writes leave the model's view).
5. **The owner walks Cogpit in a real window** (human): what the runs of 2026-10-13 to 2026-10-15
   built and checked offscreen and in tests only. The hotbar (tiles dragged into order), the
   navigator's sections, the tab bar, the palette, Preferences and Cluster settings, the welcome
   page, the create dialogs, Helm install and upgrade from Helm's `repositories.yaml`, the copy
   icons, a TLS Secret's certificates and their chain's anchor, the cluster Overview with the
   metrics status line and the node history from Prometheus, the kubectl download, selection and
   Delete of a selection, Flux CD and Argo CD on a cluster that runs them, Create Secret, the node
   shell, About, the chat column (LM Studio; LiteLLM and Open WebUI live), the layout toggles and
   the Update button, the MCP gear, a Config Map's values in the drawer, the terminal's scrollback.
   What differs goes into the parity rows.
6. **Walk the alpha-gate workflows** on kind and on a real cluster (human): the seven workflows of
   the gate end to end in the GUI, sleep and wake included once item 2 is built; every partial in
   the gate becomes yes or a finding.
7. **Credential plugins verified on EKS, GKE and AKS** ([G2](gaps.md#g2-cluster-connection-and-authentication),
   human): the exec plugins of "Connect", with item 2's one plugin run on a lasting 401.
8. **What the walks find, and the depth the gate needs** ([G16](gaps.md#g16-stability-and-the-user-bug-backlog)
   and the gaps marked *blocks alpha*, G1–G11 and part of G13): the bulk of the ~110–320 h. Alpha
   also needs ready for real work in the ~40–75% band ([stages](README.md#stages)); it is ~30%
   (ROADMAP, estimated 2026-10-13), and the two discounts of ×0.7 in
   [mainstream practitioner](target-app-parity.md#mainstream-practitioner), stability on real
   clusters and the cloud credential plugins, are what items 2 and 5 to 7 address. Re-estimate
   after them.

Beside the path to alpha:

9. **Questions for a later round with the owner**, one at a time: the lead's decisions of
   2026-10-14 not presented yet (`grep -rn "for the owner to confirm" docs/`: the signed feed and
   the appcast jobs, the signing job and its entitlements, `cogpit-cli` in the bundle, the node
   shell's journal, the scrubbed base64 forms, the rests of D-33, D-36 and D-44), and the questions
   kept for when a feature is taken up: OCI registries and `helm registry login`
   ([H-29](../security/credentials.md#h-29-authenticated-helm-repositories)), an offline preview of
   an agent's Helm upgrade ([H-30](../security/trust-boundaries.md#h-30-chart-provenance-is-not-verified),
   [H-31](../security/privilege-footprint.md#h-31-an-agents-helm-values-can-copy-what-lookup-reads)),
   zoom, a preference allowing untrusted certificate authorities for traffic outside the cluster,
   the terminal's kubectl through the cluster's proxy, kubectl mirror presets, Sparkle's Beta
   channel.
10. **GitHub settings, only with the owner's OK** ([G14](gaps.md#g14-packaging-signing-and-updates-macos)):
    A10, the `v*` tag ruleset with the semantic-release GitHub App as its only bypass and the
    environment `release`'s admin bypass off (decided 2026-10-15; checked the same day: no ruleset
    exists); a LiteLLM key as a GitHub secret for an opt-in CI test of the chat.
11. **Postponed by the owner:** several windows (R3, [D-52](../developer/decisions.md#d-52-several-windows):
    model (a) as the direction, the rest in a separate interview once the rest is done); the
    chat's text selectable and copyable (F8: waits for the terminal patch being built).
12. **The terminal widget upstream** ([G8](gaps.md#g8-logs-terminal-and-shells),
    [D-14](../developer/decisions.md#d-14-terminal-widget)): fyne-io/terminal #155 and #157 merged
    (2026-10-15, 2026-10-14); #156 waits for review; #158 is rebased once #156 is in; a pull
    request to #147 (scroll support) and Cogpit's scrollback model offered there, waiting for the
    maintainer; then Cogpit's fork moves to upstream's master. The two newer patches (right-click
    copy and paste, Ctrl/Shift+Insert) are offered at the end, the app first.
13. **Foundation and beta work:** the generated parity pages and the scorecard
    ([G1](gaps.md#g1-workspace-ci-and-progress-generators)); captures of the rows marked inferred,
    nine of them asked for explicitly (the create dialogs, the navigator's root, the Helm Upgrade
    tab with a known repository, where Lens sets `HELM_DRIVER`, the hotbar's slots and bank arrows,
    a workload drawer's pods, the Argo CD and Flux CD views; human); the window measurements of
    [G15](gaps.md#g15-performance-and-api-load-at-scale) (P3, P5–P8) with the owner's OK to open a
    window; the universal app and the Homebrew cask (G14).

## What's next, with estimates

| Next | Scope | Estimate |
|---|---|---:|
| 0.1 "connects" | the rest of M0 and M1: the generators, the CLI lists contexts and objects of any cluster (the CLI part built 2026-10-13: `cogpit-cli contexts`, `cogpit-cli get`) | 24–54 h |
| 0.2 "browses" | the rest of M2: the cluster overview, the remaining views, captures of the inferred rows | 40–120 h |
| Alpha (macOS) | M3–M6 and an unsigned build: the seven core workflows | ~150–400 h total |
| Beta (macOS) | M7–M10, the GitOps views, performance, stability | ~440–1,020 h total |
| After the macOS beta | M11 Windows and Linux, M12 built-in agent, M13 paid-plan features; Teamwork only if a relay service is wanted (G20) | 190–380 h, Teamwork 150–300 h |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-15 | minor | Current focus: the chat's API keys in an owner-only file on every platform instead of the keychain (the owner's decision of 2026-10-15, built the same day), so "the keychain on Windows and Linux" is no longer open; M11's keychain backends removed |
| 2026-10-15 | minor | Current focus: the chat's H-96 built (a conversation in which a turn ran over a private connection is not kept), no longer open |
| 2026-10-15 | minor | Current focus: the chat's H-96 decided by the owner (not built), R3 postponed by him until the rest is done |
| 2026-10-15 | minor | Current focus: the owner's audit of the night run's decisions first; the chat and his decisions of 2026-10-14 evening built but R3 (designed, D-52) and A10 |
| 2026-10-14 | minor | Current focus item 1: the About dialog fixed (the build's own version, a development build saying so, a calmer layout) and decision A60 built (About Cogpit in macOS's application menu) |
| 2026-10-14 | minor | Current focus item 1: the node history from Prometheus built in the night run (node-exporter where the Prometheus has it, A43, G12) |
| 2026-10-14 | minor | Current focus: the owner's findings in a real window (2026-10-14 evening) as item 1, the node history from Prometheus and the About dialog; item 8 builds his decisions of the same evening (the decision index in decisions.md); the walk notes the open questions answered; the two newer terminal-fork patches offered upstream later, the app first (owner) |
| 2026-10-14 | minor | M10: D-16 revised by the owner, built (agent CLIBUNDLE): `cogpit-cli` inside `Cogpit.app`, signed and notarized with it, the release's CLI archives for six platforms dropped; no tag run yet, hours not moved |
| 2026-10-14 | minor | M10 in progress: the signed, notarized disk image and the signed, notarized `cogpit-cli` for macOS built in the release pipeline (G14, D-4), no tag run yet; the alpha gate's macOS build row says so; hours not moved |
| 2026-10-14 | minor | Current focus: the afternoon run's features for the owner's walk; M10's disk image and Sparkle built; the built-in agent chat pulled forward by the owner (item 6) |
| 2026-10-14 | minor | Current focus: the owner confirmed the node shell as built (D-37, D-38), with the default image the newest supported Alpine release and the maximum duration a setting |
| 2026-10-14 | minor | Current focus: the night build's features and its night decisions (D-33 to D-44) for the owner's walk |
| 2026-10-13 | minor | GitOps in progress: Flux CD's section, views, dashboard and actions built (G23, D-33, D-34); its hours left to the lead's recount |
| 2026-10-13 | minor | M3, M4: the node shell built (night build, D-37) and Create Secret (G9); Current focus: the owner's confirmation of the node shell as built |
| 2026-10-13 | minor | Alpha gate: the downloadable macOS build is there for Apple silicon since v0.1.0 (was "nothing to download") |
| 2026-10-13 | minor | Current focus after the day run: D-29 and the lists' copy icons built, item 2 folded into the owner's walk |
| 2026-10-13 | minor | Current focus: the owner's requirements of 2026-10-13 (D-29 Helm repositories from repositories.yaml, D-25 copy icons in the lists) as item 2 |
| 2026-10-13 | minor | Current focus after the night run: the owner walks its features in a real window first |
| 2026-10-13 | minor | M6: the Helm row names the upgrade to a version from the user's chart repositories (G11, night run; status and estimate unchanged) |
| 2026-10-13 | minor | M7 in progress: the cluster Overview built (G12, night run); M2's note no longer lists it (status only; estimates unchanged) |
| 2026-10-13 | minor | M9: the stdio bridge `cogpit-cli mcp` built (status only; estimate unchanged) |
| 2026-10-13 | minor | M1 and 0.1: `cogpit-cli contexts` and `cogpit-cli get` built (status only; estimates unchanged) |
| 2026-10-12 | minor | Current focus after the fifth run and the owner's smoke test: the hotbar and the metrics requirements first |
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
