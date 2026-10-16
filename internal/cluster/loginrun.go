package cluster

import (
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"k8s.io/client-go/rest"
)

// A credential plugin's run that a connection waits for, ended when the user cancels and after a
// bound (D-73; the owner's finding of 2026-10-16 night: a connect that waited for an OIDC login in
// the browser, which the user had not finished, could not be cancelled, had no timeout, and kept
// Disconnect waiting behind it).
//
// client-go's exec authenticator runs the plugin with exec.Command and cmd.Run, without a context,
// under the Authenticator's lock (k8s.io/client-go v0.37.1, plugin/pkg/client/auth/exec/exec.go:
// 458–472), and every request that needs the credential waits for that lock whatever its own
// context says (getCreds, exec.go:402–415): a plugin that waits for a browser callback nobody sends
// keeps the connect, and every request of the connection, waiting for as long as it runs. client-go
// offers no hook to the process: no context, no *exec.Cmd, no SysProcAttr.
//
// So Cogpit follows the run where client-go starts it and ends the process itself:
//
//   - the wrapper of the Authenticator's interactiveFunc (loginhold.go), which refreshCredsLocked
//     asks before it builds the command (exec.go:436), notes a run's begin (pluginHold.begin); a
//     goroutine that takes the Authenticator's lock once the run let go of it notes its end;
//   - a run that has not ended after its bound (Options.LoginTimeout, DefaultLoginTimeout), or that
//     the user cancels (a cancelled Connect, Conn.CancelLogin, Conn.Close; Disconnect in the
//     engine), is ended: Cogpit finds the plugin's process among its own child processes by its
//     command line (pluginproc_*.go), sends it SIGTERM, and SIGKILL after killGrace if it still runs
//     (TerminateProcess on Windows). Only the plugin's own process: it shares Cogpit's process group,
//     so there is no group of its own to signal, and a browser it opened is no child of it (macOS's
//     open hands the URL to the system) and is never ended;
//   - once Cogpit ended a run, the Authenticator starts the plugin no more until the user connects
//     or logs in again (rearm): client-go would start it again for each request that waited behind
//     the lock, a browser tab each. A request that needed it ends as a *LoginEndedError.
//
// A run serves every connection whose plugin configuration is the same (client-go keeps one
// Authenticator for them, loginhold.go): a cancel ends it only when no other connection that does
// not hold its login uses it; the bound ends it whoever waits.

// DefaultLoginTimeout bounds one run of a credential plugin (Options.LoginTimeout; the lead's
// decision of the night run 2026-10-16/17, D-73): long enough for a browser login with a second
// factor, short of what a login nobody finishes would cost.
const DefaultLoginTimeout = 5 * time.Minute

// killGrace is how long a plugin has to end after SIGTERM before SIGKILL.
const killGrace = 3 * time.Second

// findTries and findPause bound the search for a run's process: the run is noted just before
// client-go starts it.
const (
	findTries = 20
	findPause = 50 * time.Millisecond
)

// Why Cogpit ended a credential plugin's run (LoginEndedError.Cause).
var (
	// ErrLoginCancelled: the user cancelled the connect or the login (Disconnect, Cancel Connecting).
	ErrLoginCancelled = errors.New("login cancelled")
	// ErrLoginTimedOut: the run did not return within its bound.
	ErrLoginTimedOut = errors.New("the login did not finish in time")
)

// LoginEndedError ends a request that needed a credential plugin's run that Cogpit ended (Cause:
// ErrLoginCancelled or ErrLoginTimedOut), or that Cogpit did not start again after it ended one,
// until the user connects or logs in again. Its text names the program, never its arguments.
type LoginEndedError struct {
	Program string
	Bound   time.Duration // the run's bound
	Cause   error
}

