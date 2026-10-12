package fyneui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// A cluster's Disconnect and Remove from Cogpit (ui-parity › Clusters, "Cluster menu: connect,
// disconnect, settings, add to hotbar, remove"; G13): in the window's Cluster menu, the context
// picker's row menu, a hotbar tile's menu, the navigator root's menu and the welcome page's rows.
// Disconnect (cluster.disconnect) closes Cogpit's connection; Remove from Cogpit (cluster.remove)
// removes Cogpit's link to the kubeconfig file the user added for the cluster, after a confirmation
// that names the path and every context that goes with it, and never edits or deletes the file. A
// context of a synced folder (the default ~/.kube too) or of a file $KUBECONFIG names cannot be
// removed that way: its item is greyed with the reason (cluster.links). The labels are Cogpit's
// (Lens's are not captured): "Remove from Cogpit" says what goes, and stays apart from the hotbar's
// "Remove from Hotbar" in the same menu. The demo cluster has no Remove: it comes from no file.

// linksFresh is how long the clusters' links are taken as current before a menu reads them again.
const linksFresh = 2 * time.Second

// contextLinkView is one entry of cluster.links' result.
type contextLinkView struct {
	Context   string       `json:"context"`
	Source    sourceStatus `json:"source"`
	File      string       `json:"file"`
	Contexts  []string     `json:"contexts"`
	Removable bool         `json:"removable"`
	Reason    string       `json:"reason"`
}

// clusterRemoveParams mirrors cluster.remove's and cluster.links' parameters.
type clusterRemoveParams struct {
	Context string `json:"context"`
}

// clusterLinks holds the clusters' links for the menus, read off the main goroutine (they read the
// kubeconfig files). A menu built before they are known offers Remove, and Remove asks the engine
// again before its confirmation. Main goroutine.
type clusterLinks struct {
	sh       *shell
	byCtx    map[string]contextLinkView
	none     bool // the engine keeps no kubeconfig sources: Remove does not apply
	at       time.Time
	loading  bool
	menuFor  string                // the window menu's Remove item as last built, to rebuild it on a change
	removing *dialog.ConfirmDialog // the open confirmation, for tests
}

func newClusterLinks(sh *shell) *clusterLinks { return &clusterLinks{sh: sh} }

// linksRead is a read of cluster.links.
type linksRead struct {
	byCtx map[string]contextLinkView
	none  bool
}

// read asks the engine for every context's link. Off the main goroutine.
func (l *clusterLinks) read() linksRead {
	res, err := l.sh.eng.Dispatch(l.sh.ctx, engine.Invocation{ID: "cluster.links", Params: json.RawMessage(`{}`), Origin: engine.OriginGUI})
	if err != nil {
		return linksRead{none: true} // no sources (or an engine without the command): nothing to remove
	}
	var out struct {
		Links []contextLinkView `json:"links"`
	}
	if decode(res, &out) != nil {
		return linksRead{none: true}
	}
	m := make(map[string]contextLinkView, len(out.Links))
	for _, link := range out.Links {
		m[link.Context] = link
	}
	return linksRead{byCtx: m}
}

// set takes a read and rebuilds the window's menu if its Remove item changes.
func (l *clusterLinks) set(r linksRead) {
	l.byCtx, l.none, l.at = r.byCtx, r.none, time.Now()
	state := ""
	if item := l.sh.removeItem(l.sh.ui.Context); item != nil {
		state = item.Label + fmt.Sprint(item.Disabled)
	}
	if state != l.menuFor {
		l.menuFor = state
		l.sh.win.SetMainMenu(l.sh.mainMenu())
	}
}

// refresh reads the links again unless a read is under way or the last one is fresh.
func (l *clusterLinks) refresh() {
	if l.loading || time.Since(l.at) < linksFresh {
		return
	}
	l.loading = true
	go func() {
		defer l.sh.recoverTo("the clusters' links")
		r := l.read()
		l.sh.do(func() {
			l.loading = false
			l.set(r)
		})
	}()
}

// clusterItems are a cluster's Disconnect (while it is the connected one) and Remove from Cogpit,
// for its menus.
func (sh *shell) clusterItems(contextID string) []*fyne.MenuItem {
	var items []*fyne.MenuItem
	st := sh.ui.Connection.State
	if c, ok := sh.eng.Registry().Get("cluster.disconnect"); ok && contextID != "" && contextID == sh.ui.Context &&
		(st == engine.Connected || st == engine.Connecting) {
		items = append(items, fyne.NewMenuItem(c.Label, sh.disconnect))
	}
	if item := sh.removeItem(contextID); item != nil {
		items = append(items, item)
	}
	if len(items) > 0 {
		items = append([]*fyne.MenuItem{fyne.NewMenuItemSeparator()}, items...)
	}
	return items
}

