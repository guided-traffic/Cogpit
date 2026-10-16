package engine

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The connection log on disk (D-66): attempts survive a restart, the file is named by a hash and
// owner-only, a cut last line is skipped, and the rotation keeps 7 days and 1 MiB.

// logClock is a settable clock for the log.
type logClock struct{ ns atomic.Int64 }

func newLogClock(t time.Time) *logClock { c := &logClock{}; c.ns.Store(t.UnixNano()); return c }
func (c *logClock) now() time.Time      { return time.Unix(0, c.ns.Load()) }
func (c *logClock) add(d time.Duration) { c.ns.Add(int64(d)) }

// fileLogs is a log kept in dir on clock.
func fileLogs(dir string, clock *logClock) *connectLogs {
	l := &connectLogs{}
	l.setup(clock.now, newConnectLogFiles(dir), nil, nil)
	return l
}

func TestConnectLogFilesKeepAttemptsAcrossARestart(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit", ConnectLogDirName)
	clock := newLogClock(time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC))
	ctx := "prod [../team/kube.yaml]\n\x1b[31m"
	l := fileLogs(dir, clock)
	a := l.begin(ctx, AttemptConnect, LogInfo, "Connecting to prod")
	clock.add(120 * time.Millisecond)
	l.stepTo(a, LogInfo, "answer", "First answer: GET /version: Kubernetes v1.36.1", 40*time.Millisecond)
	clock.add(time.Second)
	l.finish(a, OutcomeOK, LogInfo, "Connected to prod")
	clock.add(time.Minute)
	l.begin(ctx, AttemptReconnect, LogWarn, "Connection interrupted: no answer; Cogpit is reconnecting") // left running: the "crash"

	// The file: named by the hash, owner-only, in an owner-only directory.
	name := ConnectLogFileName(ctx)
	if !regexp.MustCompile(`^[0-9a-f]{32}\.jsonl$`).MatchString(name) {
		t.Errorf("file name %q", name)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != name {
		t.Fatalf("the directory holds %v (%v)", entries, err)
	}
	if runtime.GOOS != "windows" {
		if fi, _ := os.Stat(dir); fi.Mode().Perm() != 0o700 {
			t.Errorf("directory mode %v", fi.Mode().Perm())
		}
		if fi, _ := os.Stat(filepath.Join(dir, name)); fi.Mode().Perm() != 0o600 {
			t.Errorf("file mode %v", fi.Mode().Perm())
		}
	}

	// A new run reads both back: the first ended, the second without an outcome and not running.
	again := fileLogs(dir, clock)
	got, err := again.attempts(ctx, 0)
	if err != nil || len(got) != 2 {
		t.Fatalf("attempts %+v, %v", got, err)
	}
	if r := got[0]; r.Kind != AttemptReconnect || r.Outcome != "" || r.Running || r.Context != ctx {
		t.Errorf("the interrupted attempt: %+v", r)
	}
	c := got[1]
	if c.Kind != AttemptConnect || c.Outcome != OutcomeOK || c.TookMS != 1120 || len(c.Lines) != 3 ||
		c.Lines[1].Step != "answer" || c.Lines[1].TookMS != 40 || c.Lines[2].Text != "Connected to prod" {
		t.Errorf("the connect: %+v", c)
	}
	// The running session sees its own attempt running.
	if mine, _ := l.attempts(ctx, 1); len(mine) != 1 || !mine[0].Running {
		t.Errorf("the session's own attempt: %+v", mine)
	}
	// Another context's attempts are not in this file, and another file's are not read.
	if other, _ := again.attempts("prod", 0); len(other) != 0 {
		t.Errorf("another context: %+v", other)
	}
}

