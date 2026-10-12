package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Helm chart repositories (G11, architecture §8): the HTTPS repositories the user adds by name and
// URL (helm.repo.add), kept in Cogpit's own configuration folder (<config>/helm/repositories.json,
// owner-only, replaced atomically), never in Helm's repositories.yaml; their indexes downloaded
// into Cogpit's own cache folder (<cache>/helm/<name>-index.yaml) on add, on an explicit update
// (helm.repo.update) and, as architecture §5 has it for data that cannot be watched, while the
// Charts view or a Helm Install tab is open and an index is older than its time-to-live. Their
// charts are the local kind helmcharts in the connected backend's local store (one object per
// chart), which the Charts view, the drawer and agents read like any kind.

// Defaults of the repositories (Cogpit's, not the owner's decision: an interval of the order of
// ten minutes keeps a repository's new versions visible without polling a server often).
const (
	// DefaultIndexTTL is how old an index may get while the Charts view or a Helm Install tab is
	// open before it is downloaded again.
	DefaultIndexTTL = 10 * time.Minute
	// indexPollEvery is how often the open view checks the indexes' age.
	indexPollEvery = time.Minute
	// HelmReposFile is the repositories' file under <config>/helm.
	HelmReposFile = "repositories.json"
	helmDir       = "helm"
	// maxHelmRepos bounds the repositories a user adds.
	maxHelmRepos     = 64
	helmReposVersion = 1
	maxHelmReposFile = 64 << 10
)

// ErrNoHelmRepos disables the repository commands of a session that keeps none.
var ErrNoHelmRepos = errors.New("this Cogpit keeps no Helm chart repositories")

// HelmRepo is a chart repository as the user added it.
type HelmRepo struct {
	Name  string    `json:"name"`
	URL   string    `json:"url"`
	Added time.Time `json:"added"`
}

// HelmRepoInfo is a repository as helm.repo.list reports it: when its index was downloaded, how
// many charts it lists, and the error of its last download or read.
type HelmRepoInfo struct {
	HelmRepo
	Updated  time.Time `json:"updated,omitzero"`
	Charts   int       `json:"charts"`
	Error    string    `json:"error,omitempty"`
	Updating bool      `json:"updating,omitempty"`
}

type helmReposDoc struct {
	Version      int        `json:"version"`
	Repositories []HelmRepo `json:"repositories"`
}

// repoState is what Cogpit holds of one repository's index.
type repoState struct {
	index    *helm.Index
	updated  time.Time
	err      string
	gen      uint64 // the generation of the index: the chart objects' resourceVersion
	updating bool
}

// HelmRepos keeps a session's chart repositories. Create it with NewHelmRepos and pass it as
// Options.HelmRepos; it belongs to one session.
type HelmRepos struct {
	configDir, cacheDir string
	fetcher             *helm.Fetcher
	ttl, pollEvery      time.Duration
	now                 func() time.Time

	// fetchMu serialises loading and the changes of the list with the downloads: one download
	// at a time, and never an add or a remove while one runs.
	fetchMu sync.Mutex

	mu      sync.Mutex
	loaded  bool
	repos   []HelmRepo
	state   map[string]*repoState
	gen     uint64
	s       *Session
	lastRev uint64
	poll    chan struct{} // closed to stop the poller; nil while none runs

	pubMu     sync.Mutex
	pubTarget *Backend
	published map[string]string // chart object name → resourceVersion in pubTarget's local store
}

// HelmReposOption configures NewHelmRepos.
type HelmReposOption func(*HelmRepos)

// WithChartFetcher downloads with f (tests: a client that trusts a local repository).
func WithChartFetcher(f *helm.Fetcher) HelmReposOption {
	return func(h *HelmRepos) { h.fetcher = f }
}

// WithIndexTTL sets the indexes' time-to-live and how often the open view checks it.
func WithIndexTTL(ttl, every time.Duration) HelmReposOption {
	return func(h *HelmRepos) { h.ttl, h.pollEvery = ttl, every }
}

// NewHelmRepos returns the repositories kept under configDir/helm (the list) and cacheDir/helm
// (their indexes): Cogpit's own folders, never Helm's. Nothing is read or written until used.
func NewHelmRepos(configDir, cacheDir string, opts ...HelmReposOption) *HelmRepos {
	h := &HelmRepos{
		configDir: filepath.Join(configDir, helmDir), cacheDir: filepath.Join(cacheDir, helmDir),
		fetcher: &helm.Fetcher{}, ttl: DefaultIndexTTL, pollEvery: indexPollEvery, now: time.Now,
		state: map[string]*repoState{},
	}
	for _, o := range opts {
		o(h)
	}
	return h
}

