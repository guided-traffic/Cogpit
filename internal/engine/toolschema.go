package engine

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

// A command as an agent's tool (D-2, D-10). The external MCP server (internal/mcp) and the
// built-in agent (internal/agent) describe the registry's commands to a model the same way: the
// parameter schema with the root type "object", and the command's description followed by what
// its danger level means for the agent. Both live here so that the two L4 packages, which may not
// import each other, cannot drift apart.

// ToolSchema returns the command's parameter schema as JSON with the root type "object", which MCP
// and OpenAI's function tools require. A pointer parameter type yields ["null", "object"]; the
// engine reads null as {}, so the tool advertises "object" alone. The JSON is a snapshot, so a
// caller never shares the registry's schema value.
func (c *Command) ToolSchema() (json.RawMessage, error) {
	return toolSchema(c.Schema)
}

// schemaObject is JSON Schema's object type.
const schemaObject = "object"

func toolSchema(s *jsonschema.Schema) (json.RawMessage, error) {
	if s == nil {
		return json.RawMessage(`{"type":"object"}`), nil
	}
	root := *s
	if root.Type == "" && slices.Contains(root.Types, schemaObject) {
		root.Type, root.Types = schemaObject, nil
	}
	if root.Type != schemaObject {
		return nil, fmt.Errorf("parameter schema has type %q; tools take an object", strings.Join(append([]string{root.Type}, root.Types...), ","))
	}
	data, err := json.Marshal(root)
	if err != nil {
		return nil, fmt.Errorf("parameter schema: %w", err)
	}
	return data, nil
}

// ToolDescription is the description an agent reads: the command's own, then what its danger level
// means for the agent, so that a model knows before calling whether the user will be asked.
func (c *Command) ToolDescription() string {
	desc := strings.TrimSpace(c.Description)
	if desc == "" {
		desc = c.Label
	}
	var note string
	switch c.Danger {
	case DangerRead:
		note = "Danger level: read. Reads from Cogpit's cache and runs without asking the user."
	case DangerUI:
		note = "Danger level: ui. Changes only what Cogpit's window shows and runs without asking the user."
	case DangerMutating:
		note = "Danger level: mutating. Writes to the cluster: the user must confirm it in Cogpit's window " +
			"unless they allowed agent writes for this context. The call waits for the answer and fails if the user declines."
	case DangerDestructive:
		note = "Danger level: destructive. The user must always confirm it in Cogpit's window, which names the objects, " +
			"the namespace and the context. The call waits for the answer and fails if the user declines."
	default:
		note = "Danger level: " + string(c.Danger) + "."
	}
	if c.Escalates {
		note += " Some calls are destructive (the description says which): those the user must always confirm."
	}
	if c.AgentsAsk {
		note += " Some calls always wait for the user's acceptance, also where the user allowed agent writes (the description says which)."
	}
	if c.GUIOnly { // no confirmation to describe: an agent's call never gets that far
		note = "Danger level: " + string(c.Danger) + ". GUI only: only the user's own action in Cogpit's window runs it, " +
			"so every call from an agent is refused whatever its arguments, before anything is asked, read or written. " +
			"Ask the user to do it in Cogpit's window."
	}
	return desc + "\n\n" + note
}
