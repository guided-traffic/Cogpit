package fyneui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// The kubeconfig sources in the GUI (G2): the Cluster menu's and the picker's "Add Kubeconfigs"
// items, the sources dialog, and the dialog that shows a credential plugin before
// its first run (trust-boundaries H-1). Every change goes through the engine's kubeconfig commands,
// which only the GUI may run; the dialogs here are where the user does it. Lens's own dialogs are
// not captured: the layout is Cogpit's, the labels follow Lens's documentation where it names them
// ("Add Kubeconfigs").

// Result types of the kubeconfig commands, mirroring the engine's JSON.
type (
	sourceStatus struct {
		ID       string   `json:"id"`
		Kind     string   `json:"kind"`
		Path     string   `json:"path"`
		Files    []string `json:"files"`
		Contexts int      `json:"contexts"`
		Skipped  int      `json:"skipped"`
		Error    string   `json:"error"`
	}
	sourceAdded struct {
		Source   sourceStatus `json:"source"`
		Contexts []string     `json:"contexts"`
	}
	execPluginView struct {
		Context     string `json:"context"`
		Source      string `json:"source"`
		State       string `json:"state"`
		Kind        string `json:"kind"`
		Fingerprint string `json:"fingerprint"`
		Plugin      struct {
			File       string   `json:"file"`
			Context    string   `json:"context"`
			Server     string   `json:"server"`
			APIVersion string   `json:"apiVersion"`
			Command    string   `json:"command"`
			Args       []string `json:"args"`
			EnvNames   []string `json:"envNames"`
			TokenFile  string   `json:"tokenFile"`
			// TLS and Files mirror kubeconfig.TLSView and kubeconfig.FileFacts: how the credential
			// travels, and the files whose content the approval binds (R-11, R-5).
			TLS   kubeconfig.TLSView   `json:"tls"`
			Files kubeconfig.FileFacts `json:"files"`
		} `json:"plugin"`
		Found        string   `json:"found"`
		RiskyEnv     []string `json:"riskyEnv"`
		ArgsWithheld int      `json:"argsWithheld"`
	}
	pathParams struct {
		Path string `json:"path"`
	}
	sourceParams struct {
		Source string `json:"source"`
	}
	approveParams struct {
		Context     string `json:"context"`
		Fingerprint string `json:"fingerprint"`
	}
)

// Display limits for text from a kubeconfig: one line each, cut with an ellipsis.
const (
	maxShownPath = 400
	maxShownArg  = 300
	maxShownArgs = 40
)

// kubeconfigUI holds the kubeconfig dialogs of one shell.
type kubeconfigUI struct {
	sh *shell
	// pick asks for a file (folder false) or a folder and calls done with its path; Fyne's file
	// dialogs in the app, a recorder in tests.
	pick func(folder bool, done func(path string))

	approval *approvalDialog // the open approval dialog; main goroutine only
	sources  *sourcesDialog  // the open sources dialog; main goroutine only
}

func newKubeconfigUI(sh *shell) *kubeconfigUI {
	k := &kubeconfigUI{sh: sh}
	k.pick = k.pickWithDialog
	return k
}

// menuItems are the Cluster menu's kubeconfig items.
func (k *kubeconfigUI) menuItems() []*fyne.MenuItem {
	return []*fyne.MenuItem{
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Add Kubeconfig File…", k.addFile),
		fyne.NewMenuItem("Add Sync Folder…", k.addFolder),
		fyne.NewMenuItem("Kubeconfig Sources…", k.showSources),
	}
}

// addMenu is the picker's "Add Kubeconfigs" menu.
func (k *kubeconfigUI) addMenu() *fyne.Menu {
	return fyne.NewMenu("",
		fyne.NewMenuItem("Add kubeconfig file…", k.addFile),
		fyne.NewMenuItem("Add sync folder…", k.addFolder),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Manage kubeconfig sources…", k.showSources),
	)
}

// pickWithDialog shows Fyne's file or folder dialog.
func (k *kubeconfigUI) pickWithDialog(folder bool, done func(string)) {
	if folder {
		d := dialog.NewFolderOpen(func(u fyne.ListableURI, err error) {
			if err != nil {
				k.sh.toast("error", "Could not open the folder: "+err.Error())
				return
			}
			if u != nil {
				done(u.Path())
			}
		}, k.sh.win)
		d.Show()
		return
	}
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			k.sh.toast("error", "Could not open the file: "+err.Error())
			return
		}
		if r == nil {
			return // cancelled
		}
		path := r.URI().Path()
		_ = r.Close() // the engine reads the file itself
		done(path)
	}, k.sh.win)
	d.Show()
}

// addFile asks for a kubeconfig file and adds it.
func (k *kubeconfigUI) addFile() {
	k.pick(false, func(path string) { k.addPath("kubeconfig.addFile", path) })
}

