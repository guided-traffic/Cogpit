package commands_test

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestKubectlDownloadCommand: an agent can open a terminal and read its offer, but only the GUI
// downloads; nothing is fetched or written for a refused origin.
func TestKubectlDownloadCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake kubectl is a shell script")
	}
	rel := testkit.NewKubectlReleases(t)
	rel.Publish(t, "1.36.1", runtime.GOOS, runtime.GOARCH, []byte(testkit.FakeKubectl("v1.36.1")))
	bin := filepath.Join(t.TempDir(), "kubectl")
	s := engine.New(engine.Options{
		Preferences: engine.NewPreferencesStore(t.TempDir()),
		Kubectl:     engine.NewKubectl(bin, engine.KubectlHooks{Downloader: rel.Downloader}),
	})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	s.UpdateUI(func(u *engine.UIState) {
		u.Dock = append(u.Dock, engine.DockTab{ID: "terminal-1", Type: engine.DockTerminal, Title: "Terminal 1", Context: "dev",
			Kubectl: &engine.KubectlCheck{Server: "1.36.1", Offer: "1.36.1", Message: "kubectl 1.36.1 matches this cluster"}})
	})
	params := map[string]any{"tab": "terminal-1"}
	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		if err := dispatch(t, s, origin, "kubectl.download", params, nil); !errors.Is(err, engine.ErrKubectlGUIOnly) {
			t.Errorf("%s: %v, want ErrKubectlGUIOnly", origin, err)
		}
	}
	if got := rel.Requests(); len(got) != 0 {
		t.Errorf("refused origins fetched %v", got)
	}
	if _, err := os.Stat(bin); !os.IsNotExist(err) {
		t.Errorf("refused origins made the directory for binaries: %v", err)
	}

	var res commands.KubectlDownloadResult
	if err := dispatch(t, s, engine.OriginGUI, "kubectl.download", params, &res); err != nil {
		t.Fatalf("the GUI: %v", err)
	}
	if want := filepath.Join(bin, "1.36.1", "kubectl"); res.Path != want {
		t.Errorf("path %s, want %s", res.Path, want)
	}
	if tab, _ := s.UIState().DockTab("terminal-1"); tab.Kubectl.Download == nil || tab.Kubectl.Download.State != engine.KubectlDone {
		t.Errorf("the tab after the download: %+v", tab.Kubectl)
	}
	// A tab that offers nothing has the command disabled, with the reason.
	if err := dispatch(t, s, engine.OriginGUI, "kubectl.download", map[string]any{"tab": "terminal-9"}, nil); err == nil {
		t.Error("an unknown tab downloaded")
	}
}

// TestKubectlPreferencesAreTheUsers: agents switch kubectl downloads on and off, but only the GUI
// sets the path to kubectl, the directory for binaries and the mirror; nothing is written for a
// refused origin.
func TestKubectlPreferencesAreTheUsers(t *testing.T) {
	s, cfg, _ := settingsSession(t)
	var res commands.PreferencesResult
	if err := dispatch(t, s, engine.OriginExternalAgent, "preferences.set", map[string]any{"kubectlDownload": false}, &res); err != nil {
		t.Fatalf("an agent switching downloads off: %v", err)
	}
	if res.Preferences.KubectlDownload {
		t.Error("downloads still on")
	}
	before, err := os.ReadFile(filepath.Join(cfg, engine.PreferencesFile))
	if err != nil {
		t.Fatal(err)
	}
	k := filepath.Join(t.TempDir(), "kubectl")
	if err := os.WriteFile(k, []byte(testkit.FakeKubectl("v1.36.1")), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		for _, p := range []map[string]any{
			{"kubectlPath": k}, {"kubectlDir": t.TempDir()}, {"kubectlMirror": "https://mirror.example.com/k8s"},
			{"kubectlDownload": true, "kubectlMirror": ""},
		} {
			if err := dispatch(t, s, origin, "preferences.set", p, nil); !errors.Is(err, engine.ErrKubectlGUIOnly) {
				t.Errorf("%s %v: %v, want ErrKubectlGUIOnly", origin, p, err)
			}
		}
	}
	if after, _ := os.ReadFile(filepath.Join(cfg, engine.PreferencesFile)); string(after) != string(before) {
		t.Errorf("refused changes were written:\n%s", after)
	}

	dir := t.TempDir()
	gui := map[string]any{"kubectlPath": k, "kubectlDir": dir, "kubectlMirror": "https://mirror.example.com/k8s/", "kubectlDownload": true}
	if err := dispatch(t, s, engine.OriginGUI, "preferences.set", gui, &res); err != nil {
		t.Fatalf("the GUI: %v", err)
	}
	if p := res.Preferences; p.KubectlPath != k || p.KubectlDir != dir || p.KubectlMirror != "https://mirror.example.com/k8s/" || !p.KubectlDownload {
		t.Errorf("after the GUI: %+v", p)
	}
	for _, bad := range []map[string]any{
		{"kubectlPath": filepath.Join(t.TempDir(), "missing")},
		{"kubectlPath": "kubectl"},
		{"kubectlDir": "relative"},
		{"kubectlMirror": "http://dl.k8s.io/release"},
	} {
		if err := dispatch(t, s, engine.OriginGUI, "preferences.set", bad, nil); err == nil {
			t.Errorf("the GUI set %v", bad)
		}
	}
}
