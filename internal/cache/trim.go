package cache

import (
	"sync"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Trimmed scopes (architecture §5, "Trimmed scopes"; D-79). An integration that reads a few fields
// of large objects keeps only those in memory: its scope names a trim (Scope.Trim), and the
// informer's transform hands every object to it after dropping managedFields, before the object
// enters the store. A trimmed scope is a scope of its own, one informer and one watch beside the
// kind's whole objects: the kind's view, the editor and an agent's read of the kind keep reading
// the untrimmed scope, so nothing that shows or writes an object ever sees it trimmed. Kyverno's
// policy reports keep their violations only, not every pass and skip (engine/kyvernoreport.go);
// Trivy's vulnerability reports may keep their summary only.
//
// A trim is registered once, by name, before any scope names it (an init function). It runs on
// the reflector's goroutine for every object of every scope that names it: it must be quick, must
// not block, must only remove from the object what its reader does not need (never add or change
// what the server sent), and must leave metadata alone, which the store keys and the watch's
// resourceVersion come from. A panic in it keeps the object as far as it got (stripManagedFields
// recovers). A scope that names no registered trim keeps its objects whole.

// trims are the registered trims by name: func(*unstructured.Unstructured).
var trims sync.Map

// RegisterTrim registers a trim under name, replacing one registered before; a nil fn removes it.
func RegisterTrim(name string, fn func(*unstructured.Unstructured)) {
	if fn == nil {
		trims.Delete(name)
		return
	}
	trims.Store(name, fn)
}

// trimOf returns the trim registered under name, or nil (no name, none registered).
func trimOf(name string) func(*unstructured.Unstructured) {
	if name == "" {
		return nil
	}
	fn, _ := trims.Load(name)
	f, _ := fn.(func(*unstructured.Unstructured))
	return f
}
