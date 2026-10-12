package commands

import (
	"context"
	"errors"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// A cluster's menu beyond connecting (ui-parity › Clusters, "Cluster menu: connect, disconnect,
// settings, add to hotbar, remove"): Disconnect closes Cogpit's connection, Remove from Cogpit
// unlinks the context's kubeconfig file (engine/contextlink.go, engine/disconnect.go). Neither
// sends anything to a cluster, and neither writes or deletes a kubeconfig file.

type clusterLinksParams struct {
	Context string `json:"context,omitempty" jsonschema:"only this context id (cluster_contexts lists them); empty lists every context of the kubeconfig sources"`
}

type clusterRemoveParams struct {
	Context string `json:"context" jsonschema:"context id as cluster_contexts lists it: <context> [<file>]"`
}

// LinksResult is cluster.links' result.
type LinksResult struct {
	Links []engine.ContextLink `json:"links"`
}

func registerClusterMenu(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "cluster.disconnect", Label: "Disconnect", MenuPath: "File/Disconnect", Danger: engine.DangerUI,
		Description: "Closes Cogpit's connection to the connected cluster, or ends a connect under way. Nothing is sent to the " +
			"cluster: Cogpit stops watching it, its port forwards stop as when switching clusters, and the workspace shows " +
			"the welcome page. The cluster's tabs stay and connect again when selected. Returns the new UI state.",
		Enabled: func(s *engine.Session, _ struct{}) error {
			if st := s.UIState().Connection.State; st != engine.Connected && st != engine.Connecting {
				return engine.ErrNotConnected
			}
			return nil
		},
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			if err := s.Disconnect(); err != nil {
				return nil, err
			}
			return s.UIState(), nil
		},
	})

	add(g, engine.Spec[clusterLinksParams]{
		ID: "cluster.links", Label: "Show Kubeconfig Links", Danger: engine.DangerRead,
		Description: "Tells where Cogpit reads each context from: its kubeconfig source (a file the user added, a synced " +
			"folder such as the default ~/.kube, or a file $KUBECONFIG names for this run), the file's path, every context " +
			"that file defines, and whether cluster_remove can unlink it (removable) or why not (reason). Reads only the " +
			"kubeconfig files' names and contexts; no credential is returned.",
		Enabled: func(s *engine.Session, _ clusterLinksParams) error { _, err := sourcesAvailable(s); return err },
		Run: func(_ context.Context, s *engine.Session, p clusterLinksParams) (any, error) {
			if p.Context != "" {
				l, err := s.ContextLink(p.Context)
				if err != nil {
					return nil, err
				}
				return LinksResult{Links: []engine.ContextLink{l}}, nil
			}
			links, err := s.ContextLinks()
			if err != nil {
				return nil, err
			}
			return LinksResult{Links: links}, nil
		},
	})

	add(g, engine.Spec[clusterRemoveParams]{
		ID: "cluster.remove", Label: "Remove from Cogpit", MenuPath: "File/Remove from Cogpit", Danger: engine.DangerUI,
		Description: "Removes a cluster from Cogpit by removing Cogpit's link to the kubeconfig file the user added for it: " +
			"every context of that file is no longer offered, the login programs allowed for it are forgotten and a " +
			"connection made from it is closed. The file stays on disk as it is; Cogpit never edits or deletes a " +
			"kubeconfig. Refused, with the reason, for a context of a synced folder (the default ~/.kube too: its link is the " +
			"folder's), of a file $KUBECONFIG names, and for the demo cluster. The GUI asks the user first, naming the path." +
			guiOnly,
		Enabled: func(s *engine.Session, p clusterRemoveParams) error {
			if p.Context == "" {
				return errors.New("no context named")
			}
			return s.CheckRemoveContext(p.Context)
		},
		Run: func(ctx context.Context, s *engine.Session, p clusterRemoveParams) (any, error) {
			if err := requireGUI(ctx); err != nil {
				return nil, err
			}
			return s.RemoveContext(p.Context)
		},
	})
}
