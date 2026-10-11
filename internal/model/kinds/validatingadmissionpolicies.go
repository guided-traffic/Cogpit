package kinds

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// keyValidatingAdmissionPolicies is the kind key a validating binding's Policy cell links to.
const keyValidatingAdmissionPolicies = "validatingadmissionpolicies"

// admissionPolicyVersions are the versions older clusters serve the validating admission policies
// and their bindings at, preferred first: v1 from Kubernetes 1.30, v1beta1 from 1.28 (off by
// default), v1alpha1 from 1.26 (off by default).
var admissionPolicyVersions = []string{"v1beta1", "v1alpha1"}

// validatingAdmissionPolicies has the columns of Lens's documentation (not captured): Name,
// Validations (how many the policy has), Failure Policy, Age, at admissionregistration.k8s.io/v1,
// falling back to v1beta1 and v1alpha1. Inferred: the status rule (a warning when type checking
// found problems in the policy's expressions, else uncoloured) and the drawer.
var validatingAdmissionPolicies = &model.KindProfile{
	Key:              keyValidatingAdmissionPolicies,
	Group:            groupAdmission,
	Version:          "v1",
	FallbackVersions: admissionPolicyVersions,
	Resource:         "validatingadmissionpolicies",
	Kind:             "ValidatingAdmissionPolicy",
	Title:            "Validating Admission Policies",
	Columns: []model.Column{
		nameColumn,
		{ID: "validations", Title: "Validations", Type: model.ColumnNumber},
		{ID: "failurePolicy", Title: itemFailurePolicy, Type: model.ColumnText},
		ageColumn,
	},
	Cells: validatingPolicyCells,
	Fields: map[string]fieldFunc{
		"failurePolicy": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "spec", "failurePolicy"))
		},
	},
	Detail: validatingPolicyDetail,
}

func validatingPolicyCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	status := model.StatusNone
	if len(typeCheckWarnings(obj)) > 0 {
		status = model.StatusWarning
	}
	return []model.Cell{
		nameCell(obj),
		{Num: int64(len(model.NestedSlice(obj, "spec", "validations")))},
		{Text: model.NestedString(obj, "spec", "failurePolicy")},
		model.AgeCell(created(obj)),
	}, status
}

// typeCheckWarnings returns what type checking found in the policy's expressions, each as
// "field: warning".
func typeCheckWarnings(obj any) []string {
	var out []string
	for _, w := range model.NestedSlice(obj, "status", "typeChecking", "expressionWarnings") {
		text := singleLine(model.NestedString(w, "warning"))
		if field := model.NestedString(w, "fieldRef"); field != "" {
			text = field + ": " + text
		}
		out = append(out, text)
	}
	return out
}

func validatingPolicyDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.addStatus("Type Checking", strings.Join(typeCheckWarnings(obj), "\n"), model.StatusWarning)
	d.add(itemFailurePolicy, model.NestedString(obj, "spec", "failurePolicy"))
	d.addPolicySpec(obj)
	sections := withSection(nil, model.TitleProperties, d)

	for i, v := range model.NestedSlice(obj, "spec", "validations") {
		var items detailItems
		items.add("Expression", model.NestedString(v, "expression"))
		items.add("Message", model.NestedString(v, "message"))
		items.add("Message Expression", model.NestedString(v, "messageExpression"))
		items.add("Reason", model.NestedString(v, "reason"))
		sections = append(sections, model.DetailSection{
			Title: "Validation " + strconv.Itoa(i+1), Group: "Validations", Items: items,
		})
	}

	var audit detailItems
	for _, a := range model.NestedSlice(obj, "spec", "auditAnnotations") {
		audit.add(model.NestedString(a, "key"), model.NestedString(a, "valueExpression"))
	}
	return withSection(sections, "Audit Annotations", audit)
}
