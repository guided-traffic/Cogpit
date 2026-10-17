package engine

import (
	"context"
	"crypto/x509"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// reconnector is the fake connector of the reconnect tests: every connect makes a new backend,
// whose close it records; a connect can be made to fail or to wait.
type reconnector struct {
	*fakeConnector
	plugin bool // the backends log in through a credential plugin

	mu      sync.Mutex
	fail    error
	hold    chan struct{} // a connect waits for it (or its context) while set
	made    []*Backend
	closed  map[*Backend]bool
	entered chan struct{} // receives once a connect that waits began
}

func (r *reconnector) Connect(ctx context.Context, name string) (*Backend, error) {
	r.mu.Lock()
	fail, hold, entered := r.fail, r.hold, r.entered
	r.mu.Unlock()
	if hold != nil {
		if entered != nil {
			entered <- struct{}{}
		}
		select {
		case <-hold:
		case <-ctx.Done():
			return nil, context.Cause(ctx)
		}
	}
	if fail != nil {
		return nil, fail
	}
	b, err := r.fakeConnector.Connect(ctx, name)
	if err != nil {
		return nil, err
	}
	b.CredentialPlugin = r.plugin
	inner := b.Close
	b.Close = func() {
		r.mu.Lock()
		r.closed[b] = true
		r.mu.Unlock()
		inner()
	}
	r.mu.Lock()
	r.made = append(r.made, b)
	r.mu.Unlock()
	return b, nil
}

func (r *reconnector) set(fn func(*reconnector)) {
	r.mu.Lock()
	fn(r)
	r.mu.Unlock()
}

func (r *reconnector) isClosed(b *Backend) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.closed[b]
}

// reconnectSession connects "test" through a reconnector whose backends the fake prober answers for.
func reconnectSession(t *testing.T, plugin bool) (*Session, *reconnector, *fakeProber) {
	t.Helper()
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("default", "web-1"))
	src.Set(testkit.PodsGVR, testkit.Pod("default", "web-2"))
	p := &fakeProber{}
	rc := &reconnector{fakeConnector: &fakeConnector{src: src, profiles: testProfiles(), conn: &cluster.Conn{}, health: p},
		plugin: plugin, closed: map[*Backend]bool{}}
	s := New(Options{Connector: rc, SnapshotInterval: 10 * time.Millisecond, ConnWatch: fastWatch})
	t.Cleanup(s.Close)
	connectTest(t, s)
	return s, rc, p
}

// newestAttempt is the context's newest attempt in the connection log.
func newestAttempt(t *testing.T, s *Session, contextID string) ConnectAttempt {
	t.Helper()
	list, err := s.ConnectAttempts(contextID, 1)
	if err != nil || len(list) == 0 {
		t.Fatalf("no attempt for %s: %v", contextID, err)
	}
	return list[0]
}

