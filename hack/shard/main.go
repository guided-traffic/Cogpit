// Command shard splits one package's tests into stable shards for CI's sharded race jobs
// (docs/developer/releasing.md#ci). It reads the package's top-level tests, examples and fuzz
// targets from the test files the build would compile (go list with the given tags decides which
// files count), assigns each name to a shard by an FNV-1a hash of the name, and prints the -run
// pattern of one shard. A test added later lands in a shard without a change to the workflow, and a
// name keeps its shard as long as the number of shards stays the same.
//
//	go run ./hack/shard -pkg ./internal/ui/fyneui -tags race -shard 2 -of 6   # the -run pattern of shard 2
//	go run ./hack/shard -pkg ./internal/ui/fyneui -tags race -of 6 -summary   # the tests per shard
//	go run ./hack/shard ... -summary -durations race.json                     # and their time, from go test -json
package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"hash/fnv"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

func main() {
	pkg := flag.String("pkg", "", "the package, as go list takes it (./internal/ui/fyneui)")
	tags := flag.String("tags", "", "build tags, comma-separated, as go test gets them (race when the shards run with -race)")
	shard := flag.Int("shard", 0, "the shard to print the -run pattern of, 1 to -of")
	of := flag.Int("of", 0, "the number of shards")
	summary := flag.Bool("summary", false, "print the number of tests per shard instead of a pattern")
	durations := flag.String("durations", "", "with -summary: a go test -json file of the package, to sum each shard's time")
	flag.Parse()

	if *pkg == "" || *of < 1 || (!*summary && (*shard < 1 || *shard > *of)) {
		fmt.Fprintln(os.Stderr, "shard: need -pkg, -of >= 1 and -shard 1..of (or -summary)")
		os.Exit(2)
	}
	names, err := testNames(*pkg, *tags)
	if err != nil {
		fmt.Fprintln(os.Stderr, "shard:", err)
		os.Exit(2)
	}
	if len(names) == 0 {
		fmt.Fprintf(os.Stderr, "shard: %s has no tests\n", *pkg)
		os.Exit(1)
	}
	shards := split(names, *of)
	if *summary {
		times := map[string]float64{}
		if *durations != "" {
			if times, err = readDurations(*durations); err != nil {
				fmt.Fprintln(os.Stderr, "shard:", err)
				os.Exit(2)
			}
		}
		printSummary(os.Stdout, shards, times)
		return
	}
	mine := shards[*shard-1]
	if len(mine) == 0 {
		// A pattern no test name matches: go test runs nothing and passes.
		fmt.Fprintf(os.Stderr, "shard: shard %d of %d is empty\n", *shard, *of)
		fmt.Println("^$")
		return
	}
	fmt.Fprintf(os.Stderr, "shard: %d of %d tests in shard %d of %d\n", len(mine), len(names), *shard, *of)
	fmt.Println(pattern(mine))
}

// split assigns every name to one of n shards by its hash; each shard keeps the names' order.
func split(names []string, n int) [][]string {
	shards := make([][]string, n)
	for _, name := range names {
		h := fnv.New32a()
		_, _ = h.Write([]byte(name))
		i := int(h.Sum32() % uint32(n))
		shards[i] = append(shards[i], name)
	}
	return shards
}

// pattern is the -run pattern that matches exactly the given top-level names (their subtests
// run with them).
func pattern(names []string) string {
	return "^(" + strings.Join(names, "|") + ")$"
}

// testNames lists the package's top-level Test, Example and Fuzz functions, sorted, from the test
// files go list reports for the given tags (TestMain is not a test).
func testNames(pkg, tags string) ([]string, error) {
	args := []string{"list", "-json"}
	if tags != "" {
		args = append(args, "-tags", tags)
	}
	out, err := exec.Command("go", append(args, pkg)...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("go list %s: %v: %s", pkg, err, ee.Stderr)
		}
		return nil, fmt.Errorf("go list %s: %w", pkg, err)
	}
	var p struct {
		Dir          string
		TestGoFiles  []string
		XTestGoFiles []string
	}
	if err := json.Unmarshal(out, &p); err != nil {
		return nil, fmt.Errorf("go list %s: %w", pkg, err)
	}
	fset := token.NewFileSet()
	var names []string
	for _, f := range append(p.TestGoFiles, p.XTestGoFiles...) {
		file, err := parser.ParseFile(fset, filepath.Join(p.Dir, f), nil, parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || fn.Name.Name == "TestMain" {
				continue
			}
			if name := fn.Name.Name; isTest(name, "Test") || isTest(name, "Example") || isTest(name, "Fuzz") {
				names = append(names, name)
			}
		}
	}
	slices.Sort(names)
	return slices.Compact(names), nil
}

// isTest reports whether name is a test function name for prefix, by the go tool's rule: the
// prefix alone, or the prefix followed by a rune that is not a lower-case letter.
func isTest(name, prefix string) bool {
	if !strings.HasPrefix(name, prefix) {
		return false
	}
	if len(name) == len(prefix) {
		return true
	}
	r, _ := utf8.DecodeRuneInString(name[len(prefix):])
	return !unicode.IsLower(r)
}

// readDurations sums the elapsed seconds of each top-level test in a go test -json file.
func readDurations(path string) (map[string]float64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	times := map[string]float64{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var ev struct {
			Action  string
			Test    string
			Elapsed float64
		}
		if json.Unmarshal(sc.Bytes(), &ev) != nil || ev.Test == "" || strings.Contains(ev.Test, "/") {
			continue
		}
		if ev.Action == "pass" || ev.Action == "fail" || ev.Action == "skip" {
			times[ev.Test] = ev.Elapsed
		}
	}
	return times, sc.Err()
}

func printSummary(w io.Writer, shards [][]string, times map[string]float64) {
	for i, names := range shards {
		var sum, longest float64
		slowest := ""
		for _, n := range names {
			sum += times[n]
			if times[n] > longest {
				longest, slowest = times[n], n
			}
		}
		if len(times) == 0 {
			fmt.Fprintf(w, "shard %d: %d tests\n", i+1, len(names))
			continue
		}
		fmt.Fprintf(w, "shard %d: %d tests, %.0f s (longest %s, %.0f s)\n", i+1, len(names), sum, slowest, longest)
	}
}
