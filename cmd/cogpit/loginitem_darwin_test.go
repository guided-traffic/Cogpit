//go:build darwin

package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// SMAppService's statuses as the Preferences name them; a value macOS adds later reads as not
// found.
func TestLoginItemStatus(t *testing.T) {
	for raw, want := range map[int]string{0: engine.LoginItemOff, 1: engine.LoginItemOn, 2: engine.LoginItemNeedsApproval,
		3: engine.LoginItemNotFound, 4: engine.LoginItemNotFound, -1: engine.LoginItemNotFound} {
		if got := loginItemStatus(raw); got != want {
			t.Errorf("status %d: %q, want %q", raw, got, want)
		}
	}
}

// A Cogpit that does not run from its app bundle (this test binary, go run, bin/cogpit) is never
// described to SMAppService: its status is not found and registering it is refused with the
// reason, without a call to ServiceManagement; removing is nothing to do. This runs the real cgo
// check of the bundle, which reads only the process's own bundle; no login item is registered.
func TestLoginItemOutsideTheAppBundle(t *testing.T) {
	li := newLoginItem()
	if li.(macLoginItem).inBundle {
		t.Fatal("the test binary runs from an app bundle?")
	}
	if st := li.Status(); st != engine.LoginItemNotFound {
		t.Errorf("status %q", st)
	}
	if err := li.Register(); !errors.Is(err, errNotInAppBundle) {
		t.Errorf("register: %v", err)
	}
	if err := li.Unregister(); err != nil {
		t.Errorf("unregister: %v", err)
	}
}

// ServiceManagement's error text reaches the Preferences row as one bounded line.
func TestOneLine(t *testing.T) {
	if got := oneLine("The operation couldn’t be completed.\n(SMAppServiceErrorDomain 1)\u202e", 300); got !=
		"The operation couldn’t be completed. (SMAppServiceErrorDomain 1)" {
		t.Errorf("%q", got)
	}
	long := strings.Repeat("ü", 400)
	if got := oneLine(long, 301); len(got) > 301+len("…") || !strings.HasSuffix(got, "…") || strings.ContainsRune(got, '\uFFFD') {
		t.Errorf("cut %q", got)
	}
}
