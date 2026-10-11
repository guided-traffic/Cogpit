package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzMutatingWebhookConfigurations(f *testing.F) {
	fuzzProfile(f, "mutatingwebhookconfigurations")
}

func TestMutatingWebhookConfigurationsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// A configuration has no state: no row is coloured (inferred).
	checkStatusTexts(t, []statusText{
		{"mutatingwebhookconfigurations", "healthy", "webhooks", "1", none, none},
		{"mutatingwebhookconfigurations", "failing", "webhooks", "1", none, none},
		{"mutatingwebhookconfigurations", "minimal", "webhooks", "0", none, none},
		{"mutatingwebhookconfigurations", "terminating", "webhooks", "1", none, model.StatusTerminating},
	})
}
