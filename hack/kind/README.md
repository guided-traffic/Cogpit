# Local kind cluster

A single-node [kind](https://kind.sigs.k8s.io) cluster named `cogpit` for running Cogpit against a
real API server, and a small seed of workloads to look at. The cluster's kubeconfig is its own file,
`.cache/kind/kubeconfig` (context `kind-cogpit`); nothing here reads or writes `~/.kube/config`,
so Cogpit and kubectl never reach another cluster by accident.

| File | What it is |
|---|---|
| [`cluster.yaml`](cluster.yaml) | kind configuration: one control-plane node, the default service CIDR `10.96.0.0/16` |
| [`seed/`](seed/) | the manifests `make kind-seed` applies, all in the namespace `cogpit-demo` |
| [`prometheus-values.yaml`](prometheus-values.yaml) | the values `make kind-prometheus` installs the `prometheus` chart with; see [Prometheus](#prometheus) |
| [`victoriametrics/`](victoriametrics/) | the manifests `make kind-vm` applies: VictoriaMetrics single-node and a small cluster in the namespace `cogpit-vm`; see [VictoriaMetrics](#victoriametrics) |
| [`trivy-operator-values.yaml`](trivy-operator-values.yaml), [`trivy/`](trivy/) | the values `make kind-trivy` installs Trivy Operator with and the workloads it scans, in the namespaces `cogpit-trivy`, `cogpit-trivy-a` and `cogpit-trivy-b`; see [Trivy Operator](#trivy-operator) |
| [`alertmanager/`](alertmanager/) | the manifests `make kind-alertmanager` applies: one Alertmanager in the namespace `cogpit-alerts`; see [Alertmanager](#alertmanager) |
| [`kyverno/`](kyverno/) | the values, policies and objects `make kind-kyverno` installs: Kyverno in `cogpit-kyverno`, its test policies and violating objects in `cogpit-kyverno-a`; see [Kyverno](#kyverno) |
| [`drain-cluster.yaml`](drain-cluster.yaml) | the second, multi-node cluster `cogpit-drain` (one control plane, two workers) for the tests that cordon and drain a node; see [The drain cluster](#the-drain-cluster) |

## Use

Needs Docker, `kind` and `kubectl`.

```sh
make kind-up     # create the cluster (or reuse it), write .cache/kind/kubeconfig, apply the seed
make run         # build Cogpit and open it on the kind cluster
make kind-seed   # re-apply the seed (unchanged objects stay unchanged)
make kind-down   # delete the cluster and its kubeconfig file
```

`make kind-up` is idempotent: when the cluster exists it exports its kubeconfig again and re-applies
the seed. `make kind-seed` waits up to three minutes each for `web` and `api` to roll out; on the
first run that is mostly image pulls.

kubectl against the cluster, always with the explicit file:

```sh
KUBECONFIG=.cache/kind/kubeconfig kubectl --context kind-cogpit get all -n cogpit-demo
```

To reset the seed, delete its namespace and apply it again:

```sh
KUBECONFIG=.cache/kind/kubeconfig kubectl --context kind-cogpit delete namespace cogpit-demo
make kind-seed
```

## The seed

Namespace `cogpit-demo`. Every image tag is pinned.

| Object | Image | What it shows |
|---|---|---|
| ConfigMap `web-content` | – | an `index.html`, mounted into `web` |
| Deployment `web`, 3 replicas | `nginx:1.30.0-alpine` | healthy pods with a readiness probe |
| Service `web`, ClusterIP `10.96.100.100`, port 80 | – | the fixed address the MCP demo finds with `resources_find {kind: services, field: clusterIP, value: 10.96.100.100}` |
| Deployment `api`, 2 replicas | `registry.k8s.io/e2e-test-images/agnhost:2.53` (`netexec`) | an HTTP echo server, as a non-root container |
| Service `api`, port 80 → 8080 | – | a second service, with an allocated ClusterIP |
| Deployment `crashloop`, 1 replica | `busybox:1.37.0` | a container that exits with an error after 5 s: CrashLoopBackOff, a growing restart count, BackOff events, logs |
| Job `migrate` | `busybox:1.37.0` | a job that completes and leaves a Completed pod (no TTL, so it stays) |
| Pod `toolbox` | `busybox:1.37.0` | a long-running pod with a shell, for the pod-shell demo |

Except `web` (nginx binds port 80 as root), the seed's containers run as user 65534 without
privilege escalation and with all capabilities dropped.

## Prometheus

`make kind-prometheus` installs a real Prometheus into the cluster for the drawer's charts: the
prometheus-community `prometheus` chart (version pinned in the Makefile, fetched with `--repo`, so no
helm repository is added to your helm configuration), release `prometheus` in the namespace
`monitoring`, with [`prometheus-values.yaml`](prometheus-values.yaml): the server, node-exporter and
kube-state-metrics (no Alertmanager or Pushgateway), a week of retention. Its Service
`monitoring/prometheus-server` is a name Cogpit's detection looks for. The chart's
`kubernetes-nodes-cadvisor` job scrapes the kubelet's cAdvisor every minute (the pods' charts, a
node's network and filesystem); its `kubernetes-service-endpoints` job scrapes node-exporter (a
node's CPU and memory history, the Overview's CPU and Memory) and kube-state-metrics (the Overview's
Pods history), each labelled with its node. Charts fill two or three minutes after the install; the
Overview's test wants half an hour of node-exporter's history.

```sh
make kind-prometheus        # install or upgrade (helm, --wait)
make test-kind              # TestKindPrometheus, TestKindNodeExporterByAddress, TestKindClusterOverview and TestKindPrometheusDrawer run instead of skipping
make kind-prometheus-down   # uninstall and delete the namespace monitoring
```

If an image pull stalls inside the kind node (seen 2026-10-14: node-exporter's stopped at 1 MB),
pull it on the host and load it: `docker pull <image>` then `kind load docker-image <image> --name
cogpit`; the kubelet pulls one image at a time, so a stalled pull holds the next pod too.

Through the service proxy as Cogpit reads it:

```sh
KUBECONFIG=.cache/kind/kubeconfig kubectl --context kind-cogpit get --raw \
  '/api/v1/namespaces/monitoring/services/prometheus-server:http/proxy/api/v1/status/buildinfo'
```

On this cluster's containerd, cAdvisor reports no `container_fs_usage_bytes` for containers, so a
pod's Filesystem chart shows read and write rates; the node's shows the usage of its disk. A kind
node's `/` is an overlay, which node-exporter leaves out, so the Nodes list's Disk column stays
empty here. node-exporter reads the Docker VM's `/proc`, the Metrics Server the node container's
cgroup: on kind a node's history (node-exporter's) and its live values (the Metrics Server's) do not
match.

## VictoriaMetrics

`make kind-vm` (after `make kind-prometheus`) adds VictoriaMetrics for one check of D-71,
`TestKindVictoriaMetrics`: single-node VictoriaMetrics (`cogpit-vm/vmsingle-cogpit`, port `http`
8428) and a small cluster, vmstorage, vminsert, vmselect (`cogpit-vm/vmselect-cogpit`, port `http`
8481, the Prometheus API under `/select/0/prometheus`) and a vmagent, in the namespace `cogpit-vm`
([`victoriametrics/`](victoriametrics/), plain manifests, images pinned to v1.106.1). The Services are
named as VictoriaMetrics' operator names a VMSingle's and a VMCluster's. Neither scrapes the cluster:
both take the kind Prometheus's series through its `/federate` endpoint every 30 seconds, so their
history starts when they do; run the test a few minutes after. Detection prefers the kind
Prometheus to them, so the test names each in the Prometheus setting.

```sh
make kind-vm        # apply the manifests and wait for the rollouts
make test-kind      # TestKindVictoriaMetrics runs instead of skipping
make kind-vm-down   # delete the namespace cogpit-vm
```

If the images do not pull inside the kind node, pull them on the host and `kind load docker-image
<image> --name cogpit`, as for Prometheus.

## Trivy Operator

`make kind-trivy` installs Trivy Operator for the image ratings' check (D-78,
`TestKindTrivy`): the aquasecurity `trivy-operator` chart (version pinned in the Makefile, fetched
with `--repo`), release `trivy` in the namespace `cogpit-trivy`, with
[`trivy-operator-values.yaml`](trivy-operator-values.yaml): vulnerability reports only (no
configuration audits, RBAC or infrastructure assessments, exposed secrets, SBOMs or compliance), of
the namespaces `cogpit-trivy-a` and `cogpit-trivy-b` alone, one scan job at a time, the
vulnerability database downloaded once by the chart's built-in Trivy server (a StatefulSet with a
2 GiB volume), reports kept until their workload changes. Then it applies [`trivy/`](trivy/): a
Deployment `mixed` of an init container (`busybox:1.37.0`) and two containers (`nginx:1.16.1`, an
old Debian image with known vulnerabilities, and `busybox:1.37.0`), a StatefulSet `store`
(`nginx:1.30.0-alpine`), a Job `once`, a suspended CronJob `nightly` (`alpine:3.12`) and a Job made
from it named as the CronJob controller names its Jobs, and a bare Pod `bare` (`alpine:3.12`) in
`cogpit-trivy-b`. The reports follow within a few minutes (2026-10-17: all seven after about three
minutes).

```sh
make kind-trivy        # install the operator and the workloads
make test-kind         # TestKindTrivy runs instead of skipping
make kind-trivy-down   # delete the workloads' namespaces, uninstall the operator, delete its
                       # aquasecurity.github.io definitions and the namespace cogpit-trivy
```

Trivy Operator's own table of the reports:

```sh
KUBECONFIG=.cache/kind/kubeconfig kubectl --context kind-cogpit get vulnerabilityreports -A -o wide
```

## Alertmanager

`make kind-alertmanager` adds one Alertmanager for the check of D-77, `TestKindAlertmanager`:
`quay.io/prometheus/alertmanager:v0.28.1`, its Service `cogpit-alerts/cogpit-alertmanager` (port
`http` 9093, named as the alertmanager chart names its own, so Cogpit finds it by name), every alert
routed to a receiver that sends nothing ([`alertmanager/`](alertmanager/), plain manifests). Nothing
fires alerts into it: the test posts its own through the API server's service proxy (as the
cluster's admin; Cogpit itself only reads) and resolves them at its end.

```sh
make kind-alertmanager        # apply the manifests and wait for the rollout
make test-kind                # TestKindAlertmanager runs instead of skipping
make kind-alertmanager-down   # delete the namespace cogpit-alerts
```

## Kyverno

`make kind-kyverno` installs Kyverno for the Kyverno integration's check (D-79, `TestKindKyverno`):
the admission and reports controllers of Kyverno's chart (3.9.1, Kyverno 1.19.1) in the namespace
`cogpit-kyverno` ([`kyverno/values.yaml`](kyverno/values.yaml)), its webhooks limited to the
namespaces labelled `cogpit-kyverno=true`, so the other kind tests never pass through it; then the
objects made before the policies ([`kyverno/before.yaml`](kyverno/before.yaml): the namespace
`cogpit-kyverno-a`, a ConfigMap without and one with an `owner` label, a Deployment and a Service),
the policies ([`kyverno/policies.yaml`](kyverno/policies.yaml): two ClusterPolicies in Audit, one in
Enforce with `failurePolicy: Ignore`, each matching only `cogpit-kyverno-*` namespaces, and a
ValidatingPolicy in Audit for the labelled namespaces) and an object made after them
([`kyverno/after.yaml`](kyverno/after.yaml)). The background scan runs every minute; the reports
follow within a minute of the install. Cogpit itself installs nothing: this is a test setup.

```sh
make kind-kyverno        # install Kyverno, the policies and the objects
make test-kind           # TestKindKyverno runs instead of skipping
make kind-kyverno-down   # remove the policies, both namespaces, Kyverno, its CRDs and webhooks
```

## The synthetic cluster

`make run-demo` needs no cluster at all: [`internal/demo`](../../internal/demo/demo.go) generates
10,000 pods, 500 deployments, 300 services, 50 nodes, 20 namespaces and 2,000 events in memory and
changes them at 1,000 watch events per second, through the same cache and informers as a real
cluster. It has its own well-known service: `demo-mcp/web` with ClusterIP `10.96.100.100`.

## The drain cluster

Cordon, uncordon and drain change a node, and the cluster above has only one: draining it would
stop every other test. Those tests run against a second cluster, `cogpit-drain`, with one control
plane and two workers ([`drain-cluster.yaml`](drain-cluster.yaml)); its kubeconfig is its own file,
`.cache/kind-drain/kubeconfig` (context `kind-cogpit-drain`).

```sh
make kind-drain-up     # create the cluster (or reuse it) and write .cache/kind-drain/kubeconfig
make test-kind-drain   # TestKindDrainCrashFinishUncordon, in the namespace cogpit-drain-test
make kind-drain-down   # delete the cluster and its kubeconfig file
```

The test drains the first worker by name: a Deployment's two pods, a pod a PodDisruptionBudget
guards and a DaemonSet's pod are put on it; Cogpit "crashes" while the budget blocks an eviction,
the next start offers the unfinished drain, finishing it fails while the budget stands and
succeeds once it is gone, and the node is uncordoned at the end (and on cleanup). It pulls
`busybox:1.37.0`. `make test-kind` skips it, because it runs only when
`COGPIT_KIND_DRAIN_KUBECONFIG` and `COGPIT_KIND_DRAIN_CONTEXT` are set, and it refuses the context
`kind-cogpit`.
