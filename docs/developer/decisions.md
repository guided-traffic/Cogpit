# Decisions

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-13 · **Change:** minor (D-29 (owner): Helm's repositories.yaml as the repositories' source; D-25 (owner): copy icons in the lists; D-11: drafts of unsaved editor text built with the night run's defaults, never a Secret's or Helm values (D-19); D-3 built 2026-10-13, its defaults listed; D-14: two terminal input patches added to the fork as upstream candidates, not offered; D-22 amended: the theme a preference, `--theme` an override for the run, the icon colours; D-11: Helm install built on the journal as the lead's default, still open; D-11: crash reports kept locally, built; D-26 and D-27 decided, D-27 built; D-21 built: the paste feature removed; D-13 revised to a full watch on Secrets; D-14, D-15, D-16 decided) · **Target:** Lens Desktop 2026.10.51501

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
the same checkmark after a copy. **Not built yet.** Proposals for the build (not decided): every
text cell gets it, as every drawer value does; the icon sits at the cell's right end inside the
cell, so it never covers the next column; a click on the icon copies and does not select the row;
a Secret's cells hold no values, so nothing secret can be copied from a list.

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
a list of Cogpit's own, public https repositories only, Helm's file never read). **Not built yet.**

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

## Revision history

| Date | Change | Summary |
|---|---|---|
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
