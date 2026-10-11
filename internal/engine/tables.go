package engine

import (
	"slices"

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
