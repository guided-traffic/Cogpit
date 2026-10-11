package kinds

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// limitRangeBounds are the bounds of a limit range entry, in the drawer's column order.
var limitRangeBounds = []struct{ field, title string }{
	{"min", "Min"},
	{"max", "Max"},
	{"defaultRequest", "Default Request"},
	{"default", "Default Limit"},
	{"maxLimitRequestRatio", "Max Limit/Request Ratio"},
}

// limitRanges has the columns of Lens's documentation (not captured): Name, Namespace, Age. A limit
// range has no state, so no status token. The drawer's Limits table, one row per type and resource
// with its bounds, is inferred.
var limitRanges = &model.KindProfile{
	Key:        "limitranges",
	Version:    "v1",
	Resource:   "limitranges",
	Kind:       "LimitRange",
	Title:      "Limit Ranges",
	Namespaced: true,
	Columns:    []model.Column{nameColumn, namespaceColumn, ageColumn},
	Cells:      limitRangeCells,
	Fields: map[string]fieldFunc{
		// "type" lists the limit types: "Container", "Pod", "PersistentVolumeClaim".
		idType: func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, l := range model.NestedSlice(obj, "spec", "limits") {
				out = appendUnique(out, model.NestedString(l, "type"))
			}
			return out
		},
	},
	Detail: limitRangeDetail,
}

func limitRangeCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// limitRangeDetail renders spec.limits as one table: a row per limit type and resource, the
// resources of each type sorted, a bound the entry does not set left empty.
func limitRangeDetail(obj *unstructured.Unstructured) []model.DetailSection {
	t := &model.DetailTable{Columns: []string{"Type", "Resource"}}
	for _, b := range limitRangeBounds {
		t.Columns = append(t.Columns, b.title)
	}
	for _, l := range model.NestedSlice(obj, "spec", "limits") {
		var resources []string
		for _, b := range limitRangeBounds {
			resources = appendUnique(resources, sortedKeys(model.NestedMap(l, b.field))...)
		}
		typ := model.NestedString(l, "type")
		sort.Strings(resources)
		for _, r := range resources {
			row := []string{typ, r}
			for _, b := range limitRangeBounds {
				row = append(row, model.NestedIntOrString(l, b.field, r))
			}
			t.Rows = append(t.Rows, row)
		}
	}
	if len(t.Rows) == 0 {
		return nil
	}
	return []model.DetailSection{{Title: "Limits", Table: t}}
}
