package engine

import (
	"testing"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestVerticalPodAutoscalersFollowTheirDefinition: the Vertical Pod Autoscalers view exists only
// while the cluster's VerticalPodAutoscaler definition is established, uses Cogpit's curated
// profile (Lens's documented columns) rather than the generic one, and shows the autoscalers.
func TestVerticalPodAutoscalersFollowTheirDefinition(t *testing.T) {
	const key = "verticalpodautoscalers.autoscaling.k8s.io"
	vpas := schema.GroupVersionResource{Group: "autoscaling.k8s.io", Version: "v1", Resource: "verticalpodautoscalers"}
	crd := testkit.CRD("autoscaling.k8s.io", "verticalpodautoscalers", "VerticalPodAutoscaler", true,
		testkit.CRDVersion{Name: "v1", Served: true, Storage: true})
	src := testkit.NewMemorySource()
	src.Set(testkit.PodsGVR, testkit.Pod("default", "web-0"))
	src.Set(vpas, testkit.Object("autoscaling.k8s.io/v1", "VerticalPodAutoscaler", "shop", "web", map[string]any{
		"spec": map[string]any{
			"targetRef":    map[string]any{"apiVersion": "apps/v1", "kind": "Deployment", "name": "web"},
			"updatePolicy": map[string]any{"updateMode": "Recreate"},
		},
		"status": map[string]any{"conditions": []any{map[string]any{"type": "RecommendationProvided", "status": "True"}}},
	}))
	s := newCRDSession(t, src)
	b := s.Backend()
	waitFor(t, "the definitions to be listed", func() bool { set := b.customSet(); return set != nil && set.synced })
	if hasKind(s, key) {
		t.Fatal("Vertical Pod Autoscalers offered without their definition")
	}

	src.Set(crdsGVR, crd)
	waitFor(t, "the Vertical Pod Autoscalers kind", func() bool { return hasKind(s, key) })
	p, _ := b.Profile(key)
	if p != kinds.Curated()[0] || p.Title != "Vertical Pod Autoscalers" {
		t.Fatalf("profile %q at %s, want the curated one", p.Title, p.Version)
	}
	openView(s, key)
	snap := waitTable(t, s, key, func(t *TableSnapshot) bool { return t.Synced && len(t.Rows) == 1 })
	mode, status := p.ColumnIndex("mode"), p.ColumnIndex("status")
	if r := snap.Rows[0]; r.Cells[mode].Text != "Recreate" || r.Cells[status].Text != "RecommendationProvided" {
		t.Errorf("row %+v", r.Cells)
	}

	src.Delete(crdsGVR, "", key)
	waitFor(t, "the kind to disappear", func() bool { return !hasKind(s, key) })
}
