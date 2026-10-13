package engine

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// FluxActions are the writes of the Flux CD commands (commands/flux.go, D-33): a reconcile request,
// Suspend and Resume, each one JSON merge patch on one Flux object. *cluster.Conn implements them.
type FluxActions interface {
	// PatchFlux sends a Flux request to one object of the kind gvr serves; a non-empty uid must
	// match the object's (cluster.ErrReplaced otherwise).
	PatchFlux(ctx context.Context, gvr schema.GroupVersionResource, namespace, name, uid string, req cluster.FluxRequest) error
}

var _ FluxActions = (*cluster.Conn)(nil)
