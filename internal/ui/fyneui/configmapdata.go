package fyneui

import (
	"errors"
	"fmt"
	"image/color"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/ui/fyneui/yamlview"
)

// A Config Map's Data in the drawer (the owner's F3, decided 2026-10-15; decisions D-53): each key
// on a line of its own and its value below it in an editor at the details' full width, as Lens's
// ConfigMap drawer shows them (captured 2026-10-15: line numbers, colours by the value's type,
// about five lines high, a resize handle), in Cogpit's own drawing: yamlview with plain scalars
// coloured as strings, as tall as the value's lines between cmMinLines and cmMaxLines, and a grip
// at its bottom right that drags it taller or shorter. Save at the end of the data writes every
// changed value at once: a review of the diff and the server's dry run (configMap.reviewData),
// then configMap.saveData with the version the values are based on as precondition, so a Config
// Map that changed meanwhile is shown as a conflict and never overwritten unasked. The values come
// whole from configMap.readData (the drawer model's are cut); until then, and where nothing can be
// saved (the demo cluster, no update permission, an immutable Config Map, a Helm release record),
// the editors are read-only and the line under them says why. Edited values the drawer moves away
// from are kept in memory (drawer.unsaved) and come back when the Config Map is shown again.

func init() { drawerSections["configmaps\x00Data"] = configMapDataSection }

// Editor heights in lines, and what the drawer keeps across rebuilds.
const (
	cmDataKept = "configmap.data"
	cmMinLines = 3  // a one-line value's editor still looks like one, with room for a new line
	cmMaxLines = 12 // a longer value scrolls in its editor
	cmMaxDrag  = 40 // the grip drags an editor to at most this many lines
)

// configMapDataParams mirror configMap.readData's, reviewData's and saveData's parameters.
type configMapDataParams struct {
	Ref             model.ObjectRef   `json:"ref"`
	ResourceVersion string            `json:"resourceVersion"`
	Data            map[string]string `json:"data"`
	Context         *int              `json:"context,omitempty"`
}

// cmUnsaved are a Config Map's edited values the drawer moved away from: the version they are based
// on and each edited key's text.
type cmUnsaved struct {
	base   string
	values map[string]string
}

// cmData is the Data section of the Config Map the drawer shows.
type cmData struct {
	d     *drawer
	ref   model.ObjectRef
	rows  map[string]*cmRow
	keys  []string // the keys in the order shown
	more  string   // the note that more keys are not listed
	list  *fyne.Container
	moreL *textLine
	// The footer at the end of the data: Revert and Save, and the line that says what the editors
	// hold.
	save, revert *widget.Button
	buttons      *fyne.Container
	status       *paragraph
	root         *fyne.Container

	width    float32 // the details' width the keys were wrapped at
	base     string  // the version the edited values are based on
	latest   string  // the newest version read
	readOnly string  // why no value of this Config Map can be saved (immutable, a Helm release record)
	reason   string  // why no value can be saved now, the command's reasons included; "" when they can
	hidden   bool    // the save command is not in this build: no footer
	loaded   bool
	loads    int
	closed   bool
	review   *cmReview
}

// cmRow is one key and its value's editor.
type cmRow struct {
	cd     *cmData
	key    string
	name   fyne.CanvasObject
	view   *yamlview.View
	frame  *fyne.Container // the editor on its background, with the grip
	note   *textLine       // a value that is not shown (a Helm release record's payload)
	value  *fyne.Container // what shows below the key: the frame, or the note
	orig   string          // the value as last read
	read   bool            // orig is the whole value (configMap.readData), not the drawer model's cut text
	height float32         // the height the grip dragged the editor to; 0: by its lines
	shown  float32         // the editor height the list was last laid out for
}

// configMapDataSection renders a Config Map's Data: the section kept for the shown Config Map,
// its keys from the drawer model until the values are read, then read again after every change.
func configMapDataSection(d *drawer, ref model.ObjectRef, sec model.DetailSection) fyne.CanvasObject {
	cd, ok := d.kept[cmDataKept].(*cmData)
	if !ok || cd.ref != ref {
		cd = newCMData(d, ref)
		d.kept[cmDataKept] = cd
	}
	cd.fromModel(sec)
	cd.load()
	return cd.root
}

