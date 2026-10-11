//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// actionsNamespace is the shared kind cluster's namespace of these tests; they create it, act
// only inside it, and delete it.
const actionsNamespace = "cogpit-actions"

// withActionKinds adds profiles for the kinds the workload actions need but Cogpit ships no
// built-in profile for yet (Stateful Sets, Cron Jobs), when the cluster serves them.
type withActionKinds struct{ engine.Connector }

func (c withActionKinds) Connect(ctx context.Context, name string) (*engine.Backend, error) {
	b, err := c.Connector.Connect(ctx, name)
	if err != nil {
		return nil, err
	}
	for _, p := range []*model.KindProfile{
		actionKindProfile("statefulsets", "apps", "StatefulSet", "Stateful Sets"),
		actionKindProfile("cronjobs", "batch", "CronJob", "Cron Jobs"),
	} {
		if _, ok := b.Profile(p.Key); !ok {
			b.Profiles = append(b.Profiles, p)
		}
	}
	return b, nil
}

func actionKindProfile(key, group, kind, title string) *model.KindProfile {
	cols := []model.Column{{ID: "name", Title: "Name", Type: model.ColumnText}, {ID: "namespace", Title: "Namespace", Type: model.ColumnText}}
	return &model.KindProfile{
		Key: key, Group: group, Version: "v1", Resource: key, Kind: kind, Title: title, Namespaced: true, Columns: cols,
		Cells: func(o *unstructured.Unstructured) ([]model.Cell, model.Status) {
			return []model.Cell{{Text: o.GetName()}, {Text: o.GetNamespace()}}, model.StatusNone
		},
	}
}

// recordingConfirmer accepts every request and remembers it.
type recordingConfirmer struct {
	mu   sync.Mutex
	reqs []engine.ConfirmRequest
}

func (c *recordingConfirmer) Confirm(_ context.Context, req engine.ConfirmRequest) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reqs = append(c.reqs, req)
	return true, nil
}

func (c *recordingConfirmer) last() (engine.ConfirmRequest, int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.reqs) == 0 {
		return engine.ConfirmRequest{}, 0
	}
	return c.reqs[len(c.reqs)-1], len(c.reqs)
}

// sleeper is a small container that ends at once on SIGTERM, from an image the kind node has.
func sleeper(name string) corev1.Container {
	no := false
	return corev1.Container{
		Name: name, Image: "busybox:1.37.0", ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{"sh", "-c", "trap 'exit 0' TERM; while true; do sleep 3600 & wait; done"},
		Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("5m"), corev1.ResourceMemory: resource.MustParse("8Mi"),
		}},
		SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &no, Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
	}
}

func podTemplate(app string, nodeSelector map[string]string) corev1.PodTemplateSpec {
	yes, uid := true, int64(65534)
	grace := int64(5)
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": app}},
		Spec: corev1.PodSpec{
			Containers:                    []corev1.Container{sleeper(app)},
			NodeSelector:                  nodeSelector,
			TerminationGracePeriodSeconds: &grace,
			SecurityContext:               &corev1.PodSecurityContext{RunAsNonRoot: &yes, RunAsUser: &uid},
		},
	}
}

func int32p(n int32) *int32 { return &n }

// freshNamespace creates ns, waiting for a leftover of an earlier run to be gone first, and
// deletes it when the test ends.
func freshNamespace(ctx context.Context, t *testing.T, kube kubernetes.Interface, ns string) {
	t.Helper()
	if !testkit.WaitFor(ctx, func() bool {
		_, err := kube.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return true
		}
		_ = kube.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
		return false
	}) {
		t.Fatalf("namespace %s from an earlier run is still terminating", ns)
	}
	if _, err := kube.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		dctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := kube.CoreV1().Namespaces().Delete(dctx, ns, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
			t.Errorf("delete namespace %s: %v", ns, err)
		}
	})
}

// waitFor polls cond until it holds or the test's context ends.
func waitFor(ctx context.Context, t *testing.T, what string, cond func() bool) {
	t.Helper()
	if !testkit.WaitFor(ctx, cond) {
		t.Fatalf("timed out waiting for %s", what)
	}
}

// readyPods returns the running, ready, not terminating pods with the label app=app.
func readyPods(ctx context.Context, kube kubernetes.Interface, ns, app string) []corev1.Pod {
	list, err := kube.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "app=" + app})
	if err != nil {
		return nil
	}
	var out []corev1.Pod
	for _, p := range list.Items {
		if p.DeletionTimestamp != nil || p.Status.Phase != corev1.PodRunning {
			continue
		}
		for _, c := range p.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status == corev1.ConditionTrue {
				out = append(out, p)
			}
		}
	}
	return out
}

