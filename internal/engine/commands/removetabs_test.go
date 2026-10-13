package commands_test

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// Removing a cluster from Cogpit (cluster.remove) or a kubeconfig source (kubeconfig.remove) closes
// the tabs of every context that leaves Cogpit with it, the active tab too, so that neither the tab
// bar nor an agent's ui_state keeps a tab of a cluster Cogpit no longer offers; the other clusters'
// tabs stay as they are.
func TestRemovedClustersLeaveTheTabBar(t *testing.T) {
	root := t.TempDir()
	kube := filepath.Join(root, "home", ".kube")
	writeKubeconfig(t, filepath.Join(kube, "config"), "main")
	added := filepath.Join(root, "work", "team.yaml")
	writeKubeconfig(t, added, "team-a", "team-b")
	folder := filepath.Join(root, "synced")
	writeKubeconfig(t, filepath.Join(folder, "edge.yaml"), "edge")
	k := engine.NewKubeconfigSources(filepath.Join(t.TempDir(), "Cogpit"), engine.WithDefaultFolder(kube))
	if _, _, err := k.AddFile(added); err != nil {
		t.Fatal(err)
	}
	synced, err := k.AddFolder(folder)
	if err != nil {
		t.Fatal(err)
	}
	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{Sources: k}})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	// team-a's tab is the active one: the cluster was connected and then disconnected.
	s.UpdateUI(func(u *engine.UIState) {
		u.Context = "team-a [team.yaml]"
		u.Tabs = []engine.ViewTab{
			{ID: "tab-1", Context: "main [.kube/config]", View: "pods"},
			{ID: "tab-2", Context: "team-a [team.yaml]", View: "pods", Preview: true},
			{ID: "tab-3", Context: "edge [synced/edge.yaml]", View: "nodes"},
			{ID: "tab-4", Context: "team-b [team.yaml]", View: "services"},
			{ID: "tab-5", Context: "team-a [team.yaml]", View: "deployments"},
		}
		u.ActiveTab = "tab-2"
	})
	tabs := func() []string {
		var out []string
		for _, tab := range s.UIState().Tabs {
			out = append(out, tab.ID+" "+tab.Context)
		}
		return out
	}
	gui := engine.OriginGUI

	if err := dispatch(t, s, gui, "cluster.remove", map[string]string{"context": "team-b [team.yaml]"}, nil); err != nil {
		t.Fatal(err)
	}
	u := s.UIState()
	if got, want := tabs(), []string{"tab-1 main [.kube/config]", "tab-3 edge [synced/edge.yaml]"}; !slices.Equal(got, want) {
		t.Errorf("after cluster.remove: tabs %q, want %q", got, want)
	}
	if u.ActiveTab != "" || u.View != "" || !u.Welcome {
		t.Errorf("after cluster.remove: active tab %q, view %q, welcome %v", u.ActiveTab, u.View, u.Welcome)
	}

	if err := dispatch(t, s, gui, "kubeconfig.remove", map[string]string{"source": synced.ID}, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := tabs(), []string{"tab-1 main [.kube/config]"}; !slices.Equal(got, want) {
		t.Errorf("after kubeconfig.remove: tabs %q, want %q", got, want)
	}
}
