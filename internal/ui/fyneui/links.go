package fyneui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/model"
)

// followLink goes where a link of a cell or a drawer value points, through the engine's ui.*
// commands (architecture §4.4):
//
//   - a Namespace link in a list's Namespace column narrows the view's namespace filter to that
//     namespace (ui.setNamespace; owner, 2026-10-11);
//   - a link without a name opens its kind's list view (ui.openView);
//   - any other link shows the object (ui.navigate); an empty context is the row's.
//
// A link to a kind the cluster does not serve fails in the engine and shows as an error toast.
func (sh *shell) followLink(link model.ObjectRef, rowContext string, namespaceColumn bool) {
	switch {
	case namespaceColumn && link.Kind == kindNamespaces && link.Name != "":
		sh.dispatch("ui.setNamespace", namespacesParams{Namespaces: []string{link.Name}}, nil)
	case link.Name == "":
		sh.dispatch("ui.openView", openViewParams{View: link.Kind}, nil)
	default:
		ref := link
		if ref.Context == "" {
			ref.Context = rowContext
		}
		sh.dispatch("ui.navigate", refParams{Ref: ref}, nil)
	}
}

// serves reports whether a link to kind leads somewhere: the connected cluster serves the kind and
// Cogpit has a view for it (kinds.list). Until the kinds are loaded every link counts. A cell or a
// drawer value whose link does not lead anywhere shows as plain text; a link that fails anyway
// (the kinds changed meanwhile) ends in an error toast. The set is rebuilt only when the kinds
// change, so asking per table cell is a map lookup.
func (tv *tableView) serves(kind string) bool {
	kinds := tv.sh.kinds
	if len(kinds) == 0 {
		return true
	}
	if tv.kindSet == nil || tv.kindSetSrc != &kinds[0] || tv.kindSetLen != len(kinds) {
		tv.kindSet = make(map[string]bool, len(kinds))
		for _, k := range kinds {
			tv.kindSet[k.Kind] = true
		}
		tv.kindSetSrc, tv.kindSetLen = &kinds[0], len(kinds)
	}
	return tv.kindSet[kind]
}

// linkText is one line of text in the link colour that runs onTap when clicked: a drawer value
// that names an object, or a folded list's count. The pointer turns into a hand over it and the
// text is underlined while hovered.
type linkText struct {
	widget.BaseWidget
	text    string
	size    fyne.ThemeSizeName
	onTap   func()
	hovered bool
}

func newLinkText(text string, onTap func()) *linkText {
	l := &linkText{text: text, size: theme.SizeNameText, onTap: onTap}
	l.ExtendBaseWidget(l)
	return l
}

// Tapped implements fyne.Tappable.
func (l *linkText) Tapped(*fyne.PointEvent) {
	if l.onTap != nil {
		l.onTap()
	}
}

// Cursor implements desktop.Cursorable.
func (l *linkText) Cursor() desktop.Cursor { return desktop.PointerCursor }

// MouseIn implements desktop.Hoverable.
func (l *linkText) MouseIn(*desktop.MouseEvent) { l.hovered = true; l.Refresh() }

// MouseMoved implements desktop.Hoverable.
func (l *linkText) MouseMoved(*desktop.MouseEvent) {}

// MouseOut implements desktop.Hoverable.
func (l *linkText) MouseOut() { l.hovered = false; l.Refresh() }

// CreateRenderer implements fyne.Widget.
func (l *linkText) CreateRenderer() fyne.WidgetRenderer {
	r := &linkTextRenderer{l: l, text: canvas.NewText("", color.Transparent), line: canvas.NewRectangle(color.Transparent)}
	r.Refresh()
	return r
}

type linkTextRenderer struct {
	l    *linkText
	text *canvas.Text
	line *canvas.Rectangle
}

func (r *linkTextRenderer) Layout(size fyne.Size) {
	shown := ellipsize(r.l.text, size.Width, r.text.TextSize, r.text.TextStyle)
	if shown != r.text.Text {
		r.text.Text = shown
		r.text.Refresh()
	}
	ms := r.text.MinSize()
	y := (size.Height - ms.Height) / 2
	r.text.Move(fyne.NewPos(0, y))
	r.text.Resize(ms)
	r.line.Move(fyne.NewPos(0, y+ms.Height-2))
	r.line.Resize(fyne.NewSize(ms.Width, 1))
}

func (r *linkTextRenderer) MinSize() fyne.Size {
	// Like a truncating textLine: any width will do, the text is shortened to fit.
	return fyne.NewSize(0, fyne.MeasureText("Mg", r.text.TextSize, r.text.TextStyle).Height)
}

func (r *linkTextRenderer) Refresh() {
	th := r.l.Theme()
	r.text.Text = r.l.text
	r.text.TextSize = th.Size(r.l.size)
	r.text.Color = themeColor(th, colorLink)
	r.line.FillColor = themeColor(th, colorLink)
	if r.l.hovered {
		r.line.Show()
	} else {
		r.line.Hide()
	}
	r.Layout(r.l.Size())
	r.text.Refresh()
	r.line.Refresh()
}

func (r *linkTextRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.text, r.line} }
func (r *linkTextRenderer) Destroy()                     {}

// tapTarget makes any canvas object clickable with a hand pointer: a chip that links to an object.
type tapTarget struct {
	widget.BaseWidget
	content fyne.CanvasObject
	onTap   func()
}

func newTapTarget(content fyne.CanvasObject, onTap func()) *tapTarget {
	t := &tapTarget{content: content, onTap: onTap}
	t.ExtendBaseWidget(t)
	return t
}

// Tapped implements fyne.Tappable.
func (t *tapTarget) Tapped(*fyne.PointEvent) {
	if t.onTap != nil {
		t.onTap()
	}
}

// Cursor implements desktop.Cursorable.
func (t *tapTarget) Cursor() desktop.Cursor { return desktop.PointerCursor }

// CreateRenderer implements fyne.Widget.
func (t *tapTarget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.content)
}
