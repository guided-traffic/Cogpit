package cluster

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	fakediscovery "k8s.io/client-go/discovery/fake"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

var podsGVR = schema.GroupVersionResource{Version: "v1", Resource: "pods"}

// drainFake is a Conn on client-go's fake clientset that behaves like an API server for a drain:
// it serves the eviction API in discovery, filters pod lists by spec.nodeName (the fake ignores
// field selectors), and an eviction deletes the pod unless evict refuses it.
type drainFake struct {
	conn *Conn
	cs   *fake.Clientset

	mu    sync.Mutex
	evict func(namespace, name string) error // nil: every eviction succeeds
}

func newDrainFake(t *testing.T, objs ...runtime.Object) *drainFake {
	t.Helper()
	old := evictRetryDelay
	evictRetryDelay = 10 * time.Millisecond
	t.Cleanup(func() { evictRetryDelay = old })

	conn, cs := fakeConn(objs...)
	f := &drainFake{conn: conn, cs: cs}
	cs.Discovery().(*fakediscovery.FakeDiscovery).Resources = []*metav1.APIResourceList{{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "pods", Kind: "Pod", Namespaced: true},
			{Name: "pods/eviction", Kind: "Eviction", Group: "policy", Version: "v1", Namespaced: true},
		},
	}}
	cs.PrependReactor("list", "pods", func(a k8stesting.Action) (bool, runtime.Object, error) {
		obj, err := cs.Tracker().List(podsGVR, corev1.SchemeGroupVersion.WithKind("Pod"), a.GetNamespace())
		if err != nil {
			return true, nil, err
		}
		list := obj.(*corev1.PodList)
		sel := a.(k8stesting.ListAction).GetListRestrictions().Fields
		list.Items = slices.DeleteFunc(list.Items, func(p corev1.Pod) bool {
			return sel != nil && !sel.Matches(fields.Set{"spec.nodeName": p.Spec.NodeName})
		})
		return true, list, nil
	})
	cs.PrependReactor("create", "pods", func(a k8stesting.Action) (bool, runtime.Object, error) {
		if a.GetSubresource() != "eviction" {
			return false, nil, nil
		}
		ev := a.(k8stesting.CreateAction).GetObject().(*policyv1.Eviction)
		f.mu.Lock()
		refuse := f.evict
		f.mu.Unlock()
		if refuse != nil {
			if err := refuse(ev.Namespace, ev.Name); err != nil {
				return true, nil, err
			}
		}
		return true, nil, cs.Tracker().Delete(podsGVR, ev.Namespace, ev.Name)
	})
	return f
}

func (f *drainFake) setEvict(fn func(namespace, name string) error) {
	f.mu.Lock()
	f.evict = fn
	f.mu.Unlock()
}

// evictions returns the pods an eviction was requested for, "namespace/name", in order.
func (f *drainFake) evictions() []string {
	var out []string
	for _, a := range f.cs.Actions() {
		if a.GetVerb() == "create" && a.GetSubresource() == "eviction" {
			ev := a.(k8stesting.CreateAction).GetObject().(*policyv1.Eviction)
			out = append(out, ev.Namespace+"/"+ev.Name)
		}
	}
	return out
}

func (f *drainFake) exists(t *testing.T, namespace, name string) bool {
	t.Helper()
	_, err := f.cs.Tracker().Get(podsGVR, namespace, name)
	if err != nil && !apierrors.IsNotFound(err) {
		t.Fatal(err)
	}
	return err == nil
}

