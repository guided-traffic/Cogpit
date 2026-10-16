package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// Measurements on a real cluster the owner provides for tests. They run only when
// COGPIT_REAL_KUBECONFIG names its kubeconfig (its current context, or COGPIT_REAL_CONTEXT's), never
// in CI, and they only read: connect, list and watch, as the window does. They log numbers, never
// a name or an object of the cluster; run them with HOME pointing at a scratch directory and
// KUBECONFIG=/dev/null, so that nothing reads ~/.kube or writes Cogpit's own files.

// realCluster returns the real cluster's kubeconfig and context, or skips the test.
func realCluster(t *testing.T) (kubeconfig, contextName string) {
	t.Helper()
	kubeconfig = os.Getenv("COGPIT_REAL_KUBECONFIG")
	if kubeconfig == "" {
		t.Skip("COGPIT_REAL_KUBECONFIG is not set: a real cluster's measurement")
	}
	contextName = os.Getenv("COGPIT_REAL_CONTEXT")
	if contextName == "" {
		ctxs, err := cluster.ListContexts([]string{kubeconfig})
		if err != nil {
			t.Fatalf("the real cluster's kubeconfig: %v", err)
		}
		for _, c := range ctxs {
			if c.Current {
				contextName = c.Name
			}
		}
	}
	if contextName == "" {
		t.Fatal("the real cluster's kubeconfig names no current context; set COGPIT_REAL_CONTEXT")
	}
	return kubeconfig, contextName
}

// realRounds is COGPIT_REAL_ROUNDS, else def.
func realRounds(def int) int {
	if n, err := strconv.Atoi(os.Getenv("COGPIT_REAL_ROUNDS")); err == nil && n > 0 {
		return n
	}
	return def
}

// durations sums up measured times: min, median, max.
type durations []time.Duration

func (d durations) String() string {
	if len(d) == 0 {
		return "none"
	}
	s := slices.Clone(d)
	slices.Sort(s)
	r := func(x time.Duration) string { return x.Round(100 * time.Microsecond).String() }
	return "min " + r(s[0]) + ", median " + r(s[len(s)/2]) + ", max " + r(s[len(s)-1]) + " (" + strconv.Itoa(len(s)) + " runs)"
}

// TestRealDiscoveryCacheConnectTime measures what the discovery cache on disk (D-65) saves when a
// connection is opened: the engine's ClusterConnector.Connect (cluster.Connect's GET /version,
// discovery, the backend), cold (no cache file: discovery from the server, then the file written),
// warm (the file the cold run wrote) and without a cache directory (discovery in memory only, as
// before D-65); and the session's connect to the Pods view's first synced snapshot, cold and warm.
func TestRealDiscoveryCacheConnectTime(t *testing.T) {
	kubeconfig, contextName := realCluster(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	dir := filepath.Join(t.TempDir(), "discovery")

	connect := func(cacheDir string) (time.Duration, int64, cluster.DiscoveryInfo) {
		t.Helper()
		c := &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}, DiscoveryCacheDir: cacheDir}
		start := time.Now()
		be, err := c.Connect(ctx, contextName)
		took := time.Since(start)
		if err != nil {
			t.Fatalf("connect: %v", err)
		}
		defer be.Close()
		return took, be.Conn.Counter().Snapshot().ByVerb[cluster.VerbDiscovery], be.Conn.DiscoveryInfo()
	}
	rounds := realRounds(7)
	var cold, warm, memory durations
	var coldReq, warmReq int64
	var info cluster.DiscoveryInfo
	for range rounds {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
		d, n, i := connect(dir)
		if i.Source != "server" {
			t.Fatalf("cold connect read discovery from %q", i.Source)
		}
		cold, coldReq = append(cold, d), n
		d, n, info = connect(dir)
		if info.Source != "disk" {
			t.Fatalf("warm connect read discovery from %q", info.Source)
		}
		warm, warmReq = append(warm, d), n
		d, _, _ = connect("")
		memory = append(memory, d)
	}
	var size int64
	if entries, err := os.ReadDir(dir); err == nil && len(entries) == 1 {
		if fi, err := entries[0].Info(); err == nil {
			size = fi.Size()
		}
	}
	t.Logf("discovery: %d groups, %d resources; cache file %d KiB", info.Groups, info.Resources, size/1024)
	t.Logf("ClusterConnector.Connect cold (from the server, file written): %v; %d discovery requests", cold, coldReq)
	t.Logf("ClusterConnector.Connect warm (from the disk cache): %v; %d discovery requests", warm, warmReq)
	t.Logf("ClusterConnector.Connect without a cache directory: %v", memory)

	// The session: connect, the Pods view's first synced snapshot with rows.
	session := func(cacheDir string) time.Duration {
		t.Helper()
		s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}, DiscoveryCacheDir: cacheDir}})
		defer s.Close()
		start := time.Now()
		if err := s.Connect(ctx, contextName); err != nil {
			t.Fatalf("session connect: %v", err)
		}
		if !testkit.WaitFor(ctx, func() bool { tbl := s.Table("pods"); return tbl != nil && tbl.Synced && tbl.Total > 0 }) {
			t.Fatal("the Pods view never synced")
		}
		return time.Since(start)
	}
	var sCold, sWarm durations
	for range realRounds(5) {
		if err := os.RemoveAll(dir); err != nil {
			t.Fatal(err)
		}
		sCold = append(sCold, session(dir))
		sWarm = append(sWarm, session(dir))
	}
	t.Logf("Session.Connect to the Pods view's synced rows, cold: %v", sCold)
	t.Logf("Session.Connect to the Pods view's synced rows, warm: %v", sWarm)
}
