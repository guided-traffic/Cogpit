package fyneui

import (
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
)

// Copying the text of a list's cells (D-25, the owner's addition of 2026-10-13): names in Pods,
// Deployments and every other view are copyable with the drawer's copy icon on hover, with the same
// checkmark after a copy. As in the drawer it is a GUI behaviour and no command (TestCopyHasNoCommand).
//
// Cogpit's defaults (the lead's proposals, not the owner's decision, 2026-10-13):
//   - every cell that shows text has the icon, not only the Name column's; a cell that shows no
//     text has none: the checkbox, the ⋮, the warning triangle, a pod's container squares, an
//     empty cell and the placeholders "-", "–", "—" and "<none>";
//   - the icon sits at the right end of the cell, inside it and centred on the row, so that it
//     never covers the next column; a text that reaches the icon is shortened with "…" before it
//     while the icon is there, rather than running under it and looking whole (the tooltip of a
//     cut-off cell shows the whole text);
//   - a click on it copies the cell's full text, not the text shortened with "…", shows the
//     checkmark and the drawer's toast, and neither selects the row, toggles its checkbox nor
//     opens the drawer: the icon is a widget above the table, so the click never reaches the cell;
//   - a Secret's list holds no values (Name, Namespace, Labels, the names of its Keys, Type, Age:
//     kinds/secrets.go, TestSecretDetailShowsNoValue), so a copy from a list is always a plain one
//     (shell.copyText with secret false) and writes no audit line, as the drawer's copy.
//
// One icon per table, moved to the cell under the pointer (a list has 10,000 rows; a widget per
// cell would cost), in a layer above the table as the column drag's chip (columnorder.go). The
// table follows the pointer by cells: setHoverCell hands the icon the cell under the pointer, and
// a pointer move inside the table shows it while the pointer is inside that cell and hides it in
// the gaps and over the header, where Fyne reports no cell.

// tableCopy is a list's copy icon and the cell it is for.
type tableCopy struct {
	tv    *tableView
	btn   *copyButton
	layer *fyne.Container // above the table, without a layout: the icon

	cell     *tableCell    // the cell the icon is for; kept while the pointer is on the icon (entered)
	row, col int           // the cell's row and column when it got the icon
	on       bool          // the icon is placed for cell: the pointer inside lo–hi shows it
	lo, hi   fyne.Position // the cell's visible part, in window coordinates
	stale    bool          // the cells moved under a pointer that did not: the next move looks again
	done     *time.Timer   // the checkmark's
}

func newTableCopy(tv *tableView) *tableCopy {
	tc := &tableCopy{tv: tv}
	tc.btn = newCopyIcon(tv.sh, tc.copyCell, tc.entered, tc.left)
	tc.btn.alt = tc.rowMenu
	tc.layer = container.NewWithoutLayout(tc.btn)
	return tc
}

// rowMenu opens the actions menu of the icon's row on a right-click, as a right-click anywhere
// else on the row does (ui-parity › Tables).
func (tc *tableCopy) rowMenu(e *fyne.PointEvent) {
	if c := tc.cell; c != nil && c.row == tc.row && tc.over(c) {
		tc.tv.openRowMenu(c.row, e.AbsolutePosition)
	}
}

// copyText is what the copy icon copies from a cell: its full text, or "" for a cell that shows
// no text.
func (c *tableCell) copyText() string {
	if c.content.kind != cellText || c.content.marks != nil {
		return ""
	}
	switch strings.TrimSpace(c.content.text) {
	case "", "-", "–", "—", "<none>":
		return ""
	}
	return c.content.text
}

// follow gives the icon to the cell under the pointer, or hides it for nil (setHoverCell). A
// checkmark stays only on the cell it was shown for.
func (tc *tableCopy) follow(c *tableCell) {
	if c == nil {
		tc.hide()
		return
	}
	if c != tc.cell || c.row != tc.row || c.col != tc.col {
		tc.btn.resetConfirm(&tc.done)
		if tc.cell != c {
			tc.room(tc.cell, 0)
		}
	}
	tc.cell, tc.row, tc.col = c, c.row, c.col
	tc.place()
}

// hide hides the icon and gives its cell's text the whole cell again. The cell is kept: the
// pointer may have moved onto the icon (entered).
func (tc *tableCopy) hide() {
	tc.on, tc.stale = false, false
	tc.show(false)
	tc.room(tc.cell, 0)
}

// room keeps the right end of a cell free of text for the icon (w > 0), or gives the text the
// whole cell again (0): the cell shortens its text anew.
func (tc *tableCopy) room(c *tableCell, w float32) {
	if c != nil && c.room != w {
		c.room = w
		c.Refresh()
	}
}

func (tc *tableCopy) show(on bool) {
	if on != tc.btn.Visible() {
		if on {
			tc.btn.Show()
		} else {
			tc.btn.Hide()
		}
	}
}

