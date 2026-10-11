package fyneui

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/yamlview"
)

// Chips of a short list share one line; a longer list shows one chip per line.
const (
	chipsPerLine     = 6
	chipsLineMaxText = 64 // the most characters of chip text on one line
)

// drawer is the detail drawer, after Lens's (capture, 2026-10-11): the object's kind and name in
// the title bar with its actions as icons (the registry's actions for the kind, as in the row
// menu), then the Properties section, the kind's own sections and the events. Cogpit's YAML view
// is a second tab. It refreshes when the engine reports the object changed.
type drawer struct {
	sh *shell

	shown       *model.ObjectRef
	last        *engine.DetailModel
	acts        []actionState
	busy, dirty bool
	expanded    map[string]bool   // folded lists the user opened, by foldKey
	revealed    map[string]string // values item actions show in place (a Secret's), by revealKey (secretreveal.go)

	kind     *textLine
	title    *textLine
	status   *pill
	missing  *textLine
	actions  *fyne.Container // the action icons in the title bar
	tabs     *container.AppTabs
	details  *fyne.Container
	yaml     *yamlview.View
	yamlText string
	root     fyne.CanvasObject
}

func newDrawer(sh *shell) *drawer {
	d := &drawer{sh: sh, expanded: map[string]bool{}, revealed: map[string]string{}}
	d.kind = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	d.title = newTextLine("", theme.ColorNameForeground, theme.SizeNameSubHeadingText, fyne.TextStyle{Bold: true})
	d.title.truncate = true
	d.status = newPill("", colorMuted, true)
	d.status.Hide()
	d.missing = newTextLine("This object is no longer in the cluster.", colorStatusWarning, theme.SizeNameCaptionText, fyne.TextStyle{})
	d.missing.Hide()
	d.actions = container.NewHBox()

	closeB := newTipButton(sh, theme.CancelIcon(), "Close", sh.closeDrawer)

	d.details = container.NewVBox()
	d.yaml = yamlview.New()
	d.yaml.SetReadOnly(true)
	d.tabs = container.NewAppTabs(
		container.NewTabItem("Details", container.NewVScroll(inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, d.details))),
		container.NewTabItem("YAML", d.yaml),
	)

	titleRow := container.NewBorder(nil, nil, nil,
		container.NewHBox(container.NewCenter(d.status), d.actions, newDivider(true), closeB), d.title)
	header := container.NewStack(newSurface(colorDrawerHeader),
		inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
			container.NewVBox(d.kind, titleRow, d.missing)))
	d.root = container.NewStack(newSurface(colorDrawer),
		container.NewBorder(container.NewVBox(header, newDivider(false)), nil, nil, nil, d.tabs))
	return d
}

// request fetches the drawer object's detail and evaluates its actions off the main goroutine,
// then shows them. Requests that arrive while one runs collapse into one more fetch afterwards.
func (d *drawer) request() {
	ref := d.sh.ui.Drawer
	if ref == nil {
		return
	}
	if d.busy {
		d.dirty = true
		return
	}
	d.busy = true
	r := *ref
	if d.shown == nil || *d.shown != r {
		d.showLoading(r)
	}
	go func() {
		var (
			m    *engine.DetailModel
			err  error
			acts []actionState
		)
		func() {
			defer func() {
				if p := recover(); p != nil {
					err = fmt.Errorf("internal error: %v", p)
				}
			}()
			m, err = d.sh.eng.Detail(r)
			var containers []string
			if m != nil {
				containers = append(append(containers, m.Containers...), m.InitContainers...)
			}
			acts = d.sh.evaluateActions(r, containers)
		}()
		d.sh.do(func() {
			d.busy = false
			if cur := d.sh.ui.Drawer; cur != nil && *cur == r {
				d.show(r, m, err, acts)
			}
			if d.dirty {
				d.dirty = false
				d.request()
			}
		})
	}()
}

