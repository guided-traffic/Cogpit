package cache

import "slices"

// Runners returns the scopes of the informers running, sorted: what costs a watch.
func Runners(c *Cache) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, 0, len(c.runners))
	for r := range c.runners {
		out = append(out, r.scope.String())
	}
	slices.Sort(out)
	return out
}
