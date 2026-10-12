package fyneui

import (
	"slices"

	"fyne.io/fyne/v2"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// listableKinds returns the kinds the sidebar and the View menu offer: every kind but those the
// access reviews say the user may not list. A kind they have not answered for yet is offered.
func listableKinds(kinds []kindInfo) []kindInfo {
	return slices.DeleteFunc(slices.Clone(kinds), func(k kindInfo) bool { return k.Listable != nil && !*k.Listable })
}

// refreshAccessItem is the Cluster menu's Refresh Access Reviews (cluster.refreshAccess): after
// the user's roles changed, the sidebar, the View menu and every command's state follow the new
// answers as they arrive. Disabled without a connected cluster and on the demo cluster, which
// has no access reviews; the menu is built again when either changes.
func (sh *shell) refreshAccessItem() *fyne.MenuItem {
	item := fyne.NewMenuItem("Refresh Access Reviews", func() {
		sh.dispatch("cluster.refreshAccess", struct{}{}, func(_ any, err error) {
			if err != nil {
				sh.showError(err)
				return
			}
			sh.toast("info", "Asking the cluster again what you may do")
		})
	})
	item.Disabled = sh.ui.Connection.State != engine.Connected || sh.ui.Connection.Demo
	return item
}
