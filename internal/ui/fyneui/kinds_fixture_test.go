package fyneui

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Synthetic Pods and Nodes rendered through the real kind profiles, so that the list and drawer
// PNGs show what the profiles produce: warning icons, links, a cordoned node's Conditions. Every
// name and address is made up.

// ts formats a time before fixedNow for an object field.
func ts(ago time.Duration) string { return fixedNow.Add(-ago).UTC().Format(time.RFC3339) }

// podState is one of the states the fixture pods cycle through.
type podState string

const (
	podOK          podState = "ok"
	podCrashLoop   podState = "crashloop"
	podNotReady    podState = "notready"
	podCompleted   podState = "completed"
	podUnscheduled podState = "unschedulable"
	podPullBackOff podState = "imagepull"
	podDeleting    podState = "terminating"
)

type podSpec struct {
	ns, name   string
	ownerKind  string // ReplicaSet, Job, DaemonSet, StatefulSet, Node
	ownerName  string
	node       string
	state      podState
	containers []string
	init       []string
	age        time.Duration
}

func fixturePodObject(p podSpec) *unstructured.Unstructured {
	meta := map[string]any{
		"name": p.name, "namespace": p.ns, "uid": "uid-" + p.ns + "-" + p.name,
		"creationTimestamp": ts(p.age),
		"labels":            map[string]any{"app": strings.Split(p.name, "-")[0], "tier": "backend"},
		"annotations":       map[string]any{"prometheus.io/scrape": "true", "prometheus.io/port": "9102"},
	}
	if p.ownerKind != "" {
		apiVersion := map[string]string{"ReplicaSet": "apps/v1", "StatefulSet": "apps/v1", "DaemonSet": "apps/v1", "Job": "batch/v1", "Node": "v1"}[p.ownerKind]
		meta["ownerReferences"] = []any{map[string]any{
			"apiVersion": apiVersion, "kind": p.ownerKind, "name": p.ownerName, "uid": "uid-owner-" + p.ownerName, "controller": true,
		}}
	}
	if p.state == podDeleting {
		meta["deletionTimestamp"] = ts(20 * time.Second)
	}
	var specContainers, statuses []any
	for i, c := range p.containers {
		specContainers = append(specContainers, map[string]any{
			"name": c, "image": "registry.example.com/" + p.ns + "/" + c + ":2.14." + fmt.Sprint(i),
			"ports": []any{map[string]any{"name": "http", "containerPort": int64(8080 + i), "protocol": "TCP"}},
			"env": []any{
				map[string]any{"name": "LOG_LEVEL", "value": "info"},
				map[string]any{"name": "DB_PASSWORD", "valueFrom": map[string]any{"secretKeyRef": map[string]any{"name": p.ns + "-db", "key": "password"}}},
			},
			"volumeMounts": []any{map[string]any{"name": "config", "mountPath": "/etc/" + c, "readOnly": true}},
			"resources": map[string]any{
				"requests": map[string]any{"cpu": "100m", "memory": "128Mi"},
				"limits":   map[string]any{"memory": "256Mi"},
			},
		})
		st := map[string]any{"name": c, "ready": true, "started": true, "restartCount": int64(0),
			"image": "registry.example.com/" + p.ns + "/" + c + ":2.14." + fmt.Sprint(i),
			"state": map[string]any{"running": map[string]any{"startedAt": ts(p.age - time.Minute)}}}
		switch {
		case p.state == podCrashLoop && i == 0:
			st["ready"], st["started"], st["restartCount"] = false, false, int64(412)
			st["state"] = map[string]any{"waiting": map[string]any{"reason": "CrashLoopBackOff", "message": "back-off 5m0s restarting failed container"}}
			st["lastState"] = map[string]any{"terminated": map[string]any{"exitCode": int64(1), "reason": "Error", "startedAt": ts(3 * time.Minute), "finishedAt": ts(2 * time.Minute)}}
		case p.state == podNotReady && i == len(p.containers)-1:
			st["ready"], st["restartCount"] = false, int64(3)
		case p.state == podCompleted:
			st["ready"] = false
			st["state"] = map[string]any{"terminated": map[string]any{"exitCode": int64(0), "reason": "Completed", "startedAt": ts(p.age - time.Minute), "finishedAt": ts(p.age - 2*time.Minute)}}
		case p.state == podPullBackOff:
			st["ready"], st["started"] = false, false
			st["state"] = map[string]any{"waiting": map[string]any{"reason": "ImagePullBackOff", "message": "Back-off pulling image"}}
		}
		statuses = append(statuses, st)
	}
	var initSpec, initStatuses []any
	for _, c := range p.init {
		initSpec = append(initSpec, map[string]any{"name": c, "image": "registry.example.com/" + p.ns + "/" + c + ":1.0", "command": []any{"/bin/migrate", "--wait"}})
		initStatuses = append(initStatuses, map[string]any{"name": c, "ready": true, "restartCount": int64(0),
			"state": map[string]any{"terminated": map[string]any{"exitCode": int64(0), "reason": "Completed"}}})
	}
	spec := map[string]any{
		"containers": specContainers, "serviceAccountName": strings.Split(p.name, "-")[0],
		"tolerations": []any{
			map[string]any{"key": "node.kubernetes.io/not-ready", "operator": "Exists", "effect": "NoExecute", "tolerationSeconds": int64(300)},
			map[string]any{"key": "node.kubernetes.io/unreachable", "operator": "Exists", "effect": "NoExecute", "tolerationSeconds": int64(300)},
		},
		"volumes": []any{
			map[string]any{"name": "config", "configMap": map[string]any{"name": p.ns + "-config"}},
			map[string]any{"name": "kube-api-access", "projected": map[string]any{"sources": []any{}}},
		},
	}
	if len(initSpec) > 0 {
		spec["initContainers"] = initSpec
	}
	if p.node != "" {
		spec["nodeName"] = p.node
	}
	phase, ready := "Running", "True"
	conds := []any{}
	switch p.state {
	case podCompleted:
		phase, ready = "Succeeded", "False"
	case podUnscheduled:
		phase = "Pending"
		statuses = nil
		conds = append(conds, map[string]any{"type": "PodScheduled", "status": "False", "reason": "Unschedulable",
			"message": "0/9 nodes are available: 3 Insufficient cpu, 1 node(s) were unschedulable, 5 node(s) had untolerated taint(s)."})
	case podCrashLoop, podNotReady, podPullBackOff:
		ready = "False"
	}
	if p.state != podUnscheduled {
		conds = append(conds,
			map[string]any{"type": "Initialized", "status": "True"},
			map[string]any{"type": "Ready", "status": ready},
			map[string]any{"type": "ContainersReady", "status": ready},
			map[string]any{"type": "PodScheduled", "status": "True"})
	}
	status := map[string]any{"phase": phase, "qosClass": "Burstable", "conditions": conds}
	if statuses != nil {
		status["containerStatuses"] = statuses
		status["initContainerStatuses"] = initStatuses
		status["podIP"] = "10.244." + fmt.Sprint(len(p.name)%7) + "." + fmt.Sprint(10+len(p.ns))
		status["podIPs"] = []any{map[string]any{"ip": status["podIP"]}, map[string]any{"ip": "fd00:10:244::" + fmt.Sprint(len(p.name))}}
		status["hostIP"] = "192.0.2." + fmt.Sprint(10+len(p.node))
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Pod", "metadata": meta, "spec": spec, "status": status,
	}}
}

