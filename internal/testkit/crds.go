package testkit

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// CRDsGVR is the resource of custom resource definitions.
var CRDsGVR = schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}

// CRDVersion is one version of a definition CRD builds.
type CRDVersion struct {
	Name    string
	Served  bool
	Storage bool
	// Columns are additionalPrinterColumns: {"name": "State", "type": "string", "jsonPath": ".status.state"}.
	Columns []map[string]any
}

// CRD builds an established CustomResourceDefinition named <plural>.<group>, with a permissive
// schema per version. Without versions it serves v1 (the storage version) without printer
// columns.
func CRD(group, plural, kind string, namespaced bool, versions ...CRDVersion) *unstructured.Unstructured {
	if len(versions) == 0 {
		versions = []CRDVersion{{Name: "v1", Served: true, Storage: true}}
	}
	scope := "Cluster"
	if namespaced {
		scope = "Namespaced"
	}
	vs := make([]any, 0, len(versions))
	for _, v := range versions {
		cols := make([]any, 0, len(v.Columns))
		for _, c := range v.Columns {
			cols = append(cols, c)
		}
		entry := map[string]any{
			"name": v.Name, "served": v.Served, "storage": v.Storage,
			"schema": map[string]any{"openAPIV3Schema": map[string]any{
				"type": "object", "x-kubernetes-preserve-unknown-fields": true,
			}},
		}
		if len(cols) > 0 {
			entry["additionalPrinterColumns"] = cols
		}
		vs = append(vs, entry)
	}
	return Object("apiextensions.k8s.io/v1", "CustomResourceDefinition", "", plural+"."+group, map[string]any{
		"spec": map[string]any{
			"group": group, "scope": scope,
			"names":    map[string]any{"plural": plural, "singular": plural, "kind": kind, "listKind": kind + "List"},
			"versions": vs,
		},
		"status": map[string]any{"conditions": []any{
			map[string]any{"type": "NamesAccepted", "status": "True", "reason": "NoConflicts"},
			map[string]any{"type": "Established", "status": "True", "reason": "InitialNamesAccepted"},
		}},
	})
}

// NotEstablished marks a definition built by CRD as not served yet.
func NotEstablished(crd *unstructured.Unstructured) *unstructured.Unstructured {
	crd = crd.DeepCopy()
	crd.Object["status"] = map[string]any{"conditions": []any{
		map[string]any{"type": "Established", "status": "False", "reason": "Installing"},
	}}
	return crd
}
