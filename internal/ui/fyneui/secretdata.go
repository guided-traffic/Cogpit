package fyneui

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// A Secret's Data in the drawer, its values edited as a Config Map's are, in Cogpit's own window
// only (the owner's audit answer of 2026-10-15; decisions D-59). Each key stands on a line of its
// own with its value masked below and Show beside it, as before. Where the user may save the
// Secret's values (secret.saveData enabled: not the demo cluster, an update right on its Secrets,
// not immutable, no Helm release record), Show reads the whole value (secret.reveal with whole,
// GUI-only) into an editor below the key, the Config Map's editor row (configmapdata.go), and Hide
// beside the key masks it again; an edited value Hide keeps, masked, until it is saved or
// reverted. Save at the end of the data reviews every edited value at once in a dialog: a masked
// diff (the keys that change and their sizes, never a value) and the server's dry run
// (secret.reviewData), then secret.saveData writes them with the version they are based on as
// precondition. A Secret that changed meanwhile in other keys than the edited ones is reviewed
// rebased onto its newer version (the review shows it); one whose edited keys changed shows the
// conflict and "Save over version N", never an overwrite unasked. Where nothing can be saved Show
// reveals the value read-only as before (secretreveal.go), with the reason under the data.
//
// The values stay in this window's widgets and memory: no command but the GUI-only reveal, review
// and save sees them, no draft keeps them (edited values the drawer moves away from are kept in
// drawer.secretUnsaved until Cogpit quits, and come back masked), a copy from an editor goes to
// the clipboard as a Secret's value (concealed, cleared after 60 seconds), and no toast or status
// line names one.

func init() { drawerSections["secrets\x00Data"] = secretDataSection }

// secDataKept is what the drawer keeps across rebuilds of a Secret's Data.
const secDataKept = "secret.data"

// secretDataParams mirror secret.reviewData's and secret.saveData's parameters.
type secretDataParams struct {
	Ref             model.ObjectRef   `json:"ref"`
	ResourceVersion string            `json:"resourceVersion"`
	Data            map[string]string `json:"data"`
	From            map[string]string `json:"from,omitempty"`
}

// secUnsaved are a Secret's edited values the drawer moved away from, in the window's memory
// only: the version they are based on, each edited key's text and the value it is based on.
type secUnsaved struct {
	base         string
	values, from map[string]string
}

// secData is the Data section of the Secret the drawer shows.
type secData struct {
	d      *drawer
	ref    model.ObjectRef
	rows   map[string]*secRow // the keys whose values are in an editor
	list   *fyne.Container
	save   *widget.Button
	revert *widget.Button
	// buttons are Revert and Save; status the line under them; footer both.
	buttons *fyne.Container
	status  *paragraph
	root    *fyne.Container

	width    float32 // the details' width the editors are laid out at
	base     string  // the version the edited values are based on
	latest   string  // the version the drawer model lists the keys at
	reason   string  // why no value can be saved; "" when they can
	hidden   bool    // the save command is not in this build: no footer
	revealed bool    // a value is shown read-only (secretreveal.go)
	closed   bool
	review   *secReview
}

// secRow is a key whose value is in an editor.
type secRow struct {
	*cmRow
	at     string // the version the value was revealed at
	masked bool   // an edited value hidden again: the typing kept, the value masked
}

// secretDataSection renders a Secret's Data: the section kept for the shown Secret, rebuilt with
// every change of the drawer model.
func secretDataSection(d *drawer, ref model.ObjectRef, sec model.DetailSection) fyne.CanvasObject {
	sd, ok := d.kept[secDataKept].(*secData)
	if !ok || sd.ref != ref {
		sd = newSecData(d, ref)
		d.kept[secDataKept] = sd
	}
	sd.render(sec)
	return sd.root
}

