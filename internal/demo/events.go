package demo

import (
	"fmt"
	"time"
)

// Event sources.
const (
	sourceKubelet    = "kubelet"
	sourceScheduler  = "default-scheduler"
	sourceDeployment = "deployment-controller"
	sourceNode       = "node-controller"
)

// newEvent creates an Event about a random pod, deployment or node, with a reason that matches
// the object's state, last seen at last. Called with c.mu held.
func (c *Cluster) newEvent(last time.Time) *event {
	var e *event
	switch u := c.rng.Float64(); {
	case u < 0.15 && len(c.w.unstable) > 0:
		e = c.podEvent(pick(c, c.w.unstable))
	case u < 0.65 && len(c.w.pods) > 0:
		e = c.podEvent(pick(c, c.w.pods))
	case u < 0.85 && len(c.w.deploys) > 0:
		e = c.deploymentEvent(pick(c, c.w.deploys))
	default:
		e = c.nodeEvent(pick(c, c.w.nodes))
	}
	e.uid = c.newUID()
	e.count = max(1, e.count)
	e.last = last
	e.first = last.Add(-time.Duration(e.count-1) * 5 * time.Minute)
	for {
		e.name = fmt.Sprintf("%s.%s", e.involved["name"], c.hexString(16))
		if _, used := c.store[kindEvents][objKey(e.namespace, e.name)]; !used {
			return e
		}
	}
}

func (c *Cluster) involved(k kind, namespace, name, uid, fieldPath string) map[string]any {
	info := kindInfo[k]
	ref := reference(info.apiVersion, info.kind, namespace, name, uid)
	ref["resourceVersion"] = c.currentRV(k, namespace, name)
	if fieldPath != "" {
		ref["fieldPath"] = fieldPath
	}
	return ref
}

func (c *Cluster) podEvent(p *pod) *event {
	ct := p.tmpl.containers[mainContainer]
	field := "spec.containers{" + ct.name + "}"
	e := &event{namespace: p.namespace, typ: typeNormal, component: sourceKubelet}
	if p.node != nil {
		e.host = p.node.name
	}
	set := func(typ, reason, message, component, fieldPath string, count int64) {
		e.typ, e.reason, e.message, e.component, e.count = typ, reason, message, component, count
		e.involved = c.involved(kindPods, p.namespace, p.name, p.uid, fieldPath)
	}
	scheduled := func() {
		node := ""
		if p.node != nil {
			node = p.node.name
		}
		set(typeNormal, "Scheduled", fmt.Sprintf("Successfully assigned %s/%s to %s", p.namespace, p.name, node), sourceScheduler, "", 1)
		e.host = ""
	}
	switch {
	case p.phase == phaseCrashLoop:
		set(typeWarning, "BackOff", fmt.Sprintf("Back-off restarting failed container %s in pod %s_%s(%s)", ct.name, p.name, p.namespace, p.uid),
			sourceKubelet, field, max(1, p.restarts/2))
	case p.phase == phaseImagePull && c.chance(0.5):
		set(typeWarning, "Failed", fmt.Sprintf("Failed to pull image %q: %s", ct.image, pullError(ct.image)), sourceKubelet, field, int64(1+c.rng.IntN(30)))
	case p.phase == phaseImagePull:
		set(typeNormal, "BackOff", fmt.Sprintf("Back-off pulling image %q", ct.image), sourceKubelet, field, int64(10+c.rng.IntN(200)))
	case p.phase == phaseUnschedulable:
		set(typeWarning, "FailedScheduling", c.unschedulableMessage(), sourceScheduler, "", int64(1+c.rng.IntN(30)))
		e.host = ""
	case p.phase == phaseEvicted:
		set(typeWarning, "Evicted", "The node was low on resource: memory. Threshold quantity: 100Mi, available: 87232Ki.", sourceKubelet, "", 1)
	case p.phase == phaseCreating && c.chance(0.5):
		scheduled()
	case p.phase == phaseCreating:
		set(typeNormal, "Pulling", fmt.Sprintf("Pulling image %q", ct.image), sourceKubelet, field, 1)
	case !p.deleting.IsZero():
		set(typeNormal, "Killing", "Stopping container "+ct.name, sourceKubelet, field, 1)
	case p.phase == phaseRunning && !p.ready:
		set(typeWarning, "Unhealthy", "Readiness probe failed: HTTP probe failed with statuscode: 503", sourceKubelet, field, int64(1+c.rng.IntN(50)))
	default:
		switch c.rng.IntN(4) {
		case 0:
			scheduled()
		case 1:
			set(typeNormal, "Pulled", fmt.Sprintf("Container image %q already present on machine", ct.image), sourceKubelet, field, 1)
		case 2:
			set(typeNormal, "Created", "Created container: "+ct.name, sourceKubelet, field, 1)
		default:
			set(typeNormal, "Started", "Started container "+ct.name, sourceKubelet, field, 1)
		}
	}
	return e
}

func (c *Cluster) deploymentEvent(d *deployment) *event {
	rs := d.name + "-" + d.rsHash[d.current]
	to := d.replicas
	if d.profile == profileBadImage {
		to = 1
	}
	return &event{
		namespace: d.namespace, typ: typeNormal, reason: "ScalingReplicaSet", component: sourceDeployment, count: 1,
		message:  fmt.Sprintf("Scaled up replica set %s from 0 to %d", rs, to),
		involved: c.involved(kindDeployments, d.namespace, d.name, d.uid, ""),
	}
}

// nodeEvent is recorded in the default namespace, as the kubelet and node controller do.
func (c *Cluster) nodeEvent(n *node) *event {
	e := &event{namespace: saDefault, typ: typeNormal, component: sourceKubelet, host: n.name, count: 1,
		involved: c.involved(kindNodes, "", n.name, n.uid, "")}
	switch {
	case n.notReady && c.chance(0.5):
		e.reason, e.message, e.component, e.host = "NodeNotReady", "Node "+n.name+" status is now: NodeNotReady", sourceNode, ""
	case n.notReady:
		e.typ, e.reason, e.message, e.count = typeWarning, "EvictionThresholdMet", "Attempting to reclaim memory", int64(1+c.rng.IntN(20))
	case n.cordoned:
		e.reason, e.message = "NodeNotSchedulable", "Node "+n.name+" status is now: NodeNotSchedulable"
	default:
		switch c.rng.IntN(3) {
		case 0:
			e.reason, e.message = "Starting", "Starting kubelet."
		case 1:
			e.reason, e.message = "NodeReady", "Node "+n.name+" status is now: NodeReady"
		default:
			e.reason, e.message, e.component, e.host = "RegisteredNode", "Node "+n.name+" event: Registered Node "+n.name+" in Controller", sourceNode, ""
		}
	}
	return e
}
