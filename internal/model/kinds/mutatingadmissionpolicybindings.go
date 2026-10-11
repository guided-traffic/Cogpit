package kinds

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// mutatingAdmissionPolicyBindings has the columns of Lens's documentation (not captured): Name,
// Policy, Age, at admissionregistration.k8s.io/v1 only, as the policies it binds. Inferred: Policy
// as a link to the policy, no status token, and the drawer.
var mutatingAdmissionPolicyBindings = &model.KindProfile{
	Key:      "mutatingadmissionpolicybindings",
	Group:    groupAdmission,
	Version:  "v1",
	Resource: "mutatingadmissionpolicybindings",
	Kind:     "MutatingAdmissionPolicyBinding",
	Title:    "Mutating Admission Policy Bindings",
	Columns:  []model.Column{nameColumn, policyColumn, ageColumn},
	Cells:    mutatingBindingCells,
	Fields:   bindingFields,
	Detail:   mutatingBindingDetail,
}

func mutatingBindingCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		policyCell(obj, keyMutatingAdmissionPolicies),
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

func mutatingBindingDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	policy := policyCell(obj, keyMutatingAdmissionPolicies)
	d.put(linkedItem("Policy", policy.Text, policy.Link))
	d.addBinding(obj)
	return withSection(nil, model.TitleProperties, d)
}
