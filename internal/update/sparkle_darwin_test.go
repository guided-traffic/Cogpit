//go:build darwin && cgo

package update

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// sparkleFramework is the Sparkle.framework make sparkle keeps in .cache (or
// $COGPIT_SPARKLE_FRAMEWORK); the test is skipped without one.
func sparkleFramework(t *testing.T) string {
	t.Helper()
	if p := os.Getenv("COGPIT_SPARKLE_FRAMEWORK"); p != "" {
		return p
	}
	found, _ := filepath.Glob(filepath.Join("..", "..", ".cache", "sparkle", "*", "Sparkle.framework"))
	if len(found) == 0 {
		t.Skip("no Sparkle.framework in .cache/sparkle: make sparkle")
	}
	abs, err := filepath.Abs(found[len(found)-1])
	if err != nil {
		t.Fatal(err)
	}
	return abs
}

// buildHarness builds testdata/sparkle_harness.m with sparkle_darwin.m.
func buildHarness(t *testing.T) string {
	t.Helper()
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("no clang")
	}
	bin := filepath.Join(t.TempDir(), "harness")
	build := exec.Command(clang, "-fno-objc-arc", "-framework", "Cocoa", "-I.", "-o", bin,
		filepath.Join("testdata", "sparkle_harness.m"), "sparkle_darwin.m")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("building the harness: %v\n%s", err, out)
	}
	return bin
}

// makeApp lays out <dir>/Cogpit.app around the harness: its Info.plist with the given SU keys,
// and Sparkle.framework in Contents/Frameworks when framework is set. It returns the executable.
func makeApp(t *testing.T, dir, harness, framework string, keys map[string]string) string {
	t.Helper()
	app := filepath.Join(dir, "Cogpit.app")
	exe := filepath.Join(app, "Contents", "MacOS", "harness")
	if err := os.MkdirAll(filepath.Dir(exe), 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(harness)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exe, data, 0o755); err != nil { //nolint:gosec // a test bundle's executable
		t.Fatal(err)
	}
	var extra strings.Builder
	for k, v := range keys {
		fmt.Fprintf(&extra, "\t<key>%s</key>\n\t<string>%s</string>\n", k, v)
	}
	plist := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>harness</string>
	<key>CFBundleIdentifier</key>
	<string>io.guided-traffic.cogpit.test-sparkle-harness</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleVersion</key>
	<string>0.0.1</string>
	<key>SUEnableAutomaticChecks</key>
	<false/>
` + extra.String() + `</dict>
</plist>
`
	if err := os.WriteFile(filepath.Join(app, "Contents", "Info.plist"), []byte(plist), 0o600); err != nil {
		t.Fatal(err)
	}
	if framework != "" {
		dst := filepath.Join(app, "Contents", "Frameworks", "Sparkle.framework")
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("ditto", framework, dst).CombinedOutput(); err != nil {
			t.Fatalf("ditto: %v\n%s", err, out)
		}
	}
	return exe
}

// The app finds and loads Sparkle only from an installed app bundle that carries the framework,
// the appcast's URL and the update key, and says which of them is missing otherwise; loaded, the
// controller is made from another thread while the main thread pumps AppKit's events, as in the
// app (sparkle_darwin.m, in a harness process of its own, without starting the updater: no
// network, no window). Skipped without clang or make sparkle's framework.
func TestSparkleLoadsFromTheBundle(t *testing.T) {
	framework := sparkleFramework(t)
	harness := buildHarness(t)
	const key = "pfIShU4dEXqPd5ObYNfDBiQWcXozk7estwzTnF9BamQ=" // Sparkle's documentation's example key
	both := map[string]string{"SUFeedURL": "https://example.invalid/appcast.xml", "SUPublicEDKey": key}
	for _, tc := range []struct {
		name string
		exe  func(dir string) string
		args []string
		want []string
	}{
		{"bare executable", func(dir string) string {
			exe := filepath.Join(dir, "harness")
			data, _ := os.ReadFile(harness)
			if err := os.WriteFile(exe, data, 0o755); err != nil { //nolint:gosec // a test executable
				t.Fatal(err)
			}
			return exe
		}, nil, []string{"load 1"}},
		{"translocated", func(dir string) string {
			return makeApp(t, filepath.Join(dir, "AppTranslocation", "X", "d"), harness, framework, both)
		}, nil, []string{"load 2"}},
		{"no framework", func(dir string) string { return makeApp(t, dir, harness, "", both) }, nil, []string{"load 3"}},
		{"no feed", func(dir string) string {
			return makeApp(t, dir, harness, framework, map[string]string{"SUPublicEDKey": key})
		}, nil, []string{"load 4"}},
		{"no key", func(dir string) string {
			return makeApp(t, dir, harness, framework, map[string]string{"SUFeedURL": both["SUFeedURL"]})
		}, nil, []string{"load 5"}},
		{"loaded and made", func(dir string) string { return makeApp(t, dir, harness, framework, both) },
			[]string{"create"}, []string{"load 0", "state 4", "automatic 0"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := filepath.EvalSymlinks(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			out, err := exec.Command(tc.exe(dir), tc.args...).Output()
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				t.Fatalf("the harness: %v\n%s\n%s", err, out, exit.Stderr)
			}
			if err != nil {
				t.Fatal(err)
			}
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(lines) != len(tc.want) {
				t.Fatalf("output %q, want lines starting %q", out, tc.want)
			}
			for i, want := range tc.want {
				if got := strings.TrimSpace(lines[i]); got != want && !strings.HasPrefix(got, want+" ") {
					t.Errorf("line %d: %q, want %q", i, got, want)
				}
			}
		})
	}
}

// A test binary is no app bundle: New finds no updater and says so, and Start, CheckForUpdates
// and SetAutomaticChecks do nothing.
func TestNewOutsideABundle(t *testing.T) {
	u := New()
	if err := u.Unavailable(); !errors.Is(err, ErrNoBundle) {
		t.Fatalf("Unavailable() = %v, want ErrNoBundle", err)
	}
	u.Start(true)
	u.SetAutomaticChecks(false)
	if err := u.CheckForUpdates(); !errors.Is(err, ErrNoBundle) {
		t.Errorf("CheckForUpdates() = %v, want ErrNoBundle", err)
	}
}
