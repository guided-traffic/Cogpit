package integration_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
	"github.com/guided-traffic/cogpit/internal/testkit/scorecard"
)

// TestBudgetFirstRows measures P1 on Cogpit's whole read path: an engine connects a context of a
// 1,000-pod cluster (and, as a point beyond the budget, of a 10,000-pod one; testkit's fake API
// server over HTTPS, the real client-go stack, discovery, the access reviews and the user's
// SelfSubjectReview included, D-68) and its
// Pods view shows its first rows, then all of them. The fake
// server answers on the loopback, so a real cluster's round trips come on top: what this measures
// is Cogpit's own share. The budget is 1 s; the test fails only at five times that, so that a
// loaded CI runner does not turn a measurement into a flake, and logs what it measured
// (go test -run TestBudgetFirstRows -v -count=5).
func TestBudgetFirstRows(t *testing.T) {
	for _, pods := range []int{1000, 10000} {
		t.Run(fmt.Sprint(pods), func(t *testing.T) { firstRows(t, pods) })
	}
}

func firstRows(t *testing.T, pods int) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: "budget-token"})
	t.Cleanup(srv.Close)
	for i := range pods {
		srv.Set(testkit.PodsGVR, testkit.Pod(fmt.Sprintf("team-%d", i%10), fmt.Sprintf("web-%04d", i)))
	}
	s := engine.New(engine.Options{
		Connector:       &engine.ClusterConnector{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}},
		ClusterSettings: engine.NewClusterSettingsStore(t.TempDir()),
	})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	if err := s.Connect(ctx, "fake"); err != nil {
		t.Fatal(err)
	}
	connected := time.Since(start)
	// The connect asks once who the user is (D-68): one SelfSubjectReview, included in the time.
	if n := srv.Requests()["create selfsubjectreviews.authentication.k8s.io"]; n != 1 {
		t.Errorf("the connect made %d SelfSubjectReviews, want 1", n)
	}
	if u := s.UIState().User; u == nil || u.Username != testkit.DefaultFakeIdentity.Username || u.Source != engine.UserFromAPIServer {
		t.Errorf("the connected user: %+v", u)
	}
	var first, all time.Duration
	if !testkit.WaitFor(ctx, func() bool {
		ts := s.Table("pods")
		if ts == nil {
			return false
		}
		if first == 0 && len(ts.Rows) > 0 {
			first = time.Since(start)
		}
		if ts.Synced && len(ts.Rows) == pods {
			all = time.Since(start)
			return true
		}
		return false
	}) {
		t.Fatalf("the Pods view never showed %d rows: %+v", pods, s.Table("pods"))
	}
	reqs := srv.RequestsIn()
	keys := make([]string, 0, len(reqs))
	for k := range reqs {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	t.Logf("P1, %d pods: connected after %v, first rows after %v, all rows after %v; requests %v", pods, connected, first, all, keys)
	bound := scorecard.AtMost
	if pods > 1000 {
		bound = scorecard.Context // a point beyond the budget's 1,000 pods
	}
	for _, m := range []struct {
		what string
		d    time.Duration
	}{{"the first rows", first}, {"all rows", all}} {
		scorecard.Record(t, scorecard.Measurement{Budget: "P1", Metric: fmt.Sprintf("from Connect to %s in the Pods view, %d pods", m.what, pods),
			Value: scorecard.Millis(m.d), Unit: "ms", Limit: 1000, Bound: bound,
			How: "the engine on the real client-go stack against testkit's fake API server over HTTPS on the loopback; no window, no real cluster's round trips"})
	}
	if pods <= 1000 && all > 5*time.Second {
		t.Errorf("the Pods view showed its %d rows after %v, the budget is 1 s", pods, all)
	}
}
