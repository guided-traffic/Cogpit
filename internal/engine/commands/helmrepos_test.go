package commands_test

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// webChartV is the test repository's chart web at a version: a ConfigMap from .Values.greeting.
func webChartV(version string) testkit.TestChart {
	return testkit.TestChart{
		Name: "web", Version: version, AppVersion: "1." + version, Description: "A web server",
		Values: "# The greeting the ConfigMap holds.\ngreeting: hello\npassword: \"\"\n",
		Templates: map[string]string{"templates/cm.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}\n" +
			"data:\n  greeting: {{ .Values.greeting | quote }}\n"},
	}
}

// chartHarness is a session with the Helm repositories on temporary folders, a local repository
// served over TLS, the Helm releases, the local kinds and a recording confirmer.
type chartHarness struct {
	s        *engine.Session
	src      *testkit.MemorySource
	repo     *testkit.HelmRepo
	repos    *engine.HelmRepos
	cfg      string
	cacheDir string
	audit    *syncBuffer
	journal  *engine.MemoryHelmJournal

	mu     sync.Mutex
	asked  []engine.ConfirmRequest
	answer bool
	events []engine.Event
}

// chartConnector connects "test" with the test profiles, Helm's releases and the local kinds; its
// backend's writer is actions.
type chartConnector struct {
	src     *testkit.MemorySource
	actions engine.Actions
}

func (c *chartConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: "test", Current: true}}, nil
}

func (c *chartConnector) Connect(_ context.Context, name string) (*engine.Backend, error) {
	ch := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	profiles := append(append(testProfiles(), helm.NewReleaseProfile()), kinds.Local()...)
	return &engine.Backend{Context: name, Namespace: "default", Cache: ch, Close: ch.Close, Profiles: profiles,
		Conn: &cluster.Conn{}, Actions: c.actions}, nil
}

func newChartHarness(t *testing.T, actions engine.Actions, opts ...engine.HelmReposOption) *chartHarness {
	t.Helper()
	return newChartHarnessWith(t, engine.NewMemoryHelmJournal(), testkit.NewMemorySource(), actions, opts...)
}

// newChartHarnessWith is newChartHarness on a journal and a source of the test's own.
func newChartHarnessWith(t *testing.T, journal *engine.MemoryHelmJournal, src *testkit.MemorySource, actions engine.Actions, opts ...engine.HelmReposOption) *chartHarness {
	t.Helper()
	h := &chartHarness{src: src, cfg: t.TempDir(), cacheDir: t.TempDir(), audit: &syncBuffer{},
		journal: journal, answer: true}
	h.repo = testkit.NewHelmRepo(t, webChartV("0.1.0"), webChartV("0.2.0"))
	if actions == nil {
		actions = &fakeWriter{}
	}
	opts = append([]engine.HelmReposOption{engine.WithChartFetcher(&helm.Fetcher{Client: h.repo.Client()})}, opts...)
	h.repos = engine.NewHelmRepos(h.cfg, h.cacheDir, opts...)
	h.s = engine.New(engine.Options{
		Connector: &chartConnector{src: h.src, actions: actions}, SnapshotInterval: 10 * time.Millisecond,
		Audit: h.audit, Helm: engine.NewHelmOps(h.journal), HelmRepos: h.repos,
		Confirmer: engine.ConfirmFunc(func(_ context.Context, req engine.ConfirmRequest) (bool, error) {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.asked = append(h.asked, req)
			return h.answer, nil
		}),
	})
	if err := commands.RegisterAll(h.s.Registry()); err != nil {
		t.Fatal(err)
	}
	h.s.Subscribe(func(ev engine.Event) {
		h.mu.Lock()
		h.events = append(h.events, ev)
		h.mu.Unlock()
	})
	t.Cleanup(h.s.Close)
	if err := h.s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *chartHarness) dispatch(origin engine.Origin, id string, params any) (any, error) {
	raw, _ := json.Marshal(params)
	return h.s.Dispatch(context.Background(), engine.Invocation{ID: id, Params: raw, Origin: origin})
}

func (h *chartHarness) confirmations() []engine.ConfirmRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return slices.Clone(h.asked)
}

// charts waits until the helmcharts kind holds want objects and returns their names.
func (h *chartHarness) charts(t *testing.T, want int) []string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var names []string
	if !testkit.WaitFor(ctx, func() bool {
		objs, err := h.s.Objects(ctx, kinds.HelmChartsKey, nil)
		if err != nil {
			return false
		}
		names = names[:0]
		for _, o := range objs {
			names = append(names, o.GetName())
		}
		return len(names) == want
	}) {
		t.Fatalf("charts %v, want %d", names, want)
	}
	slices.Sort(names)
	return names
}

