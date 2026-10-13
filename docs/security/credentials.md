# Credentials

Where cluster credentials and Cogpit's own secrets come from, where they live while Cogpit runs,
and where they must never end up. Rules that the MVP (2026-10-11) implements link to their code
and test; the others are the plan. Cluster authentication is client-go's, the code `kubectl` uses
([D-6](../developer/decisions.md#d-6-kubernetes-libraries)), loaded in
[`internal/cluster/cluster.go`](../../internal/cluster/cluster.go).

## Where credentials come from

All cluster credentials come from kubeconfig files the user adds. The kubeconfig format carries
them in these forms ([kubeconfig](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/),
[authentication](https://kubernetes.io/docs/reference/access-authn-authz/authentication/)):

| Form | Lives in | Cogpit's handling |
|---|---|---|
| Client certificate and key (`client-certificate[-data]`, `client-key[-data]`) | the kubeconfig, or a file it names | Read when connecting; held in memory by the TLS stack |
| Static bearer token (`token`, `tokenFile`) | the kubeconfig, or a file it names | Read when connecting; held in memory |
| Exec credential plugin (`exec`) | a program the kubeconfig names | Run on connect and on expiry; the returned credential is held in memory only, never written to disk |
| A proxy's user and password in the cluster's `proxy-url` (`http://user:password@proxy:3128`) | the kubeconfig | Handed to client-go, which sends it to the proxy alone (Basic `Proxy-Authorization` on the CONNECT; in clear for an `http://` proxy); shown nowhere: the plugin approval dialog strips it, and since 2026-10-13 connect errors replace it with `•••` ([`proxyurl.go`](../../internal/cluster/proxyurl.go), `TestKubeconfigProxyPasswordNeverInErrors`) |
| Legacy auth provider `oidc` | the kubeconfig | Refreshed tokens go through a persister Cogpit supplies; see [H-4](#h-4-auth-provider-token-refresh) |
| Legacy auth providers `azure` and `gcp` | the kubeconfig | client-go 0.37.1 rejects `azure` and points to kubelogin; the user replaces them with exec plugins |

Cogpit adds no cluster credential source of its own: no account, no login, no cloud service.

## Cogpit's own secrets

| Secret | Lives in | Rule |
|---|---|---|
| MCP bearer token | `<user config dir>/Cogpit/mcp-token` (macOS: `~/Library/Application Support/Cogpit/mcp-token`), file 0600 in a 0700 directory, written atomically, never through a symlink, removed on exit ([`internal/mcp/token.go`](../../internal/mcp/token.go), `TestTokenFileLifecycle`, `TestStartRefusesSymlinks`) | 32 random bytes, new on every start; printed to the terminal that started Cogpit inside the `claude mcp add` command, never logged (`TestTokenNeverLogged`) ([agents and MCP](agents-and-mcp.md)). Since 2026-10-13 the endpoint's URL is written beside it, `mcp-endpoint`, with the same rules (not a secret: a loopback URL), for the stdio bridge `cogpit-cli mcp`, which reads both and sends the token to that loopback URL only (`TestEndpointFileLifecycle`, `TestReadEndpointTakesOnlyWhatCogpitWrites`) |
| LLM API key (built-in agent) | the operating system's keychain (macOS Keychain first) | never written to a file, never logged, sent only to the configured endpoint |
| Update public key | compiled into the app | not secret; its private counterpart never touches a user's machine ([updates](updates.md)) |

## Where they live inside Cogpit

- **Kubeconfig files stay where they are.** Cogpit stores the paths the user added, not copies
  (built 2026-10-12: `<user config dir>/Cogpit/kubeconfigs.json`, 0600, lists the added files and
  sync folders; Cogpit never writes the files themselves, `TestAddedFilesAreNeverWritten`).
  Since 2026-10-12 Cogpit also reads every regular file of at most 1 MiB in `~/.kube` (a default
  sync folder, dot files and `config` included), each on its own, and the files `$KUBECONFIG` and
  `--kubeconfig` name (D-28: nothing is merged any more); the list holds only the folder's removal,
  the files named for a run are not stored. A credential plugin or an outside token file of any of
  them waits for the user's approval ([H-1](trust-boundaries.md#h-1-kubeconfig-files-added-by-sync-folders)).
  The MVP switches off client-go's two implicit writes: the migration of a legacy
  `~/.kube/.kubeconfig` into `~/.kube/config`, and the persister that writes refreshed `oidc` tokens
  back into the file ([`cluster.go`](../../internal/cluster/cluster.go), `memoryPersister`).
- **Cogpit does not store kubeconfig text.** Pasting a kubeconfig is not a target
  ([D-21](../developer/decisions.md#d-21-pasting-a-kubeconfig), owner, 2026-10-12): a copy of a
  kubeconfig is a second place for a credential to live, and the files the user adds already
  stay where they are. A build from 2026-10-12 that still had the paste feature wrote the text to
  `<user config dir>/Cogpit/kubeconfigs/pasted-<16 hex>.yaml` (0600 in a 0700 directory) and an
  entry of kind `pasted` to `kubeconfigs.json`. Cogpit no longer lists, reads or deletes them
  (`TestPastedSourcesOfEarlierBuildsAreNotOffered`); the files hold credentials in clear text and
  are the user's to delete or to add as files again. The entry in `kubeconfigs.json` goes with
  the next change of the list of sources.
- **Exec plugin approvals** live in `<user config dir>/Cogpit/exec-approvals.json` (0600): the
  fingerprint of what each allowed plugin runs and where its credential goes (including the content
  digest of a program or CA file inside the kubeconfig's folder, and the path of a token file
  outside it), no credential.
- **Token files** (`tokenFile`) of an added or synced kubeconfig are read by client-go when the
  user connects, and only after the user allowed a file outside the kubeconfig's folder
  (since 2026-10-12, D-17, [trust boundaries › H-24](trust-boundaries.md#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token)).
- **Helm chart repositories hold no credential** (built 2026-10-13, G11): Cogpit adds public
  https repositories only, by name and URL, refuses a URL with a user or password (without
  echoing it in the error, `TestCheckRepository`), and keeps the list in its own
  `<user config dir>/Cogpit/helm/repositories.json` (0600), never in Helm's `repositories.yaml`,
  whose entries can hold a password in plain text. The indexes it downloads are public data in
  `<user cache dir>/Cogpit/helm/` (0600). A repository that needs a login is
  [H-29](#h-29-authenticated-helm-repositories).
- **A proxy's user and password** (built 2026-10-13, G13): a cluster's proxy (Cluster Settings ›
  Proxy) and the proxy of the Helm repositories' and kubectl's downloads (Preferences › Proxy) may each be
  `http://user:password@host:port`, as Lens's fields allow. Cogpit keeps each as typed in one place
  only, `<user config dir>/Cogpit/cluster-settings.json` and `preferences.json` (0600 in a 0700
  directory, replaced atomically), and hands it to the HTTP client that uses it (client-go's
  `rest.Config.Proxy` for the connection, the Helm fetcher's transport for a download), which sends
  it to the proxy alone: as Basic `Proxy-Authorization` on the CONNECT for an http or https
  proxy (in clear on the network for an `http://` proxy, over TLS for an `https://` one), as SOCKS5's
  user and password for a socks5 proxy; never to the API server or the repository. Everywhere else
  the password is `•••` (`engine.RedactProxyURL`): the UI state (so `ui_state` and the window),
  `cluster.settings.get`, `preferences.get` and their setters' results, the audit log's parameters
  (`auditParams`), the connection log ("Through the proxy http://alice:•••@proxy:3128"); a value
  typed in the window with `•••` keeps
  the stored password (`MergeProxyPassword`). No error quotes a proxy value: the checks name the
  problem only, and `cluster.Connect` scrubs the password from any connect error as a guard
  (`TestCheckProxyURL`, `TestClusterProxyKeptPrivately`, `TestClusterProxyCommands`,
  `TestProxyRefusalNeverQuotesThePassword`, `TestConnectThroughTheClusterProxy`). Only the user sets
  a proxy ([agents and MCP](agents-and-mcp.md)). The file is plaintext
  ([H-33](#h-33-proxy-passwords-are-plaintext-in-cogpits-settings)).
- **Drafts of unsaved editor text** (built 2026-10-13, G9, [D-11](../developer/decisions.md#d-11-crash-stance)):
  the unsaved text of an Edit or Create resource tab is written to
  `<user config dir>/Cogpit/drafts/<32 hex>.json`, 0600 in a 0700 folder, replaced atomically,
  never through a symlink (`writePrivateFile`); with it the tab's context, object, the version the
  text is based on, the time and the writing Cogpit process. It is removed when the text is saved
  again or the tab is closed. **Never a Secret's values** ([D-19](../developer/decisions.md#d-19-editing-and-creating-secrets-from-yaml)):
  a Secret's Edit tab has no draft, nor has a tab whose text holds a Secret's manifest (a list's
  items and text that does not parse included) or may become one while it is typed (a `kind:` that
  is a beginning of `Secret` or `SecretList`, in any case); a draft whose text turns into one is
  removed, and a draft file found holding one is deleted unread. **Never Helm values**: the Helm
  Upgrade and Install tabs have no drafts ([`drafts.go`](../../internal/engine/drafts.go),
  `TestNoDraftsForSecretsOrHelmValues`, `TestNoDraftsOfSecretTabs`, `TestDraftFilesReadBack`).
  Agents list drafts without their text; a restored tab's text follows `editor.get`'s rules. What
  other kinds' text a user types is kept as typed ([H-38](#h-38-drafts-keep-typed-text-in-plain-text)).
- **In memory:** credentials are held by the connection that uses them and dropped when it closes.
  Rule for Cogpit's own credential types: they implement `slog.LogValuer` and `fmt.Stringer` as
  `"<redacted>"`, so a stray log statement cannot print them. The MVP has no such type yet: cluster
  credentials stay inside client-go's `rest.Config`, which Cogpit never logs, and the MCP token is
  a plain string that only `cmd/cogpit` prints, inside the `claude mcp add` command.

## The terminal's kubeconfig

A terminal opened for a cluster must make `kubectl` in it talk to that cluster and namespace,
without copying credentials and without a local proxy that serves the user's identity to other
processes. Kubeconfig merging gives this for free: when `KUBECONFIG` lists several files, the
first file that sets a value wins
([merging rules](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/#merging-kubeconfig-files)).
Each terminal therefore gets

```text
KUBECONFIG=<temp file>:<the user's kubeconfig files>
```

where the temp file holds only a `current-context` and one context entry naming the cluster and
user entries that the user's own files define, plus the namespace. The temp file contains no
credential, is owner-only, and is deleted when the terminal closes. Implemented in
[`internal/kubeconfig/terminal.go`](../../internal/kubeconfig/terminal.go) and
[`internal/terminal/local.go`](../../internal/terminal/local.go); `TestStartLocalBindsKubectlToContextAndNamespace`
checks that `kubectl` in the shell sees the context and namespace, that the file is 0600, holds no
token and is gone after close, and that `kubectl config use-context` in the terminal leaves the
user's file byte-identical. A shell profile can still override `KUBECONFIG`
([H-20](#h-20-shell-profiles-can-override-the-terminals-kubeconfig)).

When Cogpit puts a `kubectl` first for the terminal (since 2026-10-13, [D-3](../developer/decisions.md#d-3-helm-and-kubectl-binaries)),
the terminal also gets a temporary directory of its own (0700, removed with the terminal) holding a
link named `kubectl` and, for zsh, a `.zshenv` that hands over to the user's own: names and a
script of Cogpit's, no credential ([`terminal/kubectl.go`](../../internal/terminal/kubectl.go)).
That `kubectl` reads the same `KUBECONFIG` as any other.

## Where credentials must never end up

| Place | Rule |
|---|---|
| Application log | No tokens, keys, certificates, exec-plugin output or Secret data, at any log level |
| Audit log | An agent's parameters are logged (D-2), with Secret values, Helm values and a proxy's password redacted ([`audit.go`](../../internal/engine/audit.go) `auditParams`) |
| Error messages and toasts | Same rule; errors name the context, never the credential. client-go's own errors can echo a kubeconfig's `users` entries; `kubeconfig.SafeError` ([`kubeconfig.go`](../../internal/kubeconfig/kubeconfig.go)) strips them, and connect errors are tested not to contain a known token. client-go's validation also quotes a `proxy-url` it cannot use whole (`invalid 'proxy-url' "http://user:password@…"`, "could not parse: …"): fixed 2026-10-13, `cluster.Connect` replaces the user information of every `proxy-url` of the kubeconfig, as written, as `%q` writes it, and the password alone, with `•••` (`TestKubeconfigProxyPasswordNeverInErrors`, `TestKubeconfigProxySecrets`). A certificate the server presents is checked, not a secret: a failed check names the kubeconfig field to look at (`TestConnectTrustsTheKubeconfigCA`) |
| Drafts of unsaved editor text | Never a Secret's values, never Helm values (2026-10-13, see [above](#where-they-live-inside-cogpit)); the drafts' listing for agents carries no text |
| Files saved from the window | The Create resource tab's save (2026-10-13) writes the tab's text only to the file the user picks in the system's save dialog, a GUI action no agent can start, and refuses a text that holds a Secret's manifest ([`createtab_save.go`](../../internal/ui/fyneui/createtab_save.go), `TestCreateTabKeepsSecretsOffDisk`) |
| Crash reports | Cogpit has no crash upload. Since 2026-10-12 the Go runtime writes an unhandled panic or fatal error into `<user config dir>/Cogpit/crash.log` (0600) as well as to standard error ([`crash.go`](../../cmd/cogpit/crash.go)); the next start moves it to `crashes/crash-<time>.log`, keeps the newest ten and names it in the window. Its text is the runtime's: the panic's message and the goroutines' stacks (function names, file lines, argument words, no strings' contents); a panic message the code formats from a value could carry it, so the file follows the log rule: local, owner-only, never sent anywhere |
| Telemetry | Cogpit sends none. Any network call to something other than a cluster, a Helm repository, the configured LLM endpoint, the release channel or the kubectl release host or mirror ([D-3](../developer/decisions.md#d-3-helm-and-kubectl-binaries), a download the user clicked; it sends the version and platform in the URL, no credential) needs a decision entry in [`decisions.md`](../developer/decisions.md) |
| The LLM endpoint | Secret values only when the user allows it for that request ([H-13](agents-and-mcp.md#h-13-cluster-data-leaves-for-the-llm-endpoint)) |
| Screenshots taken by automation | Secret values stay masked unless the user revealed them in the GUI |
| Clipboard | Only on an explicit copy action; a copied Secret value is cleared after 60 seconds if the clipboard still holds it ([H-25](#h-25-a-copied-secret-value-reaches-the-clipboard)) |

## Secret objects

Kubernetes Secrets are data the user's identity may read, not Cogpit's own credentials, but they
are handled with the same care: the list holds metadata only, an opened Secret's values are
masked until the user reveals one field, decoded values are never logged, and a Secret is edited or
created from YAML in the GUI only (since 2026-10-12, D-19: the editor shows its values to the user;
agents and the CLI are refused, and no value reaches the audit log, an error or a toast; since
2026-10-13 nor a draft on disk). As built (2026-10-11): when the
drawer closes or moves, the engine zeroes the bytes it kept of the opened Secret; the GET's
response and a revealed value's string wait for the garbage collector, so the wipe is best effort
([privilege footprint › H-8](privilege-footprint.md#h-8-secret-data-arrives-with-the-list)).
A revealed value can be copied from the drawer, by its copy icon or by selecting it (D-25, built
2026-10-12): a masked value has no icon, and the clipboard is emptied 60 seconds after such a copy
if it still holds the value (`shell.copyText`, `TestSecretValueCopyOnlyOnceRevealed`,
`TestSecretClipboardKeepsWhatReplacedIt`). Copying is a GUI behaviour with no command, so no agent
can put a Secret on the clipboard (`TestCopyHasNoCommand`).

## What this does not cover

### H-4: Auth-provider token refresh

client-go's legacy `oidc` auth provider hands refreshed tokens to a persister, which in `kubectl`
writes them back into the kubeconfig file (client-go 0.37.1,
`plugin/pkg/client/auth/oidc`). Cogpit supplies the persister, so it decides: write through
(kubectl-compatible, but writes the user's file) or keep refreshed tokens in memory (no write, but
`kubectl` in a terminal then refreshes on its own). The rule "no implicit kubeconfig writes" in
[conventions](../developer/conventions.md#never-surprise-a-cluster) argues for memory only, and
the MVP's code keeps them in memory; the owner has not decided, so the behaviour may change.

### H-5: Kubeconfig files are plaintext

Credentials embedded in kubeconfig files are plaintext files protected only by file permissions.
Cogpit does not change that and does not move them into the operating system's keychain. Cogpit
does not check or warn about world-readable kubeconfig files.

### H-6: Exec plugins inherit the environment

An exec plugin runs with Cogpit's environment plus the variables the kubeconfig declares. A GUI
app started from the dock or start menu does not see `PATH` entries and variables set in shell
profiles, so plugins that rely on them (cloud CLIs reading `AWS_PROFILE` and similar) can fail.
Lens's technical FAQ tells users that exec-plugin binaries must be on `PATH` or named with an
absolute path ([Lens technical FAQ](https://docs.lenshq.io/faq/technical-faq/)). As a stopgap the
MVP appends the usual tool directories (`/opt/homebrew/bin`, `/usr/local/bin`, `/opt/local/bin`,
`~/.local/bin`, `~/bin`) that exist and are missing from `PATH` at start
(`cluster.EnsureToolPath`); variables such as `AWS_PROFILE` are still missing. Whether Cogpit reads
the login shell's environment, and which variables it then passes to plugins, is not designed yet.

### H-25: A copied Secret value reaches the clipboard

A Secret's value that the user copies from the drawer goes on the system clipboard, where Cogpit
cannot guard it. It clears the clipboard after 60 seconds if the clipboard still holds the value,
and when the window closes. On macOS (owner's decision, 2026-10-12) the copy is marked with
nspasteboard.org's concealed and transient types, which clipboard managers that honour them
neither show nor record, and is kept to this Mac, out of Universal Clipboard
(`NSPasteboardContentsCurrentHostOnly`; [`clipboard_darwin.m`](../../internal/ui/fyneui/clipboard_darwin.m),
`TestSecretCopyGoesThroughTheConcealedClipboard` for the routing; the AppKit side is checked by
hand, not by a test). What remains: a clipboard manager that ignores those types keeps its own
history of the value; any process of the user can read it while it is there; Cogpit's process
ending within the 60 seconds (a crash, `kill -9`) leaves it on the clipboard; on Linux and Windows
the copy goes through Fyne's clipboard unmarked. Only values the user revealed and copied are
affected; an agent has no way to copy.

### H-29: Authenticated Helm repositories

Cogpit adds public https chart repositories only (2026-10-13): no username and password, no
client certificate and key, no CA file, no plain http, no OCI registry. Lens's preferences offer
all of these. A repository behind a login cannot be added, and a chart it serves cannot be
installed from Cogpit. Where such a credential would live (the operating system's keychain, as for
the LLM key, or a file next to `repositories.json`) and how it is kept out of logs, errors and the
redirects a download follows to other hosts (Helm's `pass_credentials_all`) is not designed.

### H-33: Proxy passwords are plaintext in Cogpit's settings

A proxy's password (a cluster's in `cluster-settings.json`, the Helm downloads' in
`preferences.json`) is kept as the user typed it, in a file
protected by its permissions only (0600 in a 0700 directory), as kubeconfig files keep their tokens
([H-5](#h-5-kubeconfig-files-are-plaintext)). Cogpit does not move it into the operating system's
keychain, where the LLM key is planned to live, and a backup of the configuration folder carries it.
The password reaches the proxy in clear when the proxy's URL is `http://` (Basic authentication on
the CONNECT, as every HTTP client sends it); Cogpit does not warn about that. Whether proxy
credentials move into the keychain, and whether an `http://` proxy with a password is refused or
warned about, is not decided.

### H-38: Drafts keep typed text in plain text

A draft of unsaved editor text (2026-10-13) holds what the user typed into an Edit or Create
resource tab of any kind but a Secret, as typed: a password pasted into a ConfigMap, a token in a
Deployment's `env`, a manifest that names a Secret's values under another kind's fields. It sits
in `<user config dir>/Cogpit/drafts/` protected by its permissions only (0600 in a 0700
directory) until the text is saved, the tab closed or the draft discarded, and a crash leaves it
there until the next connect of its context offers it; a backup of the configuration folder
carries it. Before drafts that text lived in memory only. A draft is not encrypted, and the
Secret check reads kinds, not values. Whether drafts are encrypted (a key in the operating
system's keychain), expire, or can be switched off in the Preferences is not decided.

### H-20: Shell profiles can override the terminal's kubeconfig

The terminal starts a login shell, so the user's shell profile runs after Cogpit set `KUBECONFIG`.
A profile that exports `KUBECONFIG` (common with tools that manage several kubeconfig files)
replaces Cogpit's value, and `kubectl` in that terminal then talks to whatever context the profile
selects, not to the cluster whose view opened the terminal. Lens has the same behaviour. Cogpit
does not detect it; a check after the profile ran, or a prompt hook, is not designed.