func newSecData(d *drawer, ref model.ObjectRef) *secData {
	sd := &secData{d: d, ref: ref, rows: map[string]*secRow{}, width: d.contentWidth()}
	sd.list = box(keyedLayout{})
	sd.save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), sd.openReview)
	sd.save.Importance = widget.HighImportance
	sd.revert = widget.NewButton("Revert", sd.revertAll)
	sd.buttons = container.NewHBox(layout.NewSpacer(), sd.revert, sd.save)
	sd.status = newParagraph("", colorMuted, d.contentWidth())
	sd.root = container.NewVBox(sd.list, sd.buttons, sd.status)
	sd.restore()
	return sd
}

// restore brings back the edited values the drawer moved away from, masked: Show reveals them.
func (sd *secData) restore() {
	key := sd.ref.String()
	u := sd.d.secretUnsaved[key]
	if u == nil {
		return
	}
	delete(sd.d.secretUnsaved, key)
	for k, v := range u.values {
		r := sd.newRow(k, u.from[k], u.base)
		r.view.SetText(v)
		r.masked = true
	}
	sd.base = u.base
	sd.setStatus(fmt.Sprintf("Your unsaved changes are back, masked, based on version %s: Show reveals them, Save writes them, "+
		"Revert drops them.", u.base), colorStatusWarning)
}

// newRow makes the editor row of a key whose value was revealed at version at.
func (sd *secData) newRow(key, value, at string) *secRow {
	r := &secRow{cmRow: newValueRow(sd.d, sd, key, sd.edited), at: at}
	r.orig, r.from, r.read = value, value, true
	r.view.SetText(value)
	r.view.OnCopy = sd.copied
	sd.rows[key] = r
	return r
}

func (sd *secData) editorWidth() float32 { return sd.width }
func (sd *secData) relayout()            { sd.list.Refresh() }

// editable reports whether Show puts a value into an editor: the user may save it.
func (sd *secData) editable() bool { return !sd.hidden && sd.reason == "" }

// render lays out the keys: an editor below a key whose value is in one, else the drawer's masked
// value with Show (and, where nothing can be saved, a value Show revealed read-only).
func (sd *secData) render(sec model.DetailSection) {
	d := sd.d
	if w := d.contentWidth(); w != sd.width {
		sd.width = w
		sd.status.width = w
		sd.status.Refresh()
	}
	if v := sectionVersion(sec); v != "" && v != sd.latest {
		sd.latest = v
		sd.dropStale()
	}
	sd.reason = sd.saveReason()
	sd.revealed = false
	listed := map[string]bool{}
	objs := make([]fyne.CanvasObject, 0, 2*len(sec.Items))
	for _, it := range sec.Items {
		listed[it.Name] = true
		r := sd.rows[it.Name]
		switch {
		case it.Name == "" || r == nil:
			if _, shown := d.revealedItem(it); shown {
				sd.revealed = true
			}
			objs = append(objs, d.keyName(it.Name), d.itemValue(sd.ref, sec, it))
		case r.masked:
			objs = append(objs, d.keyName(it.Name), sd.maskedLine(r))
		default:
			objs = append(objs, sd.keyLine(r), r.frame)
		}
	}
	for _, k := range slices.Sorted(maps.Keys(sd.rows)) { // an edited key the cluster dropped: its save is refused
		if r := sd.rows[k]; !listed[k] && r.dirty() {
			objs = append(objs, sd.keyLine(r), r.frame)
		}
	}
	sd.list.Objects = objs
	sd.list.Refresh()
	sd.refreshFooter()
}

// sectionVersion is the Secret's version the drawer model lists its keys at: what its reveal
// actions name.
func sectionVersion(sec model.DetailSection) string {
	for _, it := range sec.Items {
		for _, a := range it.Actions {
			if a.Command != engine.RevealCommand {
				continue
			}
			var p engine.RevealParams
			if json.Unmarshal(a.Params, &p) == nil && p.ResourceVersion != "" {
				return p.ResourceVersion
			}
		}
	}
	return ""
}

// dropStale masks again the values revealed at an older version that the user did not edit: the
// Secret changed (a rotation), and an old value must not stay to be copied (R-6). Edited values
// stay, based on the version they were edited at.
func (sd *secData) dropStale() {
	for k, r := range sd.rows {
		if r.at != sd.latest && !r.dirty() {
			delete(sd.rows, k)
		}
	}
}

