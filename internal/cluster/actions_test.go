package cluster

import (
	"context"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

// fakeConn is a Conn on client-go's fake clientset, seeded with objs.
func fakeConn(objs ...runtime.Object) (*Conn, *fake.Clientset) {
	cs := fake.NewClientset(objs...)
	return &Conn{contextName: "test", namespace: "default", kube: cs, counter: NewRequestCounter()}, cs
}

func TestRestartDeploymentPatchesRestartedAt(t *testing.T) {
	fixed := time.Date(2026, 10, 11, 12, 30, 0, 0, time.UTC)
	now = func() time.Time { return fixed }
	defer func() { now = time.Now }()

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "web"},
		Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{"keep": "me"}},
		}},
	}
	c, cs := fakeConn(dep)
	ctx := context.Background()
	if err := c.RestartDeployment(ctx, "team-a", "web"); err != nil {
		t.Fatalf("RestartDeployment: %v", err)
	}

	var patch k8stesting.PatchAction
	for _, a := range cs.Actions() {
		if p, ok := a.(k8stesting.PatchAction); ok {
			patch = p
		}
	}
	if patch == nil {
		t.Fatalf("no patch sent: %v", cs.Actions())
	}
	if patch.GetPatchType() != types.StrategicMergePatchType || patch.GetResource().Resource != "deployments" ||
		patch.GetNamespace() != "team-a" || patch.GetName() != "web" {
		t.Errorf("patch %s %s %s/%s", patch.GetPatchType(), patch.GetResource(), patch.GetNamespace(), patch.GetName())
	}
	want := `{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"2026-10-11T12:30:00Z"}}}}}`
	if string(patch.GetPatch()) != want {
		t.Errorf("patch body %s, want %s", patch.GetPatch(), want)
	}

	got, err := cs.AppsV1().Deployments("team-a").Get(ctx, "web", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	ann := got.Spec.Template.Annotations
	if ann[RestartedAtAnnotation] != "2026-10-11T12:30:00Z" || ann["keep"] != "me" {
		t.Errorf("template annotations after restart: %v", ann)
	}
}

func TestRestartDeploymentErrorsNameTheObject(t *testing.T) {
	c, _ := fakeConn()
	err := c.RestartDeployment(context.Background(), "team-a", "missing")
	if err == nil || !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "team-a/missing") {
		t.Errorf("err = %v", err)
	}
}

func TestDeletePod(t *testing.T) {
	pod := &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "web-0"}}
	c, cs := fakeConn(pod)
	ctx := context.Background()
	if err := c.DeletePod(ctx, "team-a", "web-0"); err != nil {
		t.Fatalf("DeletePod: %v", err)
	}
	if _, err := cs.CoreV1().Pods("team-a").Get(ctx, "web-0", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Errorf("pod still there: %v", err)
	}
	var del k8stesting.DeleteAction
	for _, a := range cs.Actions() {
		if d, ok := a.(k8stesting.DeleteAction); ok {
			del = d
		}
	}
	if del == nil || del.GetDeleteOptions().GracePeriodSeconds != nil || del.GetDeleteOptions().PropagationPolicy != nil {
		t.Errorf("delete action %+v: want default options", del)
	}

	err := c.DeletePod(ctx, "team-a", "web-0")
	if !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "team-a/web-0") {
		t.Errorf("second delete: %v", err)
	}
}
