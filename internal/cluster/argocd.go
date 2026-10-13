package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

// Argo CD's Sync and Refresh (G23) through the Kubernetes API alone: Cogpit never talks to Argo
// CD's own API server. Refresh sets the annotation the application controller watches for and
// removes after refreshing; Sync writes the Application's operation field, which the controller
// picks up and runs as it runs one argocd app sync started (its API server writes the same
// field). Both are JSON merge patches on the Application as Cogpit's field manager, bound to the
// UID the user confirmed (another Application created under the name is refused: metadata.uid is
// immutable, 422 Invalid); Sync is bound to the resourceVersion Cogpit checked as well, so that an
// operation someone started meanwhile is never replaced (409 Conflict, ErrArgoChanged).

// ArgoRefreshAnnotation is the annotation that asks Argo CD's controller to refresh an
// Application: "normal" compares it with Git again, "hard" also discards the cached manifests.
const ArgoRefreshAnnotation = "argocd.argoproj.io/refresh"

// ArgoRefreshNormal and ArgoRefreshHard are the annotation's values.
const (
	ArgoRefreshNormal = "normal"
	ArgoRefreshHard   = "hard"
)

// ErrArgoChanged is returned when a Sync's resourceVersion precondition failed: the Application
// changed after Cogpit checked it (a status update, or an operation someone started). Match it
// with errors.Is; the caller checks the newer state and tries again.
var ErrArgoChanged = errors.New("the application changed since it was checked")

// argoApplications is Argo CD's Application resource at a version.
func argoApplications(version string) schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: "argoproj.io", Version: version, Resource: "applications"}
}

// ArgoRefreshPatch is the merge patch of a Refresh: the annotation with "normal" or "hard", and
// the UID as precondition when known.
func ArgoRefreshPatch(uid string, hard bool) ([]byte, error) {
	mode := ArgoRefreshNormal
	if hard {
		mode = ArgoRefreshHard
	}
	meta := map[string]any{"annotations": map[string]any{ArgoRefreshAnnotation: mode}}
	if uid != "" {
		meta["uid"] = uid
	}
	return json.Marshal(map[string]any{"metadata": meta})
}

// ArgoSyncPatch is the merge patch of a Sync: the operation, the UID and the resourceVersion the
// operation was built from as preconditions. Both are required: a Sync must not replace an
// operation Cogpit has not seen.
func ArgoSyncPatch(uid, resourceVersion string, operation map[string]any) ([]byte, error) {
	if uid == "" || resourceVersion == "" {
		return nil, ErrNoPrecondition
	}
	if operation == nil {
		return nil, errors.New("no operation")
	}
	return json.Marshal(map[string]any{
		"metadata":  map[string]any{"uid": uid, "resourceVersion": resourceVersion},
		"operation": operation,
	})
}

// RefreshArgoApplication asks Argo CD to refresh an Application (served at version) by setting
// its refresh annotation; hard discards the manifest cache too. A non-empty uid must match the
// Application's (ErrReplaced otherwise).
func (c *Conn) RefreshArgoApplication(ctx context.Context, version, namespace, name, uid string, hard bool) error {
	gvr := argoApplications(version)
	patch, err := ArgoRefreshPatch(uid, hard)
	if err != nil {
		return fmt.Errorf("refresh %s: %w", describeObject(gvr, namespace, name), err)
	}
	if _, err := c.dyn.Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, patch,
		metav1.PatchOptions{FieldManager: FieldManager}); err != nil {
		return fmt.Errorf("refresh %s: %w", describeObject(gvr, namespace, name), replacedError(err, uid))
	}
	return nil
}

// SyncArgoApplication starts a sync of an Application (served at version) by writing its
// operation, as Argo CD's API server does for argocd app sync. uid and resourceVersion are the
// Application's as Cogpit checked it: another object under the name is ErrReplaced, a newer
// version of it ErrArgoChanged, and nothing is written in either case.
func (c *Conn) SyncArgoApplication(ctx context.Context, version, namespace, name, uid, resourceVersion string, operation map[string]any) error {
	gvr := argoApplications(version)
	patch, err := ArgoSyncPatch(uid, resourceVersion, operation)
	if err != nil {
		return fmt.Errorf("sync %s: %w", describeObject(gvr, namespace, name), err)
	}
	_, err = c.dyn.Resource(gvr).Namespace(namespace).Patch(ctx, name, types.MergePatchType, patch,
		metav1.PatchOptions{FieldManager: FieldManager})
	switch {
	case err == nil:
		return nil
	case apierrors.IsConflict(err) && !strings.Contains(err.Error(), "UID"):
		return fmt.Errorf("sync %s: %w (%v)", describeObject(gvr, namespace, name), ErrArgoChanged, err)
	}
	return fmt.Errorf("sync %s: %w", describeObject(gvr, namespace, name), replacedError(err, uid))
}
