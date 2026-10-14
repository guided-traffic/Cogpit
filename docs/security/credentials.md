# Credentials

Where cluster credentials and Cogpit's own secrets come from, where they live while Cogpit runs,
and where they must never end up. Rules that the MVP (2026-10-11) implements link to their code
and test; the others are the plan. Cluster authentication is client-go's, the code `kubectl` uses
([D-6](../developer/decisions.md#d-6-kubernetes-libraries)), loaded in
[`internal/cluster/cluster.go`](../../internal/cluster/cluster.go). The owner's decision round of
2026-10-14 evening confirms H-33, H-40 and H-66 as they are and decides changes to who sets a
proxy and to what the certificate checks show; each is marked where it applies, beside what is
built.

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

Since 2026-10-13 ([D-29](../developer/decisions.md#d-29-helm-chart-repositories-from-helms-repositoriesyaml))
Cogpit also reads the credentials of Helm chart repositories, from the file the helm CLI keeps
them in:

| Form | Lives in | Cogpit's handling |
|---|---|---|
| A repository's `username` and `password`, or a user and password in its `url` | Helm's `repositories.yaml` (plain text, written by `helm repo add`), resolved as the helm CLI resolves it | Read with the file (every 2 seconds when it changed, at most 1 MiB) into memory only, unexported in `helm.Repository`; sent as basic auth to the repository's own scheme, host and port only, unless the entry says `pass_credentials_all` (below) |
| A client certificate and key (`certFile`, `keyFile`), a CA file (`caFile`) | files the entry names by absolute path | Read at every download, at most 1 MiB each, into that download's TLS configuration; an error names the path, never the content |

Cogpit never writes Helm's file or anything in Helm's folders, and never copies a repository's
credential into its own files.

## Cogpit's own secrets

| Secret | Lives in | Rule |
|---|---|---|
| MCP bearer token | `<user config dir>/Cogpit/mcp-token` (macOS: `~/Library/Application Support/Cogpit/mcp-token`), file 0600 in a 0700 directory, written atomically, never through a symlink, removed on exit ([`internal/mcp/token.go`](../../internal/mcp/token.go), `TestTokenFileLifecycle`, `TestStartRefusesSymlinks`) | 32 random bytes, new on every start; printed to the terminal that started Cogpit inside the HTTP `claude mcp add` command, never logged; an app whose bundle carries `cogpit-cli` prints and copies the stdio registration of that `cogpit-cli` instead, which names no token (since 2026-10-14, [D-2](../developer/decisions.md#d-2-agent-interface-scope), `TestMCPControlOffersTheBundledCLI`) (`TestTokenNeverLogged`) ([agents and MCP](agents-and-mcp.md)). Since 2026-10-13 the endpoint's URL is written beside it, `mcp-endpoint`, with the same rules (not a secret: a loopback URL), for the stdio bridge `cogpit-cli mcp`, which reads both and sends the token to that loopback URL only (`TestEndpointFileLifecycle`, `TestReadEndpointTakesOnlyWhatCogpitWrites`) |
| LLM API key (built-in agent) | the operating system's keychain: on macOS the login keychain, a generic password per connection, service `io.guided-traffic.cogpit.llm`, account the connection's id, label "Cogpit LLM connection", stored and read through the Security framework (`SecItemAdd`, `SecItemCopyMatching`, `SecItemUpdate`, `SecItemDelete`), never through the `security` command line, which would put the key on a command line every process can read ([`agent/keystore_darwin.go`](../../internal/agent/keystore_darwin.go), `TestMacKeyStoreRoundTrip` with `COGPIT_KEYCHAIN_TEST=1`, run locally 2026-10-14); on Windows and Linux none yet, so a key is refused (M11) | never written to a file: `llm-connections.json` holds whether a key is stored, never the key (`TestConnectionsKeepTheKeyInTheKeychain`); never logged; never in an error, a note or a request body (`TestTurnErrors`); sent only in the `Authorization` header to the configured endpoint, which no redirect can change ([D-10](../developer/decisions.md#d-10-built-in-agent)); deleted with its connection |
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
- **Helm repositories allowed over plain http** (2026-10-14, [H-40](#h-40-credentials-over-plain-http))
  live in `<user config dir>/Cogpit/helm-http-approvals.json` (0600): per repository its name, its
  URL as Cogpit shows it (no user, password or query values), a SHA-256 of its name, URL and user
  name, and when; no credential (`TestHelmCredentialsOverHTTPWaitForTheUser`).
- **Token files** (`tokenFile`) of an added or synced kubeconfig are read by client-go when the
  user connects, and only after the user allowed a file outside the kubeconfig's folder
  (since 2026-10-12, D-17, [trust boundaries › H-24](trust-boundaries.md#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token)).
- **Helm chart repositories' credentials stay in Helm's file and Cogpit's memory** (built
  2026-10-13, [D-29](../developer/decisions.md#d-29-helm-chart-repositories-from-helms-repositoriesyaml)):
  Cogpit reads the repositories of Helm's own `repositories.yaml` and keeps an entry's user and
  password in a `helm.Repository` whose credentials are unexported; `%v`, `%+v`, `%#v`, slog and
  JSON print its name and redacted URL (`TestReadRepositoryFile`). Agents, the dialog and
  `helm.repo.list` see the name, the URL without user, password or query values, and whether
  credentials, a client certificate, a CA file, `insecure_skip_tls_verify`,
  `pass_credentials_all` or plain http are present, never a credential. Basic auth is added per
  request by a transport that compares the request's scheme, host and port with the repository's,
  so a redirect to another host or port, a subdomain included, and a chart URL the index points
  elsewhere get none, unless the entry says `pass_credentials_all` (helm's own option)
  (`TestCredentialsStayOnTheRepositoryHost`, `TestCredentialsNotSentToAChartElsewhere`). Download
  errors show URLs redacted and are scrubbed of the password and user name as a guard
  (`TestScrubbedErrors`); the list, the install tab, events, toasts and notifications, the audit log
  and the log are checked for them (`TestHelmRepoCredentialsNeverShown`; on kind
  `TestKindHelmInstall`). The indexes Cogpit downloads are written to its own
  `<user cache dir>/Cogpit/helm/indexes/` (0600 in a 0700 folder): an index of a repository behind a
  login is what that login protects, readable to the user's own processes as Helm's cache is. The
  night run's `<user config dir>/Cogpit/helm/repositories.json` (public repositories, no
  credential) is left on disk, unread. Since 2026-10-14 the chart drawer's README is read from the
  chart's archive through the same download, so the same rules apply to it, and kept in
  `<user cache dir>/Cogpit/helm/readmes/` (owner-only) like the indexes; the charts' icons are fetched
  with no credential at all, not the repository's whatever `pass_credentials_all` says, no cookies
  ([D-46](../developer/decisions.md#d-46-chart-icons-from-the-hosts-that-serve-them),
  `TestChartIconRefusals`: an icon behind the repository's login gets a 401). What remains is
  [H-29](#h-29-authenticated-helm-repositories) and [H-40](#h-40-credentials-over-plain-http).
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
  a proxy ([agents and MCP](agents-and-mcp.md)); decided (owner, 2026-10-14 evening), not built
  yet: an agent may set one as a write the user confirms, its password `•••` in the confirmation,
  the audit log and every result, the same redaction as above. The file is plaintext
  ([H-33](#h-33-proxy-passwords-are-plaintext-in-cogpits-settings), confirmed by the owner the same
  evening).
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
- **The tabs and the tables' layout** (built 2026-10-13, night build,
  [D-41](../developer/decisions.md#d-41-what-survives-a-restart)): `<user config dir>/Cogpit/workspace.json`,
  0600 in a 0700 directory, replaced atomically through `writePrivateFile`, holds per tab a context
  id, view keys and titles, namespace and API group names, the search text as typed and the sort,
  column ids per view, and since 2026-10-15 the navigator's collapsed source groups (the labels of
  synced folders and added files, A63): names of contexts, kubeconfig files and namespaces, no credential, no
  object's value, no selection and no drawer object
  ([`persist.go`](../../internal/engine/persist.go), `TestWorkspaceRoundTrip`). A context read back
  from it connects only when the user shows its tab, through the same path as the picker (its
  credential plugin still needs the approval), and only a context a kubeconfig source still offers
  keeps its tab.
- **In memory:** credentials are held by the connection that uses them and dropped when it closes.
  Rule for Cogpit's own credential types: they implement `slog.LogValuer` and `fmt.Stringer` as
  `"<redacted>"`, so a stray log statement cannot print them. The first such type is
  `helm.Repository` (2026-10-13), which also implements `fmt.GoStringer` and `json.Marshaler`
  (its name and redacted URL; its user and password in an unexported value that prints as
  `<redacted>`). Cluster credentials stay inside client-go's `rest.Config`, which Cogpit never
  logs, and the MCP token is a plain string that only `cmd/cogpit` prints, inside the
  `claude mcp add` command.

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
| Helm repositories' credentials | In memory only, sent to the repository's own scheme, host and port unless its entry says `pass_credentials_all` (2026-10-13, [above](#where-they-live-inside-cogpit)); never in a result, error, toast, the audit log or the log |
| Telemetry | Cogpit sends none. Any network call to something other than a cluster, a Helm repository, the configured LLM endpoint, the release channel or the kubectl release host or mirror ([D-3](../developer/decisions.md#d-3-helm-and-kubectl-binaries), a download the user clicked; it sends the version and platform in the URL, no credential) needs a decision entry in [`decisions.md`](../developer/decisions.md). The check of kubectl's signature against sigstore's transparency log (decided by the owner on 2026-10-14 evening, not built yet, [H-35](trust-boundaries.md#h-35-kubectls-signature-is-checked-without-sigstores-transparency-log)) is such a call |
| The LLM endpoint | Secret values only over a connection the user marked private (decided 2026-10-14 late evening, built in the engine the same night: the private turn reads what the window shows, Secret values, Helm's values, Flux's credential fields and Helm's errors included; an exec plugin's arguments stay withheld from it as from every agent: [H-13](agents-and-mcp.md#h-13-cluster-data-leaves-for-the-llm-endpoint), [H-94](agents-and-mcp.md#h-94-a-private-connection-gives-the-model-what-the-window-shows)) |
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
can put a Secret on the clipboard (`TestCopyHasNoCommand`). Since 2026-10-13 a list's cells copy
with the same icon (D-25): the Secrets list shows Name, Namespace, Labels, the names of the keys,
Type and Age, never a value (`TestSecretDetailShowsNoValue`), so a copy from it is a plain one and
carries no value (`TestSecretsListCopiesNoValue`). Since 2026-10-13 the drawer decodes the
certificates a Secret's keys hold into separate fields, for the window and agents alike
([D-30](../developer/decisions.md#d-30-certificates-in-secrets-decoded),
[`secretcerts.go`](../../internal/engine/secretcerts.go)): a certificate is public material; only
certificates are decoded (PEM `CERTIFICATE` blocks, a private key block beside one never handed to
the decoder; DER certificates, a DER private key never handed to the certificate parser), a key
named like a private key (`tls.key`, `*.key`, `*key.pem`, `*key.der`) is never searched for
certificates, and the decoded value is zeroed after parsing as the Data section's are
(`TestParseCertificatesOnlyCertificateBlocks`, `TestDecodeDERCertificates`,
`TestObjectGetDecodesCertificatesForAgents`).

**A private key is parsed in memory, for one check (since 2026-10-13, the owner's answer on
D-30).** The owner asked for the drawer to say whether `tls.key` belongs to `tls.crt`, which
cannot be known without the key: Cogpit decodes the private key of a key that pairs with a
certificate key (`tls.key` with `tls.crt`, and the other unambiguous pairs D-30 lists) from the
cached Secret, at most 64 KiB, parses it (PKCS #1, PKCS #8, SEC 1; RSA, ECDSA, Ed25519, X25519),
compares its public half with the first certificate's public key, and drops it. What keeps it in:
the parsed key lives in one function call ([`model/keymatch.go`](../../internal/model/keymatch.go),
`MatchPrivateKey`) and is never returned, stored or cached, not even for the next rebuild of the
drawer, which parses it again; the outcome is yes, no or a fixed reason (`encrypted`, `unsupported
key type`, `not readable`, `too large`), and no error of a parser leaves the check, so no section,
`object_get` result, error, log line or audit line can quote the key; the decoded value and the PEM
block are zeroed after, an Ed25519 key too, while an RSA or ECDSA key's numbers wait for the
garbage collector (best effort, as for the values: Go cannot wipe them). An encrypted key is not
decrypted (`TestSecretKeyMatchesCertificate`, `TestMatchPrivateKeyReasons`,
`TestObjectGetDecodesCertificatesForAgents`, which searches the result, the audit log, the events
and the log for the key's PEM, base64 and DER).

**Decided (owner, 2026-10-14 evening), two changes to these checks, the first built the same
night:**

- **A combined PEM** (a certificate and its private key in one value) gets the key-matches-certificate
  check too ([D-30](../developer/decisions.md#d-30-certificates-in-secrets-decoded),
  `TestSecretCombinedPEMKeyCheck`): the value's key is parsed in memory for the check, under the
  rules of the paragraph above, and only the outcome leaves it; the certificate parser still never
  sees the key block, and finding it reads the blocks' types only, zeroing each block decoded
  (`model.HoldsPrivateKeyBlock`).
- **The chain's anchor** (a new requirement): beside the key match and the chain's order, the
  drawer and agents say whether the chain leads to a public certificate authority or an internal
  (private) one, with the root's name. Built 2026-10-14 night as the lead's design (D-30): a root of
  a public root program that Cogpit carries itself (Mozilla's NSS bundle, independent of the Mac);
  a root this Mac trusts (its trust store, a corporate root included); no known root (internal or
  self-signed). Certificates are
  public material (D-30), so a root's name adds nothing secret about the cluster. What would be new
  for agents, and through the built-in agent for its LLM endpoint
  ([agents and MCP › H-13](agents-and-mcp.md#h-13-cluster-data-leaves-for-the-llm-endpoint)), is
  the second answer: it tells them about this computer's trust settings, not the cluster's.

### Secret values in the Create Secret dialog

Since 2026-10-13 (G9, [D-19](../developer/decisions.md#d-19-editing-and-creating-secrets-from-yaml))
the Secrets view's "+" opens Create Secret: name, namespace, type, the data as rows of a key and a
value, labels and annotations ([`createdialog_secret.go`](../../internal/ui/fyneui/createdialog_secret.go),
[`commands/secretcreate.go`](../../internal/engine/commands/secretcreate.go)). Where a value goes,
as built:

- **Typed into a masked field.** Each value is a Fyne entry in password mode (one dot per
  character, a line per line), whose eye shows that value only. A masked entry neither copies nor
  cuts; a shown one does, through the clipboard for Secret values the drawer uses (since
  2026-10-14, H-70 closed): marked concealed on macOS, cleared after 60 seconds if the clipboard
  still holds it, a toast that says so; ⌘C, ⌘X and the field's menu take that one path
  ([`createdialog_secret.go`](../../internal/ui/fyneui/createdialog_secret.go) `secretEntry`,
  `TestSecretDialogCopiesAShownValueConcealed`; [H-25](#h-25-a-copied-secret-value-reaches-the-clipboard)
  for what remains of any copy). The widgets,
  and the entry's undo history, hold the text until the dialog is gone and the garbage collector
  takes it, as a revealed value in the drawer does.
- **To the engine, in the GUI's own process.** The dialog dispatches `secret.create` with origin
  `gui`; the user's own commands are not audited, and the command refuses every other origin
  before anything is asked or sent (`engine.ErrSecretCreateGUIOnly`). An agent's refused call is
  audited with the Secret's name, namespace, type, its keys and the annotations' keys, each value
  replaced by `<redacted>`, and only the names of parameters the command does not have
  (`engine.auditSecretCreateParams`, `TestSecretCreateIsGUIOnly`).
- **To the API server, base64-encoded in `data`** (Cogpit's default, not `stringData`: what is sent
  is what the server keeps and returns, as `kubectl create secret` sends it), first in a server-side
  dry run, then the create. The checks before it name keys and positions, never values; an API
  message that quotes a value comes back with it removed, raw, base64 and decoded alike
  (`TestSecretCreateChecksBeforeSending`, `TestSecretCreateErrorsNeverQuoteValues`).
- **Nowhere else.** No draft (a dialog keeps none, and it opens no dock tab whose text would), no
  toast beyond "Created secrets/<namespace>/<name>", no log line.
**Argo CD's repository and cluster Secrets (since 2026-10-13, [D-35](../developer/decisions.md#d-35-what-argo-cds-repository-and-cluster-secrets-show),
decided by the owner 2026-10-14).** Settings › Repositories and Settings › Clusters of
the Argo CD section read the Secrets Argo CD labels `argocd.argoproj.io/secret-type`, which hold
repository passwords, SSH and TLS keys, GitHub App keys, bearer tokens and whole cluster client
configurations. Of each, Cogpit reads only an allowlist of configuration keys (a repository's
`name`, `url`, `type`, `project`; a cluster's `name`, `server`, `namespaces`, `clusterResources`,
`project`) into objects of its own kind and names the other keys without reading them; user
information in a URL is replaced with `<redacted>`. The window and agents see the same allowlist;
the Secret itself stays a Secret, masked in its own drawer
([`kinds/argocd_settings.go`](../../internal/model/kinds/argocd_settings.go),
`TestArgoSettingsNeverShowCredentials`, `TestArgoSettingsNeverReachAgents`). Where those values
live meanwhile: [H-66](#h-66-argo-cds-credentials-are-in-cogpits-memory-while-its-settings-are-read).

## What this does not cover

### H-66: Argo CD's credentials are in Cogpit's memory while its settings are read

The settings views of the Argo CD section read Argo CD's labelled Secrets through a watch that
holds them in full (D-13's covering scopes: from the Secrets view's watch while that runs, else a
watch of their own with the label selector), so their passwords, private keys, tokens and cluster
configurations are in Cogpit's memory while a settings view, its drawer or an agent's read uses
them and for the cache's warm period after (five minutes), though nothing shows them. RBAC cannot
narrow `list secrets` by label, so the verbs are the Secrets view's. Dropping the other keys'
values in the informer's transform for these scopes would keep them out of memory; it is not built,
because a covering scope serves the same objects to the Secrets view, whose drawer needs the
values.

Confirmed by the owner (2026-10-14 evening): Argo CD's labelled Secrets stay whole in the cache.
The reason given with it: the filter that replaces cached Secret values in what agents read ([H-82](agents-and-mcp.md#h-82-secret-values-printed-into-logs-reach-agents)) is built
from the values the cache holds, so a value dropped from the cache would reach agents unreplaced
where a log or an object prints it. What remains, live while a settings view, its drawer or an
agent's read uses them and for five minutes after: the values are in Cogpit's memory, shown
nowhere.

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

A Secret's value that the user copies from the drawer, or a value shown in the Create Secret
dialog (since 2026-10-14), goes on the system clipboard, where Cogpit
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

Built 2026-10-13 ([D-29](../developer/decisions.md#d-29-helm-chart-repositories-from-helms-repositoriesyaml)):
Cogpit reads Helm's own `repositories.yaml`, credentials included, keeps them in memory, sends
basic auth only to the repository's own scheme, host and port (unless `pass_credentials_all`),
reads a CA file, client certificate and key at every download, and keeps every credential out of
results, errors, toasts, the audit log and the log ([above](#where-they-live-inside-cogpit)).
Cogpit never writes Helm's file: repositories and their credentials are added with the helm CLI
(owner, 2026-10-13). What remains, per case:

- **Helm's file is plain text.** The password sits in `repositories.yaml` as `helm repo add` wrote
  it, readable to the user's own processes and carried by a backup, as kubeconfig files keep
  their tokens ([H-5](#h-5-kubeconfig-files-are-plaintext)); Cogpit adds no copy but does not
  improve it. Live for every repository added with `--username`/`--password`.
- **In memory for the session.** The user and password are Go strings in `helm.Repository` for as
  long as the entry is in the file, and a client key's bytes for the length of a download (cleared
  after use, best effort: the TLS stack keeps its parsed key until the download's transport is
  dropped). Nothing writes them out; a core dump of the process would hold them.
- **OCI registries.** Credentials of `helm registry login` (Helm's `registry/config.json`) are not
  read; an `oci://` entry is listed as not used (G11).
- **Plain http** sends the credentials readable on the network once the user allowed it for the
  repository (2026-10-14; nothing is sent before): [H-40](#h-40-credentials-over-plain-http).

### H-40: Credentials over plain http

Narrowed 2026-10-14 (owner's answer; [D-29](../developer/decisions.md#d-29-helm-chart-repositories-from-helms-repositoriesyaml)):
a repository in Helm's file whose URL is `http://` and that has a user and password downloads
nothing, neither its index nor a chart nor a README, on an update or in the background, until the
user allowed it once in Cogpit's window: the Charts view (and a Helm Install tab) shows a banner per
such repository, and Allow… says what crosses the network before
`helm.repo.allowHTTPCredentials` records it. The helm package refuses the request by itself
(`helm.ErrCredentialsOverHTTP`, before any connection; its authorization transport never adds the
credentials to a plain http request it was not allowed for), so neither the credentials nor a
request without them leave Cogpit (`TestPlainHTTPRepository`, `TestHTTPBinding`,
`TestHelmCredentialsOverHTTPWaitForTheUser`). The allowance is kept like a credential plugin's,
in `<user config dir>/Cogpit/helm-http-approvals.json` (0600), bound to the repository's name, URL
and user name (a SHA-256, no credential); a changed URL or user name asks again, a changed password
does not. It is the user's only: the command is GUI-only, and an agent or the CLI is refused
whatever it passes, reads why the repository waits in `helm.repo.list` (`httpCredentials`,
`error`) and its refusal is audited (`TestAllowHTTPCredentialsIsTheUsers`). A repository over http
without credentials needs no allowance (only [H-41](trust-boundaries.md#h-41-repositories-whose-content-nothing-authenticates)
applies). What remains, per case:

- **Once allowed, the credentials travel in clear.** Every download of that repository, the
  background refresh while the Charts view or a Helm Install tab is open included, sends the
  basic auth readable to anyone on the network path, a proxy included, as helm sends it; the index
  and charts can be changed on the way (H-41). Live for every repository the user allowed.
- **The allowance does not expire** and binds no password: a rotated password keeps it (the
  owner's binding). A repository removed from Helm's file and added again under the same name, URL
  and user name is allowed again without asking. Confirmed by the owner (2026-10-14 evening): no
  expiry; the allowance stays bound to the name, the URL and the user name. Live for every
  repository the user allowed.
- **The file is the user's own.** A process of the user that writes
  `helm-http-approvals.json` can allow a repository without the window, as it can write
  `exec-approvals.json` or read Helm's file itself
  ([agents and MCP › H-12](agents-and-mcp.md#h-12-every-local-process-of-the-user-can-read-the-token)).
- **Without a configuration directory** the allowances last for the session (in memory).
- **The built-in agent's files** (2026-10-14, [D-10](../developer/decisions.md#d-10-built-in-agent)):
  `<user config dir>/Cogpit/llm-connections.json` (0600 in 0700: each connection's name, kind,
  base URL, model, limits and private mark, and whether a key is in the keychain; a base URL with
  a user or password is refused, so no credential lands there) and `chat-conversations.json`
  (0600: the last 20 conversations per context, every text scrubbed of the cached Secrets' values
  as D-19 keeps them out of drafts; the request bodies sent stay in memory). Both are written
  through the engine's owner-only writer (atomic, symlinks refused). A private connection's turns
  read more than the scrubber knows (Helm's values, Flux's inline credentials, a Secret's value
  under 10 bytes), and that is kept as read
  ([H-96](agents-and-mcp.md#h-96-a-private-turns-reads-are-kept-where-the-scrubber-does-not-know-them)).

### H-95: An LLM API key over plain http

Since 2026-10-14 (the built-in agent's engine side): a connection whose base URL is `http://` sends
its API key in the clear to whoever is on the path, a LiteLLM or Open WebUI on the local network
for example. Cogpit allows it (the lead's decision for the owner to confirm,
[D-10](../developer/decisions.md#d-10-built-in-agent): refusing it would block such a set-up); the
connection test's HTTP layer says "plain http (the API key travels unencrypted)". Live once the
chat's settings can save such a connection, for every `http://` connection with a key to a host
other than this computer; a local LM Studio without a key sends none. What would close it: refusing
a key over `http://` to anything but a loopback address, or asking once per connection.

### H-33: Proxy passwords are plaintext in Cogpit's settings

A proxy's password (a cluster's in `cluster-settings.json`, the Helm downloads' in
`preferences.json`) is kept as the user typed it, in a file
protected by its permissions only (0600 in a 0700 directory), as kubeconfig files keep their tokens
([H-5](#h-5-kubeconfig-files-are-plaintext)). Cogpit does not move it into the operating system's
keychain, where the LLM key lives (since 2026-10-14), and a backup of the configuration folder carries it.
The password reaches the proxy in clear when the proxy's URL is `http://` (Basic authentication on
the CONNECT, as every HTTP client sends it); Cogpit does not warn about that.

Confirmed by the owner (2026-10-14 evening): proxy passwords stay in the 0600 settings files, and an
`http://` proxy with a password gets no warning. What remains, per case: every proxy with a
password sits in plain text in `cluster-settings.json` or `preferences.json`, readable to the
user's own processes and carried by a backup (live wherever the user typed one); an `http://`
proxy's password crosses the network in clear on every CONNECT (live for each such proxy). Once an
agent may set a proxy (decided the same evening, not built, [above](#where-they-live-inside-cogpit)),
the password an agent passes lands in the same file.

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