// keyLine is a key above its editor, with Hide at its end.
func (sd *secData) keyLine(r *secRow) fyne.CanvasObject {
	hide := widget.NewButton("Hide", func() { sd.hide(r.key) })
	hide.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, nil, hide, sd.d.keyName(r.key))
}

// maskedLine is an edited value hidden again: masked, with Show to bring the editor back.
func (sd *secData) maskedLine(r *secRow) fyne.CanvasObject {
	show := widget.NewButton("Show", func() { sd.unmask(r.key) })
	show.Importance = widget.LowImportance
	text := newTextLine(engine.MaskedValue+" (edited, not saved)", colorStatusWarning, theme.SizeNameText, fyne.TextStyle{})
	return container.NewBorder(nil, nil, nil, show, text)
}

// reveal reads a key's whole value into an editor (Show on a Secret the user may save). The answer
// of an older listing, or one for a drawer that moved on, is dropped.
func (sd *secData) reveal(a model.ItemAction) {
	var p engine.RevealParams
	if err := json.Unmarshal(a.Params, &p); err != nil {
		sd.d.sh.showError(err)
		return
	}
	p.Whole = true
	sd.d.sh.dispatch(a.Command, p, func(res any, err error) {
		if err != nil {
			sd.d.sh.showError(err)
			return
		}
		var v engine.SecretValue
		if decode(res, &v) != nil || sd.closed || sd.d.shown == nil || *sd.d.shown != sd.ref || p.ResourceVersion != sd.latest {
			return // the drawer moved on, or the Secret changed meanwhile: keep nothing
		}
		if _, ok := sd.rows[p.Key]; !ok && !v.Cut {
			sd.newRow(p.Key, v.Value, p.ResourceVersion)
		}
		sd.d.rebuild()
	})
}

// hide masks a value again: an unedited one leaves the window's memory, an edited one stays,
// masked, until it is saved or reverted.
func (sd *secData) hide(key string) {
	r := sd.rows[key]
	if r == nil {
		return
	}
	if r.dirty() {
		r.masked = true
	} else {
		delete(sd.rows, key)
	}
	sd.d.rebuild()
}

// unmask shows an edited value that Hide masked.
func (sd *secData) unmask(key string) {
	if r := sd.rows[key]; r != nil {
		r.masked = false
		sd.d.rebuild()
	}
}

// edited follows the user's typing: the editor's height, the version the edits are based on (the
// first edited value's), the footer.
func (sd *secData) edited(cr *cmRow) {
	cr.resized()
	if r := sd.rows[cr.key]; r != nil && sd.dirtyBesides(cr.key) == 0 && cr.dirty() {
		sd.base = r.at
	}
	sd.refreshFooter()
}

// dirtyBesides counts the edited values other than key's.
func (sd *secData) dirtyBesides(key string) int {
	n := 0
	for k, r := range sd.rows {
		if k != key && r.dirty() {
			n++
		}
	}
	return n
}

// dirty reports whether a value was edited.
func (sd *secData) dirty() bool { return sd.dirtyBesides("") > 0 }

// edits are the edited values by key, and froms the values they are based on.
func (sd *secData) edits() (values, from map[string]string) {
	values, from = map[string]string{}, map[string]string{}
	for k, r := range sd.rows {
		if r.dirty() {
			values[k], from[k] = r.view.Text(), r.from
		}
	}
	return values, from
}

// saveReason says why no value can be saved: the command's reason (the demo cluster, no update
// permission, an immutable Secret, a Helm release record); "" when Save is possible.
func (sd *secData) saveReason() string {
	err := sd.d.sh.commandEnabled(engine.SecretSaveDataCommand, secretDataParams{Ref: sd.ref, ResourceVersion: "0", Data: map[string]string{}})
	sd.hidden = errors.Is(err, errUnavailable)
	if err != nil {
		return errText(err)
	}
	return ""
}