func (f *drainFake) unschedulable(t *testing.T, node string) bool {
	t.Helper()
	n, err := f.cs.CoreV1().Nodes().Get(context.Background(), node, metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return n.Spec.Unschedulable
}

func node(name string, unschedulable bool) *corev1.Node {
	return &corev1.Node{ObjectMeta: metav1.ObjectMeta{Name: name}, Spec: corev1.NodeSpec{Unschedulable: unschedulable}}
}

// pod builds a running pod on a node; owner, if not empty, is "Kind/name" of its controller.
func pod(namespace, name, nodeName, owner string) *corev1.Pod {
	p := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name, UID: types.UID("uid-" + name)},
		Spec:       corev1.PodSpec{NodeName: nodeName},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}
	if owner != "" {
		kind, ownerName, _ := strings.Cut(owner, "/")
		apiVersion := "apps/v1"
		if kind == "Job" {
			apiVersion = "batch/v1"
		}
		yes := true
		p.OwnerReferences = []metav1.OwnerReference{{APIVersion: apiVersion, Kind: kind, Name: ownerName, UID: types.UID("uid-" + ownerName), Controller: &yes}}
	}
	return p
}

func withEmptyDir(p *corev1.Pod) *corev1.Pod {
	p.Spec.Volumes = []corev1.Volume{{Name: "scratch", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}}
	return p
}

// pdbBlocked is the API server's answer to an eviction a PodDisruptionBudget blocks.
func pdbBlocked(budget string) error {
	err := apierrors.NewTooManyRequests("Cannot evict pod as it would violate the pod's disruption budget.", 0)
	err.ErrStatus.Details.Causes = append(err.ErrStatus.Details.Causes, metav1.StatusCause{
		Type:    policyv1.DisruptionBudgetCause,
		Message: fmt.Sprintf("The disruption budget %s needs 1 healthy pods and has 1 currently", budget),
	})
	return err
}

// progressLog collects a drain's progress reports.
type progressLog struct {
	mu    sync.Mutex
	steps []DrainProgress
}

func (l *progressLog) add(p DrainProgress) {
	l.mu.Lock()
	l.steps = append(l.steps, p)
	l.mu.Unlock()
}

func (l *progressLog) all() []DrainProgress {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.steps)
}

func (l *progressLog) has(step DrainStep, pod string) bool {
	for _, p := range l.all() {
		if p.Step == step && p.Pod == pod {
			return true
		}
	}
	return false
}

func TestSetNodeUnschedulablePatches(t *testing.T) {
	c, cs := fakeConn(node("worker-1", false))
	ctx := context.Background()
	if err := c.SetNodeUnschedulable(ctx, "worker-1", true); err != nil {
		t.Fatal(err)
	}
	if err := c.SetNodeUnschedulable(ctx, "worker-1", false); err != nil {
		t.Fatal(err)
	}
	var bodies []string
	for _, a := range cs.Actions() {
		p, ok := a.(k8stesting.PatchAction)
		if !ok {
			continue
		}
		if p.GetPatchType() != types.StrategicMergePatchType || p.GetResource().Resource != "nodes" || p.GetName() != "worker-1" || p.GetSubresource() != "" {
			t.Errorf("patch %s on %s/%s %s", p.GetPatchType(), p.GetResource().Resource, p.GetSubresource(), p.GetName())
		}
		bodies = append(bodies, string(p.GetPatch()))
	}
	want := []string{`{"spec":{"unschedulable":true}}`, `{"spec":{"unschedulable":null}}`}
	if !slices.Equal(bodies, want) {
		t.Errorf("patch bodies %q, want %q (kubectl cordon and uncordon)", bodies, want)
	}
	n, err := cs.CoreV1().Nodes().Get(ctx, "worker-1", metav1.GetOptions{})
	if err != nil || n.Spec.Unschedulable {
		t.Errorf("after uncordon: %v, %v", n.Spec, err)
	}

	err = c.SetNodeUnschedulable(ctx, "gone", true)
	if !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "cordon node gone") {
		t.Errorf("cordon a missing node: %v", err)
	}
}

