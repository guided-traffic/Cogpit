//go:build darwin || linux

package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The owner's finding of 2026-10-16 night (D-73), end to end on client-go: a context whose
// credential plugin waits for a login nobody finishes, as an OIDC login in the browser does.

const loginToken = "login-wait-token-0042"

// loginStack is a session whose context "oidc" logs in with a test plugin against a fake API server.
func loginStack(t *testing.T, bound time.Duration) (*Session, *testkit.LoginPlugin) {
	t.Helper()
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: loginToken})
	t.Cleanup(srv.Close)
	p := testkit.NewLoginPlugin(t, srv, "oidc")
	p.Give(loginToken, time.Time{})
	s := New(Options{
		Connector:        &ClusterConnector{KubeconfigPaths: []string{p.Kubeconfig}, LoginTimeout: bound},
		SnapshotInterval: 20 * time.Millisecond, ClusterSettings: NewClusterSettingsStore(""),
		ConnWatch: ConnWatchOptions{Heartbeat: 100 * time.Millisecond, ProbeTimeout: 2 * time.Second, Confirm: 50 * time.Millisecond,
			LoginWait: 10 * time.Second, BackoffFirst: 100 * time.Millisecond, BackoffMax: 400 * time.Millisecond},
	})
	t.Cleanup(s.Close)
	return s, p
}

func within(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

// lastAttempt is the context's newest attempt of kind.
func lastAttempt(t *testing.T, s *Session, contextID, kind string) ConnectAttempt {
	t.Helper()
	list, err := s.ConnectAttempts(contextID, 0)
	if err != nil {
		t.Fatal(err)
	}
	var out ConnectAttempt
	for _, a := range list {
		if a.Kind == kind && (out.Start.IsZero() || a.Start.After(out.Start)) {
			out = a
		}
	}
	return out
}

// startConnect connects "oidc" on its own goroutine and waits until the connect waits for the
// plugin; it returns the connect's result channel and the plugin's process number.
func startConnect(t *testing.T, s *Session, p *testkit.LoginPlugin) (<-chan error, int) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- s.Connect(context.Background(), "oidc") }()
	if !testkit.WaitFor(within(t, 10*time.Second), func() bool {
		c := s.UIState().Connection
		return c.State == Connecting && c.WaitingForLogin.Program == "login-plugin" && p.LastPID() != 0
	}) {
		t.Fatalf("the connect never said it waits for the login: %+v", s.UIState().Connection)
	}
	return done, p.LastPID()
}

