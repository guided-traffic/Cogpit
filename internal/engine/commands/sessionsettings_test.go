package commands_test

import (
	"errors"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// D-69 items 4 and 8: the two settings of stage 3. Each one's "on" makes Cogpit run credential
// plugins without a click (a background cluster's login renewed while the user works elsewhere;
// the clusters of the last session connected at start), so only the user turns it on, in the
// window; agents and the CLI may turn it off.
func TestSessionSettingsAgentsMayOnlyTurnThemOff(t *testing.T) {
	s, _, _ := proxySession(t)
	agents := []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI}

	// Restore the last session: off by default.
	if p, _ := s.Preferences(); p.RestoreSession {
		t.Fatal("restoring the last session is on by default")
	}
	for _, o := range agents {
		if err := dispatch(t, s, o, "preferences.set", map[string]any{"restoreSession": true}, nil); !errors.Is(err, engine.ErrRestoreSessionGUIOnly) {
			t.Errorf("%s turned the restore on: %v", o, err)
		}
	}
	if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"restoreSession": true}, nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.Preferences(); !p.RestoreSession {
		t.Fatal("the user's restore was not kept")
	}
	for _, o := range agents {
		if err := dispatch(t, s, o, "preferences.set", map[string]any{"restoreSession": false}, nil); err != nil {
			t.Errorf("%s turned the restore off: %v", o, err)
		}
		if p, _ := s.Preferences(); p.RestoreSession {
			t.Errorf("%s's off was not kept", o)
		}
		if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"restoreSession": true}, nil); err != nil {
			t.Fatal(err)
		}
	}

	// Renew the login while the cluster is not shown: on by default.
	var v engine.ClusterSettingsView
	if err := dispatch(t, s, engine.OriginExternalAgent, "cluster.settings.get", map[string]any{"context": "kind-dev"}, &v); err != nil {
		t.Fatal(err)
	}
	if !v.RenewLoginInBackground {
		t.Fatal("the renewal in the background is off by default")
	}
	for _, o := range agents {
		if err := dispatch(t, s, o, "cluster.settings.set", map[string]any{"context": "kind-dev", "renewLoginInBackground": false}, &v); err != nil || v.RenewLoginInBackground {
			t.Errorf("%s turned the renewal off: %v (%v)", o, err, v.RenewLoginInBackground)
		}
		if err := dispatch(t, s, o, "cluster.settings.set", map[string]any{"context": "kind-dev", "renewLoginInBackground": true}, nil); !errors.Is(err, engine.ErrRenewLoginGUIOnly) {
			t.Errorf("%s turned the renewal on: %v", o, err)
		}
	}
	if err := dispatch(t, s, engine.OriginGUI, "cluster.settings.set", map[string]any{"context": "kind-dev", "renewLoginInBackground": true}, &v); err != nil || !v.RenewLoginInBackground {
		t.Errorf("the user turned the renewal on: %v (%v)", err, v.RenewLoginInBackground)
	}
	if cs := s.UIState().ClusterSettings["kind-dev"]; cs.HoldBackgroundLogin {
		t.Error("the UI state still holds the login")
	}
}