func (e *LoginEndedError) Error() string {
	if errors.Is(e.Cause, ErrLoginTimedOut) {
		return fmt.Sprintf("the login through %s did not finish within %s", e.Program, boundText(e.Bound))
	}
	return "login cancelled: Cogpit ended " + e.Program
}

func (e *LoginEndedError) Unwrap() error { return e.Cause }

// TimedOut reports whether the run ended because it did not return within its bound.
func (e *LoginEndedError) TimedOut() bool { return errors.Is(e.Cause, ErrLoginTimedOut) }

// boundText writes a bound in words: "5 minutes", "1 minute", else Go's form ("200ms").
func boundText(d time.Duration) string {
	switch {
	case d == time.Minute:
		return "1 minute"
	case d > 0 && d%time.Minute == 0:
		return fmt.Sprintf("%d minutes", d/time.Minute)
	}
	return d.String()
}

// LoginRun is a run of a connection's credential plugin that is under way.
type LoginRun struct {
	Program string    // the plugin's program, its file name ("pinniped")
	Since   time.Time // when it began
	Until   time.Time // when Cogpit ends it unless it returned
}

// LoginWatch tells the owner of a connection when a run of its credential plugin begins or ends
// (Options.Login), from Connect on and for the connection's life: notify runs on a goroutine of its
// own, never under a lock of the cluster layer's or client-go's, and Current says what is under way
// when it is asked. Nil-safe.
type LoginWatch struct {
	notify func()
	hold   atomic.Pointer[pluginHold]
}

// NewLoginWatch returns a watch that calls notify at each begin and end of a run.
func NewLoginWatch(notify func()) *LoginWatch { return &LoginWatch{notify: notify} }

// Current returns the run under way of the connection's credential plugin, if one is.
func (w *LoginWatch) Current() (LoginRun, bool) {
	if w == nil {
		return LoginRun{}, false
	}
	if p := w.hold.Load(); p != nil {
		return p.current()
	}
	return LoginRun{}, false
}

