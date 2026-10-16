package commands

import "github.com/guided-traffic/cogpit/internal/engine"

// The cluster of a command on a dock tab (D-69, several clusters connected at once): a log, an
// editor, a shell or a job tab works on its own tab's cluster, whichever one the window shows.
// The parameters below name their tab (engine.DockTargeted), so that Dispatch checks the tab's
// cluster's connection, access reviews and the allowance of agent writes, and the confirmation
// names its context; the commands read the tab's backend themselves (tabContext).

func (p dockParams) TargetDockTab() string            { return p.ID }
func (p dockReadParams) TargetDockTab() string        { return p.ID }
func (p allowAgentReadParams) TargetDockTab() string  { return p.ID }
func (p editorParams) TargetDockTab() string          { return p.ID }
func (p editorTextParams) TargetDockTab() string      { return p.ID }
func (p editParams) TargetDockTab() string            { return p.ID }
func (p applyParams) TargetDockTab() string           { return p.ID }
func (p diffParams) TargetDockTab() string            { return p.ID }
func (p jobParams) TargetDockTab() string             { return p.ID }
func (p logOptionsParams) TargetDockTab() string      { return p.ID }
func (p logSourceParams) TargetDockTab() string       { return p.ID }
func (p logJumpParams) TargetDockTab() string         { return p.ID }
func (p kubectlDownloadParams) TargetDockTab() string { return p.Tab }

var (
	_ engine.DockTargeted = dockParams{}
	_ engine.DockTargeted = kubectlDownloadParams{}
)

// tabContext is the cluster of a dock tab, "" (the shown cluster) when no tab has the id or the tab
// names none.
func tabContext(s *engine.Session, id string) string {
	if t, ok := s.UIState().DockTab(id); ok {
		return t.Context
	}
	return ""
}
