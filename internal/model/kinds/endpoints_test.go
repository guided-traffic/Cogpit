package kinds_test

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzEndpoints(f *testing.F) { fuzzProfile(f, "endpoints") }

func TestEndpointsTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"endpoints", "healthy", "endpoints", "10.244.1.5:8080, 10.244.1.5:9102, 10.244.2.7:8080 + 1 more", model.StatusNone, model.StatusNone},
		// Only ready addresses count, as kubectl prints them.
		{"endpoints", "failing", "endpoints", "-", model.StatusNone, model.StatusNone},
		{"endpoints", "subsets", "endpoints", "10.244.5.20:9200, 10.244.6.30:9201, 10.244.6.31", model.StatusNone, model.StatusNone},
		{"endpoints", "terminating", "endpoints", "192.0.2.40:5432", model.StatusNone, model.StatusTerminating},
		{"endpoints", "minimal", "endpoints", "-", model.StatusNone, model.StatusNone},
	})
	assertColumnOrder(t, "endpoints", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "endpoints", Title: "Endpoints"},
		{ID: "age", Title: "Age"},
	})
}

// TestEndpointsDrawerCapped: the addresses table shows at most 100 rows and says how many more
// there are.
func TestEndpointsDrawerCapped(t *testing.T) {
	addrs := make([]any, 0, 250)
	for i := range 250 {
		addrs = append(addrs, map[string]any{"ip": "10.0.0." + strings.Repeat("1", 1+i%3)})
	}
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{"name": "big", "namespace": "n"},
		"subsets":  []any{map[string]any{"addresses": addrs}},
	}}
	p, _ := kinds.ByKey("endpoints")
	tbl := findTable(t, p.Detail(obj), "Addresses")
	if len(tbl.Rows) != 101 || tbl.Rows[100][0] != "… 150 more" {
		t.Errorf("%d rows, last %q", len(tbl.Rows), tbl.Rows[len(tbl.Rows)-1])
	}
}
