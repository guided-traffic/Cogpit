package fyneui

import (
	"image/color"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// dragTile drags the shown bank's tile at index from to the window position to in two moves and,
// unless hold, drops it there, as the GLFW driver sends a drag (dragHeader in
// columnorder_test.go): positions from the tile, the pointer's absolute position, the delta of
// each move.
func dragTile(t *testing.T, h *hotbar, from int, to fyne.Position, hold bool) *hotbarTile {
	t.Helper()
	tl := tilesOf(h)[from]
	drv := fyne.CurrentApp().Driver()
	tp := drv.AbsolutePositionForObject(tl)
	start := tp.AddXY(tl.Size().Width/2, tl.Size().Height/2)
	half := fyne.NewDelta((to.X-start.X)/2, (to.Y-start.Y)/2)
	for _, p := range []fyne.Position{start.Add(half), to} {
		tl.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{AbsolutePosition: p, Position: p.Subtract(tp)}, Dragged: half})
	}
	if !hold {
		tl.DragEnd()
	}
	return tl
}

// upperHalf is a point in the upper half of the shown bank's tile i, in window coordinates: a drop
// there goes into the gap above it.
func upperHalf(h *hotbar, i int) fyne.Position {
	tl := tilesOf(h)[i]
	return fyne.CurrentApp().Driver().AbsolutePositionForObject(tl).AddXY(tl.Size().Width/2, tl.Size().Height/4)
}

func contextsOf(h *hotbar) []string {
	out := []string{}
	for _, t := range tilesOf(h) {
		out = append(out, t.tile.Context)
	}
	return out
}

// A tile dragged within the bar moves through hotbar.moveCluster: into the gap above another tile,
// or last when dropped on the "+"; while dragging it is drawn out of its place, the line marks the
// gap and the other tiles neither light up nor show a tooltip; the drag opens nothing.
func TestHotbarTileDragMovesIt(t *testing.T) {
	sh, w, q, f, _ := hotbarScene(t, ThemeDark)
	settle(w)
	h := sh.hot
	before := contextsOf(h)

	// kind-cogpit (first) into the gap above legacy-onprem (fourth): third place.
	tl := dragTile(t, h, 0, upperHalf(h, 3), true)
	r := test.WidgetRenderer(tl).(*hotbarTileRenderer)
	if !h.drag.active || h.drag.gap != 3 || !h.drag.line.Visible() || !tl.lifted() || r.bg.FillColor != color.Transparent || r.bg.StrokeWidth == 0 {
		t.Fatalf("while dragging: active %v gap %d line %v lifted %v fill %v", h.drag.active, h.drag.gap, h.drag.line.Visible(), tl.lifted(), r.bg.FillColor)
	}
	tiles := tilesOf(h)
	lineY := h.drag.line.Position().Y + h.drag.line.Size().Height/2
	if bottom := tiles[2].Position().Y + tiles[2].Size().Height; lineY < bottom || lineY > tiles[3].Position().Y {
		t.Errorf("the line at %v, want between the third tile's end %v and the fourth's top %v", lineY, bottom, tiles[3].Position().Y)
	}
	other := tiles[4]
	other.MouseIn(&desktop.MouseEvent{PointEvent: fyne.PointEvent{AbsolutePosition: upperHalf(h, 4)}})
	if other.hovered || sh.tip.owner == other {
		t.Error("a tile lit up or asked for its tooltip under a dragged one")
	}
	tl.DragEnd()
	q.until(t, "the move", func() bool { return len(f.callsOf("hotbar.moveCluster")) == 1 })
	if p := decodeParams[hotbarMoveParams](t, f.callsOf("hotbar.moveCluster")[0]); p.Context != "kind-cogpit" || p.Position != 3 || p.Bank != 1 {
		t.Fatalf("hotbar.moveCluster %+v", p)
	}
	want := []string{before[1], before[2], before[0], before[3], before[4], before[5]}
	q.until(t, "the new order", func() bool { return slices.Equal(contextsOf(h), want) })
	if h.drag.active || h.drag.line.Visible() || tl.lifted() {
		t.Errorf("after the drop: active %v line %v", h.drag.active, h.drag.line.Visible())
	}

	// demo (last) up into the gap above the first tile, then the first on the "+": last again.
	settle(w)
	dragTile(t, h, 5, upperHalf(h, 0), false)
	q.until(t, "demo first", func() bool { return contextsOf(h)[0] == "demo" })
	settle(w)
	add := fyne.CurrentApp().Driver().AbsolutePositionForObject(h.add).AddXY(h.add.Size().Width/2, h.add.Size().Height/2)
	dragTile(t, h, 0, add, false)
	q.until(t, "demo last", func() bool { return contextsOf(h)[5] == "demo" })
	if p := decodeParams[hotbarMoveParams](t, f.callsOf("hotbar.moveCluster")[2]); p.Position != 6 {
		t.Errorf("dropped on the +: %+v", p)
	}
	q.drain()
	if n := len(f.callsOf("hotbar.openCluster")); n != 0 {
		t.Errorf("a drag opened a cluster %d times", n)
	}
}

