package terminal

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	widget2 "github.com/fyne-io/terminal/internal/widget"
)

const cursorWidth = 2

type render struct {
	term *Terminal
}

func (r *render) Layout(s fyne.Size) {
	r.term.view.Resize(s) // Cogpit patch: the view draws the rows, see scrollback.go
	r.term.layoutBar()
}

func (r *render) MinSize() fyne.Size {
	return fyne.NewSize(0, 0) // don't get propped open by the text cells
}

func (r *render) Refresh() {
	fyne.Do(func() { // TODO fix root cause in refresh on wrong thread
		r.term.refreshView() // Cogpit patch: the view, on the main goroutine like the cursor
		r.term.layoutBar()
		r.moveCursor()
		r.term.refreshCursor()
	})
}

func (r *render) BackgroundColor() color.Color {
	return color.Transparent
}

func (r *render) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.term.view, r.term.cursor, r.term.bar}
}

func (r *render) Destroy() {
	r.term.view.StopBlink() // Cogpit patch: see StopBlink
}

func (r *render) moveCursor() {
	cell := r.term.guessCellSize()
	row := r.term.cursorRow + r.term.screenStart() - r.term.viewTop // Cogpit patch: the view's row
	r.term.cursor.Move(fyne.NewPos(cell.Width*float32(r.term.cursorCol), cell.Height*float32(row)))
}

func (t *Terminal) refreshCursor() {
	// Cogpit patch: hidden while the view shows rows above it.
	t.cursor.Hidden = !t.focused || t.cursorHidden || t.cursorRow+t.screenStart()-t.viewTop >= int(t.config.Rows)
	if t.bell {
		t.cursor.FillColor = theme.Color(theme.ColorNameError)
	} else {
		t.cursor.FillColor = theme.Color(theme.ColorNamePrimary)
	}
	t.cursor.Resize(fyne.NewSize(cursorWidth, t.guessCellSize().Height))
	t.cursor.Refresh()
}

// layoutBar puts the scroll bar at the right edge, shown while there are rows above the screen.
// Cogpit patch.
func (t *Terminal) layoutBar() {
	if t.bar == nil {
		return
	}
	if _, _, ok := t.bar.thumb(); !ok && !t.bar.hovered {
		t.bar.Hide()
		return
	}
	size := t.Size()
	width := t.bar.width()
	t.bar.Resize(fyne.NewSize(width, size.Height))
	t.bar.Move(fyne.NewPos(size.Width-width, 0))
	t.bar.Show()
	t.bar.Refresh()
}

// CreateRenderer requests a new renderer for this terminal (just a wrapper around the TextGrid)
func (t *Terminal) CreateRenderer() fyne.WidgetRenderer {
	t.ExtendBaseWidget(t)

	// Retain any existing content when a new renderer is created.
	if t.content == nil {
		t.content = widget2.NewTermGrid()
		t.setupShortcuts()
	}
	if t.view == nil { // Cogpit patch
		t.view = widget2.NewTermGrid()
		t.bar = newScrollBar(t)
		t.bar.Hide()
	}

	if t.cursor == nil {
		t.cursor = canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
		t.cursor.Hidden = true
		t.cursor.Resize(fyne.NewSize(cursorWidth, t.guessCellSize().Height))
	}

	r := &render{term: t}
	t.cursorMoved = r.moveCursor
	return r
}