// HelmRepos returns the session's chart repositories, or nil when it keeps none.
func (s *Session) HelmRepos() *HelmRepos { return s.opts.HelmRepos }

// Path returns the repositories' file.
func (h *HelmRepos) Path() string { return filepath.Join(h.configDir, HelmReposFile) }

func (h *HelmRepos) indexPath(name string) string {
	return filepath.Join(h.cacheDir, name+"-index.yaml")
}

// ensureLoaded reads the list and the downloaded indexes once. The caller holds fetchMu.
func (h *HelmRepos) ensureLoaded() error {
	h.mu.Lock()
	loaded := h.loaded
	h.mu.Unlock()
	if loaded {
		return nil
	}
	var doc helmReposDoc
	if err := readPrivateJSON(h.Path(), maxHelmReposFile, &doc); err != nil {
		return fmt.Errorf("the Helm repositories %s are unreadable (%w); Cogpit leaves the file as it is", h.Path(), err)
	}
	if doc.Version != 0 && doc.Version != helmReposVersion {
		return fmt.Errorf("the Helm repositories %s have format version %d, this Cogpit reads %d", h.Path(), doc.Version, helmReposVersion)
	}
	var repos []HelmRepo
	state := map[string]*repoState{}
	for _, r := range doc.Repositories {
		u, err := helm.CheckRepository(r.Name, r.URL)
		if err != nil || slices.ContainsFunc(repos, func(o HelmRepo) bool { return o.Name == r.Name }) || len(repos) >= maxHelmRepos {
			continue // a hand-edited entry Cogpit would not have added
		}
		r.URL = u
		repos = append(repos, r)
		st := &repoState{}
		path := h.indexPath(r.Name)
		if fi, err := os.Stat(path); err == nil && fi.ModTime().After(r.Added.Add(-time.Second)) {
			if idx, err := helm.LoadIndex(path); err == nil {
				st.index, st.updated = idx, fi.ModTime()
			} else {
				st.err = err.Error()
			}
		} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
			st.err = err.Error()
		}
		state[r.Name] = st
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.loaded {
		h.repos, h.loaded = repos, true
		for name, st := range state {
			h.gen++
			st.gen = h.gen
			h.state[name] = st
		}
	}
	return nil
}

// save writes the list. The caller holds fetchMu.
func (h *HelmRepos) save(repos []HelmRepo) error {
	if repos == nil {
		repos = []HelmRepo{}
	}
	data, err := json.MarshalIndent(helmReposDoc{Version: helmReposVersion, Repositories: repos}, "", "  ")
	if err != nil {
		return err
	}
	if err := writePrivateFile(h.Path(), append(data, '\n')); err != nil {
		return fmt.Errorf("the Helm repositories could not be saved: %w", err)
	}
	return nil
}

// List returns the repositories in the order they were added.
func (h *HelmRepos) List() ([]HelmRepoInfo, error) {
	h.fetchMu.Lock()
	err := h.ensureLoaded()
	h.fetchMu.Unlock()
	if err != nil {
		return nil, err
	}
	return h.infos(nil), nil
}

// infos returns the repositories (only those named, if names is not nil).
func (h *HelmRepos) infos(names []string) []HelmRepoInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]HelmRepoInfo, 0, len(h.repos))
	for _, r := range h.repos {
		if names != nil && !slices.Contains(names, r.Name) {
			continue
		}
		info := HelmRepoInfo{HelmRepo: r}
		if st := h.state[r.Name]; st != nil {
			info.Updated, info.Error, info.Updating = st.updated, st.err, st.updating
			if st.index != nil {
				info.Charts = len(st.index.Charts)
			}
		}
		out = append(out, info)
	}
	return out
}

// repo returns a repository by name.
func (h *HelmRepos) repo(name string) (HelmRepo, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	i := slices.IndexFunc(h.repos, func(r HelmRepo) bool { return r.Name == name })
	if i < 0 {
		return HelmRepo{}, false
	}
	return h.repos[i], true
}