// fixturePodSpecs is a deterministic mix of pod states over the fixture's apps and nodes.
func fixturePodSpecs() []podSpec {
	states := []podState{podOK, podOK, podCrashLoop, podOK, podNotReady, podOK, podCompleted, podOK, podUnscheduled, podOK, podPullBackOff, podOK, podDeleting, podOK}
	nodes := []string{"worker-a-01", "worker-a-02", "worker-a-04", "worker-b-01", "worker-c-01"}
	hashes := []string{"7d9f8c6b5", "5c4b8d7f9", "6f8b9c7d4", "8b7c6d5e4", "9d8c7b6a5"}
	var out []podSpec
	for i := range 36 {
		app := fixtureApps[i%len(fixtureApps)]
		ns := fixtureNamespaces[(i*7)%len(fixtureNamespaces)]
		rs := app + "-" + hashes[i%len(hashes)]
		p := podSpec{
			ns: ns, name: fmt.Sprintf("%s-%c%c%c%c%c", rs, 'b'+rune(i%19), 'k'+rune(i%11), 'm'+rune(i%7), 'q'+rune(i%5), 'x'+rune(i%3)),
			ownerKind: "ReplicaSet", ownerName: rs, node: nodes[i%len(nodes)], state: states[i%len(states)],
			containers: []string{"app"}, age: time.Duration(1+i*37%900) * time.Hour,
		}
		if i%3 == 0 {
			p.containers = []string{"app", "envoy"}
		}
		if i%5 == 1 {
			p.init = []string{"migrate"}
		}
		switch p.state {
		case podCompleted:
			p.ownerKind, p.ownerName = "Job", app+"-29384715"
		case podUnscheduled:
			p.node = ""
		}
		out = append(out, p)
	}
	out = append(out,
		podSpec{ns: "kube-system", name: "kube-proxy-x7k2p", ownerKind: "DaemonSet", ownerName: "kube-proxy", node: "worker-a-03", state: podOK, containers: []string{"kube-proxy"}, age: 160 * 24 * time.Hour},
		podSpec{ns: "kube-system", name: "kube-apiserver-control-plane-1", ownerKind: "Node", ownerName: "control-plane-1", node: "control-plane-1", state: podOK, containers: []string{"kube-apiserver"}, age: 160 * 24 * time.Hour},
		podSpec{ns: "orders", name: "ledger-writer-0", ownerKind: "StatefulSet", ownerName: "ledger-writer", node: "worker-b-01", state: podOK, containers: []string{"app"}, age: 12 * 24 * time.Hour},
	)
	return out
}

