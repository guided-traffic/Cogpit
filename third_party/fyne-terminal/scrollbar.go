package terminal

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// scrollBar shows where the view is among the rows there are, and moves it when its thumb is
// dragged or the bar is tapped above or below the thumb. It is thin, and wide under the mouse,
// like Fyne's scroll bars. Cogpit patch.
type scrollBar struct {
	widget.BaseWidget
	t       *Terminal
	hovered bool
	dragged float32 // rows dragged that have not moved the view yet
}

func newScrollBar(t *Terminal) *scrollBar {
	b := &scrollBar{t: t}
	b.ExtendBaseWidget(b)
	return b
}

// width is the bar's width: wide under the mouse.
func (b *scrollBar) width() float32 {
	if b.hovered {
		return theme.Size(theme.SizeNameScrollBar)
	}
	return theme.Size(theme.SizeNameScrollBarSmall)
}

// thumb returns the thumb's top and height, and false when there is nothing to scroll.
func (b *scrollBar) thumb() (top, height float32, ok bool) {
	t := b.t
	first, last := t.topRow(), t.screenStart()
	if last <= first {
		return 0, 0, false
	}
	track := b.Size().Height
	total := float32(last - first + int(t.config.Rows))
	height = fyne.Max(track*float32(t.config.Rows)/total, 2*theme.Size(theme.SizeNameScrollBar))
	height = fyne.Min(height, track)
	top = (track - height) * float32(t.viewTop-first) / float32(last-first)
	return top, height, true
}

func (b *scrollBar) CreateRenderer() fyne.WidgetRenderer {
	r := &scrollBarRenderer{bar: b, thumb: canvas.NewRectangle(theme.Color(theme.ColorNameScrollBar))}
	r.Refresh()
	return r
}

func (b *scrollBar) Dragged(ev *fyne.DragEvent) {
	_, height, ok := b.thumb()
	span := b.Size().Height - height
	if !ok || span <= 0 {
		return
	}
	t := b.t
	b.dragged += ev.Dragged.DY * float32(t.screenStart()-t.topRow()) / span
	rows := int(b.dragged)
	if rows == 0 {
		return
	}
	b.dragged -= float32(rows)
	t.scrollTo(t.viewTop + rows)
}

func (b *scrollBar) DragEnd() {
	b.dragged = 0
}

// Tapped moves the view a screen towards the tap.
func (b *scrollBar) Tapped(ev *fyne.PointEvent) {
	top, height, ok := b.thumb()
	if !ok {
		return
	}
	switch {
	case ev.Position.Y < top:
		b.t.scrollBy(int(b.t.config.Rows))
	case ev.Position.Y > top+height:
		b.t.scrollBy(-int(b.t.config.Rows))
	}
}

func (b *scrollBar) MouseIn(*desktop.MouseEvent) {
	b.hovered = true
	b.t.layoutBar()
}

func (b *scrollBar) MouseMoved(*desktop.MouseEvent) {}

func (b *scrollBar) MouseOut() {
	b.hovered = false
	b.t.layoutBar()
}

type scrollBarRenderer struct {
	bar   *scrollBar
	thumb *canvas.Rectangle
}

func (r *scrollBarRenderer) Layout(fyne.Size) {
	r.Refresh()
}

func (r *scrollBarRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 0)
}

func (r *scrollBarRenderer) Refresh() {
	top, height, ok := r.bar.thumb()
	r.thumb.Hidden = !ok
	r.thumb.FillColor = theme.Color(theme.ColorNameScrollBar)
	width := r.bar.Size().Width
	r.thumb.CornerRadius = width / 2
	r.thumb.Move(fyne.NewPos(0, top))
	r.thumb.Resize(fyne.NewSize(width, height))
	r.thumb.Refresh()
}

func (r *scrollBarRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.thumb}
}

func (r *scrollBarRenderer) Destroy() {}
