package cache

import "fmt"

// OnStreamError registers fn to be told of every list or watch that ended with an error that is
// not routine (not an expired watch or a stream the server closed): a broken connection, a stream
// that could not be decoded, a request without an answer. The engine checks the connection at once
// when one breaks (D-58). fn runs on the informer's goroutine, once per failed request, and must not
// block; nil unregisters.
func (c *Cache) OnStreamError(fn func(Scope, error)) {
	if fn == nil {
		c.streamErr.Store(nil)
		return
	}
	c.streamErr.Store(&fn)
}

// tellStreamError hands a failed list or watch to the OnStreamError hook, recovering a panic in it.
func (r *runner) tellStreamError(err error) {
	if r.c == nil {
		return
	}
	fn := r.c.streamErr.Load()
	if fn == nil {
		return
	}
	defer func() {
		if p := recover(); p != nil {
			r.log.Error("cache: the stream error hook panicked", "scope", r.scope.String(), "panic", fmt.Sprint(p))
		}
	}()
	(*fn)(r.scope, err)
}