type nodeSpec struct {
	name                         string
	role                         string
	cordoned                     bool
	ready                        string // "True", "False", "Unknown"
	memoryPressure, diskPressure bool
}

func fixtureNodeObject(n nodeSpec) *unstructured.Unstructured {
	labels := map[string]any{
		"kubernetes.io/hostname": n.name, "kubernetes.io/os": "linux", "kubernetes.io/arch": "amd64",
		"node-role.kubernetes.io/" + n.role: "", "topology.kubernetes.io/zone": "zone-" + n.name[len(n.name)-1:],
	}
	var taints []any
	if n.role == "control-plane" {
		taints = append(taints, map[string]any{"key": "node-role.kubernetes.io/control-plane", "effect": "NoSchedule"})
	}
	spec := map[string]any{"podCIDR": "10.244.0.0/24"}
	if n.cordoned {
		spec["unschedulable"] = true
		taints = append(taints, map[string]any{"key": "node.kubernetes.io/unschedulable", "effect": "NoSchedule", "timeAdded": ts(3 * time.Hour)})
	}
	if taints != nil {
		spec["taints"] = taints
	}
	cond := func(typ, status, reason, msg string) map[string]any {
		return map[string]any{"type": typ, "status": status, "reason": reason, "message": msg, "lastTransitionTime": ts(48 * time.Hour)}
	}
	boolStatus := func(b bool) string {
		if b {
			return "True"
		}
		return "False"
	}
	readyReason := map[string]string{"True": "KubeletReady", "False": "KubeletNotReady", "Unknown": "NodeStatusUnknown"}[n.ready]
	resources := func(cpu, mem, pods string) map[string]any {
		return map[string]any{"cpu": cpu, "memory": mem, "ephemeral-storage": "203056560Ki", "hugepages-1Gi": "0", "hugepages-2Mi": "0", "pods": pods}
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Node",
		"metadata": map[string]any{
			"name": n.name, "uid": "uid-node-" + n.name, "creationTimestamp": ts(160 * 24 * time.Hour), "labels": labels,
			"annotations": map[string]any{"node.alpha.kubernetes.io/ttl": "0", "volumes.kubernetes.io/controller-managed-attach-detach": "true"},
		},
		"spec": spec,
		"status": map[string]any{
			"addresses": []any{
				map[string]any{"type": "InternalIP", "address": "192.0.2." + fmt.Sprint(10+len(n.name))},
				map[string]any{"type": "Hostname", "address": n.name},
			},
			"conditions": []any{
				cond("MemoryPressure", boolStatus(n.memoryPressure), "KubeletHasSufficientMemory", "kubelet has sufficient memory available"),
				cond("DiskPressure", boolStatus(n.diskPressure), "KubeletHasNoDiskPressure", "kubelet has no disk pressure"),
				cond("PIDPressure", "False", "KubeletHasSufficientPID", "kubelet has sufficient PID available"),
				cond("Ready", n.ready, readyReason, "kubelet is posting ready status"),
			},
			"nodeInfo": map[string]any{
				"architecture": "amd64", "operatingSystem": "linux", "osImage": "Ubuntu 24.04.3 LTS", "kernelVersion": "6.8.0-85-generic",
				"containerRuntimeVersion": "containerd://2.1.4", "kubeletVersion": "v1.37.1", "kubeProxyVersion": "v1.37.1",
			},
			"capacity":    resources("8", "32863640Ki", "110"),
			"allocatable": resources("7800m", "31715736Ki", "110"),
		},
	}}
}

