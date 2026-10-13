package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The allowance of agent writes (D-2, H-14; engine/allowance.go): Cluster Settings › Agents. Only
// the user sets it, in Cogpit's window: the command refuses agents and the CLI before anything is
// read or written, so that an agent can never allow itself. Agents read it in ui_state and
// cluster_settings_get (agentWrites).

type agentWritesParams struct {
	Context string `json:"context,omitempty" jsonschema:"context id as cluster_contexts lists it; empty or omitted: the connected context"`
	Allowed bool   `json:"allowed" jsonschema:"true: agents' mutating commands in this cluster run without asking; false: each waits for the user's acceptance. Destructive commands always ask"`
}

// refuseAgentWrites refuses a change of the allowance from anywhere but the window.
func refuseAgentWrites(ctx context.Context) error {
	if engine.OriginOf(ctx) != engine.OriginGUI {
		return engine.ErrAgentWritesGUIOnly
	}
	return nil
}

func registerAllowance(g *registrar) {
	add(g, engine.Spec[agentWritesParams]{
		ID: "cluster.settings.setAgentWrites", Label: "Allow Agent Writes", Danger: engine.DangerUI,
		GUIOnly: engine.ErrAgentWritesGUIOnly,
		Description: "Allows or forbids, for one context, agents' mutating commands without the user's acceptance, as the " +
			"Agents section of its Settings page does; destructive commands always ask. Only the user, in Cogpit's window, " +
			"can change it: refused for agents and the CLI, so that no agent allows itself. Agents read it as agentWrites " +
			"in cluster_settings_get and ui_state." + clusterSettingsOnly + " Returns the settings.",
		Enabled: func(s *engine.Session, p agentWritesParams) error {
			_, err := settingsContext(s, p.Context)
			return err
		},
		Refuse: func(ctx context.Context, _ *engine.Session, _ agentWritesParams) error { return refuseAgentWrites(ctx) },
		Run: func(ctx context.Context, s *engine.Session, p agentWritesParams) (any, error) {
			if err := refuseAgentWrites(ctx); err != nil {
				return nil, err
			}
			return changeSettings(s, p.Context, func(cs *engine.ClusterSettings) error {
				cs.AgentWrites = p.Allowed
				return nil
			})
		},
	})
}
