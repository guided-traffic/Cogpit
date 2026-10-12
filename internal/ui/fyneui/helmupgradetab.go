package fyneui

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
)

// helmUpgradeTab is Lens's "Helm Upgrade: <name>" dock tab (captured 2026-10-12): save and search
// icons, the chart and its version as text ("/<chart>  version: 0.1.0 (no repository)") beside a
// separate version picker, the line "Release <name> values.yaml, version …, on <namespace>
// namespace." with the release and the namespace as links, and the release's values in a YAML
// editor whose text lives in the engine (editor.get, editor.setText, as an Edit tab's). The search
// icon opens the editor's find widget at its bottom (findbar.go). Save sends the text, then
// upgrades the release with it (helm.release.upgrade with the tab id and the text revision the
// send made). Cogpit has no chart repositories, so the picker offers the release's own version
// only and the version in the line needs no spinner past the load; the status line at the bottom
// is Cogpit's: what the values are based on, progress and errors.
type helmUpgradeTab struct {
	sh   *shell
	tab  engine.DockTab
	pane *editorPane

	saveBtn     *tipButton
	searchBtn   *tipButton
	find        *findBar
	version     *widget.Select
	versionText *textLine
	releaseLink *linkText
	nsLink      *linkText
	infoVersion *textLine
	spinner     *widget.Activity
	status      *textLine
	root        fyne.CanvasObject

	busy bool
}

// noRepository marks a version that comes from the release itself, not from a chart repository,
// as Lens writes it (captured 2026-10-12).
const noRepository = " (no repository)"

func newHelmUpgradeTab(sh *shell, t engine.DockTab) *helmUpgradeTab {
	u := &helmUpgradeTab{sh: sh, tab: t, pane: newEditorPane(sh, t.ID)}
	chart, version, versions := "", "", []string(nil)
	if t.Helm != nil {
		chart, version, versions = t.Helm.Chart, t.Helm.Version, t.Helm.Versions
	}
	name, namespace, ctx := "", "", t.Context
	if t.Ref != nil {
		name, namespace = t.Ref.Name, t.Ref.Namespace
		if t.Ref.Context != "" {
			ctx = t.Ref.Context
		}
	}

	u.saveBtn = newTipButton(sh, theme.DocumentSaveIcon(), "Save: upgrade the release with these values", u.save)
	u.searchBtn = newTipButton(sh, theme.SearchIcon(), "Find", u.toggleSearch)
	u.find = newFindBar(sh, u.pane.view)
	u.version = widget.NewSelect(versions, nil)
	u.version.PlaceHolder = "Select…"
	u.versionText = newTextLine("version: "+version+noRepository, colorMuted, theme.SizeNameText, fyne.TextStyle{})

	u.releaseLink = newLinkText(name, func() {
		sh.followLink(model.ObjectRef{Context: ctx, Kind: helm.ReleaseKind, Namespace: namespace, Name: name}, ctx, false)
	})
	u.nsLink = newLinkText(namespace, func() {
		sh.followLink(model.ObjectRef{Context: ctx, Kind: kindNamespaces, Name: namespace}, ctx, false)
	})
	u.infoVersion = newTextLine(version, colorMuted, theme.SizeNameText, fyne.TextStyle{})
	u.infoVersion.Hide()
	u.spinner = widget.NewActivity()
	u.spinner.Start()
	plain := func(s string) fyne.CanvasObject {
		return newTextLine(s, colorMuted, theme.SizeNameText, fyne.TextStyle{})
	}
	info := box(inlineLayout{}, plain("Release "), u.releaseLink, plain(" values.yaml, version "), u.infoVersion, u.spinner,
		plain(", on "), u.nsLink, plain(" namespace."))

	u.status = newTextLine("Loading…", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	u.status.truncate = true

	chartLine := container.NewHBox(u.saveBtn, u.searchBtn, newDivider(true),
		container.NewCenter(newTextLine("/"+chart, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})),
		container.NewCenter(u.versionText),
		box(fitSelect{options: append([]string{u.version.PlaceHolder}, versions...)}, u.version))
	header := container.NewStack(newSurface(colorDock), inset(theme.SizeNameInnerPadding, theme.SizeNamePadding,
		container.NewVBox(chartLine, info)))
	footer := container.NewStack(newSurface(colorDock), inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, u.status))
	editor := container.NewBorder(nil, u.find.root, nil, nil,
		container.NewStack(newSurface(theme.ColorNameInputBackground), u.pane.view))
	u.root = container.NewBorder(container.NewVBox(header, newDivider(false)), container.NewVBox(newDivider(false), footer), nil, nil, editor)
	u.setBusy(true)
	u.pane.load(func(st engine.EditorState, err error) {
		u.setBusy(false)
		u.spinner.Stop()
		u.spinner.Hide()
		u.infoVersion.Show()
		if err != nil {
			u.setStatus("Could not open the values: "+errText(err), colorStatusError)
			return
		}
		u.showBase(st)
	})
	return u
}

