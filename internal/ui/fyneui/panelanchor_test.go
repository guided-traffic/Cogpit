package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
)

// TestResizeAnchorsToThePointer: after the pointer overshoots a limit, the panel stays at the limit
// until the pointer is back over the edge, then follows it again; the stored width is the clamped one.
func TestResizeAnchorsToThePointer(t *testing.T) {
	cases := []struct {
		name  string
		panel func(*shell) *fyne.Container
		width func(*shell) float32
		pref  string
		lo    float32
		hi    func(*shell) float32
		sign  float32 // +1: dragging right widens
	}{
		{"drawer", func(sh *shell) *fyne.Container { return sh.hsplit }, func(sh *shell) float32 { return sh.drawerW }, prefDrawerWidth,
			minDrawerWidth, func(sh *shell) float32 { return sh.drawerLay.limit() }, -1},
		{"sidebar", func(sh *shell) *fyne.Container { return sh.mainRow }, func(sh *shell) float32 { return sh.sidebarW }, prefSidebarWidth,
			minSidebarWidth, func(sh *shell) float32 { return sh.sideLay.limit() }, 1},
	}
	for _, c := range cases {
		for _, atMax := range []bool{true, false} {
			name := c.name + "/min"
			if atMax {
				name = c.name + "/max"
			}
			t.Run(name, func(t *testing.T) {
				sh, w, _, _ := openLongDrawer(t)
				h := findAll[*resizeHandle](c.panel(sh))[0]
				drag := func(dx float32) { h.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(dx, 0)}) }
				limit := c.lo
				if atMax {
					limit = c.hi(sh)
				}
				// outward is the pointer direction that pushes toward the limit under test.
				outward := float32(1)
				if (limit > c.width(sh)) != (c.sign > 0) {
					outward = -1
				}
				drag((limit - c.width(sh)) * c.sign) // exactly to the limit
				if got := c.width(sh); got != limit {
					t.Fatalf("at the limit: %v, want %v", got, limit)
				}
				drag(outward * 300) // 300 beyond
				drag(-outward * 200)
				if got := c.width(sh); got != limit {
					t.Fatalf("pointer still 100 beyond the edge: %v, want the limit %v", got, limit)
				}
				drag(-outward * 150) // 50 inside
				want := limit - 50
				if !atMax {
					want = limit + 50
				}
				if got := c.width(sh); got != want {
					t.Fatalf("pointer back over the edge: %v, want %v", got, want)
				}
				h.DragEnd()
				settle(w)
				if got := float32(sh.app.Preferences().Float(c.pref)); got != want {
					t.Errorf("stored %v, want %v", got, want)
				}
			})
		}
	}
}
