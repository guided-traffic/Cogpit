package cache

import "time"

// WarmUntil returns when the last scope that no handle holds ends its warm period: the zero time
// when every scope is held, or none is kept. The engine asks it of a cluster the window no longer
// shows (D-69, several clusters connected at once): once that time has passed, the informers the
// window's views released are gone, and showing the cluster again lists them anew. It sends no
// request and walks only the cache's own bookkeeping.
func (c *Cache) WarmUntil() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	var last time.Time
	for _, e := range c.informers {
		if e.refs == 0 && e.warmUntil.After(last) {
			last = e.warmUntil
		}
	}
	return last
}
