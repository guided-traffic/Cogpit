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

// builtin lists the profiles in sidebar order. Each profile lives in its own file.
var builtin = []*model.KindProfile{
	nodes,
	pods,
	deployments,
	services,
	namespaces,
	events,
}
