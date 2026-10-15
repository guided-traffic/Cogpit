// Command parity checks the hand-written progress pages and counts what they report
// (docs/parity/README.md#make-parity, D-15). It refuses a checklist row that appears twice, a row
// marked done or partial without evidence, a command id the registry does not know, a link to a
// file that does not exist and a test that no _test.go file declares; then it counts views,
// actions and interface rows by the rules on the parity README and compares the counts with the
// numbers ROADMAP.md and the progress pages report.
//
// It also generates docs/parity/parity-checklist.md from the checklists (checklist.go): a plain run
// writes the page when it differs; -check (make parity, CI) refuses a page that differs instead.
//
//	go run ./hack/parity          # every check; writes the generated page
//	go run ./hack/parity -rows    # the rows only, not the reported numbers (while a recount is due)
//	go run ./hack/parity -check   # as CI: the generated page is checked, not written
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

func main() {
	root := flag.String("root", ".", "the repository root")
	rowsOnly := flag.Bool("rows", false, "check the rows only, not the numbers the pages report")
	check := flag.Bool("check", false, "refuse a generated page that differs instead of writing it (CI)")
	flag.Parse()

	r := engine.NewRegistry()
	if err := commands.RegisterAll(r); err != nil {
		fmt.Fprintln(os.Stderr, "parity:", err)
		os.Exit(2)
	}
	ids := map[string]bool{}
	for _, c := range r.Commands() {
		ids[c.ID] = true
	}
	tests, err := testFunctions(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "parity:", err)
		os.Exit(2)
	}
	c := &checker{root: *root, commands: ids, tests: tests}
	n, breadth := c.run(!*rowsOnly)
	report(os.Stdout, n, breadth)
	if c.syncChecklist(n, !*check) {
		fmt.Println("\nparity: wrote " + pageChecklist)
	}
	if len(c.problems) > 0 {
		fmt.Printf("\nparity: %d problems\n", len(c.problems))
		for _, p := range c.problems {
			fmt.Println("  " + p.String())
		}
		os.Exit(1)
	}
	fmt.Println("\nparity: no problems")
}

// run loads the pages and runs every check; with claims, it also compares the reported numbers.
func (c *checker) run(claims bool) (counts, float64) {
	c.load()
	c.checkDuplicates()
	c.checkRows()
	c.checkReferences()
	c.checkCommands()
	n := c.count()
	if !claims {
		return n, 0
	}
	breadth, _ := c.checkBreadth(n)
	c.checkClaims(n, breadth)
	return n, breadth
}

// report prints the counts.
func report(w *os.File, n counts, breadth float64) {
	line := func(what string, t tally, extra string) {
		fmt.Fprintf(w, "  %-34s %3d of %3d exist (%d done, %d partial)%s\n", what, t.exist, t.total, t.done, t.partial, extra)
	}
	fmt.Fprintln(w, "parity: counted from the checklists")
	line("resource views", n.views, "")
	fmt.Fprintf(w, "  %-34s %3d of %3d exist (registered commands)\n", "resource actions", n.actions.exist, n.actions.total)
	line("interface rows, free plan", n.uiFree, fmt.Sprintf("; %d more not a target", n.uiOutside))
	line("interface rows, paid plans", n.uiPaid, "")
	line("platforms and packages", n.platforms, "")
	line("authentication and connectivity", n.connect, "")
	line("platform rows, all", n.platformAll, fmt.Sprintf("; %d more paid or nothing to build", n.platformOutside))
	line("beyond Lens (not in breadth)", n.beyond, "")
	if breadth > 0 {
		fmt.Fprintf(w, "  %-34s %.1f%% weighted\n", "feature breadth", breadth)
	}
	if len(n.plannedActions) > 0 {
		fmt.Fprintf(w, "  planned actions, not registered: %s\n", strings.Join(n.plannedActions, ", "))
	}
	if len(n.notTargetActions) > 0 {
		fmt.Fprintf(w, "  actions that are not a target: %s\n", strings.Join(n.notTargetActions, ", "))
	}
}

var testFunc = regexp.MustCompile(`(?m)^func ((?:Test|Fuzz|Benchmark|Example)[A-Za-z0-9_]*)\(`)

// testFunctions are the test, fuzz, benchmark and example functions every _test.go file under
// root declares (build output, caches and dependencies left out).
func testFunctions(root string) (map[string]bool, error) {
	skip := []string{".git", "bin", ".cache", "node_modules", "log", "plan"}
	out := map[string]bool{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if p != root && slices.Contains(skip, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(p, "_test.go") {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		for _, m := range testFunc.FindAllSubmatch(b, -1) {
			out[string(m[1])] = true
		}
		return nil
	})
	return out, err
}
