# Privilege footprint

What Cogpit does inside a cluster with the user's identity, verb by verb, and what each action
changes. Cogpit installs no cluster-side component and holds no identity of its own: every request
is made with the credentials of the kubeconfig context the user opened, so Cogpit can never do
more than `kubectl` and `helm` with the same context could. Commands from agents use the same verbs
under the confirmation rules of [agents and MCP](agents-and-mcp.md). The table is the contract the
command registry's `verbs` field must match
([architecture §4.2](../developer/architecture.md#42-the-command-registry)). Implemented on
2026-10-11, in [`internal/engine/commands`](../../internal/engine/commands/): Connect (discovery,
the watch on custom resource definitions, access reviews), list views (Secrets as metadata only,
Helm releases from their Secrets), Logs (a pod's and a workload's), Pod shell, Attach, Port
forward, Edit and apply, Create, Delete (every kind), Restart (Deployment, Daemon Set, Stateful
Set), Scale (Deployment, Stateful Set, Replica Set), Cordon, Uncordon, Drain, Trigger, Suspend and
Resume (Cron Job), Evict, a Secret's drawer, and Helm rollback and uninstall. The rows below say
which.

Verbs and subresources follow Kubernetes RBAC
([RBAC](https://kubernetes.io/docs/reference/access-authn-authz/rbac/),
[request verbs](https://kubernetes.io/docs/reference/access-authn-authz/authorization/#determine-the-request-verb)).
The actions are Lens's (see [resource parity](../parity/resource-parity.md)).

## On connect and while browsing

| Action | Requests | Notes |
|---|---|---|
| Connect | `get` on the discovery endpoints; `create selfsubjectrulesreviews` (one per namespace: the context's default namespace at connect, the namespace filter's on every change, and, when the user may list namespaces, up to 32 of the namespaces the cache lists) and `create selfsubjectaccessreviews` (where the rules cannot answer: cluster-scoped kinds named only by a RoleBinding's rules, incomplete rules, wildcards; and once for "may the user list namespaces"); with that answer yes, a `list` and `watch` on `namespaces` for the life of the connection, shared with the namespace filter | The default `system:basic-user` and `system:discovery` roles grant the reviews to every authenticated user. Built 2026-10-11 ([`access.go`](../../internal/cluster/access.go), [`accessreviews.go`](../../internal/engine/accessreviews.go)): reviews have a budget of their own (10 per second, burst 20) so that they never delay a view's list; the server stores nothing; the answers live in memory per connection until `cluster.refreshAccess` or the next connect, with no time-to-live and no polling; after a review fails, no more are sent until a refresh; the answers are bounded at 512 namespaces and 2,048 questions, past which the answer is unknown and the API server decides (`TestBudgetAccessReviews`, `TestKindAccessReviews`, `TestAccessReviewsAreBounded`) |
| Custom resource kinds | `list` once, then `watch` on `customresourcedefinitions.apiextensions.k8s.io` (cluster-scoped), for as long as the context is connected | Built: the kinds of established definitions become views and follow the definitions live ([`customkinds.go`](../../internal/engine/customkinds.go)). A user without these verbs gets no custom resource views and no error; the informer keeps retrying with client-go's backoff. The objects carry their full schemas, which Cogpit holds in memory |
| Open a list view | `list` once, then `watch` on the kind, cluster-wide or per accessible namespace | One watch per kind and scope, shared by every view and agent query; it stays open for the warm period after the last consumer (default five minutes), then closes ([architecture §5](../developer/architecture.md#5-cache)). A user who may not list namespaces names them in cluster settings (Lens: "Accessible Namespaces") |
| Secrets list | `list`, `watch` on `secrets`, as metadata only (`PartialObjectMetadata`); `get` on one Secret when its drawer opens | Built (2026-10-11, [D-13](../developer/decisions.md#d-13-cache), H-8 option b): the list holds names, labels and annotations, not values, and the cache drops the `kubectl.kubernetes.io/last-applied-configuration` annotation of every Secret before the store, because it repeats the values (`TestBudgetSecretScopes`, `TestKindSecretScopes`, `TestSecretsLoseTheLastAppliedAnnotation`); opening a Secret's drawer makes one `get` (never repeated while it stays open), and the engine holds its type and, per key, the size and up to 4 KiB of text only while the drawer shows it (`TestSecretDrawerFetchesOncePerOpenAndMasks`, `TestKindSecretsView`). The Keys and Type columns stay empty: metadata carries neither. RBAC still grants the values with `list`; the server sends no data only because Cogpit asks for metadata. Revised (owner, 2026-10-12, [D-13](../developer/decisions.md#d-13-cache)): a full `list` and `watch` on `secrets` will replace the metadata-only scope, values included; not built |
| Helm releases | `list`, `watch` on `secrets` with label `owner=helm` | Helm keeps releases in Secrets by default ([Helm storage backends](https://helm.sh/docs/topics/advanced/#storage-backends)); RBAC cannot narrow `list secrets` by label, so seeing releases needs the right to read every Secret in that namespace. Built (2026-10-11, H-8 option b): the label-selected watch keeps exactly these Secrets in full in the cache while the Releases view, a release's drawer or a search across kinds uses them, plus the warm period; their payloads are decoded with a bounded memo that forgets everything 5 minutes after the last read ([`releases.go`](../../internal/helm/releases.go), `TestDeriveMemoForgetsWhenIdle`); a search across kinds no longer starts this watch (`TestBudgetFindWithoutKindStartsNoReleaseWatch`) |
| Metrics (metrics-server) | `get`, `list` on `pods.metrics.k8s.io`, `nodes.metrics.k8s.io` | |
| Metrics (Prometheus through the API server) | `get` on `services/proxy` for the Prometheus service; `create` when queries use `POST` | `services/proxy` lets the caller send any HTTP request to that service; queries run only while a visible view needs them |

## Actions that write

`Danger` is the registry's level: `Destructive` requires a confirmation naming the objects, the
namespace and the context ([conventions](../developer/conventions.md#never-surprise-a-cluster)).

| Action | Requests | Danger | What changes |
|---|---|---|---|
| Logs (`pod.logs`, `workload.logs`) | `get pods/log`; for a workload's logs the pods (and, for a Deployment, its Replica Sets) from the cache | Read | nothing; previous container and timestamps are request parameters; a workload's logs stream at most 40 containers at once |
| Secret drawer, reveal (`secret.reveal`) | none: the value comes from the drawer's `get` above | Read, GUI only | nothing; a revealed value lives only in the window's widgets until Hide, another object or closing the drawer; agents and the CLI are refused |
| Refresh access (`cluster.refreshAccess`) | `create selfsubjectrulesreviews`, `create selfsubjectaccessreviews` as on connect | Read | nothing; drops and refetches the cached answers |
| Pod shell, attach (`pod.shell`, `pod.attach`) | `create pods/exec`, `create pods/attach` (on API servers from 1.35 with `AuthorizePodWebsocketUpgradeCreatePermission`, on by default, `create` is checked for the WebSocket upgrade too, verified in the Kubernetes 1.36 source) | Mutating | Runs a process in the container, or attaches to its main process: with stdin, everything typed reaches that process, and with a TTY that includes Ctrl+C and Ctrl+D, which can end the main process and restart the container; closing an attach tab detaches and the process keeps running (`TestKindWorkloadLogsAttachRestartScale`) |
| Port forward (`pod.portForward`, `service.portForward`, `portforward.edit`) | `create pods/portforward`, sent as a WebSocket upgrade first (counted as `get pods/portforward`) and as an SPDY `POST` when the server refuses it; on API servers from 1.35 the upgrade is checked for `create` too (`AuthorizePodWebsocketUpgradeCreatePermission`, verified in the Kubernetes 1.36 source); `list`, `watch` on `pods` (and the Service from the cache) for reconnects, through the shared cache | Mutating; `portforward.open`, `stop` and `remove` are UI | Opens a local socket into the pod on `127.0.0.1` only ([H-2](trust-boundaries.md#h-2-local-processes-and-forwarded-ports)); nothing in the cluster changes and no object is created. Built 2026-10-11 ([`portforward.go`](../../internal/cluster/portforward.go)): Cogpit owns the listener for the forward's whole life, a service forward moves to another ready pod when its pod goes, and a forward ends with Stop, Delete, switching context or quitting; after a crash the operating system closes the listeners and there is nothing to clean up in the cluster (`TestKindPortForward`) |
| Edit and apply (`object.apply`) | `update` on the kind, first with `dryRun=All`, then for real; no `get`: the editor's text comes from the cache | Mutating | The object, replaced by the edited text with the resourceVersion the tab started from as precondition: a concurrent change fails with a conflict and is never overwritten. An update without `managedFields` keeps every manager's entries and records Cogpit's under `cogpit`. Secrets are refused (their values are redacted in Cogpit). An edit that sets `spec.replicas` to 0 is confirmed as destructive, as Scale to zero is |
| Server-side dry run (`object.dryRun`) | `update` (or `create`) with `dryRun=All` | Read | Nothing is stored; RBAC still asks for the write verb. Admission webhooks with side effects refuse dry runs. A dry run that meets a conflict reports it but never moves the edit's base version, so an agent's check cannot make the user's next apply overwrite a concurrent change |
| Create from template (`object.create`) | discovery `get` to map apiVersion and kind; `create` on the kind, first with `dryRun=All` | Mutating | New objects, one per document; Secrets are refused. Objects whose namespace comes from the namespace filter are created only in the namespace the confirmation named: a filter changed while the dialog was open refuses the create |
| Scale (`deployment.scale`, `statefulset.scale`, `replicaset.scale`) | `patch` on `deployments/scale`, `statefulsets/scale` or `replicasets/scale` | Mutating; `Destructive` when scaling to zero (the engine escalates that call) | Replica count; a Replica Set a Deployment controls is set back by its controller, which the GUI says after scaling it. Scaling to zero is refused when the cache holds the object under another UID than the one the user confirmed (an object created again under its name); the patch itself carries no UID precondition, so a gap remains between that check and the write |
| Restart (`deployment.restart`, `daemonset.restart`, `statefulset.restart`) | `patch` on the workload | Destructive (owner, 2026-10-11: confirmed for every origin, because it replaces every pod) | A rolling restart of every pod: kubectl's `rollout restart` sets a pod-template annotation, which the controller rolls out (the patch body equals kubectl's, `TestRestartWorkloadSendsKubectlsPatch`); with the OnDelete strategy pods are replaced only as they are deleted, which the confirmation says; every restart is refused when the cache holds the object under another UID than the one the user confirmed (created again under its name while the dialog was open), a check against the cache because kubectl's patch carries no UID precondition, which leaves a gap between the check and the write |
| Suspend / resume (Job, CronJob) | `patch` | Mutating | `spec.suspend` |
| Trigger CronJob (`cronjob.trigger`) | `create jobs`; the CronJob is read from the cache | Mutating | A new Job from the CronJob's template, as `kubectl create job --from=cronjob/<name>`: the template's labels and annotations, the CronJob as controller, `cronjob.kubernetes.io/instantiate: manual`, a generated name `<cronjob>-manual-<suffix>` (inferred) |
| Cordon / uncordon (`node.cordon`, `node.uncordon`) | `patch nodes` (a patch of its own, never the `update` kubectl's helper can fall back to) | Mutating | `spec.unschedulable` |
| Evict a pod (`pod.evict`) | `create pods/eviction`, with the UID Cogpit's cache held when the user confirmed as precondition | Destructive | Pod deleted, honouring PodDisruptionBudgets ([API-initiated eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/)); a budget's refusal is an error naming the budget, without retries |
| Drain (`node.drain`) | `patch nodes`; `list pods` in every namespace (field selector `spec.nodeName`); `get pods` while waiting; `get daemonsets.apps` (to skip their pods); `create pods/eviction` for each pod; discovery `get /api/v1`. Never `delete pods`: a server without the eviction API is refused | Destructive | Node cordoned and every evictable pod evicted, through the kubectl drain library ([safely drain a node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)). Options with their defaults (inferred): DaemonSet pods ignored; pods with emptyDir data and pods without a controller refused unless the caller allows it (both lose data); each pod's own grace period; a 300 s timeout. A drain journal in `<user config dir>/Cogpit/drains.json` (0600) records each drain before the cordon and drops it when the drain finishes: a crash, failure or timeout leaves a cordoned, half-drained node, which the next connect offers to finish, uncordon or dismiss (tested on a three-node kind cluster, `TestKindDrainCrashFinishUncordon`). The confirmation names the options that lose data (`force`, `deleteEmptyDirData`), for Finish drain too |
| Delete (`object.delete`, every kind) | `delete`, with the UID Cogpit's cache held when the user confirmed as precondition | Destructive | The object, and its dependents through garbage collection; a replacement created under the same name while the dialog was open is never deleted, and an object that was not in the cache when the user confirmed is refused |
| Force delete (pod) | `delete` with grace period 0 | Destructive | The API object goes at once; the container may keep running on an unreachable node |
| Force finalize | `patch` removing `metadata.finalizers` | Destructive | The object goes without its finalizers running; external resources they guard can be left behind ([H-9](#h-9-force-finalize-skips-cleanup)) |
| Helm rollback, uninstall (`helm.release.rollback`, `helm.release.uninstall`); install and upgrade not built | through the Helm SDK in Cogpit's process with the same context: `list`, `get`, `create`, `update`, `delete` on the release's Secrets, discovery, and whatever the release's manifests and hooks need (which no static list can name; the commands' descriptions say so) | Destructive | The release's objects and its release Secrets; rollback prunes the history to 10 revisions, as `helm rollback` does and as Lens Desktop does (captured 2026-10-12 on a real cluster: 12 revisions before a Lens rollback, 10 after). Confirmations bind the newest release Secret's UID, so a release that moved to another revision while the dialog was open is refused. Crash-safe (decided 2026-10-11, [D-11](../developer/decisions.md#d-11-crash-stance)): a journal (`<user config dir>/Cogpit/helm.json`, 0600) records the action before the SDK runs and drops it after; a release left `pending-rollback` or `uninstalling` by a Cogpit that died midway is offered on the next connect, to finish (run again) or dismiss (`TestHelmCrashJournalOffer`, `TestKindHelmReleases`). The SDK takes no context: an action runs until its 5-minute timeout |
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

Listing or watching Secrets returns their data. **Reopened (owner, 2026-10-12):** the owner revised
the decision below to a full watch on Secrets (D-13 option a), so that the Secrets view fills its
Keys and Type columns as Lens does. Once that is built, every value of every Secret in the watched
namespaces is in Cogpit's memory for as long as the scope runs, its warm period included, not only
an opened Secret's; the UI still masks values until the user reveals one, and agents still get them
redacted. Until then, what is built is the metadata-only design below. The owner decided (2026-10-11,
[D-13](../developer/decisions.md#d-13-cache), option b) that the Secrets view watches metadata only,
fetches one Secret's data when its drawer opens, and that Helm watches only its release Secrets
(`owner=helm`). Built the same day. What remains:

- RBAC cannot narrow `list secrets` by label, so the user's role still grants every value; Cogpit
  only does not ask for them.
- The values of Helm's release Secrets (chart values, which can hold credentials, and manifests,
  which render Secrets with their data) are in Cogpit's memory while their namespace is watched,
  including the warm period. The decoded newest revision of each release stays in a bounded
  memo until 5 minutes after the last read; its manifest's Secret data is redacted for everyone
  (every document is parsed, so a Secret written in an unusual shape is found too), its values
  are shown in the GUI and withheld from agents.
- An opened Secret's type, key sizes and up to 4 KiB of text per key are in memory while its
  drawer shows it; a revealed value until Hide or the drawer moves. The wipe is best effort: the
  engine zeroes the bytes it kept, but the GET's response and a revealed value's string wait for
  the garbage collector. Agents see the key names and
  sizes, never a value; a refused reveal by an agent is audited with the key name.
- The `last-applied-configuration` annotation of a Secret still crosses the wire with the
  metadata and is in memory while it is decoded; the cache drops it before the store.
- The Keys and Type columns Lens shows stay empty: metadata carries neither. The server-side Table
  format (`as=Table`, `includeObject=Metadata`) would carry the type and the number of keys
  without values; whether to use it is the owner's decision.

### H-9: Force finalize skips cleanup

Removing finalizers deletes the object without the cleanup its controllers would have done
(cloud load balancers, volumes, DNS records). Cogpit cannot know what a finalizer guards; the
confirmation can only name the finalizers being removed.

### H-22: Two Cogpit processes share the journals

The drain and Helm journals (`drains.json`, `helm.json` in the user's config directory) are
written by every Cogpit process of the user without a file lock, and nothing keeps a second
instance from starting. Two processes that change the same journal at once can lose each other's
entries (a review's test lost 199 of 400), and a drain or Helm action running in one process shows
as interrupted in the other, which offers to finish or dismiss it. Helm itself does not refuse a
second rollback of the same release. Within one process the journal and the running actions change
under one lock (`TestDrainsEndDoesNotDropTheNextDrain`, `TestHelmOpsEndDoesNotDropTheNextAction`).
A file lock and an owning process per entry are not designed yet.
