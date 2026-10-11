package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzMutatingAdmissionPolicyBindings(f *testing.F) {
	fuzzProfile(f, "mutatingadmissionpolicybindings")
}

func TestMutatingAdmissionPolicyBindingsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	checkStatusTexts(t, []statusText{
		{"mutatingadmissionpolicybindings", "healthy", "policy", "inject-log-shipper", none, none},
		{"mutatingadmissionpolicybindings", "failing", "policy", "default-requests", none, none},
		{"mutatingadmissionpolicybindings", "minimal", "policy", "", none, none},
		{"mutatingadmissionpolicybindings", "terminating", "policy", "legacy-defaults", none, model.StatusTerminating},
	})
}
