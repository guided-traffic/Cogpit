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

	// What the cache holds for the kind (architecture §5). The engine builds every cache scope of
	// the kind from these, so a view, the drawer, Lookup and agents' reads share one informer.

	// MetadataOnly makes the cache hold only the objects' metadata: the objects Cells and Detail
	// see have apiVersion meta.k8s.io/v1, kind PartialObjectMetadata and metadata, nothing else
	// (Secrets, H-8). Data the cache does not hold comes from a direct read the drawer makes.
	MetadataOnly bool
	// LabelSelector narrows the kind's watch on the server: "owner=helm".
	LabelSelector string

	// Derive makes the kind virtual: its objects are not the cached objects of GVR (with
	// LabelSelector and MetadataOnly) but what Derive makes of them: Helm releases, one per release,
	// from their release Secrets. It gets every cached object of the scope (or of one namespace) and
	// returns the kind's objects, each with a name, a namespace if Namespaced, and a UID and
	// resourceVersion that change when the object does (the row cache keys on them). It must be
	// pure, must not modify its input, and must tolerate any content without panicking; the engine
	// calls it on every snapshot, so it memoizes expensive work itself. Commands that write API
	// objects (object.edit, object.delete) do not apply to virtual kinds.
	Derive func(objs []*unstructured.Unstructured) []*unstructured.Unstructured
	// Local marks a kind whose objects Cogpit makes itself, served from the backend's local store
	// and never from the API server: the port forwards. Group and Version name Cogpit's own API
	// (LocalGroup, "v1"). Commands that write API objects do not apply to local kinds.
	Local bool

	// Needs are the indexes over other kinds' cached objects that CellsWith and DetailWith read
	// (a PVC's pods, a node's pods). While a view or the drawer of this kind is open, the engine
	// holds a cache handle on each index's kind and rebuilds the rows when those objects change.
	Needs []RelatedIndex
	// CellsWith replaces Cells when set: the same, with the related objects of Needs. rel is never
	// nil; its lookups return nothing while an index's kind is still listing or not served.
	CellsWith func(obj *unstructured.Unstructured, rel Related) ([]Cell, Status)
	// DetailWith replaces Detail when set: the same, with the related objects of Needs.
	DetailWith func(obj *unstructured.Unstructured, rel Related) []DetailSection
}

// LocalGroup is the API group of the local kinds, the objects Cogpit makes itself (port forwards).
// No Kubernetes API server serves it.
const LocalGroup = "cogpit.io"

// APIObject reports whether the kind's objects are objects of the cluster's API that commands may
// write: false for virtual (Derive) and local kinds.
func (p *KindProfile) APIObject() bool { return p.Derive == nil && !p.Local }

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
// a profile whose Cells function panics yields a row with the name and an unknown status. A
// profile with CellsWith sees no related objects; the engine's views use BuildRowWith.
func BuildRow(p *KindProfile, context string, obj *unstructured.Unstructured) (row Row) {
	return BuildRowWith(p, context, obj, nil)
}

// BuildRowWith is BuildRow with the related objects of the profile's Needs; nil means none.
func BuildRowWith(p *KindProfile, context string, obj *unstructured.Unstructured, rel Related) (row Row) {
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
	switch {
	case p.CellsWith != nil:
		row.Cells, row.Status = p.CellsWith(obj, relatedOrNone(rel))
	case p.Cells != nil:
		row.Cells, row.Status = p.Cells(obj)
	default:
		row.Cells = make([]Cell, len(p.Columns))
		if len(row.Cells) > 0 {
			row.Cells[0] = Cell{Text: obj.GetName()}
		}
		return row
	}
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
