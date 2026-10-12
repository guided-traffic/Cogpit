package metrics_test

import (
	"context"
	"errors"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakediscovery "k8s.io/client-go/discovery/fake"
	clienttesting "k8s.io/client-go/testing"
	apimetrics "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

func served(resources ...string) *fakediscovery.FakeDiscovery {
	d := &fakediscovery.FakeDiscovery{Fake: &clienttesting.Fake{}}
	if resources != nil {
		list := &metav1.APIResourceList{GroupVersion: metrics.GroupVersion}
		for _, r := range resources {
			list.APIResources = append(list.APIResources, metav1.APIResource{Name: r, Verbs: metav1.Verbs{"get", "list"}})
		}
		d.Resources = []*metav1.APIResourceList{list}
	}
	return d
}

func podMetrics(ns, name string, ts time.Time, containers map[string][2]string) *apimetrics.PodMetrics {
	pm := &apimetrics.PodMetrics{ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name}, Timestamp: metav1.Time{Time: ts}, Window: metav1.Duration{Duration: 15 * time.Second}}
	for _, n := range []string{"app", "sidecar"} {
		if v, ok := containers[n]; ok {
			pm.Containers = append(pm.Containers, apimetrics.ContainerMetrics{Name: n, Usage: corev1.ResourceList{
				"cpu": resource.MustParse(v[0]), "memory": resource.MustParse(v[1]),
			}})
		}
	}
	return pm
}

func TestDetect(t *testing.T) {
	for _, tc := range []struct {
		name   string
		disco  *fakediscovery.FakeDiscovery
		inject error
		want   metrics.Status
	}{
		{"served", served("pods", "nodes"), nil, metrics.StatusAvailable},
		{"group missing", served(), nil, metrics.StatusNotServed},
		{"group without pods and nodes", served("other"), nil, metrics.StatusNotServed},
		{"forbidden", served("pods"), apierrors.NewForbidden(schema.GroupResource{}, "", errors.New("no")), metrics.StatusForbidden},
		{"apiservice down", served("pods"), apierrors.NewServiceUnavailable("the server is currently unable to handle the request"), metrics.StatusUnavailable},
		{"network error", served("pods"), errors.New("connection refused"), metrics.StatusUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.inject != nil {
				tc.disco.PrependReactor("get", "resource", func(clienttesting.Action) (bool, runtime.Object, error) { return true, nil, tc.inject })
			}
			prov, st := metrics.NewServer(tc.disco, metricsfake.NewSimpleClientset().MetricsV1beta1()).Detect(context.Background())
			if st.Status != tc.want {
				t.Fatalf("status = %q (%s), want %q", st.Status, st.Message, tc.want)
			}
			if tc.want == metrics.StatusAvailable {
				if prov == nil || st.Label != "Kubernetes Metrics Server" || st.Provider != "metrics-server" {
					t.Errorf("available state = %+v, provider %v", st, prov)
				}
			} else if prov != nil || st.Message == "" {
				t.Errorf("an unavailable state has no provider and says why: %+v, %v", st, prov)
			}
		})
	}
}

func TestServerReadsPodsAndNodes(t *testing.T) {
	ts := time.Date(2026, 10, 12, 10, 0, 0, 0, time.UTC)
	cs := metricsfake.NewSimpleClientset()
	gv := schema.GroupVersion{Group: "metrics.k8s.io", Version: "v1beta1"}
	add := func(res string, obj runtime.Object, ns string) {
		t.Helper()
		if err := cs.Tracker().Create(gv.WithResource(res), obj, ns); err != nil {
			t.Fatal(err)
		}
	}
	add("pods", podMetrics("web", "a", ts, map[string][2]string{"app": {"250m", "64Mi"}, "sidecar": {"3m", "1Ki"}}), "web")
	add("pods", podMetrics("web", "b", ts, map[string][2]string{"app": {"1", "1Gi"}}), "web")
	add("pods", podMetrics("db", "c", ts, map[string][2]string{"app": {"100m", "10Mi"}}), "db")
	add("nodes", &apimetrics.NodeMetrics{ObjectMeta: metav1.ObjectMeta{Name: "n1"}, Timestamp: metav1.Time{Time: ts}, Usage: corev1.ResourceList{
		"cpu": resource.MustParse("1500m"), "memory": resource.MustParse("2Gi"),
	}}, "")
	s := metrics.NewServer(served("pods", "nodes"), cs.MetricsV1beta1())
	ctx := context.Background()

	all, err := s.ListPods(ctx, "")
	if err != nil || len(all) != 3 {
		t.Fatalf("all namespaces: %d pods, err %v", len(all), err)
	}
	web, err := s.ListPods(ctx, "web")
	if err != nil || len(web) != 2 {
		t.Fatalf("namespace web: %d pods, err %v", len(web), err)
	}
	p, err := s.GetPod(ctx, "web", "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Containers) != 2 || p.Containers[0].Name != "app" || p.Containers[0].CPUMilli != 250 || p.Containers[0].MemoryBytes != 64<<20 {
		t.Errorf("pod containers = %+v", p.Containers)
	}
	if tot := p.Total(); tot.CPUMilli != 253 || tot.MemoryBytes != 64<<20+1024 {
		t.Errorf("pod total = %+v", tot)
	}
	if !p.Timestamp.Equal(ts) {
		t.Errorf("timestamp = %v, want the measurement's %v", p.Timestamp, ts)
	}
	if _, err := s.GetPod(ctx, "web", "missing"); !errors.Is(err, metrics.ErrNoData) {
		t.Errorf("a pod without a measurement: err = %v, want ErrNoData", err)
	}
	nodes, err := s.ListNodes(ctx)
	if err != nil || len(nodes) != 1 || nodes[0].CPUMilli != 1500 || nodes[0].MemoryBytes != 2<<30 {
		t.Fatalf("nodes = %+v, err %v", nodes, err)
	}
	if _, err := s.GetNode(ctx, "n2"); !errors.Is(err, metrics.ErrNoData) {
		t.Errorf("a node without a measurement: err = %v, want ErrNoData", err)
	}
}

func TestStateFromError(t *testing.T) {
	for _, tc := range []struct {
		err  error
		want metrics.Status
	}{
		{apierrors.NewForbidden(schema.GroupResource{Resource: "pods"}, "", errors.New("denied")), metrics.StatusForbidden},
		{apierrors.NewUnauthorized("no"), metrics.StatusForbidden},
		{apierrors.NewNotFound(schema.GroupResource{}, "x"), metrics.StatusNotServed},
		{apierrors.NewServiceUnavailable("down"), metrics.StatusUnavailable},
		{errors.New("boom"), metrics.StatusUnavailable},
	} {
		if got := metrics.StateFromError(tc.err, "none"); got.Status != tc.want || got.Message == "" {
			t.Errorf("%v: %+v, want %q with a message", tc.err, got, tc.want)
		}
	}
}
