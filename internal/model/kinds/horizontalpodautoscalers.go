package kinds

import (
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Horizontal Pod Autoscaler condition types, the reasons the status rule reads, and the texts of
// the Metrics column.
const (
	hpaAbleToScale     = "AbleToScale"
	hpaScalingActive   = "ScalingActive"
	hpaScalingLimited  = "ScalingLimited"
	hpaScalingDisabled = "ScalingDisabled" // ScalingActive False: the target is scaled to zero
	hpaTooManyReplicas = "TooManyReplicas" // ScalingLimited True: it wants more than maxReplicas
	hpaUnknownValue    = "<unknown>"
	hpaUtilization     = "Utilization" // the metric target types
	hpaAverageValue    = "AverageValue"
	hpaValue           = "Value"
	hpaNoMetrics       = "<none>"
	hpaTypeResource    = "Resource" // a metric of the pods' resource use
	titleReplicas      = "Replicas"
	idReplicas         = "replicas"
	// hpaMetricsShown is how many metrics the Metrics column lists before "+ N more...".
	hpaMetricsShown = 2
)

// hpaSources maps a metric's type to the field of its source, in spec.metrics and in
// status.currentMetrics alike.
var hpaSources = map[string]string{
	hpaTypeResource:     idResource,
	"ContainerResource": "containerResource",
	"Pods":              "pods",
	"Object":            "object",
	"External":          "external",
}

// horizontalPodAutoscalers has the columns of Lens's documentation (not captured): Name,
// Namespace, Metrics, Min Pods, Max Pods, Replicas, Age, Status. Inferred: Metrics in kubectl's
// TARGETS form, each metric named ("cpu: 45%/80%", the first two, then "+ N more..."), Replicas as
// the current count, Status as the True condition types with the status rule of hpaStatus, and the
// drawer.
var horizontalPodAutoscalers = &model.KindProfile{
	Key:        "horizontalpodautoscalers",
	Group:      "autoscaling",
	Version:    "v2",
	Resource:   "horizontalpodautoscalers",
	Kind:       "HorizontalPodAutoscaler",
	Title:      "Horizontal Pod Autoscalers",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "metrics", Title: "Metrics", Type: model.ColumnText},
		{ID: "minPods", Title: "Min Pods", Type: model.ColumnNumber},
		{ID: "maxPods", Title: "Max Pods", Type: model.ColumnNumber},
		{ID: idReplicas, Title: titleReplicas, Type: model.ColumnNumber},
		ageColumn,
		statusColumn,
	},
	Cells: hpaCells,
	Fields: map[string]fieldFunc{
		// "target" is the scaled object as "Kind/name": "Deployment/web".
		"target": func(obj *unstructured.Unstructured) []string {
			text, _ := hpaTargetRef(obj)
			return nonEmpty(text)
		},
		// "metric" lists the metrics' names: "cpu", "packets-per-second".
		"metric": func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, m := range hpaMetrics(obj) {
				out = appendUnique(out, m.name)
			}
			return out
		},
		idStatus: func(obj *unstructured.Unstructured) []string {
			text, _ := hpaStatus(conditions(obj, "status", "conditions"))
			return nonEmpty(text)
		},
	},
	Detail: hpaDetail,
}

func hpaCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	text, token := hpaStatus(conditions(obj, "status", "conditions"))
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: hpaMetricsText(hpaMetrics(obj))},
		{Num: hpaMinReplicas(obj)},
		{Num: nonNegative(obj, "spec", "maxReplicas")},
		{Num: nonNegative(obj, "status", "currentReplicas")},
		model.AgeCell(created(obj)),
		{Text: text, Status: token},
	}, token
}

// hpaMinReplicas is spec.minReplicas, which the API defaults to 1.
func hpaMinReplicas(obj any) int64 {
	n, ok := model.NestedInt64(obj, "spec", "minReplicas")
	if !ok {
		return 1
	}
	if n < 0 {
		return 0
	}
	return n
}