// showLoading retargets the drawer at a new object while its detail loads.
func (d *drawer) showLoading(ref model.ObjectRef) {
	r := ref
	d.shown = &r
	d.last = nil
	d.acts = nil
	d.expanded = map[string]bool{}
	d.revealed = map[string]string{}
	d.kind.SetText(singular(d.sh.kindTitle(ref.Kind)))
	d.title.SetText(ref.Name)
	d.status.Hide()
	d.missing.Hide()
	d.actions.Objects = nil
	d.actions.Refresh()
	d.details.Objects = []fyne.CanvasObject{newTextLine("Loading…", colorMuted, theme.SizeNameText, fyne.TextStyle{})}
	d.details.Refresh()
	d.setYAML("", false)
	d.tabs.SelectIndex(0)
}

// show renders a detail model. Unchanged sections are not rebuilt, so that a churning object does
// not flicker or lose the scroll position.
func (d *drawer) show(ref model.ObjectRef, m *engine.DetailModel, err error, acts []actionState) {
	if d.shown == nil || *d.shown != ref {
		d.showLoading(ref)
	}
	if err != nil || m == nil {
		msg := "No details."
		if err != nil {
			msg = "Could not load the details: " + err.Error()
		}
		d.details.Objects = []fyne.CanvasObject{newTextLine(msg, colorStatusError, theme.SizeNameText, fyne.TextStyle{})}
		d.details.Refresh()
		return
	}
	kind, name := splitTitle(m.Title)
	if kind == "" {
		kind = singular(d.sh.kindTitle(ref.Kind))
	}
	d.kind.SetText(kind)
	d.title.SetText(name)
	if label := statusLabel(m); label != "" && m.Status != model.StatusNone {
		d.status.Set(label, statusColorName(m.Status))
		d.status.Show()
	} else {
		d.status.Hide()
	}
	if m.Missing {
		d.missing.Show()
	} else {
		d.missing.Hide()
	}
	d.showActions(ref, acts)

	prev := d.last
	d.last = m
	if prev == nil || !reflect.DeepEqual(prev.Sections, m.Sections) || !reflect.DeepEqual(prev.Events, m.Events) {
		d.rebuild()
	}
	// The same object changed in the cache: keep the reader's place in the YAML. Another object:
	// start at the top.
	d.setYAML(m.YAML, prev != nil && prev.Ref.Same(m.Ref))
}

// rebuild renders the shown model's sections and events again.
func (d *drawer) rebuild() {
	if d.last == nil {
		return
	}
	d.details.Objects = d.buildDetails(d.last)
	d.details.Refresh()
}

// clear forgets the shown object when the drawer closes.
func (d *drawer) clear() {
	d.shown, d.last, d.acts, d.dirty = nil, nil, nil, false
	d.revealed = map[string]string{}
	d.sh.tip.hide()
}

// setYAML hands the YAML to the viewer only when it changed. For an update of the shown object it
// keeps the scroll position, caret and selection (UpdateText); SetText would jump to the top on
// every watch event.
func (d *drawer) setYAML(text string, sameObject bool) {
	if text == d.yamlText {
		return
	}
	d.yamlText = text
	if sameObject {
		d.yaml.UpdateText(text)
	} else {
		d.yaml.SetText(text)
	}
	d.yaml.Refresh()
}

// showActions puts the object's actions in the title bar as icons, in the row menu's order (Lens:
// a pod's seven icons end with Edit and Delete; a node shows Shell, Cordon or Uncordon, Drain,
// Edit, Delete). The tooltip names the action and, for a disabled one, says why.
func (d *drawer) showActions(ref model.ObjectRef, acts []actionState) {
	if len(d.actions.Objects) > 0 && sameActions(d.acts, acts) {
		return // a churning object keeps its buttons, and a hovered button its tooltip
	}
	d.acts = acts
	d.actions.Objects = d.actions.Objects[:0]
	for _, a := range acts {
		b := newTipButton(d.sh, commandIcon(a.cmd.ID), actionTip(a), nil)
		b.OnTapped = func() { d.run(ref, a, b) }
		if a.reason != "" {
			b.Disable()
		}
		d.actions.Add(b)
	}
	d.actions.Refresh()
}

