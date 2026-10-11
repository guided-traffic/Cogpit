# Trust boundaries

Who and what Cogpit trusts, and the points where untrusted data crosses into trusted state. No
code exists yet (2026-10-11): the boundaries below exist in the problem itself, and each row names
the check the implementation must have at that boundary and the planned package that owns it.

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
| Container output to terminal | `internal/terminal` | Drop or confirm OSC 52 (clipboard write), window-title and window-manipulation sequences ([xterm control sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)); cap scrollback; cap line length |
| Editor to cluster | `internal/engine` | Show a diff and a server-side dry-run before apply; reject documents whose kind, name or namespace changed without saying so |
| Port-forward to local socket | `internal/cluster` | Bind `127.0.0.1` only; show which local port maps to which pod |
| Chart to release | `internal/helm` | The Helm SDK renders in-process; Helm plugins and post-renderers (which run external programs) are not supported; chart provenance is verified where the repository provides it |
| Request to command | `internal/engine` | Params validated against the schema; `enabled` and access review before `run`; confirmation by origin ([agents and MCP](agents-and-mcp.md)) |
| MCP request to engine | `internal/mcp` | Loopback only, bearer token, `Host` and `Origin` checks |
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