// A record that names another context in a context's file is not read, and a last line a crash cut
// short is skipped; the next attempt starts on a line of its own.
func TestConnectLogFileToleratesACutLineAndForeignRecords(t *testing.T) {
	dir := t.TempDir()
	clock := newLogClock(time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC))
	l := fileLogs(dir, clock)
	a := l.begin("a", AttemptConnect, LogInfo, "Connecting to a")
	l.finish(a, OutcomeFailed, LogError, "refused")
	path := filepath.Join(dir, ConnectLogFileName("a"))
	foreign, _ := json.Marshal(logRecord{Time: clock.now(), Context: "b", Attempt: 1, Event: recordBegin, Kind: AttemptConnect})
	line, _ := json.Marshal(logRecord{Time: clock.now(), Attempt: 1, Level: LogInfo, Text: "b's line"})
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write(append(append(foreign, '\n'), append(line, '\n')...))
	_, _ = f.WriteString(`{"t":"2026-10-16T09:00:01Z","a":`) // the crash
	_ = f.Close()

	again := fileLogs(dir, clock)
	got, err := again.attempts("a", 0)
	if err != nil || len(got) != 1 || got[0].Outcome != OutcomeFailed || len(got[0].Lines) != 2 {
		t.Fatalf("attempts %+v, %v", got, err)
	}
	clock.add(time.Second)
	b := again.begin("a", AttemptConnect, LogInfo, "Connecting to a again")
	again.finish(b, OutcomeOK, LogInfo, "Connected to a")
	got, _ = fileLogs(dir, clock).attempts("a", 0)
	if len(got) != 2 || got[0].Outcome != OutcomeOK || len(got[0].Lines) != 2 {
		t.Errorf("after the cut line: %+v", got)
	}
}

// Rotation: an attempt older than 7 days goes when the next begins; past 1 MiB the oldest go, the
// newest stays; a context's file nobody wrote for longer than 7 days is deleted at the first write
// of a run.
func TestConnectLogRotation(t *testing.T) {
	dir := t.TempDir()
	start := time.Date(2026, 10, 1, 9, 0, 0, 0, time.UTC)
	clock := newLogClock(start)
	l := fileLogs(dir, clock)
	old := l.begin("c", AttemptConnect, LogInfo, "the old one")
	l.finish(old, OutcomeOK, LogInfo, "Connected")
	clock.add(8 * 24 * time.Hour)
	recent := l.begin("c", AttemptConnect, LogInfo, "the recent one") // compacts: the run's first look was at the old one's begin
	l.finish(recent, OutcomeOK, LogInfo, "Connected")
	// A new run's first begin looks at the file again.
	l2 := fileLogs(dir, clock)
	n := l2.begin("c", AttemptDisconnect, LogInfo, "Disconnected")
	l2.finish(n, OutcomeOK, LogInfo, "")
	data, err := os.ReadFile(filepath.Join(dir, ConnectLogFileName("c")))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "the old one") || !strings.Contains(string(data), "the recent one") {
		t.Errorf("after 8 days the file holds:\n%s", data)
	}

	// Size: attempts of 100 lines of 500 bytes each, about 60 KiB per attempt.
	big := strings.Repeat("x", 500)
	for i := range 40 {
		clock.add(time.Minute)
		a := l2.begin("c", AttemptReconnect, LogWarn, "attempt "+string(rune('A'+i)))
		for range 100 {
			l2.stepTo(a, LogWarn, "error", big, 0)
		}
		l2.finish(a, OutcomeFailed, LogWarn, "end")
	}
	clock.add(time.Minute)
	l2.begin("c", AttemptConnect, LogInfo, "the newest")
	fi, err := os.Stat(filepath.Join(dir, ConnectLogFileName("c")))
	if err != nil {
		t.Fatal(err)
	}
	if fi.Size() > connectLogMaxBytes {
		t.Errorf("the file is %d bytes", fi.Size())
	}
	got, _ := fileLogs(dir, clock).attempts("c", 0)
	if len(got) == 0 || got[0].Lines[0].Text != "the newest" || got[len(got)-1].Lines[0].Text == "attempt A" {
		t.Errorf("kept %d attempts, newest %q, oldest %q", len(got), got[0].Lines[0].Text, got[len(got)-1].Lines[0].Text)
	}

	// The sweep: another context's file untouched for 8 days goes at a run's first write.
	stale := filepath.Join(dir, ConnectLogFileName("gone"))
	if err := os.WriteFile(stale, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	past := time.Now().Add(-8 * 24 * time.Hour)
	if err := os.Chtimes(stale, past, past); err != nil {
		t.Fatal(err)
	}
	l3 := fileLogs(dir, clock)
	l3.begin("c", AttemptConnect, LogInfo, "after the sweep")
	l3.files.sweepWait.Wait()
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("the stale file is still there: %v", err)
	}
}

