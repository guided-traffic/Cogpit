//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"slices"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// workloadPodsNamespace is the shared kind cluster's namespace of the workload drawer's Pods test
// (D-40); it creates it, acts only inside it, and deletes it.
const workloadPodsNamespace = "cogpit-wpods-drawer"

// TestKindWorkloadDrawerPods opens a Deployment's drawer of two replicas through the engine on the
// kind cluster: object.get's Pods table lists its two pods, through its replica set, each linking
// to a pod the replica set controls and to its node; a pod deleted is replaced in the table by the
// one the replica set creates. It writes only in its own namespace.
func TestKindWorkloadDrawerPods(t *testing.T) {
	kubeconfig, contextName := kindEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kube, ns := conn.Kube(), workloadPodsNamespace
	freshNamespace(ctx, t, kube, ns)
	if _, err := kube.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "web"},
		Spec: appsv1.DeploymentSpec{Replicas: int32p(2), Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "web"}},
			Template: podTemplate("web", nil)},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}

	s := engine.New(engine.Options{Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}, SnapshotInterval: 20 * time.Millisecond})
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	dispatch := func(id string, params, out any) {
		t.Helper()
		raw, _ := json.Marshal(params)
		res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		if out != nil {
			b, _ := json.Marshal(res)
			if err := json.Unmarshal(b, out); err != nil {
				t.Fatalf("%s: %v", id, err)
			}
		}
	}
	deploy := model.ObjectRef{Kind: "deployments", Namespace: ns, Name: "web"}
	dispatch("ui.openView", map[string]any{"view": "deployments"}, nil)
	dispatch("ui.navigate", map[string]any{"ref": deploy}, nil)

	// running returns the table's pods that are Running, by name, and the table.
	running := func() ([]string, *model.DetailTable) {
		var m struct {
			Sections []model.DetailSection `json:"sections"`
		}
		dispatch("object.get", map[string]any{"ref": deploy, "noYaml": true}, &m)
		for _, sec := range m.Sections {
			if sec.Title != "Pods" || sec.Table == nil {
				continue
			}
			var names []string
			st := slices.Index(sec.Table.IDs, "status")
			for _, r := range sec.Table.Cells {
				if r[st].Text == "Running" {
					names = append(names, r[0].Text)
				}
			}
			return names, sec.Table
		}
		return nil, nil
	}
	// fromTheReplicaSet returns the names of the Deployment's pods as the API server lists them.
	fromTheReplicaSet := func() []string {
		pods, err := kube.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{LabelSelector: "app=web"})
		if err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, p := range pods.Items {
			if p.DeletionTimestamp == nil && p.Status.Phase == "Running" && len(p.OwnerReferences) == 1 && p.OwnerReferences[0].Kind == "ReplicaSet" {
				names = append(names, p.Name)
			}
		}
		slices.Sort(names)
		return names
	}
	var names []string
	var tbl *model.DetailTable
	if !testkit.WaitFor(ctx, func() bool {
		names, tbl = running()
		return len(names) == 2 && tbl != nil && len(tbl.Rows) == 2 && slices.Equal(names, fromTheReplicaSet())
	}) {
		t.Fatalf("the Deployment's Pods: running %q of %+v; the API server's %q", names, tbl, fromTheReplicaSet())
	}
	node := slices.Index(tbl.IDs, "node")
	for _, r := range tbl.Cells {
		if r[0].Link == nil || r[0].Link.Kind != "pods" || r[0].Link.Namespace != ns || r[node].Link == nil || r[node].Link.Kind != "nodes" {
			t.Errorf("row %+v: want links to the pod and its node", r)
		}
	}
	t.Logf("the Deployment's Pods: %q", names)

	// A pod deleted: the replica set replaces it, and the table follows.
	gone := names[0]
	if err := kube.CoreV1().Pods(ns).Delete(ctx, gone, metav1.DeleteOptions{}); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool {
		names, tbl = running()
		return len(names) == 2 && !slices.Contains(names, gone) && len(tbl.Rows) == 2
	}) {
		t.Fatalf("after deleting %s the table shows %q (%d rows)", gone, names, len(tbl.Rows))
	}
	dispatch("ui.closeDrawer", map[string]any{}, nil)
}