// refreshFooter makes the editors editable or read-only and sets Save, Revert and the status line.
func (sd *secData) refreshFooter() {
	editable := sd.editable()
	for _, r := range sd.rows {
		if ro := !editable; ro != r.view.ReadOnly() {
			r.view.SetReadOnly(ro)
		}
	}
	dirty := sd.dirty()
	showButtons := !sd.hidden && (len(sd.rows) > 0 || dirty)
	setVisible(sd.buttons, showButtons)
	enable(sd.save, editable && dirty && sd.review == nil)
	enable(sd.revert, dirty && sd.review == nil)
	switch {
	case sd.hidden:
		sd.setStatus("", colorMuted)
	case sd.reason != "" && (sd.revealed || len(sd.rows) > 0):
		sd.setStatus("Read-only: "+sd.reason, colorMuted)
	case dirty && sd.latest != sd.base:
		sd.setStatus(fmt.Sprintf("The Secret changed on the cluster since your edits began (version %s, now %s): Save rebases "+
			"your changes onto the newer version when none of your keys changed there, else shows them against it first.",
			sd.base, sd.latest), colorStatusWarning)
	case dirty:
		if !strings.HasPrefix(sd.status.text, "Your unsaved changes are back") {
			values, _ := sd.edits()
			sd.setStatus(fmt.Sprintf("Edited: %s, based on version %s. Save shows the keys that change and their sizes, never a value.",
				valuesText(len(values)), sd.base), colorMuted)
		}
	case strings.HasPrefix(sd.status.text, "Saved"):
	default:
		sd.setStatus("", colorMuted)
	}
}

// setVisible shows or hides o.
func setVisible(o fyne.CanvasObject, on bool) {
	if on && !o.Visible() {
		o.Show()
	} else if !on && o.Visible() {
		o.Hide()
	}
}

// setStatus shows text under the buttons; "" hides the line.
func (sd *secData) setStatus(text string, c fyne.ThemeColorName) {
	sd.status.text, sd.status.color = text, c
	setVisible(sd.status, text != "")
	sd.status.Refresh()
}

// revertAll puts the values back: an edited value revealed at the version the drawer lists goes
// back to it, one revealed at an older version is masked again.
func (sd *secData) revertAll() {
	for k, r := range sd.rows {
		switch {
		case !r.dirty() && !r.masked:
		case r.at == sd.latest && !r.masked:
			r.from = r.orig
			r.view.SetText(r.orig)
			r.resized()
		default:
			delete(sd.rows, k)
		}
	}
	sd.base = ""
	sd.setStatus("", colorMuted)
	sd.d.rebuild()
}

// saved follows a save: the values written are the editors' values now, at the stored version.
func (sd *secData) saved(values map[string]string, rv string) {
	for k, v := range values {
		if r := sd.rows[k]; r != nil {
			r.orig, r.from, r.at = v, v, rv
		}
	}
	sd.base = ""
	sd.setStatus(fmt.Sprintf("Saved %s · version %s", valuesText(len(values)), rv), colorStatusOK)
	sd.refreshFooter()
}

// copied puts text an editor copied on the clipboard as a Secret's value: concealed for clipboard
// managers, cleared after a minute if the clipboard still holds it (D-25, H-25).
func (sd *secData) copied(text string) {
	sh := sd.d.sh
	sh.copyText(text, true)
	sh.copyToast("info", fmt.Sprintf("Copied. The clipboard is cleared in %s.", ttlText(sh.secretClipTTL)))
}

// close keeps the edited values in the window's memory when the drawer moves to another object or
// closes, and says so; a Secret's values never go to disk.
func (sd *secData) close() {
	if sd.review != nil {
		sd.review.hide()
	}
	sd.closed = true
	values, from := sd.edits()
	if len(values) == 0 {
		return
	}
	if sd.d.secretUnsaved == nil {
		sd.d.secretUnsaved = map[string]*secUnsaved{}
	}
	sd.d.secretUnsaved[sd.ref.String()] = &secUnsaved{base: sd.base, values: values, from: from}
	sd.d.sh.toast("info", fmt.Sprintf("Your unsaved changes to Secret %s are kept in this window's memory until Cogpit quits, never "+
		"on disk: open it again to save or revert them.", sd.ref.Key()))
}