// (c) While the plugin runs the connecting cluster's state says what it waits for, with since and
// until, and the connection log says it; (a) Disconnect (Cancel Connecting) ends the plugin's
// process at once and leaves the cluster disconnected, the attempt cancelled.
func TestCancelConnectingEndsTheLogin(t *testing.T) {
	s, p := loginStack(t, 0)
	p.Block(true)
	before := time.Now()
	done, pid := startConnect(t, s, p)

	c := s.UIState().Connection
	if w := c.WaitingForLogin; w.Since.Before(before) || w.Until.Sub(w.Since) != cluster.DefaultLoginTimeout {
		t.Errorf("the wait: %+v, want since the connect, until %s later", w, cluster.DefaultLoginTimeout)
	}
	if want := "waiting for the login through login-plugin; it may wait for you to log in, in a browser"; c.Reason != want {
		t.Errorf("the reason: %q, want %q", c.Reason, want)
	}
	if !testkit.WaitFor(within(t, 5*time.Second), func() bool {
		return strings.Contains(attemptText(lastAttempt(t, s, "oidc", AttemptConnect)), "login: Login: waiting for login-plugin since")
	}) {
		t.Errorf("the connect's attempt does not say it waits:\n%s", attemptText(lastAttempt(t, s, "oidc", AttemptConnect)))
	}

	start := time.Now()
	if err := s.Disconnect(); err != nil {
		t.Fatalf("Disconnect of the waiting connect: %v", err)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("Disconnect took %s behind the waiting login", took)
	}
	if c := s.UIState().Connection; c.State != Disconnected || c.WaitingForLogin != (LoginWait{}) || c.Error != "" {
		t.Errorf("after the cancel: %+v, want disconnected without an error", c)
	}
	select {
	case err := <-done:
		if err == nil {
			t.Error("the cancelled connect returned no error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the cancelled connect did not return")
	}
	if !testkit.WaitFor(within(t, 5*time.Second), func() bool { return testkit.ProcessGone(pid) }) {
		t.Fatalf("the plugin's process %d still runs after Cancel Connecting", pid)
	}
	a := lastAttempt(t, s, "oidc", AttemptConnect)
	if a.Outcome != OutcomeCancelled || !strings.Contains(attemptText(a), "Connecting cancelled (Disconnect): login cancelled: Cogpit ended login-plugin") {
		t.Errorf("the attempt: %s\n%s", a.Outcome, attemptText(a))
	}

	// Connect starts it again, and a plugin that answers connects (no regression).
	p.Block(false)
	if err := s.Connect(within(t, 10*time.Second), "oidc"); err != nil {
		t.Fatalf("Connect after the cancel: %v", err)
	}
	if c := s.UIState().Connection; c.State != Connected || c.WaitingForLogin != (LoginWait{}) || c.Reason != "" {
		t.Errorf("connected: %+v", c)
	}
}

// (b) A login that does not finish within its bound fails the connect with a plain reason, its
// process ended and the step in the connection log.
func TestALoginThatDoesNotFinishFailsTheConnect(t *testing.T) {
	s, p := loginStack(t, 2*time.Second) // room for a race-enabled binary to start the plugin
	p.Block(true)
	err := s.Connect(within(t, 10*time.Second), "oidc")
	var ended *cluster.LoginEndedError
	if !errors.As(err, &ended) || !ended.TimedOut() {
		t.Fatalf("the connect: %v, want a timed-out login", err)
	}
	c := s.UIState().Connection
	if c.State != ConnectError || !strings.Contains(c.Error, "the login through login-plugin did not finish within 2s; Connect starts it again") {
		t.Errorf("the state: %+v", c)
	}
	if pid := p.LastPID(); !testkit.WaitFor(within(t, 5*time.Second), func() bool { return testkit.ProcessGone(pid) }) {
		t.Fatalf("the plugin's process %d still runs after its bound", pid)
	}
	a := lastAttempt(t, s, "oidc", AttemptConnect)
	if text := attemptText(a); a.Outcome != OutcomeFailed ||
		!strings.Contains(text, "login: Login: the login through login-plugin did not finish within 2s; Cogpit ended it") {
		t.Errorf("the attempt: %s\n%s", a.Outcome, text)
	}
}

// Item 4: a renewal of a connection that stands, waiting for its plugin, says so, is bounded the
// same way (Login expired, with the reason), and its process ends.
func TestARenewalThatWaitsIsBounded(t *testing.T) {
	s, p := loginStack(t, 1500*time.Millisecond)
	p.Give(loginToken, time.Now().Add(2*time.Second)) // the connect's credential expires soon
	if err := s.Connect(within(t, 10*time.Second), "oidc"); err != nil {
		t.Fatal(err)
	}
	p.Block(true)
	if !testkit.WaitFor(within(t, 15*time.Second), func() bool {
		return s.UIState().Connection.WaitingForLogin.Program == "login-plugin" && p.Runs() == 2
	}) {
		t.Fatalf("the renewal never said it waits: %+v", s.UIState().Connection)
	}
	pid := p.LastPID()
	if !testkit.WaitFor(within(t, 15*time.Second), func() bool {
		c := s.UIState().Connection
		return c.State == LoginExpired && c.WaitingForLogin == (LoginWait{})
	}) {
		t.Fatalf("the renewal past its bound: %+v", s.UIState().Connection)
	}
	if r := s.UIState().Connection.Reason; r != "the login through login-plugin did not finish within 1.5s; Log in again starts it again" {
		t.Errorf("the reason: %q", r)
	}
	if !testkit.WaitFor(within(t, 5*time.Second), func() bool { return testkit.ProcessGone(pid) }) {
		t.Fatalf("the renewal's process %d still runs after its bound", pid)
	}
	if n := p.Runs(); n != 2 {
		t.Errorf("the plugin ran %d times, want twice: no run after Cogpit ended one", n)
	}

	// Log in again starts it again.
	p.Block(false)
	p.Give(loginToken, time.Time{})
	if err := s.Login(); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(within(t, 15*time.Second), func() bool { return s.UIState().Connection.State == Connected }) {
		t.Fatalf("Log in again: %+v", s.UIState().Connection)
	}
}

// Item 4: Disconnect of a connection whose renewal waits for its plugin ends the plugin.
func TestDisconnectEndsAWaitingRenewal(t *testing.T) {
	s, p := loginStack(t, 0)
	p.Give(loginToken, time.Now().Add(2*time.Second))
	if err := s.Connect(within(t, 10*time.Second), "oidc"); err != nil {
		t.Fatal(err)
	}
	p.Block(true)
	if !testkit.WaitFor(within(t, 15*time.Second), func() bool { return p.Runs() == 2 && s.UIState().Connection.WaitingForLogin.Program != "" }) {
		t.Fatalf("the renewal never ran: %+v", s.UIState().Connection)
	}
	pid := p.LastPID()
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(within(t, 5*time.Second), func() bool { return testkit.ProcessGone(pid) }) {
		t.Fatalf("the renewal's process %d still runs after Disconnect", pid)
	}
}

// A background connect of the restore that waits for its plugin shows it in its entry and is
// cancelled by Disconnect of its context: the entry goes, the process ends, no toast.
func TestCancelABackgroundConnect(t *testing.T) {
	s, p := loginStack(t, 0)
	p.Block(true)
	done := make(chan error, 1)
	go func() { done <- s.ConnectBackground(context.Background(), "oidc") }()
	if !testkit.WaitFor(within(t, 10*time.Second), func() bool {
		e := entryOf(s, "oidc")
		return e.Connection.State == Connecting && e.Connection.WaitingForLogin.Program == "login-plugin" && p.LastPID() != 0
	}) {
		t.Fatalf("the background connect never said it waits: %+v", s.UIState().Connections)
	}
	pid := p.LastPID()
	if err := s.DisconnectContext("oidc"); err != nil {
		t.Fatalf("Disconnect of the background connect: %v", err)
	}
	select {
	case err := <-done:
		if !errors.Is(err, errConnectCancelled) {
			t.Errorf("the background connect: %v, want cancelled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the background connect did not return")
	}
	if _, ok := s.UIState().ConnectionOf("oidc"); ok {
		t.Error("the cancelled background connect keeps its entry")
	}
	if !testkit.WaitFor(within(t, 5*time.Second), func() bool { return testkit.ProcessGone(pid) }) {
		t.Fatalf("the plugin's process %d still runs", pid)
	}
	if a := lastAttempt(t, s, "oidc", AttemptConnect); a.Outcome != OutcomeCancelled {
		t.Errorf("the attempt: %s\n%s", a.Outcome, attemptText(a))
	}
}
