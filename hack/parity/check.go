package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// The pages the checker reads, relative to the repository root.
const (
	pageUI        = "docs/parity/ui-parity.md"
	pageResources = "docs/parity/resource-parity.md"
	pagePlatforms = "docs/parity/platform-parity.md"
	pageBeyond    = "docs/parity/beyond-lens.md"
	pageTarget    = "docs/parity/target-app-parity.md"
	pageRoadmap   = "ROADMAP.md"
)

// checklists are the pages whose rows carry a status; progress are every page whose tables are
// checked for duplicates and broken links.
var (
	checklists = []string{pageUI, pageResources, pagePlatforms, pageBeyond}
	progress   = []string{
		pageRoadmap, "docs/parity/README.md", pageTarget, "docs/parity/gaps.md", "docs/parity/roadmap.md",
		pageUI, pageResources, pagePlatforms, pageBeyond,
	}
)

// status is the word a checklist row's Status cell starts with.
type status string

const (
	stDone      status = "done"
	stPartial   status = "partial"
	stMissing   status = "missing"
	stNotTarget status = "not a target"
	stNothing   status = "–" // nothing to build, or not Cogpit's to build
)

// counted reports whether a row with this status counts toward a denominator.
func (s status) counted() bool { return s == stDone || s == stPartial || s == stMissing }

// exists reports whether a row with this status counts as existing.
func (s status) exists() bool { return s == stDone || s == stPartial }

// parseStatus reads the status a cell starts with: the word, then the end of the cell or a
// character that is not a letter ("partial: …", "done (2026-10-11, …)").
func parseStatus(cell string) (status, bool) {
	c := strings.TrimSpace(cell)
	for _, s := range []status{stDone, stPartial, stMissing, stNotTarget, stNothing, "-"} {
		rest, ok := strings.CutPrefix(c, string(s))
		if !ok {
			continue
		}
		if rest == "" || !isLetter(rest[0]) {
			if s == "-" {
				return stNothing, true
			}
			return s, true
		}
	}
	return "", false
}

func isLetter(b byte) bool { return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' }

// problem is one thing the checker refuses, at a page and line.
type problem struct {
	page string
	line int
	msg  string
}

func (p problem) String() string { return fmt.Sprintf("%s:%d: %s", p.page, p.line, p.msg) }

// checker holds what the pages are checked against: the command registry and the test functions
// the repository declares.
type checker struct {
	root     string
	commands map[string]bool // registered command ids
	tests    map[string]bool // Test, Fuzz, Benchmark and Example functions in _test.go files
	pages    map[string][]table
	problems []problem
}

func (c *checker) fail(page string, line int, format string, args ...any) {
	c.problems = append(c.problems, problem{page, line, fmt.Sprintf(format, args...)})
}

// load reads and parses every page; a page that cannot be read is a problem.
func (c *checker) load() {
	c.pages = map[string][]table{}
	for _, p := range progress {
		b, err := os.ReadFile(filepath.Join(c.root, p))
		if err != nil {
			c.fail(p, 0, "cannot read the page: %v", err)
			continue
		}
		c.pages[p] = parseTables(p, string(b))
	}
}

// isHistory reports whether a table is a dated log (a revision history or a progress log), whose
// rows are unique as a whole rather than by their first cell.
func isHistory(t table) bool {
	return strings.HasPrefix(t.section, "Revision history") || len(t.header) > 0 && t.header[0] == "Date"
}

// checklistTables are the tables of a checklist page that carry a status per row.
func (c *checker) checklistTables(page string) []table {
	var out []table
	for _, t := range c.pages[page] {
		if !isHistory(t) && t.col("Status") >= 0 {
			out = append(out, t)
		}
	}
	return out
}

// checkDuplicates refuses a row that appears twice: in a checklist or another table, two rows of
// the same section with the same first cell; in a dated log, two identical rows.
func (c *checker) checkDuplicates() {
	for _, p := range progress {
		seen := map[string]int{}
		for _, t := range c.pages[p] {
			for _, r := range t.rows {
				key := t.section + "\x00" + rowKey(r.cell(0))
				what := "first cell"
				if isHistory(t) {
					key = t.section + "\x00" + rowKey(strings.Join(r.cells, "|"))
					what = "row"
				}
				if first, dup := seen[key]; dup {
					c.fail(p, r.line, "the same %s as line %d in %q: %.80s", what, first, t.section, r.cell(0))
					continue
				}
				seen[key] = r.line
			}
		}
	}
}

var (
	linkTarget = regexp.MustCompile(`\]\(([^)\s]+)\)`)
	codeSpan   = regexp.MustCompile("`([^`]+)`")
	testName   = regexp.MustCompile(`^(Test|Fuzz|Benchmark|Example)[A-Za-z0-9_]*(/\S*)?$`)
	issueRef   = regexp.MustCompile(`(^|[\s(])#[0-9]+\b`)
)

// repoLink is a link target inside the repository: not a URL and not an anchor on the same page.
func repoLink(target string) (string, bool) {
	if strings.HasPrefix(target, "#") || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
		return "", false
	}
	file, _, _ := strings.Cut(target, "#")
	return file, file != ""
}

