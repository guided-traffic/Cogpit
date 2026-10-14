# Decisions

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-14 · **Change:** minor (D-44 decided by the owner: an agent's read takes the Secrets it is filtered against (H-82, option b); D-4: Sparkle 2 built, the third step: the framework in the app, loaded at run time, Preferences › App's two update rows and Help › Check for Updates…, the appcast drafted, signed in the environment `sparkle` and deployed to Pages; no key yet; the lead's and agent SPARKLE's decisions for the owner to confirm listed in D-4); minor (; D-47, the lead's decision for the owner to confirm: `cogpit://` links navigate only, through the `ui.*` commands, with origin `deeplink`, built; D-48, the lead's decision for the owner to confirm: start at login as the macOS app's own login item, off by default, only the user turns it on, built; D-44 (owner, H-82): the cached Secrets' values replaced in what agents read, built with the lead's choices for the owner to confirm; H-63 (owner): Flux's credential fields withheld from agents' YAML, built; D-29 (owner): credentials over plain http asked once per repository, built; D-2: an agent's role binding always waits for the user, H-27; D-44: the terminal tab's marker and the switch's 15-minute end, H-83; D-37: the node shell journal and the cleanup at connect, H-71; all three the owner's answers of 2026-10-14, built; D-4 decided by the owner: Sparkle's feed on GitHub Pages; D-4: the release pipeline signs the app and `cogpit-cli` for macOS with the Developer ID, ships the app in `Cogpit-<v>-macos-arm64.dmg` and has both notarized, built, no tag run yet; the lead's decision for the owner to confirm: a failed signature or notarization fails the macOS job and the release gets no files until it is re-run; the bodies of D-33, D-34, D-35, D-36, D-41, D-42, D-43, D-44 and D-46 no longer say "for the owner to confirm": the owner's choices of 2026-10-14 marked decided, D-33's two and D-34's two open rules marked Cogpit's defaults; D-4: the Apple signing and notarization material exists in the environment `release`, 2026-10-14; D-40 decided by the owner: the workloads' drawers' Pods tables without a Namespace column, built; D-39 decided by the owner: a pod-metrics refusal affects only its namespace, built; D-33 decided by the owner: Reconcile with Source and Reset added as the flux CLI does them, built; D-34 (owner): a state on a dashboard opens its kind's view filtered, through a status filter every view has, built with its rules listed for the owner; D-37 and D-38 decided by the owner: the default node shell image the newest supported Alpine release (3.24) pinned by digest and kept current by Renovate, the maximum duration a setting with a 2-hour default, set by the user only, built; D-35, D-36, D-41, D-42, D-43, D-44, D-46 decided by the owner as built, 2026-10-14; D-43, a night decision for the owner to confirm: the nodes' disk from node-exporter through Prometheus, never nodes/proxy, built with Cogpit's defaults listed; D-46 (night decision, for the owner to confirm): chart icons fetched when shown, with Preferences › Load chart icons on by default, built; D-11, D-37: a tab whose context leaves Cogpit closes, keeping an editor's draft and deleting a node shell's pod, Cogpit's defaults; D-44 (night decision, for the owner to confirm): agents read the dock's logs tabs freely and a terminal tab only once the user turned on its "Agents may read" switch, built; D-42, a night decision for the owner to confirm: no context picker at start, the welcome page, built; D-41, a night decision for the owner to confirm: the tabs and the tables' layout kept in `workspace.json`, restored without connecting anything, built; D-40, a night decision for the owner to confirm: a Pods table in the workloads' drawers, built with Cogpit's defaults listed; D-39: pod metrics listed per namespace for a user who may not list pods everywhere, a night decision, built; D-36 (night decision, for the owner to confirm): Argo CD's Sync writes the operation `argocd app sync` writes, bound to UID and resourceVersion; D-35 (night decision, for the owner to confirm): Argo CD's repository and cluster Secrets show an allowlist of non-credential keys; D-33, D-34: the Flux actions and the Flux dashboard, night decisions for the owner to confirm, built; D-37 and D-38, night decisions for the owner to confirm: the node shell as Lens's privileged pod, built; its image set by the user only; D-19: Create Secret built, the user's only, the values in `data` as Cogpit's default; D-16 revised (owner): releases live, a breaking change a minor release under v0; D-4: the unsigned app and the CLI archives built per release; D-32 (owner): every byte amount in binary units with one decimal, built with one formatter, the places not converted named for the owner; D-31 (owner): every timestamp in the local time zone and the region's format, built from the operating system's region, the format as a setting not built; D-25 (owner, in a real window): the drawer's copy icon right after every kind of value, a pill's included, built; D-30, the owner's answers: agents read certificates (H-45 closed), the key check and DER certificates built; D-30 (owner): certificates in Secrets decoded in the drawer and for agents, built with Cogpit's defaults; D-25 (owner, after seeing it): the lists' copy icon only outside a blacklist of columns, right after the text, built; D-29 built: Helm's repositories.yaml read every 2 s, credentials in memory only, indexes in Cogpit's own cache, add and remove removed; D-25: copy icons in the lists built, with Cogpit's defaults; D-29 (owner): Helm's repositories.yaml as the repositories' source; D-25 (owner): copy icons in the lists; D-11: drafts of unsaved editor text built with the night run's defaults, never a Secret's or Helm values (D-19); D-3 built 2026-10-13, its defaults listed; D-14: two terminal input patches added to the fork as upstream candidates, not offered; D-22 amended: the theme a preference, `--theme` an override for the run, the icon colours; D-11: Helm install built on the journal as the lead's default, still open; D-11: crash reports kept locally, built; D-26 and D-27 decided, D-27 built; D-21 built: the paste feature removed; D-13 revised to a full watch on Secrets; D-14, D-15, D-16 decided) · **Target:** Lens Desktop 2026.10.51501

Decisions that shape the code, each with its options, the reasons, and its status. A decision
moves from **open** to **decided** only by the owner; a recommendation is a proposal, not an
outcome. Ecosystem facts were observed on 2026-10-11 from the Go module proxy, crates.io and the
projects' repositories; re-check them before relying on a version number.

| Id | Decision | Status |
|---|---|---|
| [D-1](#d-1-ui-toolkit) | UI toolkit | decided: Fyne, after the spike gate; re-checks of S7 (CI, M0) and S5 (Windows and Linux, M11) can reopen it per platform |
| [D-2](#d-2-agent-interface-scope) | What agents may do over MCP | decided: everything, the UI included; destructive actions confirmed by the user |
| [D-3](#d-3-helm-and-kubectl-binaries) | Where `helm` and `kubectl` come from | decided: Helm in-process; `kubectl` from `PATH`, a verified matching download on one click |
| [D-4](#d-4-packaging-and-updates) | Packaging, signing and updates | decided for macOS: the app in a DMG, Homebrew cask, Sparkle updates; Windows and Linux with M11. Built: `cogpit-cli` per release; since 2026-10-14 the app for Apple silicon in a signed, notarized DMG and `cogpit-cli` for macOS signed and notarized (no tag run yet; before, the unsigned app zip); since 2026-10-14 Sparkle 2 in the app and the signed appcast's jobs, waiting for the owner's update key |
| [D-5](#d-5-extensions) | Extensions | decided: built into Cogpit's own code; no external extensions for now |
| [D-6](#d-6-kubernetes-libraries) | Kubernetes libraries | decided: client-go, kubectl libraries, Helm SDK |
| [D-7](#d-7-language-and-platforms) | Language and platforms | decided: Go; a macOS app first, Windows and Linux later |
| [D-8](#d-8-licence) | Licence | decided: Apache-2.0 |
| [D-9](#d-9-scope-against-lenss-paid-and-vendor-features) | Scope against Lens's paid and vendor features | decided: Flux CD and Argo CD in the beta, other paid features after it, vendor features never |
| [D-10](#d-10-built-in-agent) | Built-in agent and LLM interface | decided: chat beside the views, OpenAI-compatible endpoint; built after the alpha |
| [D-11](#d-11-crash-stance) | Crash stance | decided: crash-tolerant UI, crash-safe cluster operations |
| [D-12](#d-12-parity-sources) | Parity sources | decided: current Lens Desktop only; no OpenLens or Freelens |
| [D-13](#d-13-cache) | Cache | decided: the UI reads only from a watch-fed cache, held in memory only; Secrets in full (revised 2026-10-12, built), Helm's releases from that watch or their own `owner=helm` watch |
| [D-33](#d-33-the-flux-actions) | The Flux actions | decided (owner, 2026-10-14): Reconcile, Reconcile with Source, Reset, Suspend and Resume as the flux CLI patches them, mutating, on objects and selections; Force an agent's parameter; built 2026-10-13 and 2026-10-14 |
| [D-34](#d-34-the-flux-dashboard) | The Flux dashboard | decided (owner, 2026-10-14): five states after kstatus, counts per kind and the objects that are not ready (built 2026-10-13); a state opens its kind's view filtered, through a status filter of every view (option a), built 2026-10-14; the filter's rules are Cogpit's defaults, standing until the owner asks |
| [D-35](#d-35-what-argo-cds-repository-and-cluster-secrets-show) | What Argo CD's repository and cluster Secrets show | decided (owner, 2026-10-14): an allowlist of non-credential keys for the window and agents (option b), built 2026-10-13 |
| [D-36](#d-36-how-argo-cds-sync-writes-the-operation) | How Argo CD's Sync writes the operation | decided (owner, 2026-10-14): what `argocd app sync` writes, bound to the UID and resourceVersion (option b), built 2026-10-13 |
| [D-37](#d-37-the-node-shell) | The node shell | decided (owner, 2026-10-14): Lens's privileged pod with nsenter (option b), built 2026-10-13; the default image the newest supported Alpine release pinned by digest and kept current by Renovate, the maximum duration a setting (default 2 hours), both built 2026-10-14 |
| [D-38](#d-38-who-sets-the-node-shells-image) | Who sets the node shell's image | decided (owner, 2026-10-14): the user only, agents may reset it (option b), the maximum duration too; built 2026-10-13 and 2026-10-14 |
| [D-39](#d-39-pod-metrics-for-a-user-who-may-list-pods-in-some-namespaces-only) | Pod metrics for a user who may list pods in some namespaces only | decided (owner, 2026-10-14): one list per namespace the Pods view reads (option b); a namespace's refusal affects that namespace only; built 2026-10-13 and 2026-10-14 |
| [D-40](#d-40-a-pods-table-in-the-workloads-drawers) | A Pods table in the workloads' drawers | decided (owner, 2026-10-14): the node drawer's table in the drawers of the six workload kinds (option b), without its Namespace column; built 2026-10-13 and 2026-10-14 |
| [D-41](#d-41-what-survives-a-restart) | What survives a restart: the tabs and the tables' layout | decided (owner, 2026-10-14): Cogpit's own `workspace.json`, restored without connecting anything; built 2026-10-13 |
| [D-42](#d-42-the-start-screen) | The start screen: the welcome page, no context picker by itself | decided (owner, 2026-10-14): no picker at start (option b); built 2026-10-13 |
| [D-43](#d-43-where-the-nodes-disk-comes-from) | Where the nodes' disk comes from | decided (owner, 2026-10-14): node-exporter's root filesystem through Prometheus, never the kubelet's nodes/proxy (option b), built 2026-10-14 |
| [D-44](#d-44-what-agents-read-of-the-docks-logs-and-terminal-tabs) | What agents read of the dock's logs and terminal tabs | decided (owner, 2026-10-14): logs freely, a terminal tab only once the user turned on its switch (option b), built 2026-10-13 |
| [D-46](#d-46-chart-icons-from-the-hosts-that-serve-them) | Chart icons from the hosts that serve them | decided (owner, 2026-10-14): fetched when shown, https only, no credentials, bounded and checked, with Preferences › Load chart icons on by default (option b); built 2026-10-14 |
| [D-47](#d-47-deep-links-cogpit) | Deep links (`cogpit://`) | open (the lead's decision of 2026-10-14, for the owner to confirm): navigation only, through the `ui.*` commands with origin `deeplink`; a write unreachable by construction; never a context whose login waits for approval (option a); built 2026-10-14 |
| [D-48](#d-48-start-at-login) | Start at login | open (the lead's decision of 2026-10-14, for the owner to confirm): the macOS app's own login item (SMAppService), read from macOS, off by default, only the user turns it on (option a); built 2026-10-14 |

## D-1: UI toolkit

**Decided (owner, 2026-10-11): Fyne** (`fyne.io/fyne/v2` v2.8.1, BSD-3-Clause), after the spike
gate. The spike's plan and gate criteria are in [`spike-fyne.md`](spike-fyne.md); it ran on
2026-10-11 as the local MVP and its [results](spike-fyne.md#results) are recorded there: S1 passes
with a reservation (the start-up stall, tracked in G15), S2–S6 pass (S5 on a patched terminal
widget), S7 is partial, S8 is recorded.

**Re-checks** (owner, 2026-10-11), because the gate ran on macOS only:

- **S7** when M0's CI lands: the app cross-builds and packages for Windows and Linux on CI runners.
- **S5** at the start of M11: local shell, pod exec, resize, copy and paste on Linux (X11 and
  Wayland) and Windows.

A re-check that fails without a workaround reopens D-1 for that platform only; macOS stays on
Fyne. Accessibility (S8: the table and the drawer announce nothing to a screen reader) is a known
gap of Fyne 2.8 and the one criterion where egui, with AccessKit, would be ahead; it does not gate.

**If a re-check fails**, the engine stays in Go and the UI for that platform moves to a separate
Rust process with egui (option D of the 2026-10-11 discussion). That move needs no engine rewrite because the
engine/UI seam is a serializable protocol from the start: the CLI and the MCP server need the same
seam ([architecture §4](architecture.md#4-the-ui-seam-the-most-important-interface)).

Options considered on 2026-10-11:

| Option | State observed | Why not chosen |
|---|---|---|
| Fyne v2.8.1 | stable, 28.7k stars; tables with header rows; terminal widget `fyne-io/terminal` for Linux, macOS, Windows; accessibility experimental (`-tags accessibility`) | chosen; gaps measured by the spike |
| Gio v0.10.3, Cogent Core v0.3.45 | pre-1.0 | too low-level (Gio) or too young (Cogent Core) |
| GTK4 through gotk4 | used by Seabird (a Go Kubernetes IDE for GNOME, no push since 2025-08) | weak on macOS and Windows |
| Wails v2.16 (v3 in beta) | Go backend, web frontend in a webview | Lens's own web stack, where Freelens already is; a second language (TypeScript); three browser engines to test |
| Rust with egui | best native toolkit of the survey, AccessKit by default | the team writes Go (12 Go repositories, 7 of them Kubernetes operators; no Rust); kept as the fallback UI process |

## D-2: Agent interface scope

**Decided (owner, 2026-10-11):** the whole tool is controllable over MCP. An agent can

- read the UI state: which view is open, which namespace and filter apply, which objects are
  selected, which object the detail drawer shows;
- operate the UI: open a view, set a filter, select an object, open its drawer, open a dock tab;
- find objects ("the Service with ClusterIP 1.2.3.4") and navigate the UI to them;
- run cluster actions: every engine command, with the same verbs and danger levels as the GUI.

**Safety rules** (engine-enforced, so they bind the GUI, the CLI, external agents and the
built-in agent alike):

- Read commands and UI operations run without asking.
- `Mutating` commands from an agent show what will happen in the GUI and run after the user
  accepts, unless the user has allowed agent writes for that cluster context.
- A role binding or cluster role binding an agent creates or changes always waits for the user's
  acceptance, also where agent writes are allowed, whatever its role; the user's own command in
  the window is unchanged (owner, 2026-10-14;
  [H-27](../security/privilege-footprint.md#h-27-a-binding-is-judged-by-its-roles-name-and-the-cache)).
- `Destructive` commands from an agent always need the user's confirmation in the GUI, naming the
  objects, the namespace and the context.
- Every agent-initiated command is recorded in an audit log the user can read.

**Transport:** MCP Streamable HTTP on `127.0.0.1` with a random bearer token, written owner-only
to the app's runtime directory; a `cogpit mcp` stdio bridge for agents that only speak stdio.
SDK: `github.com/modelcontextprotocol/go-sdk` v1.8.0 (MCP specification 2026-07-28). Lens
Desktop's own MCP server is read-only and paid ([Lens MCP server](https://docs.lenshq.io/k8slens/mcp-server/));
Cogpit's is the difference. Security perspective: [agents and MCP](../security/agents-and-mcp.md).

## D-3: Helm and kubectl binaries

**Helm, decided by D-6:** Helm runs in-process through its Go SDK (`helm.sh/helm/v4` v4.3.0, or v3
v3.22.0 for clusters whose releases need it). No `helm` binary is bundled, downloaded or required.

**kubectl in terminals, decided (owner, 2026-10-11):** a terminal bound to a cluster is where users
type `kubectl` themselves, so it needs a binary that suits the cluster. `kubectl` is supported only
within one minor version, older or newer, of the API server
([version skew policy](https://kubernetes.io/releases/version-skew-policy/#kubectl)).

- The `kubectl` on the user's `PATH` is the default; Cogpit does not override a working setup.
- When `PATH` has no `kubectl`, or its minor version is more than one away from the cluster's, the
  terminal offers to download the matching version with one click. The download comes from
  `dl.k8s.io` (or a configured mirror), is verified against the published `.sha256` and `.sig`
  files, goes into Cogpit's data directory, and is put first on `PATH` in Cogpit's terminals only.
- Preferences as in Lens: download on or off, mirror, directory for binaries, path to `kubectl`.

Options weighed: `PATH` only (fails without `kubectl` or across version skew), always downloading
like Lens (about 60 MB per minor version, overrides deliberate setups), bundling one (60 MB larger
app, still skewed for other clusters, every `kubectl` fix forces a Cogpit release).

**Built 2026-10-13** ([`internal/kubectl`](../../internal/kubectl/), [`engine/kubectl.go`](../../internal/engine/kubectl.go),
[architecture §10](architecture.md#10-terminal-internalterminal)). Where it differs from the text
above, or fills what it leaves open (Cogpit's defaults, not the owner's decisions):

- The version compared is the API server's at connect against the Preferences' path to `kubectl`,
  else the first `kubectl` on Cogpit's own `PATH` (its `version --client`, read once per file); a
  login shell may compute another `PATH`.
- A release downloaded earlier that suits the cluster is put first without asking again; the
  terminal's line says so. The offer is the server's own major.minor.patch (an EKS
  `v1.36.1-eks-…` gets 1.36.1).
- The download is GUI-only (`kubectl.download` refuses agents and the CLI), from
  `https://dl.k8s.io/release` or an https mirror base; the files go into
  `<user config dir>/Cogpit/kubectl/<version>/kubectl` unless the Preferences name another
  directory. A running terminal keeps its `PATH`; the terminals opened afterwards use the release.
- The `.sig` check is sigstore's keyless signature verified with the standard library: the
  `.cert` must chain to the Fulcio certificate authority Cogpit embeds, name the Kubernetes
  release's identity and issuer, and its key must have signed the published SHA-256. sigstore's
  transparency log is not consulted ([H-35](../security/trust-boundaries.md#h-35-kubectls-signature-is-checked-without-sigstores-transparency-log)); no new
  dependency.
- "First on `PATH`" is a link in the terminal's own directory, first in its environment; for zsh
  also after the login profile ran (a `ZDOTDIR` of Cogpit's that hands over to the user's), since
  macOS's `path_helper` moves inherited entries behind `/etc/paths`. Other shells keep only the
  environment's order ([H-36](../security/trust-boundaries.md#h-36-a-shells-profile-can-put-another-kubectl-first)).

## D-4: Packaging and updates

**Decided (owner, 2026-10-11):** Cogpit shows when an update is available and installs it with one
click. Lens Desktop does the same for its exe, dmg and AppImage builds
([install](https://docs.lenshq.io/k8slens/getting-started/install-lens/)).

**Release artifact, decided (owner, 2026-10-11):** the macOS standard way, so that it works for
users who never open a terminal. The app bundle `Cogpit.app` (universal, signed) ships in a disk
image `Cogpit-<v>-macos-universal.dmg` with the usual window to drag the app into Applications; the
disk image is signed, notarized and its ticket stapled. A Homebrew cask, for users who prefer the
shell, installs from the same disk image. Sparkle updates from the same disk image too.

What one click can mean depends on how Cogpit was installed:

| Installed as | One-click update | Status |
|---|---|---|
| macOS app installed from the disk image | download, verify signature and checksum, replace the app bundle, relaunch | decided: Sparkle 2 (below) |
| Windows MSI or portable zip | download, verify, run the MSI upgrade or replace the binary, relaunch | to design |
| Linux AppImage or tar.gz | download, verify, replace the file, relaunch | to design |
| Homebrew, winget, Flathub, deb or rpm repository | the package manager owns the files; the button runs or shows the manager's command | to design |

**macOS updater, decided (owner, 2026-10-11): Sparkle 2** (2.10.0, 2026-09-13, MIT). It verifies updates with
EdDSA signatures and Apple's code signature, installs atomically, offers delta updates, handles
`/Applications` permissions, quarantine and relaunch, signs "the published update archive (dmg,
zip, etc)" and can be instantiated from toolkits other than Cocoa
([Sparkle documentation](https://sparkle-project.org/documentation/)); it needs macOS 12 or later.
The signed update manifest of these documents is then Sparkle's EdDSA-signed appcast. The Homebrew
cask declares `auto_updates true`, which tells Homebrew that the app updates itself
([Cask Cookbook](https://docs.brew.sh/Cask-Cookbook)). Alternatives weighed: an updater written in
Go (all of Sparkle's hard cases as own security-critical code), `go-selfupdate` v1.6.0 (built for
single executables in archives, so the bundle swap would still be own code), Homebrew only (no
one-click update for users who installed from the disk image). Windows and Linux are decided with M11. `fyne.io/tools`
v1.7.3 (`fyne package`) builds the app bundle. Updates are signed with a key held outside CI's reach and verified before
anything is replaced. **Status:** decided for macOS (artifact, cask, Sparkle); Windows and Linux
are open until M11.
**Built (2026-10-13), the first step:** every release up to v0.4.0 carries `Cogpit_<v>_darwin_arm64.zip` (the app
bundle from `make dist-app`: `fyne package`, ad-hoc signed, Apple silicon only) and `cogpit-cli`
archives for Linux, macOS and Windows, with `SHA256SUMS`, built by the release pipeline
([releasing](releasing.md#the-release-pipeline)). The owner's Apple credentials
exist since 2026-10-14 in the `release` environment that only the tag pipeline may use
([releasing › secrets](releasing.md#secrets)).
**Built (2026-10-14), the second step: a signed, notarized disk image**
([releasing › signing and notarization](releasing.md#signing-and-notarization)). The release
pipeline signs the app with the Developer ID (hardened runtime, Apple's timestamp, no
entitlements), puts it into `Cogpit-<v>-macos-arm64.dmg` (a folder with the app and a link to
Applications; the universal image stays a later step), signs the disk image, has it notarized and
staples its ticket; `cogpit-cli` for macOS is signed and notarized too. The release page says so
and names the certificate. No tag run has done it yet: the first release after v0.4.0 is the
proof. Not built yet: the universal binary, the Homebrew cask and Sparkle; the cask and Sparkle's
EdDSA key come after the disk image. The decisions in this step:

- **A failure fails the release's macOS job (the lead, 2026-10-14, for the owner to confirm).**
  With the secrets present, a failed signature, notarization or check fails the job "Sign and
  notarize (macOS)", and the release gets no files until the failed jobs are re-run; Apple gets 45
  minutes, then the job names the submission and the owner re-runs it once Apple has answered
  ([when it fails](releasing.md#when-signing-or-notarization-fails)). Without any of the secrets
  (a fork), the unsigned zip as before; with some of them, a failure. Options weighed: the
  unsigned zip when signing fails (a release whose files are not what a signed release promises),
  the CLI archives attached without the macOS downloads (a release that looks finished without
  its main download).
- **Signing in a job of its own (agent REL, 2026-10-14, for the owner to confirm):** the only job
  in the environment `release`; it takes the app and the archives as artifacts and runs no Go and
  no dependency's code, so nothing but Apple's tools and the script share a job with the
  identity. Option weighed: signing where the app is built (one hosted runner less, about a
  minute), where `fyne`, the module downloads and the cgo build would run beside the key.
- **The disk image replaces the zip when signed (agent REL, 2026-10-14):** one macOS download, the
  one the cask and Sparkle are to use. Options weighed: the signed app zipped beside it (two
  downloads to explain, of no use to Sparkle's disk-image updates), the unsigned zip beside it (an
  unsigned copy next to the signed one).
- **`cogpit-cli` for macOS signed and notarized, not stapled (agent REL, 2026-10-14):** the same
  identity in the same job, submitted together with the disk image. A bare binary cannot carry a
  ticket, so Gatekeeper looks it up online when a quarantined copy first runs. Options weighed:
  unsigned as before (blocked when downloaded with a browser), signed but not notarized (still
  blocked: Gatekeeper wants a ticket).
- **Only the disk image is submitted (agent REL, 2026-10-14),** the app inside it. Option weighed:
  notarizing and stapling the app before imaging it (a second submission to wait for before the
  disk image), so that the copy in Applications carries its own ticket; whether its first start
  without a network needs that is not verified, the first signed release is where to try it.

**Decided (owner, 2026-10-13):** the GitHub
environment `release` exists (deployment rule: tags `v*`; the Apple material in it since 2026-10-14) for the Apple
credentials; Sparkle's EdDSA key will be a secret of an environment with the owner as required
reviewer, so CI signs a release's update feed only after the owner approved the run (options
weighed: the key offline with the owner, signing by hand per release; deciding later).
**Decided (owner, 2026-10-14): Sparkle's feed on GitHub Pages**,
`https://guided-traffic.github.io/cogpit/appcast.xml`, the `SUFeedURL` every installed copy keeps.
Options: (a) GitHub Pages, one appcast with the releases' history and Sparkle's channels, so a
Beta channel later needs no other URL, at the cost of Pages (source: GitHub Actions) and a deploy
job; (b) a release asset under `releases/latest/download/appcast.xml`, no setup, but "latest"
leaves prereleases out, so a Beta channel would need a second URL that installed copies do not
know; (c) a rolling release with a fixed tag whose asset each version replaces, channels without
Pages, but a pseudo-release in the releases list for good. The owner chose (a): the URL is built
into every copy and cannot move without stranding the copies that have it. The appcast is signed
in a job of the environment `sparkle` (the owner its required reviewer, the private EdDSA key its
secret `SPARKLE_ED_PRIVATE_KEY`, generated by the owner with Sparkle's `generate_keys`, never seen
by an agent), after the disk image is stapled, and deployed to Pages from there.
**Built (2026-10-14), the third step: Sparkle 2** (agent SPARKLE, the owner's go-ahead for M10
part 2; [releasing › the appcast](releasing.md#the-appcast), [updates](../security/updates.md)).
Sparkle 2.10.0, the newest stable 2.x on 2026-10-14 (MIT; it needs macOS 12, Cogpit 13), pinned in
the [`Makefile`](../../Makefile) by version and the archive's SHA-256 (`make sparkle`, Renovate
proposes new versions and never merges them on its own). `make app` puts `Sparkle.framework` into
`Cogpit.app/Contents/Frameworks` and writes `SUFeedURL`, `SUPublicEDKey` (from
`SPARKLE_PUBLIC_ED_KEY`, empty by default), `SUEnableAutomaticChecks`,
`SUVerifyUpdateBeforeExtraction`, `SURequireSignedFeed` and `CFBundleVersion` into its
`Info.plist` ([`hack/macos/sparkle.sh`](../../hack/macos/sparkle.sh)). `CFBundleVersion` is the
release's version since this step: `fyne package` wrote 1 into every build, so Sparkle would never
have seen a release as newer. The binary does not link Sparkle: [`internal/update`](../../internal/update/)
loads the framework at run time from the running bundle (cgo, darwin only) and makes
`SPUStandardUpdaterController` without starting it; the updater starts once the window's event loop
runs. Without the framework, the key, an installed bundle or macOS there is no updater, and Help ›
Check for Updates… is greyed with the reason ("this build has no update key", "not installed from
the disk image", "not installed yet", "not macOS"); `cogpit-cli` has none. Preferences › App: Check
for updates automatically (on by default, as Lens) and Update channel (Stable; the row says Beta
comes with prereleases); `app.checkForUpdates` checks now with Sparkle's window. The release
pipeline signs Sparkle's parts inside-out with the Developer ID, and three jobs draft, sign
(environment `sparkle`) and deploy (GitHub Pages) the appcast. **Checked locally** (2026-10-14):
the bundle built with the framework and a throwaway key, signed ad-hoc by `make dist-macos
MACOS_SIGN=adhoc` (four Mach-O files, each with the hardened runtime, `codesign --verify --strict
--deep` on the app and in the disk image); the framework loaded and the controller made in a
harness process of its own for each bundle layout (`TestSparkleLoadsFromTheBundle`); the three
appcast jobs run with stand-ins for `gh` and the network and a throwaway Ed25519 key: Sparkle's
`sign_update` signed the disk image and the feed, `hack/appcast verify` accepted them, a second
release built on the signed feed, a tampered feed, an environment without its rules and a
prerelease (channel beta) behaved as below. **Not checked, and only a signed release can show it:**
an installed copy finding, downloading, verifying and installing an update, its relaunch through
GLFW's terminate handling, Apple's acceptance of Sparkle's parts signed with the Developer ID,
Pages serving the feed. **What remains (human):** the owner's steps in
[releasing › setting up Sparkle](releasing.md#setting-up-sparkle), in order (the environment
`sparkle` with its reviewer and tag rule, Pages and the `github-pages` tag rule, the key, the
secret, the variable); then the first release built with the variable is the first that can
update, and the release after it is the first update. Copies installed before (v0.4.0's zip, and
the first disk images, built without the key) never update themselves: one manual download.
The decisions of this step, each the lead's or agent SPARKLE's for the owner to confirm:

- **The appcast itself is signed (agent SPARKLE).** Sparkle 2.9 and later signs the feed
  (`SURequireSignedFeed`, which needs `SUVerifyUpdateBeforeExtraction`): whoever can change what
  Pages serves cannot change what Sparkle's window shows (release notes, links, a critical or
  informational mark) or point an item at another download. Cost: losing the key stops updates
  until a release rotates it through a Developer ID-signed disk image, and after 20 days without a
  valid feed signature Sparkle falls back to the items without their notes and links. Option
  weighed: the enclosures' signatures alone (Sparkle's default), where the feed itself is
  unsigned.
- **Cogpit's preference drives the automatic checks (the lead's design).** `SUEnableAutomaticChecks`
  YES in `Info.plist`, so Sparkle never asks at the second start, and the engine sets
  `automaticallyChecksForUpdates` at start when it differs and on every change. Option weighed:
  Sparkle's own prompt and setting (two places for one setting, and agents could not see it).
- **Who may change the automatic checks (the lead's default):** the user turns them on and off;
  agents and the CLI may turn them on, not off, since off leaves the copy without the next
  release's fixes (the reverse of Load chart icons). Options weighed: free for agents (an agent
  could quietly stop security updates); the user's only both ways.
- **Any origin runs `app.checkForUpdates` (agent SPARKLE):** it changes nothing and Sparkle asks
  before it installs. Option weighed: GUI-only (an agent asked to look for an update could not).
- **The Beta channel (agent SPARKLE):** refused for every origin until Cogpit publishes
  prereleases; once offered, the user's only (a prerelease is less tested code). A prerelease's
  `CFBundleVersion` (0.6.0-beta.1 is not Apple's format) needs a numeric scheme then: open.
- **No XPC services in the bundle (agent SPARKLE):** they are for sandboxed apps, and Sparkle's
  documentation allows removing them otherwise; nothing in the bundle then carries an entitlement.
  Option weighed: keeping and signing them (`Downloader.xpc` with its entitlements kept), code that
  never runs in Cogpit, signed and notarized anyway.
- **No entitlements for Sparkle's parts (agent SPARKLE):** signed as Sparkle's documentation's
  commands sign them, which drops the `com.apple.application-identifier` of `Autoupdate`'s ad-hoc
  signature. Not verified that `Autoupdate` needs nothing it had: the first update shows it.
- **Three appcast jobs (agent SPARKLE):** the draft (Go, no secret), the signature (the key and
  Sparkle's `sign_update` only), the check with the public key and the deploy. Option weighed: one
  job, where Go, its toolchain and `hack/appcast` would run beside the key.
- **The history comes from the feed on Pages, only when the key signed it (agent SPARKLE):** an
  altered feed is never signed again. Options weighed: rebuilding the history from the releases
  (every past disk image downloaded and signed again per release), the newest item alone (a copy
  the newest release does not support would get nothing). Each channel keeps its 10 newest items
  (Cogpit's default).
- **Release notes embedded as Markdown (agent SPARKLE):** the release's text up to the pipeline's
  download note, at most 32 KiB, with a link to the release; signed with the feed. Options weighed:
  `sparkle:releaseNotesLink` to the release's page (a signed feed needs a signature over the linked
  file, and GitHub's page changes), no notes.
- **Help › Check for Updates… before About Cogpit, its label with the short reason when greyed
  (agent SPARKLE):** a native menu item has no tooltip. Lens's own place for it is not captured.
**Human:** Windows code-signing material (the Apple Developer ID exists since 2026-10-14); Sparkle's
update key and the environments' rules ([releasing › setting up Sparkle](releasing.md#setting-up-sparkle)).

## D-5: Extensions

**Decided (owner, 2026-10-11): extensions are built into Cogpit's own code.** What Lens ships as
an extension (a view for a custom resource, an integration such as Flux CD or Argo CD) becomes
part of the project: kind profiles, commands and views in Cogpit's packages, reviewed and released
with Cogpit. There is no mechanism for external extensions for now; adding one later needs a new
decision.

Lens returned to extensions in 2026.10.51501 with npm packages that need a paid plan to install
([Lens extensions](https://docs.lenshq.io/k8slens/extensions/)); Cogpit has no JavaScript runtime,
so those cannot run in any case. Lens's extension features (installing from the file system,
auto-update, the extension registry, safe mode) are therefore not targets.

Options weighed: kind profiles as data files (no third-party code, but a format to maintain),
WebAssembly plug-ins with wazero (sandboxed custom logic, a large API and review effort), external
programs as plug-ins (no sandbox), Lens-compatible JavaScript extensions (impossible without a
JavaScript runtime).

## D-6: Kubernetes libraries

**Decided (owner, 2026-10-11), with D-7:**

| Need | Module | Version observed |
|---|---|---|
| API access, informers, exec, port-forward | `k8s.io/client-go` | v0.37.1 (2026-09-23; Kubernetes 1.37) |
| kubectl behaviour (drain, rollout, describe) | `k8s.io/kubectl` | v0.37.1 |
| Metrics API | `k8s.io/metrics` | v0.37.1 |
| Helm | `helm.sh/helm/v4` | v4.3.0 (2026-09-09) |

These are the libraries `kubectl` and `helm` are built from, so their behaviour (credential
plugins, drain semantics, Helm release handling) is the reference behaviour rather than a
reimplementation of it. This replaces the earlier choice of kube-rs, whose typed API reached only
Kubernetes 1.36 in a release on 2026-10-11.

## D-7: Language and platforms

**Decided (owner, 2026-10-11): Go**, replacing the earlier Rust-only decision. Reasons: the
Kubernetes ecosystem (D-6) is Go, and the team writes Go. Costs accepted: weaker native UI
toolkits than Rust (D-1 has the gate and the fallback), and fewer crash guarantees from the
compiler (D-11).

**Platforms (owner, 2026-10-11):** Cogpit is a desktop client application, installed on the user's
machine. **macOS comes first**: alpha and beta are macOS releases. **Windows and Linux** (Ubuntu
first) follow after the macOS beta ([roadmap M11](../parity/roadmap.md#milestones)). Until then the
code stays portable: platform services (PTY, keychain, paths, file dialogs) sit behind interfaces,
and CI compiles for Windows and Linux from M0 so that portability breaks are found when they are
cheap. Cogpit is never a web UI and is never deployed into a cluster; "Helm" in these documents is
Lens's feature for managing Helm releases in the user's clusters.

## D-8: Licence

**Decided:** Apache-2.0, as in [`LICENSE`](../../LICENSE) since the initial commit.

## D-9: Scope against Lens's paid and vendor features

**Context.** Lens Desktop gates twelve features behind paid plans
([premium features](https://docs.lenshq.io/k8slens/premium-features/)) and has vendor features
(Lens ID activation, subscriptions, telemetry, surveys, Teamwork's relay through Mirantis's
servers).

**Decided (owner, 2026-10-11):**

- **Flux CD and Argo CD views are in the beta.** The owner's organization keeps GitOps
  repositories for both (`k8s-flux-base`, `k8s-base-argo`), so these views are daily work there,
  not a paid extra. Their resources already appear as generic custom-resource views; the beta adds
  the curation: kind profiles with columns and status, the two dashboards, Argo CD's Sync and
  Refresh ([G23](../parity/gaps.md#g23-flux-cd-and-argo-cd-views)).
- **The other paid features come after the beta** and count toward the full target only: the EKS,
  AKS and GKE account integrations, Security Center, cost monitoring
  ([G18](../parity/gaps.md#g18-paid-lens-desktop-features)). They need cloud accounts and data
  sources and are not part of the mainstream audience's daily work.
- **Cheap ones are built anyway:** the Hotbar (with the interface, G13) and the editor's diff (with
  the editor, G9).
- **Never:** Lens ID activation, subscriptions, telemetry, surveys.
- **Teamwork** needs a hosted relay and stays out of scope unless the owner decides to run one.
- Lens's paid MCP server and Ask AI are superseded by D-2 and D-10.

Options weighed: all paid features after the beta (GitOps views late although used daily), all
paid features in the beta (150–320 h more before it, with cloud accounts and data sources), none
(Cogpit behind Lens Pro, GitOps included).

## D-10: Built-in agent

**Decided (owner, 2026-10-11):** a chat panel runs beside the views, like the chat beside the
editor in VS Code. Its agent answers questions about what the user is looking at and acts for the
user, through the same MCP tools external agents use (D-2): it reads the current view and
selection, navigates the UI ("show me the Service with ClusterIP 1.2.3.4"), and runs cluster
actions under D-2's confirmation rules. Lens Desktop's "Ask AI" only launches an external AI
command-line tool with the cluster's context
([Ask AI](https://docs.lenshq.io/k8slens/ask-ai/)); a built-in agent that operates the UI is
Cogpit's difference.

**LLM interface:** the OpenAI Chat Completions API with tool calling and streaming, the interface
both targets offer: the LiteLLM proxy ("OpenAI-Compatible": `/chat/completions`, `/models`,
[LiteLLM proxy](https://docs.litellm.ai/docs/proxy/user_keys)) and Open WebUI (`POST
/api/chat/completions`, `GET /api/models`, bearer API key,
[Open WebUI API](https://github.com/open-webui/docs/blob/main/docs/reference/api-endpoints.md)).
Settings: base URL, API key (kept in the operating system's keychain), model. Go client candidate:
`github.com/openai/openai-go/v3` v3.76.0.

**The owner's requirements, restated (2026-10-12, "so that they are not lost"):** a chat inside
Cogpit like the chat of Claude Code for VS Code, switched on and off by the user; connected to an
OpenAI-compatible API such as Open WebUI or LiteLLM, so that local models can be used; its agent
gets Cogpit's tools (the same as the MCP tools) to operate the UI and the cluster. The external
MCP server (the loopback endpoint other apps on the computer use) can be switched off while the
chat keeps working: the chat's agent does not go through that server but dispatches the
registry's commands in-process (`Session.Dispatch` with origin `BuiltInAgent`), so the status
bar's MCP switch (built 2026-10-12, [D-23](#d-23-a-link-to-the-owner-in-the-status-bar) and
[architecture §7](architecture.md#7-mcp-and-the-built-in-agent)) affects external agents only.
The owner confirmed the timing below on 2026-10-12 (the chat comes later).

**Timing:** after the alpha (milestone M10). **Open details:** what cluster data may be sent to
the LLM endpoint (Secret values never, by default); whether Open WebUI passes client-defined tools
through to every model (not verified).

## D-11: Crash stance

**Decided (owner, 2026-10-11):** a Cogpit crash is a bug, but not a catastrophe. The cluster's
state lives in the cluster and survives; PhotoCraft's "never crash, it loses the user's work"
priority does not carry over as is. Consequences:

- **The UI is crash-tolerant.** Panics are fixed like other bugs; a crash is not a release blocker
  on its own. Goroutines that run cluster I/O still recover from panics at their boundary and
  report an error, because one panicking watch must not close every open cluster.
- **Cluster operations are crash-safe.** Some operations are several API calls in a row and leave
  the cluster between states if the process dies halfway:
  - Helm install, upgrade and rollback run in-process (D-3) and leave a release in
    `pending-install`, `pending-upgrade` or `pending-rollback`, which blocks the next Helm operation
    until it is resolved;
  - drain cordons the node, then evicts pods one by one, and leaves a cordoned, half-drained node;
  - an interactive shell in a pod ends with the connection, and the processes it started may end
    with it.

  Each such operation either runs in a worker process that survives the UI, or detects the
  half-finished state on the next start and offers to finish or revert it. Which of the two, per
  operation, is designed with the operation. Decided with the operations: drain (2026-10-11) and
  Helm rollback and uninstall (2026-10-11, third run) detect it: a journal in the user's config
  directory (`drains.json`, `helm.json`, mode 0600) records the operation before it starts, and the
  next connect of that context offers to finish (run it again) or dismiss it. Helm upgrade
  (decided and built 2026-10-12, the same road): the journal records the upgrade (never its
  values), and one interrupted with the release `pending-upgrade` is offered as a rollback to the
  revision it started from, since the Helm SDK's upgrade refuses a pending release and its
  rollback does not. The journals are locked between Cogpit processes, each entry naming its
  owning process (H-22). Helm install is open; built 2026-10-13 on the same road as the lead's
  default until the owner decides: the journal records the install, never its values, and one
  interrupted with the release `pending-install` or failed is offered as an uninstall.
- **Unsaved editor content** is kept in a local draft so that a crash does not lose it. **Built
  2026-10-13 (night run)**, with defaults of that run that the owner has not decided: an Edit or
  Create resource tab's unsaved text is written 2 seconds after it stops changing (at least every
  10 seconds while it changes) to `<user config dir>/Cogpit/drafts/<id>.json` (0600, atomic,
  naming its process as the journals do, H-22), removed when the text is saved again or the tab
  is closed (closing discards it, without asking), and written on a quit; the next connect of its
  context offers it (Restore, Discard; `editor.drafts`). A tab that closes without the user, its
  context having left Cogpit (its kubeconfig source removed, its file gone), keeps its draft,
  written at once (2026-10-14, Cogpit's default, `Drafts.keep`). Never a Secret's text and never Helm
  values (D-19). [Architecture §4.4](architecture.md#44-tables-details-editors-and-streams),
  [credentials](../security/credentials.md#where-they-live-inside-cogpit).
- **Crash reports (built 2026-10-12, after a crash in the owner's smoke test left nothing to read):**
  the Go runtime writes a crash's text into `<user config dir>/Cogpit/crash.log` as well (mode 0600,
  `runtime/debug.SetCrashOutput`), which a clean exit removes; the next start moves a report left
  behind into `crashes/` (the newest ten kept) and says where it is. Nothing is uploaded
  ([credentials](../security/credentials.md)). The same day it showed that every quit on macOS
  crashed in the title bar's cleanup (fixed, [G16](../parity/gaps.md#g16-stability-and-the-user-bug-backlog)).

## D-12: Parity sources

**Decided (owner, 2026-10-11):** the target is the current Lens Desktop. Rows in the parity
checklists come from Lens Desktop's documentation and release notes, and, where those are silent,
from a capture of the running Lens Desktop. OpenLens is outdated (its source was retired in January
2024) and is not a source, and neither is its fork Freelens. No code, text or asset from Lens
Desktop, OpenLens or Freelens is used.

## D-13: Cache

**Decided (owner, 2026-10-11):** Cogpit caches intensively so that the UI is fast and the API
server carries as little load as possible while the view stays current. Every change of cluster
state is written into a cache, and the UI consumes only the cache. The cache is kept current by
watches; what cannot be watched is refreshed as often as needed and no more. The design is
[architecture §5](architecture.md#5-cache): one shared, reference-counted informer per (cluster,
kind, namespace scope), warm for a grace period after its last consumer, no resync, bookmarks,
streaming initial lists, metadata-only informers where enough, `managedFields` dropped, discovery
and access reviews cached, polling only while visible, request counters to prove the load.

**Persistence across restarts, decided (owner, 2026-10-11): in memory only.** Writing the cache to
disk would show the last known state immediately at start-up; the options weighed:

| Option | For | Against |
|---|---|---|
| In memory only (recommended) | No stale state shown as current; no cluster data at rest | Every start lists again |
| Persist all kinds | Fastest start | Secrets and other sensitive objects at rest; stale state until the first relist |
| Persist non-sensitive kinds, mark as stale until relisted | Fast start for the common views | Complexity; the line between sensitive and not is the user's, not Cogpit's |

**Decided:** in memory only. Revisit only if the measured cold start (budget P7) is too slow, and
then with this table as the starting point.

**Secrets in the cache, decided (owner, 2026-10-11): metadata only, plus Helm's release Secrets.**
Listing or watching Secrets returns their values ([H-8](../security/privilege-footprint.md#h-8-secret-data-arrives-with-the-list)),
and the Secrets view and Helm both need Secrets. The options weighed:

| Option | For | Against |
|---|---|---|
| (a) A full watch on Secrets | The Keys column as in Lens; Helm reads its releases from the same cache | Every Secret value of the watched namespaces in Cogpit's memory, masked only in the UI |
| **(b) Metadata only for the list, a GET when a Secret's drawer opens, a watch on `owner=helm` for Helm (decided)** | Only Helm's release payloads, which Helm needs anyway, are held as data; one Secret's data is fetched when the user opens it | The Keys column cannot be filled from metadata: a documented difference from Lens |
| (c) No cache for Secrets, a GET whenever needed | Nothing held | Breaks cache-first: the list costs a request per view |

**Built (2026-10-11, third run)** as decided below, with one difference the decision did not
foresee: the Type column cannot be filled from metadata either (`type` is a top-level field), so
Keys and Type both stay empty. The server-side Table format (`as=Table`, `includeObject=Metadata`)
would carry the type and the number of keys without values, at the cost of an informer that holds
table rows and a Keys column that counts instead of naming; whether to use it is open for the
owner. A second finding: the `kubectl.kubernetes.io/last-applied-configuration` annotation repeats
a Secret's values in its metadata; the cache drops it from every Secret before the store.

**Revised (owner, 2026-10-12): (a), a full watch on Secrets.** The Secrets view will list and
watch Secrets in full, so that its Keys and Type columns are filled as in Lens. This revokes the
choice of (b) below, which is built (2026-10-11) and stays until (a) is: the full scope replaces the
metadata-only one, the drawer reads the opened Secret from the cache instead of its own GET, and
Helm's releases can come from the same informer while it runs (the `owner=helm` scope stays for when
it does not, so that the Releases view alone never holds every Secret). Every value of the watched
namespaces is then in Cogpit's memory while the scope runs, its warm period included; the UI keeps
values masked until the user reveals one, and agents keep getting them redacted
([H-8](../security/privilege-footprint.md#h-8-secret-data-arrives-with-the-list)). Whether Secrets
can then be edited from YAML is a separate decision, not made.

**Built (2026-10-12, fourth run)** as revised: the Secrets view lists and watches every Secret in
full (Keys and Type filled), the drawer reads the opened Secret from the cache (the GET is gone),
values stay masked and `secret.reveal` GUI-only, and Helm's releases read from the full watch while
it runs and from their own `owner=helm` watch otherwise, through the covering scopes of
[architecture §5](architecture.md#scopes-local-and-virtual-kinds). Edit and Create from YAML stay
refused for Secrets.

**Decided before (owner, 2026-10-11, revised above):** (b). The Secrets view lists through a metadata-only informer
(`PartialObjectMetadata`); the drawer's GET of one Secret is the direct read architecture §5
allows for data the cache does not hold, and its values stay masked until the user reveals a
field; Helm's releases come from a label-selected informer (`owner=helm`) whose Secrets are held in
full. RBAC cannot narrow `list secrets` by label, so the verbs are unchanged.

## D-14: Terminal widget

**Decided (owner, 2026-10-12):** keep the patched copy of `fyne-io/terminal` in
`third_party/fyne-terminal`, offer its patches upstream, and fuzz the widget on its own, without
Cogpit's escape filter in front of it ([H-19](../security/trust-boundaries.md#h-19-the-terminal-widget-trusts-its-input)).
The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Keep the fork, offer the patches upstream, fuzz the widget alone (decided)** | Closes the largest open risk, inputs nobody has found yet, for a few hours; the patches are small and general | The fork stays to maintain until upstream merges equivalent fixes |
| (b) Keep the fork as it is | Nothing to do | Unknown crashing inputs stay untested; the fork drifts from upstream |
| (c) Cogpit's own terminal widget on a maintained VT emulator | Full control over hostile input, colours from the theme | 20–40 h; replaces tested code; budget P5 to prove again |
| (d) Back to upstream without the patches | – | The known crashes come back |

Upstream's head asks in its `go.mod` for an unreleased Fyne (v2.8.2), but its code builds and passes
its tests on Fyne 2.8.1: Cogpit's copy moved onto it on 2026-10-12 and keeps its `go.mod` at 2.8.1
([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). Returning to upstream waits for the
patches to be merged; (c) is the step to take only if fuzzing shows the
parser cannot be kept. Two inputs still made the patched widget alone panic when this was decided
(a CSI whose final character is not ASCII, a set mode without `?`; only the filter caught them):
they were the first patches to add.

**Built (2026-10-12):** both patched, and the widget fuzzed alone (`FuzzHandleOutput` in the copy,
`FuzzRawWidget` in `termview`); the fuzzing and the review that came with it found and fixed a
panic with debug output on, five sequences whose counts made the widget loop for as long as the
count, strings parsed in quadratic time, an APC that never ended, and data races between its two
goroutines ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). Its tests and fuzz seeds run
in `make test-unit-race` and CI (`make test-fork`). The patch series is offered upstream since 2026-10-12 (owner's OK): pull requests [#155](https://github.com/fyne-io/terminal/pull/155) (crashes), [#156](https://github.com/fyne-io/terminal/pull/156) (hangs), [#157](https://github.com/fyne-io/terminal/pull/157) (OSC 7) and [#158](https://github.com/fyne-io/terminal/pull/158) (races and the fuzz test), 20 commits, each fix with a test that failed before it. The fork stays until upstream has equivalent fixes. Nothing found so far calls for option (c).

**Decided how (owner, 2026-10-12):** the lead prepares the upstream series as three or four grouped
pull requests (crashes, hangs and performance, races, OSC 7), each fix one commit with a test in
upstream's style; the owner looks at the series once; after the owner's OK the lead opens the pull
requests under the owner's GitHub account. Nothing is published before that OK. The options
weighed: the lead opens them after the owner's OK (decided), the owner opens them, issues with
reproducing inputs instead of pull requests, or keeping the fork for good (which would reverse
the decision above).

**Added (2026-10-13, night run; candidates, not offered):** two patches for terminal input, each
with a test in the copy, marked `Cogpit patch:`: what the right mouse button does is the
application's choice (`SetRightClickAction`: paste the last selection, upstream's behaviour and
the default; nothing; or copy the selection, else paste the clipboard), which Cogpit's terminal
preference needs; and ⌃Ins and ⇧Ins copy to and paste from the clipboard off macOS instead of the
widget's own selection buffer. A variable `goos` lets the tests on one system cover the keys of
the others ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). They are offered upstream
like the others once the owner looks at them.

## D-15: Progress generators

**Decided (owner, 2026-10-12):** `make parity` is built first, as a counter and checker of the
hand-written checklists that runs in CI: it counts the views, actions and interface rows the
progress pages report, and fails on a row that appears twice, a row marked partial or done without
evidence, and a command id in a row that the registry does not know. The generated
`parity-checklist.md` and the scorecard (`make scorecard`) come later, the scorecard once
performance budgets are measured regularly on a fixed runner. The options weighed: both generators
as planned (6–14 h, a scorecard with hardly any data yet), the checker alone (3–6 h, decided),
counting by hand for good (0 h), after the alpha (0 h now). Why the checker first: merging the
third run's eight branches put the same parity rows in twice, twice, and the counts needed
throwaway scripts whose denominator (121 or 123 free interface rows) could not be reproduced.

**Built (2026-10-12):** [`hack/parity`](../../hack/parity/main.go), run by `make parity` as a CI
step after the lint; it also refuses links to missing files and unknown test names, and compares
its counts with the numbers ROADMAP and the progress pages report. The 121 holds: the 123
counted the two paid keyboard shortcuts, whose table had no Plan column
([rules](../parity/README.md#make-parity)).

## D-16: The first real release

**Revised (owner, 2026-10-13): releases are live now.** After every gate on `main` is green,
semantic-release creates the tag and the GitHub release with the organisation's GitHub App token,
and the release pipeline (`build.yml`, `on: release: published`) builds and attaches the artifacts:
`cogpit-cli` for six platforms, the unsigned `Cogpit.app` for Apple silicon, `SHA256SUMS`
([releasing](releasing.md#release-flow)). The first releases are minor releases under v0, so the
owner can judge whether the pipeline runs end to end; a breaking change makes a minor release, not
1.0.0, until the owner decides on 1.0 (`releaseRules` in [`.releaserc.json`](../../.releaserc.json),
needed at once: commit `f3ad000` has a body line starting with `BREAKING CHANGE:`, which made the
dry run's next version 1.0.0). Signing and notarization follow now that the Apple credentials exist
(2026-10-14, [D-4](#d-4-packaging-and-updates)). **Dependency updates (owner, 2026-10-13):** Renovate commits
them as `build(deps):`, so they make no release of their own and ship with the next `feat:` or
`fix:`; the options weighed: `fix:` and a patch release per update (the team's convention), or
`fix:` only for security updates.

**Superseded (owner, 2026-10-12):** semantic-release stays a dry run until the alpha. The alpha's
release is the first real one, cut together with the job that builds the unsigned macOS app as a
download, so that the first version (0.1.0) has something to install; that job is part of the
alpha's work ([releasing](releasing.md#release-flow),
[G14](../parity/gaps.md#g14-packaging-signing-and-updates-macos)). The options weighed: the dry run
until the alpha (decided), releasing now as the team's other repositories do (a release on every
push to `main`, without downloads), releasing with a minimal artifact job now, or releasing by hand
once per run.

## D-17: Token files named by added kubeconfigs

**Decided (owner, 2026-10-12):** a context of an added or synced kubeconfig whose user names
a `tokenFile` outside the kubeconfig's own folder connects only after the user allowed it, in the
same GUI dialog and with the same approval store as a credential plugin
([H-24](../security/trust-boundaries.md#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token),
[H-1](../security/trust-boundaries.md#h-1-kubeconfig-files-added-by-sync-folders)): the dialog
names the file and the server its content goes to, the approval is bound to the path (and, as for
plugins, to the server and its trust), and only the GUI can give it. A token file in the
kubeconfig's own folder comes from the same source as the kubeconfig and needs no approval; the
default kubeconfig keeps kubectl's behaviour. **Built 2026-10-12** (`cluster.BindPluginFiles`,
`checkTokenFilePolicy`, [`kubeconfig/exec.go`](../../internal/kubeconfig/exec.go)); what remains is
in H-24. The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Ask, as for plugins (decided)** | Closes the gap without breaking a legitimate setup; one mechanism for everything a kubeconfig does on the user's computer; 2–4 h | One more dialog for setups that use token files |
| (b) Refuse for synced files only | No dialog | A legitimate synced setup breaks without recourse; added kubeconfigs stay open |
| (c) Refuse every file reference of an added kubeconfig | Strictest | Breaks the many kubeconfigs that name certificate files, which send nothing that matters (a client key is never sent; a client certificate only in the TLS handshake) |
| (d) Document only | Nothing to do | The gap stays live from the moment a file lands in a synced folder |

## D-18: Secret annotations seen by agents

**Decided (owner, 2026-10-12):** agents keep seeing a Secret's annotations in full (except
kubectl's `last-applied-configuration`, which the cache drops); the risk that a tool copies a
Secret's manifest, values included, into an annotation is documented and accepted
([H-23](../security/agents-and-mcp.md#h-23-secret-annotations-other-than-kubectls-reach-agents)).
Only `object_get` shows annotations to agents; lists, CSV and search do not. The options weighed:

| Option | For | Against |
|---|---|---|
| (a) Withhold every Secret annotation's value from agents | Safe without knowing the tools; 1–2 h | Agents lose harmless values (a certificate's issuer) |
| (b) An allowlist of harmless annotations | More useful to agents | A list to maintain; 2–3 h |
| (c) A denylist of known manifest copies | Keeps everything else | Never complete: an unknown tool slips through |
| **(d) Document only (decided)** | Agents see what the user sees in the drawer; nothing to build | A Secret deployed by a tool that copies its manifest into an annotation reaches agents with its values |

## D-19: Editing and creating Secrets from YAML

**Decided (owner, 2026-10-12):** Secrets can be edited and created from YAML in the GUI only. An Edit
or Create resource tab that holds a Secret shows the user its values (base64 in `data`, plain text
in `stringData`) and works like any other editor tab (diff, server-side dry run, the
resourceVersion precondition); agents and the CLI can neither open such a tab, nor read its text
(`<withheld>`), nor apply or create a Secret; values never reach the audit log, an error or a
toast. The same pattern as `secret.reveal` and the Helm Upgrade tab. Lens's Secret editor is not
captured (inferred). **Built 2026-10-12** ([`secretedit.go`](../../internal/engine/secretedit.go)).
Since 2026-10-13 (the drafts of D-11, a default of the night run that follows from this decision):
a Secret's tab, and any tab whose text holds a Secret's manifest or may be one being typed, has no
draft on disk ([`drafts.go`](../../internal/engine/drafts.go), `TestNoDraftsForSecretsOrHelmValues`).
Since 2026-10-13 (night build, G9) the same holds for Lens's Create Secret dialog: `secret.create`
is an MCP tool like every command but refuses every origin but the GUI before anything is asked or
sent, and the audit line of a refused call names the Secret, its type and its keys, never a value
([`secretcreate.go`](../../internal/engine/secretcreate.go), `TestSecretCreateIsGUIOnly`). Cogpit's
default, not the owner's decision: the dialog sends the values base64-encoded in `data`, not in
`stringData` (options: (a) `data`, (b) `stringData`; (a) because what is sent is what the API server
stores and returns, as `kubectl create secret` sends it, and the scrubbing of an API message knows
each value in both forms). The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) GUI only (decided)** | Lens's Edit for the user; values never reach an agent | Agents cannot change a Secret |
| (b) Agents too, writing values they bring without reading existing ones | Agents can rotate credentials | Keeping withheld values on save is complex; an agent can overwrite credentials |
| (c) Metadata only (labels, annotations) | Little risk | Half of Lens's Edit |
| (d) Keep refusing | Nothing to build | Users need kubectl to change a Secret |

## D-20: A port forward that cannot connect

**Decided (owner, 2026-10-12):** a forward whose connection to the pod fails with an error that
does not go away by waiting (forbidden, unauthorized, the namespace gone) turns Failed at once,
releases its local port and names the cause; an error that can pass (the network, a pod still
starting, a server error) is retried with the backoff as today. Today a forward that first waited
for a pod and then could not connect stays Reconnecting and retries for ever
([`portforward_run.go`](../../internal/engine/portforward_run.go)). Lens's behaviour is not
captured. **Built 2026-10-12** ([`cluster/portforward_errors.go`](../../internal/cluster/portforward_errors.go),
`TestPortForwardFailsAtOncePermanentErrors`, `TestDialErrorsAreClassified`). The permanent errors,
as the API server answers the portforward request: **404 Not Found** (the pod does not exist, or
its namespace is gone), **403 Forbidden** (no `create` on `pods/portforward`) and **401
Unauthorized** (credentials rejected after client-go's refresh). client-go's SPDY dialer returns
those as a plain error that keeps the message and loses the status, so Cogpit records the status of
the dialer's last response (`DialRefusedError`). Retried as before: a network error, a dial
timeout, 5xx, 429, and 400 (the answer while a pod is not running yet), and an error without a
status (a proxy resetting the connection). A refused connection to the pod's port is not a dial
error and stays a reconnect (the application may be starting); a pod or namespace that vanishes
from the cache while the forward waits keeps it Waiting, since the cache cannot tell a deleted pod
from one a controller recreates. The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Permanent errors fail at once (decided)** | A clear state, no pointless requests, the port freed; passing errors still bridged | Which errors count as permanent is a list to keep right |
| (b) Fail after N attempts that never connected | Simple | A slow pod start kills the forward |
| (c) Retry for ever with a distinct "Failing" state | Never gives up | The load and the occupied port stay |
| (d) Leave it | Nothing to build | "Reconnecting: forbidden" for ever |

## D-21: Pasting a kubeconfig

**Decided (owner, 2026-10-12), built the same day:** Cogpit does not add kubeconfigs from pasted
text. Lens's "Add Kubeconfigs" menu offers "Paste from clipboard" beside syncing files and
folders (captured 2026-10-12); Cogpit offers files and sync folders only. The paste dialog, the
`kubeconfig.paste` command and the owner-only store of pasted kubeconfigs built earlier on
2026-10-12 are removed, with their tests and the documentation that described them. What happens
to what an earlier build wrote: Cogpit no longer lists or reads a `pasted` source and leaves its
file under `<user config dir>/Cogpit/kubeconfigs/` alone, because it is the user's file and
deleting user files silently is not acceptable; the entry leaves `kubeconfigs.json` with its next
change ([architecture §6](architecture.md#kubeconfig-sources-built-2026-10-12)). The options weighed: keeping
Cogpit's paste dialog without copying Lens's, removing it (decided), or reading the clipboard
directly as Lens's menu item does.

## D-22: Colour theme

**Decided (owner, 2026-10-12):** Cogpit's dark theme takes the colours of VS Code's built-in
**Solarized Dark**, the theme the owner works in, and its light theme those of **Solarized Light**;
`--theme dark|light|system` stays. The colours are copied into Cogpit's theme tokens once, not read
from VS Code at run time (which would tie Cogpit to VS Code's files and to themes whose colours
need not suit Cogpit's status tokens). Source: VS Code's `theme-solarized-dark` and
`theme-solarized-light` (MIT; the palette is Ethan Schoonover's Solarized, MIT), credited in
`ATTRIBUTION.md`. **Built 2026-10-12** in [`theme.go`](../../internal/ui/fyneui/theme.go) (each
value carries a comment naming its VS Code key) and, for the terminal's sixteen colours, in
[`termview.go`](../../internal/ui/fyneui/termview/termview.go) through a patch to the terminal
widget ([PATCHES.md](../../third_party/fyne-terminal/PATCHES.md)). Where Solarized's own contrast
is too low for text, the build departs from the table, always in the direction of legibility, and
`TestPaletteContrast` holds the floors: the light foreground is `sideBarTitle.foreground`
(#586E75) because `editor.foreground` (#657B83) is 4.1:1 on the light view; the light accent,
buttons and focus are a darkened cyan (#1F7F78, links #1B6761) instead of the light file's tan
`focusBorder` and olive `button.background`, to match the cyan of the logo and carry text; status
tokens are the accents lightened (dark) or darkened (light) until they read at 3:1 on the views,
the navigator, the hover row and the header, and at 2.5:1 on a selected row; the text of a selected
row is base2 (dark) or base02 (light), a new token, because the foreground is 2.4:1 on
#005A6F; the dark tooltip and toast are #003847 and the light ones #EEE8D5 where VS Code's hover
widget colours (#004052, #CCC4B0) are 3.6:1 and 3.1:1 under the text; the light dock strip is
#DDD6C1. The two dark terminal colours black and bright black (#073642, #002B36) are VS Code's,
and bright black is invisible on the dark view (the session-ended notice therefore uses bright
yellow, #657B83). The mapping read from Solarized Dark (2026-10-12; Solarized Light's file gives
the light values the same way):

| Cogpit surface | VS Code key | Solarized Dark |
|---|---|---|
| Views and editors | `editor.background` | #002B36 |
| Navigator, drawer, status bar | `sideBar.background`, `statusBar.background` | #00212B |
| Header and title bar | `activityBar.background`, `titleBar.activeBackground` | #003847, #002C39 |
| Tabs, inactive and active | `tab.inactiveBackground`, `tab.activeBackground`, `tab.activeForeground` | #004052, #002B37, #D6DBDB |
| Text | `editor.foreground`, `sideBarTitle.foreground` | #839496, #93A1A1 |
| Accent, focus, buttons | `focusBorder`, `button.background` | #2AA198 (cyan) |
| Selected and hovered row | `list.activeSelectionBackground`, `list.hoverBackground` | #005A6F, #004454 |
| Inputs | `input.background` | #003847 |
| Status tokens | the Solarized accents | green #859900, yellow #B58900, orange #CB4B16, red #DC322F, blue #268BD2, magenta #D33682, violet #6C71C4 |
| Terminal | `terminal.ansi*` | the theme's sixteen colours |

**Amended 2026-10-13 (G13, Cogpit's default, not the owner's decision):** the theme is also a
preference (Preferences › App › Theme: Sync with computer, Dark, Light, Lens's three), switched at
once and kept in `preferences.json`; `--theme` stays and overrides it for one run; Dark stays the
default until the user picks another. A cluster's icon colour (Cluster settings › General) is one
of Solarized's eight accents, in each variant's legible shade (`identityColors` in
[`theme.go`](../../internal/ui/fyneui/theme.go), 3:1 held by `TestIdentityColorsContrast`).

## D-23: A link to the owner in the status bar

**Decided (owner, 2026-10-12):** the status bar shows, always (before connecting and on the demo
cluster too), "Need a Kubernetes Expert? Visit: hans-fischer.com"; a click opens
https://hans-fischer.com in the user's default browser. Nothing is sent or tracked; only the
click opens it. When the bar is too narrow, the link gives way first, so that the context and the
connection state stay readable. It is a GUI link, not an engine command: a documented exception
to "everything is a command", so that no agent can open the page. **Built 2026-10-12** in
[`bars.go`](../../internal/ui/fyneui/bars.go) (`statusLayout`): the link is a status bar item with a
divider; the bar's minimum width is the status items' alone, and the link takes the room left,
is shortened with an ellipsis and, below the width of "Need a…", hidden with its divider. It opens
through `openBrowser`, the function the port-forward view uses.

**Moved (owner, 2026-10-12):** the link sits at the right end of the bar, directly left of the MCP
item (and ends the bar under `--no-mcp`, without a divider). The other items stay at the left end
with their widths; the link still gives way first.

**The MCP item is a switch (owner, 2026-10-12).** A click on the status bar's MCP item switches the
*external* MCP server, the loopback HTTP endpoint other programs on this computer reach with the
bearer token, off and on. Off: the endpoint stops listening, every connection and every pending
confirmation of an agent ends (its command does not run), the token file is removed, the item shows
a muted, struck pill with the tooltip "External MCP server (for other apps on this computer): off —
click to start", and the top bar's MCP button and Help › Copy MCP command are disabled, so no stale
command is offered. On: the server starts again with a new token and port (the registration
command changes; the copy controls give the new one) and the tooltip names the URL. Like the link
it is a GUI control and not an engine command or an MCP tool: an agent cannot switch the server on
(`TestMCPSwitchIsNoCommand`). Nothing else is coupled to it: the engine and an in-process agent
(D-10, origin `BuiltInAgent`) keep working while the external server is off. The user's switch is
one line in the audit log (origin `gui`, command `gui.mcpServer`, `{"enabled":true|false}`).
**Persistence (built as recommended; the owner may revise):** the choice is stored in Fyne's
preferences (`mcp.enabled`, written by a click that worked) and read at the next launch; no stored
choice means on, as before. `--no-mcp` still wins (no server, no item), and `--mcp-addr` only sets
the address; there is no environment variable. A failed start at launch shows the item off with the
reason in its tooltip and leaves the stored choice alone.

## D-24: Logo

**Decided (owner, 2026-10-12):** Cogpit's logo is the "C helm": an open C (its opening to the right,
75°, the C drawn heavy) around a ship's wheel with eight spokes, an inner ring and a hexagon hub; a
needle fills the opening where the spokes stop; bar handles stand outside the C, and a thin outer
ring closes the wheel. Cyan and Solarized base1 on Solarized Dark's base03, in a macOS squircle
plate for the app icon, and without the plate as the 20-pixel mark in the header bar, drawn with
heavier lines there. It replaces the ring with a dot, which the owner found too plain. Chosen in an
interactive lab among four directions (an instrument dial, a cog with a visor, the C helm, a radar)
and twelve variations of the C helm; the lab's settings were
`{"concept":"helm","accent":"cyan","second":"cyan","plate":"base03","shape":"squircle","stroke":2,"spokes":8,"gap":75,"handles":1,"rotate":0,"weight":2.8,"ring":1,"outer":1,"pointer":1,"hexhub":1,"bars":1}`.
The exact geometry is the reference: [`assets/cogpit-icon.svg`](assets/cogpit-icon.svg) (the app
icon) and [`assets/cogpit-mark-dark.svg`](assets/cogpit-mark-dark.svg),
[`assets/cogpit-mark-light.svg`](assets/cogpit-mark-light.svg) (the mark on dark and light bars).
**Built 2026-10-12:** both are drawn from the reference's geometry and settings by one rasteriser,
[`internal/ui/logo`](../../internal/ui/logo/logo.go) (pure Go, no Fyne, so that the icon generator
and the header share it; its test checks the stroke widths against the three SVGs). The app icon
(`make icon` → `hack/icon` → `packaging/icon.png`) carries its own plate with transparent corners
and the margin of macOS icons (100 of 1024 pixels, the radius `fyne package` would mask), so `make app`
passes `--use-raw-icon`; without it `fyne package` shrinks and masks an icon a second time. The
header's mark ([`bars.go`](../../internal/ui/fyneui/bars.go), `newLogo`) is drawn at the pixel size the
canvas shows it at, 20 units (`sizeLogo`), in the heavy weights, with the theme's `logo` tokens
(base1 and cyan on the dark theme, base01 and cyan on the light one, as the reference's two marks).

## D-25: Copying values

**Decided (owner, 2026-10-12):** text in Cogpit can be selected and copied, above all in the detail
drawer, where everything is selectable (values, labels, annotations, events, notes, table cells).
Every value in the drawer gets a copy icon, shown on hover, that puts it on the clipboard and says
so in a toast; a value made of several entries (labels, annotations, ports, images) copies the one
entry, in YAML syntax (`app.kubernetes.io/name: grafana`). A Secret's value can be copied only once
revealed, in the GUI, and the clipboard is cleared 60 seconds later if it still holds that value.
Agents get no copy command (they read values through `object_get` already). This is Cogpit's own
approach; how Lens Desktop copies is not captured and not needed.

**Added (owner, 2026-10-12, H-25):** on macOS a copied Secret value is marked for clipboard
managers to skip (nspasteboard.org's concealed and transient types) and kept to this Mac (no
Universal Clipboard); the 60 seconds and the clearing on close stay. Options weighed: marking it
(decided), leaving it as built, a shorter time, no copying of Secret values at all. **Built 2026-10-12**
([`selection.go`](../../internal/ui/fyneui/selection.go), [`copy.go`](../../internal/ui/fyneui/copy.go),
[`secretreveal.go`](../../internal/ui/fyneui/secretreveal.go)):

- The drawer's text lines (names, values, table cells, section titles, events' reason and
  source) are `selLine`s: a drag selects, a double click selects a word, ⌘C or Ctrl+C copies, all
  inside one line; Tab does not stop at them (Fyne's focus chain skips a "disabled" object that has
  `SelectedText`, and a click still focuses it). An event's message is a selectable `widget.Label`.
  A revealed Secret value is a `secretText`: monospace, wrapped at any character, selectable over
  its lines. Not drag-selectable, because they are tappable or drawn as shapes: links, chips and
  status pills (they have the copy icon).
- One copy icon (`copyButton`), which the drawer moves to the value the pointer is over. It sits
  right after the value's text, or on the value's right end when the value is wider; on a chip or
  a pill it sits right after the chip, outside it and vertically centred (a chip with a neighbour
  in its row keeps a gap for it), so it covers no chip text. When the drawer rebuilds under a
  stationary pointer (a watch event), the icon is shown again at the value that now covers the
  place of the old one, without a new `MouseIn` (`keepCopy`); it goes when the pointer moves
  away. A tap puts the value on the clipboard and shows a toast
  ("Copied to the clipboard"). Values of several entries copy the entry under the pointer: a folded
  list has no icon while folded; an entry of the form `key=value` (labels, annotations, selectors,
  environment variables) is copied as `key: value`, any other entry as a YAML scalar, quoted only
  where YAML needs it (`yamlEntry`, through sigs.k8s.io/yaml). Single values are copied as shown.
- **Reach and confirmation (owner, 2026-10-12):** every chip owns the room for its icon (the
  `hoverWrap` gutter), the last of a row and one alone on its line too: the icon sits after the
  chip, and the pointer's way from the chip to it stays inside the chip's hover area. Before, only
  a chip with a right neighbour had that room, so the icon of a label chip hid (its `MouseOut`)
  before the pointer reached it, while the icon of a Pod IP, whose area spans the whole value
  cell, could be reached. After a copy the icon is a checkmark in the OK colour for 1.5 s
  (`copyConfirmTime`, the shell's `copyConfirm`), then the copy icon again while the pointer is on
  the value; a Secret's value confirms the same way. `TestCopyIconReachableOnLabelChips`,
  `TestCopyIconReachableOnWrappedChips`, `TestCopyIconTurnsIntoACheckmark`.
- **Right after the value (owner, 2026-10-13, in a real window):** "The copy button at Not After
  has wandered to the far right again": a TLS Secret's Not After is a status pill, and the icon of
  a pill sat at the drawer's right edge. A pill's wrapper (`hoverWrap`) spans the whole value cell,
  and the icon was placed after the wrapper instead of after what it draws; an event's message did
  the same. **Built 2026-10-13:** the icon follows what a value draws (`drawnWidth`: a pill or a
  chip as wide as it asks, a message to the end of its longest line), a padding after it, for
  every kind of value: a line, a link, a revealed Secret value, a pill, a chip (labels,
  annotations, a list's entries), an event's message, a line of a value of several lines (one icon
  per line, as before), a cell of a drawer table. Clamped to the value, never over another one:
  a line or a link whose text leaves no room sits at the value's end and its text is shortened
  with "…" before the icon while it is there (`iconRoom`, as in the lists; the text comes back
  when the icon goes, and stays short while the pointer is on the icon); a pill or a chip that
  reaches the details' right edge keeps the icon at that edge; a message that wraps keeps it at
  its top right. A value with action buttons (a container port with Forward…) owns the room for
  its icon after its text, so the icon sits before the buttons, which move right by the icon's
  width, as a chip owns its gutter (Cogpit's default, not the owner's decision).
  `TestDrawerCopyIconAfterThePill`, `TestDrawerCopyIconAfterTheText`,
  `TestDrawerCopyIconAfterTheMessage`, `TestDrawerCopyIconInAnActionRow`; renders
  `TestRenderDrawerCopyIconAfterTheValue` (PNG [Not After](../../internal/ui/fyneui/testdata/drawer_copy_notafter_dark.png),
  [a long fingerprint](../../internal/ui/fyneui/testdata/drawer_copy_long_light.png)), and the pod's
  status pill in [`drawer_copy_pill_dark.png`](../../internal/ui/fyneui/testdata/drawer_copy_pill_dark.png).
  Not tried in a real window again.
- A masked Secret value has no icon; a revealed one has, and copies the value alone (not the note
  that a long value was cut). A copy of a Secret value, by the icon or by ⌘C on its selection, goes
  through `shell.copyText(…, secret=true)`: 60 seconds later (`secretClipboardTTL`; the shell's
  `secretClipTTL`, shortened in tests) the clipboard is emptied if it still holds that value; a
  later copy of anything cancels the clearing, and closing the window clears at once
  (`w.SetOnClosed`).
- It is no command and no tool: `TestCopyHasNoCommand`. The one copy command, `object.copyName`
  (the drawer's first icon), is older and stays.

**Added (owner, 2026-10-13):** names in the lists are copyable too (Pods, Deployments and every
other view): a list's cells get the drawer's copy icon on hover, at least the Name column's, with
the same checkmark after a copy. **Built 2026-10-13** ([`tablecopy.go`](../../internal/ui/fyneui/tablecopy.go),
the icon generalised in [`copy.go`](../../internal/ui/fyneui/copy.go)): one icon per list, the
drawer's `copyButton`, in a layer above the table and moved to the cell under the pointer (a list
of 10,000 rows gets no widget per cell); the table hands it the cell when its row band moves
(`setHoverCell`), and a pointer move shows it only while the pointer is inside that cell, not in
the gaps or over the header. A scroll or new column widths hide it until the pointer moves; a
refresh keeps it on its cell, which copies the text it shows when clicked; a click checks first
that the icon still lies on its cell. The copy goes through `shell.copyText` as a plain copy, with
the drawer's checkmark for 1.5 s and its toast ("Copied to the clipboard"); no command, no audit
line.

**Changed (owner, 2026-10-13, after seeing it):** not every cell: "a name, an IP, a node name, a
message in the Events is great. But no Age, no CPU, no Memory, no Restarts." It works by a
blacklist of the columns where copying makes no sense, so that a custom resource's columns need no
marking: every column not on it keeps the icon, custom resource columns included. And the icon
belongs right after the text, not at the end of a wide column. **Built 2026-10-13**: the blacklist
is one place, `noCopyTypes` and `noCopyColumns` with `copyColumn` in
[`tablecopy.go`](../../internal/ui/fyneui/tablecopy.go) (the UI's alone: no command or tool
copies). The owner set the principle and the four examples; the lists below are Cogpit's reading
of it for every built-in kind (the lead's proposal, not the owner's decision): identifiers and free
text keep the icon; numbers, times, durations, counts, ratios, booleans and short state words do not.

| No icon | Columns |
|---|---|
| By type: `number` | Restarts, Count, Replicas, a workload's Pods (`3/3`), Ready, Desired, Current, Completions, Active, Size, Capacity, Taints, Value, Webhooks, Revision, Local Port, Pod Port and the other counts; a custom resource's `integer` and `number` printer columns |
| By type: `age`, `until` | Age, Last Seen, Last Schedule, Updated (Helm releases), Next Execution; a custom resource's `date` printer columns |
| By type: `metric`, `warning` | CPU, Memory; the warning triangle |
| By id, short state words | `status` (Pods, Services, PVCs, PVs, Namespaces, HPAs, Port Forwards, Helm releases, VPAs, Applications), `conditions` (Nodes, Deployments, Jobs), `state` (Resource Claims), `qos` (Pods), `mode` (VPAs), `phase` |
| By id, booleans | `suspend` (Cron Jobs), `default` (Ingress Classes), `globalDefault` (Priority Classes), `privileged` (Pod Security Policies) |
| By id, counts, ratios and durations in text | `minAvailable`, `maxUnavailable` (Pod Disruption Budgets), `metrics` (HPAs: current against target), and `ready`, `desired`, `current`, `upToDate`, `available`, `completions`, `duration` |
| By id, policy and mode words | `failurePolicy`, `reinvocationPolicy` (admission policies), `reclaimPolicy`, `volumeBindingMode` (Storage Classes), `validationActions` (validating admission policy bindings), `policyTypes` (Network Policies), `scope` (Definitions, Ingress Classes), `protocol` (Port Forwards) |

The ids apply in every view, so a custom resource's printer column whose name reads as one of them
in lowerCamelCase ("Ready", "Status", "Phase") has none either; `phase` and the text ids in the
counts row are there for those, since the built-in kinds hold such columns as numbers. A custom
resource's printer columns map to the types by their declared type
([`kinds/customresources.go`](../../internal/model/kinds/customresources.go), `compilePrinterColumn`):
`integer` and `number` to `number`, `date` to `age`, `string` and `boolean` to text, so a
`boolean` column keeps the icon unless its name is on the list (the table cannot tell it from a
string column). Every other column keeps the icon: Name, Namespace, Node, Controlled By, the IPs
(Cluster IP, External IP, Endpoints, Load Balancers), Ports, Labels, Selector, Node Selector, Keys,
Type and Address Type, Kind, API Group, Group, Resource, Version, Roles, Rules, Message, Involved
Object, Source, Schedule, Time Zone, Storage Class, Claim, Provisioner, Handler, Holder, Controller,
Driver, Pool, Devices and Device Classes of a claim, Reserved For, Bindings, Policy, Volumes, a
claim's Pods (their names), Chart, App Version, Repository, Description, Target, Managed By, App
Name. Placeholder cells (empty, `-`, `–`, `—`, `<none>`) still have none.

Cogpit's defaults besides, the lead's proposals and not the owner's decision:

- The icon sits a padding (4 points) after the cell's whole text, measured with the table's text
  style, centred on the row. When the text leaves no room for it, it sits at the cell's right end
  and the text is shortened with "…" before it while it is there (running under the icon, a
  cut-off name looked whole); the cell's tooltip then shows the whole text. Always inside the cell,
  never over the next column. A watch event that changes the text moves the icon to the new end.
- A click on it copies the cell's full text, not the shortened one, and neither selects the row,
  toggles its checkbox nor opens the drawer: the icon is a widget above the table, so the click
  never reaches the cell. A right-click on it opens the row's actions menu, as anywhere else on
  the row.
- A Secret's list holds no values: its columns are Name, Namespace, Labels, Keys (the keys' names),
  Type and Age ([`kinds/secrets.go`](../../internal/model/kinds/secrets.go),
  `TestSecretDetailShowsNoValue`), so no cell of it can put a value on the clipboard; Age has no
  icon (`TestSecretsListCopiesNoValue`).

Tests: `TestListCopyIconByColumn` (Pods with CPU and Memory values, Services, Events, Nodes and a
custom resource with an integer, a string, a "Ready" string and a date printer column),
`TestCopyColumns`, `TestListCopyIconAfterTheText`, `TestListCopyIconOnHover`,
`TestListCopyIconPlaceholders`, `TestListCopyIconCopiesTheFullText`,
`TestListCopyIconFollowsTheCells`, `TestSecretsListCopiesNoValue`, `TestListCopyIconHasNoCommand`;
renders `TestRenderListCopyIcon` (the Nodes list: PNG
[hover](../../internal/ui/fyneui/testdata/list_copy_hover_dark.png),
[after the copy](../../internal/ui/fyneui/testdata/list_copy_done_light.png)). Not tried in a real
window: the order in which the GLFW driver sends `MouseOut` to the table and `MouseIn` to the icon
was read in Fyne 2.8.1's source and replayed with Fyne's test driver.

## D-26: Binding a credential plugin's approval to its program

**Decided (owner, 2026-10-12, from R-5 of the fourth run's review):** the approval of a kubeconfig's
credential plugin (exec) also binds the content (SHA-256) of the program it runs when that program
lies inside the kubeconfig's own folder or the sync folder it came from, where a writer of the
synced folder (H-1's threat) can replace it; replacing it asks again. A program found elsewhere
(on `PATH`, `/usr/local/bin`: kubelogin, aws, gke-gcloud-auth-plugin) stays bound by its path, so
an update of the tool does not ask again; a writer of a `PATH` directory controls the user's
account anyway. Built with D-17 by the fifth run's agent D. The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Hash programs in the folder only (decided)** | Closes H-1's own threat; no prompt after a tool update | A program on `PATH` stays bound by path |
| (b) Hash every program | Binds every program | Asks again after every update of a plugin (brew, gcloud components) |
| (c) Document only | Nothing to build | The swap through the synced folder stays open |

## D-27: An agent's Helm values in the confirmation

**Decided (owner, 2026-10-12, from the remainder of R-2):** when an agent upgrades a release with
values it passes inline (`helm_release_upgrade` with `values`), the confirmation dialog shows those
values, monospace and selectable, in a box that scrolls, before the user answers. The upgrade runs
with exactly that text (it is part of the request's parameters). The text is carried in
`ConfirmRequest.Review`, which is never serialised: no audit line, log or MCP result holds it. An
upgrade from a Helm Upgrade tab shows no values in the dialog: the tab is what the user reads, and
the upgrade is bound to its text revision (R-2). Built 2026-10-12 by the lead
(`TestConfirmationShowsTheReviewText`, `TestHelmUpgradeAgents`). The options weighed:

| Option | For | Against |
|---|---|---|
| **(a) Show the values in the dialog (decided)** | One call for agents; the user reads exactly what is applied | A long text needs a scrolling box |
| (b) Refuse inline values for agents | Only the tab path, already bound | Agents need two calls; the user must open the tab to read |
| (c) Leave it, document it | Nothing to build | The user accepts values they did not see |

## D-28: The default kubeconfig as sources, as Lens has it

**Decided (owner, 2026-10-12, from the fifth run's agent C2):** every context comes from a source
file read on its own; nothing is merged any more. `~/.kube/config` is an ordinary file of the
default sync folder `~/.kube`, so removing that folder hides its contexts too, as in Lens. The
files named in `$KUBECONFIG` appear at start as additional default sources, each read alone and
removable (for the session), so a developer's or `make run`'s `KUBECONFIG` keeps working.
Consequence the owner accepted: a credential plugin in `config` asks for the user's approval once,
like a plugin in any other file (H-1). **Built 2026-10-12** (scope C3,
[architecture §6](architecture.md#kubeconfig-sources-built-2026-10-12)): `engine.WithDefaultFolder`,
`engine.WithSessionFiles`, `cluster.NamedKubeconfigFiles`; a connector with sources merges nothing.
Details the decision left open, decided in the build: a bare context name (`--context`, an agent's
`cluster.connect`) resolves to the one file that defines it and is refused, naming the ids, when
several do; a `$KUBECONFIG` file lying directly in `~/.kube` is the folder's, so removing the folder
hides it too; a session file removed in the window returns at the next start with its approvals kept;
an existing `kubeconfigs.json` is read as it is (no migration, nothing deleted or written), so a user
who had removed `~/.kube` loses `config`'s contexts now. The options weighed: `config` as kubectl
reads it, merged with `$KUBECONFIG` and not removable (as built by C2); as Lens (decided); and
for `$KUBECONFIG` under it: its files as sources (decided) or ignored, as Lens does.

**Asking once per file and plugin (owner, 2026-10-12, built the same day, scope C4).** A `config`
with ten EKS contexts and the same `aws eks get-token` plugin must not ask ten times. One dialog
per file and identical plugin lists every context of that file that uses it and is not allowed yet,
each with its server and how that server is reached and trusted (the warning lines of review R-11
per context where they apply), and one click allows all of them. "Identical" is
`kubeconfig.ExecPlugin.Identity`: the same command, arguments, environment (names and values), API
version, interactive mode, `provideClusterInfo`, and, after D-26, the same binding of the program
(its resolved path and, inside the folder, its content digest); for token files (D-17) the same token
file. Context, user, server and trust are not part of it. The approvals stay **one per context, each
with its own fingerprint** (server and trust bound, as strict as H-1 had it); only the asking is
bundled. A context added to the file later asks on its own, or with the other new ones. The approve
command stays GUI-only and takes the list of (context, fingerprint) the dialog showed; it checks
each against the file as it is now and refuses the whole list, naming the contexts, if any changed or
left meanwhile (`kubeconfig.pendingApprovals`, `kubeconfig.approveExec`; `TestPendingApprovalsGroupContextsByFileAndPlugin`,
`TestApprovingAGroupApprovesExactlyItsContexts`, `TestApprovalIsRefusedAsAWholeWhenOneContextChanged`).
The options weighed: one dialog per context (as built before; ten clicks and ten dialogs, which
invites approving without reading); one approval for the file's plugin whatever the context (one
click, but an approval that no longer binds the server: a changed server in `config` would not ask);
**one dialog, per-context approvals (decided)**.

## D-29: Helm chart repositories from Helm's repositories.yaml

**Decided (owner, 2026-10-13):** Cogpit reads Helm's own `repositories.yaml` regularly and takes
over its changes by itself: a repository added, changed or removed there (by `helm repo add`, by
hand, by another tool) is in Cogpit within 5 seconds. Repositories with credentials are included
when their credentials are available (Helm's file keeps a username and password, a client
certificate and key, a CA file). This replaces the lead's default of the night run (2026-10-13:
a list of Cogpit's own, public https repositories only, Helm's file never read). **Built
2026-10-13** (below).

What the decision leaves to the build, as the lead reads it (proposals, not decided):

- **Which file.** The one the helm CLI uses: `$HELM_REPOSITORY_CONFIG`, else
  `repositories.yaml` in `$HELM_CONFIG_HOME`, else in Helm's default configuration folder
  (`~/Library/Preferences/helm` on macOS, `$XDG_CONFIG_HOME/helm` or `~/.config/helm` on Linux,
  `%APPDATA%\helm` on Windows; Helm's `helmpath` rules, not verified on this machine).
- **How often.** A local file, not the API server: its size and modification time checked every
  2 seconds, as the kubeconfig sync folders are, and read again when they change; a file that does
  not parse keeps the last good list and says why. The 5 seconds are the owner's bound.
- **Reading, never writing (owner, 2026-10-13):** repositories are added with the helm CLI
  (`helm repo add`), not in Cogpit. Cogpit writes nothing into Helm's file; it neither adds nor
  removes a repository (removing would write the file too: the lead's reading of the decision).
  The night run's own list and its `helm.repo.add` and `helm.repo.remove` go when D-29 is built;
  the Repositories dialog then lists Helm's repositories read-only (charts, index age, the last
  error) and can refresh their indexes, which writes nothing into Helm's folders.
- **Credentials.** Read from Helm's file into memory only: never written to Cogpit's files, never
  in logs, errors, toasts, the audit log or an MCP result (agents see a repository's name and URL,
  and whether it has credentials); sent only to the repository's own host, unless Helm's
  `pass_credentials_all` says otherwise for that repository
  ([H-29](../security/credentials.md#h-29-authenticated-helm-repositories)).
- **Indexes.** Downloaded into Cogpit's own cache as today (no writes into Helm's cache), or read
  from Helm's cache when it is current: to decide at the build.

**Built (2026-10-13)** ([`helm/repofile.go`](../../internal/helm/repofile.go),
[`helm/repo.go`](../../internal/helm/repo.go), [`engine/helmrepos.go`](../../internal/engine/helmrepos.go),
[`commands/helmrepos.go`](../../internal/engine/commands/helmrepos.go),
[`fyneui/helmcharts.go`](../../internal/ui/fyneui/helmcharts.go); [architecture §8](architecture.md#8-helm-internalhelm)).
Verified in this repository by tests; on the owner's Mac only the path was resolved, never read:

- **Which file (as proposed).** The helm CLI's own resolution through the Helm SDK
  (`cli.New().RepositoryConfig`: `$HELM_REPOSITORY_CONFIG`, else `$HELM_CONFIG_HOME`, else
  `helmpath`, which also honours `$XDG_CONFIG_HOME`), in one place, `helm.RepositoryConfigPath`,
  called from `cmd/cogpit` only and handed to `engine.NewHelmRepos`, so no test can resolve the
  developer's file (`TestRepositoryConfigPath`, on a temporary `HOME`). On this Mac, without any
  `HELM_*` or `XDG_*` variable set, it resolves to `~/Library/Preferences/helm/repositories.yaml`
  (the path printed on 2026-10-13; the file itself was not opened).
- **How often (as proposed).** Size, modification time and mode looked at every 2 seconds, as the
  kubeconfig sync folders are; a change is read (at most 1 MiB, parsed with the SDK's `repo.File`)
  once the file stayed unchanged for 250 ms and taken only if it is still unchanged after reading,
  because `helm repo add` rewrites the file in place (Cogpit's default). Measured in
  `TestHelmRepoFileChangesWithinFiveSeconds` at the default poll: added and its index downloaded
  after 2.2 s, a changed URL after 2.0 s, removed after 2.0 s. A file that does not parse keeps the
  last good list, and `helm.repo.list` and the dialog say why, naming the line, never the content;
  a missing file is an empty list with the hint "add repositories with helm repo add".
- **Indexes: Cogpit's own cache (the lead's recommendation, decided at the build).**
  `<user cache dir>/Cogpit/helm/indexes/<key>-index.yaml`, owner-only, the key a hash of the
  repository's name and URL: nothing is written into Helm's folders, freshness stays under
  Cogpit's rules (an explicit update, or while the Charts view or a Helm Install tab is open), and
  a URL changed in Helm's file never shows an index fetched for the old one, which Helm's cache,
  named by the repository alone, would until the next `helm repo update`. The cost: a repository
  is downloaded once by helm and once by Cogpit. A repository that appears in the file, or whose
  URL, credentials or TLS settings change, counts as never downloaded and is fetched at once while
  the view or a tab is open (the file's change wakes the poller); its running download is stopped
  when it leaves the file or changes. At start an index an earlier run downloaded for the same
  name and URL is shown until its time-to-live passes.
- **Credentials.** Memory only (`helm.Repository`, whose credentials are unexported and print as
  its name and redacted URL through `%v`, `%#v`, slog and JSON); agents and the dialog see name,
  URL without user, password or query values, `hasCredentials`, `clientCertificate`, `caFile`,
  `insecureSkipTLSVerify`, `passCredentialsAll`, `plainHTTP`. Basic auth goes only to the
  repository's own scheme, host and port, decided per request on every redirect hop and for chart
  URLs the index points elsewhere (Go's own rule would keep it for a subdomain), unless the entry
  says `pass_credentials_all` (`TestCredentialsStayOnTheRepositoryHost`,
  `TestCredentialsNotSentToAChartElsewhere`). A CA file replaces the trust roots for that
  repository, as helm uses it; a client certificate is used when both `certFile` and `keyFile` are
  set, as helm does; the files are read at every download, at most 1 MiB, so a rotated file
  applies, and an error names the path only (`TestClientCertificateRepository`). Errors are
  scrubbed of the password and the user name as a guard (`TestScrubbedErrors`), and the proof
  that nothing leaks runs across the list, the install tab, events, notifications, the audit log
  and the log (`TestHelmRepoCredentialsNeverShown`; on kind `TestKindHelmInstall`).
- **Cogpit's defaults, not the owner's decisions:**
  - plain `http://` repositories are listed and used as helm uses them and marked "http" in the
    dialog (one with credentials only once the user allowed it: the owner's decision of
    2026-10-14, below); a redirect never goes from https to http, and an https
    repository's index never sends a download to http ([H-40](../security/credentials.md#h-40-credentials-over-plain-http),
    [H-41](../security/trust-boundaries.md#h-41-repositories-whose-content-nothing-authenticates));
  - `insecure_skip_tls_verify` is honoured for that repository only and marked "insecure";
  - a user and password in the URL are that repository's basic auth (Go's HTTP client, and so the
    helm CLI, sends them), and are shown nowhere; the query's values are shown as `•••`;
  - a repository whose name Cogpit's chart naming (`<repository>.<chart>`) cannot hold, a dot for
    one, and an OCI URL are listed with a clear reason and not used; a second entry of a name
    likewise (helm uses the first); a relative CA, certificate or key path likewise (Cogpit does
    not run in the folder helm ran in); at most 256 repositories are used;
  - the night run's `<user config dir>/Cogpit/helm/repositories.json` and its indexes in
    `<user cache dir>/Cogpit/helm/` are left on disk, unread (no implicit deletes);
  - the Repositories dialog follows the engine's new `EventHelmRepos` while it is open.
- **Commands.** `helm.repo.add` and `helm.repo.remove` are gone (registry, MCP, README, the
  dialog, tests: `TestNoCommandWritesHelmsFile`); `helm.repo.list` returns the file's path, its
  status (`ok`, `missing`, `error`), the reason, a hint and the repositories in the file's order;
  `helm.repo.update` reads the file first, then downloads.

**Decided (owner, 2026-10-14): a repository's credentials over plain http, asked once per
repository** ([H-40](../security/credentials.md#h-40-credentials-over-plain-http)). Cogpit asks
once per repository, keeps the approval like a kubeconfig credential plugin's, bound to the
repository's name, URL and user, and then uses the repository as helm does, the background refresh
included; without the approval no credentials go out over http: the index is not downloaded and the
Charts view says why. This replaces the build's default of 2026-10-13 (used as helm uses it,
credentials included, marked "http"). The options weighed:

| Option | For | Against |
|---|---|---|
| Refuse credentials over http | Nothing in clear, ever | A repository the user runs that way (an internal ChartMuseum) is unusable in Cogpit while helm uses it |
| Warn once, then send | One toast, no question | The first download has sent them already when the user reads it |
| **Ask once per repository (decided)** | Nothing leaves before the user's yes; one question per repository; a changed URL or user asks again | One more step for a repository the user knowingly runs over http |

**Built (2026-10-14)** ([`engine/helmhttp.go`](../../internal/engine/helmhttp.go),
[`commands/helmhttp.go`](../../internal/engine/commands/helmhttp.go),
[`fyneui/helmhttp.go`](../../internal/ui/fyneui/helmhttp.go), [`helm/repo.go`](../../internal/helm/repo.go)):
the helm package refuses a plain http request of a repository with a user and password before any
connection (`helm.ErrCredentialsOverHTTP`) and never adds them to a plain http request, unless the
engine hands it a copy the user allowed; the engine checks the allowance before every index, chart
and README download, leaves a waiting repository out of the background refresh and downloads it at
once when allowed. `helm.repo.list` says `httpCredentials` (`required`, `changed`, `allowed`) and,
while it waits, the reason in `error` and an `httpToken`; `helm.repo.allowHTTPCredentials` (GUI-only:
agents and the CLI are refused whatever they pass, D-2) records the allowance for the token the
window showed, all or none. The window shows a banner row per waiting repository below the Helm
banner while the Charts view or a Helm Install tab is open; Allow… says what crosses the network,
who can read it, what the allowance binds and that no agent can give it. Tests:
`TestPlainHTTPRepository`, `TestHTTPBinding`, `TestHelmCredentialsOverHTTPWaitForTheUser`,
`TestHelmHTTPWithoutCredentialsNeedsNoAllowance`, `TestHelmHTTPUnreadableApprovals`,
`TestAllowHTTPCredentialsIsTheUsers`, `TestHelmHTTPBannerAsksTheUser`. Cogpit's defaults of the
build, not the owner's decisions:

- **Where it is kept:** `<user config dir>/Cogpit/helm-http-approvals.json` (0600, written
  atomically), per repository its name, its URL redacted, a SHA-256 of its name, URL and user name,
  and when; no password, so a rotated password keeps the allowance (the owner's binding); without
  a configuration directory the allowances last for the session. An unreadable file allows nothing
  and is never overwritten.
- **What is asked about:** basic auth only. A client certificate needs TLS and is never presented
  over http, so a plain http repository with only a client certificate needs no allowance.
- **The token** the window hands back is a hash of the binding with a salt of the process: it
  changes with the URL or user name, reveals neither, and means nothing to another process.
- **Where it asks:** a banner in the window while the Charts view or a Helm Install tab is open (the
  times Cogpit would download), not a dialog that opens by itself; the Repositories dialog shows the
  reason in the repository's row.

## D-30: Certificates in Secrets, decoded

**Decided (owner, 2026-10-13):** "For Secrets of type certificate I want to see all the
certificate's information in the drawer, clearly decoded into separate fields; this information is
available to an agent too." Lens Desktop has no such section: this is beyond Lens, and nothing of
it is taken or inferred from Lens. It is a deliberate exception to the rule that a Secret's values
never reach an agent ([agents and MCP](../security/agents-and-mcp.md#rules)): a certificate is
public material, which every TLS client is sent anyway; the PEM text, the values and every private
key stay withheld. **Built 2026-10-13** ([`engine/secretcerts.go`](../../internal/engine/secretcerts.go),
[`model/certs.go`](../../internal/model/certs.go); [architecture §5](architecture.md#5-cache)).

**The owner's answers to the build's questions (2026-10-13):**

1. **Agents read the certificates, the built-in agent's LLM endpoint included** ("Absolutely
   not!" to withholding them): an agent may read every certificate, above all in its decoded form,
   and never a key. What was open as H-45 (a certificate's names, internal host names and e-mail
   addresses among them, reaching agents and the LLM endpoint) is decided: it is the rule, not a
   gap, and H-45 left [agents and MCP](../security/agents-and-mcp.md#rules) (its number is not
   reused).
2. **Check whether `tls.crt` matches `tls.key`** ("Yes, that would be fantastic!"): built the same
   day, below. For it Cogpit now parses a private key in memory, the one place it does
   ([credentials](../security/credentials.md#secret-objects)).
3. **Decode DER (binary) certificates too** ("Yes!"): built the same day, below.

What the build chose, Cogpit's defaults (the lead's, not the owner's decision):

- **Which keys.** Every key whose value holds a PEM `CERTIFICATE` block or DER certificates, in a
  Secret of any type (`kubernetes.io/tls`, Opaque, a service account token's `ca.crt`,
  cert-manager's), `tls.crt` and `ca.crt` first, then the other keys sorted. A `kubernetes.io/tls`
  Secret's `tls.crt` that holds none says "no certificate in this value". Only certificates are
  decoded (Go's `crypto/x509`): in PEM, the parser hands `pem.Decode` nothing but the text from a
  `CERTIFICATE` block's BEGIN line to its END line, and a block with another BEGIN line before its
  END is invalid, so a `PRIVATE KEY` block beside a certificate is never decoded; a block that does
  not decode or parse says "not a valid certificate" without quoting it. A key named like a private
  key (`tls.key`, `*.key`, `*key.pem` such as `server-key.pem` and Let's Encrypt's `privkey.pem`,
  `*key.der`) is never searched for certificates.
- **DER** (the owner's answer 3). A value with no PEM certificate is read as DER, one ASN.1
  element after the other, when its first element is shaped like a certificate (a SEQUENCE whose
  first element is a SEQUENCE; a private key in PKCS #1, PKCS #8 or SEC 1 starts with an INTEGER,
  so a DER key is never handed to the certificate parser) and parses as one; a later element that
  is none is "not a valid certificate". Any other binary value holds no certificate and shows
  nothing. Agents read `der: true` on such a key.
- **Key match** (the owner's answer 2). A key with certificates pairs with a private key where
  the pairing is unambiguous: `<base>.crt`, `.pem`, `.cer` or `.der` with whichever one of
  `<base>.key` and `<base>-key.pem` the Secret has (`tls.crt` with `tls.key`, `ca.crt` with
  `ca.key`, `server.pem` with `server-key.pem`; none when it has both); otherwise, when the Secret
  has exactly one key with certificates and exactly one key named like a private key, those two
  (`cert.pem` with `privkey.pem`); Let's Encrypt's four files pair with nothing. The private key
  (at most 64 KiB; PEM or DER, PKCS #1, PKCS #8 or SEC 1; RSA, ECDSA, Ed25519, X25519) is parsed in
  memory, its public half compared with the value's first certificate's public key, and dropped:
  the parsed key lives in one function call ([`model/keymatch.go`](../../internal/model/keymatch.go))
  and is never returned, stored or cached; the decoded value and the PEM block are zeroed, an
  Ed25519 key too (an RSA or ECDSA key's numbers wait for the garbage collector: Go cannot wipe
  them). A group's "Checks" section shows "Key Matches": "Yes (tls.key)" (OK), "No (tls.key holds
  another key)" (error), "Key not readable (tls.key: encrypted)" (unknown; also "unsupported key
  type" for an OpenSSH key or an algorithm Go does not know, "not readable", "too large"). Agents
  read `pairedKey` and `keyMatches`: true, false or the reason. No parser error leaves the check,
  so nothing quotes the key.
- **Chain** (Cogpit's default, which the lead left optional; built because it is cheap). In the same Checks section,
  "Chain In Order": whether each certificate of the value is signed by the next
  (`CheckSignatureFrom`: the next must be a CA whose key verifies the signature; no system roots,
  no revocation), "No, certificate 1 is not signed by certificate 2" an error. Checked for a value
  that pairs with a private key or whose first certificate is no CA (a leaf and its chain), not
  for a CA bundle of several roots; not for one certificate, a value with an invalid entry, or a
  SHA-1 signature Go no longer verifies. Agents read `chainInOrder` and `chainBreak`.
- **Bounds.** A value over 1 MiB is not decoded (a key named like a certificate, `*.crt`, `*.pem`,
  `*.cer`, `*.der`, says so); at most 16 certificates per key and 64 per Secret are decoded, the
  rest counted ("4 more certificates are not decoded"); a private key over 64 KiB is not read
  ("too large"); each text taken from a certificate is cut at 1 KiB, each list of names at 100
  entries, and control characters and bidirectional overrides in it are shown as `\x0a`, `\u202e`
  (a name with a line break would read as two names).
- **Fields,** each its own drawer item, empty ones left out: Subject (the common name) and Subject
  DN (the whole name, RFC 2253), Issuer and Issuer DN, Serial Number (hex, colon-separated), Not
  Before, Not After, DNS Names, IP Addresses, URIs and E-mail Addresses (one per line, a list of
  more than 10 folded behind its count), Public Key (`RSA 2048`, `ECDSA P-256`, `Ed25519`),
  Signature Algorithm, CA (`No`, `Yes, path length 0`, `Yes, no path length limit`), Key Usage,
  Extended Key Usage, Self-Signed (the issuer is the subject and the certificate's own key verifies
  its signature), SHA-256 Fingerprint, Subject Key ID, Authority Key ID.
- **Validity.** Times in the Preferences' time zone and the user's format, as Created is (RFC 3339
  until [D-31](#d-31-every-timestamp-in-the-local-time-zone-and-format)); Not After adds the
  time left ("in 90 days", "in 5 hours") or "expired 3 days ago" and is a status pill: OK with
  more than 30 days left, warning at 30 days or fewer, error when expired or not yet valid; Not
  Before is an error pill with "(not yet valid, in 5 days)" while it is.
- **Layout.** After the Data section, one group per key, "Certificates · tls.crt", modelled on the
  drawer's Containers: the Checks section first when there is a check, then one section per
  certificate in the value's order (a chain's leaf, then its
  intermediates and CA), titled by the subject's common name (else the subject), numbered in a
  value with several ("1 · shop.example.com"). Every field has the drawer's copy icon (D-25), which
  copies the value as shown (a name of a list in YAML syntax, as every list entry). The list views
  are not changed.
- **Agents.** `object_get` carries the same sections and a `certificates` list: per key `key`,
  `der`, `certificates`, `omitted`, `notDecoded`, `pairedKey`, `keyMatches`, `chainInOrder` and
  `chainBreak`; per certificate the same fields, the times as RFC 3339
  in UTC, `expiresInDays` (whole days left, rounded down, negative once expired), `expired`,
  `notYetValid` and `status`; an invalid block is `{"invalid": true, "error": "not a valid
  certificate"}`. Never the PEM text, a value of any key or a private key
  (`TestObjectGetDecodesCertificatesForAgents`). Nothing new in the audit log: reading is a read.
- **Not done.** Cogpit does not verify a chain against a trust store or for revocation, and an OK
  pill on Not After means only "within its validity period". An encrypted private key is not
  decrypted (Cogpit has no passphrase), and a private key inside a value not named like one (a
  combined PEM file) is not compared. A value wider than the drawer (the SHA-256 fingerprint at the
  default width) is shortened with "…"; its copy icon copies it whole.

Tests: `TestParseCertificatesChain`, `TestParseCertificatesOnlyCertificateBlocks`,
`TestParseCertificatesBounded`, `TestCertificateEvaluate`, `TestCertificateJSON`,
`TestCertificateTextsAreSafe`, `TestSecretCertificateSections`,
`TestSecretCertificateStatusesAndBounds`, `TestSecretCertificateEdges`,
`TestObjectGetDecodesCertificatesForAgents`, `TestSecretDrawerShowsCertificates`; the owner's
answers: `TestMatchPrivateKeyPairs`, `TestMatchPrivateKeyReasons`, `TestKeyMatchJSON`,
`TestChainInOrder`, `TestDecodeDERCertificates`, `TestSecretKeyMatchesCertificate` (matching RSA,
ECDSA and Ed25519 pairs, a mismatched pair, an encrypted key, garbage; the key's PEM, base64 and
DER appear nowhere in the drawer model), `TestSecretCertificatePairingAndChain`,
`TestSecretDERCertificates` (a DER `ca.der` and `tls.crt`, DER garbage, a DER private key under a
certificate's name: no fields, no leak); `TestObjectGetDecodesCertificatesForAgents` also checks
that no form of either private key reaches the result, the audit log, an event or the log; on kind
`TestKindSecretCertificates` (a `kubernetes.io/tls` Secret created through the API, read by an
agent's `object_get`, its key matching and its chain in order); renders
`TestRenderSecretCertificatesDark` and `TestRenderSecretCertificatesLight` (PNG
[dark](../../internal/ui/fyneui/testdata/secret_certs_dark.png),
[light](../../internal/ui/fyneui/testdata/secret_certs_light.png)).

## D-31: Every timestamp in the local time zone and format

**Decided (owner, 2026-10-13):** "Timestamps always in the local time zone, summer and winter time
respected, in the format that fits the location. Later it may be configurable in the Settings, but
the default is the local time zone and the local format. For me that would be 10.09.2026 17:41:01.
This applies to every timestamp shown." (Seen in the drawer: a certificate's Not Before read
`2026-09-10T17:41:01+02:00`.) **Built 2026-10-13** ([`internal/locale`](../../internal/locale/),
[`model/timefmt.go`](../../internal/model/timefmt.go); [architecture §3](architecture.md#3-dependency-layering-enforced),
[§4.4](architecture.md#44-tables-details-editors-and-streams)). **Not built:** the format as a
setting (the owner's "later"); the code takes a layout everywhere, so that a preference can replace
the computer's.

What the build chose, Cogpit's defaults (the lead's and the build's, not the owner's decision):

- **The format is the operating system's region, not `$LANG`.** On macOS Cogpit asks
  CoreFoundation for the current locale's short date style and medium time style
  (`CFLocaleCopyCurrent`, `CFDateFormatterCreate`): the region in System Settings, a user's own
  formats and the 24-hour switch included, and a Finder-started app has no `$LANG` anyway. A
  two-digit year is written with four digits; date and time are joined by one space. On the
  owner's Mac (region `en_DE`, the shell's `LANG=en_US.UTF-8`) macOS answers `dd.MM.yy` and
  `HH:mm:ss`, written `10.09.2026 17:41:01` (verified 2026-10-13, `TestSystemFormatOnMacOS` logs
  it). Linux reads the POSIX locale (`LC_ALL`, else `LC_TIME`, else `LANG`) against a table of
  common locales' CLDR patterns (42 territories and 23 languages, taken from macOS's ICU on
  2026-10-13), by territory, else by language; C, POSIX, an unknown locale or none: ISO 8601,
  `2026-09-10 17:41:01`. **Windows reads the environment too, for now**: its region settings are
  not read, so a Windows session without `LANG` gets ISO 8601. `cogpit-cli` reads the environment
  on every platform, as command-line tools do.
- **Conversion** ([`locale.GoLayout`](../../internal/locale/icu.go)). Unicode's patterns (UTS #35)
  become Go layouts: day, month (number, short or full name), year, weekday, 12- and 24-hour
  hours, minutes, seconds, fractions, AM/PM, zones, quoted literals; narrow and non-breaking
  spaces become spaces. Approximations: Go writes 24-hour hours with two digits (Spain's `H:mm:ss`
  reads `07:41:01`, not `7:41:01`) and names and AM/PM in English. A pattern with a field Go cannot
  write (an era, a quarter, a week, the day period `B`), literal text Go would read as a field, or
  more than 128 bytes falls back to ISO 8601 for its part; so do a date without day, month and
  year and a time without hours and minutes.
- **One formatter.** The binaries look the layout up once at start and hand it to the engine
  (`engine.Options.TimeLayout`); the engine writes every time the user reads in it and in the
  Preferences' time zone (`Session.TimeFormat`), and the window writes with the same layout
  (`UIState.TimeFormat`) and zone (`shell.timeFormat`). Summer and winter time come from the zone
  database: Berlin is UTC+2 in September and UTC+1 in December
  (`TestTimeFormatZoneAndSummerTime`). Derived forms: the time without seconds and the day
  without the year (charts' axes), and with milliseconds (log lines, several of which share a
  second).
- **Where.** The drawer: Created and every absolute time a kind shows (a container's start and end,
  a job's start and completion, a cron job's schedules, leases, autoscalers, volumes, events' first
  and last seen, resource claims, custom resources' date columns, a Helm chart's Created and its
  versions' dates, a Helm release's Updated, a certificate's Not Before and Not After). A kind's
  drawer writes a time as a mark that the engine replaces when it builds the drawer
  (`model.MarkTime`, `TimeFormat.WriteSections`). An event's Last Seen in the Events list (a
  column type `time`, written when the cell is drawn, sorted by time). The logs' timestamps, their
  download, and the logs' time field, which reads the same format (and ISO 8601, RFC 3339 and the
  other forms) in the Preferences' zone, where it read UTC before. The notifications and the audit
  log (the time of day today, the date before it otherwise), the drain, Helm and drafts banners,
  the Rollback dialog, the Connection Updates log, the metrics providers' tooltip, the charts'
  axes and hover panels, the crash notice, the CSV export.
- **Kept as they are.** Ages and countdowns ("5m", "160d", "In about 1 hour 57 min"), and what
  Cogpit writes for machines: its files (journals, settings, the crash report's file name), an
  object's YAML, and the structured times of MCP results (certificates' `notBefore` and
  `notAfter`, notifications, audit entries, a logs tab's jump), which stay RFC 3339. A credential
  plugin file's fingerprint note keeps its RFC 3339 time, because the approval compares it.
- **Agents.** What an agent reads as drawn, `view_rows` and `resources_list` cells and
  `object_get`'s sections, is in the user's format; `ui_state` names it (`timeFormat`, Go layouts),
  so that an agent can tell `09.10.` from `10/9` (`TestAgentsReadTimesAsDrawn`).
- **Search.** The list's search reads cells' text, and an event's Last Seen has none now (as an
  age has none): searching for a time no longer finds an event.

Tests: `TestGoLayout`, `TestFromPatterns`, `TestEnvironment`, `TestTableConverts`,
`TestSystemFormatOnMacOS` (macOS with cgo only; no assertion on the developer's region),
`TestTimeFormatZoneAndSummerTime`, `TestDerivedLayouts`, `TestMarksWrittenInTheFormat`,
`TestColumnTime`, `TestCreatedFollowsTheTimezone`, `TestParseLogTimeInTheUsersFormat`,
`TestJumpLogsInTheUsersZone`, `TestAgentsReadTimesAsDrawn`,
`TestTheWindowWritesTimesInTheUsersFormat`, `TestLogTimestampsInTheUsersFormat`; renders
`TestRenderTimesInTheUsersFormat` (PNG [dark](../../internal/ui/fyneui/testdata/logs_times_de_dark.png),
[light](../../internal/ui/fyneui/testdata/logs_times_de_light.png): Germany's format, Berlin's time).
The test fixtures carry no layout, so the other images show the ISO 8601 fallback.

## D-32: Every byte amount in binary units with one decimal

**Decided (owner, 2026-10-13):** "Unacceptable. Bytes only up to 1023; from 1024 bytes KiB, from
1024 KiB MiB, and so on, each with one decimal. This applies to every presentation of bytes,
unless you see it differently: then name me the place where we should not do it and we talk about
it." (Seen in a Node's drawer: Capacity Memory `16360780Ki`, Ephemeral Storage `119842348Ki`,
Allocatable Memory `13856852378`, plain bytes.) **Built 2026-10-13**
([`model/bytes.go`](../../internal/model/bytes.go); [architecture › Current state](architecture.md#current-state-2026-10-14)):
that drawer now reads 15.6 GiB, 114.3 GiB and 12.9 GiB (`TestNodeDrawerBytesInBinaryUnits`).

What the build chose, Cogpit's defaults (the lead's and the build's, not the owner's decision):

- **The form.** A space between number and unit, as IEC 80000-13 writes it ("512 B", "15.6 GiB");
  the units B, KiB, MiB, GiB, TiB, PiB and EiB; whole bytes below 1024 without a decimal. A
  decimal point whatever the region: the UI is English, and unlike D-31's dates the separator does
  not follow the computer's region. The unit is chosen by the number as written, so that rounding
  never shows 1024.0 of a unit: 1 MiB less one byte reads "1.0 MiB", not "1024.0 KiB". An amount
  that need not be whole (a chart's value, a rate averaged over a window) is rounded to whole
  bytes below 1 KiB ("0 B" for 0.25).
- **One formatter** in L0, for the engine, the window and the CLI alike: `model.FormatBytes` (a
  count), `FormatByteAmount` (a float), `FormatByteRate` (bytes per second, ".../s"),
  `FormatByteQuantity` (a Kubernetes quantity of bytes as an object writes it: "16360780Ki",
  "13856852378", "10Gi", "1G", "500M") and `FormatResource` (one entry of a resource list: bytes
  for `memory`, `storage`, `ephemeral-storage` and `hugepages-*`, also as a resource quota names
  them, `requests.memory`, `limits.ephemeral-storage`,
  `<class>.storageclass.storage.k8s.io/requests.storage`; CPU, counts and extended resources as
  written). A quantity in milli-units ("500m", "2000m": almost always a mistake the user should see
  as written), a fraction of a byte ("1.5"), text that is no quantity, and a quantity beyond what
  Kubernetes holds (it clamps "10Ei" to 8 EiB) are shown as written (`TestFormatBytes`,
  `TestFormatByteAmount`, `TestFormatByteQuantity`, `TestByteResource`).
- **Where.** The lists: Persistent Volumes' Capacity, Persistent Volume Claims' Size, the Pods
  list's Memory (and so the node drawer's Pods table, which reuses its cells), the Vertical Pod
  Autoscalers' Memory, Horizontal Pod Autoscalers' Metrics for a memory or ephemeral-storage
  resource metric ("memory: 288.0 MiB/512.0 MiB"). The drawers: a node's Capacity and Allocatable
  (memory, ephemeral storage, huge pages; CPU, pods and devices as written), a container's
  Requests and Limits, a volume's Capacity, a claim's Requested, Limit, Capacity and Allocated, a
  stateful set's volume claim templates, a resource quota's Used and Hard, a limit range's Min,
  Max, Default Request and Default Limit (not its Max Limit/Request Ratio, a ratio), a runtime
  class's Pod Overhead, a vertical pod autoscaler's policies and recommendations, a device's
  capacity named `memory` (resource slices and a node's Devices), a Config Map's binary data
  sizes and its cut notes ("[cut: 4.0 KiB of 8.8 KiB shown; …]", the Helm record note). The
  charts: the drawers' Memory and Filesystem axes and hover panels, the Network and Filesystem
  rates ("48.0 KiB/s"), the cluster Overview's Memory chart and its headline ("41.9 GiB of
  118.0 GiB (35%)"); the axis writes what the hover panel writes, and a byte axis steps by at
  least one byte, since its labels are whole bytes (`TestRateFormatsAndLongRanges`,
  `TestDrawerChartsHoverPerChartAndVariant`; PNG
  [dark](../../internal/ui/fyneui/testdata/chart_hover_memory_dark.png),
  [light](../../internal/ui/fyneui/testdata/chart_hover_memory_light.png)).
  Sizes of data: a Secret's masked values and binary values in the drawer ("•••••••• (38 B)",
  "binary data, 1.2 KiB"; "38 bytes" before), a revealed value's cut note (which wrote 1.5 MiB as
  "1536.0 KiB"), a certificate value too large to decode ("larger than 1.0 MiB"), the kubectl
  download's progress ("22.0 MiB of 54.4 MiB"; decimal "23.1 MB of 57.0 MB" before), the Audit
  Log tab's note on a partial read ("the log's last 4.0 MiB") and on parameters too large to list.
  The messages that state a size: files and downloads against Cogpit's bounds (kubeconfig files,
  Helm's `repositories.yaml`, indexes, chart archives, the kubectl binary and its signature files,
  Prometheus responses, a Helm release payload, private files, the cluster icon "larger than
  1.0 MiB" where it read "1024 KiB"), the editor's "larger than the 4.0 MiB an editor takes", a
  journal that is too large.
- **A size beside its limit is not named** (Cogpit's default): one decimal can make a size just
  over its limit read like the limit ("the binary has 4.0 KiB, more than the 4.0 KiB Cogpit
  accepts" for 4,096 bytes against 4,095), so those messages name the limit only ("the binary is
  larger than the 4.0 KiB Cogpit accepts"), and a download that ended short names the difference
  ("received 10 B less than the 54.4 MiB announced").
- **Sorting.** The sort keys stay bytes (`Cell.Num`), so sorting by size still works; only the text
  changed (`TestPersistentVolumeClaimSizeSorts`).
- **Agents.** What an agent reads as drawn follows the formatter: `view_rows` and
  `resources_list` cells, `object_get`'s sections. Structured numbers stay raw: `metrics_get`'s
  `memoryBytes`, the Overview's chart values, a revealed value's `size`, an object's YAML.
- **Nothing to convert yet:** an `emptyDir`'s `sizeLimit` (the pod drawer lists volumes by type
  and name only), a Helm chart archive's size (shown nowhere; its download bound is converted),
  the logs and the status bar (they show no byte amount).
- **Renders** (offscreen, re-rendered 2026-10-13): the node drawer
  ([dark](../../internal/ui/fyneui/testdata/node_drawer_dark.png), [light](../../internal/ui/fyneui/testdata/node_drawer_light.png)) and with its
  Pods ([dark](../../internal/ui/fyneui/testdata/node_pods_drawer_dark.png), [light](../../internal/ui/fyneui/testdata/node_pods_drawer_light.png)),
  a node's Devices ([dark](../../internal/ui/fyneui/testdata/node_devices_drawer_dark.png), [light](../../internal/ui/fyneui/testdata/node_devices_drawer_light.png)),
  a pod's containers ([dark](../../internal/ui/fyneui/testdata/pod_drawer_ports_dark.png), [light](../../internal/ui/fyneui/testdata/pod_drawer_ports_light.png)),
  the claims list ([dark](../../internal/ui/fyneui/testdata/pvc_list_dark.png), [light](../../internal/ui/fyneui/testdata/pvc_list_light.png)),
  a Secret ([dark](../../internal/ui/fyneui/testdata/secrets_dark.png), [light](../../internal/ui/fyneui/testdata/secrets_light.png)),
  the demo's Pods list ([PNG](../../internal/ui/fyneui/testdata/demo_pods.png)), the cluster Overview
  ([dark](../../internal/ui/fyneui/testdata/cluster_overview_dark.png), [light](../../internal/ui/fyneui/testdata/cluster_overview_light.png)).

**Not converted: the places to talk about** (the owner's "name me the place"; each kept as it is
until the owner decides):

1. **The YAML view and the editors** ([`yamlview`](../../internal/ui/fyneui/yamlview/), the Edit,
   Create and Helm tabs): they show the object as it is, `memory: 128Mi` included.
2. **Decimal suffixes are converted**, the lead's point: `memory: 1G` in a spec reads "953.7 MiB"
   in the drawer and "1G" in the YAML, so the drawer never shows what the user wrote; "500M" reads
   "476.8 MiB", "262144k" "250.0 MiB". Built as the rule says.
3. **Huge pages' names keep their page size**: the node drawer's titles "Hugepages 1Gi" and
   "Hugepages 2Mi" ([`nodes.go`](../../internal/model/kinds/nodes.go) `resourceTitles`), and
   `hugepages-2Mi` wherever a resource is named (a quota's Resource column, a container's
   requests): the resource's name, not an amount. Their amounts are converted ("0 B").
4. **A device's capacity under another name than a byte resource**
   ([`resourceslices.go`](../../internal/model/kinds/resourceslices.go) `deviceCapacity`): a DRA
   driver names its capacities and Kubernetes declares no unit for them; "memory: 40Gi" reads
   "40.0 GiB", "size: 3840Gi" stays as the driver wrote it.
5. **The audit log's records** ([`audit.go`](../../internal/engine/audit.go): "<1234 bytes of
   manifests; not shown>", "<N bytes that are not manifests; not shown>", "<N bytes of Helm
   values; not logged>"): what the file records, with exact counts and in the form older lines
   have; the same note stands for a half-typed Secret in an editor tab an agent reads. The Audit
   Log tab's own note on parameters too large to list is converted.
6. **A credential plugin file's fingerprint note**
   ([`pluginfiles.go`](../../internal/cluster/pluginfiles.go) `hashRegular`: "larger than N bytes
   (N bytes, modified …)"): the approval binds the note in place of the content; another text
   would ask again for every such approval, and the exact size is part of the binding (as D-31
   keeps its RFC 3339 time).
7. **Limits on typed text**, lengths rather than amounts: a shell path and its arguments and the
   terminal's working directory ("one line of at most N bytes",
   [`preferences.go`](../../internal/engine/preferences.go),
   [`clustersettings.go`](../../internal/engine/clustersettings.go)), the proxy URL
   ([`proxy.go`](../../internal/engine/proxy.go)), a list's regular expression
   ([`filteroptions.go`](../../internal/engine/filteroptions.go), 1000 bytes), the find bar
   ([`findbar.go`](../../internal/ui/fyneui/findbar.go)), a logs search
   ([`commands/logs.go`](../../internal/engine/commands/logs.go)), the kubectl mirror
   ([`kubectl/download.go`](../../internal/kubectl/download.go)), a terminal kubeconfig's names
   ([`kubeconfig/terminal.go`](../../internal/kubeconfig/terminal.go)), a date pattern
   ([`locale/icu.go`](../../internal/locale/icu.go)). Bytes stand for characters there; "at most
   4.0 KiB" for a shell path would read as a file size.
8. **Command descriptions and parameter schemas** ([`engine/commands`](../../internal/engine/commands/):
   "at most 1 MiB", "the last 4 MiB", "at most 1000 bytes"), and the README's command table made
   from them: prose for agents, where a limit reads as a person writes it.
9. **The performance harness** ([`perf.go`](../../internal/ui/fyneui/perf.go) "YAML editor,
   5 MB", `cogpit --perf-mode`'s help): a developer's tool, not the product.
10. **Measurements recorded in the docs** ("15.8MiB" on kind in
    [architecture §9](architecture.md#9-metrics-internalmetrics)): records of what was seen, kept
    as written.

Other details of the build, for the owner to know: an amount that need not be whole is rounded to
whole bytes below 1 KiB, so a rate of 0.25 B/s reads "0 B/s" (it read "0.3B/s"); Lens's own form
of a size (with or without a space, "Mi" or "MiB") is not captured, so Cogpit's lists may differ
from Lens's there.

## D-33: The Flux actions

**Context.** D-9 puts Flux CD in the beta; the owner's team works with Flux daily
(`k8s-flux-base`). Lens's Flux section is captured as a sidebar structure only
([ui-parity › Cluster sidebar](../parity/ui-parity.md#cluster-sidebar)); its actions are neither
documented nor captured ([resource-parity › Paid Lens Desktop views](../parity/resource-parity.md#paid-lens-desktop-views)).
Flux's own CLI does three things to an object every day: `flux reconcile`, `flux suspend`,
`flux resume`. **Built 2026-10-13** ([`commands/flux.go`](../../internal/engine/commands/flux.go),
[`cluster/flux.go`](../../internal/cluster/flux.go); `TestFluxActionsPatch`,
`TestFluxActionsApply`, `TestFluxActionsAgentsNeedAcceptance`, `TestFluxActionsOnASelection`,
`TestKindFlux`); Reconcile with Source and Reset **built 2026-10-14**
([`commands/fluxsource.go`](../../internal/engine/commands/fluxsource.go),
[`kinds/flux_withsource.go`](../../internal/model/kinds/flux_withsource.go);
`TestFluxReconcileWithSource`, `TestFluxReset`, `TestFluxReconcileWithSourceAgents`,
`TestFluxReconcileWithSourceOnASelection`, `TestFluxActionsWithSourceInTheMenus`,
`TestFluxSourceOf`, `TestFluxPatch`, `TestFluxHelmReleaseActionsInTheWindow`, `TestKindFlux`; PNGs
[drawer dark](../../internal/ui/fyneui/testdata/flux_helmrelease_drawer_dark.png),
[drawer light](../../internal/ui/fyneui/testdata/flux_helmrelease_drawer_light.png)).

- **Decided (owner, 2026-10-14): which actions.** Options (a) Reconcile only, (b) Reconcile,
  Suspend and Resume, as the flux CLI's everyday commands, (c) (b) plus Reconcile with Source,
  Reset and Force as separate menu entries; the night build had chosen (b). The owner chose (b)
  plus **Reconcile with Source and Reset as the flux CLI does them**; **Force stays as built**, a
  parameter of `flux.reconcile` and `flux.reconcileMany` for Helm releases, for agents only (no
  menu entry).
  - **Reconcile with Source** (`flux.reconcileWithSource`, `flux.reconcileWithSourceMany`;
    Kustomizations and Helm releases), as `flux reconcile kustomization|helmrelease --with-source`
    (flux2's `cmd/flux`, read 2026-10-14): `reconcile.fluxcd.io/requestedAt` on the source first,
    then on the object, each patch bound to its object's UID like Reconcile's. The source: a
    Kustomization's `spec.sourceRef` (a GitRepository, OCIRepository or Bucket, in the
    Kustomization's namespace unless it names one); a Helm release's `spec.chartRef` (an
    OCIRepository, or a HelmChart), else the HelmChart helm-controller makes of `spec.chart`
    (`<namespace>-<name>` of the release, in the namespace of the chart's `sourceRef`). A HelmChart
    is reconciled with its own source before it (a HelmRepository, GitRepository or Bucket), as the
    flux CLI does, and an OCI HelmRepository is skipped, which the flux CLI skips too ("reconciliation
    not supported by the object"). Greyed with the reason on a suspended object, on one whose
    source is of a kind no Flux controller reconciles (an ExternalArtifact), when the cluster does
    not serve the source's kind, and when the access reviews say the user may not patch the source.
    A source shared by several selected objects is patched once.
  - **Reset** (`flux.reset`, `flux.resetMany`; Helm releases), as `flux reconcile helmrelease
    --reset`: `reconcile.fluxcd.io/resetAt` and `reconcile.fluxcd.io/requestedAt` with one token
    (helm-controller's `api/v2`: resetAt takes effect only with requestedAt's value), so that
    helm-controller forgets the release's failure counts and tries again after it ran out of
    retries. Not offered for a suspended release, as Reconcile is not.
- **Cogpit's default (the lead's, 2026-10-14; the owner left it open, so it stands until he asks):
  whether Reconcile with Source waits for the source.** Options (a) as the flux CLI: annotate the source, wait until its
  controller reports it ready (a job with a timeout), then annotate the object; (b) annotate the
  source, then the object at once; chosen (b), because Cogpit's Reconcile does not wait for a
  controller either, and kustomize-controller and helm-controller reconcile an object again by
  themselves when its source's artifact changes (Flux's documentation; inferred, not verified on a
  cluster that runs the controllers), so the object ends on the new revision without a wait that
  would outlive the menu click.
- **Cogpit's default (the lead's, 2026-10-14; the owner left it open, so it stands until he asks):
  what is checked before the first patch.** Options (a) as the flux CLI: each source checked when its turn comes, so a
  suspended HelmChart stops the action after its Helm repository was annotated; (b) every object
  of the chain read and checked first (present in the cluster, not suspended), nothing patched when
  one fails; chosen (b), because a refusal then leaves nothing half done; a patch that the API
  server refuses in the middle still stops the action, and the error names what was done.
- **Decided (owner, 2026-10-14; the night build's option (a)): what each writes.**
  Options (a) the flux CLI's patches: Reconcile the annotation `reconcile.fluxcd.io/requestedAt`
  set to the time (RFC 3339 with nanoseconds), Force also `reconcile.fluxcd.io/forceAt`, Suspend
  `spec.suspend: true`, Resume `spec.suspend: false` with a reconcile request; (b) server-side
  apply of the same fields; chosen (a), one JSON merge patch per object, because it is exactly
  what the controllers and the CLI agree on, touches no other field, and needs nothing but `patch`.
  Each patch carries `metadata.uid`, as Restart's and Scale's do, so that an object deleted and
  created again under its name is refused (422 "metadata.uid: field is immutable" on Kubernetes
  1.36.1, `TestKindFlux`). Reset adds `reconcile.fluxcd.io/resetAt` with requestedAt's value.
- **Decided (owner, 2026-10-14; the night build's option (a), Reconcile with Source and Reset
  too): their danger.** Options (a)
  mutating, (b) destructive (confirmed for every origin), (c) mutating, Force destructive;
  chosen (a) for all three, Force included, because none deletes or replaces anything the
  controller would not do on its next interval anyway; Suspend stops reconciliation, which Resume
  undoes. So the GUI's user is not asked, agents need the user's acceptance unless agent writes are
  allowed for the context (D-2), and every agent's call is in the audit log.
- **Decided with D-33 as a whole (owner, 2026-10-14; the night build's option (b)): where they
  apply.** Options
  (a) every Flux kind, (b) the kinds whose controller reconciles on request, Suspend and Resume
  where the kind has `spec.suspend`; chosen (b): Reconcile on every kind but External Artifacts
  (another controller makes them), and not on an Alert or Provider that reports no status
  (notification v1beta3's static objects, which no controller reconciles); Suspend and Resume on
  every kind but Image Policies and External Artifacts. Reconcile on a suspended object is greyed
  with the reason (the controller would ignore it, as the flux CLI refuses it); Suspend and Resume
  each show where the other does not, as Cron Jobs' do.
- **Decided (owner, 2026-10-14; the night build's option (b), Reconcile with Source and Reset
  too): selections.** Options
  (a) one object at a time, (b) on a selection too, acting on the selected objects the action applies
  to and skipping the others; chosen (b), as Delete of a selection works (`object.deleteMany`),
  because "reconcile everything that failed" from the Flux dashboard's table is the case the
  dashboard exists for. Ids `flux.reconcileMany`, `flux.suspendMany`, `flux.resumeMany`,
  `flux.reconcileWithSourceMany`, `flux.resetMany`.

Cogpit's defaults, not the owner's decision: the labels Reconcile, Suspend and Resume; their place
in the row menu and the drawer's toolbar before Edit and Delete, Reconcile first, then Reconcile
with Source and Reset; their toolbar icons (Restart's, Suspend's and Resume's of the built-in
kinds; Reconcile with Source a download arrow, Reset a replay arrow); the toast "kustomizations…
reconcile requested", which does not wait for the controller (the object's Status column follows
its conditions); Reset offered whatever the release's failure counts say, as the flux CLI's
`--reset` is.

## D-34: The Flux dashboard

**Context.** Lens's Flux section starts with a Dashboard (captured as a sidebar entry only); its
contents are not captured or documented. **Built 2026-10-13**
([`engine/fluxdashboard.go`](../../internal/engine/fluxdashboard.go),
[`fyneui/fluxdashboard.go`](../../internal/ui/fyneui/fluxdashboard.go),
[`model/kinds/flux.go`](../../internal/model/kinds/flux.go); `TestFluxDashboardPage`,
`TestFluxDashboardInTheWindow`, `TestFluxStatusTexts`; PNG
[dark](../../internal/ui/fyneui/testdata/flux_dashboard_dark.png)).

- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): what a Flux object's
  state is.** Options (a) the Ready condition alone (True, False, Unknown), (b) kstatus's reading, as
  Flux documents it: suspended first, then a failure (Ready False or Stalled True), then a
  reconciliation in progress (Reconciling True, Ready Unknown, or a generation the controller has
  not observed), then Ready True, else Unknown; chosen (b), five states Ready (green), Not Ready
  (red), Reconciling (amber), Suspended (grey, the stopped token) and Unknown (grey), because (a)
  shows a suspended or freshly changed object as ready. Notification v1beta3's Alerts and Providers
  report no status at all: they count as Ready unless suspended (Flux treats them as active once
  admitted).
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): what the page shows.**
  Options (a) counts per kind by state only, (b) the counts and a table of every Flux object, (c)
  the counts and a table of the objects that are not ready; chosen (c): a tile per Flux kind the
  cluster serves (a served kind without objects too), and the table Not Ready of every kind, the
  latest change of the Ready condition first, whose selection takes the Flux actions (D-33).
- **Decided (owner, 2026-10-14): a click on a state opens the kind's view filtered to that state,
  through a status filter every view has (option a).** The options were (a) the kind's view
  filtered to that state, through a status filter the views did not have yet, (b) the view with the
  search set to the state's word, (c) the kind's view unfiltered (the night's choice, built
  2026-10-13). The owner chose (a) and asked for the filter in every view, with the Argo CD
  Dashboard's tiles, which used the search box, switched to it too. **Built 2026-10-14**
  ([`model/states.go`](../../internal/model/states.go),
  [`engine/statusfilter.go`](../../internal/engine/statusfilter.go),
  [`commands/statusfilter.go`](../../internal/engine/commands/statusfilter.go),
  [`fyneui/statusfilter.go`](../../internal/ui/fyneui/statusfilter.go); `TestStatesOfColumns`,
  `TestStateMatcher`, `TestKindStates`, `TestStatusFilterSnapshot`, `TestStatusFilterBelongsToTheView`,
  `TestStatusFilterFluxPage`, `TestArgoTilesNameTheStatusFilter`, `TestWorkspaceKeepsTheStatusFilter`,
  `TestStatusFilterCommand`, `TestStatusFilterInTheHeader`, `TestStatusFilterOnAPage`,
  `TestFluxDashboardInTheWindow`, `TestArgoDashboardPanel`; PNGs the list
  [dark](../../internal/ui/fyneui/testdata/status_filter_menu_dark.png),
  [light](../../internal/ui/fyneui/testdata/status_filter_menu_light.png), a filtered Pods view
  [dark](../../internal/ui/fyneui/testdata/status_filter_pods_dark.png),
  [light](../../internal/ui/fyneui/testdata/status_filter_pods_light.png), the Flux dashboard's Not
  Ready of Kustomizations clicked [dark](../../internal/ui/fyneui/testdata/flux_state_filtered_dark.png),
  [light](../../internal/ui/fyneui/testdata/flux_state_filtered_light.png)). What it is: a tab's own
  state, `UIState.States` (`ui.setStatusFilter`; `ui.openView` with `states` opens a view filtered,
  which is what the dashboards' state links do), applied to the view's snapshot like the search, so
  the table, `view.rows`, select all and the CSV export show the same rows; `resources.list` takes
  `states` too; the snapshot offers the states its rows carry with their counts
  (`TableSnapshot.StateChoices`, `view.rows`' `stateChoices`); the window shows it beside the
  namespace filter, a list of checks with each state's colour and count, a × that clears it in one
  click; kept across restarts with the tab's other state ([D-41](#d-41-what-survives-a-restart)).
  Filtering a 10,000-row snapshot costs no request and 2.9–3.7 ms on the development Mac
  (`BenchmarkSnapshot10kStates`, measured 2026-10-14). Lens's views have no status filter: it is
  Cogpit's addition ([beyond Lens › Everyday use](../parity/beyond-lens.md#everyday-use)).
- **Cogpit's default (2026-10-14; the owner left it open, so it stands until he asks): what a row's
  states are.** Options (a) the rows'
  status tokens only, by name (OK, Pending, Error), (b) the Status column's text, the tokens for a
  kind without one, (c) the texts of the kind's state columns: Status, State, Phase, Sync and Health
  whole, Conditions word by word (a cell drawn in parts part by part), a kind naming its own where
  the default does not fit (an event's Type; a Horizontal Pod Autoscaler's Status, a list of
  condition types), the status tokens by name for a kind without one; chosen (c), because (a)
  merges Running with Ready and cannot tell an Argo CD Application's Sync from its Health, which the
  Dashboard's tiles need, and (b) leaves Deployments, Jobs and Nodes (Conditions) and Argo CD
  Applications (Sync, Health) with tokens only (`KindProfile.States`, `model.DefaultStateColumns`).
- **Cogpit's default (2026-10-14; the owner left it open, so it stands until he asks): a kind with
  several state columns.** Options
  (a) one list of texts, a row shown when it carries any selected text; (b) each state qualified by
  its column's title ("Sync: OutOfSync", "Health: Degraded"), a row shown when it carries a selected
  state of every column the selection names (any of them within a column); chosen (b), because
  both of an Argo CD Application's columns have Unknown, and "Out of sync and Degraded" is the
  question the two columns are for. An unqualified text selected there matches it in any column.

Cogpit's defaults for the filter, not the owner's decision (2026-10-14): it belongs to the view it
was set for, as the search does (opening another view clears it; a tab keeps its own, and
`ui.navigate` clears it when it hides the object); the states offered are those of the rows the
namespace filter and the search keep, alphabetical within a column, with each one's count and the
most severe colour its cells draw it in, and the selected ones no row carries listed with 0 so
that they can be unchecked; a state no row carries is not dropped from the filter; the button
reads All states, the one state or "N states", greyed while the rows carry none; on a page (the
Flux and Argo CD dashboards, the Overviews) it sits in the table's header beside the search, since
it filters the table only; at most 100 states of at most 512 bytes each.

Cogpit's defaults, not the owner's decision: the page's title Flux Dashboard in its header and tab
(the navigator reads Dashboard, as captured), three tiles a row under the navigator's sub-groups'
headings, the table's title Not Ready, and "Every Flux object is ready." when it is empty.

## D-35: What Argo CD's repository and cluster Secrets show

**Decided (owner, 2026-10-14):** option (b), the allowlist of non-credential keys for the window and agents, as built; the night decision below is confirmed.

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14).** Argo CD keeps its
registered repositories, credential templates and clusters as Secrets labelled
`argocd.argoproj.io/secret-type` (`repository`, `repo-creds`, `cluster`), and Lens's Argo CD
section shows them as Settings › Repositories (Name, Type, URL, Project, Age) and Settings ›
Clusters (Name, Server, Age) (Lens's documentation, "Using Argo CD", read 2026-10-13; not
captured). Those Secrets hold credentials: a repository's `password`, `username`,
`sshPrivateKey`, `tlsClientCertData` and `tlsClientCertKey`, `githubApp*`,
`gcpServiceAccountKey`, `bearerToken`; a cluster's `config`, its whole client configuration
(bearer token, client key, AWS or exec settings). Options:

| Option | For | Against |
|---|---|---|
| (a) Names only: the Secret's name, namespace and age, no value | Nothing of a Secret's data shown anywhere | Not Lens's views: no URL, type, project or server, which is what a user opens the views for |
| **(b) An allowlist of non-credential keys, for the window and agents (chosen)** | Lens's columns; an agent can answer "which repository does this Application pull from, is it registered, in which project"; a repository URL or a cluster's API server is configuration that Applications and kubeconfigs carry in the clear anyway | A URL can carry a token as user information (`https://user:token@host`), handled below; a list to keep in step with Argo CD's keys |
| (c) The allowlist for the window, nothing for agents | Agents see nothing of these Secrets | Agents lose what the Applications' own specs already tell them (their `repoURL`, their destination's server), for no gain in secrecy |

**Chosen (b)**, because the allowed values are configuration, not credentials, and every other
value stays where the Secrets view already keeps it (masked, revealed in the window only). The
allowlist: repositories `name`, `url`, `type`, `project`; clusters `name`, `server`,
`namespaces`, `clusterResources`, `project` (`kinds.ArgoRepositoryKeys`, `kinds.ArgoClusterKeys`).
**How it is held to:** the views are virtual kinds whose objects Cogpit builds itself
([`kinds/argocd_settings.go`](../../internal/model/kinds/argocd_settings.go), `argoDerive`): from
each labelled Secret only the name, namespace, UID, resourceVersion, creation time, labels, the
Secret type, the allowed keys' values (decoded, at most 2 KiB, control characters dropped) and the
other keys' names. No annotation is copied (kubectl's last-applied annotation repeats the values),
no other value is read, so no row, drawer, YAML, `object_get` result, search, CSV, audit line or
log line can carry one. User information in a URL (`url`, `server`) is replaced with
`<redacted>` (`https://<redacted>@github.com/org`), except an SSH user without a password
(`ssh://git@host`, `git@host:org/repo`), which names no secret (`RedactURLUserinfo`). The other
keys are listed by name ("Values Not Shown"), as the Secrets view lists key names, and a
repository's Authentication column says which method its key names imply (SSH key, username and
password, GitHub App, …) without reading a value. The drawer links to the Secret itself, whose
drawer masks its values as every Secret's does. Not chosen as part of (b): parsing a cluster's
`config` to show its authentication method or `insecure` flag (it would read the credentials to
find the method); Cogpit shows "config" as a key name only.

What stays: the cache holds these Secrets in full while a settings view, its drawer or an agent's
read uses them, as it holds every Secret the Secrets view watches (D-13, [H-66](../security/credentials.md#h-66-argo-cds-credentials-are-in-cogpits-memory-while-its-settings-are-read)).
Tests: `TestArgoSettingsNeverShowCredentials`, `TestArgoSettingsGolden`, `TestRedactURLUserinfo`
(model); `TestArgoCDKindsFollowTheirDefinitions` (engine); `TestArgoSettingsNeverReachAgents`
(commands: `object_get`, `resources_list`, `resources_find`, `view_rows`, the audit log, the
events and the log searched for every credential in every form); on kind `TestKindArgoCD`.

## D-36: How Argo CD's Sync writes the operation

**Decided (owner, 2026-10-14):** as built: Sync writes the Application's `operation` through the Kubernetes API, bound to the UID and resourceVersion; a sync with prune keeps asking twice (the dialog, then the destructive confirmation naming what Argo CD would prune) rather than Lens's one dialog.

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14).** Cogpit talks to the
Kubernetes API only, never to Argo CD's own API server (the brief), so Sync
(`argocd.application.sync`, `argocd.application.syncMany`) starts a sync the way Argo CD's API
server does for `argocd app sync`: it writes the Application's `operation`, and Argo CD's
controller runs it. Options for what the write carries and how it guards against a sync someone
else started:

| Option | For | Against |
|---|---|---|
| (a) The minimal operation (`initiatedBy`, `sync.revision`, `sync.prune`), bound to the UID only, refused while the cache shows an operation | Simple | Drops the Application's own sync options (`CreateNamespace=true`, `ServerSideApply=true`) and retry, which `argocd app sync` keeps; an operation started between the cache's last event and the patch is replaced |
| **(b) What `argocd app sync` writes without flags, bound to the UID and the resourceVersion Cogpit checked (chosen)** | The same sync the CLI and Argo CD's UI start: each source's target revision (`revision`, or `revisions` for a multi-source Application), the spec's `syncPolicy.syncOptions` and `retry`, `initiatedBy: cogpit`; a write over a newer version of the Application is refused by the API server (409), so an operation Cogpit has not seen is never replaced | A status update of the controller between check and write refuses the patch: Cogpit then waits for its cache to show the newer version (at most 3 s), checks again that no operation runs and sends the patch again, at most three times in all |
| (c) Argo CD's API server (`/api/v1/applications/<name>/sync`) | Argo CD's own checks (sync windows, its RBAC) | A second credential and endpoint per cluster; against the brief |

**Chosen (b).** Built ([`model/kinds/argocd_ops.go`](../../internal/model/kinds/argocd_ops.go)
`ArgoSyncOperation`, [`cluster/argocd.go`](../../internal/cluster/argocd.go),
[`commands/argocd.go`](../../internal/engine/commands/argocd.go)). Refused, with the reason in the
greyed menu item and the error, while the Application has an operation requested (its `operation`
field set) or running or terminating (`status.operationState.phase`). Not carried over from Argo
CD's API server, and so not checked by Cogpit: sync windows (the controller still honours them:
a denied sync waits or fails in the operation state) and Argo CD's own RBAC (the user's identity
needs `patch` on the Application; what the sync does is Argo CD's,
[H-67](../security/privilege-footprint.md#h-67-a-sync-acts-with-argo-cds-rights-not-the-users)).
Refresh needs no such guard: it sets an annotation the controller removes, bound to the UID only.
Tests: `TestArgoSyncOperation` (model), `TestArgoSyncPatch`, `TestArgoSyncConflicts` (cluster),
`TestArgoSyncWritesTheOperation`, `TestArgoSyncRefusedWhileAnOperationRuns`,
`TestArgoSyncChecksAgainAfterAChange`, `TestArgoActionsBoundToTheConfirmedUID` (commands); on kind
`TestKindArgoCD`.

## D-39: Pod metrics for a user who may list pods in some namespaces only

**Decided (owner, 2026-10-14): option (b), and a pod-metrics refusal affects only its namespace**
(below). The night build's record (the lead's night decision of 2026-10-13; the owner's open
question, built the same night): options (a) as before: the Pods list, a node's Pods table and
`metrics.get` ask the Metrics API for every pod in all namespaces, which RBAC refuses such a user as
it refuses the pods, so CPU and Memory stay empty and the state reads "forbidden"; (b) one `list` of
`pods.metrics.k8s.io` per namespace the Pods view reads pods in (the accessible namespaces, the
namespace filter's, else the context's own; those whose rules deny pods left out), one per poll,
shared by the Pods list, a node's Pods table and the drawers, only while they show, and the one list
in all namespaces for a user who may list pods there; (c) a `get` of each visible pod's metrics per
poll; chosen (b), because it costs one request per namespace and poll where (c) costs one per pod
(a list of 300 pods: 300 requests a minute instead of one per namespace), it reads what the Pods
view reads, so the columns fill for every row the user sees, and it changes nothing for a user who
may list pods everywhere (budget P12: one list per 60 s).

**Built 2026-10-13** ([`engine/metrics.go`](../../internal/engine/metrics.go) `usageListScopes`,
[`engine/drawerpods.go`](../../internal/engine/drawerpods.go); [architecture §9](architecture.md#9-metrics-internalmetrics)):
the Pods list subscribes the metrics hub to `{pods, <namespace>}` for each namespace its plan reads
(`nsPlan.watch`), to `{pods}` when it reads all namespaces, and to nothing while the plan is
pending; the feed starts once the view's kind has listed, so a read refused in all namespaces (a
RoleBinding in the context's namespace alone, before the plan follows the refusal) polls no metrics
there either. A node's Pods table polls the same scopes (the pods' plan), a namespaced object's its
namespace, as before; `metrics.get` of every pod fetches the same scopes, at most four at once, and
merges them. The hub needed no change: its scopes by namespace existed, and `Hub.Pod` reads a pod
from its namespace's list. Measured at the HTTP level (`TestBudgetNamespaceLimitedUser`, a fake API
server that enforces RBAC): a user whose one RoleBinding is in `team-a` lists `pods.metrics.k8s.io in
team-a` once and nothing in all namespaces; a cluster-admin lists `pods.metrics.k8s.io` once, as
before (`TestPodMetricsPerNamespaceOfThePlan`, `TestPodMetricsClusterWideForAUserWhoMayList`).

**Decided (owner, 2026-10-14): a pod-metrics refusal affects only its namespace.** The Metrics
API's RBAC is not the pods': a namespace where the user may list pods but not their metrics
answers 403 for that namespace's `list pods.metrics.k8s.io`. Until 2026-10-14 the hub counted that
as the Metrics Server failing for a minute (`ProviderRetry`) for every scope, asking Prometheus in
its place where it found one (G12's fallback). Now (built 2026-10-14,
[`metrics/hub.go`](../../internal/metrics/hub.go) `namespaceRefusal`): the refused namespace's
pods get empty cells and the state `forbidden` for that namespace (`Hub.State` of its scope, its
message naming the namespace and the API server's reason); the other namespaces keep the Metrics
Server's values; the Metrics Server's status stays as it is (no failure on the status line, no
`ProviderRetry`), and nothing stands in for it in the refused namespace. Measured at the HTTP level
(`TestBudgetNamespaceLimitedUser`, a fake API server that enforces RBAC: pods in team-a and team-b,
their metrics in team-a only): `list pods.metrics.k8s.io in team-a` once, `in team-b` once (the
refusal), no read of `services` (nothing looks for a Prometheus); before the change the same steps
sent three refused lists in team-b and the status line read the Metrics Server as forbidden; a
cluster-admin's counts are unchanged (one `list pods.metrics.k8s.io` in all namespaces). Tests:
`TestPodMetricsRefusalStaysInItsNamespace`, `TestPodMetricsRefusalWithoutPrometheus`, `TestBudgetNamespaceLimitedUser`.

Cogpit's defaults with it, not the owner's decision:

- **A drawer too.** A pod's drawer in a refused namespace (`get pods.metrics.k8s.io` refused) is
  the same refusal: no live values, the provider untouched, no Prometheus live values in their
  place; Prometheus's history of the pod, which the drawer asks for whatever the Metrics Server
  says, is still shown where there is a Prometheus (it is not limited by namespace, below).
- **A refusal is an answer.** It is kept for the TTL like values (15 s, longer while polled), so an
  agent's `metrics.get` right after the view's poll sends nothing more; the view polls the refused
  namespace at the list interval, as any other, so a RoleBinding added later shows within a poll.
- **Agents see which.** `metrics.get` of every pod, read per namespace, names the refused
  namespaces in `forbidden`; its status is the others' (`available`).
- **What stays provider-wide:** a refusal of every pod (the one list in all namespaces), of the
  nodes, a 401, and a Prometheus's refusal (its service proxy), as before.

What it leaves as it was, for the owner to know:

- **Prometheus is not limited by namespace.** Its queries go through the API server's service
  proxy; RBAC decides whether the user may use that proxy at all, not which namespaces' series come
  back. Cogpit asks Prometheus, while it stands in for the Metrics Server, for the same namespaces
  (the namespace is part of the query), but nothing on the server narrows it
  ([privilege footprint](../security/privilege-footprint.md)).

## D-37: The node shell

**Decided (owner, 2026-10-14):** option (b) of the night build's decision, which the owner
confirmed in a decision round: options (a) no node shell
until the owner decides ([H-7](../security/privilege-footprint.md#h-7-node-shell-is-root-on-the-node));
(b) Lens's behaviour: a privileged pod in `kube-system` on the node that shares the host's PID
namespace, entering PID 1's namespaces with `nsenter -t 1 -m -u -i -n -p`, under the rules of
[privilege footprint › Node shell](../security/privilege-footprint.md#node-shell); (c) only a
`kubectl debug node/…`-style pod (the host's namespaces and its root filesystem at `/host`, no
privileged flag, no nsenter); (b), because it is Lens parity (a free Lens feature), every
start is a destructive-level confirmation for every origin, the GUI's included (agents always need
the user's acceptance, D-2), and the pod's life is bounded; (c) stays a later option, as a second
profile, not the default. With it the owner decided two of the build's defaults: the default image
is the newest supported Alpine release, pinned by digest and kept current by Renovate (no longer
Lens's 3.19), and the deadline is a setting with a 2-hour default (both below, built 2026-10-14).
**Built 2026-10-13** (night build;
[`engine/nodeshell.go`](../../internal/engine/nodeshell.go),
[`commands/nodeshell.go`](../../internal/engine/commands/nodeshell.go),
[`cluster/nodeshell.go`](../../internal/cluster/nodeshell.go),
[`terminal/nodeshell.go`](../../internal/terminal/nodeshell.go)); on kind 1.36.1 the shell enters
the node and `hostname` answers its name, and the pod is gone after the session closes
(`TestKindNodeShell`).

What the owner decided on 2026-10-14, built the same day:

- **The deadline: a setting, default 2 h (owner, 2026-10-14).** Of the build's options (a) 1 hour,
  (b) 2 hours, (c) 8 hours, (d) a setting, the owner chose (d) with (b) as its default: Cluster
  Settings › Node Shell › Maximum duration, kept per context in `cluster-settings.json`
  (`nodeShellMaxMinutes`, empty: 2 hours). It is the pod's `activeDeadlineSeconds` and the
  container's `sleep`, so the pod ends by then even if Cogpit dies; the confirmation names it ("ends
  by itself after 1 hour 30 minutes, its maximum duration"), and so do the tab's notice and the
  toast of a failed delete. Only the user sets it, as the image ([D-38](#d-38-who-sets-the-node-shells-image)):
  agents and the CLI may only reset it (`ErrNodeShellGUIOnly`)
  ([`engine/nodeshell_duration.go`](../../internal/engine/nodeshell_duration.go),
  [`fyneui/clustersettings_nodeshell.go`](../../internal/ui/fyneui/clustersettings_nodeshell.go);
  `TestNodeShellMaxDuration`, `TestNodeShellMaxDurationSetting`,
  `TestClusterSettingsNodeShellMaxDuration`; on kind `TestKindNodeShell` sets 30 minutes and finds
  `activeDeadlineSeconds: 1800` and `sleep 1800` on the pod).
- **The default image: the newest supported Alpine release, pinned by the digest of its multi-arch
  index, kept current by Renovate (owner, 2026-10-14).** Alpine's release table
  ([alpinelinux.org/releases](https://alpinelinux.org/releases/), read 2026-10-14) lists v3.24
  (branched 2026-06-09, minor releases 3.24.0 to 3.24.2, supported until 2028-06-01) as the newest
  stable branch, then v3.23 (until 2027-11-01), v3.22 (until 2027-05-01) and v3.21 (until
  2026-11-01); 3.19 is past its end. Chosen: `docker.io/library/alpine:3.24@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6`,
  the digest read on 2026-10-14 at 07:45 UTC with `docker buildx imagetools inspect
  docker.io/library/alpine:3.24` (the tags `3.24.2` and `latest` name the same index; its eight
  images, linux/386, amd64, arm/v6, arm/v7, arm64/v8, ppc64le, riscv64 and s390x, are 3.24.2, built
  2026-09-17). `nsenter` and `sleep` are BusyBox 1.37.0 applets in it (`/usr/bin/nsenter`,
  `/bin/sleep`), checked with `docker run` on linux/arm64 (the index) and linux/amd64 (its image):
  `nsenter --help` lists `-t`, `-m`, `-u`, `-i`, `-n`, `-p`, and `sleep 1` exits 0; on kind 1.36.1
  (arm64) `TestKindNodeShell` enters the node with it. **Renovate keeps it current:** the constant
  `DefaultNodeShellImage` in [`engine/nodeshell.go`](../../internal/engine/nodeshell.go) carries a
  `// renovate: datasource=docker depName=docker.io/library/alpine` comment that a custom regex
  manager in [`.github/renovate.json`](../../.github/renovate.json) reads (tag and digest), so a
  new Alpine release (3.25, a minor update) moves the tag and a rebuilt 3.24 moves the digest; a
  dry run of Renovate 44.155.8 (`--platform=local --dry-run=lookup`) extracted 3.24 with its digest
  and found nothing newer, and on a copy pinned to 3.23 it proposed 3.24 with the digest above and
  3.23's newest digest. The tests take the image from the constant, so Renovate's pull request
  passes them, and CI's kind job opens a node shell with the new image before it merges. The texts
  name the image from the constant (`alpine:3.24` on the settings page and in the command's
  description); this page names the pin of 2026-10-14, the constant is the current one. Lens's
  default stays the tag `docker.io/alpine:3.19`: Cogpit's default differs from it, the setting is
  Lens's. H-72 changed with it ([privilege footprint](../security/privilege-footprint.md#h-72-the-default-node-shell-image-ages-with-the-build-that-pins-it)).

What the build chose, Cogpit's defaults (not the owner's decision):

- **The maximum duration's bounds: 5 minutes to 24 hours**, in whole minutes, typed as `90m`, `2h`
  or `1h30m` (Go's duration form; the field shows a set value back in that form, `1h30m` for
  `90m`, and a line under it says it in words). Five minutes leave time to pull the image and look
  at something; a day covers a long working session without leaving an idle root pod for days.
  A value outside them, `2 hours`, `90` or `10m30s` is refused under the field; a file holding one
  reads as the default.
- **Renovate's updates of the image merge on their own** once CI passes (a minor release or a new
  digest; a major, Alpine 4, waits for review), as the other pins of the custom regex manager do,
  and they are `build(deps):` commits, which semantic-release does not release: a new default image
  reaches users with the next `feat:` or `fix:` release.
- **A floating tag is accepted** in the setting (options: (a) refuse an image without a digest,
  (b) accept it and say so; chosen (b), as Lens takes any image): the confirmation then says "a
  floating tag, not pinned by digest: what it runs can change without Cogpit knowing", and the
  settings page says so under the field.
- **The pod:** `kube-system`, `nodeName` set, one toleration `operator: Exists`, `hostPID: true`,
  the container `privileged: true`, no service account token, no service links,
  `restartPolicy: Never`, a 1-second grace period, labels `app.kubernetes.io/name:
  cogpit-node-shell` and `app.kubernetes.io/managed-by: cogpit`, a name
  `cogpit-node-shell-<8 random characters>`; an image pull secret named in the setting. The shell
  runs `nsenter -t 1 -m -u -i -n -p -- env TERM=xterm-256color HOME=/root sh -c` with the node's
  bash as a login shell, else its sh.
- **Its end:** the pod is deleted (with its UID as precondition) when the shell ends, when its tab
  closes (`dock.close`, by anyone, or since 2026-10-14 with its context when that leaves Cogpit:
  its kubeconfig source removed, its file gone) and when Cogpit quits (waiting at most 10 seconds); a failed
  delete is a warning toast, and the pod then ends by its deadline. **Since 2026-10-14 (owner,
  [H-71](../security/privilege-footprint.md#h-71-a-crash-leaves-the-node-shells-pod-until-the-next-connect)):**
  every pod is recorded in `nodeshells.json` before its create is sent (context, name, a random id
  the pod carries as `cogpit.io/node-shell-id`, then its UID), and on the next connect to its
  context Cogpit deletes the pods a crash or a failed delete left, by UID, and says so in a
  notification; never a pod it did not create. Cogpit's defaults in that build (not the owner's
  decision): the entry is written before the create, so a crash between the two leaves an entry
  without a UID, whose pod is read by name and recognised by the id (options: (a) before the
  create with the id, (b) after the create only, which a crash between the two would leave
  unrecorded; chosen (a)); a journal that cannot be written refuses the node shell; the cleanup
  follows every connect, asks nothing and is audited as `cogpit.nodeShellCleanup` with origin
  gui; another live Cogpit process's entries are left to it; without a configuration directory
  the journal lives in memory. The start waits for the pod
  with one get and a watch on that pod alone, at most 3 minutes; a pod that cannot pull its image
  or fails ends the session with the reason and is deleted.
- **What is confirmed binds what runs:** the confirmation's detail names the namespace, the node,
  the image, the pull secret and the maximum duration; a change of the image, the pull secret or
  (since 2026-10-14) the maximum duration while the dialog was open, a node created again under its
  name, or another context refuses the start before anything is created.
- **A digest wider than the dialog breaks** (2026-10-14): the confirmation's text breaks a word
  wider than the dialog at the widest part that fits and flows on after it (`wrapFlow`), since the
  3.24 digest no longer fit on one line of the dialog and Renovate changes it.
- **The access check asks in `kube-system`:** a command's declared verb may name its one
  namespace (`Verb.Namespace`, new), so a user who may create pods in `kube-system` only is not
  greyed by a cluster-wide question.

## D-38: Who sets the node shell's image

**Decided (owner, 2026-10-14):** option (b) of the night build's decision: options (a) agents set
the node shell's image and pull secret too, as the other cluster settings (UI commands, D-2); (b)
only the user sets them, in Cogpit's window, and agents and the CLI may only reset them to the
defaults, as a cluster's proxy (`ErrProxyGUIOnly`); (c) no setting, the pinned default only; (b),
because the image is what runs as root on the node at the user's next node shell, and an agent that
could choose it and then ask for a node shell would leave the user one careful reading of the
confirmation away from running the agent's image privileged; Lens has the setting, so (c) would
lose parity. The owner put the node shell's maximum duration under the same rule (2026-10-14,
[D-37](#d-37-the-node-shell)): it decides how long a root pod lives. **Built 2026-10-13**, the
maximum duration 2026-10-14 (`cluster.settings.setNodeShell` refuses a value that is not the
default from every origin but the GUI with `ErrNodeShellGUIOnly`, before anything is written;
`TestNodeShellSettings`, `TestNodeShellMaxDurationSetting`).

## D-40: A Pods table in the workloads' drawers

**Decided (owner, 2026-10-14): option (b), without a Namespace column in the workloads' drawers;
the node drawer keeps it.** A workload's pods share its namespace, so the column said nothing
there. Built 2026-10-14 ([`kinds/nodes.go`](../../internal/model/kinds/nodes.go)
`workloadPodColumns`; `TestWorkloadDrawerPods`, `TestDeploymentDrawerPodsTable`; PNGs re-rendered:
[Deployment dark](../../internal/ui/fyneui/testdata/deployment_pods_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/deployment_pods_drawer_light.png),
[Stateful Set dark](../../internal/ui/fyneui/testdata/statefulset_pods_drawer_dark.png),
[light](../../internal/ui/fyneui/testdata/statefulset_pods_drawer_light.png); the node drawer's
[PNG](../../internal/ui/fyneui/testdata/node_pods_drawer_dark.png) unchanged). The night build's
record follows.

**Night decision (the lead's, 2026-10-14, now decided by the owner, above):** options (a) no Pods table in
the workloads' drawers, as before; (b) the node drawer's Pods table (Name, warning icon, Node,
Namespace, Ready, CPU, Memory, Status; sortable; CPU and Memory from the metrics hub, polled only
while the table shows) in the drawers of Deployments, Replica Sets, Stateful Sets, Daemon Sets and
Jobs; (c) a "Pods" link that opens the Pods view filtered to the workload; chosen (b), because it is
what a Lens user expects from a workload's details (inferred: Lens's workload drawers are not
captured), and the node drawer's table, its metrics and its sorting already exist, so the cost is
the index of pods by their owner, which only an open drawer holds. **Built 2026-10-13** (night
build; [`kinds/workloadpods.go`](../../internal/model/kinds/workloadpods.go),
[`engine/drawerlimit.go`](../../internal/engine/drawerlimit.go), `KindProfile.DetailNeeds`;
`TestWorkloadDrawerPodsPerKind`, `TestWorkloadDrawerPodsFollowPods`,
`TestWorkloadPodsIndexOnlyWithTheDrawer`, `TestWorkloadDrawerPodsMetrics`,
`TestWorkloadDrawerPodsSort`, `TestDeploymentDrawerPodsTable`; on kind 1.36.1
`TestKindWorkloadDrawerPods`: a Deployment of two replicas lists its two pods over the engine, and a
deleted pod's replacement).

What the build chose, Cogpit's defaults (not the owner's decision):

- **Cron Jobs too, through their jobs.** Options: (a) the five kinds the decision names, (b) Cron
  Jobs as well, the pods of the jobs they control; chosen (b): `object_get`'s `related: true` lists
  a Cron Job's jobs and their pods already, the cost is one more index (jobs by owner) that only the
  drawer holds, and the pods of the last runs (the jobs' history limits keep three succeeded and one
  failed by default) are what a user checks after a run. Replication Controllers are left out, as
  `related: true` leaves them out.
- **The node drawer's eight columns, Node included** (superseded 2026-10-14: the owner removed
  Namespace, so a workload's table has seven, Node kept, option (c)). Options: (a) the seven of the
  lead's brief, without Node; (b) the node drawer's eight; (c) seven without Namespace, which a
  workload's pods share; the night build chose (b): Lens's node drawer shows a Node column that is redundant there (capture,
  2026-10-11), which suggests one shared table whatever the drawer (inferred); Node is what tells a
  Daemon Set's pods apart; and one builder, one set of column ids and one sort state ("Pods") serve
  both drawers. The Node cell links to the node (a node's own drawer keeps it plain). The cost: in
  the default drawer width a workload's pod names, which share their prefix, were cut with "…";
  without Namespace (2026-10-14) the Deployment's fixture names fit
  ([PNG](../../internal/ui/fyneui/testdata/deployment_pods_drawer_dark.png)). The column ids, the
  sort state ("Pods") and the builder stay shared; a sort by Namespace set for a node's table
  sorts a workload's by name, which has no such column (`SortDrawerTables` falls back to the
  default order).
- **A Deployment's pods are those of every replica set it controls**, old ones during a rollout
  included. Options: (a) the newest replica set's only, (b) every one it controls; chosen (b): the
  old pods still serve and terminate during a rollout, which is what a user watches; old replica
  sets kept at zero replicas have no pods and add no row.
- **Owned means the controller reference's UID, in the workload's namespace.** Options: (a) the
  owner's kind and name, (b) the controller reference's UID, counted only in the object's own
  namespace; chosen (b): a workload created again under its name owns none of the old one's pods,
  and Kubernetes's garbage collector treats an owner reference across namespaces as absent; the
  workload logs and `related: true` use the same rule (`controllerUID`).
- **The index lives with the drawer** (`DetailNeeds`). Options: (a) in the profiles' `Needs`, which
  makes every workload list rebuild its rows on every pod change (measured: eleven Deployments
  snapshots for ten pod changes and one change of a Deployment, against one without); (b) the
  drawer's own indexes, held while it is open; (c) no index, every drawer build scanning the pods;
  chosen (b), which also wakes the drawer only for its own pods and replica sets
  (`TestWorkloadPodsIndexOnlyWithTheDrawer`). Its cost: an open workload drawer holds a watch on the
  pods (and the replica sets or jobs) as `Backend.Acquire` reads them, all namespaces or the
  namespaces the access reviews allow, shared with every other reader and warm for the grace period
  after the drawer closes; opened alone it is one list and watch more.
- **At most 500 rows, the rest counted** (`engine.MaxDrawerTableRows`), for every sortable drawer
  table, a node's Pods included. Options: (a) no bound, as the node drawer had; (b) 200, the bound
  of `related: true`; (c) 500; chosen (c): a node runs 110 pods by default and at most a few
  hundred, so only a large workload's table is cut; the cut follows the sort (sorted by CPU, the
  busiest pods), and the window says "N more pods not shown" under the rows.
- **The place:** after the kind's own sections (Conditions included), before the Events; a workload
  without pods, or whose pods or owners are still listing, has no Pods section (inferred).
- **`related: true` reads the table like every section**: the pods first, in the table's order,
  each pod's node with the relation "Pods" as a node drawer's namespaces have it, then the replica
  sets or jobs and any pods the table cut. Options: (a) so; (b) leave the workload's own Pods out of
  the section scan, keeping the replica sets before the pods; chosen (a): the documented rule
  ("every object the drawer links to, in its order, then what a workload controls") holds without
  an exception.

## D-41: What survives a restart

**Decided (owner, 2026-10-14):** as built: the tabs with their per-tab state and the tables' column choice and order are kept and restored without connecting anything; column widths are not kept.

**Context.** Until 2026-10-13 everything the user arranged in the window but the hotbar, the
preferences, the cluster settings and the panel widths ended with the run: the tab bar's tabs and the
tables' hidden columns and column order ([ui-parity › Layout](../parity/ui-parity.md#layout),
[› Tables](../parity/ui-parity.md#tables), G5, G13). Whether Lens restores its tabs is not captured
(inferred: it does). **Built 2026-10-13** ([`engine/persist.go`](../../internal/engine/persist.go),
[`engine/persist_restore.go`](../../internal/engine/persist_restore.go); `TestWorkspaceRoundTrip`,
`TestWorkspaceNoRequestAtStart`, `TestWorkspaceDropsOnlyWhatIsCertainlyGone`,
`TestWorkspaceUnreadableFileIsKeptAside`, `TestWorkspaceWritesAfterTheChangesStop`).

- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): whether and how the tabs come
  back.** Options (a) not kept, as before; (b) kept in Cogpit's own owner-only file and restored at
  start without connecting anything, a tab connecting its cluster when the user shows it; (c) as (b),
  and the cluster of the tab shown last connected at start, as a browser reloads its tabs; chosen (b),
  because (c) would send requests to a cluster, possibly a production one, that the user has not
  asked for in this run (Never surprise a cluster), and (a) loses what the user arranged. The file is
  `<user config dir>/Cogpit/workspace.json` (0600 in a 0700 directory, replaced atomically,
  `writePrivateFile`), written 1 second after the tabs or the layout stop changing (at least every
  5 seconds while they keep changing) and when Cogpit quits; read once at start.
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): what each tab keeps.** Options
  (a) its cluster and view only; (b) its cluster, view and the view's title, preview or fixed, its
  place in the bar, when it was last shown, its namespace and API group filters, its search text with
  the match case and regular expression toggles, its status filter (since 2026-10-14, D-34), its
  sort and its history; (c) (b) plus its selection
  and the drawer's object; chosen (b), because it is the tab's own state that the bar keeps per tab
  today, while a selection and a drawer name objects that may be gone at the next start, and a
  drawer restored at connect would read an object the user did not ask for in this run.
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): the tab shown last.** Options
  (a) no tab marked at start; (b) the tab shown last marked and its cluster named as
  `UIState.Context`, disconnected, the state `cluster.disconnect` leaves (the welcome page shows, the
  tab is marked, showing it connects); (c) marked and connected; chosen (b), because it shows where
  the user left without a request ((c) is (c) above), and it is a state the engine and the window
  handle already. A cluster connected from the picker, the hotbar or the welcome page shows its most
  recently used tab, as within a run.
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): tabs of contexts that are
  gone.** Options (a) restored, failing when shown; (b) dropped with a notice; chosen (b), and only on
  a plain fact, as a connection closes only on one (`CloseGoneSourceConnection`): the context's file
  does not exist any more, its source was removed, its file parses and no longer defines it, or the
  connector's list, read without an error, does not have it (the demo cluster, a run without a
  configuration directory). A file that does not parse for a moment keeps its tabs. The notice ("2
  tabs of the last run were not restored: no kubeconfig source offers dev [dev.yaml] any more.") is
  in the notification log (the title bar's bell) and shown as a toast once the window is up
  ([D-42](#d-42-the-start-screen)).
- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13): the tables' layout.** Options
  (a) the hidden columns and the column order per view key, the same on every cluster, as
  `UIState` holds them; (b) per cluster and view; chosen (a), because a view's columns are its kind's,
  the same on every cluster, and a user who hides a column on one cluster's Pods wants it hidden on
  the next. **Column widths:** options (a) not kept (the window's, not engine state, as now); (b)
  kept in Fyne's preferences as the panels' widths; (c) engine state in this file; chosen (a) for
  now, because the table records a width whenever a header's laid-out width differs from the one it
  set, which does not yet tell a user's drag from a relayout, and keeping a relayout's width across
  a restart would make it stick; (b) once the drag's end is its own event. A question for the owner.
- **Not restored:** the dock's tabs (logs, terminals, shells, editors, Helm tabs, the Audit Log):
  a stream or a terminal would open at start, and an editor's unsaved text has its own draft,
  offered when its cluster connects ([D-11](#d-11-crash-stance)); a selection and the drawer (above);
  the connection itself.

Cogpit's defaults, not the owner's decisions: the delays; the bounds of a file read back (200 tabs,
1,000 namespaces, groups or column ids each, 1,000 views, keys of 512 bytes, a search of 1,000 bytes,
titles of 256 bytes; what is beyond them is dropped without costing the rest); a file that does not
load (not JSON, a newer format) is ignored with a warning notice and moved to
`workspace.json.unreadable` before the first write, never overwritten; two Cogpit processes keep a
copy each and the last write wins, as with the hotbar.

## D-42: The start screen

**Decided (owner, 2026-10-14):** option (b), the welcome page at start without the context picker, as built.

**Context.** Until 2026-10-13 the context picker opened by itself whenever Cogpit started without a
connection, over the welcome page and its cluster rows ([ui-parity › Layout, Welcome
page](../parity/ui-parity.md#layout): "Differs: the context picker still opens by itself at start,
over the page"). Lens starts on its own page, not on a picker (inferred; its welcome page is not
captured). Since D-41 the start can also show the tabs of the last run. **Built 2026-10-13**
([`fyneui/shell.go`](../../internal/ui/fyneui/shell.go) `start`;
`TestStartShowsRestoredTabsWithoutConnecting`, PNGs
[dark](../../internal/ui/fyneui/testdata/start_restored_dark.png),
[light](../../internal/ui/fyneui/testdata/start_restored_light.png)).

- **Decided (owner, 2026-10-14, as built; the night decision of 2026-10-13):** options (a) as before: the
  picker opens by itself at start whenever no cluster is connected; (b) no picker at start: the
  welcome page shows with its cluster rows (the hotbar's bank, the clusters of the tabs, the others
  the kubeconfigs offer), the picker opens with ⌘P, All Clusters… or the palette's Connect to
  Cluster; (c) the picker only when no tab was restored; chosen (b), because the welcome page
  already offers every cluster to connect with one click and the picker over it hid exactly that,
  because it is what the welcome row names as the difference from Lens, and because (c) would make
  the first screen depend on a file the user does not see. `--context` and `--demo` connect at start
  as before; a `--context` that fails opens the picker with the error beside the context, where a
  login program not allowed yet can be allowed (Cogpit's default).
- With it (Cogpit's default): a notice the engine gave before the window was up (the tabs of the
  last run not restored, an unreadable `workspace.json`) is shown as a toast once the window is up,
  with the crash notice, rather than only in the notification log.

## D-43: Where the nodes' disk comes from

**Decided (owner, 2026-10-14):** option (b), node-exporter's root filesystem through Prometheus, as built; without node-exporter the Disk column stays empty (no cAdvisor fallback, never `nodes/proxy`).

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14):** Lens's Nodes list shows a
Disk usage bar (capture, 2026-10-11); the Metrics Server measures no disk. Options: (a) the
kubelet's summary API (`/api/v1/nodes/<node>/proxy/stats/summary`), which every cluster has; (b)
node-exporter's root filesystem (`node_filesystem_size_bytes` and `node_filesystem_avail_bytes` of
the mountpoint `/`) through the cluster's Prometheus, matched to nodes by their node label or
instance; (c) cAdvisor's filesystem series of the node's root cgroup through Prometheus (what the
node drawer's Filesystem chart reads), which needs no node-exporter; chosen (b), because (a) needs
`get nodes/proxy`, a permission that reaches the kubelet's whole API, running commands in the
node's pods included ([privilege footprint](../security/privilege-footprint.md), Metrics), which
Cogpit must not ask a user to grant for a column; and (c) sums every block device of the node, not
the root filesystem a "Disk" bar means, and kube-prometheus-stack drops those series (G12). The
price of (b): a cluster without Prometheus and node-exporter shows an empty column, whose header's
tooltip says why. **Built 2026-10-14** ([`metrics/disk.go`](../../internal/metrics/disk.go),
[`engine/drawerdisk.go`](../../internal/engine/drawerdisk.go); `TestNodeDisksFromNodeExporter`,
`TestNodeDiskBudget`, `TestNodesListDiskColumn`, `TestNodesListDiskBudget`; architecture §9).

What the build chose, Cogpit's defaults (not the owner's decision):

- **One query per poll, the list's 60 s,** for every node at once: both gauges in one answer,
  marked by a `part` label (`label_replace(…) or label_replace(…)`), `max by (node, instance)`
  so that two jobs scraping one exporter count once. Options: (a) two queries (size, available),
  (b) one; chosen (b), half the requests for the same numbers.
- **Used is the size less the available bytes** (`df`'s Avail: what an unprivileged process may
  still write), so the share counts the blocks reserved for root as used. Options: (a) size less
  available, (b) size less free (`node_filesystem_free_bytes`); chosen (a), the share at which
  workloads stop being able to write.
- **A node is matched by its node label, else its instance's host as the node's name, else as one
  of the node's addresses** (InternalIP, ExternalIP, Hostname). The prometheus chart's service
  discovery adds the node label; an exporter scraped with host networking under its own address is
  found by the node's InternalIP. Inferred, not checked against a real node-exporter (the kind
  cluster's Prometheus has none).
- **The Nodes list looks for Prometheus while its Disk column shows** (before: only when a drawer
  opened or the Metrics Server failed). The cost on a cluster with Prometheus: the detection's
  probe (once per 10 minutes) and the watch on Services the detection reads, then one query per
  poll; a cluster whose only provider is the Metrics Server alone (the demo cluster) asks nothing.
- **Hidden with the nodes or the filesystem** (Cluster Settings › Metrics), and not polled while the
  user put the column away (the ⋮ column list), unlike CPU and Memory, which only the cluster's
  settings stop: the disk costs Prometheus a query, the usage the Metrics Server a list the Pods
  view shares.
- **A node's drawer shows its disk** as a line under the node's chart ("Disk (/): 41.2 GiB of 58.4
  GiB used (71%)", in the bars' colour at the thresholds), polled while the drawer is open, shared
  with the Nodes list's poll; Lens's drawer has no disk line that is captured (inferred).

## D-44: What agents read of the dock's logs and terminal tabs

**Decided (owner, 2026-10-14):** option (b), as built: logs tabs freely, a terminal tab only after the user turned on its switch, off for every new tab.

**Added (owner, 2026-10-14, [H-83](../security/agents-and-mcp.md#h-83-an-allowed-terminal-tab-gives-agents-everything-it-shows-from-then-on)):**
a visible marker on the tab while its switch is on, flashing at each agent's read; the switch turns
itself off after 15 minutes without an agent's read and says so in the tab. **Built 2026-10-14**
([`engine/agentread.go`](../../internal/engine/agentread.go),
[`fyneui/dockagentread.go`](../../internal/ui/fyneui/dockagentread.go)): the switch, the last read,
its end and that it turned itself off are UI state (`DockTab.AgentLastRead`, `AgentReadUntil`,
`AgentReadExpired`); a read emits `EventDockRead`, to which the window reacts with the flash; one
timer on the session's clock (`Options.AfterFunc`) ends the earliest switch.
Cogpit's defaults in the build (not the owner's decision): the 15 minutes count from the later of
switching on and the last read; every read the switch allowed counts (an agent's, or the CLI's
forwarded to the window), the user's own never; the marker an eye after the title in the warning
colour, the flash the tab tinted in that colour for 0.7 s, again at a read during a flash; the note
("Turned off after 15 minutes without an agent's read") beside the switch while that tab is the
active one, gone when the user turns it on again, none when the user turns it off by hand; the end
of a switch is not an audit entry (no one's command; `ui_state` shows it).

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14):** a terminal tab's output can
hold anything the user typed or a program printed, credentials included (a `cat` of a token file,
`env`, `kubectl create token`, a Secret piped through `base64 -d`). Options: (a) agents read terminal
output freely, as they read logs; (b) terminal output only after the user allows it per tab, with a
switch on the tab that is off when the tab opens, logs freely; (c) never terminal output, logs
freely; chosen (b), because a logs tab shows cluster data that the user's access already governs
and that is an agent's to read like events and YAML (D-2: reads run freely), while a terminal is
the user's own workspace on this computer, whose output neither the cache's redaction of Secret
values nor any other rule of Cogpit's can see; (a) would hand all of it to every connected agent,
and through the built-in agent to its LLM endpoint, without the user knowing; (c) loses the case the
feature exists for ("read the error in my terminal") and leaves the user to copy and paste, which
is (b) by hand and less visible. **Built 2026-10-13** (night build, G17;
[`engine/dockoutput.go`](../../internal/engine/dockoutput.go),
[`commands/dockoutput.go`](../../internal/engine/commands/dockoutput.go),
[`fyneui/dockoutput.go`](../../internal/ui/fyneui/dockoutput.go),
[`termview/output.go`](../../internal/ui/fyneui/termview/output.go)): `dock_read` reads a logs tab's
lines as its view shows them, a terminal tab's screen only once the user turned on the tab's
"Agents may read" switch (`dock.allowAgentRead`, GUI-only: no agent can turn it on);
`TestDockReadTerminalNeedsTheUsersSwitch`, `TestDockOutputOfATerminalTab`.

What the build chose, Cogpit's defaults (not the owner's decision):

- **Every terminal tab is a terminal:** the local terminal, a pod shell, attach and the node shell
  all need the switch (a container's shell prints its environment's Secrets as readily as a local
  one). Options: (a) all four, (b) the local terminal only; chosen (a).
- **Off whenever a tab opens**, also one an agent opened (`terminal_open`, `pod_shell`), and not
  remembered across tabs or restarts; closing the tab ends it.
- **No dialog to turn it on** (options: (a) a plain switch, (b) a dialog whose Allow button arms
  after the confirmations' 750 ms, as the allowance of agent writes has; chosen (a)): the switch sits
  in the strip of the tab the user is looking at, no command opens a dialog or moves the pointer
  there, and turning it on exposes one tab, not every write in a cluster. But an agent can make
  another terminal tab the active one (`dock_activate`, a UI command) just before the user's click
  lands, so turning the switch on is ignored for the confirmations' 750 ms after it began to show
  another tab (turning it off never is; `TestAgentReadSwitchArmsAfterATabChange`).
- **What agents read:** a logs tab's lines its view shows (the pod and container it narrows to, its
  filter applied, resource names and timestamps as the tab shows them, without escape sequences);
  a terminal's screen as text (its rows, no colours or escape sequences). The terminal widget keeps
  no scrollback (fyne-io/terminal overwrites the top row when the screen scrolls; read in the fork's
  `scrollDown`), so "screen and scrollback" is the screen.
- **Bounds:** 200 lines by default, at most 2,000 and 256 KiB a page, each line at most 8 KiB; paging
  by the lines' numbers (`after`, `before`), which stay while the tab holds the lines.
- **A read and the switch:** a read that the user's switching off (or closing the tab) overtakes
  returns nothing; reads are audited like every agent command, with their parameters, never the
  lines.
- **The built-in agent** (not built) is an agent like the others for the switch; what it then sends
  to its LLM endpoint falls under the endpoint's rules
  ([H-13](../security/agents-and-mcp.md#h-13-cluster-data-leaves-for-the-llm-endpoint)).

**Decided (owner, 2026-10-14), the answer to [H-82](../security/agents-and-mcp.md#h-82-secret-values-printed-into-logs-reach-agents):**
the values of the cached Secrets, from about 8 characters (against false matches, measured over
10,000 log lines), are replaced in what an agent reads (`dock_read`'s log lines, `object_get`'s and
`editor_get`'s YAML) by `«secret <ns>/<name>/<key>»`; the window shows the text as it is. **Built
2026-10-14** ([`engine/secretscrub.go`](../../internal/engine/secretscrub.go),
[`engine/agentview.go`](../../internal/engine/agentview.go),
[`cache/peek.go`](../../internal/cache/peek.go)): one filter in `Dispatch` for every result of an
agent's command; a matcher over 8-byte windows (a bit filter and an index, one pass over the text
whatever the number of values) built from the Secrets the cache holds, again only when they changed;
the measurement and its numbers are in H-82. The build's choices, **the lead's decisions for the
owner to confirm**:

- **The base64 form is replaced too** (options: (a) the decoded value only; (b) the decoded value
  and the value as stored, base64, plus the forms a log prints a value in: JSON-escaped, a PEM
  private key's body line by line, a registry credential's passwords and auth strings; (c) also
  URL-encoded and other encodings): chosen (b). The base64 form is what a Secret's own manifest
  carries, so a copy of one in another object's YAML or annotation
  ([H-23](../security/agents-and-mcp.md#h-23-secret-annotations-other-than-kubectls-reach-agents))
  or a debug dump in a log holds it; it costs one more pattern per value, and the 10 planted leaks
  of a base64 value are caught through it alone. (c) multiplies the patterns for forms no measured
  line held.
- **Minimum 8 bytes**, the owner's figure, in every form; the measurement shows the minimum is not
  what decides the false matches (H-82's table).
- **A value under 16 bytes only where it stands alone** (no letter, digit, `_` or `-` touching it):
  "postgres" is not replaced inside "postgresql" or "postgres-0" (options: every match; stand-alone
  for short values, chosen; stand-alone for every value, which would miss a token glued to other
  text). It halved the lines with a name replaced (2,740 to 1,238) and caught the same leaks.
- **The values of name keys are left out**: a key whose last word is user, username, login, host,
  hostname, server, port, database, db, dbname, namespace, schema, region or email. Options: (a)
  every value, as the owner's words say; (b) leave name keys out, chosen; (c) a higher minimum (10
  bytes: 73 lines, but 5 more planted leaks missed). With (a) 1,238 of 10,000 lines (12 %) had a user,
  host or database name replaced, which hides what a line is about from the agent and protects
  nothing a credential protects; (b) brought it to 0 and caught the same leaks. This narrows the
  owner's rule, hence a question for the owner; the same kind of allowlist as D-35's.
- **Also left out:** certificates (public, D-30: the cluster CA that every namespace's
  `kube-root-ca.crt` holds stays readable), a service account token Secret's `namespace` and
  `ca.crt`, a bootstrap token's non-secret keys, Helm's release records (their payload is never
  printed), values over 64 KiB.
- **Every `dock_read` page, a terminal tab's too**, not only logs tabs: a terminal the user allowed
  shows what programs print as a log does
  ([H-83](../security/agents-and-mcp.md#h-83-an-allowed-terminal-tab-gives-agents-everything-it-shows-from-then-on)).
- **Also the drawer's values, table cells and events, an editor's diff, an apply's and a revert's
  text** in what an agent reads, not only the YAML: they carry the same values. The CLI is not
  filtered (the user at a terminal).
- **Which Secrets: decided (owner, 2026-10-14), option (b).** Options: (a) only the Secrets cached
  already (the build's first choice: before the Secrets view was opened nothing was replaced);
  (b) an agent's read that the filter reads takes the Secrets as the Secrets view reads them, waits
  for them and lets them go, the cache keeping them warm for its warm period; (c) the Secrets
  watched whenever the MCP server is on; (d) a GET per read. The lead recommended (b): only then
  does the filter work when an agent reads a log, at the cost of one watch (a read) and the values
  in memory while agents read, which the Secrets view costs anyway; (c) holds them without any
  agent's read; (d) is a request per read. **Built 2026-10-14**
  ([`engine/agentsecrets.go`](../../internal/engine/agentsecrets.go); `TestAgentViewReplacesCachedSecretValues`,
  `TestAgentReadWithoutSecretsAccess`, `TestBudgetAgentReadTakesTheSecrets`: the first agent read
  11 ms and one streaming watch of Secrets on the fake API server, a second within the warm period
  no request). Cogpit's defaults in it (not the owner's decision): the wait is bounded by 10 s, and a
  read whose Secrets were not listed by then is refused (`ErrAgentSecretsPending`, fail closed)
  rather than sent as printed; a user who may not list Secrets, or may list them in none of the
  namespaces Cogpit reads, is not waited for; only results the filter reads take the Secrets
  (`ui_state`, a list, a search take none, P11).
- **An agent's text that carries a placeholder or `<withheld…>` back is refused**
  (`ErrAgentMarkerWrite`) where a Secret's text is (`GuardSecretEditor`, `GuardSecretText`):
  writing it would replace the value in the cluster (never surprise a cluster). Substituting the
  values back was the other option; rejected, since it would let an agent place a Secret's value
  where it chooses without reading it.
- **The placeholder** names one Secret when several hold the same value (the first by namespace,
  name and key), and overlapping values are replaced together.

## D-46: Chart icons from the hosts that serve them

**Decided (owner, 2026-10-14):** option (b), icons fetched from the hosts that serve them with the built safeguards, "Load chart icons" on by default, as built.

**The night decision (the lead's, 2026-10-14), confirmed by the owner (2026-10-14):** a chart repository's index
names each chart version's icon as a URL on any host (a CDN, GitHub, the project's own site), and
Lens shows icons in its Charts view (inferred; neither its icon column nor its chart drawer is
captured). Fetching one tells whoever serves it that this computer looked at the chart
(its address, the time, the URL), and an index can name any URL, a host in the user's own network
included. Options: (a) no icons: nothing fetched, Lens's look not met; (b) icons fetched from their
URL when the window shows them, https only, without credentials, bounded in size, a PNG or JPEG
decoded under bounds and an SVG checked, kept in Cogpit's cache by the URL's hash for a while, with
a Preferences switch; (c) icons only from the repository's own host, which already learns what the
user looks at from the archives and indexes (most public repositories name icons elsewhere, so most
charts would show no icon); chosen (b) with Preferences › Kubernetes › "Load chart icons" on by
default, because the icon is how a user finds a chart in a long list, and the cost is a request per
icon shown, to a host the chart's author chose, carrying nothing but the URL, which the switch turns
off for whoever minds; (c) costs most of the icons for a privacy gain only against the hosts a
chart's author chose. **Built 2026-10-14** (night build HELM2, G11;
[`helm/icon.go`](../../internal/helm/icon.go), [`engine/helmicons.go`](../../internal/engine/helmicons.go),
[`fyneui/helmcharticons.go`](../../internal/ui/fyneui/helmcharticons.go)): the Charts view's first
column and a chart drawer's header draw the icon, Cogpit's own helm-wheel glyph while it loads, when
there is none, when it failed and while the switch is off; `TestFetchIcon`,
`TestFetchIconRefusesPrivateAddresses`, `TestChartIcons`, `TestChartIconRefusals`,
`TestChartIconsPreference`, `TestChartIconsInTheWindow`, `TestPreferencesLoadChartIcons`, PNGs
[dark](../../internal/ui/fyneui/testdata/helm_chart_icons_dark.png),
[light](../../internal/ui/fyneui/testdata/helm_chart_icons_light.png).

What the build chose, Cogpit's defaults (not the owner's decision):

- **When:** only for a cell or the drawer the window draws, at most 4 downloads at once, one per URL;
  never for agents (no command fetches an icon; `object_get` and the rows carry the URL), never for
  the CLI.
- **How:** https only, a redirect too (at most 5); no credentials (not the repository's, whatever
  `pass_credentials_all` says), no cookies, no referrer, the user agent "Cogpit"; at most 256 KiB
  and 20 seconds; through the Preferences' proxy for non-cluster traffic, as every download.
- **Where to (beyond the option):** a host that resolves to a loopback, private, link-local,
  carrier-grade NAT or otherwise non-public address is refused, so that an index cannot make Cogpit
  send requests into the user's own network; a chart repository's own host (any repository in
  Helm's file) is allowed at any address, so an internal repository's icons beside its charts show.
  Each request dials the address it checked. Through a proxy the proxy resolves the host and the
  check does not apply ([H-87](../security/trust-boundaries.md#h-87-chart-icons-tell-their-hosts-what-the-user-looks-at)).
- **What:** checked as an uploaded cluster icon is (`CheckIcon`): a PNG or JPEG decoded under bounds
  (16 megapixels, 8,192 a side) and kept as a PNG of at most 256 pixels a side; an SVG kept only
  when it is one `<svg>` document without a DOCTYPE, scripts, event handlers or references outside
  itself; anything else (GIF, WebP, ICO, HTML) refused; the window draws an SVG from a PNG of 128
  pixels a side that it rasterises off the main goroutine under a recover, never through Fyne's
  renderer (2026-10-14)
  ([H-88](../security/trust-boundaries.md#h-88-chart-icons-are-decoded-in-cogpits-process)).
- **Kept:** `<user cache dir>/Cogpit/helm/icons/<hash of the URL>.png` or `.svg`, owner-only, used for
  7 days, the last 2,048; in memory the last 512; a failure remembered for an hour.
- **The switch:** `Preferences.LoadChartIcons`, on by default; off, nothing is fetched or read and the
  glyph shows everywhere at once. Agents and the CLI may only turn it off (`ErrChartIconsGUIOnly`):
  turning it on decides whether Cogpit contacts the icons' hosts.

## D-47: Deep links (`cogpit://`)

**Open: the lead's decision of 2026-10-14, for the owner to confirm; built 2026-10-14.** Lens
Desktop opens `lens://` links (the row in [ui-parity](../parity/ui-parity.md#appearance-and-application)
since 2026-10-11, from Lens's documentation; not captured, so what a Lens link can name is not
verified). Cogpit's app bundle registers the scheme `cogpit` (`CFBundleURLTypes`, the
[`Makefile`](../../Makefile)'s `app` target), and macOS hands Cogpit every `cogpit://` link that a
web page, a mail, a chat or another app opens, whether Cogpit runs or not. A link is therefore
untrusted input: its author is anyone. Options for what a link may do:

- (a) **navigation only**: connect a context of the user's kubeconfig sources, open a view, set the
  namespace filter, open an object's drawer, through the `ui.*` commands agents run freely (D-2);
- (b) (a) plus commands with a confirmation, as an agent's are confirmed (a link that asks the user
  to restart a Deployment);
- (c) no deep links.

**Chosen (a)**, because it gives what links are for (a runbook, a chat message or an alert pointing
at an object) at the cost of nothing the user's identity can write: (b) turns every web page into a
prompt the user must read with care before a click acts on a cluster with their identity, the
consent-phishing pattern; (c) loses Lens's links. Built ([`engine/deeplink.go`](../../internal/engine/deeplink.go),
[`cmd/cogpit/deeplink_darwin.m`](../../cmd/cogpit/deeplink_darwin.m),
[`fyneui/deeplink.go`](../../internal/ui/fyneui/deeplink.go); the grammar in
[usage](usage.md#deep-links-cogpit); the boundary in
[trust boundaries](../security/trust-boundaries.md#boundaries-and-required-checks)). What the build
chose, Cogpit's defaults (not the owner's decisions):

- **The grammar:** `cogpit://open?context=<context id>&view=<view key>&namespace=<namespace>&name=<name>`,
  every parameter optional but one; read strictly (the scheme, the action `open`, no user, port,
  path or fragment, each parameter once, not empty, without control or format characters, a view
  key's, a DNS label's or an object name's form, at most 2,048 bytes in all); anything else is
  refused whole, never half followed (`TestParseDeepLink`, `TestParseDeepLinkRefusals`,
  `FuzzParseDeepLink`).
- **Which contexts:** one of the kubeconfig sources' contexts, by its id or by a kubeconfig context
  name one file alone defines, or the demo cluster; never one a connector without sources offers
  (it would run the kubeconfig's credential plugins unasked), never a file path.
- **A login that waits for the user's approval stops the link:** a context whose credential plugin
  or token file the user has not allowed, or that changed since, is not connected, and the toast
  says to connect from the context picker, where the user reviews it. Weighed: showing the approval
  dialog from the link (the brief allowed either); not chosen, because a dialog that a web page
  raises at a moment of its choosing invites allowing a program the user did not ask for. An
  approved plugin runs, as when the user connects.
- **A link may switch the connected cluster without asking**, as the hotbar does; Cogpit holds one
  connection, the other cluster's tabs stay, and a write the user confirmed for the other cluster
  refuses to run (`CheckConfirmedContext`). A settings page that is open closes, or it would hide
  what the link opened ([H-89](../security/trust-boundaries.md#h-89-a-link-switches-the-cluster-and-the-view-under-the-user)).
- **Origin `deeplink`**, neither the user in the window nor an agent. `Dispatch` refuses it every
  command but the five navigation commands (`DeepLinkCommands`: `settings.close`,
  `cluster.connect`, `ui.setNamespace`, `ui.openView`, `ui.navigate`), and any of those that is not
  a UI or read command, may escalate, is GUI-only or declares a verb other than get, list and
  watch, before its parameters are decoded; and once more after `Escalate`, any invocation that is
  not UI or read. So a write is unreachable by construction, whatever a later change to
  `OpenDeepLink` dispatches (`TestDeepLinkCommandsOnlyNavigate` over the whole registry,
  `TestDeepLinkGuardsAnAlteredCommand`). `NeedsConfirmation` is not changed: it never sees a link's
  write, because none gets that far. Every `Refuse` and GUI-only mark that admits only the GUI
  treats a link as not the user.
- **What the user sees:** the window comes to the front for every link, and a toast says what it
  opened ("Opened from a link: Pods › team-a/web-0 on kind-dev [config]") or why it did not; both
  are in the notification log. The audit log gets one line per link, `deeplink.open` with origin
  `deeplink`, the link's parameters (or, for a link that does not parse, at most 256 bytes of it,
  escaped) and the outcome ([H-90](../security/trust-boundaries.md#h-90-a-links-text-reaches-agents-through-the-logs)).
- **Delivery:** the Apple event handler (`kAEGetURL`) is installed before the app finishes
  launching and again when AppKit is about to, independent of GLFW's app delegate; links wait in a
  queue (at most 8; one longer than 2,048 bytes dropped) until the window is up, then open one after
  the other (`TestDeepLinkAppleEvents` in a process of its own, `TestDeepLinkQueue`,
  `TestDeepLinksOpenInTheWindow`).
- **Not on Windows and Linux:** there are no packages to register the scheme in yet (G22).

## D-48: Start at login

**Open: the lead's decision of 2026-10-14, for the owner to confirm; built 2026-10-14.** Lens
Desktop has "Automatic start-up" among its application preferences (inferred from its
[preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/) documentation, not captured;
its default not verified). Options:

- (a) **the macOS app's own login item** (`SMAppService.mainAppService`, macOS 13 and later,
  Cogpit's minimum), which System Settings › General › Login Items lists and the user can switch
  there too; Cogpit keeps no copy and shows what macOS reports;
- (b) a LaunchAgent property list Cogpit writes into `~/Library/LaunchAgents`: works without a
  signature, but is a file outside Cogpit's own folders that System Settings shows as a background
  item, and Apple's way since macOS 13 is (a);
- (c) a preference kept in `preferences.json` that Cogpit applies again at every start: it would
  undo the user's switch in System Settings.

**Chosen (a)**, **off by default**, and **only the user turns it on** (agents and the CLI may turn it
off, as with chart icons, D-46), because it makes Cogpit run after every login without being
opened, its MCP server included unless the user switched that off. Built
([`engine/loginitem.go`](../../internal/engine/loginitem.go),
[`commands/loginitem.go`](../../internal/engine/commands/loginitem.go),
[`cmd/cogpit/loginitem_darwin.m`](../../cmd/cogpit/loginitem_darwin.m),
[`fyneui/preferences_loginitem.go`](../../internal/ui/fyneui/preferences_loginitem.go)): Preferences ›
App › "Start Cogpit at login", a check with a line saying what macOS reports (off, on, waiting for
the user's approval in System Settings with a button to Login Items, not found), read when the
section opens and after every change; `preferences.loginItem`, `preferences.setStartAtLogin`
(turning it on refused for every origin but the GUI) and `preferences.openLoginItems` (GUI-only);
`UIState.LoginItem`. Cogpit's defaults: the row's place (App) and wording; a Cogpit that does not
run from its app bundle (`bin/cogpit`, `go run`) reports not found and refuses to register without
asking macOS.

**What needs the signed app.** "Apps that use SMAppService APIs must be code signed"
(ServiceManagement's `SMAppService.h`). The release's app is ad-hoc signed today; the Developer ID
signature is being built (D-4). What macOS does with the ad-hoc signed app is **not verified**: it
may register it, report it not found, or refuse with an error, which the row shows under the check.
Nothing was registered on a development machine to find out; a signed app installed in
Applications is the check. **Not offered** on Windows and Linux (a Run key, an XDG autostart file
come with their packages, G22), rather than a switch that does nothing.

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-14 | minor | D-44 decided by the owner (H-82, option b): an agent's read that the filter reads takes the Secrets as the Secrets view reads them and waits for them; built (`engine/agentsecrets.go`) |
| 2026-10-14 | minor | D-4 built, the third step (agent SPARKLE): Sparkle 2.10.0 in `Cogpit.app`, pinned by version and SHA-256, loaded at run time (`internal/update`), `CFBundleVersion` the release's version; Preferences › App › Check for updates automatically and Update channel, Help › Check for Updates…; Sparkle's parts signed inside-out; the appcast drafted, signed (Sparkle 2.9's signed feeds) in the environment `sparkle` and deployed to Pages, in three jobs; no update key yet. For the owner to confirm: the signed feed, the preference as the source of truth, agents may turn the checks on but not off, any origin checks, Beta refused until prereleases, no XPC services and no entitlements, the three jobs, the history from the signed feed on Pages, the embedded release notes, the menu item's place and label |
| 2026-10-14 | minor | D-47 (the lead's decision, for the owner to confirm): `cogpit://open?context=&view=&namespace=&name=` links navigate only, through five `ui.*` commands with origin `deeplink`, which `Dispatch` refuses every other command and any write; a login waiting for approval stops the link; built, with Cogpit's defaults listed |
| 2026-10-14 | minor | D-48 (the lead's decision, for the owner to confirm): start at login as the macOS app's own login item (SMAppService), read from macOS, off by default, turned on by the user only; built; what the ad-hoc signed app does not verified |
| 2026-10-14 | minor | D-44 (owner, H-82): the values of the cached Secrets replaced by `«secret <ns>/<name>/<key>»` in what agents read; built, minimum 8 bytes; the base64 form, the stand-alone rule under 16 bytes, name keys left out, terminal tabs and every agent result, only cached Secrets and the refusal of placeholders in agents' texts the lead's decisions for the owner to confirm. H-63 (owner): Flux's credential fields withheld from the YAML agents read, built |
| 2026-10-14 | minor | D-29 (owner): a repository's credentials over plain http asked once per repository, bound to its name, URL and user, then used as helm uses it; nothing downloaded before (H-40 narrowed); built, the file, the binding without the password, the token and the banner Cogpit's defaults |
| 2026-10-14 | minor | D-37 (owner, H-71): every node shell pod recorded in `nodeshells.json` before its create, and the pods a crash left deleted by UID at the next connect to their context, with a notification; built, the details Cogpit's defaults |
| 2026-10-14 | minor | D-44 (owner, H-83): a terminal tab agents may read carries a marker that flashes at each agent's read, and its switch turns itself off after 15 minutes without one, which the strip then says; built, the details Cogpit's defaults |
| 2026-10-14 | minor | D-2 (owner): a role binding an agent creates or changes always waits for the user, also where agent writes are allowed: `roleBinding.create`, and `object.create` and `object.apply` when the text or the Edit tab holds a binding (a command's declared `AgentsAsk`); built. Helm charts and the role's own rules are not covered (H-27) |
| 2026-10-14 | minor | D-4 decided by the owner: Sparkle's feed on GitHub Pages (`https://guided-traffic.github.io/cogpit/appcast.xml`), signed in the environment `sparkle` with the owner as required reviewer |
| 2026-10-14 | minor | D-4 built, the second step: the release pipeline signs the app and `cogpit-cli` for macOS with the Developer ID (hardened runtime, no entitlements), ships the app in `Cogpit-<v>-macos-arm64.dmg`, notarizes both and staples the disk image's ticket, in a job of its own; no tag run yet. The lead's decision for the owner to confirm: a failure fails that job and the release gets no files until it is re-run. Agent REL's, for the owner to confirm: signing in its own job, the disk image instead of the zip, `cogpit-cli` notarized but not stapled, only the disk image submitted |
| 2026-10-14 | minor | The night decisions' bodies brought in line with the owner's answers of 2026-10-14: D-34's state reading and page, D-41's five and D-42's choice marked decided as built; D-35, D-36, D-43, D-44, D-46 confirmed; D-33's two rules (Reconcile with Source does not wait for the source; the whole chain checked before the first patch) and D-34's two filter rules (a row's states, several state columns) marked Cogpit's defaults, standing until the owner asks |
| 2026-10-14 | minor | D-4: the owner's Apple Developer ID certificate and a team API key for notarization set up in the environment `release`; the pipeline's signing steps not built yet |
| 2026-10-14 | minor | D-40 decided by the owner: option (b), the workloads' drawers' Pods tables without a Namespace column (the node drawer keeps it); built, PNGs re-rendered |
| 2026-10-14 | minor | D-39 decided by the owner: option (b), and a refusal of one namespace's pod metrics affects that namespace only (empty cells and the state forbidden there, the other namespaces' values and the Metrics Server's status kept, no Prometheus in its place); built, with Cogpit's defaults listed (a drawer too, the refusal kept for the TTL, `metrics.get`'s `forbidden`) |
| 2026-10-14 | minor | D-33 decided by the owner: Reconcile, Suspend and Resume as built, plus Reconcile with Source (the source's reconcile request first, then the object's; a Helm release's chart source and HelmChart) and Reset (resetAt with requestedAt's token) as the flux CLI does them, mutating, on selections too; Force stays an agent's parameter; built. Night decisions for the owner to confirm: Reconcile with Source does not wait for the source, and reads every object of the chain before the first patch |
| 2026-10-14 | minor | D-34 decided by the owner: a dashboard's state opens its kind's view filtered, through a status filter every view has (option a), the Argo CD Dashboard's tiles too; built, with two decisions for the owner to confirm (a row's states: the kind's state columns, tokens without one; several columns qualified and each a condition) and Cogpit's defaults listed; D-41: the tab's status filter kept |
| 2026-10-14 | minor | D-37 and D-38 decided by the owner (option b each): the default image the newest supported Alpine release, 3.24 pinned by its index digest and kept current by Renovate (no longer Lens's 3.19), the deadline a setting, Maximum duration, default 2 hours, set by the user only; built, the bounds (5 minutes to 24 hours) and the automerge of Renovate's updates Cogpit's defaults |
| 2026-10-14 | minor | D-35, D-36, D-41, D-42, D-43, D-44, D-46 decided by the owner in the morning's decision round, each as built (D-33, D-34, D-37 to D-40 follow with their builds) |
| 2026-10-14 | minor | D-43 (night decision, for the owner to confirm): the Nodes list's Disk and a node drawer's disk from node-exporter's root filesystem through Prometheus, never the kubelet's nodes/proxy; one query per 60 s poll, used as size less available, nodes matched by node label, instance or address, the Nodes list looking for Prometheus while the column shows, Cogpit's defaults; built |
| 2026-10-14 | minor | D-46 (night decision, for the owner to confirm): chart icons fetched from the hosts that serve them when the window shows them, with Preferences › Load chart icons on by default (option b); a repository's own host allowed at a private address, other private addresses refused; built |
| 2026-10-14 | minor | D-11, D-37 (the night build's seams): a dock tab whose context leaves Cogpit closes with it; an editor's draft is kept, a node shell's pod deleted (Cogpit's defaults) |
| 2026-10-13 | minor | D-44 (night decision, for the owner to confirm): agents read what the dock's logs tabs show freely, a terminal tab's screen only once the user turned on the tab's switch (off when a tab opens, GUI-only); built with Cogpit's defaults (every terminal type, no dialog, the bounds, no scrollback in the widget) |
| 2026-10-13 | minor | D-42 (night decision, for the owner to confirm): no context picker at start, the welcome page with its cluster rows; `--context` and `--demo` as before; the engine's notices from before the window shown as toasts once it is up; built |
| 2026-10-13 | minor | D-41 (night decision, for the owner to confirm): the tab bar's tabs and the tables' hidden columns and column order kept in `workspace.json` and restored at start without connecting anything, the tab shown last marked, the tabs of contexts that are gone dropped with a notice; column widths not kept (a question for the owner); built |
| 2026-10-13 | minor | D-40 (night decision, for the owner to confirm): the node drawer's Pods table in the drawers of Deployments, Replica Sets, Stateful Sets, Daemon Sets, Jobs and Cron Jobs, from indexes only the open drawer holds; Cron Jobs, the eight columns, every replica set's pods, the owner's UID in its namespace, the 500-row bound and the order of `related: true` Cogpit's defaults; built |
| 2026-10-13 | minor | D-39 (night decision, for the owner to confirm): pod metrics listed per namespace of the Pods view's plan for a user who may not list pods everywhere, built |
| 2026-10-13 | minor | D-36 (night decision, for the owner to confirm): Argo CD's Sync writes the operation `argocd app sync` writes without flags, bound to the Application's UID and resourceVersion, refused while an operation is requested or running |
| 2026-10-13 | minor | D-35 (night decision, for the owner to confirm): Argo CD's repository and cluster Secrets show an allowlist of non-credential keys to the window and agents, user information in URLs redacted, the other keys by name only |
| 2026-10-13 | minor | D-33 and D-34 (night decisions, for the owner to confirm): the Flux actions (Reconcile, Suspend, Resume, as the flux CLI patches them, mutating, on selections too) and the Flux dashboard (the states after kstatus, counts per kind, the objects that are not ready); built (G23) |
| 2026-10-13 | minor | D-37 and D-38 (night decisions, for the owner to confirm): the node shell as Lens's privileged pod in kube-system with nsenter, its 2-hour deadline, the pinned Alpine 3.19 default and the other defaults of the build; its image and pull secret set by the user only; both built |
| 2026-10-13 | minor | D-19: Create Secret built (night build, G9): `secret.create` refuses agents and the CLI before anything is asked or sent; its values go base64-encoded in `data` (Cogpit's default) |
| 2026-10-13 | minor | D-4 (owner): the environment `release` for the Apple credentials created; Sparkle's EdDSA key in an environment with the owner as required reviewer |
| 2026-10-13 | minor | D-16 (owner): dependency updates committed as `build(deps)`, no release of their own |
| 2026-10-13 | minor | D-16 revised (owner): releases live: semantic-release tags and releases after every gate on `main` with the GitHub App's token, the release pipeline attaches `cogpit-cli` and the unsigned app; a breaking change makes a minor release under v0. D-4: what is built of it (the unsigned app zip and the CLI archives; no signature, notarization, disk image, cask or Sparkle yet) |
| 2026-10-13 | minor | D-32 (owner): every byte amount in binary units with one decimal ("15.6 GiB"); built 2026-10-13 with one formatter in `model` (`FormatBytes`, `FormatByteQuantity`, `FormatResource`): lists, drawers, charts, data sizes, the kubectl download and the messages that state a size; the space before the unit, the decimal point whatever the region, milli-bytes as written, a size beside its limit not named and the places listed Cogpit's defaults; ten places not converted named for the owner |
| 2026-10-13 | minor | D-31 (owner): every timestamp in the local time zone and the region's format ("10.09.2026 17:41:01"); built 2026-10-13 from macOS's region (CoreFoundation), else the environment's locale and a CLDR table, one formatter in the engine; the format as a setting not built; the conversion, Windows reading the environment, logs with milliseconds and the list of places Cogpit's defaults |
| 2026-10-13 | minor | D-25 (owner, in a real window): the drawer's copy icon right after every kind of value, a status pill's included (it sat at the drawer's right edge), a long line shortened before it, a value with action buttons owning the room for it; built |
| 2026-10-13 | minor | D-30, the owner's answers: agents and the built-in agent's LLM endpoint read the decoded certificates, never a key (H-45 closed by the decision, removed from agents-and-mcp.md); `tls.crt` checked against `tls.key` (the key parsed in memory, the outcome yes, no or a reason) and DER certificates decoded, both built; the pairing rule, the chain check, the 64 KiB key bound and the names that count as private keys Cogpit's defaults |
| 2026-10-13 | minor | D-30 (owner): certificates in Secrets decoded into separate fields in the drawer and for agents (`object_get`'s `certificates`), an exception to "Secret values never reach an agent"; built 2026-10-13, which keys, the bounds, the fields, the thresholds and the layout Cogpit's defaults; no chain or key-pair check |
| 2026-10-13 | minor | D-25 (owner, after seeing it): the lists' copy icon not on every cell but outside a blacklist of columns (by type: numbers, ages, times, metrics, warnings; by id: state words, booleans, counts and ratios in text, policy words: Cogpit's reading of the owner's principle), and right after the text instead of at the column's end; built |
| 2026-10-13 | minor | D-29 built: Helm's repositories.yaml read every 2 s (the helm CLI's resolution, one place), credentials in memory only and to the repository's own host, indexes in Cogpit's own cache keyed by name and URL (decided at the build), `helm.repo.add` and `helm.repo.remove` removed; Cogpit's defaults listed |
| 2026-10-13 | minor | D-25: copy icons in the lists built (one icon per list on the text cell under the pointer, the drawer's checkmark and toast); every text cell, the icon inside the cell's right end with the text shortened before it, a click that never selects, and no values in a Secret's list are Cogpit's defaults |
| 2026-10-13 | minor | D-29 (owner): repositories are added with the helm CLI, Cogpit only reads Helm's file and writes nothing into it |
| 2026-10-13 | minor | D-29 (owner): Helm chart repositories read from Helm's repositories.yaml, changes within 5 seconds, credentials included; D-25 (owner): names in the lists copyable with the drawer's copy icon; neither built |
| 2026-10-13 | minor | D-11: drafts of unsaved editor text built (night run; the delays, the offer on connect and closing as a discard are that run's defaults, not decided); D-19: no draft of a Secret's text |
| 2026-10-13 | minor | D-3 built (night run): the version check when a terminal opens, the GUI-only verified download (SHA-256 and sigstore signature, no transparency log: H-35), the release first on the terminal's PATH (zsh after its profile; H-36), the Preferences; the differences listed under D-3 |
| 2026-10-13 | minor | D-14: the right mouse button's action and ⌃Ins and ⇧Ins on the clipboard, patches of the fork and upstream candidates (night run), not offered |
| 2026-10-13 | minor | D-22 amended (Cogpit's default): the theme is a preference too, `--theme` overrides it for a run; the clusters' icon colours are Solarized's accents |
| 2026-10-13 | minor | D-11: Helm install built on the journal's road as the lead's default (an interrupted install offered as an uninstall); the decision stays the owner's |
| 2026-10-12 | minor | D-11: crash reports kept locally (built), and what they found first |
| 2026-10-12 | minor | D-14: the upstream pull requests opened (#155–#158, owner's OK) |
| 2026-10-12 | minor | D-28 (owner, built, scope C4): one approval dialog per file and identical plugin, listing every context with its server and trust; the approvals stay per context with their own fingerprints; approveExec takes the list and refuses all if one changed |
| 2026-10-12 | minor | D-28 built (scope C3): every context from a source file read alone, `~/.kube/config` a file of the default folder, `$KUBECONFIG` and `--kubeconfig` files session sources, bare names resolved; the default kubeconfig no longer merged |
| 2026-10-12 | minor | D-28 (owner): the default kubeconfig as sources, as Lens (config an ordinary synced file, `$KUBECONFIG`'s files sources of their own); C1's Rollback on the current revision does what helm does (owner) |
| 2026-10-12 | minor | D-25 (owner, built): a copied Secret value marked concealed for clipboard managers and kept from Universal Clipboard on macOS (H-25) |
| 2026-10-12 | minor | D-26 (owner): a credential plugin's approval binds the program's content when it lies in the kubeconfig's or sync folder; D-27 (owner, built): an agent's inline Helm values shown in the confirmation |
| 2026-10-12 | minor | D-22 to D-25 built: Solarized theme tokens (contrast departures listed), the status bar link, the C helm logo (`internal/ui/logo`), copying values from the drawer |
| 2026-10-12 | minor | D-19 built: Secrets edited and created from YAML in the GUI only; agents and the CLI refused in the engine, values scrubbed from errors |
| 2026-10-12 | minor | D-17 built: a token file outside an added kubeconfig's folder is asked for in the plugin dialog; the approval binds the content of a program or CA file inside the folder (R-5, D-26) and the dialog shows proxy, insecure and own CA (R-11) |
| 2026-10-12 | minor | Owner's decisions on look and use: D-22 Solarized colours, D-23 the status bar link, D-24 the C helm logo, D-25 copying values |
| 2026-10-12 | minor | D-21 (owner): no adding of kubeconfigs from pasted text; the paste feature built the same day is removed (pasted sources of earlier builds are no longer listed, their files stay) |
| 2026-10-12 | minor | D-20 (owner): a port forward fails at once on permanent errors (option a); built the same day |
| 2026-10-12 | minor | D-14 (owner): the upstream pull requests grouped, opened by the lead after the owner's OK on the series |
| 2026-10-12 | minor | D-19 (owner): Secrets editable and creatable from YAML in the GUI only (option a), not built |
| 2026-10-12 | minor | D-18 (owner): Secret annotations stay visible to agents; H-23 documented and accepted (option d) |
| 2026-10-12 | minor | D-17 (owner): token files of added kubeconfigs outside their folder are asked for like credential plugins (H-24, option a) |
| 2026-10-12 | minor | D-11: Helm upgrade decided and built (journal, offered as a rollback), journals locked between processes |
| 2026-10-12 | minor | D-14 built: the two panics patched, the widget fuzzed alone, further crashes, hangs and races fixed; the upstream series prepared, not sent |
| 2026-10-12 | minor | D-13 as revised built (fourth run): the Secrets view on a full watch, the drawer from the cache, Helm's releases through covering scopes |
| 2026-10-12 | minor | D-15 built (`make parity`); D-14: the copy is on upstream's head, whose code builds on Fyne 2.8.1 |
| 2026-10-12 | minor | Owner's decisions after the third run: D-13 revised to a full watch on Secrets (option a); D-14 terminal widget (keep the fork, upstream, fuzz); D-15 the parity checker before the generators; D-16 dry runs until the alpha's release |
| 2026-10-11 | minor | D-11: Helm rollback and uninstall are crash-safe through a journal and an offer on the next connect (third run); D-13: the Secrets view built, the Type column found empty too, the Table format open for the owner |
| 2026-10-11 | minor | D-13: Secrets in the cache decided (owner): metadata only for the list, a GET for an opened Secret, a label-selected watch for Helm's releases (H-8, option b) |
| 2026-10-11 | minor | D-1 decided (owner): Fyne after the spike gate, with re-checks of S7 at M0's CI and S5 at M11, each reopening D-1 only for its platform |
| 2026-10-11 | minor | D-1: the spike ran as the local MVP; results recorded, gate decision pending |
| 2026-10-11 | minor | D-7: macOS first, Windows and Linux after the macOS beta; Cogpit is a desktop client, never a web UI |
| 2026-10-11 | minor | D-5 decided (owner): extensions are built into the code; no external extension mechanism for now |
| 2026-10-11 | minor | D-9 decided (owner): Flux CD and Argo CD views in the beta, other paid features after it, vendor features never |
| 2026-10-11 | minor | D-4 for macOS decided (owner): the app in a signed, notarized disk image as the standard install, a Homebrew cask from the same image, Sparkle 2 for updates |
| 2026-10-11 | minor | D-4: macOS release artifact is the zipped app bundle, no dmg (owner); Sparkle 2 recommended as updater |
| 2026-10-11 | minor | D-3: `kubectl` from `PATH`, verified matching download on one click (owner) |
| 2026-10-11 | minor | D-13: cache held in memory only, not persisted across restarts (owner) |
| 2026-10-11 | minor | D-13 cache: decided requirement, open persistence question |
| 2026-10-11 | major | Go instead of Rust (D-7); Fyne behind a spike gate (D-1); client-go, kubectl libraries, Helm SDK (D-6, D-3); MCP control of UI and cluster (D-2); built-in agent with an OpenAI-compatible endpoint (D-10); one-click updates (D-4); crash stance (D-11); parity sources (D-12) |
| 2026-10-11 | major | First version: D-1 to D-9 |
