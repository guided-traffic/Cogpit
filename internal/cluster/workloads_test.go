package cluster

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k8stesting "k8s.io/client-go/testing"
)

func lastPatch(t *testing.T, actions []k8stesting.Action) k8stesting.PatchAction {
	t.Helper()
	var patch k8stesting.PatchAction
	for _, a := range actions {
		if p, ok := a.(k8stesting.PatchAction); ok {
			patch = p
		}
	}
	if patch == nil {
		t.Fatalf("no patch sent: %v", actions)
	}
	return patch
}

func TestScaleWorkloadPatchesTheScaleSubresource(t *testing.T) {
	three := int32(3)
	c, cs := fakeConn(
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "web"}, Spec: appsv1.DeploymentSpec{Replicas: &three}},
		&appsv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Namespace: "shop", Name: "db"}, Spec: appsv1.StatefulSetSpec{Replicas: &three}},
	)
	ctx := context.Background()
	for _, tc := range []struct {
		resource, name string
		replicas       int32
		body           string
	}{
		{"deployments", "web", 5, `{"spec":{"replicas":5}}`},
		{"statefulsets", "db", 0, `{"spec":{"replicas":0}}`},
	} {
		if err := c.ScaleWorkload(ctx, tc.resource, "shop", tc.name, tc.replicas); err != nil {
			t.Fatalf("scale %s: %v", tc.resource, err)
		}
		p := lastPatch(t, cs.Actions())
		if p.GetResource().Resource != tc.resource || p.GetResource().Group != "apps" || p.GetSubresource() != "scale" ||
			p.GetPatchType() != types.MergePatchType || p.GetNamespace() != "shop" || p.GetName() != tc.name || string(p.GetPatch()) != tc.body {
			t.Errorf("%s: patch %s %s/%s %s/%s %s", tc.resource, p.GetPatchType(), p.GetResource().Resource, p.GetSubresource(), p.GetNamespace(), p.GetName(), p.GetPatch())
		}
	}
	// The fake applies a subresource patch to the object itself, which is enough to see it land.
	d, _ := cs.AppsV1().Deployments("shop").Get(ctx, "web", metav1.GetOptions{})
	s, _ := cs.AppsV1().StatefulSets("shop").Get(ctx, "db", metav1.GetOptions{})
	if *d.Spec.Replicas != 5 || *s.Spec.Replicas != 0 {
		t.Errorf("replicas: deployment %d, statefulset %d", *d.Spec.Replicas, *s.Spec.Replicas)
	}

	if err := c.ScaleWorkload(ctx, "daemonsets", "shop", "x", 1); err == nil || !strings.Contains(err.Error(), "not a kind") {
		t.Errorf("scale a daemonset: %v", err)
	}
	if err := c.ScaleWorkload(ctx, "deployments", "shop", "web", -1); err == nil {
		t.Error("negative replicas accepted")
	}
	if err := c.ScaleWorkload(ctx, "deployments", "shop", "gone", 1); !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "shop/gone") {
		t.Errorf("scale a missing deployment: %v", err)
	}
}

func TestSetCronJobSuspend(t *testing.T) {
	c, cs := fakeConn(&batchv1.CronJob{ObjectMeta: metav1.ObjectMeta{Namespace: "ops", Name: "backup"}})
	ctx := context.Background()
	for _, suspend := range []bool{true, false} {
		if err := c.SetCronJobSuspend(ctx, "ops", "backup", suspend); err != nil {
			t.Fatal(err)
		}
		p := lastPatch(t, cs.Actions())
		want := `{"spec":{"suspend":false}}`
		if suspend {
			want = `{"spec":{"suspend":true}}`
		}
		if p.GetResource().Resource != "cronjobs" || p.GetResource().Group != "batch" || p.GetPatchType() != types.MergePatchType || string(p.GetPatch()) != want {
			t.Errorf("patch %s %s %s", p.GetResource(), p.GetPatchType(), p.GetPatch())
		}
		cj, _ := cs.BatchV1().CronJobs("ops").Get(ctx, "backup", metav1.GetOptions{})
		if cj.Spec.Suspend == nil || *cj.Spec.Suspend != suspend {
			t.Errorf("suspend %v after patch %v", cj.Spec.Suspend, suspend)
		}
	}
	if err := c.SetCronJobSuspend(ctx, "ops", "gone", true); !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "suspend cron job ops/gone") {
		t.Errorf("suspend a missing cron job: %v", err)
	}
}

