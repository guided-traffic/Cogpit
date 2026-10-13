package model

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// Redacted is what a masked Secret value reads as.
const Redacted = "<redacted>"

// lastApplied is where kubectl apply keeps the manifest it applied, values included.
const lastApplied = "kubectl.kubernetes.io/last-applied-configuration"

// Redact returns a copy of obj that is safe to show or hand to an agent: for a Secret, every value
// under data and stringData is replaced with [Redacted], and so is the last-applied-configuration
// annotation, which carries the values too. A Helm release record kept in a ConfigMap (a v1
// ConfigMap labelled owner=helm, HELM_DRIVER=configmap) holds the release's values in its
// data.release as a Secret's would: that entry and the annotation are redacted (IsHelmRecord).
// Other kinds are returned unchanged (not copied).
func Redact(obj *unstructured.Unstructured) *unstructured.Unstructured {
	if obj == nil || obj.GetAPIVersion() != "v1" {
		return obj
	}
	switch {
	case obj.GetKind() == "Secret":
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
		redactLastApplied(out)
		return out
	case IsHelmRecord(obj):
		out := obj.DeepCopy()
		if m, ok := out.Object["data"].(map[string]any); ok {
			if _, found := m[HelmRecordKey]; found {
				m[HelmRecordKey] = Redacted
			}
		}
		redactLastApplied(out)
		return out
	}
	return obj
}

// HelmRecordKey is the data key of a Helm release record: the release, its values included, in
// Helm's encoding.
const HelmRecordKey = "release"

// IsHelmRecord reports whether obj is a Helm release record kept in a ConfigMap: a v1 ConfigMap
// labelled owner=helm, as Helm's ConfigMaps driver writes one per release revision. (A release
// Secret is a Secret, redacted as every Secret is.)
func IsHelmRecord(obj *unstructured.Unstructured) bool {
	return obj != nil && obj.GetKind() == "ConfigMap" && obj.GetAPIVersion() == "v1" && obj.GetLabels()["owner"] == "helm"
}

func redactLastApplied(out *unstructured.Unstructured) {
	if ann := out.GetAnnotations(); ann != nil {
		if _, ok := ann[lastApplied]; ok {
			ann[lastApplied] = Redacted
			out.SetAnnotations(ann)
		}
	}
}
