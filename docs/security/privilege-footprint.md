# Privilege footprint

What Cogpit does inside a cluster with the user's identity, verb by verb, and what each action
changes. Cogpit installs no cluster-side component and holds no identity of its own: every request
is made with the credentials of the kubeconfig context the user opened, so Cogpit can never do
more than `kubectl` and `helm` with the same context could. Commands from agents use the same verbs
under the confirmation rules of [agents and MCP](agents-and-mcp.md). No code exists yet
(2026-10-11); the table is the contract the command registry's `verbs` field must match
([architecture §4.2](../developer/architecture.md#42-the-command-registry)).

Verbs and subresources follow Kubernetes RBAC
([RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/),
[request verbs](https://kubernetes.io/docs/reference/access-authn-authz/authorization/#determine-the-request-verb)).
The actions are Lens's (see [resource parity](../parity/resource-parity.md)).

## On connect and while browsing

| Action | Requests | Notes |
|---|---|---|
| Connect | `get` on the discovery endpoints; `create selfsubjectaccessreviews` and `selfsubjectrulesreviews` | The default `system:basic-user` and `system:discovery` roles grant these to every authenticated user |
| Open a list view | `list` once, then `watch` on the kind, cluster-wide or per accessible namespace | One watch per kind and scope, shared by every view and agent query; it stays open for the warm period after the last consumer (default five minutes), then closes ([architecture §5](../developer/architecture.md#5-cache)). A user who may not list namespaces names them in cluster settings (Lens: "Accessible Namespaces") |
| Secrets list | `list`, `watch` on `secrets` | A list response carries every Secret's data, not only names; see [H-8](#h-8-secret-data-arrives-with-the-list) |
| Helm releases | `list` on `secrets` with label `owner=helm` | Helm keeps releases in Secrets by default ([Helm storage backends](https://helm.sh/docs/topics/advanced/#storage-backends)); RBAC cannot narrow `list secrets` by label, so seeing releases needs the right to read every Secret in that namespace |
| Metrics (metrics-server) | `get`, `list` on `pods.metrics.k8s.io`, `nodes.metrics.k8s.io` | |
| Metrics (Prometheus through the API server) | `get` on `services/proxy` for the Prometheus service; `create` when queries use `POST` | `services/proxy` lets the caller send any HTTP request to that service; queries run only while a visible view needs them |

## Actions that write

`Danger` is the registry's level: `Destructive` requires a confirmation naming the objects, the
namespace and the context ([conventions](../developer/conventions.md#never-surprise-a-cluster)).

| Action | Requests | Danger | What changes |
|---|---|---|---|
| Logs | `get pods/log` | Read | nothing |
| Pod shell, attach | `create pods/exec`, `create pods/attach` | Mutating | Runs a process in the container; anything typed runs with the container's privileges |
| Port forward | `create pods/portforward` | Mutating | Opens a local socket into the pod ([H-2](trust-boundaries.md#h-2-local-processes-and-forwarded-ports)) |
| Edit and apply | `get`, `patch` (or `update`) | Mutating | The object; the editor shows a diff and a server-side dry-run first |
| Create from template | `create` on the kind | Mutating | A new object |
| Scale | `patch` on `<kind>/scale` | Mutating; `Destructive` when scaling to zero | Replica count |
| Restart (Deployment, DaemonSet, StatefulSet) | `patch` on the workload | Mutating | A rolling restart of every pod: kubectl's `rollout restart` sets a pod-template annotation, which the controller rolls out |
| Suspend / resume (Job, CronJob) | `patch` | Mutating | `spec.suspend` |
| Trigger CronJob | `create jobs` | Mutating | A new Job from the CronJob's template, as `kubectl create job --from=cronjob/<name>` |
| Cordon / uncordon | `patch nodes` | Mutating | `spec.unschedulable` |
| Evict a pod | `create pods/eviction` | Destructive | Pod deleted, honouring PodDisruptionBudgets ([API-initiated eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/)) |
| Drain | `patch nodes`, `list pods`, `create pods/eviction` for each pod, as the kubectl library does it | Destructive | Node cordoned and every evictable pod evicted ([safely drain a node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)); a crash midway leaves a cordoned, half-drained node, which the next start offers to finish or uncordon |
| Delete | `delete` | Destructive | The object, and its dependents through garbage collection |
| Force delete (pod) | `delete` with grace period 0 | Destructive | The API object goes at once; the container may keep running on an unreachable node |
| Force finalize | `patch` removing `metadata.finalizers` | Destructive | The object goes without its finalizers running; external resources they guard can be left behind ([H-9](#h-9-force-finalize-skips-cleanup)) |
| Helm install, upgrade, rollback, uninstall | whatever the chart's manifests need, through the Helm SDK in Cogpit's process with the same context | Mutating; uninstall and rollback `Destructive` | The release's objects and its release Secret; a crash midway leaves the release `pending-*` until finished or rolled back ([D-11](../developer/decisions.md#d-11-crash-stance)) |
| Node shell | `create pods` in the node-shell namespace, then `create pods/exec`, then `delete pods` | Destructive | A privileged pod on the node ([H-7](#h-7-node-shell-is-root-on-the-node)) |

## Node shell

Lens opens a node shell through a pod; its cluster settings name the image and an image pull
secret in `kube-system` ([Lens cluster settings](https://docs.lenshq.io/k8slens/cluster/cluster-settings/)).
A shell in the node's own namespaces cannot be had from the Kubernetes API without such a pod: it
must share the host PID namespace and be privileged to enter PID 1's namespaces, or, as
`kubectl debug node/<name>` does, share the host namespaces and mount the host's root filesystem
([debug a node with kubectl](https://kubernetes.io/docs/tasks/debug/debug-cluster/kubectl-node-debug/)).
Both violate the Baseline Pod Security Standard
([pod security standards](https://kubernetes.io/docs/concepts/security/pod-security-standards/)),
so the target namespace must allow privileged pods. Rules for Cogpit's node shell:

- The confirmation says that the action creates a privileged pod with root access to the node, and
  names the namespace, the image and the node.
- The image is a setting with a pinned default (by digest), not a floating tag.
- The pod carries `activeDeadlineSeconds`, so it ends even if Cogpit dies, and is deleted when the
  shell closes.

## What this does not cover

### H-7: Node shell is root on the node

A node shell is full root on the node for as long as its pod lives. The rules above bound its
lifetime and make it explicit; they cannot make it less powerful. If Cogpit crashes, the pod lives
until `activeDeadlineSeconds` ends it. Whether a non-privileged `kubectl debug`-style profile is
offered as the default is not decided.

### H-8: Secret data arrives with the list

Listing or watching Secrets returns their data, so a Secrets view holds every Secret value of the
watched namespaces in Cogpit's memory, even though the UI masks them. A metadata-only watch
(`PartialObjectMetadataList`) would avoid that, but cannot fill the "Keys" column Lens shows. The
trade-off is not decided.

### H-9: Force finalize skips cleanup

Removing finalizers deletes the object without the cleanup its controllers would have done
(cloud load balancers, volumes, DNS records). Cogpit cannot know what a finalizer guards; the
confirmation can only name the finalizers being removed.
