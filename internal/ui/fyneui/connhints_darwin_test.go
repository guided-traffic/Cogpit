//go:build darwin

package fyneui

import "testing"

// TestConnectionHintsReachTheirHandler: the Objective-C side's call reaches the handler the window
// installed, wake and network apart (the observers themselves need AppKit's run loop and a sleeping
// computer: not tested here).
func TestConnectionHintsReachTheirHandler(t *testing.T) {
	var got []connHint
	hintMu.Lock()
	old := hintFn
	hintFn = func(h connHint) { got = append(got, h) }
	hintMu.Unlock()
	t.Cleanup(func() {
		hintMu.Lock()
		hintFn = old
		hintMu.Unlock()
	})
	cogpitConnectionHint(1)
	cogpitConnectionHint(2)
	if len(got) != 2 || got[0] != hintWake || got[1] != hintNetwork {
		t.Fatalf("hints %v", got)
	}
}
