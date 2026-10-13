package kinds

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// runtimeClasses has the columns of Lens's documentation (not captured): Name, Handler, Age. A
// runtime class has no state, so no status token. The drawer (Handler, the pod overhead and the
// scheduling constraints) is inferred.
var runtimeClasses = &model.KindProfile{
	Key:      "runtimeclasses",
	Group:    "node.k8s.io",
	Version:  "v1",
	Resource: "runtimeclasses",
	Kind:     "RuntimeClass",
	Title:    "Runtime Classes",
	Columns: []model.Column{
		nameColumn,
		{ID: "handler", Title: "Handler", Type: model.ColumnText},
		ageColumn,
	},
	Cells: runtimeClassCells,
	Fields: map[string]fieldFunc{
		"handler": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "handler"))
		},
	},
	Detail: runtimeClassDetail,
}

func runtimeClassCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		{Text: model.NestedString(obj, "handler")},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

func runtimeClassDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.add("Handler", model.NestedString(obj, "handler"))
	var overhead []string
	podFixed := model.NestedMap(obj, "overhead", "podFixed")
	for _, k := range sortedKeys(podFixed) {
		if v := model.NestedIntOrString(podFixed, k); v != "" {
			overhead = append(overhead, k+"="+model.FormatResource(k, v)) // bytes in binary units (D-32)
		}
	}
	d.add("Pod Overhead", strings.Join(overhead, ", "))
	d.add("Node Selector", strings.Join(model.SortedPairs(model.NestedStringMap(obj, "scheduling", "nodeSelector")), "\n"))
	var tolerations []string
	for _, t := range model.NestedSlice(obj, "scheduling", "tolerations") {
		if text := tolerationText(t); text != "" {
			tolerations = append(tolerations, text)
		}
	}
	d.add("Tolerations", strings.Join(tolerations, "\n"))
	return withSection(nil, model.TitleProperties, d)
}