func fixtureNodeSpecs() []nodeSpec {
	return []nodeSpec{
		{name: "control-plane-1", role: "control-plane", ready: "True"},
		{name: "worker-a-01", role: "worker", ready: "True"},
		{name: "worker-a-02", role: "worker", ready: "True"},
		{name: "worker-a-03", role: "worker", ready: "True", cordoned: true},
		{name: "worker-a-04", role: "worker", ready: "True"},
		{name: "worker-b-01", role: "worker", ready: "True"},
		{name: "worker-b-02", role: "worker", ready: "False", memoryPressure: true},
		{name: "worker-b-03", role: "worker", ready: "Unknown"},
		{name: "worker-c-01", role: "worker", ready: "True", diskPressure: true},
	}
}

// kindSnapshot renders objects through the built-in profile of a kind into a table snapshot,
// sorted by name, as the engine's view would.
func kindSnapshot(t testing.TB, key string, objs []*unstructured.Unstructured) *engine.TableSnapshot {
	return kindSnapshotWith(t, key, objs, nil)
}

// kindSnapshotWith is kindSnapshot with the related objects of the profile's Needs.
func kindSnapshotWith(t testing.TB, key string, objs []*unstructured.Unstructured, rel model.Related) *engine.TableSnapshot {
	t.Helper()
	p, ok := kinds.ByKey(key)
	if !ok {
		t.Fatalf("no built-in profile %s", key)
	}
	rows := make([]model.Row, 0, len(objs))
	for _, o := range objs {
		rows = append(rows, model.BuildRowWith(p, "demo", o, rel))
	}
	slices.SortFunc(rows, func(a, b model.Row) int { return strings.Compare(a.Ref.Name, b.Ref.Name) })
	return &engine.TableSnapshot{
		View: key, Context: "demo", Title: p.Title, Namespaced: p.Namespaced, Columns: p.Columns, Rows: rows,
		Sort: engine.SortSpec{Column: "name"}, Total: len(rows), Revision: 1, Synced: true,
	}
}

func fixturePodObjects() []*unstructured.Unstructured {
	specs := fixturePodSpecs()
	out := make([]*unstructured.Unstructured, 0, len(specs))
	for _, s := range specs {
		out = append(out, fixturePodObject(s))
	}
	return out
}

