package fyneui

import (
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// F1–F12 open the hotbar's clusters from the plain search fields too (A64): the view's search, the
// navigator's (⌘P's too) and the palette's filter (the palette closes first); other keys still
// type. The scene's bank: demo, kind-cogpit (connected), prod-eu-central.
func TestSearchFieldsHandFKeysToTheHotbar(t *testing.T) {
	sh, w, q, f, _ := paletteShell(t, ThemeDark)
	slots := func() []int {
		var out []int
		for _, c := range f.callsOf("hotbar.openCluster") {
			out = append(out, decodeParams[hotbarSlotParams](t, c).Slot)
		}
		return out
	}
	opened := func(want ...int) func() bool {
		return func() bool { return len(slots()) >= len(want) }
	}

	w.Canvas().Focus(sh.view.search)
	sh.view.search.TypedKey(&fyne.KeyEvent{Name: fyne.KeyF3})
	q.until(t, "F3 from the view's search", opened(3))
	test.Type(sh.view.search, "web")
	if sh.view.search.Text != "web" {
		t.Errorf("the field %q", sh.view.search.Text)
	}
	q.until(t, "prod connected", func() bool { return sh.ui.Context == "prod-eu-central" }) // the fake's hotbar connects it

	sh.side.head.openSearch("")
	sh.side.head.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyF1})
	q.until(t, "F1 from the navigator's search", opened(3, 1))

	openPalette(t, sh, q)
	pressInPalette(sh, q, fyne.KeyF3)
	q.until(t, "F3 from the palette, closed first", func() bool { return len(slots()) >= 3 && !sh.palette.visible() })
	if got := slots(); !slices.Equal(got, []int{3, 1, 3}) {
		t.Errorf("hotbar.openCluster's slots %v, want 3, 1, 3", got)
	}

	sh.view.openNamespaces() // the namespace filter's search (F21), its list closed first
	if !sh.view.ns.visible() {
		t.Fatal("the namespace filter's list did not open")
	}
	sh.view.ns.search.TypedKey(&fyne.KeyEvent{Name: fyne.KeyF2})
	q.until(t, "F2 from the namespace filter's search, closed first", func() bool { return len(slots()) >= 4 && !sh.view.ns.visible() })
	if got := slots(); got[3] != 2 {
		t.Errorf("hotbar.openCluster's slots %v, want 3, 1, 3, 2", got)
	}
}
