package helm

import (
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// detail is the profile's Detail, Cogpit's own (Lens's release drawer is not captured): the
// release's summary in Properties, its history as a table, then the newest revision's
// user-supplied values and its manifest. The values are private: agents read model.Withheld in
// their place (object_get), because values can hold credentials; the manifest has every Secret's
// data redacted for everyone, as the Secrets drawer masks values.
func (m *memo) detail(o *unstructured.Unstructured) []model.DetailSection {
	status := model.NestedString(o, fieldStatus)
	var props []model.DetailItem
	add := func(it model.DetailItem) {
		if it.Value != "" {
			props = append(props, it)
		}
	}
	rev, _ := model.NestedInt64(o, fieldRevision)
	add(model.DetailItem{Name: titleChart, Value: chartText(model.NestedString(o, fieldChart), model.NestedString(o, fieldChartVersion))})
	add(model.DetailItem{Name: titleRevision, Value: strconv.FormatInt(rev, 10)})
	add(model.DetailItem{Name: titleAppVersion, Value: model.NestedString(o, fieldAppVersion)})
	add(model.DetailItem{Name: titleStatus, Value: status, Status: StatusToken(status)})
	add(model.DetailItem{Name: titleUpdated, Value: model.NestedString(o, fieldUpdated)})
	add(model.DetailItem{Name: "Description", Value: model.NestedString(o, fieldDescription)})
	if s := model.NestedString(o, "secret"); s != "" {
		add(model.DetailItem{Name: "Release Secret", Value: s,
			Link: &model.ObjectRef{Kind: "secrets", Namespace: o.GetNamespace(), Name: s}})
	}
	add(model.DetailItem{Name: "Error", Value: model.NestedString(o, "error"), Status: model.StatusError})
	sections := []model.DetailSection{{Title: model.TitleProperties, Items: props}}

	table := &model.DetailTable{Columns: []string{titleRevision, titleUpdated, titleStatus, titleChart, titleAppVersion, "Description"}}
	for _, h := range model.NestedSlice(o, fieldHistory) {
		n, _ := model.NestedInt64(h, fieldRevision)
		table.Rows = append(table.Rows, []string{
			strconv.FormatInt(n, 10), model.NestedString(h, fieldUpdated), model.NestedString(h, fieldStatus),
			model.NestedString(h, fieldChart), model.NestedString(h, fieldAppVersion), model.NestedString(h, fieldDescription),
		})
	}
	if len(table.Rows) > 0 {
		sections = append(sections, model.DetailSection{Title: TitleHistory, Table: table})
	}

	values, manifest := "", ""
	if h := m.heavyOf(o); h != nil {
		values, manifest = h.values, h.manifest
		if values == "" {
			values = "No user-supplied values"
		}
	} else {
		values, manifest = "Not available: the release record could not be read", "Not available"
	}
	return append(sections,
		model.DetailSection{Title: TitleValues, Items: []model.DetailItem{{Value: values, Private: true}}},
		model.DetailSection{Title: TitleManifest, Items: []model.DetailItem{{Value: manifest}}},
	)
}

// heavyOf returns the values and manifest the memo keeps for a release object's newest revision,
// or nil.
func (m *memo) heavyOf(o *unstructured.Unstructured) *heavy {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r := m.get(o.GetUID(), o.GetResourceVersion()); r != nil && r.heavy != nil {
		h := *r.heavy
		return &h
	}
	return nil
}
