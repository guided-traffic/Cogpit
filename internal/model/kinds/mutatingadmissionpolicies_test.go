package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzMutatingAdmissionPolicies(f *testing.F) { fuzzProfile(f, "mutatingadmissionpolicies") }

func TestMutatingAdmissionPoliciesStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// The kind has no status: no row is coloured (inferred).
	checkStatusTexts(t, []statusText{
		{"mutatingadmissionpolicies", "healthy", "mutations", "2", none, none},
		{"mutatingadmissionpolicies", "healthy", "reinvocationPolicy", "IfNeeded", none, none},
		{"mutatingadmissionpolicies", "healthy", "failurePolicy", "Fail", none, none},
		{"mutatingadmissionpolicies", "failing", "reinvocationPolicy", "Never", none, none},
		{"mutatingadmissionpolicies", "minimal", "mutations", "0", none, none},
		{"mutatingadmissionpolicies", "terminating", "failurePolicy", "Ignore", none, model.StatusTerminating},
	})
}