func newCMData(d *drawer, ref model.ObjectRef) *cmData {
	cd := &cmData{d: d, ref: ref, rows: map[string]*cmRow{}}
	cd.list = box(keyedLayout{})
	cd.moreL = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	cd.moreL.Hide()
	cd.save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), cd.openReview)
	cd.save.Importance = widget.HighImportance
	cd.revert = widget.NewButton("Revert", cd.revertAll)
	cd.status = newParagraph("", colorMuted, d.contentWidth())
	cd.status.Hide()
	cd.buttons = container.NewHBox(layout.NewSpacer(), cd.revert, cd.save)
	cd.root = container.NewVBox(cd.list, cd.moreL, cd.buttons, cd.status)
	cd.width = d.contentWidth()
	cd.refreshFooter()
	return cd
}

// fromModel adds the rows of the drawer model's keys that are not shown yet, their cut values
// read-only until the values are read, and wraps the keys anew when the drawer's width changed.
func (cd *cmData) fromModel(sec model.DetailSection) {
	if w := cd.d.contentWidth(); w != cd.width {
		cd.width = w
		for _, r := range cd.rows {
			r.name = cd.d.keyName(r.key)
		}
		cd.status.width = w
		cd.status.Refresh()
	}
	if cd.loaded {
		cd.layoutRows()
		return
	}
	cd.keys = cd.keys[:0]
	for _, it := range sec.Items {
		if it.Name == "" {
			cd.more = it.Value
			continue
		}
		r := cd.row(it.Name)
		if !r.read {
			r.orig = it.Value
			r.view.UpdateText(it.Value)
		}
		cd.keys = append(cd.keys, it.Name)
	}
	cd.layoutRows()
}

// row returns the row of key, made when it is new.
func (cd *cmData) row(key string) *cmRow {
	if r := cd.rows[key]; r != nil {
		return r
	}
	r := &cmRow{cd: cd, key: key, name: cd.d.keyName(key), view: yamlview.New()}
	r.view.SetPlainStrings(true)
	r.view.SetReadOnly(true)
	r.view.OnChanged = func() { cd.edited(r) }
	r.view.OnScrollPast = cd.d.scroll.Scrolled // the drawer scrolls on where the editor cannot
	r.frame = box(cmFrameLayout{r: r}, newSurface(theme.ColorNameInputBackground), r.view, newCMGrip(r))
	r.note = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	r.note.Hide()
	r.value = container.NewStack(r.frame)
	cd.rows[key] = r
	return r
}

// layoutRows puts the rows in the list in the order of keys, then the note on more keys.
func (cd *cmData) layoutRows() {
	objs := make([]fyne.CanvasObject, 0, 2*len(cd.keys))
	for _, k := range cd.keys {
		r := cd.rows[k]
		objs = append(objs, r.name, r.value)
	}
	cd.list.Objects = objs
	cd.list.Refresh()
	if cd.more != "" {
		cd.moreL.SetText(cd.more)
		cd.moreL.Show()
	} else {
		cd.moreL.Hide()
	}
}

// load reads the values whole from the cache. An older read's answer is dropped.
func (cd *cmData) load() {
	cd.loads++
	n := cd.loads
	cd.d.sh.dispatch("configMap.readData", refParams{Ref: cd.ref}, func(res any, err error) {
		if cd.closed || n != cd.loads {
			return
		}
		var data engine.ConfigMapData
		if err == nil {
			err = decode(res, &data)
		}
		if err != nil {
			cd.setStatus("Could not read the values: "+errText(err), colorStatusError)
			return
		}
		cd.apply(data)
	})
}

