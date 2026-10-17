package integration_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// TestRealAlertmanager: on a real cluster the owner provides for tests (COGPIT_REAL_KUBECONFIG), the
// Alertmanager Cogpit finds by name answers, and the counts the pill shows are what that
// Alertmanager answers to a GET of its own, counted by D-77's rules (severity none, Watchdog and
// InfoInhibitor left out; critical; info; everything else warning). It only reads (GET through the
// service proxy) and logs numbers, never a name or a text of the cluster.
func TestRealAlertmanager(t *testing.T) {
	kubeconfig, contextName := realCluster(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	s := engine.New(engine.Options{
		Connector:        &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}},
		SnapshotInterval: 50 * time.Millisecond, ClusterSettings: engine.NewClusterSettingsStore(""),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	decideIntegrations(t, s, contextName) // Alertmanager read (D-82)
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	var u engine.UIState
	for u = s.UIState(); u.Alerts == nil || u.Alerts.State == engine.AlertsChecking; u = s.UIState() {
		if ctx.Err() != nil {
			t.Fatalf("no answer: %+v", u.Alerts)
		}
		time.Sleep(100 * time.Millisecond)
	}
	got := *u.Alerts
	if !got.Answered {
		t.Fatalf("not answered: state %s", got.State)
	}
	board, err := s.AlertBoard("")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Cogpit: state %s, total %d (critical %d, warning %d, info %d), %d groups", got.State, got.Total, got.Critical,
		got.Warning, got.Info, len(board.Groups))

	// The same Alertmanager, asked directly.
	ns, rest, _ := strings.Cut(got.Where, "/")
	svc, port, _ := strings.Cut(rest, ":")
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	body, err := conn.Kube().CoreV1().Services(ns).ProxyGet("", svc, port, "/api/v2/alerts",
		map[string]string{"active": "true", "silenced": "false", "inhibited": "false"}).DoRaw(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var raw []struct {
		Labels map[string]string `json:"labels"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, a := range raw {
		name, sev := a.Labels["alertname"], strings.ToLower(a.Labels["severity"])
		switch {
		case name == "Watchdog" || name == "InfoInhibitor" || sev == "none":
			counts["left out"]++
		case sev == "critical" || sev == "info":
			counts[sev]++
		default:
			counts["warning"]++
		}
	}
	t.Logf("Alertmanager: %d active (critical %d, warning %d, info %d, left out %d)", len(raw), counts["critical"],
		counts["warning"], counts["info"], counts["left out"])
	if got.Critical != counts["critical"] || got.Warning != counts["warning"] || got.Info != counts["info"] ||
		got.Total != len(raw)-counts["left out"] {
		t.Errorf("Cogpit counts %+v; Alertmanager answers %v of %d (the alerts may have changed between the two reads)", got, counts, len(raw))
	}
}
