package fyneui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The Helm chart repositories in the GUI (G11): the Helm menu's "Repositories…" opens a dialog
// that lists the repositories added to Cogpit with their charts' count, when each index was
// downloaded and its last error, removes one (after asking), updates them all, and adds one by
// name and https URL. Lens keeps them in its preferences (Kubernetes › Helm Charts, with public
// repositories to pick and custom ones with credentials); Cogpit has no preferences page, adds
// public repositories only (H-29) and offers no list to pick from. The Charts view
// (helmcharts, a kind view) lists their charts; the dialog only manages where they come from.

// helmRepoParams mirror helm.repo.add's and helm.repo.remove's parameters.
type (
	helmRepoAddParams struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	}
	helmRepoNameParams struct {
		Name string `json:"name"`
	}
)

// helmReposDialog is the open repositories dialog.
type helmReposDialog struct {
	sh      *shell
	d       *dialog.CustomDialog
	list    *fyne.Container
	name    *widget.Entry
	url     *widget.Entry
	addB    *widget.Button
	updateB *widget.Button
	status  *textLine
	rows    []engine.HelmRepoInfo
	removeB map[string]*widget.Button // by repository name, for tests
	// removing is the open Remove confirmation, for tests.
	removing *dialog.ConfirmDialog
}

// helmMenu is the window menu's Helm menu.
func (sh *shell) helmMenu() *fyne.Menu {
	return fyne.NewMenu("Helm",
		fyne.NewMenuItem("Repositories…", sh.showHelmRepos),
		fyne.NewMenuItem("Update Repositories", func() { sh.dispatch("helm.repo.update", struct{}{}, nil) }),
		fyne.NewMenuItem("Charts", func() { sh.dispatch("ui.openView", openViewParams{View: helmChartsKind}, nil) }),
	)
}

// helmChartsKind is the Charts view's kind key.
const helmChartsKind = "helmcharts"

// showHelmRepos opens the repositories dialog, or reloads the open one.
func (sh *shell) showHelmRepos() {
	if sh.helmRepos != nil {
		sh.helmRepos.load()
		return
	}
	rd := &helmReposDialog{sh: sh, list: container.NewVBox(), removeB: map[string]*widget.Button{}}
	th := sh.app.Settings().Theme()
	rd.status = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	rd.status.truncate = true
	intro := newParagraph("Public Helm chart repositories Cogpit lists charts from. Cogpit keeps them in its own "+
		"configuration, not in Helm's, and downloads each index over https when you add or update it, and while the "+
		"Charts view is open once it is older than 10 minutes. Removing one leaves the releases installed from it.",
		colorMuted, th.Size(sizePickerWidth))
	rd.name = widget.NewEntry()
	rd.name.SetPlaceHolder("Name, e.g. bitnami")
	rd.url = widget.NewEntry()
	rd.url.SetPlaceHolder("https://charts.example.com/stable")
	rd.addB = widget.NewButtonWithIcon("Add", theme.ContentAddIcon(), rd.add)
	rd.addB.Importance = widget.HighImportance
	form := container.NewBorder(nil, nil, nil, rd.addB,
		container.NewBorder(nil, nil, box(fixedWidth(180), rd.name), nil, rd.url))
	listBg := newSurface(theme.ColorNameInputBackground)
	listBg.radius, listBg.stroke = theme.SizeNameInputRadius, theme.ColorNameInputBorder
	list := container.NewStack(listBg, inset(theme.SizeNamePadding, theme.SizeNamePadding, container.NewVScroll(rd.list)))
	content := sized(sizePickerWidth, sizePickerHeight, container.NewBorder(
		container.NewVBox(intro, newSpace(theme.SizeNamePadding), form, newSpace(theme.SizeNamePadding)), rd.status, nil, nil, list))
	rd.d = dialog.NewCustomWithoutButtons("Helm Repositories", content, sh.win)
	rd.updateB = widget.NewButtonWithIcon("Update All", theme.ViewRefreshIcon(), rd.update)
	closeB := widget.NewButton("Close", nil)
	rd.d.SetButtons([]fyne.CanvasObject{rd.updateB, closeB})
	closeB.OnTapped = rd.d.Hide
	rd.d.SetOnClosed(func() {
		if sh.helmRepos == rd {
			sh.helmRepos = nil
		}
	})
	sh.helmRepos = rd
	rd.d.Show()
	rd.load()
}

// fixedWidth lays its objects out at a fixed width and their own height: the name field.
type fixedWidth float32

func (w fixedWidth) MinSize(objs []fyne.CanvasObject) fyne.Size {
	var h float32
	for _, o := range objs {
		h = fyne.Max(h, o.MinSize().Height)
	}
	return fyne.NewSize(float32(w), h)
}

func (w fixedWidth) Layout(objs []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objs {
		o.Move(fyne.NewPos(0, 0))
		o.Resize(size)
	}
}