// apply shows the values as read: an edited value keeps the user's text, every other one shows
// the value read. The edited values stay based on the version they were edited at; without edits
// the editors are based on the version read.
func (cd *cmData) apply(data engine.ConfigMapData) {
	first := !cd.loaded
	cd.loaded, cd.latest, cd.readOnly = true, data.ResourceVersion, data.ReadOnly
	keys := make([]string, 0, len(data.Values))
	for _, v := range data.Values {
		r := cd.row(v.Key)
		clean := !r.read || !r.dirty()
		r.orig, r.read = v.Value, true
		r.setNote(v.Note)
		if clean {
			r.view.UpdateText(v.Value)
		}
		keys = append(keys, v.Key)
	}
	for _, k := range cd.keys { // an edited key the cluster dropped stays, and its save is refused
		if r := cd.rows[k]; r != nil && r.read && r.dirty() && !slices.Contains(keys, k) {
			keys = append(keys, k)
		}
	}
	cd.keys = keys
	cd.more = ""
	if data.More > 0 {
		cd.more = fmt.Sprintf("%d more keys: see the YAML", data.More)
	}
	if !cd.dirty() {
		cd.base = data.ResourceVersion
	}
	cd.reason = cd.saveReason()
	if first {
		cd.restore()
	}
	cd.layoutRows()
	cd.refreshFooter()
}

// restore brings back the edited values the drawer moved away from, still based on the version
// they were edited at: a Config Map that changed since shows the conflict when they are saved.
func (cd *cmData) restore() {
	u := cd.d.unsaved[cd.ref.String()]
	if u == nil {
		return
	}
	delete(cd.d.unsaved, cd.ref.String())
	back := 0
	for k, v := range u.values {
		if r := cd.rows[k]; r != nil && r.read && !r.note.Visible() {
			r.view.SetText(v)
			back++
		}
	}
	if back > 0 && cd.dirty() {
		cd.base = u.base
		cd.setStatus(fmt.Sprintf("Your unsaved changes are back, based on version %s: Save writes them, Revert drops them.", u.base), colorStatusWarning)
	}
}

// dirty reports whether the user changed a value since it was read.
func (cd *cmData) dirty() bool {
	for _, r := range cd.rows {
		if r.read && r.dirty() {
			return true
		}
	}
	return false
}

// edits are the edited values by key.
func (cd *cmData) edits() map[string]string {
	out := map[string]string{}
	for k, r := range cd.rows {
		if r.read && r.dirty() {
			out[k] = r.view.Text()
		}
	}
	return out
}

// edited follows the user's typing in a row: its height, when its lines changed it, and the
// footer.
func (cd *cmData) edited(r *cmRow) {
	r.resized()
	cd.refreshFooter()
}

// saveReason says why no value can be saved: the Config Map's own reason, else the command's
// (the demo cluster, no update permission, the Config Map gone); "" when Save is possible.
func (cd *cmData) saveReason() string {
	if cd.readOnly != "" {
		return cd.readOnly
	}
	err := cd.d.sh.commandEnabled("configMap.saveData", configMapDataParams{Ref: cd.ref, ResourceVersion: "0", Data: map[string]string{}})
	cd.hidden = errors.Is(err, errUnavailable)
	if err != nil {
		return errText(err)
	}
	return ""
}

// refreshFooter makes the editors editable or read-only and sets Save, Revert and the status
// line from what the editors hold.
func (cd *cmData) refreshFooter() {
	reason := cd.reason
	for _, r := range cd.rows {
		if ro := !cd.loaded || reason != "" || !r.read || r.note.Visible(); ro != r.view.ReadOnly() {
			r.view.SetReadOnly(ro)
		}
	}
	if cd.hidden {
		cd.buttons.Hide()
	} else {
		cd.buttons.Show()
	}
	dirty := cd.dirty()
	enable(cd.save, cd.loaded && reason == "" && dirty && cd.review == nil)
	enable(cd.revert, dirty && cd.review == nil)
	switch {
	case !cd.loaded:
	case reason != "" && !cd.hidden:
		cd.setStatus("Read-only: "+reason, colorMuted)
	case dirty && cd.latest != cd.base:
		cd.setStatus(fmt.Sprintf("The Config Map changed on the cluster since your edits began (version %s, now %s): Save shows "+
			"your changes against the newer version first.", cd.base, cd.latest), colorStatusWarning)
	case dirty:
		if !strings.HasPrefix(cd.status.text, "Your unsaved changes are back") {
			cd.setStatus(fmt.Sprintf("Edited: %s, based on version %s.", valuesText(len(cd.edits())), cd.base), colorMuted)
		}
	case strings.HasPrefix(cd.status.text, "Saved"):
	default:
		cd.setStatus("", colorMuted)
	}
}

