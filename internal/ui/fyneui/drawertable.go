package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/model"
)

// table renders a section's table. A table of cells (model.CellTable: a node's pods, an endpoint
// slice's endpoints) draws its cells as the lists do: a link in the link colour that goes to its
// object, a status in its colour, and a warning column as the warning icon with the reason as its
// tooltip, under no title. Other tables are plain text (detailTable).
func (d *drawer) table(ref model.ObjectRef, t *model.DetailTable) fyne.CanvasObject {
	if len(t.Cells) == 0 || len(t.Columns) == 0 {
		return detailTable(t)
	}
	colType := func(i int) model.ColumnType {
		if i < len(t.Types) {
			return t.Types[i]
		}
		return model.ColumnText
	}
	// The cells go into the grid at once: Container.Add lays the grid out on every call, which
	// for a node's hundreds of pods measures every cell hundreds of times.
	objs := make([]fyne.CanvasObject, 0, len(t.Columns)*(len(t.Cells)+1))
	for i, c := range t.Columns {
		if colType(i) == model.ColumnWarning {
			c = ""
		}
		l := newTextLine(c, colorMuted, theme.SizeNameText, fyne.TextStyle{})
		l.truncate = true
		objs = append(objs, l)
	}
	for r, row := range t.Cells {
		for i := range t.Columns {
			var cell model.Cell
			if i < len(row) {
				cell = row[i]
			}
			text := cell.Text
			if r < len(t.Rows) && i < len(t.Rows[r]) {
				text = t.Rows[r][i]
			}
			objs = append(objs, d.tableCell(ref, colType(i), cell, text))
		}
	}
	return box(&detailTableLayout{cols: len(t.Columns)}, objs...)
}

// tableCell draws one cell of a table of cells.
func (d *drawer) tableCell(ref model.ObjectRef, typ model.ColumnType, c model.Cell, text string) fyne.CanvasObject {
	switch {
	case typ == model.ColumnWarning && c.Status != model.StatusNone:
		return newTipIcon(d.sh, theme.WarningIcon(), statusColorName(c.Status), text)
	case typ == model.ColumnWarning:
		text = ""
	case c.Link != nil && text != "" && d.sh.view.serves(c.Link.Kind):
		link := *c.Link
		return newLinkText(text, func() { d.sh.followLink(link, ref.Context, false) })
	}
	fg := theme.ColorNameForeground
	if c.Status != model.StatusNone {
		fg = statusColorName(c.Status)
	}
	return tableCellText(text, fg)
}

// tipIcon is an icon in a theme colour with a tooltip: the warning icon of a drawer table's row,
// its reason as the tooltip. It is as high as a line of text, so that it does not stretch the row.
type tipIcon struct {
	widget.BaseWidget
	sh    *shell
	res   fyne.Resource
	color fyne.ThemeColorName
	tip   string
}

func newTipIcon(sh *shell, res fyne.Resource, color fyne.ThemeColorName, tip string) *tipIcon {
	i := &tipIcon{sh: sh, res: res, color: color, tip: tip}
	i.ExtendBaseWidget(i)
	return i
}

// MouseIn implements desktop.Hoverable.
func (i *tipIcon) MouseIn(ev *desktop.MouseEvent) { i.sh.tip.hover(i, i.tip, ev.AbsolutePosition) }

// MouseMoved implements desktop.Hoverable.
func (i *tipIcon) MouseMoved(ev *desktop.MouseEvent) { i.sh.tip.moved(i, ev.AbsolutePosition) }

// MouseOut implements desktop.Hoverable.
func (i *tipIcon) MouseOut() { i.sh.tip.hideFor(i) }

// CreateRenderer implements fyne.Widget.
func (i *tipIcon) CreateRenderer() fyne.WidgetRenderer {
	img := canvas.NewImageFromResource(themedIcon(i.res, i.color))
	img.FillMode = canvas.ImageFillContain
	return &tipIconRenderer{i: i, img: img}
}

type tipIconRenderer struct {
	i   *tipIcon
	img *canvas.Image
}

func (r *tipIconRenderer) side() float32 {
	return fyne.MeasureText("Mg", r.i.Theme().Size(theme.SizeNameText), fyne.TextStyle{}).Height
}

func (r *tipIconRenderer) Layout(size fyne.Size) {
	s := fyne.Min(r.side(), fyne.Min(size.Width, size.Height))
	r.img.Move(fyne.NewPos(0, (size.Height-s)/2))
	r.img.Resize(fyne.NewSquareSize(s))
}

func (r *tipIconRenderer) MinSize() fyne.Size { return fyne.NewSquareSize(r.side()) }

func (r *tipIconRenderer) Refresh() {
	r.img.Resource = themedIcon(r.i.res, r.i.color)
	r.img.Refresh()
}

func (r *tipIconRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.img} }
func (r *tipIconRenderer) Destroy()                     {}