func fixtureNodeObjects() []*unstructured.Unstructured {
	specs := fixtureNodeSpecs()
	out := make([]*unstructured.Unstructured, 0, len(specs))
	for _, s := range specs {
		out = append(out, fixtureNodeObject(s))
	}
	return out
}

// fixtureEventObjects are core v1 events about some of the fixture pods and nodes.
func fixtureEventObjects() []*unstructured.Unstructured {
	pods := fixturePodSpecs()
	var out []*unstructured.Unstructured
	add := func(ns, name, kind, apiVersion, objNS, objName, typ, reason, msg string, count int64, ago time.Duration) {
		out = append(out, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "v1", "kind": "Event",
			"metadata": map[string]any{"name": name, "namespace": ns, "uid": "uid-ev-" + name, "creationTimestamp": ts(ago + time.Hour)},
			"involvedObject": map[string]any{"apiVersion": apiVersion, "kind": kind, "namespace": objNS, "name": objName,
				"uid": "uid-" + objNS + "-" + objName},
			"type": typ, "reason": reason, "message": msg, "count": count,
			"firstTimestamp": ts(ago + time.Hour), "lastTimestamp": ts(ago),
			"source": map[string]any{"component": "kubelet", "host": "worker-a-01"},
		}})
	}
	for i, p := range pods {
		ev := fmt.Sprintf("%s.%x", p.name, 0x17a3c0+i)
		switch p.state {
		case podCrashLoop:
			add(p.ns, ev, "Pod", "v1", p.ns, p.name, "Warning", "BackOff",
				"Back-off restarting failed container app in pod "+p.name+"_"+p.ns+"(uid-"+p.ns+"-"+p.name+")", 412, 2*time.Minute)
			add(p.ns, ev+"1", "Pod", "v1", p.ns, p.name, "Normal", "Pulled",
				"Container image \"registry.example.com/"+p.ns+"/app:2.14.0\" already present on machine and was not pulled again", 413, 7*time.Minute)
		case podNotReady:
			add(p.ns, ev, "Pod", "v1", p.ns, p.name, "Warning", "Unhealthy", "Readiness probe failed: HTTP probe failed with statuscode: 503", 27, 40*time.Second)
		case podUnscheduled:
			add(p.ns, ev, "Pod", "v1", p.ns, p.name, "Warning", "FailedScheduling",
				"0/9 nodes are available: 3 Insufficient cpu, 1 node(s) were unschedulable, 5 node(s) had untolerated taint(s). "+
					"preemption: 0/9 nodes are available: 3 No preemption victims found for incoming pod, 6 Preemption is not helpful for scheduling.", 18, 3*time.Minute)
		case podOK:
			if i < 6 {
				add(p.ns, ev, "Pod", "v1", p.ns, p.name, "Normal", "Started", "Started container app", 1, p.age-time.Minute)
			}
		}
	}
	add("default", "worker-a-03.nns", "Node", "v1", "", "worker-a-03", "Normal", "NodeNotSchedulable", "Node worker-a-03 status is now: NodeNotSchedulable", 1, 3*time.Hour)
	add("default", "worker-b-02.nnr", "Node", "v1", "", "worker-b-02", "Warning", "NodeNotReady", "Node worker-b-02 status is now: NodeNotReady", 3, 25*time.Minute)
	add("orders", "ledger-writer.scale", "StatefulSet", "apps/v1", "orders", "ledger-writer", "Normal", "SuccessfulCreate",
		"create Pod ledger-writer-0 in StatefulSet ledger-writer successful", 1, 12*24*time.Hour)
	add("cert-manager", "tls.issuing", "Certificate", "cert-manager.io/v1", "cert-manager", "storefront-tls", "Normal", "Issuing",
		"The certificate has been successfully issued", 1, 26*time.Hour)
	return out
}

