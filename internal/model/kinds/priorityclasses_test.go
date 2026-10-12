package kinds_test

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzPriorityClasses(f *testing.F) { fuzzProfile(f, "priorityclasses") }

func TestPriorityClassesStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// The goldens and these texts are built without the pods: the Pods column reads 0.
	checkStatusTexts(t, []statusText{
		{"priorityclasses", "healthy", "value", "1000", none, none},
		{"priorityclasses", "healthy", "globalDefault", "true", none, none},
		{"priorityclasses", "failing", "value", "-10", none, none},
		{"priorityclasses", "failing", "globalDefault", "false", none, none},
		{"priorityclasses", "minimal", "globalDefault", "false", none, none},
		{"priorityclasses", "minimal", "pods", "0", none, none},
		{"priorityclasses", "terminating", "value", "100000", none, model.StatusTerminating},
	})
}

// indexedRelated answers lookups from a map of index id and key to objects.
type indexedRelated map[string][]*unstructured.Unstructured

// Listed: every index of the fake is listed.
func (f indexedRelated) Listed(model.RelatedIndex) bool { return true }

func (f indexedRelated) Objects(ix model.RelatedIndex, key string) []*unstructured.Unstructured {
	return f[ix.ID()+"/"+key]
}

// TestPriorityClassPodCounts checks the pod counts (Lens 2026.9): the Pods column and the drawer
// read the pods whose spec.priorityClassName names the class, through PodsByPriorityClass.
func TestPriorityClassPodCounts(t *testing.T) {
	pod := func(ns, name, class string) *unstructured.Unstructured {
		p := &unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{"priorityClassName": class}}}
		p.SetNamespace(ns)
		p.SetName(name)
		return p
	}
	ix := kinds.PodsByPriorityClass
	if ix.Kind != "pods" {
		t.Fatalf("PodsByPriorityClass indexes %q", ix.Kind)
	}
	if got := ix.Keys(pod("shop", "web-1", "standard")); len(got) != 1 || got[0] != "standard" {
		t.Errorf("keys of a pod with a class: %q", got)
	}
	if got := ix.Keys(pod("shop", "web-2", "")); len(got) != 0 {
		t.Errorf("keys of a pod without a class: %q", got)
	}

	p, _ := kinds.ByKey("priorityclasses")
	if len(p.Needs) != 1 || p.Needs[0].ID() != ix.ID() {
		t.Fatalf("Needs = %+v", p.Needs)
	}
	rel := indexedRelated{ix.ID() + "/standard": {pod("shop", "web-1", "standard"), pod("batch", "etl-1", "standard")}}
	for _, c := range loadCases(t, "priorityclasses") {
		if c.name != "healthy" {
			continue
		}
		row := model.BuildRowWith(p, "test", c.obj, rel)
		i := p.ColumnIndex("pods")
		if got := model.CellText(p.Columns[i], row.Cells[i], now); got != "2" {
			t.Errorf("Pods column %q, want 2", got)
		}
		var pods string
		for _, it := range p.DetailSections(c.obj, rel)[0].Items {
			if it.Name == "Pods" {
				pods = it.Value
			}
		}
		if pods != "2" {
			t.Errorf("drawer's Pods %q, want 2", pods)
		}
	}
}
