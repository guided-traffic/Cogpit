package engine

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

var (
	fluxKsGVR  = schema.GroupVersionResource{Group: "kustomize.toolkit.fluxcd.io", Version: "v1", Resource: "kustomizations"}
	fluxGitGVR = schema.GroupVersionResource{Group: "source.toolkit.fluxcd.io", Version: "v1", Resource: "gitrepositories"}
	fluxHrGVR  = schema.GroupVersionResource{Group: "helm.toolkit.fluxcd.io", Version: "v2", Resource: "helmreleases"}
)

const (
	fluxKsKey  = "kustomizations.kustomize.toolkit.fluxcd.io"
	fluxGitKey = "gitrepositories.source.toolkit.fluxcd.io"
	fluxHrKey  = "helmreleases.helm.toolkit.fluxcd.io"
)

// fluxObj is a Flux object with a Ready condition of the given status ("" for none) that last
// changed at since, suspended when asked.
func fluxObj(apiVersion, kind, namespace, name, ready, since string, suspended bool) *unstructured.Unstructured {
	obj := testkit.Object(apiVersion, kind, namespace, name, map[string]any{"spec": map[string]any{"suspend": suspended}})
	if ready != "" {
		obj.Object["status"] = map[string]any{"conditions": []any{map[string]any{
			"type": "Ready", "status": ready, "message": name + " says " + ready, "lastTransitionTime": since,
		}}}
	}
	return obj
}

// TestFluxDashboardPage: the dashboard counts each served Flux kind's objects by state in the
// selected namespaces, and its table lists the ones that are not ready, the latest change first;
// it follows the objects, the namespace filter and a Flux definition installed while it is open.
func TestFluxDashboardPage(t *testing.T) {
	if title, ok := PageTitle(FluxDashboard); !ok || title != "Flux Dashboard" || !IsPage(FluxDashboard) || !MixedKinds(FluxDashboard) {
		t.Fatalf("page %q %v", title, ok)
	}
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("default", "web-0"))
	src.Set(crdsGVR, testkit.CRD(fluxKsGVR.Group, fluxKsGVR.Resource, "Kustomization", true))
	src.Set(crdsGVR, testkit.CRD(fluxGitGVR.Group, fluxGitGVR.Resource, "GitRepository", true))
	ks := func(ns, name, ready, since string, suspended bool) *unstructured.Unstructured {
		return fluxObj("kustomize.toolkit.fluxcd.io/v1", "Kustomization", ns, name, ready, since, suspended)
	}
	src.Set(fluxKsGVR, ks("flux-system", "apps", "True", "2026-10-13T10:00:00Z", false))
	src.Set(fluxKsGVR, ks("flux-system", "infra", "True", "2026-10-13T10:00:00Z", false))
	src.Set(fluxKsGVR, ks("flux-system", "broken", "False", "2026-10-13T11:00:00Z", false))
	src.Set(fluxKsGVR, ks("team-a", "paused", "True", "2026-10-13T09:00:00Z", true))
	src.Set(fluxGitGVR, fluxObj("source.toolkit.fluxcd.io/v1", "GitRepository", "flux-system", "flux-system", "Unknown", "2026-10-13T12:00:00Z", false))
	s := newCRDSession(t, src)
	waitFor(t, "the Flux kinds", func() bool { return hasKind(s, fluxKsKey) && hasKind(s, fluxGitKey) })

	// Read once, the page not open: from the cache, waiting for the initial lists.
	m, table, err := s.FluxDashboard(context.Background(), "", nil, true)
	if err != nil {
		t.Fatal(err)
	}
	if got := dashboardText(m); got != "Kustomizations 4: Ready 2, Not Ready 1, Suspended 1; Git Repositories 1: Reconciling 1" {
		t.Errorf("model %s", got)
	}
	if !m.Synced || table == nil || table.Total != 3 || rowNames(table) != "flux-system broken paused" {
		t.Errorf("table %s", summary(table))
	}

	s.OpenTab(FluxDashboard, false) // as ui.openView does: the page's default sort
	snap := waitTable(t, s, FluxDashboard, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 3 })
	if rowNames(snap) != "flux-system broken paused" || snap.Sort != fluxIssuesSort || snap.Title != "Flux Dashboard" {
		t.Errorf("page table %s sorted %+v", summary(snap), snap.Sort)
	}
	r := snap.Rows[1]
	if r.Ref.Kind != fluxKsKey || r.Cells[1].Text != "Kustomization" || r.Cells[3].Text != "Not Ready" || r.Cells[3].Status != model.StatusError ||
		r.Cells[4].Text != "broken says False" || r.Cells[0].Link == nil || r.Cells[0].Link.Name != "broken" || r.Status != model.StatusError {
		t.Errorf("row %+v", r)
	}

	// The namespace filter narrows the counts and the table.
	s.UpdateUI(func(u *UIState) { u.Namespaces = []string{"team-a"} })
	snap = waitTable(t, s, FluxDashboard, func(t *TableSnapshot) bool { return len(t.Namespaces) == 1 })
	if rowNames(snap) != "paused" {
		t.Errorf("team-a: %s", summary(snap))
	}
	m, _, _ = s.FluxDashboard(context.Background(), "", nil, false)
	if got := dashboardText(m); got != "Kustomizations 1: Suspended 1; Git Repositories 0: " {
		t.Errorf("team-a model %s", got)
	}
	s.UpdateUI(func(u *UIState) { u.Namespaces = nil })

	// A failing object recovers: its row goes.
	src.Set(fluxKsGVR, ks("flux-system", "broken", "True", "2026-10-13T12:30:00Z", false))
	waitTable(t, s, FluxDashboard, func(t *TableSnapshot) bool { return len(t.Namespaces) == 0 && rowNames(t) == "flux-system paused" })

	// Flux's Helm releases are installed while the page is open: the page reads them too.
	src.Set(crdsGVR, testkit.CRD(fluxHrGVR.Group, fluxHrGVR.Resource, "HelmRelease", true, testkit.CRDVersion{Name: "v2", Served: true, Storage: true}))
	src.Set(fluxHrGVR, fluxObj("helm.toolkit.fluxcd.io/v2", "HelmRelease", "shop", "podinfo", "False", "2026-10-13T13:00:00Z", false))
	snap = waitTable(t, s, FluxDashboard, func(t *TableSnapshot) bool { return rowNames(t) == "podinfo flux-system paused" })
	if snap.Rows[0].Ref.Kind != fluxHrKey || snap.Rows[0].Cells[1].Text != "HelmRelease" {
		t.Errorf("helm release row %+v", snap.Rows[0])
	}
	m, _, _ = s.FluxDashboard(context.Background(), "", nil, false)
	if got := dashboardText(m); !strings.HasPrefix(got, "Helm Releases 1: Not Ready 1; Kustomizations 4: Ready 3, Suspended 1") {
		t.Errorf("model with Helm releases %s", got)
	}

	// The search filters the table.
	s.UpdateUI(func(u *UIState) { u.Filter = "podinfo" })
	waitTable(t, s, FluxDashboard, func(t *TableSnapshot) bool { return t.Filter == "podinfo" && rowNames(t) == "podinfo" })
}

