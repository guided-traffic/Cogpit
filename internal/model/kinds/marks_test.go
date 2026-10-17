package kinds

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// A pod's Containers cell keeps "ready/total" to sort and search by and draws one square per
// container, init containers first: running in the Running colour, stopped grey (a finished init
// container too), waiting pending, or red on an error; each square names its container (owner,
// 2026-10-12, from Lens). A running container that is not ready is orange (the owner's F23 of
// 2026-10-17); a plain init container, which Kubernetes never marks ready while it runs, stays in the
// Running colour, a sidecar (restartPolicy Always) does not.
func TestContainerMarks(t *testing.T) {
	obj := pod(map[string]any{
		"initContainers": []any{
			map[string]any{"name": "migrate"},
			map[string]any{"name": "proxy", "restartPolicy": "Always"},
			map[string]any{"name": "seed"},
			map[string]any{"name": "mesh", "restartPolicy": "Always"},
		},
		"containers": []any{
			map[string]any{"name": "app"},
			map[string]any{"name": "worker"},
			map[string]any{"name": "late"},
			map[string]any{"name": "warming"},
		},
	}, map[string]any{
		"phase": "Running",
		"initContainerStatuses": []any{
			map[string]any{"name": "migrate", "state": map[string]any{"terminated": map[string]any{"reason": "Completed", "exitCode": int64(0)}}},
			map[string]any{"name": "proxy", "ready": true, "started": true, "state": map[string]any{"running": map[string]any{}}},
			map[string]any{"name": "seed", "ready": false, "state": map[string]any{"running": map[string]any{}}},
			map[string]any{"name": "mesh", "ready": false, "started": true, "state": map[string]any{"running": map[string]any{}}},
		},
		"containerStatuses": []any{
			map[string]any{"name": "app", "ready": true, "state": map[string]any{"running": map[string]any{}}},
			map[string]any{"name": "worker", "state": map[string]any{"waiting": map[string]any{"reason": "CrashLoopBackOff"}}},
			map[string]any{"name": "warming", "ready": false, "started": true, "state": map[string]any{"running": map[string]any{}}},
		},
	}, nil)
	marks := containerMarks(obj)
	if marks == nil {
		t.Fatal("no marks")
	}
	want := []model.CellMark{
		{Text: "migrate: terminated (Completed, exit code 0)", Status: model.StatusStopped},
		{Text: "proxy: running", Status: model.StatusOK},
		{Text: "seed: running", Status: model.StatusOK},
		{Text: "mesh: running, not ready", Status: model.StatusWarning},
		{Text: "app: running", Status: model.StatusOK},
		{Text: "worker: waiting (CrashLoopBackOff)", Status: model.StatusError},
		{Text: "late: waiting to start", Status: model.StatusPending},
		{Text: "warming: running, not ready", Status: model.StatusWarning},
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
	if cell.Text != "2/6" || cell.Marks == nil { // the sidecars count, plain init containers do not
		t.Errorf("cell %+v: the readable form stays for sorting and search", cell)
	}
}
