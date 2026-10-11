package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzValidatingAdmissionPolicyBindings(f *testing.F) {
	fuzzProfile(f, "validatingadmissionpolicybindings")
}

func TestValidatingAdmissionPolicyBindingsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	checkStatusTexts(t, []statusText{
		{"validatingadmissionpolicybindings", "healthy", "policy", "replica-limit", none, none},
		{"validatingadmissionpolicybindings", "healthy", "validationActions", "Deny, Audit", none, none},
		{"validatingadmissionpolicybindings", "failing", "validationActions", "Warn", none, none},
		{"validatingadmissionpolicybindings", "minimal", "policy", "", none, none},
		{"validatingadmissionpolicybindings", "terminating", "policy", "legacy-image-registry", none, model.StatusTerminating},
	})
}

// TestBindingPolicyLinks checks that both binding kinds link their Policy cell to the policy of
// their own kind, and leave it plain when the binding names none.
func TestBindingPolicyLinks(t *testing.T) {
	for binding, policy := range map[string]string{
		"validatingadmissionpolicybindings": "validatingadmissionpolicies",
		"mutatingadmissionpolicybindings":   "mutatingadmissionpolicies",
	} {
		p, _ := kinds.ByKey(binding)
		i := p.ColumnIndex("policy")
		for _, c := range loadCases(t, binding) {
			cell := model.BuildRow(p, "test", c.obj).Cells[i]
			switch {
			case cell.Text == "" && cell.Link != nil:
				t.Errorf("%s/%s: a link without a policy", binding, c.name)
			case cell.Text != "" && (cell.Link == nil || *cell.Link != model.ObjectRef{Kind: policy, Name: cell.Text}):
				t.Errorf("%s/%s: link %+v, want %s/%s", binding, c.name, cell.Link, policy, cell.Text)
			}
		}
	}
}
