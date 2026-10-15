package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// Reordering the hotbar's tiles by dragging them (R2, decisions.md; ui-parity › Hotbar). The move
// is hotbar.moveCluster, the command the tile's Move Up and Move Down send, so agents see the new
// order in UIState.Hotbar as the bar does (UI state is engine state).
//
// Cogpit's look and rules (the lead's decisions for the owner to confirm, 2026-10-15; Lens's is not
// captured), after the table headers' column drag (columnorder.go): once a tile is dragged further
// than minTileDrag it is shown lifted out of its place (an outline where it was) and an accent line
// marks the gap between two tiles it drops into; dropping it there moves it. A bank's tiles are
// packed without empty cells (engine/hotbar.go), so a drop below the last tile, on the "+" or the
// empty bar under it, puts the tile last: there is no empty cell to drop it into. A drop on its own
// place, outside the bar's tiles (the switcher, the navigator, outside the window) or after the
// shown bank changed does nothing. A drag shorter than minTileDrag is the click that slipped: it
// opens the cluster, as the click would have (Fyne taps only what was not dragged, and its
// driver counts two points as a drag).

// minTileDrag is how far a tile must move before it is dragged: less is a click that slipped.
const minTileDrag = minColumnDrag

// tileDrag is a tile being dragged to another place in its bank, and the line that shows where it
// would land.
type tileDrag struct {
	h       *hotbar
	active  bool
	context string        // the dragged tile's context
	bank    int           // the bank it was dragged in
	moved   float32       // how far the pointer moved
	pointer fyne.Position // where the pointer is, in window coordinates
	gap     int           // the gap it drops into: before tile gap (from 0), len(tiles) after the last; -1 outside the bar

	layer *fyne.Container   // over the tiles, without a layout: the line
	line  *canvas.Rectangle // the accent line in the gap
}

func newTileDrag(h *hotbar) *tileDrag {
	d := &tileDrag{h: h, gap: -1}
	d.line = canvas.NewRectangle(color.Transparent)
	d.line.Hide()
	d.layer = container.NewWithoutLayout(d.line)
	return d
}

var _ fyne.Draggable = (*hotbarTile)(nil)

// Dragged implements fyne.Draggable: a tile dragged within the bar moves to the gap it is dropped
// into.
func (t *hotbarTile) Dragged(ev *fyne.DragEvent) { t.h.drag.dragged(t, ev) }

// DragEnd implements fyne.Draggable.
func (t *hotbarTile) DragEnd() { t.h.drag.ended(t) }

// lifted reports whether the tile is the one being dragged, shown out of its place.
func (t *hotbarTile) lifted() bool {
	d := t.h.drag
	return d != nil && d.active && d.moved >= minTileDrag && d.context == t.tile.Context &&
		t.h.view != nil && t.h.view.Current == d.bank
}

// dragged follows a tile's drag: which tile, where it would drop.
func (d *tileDrag) dragged(t *hotbarTile, ev *fyne.DragEvent) {
	h := d.h
	if !d.active {
		if h.view == nil {
			return
		}
		d.active, d.context, d.bank, d.moved, d.gap = true, t.tile.Context, h.view.Current, 0, -1
	}
	d.pointer = ev.AbsolutePosition
	if d.pointer.IsZero() { // Fyne's test driver: test.Drag sends where the press was and how far it moved
		d.pointer = fyne.CurrentApp().Driver().AbsolutePositionForObject(t).Add(ev.Position).Add(ev.Dragged)
	}
	was := d.moved >= minTileDrag
	d.moved += max(ev.Dragged.DX, -ev.Dragged.DX) + max(ev.Dragged.DY, -ev.Dragged.DY)
	if d.moved < minTileDrag {
		return
	}
	if !was { // lifted now: no tooltip, no hover, the tile drawn out of its place
		h.sh.tip.hide()
		t.hovered = false
		d.refreshTile()
	}
	d.gap = d.target()
	d.show()
}

// follow places the line again when the bar changed under a drag (an agent moved a tile or showed
// another bank): the gap is read again from the tiles as they are now.
func (d *tileDrag) follow() {
	if d.active && d.moved >= minTileDrag {
		d.gap = d.target()
		d.show()
	}
}

