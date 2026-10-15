// Command scorecard writes docs/parity/scorecard.md (D-15, docs/parity/README.md#the-pages): it
// runs the budget tests and benchmarks of docs/developer/testing.md's performance budgets without
// the race detector, collects what they measured (the tests' internal/testkit/scorecard lines, the
// benchmarks' ns/op), and writes per budget its proposed target from testing.md, the measured
// values, pass or fail, the date and the machine, and the budgets nothing measures.
//
//	go run ./hack/scorecard [-o docs/parity/scorecard.md] [-testing docs/developer/testing.md]
//	                        [-from <dir>] [-keep <dir>]
//
// make scorecard runs it. It is not run in CI: D-15 waits with that for budgets measured regularly
// on a fixed runner, and CI's shared runners would make the numbers the load's. -from renders the
// page from a directory of measurements (its *.jsonl and bench.txt) without running anything;
// -keep keeps a run's measurements in a directory.
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit/scorecard"
)

// budgetTests are the tests that record measurements (scorecard.Record), by package. Every one of
// them must record at least one line, or the run fails: a renamed or skipped test would otherwise
// drop its budget from the page silently.
var budgetTests = []struct {
	pkg   string
	tests []string
}{
	{"./internal/testkit/integration", []string{
		"TestBudgetFirstRows", "TestBudgetIdleViewAndWarmReopen", "TestShowcaseServiceByClusterIP",
		"TestBudgetNamespaceLimitedUser", "TestNodesListDiskBudget", "TestBudgetAccessibleNamespaces",
		"TestBudgetAgentReadTakesTheSecrets",
	}},
	{"./internal/metrics", []string{"TestClusterBudget"}},
	{"./internal/ui/fyneui/termview", []string{"TestLogsP5"}},
}

// benchmark is a Go benchmark whose time per operation measures a budget.
type benchmark struct {
	pkg, name, budget string
	// metric says what one operation is; %s takes a sub-benchmark's name.
	metric string
	limit  float64 // in ms
	bound  scorecard.Bound
}

// The packages of the benchmarks.
const (
	enginePkg   = "./internal/engine"
	yamlviewPkg = "./internal/ui/fyneui/yamlview"
)

var benchmarks = []benchmark{
	{pkg: enginePkg, name: "BenchmarkSortChange10k", budget: "P2", limit: 50,
		metric: "re-sort a 10,000-pod view by another column from warm rows"},
	{pkg: enginePkg, name: "BenchmarkSortCommand10k", budget: "P2", limit: 50,
		metric: "a sort command to the new 10,000-row snapshot: the UI state change, the rebuild, the table event"},
	{pkg: enginePkg, name: "BenchmarkSnapshot10k", budget: "P4", limit: 16,
		metric: "a 10,000-row snapshot from warm rows (%s), the engine's share of a frame"},
	{pkg: yamlviewPkg, name: "BenchmarkKeystrokeLayout5MB", budget: "P8", limit: 16,
		metric: "a keystroke in the 5 MB document, highlighting on: the widget's work without the paint"},
	{pkg: yamlviewPkg, name: "BenchmarkKeystroke5MB", budget: "P8", bound: scorecard.Context,
		metric: "the same with a full software paint of a 900×700 window, which overstates a GPU frame"},
}

// notMeasured says why make scorecard measures a budget not at all.
var notMeasured = map[string]string{
	"P3": "a real window's frames while scrolling; offscreen tests have no frames (measured by hand in the spike, S1)",
	"P6": "resident memory, to be tracked against Lens Desktop on the same machine; no test measures it",
	"P7": "a cold start to a real window; no test starts a window (measured by hand in the spike)",
}

const benchFile = "bench.txt"

func main() {
	out := flag.String("o", "docs/parity/scorecard.md", "the page to write")
	testingDoc := flag.String("testing", "docs/developer/testing.md", "the page with the budgets' table")
	from := flag.String("from", "", "render from this directory's measurements instead of running the tests")
	keep := flag.String("keep", "", "keep the run's measurements in this directory")
	flag.Parse()
	if err := run(*out, *testingDoc, *from, *keep); err != nil {
		fmt.Fprintln(os.Stderr, "scorecard:", err)
		os.Exit(1)
	}
}

func run(out, testingDoc, from, keep string) error {
	budgets, err := readBudgets(testingDoc)
	if err != nil {
		return err
	}
	dir := from
	if dir == "" {
		if dir = keep; dir == "" {
			if dir, err = os.MkdirTemp("", "cogpit-scorecard-"); err != nil {
				return err
			}
			defer func() { _ = os.RemoveAll(dir) }()
		} else if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		if err := runTests(dir); err != nil {
			return err
		}
		if err := runBenchmarks(dir); err != nil {
			return err
		}
	}
	lines, err := collect(dir)
	if err != nil {
		return err
	}
	page, err := render(budgets, lines, describeMachine(), lastDate(lines))
	if err != nil {
		return err
	}
	if err := os.WriteFile(out, []byte(page), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "scorecard: %d measurements of %d budgets in %s\n", len(lines), len(budgets), out)
	return nil
}

// testEnv is the environment of the runs: no cluster reachable (make's TEST_ENV), the scorecard's
// directory named.
func testEnv(dir string) []string {
	var env []string
	for _, kv := range os.Environ() {
		switch strings.SplitN(kv, "=", 2)[0] {
		case "KUBERNETES_SERVICE_HOST", "KUBERNETES_SERVICE_PORT", "KUBECONFIG", scorecard.DirEnv:
			continue
		}
		env = append(env, kv)
	}
	return append(env, "KUBECONFIG=/dev/null", scorecard.DirEnv+"="+dir)
}

