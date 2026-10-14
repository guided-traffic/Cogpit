package commands

import "github.com/guided-traffic/cogpit/internal/engine"

// The result types of this package that carry cluster text, as an agent reads them (H-82, H-63):
// Dispatch hands an agent's result to ForAgent (engine.AgentResult, engine/agentview.go).

// ForAgent filters object.get's model with its related objects as a bare model is filtered.
func (o objectWithRelated) ForAgent(v *engine.AgentView) any {
	v.Detail(o.DetailModel)
	return o
}

// ForAgent filters an editor tab's text: manifests (object.edit, resource.createTab) or a chart's
// values (a Helm Install or Upgrade tab).
func (r EditResult) ForAgent(v *engine.AgentView) any {
	r.Text = v.TabText(r.Tab.Type, r.Text)
	return r
}