// sameActions reports whether two evaluations of an object's actions show the same toolbar.
func sameActions(a, b []actionState) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].cmd != b[i].cmd || a[i].reason != b[i].reason || a[i].dialog != b[i].dialog ||
			strings.Join(a[i].containers, ",") != strings.Join(b[i].containers, ",") || string(a[i].prefill) != string(b[i].prefill) {
			return false
		}
	}
	return true
}

// run runs an action from the toolbar. A container command on a pod with several containers asks
// which, in a menu below the icon.
func (d *drawer) run(ref model.ObjectRef, a actionState, from fyne.CanvasObject) {
	if len(a.containers) <= 1 || a.dialog {
		first := ""
		if len(a.containers) == 1 {
			first = a.containers[0]
		}
		d.sh.runAction(a, ref, first)
		return
	}
	items := make([]*fyne.MenuItem, 0, len(a.containers))
	for _, c := range a.containers {
		items = append(items, fyne.NewMenuItem(c, func() { d.sh.runAction(a, ref, c) }))
	}
	if cnv := fyne.CurrentApp().Driver().CanvasForObject(from); cnv != nil {
		widget.ShowPopUpMenuAtRelativePosition(fyne.NewMenu(a.cmd.Label, items...), cnv,
			fyne.NewPos(0, from.Size().Height), from)
	}
}

// buildDetails renders the sections in their order, a group heading ("Containers") above the
// first section of each group, then the events.
func (d *drawer) buildDetails(m *engine.DetailModel) []fyne.CanvasObject {
	out := make([]fyne.CanvasObject, 0, 4*len(m.Sections)+4)
	group := ""
	for i, sec := range m.Sections {
		if i > 0 {
			out = append(out, newSpace(theme.SizeNameInnerPadding))
		}
		if sec.Group != "" && sec.Group != group {
			out = append(out, newTextLine(sec.Group, theme.ColorNameForeground, theme.SizeNameSubHeadingText, fyne.TextStyle{Bold: true}),
				newSpace(theme.SizeNamePadding))
		}
		group = sec.Group
		out = append(out, d.sectionTitle(sec), newDivider(false))
		if sec.Table != nil {
			out = append(out, d.table(m.Ref, sec.Table))
		}
		if len(sec.Items) > 0 {
			grid := box(kvLayout{})
			for _, it := range sec.Items {
				name := newTextLine(it.Name, colorMuted, theme.SizeNameText, fyne.TextStyle{})
				grid.Objects = append(grid.Objects, name, d.itemValue(m.Ref, sec, it))
			}
			out = append(out, grid)
		}
	}
	out = append(out, newSpace(theme.SizeNameInnerPadding),
		newTextLine("Events", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true}), newDivider(false))
	return append(out, d.buildEvents(m.Events)...)
}

