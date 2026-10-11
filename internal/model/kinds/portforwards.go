package kinds

import (
	"encoding/json"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The port forwards are a local kind (architecture §5, "Scopes, local and virtual kinds"): Cogpit
// makes the objects itself, one per forward, and the engine's forward manager writes them into
// the backend's local store (engine/portforward.go). Their shape:
//
//	apiVersion: cogpit.io/v1
//	kind: PortForward
//	metadata: {name: pod-web-0-8080, namespace: shop, uid, creationTimestamp}
//	spec:
//	  kind: Pod                # or Service: what is forwarded
//	  name: web-0              # the pod or service
//	  container: app           # the container the port belongs to, if known
//	  port: 8080               # the pod's port, or the service's
//	  localPort: 0             # the local port asked for; 0 picks a free one
//	  protocol: http           # http or https: the scheme of the forward's URL
//	  openInBrowser: true
//	status:
//	  phase: Active            # Starting, Active, Waiting, Reconnecting, Stopped, Failed
//	  message: ...             # why it waits, reconnects or failed
//	  localPort: 54321         # the port it listens on, on 127.0.0.1
//	  address: 127.0.0.1:54321
//	  url: http://localhost:54321
//	  pod: web-7c9f-x2l4q      # the pod it is connected to (a service's pod)
//	  podPort: 8080            # the port in that pod (a service's target port)

// keyPortForwards is the port forwards' kind key.
const keyPortForwards = "portforwards"

// Column ids (and fields) of the port forwards.
const (
	idForwardKind = "kind"
	idLocalPort   = "localPort"
)

// Port forward phases, as status.phase holds them.
const (
	forwardActive       = "Active"
	forwardStarting     = "Starting"
	forwardWaiting      = "Waiting"
	forwardReconnecting = "Reconnecting"
	forwardFailed       = "Failed"
)

var portForwards = &model.KindProfile{
	Key:        keyPortForwards,
	Group:      model.LocalGroup,
	Version:    "v1",
	Resource:   keyPortForwards,
	Kind:       "PortForward",
	Title:      "Port Forwarding",
	Namespaced: true,
	Local:      true,
	// The columns as captured (resource-parity › Network › Port Forwarding).
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: idForwardKind, Title: "Kind", Type: model.ColumnText},
		{ID: "podPort", Title: "Pod Port", Type: model.ColumnNumber},
		{ID: idLocalPort, Title: "Local Port", Type: model.ColumnNumber},
		{ID: "protocol", Title: "Protocol", Type: model.ColumnText},
		statusColumn,
	},
	Cells: portForwardCells,
	Fields: map[string]fieldFunc{
		"target": func(obj *unstructured.Unstructured) []string {
			return []string{model.NestedString(obj, "spec", "name")}
		},
		idLocalPort: func(obj *unstructured.Unstructured) []string {
			return appendUnique(nil, model.NestedIntOrString(obj, "status", idLocalPort))
		},
		"pod": func(obj *unstructured.Unstructured) []string {
			return appendUnique(nil, model.NestedString(obj, "status", "pod"))
		},
	},
	Detail: portForwardDetail,
}

// portForwardCells: Name is the forwarded pod's or service's name (the object's own name also
// carries its kind and port, to be unique); Local Port the port it listens on, else the one asked
// for.
func portForwardCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	phase, token := forwardPhase(obj)
	port, _ := model.NestedInt64(obj, "spec", "port")
	return []model.Cell{
		{Text: model.NestedString(obj, "spec", "name")},
		namespaceCell(obj),
		{Text: model.NestedString(obj, "spec", idForwardKind)},
		{Text: model.NestedIntOrString(obj, "spec", "port"), Num: port},
		forwardLocalPort(obj),
		{Text: model.NestedString(obj, "spec", "protocol")},
		{Text: phase, Status: token},
	}, token
}

// forwardLocalPort is the port a forward listens on, else the one asked for, else "-": a free
// port is picked when it starts.
func forwardLocalPort(obj any) model.Cell {
	for _, f := range [][]string{{"status", idLocalPort}, {"spec", idLocalPort}} {
		if n, ok := model.NestedInt64(obj, f...); ok && n > 0 {
			return model.Cell{Text: itoa(n), Num: n}
		}
	}
	return model.Cell{Text: "-"}
}

// forwardPhase is the phase and its token (inferred: Lens's status texts are not captured).
func forwardPhase(obj any) (string, model.Status) {
	phase := model.NestedString(obj, "status", "phase")
	switch phase {
	case forwardActive:
		return phase, model.StatusOK
	case forwardStarting:
		return phase, model.StatusPending
	case forwardWaiting, forwardReconnecting:
		return phase, model.StatusWarning
	case forwardFailed:
		return phase, model.StatusError
	}
	return phase, model.StatusNone
}

// portForwardDetail: what is forwarded (a link to it), where it listens and the URL, which opens
// the forward in the browser (portforward.open), and its state.
func portForwardDetail(obj *unstructured.Unstructured) []model.DetailSection {
	ns := model.NestedString(obj, "metadata", "namespace")
	kind, name := model.NestedString(obj, "spec", idForwardKind), model.NestedString(obj, "spec", "name")
	var d detailItems
	switch kind {
	case "Pod":
		d.put(linkedItem("Pod", name, linkTo(podsResource, ns, name)))
		d.add("Container", model.NestedString(obj, "spec", "container"))
		d.add("Pod Port", model.NestedIntOrString(obj, "spec", "port"))
	case "Service":
		d.put(linkedItem("Service", name, linkTo("services", ns, name)))
		d.add("Service Port", model.NestedIntOrString(obj, "spec", "port"))
		pod := model.NestedString(obj, "status", "pod")
		d.put(linkedItem("Pod", pod, linkTo(podsResource, ns, pod)))
		d.add("Pod Port", model.NestedIntOrString(obj, "status", "podPort"))
	}
	if lp := forwardLocalPort(obj); lp.Num > 0 {
		d.add("Local Port", lp.Text)
	} else {
		d.add("Local Port", "Random")
	}
	d.add("Address", model.NestedString(obj, "status", "address"))
	if url := model.NestedString(obj, "status", "url"); url != "" {
		params, _ := json.Marshal(map[string]any{"ref": model.ObjectRef{Kind: keyPortForwards, Namespace: ns, Name: model.NestedString(obj, "metadata", "name")}})
		d.put(model.DetailItem{Name: "URL", Value: url, Actions: []model.ItemAction{
			{Label: "Open", Command: "portforward.open", Params: params, Link: true},
		}})
	}
	d.add("Protocol", model.NestedString(obj, "spec", "protocol"))
	phase, token := forwardPhase(obj)
	d.addStatus(titleStatus, phase, token)
	d.add("Message", model.NestedString(obj, "status", "message"))
	return []model.DetailSection{{Title: model.TitleProperties, Items: d}}
}
