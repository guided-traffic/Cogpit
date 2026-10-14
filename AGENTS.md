# AGENTS.md: guide for AI agents and contributors

Cogpit is an open-source, native Kubernetes IDE written in **Go**: a clean-room reimplementation
of the current Lens Desktop for macOS, Linux and Windows. The goal is Lens parity (the same
navigation, resource views, actions, shortcuts and workflows), a UI that stays fast because it
reads only from a cache, and a tool that agents can drive completely over MCP, the UI included.
The product name is always written **Cogpit** in user-facing text; machine names stay lowercase
(module `github.com/guided-traffic/cogpit`, binaries `cogpit` and `cogpit-cli`). Read this file
first, then `docs/`.

**State on 2026-10-14:** implementation has started (owner's go-ahead, 2026-10-11). A local MVP
runs on macOS, extended by nine further implementation runs (2026-10-11 to 2026-10-14, night runs
and day runs): 46 built-in views and the custom resource views from the cache, the cluster Overview,
the detail drawer and row menus, the hotbar, the navigator with its sections, a tab bar, the
command palette, Preferences and Cluster settings, a welcome page, YAML editing with drafts, the
create dialogs, row selection, Delete of a selection and copy icons in the lists, node and workload actions with a
crash-safe drain, Helm install from the repositories of Helm's own `repositories.yaml` (credentials included), upgrade, rollback and uninstall (Secrets and
ConfigMaps drivers), kubeconfig sources with approved credential plugins, metrics from the Metrics
Server and Prometheus together, logs, a terminal with a verified kubectl download, pod shells,
`cogpit-cli` with an MCP stdio bridge, Flux CD and Argo CD sections, a node shell, CI on every push, releases
with a signed, notarized disk image and Sparkle 2 (not live until the owner's update key), and MCP
control of all of it; it doubled
as the Fyne spike ([results](docs/developer/spike-fyne.md#results)).
What is built and where it differs from the plan: [architecture › Current state](docs/developer/architecture.md#current-state-2026-10-14). The documentation method, and parts of
this file, come from [PhotoCraft](https://github.com/storytold/photocraft)
([attribution](ATTRIBUTION.md)).

## 1. Orientation (5 minutes)

| Read | Why |
|---|---|
| [`ROADMAP.md`](ROADMAP.md) | One page: stage, headline numbers, every dimension, what's next |
| [`docs/developer/decisions.md`](docs/developer/decisions.md) | What is decided (Go, Fyne after the spike gate, MCP control, built-in agent, updates, crash stance, cache) and what is open |
| [`docs/developer/architecture.md`](docs/developer/architecture.md) | Packages, layering, the engine/UI seam, the cache, MCP and the agent |
| [`docs/parity/target-app-parity.md`](docs/parity/target-app-parity.md) | The authoritative assessment against Lens Desktop 2026.10.51501 |
| [`docs/parity/gaps.md`](docs/parity/gaps.md) | Every shortfall, ranked: the work list to pick from |
| [`docs/parity/roadmap.md`](docs/parity/roadmap.md) | Alpha gate, milestones, **current focus** |
| [`docs/parity/ui-parity.md`](docs/parity/ui-parity.md), [`resource-parity.md`](docs/parity/resource-parity.md), [`platform-parity.md`](docs/parity/platform-parity.md), [`beyond-lens.md`](docs/parity/beyond-lens.md) | The checklists |
| [`docs/parity/README.md`](docs/parity/README.md) | How the progress docs work and the rules for updating them |
| [`docs/developer/conventions.md`](docs/developer/conventions.md) | The rules in detail |
| [`docs/developer/extending.md`](docs/developer/extending.md) | Checklists: adding a command, a kind profile, a package, a cache consumer |
| [`docs/developer/spike-fyne.md`](docs/developer/spike-fyne.md) | The UI toolkit gate before M2 |
| [`docs/developer/usage.md`](docs/developer/usage.md) | Running Cogpit, flags, files it writes, naming, agents' set-up |
| [`docs/developer/commands.md`](docs/developer/commands.md) | Every command and MCP tool, generated |
| [`docs/security/`](docs/security/README.md) | Trust boundaries, credentials, privilege footprint, agents and MCP, updates |

## 2. Package map

```text
cmd/cogpit, cmd/cogpit-cli    L5 desktop app (Fyne, cgo); headless CLI (contexts, get) and MCP stdio bridge (pure Go)
internal/ui/fyneui            L4 the Fyne shell; the only package that imports Fyne
internal/ui/logo              L4 the logo, rasterised for the app icon and the header (pure Go)
internal/mcp agent update     L4 MCP server; built-in agent; update check and install
internal/app                  L4 the wiring both binaries share: Cogpit's directories, kubeconfig sources, audit log
internal/engine               L3 Session, command registry, UI state, jobs, events, confirmation, audit log
internal/helm metrics demo    L2 Helm SDK; metrics providers (not built yet); the synthetic demo cluster
internal/cluster cache        L1 connections and auth; shared informers (the cache)
internal/terminal             L1 PTY sessions
internal/kubectl              L1 kubectl for terminals: client version, skew, verified download (release host and sigstore's log only)
internal/locale               L1 the computer's date and time format (macOS region via cgo, else $LC_TIME), D-31
internal/model kubeconfig     L0 kind profiles and rows; kubeconfig parsing (no I/O)
internal/buildinfo            L0 the version both binaries report (release, else dev and the commit)
internal/testkit              fixtures, fake clients, kind and envtest helpers
```

**Layering is enforced** by golangci-lint's `depguard` in `make lint`: a package imports only
lower layers, only `internal/ui/...` imports Fyne, `model` and `kubeconfig` do no I/O, and only
`cluster`, `cache`, `terminal`, `helm` and `metrics` talk to the API server.

## 3. Golden rules

### Never surprise a cluster (outranks feature work)

Cogpit acts with the user's identity on clusters that may be production.

- **Read-only unless asked.** Browsing never writes.
- **Every mutating command declares its verbs and a danger level.** `Destructive` commands
  (delete, drain, evict, scale to zero, Helm uninstall and rollback, node shell) need a
  confirmation that names the objects, the namespace and the context. The engine enforces it for
  every origin: GUI, CLI, external agents and the built-in agent.
- **Agents follow D-2:** reads and UI operations run freely; mutating commands need the user's
  acceptance unless allowed for that context; destructive commands always need it; everything an
  agent does is in the audit log.
- **No implicit installs and no implicit kubeconfig writes.**
- The verb-by-verb contract is [privilege footprint](docs/security/privilege-footprint.md).

### Cache first

The UI and agents read only from the cache; only the cache layer lists and watches. Never call the
API server from a view, a command or an MCP tool for data the cache holds, and never poll what can
be watched. New data needs go into the cache layer ([architecture §5](docs/developer/architecture.md#5-cache)).

### Crash-tolerant UI, crash-safe cluster operations

A crash is a bug, not a catastrophe: the cluster keeps its state ([D-11](docs/developer/decisions.md#d-11-crash-stance)).
But multi-step operations (Helm install, upgrade and rollback; drain) must survive a crash or be
finishable after one, and every goroutine doing cluster I/O recovers from a panic at its boundary.
Run tests with the race detector.

### The other rules

1. **Everything is a command.** New user-visible behaviour is an engine command with an id, label,
   menu path, shortcut, parameter type, `enabled` (with a reason), `verbs`, `danger` and `run`, plus
   tests. It becomes an MCP tool automatically. Use the ids in
   [`resource-parity.md`](docs/parity/resource-parity.md) and
   [`ui-parity.md`](docs/parity/ui-parity.md).
2. **UI state is engine state.** The GUI changes view, filter, selection and drawer only through
   `ui.*` commands, so agents see and drive the same state.
3. **Discover, don't assume.** Kinds, versions, permissions and metrics providers come from the
   cluster at runtime. A missing kind, permission or provider is a state the UI shows, not an error.
4. **Clean-room against the current Lens Desktop.** Behaviour and look by observation and its
   documentation. No code, text or asset from Lens Desktop, OpenLens or Freelens
   ([D-12](docs/developer/decisions.md#d-12-parity-sources)).
5. **Tests are the gate** ([`testing.md`](docs/developer/testing.md)). Tests never read the
   developer's `~/.kube/config`; cluster tests create their own `kind` cluster or envtest
   control plane.
6. **The UI is thin and data-driven.** Colours come from theme tokens and status tokens.
7. **Verify UI changes visually** with an offscreen render (Fyne's test driver) or a screenshot,
   and attach before/after images to the pull request.
8. **Performance is a feature.** Lists at 10,000 rows, 1,000 watch events per second, and the API
   request budgets in [`testing.md`](docs/developer/testing.md#performance-budgets-proposed-not-measured).
9. **Secrets stay secret.** Tokens, keys, exec-plugin output and Secret values never reach logs,
   errors, crash reports or an LLM endpoint ([credentials](docs/security/credentials.md),
   [agents and MCP](docs/security/agents-and-mcp.md)).

## 4. Picking work

Priorities: infrastructure first, then the alpha-gate workflows, then breadth, then the long tail.

0. **Read [`ROADMAP.md`](ROADMAP.md), then [`docs/parity/gaps.md`](docs/parity/gaps.md).** When
   your work moves a number or closes a gap, update the checklist row, the gap entry and the
   numbers they feed, with the date, in the same pull request
   ([rules](docs/parity/README.md#rules-for-every-progress-page)).
1. [`docs/parity/roadmap.md`](docs/parity/roadmap.md) → **Current focus**.
2. Once the scorecard exists: its `missing` and `partial` rows and the budgets not met yet.
3. `log/devlog.md` (local, gitignored) → the "Still open" bullets of recent entries.

## 5. Before you finish a task

Locally, keep it fast (owner, 2026-10-12): CI is the full gate, so check here only what the change
touches.

```sh
go build ./...                          # the tree builds
go vet ./internal/<pkg>/...             # the packages you touched
go test ./internal/<pkg>/...            # their tests, without -race
go run ./hack/parity -rows              # only when a checklist or a progress page changed
go run ./hack/readmecmds                # only when commands changed: the table in docs/developer/commands.md
# -race on the one package only when the change is about concurrency (cache, journals, terminal)
```

CI ([`release.yml`](.github/workflows/release.yml), one job per check, none longer than ten
minutes: [releasing › CI](docs/developer/releasing.md#ci)) runs on every push what used to run
here, each with its own make target: lint (depguard layering included), parity, GoSec,
govulncheck, cyclomatic complexity, the unit tests, the UI's race tests in eight shards and the
other packages' race tests, the kind and drain cluster tests, the macOS-only tests and the app, the
Windows compile, and the combined coverage. Read its result when you next push (`gh run list`, then
`gh run view <id> --log-failed`) and fix what is red first. Once every job on `main` is green,
semantic-release publishes a release for the `feat:` and `fix:` commits since the last one and
pushes a release commit to `main`, so `git pull --rebase` before your next push.

Then commit and push. Agents may commit and push to `main` on their own (owner, 2026-10-11), in
[Conventional Commits](docs/developer/conventions.md#commits-and-pull-requests) form and **without
`Co-Authored-By` trailers or "Generated with" lines**, and **never with `!` or a `BREAKING CHANGE:`
footer**: releases continue from the owner's anchor tag `v0.0.0`, and 1.0.0 is the owner's call
(until then `.releaserc.json` turns a breaking change into a minor release). Don't wait for CI before the next step (owner, 2026-10-11): run the
quick local checks, push, and look at CI when you next push. Append a terse entry to `log/devlog.md`:
what landed, numbers, what's still open. Keep the tree building at every step.

## 6. Parallel agents

- Edit only the files you own. Shared files (the command list, the UI's menu wiring, package
  registries) get small, surgical edits; re-read before editing.
- Put new commands in a **new file** (`internal/engine/commands/<area>.go`) and new kind profiles in
  their own file (`internal/model/kinds/<kind>.go`) rather than growing a shared file.
- If someone else's in-progress edit breaks the build, wait and retry; don't fix their files.
- `go.mod` and `go.sum` change through `go get` and `go mod tidy` only, by one agent at a time.

## 7. Where things are tracked

- [`ROADMAP.md`](ROADMAP.md) and [`docs/parity/`](docs/parity/README.md): where Cogpit stands
  against Lens, what's next, the work list.
- [`docs/developer/decisions.md`](docs/developer/decisions.md): decisions, open and taken.
- [`docs/security/`](docs/security/README.md): the security architecture and its open gaps
  (`H-<n>`).
- `plan/` and `log/` (local, gitignored): research notes and the dev log.
