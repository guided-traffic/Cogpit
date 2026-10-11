package model

import (
	"strconv"
	"time"
)

// ColumnType says how a column's cells sort and render.
type ColumnType string

// Column types.
const (
	ColumnText   ColumnType = "text"   // sorts by Cell.Text, case-insensitively
	ColumnNumber ColumnType = "number" // sorts by Cell.Num
	ColumnAge    ColumnType = "age"    // Cell.Num is a Unix timestamp in seconds; rendered as an age relative to now
)

// Column describes one table column of a kind.
type Column struct {
	ID     string     `json:"id"`              // stable id, lowerCamelCase: "name", "clusterIP"
	Title  string     `json:"title"`           // header text as Lens shows it: "Cluster IP"
	Type   ColumnType `json:"type"`            // sort and render behaviour
	Width  float32    `json:"width,omitempty"` // default width hint in points; 0 lets the UI decide
	Hidden bool       `json:"hidden,omitempty"`
}

// Cell is one table cell. Text is what the user reads; Num is the sort key for number and age
// columns. Status colours the cell itself (a Status or Conditions column), independent of the
// row's status.
type Cell struct {
	Text   string `json:"text,omitempty"`
	Num    int64  `json:"num,omitempty"`
	Status Status `json:"status,omitempty"`
}

// Row is one object as a table row. Rows are immutable once built: snapshots share them.
type Row struct {
	Ref             ObjectRef `json:"ref"`
	UID             string    `json:"uid,omitempty"`
	ResourceVersion string    `json:"resourceVersion,omitempty"`
	Cells           []Cell    `json:"cells"`
	Status          Status    `json:"status,omitempty"`
}

// CellText returns the text a cell shows. Age cells are formatted relative to now, so a row built
// minutes ago still shows the right age.
func CellText(col Column, cell Cell, now time.Time) string {
	switch col.Type {
	case ColumnAge:
		if cell.Num == 0 {
			return cell.Text
		}
		return FormatAge(now.Sub(time.Unix(cell.Num, 0)))
	case ColumnNumber:
		if cell.Text == "" {
			return strconv.FormatInt(cell.Num, 10)
		}
	}
	return cell.Text
}

// FormatAge formats a duration the way Lens's Age column does: the largest unit, "42s", "5m",
// "3h", "12d", "2y".
func FormatAge(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d/time.Second)) + "s"
	case d < time.Hour:
		return strconv.Itoa(int(d/time.Minute)) + "m"
	case d < 24*time.Hour:
		return strconv.Itoa(int(d/time.Hour)) + "h"
	case d < 365*24*time.Hour:
		return strconv.Itoa(int(d/(24*time.Hour))) + "d"
	default:
		return strconv.Itoa(int(d/(365*24*time.Hour))) + "y"
	}
}
