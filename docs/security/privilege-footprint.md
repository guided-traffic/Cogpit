# Privilege footprint

What Cogpit does inside a cluster with the user's identity, verb by verb, and what each action
changes. Cogpit installs no cluster-side component and holds no identity of its own: every request
is made with the credentials of the kubeconfig context the user opened, so Cogpit can never do
more than `kubectl` and `helm` with the same context could. Commands from agents use the same verbs
under the confirmation rules of [agents and MCP](agents-and-mcp.md). The table is the contract the
command registry's `verbs` field must match
([architecture §4.2](../developer/architecture.md#42-the-command-registry)). Implemented on
2026-10-11, in [`internal/engine/commands`](../../internal/engine/commands/): Connect (discovery,
the watch on custom resource definitions, access reviews), list views (Secrets in full since
2026-10-12, Helm releases from their Secrets), Logs (a pod's and a workload's), Pod shell, Attach,
Port forward, Edit and apply, Create, Delete (every kind), Restart (Deployment, Daemon Set,
Stateful Set), Scale (Deployment, Stateful Set, Replica Set), Cordon, Uncordon, Drain, Trigger,
Suspend and Resume (Cron Job), Evict, a Secret's drawer, Helm rollback and uninstall, and Helm
upgrade (2026-10-12). The rows below say which. Adding and syncing kubeconfigs
(2026-10-12) sends no request to a cluster; what they let run on the user's computer is in
[credentials](credentials.md) and [trust boundaries](trust-boundaries.md#h-1-kubeconfig-files-added-by-sync-folders).

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
| Secrets list | `list`, `watch` on `secrets` in full, values included; no `get` | Built (2026-10-12, [D-13](../developer/decisions.md#d-13-cache) revised to option a, replacing the metadata-only scope of 2026-10-11): one full list or watchlist and its watch events for every Secret of every namespace while the scope runs (the Secrets view, a Secret's drawer, or an agent's read of kind `secrets`), plus the warm period; the Keys (names) and Type columns are filled as in Lens; a drawer and a reveal send nothing (`TestBudgetSecretScopes`, `TestBudgetSecretsViewAndDrawer`, `TestKindSecretScopes`, `TestKindSecretsView`). The cache still drops the `kubectl.kubernetes.io/last-applied-configuration` annotation of every Secret before the store, because it repeats the values in plain text (`TestSecretsLoseTheLastAppliedAnnotation`, `TestSecretDrawerRedactsTheLastAppliedAnnotation`). `resources.find` without a kind does not start this watch (`TestBudgetFindWithoutKindStartsNoReleaseWatch`). RBAC's `list secrets` now actually transfers every value ([H-8](#h-8-secret-data-arrives-with-the-list)) |
| Helm releases | `list`, `watch` on `secrets` with label `owner=helm`, or nothing while the full Secrets watch runs | Helm keeps releases in Secrets by default ([Helm storage backends](https://helm.sh/docs/topics/advanced/#storage-backends)); RBAC cannot narrow `list secrets` by label, so seeing releases needs the right to read every Secret in that namespace. Built (2026-10-11): the label-selected watch keeps exactly these Secrets in full in the cache while the Releases view, a release's drawer or a search across kinds uses them, plus the warm period; their payloads are decoded with a bounded memo that forgets everything 5 minutes after the last read ([`releases.go`](../../internal/helm/releases.go), `TestDeriveMemoForgetsWhenIdle`); a search across kinds does not start this watch (`TestBudgetFindWithoutKindStartsNoReleaseWatch`). Since 2026-10-12 the label-selected scope is served from the full Secrets watch while that one runs, with no second request, and never keeps it alive: when the full watch's warm period ends while releases are open, the `owner=helm` watch starts and the full one stops once it has caught up, at most a minute later ([architecture §5](../developer/architecture.md#scopes-local-and-virtual-kinds), `TestBudgetSecretsAndReleasesShareOneWatch`, `TestCoverExpiryHandsOverToOwnInformer`) |
| Metrics (metrics-server) | `get`, `list` on `pods.metrics.k8s.io`, `nodes.metrics.k8s.io` | |
| Metrics (Prometheus through the API server) | `get` on `services/proxy` for the Prometheus service; `create` when queries use `POST` | `services/proxy` lets the caller send any HTTP request to that service; queries run only while a visible view needs them |

## Actions that write

`Danger` is the registry's level: `Destructive` requires a confirmation naming the objects, the
namespace and the context ([conventions](../developer/conventions.md#never-surprise-a-cluster)).

| Action | Requests | Danger | What changes |
|---|---|---|---|
| Logs (`pod.logs`, `workload.logs`) | `get pods/log`; for a workload's logs the pods (and, for a Deployment, its Replica Sets) from the cache | Read | nothing; previous container and timestamps are request parameters; a workload's logs stream at most 40 containers at once |
| Secret drawer, reveal (`secret.reveal`) | none: the drawer and a revealed value come from the cache of the full Secrets watch (declared verbs `list secrets`) | Read, GUI only | nothing; a revealed value (cut to 4 KiB or 40 lines) lives only in the window's widgets until Hide, another object, closing the drawer or a change of the Secret (the reveal names the Secret's `resourceVersion`; a changed or recreated Secret is refused and its old value masked and forgotten, `TestRevealedValueDroppedWhenSecretChanges`, R-6); agents and the CLI are refused, and a refused reveal is audited with the key name only (`TestRevealIsGUIOnly`) |
| Refresh access (`cluster.refreshAccess`) | `create selfsubjectrulesreviews`, `create selfsubjectaccessreviews` as on connect | Read | nothing; drops and refetches the cached answers |
| Pod shell, attach (`pod.shell`, `pod.attach`) | `create pods/exec`, `create pods/attach` (on API servers from 1.35 with `AuthorizePodWebsocketUpgradeCreatePermission`, on by default, `create` is checked for the WebSocket upgrade too, verified in the Kubernetes 1.36 source) | Mutating | Runs a process in the container, or attaches to its main process: with stdin, everything typed reaches that process, and with a TTY that includes Ctrl+C and Ctrl+D, which can end the main process and restart the container; closing an attach tab detaches and the process keeps running (`TestKindWorkloadLogsAttachRestartScale`) |
| Port forward (`pod.portForward`, `service.portForward`, `portforward.edit`) | `create pods/portforward`, sent as a WebSocket upgrade first (counted as `get pods/portforward`) and as an SPDY `POST` when the server refuses it; on API servers from 1.35 the upgrade is checked for `create` too (`AuthorizePodWebsocketUpgradeCreatePermission`, verified in the Kubernetes 1.36 source); `list`, `watch` on `pods` (and the Service from the cache) for reconnects, through the shared cache | Mutating; `portforward.open`, `stop` and `remove` are UI | Opens a local socket into the pod on `127.0.0.1` only ([H-2](trust-boundaries.md#h-2-local-processes-and-forwarded-ports)); nothing in the cluster changes and no object is created. Built 2026-10-11 ([`portforward.go`](../../internal/cluster/portforward.go)): Cogpit owns the listener for the forward's whole life, a service forward moves to another ready pod when its pod goes, and a forward ends with Stop, Delete, switching context or quitting; after a crash the operating system closes the listeners and there is nothing to clean up in the cluster (`TestKindPortForward`) |
| Edit and apply (`object.apply`) | `update` on the kind, first with `dryRun=All`, then for real; no `get`: the editor's text comes from the cache | Mutating | The object, replaced by the edited text with the resourceVersion the tab started from as precondition: a concurrent change fails with a conflict and is never overwritten. An update without `managedFields` keeps every manager's entries and records Cogpit's under `cogpit`. Secrets are refused (decided 2026-10-12, [D-19](../developer/decisions.md#d-19-editing-and-creating-secrets-from-yaml): editable and creatable in the GUI only, agents and the CLI refused; not built) (their values are redacted in Cogpit). An edit that sets `spec.replicas` to 0 is confirmed as destructive, as Scale to zero is. An apply from the tab's text (no `yaml`) names the tab's text revision the user reviewed (`textRevision`, required): a text an agent changed since, while the confirmation was open, is refused (`TestApplyFromTabBoundToTheReviewedText`, R-10) |
| Server-side dry run (`object.dryRun`) | `update` (or `create`) with `dryRun=All` | Read | Nothing is stored; RBAC still asks for the write verb. Admission webhooks with side effects refuse dry runs. A dry run that meets a conflict reports it but never moves the edit's base version, so an agent's check cannot make the user's next apply overwrite a concurrent change |
| Create from template (`object.create`) | discovery `get` to map apiVersion and kind; `create` on the kind, first with `dryRun=All` | Mutating | New objects, one per document; Secrets are refused (decided 2026-10-12, [D-19](../developer/decisions.md#d-19-editing-and-creating-secrets-from-yaml): editable and creatable in the GUI only, agents and the CLI refused; not built). Objects whose namespace comes from the namespace filter are created only in the namespace the confirmation named: a filter changed while the dialog was open refuses the create |
| Scale (`deployment.scale`, `statefulset.scale`, `replicaset.scale`) | `patch` on `deployments/scale`, `statefulsets/scale` or `replicasets/scale`, with `metadata.uid` in the patch | Mutating; `Destructive` when scaling to zero (the engine escalates that call) | Replica count; a Replica Set a Deployment controls is set back by its controller, which the GUI says after scaling it. Since 2026-10-12 every scale patch carries the UID the user confirmed (else the cache's), and the API server refuses an object created again under its name (409, "Precondition failed: UID"; verified on Kubernetes 1.36.1, `TestKindPatchesBoundToTheUID`), so no gap remains between a check and the write; a mutating confirmation records the UIDs of the targets the cache holds, so an agent's scale to a non-zero count is bound too (`TestRestartAndScaleCarryTheUID`) |
| Restart (`deployment.restart`, `daemonset.restart`, `statefulset.restart`) | `patch` on the workload, with `metadata.uid` in the patch | Destructive (owner, 2026-10-11: confirmed for every origin, because it replaces every pod) | A rolling restart of every pod: kubectl's `rollout restart` sets a pod-template annotation, which the controller rolls out (the patch body equals kubectl's plus the UID, `TestRestartWorkloadSendsKubectlsPatch`, `TestPatchesCarryTheUID`); with the OnDelete strategy pods are replaced only as they are deleted, which the confirmation says. Since 2026-10-12 the patch carries the confirmed UID and the API server refuses an object created again under its name (422, "field is immutable"; verified on Kubernetes 1.36.1, `TestKindPatchesBoundToTheUID`), which closes the gap between the cache check and the write |
| Suspend / resume (Job, CronJob) | `patch` | Mutating | `spec.suspend` |
| Trigger CronJob (`cronjob.trigger`) | `create jobs`; the CronJob is read from the cache | Mutating | A new Job from the CronJob's template, as `kubectl create job --from=cronjob/<name>`: the template's labels and annotations, the CronJob as controller, `cronjob.kubernetes.io/instantiate: manual`, a generated name `<cronjob>-manual-<suffix>` (inferred) |
| Cordon / uncordon (`node.cordon`, `node.uncordon`) | `patch nodes` (a patch of its own, never the `update` kubectl's helper can fall back to) | Mutating | `spec.unschedulable` |
| Evict a pod (`pod.evict`) | `create pods/eviction`, with the UID Cogpit's cache held when the user confirmed as precondition | Destructive | Pod deleted, honouring PodDisruptionBudgets ([API-initiated eviction](https://kubernetes.io/docs/concepts/scheduling-eviction/api-eviction/)); a budget's refusal is an error naming the budget, without retries |
| Drain (`node.drain`) | `patch nodes`; `list pods` in every namespace (field selector `spec.nodeName`); `get pods` while waiting; `get daemonsets.apps` (to skip their pods); `create pods/eviction` for each pod; discovery `get /api/v1`. Never `delete pods`: a server without the eviction API is refused | Destructive | Node cordoned and every evictable pod evicted, through the kubectl drain library ([safely drain a node](https://kubernetes.io/docs/tasks/administer-cluster/safely-drain-node/)). Options with their defaults (inferred): DaemonSet pods ignored; pods with emptyDir data and pods without a controller refused unless the caller allows it (both lose data); each pod's own grace period; a 300 s timeout. A drain journal in `<user config dir>/Cogpit/drains.json` (0600) records each drain before the cordon and drops it when the drain finishes: a crash, failure or timeout leaves a cordoned, half-drained node, which the next connect offers to finish, uncordon or dismiss (tested on a three-node kind cluster, `TestKindDrainCrashFinishUncordon`). The confirmation names the options that lose data (`force`, `deleteEmptyDirData`), for Finish drain too |
| Delete (`object.delete`, every kind) | `delete`, with the UID Cogpit's cache held when the user confirmed as precondition | Destructive | The object, and its dependents through garbage collection; a replacement created under the same name while the dialog was open is never deleted, and an object that was not in the cache when the user confirmed is refused |
| Force delete (pod) | `delete` with grace period 0 | Destructive | The API object goes at once; the container may keep running on an unreachable node |
| Force finalize | `patch` removing `metadata.finalizers` | Destructive | The object goes without its finalizers running; external resources they guard can be left behind ([H-9](#h-9-force-finalize-skips-cleanup)) |
| Helm rollback, uninstall (`helm.release.rollback`, `helm.release.uninstall`); install not built | through the Helm SDK in Cogpit's process with the same context: `list`, `get`, `create`, `update`, `delete` on the release's Secrets, discovery, and whatever the release's manifests and hooks need (which no static list can name; the commands' descriptions say so) | Destructive | The release's objects and its release Secrets; rollback prunes the history to 10 revisions, as `helm rollback` does and as Lens Desktop does (captured 2026-10-12 on a real cluster: 12 revisions before a Lens rollback, 10 after). Confirmations bind the newest release Secret's UID, so a release that moved to another revision while the dialog was open is refused; a rollback also names the cached newest revision to the SDK, which refuses it before anything changes when the server holds a newer one the cache has not seen yet (`TestActionsRefuseWhenTheServerMoved`, review of the fourth run, R-1; an uninstall removes every revision, a newer one included, which is what its confirmation names: the release); since 2026-10-12 the GUI's Rollback dialog (Lens's, with every revision) is that confirmation, answered for the GUI origin, that release, context and picked revision only, within 10 s and once, and dropped when its dispatch ends without taking it (`TestRollbackDialogAnswerBinding`, R-3). Crash-safe (decided 2026-10-11, [D-11](../developer/decisions.md#d-11-crash-stance)): a journal (`<user config dir>/Cogpit/helm.json`, 0600) records the action before the SDK runs and drops it after; a release left `pending-rollback` or `uninstalling` by a Cogpit that died midway is offered on the next connect, to finish (run again) or dismiss (`TestHelmCrashJournalOffer`, `TestKindHelmReleases`). Rollback and uninstall take no context: an action runs until its 5-minute timeout |
| Helm upgrade (`helm.release.upgrade`; the tab `helm.release.upgradeTab` is UI) | through the Helm SDK as above; measured on kind for a release with one ConfigMap and one Secret (`TestKindHelmUpgrade`): `list secrets` 3 (the release history), `create secrets` 1 (the new revision), `update secrets` 2 (the old revision superseded, the new one deployed), `get` and `patch` on each of the release's objects, discovery; `create` and `delete` on objects the new values add or remove; `delete secrets` once more than 10 revisions exist | Mutating | The release's objects and a new release revision with the chart's own version (no chart repositories yet) and the new values; history kept to 10 revisions as Lens keeps it. Agents need the user's acceptance (D-2); the confirmation shows the values an agent passes inline, never serialised ([D-27](../developer/decisions.md#d-27-an-agents-helm-values-in-the-confirmation)); the audit log records the values by size only. Bound when it runs to the revision its values were read from (a tab's revision and release Secret UID, or the required `revision` parameter): a release that moved is refused (`TestHelmUpgradeRefusedWhenTheReleaseMoved`). The revision is checked again against the server's newest one right before the SDK runs, so a cache that lags does not let the values replace a revision nobody saw (`TestActionsRefuseWhenTheServerMoved`, `TestHelmActionsRefusedWhenTheServerMovedUnseen`; R-1). An upgrade from a tab is also bound to the tab's text revision the user reviewed (`textRevision`): text an agent changes while the confirmation is open, or between the GUI Save's two steps, refuses it (`TestHelmUpgradeTabBoundToTheReviewedText`, `TestHelmUpgradeTabSaves`; review of the fourth run, R-2). Values are scrubbed from errors, toasts, the journal, failed revisions' descriptions Cogpit shows and the SDK's log lines (best effort: values of 4 characters or more, also in base64, in the escaped forms error texts quote them (Go's `%q`, JSON) and numbers as a template renders them (`1e+06`); a value a template transforms is not found, `TestUpgradeScrubsValuesFromErrors`, `TestScrubValuesEscapedForms`, `TestRedactSDKLog`). Crash-safe: the journal records `upgrade` with the revision it started from and the chart, never the values; an upgrade interrupted with the release `pending-upgrade` is offered on the next connect as a rollback to that revision, the SDK's remedy (its upgrade refuses a pending release) (`TestHelmUpgradeCrashJournal`, `TestKindHelmUpgrade`). The upgrade takes a context |
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

Listing or watching Secrets returns their data, and since 2026-10-12 Cogpit asks for it: the owner
revised [D-13](../developer/decisions.md#d-13-cache) to a full watch on Secrets (option a), so that
the Secrets view fills its Keys and Type columns as Lens does. Built the same day; the
metadata-only design of 2026-10-11 and its one GET per opened drawer are gone. What remains:

- **Every value of every Secret in memory while the scope runs.** The full Secrets watch holds
  every Secret of all namespaces, `data` included, while the Secrets view, a Secret's drawer or an
  agent's read of kind `secrets` (`resources_list`, `resources_find` with a kind, `object_get`,
  `ui_navigate`) holds it, plus the warm period (5 minutes by default). Helm's releases read the
  same objects while the full watch runs (no copy); when its warm period ends while releases are
  open, it stays until Helm's own `owner=helm` watch has caught up, at most a minute.
- **RBAC cannot narrow `list secrets`,** and Cogpit now uses all of it: a user who may list Secrets
  transfers every value of the namespaces the watch covers.
- **What is wiped and what is not.** The drawer keeps no copy: each build decodes each value to
  measure it and zeroes those bytes; a reveal decodes the value, cuts it to 4 KiB or 40 lines and
  hands a Go string to the GUI. The base64 strings in the cache and a revealed string cannot be
  overwritten and wait for the garbage collector.
- **What agents get.** Names, namespaces, labels, annotations (except last-applied), key names,
  the type, and per key its size and whether it is text: never a value. The YAML redacts `data`
  and `stringData`, the drawer's Data section is masked, a list filter matches the cells only, and
  `secret.reveal` refuses every origin but the GUI. Verified through every read path an agent has
  (`kinds.list`, `resources.list` with a filter probe, `resources.find` by kind, by label and
  without a kind, `object.get`, `ui.openView`, `ui.navigate`, `ui.state`, `table.exportCsv`,
  `workloads.overview`), the audit log, events and the log (`TestAgentsReadSecretsRedacted`).
- **The last-applied annotation** still crosses the wire and is decoded before the transform drops
  it; should one reach the drawer another way, its Annotations read `<redacted>`.
- **Helm's release Secrets** (chart values, which can hold credentials, and manifests, which
  render Secrets with their data): the decoded newest revision of each release stays in a bounded
  memo until 5 minutes after the last read; the drawer shows the values in the GUI and withholds
  them from agents. The Helm upgrade scrubs values from what it shows and logs, best effort
  ([Helm upgrade](#actions-that-write)).

### H-9: Force finalize skips cleanup

Removing finalizers deletes the object without the cleanup its controllers would have done
(cloud load balancers, volumes, DNS records). Cogpit cannot know what a finalizer guards; the
confirmation can only name the finalizers being removed.

### H-22: Two Cogpit processes share the journals

The drain and Helm journals (`drains.json`, `helm.json` in the user's config directory) are
shared by every Cogpit process of the user. Built (2026-10-12): every read and every
read-modify-write of either journal takes an exclusive lock on `<dir>/journals.lock` (0600;
`flock` on macOS and Linux, `LockFileEx` on Windows); the kernel drops the lock when a process
dies, so a stale lock cannot exist, and a writer that waits more than 10 s gets an error and the
action does not start. On its first write a process creates `<dir>/owners/<random 128-bit id>.lock`
and keeps it locked for its life; each entry records that owner and its PID, and "the owner is
alive" means "its lock file is locked", so a reused PID or a crash cannot fake it. An action
running in another live Cogpit process shows as running there ("in another Cogpit window") and is
neither offered as interrupted nor finished or dismissed from here; entries of a dead owner, and
older entries without one, are offered as interrupted as before. Tested with real processes (with
the lock taken out, 35 of 120 entries were lost; with it, none) and with a process killed by
SIGKILL (`TestJournalsKeepEveryProcessesEntries`, `TestJournalEntryOfAKilledProcess`). What
remains:

- The Windows lock compiles and vets but has never run; other platforms get no lock.
- A config directory on a network file system, where `flock` may not hold across machines, is not
  covered.
- Entries are keyed by context name: two kubeconfigs that name one cluster with different context
  names, or a source file renamed (its contexts' ids change), leave a journal entry that the
  other name does not see.
- Helm itself does not refuse a second rollback of the same release by another tool.
