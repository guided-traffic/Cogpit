package cache

import (
	"cmp"
	"slices"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Peeking (H-82): a consumer that must use only what the cache holds already, and must not change
// how long it holds it, reads the running scopes without a handle. Nothing starts, nothing is held,
// and no warm period is extended: a scope that expires while it is peeked at is simply gone at the
// next peek. The engine builds the matcher that replaces the cached Secrets' values in what agents
// read this way, and rebuilds it only when PeekVersion changes.

// Peeked is one running scope as Peek read it.
type Peeked struct {
	Scope Scope
	// Objects are the scope's objects, shared with the store: they must not be modified.
	Objects []*unstructured.Unstructured
}

// PeekVersion describes the running scopes that match selects, without reading their objects: it
// changes whenever one of them starts, stops or delivers an event (an object added, changed or
// deleted), and is the same while none does. Read it before Peek: the objects Peek then returns
// are at least as new as the version says.
func (c *Cache) PeekVersion(match func(Scope) bool) string {
	type row struct {
		key       string
		seq       uint64
		delivered int64
	}
	c.mu.Lock()
	rows := make([]row, 0, 4)
	for scope, e := range c.informers {
		if match(scope) {
			rows = append(rows, row{key: scope.String(), seq: e.seq, delivered: e.events.Load()})
		}
	}
	c.mu.Unlock()
	slices.SortFunc(rows, func(a, b row) int { return cmp.Or(strings.Compare(a.key, b.key), cmp.Compare(a.seq, b.seq)) })
	var b strings.Builder
	for _, r := range rows {
		b.WriteString(r.key)
		b.WriteByte(0)
		b.WriteString(strconv.FormatUint(r.seq, 10))
		b.WriteByte(':')
		b.WriteString(strconv.FormatInt(r.delivered, 10))
		b.WriteByte(0)
	}
	return b.String()
}

// Peek returns the objects of every running scope that match selects (a handle holds it or it is
// warm), without acquiring any: see the package's note on peeking. A label-selected scope served by
// its cover returns the objects its selector selects.
func (c *Cache) Peek(match func(Scope) bool) []Peeked {
	c.mu.Lock()
	entries := make([]*entry, 0, 4)
	for scope, e := range c.informers {
		if match(scope) {
			entries = append(entries, e)
		}
	}
	c.mu.Unlock()
	out := make([]Peeked, 0, len(entries))
	for _, e := range entries {
		var objs []*unstructured.Unstructured
		if e.view != nil {
			objs = e.view.list()
		} else {
			objs = toUnstructured(e.run.inf.GetStore().List())
		}
		out = append(out, Peeked{Scope: e.scope, Objects: objs})
	}
	slices.SortFunc(out, func(a, b Peeked) int { return strings.Compare(a.Scope.String(), b.Scope.String()) })
	return out
}