func cronJob() *batchv1.CronJob {
	backoff := int32(2)
	return &batchv1.CronJob{
		TypeMeta:   metav1.TypeMeta{APIVersion: "batch/v1", Kind: "CronJob"},
		ObjectMeta: metav1.ObjectMeta{Namespace: "ops", Name: "backup", UID: "uid-backup", Labels: map[string]string{"team": "ops"}},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 3 * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      map[string]string{"app": "backup"},
					Annotations: map[string]string{"owner": "ops"},
				},
				Spec: batchv1.JobSpec{
					BackoffLimit: &backoff,
					Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
						RestartPolicy: corev1.RestartPolicyNever,
						Containers:    []corev1.Container{{Name: "backup", Image: "busybox:1.37.0", Command: []string{"true"}}},
					}},
				},
			},
		},
	}
}

func toUnstructured(t *testing.T, obj runtime.Object) *unstructured.Unstructured {
	t.Helper()
	m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		t.Fatal(err)
	}
	return &unstructured.Unstructured{Object: m}
}

func fixedSuffixes(t *testing.T, suffixes ...string) {
	t.Helper()
	old := jobSuffix
	jobSuffix = func() (string, error) {
		if len(suffixes) == 0 {
			return "", errors.New("out of suffixes")
		}
		s := suffixes[0]
		suffixes = suffixes[1:]
		return s, nil
	}
	t.Cleanup(func() { jobSuffix = old })
}

// TestTriggerCronJobCreatesTheJobKubectlWould: the created Job is what `kubectl create job
// --from=cronjob/backup` creates, under a generated name.
func TestTriggerCronJobCreatesTheJobKubectlWould(t *testing.T) {
	fixedSuffixes(t, "x7k2p")
	cj := cronJob()
	c, cs := fakeConn(cj)
	name, err := c.TriggerCronJob(context.Background(), toUnstructured(t, cj))
	if err != nil {
		t.Fatal(err)
	}
	if name != "backup-manual-x7k2p" {
		t.Errorf("job name %q", name)
	}
	var created *batchv1.Job
	for _, a := range cs.Actions() {
		if ca, ok := a.(k8stesting.CreateAction); ok && a.GetResource().Resource == "jobs" {
			created = ca.GetObject().(*batchv1.Job)
			if a.GetNamespace() != "ops" || a.GetSubresource() != "" {
				t.Errorf("create in %q, subresource %q", a.GetNamespace(), a.GetSubresource())
			}
		}
	}
	if created == nil {
		t.Fatal("no job created")
	}
	if !reflect.DeepEqual(created.Labels, map[string]string{"app": "backup"}) {
		t.Errorf("labels %v: the template's, not the cron job's", created.Labels)
	}
	if !reflect.DeepEqual(created.Annotations, map[string]string{"owner": "ops", ManualJobAnnotation: "manual"}) {
		t.Errorf("annotations %v", created.Annotations)
	}
	if len(created.OwnerReferences) != 1 {
		t.Fatalf("owner references %v", created.OwnerReferences)
	}
	ref := created.OwnerReferences[0]
	if ref.APIVersion != "batch/v1" || ref.Kind != "CronJob" || ref.Name != "backup" || ref.UID != "uid-backup" ||
		ref.Controller == nil || !*ref.Controller || ref.BlockOwnerDeletion != nil {
		t.Errorf("owner reference %+v", ref)
	}
	if !reflect.DeepEqual(created.Spec, cj.Spec.JobTemplate.Spec) {
		t.Errorf("spec %+v, want the template's", created.Spec)
	}
}

