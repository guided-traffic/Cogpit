# Trust boundaries

Who and what Cogpit trusts, and the points where untrusted data crosses into trusted state. The
boundaries below exist in the problem itself; each row names the check the implementation must
have and the package that owns it, with a link where the MVP (2026-10-11) implements it.

## Roles

| Role | Trust | Why |
|---|---|---|
| Local user | trusted | Owns the desktop session, the kubeconfig files and the cluster identities Cogpit uses |
| Kubeconfig file | trusted only as much as its origin | A kubeconfig can name a program to run (`users[].user.exec`). Kubernetes warns: "Only use kubeconfig files from trusted sources. Using a specially-crafted kubeconfig file could result in malicious code execution or file exposure." ([kubeconfig docs](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/)) |
| Credential plugin | runs with the user's privileges | It is whatever program the kubeconfig names ([client-go credential plugins](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#client-go-credential-plugins)) |
| Kubernetes API server | authenticated, data untrusted | TLS proves which server answers; it does not make object names, annotations, events or logs safe to render, execute or hand to a model |
| Workload output | untrusted | Logs, exec and attach streams are written by containers, which anyone who can deploy to the cluster controls |
| A proxy the user names (a cluster's, and the Helm repositories' downloads', 2026-10-13) | trusted to forward, not to read | The user picks it in the window. Where the server's TLS is verified it sees the server's name and the traffic's timing and size; where it is not (`insecure-skip-tls-verify`, plain http) it reads everything, credentials included ([H-34](#h-34-the-plugin-approval-shows-the-kubeconfigs-proxy-not-cogpits)). The Helm downloads are https only and their certificate check is never switched off (Lens's "Allow untrusted certificate authorities" is not built: a question for the owner) |
| Helm repositories and charts | untrusted | Remote content; rendering a chart runs its templates in Cogpit's process through the Helm SDK |
| External agents over MCP | as trusted as the local user who connected them; decisions are a model's | [agents and MCP](agents-and-mcp.md) |
| Built-in agent and its LLM endpoint | the endpoint's output is untrusted | [agents and MCP](agents-and-mcp.md) |
| Release host and update manifest | trusted after signature verification | [updates](updates.md) |
| kubectl release host (`dl.k8s.io`) or the mirror set in Preferences | untrusted; a binary is trusted after the Kubernetes release's signature on it verified | A downloaded `kubectl` runs whenever the user types `kubectl` in a terminal of Cogpit's, with the user's privileges and credentials ([D-3](../developer/decisions.md#d-3-helm-and-kubectl-binaries)) |

```text
UNTRUSTED                                   BOUNDARY                          TRUSTED STATE
  kubeconfig files (exec, auth-provider) ─► parse + exec policy ────────────► connection settings
  proxy URL the user types (window only) ─► URL check, password ────────────► the connection's proxy
                                             redacted outside the file
  API responses, watch events            ─► schema-tolerant decoding, ───────► cache, view-models
                                             size limits, bounded walks
  log lines, exec/attach output          ─► terminal emulator with ─────────► terminal and log views
                                             filtered escape sequences
  YAML typed by the user                 ─► parse, diff, dry-run ───────────► apply request
  image file picked as a cluster icon    ─► bounded decode, re-encode, ──────► kept icon copy
                                             SVG checked
  Helm repository index and charts       ─► Helm SDK: no plugins, ──────────► release operations
                                             no post-renderers
  MCP requests, LLM tool calls           ─► token, origin, schema, ─────────► dispatched commands
                                             access review, confirmation
  update manifest and artifact           ─► signature, checksum, ───────────► installed app
                                             OS code signature
  kubectl release (dl.k8s.io or mirror)  ─► GUI click, https, size bound, ──► first on PATH in
                                             SHA-256, sigstore signature        Cogpit's terminals
```

## Boundaries and required checks

| Boundary | Planned package | Required checks |
|---|---|---|
| Kubeconfig to connection | `internal/kubeconfig`, `internal/cluster` | Parse without executing; run an exec plugin only when the user connects to that context, with an argument vector; show the command before the first run of a plugin from a newly added file; never follow `exec` from a kubeconfig the user did not add. Built 2026-10-12 for added and synced kubeconfigs ([H-1](#h-1-kubeconfig-files-added-by-sync-folders)): each source file is read on its own, never merged with another (a dropped file cannot point a context at the user's real credentials); a context whose user has an `exec` section, or names a `tokenFile` outside the kubeconfig's folder ([H-24](#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token)), connects only after the user allowed exactly that plugin or file in the GUI, checked by `cluster.Connect` before any client exists, from the very `rest.Config` the client would run ([`exec.go`](../../internal/kubeconfig/exec.go), `TestConnectAsksTheExecPolicyBeforeThePluginRuns`, `TestExecPluginRunsOnlyOnceAllowed`, `TestKindExecPluginAfterApproval`); the default kubeconfig (`--kubeconfig`, `$KUBECONFIG`, `~/.kube/config`) keeps kubectl's behaviour |
| Proxy setting to connection | `internal/engine`, `internal/cluster` | Built 2026-10-13 ([`engine/proxy.go`](../../internal/engine/proxy.go), [`cluster/proxy.go`](../../internal/cluster/proxy.go)): a proxy is set only by the user in the window (`ErrProxyGUIOnly` for agents and the CLI, who may clear it); it must be an `http`, `https` or `socks5` URL with a host and nothing but an optional port and `user:password@`; it applies to its own context's connection only, at the next connect, through `rest.Config.Proxy`; its password stays in the settings file and is `•••` everywhere else ([credentials](credentials.md#where-they-live-inside-cogpit)); the credential plugin's approval does not know it ([H-34](#h-34-the-plugin-approval-shows-the-kubeconfigs-proxy-not-cogpits)) |
| API response to cache | `internal/cache`, `internal/model` | Tolerate unknown and missing fields; cap object and list sizes; bound owner-reference walks |
| Cluster text to UI | `internal/model`, `internal/ui/fyneui` | Object names, labels, annotations and events are text, never markup; strip control characters before display |
| Container output to terminal | `internal/terminal`, `internal/ui/fyneui/termview` | Drop or confirm OSC 52 (clipboard write), window-title and window-manipulation sequences ([xterm control sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)); cap scrollback; cap line length. The MVP drops every OSC, DCS, APC, PM and SOS string, media copy and window operations, clamps large counts, and strips escapes from log lines, which are capped at 64 KiB per line and 10,000 lines ([`termview/sanitize.go`](../../internal/ui/fyneui/termview/sanitize.go), fuzzed; [`termview/logs.go`](../../internal/ui/fyneui/termview/logs.go)); see [H-19](#h-19-the-terminal-widget-trusts-its-input) |
| Editor to cluster | `internal/engine` | Show a diff and a server-side dry-run before apply; reject documents whose kind, name or namespace changed without saying so |
| Port-forward to local socket | `internal/cluster` | Bind `127.0.0.1` only; show which local port maps to which pod. Built 2026-10-11: `ListenLoopback` binds `tcp4` 127.0.0.1 and refuses any listener that ends up elsewhere ([`portforward.go`](../../internal/cluster/portforward.go), `TestListenLoopbackBindsIPv4LoopbackOnly`; the engine's tests try `::1` and every non-loopback interface address and expect refusal); the Port Forwarding view and a forward's drawer show the address, the pod and its port; a pod's error text is cut to one line of at most 512 bytes without control characters before it reaches the GUI or an agent; the forward's URL names `127.0.0.1`, not `localhost`, which macOS resolves to `::1` first, where another process may listen (`TestForwardURLReachesTheForward`), and the GUI opens only such URLs; a forward whose connection fails for a reason waiting does not cure (404, 403, 401) fails at once and closes its listener, so no port stays bound for a forward that cannot work (D-20, `TestPortForwardFailsAtOncePermanentErrors`) |
| Image file to cluster icon | `internal/engine` | Built 2026-10-13 ([`clustersettings_icon.go`](../../internal/engine/clustersettings_icon.go), `TestCheckIcon`, `TestClusterIconUpload`): only the user's upload in the window reads a file (agents and the CLI are refused, [agents and MCP](agents-and-mcp.md)); at most 1 MiB read; a PNG or JPEG's size is checked from its header before any pixel is decoded (at most 8,192 pixels a side and 16 megapixels: a small file cannot expand into a huge image), then decoded with Go's own decoders and kept as a PNG of at most 256 pixels a side that Cogpit wrote; an SVG (at most 256 KiB) must be one well-formed `<svg>` document of bounded depth and token count, without a DOCTYPE or other directive (no entities), processing instructions, `<script>`, `<foreignObject>` or animation elements, event handler attributes, or references to anything but a fragment of itself; the copy lives in `<user config dir>/Cogpit/cluster-icons/` under its SHA-256, owner-only, and is checked again before the GUI draws it, so a file someone else put there is never decoded as is |
| Chart to release | `internal/helm` | The Helm SDK renders in-process; Helm plugins and post-renderers (which run external programs) are not supported. Built 2026-10-13 ([architecture §8](../developer/architecture.md#8-helm-internalhelm)): repositories are public https only, redirects followed to https only, the downloads through the Preferences' proxy, else the environment's, never without the certificate check (2026-10-13); an index (32 MiB) and a chart archive (16 MiB) are bounded and parsed by the SDK (`FuzzLoadIndex`); an archive must match the SHA-256 digest its index lists, when it lists one, and hold the chart and version the index names (`TestRepositoryRefusesWhatDoesNotFit`); a library chart and a chart missing its dependencies are refused; DNS lookups in templates stay off; the install is previewed with the SDK's server-side dry run, which writes nothing (`TestKindHelmInstall`). An upgrade to a version a repository lists (2026-10-13) downloads the chart through the same checks, before anything is recorded, and refuses a chart whose name is not the release's (`TestUpgradeToARepositoryChart`, `TestHelmUpgradeRepositoryRefusals`); it has no preview. Provenance files are not checked ([H-30](#h-30-chart-provenance-is-not-verified)) |
| Request to command | `internal/engine` | Params validated against the schema; `enabled` and access review before `run`; confirmation by origin ([agents and MCP](agents-and-mcp.md)). Built: the schema check refuses unknown fields, then `Command.Enabled` asks the connected user's cached access reviews about every declared verb whenever the parameters name an object, and a known denial refuses the command for every origin ("not permitted: you may not …"), an unknown answer passes and the API server decides; then the confirmation ([`engine.go`](../../internal/engine/engine.go) `Dispatch`, [`access.go`](../../internal/engine/access.go)); see [H-21](#h-21-access-reviews-guide-they-do-not-guard) |
| MCP request to engine | `internal/mcp` | Loopback only, bearer token, `Host` and `Origin` checks: [`internal/mcp/http.go`](../../internal/mcp/http.go), `TestHTTPGuard`, `TestNewRefusesNonLoopbackAddresses` |
| Cluster data to the LLM endpoint | `internal/agent` | Secret values withheld by default; what is sent is shown |
| Download to installed app | `internal/update` | Signed manifest, checksum, OS code signature, no downgrade ([updates](updates.md)) |
| kubectl download to a terminal's `PATH` | `internal/kubectl`, `internal/engine`, `internal/terminal` | Built 2026-10-13 ([D-3](../developer/decisions.md#d-3-helm-and-kubectl-binaries)): only the user's click in the window downloads (`kubectl.download` refuses agents and the CLI before anything is fetched, `TestKubectlDownloadCommand`), and only the user sets the path to kubectl, the directory for binaries and the mirror (`ErrKubectlGUIOnly`); the mirror is an https URL without user information, query or fragment, and redirects are followed to https only (`TestCheckMirror`, `TestDownloadHTTPSOnly`); the `.sha256`, `.sig` and `.cert` files are read first (16 KiB each at most) and the signature is checked over the published digest before the binary is asked for: the signing certificate must chain to the sigstore certificate authority Cogpit embeds ([`sigstore_fulcio.pem`](../../internal/kubectl/sigstore_fulcio.pem)) at the time it was issued, be for code signing, and name `krel-staging@k8s-releng-prod.iam.gserviceaccount.com` with the OIDC issuer `https://accounts.google.com`, and its key must have signed the digest ([`verify.go`](../../internal/kubectl/verify.go); `TestRealReleaseSignature` verifies kubectl 1.36.1's published signature offline, `TestVerifierRefuses`, `TestDownloadRefusesBadSignatures`); the binary (at most 128 MiB, `TestDownloadSizeBound`) goes into a temporary file beside its place, must have the published digest (`TestDownloadRefusesTamperedBinary`), and is made 0755 and renamed into `<directory for binaries>/<version>/kubectl` (directories 0700); nothing is left on failure. It goes first on `PATH` in Cogpit's terminals only, as a link in the terminal's own temporary directory, never in a shell profile or a global location (`TestStartLocalKubectlFirst`, `TestZshKubectlFirstAfterTheProfile`). Not checked: sigstore's transparency log ([H-35](#h-35-kubectls-signature-is-checked-without-sigstores-transparency-log)); a profile can still reorder `PATH` ([H-36](#h-36-a-shells-profile-can-put-another-kubectl-first)) |

## What this does not cover

### H-1: Kubeconfig files added by sync folders

Lens watches `~/.kube/` and any files and folders listed under Preferences › Kubernetes ›
Kubeconfig Syncs, and adds every kubeconfig that appears there; captured 2026-10-12, it runs a
synced file's credential plugin on connect without asking
([add a local cluster](https://docs.lenshq.io/k8slens/getting-started/add-clusters/add-local-cluster/),
[preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/)). Cogpit adds kubeconfig
files and sync folders since 2026-10-12 (and `~/.kube` by default, below), and a file dropped into a synced
folder (a download, a shared drive) becomes a set of contexts; hidden files too, as in Lens. The check of the boundary table is
built as follows:

- **When.** For a context of an added or synced kubeconfig whose user has an `exec`
  section (or a `tokenFile` outside the folder, H-24), the picker shows the plugin before it connects, when there is no approval or the
  approval no longer matches. The engine refuses the connect for every origin on its own
  ("the kubeconfig X runs the program Y to log in, which you have not allowed yet"); a `--context`
  at start-up lands on that refusal and opens the picker.
- **What is shown.** The program as client-go will run it (a relative path resolved against the
  kubeconfig's folder), where it is found on `PATH` now or that it is not found, each argument on
  its own line, the environment variables by name only (a warning when they set `PATH`, `LD_*` or
  `DYLD_*`), the server the credential goes to, **how that server is reached and trusted** (since
  2026-10-12, review of the fourth run, R-11: certificate checks off, a proxy URL without user
  information, a certificate authority the kubeconfig brings, embedded or as a file, and a TLS
  server name; each of the first three is a line in the warning colour above the rows, and a plain
  connection with the system's trust adds nothing), whether the program or the CA file lies inside
  the kubeconfig's folder with the content digest the approval binds, the kubeconfig file, the
  context and its source, and, for a changed plugin, that it changed since it was allowed. All of
  it is escaped for control characters, bidirectional overrides and invalid UTF-8, and capped in
  length (`TestApprovalDialogShowsProxyAndInsecure`, `TestApprovalDialogEscapesUntrustedText`).
  "Allow and Connect" arms after 750 ms; nothing has focus by default.
- **What is remembered.** One approval per file and context in `<user config dir>/Cogpit/exec-approvals.json`
  (0600, written atomically, never through a symlink), bound to a SHA-256 fingerprint (version 2
  since 2026-10-12; every approval of version 1 asks again once) of the file, the context, the
  user, the server URL, its CA data and file, the insecure flag, the TLS server name, the proxy
  URL, the plugin's API version, command, arguments and environment (names and values, in order),
  its interactive mode and `provideClusterInfo`, the token file when it needs approval
  ([H-24](#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token)), and, **for a program or
  a CA file that resolves to a file inside the kubeconfig's own folder (the place a writer of a
  synced folder can write), the SHA-256 of its content** (review of the fourth run, R-5, owner
  decision, [D-26](../developer/decisions.md#d-26-binding-a-credential-plugins-approval-to-its-program), "programs in the folder only"): replacing `./login.sh` or the CA file
  after the approval asks again, at the next listing and at connect time, where the files are read
  again right before the policy decides (`TestExecApprovalRefusedAfterPluginBinaryChanged`,
  `TestCAFileSwapAsksAgain`). A link in the folder is bound by its resolved target as well, so
  repointing it asks again; a program resolved elsewhere (`PATH`, `/usr/local/bin`, `/opt/homebrew`:
  `kubelogin`, `aws`, `gke-gcloud-auth-plugin`) stays bound by its path, so a tool update does
  not ask again (`TestProgramOutsideTheFolderIsBoundByPathOnly`). The hashing is bounded: regular
  files only, at most 32 MiB for a program and 1 MiB for a CA file; a larger or unreadable file
  inside the folder is bound by a note (its size and modification time for a large one), so a
  change still asks again. The approval names the fingerprint the dialog showed and is refused
  if the file changed meanwhile; an unreadable approvals file allows nothing; removing a source
  forgets its approvals.
  The GUI asks before removing one, naming the path and saying that only Cogpit's link goes and
  the files stay on disk, and a connection made from the source closes.
- **Asked once per file and plugin** (owner, 2026-10-12, [D-28](../developer/decisions.md#d-28-the-default-kubeconfig-as-sources-as-lens-has-it);
  only the asking is bundled). Contexts of one file whose logins are the same plugin (the same
  command, arguments, environment, API version, interactive mode, `provideClusterInfo`, program
  binding and, for D-17, token file: `kubeconfig.ExecPlugin.Identity`) are one dialog, which lists
  **every context that is not allowed yet, each with its own server and how that server is reached
  and trusted** (the R-11 warning lines per context, under its name), and one click allows them all.
  Nothing in what an approval binds got looser: the engine still stores **one approval per context
  with that context's own fingerprint**, which binds the server, its CA, proxy, TLS server name and
  insecure flag, so a changed server in one context asks again for that context only, even when
  its siblings are allowed. The approving command takes the `(context, fingerprint)` list the
  dialog showed, checks each against the file as it is now, and refuses the **whole** list, naming
  the contexts, if any changed or left meanwhile; nothing is stored then
  (`TestApprovalIsRefusedAsAWholeWhenOneContextChanged`). A context added to the file later is not
  in an earlier approval and asks on its own, or with the other new ones
  (`TestAContextAddedLaterAsksOnItsOwn`). What one click means is wider than before, and that is the
  point of the owner's decision: the user allows every listed context, each shown with its server
  and trust, in one step; with many contexts the dialog scrolls, and its buttons stay visible
  without the user having scrolled to the end of the list. Agents and the CLI still cannot approve,
  and read the grouped listing (`kubeconfig.pendingApprovals`) without arguments and environment
  values (`TestPendingApprovalsAreBundledAndApprovedByTheGUIOnly`).
- **Who.** Adding a file or a folder, removing a source and allowing a plugin are GUI
  only: the engine refuses them for the CLI, external agents and the built-in agent, whose
  confirmation dialog could not show the user an agent-chosen path for review
  (`TestKubeconfigCommandsAreGUIOnly`).
- **`cogpit-cli`** (built 2026-10-13) offers the same sources and honours the same approvals: both
  binaries build them in [`internal/app`](../../internal/app/app.go), and the CLI connects through
  the same `ExecPolicy`, so a plugin the user has not allowed in the window is refused before it
  runs (`TestGetRunsNoPluginTheUserHasNotAllowed`). It writes neither the sources nor the
  approvals, and without a configuration directory it refuses to connect, where the app falls back
  to merging the files as kubectl does (`TestNoConfigDirNoConnection`; Cogpit's default, not an
  owner decision). Context names and file names are escaped before they reach the terminal
  (`TestContextsListsTheSourcesWithoutSecrets`).

**`~/.kube` is synced by default, and so is everything else** (2026-10-12; Lens does the same,
captured; [D-28](../developer/decisions.md#d-28-the-default-kubeconfig-as-sources-as-lens-has-it), built
the same day). The folder is the user's own and Cogpit neither writes nor runs anything in it
unasked, but a folder like that collects files nobody chose to add (a kubeconfig saved or generated
into it by the user or a tool). Every file of it is therefore handled as the file of any sync
folder, **`config` included**: read on its own, never merged; a credential plugin runs only after the
user allowed it (the approval above, per file and context, forgotten when the folder is removed).
There is no longer a "default kubeconfig" under kubectl's rules: the files `$KUBECONFIG` lists and
Cogpit's `--kubeconfig` paths are sources of their own for the run, under the same approval, and
removable for the session only (their approvals are kept; a file that lies directly in `~/.kube` is
the folder's, so removing the folder hides it). Per case: a process that can write into `~/.kube`
could replace `config` and have kubectl run its plugin; in Cogpit it gets a dialog instead
(dormant, and narrower than before, when `config`'s plugin ran unasked); a kubeconfig that arrives in
the folder by accident becomes contexts, and its plugin waits for the user (the approval holds); its
`tokenFile` is H-24 below, live for these files as for every file. The cost, accepted by the owner:
a `config` with a plugin (EKS, GKE, AKS) asks, as any other file does, and again
when the plugin changes (once per file and plugin, listing its contexts, since the owner's decision of the same day, above). The user can remove the folder (Kubeconfig Sources), and the engine takes
the folder and the session files as options, so tests never read the real ones
(`TestNoDefaultFolderWithoutTheOption`, `TestConfigWithAPluginAsksForApprovalOnce`,
`TestDefaultKubeconfigIsOneFileOfTheSyncedFolder`).

What remains: kubectl in a terminal of such a context reads the source file again, so a
plugin changed after it was allowed runs there without being asked (the user's own kubectl
command); and a context without a plugin connects without asking only when it names no token file
outside its folder ([H-24](#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token)).
Content binding is narrower than "everything the plugin runs", by decision (owner, 2026-10-12):

- a program found outside the kubeconfig's folder is bound by its path, not its content: whoever
  can write to a `PATH` directory or to `/usr/local/bin` already controls the user's account, and
  binding them would ask again after every tool update;
- only the program file itself is hashed: a script in the folder that sources or imports a sibling
  file, or an interpreter that loads libraries, is bound through the script only, and so is a
  binary's own configuration;
- "the folder" is the directory of the kubeconfig file, its subdirectories included; for a file
  the user added from a broad directory (the home directory, Downloads) that is every file below
  it, not only what a sync writer controls;
- the files are read right before the policy decides and client-go starts the program a moment
  later: a writer who swaps the program inside that window, and swaps it back, is not caught (a
  file is not locked; the window is microseconds and needs write access to the folder at that
  moment);
- the client certificate and key files a kubeconfig names are bound by path only (the key is never
  sent; the certificate only in the TLS handshake).

A link in the folder that points to a file outside is bound by its target path; a tool manager that
repoints it (a Homebrew upgrade of a linked `aws`) asks once again.

### H-34: The plugin approval shows the kubeconfig's proxy, not Cogpit's

The approval of a credential plugin binds, and its dialog shows, the kubeconfig's own `proxy-url`
(R-11). Since 2026-10-13 a cluster's settings in Cogpit can name a proxy of their own, which replaces
the kubeconfig's for Cogpit's connections ([credentials](credentials.md#where-they-live-inside-cogpit)):
the credential the approved plugin returns then travels through a proxy the dialog did not name, and
changing that proxy does not ask for the approval again. What keeps it narrow today: only the user
sets a proxy, in the window (agents and the CLI are refused, [agents and MCP](agents-and-mcp.md));
with the server's TLS verified a proxy sees the server's name and the traffic's timing and size, not
the token. It is live for a context with a plugin whose user also set a proxy, and real where the
server's certificate is not verified (`insecure-skip-tls-verify`, a plain-http server): there the
proxy reads the bearer token. Not designed: whether the dialog shows Cogpit's proxy beside the
kubeconfig's, and whether the approval's fingerprint binds it.

### H-30: Chart provenance is not verified

The digest Cogpit checks a chart archive against comes from the repository's own `index.yaml`,
served by the same host as the archive (2026-10-13). It catches a corrupted download or an index
and archive that do not belong together, not a repository that serves a malicious chart with a
matching digest. Helm's provenance files (`<chart>-<version>.tgz.prov`, an OpenPGP signature over
the chart's digest, `helm install --verify`) are not downloaded or checked, and Cogpit keeps no
keyring. A chart then runs as templates in Cogpit's process (no plugins, post-renderers or DNS
lookups, but `lookup` reads what the user may read) and its objects are created with the user's
identity after the preview and, for an agent, the user's acceptance; an upgrade to a repository's
version (2026-10-13) changes a release's objects to such a chart the same way, without a preview.
Which keys to trust and where
they would live is not designed.

### H-35: kubectl's signature is checked without sigstore's transparency log

The Kubernetes release signs kubectl with sigstore's keyless signing: a key made for one signing,
certified by sigstore's certificate authority (Fulcio) for ten minutes for the release's identity,
and the signature recorded in sigstore's public transparency log (Rekor). Cogpit checks the
certificate's chain to the authority it embeds, the identity, the issuer and the signature over the
published SHA-256 (built 2026-10-13, the standard library only), but not the log: the `.sig` and
`.cert` files `dl.k8s.io` publishes carry no log entry, and finding one needs a request to
`rekor.sigstore.dev` (a host no decision covers yet, [credentials](credentials.md#where-credentials-must-never-end-up))
and a check of its proof, which sigstore-go does (a large dependency Cogpit does not have). Per case:

- **A compromised mirror or CDN** (the `dl.k8s.io` redirect, or a mirror set in Preferences) cannot
  forge the signature: covered, since every file it serves is checked against the release identity.
- **A leaked signing key.** The key is made for one signing and discarded; without the log, Cogpit
  cannot tell that a signature was made while its certificate was valid, so whoever held such a key
  could sign another binary later and Cogpit would accept it. Dormant: no such leak is known, and it
  needs the Kubernetes release's infrastructure.
- **A misissued certificate.** A certificate Fulcio issued for the release's identity to someone
  else would verify; the certificate transparency timestamps it carries are not checked either.
  Dormant, and it needs a compromise of Fulcio or of the Google service account.
- **The embedded authority.** The root and the intermediate are valid until 2031-10-05; when
  sigstore rotates the intermediate, new releases stop verifying until Cogpit ships the new one
  (fails closed: the download is refused, nothing runs).

### H-36: A shell's profile can put another kubectl first

Cogpit puts the downloaded kubectl first on the terminal's `PATH` in the environment it starts the
login shell with, and the profile runs after that. Verified 2026-10-13 on macOS 26.6:
`/usr/libexec/path_helper`, which `/etc/zprofile` and `/etc/profile` run, moves an inherited entry
behind `/etc/paths` and `/etc/paths.d` (a directory put first came out last). Per shell:

- **zsh** (macOS's default): Cogpit starts it with a `ZDOTDIR` of its own whose `.zshenv` gives zsh
  the user's `ZDOTDIR` back, reads the user's `.zshenv` and puts the directory first again before the
  first prompt, after every profile ran (`TestZshKubectlFirstAfterTheProfile`, with a `.zprofile` that
  prepends another kubectl). What remains: a `.zshrc` that empties `precmd_functions`, and a tool that
  changes `PATH` at a later prompt (direnv, asdf, a typed `export`).
- **bash, sh and other shells**: only the environment's order. Live on macOS whenever `/etc/paths`
  or a profile's directory holds a kubectl (Docker Desktop's `/usr/local/bin/kubectl`, Homebrew's),
  and on Linux distributions whose `/etc/profile` sets `PATH` anew. The terminal's line names the
  version Cogpit put first; `kubectl version --client` in the terminal shows which one runs.

### H-2: Local processes and forwarded ports

A forwarded port on `127.0.0.1` is reachable by every process of every local user on the machine,
with the pod's identity behind it. Binding to loopback limits it to the host, not to Cogpit. No
per-connection authentication is possible for arbitrary TCP protocols. As built (2026-10-11): a
forward lives from Start until Stop, Delete, switching context or quitting, and is not restored on
the next start; agents need the user's acceptance to start or edit one (`Mutating`, D-2) unless
agent writes are allowed for the context, and may open, stop and delete forwards freely (`UI`);
an agent that learns the URL can connect like any other local process. The URL names `127.0.0.1`
because `localhost` may resolve to `::1`, where Cogpit does not listen and another process may.
Whether another process can bind the same port on Windows (`SO_EXCLUSIVEADDRUSE`) is not
verified; kubectl has the same exposure.

### H-19: The terminal widget trusts its input

The terminal widget, `github.com/fyne-io/terminal` (Cogpit's patched copy on upstream
`46cd12724ff3` since 2026-10-12, [`third_party/fyne-terminal`](../../third_party/fyne-terminal/PATCHES.md)),
froze or crashed the whole app on output a container can send, on the main goroutine where nothing
recovers, and changed Cogpit's working directory on OSC 7; the escape filter stays in front of it.
Fuzzed on its own as decided ([D-14](../developer/decisions.md#d-14-terminal-widget), 2026-10-12):
`FuzzHandleOutput` in the copy and `FuzzRawWidget` through the public API in termview, each failing
on a panic or on input not done within 30 s, seeded with every known hazard and recordings of vim,
less, pico and screen. Found and fixed, each with a regression test (PATCHES.md lists them): a CSI
whose final character is not ASCII and a set or reset mode without `?` (both panicked; ANSI mode
20 is new-line mode again); a debug-mode panic on `ESC [ > c`; repeat, delete, insert and scroll
counts that looped once per count (a hang at the largest count); insert lines copying instead of
inserting; OSC, APC and DCS strings and long CSI parameters built quadratically (1 MiB took about
60 s on the reading goroutine; now linear and capped at 1 MiB); an APC ended by ST that swallowed
all later output; and five data races moved onto one goroutine (start-up configuration and the
input writer, the selection, the character sets, media copy, the bell). The runs were short: about
1.0 million executions in 12 minutes on the copy and 165,000 in 10 minutes through termview, with
no finding left; longer runs with the corrected harness are open. What remains: inputs the short
runs did not reach; known and unpatched (a DCS ended by ST swallows the next character, an ESC at
byte 5000 of a read is not recognised, cursor movement overflows at the largest counts, media copy
buffers without a bound, blink tickers outlive a closed terminal, `run()` reads `content` on the
reading goroutine, races in `RunLocalShell`, which Cogpit does not call); its ANSI colours are
fixed, not theme tokens; and the copy must be kept in step with upstream until upstream takes the
patches, whose series is not prepared yet.

### H-21: Access reviews guide, they do not guard

Cogpit greys and refuses what the connected user's access reviews deny, and hides the kinds they
may not list. The reviews are guidance for what Cogpit shows and offers, not an authorization
boundary: their answers are kept until a refresh or the next connect, so a role changed meanwhile
is not seen; a rules review can over-report a cluster-scoped kind that only a RoleBinding names,
which is why Cogpit confirms such answers with an access review; and Kubernetes' documentation
advises against rules reviews for authorization decisions. An unknown answer offers the command,
and the API server stays the authority for every request.

### H-24: A dropped kubeconfig can send a local file as its token

A kubeconfig names files as well as programs: `tokenFile` (and a user's `client-certificate`,
`client-key` and the cluster's `certificate-authority`). A kubeconfig dropped into a synced folder
or added by the user can pair `tokenFile: <any file the user can read>` with a server of its own;
when the user connects to that context, client-go reads the file and sends its content to that
server as a bearer token. Kubernetes' documentation names this ("file exposure",
[kubeconfig docs](https://kubernetes.io/docs/concepts/configuration/organize-cluster-access-kubeconfig/)).

**Built 2026-10-12** ([D-17](../developer/decisions.md#d-17-token-files-named-by-added-kubeconfigs),
owner, option a): a context of any kubeconfig file (since D-28, 2026-10-12, `config` and the
files `$KUBECONFIG` names included: there is no unasked default kubeconfig left) whose user names a `tokenFile`
outside the kubeconfig's folder connects only after the user allowed it, in the dialog of the
credential plugin ([H-1](#h-1-kubeconfig-files-added-by-sync-folders)), with the same approval
store and the same GUI-only rule. The dialog is titled "Allow this file as the login token?" and
names the file, the server its content goes to, how that server is reached and trusted, the
kubeconfig and the context; a context that also runs a plugin shows both in one dialog and one
approval. The approval's fingerprint binds the token file's path, the server and its trust (a
changed path or a changed server asks again, `TestTokenFileOutsideTheFolderIsAskedFor`,
`TestTokenFileOutsideTheFolderNeedsApproval`); the file's content is not bound, so a rotated token
does not ask again. `cluster.Connect` asks the policy **before** client-go builds its configuration,
because that is what reads the token file: nothing is read from a file outside the folder before
the user allowed it. A token file resolved inside the kubeconfig's folder comes from the same
source as the kubeconfig and needs no approval; a link in the folder that points outside counts as
outside (`TestTokenFileLinkedOutOfTheFolderNeedsApproval`), and the engine's refusal names the
file and the server, never the content. Since D-28 the gap no longer has a dormant corner: it
holds for `~/.kube/config` and the `$KUBECONFIG` files as for every file.

What remains, per case:

- **A broad folder.** "The kubeconfig's folder" is the directory of the file, subdirectories
  included. For a file the user added from the home directory, Downloads or a shared drive's root,
  a `tokenFile` anywhere below that directory (`~/.ssh/id_ed25519` for a kubeconfig in `~`) counts
  as inside and is not asked for. Live for added files in such directories, and for a sync folder
  only as far as its own writer can already write.
- **A link repointed after the connect.** client-go re-reads a token file about once a minute for
  the life of the connection. A link in the folder that resolved inside when the user connected and
  is repointed outside afterwards is read at the next refresh without a new question; the next
  connect asks. Live while a connection to such a context is open and the folder is writable by
  someone else.
- **The window between the check and the read.** The policy decides, then client-go opens the
  path; a swap to a link outside in that window is not caught (microseconds, write access to the
  folder at that moment).
- **Other files.** A user's `client-certificate` and `client-key` and the cluster's
  `certificate-authority` are bound by path only: the key is never sent, a certificate only in the
  TLS handshake. A CA file inside the folder is bound by content ([H-1](#h-1-kubeconfig-files-added-by-sync-folders)).
