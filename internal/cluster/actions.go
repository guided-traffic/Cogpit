package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// RestartedAtAnnotation is the pod-template annotation `kubectl rollout restart` sets; changing it
// changes the template, so the controller rolls out new pods.
const RestartedAtAnnotation = "kubectl.kubernetes.io/restartedAt"

// FieldManager names Cogpit in managedFields for the writes it makes.
const FieldManager = "cogpit"

// now is the clock RestartDeployment stamps; tests replace it.
var now = time.Now

// RestartDeployment restarts a Deployment the way `kubectl rollout restart` does: it patches the
// pod template's restartedAt annotation. Like kubectl, it does not wait for the rollout. A paused
// Deployment takes the patch but rolls out only when resumed; the caller checks that from the
// cache before offering the action.
func (c *Conn) RestartDeployment(ctx context.Context, namespace, name string) error {
	patch, err := json.Marshal(map[string]any{
		"spec": map[string]any{
			"template": map[string]any{
				"metadata": map[string]any{
					"annotations": map[string]string{RestartedAtAnnotation: now().Format(time.RFC3339)},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("restart deployment %s/%s: %w", namespace, name, err)
	}
	_, err = c.kube.AppsV1().Deployments(namespace).Patch(ctx, name, types.StrategicMergePatchType, patch,
		metav1.PatchOptions{FieldManager: FieldManager})
	if err != nil {
		return fmt.Errorf("restart deployment %s/%s: %w", namespace, name, err)
	}
	return nil
}

// DeletePod deletes one pod with the default grace period. A non-empty uid is a precondition: the
// API server refuses the delete if the pod with that name is another object by now (a StatefulSet
// pod replaced while the user confirmed).
func (c *Conn) DeletePod(ctx context.Context, namespace, name, uid string) error {
	opts := metav1.DeleteOptions{}
	if uid != "" {
		u := types.UID(uid)
		opts.Preconditions = &metav1.Preconditions{UID: &u}
	}
	if err := c.kube.CoreV1().Pods(namespace).Delete(ctx, name, opts); err != nil {
		return fmt.Errorf("delete pod %s/%s: %w", namespace, name, err)
	}
	return nil
}
