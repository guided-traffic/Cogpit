package engine

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Waiting for a Flux controller (A91: Reconcile with Source waits as the flux CLI does, D-33). The
// flux CLI polls the object with a GET every two seconds until its controller has handled the
// request (status.lastHandledReconcileAt is no longer what it was) and kstatus calls it current,
// then reads its Ready condition. Cogpit watches instead (cache first): the object's kind in its
// namespace through the cache, shared with any view of it, and decides at each event. A Flux
// controller copies the request's token, the requestedAt annotation, into
// status.lastHandledReconcileAt when it handles it; Cogpit's token is the time it sent the request
// (RFC 3339 with nanoseconds, cluster.FluxPatch), so a token at or after the time the caller sent
// it is that request's or a later one's, which covers it too. A token that is no time counts as
// the flux CLI counts it: when it differs from what the object held before.

// FluxReconciled is how a Flux object's controller ended a reconcile request.
type FluxReconciled struct {
	// Ready: the Ready condition is True; otherwise the object is not ready (Ready False or Stalled
	// True) and Message says why.
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"` // the Ready condition's message, one line
}

// Errors of WaitFluxReconciled.
var (
	// ErrFluxObjectGone: the object was deleted, or replaced by another under its name, while
	// Cogpit waited.
	ErrFluxObjectGone = errors.New("the object is gone")
	// ErrFluxSuspended: the object was suspended while Cogpit waited; its controller leaves it alone.
	ErrFluxSuspended = errors.New("it was suspended meanwhile, so its controller leaves it alone")
)

// LastHandledReconcile returns a Flux object's status.lastHandledReconcileAt: the token of the
// last reconcile request its controller handled, "" before the first.
func LastHandledReconcile(obj *unstructured.Unstructured) string {
	return model.NestedString(obj, "status", "lastHandledReconcileAt")
}

// WaitFluxReconciled waits, through the cache and bounded by ctx, until the controller of a Flux
// object (uid: the object the request went to) has handled a reconcile request sent at since or
// later (prev: the token the object held before it) and has finished with it: Ready True, or Ready
// False or Stalled, not Reconciling, at the object's generation. While it waits, progress is
// called with the object's state as the Flux views write it ("Reconciling: …") whenever it changes.
// It fails when ctx ends, the object goes or is replaced, or is suspended.
func (s *Session) WaitFluxReconciled(ctx context.Context, ref model.ObjectRef, uid string, since time.Time, prev string, progress func(string)) (FluxReconciled, error) {
	ref, p, err := s.ResolveRef(ref)
	if err != nil {
		return FluxReconciled{}, err
	}
	b := s.Backend()
	if b == nil || b.Context != ref.Context {
		return FluxReconciled{}, fmt.Errorf("%w: the connected cluster changed", ErrWrongCluster)
	}
	h, err := b.readFollowing(ctx, p, p.Title, b.inNamespace(p, ref.Namespace))
	if err != nil {
		return FluxReconciled{}, err
	}
	defer h.Release()
	wake := make(chan struct{}, 1)
	unsub := h.Subscribe(func(cache.Event) { signal(wake) })
	defer unsub()
	last := ""
	for {
		obj, ok := h.Get(ref.Namespace, ref.Name)
		switch {
		case !ok:
			return FluxReconciled{}, fmt.Errorf("%s: %w: deleted while Cogpit waited", ref.String(), ErrFluxObjectGone)
		case uid != "" && string(obj.GetUID()) != uid:
			return FluxReconciled{}, fmt.Errorf("%s: %w: replaced by another object under its name", ref.String(), ErrFluxObjectGone)
		case kinds.IsFluxSuspended(obj):
			return FluxReconciled{}, fmt.Errorf("%s: %w", ref.String(), ErrFluxSuspended)
		}
		state, msg := kinds.FluxSourceStatus(obj) // a HelmChart's too, which has no Flux view
		if fluxHandled(LastHandledReconcile(obj), since, prev) && fluxSettled(obj) {
			switch state {
			case kinds.FluxReady:
				return FluxReconciled{Ready: true, Message: msg}, nil
			case kinds.FluxNotReady:
				return FluxReconciled{Message: msg}, nil
			}
		}
		if now := fluxProgressText(state, msg); now != last && progress != nil {
			last = now
			progress(now)
		}
		select {
		case <-ctx.Done():
			return FluxReconciled{}, ctx.Err()
		case <-wake:
		}
	}
}

// fluxHandled reports whether a handled token answers a request sent at since: a time at or after
// it; a token that is no time, when it differs from prev.
func fluxHandled(token string, since time.Time, prev string) bool {
	if token == "" {
		return false
	}
	if t, err := time.Parse(time.RFC3339Nano, token); err == nil {
		return !t.Before(since.Round(0)) // the wall clock alone: the token has no monotonic reading
	}
	return token != prev
}

// fluxSettled reports whether a Flux object's controller is done with it, as kstatus reads it:
// not Reconciling, and its status observed at the object's generation (Ready False from before a
// reconciliation that is still under way does not count).
func fluxSettled(obj *unstructured.Unstructured) bool {
	for _, c := range model.NestedSlice(obj, "status", "conditions") {
		if model.NestedString(c, "type") == "Reconciling" && model.NestedString(c, "status") == "True" {
			return false
		}
	}
	observed, ok := model.NestedInt64(obj, "status", "observedGeneration")
	return !ok || observed >= obj.GetGeneration()
}

// fluxProgressText writes an object's state while Cogpit waits: "Reconciling: <message>".
func fluxProgressText(state kinds.FluxState, msg string) string {
	text := kinds.FluxStateText(state)
	if msg != "" {
		text += ": " + msg
	}
	return text
}