// hpaStatus renders the Status column, the condition types that are True, sorted, and colours
// it: an autoscaler that cannot scale its target or cannot compute its metrics is an error; one
// held at maxReplicas while it wants more is a warning; scaling switched off (the target scaled to
// zero) is not coloured; without conditions the controller has not seen it yet.
func hpaStatus(conds []condition) (string, model.Status) {
	trueTypes := make([]string, 0, len(conds))
	for _, c := range conds {
		if c.Status == condTrue {
			trueTypes = append(trueTypes, c.Type)
		}
	}
	sort.Strings(trueTypes)
	text := strings.Join(trueTypes, " ")
	if len(conds) == 0 {
		return text, model.StatusPending
	}
	able, _ := findCondition(conds, hpaAbleToScale)
	active, _ := findCondition(conds, hpaScalingActive)
	limited, _ := findCondition(conds, hpaScalingLimited)
	switch {
	case able.Status == condFalse:
		return text, model.StatusError
	case active.Status == condFalse && active.Reason == hpaScalingDisabled:
		return text, model.StatusNone
	case active.Status == condFalse:
		return text, model.StatusError
	case limited.Status == condTrue && limited.Reason == hpaTooManyReplicas:
		return text, model.StatusWarning
	}
	return text, model.StatusOK
}

// hpaConditionToken colours one condition in the drawer, by the rule of hpaStatus.
func hpaConditionToken(c condition) model.Status {
	switch {
	case c.Type == hpaScalingLimited:
		if c.Status == condTrue && c.Reason == hpaTooManyReplicas {
			return model.StatusWarning
		}
		return model.StatusNone
	case c.Type == hpaScalingActive && c.Status == condFalse && c.Reason == hpaScalingDisabled:
		return model.StatusNone
	}
	return conditionToken(c.Status, model.StatusOK, model.StatusError)
}

// hpaMetric is one entry of spec.metrics with its current value from status.currentMetrics.
type hpaMetric struct {
	typ, name       string
	current, target string
}

// hpaMetrics reads spec.metrics and finds each metric's current value in status.currentMetrics by
// its type and name, not by position as kubectl does, so a status written in another order still
// lines up.
func hpaMetrics(obj any) []hpaMetric {
	specs := model.NestedSlice(obj, "spec", "metrics")
	statuses := model.NestedSlice(obj, "status", "currentMetrics")
	out := make([]hpaMetric, 0, len(specs))
	for _, s := range specs {
		typ := model.NestedString(s, "type")
		field, known := hpaSources[typ]
		if !known {
			out = append(out, hpaMetric{typ: typ, name: typ, current: hpaUnknownValue, target: hpaUnknownValue})
			continue
		}
		src := model.NestedMap(s, field)
		m := hpaMetric{typ: typ, name: hpaMetricName(src), current: hpaUnknownValue}
		targetType, target := hpaMetricTarget(src)
		m.target = target
		for _, st := range statuses {
			cur := model.NestedMap(st, field)
			if model.NestedString(st, "type") != typ || hpaMetricName(cur) != m.name {
				continue
			}
			if v := hpaCurrent(model.NestedMap(cur, "current"), targetType); v != "" {
				m.current = v
			}
			break
		}
		out = append(out, m)
	}
	return out
}

// hpaMetricName names a metric source: the resource ("cpu"), with its container for a container
// resource ("cpu (app)"), else the metric's name.
func hpaMetricName(src map[string]any) string {
	if name := model.NestedString(src, "metric", "name"); name != "" {
		return name
	}
	name := model.NestedString(src, "name")
	if c := model.NestedString(src, "container"); c != "" {
		name += " (" + c + ")"
	}
	return name
}

// hpaMetricTarget returns a metric's target type and its text: "80%" for a utilization, the
// quantity for an average value or a value. A target without a type is typed by its field.
func hpaMetricTarget(src map[string]any) (string, string) {
	t := model.NestedMap(src, "target")
	typ := model.NestedString(t, "type")
	switch {
	case typ == hpaUtilization || (typ == "" && hasValue(t, "averageUtilization")):
		if n, ok := model.NestedInt64(t, "averageUtilization"); ok {
			return hpaUtilization, itoa(n) + "%"
		}
		return hpaUtilization, hpaUnknownValue
	case typ == hpaAverageValue || (typ == "" && hasValue(t, "averageValue")):
		return hpaAverageValue, orUnknown(model.NestedIntOrString(t, "averageValue"))
	}
	return hpaValue, orUnknown(model.NestedIntOrString(t, "value"))
}

// hpaCurrent renders a metric's current value in the form of its target.
func hpaCurrent(cur map[string]any, targetType string) string {
	switch targetType {
	case hpaUtilization:
		if n, ok := model.NestedInt64(cur, "averageUtilization"); ok {
			return itoa(n) + "%"
		}
		return ""
	case hpaAverageValue:
		return model.NestedIntOrString(cur, "averageValue")
	}
	if v := model.NestedIntOrString(cur, "value"); v != "" {
		return v
	}
	return model.NestedIntOrString(cur, "averageValue")
}

