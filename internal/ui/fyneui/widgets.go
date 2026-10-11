package fyneui

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// The small widgets in this file read every colour and size from the theme in their renderer's
// Refresh, so that a theme change (Fyne refreshes every widget) recolours them; raw canvas objects
// in a container would keep the colours they were created with.

// variant is the theme variant Fyne currently asks for; cogTheme resolves it against its mode.
func variant() fyne.ThemeVariant { return fyne.CurrentApp().Settings().ThemeVariant() }

// themeColor looks a colour up in th for the current variant.
func themeColor(th fyne.Theme, n fyne.ThemeColorName) color.Color { return th.Color(n, variant()) }

// withAlpha returns c with its alpha replaced, for tinted backgrounds of a status hue.
func withAlpha(c color.Color, a uint8) color.Color {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	n.A = a
	return n
}

// textLine is one line of text in a theme colour and size, without the padding widget.Label adds.
type textLine struct {
	widget.BaseWidget
	text     string
	color    fyne.ThemeColorName
	size     fyne.ThemeSizeName
	style    fyne.TextStyle
	align    fyne.TextAlign
	truncate bool // shorten with an ellipsis to the given width instead of asking for room
}

func newTextLine(text string, c fyne.ThemeColorName, size fyne.ThemeSizeName, style fyne.TextStyle) *textLine {
	t := &textLine{text: text, color: c, size: size, style: style}
	t.ExtendBaseWidget(t)
	return t
}

// SetText changes the text; an unchanged text costs nothing.
func (t *textLine) SetText(s string) {
	if s == t.text {
		return
	}
	t.text = s
	t.Refresh()
}

// SetColor changes the text colour token.
func (t *textLine) SetColor(c fyne.ThemeColorName) {
	if c == t.color {
		return
	}
	t.color = c
	t.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (t *textLine) CreateRenderer() fyne.WidgetRenderer {
	r := &textLineRenderer{t: t, obj: canvas.NewText(t.text, color.Transparent)}
	r.Refresh()
	return r
}

type textLineRenderer struct {
	t   *textLine
	obj *canvas.Text
}

func (r *textLineRenderer) Layout(size fyne.Size) {
	if r.t.truncate {
		shown := ellipsize(r.t.text, size.Width, r.obj.TextSize, r.obj.TextStyle)
		if shown != r.obj.Text {
			r.obj.Text = shown
			r.obj.Refresh()
		}
	}
	ms := r.obj.MinSize()
	x := float32(0)
	switch r.t.align {
	case fyne.TextAlignCenter:
		x = (size.Width - ms.Width) / 2
	case fyne.TextAlignTrailing:
		x = size.Width - ms.Width
	}
	r.obj.Move(fyne.NewPos(x, (size.Height-ms.Height)/2))
	r.obj.Resize(ms)
}

func (r *textLineRenderer) MinSize() fyne.Size {
	if r.t.truncate {
		// Any width will do: the text is shortened to fit. Ask for the height of one line.
		return fyne.NewSize(0, fyne.MeasureText("Mg", r.obj.TextSize, r.obj.TextStyle).Height)
	}
	return r.obj.MinSize()
}

func (r *textLineRenderer) Refresh() {
	th := r.t.Theme()
	r.obj.Text = r.t.text
	r.obj.Color = themeColor(th, r.t.color)
	r.obj.TextSize = th.Size(r.t.size)
	r.obj.TextStyle = r.t.style
	r.Layout(r.t.Size())
	r.obj.Refresh()
}

func (r *textLineRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.obj} }
func (r *textLineRenderer) Destroy()                     {}

// surface is a rectangle in a theme colour: a region's background, a border or a divider.
type surface struct {
	widget.BaseWidget
	fill   fyne.ThemeColorName
	radius fyne.ThemeSizeName  // optional corner radius
	stroke fyne.ThemeColorName // optional one-point outline
	min    fyne.Size
}

func newSurface(fill fyne.ThemeColorName) *surface {
	s := &surface{fill: fill}
	s.ExtendBaseWidget(s)
	return s
}