func (h *chartHarness) addRepo(t *testing.T) engine.HelmRepoInfo {
	t.Helper()
	res, err := h.dispatch(engine.OriginGUI, "helm.repo.add", map[string]any{"name": "test", "url": h.repo.URL})
	if err != nil {
		t.Fatal(err)
	}
	return res.(engine.HelmRepoInfo)
}

// Adding a repository downloads its index and keeps the repository in Cogpit's own configuration
// folder, its index in Cogpit's own cache folder, both owner-only; its charts become the
// helmcharts kind. A second Cogpit reads both without a download. Removing it deletes its index
// and its charts. Nothing is written anywhere else.
func TestHelmReposAddListRemove(t *testing.T) {
	h := newChartHarness(t, nil)
	info := h.addRepo(t)
	if info.Name != "test" || info.URL != h.repo.URL || info.Charts != 1 || info.Updated.IsZero() || info.Error != "" {
		t.Errorf("added %+v", info)
	}
	if names := h.charts(t, 1); names[0] != "test.web" {
		t.Errorf("charts %v", names)
	}
	web, err := h.s.Lookup(engine.HelmChartRef("test", "test", "web"))
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := web.Object["spec"].(map[string]any)
	if spec["version"] != "0.2.0" || spec["repository"] != "test" || len(spec["versions"].([]any)) != 2 || spec["digest"] == "" {
		t.Errorf("chart object %v", spec)
	}
	listFile := filepath.Join(h.cfg, "helm", engine.HelmReposFile)
	indexFile := filepath.Join(h.cacheDir, "helm", "test-index.yaml")
	for _, p := range []string{listFile, indexFile} {
		if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("%s: %v %v", p, fi, err)
		}
	}
	if data, _ := os.ReadFile(listFile); !strings.Contains(string(data), h.repo.URL) {
		t.Errorf("repositories.json: %s", data)
	}

	if _, err := h.dispatch(engine.OriginGUI, "helm.repo.add", map[string]any{"name": "test", "url": h.repo.URL}); err == nil {
		t.Error("a second repository named test was added")
	}
	if _, err := h.dispatch(engine.OriginGUI, "helm.repo.add", map[string]any{"name": "plain", "url": "http://charts.example.com"}); !errors.Is(err, helm.ErrNotHTTPS) {
		t.Errorf("a plain http repository: %v", err)
	}
	h.repo.Fail(true)
	if _, err := h.dispatch(engine.OriginGUI, "helm.repo.add", map[string]any{"name": "down", "url": h.repo.URL + "/down"}); err == nil {
		t.Error("a repository whose index does not download was added")
	}
	h.repo.Fail(false)

	// A second Cogpit on the same folders reads the list and the index from disk.
	before := h.repo.Requests("/index.yaml")
	again := engine.NewHelmRepos(h.cfg, h.cacheDir, engine.WithChartFetcher(&helm.Fetcher{Client: h.repo.Client()}))
	list, err := again.List()
	if err != nil || len(list) != 1 || list[0].Charts != 1 || h.repo.Requests("/index.yaml") != before {
		t.Errorf("reloaded %+v %v (%d downloads)", list, err, h.repo.Requests("/index.yaml")-before)
	}

	// An update downloads the index again and lists a new version.
	h.repo.Add(webChartV("0.3.0"))
	res, err := h.dispatch(engine.OriginExternalAgent, "helm.repo.update", struct{}{})
	if err != nil || h.repo.Requests("/index.yaml") != before+1 {
		t.Fatalf("update: %v, %v (%d downloads)", res, err, h.repo.Requests("/index.yaml")-before)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool {
		o, err := h.s.Lookup(engine.HelmChartRef("test", "test", "web"))
		return err == nil && o.Object["spec"].(map[string]any)["version"] == "0.3.0"
	}) {
		t.Error("the update's new version is not listed")
	}
	if len(h.confirmations()) != 0 {
		t.Errorf("an update asked: %+v", h.confirmations())
	}

	if _, err := h.dispatch(engine.OriginGUI, "helm.repo.remove", map[string]any{"name": "test"}); err != nil {
		t.Fatal(err)
	}
	h.charts(t, 0)
	if _, err := os.Stat(indexFile); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the index stayed: %v", err)
	}
	var files []string
	for _, root := range []string{h.cfg, h.cacheDir} {
		_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, _ error) error {
			if !d.IsDir() {
				files = append(files, strings.TrimPrefix(p, root))
			}
			return nil
		})
	}
	if len(files) != 1 || files[0] != "/helm/"+engine.HelmReposFile {
		t.Errorf("files left: %v", files)
	}
}