// Add adds a public https repository: its index is downloaded first, and only a repository whose
// index downloads and parses is kept. The name must be new.
func (h *HelmRepos) Add(ctx context.Context, name, rawURL string) (HelmRepoInfo, error) {
	u, err := helm.CheckRepository(name, rawURL)
	if err != nil {
		return HelmRepoInfo{}, err
	}
	h.fetchMu.Lock()
	defer h.fetchMu.Unlock()
	if err := h.ensureLoaded(); err != nil {
		return HelmRepoInfo{}, err
	}
	h.mu.Lock()
	repos := slices.Clone(h.repos)
	h.mu.Unlock()
	switch {
	case slices.ContainsFunc(repos, func(r HelmRepo) bool { return r.Name == name }):
		return HelmRepoInfo{}, fmt.Errorf("a Helm repository named %s exists already; remove it first to add it with another URL", name)
	case len(repos) >= maxHelmRepos:
		return HelmRepoInfo{}, fmt.Errorf("at most %d Helm repositories can be added", maxHelmRepos)
	}
	idx, err := h.fetcher.DownloadIndex(ctx, u, h.indexPath(name))
	if err != nil {
		return HelmRepoInfo{}, fmt.Errorf("repository %s was not added: %w", name, err)
	}
	r := HelmRepo{Name: name, URL: u, Added: h.now().UTC()}
	if err := h.save(append(repos, r)); err != nil {
		_ = os.Remove(h.indexPath(name))
		return HelmRepoInfo{}, err
	}
	h.mu.Lock()
	h.repos = append(h.repos, r)
	h.gen++
	h.state[name] = &repoState{index: idx, updated: h.now(), gen: h.gen}
	h.mu.Unlock()
	h.publish()
	return h.infos([]string{name})[0], nil
}

// Remove forgets a repository and deletes its downloaded index. Releases installed from it are
// not touched.
func (h *HelmRepos) Remove(name string) error {
	h.fetchMu.Lock()
	defer h.fetchMu.Unlock()
	if err := h.ensureLoaded(); err != nil {
		return err
	}
	h.mu.Lock()
	repos := slices.DeleteFunc(slices.Clone(h.repos), func(r HelmRepo) bool { return r.Name == name })
	found := len(repos) != len(h.repos)
	h.mu.Unlock()
	if !found {
		return fmt.Errorf("no Helm repository named %s", name)
	}
	if err := h.save(repos); err != nil {
		return err
	}
	h.mu.Lock()
	h.repos = repos
	delete(h.state, name)
	h.mu.Unlock()
	if err := os.Remove(h.indexPath(name)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		h.logger().Warn("helm: the removed repository's index stays", "repository", name, "err", err)
	}
	h.publish()
	return nil
}

// Update downloads the indexes of the named repositories (all when names is empty), one after
// the other. A repository whose download fails keeps the index it had, and its error is in the
// result; the returned error joins the failures.
func (h *HelmRepos) Update(ctx context.Context, names []string) ([]HelmRepoInfo, error) {
	h.fetchMu.Lock()
	defer h.fetchMu.Unlock()
	if err := h.ensureLoaded(); err != nil {
		return nil, err
	}
	h.mu.Lock()
	var targets []HelmRepo
	for _, r := range h.repos {
		if len(names) == 0 || slices.Contains(names, r.Name) {
			targets = append(targets, r)
		}
	}
	h.mu.Unlock()
	for _, n := range names {
		if !slices.ContainsFunc(targets, func(r HelmRepo) bool { return r.Name == n }) {
			return nil, fmt.Errorf("no Helm repository named %s", n)
		}
	}
	var errs []error
	for _, r := range targets {
		if err := h.download(ctx, r); err != nil {
			errs = append(errs, fmt.Errorf("repository %s: %w", r.Name, err))
		}
	}
	h.publish()
	out := make([]string, len(targets))
	for i, r := range targets {
		out[i] = r.Name
	}
	return h.infos(out), errors.Join(errs...)
}

// download downloads one repository's index. The caller holds fetchMu.
func (h *HelmRepos) download(ctx context.Context, r HelmRepo) error {
	h.setUpdating(r.Name, true)
	idx, err := h.fetcher.DownloadIndex(ctx, r.URL, h.indexPath(r.Name))
	h.mu.Lock()
	defer h.mu.Unlock()
	st := h.state[r.Name]
	if st == nil { // removed meanwhile: cannot happen under fetchMu, but stay safe
		return err
	}
	st.updating = false
	if err != nil {
		st.err = err.Error()
		return err
	}
	h.gen++
	st.index, st.updated, st.err, st.gen = idx, h.now(), "", h.gen
	return nil
}

func (h *HelmRepos) setUpdating(name string, on bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if st := h.state[name]; st != nil {
		st.updating = on
	}
}

