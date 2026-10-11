// Package kinds holds Cogpit's built-in kind profiles: one file per kind, each matching the
// columns and status rules of its row in docs/parity/resource-parity.md.
//
// Layer L0, like package model: no I/O.
package kinds

import "github.com/guided-traffic/cogpit/internal/model"

// Builtin returns the built-in kind profiles in sidebar order. Each call returns the same
// profiles; callers must not modify them.
func Builtin() []*model.KindProfile {
	return builtin
}

// ByKey returns the built-in profile with the given kind key ("pods").
func ByKey(key string) (*model.KindProfile, bool) {
	for _, p := range builtin {
		if p.Key == key {
			return p, true
		}
	}
	return nil, false
}

// Local returns the profiles of the local kinds, the objects Cogpit makes itself (port forwards):
// every real cluster's backend offers them, whatever its discovery says.
func Local() []*model.KindProfile {
	return local
}

// local lists the local kinds' profiles. Each profile lives in its own file.
var local = []*model.KindProfile{}

// builtin lists the profiles in sidebar order. Each profile lives in its own file.
var builtin = []*model.KindProfile{
	nodes,
	pods,
	deployments,
	daemonSets,
	statefulSets,
	replicaSets,
	jobs,
	cronJobs,
	services,
	ingresses,
	ingressClasses,
	configMaps,
	resourceQuotas,
	limitRanges,
	horizontalPodAutoscalers,
	podDisruptionBudgets,
	priorityClasses,
	runtimeClasses,
	leases,
	persistentVolumeClaims,
	namespaces,
	events,
	customResourceDefinitions,
}
