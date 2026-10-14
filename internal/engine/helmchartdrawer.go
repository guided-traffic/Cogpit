package engine

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// A Helm chart's drawer (G11, HELM2; resource-parity › Helm). Lens's chart details show the
// chart's README and let the user pick a version (inferred from Lens's documentation of the Charts
// view, not captured). Cogpit's drawer shows one version of the chart: the one it is listed with,
// or the one ui.setChartVersion picked (UIState.DrawerChartVersion, which ends with the drawer),
// its properties from the repository's index and its README from its archive.
//
// The README is read only when it is asked for: when the drawer shows the chart (the GUI's Detail
// of the drawer's object) or when an agent reads the chart (object_get, DetailWait), never for the
// list. The archive is downloaded as an install downloads it (HelmRepos.FetchChart: the
// repository's credentials only for its own scheme, host and port unless pass_credentials_all,
// https stays https, at most helm.MaxChartBytes, checked against the index's digest), and the
// README is cut at helm.MaxReadmeBytes. A README is kept in memory (the last maxReadmesKept) and,
// when the index's digest verified its archive, in Cogpit's own cache folder
// (<cache>/helm/readmes/<key>.json, owner-only, at most maxReadmeFiles), keyed by the repository's
// name and URL, the chart, the version and the digest: a version republished with another archive
// is another key, and an archive no digest verified is read again in the next session. A drawer
// that moves to another chart stops the download it started. The text is public chart content:
// agents read it in object_get's "README" section (Cogpit's default), as untrusted text a chart's
// author wrote (docs/security/agents-and-mcp.md, H-86).

// The states of a chart's README (ChartReadme.State).
const (
	// ReadmeLoaded: the README section holds the text.
	ReadmeLoaded = "loaded"
	// ReadmeLoading: the version's archive is downloading; an EventDetail follows when it ends.
	ReadmeLoading = "loading"
	// ReadmeNone: the archive holds no README.
	ReadmeNone = "none"
	// ReadmeFailed: the download or the archive failed; Error says why. The drawer tries again
	// once readmeRetryAfter has passed.
	ReadmeFailed = "error"
	// ReadmeNotLoaded: nobody asked for it: the drawer does not show the chart.
	ReadmeNotLoaded = "notLoaded"
)

// ReadmeSection is the title of the drawer section that holds a chart's README.
const ReadmeSection = "README"

// Bounds of the READMEs (Cogpit's defaults).
const (
	maxReadmesKept   = 32               // READMEs held in memory
	maxReadmeFiles   = 256              // READMEs kept in the cache folder; the oldest go first
	readmeRetryAfter = 30 * time.Second // a failed README is downloaded again after this, when asked
	readmeAgentWait  = 30 * time.Second // the longest object_get waits for a README
	// maxReadmeFileBytes bounds a README file read from the cache folder: the text, JSON-escaped.
	maxReadmeFileBytes = 3*helm.MaxReadmeBytes + 4<<10
)

// HelmChartDrawer is what a Helm chart's drawer adds to its model (DetailModel.HelmChart).
type HelmChartDrawer struct {
	Repository string `json:"repository"`
	Chart      string `json:"chart"`
	// Version is the version the drawer shows; Listed the one the chart is listed with (the
	// newest that is not a prerelease).
	Version string `json:"version"`
	Listed  string `json:"listed"`
	// Versions are every version the repository's index lists, newest first: what ui.setChartVersion
	// takes.
	Versions []string `json:"versions"`
	// Readme is the state of Version's README; its text is the drawer's "README" section.
	Readme ChartReadme `json:"readme"`
}

// ChartReadme is the state of a chart version's README.
type ChartReadme struct {
	// State is ReadmeLoaded, ReadmeLoading, ReadmeNone, ReadmeFailed or ReadmeNotLoaded.
	State string `json:"state"`
	// File is its name in the archive ("README.md").
	File string `json:"file,omitempty"`
	// Size is its size in the archive in bytes; Truncated says the text is its first 256 KiB.
	Size      int  `json:"size,omitempty"`
	Truncated bool `json:"truncated,omitempty"`
	// Verified: the archive it was read from matched the digest the repository's index lists.
	Verified bool `json:"verified,omitempty"`
	// Error says why it could not be read (ReadmeFailed).
	Error string `json:"error,omitempty"`
}

