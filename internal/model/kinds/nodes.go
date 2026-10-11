package kinds

import (
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

const (
	nodeRolePrefix = "node-role.kubernetes.io/"
	// nodeRoleLegacy is the label older installers set instead of a node-role label.
	nodeRoleLegacy = "kubernetes.io/role"
	nodeReady      = "Ready"
)

var nodes = &model.KindProfile{
	Key:      "nodes",
	Version:  "v1",
	Resource: "nodes",
	Kind:     "Node",
	Title:    "Nodes",
	Columns: []model.Column{
		nameColumn,
		{ID: "taints", Title: "Taints", Type: model.ColumnNumber},
		{ID: "roles", Title: "Roles", Type: model.ColumnText},
		{ID: "version", Title: "Version", Type: model.ColumnText},
		ageColumn,
		conditionsColumn,
	},
	Cells: nodeCells,
	Fields: map[string]fieldFunc{
		"internalIP": func(obj *unstructured.Unstructured) []string {
			return nodeAddresses(obj, "InternalIP")
		},
		idExternalIP: func(obj *unstructured.Unstructured) []string {
			return nodeAddresses(obj, "ExternalIP")
		},
		"hostname": func(obj *unstructured.Unstructured) []string {
			return nodeAddresses(obj, "Hostname")
		},
		"role": func(obj *unstructured.Unstructured) []string {
			return nodeRoles(obj)
		},
		"version": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "status", "nodeInfo", "kubeletVersion"))
		},
	},
	Detail: nodeDetail,
}

func nodeCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	roles := "<none>"
	if r := nodeRoles(obj); len(r) > 0 {
		roles = strings.Join(r, ", ")
	}
	text, token := nodeConditions(obj)
	return []model.Cell{
		nameCell(obj),
		{Num: int64(len(model.NestedSlice(obj, "spec", "taints")))},
		{Text: roles},
		{Text: model.NestedString(obj, "status", "nodeInfo", "kubeletVersion")},
		model.AgeCell(created(obj)),
		{Text: text, Status: token},
	}, token
}

// nodeRoles returns the roles from node-role.kubernetes.io/<role> labels and the legacy
// kubernetes.io/role label, sorted.
func nodeRoles(obj any) []string {
	var roles []string
	for k, v := range model.NestedStringMap(obj, "metadata", "labels") {
		switch {
		case strings.HasPrefix(k, nodeRolePrefix):
			roles = appendUnique(roles, strings.TrimPrefix(k, nodeRolePrefix))
		case k == nodeRoleLegacy:
			roles = appendUnique(roles, v)
		}
	}
	sort.Strings(roles)
	return roles
}

// nodeConditions renders the Conditions column: Ready, NotReady or Unknown from the Ready
// condition, then every other condition that is True (MemoryPressure, DiskPressure, PIDPressure,
// NetworkUnavailable, and what node-problem-detector adds: all of them name a problem when True),
// then SchedulingDisabled for a cordoned node. Anything appended to a Ready node makes it a
// warning; NotReady stays an error.
func nodeConditions(obj any) (string, model.Status) {
	base, token := textUnknown, model.StatusUnknown
	var problems []string
	for _, e := range model.NestedSlice(obj, "status", "conditions") {
		typ, status := model.NestedString(e, "type"), model.NestedString(e, "status")
		switch {
		case typ == nodeReady && status == condTrue:
			base, token = nodeReady, model.StatusOK
		case typ == nodeReady && status == condFalse:
			base, token = "NotReady", model.StatusError
		case typ != nodeReady && typ != "" && status == condTrue:
			problems = appendUnique(problems, typ)
		}
	}
	sort.Strings(problems)
	if unschedulable, _ := model.NestedBool(obj, "spec", "unschedulable"); unschedulable {
		problems = append(problems, "SchedulingDisabled")
	}
	if len(problems) == 0 {
		return base, token
	}
	if token == model.StatusOK {
		token = model.StatusWarning
	}
	return base + " " + strings.Join(problems, " "), token
}

// nodeAddresses returns the node's addresses of one type.
func nodeAddresses(obj any, typ string) []string {
	var out []string
	for _, a := range model.NestedSlice(obj, "status", "addresses") {
		if model.NestedString(a, "type") == typ {
			out = appendUnique(out, model.NestedString(a, "address"))
		}
	}
	return out
}

func nodeDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var sections []model.DetailSection
	add := func(title string, items detailItems) {
		if len(items) > 0 {
			sections = append(sections, model.DetailSection{Title: title, Items: items})
		}
	}

	var addresses detailItems
	for _, a := range model.NestedSlice(obj, "status", "addresses") {
		addresses.add(model.NestedString(a, "type"), model.NestedString(a, "address"))
	}
	add("Addresses", addresses)

	var info detailItems
	for _, f := range []struct{ name, field string }{
		{"OS Image", "osImage"},
		{"Kernel Version", "kernelVersion"},
		{"Container Runtime", "containerRuntimeVersion"},
		{"Kubelet Version", "kubeletVersion"},
		{"Architecture", "architecture"},
	} {
		info.add(f.name, model.NestedString(obj, "status", "nodeInfo", f.field))
	}
	add("System Info", info)

	add("Capacity", resourceList(obj, "capacity"))
	add("Allocatable", resourceList(obj, "allocatable"))

	var taints detailItems
	for _, t := range model.NestedSlice(obj, "spec", "taints") {
		taints.add("", taintText(t))
	}
	add("Taints", taints)

	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), nodeConditionToken); ok {
		sections = append(sections, cs)
	}
	return sections
}

// resourceList renders status.capacity or status.allocatable, one resource per item, sorted.
// Quantities are strings in the API; a Go-built object may hold numbers, which render too.
func resourceList(obj any, field string) detailItems {
	m := model.NestedMap(obj, "status", field)
	var items detailItems
	for _, k := range sortedKeys(m) {
		items.add(k, model.NestedIntOrString(m, k))
	}
	return items
}

// taintText renders a taint as "key=value:Effect", or "key:Effect" without a value.
func taintText(t any) string {
	key := model.NestedString(t, "key")
	if key == "" {
		return ""
	}
	if v := model.NestedString(t, "value"); v != "" {
		key += "=" + v
	}
	if e := model.NestedString(t, "effect"); e != "" {
		key += ":" + e
	}
	return key
}

func nodeConditionToken(c condition) model.Status {
	if c.Type == nodeReady {
		return conditionToken(c.Status, model.StatusOK, model.StatusError)
	}
	return conditionToken(c.Status, model.StatusWarning, model.StatusOK)
}