// budgetSettled reports whether a PodDisruptionBudget's status is current and allows want
// disruptions.
func budgetSettled(ctx context.Context, kube kubernetes.Interface, ns, name string, want int32) bool {
	pdb, err := kube.PolicyV1().PodDisruptionBudgets(ns).Get(ctx, name, metav1.GetOptions{})
	return err == nil && pdb.Status.ObservedGeneration >= pdb.Generation && pdb.Status.DisruptionsAllowed == want && pdb.Status.ExpectedPods > 0
}

func guardingBudget(ns, name, app string) *policyv1.PodDisruptionBudget {
	one := intstr.FromInt32(1)
	return &policyv1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name},
		Spec: policyv1.PodDisruptionBudgetSpec{
			MinAvailable: &one, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": app}},
		},
	}
}

type dispatcher struct {
	t   *testing.T
	ctx context.Context
	s   *engine.Session
}

// run dispatches from the GUI and decodes the result through JSON, as a client sees it.
func (d dispatcher) run(id string, params any, out any) error {
	d.t.Helper()
	raw, err := json.Marshal(params)
	if err != nil {
		d.t.Fatal(err)
	}
	res, err := d.s.Dispatch(d.ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginGUI})
	if err == nil && out != nil {
		b, _ := json.Marshal(res)
		if uErr := json.Unmarshal(b, out); uErr != nil {
			d.t.Fatalf("%s: %v", id, uErr)
		}
	}
	return err
}

func (d dispatcher) must(id string, params any, out any) {
	d.t.Helper()
	if err := d.run(id, params, out); err != nil {
		d.t.Fatalf("%s: %v", id, err)
	}
}

// cached waits until the session's cache holds an object that satisfies cond.
func (d dispatcher) cached(ref model.ObjectRef, cond func(*unstructured.Unstructured) bool) {
	d.t.Helper()
	waitFor(d.ctx, d.t, "the cache to show "+ref.String(), func() bool {
		obj, err := d.s.Object(d.ctx, ref)
		return err == nil && (cond == nil || cond(obj))
	})
}

