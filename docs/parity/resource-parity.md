# Resource parity: views, columns and actions

> **Last reviewed:** 2026-10-11 · **Last updated:** 2026-10-11 · **Change:** major (sources narrowed to Lens Desktop: Freelens views, columns and actions removed per D-12) · **Target:** Lens Desktop 2026.10.51501

Every Kubernetes resource view Lens Desktop has, with the columns of its list and the actions it
offers, against Cogpit. The layout around these views (sidebar, tables, detail drawer, dock) is in
[`ui-parity.md`](ui-parity.md). Method and status vocabulary: [`README.md`](README.md).

**Sources.** Lens Desktop's documentation, one page per view under
[Using Lens](https://docs.lenshq.io/k8slens/using-lens/layout/) (Workloads, Config, Network,
Storage, Device Resources, Namespaces, Events, Helm, Access Control, Applications, Nodes,
Overview), and its release notes to 2026.10.51501. Lens Desktop was not run for this page; *capture
needed* marks what its documentation leaves open ([D-12](../developer/decisions.md#d-12-parity-sources)).

All views and actions on this page are in Lens Desktop's free plan, except the section
[Paid Lens Desktop views](#paid-lens-desktop-views). **Status:** all rows are **missing**: no code
exists. **Command ids** are Cogpit's, following [conventions](../developer/conventions.md#style);
they become the registry ids and, with underscores, the MCP tool names.

**Summary (measured, 2026-10-11):** 0 of 63 resource views and 0 of 40 actions; Lens Desktop's
paid features add 24 views and 3 actions. Remaining (estimated, 2–6 h per view or action with
tests): 130–380 h for the views ([G6](gaps.md#g6-resource-views)) and 70–210 h for the actions
([G7](gaps.md#g7-resource-actions)), on top of the foundation in [G1–G4](gaps.md#summary).

## Every kind

What every view has, built-in or custom. A kind that Cogpit has no profile for still gets these
from discovery.

| Behaviour | Status | Evidence |
|---|---|---|
| List view per kind the cluster serves and the user may list; hidden otherwise | missing | – |
| Generic columns: Name, Namespace (namespaced kinds), Age; plus `additionalPrinterColumns` for custom resources | missing | – |
| Detail drawer: metadata, labels, annotations, owner references, events of the object | missing | – |
| Edit (`object.edit`): YAML in a dock tab, offered when the user may patch | missing | – |
| Delete (`object.delete`): confirm, offered when the user may delete; acts on the selection | missing | – |
| Copy name (`object.copyName`) in the detail drawer | missing | – |
| Right-click on a row opens the actions menu (since 2025.10.230725) | missing | – |

Lens Desktop's paid "Ask AI" entries on every object (Summarize, Analyze events, metrics, logs,
related resources) are superseded by Cogpit's built-in agent
([D-10](../developer/decisions.md#d-10-built-in-agent), [beyond Lens](beyond-lens.md)).

## Cluster

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Overview | metrics | CPU, memory and pod usage with requests, limits, allocatable, capacity; master and worker charts; Issues | time range from 1 hour to 2 months | missing |
| Applications | workloads grouped by application | Instance, Namespace, Managed by, Version, Violations, Age, Status | Logs `workload.logs` | missing |
| Nodes | `v1` Node | Name, CPU, Memory, Disk, Taints, Roles, Version, Age, Conditions | Shell `node.shell`, Cordon `node.cordon`, Uncordon `node.uncordon` (whether it is a separate item: capture needed), Drain `node.drain`; detail: Devices (DRA) | missing |
| Namespaces | `v1` Namespace | Name, Labels, Age, Status | Create `namespace.create` | missing |
| Events | `v1` Event | Type, Message, Namespace, Involved Object, Source, Count, Age, Last Seen | – | missing |

## Workloads

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Overview | all workload kinds | status summary per kind (release notes; layout: capture needed) | – | missing |
| Pods | `v1` Pod | Name, Namespace, Containers, CPU, Memory, Restarts, Controlled By, Node, QoS, Age, Status | Logs `pod.logs`, Shell `pod.shell`, Attach `pod.attach`, Evict `pod.evict`, Share `pod.share` (semantics: capture needed), forward a container port `pod.portForward`; detail: Resource Claims | missing |
| Deployments | `apps/v1` Deployment | Name, Namespace, Pods, Replicas, Age, Conditions | Scale `deployment.scale`, Restart `deployment.restart`, Logs `workload.logs` | missing |
| Daemon Sets | `apps/v1` DaemonSet | Name, Namespace, Pods, Node Selector, Age | Restart `daemonset.restart`, Logs `workload.logs` | missing |
| Stateful Sets | `apps/v1` StatefulSet | Name, Namespace, Pods, Replicas, Age | Restart `statefulset.restart`, Scale `statefulset.scale`, Logs `workload.logs` | missing |
| Replica Sets | `apps/v1` ReplicaSet | Name, Namespace, Desired, Current, Ready, Age | Scale `replicaset.scale`, Logs `workload.logs` | missing |
| Replication Controllers | `v1` ReplicationController | Name, Namespace, Replicas, Desired Replicas, Selector | – | missing |
| Jobs | `batch/v1` Job | Name, Namespace, Completions, Age, Conditions | Logs `workload.logs` | missing |
| Cron Jobs | `batch/v1` CronJob | Name, Namespace, Schedule, Suspend, Active, Last Schedule, Age | Trigger `cronjob.trigger`, Suspend `cronjob.suspend`, Resume `cronjob.resume` (whether it is a separate item: capture needed) | missing |

## Config

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Config Maps | `v1` ConfigMap | Name, Namespace, Keys, Age | – | missing |
| Secrets | `v1` Secret | Name, Namespace, Labels, Keys, Type, Age | Create `secret.create` | missing |
| Resource Quotas | `v1` ResourceQuota | Name, Namespace, Age | Create `resourceQuota.create` | missing |
| Limit Ranges | `v1` LimitRange | Name, Namespace, Age | – | missing |
| Horizontal Pod Autoscalers | `autoscaling/v2` HorizontalPodAutoscaler | Name, Namespace, Metrics, Min Pods, Max Pods, Replicas, Age, Status | – | missing |
| Vertical Pod Autoscalers | `autoscaling.k8s.io` VerticalPodAutoscaler (CRD) | Name, Namespace, Mode, Age, Status | – | missing |
| Pod Disruption Budgets | `policy/v1` PodDisruptionBudget | capture needed (Lens's page repeats the Secrets columns) | – | missing |
| Priority Classes | `scheduling.k8s.io/v1` PriorityClass | Name, Value, Global Default, Age; pod counts (2026.9) | – | missing |
| Runtime Classes | `node.k8s.io/v1` RuntimeClass | Name, Handler, Age | – | missing |
| Leases | `coordination.k8s.io/v1` Lease | Name, Namespace, Holder, Age | – | missing |
| Mutating Webhook Configs | `admissionregistration.k8s.io/v1` | Name, Webhooks, Age | – | missing |
| Validating Webhook Configs | `admissionregistration.k8s.io/v1` | Name, Webhooks, Age | – | missing |
| Validating Admission Policies | `admissionregistration.k8s.io` (v1, falling back to v1beta1, v1alpha1) | Name, Validations, Failure Policy, Age | – | missing |
| Validating Admission Policy Bindings | same | Name, Policy, Validation Actions, Age | – | missing |
| Mutating Admission Policies | same | Name, Mutations, Reinvocation Policy, Failure Policy, Age | – | missing |
| Mutating Admission Policy Bindings | same | Name, Policy, Age | – | missing |

## Network

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Services | `v1` Service | Name, Namespace, Type, Cluster IP, Ports, External IP, Selector, Age, Status | Forward a port `service.portForward`; detail tabs: Endpoint Slices, Routes | missing |
| Endpoints | `v1` Endpoints | Name, Namespace, Endpoints, Age | – | missing |
| Endpoint Slices | `discovery.k8s.io/v1` EndpointSlice | Name, Namespace, Address Type, Endpoints, Ports, Age | – | missing |
| Ingresses | `networking.k8s.io/v1` Ingress | Name, Namespace, Load Balancers, Rules, Age | – | missing |
| Ingress Classes | `networking.k8s.io/v1` IngressClass | Name, Controller, API Group, Scope, Kind (Lens's page also lists Namespace: capture needed) | – | missing |
| Network Policies | `networking.k8s.io/v1` NetworkPolicy | Name, Namespace, Policy Types, Age | – | missing |
| Port Forwarding | Cogpit's own state, not a kind | Name, Namespace, Kind, Pod Port, Local Port, Protocol, Status | Open `portforward.open`, Edit `portforward.edit`, Stop `portforward.stop`, Delete `portforward.remove` | missing |
| Gateway Classes, Gateways, HTTP Routes, GRPC Routes, Reference Grants, Backend TLS Policies | `gateway.networking.k8s.io` (CRDs) | capture needed | – | missing |
| TCP Routes, UDP Routes, TLS Routes, Listener Sets (experimental) | `gateway.networking.k8s.io` (CRDs) | capture needed | – | missing |

Gateway API views appear only when their CRDs exist; Lens needs a reconnect after the CRDs change.

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
| Cluster Roles | `rbac.authorization.k8s.io/v1` ClusterRole | capture needed | Create `clusterRole.create` | missing |
| Roles | `rbac.authorization.k8s.io/v1` Role | capture needed | Create `role.create` | missing |
| Cluster Role Bindings | `rbac.authorization.k8s.io/v1` ClusterRoleBinding | capture needed | Create `clusterRoleBinding.create` | missing |
| Role Bindings | `rbac.authorization.k8s.io/v1` RoleBinding | capture needed | Create `roleBinding.create` | missing |
| Pod Security Policies | `policy/v1beta1` PodSecurityPolicy (clusters before 1.25) | Name, Privileged, Volume, Age | – | missing |

## Custom Resources

| View | API | Lens columns | Actions | Status |
|---|---|---|---|---|
| Custom Resources: one view per custom resource kind the cluster serves (grouping: capture needed) | from discovery | capture needed; Cogpit uses `additionalPrinterColumns` | – | missing |

## Paid Lens Desktop views

Lens Desktop gates these behind its Plus, Pro and Enterprise plans
([premium features](https://docs.lenshq.io/k8slens/premium-features/)). They count toward the
full target only, not toward the mainstream or essentials audiences
([target-app parity](target-app-parity.md#readiness-by-audience)). Flux CD and Argo CD are in
Cogpit's beta ([G23](gaps.md#g23-flux-cd-and-argo-cd-views)); Security Center and Cost Monitoring
come after it ([D-9](../developer/decisions.md#d-9-scope-against-lenss-paid-and-vendor-features)).

| View | Lens contents | Actions | Status |
|---|---|---|---|
| Flux CD | Sources (GitRepository, OCIRepository, HelmRepository, Bucket, ExternalArtifact), Workloads (Kustomization, HelmRelease), Image automation (ImageRepository, ImagePolicy, ImageUpdateAutomation), Notifications (Alert, Provider, Receiver), dashboard | reconcile and suspend not documented | missing |
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
| 2026-10-11 | major | Sources narrowed to Lens Desktop (D-12): the Freelens-only Definitions view, seven Freelens-only actions, `portforward.start` and Freelens column details removed; Ask AI superseded by the built-in agent |
| 2026-10-11 | major | First version, from Lens Desktop's documentation and release notes to 2026.10.51501 and Freelens 1.10.3 source |
