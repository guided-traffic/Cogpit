package commands_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// TestRightClickPreferenceIsTheUsers: copy and paste with the right mouse button is off by
// default; agents and the CLI cannot set it, alone or beside a preference they may set (nothing is
// written then); the user can, and it is kept in the preferences' file and shown in ui_state.
func TestRightClickPreferenceIsTheUsers(t *testing.T) {
	s, cfg, _ := settingsSession(t)
	var res commands.PreferencesResult
	if err := dispatch(t, s, engine.OriginExternalAgent, "preferences.get", struct{}{}, &res); err != nil {
		t.Fatal(err)
	}
	if res.Preferences.TerminalRightClickCopyPaste {
		t.Fatal("on by default")
	}
	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		for _, params := range []map[string]any{
			{"terminalRightClickCopyPaste": true},
			{"terminalRightClickCopyPaste": false, "theme": "light"},
		} {
			if err := dispatch(t, s, origin, "preferences.set", params, nil); !errors.Is(err, engine.ErrRightClickGUIOnly) {
				t.Errorf("%s set %v: %v, want ErrRightClickGUIOnly", origin, params, err)
			}
		}
	}
	if p, _ := s.Preferences(); p.TerminalRightClickCopyPaste || p.Theme != engine.ThemeSystem {
		t.Errorf("a refused change was kept: %+v", p)
	}

	if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"terminalRightClickCopyPaste": true}, &res); err != nil {
		t.Fatal(err)
	}
	if !res.Preferences.TerminalRightClickCopyPaste {
		t.Errorf("the user's choice: %+v", res.Preferences)
	}
	if u := s.UIState().Preferences; u == nil || !u.TerminalRightClickCopyPaste {
		t.Errorf("ui_state shows %+v", u)
	}
	raw, err := os.ReadFile(filepath.Join(cfg, engine.PreferencesFile))
	if err != nil || !strings.Contains(string(raw), `"terminalRightClickCopyPaste": true`) && !strings.Contains(string(raw), `"terminalRightClickCopyPaste":true`) {
		t.Errorf("the file holds %s (%v)", raw, err)
	}
	// An agent changing another preference leaves it as the user set it.
	if err := dispatch(t, s, engine.OriginExternalAgent, "preferences.set", map[string]any{"theme": "light"}, &res); err != nil {
		t.Fatal(err)
	}
	if !res.Preferences.TerminalRightClickCopyPaste || res.Preferences.Theme != engine.ThemeLight {
		t.Errorf("after an agent's theme: %+v", res.Preferences)
	}
}
