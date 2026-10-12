# Where Cogpit falls short of Lens

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-13 · **Change:** minor (G5, G13: the welcome page, the Help and Window menus, Disconnect and Remove from Cogpit built, four rows partial, estimates not moved; G8: the log tab's pod and container selectors, older lines and jump to a time partial; G8: the log tab's pod and container selectors and older lines partial; G8: the log tab's pod and container selectors partial; G11: a release upgraded to a version from the user's repositories; G5, G13: the tab bar of cluster views built, its five shortcut rows, the navigator's ⇧↵ fixed tab, estimates not moved; G13: Preferences and Cluster settings built, night run, 11 rows partial, G8's terminal rows 2, estimates not moved; G12: the cluster Overview built, the remainder 15–35 h; G6: Applications, Vertical Pod Autoscalers and the four Device Resources views partial; G6: Vertical Pod Autoscalers and the four Device Resources views partial; G6: the four Device Resources views partial; G11: chart repositories, the Charts view and the install built; G12: the owner's evening requirements built (both providers together, the status line, the fallback, a missing value as an empty cell; measured on kind); G13: the command palette and its shortcuts built, estimate 50–140 h, the summary table not recounted; G2: `cogpit-cli contexts` and `get` built; G5, G13: the navigator's header, root, tooltips and keyboard built; G12: the owner's evening requirements (provider status line, Prometheus history then Metrics Server live) and the open step question; G16: the quit crash fixed; G12: Prometheus in the window, checked against a real Prometheus on kind; G12: Prometheus, backend; G2: Lens's Kubeconfig Syncs captures followed, scope C2; totals not recounted) · **Target:** Lens Desktop 2026.10.51501

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
| G1 | Workspace, CI and progress generators | foundation | blocks alpha | 3–8 h | [roadmap](roadmap.md#milestones) |
| G2 | Cluster connection and authentication | connectivity | blocks alpha | 10–28 h + human | [platform-parity](platform-parity.md#cluster-authentication-and-connectivity) |
| G3 | Cache and the resource model | foundation | blocks alpha | 16–38 h | [architecture §5](../developer/architecture.md#5-cache) |
| G4 | Engine and command registry | foundation | blocks alpha | 10–28 h | [architecture §4](../developer/architecture.md#4-the-ui-seam-the-most-important-interface) |
| G5 | App shell, navigation, tables and drawer | UI | blocks alpha | 35–110 h | [ui-parity](ui-parity.md#layout) |
| G6 | Resource views | features | blocks alpha (core kinds) | 30–100 h | [resource-parity](resource-parity.md) |
| G7 | Resource actions | features | blocks alpha (core actions) | 12–45 h | [resource-parity](resource-parity.md) |
| G8 | Logs, terminal and shells | features | blocks alpha | 16–40 h | [ui-parity](ui-parity.md#logs) |
| G9 | YAML editor and create resource | features | blocks alpha | 8–22 h | [ui-parity](ui-parity.md#editor-and-create-resource) |
| G10 | Port forwarding | features | blocks alpha | 2–6 h | [ui-parity](ui-parity.md#port-forwarding) |
| G11 | Helm | features | blocks alpha | 4–12 h | [resource-parity](resource-parity.md#helm) |
| G12 | Metrics and cluster overview | features | blocks beta | 40–80 h | [ui-parity](ui-parity.md#cluster-overview-and-charts) |
| G13 | Clusters, hotbar, palette, menus, shortcuts and preferences | UI | partly alpha, rest beta | 50–140 h | [ui-parity](ui-parity.md#clusters) |
| G14 | Packaging, signing and updates (macOS) | platforms | an unsigned build blocks alpha; signing and updates block beta | 22–45 h + human | [platform-parity](platform-parity.md#operating-systems-and-packages) |
| G15 | Performance and API load at scale | performance | blocks beta | 28–66 h | [testing](../developer/testing.md#performance-budgets-proposed-not-measured) |
| G16 | Stability and the user bug backlog | stability | blocks beta | 100–200 h | this page |
| G17 | MCP control | beyond Lens | basics in alpha, the rest beta | 12–30 h | [beyond-lens](beyond-lens.md#mcp-control-of-ui-and-cluster) |
| G18 | Paid Lens Desktop features | features | low-medium | 110–220 h + human | [resource-parity](resource-parity.md#paid-lens-desktop-views) |
| G19 | External extensions | ecosystem | not a target ([D-5](../developer/decisions.md#d-5-extensions)) | – | [decisions D-5](../developer/decisions.md#d-5-extensions) |
| G20 | Teamwork and remote access | ecosystem | not a target by default | 150–300 h + hosted service | [decisions D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features) |
| G21 | Built-in agent | beyond Lens | after alpha | 40–80 h + human | [beyond-lens](beyond-lens.md#built-in-agent) |
| G22 | Windows and Linux | platforms | after the macOS beta | 40–80 h + human | [platform-parity](platform-parity.md#operating-systems-and-packages) |
| G23 | Flux CD and Argo CD views | features | blocks beta ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | 55–115 h | [resource-parity](resource-parity.md#paid-lens-desktop-views) |

Totals (2026-10-12, after the fourth implementation run, the sums of the table): Lens's free
feature set on macOS (G1–G16) is **~420–1,040 h** (417–1,036; was ~430–1,060 after the third run);
the beta adds the Flux CD and Argo CD views (G23); with Lens's other paid features, Windows and
Linux (G18, G22, G23), the full Lens target is **~620–1,450 h** (was ~640–1,480). The fourth run
re-estimated G1, G2 and G11 from what landed, with the same borrowed calibration. Cogpit's own
targets beyond Lens (G17, G21) add **~50–110 h** (unchanged).
G19 and G20 are not targets and not counted. The Fyne spike before M2 ran as part of the MVP and
D-1 is decided for Fyne ([results](../developer/spike-fyne.md#results)).

## Foundation gaps

### G1. Workspace, CI and progress generators
- **Missing:** the generated pages (`parity-checklist.md`) and `make scorecard`
  ([method](README.md#the-pages); later, by [D-15](../developer/decisions.md#d-15-progress-generators));
  an envtest job (no test uses envtest yet); the
  first real release, which is the owner's call ([releasing](../developer/releasing.md#the-first-real-release)).
- **Evidence:** built on 2026-10-11: [`go.mod`](../../go.mod) (Go 1.27.2, pinned versions), the
  [`Makefile`](../../Makefile) with local targets, [`.golangci.yml`](../../.golangci.yml) with the
  team's linters and `depguard` rules for every layer (a forbidden import fails `make lint`;
  verified with probe imports). golangci-lint is pinned to v2.14.0 because v2.13 cannot read Go
  1.27.2 export data. CI as workflow files (2026-10-11,
  [releasing › CI](../developer/releasing.md#ci)): [`release.yml`](../../.github/workflows/release.yml)
  calls [`build.yml`](../../.github/workflows/build.yml) on every push and pull request: macOS runs
  `make lint`, the race tests with coverage and `make build`; Linux and Windows compile and link
  every package and test with cgo and Fyne's dependencies; a Linux job runs `make test-kind` on a
  kind cluster; govulncheck. On `main` it commits the coverage badge (84.5 % measured locally) and
  runs semantic-release as a dry run ([`.releaserc.json`](../../.releaserc.json), the
  `conventionalcommits` preset as in the team's other repositories); Renovate runs from
  [`renovate.yml`](../../.github/workflows/renovate.yml). The README carries the build, coverage,
  Go Report Card and licence badges. Verified locally: the macOS job's `make` targets pass,
  actionlint, the Renovate config validator, the dry run against a stand-in for the GitHub API.
  Observed on GitHub (2026-10-11): the first run failed Compile (linux) and the vulnerability
  check (GLFW 3.4 needs the Wayland headers; fixed in the workflow) and the kind job (a flaky
  test, fixed in the cache); from then on every job is green on `main`, including the kind job on
  Kubernetes 1.37.0, govulncheck (after `golang.org/x/net` v0.60.0 and `golang.org/x/image`
  v0.43.0), the coverage badge commit and the semantic-release dry run, which created no tag and
  no release.
  Built on 2026-10-12: `make parity` ([`hack/parity`](../../hack/parity/main.go), D-15), a CI
  step after the lint: it refuses duplicated rows, rows marked done or partial without evidence,
  command ids the registry does not know, links to missing files and unknown test names, and
  compares its counts with the numbers ROADMAP and the progress pages report
  ([rules](README.md#make-parity)); its first run found six partial rows without evidence and a
  platform summary that counted 7 partial rows instead of 8, and reproduced the 121 free
  interface rows (the 123 counted the two paid shortcuts). The dependency backlog of 2026-10-12
  (44 Renovate pull requests) was held by flaky tests, not by the updates
  ([releasing › Renovate](../developer/releasing.md#renovate)).
- **Impact:** CI guards `main`; `make parity` keeps the hand-written checklists honest until the
  generated pages exist.
- **Estimate:** 3–8 h (was 6–14; `make parity` built, 2026-10-12).

### G2. Cluster connection and authentication
- **Lens's captures of 2026-10-12 followed** ([ui parity › Clusters](ui-parity.md#clusters), built 2026-10-12, scope C2): dot files in sync
  folders read (regular files only, at most 1 MiB, a FIFO never opened: `readBounded` also looks before it opens now); a connected
  cluster closed when its file is deleted or its source removed; one dialog, "Sync Files and Folders", for files and folders;
  `~/.kube` a default synced folder (removable, the removal remembered); a connect's log ("Connection Updates") behind a
  warning icon in the picker; owner requirement: removing a synced item asks, naming the path, that only the link is removed.
  **Still open:** the plugin's standard error in the connect log (client-go writes it to Cogpit's own standard error; capturing it
  means replacing the process-wide `os.Stderr`, not done); the ↻ beside the ⚠; Lens's system file dialog instead of Cogpit's own
  browser (a native `NSOpenPanel` for files and folders needs cgo per platform). Since D-28 (built 2026-10-12, C3) `config` is an
  ordinary file of `~/.kube` and every context comes from one file read alone, so a deleted `config` closes its connections as any
  synced file does. A plugin shared by several contexts of one file is approved in one dialog that lists
  every context with its server (owner, 2026-10-12, built by C4), stored per context.
- **Decided (owner, 2026-10-12, [D-17](../developer/decisions.md#d-17-token-files-named-by-added-kubeconfigs)):**
  a `tokenFile` of an added or synced kubeconfig outside its folder is asked for in the
  plugin approval dialog (H-24); **built 2026-10-12** (`TestTokenFileOutsideTheFolderIsAskedFor`;
  what remains is in H-24). The same run bound a program or CA file inside the kubeconfig's folder
  by content (review R-5) and made the dialog show the proxy, insecure and own-CA settings (R-11).
- **Decided (owner, 2026-10-12, [D-21](../developer/decisions.md#d-21-pasting-a-kubeconfig)):**
  no pasted kubeconfigs; the paste feature built on 2026-10-12 is removed (done 2026-10-12;
  sources of kind `pasted` from earlier builds are no longer listed, their files stay).
- **Missing:** kubeconfig discovery, merging and validation;
  connections through client-go with every authentication form; the persister for the legacy
  `oidc` provider; the environment for exec plugins; access reviews; connection states and errors
  a user can act on; the per-terminal kubeconfig
  ([security › credentials](../security/credentials.md#the-terminals-kubeconfig)).
- **Evidence:** [platform-parity](platform-parity.md#cluster-authentication-and-connectivity),
  2 of 13 rows partial (certificates on kind, tokens on a fake server), 11 missing (the
  cloud-account row belongs to G18). Built: contexts from the merged kubeconfig files, connections
  through client-go's loading rules with credential-free errors, implicit kubeconfig writes
  switched off, cached discovery, the per-terminal kubeconfig ([`internal/cluster`](../../internal/cluster/),
  [`internal/kubeconfig`](../../internal/kubeconfig/)). Access reviews built on 2026-10-11
  ([`cluster/access.go`](../../internal/cluster/access.go),
  [`engine/accessreviews.go`](../../internal/engine/accessreviews.go)): `SelfSubjectRulesReview` per
  namespace and `SelfSubjectAccessReview` where rules cannot answer, with their own request budget,
  cached per connection until refreshed or reconnected; measured, connecting costs one rules review
  and settling the sidebar one more access review for a user limited to a namespace
  (`TestBudgetAccessReviews`; on kind with a ServiceAccount token, `TestKindAccessReviews`). Built
  on 2026-10-12 (fourth run): kubeconfig files and sync folders added in Cogpit, each file read on its own and offered as
  `<context> [<source>]`, sync folders watched by polling; a source's exec credential plugin runs
  only after the user allowed it in the GUI, bound to a fingerprint of the command, arguments,
  environment and server; adding, removing and allowing refused for agents and the CLI
  ([`engine/kubeconfigs.go`](../../internal/engine/kubeconfigs.go); `TestExecPluginRunsOnlyOnceAllowed`,
  `TestSourcesFileIsOwnerOnly`, `TestAddedFilesAreNeverWritten`, `TestSyncFolderPicksUpAndDropsFiles`;
  on kind `TestKindAddedKubeconfig`, `TestKindExecPluginAfterApproval`); exec plugins partial in
  [platform-parity](platform-parity.md#cluster-authentication-and-connectivity). Still missing from
  the list above: the exec-plugin environment (only a `PATH` stopgap); Lens's own dialogs for
  adding kubeconfigs are not captured (the built ones are inferred), and `~/.kube/` is not watched
  without setup. Built 2026-10-13 (milestone 0.1): `cogpit-cli contexts` and `cogpit-cli get` over the
  same sources and approvals as the app (shared through [`internal/app`](../../internal/app/app.go)),
  read-only through a headless engine, a plugin the user has not allowed refused before it runs
  (`TestContextsListsTheSourcesWithoutSecrets`, `TestGetRunsNoPluginTheUserHasNotAllowed`,
  `TestGetKeepsSecretsSecret`, on kind `TestKindCLIContextsAndGet`).
- **Impact:** a user who cannot connect cannot use anything; exec plugins are how most cloud
  clusters authenticate. client-go carries the authentication itself, which is why this is smaller
  than it was in the Rust plan.
- **Estimate:** 10–28 h (was 14–34; kubeconfig sources and exec plugins built, fourth run). **Human:** accounts on EKS, GKE and AKS to verify the plugins against.

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
  on disk, polling sources, the diagnostics view (access reviews are cached since 2026-10-11, G2).
  Built in the second run
  (2026-10-11): the generic custom-resource profile from a definition and its printer columns,
  kept current through a watch on the cluster's definitions
  ([`customresources.go`](../../internal/model/kinds/customresources.go),
  [`customkinds.go`](../../internal/engine/customkinds.go), `TestBudgetCustomResourceView`); a
  handle reports synced only once the initial list reached its subscribers
  (`TestSyncWaitsForTheInitialListToReachSubscribers`). Built in the fourth run (2026-10-12):
  covering scopes ([architecture §5](../developer/architecture.md#scopes-local-and-virtual-kinds),
  [`cover.go`](../../internal/cache/cover.go)): a label-selected scope reads from the running
  unselected scope of the same resource, namespace and metadata flag instead of a watch of its own,
  with handovers both ways whose events stay exact under churn (`TestCoverHandoversUnderChurn`,
  `TestBudgetSecretScopes`, `TestKindSecretScopes`).
- **Impact:** every view, every MCP read and the agent depend on it; it decides budgets P1–P4 and
  P9–P11 and the load Cogpit puts on API servers.
  Fourth run (2026-10-12): the related-objects indexes are kept current by their kind's events
  instead of being rebuilt per lookup, and a drawer shares its indexes with its fetches and wakes
  only for the keys it shows: a node drawer's fetch on the demo cluster fell from 13–33 ms to
  2.6 ms (p50), its fetches from 14–43 % to 2.3–2.5 % of a core
  ([architecture §4.4](../developer/architecture.md#44-tables-details-editors-and-streams),
  `TestNodeDrawerUnderDemoChurn`, `TestLiveIndexBuiltDuringChurn`).
- **Estimate:** 16–38 h (was 22–50; scopes with label selectors and metadata only, local and virtual kinds and the related-objects index built, third run).

### G4. Engine and command registry
- **Missing:** `Session`, the command registry with schemas, verbs, danger levels and disabled
  reasons; UI state in the engine; jobs and cancellation; events; the confirmation policy per
  origin; the audit log; the command fuzz target; the first `cogpit-cli` commands.
- **Evidence:** built on 2026-10-11 ([`internal/engine`](../../internal/engine/)): `Session`, the
  registry with JSON schemas, verbs, danger levels and disabled reasons, UI state, events, the
  confirmation policy per origin (`TestDispatchConfirmationPolicy`), the audit log, `FuzzDispatch`,
  21 commands (41 after the second run, with the confirmation bound to its objects and their
  UIDs). Access reviews in `Enabled` built on 2026-10-11: a command whose verbs the connected
  user's reviews deny is greyed "not permitted: you may not …" in every menu and refused for every
  origin; answers never block a menu and arrive as events (`TestAccessReviewsAnswerWithoutBlocking`,
  `TestAccessReviewsGreyCommands`); `cluster.refreshAccess` asks again (the Cluster menu's Refresh Access Reviews, 2026-10-12), and the View menu, like the sidebar, leaves out the kinds the user may not list (2026-10-12). Still missing: jobs and
  cancellation, `cogpit-cli` commands.
- **Impact:** every action, menu, shortcut and MCP tool dispatches through it.
- **Estimate:** 10–28 h (was 15–35; access reviews in `Enabled`, drawer-only commands, third run).

## Feature gaps

### G6. Resource views
- **Missing:** 3 of the 64 resource views ([resource-parity](resource-parity.md), 2026-10-13: the
  cluster Overview, Helm Charts, Pod Security Policies; Applications, Vertical Pod Autoscalers and
  the four Device Resources views partial since 2026-10-13), and the depth of the partial ones
  (metrics columns, texts and rules checked against a capture). Of the about 20 the alpha gate
  needs, still missing: the cluster overview.
- **Evidence:** resource-parity, 27 of 64 (2026-10-11): Nodes, Pods, Deployments, Services,
  Namespaces, Events (the MVP); Daemon Sets, Stateful Sets, Replica Sets, Jobs, Cron Jobs
  ([`kinds/`](../../internal/model/kinds/) `daemonsets.go` … `cronjobs.go`; Cron Jobs with the
  captured columns and texts, Next execution from the schedule); Ingresses, Ingress Classes (the
  captured columns), Config Maps, Persistent Volume Claims; Definitions (no group filter) and the
  custom resource views (done: titled by Kind, grouped by API group, printer columns of priority
  0, following the definitions live); the ten Gateway API views, shown as generic custom resource
  views. Golden, status-text and fuzz tests per kind
  (`TestGolden` and `TestCustomGolden` over 119 synthetic objects), views filled from watches on kind
  (`TestKindEngineWorkflows`, `TestKindCustomResources`). Inferred and marked per row: most status
  rules, the Cron Jobs warning rule, the drawers of the new kinds. Third implementation run
  (2026-10-11): the 13 Config views beyond Config Maps and Secrets, all partial: Resource Quotas,
  Limit Ranges, Horizontal Pod Autoscalers, Pod Disruption Budgets (the captured columns with the
  warning icon and "N/A"), Priority Classes with the pod counts of 2026.9 (the related-objects
  index `PodsByPriorityClass`), Runtime Classes, Leases, Mutating and Validating Webhook
  Configurations, and the four admission policy views, the validating ones falling back to v1beta1
  and v1alpha1 through the profiles' `FallbackVersions`, the mutating ones at v1 only
  ([`kinds/`](../../internal/model/kinds/) `resourcequotas.go` … `validatingadmissionpolicybindings.go`;
  golden, status-text and fuzz tests per kind; `TestBuiltinAdmissionPoliciesFollowTheServedVersion`;
  the PDB list rendered offscreen). On kind 1.36.1 (`TestKindConfigViews`): a budget, a quota, a
  limit range and an autoscaler created in `cogpit-c` fill their views with what the controllers
  report, the priority classes count the cluster's pods, a node's lease links its node, and the
  runtime class, webhook and admission policy views sync (the test writes no cluster-scoped
  objects, so those show no rows). Inferred and marked per row: the status and warning rules, the
  HPA Metrics and Status forms, the Pods column's place, the drawers; Vertical Pod Autoscalers (a
  CRD) are not covered.
  Access Control (2026-10-11, third run): Service Accounts, Cluster Roles, Roles, Cluster Role Bindings and Role Bindings
  partial, with the captured columns ([`kinds/rbac.go`](../../internal/model/kinds/rbac.go) and
  one file per kind); the warning rules (a role or binding that grants nothing; a binding's
  missing cluster role read through the related-objects index `ClusterRolesByName`) and the
  drawers (rules and subjects as tables, links to roles and service accounts) inferred; 24
  synthetic objects in the goldens, the views filled on kind (`TestKindAccessControlViews`);
  Pod Security Policies (clusters before 1.25) and the Create dialogs still missing.
  Network and storage (2026-10-11, third run): Endpoint Slices, Endpoints, Network Policies, Persistent
  Volumes, Storage Classes and Replication Controllers partial (the documented columns; cell
  formats, status rules and drawers inferred; [`kinds/`](../../internal/model/kinds/)
  `endpointslices.go` … `replicationcontrollers.go`, golden, status-text and fuzz tests per kind);
  the related-objects index's first users: a claim's Pods column and drawer item
  (`TestClaimPodsColumnFollowsPods`), a node's Pods table as captured, CPU and Memory empty until
  metrics (`TestNodeDrawerPodsFollowPods`, under the demo's churn `TestNodeDrawerUnderDemoChurn`),
  a Service's endpoint slices in its drawer (`TestServiceEndpointSlices`); links to storage
  classes, volumes, claims and pods; on kind 1.36.1, a claim provisioned by the default class and
  mounted by a pod fills the Pods column, its volume, the node's Pods table and the Service's
  endpoint slices (`TestKindStorageAndNetworkViews`).
  Secrets and Helm (2026-10-11, third run): Secrets partial, then from a metadata-only watch (Keys
  and Type empty), the drawer's one GET per open, values masked and revealed in the GUI only;
  Helm Releases partial, a virtual kind over the `owner=helm` Secrets
  ([`releases.go`](../../internal/helm/releases.go), G11). Fourth run (2026-10-12), as the owner
  revised [D-13](../developer/decisions.md#d-13-cache): the Secrets view on a full watch, Keys and
  Type filled as in Lens, the drawer read from the cache without a GET, values still masked and
  revealed in the GUI only, and redacted in every result an agent reads
  ([`secrets.go`](../../internal/model/kinds/secrets.go), [`secretdata.go`](../../internal/engine/secretdata.go),
  `TestBudgetSecretsViewAndDrawer`, `TestAgentsReadSecretsRedacted`, `TestKindSecretsView`); Helm's
  releases read from that watch while it runs and from their own `owner=helm` watch otherwise
  (covering scopes, G3, `TestBudgetSecretsAndReleasesShareOneWatch`).
  Device Resources (2026-10-13): Resource Claims, Resource Claim Templates, Device Classes and
  Resource Slices partial, at `resource.k8s.io/v1` only as Lens reads them, with the columns, the
  claim's State texts and the warning icon's tooltip of Lens's documentation (no capture;
  [`resourceclaims.go`](../../internal/model/kinds/resourceclaims.go) and one file per kind; 21
  synthetic objects in the goldens, `TestDeviceResourcesColumns`, `TestResourceClaimsStatusTexts`);
  Resource Slices' Allocated from the related-objects index `ClaimsByPool` (`TestResourceSliceAllocated`);
  the pod drawer's Resource Claims section without the claim's state (`TestPodResourceClaimsSection`);
  offered at v1 only (`TestBuiltinDeviceResourcesFollowTheServedVersion`), filled on kind 1.36.1
  (`TestKindDeviceResources`).
  Inferred and marked per row: which device conditions warn, the colours, the drawers. Still
  missing: the node drawer's Devices section.
  Vertical Pod Autoscalers (2026-10-13): partial, Cogpit's first curated profile of a custom
  resource kind (`kinds.ProfileFor`, chosen by the custom kinds watch while the definition serves
  v1 or v1beta2): the documented default columns, the status rule, hidden Target, CPU and Memory
  columns and the drawer inferred ([`verticalpodautoscalers.go`](../../internal/model/kinds/verticalpodautoscalers.go),
  `TestVerticalPodAutoscalersFollowTheirDefinition`; on kind with the definition only,
  `TestKindVerticalPodAutoscalers`). The demo cluster serves no definitions, so it shows none.
  Applications (2026-10-13): partial, a virtual kind made of several kinds: the cached workloads
  grouped per namespace by `app.kubernetes.io/instance` ([`applications.go`](../../internal/model/kinds/applications.go)),
  served from a cache of its own whose source holds the workloads' handles only while something
  reads the applications ([`engine/applications.go`](../../internal/engine/applications.go);
  one watch per workload kind and no other request, `TestBudgetApplicationsView`; the handles go
  back after the view closes, `TestApplicationsReleaseTheWorkloads`); Logs as `application.logs`,
  a `workload.logs` tab per workload (`TestApplicationLogsOpensATabPerWorkload`); the demo's
  deployments carry the recommended labels. Inferred and marked: Status, Violations, which
  workloads count, the drawer.
- **Estimate:** 12–60 h (was 30–100; 2–6 h each for the 3 missing views, plus depth: captures, metrics columns).

### G7. Resource actions
- **Missing:** the actions not listed under Evidence, of 39 (Share is not a target, owner 2026-10-11). The alpha gate needs about 20: logs, shell, attach, edit, delete,
  scale, restart, cordon, uncordon, drain, trigger, the port-forward actions and the Helm release
  actions. Rollout comes from the kubectl libraries.
- **Evidence:** resource-parity: Logs, Shell, Restart (Deployment), Delete (pods only) partial (the
  MVP); Cordon, Uncordon, Drain, Scale (Deployment, Stateful Set), Trigger, Suspend, Resume (Cron
  Job) and Evict (Pod) partial (2026-10-11), as engine commands with the captured places and
  applicability, tested against client-go's fake clientset and the kind cluster
  ([`commands/nodes.go`](../../internal/engine/commands/nodes.go),
  [`commands/workloads.go`](../../internal/engine/commands/workloads.go)). Restart (Daemon Set,
  Stateful Set) and Scale (Replica Set) partial (third run, 2026-10-11): kubectl rollout restart's
  patch, compared with the one kubectl computes (`TestRestartWorkloadSendsKubectlsPatch`),
  confirmed for every origin ([`commands/workloadactions.go`](../../internal/engine/commands/workloadactions.go)). Drain uses kubectl's
  drain library and is crash-safe ([D-11](../developer/decisions.md#d-11-crash-stance)): a journal
  records it before the node is cordoned, and the next connect of its context offers to finish,
  uncordon or dismiss it ([`drains.go`](../../internal/engine/drains.go)). The row menus and the
  drawer's icons show them from the registry (G5, `TestRegistryMenusAsLens`); Scale and Drain
  open a dialog generated from their schemas, prefilled (the replicas from the cache, the drain
  options with their defaults). Still open for these: Lens's drain and scale dialogs (not
  captured; drain defaults inferred). The drain test passed on a three-node kind cluster
  (2026-10-11, `make test-kind-drain`, `TestKindDrainCrashFinishUncordon`: a drain blocked by a
  PodDisruptionBudget and a simulated crash leave the journal entry, the next session offers it,
  finishing and uncordoning clear it).
  Fourth run (2026-10-12, [`journallock.go`](../../internal/engine/journallock.go)): the drain and
  Helm journals are shared by every Cogpit process of the user under a file lock, and each entry
  names its process, so a drain or Helm action running in another live Cogpit is reported running
  there, not offered as interrupted, and cannot be finished or dismissed from here
  (`TestJournalsKeepEveryProcessesEntries`, `TestJournalEntryOfAKilledProcess`; [architecture §8](../developer/architecture.md#8-helm-internalhelm)).
  Restart and Scale send the UID they act on in their patches (2026-10-12), which the API server
  checks: a workload (Deployment, Daemon Set, Stateful Set, Replica Set) created again under the name is never restarted
  or scaled, whatever the origin, also when Cogpit's cache has not seen it yet
  (`TestKindPatchesBoundToTheUID` on Kubernetes 1.36.1, `TestRestartAndScaleCarryTheUID`;
  [architecture §4.2](../developer/architecture.md#42-the-command-registry)).
  Create dialogs (2026-10-12, [resource-parity › Create dialogs](resource-parity.md#create-dialogs)):
  Namespace, Service Account, Role, Cluster Role, Role Binding, Cluster Role Binding and Resource
  Quota partial, behind their views' "+", as engine commands that check the fields, dry-run and
  create ([`commands/createdialogs.go`](../../internal/engine/commands/createdialogs.go),
  [`createdialog.go`](../../internal/ui/fyneui/createdialog.go); `TestKindCreateDialogs` on
  Kubernetes 1.36.1); their fields are inferred, Lens's dialogs not captured. Bindings to
  cluster-admin and cluster-wide ones are confirmed for every origin (the lead's default, not the
  owner's decision; [H-27](../security/privilege-footprint.md#h-27-a-binding-is-judged-by-its-roles-name-and-the-cache)).
  Still open: the Secret's dialog, the node shell, Helm install, captures of Lens's dialogs.
- **Estimate:** 12–45 h, estimated (was 20–70; 3 of 39 actions missing: the node shell, Helm install and the Secret's dialog, plus depth and captures).

### G8. Logs, terminal and shells
- **Missing:** the terminal tab on `fyne-io/terminal`, connected to Cogpit's own PTY (local shell
  with its own kubeconfig) and to client-go exec and attach streams; the node shell with the
  safeguards in [privilege footprint](../security/privilege-footprint.md#node-shell); the log
  viewer (virtualized, follow, filter, timestamps, previous container, download, multi-pod).
- **Evidence:** [ui-parity](ui-parity.md#logs), 13 of 16 rows partial across logs and terminal
  (was 10 before 2026-10-13, 4 before the third run): the terminal tab on Cogpit's own PTY with
  its own kubeconfig, the pod shell over exec, the log view with follow ([`internal/terminal`](../../internal/terminal/),
  [`termview`](../../internal/ui/fyneui/termview/)); from the third run (2026-10-11): attach to a
  container's main process ([`terminal/attach.go`](../../internal/terminal/attach.go)), and the
  log options as engine state of the tab (`logs.setOptions`): timestamps, previous container, word
  wrap, search with a filter and `/`, download, and the logs of a controlling resource
  (`workload.logs`, `logs.controller`: every pod's containers from the cache, joining while
  following, each line named) ([`engine/logs.go`](../../internal/engine/logs.go),
  [`engine/workloadlogs.go`](../../internal/engine/workloadlogs.go),
  [`logtab.go`](../../internal/ui/fyneui/logtab.go)). Night run (2026-10-13): the log tab's pod
  and container selectors (`logs.sources`, `logs.setSource`): a controlling resource's merged
  stream narrows without opening the streams that stay again, and every stream carries the
  server's timestamps ([`engine/logsources.go`](../../internal/engine/logsources.go),
  [`logtab_sources.go`](../../internal/ui/fyneui/logtab_sources.go)); older lines on an
  explicit "Load older lines" row (`logs.loadOlder`), asked as a longer tail and put on top
  without repeating a line ([`engine/logsolder.go`](../../internal/engine/logsolder.go),
  [`logtab_older.go`](../../internal/ui/fyneui/logtab_older.go)); a jump to a time from a text
  field (`logs.jumpTo`), searching the held lines first and asking `sinceTime` only for what is
  not held ([`engine/logsjump.go`](../../internal/engine/logsjump.go),
  [`logtab_jump.go`](../../internal/ui/fyneui/logtab_jump.go)); every Logs row now exists,
  checked on kind by `TestKindLogSourcesOlderAndJump`. Still missing: the node shell. The terminal
  widget (2026-10-12, fourth run): the patched copy survives every input found by fuzzing it
  alone, without the escape filter (`FuzzHandleOutput`, `FuzzRawWidget`; the crash, hang and
  race patches in [PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)), and its tests run
  in CI (`make test-fork`); its upstream series is prepared, not sent
  ([H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)).
  Fourth run (2026-10-12): a workload's logs resume a follow stream the server ended while its
  container runs, from the newest line's timestamp without repeating lines, and forget ended pods
  and container instances (`TestWorkloadLogsResumeADroppedStream`,
  `TestWorkloadLogsForgetEndedInstances`; [architecture §4.4](../developer/architecture.md#44-tables-details-editors-and-streams)).
  Since 2026-10-13 (G13's Preferences and Cluster settings): the local terminal's shell, its
  arguments and the terminals' font size are preferences, and a cluster's terminals start in its
  working directory with its namespace (`TestLocalTerminalOptionsFromTheSettings`); two rows
  partial (font size, shell path and arguments), one updated.
- **Impact:** triaging a failing pod (logs, then a shell) is the most common Lens workflow.
- **Decided (owner, 2026-10-12, [D-14](../developer/decisions.md#d-14-terminal-widget)):** the
  patched terminal widget stays; its patches are offered upstream and it is fuzzed on its own,
  the two inputs that still make it panic alone patched first (3–8 h). Built 2026-10-12 except
  the offer, which waits for the owner.
- **Estimate:** 16–40 h (was 30–70; log options, workload logs and attach built, third run); the terminal widget's fuzzing and patches done (fourth run), sending the upstream series under 1 h.

### G9. YAML editor and create resource
- **Missing:** drafts kept for crash recovery, "Revert to original", the Create resource tab's save
  and search buttons, Delete of a selection, the Secret create dialog (the other seven partial since 2026-10-12, G7)
  and their YAML fallback, a Secrets path (**built 2026-10-12**, `TestSecretEditAndCreateAreGUIOnly`; decided 2026-10-12, [D-19](../developer/decisions.md#d-19-editing-and-creating-secrets-from-yaml): Secrets editable and creatable from YAML in the GUI only, the values from the cache since the full watch, agents refused; 4–8 h).
- **Evidence:** [ui-parity](ui-parity.md#editor-and-create-resource): 4 rows partial, 1 missing;
  [resource-parity](resource-parity.md#every-kind): Edit and Delete partial. Built on 2026-10-11:
  an Edit tab per object on the editable [`yamlview`](../../internal/ui/fyneui/yamlview/), filled
  from the cache; Save shows the line diff against the object as the cache holds it, then a
  server-side dry run, and Apply writes it as an update with the resourceVersion the tab started
  from, so a concurrent change comes back as a conflict with the new diff and is never overwritten
  ([`engine/editor.go`](../../internal/engine/editor.go)); Create resource tabs with Cogpit's own
  templates and multi-document create, each object dry-run first
  ([`engine/create.go`](../../internal/engine/create.go)); Delete for every kind with the cached UID
  as precondition; all of it as commands agents drive too, tested against fakes, fuzzed, and on kind
  (`TestKindEditCreateConflictDelete`). Edit and Delete are greyed "not permitted" when the access
  reviews deny `update` or `delete` on the object (2026-10-11, `TestEditAndDeleteFollowAccessReviews`,
  `TestKindAccessReviews`).
- **Estimate:** 8–22 h (was 10–25; Edit and Delete follow the access reviews, third run).

### G10. Port forwarding
- **Decided (owner, 2026-10-12, [D-20](../developer/decisions.md#d-20-a-port-forward-that-cannot-connect)):**
  a forward that cannot connect for a permanent reason (forbidden, unauthorized, the namespace
  gone) turns Failed at once and frees its port; passing errors keep the backoff; **built 2026-10-12**
  (404, 403 and 401 from the dial; `TestPortForwardFailsAtOncePermanentErrors`, `TestDialErrorsAreClassified`).
- **Missing:** Lens's status texts in the Port Forwarding view and its Edit dialog (capture
  needed; Cogpit's are inferred), the access review of the pod behind a forward before Edit offers
  it, a check in a real window that the browser opens (offscreen tests stop at Fyne's `OpenURL`).
  The Port Forwarding view and its actions count in G6 and G7.
- **Evidence:** [ui-parity](ui-parity.md#port-forwarding): 2 rows done, 1 partial. Built on
  2026-10-11: a forward manager in the engine ([`engine/portforward.go`](../../internal/engine/portforward.go),
  [`portforward_run.go`](../../internal/engine/portforward_run.go)) that listens on `127.0.0.1` only
  ([`cluster/portforward.go`](../../internal/cluster/portforward.go) `ListenLoopback`) and carries
  each local connection over a port-forward connection (WebSocket first, SPDY as the fallback, as
  kubectl 0.37); forwards are objects of the local kind `portforwards`
  ([`kinds/portforwards.go`](../../internal/model/kinds/portforwards.go)), so the view, the drawer,
  row menus and agents read them like cluster objects; reconnects come from the cache's events (a
  pod forward while its pod runs, a service forward to another ready pod of its selector) and from
  dial retries after 1 s doubling to 30 s; every forward of a context ends when Cogpit leaves it;
  commands `pod.portForward`, `service.portForward`, `portforward.open`, `portforward.edit`,
  `portforward.stop`, `portforward.remove`
  ([`commands/portforward.go`](../../internal/engine/commands/portforward.go)); the
  captured dialog and the port links ([`fyneui/portforward.go`](../../internal/ui/fyneui/portforward.go)).
  Tested against a fake forwarder and a fake kubelet (SPDY), and on kind: HTTP through forwards to
  the seed's service and pod, and a forward that reaches the replacement of a deleted pod
  (`TestKindPortForward`).
- **Estimate:** 2–6 h (was 10–20; built in the third run, left: Lens's status texts and Edit dialog by capture, a check in a real window).

### G11. Helm
- **Followed from Lens's captures of 2026-10-12** ([resource parity › Helm](resource-parity.md#helm),
  built 2026-10-12, was 2–5 h): the drawer's title on one line without Details and YAML tabs, Chart
  as "<chart>-<version>", Namespace as text, a grey status pill, "User-supplied values only"
  unchecked at first, Save as a full-width bar; Rollback enabled with the current revision (a new
  revision equal to it, as helm does; what Lens does is not captured); the Upgrade tab's separate
  version picker, its info line with links, and the editor's find widget (`findbar.go`). Cogpit
  keeps upgrading with the release's stored chart where Lens needs a repository (deliberate), so
  the picker offers that version only and the info line's spinner lasts only while the values load.
- **Missing:** authenticated, plain-http and OCI repositories
  ([H-29](../security/credentials.md#h-29-authenticated-helm-repositories)); a list of public
  repositories to pick; chart icons and the README in the chart's drawer; the ConfigMaps and SQL
  storage drivers; captures of Lens's Charts view, chart drawer and Helm Install tab (Cogpit's
  are inferred). Views and actions count in G6 and G7.
- **Evidence (2026-10-13, night run, the upgrade from repositories):** a release whose chart a
  repository added to Cogpit lists (by name) is upgraded to a version of it: the Upgrade tab's
  version picker with the repository's versions and a repository picker when several list the
  chart (inferred: Lens with a repository is not captured), the engine's choice (`helm.release.upgradeTab` with `repository`, `version`: the repository's versions newest
  first, prereleases after, the release's own marked; a new tab on the newest when it is newer,
  Cogpit's defaults), `helm.release.upgrade` with `version` and `repository`, the chart downloaded
  and checked against the index's digest before the journal records anything, the confirmation
  naming the chart version from and to and refusing when that changed while the user was asked
  ([architecture §8](../developer/architecture.md#8-helm-internalhelm);
  `TestHelmUpgradeTabOffersTheRepositorysVersions`, `TestHelmUpgradeFromARepository`,
  `TestHelmUpgradeFromARepositoryAgents`, `TestHelmUpgradeRepositoryRefusals`,
  `TestHelmUpgradeRepositoryCrashJournal`, `TestUpgradeToARepositoryChart`; GUI
  `TestHelmUpgradeTabRepositoryVersions`, PNGs `helm_upgrade_picker_{dark,light}.png`; on kind
  `TestKindHelmUpgradeFromRepository`: 0.1.0 installed, upgraded to 0.2.0 from the tab, an agent's
  upgrade back to 0.1.0, uninstalled).
- **Evidence (2026-10-13, night run):** Charts partial and Install partial: public https chart
  repositories the user adds (`helm.repo.add`, `remove`, `update`, `list`; Helm › Repositories…),
  kept in Cogpit's own folders (`<user config dir>/Cogpit/helm/repositories.json`, the indexes in
  `<user cache dir>/Cogpit/helm/`; the lead's default, the owner's question stays open), indexes
  downloaded on add, on update and while the Charts view or an Install tab is open past a
  10-minute TTL, bounded and parsed with the SDK; the charts as the local kind `helmcharts`
  (Helm › Charts); the "Helm Install: <chart>" dock tab with the chart's values.yaml, its versions
  and the digest check; `helm.chart.preview` (the SDK's server-side dry run) and
  `helm.chart.install` (mutating, crash-safe, values withheld from agents and scrubbed from errors)
  ([resource-parity](resource-parity.md#helm), [architecture §8](../developer/architecture.md#8-helm-internalhelm);
  `TestHelmReposAddListRemove`, `TestHelmIndexRefreshedOnlyWhileShown`, `TestHelmInstallFromTab`,
  `TestHelmInstallCrashJournal`, `TestHelmInstallTabInstalls`; on kind `TestKindHelmInstall`).
- **Evidence (2026-10-12, fourth run):** Upgrade partial: Lens's "Helm Upgrade: <name>" dock tab
  and `helm.release.upgrade` through the Helm SDK (the release's own chart, values replacing the
  user-supplied ones, ten revisions kept), mutating, bound to the revision its values were read
  from, crash-safe (an interrupted upgrade is offered as a rollback to the revision it started
  from), values withheld from agents, the audit log, errors and logs; Lens's release drawer (the
  values in an editor with Save, notes, resources by kind) and Rollback dialog, which is the
  rollback's confirmation in the GUI ([resource-parity](resource-parity.md#helm),
  [architecture §8](../developer/architecture.md#8-helm-internalhelm); `TestHelmUpgradeTab`,
  `TestHelmUpgradeCrashJournal`, `TestUpgradeThroughTheSDK`, `TestRollbackDialogConfirms`; on kind
  `TestKindHelmUpgrade`).
- **Evidence:** [resource-parity](resource-parity.md#helm), 2026-10-11 (third run): Releases partial,
  Rollback and Delete (uninstall) partial; Charts, Install and Upgrade missing. Built: the release
  Secrets of the `owner=helm` watch decoded into the virtual kind `helmreleases` (memoized per
  Secret UID and resourceVersion, bounded; hostile payloads fuzzed), the drawer's history, values
  (withheld from agents) and manifest (Secret data redacted), and `helm.release.rollback` and
  `helm.release.uninstall` through the Helm SDK in-process, destructive, bound to the revision the
  user confirmed, crash-safe through a journal offered on the next connect
  ([`internal/helm`](../../internal/helm/), [`commands/helm.go`](../../internal/engine/commands/helm.go),
  [`helmops.go`](../../internal/engine/helmops.go); `TestHelmCrashJournalOffer`; on kind
  `TestKindHelmReleases`: a local chart installed and upgraded with the SDK, then listed, read, rolled
  back and uninstalled through the engine, and a simulated crash offered and finished). The
  drawer's history and manifest gave way to Lens's drawer in the fourth run (above).
- **Captured (2026-10-12, Lens Desktop on a real cluster):** the release drawer (Chart with an
  Upgrade button, Updated, Namespace, Version, Status; editable values with "User-supplied values
  only" and Save; Notes; Resources by kind), its icons (Rollback, Upgrade, Delete), the Rollback
  dialog (every revision, newest first, the current one selected), the Delete dialog, the Helm
  Upgrade dock tab, and a rollback keeping 10 revisions ([resource-parity](resource-parity.md#helm)).
- **Estimate:** 4–12 h (was 5–14; the repositories' versions in the Upgrade tab built in the night run, 2026-10-13; left: authenticated and OCI repositories, captures of Lens's Charts view, Install tab and Upgrade tab with a repository, the other storage drivers).
- **Still to capture in Lens** (night run, 2026-10-13): the Charts view (its icon column, search hint and order), a chart's drawer, the Helm Install tab (its fields, the default release name and namespace, what Install does first), the Helm Charts preferences, and the Upgrade tab of a release whose chart a repository holds (what "/<chart>" and "(no repository)" become, the picker's order, which version it starts on, prereleases, several repositories).
- **Still to capture in Lens** (the staging cluster's release with five revisions, one failed): what the Rollback dialog's button does with the current revision selected, what Save does with the box cleared, where the checkbox sits relative to the editor, the texts of states other than Deployed and Failed, what the find widget's Replace does with a regular expression's groups.

### G12. Metrics and cluster overview
- **Required next (owner, 2026-10-12: "do not forget"):**
  - **metrics-server** (Kubernetes Metrics API, `metrics.k8s.io`): the pods' and nodes' current
    CPU and memory in the lists and the drawer, and as the source of the drawer's charts when no
    Prometheus is found (Lens: "Displaying metrics from Kubernetes Metrics Server").
  - **Prometheus** (through the API server's service proxy, detected per cluster): the drawer's
    resource graphs as Lens shows them, with a time range (1h by default), one graph for the whole
    pod and one per container, each in the variants **CPU, Memory, Network and Filesystem**; the
    containers' **requests and limits drawn as lines** in the graphs; **events as yellow markers**
    (vertical lines with a diamond, toggled by a flag button) at the time they happened.
  - **Interactive charts (owner, 2026-10-12):** hovering a chart shows the exact time and value
    under the pointer (each series' value, the request and limit, an event marker's event).
    **Built 2026-10-12:** a crosshair on the nearest data point, a dot per series and a panel with
    the exact time, the values, the request and limit and the event near a marker, on every chart
    and variant ([architecture §9](../developer/architecture.md#9-metrics-internalmetrics)).
  - Captured from Lens with the owner (screenshot, 2026-10-12): the Metrics panel's range dropdown,
    the four variant buttons and the event-marker toggle, the source line, the time axis, a legend.
- **Built (2026-10-12, first slice):** the Metrics Server end to end. Provider detection per cluster
  through discovery, with "not served", "forbidden" and "unavailable" as states
  ([`internal/metrics`](../../internal/metrics/)); polling only while a view or drawer that shows
  metrics is open (a list every 60 s, an open drawer's object every 15 s, shared by every consumer
  of a scope, cached for a 15 s TTL, no request when nothing is open: P12 in
  [testing](../developer/testing.md#performance-budgets-proposed-not-measured)); CPU and Memory
  columns in the Pods and Nodes lists; the drawer's Metrics section for a pod (per container too) and a
  node with a chart, requests and limits as lines and event markers, every chart (the pod's and each container's) with its own range, variant buttons and events flag; `metrics.get` for agents; the
  demo cluster's synthetic Metrics Server. Design and numbers: [architecture §9](../developer/architecture.md#9-metrics-internalmetrics).
- **Built (2026-10-12, second slice, backend only):** Prometheus through the API server's service proxy
  ([architecture §9](../developer/architecture.md#9-metrics-internalmetrics)). Detection among the
  cached Services by the charts' and Lens's well-known names (kube-prometheus-stack, `prometheus-k8s`,
  `prometheus-operated`, the `prometheus` chart's `prometheus-server`, `prometheus` in `monitoring`,
  `prometheus` or `lens-metrics`) with one buildinfo probe per candidate (at most 3), cached like the
  Metrics Server (10 minutes found, 2 not), forbidden/unavailable/not found as states, an override
  (namespace, service, port, prefix) on the detector; the chain Prometheus, then Metrics Server; range
  queries for the pod and each container in CPU, Memory, Network (per pod) and Filesystem (usage, else
  read and write rates), requests and limits from the pod spec as before, the source line
  `Prometheus (<namespace>/<service>)`, per-chart ranges (`Session.SetMetricsRange`, 1 minute to 30
  days), node charts from cAdvisor's root cgroup; budget P13, tested against a fake Prometheus behind
  the fake API server's service proxy. **Not verified against a real Prometheus.**
- **Built (2026-10-12, third slice, the window):** the Network and Filesystem buttons enabled from
  `ObjectMetrics.Capabilities` (icons, names as tooltips; disabled with "needs Prometheus" without
  them); Network drawn as Receive and Transmit in bytes per second, a container's chart showing the
  pod's with the line "Network is per pod" (Lens has no network tab on a container: Cogpit's, by the
  owner's "each in the variants"); Filesystem as Usage in bytes or Read and Write per second, as the
  cluster's Prometheus has it; requests and limits only on CPU and Memory; a second series colour
  token held to contrast and colour-difference tests in both themes; byte axes in round KiB, MiB, GiB
  steps, dates on the time axis from a day's range; the hover naming every new series with its unit;
  ranges up to 30 days with Prometheus (5m, 15m, 30m, 1h, 3h, 6h, 12h, 1d, 2d, 7d, 14d, 30d; up to 1h
  with the Metrics Server), **each chart's range held by the engine** (`ui.setMetricsRange` for the
  GUI and agents, `ui.state`'s `drawerCharts`), the source line "Displaying metrics from Prometheus
  (<namespace>/<service>)" without the Metrics Server's "keeps no history" note
  ([architecture §9](../developer/architecture.md#9-metrics-internalmetrics)). **Captured:** the default
  range "1h", the four variant icons, the source line's wording up to the provider. **Inferred:** every
  range value but 1h, the Network and Filesystem series' names and units, the per-pod line.
- **Checked against a real Prometheus (2026-10-12):** the `prometheus` chart on the kind cluster
  (`make kind-prometheus`, Prometheus 3.15.0, one-minute scrape): detection, a node's and a pod's
  CPU, memory, network and filesystem from range queries, the window's variants, ranges and hover,
  `ui.setMetricsRange`, `metrics.get` and budget P13, with memory equal to the Metrics Server's to
  the byte ([architecture §9](../developer/architecture.md#9-metrics-internalmetrics) has the numbers).
  It found three faults the fake did not show, fixed: the filesystem metric per kind, a node's
  usage over its disks only, rate windows that fit a one-minute scrape. **Only against the fake:**
  kube-prometheus-stack, containers' filesystem usage, a refused service proxy, a route prefix,
  history longer than an hour.
- **Required (owner, 2026-10-12, evening):**
  - **A status line per provider under the charts:** whether Cogpit reached Prometheus and the
    Metrics Server and whether they answered as expected (found, forbidden, unavailable, the last
    good answer's age, the last error), so the user sees why a chart is empty or stale.
  - **History from Prometheus, then live values from the Metrics Server:** when a drawer opens,
    Prometheus is asked once for the chart's history; the chart then continues with the Metrics
    Server's measurements, to spare Prometheus. The Metrics Server has CPU and Memory only, so
    Network and Filesystem still come from Prometheus: proposed (lead, not decided) that they are
    asked for only while their chart is shown. A range change asks Prometheus once for that range.
  - **Falling back (owner, 2026-10-12):** both together is the normal case; Cogpit falls back to one
    of the two only when the other fails, cannot be reached or is not found, also while a drawer is
    open: a Metrics Server that stops answering hands the live values to Prometheus, a Prometheus
    that is gone leaves the Metrics Server's live values without history. The status line says
    which provider serves what.
- **Built (2026-10-12, fourth slice: the evening's requirements):** the hub holds both providers
  apart (`metrics.NewHubFor`, [architecture §9](../developer/architecture.md#9-metrics-internalmetrics)).
  A drawer that opens asks Prometheus once for its charts' range (CPU and memory: two range queries;
  once more on a range change), then the live values come from the Metrics Server's get every 15 s;
  history and live values meet without a point twice, and a hole after both were away is filled
  from Prometheus once. The network and the filesystem are asked of Prometheus only while a chart
  shows them (**the lead's proposal, not the owner's decision**); a chart's variant is engine state
  for that (`ui.setMetricsVariant`, `ui.state`'s `drawerCharts`). Falling back as required, also
  while a drawer is open; a failed provider is asked again after **a minute (Cogpit's default)** and
  takes its part back when it answers. The lists and `metrics.get` ask the Metrics Server and
  Prometheus only when it fails or is missing; Prometheus is looked for only then or when a drawer
  opens. **The status line** under the charts: a row per provider (Prometheus with
  namespace/service:port, the Metrics Server), its state (answering, not answering, not allowed, not
  found, not looked for yet) as an icon in a status token with the whole state as its tooltip, what
  it serves (history, live values, network, filesystem), the last good answer's age and the last
  failure's class (`unavailable (HTTP 503)`: never a response body); the same in `metrics.get`'s
  `providers` and `ui.state`'s `drawerMetricsProviders`. The wording and icons are Cogpit's
  (**inferred**); Lens shows a source line only, which now names both providers. **A missing value
  is an empty cell, not 0%:** the Nodes list's CPU at 0% beside a memory value came from a
  Prometheus instant query with memory but no CPU series for the node; such a part is now
  `Usage.Missing`, an empty cell, and a point left out of the chart. Tests against fakes:
  `TestHistoryThenLiveValues`, `TestNetworkAndFilesystemOnlyWhileShown`,
  `TestMetricsServerFailsPrometheusTakesOver`, `TestPrometheusGoneLeavesTheLiveValues`,
  `TestGapIsFilledOnceFromPrometheus`, `TestListsUseTheMetricsServerAndFallBack`,
  `TestBothProvidersBudget`, `TestPrometheusInTheDrawer`, `TestProviderStatusLine`,
  `TestMetricsVariantIsEngineState`, `TestUsageCellsLeaveAMissingPartEmpty`. Budget P13 per open
  drawer at 1 h showing CPU or Memory: was a query set of five or six every 30 s (600–720 an hour),
  now two queries when it opens. **Measured on kind** (2026-10-12, the cluster's metrics-server and
  the `prometheus` chart): 90 s with the Pods list and a pod's drawer showing CPU cost 0 Prometheus
  requests (was 16), 6 Metrics Server gets and 2 lists; with the network shown 6 Prometheus requests.
  The fallback watched live (`TestKindMetricsFallback`, both deployments scaled to zero and back,
  then restored): the Metrics Server's absence seen at the next poll (15 s) with Prometheus's values
  in the same poll, its return served 9 s after it was ready; Prometheus's absence seen at its next
  request (`unavailable (HTTP 503)`) while the live values went on, its return served 28 s after it
  was ready ([architecture §9](../developer/architecture.md#9-metrics-internalmetrics)).
- **Waiting for the owner (2026-10-12):** charts of one drawer whose ranges lie far apart (a 7-day
  pod chart beside 1-hour container charts) share one query step, set by the longest range, so the
  short charts get about six points. Recommended: a query set of its own per range once the ranges
  differ by more than ten times (more requests only while ranges are mixed); the alternative, one
  range per drawer, gives up the per-chart ranges. Also open, no decision needed yet: the rate
  window of four minutes smooths charts of 5 to 15 minutes; revisit after kube-prometheus-stack (30 s
  scrape), possibly reading the scrape interval from Prometheus.
- **Built (2026-10-12): the cluster's Prometheus, set by the user:** `cluster.setPrometheus`
  (`namespace/service:port`, a path prefix, or `auto`) and `cluster.prometheus`, kept per context in
  Cogpit's own preferences (`metrics.json`, never the kubeconfig), applied at once (the open drawer
  asks the named service) and at the next connection before anything is detected; a dialog
  "Prometheus for <context>" from a button under the drawer's status line (Cogpit's placement:
  Lens's cluster settings page is not built). **Cogpit's default, not the owner's:** agents may set
  it (`DangerUI`; it writes no cluster object). Lens's "Metrics source", query format, empty
  container filter and HTTPS are not built ([ui parity](ui-parity.md), the Metrics provider row).
- **Checked against a real kube-prometheus-stack (2026-10-12):** chart 92.3.0 in its own namespace
  on kind (`make kind-kps`, `TestKindKubePrometheusStack`, then `make kind-kps-down`): detected as
  `cogpit-kps/kps-kube-prometheus-stack-prometheus:http-web` (a short release keeps "-stack" in the
  name: the name pattern missed it, fixed), a pod's history at the 30 s scrape with network and
  filesystem. **A node has no history there:** the chart's kubelet ServiceMonitor drops cAdvisor's
  series without a pod, the root cgroup the node charts read; the node's chart shows the Metrics
  Server's live values only. Node history on kube-prometheus-stack needs node-exporter's series
  (a decision: which series, and how a node-exporter target is tied to its node).
- **Left of Prometheus:** Lens's cluster settings page (the setting lives in a dialog under the
  drawer's status line until then), node history on kube-prometheus-stack (above), the namespace
  and service as links (Lens:
  "Prometheus: <namespace>/<service>:<port>"; the port shows on the status line since the fourth
  slice), the Lens-provider variants that are not
  Prometheus-compatible by name (Thanos, VictoriaMetrics under other service names: reachable through
  the override only). A provider nothing asked since its last answer (Prometheus while only CPU and
  Memory are shown) is shown as of that answer, with its age: no probe keeps the status line current.
- **Built (2026-10-13, night run, M7): the cluster Overview** (`cluster/overview`, the navigator's first
  item; [ui parity](ui-parity.md#cluster-overview-and-charts)). The CPU, Memory and Pods charts of the
  master or the worker nodes over the drawer's ranges, with requests, limits, allocatable capacity and
  capacity as lines; CPU and Memory through a cluster scope of the hub (`metrics.ClusterScope`: the
  same loop, roles and fallback as a drawer, Prometheus's history once as sums over the nodes' root
  cgroups, then the Metrics Server's node list summed; [architecture §9](../developer/architecture.md#9-metrics-internalmetrics)),
  the lines and the Pods chart from the cache (**Pods have no history from before the page opened:
  Cogpit's default**); the status line per provider under the charts; the Issues from the cache:
  pods and nodes with their views' warning, workloads short of ready pods, warning events of the last
  hour (**the rule is Cogpit's: Lens documents "system messages and warnings" only**); `cluster.overview`,
  `ui.setOverviewRange`, `ui.setOverviewNodes`; budget P14 ([testing](../developer/testing.md#performance-budgets-proposed-not-measured)).
  Inferred: the page's layout, colours, wording ("No issues found."), the default group (the workers),
  the newest-first order of the Issues. Differs: ranges stop at 30 days (Lens: 2 months). **Checked on
  kind** (2026-10-13, `TestKindClusterOverview`, the cluster's metrics-server and the `prometheus` chart):
  the master's hour from Prometheus (120 points at 30 s), the Metrics Server's sums after it (0.204 cores
  against metrics.get's 206m), 5 Prometheus requests when the page opened and none after, 3 issues
  (a crash-looping pod, its Deployment, its BackOff event).
- **Missing:** the Overview's Pods history from Prometheus, its ranges beyond 30 days, the explicit Lens
  Metrics install, the settings rows (metrics source, hide metrics), the Disk column, Lens's usage bars in
  the Nodes list, CPU and Memory in a node drawer's Pods table, history that outlives a closed drawer.
- **Evidence:** [ui-parity](ui-parity.md#cluster-overview-and-charts), its five rows partial (the four
  Overview rows since 2026-10-13, the drawer charts), plus the metrics rows of cluster settings.
- **Estimate:** 40–80 h before the first slice; after the Overview (2026-10-13) the remainder (the
  settings rows, the Lens Metrics install, the Nodes list's bars and Disk, node history on
  kube-prometheus-stack, the Overview's Pods history) is about 15–35 h (the lead's estimate to apply
  to the table above).

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
- **Decided (owner, 2026-10-12):** the Solarized colours ([D-22](../developer/decisions.md#d-22-colour-theme)), the
  status bar link ([D-23](../developer/decisions.md#d-23-a-link-to-the-owner-in-the-status-bar)), the C helm logo
  ([D-24](../developer/decisions.md#d-24-logo)) and copying values from the drawer
  ([D-25](../developer/decisions.md#d-25-copying-values)); built 2026-10-12 (fifth run).
- **Required (owner, 2026-10-12, seen in the demo cluster):** the detail drawer's width never
  follows its content: a long value (a container's `Waiting: CrashLoopBackOff (back-off …)` status,
  which the demo changes every few seconds) widened the drawer and made the whole layout jump. The
  user sets the width by dragging the drawer's edge, and it stays; long values wrap or are cut.
  The user can also set the width of the primary sidebar (the navigator) by dragging its edge.
  Both widths persist. **Built 2026-10-12** ([`panelwidths.go`](../../internal/ui/fyneui/panelwidths.go),
  [architecture §4.3](../developer/architecture.md#43-ui-state)): the drawer and the sidebar are a fixed
  number of points wide, set by dragging their edge (drawer 320 to 1000, default 600; sidebar 160 to 480,
  default 216; the table keeps 480 beside the drawer), stored in Fyne's preferences when the drag ends
  and restored at the next start; GUI-only, not engine state. The cause was the `container.Split`
  clamping its offset by the drawer's content-driven minimum width. Long values now wrap (statuses,
  properties, chips) or are cut (title bar, event header, links, lines), the full text stays in the copy
  icon and the selection (`TestDrawerWidthNeverFollowsItsContent`, `TestDrawerWidthByDraggingItsEdge`,
  `TestSidebarWidthByDraggingItsEdge`, `TestPanelWidthsSurviveARestart`, `TestDrawerRebuildsToItsWidth`;
  render `TestRenderDrawerLongStatus`). Not verified: dragging with a real pointer in a window (the tests
  drive the handle's `Dragged` events), Fyne's preference file on disk across a real restart.
- **Navigator (built 2026-10-13, night run):** the captured header (a scope dropdown and a search
  button), the cluster as the tree's root (its initials badge, name, kubeconfig, a connection dot, ›
  and ‹), "<item> - <context>" on every item, and the keyboard of Lens's documentation (↑↓, → and ←,
  Tab and ⇧Tab, ↵ through `ui.openView`), the tree now Cogpit's own rows instead of Fyne's
  ([`navigator.go`](../../internal/ui/fyneui/navigator.go), `TestNavigatorKeyboard`,
  `TestNavigatorSearch`, [ui-parity › Layout](ui-parity.md#layout), [› Clusters](ui-parity.md#clusters)).
  Differs from Lens: no section level, so the dropdown's "Local Kubeconfigs…" and the ‹ open the
  context picker and the › does nothing; inferred, Cogpit's defaults:
  the search's matching rules, ↵ on a group, Space, Home and End, a letter typed in the tree starting
  a search. Not tried in a real window. Estimate not moved (the lead recounts). Since the tab bar
  (below) ↵ and a click open the preview tab, ⇧↵ and a double click a fixed tab.
- **Tab bar (built 2026-10-13, night run):** the tabs of cluster views above the workspace as
  captured: "<view> - <context>" with the cluster's badge, the preview tab italic, a close button,
  the full title as tooltip; the navigator's click opens the preview tab, its double click and ⇧↵
  a fixed tab, a double click on the preview tab fixes it; ⌘W, ⌘⌥→ and ⌘⌥←, ⌘1–⌘9, ⌃- and ⌃⇧- as
  View menu items from any focus. The tabs are engine state (`UIState.Tabs`, eight `ui.tab.*`
  commands and MCP tools; `ui.openView` opens in the preview tab), each with its own namespace
  filter, search, sort, selection, drawer and history, kept in step by one function that
  `UpdateUI` runs ([`engine/tabs.go`](../../internal/engine/tabs.go),
  [`fyneui/tabbar.go`](../../internal/ui/fyneui/tabbar.go), `TestTabsOfAnotherCluster`,
  `TestTabBarOverTheEngine`, [PNG](../../internal/ui/fyneui/testdata/tabbar_dark.png),
  [ui-parity › Layout](ui-parity.md#layout)). Differs from Lens: initials badges instead of the
  cluster's icon (cluster identity missing); one connection at a time, so showing a tab of another
  cluster connects to it; Windows and Linux get the Ctrl forms of the macOS keys, not Lens's
  Ctrl+PageUp/PageDown and Alt+←/→ (G22). Inferred, Cogpit's defaults: one preview tab in all, the
  neighbour shown on close, the history's contents and bound, ⌘9 the last tab, wrapping next and
  previous. Still missing: the title bar's history arrows, "Tab icons take the cluster's colour"
  (needs cluster identity), filtering the tabs to a section's scope. Not tried in a real window.
  Estimate not moved (the lead recounts).
- **Welcome page (built 2026-10-13, night run):** the workspace shows Cogpit's welcome page while
  it has nothing else to show (no cluster connected, or no tab active) and from Help › Welcome:
  the logo, a line about Cogpit, the clusters to connect with one click (the hotbar's bank, this
  session's, on a first start the kubeconfigs'), All Clusters…, Add Kubeconfigs… and the README on
  GitHub. Engine state: `UIState.Welcome`, derived after every change, and `ui.welcome.open`, which
  leaves the active tab keeping its state ([`fyneui/welcome.go`](../../internal/ui/fyneui/welcome.go),
  [`engine/welcome.go`](../../internal/engine/welcome.go), `TestWelcomePageOverTheEngine`,
  [PNG](../../internal/ui/fyneui/testdata/welcome_dark.png), [ui-parity › Layout](ui-parity.md#layout)).
  Lens's page is not captured: its contents, wording and layout are Cogpit's (inferred, Cogpit's
  defaults); the context picker still opens by itself at start, over the page. Not tried in a real
  window. Estimate not moved (the lead recounts).
- **Missing:** the Fyne window layout, the cluster sidebar, the table, the detail drawer and the
  dock frame; Cogpit's Fyne theme. Needs the [spike](../developer/spike-fyne.md) to pass.
- **Evidence:** [ui-parity](ui-parity.md), 18 of 40 rows partial across Layout, Cluster sidebar,
  Tables, Detail drawer and Dock ([`internal/ui/fyneui`](../../internal/ui/fyneui/), Cogpit's theme
  with status tokens). The spike's gate results are recorded for the owner's decision
  ([spike](../developer/spike-fyne.md#results)). Since 2026-10-11 the captured differences in the
  existing views are closed or narrowed: row menus (⋮ and right-click) built from the registry
  with container submenus and schema-generated dialogs, the drawer's toolbar as icons with
  tooltips, the warning column (Pods, Nodes; rules inferred), links in cells, tooltips on cut-off
  cells, the drawer's Properties section with folded lists, links and the events at the end, a
  cordoned node's Conditions; cell tooltips and right-click are done. Still missing: title bar
  with history, welcome page, keyboard navigation, CSV export, column reorder and hiding,
  row checkboxes and multi-selection in row menus, the drawer's Copy and port Forward…, the Pods
  view's container squares, Cogpit's own title bar on macOS (owner, 2026-10-11: no standard title
  bar, the logo beside the window buttons; [ui-parity](ui-parity.md#layout)).
- **Evidence (2026-10-11, third run, tables and window):** cluster-scoped views hide the namespace
  filter; show and hide columns from the header's ⋮ (`ui.setColumns`, per view in the UI state);
  the Definitions view's group filter (`ui.setGroups`); keyboard navigation (done); a double click
  on a column separator fits the column; "18 items" with a CSV download (`table.exportCsv`, written
  in the engine); the Workloads Overview as a page (`workloads/overview`, `workloads.overview`);
  Cogpit's own title bar on macOS, built but not yet seen in a real window (smoke launch). Still
  missing from this list: history in the title bar (the tab bar and its history built 2026-10-13), column reorder (the welcome page built 2026-10-13), row
  checkboxes and multi-selection, the drawer's Copy and Forward…, container squares.
- **Estimate:** 35–110 h (was 45–130; title bar, keyboard, columns, CSV and the Workloads Overview built, third run).

### G13. Clusters, hotbar, palette, menus, shortcuts and preferences
- **Required next (owner, 2026-10-12: "still missing, do not forget"):** the hotbar (Lens's
  vertical bar at the window's left edge with the pinned clusters' icons). **Built 2026-10-12**
  (night run, [`engine/hotbar.go`](../../internal/engine/hotbar.go),
  [`fyneui/hotbar.go`](../../internal/ui/fyneui/hotbar.go)): banks of up to 12 pinned clusters as engine
  state (`UIState.Hotbar`, nine `hotbar.*` commands and MCP tools that write only
  `<user config dir>/Cogpit/hotbar.json`), the bar left of the navigator with initials tiles, a
  connection dot, the context as tooltip and the "‹ 1 ›" switcher, a click or F1–F12 to connect,
  ⇧F1–⇧F12 to switch banks, Add to Hotbar in the picker's row menu and the Cluster menu
  (`TestHotbarCommands`, `TestHotbarFunctionKeys`, [PNG](../../internal/ui/fyneui/testdata/hotbar_bank_dark.png)).
  Differs from Lens: Move Up and
  Down instead of dragging, the F-keys do nothing while a text field or a
  terminal has the focus, not tried in a real window; inferred: missing contexts kept as marked
  tiles, at most 12 banks, the arrows wrap, the confirmation before removing a bank with clusters
  ([ui-parity › Hotbar](ui-parity.md#hotbar)).
- **Built 2026-10-13 (night run):** the command palette, ⌘⇧P
  ([`engine/palette.go`](../../internal/engine/palette.go), [`fyneui/palette.go`](../../internal/ui/fyneui/palette.go)):
  an overlay with a filter over what the menus offer in the current state, listed from the
  registry's metadata (the drawer object's actions, menu commands only, no tool meant for agents,
  Switch to *view* per navigator view, the hotbar's clusters and banks, commands with required
  parameters only where the window has a dialog for them), run as their menu items run them
  (confirmation, dialogs, audit); its open state is UI state (`ui.palette.open`, `ui.palette.close`).
  Shortcuts through the registry: ⌘⇧P and ⌘P (the picker) on main-menu items, so they work from
  any focus; ⌘F and ⌘⇧T taken from the registry; ⌘Q left to the platform (GLFW's app menu), ⌘R
  not bound (nothing to reload in a watch-fed view). Inferred: the list's order and contents beyond
  the captured rows, the greyed rows, the ranking; not tried in a real window
  (`TestCommandPaletteTypeAndRun`, `TestPaletteListsWhatTheMenusOffer`,
  [PNG](../../internal/ui/fyneui/testdata/command_palette_dark.png)).
- **Built 2026-10-13 (night run):** Preferences (⌘,) and a cluster's Settings (⌘⇧,), as engine
  state and commands ([`engine/preferences.go`](../../internal/engine/preferences.go),
  [`engine/clustersettings.go`](../../internal/engine/clustersettings.go): `UIState.Preferences`,
  `UIState.ClusterSettings`, the open page in `UIState.Settings`; `preferences.*`,
  `cluster.settings.*`, `settings.close`, each an MCP tool) kept in Cogpit's own
  `preferences.json`, `cluster-settings.json` and `cluster-icons/`, never in a kubeconfig. The
  pages ([`fyneui/preferences_page.go`](../../internal/ui/fyneui/preferences_page.go)) take the
  views' place until closed (inferred). Preferences: the theme (switched at once;
  `--theme` overrides it for a run), the time zone (the drawer's Created, the charts), the
  tables' row height, the editor's font size, line numbers and tab size, the terminal's font
  size, shell and arguments (the shell GUI-only). Cluster settings: the display name and the
  identity (the theme's accent colours or an uploaded image, checked and kept as Cogpit's copy)
  that the hotbar, the picker and the navigator show; the kubeconfig's path with its folder in
  the file manager; the terminals' working directory and namespace; the Prometheus dialog linked.
  Settings… in the picker's, the hotbar tile's and the navigator root's menus; Preferences… moves
  into macOS's application menu (`TestPreferencesOpenWithTheShortcutAndSwitchTheTheme`,
  `TestClusterSettingsFromThePickerSetIdentityAndTerminal`, `TestClusterSettingsCommandsNeverTouchTheKubeconfig`,
  [PNG](../../internal/ui/fyneui/testdata/cluster_settings_dark.png)). Inferred or Cogpit's
  defaults: the page in place of the views, the sections' grouping, the swatches, the bounds, the
  cluster namespace replacing the filter's, Dark as the default theme; not tried in a real window.
- **Built 2026-10-13 (night run):** the Help and Window menus and a cluster's Disconnect and
  Remove from Cogpit ([`fyneui/menus.go`](../../internal/ui/fyneui/menus.go),
  [`fyneui/menus_cluster.go`](../../internal/ui/fyneui/menus_cluster.go)). Help: Welcome
  (`ui.welcome.open`), Documentation (the README on GitHub), Copy MCP command, About Cogpit (the
  build's version); "Open Onboarding Wizard..." not a target (the lead's default). Window: on macOS
  GLFW's own menu stays (Minimize ⌘M, Zoom, Bring All to Front, Enter Full Screen ⌃⌘F, the windows;
  verified in GLFW 3.4's and Fyne 2.8.1's source, Cogpit adds none); on Windows and Linux Close
  Window and Toggle Full Screen, Minimize missing (Fyne has no call for it). Disconnect
  (`cluster.disconnect`) closes the connection without a request to the cluster; Remove from Cogpit
  (`cluster.remove`, GUI only) removes Cogpit's link to a kubeconfig file the user added after a
  confirmation naming the path and the contexts that leave, never the file; a context of a synced
  folder (the default `~/.kube` too) or of a `$KUBECONFIG` file is greyed with the reason
  (`cluster.links`). Both in the Cluster menu and the clusters' menus (picker, hotbar tile, navigator
  root, welcome page). Zoom stays missing: Fyne has no runtime scale, ⌘- is the tab history's Back on
  Windows and Linux, and the level would need a Preferences field. Inferred or Cogpit's defaults:
  the labels, which contexts can be removed, the Window menu on Windows and Linux; not tried in a
  real window (`TestRemoveFromCogpitAsksFirst`, `TestHelpAndWindowMenus`, `TestClusterRemove`,
  `TestDisconnect`; [ui-parity › Application menu](ui-parity.md#application-menu)). Estimate not
  moved (the lead recounts).
- **Missing:** kubeconfig syncs (alpha needs these; pasting is not a target, D-21), dragging on the
  hotbar, the shortcuts of features that do not exist yet (Ask AI; the tabs' five built 2026-10-13), the
  Preferences that are not built (proxy, untrusted certificate authorities, kubectl download,
  Helm repositories and `HELM_DRIVER`, update channel, start at login, the editor's minimap and
  font family, the terminal's copy and paste behaviour, theme and font family), the cluster
  settings' Proxy, Namespaces, Node Shell and Lens Metrics pages, and the rest of appearance and
  application integration (zoom, Minimize on Windows and Linux, several windows, deep links).
- **Evidence:** [ui-parity](ui-parity.md), 65 rows across those sections (6 more are marked not a
  target); 7 of them partial since 2026-10-12 (the hotbar's six and the Cluster menu), the
  navigator keyboard since 2026-10-13 ([G5](#g5-app-shell-navigation-tables-and-drawer)'s navigator; ⇧↵ opens a fixed tab since the tab bar),
  and 7 more since 2026-10-13 (the palette's two free rows and five shortcut rows; the paid bank
  commands' row updated); 11 more since 2026-10-13 (Preferences and Cluster settings: the two
  shortcut rows, the application menu's Preferences, Cluster identity, Theme, Locale timezone,
  Editor, Tables, Terminal, Cluster settings › General and Terminal; the Cluster menu, Kubeconfig
  syncs, Metrics provider and Appearance's theme row updated); and 5 more since 2026-10-13 (the tab
  bar's shortcut rows: close the tab, back and forward, next, previous, tab *n*; G5's tab bar); and
  3 more since 2026-10-13 (the Window menu, Help › Welcome, zoom and full screen; the Cluster menu
  updated; G5's welcome page is the fourth).
- **Estimate:** 50–140 h (was 58–150; the palette and five shortcut rows built, night run; 1–2.5 h per row, the registry makes shortcuts and menus cheap); not moved for Preferences and Cluster settings (2026-10-13): the lead recounts.

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
- **Decided (owner, 2026-10-12, [D-16](../developer/decisions.md#d-16-the-first-real-release)):**
  releases stay dry runs until the alpha; the job that builds the unsigned app as a download
  belongs to the alpha's work and is switched on with the first real release.
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
- **Evidence (2026-10-11, the gate's offscreen test):** `TestPerfGateRunsEveryPhase` hung once
  until go test's 10-minute timeout on the macOS CI runner (7 s locally). Cause, reproduced with
  a main queue slower than one callback per frame (the race detector and coverage on a loaded
  runner): the jank meter kept posting a callback every 16 ms after it had reported, so the test's
  drain never emptied the queue and never reached its deadline; and a sample still queued at the
  report ended the measurement a second time, closing the YAML window twice, which panics inside
  Fyne's test driver while it holds its window lock, so the cleanup's next driver call blocked for
  good. Fixed in the meter ([`perf.go`](../../internal/ui/fyneui/perf.go)): it stops its ticker
  when it reports, samples after the report do nothing, it never waits on the main goroutine for
  the YAML document, and a view switch waits (at most 10 s) for the view's rows instead of
  switching away from a slow view; the test drains in batches and checks its deadline between
  them. `TestPerfGateFinishesOnASlowMainGoroutine` runs the gate with every callback slower than
  a frame (it hangs on the old meter, passes on the new).
- **Impact:** large clusters are where users notice an IDE first, and where an IDE costs the API
  server most.
- **Estimate:** 28–66 h (was 30–70; the perf gate no longer hangs on a slow runner, third run).

### G16. Stability and the user bug backlog
- **Found in the owner's smoke test (2026-10-12):**
  - **Every quit on macOS crashed** (window closed, Cmd+Q or SIGTERM): the shell stopped Cogpit's
    title bar after the app's run ended by asking GLFW for the window, which GLFW had terminated
    ("NotInitialized"). Fixed (f480186, `TestTitlebarStopAfterTheRunEnded`, which fails on the old
    code); checked on a real launch ended by SIGTERM and by the owner closing the window: exit 0, no
    crash report. Crash reports are kept since 7ac22e0
    ([D-11](../developer/decisions.md#d-11-crash-stance)).
- **Missing:** PhotoCraft's first public week produced hundreds of issues on features its tests
  counted as done; this entry reserves the time for the same after Cogpit's first release.
- **Estimate:** 100–200 h to beta.

## Beyond Lens

### G17. MCP control
- **Missing:** the MCP server over the registry with the UI tools of
  [beyond-lens](beyond-lens.md#mcp-control-of-ui-and-cluster), Streamable HTTP with a bearer token,
  the stdio bridge, the GUI side of the confirmation flow, the audit log view
  ([D-2](../developer/decisions.md#d-2-agent-interface-scope)). `ui_state`, `resources_find` and
  `ui_navigate` come early (M2), because they also let agents test the UI.
- **Evidence:** beyond-lens, 10 done and 6 partial of the 17 MCP rows: tools from the registry,
  Streamable HTTP on loopback with a bearer token and `Host`/`Origin` checks, `ui_state`,
  `resources_find`, `ui_navigate` and the other UI tools, the GUI confirmation for agents, the audit
  log file ([`internal/mcp`](../../internal/mcp/), `TestShowcaseServiceByClusterIP`); since
  2026-10-13 the stdio bridge `cogpit-cli mcp`, so a registration survives restarts (it forwards to
  the running app and follows its restarts, else serves a headless engine that refuses what needs a
  confirmation; [`bridge.go`](../../internal/mcp/bridge.go), `TestBridgeFollowsARestart`,
  `TestMCPHeadlessServesTheRegistry`). Still missing: `view_rows`, reading dock tabs, the audit log
  in the GUI, the per-context allowance in the GUI.
- **Estimate:** 12–30 h (was 30–60).

### G21. Built-in agent
- **Required (owner, 2026-10-12, later, after the alpha):** a VS Code-like chat in Cogpit (as in
  Claude Code for VS Code), switchable on and off; an OpenAI-compatible endpoint (Open WebUI,
  LiteLLM) so that local models work; the agent gets Cogpit's tools to operate the UI; it keeps
  working while the external MCP server is switched off (in-process dispatch, D-10).
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
| 2026-10-13 | minor | G5, G13: the welcome page (`UIState.Welcome`, `ui.welcome.open`), the Help and Window menus (macOS keeps GLFW's Window menu, verified in source), Disconnect and Remove from Cogpit (`cluster.disconnect`, `cluster.links`, `cluster.remove`) built (night run); four free rows partial; zoom missing with its reasons; estimates not moved (the lead recounts) |
| 2026-10-13 | minor | G8 evidence (night run): jumping to a time partial, every Logs row exists; estimate not moved |
| 2026-10-13 | minor | G8 evidence (night run): loading older lines partial; estimate not moved |
| 2026-10-13 | minor | G8 evidence (night run): the log tab's pod and container selectors partial; estimate not moved |
| 2026-10-13 | minor | G11: a release upgraded to a chart version from the user's repositories (the Upgrade tab's choice, `helm.release.upgrade` with `version` and `repository`, the chart downloaded and checked first, the confirmation's from and to bound to what runs; night run) |
| 2026-10-13 | minor | G5, G13: the tab bar of cluster views built (night run): engine state with eight `ui.tab.*` commands, the bar, the navigator's preview and fixed tabs, five shortcut rows; what differs from Lens and what is inferred; estimates not moved (the lead recounts) |
| 2026-10-13 | minor | G13: Preferences and Cluster settings built (night run): the two pages and their shortcuts, cluster identity, the theme, time zone, editor, tables and terminal preferences, the cluster's terminal settings; G8: the terminal's font size, shell and arguments; what is inferred and Cogpit's defaults named; estimates not moved (the lead recounts) |
| 2026-10-13 | minor | G12: the cluster Overview built (night run, M7): charts of the master or worker nodes through a cluster scope of the hub, lines and Pods from the cache, the Issues from the cache, three commands, budget P14; the remainder re-estimated 15–35 h (the summary table not recounted) |
| 2026-10-13 | minor | G6: Applications partial (a virtual kind over the cached workloads, Logs per workload); 3 views missing, estimate 12–60 h |
| 2026-10-13 | minor | G6: Vertical Pod Autoscalers partial (a curated profile of the CRD kind); 4 views missing, estimate 16–70 h |
| 2026-10-13 | minor | G6: Resource Claims, Resource Claim Templates, Device Classes and Resource Slices partial (Lens's documented columns), the pod drawer's Resource Claims section; 5 views missing, estimate 20–80 h |
| 2026-10-13 | minor | G11: chart repositories in Cogpit's own folders, the Charts view, the Helm Install tab and the install as a crash-safe job (night run); what is left and what to capture |
| 2026-10-13 | minor | G17: the `cogpit-cli mcp` stdio bridge built, evidence added; estimate unchanged |
| 2026-10-13 | minor | G2: `cogpit-cli contexts` and `cogpit-cli get` built (milestone 0.1), evidence added; estimate unchanged |
| 2026-10-13 | minor | G5, G13: the navigator built (night run): header, the cluster as the tree's root, item tooltips, keyboard; what differs from Lens and what is inferred; estimates not moved (the lead recounts) |
| 2026-10-12 | minor | G12: the evening's requirements built (fourth slice): both providers together (Prometheus's history once, the Metrics Server's live values, the network and the filesystem only while shown: the lead's proposal), the fallback, a status line per provider (also in `metrics.get` and `ui.state`), a chart's variant as engine state, a missing value as an empty cell; budget P13 measured on kind; no count moved |
| 2026-10-12 | minor | G7: seven create dialogs partial (Namespace, Service Account, the four RBAC kinds, Resource Quota); 3 of 39 actions missing; estimate 12–45 h (was 20–70); totals not recounted |
| 2026-10-12 | minor | G13: the hotbar built (night run), what differs from Lens and what is inferred; estimate 58–150 h (the totals not recounted) |
| 2026-10-12 | minor | G12: the owner's evening requirements (a status line per provider; history from Prometheus, then live values from the Metrics Server; one of the two alone only when the other fails or is missing) and the step of mixed ranges waiting for the owner; G16: the quit crash fixed |
| 2026-10-12 | minor | G12: checked against a real Prometheus on kind (the `prometheus` chart); three faults it found fixed; only kube-prometheus-stack and the override left unchecked; estimate not moved |
| 2026-10-12 | minor | G12: the window draws Prometheus (Network, Filesystem, ranges to 30 days held by the engine through `ui.setMetricsRange`); range values beyond 1h inferred; estimate not moved |
| 2026-10-12 | minor | G12: Prometheus through the service proxy, backend (detection, range queries for CPU, Memory, Network and Filesystem, per-chart ranges, budget P13); the window's Network and Filesystem buttons left; estimate not moved |
| 2026-10-12 | minor | G12: the owner's hover requirement built (crosshair, dots and a panel with time, values, request and limit, events); estimate not moved |
| 2026-10-12 | minor | G12: every drawer chart has its own controls (owner's smoke launch: "no main button"); estimate not moved |
| 2026-10-12 | minor | G2 (scope C2): the Kubeconfig Syncs captures followed: dot files, a closed connection, one dialog, `~/.kube` by default, the connect log, the Remove confirmation; estimate not moved (the lead recounts) |
| 2026-10-12 | major | Totals after the fourth run: free parity ~420–1,040 h, full target ~620–1,450 h |
| 2026-10-12 | minor | G8: the terminal widget fuzzed alone, its crashes, hangs and races patched, the upstream series prepared (D-14, fourth run); estimate of that part done |
| 2026-10-12 | minor | Fourth run, the third run's leftovers: G7 evidence (the journals shared by Cogpit processes); G4 evidence (Refresh Access Reviews in the menu, the View menu hides what the sidebar hides); G3 evidence (live related-objects indexes, measured); G8 evidence (workload log streams resumed and pruned); G7 evidence (Restart and Scale bound to the UID in their patches); estimates not moved |
| 2026-10-12 | minor | G2 (fourth run, agent D): kubeconfig files, sync folders and pasted kubeconfigs, the approval of their credential plugins; estimate 10–28 h |
| 2026-10-12 | minor | G11 (C1): the release drawer, Rollback dialog and Upgrade tab follow the second capture; estimate 10–26 h (the totals not recounted) |
| 2026-10-12 | minor | G11 (fourth run): the Helm upgrade and its dock tab, Lens's release drawer and Rollback dialog built; estimate 12–30 h (the totals not recounted) |
| 2026-10-12 | minor | G3 evidence: covering scopes (fourth run); G6 evidence: Secrets on a full watch, Keys and Type filled, the drawer from the cache (D-13 revised, built) |
| 2026-10-12 | minor | G1: `make parity` built (D-15); its first run's findings; estimate 3–8 h |
| 2026-10-12 | minor | Owner's decisions: G1 the parity checker first (D-15); G6 Secrets as a full watch (D-13 revised); G8 the terminal widget kept, upstreamed and fuzzed (D-14); G11 Lens's Helm release drawer, dialogs and upgrade tab captured; G14 the download job belongs to the alpha (D-16) |
| 2026-10-11 | minor | G5: evidence of the third run's tables and window work; G15: why the gate's offscreen test hung in CI, and the fix |
| 2026-10-11 | minor | G6 and G11 evidence: Secrets and Helm Releases partial, Helm rollback and uninstall partial (third run) |
| 2026-10-11 | minor | G10: port forwarding built (manager, local kind, commands, dialog, view); what is left: Lens's status texts and Edit dialog, access review of the forward's pod, a real-window browser check |
| 2026-10-11 | minor | Access reviews built: G2 and G4 evidence, G3 and G9 no longer list them as missing; estimates not moved |
| 2026-10-11 | minor | G6 evidence (third run, agent D): Endpoint Slices, Endpoints, Network Policies, Persistent Volumes, Storage Classes and Replication Controllers views partial; the related-objects index's first users (a claim's pods, a node's Pods table, a Service's endpoint slices) |
| 2026-10-11 | minor | G6 Evidence: the Access Control views (five of them partial) |
| 2026-10-11 | minor | G6 Evidence: the 13 Config views (third implementation run), admission policies by served version |
| 2026-10-11 | major | Third implementation run: G2–G11 and G15 re-estimated from what landed; totals ~430–1,060 h free parity, ~640–1,480 h full target |
| 2026-10-11 | minor | G11 unblocked by the H-8 decision; G5: Cogpit's own title bar on macOS (owner) |
| 2026-10-11 | major | Second implementation run: G1 (CI green), G3, G5, G6 (27 of 64 views), G7 (14 of 39 actions, drain tested on three nodes) and G9 re-estimated; totals ~570–1,400 h free parity, ~780–1,820 h full target |
| 2026-10-11 | minor | G5: row menus, drawer toolbar, warning column, links, tooltips and the drawer's Properties built; what remains |
| 2026-10-11 | minor | G9: editing, create from templates and Delete for every kind built; what is left listed |
| 2026-10-11 | minor | G7: node and workload actions partial (cordon, uncordon, crash-safe drain, scale, trigger, suspend, resume, evict) |
| 2026-10-11 | minor | G1: CI and the semantic-release dry run exist as workflow files, not yet observed green; generators, the envtest job and a passing `make vuln` remain; estimate not moved |
| 2026-10-11 | major | The MVP: G1–G9, G14, G15 and G17 re-estimated with evidence; new terminal-widget risk in G8 |
| 2026-10-11 | minor | D-5: G19 (external extensions) is not a target and leaves the totals |
| 2026-10-11 | minor | D-9: Flux CD and Argo CD views split from G18 into G23, in the beta |
| 2026-10-11 | minor | macOS first (D-7): G14 covers macOS, G22 Windows and Linux after the macOS beta |
| 2026-10-11 | major | Re-estimated for Go (G2, G7, G11, G18 smaller; G3 larger for the cache; G9 own editor); G17 is MCP control of UI and cluster; G21 built-in agent added; G14 includes one-click updates; Freelens rows removed |
| 2026-10-11 | major | First version: G1–G20 ranked, before any code |