// ended drops the dragged tile into its gap through hotbar.moveCluster; the hotbar that follows
// shows it there.
func (d *tileDrag) ended(t *hotbarTile) {
	if !d.active {
		return
	}
	h := d.h
	ctx, bank, gap, moved := d.context, d.bank, d.gap, d.moved
	d.reset()
	b, ok := h.bank()
	if !ok || h.view.Current != bank || b.Position(ctx) < 1 {
		return // the shown bank changed while dragging
	}
	from := b.Position(ctx) - 1
	if moved < minTileDrag { // the click that slipped
		h.sh.tip.hideFor(t)
		h.open(from + 1)
		return
	}
	if gap < 0 {
		// Dropped outside the bar's tiles: nothing moves. R3, a tile dropped outside the window
		// opening another window with its cluster, would start here; it waits for the owner's
		// decision on several windows (D-52, decisions.md).
		return
	}
	to, ok := movedPosition(from, gap, len(b.Clusters))
	if !ok {
		return
	}
	h.sh.dispatch("hotbar.moveCluster", hotbarMoveParams{Context: ctx, Position: to, Bank: bank}, nil)
}

func (d *tileDrag) reset() {
	lifted := d.active && d.moved >= minTileDrag
	d.active, d.gap, d.moved = false, -1, 0
	d.line.Hide()
	if lifted {
		d.refreshTile()
	}
	d.context = ""
}

// refreshTile redraws the dragged tile: lifted while dragging, in place after.
func (d *tileDrag) refreshTile() {
	for _, t := range d.h.tileWidgets() {
		if t.tile.Context == d.context {
			t.Refresh()
		}
	}
}

// target returns the gap the pointer is over: before the tile whose upper half it is in, after the
// last tile below them all (the "+" and the empty bar under it), -1 outside the tiles' area.
func (d *tileDrag) target() int {
	h := d.h
	drv := fyne.CurrentApp().Driver()
	area, size := drv.AbsolutePositionForObject(h.scroll), h.scroll.Size()
	p := d.pointer
	if p.X < area.X || p.Y < area.Y || p.X >= area.X+size.Width || p.Y >= area.Y+size.Height {
		return -1
	}
	y := p.Y - drv.AbsolutePositionForObject(h.tiles).Y
	tiles := h.tileWidgets()
	for i, t := range tiles {
		if y < t.Position().Y+t.Size().Height/2 {
			return i
		}
	}
	return len(tiles)
}

// show draws the line in the drop gap; none outside the bar, on the tile's own place, or after the
// shown bank changed.
func (d *tileDrag) show() {
	h := d.h
	b, ok := h.bank()
	tiles := h.tileWidgets()
	if !ok || h.view.Current != d.bank || len(tiles) != len(b.Clusters) {
		d.line.Hide()
		return
	}
	if _, moves := movedPosition(b.Position(d.context)-1, d.gap, len(tiles)); !moves {
		d.line.Hide()
		return
	}
	th := h.sh.app.Settings().Theme()
	pad := th.Size(theme.SizeNamePadding)
	thick := th.Size(theme.SizeNameInputBorder) * 2
	var y float32
	if d.gap < len(tiles) {
		y = max(tiles[d.gap].Position().Y-pad/2, thick/2)
	} else {
		last := tiles[len(tiles)-1]
		y = last.Position().Y + last.Size().Height + pad/2
	}
	w := th.Size(sizeHotbarTile) + pad
	d.line.FillColor = themeColor(th, colorAccent)
	d.line.CornerRadius = thick / 2
	d.line.Move(fyne.NewPos((h.tiles.Size().Width-w)/2, y-thick/2))
	d.line.Resize(fyne.NewSize(w, thick))
	d.line.Show()
	d.line.Refresh()
}

// movedPosition returns the place (from 1) a tile at index from takes when dropped into gap of a
// bank of n tiles; ok is false when it would stay where it is.
func movedPosition(from, gap, n int) (int, bool) {
	if from < 0 || from >= n || gap < 0 || gap > n || gap == from || gap == from+1 {
		return 0, false
	}
	if gap < from {
		return gap + 1, true
	}
	return gap, true
}

// tileWidgets returns the shown bank's tiles, top to bottom.
func (h *hotbar) tileWidgets() []*hotbarTile {
	out := make([]*hotbarTile, 0, len(h.tiles.Objects))
	for _, o := range h.tiles.Objects {
		if t, ok := o.(*hotbarTile); ok {
			out = append(out, t)
		}
	}
	return out
}
