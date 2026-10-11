package fyneui

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

// itemCount is the header's count of a view's rows: "18 items" as captured (2026-10-11); with a
// filter that hides some, "18 of 10,000 items" (Cogpit's).
func itemCount(shown, total int) string {
	noun := " items"
	if total == 1 {
		noun = " item"
	}
	if shown != total {
		return formatCount(int64(shown)) + " of " + formatCount(int64(total)) + noun
	}
	return formatCount(int64(shown)) + noun
}

// csvExport is what table.exportCsv returns.
type csvExport struct {
	Filename string `json:"filename"`
	Rows     int    `json:"rows"`
	CSV      string `json:"csv"`
}

// exportCSV is the download button beside the item count: the engine writes the open view as CSV
// (table.exportCsv, so agents export the same text), and a file save dialog stores it.
func (tv *tableView) exportCSV() {
	tv.sh.dispatch("table.exportCsv", struct{}{}, func(res any, err error) {
		if err != nil {
			tv.sh.showError(err)
			return
		}
		var out csvExport
		if derr := decode(res, &out); derr != nil {
			tv.sh.toast("error", "Unexpected table.exportCsv result: "+derr.Error())
			return
		}
		tv.sh.saveFile(out.Filename, out.Rows, []byte(out.CSV))
	})
}

// saveWithDialog asks where to save data, suggesting name, and writes it there; it is the shell's
// saveFile. start, if set, is the folder the dialog opens in.
func (sh *shell) saveWithDialog(name string, rows int, data []byte, start fyne.ListableURI) *dialog.FileDialog {
	d := dialog.NewFileSave(func(w fyne.URIWriteCloser, err error) {
		if err == nil && w == nil {
			return // cancelled
		}
		if err == nil {
			err = writeAndClose(w, data)
		}
		if err != nil {
			sh.toast("error", "Could not save the table: "+err.Error())
			return
		}
		sh.toast("info", fmt.Sprintf("Saved %s to %s", itemCount(rows, rows), w.URI().Name()))
	}, sh.win)
	d.SetFileName(name)
	d.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
	if start != nil {
		d.SetLocation(start)
	}
	d.Show()
	return d
}

// writeAndClose writes data and closes w; a failing close (the file system did not take the
// last bytes) is an error too.
func writeAndClose(w fyne.URIWriteCloser, data []byte) error {
	_, err := w.Write(data)
	return errors.Join(err, w.Close())
}

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