// sectionTitle is a section's bold title; a container's carries a dot in its status's colour.
func (d *drawer) sectionTitle(sec model.DetailSection) fyne.CanvasObject {
	title := newTextLine(sec.Title, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	if sec.Group == "" {
		return title
	}
	for _, it := range sec.Items {
		if it.Name == "Status" && it.Status != model.StatusNone {
			return container.NewHBox(container.NewCenter(newDot(statusColorName(it.Status))), title)
		}
	}
	return title
}

// foldKey names a folded list for the expanded state, which survives refreshes of the object.
func foldKey(sec model.DetailSection, it model.DetailItem) string {
	return sec.Group + "\x00" + sec.Title + "\x00" + it.Name
}

// itemValue renders one value: a folded list as its count ("6 Labels", a link that unfolds it),
// chips, a link to the object it names, a status pill, else one line per line. An item's actions
// follow the value as buttons; an action marked Link is the value itself, a link that runs it.
func (d *drawer) itemValue(ref model.ObjectRef, sec model.DetailSection, it model.DetailItem) fyne.CanvasObject {
	it, revealed := d.revealedItem(it)
	if len(it.Actions) == 0 {
		return d.foldedValue(ref, sec, it)
	}
	var v fyne.CanvasObject
	if revealed {
		v = revealedValue(it.Value)
	}
	objs := []fyne.CanvasObject{nil}
	for _, a := range it.Actions {
		act := a
		if act.Link && v == nil {
			v = newLinkText(it.Value, func() { d.sh.runItemAction(ref, act) })
			continue
		}
		b := widget.NewButton(d.actionLabel(act), func() { d.sh.runItemAction(ref, act) })
		b.Importance = widget.LowImportance
		objs = append(objs, b)
	}
	if v == nil {
		v = d.foldedValue(ref, sec, it)
	}
	objs[0] = v
	if revealable(it) { // a revealed value takes the width the buttons leave and wraps in it
		return container.NewBorder(nil, nil, nil, container.NewHBox(objs[1:]...), v)
	}
	natural := float32(0)
	if it.Fold == "" {
		size := d.sh.app.Settings().Theme().Size(theme.SizeNameText)
		for _, line := range strings.Split(it.Value, "\n") {
			natural = fyne.Max(natural, fyne.MeasureText(line, size, fyne.TextStyle{}).Width)
		}
	}
	return box(actionRowLayout{natural: natural}, objs...)
}

// actionRowLayout puts a drawer value and its action buttons on one line: the value as wide as
// its text (natural; 0 takes what the buttons leave), shortened when the drawer is narrower, the
// buttons right after it. The value's own widgets ask for no width, since they shorten to fit.
type actionRowLayout struct{ natural float32 }

func (actionRowLayout) gap() float32 {
	return fyne.CurrentApp().Settings().Theme().Size(theme.SizeNamePadding)
}

func (l actionRowLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var w, h float32
	for _, o := range objs {
		ms := o.MinSize()
		w += ms.Width + l.gap()
		h = fyne.Max(h, ms.Height)
	}
	return fyne.NewSize(w, h)
}

func (l actionRowLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	if len(objs) == 0 {
		return
	}
	var buttons float32
	for _, b := range objs[1:] {
		buttons += b.MinSize().Width + l.gap()
	}
	vw := fyne.Max(size.Width-buttons, 0)
	if l.natural > 0 {
		vw = fyne.Min(vw, l.natural+l.gap())
	}
	vh := objs[0].MinSize().Height
	objs[0].Move(fyne.NewPos(0, (size.Height-vh)/2))
	objs[0].Resize(fyne.NewSize(vw, vh))
	x := vw + l.gap()
	for _, b := range objs[1:] {
		ms := b.MinSize()
		b.Move(fyne.NewPos(x, (size.Height-ms.Height)/2))
		b.Resize(ms)
		x += ms.Width + l.gap()
	}
}

// foldedValue renders a value, folded behind its count when the item folds.
func (d *drawer) foldedValue(ref model.ObjectRef, sec model.DetailSection, it model.DetailItem) fyne.CanvasObject {
	if it.Fold != "" {
		key := foldKey(sec, it)
		n := it.Entries()
		noun := it.Fold
		if n == 1 {
			noun = singular(noun)
		}
		toggle := newLinkText(strconv.Itoa(n)+" "+noun, func() {
			d.expanded[key] = !d.expanded[key]
			d.rebuild()
		})
		if !d.expanded[key] {
			return toggle
		}
		return container.NewVBox(toggle, d.plainValue(ref, it))
	}
	return d.plainValue(ref, it)
}

// plainValue renders a value without folding.
func (d *drawer) plainValue(ref model.ObjectRef, it model.DetailItem) fyne.CanvasObject {
	switch {
	case len(it.Chips) > 0:
		return d.chips(ref, it.Chips)
	case it.Link != nil && d.sh.view.serves(it.Link.Kind):
		link := *it.Link
		return newLinkText(it.Value, func() { d.sh.followLink(link, ref.Context, false) })
	case it.Status != model.StatusNone:
		return container.NewHBox(newPill(it.Value, statusColorName(it.Status), true))
	}
	lines := strings.Split(it.Value, "\n")
	if len(lines) == 1 {
		t := newTextLine(it.Value, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{})
		t.truncate = true
		return t
	}
	v := container.NewVBox()
	for _, l := range lines {
		t := newTextLine(l, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{})
		t.truncate = true
		v.Add(t)
	}
	return v
}

// chips renders a list as chips: a short list on one line, a longer one a chip per line. A chip
// with a status is tinted in its colour; one with a link goes to the object.
func (d *drawer) chips(ref model.ObjectRef, chips []model.Chip) fyne.CanvasObject {
	total := 0
	for _, c := range chips {
		total += len(c.Text)
	}
	oneLine := len(chips) <= chipsPerLine && total <= chipsLineMaxText
	out := container.NewVBox()
	line := container.NewHBox()
	for _, c := range chips {
		var o fyne.CanvasObject
		switch {
		case c.Link != nil && d.sh.view.serves(c.Link.Kind):
			link := *c.Link
			o = newTapTarget(newPill(c.Text, colorLink, false), func() { d.sh.followLink(link, ref.Context, false) })
		case c.Status != model.StatusNone:
			o = newPill(c.Text, statusColorName(c.Status), true)
		default:
			o = newPill(c.Text, theme.ColorNameForeground, false)
		}
		if oneLine {
			line.Add(o)
		} else {
			out.Add(container.NewHBox(o))
		}
	}
	if oneLine {
		return line
	}
	return out
}

// detailTable renders a plain table, its texts only: the column titles muted above the rows, each
// column as wide as its widest line (detailTableLayout), a cell of several lines one line under
// the other. Tables of cells (links, statuses, warning icons) are drawn by table.
func detailTable(t *model.DetailTable) fyne.CanvasObject {
	if len(t.Columns) == 0 {
		return container.NewVBox()
	}
	// The cells go into the grid at once: Container.Add lays the grid out on every call.
	objs := make([]fyne.CanvasObject, 0, len(t.Columns)*(len(t.Rows)+1))
	for _, c := range t.Columns {
		l := newTextLine(c, colorMuted, theme.SizeNameText, fyne.TextStyle{})
		l.truncate = true
		objs = append(objs, l)
	}
	for _, row := range t.Rows {
		for i := range t.Columns {
			text := ""
			if i < len(row) {
				text = row[i]
			}
			objs = append(objs, tableCellText(text, theme.ColorNameForeground))
		}
	}
	return box(&detailTableLayout{cols: len(t.Columns)}, objs...)
}

// tableCellText is a table cell of text in colour fg: a truncating line per line of its text.
func tableCellText(text string, fg fyne.ThemeColorName) fyne.CanvasObject {
	lines := strings.Split(text, "\n")
	cell := func(s string) *textLine {
		v := newTextLine(s, fg, theme.SizeNameText, fyne.TextStyle{})
		v.truncate = true
		return v
	}
	if len(lines) == 1 {
		return cell(text)
	}
	v := container.NewVBox()
	for _, l := range lines {
		v.Add(cell(l))
	}
	return v
}

// detailTableLayout lays out table cells (textLines, linkTexts, and boxes of textLines for cells of
// several lines) row by row in cols columns, each as wide as its widest line and each row as tall
// as its tallest cell, rows of several lines further apart; when the columns do not fit, the
// widest give up width first (capWidths) and their texts shorten.
//
// It keeps its last measurement while the objects and the text size stay the same: a table's
// cells never change once built (the drawer builds a new table instead), and Fyne asks for MinSize
// and Layout several times per pass, which for a node's hundreds of pods means thousands of texts.
type detailTableLayout struct {
	cols int

	measured    []fyne.CanvasObject
	textSize    float32
	w, rowH     []float32
	gap, rowGap float32
}

// widths measures the cells: each column's width, each row's height, the gap between columns and
// the one between rows. Callers may change the widths they get.
func (l *detailTableLayout) widths(objs []fyne.CanvasObject) (w, rowH []float32, gap, rowGap float32) {
	th := fyne.CurrentApp().Settings().Theme()
	size := th.Size(theme.SizeNameText)
	if len(objs) > 0 && len(objs) == len(l.measured) && &objs[0] == &l.measured[0] && size == l.textSize {
		return slices.Clone(l.w), l.rowH, l.gap, l.rowGap
	}
	defer func() {
		l.measured, l.textSize, l.w, l.rowH, l.gap, l.rowGap = objs, size, slices.Clone(w), rowH, gap, rowGap
	}()
	gap = th.Size(theme.SizeNamePadding) * 3
	rowGap = gap / 3
	w = make([]float32, l.cols)
	rowH = make([]float32, (len(objs)+l.cols-1)/l.cols)
	var measure func(o fyne.CanvasObject) float32
	measure = func(o fyne.CanvasObject) float32 {
		switch t := o.(type) {
		case *textLine:
			return fyne.MeasureText(t.text, size, t.style).Width
		case *linkText:
			return fyne.MeasureText(t.text, size, fyne.TextStyle{}).Width
		case *fyne.Container:
			widest := float32(0)
			for _, c := range t.Objects {
				widest = fyne.Max(widest, measure(c))
			}
			return widest
		}
		return o.MinSize().Width // an icon
	}
	for i, o := range objs {
		w[i%l.cols] = fyne.Max(w[i%l.cols], measure(o))
		rowH[i/l.cols] = fyne.Max(rowH[i/l.cols], o.MinSize().Height)
	}
	if len(rowH) > 1 && slices.Max(rowH[1:]) > rowH[0] { // a row of several lines
		rowGap = gap * 2 / 3
	}
	return w, rowH, gap, rowGap
}

func (l *detailTableLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	_, rowH, _, rowGap := l.widths(objs)
	h := float32(0)
	for _, rh := range rowH {
		h += rh + rowGap
	}
	return fyne.NewSize(0, h)
}

func (l *detailTableLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	w, rowH, gap, rowGap := l.widths(objs)
	total := gap * float32(l.cols-1)
	for _, x := range w {
		total += x
	}
	if total > size.Width && total > 0 {
		capWidths(w, fyne.Max(size.Width-gap*float32(l.cols-1), 0))
	}
	y := float32(0)
	for i, o := range objs {
		col, row := i%l.cols, i/l.cols
		if col == 0 && row > 0 {
			y += rowH[row-1] + rowGap
		}
		x := float32(0)
		for c := 0; c < col; c++ {
			x += w[c] + gap
		}
		o.Move(fyne.NewPos(x, y))
		o.Resize(fyne.NewSize(w[col], o.MinSize().Height)) // at the top of a taller row
	}
}

// capWidths makes column widths fit avail by capping the widest at one width: columns narrower
// than their share keep their width, so a short column (a rule's verbs) is never cut because a
// long one (its API groups) is too wide.
func capWidths(w []float32, avail float32) {
	if len(w) == 0 {
		return
	}
	sorted := slices.Clone(w)
	slices.Sort(sorted)
	limit := sorted[len(sorted)-1]
	remaining := avail
	for i, x := range sorted {
		if share := remaining / float32(len(sorted)-i); x > share {
			limit = share
			break
		}
		remaining -= x
	}
	for i := range w {
		w[i] = fyne.Min(w[i], limit)
	}
}

// buildEvents renders the object's events, newest first, at the end of the drawer as Lens does;
// "No events found" when there are none (capture, 2026-10-11).
func (d *drawer) buildEvents(events []engine.EventItem) []fyne.CanvasObject {
	if len(events) == 0 {
		return []fyne.CanvasObject{newTextLine("No events found", colorMuted, theme.SizeNameText, fyne.TextStyle{})}
	}
	now := d.sh.now()
	out := make([]fyne.CanvasObject, 0, 2*len(events))
	for _, e := range events {
		st := e.Status
		if st == model.StatusNone {
			st = model.StatusOK
			if e.Type == "Warning" {
				st = model.StatusWarning
			}
		}
		var meta []string
		if e.Source != "" {
			meta = append(meta, e.Source)
		}
		if e.Count > 1 {
			meta = append(meta, fmt.Sprintf("×%d", e.Count))
		}
		if e.LastSeen > 0 {
			meta = append(meta, model.FormatAge(now.Sub(time.Unix(e.LastSeen, 0)))+" ago")
		}
		head := container.NewBorder(nil, nil,
			container.NewHBox(newPill(e.Type, statusColorName(st), true),
				newTextLine(e.Reason, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})),
			newTextLine(strings.Join(meta, " · "), colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{}))
		msg := widget.NewLabel(e.Message)
		msg.Wrapping = fyne.TextWrapWord
		out = append(out, container.NewVBox(head, msg), newDivider(false))
	}
	return out
}

