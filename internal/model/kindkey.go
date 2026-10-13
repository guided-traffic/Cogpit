package model

import "strings"

// builtinKind is a Kubernetes kind the API server serves itself: its kind key (the bare plural
// resource name, docs/developer/architecture.md §4.5) and whether its objects live in a namespace.
type builtinKind struct {
	key        string
	namespaced bool
}

// builtinKinds maps "group/Kind" of the kinds the Kubernetes API server serves itself to their
// kind keys. A reference to a kind (an owner reference, an event's involved object) carries the
// API version and the Kind, not the plural; for built-in kinds the plural is known, for kinds from
// custom resource definitions it is not (it needs discovery), so those are not listed.
var builtinKinds = map[string]builtinKind{
	"/Pod":                                          {"pods", true},
	"/Node":                                         {"nodes", false},
	"/Namespace":                                    {"namespaces", false},
	"/Service":                                      {"services", true},
	"/Endpoints":                                    {"endpoints", true},
	"/ConfigMap":                                    {"configmaps", true},
	"/Secret":                                       {"secrets", true},
	"/ServiceAccount":                               {"serviceaccounts", true},
	"/PersistentVolumeClaim":                        {"persistentvolumeclaims", true},
	"/PersistentVolume":                             {"persistentvolumes", false},
	"/ReplicationController":                        {"replicationcontrollers", true},
	"/ResourceQuota":                                {"resourcequotas", true},
	"/LimitRange":                                   {"limitranges", true},
	"/Event":                                        {"events", true},
	"apps/Deployment":                               {"deployments", true},
	"apps/ReplicaSet":                               {"replicasets", true},
	"apps/StatefulSet":                              {"statefulsets", true},
	"apps/DaemonSet":                                {"daemonsets", true},
	"apps/ControllerRevision":                       {"controllerrevisions", true},
	"batch/Job":                                     {"jobs", true},
	"batch/CronJob":                                 {"cronjobs", true},
	"autoscaling/HorizontalPodAutoscaler":           {"horizontalpodautoscalers", true},
	"policy/PodDisruptionBudget":                    {"poddisruptionbudgets", true},
	"policy/PodSecurityPolicy":                      {"podsecuritypolicies", false}, // policy/v1beta1, clusters before 1.25
	"networking.k8s.io/Ingress":                     {"ingresses", true},
	"networking.k8s.io/IngressClass":                {"ingressclasses", false},
	"networking.k8s.io/NetworkPolicy":               {"networkpolicies", true},
	"discovery.k8s.io/EndpointSlice":                {"endpointslices", true},
	"storage.k8s.io/StorageClass":                   {"storageclasses", false},
	"scheduling.k8s.io/PriorityClass":               {"priorityclasses", false},
	"node.k8s.io/RuntimeClass":                      {"runtimeclasses", false},
	"coordination.k8s.io/Lease":                     {"leases", true},
	"rbac.authorization.k8s.io/Role":                {"roles", true},
	"rbac.authorization.k8s.io/RoleBinding":         {"rolebindings", true},
	"rbac.authorization.k8s.io/ClusterRole":         {"clusterroles", false},
	"rbac.authorization.k8s.io/ClusterRoleBinding":  {"clusterrolebindings", false},
	"apiextensions.k8s.io/CustomResourceDefinition": {"customresourcedefinitions", false},
	"admissionregistration.k8s.io/MutatingWebhookConfiguration":     {"mutatingwebhookconfigurations", false},
	"admissionregistration.k8s.io/ValidatingWebhookConfiguration":   {"validatingwebhookconfigurations", false},
	"admissionregistration.k8s.io/MutatingAdmissionPolicy":          {"mutatingadmissionpolicies", false},
	"admissionregistration.k8s.io/MutatingAdmissionPolicyBinding":   {"mutatingadmissionpolicybindings", false},
	"admissionregistration.k8s.io/ValidatingAdmissionPolicy":        {"validatingadmissionpolicies", false},
	"admissionregistration.k8s.io/ValidatingAdmissionPolicyBinding": {"validatingadmissionpolicybindings", false},
	"resource.k8s.io/ResourceClaim":                                 {"resourceclaims", true},
	"resource.k8s.io/ResourceClaimTemplate":                         {"resourceclaimtemplates", true},
	"resource.k8s.io/DeviceClass":                                   {"deviceclasses", false},
	"resource.k8s.io/ResourceSlice":                                 {"resourceslices", false},
}

// BuiltinKindKey returns the kind key of a built-in Kubernetes kind, given an API version as
// references carry it ("apps/v1", "v1") and the Kind ("ReplicaSet"), and whether the kind is
// namespaced. Kinds from custom resource definitions, and built-in kinds Cogpit does not know,
// return ok false: their plural, and so their key, needs discovery.
// A reference without an API version is looked up by its Kind alone, which is unique among the
// built-in kinds.
func BuiltinKindKey(apiVersion, kind string) (key string, namespaced, ok bool) {
	var k builtinKind
	if apiVersion == "" {
		k, ok = builtinByKind[kind]
	} else {
		group := ""
		if g, _, found := strings.Cut(apiVersion, "/"); found {
			group = g
		}
		k, ok = builtinKinds[group+"/"+kind]
	}
	return k.key, k.namespaced, ok
}

// builtinByKind indexes builtinKinds by Kind alone.
var builtinByKind = func() map[string]builtinKind {
	m := make(map[string]builtinKind, len(builtinKinds))
	for gk, k := range builtinKinds {
		_, kind, _ := strings.Cut(gk, "/")
		m[kind] = k
	}
	return m
}()

// RefTo returns a link to the object a reference names (an owner reference, an event's involved
// object), or nil when the kind is not a known built-in kind or the name is empty. namespace is
// the object's namespace; it is dropped for cluster-scoped kinds.
func RefTo(apiVersion, kind, namespace, name string) *ObjectRef {
	key, namespaced, ok := BuiltinKindKey(apiVersion, kind)
	if !ok || name == "" {
		return nil
	}
	if !namespaced {
		namespace = ""
	}
	return &ObjectRef{Kind: key, Namespace: namespace, Name: name}
}
