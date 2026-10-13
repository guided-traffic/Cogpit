package engine

import "errors"

// The allowance of agent writes (D-2; G17; docs/security/agents-and-mcp.md › H-14): for one
// context, the user may let agents' mutating commands run without asking. It is a cluster setting
// (ClusterSettings.AgentWrites, kept in cluster-settings.json, shown in UIState.ClusterSettings and
// cluster.settings.get, so agents see whether their writes will ask), off by default, and only the
// user sets it, in Cogpit's window (cluster.settings.setAgentWrites refuses every other origin:
// an agent can never allow itself). Destructive commands always ask, allowed or not
// (NeedsConfirmation). The kept allowance counts only in a session with a confirmer, the window's:
// a headless engine (cogpit-cli, the stdio bridge without a running app) reads no window's
// allowance and refuses every write that would need one. SetAgentWrites is the run's own allowance,
// kept in memory (tests, and sessions without cluster settings).

// ErrAgentWritesGUIOnly refuses a change of the allowance from anywhere but the window.
var ErrAgentWritesGUIOnly = errors.New("only the user, in Cogpit's own window, can allow or forbid agents' changes " +
	"without asking (Cluster Settings › Agents); agents and the CLI are refused, so that an agent cannot allow itself")

// agentWritesLocked reports whether agents' mutating commands run without asking in context: the
// run's allowance (SetAgentWrites), or the kept one while the session has a confirmer. s.mu is held.
func (s *Session) agentWritesLocked(context string) bool {
	if context == "" {
		return false
	}
	return s.agentWrites[context] || (s.confirmer != nil && s.ui.ClusterSettings[context].AgentWrites)
}

// AgentWritesAllowed reports whether agents' mutating commands run without asking in context, as
// Dispatch decides it.
func (s *Session) AgentWritesAllowed(context string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.agentWritesLocked(context)
}