// hasValue reports whether the object has the field, null included.
func hasValue(m map[string]any, field string) bool {
	_, ok := model.NestedValue(m, field)
	return ok
}

// orUnknown renders a value the object does not set as "<unknown>", as kubectl does.
func orUnknown(s string) string {
	if s == "" {
		return hpaUnknownValue
	}
	return s
}

// hpaMetricsText is the Metrics column: "cpu: 45%/80%, memory: 60%/70% + 1 more...".
func hpaMetricsText(metrics []hpaMetric) string {
	if len(metrics) == 0 {
		return hpaNoMetrics
	}
	shown := metrics[:min(len(metrics), hpaMetricsShown)]
	parts := make([]string, len(shown))
	for i, m := range shown {
		parts[i] = m.name + ": " + m.current + "/" + m.target
	}
	s := strings.Join(parts, ", ")
	if more := len(metrics) - len(shown); more > 0 {
		s += " + " + itoa(int64(more)) + " more..."
	}
	return s
}

// hpaTargetRef returns spec.scaleTargetRef as "Kind/name" and a link to it for built-in kinds.
func hpaTargetRef(obj *unstructured.Unstructured) (string, *model.ObjectRef) {
	ref := model.NestedMap(obj, "spec", "scaleTargetRef")
	kind, name := model.NestedString(ref, "kind"), model.NestedString(ref, "name")
	if name == "" {
		return "", nil
	}
	text := name
	if kind != "" {
		text = kind + "/" + name
	}
	return text, model.RefTo(model.NestedString(ref, "apiVersion"), kind, obj.GetNamespace(), name)
}

func hpaDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	text, link := hpaTargetRef(obj)
	d.put(linkedItem("Reference", text, link))
	d.add("Min Pods", itoa(hpaMinReplicas(obj)))
	if n, ok := model.NestedInt64(obj, "spec", "maxReplicas"); ok {
		d.add("Max Pods", itoa(n))
	}
	if n, ok := model.NestedInt64(obj, "status", "currentReplicas"); ok {
		d.add("Current Replicas", itoa(n))
	}
	if n, ok := model.NestedInt64(obj, "status", "desiredReplicas"); ok {
		d.add("Desired Replicas", itoa(n))
	}
	d.add("Last Scale", timestamp(model.NestedTime(obj, "status", "lastScaleTime")))
	d.add("Scale Up", hpaRules(model.NestedMap(obj, "spec", "behavior", "scaleUp")))
	d.add("Scale Down", hpaRules(model.NestedMap(obj, "spec", "behavior", "scaleDown")))
	sections := withSection(nil, model.TitleProperties, d)

	if metrics := hpaMetrics(obj); len(metrics) > 0 {
		t := &model.DetailTable{Columns: []string{"Metric", titleType, "Current", "Target"}}
		for _, m := range metrics {
			t.Rows = append(t.Rows, []string{m.name, m.typ, m.current, m.target})
		}
		sections = append(sections, model.DetailSection{Title: "Metrics", Table: t})
	}
	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), hpaConditionToken); ok {
		sections = append(sections, cs)
	}
	return sections
}

// hpaRules renders one direction of spec.behavior: its policies ("4 pods per 15s, 100% per 15s"),
// the select policy and the stabilization window; "" when the direction is not set.
func hpaRules(rules map[string]any) string {
	if rules == nil {
		return ""
	}
	var parts []string
	for _, p := range model.NestedSlice(rules, "policies") {
		value, ok := model.NestedInt64(p, "value")
		if !ok {
			continue
		}
		amount := itoa(value) + " pods"
		if model.NestedString(p, "type") == "Percent" {
			amount = itoa(value) + "%"
		}
		if period, ok := model.NestedInt64(p, "periodSeconds"); ok {
			amount += " per " + itoa(period) + "s"
		}
		parts = append(parts, amount)
	}
	s := strings.Join(parts, ", ")
	if sel := model.NestedString(rules, "selectPolicy"); sel != "" {
		s = appendClause(s, "select "+sel)
	}
	if w, ok := model.NestedInt64(rules, "stabilizationWindowSeconds"); ok {
		s = appendClause(s, "stabilization window "+itoa(w)+"s")
	}
	return s
}

// appendClause joins clauses of a drawer value with "; ".
func appendClause(s, clause string) string {
	if s == "" {
		return clause
	}
	return s + "; " + clause
}