// hasEvidence reports whether text names evidence as the method demands
// (docs/parity/README.md, rule 3): a file of the code or a picture (a link that is not to another
// page of documentation), a test, or an issue.
func hasEvidence(text string) bool {
	for _, m := range linkTarget.FindAllStringSubmatch(text, -1) {
		if f, ok := repoLink(m[1]); ok && !strings.HasSuffix(f, ".md") {
			return true
		}
	}
	for _, m := range codeSpan.FindAllStringSubmatch(text, -1) {
		if testName.MatchString(m[1]) {
			return true
		}
	}
	return issueRef.MatchString(text)
}

// checkRows refuses, in every checklist, a row whose status cannot be read and a row marked done
// or partial without evidence.
func (c *checker) checkRows() {
	for _, p := range checklists {
		for _, t := range c.checklistTables(p) {
			sc, ec := t.col("Status"), t.col("Evidence")
			for _, r := range t.rows {
				st, ok := parseStatus(r.cell(sc))
				if !ok {
					c.fail(p, r.line, "the status must start with done, partial, missing, not a target or –: %.60q", r.cell(sc))
					continue
				}
				if st.exists() && !hasEvidence(r.cell(sc)+" "+r.cell(ec)) {
					c.fail(p, r.line, "marked %s without evidence (a code file, a test or an issue): %.80s", st, r.cell(0))
				}
			}
		}
	}
}

// checkReferences refuses a link to a file that does not exist on any progress page, and a test
// named in a checklist that no _test.go file declares.
func (c *checker) checkReferences() {
	for _, p := range progress {
		isChecklist := slices.Contains(checklists, p)
		for _, t := range c.pages[p] {
			for _, r := range t.rows {
				for _, cell := range r.cells {
					for _, m := range linkTarget.FindAllStringSubmatch(cell, -1) {
						f, ok := repoLink(m[1])
						if !ok {
							continue
						}
						if _, err := os.Stat(filepath.Join(c.root, path.Dir(p), f)); err != nil {
							c.fail(p, r.line, "links to %s, which does not exist", f)
						}
					}
					if !isChecklist {
						continue
					}
					for _, m := range codeSpan.FindAllStringSubmatch(cell, -1) {
						if !testName.MatchString(m[1]) {
							continue
						}
						name, _, _ := strings.Cut(m[1], "/")
						if !c.tests[name] {
							c.fail(p, r.line, "names %s, which no _test.go file declares", name)
						}
					}
				}
			}
		}
	}
}

var (
	dottedID = regexp.MustCompile(`^[a-z][A-Za-z0-9]*(\.[a-z][A-Za-z0-9]*)+$`)
	toolID   = regexp.MustCompile(`^[a-z][A-Za-z0-9]*(_[a-z][A-Za-z0-9]*)+$`)
	// File extensions and top-level domains: a dotted name ending in one is a file or a host, not a
	// command (`helm.json`, `resource.k8s.io`).
	notCommandEnd = map[string]bool{
		"go": true, "md": true, "png": true, "json": true, "yml": true, "yaml": true, "mod": true, "sum": true,
		"txt": true, "sh": true, "log": true, "toml": true, "plist": true, "svg": true,
		"io": true, "com": true, "org": true, "dev": true, "net": true,
	}
)

// specColumn reports whether column i of t names the target rather than evidence: the first
// column (the behaviour or view), Actions and MCP tool. Command ids there may be planned ones.
func specColumn(t table, i int) bool {
	return i == 0 || t.header[i] == "Actions" || t.header[i] == "MCP tool"
}

// commandIn returns the command id a code span names, if it looks like one: a dotted id, or an
// MCP tool name (dots as underscores), whose first segment is one of the namespaces.
func commandIn(span string, namespaces map[string]bool) (string, bool) {
	id := span
	if toolID.MatchString(span) {
		id = strings.ReplaceAll(span, "_", ".")
	} else if !dottedID.MatchString(span) {
		return "", false
	}
	segs := strings.Split(id, ".")
	if notCommandEnd[segs[len(segs)-1]] || !namespaces[segs[0]] {
		return "", false
	}
	return id, true
}

// namespaces are the first segments of the registered ids and of the ids the checklists plan.
func (c *checker) namespaces() map[string]bool {
	ns := map[string]bool{}
	for id := range c.commands {
		ns[strings.Split(id, ".")[0]] = true
	}
	for _, p := range checklists {
		for _, t := range c.checklistTables(p) {
			for i := range t.header {
				if !specColumn(t, i) {
					continue
				}
				for _, r := range t.rows {
					for _, m := range codeSpan.FindAllStringSubmatch(r.cell(i), -1) {
						s := m[1]
						if toolID.MatchString(s) && t.header[i] == "MCP tool" {
							s = strings.ReplaceAll(s, "_", ".")
						}
						if dottedID.MatchString(s) {
							ns[strings.Split(s, ".")[0]] = true
						}
					}
				}
			}
		}
	}
	for end := range notCommandEnd {
		delete(ns, end)
	}
	return ns
}

