# Conventions

The rules every change follows. [`AGENTS.md`](../../AGENTS.md) states them in short; this page has
the detail and the reasons.

## Language, module and licence

- **Go** ([D-7](decisions.md#d-7-language-and-platforms)), module
  `github.com/guided-traffic/cogpit`, the Go version the team's other repositories use (their
  `go.mod` files declare `go 1.27.2` on 2026-10-11).
- **No JavaScript, TypeScript or webview** in the product. The UI is Fyne
  ([D-1](decisions.md#d-1-ui-toolkit)); only if the spike gate fails does a separate Rust/egui UI
  process join, and then only behind the engine's protocol.
- **Licence:** Apache-2.0 ([`LICENSE`](../../LICENSE)). Dependencies must be compatible
  (Apache-2.0, MIT, BSD, ISC, MPL-2.0 as a library).

## Clean-room

The target is the current Lens Desktop ([D-12](decisions.md#d-12-parity-sources)), proprietary
software of Mirantis. We match its behaviour and look by observation and its public
documentation.

- **Never use** code, text, icons, logos, illustrations, fonts or other assets of Lens Desktop,
  OpenLens or Freelens, even where they ship as readable files or are open source.
- **Branding.** Lens is Mirantis's product and brand. User-facing text never presents Cogpit as
  Lens, as made by Mirantis, or as endorsed by it. Comparisons in documentation ("matches Lens's
  Pods view") are fine.
- **Third-party assets** must be permissively licensed, keep their licence file next to them, and
  get a row in [`ATTRIBUTION.md`](../../ATTRIBUTION.md) (path, title, author, source, licence) in
  the same change.

## Never surprise a cluster

- **Read-only unless asked.** Opening a view, hovering or selecting never writes to a cluster.
- **Declare verbs.** Every mutating command lists the API verbs and resources it uses.
- **Confirm destructive actions** (delete, drain, evict, scale to zero, Helm uninstall and
  rollback, node shell) with a dialog naming the objects, the namespace and the context. The engine
  enforces it, for the CLI and agents too.
- **Agent origin** follows [D-2](decisions.md#d-2-agent-interface-scope): mutating commands need the
  user's acceptance unless allowed for that context, destructive ones always; all of it is logged.
- **No implicit installs, no implicit kubeconfig writes.** A node shell, which creates a privileged
  pod, says so before it runs ([privilege footprint](../security/privilege-footprint.md)).

## Cache first

- Views, commands and MCP tools read from the cache ([architecture §5](architecture.md#5-cache)).
  Only `cluster`, `cache`, `terminal`, `helm` and `metrics` talk to the API server.
- Acquire a cache handle for what you show, release it when you stop showing it. Never start your
  own informer or watch.
- Never poll a resource that can be watched. Polled data (metrics, chart indexes) is polled only
  while visible.
- A new data need is a change to the cache layer, with a test that counts the API requests it costs.

## Crash-tolerant, crash-safe

[D-11](decisions.md#d-11-crash-stance): a crash is a bug, not a catastrophe; a half-finished
cluster operation is worse than a crash.

- **Errors, not panics.** Return errors and wrap them with `%w`; `errcheck` is on. Do not call
  `panic` in non-test code; a programming error found at start-up (duplicate registry ids) may stop
  start-up with an error message.
- **Recover at goroutine boundaries.** Every goroutine that does cluster I/O starts through a helper
  that recovers a panic, logs it with its stack, and reports it to the engine as an error. One
  failing watch must not close every cluster.
- **Unstructured data is hostile input.** Read it through the `model` helpers, which never panic on
  unexpected shapes; check bounds; cap buffers sized by input (log lines, YAML documents, terminal
  scrollback).
- **No data races.** `go test -race` runs in CI; maps shared between goroutines are guarded,
  because a concurrent map write ends the process and cannot be recovered.
- **Multi-step mutations are crash-safe:** they run in a worker process that survives the UI, or
  the next start detects the half-finished state and offers to finish or revert it.
- **Unsaved editor content** is kept as a local draft.

## Secrets stay secret

- Secret values, bearer tokens, client keys, exec-plugin output and LLM API keys never appear in
  logs, errors, crash reports or telemetry. Credential types implement `slog.LogValuer` and
  `fmt.Stringer` as `"<redacted>"`.
- Secret data in the UI is hidden until the user reveals one field.
- Nothing from a Secret's data is sent to an LLM endpoint unless the user explicitly allows it for
  that request ([agents and MCP](../security/agents-and-mcp.md)).

## Commands, not handlers

New user-visible behaviour is an engine command with tests; the UI and MCP call it. Checklist:
[`extending.md`](extending.md#adding-a-command).

## Tests

Every change comes with tests ([`testing.md`](testing.md)). Kind profiles have golden tests;
anything that parses input (kubeconfig, YAML, terminal output, MCP and LLM messages) has
malformed-input tests or fuzz targets.

## Style

- `gofmt` and `goimports`; golangci-lint with the team's linter set (errcheck, govet, ineffassign,
  staticcheck, unused, misspell, unconvert, unparam, goconst, prealloc, revive) plus `depguard` for
  the layering.
- Doc comments on every exported identifier (revive's `exported` rule). Comments explain *why*.
- Names: packages short and lowercase (`cache`, `engine`); command ids `<kind-or-area>.<action>`
  in lowerCamelCase segments (`deployment.restart`, `helm.release.rollback`), their MCP tool names
  with underscores (`helm_release_rollback`); environment variables `COGPIT_<NAME>`.
- Logging with `log/slog`.
- User-facing text says **Cogpit**. Machine names stay lowercase.

## UI

- Colours, sizes and paddings come from Cogpit's Fyne theme, never hard-coded in a widget.
- Status colours come from the row's status token, not from the view.
- Widget updates from other goroutines go through `fyne.Do`.
- Verify every visual change with an offscreen render (Fyne's `test` package) or a screenshot, in
  dark and light themes, and attach before/after images to the pull request.

## Commits and pull requests

Small, focused commits in the [Conventional Commits](https://www.conventionalcommits.org/) form
the team's other repositories use (`feat:`, `fix:`, `chore:`, `test:`, `docs:`, with a scope where
it helps: `fix(cache): …`), because semantic-release derives the next version from them
([releasing](releasing.md)). Commits and pull requests carry no `Co-Authored-By` trailers and no
"Generated with" lines (owner, 2026-10-11). A pull request that moves a parity number updates the progress docs in
the same change ([rules](../parity/README.md#rules-for-every-progress-page)).
