# Cogpit vs Lens Desktop: target-app parity

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (re-estimated for the Go plan; macOS first; sources narrowed to Lens Desktop; beyond-Lens targets tracked separately) · **Target:** Lens Desktop 2026.10.51501

The authoritative answer to "how close is Cogpit to Lens, and how much work is left". The
one-page summary is [`ROADMAP.md`](../../ROADMAP.md); the ranked work list is
[`gaps.md`](gaps.md); the checklists are [`ui-parity.md`](ui-parity.md),
[`resource-parity.md`](resource-parity.md) and [`platform-parity.md`](platform-parity.md); what
Cogpit adds beyond Lens is in [`beyond-lens.md`](beyond-lens.md). Method and vocabulary:
[`README.md`](README.md).

## Headline (2026-10-11, before any code)

| Number | Value | Kind |
|---|---|---|
| **Feature breadth** (does each Lens feature exist?) | **0%** | measured: no code exists |
| **Ready for real work** (could a daily Lens user on macOS switch?) | **0%** | measured: nothing runs |
| **Mainstream practitioner** (daily user of Lens's free features) | **0%** | measured |
| **Essentials user** (connect, pods, logs, shell, edit) | **0%** | measured |
| **Stage** | **pre-alpha** | the [alpha gate](roadmap.md#alpha-gate) fails: no core workflow runs |
| Remaining to **alpha** on macOS | **~400–950 agent-hours** | estimated, [calibration](#remaining-effort-and-calibration) |
| Remaining to **beta** on macOS (~75% ready, GitOps views, notarized, updates, budgets met) | **~840–1,870 agent-hours** | estimated |
| Remaining to **full parity** with Lens's free features on macOS | **~800–1,850 agent-hours** | estimated |
| Remaining to the **full Lens target** (paid features, Windows and Linux) | **~1,000–2,300 agent-hours** | estimated; several parts need owner decisions |
| Cogpit's targets **beyond Lens** (MCP control, built-in agent) | **~70–140 agent-hours** | estimated; not part of the parity numbers |

The baseline is zero by construction. It is recorded so that the first measured number has
something to move from, and so that the weights below are fixed before there is anything to
flatter.

### Readiness by audience

| Audience | Ready % | Agent-hours to ~95% | Work that dominates |
|---|---:|---:|---|
| Full Lens target | 0% | 1,000–2,300 | Everything below, plus paid-plan features (G18, G23), Windows and Linux (G22) |
| Mainstream practitioner (macOS) | 0% | 800–1,850 | Resource views and actions (G6, G7: 200–590), the interface (G5, G13: 145–365), logs and terminal (G8) |
| Essentials user (macOS) | 0% | 300–700 | The foundation (G1–G4: 125–250), the spike, a dozen core views and actions, logs and shell, the editor |

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

**0% (weighted).** Denominators count Lens Desktop's free features; paid features have their own
component.

| Component | Weight | Ours / Lens | % | Kind |
|---|---:|---|---:|---|
| Resource views | 30% | 0 / 63 | 0% | measured ([resource-parity](resource-parity.md)) |
| Resource actions | 20% | 0 / 40 | 0% | measured ([resource-parity](resource-parity.md)) |
| Interface: layout, navigation, tables, drawer, dock, editor, logs, terminal, palette, menus, shortcuts, preferences | 25% | 0 / 111 | 0% | measured ([ui-parity](ui-parity.md), free rows) |
| Cluster authentication and connectivity | 10% | 0 / 13 | 0% | measured ([platform-parity](platform-parity.md#cluster-authentication-and-connectivity)) |
| Platforms and packages | 5% | 0 / 13 | 0% | measured ([platform-parity](platform-parity.md#operating-systems-and-packages)) |
| Paid Lens Desktop features | 10% | 0 / 11 | 0% | measured ([premium list](https://docs.lenshq.io/k8slens/premium-features/) without extensions, D-5) |
| **Weighted** | 100% | | **0%** | |

## Ready for real work

**0% (weighted).** Weights reflect what a daily Lens user depends on. Connectivity weighs as much
as the whole interface because a user who cannot authenticate to their cluster cannot use anything
else.

| Dimension | Weight | % ready | Remaining (agent-h) | Evidence | Detail |
|---|---:|---:|---:|---|---|
| Resource views and actions in depth | 28% | 0% | 200–590 | 0 of 63 views, 0 of 40 actions | [resource-parity](resource-parity.md), G6, G7 |
| Interface fidelity and feel | 15% | 0% | 180–435 | 0 of 111 rows | [ui-parity](ui-parity.md), G5, G9, G13 |
| Cluster connectivity and authentication | 15% | 0% | 25–50 + human | 0 of 13 rows; no plugin verified | [platform-parity](platform-parity.md#cluster-authentication-and-connectivity), G2 |
| Logs, terminal, shells, port forwarding | 10% | 0% | 55–110 | nothing built | [ui-parity](ui-parity.md#logs), G8, G10 |
| Performance and API load | 10% | 0% | 40–80 | no budget measured | [testing](../developer/testing.md#performance-budgets-proposed-not-measured), G15 |
| Stability | 10% | 0% | 100–200 | nothing to run | [gaps](gaps.md#g16-stability-and-the-user-bug-backlog), G16 |
| Helm releases and metrics | 5% | 0% | 70–140 | nothing built | G11, G12 |
| Platforms and distribution | 5% | 0% | 65–130 + human | 0 of 13 rows | [platform-parity](platform-parity.md#operating-systems-and-packages), G14, G22 |
| Paid-plan integrations | 2% | 0% | 165–335 + human | 0 of 11 | G18, G23 |
| Foundation: workspace, cache, engine (enables every row) | – | – | 100–200 | nothing built | G1, G3, G4 |
| **Weighted** | 100% | **0%** | **1,000–2,270** | | |

## Mainstream practitioner

**0%.** A daily user of Lens's free features on macOS. Method as in PhotoCraft: weighted depth of
the areas used every week, times discounts for what stops real work.

| Area | Weight | Depth |
|---|---:|---:|
| Connect to and switch between clusters | 15% | 0% |
| Workload views and actions | 25% | 0% |
| Logs | 15% | 0% |
| Shell into containers | 10% | 0% |
| Edit and apply YAML | 10% | 0% |
| Config, network and storage views | 10% | 0% |
| Port forwarding | 5% | 0% |
| Helm releases in the cluster | 5% | 0% |
| Metrics at a glance | 5% | 0% |
| **Weighted depth** | 100% | **0%** |

Discounts, applied once there is depth to discount: stability on real clusters, speed on large
clusters, and authentication against the cloud providers' credential plugins.

## Essentials user

**0%.** Someone who opens Lens to look at their pods and fix one.

| Core feature | Weight | Depth |
|---|---:|---:|
| Add a kubeconfig and connect | 20% | 0% |
| Pods list with status | 15% | 0% |
| Logs of a pod | 20% | 0% |
| Shell into a pod | 15% | 0% |
| Details and events of an object | 10% | 0% |
| Edit YAML and apply | 10% | 0% |
| Delete, scale, restart | 10% | 0% |
| **Weighted depth** | 100% | **0%** |

## Feature areas

Percent is ready-for-real-work for the area; hours are the views and actions of the area (2–6 h
each), contained in G6 and G7 above, plus the named subsystem where one exists.

| Area | Views | Actions | % | Hours |
|---|---:|---:|---:|---:|
| Cluster: overview, applications, nodes, namespaces, events | 5 | 5 | 0% | 20–60, plus metrics (G12) |
| Workloads | 9 | 16 | 0% | 50–150 |
| Config | 16 | 2 | 0% | 35–110 |
| Network, Gateway API included | 17 | 5 | 0% | 45–130, plus port forwarding (G10) |
| Storage | 3 | – | 0% | 5–20 |
| Device resources | 4 | – | 0% | 10–25 |
| Helm releases and charts | 2 | 4 | 0% | 10–35, plus Helm (G11) |
| Access control | 6 | 5 | 0% | 20–65 |
| Custom resources | 1 | – | 0% | 2–6 |
| Actions on every kind (edit, delete, copy name) | – | 3 | 0% | 5–20 |

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
| **Alpha (macOS)** | Foundation (G1–G4), the spike, core shell, ~20 views and ~20 actions, logs, shells, editor, port forwarding, Helm releases, MCP basics, an unsigned build | **~400–950** |
| **Beta (macOS)** | Alpha plus metrics, most of the interface, MCP control, the Flux CD and Argo CD views, ~75% of views and actions, a notarized app with one-click updates, performance budgets, the first bug backlog | **~840–1,870** |
| **Full parity (free features, macOS)** | G1–G16 | **~800–1,850** |
| **Full Lens target** | plus G18, G22, G23 | **~1,000–2,300** |
| **Beyond Lens** | G17, G21 | **~70–140** |

**Parallelism.** The foundation (G1–G4, 125–250 h) and the spike are serial: everything else builds
on them. After that, about 80% parallelizes, as in PhotoCraft: one kind profile or command per agent.
With about ten agents after the foundation, alpha is roughly 220–460 wall-clock hours, most of it
the serial foundation.

**Needs a human.** The go-ahead to implement; the open decision D-4 for Windows and Linux (with M11)
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
| 2026-10-11 | major | Re-estimated for Go, client-go, the kubectl libraries and the Helm SDK; macOS first; sources narrowed to Lens Desktop; beyond-Lens targets separated |
| 2026-10-11 | major | First version: target, scope, weights, audiences and a zero baseline before any code |
