package model

import (
	"strconv"
	"strings"
	"time"
)

// ColumnType says how a column's cells sort and render.
type ColumnType string

// Column types.
const (
	ColumnText   ColumnType = "text"   // sorts by Cell.Text, case-insensitively
	ColumnNumber ColumnType = "number" // sorts by Cell.Num
	ColumnAge    ColumnType = "age"    // Cell.Num is a Unix timestamp in seconds; rendered as an age relative to now
	// ColumnWarning is Lens's warning-icon column right after Name. A cell with a Status shows an
	// icon in that status's colour and its Text (the reason) as the icon's tooltip; an empty cell
	// shows nothing. Num is 1 when the row has a warning, so that the column sorts warnings
	// together; [CellText] returns the reason.
	ColumnWarning ColumnType = "warning"
	// ColumnUntil is a time in the future: Cell.Num is a Unix timestamp in seconds, rendered as the
	// time left from now ([FormatUntil]): a Cron Job's Next execution.
	ColumnUntil ColumnType = "until"
)

// Column describes one table column of a kind.
type Column struct {
	ID     string     `json:"id"`              // stable id, lowerCamelCase: "name", "clusterIP"
	Title  string     `json:"title"`           // header text as Lens shows it: "Cluster IP"
	Type   ColumnType `json:"type"`            // sort and render behaviour
	Width  float32    `json:"width,omitempty"` // default width hint in points; 0 lets the UI decide
	Hidden bool       `json:"hidden,omitempty"`
}

// Cell is one table cell. Text is what the user reads; Num is the sort key for number, age and
// warning columns. Status colours the cell itself (a Status or Conditions column), independent of
// the row's status.
//
// Link, when set, is the object the cell names (a Namespace, Node, Controlled By or Involved
// Object cell), and the UI renders the cell as a link. Its Kind is a kind key; an empty Context
// means the row's context. A link to a namespace in a list's Namespace column narrows the open
// view's namespace filter to it (ui.setNamespace, owner 2026-10-11); other links go to the object
// (ui.navigate); a link with an empty Name opens the list view of its kind (ui.openView), as a
// definition's Resource cell does. Rows are shared and immutable, and so are their links.
//
// Parts, when set, colours the cell in runs, each in its own status colour, drawn in order with a
// space between them: a cordoned node's Conditions read "SchedulingDisabled Ready" with only
// "Ready" green. Text is then the parts' texts joined by single spaces, so that sorting, search and
// CellText work on the plain text, and Status is the cell's overall status. Parts is a pointer so
// that Cell stays comparable; like links, parts are shared and immutable. Build it with
// [PartsCell].
type Cell struct {
	Text   string      `json:"text,omitempty"`
	Num    int64       `json:"num,omitempty"`
	Status Status      `json:"status,omitempty"`
	Link   *ObjectRef  `json:"link,omitempty"`
	Parts  *[]CellPart `json:"parts,omitempty"`
}

// CellPart is one run of a cell's text in its own status colour; StatusNone draws it in the
// foreground colour.
type CellPart struct {
	Text   string `json:"text"`
	Status Status `json:"status,omitempty"`
}

// PartsCell returns a cell drawn in coloured parts, with status as the cell's overall status.
// Empty parts are dropped; a single part needs no parts and becomes a plain cell with that part's
// text.
func PartsCell(status Status, parts ...CellPart) Cell {
	kept := parts[:0:0]
	for _, p := range parts {
		if p.Text != "" {
			kept = append(kept, p)
		}
	}
	switch len(kept) {
	case 0:
		return Cell{Status: status}
	case 1:
		return Cell{Text: kept[0].Text, Status: status}
	}
	n := len(kept) - 1
	for _, p := range kept {
		n += len(p.Text)
	}
	var b strings.Builder
	b.Grow(n)
	for i, p := range kept {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(p.Text)
	}
	return Cell{Text: b.String(), Status: status, Parts: &kept}
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
	case ColumnUntil:
		if cell.Num == 0 {
			return cell.Text
		}
		return FormatUntil(time.Unix(cell.Num, 0).Sub(now))
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
