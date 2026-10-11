package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Resources RestartWorkload restarts.
const (
	resourceDeployments  = "deployments"
	resourceDaemonSets   = "daemonsets"
	resourceStatefulSets = "statefulsets"
)

// RestartWorkload restarts a Deployment, DaemonSet or StatefulSet (resource is "deployments",
// "daemonsets" or "statefulsets") the way `kubectl rollout restart` does: a strategic merge patch
// that sets the pod template's restartedAt annotation to the current time, which changes the
// template, so that the controller replaces every pod by its update strategy. Like kubectl, it
// does not wait for the rollout. A StatefulSet or DaemonSet with the OnDelete strategy takes the
// patch but replaces a pod only when it is deleted.
func (c *Conn) RestartWorkload(ctx context.Context, resource, namespace, name string) error {
	patch, err := restartPatch(now())
	if err != nil {
		return fmt.Errorf("restart %s %s/%s: %w", resource, namespace, name, err)
	}
	opts := metav1.PatchOptions{FieldManager: FieldManager}
	var noun string
	switch resource {
	case resourceDeployments:
		noun = "deployment"
		_, err = c.kube.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, opts)
	case resourceDaemonSets:
		noun = "daemon set"
		_, err = c.kube.AppsV1().DaemonSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, opts)
	case resourceStatefulSets:
		noun = "stateful set"
		_, err = c.kube.AppsV1().StatefulSets(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch, opts)
	default:
		return fmt.Errorf("restart %s %s/%s: not a kind Cogpit restarts", resource, namespace, name)
	}
	if err != nil {
		return fmt.Errorf("restart %s %s/%s: %w", noun, namespace, name, err)
	}
	return nil
}

// restartPatch is the patch kubectl rollout restart computes: the two-way merge of the object
// before and after setting the template's restartedAt annotation, which is that annotation alone.
func restartPatch(at time.Time) ([]byte, error) {
	return json.Marshal(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{RestartedAtAnnotation: at.Format(time.RFC3339)},
				},
			},
		},
	})
}
