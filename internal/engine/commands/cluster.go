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

// ConnectLogResult is cluster.connectLog's result.
type ConnectLogResult struct {
	Context string                  `json:"context"`
	Lines   []engine.ConnectLogLine `json:"lines"`
}

type connectLogParams struct {
	Context string `json:"context" jsonschema:"context id as cluster_contexts lists it"`
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

	add(g, engine.Spec[connectLogParams]{
		ID: "cluster.connectLog", Label: "Show Connection Updates", Danger: engine.DangerRead,
		Description: "Returns the log of the last attempt to connect to a context this session (and of a connection Cogpit " +
			"closed because its kubeconfig file went away): lines with a time, a level (INFO or ERROR) and a text. The " +
			"contexts cluster_contexts lists with connectProblem set have an error in it. Lines never contain a credential. " +
			"Empty when there was no attempt.",
		Run: func(_ context.Context, s *engine.Session, p connectLogParams) (any, error) {
			return ConnectLogResult{Context: p.Context, Lines: s.ConnectLog(p.Context)}, nil
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