// A directory that cannot be written is said once; the session keeps the log in memory.
func TestConnectLogUnwritableIsSaidOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("a file in the directory's place is enough, but the modes differ")
	}
	parent := t.TempDir()
	dir := filepath.Join(parent, "logs")
	if err := os.WriteFile(dir, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	clock := newLogClock(time.Date(2026, 10, 16, 9, 0, 0, 0, time.UTC))
	var warned atomic.Int32
	l := &connectLogs{}
	l.setup(clock.now, newConnectLogFiles(dir), nil, func(error) { warned.Add(1) })
	for range 3 {
		a := l.begin("c", AttemptConnect, LogInfo, "Connecting")
		l.finish(a, OutcomeOK, LogInfo, "Connected")
	}
	if warned.Load() != 1 {
		t.Errorf("warned %d times", warned.Load())
	}
	if got, err := l.attempts("c", 0); err == nil || len(got) != 3 {
		t.Errorf("attempts %d, %v", len(got), err)
	}
}

// A real connect (the fake API server, through the cluster layer) documents its steps: the cluster
// layer's, discovery, the cache, the heartbeat, and the access reviews' first answer once it came;
// the attempt succeeded, and no line is empty.
func TestConnectLogStepsOfARealConnect(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Access: &testkit.FakeAccess{}})
	defer srv.Close()
	conn := &ClusterConnector{KubeconfigPaths: []string{insecureKubeconfig(t, srv)}}
	s := New(Options{Connector: conn, ConnectLogDir: t.TempDir()})
	t.Cleanup(s.Close)
	if err := s.Connect(context.Background(), "checked"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var got []ConnectAttempt
	if !testkit.WaitFor(ctx, func() bool {
		got, _ = s.ConnectAttempts("checked", 1)
		for _, l := range got[0].Lines {
			if l.Step == StepAccess {
				return true
			}
		}
		return false
	}) {
		t.Fatalf("no access reviews line: %+v", got)
	}
	steps := map[string]string{}
	for _, l := range got[0].Lines {
		if strings.TrimSpace(l.Text) == "" {
			t.Errorf("an empty line: %+v", l)
		}
		if l.Step != "" && steps[l.Step] == "" {
			steps[l.Step] = l.Text
		}
	}
	for step, want := range map[string]string{
		"kubeconfig": `Kubeconfig: context "checked"`, "server": "API server: " + srv.URL, "login": "Login: none",
		"tls":   "TLS: the server's certificate is checked against the kubeconfig's certificate-authority-data",
		"proxy": "Proxy: ", "answer": "First answer: GET /version: Kubernetes v",
		StepDiscovery: "Discovery: ", StepCache: "Cache: ", StepHeartbeat: "Heartbeat: GET /version every 15s",
		StepAccess: "Access reviews: the rules of 1 namespace(s) answered",
	} {
		if !strings.HasPrefix(steps[step], want) {
			t.Errorf("step %s: %q, want %q…", step, steps[step], want)
		}
	}
	if got[0].Outcome != OutcomeOK || got[0].Kind != AttemptConnect {
		t.Errorf("the attempt: %+v", got[0])
	}
}

// A session with a directory writes a connect's attempt and announces each change (EventConnectLog),
// coalesced, for the context.
func TestConnectLogSessionWritesAndAnnounces(t *testing.T) {
	dir := t.TempDir()
	fc := &fakeConnector{src: testkit.NewMemorySource(), profiles: testProfiles()}
	s := New(Options{Connector: fc, ConnectLogDir: dir})
	t.Cleanup(s.Close)
	var mu sync.Mutex
	seen := map[string]int{}
	s.Subscribe(func(ev Event) {
		if ev.Type == EventConnectLog {
			mu.Lock()
			seen[ev.View]++
			mu.Unlock()
		}
	})
	connectTest(t, s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool { mu.Lock(); defer mu.Unlock(); return seen["test"] > 0 }) {
		t.Fatal("no EventConnectLog for test")
	}
	if _, err := os.Stat(filepath.Join(dir, ConnectLogFileName("test"))); err != nil {
		t.Errorf("no file: %v", err)
	}
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	got, err := s.ConnectAttempts("test", 0)
	if err != nil || len(got) != 2 || got[0].Kind != AttemptDisconnect || got[1].Kind != AttemptConnect || got[1].Outcome != OutcomeOK {
		t.Errorf("attempts %+v, %v", got, err)
	}
}
