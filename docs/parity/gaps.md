# Where Cogpit falls short of Lens

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (the MVP: evidence and new estimates for G1–G9, G14, G15, G17) · **Target:** Lens Desktop 2026.10.51501

Every known shortfall against Lens Desktop 2026.10.51501, and against Cogpit's own targets beyond
Lens, one entry each, ranked by what stands between Cogpit and alpha, then beta. This is the work
list: take the highest-ranked entry nobody is working on, and when you close part of one, update its
evidence and estimate here and the row in the parity page it points to
([rules](README.md#rules-for-every-progress-page)).

On 2026-10-11 the local MVP closed parts of G1–G9, G14, G15 and G17; their entries say what is
left, with evidence. Entries without code are the whole of their area. Numbers come from
[`target-app-parity.md`](target-app-parity.md) and the checklists in
[`ui-parity.md`](ui-parity.md), [`resource-parity.md`](resource-parity.md),
[`platform-parity.md`](platform-parity.md) and [`beyond-lens.md`](beyond-lens.md).

Hours are Opus 5.5 agent-hours, one agent working sequentially, including tests, estimated with
PhotoCraft's calibration (2–9 h per checklist item, 8–20 h per subsystem) until Cogpit has its
own. **Human** marks what an agent can't finish alone.

## Summary

| # | Gap | Kind | Impact | Estimate | Parity doc |
|---|---|---|---|---:|---|
| G1 | Workspace, CI and progress generators | foundation | blocks alpha | 12–28 h | [roadmap](roadmap.md#milestones) |
| G2 | Cluster connection and authentication | connectivity | blocks alpha | 18–40 h + human | [platform-parity](platform-parity.md#cluster-authentication-and-connectivity) |
| G3 | Cache and the resource model | foundation | blocks alpha | 25–55 h | [architecture §5](../developer/architecture.md#5-cache) |
| G4 | Engine and command registry | foundation | blocks alpha | 15–35 h | [architecture §4](../developer/architecture.md#4-the-ui-seam-the-most-important-interface) |
| G5 | App shell, navigation, tables and drawer | UI | blocks alpha | 60–165 h | [ui-parity](ui-parity.md#layout) |
| G6 | Resource views | features | blocks alpha (core kinds) | 120–360 h | [resource-parity](resource-parity.md) |
| G7 | Resource actions | features | blocks alpha (core actions) | 62–195 h | [resource-parity](resource-parity.md) |
| G8 | Logs, terminal and shells | features | blocks alpha | 30–70 h | [ui-parity](ui-parity.md#logs) |
| G9 | YAML editor and create resource | features | blocks alpha | 27–60 h | [ui-parity](ui-parity.md#editor-and-create-resource) |
| G10 | Port forwarding | features | blocks alpha | 10–20 h | [ui-parity](ui-parity.md#port-forwarding) |
| G11 | Helm | features | blocks alpha | 30–60 h | [resource-parity](resource-parity.md#helm) |
| G12 | Metrics and cluster overview | features | blocks beta | 40–80 h | [ui-parity](ui-parity.md#cluster-overview-and-charts) |
| G13 | Clusters, hotbar, palette, menus, shortcuts and preferences | UI | partly alpha, rest beta | 65–165 h | [ui-parity](ui-parity.md#clusters) |
| G14 | Packaging, signing and updates (macOS) | platforms | an unsigned build blocks alpha; signing and updates block beta | 22–45 h + human | [platform-parity](platform-parity.md#operating-systems-and-packages) |
| G15 | Performance and API load at scale | performance | blocks beta | 30–70 h | [testing](../developer/testing.md#performance-budgets-proposed-not-measured) |
| G16 | Stability and the user bug backlog | stability | blocks beta | 100–200 h | this page |
| G17 | MCP control | beyond Lens | basics in alpha, the rest beta | 12–30 h | [beyond-lens](beyond-lens.md#mcp-control-of-ui-and-cluster) |
| G18 | Paid Lens Desktop features | features | low-medium | 110–220 h + human | [resource-parity](resource-parity.md#paid-lens-desktop-views) |
| G19 | External extensions | ecosystem | not a target ([D-5](../developer/decisions.md#d-5-extensions)) | – | [decisions D-5](../developer/decisions.md#d-5-extensions) |
| G20 | Teamwork and remote access | ecosystem | not a target by default | 150–300 h + hosted service | [decisions D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features) |
| G21 | Built-in agent | beyond Lens | after alpha | 40–80 h + human | [beyond-lens](beyond-lens.md#built-in-agent) |
| G22 | Windows and Linux | platforms | after the macOS beta | 40–80 h + human | [platform-parity](platform-parity.md#operating-systems-and-packages) |
| G23 | Flux CD and Argo CD views | features | blocks beta ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | 55–115 h | [resource-parity](resource-parity.md#paid-lens-desktop-views) |

Totals (2026-10-11, after the MVP): Lens's free feature set on macOS (G1–G16) is **~670–1,650 h**
(was ~800–1,850); the beta adds the Flux CD and Argo CD views (G23); with Lens's other paid
features, Windows and Linux (G18, G22, G23), the full Lens target is **~870–2,060 h** (was
~1,000–2,300). Cogpit's own targets beyond Lens (G17, G21) add **~50–110 h** (was ~70–140).
G19 and G20 are not targets and not counted. The Fyne spike before M2 ran as part of the MVP and
D-1 is decided for Fyne ([results](../developer/spike-fyne.md#results)).

## Foundation gaps

### G1. Workspace, CI and progress generators
- **Missing:** CI on macOS, Windows and Linux with cgo and Fyne's build dependencies, the `kind`
  and envtest jobs, semantic-release as in the team's other repositories, the generators
  `make parity` and `make scorecard` that turn the checklists into measured pages
  ([method](README.md#the-pages)), and the README's build, coverage and Go Report Card badges once
  `release.yml` and the coverage JSON exist.
- **Evidence:** built on 2026-10-11: [`go.mod`](../../go.mod) (Go 1.27.2, pinned versions), the
  [`Makefile`](../../Makefile) with local targets, [`.golangci.yml`](../../.golangci.yml) with the
  team's linters and `depguard` rules for every layer (a forbidden import fails `make lint`;
  verified with probe imports). golangci-lint is pinned to v2.14.0 because v2.13 cannot read Go
  1.27.2 export data.
- **Impact:** without CI, nothing guards `main`.
- **Estimate:** 12–28 h (was 20–40).

### G2. Cluster connection and authentication
- **Missing:** kubeconfig discovery, merging and validation; pasted kubeconfigs stored owner-only;
  connections through client-go with every authentication form; the persister for the legacy
  `oidc` provider; the environment for exec plugins; access reviews; connection states and errors
  a user can act on; the per-terminal kubeconfig
  ([security › credentials](../security/credentials.md#the-terminals-kubeconfig)).
- **Evidence:** [platform-parity](platform-parity.md#cluster-authentication-and-connectivity),
  2 of 13 rows partial (certificates on kind, tokens on a fake server), 11 missing (the
  cloud-account row belongs to G18). Built: contexts from the merged kubeconfig files, connections
  through client-go's loading rules with credential-free errors, implicit kubeconfig writes
  switched off, cached discovery, the per-terminal kubeconfig ([`internal/cluster`](../../internal/cluster/),
  [`internal/kubeconfig`](../../internal/kubeconfig/)). Still missing from the list above: pasted
  kubeconfigs, the exec-plugin environment (only a `PATH` stopgap), access reviews.
- **Impact:** a user who cannot connect cannot use anything; exec plugins are how most cloud
  clusters authenticate. client-go carries the authentication itself, which is why this is smaller
  than it was in the Rust plan.
- **Estimate:** 18–40 h (was 25–50). **Human:** accounts on EKS, GKE and AKS to verify the plugins against.

### G3. Cache and the resource model
- **Missing:** the cache of [architecture §5](../developer/architecture.md#5-cache): shared,
  reference-counted informers per cluster, kind and namespace scope; the warm period; no resync;
  bookmarks and streaming lists; metadata-only informers; the `managedFields` transform; discovery
  and access reviews cached; polling only while visible; request counters and the diagnostics
  view. The kind-profile framework, the generic profile for custom resources, status tokens;
  snapshots coalesced to at most ten per second per view.
- **Evidence:** built on 2026-10-11 ([`internal/cache`](../../internal/cache/), [`internal/model`](../../internal/model/)):
  shared, reference-counted informers, the warm period, no resync, streaming initial lists, the
  `managedFields` transform, request counters, the kind-profile framework with status tokens, and
  coalesced snapshots (≤ 10 per second per view). P9–P11 hold in tests
  ([testing](../developer/testing.md#performance-budgets-proposed-not-measured)). Still missing:
  namespace-scoped informers (views filter client-side), metadata-only informers, discovery cached
  on disk, access reviews, polling sources, the diagnostics view, the generic custom-resource
  profile.
- **Impact:** every view, every MCP read and the agent depend on it; it decides budgets P1–P4 and
  P9–P11 and the load Cogpit puts on API servers.
- **Estimate:** 25–55 h (was 45–90).

### G4. Engine and command registry
- **Missing:** `Session`, the command registry with schemas, verbs, danger levels and disabled
  reasons; UI state in the engine; jobs and cancellation; events; the confirmation policy per
  origin; the audit log; the command fuzz target; the first `cogpit-cli` commands.
- **Evidence:** built on 2026-10-11 ([`internal/engine`](../../internal/engine/)): `Session`, the
  registry with JSON schemas, verbs, danger levels and disabled reasons, UI state, events, the
  confirmation policy per origin (`TestDispatchConfirmationPolicy`), the audit log, `FuzzDispatch`,
  21 commands. Still missing: jobs and cancellation, access reviews, `cogpit-cli` commands.
- **Impact:** every action, menu, shortcut and MCP tool dispatches through it.
- **Estimate:** 15–35 h (was 35–70).

## Feature gaps

### G6. Resource views
- **Missing:** all but six of the 64 resource views ([resource-parity](resource-parity.md)). The alpha gate needs
  about 20: Pods, Deployments, Daemon Sets, Stateful Sets, Replica Sets, Jobs, Cron Jobs, Nodes,
  Namespaces, Events, Services, Ingresses, Config Maps, Secrets, Persistent Volume Claims, Helm
  Releases, the cluster overview and the generic custom-resource view.
- **Evidence:** resource-parity, 6 of 64 partial (Nodes, Pods, Deployments, Services, Namespaces,
  Events; no metrics columns).
- **Estimate:** 120–360 h (was 130–380; 2–6 h each; Pods, Nodes and the overview at the high end).

### G7. Resource actions
- **Missing:** all 40 actions. The alpha gate needs about 20: logs, shell, attach, edit, delete,
  scale, restart, cordon, uncordon, drain, trigger, the port-forward actions and the Helm release
  actions. Drain and rollout come from the kubectl libraries; drain is crash-safe
  ([D-11](../developer/decisions.md#d-11-crash-stance)).
- **Evidence:** resource-parity, 4 of 40 partial: Logs, Shell, Restart (Deployment), Delete (pods
  only).
- **Estimate:** 62–195 h (was 70–210).

### G8. Logs, terminal and shells
- **Missing:** the terminal tab on `fyne-io/terminal`, connected to Cogpit's own PTY (local shell
  with its own kubeconfig) and to client-go exec and attach streams; the node shell with the
  safeguards in [privilege footprint](../security/privilege-footprint.md#node-shell); the log
  viewer (virtualized, follow, filter, timestamps, previous container, download, multi-pod).
- **Evidence:** [ui-parity](ui-parity.md#logs), 4 of 16 rows partial across logs and terminal:
  the terminal tab on Cogpit's own PTY with its own kubeconfig, the pod shell over exec, the log
  view with follow ([`internal/terminal`](../../internal/terminal/),
  [`termview`](../../internal/ui/fyneui/termview/)). Still missing: attach, the node shell, log
  options (timestamps, previous container, filter, download, multi-pod), and a terminal widget
  hardened beyond the patched copy Cogpit builds against
  ([H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)).
- **Impact:** triaging a failing pod (logs, then a shell) is the most common Lens workflow.
- **Estimate:** 30–70 h (was 45–90), plus upstreaming or replacing the widget.

### G9. YAML editor and create resource
- **Missing:** an editor that keeps its own lines and fills a viewport-sized grid (Fyne's TextGrid
  stores every character, [spike](../developer/spike-fyne.md#known-before-the-spike)), YAML
  highlighting, apply after a diff and a server-side dry-run, templates, create dialogs' YAML
  fallback, drafts kept for crash recovery.
- **Evidence:** [ui-parity](ui-parity.md#editor-and-create-resource), 5 rows missing. Built: the
  editor widget that keeps its own lines and draws only the viewport, with YAML highlighting
  ([`yamlview`](../../internal/ui/fyneui/yamlview/), spike S4); it is used read-only in the drawer.
  Still missing: editing wired to an Edit tab, apply after a diff and a server-side dry-run,
  templates, drafts.
- **Estimate:** 27–60 h (was 35–70).

### G10. Port forwarding
- **Missing:** the forward manager (local listener on `127.0.0.1`, lifecycle, reconnects), the
  forward dialog, opening in a browser. The Port Forwarding view and its actions count in G6 and G7.
- **Evidence:** [ui-parity](ui-parity.md#port-forwarding), 3 rows missing.
- **Estimate:** 10–20 h.

### G11. Helm
- **Missing:** the Helm SDK in-process: releases, history, values and manifests from the cached
  release Secrets; install, upgrade, rollback and uninstall as crash-safe jobs; repositories, chart
  index and search; the Helm Install and Helm Upgrade dock tabs. Views and actions count in G6 and G7.
- **Evidence:** [resource-parity](resource-parity.md#helm), 2 views and 4 actions missing.
- **Estimate:** 30–60 h.

### G12. Metrics and cluster overview
- **Missing:** provider detection (metrics-server, Prometheus through the service proxy, the
  provider variants Lens lists), polling while visible, charts, the overview page, the explicit
  Lens Metrics install.
- **Evidence:** [ui-parity](ui-parity.md#cluster-overview-and-charts), 5 rows missing, plus the
  metrics rows of cluster settings.
- **Estimate:** 40–80 h.

### G23. Flux CD and Argo CD views
- **Missing:** kind profiles with columns, status and searchable fields for Flux CD (GitRepository,
  OCIRepository, HelmRepository, Bucket, ExternalArtifact, Kustomization, HelmRelease,
  ImageRepository, ImagePolicy, ImageUpdateAutomation, Alert, Provider, Receiver) and Argo CD
  (Applications, Application Sets, Projects, repositories, clusters); the two dashboards; Argo CD's
  Sync (with Prune) and Refresh (Normal or Hard), in bulk too
  ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)). Without
  them, the resources still appear as generic custom-resource views.
- **Evidence:** [resource-parity](resource-parity.md#paid-lens-desktop-views), Flux CD and Argo CD
  rows.
- **Impact:** GitOps is daily work for the owner's team; in Lens it needs a paid plan.
- **Estimate:** 55–115 h (2–4 h per kind profile, the dashboards, two actions), built into the
  code like every integration ([D-5](../developer/decisions.md#d-5-extensions)).

## UI gaps

### G5. App shell, navigation, tables and drawer
- **Missing:** the Fyne window layout, the cluster sidebar, the table, the detail drawer and the
  dock frame; Cogpit's Fyne theme. Needs the [spike](../developer/spike-fyne.md) to pass.
- **Evidence:** [ui-parity](ui-parity.md), 18 of 39 rows partial across Layout, Cluster sidebar,
  Tables, Detail drawer and Dock ([`internal/ui/fyneui`](../../internal/ui/fyneui/), Cogpit's theme
  with status tokens). The spike's gate results are recorded for the owner's decision
  ([spike](../developer/spike-fyne.md#results)). Still missing: title bar with history, tab bar,
  welcome page, row actions menu and right-click, keyboard navigation, CSV export, column reorder
  and hiding.
- **Estimate:** 60–165 h (was 80–200).

### G13. Clusters, hotbar, palette, menus, shortcuts and preferences
- **Missing:** kubeconfig syncs and pasting (alpha needs these), the hotbar, the command palette,
  the application menu, every keyboard shortcut, preferences, cluster settings, appearance and
  application integration.
- **Evidence:** [ui-parity](ui-parity.md), 65 rows across those sections (6 more are marked not a
  target).
- **Estimate:** 65–165 h (1–2.5 h per row; the registry makes shortcuts and menus cheap).

## Platform gaps

### G14. Packaging, signing and updates (macOS)
- **Missing:** the universal macOS app bundle in a disk image, Developer ID signing and notarization, the Homebrew
  cask, checksums and SBOM, the signed update manifest, the update notice and the one-click update
  ([D-4](../developer/decisions.md#d-4-packaging-and-updates)).
- **Evidence:** [platform-parity](platform-parity.md#operating-systems-and-packages), the macOS and
  update rows.
- **Impact:** an unsigned build is enough for alpha testers; beta users need a notarized app and
  updates.
- **Evidence (2026-10-11):** `make app` builds an ad-hoc signed `Cogpit.app` for arm64 locally.
- **Estimate:** 22–45 h (was 25–50). **Human:** Apple Developer ID, the update signing key.

### G22. Windows and Linux
- **Missing:** everything platform-specific beyond macOS
  ([D-7](../developer/decisions.md#d-7-language-and-platforms)): ConPTY for terminals, the keychain
  backends of Windows and Linux, X11 and Wayland, MSI and portable builds for x64 and arm64,
  AppImage, deb, rpm, tar.xz and Flatpak for x86_64 and aarch64 (Ubuntu first), winget, Scoop and
  Flathub, the one-click update per format, and the UI and terminal tests on each system. CI
  compiles for both from M0, so the code stays portable until then.
- **Evidence:** [platform-parity](platform-parity.md#operating-systems-and-packages), the Windows
  and Linux rows.
- **Estimate:** 40–80 h. **Human:** Windows code-signing material.

## Quality gaps

### G15. Performance and API load at scale
- **Missing:** budgets P1, P3, P5–P8 on the GPU renderer and against Lens on the same machine
  ([testing](../developer/testing.md#performance-budgets-proposed-not-measured)), the nightly run on a
  fixed runner, request counts for the scenarios not covered yet. Measured on 2026-10-11: P2,
  P3, P4, P7, P8, P9–P11 (GPU numbers on one Mac, no Lens comparison); `make bench` and
  `cogpit --perf-mode gate` run the measurements.
- **Start-up reservation of gate S1** (owner, 2026-10-11): every window stalls its main thread for
  0.3–0.56 s while Fyne sets up GL, and the first fill of a 10,000-row table for 151–193 ms (text
  textures). To try: render the first frame before the rows arrive, warm the text textures of the
  first screen off the critical frame, and avoid second windows ([spike S1](../developer/spike-fyne.md#results)).
- **Impact:** large clusters are where users notice an IDE first, and where an IDE costs the API
  server most.
- **Estimate:** 30–70 h (was 40–80).

### G16. Stability and the user bug backlog
- **Missing:** nothing to fix yet. PhotoCraft's first public week produced hundreds of issues on
  features its tests counted as done; this entry reserves the time for the same after Cogpit's
  first release.
- **Estimate:** 100–200 h to beta.

## Beyond Lens

### G17. MCP control
- **Missing:** the MCP server over the registry with the UI tools of
  [beyond-lens](beyond-lens.md#mcp-control-of-ui-and-cluster), Streamable HTTP with a bearer token,
  the stdio bridge, the GUI side of the confirmation flow, the audit log view
  ([D-2](../developer/decisions.md#d-2-agent-interface-scope)). `ui_state`, `resources_find` and
  `ui_navigate` come early (M2), because they also let agents test the UI.
- **Evidence:** beyond-lens, 9 done and 6 partial of the 17 MCP rows: tools from the registry,
  Streamable HTTP on loopback with a bearer token and `Host`/`Origin` checks, `ui_state`,
  `resources_find`, `ui_navigate` and the other UI tools, the GUI confirmation for agents, the audit
  log file ([`internal/mcp`](../../internal/mcp/), `TestShowcaseServiceByClusterIP`). Still missing:
  the stdio bridge (so a registration survives restarts), `view_rows`, reading dock tabs, the audit
  log in the GUI, the per-context allowance in the GUI.
- **Estimate:** 12–30 h (was 30–60).

### G21. Built-in agent
- **Missing:** the chat tab, the OpenAI-compatible client, the tool loop over the registry, limits,
  the data-egress controls, settings with the API key in the keychain
  ([D-10](../developer/decisions.md#d-10-built-in-agent)).
- **Evidence:** beyond-lens, 10 rows missing.
- **Estimate:** 40–80 h. **Human:** a LiteLLM or Open WebUI instance with a tool-calling model to
  test against.

## Ecosystem gaps

### G18. Paid Lens Desktop features
- **Missing:** EKS, AKS and GKE account integrations (their Go SDKs); Security Center (Trivy
  Operator); cost monitoring. Flux CD and Argo CD are G23, in the beta; the Hotbar and the editor's
  diff are built with G13 and G9; Lens's Ask AI and MCP server are superseded by G21 and G17
  ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)).
- **Evidence:** [resource-parity](resource-parity.md#paid-lens-desktop-views), Security Center and
  Cost Monitoring; [platform-parity](platform-parity.md#cluster-authentication-and-connectivity).
- **Impact:** high for users of Lens's paid plans, none for the mainstream audience.
- **Estimate:** 110–220 h. **Human:** cloud accounts; a cost data source (owner decision).

### G19. External extensions
- **Missing:** a way to install extensions from outside the project, as Lens does with npm
  packages.
- **Impact:** not a target ([D-5](../developer/decisions.md#d-5-extensions)): integrations are built
  into Cogpit's code; Lens's JavaScript extensions could not run in Cogpit anyway.
- **Estimate:** not counted.

### G20. Teamwork and remote access
- **Missing:** Lens's Teamwork spaces and Cluster Connect relay.
- **Impact:** not a target unless the owner decides to run a relay service
  ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)).
- **Estimate:** 150–300 h plus a hosted service; not counted in the totals.

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | The MVP: G1–G9, G14, G15 and G17 re-estimated with evidence; new terminal-widget risk in G8 |
| 2026-10-11 | minor | D-5: G19 (external extensions) is not a target and leaves the totals |
| 2026-10-11 | minor | D-9: Flux CD and Argo CD views split from G18 into G23, in the beta |
| 2026-10-11 | minor | macOS first (D-7): G14 covers macOS, G22 Windows and Linux after the macOS beta |
| 2026-10-11 | major | Re-estimated for Go (G2, G7, G11, G18 smaller; G3 larger for the cache; G9 own editor); G17 is MCP control of UI and cluster; G21 built-in agent added; G14 includes one-click updates; Freelens rows removed |
| 2026-10-11 | major | First version: G1–G20 ranked, before any code |
