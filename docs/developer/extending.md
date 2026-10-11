# Extending Cogpit

Ordered checklists for the common additions. The packages they name are the planned layout in
[`architecture.md`](architecture.md#2-repository-layout-planned); none exists yet (2026-10-11).

## Adding a command

1. **Find the action in Lens.** Look it up in [`resource-parity.md`](../parity/resource-parity.md)
   or [`ui-parity.md`](../parity/ui-parity.md) and use the id given there. If the row has none,
   name it `<kind-or-area>.<action>` (`statefulset.restart`, `cluster.disconnect`) and add the id to
   the row in the same change.
2. **Logic** goes in the lowest package that fits: row and status rules in `model`, Kubernetes
   calls in `cluster`, Helm in `helm`, using the kubectl libraries where `kubectl` already has the
   behaviour (`k8s.io/kubectl/pkg/drain`, rollout). Unit-test it there.
3. **Command** goes in `internal/engine/commands/<area>.go` and registers:
   - `id`, `label`, `menu path`, the kinds it applies to and Lens's default shortcut, if any;
   - a parameter struct with `json` and `jsonschema` tags (the schema agents read);
   - `enabled`, returning the reason when disabled;
   - `verbs` (every API verb and resource it uses) and `danger` (`Read`, `UI`, `Mutating`,
     `Destructive`);
   - `run`, returning a JSON-serializable result. Reads come from the cache.
4. **Multi-step mutations** (several API calls in a row) are crash-safe
   ([conventions](conventions.md#crash-tolerant-crash-safe)): a worker process, or detection of the
   half-finished state on the next start.
5. **Respect the context:** cluster context and namespace, a multi-selection (a row inside the
   selection acts on the whole selection), the user's permissions, and the origin's confirmation
   rules.
6. **Tests:** behaviour against fake clients, the requests sent, disabled states with their
   reasons, bad params returning errors, confirmation for each origin, and the fuzz target.
7. **UI and MCP** follow from the registry: menus, context menus, the palette and the MCP tool
   appear without UI code. A command with parameters gets a schema-generated dialog; add a custom
   dialog only when the schema can't express it.
8. **Progress docs:** flip the row in the parity page and update [`gaps.md`](../parity/gaps.md).

## Adding an integration (what Lens calls an extension)

Integrations are built into Cogpit's code ([D-5](decisions.md#d-5-extensions)); there is no
mechanism for external extensions. An integration such as Flux CD or Argo CD is a set of kind
profiles (below), its commands (above) and, where Lens has one, a dashboard view, in its own
package under `internal/`, with its own layer and `depguard` rule.

## Adding a built-in kind profile

A kind without a profile already gets a generic view from discovery; a profile makes it match Lens.

1. Take the columns, status rules and actions from the kind's row in
   [`resource-parity.md`](../parity/resource-parity.md).
2. Add `internal/model/kinds/<kind>.go`: columns (title, value, sort key, default visibility), the
   status token rule, searchable fields for agents (`clusterIP`, `podIP`, `node`), and the detail
   sections.
3. Decide what the cache needs: full objects (`dynamicinformer`) or metadata only
   (`metadatainformer`).
4. Golden tests from recorded objects in `internal/model/kinds/testdata/<kind>/`: healthy,
   failing, terminating, and an object with optional fields missing.
5. Register the profile with its group, version and resource. Discovery decides at runtime whether
   the cluster serves it.
6. Add its actions as commands (above).

## Adding a cache consumer

1. Acquire a handle for the (cluster, kind, namespace scope) you show; release it when the view,
   drawer or query ends. The cache keeps the informer warm for the grace period after the last
   release.
2. Read snapshots; never start an informer, a watch or a poll yourself.
3. Add a test with the counting round-tripper that asserts the API requests your consumer costs
   (budgets P9–P11 in [`testing.md`](testing.md#performance-budgets-proposed-not-measured)).

## Adding a package

1. Pick its layer in [`architecture.md`](architecture.md#3-dependency-layering-enforced) and add its
   `depguard` rule to `.golangci.yml` in the same change.
2. Give it a package doc comment.
3. Add a row to the package table in [`README.md`](README.md).
