package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Errors of the cache reads. Match them with errors.Is.
var (
	ErrNotFound     = errors.New("not found in the cache")
	ErrUnknownKind  = errors.New("unknown kind")
	ErrNotSynced    = errors.New("the cache has not finished listing")
	ErrWrongCluster = errors.New("the reference names another cluster than the connected one")
)

// syncTimeout bounds how long a read waits for an informer's initial list.
const syncTimeout = 15 * time.Second

// Kind returns the connected backend and its profile for a kind key, built-in or custom. The error
// names the served kinds, custom ones included, so an agent can correct itself.
func (s *Session) Kind(kind string) (*Backend, *model.KindProfile, error) {
	b := s.Backend()
	if b == nil {
		return nil, nil, ErrNotConnected
	}
	p, ok := b.Profile(kind)
	if !ok {
		all := b.AllProfiles()
		keys := make([]string, 0, len(all))
		for _, bp := range all {
			keys = append(keys, bp.Key)
		}
		return nil, nil, fmt.Errorf("%w %q; this cluster serves: %s%s", ErrUnknownKind, kind, strings.Join(keys, ", "), b.customKindsNote(kind))
	}
	return b, p, nil
}

// ResolveRef checks a reference against the connected cluster and normalises it: the kind must be
// served, a context must be empty or the connected one (and is filled in), a namespaced kind needs
// a namespace, and a cluster-scoped kind's namespace is dropped.
func (s *Session) ResolveRef(ref model.ObjectRef) (model.ObjectRef, *model.KindProfile, error) {
	_, ref, p, err := s.resolve(ref)
	return ref, p, err
}

func (s *Session) resolve(ref model.ObjectRef) (*Backend, model.ObjectRef, *model.KindProfile, error) {
	b, p, err := s.Kind(ref.Kind)
	if err != nil {
		return nil, ref, nil, err
	}
	if ref.Context != "" && ref.Context != b.Context {
		return nil, ref, nil, fmt.Errorf("%w: %s is in context %q, connected is %q", ErrWrongCluster, ref.String(), ref.Context, b.Context)
	}
	if ref.Name == "" {
		return nil, ref, nil, errors.New("the reference has no name")
	}
	ref.Context = b.Context
	if !p.Namespaced {
		ref.Namespace = ""
	} else if ref.Namespace == "" {
		return nil, ref, nil, fmt.Errorf("%s are namespaced: the reference needs a namespace", p.Key)
	}
	return b, ref, p, nil
}

// waitSynced waits, bounded by ctx and syncTimeout, for a handle's initial list.
func waitSynced(ctx context.Context, h *cache.Handle, kind string) error {
	wctx, cancel := context.WithTimeout(ctx, syncTimeout)
	defer cancel()
	if h.WaitForSync(wctx) {
		return nil
	}
	if err := h.Err(); err != nil {
		return fmt.Errorf("%w %s: %v", ErrNotSynced, kind, err)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%w %s yet", ErrNotSynced, kind)
}

// Objects returns the cached objects of a kind, in the given namespaces (empty: all), acquiring a
// cache handle for the call and waiting for the initial list (bounded by ctx and 15 seconds). The
// handle is released afterwards and the informer stays warm. Objects are shared and must not be
// modified.
func (s *Session) Objects(ctx context.Context, kind string, namespaces []string) ([]*unstructured.Unstructured, error) {
	b, p, err := s.Kind(kind)
	if err != nil {
		return nil, err
	}
	// In all namespaces, or in each the user named; a refusal in all namespaces is followed
	// (accessiblens_plan.go readFollowing).
	h, err := s.readFollowing(ctx, b, p, kind, func(ctx context.Context) *cache.Handle { return b.acquireSettled(ctx, p) })
	if err != nil {
		return nil, err
	}
	defer h.Release()
	objs := KindList(h, p)
	if !p.Namespaced || len(namespaces) == 0 {
		return objs, nil
	}
	out := objs[:0]
	for _, o := range objs {
		if slices.Contains(namespaces, o.GetNamespace()) {
			out = append(out, o)
		}
	}
	return out, nil
}

// Object returns one cached object, waiting for its kind's initial list like Objects. A missing
// object returns ErrNotFound.
func (s *Session) Object(ctx context.Context, ref model.ObjectRef) (*unstructured.Unstructured, error) {
	b, ref, p, err := s.resolve(ref)
	if err != nil {
		return nil, err
	}
	// The object's namespace alone for a user who may not list everywhere, also after a refusal
	// in all namespaces (accessiblens_plan.go readFollowing).
	h, err := s.readFollowing(ctx, b, p, p.Key, b.inNamespace(p, ref.Namespace))
	if err != nil {
		return nil, err
	}
	defer h.Release()
	obj, ok := KindGet(h, p, ref.Namespace, ref.Name)
	if !ok {
		if err := refusedIn(h, p.Key); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%s: %w", ref.String(), ErrNotFound)
	}
	return obj, nil
}

// Lookup returns one cached object without waiting: when the kind is still being listed it
// returns ErrNotSynced. Enabled checks use it, which must not block.
func (s *Session) Lookup(ref model.ObjectRef) (*unstructured.Unstructured, error) {
	b, ref, p, err := s.resolve(ref)
	if err != nil {
		return nil, err
	}
	h := b.Acquire(p)
	defer h.Release()
	if !h.HasSynced() {
		return nil, fmt.Errorf("%w %s yet", ErrNotSynced, p.Key)
	}
	obj, ok := KindGet(h, p, ref.Namespace, ref.Name)
	if !ok {
		return nil, fmt.Errorf("%s: %w", ref.String(), ErrNotFound)
	}
	return obj, nil
}
