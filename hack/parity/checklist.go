package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

// The generated page (D-15, docs/parity/README.md#the-pages): every row of the checklists that
// carries a status, one list per checklist, with the counts make parity measures. It is written
// from the hand-written pages and never edited by hand: a plain run writes it when it differs, and
// make parity (-check, CI) refuses a page that no longer matches the checklists.
const pageChecklist = "docs/parity/parity-checklist.md"

// maxRowText is how much of a row's first cell the generated page shows.
const maxRowText = 140

// checklistPage returns the generated page's content.
func (c *checker) checklistPage(n counts) string {
	var b strings.Builder
	b.WriteString("# Parity checklist\n\n")
	b.WriteString("> **Generated** by `go run ./hack/parity` from the checklists; do not edit. `make parity` (CI)\n")
	b.WriteString("> fails when this page no longer matches them. Method and rules: [README](README.md);\n")
	b.WriteString("> the assessment: [target-app-parity](target-app-parity.md); the work list: [gaps](gaps.md).\n\n")
	b.WriteString("Every row of the checklists that carries a status, in one place: each Lens Desktop view, action,\n")
	b.WriteString("interface and platform row, and Cogpit's own rows beyond Lens, with its status and the section\n")
	b.WriteString("of the checklist that holds its evidence. The status is the row's first word there.\n\n")

	b.WriteString("## Counts\n\n")
	b.WriteString("| What | Rows | Exist | Done | Partial | Missing |\n|---|---|---|---|---|---|\n")
	line := func(what string, t tally) {
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %d |\n", what, t.total, t.exist, t.done, t.partial, t.total-t.exist)
	}
	line("Resource views (free)", n.views)
	fmt.Fprintf(&b, "| Resource actions (free; exist = registered commands) | %d | %d | – | – | %d |\n",
		n.actions.total, n.actions.exist, n.actions.total-n.actions.exist)
	line("Interface rows, free plan", n.uiFree)
	line("Interface rows, paid plans", n.uiPaid)
	line("Platform rows (packages, authentication, versions, tools, display)", n.platformAll)
	line("Beyond Lens (not in feature breadth)", n.beyond)
	fmt.Fprintf(&b, "\nNot counted: %d interface rows that are not a target, %d platform rows that are paid or have nothing to build.\n",
		n.uiOutside, n.platformOutside)

	b.WriteString("\n## Resource views and actions\n\n")
	c.writeRows(&b, pageResources)
	b.WriteString("\n## Interface\n\n")
	c.writeRows(&b, pageUI)
	b.WriteString("\n## Platforms\n\n")
	c.writeRows(&b, pagePlatforms)
	b.WriteString("\n## Beyond Lens\n\n")
	c.writeRows(&b, pageBeyond)
	return b.String()
}

// writeRows writes one table of the rows of a checklist's status tables: the row, its status and
// its section, linked.
func (c *checker) writeRows(b *strings.Builder, page string) {
	file := strings.TrimPrefix(page, "docs/parity/")
	b.WriteString("| Row | Status | Section |\n|---|---|---|\n")
	for _, t := range c.checklistTables(page) {
		sc := t.col("Status")
		where := fmt.Sprintf("[%s](%s#%s)", escapeCell(t.section), file, slug(t.section))
		for _, r := range t.rows {
			st, _ := parseStatus(r.cell(sc))
			fmt.Fprintf(b, "| %s | %s | %s |\n", rowText(r.cell(0)), st, where)
		}
	}
}

var emphasis = strings.NewReplacer("**", "", "`", "")

// rowText is a row's first cell as plain text: links to their text, no code marks or emphasis,
// spaces collapsed, cut at maxRowText runes.
func rowText(cell string) string {
	s := linkText.ReplaceAllString(cell, "$1")
	s = strings.TrimSpace(spaces.ReplaceAllString(emphasis.Replace(s), " "))
	if r := []rune(s); len(r) > maxRowText {
		s = strings.TrimSpace(string(r[:maxRowText-1])) + "…"
	}
	return escapeCell(s)
}

// escapeCell escapes the pipes a cell's text holds, leaving escaped ones as they are.
func escapeCell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, `\|`, "|"), "|", `\|`)
}

var slugDrop = regexp.MustCompile(`[^\p{L}\p{N} _-]`)

// slug is GitHub's anchor of a heading: lower case, everything but letters, digits, spaces,
// hyphens and underscores dropped, spaces turned into hyphens.
func slug(h string) string {
	s := strings.ToLower(linkText.ReplaceAllString(h, "$1"))
	s = slugDrop.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return '-'
		}
		return r
	}, s)
}

// syncChecklist compares the generated page with the file: with write, a page that differs is
// written (and reported); without, it is a problem.
func (c *checker) syncChecklist(n counts, write bool) (wrote bool) {
	want := c.checklistPage(n)
	path := filepath.Join(c.root, pageChecklist)
	have, err := os.ReadFile(path)
	if err == nil && string(have) == want {
		return false
	}
	if !write {
		c.fail(pageChecklist, 0, "does not match the checklists: run go run ./hack/parity, which writes it, and commit it")
		return false
	}
	if err := os.WriteFile(path, []byte(want), 0o644); err != nil { //nolint:gosec // a documentation page, world-readable as the rest of the repository
		c.fail(pageChecklist, 0, "cannot write the page: %v", err)
		return false
	}
	return true
}
