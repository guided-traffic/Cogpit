package fyneui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/model"
)

// reachIcon plays the pointer's way from a chip to its copy icon as the driver sends it: into the
// chip, then onto the icon's middle. The middle must lie inside the chip's hover area (its
// content and its gutter), so that the icon was never hidden on the way; the driver then sends
// MouseOut to the chip and MouseIn to the icon, which has to stay shown and copy the chip.
func reachIcon(t *testing.T, sh *shell, hw *hoverWrap, want string) {
	t.Helper()
	d := sh.drawer
	name := findAll[*pill](hw.content)[0].text
	drv := fyne.CurrentApp().Driver()
	hw.MouseIn(mouse())
	if !d.copyBtn.Visible() {
		t.Fatalf("no icon over the chip %q", name)
	}
	btn := drv.AbsolutePositionForObject(d.copyBtn).Add(fyne.NewPos(d.copyBtn.Size().Width/2, d.copyBtn.Size().Height/2))
	chip := drv.AbsolutePositionForObject(hw)
	if btn.X < chip.X || btn.X > chip.X+hw.Size().Width || btn.Y < chip.Y || btn.Y > chip.Y+hw.Size().Height {
		t.Fatalf("the icon of %q is centred at %v, outside the chip's hover area %v + %v: the pointer leaves it on the way", name, btn, chip, hw.Size())
	}
	hw.MouseOut()
	d.copyBtn.MouseIn(mouse())
	if !d.copyBtn.Visible() {
		t.Fatalf("the icon of %q vanished under the pointer", name)
	}
	sh.app.Clipboard().SetContent("")
	test.Tap(d.copyBtn)
	if got := sh.app.Clipboard().Content(); got != want {
		t.Errorf("the icon of %q copied %q, want %q", name, got, want)
	}
	d.copyBtn.MouseOut()
	if d.copyBtn.Visible() {
		t.Errorf("the icon of %q stays after the pointer left it", name)
	}
}

// TestCopyIconReachableOnLabelChips: the owner could reach the icon of a Pod IP but not the icon
// of a label chip. A chip in a row was only as wide as itself (the last one of a row had no
// gutter at all), so the icon, which sits after the chip, lay outside its hover area. Now every
// chip, first, middle or last, owns the room for its icon.
func TestCopyIconReachableOnLabelChips(t *testing.T) {
	sh, w, _, _ := podDrawerScene(t, ThemeDark)
	test.Tap(drawerLinks(sh)["2 Labels"])
	settle(w)
	n := 0
	for _, hw := range findAll[*hoverWrap](sh.drawer.details) {
		p := findAll[*pill](hw.content)
		if len(p) != 1 || !strings.Contains(p[0].text, "=") {
			continue
		}
		n++
		reachIcon(t, sh, hw, yamlEntry(p[0].text))
	}
	if n != 2 {
		t.Fatalf("%d label chips, want 2", n)
	}
	// Every other chip and pill of the drawer keeps the icon under the pointer too.
	m := 0
	for _, hw := range findAll[*hoverWrap](sh.drawer.details) {
		if p := findAll[*pill](hw.content); len(p) == 1 {
			m++
			hw.MouseIn(mouse())
			hw.MouseOut()
			sh.drawer.copyBtn.MouseIn(mouse())
			if !sh.drawer.copyBtn.Visible() {
				t.Errorf("the icon of %q vanished under the pointer", p[0].text)
			}
			sh.drawer.copyBtn.MouseOut()
		}
	}
	if m < 3 {
		t.Fatalf("only %d chips checked", m)
	}
}

