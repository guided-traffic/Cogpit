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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// logSourcesNamespace is the shared kind cluster's namespace of the log selectors, older lines
// and jump test; it creates it, acts only inside it, and deletes it.
const logSourcesNamespace = "cogpit-l-logs"

// stampOf returns a merged stream line's source and timestamp ("<pod>/<container> <ts> …").
func stampOf(t *testing.T, line string) (string, time.Time) {
	t.Helper()
	parts := strings.SplitN(line, " ", 3)
	if len(parts) < 3 {
		return "", time.Time{}
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[1])
	if err != nil {
		return "", time.Time{}
	}
	return parts[0], ts
}

// TestKindLogSourcesOlderAndJump runs a log tab's selectors, older lines and a jump through the
// engine against the shared kind cluster: a Deployment of two pods, each with two talking
// containers and an init container, in its own namespace. It checks against the real API server
// that narrowing ends the left-out streams and selecting again resumes without repeating a line,
// that an init container's log shows when named, and how tailLines and sinceTime answer the
// requests for older lines.
func TestKindLogSourcesOlderAndJump(t *testing.T) {
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
	kube, ns := conn.Kube(), logSourcesNamespace
	freshNamespace(ctx, t, kube, ns)

	tmpl := talkingTemplate("talk", "app", "side")
	setup := sleeper("setup")
	setup.Command = []string{"sh", "-c", `echo "setup done on $(hostname)"`}
	tmpl.Spec.InitContainers = []corev1.Container{setup}
	if _, err := kube.AppsV1().Deployments(ns).Create(ctx, &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Namespace: ns, Name: "talk"},
		Spec: appsv1.DeploymentSpec{Replicas: int32p(2), Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "talk"}},
			Template: tmpl},
	}, metav1.CreateOptions{}); err != nil {
		t.Fatal(err)
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
	waitFor(ctx, t, "two ready talk pods", func() bool { return len(readyPods(ctx, kube, ns, "talk")) == 2 })
	dep := model.ObjectRef{Kind: "deployments", Namespace: ns, Name: "talk"}
	d.cached(dep, nil)
	var tab commands.DockResult
	d.must("workload.logs", map[string]any{"ref": dep}, &tab)
	id := tab.Tab.ID

	// The selectors offer both pods, the two containers and the init container apart.
	var src engine.LogSources
	waitFor(ctx, t, "the pods in the cache", func() bool {
		d.must("logs.sources", map[string]any{"id": id}, &src)
		return len(src.Pods) == 2
	})
	if !src.AllPods || !slices.Equal(src.Containers, []string{"app", "side"}) || !slices.Equal(src.InitContainers, []string{"setup"}) {
		t.Fatalf("sources %+v", src)
	}
	a, b := src.Pods[0], src.Pods[1]

	stream, err := s.OpenLogs(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	merged := collect(stream)
	count := func(source string) int {
		n := 0
		for _, l := range merged.snapshot() {
			if strings.HasPrefix(l, source+" ") {
				n++
			}
		}
		return n
	}
	merged.waitLines(ctx, t, "lines of every container", func([]string) bool {
		return count(a+"/app") > 0 && count(a+"/side") > 0 && count(b+"/app") > 0 && count(b+"/side") > 0
	})

	// Narrowed to pod a: b's lines stop while a's go on.
	d.must("logs.setSource", map[string]any{"id": id, "pod": a}, nil)
	base := count(a + "/app")
	merged.waitLines(ctx, t, "two more lines of a", func([]string) bool { return count(a+"/app") >= base+2 })
	bApp, bSide := count(b+"/app"), count(b+"/side") // the lines in flight have arrived
	base = count(a + "/app")
	merged.waitLines(ctx, t, "four more lines of a", func([]string) bool { return count(a+"/app") >= base+4 })
	if count(b+"/app") != bApp || count(b+"/side") != bSide {
		t.Errorf("pod b's lines went on after narrowing to a: %d→%d, %d→%d", bApp, count(b+"/app"), bSide, count(b+"/side"))
	}

	// Every pod again: b resumes with the lines written meanwhile, none twice.
	d.must("logs.setSource", map[string]any{"id": id, "pod": ""}, nil)
	merged.waitLines(ctx, t, "pod b's lines again", func([]string) bool { return count(b+"/app") >= bApp+5 })
	seen := map[string]bool{}
	for _, l := range merged.snapshot() {
		if seen[l] && !strings.HasPrefix(l, engine.NoticePrefix) {
			t.Errorf("a line twice: %q", l)
		}
		seen[l] = true
	}

	// The init container, named: its log shows.
	d.must("logs.setSource", map[string]any{"id": id, "container": "setup"}, nil)
	merged.waitLines(ctx, t, "the init container's line", func(got []string) bool {
		return slices.ContainsFunc(got, func(l string) bool {
			return strings.HasPrefix(l, a+"/setup ") && strings.Contains(l, "setup done on "+a)
		})
	})
	d.must("logs.setSource", map[string]any{"id": id, "container": ""}, nil)

	// Older lines: what the tab holds of a/app, pretending it holds only its last three.
	merged.waitLines(ctx, t, "eight lines of a/app", func([]string) bool { return count(a+"/app") >= 8 })
	var aLines []time.Time
	for _, l := range merged.snapshot() {
		if src, ts := stampOf(t, l); src == a+"/app" {
			aLines = append(aLines, ts)
		}
	}
	if len(aLines) < 8 {
		t.Fatalf("%d lines of %s/app", len(aLines), a)
	}
	slices.SortFunc(aLines, func(x, y time.Time) int { return x.Compare(y) })
	oldest := aLines[len(aLines)-3]
	tabNow, err := s.LogsTab(id)
	if err != nil {
		t.Fatal(err)
	}
	held := []engine.HeldLog{{Source: a + "/app", Lines: 3, Oldest: oldest}}
	for _, other := range []string{a + "/side", b + "/app", b + "/side"} {
		held = append(held, engine.HeldLog{Source: other, Lines: count(other), Whole: true})
	}
	older, err := s.OlderLogs(ctx, tabNow.ID, engine.OlderRequest{Held: held, Room: 5000})
	if err != nil {
		t.Fatal(err)
	}
	if len(older.Lines) < len(aLines)-3 || len(older.Notes) != 0 || !older.Complete || !slices.Contains(older.Whole, a+"/app") {
		t.Errorf("older lines: %d (want at least %d), notes %q, complete %v, whole %v", len(older.Lines), len(aLines)-3, older.Notes, older.Complete, older.Whole)
	}
	var prev time.Time
	for _, l := range older.Lines {
		src, ts := stampOf(t, l)
		if src != a+"/app" || ts.After(oldest) || ts.Before(prev) {
			t.Errorf("an older line out of place: %q (oldest held %v)", l, oldest)
		}
		prev = ts
	}

	// A jump between: sinceTime from the API server, the lines from then to the oldest held.
	since := aLines[2].Add(time.Millisecond)
	jumped, err := s.OlderLogs(ctx, tabNow.ID, engine.OlderRequest{Held: held, Since: since, Room: 5000})
	if err != nil || len(jumped.Lines) == 0 || !jumped.Complete || len(jumped.Whole) != 0 {
		t.Fatalf("since %v: %+v, %v", since, jumped, err)
	}
	if _, first := stampOf(t, jumped.Lines[0]); first.Before(since.Truncate(time.Second)) {
		t.Errorf("a line before the second jumped to: %q", jumped.Lines[0])
	}
	d.must("logs.jumpTo", map[string]any{"id": id, "time": since.UTC().Format(time.RFC3339Nano)}, &tab.Tab)
	if !tab.Tab.Logs.Jump.Time.Equal(since) || tab.Tab.Logs.Jump.Seq != 1 {
		t.Errorf("the tab's jump %+v", tab.Tab.Logs.Jump)
	}
}