// setStatus shows text under the buttons; "" hides the line.
func (cd *cmData) setStatus(text string, c fyne.ThemeColorName) {
	cd.status.text, cd.status.color = text, c
	if text == "" {
		cd.status.Hide()
	} else {
		cd.status.Show()
	}
	cd.status.Refresh()
}

// revertAll puts the values back as the cluster holds them now.
func (cd *cmData) revertAll() {
	for _, r := range cd.rows {
		if r.read && r.dirty() {
			r.view.SetText(r.orig)
			r.resized()
		}
	}
	cd.base = cd.latest
	cd.setStatus("", colorMuted)
	cd.refreshFooter()
}

// saved follows a save: the values written are the rows' values now, based on the stored version.
func (cd *cmData) saved(values map[string]string, rv string) {
	for k, v := range values {
		if r := cd.rows[k]; r != nil {
			r.orig = v
		}
	}
	cd.base, cd.latest = rv, rv
	cd.setStatus(fmt.Sprintf("Saved %s · version %s", valuesText(len(values)), rv), colorStatusOK)
	cd.refreshFooter()
}

// close keeps the edited values when the drawer moves to another object or closes, and says so.
func (cd *cmData) close() {
	cd.closed = true
	if cd.review != nil {
		cd.review.hide()
	}
	values := cd.edits()
	if len(values) == 0 {
		return
	}
	if cd.d.unsaved == nil {
		cd.d.unsaved = map[string]*cmUnsaved{}
	}
	cd.d.unsaved[cd.ref.String()] = &cmUnsaved{base: cd.base, values: values}
	cd.d.sh.toast("info", fmt.Sprintf("Your unsaved changes to ConfigMap %s are kept until Cogpit quits: open it again to save or revert them.", cd.ref.Key()))
}

// dirty reports whether the row's text differs from the value read (line ends as the editor
// keeps them).
func (r *cmRow) dirty() bool { return r.view.Text() != normaliseNewlines(r.orig) }

// setNote shows a note in place of the editor for a value that is not shown, or the editor again.
func (r *cmRow) setNote(note string) {
	if note == "" {
		if r.note.Visible() {
			r.note.Hide()
			r.value.Objects = []fyne.CanvasObject{r.frame}
			r.value.Refresh()
		}
		return
	}
	r.note.SetText(note)
	r.note.Show()
	r.value.Objects = []fyne.CanvasObject{r.note}
	r.value.Refresh()
}

// editorHeight is the editor's height: the one the grip dragged, else its value's lines between
// cmMinLines and cmMaxLines.
func (r *cmRow) editorHeight() float32 {
	if r.height > 0 {
		return r.height
	}
	return r.view.HeightFor(min(max(r.view.LineCount(), cmMinLines), cmMaxLines), r.cd.width)
}

// drag changes the editor's height by dy, between one line and cmMaxDrag.
func (r *cmRow) drag(dy float32) {
	lo, hi := r.view.HeightFor(1, r.cd.width), r.view.HeightFor(cmMaxDrag, r.cd.width)
	r.height = min(max(r.editorHeight()+dy, lo), hi)
	r.resized()
}

// resized lays the list out again when the editor's height changed.
func (r *cmRow) resized() {
	if h := r.editorHeight(); h != r.shown {
		r.shown = h
		r.frame.Refresh()
		r.cd.list.Refresh()
	}
}

// cmFrameLayout lays out an editor's frame: the background and the editor fill it, as tall as the
// row's editorHeight; the grip sits in its bottom right corner.
type cmFrameLayout struct{ r *cmRow }

func (l cmFrameLayout) MinSize([]fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, l.r.editorHeight())
}

