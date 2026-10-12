package cluster

import (
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// ErrReplaced is returned when a write named the UID of the object it meant and the API server
// holds another object under the name: it was deleted and created again since. Match it with
// errors.Is.
var ErrReplaced = errors.New("the object was deleted and created again under its name; nothing was done")

// Patches bound to a UID (restart, scale). kubectl's patches carry no UID; Cogpit adds
// metadata.uid to the body when it knows the UID the user confirmed or saw, and the API server
// refuses the patch on another object: a pod template patch fails validation (metadata.uid is
// immutable, 422 Invalid), and a scale subresource checks it as a precondition (409 Conflict,
// "Precondition failed: UID in precondition"). Verified on Kubernetes 1.36.1
// (TestKindPatchesBoundToTheUID).

// fieldSpec is the spec field of the objects Cogpit patches.
const fieldSpec = "spec"

// uidMeta is the metadata a patch carries for its UID precondition, or nil without one.
func uidMeta(uid string) map[string]any {
	if uid == "" {
		return nil
	}
	return map[string]any{"uid": uid}
}

// replacedError turns the API server's refusal of a UID-bound patch into ErrReplaced; other errors
// pass unchanged.
func replacedError(err error, uid string) error {
	if err == nil || uid == "" {
		return err
	}
	switch {
	case apierrors.IsConflict(err) && strings.Contains(err.Error(), "UID"):
	case apierrors.IsInvalid(err) && invalidField(err, "metadata.uid"):
	default:
		return err
	}
	return fmt.Errorf("%w (%v)", ErrReplaced, err)
}

// invalidField reports whether an Invalid error names field among its causes.
func invalidField(err error, field string) bool {
	var status apierrors.APIStatus
	if !errors.As(err, &status) || status.Status().Details == nil {
		return false
	}
	for _, c := range status.Status().Details.Causes {
		if c.Field == field {
			return true
		}
	}
	return false
}
