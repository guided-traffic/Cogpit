# Extending Cogpit

Ordered checklists for the common additions. The packages they name are in
[`architecture.md`](architecture.md#2-repository-layout-planned); which of them exist is in its
[Current state](architecture.md#current-state-2026-10-13). Worked examples: the commands in
[`internal/engine/commands`](../../internal/engine/commands/) and the kind profiles in
[`internal/model/kinds`](../../internal/model/kinds/).

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
   - `Order`, its place in the kind's row menu and drawer ([architecture §4.2](architecture.md#42-the-command-registry)),
     with `DrawerOnly` when Lens shows it in the drawer but not in the row menu (Copy);
     `Enabled` wraps `engine.ErrNotApplicable` when the command does not apply to the object at all
     (Cordon on a cordoned node), so that the menu hides it; `Escalate` when some invocations of a
     mutating command are destructive; `AgentsAsk` when some invocations by an agent must wait for
     the user even where agent writes are allowed (a role binding: a command that can write one
     declares it, [H-27](../security/privilege-footprint.md#h-27-a-binding-is-judged-by-its-roles-name-and-the-cache));
     `Prefill` when a dialog should start from cached values;
   - a parameter struct with `json` and `jsonschema` tags (the schema agents read);
   - `enabled`, returning the reason when disabled;
   - `verbs` (every API verb and resource it uses) and `danger` (`Read`, `UI`, `Mutating`,
     `Destructive`). When the parameters name an object (`ref`), the registry asks the access
     reviews about each verb and greys the command "not permitted" on a denial, so declare the
     verb its purpose needs (Edit declares `update`, which its tab ends in);
   - `run`, returning a JSON-serializable result. Reads come from the cache.
   Cluster writes go through an interface the backend's writer implements: `*cluster.Conn` in
   the app, a fake in tests. Each area declares its own interface in its own engine file (as
   `engine.Actions` does for the MVP's actions) and reaches it with a type assertion on
   `Backend.Writer()`, so that two areas never edit the same interface; the test fake gets the
   area's methods in the area's own test file.
4. **Multi-step mutations** (several API calls in a row) are crash-safe
   ([conventions](conventions.md#crash-tolerant-crash-safe)): a worker process, or detection of the
   half-finished state on the next start.
5. **Respect the context:** cluster context and namespace, a multi-selection (a row inside the
   selection acts on the whole selection), the user's permissions, and the origin's confirmation
   rules.
6. **Tests:** behaviour against fake clients, the requests sent, disabled states with their
   reasons, bad params returning errors, confirmation for each origin, and the fuzz target. Add
   the command's id and danger to `wantCommands` in
   [`commands_test.go`](../../internal/engine/commands/commands_test.go).
7. **UI and MCP** follow from the registry: menus, context menus, the palette and the MCP tool
   appear without UI code. With an `Order`, the command is in its kinds' row menu and drawer
   toolbar ([architecture §4.2](architecture.md#42-the-command-registry)); a `container`
   parameter makes it a submenu of the pod's containers; required parameters beyond `ref` and
   `container` get a schema-generated dialog (integer, number, string and boolean fields, the
   `jsonschema` tag as the hint), started from `Prefill`; a command with a `Prefill` gets the
   dialog for its optional parameters too. Add a custom dialog only when the schema
   can't express it. Give the command an icon in the drawer's toolbar in
   [`cmdicons.go`](../../internal/ui/fyneui/cmdicons.go); others get a generic one. The command
   palette lists an object action with its object, and any other command with a `MenuPath`
   ([rules](architecture.md#42-the-command-registry)); one with a required parameter beyond its
   object (a context, a name, a file) is listed only where
   [`paletteRunners`](../../internal/ui/fyneui/palette.go) opens a dialog for it. A `Shortcut` such as
   `Cmd+Shift+P` (Cmd is Ctrl on Windows and Linux) is shown in the palette; check it against
   `TestRegistryShortcutsDoNotCollide` and bind it where the window binds the others.
8. **Command table:** regenerate the table in [`commands.md`](commands.md) with
   `go run ./hack/readmecmds` and replace the content of its details block with the output (do not
   edit the table by hand).
9. **Progress docs:** flip the row in the parity page and update [`gaps.md`](../parity/gaps.md);
   `make parity` checks the row and the numbers it feeds.

## Adding an integration (what Lens calls an extension)

Integrations are built into Cogpit's code ([D-5](decisions.md#d-5-extensions)); there is no
mechanism for external extensions. An integration such as Flux CD or Argo CD is a set of kind
profiles (below), its commands (above) and, where Lens has one, a dashboard view. The plan was a
package of its own under `internal/`; Flux CD, the first built (2026-10-13, G23), showed that the
existing layers hold it without one, each part in its own files of the package whose layer it
belongs to:

1. **Curated profiles** of the integration's CRD kinds in `internal/model/kinds/<integration>*.go`,
   appended to `curated` by the file's `init` (as [`flux.go`](../../internal/model/kinds/flux.go)
   does), keyed `<plural>.<group>`, at the version the integration serves with older ones as
   `FallbackVersions`; `kinds.ProfileFor` gives a definition at another version the generic
   profile. Goldens per kind in `testdata/<key>/`, run by `TestCuratedGolden` and
   `TestCuratedTypeConfusion`.
2. **Commands** in `internal/engine/commands/<integration>.go` with `Kinds` set to the curated
   keys, writes through an interface of the engine that `*cluster.Conn` implements
   ([`engine/flux.go`](../../internal/engine/flux.go), [`cluster/flux.go`](../../internal/cluster/flux.go)),
   bound to the object's UID; a `refs` variant for selections.
3. **A dashboard** as a page: a key `<group>/<page>` with its title in `pageTitles`, a worker in
   `internal/engine/<integration>dashboard.go` hooked into `state` (views.go) as the Overviews are,
   its read command, and its panel in `internal/ui/fyneui` (`tableview.go`'s page layout).
4. **The navigator group** in `internal/ui/fyneui/<integration>nav.go`, one line in `navLayout`;
   a group of pages alone is left out, so the section goes with the integration's CRDs.

## Adding a built-in kind profile

A kind without a profile already gets a generic view from discovery; a profile makes it match Lens.

1. Take the columns, status rules and actions from the kind's row in
   [`resource-parity.md`](../parity/resource-parity.md).
2. Add `internal/model/kinds/<kind>.go`: columns (title, value, sort key, default visibility), the
   status token rule, searchable fields for agents (`clusterIP`, `podIP`, `node`), and the detail
   sections. The drawer starts with Lens's Properties section, which the engine fills with the
   metadata; a first section titled `model.TitleProperties` adds the kind's own properties to it.
   Items that name objects carry a `Link`, lists of short values `Chips`, long lists a `Fold`
   noun ([`model/detail.go`](../../internal/model/detail.go)).
3. Decide what the cache needs ([architecture § Scopes](architecture.md#scopes-local-and-virtual-kinds)):
   full objects, or metadata only (`MetadataOnly`, when no column or drawer item needs more), and a
   `LabelSelector` if the kind is a labelled subset (it then reads from the unselected scope of the
   same resource while that runs: covering scopes, as Helm's releases read from the Secrets). A column or drawer section that shows objects of another kind
   declares the index it reads in `Needs` (add the index to
   [`kinds/related.go`](../../internal/model/kinds/related.go) if it is new) and reads it in
   `CellsWith` or `DetailWith` ([architecture §4.4](architecture.md#44-tables-details-editors-and-streams)).
   Objects Cogpit makes itself are a local kind (`Local`, listed in `kinds.Local()`, written with
   `Backend.PutLocal`); objects made from other cached objects are a virtual kind (`Derive`). A
   virtual kind made of several kinds' objects cannot derive from one scope: it gets a cache of its
   own over `cache.Held`, filled while it is read, as the Applications are
   ([architecture § Scopes](architecture.md#scopes-local-and-virtual-kinds),
   [`engine/applications.go`](../../internal/engine/applications.go)).
4. Golden tests from recorded objects in `internal/model/kinds/testdata/<kind>/`: healthy,
   failing, terminating, and an object with optional fields missing (`minimal`). `TestGolden`
   and `TestTestdataComplete` pick them up; run `go test ./internal/model/kinds/ -update` and read
   the golden diff. The kind's own `internal/model/kinds/<kind>_test.go` holds its fuzz target
   (`func FuzzCronJobs(f *testing.F) { fuzzProfile(f, "cronjobs") }`) and its status assertions
   (`checkStatusTexts(t, []statusText{...})`), so that adding a kind edits no shared test table.
5. Cells that name another object carry a link (`linkTo`; `namespaceCell` already links the
   namespace); a view with Lens's warning icon has `warningColumn` right after Name, filled with
   `warningCell`. Mark any rule that is not captured from Lens as inferred in the parity row.
6. Register the profile with its group, version and resource: one line in `builtin` in
   [`kinds.go`](../../internal/model/kinds/kinds.go). Discovery decides at runtime whether the
   cluster serves it; a kind that older clusters serve at another version of the same shape lists
   those in `FallbackVersions`, preferred first. Built-in kinds keep the bare plural as their key; kinds from custom resource
   definitions are `<plural>.<group>` ([architecture §4.5](architecture.md#45-kind-keys)).
   A curated profile of a kind from a custom resource definition (the Vertical Pod Autoscalers)
   keeps that key and goes in `curated` in
   [`verticalpodautoscalers.go`](../../internal/model/kinds/verticalpodautoscalers.go), not in
   `builtin`: the custom kinds watch takes it through `kinds.ProfileFor` while the cluster's
   definition serves its version; its goldens are `TestCuratedGolden`'s.
7. Add its actions as commands (above).

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