func runTests(dir string) error {
	var names []string
	pkgs := make([]string, 0, len(budgetTests))
	for _, b := range budgetTests {
		names = append(names, b.tests...)
		pkgs = append(pkgs, b.pkg)
	}
	args := append([]string{"test", "-count=1", "-timeout", "15m", "-run", "^(" + strings.Join(names, "|") + ")$"}, pkgs...)
	fmt.Fprintln(os.Stderr, "go", strings.Join(args, " "))
	cmd := exec.Command("go", args...)
	cmd.Env, cmd.Stdout, cmd.Stderr = testEnv(dir), os.Stderr, os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("the budget tests: %w", err)
	}
	return nil
}

func runBenchmarks(dir string) error {
	byPkg := map[string][]string{}
	var pkgs []string
	for _, b := range benchmarks {
		if byPkg[b.pkg] == nil {
			pkgs = append(pkgs, b.pkg)
		}
		byPkg[b.pkg] = append(byPkg[b.pkg], b.name)
	}
	f, err := os.Create(filepath.Join(dir, benchFile))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	for _, pkg := range pkgs {
		args := []string{"test", "-count=1", "-run", "^$", "-bench", "^(" + strings.Join(byPkg[pkg], "|") + ")$", pkg}
		fmt.Fprintln(os.Stderr, "go", strings.Join(args, " "))
		cmd := exec.Command("go", args...)
		cmd.Env, cmd.Stdout, cmd.Stderr = testEnv(dir), io.MultiWriter(f, os.Stderr), os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("the benchmarks of %s: %w", pkg, err)
		}
	}
	return f.Close()
}

// collect reads the tests' lines and the benchmarks' results and checks that every budget test
// and benchmark measured something and that nothing ran under the race detector.
func collect(dir string) ([]scorecard.Line, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var lines []scorecard.Line
	for _, name := range files {
		got, err := readLines(name)
		if err != nil {
			return nil, err
		}
		lines = append(lines, got...)
	}
	var problems []string
	for _, b := range budgetTests {
		for _, test := range b.tests {
			if !slicesContainsFunc(lines, func(l scorecard.Line) bool { return topLevel(l.Test) == test }) {
				problems = append(problems, test+" recorded nothing: renamed, skipped, or its scorecard.Record calls gone?")
			}
		}
	}
	benched, err := readBench(filepath.Join(dir, benchFile))
	if err != nil {
		return nil, err
	}
	for _, b := range benchmarks {
		if !slicesContainsFunc(benched, func(l scorecard.Line) bool { return topLevel(l.Test) == b.name }) {
			problems = append(problems, b.name+" reported no result")
		}
	}
	lines = append(lines, benched...)
	for _, l := range lines {
		if l.Race {
			problems = append(problems, l.Test+" ran under the race detector: budgets are judged without it")
			break
		}
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "\n"))
	}
	return lines, nil
}

func readLines(name string) ([]scorecard.Line, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var lines []scorecard.Line
	sc := bufio.NewScanner(f)
	for n := 1; sc.Scan(); n++ {
		var l scorecard.Line
		if err := json.Unmarshal(sc.Bytes(), &l); err != nil {
			return nil, fmt.Errorf("%s:%d: %w", name, n, err)
		}
		lines = append(lines, l)
	}
	return lines, sc.Err()
}

// benchLine is a result line of go test -bench: name, GOMAXPROCS, iterations, ns/op.
var benchLine = regexp.MustCompile(`^(Benchmark\S+?)(?:-(\d+))?\s+(\d+)\s+([0-9.]+) ns/op`)

// readBench turns the benchmarks' results into lines: time per operation in ms against the
// benchmark's budget.
func readBench(name string) ([]scorecard.Line, error) {
	f, err := os.Open(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	var lines []scorecard.Line
	pkg := ""
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		text := sc.Text()
		if p, ok := strings.CutPrefix(text, "pkg: "); ok {
			pkg = p
			continue
		}
		m := benchLine.FindStringSubmatch(text)
		if m == nil {
			continue
		}
		top, sub, _ := strings.Cut(m[1], "/")
		b, ok := benchmarkNamed(top)
		if !ok {
			continue
		}
		ns, err := strconv.ParseFloat(m[4], 64)
		if err != nil {
			return nil, err
		}
		procs, _ := strconv.Atoi(m[2])
		metric := b.metric
		if strings.Contains(metric, "%s") {
			metric = fmt.Sprintf(metric, sub)
		}
		bound := b.bound
		if bound == "" {
			bound = scorecard.AtMost
		}
		l := scorecard.Line{
			Measurement: scorecard.Measurement{Budget: b.budget, Metric: metric, Value: ns / 1e6, Unit: "ms", Limit: b.limit, Bound: bound,
				How: "time per operation of the benchmark (" + m[3] + " operations)"},
			Test: m[1], Package: pkg, GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, CPUs: runtime.NumCPU(), GOMAXPROCS: procs,
			Time: st.ModTime().UTC(),
		}
		if pass, bounded := l.Within(); bounded {
			l.Pass = &pass
		}
		lines = append(lines, l)
	}
	return lines, sc.Err()
}

func benchmarkNamed(name string) (benchmark, bool) {
	for _, b := range benchmarks {
		if b.name == name {
			return b, true
		}
	}
	return benchmark{}, false
}

func topLevel(test string) string {
	top, _, _ := strings.Cut(test, "/")
	return top
}

func slicesContainsFunc(lines []scorecard.Line, f func(scorecard.Line) bool) bool {
	for _, l := range lines {
		if f(l) {
			return true
		}
	}
	return false
}

func lastDate(lines []scorecard.Line) time.Time {
	var last time.Time
	for _, l := range lines {
		if l.Time.After(last) {
			last = l.Time
		}
	}
	if last.IsZero() {
		return time.Now()
	}
	return last.Local()
}
