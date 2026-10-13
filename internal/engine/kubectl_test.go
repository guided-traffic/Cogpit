//go:build !windows

package engine

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/kubectl"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// writeFakeKubectl writes testkit.FakeKubectl(gitVersion) to dir/kubectl.
func writeFakeKubectl(t *testing.T, dir, gitVersion string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "kubectl")
	if err := os.WriteFile(path, []byte(testkit.FakeKubectl(gitVersion)), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// kubectlSession is a session whose kubectl on PATH is onPath ("" for none), with a directory for
// binaries of its own and downloads from rel.
func kubectlSession(t *testing.T, onPath string, rel *testkit.KubectlReleases) (*Session, string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "kubectl")
	hooks := KubectlHooks{LookPath: func(string) (string, error) {
		if onPath == "" {
			return "", exec.ErrNotFound
		}
		return onPath, nil
	}}
	if rel != nil {
		hooks.Downloader = rel.Downloader
	}
	cfg := t.TempDir()
	s := New(Options{Preferences: NewPreferencesStore(cfg), Kubectl: NewKubectl(bin, hooks)})
	t.Cleanup(s.Close)
	return s, bin
}

// TestCheckKubectl is the decision when a terminal opens: the kubectl it finds against the
// cluster's version, a release Cogpit downloaded, the offer and the tab's line.
func TestCheckKubectl(t *testing.T) {
	ctx := context.Background()
	tools := t.TempDir()
	k136 := writeFakeKubectl(t, filepath.Join(tools, "136"), "v1.36.0")
	k130 := writeFakeKubectl(t, filepath.Join(tools, "130"), "v1.30.2")
	broken := filepath.Join(tools, "broken", "kubectl")
	if err := os.MkdirAll(filepath.Dir(broken), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(broken, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Within one minor version: nothing changes, no line.
	s, _ := kubectlSession(t, k136, nil)
	c, first := s.checkKubectl(ctx, "v1.36.1")
	if c == nil || !c.Supported || first != "" || c.Offer != "" || c.Message != "" || c.FoundVersion != "1.36.0" || c.Found != k136 {
		t.Errorf("supported: %+v, first %q", c, first)
	}
	if c, _ := s.checkKubectl(ctx, "v1.37.4-eks-1a2b3c"); !c.Supported || c.Server != "1.37.4" {
		t.Errorf("one minor newer, an EKS version: %+v", c)
	}

	// Skewed: the offer.
	s, bin := kubectlSession(t, k130, nil)
	c, first = s.checkKubectl(ctx, "v1.36.1")
	if c.Supported || first != "" || c.Offer != "1.36.1" ||
		c.Message != "kubectl 1.36.1 matches this cluster; the kubectl on PATH is 1.30.2, more than one minor version away." {
		t.Errorf("skewed: %+v", c)
	}
	// A release downloaded earlier that suits the cluster goes first, and the line says so.
	installed := writeFakeKubectl(t, filepath.Join(bin, "1.35.2"), "v1.35.2")
	writeFakeKubectl(t, filepath.Join(bin, "1.30.9"), "v1.30.9")
	c, first = s.checkKubectl(ctx, "v1.36.1")
	if first != installed || c.First != installed || c.FirstVersion != "1.35.2" || c.Offer != "" ||
		!strings.HasPrefix(c.Message, "kubectl 1.35.2, downloaded by Cogpit for this cluster's 1.36.1, comes first on this terminal's PATH; the kubectl on PATH is 1.30.2") {
		t.Errorf("installed: %+v, first %q", c, first)
	}

	// No kubectl on PATH, or one whose version cannot be read.
	s, _ = kubectlSession(t, "", nil)
	if c, _ := s.checkKubectl(ctx, "v1.36.1"); c.Offer != "1.36.1" || !strings.Contains(c.Message, "there is no kubectl on PATH") {
		t.Errorf("none: %+v", c)
	}
	s, _ = kubectlSession(t, broken, nil)
	if c, _ := s.checkKubectl(ctx, "v1.36.1"); c.Offer != "1.36.1" || c.FoundError == "" || !strings.Contains(c.Message, "could not be read") {
		t.Errorf("unreadable: %+v", c)
	}

	// Downloading off: the offer is a note.
	s, _ = kubectlSession(t, k130, nil)
	if _, err := s.ChangePreferences(func(p *Preferences) error { p.KubectlDownload = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if c, _ := s.checkKubectl(ctx, "v1.36.1"); !c.DownloadOff || !strings.HasSuffix(c.Message, "Downloading is off in Preferences › Kubernetes.") {
		t.Errorf("off: %+v", c)
	}

	// The Preferences' path to kubectl replaces PATH's and leads the terminal's PATH.
	s, _ = kubectlSession(t, k130, nil)
	if _, err := s.ChangePreferences(func(p *Preferences) error { p.KubectlPath = k136; return nil }); err != nil {
		t.Fatal(err)
	}
	if c, first := s.checkKubectl(ctx, "v1.36.1"); !c.Supported || !c.FromPreferences || first != k136 || c.Message != "" {
		t.Errorf("preferences, supported: %+v, %q", c, first)
	}
	if c, first := s.checkKubectl(ctx, "v1.38.0"); c.Supported || first != k136 || c.Offer != "1.38.0" ||
		!strings.Contains(c.Message, "the kubectl set in Preferences is 1.36.0") {
		t.Errorf("preferences, skewed: %+v, %q", c, first)
	}
	if c, first := s.checkKubectl(ctx, "not-a-version"); c != nil || first != k136 {
		t.Errorf("no server version: %+v, %q", c, first)
	}

	// Without kubectl state there is no check.
	plain := New(Options{})
	defer plain.Close()
	if c, first := plain.checkKubectl(ctx, "v1.36.1"); c != nil || first != "" {
		t.Errorf("no state: %+v %q", c, first)
	}
}

// openTerminalTab adds a terminal tab with the check a terminal of a v1.36.1 cluster gets.
func openTerminalTab(t *testing.T, s *Session, id string) {
	t.Helper()
	c, _ := s.checkKubectl(context.Background(), "v1.36.1")
	s.UpdateUI(func(u *UIState) {
		u.Dock = append(u.Dock, DockTab{ID: id, Type: DockTerminal, Title: id, Context: "dev"})
	})
	s.setKubectlCheck(id, c)
}

// TestDownloadKubectl downloads the offered release from a local release host into the directory
// for binaries, shows the progress and the outcome in every tab that offers it, and has the next
// terminal find it first.
func TestDownloadKubectl(t *testing.T) {
	rel := testkit.NewKubectlReleases(t)
	binary := []byte(testkit.FakeKubectl("v1.36.1") + "#" + strings.Repeat("x", 2<<20) + "\n")
	rel.Publish(t, "1.36.1", runtime.GOOS, runtime.GOARCH, binary)
	k130 := writeFakeKubectl(t, t.TempDir(), "v1.30.2")
	s, bin := kubectlSession(t, k130, rel)
	openTerminalTab(t, s, "terminal-1")
	openTerminalTab(t, s, "terminal-2")

	var states []string
	cancel := s.Subscribe(func(e Event) {
		if e.Type == EventUIState && e.UIState != nil {
			if tab, ok := e.UIState.DockTab("terminal-2"); ok && tab.Kubectl != nil && tab.Kubectl.Download != nil {
				states = append(states, tab.Kubectl.Download.State)
			}
		}
	})
	path, err := s.DownloadKubectl(context.Background(), "terminal-1")
	cancel()
	if err != nil {
		t.Fatalf("DownloadKubectl: %v", err)
	}
	if want := filepath.Join(bin, "1.36.1", "kubectl"); path != want {
		t.Errorf("path %s, want %s", path, want)
	}
	if got, _ := os.ReadFile(path); string(got) != string(binary) {
		t.Error("the file is not the release")
	}
	if len(states) < 2 || states[0] != KubectlRunning || states[len(states)-1] != KubectlDone {
		t.Errorf("terminal-2's states %v", states)
	}
	for _, id := range []string{"terminal-1", "terminal-2"} {
		tab, _ := s.UIState().DockTab(id)
		if tab.Kubectl.Download == nil || tab.Kubectl.Download.Path != path ||
			tab.Kubectl.Message != "kubectl 1.36.1 is downloaded and verified: terminals opened from now on find it first." {
			t.Errorf("%s: %+v", id, tab.Kubectl)
		}
	}
	// The next terminal finds it first.
	if c, first := s.checkKubectl(context.Background(), "v1.36.1"); first != path || c.Offer != "" {
		t.Errorf("after the download: %+v, %q", c, first)
	}
	if v, err := kubectl.NewProber().ClientVersion(context.Background(), path); err != nil || v.String() != "1.36.1" {
		t.Errorf("the downloaded kubectl answers %v, %v", v, err)
	}
}

// TestDownloadKubectlRefusals: a tampered release fails and the tab says so; a tab without an
// offer, downloading off and an unknown tab are refused before anything is fetched.
func TestDownloadKubectlRefusals(t *testing.T) {
	rel := testkit.NewKubectlReleases(t)
	rel.Publish(t, "1.36.1", runtime.GOOS, runtime.GOARCH, []byte(testkit.FakeKubectl("v1.36.1")))
	at := "/release/v1.36.1/bin/" + runtime.GOOS + "/" + runtime.GOARCH + "/kubectl"
	rel.Set(at, []byte("#!/bin/sh\necho tampered\n"))
	k130 := writeFakeKubectl(t, t.TempDir(), "v1.30.2")
	s, bin := kubectlSession(t, k130, rel)
	openTerminalTab(t, s, "terminal-1")
	if _, err := s.DownloadKubectl(context.Background(), "terminal-1"); !errors.Is(err, kubectl.ErrChecksum) {
		t.Fatalf("a tampered release: %v", err)
	}
	tab, _ := s.UIState().DockTab("terminal-1")
	if tab.Kubectl.Download == nil || tab.Kubectl.Download.State != KubectlFailed ||
		!strings.HasPrefix(tab.Kubectl.Message, "Downloading kubectl 1.36.1 failed: ") {
		t.Errorf("after the failure: %+v", tab.Kubectl)
	}
	if in := kubectl.Installed(bin, runtime.GOOS); len(in) != 0 {
		t.Errorf("installed after a failure: %+v", in)
	}

	asked := len(rel.Requests())
	s2, _ := kubectlSession(t, writeFakeKubectl(t, t.TempDir(), "v1.36.0"), rel)
	openTerminalTab(t, s2, "terminal-1") // supported: no offer
	if err := s2.KubectlDownloadable("terminal-1"); err == nil {
		t.Error("a tab without an offer can download")
	}
	if err := s2.KubectlDownloadable("terminal-9"); err == nil {
		t.Error("an unknown tab can download")
	}
	if _, err := s.ChangePreferences(func(p *Preferences) error { p.KubectlDownload = false; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DownloadKubectl(context.Background(), "terminal-1"); err == nil || !strings.Contains(err.Error(), "off") {
		t.Errorf("downloading off: %v", err)
	}
	if len(rel.Requests()) != asked {
		t.Errorf("refused downloads fetched %v", rel.Requests()[asked:])
	}
}

// TestKubectlPreferences: the fields' defaults, bounds and normalisation.
func TestKubectlPreferences(t *testing.T) {
	if d := DefaultPreferences(); !d.KubectlDownload || d.KubectlMirror != "" || d.KubectlDir != "" || d.KubectlPath != "" {
		t.Errorf("defaults %+v", d)
	}
	for _, bad := range []Preferences{
		{KubectlMirror: "http://dl.k8s.io/release"},
		{KubectlMirror: "https://u:p@mirror.example.com"},
		{KubectlDir: "relative/dir"},
		{KubectlPath: "kubectl"},
		{KubectlPath: "/usr/bin/kubectl\n-rm"},
	} {
		p := DefaultPreferences()
		p.KubectlMirror, p.KubectlDir, p.KubectlPath = bad.KubectlMirror, bad.KubectlDir, bad.KubectlPath
		if err := p.Check(); err == nil {
			t.Errorf("%+v passed the check", bad)
		}
		p.normalize()
		if p.KubectlMirror != "" || p.KubectlDir != "" || p.KubectlPath != "" {
			t.Errorf("normalize kept %+v", p)
		}
	}
	good := DefaultPreferences()
	good.KubectlMirror, good.KubectlDir, good.KubectlPath = "https://mirror.example.com/k8s", "/opt/kubectl", "/usr/local/bin/kubectl"
	if err := good.Check(); err != nil {
		t.Errorf("good: %v", err)
	}
	k := writeFakeKubectl(t, t.TempDir(), "v1.36.1")
	if err := CheckKubectlPath(k); err != nil {
		t.Errorf("CheckKubectlPath(%s): %v", k, err)
	}
	if err := CheckKubectlPath(filepath.Join(t.TempDir(), "kubectl")); err == nil {
		t.Error("a missing file")
	}
	// A preferences file without the fields (written before they existed) keeps downloads on.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, PreferencesFile), []byte(`{"version":1,"theme":"light"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	s := New(Options{Preferences: NewPreferencesStore(dir)})
	defer s.Close()
	if p, _ := s.Preferences(); !p.KubectlDownload || p.Theme != ThemeLight {
		t.Errorf("an older file: %+v", p)
	}
}