// TestReconnectKeepsTheUIStateAndSwapsTheBackend is F29: a reconnect builds a new backend from the
// connector and puts it in the old one's place, the old one closed; the view, filter, selection,
// drawer and tabs stay, and the view reads the new backend's cache.
func TestReconnectKeepsTheUIStateAndSwapsTheBackend(t *testing.T) {
	s, rc, _ := reconnectSession(t, false)
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && len(tb.Rows) == 2 })
	ref := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "default", Name: "web-1"}
	s.OpenTab("pods", true)
	s.UpdateUI(func(u *UIState) { u.Filter = "web"; u.Selected = []model.ObjectRef{ref}; u.Drawer = &ref })
	before := s.UIState()
	old := s.Backend()

	if err := s.Reconnect(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	b := s.Backend()
	if b == nil || b == old {
		t.Fatalf("the backend after the reconnect: %p, before %p", b, old)
	}
	if !rc.isClosed(old) || rc.isClosed(b) {
		t.Errorf("closed: old %v, new %v", rc.isClosed(old), rc.isClosed(b))
	}
	u := s.UIState()
	if u.Context != "test" || u.View != before.View || u.Filter != "web" || u.Drawer == nil || *u.Drawer != ref ||
		!slices.Equal(u.Selected, before.Selected) || u.Welcome || u.ActiveTab != before.ActiveTab ||
		!slices.EqualFunc(u.Tabs, before.Tabs, func(a, b ViewTab) bool { return a.ID == b.ID && a.View == b.View }) {
		t.Errorf("the UI after the reconnect: view %q filter %q drawer %v selected %v tabs %d/%d active %q/%q",
			u.View, u.Filter, u.Drawer, u.Selected, len(u.Tabs), len(before.Tabs), u.ActiveTab, before.ActiveTab)
	}
	if c := u.Connection; c.State != Connected || c.Reconnecting || u.connGen == before.connGen {
		t.Errorf("the connection: %+v, gen %d (was %d)", c, u.connGen, before.connGen)
	}
	if e, ok := u.ConnectionOf("test"); !ok || e.gen != u.connGen || !e.Shown {
		t.Errorf("its entry: %+v %v", e, ok)
	}
	// The view reads the new backend: an object the new cache lists shows.
	rc.src.Set(testkit.PodsGVR, testkit.Pod("default", "web-3"))
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && len(tb.Rows) == 3 })
	if a := newestAttempt(t, s, "test"); a.Kind != AttemptReconnect || a.Outcome != OutcomeOK {
		t.Errorf("the connection log's attempt: %s %s", a.Kind, a.Outcome)
	}
	if !strings.Contains(lastToast(s), "Reconnected to test") {
		t.Errorf("toast %q", lastToast(s))
	}
	// Its watch is the new backend's: a lost connection still says so.
	if b.watch == nil || b.watch == old.watch {
		t.Error("the new backend has no watch of its own")
	}
}

// TestReconnectRestartsLogsAndEndsShells: a logs tab's stream opens again on the new connection
// (its Session moves), a shell's session ends as at an interruption, its tab saying a reconnect
// ended it; a tab of another cluster is left alone.
func TestReconnectRestartsLogsAndEndsShells(t *testing.T) {
	s, _, _ := reconnectSession(t, false)
	ref := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "default", Name: "web-1"}
	logs := DockTab{ID: s.NewDockID(DockLogs), Type: DockLogs, Title: "Logs: web-1", Context: "test", Ref: &ref, Container: "app"}
	shell := DockTab{ID: s.NewDockID(DockShell), Type: DockShell, Title: "Shell: web-1", Context: "test", Ref: &ref, Container: "app"}
	other := DockTab{ID: s.NewDockID(DockLogs), Type: DockLogs, Title: "Logs: elsewhere", Context: "other", Ref: &ref}
	s.UpdateUI(func(u *UIState) { u.Dock = append(u.Dock, logs, shell, other) })
	inner := newFakeSession()
	cancelled := make(chan struct{})
	sess := s.connWatch("").trackSession(shell.ID, inner, func() { close(cancelled) })

	if err := s.Reconnect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-inner.Done():
	case <-time.After(10 * time.Second):
		t.Fatal("the shell's session did not end")
	}
	<-cancelled
	u := s.UIState()
	if got, _ := u.DockTab(logs.ID); got.Session != 1 {
		t.Errorf("the logs tab's session: %d", got.Session)
	}
	if got, _ := u.DockTab(shell.ID); got.Interrupted.IsZero() || !got.Reconnected || got.Session != 0 {
		t.Errorf("the shell tab: interrupted %v reconnected %v session %d", got.Interrupted, got.Reconnected, got.Session)
	}
	if got, _ := u.DockTab(other.ID); got.Session != 0 || !got.Interrupted.IsZero() {
		t.Errorf("another cluster's tab: %+v", got)
	}
	if !sess.(*trackedSession).interrupted() {
		t.Error("the session does not say its connection's end ended it")
	}
	// The logs tab opens on the new connection's backend.
	if _, b, err := s.streamTab(logs.ID, DockLogs); err != nil || b != s.Backend() {
		t.Errorf("the logs tab's backend: %v", err)
	}
}

