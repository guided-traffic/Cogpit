# Local kind cluster

A single-node [kind](https://kind.sigs.k8s.io) cluster named `cogpit` for running Cogpit against a
real API server, and a small seed of workloads to look at. The cluster's kubeconfig is its own file,
`.cache/kind/kubeconfig` (context `kind-cogpit`); nothing here reads or writes `~/.kube/config`,
so Cogpit and kubectl never reach another cluster by accident.

| File | What it is |
|---|---|
| [`cluster.yaml`](cluster.yaml) | kind configuration: one control-plane node, the default service CIDR `10.96.0.0/16` |
| [`seed/`](seed/) | the manifests `make kind-seed` applies, all in the namespace `cogpit-demo` |

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

## The synthetic cluster

`make run-demo` needs no cluster at all: [`internal/demo`](../../internal/demo/demo.go) generates
10,000 pods, 500 deployments, 300 services, 50 nodes, 20 namespaces and 2,000 events in memory and
changes them at 1,000 watch events per second, through the same cache and informers as a real
cluster. It has its own well-known service: `demo-mcp/web` with ClusterIP `10.96.100.100`.
