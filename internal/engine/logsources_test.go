package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

func strp(s string) *string { return &s }

// openPodLogsTab adds a pod's logs tab on container c, as pod.logs does.
func openPodLogsTab(s *Session, pod, c string) DockTab {
	ref := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "shop", Name: pod}
	tab := DockTab{ID: s.NewDockID(DockLogs), Type: DockLogs, Title: "Logs: " + pod + "/" + c, Context: "test", Ref: &ref, Container: c}
	s.UpdateUI(func(u *UIState) { u.Dock = append(u.Dock, tab) })
	return tab
}

// streamsOf returns the containers a merged stream streams now, as "<pod>/<container>".
func streamsOf(wl *workloadLogs) []string {
	wl.mu.Lock()
	defer wl.mu.Unlock()
	var out []string
	for _, e := range wl.streaming {
		out = append(out, e.target.pod+"/"+e.target.container)
	}
	slices.Sort(out)
	return out
}

// TestWorkloadLogsSelectionNarrowsAndRestores: narrowing a Deployment's tab to one pod ends the
// other pod's streams and leaves the selected pod's alone (no new request); its lines written
// meanwhile do not show; selecting every pod again resumes the left-out stream from its newest
// line (sinceTime, bounded by the tail), so the lines written meanwhile arrive once and none
// twice. Narrowing to a container works across pods the same way.
func TestWorkloadLogsSelectionNarrowsAndRestores(t *testing.T) {
	s, _ := logsSession(t)
	f := newServerLogs()
	t0 := time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)
	f.add("web-1-a/app", t0.Add(100*time.Millisecond), "a1")
	f.add("web-1-a/proxy", t0.Add(200*time.Millisecond), "p1")
	f.add("web-1-b/app", t0.Add(300*time.Millisecond), "b1")
	tab, _, err := s.OpenWorkloadLogsTab(context.Background(), model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	rc := openTabLogs(t, s, tab.ID, f)
	wl := rc.(*workloadLogs)
	lr := readLines(rc)
	lr.wait(t, stamped("web-1-a/app", t0.Add(100*time.Millisecond), "a1"), stamped("web-1-b/app", t0.Add(300*time.Millisecond), "b1"))
	waitUntil(t, "three streams following", func() bool {
		return f.opened("web-1-a/app") && f.opened("web-1-a/proxy") && f.opened("web-1-b/app")
	})

	if _, err := s.SetLogSource(context.Background(), tab.ID, strp("web-1-a"), nil); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "web-1-b's stream to end", func() bool { return slices.Equal(streamsOf(wl), []string{"web-1-a/app", "web-1-a/proxy"}) })
	f.add("web-1-b/app", t0.Add(2*time.Second), "b while away")
	f.add("web-1-a/app", t0.Add(3*time.Second), "a2")
	lr.wait(t, stamped("web-1-a/app", t0.Add(3*time.Second), "a2"))
	if slices.ContainsFunc(lr.all(), func(l string) bool { return strings.Contains(l, "b while away") }) {
		t.Error("a line of the pod the tab left out")
	}
	for _, key := range []string{"web-1-a/app", "web-1-a/proxy"} {
		if n := len(f.requestsFor(key)); n != 1 {
			t.Errorf("%s asked %d times: a stream that stays selected was opened again", key, n)
		}
	}

	// Every pod again: web-1-b resumes from its newest line, bounded.
	if _, err := s.SetLogSource(context.Background(), tab.ID, strp(""), nil); err != nil {
		t.Fatal(err)
	}
	lr.wait(t, stamped("web-1-b/app", t0.Add(2*time.Second), "b while away"))
	reqs := f.requestsFor("web-1-b/app")
	if len(reqs) != 2 {
		t.Fatalf("web-1-b/app requests %+v", reqs)
	}
	if r := reqs[1]; !r.SinceTime.Equal(t0) || r.TailLines != workloadLogTail || !r.Follow || !r.Timestamps {
		t.Errorf("the resumption %+v: want sinceTime %v and the tail as its bound", r, t0)
	}
	waitUntil(t, "web-1-b to follow again", func() bool { return len(streamsOf(wl)) == 3 && f.opened("web-1-b/app") })
	f.add("web-1-b/app", t0.Add(4*time.Second), "b2")
	lr.wait(t, stamped("web-1-b/app", t0.Add(4*time.Second), "b2"))
	count := map[string]int{}
	for _, l := range lr.all() {
		count[l]++
	}
	for l, n := range count {
		if n > 1 {
			t.Errorf("%q came %d times", l, n)
		}
	}

	// One container's name across pods: the proxies only; web-1-a/proxy's stream stays.
	if _, err := s.SetLogSource(context.Background(), tab.ID, nil, strp("proxy")); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "the proxies only", func() bool { return slices.Equal(streamsOf(wl), []string{"web-1-a/proxy"}) })
	if n := len(f.requestsFor("web-1-a/proxy")); n != 1 {
		t.Errorf("web-1-a/proxy asked %d times", n)
	}
	if u, _ := s.UIState().DockTab(tab.ID); u.Logs.Pod != "" || u.Container != "proxy" {
		t.Errorf("the tab's source %q %q", u.Logs.Pod, u.Container)
	}
}

