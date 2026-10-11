package kinds

import (
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// secrets has the columns of Lens's documentation (not captured). The cache holds Secrets as
// metadata only (D-13, H-8 option b): an object's metadata carries neither its keys nor its type,
// so the Keys and Type cells stay empty for every Secret the cache holds, a documented difference
// from Lens. A full Secret (the cache never holds one) fills them. A Secret has no state, so no
// status token (inferred). The drawer's Type and Data come from the engine's GET of the Secret the
// drawer shows, never from this profile.
var secrets = &model.KindProfile{
	Key:          keySecrets,
	Version:      "v1",
	Resource:     keySecrets,
	Kind:         "Secret",
	Title:        "Secrets",
	Namespaced:   true,
	MetadataOnly: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "labels", Title: "Labels", Type: model.ColumnText},
		{ID: "keys", Title: "Keys", Type: model.ColumnText},
		typeColumn,
		ageColumn,
	},
	Cells:  secretCells,
	Detail: secretDetail,
}

// keySecrets is the kind key and resource of Secrets.
const keySecrets = "secrets"

func secretCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: strings.Join(model.SortedPairs(model.NestedStringMap(obj, "metadata", "labels")), ", ")},
		{Text: strings.Join(secretKeys(obj), ", ")},
		{Text: model.NestedString(obj, idType)},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// secretKeys returns the keys of a full Secret's data and stringData, sorted, each once; nil for
// the metadata the cache holds. Only the keys: a profile never reads a Secret's values.
func secretKeys(obj any) []string {
	data, plain := model.NestedMap(obj, "data"), model.NestedMap(obj, "stringData")
	if len(data)+len(plain) == 0 {
		return nil
	}
	keys := make([]string, 0, len(data)+len(plain))
	for k := range data {
		keys = append(keys, k)
	}
	for k := range plain {
		if _, dup := data[k]; !dup {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// secretDetail adds the Type to the drawer's Properties when the object carries one; the
// metadata the cache holds does not, and the engine adds the Type and the Data it fetched.
func secretDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var props detailItems
	props.add(typeColumn.Title, model.NestedString(obj, idType))
	return withSection(nil, model.TitleProperties, props)
}