// TestKindNodeAndWorkloadActions runs Scale, Trigger, Suspend and Resume, and Evict through the
// engine against the shared kind cluster, on objects of its own in the namespace cogpit-actions,
// and checks each change on the API server. It never touches the node. Cordon, uncordon and drain
// run against the multi-node drain cluster only (drain_kind_test.go).
func TestKindNodeAndWorkloadActions(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kube, ns := conn.Kube(), actionsNamespace
	freshNamespace(ctx, t, kube, ns)

	if _, err := kube.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "web"},
		Spec: appsv1.DeploymentSpec{Replicas: int32p(1), Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			Template: podTemplate("web", nil)},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.AppsV1().StatefulSets(ns).Create(ctx, &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "db"},
		Spec: appsv1.StatefulSetSpec{Replicas: int32p(1), ServiceName: "db", Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "db"}},
			Template: podTemplate("db", nil)},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	jobPod := podTemplate("tick", nil)
	jobPod.Spec.RestartPolicy = corev1.RestartPolicyNever
	jobPod.Spec.Containers[0].Command = []string{"true"}
	if _, err := kube.BatchV1().CronJobs(ns).Create(ctx, &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "tick"},
		Spec: batchv1.CronJobSpec{
			Schedule: "0 0 1 1 *", // once a year: only the test's trigger runs it
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "tick"}, Annotations: map[string]string{"cogpit.test/from": "template"}},
				Spec:       batchv1.JobSpec{Template: jobPod, BackoffLimit: int32p(0)},
			},
		},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.PolicyV1().PodDisruptionBudgets(ns).Create(ctx, guardingBudget(ns, "web-pdb", "web"), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	confirmer := &recordingConfirmer{}
	s := engine.New(engine.Options{
		Connector: withActionKinds{&engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}},
		Confirmer: confirmer, Drains: engine.NewDrains(engine.NewMemoryDrainJournal()),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	d := dispatcher{t: t, ctx: ctx, s: s}
	web := model.ObjectRef{Kind: "deployments", Namespace: ns, Name: "web"}
	db := model.ObjectRef{Kind: "statefulsets", Namespace: ns, Name: "db"}
	tick := model.ObjectRef{Kind: "cronjobs", Namespace: ns, Name: "tick"}
	d.cached(web, nil)
	d.cached(db, nil)
	d.cached(tick, nil)

	// Scale: through the scale subresource; to zero only after a destructive confirmation.
	replicas := func(get func() (*int32, error)) int32 {
		r, err := get()
		if err != nil || r == nil {
			return -1
		}
		return *r
	}
	deployReplicas := func() int32 {
		return replicas(func() (*int32, error) {
			o, err := kube.AppsV1().Deployments(ns).Get(ctx, "web", metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return o.Spec.Replicas, nil
		})
	}
	setReplicas := func(id string, ref model.ObjectRef, n int, current func() int32) {
		t.Helper()
		d.must(id, map[string]any{"ref": ref, "replicas": n}, nil)
		waitFor(ctx, t, ref.String()+" scaled", func() bool { return current() == int32(n) })
	}
	setReplicas("deployment.scale", web, 2, deployReplicas)
	if _, n := confirmer.last(); n != 0 {
		t.Errorf("scaling up from the GUI asked for a confirmation")
	}
	setReplicas("deployment.scale", web, 0, deployReplicas)
	if req, n := confirmer.last(); n != 1 || req.Danger != engine.DangerDestructive || req.Command != "deployment.scale" {
		t.Errorf("scale to zero: confirmation %+v (%d asked)", req, n)
	}
	setReplicas("deployment.scale", web, 1, deployReplicas)
	stsReplicas := func() int32 {
		return replicas(func() (*int32, error) {
			o, err := kube.AppsV1().StatefulSets(ns).Get(ctx, "db", metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
			return o.Spec.Replicas, nil
		})
	}
	setReplicas("statefulset.scale", db, 2, stsReplicas)
	setReplicas("statefulset.scale", db, 0, stsReplicas)

	// Trigger: the Job kubectl create job --from=cronjob/tick creates.
	var trig commands.TriggerResult
	d.must("cronjob.trigger", map[string]any{"ref": tick}, &trig)
	job, err := kube.BatchV1().Jobs(ns).Get(ctx, trig.Job.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("the triggered job %q: %v", trig.Job.Name, err)
	}
	owner := metav1.GetControllerOf(job)
	if !strings.HasPrefix(job.Name, "tick-manual-") || len(job.Name) != len("tick-manual-")+5 ||
		job.Annotations[cluster.ManualJobAnnotation] != "manual" || job.Annotations["cogpit.test/from"] != "template" ||
		job.Labels["app"] != "tick" || owner == nil || owner.Kind != "CronJob" || owner.Name != "tick" {
		t.Errorf("triggered job %s: labels %v annotations %v owner %+v", job.Name, job.Labels, job.Annotations, owner)
	}

	// Suspend, then Resume, which takes Suspend's place once the cache sees the change.
	suspended := func() bool {
		cj, err := kube.BatchV1().CronJobs(ns).Get(ctx, "tick", metav1.GetOptions{})
		return err == nil && cj.Spec.Suspend != nil && *cj.Spec.Suspend
	}
	d.must("cronjob.suspend", map[string]any{"ref": tick}, nil)
	if !suspended() {
		t.Error("cron job not suspended")
	}
	d.cached(tick, func(o *unstructured.Unstructured) bool {
		v, _ := model.NestedBool(o.Object, "spec", "suspend")
		return v
	})
	d.must("cronjob.resume", map[string]any{"ref": tick}, nil)
	if suspended() {
		t.Error("cron job not resumed")
	}

	// Evict: refused while the budget allows no disruption, naming it; allowed once it is gone.
	waitFor(ctx, t, "one ready web pod and its budget settled", func() bool {
		return len(readyPods(ctx, kube, ns, "web")) == 1 && budgetSettled(ctx, kube, ns, "web-pdb", 0)
	})
	podRef := model.ObjectRef{Kind: "pods", Namespace: ns, Name: readyPods(ctx, kube, ns, "web")[0].Name}
	d.cached(podRef, nil)
	err = d.run("pod.evict", map[string]any{"ref": podRef}, nil)
	var blocked *cluster.EvictionBlockedError
	if !errors.As(err, &blocked) || blocked.Budget != "web-pdb" || !strings.Contains(err.Error(), "PodDisruptionBudget web-pdb") {
		t.Fatalf("evict under the budget: %v", err)
	}
	t.Logf("blocked eviction: %v", err)
	if err := kube.PolicyV1().PodDisruptionBudgets(ns).Delete(ctx, "web-pdb", metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	waitFor(ctx, t, "the eviction once the budget is gone", func() bool { return d.run("pod.evict", map[string]any{"ref": podRef}, nil) == nil })
	waitFor(ctx, t, "the evicted pod to go", func() bool {
		p, err := kube.CoreV1().Pods(ns).Get(ctx, podRef.Name, metav1.GetOptions{})
		return apierrors.IsNotFound(err) || (err == nil && p.DeletionTimestamp != nil)
	})
}