// checkCommands refuses a command id in a checklist's status or evidence that the registry does
// not know. The first column, Actions and MCP tool name the target and may plan an id; anywhere
// else an id is a claim about the code.
func (c *checker) checkCommands() {
	ns := c.namespaces()
	for _, p := range checklists {
		for _, t := range c.checklistTables(p) {
			for _, r := range t.rows {
				for i, cell := range r.cells {
					if i < len(t.header) && specColumn(t, i) {
						continue
					}
					for _, m := range codeSpan.FindAllStringSubmatch(cell, -1) {
						if id, ok := commandIn(m[1], ns); ok && !c.commands[id] {
							c.fail(p, r.line, "names the command %s, which the registry does not know", m[1])
						}
					}
				}
			}
		}
	}
}

// tally counts the rows of one component.
type tally struct{ total, exist, done, partial int }

func (t *tally) add(s status, weight int) {
	if !s.counted() {
		return
	}
	t.total += weight
	switch s {
	case stDone:
		t.exist += weight
		t.done += weight
	case stPartial:
		t.exist += weight
		t.partial += weight
	}
}

// counts are the numbers the progress pages report, measured from the checklists.
type counts struct {
	views, actions   tally
	uiFree, uiPaid   tally
	uiOutside        int // interface rows that are not a target
	platforms        tally
	connect          tally
	platformAll      tally // every platform row: packages, authentication, versions, tools, display
	platformOutside  int   // platform rows that are paid (another gap) or have nothing to build
	beyond           tally
	plannedActions   []string // action ids in the checklists that the registry does not know
	notTargetActions []string
}

var viewsInRow = regexp.MustCompile(`\((\d+) views\b`)

// count measures the components of feature breadth and the beyond-Lens rows
// (docs/parity/README.md#how-make-parity-counts).
func (c *checker) count() counts {
	var n counts
	c.countViewsAndActions(&n)
	for _, t := range c.checklistTables(pageUI) {
		pc, sc := t.col("Plan"), t.col("Status")
		if pc < 0 {
			c.fail(pageUI, t.line, "the table in %q has no Plan column", t.section)
			continue
		}
		for _, r := range t.rows {
			st, _ := parseStatus(r.cell(sc))
			if !st.counted() {
				n.uiOutside++
				continue
			}
			switch r.cell(pc) {
			case "free":
				n.uiFree.add(st, 1)
			case "paid":
				n.uiPaid.add(st, 1)
			default:
				c.fail(pageUI, r.line, "the plan must be free or paid: %q", r.cell(pc))
			}
		}
	}
	for _, t := range c.checklistTables(pagePlatforms) {
		sc := t.col("Status")
		for _, r := range t.rows {
			st, _ := parseStatus(r.cell(sc))
			if !st.counted() || strings.Contains(r.cell(0), "(paid)") {
				n.platformOutside++
				continue
			}
			n.platformAll.add(st, 1)
			switch t.section {
			case "Operating systems and packages":
				n.platforms.add(st, 1)
			case "Cluster authentication and connectivity":
				n.connect.add(st, 1)
			}
		}
	}
	for _, t := range c.checklistTables(pageBeyond) {
		sc := t.col("Status")
		for _, r := range t.rows {
			st, _ := parseStatus(r.cell(sc))
			n.beyond.add(st, 1)
		}
	}
	return n
}

// countViewsAndActions counts the free views (a row is one view, or the number its first cell
// gives as "(N views") and the free actions: every distinct command id in an Actions column and in
// the first column of a table without one ("Every kind"), except one marked not a target; an
// action exists when the registry has its id.
func (c *checker) countViewsAndActions(n *counts) {
	actions := map[string]bool{}
	for _, t := range c.checklistTables(pageResources) {
		if strings.HasPrefix(t.section, "Paid") {
			continue
		}
		sc, ac := t.col("Status"), t.col("Actions")
		if ac < 0 {
			ac = 0
		}
		for _, r := range t.rows {
			if t.header[0] == "View" {
				st, _ := parseStatus(r.cell(sc))
				w := 1
				if m := viewsInRow.FindStringSubmatch(r.cell(0)); m != nil {
					w, _ = strconv.Atoi(m[1])
				}
				n.views.add(st, w)
			}
			cell := r.cell(ac)
			for _, loc := range codeSpan.FindAllStringSubmatchIndex(cell, -1) {
				id := cell[loc[2]:loc[3]]
				if !dottedID.MatchString(id) {
					continue
				}
				rest := cell[loc[1]:]
				if i := strings.IndexAny(rest, ",;"); i >= 0 {
					rest = rest[:i]
				}
				if strings.Contains(rest, "not a target") {
					if !slices.Contains(n.notTargetActions, id) {
						n.notTargetActions = append(n.notTargetActions, id)
					}
					continue
				}
				actions[id] = true
			}
		}
	}
	for id := range actions {
		n.actions.total++
		if c.commands[id] {
			n.actions.exist++
		} else {
			n.plannedActions = append(n.plannedActions, id)
		}
	}
	slices.Sort(n.plannedActions)
	slices.Sort(n.notTargetActions)
}
