package model

import "strings"

// ObjectRef names one object in one cluster. Kind is the kind key of a [KindProfile], not the
// Kubernetes Kind ("Pod"), because agents and views address kinds by that key: the plural resource
// name for built-in kinds ("pods", "services"), "<plural>.<group>" for kinds from custom resource
// definitions ("gateways.gateway.networking.k8s.io"). Namespace is empty for cluster-scoped kinds.
type ObjectRef struct {
	Context   string `json:"context,omitempty" jsonschema:"kubeconfig context; empty means the active one"`
	Kind      string `json:"kind" jsonschema:"kind key: the plural resource name for built-in kinds (pods, services), <plural>.<group> for custom resources (gateways.gateway.networking.k8s.io); kinds_list lists them"`
	Namespace string `json:"namespace,omitempty" jsonschema:"namespace; empty for cluster-scoped kinds"`
	Name      string `json:"name" jsonschema:"object name"`
}

// Key returns the object's cache key: "namespace/name", or "name" for cluster-scoped objects.
func (r ObjectRef) Key() string {
	if r.Namespace == "" {
		return r.Name
	}
	return r.Namespace + "/" + r.Name
}

// String returns a human-readable form, "pods/default/web-0".
func (r ObjectRef) String() string {
	var b strings.Builder
	b.WriteString(r.Kind)
	b.WriteByte('/')
	b.WriteString(r.Key())
	return b.String()
}

// Same reports whether two references name the same object. An empty Context matches any
// context, so that a reference without one (from an agent) matches the active cluster's rows.
func (r ObjectRef) Same(o ObjectRef) bool {
	if r.Kind != o.Kind || r.Namespace != o.Namespace || r.Name != o.Name {
		return false
	}
	return r.Context == "" || o.Context == "" || r.Context == o.Context
}
