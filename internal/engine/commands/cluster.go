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
	// TLSUnchecked: the connection does not check the API server's certificate
	// (engine.Connection.TLSUnchecked, A29).
	TLSUnchecked bool `json:"tlsUnchecked,omitempty"`
}

// connectResult is the result of a connect from the UI state after it.
func connectResult(u engine.UIState) ConnectResult {
	return ConnectResult{Context: u.Context, State: u.Connection.State, Error: u.Connection.Error, TLSUnchecked: u.Connection.TLSUnchecked}
}

type connectParams struct {
	Context string `json:"context" jsonschema:"context id as cluster_contexts lists it: <context> [<file>] (a bare context name works when one file only defines it)"`
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
			"whether it is the kubeconfig's current context). Every context comes " +
			"from a kubeconfig file read on its own, never merged with another (kubeconfig_sources lists the files: ~/.kube, " +
			"the files $KUBECONFIG names, and those the user added). It is named \"<context> [<file>]\", so a name that two " +
			"files share gives two contexts, and carries source, " +
			"kubeContext, file and exec (approved, required or changed: whether their credential plugin may run, or, with " +
			"execKind insecure, whether a context whose cluster skips the certificate check may connect; the user " +
			"allows it in Cogpit's window), and insecure for such a context. Reads only the kubeconfig files; no credential is returned.",
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
		ID: "cluster.connect", Label: "Connect to Cluster", MenuPath: "Cluster/Connect to Cluster", Shortcut: "Cmd+P", Danger: engine.DangerUI,
		Description: "Connects to a context as cluster_contexts lists it, replacing the current connection, and shows its most " +
			"recently used tab, or else its Pods view in the preview tab; the other clusters' tabs stay. Connecting only reads from the cluster. " +
			"A context whose credential plugin, or whose connection without the certificate check (insecure-skip-tls-verify), " +
			"the user has not allowed in Cogpit's window ends in state error saying so: only the user allows it there. " +
			"Returns {context, state, error, tlsUnchecked}; state is connected or error; tlsUnchecked is set when the " +
			"connection does not check the server's certificate (ui_state's connection says the same while it stands).",
		Run: func(ctx context.Context, s *engine.Session, p connectParams) (any, error) {
			err := s.Connect(ctx, p.Context)
			u := s.UIState()
			if err != nil && u.Connection.State != engine.ConnectError {
				return nil, err // refused before connecting: no connector, closed, or superseded
			}
			return connectResult(u), nil
		},
	})
}
