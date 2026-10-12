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
| Legacy auth provider `oidc` | the kubeconfig | Refreshed tokens go through a persister Cogpit supplies; see [H-4](#h-4-auth-provider-token-refresh) |
| Legacy auth providers `azure` and `gcp` | the kubeconfig | client-go 0.37.1 rejects `azure` and points to kubelogin; the user replaces them with exec plugins |

Cogpit adds no cluster credential source of its own: no account, no login, no cloud service.

## Cogpit's own secrets

| Secret | Lives in | Rule |
|---|---|---|
| MCP bearer token | `<user config dir>/Cogpit/mcp-token` (macOS: `~/Library/Application Support/Cogpit/mcp-token`), file 0600 in a 0700 directory, written atomically, never through a symlink, removed on exit ([`internal/mcp/token.go`](../../internal/mcp/token.go), `TestTokenFileLifecycle`, `TestStartRefusesSymlinks`) | 32 random bytes, new on every start; printed to the terminal that started Cogpit inside the `claude mcp add` command, never logged (`TestTokenNeverLogged`) ([agents and MCP](agents-and-mcp.md)) |
| LLM API key (built-in agent) | the operating system's keychain (macOS Keychain first) | never written to a file, never logged, sent only to the configured endpoint |
| Update public key | compiled into the app | not secret; its private counterpart never touches a user's machine ([updates](updates.md)) |

## Where they live inside Cogpit

- **Kubeconfig files stay where they are.** Cogpit stores the paths the user added, not copies
  (built 2026-10-12: `<user config dir>/Cogpit/kubeconfigs.json`, 0600, lists the added files and
  sync folders; Cogpit never writes the files themselves, `TestAddedFilesAreNeverWritten`).
  The MVP switches off client-go's two implicit writes: the migration of a legacy
  `~/.kube/.kubeconfig` into `~/.kube/config`, and the persister that writes refreshed `oidc` tokens
  back into the file ([`cluster.go`](../../internal/cluster/cluster.go), `memoryPersister`).
- **Pasted kubeconfigs** ("add cluster from text") are the one exception: built 2026-10-12, they
  are written byte for byte to `<user config dir>/Cogpit/kubeconfigs/pasted-<16 hex>.yaml`, mode
  `0600` in a `0700` directory (tightened when found wider), never through a symlink, and deleted
  when the user removes the source, only if it is a regular file in that directory
  (`TestPasteStoresItOwnerOnly`, `TestRemoveDeletesOnlyCogpitsOwnCopy`). Windows gets the file
  modes only, not an owner-only ACL yet. The pasted text never reaches a log, an error, a toast or
  the audit log (an agent's attempt is audited by the size of its parameters); agents cannot paste
  ([trust boundaries › H-1](trust-boundaries.md#h-1-kubeconfig-files-added-by-sync-folders)).
- **Exec plugin approvals** live in `<user config dir>/Cogpit/exec-approvals.json` (0600): the
  fingerprint of what each allowed plugin runs and where its credential goes, no credential.
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

## Where credentials must never end up

| Place | Rule |
|---|---|
| Application log | No tokens, keys, certificates, exec-plugin output or Secret data, at any log level |
| Error messages and toasts | Same rule; errors name the context, never the credential. client-go's own errors can echo a kubeconfig's `users` entries; `kubeconfig.SafeError` ([`kubeconfig.go`](../../internal/kubeconfig/kubeconfig.go)) strips them, and connect errors are tested not to contain a known token |
| Crash reports | Cogpit has no crash upload. A local crash report follows the log rule |
| Telemetry | Cogpit sends none. Any network call to something other than a cluster, a Helm repository, the configured LLM endpoint or the release channel needs a decision entry in [`decisions.md`](../developer/decisions.md) |
| The LLM endpoint | Secret values only when the user allows it for that request ([H-13](agents-and-mcp.md#h-13-cluster-data-leaves-for-the-llm-endpoint)) |
| Screenshots taken by automation | Secret values stay masked unless the user revealed them in the GUI |
| Clipboard | Only on an explicit copy action |

## Secret objects

Kubernetes Secrets are data the user's identity may read, not Cogpit's own credentials, but they
are handled with the same care: the list holds metadata only, an opened Secret's values are
masked until the user reveals one field, decoded values are never logged, and editing or creating
a Secret from YAML is refused (its values are redacted there). As built (2026-10-11): when the
drawer closes or moves, the engine zeroes the bytes it kept of the opened Secret; the GET's
response and a revealed value's string wait for the garbage collector, so the wipe is best effort
([privilege footprint › H-8](privilege-footprint.md#h-8-secret-data-arrives-with-the-list)).

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

### H-20: Shell profiles can override the terminal's kubeconfig

The terminal starts a login shell, so the user's shell profile runs after Cogpit set `KUBECONFIG`.
A profile that exports `KUBECONFIG` (common with tools that manage several kubeconfig files)
replaces Cogpit's value, and `kubectl` in that terminal then talks to whatever context the profile
selects, not to the cluster whose view opened the terminal. Lens has the same behaviour. Cogpit
does not detect it; a check after the profile ran, or a prompt hook, is not designed.
