package kinds

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// leases has the columns of Lens's documentation (not captured): Name, Namespace, Holder, Age.
// Inferred: Holder as spec.holderIdentity, a link to the node for a node's heartbeat lease, no
// status token (whether a lease has expired depends on the clock, and rows are rebuilt only when
// the lease changes), and the drawer (the holder, the lease's times and transitions, the
// coordinated leader election fields).
var leases = &model.KindProfile{
	Key:        "leases",
	Group:      "coordination.k8s.io",
	Version:    "v1",
	Resource:   "leases",
	Kind:       "Lease",
	Title:      "Leases",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "holder", Title: "Holder", Type: model.ColumnText, Width: 280},
		ageColumn,
	},
	Cells: leaseCells,
	Fields: map[string]fieldFunc{
		"holder": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "spec", "holderIdentity"))
		},
	},
	Detail: leaseDetail,
}

// nodeLeaseNamespace holds the nodes' heartbeat leases, one per node, named and held by the node.
const nodeLeaseNamespace = "kube-node-lease"

func leaseCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	holder, link := leaseHolder(obj)
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: holder, Link: link},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// leaseHolder returns spec.holderIdentity and, for a node's heartbeat lease, a link to the node
// that holds it. Other holders are free-form identities ("<host>_<uuid>") that name no object.
func leaseHolder(obj *unstructured.Unstructured) (string, *model.ObjectRef) {
	holder := model.NestedString(obj, "spec", "holderIdentity")
	if obj.GetNamespace() == nodeLeaseNamespace {
		return holder, linkTo(keyNodes, "", holder)
	}
	return holder, nil
}

func leaseDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	holder, link := leaseHolder(obj)
	d.put(linkedItem("Holder", holder, link))
	if n, ok := model.NestedInt64(obj, "spec", "leaseDurationSeconds"); ok {
		d.add("Lease Duration", itoa(n)+"s")
	}
	d.add("Acquired", timestamp(model.NestedTime(obj, "spec", "acquireTime")))
	d.add("Renewed", timestamp(model.NestedTime(obj, "spec", "renewTime")))
	if n, ok := model.NestedInt64(obj, "spec", "leaseTransitions"); ok {
		d.add("Transitions", itoa(n))
	}
	d.add("Strategy", model.NestedString(obj, "spec", "strategy"))
	d.add("Preferred Holder", model.NestedString(obj, "spec", "preferredHolder"))
	return withSection(nil, model.TitleProperties, d)
}
