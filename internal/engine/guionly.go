package engine

import "fmt"

// GUI-only commands (G17, D-2). Some commands only the user's own action in Cogpit's window may
// run, whatever their parameters: what they do runs a program on the user's computer, adds a
// credential, shows a Secret's value or decides what agents may do (kubeconfig.addFile,
// secret.reveal, secret.create, cluster.settings.setAgentWrites, ...). Such a command declares it
// (Spec.GUIOnly, with its reason), and Dispatch refuses every other origin, the CLI and every
// agent, once the parameters decode and before Enabled, Refuse, a confirmation or Run: nobody is
// asked, nothing is read or written, and the refused origin learns nothing from Enabled's reason
// (whether the drawer shows a Secret's key, whether a context can be removed); the window's menus
// ask Enabled directly and still say why a command is greyed. It stays an MCP tool like every command (an agent learns from it
// what to ask the user to do), described as GUI-only, and its refusal is audited like any agent
// call. Commands that refuse only some of their parameters to agents (a proxy in
// cluster.settings.set, the terminal's shell in preferences.set) keep their own Refuse.

// GUIOnlyError is a GUI-only command refused to an origin other than the GUI. It unwraps to the
// command's own reason (ErrGUIOnly, ErrSecretCreateGUIOnly, ...).
type GUIOnlyError struct {
	Command string
	Origin  Origin
	Reason  error
}

func (e *GUIOnlyError) Error() string { return fmt.Sprintf("command %s: %v", e.Command, e.Reason) }

// Unwrap returns the command's reason.
func (e *GUIOnlyError) Unwrap() error { return e.Reason }

// refuseOrigin refuses a GUI-only command to every origin but the GUI.
func (c *Command) refuseOrigin(o Origin) error {
	if c.guiOnly == nil || o == OriginGUI {
		return nil
	}
	return &GUIOnlyError{Command: c.ID, Origin: o, Reason: c.guiOnly}
}
