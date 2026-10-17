package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// Several clusters connected at once (D-69): the pool of backends, one shown.

// poolCluster is one cluster of a poolConnector: its own objects, its own connection's answers and
// how often its backend was closed.
type poolCluster struct {
	src    *testkit.MemorySource
	prober *fakeProber
	closed atomic.Int32
}

// poolConnector connects each of its clusters to a cache of its own over the cluster's source, so
// that each source's counters are the requests one cluster costs.
type poolConnector struct {
	warm time.Duration

	mu       sync.Mutex
	clusters map[string]*poolCluster
	connects map[string]int
}

func newPoolConnector(warm time.Duration, names ...string) *poolConnector {
	c := &poolConnector{warm: warm, clusters: map[string]*poolCluster{}, connects: map[string]int{}}
	for _, n := range names {
		src := testkit.NewMemorySource()
		src.Set(testkit.PodsGVR, testkit.Pod("default", n+"-pod"))
		c.clusters[n] = &poolCluster{src: src, prober: &fakeProber{}}
	}
	return c
}

func (c *poolConnector) Contexts() ([]ContextInfo, error) {
	var out []ContextInfo
	for n := range c.clusters {
		out = append(out, ContextInfo{Name: n})
	}
	return out, nil
}

func (c *poolConnector) Connect(_ context.Context, name string) (*Backend, error) {
	c.mu.Lock()
	cl := c.clusters[name]
	c.connects[name]++
	c.mu.Unlock()
	if cl == nil {
		return nil, errors.New("dial tcp 127.0.0.1:1: connect: connection refused")
	}
	ch := cache.New(cl.src, cache.Options{WarmPeriod: c.warm})
	return &Backend{
		Context: name, Namespace: "default", Cache: ch, Profiles: testProfiles(), Conn: &cluster.Conn{}, Health: cl.prober,
		Close: func() { ch.Close(); cl.closed.Add(1) },
	}, nil
}

func (c *poolConnector) connectCount(name string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connects[name]
}

func poolSession(t *testing.T, warm time.Duration, names ...string) (*Session, *poolConnector) {
	t.Helper()
	pc := newPoolConnector(warm, names...)
	s := New(Options{Connector: pc, SnapshotInterval: 10 * time.Millisecond, ConnWatch: fastWatch})
	t.Cleanup(s.Close)
	return s, pc
}

func mustConnect(t *testing.T, s *Session, name string) {
	t.Helper()
	if err := s.Connect(context.Background(), name); err != nil {
		t.Fatalf("connect %s: %v", name, err)
	}
}

// connectionsOf sums up UIState.Connections as "context state shown cold".
func poolConns(u UIState) []string {
	var out []string
	for _, c := range u.Connections {
		out = append(out, fmt.Sprintf("%s %s shown=%v cold=%v", c.Context, c.Connection.State, c.Shown, c.Cold))
	}
	return out
}

// TestConnectKeepsTheShownClusterConnected: a second cluster's connect leaves the first connected,
// its watch beating and a shell of its dock running; showing the first again connects nothing and
// lists nothing; Connections lists both with the shown one marked.
func TestConnectKeepsTheShownClusterConnected(t *testing.T) {
	s, pc := poolSession(t, time.Minute, "alpha", "beta")
	mustConnect(t, s, "alpha")
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && len(tb.Rows) == 1 })
	alpha := s.Backend()

	shell := newFakeSession()
	ref := model.ObjectRef{Context: "alpha", Kind: "pods", Namespace: "default", Name: "alpha-pod"}
	tab := DockTab{ID: s.NewDockID(DockShell), Type: DockShell, Title: "Shell", Context: "alpha", Ref: &ref, Container: "app"}
	s.UpdateUI(func(u *UIState) { u.Dock = append(u.Dock, tab) })
	tracked := s.connWatch("alpha").trackSession(tab.ID, shell, func() {})
	defer tracked.Close()

	mustConnect(t, s, "beta")
	u := s.UIState()
	if u.Context != "beta" || u.Connection.State != Connected {
		t.Fatalf("shown: %q %s", u.Context, u.Connection.State)
	}
	if got, want := poolConns(u), []string{"alpha connected shown=false cold=false", "beta connected shown=true cold=false"}; !slices.Equal(got, want) {
		t.Errorf("connections %q, want %q", got, want)
	}
	if s.BackendOf("alpha") != alpha || pc.clusters["alpha"].closed.Load() != 0 {
		t.Fatal("alpha's backend was closed or replaced")
	}
	if _, ok := s.UIState().DockTab(tab.ID); !ok {
		t.Error("alpha's shell tab went")
	}
	select {
	case <-shell.Done():
		t.Fatal("alpha's shell ended when beta was connected")
	default:
	}
	beats := pc.clusters["alpha"].prober.get().beats
	waitUntil(t, "alpha's heartbeat to go on", func() bool { return pc.clusters["alpha"].prober.get().beats > beats+2 })

	// Back to alpha: shown at once, from its cache.
	lists := pc.clusters["alpha"].src.Lists()
	mustConnect(t, s, "alpha")
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && tb.Context == "alpha" && len(tb.Rows) == 1 })
	if n := pc.connectCount("alpha"); n != 1 {
		t.Errorf("alpha connected %d times", n)
	}
	if got := pc.clusters["alpha"].src.Lists(); got != lists {
		t.Errorf("showing alpha again listed %d times", got-lists)
	}
	if u := s.UIState(); u.Context != "alpha" || u.NavigatorScope != NavScopeCluster("alpha") || poolConns(u)[0] != "alpha connected shown=true cold=false" {
		t.Errorf("after showing alpha: %q %q %q", u.Context, u.NavigatorScope, poolConns(u))
	}
}

