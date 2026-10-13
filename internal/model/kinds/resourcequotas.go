package kinds

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

const (
	// itemAtLimit names the drawer item that lists a quota's resources at their hard limit.
	itemAtLimit = "At Limit"
	// idResource is the searchable field of the resources a quota limits.
	idResource = "resource"
)

// resourceQuotas has the columns of Lens's documentation (not captured): Name, Namespace, Age.
// Inferred: the status rule (a quota whose use reached a hard limit is a warning, because the API
// server refuses what would go beyond it; pending until the quota controller has counted the use)
// and the drawer (the scopes, then a table of each resource's use against its hard limit).
var resourceQuotas = &model.KindProfile{
	Key:        "resourcequotas",
	Version:    "v1",
	Resource:   "resourcequotas",
	Kind:       "ResourceQuota",
	Title:      "Resource Quotas",
	Namespaced: true,
	Columns:    []model.Column{nameColumn, namespaceColumn, ageColumn},
	Cells:      resourceQuotaCells,
	Fields: map[string]fieldFunc{
		// "resource" lists the resources the quota limits: "cpu", "requests.memory", "pods".
		idResource: func(obj *unstructured.Unstructured) []string {
			return sortedKeys(quotaHard(obj))
		},
		"scope": func(obj *unstructured.Unstructured) []string {
			return model.NestedStrings(obj, "spec", "scopes")
		},
		"atLimit": func(obj *unstructured.Unstructured) []string {
			return quotaAtLimit(obj)
		},
	},
	Detail: resourceQuotaDetail,
}

func resourceQuotaCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		model.AgeCell(created(obj)),
	}, resourceQuotaStatus(obj)
}

// resourceQuotaStatus colours a quota: uncoloured without hard limits, pending until the quota
// controller has written the use, a warning when a resource is at or over its limit.
func resourceQuotaStatus(obj any) model.Status {
	switch {
	case len(quotaHard(obj)) == 0:
		return model.StatusNone
	case model.NestedMap(obj, "status", "used") == nil:
		return model.StatusPending
	case len(quotaAtLimit(obj)) > 0:
		return model.StatusWarning
	}
	return model.StatusOK
}

// quotaHard is the quota's hard limits as the controller enforces them (status.hard), else as
// specified (spec.hard) before the controller has seen the quota.
func quotaHard(obj any) map[string]any {
	if hard := model.NestedMap(obj, "status", "hard"); len(hard) > 0 {
		return hard
	}
	return model.NestedMap(obj, "spec", "hard")
}

// quotaAtLimit returns the resources whose use reached their hard limit, sorted: used above the
// limit (a limit lowered below the use), or equal to a limit above zero. A limit of zero that is
// not used forbids the resource, which is the quota working, not running out.
func quotaAtLimit(obj any) []string {
	hard := quotaHard(obj)
	used := model.NestedMap(obj, "status", "used")
	var out []string
	for _, name := range sortedKeys(hard) {
		h, okHard := parseQuantity(model.NestedIntOrString(hard, name))
		u, okUsed := parseQuantity(model.NestedIntOrString(used, name))
		if !okHard || !okUsed {
			continue
		}
		if c := u.Cmp(h); c > 0 || (c == 0 && !h.IsZero()) {
			out = append(out, name)
		}
	}
	return out
}

// parseQuantity parses a resource quantity with quantityValue's guard: it refuses input longer
// than any quantity and decimal exponents beyond ±30, which resource.ParseQuantity spends
// unbounded time on.
func parseQuantity(s string) (resource.Quantity, bool) {
	if s == "" || len(s) > 64 {
		return resource.Quantity{}, false
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		if rest := s[i+1:]; rest != "" && rest != "i" { // "E" and "Ei" are suffixes
			exp, err := strconv.Atoi(rest)
			if err != nil || exp < -30 || exp > 30 {
				return resource.Quantity{}, false
			}
		}
	}
	q, err := resource.ParseQuantity(s)
	return q, err == nil
}

func resourceQuotaDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.add("Scopes", strings.Join(model.NestedStrings(obj, "spec", "scopes"), ", "))
	d.add("Scope Selector", strings.Join(scopeSelector(obj), "\n"))
	d.addStatus(itemAtLimit, strings.Join(quotaAtLimit(obj), ", "), model.StatusWarning)
	sections := withSection(nil, model.TitleProperties, d)

	hard := quotaHard(obj)
	if len(hard) == 0 {
		return sections
	}
	used := model.NestedMap(obj, "status", "used")
	t := &model.DetailTable{Columns: []string{titleResource, "Used", "Hard"}}
	for _, name := range sortedKeys(hard) {
		u := model.FormatResource(name, model.NestedIntOrString(used, name)) // bytes in binary units (D-32)
		if u == "" {
			u = "-"
		}
		t.Rows = append(t.Rows, []string{name, u, model.FormatResource(name, model.NestedIntOrString(hard, name))})
	}
	return append(sections, model.DetailSection{Title: "Quotas", Table: t})
}

// scopeSelector renders spec.scopeSelector's expressions: "PriorityClass In (high, critical)",
// "Terminating Exists".
func scopeSelector(obj any) []string {
	var out []string
	for _, e := range model.NestedSlice(obj, "spec", "scopeSelector", "matchExpressions") {
		scope, op := model.NestedString(e, "scopeName"), model.NestedString(e, "operator")
		if scope == "" {
			continue
		}
		s := scope + " " + op
		if values := model.NestedStrings(e, "values"); len(values) > 0 {
			s += " (" + strings.Join(values, ", ") + ")"
		}
		out = append(out, s)
	}
	return out
}
