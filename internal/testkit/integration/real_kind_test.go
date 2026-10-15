//go:build kind

package integration_test

// Tests against a real cluster someone names, not kind: each runs only when COGPIT_REAL_KUBECONFIG
// names a kubeconfig file and COGPIT_REAL_CONTEXT one of its contexts, and is skipped otherwise. They
// never run in CI: make test-kind runs the TestKind* functions, these are TestReal* (make test-real
// runs them). Each test writes only into namespaces of its own (cogpit-real-*), deleted at the end;
// the node, Helm and Flux tests act only on what an environment variable of their own names, and
// every write goes through the engine with a confirming hook, as the window's user would confirm it.
// The tests read the kubeconfig they are given and nothing of ~/.kube; give them a HOME of their own
// too (make test-real does).
//
//	COGPIT_REAL_IDLE            how long TestRealDiscoveryAndBudgets leaves its views idle (P9; default 30s)
//	COGPIT_REAL_CUSTOM_VIEW     a custom resource kind key it opens besides the built-in views
//	COGPIT_REAL_CORDON_NODE     the node TestRealNodeCordon cordons and uncordons
//	COGPIT_REAL_DRAIN_NODE      the node TestRealNodeDrain drains (then uncordons)
//	COGPIT_REAL_DRAIN_EMPTYDIR  1: the drain also evicts pods with emptyDir data (kubectl's --delete-emptydir-data)
//	COGPIT_REAL_DRAIN_TIMEOUT   the drain's timeout in seconds (default: Cogpit's, 300)
//	COGPIT_REAL_SHELL_NODE      the node TestRealNodeShell opens a node shell on
//	COGPIT_REAL_HELM_REPO       a public Helm repository's URL; COGPIT_REAL_HELM_CHART a chart of it
//	COGPIT_REAL_FLUX            Flux objects to reconcile, suspend and resume: <kind key>/<namespace>/<name>, comma-separated

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	helmrepo "helm.sh/helm/v4/pkg/repo/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// realEnv returns the real cluster's kubeconfig and context, or skips the test.
func realEnv(t *testing.T) (kubeconfig, contextName string) {
	t.Helper()
	kubeconfig, contextName = os.Getenv("COGPIT_REAL_KUBECONFIG"), os.Getenv("COGPIT_REAL_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_REAL_KUBECONFIG and COGPIT_REAL_CONTEXT are not set: these tests run only against a real cluster someone names (make test-real)")
	}
	return kubeconfig, contextName
}

// realVar returns an environment variable the test needs, or skips the test.
func realVar(t *testing.T, name string) string {
	t.Helper()
	v := os.Getenv(name)
	if v == "" {
		t.Skip(name + " is not set")
	}
	return v
}

// realSession connects an engine to the real cluster with the window's user confirming everything
// (the recordingConfirmer), and the given options besides.
func realSession(ctx context.Context, t *testing.T, opts engine.Options) (*engine.Session, *recordingConfirmer, *drainEvents) {
	t.Helper()
	kubeconfig, contextName := realEnv(t)
	confirm := &recordingConfirmer{}
	opts.Connector = &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}
	opts.Confirmer = confirm
	if opts.SnapshotInterval == 0 {
		opts.SnapshotInterval = 20 * time.Millisecond
	}
	s := engine.New(opts)
	t.Cleanup(s.Close)
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	ev := &drainEvents{}
	s.Subscribe(ev.add)
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	// Custom kinds (Flux's among them) are views once their definitions are listed.
	waitFor(ctx, t, "the custom resource definitions to settle", s.Backend().CustomKindsSettled)
	return s, confirm, ev
}

// nonMetrics drops the metrics requests (the metrics APIs and Prometheus through the service
// proxy), which budget P12 counts, from a diff.
func nonMetrics(d map[string]int64) map[string]int64 {
	out := map[string]int64{}
	for k, v := range d {
		if strings.Contains(k, "metrics.k8s.io") || strings.Contains(k, "services/proxy") {
			continue
		}
		out[k] = v
	}
	return out
}

func verbCount(d map[string]int64, verb string) int64 {
	var n int64
	for k, v := range d {
		if k == verb || strings.HasPrefix(k, verb+" ") {
			n += v
		}
	}
	return n
}

