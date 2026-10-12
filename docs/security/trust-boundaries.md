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
| Helm repositories and charts | untrusted | Remote content; rendering a chart runs its templates in Cogpit's process through the Helm SDK |
| External agents over MCP | as trusted as the local user who connected them; decisions are a model's | [agents and MCP](agents-and-mcp.md) |
| Built-in agent and its LLM endpoint | the endpoint's output is untrusted | [agents and MCP](agents-and-mcp.md) |
| Release host and update manifest | trusted after signature verification | [updates](updates.md) |

```text
UNTRUSTED                                   BOUNDARY                          TRUSTED STATE
  kubeconfig files (exec, auth-provider) ─► parse + exec policy ────────────► connection settings
  API responses, watch events            ─► schema-tolerant decoding, ───────► cache, view-models
                                             size limits, bounded walks
  log lines, exec/attach output          ─► terminal emulator with ─────────► terminal and log views
                                             filtered escape sequences
  YAML typed by the user                 ─► parse, diff, dry-run ───────────► apply request
  Helm repository index and charts       ─► Helm SDK: no plugins, ──────────► release operations
                                             no post-renderers
  MCP requests, LLM tool calls           ─► token, origin, schema, ─────────► dispatched commands
                                             access review, confirmation
  update manifest and artifact           ─► signature, checksum, ───────────► installed app
                                             OS code signature
```

## Boundaries and required checks

| Boundary | Planned package | Required checks |
|---|---|---|
| Kubeconfig to connection | `internal/kubeconfig`, `internal/cluster` | Parse without executing; run an exec plugin only when the user connects to that context, with an argument vector; show the command before the first run of a plugin from a newly added file; never follow `exec` from a kubeconfig the user did not add. Built 2026-10-12 for added and synced kubeconfigs ([H-1](#h-1-kubeconfig-files-added-by-sync-folders)): each source file is read on its own, never merged with another (a dropped file cannot point a context at the user's real credentials); a context whose user has an `exec` section, or names a `tokenFile` outside the kubeconfig's folder ([H-24](#h-24-a-dropped-kubeconfig-can-send-a-local-file-as-its-token)), connects only after the user allowed exactly that plugin or file in the GUI, checked by `cluster.Connect` before any client exists, from the very `rest.Config` the client would run ([`exec.go`](../../internal/kubeconfig/exec.go), `TestConnectAsksTheExecPolicyBeforeThePluginRuns`, `TestExecPluginRunsOnlyOnceAllowed`, `TestKindExecPluginAfterApproval`); the default kubeconfig (`--kubeconfig`, `$KUBECONFIG`, `~/.kube/config`) keeps kubectl's behaviour |
| API response to cache | `internal/cache`, `internal/model` | Tolerate unknown and missing fields; cap object and list sizes; bound owner-reference walks |
| Cluster text to UI | `internal/model`, `internal/ui/fyneui` | Object names, labels, annotations and events are text, never markup; strip control characters before display |
| Container output to terminal | `internal/terminal`, `internal/ui/fyneui/termview` | Drop or confirm OSC 52 (clipboard write), window-title and window-manipulation sequences ([xterm control sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)); cap scrollback; cap line length. The MVP drops every OSC, DCS, APC, PM and SOS string, media copy and window operations, clamps large counts, and strips escapes from log lines, which are capped at 64 KiB per line and 10,000 lines ([`termview/sanitize.go`](../../internal/ui/fyneui/termview/sanitize.go), fuzzed; [`termview/logs.go`](../../internal/ui/fyneui/termview/logs.go)); see [H-19](#h-19-the-terminal-widget-trusts-its-input) |
| Editor to cluster | `internal/engine` | Show a diff and a server-side dry-run before apply; reject documents whose kind, name or namespace changed without saying so |
| Port-forward to local socket | `internal/cluster` | Bind `127.0.0.1` only; show which local port maps to which pod. Built 2026-10-11: `ListenLoopback` binds `tcp4` 127.0.0.1 and refuses any listener that ends up elsewhere ([`portforward.go`](../../internal/cluster/portforward.go), `TestListenLoopbackBindsIPv4LoopbackOnly`; the engine's tests try `::1` and every non-loopback interface address and expect refusal); the Port Forwarding view and a forward's drawer show the address, the pod and its port; a pod's error text is cut to one line of at most 512 bytes without control characters before it reaches the GUI or an agent; the forward's URL names `127.0.0.1`, not `localhost`, which macOS resolves to `::1` first, where another process may listen (`TestForwardURLReachesTheForward`), and the GUI opens only such URLs |
| Chart to release | `internal/helm` | The Helm SDK renders in-process; Helm plugins and post-renderers (which run external programs) are not supported; chart provenance is verified where the repository provides it |
| Request to command | `internal/engine` | Params validated against the schema; `enabled` and access review before `run`; confirmation by origin ([agents and MCP](agents-and-mcp.md)). Built: the schema check refuses unknown fields, then `Command.Enabled` asks the connected user's cached access reviews about every declared verb whenever the parameters name an object, and a known denial refuses the command for every origin ("not permitted: you may not …"), an unknown answer passes and the API server decides; then the confirmation ([`engine.go`](../../internal/engine/engine.go) `Dispatch`, [`access.go`](../../internal/engine/access.go)); see [H-21](#h-21-access-reviews-guide-they-do-not-guard) |
| MCP request to engine | `internal/mcp` | Loopback only, bearer token, `Host` and `Origin` checks: [`internal/mcp/http.go`](../../internal/mcp/http.go), `TestHTTPGuard`, `TestNewRefusesNonLoopbackAddresses` |
| Cluster data to the LLM endpoint | `internal/agent` | Secret values withheld by default; what is sent is shown |
| Download to installed app | `internal/update` | Signed manifest, checksum, OS code signature, no downgrade ([updates](updates.md)) |

## What this does not cover

### H-1: Kubeconfig files added by sync folders

Lens watches `~/.kube/` and any files and folders listed under Preferences › Kubernetes ›
Kubeconfig Syncs, and adds every kubeconfig that appears there; captured 2026-10-12, it runs a
synced file's credential plugin on connect without asking
([add a local cluster](https://docs.lenshq.io/k8slens/getting-started/add-clusters/add-local-cluster/),
[preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/)). Cogpit adds kubeconfig
files and sync folders since 2026-10-12, and a file dropped into a synced
folder (a download, a shared drive) becomes a set of contexts. The check of the boundary table is
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
- **Who.** Adding a file or a folder, removing a source and allowing a plugin are GUI
  only: the engine refuses them for the CLI, external agents and the built-in agent, whose
  confirmation dialog could not show the user an agent-chosen path for review
  (`TestKubeconfigCommandsAreGUIOnly`).

What remains: `~/.kube/` is not watched without setup (Lens does; Cogpit reads only the default
kubeconfig there); kubectl in a terminal of such a context reads the source file again, so a
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
owner, option a): a context of an added or synced kubeconfig whose user names a `tokenFile`
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
file and the server, never the content. The default kubeconfig keeps kubectl's behaviour, so the
gap is dormant there.

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
