// Package agent is Cogpit's built-in agent (D-10, M12, architecture §7): it implements
// engine.ChatAgent, the chat the window shows. It talks to an OpenAI-compatible Chat Completions
// endpoint (LM Studio, LiteLLM, Open WebUI, or another compatible server) with streaming and tool
// calling, and runs the model's tool calls as the registry's commands through Session.Dispatch with
// origin builtInAgent, so D-2's confirmations, the audit log and the agents' read rules apply.
//
// Layer L4. It dispatches engine commands only: no cluster, cache or terminal access of its own,
// no other L4 package (depguard l4-agent). Paths come from the binary (cmd/cogpit).
package agent
