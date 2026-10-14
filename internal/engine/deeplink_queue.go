package engine

import (
	"log/slog"
	"sync"
)

// DeepLinkQueue holds the cogpit:// links the operating system hands the app (D-47) until the
// window can open them: macOS delivers a link that started Cogpit as soon as the app finishes
// launching, before the window and the engine's connection are up. Deliver queues a link from any
// thread and never blocks; Attach hands the queued links, in their order, and every later one to
// the window's handler, one at a time on a goroutine of the queue's own, so that a link that
// connects a cluster finishes before the next starts. At most MaxQueuedDeepLinks wait; more are
// dropped and logged, a link longer than MaxDeepLink too (a page that opens links in a loop cannot
// grow the queue).
type DeepLinkQueue struct {
	log *slog.Logger

	mu       sync.Mutex
	pending  []string
	handle   func(string)
	draining bool
	dropped  int
}

// MaxQueuedDeepLinks bounds the links that wait for the window. Cogpit's default (D-47).
const MaxQueuedDeepLinks = 8

// NewDeepLinkQueue returns an empty queue that logs what it drops to log (nil: slog.Default()).
func NewDeepLinkQueue(log *slog.Logger) *DeepLinkQueue {
	if log == nil {
		log = slog.Default()
	}
	return &DeepLinkQueue{log: log}
}

// Deliver queues a link. Any goroutine or thread (macOS's Apple event handler calls it on the main
// thread); it never blocks and never calls the handler itself.
func (q *DeepLinkQueue) Deliver(raw string) {
	if q == nil {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(raw) > MaxDeepLink || len(q.pending) >= MaxQueuedDeepLinks {
		q.dropped++
		q.log.Warn("a cogpit:// link was dropped", "bytes", len(raw), "queued", len(q.pending), "dropped", q.dropped)
		return
	}
	q.pending = append(q.pending, raw)
	q.drainLocked()
}

// Attach hands the queued links and every later one to handle, in their order, one at a time on
// the queue's goroutine. It returns the function that detaches it: links that arrive afterwards
// wait again, and a link being handled finishes.
func (q *DeepLinkQueue) Attach(handle func(raw string)) (detach func()) {
	if q == nil || handle == nil {
		return func() {}
	}
	q.mu.Lock()
	q.handle = handle
	q.drainLocked()
	q.mu.Unlock()
	return func() {
		q.mu.Lock()
		q.handle = nil
		q.mu.Unlock()
	}
}

// Pending returns how many links wait.
func (q *DeepLinkQueue) Pending() int {
	if q == nil {
		return 0
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending)
}

// drainLocked starts the goroutine that hands the waiting links on, unless one runs or no handler
// is attached. q.mu is held.
func (q *DeepLinkQueue) drainLocked() {
	if q.handle == nil || q.draining || len(q.pending) == 0 {
		return
	}
	q.draining = true
	go q.drain()
}

func (q *DeepLinkQueue) drain() {
	for {
		q.mu.Lock()
		if q.handle == nil || len(q.pending) == 0 {
			q.draining = false
			q.mu.Unlock()
			return
		}
		raw, handle := q.pending[0], q.handle
		q.pending = q.pending[1:]
		q.mu.Unlock()
		q.call(handle, raw)
	}
}

// call runs the handler under a recover: one link must not end the app (D-11).
func (q *DeepLinkQueue) call(handle func(string), raw string) {
	defer func() {
		if r := recover(); r != nil {
			q.log.Error("handling a cogpit:// link panicked", "panic", r)
		}
	}()
	handle(raw)
}