// An agent's add and remove are refused before anyone is asked, and download nothing; the list and
// an update run for agents, and nobody is asked.
func TestHelmReposAreTheUsers(t *testing.T) {
	h := newChartHarness(t, nil)
	if _, err := h.dispatch(engine.OriginExternalAgent, "helm.repo.add", map[string]any{"name": "test", "url": h.repo.URL}); err == nil ||
		!strings.Contains(err.Error(), "the user's to add") {
		t.Fatalf("an agent's add: %v", err)
	}
	if list, _ := h.repos.List(); len(list) != 0 || h.repo.Requests("/index.yaml") != 0 {
		t.Errorf("refused, yet %+v (%d downloads)", list, h.repo.Requests("/index.yaml"))
	}
	h.addRepo(t)
	if _, err := h.dispatch(engine.OriginExternalAgent, "helm.repo.remove", map[string]any{"name": "test"}); err == nil {
		t.Error("an agent removed a repository")
	}
	if _, err := h.dispatch(engine.OriginCLI, "helm.repo.update", map[string]any{"names": []string{"test"}}); err != nil {
		t.Fatal(err)
	}
	var list []engine.HelmRepoInfo
	res, err := h.dispatch(engine.OriginExternalAgent, "helm.repo.list", struct{}{})
	if err != nil {
		t.Fatal(err)
	}
	list = res.([]engine.HelmRepoInfo)
	if len(list) != 1 || list[0].Charts != 1 || h.repo.Requests("/index.yaml") != 2 {
		t.Errorf("list %+v (%d downloads)", list, h.repo.Requests("/index.yaml"))
	}
	if n := len(h.confirmations()); n != 0 {
		t.Errorf("%d confirmations", n)
	}
	if _, err := h.dispatch(engine.OriginCLI, "helm.repo.remove", map[string]any{"name": "test"}); err != nil {
		t.Errorf("the CLI's remove: %v", err)
	}
}

// An index is downloaded again only while the Charts view or a Helm Install tab is open and its
// time-to-live has passed (architecture §5): never while neither is shown.
func TestHelmIndexRefreshedOnlyWhileShown(t *testing.T) {
	h := newChartHarness(t, nil, engine.WithIndexTTL(40*time.Millisecond, 10*time.Millisecond))
	h.addRepo(t)
	if _, err := h.dispatch(engine.OriginGUI, "ui.openView", map[string]any{"view": "pods"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if n := h.repo.Requests("/index.yaml"); n != 1 {
		t.Fatalf("%d downloads with the Charts view closed, want the add's 1", n)
	}
	if _, err := h.dispatch(engine.OriginGUI, "ui.openView", map[string]any{"view": kinds.HelmChartsKey}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool { return h.repo.Requests("/index.yaml") >= 3 }) {
		t.Fatalf("%d downloads with the Charts view open", h.repo.Requests("/index.yaml"))
	}
	if _, err := h.dispatch(engine.OriginGUI, "ui.openView", map[string]any{"view": "pods"}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond) // a download that had started ends
	n := h.repo.Requests("/index.yaml")
	time.Sleep(200 * time.Millisecond)
	if m := h.repo.Requests("/index.yaml"); m != n {
		t.Errorf("%d downloads after the Charts view closed", m-n)
	}
}

// TestChartDrawerStartsNoEventsWatch: a chart is an object Cogpit makes itself, so no event
// involves it; its drawer has no Events section and starts no cluster-wide events watch.
func TestChartDrawerStartsNoEventsWatch(t *testing.T) {
	h := newChartHarness(t, nil)
	h.addRepo(t)
	h.charts(t, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	m, err := h.s.DetailWait(ctx, engine.HelmChartRef("test", "test", "web"))
	if err != nil {
		t.Fatal(err)
	}
	if !m.NoEvents || len(m.Events) != 0 {
		t.Errorf("chart drawer: no events %v, events %v", m.NoEvents, m.Events)
	}
	if evp, ok := h.s.Backend().Profile("events"); ok && h.s.Backend().Cached(evp) {
		t.Error("the chart's drawer started the events watch")
	}
}
