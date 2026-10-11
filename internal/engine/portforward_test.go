package engine

import (
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

func readyPod(name string, ready bool, age time.Duration, labels map[string]any, ports ...any) *unstructured.Unstructured {
	status := "False"
	if ready {
		status = "True"
	}
	p := testkit.Object("v1", "Pod", "shop", name, map[string]any{
		"metadata": map[string]any{"labels": labels, "creationTimestamp": time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC).Add(-age).Format(time.RFC3339)},
		"spec":     map[string]any{"containers": []any{map[string]any{"name": "app", "ports": ports}}},
		"status":   map[string]any{"phase": "Running", "conditions": []any{map[string]any{"type": "Ready", "status": status}}},
	})
	return p
}

func service(ports ...any) *unstructured.Unstructured {
	return testkit.Object("v1", "Service", "shop", "web", map[string]any{
		"spec": map[string]any{"selector": map[string]any{"app": "web"}, "ports": ports},
	})
}

// servicePod picks a ready pod of the selector, the oldest unless the current one is still
// ready, and the port the service targets in it, as kubectl port-forward svc/… does.
func TestServicePod(t *testing.T) {
	web := map[string]any{"app": "web"}
	httpPort := map[string]any{"name": "http", "containerPort": int64(8080)}
	pods := []*unstructured.Unstructured{
		readyPod("young", true, time.Hour, web, httpPort),
		readyPod("old", true, 48*time.Hour, web, httpPort),
		readyPod("unready", false, 72*time.Hour, web, httpPort),
		readyPod("other-app", true, 96*time.Hour, map[string]any{"app": "db"}, httpPort),
		readyPod("no-http", true, 120*time.Hour, web),
	}
	for _, tt := range []struct {
		name    string
		svc     *unstructured.Unstructured
		port    uint16
		prefer  string
		pod     string
		podPort uint16
		why     string
	}{
		{"named target port, oldest ready pod with it", service(map[string]any{"port": int64(80), "targetPort": "http"}), 80, "", "old", 8080, ""},
		{"the current pod while it is ready", service(map[string]any{"port": int64(80), "targetPort": "http"}), 80, "young", "young", 8080, ""},
		{"numeric target port", service(map[string]any{"port": int64(80), "targetPort": int64(9000)}), 80, "", "no-http", 9000, ""},
		{"no target port: the service's port", service(map[string]any{"port": int64(80)}), 80, "", "no-http", 80, ""},
		{"no such port", service(map[string]any{"port": int64(80)}), 81, "", "", 0, "has no TCP port 81"},
		{"UDP port", service(map[string]any{"port": int64(53), "protocol": "UDP"}), 53, "", "", 0, "has no TCP port 53"},
		{"named port no pod has", service(map[string]any{"port": int64(80), "targetPort": "grpc"}), 80, "", "", 0, "has the port grpc"},
	} {
		got, why := servicePod(tt.svc, tt.port, pods, tt.prefer)
		if got.pod != tt.pod || got.port != tt.podPort || !strings.Contains(why, tt.why) || (tt.why == "") != (why == "") {
			t.Errorf("%s: %+v %q, want %s:%d %q", tt.name, got, why, tt.pod, tt.podPort, tt.why)
		}
	}
	noSelector := testkit.Object("v1", "Service", "shop", "ext", map[string]any{"spec": map[string]any{"ports": []any{map[string]any{"port": int64(80)}}}})
	if _, why := servicePod(noSelector, 80, pods, ""); !strings.Contains(why, "no selector") {
		t.Errorf("no selector: %q", why)
	}
	if _, why := servicePod(service(map[string]any{"port": int64(80)}), 80, pods[2:3], ""); why != "no ready pod for service web" {
		t.Errorf("no ready pod: %q", why)
	}
}

func TestPodNotRunning(t *testing.T) {
	p := readyPod("web-0", true, 0, nil)
	if why := podNotRunning(p); why != "" {
		t.Errorf("running pod: %q", why)
	}
	_ = unstructured.SetNestedField(p.Object, "Pending", "status", "phase")
	if why := podNotRunning(p); why != "pod web-0 is pending" {
		t.Errorf("pending pod: %q", why)
	}
	p.SetDeletionTimestamp(nil)
	unstructured.RemoveNestedField(p.Object, "status", "phase")
	if why := podNotRunning(p); why != "pod web-0 is not running" {
		t.Errorf("pod without a phase: %q", why)
	}
}

// A forward's object reads back as the forward, and the names and URLs are what the views show.
func TestForwardObjectRoundTrip(t *testing.T) {
	f := Forward{
		Ref: model.ObjectRef{Kind: PortForwardKind, Namespace: "shop", Name: "service-web-80"},
		Spec: ForwardSpec{Target: model.ObjectRef{Kind: "services", Namespace: "shop", Name: "web"}, Port: 80, LocalPort: 9080,
			HTTPS: true, OpenInBrowser: true},
		Phase: ForwardActive, LocalPort: 9080, Address: "127.0.0.1:9080", URL: forwardURL(true, 9080), Pod: "web-1", PodPort: 8080,
	}
	got := parseForward(forwardObject(f, "uid-1", time.Now()))
	got.Ref = f.Ref
	if got != f {
		t.Errorf("round trip:\n got %+v\nwant %+v", got, f)
	}
	if f.URL != "https://localhost:9080" || ForwardName(f.Spec.Target, 80) != "service-web-80" ||
		ForwardName(model.ObjectRef{Kind: "pods", Name: "web-0"}, 8080) != "pod-web-0-8080" {
		t.Errorf("url %s, names %s", f.URL, ForwardName(f.Spec.Target, 80))
	}
}