// Last returns the last run of the connection's credential plugin that ended, when it ended, and
// the error if Cogpit ended it; ok is false before the first one ended.
func (w *LoginWatch) Last() (run LoginRun, ended time.Time, endedBy *LoginEndedError, ok bool) {
	if w == nil {
		return LoginRun{}, time.Time{}, nil, false
	}
	p := w.hold.Load()
	if p == nil {
		return LoginRun{}, time.Time{}, nil, false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.last == nil {
		return LoginRun{}, time.Time{}, nil, false
	}
	return p.last.LoginRun, p.last.ended, p.last.ending, true
}

// pluginRun is one run of an Authenticator's plugin.
type pluginRun struct {
	LoginRun
	command []string      // the command line it runs, to find its process
	done    chan struct{} // closed when it ended
	timer   *time.Timer   // its bound
	ended   time.Time     // when it ended
	ending  *LoginEndedError
	// watches were told its begin: each is told its end too, also when its connection stopped
	// using the plugin meanwhile (a cancelled connect unregisters before the run ends).
	watches []*LoginWatch
}

// loginEnds is a connection's part: the runs Cogpit ended, or refused to start after it ended one,
// while the connection used the plugin (a request that failed meanwhile needed it), the last one's
// error, and the connection's watch.
type loginEnds struct {
	count atomic.Uint64
	last  atomic.Pointer[LoginEndedError]
	watch *LoginWatch // set before the connection registers; nil without one
}

// configureRuns gives the hold what its runs need: the plugin's command line, its program and the
// runs' bound (the last connection's to register sets it); and the watch of the connection g.
func (p *pluginHold) configureRuns(cfg *rest.Config, g *loginGuard, bound time.Duration) {
	command := append([]string{filepath.Clean(cfg.ExecProvider.Command)}, cfg.ExecProvider.Args...)
	p.mu.Lock()
	p.command, p.program = command, filepath.Base(command[0])
	if bound > 0 {
		p.bound = bound
	}
	p.stopped = nil // a connect is the user's: the plugin may run again (rearm)
	p.mu.Unlock()
	if g.ends.watch != nil {
		g.ends.watch.hold.Store(p)
	}
}

// current is the run under way.
func (p *pluginHold) current() (LoginRun, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.run == nil {
		return LoginRun{}, false
	}
	return p.run.LoginRun, true
}

// begin notes a run that client-go starts now. It runs inside refreshCredsLocked, under the
// Authenticator's lock (lock), which client-go lets go once the plugin's process ended.
func (p *pluginHold) begin(lock *sync.Mutex) {
	now := time.Now()
	p.mu.Lock()
	var told []*LoginWatch
	if prev := p.run; prev != nil {
		p.finishLocked(prev, now) // the lock serialises the runs: the previous one ended
		told = prev.watches
	}
	bound := p.bound
	if bound <= 0 {
		bound = DefaultLoginTimeout
	}
	run := &pluginRun{LoginRun: LoginRun{Program: p.program, Since: now, Until: now.Add(bound)},
		command: slices.Clone(p.command), done: make(chan struct{})}
	run.timer = time.AfterFunc(bound, func() { p.end(run, ErrLoginTimedOut) })
	p.run = run
	run.watches = p.watchesLocked()
	p.mu.Unlock()
	notifyAll(unionWatches(told, run.watches))
	go func() {
		// refreshCredsLocked holds the Authenticator's lock until cmd.Run returned: taking it is
		// the run's end, the only one client-go lets Cogpit see.
		lock.Lock()
		lock.Unlock() //nolint:staticcheck // SA2001: the empty critical section is the point: a wait for the run's end
		p.finish(run)
	}()
}

// finish notes that run ended.
func (p *pluginHold) finish(run *pluginRun) {
	p.mu.Lock()
	if p.run != run {
		p.mu.Unlock()
		return
	}
	p.finishLocked(run, time.Now())
	watches := unionWatches(run.watches, p.watchesLocked())
	p.mu.Unlock()
	notifyAll(watches)
}

// unionWatches is a and b without a watch twice.
func unionWatches(a, b []*LoginWatch) []*LoginWatch {
	out := slices.Clone(a)
	for _, w := range b {
		if !slices.Contains(out, w) {
			out = append(out, w)
		}
	}
	return out
}

func (p *pluginHold) finishLocked(run *pluginRun, now time.Time) {
	run.timer.Stop()
	run.ended = now
	close(run.done)
	p.run = nil
	p.last = run
}

// watchesLocked are the watches of the connections that use the plugin.
func (p *pluginHold) watchesLocked() []*LoginWatch {
	var out []*LoginWatch
	for g := range p.conns {
		if w := g.ends.watch; w != nil && w.notify != nil {
			out = append(out, w)
		}
	}
	return out
}

func notifyAll(watches []*LoginWatch) {
	for _, w := range watches {
		go w.notify()
	}
}

// end ends run for cause, unless it ended or is being ended: no run starts any more until rearm,
// each connection using the plugin is told, and its process is ended.
func (p *pluginHold) end(run *pluginRun, cause error) bool {
	p.mu.Lock()
	if p.run != run || run.ending != nil {
		p.mu.Unlock()
		return false
	}
	e := &LoginEndedError{Program: run.Program, Bound: run.Until.Sub(run.Since), Cause: cause}
	run.ending = e
	p.stopped = e
	p.tellLocked(e)
	p.mu.Unlock()
	slog.Warn("cluster: ending a credential plugin's login", "program", run.Program, "why", e.Error(),
		"after", time.Since(run.Since).Round(time.Millisecond))
	go endProcess(run)
	return true
}

// tellLocked counts an ended or refused run for every connection that uses the plugin.
func (p *pluginHold) tellLocked(e *LoginEndedError) {
	for g := range p.conns {
		g.ends.last.Store(e)
		g.ends.count.Add(1)
	}
}

// refuseStopped is the error of a run asked for after Cogpit ended one, nil while runs may start.
// It runs under the Authenticator's lock (inside refreshCredsLocked).
func (p *pluginHold) refuseStopped() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped == nil {
		return nil
	}
	p.tellLocked(p.stopped)
	return p.stopped
}