// TestReconnectThatFailsKeepsTheConnection: a reconnect whose connect fails leaves the connection,
// its backend, views and tabs as they were and says why, in its error, a toast and the log.
func TestReconnectThatFailsKeepsTheConnection(t *testing.T) {
	s, rc, _ := reconnectSession(t, false)
	old := s.Backend()
	before := s.UIState()
	rc.set(func(r *reconnector) {
		r.fail = &cluster.ConnectError{Context: "test", Problem: "the API server rejected the credentials (401 Unauthorized)"}
	})
	err := s.Reconnect(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("the reconnect's error: %v", err)
	}
	if s.Backend() != old || rc.isClosed(old) {
		t.Error("the old backend went")
	}
	u := s.UIState()
	if u.Connection.State != Connected || u.Connection.Reconnecting || u.connGen != before.connGen || u.View != before.View {
		t.Errorf("the connection after a failed reconnect: %+v", u.Connection)
	}
	if a := newestAttempt(t, s, "test"); a.Kind != AttemptReconnect || a.Outcome != OutcomeFailed {
		t.Errorf("the attempt: %s %s", a.Kind, a.Outcome)
	}
	if !strings.Contains(lastToast(s), "Could not reconnect to test") {
		t.Errorf("toast %q", lastToast(s))
	}
}

// TestReconnectingShowsWhileItRuns: while the new connection is made, the old one goes on and says
// it reconnects (connection.reconnecting); a second reconnect is refused; a disconnect ends it, its
// attempt cancelled.
func TestReconnectingShowsWhileItRuns(t *testing.T) {
	s, rc, _ := reconnectSession(t, false)
	entered := make(chan struct{}, 1)
	rc.set(func(r *reconnector) { r.hold, r.entered = make(chan struct{}), entered })
	done := make(chan error, 1)
	go func() { done <- s.Reconnect(context.Background(), "test") }()
	<-entered
	waitUntil(t, "reconnecting", func() bool { return s.UIState().Connection.Reconnecting })
	if c, _ := s.UIState().ConnectionOf("test"); !c.Connection.Reconnecting || c.Connection.State != Connected {
		t.Errorf("its entry while it reconnects: %+v", c.Connection)
	}
	if err := s.ReconnectBlocked("test"); !errors.Is(err, errReconnecting) {
		t.Errorf("a second reconnect: %v", err)
	}
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, errReconnectLeft) {
			t.Errorf("the reconnect a disconnect ended: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the reconnect did not end with the disconnect")
	}
	if a := newestAttempt(t, s, "test"); a.Kind != AttemptDisconnect {
		list, _ := s.ConnectAttempts("test", 3)
		t.Errorf("the log's newest attempts: %+v", list)
	}
	list, _ := s.ConnectAttempts("test", 5)
	// Cancelled, or abandoned when the disconnect's own attempt began first.
	if i := slices.IndexFunc(list, func(a ConnectAttempt) bool { return a.Kind == AttemptReconnect }); i < 0 ||
		(list[i].Outcome != OutcomeCancelled && list[i].Outcome != OutcomeAbandoned) {
		t.Errorf("the reconnect's attempt: %+v", list)
	}
}

// TestReconnectWaitsForAJob: a job running on the cluster refuses the reconnect, with its title; once
// it ended the reconnect runs.
func TestReconnectWaitsForAJob(t *testing.T) {
	s, _, _ := reconnectSession(t, false)
	release := make(chan struct{})
	tab, err := s.StartJob(JobSpec{Command: "node.drain", Title: "Drain n1", Context: "test",
		Run: func(ctx context.Context, _ *Job) (string, error) {
			select {
			case <-release:
			case <-ctx.Done():
			}
			return "drained", nil
		}})
	if err != nil {
		t.Fatal(err)
	}
	err = s.ReconnectBlocked("test")
	if err == nil || !strings.Contains(err.Error(), `"Drain n1"`) {
		t.Fatalf("a reconnect while the drain runs: %v", err)
	}
	if err := s.Reconnect(context.Background(), "test"); err == nil {
		t.Fatal("the reconnect ran while the drain runs")
	}
	close(release)
	waitUntil(t, "the job's end", func() bool {
		got, _ := s.UIState().DockTab(tab.ID)
		return got.Job != nil && got.Job.State.Ended()
	})
	if err := s.Reconnect(context.Background(), "test"); err != nil {
		t.Fatalf("the reconnect after the drain: %v", err)
	}
}