func (rd *helmReposDialog) setStatus(text string, c fyne.ThemeColorName) {
	rd.status.SetText(text)
	rd.status.SetColor(c)
}

// load reads the repositories and shows them.
func (rd *helmReposDialog) load() {
	rd.sh.dispatch("helm.repo.list", struct{}{}, func(res any, err error) {
		var rows []engine.HelmRepoInfo
		if err == nil {
			err = decode(res, &rows)
		}
		if err != nil {
			rd.setStatus(errText(err), colorStatusError)
			return
		}
		rd.rows = rows
		rd.render()
	})
}

func (rd *helmReposDialog) render() {
	th := rd.sh.app.Settings().Theme()
	width := th.Size(sizePickerWidth) - 8*th.Size(theme.SizeNamePadding)
	rd.removeB = map[string]*widget.Button{}
	objs := make([]fyne.CanvasObject, 0, len(rd.rows))
	for _, r := range rd.rows {
		repo := r
		title := newTextLine(repo.Name, theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
		u := newTextLine(repo.URL, colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
		u.truncate = true
		remove := widget.NewButtonWithIcon("Remove", theme.DeleteIcon(), func() { rd.confirmRemove(repo.Name) })
		remove.Importance = widget.LowImportance
		rd.removeB[repo.Name] = remove
		var info fyne.CanvasObject
		if repo.Error != "" {
			info = newParagraph(repo.Error, colorStatusError, width-remove.MinSize().Width-2*th.Size(theme.SizeNamePadding))
		} else {
			info = newTextLine(repoSummary(repo, rd.sh.now()), colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
		}
		objs = append(objs, inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, container.NewBorder(nil, nil, nil,
			container.NewCenter(remove), container.NewVBox(title, u, info))))
	}
	if len(objs) == 0 {
		objs = append(objs, inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
			newTextLine("No repositories added yet: add one by its name and https URL above.", colorMuted, theme.SizeNameText, fyne.TextStyle{})))
	}
	rd.list.Objects = objs
	rd.list.Refresh()
}

// repoSummary is a repository's second line: its charts and the age of its index.
func repoSummary(r engine.HelmRepoInfo, now time.Time) string {
	charts := fmt.Sprintf("%d charts", r.Charts)
	if r.Charts == 1 {
		charts = "1 chart"
	}
	switch {
	case r.Updating:
		return charts + " · updating…"
	case r.Updated.IsZero():
		return charts + " · no index downloaded"
	}
	return charts + " · index downloaded " + model.FormatAge(now.Sub(r.Updated)) + " ago"
}

// add adds the repository the form names.
func (rd *helmReposDialog) add() {
	p := helmRepoAddParams{Name: strings.TrimSpace(rd.name.Text), URL: strings.TrimSpace(rd.url.Text)}
	if p.Name == "" || p.URL == "" {
		rd.setStatus("Give the repository's name and its https URL.", colorStatusWarning)
		return
	}
	rd.addB.Disable()
	rd.setStatus("Downloading the index of "+p.Name+"…", colorMuted)
	rd.sh.dispatch("helm.repo.add", p, func(res any, err error) {
		rd.addB.Enable()
		if err != nil {
			rd.setStatus("Not added: "+errText(err), colorStatusError)
			return
		}
		var info engine.HelmRepoInfo
		_ = decode(res, &info)
		rd.name.SetText("")
		rd.url.SetText("")
		rd.setStatus(fmt.Sprintf("Added %s: %d charts.", info.Name, info.Charts), colorStatusOK)
		rd.load()
	})
}

// update downloads every index again.
func (rd *helmReposDialog) update() {
	rd.updateB.Disable()
	rd.setStatus("Updating the repositories…", colorMuted)
	rd.sh.dispatch("helm.repo.update", struct{}{}, func(_ any, err error) {
		rd.updateB.Enable()
		if err != nil {
			rd.setStatus("Not updated: "+errText(err), colorStatusError)
		} else {
			rd.setStatus("Updated.", colorStatusOK)
		}
		rd.load()
	})
}

// confirmRemove asks before removing a repository, naming what goes and what stays.
func (rd *helmReposDialog) confirmRemove(name string) {
	rd.removing = dialog.NewConfirm("Remove "+name+"?",
		"Cogpit forgets the repository "+name+" and its downloaded index; its charts leave the Charts view. "+
			"Releases installed from it are not touched.",
		func(ok bool) {
			rd.removing = nil
			if !ok {
				return
			}
			rd.sh.dispatch("helm.repo.remove", helmRepoNameParams{Name: name}, func(_ any, err error) {
				if err != nil {
					rd.setStatus("Not removed: "+errText(err), colorStatusError)
					return
				}
				rd.setStatus("Removed "+name+".", colorMuted)
				rd.load()
			})
		}, rd.sh.win)
	rd.removing.Show()
}
