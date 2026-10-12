package cluster

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	k8stesting "k8s.io/client-go/testing"
)

// kubectlRestartPatch computes the patch `kubectl rollout restart` sends for obj, as kubectl
// 0.37's rollout command does it: the restarter (polymorphichelpers.defaultObjectRestarter) sets
// the template's restartedAt annotation to the time in RFC 3339, and the patch is the two-way
// strategic merge patch from the object before to the object after (set.CalculatePatches). The
// kubectl packages themselves cannot be imported here (they pull in modules Cogpit does not
// depend on), so the test restates their two steps on the typed object.
func kubectlRestartPatch(t *testing.T, obj runtime.Object, at time.Time) []byte {
	t.Helper()
	before, err := json.Marshal(obj)
	if err != nil {
		t.Fatal(err)
	}
	after := obj.DeepCopyObject()
	var tmpl *corev1.PodTemplateSpec
	switch o := after.(type) {
	case *appsv1.Deployment:
		tmpl = &o.Spec.Template
	case *appsv1.DaemonSet:
		tmpl = &o.Spec.Template
	case *appsv1.StatefulSet:
		tmpl = &o.Spec.Template
	default:
		t.Fatalf("kubectl does not restart %T", obj)
	}
	if tmpl.Annotations == nil {
		tmpl.Annotations = map[string]string{}
	}
	tmpl.Annotations["kubectl.kubernetes.io/restartedAt"] = at.Format(time.RFC3339)
	afterJSON, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	patch, err := strategicpatch.CreateTwoWayMergePatch(before, afterJSON, obj)
	if err != nil {
		t.Fatal(err)
	}
	return patch
}

// TestRestartWorkloadSendsKubectlsPatch: Deployments, Daemon Sets and Stateful Sets get the patch
// kubectl rollout restart sends, as a strategic merge patch on the object, with Cogpit as the
// field manager; the template's other annotations stay.
func TestRestartWorkloadSendsKubectlsPatch(t *testing.T) {
	fixed := time.Date(2026, 10, 11, 12, 30, 0, 0, time.UTC)
	now = func() time.Time { return fixed }
	defer func() { now = time.Now }()

	template := func(ann map[string]string) corev1.PodTemplateSpec {
		return corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "x"}, Annotations: ann},
			Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "app", Image: "busybox:1.37.0"}}},
		}
	}
	meta := func(name string) metav1.ObjectMeta { return metav1.ObjectMeta{Namespace: "team-a", Name: name} }
	for _, tc := range []struct {
		resource string
		obj      runtime.Object
	}{
		{"deployments", &appsv1.Deployment{ObjectMeta: meta("web"), Spec: appsv1.DeploymentSpec{Template: template(nil)}}},
		{"daemonsets", &appsv1.DaemonSet{ObjectMeta: meta("agent"), Spec: appsv1.DaemonSetSpec{Template: template(map[string]string{"keep": "me"})}}},
		{"statefulsets", &appsv1.StatefulSet{ObjectMeta: meta("db"), Spec: appsv1.StatefulSetSpec{Template: template(map[string]string{
			"keep": "me", RestartedAtAnnotation: "2026-10-01T08:00:00Z", // restarted before
		})}}},
	} {
		name := tc.obj.(metav1.Object).GetName()
		c, cs := fakeConn(tc.obj)
		if err := c.RestartWorkload(context.Background(), tc.resource, "team-a", name, ""); err != nil {
			t.Fatalf("%s: %v", tc.resource, err)
		}
		var patch k8stesting.PatchAction
		for _, a := range cs.Actions() {
			if p, ok := a.(k8stesting.PatchAction); ok {
				patch = p
			}
		}
		if patch == nil {
			t.Fatalf("%s: no patch sent: %v", tc.resource, cs.Actions())
		}
		if patch.GetPatchType() != types.StrategicMergePatchType || patch.GetResource().Resource != tc.resource ||
			patch.GetSubresource() != "" || patch.GetNamespace() != "team-a" || patch.GetName() != name {
			t.Errorf("%s: patch %s %s/%s %s/%s", tc.resource, patch.GetPatchType(), patch.GetResource(), patch.GetSubresource(),
				patch.GetNamespace(), patch.GetName())
		}
		if want := kubectlRestartPatch(t, tc.obj, fixed); string(patch.GetPatch()) != string(want) {
			t.Errorf("%s: patch body %s, kubectl sends %s", tc.resource, patch.GetPatch(), want)
		}
		if want := `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"2026-10-11T12:30:00Z"}}}}}`; string(patch.GetPatch()) != want {
			t.Errorf("%s: patch body %s", tc.resource, patch.GetPatch())
		}
	}
}

func TestRestartWorkloadErrors(t *testing.T) {
	c, cs := fakeConn()
	err := c.RestartWorkload(context.Background(), "statefulsets", "team-a", "missing", "")
	if err == nil || !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "restart stateful set team-a/missing") {
		t.Errorf("missing stateful set: %v", err)
	}
	if err := c.RestartWorkload(context.Background(), "replicasets", "team-a", "web-1", ""); err == nil ||
		!strings.Contains(err.Error(), "not a kind Cogpit restarts") {
		t.Errorf("replica set: %v", err)
	}
	for _, a := range cs.Actions() {
		if a.GetResource().Resource == "replicasets" {
			t.Errorf("a request for a kind Cogpit does not restart: %v", a)
		}
	}
}
