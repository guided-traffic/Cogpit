# Resource parity: views, columns and actions

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** minor (the MVP: six views and four actions partial) · **Target:** Lens Desktop 2026.10.51501

Every Kubernetes resource view Lens Desktop has, with the columns of its list and the actions it
offers, against Cogpit. The layout around these views (sidebar, tables, detail drawer, dock) is in
[`ui-parity.md`](ui-parity.md). Method and status vocabulary: [`README.md`](README.md).

**Sources.** Lens Desktop's documentation, one page per view under
[Using Lens](https://docs.lenshq.io/k8slens/using-lens/layout/) (Workloads, Config, Network,
Storage, Device Resources, Namespaces, Events, Helm, Access Control, Applications, Nodes,
Overview), and its release notes to 2026.10.51501. Lens Desktop was not run for this page; *capture
needed* marks what its documentation leaves open ([D-12](../developer/decisions.md#d-12-parity-sources)).

All views and actions on this page are in Lens Desktop's free plan, except the section
[Paid Lens Desktop views](#paid-lens-desktop-views). **Status:** as marked per row; the MVP
(2026-10-11) made the first rows partial. **Command ids** are Cogpit's, following [conventions](../developer/conventions.md#style);
they become the registry ids and, with underscores, the MCP tool names.

**Summary (measured, 2026-10-11):** 6 of 64 resource views and 4 of 39 actions exist, all
**partial** (none done): the views lack the metrics columns and their texts are not checked against
a running Lens; the actions are limited as noted. Lens Desktop's paid features add 24 views and
3 actions. Remaining (estimated, 2–6 h per view or action with tests): 120–360 h for the views
([G6](gaps.md#g6-resource-views)) and 62–195 h for the actions ([G7](gaps.md#g7-resource-actions)),
on top of the foundation in [G1–G4](gaps.md#summary).

Evidence for the view rows: the kind profiles in
[`internal/model/kinds`](../../internal/model/kinds/) with their golden tests (`TestGolden`,
`TestStatusTexts`, `TestBuiltinProfiles`), the engine's views, and the views filled from watches on
kind (`TestKindEngineWorkflows`).

## Every kind

What every view has, built-in or custom. A kind that Cogpit has no profile for still gets these
from discovery.

| Behaviour | Status | Evidence |
|---|---|---|
| List view per kind the cluster serves and the user may list; hidden otherwise | partial: for the six built-in profiles, filtered by discovery; no permission check | [`connector.go`](../../internal/engine/connector.go) |
| Generic columns: Name, Namespace (namespaced kinds), Age; plus `additionalPrinterColumns` for custom resources | missing | – |
| Detail drawer: metadata, labels, annotations, owner references, events of the object; Lens merges metadata and the kind's state into one Properties section, shows no UID, and folds labels and annotations behind a count (capture, 2026-10-11) | partial: plus the kind's sections and the YAML (Secret values redacted); a separate Metadata section with the UID, labels unfolded | [`drawer.go`](../../internal/engine/drawer.go), `TestDetailWithEventsNewestFirst`, `TestDetailRedactsSecrets`; [PNG](../../internal/ui/fyneui/testdata/shell_dark.png) |
| Edit (`object.edit`): YAML in a dock tab, offered when the user may patch | missing | – |
| Delete (`object.delete`): confirm, offered when the user may delete; acts on the selection | partial: pods only, one object, confirmed for every origin; no permission check | [`actions.go`](../../internal/engine/commands/actions.go) |
| Copy name (`object.copyName`) in the detail drawer | missing | – |
| Row menu (⋮): Show Details, Edit, Delete for a custom resource; the kinds' own actions are added for built-in kinds (capture, 2026-10-11, of a custom resource only) | missing | – |
| Right-click on a row opens the actions menu (since 2025.10.230725) | missing | – |

Lens Desktop's paid "Ask AI" entries on every object (Summarize, Analyze events, metrics, logs,
related resources) are superseded by Cogpit's built-in agent
([D-10](../developer/decisions.md#d-10-built-in-agent), [beyond Lens](beyond-lens.md)).

## Cluster

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Overview | metrics | CPU, memory and pod usage with requests, limits, allocatable, capacity; master and worker charts; Issues | time range from 1 hour to 2 months | missing |
| Applications | workloads grouped by application | Instance, Namespace, Managed by, Version, Violations, Age, Status | Logs `workload.logs` | missing |
| Nodes | `v1` Node | Name, (warning icon), CPU, Memory, Disk (usage bars), Taints (a count), Roles (comma-separated), Version, Age, Conditions ("Ready" in green; a cordoned node reads "SchedulingDisabled Ready", SchedulingDisabled first and uncoloured, only Ready green, and gets no warning icon) (capture, 2026-10-11); no namespace filter | Shell `node.shell`, Cordon `node.cordon`, Uncordon `node.uncordon` (takes Cordon's place on a cordoned node, in the row menu and the drawer: capture), Drain `node.drain`; detail: Devices (DRA); row menu of a cordoned node (capture): Show Details, Shell, Uncordon, Drain, Edit, Delete; in the drawer the same actions as icons: Shell, Cordon (a pause icon) or Uncordon (a play icon), Drain, Edit, Delete; detail (captured): Metrics; Properties (Created, Name, Labels, Annotations, Managed by, Addresses, OS, OS Image, Kernel version, Container runtime, Kubelet version, Conditions); Capacity and Allocatable as tables (CPU, Memory, Ephemeral Storage, Hugepages, Pods); the node's Pods (Name, warning icon, Node, Namespace, Ready, CPU, Memory, Status); Events | partial: view without CPU, Memory, Disk; a cordoned node differs (Cogpit: "Ready SchedulingDisabled", the whole cell and the row in the warning colour); no actions ([`nodes.go`](../../internal/model/kinds/nodes.go)) |
| Namespaces | `v1` Namespace | Name, Labels, Age, Status | Create `namespace.create` | partial: view; no Create ([`namespaces.go`](../../internal/model/kinds/namespaces.go)) |
| Events | `v1` Event | Type, Message, Namespace, Involved Object ("Kind: name", a link, custom kinds included), Source, Count, Age, Last Seen (captures, 2026-10-11, confirm the documented order; Last Seen is a UTC timestamp, "2026-10-11T16:04:00Z", where Age is relative; a warning's message is red, the Type cell plain; the Workloads Overview's event table orders them Type, Source, Namespace, Involved Object, Message, … and shows Last Seen relative) | – | partial: columns, Last Seen timestamp and red warning messages as captured ([`events.go`](../../internal/model/kinds/events.go), `TestStatusTexts`)

## Workloads

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Overview | all workload kinds | one link per kind with its count and a health bar, in two columns (Pods, Deployments, Daemon Sets, Stateful Sets; Replica Sets, Jobs, Cron Jobs), then the events of the selected namespaces with their own search: Type, Source, Namespace, Involved Object ("Kind: name", a link), Message, Count, Age, Last Seen (capture, 2026-10-11) | – | missing |
| Pods | `v1` Pod | Name, (warning icon: an orange triangle on a pod with problems, e.g. many restarts and a container not ready; the rule is inferred), Namespace, CPU, Memory, Containers (a square per container: green, orange, or grey for terminated or not started; colours' meaning inferred), Restarts, Controlled By (owner kind, a link), Node (a link), QoS, Status, Age (capture, 2026-10-11; Lens's documentation lists Containers before CPU and Age before Status) | Row menu (capture, 2026-10-11): Show Details, Attach Pod `pod.attach` ›, Shell `pod.shell` ›, Evict `pod.evict`, Share `pod.share` › (not a target, owner 2026-10-11), Logs `pod.logs` ›, Edit, Delete; items with › open a submenu (per container: inferred); the drawer shows the same actions as icons in the same order; forward a container port `pod.portForward` ("Forward…" at a container's port in the drawer); detail: Resource Claims; detail (captured): Metrics; Properties (Created, Name, Namespace, Labels, Controlled By, Status, Node, Pod IP, Pod IPs, Service Account, QoS Class, Conditions, Tolerations, Secrets, Pod Anti Affinities (a rule count)); Pod Volumes (count per volume type); Init Containers, then Containers, each with its own metrics, Status, Last Status, Image, Ports (a link "<name>: <port>/<protocol>") with Forward…, Environment, Mounts, Command, Arguments, Requests, Limits; Events | partial: view without CPU, Memory ([`pods.go`](../../internal/model/kinds/pods.go)); Logs partial (follow only, [`dock.go`](../../internal/engine/commands/dock.go)); Shell partial (`TestKindEngineWorkflows`, `TestKindPodExec`); the other actions missing |
| Deployments | `apps/v1` Deployment | Name, Namespace, Pods, Replicas, Age, Conditions | Scale `deployment.scale`, Restart `deployment.restart`, Logs `workload.logs` | partial: view ([`deployments.go`](../../internal/model/kinds/deployments.go)); Restart partial (confirmed, tested against a fake clientset, [`actions.go`](../../internal/engine/commands/actions.go)); Scale and Logs missing |
| Daemon Sets | `apps/v1` DaemonSet | Name, Namespace, Pods, Node Selector, Age | Restart `daemonset.restart`, Logs `workload.logs` | partial: view with the documented columns, not checked against a capture ([`daemonsets.go`](../../internal/model/kinds/daemonsets.go), `TestDaemonSetsStatusTexts`, `TestGolden`; filled from watches on kind: `TestKindEngineWorkflows`); inferred: Pods as ready/desired, Node Selector as "key=value, …" (empty without one), the status rule (pending while pods are created or rolled, a warning when short of ready pods, uncoloured when no node should run one) and the drawer section (Pods, Selector, Node Selector, Update Strategy, Images); actions missing |
| Stateful Sets | `apps/v1` StatefulSet | Name, Namespace, Pods, Replicas, Age | Restart `statefulset.restart`, Scale `statefulset.scale`, Logs `workload.logs` | partial: view with the documented columns, not checked against a capture ([`statefulsets.go`](../../internal/model/kinds/statefulsets.go), `TestStatefulSetsStatusTexts`, `TestGolden`); inferred: Pods as ready/desired and Replicas as desired (as Cogpit's Deployments), the status rule (pending while scaling or rolling, a partition and OnDelete not counted as rolling, a warning when short of ready pods) and the drawer section (Replicas, Service, Selector, Update Strategy, Pod Management Policy, revisions, Volume Claim Templates, Images); actions missing |
| Replica Sets | `apps/v1` ReplicaSet | Name, Namespace, Desired, Current, Ready, Age | Scale `replicaset.scale`, Logs `workload.logs` | partial: view with the documented columns, not checked against a capture ([`replicasets.go`](../../internal/model/kinds/replicasets.go), `TestReplicaSetsStatusTexts`, `TestGolden`; filled from watches on kind: `TestKindEngineWorkflows`); inferred: the status rule (an error on a ReplicaFailure condition, uncoloured when scaled to zero, pending while scaling, a warning when pods are not ready) and the drawer section (Replicas, Selector, Images, Conditions); actions missing |
| Replication Controllers | `v1` ReplicationController | Name, Namespace, Replicas, Desired Replicas, Selector | – | missing |
| Jobs | `batch/v1` Job | Name, Namespace, Completions, Age, Conditions | Logs `workload.logs` | partial: view with the documented columns, not checked against a capture ([`jobs.go`](../../internal/model/kinds/jobs.go), `TestJobsStatusTexts`, `TestGolden`; filled from watches on kind: `TestKindEngineWorkflows`); inferred, capture needed: Completions as kubectl prints it ("1/1", "0/1 of 2" for a work queue), Conditions as the True condition types, sorted ("Complete SuccessCriteriaMet", "Failed FailureTarget", "Suspended"; empty while running), the status rule (failed an error, finished succeeded, suspended uncoloured, running pending, a warning once pods failed and it retries) and the drawer section; Logs missing |
| Cron Jobs | `batch/v1` CronJob | Name, (warning icon), Namespace, Schedule, Suspend, Active, Last schedule, Next execution ("In about 1 hour 57 min"), Time zone, Age (capture, 2026-10-11; Next execution and Time zone are not in Lens's documentation); a suspended job shows Suspend "true" and still a Next execution; Time zone "-" when unset (capture) | Row menu (capture): Show Details, Trigger `cronjob.trigger`, Suspend `cronjob.suspend`, Edit, Delete; on a suspended job Resume `cronjob.resume` takes Suspend's place (capture: Show Details, Trigger, Resume, Edit, Delete) | partial: view with the captured columns and texts: Suspend "true"/"false", Time zone "-" when unset, Next execution "In about 1 hour 57 min", also on a suspended job ([`cronjobs.go`](../../internal/model/kinds/cronjobs.go), `TestCronJobsStatusTexts`, `TestCronJobNext`, `TestGolden`); inferred: the warning icon's rule (an error for a schedule or time zone the CronJob controller cannot use, a warning when the last scheduled job did not succeed), Last schedule as an age, a schedule without a time zone read in UTC (the controller uses its own zone), and the drawer section; Next execution is computed when the row is built: once the run is due it reads "now" until the controller records the run, and on a suspended job until anything else changes the object (`TestCronJobNextStaleness`); actions missing |

## Config

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Config Maps | `v1` ConfigMap | Name, Namespace, Keys, Age | – | missing |
| Secrets | `v1` Secret | Name, Namespace, Labels, Keys, Type, Age | Create `secret.create` | missing |
| Resource Quotas | `v1` ResourceQuota | Name, Namespace, Age | Create `resourceQuota.create` | missing |
| Limit Ranges | `v1` LimitRange | Name, Namespace, Age | – | missing |
| Horizontal Pod Autoscalers | `autoscaling/v2` HorizontalPodAutoscaler | Name, Namespace, Metrics, Min Pods, Max Pods, Replicas, Age, Status | – | missing |
| Vertical Pod Autoscalers | `autoscaling.k8s.io` VerticalPodAutoscaler (CRD) | Name, Namespace, Mode, Age, Status | – | missing |
| Pod Disruption Budgets | `policy/v1` PodDisruptionBudget | Name, (warning icon), Namespace, Min Available, Max Unavailable, Current Healthy, Desired Healthy, Age; "N/A" when a bound is not set (capture, 2026-10-11) | – | missing |
| Priority Classes | `scheduling.k8s.io/v1` PriorityClass | Name, Value, Global Default, Age; pod counts (2026.9) | – | missing |
| Runtime Classes | `node.k8s.io/v1` RuntimeClass | Name, Handler, Age | – | missing |
| Leases | `coordination.k8s.io/v1` Lease | Name, Namespace, Holder, Age | – | missing |
| Mutating Webhook Configurations | `admissionregistration.k8s.io/v1` | Name, Webhooks, Age | – | missing |
| Validating Webhook Configurations | `admissionregistration.k8s.io/v1` | Name, Webhooks, Age | – | missing |
| Mutating Admission Policies | same | Name, Mutations, Reinvocation Policy, Failure Policy, Age | – | missing |
| Mutating Admission Policy Bindings | same | Name, Policy, Age | – | missing |
| Validating Admission Policies | `admissionregistration.k8s.io` (v1, falling back to v1beta1, v1alpha1) | Name, Validations, Failure Policy, Age | – | missing |
| Validating Admission Policy Bindings | same | Name, Policy, Validation Actions, Age | – | missing |

## Network

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Services | `v1` Service | Name, Namespace, Type, Cluster IP, Ports, External IP, Selector, Age, Status | Forward a port `service.portForward`; detail tabs: Endpoint Slices, Routes | partial: view ([`services.go`](../../internal/model/kinds/services.go)); no port forward, no detail tabs |
| Endpoint Slices | `discovery.k8s.io/v1` EndpointSlice | Name, Namespace, Address Type, Endpoints, Ports, Age | – | missing |
| Endpoints | `v1` Endpoints | Name, Namespace, Endpoints, Age | – | missing |
| Ingresses | `networking.k8s.io/v1` Ingress | Name, Namespace, Load Balancers, Rules, Age | – | missing |
| Ingress Classes | `networking.k8s.io/v1` IngressClass | Name, (a star column, presumably marking the default class: inferred), Namespace (empty: the kind is cluster-scoped), Controller, API Group, Scope, Kind; no Age column and no namespace filter (capture, 2026-10-11) | – | missing |
| Network Policies | `networking.k8s.io/v1` NetworkPolicy | Name, Namespace, Policy Types, Age | – | missing |
| Port Forwarding | Cogpit's own state, not a kind | Name, Namespace, Kind, Pod Port, Local Port, Protocol, Status | Open `portforward.open`, Edit `portforward.edit`, Stop `portforward.stop`, Delete `portforward.remove` | missing |
| Gateway API › Gateway Classes, Gateways, HTTP Routes, Backend TLS Policies, gRPC Routes, Listener Sets, Reference Grants (sidebar order, capture 2026-10-11) | `gateway.networking.k8s.io` (CRDs); Listener Sets from `gateway.networking.x-k8s.io` | capture needed | – | missing |
| Gateway API › TCP Routes, TLS Routes, UDP Routes (experimental channel) | `gateway.networking.k8s.io` (CRDs) | capture needed | – | missing |

Gateway API views appear only when their CRDs exist, as a sub-group of Network after Network
Policies (capture, Kubernetes 1.36.2); Lens needs a reconnect after the CRDs change. The Admission
Policies views follow the Kubernetes version instead: the mutating ones need
`admissionregistration.k8s.io/v1` (1.36 and later), see [ui-parity](ui-parity.md#cluster-sidebar).

## Storage

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Persistent Volume Claims | `v1` PersistentVolumeClaim | Name, Namespace, Storage Class, Size, Pods, Age, Status | – | missing |
| Persistent Volumes | `v1` PersistentVolume | Name, Storage Class, Capacity, Claim, Age, Status | – | missing |
| Storage Classes | `storage.k8s.io/v1` StorageClass | Name, Provisioner, Age, Reclaim Policy, Volume Binding Mode | – | missing |

## Device Resources (dynamic resource allocation)

Shown only when the cluster serves `resource.k8s.io/v1` (Kubernetes 1.34 and later).

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Resource Claims | `resource.k8s.io/v1` ResourceClaim | capture needed | – | missing |
| Resource Claim Templates | `resource.k8s.io/v1` ResourceClaimTemplate | capture needed | – | missing |
| Device Classes | `resource.k8s.io/v1` DeviceClass | capture needed | – | missing |
| Resource Slices | `resource.k8s.io/v1` ResourceSlice | capture needed | – | missing |

## Helm

| View | Source | Lens columns | Actions | Status |
|---|---|---|---|---|
| Charts | configured Helm repositories | Name, Description, Version, App Version, Repository; search "Search Helm Charts" | Install `helm.chart.install`: version picker, values in a "Helm Install" dock tab | missing |
| Releases | release Secrets (or ConfigMaps, by `HELM_DRIVER`) | Name, Namespace, Chart, Revision, Version, App Version, Status, Updated | Upgrade `helm.release.upgrade` ("Helm Upgrade" dock tab), Rollback `helm.release.rollback` (to a chosen revision; offered only after an upgrade), Delete `helm.release.uninstall` | missing |

## Access Control

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Service Accounts | `v1` ServiceAccount | Name, Namespace, Age | Create `serviceAccount.create` | missing |
| Cluster Roles | `rbac.authorization.k8s.io/v1` ClusterRole | Name, (warning icon), Age; no namespace filter (capture, 2026-10-11) | Create `clusterRole.create` | missing |
| Roles | `rbac.authorization.k8s.io/v1` Role | Name, (warning icon), Namespace, Age (capture, 2026-10-11) | Create `role.create` | missing |
| Cluster Role Bindings | `rbac.authorization.k8s.io/v1` ClusterRoleBinding | Name, (warning icon), Bindings (the bound subjects' names, comma-separated), Age; no namespace filter (capture, 2026-10-11) | Create `clusterRoleBinding.create` | missing |
| Role Bindings | `rbac.authorization.k8s.io/v1` RoleBinding | Name, Namespace, Bindings (the bound subjects' names, comma-separated), Age; no warning icon seen (capture, 2026-10-11) | Create `roleBinding.create` | missing |
| Pod Security Policies | `policy/v1beta1` PodSecurityPolicy (clusters before 1.25) | Name, Privileged, Volume, Age | – | missing |

## Custom Resources

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Definitions | `apiextensions.k8s.io/v1` CustomResourceDefinition | Resource (the Kind, a link), Group, Version, Scope, Age; a group filter ("All groups") instead of the namespace filter; listed in the order of the definitions' names, plural and group (capture, 2026-10-11; the order is inferred) | – | missing |
| Custom Resources: one view per custom resource kind the cluster serves, titled by its singular Kind name, under one collapsed group per API group; groups and kinds sorted by name; groups with a curated section (Flux, Gateway API) are listed here too (captures, 2026-10-11) | from discovery | Name, Namespace (namespaced kinds), the `additionalPrinterColumns` of priority 0, Age (capture of cert-manager's Order: Name, Namespace, State, Age; that Issuer and Reason are hidden because of priority 1 is inferred) | – | missing |

## Paid Lens Desktop views

Lens Desktop gates these behind its Plus, Pro and Enterprise plans
([premium features](https://docs.lenshq.io/k8slens/premium-features/)). They count toward the
full target only, not toward the mainstream or essentials audiences
([target-app parity](target-app-parity.md#readiness-by-audience)). Flux CD and Argo CD are in
Cogpit's beta ([G23](gaps.md#g23-flux-cd-and-argo-cd-views)); Security Center and Cost Monitoring
come after it ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)).

| View | Lens contents | Actions | Status |
|---|---|---|---|
| Flux CD | sidebar group between Applications and Nodes (capture, 2026-10-11): Dashboard, Helm Releases, Kustomizations; Image Automation › Image Repositories, Image Policies, Image Update Automations; Notifications › Alerts, Providers, Receivers; Sources › Git Repositories, OCI Repositories, Helm Repositories, Buckets, External Artifacts | reconcile and suspend not documented | missing |
| Argo CD | Dashboard, Applications, Application Sets, Projects, Settings › Repositories, Settings › Clusters | Sync `argocd.application.sync` (with Prune), Refresh `argocd.application.refresh` (Normal or Hard), both in bulk | missing |
| Security Center | Images (Name, Platforms, Pods, Vulnerabilities, Exposed Secrets, Status), Resources, Roles; from the Trivy Operator | Scan image `securityCenter.image.scan`, Show details, CSV export | missing |
| Cost Monitoring | costs over 24h, 7d, 30d; "Include idle" | Optimize cost with AI | missing |

## Create dialogs

Lens Desktop has dialogs (not YAML) to create Namespaces, Secrets, Resource Quotas, Service
Accounts, Cluster Roles, Roles, Cluster Role Bindings and Role Bindings; their command ids are in
the tables above. Everything else is created from YAML templates in the dock
([ui-parity › Editor](ui-parity.md#editor-and-create-resource)).

## Revision history

| Date | Change | Summary |
|---|---|---|
| 2026-10-11 | minor | Daemon Sets, Stateful Sets, Replica Sets, Jobs and Cron Jobs views partial: kind profiles with golden and fuzz tests; Cron Jobs as captured, its warning rule and the other views' status rules inferred |
| 2026-10-11 | minor | Capture of a pod drawer: Pod Anti Affinities, Init Containers before Containers, the port link's form |
| 2026-10-11 | minor | Capture of Ingress Classes: its columns, a star column and an empty Namespace column, no Age |
| 2026-10-11 | minor | Captures of a suspended Cron Job and a cordoned Node: Resume and Uncordon take Suspend's and Cordon's place; a cordoned node's Conditions and how Cogpit differs |
| 2026-10-11 | minor | Columns of Cluster Roles, Roles, Cluster Role Bindings and Role Bindings from captures |
| 2026-10-11 | minor | Events' Last Seen is a UTC timestamp and warning messages are red (Cogpit follows); Nodes columns captured; Share is not a target (owner), 39 actions |
| 2026-10-11 | minor | Captures of the Pods row menu, the warning icon, an evicted pod and Cron Jobs: pod actions with submenus, Cron Jobs columns and actions |
| 2026-10-11 | minor | Capture of the Events view: documented column order confirmed; Last Seen as a date |
| 2026-10-11 | minor | Captures of the pod and node drawers and the Nodes list: their sections and fields, the node actions in the drawer |
| 2026-10-11 | minor | Captures of Definitions, Pods, Pod Disruption Budgets and the Workloads Overview: their columns; Pods show Status before Age |
| 2026-10-11 | minor | Capture of a custom resource view: its columns (printer columns of priority 0) and its row menu |
| 2026-10-11 | minor | Second capture (Kubernetes 1.36.2): Gateway API sub-group and its order, Mutating Admission Policies first and version-dependent, custom resource kinds by singular Kind name |
| 2026-10-11 | minor | From a capture of the running Lens Desktop (owner): Definitions (custom resource definitions) is a Lens view after all, 64 views; custom resources grouped by API group; Endpoint Slices before Endpoints; webhook view titles; the Flux CD sidebar structure |
| 2026-10-11 | minor | The MVP: Nodes, Pods, Deployments, Services, Namespaces and Events views partial; Logs, Shell, Restart and Delete (pods) partial |
| 2026-10-11 | major | Sources narrowed to Lens Desktop (D-12): the Freelens-only Definitions view, seven Freelens-only actions, `portforward.start` and Freelens column details removed; Ask AI superseded by the built-in agent |
| 2026-10-11 | major | First version, from Lens Desktop's documentation and release notes to 2026.10.51501 and Freelens 1.10.3 source |
