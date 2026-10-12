package fyneui

import (
	"errors"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
)

// helmInstallTab is the "Helm Install: <chart>" dock tab (G11; inferred: Lens's install tab is not
// captured, so its layout follows the captured Helm Upgrade tab): Install and search icons, the
// chart as "/<repository>/<chart>" with a version picker of every version its repository lists,
// whether the archive matched the index's digest, the namespace and release name to install into
// with "Create namespace", the chart's values.yaml in a YAML editor whose text lives in the engine,
// and Cogpit's status line. Install sends the text, renders the install against the cluster
// without changing it (helm.chart.preview) and shows what it would create; the dialog's Install
// runs helm.chart.install bound to the text revision the send made. The namespace, the release
// name and the checkbox are the window's (the engine's tab holds what they start with); the
// version picker reloads the tab at another version (helm.chart.installTab).
type helmInstallTab struct {
	sh   *shell
	tab  engine.DockTab
	pane *editorPane

	installBtn *widget.Button
	searchBtn  *tipButton
	find       *findBar
	version    *widget.Select
	namespace  *widget.Entry
	release    *widget.Entry
	createNS   *widget.Check
	digest     *textLine
	status     *textLine
	root       fyne.CanvasObject

	// preview is the dialog that shows what an install renders, while it is open.
	preview *installPreviewDialog
	busy    bool
	// loaded is the text revision of the values the tab loaded: a higher one holds the user's
	// changes, which another version would replace.
	loaded uint64
}

// helmChartInstallParams mirror helm.chart.install's and helm.chart.preview's parameters (from a
// tab), installTabParams helm.chart.installTab's.
type (
	helmChartInstallParams struct {
		Tab             string `json:"tab,omitempty"`
		TextRevision    uint64 `json:"textRevision,omitempty"`
		Namespace       string `json:"namespace"`
		Name            string `json:"name"`
		CreateNamespace bool   `json:"createNamespace,omitempty"`
	}
	installTabParams struct {
		refParams
		Version string `json:"version,omitempty"`
	}
)

// installPreview mirrors helm.chart.preview's result.
type installPreview struct {
	Release    model.ObjectRef       `json:"release"`
	Repository string                `json:"repository"`
	Chart      string                `json:"chart"`
	Version    string                `json:"version"`
	Verified   bool                  `json:"verified"`
	Objects    []helm.RenderedObject `json:"objects"`
	Hooks      []helm.RenderedObject `json:"hooks"`
	CRDs       []helm.RenderedObject `json:"crds"`
}

func newHelmInstallTab(sh *shell, t engine.DockTab) *helmInstallTab {
	u := &helmInstallTab{sh: sh, tab: t, pane: newEditorPane(sh, t.ID)}
	hi := t.HelmInstall
	if hi == nil {
		hi = &engine.HelmInstallTab{}
	}
	u.installBtn = widget.NewButtonWithIcon("Install", theme.DownloadIcon(), u.install)
	u.installBtn.Importance = widget.HighImportance
	u.searchBtn = newTipButton(sh, theme.SearchIcon(), "Find", u.toggleSearch)
	u.find = newFindBar(sh, u.pane.view)
	u.version = widget.NewSelect(hi.Versions, nil)
	u.version.SetSelected(hi.Version)
	u.version.OnChanged = u.pickVersion
	u.namespace = widget.NewEntry()
	u.namespace.SetText(hi.Namespace)
	u.namespace.SetPlaceHolder("Namespace")
	u.release = widget.NewEntry()
	u.release.SetText(hi.ReleaseName)
	u.release.SetPlaceHolder("Release name")
	u.createNS = widget.NewCheck("Create namespace", nil)
	u.digest = newTextLine(digestText(hi), colorMuted, theme.SizeNameText, fyne.TextStyle{})
	if !hi.Verified {
		u.digest.SetColor(colorStatusWarning)
	}
	u.status = newTextLine("Loading…", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	u.status.truncate = true

	label := func(s string) fyne.CanvasObject {
		return container.NewCenter(newTextLine(s, colorMuted, theme.SizeNameText, fyne.TextStyle{}))
	}
	chartLine := container.NewHBox(u.installBtn, u.searchBtn, newDivider(true),
		container.NewCenter(newTextLine(hi.Repository+"/"+hi.Chart, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})),
		label("version:"), box(fitSelect{options: hi.Versions}, u.version), container.NewCenter(u.digest))
	target := container.NewBorder(nil, nil, nil, u.createNS,
		container.NewGridWithColumns(2,
			container.NewBorder(nil, nil, label("Namespace "), nil, u.namespace),
			container.NewBorder(nil, nil, label("  Release name "), nil, u.release)))
	header := container.NewStack(newSurface(colorDock), inset(theme.SizeNameInnerPadding, theme.SizeNamePadding,
		container.NewVBox(chartLine, target)))
	footer := container.NewStack(newSurface(colorDock), inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, u.status))
	editor := container.NewBorder(nil, u.find.root, nil, nil,
		container.NewStack(newSurface(theme.ColorNameInputBackground), u.pane.view))
	u.root = container.NewBorder(container.NewVBox(header, newDivider(false)), container.NewVBox(newDivider(false), footer), nil, nil, editor)
	u.setBusy(true)
	u.pane.load(func(st engine.EditorState, err error) {
		u.setBusy(false)
		u.loaded = st.Revision
		if err != nil {
			u.setStatus("Could not open the values: "+errText(err), colorStatusError)
			return
		}
		u.setStatus(fmt.Sprintf("The values.yaml of %s %s. Install renders the chart against the cluster first and shows what it creates.", hi.Chart, hi.Version), colorMuted)
	})
	return u
}