// TestReconnectRestartsTheIntegrations: the integrations' followers run for the new backend, and
// the old one's reads end with it.
func TestReconnectRestartsTheIntegrations(t *testing.T) {
	var mu sync.Mutex
	seen := map[*Backend]bool{}
	n := len(integrationFollowers)
	integrationFollowers = append(integrationFollowers, func(_ *state, b *Backend, _ UIState) {
		mu.Lock()
		seen[b] = true
		mu.Unlock()
	})
	t.Cleanup(func() { integrationFollowers = integrationFollowers[:n] })
	s, rc, _ := reconnectSession(t, false)
	old := s.Backend()
	if err := s.Reconnect(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	b := s.Backend()
	waitUntil(t, "the new backend's integrations", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return seen[b]
	})
	if !rc.isClosed(old) {
		t.Error("the old backend, and its integrations' reads, did not close")
	}
	if q := s.UIState().IntegrationsQuestion; q != "" {
		t.Errorf("the integrations question after a reconnect: %q", q)
	}
}

// TestReconnectOfAClusterNotShown: a connected cluster the window does not show reconnects in
// place; the shown one and the window stay.
func TestReconnectOfAClusterNotShown(t *testing.T) {
	s, rc, _ := reconnectSession(t, false)
	if err := s.Connect(context.Background(), "alpha"); err != nil {
		t.Fatal(err)
	}
	shown := s.Backend()
	old := s.BackendOf("test")
	if err := s.Reconnect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	if s.Backend() != shown || s.UIState().Context != "alpha" {
		t.Error("the shown cluster changed")
	}
	nb := s.BackendOf("test")
	if nb == nil || nb == old || !rc.isClosed(old) {
		t.Fatalf("test's backend: %p, was %p", nb, old)
	}
	if e, ok := s.UIState().ConnectionOf("test"); !ok || e.Shown || e.Connection.State != Connected || e.gen == 0 {
		t.Errorf("test's entry: %+v", e)
	}
}

// TestLogInAgainChoosesHowToRenew: a context with a credential plugin logs in again on its
// connection (one connect only); one without reconnects (D-81).
func TestLogInAgainChoosesHowToRenew(t *testing.T) {
	for _, plugin := range []bool{true, false} {
		s, rc, p := reconnectSession(t, plugin)
		p.set(func(f *fakeProber) { f.reject = true })
		waitState(t, s, LoginExpired)
		p.set(func(f *fakeProber) { f.reject = false })
		if err := s.LogInAgain(context.Background(), ""); err != nil {
			t.Fatalf("plugin %v: %v", plugin, err)
		}
		waitState(t, s, Connected)
		rc.mu.Lock()
		made := len(rc.made)
		rc.mu.Unlock()
		if want := map[bool]int{true: 1, false: 2}[plugin]; made != want {
			t.Errorf("plugin %v: %d connects, want %d", plugin, made, want)
		}
		if c := s.UIState().Connection; c.CredentialPlugin != plugin {
			t.Errorf("plugin %v: connection.credentialPlugin %v", plugin, c.CredentialPlugin)
		}
	}
}

// TestReconnectRefusedToAgentsForAnExpiredPluginLogin: an agent's reconnect of a cluster whose login
// expired while it runs a credential plugin is Log in again, the user's; without a plugin, or with a
// working login, agents may reconnect.
func TestReconnectRefusedToAgentsForAnExpiredPluginLogin(t *testing.T) {
	s, _, p := reconnectSession(t, true)
	if err := s.ReconnectRefused(OriginExternalAgent, "test"); err != nil {
		t.Errorf("an agent's reconnect of a working login: %v", err)
	}
	p.set(func(f *fakeProber) { f.reject = true })
	waitState(t, s, LoginExpired)
	for _, o := range []Origin{OriginExternalAgent, OriginBuiltInAgent, OriginCLI} {
		if err := s.ReconnectRefused(o, "test"); !errors.Is(err, ErrReconnectLoginGUIOnly) {
			t.Errorf("%s: %v", o, err)
		}
	}
	if err := s.ReconnectRefused(OriginGUI, "test"); err != nil {
		t.Errorf("the window's: %v", err)
	}
	s2, _, p2 := reconnectSession(t, false)
	p2.set(func(f *fakeProber) { f.reject = true })
	waitState(t, s2, LoginExpired)
	if err := s2.ReconnectRefused(OriginExternalAgent, "test"); err != nil {
		t.Errorf("an agent's reconnect without a plugin: %v", err)
	}
}