// clusterMenuItems are the window's Cluster menu's Disconnect, greyed while no cluster is
// connected, and Remove from Cogpit for the window's cluster.
func (sh *shell) clusterMenuItems() []*fyne.MenuItem {
	var items []*fyne.MenuItem
	if c, ok := sh.eng.Registry().Get("cluster.disconnect"); ok {
		item := fyne.NewMenuItem(c.Label, sh.disconnect)
		st := sh.ui.Connection.State
		item.Disabled = st != engine.Connected && st != engine.Connecting
		items = append(items, item)
	}
	if item := sh.removeItem(sh.ui.Context); item != nil {
		items = append(items, item)
	}
	return items
}

// disconnect closes the connection (cluster.disconnect); the workspace shows the welcome page.
func (sh *shell) disconnect() {
	if st := sh.ui.Connection.State; st == engine.Connected || st == engine.Connecting {
		sh.dispatch("cluster.disconnect", struct{}{}, nil)
	}
}

// removeItem is a context's "Remove from Cogpit…", greyed with the reason when its link cannot be
// removed; nil where Remove does not apply (no context, the demo cluster, an engine without
// kubeconfig sources).
func (sh *shell) removeItem(contextID string) *fyne.MenuItem {
	c, ok := sh.eng.Registry().Get("cluster.remove")
	links := sh.tabs.welcome.links
	if !ok || contextID == "" || contextID == engine.DemoContext {
		return nil
	}
	links.refresh()
	if links.none {
		return nil
	}
	item := fyne.NewMenuItem(c.Label+"…", func() { sh.confirmRemoveCluster(contextID) })
	if link, known := links.byCtx[contextID]; known && !link.Removable {
		item.Label = c.Label + " — " + shorten(link.Reason, maxReasonRunes)
		item.Disabled = true
	}
	return item
}

// confirmRemoveCluster asks the engine for the context's link as it is now, then asks the user,
// naming the file's path and every context that leaves with it; only then does cluster.remove run.
func (sh *shell) confirmRemoveCluster(contextID string) {
	links := sh.tabs.welcome.links
	if sh.picker.visible() {
		sh.picker.hide() // the confirmation is the window's dialog; the picker lies over it
	}
	raw, _ := json.Marshal(clusterRemoveParams{Context: contextID})
	go func() {
		defer sh.recoverTo("cluster.links")
		res, err := sh.eng.Dispatch(sh.ctx, engine.Invocation{ID: "cluster.links", Params: raw, Origin: engine.OriginGUI})
		sh.do(func() {
			var out struct {
				Links []contextLinkView `json:"links"`
			}
			if err == nil {
				err = decode(res, &out)
			}
			if err == nil && len(out.Links) != 1 {
				err = fmt.Errorf("no link for %s", kubeconfig.DisplayText(contextID, maxShownPath))
			}
			if err != nil {
				sh.showError(err)
				return
			}
			link := out.Links[0]
			name := sh.identity(contextID).name
			if !link.Removable {
				sh.toast("warning", "Cannot remove "+name+" from Cogpit: "+link.Reason)
				return
			}
			title, msg := removeClusterText(link, name)
			width := sh.app.Settings().Theme().Size(sizePickerWidth)
			d := dialog.NewCustomConfirm(title, "Remove", "Cancel", newParagraph(msg, theme.ColorNameForeground, width), func(ok bool) {
				links.removing = nil
				if !ok {
					return
				}
				sh.dispatch("cluster.remove", clusterRemoveParams{Context: contextID}, func(_ any, err error) {
					if err != nil {
						sh.showError(err)
					} else {
						sh.toast("info", "Removed "+name+" from Cogpit; "+kubeconfig.DisplayText(link.File, maxShownPath)+" stays on disk.")
					}
					links.at = time.Time{}
					links.refresh()
					sh.kc.contextsChanged()
				})
			}, sh.win)
			links.removing = d
			d.Show()
		})
	}()
}

// removeClusterText is the Remove from Cogpit confirmation: the Kubeconfig Sources' text for the
// file (its path, that only Cogpit's link goes and the file stays), and every context that leaves
// Cogpit with the link.
func removeClusterText(link contextLinkView, name string) (title, msg string) {
	_, msg = removeText(link.Source)
	title = "Remove " + kubeconfig.DisplayText(name, maxShownArg) + " from Cogpit?"
	if len(link.Contexts) > 1 {
		names := make([]string, 0, len(link.Contexts))
		for _, id := range link.Contexts {
			names = append(names, kubeconfig.DisplayText(engine.ContextLabel(id), maxShownArg))
		}
		msg += fmt.Sprintf("\n\nThe file defines %d contexts, and all of them leave Cogpit: %s.", len(names), strings.Join(names, ", "))
	}
	return title, msg
}
