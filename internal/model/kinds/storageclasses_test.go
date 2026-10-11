package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzStorageClasses(f *testing.F) { fuzzProfile(f, "storageclasses") }

func TestStorageClassTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"storageclasses", "healthy", "provisioner", "rancher.io/local-path", model.StatusNone, model.StatusNone},
		{"storageclasses", "healthy", "volumeBindingMode", "WaitForFirstConsumer", model.StatusNone, model.StatusNone},
		{"storageclasses", "failing", "reclaimPolicy", "Retain", model.StatusNone, model.StatusNone},
		{"storageclasses", "terminating", "provisioner", "nfs.csi.example.com", model.StatusNone, model.StatusTerminating},
		// The API server defaults the reclaim policy to Delete and the binding mode to Immediate.
		{"storageclasses", "minimal", "reclaimPolicy", "Delete", model.StatusNone, model.StatusNone},
		{"storageclasses", "minimal", "volumeBindingMode", "Immediate", model.StatusNone, model.StatusNone},
	})
	// Documented columns (not captured): no Namespace, Age before the policies.
	assertColumnOrder(t, "storageclasses", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "provisioner", Title: "Provisioner"},
		{ID: "age", Title: "Age"},
		{ID: "reclaimPolicy", Title: "Reclaim Policy"},
		{ID: "volumeBindingMode", Title: "Volume Binding Mode"},
	})
	if p, _ := kinds.ByKey("storageclasses"); p.Namespaced || p.GVR().Group != "storage.k8s.io" {
		t.Errorf("storageclasses: namespaced %v, group %q", p.Namespaced, p.GVR().Group)
	}
}

// TestStorageClassDefault: only "true" in the annotation, or in its beta predecessor, makes the
// default class.
func TestStorageClassDefault(t *testing.T) {
	p, _ := kinds.ByKey("storageclasses")
	for _, c := range loadCases(t, "storageclasses") {
		want := c.name == "healthy" || c.name == "beta-default"
		if got := model.FieldValues(p, "default", c.obj); len(got) != 1 || (got[0] == "true") != want {
			t.Errorf("%s: default %q, want %v", c.name, got, want)
		}
	}
}
