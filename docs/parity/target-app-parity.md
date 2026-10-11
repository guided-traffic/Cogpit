# Cogpit vs Lens Desktop: target-app parity

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (third measurement, after the third implementation run) · **Target:** Lens Desktop 2026.10.51501

The authoritative answer to "how close is Cogpit to Lens, and how much work is left". The
one-page summary is [`ROADMAP.md`](../../ROADMAP.md); the ranked work list is
[`gaps.md`](gaps.md); the checklists are [`ui-parity.md`](ui-parity.md),
[`resource-parity.md`](resource-parity.md) and [`platform-parity.md`](platform-parity.md); what
Cogpit adds beyond Lens is in [`beyond-lens.md`](beyond-lens.md). Method and vocabulary:
[`README.md`](README.md).

## Headline (2026-10-11, after the third implementation run)

| Number | Value | Kind |
|---|---|---|
| **Feature breadth** (does each Lens feature exist?) | **~54%** (was ~30%) | measured: [breadth](#feature-breadth); 9 rows done, the rest partial |
| **Ready for real work** (could a daily Lens user on macOS switch?) | **~23%** (was ~14%) | estimated: [weights](#ready-for-real-work) |
| **Mainstream practitioner** (daily user of Lens's free features) | **~24%** (was ~14%) | estimated |
| **Essentials user** (connect, pods, logs, shell, edit) | **~32%** (was ~28%) | estimated |
| **Stage** | **pre-alpha** | the [alpha gate](roadmap.md#alpha-gate) fails: every workflow exists in part now, port forwarding and Helm's read side, rollback and uninstall included, but none is checked end to end in the GUI on a real cluster, and Helm upgrade does not exist |
| Remaining to **alpha** on macOS | **~150–400 agent-hours** | estimated (was ~210–520), [calibration](#remaining-effort-and-calibration) |
| Remaining to **beta** on macOS (~75% ready, GitOps views, notarized, updates, budgets met) | **~440–1,020 agent-hours** | estimated (was ~580–1,360) |
| Remaining to **full parity** with Lens's free features on macOS | **~430–1,060 agent-hours** | estimated (was ~570–1,400) |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~640–1,480 agent-hours** | estimated (was ~780–1,820); several parts need owner decisions |
| Cogpit's targets **beyond Lens** (MCP control, built-in agent) | **~50–110 agent-hours** | estimated, unchanged; not part of the parity numbers |

The third measurement. Breadth grew by more than half again: 28 views (55 of 64 exist), 14
actions (28 of 39) and 15 interface rows (52 of 121) came in one run, among them port forwarding
and Helm's releases, the two alpha-gate workflows that did not exist. Readiness moved less, for
the same reason as before: the new rows are partial, most texts and rules are inferred rather than
captured, and no workflow was walked through end to end by a person on a real cluster.
The remaining hours fall by what [gaps](gaps.md#summary) says is built, row by row; the borrowed
calibration itself is unchanged ([data points](#remaining-effort-and-calibration)).

### Readiness by audience

| Audience | Ready % | Agent-hours to ~95% | Work that dominates |
|---|---:|---:|---|
| Full Lens target | ~16% | 640–1,480 | Everything below, plus paid-plan features (G18, G23), Windows and Linux (G22) |
| Mainstream practitioner (macOS) | ~24% | 430–1,060 | The interface (G5, G13: 100–275), stability (G16), metrics (G12), the depth of the views and actions (G6, G7: 50–170), Helm upgrade and charts |
| Essentials user (macOS) | ~32% | 100–300 | Cloud credential plugins, pasting and syncing kubeconfigs, checking the flows end to end on real clusters, the remaining foundation |

Each tier is a subset of the one above.

## Target measured

- **Lens Desktop 2026.10.51501**, the stable release in Lens's update feeds on 2026-10-11 (feed
  date 2026-10-05; the release notes, dated 2026-10-08, say it rolls out gradually); Electron 44,
  Chromium 152. Lens's documentation lives at [docs.lenshq.io](https://docs.lenshq.io/)
  (`docs.k8slens.dev` redirects there) and calls the product "Lens K8S IDE". Measured from that
  documentation, the release notes from 2025-10 to 2026-10, the update feeds and the pricing page.
- **Not run.** Lens Desktop requires a Lens ID at first launch; rows marked *capture needed* in the
  checklists need a look at an activated install (human).
- **Only Lens Desktop** ([D-12](../developer/decisions.md#d-12-parity-sources)). OpenLens, whose
  source was retired in January 2024, and its fork Freelens describe an older Lens and are not
  sources.
- **Plans.** Lens Desktop Personal is free for individuals and organizations under $10M revenue
  or funding; Plus and Pro cost $25 per seat per month billed annually, Enterprise $50
  ([pricing](https://lenshq.io/pricing)). Twelve features are paid
  ([premium features](https://docs.lenshq.io/k8slens/premium-features/)).

## Scope

| Lens feature group | Counts toward | Why |
|---|---|---|
| Free (Personal plan) features | every number | What a typical Lens user uses daily |
| Paid features: Flux CD and Argo CD views | full Lens target; built for the beta | Daily work for the owner's team ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) |
| Paid features: EKS, AKS, GKE integrations, Security Center, cost monitoring | full Lens target only, after the beta | Need cloud accounts and data sources; not the mainstream audience's daily work |
| Extensions installed from outside (paid in Lens) | nothing | Integrations are built into Cogpit's code ([D-5](../developer/decisions.md#d-5-extensions)) |
| Paid features: Hotbar, editor diff | full Lens target; built with the interface and the editor | Cheap |
| Paid Ask AI and MCP server | superseded | Cogpit's built-in agent and MCP control go further ([beyond Lens](beyond-lens.md)) |
| Vendor features: Lens ID activation, subscriptions, telemetry, surveys | nothing | They serve the vendor, not the user ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) |
| Teamwork and Cluster Connect | nothing, unless the owner decides | Needs a hosted relay service (G20) |
| Windows and Linux | full Lens target | macOS comes first ([D-7](../developer/decisions.md#d-7-language-and-platforms)) |

## Feature breadth

**~54% (weighted, measured 2026-10-11, after the third implementation run; was ~30%).**
Denominators count Lens Desktop's free features; paid features have their own component. A row
counts when it exists in some form; nine counted rows are done, the rest partial.

| Component | Weight | Ours / Lens | % | Kind |
|---|---:|---|---:|---|
| Resource views | 30% | 55 / 64 | 85.9% | measured ([resource-parity](resource-parity.md)); ten of them the Gateway API views, generic until captured |
| Resource actions | 20% | 28 / 39 | 71.8% | measured ([resource-parity](resource-parity.md)) |
| Interface: layout, navigation, tables, drawer, dock, editor, logs, terminal, palette, menus, shortcuts, preferences | 25% | 52 / 121 | 43.0% | measured ([ui-parity](ui-parity.md), free rows) |
| Cluster authentication and connectivity | 10% | 3 / 13 | 23.1% | measured ([platform-parity](platform-parity.md#cluster-authentication-and-connectivity)) |
| Platforms and packages | 5% | 1 / 13 | 7.7% | measured ([platform-parity](platform-parity.md#operating-systems-and-packages)) |
| Paid Lens Desktop features | 10% | 1 / 11 | 9.1% | measured ([premium list](https://docs.lenshq.io/k8slens/premium-features/) without extensions, D-5): the editor's diff, partial |
| **Weighted** | 100% | | **54.5%** | |

## Ready for real work

**~23% (weighted, estimated 2026-10-11, after the third implementation run; was ~14%).** Weights reflect what a daily Lens user depends on. Connectivity weighs as much
as the whole interface because a user who cannot authenticate to their cluster cannot use anything
else.

| Dimension | Weight | % ready | Remaining (agent-h) | Evidence | Detail |
|---|---:|---:|---:|---|---|
| Resource views and actions in depth | 28% | 28% | 50–170 | 55 of 64 views (one done) and 28 of 39 actions, the rest partial; most texts and rules inferred, not checked against a capture | [resource-parity](resource-parity.md), G6, G7 |
| Interface fidelity and feel | 15% | 22% | 100–275 | 52 of 121 rows (8 done); Cogpit's own title bar on macOS (not yet seen in a real window), keyboard navigation, column choice, CSV, the Workloads Overview | [ui-parity](ui-parity.md), G5, G9, G13 |
| Cluster connectivity and authentication | 15% | 18% | 14–34 + human | 3 of 13 rows partial; access reviews grey what the user may not do; no plugin verified | [platform-parity](platform-parity.md#cluster-authentication-and-connectivity), G2 |
| Logs, terminal, shells, port forwarding | 10% | 45% | 18–46 | log options, logs of a workload's pods, attach, port forwarding with reconnects (on kind); no node shell | [ui-parity](ui-parity.md#logs), G8, G10 |
| Performance and API load | 10% | 25% | 28–66 | P2, P9–P11 measured; frames on the GPU only once; the perf gate no longer hangs on a slow runner | [testing](../developer/testing.md#performance-budgets-proposed-not-measured), G15 |
| Stability | 10% | 9% | 100–200 | race-clean tests and CI on every push; four timing-dependent tests fixed after the merge; no use on real clusters yet; the terminal widget runs as a patched copy (H-19) | [gaps](gaps.md#g16-stability-and-the-user-bug-backlog), G16 |
| Helm releases and metrics | 5% | 15% | 60–125 | Helm releases read from the cache, rollback and uninstall crash-safe on kind; no upgrade, install, charts or metrics | G11, G12 |
| Platforms and distribution | 5% | 5% | 62–125 + human | a local unsigned app bundle | [platform-parity](platform-parity.md#operating-systems-and-packages), G14, G22 |
| Paid-plan integrations | 2% | 5% | 165–335 + human | 1 of 11 partial (the editor's diff) | G18, G23 |
| Foundation: workspace, cache, engine (enables every row) | – | – | 32–80 | the cache with scoped, metadata-only, local and virtual kinds, the related-objects index, access reviews, CI; generators and jobs are not | G1, G3, G4 |
| **Weighted** | 100% | **~23%** | **629–1,456** | | |

## Mainstream practitioner

**~24% (estimated; was ~14%).** A daily user of Lens's free features on macOS. Method as in PhotoCraft:
weighted depth of the areas used every week, times discounts for what stops real work.

| Area | Weight | Depth |
|---|---:|---:|
| Connect to and switch between clusters | 15% | 35% |
| Workload views and actions | 25% | 55% |
| Logs | 15% | 55% |
| Shell into containers | 10% | 65% |
| Edit and apply YAML | 10% | 55% |
| Config, network and storage views | 10% | 55% |
| Port forwarding | 5% | 60% |
| Helm releases in the cluster | 5% | 35% |
| Metrics at a glance | 5% | 0% |
| **Weighted depth** | 100% | **~50%** |

Discounts: stability on real clusters (unproven: ×0.7) and authentication against the cloud
providers' credential plugins (unverified: ×0.7) give **~24%**. Speed on large clusters is not
discounted: the demo cluster's 10,000 pods meet the budgets measured so far.

## Essentials user

**~32% (estimated; was ~28%).** Someone who opens Lens to look at their pods and fix one.

| Core feature | Weight | Depth |
|---|---:|---:|
| Add a kubeconfig and connect | 20% | 42% |
| Pods list with status | 15% | 78% |
| Logs of a pod | 20% | 70% |
| Shell into a pod | 15% | 72% |
| Details and events of an object | 10% | 75% |
| Edit YAML and apply | 10% | 55% |
| Delete, scale, restart | 10% | 75% |
| **Weighted depth** | 100% | **~65%** |

With the same two discounts as above (×0.7 each): **~32%**.

## Feature areas

Percent is ready-for-real-work for the area; hours are the views and actions of the area (2–6 h
each), contained in G6 and G7 above, plus the named subsystem where one exists.

| Area | Views | Actions | % | Hours |
|---|---:|---:|---:|---:|
| Cluster: overview, applications, nodes, namespaces, events | 5 | 5 | ~25% (Nodes with its Pods table, Namespaces, Events; Cordon, Uncordon, Drain partial; no node shell, no Create Namespace) | 12–35, plus metrics (G12) |
| Workloads | 9 | 16 | ~55% (every view, the Overview page included; all 15 actions that are targets exist, partial; Share is not a target) | 8–30 |
| Config | 16 | 2 | ~40% (15 of 16 views partial, Secrets metadata only; no create dialogs) | 10–35 |
| Network, Gateway API included | 17 | 5 | ~45% (every view; port forwarding and its four actions; the Gateway API views generic) | 6–25 |
| Storage | 3 | – | ~40% (every view partial, a claim's pods) | 1–5 |
| Device resources | 4 | – | 0% | 10–25 |
| Helm releases and charts | 2 | 4 | ~30% (Releases with history, values and manifest; rollback and uninstall; no charts, install, upgrade) | 6–20, plus Helm (G11) |
| Access control | 6 | 5 | ~35% (5 of 6 views partial; no create dialogs) | 12–35 |
| Custom resources (Definitions and the custom resource views) | 2 | – | ~65% (custom resource views done, Definitions with its group filter partial) | 1–3 |
| Actions on every kind (edit, delete, copy name) | – | 3 | ~55% (all three, partial; Edit and Delete greyed by access reviews) | 1–5 |

## Remaining effort and calibration

**Calibration (borrowed).** Cogpit has no history yet. Estimates use PhotoCraft's calibration from
its first 1,004 merged pull requests (2026-09-30 to 2026-10-10, median open-to-merge 3.4 h):
2–9 agent-hours per checklist item of tool or UI depth and 8–20 h per format or subsystem, with
about 2–4 agent-hours per merged pull request. A Kubernetes resource view or action is smaller
than a Photoshop tool behaviour, so this page uses 2–6 h for those and 1–4 h for small interface
rows. Building on client-go, the kubectl libraries and the Helm SDK makes authentication, actions
and Helm cheaper than in the earlier Rust plan; the cache requirement and Cogpit's own YAML editor
make the cache and the editor more expensive. Replace the calibration with Cogpit's own after about
100 merged pull requests.

| Milestone | Scope | Opus 5.5 agent-hours |
|---|---|---:|
| **Alpha (macOS)** | Foundation (G1–G4), the spike, core shell, ~20 views and ~20 actions, logs, shells, editor, port forwarding, Helm releases, MCP basics, an unsigned build | **~150–400** (was ~210–520): the alpha gate's rows (120–317) plus the foundation left (G1, G3, G4: 32–80) |
| **Beta (macOS)** | Alpha plus metrics, most of the interface, MCP control, the Flux CD and Argo CD views, ~75% of views and actions, a notarized app with one-click updates, performance budgets, the first bug backlog | **~440–1,020** (was ~580–1,360; lowered by what the free-parity total lost) |
| **Full parity (free features, macOS)** | G1–G16 | **~430–1,060** (was ~570–1,400) |
| **Full Lens target** | plus G18, G22, G23 | **~640–1,480** (was ~780–1,820) |
| **Beyond Lens** | G17, G21 | **~50–110** (unchanged) |

**Cogpit's first data point (2026-10-11), not yet a calibration.** The MVP was built in one session:
a serial skeleton by the lead, then eight agents in parallel for 18 to 45 minutes each (about
4.7 agent-hours together), then integration, review and documentation. Against the borrowed
calibration that work retired roughly 150–250 estimated agent-hours (the drop of the alpha
estimate plus the spike), but it leaves every row partial: it is the first, shallow pass over each
row, which is where agents are fastest. One session does not replace PhotoCraft's calibration;
it says the remaining estimates may be high for first passes and says nothing yet about the depth
work.

**Second data point (2026-10-11).** The second run: a serial phase by the lead (contracts, kind,
dependencies), then seven agents in parallel for 22 to 72 minutes each (about 4.8 agent-hours
together), then merging, CI, a review and documentation. It retired about 100–250 estimated
agent-hours of the free-parity total (G1, G3, G5, G6, G7, G9) and 40–180 of the alpha estimate.
As in the first run, most rows it touched are partial: the ratio says first passes are cheap, not
that depth is.

**Third data point (2026-10-11).** The third run: a serial Phase 0 by the lead (the shared
contracts: scoped, metadata-only, local and virtual kinds; the related-objects index; access
reviews in `Enabled`; item actions in the drawer; the Helm SDK in `go.mod`), then eight agents in
parallel for 36 to 86 minutes each (about 7.8 agent-hours together), then merging one branch at a
time, CI, a review and documentation. It retired about 140–340 estimated agent-hours of the
free-parity total (G2–G11, G15) and 60–120 of the alpha estimate. Merging cost more than in the
earlier runs: eight branches touched the same tables, constants and parity rows, and four tests
that passed on every branch alone turned out to be timing-dependent on CI's slower runners.

**Parallelism.** The foundation (G1–G4, 125–250 h) and the spike are serial: everything else builds
on them. After that, about 80% parallelizes, as in PhotoCraft: one kind profile or command per agent.
With about ten agents after the foundation, alpha is roughly 220–460 wall-clock hours, most of it
the serial foundation.

**Needs a human.** The decision on the patched terminal widget ([H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)); the open decision D-4 for Windows and Linux (with M11)
([decisions](../developer/decisions.md)); EKS, GKE and AKS accounts to verify credential plugins;
Apple signing material and the update signing key; an activated Lens Desktop to capture the rows
marked *capture needed*; Lens Desktop on the same Mac as Cogpit to set the performance budgets; a
LiteLLM or Open WebUI endpoint for the built-in agent.

## How to re-measure

1. Once the generators exist (G1): `make parity` and `make scorecard` for the measured rows.
2. Views and actions: compare the kind-profile and command registries with the tables in
   [`resource-parity.md`](resource-parity.md).
3. Interface: walk the rows of [`ui-parity.md`](ui-parity.md) in the running app, with a screenshot
   as evidence for visual rows.
4. Lens side: for every Lens Desktop release after the target, read its release notes, update the
   rows they touch and move the `Target` of every progress page in one change.
5. Move an estimated number only with new evidence, and add a revision-history row.

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | major | Third measurement after the third implementation run: breadth ~54% (measured: 55 of 64 views, 28 of 39 actions, 52 of 121 interface rows, 3 of 13 connectivity rows), ready ~23%, mainstream ~24%, essentials ~32% (estimated); alpha ~150–400 h, free parity ~430–1,060 h |
| 2026-10-11 | major | Second measurement after the second implementation run: breadth ~30% (measured), ready ~14%, mainstream ~14%, essentials ~28% (estimated); remaining hours re-estimated from the gaps |
| 2026-10-11 | major | First measurement after the local MVP: breadth ~12% (measured), ready ~10%, mainstream ~8%, essentials ~23% (estimated); remaining hours re-estimated from the gaps |
| 2026-10-11 | major | Re-estimated for Go, client-go, the kubectl libraries and the Helm SDK; macOS first; sources narrowed to Lens Desktop; beyond-Lens targets separated |
| 2026-10-11 | major | First version: target, scope, weights, audiences and a zero baseline before any code |