// splitTitle splits "Pod: web-0" into "Pod" and "web-0".
func splitTitle(t string) (kind, name string) {
	if k, n, ok := strings.Cut(t, ": "); ok {
		return k, n
	}
	return "", t
}

// statusLabel is the drawer's status pill text: the Status item of the first section that has
// one, else nothing (a node's state is its Conditions).
func statusLabel(m *engine.DetailModel) string {
	for _, s := range m.Sections {
		for _, it := range s.Items {
			if it.Name == "Status" && it.Value != "" {
				return it.Value
			}
		}
	}
	return ""
}

// singular turns a plural noun into its singular: "Pods" → "Pod", "Ingresses" → "Ingress",
// "Policies" → "Policy", "volumes" → "volume".
func singular(title string) string {
	switch {
	case strings.HasSuffix(title, "sses"):
		return strings.TrimSuffix(title, "es")
	case strings.HasSuffix(title, "ies"):
		return strings.TrimSuffix(title, "ies") + "y"
	case strings.HasSuffix(title, "s"):
		return strings.TrimSuffix(title, "s")
	}
	return title
}

// kvLayout lays out name and value pairs in two columns: names in the theme's label width,
// values in the rest. Objects alternate name, value.
type kvLayout struct{}

func (kvLayout) dims() (nameW, gap float32) {
	th := fyne.CurrentApp().Settings().Theme()
	return th.Size(sizeDrawerLabel), th.Size(theme.SizeNamePadding) * 2
}