// TestWorkloadLogsSelectedAgainAfterManyLines: a container selected again after more lines than
// the tail gets the newest of them and a notice that the older ones are not shown.
func TestWorkloadLogsSelectedAgainAfterManyLines(t *testing.T) {
	s, _ := logsSession(t)
	f := newServerLogs()
	t0 := time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC)
	f.add("web-1-b/app", t0, "b0")
	tab, _, err := s.OpenWorkloadLogsTab(context.Background(), model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	rc := openTabLogs(t, s, tab.ID, f)
	wl := rc.(*workloadLogs)
	lr := readLines(rc)
	lr.wait(t, stamped("web-1-b/app", t0, "b0"))
	if _, err := s.SetLogSource(context.Background(), tab.ID, strp("web-1-a"), nil); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, "web-1-b's stream to end", func() bool { return !slices.Contains(streamsOf(wl), "web-1-b/app") })
	for i := range workloadLogTail + 5 {
		f.add("web-1-b/app", t0.Add(time.Duration(i+1)*time.Second), fmt.Sprintf("away %d", i))
	}
	if _, err := s.SetLogSource(context.Background(), tab.ID, strp(""), nil); err != nil {
		t.Fatal(err)
	}
	last := stamped("web-1-b/app", t0.Add(time.Duration(workloadLogTail+5)*time.Second), fmt.Sprintf("away %d", workloadLogTail+4))
	lr.wait(t, last, NoticePrefix+" web-1-b/app: more lines were written while it was not selected than Cogpit loads again (1000); the older ones are not shown")
	got := lr.all()
	if slices.ContainsFunc(got, func(l string) bool { return strings.HasSuffix(l, " away 4") }) || !slices.ContainsFunc(got, func(l string) bool { return strings.HasSuffix(l, " away 5") }) {
		t.Errorf("want the newest %d lines written meanwhile, from \"away 5\"", workloadLogTail)
	}
}

// initPod is a pod of the web replica set with a finished init container migrate, a running
// sidecar init container mesh and a running container app.
func initPod(name string) *unstructured.Unstructured {
	pod := testkit.Object("v1", "Pod", "shop", name, map[string]any{
		"spec": map[string]any{
			"initContainers": []any{
				map[string]any{"name": "migrate", "image": "registry.example/migrate:1"},
				map[string]any{"name": "mesh", "image": "registry.example/mesh:1", "restartPolicy": "Always"},
			},
			"containers": []any{map[string]any{"name": "app", "image": "registry.example/app:1"}},
		},
		"status": map[string]any{
			"phase": "Running",
			"initContainerStatuses": []any{
				map[string]any{"name": "migrate", "restartCount": int64(0), "state": map[string]any{"terminated": map[string]any{"exitCode": int64(0)}}},
				map[string]any{"name": "mesh", "restartCount": int64(0), "state": map[string]any{"running": map[string]any{}}},
			},
			"containerStatuses": []any{map[string]any{"name": "app", "restartCount": int64(0), "state": map[string]any{"running": map[string]any{}}}},
		},
	})
	return owned(pod, "apps/v1", "ReplicaSet", "web-1", "uid-shop-web-1")
}

