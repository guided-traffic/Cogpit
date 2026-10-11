package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzEndpointSlices(f *testing.F) { fuzzProfile(f, "endpointslices") }

func TestEndpointSliceTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"endpointslices", "healthy", "addressType", "IPv4", model.StatusNone, model.StatusNone},
		{"endpointslices", "healthy", "endpoints", "10.244.1.5, 10.244.2.7, 10.244.3.9", model.StatusNone, model.StatusNone},
		{"endpointslices", "healthy", "ports", "8080/TCP", model.StatusNone, model.StatusNone},
		// Endpoints that are not ready are listed too; the drawer colours them.
		{"endpointslices", "failing", "endpoints", "10.244.4.11, 10.244.4.12, 10.244.4.13", model.StatusNone, model.StatusNone},
		{"endpointslices", "failing", "ports", "9090/TCP, 9102/TCP", model.StatusNone, model.StatusNone},
		{"endpointslices", "headless-ipv6", "endpoints", "fd00:10:244::a, fd00:10:244::b, fd00:10:244::c + 2 more", model.StatusNone, model.StatusNone},
		{"endpointslices", "headless-ipv6", "ports", "*/TCP", model.StatusNone, model.StatusNone},
		{"endpointslices", "terminating", "endpoints", "192.0.2.40", model.StatusNone, model.StatusTerminating},
		{"endpointslices", "minimal", "endpoints", "-", model.StatusNone, model.StatusNone},
		{"endpointslices", "minimal", "ports", "-", model.StatusNone, model.StatusNone},
	})
	assertColumnOrder(t, "endpointslices", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "addressType", Title: "Address Type"},
		{ID: "endpoints", Title: "Endpoints"},
		{ID: "ports", Title: "Ports"},
		{ID: "age", Title: "Age"},
	})
}

// TestEndpointSliceDrawer: the slice links to its Service, and its endpoints table colours each
// endpoint by its conditions and links its node and target.
func TestEndpointSliceDrawer(t *testing.T) {
	p, _ := kinds.ByKey("endpointslices")
	cases := map[string][]model.DetailSection{}
	for _, c := range loadCases(t, "endpointslices") {
		cases[c.name] = p.Detail(c.obj)
	}
	healthy := cases["healthy"]
	if it := findItem(healthy, "Service"); it.Link == nil || *it.Link != (model.ObjectRef{Kind: "services", Namespace: "shop", Name: "web"}) {
		t.Errorf("service item %+v", it)
	}
	tbl := findTable(t, cases["failing"], "Endpoints")
	want := []struct {
		text   string
		status model.Status
	}{{"Not ready", model.StatusWarning}, {"Serving, Terminating", model.StatusTerminating}, {"Not ready, Serving", model.StatusWarning}}
	for i, w := range want {
		if c := tbl.Cells[i][1]; c.Text != w.text || c.Status != w.status {
			t.Errorf("endpoint %d conditions %+v, want %+v", i, c, w)
		}
	}
	if l := tbl.Cells[0][5].Link; l == nil || *l != (model.ObjectRef{Kind: "pods", Namespace: "payments", Name: "payments-api-6f8b9c7d4-q9m2z"}) {
		t.Errorf("target link %+v", l)
	}
	if l := tbl.Cells[0][3].Link; l == nil || *l != (model.ObjectRef{Kind: "nodes", Name: "worker-b-02"}) {
		t.Errorf("node link %+v", l)
	}
	// Unset conditions read as ready, as consumers are told to read them.
	if c := findTable(t, cases["headless-ipv6"], "Endpoints").Cells[2][1]; c.Text != "Ready" {
		t.Errorf("unset conditions read %q", c.Text)
	}
}

func findItem(sections []model.DetailSection, name string) model.DetailItem {
	for _, s := range sections {
		for _, it := range s.Items {
			if it.Name == name {
				return it
			}
		}
	}
	return model.DetailItem{}
}

func findTable(t *testing.T, sections []model.DetailSection, title string) *model.DetailTable {
	t.Helper()
	for _, s := range sections {
		if s.Title == title && s.Table != nil {
			return s.Table
		}
	}
	t.Fatalf("no table %q in %+v", title, sections)
	return nil
}
