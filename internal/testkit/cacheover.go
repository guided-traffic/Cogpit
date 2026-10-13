package testkit

import (
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
)

// CacheOver returns a cache over src with a minute's warm period, for tests that build an engine
// backend over an in-memory cluster where they may not import package cache themselves (the
// window's tests: the UI reads only engine state). The caller closes it.
func CacheOver(src cache.Source) *cache.Cache {
	return cache.New(src, cache.Options{WarmPeriod: time.Minute})
}