// certLogin is a fake API server that logs users in by client certificate, a kubeconfig file whose
// certificate it accepts, and a session connected through it.
type certLogin struct {
	srv    *testkit.FakeAPIServer
	ca     *testkit.TestCert
	file   string
	accept *testkit.TestCert
	mu     sync.Mutex
}

func newCertLogin(t *testing.T) *certLogin {
	t.Helper()
	c := &certLogin{ca: testkit.NewCert(t, testkit.CertSpec{CommonName: "cluster-ca", CA: true})}
	c.accept = c.issue(t)
	c.srv = testkit.NewFakeAPIServer(testkit.FakeAPIOptions{ClientCert: c.accepts})
	t.Cleanup(c.srv.Close)
	c.srv.Set(testkit.PodsGVR, testkit.Pod("default", "web-1"))
	c.file = filepath.Join(t.TempDir(), "ske-mgmt.yaml")
	c.write(t, c.accept)
	return c
}

// issue makes a client certificate of the cluster's CA, as a cloud hands one out for two hours.
func (c *certLogin) issue(t *testing.T) *testkit.TestCert {
	return testkit.NewCert(t, testkit.CertSpec{CommonName: "admin", Organization: []string{"system:masters"}, Parent: c.ca,
		NotBefore: time.Now().Add(-time.Minute), NotAfter: time.Now().Add(2 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}})
}

func (c *certLogin) accepts(leaf *x509.Certificate) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return leaf.SerialNumber.Cmp(c.accept.Cert.SerialNumber) == 0
}

// renew makes the server accept only cert, as at the old certificate's end.
func (c *certLogin) renew(cert *testkit.TestCert) {
	c.mu.Lock()
	c.accept = cert
	c.mu.Unlock()
}

// write puts cert into the kubeconfig file, in place, as the user's download replaces it.
func (c *certLogin) write(t *testing.T, cert *testkit.TestCert) {
	t.Helper()
	data := c.srv.KubeconfigWithClientCert("mgmt", cert.PEM(), cert.KeyPEM(t))
	if err := os.WriteFile(c.file, data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestLogInAgainUsesTheRenewedCertificate is F28 against a real client-go stack: the connection's
// client certificate expires (the server rejects it), the user writes the renewed certificate into
// the same kubeconfig file, and Log in again, which connects anew from the file, connects with it;
// the tabs stay. Before D-81, Log in again checked the old connection again and stayed rejected.
func TestLogInAgainUsesTheRenewedCertificate(t *testing.T) {
	c := newCertLogin(t)
	s := New(Options{Connector: &ClusterConnector{KubeconfigPaths: []string{c.file}}, SnapshotInterval: 10 * time.Millisecond, ConnWatch: fastWatch})
	t.Cleanup(s.Close)
	if err := s.Connect(context.Background(), "mgmt"); err != nil {
		t.Fatal(err)
	}
	if c := s.UIState().Connection; c.State != Connected || c.CredentialPlugin {
		t.Fatalf("the connection: %+v", c)
	}
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && len(tb.Rows) == 1 })
	s.OpenTab("pods", true)
	tabs := s.UIState().Tabs
	old := s.Backend()

	renewed := c.issue(t)
	c.renew(renewed) // the old certificate ends: the server rejects it
	waitState(t, s, LoginExpired)
	// The old connection cannot be renewed in place: nothing re-reads its certificate.
	if err := s.LoginContext(""); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the in-place login's end", func() bool { // its own attempt, not the expiry's
		a := newestAttempt(t, s, "mgmt")
		return a.Kind == AttemptLogin && !a.Running && a.Outcome == OutcomeFailed && len(a.Lines) > 0 &&
			strings.HasPrefix(a.Lines[0].Text, "Logging in again (Log in again)") && !s.UIState().Connection.LoggingIn
	})
	if st := s.UIState().Connection.State; st != LoginExpired {
		t.Fatalf("the in-place login made it %s", st)
	}

	c.write(t, renewed) // the user's renewed kubeconfig
	if err := s.LogInAgain(context.Background(), ""); err != nil {
		t.Fatalf("Log in again: %v", err)
	}
	u := waitState(t, s, Connected)
	if s.Backend() == old || u.Reconnecting {
		t.Errorf("the backend after Log in again: same %v, reconnecting %v", s.Backend() == old, u.Reconnecting)
	}
	if got := s.UIState().Tabs; !slices.EqualFunc(got, tabs, func(a, b ViewTab) bool { return a.ID == b.ID }) {
		t.Errorf("the tabs: %d, were %d", len(got), len(tabs))
	}
	c.srv.Set(testkit.PodsGVR, testkit.Pod("default", "web-2"))
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && len(tb.Rows) == 2 })
	if a := newestAttempt(t, s, "mgmt"); a.Kind != AttemptReconnect || a.Outcome != OutcomeOK {
		t.Errorf("the log: %s %s", a.Kind, a.Outcome)
	}
}