// digestText says how the chart's archive was checked.
func digestText(hi *engine.HelmInstallTab) string {
	if hi.Verified {
		return "digest verified"
	}
	return "unverified: the index lists no digest"
}

func (u *helmInstallTab) setStatus(text string, c fyne.ThemeColorName) {
	u.status.SetText(text)
	u.status.SetColor(c)
}

func (u *helmInstallTab) setBusy(b bool) {
	u.busy = b
	if b {
		u.installBtn.Disable()
	} else {
		u.installBtn.Enable()
	}
}

// params are the install's parameters from the tab and its fields.
func (u *helmInstallTab) params(textRevision uint64) helmChartInstallParams {
	return helmChartInstallParams{Tab: u.tab.ID, TextRevision: textRevision, Namespace: strings.TrimSpace(u.namespace.Text),
		Name: strings.TrimSpace(u.release.Text), CreateNamespace: u.createNS.Checked}
}

// install hands the text to the engine, renders the install against the cluster and shows what it
// creates; the dialog's Install installs at the text revision the send made, so that the install
// never runs on text other than what the user saw.
func (u *helmInstallTab) install() {
	if u.busy {
		return
	}
	text := u.pane.view.Text()
	u.setBusy(true)
	u.setStatus("Rendering the chart against the cluster…", colorMuted)
	u.sh.dispatch("editor.setText", editorTextParams{ID: u.tab.ID, YAML: text}, func(res any, err error) {
		if u.pane.closed {
			return
		}
		if err != nil {
			u.setBusy(false)
			u.setStatus("Not installed: "+errText(err), colorStatusError)
			return
		}
		var tr struct {
			Revision uint64 `json:"revision"`
		}
		if decode(res, &tr) == nil && tr.Revision > u.pane.pushed {
			u.pane.pushed = tr.Revision
		}
		u.pane.dirty = u.pane.view.Text() != text
		p := u.params(0)
		u.sh.dispatch("helm.chart.preview", p, func(res any, err error) {
			if u.pane.closed {
				return
			}
			var pre installPreview
			if err == nil {
				err = decode(res, &pre)
			}
			if err != nil {
				u.setBusy(false)
				u.setStatus("Not installed: "+errText(err), colorStatusError)
				return
			}
			u.setStatus("Review what the install creates.", colorMuted)
			u.preview = u.sh.showInstallPreview(pre, p.CreateNamespace, func(ok bool) {
				u.preview = nil
				if !ok {
					u.setBusy(false)
					u.setStatus("Not installed.", colorMuted)
					return
				}
				u.runInstall(u.params(tr.Revision))
			})
		})
	})
}

func (u *helmInstallTab) runInstall(p helmChartInstallParams) {
	u.setStatus(fmt.Sprintf("Installing release %s…", p.Name), colorMuted)
	u.sh.dispatch("helm.chart.install", p, func(res any, err error) {
		if u.pane.closed {
			return
		}
		u.setBusy(false)
		switch {
		case errors.Is(err, engine.ErrDeclined):
			u.setStatus("Not installed.", colorMuted)
		case err != nil:
			u.setStatus("Not installed: "+errText(err), colorStatusError)
		default:
			var r struct {
				Revision int `json:"revision"`
			}
			_ = decode(res, &r)
			u.setStatus(fmt.Sprintf("Installed release %s in namespace %s (revision %d).", p.Name, p.Namespace, r.Revision), colorStatusOK)
		}
	})
}

// pickVersion reloads the tab at another version: its values become that version's values.yaml.
// Edits to the values are lost, so the user is asked first when there are any.
func (u *helmInstallTab) pickVersion(v string) {
	hi := u.tab.HelmInstall
	if hi == nil || v == "" || v == hi.Version || u.busy {
		return
	}
	load := func() {
		u.setBusy(true)
		u.setStatus("Loading version "+v+"…", colorMuted)
		u.sh.dispatch("helm.chart.installTab", installTabParams{refParams: refParams{Ref: *u.tab.Ref}, Version: v}, func(_ any, err error) {
			if u.pane.closed {
				return
			}
			u.setBusy(false)
			if err != nil {
				u.version.SetSelected(hi.Version)
				u.setStatus("Version "+v+" not loaded: "+errText(err), colorStatusError)
			}
		})
	}
	if u.tab.Ref == nil {
		return
	}
	if !u.pane.dirty && u.pane.pushed <= u.loaded {
		load()
		return
	}
	dialog.ShowConfirm("Load version "+v+"?", "The values become version "+v+"'s values.yaml; your changes to the values are lost.",
		func(ok bool) {
			if ok {
				load()
			} else {
				u.version.SetSelected(hi.Version)
			}
		}, u.sh.win)
}

