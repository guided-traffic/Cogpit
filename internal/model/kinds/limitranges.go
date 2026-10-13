package kinds

import (
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Column titles of the drawer tables of Limit Ranges, Resource Quotas and Horizontal Pod
// Autoscalers.
const (
	titleType     = "Type"
	titleResource = "Resource"
)

// limitRangeBounds are the bounds of a limit range entry, in the drawer's column order. A ratio
// is a plain number, never bytes.
var limitRangeBounds = []struct {
	field, title string
	ratio        bool
}{
	{"min", "Min", false},
	{"max", "Max", false},
	{"defaultRequest", "Default Request", false},
	{"default", "Default Limit", false},
	{"maxLimitRequestRatio", "Max Limit/Request Ratio", true},
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
// resources of each type sorted, a bound the entry does not set left empty, a bound of bytes in
// binary units (D-32).
func limitRangeDetail(obj *unstructured.Unstructured) []model.DetailSection {
	t := &model.DetailTable{Columns: []string{titleType, titleResource}}
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
			row := make([]string, 0, 2+len(limitRangeBounds))
			row = append(row, typ, r)
			for _, b := range limitRangeBounds {
				v := model.NestedIntOrString(l, b.field, r)
				if !b.ratio {
					v = model.FormatResource(r, v)
				}
				row = append(row, v)
			}
			t.Rows = append(t.Rows, row)
		}
	}
	if len(t.Rows) == 0 {
		return nil
	}
	return []model.DetailSection{{Title: "Limits", Table: t}}
}
