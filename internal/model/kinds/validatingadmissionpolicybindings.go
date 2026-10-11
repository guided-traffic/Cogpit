package kinds

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// validatingAdmissionPolicyBindings has the columns of Lens's documentation (not captured): Name,
// Policy, Validation Actions, Age, at the versions of the policies it binds. Inferred: Policy as a
// link to the policy, Validation Actions comma-separated ("Deny, Audit"), no status token, and the
// drawer.
var validatingAdmissionPolicyBindings = &model.KindProfile{
	Key:              "validatingadmissionpolicybindings",
	Group:            groupAdmission,
	Version:          "v1",
	FallbackVersions: admissionPolicyVersions,
	Resource:         "validatingadmissionpolicybindings",
	Kind:             "ValidatingAdmissionPolicyBinding",
	Title:            "Validating Admission Policy Bindings",
	Columns: []model.Column{
		nameColumn,
		policyColumn,
		{ID: "validationActions", Title: "Validation Actions", Type: model.ColumnText},
		ageColumn,
	},
	Cells:  validatingBindingCells,
	Fields: bindingFields,
	Detail: validatingBindingDetail,
}

func validatingBindingCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		policyCell(obj, keyValidatingAdmissionPolicies),
		{Text: validationActions(obj)},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// validationActions is spec.validationActions, comma-separated: "Deny, Audit".
func validationActions(obj any) string {
	return strings.Join(model.NestedStrings(obj, "spec", "validationActions"), ", ")
}

func validatingBindingDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	policy := policyCell(obj, keyValidatingAdmissionPolicies)
	d.put(linkedItem("Policy", policy.Text, policy.Link))
	d.add("Validation Actions", validationActions(obj))
	d.addBinding(obj)
	return withSection(nil, model.TitleProperties, d)
}