// refreshStale downloads the indexes older than their time-to-live (or never downloaded), while
// the Charts view or a Helm Install tab is open (architecture §5).
func (h *HelmRepos) refreshStale(ctx context.Context) {
	h.fetchMu.Lock()
	defer h.fetchMu.Unlock()
	if err := h.ensureLoaded(); err != nil {
		h.logger().Warn("helm: repositories", "err", err)
		return
	}
	h.mu.Lock()
	var stale []HelmRepo
	for _, r := range h.repos {
		if st := h.state[r.Name]; st == nil || st.updated.IsZero() || h.now().Sub(st.updated) >= h.ttl {
			stale = append(stale, r)
		}
	}
	h.mu.Unlock()
	changed := false
	for _, r := range stale {
		if ctx.Err() != nil {
			break
		}
		if err := h.download(ctx, r); err != nil {
			h.logger().Warn("helm: the repository's index was not refreshed", "repository", r.Name, "err", err)
			continue
		}
		changed = true
	}
	if changed {
		h.publish()
	}
}

// ChartVersion finds a chart's version in a repository's index: version "" is the version the
// chart is listed with (the newest that is not a prerelease). It returns the repository too.
func (h *HelmRepos) ChartVersion(repository, chart, version string) (HelmRepo, helm.IndexChart, helm.ChartVersion, error) {
	h.fetchMu.Lock()
	err := h.ensureLoaded()
	h.fetchMu.Unlock()
	if err != nil {
		return HelmRepo{}, helm.IndexChart{}, helm.ChartVersion{}, err
	}
	r, ok := h.repo(repository)
	if !ok {
		return HelmRepo{}, helm.IndexChart{}, helm.ChartVersion{}, fmt.Errorf("no Helm repository named %s", repository)
	}
	h.mu.Lock()
	var idx *helm.Index
	if st := h.state[repository]; st != nil {
		idx = st.index
	}
	h.mu.Unlock()
	c, ok := idx.Chart(chart)
	if !ok {
		return r, helm.IndexChart{}, helm.ChartVersion{}, fmt.Errorf("repository %s has no chart %s (helm_repo_update downloads its index again)", repository, chart)
	}
	var v helm.ChartVersion
	if version == "" {
		v, ok = c.Latest()
	} else {
		v, ok = c.Version(version)
	}
	if !ok {
		return r, c, helm.ChartVersion{}, fmt.Errorf("chart %s/%s has no version %s", repository, chart, version)
	}
	return r, c, v, nil
}

// FetchChart downloads a chart version from its repository, checked against the index's digest.
func (h *HelmRepos) FetchChart(ctx context.Context, repository, chart, version string) (*helm.Chart, helm.IndexChart, error) {
	r, c, v, err := h.ChartVersion(repository, chart, version)
	if err != nil {
		return nil, c, err
	}
	ch, err := h.fetcher.FetchChart(ctx, r.Name, r.URL, chart, v)
	return ch, c, err
}

func (h *HelmRepos) logger() *slog.Logger {
	h.mu.Lock()
	s := h.s
	h.mu.Unlock()
	if s != nil {
		return s.log
	}
	return slog.Default()
}

// attach binds the repositories to their session: the connected backend gets the charts, and
// the indexes are refreshed while the Charts view or a Helm Install tab is open. Nil-safe.
func (h *HelmRepos) attach(s *Session) {
	if h == nil {
		return
	}
	h.mu.Lock()
	h.s = s
	h.mu.Unlock()
	s.Subscribe(h.onEvent)
}

// onEvent runs on the goroutine that changed the UI state: it must not block.
func (h *HelmRepos) onEvent(ev Event) {
	if ev.Type != EventUIState || ev.UIState == nil {
		return
	}
	u := ev.UIState
	h.mu.Lock()
	if u.Revision <= h.lastRev {
		h.mu.Unlock()
		return
	}
	h.lastRev = u.Revision
	s := h.s
	wanted := u.Connection.State == Connected && (u.View == kinds.HelmChartsKey || slices.ContainsFunc(u.Dock, func(t DockTab) bool { return t.Type == DockHelmInstall }))
	var stop chan struct{}
	start := false
	switch {
	case wanted && h.poll == nil:
		h.poll, start = make(chan struct{}), true
		stop = h.poll
	case !wanted && h.poll != nil:
		close(h.poll)
		h.poll = nil
	}
	h.mu.Unlock()
	if s == nil {
		return
	}
	if b := s.Backend(); b != nil && u.Connection.State == Connected && h.needsPublish(b) {
		s.goSafe("Helm charts", func() {
			h.fetchMu.Lock()
			err := h.ensureLoaded()
			h.fetchMu.Unlock()
			if err != nil {
				s.log.Warn("helm: repositories", "err", err)
			}
			h.publish()
		})
	}
	if start {
		go h.pollLoop(s, stop)
	}
}

