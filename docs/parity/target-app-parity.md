# Cogpit vs Lens Desktop: target-app parity

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (first measurement after the local MVP) · **Target:** Lens Desktop 2026.10.51501

The authoritative answer to "how close is Cogpit to Lens, and how much work is left". The
one-page summary is [`ROADMAP.md`](../../ROADMAP.md); the ranked work list is
[`gaps.md`](gaps.md); the checklists are [`ui-parity.md`](ui-parity.md),
[`resource-parity.md`](resource-parity.md) and [`platform-parity.md`](platform-parity.md); what
Cogpit adds beyond Lens is in [`beyond-lens.md`](beyond-lens.md). Method and vocabulary:
[`README.md`](README.md).

## Headline (2026-10-11, after the local MVP)

| Number | Value | Kind |
|---|---|---|
| **Feature breadth** (does each Lens feature exist?) | **~12%** | measured: [breadth](#feature-breadth); every existing row is partial |
| **Ready for real work** (could a daily Lens user on macOS switch?) | **~10%** | estimated: [weights](#ready-for-real-work) |
| **Mainstream practitioner** (daily user of Lens's free features) | **~8%** | estimated |
| **Essentials user** (connect, pods, logs, shell, edit) | **~23%** | estimated |
| **Stage** | **pre-alpha** | the [alpha gate](roadmap.md#alpha-gate) fails: editing, port forwarding, Helm and node maintenance do not exist |
| Remaining to **alpha** on macOS | **~250–700 agent-hours** | estimated (was ~400–950), [calibration](#remaining-effort-and-calibration) |
| Remaining to **beta** on macOS (~75% ready, GitOps views, notarized, updates, budgets met) | **~680–1,610 agent-hours** | estimated (was ~840–1,870) |
| Remaining to **full parity** with Lens's free features on macOS | **~670–1,650 agent-hours** | estimated (was ~800–1,850) |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~870–2,060 agent-hours** | estimated (was ~1,000–2,300); several parts need owner decisions |
| Cogpit's targets **beyond Lens** (MCP control, built-in agent) | **~50–110 agent-hours** | estimated (was ~70–140); not part of the parity numbers |

The first measurement after code. Breadth runs ahead of readiness, as the method expects: the MVP
touches many rows shallowly. The remaining hours fall by what [gaps](gaps.md#summary) says is built,
row by row; the borrowed calibration itself is unchanged
([data point](#remaining-effort-and-calibration)).

### Readiness by audience

| Audience | Ready % | Agent-hours to ~95% | Work that dominates |
|---|---:|---:|---|
| Full Lens target | ~7% | 870–2,060 | Everything below, plus paid-plan features (G18, G23), Windows and Linux (G22) |
| Mainstream practitioner (macOS) | ~8% | 670–1,650 | Resource views and actions (G6, G7: 182–555), the interface (G5, G13: 125–330), logs and terminal (G8) |
| Essentials user (macOS) | ~23% | 200–520 | Editing YAML (G9), the remaining foundation, more core views and actions, log options, cloud credential plugins |

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

**~12% (weighted, measured 2026-10-11).** Denominators count Lens Desktop's free features; paid
features have their own component. A row counts when it exists in some form; every counted row is
partial.

| Component | Weight | Ours / Lens | % | Kind |
|---|---:|---|---:|---|
| Resource views | 30% | 6 / 64 | 9.4% | measured ([resource-parity](resource-parity.md)) |
| Resource actions | 20% | 4 / 39 | 10.3% | measured ([resource-parity](resource-parity.md)) |
| Interface: layout, navigation, tables, drawer, dock, editor, logs, terminal, palette, menus, shortcuts, preferences | 25% | 26 / 121 | 21.5% | measured ([ui-parity](ui-parity.md), free rows) |
| Cluster authentication and connectivity | 10% | 2 / 13 | 15.4% | measured ([platform-parity](platform-parity.md#cluster-authentication-and-connectivity)) |
| Platforms and packages | 5% | 1 / 13 | 7.7% | measured ([platform-parity](platform-parity.md#operating-systems-and-packages)) |
| Paid Lens Desktop features | 10% | 0 / 11 | 0% | measured ([premium list](https://docs.lenshq.io/k8slens/premium-features/) without extensions, D-5) |
| **Weighted** | 100% | | **12.2%** | |

## Ready for real work

**~10% (weighted, estimated 2026-10-11).** Weights reflect what a daily Lens user depends on. Connectivity weighs as much
as the whole interface because a user who cannot authenticate to their cluster cannot use anything
else.

| Dimension | Weight | % ready | Remaining (agent-h) | Evidence | Detail |
|---|---:|---:|---:|---|---|
| Resource views and actions in depth | 28% | 5% | 182–555 | 6 of 64 views and 4 of 39 actions, all partial | [resource-parity](resource-parity.md), G6, G7 |
| Interface fidelity and feel | 15% | 8% | 152–390 | 26 of 121 rows partial; no editing | [ui-parity](ui-parity.md), G5, G9, G13 |
| Cluster connectivity and authentication | 15% | 15% | 18–40 + human | 2 of 13 rows partial; contexts from the default kubeconfig; no plugin verified | [platform-parity](platform-parity.md#cluster-authentication-and-connectivity), G2 |
| Logs, terminal, shells, port forwarding | 10% | 20% | 40–90 | logs with follow, cluster-bound terminal, pod shell; no attach, node shell, port forwarding | [ui-parity](ui-parity.md#logs), G8, G10 |
| Performance and API load | 10% | 25% | 30–70 | P2, P9–P11 measured; frames on the GPU only once | [testing](../developer/testing.md#performance-budgets-proposed-not-measured), G15 |
| Stability | 10% | 5% | 100–200 | race-clean tests; no use on real clusters yet; the terminal widget runs as a patched copy (H-19) | [gaps](gaps.md#g16-stability-and-the-user-bug-backlog), G16 |
| Helm releases and metrics | 5% | 0% | 70–140 | nothing built | G11, G12 |
| Platforms and distribution | 5% | 5% | 62–125 + human | a local unsigned app bundle | [platform-parity](platform-parity.md#operating-systems-and-packages), G14, G22 |
| Paid-plan integrations | 2% | 0% | 165–335 + human | 0 of 11 | G18, G23 |
| Foundation: workspace, cache, engine (enables every row) | – | – | 52–118 | the cache, engine and lint layering are built; CI, generators, jobs and access reviews are not | G1, G3, G4 |
| **Weighted** | 100% | **~10%** | **871–2,063** | | |

## Mainstream practitioner

**~8% (estimated).** A daily user of Lens's free features on macOS. Method as in PhotoCraft:
weighted depth of the areas used every week, times discounts for what stops real work.

| Area | Weight | Depth |
|---|---:|---:|
| Connect to and switch between clusters | 15% | 30% |
| Workload views and actions | 25% | 10% |
| Logs | 15% | 25% |
| Shell into containers | 10% | 60% |
| Edit and apply YAML | 10% | 0% |
| Config, network and storage views | 10% | 5% |
| Port forwarding | 5% | 0% |
| Helm releases in the cluster | 5% | 0% |
| Metrics at a glance | 5% | 0% |
| **Weighted depth** | 100% | **~17%** |

Discounts: stability on real clusters (unproven: ×0.7) and authentication against the cloud
providers' credential plugins (unverified: ×0.7) give **~8%**. Speed on large clusters is not
discounted: the demo cluster's 10,000 pods meet the budgets measured so far.

## Essentials user

**~23% (estimated).** Someone who opens Lens to look at their pods and fix one.

| Core feature | Weight | Depth |
|---|---:|---:|
| Add a kubeconfig and connect | 20% | 40% |
| Pods list with status | 15% | 70% |
| Logs of a pod | 20% | 40% |
| Shell into a pod | 15% | 70% |
| Details and events of an object | 10% | 60% |
| Edit YAML and apply | 10% | 0% |
| Delete, scale, restart | 10% | 40% |
| **Weighted depth** | 100% | **~47%** |

With the same two discounts as above (×0.7 each): **~23%**.

## Feature areas

Percent is ready-for-real-work for the area; hours are the views and actions of the area (2–6 h
each), contained in G6 and G7 above, plus the named subsystem where one exists.

| Area | Views | Actions | % | Hours |
|---|---:|---:|---:|---:|
| Cluster: overview, applications, nodes, namespaces, events | 5 | 5 | ~15% (Nodes, Namespaces, Events views partial) | 18–55, plus metrics (G12) |
| Workloads | 9 | 16 | ~10% (Pods, Deployments views; Logs, Shell, Restart partial) | 45–140 |
| Config | 16 | 2 | 0% | 35–110 |
| Network, Gateway API included | 17 | 5 | ~3% (Services view partial) | 43–125, plus port forwarding (G10) |
| Storage | 3 | – | 0% | 5–20 |
| Device resources | 4 | – | 0% | 10–25 |
| Helm releases and charts | 2 | 4 | 0% | 10–35, plus Helm (G11) |
| Access control | 6 | 5 | 0% | 20–65 |
| Custom resources | 1 | – | 0% | 2–6 |
| Actions on every kind (edit, delete, copy name) | – | 3 | ~10% (Delete for pods) | 5–18 |

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
| **Alpha (macOS)** | Foundation (G1–G4), the spike, core shell, ~20 views and ~20 actions, logs, shells, editor, port forwarding, Helm releases, MCP basics, an unsigned build | **~250–700** (was ~400–950) |
| **Beta (macOS)** | Alpha plus metrics, most of the interface, MCP control, the Flux CD and Argo CD views, ~75% of views and actions, a notarized app with one-click updates, performance budgets, the first bug backlog | **~680–1,610** (was ~840–1,870) |
| **Full parity (free features, macOS)** | G1–G16 | **~670–1,650** (was ~800–1,850) |
| **Full Lens target** | plus G18, G22, G23 | **~870–2,060** (was ~1,000–2,300) |
| **Beyond Lens** | G17, G21 | **~50–110** (was ~70–140) |

**Cogpit's first data point (2026-10-11), not yet a calibration.** The MVP was built in one session:
a serial skeleton by the lead, then eight agents in parallel for 18 to 45 minutes each (about
4.7 agent-hours together), then integration, review and documentation. Against the borrowed
calibration that work retired roughly 150–250 estimated agent-hours (the drop of the alpha
estimate plus the spike), but it leaves every row partial: it is the first, shallow pass over each
row, which is where agents are fastest. One session does not replace PhotoCraft's calibration;
it says the remaining estimates may be high for first passes and says nothing yet about the depth
work.

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
| 2026-10-11 | major | First measurement after the local MVP: breadth ~12% (measured), ready ~10%, mainstream ~8%, essentials ~23% (estimated); remaining hours re-estimated from the gaps |
| 2026-10-11 | major | Re-estimated for Go, client-go, the kubectl libraries and the Helm SDK; macOS first; sources narrowed to Lens Desktop; beyond-Lens targets separated |
| 2026-10-11 | major | First version: target, scope, weights, audiences and a zero baseline before any code |
