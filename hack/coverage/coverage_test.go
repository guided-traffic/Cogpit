package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A block in several profiles counts once, covered when any profile covered it.
func TestMergeCountsABlockOnce(t *testing.T) {
	dir := t.TempDir()
	unit := filepath.Join(dir, "unit.out")
	race := filepath.Join(dir, "race.out")
	write := func(name, body string) {
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(unit, "mode: atomic\nexample.com/m/a/a.go:1.1,2.2 3 0\nexample.com/m/a/a.go:3.1,4.2 2 5\n")
	write(race, "mode: atomic\nexample.com/m/a/a.go:1.1,2.2 3 1\nexample.com/m/b/b.go:1.1,2.2 5 0\n")
	m := merged{}
	for _, p := range []string{unit, race} {
		if err := m.add(p); err != nil {
			t.Fatal(err)
		}
	}
	if covered, total := m.total(); covered != 5 || total != 10 {
		t.Errorf("total = %d of %d, want 5 of 10", covered, total)
	}
	var profile, summary, badge bytes.Buffer
	if err := m.writeProfile(&profile); err != nil {
		t.Fatal(err)
	}
	if want := "mode: set\nexample.com/m/a/a.go:1.1,2.2 3 1\nexample.com/m/a/a.go:3.1,4.2 2 1\nexample.com/m/b/b.go:1.1,2.2 5 0\n"; profile.String() != want {
		t.Errorf("profile:\n%s\nwant:\n%s", profile.String(), want)
	}
	if err := m.writeSummary(&summary, []string{unit, race}); err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"## Combined coverage: 50.0%", "| `example.com/m/a` | 100.0% | 5/5 |", "| `example.com/m/b` | 0.0% | 0/5 |", "`unit.out`, `race.out`"} {
		if !strings.Contains(summary.String(), line) {
			t.Errorf("summary misses %q:\n%s", line, summary.String())
		}
	}
	if err := writeBadge(&badge, 82.94); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(badge.String(), `"message": "82.9%"`) || !strings.Contains(badge.String(), `"color": "green"`) {
		t.Errorf("badge:\n%s", badge.String())
	}
}

func TestNotAProfile(t *testing.T) {
	name := filepath.Join(t.TempDir(), "x.out")
	if err := os.WriteFile(name, []byte("hello\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (merged{}).add(name); err == nil {
		t.Error("a file without a mode line was read as a profile")
	}
}