// rearm lets the plugin run again: the user logs in again.
func (p *pluginHold) rearm() {
	p.mu.Lock()
	p.stopped = nil
	p.mu.Unlock()
}

// cancelFor ends the run under way for connection g's cancel, unless another connection that does
// not hold its login uses the plugin (the run serves it too).
func (p *pluginHold) cancelFor(g *loginGuard) bool {
	p.mu.Lock()
	run := p.run
	shared := false
	for o, held := range p.conns {
		if o != g && !held {
			shared = true
		}
	}
	p.mu.Unlock()
	if run == nil || shared {
		return false
	}
	return p.end(run, ErrLoginCancelled)
}

// lastSince is the last run that began at or after t: the one under way or the last that ended.
func (p *pluginHold) lastSince(t time.Time) (pluginRun, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, r := range []*pluginRun{p.run, p.last} {
		if r != nil && !r.Since.Before(t) {
			return *r, true
		}
	}
	return pluginRun{}, false
}

// endProcess ends run's process: SIGTERM, and SIGKILL after killGrace if the run has not ended,
// both only to the process found by the plugin's command line among Cogpit's own children.
func endProcess(run *pluginRun) {
	defer func() {
		if r := recover(); r != nil { // a crash is a bug, not a reason to end Cogpit (D-11)
			slog.Error("cluster: ending a credential plugin failed", "program", run.Program, "panic", fmt.Sprint(r))
		}
	}()
	pid := 0
	for range findTries {
		if pid = findPlugin(run.command); pid != 0 {
			break
		}
		select {
		case <-run.done:
			return
		case <-time.After(findPause):
		}
	}
	if pid == 0 {
		slog.Warn("cluster: the credential plugin's process was not found among Cogpit's; it runs on, and its login is not "+
			"used until the user connects or logs in again", "program", run.Program)
		return
	}
	if err := signalPlugin(pid, false); err != nil {
		slog.Warn("cluster: cannot end the credential plugin", "program", run.Program, "pid", pid, "err", err)
	}
	select {
	case <-run.done:
		return
	case <-time.After(killGrace):
	}
	if findPlugin(run.command) != pid { // it ended, or the number is another process's now
		return
	}
	slog.Warn("cluster: the credential plugin did not end on SIGTERM; killing it", "program", run.Program, "pid", pid)
	if err := signalPlugin(pid, true); err != nil {
		slog.Warn("cluster: cannot kill the credential plugin", "program", run.Program, "pid", pid, "err", err)
	}
}

// childProcess is one of Cogpit's child processes: its number and command line (argv, nil where the
// system does not tell it) or its program's file name (exe, Windows).
type childProcess struct {
	pid  int
	argv []string
	exe  string
}

// findPlugin returns the number of the one child process of Cogpit's whose command line is
// command's; 0 when there is none, or more than one (two runs of the same command line from
// different plugin configurations are not told apart, and neither is ended).
func findPlugin(command []string) int {
	found := 0
	for _, c := range childProcesses() {
		if !c.runs(command) {
			continue
		}
		if found != 0 {
			return 0
		}
		found = c.pid
	}
	return found
}

