package commands_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// fakeUpdater records what the engine asks of the app's updater.
type fakeUpdater struct {
	mu        sync.Mutex
	why       error
	starts    []bool
	automatic []bool
	checks    int
}

func (f *fakeUpdater) Unavailable() error { return f.why }

func (f *fakeUpdater) Start(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts = append(f.starts, on)
}

func (f *fakeUpdater) CheckForUpdates() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checks++
	return nil
}

func (f *fakeUpdater) SetAutomaticChecks(on bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.automatic = append(f.automatic, on)
}

func (f *fakeUpdater) seen() (starts, automatic []bool, checks int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]bool(nil), f.starts...), append([]bool(nil), f.automatic...), f.checks
}

// The automatic update checks are on by default and on the stable channel; the updater starts
// with the preference and follows every change of it. The user turns them on and off; agents and
// the CLI may turn them on but not off. Only the stable channel exists: beta is refused for
// everyone until prereleases exist.
func TestUpdatePreferences(t *testing.T) {
	s, cfg, _ := settingsSession(t)
	if p, _ := s.Preferences(); !p.CheckForUpdates || p.UpdateChannel != engine.UpdateChannelStable {
		t.Fatalf("defaults: checkForUpdates %v, channel %q", p.CheckForUpdates, p.UpdateChannel)
	}
	u := &fakeUpdater{}
	s.SetUpdater(u)
	if starts, _, _ := u.seen(); len(starts) != 1 || !starts[0] {
		t.Fatalf("started %v, want once with automatic checks on", starts)
	}

	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		err := dispatch(t, s, origin, "preferences.set", map[string]any{"checkForUpdates": false}, nil)
		if !errors.Is(err, engine.ErrUpdateChecksGUIOnly) {
			t.Errorf("%s turned the update checks off: %v", origin, err)
		}
	}
	if p, _ := s.Preferences(); !p.CheckForUpdates {
		t.Fatal("an agent's refused change turned the checks off")
	}
	if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"checkForUpdates": false}, nil); err != nil {
		t.Fatal(err)
	}
	if err := dispatch(t, s, engine.OriginExternalAgent, "preferences.set", map[string]any{"checkForUpdates": true}, nil); err != nil {
		t.Fatalf("an agent turning the checks on: %v", err)
	}
	// The same value again changes nothing for the updater.
	if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"checkForUpdates": true}, nil); err != nil {
		t.Fatal(err)
	}
	if _, automatic, _ := u.seen(); len(automatic) != 2 || automatic[0] || !automatic[1] {
		t.Errorf("the updater was told %v, want [false true]", automatic)
	}

	for _, origin := range []engine.Origin{engine.OriginGUI, engine.OriginExternalAgent} {
		err := dispatch(t, s, origin, "preferences.set", map[string]any{"updateChannel": "beta"}, nil)
		if !errors.Is(err, engine.ErrNoBetaChannel) {
			t.Errorf("%s chose beta: %v", origin, err)
		}
		if err := dispatch(t, s, origin, "preferences.set", map[string]any{"updateChannel": "nightly"}, nil); err == nil {
			t.Errorf("%s chose nightly", origin)
		}
		if err := dispatch(t, s, origin, "preferences.set", map[string]any{"updateChannel": ""}, nil); err != nil {
			t.Errorf("%s set the channel back to the default: %v", origin, err)
		}
	}
	if p, _ := s.Preferences(); p.UpdateChannel != engine.UpdateChannelStable {
		t.Errorf("channel %q", p.UpdateChannel)
	}

	// Kept in the preferences file; a file with an unknown channel or without the fields reads as
	// the defaults.
	data, err := os.ReadFile(filepath.Join(cfg, engine.PreferencesFile))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil || doc["checkForUpdates"] != true || doc["updateChannel"] != "stable" {
		t.Errorf("preferences file %s (%v)", data, err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, engine.PreferencesFile), []byte(`{"version":1,"theme":"light","updateChannel":"nightly"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	old := engine.New(engine.Options{Preferences: engine.NewPreferencesStore(dir)})
	t.Cleanup(old.Close)
	if p, _ := old.Preferences(); p.Theme != "light" || !p.CheckForUpdates || p.UpdateChannel != engine.UpdateChannelStable {
		t.Errorf("an older file: theme %q, checkForUpdates %v, channel %q", p.Theme, p.CheckForUpdates, p.UpdateChannel)
	}
}

// The updater starts with the automatic checks the user chose before; a change made while it is
// being wired in reaches it after its start.
func TestUpdaterStartsWithThePreference(t *testing.T) {
	s, _, _ := settingsSession(t)
	if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"checkForUpdates": false}, nil); err != nil {
		t.Fatal(err)
	}
	u := &fakeUpdater{}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"checkForUpdates": true}, nil); err != nil {
			t.Error(err)
		}
	}()
	s.SetUpdater(u)
	wg.Wait()
	starts, automatic, _ := u.seen()
	if len(starts) != 1 {
		t.Fatalf("started %v", starts)
	}
	last := starts[0]
	if len(automatic) > 0 {
		last = automatic[len(automatic)-1]
	}
	if !last {
		t.Errorf("started %v then told %v: the updater ends with the checks off, the preference is on", starts, automatic)
	}
}

// Help › Check for Updates… is enabled only where this Cogpit updates itself, and says why not
// otherwise: no updater at all (cogpit-cli), or the updater's own reason. Any origin may run it;
// an updater that cannot update is never started or asked.
func TestCheckForUpdatesCommand(t *testing.T) {
	s, _, _ := settingsSession(t)
	cmd, ok := s.Registry().Get("app.checkForUpdates")
	if !ok || cmd.MenuPath != "Help/Check for Updates…" || cmd.Danger != engine.DangerUI || cmd.GUIOnly {
		t.Fatalf("app.checkForUpdates: %+v", cmd)
	}
	if err := cmd.Enabled(s, nil); !errors.Is(err, engine.ErrNoUpdater) {
		t.Errorf("without an updater: %v", err)
	}
	if err := dispatch(t, s, engine.OriginExternalAgent, "app.checkForUpdates", struct{}{}, nil); err == nil {
		t.Error("ran without an updater")
	}

	noKey := errors.New("this build has no update key")
	off := &fakeUpdater{why: noKey}
	s.SetUpdater(off)
	if err := cmd.Enabled(s, nil); !errors.Is(err, noKey) {
		t.Errorf("an updater without a key: %v", err)
	}
	if err := dispatch(t, s, engine.OriginGUI, "preferences.set", map[string]any{"checkForUpdates": false}, nil); err != nil {
		t.Fatal(err)
	}
	if starts, automatic, checks := off.seen(); len(starts) != 0 || len(automatic) != 0 || checks != 0 {
		t.Errorf("an unavailable updater was used: starts %v, automatic %v, checks %d", starts, automatic, checks)
	}

	u := &fakeUpdater{}
	s.SetUpdater(u)
	if err := cmd.Enabled(s, nil); err != nil {
		t.Fatalf("enabled: %v", err)
	}
	for _, origin := range []engine.Origin{engine.OriginGUI, engine.OriginExternalAgent} {
		var res struct {
			Started bool `json:"started"`
		}
		if err := dispatch(t, s, origin, "app.checkForUpdates", struct{}{}, &res); err != nil || !res.Started {
			t.Errorf("%s: %+v %v", origin, res, err)
		}
	}
	if starts, _, checks := u.seen(); checks != 2 || len(starts) != 1 || starts[0] {
		t.Errorf("checks %d, starts %v (want 2 checks, one start with the checks off as set)", checks, starts)
	}
}