// newDivider returns a one-point line in the separator colour; vertical when vertical is set.
func newDivider(vertical bool) *surface {
	s := newSurface(theme.ColorNameSeparator)
	if vertical {
		s.min = fyne.NewSize(1, 0)
	} else {
		s.min = fyne.NewSize(0, 1)
	}
	return s
}

// CreateRenderer implements fyne.Widget.
func (s *surface) CreateRenderer() fyne.WidgetRenderer {
	r := &surfaceRenderer{s: s, rect: canvas.NewRectangle(color.Transparent)}
	r.Refresh()
	return r
}

type surfaceRenderer struct {
	s    *surface
	rect *canvas.Rectangle
}

func (r *surfaceRenderer) Layout(size fyne.Size)        { r.rect.Resize(size) }
func (r *surfaceRenderer) MinSize() fyne.Size           { return r.s.min }
func (r *surfaceRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.rect} }
func (r *surfaceRenderer) Destroy()                     {}

func (r *surfaceRenderer) Refresh() {
	th := r.s.Theme()
	r.rect.FillColor = themeColor(th, r.s.fill)
	if r.s.radius != "" {
		r.rect.CornerRadius = th.Size(r.s.radius)
	}
	if r.s.stroke != "" {
		r.rect.StrokeColor = themeColor(th, r.s.stroke)
		r.rect.StrokeWidth = th.Size(theme.SizeNameInputBorder)
	}
	r.rect.Refresh()
}

// pill is a short text on a rounded background: a status badge, or a tag such as "current".
// With a status colour the background is the same hue, translucent; otherwise it is the chip
// colour.
type pill struct {
	widget.BaseWidget
	text   string
	fg     fyne.ThemeColorName
	tinted bool // background tinted from fg instead of the chip colour
	bold   bool
}

func newPill(text string, fg fyne.ThemeColorName, tinted bool) *pill {
	p := &pill{text: text, fg: fg, tinted: tinted}
	p.ExtendBaseWidget(p)
	return p
}

// Set changes text and colour.
func (p *pill) Set(text string, fg fyne.ThemeColorName) {
	if text == p.text && fg == p.fg {
		return
	}
	p.text, p.fg = text, fg
	p.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (p *pill) CreateRenderer() fyne.WidgetRenderer {
	r := &pillRenderer{p: p, bg: canvas.NewRectangle(color.Transparent), txt: canvas.NewText("", color.Transparent)}
	r.Refresh()
	return r
}

type pillRenderer struct {
	p   *pill
	bg  *canvas.Rectangle
	txt *canvas.Text
}

func (r *pillRenderer) padding() (h, v float32) {
	pad := r.p.Theme().Size(theme.SizeNamePadding)
	return pad * 2, pad / 2
}

func (r *pillRenderer) MinSize() fyne.Size {
	h, v := r.padding()
	t := r.txt.MinSize()
	return fyne.NewSize(t.Width+2*h, t.Height+2*v)
}

func (r *pillRenderer) Layout(size fyne.Size) {
	ms := r.MinSize()
	// A pill keeps its own height and sits centred, so that a tall row does not stretch it.
	y := (size.Height - ms.Height) / 2
	r.bg.Move(fyne.NewPos(0, y))
	r.bg.Resize(fyne.NewSize(ms.Width, ms.Height))
	t := r.txt.MinSize()
	r.txt.Move(fyne.NewPos((ms.Width-t.Width)/2, y+(ms.Height-t.Height)/2))
	r.txt.Resize(t)
}

func (r *pillRenderer) Refresh() {
	th := r.p.Theme()
	fg := themeColor(th, r.p.fg)
	r.txt.Text = r.p.text
	r.txt.Color = fg
	r.txt.TextSize = th.Size(theme.SizeNameCaptionText)
	r.txt.TextStyle = fyne.TextStyle{Bold: r.p.bold}
	if r.p.tinted {
		r.bg.FillColor = withAlpha(fg, 0x2c)
	} else {
		r.bg.FillColor = themeColor(th, colorChip)
	}
	r.bg.CornerRadius = th.Size(sizeChipRadius)
	r.Layout(r.p.Size())
	r.bg.Refresh()
	r.txt.Refresh()
}

func (r *pillRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.bg, r.txt} }
func (r *pillRenderer) Destroy()                     {}

