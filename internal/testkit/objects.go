package testkit

import (
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// GVRs of the built-in kinds, for tests.
var (
	PodsGVR        = schema.GroupVersionResource{Version: "v1", Resource: "pods"}
	ServicesGVR    = schema.GroupVersionResource{Version: "v1", Resource: "services"}
	NamespacesGVR  = schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}
	NodesGVR       = schema.GroupVersionResource{Version: "v1", Resource: "nodes"}
	EventsGVR      = schema.GroupVersionResource{Version: "v1", Resource: "events"}
	DeploymentsGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}
)

// Object builds an unstructured object with apiVersion, kind, namespace, name, a UID derived from
// the name and a fixed creation time, then merges fields into it (top-level keys such as "spec").
func Object(apiVersion, kind, namespace, name string, fields map[string]any) *unstructured.Unstructured {
	meta := map[string]any{
		"name":              name,
		"uid":               "uid-" + namespace + "-" + name,
		"creationTimestamp": time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
	}
	if namespace != "" {
		meta["namespace"] = namespace
	}
	obj := map[string]any{"apiVersion": apiVersion, "kind": kind, "metadata": meta}
	for k, v := range fields {
		if k == "metadata" {
			if m, ok := v.(map[string]any); ok {
				for mk, mv := range m {
					meta[mk] = mv
				}
			}
			continue
		}
		obj[k] = v
	}
	return &unstructured.Unstructured{Object: obj}
}

// Pod builds a running pod with one container.
func Pod(namespace, name string) *unstructured.Unstructured {
	return Object("v1", "Pod", namespace, name, map[string]any{
		"spec": map[string]any{
			"nodeName":   "node-1",
			"containers": []any{map[string]any{"name": "app", "image": "registry.example/app:1.0"}},
		},
		"status": map[string]any{
			"phase": "Running",
			"podIP": "10.244.0.10",
			"containerStatuses": []any{map[string]any{
				"name": "app", "ready": true, "restartCount": int64(0),
				"state": map[string]any{"running": map[string]any{}},
			}},
		},
	})
}

// Service builds a ClusterIP service.
func Service(namespace, name, clusterIP string) *unstructured.Unstructured {
	return Object("v1", "Service", namespace, name, map[string]any{
		"spec": map[string]any{
			"type":       "ClusterIP",
			"clusterIP":  clusterIP,
			"clusterIPs": []any{clusterIP},
			"ports":      []any{map[string]any{"port": int64(80), "protocol": "TCP", "targetPort": int64(8080)}},
			"selector":   map[string]any{"app": name},
		},
	})
}

func afterTick() <-chan time.Time { return time.After(5 * time.Millisecond) }
