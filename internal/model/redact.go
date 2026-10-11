package model

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// Redacted is what a masked Secret value reads as.
const Redacted = "<redacted>"

// Redact returns a copy of obj that is safe to show or hand to an agent: for a Secret, every value
// under data and stringData is replaced with [Redacted], and so is the last-applied-configuration
// annotation, which carries the values too. Other kinds are returned unchanged (not copied).
func Redact(obj *unstructured.Unstructured) *unstructured.Unstructured {
	if obj == nil || obj.GetKind() != "Secret" || obj.GetAPIVersion() != "v1" {
		return obj
	}
	out := obj.DeepCopy()
	for _, field := range []string{"data", "stringData"} {
		m, ok := out.Object[field].(map[string]any)
		if !ok {
			continue
		}
		for k := range m {
			m[k] = Redacted
		}
	}
	if ann := out.GetAnnotations(); ann != nil {
		if _, ok := ann["kubectl.kubernetes.io/last-applied-configuration"]; ok {
			ann["kubectl.kubernetes.io/last-applied-configuration"] = Redacted
			out.SetAnnotations(ann)
		}
	}
	return out
}
