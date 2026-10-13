package cache

import (
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// A joined handle reads several scopes as one: Helm's releases, whose records Helm keeps in
// Secrets or, with HELM_DRIVER=configmap, in ConfigMaps, both labelled owner=helm (the engine's
// helmreleases kind, architecture §5). Each part is an ordinary handle on its own scope, so each
// keeps its own informer, its covering scope and its warm period; the joined handle only reads
// them together. Nothing is copied: List returns the parts' shared objects.

// Join returns one handle over the objects of parts, which the joined handle owns: its Release
// releases every part. Its scope is the first part's. One part is returned as it is.
//
//   - List and ListNamespace return every part's objects, in the parts' order. Two parts may hold
//     objects of the same namespace and name (a Secret and a ConfigMap): both are returned, and Get
//     returns the first part's. Parts tells them apart.
//   - HasSynced is true once every part has synced or failed (Err), and at least one has synced:
//     a part the user may not list does not keep the others from showing.
//   - Err is the first part's error, nil when none has one.
//   - Subscribe delivers every part's events, one at a time (a part's informer delivers on its
//     own goroutine; the joined subscription serialises them), in no order across parts.
func Join(parts ...*Handle) *Handle {
	if len(parts) == 1 {
		return parts[0]
	}
	var scope Scope
	if len(parts) > 0 {
		scope = parts[0].scope
	}
	return &Handle{scope: scope, parts: parts}
}

// Parts returns a joined handle's parts in the order Join was given them, or nil for a handle on
// one scope. The parts must not be released on their own.
func (h *Handle) Parts() []*Handle {
	if h.parts == nil {
		return nil
	}
	return append([]*Handle(nil), h.parts...)
}

func (h *Handle) joinedSynced() bool {
	synced := false
	for _, p := range h.parts {
		switch {
		case p.HasSynced():
			synced = true
		case p.Err() != nil:
		default:
			return false
		}
	}
	return synced
}

func (h *Handle) joinedList(namespace string, all bool) []*unstructured.Unstructured {
	var out []*unstructured.Unstructured
	for _, p := range h.parts {
		if all {
			out = append(out, p.List()...)
		} else {
			out = append(out, p.ListNamespace(namespace)...)
		}
	}
	return out
}

func (h *Handle) joinedGet(namespace, name string) (*unstructured.Unstructured, bool) {
	for _, p := range h.parts {
		if o, ok := p.Get(namespace, name); ok {
			return o, true
		}
	}
	return nil, false
}

func (h *Handle) joinedErr() error {
	for _, p := range h.parts {
		if err := p.Err(); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handle) joinedSubscribe(fn func(Event)) func() {
	var mu sync.Mutex
	serial := func(ev Event) {
		mu.Lock()
		defer mu.Unlock()
		fn(ev)
	}
	cancels := make([]func(), 0, len(h.parts))
	for _, p := range h.parts {
		cancels = append(cancels, p.Subscribe(serial))
	}
	return func() {
		for _, c := range cancels {
			c()
		}
	}
}

func (h *Handle) joinedRelease() {
	if !h.released.CompareAndSwap(false, true) {
		return
	}
	for _, p := range h.parts {
		p.Release()
	}
}
