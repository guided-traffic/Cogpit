package fyneui

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The widths of the two side panels, the primary sidebar and the detail drawer, are the user's:
// each is set by dragging the panel's edge, stored in Fyne's preferences and restored at the next
// start. Content never moves them. A panel is a fixed number of points wide, laid out by
// panelLayout, which reports a constant minimum width whatever the panel holds, so that a long
// value in the drawer (a container's CrashLoopBackOff status, which changes every few seconds in
// the demo cluster) cannot widen it, shift the table's columns or raise the window's minimum size.
//
// The widths are a window-layout preference, not engine state (AGENTS.md rule 2): no command sets
// them, no agent reads them. An agent drives what the views show (view, filter, selection, which
// object the drawer shows); how many points wide a panel is has no effect on any of it.

const (
	prefSidebarWidth = "cogpit.ui.sidebarWidth" // Fyne preferences key, in points
	prefDrawerWidth  = "cogpit.ui.drawerWidth"  // Fyne preferences key, in points

	defaultSidebarWidth float32 = 216 // the theme's sizeSidebarWidth
	defaultDrawerWidth  float32 = 600

	minSidebarWidth float32 = 160
	maxSidebarWidth float32 = 480
	minDrawerWidth  float32 = 320
	maxDrawerWidth  float32 = 1000

	// minTableRoom is the width the table keeps beside the drawer, and the workspace beside the
	// sidebar: dragging an edge stops there.
	minTableRoom float32 = 480

	handleWidth float32 = 5 // the grab area of a panel's edge; the line in it is one point
)

// clampWidth keeps w in [lo, hi]; a value that is not a number or not above zero (a hand-edited
// preference file) becomes def first.
func clampWidth(w, lo, hi, def float32) float32 {
	if math.IsNaN(float64(w)) || w <= 0 {
		w = def
	}
	return min(max(w, lo), max(hi, lo))
}

// loadWidths reads the stored widths; a missing or invalid value is the default.
func (sh *shell) loadWidths() {
	sh.sidebarW, sh.drawerW = defaultSidebarWidth, defaultDrawerWidth
	if sh.app == nil {
		return
	}
	p := sh.app.Preferences()
	sh.sidebarW = clampWidth(float32(p.FloatWithFallback(prefSidebarWidth, float64(defaultSidebarWidth))), minSidebarWidth, maxSidebarWidth, defaultSidebarWidth)
	sh.drawerW = clampWidth(float32(p.FloatWithFallback(prefDrawerWidth, float64(defaultDrawerWidth))), minDrawerWidth, maxDrawerWidth, defaultDrawerWidth)
}

// saveWidths stores the widths; called when a drag ends, not on every step of it.
func (sh *shell) saveWidths() {
	if sh.app == nil {
		return
	}
	p := sh.app.Preferences()
	p.SetFloat(prefSidebarWidth, float64(sh.sidebarW))
	p.SetFloat(prefDrawerWidth, float64(sh.drawerW))
}

// panelLayout lays out three objects: the panel (fixed width), the handle on its edge and the rest
// (what is left). With first set the objects are panel, handle, rest (the sidebar, on the left);
// otherwise rest, handle, panel (the drawer, on the right). The panel's width is want() clamped
// to [lo, hi] and to what leaves the rest at least restMin.
type panelLayout struct {
	first   bool
	want    func() float32
	lo, hi  float32
	restMin float32

	total float32         // the width of the last layout, for the drag's limit
	got   float32         // the panel's width at the last layout
	onGot func(w float32) // the panel's width changed in a layout
}

func (l *panelLayout) parts(objs []fyne.CanvasObject) (panel, handle, rest fyne.CanvasObject) {
	if l.first {
		return objs[0], objs[1], objs[2]
	}
	return objs[2], objs[1], objs[0]
}

// width is the panel's width in a container total points wide.
func (l *panelLayout) width(total float32) float32 {
	return clampWidth(l.want(), l.lo, min(l.hi, total-handleWidth-l.restMin), l.lo)
}

// limit is the widest the panel may be now.
func (l *panelLayout) limit() float32 {
	if l.total <= 0 {
		return l.hi
	}
	return max(l.lo, min(l.hi, l.total-handleWidth-l.restMin))
}

// MinSize is the panel's least width, the handle and the rest's minimum: not the panel's content.
func (l *panelLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	_, _, rest := l.parts(objs)
	r := rest.MinSize()
	return fyne.NewSize(l.lo+handleWidth+r.Width, r.Height)
}

func (l *panelLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	w := l.width(size.Width)
	restW := max(size.Width-w-handleWidth, 0)
	panel, handle, rest := l.parts(objs)
	panelX, handleX, restX := float32(0), w, w+handleWidth
	if !l.first {
		panelX, handleX, restX = restW+handleWidth, restW, 0
	}
	panel.Move(fyne.NewPos(panelX, 0))
	panel.Resize(fyne.NewSize(w, size.Height))
	handle.Move(fyne.NewPos(handleX, 0))
	handle.Resize(fyne.NewSize(handleWidth, size.Height))
	rest.Move(fyne.NewPos(restX, 0))
	rest.Resize(fyne.NewSize(restW, size.Height))
	l.total = size.Width
	if w != l.got {
		l.got = w
		if l.onGot != nil {
			l.onGot(w)
		}
	}
}

