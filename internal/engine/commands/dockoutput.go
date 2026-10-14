package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// dock.read (G17, D-44; engine/dockoutput.go): what a logs or terminal dock tab shows, for agents,
// read from the window page by page. A terminal tab only once the user allowed it with the tab's
// switch, dock.allowAgentRead, which only the window runs.

type dockReadParams struct {
	ID     string `json:"id" jsonschema:"dock tab id as ui_state lists it (dock), e.g. logs-1 or terminal-2"`
	After  *int64 `json:"after,omitempty" jsonschema:"return the oldest lines numbered above this: a page's last, to follow a log"`
	Before *int64 `json:"before,omitempty" jsonschema:"return the newest lines numbered below this: a page's first, to page back; without after and before, the newest lines"`
	Limit  int    `json:"limit,omitempty" jsonschema:"lines to return; default 200, at most 2000 (and at most 256 KiB of text)"`
}

type allowAgentReadParams struct {
	ID      string `json:"id" jsonschema:"terminal dock tab id as ui_state lists it (dock)"`
	Allowed bool   `json:"allowed" jsonschema:"true: agents may read the tab's output with dock_read; false: they may not"`
}

func registerDockOutput(g *registrar) {
	add(g, engine.Spec[dockReadParams]{
		ID: "dock.read", Label: "Read Tab Output", Danger: engine.DangerRead,
		Description: "Returns what a dock tab shows, as the user sees it, a page of its lines: a logs tab's lines its view shows " +
			"(the pod and container it narrows to and its filter applied, resource names and timestamps as shown), or a " +
			"terminal, shell, attach or node shell tab's screen (its rows top to bottom, no colours or escape sequences; the " +
			"terminal keeps no scrollback). Without after and before the newest lines; after: last pages forward, before: " +
			"first pages back; limit default 200, at most 2000 lines and 256 KiB, each line at most 8 KiB. shown counts the " +
			"lines the tab shows, held and dropped what a logs tab holds and no longer holds; more says whether lines lie " +
			"beyond the page. A terminal tab is refused unless the user allowed agents to read it with the tab's switch " +
			"(agentRead in ui_state; D-44). Reads the window, never the cluster; Secret values a program printed into a log " +
			"or a terminal are in its lines as printed. Editor tabs: editor_get; the audit log: audit_list.",
		Enabled: func(s *engine.Session, _ dockReadParams) error { return s.CheckDockOutput() },
		Run: func(ctx context.Context, s *engine.Session, p dockReadParams) (any, error) {
			return s.ReadDockOutput(ctx, p.ID, engine.DockOutputQuery{After: p.After, Before: p.Before, Limit: p.Limit})
		},
	})

	add(g, engine.Spec[allowAgentReadParams]{
		ID: "dock.allowAgentRead", Label: "Agents May Read This Tab", Danger: engine.DangerUI,
		GUIOnly: engine.ErrDockReadGUIOnly,
		Description: "Lets agents read a terminal, shell, attach or node shell tab's output with dock_read, or stops them, as " +
			"the tab's \"Agents may read\" switch does; off when a tab opens. Only the user, in Cogpit's window, can change " +
			"it: refused for agents and the CLI, so that no agent allows itself (D-44). While it is on, the tab carries a marker " +
			"that flashes at each agent's read, and it turns itself off after 15 minutes without one (ui_state's agentReadUntil, " +
			"agentLastRead and agentReadExpired; H-83). Agents read logs tabs without it. Returns the tab.",
		Run: func(_ context.Context, s *engine.Session, p allowAgentReadParams) (any, error) {
			return s.AllowAgentRead(p.ID, p.Allowed)
		},
	})
}