// pollLoop refreshes stale indexes at once and then every pollEvery, until stop is closed or the
// session ends. It is not one of the session's goroutines that Close waits for: a download it
// started ends with its own timeout.
func (h *HelmRepos) pollLoop(s *Session, stop chan struct{}) {
	defer s.recoverTo("Helm repository indexes")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		select {
		case <-stop:
			cancel()
		case <-ctx.Done():
		}
	}()
	t := time.NewTicker(h.pollEvery)
	defer t.Stop()
	for {
		if s.isClosed() {
			return
		}
		h.refreshStale(ctx)
		select {
		case <-stop:
			return
		case <-t.C:
		}
	}
}

// needsPublish reports whether b has not been given the charts yet.
func (h *HelmRepos) needsPublish(b *Backend) bool {
	h.pubMu.Lock()
	defer h.pubMu.Unlock()
	return h.pubTarget != b
}

// publish writes the charts of every downloaded index into the connected backend's local store as
// the local kind helmcharts: objects whose index changed are replaced, those of a removed
// repository or chart deleted. A backend without the kind (the demo cluster) gets none.
func (h *HelmRepos) publish() {
	h.mu.Lock()
	s := h.s
	h.mu.Unlock()
	if s == nil {
		return
	}
	b := s.Backend()
	h.pubMu.Lock()
	defer h.pubMu.Unlock()
	if b == nil {
		h.pubTarget, h.published = nil, nil
		return
	}
	p, ok := b.Profile(kinds.HelmChartsKey)
	if !ok || !p.Local {
		h.pubTarget, h.published = b, map[string]string{}
		return
	}
	if h.pubTarget != b {
		h.pubTarget, h.published = b, map[string]string{}
	}
	want := h.chartObjects()
	for name, obj := range want {
		if h.published[name] == obj.GetResourceVersion() {
			continue
		}
		if err := b.PutLocal(p, obj); err != nil {
			s.log.Warn("helm: a chart was not listed", "chart", name, "err", err)
			continue
		}
		h.published[name] = obj.GetResourceVersion()
	}
	for name := range h.published {
		if _, ok := want[name]; !ok {
			_ = b.DeleteLocal(p, "", name)
			delete(h.published, name)
		}
	}
}

// chartObjects makes the charts' objects from the downloaded indexes, by object name.
func (h *HelmRepos) chartObjects() map[string]*unstructured.Unstructured {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := map[string]*unstructured.Unstructured{}
	for _, r := range h.repos {
		st := h.state[r.Name]
		if st == nil || st.index == nil {
			continue
		}
		rv := strconv.FormatUint(st.gen, 10)
		for _, c := range st.index.Charts {
			latest, ok := c.Latest()
			if !ok {
				continue
			}
			kc := kinds.HelmChart{
				Repository: r.Name, Name: c.Name, Version: latest.Version, AppVersion: latest.AppVersion,
				Description: latest.Description, Home: latest.Home, Icon: latest.Icon, KubeVersion: latest.KubeVersion,
				Digest: latest.Digest, Keywords: latest.Keywords, Maintainers: latest.Maintainers, Sources: latest.Sources,
				Deprecated: latest.Deprecated, Created: latest.Created,
			}
			for _, v := range c.Versions {
				kc.Versions = append(kc.Versions, kinds.HelmChartVersion{Version: v.Version, AppVersion: v.AppVersion, Created: v.Created})
			}
			obj := kinds.HelmChartObject(kc, chartUID(r, c.Name), rv)
			out[obj.GetName()] = obj
		}
	}
	return out
}

// chartUID names a chart for as long as its repository is added: a repository removed and added
// again gives its charts new UIDs.
func chartUID(r HelmRepo, chart string) string {
	sum := sha256.Sum256([]byte(r.Name + "\x00" + r.Added.UTC().Format(time.RFC3339Nano) + "\x00" + chart))
	return hex.EncodeToString(sum[:16])
}

// HelmChartRef is a chart's reference in the Charts view.
func HelmChartRef(context, repository, chart string) model.ObjectRef {
	return model.ObjectRef{Context: context, Kind: kinds.HelmChartsKey, Name: kinds.HelmChartName(repository, chart)}
}