// secretDataOf returns the Data section the drawer keeps for the Secret ref; nil when it shows
// another object.
func (d *drawer) secretDataOf(ref model.ObjectRef) *secData {
	sd, ok := d.kept[secDataKept].(*secData)
	if !ok || sd.ref != ref || sd.closed {
		return nil
	}
	return sd
}

// secReview is the review of a Secret's save: the masked diff and the server's dry run, then Save.
type secReview struct {
	sd           *secData
	values, from map[string]string
	base         string // the version the shown changes and dry run are of, which Save writes with
	dlg          *dialog.CustomDialog
	changes      *fyne.Container // one line per key that changes
	banner       *textLine
	status       *paragraph
	save         *widget.Button
	over         string // the newer version Save writes the values over; "" none
	closed       bool
}

// openReview reviews the edited values in a dialog over the window.
func (sd *secData) openReview() {
	values, from := sd.edits()
	if len(values) == 0 || sd.review != nil {
		return
	}
	width := min(float32(620), sd.d.sh.win.Canvas().Size().Width-120)
	rv := &secReview{sd: sd, values: values, from: from, base: sd.base, changes: container.NewVBox()}
	rv.banner = newTextLine("Changes to Secret "+sd.ref.Key(), theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	rv.banner.truncate = true
	rv.status = newParagraph("", colorMuted, width)
	rv.save = widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), rv.commit)
	rv.save.Importance = widget.HighImportance
	cancel := widget.NewButton("Cancel", func() { rv.hide() })
	list := box(sizedMin{size: fyne.NewSize(width, 120)}, container.NewStack(newSurface(theme.ColorNameInputBackground),
		inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, container.NewVScroll(rv.changes))))
	content := container.NewBorder(container.NewVBox(rv.banner, newDivider(false)), container.NewVBox(newDivider(false), rv.status), nil, nil, list)
	rv.dlg = dialog.NewCustomWithoutButtons("Save Secret "+sd.ref.Name, content, sd.d.sh.win)
	rv.dlg.SetButtons([]fyne.CanvasObject{cancel, rv.save})
	rv.dlg.SetOnClosed(func() { rv.closed = true; sd.review = nil; sd.refreshFooter() })
	sd.review = rv
	sd.refreshFooter()
	rv.dlg.Show()
	rv.check()
}

// setChanges shows the masked diff: each key, what changes and the sizes, never a value.
func (rv *secReview) setChanges(changes []engine.SecretKeyChange) {
	objs := make([]fyne.CanvasObject, 0, len(changes))
	for _, c := range changes {
		var what string
		switch c.Change {
		case engine.SecretKeyAdded:
			what = "added, " + byteText(c.NewBytes)
		case engine.SecretKeyRemoved:
			what = "removed, " + byteText(c.Bytes)
		default:
			what = "changed, " + byteText(c.Bytes) + " → " + byteText(c.NewBytes)
		}
		key := newTextLine(c.Key, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Monospace: true})
		key.truncate = true
		objs = append(objs, container.NewBorder(nil, nil, nil,
			newTextLine(what+"  "+engine.MaskedValue, colorMuted, theme.SizeNameText, fyne.TextStyle{}), key))
	}
	rv.changes.Objects = objs
	rv.changes.Refresh()
}

