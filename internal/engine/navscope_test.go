package engine

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// scopeSession is a session over a small cluster, not connected yet, and its connector.
func scopeSession(t *testing.T) (*Session, *fakeConnector) {
	t.Helper()
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("shop", "web-0"))
	src.Set(testkit.ServicesGVR, testkit.Service("shop", "checkout", "10.96.0.1"))
	return newTestSession(t, src, 10*time.Millisecond)
}

func (f *fakeConnector) connectCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connects
}

func wantScope(t *testing.T, u UIState, scope string) {
	t.Helper()
	if u.NavigatorScope != scope || u.Scope() != scope {
		t.Fatalf("scope %q (reads %q), want %q; connection %s %q", u.NavigatorScope, u.Scope(), scope, u.Connection.State, u.Context)
	}
}

// The navigator starts at the section; every connect drills into its cluster as it starts; going up
// to the section and back into the connected cluster keeps the connection; a disconnect and a
// failed connect go back to the section.
func TestNavScopeFollowsTheConnection(t *testing.T) {
	s, fc := scopeSession(t)
	ctx := context.Background()
	wantScope(t, s.UIState(), NavScopeLocal)

	if err := s.Connect(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	wantScope(t, s.UIState(), NavScopeCluster("test"))

	u, err := s.SetNavigatorScope(ctx, NavScopeLocal)
	if err != nil {
		t.Fatal(err)
	}
	wantScope(t, u, NavScopeLocal)
	if u.Context != "test" || u.Connection.State != Connected || u.View != "pods" || u.ActiveTab == "" {
		t.Errorf("up to the section changed the connection or the view: %s %q view %q tab %q", u.Connection.State, u.Context, u.View, u.ActiveTab)
	}
	s.OpenTab("services", false) // the views go on at the section
	wantScope(t, s.UIState(), NavScopeLocal)

	n := fc.connectCount()
	u, err = s.SetNavigatorScope(ctx, NavScopeCluster("test"))
	if err != nil {
		t.Fatal(err)
	}
	wantScope(t, u, NavScopeCluster("test"))
	if fc.connectCount() != n || u.View != "services" {
		t.Errorf("into the connected cluster: %d connects (was %d), view %q", fc.connectCount(), n, u.View)
	}

	// From the section, a connect from anywhere (the picker, the hotbar, an agent) drills in.
	s.SetNavigatorScope(ctx, NavScopeLocal)
	if err := s.Connect(ctx, "prod"); err != nil {
		t.Fatal(err)
	}
	wantScope(t, s.UIState(), NavScopeCluster("prod"))
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	wantScope(t, s.UIState(), NavScopeLocal)

	// Into a cluster that is not connected: connected first, as cluster.connect does.
	u, err = s.SetNavigatorScope(ctx, NavScopeCluster("test"))
	if err != nil {
		t.Fatal(err)
	}
	wantScope(t, u, NavScopeCluster("test"))
	if u.Context != "test" || u.Connection.State != Connected || fc.connectCount() != n+2 {
		t.Errorf("drilled into test: %s %q, %d connects", u.Connection.State, u.Context, fc.connectCount())
	}

	// A failed connect: the state says why, the navigator shows the section, no error.
	u, err = s.SetNavigatorScope(ctx, NavScopeCluster("broken"))
	if err != nil {
		t.Fatal(err)
	}
	wantScope(t, u, NavScopeLocal)
	if u.Connection.State != ConnectError || u.Context != "broken" {
		t.Errorf("a failed connect: %s %q", u.Connection.State, u.Context)
	}
}

// A cluster scope always names the connected cluster; a scope the engine does not know is refused.
func TestNavScopeIsChecked(t *testing.T) {
	s, _ := scopeSession(t)
	ctx := context.Background()
	for _, bad := range []string{"", "section:cloud", "cluster:", "cluster: ", "kind-dev"} {
		if _, err := s.SetNavigatorScope(ctx, bad); !errors.Is(err, ErrNavScope) {
			t.Errorf("%q: %v", bad, err)
		}
	}
	if err := s.Connect(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	u := s.UpdateUI(func(u *UIState) { u.NavigatorScope = NavScopeCluster("elsewhere") })
	wantScope(t, u, NavScopeCluster("test"))

	// A state without a scope (the zero value) reads as the engine would set it.
	if got := (UIState{}).Scope(); got != NavScopeLocal {
		t.Errorf("zero state: %q", got)
	}
	if got := (UIState{Context: "kind-dev", Connection: Connection{State: Connecting}}).ScopeCluster(); got != "kind-dev" {
		t.Errorf("connecting state: %q", got)
	}
	if got := (UIState{Context: "kind-dev", Connection: Connection{State: ConnectError}}).ScopeCluster(); got != "" {
		t.Errorf("failed state: %q", got)
	}
	if id, err := ParseNavScope("cluster:kind-dev [.kube/config]"); err != nil || id != "kind-dev [.kube/config]" {
		t.Errorf("parse: %q %v", id, err)
	}
}

// The tab bar shows the tabs of the scope: every tab at the section, the cluster's at a cluster;
// the n-th tab, next and previous count those, so that ⌘n and ⌘⌥→ never leave the cluster shown.
func TestTabsInScope(t *testing.T) {
	s, _ := scopeSession(t)
	ctx := context.Background()
	if err := s.Connect(ctx, "test"); err != nil {
		t.Fatal(err)
	}
	s.OpenTab("services", true)
	if err := s.Connect(ctx, "prod"); err != nil {
		t.Fatal(err)
	}
	s.OpenTab("deployments", true)
	u := s.UIState()
	wantTabs(t, u, "tab-1 pods@prod p", "tab-3 deployments@prod *", "tab-2 services@test")
	ids := func(tabs []ViewTab) []string {
		var out []string
		for _, t := range tabs {
			out = append(out, t.ID)
		}
		return out
	}
	if got := ids(u.TabsInScope()); !slices.Equal(got, []string{"tab-1", "tab-3"}) {
		t.Errorf("prod's tabs %q", got)
	}
	u, err := s.CycleTab(ctx, 1)
	if err != nil || u.ActiveTab != "tab-1" || u.Context != "prod" {
		t.Errorf("next after prod's last: %v, %q on %q", err, u.ActiveTab, u.Context)
	}
	u, err = s.SelectTabAt(ctx, 0, true)
	if err != nil || u.ActiveTab != "tab-3" {
		t.Errorf("prod's last tab: %v, %q", err, u.ActiveTab)
	}
	if _, err := s.SelectTabAt(ctx, 3, false); !errors.Is(err, ErrUnknownTab) {
		t.Errorf("tab 3 of prod's 2: %v", err)
	}

	u, _ = s.SetNavigatorScope(ctx, NavScopeLocal)
	if got := ids(u.TabsInScope()); !slices.Equal(got, []string{"tab-1", "tab-3", "tab-2"}) {
		t.Errorf("the section's tabs %q", got)
	}
	u, err = s.SelectTabAt(ctx, 3, false) // test's tab: connects, and the connect drills in
	if err != nil || u.ActiveTab != "tab-2" || u.Context != "test" || u.View != "services" {
		t.Fatalf("tab 3 at the section: %v, %q on %q view %q", err, u.ActiveTab, u.Context, u.View)
	}
	wantScope(t, u, NavScopeCluster("test"))
	if got := ids(u.TabsInScope()); !slices.Equal(got, []string{"tab-2"}) {
		t.Errorf("test's tabs %q", got)
	}
}
