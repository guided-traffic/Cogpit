package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzIngresses(f *testing.F) { fuzzProfile(f, "ingresses") }

// Ingresses have no status token (inferred: Lens shows none, and an Ingress without an address is
// not necessarily broken); only a deletion makes the row terminating.
func TestIngressTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"ingresses", "healthy", "loadBalancers", "203.0.113.30", model.StatusNone, model.StatusNone},
		{"ingresses", "healthy", "rules", "shop.example.com/api -> api:8080, shop.example.com/ -> web:80, www.shop.example.com/ -> web:http, (default) -> web:http", model.StatusNone, model.StatusNone},
		{"ingresses", "failing", "loadBalancers", "-", model.StatusNone, model.StatusNone},
		{"ingresses", "failing", "rules", "admin.example.internal/ -> admin-ui:3000", model.StatusNone, model.StatusNone},
		{"ingresses", "terminating", "loadBalancers", "legacy-lb.example.net", model.StatusNone, model.StatusTerminating},
		{"ingresses", "minimal", "rules", "", model.StatusNone, model.StatusNone},
		{"ingresses", "minimal", "loadBalancers", "-", model.StatusNone, model.StatusNone},
		{"ingresses", "default-backend", "rules", "(default) -> StorageBucket/static-assets", model.StatusNone, model.StatusNone},
		{"ingresses", "hostname-lb", "rules", "*.tenants.example.com -> tenant-router:8443, */healthz -> health:8080, parked.example.com", model.StatusNone, model.StatusNone},
		{"ingresses", "hostname-lb", "loadBalancers", "k8s-saas-tenants-0123456789.elb.example.net", model.StatusNone, model.StatusNone},
	})
}

// TestIngressColumns pins Lens's documented columns (not captured) in their order.
func TestIngressColumns(t *testing.T) {
	assertColumnOrder(t, "ingresses", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "loadBalancers", Title: "Load Balancers"},
		{ID: "rules", Title: "Rules"},
		{ID: "age", Title: "Age"},
	})
}
