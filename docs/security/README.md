# Security architecture

Cogpit's security architecture, one page per perspective. No page tries to cover everything: a
document nobody finishes is a document nobody checks. The file names are the index.

| Page | Perspective |
|---|---|
| [`trust-boundaries.md`](trust-boundaries.md) | Who and what Cogpit trusts, and where untrusted data crosses into trusted state |
| [`credentials.md`](credentials.md) | Where cluster credentials and Cogpit's own secrets come from, where they live, and where they must never end up |
| [`privilege-footprint.md`](privilege-footprint.md) | What Cogpit does in a cluster with the user's identity, verb by verb |
| [`agents-and-mcp.md`](agents-and-mcp.md) | Who may drive Cogpit besides the user: external agents over MCP, the built-in agent, the LLM endpoint |
| [`updates.md`](updates.md) | The update channel: how Cogpit replaces itself, and the chain of trust behind it |

## State of this documentation

These pages describe the boundaries that exist in the problem itself (kubeconfig, the Kubernetes
API, credential plugins, terminals, agents, updates) and the check each boundary needs. Since the
MVP (2026-10-11) the mechanisms it implements link to their code and tests; the rest names planned
packages. When a mechanism lands, its row gains a link to the code and the test that proves it, in
the same change. The owner's decision round of 2026-10-14 evening is recorded on every page whose
mechanisms it changes, in the form below; none of the changes it decided is built yet.

## Form of every page

- One perspective per page, stated in its first paragraph.
- Facts about Kubernetes, kubectl or Lens cite their source. Facts about Cogpit link to code.
- Every page ends with `## What this does not cover`. An open gap lives there, in the page whose
  mechanism has it, under a heading with a stable identifier `H-<n>`. Identifiers are never reused
  and never collected in a central list.
- An owner's decision that is not built yet is written where the mechanism it changes is
  described: "Decided (owner, <date>): <what>. Not built yet; until then <today's behaviour>, live
  or dormant." The gap stays open until the change is built. A default the owner confirmed reads
  "Confirmed by the owner (<date>): <what stays>", followed by what remains. Decided and built are
  never merged into one sentence.
- No tickets, checkbox lists or schedules on these pages. Work items belong in
  [`../parity/gaps.md`](../parity/gaps.md).

Reporting a vulnerability: [`SECURITY.md`](../../SECURITY.md).

## What this does not cover

- **Isolation between clusters:** Cogpit is a single-user desktop app with no server side; the
  isolation that matters is between the clusters open in one process. An action, an editor apply,
  a terminal or an agent's tool call must reach only the cluster whose view started it. The MVP has
  one connection at a time; a write confirmed for one context refuses to run after the user
  switched to another (`CheckConfirmedContext` in [`confirm.go`](../../internal/engine/confirm.go)),
  and each terminal is bound to its context by its own kubeconfig. No page covers this yet; it is
  due when several clusters can be open at once
  ([G4](../parity/gaps.md#g4-engine-and-command-registry)). The owner asked on 2026-10-14 evening
  for a hotbar tile dragged out of the window to open another Cogpit window with that cluster; it
  is designed (2026-10-15, [D-52](../developer/decisions.md#d-52-several-windows), postponed by
  the owner until the rest is done: one UI state per window, which window agents' `ui_*` tools drive, which window
  a confirmation belongs to), and building it is where this page becomes due.
