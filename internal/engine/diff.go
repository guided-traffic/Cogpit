package engine

import (
	"strings"

	"github.com/pmezard/go-difflib/difflib"
)

// DiffOp says what happened to one line of a diff.
type DiffOp string

// Diff operations.
const (
	DiffSame    DiffOp = "="
	DiffRemoved DiffOp = "-"
	DiffAdded   DiffOp = "+"
)

// DiffLine is one line of a line diff: the editor's diff of an edited object against the object
// as the cache holds it (G9).
type DiffLine struct {
	Op   DiffOp `json:"op"`
	Text string `json:"text"`
	Old  int    `json:"old,omitempty"` // 1-based line in the old text; 0 for an added line
	New  int    `json:"new,omitempty"` // 1-based line in the new text; 0 for a removed line
}

// DiffLines returns the line diff from before to after: unchanged lines, then for each change the
// removed lines before the added ones. Line ends are normalised to "\n"; a final newline does not
// make a line of its own.
func DiffLines(before, after string) []DiffLine {
	a, b := splitLines(before), splitLines(after)
	m := difflib.NewMatcher(a, b)
	out := make([]DiffLine, 0, max(len(a), len(b)))
	for _, op := range m.GetOpCodes() {
		switch op.Tag {
		case 'e':
			for i := op.I1; i < op.I2; i++ {
				out = append(out, DiffLine{Op: DiffSame, Text: a[i], Old: i + 1, New: op.J1 + i - op.I1 + 1})
			}
		case 'd', 'r', 'i':
			for i := op.I1; i < op.I2; i++ {
				out = append(out, DiffLine{Op: DiffRemoved, Text: a[i], Old: i + 1})
			}
			for j := op.J1; j < op.J2; j++ {
				out = append(out, DiffLine{Op: DiffAdded, Text: b[j], New: j + 1})
			}
		}
	}
	return out
}

// DiffChanged reports whether a diff has any added or removed line.
func DiffChanged(d []DiffLine) bool {
	for _, l := range d {
		if l.Op != DiffSame {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