func (l cmFrameLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs[:2] {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
	g := objs[2].MinSize()
	objs[2].Move(fyne.NewPos(size.Width-g.Width, size.Height-g.Height))
	objs[2].Resize(g)
}

// cmGrip is the handle in an editor's bottom right corner: dragging it makes the editor taller or
// shorter.
type cmGrip struct {
	widget.BaseWidget
	r *cmRow
}

var (
	_ fyne.Draggable     = (*cmGrip)(nil)
	_ desktop.Cursorable = (*cmGrip)(nil)
)

func newCMGrip(r *cmRow) *cmGrip {
	g := &cmGrip{r: r}
	g.ExtendBaseWidget(g)
	return g
}

// Dragged implements fyne.Draggable.
func (g *cmGrip) Dragged(ev *fyne.DragEvent) { g.r.drag(ev.Dragged.DY) }

// DragEnd implements fyne.Draggable.
func (g *cmGrip) DragEnd() {}

// Cursor implements desktop.Cursorable.
func (g *cmGrip) Cursor() desktop.Cursor { return desktop.VResizeCursor }

// CreateRenderer implements fyne.Widget: two short diagonal strokes.
func (g *cmGrip) CreateRenderer() fyne.WidgetRenderer {
	r := &cmGripRenderer{g: g}
	for range 2 {
		l := canvas.NewLine(color.Transparent)
		l.StrokeWidth = 1
		r.lines = append(r.lines, l)
	}
	r.Refresh()
	return r
}

type cmGripRenderer struct {
	g     *cmGrip
	lines []*canvas.Line
}

func (r *cmGripRenderer) MinSize() fyne.Size { return fyne.NewSize(12, 12) }

func (r *cmGripRenderer) Layout(size fyne.Size) {
	for i, l := range r.lines {
		inset := float32(3 + 4*i)
		l.Position1 = fyne.NewPos(size.Width-2, inset)
		l.Position2 = fyne.NewPos(inset, size.Height-2)
	}
}

func (r *cmGripRenderer) Refresh() {
	c := themeColor(r.g.Theme(), colorMuted)
	for _, l := range r.lines {
		l.StrokeColor = c
		l.Refresh()
	}
	r.Layout(r.g.Size())
}

func (r *cmGripRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.lines[0], r.lines[1]}
}

func (r *cmGripRenderer) Destroy() {}

// cmReview is the review of a save: the diff of the edited values against the Config Map in the
// cluster and the server's dry run, then Save.
type cmReview struct {
	cd     *cmData
	values map[string]string
	dlg    *dialog.CustomDialog
	diff   *diffView
	banner *textLine
	status *paragraph
	save   *widget.Button
	over   string // the newer version Save writes the values over; "" none
	closed bool
}

// openReview reviews the edited values in a dialog over the window.
func (cd *cmData) openReview() {
	values := cd.edits()
	if len(values) == 0 || cd.review != nil {
		return
	}
	width := min(float32(760), cd.d.sh.win.Canvas().Size().Width-120)
	rv := &cmReview{cd: cd, values: values, diff: newDiffView()}
	rv.banner = newTextLine("Changes to ConfigMap "+cd.ref.Key(), theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	rv.banner.truncate = true
	rv.status = newParagraph("", colorMuted, width)
	rv.save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), rv.commit)
	rv.save.Importance = widget.HighImportance
	rv.save.Disable()
	cancel := widget.NewButton("Cancel", func() { rv.hide() })
	list := box(sizedMin{size: fyne.NewSize(width, 300)}, container.NewStack(newSurface(theme.ColorNameInputBackground), rv.diff.list))
	content := container.NewBorder(container.NewVBox(rv.banner, newDivider(false)), container.NewVBox(newDivider(false), rv.status), nil, nil, list)
	rv.dlg = dialog.NewCustomWithoutButtons("Save ConfigMap "+cd.ref.Name, content, cd.d.sh.win)
	rv.dlg.SetButtons([]fyne.CanvasObject{cancel, rv.save})
	rv.dlg.SetOnClosed(func() { rv.closed = true; cd.review = nil; cd.refreshFooter() })
	cd.review = rv
	cd.refreshFooter()
	rv.dlg.Show()
	rv.check()
}