// TestRealDiscoveryAndBudgets measures, against a real API server, how long connecting and the
// custom resource definitions take, and the API-load budgets P9 (a view left idle: no list, only
// its watches; COGPIT_REAL_IDLE long, 10 minutes for the budget as written), P10 (a view reopened
// within the warm period: no list, its rows at once) and P11 (an agent's ui_state, resources_find
// and ui_navigate on a warm cache: no request). It only reads.
func TestRealDiscoveryAndBudgets(t *testing.T) {
	realEnv(t)
	idle := 30 * time.Second
	if v := os.Getenv("COGPIT_REAL_IDLE"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			t.Fatalf("COGPIT_REAL_IDLE: %v", err)
		}
		idle = d
	}
	ctx, cancel := context.WithTimeout(context.Background(), idle+5*time.Minute)
	defer cancel()

	start := time.Now()
	s, _, _ := realSession(ctx, t, engine.Options{})
	connected := time.Since(start)
	b := s.Backend()
	waitFor(ctx, t, "the custom resource definitions to settle", b.CustomKindsSettled)
	settled := time.Since(start)
	custom := len(b.AllProfiles()) - len(b.Profiles)
	counter := b.Conn.Counter()
	t.Logf("discovery: connected after %v (Kubernetes %s), the custom kinds settled after %v: %d built-in kinds served, %d custom kinds",
		connected.Round(time.Millisecond), b.Conn.ServerVersion(), settled.Round(time.Millisecond), len(b.Profiles), custom)
	t.Logf("requests of connecting: %v", counter.Snapshot().ByVerbResource)

	d := dispatcher{t: t, ctx: ctx, s: s}
	views := []string{"customresourcedefinitions", "nodes", "services", "deployments", "events", "pods"}
	if v := os.Getenv("COGPIT_REAL_CUSTOM_VIEW"); v != "" {
		views = append([]string{v}, views...)
	}
	for _, view := range views {
		opened, at := time.Now(), counter.Snapshot()
		d.must("ui.openView", map[string]any{"view": view}, nil)
		waitFor(ctx, t, "the view "+view+" to fill", func() bool {
			tbl := s.Table(view)
			return tbl != nil && tbl.Synced
		})
		t.Logf("view %s: %d rows after %v, requests %v", view, s.Table(view).Total, time.Since(opened).Round(time.Millisecond),
			diff(at, counter.Snapshot()))
	}

	// quiet waits until the connection has sent nothing for a second: every view's watch open.
	quiet := func() cluster.RequestStats {
		t.Helper()
		var at cluster.RequestStats
		waitFor(ctx, t, "the requests to settle", func() bool {
			at = counter.Snapshot()
			time.Sleep(time.Second)
			return counter.Snapshot().Total == at.Total
		})
		return at
	}

	// P11, while the cache is warm (the views above just opened): an agent finds a Service by its
	// cluster IP and shows it.
	services, err := s.Objects(ctx, "services", nil)
	if err != nil || len(services) == 0 {
		t.Fatalf("services: %v (%d)", err, len(services))
	}
	var target *unstructured.Unstructured
	for _, svc := range services {
		if ip, _, _ := unstructured.NestedString(svc.Object, "spec", "clusterIP"); ip != "" && ip != "None" {
			target = svc
			break
		}
	}
	if target == nil {
		t.Fatal("no Service with a cluster IP")
	}
	ip, _, _ := unstructured.NestedString(target.Object, "spec", "clusterIP")
	agent := func(id string, params any) any {
		t.Helper()
		raw, _ := json.Marshal(params)
		res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		return res
	}
	before := quiet()
	agent("ui.state", struct{}{})
	found := agent("resources.find", map[string]any{"kind": "services", "field": "clusterIP", "value": ip}).(commands.FindResult)
	if found.Count != 1 {
		t.Fatalf("resources.find %s: %+v", ip, found)
	}
	agent("ui.navigate", map[string]any{"ref": found.Matches[0].Ref})
	if _, err := s.DetailWait(ctx, found.Matches[0].Ref); err != nil {
		t.Fatal(err)
	}
	agentCost := diff(before, quiet())
	t.Logf("P11: ui_state, resources_find and ui_navigate for %s: %v", found.Matches[0].Ref, agentCost)
	if n := verbCount(agentCost, cluster.VerbList) + verbCount(agentCost, cluster.VerbWatchList) + verbCount(agentCost, cluster.VerbWatch); n != 0 {
		t.Errorf("P11: %d reads: %v", n, agentCost)
	}

	// P10: away to another view and back, within the warm period.
	d.must("ui.openView", map[string]any{"view": "services"}, nil)
	before = quiet()
	back := time.Now()
	d.must("ui.openView", map[string]any{"view": "pods"}, nil)
	waitFor(ctx, t, "the Pods view's rows", func() bool {
		tbl := s.Table("pods")
		return tbl != nil && tbl.Synced && tbl.Total > 0
	})
	shown := time.Since(back)
	reopen := diff(before, quiet())
	t.Logf("P10: Pods reopened, rows after %v, requests %v", shown.Round(time.Microsecond), reopen)
	if n := verbCount(nonMetrics(reopen), cluster.VerbList) + verbCount(nonMetrics(reopen), cluster.VerbWatchList); n != 0 {
		t.Errorf("P10: reopening cost %d lists: %v", n, reopen)
	}
	if shown > 100*time.Millisecond {
		t.Errorf("P10: rows after %v, want at most 100 ms", shown)
	}

	// P9, last: the Pods view open and idle. client-go renews a watch the API server ends (every 5
	// to 10 minutes): a watch request, never a list. The Pods list's metrics are P12's.
	before = quiet()
	time.Sleep(idle)
	idleCost := diff(before, counter.Snapshot())
	t.Logf("P9: %v idle on the Pods view: %v", idle, idleCost)
	if n := verbCount(nonMetrics(idleCost), cluster.VerbList) + verbCount(nonMetrics(idleCost), cluster.VerbWatchList); n != 0 {
		t.Errorf("P9: %d list requests while idle: %v", n, idleCost)
	}
}

