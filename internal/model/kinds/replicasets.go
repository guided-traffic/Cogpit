package kinds

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// idControlledBy is the searchable field "Kind/name" of an object's controller.
const idControlledBy = "controlledBy"

var replicaSets = &model.KindProfile{
	Key:        "replicasets",
	Group:      groupApps,
	Version:    "v1",
	Resource:   "replicasets",
	Kind:       "ReplicaSet",
	Title:      "Replica Sets",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "desired", Title: "Desired", Type: model.ColumnNumber},
		{ID: "current", Title: "Current", Type: model.ColumnNumber},
		{ID: "ready", Title: "Ready", Type: model.ColumnNumber},
		ageColumn,
	},
	Cells: replicaSetCells,
	Fields: map[string]fieldFunc{
		idImage: func(obj *unstructured.Unstructured) []string {
			return templateImages(obj)
		},
		idSelector: func(obj *unstructured.Unstructured) []string {
			return labelSelector(obj, "spec", "selector")
		},
		idControlledBy: controlledByField,
	},
	Detail: replicaSetDetail,
}

func replicaSetCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	desired := desiredReplicas(obj)
	current := nonNegative(obj, "status", "replicas")
	ready := nonNegative(obj, "status", "readyReplicas")
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Num: desired},
		{Num: current},
		{Num: ready},
		model.AgeCell(created(obj)),
	}, replicaSetStatus(obj, desired, current, ready)
}

// replicaSetStatus is the row's token: an error when the controller cannot create pods (a
// ReplicaFailure condition: quota, admission), nothing to colour when scaled to zero (the old
// replica sets a deployment keeps), ok when every desired pod is ready, pending while pods are
// created or deleted, a warning when the pods exist but are not all ready. A replica set has no
// progress condition, so pods that are still starting read as a warning until they are ready.
func replicaSetStatus(obj any, desired, current, ready int64) model.Status {
	generation, _ := model.NestedInt64(obj, "metadata", "generation")
	observed, _ := model.NestedInt64(obj, "status", "observedGeneration")
	switch {
	case conditionStatus(obj, deployReplicaFailure, "status", "conditions") == condTrue:
		return model.StatusError
	case desired == 0 && current == 0:
		return model.StatusNone
	case ready == desired && current == desired:
		return model.StatusOK
	case current != desired || ready > desired || observed < generation:
		return model.StatusPending
	}
	return model.StatusWarning
}

func replicaSetDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.add("Replicas", itoa(desiredReplicas(obj))+" desired / "+
		itoa(nonNegative(obj, "status", "replicas"))+" current / "+
		itoa(nonNegative(obj, "status", "readyReplicas"))+" ready / "+
		itoa(nonNegative(obj, "status", "availableReplicas"))+" available")
	d.add("Selector", strings.Join(labelSelector(obj, "spec", "selector"), "\n"))
	d.add("Images", strings.Join(templateImages(obj), "\n"))
	sections := []model.DetailSection{{Title: "Replica Set", Items: d}}
	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), workloadConditionToken); ok {
		sections = append(sections, cs)
	}
	return sections
}

// controlledByField is the controlledBy field of workloads: "Kind/name" of the controlling owner
// (a replica set's Deployment, a job's CronJob), as the Pods view has it.
func controlledByField(obj *unstructured.Unstructured) []string {
	kind, name := controller(obj)
	if kind == "" && name == "" {
		return nil
	}
	return []string{kind + "/" + name}
}
