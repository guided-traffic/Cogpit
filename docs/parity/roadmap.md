# Roadmap

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-15 · **Change:** minor (Current focus: H-96 decided, R3 postponed by the owner); minor (Current focus: the owner's audit of the night run's decisions first; the chat and the evening's decisions built but R3 and A10); minor (Current focus: the About dialog of item 1 fixed, A60 built with it); minor (Current focus: the owner's findings in a real window of 2026-10-14 evening first (node history from Prometheus, the About dialog), his decision round of the same evening as item 8 after the agent chat; the open questions answered; the terminal fork's two newer patches offered upstream later); minor (M10: `cogpit-cli` inside the app and the disk image the release's only download, D-16 revised by the owner, built, no tag run yet); minor (M10 in progress: the signed, notarized disk image built, no tag run yet; Current focus: the node shell as built confirmed by the owner, D-37; GitOps: Flux CD's section built, the milestone in progress, G23; M3, M4: the node shell built, D-37, a night decision for the owner to confirm; Current focus: its confirmation replaces the owner's decision on H-7; alpha gate: the downloadable macOS build exists since v0.1.0, for Apple silicon; Current focus after the day run: D-29 and the copy icons built; M6: a release upgraded to a version from the user's chart repositories; M7: the cluster Overview built; M1 and 0.1: `cogpit-cli contexts` and `get` built; M9: the stdio bridge built; current focus after the fifth run and the owner's smoke test; M12 in progress: the built-in agent's engine side built, 20–40 h left; Current focus item 1: the node history from Prometheus built in the night run (G12, A43)) · **Target:** Lens Desktop 2026.10.51501

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
| **M11** Windows and Linux | ⬜ | ConPTY, keychain backends, X11 and Wayland, packages and updates per format, Ubuntu first (G22) | The alpha gate passes on Windows and on Ubuntu | 40–80 |
| **M12** Built-in agent | 🟡 (2026-10-14 late evening: the engine side built, `internal/agent`; the chat column being built) | Chat column, OpenAI-compatible endpoint (LM Studio, LiteLLM, Open WebUI), tool loop with D-2's confirmation rules (G21) | The agent answers questions about the selected object, navigates the UI on request, and runs a confirmed action, against a LiteLLM and an Open WebUI endpoint | 20–40 left |
| **M13** Paid-plan features | ⬜ | The remaining paid Lens features, built into the code: cloud account integrations, Security Center, cost monitoring (G18, [D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features), [D-5](../developer/decisions.md#d-5-extensions)) | Each matches its Lens Desktop rows | 110–220 |

M11 and M12 do not depend on each other; their order is the owner's call.

## Current focus

Ranked; each item links to its gap. Pick from the top unless an issue is assigned to you.

1. **The owner's audit of the night run's decisions (2026-10-14/15), first:** every choice the
   owner had not made was weighed and recorded where it lives as "the lead's decision for the owner
   to confirm" (`grep -rn "for the owner to confirm" docs/` after 2026-10-14): D-10 (the agent),
   D-14 (the fork's races), D-22 (a free colour's legible shade), D-25 (the copy icons), D-28
   (Remove from Cogpit bound to the contexts named), D-29 and D-11 (Helm), D-33 (Reconcile with
   Source), D-50 (no fallback when Rekor cannot be asked), D-51 (the navigator's state), the
   README's pictures, and the security pages.
2. **The owner's findings from the real window (2026-10-14 evening), both built, to check in a real window:** the node history
   from Prometheus where it is available (no past data shows for a node; built 2026-10-14 night:
   node-exporter where the Prometheus has it, A43, [G12](gaps.md#g12-metrics-and-cluster-overview));
   the About dialog, which shows version 0.0.1 and is laid out confusingly (where the version comes
   from, a development build saying so; [G13](gaps.md#g13-clusters-hotbar-palette-menus-shortcuts-and-preferences);
   fixed 2026-10-14, with decision A60, About in the application menu: not tried in a real window yet).
3. **The owner walks the night runs', the day run's and the night build's features in a real window** (2026-10-15: the chat column with his LM Studio and the night run's changes first; 2026-10-14;
   every one was verified offscreen and in tests only; the night build's: the Flux CD and Argo CD sections
   on a cluster that runs them, Create Secret, the node shell, the workloads' drawers' Pods, tabs restored
   after a restart, a namespace-limited user's views; then its night decisions D-33 to D-44): the hotbar, the navigator's sections, the
   tab bar, the command palette, Preferences and Cluster settings, the welcome page, the create
   dialogs, Helm install and upgrade from the repositories of Helm's own `repositories.yaml`
   (built the same day, [D-29](../developer/decisions.md#d-29-helm-chart-repositories-from-helms-repositoriesyaml):
   a `helm repo add` while Cogpit runs, an authenticated repository), the copy icons in the lists
   ([D-25](../developer/decisions.md#d-25-copying-values)), the cluster Overview and the metrics
   status line, kubectl for terminals, row selection and Delete of a selection; record what
   differs in the parity rows. The open questions (each marked "Cogpit's default, not
   the owner's decision" in its row or gap; D-29's in its notes and
   [H-40](../security/credentials.md#h-40-credentials-over-plain-http), H-41 and H-42 in [trust boundaries](../security/trust-boundaries.md))
   were answered in the owner's decision round of 2026-10-14 evening (item 8; the rows and gaps say
   "confirmed" or "decided … not built").
   Since the afternoon of 2026-10-14 also: the http repository question and its banner (H-40), a
   terminal tab's agent marker and its 15-minute end (H-83), Preferences › App's Start at login and
   update rows, Help › Check for Updates…, a `cogpit://` link (D-47), the signed disk image of
   v0.5.0 on a Mac that has never run Cogpit.
4. **Walk the alpha-gate workflows in the GUI** on kind and a real cluster (owner): change a
   workload, maintain a node, reach a service, Helm releases, adding and syncing kubeconfigs,
   Cogpit's own title bar.
5. **The terminal widget's upstream pull requests** ([#155–#158](https://github.com/fyne-io/terminal/pull/158),
   G8; #157 approved 2026-10-12) and the two new widget patches of the night run (right-click
   copy and paste, Ctrl/Shift+Insert: to be offered upstream later, at the end; the app first,
   owner, 2026-10-14 evening).
6. **Credential plugins verified on EKS, GKE and AKS** ([G2](gaps.md#g2-cluster-connection-and-authentication), human).
   The node shell as built was confirmed by the owner on 2026-10-14 ([D-37](../developer/decisions.md#d-37-the-node-shell);
   [H-7](../security/privilege-footprint.md#h-7-node-shell-is-root-on-the-node) stays: it is root on the node).
7. **M0 and M10:** the generated parity pages and the scorecard ([G1](gaps.md#g1-workspace-ci-and-progress-generators)),
   captures of the rows marked inferred, packaging and updates for macOS ([G14](gaps.md#g14-packaging-signing-and-updates-macos)).
   Built 2026-10-14: the signed, notarized disk image (first tag run v0.5.0) and Sparkle 2, which
   updates nothing until the owner has made the update key and the environments `sparkle` and
   `github-pages` ([releasing › Setting up Sparkle](../developer/releasing.md#setting-up-sparkle)).
8. **The built-in agent chat** ([G21](gaps.md#g21-built-in-agent), [D-10](../developer/decisions.md#d-10-built-in-agent)):
   built in the night run of 2026-10-14/15 (the engine side, the chat column at the right edge,
   Preferences › Agent, the private connection's reads; checked against the owner's LM Studio);
   open: a LiteLLM run in CI once the key is a GitHub secret, Open WebUI and LiteLLM live, the
   keychain on Windows and Linux, and a private connection's conversations no longer kept (decided
   by the owner 2026-10-15, not built,
   [H-96](../security/agents-and-mcp.md#h-96-a-private-turns-reads-are-kept-where-the-scrubber-does-not-know-them)).
9. **The rest of the owner's decisions of 2026-10-14 evening** (the decision index in
   [decisions](../developer/decisions.md#decision-round-2026-10-14-evening)): built in the night
   run of 2026-10-14/15 (hotbar tiles reordered by drag too, R2, a paid row,
   [ui parity › Hotbar](ui-parity.md#hotbar)) but two: a tile dragged out of the window opening
   another window (R3), designed in [D-52](../developer/decisions.md#d-52-several-windows) and postponed by the
   owner until the rest is done (model (a) as the direction, the rest in a separate interview); and the ruleset for `v*` tags (A10), a GitHub setting, only with the owner's
   go-ahead ([G14](gaps.md#g14-packaging-signing-and-updates-macos)).

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