func TestTriggerCronJobRetriesATakenName(t *testing.T) {
	fixedSuffixes(t, "aaaaa", "bbbbb")
	cj := cronJob()
	taken := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{Namespace: "ops", Name: "backup-manual-aaaaa"}}
	c, _ := fakeConn(cj, taken)
	name, err := c.TriggerCronJob(context.Background(), toUnstructured(t, cj))
	if err != nil || name != "backup-manual-bbbbb" {
		t.Errorf("name %q, err %v", name, err)
	}
}

func TestTriggerCronJobRefusesBadInput(t *testing.T) {
	c, _ := fakeConn()
	if _, err := c.TriggerCronJob(context.Background(), nil); err == nil {
		t.Error("nil accepted")
	}
	noUID := cronJob()
	noUID.UID = ""
	if _, err := c.TriggerCronJob(context.Background(), toUnstructured(t, noUID)); err == nil || !strings.Contains(err.Error(), "UID") {
		t.Errorf("no UID: %v", err)
	}
	bad := &unstructured.Unstructured{Object: map[string]any{"metadata": map[string]any{"name": "x", "namespace": "ops"}, "spec": "not a map"}}
	if _, err := c.TriggerCronJob(context.Background(), bad); err == nil {
		t.Error("malformed cron job accepted")
	}
}

func TestManualJobName(t *testing.T) {
	if got := ManualJobName("backup", "x7k2p"); got != "backup-manual-x7k2p" {
		t.Errorf("short: %q", got)
	}
	long := strings.Repeat("a", 52) // the longest name a CronJob may have
	got := ManualJobName(long, "x7k2p")
	if len(got) != maxJobName || !strings.HasSuffix(got, "x7k2p") || !strings.HasPrefix(got, long) {
		t.Errorf("long: %q (%d)", got, len(got))
	}
	s, err := randomSuffix()
	if err != nil || len(s) != manualSuffixLen || strings.Trim(s, suffixAlphabet) != "" {
		t.Errorf("random suffix %q, %v", s, err)
	}
}

func TestEvictPod(t *testing.T) {
	p := pod("shop", "web-0", "worker-1", "ReplicaSet/web")
	f := newDrainFake(t, p)
	if err := f.conn.EvictPod(context.Background(), "shop", "web-0", "uid-web-0"); err != nil {
		t.Fatal(err)
	}
	if f.exists(t, "shop", "web-0") {
		t.Error("pod still there")
	}
	var ev *policyv1.Eviction
	for _, a := range f.cs.Actions() {
		if ca, ok := a.(k8stesting.CreateAction); ok && a.GetSubresource() == "eviction" {
			ev = ca.GetObject().(*policyv1.Eviction)
			if a.GetResource().Resource != "pods" || a.GetNamespace() != "shop" {
				t.Errorf("eviction on %s in %s", a.GetResource(), a.GetNamespace())
			}
		}
	}
	if ev == nil || ev.Name != "web-0" || ev.Namespace != "shop" || ev.DeleteOptions == nil ||
		ev.DeleteOptions.Preconditions == nil || *ev.DeleteOptions.Preconditions.UID != "uid-web-0" || ev.DeleteOptions.GracePeriodSeconds != nil {
		t.Errorf("eviction %+v", ev)
	}
}

// TestEvictPodBlockedByBudget: a 429 from a PodDisruptionBudget is a clear error naming the
// budget, returned at once: no retry loop.
func TestEvictPodBlockedByBudget(t *testing.T) {
	f := newDrainFake(t, pod("shop", "db-0", "worker-1", "StatefulSet/db"))
	f.setEvict(func(string, string) error { return pdbBlocked("db-pdb") })
	err := f.conn.EvictPod(context.Background(), "shop", "db-0", "")
	var blocked *EvictionBlockedError
	if !errors.As(err, &blocked) || blocked.Budget != "db-pdb" || !apierrors.IsTooManyRequests(err) {
		t.Fatalf("err = %#v", err)
	}
	for _, want := range []string{"evict pod shop/db-0", "PodDisruptionBudget db-pdb", "needs 1 healthy pods"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %v", want, err)
		}
	}
	if n := len(f.evictions()); n != 1 {
		t.Errorf("%d eviction requests, want 1", n)
	}
	if !f.exists(t, "shop", "db-0") {
		t.Error("the blocked pod is gone")
	}
}
