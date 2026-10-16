package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The short name (D-68): the part after the last "@", else the whole name; nothing after the last
// "@" keeps the whole name.
func TestShortContextName(t *testing.T) {
	for name, want := range map[string]string{
		"kubernetes-admin@prod-eu": "prod-eu",
		"a@b@c":                    "c",
		"@prod":                    "prod",
		"admin@":                   "admin@",
		"@":                        "@",
		"admin@  ":                 "admin@  ",
		"user@ cluster-1":          "cluster-1",
		"kind-dev":                 "kind-dev",
		"":                         "",
	} {
		if got := ShortContextName(name); got != want {
			t.Errorf("ShortContextName(%q) = %q, want %q", name, got, want)
		}
	}
}

// ClusterName is the one name of a context everywhere: the display name the user set wins, else the
// short name of the context's name without its file.
func TestClusterNameIsTheShortName(t *testing.T) {
	for _, tc := range []struct {
		id   string
		cs   ClusterSettings
		want string
	}{
		{"kubernetes-admin@prod-eu [dev.yaml]", ClusterSettings{}, "prod-eu"},
		{"kubernetes-admin@prod-eu [dev.yaml]", ClusterSettings{Name: "Production"}, "Production"},
		{"kubernetes-admin@prod-eu", ClusterSettings{}, "prod-eu"},
		{"kind-dev [.kube/config]", ClusterSettings{}, "kind-dev"},
		{"admin@ [x@y.yaml]", ClusterSettings{}, "admin@"},
		{"demo", ClusterSettings{}, "demo"},
	} {
		if got := ClusterName(tc.id, tc.cs); got != tc.want {
			t.Errorf("ClusterName(%q, %+v) = %q, want %q", tc.id, tc.cs, got, tc.want)
		}
	}
}

// userSession is a session on a fake API server whose user is id, its last known users kept in dir.
func userSession(t *testing.T, id *testkit.FakeIdentity, dir string) (*Session, *testkit.FakeAPIServer) {
	t.Helper()
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: "user-token", Identity: id})
	t.Cleanup(srv.Close)
	s := New(Options{
		Connector:       &ClusterConnector{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}},
		ClusterSettings: NewClusterSettingsStore(dir),
	})
	t.Cleanup(s.Close)
	return s, srv
}

func connectLogHas(s *Session, ctx, text string) bool {
	return slices.ContainsFunc(s.ConnectLog(ctx), func(l ConnectLogLine) bool { return strings.Contains(l.Text, text) })
}

