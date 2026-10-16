//go:build race

package cache

// raceEnabled: the tests run under the race detector, whose allocator lays the heap out otherwise.
const raceEnabled = true
