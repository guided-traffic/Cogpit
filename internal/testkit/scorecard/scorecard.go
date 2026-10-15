// Package scorecard records the measurements of the performance and API-load budgets of
// docs/developer/testing.md (P1, P2, …) for make scorecard, which writes docs/parity/scorecard.md
// from them (hack/scorecard, D-15). A budget test calls Record once per measurement: when
// COGPIT_SCORECARD names a directory the measurement goes there as one JSON line, in a file of the
// test process's own; otherwise Record only logs it. The tests keep their own, more generous
// failure bounds (a loaded CI runner, the race detector); the scorecard judges the budget's bound,
// from a run without the race detector.
//
// It imports nothing of Cogpit's, so the tests of every package may use it, the window's included.
// For tests only (depguard's no-testkit-in-product).
package scorecard

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// DirEnv names the directory make scorecard collects the measurements in.
const DirEnv = "COGPIT_SCORECARD"

// Bound says how a measurement meets its budget's limit.
type Bound string

// The bounds.
const (
	AtMost  Bound = "at most"  // Value ≤ Limit; the zero Bound means this
	AtLeast Bound = "at least" // Value ≥ Limit
	Context Bound = "context"  // no limit: a value that explains the others
)

// Measurement is one measured value of a budget.
type Measurement struct {
	// Budget is the budget's id in testing.md: "P1".
	Budget string `json:"budget"`
	// Metric says in a few words what was measured: "first rows, 1,000 pods".
	Metric string  `json:"metric"`
	Value  float64 `json:"value"`
	// Unit is Value's and Limit's unit: "ms", "requests", "lines/s", "MiB".
	Unit string `json:"unit"`
	// Limit is the budget's bound in Unit, as Bound says (AtMost when empty).
	Limit float64 `json:"limit"`
	Bound Bound   `json:"bound"`
	// How says how it was measured, where the test's name and Metric do not.
	How string `json:"how,omitempty"`
}

// Line is a Measurement as make scorecard reads it: one JSON line with the test and the machine.
type Line struct {
	Measurement
	// Pass says whether Value is within Limit; nil for a Context value.
	Pass       *bool     `json:"pass,omitempty"`
	Test       string    `json:"test"`
	Package    string    `json:"package"`
	Race       bool      `json:"race"`
	GOOS       string    `json:"goos"`
	GOARCH     string    `json:"goarch"`
	CPUs       int       `json:"cpus"`
	GOMAXPROCS int       `json:"gomaxprocs"`
	Time       time.Time `json:"time"`
}

var mu sync.Mutex // one test process's writes to its file

// Record records m for the test t: logged, and written to COGPIT_SCORECARD's directory when it is
// set. A line that cannot be written fails the test.
func Record(t testing.TB, m Measurement) Line {
	t.Helper()
	if m.Bound == "" {
		m.Bound = AtMost
	}
	l := Line{
		Measurement: m, Test: t.Name(), Package: callerPackage(), Race: raceEnabled,
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUs: runtime.NumCPU(), GOMAXPROCS: runtime.GOMAXPROCS(0),
		Time: time.Now().UTC(),
	}
	verdict := "context"
	if pass, ok := l.Within(); ok {
		l.Pass = &pass
		verdict = map[bool]string{true: "pass", false: "over the budget"}[pass]
	}
	t.Logf("scorecard %s: %s = %s %s (budget %s %s %s): %s", m.Budget, m.Metric, Format(m.Value), m.Unit,
		m.Bound, Format(m.Limit), m.Unit, verdict)
	dir := os.Getenv(DirEnv)
	if dir == "" {
		return l
	}
	raw, err := json.Marshal(l)
	if err != nil {
		t.Errorf("scorecard: %v", err)
		return l
	}
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(filepath.Join(dir, fmt.Sprintf("scorecard-%d.jsonl", os.Getpid())), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Errorf("scorecard: %v", err)
		return l
	}
	if _, err := f.Write(append(raw, '\n')); err != nil {
		t.Errorf("scorecard: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Errorf("scorecard: %v", err)
	}
	return l
}

// Within reports whether the value meets its limit; false as its second result for a Context value.
func (m Measurement) Within() (pass, bounded bool) {
	switch m.Bound {
	case AtLeast:
		return m.Value >= m.Limit, true
	case Context:
		return false, false
	default:
		return m.Value <= m.Limit, true
	}
}

// Millis is d in milliseconds, the unit of the budgets' durations.
func Millis(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// Format writes a value with three significant digits at most, without a trailing ".0".
func Format(v float64) string {
	s := fmt.Sprintf("%.3g", v)
	if strings.ContainsAny(s, "e") {
		s = fmt.Sprintf("%.0f", v)
	}
	return s
}

// callerPackage is the import path of the test that called Record: the function two frames up,
// its package's "_test" suffix dropped.
func callerPackage() string {
	pc, _, _, ok := runtime.Caller(2)
	if !ok {
		return ""
	}
	name := runtime.FuncForPC(pc).Name() // github.com/…/integration_test.firstRows.func1
	slash := strings.LastIndexByte(name, '/')
	if dot := strings.IndexByte(name[slash+1:], '.'); dot >= 0 {
		name = name[:slash+1+dot]
	}
	return strings.TrimSuffix(name, "_test")
}