// check asks for the masked diff and the server's dry run, based on the version the review is
// of. A Secret that changed since in other keys than the edited ones comes back rebased onto its
// newer version, which the review is of from then on.
func (rv *secReview) check() {
	sd := rv.sd
	rv.over = ""
	rv.save.SetText("Save")
	rv.save.Disable()
	rv.banner.SetColor(theme.ColorNameForeground)
	rv.setStatus("Comparing with the Secret in the cluster and checking with the server (dry run)…", colorMuted)
	params := secretDataParams{Ref: sd.ref, ResourceVersion: rv.base, Data: rv.values, From: rv.from}
	sd.d.sh.dispatch(engine.SecretReviewDataCommand, params, func(res any, err error) {
		if rv.closed {
			return
		}
		var r engine.SecretDataReview
		if err == nil {
			err = decode(res, &r)
		}
		switch {
		case err != nil:
			rv.setStatus("Could not review the change: "+errText(err), colorStatusError)
			return
		case len(r.Changes) == 0:
			rv.hide()
			sd.setStatus("No changes to save: the values are the Secret's.", colorMuted)
			return
		}
		rv.setChanges(r.Changes)
		rebased := ""
		if r.RebasedFrom != "" && !r.Stale {
			rv.base, sd.base = r.BaseResourceVersion, r.BaseResourceVersion
			rebased = fmt.Sprintf("The Secret changed on the cluster since version %s, but none of the keys you edited did: your "+
				"changes are rebased onto version %s, and these are their changes against it. ", r.RebasedFrom, r.BaseResourceVersion)
		}
		switch {
		case r.Stale:
			rv.showOver(r.BaseResourceVersion, r.CachedResourceVersion)
		case r.Accepted:
			title := fmt.Sprintf("Changes to Secret %s: %s", sd.ref.Key(), valuesText(len(r.Changes)))
			if rebased != "" {
				title += ", rebased onto version " + r.BaseResourceVersion
			}
			rv.banner.SetText(title)
			rv.setStatus(rebased+"The server accepts the change (dry run). Save writes it to the cluster; no value is shown here.", colorStatusOK)
			rv.save.Enable()
		default:
			rv.setStatus(rebased+"The server refuses the change: "+r.Refused, colorStatusError)
		}
		sd.refreshFooter()
	})
}

// showOver shows the changes against a newer version of the Secret: Save then writes the values
// over it, a second, explicit step.
func (rv *secReview) showOver(base, current string) {
	if current == "" || current == base {
		rv.setStatus(fmt.Sprintf("The Secret changed on the cluster since version %s and Cogpit has not seen the new version yet; "+
			"nothing was saved. Save again in a moment.", base), colorStatusWarning)
		rv.save.Enable()
		return
	}
	rv.over = current
	rv.banner.SetText(fmt.Sprintf("Secret %s changed on the cluster since version %s: your changes against version %s", rv.sd.ref.Key(), base, current))
	rv.banner.SetColor(colorStatusWarning)
	rv.setStatus(fmt.Sprintf("Nothing was saved: a key you edited changed there too. Save over version %s writes your values over "+
		"it; the cluster's other changes stay.", current), colorStatusWarning)
	rv.save.SetText("Save over version " + current)
	rv.save.Enable()
}

// commit saves the reviewed values, over the newer version when the review showed one; a save
// that meets a still newer version reviews the values again and writes nothing until Save.
func (rv *secReview) commit() {
	sd := rv.sd
	if rv.over != "" {
		rv.base, sd.base, rv.over = rv.over, rv.over, ""
	}
	rv.save.Disable()
	rv.setStatus("Saving…", colorMuted)
	sd.d.sh.dispatch(engine.SecretSaveDataCommand, secretDataParams{Ref: sd.ref, ResourceVersion: rv.base, Data: rv.values}, func(res any, err error) {
		if rv.closed {
			return
		}
		var ce *engine.DataConflictError
		switch {
		case errors.As(err, &ce):
			rv.check()
			return
		case err != nil:
			rv.setStatus("Not saved: "+errText(err), colorStatusError)
			rv.save.Enable()
			return
		}
		var s engine.SecretDataSaved
		_ = decode(res, &s)
		rv.hide()
		sd.saved(rv.values, s.ResourceVersion)
	})
}

func (rv *secReview) setStatus(text string, c fyne.ThemeColorName) {
	rv.status.text, rv.status.color = text, c
	rv.status.Refresh()
}

// hide closes the review; the edited values stay in the editors.
func (rv *secReview) hide() {
	if rv.closed {
		return
	}
	rv.dlg.Hide() // SetOnClosed marks it closed
}
