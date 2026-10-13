# Platform parity: systems, packages, authentication and tools

> **Last reviewed:** 2026-10-12 · **Last updated:** 2026-10-13 · **Change:** minor (`proxy-url` in the kubeconfig and the custom CA, `tls-server-name` and `insecure-skip-tls-verify` partial, against TLS test servers; `kubectl` for terminals: the version check and the verified download, D-3; the HTTP proxy per cluster partial; fourth run recount: 9 of 38 rows partial; exec credential plugins partial) · **Target:** Lens Desktop 2026.10.51501

Where and how Cogpit runs compared with Lens Desktop 2026.10.51501: operating systems and
architectures, packages and updates, how it authenticates to clusters, which Kubernetes versions
it handles, and the external tools it depends on. Method: [`README.md`](README.md).

**Sources.** Lens Desktop: [install](https://docs.lenshq.io/k8slens/getting-started/install-lens/),
[technical FAQ](https://docs.lenshq.io/faq/technical-faq/),
[preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/), its update feeds
(`latest.yml`, `latest-mac.yml`, `latest-linux.yml` on `downloads.k8slens.dev`) and the
2026.10.51501 release notes. Cogpit's ecosystem facts: [decisions](../developer/decisions.md).

**Summary (measured, 2026-10-13, after the night run, counted by `make parity`):** 12 of 38 rows partial,
none done (four more are not targets or have nothing to build): the macOS app bundle (1 of 13
package rows), client certificates, bearer tokens, exec credential plugins (run only after the user allowed
them in the GUI, not yet verified against a cloud provider), users limited to some namespaces
through the access reviews, the HTTP proxy per cluster, `proxy-url` and the kubeconfig's TLS
settings (custom CA, `tls-server-name`, `insecure-skip-tls-verify`; against TLS test servers, not a
cloud cluster) (7 of 13 authentication rows), Kubernetes 1.36 on kind locally and 1.37 in CI, kinds newer
than the client libraries (custom resources and edits), the sidebar following the served APIs
version by version, and `kubectl` from `PATH`. Cogpit ships for macOS first; Windows and Linux follow after the macOS beta
([D-7](../developer/decisions.md#d-7-language-and-platforms)). Remaining (estimated): 25–50 h for
the macOS app, signing and one-click updates ([G14](gaps.md#g14-packaging-signing-and-updates-macos)),
40–80 h for Windows and Linux ([G22](gaps.md#g22-windows-and-linux)) and 20–45 h for authentication
([G2](gaps.md#g2-cluster-connection-and-authentication)), both with work a human must do: signing
material, and real cloud accounts to test credential plugins against.

## Operating systems and packages

| Platform | Lens Desktop | Cogpit target | Status |
|---|---|---|---|
| macOS | dmg for x64 and arm64, macOS 13 or later | the universal app bundle `Cogpit.app` (arm64 + x86_64) in a signed, notarized disk image to drag into Applications; macOS 13 or later ([D-4](../developer/decisions.md#d-4-packaging-and-updates)) | partial: `make app` ([`Makefile`](../../Makefile)) builds an ad-hoc signed `Cogpit.app` for arm64 locally; not universal, no disk image, not notarized |
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
| Bearer token, inline or token file | yes | yes | partial: inline, against a fake API server over HTTPS (`TestBudgetIdleViewAndWarmReopen`, [`cluster_test.go`](../../internal/cluster/cluster_test.go)); a token file of a kubeconfig source outside the kubeconfig's folder is asked for in the plugin dialog before it is read (2026-10-12, D-17, `TestTokenFileOutsideTheFolderIsAskedFor`); a token file read against a server not tested |
| Exec credential plugin (`client.authentication.k8s.io`) | yes; the binary must be on `PATH` or absolute; a synced file's plugin runs on connect without asking (capture, 2026-10-12) | yes, kubectl's own implementation | partial (2026-10-12): a test plugin authenticated against kind (an ExecCredential with the kind user's client certificate, `TestKindExecPluginAfterApproval`) and a fake API server (a token, `TestConnectAsksTheExecPolicyBeforeThePluginRuns`), a relative command resolved against its kubeconfig's folder as client-go does; a plugin of a kubeconfig source added in Cogpit runs only after the user allowed it in the GUI, a program or CA file inside the kubeconfig's folder is bound by its content, so replacing it asks again (2026-10-12, `TestExecApprovalRefusedAfterPluginBinaryChanged`), and the dialog shows the proxy, insecure and own-CA settings the approval binds; the default kubeconfig's run as kubectl runs them ([`cluster/sources.go`](../../internal/cluster/sources.go), [`engine/kubeconfigs.go`](../../internal/engine/kubeconfigs.go)). Missing: the cloud plugins (rows below), the plugin's environment ([H-6](../security/credentials.md#h-6-exec-plugins-inherit-the-environment)) |
| Exec plugin: `aws eks get-token` and `aws-iam-authenticator` (EKS) | yes | to verify against a real cluster (human) | missing |
| Exec plugin: `gke-gcloud-auth-plugin` (GKE) | yes | to verify (human) | missing |
| Exec plugin: `kubelogin` (AKS, Entra ID) | yes | to verify (human) | missing |
| Exec plugin: `kubectl oidc-login` (generic OIDC) | not documented | to verify | missing |
| Exec plugins that open a browser or prompt (interactive mode) | not documented | to design ([H-6](../security/credentials.md#h-6-exec-plugins-inherit-the-environment)) | missing |
| Legacy `oidc` auth provider | not documented | yes; refreshed tokens through a persister Cogpit supplies ([H-4](../security/credentials.md#h-4-auth-provider-token-refresh)) | missing |
| Legacy `azure` auth provider | not documented | rejected by client-go, which points to kubelogin | – (nothing to build) |
| `proxy-url` in the kubeconfig | not documented | yes | partial (2026-10-13): client-go's `clientcmd` turns it into `rest.Config.Proxy` (code read, client-go 0.37.1); verified through the cluster layer: every request of the connection goes through the test forward proxy, the URL's user and password to the proxy alone, and the proxy of Cogpit's cluster settings replaces it, the kubeconfig's proxy then seeing no request (`TestKubeconfigProxyURLIsHonoured`); fixed the same day: client-go quotes a `proxy-url` it cannot use whole in its error (`invalid 'proxy-url' "http://user:password@…"`), which reached the connect log, the window and agents; `cluster.Connect` now replaces its user information with `•••` (`TestKubeconfigProxyPasswordNeverInErrors`, `TestKubeconfigProxySecrets`). Not against a real proxy; the picker does not show it (the plugin approval dialog does, without its user information) |
| HTTP proxy set per cluster in cluster settings | yes | yes | partial (2026-10-13): Settings › Proxy ([ui parity](ui-parity.md#cluster-settings)), an http, https or socks5 URL kept in Cogpit's `cluster-settings.json`, applied at the next connect through `rest.Config.Proxy` in place of the kubeconfig's `proxy-url` (inferred precedence), its password redacted everywhere but that file; against a fake API server behind a test proxy that tunnels CONNECT and checks the Basic credentials (`TestConnectGoesThroughTheProxy`, `TestConnectThroughTheClusterProxy`); not against a real proxy; the local terminal's kubectl does not use it |
| Custom CA, `tls-server-name`, `insecure-skip-tls-verify` | not documented | yes | partial (2026-10-13): honoured by client-go's `clientcmd`, which copies them into `rest.Config`'s TLS settings (code read); verified through the cluster layer against the fake API server presenting a test CA's certificate ([`testkit/tlsca.go`](../../internal/testkit/tlsca.go)): `certificate-authority-data` and `certificate-authority` connect, without them or with another CA the connect fails with a sentence that names the certificate problem and the kubeconfig fields to check, never the token (`TestConnectTrustsTheKubeconfigCA`); a certificate for another host connects only with `tls-server-name` naming it (`TestConnectWithTLSServerName`); `insecure-skip-tls-verify` connects to a certificate nobody trusts and sends the token there, and together with the kubeconfig's own CA is refused with client-go's sentence (`TestInsecureSkipTLSVerifyConnectsToAnUntrustedServer`). Such a context is marked in the picker ("certificate not checked" and a warning line) and its connect log gets a WARN line (Cogpit's default, Lens documents nothing; `TestInsecureContextIsMarkedAndLogged`, `TestPickerMarksAnInsecureContext`, [PNG](../../internal/ui/fyneui/testdata/picker_insecure_dark.png); the risk: [H-35](../security/trust-boundaries.md#h-35-a-context-that-skips-the-certificate-check)). A plain-http server gets no credential (`TestPlainHTTPServerGetsNoCredential`). Not against a cloud cluster's own CA |
| Users who may list only some namespaces (accessible namespaces) | yes | yes | partial (2026-10-11): access reviews per namespace (`SelfSubjectRulesReview`) of the context's default namespace and the namespace filter's: the sidebar shows the kinds such a user may list there and hides the others, menus grey what they may not do ([`accessreviews.go`](../../internal/engine/accessreviews.go); `TestKindAccessReviews` with a ServiceAccount limited to pods in one namespace, `TestBudgetAccessReviews`). Missing: views still list every namespace (no namespace-scoped informers), so such a user's Pods view shows the API server's refusal; no setting for accessible namespaces when the kubeconfig names none |
| Cluster Connect: reach clusters through Lens's relay (paid) | yes | not a target ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)) | – |
| Cloud account integrations: list and add EKS, AKS, GKE clusters (paid) | yes | gap [G18](gaps.md#g18-paid-lens-desktop-features) | missing |

## Kubernetes versions

Lens Desktop does not document which Kubernetes versions it supports (not verified).

| Row | Cogpit target | Status |
|---|---|---|
| Every minor version the Kubernetes project supports (1.35, 1.36 and 1.37 on 2026-10-11) | full support, tested against `kind` and envtest in CI; client-go 0.37.1 is the 1.37 release | partial: 1.37.0 on kind in CI (the Cluster Tests job of [`build.yml`](../../.github/workflows/build.yml)) and 1.36.1 locally (`make test-kind`); no 1.35, no envtest |
| Older clusters | views from discovery; Pod Security Policies shown only before 1.25 | missing |
| Kinds and fields newer than the client libraries | discovery-driven views; objects handled as unstructured content, so edits keep unknown fields | partial: kinds from custom resource definitions get views from their definitions ([`customkinds.go`](../../internal/engine/customkinds.go)); edits replace the object with the edited unstructured text, so fields the client libraries don't know survive ([`editor.go`](../../internal/engine/editor.go)); built-in kinds without a profile and aggregated APIs get no view yet |
| The sidebar follows the served APIs, version by version | Lens shows Mutating Admission Policies on a 1.36.2 cluster and not on a 1.35.x one (captures, 2026-10-11; `admissionregistration.k8s.io/v1` from 1.36); Cogpit's profiles are filtered by discovery, so a kind appears where it is served; to be tested against kind node images of each supported minor version | partial: discovery filtering ([`connector.go`](../../internal/engine/connector.go)); only 1.36.1 on kind |
| Device resources (`resource.k8s.io/v1`, 1.34 and later) and Gateway API (CRDs) | shown only when served | missing |

## External tools

| Tool | Lens Desktop | Cogpit | Status |
|---|---|---|---|
| `kubectl` for terminals | downloads the version matching the cluster (mirror: Google or Azure; directory and path settings) | `PATH` first; when missing or more than one minor version away, a verified download of the matching version on one click ([D-3](../developer/decisions.md#d-3-helm-and-kubectl-binaries)) | partial (2026-10-13, D-3 built): `PATH` first, with the usual tool directories added for apps started from Finder ([`cluster.go`](../../internal/cluster/cluster.go) `EnsureToolPath`, `TestEnsureToolPath`); when a terminal opens, its `kubectl` (the Preferences' path, else the first on Cogpit's `PATH`, `version --client` read once per file) is compared with the API server's version from connect: within one minor version nothing changes; otherwise a release Cogpit downloaded earlier that suits the cluster goes first, else the tab offers the server's release with a Download button (GUI-only, `kubectl.download`), fetched from `dl.k8s.io` or an https mirror and verified against the published SHA-256 and the Kubernetes release's sigstore signature before it goes into `<user config dir>/Cogpit/kubectl/<version>/kubectl`; the terminals opened afterwards find it first on `PATH` (zsh even after its profile; other shells as their profile orders `PATH`, [H-36](../security/trust-boundaries.md#h-36-a-shells-profile-can-put-another-kubectl-first)). Differs from Lens (inferred from its documentation: it downloads on its own): asks first and leaves a working setup alone; the line and its wording are Cogpit's; the mirror is `dl.k8s.io` or a custom https base, not a Google or Azure choice; the transparency log is not checked ([H-35](../security/trust-boundaries.md#h-35-kubectls-signature-is-checked-without-sigstores-transparency-log)). Not tried in a real window; checked against the real `dl.k8s.io` by hand (1.36.1, darwin/arm64); `TestCheckKubectl`, `TestDownloadKubectl`, `TestDownloadVerifiesAndInstalls`, `TestRealReleaseSignature`, `TestZshKubectlFirstAfterTheProfile`, `TestKubectlOfferDownloadsThroughTheEngine`; [`engine/kubectl.go`](../../internal/engine/kubectl.go), [`internal/kubectl`](../../internal/kubectl/kubectl.go), PNG [dark](../../internal/ui/fyneui/testdata/terminal_kubectl_offer_dark.png), [light](../../internal/ui/fyneui/testdata/terminal_kubectl_offer_light.png) |
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
| 2026-10-13 | minor | Night run (V2, G2): `proxy-url` in the kubeconfig and the custom CA, `tls-server-name` and `insecure-skip-tls-verify` partial, through the cluster layer against TLS test servers, a `proxy-url` password kept out of connect errors (fixed), insecure contexts marked; 12 of 38 rows partial, 7 of 13 authentication rows; authentication's estimate 20–45 h (was 25–50) |
| 2026-10-13 | minor | `kubectl` for terminals (External tools, not counted): the version check when a terminal opens, the verified download on one click, the release first on the terminal's `PATH` (D-3 built, night run) |
| 2026-10-13 | minor | The HTTP proxy set per cluster partial (night run, G13): Cogpit's cluster settings, applied through `rest.Config.Proxy`, tested against a test proxy; 10 of 38 rows partial (the lead recounts) |
| 2026-10-12 | major | Recount after the fourth run (`make parity`): 9 of 38 rows partial, 4 of 13 authentication rows |
| 2026-10-12 | minor | Fourth run (agent D): exec credential plugins partial, on kind and a fake API server, with the approval of a kubeconfig source's plugin |
| 2026-10-12 | minor | Recount by `make parity`: 8 of 38 rows partial (the summary left out the sidebar following the served APIs); evidence named for the macOS app bundle and `kubectl` from `PATH` |
| 2026-10-11 | minor | Users who may list only some namespaces partial: access reviews per namespace hide the kinds they may not list and grey what they may not do |
| 2026-10-11 | minor | Kubernetes 1.37.0 on kind in CI; kinds and fields newer than the client libraries partial (custom resource views, edits keep unknown fields); 6 of 38 rows |
| 2026-10-11 | minor | The MVP: local macOS app bundle, certificate and token authentication, Kubernetes 1.36 on kind, `kubectl` from `PATH` partial |
| 2026-10-11 | minor | macOS first (D-7): Windows and Linux rows move after the macOS beta (G22) |
| 2026-10-11 | major | Freelens column removed (D-12); client-go facts replace kube-rs; Helm in-process; one-click updates and signed update manifests; Fyne display facts |
| 2026-10-11 | major | First version, from Lens Desktop's install docs, FAQ and update feeds, Freelens 1.10.3, and the 2026-10-11 ecosystem survey |
