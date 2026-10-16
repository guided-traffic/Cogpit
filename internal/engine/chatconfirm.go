package engine

// What the built-in agent's chat shows while one of its tool calls runs (D-10, M12): a call that
// waits for the user's confirmation says so ("waiting for your confirmation") instead of
// "running". The agent cannot see inside Dispatch whether the window's confirmer is showing, so it
// asks beforehand with the same rules Dispatch applies.

// AsksUser reports whether Dispatch of inv would now wait for the user's confirmation (D-2): the
// command exists, its parameters decode, the origin is not refused (GUI-only), it is enabled, a
// confirmer is installed, and NeedsConfirmation says so for its danger with these parameters, the
// allowance of agent writes for the connected context and the command's own agents-ask rule. It
// runs no command and asks nobody; a panic in a command's hooks reads as false. The answer is a
// forecast for display: the allowance or the cluster can change before Dispatch decides.
func (s *Session) AsksUser(inv Invocation) bool {
	cmd, ok := s.reg.Get(inv.ID)
	if !ok || !inv.Origin.valid() {
		return false
	}
	p, err := cmd.decode(inv.Params)
	if err != nil || cmd.refuseOrigin(inv.Origin) != nil {
		return false
	}
	if err := s.safeCall(cmd.ID, func() error { return cmd.enabled(s, p) }); err != nil {
		return false
	}
	danger := cmd.Danger
	if err := s.safeCall(cmd.ID, func() error { danger = cmd.dangerFor(s, p); return nil }); err != nil {
		return false
	}
	agentsAsk := false
	if inv.Origin.IsAgent() && danger == DangerMutating {
		if err := s.safeCall(cmd.ID, func() error { agentsAsk = cmd.agentsAskFor(s, p); return nil }); err != nil {
			return false
		}
	}
	target := targetContext(s, p) // the command's cluster (D-69, target.go)
	s.mu.Lock()
	if target == "" {
		target = s.ui.Context
	}
	allowed := s.agentWritesLocked(target)
	hasConfirmer := s.confirmer != nil
	s.mu.Unlock()
	return hasConfirmer && NeedsConfirmation(danger, inv.Origin, allowed, agentsAsk)
}
