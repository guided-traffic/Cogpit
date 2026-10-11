package helm

import (
	"errors"
	"io"
	"strings"
	"testing"

	"helm.sh/helm/v4/pkg/action"
	chartcommon "helm.sh/helm/v4/pkg/chart/common"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"
)

// sdkFixture is the Helm SDK on an in-memory release store and a kube client that only prints:
// the SDK's own rollback and uninstall logic, without a cluster.
func sdkFixture(t *testing.T, revisions ...common.Status) (*Actions, *storage.Storage) {
	t.Helper()
	store := storage.Init(driver.NewMemory())
	for i, st := range revisions {
		rls := &rspb.Release{
			Name: "web", Namespace: "apps", Version: i + 1,
			Info:     &rspb.Info{Status: st, FirstDeployed: t0, LastDeployed: t0},
			Chart:    &chart.Chart{Metadata: &chart.Metadata{Name: "web", Version: "1.0.0", APIVersion: "v2"}},
			Manifest: "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: web\n",
			Config:   map[string]any{"rev": i + 1},
		}
		if err := store.Create(rls); err != nil {
			t.Fatal(err)
		}
	}
	cfg := &action.Configuration{
		Releases:     store,
		KubeClient:   &kubefake.PrintingKubeClient{Out: io.Discard},
		Capabilities: chartcommon.DefaultCapabilities,
	}
	return &Actions{config: func(string) (*action.Configuration, error) { return cfg, nil }}, store
}

func last(t *testing.T, store *storage.Storage) *rspb.Release {
	t.Helper()
	r, err := store.Last("web")
	if err != nil {
		t.Fatal(err)
	}
	rls, ok := r.(*rspb.Release)
	if !ok {
		t.Fatalf("release of type %T", r)
	}
	return rls
}

// Rollback makes a new revision from the chosen one, deployed, and supersedes the others; a
// revision the history lacks is an error that names it. A release a crash left pending-rollback
// can be rolled back again.
func TestRollbackThroughTheSDK(t *testing.T) {
	a, store := sdkFixture(t, common.StatusSuperseded, common.StatusDeployed)
	if err := a.Rollback("apps", "web", 1); err != nil {
		t.Fatal(err)
	}
	rls := last(t, store)
	if rls.Version != 3 || rls.Info.Status != common.StatusDeployed || rls.Config["rev"] != 1 || rls.Info.RollbackRevision != 1 {
		t.Errorf("after rollback: revision %d status %s config %v", rls.Version, rls.Info.Status, rls.Config)
	}
	if err := a.Rollback("apps", "web", 9); err == nil || !strings.Contains(err.Error(), "helm rollback apps/web to revision 9") {
		t.Errorf("rollback to a missing revision: %v", err)
	}

	crashed, store2 := sdkFixture(t, common.StatusSuperseded, common.StatusDeployed, common.StatusPendingRollback)
	if err := crashed.Rollback("apps", "web", 1); err != nil {
		t.Fatalf("finishing a pending rollback: %v", err)
	}
	if rls := last(t, store2); rls.Version != 4 || rls.Info.Status != common.StatusDeployed {
		t.Errorf("after finishing: revision %d status %s", rls.Version, rls.Info.Status)
	}
}

// Uninstall removes the release and its history; one a crash left uninstalling is uninstalled
// again; a release that does not exist is an error.
func TestUninstallThroughTheSDK(t *testing.T) {
	for _, states := range [][]common.Status{
		{common.StatusSuperseded, common.StatusDeployed},
		{common.StatusDeployed, common.StatusUninstalling},
	} {
		a, store := sdkFixture(t, states...)
		if err := a.Uninstall("apps", "web"); err != nil {
			t.Fatalf("%v: %v", states, err)
		}
		if h, err := store.History("web"); err == nil && len(h) > 0 {
			t.Errorf("%v: %d revisions left", states, len(h))
		}
		if err := a.Uninstall("apps", "web"); err == nil {
			t.Error("uninstalling a release that is gone succeeded")
		}
	}
}

// A configuration that cannot be made, or an SDK that panics, is the action's error.
func TestActionsReportFailures(t *testing.T) {
	broken := &Actions{config: func(string) (*action.Configuration, error) { return nil, errors.New("no client") }}
	if err := broken.Rollback("apps", "web", 1); err == nil || !strings.Contains(err.Error(), "no client") {
		t.Errorf("rollback: %v", err)
	}
	panicky := &Actions{config: func(string) (*action.Configuration, error) { return &action.Configuration{}, nil }}
	if err := panicky.Uninstall("apps", "web"); err == nil || !strings.Contains(err.Error(), "internal error") {
		t.Errorf("uninstall with an empty configuration: %v", err)
	}
	if err := NewActions(nil, nil).Uninstall("apps", "web"); err == nil {
		t.Error("an action without a connection succeeded")
	}
}