// TestSwitchingBackWithinTheWarmPeriodListsNothing: the views of a left cluster release their
// informers to the cache's warm period: coming back within it costs no list; once it ended the
// cluster is cold (Connections) and coming back lists again.
func TestSwitchingBackWithinTheWarmPeriodListsNothing(t *testing.T) {
	warm := 300 * time.Millisecond
	s, pc := poolSession(t, warm, "alpha", "beta")
	src := pc.clusters["alpha"].src
	mustConnect(t, s, "alpha")
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced })
	listed := src.Lists()

	mustConnect(t, s, "beta")
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && tb.Context == "beta" })
	mustConnect(t, s, "alpha") // within the warm period
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && tb.Context == "alpha" })
	if got := src.Lists(); got != listed {
		t.Errorf("back within the warm period: %d lists", got-listed)
	}

	mustConnect(t, s, "beta")
	waitUntil(t, "alpha to turn cold", func() bool {
		c, ok := s.UIState().ConnectionOf("alpha")
		return ok && c.Cold && !c.Shown
	})
	if !s.Backend().Cache.WarmUntil().IsZero() && s.BackendOf("alpha").Cache.WarmUntil().After(time.Now()) {
		t.Error("cold while its cache still holds warm informers")
	}
	mustConnect(t, s, "alpha")
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced && tb.Context == "alpha" })
	if got := src.Lists(); got <= listed {
		t.Error("back after the warm period listed nothing: the informers were kept")
	}
	if c, _ := s.UIState().ConnectionOf("alpha"); c.Cold || !c.Shown {
		t.Errorf("shown again: %+v", c)
	}
}

// TestReadsOfAnotherConnectedCluster: a read names a connected cluster's context and reads its
// cache; a context that is not connected is refused and nothing connects.
func TestReadsOfAnotherConnectedCluster(t *testing.T) {
	s, pc := poolSession(t, time.Minute, "alpha", "beta", "gamma")
	mustConnect(t, s, "alpha")
	mustConnect(t, s, "beta")
	objs, err := s.ObjectsIn(context.Background(), "alpha", "pods", nil)
	if err != nil || len(objs) != 1 || objs[0].GetName() != "alpha-pod" {
		t.Fatalf("alpha's pods from beta's window: %v %v", objs, err)
	}
	if obj, err := s.Object(context.Background(), model.ObjectRef{Context: "alpha", Kind: "pods", Namespace: "default", Name: "alpha-pod"}); err != nil || obj == nil {
		t.Errorf("an object of alpha: %v", err)
	}
	_, err = s.ObjectsIn(context.Background(), "gamma", "pods", nil)
	var nc *NotConnectedError
	if !errors.As(err, &nc) || !errors.Is(err, ErrNotConnected) || nc.Context != "gamma" {
		t.Errorf("gamma, not connected: %v", err)
	}
	if pc.connectCount("gamma") != 0 {
		t.Error("a read connected gamma")
	}
}

