package engine

import (
	"context"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// NodeActions are the cluster writes of the node commands: cordon, uncordon and drain.
// *cluster.Conn implements them; commands reach them with a type assertion on Backend.Writer(),
// and tests substitute a fake as Backend.Actions.
type NodeActions interface {
	// SetNodeUnschedulable cordons (true) or uncordons (false) a node: patch nodes.
	SetNodeUnschedulable(ctx context.Context, name string, unschedulable bool) error
	// DrainNode cordons a node and evicts its pods as kubectl drain does, reporting progress.
	DrainNode(ctx context.Context, name string, opts cluster.DrainOptions, progress func(cluster.DrainProgress)) error
}

// WorkloadActions are the cluster writes of the workload commands: scale, trigger, suspend and
// resume, evict. *cluster.Conn implements them.
type WorkloadActions interface {
	// ScaleWorkload patches the scale subresource of "deployments", "statefulsets" or
	// "replicasets"; a non-empty uid must match the object's (cluster.ErrReplaced otherwise).
	ScaleWorkload(ctx context.Context, resource, namespace, name, uid string, replicas int32) error
	// TriggerCronJob creates a Job from the cached CronJob, as kubectl create job --from does, and
	// returns its name.
	TriggerCronJob(ctx context.Context, cronJob *unstructured.Unstructured) (string, error)
	// SetCronJobSuspend patches spec.suspend.
	SetCronJobSuspend(ctx context.Context, namespace, name string, suspend bool) error
	// EvictPod evicts a pod through the eviction API; a non-empty uid must match the pod's.
	EvictPod(ctx context.Context, namespace, name, uid string) error
}

var (
	_ NodeActions     = (*cluster.Conn)(nil)
	_ WorkloadActions = (*cluster.Conn)(nil)
)
