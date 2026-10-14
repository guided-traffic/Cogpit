# Using Cogpit: running, flags, files and agents

The reference that used to be in the root [README](../../README.md), moved here on 2026-10-14
(the owner's call: the README is the showcase page). For whoever builds and runs Cogpit from
source, sets up an agent, or needs the exact flags, preferences, files and names; contributors
start at the [developer documentation](README.md). Every command and MCP tool is listed in
[`commands.md`](commands.md).

## What Cogpit is and its status

Cogpit is a native Kubernetes IDE for the desktop, written in Go: an open-source, clean-room
reimplementation of the current [Lens Desktop](https://lenshq.io/). It runs on your Mac first,
later on Windows and Linux. It reads your kubeconfig files, connects to your clusters with your
own credentials, and gives you the views Lens users know: workloads, logs, shells, YAML editing,
port forwarding and the Helm releases in your clusters. Every view reads from a cache that watches
keep current, so it is fast for you and light on the API server. And agents can drive all of it
over MCP, the UI included. Cogpit is a desktop client: it is never a web UI and is never deployed
into a cluster. **Status: pre-alpha** (updated 2026-10-14): every view and every action on Lens's
list exists at least in part (64 of 64 views, 39 of 39 actions, counted by `make parity`), with
the Flux CD and Argo CD sections, Helm releases and charts, metrics, logs, a terminal, pod and node
shells and MCP control of all of it; none of it has been walked through by a person on a real
cluster yet. The releases are builds for Apple silicon, signed and notarized from the first release
after v0.4.0 on ([Download a release](#download-a-release),
[Run the MVP](#run-the-mvp)); where Cogpit stands: [roadmap](../../ROADMAP.md).

How data flows from the kubeconfig files to the views and agents:
[developer documentation › Data flow](README.md#data-flow).

## Features in detail

What Cogpit does and is meant to do, feature by feature. Everything below exists at least in part
unless it is marked *(planned)*; how far each one goes is measured row by row in
[docs/parity](../parity/README.md) and summed up in [ROADMAP.md](../../ROADMAP.md).

- 🖥️ A native macOS app first; Windows and Linux (Ubuntu first) after the macOS beta *(planned)*.
- 🗂️ Every cluster from your kubeconfig files and synced folders, side by side.
- 🧭 Lens's navigator levels: Local Kubeconfigs lists every cluster and the tabs of all of them; › drills into one cluster's resources and its tabs, ‹ goes back up.
- 📌 A hotbar of pinned clusters at the window's edge, in banks of twelve: F1–F12 to switch cluster, ⇧F1–⇧F12 to switch bank.
- ⌨️ A command palette on ⌘⇧P (Ctrl+Shift+P): every action the menus offer, Switch to any view, the hotbar's clusters and banks; ⌘P to search a cluster by name.
- 👋 A welcome page whenever no cluster is open (and from Help › Welcome): your clusters one click away, Add Kubeconfigs, the documentation. Disconnect a cluster, or remove it from Cogpit, which unlinks the kubeconfig file you added and never touches the file.
- ⚙️ Preferences on ⌘, and a cluster's Settings on ⌘⇧,: the theme switched live, the time zone, the editor's and the terminal's settings, a cluster's name and icon (a colour or your image), its terminals' folder and namespace; never written to a kubeconfig.
- 🕐 Every timestamp in your time zone and your region's format, as your Mac's region settings write it ("10.09.2026 17:41:01" in Germany): the drawer, events, logs, notifications, charts.
- 🔐 Your credentials stay yours: kubectl's own authentication code (certificates, tokens, exec plugins for EKS, GKE, AKS); no local proxy, no login.
- ⚡ Cache first: views read from a watch-fed cache; switching back to a view costs no API request.
- 📋 Lens's resource views: workloads, config, network, storage, access control, device resources (with each node's devices in its drawer), custom resources; each searchable by words, with match case or a regular expression, and filtered by the states its rows carry (Running, Pending, Not Ready, Degraded…), several at once, cleared with one click.
- 🔏 The certificates in your Secrets decoded in the drawer, PEM or DER, a section per certificate of a chain: subject, issuer, serial, validity (a warning 30 days before it ends), names, key, usages, fingerprint, each field copyable; whether `tls.key` belongs to `tls.crt` and the chain is in order; agents read the same fields, never the PEM text or the private key.
- 📜 Logs with follow, filter, timestamps, the previous container, pod and container selectors, older lines and a jump to a time.
- 💻 Pod shells, attach and a terminal bound to the cluster's context; copy and paste by keys or, if you choose, the right mouse button.
- 🖥️ A node shell, as Lens opens it: a privileged pod in `kube-system` from an image pinned by digest, confirmed every time, deleted when the shell closes.
- 🔐 Create Secret from a dialog, values masked as you type, for you alone: agents can neither create a Secret nor read its values.
- 🧰 The terminal's `kubectl` checked against the cluster: more than one minor version off, or none, and one click downloads the matching release from `dl.k8s.io`, verified against its SHA-256 and the Kubernetes release's signature, for Cogpit's terminals only.
- ✏️ YAML editing with a diff and a server-side dry-run before every apply.
- 🔌 Port forwarding, one click to the browser.
- ⎈ The Helm releases in your clusters (Lens's Helm view), kept in Secrets or ConfigMaps: history, values, upgrade, rollback, through the Helm SDK, no `helm` binary needed.
- 📦 Charts from the repositories in Helm's own `repositories.yaml` (add them with `helm repo add`; Cogpit picks a change up within seconds, logins and CA files included, and never writes the file), installed from a Helm Install tab; a chart's drawer shows the README of the version you pick, and charts show their icons (switch them off in Preferences).
- 🔁 Flux CD: its section in the navigator (Helm releases, Kustomizations, sources, image automation, notifications) with each object's state, revision and message, a dashboard of what is not ready whose states open the kind's list filtered to them, and Reconcile, Suspend and Resume, on a selection too.
- 📈 CPU and memory from the cluster's metrics-server and its Prometheus: columns in the Pods and Nodes lists, charts with requests and limits in the pod and node drawers, and the cluster Overview: CPU, Memory and Pods of the master or worker nodes over a time range, with the Issues (pods, nodes and workloads in trouble, warning events) below.
- 🛡️ Never surprises a cluster: destructive actions confirm, nothing is installed or written implicitly.
- 🤖 Fully controllable over MCP, the UI included: an agent sees what you see and navigates for you.
- ⌨️ `cogpit-cli`, inside the app: your contexts and any kind's objects on the command line, read-only, and an MCP stdio bridge you register once.
- 💬 *(planned)* A built-in agent in a chat beside your views, on any OpenAI-compatible endpoint such as LiteLLM or Open WebUI.
- 📦 Installs the Mac way: open the disk image, drag Cogpit into Applications, signed and notarized (from the first release after v0.4.0 on); *(planned)* a Homebrew cask for shell users.
- ⬆️ Updates with one click in the macOS app (Sparkle), verified against a signed update feed: built, and active from the first release that carries the update key (none yet).

## Get it and run it

### Download a release

Every [release](https://github.com/guided-traffic/cogpit/releases) carries the complete product
only ([D-16](decisions.md#d-16-the-first-real-release)): today Cogpit for Apple silicon as
`Cogpit-<version>-macos-arm64.dmg`, and `SHA256SUMS`. Open the disk image and drag Cogpit into
Applications; `cogpit-cli` comes inside the app ([`cogpit-cli`](#cogpit-cli)). The app with its
`cogpit-cli` and the disk image are signed with the project's Developer ID and notarized by Apple,
so macOS starts them without a warning
([releasing › signing and notarization](releasing.md#signing-and-notarization)). v0.1.0 to v0.4.0
carry `Cogpit_<version>_darwin_arm64.zip` instead, **unsigned**: macOS blocks that app's first
start until you allow it under System Settings › Privacy & Security › Open Anyway. Releases up to
v0.6.0 also carry `cogpit-cli` archives for Linux, macOS and Windows, those for Linux and Windows
not signed ([updates](../security/updates.md#what-a-release-holds-today)); later releases carry
none, and Cogpit for Linux and Windows comes with its desktop app (M11).

```sh
shasum -a 256 -c SHA256SUMS
spctl -a -t open --context context:primary-signature -vv Cogpit-<version>-macos-arm64.dmg   # source=Notarized Developer ID
/Applications/Cogpit.app/Contents/MacOS/cogpit-cli version   # once the app is in Applications
```

### Run the MVP

The MVP is a local build for macOS (tested on Apple silicon, macOS 26). It needs Go (1.26 or
later; the build fetches Go 1.27.2 itself), the Xcode Command Line Tools (Fyne uses cgo), and for
the kind cluster Docker, `kind` and `kubectl`.

```sh
git clone https://github.com/guided-traffic/Cogpit.git
cd Cogpit

make run-demo                 # synthetic cluster: 10,000 pods, 1,000 watch events per second

make kind-up                  # a kind cluster "cogpit" with its own kubeconfig (.cache/kind/kubeconfig)
make run                      # Cogpit connected to it; make kind-down removes it

make build && bin/cogpit      # your own clusters: syncs ~/.kube (config included) and offers the files $KUBECONFIG names, each file on its own
make app                      # bin/Cogpit.app (ad-hoc signed), starts by double-click
```

`make kind-up` seeds the namespace `cogpit-demo` ([hack/kind](../../hack/kind/README.md)): deployments
`web` and `api`, a crash-looping pod, a completed job, a `toolbox` pod with a shell, and the Service
`web` with the ClusterIP `10.96.100.100`. Nothing writes `~/.kube/config`; `make run` lists your `~/.kube` contexts too (Kubeconfig Sources removes the folder from the list, never from disk).

**Drive it with an agent.** Help › Copy MCP command (or the MCP button in the top bar) copies the
command that registers Cogpit with Claude Code, and Cogpit prints it at start too. The installed
app copies the registration of the `cogpit-cli` inside it, which you run once
([D-2](decisions.md#d-2-agent-interface-scope)):

```sh
claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user cogpit -- /Applications/Cogpit.app/Contents/MacOS/cogpit-cli mcp   # example
```

Claude Code then starts that `cogpit-cli mcp`, which reads the running app's port and token
itself, so the registration follows every restart and update of Cogpit; run it again only if you
move `Cogpit.app`. A build from source (`bin/cogpit`), and the app while it still runs from its
disk image, offer the HTTP registration instead, whose port and token change on every start:

```sh
claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user --transport http cogpit http://127.0.0.1:<port>/mcp --header "Authorization: Bearer <token>"
```

so repeat it after a restart, or register the source build's bridge once instead:
`claude mcp add --scope user cogpit -- "$PWD/bin/cogpit-cli" mcp` # example. Then ask Claude
Code "show me the Service with ClusterIP 10.96.100.100": Cogpit opens Services, selects `web` and
opens its drawer. Agents may read and navigate freely; Restart, Delete and the pod shell wait for
your confirmation in Cogpit's window ([agents and MCP](../security/agents-and-mcp.md)).

**Verify.** `make test-unit-race` runs every test without a cluster; with the kind cluster up,
`make test-kind` also runs the cluster tests (views, logs, pod shell, `kubectl` in the terminal).

**Not built yet (2026-10-14):** Windows and Linux packages (the only way `cogpit-cli` is to reach them in a release, M11), the app for Intel Macs, updates, the built-in agent chat; what exists and
what is missing, row by row: [docs/parity](../parity/README.md).

<details>
<summary>Clean up</summary>

```sh
make kind-down     # deletes the kind cluster and its kubeconfig file
make clean         # removes bin/ and dist/
rm -rf ~/Library/Caches/Cogpit "~/Library/Application Support/Cogpit"   # logs, audit log, token, downloaded kubectl releases
```

</details>

## Reference

### `cogpit` flags

| Flag | Default | Meaning |
|---|---|---|
| `--kubeconfig <files>` | empty: the files `$KUBECONFIG` lists # default | kubeconfig files, separated by `:`, offered as sources of their own for this run, each read alone and removable until the next start; `~/.kube` is synced either way (a bare `--context` name resolves to the one file that defines it) |
| `--context <name>` | empty: start on the welcome page, under the tabs of the last run, nothing connected # default | connect to this context at start; one that fails opens the context picker with the error |
| `--demo` | `false` # default | connect to the synthetic demo cluster |
| `--demo-pods <n>` | `10000` # default | pods in the demo cluster |
| `--demo-events <n>` | `1000` # default | watch events per second in the demo cluster |
| `--mcp-addr <addr>` | `127.0.0.1:0` # default | MCP listen address; anything but a loopback address is refused |
| `--no-mcp` | `false` # default | do not start the external MCP server; without it, click the MCP item in the status bar to switch the server off and on (the choice is remembered; `--no-mcp` wins over it) |
| `--perf <duration>` | `0` # default | measure main-loop latency while auto-scrolling the table, print it as JSON and quit |
| `--theme <mode>` | `dark` # example | `dark` (Solarized Dark), `light` (Solarized Light) or `system`, for this run; without it the theme of Preferences › App (⌘,) applies, Dark until changed |
| `-v` | `false` # default | debug logging |

Security note on `--mcp-addr`: the server checks `Host`, `Origin` and the bearer token, and binds
loopback addresses only, because any reachable listener would let others drive Cogpit with your
cluster identities.

### `cogpit-cli`

The headless command-line tool (pure Go, no window). It comes inside the app, beside the app's own
executable, from the first release after v0.6.0 on
([D-16](decisions.md#d-16-the-first-real-release)); a source checkout builds it as
`bin/cogpit-cli` (`make build`):

```sh
/Applications/Cogpit.app/Contents/MacOS/cogpit-cli contexts                  # example: the installed app's
ln -s /Applications/Cogpit.app/Contents/MacOS/cogpit-cli ~/bin/cogpit-cli    # example: as cogpit-cli on your PATH; updates follow
```

It is signed and notarized with the app and replaced with it by an update. Releases no longer carry
it for Linux and Windows (they get Cogpit with its desktop app, M11); there, build it from source:
`CGO_ENABLED=0 go build ./cmd/cogpit-cli`. It offers the same contexts as the app, from
the same kubeconfig sources (`~/.kube`, the files `--kubeconfig` or `$KUBECONFIG` names, the files
and sync folders added in Cogpit), and runs a context's credential plugin only once you allowed it
in Cogpit's window; it never asks itself and never writes Cogpit's sources or approvals. Exit
codes: `0` done, `1` the command failed, `2` the command line is wrong. `cogpit-cli <command>
--help` prints a command's flags.

| Command | Flags | What it does |
|---|---|---|
| `cogpit-cli contexts` | `--kubeconfig <files>` (empty: the files `$KUBECONFIG` lists # default), `-o table` # default (`json`, `yaml`), `-v` (`false` # default) | lists every context as `<context> [<file>]`, with its namespace, server and whether its login (a credential plugin, or a token file outside its folder) is `approved`, `required` or `changed`, and `not checked` under CERTIFICATE for a context whose cluster sets `insecure-skip-tls-verify` (the last column; Cogpit's default); `json` and `yaml` are the `cluster_contexts` tool's result (`insecure: true`); reads the kubeconfig files only |
| `cogpit-cli get <kind> [name]` | `--context <context>` (required; an id as `contexts` lists it, a context name only one file defines, or `demo`), `-n`/`--namespace <ns,...>` (empty: every namespace, for a named object the context's namespace # default), `-o table` # default (`json`, `yaml`), `--filter <text>` (empty # default), `--timeout 1m` # default, `--kubeconfig <files>`, `-v` | connects through a headless engine, waits for the cache's first list of the kind and prints it as Cogpit's view does (its columns, those empty in every row left out), then exits; `json` and `yaml` print the `resources_list` result for a list (at most 1,000 rows; stderr says when there are more) and the object's manifest for a named object (managed fields dropped, Secret values redacted). `<kind>` is a kind key (`pods`, `certificates.cert-manager.io`), or a Kubernetes kind or view title that names one kind (`Pod`) |
| `cogpit-cli mcp` | `--headless` (`false` # default), `--kubeconfig <files>` (the headless engine only), `-v` | serves Cogpit's MCP tools over stdin and stdout for agents that speak only stdio: with Cogpit running, every call is forwarded to it (its confirmations and audit log apply; a restart of Cogpit is followed); without it, or with `--headless`, an engine of its own serves them, and what needs your confirmation is refused. stdout carries the protocol only; logs go to stderr |
| `cogpit-cli version` | – | prints the version, the Go version and the platform (`--version` too): a release's `cogpit-cli 0.7.0 (go1.27.2, darwin/arm64)` # example, a development build's `dev` and its commit, `cogpit-cli dev 0691fb3277bb-dirty (…)` # example ([which build reports what](releasing.md#the-version-a-build-reports)) |

```sh
cogpit-cli contexts                                         # every context Cogpit offers
cogpit-cli get pods -n kube-system --kubeconfig .cache/kind/kubeconfig --context kind-cogpit   # example: kind's system pods
cogpit-cli get deployments web -n cogpit-demo -o yaml --kubeconfig .cache/kind/kubeconfig --context kind-cogpit   # example: one manifest
cogpit-cli get services --context demo                      # the synthetic demo cluster
```

Security note on `mcp`: the bridge sends the token from Cogpit's token file only to the loopback
URL in `mcp-endpoint`, follows no redirect and uses no proxy. Switching the MCP server off in the
status bar stops a bridge that forwards to the app, but not one that serves its own engine, which
reads clusters and refuses every write ([H-28](../security/agents-and-mcp.md#h-28-the-status-bars-switch-does-not-reach-a-headless-bridge)).

Security note on `get`: it only reads (`resources_list` and `object_get` through the cache, origin
CLI), prints a Secret's key names and sizes but never a value, and refuses a context whose
credential plugin you have not allowed in Cogpit's window before the plugin runs. Like the app's
views it watches a kind in every namespace and filters by `-n` afterwards, so a user who may list
a kind only in some namespaces gets the API server's refusal.

Security note on `contexts`: context names, file names and every other text from a kubeconfig or a
cluster are escaped before they reach your terminal (`\u001b` for an escape character), since
anyone who can drop a file into a sync folder chooses them. Without a configuration directory
(no `$HOME`) the CLI refuses to connect rather than merging the files as kubectl does, which
would run their plugins unasked.

### Preferences › Kubernetes: `kubectl` for terminals

Kept in `<user config dir>/Cogpit/preferences.json` (owner-only), set on the Preferences page (⌘,)
or with `preferences_set`. When a terminal opens, its `kubectl` is compared with the cluster's
version; more than one minor version off, or none, and the terminal offers the matching release
([D-3](decisions.md#d-3-helm-and-kubectl-binaries)).

```json
{
  "kubectlDownload": true,
  "kubectlMirror": "",
  "kubectlDir": "",
  "kubectlPath": ""
}
```

| Field | Default | Meaning |
|---|---|---|
| `kubectlDownload` | `true` # default | offer the matching release in a terminal whose `kubectl` does not suit the cluster; off, the terminal only says so. The download itself always waits for your click |
| `kubectlMirror` | `""`: `https://dl.k8s.io/release` # default | the https base URL under which `v<version>/bin/<os>/<arch>/kubectl` and its `.sha256`, `.sig` and `.cert` lie, such as `https://mirror.example.com/kubernetes/kubectl` # example |
| `kubectlDir` | `""`: `<user config dir>/Cogpit/kubectl` # default | the directory for binaries, an absolute path; releases go into `<dir>/<version>/kubectl` |
| `kubectlPath` | `""`: the first `kubectl` on `PATH` # default | the `kubectl` your terminals run, an absolute path; Cogpit checks it against the cluster as it would check `PATH`'s and puts it first on the terminal's `PATH` |

Security note: the last three decide which program your terminals run as `kubectl`, so only you, in
Cogpit's window, can set them, and only you can start a download; agents and `cogpit-cli` are
refused. A mirror needs no trust of its own: every download is checked against the Kubernetes
release's sigstore signature (its certificate chained to sigstore's authority, which Cogpit embeds,
for the release's identity), and refused when the mirror serves no `.sig` or `.cert`. The download
goes through the HTTP proxy for non-cluster traffic (`httpProxy`, Preferences › Proxy) when one is set. sigstore's
transparency log is not consulted ([H-35](../security/trust-boundaries.md#h-35-kubectls-signature-is-checked-without-sigstores-transparency-log)).

### Preferences › Kubernetes: chart icons

The Charts view and a chart's drawer show the icon a chart repository's index names
([D-46](decisions.md#d-46-chart-icons-from-the-hosts-that-serve-them)), fetched only for what the window shows.

```json
{
  "loadChartIcons": true
}
```

| Field | Default | Meaning |
|---|---|---|
| `loadChartIcons` | `true` # default | fetch each chart's icon from the host its URL names; off, Cogpit's own helm-wheel glyph and nothing fetched |

Security note: an icon's URL may point at any host, which then learns that your computer looked at
the chart ([H-87](../security/trust-boundaries.md#h-87-chart-icons-tell-their-hosts-what-the-user-looks-at)).
Cogpit fetches it over https only, without the repository's credentials, cookies or a referrer, at
most 256 KiB, never from an address in your own network unless it is a chart repository's own host,
and through the HTTP proxy for non-cluster traffic when one is set; a PNG or JPEG is decoded under
bounds and kept as a small PNG, an SVG only when it refers to nothing outside itself. Agents never
fetch icons, and only you, in Cogpit's window, can turn the switch on; agents may turn it off.

### Preferences › App: start at login

macOS only (13 or later): Preferences › App › Start at login makes macOS open Cogpit when you log
in, as the app's own login item, which System Settings › General › Login Items lists too
([D-48](decisions.md#d-48-start-at-login)). It is macOS's state, not a field of `preferences.json`:
the row shows what macOS reports, read when the section opens.

| Status (`preferences_loginItem`, `ui_state`'s `loginItem`) | Meaning |
|---|---|
| `off` # default | not registered: Cogpit starts when you open it |
| `on` | registered: macOS opens Cogpit at login |
| `requiresApproval` | registered, but macOS waits for you to allow it in System Settings › General › Login Items (the row's Open Login Items… button goes there) |
| `notFound` | macOS finds no login item for this Cogpit: it needs the signed `Cogpit.app` in Applications; a `bin/cogpit` or `go run` build reports this without asking macOS |

Security note: with it on, Cogpit runs after every login without being opened, its MCP endpoint
included unless you switched that off; so only you, in Cogpit's window, can turn it on, and agents
may only turn it off. Not offered on Windows and Linux yet. What macOS does with the ad-hoc signed
app of today's releases is not verified (the Developer ID signature is being built).

### Deep links (`cogpit://`)

macOS only: the app registers the URL scheme `cogpit`, so a link in a runbook, a chat or an alert
opens Cogpit at a cluster, a view or an object ([D-47](decisions.md#d-47-deep-links-cogpit)).

```text
cogpit://open?context=kind-dev%20%5Bconfig%5D&view=pods&namespace=team-a&name=web-0   # example
```

| Parameter | Value | Without it |
|---|---|---|
| `context` | a context id as `cluster_contexts` lists it (`<context> [<file>]`), or a kubeconfig context name that one source file alone defines; `demo` for the demo cluster | the connected cluster |
| `view` | a kind key (`pods`, `deployments`, `gateways.gateway.networking.k8s.io`) or a page key (`cluster/overview`) | the view the connect shows |
| `namespace` | a namespace: with `name` the object's; without it the namespace filter | the filter as it is |
| `name` | an object of the view's kind: its drawer opens; needs `view` to be a kind, not a page | no drawer |

At least one parameter; each at most once, percent-encoded, at most 2,048 bytes in all. Cogpit
comes to the front, connects the context unless it is connected, and opens what the link names;
a toast says what it opened or why not, and the audit log records the link (origin `deeplink`).

Security note: any web page or app can open such a link, so a link only navigates: it never
writes to a cluster, never runs a command with a confirmation, never names a file, and never
connects a context whose login program or token file you have not allowed yet (connect from the
picker to review it). A malformed link is refused whole. What remains is in
[H-89](../security/trust-boundaries.md#h-89-a-link-switches-the-cluster-and-the-view-under-the-user)
and [H-90](../security/trust-boundaries.md#h-90-a-links-text-reaches-agents-through-the-logs).
Not on Windows and Linux yet.

### Preferences › App: updates

The macOS app checks Cogpit's signed appcast with Sparkle 2 and offers a newer release in Sparkle's
window, installed on your click ([D-4](decisions.md#d-4-packaging-and-updates)); Help › Check for
Updates… checks now. Kept in `<user config dir>/Cogpit/preferences.json` with the other
preferences.

```json
{
  "checkForUpdates": true,
  "updateChannel": "stable"
}
```

| Field | Default | Meaning |
|---|---|---|
| `checkForUpdates` | `true` # default | Sparkle checks `https://guided-traffic.github.io/cogpit/appcast.xml` once a day; off, only Help › Check for Updates… checks |
| `updateChannel` | `"stable"` # default | the only channel until Cogpit publishes prereleases; `beta` is refused for now |

Security note: an agent or `cogpit-cli` may turn `checkForUpdates` on, never off; only you can, in
Cogpit's window ([agents and MCP](../security/agents-and-mcp.md)). A build without the update key
(every local build, and every release until the owner has made the key), an app still on its disk
image and other systems than macOS do not update themselves, and the row says why. `cogpit-cli`
has no updater of its own: the one inside the app is replaced with the app, one built from source
is not.
Sparkle installs an update only when its disk image and the feed carry the update key's signature
and the new app carries Cogpit's Developer ID ([updates](../security/updates.md)).

### Commands and MCP tools

Every command is an MCP tool. The table of all of them, with each one's danger and parameters,
is generated from the registry: [`commands.md`](commands.md).

## Naming conventions

Names Cogpit generates or expects. Paths are macOS paths; `<user config dir>` is
`~/Library/Application Support`, `<user cache dir>` is `~/Library/Caches`.

<details>
<summary>Names, paths and files</summary>

| What | Pattern | Example |
|---|---|---|
| Go module | `github.com/guided-traffic/cogpit` | |
| Binaries | `cogpit` (desktop app), `cogpit-cli` (headless CLI and MCP stdio bridge) | |
| App bundle | `Cogpit.app`, bundle id `io.guided-traffic.cogpit` | `bin/Cogpit.app` # example |
| `cogpit-cli` in the app | `Cogpit.app/Contents/MacOS/cogpit-cli`, code-signing identifier `io.guided-traffic.cogpit.cli` ([D-16](decisions.md#d-16-the-first-real-release)) | `/Applications/Cogpit.app/Contents/MacOS/cogpit-cli` # example |
| Packages | `internal/<area>` | `internal/cache` # example |
| Command ids | `<kind-or-area>.<action>`, lowerCamelCase segments | `deployment.restart`, `ui.navigate` |
| MCP tool names | the command id with dots replaced by underscores | `deployment_restart`, `ui_navigate` |
| Kind keys (views, `kind` parameters) | the plural resource name for built-in kinds; `<plural>.<group>` for kinds from custom resource definitions ([why](architecture.md#45-kind-keys)) | `pods`, `services`, `gateways.gateway.networking.k8s.io` |
| Page keys (views that are not one kind's list, `ui.openView`'s `view`) | `<navigator group>/<page>` | `cluster/overview`, `workloads/overview`, `flux/dashboard` |
| Annotations the Flux actions set | `reconcile.fluxcd.io/requestedAt` (Reconcile, Resume) and, for a forced Helm release, `reconcile.fluxcd.io/forceAt`: the time in RFC 3339 with nanoseconds, as the flux CLI writes it | `2026-10-13T22:01:02.345678901Z` # example |
| Environment variables | `COGPIT_<NAME>`; today only the test variables `COGPIT_KIND_KUBECONFIG`, `COGPIT_KIND_CONTEXT` and `COGPIT_PNG_DIR` | |
| MCP endpoint | `http://127.0.0.1:<random port>/mcp`, new port on every start | `http://127.0.0.1:59449/mcp` # example |
| MCP bearer token | `<user config dir>/Cogpit/mcp-token`, owner-only (0600, directory 0700), new on every start, removed on exit and when you switch the external MCP server off | |
| MCP endpoint file | `<user config dir>/Cogpit/mcp-endpoint`, beside the token and with its rules: the endpoint's URL, for `cogpit-cli mcp` | `http://127.0.0.1:59449/mcp` # example |
| Audit log | `<user config dir>/Cogpit/audit.log`, owner-only, one JSON line per agent-initiated command; read back newest first in Help › Audit Log and by `audit_list` (its last 4 MiB) | |
| Drain journal | `<user config dir>/Cogpit/drains.json`, owner-only (0600, directory 0700): the drains Cogpit started and has not finished, offered again on the next connect | |
| Node shell journal | `<user config dir>/Cogpit/nodeshells.json`, owner-only (0600, directory 0700): the node shell pods Cogpit created and has not seen deleted (context, namespace, name, UID, the id in the pod's `cogpit.io/node-shell-id` annotation); one a crash left is deleted at the next connect to its context ([H-71](../security/privilege-footprint.md#h-71-a-crash-leaves-the-node-shells-pod-until-the-next-connect)) | |
| Hotbar | `<user config dir>/Cogpit/hotbar.json`, owner-only (0600, directory 0700): the banks of pinned clusters by context id; an unreadable one is moved to `hotbar.json.unreadable` at the first change | |
| Tabs and column layout | `<user config dir>/Cogpit/workspace.json`, owner-only (0600, directory 0700): the tab bar's tabs (context id, view, filters, search, sort, history) and each view's hidden columns and column order, restored at start without connecting anything ([D-41](decisions.md#d-41-what-survives-a-restart)); an unreadable one is moved to `workspace.json.unreadable` at the first change | |
| Helm chart repositories | Helm's own `repositories.yaml`, as the helm CLI resolves it: `$HELM_REPOSITORY_CONFIG`, else `$HELM_CONFIG_HOME/repositories.yaml`, else Helm's folder; read every 2 seconds, never written (add and remove repositories with `helm repo add` and `helm repo remove`, [D-29](decisions.md#d-29-helm-chart-repositories-from-helms-repositoriesyaml)); credentials stay in memory | `~/Library/Preferences/helm/repositories.yaml` # default on macOS |
| Helm repositories allowed over http | `<user config dir>/Cogpit/helm-http-approvals.json`, owner-only (0600, directory 0700): the repositories whose user and password you let Cogpit send over plain http, each by its name, its URL without user, password or query values, a SHA-256 of its name, URL and user name and when; no credential ([H-40](../security/credentials.md#h-40-credentials-over-plain-http)) | |
| Helm repository indexes | `<user cache dir>/Cogpit/helm/indexes/<key>-index.yaml`, owner-only (0600, directory 0700); the key a hash of the repository's name and URL. Nothing goes into Helm's own cache | `~/Library/Caches/Cogpit/helm/indexes/3f2a9c61d0b4e8a7c5f19e2d4b6a8c0e-index.yaml` # example |
| Helm chart READMEs | `<user cache dir>/Cogpit/helm/readmes/<key>.json`, owner-only (0600, directory 0700): the README of a chart version whose archive the index's digest verified, as the chart's drawer showed it (at most 256 KiB), the last 256 kept; the key a hash of the repository's name and URL, the chart, the version and the digest | `~/Library/Caches/Cogpit/helm/readmes/5d0c3f9e2a7b41c8e6f0a19b3d7e2c45.json` # example |
| Helm chart icons | `<user cache dir>/Cogpit/helm/icons/<key>.png` or `.svg`, owner-only (0600, directory 0700): a chart's icon as Cogpit checked it (a PNG of at most 256 pixels a side, or an SVG that refers to nothing outside itself), used for 7 days, the last 2,048 kept; the key a hash of the icon's URL ([D-46](decisions.md#d-46-chart-icons-from-the-hosts-that-serve-them)) | `~/Library/Caches/Cogpit/helm/icons/0b7e5d2c9a14f3e8b6c1d0a9e7f42c51.png` # example |
| Helm chart objects | kind `helmcharts`, named `<repository>.<chart>` (a repository whose name has a dot is listed as not used) | `bitnami.nginx` # example |
| App log | `<user cache dir>/Cogpit/cogpit.log`, and stderr | |
| Terminal kubeconfig | a temporary file per terminal holding only a context, owner-only, deleted on close ([why](../security/credentials.md#the-terminals-kubeconfig)) | |
| Downloaded `kubectl` | `<user config dir>/Cogpit/kubectl/<version>/kubectl` (0755, directories 0700), or the directory for binaries set in Preferences › Kubernetes ([why](decisions.md#d-3-helm-and-kubectl-binaries)) | `~/Library/Application Support/Cogpit/kubectl/1.36.1/kubectl` # example |
| A terminal's own directory | `<temp dir>/cogpit-terminal-<random>/`, owner-only, removed on close, only when Cogpit puts a `kubectl` first: `bin/kubectl` (a link), `zsh/.zshenv` (zsh only) | |
| Node shell pod | `kube-system/cogpit-node-shell-<8 random characters>`, labels `app.kubernetes.io/name=cogpit-node-shell` and `app.kubernetes.io/managed-by=cogpit`, annotation `cogpit.io/node-shell-id=<random id>` (its journal entry's), deleted when the shell closes, else at the next connect to its context after a crash, else ended by its deadline, Cluster Settings › Node Shell › Maximum duration (2 hours unless the user sets 5 minutes to 24 hours; [why](../security/privilege-footprint.md#node-shell)) | `kube-system/cogpit-node-shell-k3v9q2mx` # example |
| Node shell dock tab | id `nodeShell-<n>`, title `Node shell: <node>` | `Node shell: worker-1` # example |
| Demo context | `demo` (the synthetic cluster) | |
| Local kind cluster | cluster `cogpit`, context `kind-cogpit`, kubeconfig `.cache/kind/kubeconfig` | |
| Release artifacts (planned) | `Cogpit-<version>-macos-universal.dmg`; the desktop app for Windows and Linux (M11), each with `cogpit-cli` inside | `Cogpit-0.1.0-macos-universal.dmg` # example |
| Deep links (macOS) | `cogpit://open?context=<context id>&view=<view key>&namespace=<namespace>&name=<name>`, the scheme registered in the app's `CFBundleURLTypes` (name `io.guided-traffic.cogpit`); audited as `deeplink.open` with origin `deeplink` | `cogpit://open?view=pods&namespace=team-a` # example |
| Release artifacts | `Cogpit-<version>-macos-<arch>.dmg` (signed releases; `universal` once the app is built for Intel too, planned), `Cogpit_<version>_darwin_<arch>.zip` (v0.4.0 and earlier, and builds without the signing secrets), `SHA256SUMS`; nothing else since D-16's revision (2026-10-14): `cogpit-cli_<version>_<os>_<arch>.tar.gz` (`.zip` for Windows) only up to v0.6.0 | `Cogpit-0.5.0-macos-arm64.dmg` # example |
| Deep links (planned) | `cogpit://` | |
| Update feed | the appcast every macOS app with the update key reads (`SUFeedURL`, [D-4](decisions.md#d-4-packaging-and-updates)) | `https://guided-traffic.github.io/cogpit/appcast.xml` # default |
| Sparkle's own state | the app's user defaults, written by Sparkle: whether it checks automatically (`SUEnableAutomaticChecks`, which Cogpit sets from its preference), the last check's time, a skipped version | `~/Library/Preferences/io.guided-traffic.cogpit.plist` # default |

</details>

Identifiers the documentation uses:

<details>
<summary>Identifiers</summary>

| Identifier | Meaning | Where |
|---|---|---|
| `G<n>` | A gap against Lens or a beyond-Lens target | [docs/parity/gaps.md](../parity/gaps.md) |
| `M<n>` | A milestone | [docs/parity/roadmap.md](../parity/roadmap.md) |
| `D-<n>` | A decision | [docs/developer/decisions.md](decisions.md) |
| `H-<n>` | An open security gap | [docs/security/](../security/README.md) |
| `P<n>` | A performance or API-load budget | [docs/developer/testing.md](testing.md#performance-budgets-proposed-not-measured) |
| `S<n>` | A gate criterion of the Fyne spike | [docs/developer/spike-fyne.md](spike-fyne.md#gate-criteria) |

</details>

## Make targets

| Target | What it does |
|---|---|
| `make build` | `bin/cogpit` (cgo, Fyne) and `bin/cogpit-cli` (pure Go) |
| `make run`, `make run-demo` | run against the kind cluster, or the demo cluster |
| `make test-unit-race`, `make test` | every test without a cluster, race detector on |
| `make test-kind` | cluster tests against the kind cluster (build tag `kind`) |
| `make test-unit-coverage`, `make coverage-badge` | the race tests with a coverage profile; `.github/badges/coverage.json` from it (CI keeps it current on main) |
| `make vuln` | govulncheck over the code paths Cogpit calls |
| `make kind-drain-up`, `make test-kind-drain`, `make kind-drain-down` | a separate three-node kind cluster (`cogpit-drain`, own kubeconfig `.cache/kind-drain/kubeconfig`) for the cordon, uncordon and drain tests |
| `make bench` | the gate benchmarks (S1, S2) |
| `make lint`, `make fmt` | golangci-lint v2.14.0 (installed into `bin/`) with the team's linters and the `depguard` layering |
| `make kind-up`, `make kind-seed`, `make kind-down` | the local kind cluster and its seed |
| `make kind-prometheus`, `make kind-prometheus-down` | a real Prometheus in the kind cluster (`monitoring/prometheus-server`, the `prometheus` chart) for the drawer's charts ([hack/kind](../../hack/kind/README.md#prometheus)) |
| `make app` | `bin/Cogpit.app` via `fyne package`, with Sparkle in `Contents/Frameworks` (`SPARKLE_PUBLIC_ED_KEY=<base64>` for an app that updates itself) and `cogpit-cli` in `Contents/MacOS` |
| `make sparkle` | the pinned Sparkle release in `.cache/sparkle/<version>`, its SHA-256 checked |
| `make appcast-draft`, `make appcast-sign`, `make appcast-verify` | the release pipeline's appcast jobs ([releasing › the appcast](releasing.md#the-appcast)) |

## Development

Through the `Makefile`, as in the team's other Go repositories:

```sh
make build
make lint
make test-unit-race
make test-kind      # needs make kind-up
```

CI runs each check as a job of its own with its own make target (`make gosec`, `make cyclo`,
`make test-race-ui RACE_SHARD=n`, …), and semantic-release publishes every `feat:` and `fix:` on
`main` once CI is green ([releasing](releasing.md)). Everything else:
[docs/developer/](README.md).