// TestCopyIconReachableOnWrappedChips: a list too long for one line takes a line per chip; the
// first, a middle one, the last of the list and one that wraps itself keep their icon in reach.
func TestCopyIconReachableOnWrappedChips(t *testing.T) {
	sh, w, _, ref := podDrawerScene(t, ThemeDark)
	d := sh.drawer
	var chips []model.Chip
	for i := range 8 {
		chips = append(chips, model.Chip{Text: fmt.Sprintf("example.com/label-number-%d=value-%d", i, i)})
	}
	chips = append(chips, model.Chip{Text: "long=" + strings.Repeat("x-", 150)})
	d.details.Add(d.chips(ref, chips, yamlEntry))
	d.details.Refresh()
	settle(w)
	var got []*hoverWrap
	for _, hw := range findAll[*hoverWrap](d.details) {
		p := findAll[*pill](hw.content)
		if len(p) == 1 && (strings.HasPrefix(p[0].text, "example.com/") || strings.HasPrefix(p[0].text, "long=")) {
			got = append(got, hw)
		}
	}
	if len(got) != len(chips) {
		t.Fatalf("%d chips in the drawer, want %d", len(got), len(chips))
	}
	for i, hw := range got {
		if i == 0 || i == 4 || i >= len(got)-2 {
			reachIcon(t, sh, hw, yamlEntry(findAll[*pill](hw.content)[0].text))
		}
	}
}

// TestCopyIconTurnsIntoACheckmark: after a copy the icon is a checkmark in the OK colour for a
// moment, then the copy icon again while the pointer is on the value; a pointer that left hides it.
func TestCopyIconTurnsIntoACheckmark(t *testing.T) {
	sh, _, q, ref := podDrawerScene(t, ThemeDark)
	d := sh.drawer
	sh.copyConfirm = 60 * time.Millisecond
	name := selLineWith(t, d, ref.Name)
	name.MouseIn(mouse())
	if d.copyBtn.done {
		t.Fatal("a checkmark before any copy")
	}
	test.Tap(d.copyBtn)
	if !d.copyBtn.done {
		t.Fatal("no checkmark after the copy")
	}
	q.until(t, "the copy icon to return", func() bool { return !d.copyBtn.done })
	if !d.copyBtn.Visible() {
		t.Error("the icon went with the checkmark although the pointer is still on the value")
	}
	// The pointer leaves during the checkmark: it hides as usual, and the state resets.
	test.Tap(d.copyBtn)
	name.MouseOut()
	if d.copyBtn.Visible() {
		t.Error("the checkmark stays after the pointer left")
	}
	q.until(t, "the state to reset", func() bool { return !d.copyBtn.done })
	if d.copyBtn.Visible() {
		t.Error("the icon came back without the pointer")
	}
}

// TestCopyCheckmarkForASecretValue: a revealed Secret value confirms the same way.
func TestCopyCheckmarkForASecretValue(t *testing.T) {
	sh, _, q, _ := podDrawerScene(t, ThemeDark)
	d := sh.drawer
	sh.copyConfirm = 40 * time.Millisecond
	d.copyOwner, d.copyHook = d.details, d.copyFor("hunter2", true)
	d.copyBtn.Show()
	test.Tap(d.copyBtn)
	if !d.copyBtn.done {
		t.Fatal("no checkmark after copying a Secret value")
	}
	q.until(t, "the copy icon to return", func() bool { return !d.copyBtn.done })
	sh.clearSecretClipboard()
}

func renderCopyChip(t *testing.T, mode ThemeMode, confirm bool, name string) {
	sh, w, _, _ := podDrawerScene(t, mode)
	test.Tap(drawerLinks(sh)["2 Labels"])
	settle(w)
	for _, hw := range findAll[*hoverWrap](sh.drawer.details) {
		if p := findAll[*pill](hw.content); len(p) == 1 && strings.Contains(p[0].text, "=") {
			hw.MouseIn(mouse())
			break
		}
	}
	if confirm {
		test.Tap(sh.drawer.copyBtn)
	}
	savePNG(t, w, name)
}

func TestRenderCopyIconOnLabelChip(t *testing.T) {
	renderCopyChip(t, ThemeDark, false, "chip_copy_hover_dark.png")
	renderCopyChip(t, ThemeDark, true, "chip_copy_done_dark.png")
	renderCopyChip(t, ThemeLight, true, "chip_copy_done_light.png")
}