// dot is a small filled circle in a theme colour: connection state, a status marker.
type dot struct {
	widget.BaseWidget
	color fyne.ThemeColorName
}

func newDot(c fyne.ThemeColorName) *dot {
	d := &dot{color: c}
	d.ExtendBaseWidget(d)
	return d
}

// SetColor changes the dot's colour token.
func (d *dot) SetColor(c fyne.ThemeColorName) {
	if c == d.color {
		return
	}
	d.color = c
	d.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (d *dot) CreateRenderer() fyne.WidgetRenderer {
	r := &dotRenderer{d: d, c: canvas.NewCircle(color.Transparent)}
	r.Refresh()
	return r
}

type dotRenderer struct {
	d *dot
	c *canvas.Circle
}

func (r *dotRenderer) diameter() float32 { return r.d.Theme().Size(sizeDotDiameter) }

func (r *dotRenderer) Layout(size fyne.Size) {
	d := r.diameter()
	r.c.Move(fyne.NewPos((size.Width-d)/2, (size.Height-d)/2))
	r.c.Resize(fyne.NewSize(d, d))
}

func (r *dotRenderer) MinSize() fyne.Size {
	d := r.diameter()
	return fyne.NewSize(d, d)
}

func (r *dotRenderer) Refresh() {
	r.c.FillColor = themeColor(r.d.Theme(), r.d.color)
	r.Layout(r.d.Size())
	r.c.Refresh()
}

func (r *dotRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.c} }
func (r *dotRenderer) Destroy()                     {}

// sizedLayout gives its objects a minimum size from theme sizes: a fixed-width sidebar, a fixed
// height bar. An empty size name leaves that dimension to the content.
type sizedLayout struct {
	width, height fyne.ThemeSizeName
}

func (l sizedLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var ms fyne.Size
	for _, o := range objs {
		if o.Visible() {
			ms = ms.Max(o.MinSize())
		}
	}
	th := fyne.CurrentApp().Settings().Theme()
	if l.width != "" {
		ms.Width = th.Size(l.width)
	}
	if l.height != "" {
		ms.Height = th.Size(l.height)
	}
	return ms
}

