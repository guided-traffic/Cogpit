package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
)

// ObjectWriter is the cluster side of editing, creating and deleting objects of any kind (G9):
// mapping kinds to resources through discovery, and the generic writes through the dynamic
// client. *cluster.Conn implements it; tests put a fake in Backend.Actions. The engine reaches it
// with a type assertion on Backend.Writer, so that this area never grows Actions.
type ObjectWriter interface {
	// MapKind maps a manifest's apiVersion and kind to its resource and scope.
	MapKind(ctx context.Context, apiVersion, kind string) (cluster.Mapping, error)
	// MapResource maps a resource and group (empty: any group) to its preferred version, kind and
	// scope.
	MapResource(ctx context.Context, group, resource string) (cluster.Mapping, error)
	// UpdateObject replaces an object; obj carries the resourceVersion precondition.
	UpdateObject(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured, dryRun bool) (*unstructured.Unstructured, error)
	// CreateObject creates obj in namespace (empty for cluster-scoped kinds).
	CreateObject(ctx context.Context, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured, dryRun bool) (*unstructured.Unstructured, error)
	// DeleteObject deletes one object; a non-empty uid is a precondition.
	DeleteObject(ctx context.Context, gvr schema.GroupVersionResource, namespace, name, uid string) error
}

// ErrNoObjectWriter is returned when the connected backend cannot edit, create or delete objects:
// the demo cluster, or a writer that lacks the generic writes.
var ErrNoObjectWriter = errors.New("this cluster connection cannot edit, create or delete objects")

// ObjectWriter returns the backend's writer for the generic object commands.
func (b *Backend) ObjectWriter() (ObjectWriter, error) {
	w := b.Writer()
	if w == nil {
		return nil, ErrNoObjectWriter
	}
	ow, ok := w.(ObjectWriter)
	if !ok {
		return nil, ErrNoObjectWriter
	}
	return ow, nil
}

// ObjectTarget is one object the generic commands act on: its normalised reference and what
// discovery or its kind profile says about its kind.
type ObjectTarget struct {
	Ref        model.ObjectRef             `json:"ref"`
	GVR        schema.GroupVersionResource `json:"gvr"`
	APIVersion string                      `json:"apiVersion"` // "apps/v1"
	Kind       string                      `json:"kind"`       // the Kubernetes kind, "Deployment"
	Namespaced bool                        `json:"namespaced"`
}

// mappingTimeout bounds a discovery lookup made to resolve a kind without a profile.
const mappingTimeout = 15 * time.Second

// ResolveTarget resolves a reference against the connected cluster for the generic commands
// (Edit, Delete). A kind with a profile resolves from the profile; any other kind key ("configmaps",
// "<plural>.<group>") through the writer's discovery mapping, so that agents can edit kinds no view
// shows yet. The reference is normalised as ResolveRef does it.
func (s *Session) ResolveTarget(ctx context.Context, ref model.ObjectRef) (ObjectTarget, *Backend, error) {
	b := s.Backend()
	if b == nil {
		return ObjectTarget{}, nil, ErrNotConnected
	}
	if p, ok := b.Profile(ref.Kind); ok {
		r, _, err := s.ResolveRef(ref)
		if err != nil {
			return ObjectTarget{}, nil, err
		}
		if r.Context != b.Context { // the user switched clusters in between
			return ObjectTarget{}, nil, fmt.Errorf("%w: the connected cluster changed", ErrWrongCluster)
		}
		gv := schema.GroupVersion{Group: p.Group, Version: p.Version}
		return ObjectTarget{Ref: r, GVR: p.GVR(), APIVersion: gv.String(), Kind: p.Kind, Namespaced: p.Namespaced}, b, nil
	}
	if ref.Context != "" && ref.Context != b.Context {
		return ObjectTarget{}, nil, fmt.Errorf("%w: %s is in context %q, connected is %q", ErrWrongCluster, ref.String(), ref.Context, b.Context)
	}
	if ref.Name == "" {
		return ObjectTarget{}, nil, errors.New("the reference has no name")
	}
	w, err := b.ObjectWriter()
	if err != nil {
		return ObjectTarget{}, nil, fmt.Errorf("%w %q", ErrUnknownKind, ref.Kind)
	}
	resource, group := splitKindKey(ref.Kind)
	mctx, cancel := context.WithTimeout(ctx, mappingTimeout)
	defer cancel()
	m, err := w.MapResource(mctx, group, resource)
	if err != nil {
		return ObjectTarget{}, nil, fmt.Errorf("%w %q: %v", ErrUnknownKind, ref.Kind, err)
	}
	if key := KindKeyFor(b, m.GVR); key != ref.Kind {
		return ObjectTarget{}, nil, fmt.Errorf("%w %q: the kind key of %s is %q", ErrUnknownKind, ref.Kind, m.GVR.GroupResource(), key)
	}
	ref.Context = b.Context
	if !m.Namespaced {
		ref.Namespace = ""
	} else if ref.Namespace == "" {
		return ObjectTarget{}, nil, fmt.Errorf("%s are namespaced: the reference needs a namespace", ref.Kind)
	}
	return ObjectTarget{Ref: ref, GVR: m.GVR, APIVersion: m.GVK.GroupVersion().String(), Kind: m.GVK.Kind, Namespaced: m.Namespaced}, b, nil
}

