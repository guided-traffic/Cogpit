package commands_test

import (
	"errors"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// stubChat is a ChatAgent that holds nothing: chat.toggle only needs the session to have one.
type stubChat struct{ engine.ChatAgent }

// TestChatToggle: ⌘I shows and hides the chat column as UI state, an agent may open it, and a
// session without a chat agent (the CLI's, a headless bridge's) disables the command.
func TestChatToggle(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})

	var disabled *engine.DisabledError
	if err := h.run(t, engine.OriginGUI, "chat.toggle", struct{}{}, nil); !errors.As(err, &disabled) {
		t.Fatalf("chat.toggle without a chat agent: %v, want disabled", err)
	}

	h.s.SetChatAgent(stubChat{})
	toggle := func(origin engine.Origin, params any) engine.UIState {
		var u engine.UIState // fresh: chat is omitted when false
		h.must(t, origin, "chat.toggle", params, &u)
		return u
	}
	if !toggle(engine.OriginGUI, struct{}{}).Chat {
		t.Fatal("chat.toggle did not show the chat column")
	}
	if toggle(engine.OriginGUI, struct{}{}).Chat {
		t.Fatal("chat.toggle did not hide the chat column")
	}
	toggle(engine.OriginExternalAgent, map[string]any{"open": true})
	if !toggle(engine.OriginExternalAgent, map[string]any{"open": true}).Chat {
		t.Fatal("chat.toggle {open: true} twice did not leave the chat column shown")
	}
	var st engine.UIState // ui.state embeds the UI state
	h.must(t, engine.OriginGUI, "ui.state", struct{}{}, &st)
	if !st.Chat {
		t.Fatal("ui.state does not report the chat column as shown")
	}
}