// helmChartDrawer adds a Helm chart's drawer its version and README; m of any other kind, or of a
// session without chart repositories, is returned as it is. The README is asked for (downloaded
// when it is not kept) when the drawer shows ref or when wait is set (agents); wait also waits for
// it, bounded by ctx and readmeAgentWait.
func (s *Session) helmChartDrawer(ctx context.Context, p *model.KindProfile, ref model.ObjectRef, obj *unstructured.Unstructured, m *DetailModel, wait bool) *DetailModel {
	h := s.HelmRepos()
	if h == nil || m == nil || p.Key != kinds.HelmChartsKey || !p.Local {
		return m
	}
	repository, chart, ok := kinds.SplitHelmChartName(ref.Name)
	if !ok {
		return m
	}
	listed := model.NestedString(obj, "spec", "version")
	shown, version := s.drawerChart(ref)
	if version != "" && version != listed {
		at := h.chartObjectAt(repository, chart, version, obj)
		if at == nil {
			version = "" // gone from the index since it was picked: the listed version
		} else if vm, _ := BuildDetailIn(p, ref, at, nil, s.displayNow(), nil, s.TimeFormat()); vm != nil {
			m = vm
		}
	}
	if version == "" {
		version = listed
	}
	hc := HelmChartDrawer{Repository: repository, Chart: chart, Version: version, Listed: listed, Versions: chartVersions(obj)}
	readme, text := h.readmeOf(ctx, s, repository, chart, version, shown || wait, shown && !wait, wait)
	hc.Readme = readme
	AddHelmChartDrawer(m, hc, text)
	return m
}

// AddHelmChartDrawer adds a chart drawer's parts to m: hc, and the README section with text (when
// hc.Readme is loaded; else what the state says) before the Versions table. Session.Detail adds
// them to a chart's model; tests of the UI build chart drawers with it.
func AddHelmChartDrawer(m *DetailModel, hc HelmChartDrawer, text string) {
	m.HelmChart = &hc
	m.Sections = withReadmeSection(m.Sections, hc.Readme, text)
}

// withReadmeSection puts the README section before the Versions table (after the description),
// or last.
func withReadmeSection(sections []model.DetailSection, r ChartReadme, text string) []model.DetailSection {
	var value string
	switch r.State {
	case ReadmeLoaded:
		value = text
	case ReadmeLoading:
		value = "Loading the README…"
	case ReadmeNone:
		value = "The chart has no README."
	case ReadmeFailed:
		value = "The README could not be loaded: " + r.Error
	default:
		value = "The README loads when the chart's drawer shows it."
	}
	sec := model.DetailSection{Title: ReadmeSection, Items: []model.DetailItem{{Value: value}}}
	i := slices.IndexFunc(sections, func(s model.DetailSection) bool { return s.Title == "Versions" })
	if i < 0 {
		return append(sections, sec)
	}
	return slices.Insert(sections, i, sec)
}