// TestDrainNodeEvictsWhatKubectlWould drains a node with pods of several kinds: a ReplicaSet's pod
// and a finished unmanaged pod are evicted, the DaemonSet's pod and a mirror pod stay, and a pod
// on another node is not touched.
func TestDrainNodeEvictsWhatKubectlWould(t *testing.T) {
	mirror := pod("kube-system", "etcd-worker-1", "worker-1", "")
	mirror.Annotations = map[string]string{corev1.MirrorPodAnnotationKey: "hash"}
	done := pod("batch", "report-1", "worker-1", "")
	done.Status.Phase = corev1.PodSucceeded
	f := newDrainFake(t,
		node("worker-1", false), node("worker-2", false),
		&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "kube-proxy"}},
		pod("shop", "web-1", "worker-1", "ReplicaSet/web-5d8"),
		pod("kube-system", "kube-proxy-x1", "worker-1", "DaemonSet/kube-proxy"),
		mirror, done,
		pod("shop", "web-2", "worker-2", "ReplicaSet/web-5d8"),
	)
	var log progressLog
	err := f.conn.DrainNode(context.Background(), "worker-1", DrainOptions{IgnoreDaemonSets: true, GracePeriodSeconds: -1}, log.add)
	if err != nil {
		t.Fatalf("DrainNode: %v", err)
	}
	if !f.unschedulable(t, "worker-1") || f.unschedulable(t, "worker-2") {
		t.Error("worker-1 not cordoned, or worker-2 cordoned")
	}
	got := f.evictions()
	slices.Sort(got)
	if want := []string{"batch/report-1", "shop/web-1"}; !slices.Equal(got, want) {
		t.Errorf("evictions %v, want %v", got, want)
	}
	for _, p := range []string{"kube-system/kube-proxy-x1", "kube-system/etcd-worker-1", "shop/web-2"} {
		ns, name, _ := strings.Cut(p, "/")
		if !f.exists(t, ns, name) {
			t.Errorf("%s was removed", p)
		}
	}

	// The requests, verb by verb: the contract privilege-footprint.md documents.
	verbs := map[string]bool{}
	for _, a := range f.cs.Actions() {
		res := a.GetResource().Resource
		if a.GetSubresource() != "" {
			res += "/" + a.GetSubresource()
		}
		verbs[a.GetVerb()+" "+res] = true
		if l, ok := a.(k8stesting.ListAction); ok && res == "pods" {
			if fs := l.GetListRestrictions().Fields.String(); fs != "spec.nodeName=worker-1" || a.GetNamespace() != "" {
				t.Errorf("pod list in %q with fields %q", a.GetNamespace(), fs)
			}
		}
		if c, ok := a.(k8stesting.CreateAction); ok && a.GetSubresource() == "eviction" {
			if ev := c.GetObject().(*policyv1.Eviction); ev.DeleteOptions == nil || ev.DeleteOptions.GracePeriodSeconds != nil {
				t.Errorf("eviction %s: delete options %+v, want the pod's own grace period", ev.Name, ev.DeleteOptions)
			}
		}
	}
	for v := range verbs {
		switch v {
		case "patch nodes", "get resource", "list pods", "get daemonsets", "create pods/eviction", "get pods", "get nodes":
		default:
			t.Errorf("unexpected request %s", v)
		}
	}
	for _, v := range []string{"patch nodes", "list pods", "get daemonsets", "create pods/eviction", "get pods"} {
		if !verbs[v] {
			t.Errorf("no %s request; requests: %v", v, verbs)
		}
	}

	steps := log.all()
	if len(steps) == 0 || steps[0].Step != DrainCordoned || steps[1].Step != DrainListed || steps[1].Total != 2 {
		t.Fatalf("progress starts %+v", steps)
	}
	if !strings.Contains(steps[1].Message, "kube-proxy-x1") {
		t.Errorf("listed step does not name the DaemonSet pod left running: %q", steps[1].Message)
	}
	last := steps[len(steps)-1]
	if last.Done != 2 || last.Total != 2 || last.Node != "worker-1" || !log.has(DrainEvicted, "shop/web-1") || !log.has(DrainEvicting, "batch/report-1") {
		t.Errorf("progress %+v", steps)
	}
}

