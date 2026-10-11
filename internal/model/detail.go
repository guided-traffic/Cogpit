package model

import (
	"sort"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// DetailSection is one titled block of the detail drawer: a list of name/value items, or, for
// lists such as labels, items without names.
type DetailSection struct {
	Title string       `json:"title"`
	Items []DetailItem `json:"items,omitempty"`
}

// DetailItem is one line of a detail section.
type DetailItem struct {
	Name   string `json:"name,omitempty"`
	Value  string `json:"value"`
	Status Status `json:"status,omitempty"`
}

// MetadataSection returns the generic metadata section every kind shows first: created, name,
// namespace, labels, annotations, controlled by, UID.
func MetadataSection(obj *unstructured.Unstructured, now time.Time) DetailSection {
	s := DetailSection{Title: "Metadata"}
	if ts := obj.GetCreationTimestamp(); !ts.IsZero() {
		s.Items = append(s.Items, DetailItem{
			Name:  "Created",
			Value: FormatAge(now.Sub(ts.Time)) + " ago (" + ts.UTC().Format(time.RFC3339) + ")",
		})
	}
	s.Items = append(s.Items, DetailItem{Name: "Name", Value: obj.GetName()})
	if ns := obj.GetNamespace(); ns != "" {
		s.Items = append(s.Items, DetailItem{Name: "Namespace", Value: ns})
	}
	if labels := sortedPairs(obj.GetLabels()); len(labels) > 0 {
		s.Items = append(s.Items, DetailItem{Name: "Labels", Value: strings.Join(labels, "\n")})
	}
	if ann := obj.GetAnnotations(); len(ann) > 0 {
		filtered := make(map[string]string, len(ann))
		for k, v := range ann {
			if k == "kubectl.kubernetes.io/last-applied-configuration" {
				continue // shown in the YAML; too long for the drawer, as in Lens
			}
			filtered[k] = v
		}
		if pairs := sortedPairs(filtered); len(pairs) > 0 {
			s.Items = append(s.Items, DetailItem{Name: "Annotations", Value: strings.Join(pairs, "\n")})
		}
	}
	for _, o := range obj.GetOwnerReferences() {
		s.Items = append(s.Items, DetailItem{Name: "Controlled By", Value: o.Kind + " " + o.Name})
	}
	if uid := obj.GetUID(); uid != "" {
		s.Items = append(s.Items, DetailItem{Name: "UID", Value: string(uid)})
	}
	return s
}

func sortedPairs(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}
