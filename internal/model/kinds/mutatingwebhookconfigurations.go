package kinds

import "github.com/guided-traffic/cogpit/internal/model"

// mutatingWebhookConfigurations has the columns of Lens's documentation (not captured) and the
// title of the captured sidebar ("Configurations"); the drawer is inferred (admission.go).
var mutatingWebhookConfigurations = &model.KindProfile{
	Key:      "mutatingwebhookconfigurations",
	Group:    groupAdmission,
	Version:  "v1",
	Resource: "mutatingwebhookconfigurations",
	Kind:     "MutatingWebhookConfiguration",
	Title:    "Mutating Webhook Configurations",
	Columns:  webhooksColumns,
	Cells:    webhookConfigurationCells,
	Fields:   webhookFields,
	Detail:   webhookConfigurationDetail,
}
