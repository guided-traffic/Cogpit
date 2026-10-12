package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestRunFlagConnectsABareContextName: `make run` starts Cogpit with `--kubeconfig <kind file>
// --context <name>`. The file is a session source (D-28), read alone, and the bare name resolves to
// its id "<name> [<file>]", which the connection and the UI state then carry. A ~/.kube folder
// next to it does not interfere, and nothing is merged.
func TestRunFlagConnectsABareContextName(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: sourceToken})
	defer srv.Close()
	file := srv.WriteKubeconfig(t, "kind-cogpit", "")

	home := filepath.Join(t.TempDir(), ".kube") // the developer's "~/.kube", with a context of the same name
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	other := srv.WriteKubeconfig(t, "kind-cogpit", "")
	if err := os.Rename(other, filepath.Join(home, "config")); err != nil {
		t.Fatal(err)
	}
	k := engine.NewKubeconfigSources(filepath.Join(t.TempDir(), "Cogpit"), engine.WithDefaultFolder(home), engine.WithSessionFiles([]string{file}))
	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{Sources: k}})
	defer s.Close()

	// The name is in two files: asking for it bare is refused, naming both.
	if err := s.Connect(t.Context(), "kind-cogpit"); err == nil {
		t.Fatal("an ambiguous bare name connected")
	}
	id := engine.SourceContextID("kind-cogpit", "kubeconfig")
	if err := s.Connect(t.Context(), id); err != nil {
		t.Fatal(err)
	}
	if got := s.UIState().Context; got != id {
		t.Errorf("UI context %q, want %q", got, id)
	}
	if b := s.Backend(); b == nil || b.SourceFile != file || b.Context != id {
		t.Errorf("backend %+v", b)
	}

	// As `make run` has it (no other file defines the name): the bare name resolves.
	k2 := engine.NewKubeconfigSources(filepath.Join(t.TempDir(), "Cogpit"), engine.WithSessionFiles([]string{file}))
	s2 := engine.New(engine.Options{Connector: &engine.ClusterConnector{Sources: k2}})
	defer s2.Close()
	if err := s2.Connect(t.Context(), "kind-cogpit"); err != nil {
		t.Fatal(err)
	}
	if got := s2.UIState().Context; got != id {
		t.Errorf("UI context %q, want %q", got, id)
	}
}
