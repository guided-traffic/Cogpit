package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzCustomResourceDefinitions(f *testing.F) { fuzzProfile(f, kinds.KeyCustomResourceDefinitions) }

func TestCustomResourceDefinitionTexts(t *testing.T) {
	const k = kinds.KeyCustomResourceDefinitions
	checkStatusTexts(t, []statusText{
		{k, "healthy", "resource", "Order", model.StatusNone, model.StatusOK},
		{k, "healthy", "group", "acme.cert-manager.io", model.StatusNone, model.StatusOK},
		{k, "healthy", "version", "v1", model.StatusNone, model.StatusOK},
		{k, "healthy", "scope", "Namespaced", model.StatusNone, model.StatusOK},
		{k, "healthy", "age", "40d", model.StatusNone, model.StatusOK},
		{k, "failing", "resource", "Widget", model.StatusNone, model.StatusError},
		{k, "terminating", "scope", "Cluster", model.StatusNone, model.StatusTerminating},
		{k, "minimal", "resource", "", model.StatusNone, model.StatusPending},
		// The view uses the highest served version, not the storage version.
		{k, "multi-version", "version", "v1beta1", model.StatusNone, model.StatusOK},
	})
}

// TestCustomResourceDefinitionLinks checks the Resource cell: a link to the kind's list view (a
// link without a name) once the definition is established, none before.
func TestCustomResourceDefinitionLinks(t *testing.T) {
	p, ok := kinds.ByKey(kinds.KeyCustomResourceDefinitions)
	if !ok {
		t.Fatal("no Definitions profile")
	}
	if p.Namespaced || p.Columns[0].ID != "name" || !p.Columns[0].Hidden || p.Columns[1].Title != "Resource" {
		t.Fatalf("Definitions: namespaced %v, columns %+v", p.Namespaced, p.Columns)
	}
	want := map[string]*model.ObjectRef{
		"healthy":       {Kind: "orders.acme.cert-manager.io"},
		"multi-version": {Kind: "backups.example.io"},
		"terminating":   {Kind: "snapshots.backup.example.io"},
		"failing":       nil,
		"minimal":       nil,
	}
	resource := p.ColumnIndex("resource")
	for _, c := range loadCases(t, kinds.KeyCustomResourceDefinitions) {
		link := model.BuildRow(p, "test", c.obj).Cells[resource].Link
		w, listed := want[c.name]
		if !listed {
			continue
		}
		if (link == nil) != (w == nil) || (link != nil && *link != *w) {
			t.Errorf("%s: link %+v, want %+v", c.name, link, w)
		}
		if link != nil && link.Name != "" {
			t.Errorf("%s: the link names an object (%q), want the kind's list view", c.name, link.Name)
		}
	}
}
