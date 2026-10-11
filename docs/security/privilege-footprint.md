# Privilege footprint

What Cogpit does inside a cluster with the user's identity, verb by verb, and what each action
changes. Cogpit installs no cluster-side component and holds no identity of its own: every request
is made with the credentials of the kubeconfig context the user opened, so Cogpit can never do
more than `kubectl` and `helm` with the same context could. Commands from agents use the same verbs
under the confirmation rules of [agents and MCP](agents-and-mcp.md). The table is the contract the
command registry's `verbs` field must match
([architecture §4.2](../developer/architecture.md#42-the-command-registry)). Implemented on
2026-10-11, in [`internal/engine/commands`](../../internal/engine/commands/): Connect (discovery
and the watch on custom resource definitions; no access reviews yet), list views, Logs, Pod shell,
Edit and apply, Create, Delete (every kind), Restart (Deployment), Scale (Deployment, Stateful
Set), Cordon, Uncordon, Drain, Trigger, Suspend and Resume (Cron Job) and Evict. The rows below say
which.

Verbs and subresources follow Kubernetes RBAC
([RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/),
[request verbs](https://kubernetes.io/docs/reference/access-authn-authz/authorization/#determine-the-request-verb)).
The actions are Lens's (see [resource parity](../parity/resource-parity.md)).

## On connect and while browsing

| Action | Requests | Notes |
|---|---|---|
| Connect | `get` on the discovery endpoints; `create selfsubjectaccessreviews` and `selfsubjectrulesreviews` (planned: access reviews are not built) | The default `system:basic-user` and `system:discovery` roles grant these to every authenticated user |
| Custom resource kinds | `list` once, then `watch` on `customresourcedefinitions.apiextensions.k8s.io` (cluster-scoped), for as long as the context is connected | Built: the kinds of established definitions become views and follow the definitions live ([`customkinds.go`](../../internal/engine/customkinds.go)). A user without these verbs gets no custom resource views and no error; the informer keeps retrying with client-go's backoff. The objects carry their full schemas, which Cogpit holds in memory |
| Open a list view | `list` once, then `watch` on the kind, cluster-wide or per accessible namespace | One watch per kind and scope, shared by every view and agent query; it stays open for the warm period after the last consumer (default five minutes), then closes ([architecture §5](../developer/architecture.md#5-cache)). A user who may not list namespaces names them in cluster settings (Lens: "Accessible Namespaces") |
| Secrets list | `list`, `watch` on `secrets`, as metadata only (`PartialObjectMetadata`); `get` on one Secret when its drawer opens | Decided ([D-13](../developer/decisions.md#d-13-cache), H-8 option b, owner 2026-10-11; not built): the list holds names, labels and annotations, not values; the opened Secret's data is fetched once and masked until the user reveals a field. RBAC still grants the values with `list`, and the server sends no data only because Cogpit asks for metadata |
| Helm releases | `list` on `secrets` with label `owner=helm` | Helm keeps releases in Secrets by default ([Helm storage backends](https://helm.sh/docs/topics/advanced/#storage-backends)); RBAC cannot narrow `list secrets` by label, so seeing releases needs the right to read every Secret in that namespace. Decided (H-8 option b): a watch with the label selector `owner=helm` keeps exactly these Secrets in full in the cache, because Helm needs their payloads |
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
| Edit and apply (`object.apply`) | `update` on the kind, first with `dryRun=All`, then for real; no `get`: the editor's text comes from the cache | Mutating | The object, replaced by the edited text with the resourceVersion the tab started from as precondition: a concurrent change fails with a conflict and is never overwritten. An update without `managedFields` keeps every manager's entries and records Cogpit's under `cogpit`. Secrets are refused (their values are redacted in Cogpit). An edit that sets `spec.replicas` to 0 is confirmed as destructive, as Scale to zero is |
| Server-side dry run (`object.dryRun`) | `update` (or `create`) with `dryRun=All` | Read | Nothing is stored; RBAC still asks for the write verb. Admission webhooks with side effects refuse dry runs. A dry run that meets a conflict reports it but never moves the edit's base version, so an agent's check cannot make the user's next apply overwrite a concurrent change |
| Create from template (`object.create`) | discovery `get` to map apiVersion and kind; `create` on the kind, first with `dryRun=All` | Mutating | New objects, one per document; Secrets are refused. Objects whose namespace comes from the namespace filter are created only in the namespace the confirmation named: a filter changed while the dialog was open refuses the create |
| Scale (`deployment.scale`, `statefulset.scale`) | `patch` on `deployments/scale` or `statefulsets/scale` | Mutating; `Destructive` when scaling to zero (the engine escalates that call) | Replica count |
| Restart (Deployment, DaemonSet, StatefulSet) | `patch` on the workload | Destructive (owner, 2026-10-11: confirmed for every origin, because it replaces every pod) | A rolling restart of every pod: kubectl's `rollout restart` sets a pod-template annotation, which the controller rolls out |
| Suspend / resume (Job, CronJob) | `patch` | Mutating | `spec.suspend` |
| Trigger CronJob (`cronjob.trigger`) | `create jobs`; the CronJob is read from the cache | Mutating | A new Job from the CronJob's template, as `kubectl create job --from=cronjob/<name>`: the template's labels and annotations, the CronJob as controller, `cronjob.kubernetes.io/instantiate: manual`, a generated name `<cronjob>-manual-<suffix>` (inferred) |
| Cordon / uncordon (`node.cordon`, `node.uncordon`) | `patch nodes` (a patch of its own, never the `update` kubectl's helper can fall back to) | Mutating | `spec.unschedulable` |
| Evict a pod (`pod.evict`) | `create pods/eviction`, with the UID Cogpit's cache held when the user confirmed as precondition | Destructive | Pod deleted, honouring PodDisruptionBudgets ([API-initiated eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/)); a budget's refusal is an error naming the budget, without retries |
| Drain (`node.drain`) | `patch nodes`; `list pods` in every namespace (field selector `spec.nodeName`); `get pods` while waiting; `get daemonsets.apps` (to skip their pods); `create pods/eviction` for each pod; discovery `get /api/v1`. Never `delete pods`: a server without the eviction API is refused | Destructive | Node cordoned and every evictable pod evicted, through the kubectl drain library ([safely drain a node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)). Options with their defaults (inferred): DaemonSet pods ignored; pods with emptyDir data and pods without a controller refused unless the caller allows it (both lose data); each pod's own grace period; a 300 s timeout. A drain journal in `<user config dir>/Cogpit/drains.json` (0600) records each drain before the cordon and drops it when the drain finishes: a crash, failure or timeout leaves a cordoned, half-drained node, which the next connect offers to finish, uncordon or dismiss (tested on a three-node kind cluster, `TestKindDrainCrashFinishUncordon`). The confirmation names the options that lose data (`force`, `deleteEmptyDirData`), for Finish drain too |
| Delete (`object.delete`, every kind) | `delete`, with the UID Cogpit's cache held when the user confirmed as precondition | Destructive | The object, and its dependents through garbage collection; a replacement created under the same name while the dialog was open is never deleted, and an object that was not in the cache when the user confirmed is refused |
| Force delete (pod) | `delete` with grace period 0 | Destructive | The API object goes at once; the container may keep running on an unreachable node |
| Force finalize | `patch` removing `metadata.finalizers` | Destructive | The object goes without its finalizers running; external resources they guard can be left behind ([H-9](#h-9-force-finalize-skips-cleanup)) |
| Helm install, upgrade, rollback, uninstall | whatever the chart's manifests need, through the Helm SDK in Cogpit's process with the same context | Mutating; uninstall and rollback `Destructive` | The release's objects and its release Secret; a crash midway leaves the release `pending-*` until finished or rolled back ([D-11](../developer/decisions.md#d-11-crash-stance)) |
| Node shell | `create pods` in the node-shell namespace, then `create pods/exec`, then `delete pods` | Destructive | A privileged pod on the node ([H-7](#h-7-node-shell-is-root-on-the-node)) |

## Node shell

Lens opens a node shell through a pod; its cluster settings name the image and an image pull
secret in `kube-system` ([Lens cluster settings](https://docs.lenshq.io/k8slens/cluster/cluster-settings/));
its default image is a tag, `docker.io/alpine:3.19` (captured 2026-10-11).
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

Listing or watching Secrets returns their data. The owner decided (2026-10-11,
[D-13](../developer/decisions.md#d-13-cache), option b) that the Secrets view watches metadata only
(`PartialObjectMetadataList`), fetches one Secret's data when its drawer opens, and that Helm
watches only its release Secrets (`owner=helm`). Not built yet. What remains once it is: the
values of Helm's release Secrets (chart values, which can hold credentials) are in Cogpit's memory
for as long as their namespace is watched; an opened Secret's values are in memory while its
drawer shows them; the "Keys" column Lens shows stays empty, because metadata carries no keys; and
RBAC cannot narrow `list secrets` by label, so the user's role still grants every value.

### H-9: Force finalize skips cleanup

Removing finalizers deletes the object without the cleanup its controllers would have done
(cloud load balancers, volumes, DNS records). Cogpit cannot know what a finalizer guards; the
confirmation can only name the finalizers being removed.
