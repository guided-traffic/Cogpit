package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// ConnectResult is cluster.connect's result.
type ConnectResult struct {
	Context string                 `json:"context"`
	State   engine.ConnectionState `json:"state"`
	Error   string                 `json:"error,omitempty"`
}

type connectParams struct {
	Context string `json:"context" jsonschema:"context id as cluster_contexts lists it: a default kubeconfig context's name, <context> [<source>] for a kubeconfig source added in Cogpit, demo for the synthetic demo cluster"`
}

func registerCluster(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "cluster.contexts", Label: "List Contexts", Danger: engine.DangerRead,
		Description: "Lists the kubeconfig contexts Cogpit can connect to (name, cluster, user, default namespace, server, " +
			"whether it is the kubeconfig's current context) and the synthetic demo cluster if offered. Contexts of the " +
			"kubeconfig sources added in Cogpit (kubeconfig_sources) are named \"<context> [<source>]\" and carry source, " +
			"kubeContext, file and exec (approved, required or changed: whether their credential plugin may run; the user " +
			"allows it in Cogpit's window). Reads only the kubeconfig files; no credential is returned.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			list, err := s.Contexts()
			if err != nil && len(list) == 0 {
				return nil, err
			}
			if err != nil {
				s.Logger().Warn("commands: contexts listed partially", "err", err)
			}
			if list == nil {
				list = []engine.ContextInfo{}
			}
			return list, nil
		},
	})

	add(g, engine.Spec[connectParams]{
		ID: "cluster.connect", Label: "Connect to Cluster", MenuPath: "File/Connect to Cluster", Danger: engine.DangerUI,
		Description: "Connects to a kubeconfig context (or the demo cluster) and opens its Pods view, replacing the current " +
			"connection. Connecting only reads from the cluster. Returns {context, state, error}; state is connected or error.",
		Run: func(ctx context.Context, s *engine.Session, p connectParams) (any, error) {
			err := s.Connect(ctx, p.Context)
			u := s.UIState()
			if err != nil && u.Connection.State != engine.ConnectError {
				return nil, err // refused before connecting: no connector, closed, or superseded
			}
			return ConnectResult{Context: u.Context, State: u.Connection.State, Error: u.Connection.Error}, nil
		},
	})
}
