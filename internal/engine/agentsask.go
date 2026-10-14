package engine

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Role bindings an agent asks for (H-27, owner 2026-10-14): a RoleBinding or ClusterRoleBinding
// that an agent creates or changes always waits for the user's acceptance, also in a context where
// the user allowed agent writes (D-2, H-14), whatever role it names. A binding hands on a role,
// and what the role allows is not something the allowance of "scale, cordon, apply" was given
// for. Commands declare it with Spec.AgentsAsk, judged per invocation and only for agents:
// roleBinding.create always, object.apply when its Edit tab edits a binding or its text holds one,
// object.create when its text holds one. clusterRoleBinding.create is destructive and asks every
// origin anyway. The GUI's own commands are unchanged: the user is the one asking.

// Kubernetes' RBAC API group and the binding kinds and resources it serves.
const (
	rbacGroup                   = "rbac.authorization.k8s.io"
	kindRoleBinding             = "RoleBinding"
	kindClusterRoleBinding      = "ClusterRoleBinding"
	resourceRoleBindings        = "rolebindings"
	resourceClusterRoleBindings = "clusterrolebindings"
)

// BindingKind reports whether apiVersion and kind name a RoleBinding or ClusterRoleBinding of
// Kubernetes' RBAC group. It matches without regard to case, so that it never judges a binding
// to be something else: an over-match asks once too often, never once too few. An apiVersion
// that does not parse is judged by the kind alone.
func BindingKind(apiVersion, kind string) bool {
	if !strings.EqualFold(kind, kindRoleBinding) && !strings.EqualFold(kind, kindClusterRoleBinding) {
		return false
	}
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return true // ask rather than guess
	}
	return strings.EqualFold(gv.Group, rbacGroup)
}

// bindingResource reports whether gvr is the rolebindings or clusterrolebindings resource.
func bindingResource(gvr schema.GroupVersionResource) bool {
	return gvr.Group == rbacGroup && (gvr.Resource == resourceRoleBindings || gvr.Resource == resourceClusterRoleBindings)
}

// TextHoldsBinding reports whether manifest text holds a role binding or a cluster role binding,
// as a document or as an item of a v1 List. Text that does not parse counts as holding one: the
// command refuses it before anything is sent, so asking costs nothing, and a parser's leniency
// can never let a binding past. Empty text holds none.
func TextHoldsBinding(text string) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	objs, err := ParseManifests(text)
	if err != nil {
		return true
	}
	for _, o := range objs {
		if BindingKind(o.GetAPIVersion(), o.GetKind()) {
			return true
		}
	}
	return false
}

// EditorHoldsBinding reports whether an apply in the editor tab id writes a role binding: the tab
// is an Edit tab on a binding, or text (the command's own yaml parameter) holds one, or text is
// empty and the tab's text holds one. A tab that does not exist holds none: the apply is refused
// before anything is sent.
func (s *Session) EditorHoldsBinding(id, text string) bool {
	if TextHoldsBinding(text) {
		return true
	}
	e, err := s.editorOf(id)
	if err != nil {
		return false
	}
	if e.tab.Type == DockEdit && (bindingResource(e.target.GVR) || BindingKind(e.target.APIVersion, e.target.Kind)) {
		return true
	}
	return strings.TrimSpace(text) == "" && !HelmValuesTab(e.tab.Type) && TextHoldsBinding(e.text)
}
