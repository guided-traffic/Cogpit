package update

import (
	"testing"
	"time"
)

// What Sparkle's delegate reports is kept and tells the listeners on goroutines of their own: an
// update held with its version, a repeat that changes nothing tells nobody, the session's end
// drops the version. An updater that cannot update this Cogpit never reports one and registers
// nobody.
func TestUpdateNoticeTellsTheListeners(t *testing.T) {
	notice.mu.Lock()
	available, version, listeners := notice.available, notice.version, notice.listeners
	notice.mu.Unlock()
	t.Cleanup(func() {
		notice.mu.Lock()
		notice.available, notice.version, notice.listeners = available, version, listeners
		notice.mu.Unlock()
	})
	calls := make(chan struct{}, 8)
	notice.listen(func() { calls <- struct{}{} })
	wait := func(what string) {
		t.Helper()
		select {
		case <-calls:
		case <-time.After(5 * time.Second):
			t.Fatalf("no call after %s", what)
		}
	}
	notice.set(true, "0.12.0")
	wait("the update")
	if v, ok := notice.get(); !ok || v != "0.12.0" {
		t.Fatalf("held %q %v, want 0.12.0", v, ok)
	}
	notice.set(true, "0.12.0")
	notice.set(false, "0.12.0")
	wait("the session's end")
	if v, ok := notice.get(); ok || v != "" {
		t.Fatalf("held %q %v after the session's end", v, ok)
	}
	select {
	case <-calls:
		t.Fatal("a report that changed nothing told the listeners")
	case <-time.After(50 * time.Millisecond):
	}

	none := &Updater{err: ErrNoKey}
	notice.set(true, "0.13.0")
	wait("the next update")
	if v, ok := none.AvailableUpdate(); ok || v != "" {
		t.Errorf("an updater without the key reports %q %v", v, ok)
	}
	n := len(notice.listeners)
	none.OnUpdateChange(func() {})
	var nilUpdater *Updater
	nilUpdater.OnUpdateChange(func() {})
	if len(notice.listeners) != n {
		t.Error("an updater that cannot update registered a listener")
	}
}
