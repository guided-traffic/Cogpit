package kinds

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// endpoints has the columns of Lens's documentation (not captured): Name, Namespace, Endpoints,
// Age. Endpoints lists the ready addresses with each port, "10.244.1.5:8080", the first three and
// "+ N more", as kubectl prints them (inferred). No status token, as for endpoint slices.
//
// v1 Endpoints are deprecated since Kubernetes 1.33 in favour of endpoint slices; the API server
// answers every request for them with a deprecation warning, which the cluster package logs once
// per connection (internal/cluster/warnings.go).
var endpoints = &model.KindProfile{
	Key:        keyEndpoints,
	Version:    "v1",
	Resource:   keyEndpoints,
	Kind:       "Endpoints",
	Title:      titleEndpoints,
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: keyEndpoints, Title: titleEndpoints, Type: model.ColumnText},
		ageColumn,
	},
	Cells: endpointsCells,
	Fields: map[string]fieldFunc{
		idAddress: func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, a := range endpointsAddresses(obj) {
				out = appendUnique(out, model.NestedString(a.addr, "ip"))
			}
			return out
		},
		idPort: func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, s := range model.NestedSlice(obj, "subsets") {
				for _, p := range model.NestedSlice(s, "ports") {
					out = appendUnique(out, model.NestedIntOrString(p, "port"))
				}
			}
			return out
		},
		idNode: func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, a := range endpointsAddresses(obj) {
				out = appendUnique(out, model.NestedString(a.addr, "nodeName"))
			}
			return out
		},
		idTarget: func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, a := range endpointsAddresses(obj) {
				out = appendUnique(out, targetText(model.NestedMap(a.addr, "targetRef")))
			}
			return out
		},
	},
	Detail: endpointsDetail,
}

func endpointsCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: shortList(endpointsReady(obj))},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// endpointsReady renders each ready address with each port of its subset, "10.244.1.5:8080"; an
// address of a subset without ports is the bare IP.
func endpointsReady(obj any) []string {
	var out []string
	for _, s := range model.NestedSlice(obj, "subsets") {
		ports := model.NestedSlice(s, "ports")
		for _, a := range model.NestedSlice(s, "addresses") {
			ip := model.NestedString(a, "ip")
			if ip == "" {
				continue
			}
			if len(ports) == 0 {
				out = append(out, ip)
				continue
			}
			for _, p := range ports {
				if port := model.NestedIntOrString(p, "port"); port != "" {
					out = append(out, ip+":"+port)
				}
			}
		}
	}
	return out
}

// endpointsAddress is one address of a subset, with whether it is ready and the subset's ports.
type endpointsAddress struct {
	addr  any
	ready bool
	ports []any
}

// endpointsAddresses lists the addresses of every subset, the ready ones of a subset first.
func endpointsAddresses(obj any) []endpointsAddress {
	var out []endpointsAddress
	for _, s := range model.NestedSlice(obj, "subsets") {
		ports := model.NestedSlice(s, "ports")
		for _, a := range model.NestedSlice(s, "addresses") {
			out = append(out, endpointsAddress{addr: a, ready: true, ports: ports})
		}
		for _, a := range model.NestedSlice(s, "notReadyAddresses") {
			out = append(out, endpointsAddress{addr: a, ports: ports})
		}
	}
	return out
}

// endpointsAddressColumns are the columns of the drawer's addresses table (inferred).
var endpointsAddressColumns = []model.Column{
	{ID: idAddress, Title: "Address", Type: model.ColumnText},
	{ID: fieldReady, Title: nodeReady, Type: model.ColumnText},
	{ID: fieldHostname, Title: titleHostname, Type: model.ColumnText},
	{ID: idNode, Title: titleNode, Type: model.ColumnText},
	{ID: idTarget, Title: "Target", Type: model.ColumnText},
	{ID: idPorts, Title: titlePorts, Type: model.ColumnText},
}

// endpointsDetail is Cogpit's drawer for an Endpoints object (inferred, Lens's is not captured):
// every address of every subset as a table, with its readiness, node and target as links, and the
// ports of its subset.
func endpointsDetail(obj *unstructured.Unstructured) []model.DetailSection {
	addrs := endpointsAddresses(obj)
	if len(addrs) == 0 {
		return nil
	}
	ns := obj.GetNamespace()
	rows := make([][]model.Cell, 0, min(len(addrs), maxEndpointRows+1))
	for i, a := range addrs {
		if i == maxEndpointRows {
			rows = append(rows, []model.Cell{{Text: "… " + itoa(int64(len(addrs)-i)) + " more"}})
			break
		}
		ready := model.Cell{Text: nodeReady, Status: model.StatusOK}
		if !a.ready {
			ready = model.Cell{Text: "Not ready", Status: model.StatusWarning}
		}
		node := model.NestedString(a.addr, "nodeName")
		rows = append(rows, []model.Cell{
			{Text: model.NestedString(a.addr, "ip")},
			ready,
			{Text: model.NestedString(a.addr, fieldHostname)},
			{Text: node, Link: linkTo(keyNodes, "", node)},
			targetCell(model.NestedMap(a.addr, "targetRef"), ns),
			{Text: strings.Join(slicePortLines(a.ports), ", ")},
		})
	}
	return []model.DetailSection{{Title: "Addresses", Table: model.CellTable(endpointsAddressColumns, rows)}}
}
