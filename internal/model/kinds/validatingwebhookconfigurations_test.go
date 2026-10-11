package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzValidatingWebhookConfigurations(f *testing.F) {
	fuzzProfile(f, "validatingwebhookconfigurations")
}

func TestValidatingWebhookConfigurationsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// A configuration has no state: no row is coloured (inferred).
	checkStatusTexts(t, []statusText{
		{"validatingwebhookconfigurations", "healthy", "webhooks", "2", none, none},
		{"validatingwebhookconfigurations", "failing", "webhooks", "2", none, none},
		{"validatingwebhookconfigurations", "minimal", "webhooks", "0", none, none},
		{"validatingwebhookconfigurations", "terminating", "webhooks", "1", none, model.StatusTerminating},
	})
}

// TestWebhookDrawer pins the drawer of a webhook configuration: a section per webhook under the
// heading Webhooks, its service as a link, its rules as text.
func TestWebhookDrawer(t *testing.T) {
	p, _ := kinds.ByKey("validatingwebhookconfigurations")
	for _, c := range loadCases(t, "validatingwebhookconfigurations") {
		if c.name != "healthy" {
			continue
		}
		sections := p.Detail(c.obj)
		if len(sections) != 2 || sections[0].Group != "Webhooks" || sections[0].Title != "workloads.policy.example.com" {
			t.Fatalf("sections: %+v", sections)
		}
		items := map[string]model.DetailItem{}
		for _, it := range sections[0].Items {
			items[it.Name] = it
		}
		client := items["Client"]
		if client.Value != "policy-system/policy-engine:443/validate/workloads" ||
			client.Link == nil || *client.Link != (model.ObjectRef{Kind: "services", Namespace: "policy-system", Name: "policy-engine"}) {
			t.Errorf("Client: %+v", client)
		}
		if got := items["Rules"].Value; got != "CREATE, UPDATE apps/v1 deployments, statefulsets (Namespaced)" {
			t.Errorf("Rules: %q", got)
		}
		if _, shown := items["CA Bundle"]; shown {
			t.Error("the CA bundle is shown")
		}
	}
}