func TestDrainNodeRefusesWhatWouldLoseData(t *testing.T) {
	for _, tc := range []struct {
		name    string
		pod     *corev1.Pod
		opts    DrainOptions
		wantErr string
	}{
		{"unmanaged pod", pod("shop", "debug", "worker-1", ""), DrainOptions{IgnoreDaemonSets: true}, "set force"},
		{"emptyDir pod", withEmptyDir(pod("shop", "cache-0", "worker-1", "ReplicaSet/cache")), DrainOptions{IgnoreDaemonSets: true}, "set deleteEmptyDirData"},
		{"DaemonSet pod", pod("kube-system", "kube-proxy-x1", "worker-1", "DaemonSet/kube-proxy"), DrainOptions{}, "set ignoreDaemonSets"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDrainFake(t, node("worker-1", false), tc.pod,
				&appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Namespace: "kube-system", Name: "kube-proxy"}})
			err := f.conn.DrainNode(context.Background(), "worker-1", tc.opts, nil)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), tc.pod.Namespace+"/"+tc.pod.Name) {
				t.Fatalf("err = %v, want it to name the pod and %q", err, tc.wantErr)
			}
			if strings.Contains(err.Error(), "--") {
				t.Errorf("the error names kubectl's flags: %v", err)
			}
			if got := f.evictions(); len(got) != 0 {
				t.Errorf("evicted %v before refusing", got)
			}
			if !f.unschedulable(t, "worker-1") {
				t.Error("the node is not cordoned (kubectl leaves a refused drain cordoned)")
			}
		})
	}

	// With the option the user accepted, the pods go.
	f := newDrainFake(t, node("worker-1", false), pod("shop", "debug", "worker-1", ""),
		withEmptyDir(pod("shop", "cache-0", "worker-1", "ReplicaSet/cache")))
	if err := f.conn.DrainNode(context.Background(), "worker-1", DrainOptions{Force: true, DeleteEmptyDirData: true, GracePeriodSeconds: 30}, nil); err != nil {
		t.Fatal(err)
	}
	if got := f.evictions(); len(got) != 2 {
		t.Errorf("evictions %v", got)
	}
	for _, a := range f.cs.Actions() {
		if c, ok := a.(k8stesting.CreateAction); ok && a.GetSubresource() == "eviction" {
			if g := c.GetObject().(*policyv1.Eviction).DeleteOptions.GracePeriodSeconds; g == nil || *g != 30 {
				t.Errorf("grace period %v, want 30", g)
			}
		}
	}
}

// TestDrainNodeBlockedByBudget: a PodDisruptionBudget refuses one eviction (HTTP 429) until the
// drain's timeout; the other pod goes, the drain fails naming the budget, the blocked pod stays.
func TestDrainNodeBlockedByBudget(t *testing.T) {
	f := newDrainFake(t, node("worker-1", false),
		pod("shop", "web-1", "worker-1", "ReplicaSet/web-5d8"), pod("shop", "db-0", "worker-1", "StatefulSet/db"))
	f.setEvict(func(_, name string) error {
		if name == "db-0" {
			return pdbBlocked("db-pdb")
		}
		return nil
	})
	var log progressLog
	start := time.Now()
	err := f.conn.DrainNode(context.Background(), "worker-1", DrainOptions{GracePeriodSeconds: -1, Timeout: 300 * time.Millisecond}, log.add)
	if err == nil {
		t.Fatal("the drain finished although a budget blocked an eviction")
	}
	for _, want := range []string{"drain node worker-1", "timed out after 300ms", "pod shop/db-0", "PodDisruptionBudget db-pdb", "needs 1 healthy pods"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error lacks %q: %v", want, err)
		}
	}
	if time.Since(start) > 5*time.Second {
		t.Errorf("the drain took %s with a 300ms timeout", time.Since(start))
	}
	if f.exists(t, "shop", "web-1") || !f.exists(t, "shop", "db-0") {
		t.Error("web-1 should be evicted and db-0 should stay")
	}
	if n := strings.Count(strings.Join(f.evictions(), " "), "shop/db-0"); n < 2 {
		t.Errorf("db-0's eviction was tried %d times; the drain retries a blocked eviction", n)
	}
	blocked := 0
	for _, p := range log.all() {
		if p.Step == DrainBlocked {
			blocked++
			if p.Pod != "shop/db-0" || !strings.Contains(p.Message, "db-pdb") {
				t.Errorf("blocked step %+v", p)
			}
		}
	}
	if blocked != 1 {
		t.Errorf("%d blocked steps, want one per pod and reason", blocked)
	}
}

