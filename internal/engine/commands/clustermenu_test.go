package commands_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// Disconnect and Help › Welcome are UI commands: an agent runs them as the user does (D-2), the
// workspace shows the welcome page afterwards, ui_state says so, and nothing reaches the cluster.
func TestDisconnectAndWelcomeCommands(t *testing.T) {
	h := newHarness(t, seed(), engine.Options{})
	agent := engine.OriginExternalAgent
	var st commands.StateResult
	state := func(id string, params any) {
		t.Helper()
		st = commands.StateResult{} // omitted fields (activeTab) must not keep the last call's values
		h.must(t, agent, id, params, &st)
	}
	state("ui.state", nil)
	if st.Welcome || st.Connection.State != engine.Connected {
		t.Fatalf("connected: %+v", st.UIState)
	}
	state("ui.welcome.open", nil)
	if !st.Welcome || st.ActiveTab != "" || st.View != "" || len(st.Tabs) != 1 || st.Connection.State != engine.Connected {
		t.Fatalf("ui.welcome.open: %+v", st.UIState)
	}
	state("ui.tab.select", map[string]any{"index": 1})
	if st.Welcome || st.View != "pods" {
		t.Fatalf("back to the tab: %+v", st.UIState)
	}

	h.writer.mu.Lock()
	writes := len(h.writer.calls)
	h.writer.mu.Unlock()
	state("cluster.disconnect", nil)
	if st.Connection != (engine.Connection{State: engine.Disconnected}) || !st.Welcome || h.s.Backend() != nil || st.Context != "test" {
		t.Fatalf("cluster.disconnect: %+v", st.UIState)
	}
	h.writer.mu.Lock()
	if len(h.writer.calls) != writes {
		t.Errorf("a disconnect wrote to the cluster: %v", h.writer.calls[writes:])
	}
	h.writer.mu.Unlock()
	var dis *engine.DisabledError
	if err := h.run(t, agent, "cluster.disconnect", nil, nil); !errors.As(err, &dis) || !strings.Contains(dis.Reason, "no cluster connected") {
		t.Errorf("a second disconnect: %v", err)
	}
	state("ui.welcome.open", nil) // the page shows already: nothing to leave
	if !st.Welcome {
		t.Errorf("welcome while disconnected: %+v", st.UIState)
	}
}

// writeKubeconfig writes a kubeconfig file defining the contexts, on one cluster, owner-only.
func writeKubeconfig(t *testing.T, path string, contexts ...string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("apiVersion: v1\nkind: Config\nclusters:\n- {name: c, cluster: {server: \"https://127.0.0.1:1\"}}\n" +
		"users:\n- {name: u, user: {token: t}}\ncontexts:\n")
	for _, c := range contexts {
		b.WriteString("- {name: " + c + ", context: {cluster: c, user: u}}\n")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Remove from Cogpit unlinks a kubeconfig file the user added, never the file itself, and only in
// Cogpit's window; a context of the default ~/.kube or of a file $KUBECONFIG names is refused with
// the reason, for every origin, and so is the demo cluster. cluster.links tells which is which.
func TestClusterRemove(t *testing.T) {
	root := t.TempDir()
	kube := filepath.Join(root, "home", ".kube")
	writeKubeconfig(t, filepath.Join(kube, "config"), "main")
	env := filepath.Join(root, "env.yaml")
	writeKubeconfig(t, env, "from-env")
	added := filepath.Join(root, "work", "team.yaml")
	writeKubeconfig(t, added, "team-a", "team-b")
	content, err := os.ReadFile(added)
	if err != nil {
		t.Fatal(err)
	}
	k := engine.NewKubeconfigSources(filepath.Join(t.TempDir(), "Cogpit"), engine.WithDefaultFolder(kube), engine.WithSessionFiles([]string{env}))
	if _, _, err := k.AddFile(added); err != nil {
		t.Fatal(err)
	}
	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{Sources: k, IncludeDemo: true}})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	gui, agent := engine.OriginGUI, engine.OriginExternalAgent

	var links commands.LinksResult
	if err := dispatch(t, s, agent, "cluster.links", nil, &links); err != nil {
		t.Fatal(err)
	}
	removable := map[string]bool{}
	for _, l := range links.Links {
		removable[l.Context] = l.Removable
	}
	want := map[string]bool{"main [.kube/config]": false, "from-env [env.yaml]": false, "team-a [team.yaml]": true, "team-b [team.yaml]": true}
	if len(removable) != len(want) {
		t.Errorf("links %+v", links.Links)
	}
	for id, r := range want {
		if got, ok := removable[id]; !ok || got != r {
			t.Errorf("%s: removable %v (listed %v), want %v", id, got, ok, r)
		}
	}

	for id, reason := range map[string]string{
		"main [.kube/config]": "in the synced folder", "from-env [env.yaml]": "$KUBECONFIG", engine.DemoContext: "demo cluster",
	} {
		var dis *engine.DisabledError
		err := dispatch(t, s, gui, "cluster.remove", map[string]string{"context": id}, nil)
		if !errors.As(err, &dis) || !strings.Contains(dis.Reason, reason) {
			t.Errorf("remove %s from %s: %v, want refused: %s", id, gui, err, reason)
		}
		// An agent is refused as for every GUI-only command, before the reason (engine/guionly.go).
		if err := dispatch(t, s, agent, "cluster.remove", map[string]string{"context": id}, nil); !errors.Is(err, engine.ErrGUIOnly) {
			t.Errorf("remove %s from %s: %v, want ErrGUIOnly", id, agent, err)
		}
	}
	for _, origin := range []engine.Origin{agent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		if err := dispatch(t, s, origin, "cluster.remove", map[string]string{"context": "team-a [team.yaml]"}, nil); !errors.Is(err, engine.ErrGUIOnly) {
			t.Errorf("remove from %s: %v, want ErrGUIOnly", origin, err)
		}
	}

	var removed engine.ContextLink
	if err := dispatch(t, s, gui, "cluster.remove", map[string]string{"context": "team-a [team.yaml]"}, &removed); err != nil {
		t.Fatal(err)
	}
	if removed.File != added || !slices.Equal(removed.Contexts, []string{"team-a [team.yaml]", "team-b [team.yaml]"}) {
		t.Errorf("removed %+v", removed)
	}
	ctxs, err := s.Contexts()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range ctxs {
		if strings.HasSuffix(c.Name, "[team.yaml]") {
			t.Errorf("%s is still offered", c.Name)
		}
	}
	if after, err := os.ReadFile(added); err != nil || !bytes.Equal(after, content) {
		t.Errorf("the kubeconfig file changed: %v", err)
	}
}
