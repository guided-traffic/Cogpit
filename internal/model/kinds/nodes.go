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
	// nodeSchedulingDisabled is how a cordoned node's Conditions start.
	nodeSchedulingDisabled = "SchedulingDisabled"
)

var nodes = &model.KindProfile{
	Key:      "nodes",
	Version:  "v1",
	Resource: "nodes",
	Kind:     "Node",
	Title:    "Nodes",
	Columns: []model.Column{
		nameColumn,
		warningColumn,
		{ID: "taints", Title: itemTaints, Type: model.ColumnNumber},
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
	Detail:     nodeDetail,
	Needs:      []model.RelatedIndex{PodsByNode},
	DetailWith: nodeDetailWith,
}

func nodeCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	roles := "<none>"
	if r := nodeRoles(obj); len(r) > 0 {
		roles = strings.Join(r, ", ")
	}
	s := summariseNode(obj)
	return []model.Cell{
		nameCell(obj),
		s.warning(),
		{Num: int64(len(model.NestedSlice(obj, "spec", "taints")))},
		{Text: roles},
		{Text: model.NestedString(obj, "status", "nodeInfo", "kubeletVersion")},
		model.AgeCell(created(obj)),
		s.conditionsCell(),
	}, s.token
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

// nodeSummary is what the Nodes list derives from a node's conditions and spec.
type nodeSummary struct {
	base        string       // Ready, NotReady or Unknown, from the Ready condition
	baseToken   model.Status // ok, error or unknown
	readyStatus string       // the Ready condition's status; "" when the node reports none
	problems    []string     // other conditions that are True, sorted: they name a problem
	cordoned    bool         // spec.unschedulable
	token       model.Status // the row's status
}

// summariseNode reads the Ready condition, every other condition that is True (MemoryPressure,
// DiskPressure, PIDPressure, NetworkUnavailable, and what node-problem-detector adds: all of them
// name a problem when True), and whether the node is cordoned. The row's status is the Ready
// state's; a Ready node with a problem condition is a warning. Cordoning is not a problem: Lens
// shows a cordoned Ready node without a warning (capture, 2026-10-11).
func summariseNode(obj any) nodeSummary {
	s := nodeSummary{base: textUnknown, baseToken: model.StatusUnknown}
	for _, e := range model.NestedSlice(obj, "status", "conditions") {
		typ, status := model.NestedString(e, "type"), model.NestedString(e, "status")
		switch {
		case typ == nodeReady:
			s.readyStatus = status
			switch status {
			case condTrue:
				s.base, s.baseToken = nodeReady, model.StatusOK
			case condFalse:
				s.base, s.baseToken = "NotReady", model.StatusError
			}
		case typ != "" && status == condTrue:
			s.problems = appendUnique(s.problems, typ)
		}
	}
	sort.Strings(s.problems)
	s.cordoned, _ = model.NestedBool(obj, "spec", "unschedulable")
	s.token = s.baseToken
	if s.token == model.StatusOK && len(s.problems) > 0 {
		s.token = model.StatusWarning
	}
	return s
}

// parts are the Conditions column's coloured parts: "SchedulingDisabled" first and uncoloured on
// a cordoned node, then the Ready state in its colour ("Ready" green), then the problem conditions
// in the warning colour (capture of a cordoned node, 2026-10-11; the place and colour of the
// problem conditions are inferred).
func (s nodeSummary) parts() []model.CellPart {
	parts := make([]model.CellPart, 0, 2+len(s.problems))
	if s.cordoned {
		parts = append(parts, model.CellPart{Text: nodeSchedulingDisabled})
	}
	parts = append(parts, model.CellPart{Text: s.base, Status: s.baseToken})
	for _, p := range s.problems {
		parts = append(parts, model.CellPart{Text: p, Status: model.StatusWarning})
	}
	return parts
}

// conditionsCell is the Conditions column's cell; its overall status is the Ready state's.
func (s nodeSummary) conditionsCell() model.Cell {
	return model.PartsCell(s.baseToken, s.parts()...)
}

// warning is the node's cell of the warning column. Inferred: Lens shows the column for nodes and
// gives a cordoned node no icon (capture, 2026-10-11); which states do get one is not captured.
// Cogpit's rule: a node warns when it is not Ready (the Ready condition False or Unknown) or
// reports a problem condition, and the reason names them.
func (s nodeSummary) warning() model.Cell {
	var reasons []string
	switch s.readyStatus {
	case condFalse:
		reasons = append(reasons, "NotReady")
	case textUnknown:
		reasons = append(reasons, "Ready status Unknown: the kubelet stopped reporting")
	}
	reasons = append(reasons, s.problems...)
	if len(reasons) == 0 {
		return warningCell(model.StatusNone, "")
	}
	return warningCell(model.StatusWarning, strings.Join(reasons, ", "))
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

// nodeDetail is the node's drawer as Lens shows it (capture, 2026-10-11): Properties (after the
// generic metadata: Addresses, OS, OS Image, Kernel version, Container runtime, Kubelet version,
// Conditions), then Capacity and Allocatable as tables. The node's Pods follow in nodeDetailWith,
// which has them. Taints are Cogpit's own addition to Properties.
func nodeDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var props detailItems
	var addresses []string
	for _, a := range model.NestedSlice(obj, "status", "addresses") {
		if addr := model.NestedString(a, "address"); addr != "" {
			addresses = append(addresses, model.NestedString(a, "type")+": "+addr)
		}
	}
	props.add("Addresses", strings.Join(addresses, "\n"))
	info := func(field string) string { return model.NestedString(obj, "status", "nodeInfo", field) }
	os := info("operatingSystem")
	if arch := info("architecture"); arch != "" {
		if os == "" {
			os = arch
		} else {
			os += " (" + arch + ")"
		}
	}
	props.add("OS", os)
	props.add("OS Image", info("osImage"))
	props.add("Kernel version", info("kernelVersion"))
	props.add("Container runtime", info("containerRuntimeVersion"))
	props.add("Kubelet version", info("kubeletVersion"))

	// Conditions: chips as the list's Conditions column reads, the full conditions as the text.
	conds := conditions(obj, "status", "conditions")
	lines := make([]string, 0, len(conds))
	for _, c := range conds {
		lines = append(lines, c.Type+": "+c.text())
	}
	if len(lines) > 0 {
		parts := summariseNode(obj).parts()
		chips := make([]model.Chip, 0, len(parts))
		for _, p := range parts {
			chips = append(chips, model.Chip{Text: p.Text, Status: p.Status})
		}
		props.put(model.DetailItem{Name: itemConditions, Value: strings.Join(lines, "\n"), Chips: chips})
	}

	var taints []string
	for _, t := range model.NestedSlice(obj, "spec", "taints") {
		if text := taintText(t); text != "" {
			taints = append(taints, text)
		}
	}
	props.put(model.DetailItem{Name: itemTaints, Value: strings.Join(taints, "\n"), Fold: itemTaints})

	sections := []model.DetailSection{{Title: model.TitleProperties, Items: props}}
	for _, f := range []struct{ title, field string }{{"Capacity", "capacity"}, {"Allocatable", "allocatable"}} {
		if t := resourceTable(obj, f.field); t != nil {
			sections = append(sections, model.DetailSection{Title: f.title, Table: t})
		}
	}
	return sections
}

// resourceTitles names the resources of a node's capacity in the order Lens's tables show them
// (capture, 2026-10-11: CPU, Memory, Ephemeral Storage, Hugepages, Pods); other resources (GPUs,
// devices) follow, sorted, under their own names.
var resourceTitles = []struct{ name, title string }{
	{"cpu", "CPU"}, {"memory", "Memory"}, {"ephemeral-storage", "Ephemeral Storage"},
	{"hugepages-1Gi", "Hugepages 1Gi"}, {"hugepages-2Mi", "Hugepages 2Mi"}, {podsResource, podsTitle},
}

// resourceTable renders status.capacity or status.allocatable as a one-row table, or nil when the
// node reports none. Quantities are shown as the API reports them; a Go-built object may hold
// numbers, which render too.
func resourceTable(obj any, field string) *model.DetailTable {
	m := model.NestedMap(obj, "status", field)
	if len(m) == 0 {
		return nil
	}
	t := &model.DetailTable{Rows: [][]string{{}}}
	known := map[string]bool{}
	for _, r := range resourceTitles {
		known[r.name] = true
		if v := model.NestedIntOrString(m, r.name); v != "" {
			t.Columns = append(t.Columns, r.title)
			t.Rows[0] = append(t.Rows[0], v)
		}
	}
	for _, k := range sortedKeys(m) {
		if known[k] {
			continue
		}
		if v := model.NestedIntOrString(m, k); v != "" {
			t.Columns = append(t.Columns, k)
			t.Rows[0] = append(t.Rows[0], v)
		}
	}
	if len(t.Columns) == 0 {
		return nil
	}
	return t
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

// nodePodColumns are the columns of the node's Pods table as Lens shows them (capture,
// 2026-10-11): Name, the warning icon, Node, Namespace, Ready, CPU, Memory, Status. CPU and Memory
// need a metrics provider (M7) and stay empty until then.
var nodePodColumns = []model.Column{
	nameColumn,
	warningColumn,
	{ID: idNode, Title: titleNode, Type: model.ColumnText},
	namespaceColumn,
	{ID: fieldReady, Title: nodeReady, Type: model.ColumnText},
	{ID: "cpu", Title: "CPU", Type: model.ColumnText},
	{ID: "memory", Title: "Memory", Type: model.ColumnText},
	statusColumn,
}

// nodeDetailWith is nodeDetail with the node's Pods table after Allocatable, from the
// related-objects index PodsByNode: one row per pod scheduled to the node, sorted by name, the
// name and namespace as links, the warning icon and status as in the Pods view, Ready as the
// ready containers of all ("1/2"). The node's own name is plain text: it links to the drawer's
// object. A node without pods, or whose pods are still listing, has no Pods section.
func nodeDetailWith(obj *unstructured.Unstructured, rel model.Related) []model.DetailSection {
	sections := nodeDetail(obj)
	pods := append([]*unstructured.Unstructured(nil), rel.Objects(PodsByNode, obj.GetName())...)
	if len(pods) == 0 {
		return sections
	}
	sortByName(pods)
	rows := make([][]model.Cell, 0, len(pods))
	for _, p := range pods {
		s := summarisePod(p)
		rows = append(rows, []model.Cell{
			{Text: p.GetName(), Link: linkTo(podsResource, p.GetNamespace(), p.GetName())},
			podWarning(p, s),
			{Text: model.NestedString(p, "spec", "nodeName")},
			namespaceCell(p),
			ratio(s.ready, s.total),
			{}, // CPU: metrics (M7)
			{}, // Memory: metrics (M7)
			{Text: s.text, Status: s.status},
		})
	}
	return append(sections, model.DetailSection{Title: podsTitle, Table: model.CellTable(nodePodColumns, rows)})
}
