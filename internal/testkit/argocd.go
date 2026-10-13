package testkit

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Argo CD fixtures (G23): its three definitions as Argo CD names them, served at v1alpha1 with a
// permissive schema (no Argo CD is installed), and objects of its kinds.

// ArgoApplicationsGVR, ArgoAppSetsGVR and ArgoProjectsGVR are Argo CD's resources.
var (
	ArgoApplicationsGVR = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applications"}
	ArgoAppSetsGVR      = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "applicationsets"}
	ArgoProjectsGVR     = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "appprojects"}
)

// ArgoCRDs returns Argo CD's Application, ApplicationSet and AppProject definitions, established.
func ArgoCRDs() []*unstructured.Unstructured {
	v := CRDVersion{Name: "v1alpha1", Served: true, Storage: true}
	return []*unstructured.Unstructured{
		CRD("argoproj.io", "applications", "Application", true, v),
		CRD("argoproj.io", "applicationsets", "ApplicationSet", true, v),
		CRD("argoproj.io", "appprojects", "AppProject", true, v),
	}
}

// ArgoApplication builds an Application of a project from a Git repository at main, with the given
// sync and health status ("" leaves the status out).
func ArgoApplication(namespace, name, sync, health string) *unstructured.Unstructured {
	fields := map[string]any{
		"spec": map[string]any{
			"project": "default",
			"source": map[string]any{
				"repoURL": "https://git.example.com/apps.git", "targetRevision": "main", "path": name,
			},
			"destination": map[string]any{"server": "https://kubernetes.default.svc", "namespace": name},
		},
	}
	if sync != "" || health != "" {
		fields["status"] = map[string]any{
			"sync":   map[string]any{"status": sync, "revision": "4f2c9e1b7a6d5c3b2a1f0e9d8c7b6a5f4e3d2c1b"},
			"health": map[string]any{"status": health},
		}
	}
	return Object("argoproj.io/v1alpha1", "Application", namespace, name, fields)
}

// ArgoRepoSecret builds an Argo CD repository Secret with a URL and a fake password.
func ArgoRepoSecret(namespace, name, url, password string) *unstructured.Unstructured {
	return Secret(namespace, name, map[string]string{"argocd.argoproj.io/secret-type": "repository"},
		map[string]string{"url": url, "type": "git", "project": "default", "username": "deploy-bot", "password": password})
}

// ArgoClusterSecret builds an Argo CD cluster Secret whose config holds a fake bearer token.
func ArgoClusterSecret(namespace, name, server, token string) *unstructured.Unstructured {
	return Secret(namespace, name, map[string]string{"argocd.argoproj.io/secret-type": "cluster"},
		map[string]string{"name": name, "server": server, "config": `{"bearerToken":"` + token + `"}`})
}
