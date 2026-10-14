package commands

import (
	"context"
	"errors"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The built-in agent's chat column (D-10, M12; engine/chat.go). Whether it is shown is UI state, so
// agents see in ui_state that the user has the chat open and can open it for them; what is said
// in it stays in the window (the chat itself is no command and no MCP tool).

type chatToggleParams struct {
	Open *bool `json:"open,omitempty" jsonschema:"true shows the chat column, false hides it; omitted: switches it"`
}

var errNoChat = errors.New("this Cogpit has no agent chat (the command line and a headless engine have none)")

func registerChat(g *registrar) {
	add(g, engine.Spec[chatToggleParams]{
		ID: "chat.toggle", Label: "Agent Chat", MenuPath: "View/Agent Chat", Shortcut: "Cmd+I", Danger: engine.DangerUI,
		Description: "Shows or hides the built-in agent's chat column at the right edge of Cogpit's window, as ⌘I " +
			"(Ctrl+I) does. ui_state reports chat: true while it is shown. Returns the new UI state.",
		Enabled: func(s *engine.Session, _ chatToggleParams) error {
			if s.Chat() == nil {
				return errNoChat
			}
			return nil
		},
		Run: func(_ context.Context, s *engine.Session, p chatToggleParams) (any, error) {
			return s.UpdateUI(func(u *engine.UIState) {
				if p.Open != nil {
					u.Chat = *p.Open
					return
				}
				u.Chat = !u.Chat
			}), nil
		},
	})
}
