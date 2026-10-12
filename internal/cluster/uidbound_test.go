package cluster

import (
	"context"
	"errors"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	k8stesting "k8s.io/client-go/testing"
)

// TestPatchesCarryTheUID: with a UID, Restart's and Scale's patches carry it as metadata.uid
// beside kubectl's body; the API server's refusals of such a patch become ErrReplaced.
func TestPatchesCarryTheUID(t *testing.T) {
	fixed := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC)
	now = func() time.Time { return fixed }
	defer func() { now = time.Now }()
	c, cs := fakeConn(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "web", UID: "uid-web"}},
	)
	ctx := context.Background()
	if err := c.RestartWorkload(ctx, "deployments", "team-a", "web", "uid-web"); err != nil {
		t.Fatal(err)
	}
	if err := c.ScaleWorkload(ctx, "deployments", "team-a", "web", "uid-web", 3); err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, a := range cs.Actions() {
		if p, ok := a.(k8stesting.PatchAction); ok {
			bodies = append(bodies, string(p.GetPatch()))
		}
	}
	want := []string{
		`{"metadata":{"uid":"uid-web"},"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"2026-10-12T09:00:00Z"}}}}}`,
		`{"metadata":{"uid":"uid-web"},"spec":{"replicas":3}}`,
	}
	if len(bodies) != 2 || bodies[0] != want[0] || bodies[1] != want[1] {
		t.Errorf("patches %q, want %q", bodies, want)
	}

	// The API server's answers to a patch on an object created again under the name.
	gr := schema.GroupResource{Group: "apps", Resource: "deployments"}
	conflict := apierrors.NewConflict(gr, "web", errors.New("Precondition failed: UID in precondition: uid-web, UID in object meta: uid-new"))
	invalid := apierrors.NewInvalid(schema.GroupKind{Group: "apps", Kind: "Deployment"}, "web", field.ErrorList{
		field.Invalid(field.NewPath("metadata", "uid"), "uid-web", "field is immutable"),
	})
	other := apierrors.NewConflict(gr, "web", errors.New("the object has been modified"))
	for name, tc := range map[string]struct {
		err      error
		uid      string
		replaced bool
	}{
		"scale precondition":       {conflict, "uid-web", true},
		"template patch":           {invalid, "uid-web", true},
		"another conflict":         {other, "uid-web", false},
		"without a uid":            {conflict, "", false},
		"another invalid field":    {apierrors.NewInvalid(schema.GroupKind{Kind: "Deployment"}, "web", field.ErrorList{field.Invalid(field.NewPath("spec", "replicas"), -1, "must be >= 0")}), "uid-web", false},
		"not found stays notfound": {apierrors.NewNotFound(gr, "web"), "uid-web", false},
	} {
		got := replacedError(tc.err, tc.uid)
		if errors.Is(got, ErrReplaced) != tc.replaced || !errors.Is(got, tc.err) && !tc.replaced {
			t.Errorf("%s: %v", name, got)
		}
	}

	// Through the clientset: a reactor answering as the API server does.
	cs.PrependReactor("patch", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) { return true, nil, conflict })
	if err := c.ScaleWorkload(ctx, "deployments", "team-a", "web", "uid-web", 0); !errors.Is(err, ErrReplaced) {
		t.Errorf("scale of a replaced deployment: %v", err)
	}
}
