package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The command palette (G13, engine/palette.go): its open state is UI state, so that agents see in
// ui_state when the user is picking a command and can open the palette for them; what the palette
// lists follows from the registry (Registry.Palette), and what it runs is dispatched like a menu
// item, with the same confirmation and audit.

func registerPalette(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: engine.PaletteOpenCommand, Label: "Command Palette", MenuPath: "View/Command Palette", Shortcut: "Cmd+Shift+P",
		Danger: engine.DangerUI,
		Description: "Opens the command palette in Cogpit's window, as ⌘⇧P (Ctrl+Shift+P) does. The user searches it for what " +
			"the window's menus offer in the current state (the actions of the object in the detail drawer, Switch to a view, " +
			"the hotbar's banks and clusters, ...) and picks one. ui_state reports palette: true while it is open. A command " +
			"run from the palette asks for confirmation exactly as from a menu. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.UpdateUI(func(u *engine.UIState) { u.Palette = true }), nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: engine.PaletteCloseCommand, Label: "Close Command Palette", Danger: engine.DangerUI,
		Description: "Closes the command palette in Cogpit's window, as Escape or a click outside it does. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.UpdateUI(func(u *engine.UIState) { u.Palette = false }), nil
		},
	})
}