// runs reports whether the process runs command: its arguments last, after the program (argv[0], or
// a script's path after its interpreter, as the kernel starts a #! script); by the program's file
// name where the command line is not known.
func (c childProcess) runs(command []string) bool {
	if len(command) == 0 {
		return false
	}
	prog := filepath.Base(command[0])
	if c.argv == nil {
		exe := strings.TrimSuffix(strings.ToLower(c.exe), ".exe")
		return exe != "" && exe == strings.TrimSuffix(strings.ToLower(prog), ".exe")
	}
	args := command[1:]
	if len(c.argv) < len(command) || !slices.Equal(c.argv[len(c.argv)-len(args):], args) {
		return false
	}
	ran := c.argv[len(c.argv)-len(args)-1]
	if filepath.IsAbs(command[0]) && strings.ContainsRune(ran, filepath.Separator) {
		// A program the kubeconfig names by its path runs under that path, a script's too (the
		// kernel passes it to the interpreter as given): a program of the same name in another
		// folder, a plugin of another context, is not it.
		return filepath.Clean(ran) == filepath.Clean(command[0])
	}
	return ran == command[0] || filepath.Base(ran) == prog // a name looked up in PATH; a script's resolved path
}

// LoginRun returns the run of the connection's credential plugin that is under way, if one is: a
// run serves every connection with the same plugin configuration, and each request of theirs that
// needs the credential waits for it.
func (c *Conn) LoginRun() (LoginRun, bool) {
	if c == nil || c.hold == nil {
		return LoginRun{}, false
	}
	return c.hold.p.current()
}

// CancelLogin ends the run of the connection's credential plugin that is under way, as the user's
// cancel: its process is ended and the plugin does not run again until the user connects or logs in
// again (ArmLogin). A run that another open connection with the same plugin configuration needs
// too, and that one does not hold, runs on. It reports whether it ended one.
func (c *Conn) CancelLogin() bool {
	if c == nil || c.hold == nil {
		return false
	}
	return c.hold.p.cancelFor(c.hold.g)
}

// loginEndMark is the count of runs Cogpit ended or refused for this connection: a request that
// fails while it grows needed one.
func (c *Conn) loginEndMark() uint64 {
	if c == nil || c.hold == nil {
		return 0
	}
	return c.hold.g.ends.count.Load()
}

// loginEndedSince is the error of a run Cogpit ended or refused for this connection since mark; nil
// when none was.
func (c *Conn) loginEndedSince(mark uint64) *LoginEndedError {
	if c == nil || c.hold == nil || c.hold.g.ends.count.Load() == mark {
		return nil
	}
	return c.hold.g.ends.last.Load()
}

// endedWithStderr is the error of a request that failed because Cogpit ended (or did not start
// again) the plugin's run, e: after a timeout with what the plugin wrote to its standard error since
// mark, scrubbed (D-66: a failed login's; it may say what it waited for), after a cancel without it.
func endedWithStderr(err error, e *LoginEndedError, sink *pluginStderr, mark int64, known []string) error {
	if !e.TimedOut() {
		sink.discard()
		return e
	}
	var pe *PluginError
	if errors.As(withPluginStderr(err, sink, mark, known), &pe) && pe.Stderr != "" {
		return &PluginError{Err: e, Stderr: pe.Stderr}
	}
	return e
}

// traceLoginRun reports the plugin's run that began at or after start for the connection log:
// how long it took, or that Cogpit ended it and why.
func (c *Conn) traceLoginRun(start time.Time) {
	if c == nil || c.hold == nil || c.trace == nil {
		return
	}
	run, ok := c.hold.p.lastSince(start)
	if !ok {
		return
	}
	end := run.ended
	if end.IsZero() {
		end = time.Now()
	}
	took := end.Sub(run.Since)
	switch {
	case run.ending != nil && run.ending.TimedOut():
		c.trace.timed(StepLogin, "Login: "+run.ending.Error()+"; Cogpit ended it", took, true)
	case run.ending != nil:
		c.trace.timed(StepLogin, "Login cancelled: Cogpit ended "+run.Program, took, true)
	case run.ended.IsZero():
		c.trace.timed(StepLogin, "Login: "+run.Program+" still runs", took, true)
	default:
		c.trace.timed(StepLogin, "Login: "+run.Program+" answered", took, false)
	}
}
