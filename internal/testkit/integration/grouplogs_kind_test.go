//go:build kind

package integration_test

import (
	"context"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// appLogsNamespace is TestKindApplicationLogs' namespace; it creates it, acts only inside it, and
// deletes it.
const appLogsNamespace = "cogpit-l-applogs"

// TestKindApplicationLogs (A81) runs an application's logs in one tab against the shared kind
// cluster: two Deployments labelled app.kubernetes.io/instance=shop, one talking pod each. Logs of
// the application merges both, each line named <workload>/<pod>/<container>; narrowed to one
// workload the other's lines stop, and the selection lists both workloads.
func TestKindApplicationLogs(t *testing.T) {
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
	kube, ns := conn.Kube(), appLogsNamespace
	freshNamespace(ctx, t, kube, ns)
	for _, name := range []string{"front", "back"} {
		if _, err := kube.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, Labels: map[string]string{kinds.LabelAppInstance: "shop"}},
			Spec: appsv1.DeploymentSpec{Replicas: int32p(1), Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}},
				Template: talkingTemplate(name, "app")},
		}, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	d := dispatcher{t: t, ctx: ctx, s: s}
	waitFor(ctx, t, "a ready pod each", func() bool {
		return len(readyPods(ctx, kube, ns, "front")) == 1 && len(readyPods(ctx, kube, ns, "back")) == 1
	})
	app := model.ObjectRef{Kind: kinds.ApplicationsKey, Namespace: ns, Name: "shop"}
	d.cached(app, func(o *unstructured.Unstructured) bool { return len(kinds.ApplicationWorkloads(o)) == 2 })
	var tab commands.DockResult
	d.must("application.logs", map[string]any{"ref": app}, &tab)
	id := tab.Tab.ID
	if tab.Tab.Title != "Logs: Application shop" {
		t.Errorf("tab %+v", tab.Tab)
	}

	stream, err := s.OpenLogs(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	merged := collect(stream)
	count := func(workload string) int {
		n := 0
		for _, l := range merged.snapshot() {
			if strings.HasPrefix(l, workload+"/"+workload+"-") && strings.Contains(strings.SplitN(l, " ", 2)[0], "/app") {
				n++
			}
		}
		return n
	}
	merged.waitLines(ctx, t, "lines of both workloads", func([]string) bool { return count("front") > 0 && count("back") > 0 })

	var src engine.LogSources
	d.must("logs.sources", map[string]any{"id": id}, &src)
	if !slices.Equal(src.Workloads, []string{"back", "front"}) && !slices.Equal(src.Workloads, []string{"front", "back"}) {
		t.Errorf("sources %+v", src)
	}

	// Narrowed to back: front's lines stop.
	d.must("logs.setSource", map[string]any{"id": id, "workload": "back"}, nil)
	time.Sleep(2 * time.Second) // lines in flight
	front, back := count("front"), count("back")
	merged.waitLines(ctx, t, "more of back's lines", func([]string) bool { return count("back") >= back+3 })
	if n := count("front"); n != front {
		t.Errorf("front wrote %d more lines after the tab narrowed to back", n-front)
	}
}
