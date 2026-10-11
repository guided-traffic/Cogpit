// Package kinds holds Cogpit's built-in kind profiles: one file per kind, each matching the
// columns, status rules and actions of its row in docs/parity/resource-parity.md.
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

// builtin lists the profiles. Phase 1 replaces these minimal placeholders with the full
// profiles, one file per kind.
var builtin = []*model.KindProfile{
	placeholder("pods", "", "Pod", "Pods", true),
	placeholder("deployments", "apps", "Deployment", "Deployments", true),
	placeholder("services", "", "Service", "Services", true),
	placeholder("namespaces", "", "Namespace", "Namespaces", false),
	placeholder("nodes", "", "Node", "Nodes", false),
	placeholder("events", "", "Event", "Events", true),
}

func placeholder(key, group, kind, title string, namespaced bool) *model.KindProfile {
	cols := []model.Column{{ID: "name", Title: "Name", Type: model.ColumnText}}
	if namespaced {
		cols = append(cols, model.Column{ID: "namespace", Title: "Namespace", Type: model.ColumnText})
	}
	cols = append(cols, model.Column{ID: "age", Title: "Age", Type: model.ColumnAge})
	return &model.KindProfile{
		Key: key, Group: group, Version: "v1", Resource: key, Kind: kind, Title: title,
		Namespaced: namespaced, Columns: cols,
	}
}
