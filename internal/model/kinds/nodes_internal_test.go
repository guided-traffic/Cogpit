package kinds

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// TestNodeWarningsAndConditions pins the Nodes list as Lens shows it (capture, 2026-10-11): a
// cordoned Ready node reads "SchedulingDisabled Ready", only Ready green, with no warning icon and
// no warning row. Which nodes do get the icon is inferred: not Ready, or a problem condition.
func TestNodeWarningsAndConditions(t *testing.T) {
	cordoned := map[string]any{
		"spec": map[string]any{"unschedulable": true},
		"status": map[string]any{"conditions": []any{
			map[string]any{"type": "Ready", "status": "True"},
			map[string]any{"type": "MemoryPressure", "status": "False"},
		}},
	}
	s := summariseNode(cordoned)
	c := s.conditionsCell()
	want := []model.CellPart{{Text: "SchedulingDisabled"}, {Text: "Ready", Status: model.StatusOK}}
	if c.Parts == nil || len(*c.Parts) != len(want) || (*c.Parts)[0] != want[0] || (*c.Parts)[1] != want[1] {
		t.Fatalf("parts = %+v, want %+v", c.Parts, want)
	}
	if c.Text != "SchedulingDisabled Ready" || c.Status != model.StatusOK || s.token != model.StatusOK {
		t.Errorf("cordoned: %q %q row %q", c.Text, c.Status, s.token)
	}
	if w := s.warning(); w != (model.Cell{}) {
		t.Errorf("a cordoned Ready node must not warn: %+v", w)
	}

	// A Ready node under pressure is a warning row and warns; the Ready part stays green.
	pressure := map[string]any{"status": map[string]any{"conditions": []any{
		map[string]any{"type": "Ready", "status": "True"},
		map[string]any{"type": "DiskPressure", "status": "True"},
	}}}
	s = summariseNode(pressure)
	if c := s.conditionsCell(); c.Text != "Ready DiskPressure" || (*c.Parts)[0].Status != model.StatusOK || (*c.Parts)[1].Status != model.StatusWarning {
		t.Errorf("pressure cell = %+v %+v", c, c.Parts)
	}
	if s.token != model.StatusWarning || s.warning().Text != "DiskPressure" {
		t.Errorf("pressure: row %q, warning %+v", s.token, s.warning())
	}

	// The node controller's Unknown: the kubelet stopped reporting.
	unknown := map[string]any{"status": map[string]any{"conditions": []any{
		map[string]any{"type": "Ready", "status": "Unknown"},
	}}}
	if w := summariseNode(unknown).warning(); w.Status != model.StatusWarning || w.Num != 1 {
		t.Errorf("unknown ready = %+v", w)
	}
}

func TestNodeDetailTables(t *testing.T) {
	obj := map[string]any{"status": map[string]any{
		"capacity": map[string]any{"pods": "110", "nvidia.com/gpu": "2", "cpu": "8", "memory": "32Gi", "hugepages-2Mi": "0"},
	}}
	tbl := resourceTable(obj, "capacity")
	if tbl == nil {
		t.Fatal("no table")
	}
	wantCols := []string{"CPU", "Memory", "Hugepages 2Mi", "Pods", "nvidia.com/gpu"}
	wantRow := []string{"8", "32Gi", "0", "110", "2"}
	if len(tbl.Columns) != len(wantCols) || len(tbl.Rows) != 1 {
		t.Fatalf("table = %+v", tbl)
	}
	for i := range wantCols {
		if tbl.Columns[i] != wantCols[i] || tbl.Rows[0][i] != wantRow[i] {
			t.Errorf("column %d = %s %s, want %s %s", i, tbl.Columns[i], tbl.Rows[0][i], wantCols[i], wantRow[i])
		}
	}
	if resourceTable(map[string]any{}, "allocatable") != nil {
		t.Error("a node without allocatable resources should have no table")
	}
}