// chartVersions are the versions a chart's object lists, newest first.
func chartVersions(obj *unstructured.Unstructured) []string {
	var out []string
	for _, v := range model.NestedSlice(obj, "spec", "versions") {
		if s := model.NestedString(v, "version"); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// chartObjectAt is a chart's object at another version than the one it is listed with: that
// version's properties from the repository's index under the object's name, UID and
// resourceVersion; nil when the index does not list the version.
func (h *HelmRepos) chartObjectAt(repository, chart, version string, obj *unstructured.Unstructured) *unstructured.Unstructured {
	_, c, v, err := h.ChartVersion(repository, chart, version)
	if err != nil {
		return nil
	}
	return kinds.HelmChartObject(helmChartAt(repository, c, v), string(obj.GetUID()), obj.GetResourceVersion())
}

// drawerChart reports whether the drawer shows ref, and the version it shows of it ("" the listed
// one).
func (s *Session) drawerChart(ref model.ObjectRef) (shown bool, version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ui.Drawer == nil || !s.ui.Drawer.Same(ref) {
		return false, ""
	}
	return true, s.ui.DrawerChartVersion
}

// DrawerChartVersionOf returns the version the drawer shows of the chart ref: "" when it shows
// the version the chart is listed with, or another object. Install from the drawer installs it.
func (s *Session) DrawerChartVersionOf(ref model.ObjectRef) string {
	_, v := s.drawerChart(ref)
	return v
}

// SetDrawerChartVersion makes the drawer, which must show the chart ref, show another version of
// it: one its object lists; "" or the listed version shows the listed one. The drawer's model
// changes at once (EventDetail) and the version's README is downloaded when it is not kept.
func (s *Session) SetDrawerChartVersion(ref model.ObjectRef, version string) (UIState, error) {
	ref, p, err := s.ResolveRef(ref)
	if err != nil {
		return UIState{}, err
	}
	if p.Key != kinds.HelmChartsKey {
		return UIState{}, fmt.Errorf("%w: %s is not a Helm chart", ErrNotApplicable, ref.String())
	}
	if shown, _ := s.drawerChart(ref); !shown {
		return UIState{}, ErrDrawerNotOpen
	}
	obj, err := s.Lookup(ref)
	if err != nil {
		return UIState{}, err
	}
	versions := chartVersions(obj)
	if version != "" && !slices.Contains(versions, version) {
		return UIState{}, fmt.Errorf("chart %s has no version %q; its versions: %s", ref.Name, version, strings.Join(versions[:min(len(versions), 20)], ", "))
	}
	if version == model.NestedString(obj, "spec", "version") {
		version = ""
	}
	shown := false
	u := s.UpdateUI(func(u *UIState) {
		if u.Drawer != nil && u.Drawer.Same(ref) {
			u.DrawerChartVersion, shown = version, true
		}
	})
	if !shown {
		return u, ErrDrawerNotOpen
	}
	s.emit(Event{Type: EventDetail, View: kinds.HelmChartsKey})
	return u, nil
}

// syncDrawerChartVersion ends a chart drawer's version when the drawer closes or shows another
// object, unless the same change set a version for it.
func (u *UIState) syncDrawerChartVersion(old UIState) {
	switch {
	case u.DrawerChartVersion == "":
	case u.Drawer == nil || u.Drawer.Kind != kinds.HelmChartsKey:
		u.DrawerChartVersion = ""
	case u.DrawerChartVersion == old.DrawerChartVersion && (old.Drawer == nil || *old.Drawer != *u.Drawer):
		u.DrawerChartVersion = ""
	}
}

// readmeKey names a chart version's README: the repository (its cache key: name and URL), the
// chart, the version and the archive's digest as the index lists it (lower case, without
// "sha256:"; empty when it lists none).
type readmeKey struct{ repo, chart, version, digest string }

// file is the README's file name in the cache folder.
func (k readmeKey) file() string {
	sum := sha256.Sum256([]byte(k.repo + "\x00" + k.chart + "\x00" + k.version + "\x00" + k.digest))
	return hex.EncodeToString(sum[:16]) + ".json"
}

// readmeEntry is a README held in memory: loaded, failed, or loading.
type readmeEntry struct {
	key      readmeKey
	readme   helm.Readme
	verified bool
	err      string
	at       time.Time // when its load ended
	loading  bool
	done     chan struct{} // closed when its load ends
	cancel   context.CancelFunc
	waiters  int    // agents waiting for it: a drawer that moves on does not stop it
	used     uint64 // the store's clock at its last use
}

// readmeStore holds the READMEs of a session's chart drawer. Its zero value with dir set is ready.
type readmeStore struct {
	dir     string // the cache folder; empty: memory only
	mu      sync.Mutex
	entries map[readmeKey]*readmeEntry
	clock   uint64
	drawer  *readmeEntry // the load the drawer started last
}

// readmeOf returns the state of a chart version's README and its text, asking for it (a download
// unless it is kept) when start is set. A load the drawer starts (drawer) stops the drawer's
// previous one; wait waits for it.
func (h *HelmRepos) readmeOf(ctx context.Context, s *Session, repository, chart, version string, start, drawer, wait bool) (ChartReadme, string) {
	r, _, v, err := h.ChartVersion(repository, chart, version)
	if err != nil {
		return ChartReadme{State: ReadmeFailed, Error: err.Error()}, ""
	}
	rs := &h.readmes
	key := readmeKey{repo: r.CacheKey(), chart: chart, version: v.Version,
		digest: strings.TrimPrefix(strings.ToLower(strings.TrimSpace(v.Digest)), "sha256:")}
	e := rs.get(key, h.now())
	if e == nil {
		if !start {
			return ChartReadme{State: ReadmeNotLoaded}, ""
		}
		e = rs.start(key, drawer, h.now, func(ctx context.Context) (helm.Readme, bool, error) {
			ch, err := h.fetcher.FetchChart(ctx, r, chart, v)
			if err != nil {
				return helm.Readme{}, false, err
			}
			return ch.Readme(), ch.Verified, nil
		}, func() { s.emit(Event{Type: EventDetail, View: kinds.HelmChartsKey}) })
	}
	if wait {
		rs.wait(ctx, e, readmeAgentWait)
	}
	return rs.view(e)
}

// get returns key's README from memory or the cache folder; nil when it has none, or when its last
// load failed readmeRetryAfter ago or longer.
func (rs *readmeStore) get(key readmeKey, now time.Time) *readmeEntry {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if e := rs.entries[key]; e != nil {
		if !e.loading && e.err != "" && now.Sub(e.at) >= readmeRetryAfter {
			delete(rs.entries, key)
			return nil
		}
		rs.clock++
		e.used = rs.clock
		return e
	}
	if key.digest == "" || rs.dir == "" {
		return nil
	}
	r, ok := readReadmeFile(filepath.Join(rs.dir, key.file()))
	if !ok {
		return nil
	}
	e := &readmeEntry{key: key, readme: r, verified: true, at: now, done: closedDone}
	rs.put(e)
	return e
}

// closedDone is the done channel of an entry read from the cache folder.
var closedDone = func() chan struct{} { c := make(chan struct{}); close(c); return c }()

// put holds e, dropping the least recently used finished entry beyond maxReadmesKept. The caller
// holds mu.
func (rs *readmeStore) put(e *readmeEntry) {
	if rs.entries == nil {
		rs.entries = map[readmeKey]*readmeEntry{}
	}
	rs.clock++
	e.used = rs.clock
	rs.entries[e.key] = e
	for len(rs.entries) > maxReadmesKept {
		var oldest *readmeEntry
		for _, c := range rs.entries {
			if !c.loading && (oldest == nil || c.used < oldest.used) {
				oldest = c
			}
		}
		if oldest == nil {
			return
		}
		delete(rs.entries, oldest.key)
	}
}

// start loads key with load on a goroutine of its own and calls done when it ended (not when it
// was stopped). A load the drawer starts stops the drawer's previous load of another key, unless
// an agent waits for that one. A load that panics fails.
func (rs *readmeStore) start(key readmeKey, drawer bool, now func() time.Time, load func(context.Context) (helm.Readme, bool, error), done func()) *readmeEntry {
	rs.mu.Lock()
	if e := rs.entries[key]; e != nil {
		rs.mu.Unlock()
		return e // another caller started it meanwhile
	}
	if prev := rs.drawer; drawer && prev != nil && prev.loading && prev.waiters == 0 && prev.key != key {
		prev.cancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	e := &readmeEntry{key: key, loading: true, done: make(chan struct{}), cancel: cancel}
	rs.put(e)
	if drawer {
		rs.drawer = e
	}
	rs.mu.Unlock()
	go func() {
		defer cancel()
		var (
			r        helm.Readme
			verified bool
			err      = errors.New("internal error")
		)
		defer func() {
			if p := recover(); p != nil {
				err = fmt.Errorf("internal error: %v", p)
			}
			if rs.finish(e, r, verified, err, ctx.Err() != nil, now()) {
				done()
			}
		}()
		r, verified, err = load(ctx)
	}()
	return e
}

// finish records a load's end and reports whether it ended on its own: a stopped load is dropped,
// so that the drawer asks again when it comes back. A README the index's digest verified (or that
// it has none) is kept in the cache folder first, so that it is there once the entry shows it.
func (rs *readmeStore) finish(e *readmeEntry, r helm.Readme, verified bool, err error, stopped bool, now time.Time) bool {
	if !stopped && err == nil && verified && e.key.digest != "" && rs.dir != "" {
		rs.keep(e.key, r)
	}
	rs.mu.Lock()
	e.loading, e.at = false, now
	switch {
	case stopped:
		if rs.entries[e.key] == e {
			delete(rs.entries, e.key)
		}
		e.err = "stopped: the drawer shows another chart"
	case err != nil:
		e.err = err.Error()
	default:
		e.readme, e.verified = r, verified
	}
	if rs.drawer == e {
		rs.drawer = nil
	}
	close(e.done)
	rs.mu.Unlock()
	return !stopped
}

// wait waits for e's load, bounded by ctx and most.
func (rs *readmeStore) wait(ctx context.Context, e *readmeEntry, most time.Duration) {
	rs.mu.Lock()
	e.waiters++
	rs.mu.Unlock()
	defer func() {
		rs.mu.Lock()
		e.waiters--
		rs.mu.Unlock()
	}()
	t := time.NewTimer(most)
	defer t.Stop()
	select {
	case <-e.done:
	case <-ctx.Done():
	case <-t.C:
	}
}

// view is e's state and text.
func (rs *readmeStore) view(e *readmeEntry) (ChartReadme, string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	switch {
	case e.loading:
		return ChartReadme{State: ReadmeLoading}, ""
	case e.err != "":
		return ChartReadme{State: ReadmeFailed, Error: e.err}, ""
	case e.readme.File == "":
		return ChartReadme{State: ReadmeNone, Verified: e.verified}, ""
	}
	r := e.readme
	return ChartReadme{State: ReadmeLoaded, File: r.File, Size: r.Size, Truncated: r.Truncated, Verified: e.verified}, r.Text
}

// readmeFile is a README as the cache folder keeps it.
type readmeFile struct {
	File      string `json:"file"`
	Text      string `json:"text"`
	Size      int    `json:"size"`
	Truncated bool   `json:"truncated,omitempty"`
}

// keep writes a verified README into the cache folder (owner-only, atomically) and removes the
// oldest beyond maxReadmeFiles. A failure costs a download in the next session only.
func (rs *readmeStore) keep(key readmeKey, r helm.Readme) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(readmeFile{File: r.File, Text: r.Text, Size: r.Size, Truncated: r.Truncated}); err != nil {
		return
	}
	if err := writePrivateFile(filepath.Join(rs.dir, key.file()), buf.Bytes()); err != nil {
		return
	}
	pruneReadmeFiles(rs.dir, maxReadmeFiles)
}

// readmeFileName reports whether name is a kept README's file name: 32 hex digits and ".json".
func readmeFileName(name string) bool {
	base, ok := strings.CutSuffix(name, ".json")
	if !ok || len(base) != 32 {
		return false
	}
	_, err := hex.DecodeString(base)
	return err == nil && strings.ToLower(base) == base
}

// pruneReadmeFiles removes the oldest READMEs of dir beyond most.
func pruneReadmeFiles(dir string, most int) {
	des, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	type kept struct {
		name string
		at   time.Time
	}
	var files []kept
	for _, de := range des {
		if !de.Type().IsRegular() || !readmeFileName(de.Name()) {
			continue
		}
		if fi, err := de.Info(); err == nil {
			files = append(files, kept{de.Name(), fi.ModTime()})
		}
	}
	if len(files) <= most {
		return
	}
	slices.SortFunc(files, func(a, b kept) int { return a.at.Compare(b.at) })
	for _, f := range files[:len(files)-most] {
		_ = os.Remove(filepath.Join(dir, f.name))
	}
}

// readReadmeFile reads a README the cache folder keeps, checked again as if it were downloaded: a
// regular file, bounded, its text cleaned and within helm.MaxReadmeBytes, named as a README (or
// no file and no text: the archive holds none).
func readReadmeFile(path string) (helm.Readme, bool) {
	fi, err := os.Lstat(path)
	if err != nil || !fi.Mode().IsRegular() || fi.Size() > maxReadmeFileBytes {
		return helm.Readme{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) > maxReadmeFileBytes {
		return helm.Readme{}, false
	}
	var f readmeFile
	if err := json.Unmarshal(data, &f); err != nil || len(f.Text) > helm.MaxReadmeBytes || f.Size < 0 ||
		(!helm.IsReadmeName(f.File) && (f.File != "" || f.Text != "")) {
		return helm.Readme{}, false
	}
	if f.File == "" {
		return helm.Readme{}, true
	}
	text, cut := helm.CleanReadme([]byte(f.Text))
	return helm.Readme{File: f.File, Text: text, Size: max(f.Size, len(text)), Truncated: f.Truncated || cut}, true
}