func (l sizedLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

// sized stacks objs in a container whose minimum width and height come from theme sizes.
func sized(width, height fyne.ThemeSizeName, objs ...fyne.CanvasObject) *fyne.Container {
	return box(sizedLayout{width: width, height: height}, objs...)
}

// box is a fyne.Container with a layout.
func box(l fyne.Layout, objs ...fyne.CanvasObject) *fyne.Container {
	return &fyne.Container{Layout: l, Objects: objs}
}

// insetLayout pads its objects by a theme size on the given sides.
type insetLayout struct {
	h, v fyne.ThemeSizeName // horizontal and vertical padding; empty: none
}

func (l insetLayout) pads() (float32, float32) {
	th := fyne.CurrentApp().Settings().Theme()
	var h, v float32
	if l.h != "" {
		h = th.Size(l.h)
	}
	if l.v != "" {
		v = th.Size(l.v)
	}
	return h, v
}

func (l insetLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	h, v := l.pads()
	var ms fyne.Size
	for _, o := range objs {
		if o.Visible() {
			ms = ms.Max(o.MinSize())
		}
	}
	return ms.AddWidthHeight(2*h, 2*v)
}

func (l insetLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	h, v := l.pads()
	for _, o := range objs {
		o.Move(fyne.NewPos(h, v))
		o.Resize(size.SubtractWidthHeight(2*h, 2*v))
	}
}

// inset pads objs horizontally by the h size and vertically by the v size.
func inset(h, v fyne.ThemeSizeName, objs ...fyne.CanvasObject) *fyne.Container {
	return box(insetLayout{h: h, v: v}, objs...)
}

// wrapLines breaks text into lines no wider than width, at spaces and newlines. A word wider than
// the width stays whole on its own line.
func wrapLines(text string, width, size float32, style fyne.TextStyle) []string {
	var lines []string
	for _, para := range strings.Split(text, "\n") {
		line := ""
		for _, word := range strings.Fields(para) {
			try := word
			if line != "" {
				try = line + " " + word
			}
			if line != "" && fyne.MeasureText(try, size, style).Width > width {
				lines = append(lines, line)
				line = word
				continue
			}
			line = try
		}
		lines = append(lines, line)
	}
	return lines
}

// paragraph is text wrapped to a known width. widget.Label wraps too, but reports its height
// before it knows its width, which mis-sizes dialogs and toasts; a paragraph is told its width.
type paragraph struct {
	widget.BaseWidget
	text  string
	color fyne.ThemeColorName
	size  fyne.ThemeSizeName
	style fyne.TextStyle
	width float32 // the wrap width; the paragraph's minimum width
}

func newParagraph(text string, c fyne.ThemeColorName, width float32) *paragraph {
	p := &paragraph{text: text, color: c, size: theme.SizeNameText, width: width}
	p.ExtendBaseWidget(p)
	return p
}

// CreateRenderer implements fyne.Widget.
func (p *paragraph) CreateRenderer() fyne.WidgetRenderer {
	r := &paragraphRenderer{p: p}
	r.Refresh()
	return r
}

type paragraphRenderer struct {
	p     *paragraph
	lines []fyne.CanvasObject
}

func (r *paragraphRenderer) lineHeight() float32 {
	th := r.p.Theme()
	return fyne.MeasureText("Mg", th.Size(r.p.size), r.p.style).Height + th.Size(theme.SizeNameLineSpacing)
}

func (r *paragraphRenderer) MinSize() fyne.Size {
	return fyne.NewSize(r.p.width, float32(len(r.lines))*r.lineHeight())
}

func (r *paragraphRenderer) Layout(fyne.Size) {
	h := r.lineHeight()
	for i, l := range r.lines {
		l.Move(fyne.NewPos(0, float32(i)*h))
		l.Resize(l.MinSize())
	}
}

func (r *paragraphRenderer) Refresh() {
	th := r.p.Theme()
	size := th.Size(r.p.size)
	c := themeColor(th, r.p.color)
	texts := wrapLines(r.p.text, r.p.width, size, r.p.style)
	r.lines = r.lines[:0]
	for _, t := range texts {
		o := canvas.NewText(t, c)
		o.TextSize, o.TextStyle = size, r.p.style
		r.lines = append(r.lines, o)
	}
	r.Layout(r.p.Size())
	canvas.Refresh(r.p)
}

func (r *paragraphRenderer) Objects() []fyne.CanvasObject { return r.lines }
func (r *paragraphRenderer) Destroy()                     {}

// iconImage is a canvas.Image showing a tinted icon that re-renders only when the icon or its
// colour changes. canvas.Image.Refresh parses an SVG resource again on every call, and container
// refreshes reach it often: Fyne's table refreshes its whole header row on every table refresh.
type iconImage struct {
	img   *canvas.Image
	res   fyne.Resource
	color color.Color
}

func newIconImage() *iconImage {
	img := canvas.NewImageFromResource(nil)
	img.FillMode = canvas.ImageFillContain
	img.Hide()
	return &iconImage{img: img}
}

// set shows res tinted with the colour token c, or hides the image when res is nil.
func (i *iconImage) set(th fyne.Theme, res fyne.Resource, c fyne.ThemeColorName) {
	col := themeColor(th, c)
	if res == i.res && col == i.color {
		return
	}
	i.res, i.color = res, col
	if res == nil {
		i.img.Resource = nil
		i.img.Hide()
		return
	}
	i.img.Resource = themedIcon(res, c)
	i.img.Show()
	i.img.Refresh()
}

// themedIcon returns an icon resource tinted with a theme colour.
func themedIcon(res fyne.Resource, c fyne.ThemeColorName) fyne.Resource {
	return theme.NewColoredResource(res, c)
}
