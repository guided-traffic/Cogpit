//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// selectionJobNamespace is the namespace of TestKindDeleteSelectionAsAJob: it creates it, acts only
// inside it, and deletes it.
const selectionJobNamespace = "cogpit-w-selection-job"

// TestKindDeleteSelectionAsAJob (G4): Delete of 30 ConfigMaps selected through the engine runs as a
// background job on a real API server, at the connection's 5 requests per second: the job's tab
// shows the deletes as they go; job.cancel stops it with the step under way finished and the rest
// not attempted, which the API server confirms (each step's state is what happened to its
// ConfigMap); a second Delete of what is left finishes the job's work.
func TestKindDeleteSelectionAsAJob(t *testing.T) {
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
	kube, ns := conn.Kube(), selectionJobNamespace
	freshNamespace(ctx, t, kube, ns)
	const n = 30
	for i := range n {
		if _, err := kube.CoreV1().ConfigMaps(ns).Create(ctx, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: fmt.Sprintf("job-%02d", i)}, Data: map[string]string{"a": "1"},
		}, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	s := engine.New(engine.Options{
		Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}},
		Confirmer: engine.ConfirmFunc(func(context.Context, engine.ConfirmRequest) (bool, error) { return true, nil }),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	must := func(id string, params any) any {
		t.Helper()
		raw, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}
		res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginGUI})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		return res
	}
	job := func(id string) *engine.JobTab {
		if tab, ok := s.UIState().DockTab(id); ok {
			return tab.Job
		}
		return nil
	}
	selectShown := func(want int) {
		t.Helper()
		wctx, wcancel := context.WithTimeout(ctx, time.Minute)
		defer wcancel()
		if !testkit.WaitFor(wctx, func() bool {
			tbl := s.Table("configmaps")
			return tbl != nil && tbl.Synced && len(tbl.Rows) == want
		}) {
			t.Fatalf("the view never showed %d configmaps", want)
		}
		must("ui.select", map[string]any{"all": true, "noDrawer": true})
	}

	must("ui.openView", map[string]any{"view": "configmaps"})
	must("ui.setNamespace", map[string]any{"namespaces": []string{ns}})
	must("ui.setFilter", map[string]any{"filter": "job-"}) // not the namespace's kube-root-ca.crt
	selectShown(n)
	started := must("object.deleteMany", map[string]any{"refs": s.UIState().Selected}).(commands.ManyJobResult)
	if len(started.Targets) != n || started.Job.Type != engine.DockJob {
		t.Fatalf("started %+v", started)
	}

	// Cancelled once twelve are deleted: at most the request under way still goes through.
	waitFor(ctx, t, "twelve deletes", func() bool { j := job(started.Job.ID); return j != nil && j.Counts.Done >= 12 })
	cancelled := must("job.cancel", map[string]any{"id": started.Job.ID}).(engine.DockTab)
	j := cancelled.Job
	if j == nil || j.State != engine.JobCancelled || j.Counts.Done+j.Counts.Cancelled != n || j.Counts.Cancelled == 0 {
		t.Fatalf("after job.cancel %+v", j)
	}
	t.Logf("cancelled after %d of %d: %s", j.Counts.Done, n, j.Outcome)
	left := 0
	for _, st := range j.Steps {
		_, err := kube.CoreV1().ConfigMaps(ns).Get(ctx, st.Ref.Name, metav1.GetOptions{})
		switch st.State {
		case engine.JobDone:
			if !apierrors.IsNotFound(err) {
				t.Errorf("%s done but still there: %v", st.Ref.Name, err)
			}
		case engine.JobCancelled:
			left++
			if err != nil {
				t.Errorf("%s not attempted but gone: %v", st.Ref.Name, err)
			}
		default:
			t.Errorf("%s: step %s after the job ended", st.Ref.Name, st.State)
		}
	}

	// What is left, deleted to the end (a job again when more than ten are left).
	selectShown(left)
	res := must("object.deleteMany", map[string]any{"refs": s.UIState().Selected})
	if again, ok := res.(commands.ManyJobResult); ok {
		waitFor(ctx, t, "the second job's end", func() bool { j := job(again.Job.ID); return j != nil && j.State.Ended() })
		if j := job(again.Job.ID); j.State != engine.JobDone {
			t.Errorf("the second job %+v", j)
		}
	}
	list, err := kube.CoreV1().ConfigMaps(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, cm := range list.Items {
		if cm.Name != "kube-root-ca.crt" {
			t.Errorf("configmap %s left", cm.Name)
		}
	}
}