// newPanel puts panel and rest side by side with a draggable edge between them.
func newPanel(l *panelLayout, panel, rest fyne.CanvasObject, h *resizeHandle) *fyne.Container {
	if l.first {
		return box(l, panel, h, rest)
	}
	return box(l, rest, h, panel)
}

// resizeHandle is the grab area on a panel's edge: a one-point line, highlighted under the pointer,
// and a horizontal resize cursor. A drag moves the edge by the pointer's distance; sign is +1 when
// the panel is on the left of the edge (dragging right widens it) and -1 when it is on the right.
type resizeHandle struct {
	widget.BaseWidget
	sign     float32
	current  func() float32          // the panel's width now
	set      func(w float32) float32 // applies a width and returns the one that took effect
	done     func()                  // a drag ended
	hovered  bool
	dragging bool
	start    float32
	moved    float32
}

var (
	_ desktop.Hoverable  = (*resizeHandle)(nil)
	_ desktop.Cursorable = (*resizeHandle)(nil)
	_ fyne.Draggable     = (*resizeHandle)(nil)
)

func newResizeHandle(sign float32, current func() float32, set func(float32) float32, done func()) *resizeHandle {
	h := &resizeHandle{sign: sign, current: current, set: set, done: done}
	h.ExtendBaseWidget(h)
	return h
}

// Cursor implements desktop.Cursorable.
func (h *resizeHandle) Cursor() desktop.Cursor { return desktop.HResizeCursor }

// MouseIn implements desktop.Hoverable.
func (h *resizeHandle) MouseIn(*desktop.MouseEvent) { h.hovered = true; h.Refresh() }

// MouseMoved implements desktop.Hoverable.
func (h *resizeHandle) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (h *resizeHandle) MouseOut() { h.hovered = false; h.Refresh() }

// Dragged implements fyne.Draggable. The distance is summed from the width at the drag's start,
// and what the limits did not allow is not kept, so that the edge follows the pointer back as
// soon as it returns.
func (h *resizeHandle) Dragged(e *fyne.DragEvent) {
	if !h.dragging {
		h.dragging, h.start, h.moved = true, h.current(), 0
	}
	h.moved += e.Dragged.DX
	got := h.set(h.start + h.sign*h.moved)
	h.moved = (got - h.start) * h.sign
}

// DragEnd implements fyne.Draggable.
func (h *resizeHandle) DragEnd() {
	if !h.dragging {
		return
	}
	h.dragging = false
	h.Refresh()
	if h.done != nil {
		h.done()
	}
}

// CreateRenderer implements fyne.Widget.
func (h *resizeHandle) CreateRenderer() fyne.WidgetRenderer {
	r := &handleRenderer{h: h, bg: canvas.NewRectangle(color.Transparent), line: canvas.NewRectangle(color.Transparent)}
	r.Refresh()
	return r
}

type handleRenderer struct {
	h    *resizeHandle
	bg   *canvas.Rectangle
	line *canvas.Rectangle
}

func (r *handleRenderer) Layout(size fyne.Size) {
	r.bg.Move(fyne.NewPos(0, 0))
	r.bg.Resize(size)
	r.line.Move(fyne.NewPos((size.Width-1)/2, 0))
	r.line.Resize(fyne.NewSize(1, size.Height))
}

func (r *handleRenderer) MinSize() fyne.Size           { return fyne.NewSize(handleWidth, 0) }
func (r *handleRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.line} }
func (r *handleRenderer) Destroy()                     {}

func (r *handleRenderer) Refresh() {
	th := r.h.Theme()
	r.line.FillColor = themeColor(th, theme.ColorNameSeparator)
	r.bg.FillColor = color.Transparent
	if r.h.hovered || r.h.dragging {
		r.bg.FillColor = withAlpha(themeColor(th, colorLink), 0x55)
	}
	r.Layout(r.h.Size())
	r.bg.Refresh()
	r.line.Refresh()
}

// lockedWidth is a layout for a panel's root: its minimum width is a constant, not the widest
// thing inside, and its objects get the size it is given. What does not fit is cut off by the
// scroll container inside, never passed up to the split.
type lockedWidth struct{ min float32 }

func (l lockedWidth) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var h float32
	for _, o := range objs {
		if o.Visible() {
			h = max(h, o.MinSize().Height)
		}
	}
	return fyne.NewSize(l.min, h)
}

func (lockedWidth) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

// setDrawerWidth applies the width of a drag step to the drawer and returns the one that took
// effect (within the bounds and the room the table keeps). It stores nothing: saveWidths does,
// when the drag ends.
func (sh *shell) setDrawerWidth(w float32) float32 {
	sh.drawerW = clampWidth(w, minDrawerWidth, sh.drawerLay.limit(), minDrawerWidth)
	sh.hsplit.Refresh()
	return sh.drawerW
}

// setSidebarWidth is setDrawerWidth for the primary sidebar.
func (sh *shell) setSidebarWidth(w float32) float32 {
	sh.sidebarW = clampWidth(w, minSidebarWidth, sh.sideLay.limit(), minSidebarWidth)
	sh.mainRow.Refresh()
	return sh.sidebarW
}