// TestWorkloadLogsInitContainers: every container of a tab means the containers and the sidecar
// init containers, not an init container that ran to completion; naming that one streams its log.
// The selectors list the init containers apart.
func TestWorkloadLogsInitContainers(t *testing.T) {
	s, src := logsSession(t)
	src.Delete(testkit.PodsGVR, "shop", "web-1-a")
	src.Delete(testkit.PodsGVR, "shop", "web-1-b")
	src.Set(testkit.PodsGVR, initPod("web-1-i"))
	waitUntil(t, "the pod in the cache", func() bool {
		_, err := s.Lookup(model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-1-i"})
		_, gone := s.Lookup(model.ObjectRef{Kind: "pods", Namespace: "shop", Name: "web-1-a"})
		return err == nil && gone != nil
	})
	f := newFakeLogs()
	f.content["web-1-i/migrate"] = "migrated\n"
	tab, _, err := s.OpenWorkloadLogsTab(context.Background(), model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	src2, err := s.LogSources(context.Background(), tab.ID)
	if err != nil || !src2.AllPods || !slices.Equal(src2.Pods, []string{"web-1-i"}) || !slices.Equal(src2.Containers, []string{"app"}) ||
		!slices.Equal(src2.InitContainers, []string{"migrate", "mesh"}) {
		t.Errorf("sources %+v: %v", src2, err)
	}
	rc := openTabLogs(t, s, tab.ID, f)
	wl := rc.(*workloadLogs)
	lr := readLines(rc)
	waitUntil(t, "app and the sidecar", func() bool { return slices.Equal(streamsOf(wl), []string{"web-1-i/app", "web-1-i/mesh"}) })
	if _, err := s.SetLogSource(context.Background(), tab.ID, nil, strp("migrate")); err != nil {
		t.Fatal(err)
	}
	lr.wait(t, "web-1-i/migrate migrated")
	waitUntil(t, "the others to end", func() bool { return slices.Equal(streamsOf(wl), []string{"web-1-i/migrate"}) })
	if r := f.requestFor(t, "web-1-i", "migrate"); !r.Follow || !r.Timestamps {
		t.Errorf("the init container's request %+v", r)
	}
}

// TestLogSourcesAndSetLogSource: what the selectors offer, and what a source change accepts, for
// a controlling resource's tab and for a pod's tab.
func TestLogSourcesAndSetLogSource(t *testing.T) {
	s, _ := logsSession(t)
	ctx := context.Background()
	tab, _, err := s.OpenWorkloadLogsTab(ctx, model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.LogSources(ctx, tab.ID)
	if err != nil || !got.AllPods || !slices.Equal(got.Pods, []string{"web-1-a", "web-1-b"}) || !slices.Equal(got.Containers, []string{"app", "proxy"}) || len(got.InitContainers) != 0 {
		t.Errorf("deployment's sources %+v: %v", got, err)
	}
	for _, tc := range []struct {
		pod, container *string
		want           string
	}{
		{strp("db-0"), nil, `has no pod "db-0"; its pods: web-1-a, web-1-b`},
		{nil, strp("sidecar"), `no container "sidecar" in deployments/shop/web; containers: app, proxy`},
		{strp("web-1-a"), strp("sidecar"), `no container "sidecar" in pod web-1-a`},
	} {
		if _, err := s.SetLogSource(ctx, tab.ID, tc.pod, tc.container); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v %v: %v, want %q", tc.pod, tc.container, err, tc.want)
		}
	}
	if out, err := s.SetLogSource(ctx, tab.ID, strp("web-1-b"), strp("proxy")); err != nil || out.Logs.Pod != "web-1-b" || out.Container != "proxy" {
		t.Errorf("narrowed %+v: %v", out, err)
	}
	if got, _ := s.LogSources(ctx, tab.ID); !slices.Equal(got.Containers, []string{"app", "proxy"}) {
		t.Errorf("the narrowed pod's containers %v", got.Containers)
	}

	// A pod's tab: the pods of its deployment; another pod keeps the container.
	pt := openPodLogsTab(s, "web-1-a", "proxy")
	got, err = s.LogSources(ctx, pt.ID)
	if err != nil || got.AllPods || !slices.Equal(got.Pods, []string{"web-1-a", "web-1-b"}) || !slices.Equal(got.Containers, []string{"app", "proxy"}) {
		t.Errorf("pod's sources %+v: %v", got, err)
	}
	out, err := s.SetLogSource(ctx, pt.ID, strp("web-1-b"), nil)
	if err != nil || out.Ref.Name != "web-1-b" || out.Container != "proxy" || out.Title != "Logs: web-1-b/proxy" {
		t.Errorf("switched pod %+v: %v", out, err)
	}
	out, err = s.SetLogSource(ctx, pt.ID, nil, strp("app"))
	if err != nil || out.Ref.Name != "web-1-b" || out.Container != "app" || out.Title != "Logs: web-1-b/app" {
		t.Errorf("switched container %+v: %v", out, err)
	}
	for _, tc := range []struct {
		pod, container *string
		want           error
		text           string
	}{
		{strp(""), nil, ErrNotApplicable, "one pod"},
		{nil, strp(""), ErrNotApplicable, "one container"},
		{strp("db-0"), nil, nil, `"db-0" is not a pod of the resource that controls web-1-b`},
		{nil, strp("db"), nil, `has no container "db"`},
	} {
		_, err := s.SetLogSource(ctx, pt.ID, tc.pod, tc.container)
		if err == nil || (tc.want != nil && !errors.Is(err, tc.want)) || !strings.Contains(err.Error(), tc.text) {
			t.Errorf("pod tab %v %v: %v", tc.pod, tc.container, err)
		}
	}

	// A pod nobody controls offers itself alone.
	lt := openPodLogsTab(s, "loner", "app")
	if got, err := s.LogSources(ctx, lt.ID); err != nil || !slices.Equal(got.Pods, []string{"loner"}) {
		t.Errorf("loner's sources %+v: %v", got, err)
	}
	if _, err := s.SetLogSource(ctx, "logs-99", nil, strp("app")); err == nil {
		t.Error("a tab that does not exist")
	}
}