func readLastUsers(t *testing.T, dir string) lastUsersDoc {
	t.Helper()
	var doc lastUsersDoc
	data, err := os.ReadFile(filepath.Join(dir, LastUsersFile))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// At the connect the API server says who the user is: the UI state names the username and the
// groups the moment the connection shows Connected, the connect log says so, and the username
// alone is kept on disk, owner-only. Disconnected, the cluster shows its last known user and when
// it was last seen, and a new session (a restart) knows it too.
func TestConnectLearnsTheUser(t *testing.T) {
	dir := t.TempDir()
	s, srv := userSession(t, &testkit.FakeIdentity{Username: "alice@example.com", Groups: []string{"team-a", "system:authenticated"},
		UID: "uid-not-kept", Extra: map[string][]string{"secret-claim": {"not-kept"}}}, dir)
	start := time.Now()
	if err := s.Connect(context.Background(), "fake"); err != nil {
		t.Fatal(err)
	}
	u := s.UIState()
	want := ClusterUser{Context: "fake", Username: "alice@example.com", Groups: []string{"team-a", "system:authenticated"}, Source: UserFromAPIServer}
	if u.User == nil || u.User.Context != want.Context || u.User.Username != want.Username || !slices.Equal(u.User.Groups, want.Groups) ||
		u.User.Source != want.Source || u.User.Reason != "" || !u.User.LastSeen.IsZero() {
		t.Fatalf("user at the connect: %+v", u.User)
	}
	raw, _ := json.Marshal(u)
	if strings.Contains(string(raw), "uid-not-kept") || strings.Contains(string(raw), "secret-claim") {
		t.Errorf("the UI state carries the UID or the extra fields: %s", raw)
	}
	if !connectLogHas(s, "fake", "Logged in as alice@example.com (confirmed by the API server); groups: team-a, system:authenticated") {
		t.Errorf("the connect log: %+v", s.ConnectLog("fake"))
	}
	if n := srv.Requests()["create selfsubjectreviews.authentication.k8s.io"]; n != 1 {
		t.Errorf("%d reviews at the connect, want 1", n)
	}
	doc := readLastUsers(t, dir)
	if got := doc.Users["fake"]; doc.Version != 1 || got.Username != "alice@example.com" || got.Seen.Before(start.Add(-time.Second)) {
		t.Errorf("kept: %+v", doc)
	}
	data, _ := os.ReadFile(filepath.Join(dir, LastUsersFile))
	if strings.Contains(string(data), "team-a") {
		t.Errorf("the groups are kept: %s", data)
	}
	if fi, err := os.Stat(filepath.Join(dir, LastUsersFile)); err != nil || (runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600) {
		t.Errorf("the file: %v, %v", fi.Mode(), err)
	}

	beforeDisconnect := time.Now()
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	u = s.UIState()
	if u.User == nil || u.User.Source != UserLastSeen || u.User.Username != "alice@example.com" || len(u.User.Groups) != 0 ||
		u.User.LastSeen.Before(beforeDisconnect.Add(-time.Second)) {
		t.Fatalf("user after the disconnect: %+v", u.User)
	}
	if got := readLastUsers(t, dir).Users["fake"]; !got.Seen.Equal(u.User.LastSeen) {
		t.Errorf("the disconnect's sighting is not kept: %v, shown %v", got.Seen, u.User.LastSeen)
	}

	// A restart: the next session knows the last known user before it connects.
	next := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	t.Cleanup(next.Close)
	next.UpdateUI(func(u *UIState) { u.Context = "fake" })
	if got := next.UIState().User; got == nil || got.Source != UserLastSeen || got.Username != "alice@example.com" {
		t.Errorf("after a restart: %+v", got)
	}
	next.UpdateUI(func(u *UIState) { u.Context = "never-connected" })
	if got := next.UIState().User; got != nil {
		t.Errorf("a context never connected: %+v", got)
	}
}

// An API server that does not know the review (404, older than 1.27) or refuses it (403): the
// connect succeeds, the kubeconfig's user entry's name stands in, marked, the reason in the connect
// log; nothing is kept as a last known user.
func TestUserFallsBackToTheKubeconfig(t *testing.T) {
	for _, tc := range []struct {
		status int
		reason string
	}{
		{http.StatusNotFound, "does not serve SelfSubjectReview"},
		{http.StatusForbidden, "refused the SelfSubjectReview (403 Forbidden)"},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			dir := t.TempDir()
			s, _ := userSession(t, &testkit.FakeIdentity{Username: "not-shown", Status: tc.status}, dir)
			if err := s.Connect(context.Background(), "fake"); err != nil {
				t.Fatalf("the connect failed with the review: %v", err)
			}
			u := s.UIState()
			if u.Connection.State != Connected || u.User == nil || u.User.Source != UserFromKubeconfig || u.User.Username != "fake" ||
				len(u.User.Groups) != 0 || !strings.Contains(u.User.Reason, tc.reason) {
				t.Fatalf("connection %v, user %+v", u.Connection.State, u.User)
			}
			if !connectLogHas(s, "fake", "The API server did not confirm the user: ") || !connectLogHas(s, "fake", tc.reason) ||
				!connectLogHas(s, "fake", `the kubeconfig's user entry "fake"`) {
				t.Errorf("the connect log: %+v", s.ConnectLog("fake"))
			}
			if _, err := os.Stat(filepath.Join(dir, LastUsersFile)); err == nil {
				t.Errorf("a fallback was kept: %+v", readLastUsers(t, dir))
			}
			if err := s.Disconnect(); err != nil {
				t.Fatal(err)
			}
			if got := s.UIState().User; got != nil {
				t.Errorf("not connected after a fallback: %+v", got)
			}
		})
	}
}

