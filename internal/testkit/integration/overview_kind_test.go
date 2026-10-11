//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestKindWorkloadsOverviewAndCSV opens the Workloads Overview on the seeded kind cluster: its
// kinds count the cluster's workloads (kind's own kube-proxy and kindnet daemon sets, the seed's
// deployments and pods) with their health, the namespace filter narrows them, and its events and a
// kind's view export as CSV. It writes nothing to the cluster.
func TestKindWorkloadsOverviewAndCSV(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	dispatch := func(id string, params, out any) {
		t.Helper()
		raw, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if out != nil {
			b, _ := json.Marshal(res)
			if err := json.Unmarshal(b, out); err != nil {
				t.Fatalf("%s: %v", id, err)
			}
		}
	}

	dispatch("ui.openView", map[string]any{"view": engine.WorkloadsOverview}, nil)
	if !testkit.WaitFor(ctx, func() bool { t := s.Table(engine.WorkloadsOverview); return t != nil && t.Synced }) {
		t.Fatalf("the overview's events never synced: %+v", s.Table(engine.WorkloadsOverview))
	}
	var all commands.OverviewResult
	if !testkit.WaitFor(ctx, func() bool {
		all = commands.OverviewResult{}
		dispatch("workloads.overview", nil, &all)
		return all.Synced
	}) {
		t.Fatalf("the overview's kinds never synced: %+v", all)
	}
	counts := map[string]engine.OverviewKind{}
	for _, k := range all.Kinds {
		counts[k.Kind] = k
	}
	pods, ds := counts["pods"], counts["daemonsets"]
	if pods.Count == 0 || ds.Count < 2 || len(all.Kinds) < 6 {
		t.Fatalf("overview of kind: %+v", all.Kinds)
	}
	ok := 0
	for _, st := range pods.Statuses {
		if st.Status == model.StatusOK {
			ok = st.Count
		}
	}
	if ok == 0 {
		t.Fatalf("no running pod on kind: %+v", pods)
	}
	t.Logf("overview: %+v", all.Kinds)

	var system commands.OverviewResult
	dispatch("workloads.overview", map[string]any{"namespaces": []string{"kube-system"}}, &system)
	if sys := system.Kinds[0]; sys.Kind != "pods" || sys.Count == 0 || sys.Count >= pods.Count {
		t.Fatalf("kube-system pods %+v of %d", sys, pods.Count)
	}

	var page commands.CSVResult
	dispatch("table.exportCsv", nil, &page)
	if !strings.HasPrefix(page.CSV, "Type,Source,Namespace,Involved Object,Message,Count,Age,Last Seen\n") {
		t.Fatalf("the overview's events as CSV: %.200q", page.CSV)
	}
	dispatch("ui.openView", map[string]any{"view": "pods"}, nil)
	if !testkit.WaitFor(ctx, func() bool { t := s.Table("pods"); return t != nil && t.Synced && t.Total > 0 }) {
		t.Fatal("pods never filled")
	}
	var csv commands.CSVResult
	dispatch("table.exportCsv", nil, &csv)
	if lines := strings.Count(csv.CSV, "\n"); lines != csv.Rows+1 || !strings.HasPrefix(csv.CSV, "Name,") || csv.Rows == 0 {
		t.Fatalf("pods as CSV: %d lines for %d rows: %.200q", lines, csv.Rows, csv.CSV)
	}
}
