# Platform parity: systems, packages, authentication and tools

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** minor (the MVP: five rows partial) · **Target:** Lens Desktop 2026.10.51501

Where and how Cogpit runs compared with Lens Desktop 2026.10.51501: operating systems and
architectures, packages and updates, how it authenticates to clusters, which Kubernetes versions
it handles, and the external tools it depends on. Method: [`README.md`](README.md).

**Sources.** Lens Desktop: [install](https://docs.lenshq.io/k8slens/getting-started/install-lens/),
[technical FAQ](https://docs.lenshq.io/faq/technical-faq/),
[preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/), its update feeds
(`latest.yml`, `latest-mac.yml`, `latest-linux.yml` on `downloads.k8slens.dev`) and the
2026.10.51501 release notes. Cogpit's ecosystem facts: [decisions](../developer/decisions.md).

**Summary (measured, 2026-10-11):** 5 of 38 rows partial, none done (four more are not targets or
have nothing to build): the macOS app bundle (1 of 13 package rows), client certificates and bearer
tokens (2 of 13 authentication rows), Kubernetes 1.36 on kind, and `kubectl` from `PATH`. Cogpit ships for macOS first; Windows and Linux follow after the macOS beta
([D-7](../developer/decisions.md#d-7-language-and-platforms)). Remaining (estimated): 25–50 h for
the macOS app, signing and one-click updates ([G14](gaps.md#g14-packaging-signing-and-updates-macos)),
40–80 h for Windows and Linux ([G22](gaps.md#g22-windows-and-linux)) and 25–50 h for authentication
([G2](gaps.md#g2-cluster-connection-and-authentication)), both with work a human must do: signing
material, and real cloud accounts to test credential plugins against.

## Operating systems and packages

| Platform | Lens Desktop | Cogpit target | Status |
|---|---|---|---|
| macOS | dmg for x64 and arm64, macOS 13 or later | the universal app bundle `Cogpit.app` (arm64 + x86_64) in a signed, notarized disk image to drag into Applications; macOS 13 or later ([D-4](../developer/decisions.md#d-4-packaging-and-updates)) | partial: `make app` builds an ad-hoc signed `Cogpit.app` for arm64 locally; not universal, no disk image, not notarized |
| Windows | setup exe (architecture not stated); silent install with `/S`, `/allusers`, `/D=`; Windows 10 22H2 or Server 2022 | after the macOS beta: MSI and portable zip for x64 and arm64, code-signed | missing |
| Linux | AppImage, deb, rpm for x86_64; APT and RPM repositories; Snap | after the macOS beta, Ubuntu first: AppImage, deb, rpm, tar.xz and Flatpak for x86_64 and aarch64 | missing |
| Homebrew cask (macOS) | mentioned, name not verified | yes, from the same disk image, with `auto_updates true` | missing |
| winget and Scoop (Windows) | – | with Windows | missing |
| Flathub (Linux) | – | with Linux | missing |
| APT and RPM repositories | yes | later | missing |
| Snap | yes | not planned | – |
| Update notice | yes | yes | missing |
| One-click update | Windows exe, macOS dmg and AppImage, with staged rollouts; APT and RPM manual | macOS through Sparkle 2; Windows and Linux formats with M11; package-manager installs hand over to the manager ([D-4](../developer/decisions.md#d-4-packaging-and-updates)) | missing |
| Update channels | Stable, Beta, Alpha | Stable and Beta | missing |
| Signed update manifest, verified before install | not documented | yes ([security › updates](../security/updates.md)) | missing |
| Checksums and SBOM per release | not documented | `SHA256SUMS.txt`, SPDX SBOM | missing |
| Activation or login before first use | Lens ID required (offline activation: Enterprise) | none ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | – (nothing to build) |
| Network endpoints the app needs besides clusters | `api.k8slens.dev`, `downloads.k8slens.dev`, `*.bored.k8slens.dev`, npm registry | none; Helm repositories, the LLM endpoint and the update check only when used | missing |

## Cluster authentication and connectivity

How kubeconfig credentials reach the API server ([security › credentials](../security/credentials.md)).
Lens Desktop connects through "a series of internal proxies" with a background HTTP proxy on a free
local port, and its FAQ therefore requires `localhost` to resolve to `127.0.0.1`
([technical FAQ](https://docs.lenshq.io/faq/technical-faq/)). Cogpit connects directly with
client-go from the process that holds the credentials, so that requirement does not apply.

| Method | Lens Desktop | Cogpit (client-go 0.37.1) | Status |
|---|---|---|---|
| Client certificate and key, inline or as files | yes | yes | partial: inline, against kind (`TestKindEngineWorkflows`, `TestKindIdleViewsCostNoLists`); files not tested |
| Bearer token, inline or token file | yes | yes | partial: inline, against a fake API server over HTTPS (`TestBudgetIdleViewAndWarmReopen`, [`cluster_test.go`](../../internal/cluster/cluster_test.go)); token files not tested |
| Exec credential plugin (`client.authentication.k8s.io`) | yes; the binary must be on `PATH` or absolute | yes, kubectl's own implementation | missing |
| Exec plugin: `aws eks get-token` and `aws-iam-authenticator` (EKS) | yes | to verify against a real cluster (human) | missing |
| Exec plugin: `gke-gcloud-auth-plugin` (GKE) | yes | to verify (human) | missing |
| Exec plugin: `kubelogin` (AKS, Entra ID) | yes | to verify (human) | missing |
| Exec plugin: `kubectl oidc-login` (generic OIDC) | not documented | to verify | missing |
| Exec plugins that open a browser or prompt (interactive mode) | not documented | to design ([H-6](../security/credentials.md#h-6-exec-plugins-inherit-the-environment)) | missing |
| Legacy `oidc` auth provider | not documented | yes; refreshed tokens through a persister Cogpit supplies ([H-4](../security/credentials.md#h-4-auth-provider-token-refresh)) | missing |
| Legacy `azure` auth provider | not documented | rejected by client-go, which points to kubelogin | – (nothing to build) |
| `proxy-url` in the kubeconfig | not documented | yes | missing |
| HTTP proxy set per cluster in cluster settings | yes | yes | missing |
| Custom CA, `tls-server-name`, `insecure-skip-tls-verify` | not documented | yes | missing |
| Users who may list only some namespaces (accessible namespaces) | yes | yes | missing |
| Cluster Connect: reach clusters through Lens's relay (paid) | yes | not a target ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | – |
| Cloud account integrations: list and add EKS, AKS, GKE clusters (paid) | yes | gap [G18](gaps.md#g18-paid-lens-desktop-features) | missing |

## Kubernetes versions

Lens Desktop does not document which Kubernetes versions it supports (not verified).

| Row | Cogpit target | Status |
|---|---|---|
| Every minor version the Kubernetes project supports (1.35, 1.36 and 1.37 on 2026-10-11) | full support, tested against `kind` and envtest in CI; client-go 0.37.1 is the 1.37 release | partial: 1.36.1 on kind, locally (`make test-kind`); no CI |
| Older clusters | views from discovery; Pod Security Policies shown only before 1.25 | missing |
| Kinds and fields newer than the client libraries | discovery-driven views; objects handled as unstructured content, so edits keep unknown fields | missing |
| Device resources (`resource.k8s.io/v1`, 1.34 and later) and Gateway API (CRDs) | shown only when served | missing |

## External tools

| Tool | Lens Desktop | Cogpit | Status |
|---|---|---|---|
| `kubectl` for terminals | downloads the version matching the cluster (mirror: Google or Azure; directory and path settings) | `PATH` first; when missing or more than one minor version away, a verified download of the matching version on one click ([D-3](../developer/decisions.md#d-3-helm-and-kubectl-binaries)) | partial: `PATH` only, with the usual tool directories added for apps started from Finder; no download, no version check |
| `helm` | bundled binary, Helm 3 ("latest 3.x" per 2026.3.130712) | none needed: the Helm SDK runs in-process | missing |

## Display, input and language

| Row | Lens Desktop | Cogpit target | Status |
|---|---|---|---|
| High-DPI displays and per-monitor scaling | yes (Chromium) | yes | missing |
| Wayland and X11 on Linux | not documented | with Linux: both; Fyne picks Wayland at runtime by default | missing |
| Screen readers | not documented | Fyne's accessibility support, experimental in 2.8 ([spike S8](../developer/spike-fyne.md)) | missing |
| Input methods (IME) in editor and terminal | yes (Chromium) | yes | missing |
| Interface language | English; Preferences offer only a locale timezone | English; text kept in one place so it can be translated later | missing |

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | minor | The MVP: local macOS app bundle, certificate and token authentication, Kubernetes 1.36 on kind, `kubectl` from `PATH` partial |
| 2026-10-11 | minor | macOS first (D-7): Windows and Linux rows move after the macOS beta (G22) |
| 2026-10-11 | major | Freelens column removed (D-12); client-go facts replace kube-rs; Helm in-process; one-click updates and signed update manifests; Fyne display facts |
| 2026-10-11 | major | First version, from Lens Desktop's install docs, FAQ and update feeds, Freelens 1.10.3, and the 2026-10-11 ecosystem survey |
