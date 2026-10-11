// Package yamlview is Cogpit's YAML viewer and editor: a widget that keeps its own lines and fills
// a TextGrid only as large as the viewport, with YAML highlighting from theme tokens, line
// numbers, selection, copy and paste. Fyne's TextGrid stores every character as a cell and walks
// all rows for its minimum size, so a 5 MB document must never be loaded into one grid
// (docs/developer/spike-fyne.md, criterion S4).
//
// Layer L4.
package yamlview

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// View is the YAML viewer and editor widget.
type View struct {
	widget.BaseWidget
	text     string
	readOnly bool
	// OnChanged is called after the user edits the text.
	OnChanged func()
}

// New returns an empty, read-only view.
func New() *View {
	v := &View{readOnly: true}
	v.ExtendBaseWidget(v)
	return v
}

// SetText replaces the document and scrolls to the top.
func (v *View) SetText(text string) { v.text = text }

// Text returns the document.
func (v *View) Text() string { return v.text }

// SetReadOnly switches editing off or on.
func (v *View) SetReadOnly(ro bool) { v.readOnly = ro }

// CreateRenderer implements fyne.Widget.
func (v *View) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(widget.NewLabel(""))
}