func TestDrainNodeCancelled(t *testing.T) {
	f := newDrainFake(t, node("worker-1", false), pod("shop", "db-0", "worker-1", "StatefulSet/db"))
	f.setEvict(func(string, string) error { return pdbBlocked("db-pdb") })
	ctx, cancel := context.WithCancel(context.Background())
	err := f.conn.DrainNode(ctx, "worker-1", DrainOptions{}, func(p DrainProgress) {
		if p.Step == DrainBlocked {
			cancel() // the user, or a crash of the caller, ends the drain halfway
		}
	})
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("err = %v, want cancelled", err)
	}
	if !f.unschedulable(t, "worker-1") || !f.exists(t, "shop", "db-0") {
		t.Error("a cancelled drain leaves the node cordoned and the pod in place")
	}
}

func TestDrainNodeNeedsTheEvictionAPI(t *testing.T) {
	f := newDrainFake(t, node("worker-1", false), pod("shop", "web-1", "worker-1", "ReplicaSet/web"))
	f.cs.Discovery().(*fakediscovery.FakeDiscovery).Resources[0].APIResources = []metav1.APIResource{{Name: "pods", Kind: "Pod"}}
	err := f.conn.DrainNode(context.Background(), "worker-1", DrainOptions{}, nil)
	if err == nil || !strings.Contains(err.Error(), "no eviction API") {
		t.Fatalf("err = %v", err)
	}
	for _, a := range f.cs.Actions() {
		if a.GetVerb() == "delete" {
			t.Errorf("deleted instead of evicting: %v", a)
		}
	}
}

func TestDrainNodeMissingNode(t *testing.T) {
	f := newDrainFake(t)
	err := f.conn.DrainNode(context.Background(), "gone", DrainOptions{}, nil)
	if !apierrors.IsNotFound(err) || !strings.Contains(err.Error(), "drain node gone") {
		t.Errorf("err = %v", err)
	}
}

func TestDrainReporterSurvivesAPanickingCallback(t *testing.T) {
	f := newDrainFake(t, node("worker-1", false), pod("shop", "web-1", "worker-1", "ReplicaSet/web"))
	err := f.conn.DrainNode(context.Background(), "worker-1", DrainOptions{}, func(DrainProgress) { panic("callback bug") })
	if err != nil {
		t.Fatalf("a panicking progress callback broke the drain: %v", err)
	}
	if f.exists(t, "shop", "web-1") {
		t.Error("web-1 not evicted")
	}
}

func TestExplainEviction(t *testing.T) {
	blockedErr := explainEviction("shop/db-0", pdbBlocked("db-pdb"))
	var blocked *EvictionBlockedError
	if !errors.As(blockedErr, &blocked) || blocked.Budget != "db-pdb" || blocked.Pod != "shop/db-0" || !apierrors.IsTooManyRequests(blocked) {
		t.Fatalf("blocked: %#v", blockedErr)
	}
	processing := apierrors.NewTooManyRequests("Cannot evict pod as it would violate the pod's disruption budget.", 10)
	processing.ErrStatus.Details.Causes = []metav1.StatusCause{{Type: policyv1.DisruptionBudgetCause, Message: "The disruption budget web is still being processed by the server."}}
	if err := explainEviction("shop/web-0", processing); !errors.As(err, &blocked) || blocked.Budget != "web" {
		t.Errorf("processing: %v", err)
	}
	throttled := explainEviction("shop/web-0", apierrors.NewTooManyRequests("slow down", 1))
	if !errors.As(throttled, &blocked) || blocked.Budget != "" || !strings.Contains(throttled.Error(), "slow down") {
		t.Errorf("throttled: %v", throttled)
	}
	other := apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "web-0", errors.New("no"))
	if err := explainEviction("shop/web-0", other); err != other { //nolint:errorlint // identity is the point
		t.Errorf("other errors pass through: %v", err)
	}
}