// TestDisconnectingOneLeavesTheOther: disconnecting a cluster the window does not show closes its
// backend and its tabs, not the shown one's; disconnecting the shown one leaves the other
// connected.
func TestDisconnectingOneLeavesTheOther(t *testing.T) {
	s, pc := poolSession(t, time.Minute, "alpha", "beta")
	mustConnect(t, s, "alpha")
	s.OpenTab("services", true) // a fixed tab of alpha's (the preview tab moves to the cluster shown next)
	alphaTab := s.UIState().ActiveTab
	mustConnect(t, s, "beta")
	if tab, ok := s.UIState().Tab(alphaTab); !ok || tab.Context != "alpha" {
		t.Fatalf("alpha's fixed tab went with beta's connect: %+v", tab)
	}
	if err := s.DisconnectContext("alpha"); err != nil {
		t.Fatal(err)
	}
	u := s.UIState()
	if u.Context != "beta" || u.Connection.State != Connected || pc.clusters["alpha"].closed.Load() != 1 || s.BackendOf("alpha") != nil {
		t.Fatalf("after alpha's disconnect: %q %s, alpha closed %d", u.Context, u.Connection.State, pc.clusters["alpha"].closed.Load())
	}
	if _, ok := u.Tab(alphaTab); ok {
		t.Error("alpha's tab stayed")
	}
	if got := poolConns(u); !slices.Equal(got, []string{"beta connected shown=true cold=false"}) {
		t.Errorf("connections %q", got)
	}
	if err := s.DisconnectContext("alpha"); !errors.Is(err, ErrNotConnected) {
		t.Errorf("alpha again: %v", err)
	}

	mustConnect(t, s, "alpha")
	if err := s.Disconnect(); err != nil { // the shown one
		t.Fatal(err)
	}
	u = s.UIState()
	if u.Connection.State != Disconnected || s.BackendOf("beta") == nil || pc.clusters["beta"].closed.Load() != 0 {
		t.Errorf("after the shown one's disconnect: %s, beta %v", u.Connection.State, s.BackendOf("beta"))
	}
	if got := poolConns(u); !slices.Equal(got, []string{"beta connected shown=false cold=false"}) {
		t.Errorf("connections %q", got)
	}
}

// TestABackgroundInterruptionBlocksItsWritesOnly: each connected cluster has its own connection
// watch; an interrupted cluster the window does not show blocks the writes to it, not those to
// the shown one, and the banner's state stays the shown one's.
func TestABackgroundInterruptionBlocksItsWritesOnly(t *testing.T) {
	s, pc := poolSession(t, time.Minute, "alpha", "beta")
	mustConnect(t, s, "alpha")
	mustConnect(t, s, "beta")
	pc.clusters["alpha"].prober.set(func(f *fakeProber) { f.down = true })
	waitUntil(t, "alpha interrupted", func() bool {
		c, ok := s.UIState().ConnectionOf("alpha")
		return ok && c.Connection.State == Interrupted && !c.Connection.Since.IsZero()
	})
	if u := s.UIState(); u.Connection.State != Connected {
		t.Errorf("the shown cluster's connection: %s", u.Connection.State)
	}
	if err := s.writeBlocked(DangerMutating, "alpha"); !errors.Is(err, errConnInterrupted) {
		t.Errorf("a write to alpha: %v", err)
	}
	if err := s.writeBlocked(DangerMutating, ""); err != nil {
		t.Errorf("a write to the shown beta: %v", err)
	}
	if err := s.writeBlocked(DangerRead, "alpha"); err != nil {
		t.Errorf("a read of alpha: %v", err)
	}
	if note := s.StaleNote(); note != "" {
		t.Errorf("the shown cluster's reads are noted: %q", note)
	}
	if err := s.TryNow("alpha"); err != nil {
		t.Errorf("TryNow of alpha: %v", err)
	}
	pc.clusters["alpha"].prober.set(func(f *fakeProber) { f.down = false })
	waitUntil(t, "alpha back", func() bool {
		c, ok := s.UIState().ConnectionOf("alpha")
		return ok && c.Connection.State == Connected
	})
	// Shown again, its connection is the window's: the banner reads it.
	mustConnect(t, s, "alpha")
	pc.clusters["alpha"].prober.set(func(f *fakeProber) { f.down = true })
	waitState(t, s, Interrupted)
	if c, _ := s.UIState().ConnectionOf("beta"); c.Connection.State != Connected {
		t.Errorf("beta in the background: %s", c.Connection.State)
	}
}

