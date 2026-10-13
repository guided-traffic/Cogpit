package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// The annotations the Flux controllers read (fluxcd/pkg/apis/meta), as the flux CLI sets them:
// a new value of requestedAt asks the object's controller to reconcile it now; forceAt (Helm
// releases) also forces a one-off install or upgrade.
const (
	FluxReconcileAnnotation = "reconcile.fluxcd.io/requestedAt"
	FluxForceAnnotation     = "reconcile.fluxcd.io/forceAt"
)

// FluxRequest is one write of the Flux CD actions (D-33): a reconcile request, a forced one, a
// change of spec.suspend, or both (Resume).
type FluxRequest struct {
	// Reconcile sets reconcile.fluxcd.io/requestedAt to the time of the request.
	Reconcile bool
	// Force also sets reconcile.fluxcd.io/forceAt to the same time (Helm releases only).
	Force bool
	// Suspend, when set, sets spec.suspend to its value.
	Suspend *bool
}

// errEmptyFluxRequest refuses a request that would change nothing.
var errEmptyFluxRequest = errors.New("nothing to change")

// FluxPatch is the JSON merge patch of a Flux request, as `flux reconcile`, `flux suspend` and
// `flux resume` send it: the annotations with the time in RFC 3339 with nanoseconds, and
// spec.suspend; with a uid, also metadata.uid, Cogpit's precondition (the API server refuses the
// patch on another object under the name: metadata.uid is immutable).
func FluxPatch(at time.Time, uid string, req FluxRequest) ([]byte, error) {
	if !req.Reconcile && !req.Force && req.Suspend == nil {
		return nil, errEmptyFluxRequest
	}
	meta := map[string]any{}
	if req.Reconcile || req.Force {
		stamp := at.Format(time.RFC3339Nano)
		ann := map[string]string{FluxReconcileAnnotation: stamp}
		if req.Force {
			ann[FluxForceAnnotation] = stamp
		}
		meta["annotations"] = ann
	}
	if uid != "" {
		meta["uid"] = uid
	}
	patch := map[string]any{}
	if len(meta) > 0 {
		patch["metadata"] = meta
	}
	if req.Suspend != nil {
		patch[fieldSpec] = map[string]any{"suspend": *req.Suspend}
	}
	return json.Marshal(patch)
}

// PatchFlux sends a Flux request to one object of a Flux kind (gvr) with a JSON merge patch through
// the dynamic client, as Cogpit's field manager. A non-empty uid binds it to that object
// (ErrReplaced on another object under the name). Like the flux CLI's commands without --wait, it
// does not wait for the controller.
func (c *Conn) PatchFlux(ctx context.Context, gvr schema.GroupVersionResource, namespace, name, uid string, req FluxRequest) error {
	what := describeObject(gvr, namespace, name)
	patch, err := FluxPatch(now(), uid, req)
	if err != nil {
		return fmt.Errorf("patch %s: %w", what, err)
	}
	if _, err := c.dyn.Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, patch,
		metav1.PatchOptions{FieldManager: FieldManager}); err != nil {
		return fmt.Errorf("patch %s: %w", what, replacedError(err, uid))
	}
	return nil
}
