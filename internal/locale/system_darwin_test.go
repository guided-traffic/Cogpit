//go:build darwin && cgo

package locale

import (
	"testing"
	"time"
)

// TestSystemFormatOnMacOS: CoreFoundation answers with a date and a time pattern, whatever this
// Mac's region is (no assertion on the developer's locale), and System uses them. The log shows
// the patterns and the owner's example time in them (D-31's verification on a Mac).
func TestSystemFormatOnMacOS(t *testing.T) {
	date, timeOfDay, ok := systemPatterns()
	if !ok || date == "" || timeOfDay == "" {
		t.Fatalf("CoreFoundation gave no patterns: %q, %q", date, timeOfDay)
	}
	f, ok := systemFormat()
	if !ok {
		t.Fatal("systemFormat failed where systemPatterns did not")
	}
	sys := System()
	if sys != f {
		t.Errorf("System = %+v, want the macOS format %+v", sys, f)
	}
	at := time.Date(2026, 9, 10, 17, 41, 1, 0, time.UTC)
	t.Logf("%s: %q, %q → Go %q; 2026-09-10 17:41:01 UTC reads %q", f.Source, date, timeOfDay, f.Layout,
		f.Layout.In(time.UTC).Format(at))
}
