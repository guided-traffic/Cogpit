package engine

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

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

	src.Delete(crdsGVR, "", kinds.ArgoApplicationsKey)
	waitFor(t, "the settings views to leave with the definition", func() bool {
		return !hasKind(s, kinds.ArgoRepositoriesKey) && !hasKind(s, kinds.ArgoClustersKey) && hasKind(s, kinds.ArgoProjectsKey)
	})
}
