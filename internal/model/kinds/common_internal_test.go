package kinds

import (
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

func TestWarningCell(t *testing.T) {
	if c := warningCell(model.StatusNone, "ignored"); c != (model.Cell{}) {
		t.Errorf("no warning: got %+v, want an empty cell", c)
	}
	c := warningCell(model.StatusWarning, "3 restarts")
	if c.Text != "3 restarts" || c.Num != 1 || c.Status != model.StatusWarning || c.Link != nil {
		t.Errorf("warning: got %+v", c)
	}
	if warningColumn.Type != model.ColumnWarning || warningColumn.ID != "warning" || warningColumn.Title == "" {
		t.Errorf("warningColumn = %+v", warningColumn)
	}
	if got := model.CellText(warningColumn, c, time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)); got != "3 restarts" {
		t.Errorf("CellText = %q, want the reason", got)
	}
}

func TestNamespaceCellLinks(t *testing.T) {
	obj := map[string]any{"metadata": map[string]any{"name": "web", "namespace": "shop"}}
	c := namespaceCell(obj)
	want := model.ObjectRef{Kind: "namespaces", Name: "shop"}
	if c.Text != "shop" || c.Link == nil || *c.Link != want {
		t.Errorf("namespaceCell = %+v, link %v; want text shop and link %+v", c, c.Link, want)
	}
	if c := namespaceCell(map[string]any{}); c.Link != nil || c.Text != "" {
		t.Errorf("cluster-scoped object: got %+v, want no text and no link", c)
	}
	if l := linkTo("nodes", "", ""); l != nil {
		t.Errorf("linkTo without a name = %+v, want nil", l)
	}
}
