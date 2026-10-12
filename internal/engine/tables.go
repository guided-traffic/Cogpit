package engine

import (
	"encoding/csv"
	"io"
	"slices"
	"strconv"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// What a view's table shows besides its rows, as the UI state asks (architecture §4.3): which
// columns are hidden, and the Definitions view's group filter.

// groupColumn is the column of the Definitions view that its group filter matches.
const groupColumn = "group"

// ColumnsShown returns a kind's columns with Hidden set as a view hides them: the kind's defaults
// when the view has no entry in UIState.HiddenColumns, else exactly the ids listed there. The
// columns keep their order and indexes, so rows built for the kind fit them.
func ColumnsShown(cols []model.Column, hidden []string, set bool) []model.Column {
	if !set {
		return cols
	}
	out := slices.Clone(cols)
	for i := range out {
		out[i].Hidden = slices.Contains(hidden, out[i].ID)
	}
	return out
}

// ColumnOrder returns the order a view shows a kind's columns in, as indexes into cols: the
// columns ids names, in its order (unknown and repeated ids skipped), then every other column in
// the kind's order. It returns nil for no ids and for the kind's own order.
func ColumnOrder(cols []model.Column, ids []string) []int {
	if len(ids) == 0 {
		return nil
	}
	out := make([]int, 0, len(cols))
	used := make([]bool, len(cols))
	for _, id := range ids {
		for i, c := range cols {
			if c.ID == id && !used[i] {
				used[i] = true
				out = append(out, i)
				break
			}
		}
	}
	for i := range cols {
		if !used[i] {
			out = append(out, i)
		}
	}
	for i, ci := range out {
		if i != ci {
			return out
		}
	}
	return nil
}

// HiddenByDefault returns the ids of the columns a kind hides by default.
func HiddenByDefault(cols []model.Column) []string {
	var ids []string
	for _, c := range cols {
		if c.Hidden {
			ids = append(ids, c.ID)
		}
	}
	return ids
}

// FiltersByGroup reports whether a view has the API group filter (UIState.Groups): the
// Definitions view, where Lens shows "All groups" in place of the namespace filter.
func FiltersByGroup(view string) bool { return view == kinds.KeyCustomResourceDefinitions }

// groupFilter returns the index of the column the group filter matches and the groups it keeps,
// or -1 when the view is not filtered by group.
func groupFilter(p *model.KindProfile, groups []string) (int, map[string]struct{}) {
	if !FiltersByGroup(p.Key) || len(groups) == 0 {
		return -1, nil
	}
	ci := p.ColumnIndex(groupColumn)
	if ci < 0 {
		return -1, nil
	}
	set := make(map[string]struct{}, len(groups))
	for _, g := range groups {
		set[g] = struct{}{}
	}
	return ci, set
}

// WriteCSV writes a table snapshot as CSV, as the table shows it (Lens's download button beside
// the item count; its format is inferred): a header row with the titles of the visible columns in
// their shown order (DisplayOrder), then one record per row in the snapshot's order (filtered and
// sorted), each cell's text as the table renders it at now (ages relative to now). The snapshot
// holds only what its cells show, so
// a metadata-only kind (Secrets) exports no values.
//
// Cluster data is not trusted: a cell that a spreadsheet would read as a formula (one starting
// with =, +, -, @, a tab or a carriage return, unless it is a plain number) is written with a
// leading apostrophe, so that opening the file does not run an event message as a formula.
func WriteCSV(w io.Writer, t *TableSnapshot, now time.Time) error {
	cw := csv.NewWriter(w)
	var visible []int
	header := make([]string, 0, len(t.Columns))
	for _, i := range t.DisplayOrder() { // the columns as the user arranged them
		if c := t.Columns[i]; !c.Hidden {
			visible = append(visible, i)
			header = append(header, csvSafe(c.Title))
		}
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	record := make([]string, len(visible))
	for ri := range t.Rows {
		row := &t.Rows[ri]
		for i, ci := range visible {
			record[i] = csvSafe(model.CellText(t.Columns[ci], cellAt(row, ci), now))
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// csvSafe neutralises a cell a spreadsheet would evaluate as a formula (OWASP's advice on CSV
// injection): a leading =, +, -, @, tab or carriage return gets an apostrophe in front. Plain
// numbers ("-1") and a lone "-" stay as they are.
func csvSafe(s string) string {
	if s == "" || s == "-" {
		return s
	}
	switch s[0] {
	case '=', '+', '-', '@', '\t', '\r':
		if _, err := strconv.ParseFloat(s, 64); err == nil {
			return s
		}
		return "'" + s
	}
	return s
}