func (l kvLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	nameW, gap := l.dims()
	var h, valueW float32
	for i := 0; i+1 < len(objs); i += 2 {
		n, v := objs[i].MinSize(), objs[i+1].MinSize()
		h += fyne.Max(n.Height, v.Height) + gap/2
		valueW = fyne.Max(valueW, v.Width)
	}
	return fyne.NewSize(nameW+gap+valueW, h)
}

func (l kvLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	nameW, gap := l.dims()
	y := float32(0)
	for i := 0; i+1 < len(objs); i += 2 {
		n, v := objs[i].MinSize(), objs[i+1].MinSize()
		rowH := fyne.Max(n.Height, v.Height)
		objs[i].Move(fyne.NewPos(0, y))
		objs[i].Resize(fyne.NewSize(nameW, n.Height))
		objs[i+1].Move(fyne.NewPos(nameW+gap, y))
		objs[i+1].Resize(fyne.NewSize(fyne.Max(size.Width-nameW-gap, 0), v.Height))
		y += rowH + gap/2
	}
}

// newSpace is vertical empty space of a theme size.
func newSpace(size fyne.ThemeSizeName) fyne.CanvasObject {
	return sized("", size)
}

// newHGap is horizontal empty space of a theme size.
func newHGap(size fyne.ThemeSizeName) fyne.CanvasObject {
	return sized(size, "")
}