// splitKindKey splits a kind key into its resource and group: "pods" → ("pods", ""),
// "gateways.gateway.networking.k8s.io" → ("gateways", "gateway.networking.k8s.io").
func splitKindKey(key string) (resource, group string) {
	resource, group, _ = strings.Cut(key, ".")
	return resource, group
}

// builtinGroups are the API groups the Kubernetes API server serves itself (Kubernetes 1.36):
// their kinds' keys are the bare plural (architecture §4.5). Groups under k8s.io that come from
// custom resource definitions (gateway.networking.k8s.io, snapshot.storage.k8s.io) are not here.
var builtinGroups = map[string]bool{
	"": true, "admissionregistration.k8s.io": true, "apiextensions.k8s.io": true,
	"apiregistration.k8s.io": true, "apps": true, "authentication.k8s.io": true,
	"authorization.k8s.io": true, "autoscaling": true, "batch": true, "certificates.k8s.io": true,
	"coordination.k8s.io": true, "discovery.k8s.io": true, "events.k8s.io": true,
	"flowcontrol.apiserver.k8s.io": true, "internal.apiserver.k8s.io": true,
	"networking.k8s.io": true, "node.k8s.io": true, "policy": true,
	"rbac.authorization.k8s.io": true, "resource.k8s.io": true, "scheduling.k8s.io": true,
	"storage.k8s.io": true, "storagemigration.k8s.io": true,
}

// KindKeyFor returns the kind key of a resource: the key of the backend's profile that serves it,
// else the bare plural for the API server's own groups and "<plural>.<group>" for every other
// group (architecture §4.5).
func KindKeyFor(b *Backend, gvr schema.GroupVersionResource) string {
	if b != nil {
		for _, p := range b.Profiles {
			if p.Group == gvr.Group && p.Resource == gvr.Resource {
				return p.Key
			}
		}
	}
	if builtinGroups[gvr.Group] {
		return gvr.Resource
	}
	return gvr.Resource + "." + gvr.Group
}

// targetUIDs reads the targets' UIDs from the cache for a destructive command's confirmation,
// in order, "" for an object it cannot resolve or does not hold. The waits are bounded by the
// lookup timeouts; a target's kind is usually warm already, since the user sees it.
func (s *Session) targetUIDs(ctx context.Context, targets []model.ObjectRef) []string {
	uids := make([]string, len(targets))
	for i, ref := range targets {
		t, b, err := s.ResolveTarget(ctx, ref)
		if err != nil {
			continue
		}
		if obj, err := cachedObject(ctx, b, t); err == nil {
			uids[i] = string(obj.GetUID())
		}
	}
	return uids
}

