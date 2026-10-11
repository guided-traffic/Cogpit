package engine

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/guided-traffic/cogpit/internal/terminal"
)

// ErrDemoCluster is returned for streams the synthetic demo cluster cannot serve.
var ErrDemoCluster = errors.New("not available on the demo cluster")

// logTailLines is how much history a logs tab starts with.
const logTailLines = 1000

// streamTab returns a dock tab of the wanted types with the connected backend, checking that the
// tab belongs to the connected cluster and that the cluster has a connection for streams.
func (s *Session) streamTab(tabID string, types ...DockType) (DockTab, *Backend, error) {
	tab, ok := s.UIState().DockTab(tabID)
	if !ok {
		return DockTab{}, nil, fmt.Errorf("no dock tab %q", tabID)
	}
	typeOK := false
	for _, t := range types {
		typeOK = typeOK || tab.Type == t
	}
	if !typeOK {
		return DockTab{}, nil, fmt.Errorf("dock tab %q is a %s tab", tabID, tab.Type)
	}
	b := s.Backend()
	if b == nil {
		return DockTab{}, nil, ErrNotConnected
	}
	if tab.Context != "" && tab.Context != b.Context {
		return DockTab{}, nil, fmt.Errorf("%w: dock tab %q belongs to context %q, connected is %q", ErrWrongCluster, tabID, tab.Context, b.Context)
	}
	if b.Conn == nil {
		return DockTab{}, nil, fmt.Errorf("%s: %w", tab.Type, ErrDemoCluster)
	}
	if tab.Type != DockTerminal && tab.Ref == nil {
		return DockTab{}, nil, fmt.Errorf("dock tab %q names no pod", tabID)
	}
	return tab, b, nil
}

// OpenLogs opens the log stream of a logs dock tab, with the tab's options (logs.go): for a pod,
// the container's last 1,000 lines, then follow; for a controlling resource, the merged logs of
// its pods. The caller closes it.
func (s *Session) OpenLogs(ctx context.Context, tabID string) (io.ReadCloser, error) {
	tab, b, err := s.streamTab(tabID, DockLogs)
	if err != nil {
		return nil, err
	}
	return s.openLogs(ctx, tab, b, b.Conn)
}

// OpenTerminal starts the session of a terminal, shell or attach dock tab. A terminal tab is a
// local shell bound to the connected context and to the namespace filter if it names exactly one
// namespace (else the context's namespace); a shell tab is a shell in the tab's container; an
// attach tab is attached to the container's main process. The caller closes it.
func (s *Session) OpenTerminal(ctx context.Context, tabID string) (terminal.Session, error) {
	tab, b, err := s.streamTab(tabID, DockTerminal, DockShell, DockAttach)
	if err != nil {
		return nil, err
	}
	if tab.Type == DockAttach {
		return s.openAttach(ctx, tab, b)
	}
	if tab.Type == DockTerminal {
		ns := b.Namespace
		if u := s.UIState(); len(u.Namespaces) == 1 {
			ns = u.Namespaces[0]
		}
		return terminal.StartLocal(terminal.LocalOptions{
			KubeconfigPaths: b.Conn.KubeconfigPaths(), Context: b.Context, Namespace: ns,
		})
	}
	return terminal.StartPodExec(ctx, b.Conn.RESTConfig(), b.Conn.Kube(), terminal.ExecOptions{
		Namespace: tab.Ref.Namespace, Pod: tab.Ref.Name, Container: tab.Container,
	})
}
