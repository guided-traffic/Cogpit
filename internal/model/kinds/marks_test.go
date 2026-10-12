package kinds

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// A pod's Containers cell keeps "ready/total" to sort and search by and draws one square per
// container, init containers first: running in the Running colour, stopped grey (a finished init
// container too), waiting pending, or red on an error; each square names its container (owner,
// 2026-10-12, from Lens).
func TestContainerMarks(t *testing.T) {
	obj := pod(map[string]any{
		"initContainers": []any{
			map[string]any{"name": "migrate"},
			map[string]any{"name": "proxy", "restartPolicy": "Always"},
		},
		"containers": []any{
			map[string]any{"name": "app"},
			map[string]any{"name": "worker"},
			map[string]any{"name": "late"},
		},
	}, map[string]any{
		"phase": "Running",
		"initContainerStatuses": []any{
			map[string]any{"name": "migrate", "state": map[string]any{"terminated": map[string]any{"reason": "Completed", "exitCode": int64(0)}}},
			map[string]any{"name": "proxy", "ready": true, "started": true, "state": map[string]any{"running": map[string]any{}}},
		},
		"containerStatuses": []any{
			map[string]any{"name": "app", "ready": true, "state": map[string]any{"running": map[string]any{}}},
			map[string]any{"name": "worker", "state": map[string]any{"waiting": map[string]any{"reason": "CrashLoopBackOff"}}},
		},
	}, nil)
	marks := containerMarks(obj)
	if marks == nil {
		t.Fatal("no marks")
	}
	want := []model.CellMark{
		{Text: "migrate: terminated (Completed, exit code 0)", Status: model.StatusStopped},
		{Text: "proxy: running", Status: model.StatusOK},
		{Text: "app: running", Status: model.StatusOK},
		{Text: "worker: waiting (CrashLoopBackOff)", Status: model.StatusError},
		{Text: "late: waiting to start", Status: model.StatusPending},
	}
	if len(*marks) != len(want) {
		t.Fatalf("marks %+v", *marks)
	}
	for i, m := range *marks {
		if m != want[i] {
			t.Errorf("mark %d = %+v, want %+v", i, m, want[i])
		}
	}
	cell := containersCell(obj, summarisePod(obj))
	if cell.Text != "2/4" || cell.Marks == nil {
		t.Errorf("cell %+v: the readable form stays for sorting and search", cell)
	}
}
