# Where Cogpit falls short of Lens

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-12 · **Change:** major (fourth run: totals recounted from the table; G1, G2, G11 re-estimated) · **Target:** Lens Desktop 2026.10.51501

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
| G7 | Resource actions | features | blocks alpha (core actions) | 20–70 h | [resource-parity](resource-parity.md) |
| G8 | Logs, terminal and shells | features | blocks alpha | 16–40 h | [ui-parity](ui-parity.md#logs) |
| G9 | YAML editor and create resource | features | blocks alpha | 8–22 h | [ui-parity](ui-parity.md#editor-and-create-resource) |
| G10 | Port forwarding | features | blocks alpha | 2–6 h | [ui-parity](ui-parity.md#port-forwarding) |
| G11 | Helm | features | blocks alpha | 12–30 h | [resource-parity](resource-parity.md#helm) |
| G12 | Metrics and cluster overview | features | blocks beta | 40–80 h | [ui-parity](ui-parity.md#cluster-overview-and-charts) |
| G13 | Clusters, hotbar, palette, menus, shortcuts and preferences | UI | partly alpha, rest beta | 65–165 h | [ui-parity](ui-parity.md#clusters) |
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
- **To follow Lens's captures of 2026-10-12** ([ui parity › Clusters](ui-parity.md#clusters), 2–4 h):
  dot files in sync folders read; a connected cluster closed when its file goes; the "Add
  Kubeconfigs" entry as one file dialog for files and folders; `~/.kube` as a default synced
  folder; a connect's log ("Connection Updates") behind a warning icon. Owner requirement: removing
  a synced item asks, naming the path, that only the link is removed.
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
  without setup.
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
- **Missing:** 9 of the 64 resource views ([resource-parity](resource-parity.md), 2026-10-11 after
  the third run: the cluster Overview, Applications, Vertical Pod Autoscalers, the four Device
  Resources views, Helm Charts, Pod Security Policies), and the depth of the 54 partial ones
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
- **Estimate:** 30–100 h (was 85–260; 2–6 h each for the 9 missing views, plus depth: captures, metrics columns).

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
- **Estimate:** 20–70 h (was 45–140; 11 of 39 actions missing, mostly create dialogs, 2–5 h each, plus depth).

### G8. Logs, terminal and shells
- **Missing:** the terminal tab on `fyne-io/terminal`, connected to Cogpit's own PTY (local shell
  with its own kubeconfig) and to client-go exec and attach streams; the node shell with the
  safeguards in [privilege footprint](../security/privilege-footprint.md#node-shell); the log
  viewer (virtualized, follow, filter, timestamps, previous container, download, multi-pod).
- **Evidence:** [ui-parity](ui-parity.md#logs), 10 of 16 rows partial across logs and terminal
  (was 4): the terminal tab on Cogpit's own PTY with its own kubeconfig, the pod shell over exec,
  the log view with follow ([`internal/terminal`](../../internal/terminal/),
  [`termview`](../../internal/ui/fyneui/termview/)); from the third run (2026-10-11): attach to a
  container's main process ([`terminal/attach.go`](../../internal/terminal/attach.go)), and the
  log options as engine state of the tab (`logs.setOptions`): timestamps, previous container, word
  wrap, search with a filter and `/`, download, and the logs of a controlling resource
  (`workload.logs`, `logs.controller`: every pod's containers from the cache, joining while
  following, each line named) ([`engine/logs.go`](../../internal/engine/logs.go),
  [`engine/workloadlogs.go`](../../internal/engine/workloadlogs.go),
  [`logtab.go`](../../internal/ui/fyneui/logtab.go)). Still missing: the node shell, the log
  tab's pod and container selectors, jumping to a time and loading older lines. The terminal
  widget (2026-10-12, fourth run): the patched copy survives every input found by fuzzing it
  alone, without the escape filter (`FuzzHandleOutput`, `FuzzRawWidget`; the crash, hang and
  race patches in [PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)), and its tests run
  in CI (`make test-fork`); its upstream series is prepared, not sent
  ([H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)).
  Fourth run (2026-10-12): a workload's logs resume a follow stream the server ended while its
  container runs, from the newest line's timestamp without repeating lines, and forget ended pods
  and container instances (`TestWorkloadLogsResumeADroppedStream`,
  `TestWorkloadLogsForgetEndedInstances`; [architecture §4.4](../developer/architecture.md#44-tables-details-editors-and-streams)).
- **Impact:** triaging a failing pod (logs, then a shell) is the most common Lens workflow.
- **Decided (owner, 2026-10-12, [D-14](../developer/decisions.md#d-14-terminal-widget)):** the
  patched terminal widget stays; its patches are offered upstream and it is fuzzed on its own,
  the two inputs that still make it panic alone patched first (3–8 h). Built 2026-10-12 except
  the offer, which waits for the owner.
- **Estimate:** 16–40 h (was 30–70; log options, workload logs and attach built, third run); the terminal widget's fuzzing and patches done (fourth run), sending the upstream series under 1 h.

### G9. YAML editor and create resource
- **Missing:** drafts kept for crash recovery, "Revert to original", the Create resource tab's save
  and search buttons, Delete of a selection, the create dialogs (Namespaces, Secrets, Resource Quotas, Service Accounts, RBAC)
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
  gone) turns Failed at once and frees its port; passing errors keep the backoff; not built, 1–2 h.
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
- **To follow Lens's captures of 2026-10-12** ([resource parity › Helm](resource-parity.md#helm), 2–5 h):
  the drawer's title on one line without Details and YAML tabs, Chart as "<chart>-<version>",
  Namespace as text, a grey status pill, "User-supplied values only" unchecked at first, Save as a
  full-width bar; Rollback enabled with the current revision; the Upgrade tab's separate version
  picker, its info line with links, and the editor's find widget for search. Cogpit keeps
  upgrading with the release's stored chart where Lens needs a repository (deliberate).
- **Missing:** install as a crash-safe job; repositories, chart index and search (so an upgrade
  can install the release's own chart version only); the Helm Install dock tab; the ConfigMaps
  and SQL storage drivers. Views and actions count in G6 and G7.
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
- **Estimate:** 12–30 h (was 20–45; the upgrade, its dock tab and Lens's drawer built in the fourth run; left: install and its dock tab, repositories and charts, the other storage drivers).
- **Still to capture in Lens** (a release with five revisions, one failed, prepared on the staging cluster in its own namespace, 2026-10-12): the drawer's Version and Chart texts, "User-supplied values only" at first and what Save does, the texts of states other than Deployed, Rollback with the current revision selected, the upgrade tab's search.

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
- **Decided (owner, 2026-10-12):** the Solarized colours ([D-22](../developer/decisions.md#d-22-colour-theme), 3–6 h), the
  status bar link ([D-23](../developer/decisions.md#d-23-a-link-to-the-owner-in-the-status-bar), 1–2 h), the C helm logo
  ([D-24](../developer/decisions.md#d-24-logo), 2–4 h) and copying values from the drawer
  ([D-25](../developer/decisions.md#d-25-copying-values), 4–8 h); not built.
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
  with history, tab bar, welcome page, keyboard navigation, CSV export, column reorder and hiding,
  row checkboxes and multi-selection in row menus, the drawer's Copy and port Forward…, the Pods
  view's container squares, Cogpit's own title bar on macOS (owner, 2026-10-11: no standard title
  bar, the logo beside the window buttons; [ui-parity](ui-parity.md#layout)).
- **Evidence (2026-10-11, third run, tables and window):** cluster-scoped views hide the namespace
  filter; show and hide columns from the header's ⋮ (`ui.setColumns`, per view in the UI state);
  the Definitions view's group filter (`ui.setGroups`); keyboard navigation (done); a double click
  on a column separator fits the column; "18 items" with a CSV download (`table.exportCsv`, written
  in the engine); the Workloads Overview as a page (`workloads/overview`, `workloads.overview`);
  Cogpit's own title bar on macOS, built but not yet seen in a real window (smoke launch). Still
  missing from this list: history in the title bar, tab bar, welcome page, column reorder, row
  checkboxes and multi-selection, the drawer's Copy and Forward…, container squares.
- **Estimate:** 35–110 h (was 45–130; title bar, keyboard, columns, CSV and the Workloads Overview built, third run).

### G13. Clusters, hotbar, palette, menus, shortcuts and preferences
- **Missing:** kubeconfig syncs (alpha needs these; pasting is not a target, D-21), the hotbar, the command palette,
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
| 2026-10-12 | major | Totals after the fourth run: free parity ~420–1,040 h, full target ~620–1,450 h |
| 2026-10-12 | minor | G8: the terminal widget fuzzed alone, its crashes, hangs and races patched, the upstream series prepared (D-14, fourth run); estimate of that part done |
| 2026-10-12 | minor | Fourth run, the third run's leftovers: G7 evidence (the journals shared by Cogpit processes); G4 evidence (Refresh Access Reviews in the menu, the View menu hides what the sidebar hides); G3 evidence (live related-objects indexes, measured); G8 evidence (workload log streams resumed and pruned); G7 evidence (Restart and Scale bound to the UID in their patches); estimates not moved |
| 2026-10-12 | minor | G2 (fourth run, agent D): kubeconfig files, sync folders and pasted kubeconfigs, the approval of their credential plugins; estimate 10–28 h |
| 2026-10-12 | minor | G11 (fourth run): the Helm upgrade and its dock tab, Lens's release drawer and Rollback dialog built; estimate 12–30 h (the totals not recounted) |
| 2026-10-12 | minor | G3 evidence: covering scopes (fourth run); G6 evidence: Secrets on a full watch, Keys and Type filled, the drawer from the cache (D-13 revised, built) |
| 2026-10-12 | minor | G1: `make parity` built (D-15); its first run's findings; estimate 3–8 h |
| 2026-10-11 | minor | G5: evidence of the third run's tables and window work; G15: why the gate's offscreen test hung in CI, and the fix |
| 2026-10-11 | minor | G6 and G11 evidence: Secrets and Helm Releases partial, Helm rollback and uninstall partial (third run) |
| 2026-10-11 | minor | G10: port forwarding built (manager, local kind, commands, dialog, view); what is left: Lens's status texts and Edit dialog, access review of the forward's pod, a real-window browser check |
| 2026-10-11 | minor | Access reviews built: G2 and G4 evidence, G3 and G9 no longer list them as missing; estimates not moved |
| 2026-10-11 | minor | G6 evidence (third run, agent D): Endpoint Slices, Endpoints, Network Policies, Persistent Volumes, Storage Classes and Replication Controllers views partial; the related-objects index's first users (a claim's pods, a node's Pods table, a Service's endpoint slices) |
| 2026-10-11 | minor | G6 Evidence: the Access Control views (five of them partial) |
| 2026-10-11 | minor | G6 Evidence: the 13 Config views (third implementation run), admission policies by served version |
| 2026-10-12 | minor | Owner's decisions: G1 the parity checker first (D-15); G6 Secrets as a full watch (D-13 revised); G8 the terminal widget kept, upstreamed and fuzzed (D-14); G11 Lens's Helm release drawer, dialogs and upgrade tab captured; G14 the download job belongs to the alpha (D-16) |
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