// update follows the engine's tab: a reload at another version replaces the values.
func (u *helmInstallTab) update(t engine.DockTab) {
	old := u.tab.HelmInstall
	u.tab = t
	hi := t.HelmInstall
	if hi == nil || old == nil || (hi.Version == old.Version && hi.Digest == old.Digest) {
		return
	}
	u.version.Options = hi.Versions
	u.version.SetSelected(hi.Version)
	u.digest.SetText(digestText(hi))
	if hi.Verified {
		u.digest.SetColor(colorMuted)
	} else {
		u.digest.SetColor(colorStatusWarning)
	}
	u.pane.load(func(st engine.EditorState, err error) {
		u.loaded = st.Revision
		if err == nil {
			u.setStatus(fmt.Sprintf("The values.yaml of %s %s.", hi.Chart, hi.Version), colorMuted)
		}
	})
}

func (u *helmInstallTab) toggleSearch() {
	if u.find.visible() {
		u.find.close()
		return
	}
	u.find.open()
}

func (u *helmInstallTab) remote(rev uint64) { u.pane.remote(rev) }

func (u *helmInstallTab) close() {
	if u.preview != nil {
		u.preview.d.Hide()
	}
	u.pane.close()
}

// installPreviewDialog shows what an install creates before it runs.
type installPreviewDialog struct {
	d       *dialog.CustomDialog
	install *widget.Button
	cancel  *widget.Button
	lines   []string
}

// maxPreviewLines bounds the objects the dialog lists; the rest are counted.
const maxPreviewLines = 200

// showInstallPreview shows the objects an install creates, the release, namespace and context, the
// chart and how its archive was checked; answer gets the user's choice.
func (sh *shell) showInstallPreview(p installPreview, createNamespace bool, answer func(bool)) *installPreviewDialog {
	width := sh.app.Settings().Theme().Size(sizeConfirmWidth) * 1.4
	pd := &installPreviewDialog{}
	add := func(prefix string, objs []helm.RenderedObject) {
		for _, o := range objs {
			name := o.Name
			if o.Namespace != "" {
				name = o.Namespace + "/" + o.Name
			}
			pd.lines = append(pd.lines, prefix+o.Kind+" "+name)
		}
	}
	add("", p.Objects)
	add("Hook: ", p.Hooks)
	add("CRD (stays after an uninstall): ", p.CRDs)
	list := container.NewVBox()
	for i, l := range pd.lines {
		if i == maxPreviewLines {
			list.Add(newTextLine(fmt.Sprintf("… and %d more", len(pd.lines)-i), colorMuted, theme.SizeNameText, fyne.TextStyle{}))
			break
		}
		list.Add(newTextLine(l, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Monospace: true}))
	}
	if len(pd.lines) == 0 {
		list.Add(newTextLine("The chart renders no objects.", colorMuted, theme.SizeNameText, fyne.TextStyle{}))
	}
	scroll := container.NewVScroll(list)
	scroll.SetMinSize(fyne.NewSize(width, 220))
	checked := "its archive matched the repository index's digest"
	if !p.Verified {
		checked = "its archive is unverified: the repository's index lists no digest"
	}
	where := fmt.Sprintf("Release %s in namespace %s (context %s), from chart %s/%s %s: %s.", p.Release.Name, p.Release.Namespace,
		p.Release.Context, p.Repository, p.Chart, p.Version, checked)
	if createNamespace {
		where += " The namespace is created if it does not exist."
	}
	content := container.NewVBox(
		newParagraph(where, colorMuted, width),
		newSpace(theme.SizeNamePadding),
		newTextLine(fmt.Sprintf("WHAT THE INSTALL CREATES (%d)", len(pd.lines)), colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{Bold: true}),
		scroll,
	)
	pd.install = widget.NewButton("Install", nil)
	pd.install.Importance = widget.HighImportance
	pd.cancel = widget.NewButton("Cancel", nil)
	pd.d = dialog.NewCustomWithoutButtons("Install "+p.Release.Name+"?", content, sh.win)
	pd.d.SetButtons([]fyne.CanvasObject{pd.cancel, pd.install})
	done := false
	finish := func(ok bool) {
		if done {
			return
		}
		done = true
		pd.d.Hide()
		answer(ok)
	}
	pd.cancel.OnTapped = func() { finish(false) }
	pd.install.OnTapped = func() { finish(true) }
	pd.d.SetOnClosed(func() { finish(false) })
	pd.d.Resize(fyne.NewSize(width, pd.d.MinSize().Height))
	pd.d.Show()
	return pd
}
