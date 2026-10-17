package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The command palette (G13, engine/palette.go): its open state and its typed text are UI state
// (D-51, A54: engine/palettetext.go), so that agents see in ui_state when the user is picking a
// command and what is typed, and can open the palette and type into it for them; what the palette
// lists follows from the registry (Registry.Palette), and what it runs is dispatched like a menu
// item, with the same confirmation and audit. On a text an agent typed, the window's Enter runs no
// change until the user picked the entry.

type paletteTextParams struct {
	Text string `json:"text" jsonschema:"the text to type into the command palette's field, one line of at most 1000 bytes; empty clears it"`
}

func registerPalette(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: engine.PaletteOpenCommand, Label: "Command Palette", MenuPath: "View/Command Palette", Shortcut: "Cmd+Shift+P",
		Danger: engine.DangerUI,
		Description: "Opens the command palette in Cogpit's window, as ⌘⇧P (Ctrl+Shift+P) does. The user searches it for what " +
			"the window's menus offer in the current state (the actions of the object in the detail drawer, Switch to a view, " +
			"the hotbar's clusters, ...) and picks one. ui_state reports palette: true while it is open, and " +
			"paletteText what is typed. A command run from the palette asks for confirmation exactly as from a menu. Opened " +
			"by an agent it does not take the keyboard focus. Returns the new UI state.",
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

	add(g, engine.Spec[paletteTextParams]{
		ID: "ui.palette.setText", Label: "Type in Command Palette", Danger: engine.DangerUI,
		Description: "Types into the command palette's field, opening the palette when it is closed (without the " +
			"keyboard focus), as the user types there: the palette lists the entries the text matches, best first. The " +
			"user still decides what runs: on a text an agent typed, Enter runs only an entry that reads or changes the " +
			"window (a view to switch to, a search); a change to a cluster or to Cogpit needs the user to pick its entry " +
			"with the arrow keys or a click. ui_state reports the text as paletteText and who typed it last as " +
			"paletteTextOrigin. Returns the new UI state.",
		Run: func(ctx context.Context, s *engine.Session, p paletteTextParams) (any, error) {
			return s.SetPaletteText(p.Text, engine.OriginOf(ctx))
		},
	})
}
