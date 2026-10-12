package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// A claim is a number a progress page reports in its text: a pattern whose groups must read as
// the counted numbers. A page that no longer has the pattern fails too, so that a reworded summary
// is noticed instead of going unchecked.
type claim struct {
	page  string
	scope func(text string) string // the part of the page the pattern is looked for in
	re    *regexp.Regexp
	want  func(n counts, breadth float64) []int
	what  string
}

// summary is a page's "**Summary" paragraph.
func summary(text string) string { return paragraph(text, "**Summary") }

// paragraph is the paragraph of text that starts with prefix, up to the next blank line.
func paragraph(text, prefix string) string {
	i := strings.Index(text, "\n"+prefix)
	if i < 0 {
		return ""
	}
	rest := text[i+1:]
	if j := strings.Index(rest, "\n\n"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

// tableRow is the line of a table row whose first cell starts with prefix, inside the section
// whose heading starts with section.
func tableRow(section, prefix string) func(string) string {
	return func(text string) string {
		in := false
		for _, l := range strings.Split(text, "\n") {
			if h, ok := heading(l); ok {
				in = strings.HasPrefix(h, section)
				continue
			}
			if in && strings.HasPrefix(l, "| "+prefix) {
				return l
			}
		}
		return ""
	}
}

func ints(v ...int) []int { return v }

// claims are the numbers ROADMAP.md and the progress pages report about breadth.
var claims = []claim{
	{pageRoadmap, tableRow("Headline numbers", "**Feature breadth**"), regexp.MustCompile(`\*\*~(\d+)%\*\*`),
		func(_ counts, b float64) []int { return ints(int(math.Round(b))) }, "feature breadth"},
	{pageRoadmap, tableRow("Headline numbers", "**Feature breadth**"), regexp.MustCompile(`(\d+) of (\d+) views`),
		func(n counts, _ float64) []int { return ints(n.views.exist, n.views.total) }, "views"},
	{pageRoadmap, tableRow("Headline numbers", "**Feature breadth**"), regexp.MustCompile(`(\d+) of (\d+) actions`),
		func(n counts, _ float64) []int { return ints(n.actions.exist, n.actions.total) }, "actions"},
	{pageRoadmap, tableRow("Headline numbers", "**Feature breadth**"), regexp.MustCompile(`(\d+) of (\d+) interface rows`),
		func(n counts, _ float64) []int { return ints(n.uiFree.exist, n.uiFree.total) }, "free interface rows"},
	{pageRoadmap, tableRow("Headline numbers", "**Beyond Lens**"), regexp.MustCompile(`(\d+) of (\d+) rows done, (\d+) partial`),
		func(n counts, _ float64) []int { return ints(n.beyond.done, n.beyond.total, n.beyond.partial) }, "beyond-Lens rows"},
	{pageTarget, tableRow("Headline", "**Feature breadth**"), regexp.MustCompile(`\*\*~(\d+)%\*\*`),
		func(_ counts, b float64) []int { return ints(int(math.Round(b))) }, "feature breadth"},
	{pageResources, summary, regexp.MustCompile(`(\d+) of (\d+) resource views\s+exist`),
		func(n counts, _ float64) []int { return ints(n.views.exist, n.views.total) }, "views"},
	{pageResources, summary, regexp.MustCompile(`(\d+) of (\d+) actions\s+exist`),
		func(n counts, _ float64) []int { return ints(n.actions.exist, n.actions.total) }, "actions"},
	{pageUI, summary, regexp.MustCompile(`(\d+) of (\d+) rows\s+exist`),
		func(n counts, _ float64) []int {
			return ints(n.uiFree.exist+n.uiPaid.exist, n.uiFree.total+n.uiPaid.total)
		}, "interface rows"},
	{pageUI, summary, regexp.MustCompile(`(\d+) of the (\d+)\s+free-plan rows \((\d+) partial, (\d+) done`),
		func(n counts, _ float64) []int {
			return ints(n.uiFree.exist, n.uiFree.total, n.uiFree.partial, n.uiFree.done)
		}, "free interface rows"},
	{pageUI, summary, regexp.MustCompile(`(\d+) more rows are\s+not a target`),
		func(n counts, _ float64) []int { return ints(n.uiOutside) }, "interface rows that are not a target"},
	{pagePlatforms, summary, regexp.MustCompile(`(\d+) of (\d+) rows\s+partial,\s+(\d+|none)\s+done`),
		func(n counts, _ float64) []int {
			return ints(n.platformAll.partial, n.platformAll.total, n.platformAll.done)
		}, "platform rows"},
	{pagePlatforms, summary, regexp.MustCompile(`\((\d+) of (\d+)\s+package rows\)`),
		func(n counts, _ float64) []int { return ints(n.platforms.exist, n.platforms.total) }, "package rows"},
	{pagePlatforms, summary, regexp.MustCompile(`\((\d+) of (\d+)\s+authentication rows\)`),
		func(n counts, _ float64) []int { return ints(n.connect.exist, n.connect.total) }, "authentication rows"},
	{pageBeyond, summary, regexp.MustCompile(`(\d+) of (\d+) rows\s+exist:\s+(\d+) done and (\d+) partial`),
		func(n counts, _ float64) []int {
			return ints(n.beyond.exist, n.beyond.total, n.beyond.done, n.beyond.partial)
		}, "beyond-Lens rows"},
}

// breadthComponent maps the first cell of a row of target-app-parity's breadth table to the
// counted numbers; nil for the paid features, which come from Lens's premium list, not a checklist.
func breadthComponent(first string, n counts) *tally {
	switch {
	case strings.HasPrefix(first, "Resource views"):
		return &n.views
	case strings.HasPrefix(first, "Resource actions"):
		return &n.actions
	case strings.HasPrefix(first, "Interface"):
		return &n.uiFree
	case strings.HasPrefix(first, "Cluster authentication"):
		return &n.connect
	case strings.HasPrefix(first, "Platforms"):
		return &n.platforms
	}
	return nil
}

var (
	fraction = regexp.MustCompile(`^(\d+) / (\d+)$`)
	percent  = regexp.MustCompile(`([\d.]+)%`)
)

// checkBreadth compares target-app-parity's breadth table with the counts and returns the
// weighted breadth from the counts (and the paid features' row as the table gives it).
func (c *checker) checkBreadth(n counts) (float64, [2]int) {
	var paid [2]int
	var t *table
	for _, tb := range c.pages[pageTarget] {
		if tb.section == "Feature breadth" && tb.col("Ours / Lens") >= 0 {
			t = &tb
			break
		}
	}
	if t == nil {
		c.fail(pageTarget, 0, "no table with an \"Ours / Lens\" column under \"Feature breadth\"")
		return 0, paid
	}
	wc, oc, pc := t.col("Weight"), t.col("Ours / Lens"), t.col("%")
	weighted, total := 0.0, 0.0
	var weightedRow *row
	for _, r := range t.rows {
		if strings.Contains(r.cell(0), "Weighted") {
			weightedRow = &r
			continue
		}
		w, err := strconv.ParseFloat(strings.TrimSuffix(r.cell(wc), "%"), 64)
		m := fraction.FindStringSubmatch(r.cell(oc))
		if err != nil || m == nil {
			c.fail(pageTarget, r.line, "cannot read the weight %q or the fraction %q", r.cell(wc), r.cell(oc))
			continue
		}
		ours, _ := strconv.Atoi(m[1])
		lens, _ := strconv.Atoi(m[2])
		if k := breadthComponent(r.cell(0), n); k != nil {
			if ours != k.exist || lens != k.total {
				c.fail(pageTarget, r.line, "%.40s: the table says %d / %d, the checklists count %d / %d", r.cell(0), ours, lens, k.exist, k.total)
			}
			ours, lens = k.exist, k.total
		} else {
			paid = [2]int{ours, lens}
		}
		share := 0.0
		if lens > 0 {
			share = 100 * float64(ours) / float64(lens)
		}
		if p := percent.FindStringSubmatch(r.cell(pc)); p == nil || p[1] != fmt.Sprintf("%.1f", share) {
			c.fail(pageTarget, r.line, "%.40s: the %% column says %q, %d / %d is %.1f%%", r.cell(0), r.cell(pc), ours, lens, share)
		}
		weighted += w * share / 100
		total += w
	}
	if total != 100 {
		c.fail(pageTarget, t.line, "the weights add up to %g%%, not 100%%", total)
	}
	if weightedRow == nil {
		c.fail(pageTarget, t.line, "no Weighted row in the breadth table")
	} else if p := percent.FindStringSubmatch(weightedRow.cell(pc)); p == nil || p[1] != fmt.Sprintf("%.1f", weighted) {
		c.fail(pageTarget, weightedRow.line, "the weighted breadth says %q, the counts give %.1f%%", weightedRow.cell(pc), weighted)
	}
	return weighted, paid
}

// checkClaims compares every claim with the counts.
func (c *checker) checkClaims(n counts, breadth float64) {
	texts := map[string]string{}
	for _, cl := range claims {
		text, ok := texts[cl.page]
		if !ok {
			b, _ := os.ReadFile(filepath.Join(c.root, cl.page))
			text = string(b)
			texts[cl.page] = text
		}
		part := cl.scope(text)
		m := cl.re.FindStringSubmatch(part)
		if part == "" || m == nil {
			c.fail(cl.page, 0, "the page no longer reports the %s where the checker looks (%s); keep the sentence's form", cl.what, cl.re)
			continue
		}
		want := cl.want(n, breadth)
		line := lineOf(text, part, m[0])
		for i, w := range want {
			got := m[i+1]
			if got == "none" {
				got = "0"
			}
			if got != strconv.Itoa(w) {
				c.fail(cl.page, line, "the %s: the page says %q, the checklists count %v", cl.what, m[0], want)
				break
			}
		}
	}
}

// lineOf is the 1-based line of text on which match, found inside part, begins.
func lineOf(text, part, match string) int {
	i := strings.Index(text, part)
	j := strings.Index(part, match)
	if i < 0 || j < 0 {
		return 0
	}
	return strings.Count(text[:i+j], "\n") + 1
}
