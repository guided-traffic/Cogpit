package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The navigator's scope (engine/navscope.go; ui-parity › Layout and Clusters). Which level the
// navigator shows, and with it which tabs the tab bar shows, is UI state, so agents see and drive
// it as the user does (D-2: a UI command, run freely; into another cluster it connects, as
// cluster.connect does, which only reads).

type navScopeParams struct {
	Scope string `json:"scope" jsonschema:"section:local for the Local Kubeconfigs section (every cluster Cogpit offers; the tab bar shows every tab), or cluster:<context id> for a cluster's tree (context id as cluster_contexts lists it; the tab bar shows that cluster's tabs). A cluster that is not connected is connected first"`
}

func registerNavScope(g *registrar) {
	add(g, engine.Spec[navScopeParams]{
		ID: "ui.setNavigatorScope", Label: "Show in Navigator", Danger: engine.DangerUI, Verbs: cacheReads,
		Description: "Sets what the navigator shows, as its scope dropdown, the ‹ on a cluster's root and the › on a " +
			"cluster's row do: section:local, the Local Kubeconfigs section listing every cluster Cogpit offers, or " +
			"cluster:<context id>, that cluster's resource tree. The tab bar shows the tabs of the scope: every tab at the " +
			"section, the cluster's tabs at a cluster (ui_tab_select's index, next and previous count those). Going up to " +
			"the section keeps the connection. A cluster that is not the connected one is connected first, as " +
			"cluster_connect does (Cogpit holds one connection; connecting only reads), and a failed connect leaves the " +
			"section with connection saying why. Every connect drills into its cluster, and without a connected cluster " +
			"the scope is the section. Returns the new UI state; navigatorScope is the scope.",
		Run: func(ctx context.Context, s *engine.Session, p navScopeParams) (any, error) {
			return s.SetNavigatorScope(ctx, p.Scope)
		},
	})
}
