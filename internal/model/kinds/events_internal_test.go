package kinds

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

func asUnstructured(obj map[string]any) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: obj}
}

// TestEventInvolvedObjectLinks pins the Involved Object link of the Events list: built-in kinds
// link by their key; a custom kind stays unlinked, because its key needs the plural.
func TestEventInvolvedObjectLinks(t *testing.T) {
	ev := func(ref map[string]any) map[string]any {
		return map[string]any{"metadata": map[string]any{"name": "e", "namespace": "shop"}, "involvedObject": ref}
	}
	tests := []struct {
		obj  map[string]any
		want *model.ObjectRef
	}{
		{ev(map[string]any{"apiVersion": "v1", "kind": "Pod", "namespace": "shop", "name": "web-0"}), &model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-0"}},
		{ev(map[string]any{"apiVersion": "apps/v1", "kind": "Deployment", "name": "web"}), &model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"}},
		{ev(map[string]any{"kind": "Node", "name": "worker-1"}), &model.ObjectRef{Kind: "nodes", Name: "worker-1"}},
		{ev(map[string]any{"apiVersion": "cert-manager.io/v1", "kind": "Certificate", "name": "tls"}), nil},
		{map[string]any{"regarding": map[string]any{"apiVersion": "batch/v1", "kind": "Job", "namespace": "ci", "name": "build"}}, &model.ObjectRef{Kind: "jobs", Namespace: "ci", Name: "build"}},
	}
	col := events.ColumnIndex("involvedObject")
	for _, tt := range tests {
		cells, _ := eventCells(asUnstructured(tt.obj))
		got := cells[col].Link
		if (got == nil) != (tt.want == nil) || (got != nil && *got != *tt.want) {
			t.Errorf("%v: link %+v, want %+v", tt.obj, got, tt.want)
		}
	}
}
