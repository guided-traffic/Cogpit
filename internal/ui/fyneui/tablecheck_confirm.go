package fyneui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// confirmListRows is how many objects the confirmation shows before their list scrolls: Delete of
// a selection names every object, up to engine.MaxSelection of them, and the buttons must stay in
// the window.
const confirmListRows = 8

// confirmObjects is the confirmation's Objects row: its name, with the count when there are
// several ("Objects (12)"), and every object on a line of its own, in a scrolling box no taller
// than confirmListRows lines when there are more.
func confirmObjects(objects []string, width float32) (name, value fyne.CanvasObject) {
	label := "Objects"
	if len(objects) > 1 {
		label = fmt.Sprintf("Objects (%d)", len(objects))
	}
	name = newTextLine(label, colorMuted, theme.SizeNameText, fyne.TextStyle{})
	list := newParagraph(strings.Join(objects, "\n"), theme.ColorNameForeground, width)
	if len(objects) <= confirmListRows {
		return name, list
	}
	th := fyne.CurrentApp().Settings().Theme()
	line := fyne.MeasureText("Mg", th.Size(theme.SizeNameText), fyne.TextStyle{}).Height + th.Size(theme.SizeNameLineSpacing)
	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(width, line*confirmListRows))
	return name, scroll
}