// addFolder asks for a sync folder and adds it.
func (k *kubeconfigUI) addFolder() {
	k.pick(true, func(path string) { k.addPath("kubeconfig.addFolder", path) })
}

func (k *kubeconfigUI) addPath(id, path string) {
	k.sh.dispatch(id, pathParams{Path: path}, func(res any, err error) {
		if err != nil {
			k.sh.showError(err)
			return
		}
		k.announce(res)
	})
}

// announce tells the user what a new source offers and refreshes what lists contexts.
func (k *kubeconfigUI) announce(res any) {
	var a sourceAdded
	if err := decode(res, &a); err != nil {
		return
	}
	n := len(a.Contexts)
	k.sh.toast("info", fmt.Sprintf("Added %s: %d %s", kubeconfig.DisplayText(a.Source.ID, maxShownPath), n, nounFor(n, "context", "contexts")))
	k.contextsChanged()
}

// contextsChanged refreshes the picker and the sources dialog if they are open.
func (k *kubeconfigUI) contextsChanged() {
	if k.sh.picker.visible() {
		k.sh.picker.load()
	}
	if k.sources != nil {
		k.sources.load()
	}
}

func nounFor(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// errorText is the user-facing text of a dispatch error.
func errorText(err error) string {
	var dis *engine.DisabledError
	if errors.As(err, &dis) {
		return dis.Reason
	}
	return err.Error()
}

// reviewPlugin shows the credential plugin of a source context and asks whether it may run. done
// gets true once the user allowed it and the engine recorded that; false when the user cancelled
// or the plugin could not be shown.
func (k *kubeconfigUI) reviewPlugin(contextID string, done func(allowed bool)) {
	k.sh.dispatch("kubeconfig.execPlugin", map[string]string{"context": contextID}, func(res any, err error) {
		if err != nil {
			k.sh.showError(err)
			done(false)
			return
		}
		var info execPluginView
		if err := decode(res, &info); err != nil {
			k.sh.toast("error", "Unexpected credential plugin description: "+err.Error())
			done(false)
			return
		}
		if info.State == engine.ExecApproved {
			done(true)
			return
		}
		k.showApproval(info, done)
	})
}

// approvalDialog shows a credential plugin before its first run.
type approvalDialog struct {
	d      *dialog.CustomDialog
	allow  *widget.Button
	cancel *widget.Button
	info   execPluginView
}

// showApproval builds and shows the approval dialog for info.
func (k *kubeconfigUI) showApproval(info execPluginView, done func(bool)) {
	if k.approval != nil {
		k.approval.d.Hide()
	}
	th := k.sh.app.Settings().Theme()
	width := th.Size(sizePickerWidth)
	valueW := width - th.Size(sizeDrawerLabel) - 2*th.Size(theme.SizeNamePadding)
	pl := info.Plugin
	show := func(s string) string { return kubeconfig.DisplayText(s, maxShownPath) }

	rows := box(kvLayout{})
	addRow := func(name string, value fyne.CanvasObject) {
		rows.Objects = append(rows.Objects, newTextLine(name, colorMuted, theme.SizeNameText, fyne.TextStyle{}), value)
	}
	mono := func(s string) fyne.CanvasObject {
		l := newTextLine(s, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Monospace: true})
		l.truncate = true
		return l
	}
	hasProgram := info.Kind != kubeconfig.LoginTokenFile
	if hasProgram {
		addRow("Program", mono(show(pl.Command)))
		if t := programFileText(info); t != "" {
			addRow("", newParagraph(t, theme.ColorNameForeground, valueW))
		}
		switch {
		case info.Found == "":
			addRow("Found", newParagraph("not found on Cogpit's PATH now: connecting would fail", colorStatusWarning, valueW))
		case info.Found != pl.Command:
			addRow("Found at", mono(show(info.Found)))
		}
		args := container.NewVBox()
		for i, a := range pl.Args {
			if i == maxShownArgs {
				args.Add(newTextLine(fmt.Sprintf("… and %d more", len(pl.Args)-maxShownArgs), colorMuted, theme.SizeNameText, fyne.TextStyle{}))
				break
			}
			args.Add(mono(kubeconfig.DisplayText(a, maxShownArg)))
		}
		if len(pl.Args) == 0 {
			args.Add(newTextLine("none", colorMuted, theme.SizeNameText, fyne.TextStyle{}))
		}
		addRow("Arguments", args)
		env := "none"
		if len(pl.EnvNames) > 0 {
			env = joinNames(pl.EnvNames) + " (values not shown)"
		}
		addRow("Environment", newParagraph(env, theme.ColorNameForeground, valueW))
		if len(info.RiskyEnv) > 0 {
			addRow("", newParagraph("Sets "+strings.Join(info.RiskyEnv, ", ")+", which decides which programs run or what they load.",
				colorStatusWarning, valueW))
		}
	}
	if info.Kind == kubeconfig.LoginTokenFile || info.Kind == kubeconfig.LoginBoth {
		addRow("Token file", mono(show(pl.TokenFile)))
		if pl.Files.TokenFileResolved != "" {
			addRow("", newParagraph("a link in the kubeconfig's folder, pointing to "+show(pl.Files.TokenFileResolved), colorMuted, valueW))
		}
	}
	addRow("Credential for", mono(show(pl.Server)))
	facts := connectionFacts(info)
	for _, r := range facts.rows {
		addRow(r[0], newParagraph(r[1], theme.ColorNameForeground, valueW))
	}
	addRow("Kubeconfig", newParagraph(show(pl.File), theme.ColorNameForeground, valueW))
	addRow("Context", newParagraph(show(pl.Context)+" ("+show(info.Source)+")", theme.ColorNameForeground, valueW))

	content := container.NewVBox(newParagraph(approvalLead(info), theme.ColorNameForeground, width), newSpace(theme.SizeNamePadding))
	if info.State == engine.ExecChanged {
		content.Add(newParagraph(approvalChangedNote(info), colorStatusWarning, width))
		content.Add(newSpace(theme.SizeNamePadding))
	}
	// An insecure, proxied or own-CA connection decides who can read the credential: said before
	// the rows, in the warning colour, not as one more row (R-11).
	for _, w := range facts.warnings {
		content.Add(newParagraph(w, colorStatusWarning, width))
		content.Add(newSpace(theme.SizeNamePadding))
	}
	content.Add(rows)

	a := &approvalDialog{info: info}
	a.allow = widget.NewButtonWithIcon("Allow and Connect", theme.ConfirmIcon(), nil)
	a.allow.Importance = widget.WarningImportance
	a.cancel = widget.NewButton("Cancel", nil)
	a.d = dialog.NewCustomWithoutButtons(approvalTitle(info), content, k.sh.win)
	a.d.SetButtons([]fyne.CanvasObject{a.cancel, a.allow})
	answered := false
	a.d.SetOnClosed(func() {
		if k.approval == a {
			k.approval = nil
		}
		if !answered {
			answered = true
			done(false)
		}
	})
	a.cancel.OnTapped = a.d.Hide
	a.allow.OnTapped = func() {
		a.allow.Disable()
		k.sh.dispatch("kubeconfig.approveExec", approveParams{Context: info.Context, Fingerprint: info.Fingerprint}, func(_ any, err error) {
			if err != nil {
				k.sh.showError(err) // a stale fingerprint: the dialog closes, connecting again shows the new form
				a.d.Hide()
				return
			}
			answered = true
			a.d.Hide()
			done(true)
		})
	}
	// As the confirmation's: a click meant for something else, or a double click, must not allow.
	if delay := k.sh.confirmer.armDelay; delay > 0 {
		a.allow.Disable()
		time.AfterFunc(delay, func() { k.sh.do(a.allow.Enable) })
	}
	k.approval = a
	a.d.Show() // no button has the focus: Enter must not allow a program by accident
}