// check asks for the diff and the server's dry run.
func (rv *cmReview) check() {
	cd := rv.cd
	rv.setStatus("Comparing with the Config Map in the cluster and checking with the server (dry run)…", colorMuted)
	cd.d.sh.dispatch("configMap.reviewData", configMapDataParams{Ref: cd.ref, ResourceVersion: cd.base, Data: rv.values}, func(res any, err error) {
		if rv.closed {
			return
		}
		var r engine.ConfigMapDataReview
		if err == nil {
			err = decode(res, &r)
		}
		switch {
		case err != nil:
			rv.setStatus("Could not review the change: "+errText(err), colorStatusError)
			return
		case !r.Changed:
			rv.hide()
			cd.setStatus("No changes to save: the values are the Config Map's.", colorMuted)
			return
		}
		rv.diff.set(r.Lines)
		switch {
		case r.Stale:
			rv.showOver(r.BaseResourceVersion, r.CachedResourceVersion)
		case r.Accepted:
			rv.banner.SetText(fmt.Sprintf("Changes to ConfigMap %s: %s, %d lines added, %d removed", cd.ref.Key(), valuesText(len(r.Keys)), r.Added, r.Removed))
			rv.setStatus("The server accepts the change (dry run). Save writes it to the cluster.", colorStatusOK)
			rv.save.Enable()
		default:
			rv.setStatus("The server refuses the change: "+r.Refused, colorStatusError)
		}
	})
}

// showOver shows the values against a newer version of the Config Map: Save then writes them over
// it, a second, explicit step.
func (rv *cmReview) showOver(base, current string) {
	if current == "" || current == base {
		rv.setStatus(fmt.Sprintf("The Config Map changed on the cluster since version %s and Cogpit has not seen the new version "+
			"yet; nothing was saved. Save again in a moment.", base), colorStatusWarning)
		rv.save.Enable()
		return
	}
	rv.over = current
	rv.banner.SetText(fmt.Sprintf("ConfigMap %s changed on the cluster since version %s: your changes against version %s", rv.cd.ref.Key(), base, current))
	rv.banner.SetColor(colorStatusWarning)
	rv.setStatus(fmt.Sprintf("Nothing was saved. Save over version %s writes your values over it; the cluster's other changes stay.", current), colorStatusWarning)
	rv.save.SetText("Save over version " + current)
	rv.save.Enable()
}

// commit saves the reviewed values, over the newer version when the review showed one.
func (rv *cmReview) commit() {
	cd := rv.cd
	if rv.over != "" {
		cd.base, rv.over = rv.over, ""
	}
	rv.save.Disable()
	rv.setStatus("Saving…", colorMuted)
	cd.d.sh.dispatch("configMap.saveData", configMapDataParams{Ref: cd.ref, ResourceVersion: cd.base, Data: rv.values}, func(res any, err error) {
		if rv.closed {
			return
		}
		var ce *engine.DataConflictError
		switch {
		case errors.As(err, &ce):
			rv.diff.set(ce.Diff)
			rv.showOver(ce.BaseResourceVersion, ce.CurrentResourceVersion)
			return
		case err != nil:
			rv.setStatus("Not saved: "+errText(err), colorStatusError)
			rv.save.Enable()
			return
		}
		var s engine.ConfigMapDataSaved
		_ = decode(res, &s)
		rv.hide()
		cd.saved(rv.values, s.ResourceVersion)
	})
}

func (rv *cmReview) setStatus(text string, c fyne.ThemeColorName) {
	rv.status.text, rv.status.color = text, c
	rv.status.Refresh()
}

// hide closes the review; the edited values stay in the editors.
func (rv *cmReview) hide() {
	if rv.closed {
		return
	}
	rv.dlg.Hide() // SetOnClosed marks it closed
}

// sizedMin gives its objects at least a size.
type sizedMin struct{ size fyne.Size }

func (l sizedMin) MinSize(objs []fyne.CanvasObject) fyne.Size {
	s := l.size
	for _, o := range objs {
		s = s.Max(o.MinSize())
	}
	return s
}

func (l sizedMin) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

// enable enables or disables a button.
func enable(b *widget.Button, on bool) {
	if on {
		b.Enable()
	} else {
		b.Disable()
	}
}

// valuesText writes a count of values: "1 value", "3 values".
func valuesText(n int) string { return fmt.Sprintf("%d value%s", n, plural(n)) }

// normaliseNewlines turns CRLF and CR line ends into LF, as the editor keeps them.
func normaliseNewlines(s string) string {
	if !strings.ContainsRune(s, '\r') {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
