package cluster

import "context"

// RestartDeployment restarts a Deployment the way `kubectl rollout restart` does: it patches the
// pod template's restartedAt annotation.
func (c *Conn) RestartDeployment(ctx context.Context, namespace, name string) error {
	_, _, _ = ctx, namespace, name
	return ErrNotImplemented
}

// DeletePod deletes one pod with the default grace period.
func (c *Conn) DeletePod(ctx context.Context, namespace, name string) error {
	_, _, _ = ctx, namespace, name
	return ErrNotImplemented
}
