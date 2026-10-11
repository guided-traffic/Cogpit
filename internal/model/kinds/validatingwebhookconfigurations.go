package kinds

import "github.com/guided-traffic/cogpit/internal/model"

// validatingWebhookConfigurations has the columns of Lens's documentation (not captured) and the
// title of the captured sidebar ("Configurations"); the drawer is inferred (admission.go).
var validatingWebhookConfigurations = &model.KindProfile{
	Key:      "validatingwebhookconfigurations",
	Group:    groupAdmission,
	Version:  "v1",
	Resource: "validatingwebhookconfigurations",
	Kind:     "ValidatingWebhookConfiguration",
	Title:    "Validating Webhook Configurations",
	Columns:  webhooksColumns,
	Cells:    webhookConfigurationCells,
	Fields:   webhookFields,
	Detail:   webhookConfigurationDetail,
}
