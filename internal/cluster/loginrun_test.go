//go:build darwin || linux

package cluster

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// waitingPlugin is a credential plugin that, while its block file exists, waits for ever as a
// browser login nobody finishes does (a loop of short sleeps, so that its own children never
// outlive it by more than 100 ms); otherwise it hands out the ExecCredential the test wrote, after
// the delay its delay file names. It notes the number of each of its processes.
type waitingPlugin struct {
	*countingPlugin
	pids  string
	block string
	delay string
}

func newWaitingPlugin(t *testing.T, srv *testkit.FakeAPIServer) *waitingPlugin {
	t.Helper()
	p := &waitingPlugin{countingPlugin: newCountingPlugin(t, srv)}
	dir := filepath.Dir(p.path)
	p.pids, p.block, p.delay = filepath.Join(dir, "pids"), filepath.Join(dir, "block"), filepath.Join(dir, "delay")
	script := `#!/bin/sh
echo $$ >> "` + p.pids + `"
echo run >> "$PLUGIN_RUNS"
if [ -f "` + p.block + `" ]; then
  echo "Log in at http://127.0.0.1:1/callback?code=secret-code-0042 to continue" >&2
  while :; do sleep 0.1; done
fi
if [ -f "` + p.delay + `" ]; then sleep "$(cat "` + p.delay + `")"; fi
cat "$PLUGIN_CRED"
`
	if err := os.WriteFile(p.path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return p
}

// blocking makes the plugin wait for ever, or answer again.
func (p *waitingPlugin) blocking(on bool) {
	p.t.Helper()
	if !on {
		if err := os.Remove(p.block); err != nil && !errors.Is(err, os.ErrNotExist) {
			p.t.Fatal(err)
		}
		return
	}
	if err := os.WriteFile(p.block, nil, 0o600); err != nil {
		p.t.Fatal(err)
	}
}

// lastPID is the number of the plugin's latest process; 0 before its first run.
func (p *waitingPlugin) lastPID() int {
	data, err := os.ReadFile(p.pids)
	if err != nil {
		return 0
	}
	lines := strings.Fields(string(data))
	if len(lines) == 0 {
		return 0
	}
	pid, _ := strconv.Atoi(lines[len(lines)-1])
	return pid
}

// gone reports whether process pid has ended (and was reaped).
func gone(pid int) bool { return pid > 0 && errors.Is(syscall.Kill(pid, 0), syscall.ESRCH) }

func waitCtx(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

// The owner's finding (2026-10-16 night): a connect waiting for an interactive login that the user
// abandoned could not be cancelled. Now a cancel of the connect ends the plugin's process at once,
// the connect returns with "login cancelled", and the watch says the login waited until then.
func TestCancelledConnectEndsThePlugin(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newWaitingPlugin(t, srv)
	p.give(secretToken, false)
	p.blocking(true)
	var mu sync.Mutex
	var steps []ConnectStep
	var notes int
	watch := NewLoginWatch(func() { mu.Lock(); notes++; mu.Unlock() })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	type result struct {
		c   *Conn
		err error
	}
	done := make(chan result, 1)
	go func() {
		c, err := Connect(ctx, Options{KubeconfigPaths: []string{p.kconf}, Login: watch,
			Trace: func(s ConnectStep) { mu.Lock(); steps = append(steps, s); mu.Unlock() }})
		done <- result{c, err}
	}()
	wait := waitCtx(t, 10*time.Second)
	if !testkit.WaitFor(wait, func() bool { _, on := watch.Current(); return on && p.lastPID() != 0 }) {
		t.Fatal("the connect never waited for the plugin")
	}
	run, _ := watch.Current()
	if run.Program != "login-plugin" || run.Until.Sub(run.Since) != DefaultLoginTimeout {
		t.Errorf("the run under way: %+v, want login-plugin bounded by %s", run, DefaultLoginTimeout)
	}
	pid := p.lastPID()

	cancel()
	var r result
	select {
	case r = <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the cancelled connect did not return within 5 s")
	}
	var ended *LoginEndedError
	var ce *ConnectError
	if r.err == nil || !errors.As(r.err, &ce) || !errors.As(r.err, &ended) || ended.TimedOut() {
		t.Fatalf("the cancelled connect: %v, want a ConnectError of a cancelled login", r.err)
	}
	if ce.Problem != "login cancelled: Cogpit ended login-plugin" {
		t.Errorf("the problem: %q", ce.Problem)
	}
	if !testkit.WaitFor(wait, func() bool { return gone(pid) }) {
		t.Fatalf("the plugin's process %d still runs after the cancel", pid)
	}
	if !testkit.WaitFor(wait, func() bool { _, on := watch.Current(); return !on }) {
		t.Error("the watch still says the login runs")
	}
	// The watch tells its listener after its state changed: wait for the end's notice.
	if !testkit.WaitFor(wait, func() bool { mu.Lock(); defer mu.Unlock(); return notes >= 2 }) {
		mu.Lock()
		t.Errorf("the watch was told %d times, want the run's begin and end", notes)
		mu.Unlock()
	}
	mu.Lock()
	defer mu.Unlock()
	var logged bool
	for _, s := range steps {
		logged = logged || (s.Name == StepLogin && s.Failed && strings.Contains(s.Text, "Login cancelled: Cogpit ended login-plugin"))
		if strings.Contains(s.Text, "secret-code-0042") {
			t.Errorf("a step holds the plugin's code: %q", s.Text)
		}
	}
	if !logged {
		t.Errorf("no step says the login was cancelled: %+v", steps)
	}
}

// A plugin's run that does not return within its bound is ended, and the connect fails with a
// reason that says so; the plugin's standard error is in the log, scrubbed (D-66).
func TestLoginTimeoutEndsThePlugin(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newWaitingPlugin(t, srv)
	p.give(secretToken, false)
	p.blocking(true)
	var steps []ConnectStep
	start := time.Now()
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{p.kconf}, LoginTimeout: time.Second, // room for a race-enabled binary to start the plugin
		Trace: func(s ConnectStep) { steps = append(steps, s) }})
	if c != nil {
		c.Close()
	}
	var ended *LoginEndedError
	var ce *ConnectError
	if !errors.As(err, &ce) || !errors.As(err, &ended) || !ended.TimedOut() {
		t.Fatalf("the connect: %v, want a timed-out login", err)
	}
	if want := "the login through login-plugin did not finish within 1s; Connect starts it again"; ce.Problem != want {
		t.Errorf("the problem: %q, want %q", ce.Problem, want)
	}
	if took := time.Since(start); took > 5*time.Second {
		t.Errorf("the connect took %s with a bound of 1s", took)
	}
	if pid := p.lastPID(); !testkit.WaitFor(waitCtx(t, 5*time.Second), func() bool { return gone(pid) }) {
		t.Fatalf("the plugin's process %d still runs after its bound", pid)
	}
	var text strings.Builder
	for _, s := range steps {
		text.WriteString(s.Name + ": " + s.Text + "\n")
	}
	got := text.String()
	for _, want := range []string{"login: Login: the login through login-plugin did not finish within 1s; Cogpit ended it",
		"Class: timeout: the login through login-plugin did not finish within 1s", "Log in at http://127.0.0.1:1/callback"} {
		if !strings.Contains(got, want) {
			t.Errorf("the steps lack %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "secret-code-0042") {
		t.Errorf("the steps hold the plugin's code:\n%s", got)
	}
}

// Once Cogpit ended a run, the requests that waited behind client-go's lock do not start the plugin
// again (a browser tab each): they end as LoginEndedError. Log in again (ArmLogin) lets it run.
func TestEndedLoginDoesNotRunAgainUntilLogInAgain(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newWaitingPlugin(t, srv)
	p.give(secretToken, true) // expired at once: every request needs a run
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{p.kconf}, LoginTimeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	base := p.count()
	p.blocking(true)
	errs := make(chan error, 5)
	for range 5 {
		go func() { errs <- c.Heartbeat(context.Background()) }()
	}
	for range 5 {
		select {
		case err := <-errs:
			var ended *LoginEndedError
			if !errors.As(err, &ended) || !ended.TimedOut() {
				t.Errorf("a heartbeat behind the ended run: %v, want the timed-out login", err)
			}
		case <-time.After(10 * time.Second):
			t.Fatal("a heartbeat behind the ended run did not return")
		}
	}
	if n := p.count() - base; n != 1 {
		t.Fatalf("the plugin ran %d times for five waiting requests, want once", n)
	}
	if err := c.Heartbeat(context.Background()); err == nil || p.count()-base != 1 {
		t.Fatalf("a later heartbeat: %v, %d runs; want refused without a run", err, p.count()-base)
	}

	p.blocking(false)
	c.ArmLogin()
	if err := c.Heartbeat(context.Background()); err != nil {
		t.Fatalf("the heartbeat after Log in again: %v", err)
	}
	if n := p.count() - base; n != 2 {
		t.Errorf("Log in again ran the plugin %d times in all, want once more", n)
	}
}

// Closing a connection (Disconnect) whose renewal waits for its plugin ends the plugin's process,
// and the waiting request returns.
func TestCloseEndsARenewalsPlugin(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newWaitingPlugin(t, srv)
	p.give(secretToken, true)
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{p.kconf}})
	if err != nil {
		t.Fatal(err)
	}
	p.blocking(true)
	done := make(chan error, 1)
	go func() { done <- c.Heartbeat(context.Background()) }()
	wait := waitCtx(t, 10*time.Second)
	if !testkit.WaitFor(wait, func() bool { _, on := c.LoginRun(); return on && p.count() >= 2 }) {
		t.Fatal("the renewal never ran the plugin")
	}
	pid := p.lastPID()
	c.Close()
	select {
	case err := <-done:
		if !errors.Is(err, ErrLoginCancelled) {
			t.Errorf("the renewal after Close: %v, want a cancelled login", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the renewal still waits after Close")
	}
	if !testkit.WaitFor(wait, func() bool { return gone(pid) }) {
		t.Fatalf("the plugin's process %d still runs after Close", pid)
	}
}

// A run another open connection with the same plugin configuration needs is not ended by this
// connection's cancel; closing that one too ends it.
func TestSharedRunEndsWithTheLastConnection(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newWaitingPlugin(t, srv)
	p.give(secretToken, true)
	a, b := p.connect(), p.connect()
	base := p.count()
	p.blocking(true)
	go func() { _ = a.Heartbeat(context.Background()) }()
	wait := waitCtx(t, 10*time.Second)
	if !testkit.WaitFor(wait, func() bool { _, on := b.LoginRun(); return on && p.count() > base }) {
		t.Fatal("the shared login never ran")
	}
	pid, ends := p.lastPID(), a.loginEndMark()
	if b.CancelLogin() {
		t.Fatal("one connection's cancel ended a run the other one needs")
	}
	b.Close()
	if a.loginEndMark() != ends || gone(pid) { // end counts before it signals the process
		t.Fatal("closing one connection ended the shared run")
	}
	a.Close()
	if !testkit.WaitFor(wait, func() bool { return gone(pid) }) {
		t.Fatalf("the plugin's process %d still runs after both closed", pid)
	}
}

// The connect's own timeout bounds its request below the plugin: a login that takes longer than it
// (a browser login of 1.5 s against a timeout of 500 ms) connects. Before D-73 the connect failed
// with "no answer within" its timeout after the login had worked.
func TestConnectTimeoutDoesNotCountTheLogin(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newWaitingPlugin(t, srv)
	p.give(secretToken, false)
	if err := os.WriteFile(p.delay, []byte("1.5"), 0o600); err != nil {
		t.Fatal(err)
	}
	var steps []ConnectStep
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{p.kconf}, ConnectTimeout: 500 * time.Millisecond,
		Trace: func(s ConnectStep) { steps = append(steps, s) }})
	if err != nil {
		t.Fatalf("a login slower than the connect's timeout: %v", err)
	}
	defer c.Close()
	var logged bool
	for _, s := range steps {
		logged = logged || (s.Name == StepLogin && !s.Failed && s.Text == "Login: login-plugin answered" && s.Took >= time.Second)
	}
	if !logged {
		t.Errorf("no step says how long the login took: %+v", steps)
	}
}

// The plugin's process is found among Cogpit's children by its command line, a #! script's too.
func TestFindPluginByItsCommandLine(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "find-me")
	if err := os.WriteFile(script, []byte("#!/bin/sh\nwhile :; do sleep 0.1; done\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(script, "oidc", "--issuer=https://issuer.example")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	want := cmd.Process.Pid
	if !testkit.WaitFor(waitCtx(t, 5*time.Second), func() bool { return findPlugin([]string{script, "oidc", "--issuer=https://issuer.example"}) == want }) {
		t.Fatalf("the plugin's process %d was not found among %+v", want, childProcesses())
	}
	if pid := findPlugin([]string{script, "oidc", "--issuer=https://other.example"}); pid != 0 {
		t.Errorf("another command line found process %d", pid)
	}
}