// A renewed login asks again: the user may be another one now. The connection came back through
// the connection watch (Log in again, D-58); the UI state follows the new answer.
func TestLoginRenewsTheUser(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: "user-token", Identity: &testkit.FakeIdentity{Username: "alice", Groups: []string{"a"}}})
	t.Cleanup(srv.Close)
	conn, err := cluster.Connect(context.Background(), cluster.Options{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(conn.Close)
	src := testkit.NewMemorySource()
	p := &fakeProber{}
	s := New(Options{Connector: &fakeConnector{src: src, profiles: testProfiles(), conn: conn, health: p},
		SnapshotInterval: 10 * time.Millisecond, ConnWatch: fastWatch})
	t.Cleanup(s.Close)
	connectTest(t, s)
	if u := s.UIState().User; u == nil || u.Username != "alice" || u.Source != UserFromAPIServer {
		t.Fatalf("user at the connect: %+v", u)
	}
	p.set(func(f *fakeProber) { f.reject = true })
	waitState(t, s, LoginExpired)
	if u := s.UIState().User; u == nil || u.Username != "alice" || u.Source != UserFromAPIServer {
		t.Errorf("while the login expired the connection's user stays: %+v", u)
	}
	srv.SetIdentity(testkit.FakeIdentity{Username: "bob", Groups: []string{"b1", "b2"}})
	p.set(func(f *fakeProber) { f.reject = false })
	if err := s.Login(); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the renewed user", func() bool {
		u := s.UIState().User
		return u != nil && u.Username == "bob" && slices.Equal(u.Groups, []string{"b1", "b2"})
	})
	if !connectLogHas(s, "test", "Logged in again as bob") {
		t.Errorf("the connect log: %+v", s.ConnectLog("test"))
	}
	if n := srv.Requests()["create selfsubjectreviews.authentication.k8s.io"]; n != 2 {
		t.Errorf("%d reviews, want 2 (the connect, the login)", n)
	}
}

// The demo cluster has no user; the shown user follows the context: another cluster's never shows.
func TestSyncUserFollowsTheContext(t *testing.T) {
	var b userBook
	b.setLive(ClusterUser{Context: "a", Username: "alice", Groups: []string{"g"}, Source: UserFromAPIServer}, time.Unix(100, 0))
	b.setLive(ClusterUser{Context: "k", Username: "entry", Source: UserFromKubeconfig, Reason: "refused"}, time.Unix(100, 0))
	now := time.Unix(200, 0)
	u := UIState{Context: "a", Connection: Connection{State: Connecting}}
	u.syncUser(UIState{}, &b, now)
	if u.User == nil || u.User.Source != UserLastSeen || !u.User.LastSeen.Equal(time.Unix(100, 0)) {
		t.Errorf("connecting: %+v", u.User)
	}
	old := u
	u.Connection.State = Connected
	u.syncUser(old, &b, now)
	if u.User == nil || u.User.Source != UserFromAPIServer || u.User.Username != "alice" {
		t.Errorf("connected: %+v", u.User)
	}
	old = u
	u.Connection.State = Interrupted
	u.syncUser(old, &b, now)
	if u.User == nil || u.User.Source != UserFromAPIServer {
		t.Errorf("interrupted: %+v", u.User)
	}
	old = u
	u.Context, u.Connection.State = "b", Connecting
	u.syncUser(old, &b, now)
	if u.User != nil {
		t.Errorf("another cluster shows %+v", u.User)
	}
	if got := b.shown("a", false); got == nil || !got.LastSeen.Equal(now) {
		t.Errorf("a's sighting when b took its place: %+v", got)
	}
	old = u
	u.Context, u.Connection.State = "k", Connected
	u.syncUser(old, &b, now)
	if u.User == nil || u.User.Source != UserFromKubeconfig || u.User.Reason != "refused" {
		t.Errorf("the fallback: %+v", u.User)
	}
	old = u
	u.Connection.State = Disconnected
	u.syncUser(old, &b, now)
	if u.User != nil {
		t.Errorf("a fallback is no last known user: %+v", u.User)
	}
	b.forget("k")
	if got := b.shown("k", true); got != nil {
		t.Errorf("forgotten: %+v", got)
	}
	if (&UIState{}).User.clone() != nil {
		t.Error("clone of nil")
	}
}

