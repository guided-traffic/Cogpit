package scorecard

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestRecordWritesOneLinePerMeasurement records three measurements with COGPIT_SCORECARD set and
// reads them back as make scorecard does: one JSON line each, the verdict per bound, the test's
// name and package, the machine.
func TestRecordWritesOneLinePerMeasurement(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(DirEnv, dir)
	Record(t, Measurement{Budget: "P1", Metric: "first rows", Value: Millis(120 * time.Millisecond), Unit: "ms", Limit: 1000})
	Record(t, Measurement{Budget: "P5", Metric: "lines per second", Value: 9000, Unit: "lines/s", Limit: 10000, Bound: AtLeast})
	Record(t, Measurement{Budget: "P5", Metric: "heap", Value: 1.5, Unit: "MiB", Bound: Context})

	files, err := filepath.Glob(filepath.Join(dir, "scorecard-*.jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("files %v, %v", files, err)
	}
	f, err := os.Open(files[0])
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var lines []Line
	for sc := bufio.NewScanner(f); sc.Scan(); {
		var l Line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			t.Fatalf("%q: %v", sc.Text(), err)
		}
		lines = append(lines, l)
	}
	if len(lines) != 3 {
		t.Fatalf("%d lines", len(lines))
	}
	if l := lines[0]; l.Pass == nil || !*l.Pass || l.Bound != AtMost || l.Value != 120 || l.Unit != "ms" {
		t.Errorf("P1: %+v", l)
	}
	if l := lines[1]; l.Pass == nil || *l.Pass {
		t.Errorf("9000 lines/s at least 10000 passed: %+v", l)
	}
	if l := lines[2]; l.Pass != nil {
		t.Errorf("a context value has a verdict: %+v", l)
	}
	l := lines[0]
	if l.Test != t.Name() || l.Package != "github.com/guided-traffic/cogpit/internal/testkit/scorecard" ||
		l.GOOS == "" || l.CPUs < 1 || l.Race != raceEnabled || l.Time.IsZero() {
		t.Errorf("the line's test and machine: %+v", l)
	}
}

// TestRecordWithoutTheDirectoryOnlyLogs writes nothing when COGPIT_SCORECARD is not set.
func TestRecordWithoutTheDirectoryOnlyLogs(t *testing.T) {
	t.Setenv(DirEnv, "")
	if l := Record(t, Measurement{Budget: "P9", Metric: "idle requests", Unit: "requests"}); l.Pass == nil || !*l.Pass {
		t.Errorf("0 requests at most 0: %+v", l)
	}
}

func TestFormat(t *testing.T) {
	for v, want := range map[float64]string{0: "0", 0.15: "0.15", 12.34: "12.3", 1000: "1000", 9800: "9800", 123456: "123456"} {
		if got := Format(v); got != want {
			t.Errorf("Format(%v) = %q, want %q", v, got, want)
		}
	}
}
