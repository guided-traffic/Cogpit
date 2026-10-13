package fyneui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Argo CD in the window (G23): the navigator's group, the Sync and Refresh dialogs, the
// Dashboard's panel (argocd_dashboard.go).
//
// Lens's documentation (read 2026-10-13, not captured) lists the group's entries: Dashboard,
// Applications, Application Sets, Projects, Settings › Repositories, Settings › Clusters. Where
// Lens puts the group is not documented: Cogpit places it right after Flux CD (captured between
// Applications and Nodes), where Lens's documentation lists Argo CD too, after Flux CD (inferred).
// It shows while the cluster serves Argo CD's kinds; the argoproj.io group stays under Custom
// Resources as well, as curated sections' groups do.

// argoGroup is the navigator group's path.
const argoGroup = "argocd"

// argoNavGroup is the Argo CD group of navLayout.
var argoNavGroup = func() navEntry {
	dash := page(engine.ArgoCDDashboard)
	dash.needs = kinds.ArgoApplicationsKey
	return navGroup(argoGroup, "Argo CD", dash,
		titled(kinds.ArgoApplicationsKey, "Applications"),
		titled(kinds.ArgoApplicationSetKey, "Application Sets"),
		titled(kinds.ArgoProjectsKey, "Projects"),
		navGroup(argoGroup+"/settings", "Settings",
			titled(kinds.ArgoRepositoriesKey, "Repositories"),
			titled(kinds.ArgoClustersKey, "Clusters")))
}()

// The commands of Argo CD's dialogs.
const (
	argoSyncCommand        = "argocd.application.sync"
	argoSyncManyCommand    = "argocd.application.syncMany"
	argoRefreshCommand     = "argocd.application.refresh"
	argoRefreshManyCommand = "argocd.application.refreshMany"
)

// selectionDialogs are the GUI's own dialogs for actions on a selection, by command id: a command
// listed here opens its dialog from the selection's menu instead of running at once
// (tablecheck.go runOnSelection). Each feature registers its own in an init of its own file.
var selectionDialogs = map[string]func(sh *shell, refs []model.ObjectRef){}

func init() {
	itemActionDialogs[argoSyncCommand] = func(sh *shell, ref model.ObjectRef, _ model.ItemAction) {
		sh.openArgoSyncDialog([]model.ObjectRef{ref})
	}
	itemActionDialogs[argoRefreshCommand] = func(sh *shell, ref model.ObjectRef, _ model.ItemAction) {
		sh.openArgoRefreshDialog([]model.ObjectRef{ref})
	}
	selectionDialogs[argoSyncManyCommand] = (*shell).openArgoSyncDialog
	selectionDialogs[argoRefreshManyCommand] = (*shell).openArgoRefreshDialog
	commandIcons[argoSyncCommand] = theme.IconNameUpload // the live state moves to Git's (Cogpit's choice)
	commandIcons[argoRefreshCommand] = theme.IconNameViewRefresh
}

// argoRowNoun names the Dashboard's rows in hints: Needs attention lists Applications.
func argoRowNoun(sh *shell) string { return strings.ToLower(sh.kindTitle(kinds.ArgoApplicationsKey)) }

// argoTitle names the Applications a dialog acts on: "Application shop", "3 Applications".
func argoTitle(refs []model.ObjectRef) string {
	if len(refs) == 1 {
		return "Application " + refs[0].Name
	}
	return fmt.Sprintf("%d Applications", len(refs))
}

// argoNames lists the Applications a dialog acts on, at most ten by name.
func argoNames(refs []model.ObjectRef) string {
	names := make([]string, 0, min(len(refs), engine.ConfirmNamed))
	for _, r := range refs[:min(len(refs), engine.ConfirmNamed)] {
		names = append(names, r.Key())
	}
	s := strings.Join(names, ", ")
	if n := len(refs) - len(names); n > 0 {
		s += fmt.Sprintf(" and %d more", n)
	}
	return s
}

// argoDialog shows a dialog of Argo CD's actions: a line naming the Applications and their
// context, the options, and the action's button (Lens's "Sync app(s)" and "Refresh app(s)"
// dialogs, documented, not captured: the layout and texts are Cogpit's).
func (sh *shell) argoDialog(title, action string, refs []model.ObjectRef, options fyne.CanvasObject, run func()) {
	width := sh.app.Settings().Theme().Size(sizeConfirmWidth)
	context := ""
	if len(refs) > 0 {
		context = refs[0].Context
	}
	what := newParagraph(argoNames(refs)+" (context "+context+")", colorMuted, width)
	d := dialog.NewCustomConfirm(title, action, "Cancel", container.NewVBox(what, newSpace(theme.SizeNamePadding), options),
		func(ok bool) {
			if ok {
				run()
			}
		}, sh.win)
	d.Resize(fyne.NewSize(width, d.MinSize().Height))
	d.Show()
}

// openArgoSyncDialog asks whether to prune, then syncs: one Application through
// argocd.application.sync, several through argocd.application.syncMany. Prune starts off, as in
// Lens's documentation ("leave it off unless you intend that"); a sync with prune is destructive,
// so the engine's confirmation follows, naming what prune deletes (Cogpit's default: Lens shows
// the risk in its dialog).
func (sh *shell) openArgoSyncDialog(refs []model.ObjectRef) {
	if len(refs) == 0 {
		return
	}
	prune := widget.NewCheck("Prune resources not in Git", nil)
	width := sh.app.Settings().Theme().Size(sizeConfirmWidth)
	hint := newParagraph("Pruning deletes the live resources the Application manages that are no longer in Git; Cogpit asks once more before it does.", colorMuted, width)
	sh.argoDialog("Sync "+argoTitle(refs), "Sync", refs, container.NewVBox(prune, hint), func() {
		if len(refs) == 1 {
			sh.dispatch(argoSyncCommand, argoSyncParams{Ref: refs[0], Prune: prune.Checked}, nil)
			return
		}
		sh.dispatch(argoSyncManyCommand, argoSyncManyParams{Refs: refs, Prune: prune.Checked}, nil)
	})
}

// openArgoRefreshDialog asks Normal or Hard, then refreshes.
func (sh *shell) openArgoRefreshDialog(refs []model.ObjectRef) {
	if len(refs) == 0 {
		return
	}
	const normal, hard = "Normal: compare with Git again", "Hard: also discard the cached manifests"
	mode := widget.NewRadioGroup([]string{normal, hard}, nil)
	mode.Required = true
	mode.SetSelected(normal)
	sh.argoDialog("Refresh "+argoTitle(refs), "Refresh", refs, mode, func() {
		isHard := mode.Selected == hard
		if len(refs) == 1 {
			sh.dispatch(argoRefreshCommand, argoRefreshParams{Ref: refs[0], Hard: isHard}, nil)
			return
		}
		sh.dispatch(argoRefreshManyCommand, argoRefreshManyParams{Refs: refs, Hard: isHard}, nil)
	})
}

// Parameters of Argo CD's commands, mirroring their schemas.
type (
	argoSyncParams struct {
		Ref   model.ObjectRef `json:"ref"`
		Prune bool            `json:"prune"`
	}
	argoSyncManyParams struct {
		Refs  []model.ObjectRef `json:"refs"`
		Prune bool              `json:"prune"`
	}
	argoRefreshParams struct {
		Ref  model.ObjectRef `json:"ref"`
		Hard bool            `json:"hard"`
	}
	argoRefreshManyParams struct {
		Refs []model.ObjectRef `json:"refs"`
		Hard bool              `json:"hard"`
	}
)
