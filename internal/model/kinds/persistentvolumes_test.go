package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzPersistentVolumes(f *testing.F) { fuzzProfile(f, "persistentvolumes") }

func TestPersistentVolumeTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"persistentvolumes", "healthy", "status", "Bound", model.StatusOK, model.StatusOK},
		{"persistentvolumes", "healthy", "capacity", "10.0 GiB", model.StatusNone, model.StatusOK},
		{"persistentvolumes", "healthy", "claim", "payments/data-postgres-0", model.StatusNone, model.StatusOK},
		{"persistentvolumes", "healthy", "storageClass", "fast-ssd", model.StatusNone, model.StatusOK},
		{"persistentvolumes", "available", "status", "Available", model.StatusOK, model.StatusOK},
		{"persistentvolumes", "available", "claim", "", model.StatusNone, model.StatusOK},
		{"persistentvolumes", "available", "storageClass", "", model.StatusNone, model.StatusOK},
		{"persistentvolumes", "released", "status", "Released", model.StatusWarning, model.StatusWarning},
		{"persistentvolumes", "failing", "status", "Failed", model.StatusError, model.StatusError},
		{"persistentvolumes", "legacy-class", "status", "Pending", model.StatusPending, model.StatusPending},
		{"persistentvolumes", "legacy-class", "storageClass", "slow", model.StatusNone, model.StatusPending},
		{"persistentvolumes", "terminating", "status", "Terminating", model.StatusTerminating, model.StatusTerminating},
		{"persistentvolumes", "minimal", "status", "Unknown", model.StatusUnknown, model.StatusUnknown},
		{"persistentvolumes", "minimal", "capacity", "-", model.StatusNone, model.StatusUnknown},
	})
	assertColumnOrder(t, "persistentvolumes", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "storageClass", Title: "Storage Class"},
		{ID: "capacity", Title: "Capacity"},
		{ID: "claim", Title: "Claim"},
		{ID: "age", Title: "Age"},
		{ID: "status", Title: "Status"},
	})
}

// TestPersistentVolumeLinks: Storage Class links to the class, Claim to the claim; a volume
// without them links nowhere.
func TestPersistentVolumeLinks(t *testing.T) {
	p, _ := kinds.ByKey("persistentvolumes")
	byName := map[string]model.Row{}
	for _, c := range loadCases(t, "persistentvolumes") {
		byName[c.name] = model.BuildRow(p, "test", c.obj)
	}
	class, claim := p.ColumnIndex("storageClass"), p.ColumnIndex("claim")
	healthy := byName["healthy"]
	if l := healthy.Cells[class].Link; l == nil || *l != (model.ObjectRef{Kind: "storageclasses", Name: "fast-ssd"}) {
		t.Errorf("class link %+v", l)
	}
	if l := healthy.Cells[claim].Link; l == nil || *l != (model.ObjectRef{Kind: "persistentvolumeclaims", Namespace: "payments", Name: "data-postgres-0"}) {
		t.Errorf("claim link %+v", l)
	}
	if a := byName["available"]; a.Cells[class].Link != nil || a.Cells[claim].Link != nil {
		t.Errorf("a volume without class and claim links: %+v", a.Cells)
	}
}

// TestPersistentVolumeCapacitySorts checks that Capacity sorts by bytes.
func TestPersistentVolumeCapacitySorts(t *testing.T) {
	p, _ := kinds.ByKey("persistentvolumes")
	col := p.ColumnIndex("capacity")
	got := map[string]int64{}
	for _, c := range loadCases(t, "persistentvolumes") {
		got[c.name] = model.BuildRow(p, "test", c.obj).Cells[col].Num
	}
	if got["legacy-class"] >= got["healthy"] || got["healthy"] >= got["failing"] || got["failing"] >= got["available"] {
		t.Errorf("capacities do not sort by bytes: %v", got)
	}
}
