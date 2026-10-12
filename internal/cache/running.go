package cache

// Running reports whether an informer serves scope without starting one: the scope is held by a
// handle or still warm, or, for a label-selected scope, its cover is. A caller that must not start
// a watch (a search across every custom resource kind) reads only the scopes that are running.
func (c *Cache) Running(scope Scope) bool {
	scope = scope.normalize()
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.informers[scope] != nil {
		return true
	}
	if _, ok := selectorOf(scope); ok {
		return c.informers[coverOf(scope)] != nil
	}
	return false
}
