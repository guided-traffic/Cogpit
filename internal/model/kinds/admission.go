package kinds

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// What the admission webhook configurations and the admission policies share: their API group,
// how a rule and a match condition read in the drawer, and the items of the matching resources.

// groupAdmission is the API group of the webhook configurations and the admission policies.
const groupAdmission = "admissionregistration.k8s.io"

// Drawer item names the admission kinds share.
const (
	itemFailurePolicy      = "Failure Policy"
	itemMatchConditions    = "Match Conditions"
	itemMatchPolicy        = "Match Policy"
	itemReinvocationPolicy = "Reinvocation Policy"
)

// admissionRuleText renders a rule with operations: "CREATE, UPDATE apps/v1 deployments,
// statefulsets (Namespaced)", the core group as "core", a rule's resource names after a colon;
// "" for a rule that names nothing.
func admissionRuleText(rule any) string {
	var parts []string
	if ops := model.NestedStrings(rule, "operations"); len(ops) > 0 {
		parts = append(parts, strings.Join(ops, ", "))
	}
	groups := model.NestedStrings(rule, "apiGroups")
	for i, g := range groups {
		if g == "" {
			groups[i] = "core"
		}
	}
	versions := model.NestedStrings(rule, "apiVersions")
	if len(groups) > 0 || len(versions) > 0 {
		parts = append(parts, strings.Join(groups, ",")+"/"+strings.Join(versions, ","))
	}
	if resources := model.NestedStrings(rule, "resources"); len(resources) > 0 {
		parts = append(parts, strings.Join(resources, ", "))
	}
	if names := model.NestedStrings(rule, "resourceNames"); len(names) > 0 {
		parts = append(parts, "named "+strings.Join(names, ", "))
	}
	if len(parts) == 0 {
		return ""
	}
	text := strings.Join(parts, " ")
	if scope := model.NestedString(rule, "scope"); scope != "" && scope != "*" {
		text += " (" + scope + ")"
	}
	return text
}

// admissionRules renders a list of rules, one per line.
func admissionRules(root any, fields ...string) string {
	var lines []string
	for _, r := range model.NestedSlice(root, fields...) {
		if text := admissionRuleText(r); text != "" {
			lines = append(lines, text)
		}
	}
	return strings.Join(lines, "\n")
}

// matchConditions renders matchConditions as "name: expression", one per line.
func matchConditions(root any, fields ...string) string {
	var lines []string
	for _, c := range model.NestedSlice(root, fields...) {
		name, expr := model.NestedString(c, "name"), model.NestedString(c, "expression")
		if name == "" && expr == "" {
			continue
		}
		lines = append(lines, name+": "+singleLine(expr))
	}
	return strings.Join(lines, "\n")
}

// addMatchResources adds the items of a policy's matchConstraints or a binding's matchResources:
// the selectors, the rules it matches and the rules it excludes, and the match policy.
func (d *detailItems) addMatchResources(match map[string]any) {
	d.add("Namespace Selector", strings.Join(labelSelector(match, "namespaceSelector"), "\n"))
	d.add("Object Selector", strings.Join(labelSelector(match, "objectSelector"), "\n"))
	d.add("Resource Rules", admissionRules(match, "resourceRules"))
	d.add("Excluded Rules", admissionRules(match, "excludeResourceRules"))
	d.add(itemMatchPolicy, model.NestedString(match, "matchPolicy"))
}

// keyServices is the kind key a webhook's service links to.
const keyServices = "services"

// webhooksColumns are the columns of both webhook configuration views (Lens's documentation, not
// captured): Name, Webhooks (how many the configuration has), Age.
var webhooksColumns = []model.Column{
	nameColumn,
	{ID: "webhooks", Title: "Webhooks", Type: model.ColumnNumber},
	ageColumn,
}

// webhookFields are the searchable fields of both webhook configuration kinds.
var webhookFields = map[string]fieldFunc{
	// "webhook" lists the webhooks' names: "validate.policy.example.com".
	"webhook": func(obj *unstructured.Unstructured) []string {
		var out []string
		for _, w := range model.NestedSlice(obj, "webhooks") {
			out = appendUnique(out, model.NestedString(w, "name"))
		}
		return out
	},
	// "service" lists the services the webhooks call, as "namespace/name".
	"service": func(obj *unstructured.Unstructured) []string {
		var out []string
		for _, w := range model.NestedSlice(obj, "webhooks") {
			svc := model.NestedMap(w, "clientConfig", "service")
			if name := model.NestedString(svc, "name"); name != "" {
				out = appendUnique(out, model.NestedString(svc, "namespace")+"/"+name)
			}
		}
		return out
	},
}

// webhookConfigurationCells are the cells of both webhook configuration views. A configuration
// has no state, so no status token.
func webhookConfigurationCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		{Num: int64(len(model.NestedSlice(obj, "webhooks")))},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// webhookConfigurationDetail is the drawer of both webhook configuration kinds: one section per
// webhook, titled by its name, under the heading Webhooks (inferred).
func webhookConfigurationDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var sections []model.DetailSection
	for i, w := range model.NestedSlice(obj, "webhooks") {
		var d detailItems
		client, link := webhookClient(w)
		d.put(linkedItem("Client", client, link))
		d.add("Rules", admissionRules(w, "rules"))
		d.add(itemFailurePolicy, model.NestedString(w, "failurePolicy"))
		d.add(itemMatchPolicy, model.NestedString(w, "matchPolicy"))
		d.add("Side Effects", model.NestedString(w, "sideEffects"))
		if n, ok := model.NestedInt64(w, "timeoutSeconds"); ok {
			d.add("Timeout", itoa(n)+"s")
		}
		d.add(itemReinvocationPolicy, model.NestedString(w, "reinvocationPolicy"))
		d.add("Namespace Selector", strings.Join(labelSelector(w, "namespaceSelector"), "\n"))
		d.add("Object Selector", strings.Join(labelSelector(w, "objectSelector"), "\n"))
		d.add(itemMatchConditions, matchConditions(w, "matchConditions"))
		d.add("Admission Review Versions", strings.Join(model.NestedStrings(w, "admissionReviewVersions"), ", "))
		title := model.NestedString(w, "name")
		if title == "" {
			title = "Webhook " + strconv.Itoa(i+1)
		}
		sections = append(sections, model.DetailSection{Title: title, Group: "Webhooks", Items: d})
	}
	return sections
}

// webhookClient renders where a webhook sends its requests: the service as
// "namespace/name:port/path", with a link to it, or the URL. The CA bundle is left out: it is a
// certificate, long and of no use to read.
func webhookClient(w any) (string, *model.ObjectRef) {
	svc := model.NestedMap(w, "clientConfig", "service")
	if name := model.NestedString(svc, "name"); name != "" {
		ns := model.NestedString(svc, "namespace")
		text := ns + "/" + name
		if port, ok := model.NestedInt64(svc, "port"); ok {
			text += ":" + itoa(port)
		}
		text += model.NestedString(svc, "path")
		return text, linkTo(keyServices, ns, name)
	}
	return model.NestedString(w, "clientConfig", "url"), nil
}
