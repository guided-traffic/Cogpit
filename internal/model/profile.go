package model

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// KindProfile makes a Kubernetes kind look the way Lens shows it: its columns, its status rule,
// the fields agents can search by, and the sections of its detail drawer. Built-in profiles live
// in package kinds, one file per kind.
//
// Every function field must tolerate any unstructured content without panicking: missing fields,
// wrong types, null. Return empty cells instead.
type KindProfile struct {
	Key        string // kind key: the plural resource name, "pods"
	Group      string // API group, "" for the core group
	Version    string // preferred version, "v1"
	Resource   string // plural resource, "pods"
	Kind       string // Kubernetes Kind, "Pod"
	Title      string // view title as Lens shows it, "Pods"
	Namespaced bool

	// Columns in Lens's order. The first column is the name.
	Columns []Column

	// Cells returns one cell per column, in the order of Columns, and the row's status token.
	Cells func(obj *unstructured.Unstructured) ([]Cell, Status)

	// Fields are the searchable fields agents use with resources_find, beyond the generic "name",
	// "namespace" and "label" fields that every kind has (see FieldValues). Each returns the
	// object's values for that field: "clusterIP" → ["10.96.100.100"].
	Fields map[string]func(obj *unstructured.Unstructured) []string

	// Detail returns the kind's own drawer sections (containers, conditions, ports). The engine
	// adds the generic metadata section, the events and the YAML itself.
	Detail func(obj *unstructured.Unstructured) []DetailSection
}

// GVR returns the profile's group, version and resource.
func (p *KindProfile) GVR() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: p.Group, Version: p.Version, Resource: p.Resource}
}

// ColumnIndex returns the index of the column with the given id, or -1.
func (p *KindProfile) ColumnIndex(id string) int {
	for i, c := range p.Columns {
		if c.ID == id {
			return i
		}
	}
	return -1
}

// FieldNames returns every field name resources_find accepts for this kind, generic ones first.
func (p *KindProfile) FieldNames() []string {
	names := make([]string, 0, 3+len(p.Fields))
	names = append(names, "name", "namespace", "label")
	for k := range p.Fields {
		names = append(names, k)
	}
	return names
}

// BuildRow turns an object into a table row for the given kubeconfig context. It never panics:
// a profile whose Cells function panics yields a row with the name and an unknown status.
func BuildRow(p *KindProfile, context string, obj *unstructured.Unstructured) (row Row) {
	row = Row{
		Ref: ObjectRef{
			Context:   context,
			Kind:      p.Key,
			Namespace: obj.GetNamespace(),
			Name:      obj.GetName(),
		},
		UID:             string(obj.GetUID()),
		ResourceVersion: obj.GetResourceVersion(),
	}
	defer func() {
		if r := recover(); r != nil {
			cells := make([]Cell, len(p.Columns))
			if len(cells) > 0 {
				cells[0] = Cell{Text: obj.GetName()}
			}
			row.Cells, row.Status = cells, StatusUnknown
		}
	}()
	if p.Cells == nil {
		row.Cells = make([]Cell, len(p.Columns))
		if len(row.Cells) > 0 {
			row.Cells[0] = Cell{Text: obj.GetName()}
		}
		return row
	}
	row.Cells, row.Status = p.Cells(obj)
	if len(row.Cells) != len(p.Columns) {
		fixed := make([]Cell, len(p.Columns))
		copy(fixed, row.Cells)
		row.Cells = fixed
	}
	if obj.GetDeletionTimestamp() != nil {
		row.Status = StatusTerminating
	}
	return row
}

// FieldValues returns the object's values for a searchable field. The generic fields are "name",
// "namespace" and "label" (each label as "key=value"); the rest come from the profile. An unknown
// field returns nil.
func FieldValues(p *KindProfile, field string, obj *unstructured.Unstructured) []string {
	switch field {
	case "name":
		return []string{obj.GetName()}
	case "namespace":
		return []string{obj.GetNamespace()}
	case "label", "labels":
		labels := obj.GetLabels()
		out := make([]string, 0, len(labels))
		for k, v := range labels {
			out = append(out, k+"="+v)
		}
		return out
	}
	if f, ok := p.Fields[field]; ok && f != nil {
		return safeStrings(f, obj)
	}
	return nil
}

func safeStrings(f func(*unstructured.Unstructured) []string, obj *unstructured.Unstructured) (out []string) {
	defer func() {
		if recover() != nil {
			out = nil
		}
	}()
	return f(obj)
}

// MatchesFilter reports whether a row matches the search text of a view: every whitespace-
// separated term must occur, case-insensitively, in one of the row's cells.
func MatchesFilter(row Row, filter string) bool {
	terms := strings.Fields(strings.ToLower(filter))
	if len(terms) == 0 {
		return true
	}
	for _, term := range terms {
		found := false
		for _, c := range row.Cells {
			if strings.Contains(strings.ToLower(c.Text), term) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
