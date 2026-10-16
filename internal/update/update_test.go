package update

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

// sparkleMinimumCheckInterval is the least scheduled-check interval Sparkle 2.10.0 uses, in
// seconds: SPUUpdater clamps a shorter one up to it (minimumUpdateCheckInterval,
// Sparkle/SPUUpdaterSettings.m).
const sparkleMinimumCheckInterval = 60 * 60

// The app bundle's Info.plist gets the settings this package relies on from hack/macos/sparkle.sh:
// scheduled checks on (Cogpit's preference turns them off), every hour (owner, 2026-10-16), no
// shorter than Sparkle allows, and the update verified before extraction against a signed feed.
func TestSparkleScriptWritesTheSettings(t *testing.T) {
	b, err := os.ReadFile("../../hack/macos/sparkle.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(b)
	for _, line := range []string{
		`plutil -replace SUEnableAutomaticChecks -bool YES "$plist"`,
		`plutil -replace SUVerifyUpdateBeforeExtraction -bool YES "$plist"`,
		`plutil -replace SURequireSignedFeed -bool YES "$plist"`,
	} {
		if !strings.Contains(script, line) {
			t.Errorf("sparkle.sh does not run %s", line)
		}
	}
	m := regexp.MustCompile(`plutil -replace SUScheduledCheckInterval -integer (\d+) "\$plist"`).FindStringSubmatch(script)
	if m == nil {
		t.Fatal("sparkle.sh sets no SUScheduledCheckInterval: Sparkle checks once a day")
	}
	if n, _ := strconv.Atoi(m[1]); n != 3600 || n < sparkleMinimumCheckInterval {
		t.Errorf("SUScheduledCheckInterval %d, want 3600 (an hour; Sparkle's least is %d)", n, sparkleMinimumCheckInterval)
	}
}

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
