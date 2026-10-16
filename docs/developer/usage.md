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
into a cluster. **Status: pre-alpha** (updated 2026-10-15): every view and every action on Lens's
list exists at least in part (64 of 64 views, 42 of 42 actions, counted by `make parity`), with
the Flux CD and Argo CD sections, Helm releases and charts, metrics, logs, a terminal, pod and node
shells, the built-in agent's chat and MCP control of all of it; none of it has been walked through
by a person on a real cluster yet. The releases are builds for Apple silicon, signed and notarized
since v0.6.0 (v0.6.0 to v0.8.0 carry the download; v0.5.0 and v0.9.0 to v0.12.0 got no files)
([Download a release](#download-a-release),
[Run the MVP](#run-the-mvp)); where Cogpit stands: [roadmap](../../ROADMAP.md).

How data flows from the kubeconfig files to the views and agents:
[developer documentation › Data flow](README.md#data-flow).

## Features in detail

What Cogpit does and is meant to do, feature by feature. Everything below exists at least in part
unless it is marked *(planned)*; how far each one goes is measured row by row in
[docs/parity](../parity/README.md) and summed up in [ROADMAP.md](../../ROADMAP.md).

- 🖥️ A native macOS app first; Windows and Linux (Ubuntu first) after the macOS beta *(planned)*.
- 🗂️ Every cluster from your kubeconfig files and synced folders, side by side.
- 🧭 Lens's navigator levels: All Clusters (Local Kubeconfigs) lists every cluster and the tabs of all of them, the one list to choose a cluster from, with a + to add kubeconfigs; › drills into one cluster's resources and its tabs, ‹ All Clusters goes back up.
- 📌 A hotbar of pinned clusters at the window's edge, in banks of twelve: F1–F12 to switch cluster, ⇧F1–⇧F12 to switch bank.
- ⌨️ A command palette on ⌘⇧P (Ctrl+Shift+P): every action the menus offer, Switch to any view, the hotbar's clusters and banks; ⌘P (Cluster › Connect to Cluster…) to search a cluster by name in the navigator's list, ↵ to connect, Escape to go back.
- 👋 A welcome page whenever no cluster is open (and from Help › Welcome): your clusters one click away, Add Kubeconfigs, the documentation. Disconnect a cluster, or remove it from Cogpit, which unlinks the kubeconfig file you added and never touches the file.
- ⚙️ Preferences on ⌘, and a cluster's Settings on ⌘⇧,: the theme switched live, the time zone, the editor's and the terminal's settings, a cluster's name and icon (a colour or your image), its terminals' folder and namespace, its Prometheus and how it is asked (a query format, empty containers filtered, HTTPS), Cogpit Metrics; never written to a kubeconfig.
- 🕐 Every timestamp in your time zone and your region's format, as your Mac's region settings write it ("10.09.2026 17:41:01" in Germany): the drawer, events, logs, notifications, charts.
- 🔐 Your credentials stay yours: kubectl's own authentication code (certificates, tokens, exec plugins for EKS, GKE, AKS); no local proxy, no login.
- ⚡ Cache first: views read from a watch-fed cache; switching back to a view costs no API request.
- 📋 Lens's resource views: workloads, config, network, storage, access control, device resources (with each node's devices in its drawer), custom resources; each searchable by words, with match case or a regular expression, and filtered by the states its rows carry (Running, Pending, Not Ready, Degraded…), several at once, cleared with one click.
- 🔏 The certificates in your Secrets decoded in the drawer, PEM or DER, a section per certificate of a chain: subject, issuer, serial, validity (a warning 30 days before it ends), names, key, usages, fingerprint, each field copyable; whether `tls.key` belongs to `tls.crt` and the chain is in order; agents read the same fields, never the PEM text or the private key.
- 📜 Logs with follow, filter, timestamps, the previous container, pod and container selectors, older lines and a jump to a time.
- 💻 Pod shells, attach and a terminal bound to the cluster's context; copy and paste by keys or, if you choose, the right mouse button.
- 🖥️ A node shell, as Lens opens it: a privileged pod in `kube-system` from an image pinned by digest, confirmed every time, deleted when the shell closes.
- 🔐 Create Secret from a dialog, values masked as you type, for you alone: agents can neither create a Secret nor read its values.
- 🗝️ A Secret's values edited in its drawer, for you alone: Show puts a value into an editor, Save shows which keys change and their sizes, never a value, and checks with the server first; agents can neither save nor review a Secret's values, and unsaved ones never touch the disk.
- 🧰 The terminal's `kubectl` checked against the cluster: more than one minor version off, or none, and one click downloads the matching release from `dl.k8s.io`, verified against its SHA-256, the Kubernetes release's signature and that signature's entry in sigstore's transparency log, for Cogpit's terminals only.
- ✏️ YAML editing with a diff and a server-side dry-run before every apply.
- 📝 A Config Map's values edited right in its drawer, each key above its editor, saved with a diff and a dry run; when someone else changed other keys meanwhile, Save moves your edits onto the newer version and shows you the diff again; unsaved values survive a quit or a crash as a draft.
- ☑️ Delete, Restart and the Flux and Argo CD actions on the rows you checked, up to 500 at once, after one confirmation that names them all; more than ten run in the background in a dock tab of their own, which shows each object as it is done or fails, counts them, and has Cancel to stop what is not done yet (nothing done is undone); a toast says how it ended. Agents follow the same tab in `ui_state` and may cancel it too.
- 🚧 Drain a node in the background, after its confirmation: a dock tab shows the cordon and each pod as it is evicted, which one a PodDisruptionBudget holds and why (the drain retries it until its timeout), and has Cancel. A drain that is cancelled, fails or is cut off by a lost connection or a crash leaves the node cordoned and partly drained, and a banner offers Finish drain, Uncordon and Dismiss; nothing is uncordoned by itself. An agent's drain goes on when the agent's call ends.
- 🔌 Port forwarding, one click to the browser.
- ⎈ The Helm releases in your clusters (Lens's Helm view), kept in Secrets or ConfigMaps: history, values, upgrade, rollback, through the Helm SDK, no `helm` binary needed.
- 📦 Charts from the repositories in Helm's own `repositories.yaml` (add them with `helm repo add`; Cogpit picks a change up within seconds, logins and CA files included, and never writes the file), installed from a Helm Install tab; a chart's drawer shows the README of the version you pick, and charts show their icons (switch them off in Preferences).
- 🔁 Flux CD: its section in the navigator (Helm releases, Kustomizations, sources, image automation, notifications) with each object's state, revision and message, a dashboard of what is not ready whose states open the kind's list filtered to them, and Reconcile, Reconcile with Source (which waits for the source, as the flux CLI does, in a dock tab of its own), Reset, Suspend and Resume, on a selection too.
- 🐙 Argo CD: its section in the navigator (Applications, Application Sets, Projects, and the repositories and clusters without their credentials), a dashboard of the Applications' health and sync state, Sync (with prune) and Refresh (normal or hard) through the Kubernetes API, on a selection too and within the project's sync windows, an Application's logs in one tab, and Argo CD's own UI through a port forward.
- 📈 CPU and memory from the cluster's metrics-server and its Prometheus: columns in the Pods and Nodes lists, charts with requests and limits in the pod and node drawers, and the cluster Overview: CPU, Memory and Pods of the master or worker nodes over a time range, with the Issues (pods, nodes and workloads in trouble, warning events) below. A node's history, its network and its filesystem come from node-exporter where it runs. A cluster without a Prometheus gets one on request: Cluster Settings › Cogpit Metrics installs Prometheus, kube-state-metrics and node-exporter from charts Cogpit bundles, each a Helm release in the namespace `cogpit-metrics`, and removes them again.
- 🛡️ Never surprises a cluster: destructive actions confirm, nothing is installed or written implicitly.
- 😴 Sleep and lost connections: when the Mac wakes, the network or a VPN changes, or the cluster stops answering, Cogpit notices within seconds, says "Reconnecting…" with a banner, keeps your views and tabs as they were, disables changes, and reconnects by itself; a login that expired asks you once ([below](#when-the-connection-breaks)).
- 🤖 Fully controllable over MCP, the UI included: an agent sees what you see and navigates for you.
- ⌨️ `cogpit-cli`, inside the app: your contexts and any kind's objects on the command line, read-only, and an MCP stdio bridge you register once.
- 💬 A built-in agent in a chat beside your views (⌘I), on any OpenAI-compatible endpoint such as LM Studio, LiteLLM or Open WebUI.
- 📦 Installs the Mac way: open the disk image, drag Cogpit into Applications, signed and notarized (since v0.6.0); *(planned)* a Homebrew cask for shell users.
- ⬆️ Updates with one click in the macOS app (Sparkle), verified against a signed update feed: built; the releases carry the update key since v0.11.0, and updates start once the first of them has passed Apple's notarization and its feed is served (not yet on 2026-10-15).

## Get it and run it

### Download a release

Every [release](https://github.com/guided-traffic/cogpit/releases) carries the complete product
only ([D-16](decisions.md#d-16-the-first-real-release)): today Cogpit for Apple silicon as
`Cogpit-<version>-macos-arm64.dmg`, and `SHA256SUMS`. Open the disk image and drag Cogpit into
Applications; `cogpit-cli` comes inside the app ([`cogpit-cli`](#cogpit-cli)). v0.6.0, v0.7.0 and
v0.8.0 carry these files; v0.5.0 and v0.9.0 to v0.12.0 have none, their signing or notarization
having not finished ([releasing](releasing.md#when-signing-or-notarization-fails)), and v0.13.0's
pipeline was still waiting for Apple's notarization on 2026-10-15. The app with its
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
git clone https://github.com/guided-traffic/cogpit.git
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

**Drive it with an agent.** The gear beside the MCP item at the right end of the status bar opens
the MCP details: the server's switch, its endpoint, its token (masked until you click Show), the
registration commands with a copy button each, and Regenerate token
([D-53](decisions.md#d-53-the-mcp-details-and-a-token-kept-until-the-user-regenerates-it)).
Help › Copy MCP command copies the command and opens them too, and Cogpit prints it at start. The
installed app offers the registration of the `cogpit-cli` inside it, which you run once
([D-2](decisions.md#d-2-agent-interface-scope)):

```sh
claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user cogpit -- /Applications/Cogpit.app/Contents/MacOS/cogpit-cli mcp   # example
```

Claude Code then starts that `cogpit-cli mcp`, which reads the running app's port and token
itself before every call, so the registration follows every restart and update of Cogpit and a
regenerated token; run it again only if you move `Cogpit.app`. Beside it, and alone in a build
from source (`bin/cogpit`) or the app while it still runs from its disk image, the details offer
the HTTP registration, with the URL and the token:

```sh
claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user --transport http cogpit http://127.0.0.1:<port>/mcp --header "Authorization: Bearer <token>"
```

Cogpit keeps the port and the token across restarts and the switch, so you run it once, and again
after you regenerate the token or when Cogpit had to move to another port because the kept one was
taken (it says so). Its token sits in Claude Code's configuration and stays valid until you
regenerate it ([H-98](../security/agents-and-mcp.md#h-98-a-leaked-token-stays-valid-until-the-user-regenerates-it));
a source build's bridge holds none:
`claude mcp add --scope user cogpit -- "$PWD/bin/cogpit-cli" mcp` # example. Then ask Claude
Code "show me the Service with ClusterIP 10.96.100.100": Cogpit opens Services, selects `web` and
opens its drawer. Agents may read and navigate freely; Restart, Delete and the pod shell wait for
your confirmation in Cogpit's window ([agents and MCP](../security/agents-and-mcp.md)).

**Verify.** `make test-unit-race` runs every test without a cluster; with the kind cluster up,
`make test-kind` also runs the cluster tests (views, logs, pod shell, `kubectl` in the terminal).

**Not built yet (2026-10-16):** Windows and Linux packages (the only way `cogpit-cli` is to reach them in a release, M11), the app for Intel Macs, the Homebrew cask. Updates work on
macOS: the first installed copy updated from 0.15.0 to 0.16.0 on 2026-10-16
([Preferences › App: updates](#preferences--app-updates)). What exists and what is missing, row by
row: [docs/parity](../parity/README.md).

<details>
<summary>Clean up</summary>

```sh
make kind-down     # deletes the kind cluster and its kubeconfig file
make clean         # removes bin/ and dist/
rm -rf ~/Library/Caches/Cogpit "~/Library/Application Support/Cogpit"   # logs, the discovery and Helm caches, audit log, token, downloaded kubectl releases
```

</details>

## Reference

### `cogpit` flags

| Flag | Default | Meaning |
|---|---|---|
| `--kubeconfig <files>` | empty: the files `$KUBECONFIG` lists # default | kubeconfig files, separated by `:`, offered as sources of their own for this run, each read alone and removable until the next start; `~/.kube` is synced either way (a bare `--context` name resolves to the one file that defines it) |
| `--context <name>` | empty: start on the welcome page, under the tabs of the last run, nothing connected # default | connect to this context at start; one Cogpit refuses (no such context) says why in a toast and shows the navigator's clusters with the search focused, as ⌘P does; a connect that fails says why on the welcome page |
| `--demo` | `false` # default | connect to the synthetic demo cluster |
| `--demo-pods <n>` | `10000` # default | pods in the demo cluster |
| `--demo-events <n>` | `1000` # default | watch events per second in the demo cluster |
| `--mcp-addr <addr>` | `127.0.0.1:0` # default | MCP listen address; anything but a loopback address is refused. Port 0 binds the port Cogpit kept in `mcp-port` (a free one the first time, or when the kept one is taken, which it then keeps); a port given here binds that port for this run and is not kept |
| `--no-mcp` | `false` # default | do not start the external MCP server; without it, click the MCP item in the status bar to switch the server off and on (the choice is remembered; `--no-mcp` wins over it), or its gear for the MCP details |
| `--perf <duration>` | `0` # default | measure main-loop latency while auto-scrolling the table, print it as JSON and quit |
| `--perf-mode <mode>` | empty # default | with `--perf`: `scroll`; or `gate`, alone too: start-up, scrolling, every view and typing into 5 MB of YAML in one run (the [spike](spike-fyne.md)'s measurements) |
| `--cpuprofile <file>` | empty # default | write a CPU profile of the first `--cpuprofile-for` to this file |
| `--cpuprofile-for <duration>` | `5s` # default | how long `--cpuprofile` records |
| `--theme <mode>` | `dark` # example | `dark` (Solarized Dark), `light` (Solarized Light) or `system`, for this run; without it the theme of Preferences › App (⌘,) applies, System (follow macOS) until changed (since 2026-10-14, [D-22](decisions.md#d-22-colour-theme)) |
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
and sync folders added in Cogpit), and runs a context's credential plugin, or connects a context
whose cluster sets `insecure-skip-tls-verify` (A28, since 2026-10-14), only once you allowed it in
Cogpit's window; it never asks itself and never writes Cogpit's sources or approvals. Exit
codes: `0` done, `1` the command failed, `2` the command line is wrong. `cogpit-cli <command>
--help` prints a command's flags.

| Command | Flags | What it does |
|---|---|---|
| `cogpit-cli contexts` | `--kubeconfig <files>` (empty: the files `$KUBECONFIG` lists # default), `-o table` # default (`json`, `yaml`), `-v` (`false` # default) | lists every context as `<context> [<file>]`, with its namespace, server and whether its login (a credential plugin, or a token file outside its folder) is `approved`, `required` or `changed`, and `not checked` under CERTIFICATE for a context whose cluster sets `insecure-skip-tls-verify` (the last column; Cogpit's default), followed for a source's context by whether you allowed connecting to it so (`not checked, required` # example; `approved`, `required` or `changed`, A28); `json` and `yaml` are the `cluster_contexts` tool's result (`insecure: true`); reads the kubeconfig files only |
| `cogpit-cli get <kind> [name]` | `--context <context>` (required; an id as `contexts` lists it, a context name only one file defines, or `demo`), `-n`/`--namespace <ns,...>` (empty: every namespace, for a named object the context's namespace # default), `-o table` # default (`json`, `yaml`), `--filter <text>` (empty # default), `--timeout 1m` # default, `--kubeconfig <files>`, `-v` | connects through a headless engine, waits for the cache's first list of the kind and prints it as Cogpit's view does (its columns, those empty in every row left out), then exits; `json` and `yaml` print the `resources_list` result for a list (at most 1,000 rows; stderr says when there are more) and the object's manifest for a named object (managed fields dropped, Secret values redacted). `<kind>` is a kind key (`pods`, `certificates.cert-manager.io`), or a Kubernetes kind or view title that names one kind (`Pod`). Connected to a context that skips the certificate check (`insecure-skip-tls-verify`, allowed once in Cogpit's window), it writes one line to stderr, `cogpit-cli: warning: TLS not checked for <context>. Certificate checks are off for this context …` (A29), and stdout stays the result |
| `cogpit-cli mcp` | `--headless` (`false` # default), `--kubeconfig <files>` (the headless engine only), `-v` | serves Cogpit's MCP tools over stdin and stdout for agents that speak only stdio: with Cogpit running, every call is forwarded to it (its confirmations and audit log apply; a restart of Cogpit, a new port and a regenerated token are followed); without it, or with `--headless`, an engine of its own serves them, and what needs your confirmation is refused. stdout carries the protocol only; logs go to stderr |
| `cogpit-cli version` | – | prints the version, the Go version and the platform (`--version` too): a release's `cogpit-cli 0.7.0 (go1.27.2, darwin/arm64)` # example, a development build's `dev` and its commit, `cogpit-cli dev 0691fb3277bb-dirty (…)` # example ([which build reports what](releasing.md#the-version-a-build-reports)) |

```sh
cogpit-cli contexts                                         # every context Cogpit offers
cogpit-cli get pods -n kube-system --kubeconfig .cache/kind/kubeconfig --context kind-cogpit   # example: kind's system pods
cogpit-cli get deployments web -n cogpit-demo -o yaml --kubeconfig .cache/kind/kubeconfig --context kind-cogpit   # example: one manifest
cogpit-cli get services --context demo                      # the synthetic demo cluster
```

Security note on `mcp`: the bridge sends the token from Cogpit's token file only to the loopback
URL in `mcp-endpoint`, reads both again before every call and sends nothing while there is no
endpoint file, follows no redirect and uses no proxy. Switching the MCP server off in the
status bar stops a bridge that forwards to the app, and a bridge started while it is off serves
no engine, headless included: the agent is told the server is off (`mcp-switch`, D-49). A run
with `--no-mcp` leaves the switch as it is ([H-28](../security/agents-and-mcp.md#h-28-the-status-bars-switch-does-not-reach-a-headless-bridge)).

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
  "kubectlPath": "",
  "kubectlEnforce": false
}
```

| Field | Default | Meaning |
|---|---|---|
| `kubectlDownload` | `true` # default | offer the matching release in a terminal whose `kubectl` does not suit the cluster; off, the terminal only says so. The download itself always waits for your click |
| `kubectlMirror` | `""`: `https://dl.k8s.io/release` # default | the https base URL under which `v<version>/bin/<os>/<arch>/kubectl` and its `.sha256`, `.sig` and `.cert` lie, such as `https://mirror.example.com/kubernetes/kubectl` # example |
| `kubectlDir` | `""`: `<user config dir>/Cogpit/kubectl` # default | the directory for binaries, an absolute path; releases go into `<dir>/<version>/kubectl` |
| `kubectlPath` | `""`: the first `kubectl` on `PATH` # default | the `kubectl` your terminals run, an absolute path; Cogpit checks it against the cluster as it would check `PATH`'s and puts it first on the terminal's `PATH` |
| `kubectlEnforce` | `false` # default | off: the chosen `kubectl` goes first on the terminal's `PATH` once the shell's profile ran (zsh through `ZDOTDIR`, bash through `--init-file`, fish through `-C`); on: zsh, bash and fish also get a function `kubectl` that runs it, so that a folder the profile puts first later, or an alias, does not win (A27, since 2026-10-14). Other shells keep the environment's `PATH` order ([H-36](../security/trust-boundaries.md#h-36-a-shells-profile-can-put-another-kubectl-first)) |

Security note: the last four decide which program your terminals run as `kubectl`, so only you, in
Cogpit's window, can set them, and only you can start a download; agents and `cogpit-cli` are
refused. A mirror needs no trust of its own: every download is checked against the Kubernetes
release's sigstore signature (its certificate chained to sigstore's authority for the release's
identity) and that signature's entry in sigstore's transparency log, all against the sigstore
trusted root Cogpit embeds, and refused when the mirror serves no `.sig` or `.cert`. The download
goes through the HTTP proxy for non-cluster traffic (`httpProxy`, Preferences › Proxy) when one is
set.

The download makes one more outgoing request (since 2026-10-15,
[D-50](decisions.md#d-50-kubectls-signature-check)): before the binary is fetched, Cogpit asks
sigstore's transparency log at `https://rekor.sigstore.dev` (`POST /api/v1/log/entries/retrieve`)
for its entry of the release's signature, through the same proxy, with the published SHA-256, the
signature and the certificate (public material, no credential), within 30 seconds. Whatever the
mirror, this host must be reachable: without its answer the download is refused ("sigstore's
transparency log could not be asked"), and there is no fallback to a check without it. In a network
that cannot reach it, put a `kubectl` on `PATH` or set `kubectlPath`
([H-35](../security/trust-boundaries.md#h-35-kubectls-signature-check-rests-on-an-embedded-trusted-root-and-rekor-v1s-search)).

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
may only turn it off. Not offered on Windows and Linux yet. What macOS does with the releases' app,
signed with the Developer ID since v0.6.0, is not verified: no signed app installed in Applications
has been tried ([D-48](decisions.md#d-48-start-at-login)).

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
navigator's clusters, ⌘P, to review it). A malformed link is refused whole. What remains is in
[H-89](../security/trust-boundaries.md#h-89-a-link-switches-the-cluster-and-the-view-under-the-user)
and [H-90](../security/trust-boundaries.md#h-90-a-links-text-reaches-agents-through-the-logs).
Not on Windows and Linux yet.

### Preferences › App: updates

The macOS app checks Cogpit's signed appcast with Sparkle 2 and offers a newer release in Sparkle's
window, installed on your click ([D-4](decisions.md#d-4-packaging-and-updates)); Help › Check for
Updates… checks now. A release a scheduled check finds shows as a button at the top bar's right
end, left of the layout toggles, not as Sparkle's window: "Update to v0.16.0" (# example: the
version the feed names), "Update" when it names none or the bar has no room for the version (a
narrow window beside a long context name); the tooltip names the version, and a click opens
Sparkle's window with the release. The button stays until Sparkle's update session ends (the
release installed or skipped, or put off with "Remind Me Later" until the next scheduled check)
([D-54](decisions.md#d-54-the-layout-toggles-and-the-update-button)). Kept in
`<user config dir>/Cogpit/preferences.json` with the other preferences.

```json
{
  "checkForUpdates": true,
  "updateChannel": "stable"
}
```

| Field | Default | Meaning |
|---|---|---|
| `checkForUpdates` | `true` # default | Sparkle checks `https://guided-traffic.github.io/cogpit/appcast.xml` every hour, and at a launch when the last check is older than that (since 2026-10-16; once a day before); off, only Help › Check for Updates… checks |
| `updateChannel` | `"stable"` # default | the only channel until Cogpit publishes prereleases; `beta` is refused for now |

Security note: an agent or `cogpit-cli` may turn `checkForUpdates` on, never off; only you can, in
Cogpit's window ([agents and MCP](../security/agents-and-mcp.md)). A build without the update key
(every local build, and every release up to v0.10.0; the releases carry the key since v0.11.0), an app still on its disk
image and other systems than macOS do not update themselves, and the row says why. `cogpit-cli`
has no updater of its own: the one inside the app is replaced with the app, one built from source
is not.
Sparkle installs an update only when its disk image and the feed carry the update key's signature
and the new app carries Cogpit's Developer ID ([updates](../security/updates.md)).

### Preferences › Agent: the chat's connections

The built-in agent's chat (⌘I) talks to the LLM connections you add here
([D-10](decisions.md#d-10-built-in-agent)); each conversation uses one, a new one the default.
Only you edit them, in the window: none of it is a command or an MCP tool. A connection's fields,
kept in `llm-connections.json` (its key apart in `llm-keys.json`, [naming](#naming-conventions)):

| Field | Default | Meaning |
|---|---|---|
| Name | – | how the chat's connection menu names it, `LM Studio on this Mac` # example |
| Kind | LM Studio # default | LM Studio, LiteLLM, Open WebUI or another OpenAI-compatible server: sets the paths (`/chat/completions` and `/models`, Open WebUI's under `/api`) |
| Base URL | `http://localhost:1234/v1` # default for LM Studio | where the API is; `https://litellm.example.com` # example. Plain `http://` to another machine (anything but `localhost`, 127.0.0.0/8 and ::1) is allowed, and the form, the list ("plain http") and the connection test warn in the warning colour that not only the key but the whole conversation and every Secret or other secret the agent reads travel unencrypted ([H-95](../security/credentials.md#h-95-an-llm-api-key-over-plain-http)) |
| API key | none # default | sent as `Authorization: Bearer …` to the base URL only (no redirects are followed); empty keeps a stored key, Remove Key removes it |
| Model | – | from the endpoint's list or typed, `qwen/qwen3.8-27b` # example |
| CA file | none # default | for an `https://` server with a private CA: an absolute path (`~/` is your home folder; Choose… opens the file dialog) to a PEM file of its CA certificates, `/Users/you/certs/team-ca.pem` # example; trusted for this connection beside the system's roots (on macOS its trust store), read whenever the chat connects, so a renewed file counts at once; refused with an `http://` base URL. Without it, a certificate macOS does not trust fails the test's HTTP and TLS layer with a hint at this field ([H-104](../security/trust-boundaries.md#h-104-a-connections-ca-file-is-trusted-beside-the-systems-roots-by-gos-verifier)) |
| Private connection | off # default | a local model or a private cloud: the chat then reads what the window shows, Secret values included; its conversations are not saved ([H-94](../security/agents-and-mcp.md#h-94-a-private-connection-gives-the-model-what-the-window-shows), [H-96](../security/agents-and-mcp.md#h-96-a-private-turns-reads-are-kept-where-the-scrubber-does-not-know-them)) |
| Send all tools | off # default | every Cogpit command the chat may run as a tool of its own with each request (179 tools, about 56,000 tokens, measured 2026-10-15), for large cloud models; off, 20 core tools and a search that finds the rest (about 7,600 tokens). The same rules either way: what the agent reads follows the private mark, every write asks as D-2 says |
| Limits per turn | 3 identical tool calls, 25 tool calls, 300,000 tokens, 10 minutes # default | a turn that reaches one stops with a note |

Test Connection checks the form's values layer by layer (TCP, HTTP and TLS, authentication, tool
use) without saving them and without cluster data.

In the chat column, the pencil at the end of the line under each of your messages opens it for
editing (since 2026-10-15, F9): Enter or Send sends it again, and that message and everything after
it are gone from the chat, from what the model is sent and from the kept conversation (the audit
log keeps it); sent unchanged, it retries a failed or stopped turn. The resent message attaches the
view and selection the input's chip shows then. While a turn runs the pencil is dimmed: stop the
turn first. The model is not told what was cut, so where you allowed agent writes for the cluster,
a write of the cut part can run again unasked
([H-100](../security/agents-and-mcp.md#h-100-a-cut-parts-tool-calls-leave-the-models-view)).

### When the connection breaks

Since 2026-10-16 ([D-58](decisions.md#d-58-standby-and-lost-connections)). Cogpit checks the
connected cluster when the Mac wakes, when its network changes (Wi-Fi, a VPN), when a watch breaks,
and every 15 seconds; two checks without an answer (one after a wake) and the connection is
**interrupted**:

- The top bar's dot turns amber and reads **Reconnecting…**; a banner over the view says which
  cluster, since when, why, and when Cogpit tries next; **Reconnect now** tries at once.
- The views, the tabs, the drawer and the dock stay as they were, showing the cluster as of the
  interruption. Everything that would change the cluster is disabled, its menus saying
  "connection interrupted"; an agent's change is refused the same way, and its reads are marked as
  the cluster's state of then.
- Cogpit reconnects by itself: at once on a wake or a network change, otherwise after 1, 2, 4, 8, 16,
  then every 30 seconds, for as long as the window is open. Back, the dot is green, a notification
  says how long it was away, and the views catch up.
- A followed log goes on by itself after a line `[cogpit] — interrupted 14:03–14:07 —`, from where it
  stopped. A port forward waits and connects again at once. A pod shell, a node shell or an attach
  ends (a process cannot be resumed): its tab says so and offers **Reconnect**, which opens a new
  one in the same tab once the cluster answers. A local terminal is not affected.
- A drain or a Helm install, upgrade or rollback that was running stops, as after a crash (a drain's
  tab says so); after the reconnect its banner offers to finish or dismiss it.
- A change you made just as the connection broke may have reached the cluster or not: after 30
  seconds without an answer it ends with "outcome unknown", and the view shows the object's state
  once Cogpit is back.

When the cluster rejects your login (it expired while the Mac slept) Cogpit lets the context's
login program renew it once by itself; if that does not help, or the program waits for you (a
browser login), the dot turns red and reads **Login expired**, and the banner's **Log in again**
runs the login program, browser included, keeping your views. Cogpit never runs it again on its own
meanwhile. A login program that changed since you allowed it is shown to you first, as at a connect.

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
| Environment variables | `COGPIT_<NAME>`; the app reads one, `COGPIT_SYSTEM_TITLEBAR=1` (macOS's own title bar instead of Cogpit's); the others are the tests': `COGPIT_KIND_KUBECONFIG`, `COGPIT_KIND_CONTEXT`, `COGPIT_KIND_DRAIN_KUBECONFIG`, `COGPIT_KIND_DRAIN_CONTEXT`, `COGPIT_KIND_FALLBACK`, `COGPIT_PNG_DIR`, `COGPIT_SIGSTORE_LIVE`, `COGPIT_SPARKLE_FRAMEWORK`, `COGPIT_KIND_METRICS_STACK` (`TestKindMetricsStack`, which installs and removes Cogpit Metrics on the kind cluster), and for a real cluster someone lends for a test run `COGPIT_REAL_KUBECONFIG` (its kubeconfig, passed to Cogpit alone), `COGPIT_REAL_CONTEXT` (default: the file's current context) and `COGPIT_REAL_INSTALL=1` (the tests that write there, `TestRealClusterMetricsStack`), never set in CI (and a few that tests set for their own child processes); for measurements on a real cluster (read-only, skipped without them, never in CI) `COGPIT_REAL_KUBECONFIG`, `COGPIT_REAL_CONTEXT`, `COGPIT_REAL_ROUNDS`, `COGPIT_REAL_IDLE` ([testing](testing.md#tiers)); for the built-in agent's local tests `COGPIT_LLM_TEST_URL`, `COGPIT_LLM_TEST_MODEL`, `COGPIT_LLM_TEST_KIND`, `COGPIT_LLM_TEST_KEY` (`TestLiveEndpoint` against a real endpoint over the demo cluster; skipped without them) | `COGPIT_LLM_TEST_URL=http://localhost:1234/v1` # example |
| MCP endpoint | `http://127.0.0.1:<port>/mcp`, a free port the first time, kept in `mcp-port` for every later start; another one, kept from then on, when the kept port is taken | `http://127.0.0.1:59449/mcp` # example |
| MCP bearer token | `<user config dir>/Cogpit/mcp-token`, owner-only (0600, directory 0700), kept across starts and the switch until you click Regenerate token in the MCP details (D-53); a file that holds no token Cogpit wrote, or that other users may read, is replaced at the next start | |
| MCP port file | `<user config dir>/Cogpit/mcp-port`, beside the token and with its rules: the port the next start binds (not with `--mcp-addr` and a port) | `59449` # example |
| MCP endpoint file | `<user config dir>/Cogpit/mcp-endpoint`, beside the token and with its rules: the endpoint's URL, for `cogpit-cli mcp`, written while the server listens and removed when it stops (on exit and when you switch the external MCP server off) | `http://127.0.0.1:59449/mcp` # example |
| MCP switch | `<user config dir>/Cogpit/mcp-switch`, beside the token and with its rules: `on` or `off`, the status bar's MCP switch, written only by a click in the window; `cogpit-cli mcp` serves nothing but a note that the server is off while it says `off` (D-49) | `off` # example; no file means on |
| Audit log | `<user config dir>/Cogpit/audit.log`, owner-only, one JSON line per agent-initiated command; read back newest first in Help › Audit Log and by `audit_list` (its last 4 MiB) | |
| Agent chat connections | `<user config dir>/Cogpit/llm-connections.json`, owner-only (0600, directory 0700): each LLM connection's id, name, kind (`lmstudio`, `litellm`, `openwebui`, `openai`), base URL, model, limits, private mark, whether it sends all tools (`allTools`, since 2026-10-15), its CA file's path (`caFile`, since 2026-10-15), whether a key was given and which one is the default; never a key ([D-10](decisions.md#d-10-built-in-agent)) | |
| Agent chat API keys | `<user config dir>/Cogpit/llm-keys.json`, owner-only (0600, directory 0700), on every platform: the connection's id to its API key, in plain text ([H-99](../security/credentials.md#h-99-llm-api-keys-are-plaintext-in-cogpits-settings-folder)); written when you save a key, a key removed with Remove Key or its connection; a file other users could access is made owner-only again at the next read, and the log says so. Never in a keychain since 2026-10-15 (the owner's decision): a key an earlier version kept in the macOS login keychain is not read, so its connection shows "no API key" until you enter the key once again; that keychain item (named `Cogpit LLM connection`, service `io.guided-traffic.cogpit.llm`) is left alone, and you can delete it in Keychain Access | `{"version":1,"keys":{"3f9a…":"sk-…"}}` # example |
| Agent chat conversations | `<user config dir>/Cogpit/chat-conversations.json`, owner-only (0600, directory 0700): the last 20 conversations per context with their items and the history the model is sent, every text with the cached Secrets' values replaced (`«secret <ns>/<name>/<key>»`), a text that could not be scrubbed `<withheld: …>`; the request bodies sent are never written, and never a conversation in which a turn ran over a private connection: it lives in memory only and is gone when Cogpit quits, and one an earlier version (v0.8.0 to v0.10.0) wrote is removed from the file at the next start ([H-96](../security/agents-and-mcp.md#h-96-a-private-turns-reads-are-kept-where-the-scrubber-does-not-know-them)) | |
| Drain journal | `<user config dir>/Cogpit/drains.json`, owner-only (0600, directory 0700): the drains Cogpit started and has not finished, offered again on the next connect | |
| Node shell journal | `<user config dir>/Cogpit/nodeshells.json`, owner-only (0600, directory 0700): the node shell pods Cogpit created and has not seen deleted (context, namespace, name, UID, the id in the pod's `cogpit.io/node-shell-id` annotation); one a crash left is deleted at the next connect to its context ([H-71](../security/privilege-footprint.md#h-71-a-crash-leaves-the-node-shells-pod-until-the-next-connect)) | |
| Hotbar | `<user config dir>/Cogpit/hotbar.json`, owner-only (0600, directory 0700): the banks of pinned clusters by context id; an unreadable one is moved to `hotbar.json.unreadable` at the first change | |
| Tabs and column layout | `<user config dir>/Cogpit/workspace.json`, owner-only (0600, directory 0700): the tab bar's tabs (context id, view, filters, search, sort, history), each view's hidden columns and column order, and since 2026-10-15 the navigator's collapsed source groups (their folders' and files' labels, A63) and whether the navigator is hidden and the chat column shown (F7, [D-54](decisions.md#d-54-the-layout-toggles-and-the-update-button)), restored at start without connecting anything ([D-41](decisions.md#d-41-what-survives-a-restart)); an unreadable one is moved to `workspace.json.unreadable` at the first change | |
| Drafts of unsaved text | `<user config dir>/Cogpit/drafts/<32 hex digits>.json`, owner-only (0600, directory 0700): an Edit or Create resource tab's unsaved text, and since 2026-10-15 a Config Map's values edited in the drawer (type `configMapData`), removed when saved, reverted or discarded, offered again at the next connect of their context ([D-11](decisions.md#d-11-crash-stance), [D-55](decisions.md#d-55-a-config-maps-values-edited-in-the-drawer)); never a Secret's values or Helm values | `drafts/3f9a…c2.json` # example |
| Helm chart repositories | Helm's own `repositories.yaml`, as the helm CLI resolves it: `$HELM_REPOSITORY_CONFIG`, else `$HELM_CONFIG_HOME/repositories.yaml`, else Helm's folder; read every 2 seconds, never written (add and remove repositories with `helm repo add` and `helm repo remove`, [D-29](decisions.md#d-29-helm-chart-repositories-from-helms-repositoriesyaml)); credentials stay in memory | `~/Library/Preferences/helm/repositories.yaml` # default on macOS |
| Helm repositories allowed over http | `<user config dir>/Cogpit/helm-http-approvals.json`, owner-only (0600, directory 0700): the repositories whose user and password you let Cogpit send over plain http, each by its name, its URL without user, password or query values, a SHA-256 of its name, URL and user name and when; no credential ([H-40](../security/credentials.md#h-40-credentials-over-plain-http)) | |
| Helm repository indexes | `<user cache dir>/Cogpit/helm/indexes/<key>-index.yaml`, owner-only (0600, directory 0700); the key a hash of the repository's name and URL. Nothing goes into Helm's own cache | `~/Library/Caches/Cogpit/helm/indexes/3f2a9c61d0b4e8a7c5f19e2d4b6a8c0e-index.yaml` # example |
| Helm chart READMEs | `<user cache dir>/Cogpit/helm/readmes/<key>.json`, owner-only (0600, directory 0700): the README of a chart version whose archive the index's digest verified, as the chart's drawer showed it (at most 256 KiB), the last 256 kept; the key a hash of the repository's name and URL, the chart, the version and the digest | `~/Library/Caches/Cogpit/helm/readmes/5d0c3f9e2a7b41c8e6f0a19b3d7e2c45.json` # example |
| Helm chart icons | `<user cache dir>/Cogpit/helm/icons/<key>.png` or `.svg`, owner-only (0600, directory 0700): a chart's icon as Cogpit checked it (a PNG of at most 256 pixels a side, or an SVG that refers to nothing outside itself), used for 7 days, the last 2,048 kept; the key a hash of the icon's URL ([D-46](decisions.md#d-46-chart-icons-from-the-hosts-that-serve-them)) | `~/Library/Caches/Cogpit/helm/icons/0b7e5d2c9a14f3e8b6c1d0a9e7f42c51.png` # example |
| Helm chart objects | kind `helmcharts`, named by the chart's id: `<repository>.<chart>`, every dot of the repository's name doubled, so that the first single dot separates the two (a repository's name: 1 to 63 letters, digits, `.`, `-` or `_`, the first a letter or a digit; another is listed as not used) | `bitnami.nginx`, `charts..example.nginx` for repository `charts.example` # example |
| App log | `<user cache dir>/Cogpit/cogpit.log`, and stderr | |
| Discovery cache | `<user cache dir>/Cogpit/discovery/<32 hex digits>.json`, owner-only (0600, directory 0700), shared by the app and `cogpit-cli` (since 2026-10-16, [D-65](decisions.md#d-65-discovery-cached-on-disk)): one file per API server and identity, named by a SHA-256 over the server URL, its TLS server name, the kubeconfig files, the context's cluster and user entry names and the user it impersonates (never a credential); it holds the server's URL and version, when the answer was fetched, the digest of the custom resource definitions it was checked against, and the groups and resources the server serves, no object. Reused for the same server version only, for 6 hours once the definitions confirmed it, else 10 minutes; dropped when the definitions change. About 140 KiB for a cluster with 230 custom resource definitions (measured 2026-10-16); deleting the folder costs the next connect its discovery requests, nothing else | `~/Library/Caches/Cogpit/discovery/9c1e4b7a2d5f8e0c3b6a9d2e5f8c1b4a.json` # example |
| Terminal kubeconfig | a temporary file per terminal holding only a context, owner-only, deleted on close ([why](../security/credentials.md#the-terminals-kubeconfig)) | |
| Downloaded `kubectl` | `<user config dir>/Cogpit/kubectl/<version>/kubectl` (0755, directories 0700), or the directory for binaries set in Preferences › Kubernetes ([why](decisions.md#d-3-helm-and-kubectl-binaries)) | `~/Library/Application Support/Cogpit/kubectl/1.36.1/kubectl` # example |
| A terminal's own directory | `<temp dir>/cogpit-terminal-<random>/`, owner-only, removed on close, only when Cogpit puts a `kubectl` first: `bin/kubectl` (a link), `zsh/.zshenv` (zsh only), `bash/rc` (bash only, since 2026-10-14) | |
| Node shell pod | `kube-system/cogpit-node-shell-<8 random characters>`, labels `app.kubernetes.io/name=cogpit-node-shell` and `app.kubernetes.io/managed-by=cogpit`, annotation `cogpit.io/node-shell-id=<random id>` (its journal entry's), deleted when the shell closes, else at the next connect to its context after a crash, else ended by its deadline, Cluster Settings › Node Shell › Maximum duration (2 hours unless the user sets 5 minutes to 24 hours; [why](../security/privilege-footprint.md#node-shell)) | `kube-system/cogpit-node-shell-k3v9q2mx` # example |
| Node shell dock tab | id `nodeShell-<n>`, title `Node shell: <node>` | `Node shell: worker-1` # example |
| Cogpit Metrics (Cluster Settings › Cogpit Metrics, [D-61](decisions.md#d-61-cogpit-metrics-the-bundled-metrics-stack)) | namespace `cogpit-metrics`, labelled `app.kubernetes.io/managed-by=cogpit` and `app.kubernetes.io/part-of=cogpit-metrics` (deleted with the last component only with these labels); Helm releases `cogpit-prometheus`, `cogpit-kube-state-metrics`, `cogpit-node-exporter`; Prometheus's Service `cogpit-prometheus-server` (port 80, `http`), which detection ranks first; cluster roles and bindings `cogpit-prometheus-server` and `cogpit-kube-state-metrics` | `cogpit-metrics/cogpit-prometheus-server:http` |
| Prometheus settings | `<user config dir>/Cogpit/metrics.json`, owner-only (0600): per context the Prometheus you named (namespace, service, port, path prefix) and, since 2026-10-16, how it is asked when not the defaults (`format`: `prometheus-operator` or `cadvisor`; `keepEmptyContainers`; `https`); never the kubeconfig | `{"prometheus":{},"query":{"kind-dev":{"https":true}}}` # example |
| Demo context | `demo` (the synthetic cluster) | |
| Local kind cluster | cluster `cogpit`, context `kind-cogpit`, kubeconfig `.cache/kind/kubeconfig` | |
| Release artifacts (planned) | `Cogpit-<version>-macos-universal.dmg`; the desktop app for Windows and Linux (M11), each with `cogpit-cli` inside | `Cogpit-0.1.0-macos-universal.dmg` # example |
| Deep links (macOS) | `cogpit://open?context=<context id>&view=<view key>&namespace=<namespace>&name=<name>`, the scheme registered in the app's `CFBundleURLTypes` (name `io.guided-traffic.cogpit`); audited as `deeplink.open` with origin `deeplink` | `cogpit://open?view=pods&namespace=team-a` # example |
| Release artifacts | `Cogpit-<version>-macos-<arch>.dmg` (signed releases; `universal` once the app is built for Intel too, planned), `Cogpit_<version>_darwin_<arch>.zip` (v0.4.0 and earlier, and builds without the signing secrets), `SHA256SUMS`; nothing else since D-16's revision (2026-10-14): `cogpit-cli_<version>_<os>_<arch>.tar.gz` (`.zip` for Windows) only up to v0.6.0 | `Cogpit-0.5.0-macos-arm64.dmg` # example |
| Deep links | `cogpit://` ([D-47](decisions.md#d-47-deep-links-cogpit)) | |
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
| `make test-unit-race`, `make test` | every test without a cluster, race detector on, the patched terminal widget's (`make test-fork`) first |
| `make test-unit` | every test without a cluster, without the race detector |
| `make test-kind` | cluster tests against the kind cluster (build tag `kind`) |
| `make test-unit-coverage`, `make test-race-ui RACE_SHARD=n RACE_SHARDS=m`, `make test-race-rest`, `make test-darwin` | CI's test jobs: the unit tests with a coverage profile; one shard of the UI's race tests; every other package's race tests; the macOS-only tests ([releasing › CI](releasing.md#ci)) |
| `make coverage-report` | the profiles in `.cache/coverage` merged into `combined.out`, the badge `.github/badges/coverage.json` and `summary.md` (CI keeps the badge current on main) |
| `make vuln`, `make gosec`, `make cyclo`, `make cyclo-report` | govulncheck over the code paths Cogpit calls; gosec over the shipped code; no function above `CYCLO_THRESHOLD`; the 20 most complex functions |
| `make parity` | the parity checklists and the numbers the progress pages report ([rules](../parity/README.md#make-parity)) |
| `make kind-drain-up`, `make test-kind-drain`, `make kind-drain-down` | a separate three-node kind cluster (`cogpit-drain`, own kubeconfig `.cache/kind-drain/kubeconfig`) for the cordon, uncordon and drain tests |
| `make bench` | the gate benchmarks (S1, S2) |
| `make lint`, `make fmt` | golangci-lint v2.14.0 (installed into `bin/`) with the team's linters and the `depguard` layering |
| `make kind-up`, `make kind-seed`, `make kind-down` | the local kind cluster and its seed |
| `make kind-prometheus`, `make kind-prometheus-down` | a real Prometheus in the kind cluster (`monitoring/prometheus-server`, the `prometheus` chart) for the drawer's charts ([hack/kind](../../hack/kind/README.md#prometheus)) |
| `make kind-kps`, `make kind-kps-down` | a minimal kube-prometheus-stack in the kind cluster (namespace `cogpit-kps`) |
| `make sigstore-root` | refresh the sigstore trusted root `internal/kubectl` embeds ([D-50](decisions.md#d-50-kubectls-signature-check)) |
| `make metrics-bundle` | download the Cogpit Metrics chart at the version `internal/helm/metricsstack.go` pins into `internal/helm/bundle`, its SHA-256 checked against the repository's index ([D-61](decisions.md#d-61-cogpit-metrics-the-bundled-metrics-stack)) |
| `make icon` | redraw `packaging/icon.png` from the current logo ([extending › Changing the logo](extending.md#changing-the-logo)) |
| `make app` | `bin/Cogpit.app` via `fyne package`, with Sparkle in `Contents/Frameworks` (`SPARKLE_PUBLIC_ED_KEY=<base64>` for an app that updates itself) and `cogpit-cli` in `Contents/MacOS` |
| `make sparkle` | the pinned Sparkle release in `.cache/sparkle/<version>`, its SHA-256 checked |
| `make showcase` | the README's pictures in `docs/images`, each in the dark and the light theme, then the pinned oxipng over them, lossless (`OXIPNG_FLAGS` without `--zopfli` for a quick look) |
| `make oxipng` | the pinned oxipng release in `.cache/oxipng/<version>`, its SHA-256 checked (macOS on Apple silicon and Linux on x86-64) |
| `make dist-app`, `make dist-macos`, `make dist-macos-sign`, `make dist-macos-notarize`, `make dist-checksums` | the release pipeline's macOS download and `SHA256SUMS` in `dist/`; `MACOS_SIGN=adhoc` for a local check ([releasing › Running the signing locally](releasing.md#running-the-signing-locally)) |
| `make appcast-draft`, `make appcast-sign`, `make appcast-verify` | the release pipeline's appcast jobs ([releasing › the appcast](releasing.md#the-appcast)) |
| `make clean` | removes `bin/` and `dist/`; `make help` lists every target |

## Development

Through the `Makefile`, as in the team's other Go repositories:

```sh
make build
make lint
make test-unit-race
make test-kind      # needs make kind-up
```

Locally, check what a change touches and leave the full gate to CI
([testing › Before a pull request](testing.md#before-a-pull-request)). CI runs each check as a job of its own with its own make target (`make gosec`, `make cyclo`,
`make test-race-ui RACE_SHARD=n`, …), and semantic-release publishes every `feat:` and `fix:` on
`main` once CI is green ([releasing](releasing.md)). Everything else:
[docs/developer/](README.md).
