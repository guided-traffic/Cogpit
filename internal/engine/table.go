package engine

import (
	"github.com/guided-traffic/cogpit/internal/model"
)

// TableSnapshot is one immutable state of a view's table: the rows that pass the namespace and
// search filters, sorted. A newer snapshot replaces it; nobody mutates it.
type TableSnapshot struct {
	View       string         `json:"view"`
	Context    string         `json:"context"`
	Title      string         `json:"title"`
	Namespaced bool           `json:"namespaced"`
	Columns    []model.Column `json:"columns"`
	// Order is the order the table shows its columns in, as indexes into Columns (hidden ones
	// included), when the user arranged them (UIState.ColumnOrder); nil: Columns' order. Columns
	// keep the kind's order and indexes, so that rows built for the kind fit them.
	Order      []int       `json:"order,omitempty"`
	Rows       []model.Row `json:"rows"`
	Sort       SortSpec    `json:"sort"`
	Namespaces []string    `json:"namespaces,omitempty"`
	Filter     string      `json:"filter,omitempty"`
	// FilterOptions are the search's toggles the rows were filtered with; FilterError says why the
	// search filtered nothing out (an invalid regular expression), "" when it applied.
	FilterOptions FilterOptions `json:"filterOptions,omitzero"`
	FilterError   string        `json:"filterError,omitempty"`
	// Total is the number of objects before filtering.
	Total int `json:"total"`
	// Revision increases with every snapshot of the view.
	Revision uint64 `json:"revision"`
	// Synced is false until the informer finished its initial list.
	Synced bool `json:"synced"`
	// Error is the informer's last error, if any (a forbidden kind, a lost connection).
	Error string `json:"error,omitempty"`
	// Watched are the namespaces the kind is read in one by one, because the user may not list it
	// in all namespaces (accessiblens_plan.go); empty: in all namespaces.
	Watched []string `json:"watchedNamespaces,omitempty"`
	// Denied are the namespaces named for the kind whose access reviews deny listing it there: not
	// read, a state the view shows.
	Denied []string `json:"deniedNamespaces,omitempty"`
	// MetricsHidden are the columns the cluster's settings hide (Cluster Settings › Metrics,
	// hidemetrics.go) and the view itself shows: Hidden in Columns, yet not among the view's own
	// hidden columns, which ui.setColumns sets (UIState.HiddenColumns).
	MetricsHidden []string `json:"metricsHidden,omitempty"`
	// ColumnTips are tooltips of column headers by column id: the Nodes list's Disk says where its
	// numbers come from, or why it is empty (no Prometheus, no node-exporter series).
	ColumnTips map[string]string `json:"columnTips,omitempty"`
}

// DisplayOrder returns the indexes of Columns in the order the table shows them: Order, or
// Columns' own order.
func (t *TableSnapshot) DisplayOrder() []int {
	if t.Order != nil {
		return t.Order
	}
	out := make([]int, len(t.Columns))
	for i := range out {
		out[i] = i
	}
	return out
}

// IndexOf returns the index of the row for ref, or -1.
func (t *TableSnapshot) IndexOf(ref model.ObjectRef) int {
	if t == nil {
		return -1
	}
	for i := range t.Rows {
		if t.Rows[i].Ref.Same(ref) {
			return i
		}
	}
	return -1
}