// sourcesDialog lists the kubeconfig sources with a Remove button each.
type sourcesDialog struct {
	k       *kubeconfigUI
	d       *dialog.CustomDialog
	list    *fyne.Container
	status  *textLine
	removeB map[string]*widget.Button // by source id, for tests
	rows    []sourceStatus
}

// showSources opens the sources dialog.
func (k *kubeconfigUI) showSources() {
	if k.sources != nil {
		k.sources.load()
		return
	}
	sd := &sourcesDialog{k: k, list: container.NewVBox(), removeB: map[string]*widget.Button{}}
	sd.status = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	sd.status.truncate = true
	intro := newParagraph("Kubeconfig files and sync folders Cogpit reads (and never writes). Your default kubeconfig ($KUBECONFIG or ~/.kube/config) is "+
		"read as well and is not listed here.", colorMuted, k.sh.app.Settings().Theme().Size(sizePickerWidth))
	listBg := newSurface(theme.ColorNameInputBackground)
	listBg.radius, listBg.stroke = theme.SizeNameInputRadius, theme.ColorNameInputBorder
	list := container.NewStack(listBg, inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewVScroll(sd.list)))
	content := sized(sizePickerWidth, sizePickerHeight, container.NewBorder(
		container.NewVBox(intro, newSpace(theme.SizeNamePadding)), sd.status, nil, nil, list))
	sd.d = dialog.NewCustomWithoutButtons("Kubeconfig Sources", content, k.sh.win)
	addB := widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), nil)
	addB.OnTapped = func() {
		widget.ShowPopUpMenuAtRelativePosition(fyne.NewMenu("",
			fyne.NewMenuItem("Add kubeconfig file…", k.addFile),
			fyne.NewMenuItem("Add sync folder…", k.addFolder),
		), k.sh.win.Canvas(), fyne.NewPos(0, addB.Size().Height), addB)
	}
	closeB := widget.NewButton("Close", nil)
	sd.d.SetButtons([]fyne.CanvasObject{addB, closeB})
	closeB.OnTapped = sd.d.Hide
	sd.d.SetOnClosed(func() {
		if k.sources == sd {
			k.sources = nil
		}
	})
	k.sources = sd
	sd.d.Show()
	sd.load()
}

