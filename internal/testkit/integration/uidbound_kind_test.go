//go:build kind

package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestKindPatchesBoundToTheUID: Restart's and Scale's patches carry the UID Cogpit meant, and the
// API server refuses them on an object created again under the name (cluster.ErrReplaced): the
// pod template patch by metadata.uid's immutability, the scale subresource by its UID
// precondition. The workloads have no replicas, so no pod is scheduled; namespace cogpit-e.
func TestKindPatchesBoundToTheUID(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	t.Logf("Kubernetes %s", conn.ServerVersion())
	kube, ns := conn.Kube(), workloadNamespace
	freshNamespace(ctx, t, kube, ns)

	selector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "uid"}}
	create := func() (deploy, sts string) {
		t.Helper()
		d, err := kube.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "uid"},
			Spec:       appsv1.DeploymentSpec{Replicas: int32p(0), Selector: selector, Template: podTemplate("uid", nil)},
		}, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		s, err := kube.AppsV1().StatefulSets(ns).Create(ctx, &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "uid"},
			Spec:       appsv1.StatefulSetSpec{Replicas: int32p(0), ServiceName: "uid", Selector: selector, Template: podTemplate("uid", nil)},
		}, metav1.CreateOptions{})
		if err != nil {
			t.Fatal(err)
		}
		return string(d.UID), string(s.UID)
	}
	oldDeploy, oldSts := create()

	// The UID they have: both writes go through.
	if err := conn.RestartWorkload(ctx, "deployments", ns, "uid", oldDeploy); err != nil {
		t.Fatalf("restart with the object's UID: %v", err)
	}
	if err := conn.ScaleWorkload(ctx, "statefulsets", ns, "uid", oldSts, 0); err != nil {
		t.Fatalf("scale with the object's UID: %v", err)
	}

	// Deleted and created again under the names.
	for _, del := range []func() error{
		func() error { return kube.AppsV1().Deployments(ns).Delete(ctx, "uid", metav1.DeleteOptions{}) },
		func() error { return kube.AppsV1().StatefulSets(ns).Delete(ctx, "uid", metav1.DeleteOptions{}) },
	} {
		if err := del(); err != nil {
			t.Fatal(err)
		}
	}
	waitGone(ctx, t, kube, ns)
	newDeploy, newSts := create()

	for name, err := range map[string]error{
		"restart deployments":  conn.RestartWorkload(ctx, "deployments", ns, "uid", oldDeploy),
		"restart statefulsets": conn.RestartWorkload(ctx, "statefulsets", ns, "uid", oldSts),
		"scale deployments":    conn.ScaleWorkload(ctx, "deployments", ns, "uid", oldDeploy, 3),
		"scale statefulsets":   conn.ScaleWorkload(ctx, "statefulsets", ns, "uid", oldSts, 3),
	} {
		if !errors.Is(err, cluster.ErrReplaced) {
			t.Errorf("%s with the old UID: %v, want ErrReplaced", name, err)
		} else {
			t.Logf("%s with the old UID: %v", name, err)
		}
	}
	d, err := kube.AppsV1().Deployments(ns).Get(ctx, "uid", metav1.GetOptions{})
	if err != nil || *d.Spec.Replicas != 0 || d.Spec.Template.Annotations[cluster.RestartedAtAnnotation] != "" {
		t.Errorf("the new deployment was changed: %v", err)
	}
	if err := conn.ScaleWorkload(ctx, "deployments", ns, "uid", newDeploy, 0); err != nil {
		t.Errorf("scale with the new UID: %v", err)
	}
	if err := conn.RestartWorkload(ctx, "statefulsets", ns, "uid", newSts); err != nil {
		t.Errorf("restart with the new UID: %v", err)
	}
}

// waitGone waits until the uid workloads are gone.
func waitGone(ctx context.Context, t *testing.T, kube kubernetes.Interface, ns string) {
	t.Helper()
	if !testkit.WaitFor(ctx, func() bool {
		_, e1 := kube.AppsV1().Deployments(ns).Get(ctx, "uid", metav1.GetOptions{})
		_, e2 := kube.AppsV1().StatefulSets(ns).Get(ctx, "uid", metav1.GetOptions{})
		return apierrors.IsNotFound(e1) && apierrors.IsNotFound(e2)
	}) {
		t.Fatal("the deleted workloads did not go")
	}
}
