package fyneui

import "testing"

// TestCreateButtonHiddenOnLocalAndVirtualKinds: Port Forwarding and Helm Releases have no "+":
// no manifest creates their objects.
func TestCreateButtonHiddenOnLocalAndVirtualKinds(t *testing.T) {
	f := connectedFake(t, 20)
	f.mu.Lock()
	f.kinds = append(f.kinds,
		kindInfo{Kind: "portforwards", Title: "Port Forwarding", APIKind: "PortForward", Group: "cogpit.io", Version: "v1", Namespaced: true, Local: true},
		kindInfo{Kind: "helmreleases", Title: "Releases", APIKind: "HelmRelease", Group: "cogpit.io", Version: "v1", Namespaced: true, Virtual: true})
	f.mu.Unlock()
	sh, _, q := newTestShell(t, f, Options{})
	q.until(t, "kinds", func() bool { return len(sh.kinds) > 0 })
	q.until(t, "the + on Pods", func() bool { return sh.view.create.Visible() })

	for _, view := range []string{"portforwards", "helmreleases", "pods"} {
		sh.dispatch("ui.openView", openViewParams{View: view}, nil)
		want := view == "pods"
		q.until(t, "the + on "+view, func() bool { return sh.ui.View == view && sh.view.create.Visible() == want })
	}
}
