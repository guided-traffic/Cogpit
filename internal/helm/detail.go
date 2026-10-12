package helm

import (
	"encoding/json"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/duration"

	"github.com/guided-traffic/cogpit/internal/model"
)

// UpgradeTabCommand is the command that opens a release's Helm Upgrade dock tab: Lens's Upgrade,
// the icon in the drawer's title bar and the button beside the drawer's Chart (captured
// 2026-10-12).
const UpgradeTabCommand = "helm.release.upgradeTab"

// The drawer's clock and zone: Updated reads "68s ago (12 Oct 2026, 8:30:16 CEST)", in the local
// zone, as Lens writes it (captured 2026-10-12). Vars for tests.
var (
	clock = time.Now
	zone  = time.Local
)

// detail is the profile's Detail: Lens's release drawer (captured 2026-10-12). Properties holds
// Chart (with an Upgrade button), Updated, Namespace, Version (the chart's version, inferred: the
// list's Version column) and Status (a pill, written as Lens writes it); then the newest revision's
// user-supplied values, which the GUI shows in an editor with Save and "User-supplied values
// only"; the chart's notes; and the release's resources, one table per kind. The values and the
// notes are private: agents read model.Withheld in their place (object_get), because values can
// hold credentials and notes are rendered from them. The engine adds no generic metadata and no
// events to a virtual kind's drawer (architecture §4.4); the title is "Release: <name>".
func (m *memo) detail(o *unstructured.Unstructured) []model.DetailSection {
	status := model.NestedString(o, fieldStatus)
	ns := o.GetNamespace()
	chart, version := ReleaseChart(o)
	var props []model.DetailItem
	add := func(it model.DetailItem) {
		if it.Value != "" {
			props = append(props, it)
		}
	}
	chartItem := model.DetailItem{Name: titleChart, Value: chart}
	if params, err := json.Marshal(map[string]any{"ref": model.ObjectRef{Kind: ReleaseKind, Namespace: ns, Name: o.GetName()}}); err == nil {
		chartItem.Actions = []model.ItemAction{{Label: "Upgrade", Command: UpgradeTabCommand, Params: params}}
	}
	add(chartItem)
	if t := model.NestedTime(o, fieldUpdated); !t.IsZero() {
		add(model.DetailItem{Name: titleUpdated, Value: UpdatedText(t, clock(), zone)})
	}
	add(model.DetailItem{Name: titleNamespace, Value: ns, Link: &model.ObjectRef{Kind: kindNamespaces, Name: ns}})
	add(model.DetailItem{Name: "Version", Value: version})
	add(model.DetailItem{Name: titleStatus, Value: StatusText(status), Status: StatusToken(status)})
	add(model.DetailItem{Name: "Error", Value: model.NestedString(o, "error"), Status: model.StatusError})
	sections := []model.DetailSection{{Title: model.TitleProperties, Items: props}}

	h := m.heavyOf(o)
	values := "Not available: the release record could not be read"
	if h != nil {
		values = h.values
		if values == "" {
			values = "No user-supplied values"
		}
	}
	sections = append(sections, model.DetailSection{Title: TitleValues, Items: []model.DetailItem{{Value: values, Private: true}}})
	if h == nil {
		return sections
	}
	if h.notes != "" {
		sections = append(sections, model.DetailSection{Title: TitleNotes, Items: []model.DetailItem{{Value: h.notes, Private: true}}})
	}
	return append(sections, resourceSections(h.resources)...)
}

// heavyOf returns the values, notes and resources the memo keeps for a release object's newest
// revision, or nil.
func (m *memo) heavyOf(o *unstructured.Unstructured) *heavy {
	m.mu.Lock()
	defer m.mu.Unlock()
	if r := m.get(o.GetUID(), o.GetResourceVersion()); r != nil && r.heavy != nil {
		h := *r.heavy
		return &h
	}
	return nil
}

// UpdatedText is a release's Updated as Lens's drawer writes it: the age, then the time in loc,
// "68s ago (12 Oct 2026, 8:30:16 CEST)" (captured 2026-10-12). The age is kubectl's human
// duration, which writes 68 seconds as "68s" as Lens does (the other lengths inferred).
func UpdatedText(t, now time.Time, loc *time.Location) string {
	l := t.In(loc)
	return fmt.Sprintf("%s ago (%d %s %d, %d:%02d:%02d %s)", duration.HumanDuration(now.Sub(t)),
		l.Day(), l.Month().String()[:3], l.Year(), l.Hour(), l.Minute(), l.Second(), l.Format("MST"))
}

// RevisionText is a revision as Lens's Rollback dialog lists it (captured 2026-10-12):
// "12 - lens-history-test-0.1.0 - 1.0, updated: 12/10/2026, 08:30:16", the time in loc.
func RevisionText(e HistoryEntry, loc *time.Location) string {
	updated := ""
	if !e.Updated.IsZero() {
		updated = e.Updated.In(loc).Format("02/01/2006, 15:04:05")
	}
	return fmt.Sprintf("%d - %s - %s, updated: %s", e.Revision, chartText(e.Chart, e.ChartVersion), e.AppVersion, updated)
}