// showBase says which revision the values are based on, and whether the release has moved on.
func (u *helmUpgradeTab) showBase(st engine.EditorState) {
	switch {
	case st.Missing:
		u.setStatus("The release no longer exists.", colorStatusWarning)
	case st.Stale:
		u.setStatus(fmt.Sprintf("The release moved to revision %d since these values (revision %d) were read: Save is refused.", st.CachedRevision, st.BaseRevision), colorStatusWarning)
	default:
		u.setStatus(fmt.Sprintf("The user-supplied values of revision %d. Save upgrades the release with them.", st.BaseRevision), colorMuted)
	}
}

func (u *helmUpgradeTab) setStatus(text string, c fyne.ThemeColorName) {
	u.status.SetText(text)
	u.status.SetColor(c)
}

func (u *helmUpgradeTab) setBusy(b bool) {
	u.busy = b
	if b {
		u.saveBtn.Disable()
	} else {
		u.saveBtn.Enable()
	}
}

// save hands the text to the engine, then upgrades the release with the tab's values: in that
// order, so that the upgrade never runs on text older than what the user sees.
func (u *helmUpgradeTab) save() {
	if u.busy || u.tab.Ref == nil {
		return
	}
	text := u.pane.view.Text()
	u.setBusy(true)
	u.setStatus("Upgrading…", colorMuted)
	u.sh.dispatch("editor.setText", editorTextParams{ID: u.tab.ID, YAML: text}, func(res any, err error) {
		if u.pane.closed {
			return
		}
		if err != nil {
			u.setBusy(false)
			u.setStatus("Not upgraded: "+errText(err), colorStatusError)
			return
		}
		var tr struct {
			Revision uint64 `json:"revision"`
		}
		if decode(res, &tr) == nil && tr.Revision > u.pane.pushed {
			u.pane.pushed = tr.Revision
		}
		u.pane.dirty = u.pane.view.Text() != text
		u.sh.dispatch("helm.release.upgrade", helmUpgradeParams{refParams: refParams{Ref: *u.tab.Ref}, Tab: u.tab.ID, TextRevision: tr.Revision}, func(res any, err error) {
			if u.pane.closed {
				return
			}
			u.setBusy(false)
			switch {
			case errors.Is(err, engine.ErrDeclined):
				u.setStatus("Not upgraded.", colorMuted)
			case err != nil:
				u.setStatus("Not upgraded: "+errText(err), colorStatusError)
			default:
				var r struct {
					Revision int `json:"revision"`
				}
				_ = decode(res, &r)
				u.setStatus(fmt.Sprintf("Upgraded to revision %d. Save upgrades it again with these values.", r.Revision), colorStatusOK)
			}
		})
	})
}

// toggleSearch opens the editor's find widget, or closes it.
func (u *helmUpgradeTab) toggleSearch() {
	if u.find.visible() {
		u.find.close()
		return
	}
	u.find.open()
}

// fitSelect sizes a select as wide as its longest option, which widget.Select would cut short.
type fitSelect struct{ options []string }

func (l fitSelect) MinSize(objs []fyne.CanvasObject) fyne.Size {
	th := fyne.CurrentApp().Settings().Theme()
	var w, h float32
	for _, o := range objs {
		ms := o.MinSize()
		w, h = fyne.Max(w, ms.Width), fyne.Max(h, ms.Height)
	}
	for _, s := range l.options {
		text := fyne.MeasureText(s, th.Size(theme.SizeNameText), fyne.TextStyle{}).Width
		w = fyne.Max(w, text+th.Size(theme.SizeNameInlineIcon)+4*th.Size(theme.SizeNameInnerPadding))
	}
	return fyne.NewSize(w, h)
}

func (l fitSelect) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

func (u *helmUpgradeTab) remote(rev uint64) { u.pane.remote(rev) }

func (u *helmUpgradeTab) close() {
	u.spinner.Stop()
	u.pane.close()
}

// inlineLayout lays its visible objects out on one line, each as wide as its text or content: a
// sentence of text, links and a spinner. Links ask for no width of their own (they shorten to fit),
// so their text is measured here.
type inlineLayout struct{}

func (inlineLayout) natural(o fyne.CanvasObject) fyne.Size {
	if l, ok := o.(*linkText); ok {
		th := fyne.CurrentApp().Settings().Theme()
		return fyne.MeasureText(l.text, th.Size(l.size), fyne.TextStyle{})
	}
	return o.MinSize()
}

func (l inlineLayout) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var w, h float32
	for _, o := range objs {
		if o.Visible() {
			ms := l.natural(o)
			w += ms.Width
			h = fyne.Max(h, ms.Height)
		}
	}
	return fyne.NewSize(w, h)
}

func (l inlineLayout) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	var x float32
	for _, o := range objs {
		if !o.Visible() {
			continue
		}
		ms := l.natural(o)
		o.Move(fyne.NewPos(x, (size.Height-ms.Height)/2))
		o.Resize(ms)
		x += ms.Width
	}
}
