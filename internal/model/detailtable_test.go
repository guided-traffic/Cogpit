package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestDetailTableLink checks the table's links: by row and column, nil for a plain cell, out of
// range or without links, and left out of the JSON when the table has none.
func TestDetailTableLink(t *testing.T) {
	sa := &ObjectRef{Kind: "serviceaccounts", Namespace: "kube-system", Name: "kube-proxy"}
	tbl := &DetailTable{
		Columns: []string{"Kind", "Name", "Namespace"},
		Rows:    [][]string{{"ServiceAccount", "kube-proxy", "kube-system"}, {"Group", "system:nodes", ""}},
		Links:   [][]*ObjectRef{{nil, sa, nil}},
	}
	if got := tbl.Link(0, 1); got != sa {
		t.Errorf("Link(0, 1) = %v, want %v", got, sa)
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
	if strings.Contains(string(plain), "links") {
		t.Errorf("a table without links marshals them: %s", plain)
	}
}
