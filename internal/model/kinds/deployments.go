package kinds

import (
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Deployment condition types.
const (
	deployAvailable      = "Available"
	deployProgressing    = "Progressing"
	deployReplicaFailure = "ReplicaFailure"
)

var deployments = &model.KindProfile{
	Key:        "deployments",
	Group:      "apps",
	Version:    "v1",
	Resource:   "deployments",
	Kind:       "Deployment",
	Title:      "Deployments",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "pods", Title: "Pods", Type: model.ColumnNumber}, // ready/desired, sorted by ready
		{ID: "replicas", Title: "Replicas", Type: model.ColumnNumber},
		ageColumn,
		conditionsColumn,
	},
	Cells: deploymentCells,
	Fields: map[string]fieldFunc{
		"image": func(obj *unstructured.Unstructured) []string {
			return podSpecImages(model.NestedMap(obj, "spec", "template", "spec"))
		},
		idSelector: func(obj *unstructured.Unstructured) []string {
			return labelSelector(obj, "spec", "selector")
		},
	},
	Detail: deploymentDetail,
	// The Pods table in the drawer, through the replica sets (workloadpods.go, D-40).
	DetailNeeds: deploymentPodsNeeds,
	DetailWith:  withPods(deploymentDetail, &ReplicaSetsByOwner),
}

func deploymentCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	desired := desiredReplicas(obj)
	ready := nonNegative(obj, "status", "readyReplicas")
	conds := conditions(obj, "status", "conditions")
	text, token := deploymentConditions(conds)
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		ratio(ready, desired),
		{Num: desired},
		model.AgeCell(created(obj)),
		{Text: text, Status: token},
	}, deploymentStatus(obj, desired, ready, conds)
}

// desiredReplicas reads spec.replicas, which the API defaults to 1 when it is not set.
func desiredReplicas(obj any) int64 {
	n, ok := model.NestedInt64(obj, "spec", "replicas")
	if !ok {
		return 1
	}
	if n < 0 {
		return 0
	}
	return n
}

// deploymentConditions renders the Conditions column, the condition types that are True, sorted,
// and colours it: a replica failure or an unavailable deployment is an error, a rollout that has
// not become available yet is pending.
func deploymentConditions(conds []condition) (string, model.Status) {
	var trueTypes []string
	for _, c := range conds {
		if c.Status == condTrue {
			trueTypes = append(trueTypes, c.Type)
		}
	}
	sort.Strings(trueTypes)
	text := strings.Join(trueTypes, " ")

	available, hasAvailable := findCondition(conds, deployAvailable)
	failure, _ := findCondition(conds, deployReplicaFailure)
	progressing, _ := findCondition(conds, deployProgressing)
	switch {
	case failure.Status == condTrue, available.Status == condFalse:
		return text, model.StatusError
	case available.Status == condTrue:
		return text, model.StatusOK
	case progressing.Status == condTrue && !hasAvailable:
		return text, model.StatusPending
	}
	return text, model.StatusNone
}

// deploymentStatus is the row's token: nothing to colour when scaled to zero, ok when every
// desired pod is ready, pending while a rollout or scaling is under way, warning when the
// deployment is short of ready pods with nothing in progress to fix it.
func deploymentStatus(obj any, desired, ready int64, conds []condition) model.Status {
	switch {
	case desired == 0:
		return model.StatusNone
	case ready == desired:
		return model.StatusOK
	case rollingOut(obj, desired, ready, conds):
		return model.StatusPending
	}
	return model.StatusWarning
}

// rollingOut reports whether the deployment controller is still working towards the spec. A
// paused deployment does not progress, and a Progressing condition that is False (the progress
// deadline passed) means the controller has given up.
func rollingOut(obj any, desired, ready int64, conds []condition) bool {
	if paused, _ := model.NestedBool(obj, "spec", "paused"); paused {
		return false
	}
	progressing, ok := findCondition(conds, deployProgressing)
	if ok && progressing.Status == condFalse {
		return false
	}
	if ok && progressing.Status == condTrue && progressing.Reason != "NewReplicaSetAvailable" {
		return true
	}
	generation, _ := model.NestedInt64(obj, "metadata", "generation")
	observed, _ := model.NestedInt64(obj, "status", "observedGeneration")
	updated := nonNegative(obj, "status", "updatedReplicas")
	return observed < generation || updated < desired || ready > desired
}

func deploymentDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	desired := desiredReplicas(obj)
	d.add("Replicas", itoa(desired)+" desired / "+
		itoa(nonNegative(obj, "status", "updatedReplicas"))+" updated / "+
		itoa(nonNegative(obj, "status", "readyReplicas"))+" ready / "+
		itoa(nonNegative(obj, "status", "availableReplicas"))+" available")
	d.add("Selector", strings.Join(labelSelector(obj, "spec", "selector"), "\n"))
	d.add("Strategy", deploymentStrategy(obj))
	d.add("Revision", model.NestedString(obj, "metadata", "annotations", "deployment.kubernetes.io/revision"))
	if paused, _ := model.NestedBool(obj, "spec", "paused"); paused {
		d.add("Paused", condTrue)
	}
	sections := []model.DetailSection{{Title: "Deployment", Items: d}}
	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), deploymentConditionToken); ok {
		sections = append(sections, cs)
	}
	return sections
}

// deploymentStrategy renders "RollingUpdate (max surge 25%, max unavailable 25%)" or "Recreate".
func deploymentStrategy(obj any) string {
	strategy := model.NestedString(obj, "spec", "strategy", "type")
	if strategy != "RollingUpdate" {
		return strategy
	}
	var opts []string
	if s := model.NestedIntOrString(obj, "spec", "strategy", "rollingUpdate", "maxSurge"); s != "" {
		opts = append(opts, "max surge "+s)
	}
	if s := model.NestedIntOrString(obj, "spec", "strategy", "rollingUpdate", "maxUnavailable"); s != "" {
		opts = append(opts, "max unavailable "+s)
	}
	if len(opts) == 0 {
		return strategy
	}
	return strategy + " (" + strings.Join(opts, ", ") + ")"
}

func deploymentConditionToken(c condition) model.Status {
	switch c.Type {
	case deployReplicaFailure:
		return conditionToken(c.Status, model.StatusError, model.StatusOK)
	case deployProgressing:
		if c.Status == condTrue && c.Reason != "NewReplicaSetAvailable" {
			return model.StatusPending
		}
	}
	return conditionToken(c.Status, model.StatusOK, model.StatusError)
}
