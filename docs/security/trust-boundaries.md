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
| Kubeconfig to connection | `internal/kubeconfig`, `internal/cluster` | Parse without executing; run an exec plugin only when the user connects to that context, with an argument vector; show the command before the first run of a plugin from a newly added file; never follow `exec` from a kubeconfig the user did not add |
| API response to cache | `internal/cache`, `internal/model` | Tolerate unknown and missing fields; cap object and list sizes; bound owner-reference walks |
| Cluster text to UI | `internal/model`, `internal/ui/fyneui` | Object names, labels, annotations and events are text, never markup; strip control characters before display |
| Container output to terminal | `internal/terminal`, `internal/ui/fyneui/termview` | Drop or confirm OSC 52 (clipboard write), window-title and window-manipulation sequences ([xterm control sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)); cap scrollback; cap line length. The MVP drops every OSC, DCS, APC, PM and SOS string, media copy and window operations, clamps large counts, and strips escapes from log lines, which are capped at 64 KiB per line and 10,000 lines ([`termview/sanitize.go`](../../internal/ui/fyneui/termview/sanitize.go), fuzzed; [`termview/logs.go`](../../internal/ui/fyneui/termview/logs.go)); see [H-19](#h-19-the-terminal-widget-trusts-its-input) |
| Editor to cluster | `internal/engine` | Show a diff and a server-side dry-run before apply; reject documents whose kind, name or namespace changed without saying so |
| Port-forward to local socket | `internal/cluster` | Bind `127.0.0.1` only; show which local port maps to which pod |
| Chart to release | `internal/helm` | The Helm SDK renders in-process; Helm plugins and post-renderers (which run external programs) are not supported; chart provenance is verified where the repository provides it |
| Request to command | `internal/engine` | Params validated against the schema; `enabled` and access review before `run`; confirmation by origin ([agents and MCP](agents-and-mcp.md)). The MVP validates against the schema and refuses unknown fields, checks `enabled` and the confirmation ([`engine.go`](../../internal/engine/engine.go) `Dispatch`); access reviews are not built |
| MCP request to engine | `internal/mcp` | Loopback only, bearer token, `Host` and `Origin` checks: [`internal/mcp/http.go`](../../internal/mcp/http.go), `TestHTTPGuard`, `TestNewRefusesNonLoopbackAddresses` |
| Cluster data to the LLM endpoint | `internal/agent` | Secret values withheld by default; what is sent is shown |
| Download to installed app | `internal/update` | Signed manifest, checksum, OS code signature, no downgrade ([updates](updates.md)) |

## What this does not cover

### H-1: Kubeconfig files added by sync folders

Lens watches `~/.kube/` and any files and folders listed under Preferences › Kubernetes ›
Kubeconfig Syncs, and adds every kubeconfig that appears there
([add a local cluster](https://docs.lenshq.io/k8slens/getting-started/add-clusters/add-local-cluster/),
[preferences](https://docs.lenshq.io/k8slens/using-lens/preferences/)). If Cogpit adopts that,
a file dropped into a synced folder (a download, a shared drive) becomes a cluster entry whose
exec plugin runs on the next connect. The required check (show the plugin command before its
first run) is defined above but its user interaction is not designed yet.

### H-2: Local processes and forwarded ports

A forwarded port on `127.0.0.1` is reachable by every process of every local user on the machine,
with the pod's identity behind it. Binding to loopback limits it to the host, not to Cogpit. No
per-connection authentication is possible for arbitrary TCP protocols.

### H-19: The terminal widget trusts its input

The terminal widget, `github.com/fyne-io/terminal` at the pinned commit `91063bb29d69`, froze or
crashed the whole app on output a container can send (a tab at the right margin looped forever;
delete and insert characters and some scroll regions indexed out of range, on the main goroutine
where nothing recovers), and changed Cogpit's working directory on OSC 7. Cogpit builds against a
patched copy ([`third_party/fyne-terminal`](../../third_party/fyne-terminal/PATCHES.md)) that fixes
each of these, and keeps the escape filter in front of it; tests feed every known crashing input
to the patched widget alone and behind the filter
([`termview/terminal_test.go`](../../internal/ui/fyneui/termview/terminal_test.go)). What remains:
inputs nobody has found yet can still crash the widget (it was not written for hostile input and
has had no fuzzing of its own); it has data races of its own (a configuration write at start-up,
parser state on the reader goroutine); its ANSI colours are fixed, not theme tokens; and the fork
must be kept in step with upstream until upstream takes equivalent fixes.
