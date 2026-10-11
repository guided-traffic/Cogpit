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
	before := tv.setWidths[0]
	header := headerFor(t, sh, 0)
	rel := fyne.NewPos(before+gap/2, header.Size().Height/2) // between the Name and Namespace headers
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
	test.Drag(c, abs.AddXY(60, 0), 60, 0)
	q.drain()
	settle(w)

	if got := headerFor(t, sh, 0).Size().Width; got != before+60 {
		t.Fatalf("Name column %v wide after dragging 60 from %v", got, before)
	}
	if got := tv.userWidths["pods/name"]; got != before+60 {
		t.Fatalf("the shell recorded %v as the user's width, want %v", got, before+60)
	}
	// A refit (window resize, new snapshot) keeps the user's width instead of stretching the column.
	tv.fitWidth = 0
	tv.fitColumns(tv.table.Size().Width + 200)
	if got := tv.setWidths[0]; got != before+60 {
		t.Fatalf("refit changed the dragged width to %v", got)
	}
}
