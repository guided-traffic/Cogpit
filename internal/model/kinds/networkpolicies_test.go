package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzNetworkPolicies(f *testing.F) { fuzzProfile(f, "networkpolicies") }

func TestNetworkPolicyTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"networkpolicies", "healthy", "policyTypes", "Ingress, Egress", model.StatusNone, model.StatusNone},
		{"networkpolicies", "failing", "policyTypes", "Ingress, Egress", model.StatusNone, model.StatusNone},
		{"networkpolicies", "terminating", "policyTypes", "Ingress", model.StatusNone, model.StatusTerminating},
		// Without policyTypes the API server defaults to Ingress, plus Egress with egress rules.
		{"networkpolicies", "minimal", "policyTypes", "Ingress", model.StatusNone, model.StatusNone},
		{"networkpolicies", "egress-default", "policyTypes", "Ingress, Egress", model.StatusNone, model.StatusNone},
	})
	assertColumnOrder(t, "networkpolicies", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "policyTypes", Title: "Policy Types"},
		{ID: "age", Title: "Age"},
	})
}

// TestNetworkPolicyRules pins how the drawer reads rules: a direction without rules denies
// everything, an empty rule allows everything, peers and ports in words.
func TestNetworkPolicyRules(t *testing.T) {
	p, _ := kinds.ByKey("networkpolicies")
	detail := map[string][]model.DetailSection{}
	for _, c := range loadCases(t, "networkpolicies") {
		detail[c.name] = p.Detail(c.obj)
	}
	section := func(name, title string) []model.DetailItem {
		for _, s := range detail[name] {
			if s.Title == title {
				return s.Items
			}
		}
		t.Fatalf("%s: no section %s", name, title)
		return nil
	}
	if deny := section("failing", "Egress"); len(deny) != 1 || deny[0].Status != model.StatusWarning ||
		deny[0].Value != "none: all outgoing traffic of the selected pods is denied" {
		t.Errorf("deny-all egress %+v", deny)
	}
	for _, tt := range []struct{ name, title, rule, want string }{
		{"healthy", "Ingress", "Rule 1", "From: all pods in namespaces kubernetes.io/metadata.name=ingress-nginx; pods role=frontend in this namespace\nPorts: 8080/TCP"},
		{"healthy", "Ingress", "Rule 2", "From: 10.0.0.0/8 except 10.1.0.0/16\nPorts: all"},
		{"healthy", "Egress", "Rule 1", "To: pods app=api in namespaces team in (shop,payments)\nPorts: http/TCP"},
		{"terminating", "Ingress", "Rule 1", "From: pods app=recorder in all namespaces\nPorts: 32000-32768/UDP"},
		{"terminating", "Ingress", "Rule 2", "From: anywhere\nPorts: all"},
	} {
		got := ""
		for _, it := range section(tt.name, tt.title) {
			if it.Name == tt.rule {
				got = it.Value
			}
		}
		if got != tt.want {
			t.Errorf("%s %s %s = %q, want %q", tt.name, tt.title, tt.rule, got, tt.want)
		}
	}
	if pods := findItem(detail["failing"], "Pod Selector"); pods.Value != "every pod in the namespace" {
		t.Errorf("empty pod selector reads %q", pods.Value)
	}
}
