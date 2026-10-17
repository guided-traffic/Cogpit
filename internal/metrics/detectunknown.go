package metrics

import (
	"context"
	"fmt"
	"runtime/debug"
	"time"
)

// DetectUnknown detects, once and in the background, every provider still on that nothing has
// looked for yet (its status unknown): what the cluster's Settings › Integrations and the window's
// question about a new cluster's integrations show right after a connect (D-76), before a drawer
// or a failing Metrics Server would look. It is bounded as detection is (the Metrics Server: one
// discovery request; Prometheus: the cached Services and at most MaxProbes probes, D-71), each
// provider within the request timeout, and it repeats nothing: a provider looked for before keeps
// its answer until a poll asks again (Forget and SetSource make it unknown again), and one whose
// detection the timeout cut short is tried again after the retry TTL at the earliest. done, if set,
// runs after the detection ended, unless the hub closed meanwhile. It reports whether a detection
// started; one already running, or nothing unknown, starts none. It never polls.
func (h *Hub) DetectUnknown(done func()) bool {
	h.mu.Lock()
	if h.closed || h.detectingUnknown {
		h.mu.Unlock()
		return false
	}
	var due []*slot
	now := h.opts.Now()
	for _, sl := range h.slots {
		// A detection cut short by the timeout leaves the provider unknown: it is not tried again
		// before the retry TTL, so that a cluster that does not answer is not asked at every change.
		if !sl.off && !sl.asked && now.Sub(h.triedUnknown[sl]) >= h.opts.RetryDetectTTL {
			due = append(due, sl)
		}
	}
	if len(due) == 0 {
		h.mu.Unlock()
		return false
	}
	if h.triedUnknown == nil {
		h.triedUnknown = map[*slot]time.Time{}
	}
	for _, sl := range due {
		h.triedUnknown[sl] = now
	}
	h.detectingUnknown = true
	h.wg.Add(1) // under h.mu: Close sets closed under it before it waits
	h.mu.Unlock()
	go func() {
		defer h.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				h.opts.Logger.Error("metrics: detecting the unknown providers panicked", "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			}
		}()
		fns := make([]func(), len(due))
		for i, sl := range due {
			fns[i] = func() {
				ctx, cancel := context.WithTimeout(h.ctx, h.opts.RequestTimeout)
				defer cancel()
				h.detect(ctx, sl)
			}
		}
		runParallel(fns...)
		h.mu.Lock()
		h.detectingUnknown = false
		for _, sl := range due {
			if sl.asked { // answered: a Forget later may ask again at once
				delete(h.triedUnknown, sl)
			}
		}
		h.mu.Unlock()
		if done != nil && h.ctx.Err() == nil {
			done()
		}
	}()
	return true
}