// fixtureDeploymentObjects are three Deployments, one of them scaled to zero.
func fixtureDeploymentObjects() []*unstructured.Unstructured {
	var out []*unstructured.Unstructured
	for i, d := range []struct {
		ns, name        string
		replicas, ready int64
	}{{"checkout", "checkout-api", 3, 3}, {"payments", "payments-worker", 2, 1}, {"qa", "image-resizer", 0, 0}} {
		out = append(out, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "apps/v1", "kind": "Deployment",
			"metadata": map[string]any{"name": d.name, "namespace": d.ns, "uid": "uid-dep-" + d.name, "generation": int64(2),
				"creationTimestamp": ts(time.Duration(20+i) * 24 * time.Hour)},
			"spec": map[string]any{"replicas": d.replicas, "selector": map[string]any{"matchLabels": map[string]any{"app": d.name}}},
			"status": map[string]any{"observedGeneration": int64(2), "replicas": d.replicas, "readyReplicas": d.ready,
				"availableReplicas": d.ready, "updatedReplicas": d.replicas},
		}})
	}
	return out
}

// kindsFake is a fake connected to the demo context whose Pods, Nodes, Deployments and Events
// come from the synthetic objects above, through the real profiles; their drawers are the real
// drawer models (engine.BuildDetailWith), with the fixture's pods as related objects (a node's
// Pods).
func kindsFake(t testing.TB, view string) *fakeEngine {
	f := connectedFake(t, 1)
	// The owners' kinds too, so that Controlled By and Involved Object links lead somewhere.
	f.kinds = append(f.kinds,
		kindInfo{Kind: "replicasets", Title: "Replica Sets", APIKind: "ReplicaSet", Group: "apps", Version: "v1", Namespaced: true},
		kindInfo{Kind: "statefulsets", Title: "Stateful Sets", APIKind: "StatefulSet", Group: "apps", Version: "v1", Namespaced: true},
		kindInfo{Kind: "daemonsets", Title: "Daemon Sets", APIKind: "DaemonSet", Group: "apps", Version: "v1", Namespaced: true},
		kindInfo{Kind: "jobs", Title: "Jobs", APIKind: "Job", Group: "batch", Version: "v1", Namespaced: true},
		kindInfo{Kind: "configmaps", Title: "Config Maps", APIKind: "ConfigMap", Version: "v1", Namespaced: true})
	events := fixtureEventObjects()
	pods := fixturePodObjects()
	rel := fixtureRelated{"pods": pods}
	for key, objs := range map[string][]*unstructured.Unstructured{
		"pods": pods, "nodes": fixtureNodeObjects(), "deployments": fixtureDeploymentObjects(), "events": events,
	} {
		f.tables[key] = kindSnapshot(t, key, objs)
		p, _ := kinds.ByKey(key)
		for _, o := range objs {
			ref := model.ObjectRef{Context: "demo", Kind: key, Namespace: o.GetNamespace(), Name: o.GetName()}
			m, err := engine.BuildDetailWith(p, ref, o, events, fixedNow, rel)
			if err != nil {
				t.Fatal(err)
			}
			engine.SortDrawerTables(m, nil) // as the session's models come: a node's Pods by name
			f.details[ref.String()] = m
		}
	}
	f.ui.View = view
	return f
}

// rowOf returns the index of the named object in a view's fixture snapshot.
func rowOf(t testing.TB, f *fakeEngine, view, name string) int {
	t.Helper()
	for i, r := range f.tables[view].Rows {
		if r.Ref.Name == name {
			return i
		}
	}
	t.Fatalf("no %s %s in the fixture", view, name)
	return -1
}

// podNamed returns the fixture pod spec in a state, the first one.
func podNamed(t testing.TB, state podState) podSpec {
	t.Helper()
	for _, p := range fixturePodSpecs() {
		if p.state == state {
			return p
		}
	}
	t.Fatalf("no fixture pod in state %s", state)
	return podSpec{}
}
