package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

// TestColumnResizeByDraggingHeaderSeparator is the evidence for gate criterion S3: in Fyne 2.8.1
// a user resizes a column by dragging the gap between two header cells. The test drives the same
// sequence as the GLFW driver (internal/driver/glfw/window.go): the pointer moves over the gap
// (the table's hover handler marks the column, Cursor() turns into a resize cursor), the button
// goes down on the table (MouseDown starts the drag at the column's width), and the drag events go
// to the draggable under the press point, which forwards them to Table.Dragged.
func TestColumnResizeByDraggingHeaderSeparator(t *testing.T) {
	f := connectedFake(t, 200)
	sh, w, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	settle(w)
	tv := sh.view
	c := w.Canvas()

	gap := tv.table.Theme().Size(theme.SizeNamePadding)
	before := tv.setWidths[firstDataColumn]
	header := headerFor(t, sh, firstDataColumn)
	// between the Name and Namespace headers; the checkbox's column comes first
	rel := fyne.NewPos(tv.setWidths[checkColumn]+gap+before+gap/2, header.Size().Height/2)
	abs := fyne.CurrentApp().Driver().AbsolutePositionForObject(tv.table).Add(rel)

	test.MoveMouse(c, fyne.CurrentApp().Driver().AbsolutePositionForObject(header).AddXY(10, 5))
	if cur := tv.table.Cursor(); cur != desktop.DefaultCursor {
		t.Fatalf("cursor over a header cell: %v, want the default", cur)
	}
	test.MoveMouse(c, abs)
	if cur := tv.table.Cursor(); cur != desktop.HResizeCursor {
		t.Fatalf("cursor over the header gap: %v, want the horizontal resize cursor", cur)
	}
	tv.table.MouseDown(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: rel, AbsolutePosition: abs}, Button: desktop.MouseButtonPrimary})
	// The press point is in the gap, where no header cell is: the draggable there is the table's
	// header row, which forwards to Table.Dragged (a header cell's drag moves its column instead,
	// columnorder.go). test.Drag would look the draggable up at the end point and send it the
	// start position, so the events are sent as the GLFW driver sends them.
	for _, h := range findAll[*headerCell](tv.table) {
		p := fyne.CurrentApp().Driver().AbsolutePositionForObject(h)
		if h.Visible() && abs.X >= p.X && abs.X < p.X+h.Size().Width && abs.Y >= p.Y && abs.Y < p.Y+h.Size().Height {
			t.Fatalf("the press point is on the header of column %d, not in the gap", h.col)
		}
	}
	tv.table.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: rel.AddXY(60, 0), AbsolutePosition: abs.AddXY(60, 0)}, Dragged: fyne.NewDelta(60, 0)})
	tv.table.DragEnd()
	q.drain()
	settle(w)

	if got := headerFor(t, sh, firstDataColumn).Size().Width; got != before+60 {
		t.Fatalf("Name column %v wide after dragging 60 from %v", got, before)
	}
	if got := tv.userWidths["pods/name"]; got != before+60 {
		t.Fatalf("the shell recorded %v as the user's width, want %v", got, before+60)
	}
	// A refit (window resize, new snapshot) keeps the user's width instead of stretching the column.
	tv.fitWidth = 0
	tv.fitColumns(tv.table.Size().Width + 200)
	if got := tv.setWidths[firstDataColumn]; got != before+60 {
		t.Fatalf("refit changed the dragged width to %v", got)
	}
}