// TestFluxDashboardWithoutFlux: a cluster without Flux has an empty dashboard whose table says so.
func TestFluxDashboardWithoutFlux(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("default", "web-0"))
	s := newCRDSession(t, src)
	waitFor(t, "the definitions to be listed", func() bool { set := s.Backend().customSet(); return set != nil && set.synced })
	m, table, err := s.FluxDashboard(context.Background(), "", nil, true)
	if err != nil || len(m.Kinds) != 0 || table.Error != "The cluster serves no Flux kinds." {
		t.Fatalf("%+v %s %v", m, summary(table), err)
	}
	s.OpenTab(FluxDashboard, false) // as ui.openView does: the page's default sort
	waitTable(t, s, FluxDashboard, func(t *TableSnapshot) bool { return t.Error == "The cluster serves no Flux kinds." && len(t.Rows) == 0 })
}

// dashboardText renders a model as "Title count: State n, …; …".
func dashboardText(m *FluxDashboardModel) string {
	parts := make([]string, 0, len(m.Kinds))
	for _, k := range m.Kinds {
		states := make([]string, 0, len(k.States))
		for _, st := range k.States {
			if st.Status != kinds.FluxStateToken(st.State) || st.Title != kinds.FluxStateText(st.State) {
				return "bad state " + string(st.State)
			}
			states = append(states, st.Title+" "+strconv.Itoa(st.Count))
		}
		parts = append(parts, k.Title+" "+strconv.Itoa(k.Count)+": "+strings.Join(states, ", "))
	}
	return strings.Join(parts, "; ")
}

func rowNames(t *TableSnapshot) string {
	if t == nil {
		return "<nil>"
	}
	names := make([]string, len(t.Rows))
	for i, r := range t.Rows {
		names[i] = r.Ref.Name
	}
	return strings.Join(names, " ")
}
