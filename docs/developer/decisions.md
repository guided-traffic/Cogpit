# Decisions

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (language switched to Go; Fyne behind a spike gate; MCP control, built-in agent, one-click updates, crash stance and parity sources decided by the owner) · **Target:** Lens Desktop 2026.10.51501

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
| [D-13](#d-13-cache) | Cache | decided: the UI reads only from a watch-fed cache, held in memory only |

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
  operation, is designed with the operation.
- **Unsaved editor content** is kept in a local draft so that a crash does not lose it.

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

## Revision history

| Date | Change | Summary |
|---|---|---|
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