// TestARenewedCertificateReconnectsByItself: a connection whose certificate the server rejects
// reconnects by itself once its kubeconfig file holds another one, without a click: when the login
// expires after the file was renewed, and when the file is renewed after it expired (the kubeconfig
// sources' watch). Each version of the file is tried once.
func TestARenewedCertificateReconnectsByItself(t *testing.T) {
	c := newCertLogin(t)
	cfg := filepath.Join(t.TempDir(), "Cogpit")
	if _, _, err := NewKubeconfigSources(cfg).AddFile(c.file); err != nil {
		t.Fatal(err)
	}
	s := New(Options{Connector: &ClusterConnector{Sources: NewKubeconfigSources(cfg)}, SnapshotInterval: 10 * time.Millisecond, ConnWatch: fastWatch})
	t.Cleanup(s.Close)
	id := SourceContextID("mgmt", "ske-mgmt.yaml")
	if err := s.Connect(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	looked := make(chan struct{}, 16) // the sources' watch saw a change and looked at the logins
	s.Subscribe(func(ev Event) {
		if ev.Type == EventContexts {
			select {
			case looked <- struct{}{}:
			default:
			}
		}
	})
	stop := s.WatchKubeconfigs(20 * time.Millisecond)
	t.Cleanup(stop)

	// The file renewed first, while the old certificate still works: nothing happens yet.
	first := s.Backend()
	next := c.issue(t)
	c.write(t, next)
	select {
	case <-looked:
	case <-time.After(10 * time.Second):
		t.Fatal("the sources' watch did not see the file change")
	}
	if s.Backend() != first {
		t.Fatal("a working connection reconnected because its file changed")
	}
	c.renew(next) // the old certificate ends: the expiry finds the renewed file
	waitUntil(t, "the reconnect at the expiry", func() bool { b := s.Backend(); return b != nil && b != first })
	waitState(t, s, Connected)

	// Expired first, then the file renewed: the sources' watch finds it.
	second := s.Backend()
	last := c.issue(t)
	c.renew(last)
	waitState(t, s, LoginExpired)
	c.write(t, last)
	waitUntil(t, "the reconnect at the file's change", func() bool { b := s.Backend(); return b != nil && b != second })
	waitState(t, s, Connected)
	if !strings.Contains(lastToast(s), "with the renewed login from its kubeconfig file") {
		t.Errorf("toast %q", lastToast(s))
	}

	// A renewed file whose certificate the server rejects too is tried once.
	third := s.Backend()
	c.renew(c.issue(t))
	waitState(t, s, LoginExpired)
	c.write(t, c.issue(t)) // another certificate the server does not accept
	waitUntil(t, "the one try", func() bool {
		a := newestAttempt(t, s, id)
		return a.Kind == AttemptReconnect && a.Outcome == OutcomeFailed
	})
	s.renewChangedLogins()
	if s.Backend() != third {
		t.Error("the failed version was tried again")
	}
	list, _ := s.ConnectAttempts(id, 10)
	tries := 0
	for _, a := range list {
		if a.Kind == AttemptReconnect && a.Outcome == OutcomeFailed {
			tries++
		}
	}
	if tries != 1 {
		t.Errorf("%d failed tries of the same file", tries)
	}
}
