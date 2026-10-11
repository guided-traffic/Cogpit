//go:build kind

package integration_test

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// storageNamespace is the shared kind cluster's namespace of this test; it creates it, acts only
// inside it, and deletes it.
const storageNamespace = "cogpit-d"

// TestKindStorageAndNetworkViews creates a claim (kind's default class provisions a local volume
// once a pod uses it), a pod that mounts it, a Service for the pod, a network policy and a
// replication controller in its own namespace, and checks the network and storage views against
// the real API server: the claim's Pods column names the pod, the volume names the claim, the
// node's drawer lists the pod, the Service's drawer its endpoint slice, and the API server's
// deprecation warning for v1 Endpoints is kept once per connection however often it comes.
func TestKindStorageAndNetworkViews(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	conn := s.Backend().Conn
	kube, ns := conn.Kube(), storageNamespace
	freshNamespace(ctx, t, kube, ns)

	if _, err := kube.CoreV1().PersistentVolumeClaims(ns).Create(ctx, &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "data"},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources:   corev1.VolumeResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("64Mi")}},
		},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	tmpl := podTemplate("app", nil)
	tmpl.Spec.Volumes = []corev1.Volume{{Name: "data", VolumeSource: corev1.VolumeSource{
		PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: "data"},
	}}}
	tmpl.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{{Name: "data", MountPath: "/data"}}
	if _, err := kube.CoreV1().Pods(ns).Create(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "app", Labels: tmpl.Labels}, Spec: tmpl.Spec}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.CoreV1().Services(ns).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "app"},
		Spec: corev1.ServiceSpec{Selector: map[string]string{"app": "app"},
			Ports: []corev1.ServicePort{{Name: "http", Port: 80, TargetPort: intstr.FromInt32(8080)}}},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.NetworkingV1().NetworkPolicies(ns).Create(ctx, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "deny-ingress"},
		Spec:       networkingv1.NetworkPolicySpec{PolicyTypes: []networkingv1.PolicyType{networkingv1.PolicyTypeIngress}},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	rcTmpl := podTemplate("legacy", nil)
	if _, err := kube.CoreV1().ReplicationControllers(ns).Create(ctx, &corev1.ReplicationController{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "legacy"},
		Spec:       corev1.ReplicationControllerSpec{Replicas: int32p(1), Selector: map[string]string{"app": "legacy"}, Template: &rcTmpl},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	waitFor(ctx, t, "the pod to run with its claim bound", func() bool {
		pvc, err := kube.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "data", metav1.GetOptions{})
		return err == nil && pvc.Status.Phase == corev1.ClaimBound && len(readyPods(ctx, kube, ns, "app")) == 1
	})
	pod := readyPods(ctx, kube, ns, "app")[0]
	pvc, err := kube.CoreV1().PersistentVolumeClaims(ns).Get(ctx, "data", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("claim %s/data bound to %s (class %s) on node %s, pod IP %s", ns, pvc.Spec.VolumeName, ptrString(pvc.Spec.StorageClassName), pod.Spec.NodeName, pod.Status.PodIP)

	d := dispatcher{t: t, ctx: ctx, s: s}
	// row waits until the view of kind shows the object with the name, namespace ns or none, and
	// returns its cells by column id.
	row := func(kind, namespace, name string, cond func(map[string]model.Cell) bool) map[string]model.Cell {
		t.Helper()
		d.must("ui.openView", map[string]any{"view": kind}, nil)
		var cells map[string]model.Cell
		waitFor(ctx, t, kind+" to show "+namespace+"/"+name, func() bool {
			tbl := s.Table(kind)
			if tbl == nil || !tbl.Synced {
				return false
			}
			for _, r := range tbl.Rows {
				if r.Ref.Namespace == namespace && (r.Ref.Name == name || name == "" && strings.HasPrefix(r.Ref.Name, "app-")) {
					cells = map[string]model.Cell{}
					for i, c := range tbl.Columns {
						cells[c.ID] = r.Cells[i]
					}
					return cond == nil || cond(cells)
				}
			}
			return false
		})
		return cells
	}

	claim := row("persistentvolumeclaims", ns, "data", func(c map[string]model.Cell) bool { return c["pods"].Text == "app" })
	if l := claim["pods"].Link; l == nil || *l != (model.ObjectRef{Kind: "pods", Namespace: ns, Name: "app"}) {
		t.Errorf("claim's Pods link %+v", l)
	}
	if c := claim["storageClass"]; c.Link == nil || c.Text != ptrString(pvc.Spec.StorageClassName) || claim["status"].Text != "Bound" {
		t.Errorf("claim's class %+v, status %+v", c, claim["status"])
	}
	volume := row("persistentvolumes", "", pvc.Spec.VolumeName, nil)
	if volume["claim"].Text != ns+"/data" || volume["status"].Text != "Bound" || volume["status"].Status != model.StatusOK {
		t.Errorf("volume's claim %+v, status %+v", volume["claim"], volume["status"])
	}
	class := row("storageclasses", "", ptrString(pvc.Spec.StorageClassName), nil)
	t.Logf("storage class %s: provisioner %s, reclaim %s, binding %s", ptrString(pvc.Spec.StorageClassName),
		class["provisioner"].Text, class["reclaimPolicy"].Text, class["volumeBindingMode"].Text)

	slice := row("endpointslices", ns, "", func(c map[string]model.Cell) bool { return strings.Contains(c["endpoints"].Text, pod.Status.PodIP) })
	if slice["ports"].Text != "8080/TCP" || slice["addressType"].Text != "IPv4" {
		t.Errorf("endpoint slice %+v", slice)
	}
	row("endpoints", ns, "app", func(c map[string]model.Cell) bool { return c["endpoints"].Text == pod.Status.PodIP+":8080" })
	if policy := row("networkpolicies", ns, "deny-ingress", nil); policy["policyTypes"].Text != "Ingress" {
		t.Errorf("network policy %+v", policy)
	}
	row("replicationcontrollers", ns, "legacy", func(c map[string]model.Cell) bool {
		return c["replicas"].Num == 1 && c["desiredReplicas"].Num == 1 && c["selector"].Text == "app=legacy"
	})

	// The node's drawer lists the pod; the Service's drawer its endpoint slice.
	waitFor(ctx, t, "the node's Pods table to list the pod", func() bool {
		m, err := s.DetailWait(ctx, model.ObjectRef{Kind: "nodes", Name: pod.Spec.NodeName})
		return err == nil && tableHas(m, "Pods", "app", ns)
	})
	waitFor(ctx, t, "the Service's drawer to list its endpoint slice", func() bool {
		m, err := s.DetailWait(ctx, model.ObjectRef{Kind: "services", Namespace: ns, Name: "app"})
		return err == nil && tableHas(m, "Endpoint Slices", "app-", "")
	})

	// v1 Endpoints are deprecated: the API server warns on every request for them (the view's
	// informer: its initial list, then each watch it reconnects). The engine's connection keeps
	// the warning once, however often it comes: two more lists add two more.
	_, before := conn.Warnings()
	for range 2 {
		if _, err := kube.CoreV1().Endpoints(ns).List(ctx, metav1.ListOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	distinct, received := conn.Warnings()
	t.Logf("API server warnings on the connection: %d from the views, %d with two more lists, %d distinct: %q", before, received, len(distinct), distinct)
	if n := countPrefix(distinct, "v1 Endpoints is deprecated"); n != 1 || received != before+2 {
		t.Errorf("the Endpoints deprecation is kept %d times; %d received after %d", n, received, before)
	}
}

func ptrString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func countPrefix(list []string, prefix string) int {
	return len(slices.DeleteFunc(slices.Clone(list), func(s string) bool { return !strings.HasPrefix(s, prefix) }))
}

// tableHas reports whether the drawer section titled title has a table row whose first cell
// starts with name and, when namespace is set, names that namespace in a cell.
func tableHas(m *engine.DetailModel, title, name, namespace string) bool {
	for _, sec := range m.Sections {
		if sec.Title != title || sec.Table == nil {
			continue
		}
		for _, r := range sec.Table.Rows {
			if len(r) > 0 && strings.HasPrefix(r[0], name) && (namespace == "" || slices.Contains(r, namespace)) {
				return true
			}
		}
	}
	return false
}
