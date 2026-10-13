package engine

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestArgoDashboard: the Dashboard counts the Applications by health and sync for the namespace
// filter, lists those that need attention as its table, and follows a change; agents read it
// built once from the cache while the page is closed.
func TestArgoDashboard(t *testing.T) {
	src := testkit.NewMemorySource()
	for _, crd := range testkit.ArgoCRDs() {
		src.Set(crdsGVR, crd)
	}
	src.Set(testkit.ArgoApplicationsGVR, testkit.ArgoApplication("argocd", "good", "Synced", "Healthy"))
	src.Set(testkit.ArgoApplicationsGVR, testkit.ArgoApplication("argocd", "drift", "OutOfSync", "Healthy"))
	src.Set(testkit.ArgoApplicationsGVR, testkit.ArgoApplication("argocd", "broken", "Synced", "Degraded"))
	src.Set(testkit.ArgoApplicationsGVR, testkit.ArgoApplication("team", "fresh", "", ""))
	src.Set(testkit.ArgoProjectsGVR, testkit.Object("argoproj.io/v1alpha1", "AppProject", "argocd", "default", nil))
	src.Set(testkit.SecretsGVR, testkit.ArgoRepoSecret("argocd", "repo", "https://git.example.com/apps.git", "fake-pw"))
	s := newCRDSession(t, src)
	waitFor(t, "Argo CD's kinds", func() bool { return hasKind(s, kinds.ArgoClustersKey) })

	// Closed page: built once from the cache.
	m, attention, err := s.ArgoCDDashboard(context.Background(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	strip := map[string]int{}
	for _, tile := range m.Strip {
		strip[tile.Label] = tile.Count
	}
	if !m.Served || !m.Synced || strip["Applications"] != 4 || strip["Synced"] != 2 || strip["Healthy"] != 2 || strip["Out of sync"] != 1 ||
		strip["Degraded"] != 1 || strip["Missing"] != 0 {
		t.Fatalf("model %+v", m)
	}
	if len(m.OutOfSync) != 1 || m.OutOfSync[0].Name != "drift" || len(m.All) != 4 {
		t.Errorf("sync and refresh sets %+v %+v", m.OutOfSync, m.All)
	}
	cards := map[string]int{}
	for _, k := range m.Kinds {
		cards[k.Kind] = k.Count
	}
	if cards[kinds.ArgoApplicationsKey] != 4 || cards[kinds.ArgoProjectsKey] != 1 || cards[kinds.ArgoRepositoriesKey] != 1 || cards[kinds.ArgoClustersKey] != 0 {
		t.Errorf("cards %+v", m.Kinds)
	}
	var names []string
	for _, r := range attention.Rows {
		names = append(names, r.Ref.Name)
	}
	if strings.Join(names, ",") != "broken,drift,fresh" {
		t.Errorf("needs attention %v", names)
	}

	// The open page follows the namespace filter and a change.
	s.UpdateUI(func(u *UIState) { u.View, u.Namespaces = ArgoCDDashboard, []string{"argocd"} })
	waitTable(t, s, ArgoCDDashboard, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 2 })
	src.Set(testkit.ArgoApplicationsGVR, testkit.ArgoApplication("argocd", "drift", "Synced", "Healthy"))
	waitTable(t, s, ArgoCDDashboard, func(t *TableSnapshot) bool { return len(t.Rows) == 1 && t.Rows[0].Ref.Name == "broken" })
	m, _, err = s.ArgoCDDashboard(context.Background(), nil, false)
	if err != nil || len(m.OutOfSync) != 0 || len(m.All) != 3 {
		t.Errorf("the open page's model %+v %v", m, err)
	}
	p, ok := s.ViewProfile(ArgoCDDashboard)
	if !ok || p.Columns[p.ColumnIndex("message")].Hidden || !p.Columns[p.ColumnIndex("project")].Hidden {
		t.Errorf("needs attention columns %+v", p)
	}
}

// TestArgoCDKindsFollowTheirDefinitions: Argo CD's kinds appear with their definitions, with
// Cogpit's curated profiles; the settings views come with the Application definition, list only
// the labelled Secrets and none of their credentials, and leave with it.
func TestArgoCDKindsFollowTheirDefinitions(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.ArgoApplicationsGVR, testkit.ArgoApplication("argocd", "shop", "OutOfSync", "Healthy"))
	src.Set(testkit.SecretsGVR, testkit.ArgoRepoSecret("argocd", "repo-shop", "https://git.example.com/shop.git", "fake-password-hunter2"))
	src.Set(testkit.SecretsGVR, testkit.ArgoClusterSecret("argocd", "prod", "https://prod.example.com:6443", "fake-bearer-token"))
	src.Set(testkit.SecretsGVR, testkit.Secret("argocd", "argocd-secret", nil, map[string]string{"admin.password": "fake-admin"}))
	s := newCRDSession(t, src)
	b := s.Backend()
	waitFor(t, "the definitions to be listed", func() bool { set := b.customSet(); return set != nil && set.synced })
	for _, key := range []string{kinds.ArgoApplicationsKey, kinds.ArgoRepositoriesKey, kinds.ArgoClustersKey} {
		if hasKind(s, key) {
			t.Fatalf("%s offered without Argo CD's definitions", key)
		}
	}

	for _, crd := range testkit.ArgoCRDs() {
		src.Set(crdsGVR, crd)
	}
	waitFor(t, "Argo CD's kinds", func() bool { return hasKind(s, kinds.ArgoClustersKey) && hasKind(s, kinds.ArgoProjectsKey) })
	for _, want := range append(kinds.ArgoProfiles(), kinds.ArgoSettings()...) {
		if p, _ := b.Profile(want.Key); p != want {
			t.Errorf("%s: profile %q, want the curated one", want.Key, p.Title)
		}
	}

	openView(s, kinds.ArgoApplicationsKey)
	apps := waitTable(t, s, kinds.ArgoApplicationsKey, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 1 })
	p, _ := b.Profile(kinds.ArgoApplicationsKey)
	if c := apps.Rows[0].Cells; c[p.ColumnIndex("sync")].Text != "OutOfSync" || c[p.ColumnIndex("health")].Text != "Healthy" {
		t.Errorf("application row %+v", c)
	}

	openView(s, kinds.ArgoRepositoriesKey)
	repos := waitTable(t, s, kinds.ArgoRepositoriesKey, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 1 })
	rp, _ := b.Profile(kinds.ArgoRepositoriesKey)
	if r := repos.Rows[0]; r.Ref.Name != "repo-shop" || r.Cells[rp.ColumnIndex("url")].Text != "https://git.example.com/shop.git" {
		t.Errorf("repository row %+v", r)
	}
	openView(s, kinds.ArgoClustersKey)
	clusters := waitTable(t, s, kinds.ArgoClustersKey, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 1 })
	m, err := s.Detail(clusters.Rows[0].Ref)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range []any{repos, clusters, m} {
		raw, _ := json.Marshal(v)
		for _, f := range []string{"fake-password-hunter2", "deploy-bot", "fake-bearer-token", "bearerToken\":", "fake-admin", "argocd-secret"} {
			if strings.Contains(string(raw), f) {
				t.Errorf("%q reached %T", f, v)
			}
		}
	}
	if !strings.Contains(m.YAML, "server: https://prod.example.com:6443") || !strings.Contains(m.YAML, "- config") {
		t.Errorf("the cluster's YAML:\n%s", m.YAML)
	}

	// An Application Set counts the Applications it owns, from the cache.
	set := testkit.Object("argoproj.io/v1alpha1", "ApplicationSet", "argocd", "shops", map[string]any{
		"spec": map[string]any{"generators": []any{map[string]any{"list": map[string]any{}}}},
	})
	src.Set(testkit.ArgoAppSetsGVR, set)
	owned := testkit.ArgoApplication("argocd", "shop-eu", "Synced", "Healthy")
	owned.SetOwnerReferences([]metav1.OwnerReference{{APIVersion: "argoproj.io/v1alpha1", Kind: "ApplicationSet", Name: "shops", UID: set.GetUID()}})
	src.Set(testkit.ArgoApplicationsGVR, owned)
	openView(s, kinds.ArgoApplicationSetKey)
	sp, _ := b.Profile(kinds.ArgoApplicationSetKey)
	waitTable(t, s, kinds.ArgoApplicationSetKey, func(t *TableSnapshot) bool {
		return t.Synced && len(t.Rows) == 1 && t.Rows[0].Cells[sp.ColumnIndex("applications")].Text == "1"
	})
	src.Delete(testkit.ArgoApplicationsGVR, "argocd", "shop-eu")

	// The Dashboard: the strip, the cards, Needs attention with the out-of-sync Application.
	openView(s, ArgoCDDashboard)
	snap := waitTable(t, s, ArgoCDDashboard, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 1 })
	if snap.Title != "Dashboard" || snap.Rows[0].Ref.Kind != kinds.ArgoApplicationsKey || snap.Rows[0].Ref.Name != "shop" {
		t.Errorf("needs attention %+v", snap)
	}

	src.Delete(crdsGVR, "", kinds.ArgoApplicationsKey)
	waitFor(t, "the settings views to leave with the definition", func() bool {
		return !hasKind(s, kinds.ArgoRepositoriesKey) && !hasKind(s, kinds.ArgoClustersKey) && hasKind(s, kinds.ArgoProjectsKey)
	})
}
