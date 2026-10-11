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
	Rows       []model.Row    `json:"rows"`
	Sort       SortSpec       `json:"sort"`
	Namespaces []string       `json:"namespaces,omitempty"`
	Filter     string         `json:"filter,omitempty"`
	// Total is the number of objects before filtering.
	Total int `json:"total"`
	// Revision increases with every snapshot of the view.
	Revision uint64 `json:"revision"`
	// Synced is false until the informer finished its initial list.
	Synced bool `json:"synced"`
	// Error is the informer's last error, if any (a forbidden kind, a lost connection).
	Error string `json:"error,omitempty"`
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
