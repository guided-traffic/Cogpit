package engine

import (
	"fmt"
	"runtime/debug"
	"time"
)

// goSafe runs fn on a new goroutine that the session waits for on Close. A panic is recovered,
// logged with its stack and reported to the user as an error toast, so that one failing loop does
// not end the process (D-11).
func (s *Session) goSafe(what string, fn func()) {
	s.state.wg.Add(1)
	go func() {
		defer s.state.wg.Done()
		defer s.recoverTo(what)
		fn()
	}()
}

// recoverTo is deferred by goroutines and loops: it turns a panic into a log line and a toast.
// It must be the deferred function itself, because recover only works there.
func (s *Session) recoverTo(what string) {
	if r := recover(); r != nil {
		s.log.Error("engine: internal error", "what", what, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
		s.emit(Event{Type: EventToast, Level: LevelError, Message: what + " failed: internal error"})
	}
}

// signal wakes a coalescing loop without blocking: a pending wake already covers the change.
func signal(wake chan<- struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}

// runCoalesced calls fn after every wake, but at most once per interval, so a burst of a thousand
// watch events costs one run now and one when the interval has passed (architecture §5). poll, if
// not nil, is asked every interval whether fn should run anyway: state that changes without an
// event, such as an informer finishing its initial list. The loop ends when stop is closed.
func runCoalesced(stop <-chan struct{}, wake chan struct{}, interval time.Duration, poll func() bool, fn func()) {
	var tick <-chan time.Time
	if poll != nil {
		t := time.NewTicker(interval)
		defer t.Stop()
		tick = t.C
	}
	var last time.Time
	for {
		select {
		case <-stop:
			return
		case <-wake:
		case <-tick:
			if !poll() {
				continue
			}
		}
		if !last.IsZero() {
			if wait := interval - time.Since(last); wait > 0 {
				t := time.NewTimer(wait)
				select {
				case <-stop:
					t.Stop()
					return
				case <-t.C:
				}
			}
		}
		// A wake that arrived while waiting is covered by this run: fn reads the latest state.
		select {
		case <-wake:
		default:
		}
		select {
		case <-stop:
			return
		default:
		}
		last = time.Now()
		fn()
	}
}
