package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// The column list behind the ⋮ at the end of the header row (ui-parity › Tables): one check per
// column of the view, checked when the column shows. Hidden columns are engine state
// (UIState.HiddenColumns, set with ui.setColumns), so the list shows what the snapshot says and a
// toggle only dispatches; the snapshot that follows hides or shows the column.

// openColumns shows the open view's columns below the header's ⋮, at the table's right edge.
// Checks toggle without closing the list, as the namespace filter's do.
func (tv *tableView) openColumns() {
	if tv.snap == nil {
		return
	}
	tv.colView = tv.snap.View
	tv.colChecks, tv.colIDs = tv.colChecks[:0], tv.colIDs[:0]
	list := container.NewVBox()
	for _, c := range tv.snap.Columns {
		id := c.ID
		ch := widget.NewCheck(c.Title, func(on bool) { tv.toggleColumn(id, on) })
		tv.colChecks = append(tv.colChecks, ch)
		tv.colIDs = append(tv.colIDs, id)
		list.Add(ch)
	}
	tv.syncColumnChecks()
	th := tv.table.Theme()
	scroll := container.NewVScroll(list)
	header := newHeaderCell(nil).MinSize().Height
	// The list may reach the table's bottom: a kind has a dozen columns or so, and seeing them all
	// at once is the point of the list.
	h := min(list.MinSize().Height, max(tv.table.Size().Height-2*header, th.Size(sizePopupMaxList)))
	w := max(list.MinSize().Width+th.Size(sizeCellPadding)*2, th.Size(sizeFilterWidth))
	scroll.SetMinSize(fyne.NewSize(w, h))
	c := fyne.CurrentApp().Driver().CanvasForObject(tv.table)
	if c == nil {
		return
	}
	tv.colPopup = widget.NewPopUp(scroll, c)
	pos := fyne.NewPos(tv.table.Size().Width-tv.colPopup.MinSize().Width, header)
	tv.colPopup.ShowAtRelativePosition(pos, tv.table)
}

// columnsOpen reports whether the column list is showing.
func (tv *tableView) columnsOpen() bool { return tv.colPopup != nil && tv.colPopup.Visible() }

// syncColumnChecks makes the list's checks show the snapshot's columns. A list opened for another
// view closes.
func (tv *tableView) syncColumnChecks() {
	if len(tv.colChecks) == 0 || tv.snap == nil {
		return
	}
	if tv.snap.View != tv.colView {
		if tv.colPopup != nil {
			tv.colPopup.Hide()
		}
		tv.colChecks, tv.colIDs = nil, nil
		return
	}
	tv.colSync = true
	defer func() { tv.colSync = false }()
	for i, ch := range tv.colChecks {
		shown := false
		for _, c := range tv.snap.Columns {
			if c.ID == tv.colIDs[i] {
				shown = !c.Hidden
				break
			}
		}
		if ch.Checked != shown {
			ch.SetChecked(shown)
		}
	}
}

// toggleColumn shows or hides a column: it sends the view's new list of hidden columns. The
// engine refuses to hide the last visible column; the check then goes back.
func (tv *tableView) toggleColumn(id string, show bool) {
	if tv.colSync || tv.snap == nil || tv.snap.View != tv.colView {
		return
	}
	hidden := []string{}
	for _, c := range tv.snap.Columns {
		if c.Hidden && c.ID != id {
			hidden = append(hidden, c.ID)
		}
	}
	if !show {
		hidden = append(hidden, id)
	}
	if len(hidden) == len(tv.snap.Columns) {
		tv.sh.toast("warning", "At least one column stays visible.")
		tv.syncColumnChecks()
		return
	}
	tv.sh.dispatch("ui.setColumns", columnsParams{View: tv.colView, Hidden: hidden}, func(_ any, err error) {
		tv.sh.showError(err)
		tv.syncColumnChecks()
	})
}
