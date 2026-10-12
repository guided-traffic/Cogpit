package kinds

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The related-objects indexes the built-in profiles declare in Needs (architecture §4.4).
var (
	// PodsByNode indexes pods by spec.nodeName: a node's pods.
	PodsByNode = model.RelatedIndex{Kind: podsResource, Name: "byNode", Keys: podNodeKeys}
	// PodsByClaim indexes pods by the persistent volume claims they mount, as
	// "<namespace>/<claimName>": a claim's pods.
	PodsByClaim = model.RelatedIndex{Kind: podsResource, Name: "byClaim", Keys: podClaimKeys}
	// PodsByPriorityClass indexes pods by spec.priorityClassName: a priority class's pods.
	PodsByPriorityClass = model.RelatedIndex{Kind: podsResource, Name: "byPriorityClass", Keys: podPriorityClassKeys}
	// ClusterRolesByName indexes cluster roles by name; Related.Listed tells a cluster role that
	// does not exist from one not listed yet (rbac.go).
	ClusterRolesByName = model.RelatedIndex{Kind: keyClusterRoles, Name: "byName", Keys: clusterRoleKeys}
	// EndpointSlicesByService indexes endpoint slices by the Service they belong to (their
	// kubernetes.io/service-name label), as "<namespace>/<service>": a Service's slices.
	EndpointSlicesByService = model.RelatedIndex{Kind: keyEndpointSlices, Name: "byService", Keys: sliceServiceKeys}
)

func sliceServiceKeys(slice *unstructured.Unstructured) []string {
	if svc := model.NestedString(slice, "metadata", "labels", serviceNameLabel); svc != "" {
		return []string{slice.GetNamespace() + "/" + svc}
	}
	return nil
}

func podNodeKeys(pod *unstructured.Unstructured) []string {
	if n := model.NestedString(pod, "spec", "nodeName"); n != "" {
		return []string{n}
	}
	return nil
}

func podClaimKeys(pod *unstructured.Unstructured) []string {
	var keys []string
	for _, v := range model.NestedSlice(pod, "spec", "volumes") {
		if c := model.NestedString(v, "persistentVolumeClaim", "claimName"); c != "" {
			keys = append(keys, pod.GetNamespace()+"/"+c)
		}
	}
	return keys
}

// ClaimKey is the key PodsByClaim finds a claim's pods under.
func ClaimKey(namespace, claim string) string { return namespace + "/" + claim }
