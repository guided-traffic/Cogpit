package fyneui

import (
	"context"
	"slices"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/guided-traffic/cogpit/internal/engine"
)

func menuLabels(m *fyne.Menu) []string {
	var out []string
	for _, it := range m.Items {
		out = append(out, it.Label)
	}
	return out
}

func menuItem(t *testing.T, m *fyne.Menu, label string) *fyne.MenuItem {
	t.Helper()
	for _, it := range m.Items {
		if it.Label == label {
			return it
		}
	}
	t.Fatalf("menu %s has no %q: %v", m.Label, label, menuLabels(m))
	return nil
}

// TestViewMenuHidesKindsNotListable: the View menu offers the kinds the sidebar shows, no kind
// the access reviews say the user may not list, and follows the answers as the sidebar does.
func TestViewMenuHidesKindsNotListable(t *testing.T) {
	f := connectedFake(t, 50)
	setListable(f, map[string]bool{"nodes": false, "deployments": false, "pods": true, "namespaces": true, "events": true})
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	if got := menuLabels(sh.mainMenu().Items[1]); !slices.Equal(got, []string{"Pods", "Services", "Namespaces", "Events"}) {
		t.Errorf("View menu %v, want Nodes and Deployments left out and Services, not known yet, offered", got)
	}

	setListable(f, map[string]bool{"nodes": false, "deployments": true, "pods": true})
	f.emit(engine.Event{Type: engine.EventKinds})
	q.until(t, "Deployments back", func() bool {
		return slices.Contains(menuLabels(sh.mainMenu().Items[1]), "Deployments")
	})
	if got := menuLabels(sh.mainMenu().Items[1]); slices.Contains(got, "Nodes") {
		t.Errorf("View menu %v offers Nodes", got)
	}
}

// TestRefreshAccessMenuItem: Cluster › Refresh Access Reviews dispatches cluster.refreshAccess
// on a real cluster; it is disabled on the demo cluster, which has no access reviews, and
// without a cluster.
func TestRefreshAccessMenuItem(t *testing.T) {
	f := connectedFake(t, 20)
	f.ui.Context, f.ui.Connection = "kind-cogpit", engine.Connection{State: engine.Connected}
	f.handlers["cluster.refreshAccess"] = func(context.Context, engine.Invocation) (any, error) {
		return map[string]any{"context": "kind-cogpit", "namespaces": []string{"default"}}, nil
	}
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })

	item := menuItem(t, sh.mainMenu().Items[0], "Refresh Access Reviews")
	if item.Disabled {
		t.Fatal("disabled on a connected cluster")
	}
	item.Action()
	q.until(t, "the refresh", func() bool { return len(f.callsOf("cluster.refreshAccess")) == 1 })
	if c := f.callsOf("cluster.refreshAccess")[0]; c.Origin != engine.OriginGUI {
		t.Errorf("origin %s", c.Origin)
	}

	f.update(func(u *engine.UIState) {
		u.Context, u.Connection = "demo", engine.Connection{State: engine.Connected, Demo: true}
	})
	q.until(t, "the demo cluster", func() bool { return sh.ui.Connection.Demo })
	if !menuItem(t, sh.mainMenu().Items[0], "Refresh Access Reviews").Disabled {
		t.Error("enabled on the demo cluster")
	}
	f.update(func(u *engine.UIState) { u.Connection = engine.Connection{State: engine.Disconnected} })
	q.until(t, "no cluster", func() bool { return sh.ui.Connection.State == engine.Disconnected })
	if !menuItem(t, sh.mainMenu().Items[0], "Refresh Access Reviews").Disabled {
		t.Error("enabled without a cluster")
	}
}
