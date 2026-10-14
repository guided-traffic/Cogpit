package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Start at login (D-48; engine/loginitem.go): Preferences › App › Start at login, the operating
// system's login item of Cogpit's app, read from and written to macOS, never to Cogpit's files.
// Agents read it and may turn it off; only the user turns it on, and only the user opens System
// Settings from Cogpit.

type startAtLoginParams struct {
	Enabled bool `json:"enabled" jsonschema:"true: macOS starts Cogpit when the user logs in (only the user, in Cogpit's window, may turn it on); false: it does not"`
}

// loginItemEnabled is the commands' Enabled: offered only where the binary wired a login item in.
func loginItemEnabled(s *engine.Session) error {
	if !s.HasLoginItem() {
		return engine.ErrNoLoginItem
	}
	return nil
}

func registerLoginItem(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "preferences.loginItem", Label: "Show Start at Login", Danger: engine.DangerRead,
		Description: "Reads whether macOS starts Cogpit when the user logs in (Preferences › App › Start at login) and shows it: " +
			"status off, on, requiresApproval (registered; the user must allow it in System Settings › General › Login Items) or " +
			"notFound (macOS finds no login item for this app: not a signed Cogpit.app it knows, such as a development build). " +
			"ui_state carries the last status read as loginItem. Not offered on Windows, Linux or the CLI.",
		Enabled: func(s *engine.Session, _ struct{}) error { return loginItemEnabled(s) },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.RefreshLoginItem()
		},
	})

	add(g, engine.Spec[startAtLoginParams]{
		ID: "preferences.setStartAtLogin", Label: "Start at Login", Danger: engine.DangerUI,
		Description: "Makes macOS start Cogpit when the user logs in, or stops it, as Preferences › App › Start at login does: " +
			"registers or removes the app's login item, which System Settings › General › Login Items lists, and returns the " +
			"status macOS then reports (off, on, requiresApproval, notFound). Agents and the CLI may only turn it off: turning " +
			"it on is the user's, in Cogpit's window. Changes nothing in a cluster or a kubeconfig.",
		Enabled: func(s *engine.Session, _ startAtLoginParams) error { return loginItemEnabled(s) },
		Refuse: func(ctx context.Context, _ *engine.Session, p startAtLoginParams) error {
			return engine.RefuseStartAtLogin(ctx, p.Enabled)
		},
		Run: func(ctx context.Context, s *engine.Session, p startAtLoginParams) (any, error) {
			return s.SetStartAtLogin(ctx, p.Enabled)
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "preferences.openLoginItems", Label: "Open Login Items", Danger: engine.DangerUI, GUIOnly: engine.ErrLoginItemsGUIOnly,
		Description: "Opens System Settings › General › Login Items, where the user allows Cogpit's login item when its " +
			"status is requiresApproval. GUI only: it opens a window outside Cogpit's.",
		Enabled: func(s *engine.Session, _ struct{}) error { return loginItemEnabled(s) },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return nil, s.OpenLoginItemsSettings()
		},
	})
}
