# Parity and progress documents

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** minor (beyond-Lens page added; sources limited to Lens Desktop; generators run through make) · **Target:** Lens Desktop 2026.10.51501

Cogpit is a clean-room reimplementation of the Kubernetes IDE Lens. These pages answer two
questions and nothing else: **how close are we to Lens**, and **what is left to do**. The method is
taken from [PhotoCraft](https://github.com/storytold/photocraft) (a Photoshop reimplementation in
Rust), whose progress docs follow the same structure.

## The pages

| Page | Answers | Kind |
|---|---|---|
| [`../../ROADMAP.md`](../../ROADMAP.md) | One-page summary: stage, the two headline numbers, every dimension, what's next | summary |
| [`target-app-parity.md`](target-app-parity.md) | The authoritative assessment: numbers, weights, evidence, hours, how to re-measure | assessment |
| [`gaps.md`](gaps.md) | Every known shortfall, ranked: the work list to pick from | work list |
| [`roadmap.md`](roadmap.md) | Milestones with definitions of done, the alpha gate, the current focus | plan |
| [`ui-parity.md`](ui-parity.md) | Layout, navigation, tables, detail panels, dock, shortcuts, menus, preferences | checklist |
| [`resource-parity.md`](resource-parity.md) | Every Kubernetes resource view Lens has, with its columns and actions | checklist |
| [`platform-parity.md`](platform-parity.md) | Operating systems, architectures, packaging, signing, updates, cluster auth methods | checklist |
| [`beyond-lens.md`](beyond-lens.md) | Cogpit's own targets beyond Lens: MCP control of UI and cluster, built-in agent, updates, cache transparency, crash recovery | checklist, not counted in parity numbers |

Two generated pages exist in PhotoCraft and are part of this method, but have no generator here
yet: `parity-checklist.md` (from `make parity`, every Lens view, action and interface row, live or
missing) and `scorecard.md` (from `make scorecard`, measured checklists and performance and API-load
budgets). Until their generators exist, the checklists in the hand-written pages are the only
source. The generators are part of milestone M0 in [`roadmap.md`](roadmap.md).

## Rules for every progress page

1. **Status line.** Each page starts with one quoted line: `Last reviewed` (someone checked the
   page against the code), `Last updated` (the content changed), `Change` (`major`, `minor` or
   `trivial`, with a few words on what changed) and `Target` (the Lens version measured against).
2. **Status vocabulary.** A checklist row is **done** (behaves as in Lens, and a test proves it),
   **partial** (exists, but differs from Lens or is untested) or **missing**. "Exists" is not
   "done": a row without a test is partial at best.
3. **Evidence.** Every non-missing row names its evidence: a file and line, a test name, a
   scorecard id or an issue number. A row without evidence is missing.
4. **Sources.** A row about Lens comes from Lens Desktop's documentation or release notes, or from
   a capture of the running Lens Desktop, never from OpenLens or Freelens
   ([D-12](../developer/decisions.md#d-12-parity-sources)). What the documentation leaves open is
   marked *capture needed*.
5. **Measured or estimated.** Every number says which it is. A measured number names how it was
   measured, so the next person can repeat it. Move an estimate only with new evidence.
6. **Hours.** Estimates are agent-hours: one Opus 5.5 agent working sequentially, including tests.
   Cogpit has no history to calibrate against yet; until it has, estimates use PhotoCraft's
   calibration (2–9 h per checklist item of UI or tool depth, 8–20 h per subsystem, from its 1,004
   merged PRs) and say so. Replace the borrowed calibration with Cogpit's own once ~100 PRs have
   merged.
7. **Revision history.** Each page ends with a table of dated changes. Add a row whenever the
   status line's `Last updated` moves.
8. **Same change.** Work that moves a number or closes a gap updates, in the same pull request,
   the checklist row, the gap entry in [`gaps.md`](gaps.md), the number in
   [`target-app-parity.md`](target-app-parity.md) and, when a headline moves,
   [`ROADMAP.md`](../../ROADMAP.md).
9. **Stable headings.** Numbers go in the text under a heading, never in the heading itself, so
   links to a section survive every re-measurement.

## Stages

| Stage | Meaning |
|---|---|
| **pre-alpha** | The [alpha gate](roadmap.md#alpha-gate) fails: at least one core workflow does not complete end to end |
| **alpha** | Every core workflow completes end to end on macOS; ready for real work is in the ~40–75% band |
| **beta** | Ready for real work ≥ ~75%, and no gap marked *blocks beta* in [`gaps.md`](gaps.md) |
| **1.0** | Ready for real work ≥ ~95% for the mainstream audience on macOS, Windows and Linux, signed releases with one-click updates |

## The two headline numbers

- **Feature breadth:** does each Lens feature exist in some form? Partly measurable (resource
  views, actions, menu items, shortcuts, preferences), weighted in
  [`target-app-parity.md`](target-app-parity.md#feature-breadth).
- **Ready for real work:** could someone who uses Lens every day switch to Cogpit? Estimated,
  weighted by dimension, with evidence per row. This is the number that decides the stage.

Breadth always runs ahead of readiness. PhotoCraft learnt this the expensive way: its menu
breadth reached 100% while readiness was ~45%, and its first users found broken basics that all
counted as "live" in the menu checklist. Never quote breadth as progress on its own.

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | minor | Beyond-Lens page; source rule (D-12); generators through make; stages macOS first |
| 2026-10-11 | major | First version: method, page map, rules and stages adopted from PhotoCraft |