// TestJobsSurviveASwitch: a job on a cluster goes on while the window shows another one, and ends
// when its own cluster is disconnected.
func TestJobsSurviveASwitch(t *testing.T) {
	s, _ := poolSession(t, time.Minute, "alpha", "beta")
	mustConnect(t, s, "alpha")
	cause := make(chan error, 1)
	tab, err := s.StartJob(JobSpec{Title: "long", Context: "alpha", Run: func(ctx context.Context, _ *Job) (string, error) {
		<-ctx.Done()
		cause <- context.Cause(ctx)
		return "", ctx.Err()
	}})
	if err != nil {
		t.Fatal(err)
	}
	mustConnect(t, s, "beta")
	if err := s.DisconnectContext("beta"); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-cause:
		t.Fatalf("the job ended with another cluster's switch and disconnect: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	if j, ok := jobTabOf(s, tab.ID); !ok || j.State != JobRunning {
		t.Fatalf("the job %+v", j)
	}
	// A job may start on a connected cluster the window does not show.
	mustConnect(t, s, "beta")
	if _, err := s.StartJob(JobSpec{Title: "background", Context: "alpha", Run: func(context.Context, *Job) (string, error) { return "done", nil }}); err != nil {
		t.Errorf("a job on alpha while beta is shown: %v", err)
	}
	if err := s.DisconnectContext("alpha"); err != nil {
		t.Fatal(err)
	}
	if err := <-cause; !errors.Is(err, errJobClusterChanged) {
		t.Errorf("alpha's disconnect: %v", err)
	}
}

// TestConnectionsFollowTheConnects: a connect under way has its entry while it runs, a failed one
// leaves none, and the others stay.
func TestConnectionsFollowTheConnects(t *testing.T) {
	s, _ := poolSession(t, time.Minute, "alpha")
	mustConnect(t, s, "alpha")
	var seen []string
	unsub := s.Subscribe(func(ev Event) {
		if ev.Type == EventUIState && ev.UIState != nil && ev.UIState.Connection.State == Connecting {
			seen = poolConns(*ev.UIState)
		}
	})
	if err := s.Connect(context.Background(), "nowhere"); err == nil {
		t.Fatal("connected nowhere")
	}
	unsub()
	if want := []string{"alpha connected shown=false cold=false", "nowhere connecting shown=true cold=false"}; !slices.Equal(seen, want) {
		t.Errorf("while connecting: %q, want %q", seen, want)
	}
	u := s.UIState()
	if u.Context != "nowhere" || u.Connection.State != ConnectError {
		t.Errorf("after the failure: %q %s", u.Context, u.Connection.State)
	}
	if got := poolConns(u); !slices.Equal(got, []string{"alpha connected shown=false cold=false"}) {
		t.Errorf("after the failure: %q", got)
	}
	if s.BackendOf("alpha") == nil {
		t.Error("alpha went with another cluster's failed connect")
	}
}

// TestDiagnosticsShowEveryCluster: the Diagnostics report sums up every connected cluster, and
// reads a background one's in full when asked.
func TestDiagnosticsShowEveryCluster(t *testing.T) {
	s, _ := poolSession(t, time.Minute, "alpha", "beta")
	mustConnect(t, s, "alpha")
	waitTable(t, s, "pods", func(tb *TableSnapshot) bool { return tb.Synced })
	mustConnect(t, s, "beta")
	r := s.DiagnosticsReport()
	if r.Context != "beta" || len(r.Clusters) != 2 || r.Clusters[0].Context != "alpha" || r.Clusters[0].Shown || !r.Clusters[1].Shown {
		t.Fatalf("report %q, clusters %+v", r.Context, r.Clusters)
	}
	if r.Clusters[0].Objects == 0 {
		t.Errorf("alpha's objects: %+v", r.Clusters[0])
	}
	a, err := s.DiagnosticsReportOf("alpha")
	if err != nil || a.Context != "alpha" || len(a.Watches) == 0 {
		t.Errorf("alpha's report: %q %d watches, %v", a.Context, len(a.Watches), err)
	}
	if _, err := s.DiagnosticsReportOf("gamma"); !errors.Is(err, ErrNotConnected) {
		t.Errorf("gamma: %v", err)
	}
}
