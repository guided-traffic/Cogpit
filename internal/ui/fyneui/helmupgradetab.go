package fyneui

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// helmUpgradeTab is Lens's "Helm Upgrade: <name>" dock tab (captured 2026-10-12): save and search
// icons, the chart ("/<chart>": no repository) and a version picker, the line "Release <name>
// values.yaml, version …, on <namespace> namespace.", and the release's values in a YAML editor
// whose text lives in the engine (editor.get, editor.setText, as an Edit tab's). Save sends the
// text, then upgrades the release with it (helm.release.upgrade with the tab id). The status line
// at the bottom is Cogpit's: what the values are based on, progress and errors. Search finds the
// next line that contains the text (Cogpit's; Lens's search is not captured).
type helmUpgradeTab struct {
	sh   *shell
	tab  engine.DockTab
	pane *editorPane

	saveBtn   *tipButton
	searchBtn *tipButton
	search    *widget.Entry
	version   *widget.Select
	info      *textLine
	status    *textLine
	root      fyne.CanvasObject

	busy bool
}

// noRepository marks a version picker's choice that comes from the release itself, not from a
// chart repository, as Lens writes it (captured 2026-10-12).
const noRepository = " (no repository)"

func newHelmUpgradeTab(sh *shell, t engine.DockTab) *helmUpgradeTab {
	u := &helmUpgradeTab{sh: sh, tab: t, pane: newEditorPane(sh, t.ID)}
	chart, version, versions := "", "", []string(nil)
	if t.Helm != nil {
		chart, version = t.Helm.Chart, t.Helm.Version
		for _, v := range t.Helm.Versions {
			versions = append(versions, v+noRepository)
		}
	}
	name, namespace := "", ""
	if t.Ref != nil {
		name, namespace = t.Ref.Name, t.Ref.Namespace
	}

	u.saveBtn = newTipButton(sh, theme.DocumentSaveIcon(), "Save: upgrade the release with these values", u.save)
	u.searchBtn = newTipButton(sh, theme.SearchIcon(), "Search", u.toggleSearch)
	u.search = widget.NewEntry()
	u.search.SetPlaceHolder("Search")
	u.search.OnChanged = func(string) { u.find(false) }
	u.search.OnSubmitted = func(string) { u.find(true) }
	u.search.Hide()
	u.version = widget.NewSelect(versions, nil)
	if version != "" {
		u.version.SetSelected(version + noRepository)
	}
	u.info = newTextLine(fmt.Sprintf("Release %s values.yaml, version %s, on %s namespace.", name, version, namespace),
		colorMuted, theme.SizeNameText, fyne.TextStyle{})
	u.info.truncate = true
	u.status = newTextLine("Loading…", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	u.status.truncate = true

	chartLine := container.NewHBox(u.saveBtn, u.searchBtn, newDivider(true),
		container.NewCenter(newTextLine("/"+chart, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})),
		container.NewCenter(newTextLine("version:", colorMuted, theme.SizeNameText, fyne.TextStyle{})),
		box(fitSelect{options: versions}, u.version))
	header := container.NewStack(newSurface(colorDock), inset(theme.SizeNameInnerPadding, theme.SizeNamePadding,
		container.NewVBox(container.NewBorder(nil, nil, chartLine, nil, sized(sizeSearchWidth, "", u.search)), u.info)))
	footer := container.NewStack(newSurface(colorDock), inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, u.status))
	u.root = container.NewBorder(container.NewVBox(header, newDivider(false)), container.NewVBox(newDivider(false), footer), nil, nil,
		container.NewStack(newSurface(theme.ColorNameInputBackground), u.pane.view))
	u.setBusy(true)
	u.pane.load(func(st engine.EditorState, err error) {
		u.setBusy(false)
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
		u.sh.dispatch("helm.release.upgrade", helmUpgradeParams{refParams: refParams{Ref: *u.tab.Ref}, Tab: u.tab.ID}, func(res any, err error) {
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

// toggleSearch shows or hides the search entry.
func (u *helmUpgradeTab) toggleSearch() {
	if u.search.Visible() {
		u.search.Hide()
		return
	}
	u.search.Show()
	if c := fyne.CurrentApp().Driver().CanvasForObject(u.search); c != nil {
		c.Focus(u.search)
	}
}

// find moves the caret to the next line that contains the search text, ignoring case: from the
// caret's line on when typing, after it when next is set (Enter), wrapping at the end.
func (u *helmUpgradeTab) find(next bool) {
	needle := strings.ToLower(u.search.Text)
	if needle == "" {
		return
	}
	lines := strings.Split(u.pane.view.Text(), "\n")
	start, _ := u.pane.view.CursorPosition()
	if next {
		start++
	}
	for i := range lines {
		n := (start + i) % len(lines)
		lower := strings.ToLower(lines[n]) // lowering keeps every rune a rune: rune columns match
		if col := strings.Index(lower, needle); col >= 0 {
			u.pane.view.SetCursorPosition(n, utf8.RuneCountInString(lower[:col])) // scrolls the caret into view
			return
		}
	}
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

func (u *helmUpgradeTab) close() { u.pane.close() }