// realNamespace makes a namespace of the test's own, cogpit-real-<suffix>, deleted at the end.
func realNamespace(ctx context.Context, t *testing.T, s *engine.Session, suffix string) string {
	t.Helper()
	ns := "cogpit-real-" + suffix
	freshNamespace(ctx, t, s.Backend().Conn.Kube(), ns)
	return ns
}

// ticker is a pod that prints a line every second, from busybox named in full (a cluster's runtime
// may refuse short names).
func ticker(ns, name string) *corev1.Pod {
	no, yes, uid, grace := false, true, int64(65534), int64(1)
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: name, Labels: map[string]string{"app": name}},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &grace,
			SecurityContext:               &corev1.PodSecurityContext{RunAsNonRoot: &yes, RunAsUser: &uid},
			Containers: []corev1.Container{{
				Name: "tick", Image: "docker.io/library/busybox:1.37.0", ImagePullPolicy: corev1.PullIfNotPresent,
				Command: []string{"sh", "-c", "trap 'exit 0' TERM; i=0; while true; do i=$((i+1)); echo tick-$i; sleep 1; done"},
				Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("5m"), corev1.ResourceMemory: resource.MustParse("8Mi"),
				}},
				SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &no, Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
			}},
		},
	}
}

// TestRealWorkloadActions drives the actions on workloads of its own through the engine, against a
// real cluster: Scale and Restart of a Deployment (its view following), port forwarding to a
// Service and a pod, a pod's logs, an attach to its main process, a shell, and Delete. Edit, create
// with diff and dry run, and the conflict are TestKindEditCreateConflictDelete's, which runs
// unchanged against a real cluster too.
func TestRealWorkloadActions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	s, confirm, _ := realSession(ctx, t, engine.Options{})
	ns := realNamespace(ctx, t, s, "work")
	kube := s.Backend().Conn.Kube()
	dep, svc := echoObjects(ns)
	if _, err := kube.AppsV1().Deployments(ns).Create(ctx, dep, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.CoreV1().Services(ns).Create(ctx, svc, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	if _, err := kube.CoreV1().Pods(ns).Create(ctx, ticker(ns, "ticker"), metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
	}
	d := dispatcher{t: t, ctx: ctx, s: s}
	d.must("ui.openView", map[string]any{"view": "deployments"}, nil)
	echo := model.ObjectRef{Kind: "deployments", Namespace: ns, Name: "echo"}
	podsCell := func() string {
		tbl := s.Table("deployments")
		if tbl == nil {
			return ""
		}
		p, _ := s.Backend().Profile("deployments")
		col := p.ColumnIndex("pods")
		for _, r := range tbl.Rows {
			if r.Ref.Namespace == ns && r.Ref.Name == "echo" {
				return model.CellText(p.Columns[col], r.Cells[col], time.Now()) + " " + string(r.Status)
			}
		}
		return ""
	}
	waitFor(ctx, t, "the echo Deployment ready in its view", func() bool { return podsCell() == "1/1 ok" })

	// Scale to 2: the view follows.
	d.cached(echo, nil)
	d.must("deployment.scale", map[string]any{"ref": echo, "replicas": 2}, nil)
	waitFor(ctx, t, "two ready echo pods in the view", func() bool { return podsCell() == "2/2 ok" })

	// Restart: every pod replaced, the view ok again.
	old := map[string]bool{}
	for _, p := range readyPods(ctx, kube, ns, "echo") {
		old[p.Name] = true
	}
	d.must("deployment.restart", map[string]any{"ref": echo}, nil)
	waitFor(ctx, t, "the restart to replace both pods", func() bool {
		ready := readyPods(ctx, kube, ns, "echo")
		if len(ready) != 2 {
			return false
		}
		for _, p := range ready {
			if old[p.Name] {
				return false
			}
		}
		return podsCell() == "2/2 ok"
	})
	if req, n := confirm.last(); n == 0 || !strings.Contains(req.Message(), "echo") {
		t.Errorf("the restart's confirmation: %+v (%d asked)", req, n)
	}

	// Port forwarding: the Service, then a pod directly.
	svcRef := model.ObjectRef{Kind: "services", Namespace: ns, Name: "echo"}
	d.cached(svcRef, nil)
	var fwd engine.Forward
	d.must("service.portForward", map[string]any{"ref": svcRef, "port": 80}, &fwd)
	var answered string
	waitFor(ctx, t, "an answer through the Service's forward", func() bool {
		got, err := httpGet(ctx, "http://"+fwd.Address+"/hostname")
		answered = got
		return err == nil && strings.HasPrefix(got, "echo-")
	})
	t.Logf("service forward %s answered by %s", fwd.Address, answered)
	podRef := model.ObjectRef{Kind: "pods", Namespace: ns, Name: answered}
	d.cached(podRef, nil)
	var podFwd engine.Forward
	d.must("pod.portForward", map[string]any{"ref": podRef, "container": "netexec", "port": 8080}, &podFwd)
	if got, err := httpGet(ctx, "http://"+podFwd.Address+"/hostname"); err != nil || strings.TrimSpace(got) != answered {
		t.Errorf("GET /hostname through the pod forward: %q, %v", got, err)
	}
	d.must("portforward.remove", map[string]any{"ref": podFwd.Ref}, nil)
	d.must("portforward.remove", map[string]any{"ref": fwd.Ref}, nil)

	// Logs and attach of the ticker, a shell in the echo pod.
	tick := model.ObjectRef{Kind: "pods", Namespace: ns, Name: "ticker"}
	d.cached(tick, func(o *unstructured.Unstructured) bool {
		phase, _, _ := unstructured.NestedString(o.Object, "status", "phase")
		return phase == string(corev1.PodRunning)
	})
	var logsTab commands.DockResult
	d.must("pod.logs", map[string]any{"ref": tick}, &logsTab)
	logs, err := s.OpenLogs(ctx, logsTab.Tab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out := readUntil(t, logs, "tick-3", time.Minute); !strings.Contains(out, "tick-3") {
		t.Errorf("logs: %q", out)
	}
	_ = logs.Close()
	var attachTab commands.DockResult
	d.must("pod.attach", map[string]any{"ref": tick}, &attachTab)
	attach, err := s.OpenTerminal(ctx, attachTab.Tab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if out := readUntil(t, attach, "tick-", time.Minute); !strings.Contains(out, "tick-") {
		t.Errorf("attach: %q", out)
	}
	_ = attach.Close()
	var shellTab commands.DockResult
	d.must("pod.shell", map[string]any{"ref": podRef}, &shellTab)
	shell, err := s.OpenTerminal(ctx, shellTab.Tab.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(shell, "echo cogpit-$((40+2))\n"); err != nil {
		t.Fatal(err)
	}
	if out := readUntil(t, shell, "cogpit-42", time.Minute); !strings.Contains(out, "cogpit-42") {
		t.Errorf("shell: %q", out)
	}
	_ = shell.Close()

	// Delete the ticker; then scale to zero, which asks as destructive.
	d.must("object.delete", map[string]any{"ref": tick}, nil)
	waitFor(ctx, t, "the ticker to be gone", func() bool {
		_, err := kube.CoreV1().Pods(ns).Get(ctx, "ticker", metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	})
	d.must("deployment.scale", map[string]any{"ref": echo, "replicas": 0}, nil)
	if req, _ := confirm.last(); req.Danger != engine.DangerDestructive {
		t.Errorf("scaling to zero asked as %q", req.Danger)
	}
	waitFor(ctx, t, "the echo Deployment at zero in its view", func() bool { return strings.HasPrefix(podsCell(), "0/0") })
}

// nodeRow returns a node's row in the Nodes view: its Conditions text and status.
func nodeRow(s *engine.Session, name string) (conditions string, status model.Status) {
	tbl := s.Table("nodes")
	p, ok := s.Backend().Profile("nodes")
	if tbl == nil || !ok {
		return "", ""
	}
	col := p.ColumnIndex("conditions")
	for _, r := range tbl.Rows {
		if r.Ref.Name == name {
			return model.CellText(p.Columns[col], r.Cells[col], time.Now()), r.Status
		}
	}
	return "", ""
}

// uncordonAtEnd makes sure the node is schedulable when the test ends, whatever happened.
func uncordonAtEnd(t *testing.T, s *engine.Session, node string) {
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		patch := []byte(`{"spec":{"unschedulable":null}}`)
		if _, err := s.Backend().Conn.Kube().CoreV1().Nodes().Patch(ctx, node, types.MergePatchType, patch, metav1.PatchOptions{}); err != nil {
			t.Errorf("uncordon %s at the end: %v", node, err)
		}
	})
}

// TestRealNodeCordon cordons and uncordons the node COGPIT_REAL_CORDON_NODE names through the
// engine, and checks the Nodes view: "SchedulingDisabled Ready" while cordoned (captured in Lens),
// "Ready" after.
func TestRealNodeCordon(t *testing.T) {
	realEnv(t)
	node := realVar(t, "COGPIT_REAL_CORDON_NODE")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	s, confirm, _ := realSession(ctx, t, engine.Options{})
	uncordonAtEnd(t, s, node)
	d := dispatcher{t: t, ctx: ctx, s: s}
	d.must("ui.openView", map[string]any{"view": "nodes"}, nil)
	ref := model.ObjectRef{Kind: "nodes", Name: node}
	d.cached(ref, nil)
	waitFor(ctx, t, "node "+node+" Ready in the view", func() bool { c, _ := nodeRow(s, node); return c == "Ready" })
	d.must("node.cordon", map[string]any{"ref": ref}, nil)
	waitFor(ctx, t, "the cordon in the view", func() bool { c, _ := nodeRow(s, node); return c == "SchedulingDisabled Ready" })
	if _, n := confirm.last(); n != 0 { // mutating, not destructive: the window's user is not asked (D-2)
		t.Errorf("the cordon asked %d confirmations: %v", n, confirm.messages())
	}
	got, err := s.Backend().Conn.Kube().CoreV1().Nodes().Get(ctx, node, metav1.GetOptions{})
	if err != nil || !got.Spec.Unschedulable {
		t.Fatalf("the node is not cordoned: %v", err)
	}
	d.must("node.uncordon", map[string]any{"ref": ref}, nil)
	waitFor(ctx, t, "the uncordon in the view", func() bool { c, _ := nodeRow(s, node); return c == "Ready" })
	_, status := nodeRow(s, node)
	t.Logf("node %s cordoned and uncordoned, row status %q", node, status)
}

// TestRealNodeDrain drains the node COGPIT_REAL_DRAIN_NODE names with Cogpit's crash-safe drain,
// reports what Cogpit tells the user on the way (a refusal, an eviction a PodDisruptionBudget holds
// up, the pods evicted), and uncordons it. It waits for nothing the cluster's own controllers do
// afterwards: the caller checks the cluster's health before the next node action.
func TestRealNodeDrain(t *testing.T) {
	realEnv(t)
	node := realVar(t, "COGPIT_REAL_DRAIN_NODE")
	timeout := 0
	if v := os.Getenv("COGPIT_REAL_DRAIN_TIMEOUT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatalf("COGPIT_REAL_DRAIN_TIMEOUT: %v", err)
		}
		timeout = n
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()
	s, confirm, events := realSession(ctx, t, engine.Options{Drains: engine.NewDrains(engine.NewMemoryDrainJournal())})
	uncordonAtEnd(t, s, node)
	d := dispatcher{t: t, ctx: ctx, s: s}
	ref := model.ObjectRef{Kind: "nodes", Name: node}
	d.must("ui.openView", map[string]any{"view": "nodes"}, nil)
	d.cached(ref, nil)

	params := map[string]any{"ref": ref}
	if os.Getenv("COGPIT_REAL_DRAIN_EMPTYDIR") == "1" {
		params["deleteEmptyDirData"] = true
	}
	if timeout > 0 {
		params["timeoutSeconds"] = timeout
	}
	started := time.Now()
	var res commands.DrainResult
	err := d.run("node.drain", params, &res)
	took := time.Since(started).Round(time.Second)
	req, _ := confirm.last()
	t.Logf("the confirmation: %s", req.Message())
	for _, ev := range events.all() {
		t.Logf("event %s %s: %s", ev.Type, ev.Level, ev.Message)
	}
	pending, perr := s.PendingDrains(false)
	if perr != nil {
		t.Fatal(perr)
	}
	if err != nil {
		t.Logf("the drain ended after %v with: %v", took, err)
		if len(pending) != 1 || pending[0].Node != node {
			t.Errorf("an unfinished drain is not offered: %+v", pending)
		} else {
			t.Logf("offered as unfinished: status %s, node %s", pending[0].Status, pending[0].NodeState)
		}
		got, gerr := s.Backend().Conn.Kube().CoreV1().Nodes().Get(ctx, node, metav1.GetOptions{})
		if gerr != nil || !got.Spec.Unschedulable {
			t.Errorf("after a failed drain the node should stay cordoned: %v", gerr)
		}
	} else {
		t.Logf("drained after %v: %d pods evicted, left: %s", took, res.Evicted, res.Left)
		if len(pending) != 0 {
			t.Errorf("a finished drain is still offered: %+v", pending)
		}
	}
	waitFor(ctx, t, "the drain in the view", func() bool {
		c, _ := nodeRow(s, node)
		return strings.HasPrefix(c, "SchedulingDisabled")
	})
	d.must("node.uncordon", map[string]any{"ref": ref}, nil)
	waitFor(ctx, t, "the uncordon in the view", func() bool { c, _ := nodeRow(s, node); return c == "Ready" })
	if pending, _ := s.PendingDrains(false); len(pending) != 0 {
		t.Errorf("the uncordon did not forget the drain: %+v", pending)
	}
}

// TestRealNodeShell opens a node shell on COGPIT_REAL_SHELL_NODE, runs a command in it, closes it
// and checks that its pod is gone.
func TestRealNodeShell(t *testing.T) {
	realEnv(t)
	node := realVar(t, "COGPIT_REAL_SHELL_NODE")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	s, confirm, _ := realSession(ctx, t, engine.Options{ClusterSettings: engine.NewClusterSettingsStore("")})
	kube := s.Backend().Conn.Kube()
	d := dispatcher{t: t, ctx: ctx, s: s}
	ref := model.ObjectRef{Kind: "nodes", Name: node}
	d.must("ui.openView", map[string]any{"view": "nodes"}, nil)
	d.cached(ref, nil)
	var res commands.DockResult
	d.must(engine.NodeShellCommand, map[string]any{"ref": ref}, &res)
	pod := res.Tab.NodeShell.Pod
	t.Cleanup(func() {
		cctx, ccancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer ccancel()
		_ = kube.CoreV1().Pods(engine.NodeShellNamespace).Delete(cctx, pod, metav1.DeleteOptions{})
	})
	if req, _ := confirm.last(); !strings.Contains(req.Message(), "on node "+node) {
		t.Errorf("the confirmation: %s", req.Message())
	}
	sess, err := s.OpenTerminal(ctx, res.Tab.ID)
	if err != nil {
		t.Fatal(err)
	}
	out := collectShell(sess)
	if _, err := io.WriteString(sess, "echo ON-$(hostname)-DONE\r"); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool { return strings.Contains(out.String(), "-DONE") && strings.Contains(out.String(), "ON-") }) {
		t.Fatalf("the node shell did not answer: %q", out.String())
	}
	_ = sess.Close()
	waitFor(ctx, t, "the node shell's pod to be gone", func() bool {
		_, err := kube.CoreV1().Pods(engine.NodeShellNamespace).Get(ctx, pod, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	})
	t.Logf("node shell pod %s on %s gone after the tab closed", pod, node)
}

// TestRealHelm installs a public chart (COGPIT_REAL_HELM_REPO, COGPIT_REAL_HELM_CHART) into a
// namespace of its own through the engine, from Helm's repositories file in a temporary folder,
// at the repository's second newest version; upgrades it to the newest with other values; rolls it
// back to the first revision; uninstalls it. It never touches another release.
func TestRealHelm(t *testing.T) {
	_, contextName := realEnv(t)
	repoURL, chart := realVar(t, "COGPIT_REAL_HELM_REPO"), realVar(t, "COGPIT_REAL_HELM_CHART")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	helmFile := testkit.NewHelmRepoFile(t)
	helmFile.Write(&helmrepo.Entry{Name: "real", URL: repoURL})
	s, confirm, _ := realSession(ctx, t, engine.Options{
		Helm:      engine.NewHelmOps(engine.NewMemoryHelmJournal()),
		HelmRepos: engine.NewHelmRepos(helmFile.Path, t.TempDir()),
	})
	ns := realNamespace(ctx, t, s, "helm")
	d := dispatcher{t: t, ctx: ctx, s: s}
	var infos []engine.HelmRepoInfo
	d.must("helm.repo.update", map[string]any{"names": []string{"real"}}, &infos)
	if len(infos) != 1 || infos[0].Error != "" {
		t.Fatalf("the repository: %+v", infos)
	}
	chartRef := engine.HelmChartRef(contextName, "real", chart)
	waitFor(ctx, t, "the chart listed", func() bool { _, err := s.Lookup(chartRef); return err == nil })
	if _, err := s.Objects(ctx, kinds.HelmChartsKey, nil); err != nil {
		t.Fatal(err)
	}
	var tab commands.EditResult
	d.must("helm.chart.installTab", map[string]any{"ref": chartRef}, &tab)
	versions := tab.Tab.HelmInstall.Versions
	if len(versions) < 2 {
		t.Fatalf("the chart has %d versions", len(versions))
	}
	newest, older := versions[0], versions[1]
	t.Logf("chart %s: %d versions, installing %s then %s (archive verified: %v)", chart, len(versions), older, newest, tab.Tab.HelmInstall.Verified)
	if _, err := s.Objects(ctx, helm.ReleaseKind, nil); err != nil {
		t.Fatal(err)
	}

	const name = "cogpit-real"
	ref := engine.HelmReleaseRef(contextName, ns, name)
	d.must("helm.chart.install", map[string]any{"repository": "real", "chart": chart, "version": older,
		"values": "replicaCount: 1\n", "namespace": ns, "name": name}, nil)
	t.Cleanup(func() { _ = d.run("helm.release.uninstall", map[string]any{"ref": ref}, nil) })
	at := func(rev int, version string) {
		t.Helper()
		waitFor(ctx, t, "the release at revision "+strconv.Itoa(rev)+" with "+version, func() bool {
			obj, err := s.Lookup(ref)
			if err != nil {
				return false
			}
			_, v := helm.ReleaseChart(obj)
			return helm.ReleaseRevision(obj) == rev && helm.ReleaseStatus(obj) == "deployed" && v == version
		})
	}
	at(1, older)
	d.must("helm.release.upgrade", map[string]any{"ref": ref, "repository": "real", "version": newest,
		"values": "replicaCount: 2\n", "revision": 1}, nil)
	at(2, newest)
	d.must("helm.release.rollback", map[string]any{"ref": ref, "revision": 1}, nil)
	at(3, older)
	var history commands.ReleaseHistory
	d.must("helm.release.history", map[string]any{"ref": ref}, &history)
	t.Logf("history: %+v", history.History)
	d.must("helm.release.uninstall", map[string]any{"ref": ref}, nil)
	waitFor(ctx, t, "the release to be gone", func() bool { _, err := s.Lookup(ref); return err != nil })
	for _, m := range confirm.messages() {
		t.Logf("confirmed: %s", m)
	}
}

// TestRealFlux reconciles, suspends and resumes the Flux objects COGPIT_REAL_FLUX names through the
// engine, and resumes every one it suspended whatever happens.
func TestRealFlux(t *testing.T) {
	realEnv(t)
	list := realVar(t, "COGPIT_REAL_FLUX")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	s, _, _ := realSession(ctx, t, engine.Options{})
	for _, item := range strings.Split(list, ",") {
		parts := strings.Split(strings.TrimSpace(item), "/")
		if len(parts) != 3 {
			t.Fatalf("COGPIT_REAL_FLUX: %q is not <kind key>/<namespace>/<name>", item)
		}
		ref := model.ObjectRef{Kind: parts[0], Namespace: parts[1], Name: parts[2]}
		t.Run(item, func(t *testing.T) {
			d := dispatcher{t: t, ctx: ctx, s: s}
			d.must("ui.openView", map[string]any{"view": ref.Kind}, nil)
			d.cached(ref, nil)
			t.Cleanup(func() { // whatever happened: resumed
				obj, err := s.Object(context.Background(), ref)
				if err == nil && kinds.IsFluxSuspended(obj) {
					if err := d.run("flux.resume", map[string]any{"ref": ref}, nil); err != nil {
						t.Errorf("resume %s at the end: %v", ref, err)
					}
				}
			})
			d.must("flux.reconcile", map[string]any{"ref": ref}, nil)
			d.cached(ref, func(o *unstructured.Unstructured) bool {
				asked := o.GetAnnotations()["reconcile.fluxcd.io/requestedAt"]
				handled, _, _ := unstructured.NestedString(o.Object, "status", "lastHandledReconcileAt")
				return asked != "" && asked == handled
			})
			d.must("flux.suspend", map[string]any{"ref": ref}, nil)
			d.cached(ref, func(o *unstructured.Unstructured) bool { return kinds.FluxStateOf(ref.Kind, o) == kinds.FluxSuspended })
			d.must("flux.resume", map[string]any{"ref": ref}, nil)
			d.cached(ref, func(o *unstructured.Unstructured) bool {
				return !kinds.IsFluxSuspended(o) && slices.Contains([]kinds.FluxState{kinds.FluxReady, kinds.FluxReconciling, kinds.FluxNotReady}, kinds.FluxStateOf(ref.Kind, o))
			})
			obj, _ := s.Object(ctx, ref)
			t.Logf("%s: reconciled, suspended, resumed; now %s", ref, kinds.FluxStateOf(ref.Kind, obj))
		})
	}
}

// messages are the confirmations asked so far, as the dialog words them.
func (c *recordingConfirmer) messages() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]string, len(c.reqs))
	for i, r := range c.reqs {
		out[i] = r.Message()
	}
	return out
}

// all returns the toasts and drain events so far.
func (e *drainEvents) all() []engine.Event {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.events)
}