// A drop on the tile's own place, outside the bar (in the table, on the switcher) or after the shown
// bank changed moves nothing; a drag shorter than minTileDrag opens the cluster, as the click it
// was meant to be.
func TestHotbarTileDragCancels(t *testing.T) {
	sh, w, q, f, _ := hotbarScene(t, ThemeDark)
	settle(w)
	h := sh.hot
	drv := fyne.CurrentApp().Driver()

	// Its own place: the gap below it; no line while it would stay.
	tl := dragTile(t, h, 1, upperHalf(h, 2), true)
	if h.drag.gap != 2 || h.drag.line.Visible() || !tl.lifted() {
		t.Errorf("own place: gap %d line %v lifted %v", h.drag.gap, h.drag.line.Visible(), tl.lifted())
	}
	tl.DragEnd()

	// Outside the bar: right of it in the table, then on the bank switcher. R3's seam (D-52).
	bar := drv.AbsolutePositionForObject(h.root)
	tl = dragTile(t, h, 1, upperHalf(h, 4).AddXY(h.root.Size().Width+200, 0), true)
	if h.drag.gap != -1 || h.drag.line.Visible() {
		t.Errorf("outside the bar: gap %d line %v", h.drag.gap, h.drag.line.Visible())
	}
	tl.DragEnd()
	dragTile(t, h, 1, drv.AbsolutePositionForObject(h.number).AddXY(2, 2), false)
	dragTile(t, h, 1, fyne.NewPos(bar.X+2, bar.Y+h.root.Size().Height+40), false) // below the window's bar: the status bar

	// The shown bank changed while dragging.
	tl = dragTile(t, h, 0, upperHalf(h, 4), true)
	f.update(func(u *engine.UIState) { hb := u.Hotbar.Clone(); hb.Current = 2; u.Hotbar = &hb })
	q.until(t, "bank 2", func() bool { return h.number.text == "2" })
	if h.drag.line.Visible() {
		t.Error("the line stays on another bank")
	}
	tl.DragEnd()
	q.drain()
	if n := len(f.callsOf("hotbar.moveCluster")); n != 0 {
		t.Fatalf("%d moves sent", n)
	}

	// A twitch: the click that slipped opens the tile's cluster (prod-eu-central, bank 2's first).
	settle(w)
	tl = tilesOf(h)[0]
	tl.Dragged(&fyne.DragEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(20, 20)}, Dragged: fyne.NewDelta(3, 0)})
	if tl.lifted() {
		t.Error("lifted after three points")
	}
	tl.DragEnd()
	q.until(t, "the slipped click's connect", func() bool { return len(f.callsOf("hotbar.openCluster")) == 1 })
	if p := decodeParams[hotbarSlotParams](t, f.callsOf("hotbar.openCluster")[0]); p.Slot != 1 || p.Bank != 2 {
		t.Errorf("hotbar.openCluster %+v", p)
	}
	if n := len(f.callsOf("hotbar.moveCluster")); n != 0 {
		t.Errorf("a twitch moved a tile: %d", n)
	}
}

// Fyne's test driver finds the tile under the pointer as the drag's target (not the bar's scroll)
// and the drop moves it: test.Drag through the window's canvas.
func TestHotbarTileDragThroughTheCanvas(t *testing.T) {
	sh, w, q, f, _ := hotbarScene(t, ThemeDark)
	settle(w)
	h := sh.hot
	tl := tilesOf(h)[4] // dev-hfi
	start := fyne.CurrentApp().Driver().AbsolutePositionForObject(tl).AddXY(tl.Size().Width/2, tl.Size().Height/2)
	to := upperHalf(h, 1)
	test.Drag(w.Canvas(), start, 0, to.Y-start.Y)
	q.until(t, "the move", func() bool { return len(f.callsOf("hotbar.moveCluster")) == 1 })
	if p := decodeParams[hotbarMoveParams](t, f.callsOf("hotbar.moveCluster")[0]); p.Context != "dev-hfi" || p.Position != 2 {
		t.Errorf("hotbar.moveCluster %+v", p)
	}
	q.until(t, "dev-hfi second", func() bool { return contextsOf(h)[1] == "dev-hfi" })
}

func TestMovedPosition(t *testing.T) {
	for _, c := range []struct {
		from, gap, n, want int
		ok                 bool
	}{
		{0, 3, 6, 3, true}, {0, 6, 6, 6, true}, {5, 0, 6, 1, true}, {2, 4, 6, 4, true}, {3, 1, 6, 2, true},
		{2, 2, 6, 0, false}, {2, 3, 6, 0, false}, {0, -1, 6, 0, false}, {-1, 2, 6, 0, false}, {0, 7, 6, 0, false},
		{0, 1, 1, 0, false}, {0, 0, 1, 0, false},
	} {
		if got, ok := movedPosition(c.from, c.gap, c.n); got != c.want || ok != c.ok {
			t.Errorf("movedPosition(%d, %d, %d) = %d, %v; want %d, %v", c.from, c.gap, c.n, got, ok, c.want, c.ok)
		}
	}
}

func TestRenderHotbarDragDark(t *testing.T) { renderHotbarDrag(t, ThemeDark, "hotbar_drag_dark.png") }
func TestRenderHotbarDragLight(t *testing.T) {
	renderHotbarDrag(t, ThemeLight, "hotbar_drag_light.png")
}

// renderHotbarDrag: prod-eu-central (second) dragged over dev-hfi (fifth): an outline where it
// was, the accent line in the gap above dev-hfi.
func renderHotbarDrag(t *testing.T, mode ThemeMode, name string) {
	sh, w, q, _, _ := hotbarScene(t, mode)
	q.until(t, "diagnostics", func() bool { return sh.diag.total > 0 })
	settle(w)
	tl := dragTile(t, sh.hot, 1, upperHalf(sh.hot, 4), true)
	if !sh.hot.drag.line.Visible() {
		t.Fatal("no drop line")
	}
	savePNG(t, w, name)
	tl.DragEnd()
}