// place puts the icon at the right end of its cell's visible part, centred on the row, and shows
// it while the pointer is inside that part; a cell without text, or with too little of it in
// view, hides it.
func (tc *tableCopy) place() {
	c, tv := tc.cell, tc.tv
	if c == nil || !tv.pointerIn || !c.Visible() || c.copyText() == "" {
		tc.hide()
		return
	}
	drv := fyne.CurrentApp().Driver()
	at, sz := drv.AbsolutePositionForObject(c), c.Size()
	tp, ts := drv.AbsolutePositionForObject(tv.table), tv.table.Size()
	// The part of the cell in the table's body: the header row, as high as a row (both are a line
	// of text and the row padding), covers the top of a cell scrolled under it.
	lo := fyne.NewPos(max(at.X, tp.X), max(at.Y, tp.Y+sz.Height))
	hi := fyne.NewPos(min(at.X+sz.Width, tp.X+ts.Width), min(at.Y+sz.Height, tp.Y+ts.Height))
	side := min(tc.btn.MinSize().Width, sz.Height)
	if hi.X-lo.X < side || hi.Y-lo.Y < side {
		tc.hide()
		return
	}
	tc.lo, tc.hi, tc.on, tc.stale = lo, hi, true, false
	// The text ends a padding before the icon; it has the cell padding at its right end already.
	th := c.Theme()
	tc.room(c, max(0, at.X+sz.Width-(hi.X-side)+th.Size(theme.SizeNamePadding)-th.Size(sizeCellPadding)))
	y := min(max(at.Y+(sz.Height-side)/2, lo.Y), hi.Y-side)
	tc.btn.Move(fyne.NewPos(hi.X-side, y).Subtract(drv.AbsolutePositionForObject(tc.layer)))
	tc.btn.Resize(fyne.NewSquareSize(side))
	tc.show(tc.inside(tv.pointer))
}

// over reports whether the icon still lies on the cell c: a scroll the table did not report (Fyne
// scrolls on its own for the arrow keys) moves the cells, not the icon.
func (tc *tableCopy) over(c *tableCell) bool {
	if !c.Visible() {
		return false
	}
	drv := fyne.CurrentApp().Driver()
	at := drv.AbsolutePositionForObject(c)
	mid := drv.AbsolutePositionForObject(tc.btn).AddXY(tc.btn.Size().Width/2, tc.btn.Size().Height/2)
	return mid.X >= at.X && mid.X < at.X+c.Size().Width && mid.Y >= at.Y && mid.Y < at.Y+c.Size().Height
}

// inside reports whether a point in window coordinates lies in the icon's cell.
func (tc *tableCopy) inside(p fyne.Position) bool {
	return p.X >= tc.lo.X && p.X < tc.hi.X && p.Y >= tc.lo.Y && p.Y < tc.hi.Y
}

// moved follows the pointer over the table (resourceTable.MouseMoved), after the table moved its
// row band: the icon shows while the pointer is inside its cell and hides in the gaps between
// cells and over the header, where the table keeps its band; once the cells moved under the
// pointer (a scroll, new column widths) it looks for the cell under the pointer again. O(1) per
// move unless the cells moved.
func (tc *tableCopy) moved() {
	tv := tc.tv
	if tc.stale {
		tc.stale = false
		tc.follow(tv.cellUnderPointer(tv.hoverRow, tv.hoverCol))
		return
	}
	if tc.on {
		tc.show(tc.inside(tv.pointer))
	}
}

// moving hides the icon while the cells move under a pointer that does not: a scroll, new column
// widths. The next pointer move puts it on the cell under the pointer again.
func (tc *tableCopy) moving() {
	if !tc.on && !tc.btn.Visible() {
		return
	}
	tc.hide()
	tc.stale = tc.tv.pointerIn
}

// updated follows a cell the table filled (updateCell). The icon's cell keeps it with new text (a
// watch event, an age that advanced: the icon copies the text the cell shows when clicked) and
// loses it without text; a cell the table gave to another row or column moved away from the
// pointer. O(1): nothing but a comparison for every other cell.
func (tc *tableCopy) updated(c *tableCell) {
	if c != tc.cell {
		return
	}
	switch {
	case c.row != tc.row || c.col != tc.col, c.copyText() == "":
		if tc.on || tc.btn.Visible() {
			tc.hide()
			tc.stale = tc.tv.pointerIn
		}
	case !tc.on && tc.tv.pointerIn && tc.tv.hoverRow == c.row && tc.tv.hoverCol == c.col:
		tc.stale = true // the cell under the pointer got text: the next move shows the icon
	}
}

// entered keeps the row band and the icon when the pointer moves from the cell onto the icon: the
// table has just seen the pointer leave (resourceTable.MouseOut), which cleared both.
func (tc *tableCopy) entered(ev *desktop.MouseEvent) {
	c, tv := tc.cell, tc.tv
	if c == nil || c.row != tc.row || c.col != tc.col {
		return
	}
	tv.pointer, tv.pointerIn = ev.AbsolutePosition, true
	tv.setHoverCell(tc.row, tc.col)
}

// left ends what entered kept, as the table's MouseOut does; when the pointer moves back onto the
// table, its MouseIn brings the band and the icon back.
func (tc *tableCopy) left() {
	tc.tv.pointerIn = false
	tc.tv.setHover(-1)
}

// copyCell puts the full text of the icon's cell on the clipboard, as the drawer's icon does: the
// checkmark and the toast, a plain copy (no Secret's value is in a list), no audit line.
func (tc *tableCopy) copyCell() {
	c := tc.cell
	if c == nil || c.row != tc.row || c.col != tc.col || !tc.over(c) {
		tc.hide() // the cells moved under the icon (Fyne's own scroll, as its keys do): not its cell
		return
	}
	text := c.copyText()
	if text == "" {
		return
	}
	tc.tv.sh.copyText(text, false)
	tc.btn.confirm(&tc.done)
	tc.tv.sh.toast("info", copiedToast)
}
