package commands_test

import (
	"context"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// claimsConnector serves the test profiles and the built-in Persistent Volume Claims profile,
// whose Pods column reads the pods through the related-objects index.
type claimsConnector struct{ src *testkit.MemorySource }

func (c *claimsConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: "test", Current: true}}, nil
}

func (c *claimsConnector) Connect(_ context.Context, name string) (*engine.Backend, error) {
	pvc, _ := kinds.ByKey("persistentvolumeclaims")
	ch := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	return &engine.Backend{Context: name, Namespace: "default", Cache: ch, Profiles: append(testProfiles(), pvc), Close: ch.Close}, nil
}

// TestNavigateKeepsAFilterARelatedCellMatches: the claims view filtered by a pod's name shows the
// claim that pod mounts (its Pods cell); ui.navigate to that claim keeps the filter, and resets
// one that hides it.
func TestNavigateKeepsAFilterARelatedCellMatches(t *testing.T) {
	src := testkit.NewMemorySource()
	claims := schema.GroupVersionResource{Version: "v1", Resource: "persistentvolumeclaims"}
	src.Set(claims, testkit.Object("v1", "PersistentVolumeClaim", "apps", "data", map[string]any{"status": map[string]any{"phase": "Bound"}}))
	pod := testkit.Pod("apps", "db-0")
	_ = unstructured.SetNestedSlice(pod.Object, []any{map[string]any{"name": "data", "persistentVolumeClaim": map[string]any{"claimName": "data"}}}, "spec", "volumes")
	src.Set(testkit.PodsGVR, pod)

	s := engine.New(engine.Options{Connector: &claimsConnector{src: src}, SnapshotInterval: 10 * time.Millisecond})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	h := &harness{s: s, src: src}
	gui := engine.OriginGUI
	h.must(t, gui, "ui.openView", map[string]any{"view": "persistentvolumeclaims"}, nil)
	h.must(t, gui, "ui.setFilter", map[string]any{"filter": "db-0"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool {
		tbl := s.Table("persistentvolumeclaims")
		return tbl != nil && tbl.Synced && len(tbl.Rows) == 1
	}) {
		t.Fatalf("the filtered claims view: %+v", s.Table("persistentvolumeclaims"))
	}

	claim := model.ObjectRef{Context: "test", Kind: "persistentvolumeclaims", Namespace: "apps", Name: "data"}
	h.must(t, gui, "ui.navigate", map[string]any{"ref": claim}, nil)
	if st := s.UIState(); st.Filter != "db-0" || st.Drawer == nil || st.Drawer.Name != "data" {
		t.Errorf("after navigating to a claim the filter shows: filter %q, drawer %+v", st.Filter, st.Drawer)
	}

	h.must(t, gui, "ui.setFilter", map[string]any{"filter": "web-9"}, nil)
	h.must(t, gui, "ui.navigate", map[string]any{"ref": claim}, nil)
	if st := s.UIState(); st.Filter != "" {
		t.Errorf("a filter that hides the claim was kept: %q", st.Filter)
	}
}
