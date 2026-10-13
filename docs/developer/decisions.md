# Decisions

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-13 · **Change:** minor (D-32 (owner): every byte amount in binary units with one decimal, built with one formatter, the places not converted named for the owner; D-31 (owner): every timestamp in the local time zone and the region's format, built from the operating system's region, the format as a setting not built; D-25 (owner, in a real window): the drawer's copy icon right after every kind of value, a pill's included, built; D-30, the owner's answers: agents read certificates (H-45 closed), the key check and DER certificates built; D-30 (owner): certificates in Secrets decoded in the drawer and for agents, built with Cogpit's defaults; D-25 (owner, after seeing it): the lists' copy icon only outside a blacklist of columns, right after the text, built; D-29 built: Helm's repositories.yaml read every 2 s, credentials in memory only, indexes in Cogpit's own cache, add and remove removed; D-25: copy icons in the lists built, with Cogpit's defaults; D-29 (owner): Helm's repositories.yaml as the repositories' source; D-25 (owner): copy icons in the lists; D-11: drafts of unsaved editor text built with the night run's defaults, never a Secret's or Helm values (D-19); D-3 built 2026-10-13, its defaults listed; D-14: two terminal input patches added to the fork as upstream candidates, not offered; D-22 amended: the theme a preference, `--theme` an override for the run, the icon colours; D-11: Helm install built on the journal as the lead's default, still open; D-11: crash reports kept locally, built; D-26 and D-27 decided, D-27 built; D-21 built: the paste feature removed; D-13 revised to a full watch on Secrets; D-14, D-15, D-16 decided) · **Target:** Lens Desktop 2026.10.51501

Decisions that shape the code, each with its options, the reasons, and its status. A decision
moves from **open** to **decided** only by the owner; a recommendation is a proposal, not an
outcome. Ecosystem facts were observed on 2026-10-11 from the Go module proxy, crates.io and the
projects' repositories; re-check them before relying on a version number.

| Id | Decision | Status |
|---|---|---|
| [D-1](#d-1-ui-toolkit) | UI toolkit | decided: Fyne, after the spike gate; re-checks of S7 (CI, M0) and S5 (Windows and Linux, M11) can reopen it per platform |
| [D-2](#d-2-agent-interface-scope) | What agents may do over MCP | decided: everything, the UI included; destructive actions confirmed by the user |
| [D-3](#d-3-helm-and-kubectl-binaries) | Where `helm` and `kubectl` come from | decided: Helm in-process; `kubectl` from `PATH`, a verified matching download on one click |
| [D-4](#d-4-packaging-and-updates) | Packaging, signing and updates | decided for macOS: the app in a DMG, Homebrew cask, Sparkle updates; Windows and Linux with M11 |
| [D-5](#d-5-extensions) | Extensions | decided: built into Cogpit's own code; no external extensions for now |
| [D-6](#d-6-kubernetes-libraries) | Kubernetes libraries | decided: client-go, kubectl libraries, Helm SDK |
| [D-7](#d-7-language-and-platforms) | Language and platforms | decided: Go; a macOS app first, Windows and Linux later |
| [D-8](#d-8-licence) | Licence | decided: Apache-2.0 |
| [D-9](#d-9-scope-against-lenss-paid-and-vendor-features) | Scope against Lens's paid and vendor features | decided: Flux CD and Argo CD in the beta, other paid features after it, vendor features never |
| [D-10](#d-10-built-in-agent) | Built-in agent and LLM interface | decided: chat beside the views, OpenAI-compatible endpoint; built after the alpha |
| [D-11](#d-11-crash-stance) | Crash stance | decided: crash-tolerant UI, crash-safe cluster operations |
| [D-12](#d-12-parity-sources) | Parity sources | decided: current Lens Desktop only; no OpenLens or Freelens |
| [D-13](#d-13-cache) | Cache | decided: the UI reads only from a watch-fed cache, held in memory only; Secrets in full (revised 2026-10-12, built), Helm's releases from that watch or their own `owner=helm` watch |

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
**Human:** Apple Developer ID and Windows code-signing material.

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
  context offers it (Restore, Discard; `editor.drafts`). Never a Secret's text and never Helm
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

**Decided (owner, 2026-10-12):** semantic-release stays a dry run until the alpha. The alpha's
release is the first real one, cut together with the job that builds the unsigned macOS app as a
download, so that the first version (0.1.0) has something to install; that job is part of the
alpha's work ([releasing](releasing.md#the-first-real-release),
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
draft on disk ([`drafts.go`](../../internal/engine/drafts.go), `TestNoDraftsForSecretsOrHelmValues`). The options weighed:

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
  - plain `http://` repositories are listed and used as helm uses them, credentials included, and
    marked "http" in the dialog; a redirect never goes from https to http, and an https
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
([`model/bytes.go`](../../internal/model/bytes.go); [architecture › Current state](architecture.md#current-state-2026-10-13)):
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

## Revision history

| Date | Change | Summary |
|---|---|---|
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