// ActUID returns the UID a destructive write on ref must carry as its precondition: the one the
// user confirmed, else, without a confirmation, the cache's. It fails when the object was not in
// the cache when the user confirmed: whatever has the name now is not what they saw.
func ActUID(ctx context.Context, ref model.ObjectRef, cached func() (string, bool)) (string, error) {
	uid, confirmed := ConfirmedUID(ctx, ref)
	if confirmed {
		if uid == "" {
			return "", fmt.Errorf("%s was not in Cogpit's cache when the action was confirmed; nothing was done", ref.String())
		}
		return uid, nil
	}
	if c, ok := cached(); ok && c != "" {
		return c, nil
	}
	return "", fmt.Errorf("%s: %w", ref.String(), ErrNotFound)
}

// cachedObject returns a target's object as the cache holds it, acquiring a handle on its kind for
// the call and waiting for the initial list like Session.Object. The object is shared and must not
// be modified.
func cachedObject(ctx context.Context, b *Backend, t ObjectTarget) (*unstructured.Unstructured, error) {
	h := b.Cache.Acquire(cache.Scope{GVR: t.GVR})
	defer h.Release()
	if err := waitSynced(ctx, h, t.Ref.Kind); err != nil {
		return nil, err
	}
	obj, ok := h.Get(t.Ref.Namespace, t.Ref.Name)
	if !ok {
		return nil, fmt.Errorf("%s: %w", t.Ref.String(), ErrNotFound)
	}
	return obj, nil
}

// waitForNewer waits, bounded by timeout, until the cache holds the target's object in another
// version than rv (or holds none), and returns what it holds then: after a 409 Conflict the
// watch event with the newer version may still be on its way.
func waitForNewer(ctx context.Context, b *Backend, t ObjectTarget, rv string, timeout time.Duration) (*unstructured.Unstructured, bool) {
	h := b.Cache.Acquire(cache.Scope{GVR: t.GVR})
	defer h.Release()
	wake := make(chan struct{}, 1)
	cancel := h.Subscribe(func(ev cache.Event) {
		if ev.Object != nil && ev.Object.GetName() == t.Ref.Name && ev.Object.GetNamespace() == t.Ref.Namespace {
			signal(wake)
		}
	})
	defer cancel()
	wctx, stop := context.WithTimeout(ctx, timeout)
	defer stop()
	if err := waitSynced(wctx, h, t.Ref.Kind); err != nil {
		return nil, false
	}
	for {
		obj, ok := h.Get(t.Ref.Namespace, t.Ref.Name)
		if !ok {
			return nil, true
		}
		if obj.GetResourceVersion() != rv {
			return obj, true
		}
		select {
		case <-wake:
		case <-wctx.Done():
			return obj, false
		}
	}
}

// DeleteResult is what object.delete reports.
type DeleteResult struct {
	Ref model.ObjectRef `json:"ref"`
	UID string          `json:"uid,omitempty"`
}

// DeleteObject deletes one object of any kind with the UID the cache holds for it as a
// precondition: the object the user saw and confirmed, not one created again under its name.
// The caller has checked the confirmation's context.
func (s *Session) DeleteObject(ctx context.Context, ref model.ObjectRef) (DeleteResult, error) {
	t, b, err := s.ResolveTarget(ctx, ref)
	if err != nil {
		return DeleteResult{}, err
	}
	w, err := b.ObjectWriter()
	if err != nil {
		return DeleteResult{}, err
	}
	uid, err := ActUID(ctx, t.Ref, func() (string, bool) {
		obj, err := cachedObject(ctx, b, t)
		if err != nil {
			return "", false
		}
		return string(obj.GetUID()), true
	})
	if err != nil {
		return DeleteResult{}, err
	}
	if err := w.DeleteObject(ctx, t.GVR, t.Ref.Namespace, t.Ref.Name, uid); err != nil {
		return DeleteResult{}, fmt.Errorf("%s: %w", t.Ref.String(), err)
	}
	return DeleteResult{Ref: t.Ref, UID: uid}, nil
}