// An unreadable file of last known users is set aside at the next write, never a reason not to start.
func TestLastUsersFileUnreadable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, LastUsersFile), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, _ := userSession(t, nil, dir)
	if err := s.Connect(context.Background(), "fake"); err != nil {
		t.Fatal(err)
	}
	if got := readLastUsers(t, dir).Users["fake"]; got.Username != testkit.DefaultFakeIdentity.Username {
		t.Errorf("kept: %+v", got)
	}
	if _, err := os.Stat(filepath.Join(dir, LastUsersFile+".unreadable")); err != nil {
		t.Errorf("the unreadable file was not set aside: %v", err)
	}
}

// The palette opens a hotbar tile by the cluster's short name, or the display name set; its full
// context name is the row's category, where the palette's search finds it (D-68).
func TestPaletteNamesTilesByTheShortName(t *testing.T) {
	r := NewRegistry()
	type slotP struct {
		Slot int `json:"slot"`
		Bank int `json:"bank,omitempty"`
	}
	if err := Register(r, Spec[slotP]{ID: openClusterCommand, Label: "Open Cluster", MenuPath: "Hotbar/Open Cluster", Danger: DangerUI,
		Run: func(context.Context, *Session, slotP) (any, error) { return nil, nil }}); err != nil {
		t.Fatal(err)
	}
	st := UIState{
		Hotbar: &HotbarState{Current: 1, Banks: []HotbarBank{{Name: DefaultHotbarBank, Clusters: []HotbarTile{
			{Context: "kubernetes-admin@prod-eu [dev.yaml]"}, {Context: "admin@staging [s.yaml]"}, {Context: "kind-dev"}}}}},
		ClusterSettings: map[string]ClusterSettings{"admin@staging [s.yaml]": {Name: "Staging"}},
	}
	var rows []string
	for _, e := range r.Palette(nil, PaletteInput{State: st}) {
		if e.Command == openClusterCommand {
			rows = append(rows, e.Label+" ["+e.Category+"] "+e.Shortcut)
		}
	}
	want := []string{"Open prod-eu [kubernetes-admin@prod-eu] F1", "Open Staging [admin@staging] F2", "Open kind-dev [kind-dev] F3"}
	if !slices.Equal(rows, want) {
		t.Errorf("tile entries %q, want %q", rows, want)
	}
	if got := labels(FilterPalette(r.Palette(nil, PaletteInput{State: st}), "kubernetes-admin")); !slices.Equal(got, []string{"Open prod-eu"}) {
		t.Errorf("the search by the full name: %q", got)
	}
}

// The hotbar's tiles carry the name the window shows beside their label (ui_state, hotbar.list).
func TestHotbarTilesCarryTheDisplayName(t *testing.T) {
	s := New(Options{Hotbar: NewHotbar(""), ClusterSettings: NewClusterSettingsStore("")})
	t.Cleanup(s.Close)
	if _, err := s.ChangeClusterSettings("named@ctx", func(cs *ClusterSettings) error { cs.Name = "Named"; return nil }); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChangeHotbar(func(st *HotbarState) error {
		st.Banks[0].Clusters = append(st.Banks[0].Clusters,
			HotbarTile{Context: "kubernetes-admin@prod-eu [dev.yaml]"}, HotbarTile{Context: "kind-dev"}, HotbarTile{Context: "named@ctx"})
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	v, err := s.HotbarView()
	if err != nil {
		t.Fatal(err)
	}
	tiles := v.Banks[0].Clusters
	if tiles[0].Label != "kubernetes-admin@prod-eu" || tiles[0].DisplayName != "prod-eu" || tiles[1].DisplayName != "kind-dev" ||
		tiles[2].DisplayName != "Named" {
		t.Errorf("tiles %+v", tiles)
	}
}
