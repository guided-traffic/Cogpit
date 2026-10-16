package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The commands of an interrupted connection (D-58): Reconnect now and Log in again of the
// connection banner, and a shell tab's Reconnect.

// ConnectionResult is cluster.reconnect's and cluster.login's result: the connection as the UI
// state holds it.
type ConnectionResult struct {
	Context    string            `json:"context"`
	Connection engine.Connection `json:"connection"`
}

func connectionResult(s *engine.Session, contextID string) ConnectionResult {
	u := s.UIState()
	if contextID != "" && contextID != u.Context {
		c, _ := u.ConnectionOf(contextID)
		return ConnectionResult{Context: contextID, Connection: c.Connection}
	}
	return ConnectionResult{Context: u.Context, Connection: u.Connection}
}

// subresourceExec is the pods subresource a shell opens.
const subresourceExec = "exec"

// errConnNotInterrupted and errLoginNotExpired are the commands' Enabled reasons.
var (
	errConnNotInterrupted = errors.New("the connection is not interrupted")
	errLoginNotExpired    = errors.New("the login has not expired")
)

func registerConnection(g *registrar) {
	add(g, engine.Spec[engine.OnCluster]{
		ID: "cluster.reconnect", Label: "Reconnect Now", MenuPath: "File/Reconnect Now", Danger: engine.DangerUI,
		Description: "Tries at once to reach a connected cluster (context, else the shown one) while its connection is " +
			"interrupted (ui_state's connection.state, or its entry of connections, is interrupted: the computer slept, the " +
			"network or a VPN went away) and starts the backoff of the attempts again (1, 2, 4, 8, 16, then every 30 " +
			"seconds). Cogpit reconnects by itself; this only does not wait. Sends no credentials and runs no login program. " +
			"Returns the connection.",
		Enabled: func(s *engine.Session, p engine.OnCluster) error {
			u := s.UIState()
			st := u.Connection.State
			if p.Context != "" && p.Context != u.Context {
				c, ok := u.ConnectionOf(p.Context)
				if !ok {
					return &engine.NotConnectedError{Context: p.Context}
				}
				st = c.Connection.State
			}
			if st != engine.Interrupted {
				return errConnNotInterrupted
			}
			return nil
		},
		Run: func(_ context.Context, s *engine.Session, p engine.OnCluster) (any, error) {
			if err := s.ReconnectContext(p.Context); err != nil {
				return nil, err
			}
			return connectionResult(s, p.Context), nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "cluster.login", Label: "Log In Again", MenuPath: "File/Log In Again", Danger: engine.DangerUI,
		GUIOnly: engine.ErrGUIOnly,
		Description: "Renews the shown cluster's login after it expired (ui_state's connection.state is loginExpired): " +
			"the context's credential plugin runs once, and may open a browser for the user to log in. The cache, the tabs " +
			"and the views stay. A plugin whose program changed since the user allowed it is not run: then the context is " +
			"connected anew, which asks the user first. Returns the connection." + guiOnly,
		Enabled: func(s *engine.Session, _ struct{}) error {
			u := s.UIState()
			switch {
			case u.Connection.State != engine.LoginExpired:
				return errLoginNotExpired
			case u.Connection.LoggingIn:
				return errors.New("a login is under way")
			}
			return nil
		},
		Run: func(ctx context.Context, s *engine.Session, _ struct{}) (any, error) {
			if err := requireGUI(ctx); err != nil {
				return nil, err
			}
			if err := s.Login(); err != nil {
				return nil, err
			}
			return connectionResult(s, ""), nil
		},
	})

	add(g, engine.Spec[dockParams]{
		ID: "dock.reconnect", Label: "Reconnect", Danger: engine.DangerMutating,
		Verbs: []engine.Verb{
			{Verb: nwVerbCreate, Resource: kindPods, Subresource: subresourceExec},
			{Verb: nwVerbCreate, Resource: kindPods, Subresource: "attach"},
		},
		Description: "Opens a new session in a shell, attach or node shell tab whose session ended because the connection " +
			"was interrupted (its dock tab's interrupted is set): a new shell in the same container, attached again, or a new " +
			"root shell in the node shell's pod while it still runs. A process cannot be resumed: what ran in the old session " +
			"is gone. Anything typed there runs with the container's or the node's privileges, so an agent's request waits for " +
			"the user's acceptance; a node shell's always does. Returns the tab.",
		Enabled: func(s *engine.Session, p dockParams) error {
			if _, err := streamableOn(s, tabContext(s, p.ID), "shells"); err != nil {
				return err
			}
			tab, ok := s.UIState().DockTab(p.ID)
			if !ok {
				return fmt.Errorf("no dock tab %q", p.ID)
			}
			if !reconnectable(tab) {
				return fmt.Errorf("%w: the tab's session was not ended by an interruption", engine.ErrNotApplicable)
			}
			return nil
		},
		Escalate: func(s *engine.Session, p dockParams) bool {
			tab, ok := s.UIState().DockTab(p.ID)
			return ok && tab.Type == engine.DockNodeShell // a root shell on a node (D-37)
		},
		Targets: func(s *engine.Session, p dockParams) ([]model.ObjectRef, error) {
			tab, ok := s.UIState().DockTab(p.ID)
			if !ok || tab.Ref == nil {
				return nil, fmt.Errorf("no dock tab %q", p.ID)
			}
			if tab.NodeShell != nil { // the node shell's pod
				return []model.ObjectRef{{Context: tab.Context, Kind: kindPods, Namespace: tab.NodeShell.Namespace, Name: tab.NodeShell.Pod}}, nil
			}
			return []model.ObjectRef{*tab.Ref}, nil
		},
		Run: func(ctx context.Context, s *engine.Session, p dockParams) (any, error) {
			if cur, ok := s.UIState().DockTab(p.ID); ok {
				if err := engine.CheckConfirmedContext(ctx, cur.Context); err != nil {
					return nil, err
				}
			}
			var tab engine.DockTab
			found := false
			s.UpdateUI(func(u *engine.UIState) {
				i := slices.IndexFunc(u.Dock, func(t engine.DockTab) bool { return t.ID == p.ID })
				if i < 0 || !reconnectable(u.Dock[i]) {
					return
				}
				u.Dock[i].Interrupted = time.Time{}
				u.Dock[i].Session++ // the window opens the new session in the tab
				u.ActiveDock = p.ID
				tab, found = u.Dock[i], true
			})
			if !found {
				return nil, fmt.Errorf("dock tab %q has no session to reconnect", p.ID)
			}
			return DockResult{Tab: tab}, nil
		},
	})
}

// reconnectable reports whether a dock tab's session ended with the connection.
func reconnectable(t engine.DockTab) bool {
	switch t.Type {
	case engine.DockShell, engine.DockAttach, engine.DockNodeShell:
		return !t.Interrupted.IsZero()
	}
	return false
}