// load reads the sources off the main goroutine and shows them.
func (sd *sourcesDialog) load() {
	sd.k.sh.dispatch("kubeconfig.sources", struct{}{}, func(res any, err error) {
		if err != nil {
			sd.status.SetColor(colorStatusError)
			sd.status.SetText(errorText(err))
			return
		}
		var out struct {
			Sources []sourceStatus `json:"sources"`
		}
		if err := decode(res, &out); err != nil {
			sd.status.SetColor(colorStatusError)
			sd.status.SetText("Unexpected result: " + err.Error())
			return
		}
		sd.rows = out.Sources
		sd.render()
	})
}

func (sd *sourcesDialog) render() {
	th := sd.k.sh.app.Settings().Theme()
	width := th.Size(sizePickerWidth) - 8*th.Size(theme.SizeNamePadding)
	sd.removeB = map[string]*widget.Button{}
	objs := make([]fyne.CanvasObject, 0, len(sd.rows))
	for _, s := range sd.rows {
		src := s
		title := newTextLine(kubeconfig.DisplayText(src.ID, maxShownPath), theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
		kind := newPill(sourceKindLabel(src.Kind), colorAccent, true)
		path := newTextLine(kubeconfig.DisplayText(src.Path, maxShownPath), colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
		path.truncate = true
		remove := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() { sd.confirmRemove(src) })
		remove.Importance = widget.LowImportance
		sd.removeB[src.ID] = remove
		var info fyne.CanvasObject
		if src.Error != "" { // wrapped beside the button, never wider than the list
			info = newParagraph(src.Error, colorStatusError, width-remove.MinSize().Width-2*th.Size(theme.SizeNamePadding))
		} else {
			info = newTextLine(sourceSummary(src), colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
		}
		row := inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, container.NewBorder(nil, nil, nil,
			container.NewCenter(remove), container.NewVBox(container.NewHBox(title, kind), path, info)))
		objs = append(objs, row)
	}
	if len(objs) == 0 {
		objs = append(objs, inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
			newTextLine("No sources added yet.", colorMuted, theme.SizeNameText, fyne.TextStyle{})))
	}
	sd.list.Objects = objs
	sd.list.Refresh()
	sd.status.SetColor(colorMuted)
	sd.status.SetText(fmt.Sprintf("%d %s", len(sd.rows), nounFor(len(sd.rows), "source", "sources")))
}

func sourceKindLabel(kind string) string {
	switch kind {
	case string(engine.SourceFolder):
		return "sync folder"
	}
	return "file"
}

func sourceSummary(s sourceStatus) string {
	parts := []string{fmt.Sprintf("%d %s", s.Contexts, nounFor(s.Contexts, "context", "contexts"))}
	if s.Kind == string(engine.SourceFolder) {
		parts = append([]string{fmt.Sprintf("%d %s", len(s.Files), nounFor(len(s.Files), "kubeconfig", "kubeconfigs"))}, parts...)
		if s.Skipped > 0 {
			parts = append(parts, fmt.Sprintf("%d other %s skipped", s.Skipped, nounFor(s.Skipped, "file", "files")))
		}
	}
	return strings.Join(parts, " · ")
}

// confirmRemove asks before removing a source.
func (sd *sourcesDialog) confirmRemove(src sourceStatus) {
	msg := "Its contexts are no longer offered, and the login programs you allowed for it are forgotten. The " +
		"file stays as it is."
	switch src.Kind {
	case string(engine.SourceFolder):
		msg = "Its kubeconfigs are no longer offered, and the login programs you allowed for them are forgotten. " +
			"The folder and its files stay as they are."
	}
	dialog.ShowConfirm("Remove "+kubeconfig.DisplayText(src.ID, maxShownPath)+"?", msg, func(ok bool) {
		if ok {
			sd.remove(src.ID)
		}
	}, sd.k.sh.win)
}

func (sd *sourcesDialog) remove(id string) {
	sd.k.sh.dispatch("kubeconfig.remove", sourceParams{Source: id}, func(_ any, err error) {
		if err != nil {
			sd.k.sh.showError(err)
		}
		sd.k.contextsChanged()
	})
}
