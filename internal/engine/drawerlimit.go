package engine

// MaxDrawerTableRows bounds the rows of a sortable drawer table (a node's or a workload's Pods,
// D-40): the drawer shows the first ones in the table's order and counts the rest
// (DetailTable.Omitted). Cogpit's default, not the owner's decision: well above the pods a node
// runs (110 by default, at most a few hundred), so that only a large workload's table is cut. The
// cut follows the sort, so a table sorted by CPU shows the busiest pods.
const MaxDrawerTableRows = 500

// LimitDrawerTables cuts the model's sortable tables to at most max rows, after they were sorted,
// and counts what it left out in Omitted. A table within the bound is left as it is.
func LimitDrawerTables(m *DetailModel, limit int) {
	if limit <= 0 {
		return
	}
	for i := range m.Sections {
		t := m.Sections[i].Table
		if t == nil || !t.Sortable || len(t.Cells) <= limit {
			continue
		}
		nt := *t
		nt.Omitted = t.Omitted + len(t.Cells) - limit
		nt.Cells = t.Cells[:limit:limit]
		if len(t.Rows) > limit {
			nt.Rows = t.Rows[:limit:limit]
		}
		m.Sections[i].Table = &nt
	}
}
