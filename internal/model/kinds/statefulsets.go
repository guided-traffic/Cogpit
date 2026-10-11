package kinds

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

var statefulSets = &model.KindProfile{
	Key:        "statefulsets",
	Group:      groupApps,
	Version:    "v1",
	Resource:   "statefulsets",
	Kind:       "StatefulSet",
	Title:      "Stateful Sets",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: idPods, Title: titlePods, Type: model.ColumnNumber}, // ready/desired, sorted by ready
		{ID: "replicas", Title: "Replicas", Type: model.ColumnNumber},
		ageColumn,
	},
	Cells: statefulSetCells,
	Fields: map[string]fieldFunc{
		idImage: func(obj *unstructured.Unstructured) []string {
			return templateImages(obj)
		},
		idSelector: func(obj *unstructured.Unstructured) []string {
			return labelSelector(obj, "spec", "selector")
		},
		"serviceName": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "spec", "serviceName"))
		},
	},
	Detail: statefulSetDetail,
}

func statefulSetCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	desired := desiredReplicas(obj)
	ready := nonNegative(obj, "status", "readyReplicas")
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		ratio(ready, desired),
		{Num: desired},
		model.AgeCell(created(obj)),
	}, statefulSetStatus(obj, desired, ready)
}

// statefulSetStatus is the row's token, after the deployment rule: nothing to colour when scaled
// to zero, ok when every desired pod is ready, pending while scaling or a rolling update is under
// way, a warning when the stateful set is short of ready pods with nothing in progress to fix it.
//
// A stateful set has no progress condition either. With the OrderedReady policy the controller
// creates pod N+1 only once pod N is ready, so a pod that never becomes ready holds the set in
// "scaling" (pending) rather than a warning; the Pods column shows how far it got.
func statefulSetStatus(obj any, desired, ready int64) model.Status {
	switch {
	case desired == 0:
		return model.StatusNone
	case ready == desired:
		return model.StatusOK
	case statefulSetRolling(obj, desired, ready):
		return model.StatusPending
	}
	return model.StatusWarning
}

// statefulSetRolling reports whether the controller is still working towards the spec: it has not
// seen the latest generation, it is creating or deleting pods, or a rolling update has pods left
// to replace. A partition keeps the pods below it on the old revision for good, and OnDelete
// replaces nothing by itself, so neither counts as progress.
func statefulSetRolling(obj any, desired, ready int64) bool {
	generation, _ := model.NestedInt64(obj, "metadata", "generation")
	observed, _ := model.NestedInt64(obj, "status", "observedGeneration")
	if observed < generation || ready > desired || nonNegative(obj, "status", "replicas") != desired {
		return true
	}
	target := desired
	switch model.NestedString(obj, "spec", "updateStrategy", "type") {
	case strategyOnDel:
		target = 0
	default:
		if p, ok := model.NestedInt64(obj, "spec", "updateStrategy", "rollingUpdate", "partition"); ok && p > 0 {
			target = max(desired-p, 0)
		}
	}
	return nonNegative(obj, "status", "updatedReplicas") < target
}

func statefulSetDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.add("Replicas", itoa(desiredReplicas(obj))+" desired / "+
		itoa(nonNegative(obj, "status", "updatedReplicas"))+" updated / "+
		itoa(nonNegative(obj, "status", "readyReplicas"))+" ready / "+
		itoa(nonNegative(obj, "status", "availableReplicas"))+" available")
	d.add("Service", model.NestedString(obj, "spec", "serviceName"))
	d.add("Selector", strings.Join(labelSelector(obj, "spec", "selector"), "\n"))
	d.add("Update Strategy", updateStrategyText(model.NestedMap(obj, "spec", "updateStrategy")))
	d.add("Pod Management Policy", model.NestedString(obj, "spec", "podManagementPolicy"))
	current := model.NestedString(obj, "status", "currentRevision")
	update := model.NestedString(obj, "status", "updateRevision")
	d.add("Current Revision", current)
	if update != current {
		d.add("Update Revision", update)
	}
	d.add("Volume Claim Templates", volumeClaimTemplates(obj))
	d.add("Images", strings.Join(templateImages(obj), "\n"))
	sections := []model.DetailSection{{Title: "Stateful Set", Items: d}}
	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), workloadConditionToken); ok {
		sections = append(sections, cs)
	}
	return sections
}

// volumeClaimTemplates renders "data: 1Gi, ReadWriteOnce, class standard" lines, one per template.
func volumeClaimTemplates(obj any) string {
	var lines []string
	for _, t := range model.NestedSlice(obj, "spec", "volumeClaimTemplates") {
		name := model.NestedString(t, "metadata", "name")
		if name == "" {
			continue
		}
		var parts []string
		if size := model.NestedIntOrString(t, "spec", "resources", "requests", "storage"); size != "" {
			parts = append(parts, size)
		}
		if modes := model.NestedStrings(t, "spec", "accessModes"); len(modes) > 0 {
			parts = append(parts, strings.Join(modes, ", "))
		}
		if class := model.NestedString(t, "spec", "storageClassName"); class != "" {
			parts = append(parts, "class "+class)
		}
		if len(parts) > 0 {
			name += ": " + strings.Join(parts, ", ")
		}
		lines = append(lines, name)
	}
	return strings.Join(lines, "\n")
}
