package engine

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// WorkloadRestarts are the cluster writes of the Restart commands of Daemon Sets and Stateful
// Sets. *cluster.Conn implements them; commands reach them with a type assertion on
// Backend.Writer(), and tests substitute a fake as Backend.Actions.
type WorkloadRestarts interface {
	// RestartWorkload restarts a "deployments", "daemonsets" or "statefulsets" object as
	// `kubectl rollout restart` does: patch of the pod template's restartedAt annotation. A
	// non-empty uid must match the object's (cluster.ErrReplaced otherwise).
	RestartWorkload(ctx context.Context, resource, namespace, name, uid string) error
}

var _ WorkloadRestarts = (*cluster.Conn)(nil)
