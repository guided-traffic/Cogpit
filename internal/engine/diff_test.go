package engine

import (
	"slices"
	"testing"
)

func TestDiffLines(t *testing.T) {
	old := "a: 1\nb: 2\nc: 3\n"
	edited := "a: 1\r\nb: 20\r\nc: 3\r\nd: 4"
	want := []DiffLine{
		{Op: DiffSame, Text: "a: 1", Old: 1, New: 1},
		{Op: DiffRemoved, Text: "b: 2", Old: 2},
		{Op: DiffAdded, Text: "b: 20", New: 2},
		{Op: DiffSame, Text: "c: 3", Old: 3, New: 3},
		{Op: DiffAdded, Text: "d: 4", New: 4},
	}
	got := DiffLines(old, edited)
	if !slices.Equal(got, want) {
		t.Errorf("DiffLines =\n%+v\nwant\n%+v", got, want)
	}
	if !DiffChanged(got) {
		t.Error("DiffChanged = false for a changed text")
	}
	if d := DiffLines(old, "a: 1\nb: 2\nc: 3"); DiffChanged(d) || len(d) != 3 {
		t.Errorf("a missing final newline is no change: %+v", d)
	}
	if d := DiffLines("", ""); len(d) != 0 {
		t.Errorf("empty texts: %+v", d)
	}
}
