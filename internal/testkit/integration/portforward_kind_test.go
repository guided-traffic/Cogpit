//go:build kind

package integration_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// forwardNamespace is the namespace of the port-forward test's own objects; the seed's services
// in cogpit-demo are only read and forwarded to.
const forwardNamespace = "cogpit-a"

// httpGet fetches a forward's URL path, bounded.
func httpGet(ctx context.Context, url string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(io.LimitReader(res.Body, 1<<16))
	if res.StatusCode != http.StatusOK {
		return string(b), fmt.Errorf("HTTP %d", res.StatusCode)
	}
	return string(b), err
}

// echoDeployment is one replica of the e2e test server (agnhost netexec) behind a service: its
// /hostname answers the pod's name, so a test sees which pod a forward reached.
func echoObjects(ns string) (*appsv1.Deployment, *corev1.Service) {
	yes, no, uid := true, false, int64(65534)
	grace := int64(1)
	labels := map[string]string{"app": "echo"}
	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "echo"},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32p(1), Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &grace,
					SecurityContext:               &corev1.PodSecurityContext{RunAsNonRoot: &yes, RunAsUser: &uid},
					Containers: []corev1.Container{{
						Name: "netexec", Image: "registry.k8s.io/e2e-test-images/agnhost:2.53", ImagePullPolicy: corev1.PullIfNotPresent,
						Args:  []string{"netexec", "--http-port=8080"},
						Ports: []corev1.ContainerPort{{Name: "http", ContainerPort: 8080}},
						ReadinessProbe: &corev1.Probe{ProbeHandler: corev1.ProbeHandler{TCPSocket: &corev1.TCPSocketAction{Port: intstr.FromString("http")}},
							PeriodSeconds: 1},
						Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5m"), corev1.ResourceMemory: resource.MustParse("16Mi"),
						}},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &no, Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
					}},
				},
			},
		},
	}
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "echo"},
		Spec: corev1.ServiceSpec{Selector: labels, Ports: []corev1.ServicePort{{
			Name: "http", Port: 80, Protocol: corev1.ProtocolTCP, TargetPort: intstr.FromString("http"),
		}}},
	}
	return dep, svc
}

// TestKindPortForward forwards the seed's web service and one of its api pods through the engine
// and fetches them over HTTP on 127.0.0.1; then forwards a service of its own in cogpit-a, deletes
// the pod behind it and sees the forward reach the replacement through the cache's events; stop,
// delete and leaving the cluster close the local ports.
func TestKindPortForward(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kube, ns := conn.Kube(), forwardNamespace
	freshNamespace(ctx, t, kube, ns)
	dep, svc := echoObjects(ns)
	if _, err := kube.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}})
	closed := false
	defer func() {
		if !closed {
			s.Close()
		}
	}()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	d := dispatcher{t: t, ctx: ctx, s: s}

	// The seed's web service: nginx serves the seed's page.
	web := model.ObjectRef{Kind: "services", Namespace: "cogpit-demo", Name: "web"}
	d.cached(web, nil)
	var webFwd engine.Forward
	d.must("service.portForward", map[string]any{"ref": web, "port": 80}, &webFwd)
	host, _, _ := net.SplitHostPort(webFwd.Address)
	if webFwd.Phase != engine.ForwardActive || host != "127.0.0.1" || !strings.HasPrefix(webFwd.URL, "http://localhost:") {
		t.Fatalf("web forward %+v", webFwd)
	}
	body, err := httpGet(ctx, "http://"+webFwd.Address+"/")
	if err != nil || !strings.Contains(body, "Hello from the Cogpit kind seed.") {
		t.Fatalf("GET through the web forward: %v\n%s", err, body)
	}
	if body, err := httpGet(ctx, webFwd.URL+"/"); err != nil || !strings.Contains(body, "cogpit-demo") {
		t.Errorf("GET %s: %v", webFwd.URL, err)
	}
	t.Logf("web: %s -> pod %s port %d", webFwd.Address, webFwd.Pod, webFwd.PodPort)

	// One of the seed's api pods, directly: agnhost answers its pod's name.
	var apiPod string
	for _, p := range readyPods(ctx, kube, "cogpit-demo", "api") {
		apiPod = p.Name
		break
	}
	if apiPod == "" {
		t.Fatal("no ready api pod in cogpit-demo")
	}
	podRef := model.ObjectRef{Kind: "pods", Namespace: "cogpit-demo", Name: apiPod}
	d.cached(podRef, nil)
	var podFwd engine.Forward
	d.must("pod.portForward", map[string]any{"ref": podRef, "container": "netexec", "port": 8080}, &podFwd)
	if got, err := httpGet(ctx, "http://"+podFwd.Address+"/hostname"); err != nil || strings.TrimSpace(got) != apiPod {
		t.Fatalf("GET /hostname through the pod forward: %q, %v", got, err)
	}

	// A service of our own: the forward follows its pod being replaced.
	echo := model.ObjectRef{Kind: "services", Namespace: ns, Name: "echo"}
	waitFor(ctx, t, "an echo pod ready", func() bool { return len(readyPods(ctx, kube, ns, "echo")) == 1 })
	d.cached(echo, nil)
	var echoFwd engine.Forward
	d.must("service.portForward", map[string]any{"ref": echo, "port": 80}, &echoFwd)
	first, err := httpGet(ctx, "http://"+echoFwd.Address+"/hostname")
	if err != nil || first != echoFwd.Pod {
		t.Fatalf("GET /hostname through the echo forward: %q, %v (forward %+v)", first, err, echoFwd)
	}
	if err := kube.CoreV1().Pods(ns).Delete(ctx, first, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	var second string
	waitFor(ctx, t, "the forward to reach the replacement pod", func() bool {
		got, err := httpGet(ctx, "http://"+echoFwd.Address+"/hostname")
		second = got
		return err == nil && got != "" && got != first
	})
	moved, err := s.ForwardOf(echoFwd.Ref)
	if err != nil || moved.Pod != second || moved.LocalPort != echoFwd.LocalPort || moved.Phase != engine.ForwardActive {
		t.Errorf("after the pod was replaced: %+v, %v (served by %s)", moved, err, second)
	}
	t.Logf("echo: %s moved from pod %s to %s", echoFwd.Address, first, second)

	if b := s.Backend(); b != nil && b.Conn != nil {
		stats := b.Conn.Counter().Snapshot()
		t.Logf("port-forward requests: GET (WebSocket) %d, POST (SPDY) %d",
			stats.ByVerbResource["get pods/portforward"], stats.ByVerbResource["create pods/portforward"])
		if stats.ByVerbResource["get pods/portforward"] == 0 {
			t.Error("the WebSocket dialer was not tried first")
		}
	}

	// Stop and delete close the local ports; leaving the cluster closes the rest.
	d.must("portforward.stop", map[string]any{"ref": podFwd.Ref}, nil)
	if _, err := httpGet(ctx, "http://"+podFwd.Address+"/hostname"); err == nil {
		t.Error("a stopped forward still answers")
	}
	d.must("portforward.remove", map[string]any{"ref": webFwd.Ref}, nil)
	if _, err := httpGet(ctx, "http://"+webFwd.Address+"/"); err == nil {
		t.Error("a deleted forward still answers")
	}
	s.Close()
	closed = true
	if _, err := httpGet(ctx, "http://"+echoFwd.Address+"/hostname"); err == nil {
		t.Error("a forward still answers after the session closed")
	}
}
