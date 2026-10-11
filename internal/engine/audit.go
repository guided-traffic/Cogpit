package engine

import (
	"encoding/json"
	"time"
)

// AuditEntry is one line of the audit log: an agent-initiated command and its outcome.
type AuditEntry struct {
	Time    time.Time       `json:"time"`
	Origin  Origin          `json:"origin"`
	Command string          `json:"command"`
	Params  json.RawMessage `json:"params,omitempty"`
	// Context is the context the command was dispatched for, or the one the user confirmed.
	Context string `json:"context,omitempty"`
	// Phase is "requested" for the line a write gets before it runs, "done" for the outcome.
	Phase string `json:"phase"`
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// Audit phases.
const (
	auditRequested = "requested"
	auditDone      = "done"
)

// audit writes one JSON line for an agent-initiated command. Results are not logged: they can be
// large and may carry cluster data; the parameters and the outcome are what the user reviews.
func (s *Session) audit(inv Invocation, context, phase string, err error) {
	if s.opts.Audit == nil {
		return
	}
	e := AuditEntry{
		Time: s.opts.Now().UTC(), Origin: inv.Origin, Command: inv.ID,
		Params: normalizeParams(inv.Params), Context: context, Phase: phase, OK: err == nil,
	}
	if !json.Valid(e.Params) {
		e.Params = nil
	}
	if err != nil {
		e.Error = err.Error()
	}
	line, mErr := json.Marshal(e)
	if mErr != nil {
		return
	}
	s.auditMu.Lock()
	defer s.auditMu.Unlock()
	if _, wErr := s.opts.Audit.Write(append(line, '\n')); wErr != nil {
		s.log.Warn("audit log write failed", "err", wErr)
	}
}
