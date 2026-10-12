package main

import (
	"regexp"
	"strings"
)

// table is one GitHub-flavoured Markdown table of a progress page.
type table struct {
	page    string // path relative to the repository root
	section string // the nearest heading above the table, without its hashes
	line    int    // 1-based line of the header row
	header  []string
	rows    []row
}

// row is one body row of a table.
type row struct {
	line  int
	cells []string
}

// col is the index of the column whose header is name, or -1.
func (t *table) col(name string) int {
	for i, h := range t.header {
		if h == name {
			return i
		}
	}
	return -1
}

// cell is the row's cell in column i, or "" when the row is shorter.
func (r row) cell(i int) string {
	if i < 0 || i >= len(r.cells) {
		return ""
	}
	return r.cells[i]
}

var delimiterRow = regexp.MustCompile(`^\|(\s*:?-+:?\s*\|)+\s*$`)

// parseTables returns every table of a page, outside fenced code blocks, each with the heading it
// sits under.
func parseTables(page, text string) []table {
	lines := strings.Split(text, "\n")
	var out []table
	section, fenced := "", false
	for i := 0; i < len(lines); i++ {
		l := lines[i]
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		if h, ok := heading(l); ok {
			section = h
			continue
		}
		if !strings.HasPrefix(l, "|") || i+1 >= len(lines) || !delimiterRow.MatchString(strings.TrimSpace(lines[i+1])) {
			continue
		}
		t := table{page: page, section: section, line: i + 1, header: splitRow(l)}
		j := i + 2
		for ; j < len(lines) && strings.HasPrefix(lines[j], "|"); j++ {
			t.rows = append(t.rows, row{line: j + 1, cells: splitRow(lines[j])})
		}
		out = append(out, t)
		i = j - 1
	}
	return out
}

// heading returns the text of a Markdown heading line (#, ## or ###).
func heading(l string) (string, bool) {
	for _, p := range []string{"### ", "## ", "# "} {
		if strings.HasPrefix(l, p) {
			return strings.TrimSpace(l[len(p):]), true
		}
	}
	return "", false
}

// splitRow splits a table row on its unescaped pipes and trims each cell. GitHub's tables need a
// pipe inside a cell escaped even in a code span, so an unescaped pipe always separates cells.
func splitRow(l string) []string {
	l = strings.TrimSpace(l)
	l = strings.TrimPrefix(l, "|")
	if strings.HasSuffix(l, "|") && !strings.HasSuffix(l, `\|`) {
		l = l[:len(l)-1]
	}
	var cells []string
	var b strings.Builder
	for i := 0; i < len(l); i++ {
		switch {
		case l[i] == '\\' && i+1 < len(l) && l[i+1] == '|':
			b.WriteString(`\|`)
			i++
		case l[i] == '|':
			cells = append(cells, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(l[i])
		}
	}
	return append(cells, strings.TrimSpace(b.String()))
}

var (
	linkText = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	spaces   = regexp.MustCompile(`\s+`)
)

// rowKey is a cell reduced to what a reader sees: links to their text, without code marks and
// emphasis, spaces collapsed, lower case. Two rows with the same key read as the same row.
func rowKey(cell string) string {
	s := linkText.ReplaceAllString(cell, "$1")
	s = strings.NewReplacer("`", "", "**", "", "*", "").Replace(s)
	return strings.ToLower(strings.TrimSpace(spaces.ReplaceAllString(s, " ")))
}
