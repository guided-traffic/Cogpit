package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDetailTableLink checks the table's links: by row and column from its cells, nil for a plain
// cell, out of range or a table without cells, and no cells in the JSON of a plain table.
func TestDetailTableLink(t *testing.T) {
	sa := &ObjectRef{Kind: "serviceaccounts", Namespace: "kube-system", Name: "kube-proxy"}
	cols := []Column{{ID: "kind", Title: "Kind"}, {ID: "name", Title: "Name"}, {ID: "namespace", Title: "Namespace"}}
	tbl := CellTable(cols, [][]Cell{
		{{Text: "ServiceAccount"}, {Text: "kube-proxy", Link: sa}, {Text: "kube-system"}},
		{{Text: "Group"}, {Text: "system:nodes"}},
	})
	if got := tbl.Link(0, 1); got != sa {
		t.Errorf("Link(0, 1) = %v, want %v", got, sa)
	}
	if got := tbl.Rows[1]; len(got) != 3 || got[2] != "" {
		t.Errorf("a short row is not padded: %q", got)
	}
	for _, c := range [][2]int{{0, 0}, {0, 3}, {1, 1}, {-1, 0}, {0, -1}, {2, 1}} {
		if got := tbl.Link(c[0], c[1]); got != nil {
			t.Errorf("Link(%d, %d) = %v, want nil", c[0], c[1], got)
		}
	}
	var none *DetailTable
	if none.Link(0, 0) != nil {
		t.Error("a nil table has a link")
	}
	plain, err := json.Marshal(DetailTable{Columns: []string{"CPU"}, Rows: [][]string{{"8"}}})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "cells") || strings.Contains(string(plain), "types") {
		t.Errorf("a plain table marshals cells: %s", plain)
	}
}
