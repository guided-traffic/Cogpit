package cache

// Running reports whether an informer for scope is running: held by a handle or still warm. A
// caller that must not start a watch (a search across every custom resource kind) reads only the
// scopes that are running.
func (c *Cache) Running(scope Scope) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.informers[scope.normalize()] != nil
}
