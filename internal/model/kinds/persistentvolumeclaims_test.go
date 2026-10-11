package kinds_test

import (
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzPersistentVolumeClaims(f *testing.F) { fuzzProfile(f, "persistentvolumeclaims") }

func TestPersistentVolumeClaimTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"persistentvolumeclaims", "healthy", "status", "Bound", model.StatusOK, model.StatusOK},
		{"persistentvolumeclaims", "healthy", "size", "10Gi", model.StatusNone, model.StatusOK},
		{"persistentvolumeclaims", "healthy", "storageClass", "fast-ssd", model.StatusNone, model.StatusOK},
		{"persistentvolumeclaims", "failing", "status", "Lost", model.StatusError, model.StatusError},
		{"persistentvolumeclaims", "pending", "status", "Pending", model.StatusPending, model.StatusPending},
		{"persistentvolumeclaims", "pending", "size", "1Gi", model.StatusNone, model.StatusPending}, // the request: no capacity yet
		{"persistentvolumeclaims", "terminating", "status", "Terminating", model.StatusTerminating, model.StatusTerminating},
		{"persistentvolumeclaims", "minimal", "status", "Unknown", model.StatusUnknown, model.StatusUnknown},
		{"persistentvolumeclaims", "minimal", "size", "-", model.StatusNone, model.StatusUnknown},
		// A resize shows the capacity until it is done; the drawer has the requested size.
		{"persistentvolumeclaims", "resize-pending", "size", "10Gi", model.StatusNone, model.StatusOK},
		{"persistentvolumeclaims", "resize-failed", "status", "Bound", model.StatusOK, model.StatusOK},
		{"persistentvolumeclaims", "static-binding", "storageClass", "", model.StatusNone, model.StatusOK},
		{"persistentvolumeclaims", "legacy-class", "storageClass", "slow", model.StatusNone, model.StatusPending},
		// Pods needs a cross-kind index; the column keeps its place and stays empty.
		{"persistentvolumeclaims", "healthy", "pods", "", model.StatusNone, model.StatusOK},
	})
	assertColumnOrder(t, "persistentvolumeclaims", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "storageClass", Title: "Storage Class"},
		{ID: "size", Title: "Size"},
		{ID: "pods", Title: "Pods"},
		{ID: "age", Title: "Age"},
		{ID: "status", Title: "Status"},
	})
}

// TestPersistentVolumeClaimSizeSorts checks that Size sorts by bytes, not by text, and that
// quantities resource.ParseQuantity would spend unbounded time on are skipped.
func TestPersistentVolumeClaimSizeSorts(t *testing.T) {
	p, _ := kinds.ByKey("persistentvolumeclaims")
	col := p.ColumnIndex("size")
	size := func(q string) model.Cell {
		obj := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "c", "namespace": "n"},
			"spec":     map[string]any{"resources": map[string]any{"requests": map[string]any{"storage": q}}},
		}}
		start := time.Now()
		cells, _ := p.Cells(obj)
		if d := time.Since(start); d > time.Second {
			t.Errorf("%q took %v", q, d)
		}
		return cells[col]
	}
	tests := []struct {
		q    string
		want int64
	}{
		{"500Mi", 500 << 20},
		{"1Gi", 1 << 30},
		{"10Gi", 10 << 30},
		{"1Ti", 1 << 40},
		{"2E", 2e18},
		{"1Ei", 1 << 60},
		{"1e9", 1e9},
		{"1e-2000000000", 0},
		{"1e2000000000", 0},
		{"garbage", 0},
	}
	for _, tt := range tests {
		if c := size(tt.q); c.Num != tt.want || c.Text != tt.q {
			t.Errorf("size %q = %+v, want num %d", tt.q, c, tt.want)
		}
	}
}
