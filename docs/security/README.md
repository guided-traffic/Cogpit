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

No code exists yet (2026-10-11). These pages therefore describe the boundaries that exist in the
problem itself (kubeconfig, the Kubernetes API, credential plugins, terminals, agents, updates) and
the check each boundary needs. They name planned packages, not code. When a mechanism lands, its
row gains a link to the code and the test that proves it, in the same change.

## Form of every page

- One perspective per page, stated in its first paragraph.
- Facts about Kubernetes, kubectl or Lens cite their source. Facts about Cogpit link to code.
- Every page ends with `## What this does not cover`. An open gap lives there, in the page whose
  mechanism has it, under a heading with a stable identifier `H-<n>`. Identifiers are never reused
  and never collected in a central list.
- No tickets, checkbox lists or schedules on these pages. Work items belong in
  [`../parity/gaps.md`](../parity/gaps.md).

Reporting a vulnerability: [`SECURITY.md`](../../SECURITY.md).

## What this does not cover

- **Isolation between clusters:** Cogpit is a single-user desktop app with no server side; the
  isolation that matters is between the clusters open in one process. An action, an editor apply,
  a terminal or an agent's tool call must reach only the cluster whose view started it. No page
  covers this yet; it is written together with the engine's command dispatch
  ([G4](../parity/gaps.md#g4-engine-and-command-registry)).
