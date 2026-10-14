package testkit

import (
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	chartcommon "helm.sh/helm/v4/pkg/chart/common"
	chart "helm.sh/helm/v4/pkg/chart/v2"
	chartutil "helm.sh/helm/v4/pkg/chart/v2/util"
	repo "helm.sh/helm/v4/pkg/repo/v1"
	"sigs.k8s.io/yaml"
)

// A local Helm chart repository for tests (G11): charts built and packaged in the test with the
// Helm SDK, an index.yaml the SDK writes over them, served by httptest over TLS. No test reaches a
// repository on the internet; the repository's own client trusts the test server's certificate.

// TestChart is a chart HelmRepo builds and packages.
type TestChart struct {
	Name, Version, AppVersion, Description string
	// Values is the chart's values.yaml as text; Templates maps "templates/<file>" to its text.
	Values    string
	Templates map[string]string
	// Type is the chart's type ("" or "application", "library").
	Type string
	// Readme is a README at the chart's root, named ReadmeName ("README.md" when empty); none when
	// empty. Icon is Chart.yaml's icon URL, which the index lists.
	Readme, ReadmeName, Icon string
}

// HelmRepo is a chart repository served by an httptest TLS server.
type HelmRepo struct {
	Server *httptest.Server
	// URL is the repository's URL (https, the server's root).
	URL string
	dir string
	t   testing.TB

	mu         sync.Mutex
	requests   map[string]int
	authorized map[string]int    // path → requests that carried an Authorization header
	override   map[string][]byte // path → served bytes, in place of the file
	failing    bool
	user, pass string // RequireBasicAuth's; empty: no login
}

// chartModTime is every packaged file's time, so that an archive and its digest are the same in
// every run.
var chartModTime = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

// NewHelmRepo packages charts into a temporary directory, indexes them with the Helm SDK and
// serves the directory over TLS until the test ends.
func NewHelmRepo(t testing.TB, charts ...TestChart) *HelmRepo {
	t.Helper()
	r := &HelmRepo{dir: t.TempDir(), t: t, requests: map[string]int{}, authorized: map[string]int{}, override: map[string][]byte{}}
	r.Server = httptest.NewTLSServer(http.HandlerFunc(r.serve))
	t.Cleanup(r.Server.Close)
	r.URL = r.Server.URL
	r.Add(charts...)
	return r
}

// NewPlainHelmRepo is NewHelmRepo served over plain http, unencrypted (credentials H-40): its URL
// is http://, and its Client is a plain HTTP client.
func NewPlainHelmRepo(t testing.TB, charts ...TestChart) *HelmRepo {
	t.Helper()
	r := &HelmRepo{dir: t.TempDir(), t: t, requests: map[string]int{}, authorized: map[string]int{}, override: map[string][]byte{}}
	r.Server = httptest.NewServer(http.HandlerFunc(r.serve))
	t.Cleanup(r.Server.Close)
	r.URL = r.Server.URL
	r.Add(charts...)
	return r
}

// Add packages more charts and indexes the repository again.
func (r *HelmRepo) Add(charts ...TestChart) {
	r.t.Helper()
	for _, c := range charts {
		if _, err := chartutil.Save(BuildChart(r.t, c), r.dir); err != nil {
			r.t.Fatal(err)
		}
	}
	idx, err := repo.IndexDirectory(r.dir, r.URL)
	if err != nil {
		r.t.Fatal(err)
	}
	idx.SortEntries()
	idx.Generated = chartModTime
	if err := idx.WriteFile(filepath.Join(r.dir, "index.yaml"), 0o644); err != nil {
		r.t.Fatal(err)
	}
}

// BuildChart builds a chart in memory: Chart.yaml from c, its values.yaml and templates.
func BuildChart(t testing.TB, c TestChart) *chart.Chart {
	t.Helper()
	vals := map[string]any{}
	if c.Values != "" {
		if err := yaml.Unmarshal([]byte(c.Values), &vals); err != nil {
			t.Fatal(err)
		}
	}
	ch := &chart.Chart{
		Metadata: &chart.Metadata{APIVersion: chart.APIVersionV2, Name: c.Name, Version: c.Version, AppVersion: c.AppVersion,
			Description: c.Description, Type: c.Type, Icon: c.Icon},
		Values:  vals,
		ModTime: chartModTime,
	}
	if c.Values != "" {
		ch.Raw = []*chartcommon.File{{Name: chartutil.ValuesfileName, Data: []byte(c.Values), ModTime: chartModTime}}
	}
	for name, text := range c.Templates {
		ch.Templates = append(ch.Templates, &chartcommon.File{Name: name, Data: []byte(text), ModTime: chartModTime})
	}
	if c.Readme != "" {
		name := c.ReadmeName
		if name == "" {
			name = "README.md"
		}
		ch.Files = append(ch.Files, &chartcommon.File{Name: name, Data: []byte(c.Readme), ModTime: chartModTime})
	}
	return ch
}

// Client is an HTTP client that trusts the repository's certificate.
func (r *HelmRepo) Client() *http.Client { return r.Server.Client() }

// Requests returns how often a path ("/index.yaml") was asked for.
func (r *HelmRepo) Requests(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.requests[path]
}

// Serve makes the repository answer path with data instead of its file; nil serves the file again.
func (r *HelmRepo) Serve(path string, data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if data == nil {
		delete(r.override, path)
		return
	}
	r.override[path] = data
}

// RequireBasicAuth makes the repository answer 401 to a request without this user and password,
// as a repository behind a login does.
func (r *HelmRepo) RequireBasicAuth(user, pass string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.user, r.pass = user, pass
}

// Authorized returns how often a path was asked for with an Authorization header, right or wrong.
func (r *HelmRepo) Authorized(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.authorized[path]
}

// CertificatePEM returns the repository server's certificate as PEM: a CA file that trusts it.
func (r *HelmRepo) CertificatePEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: r.Server.Certificate().Raw})
}

// Fail makes every request fail with 503 (true) or answer again (false).
func (r *HelmRepo) Fail(failing bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.failing = failing
}

// Archive returns a packaged chart's bytes ("web-0.1.0.tgz").
func (r *HelmRepo) Archive(name string) []byte {
	r.t.Helper()
	data, err := os.ReadFile(filepath.Join(r.dir, name))
	if err != nil {
		r.t.Fatal(err)
	}
	return data
}

func (r *HelmRepo) serve(w http.ResponseWriter, req *http.Request) {
	r.mu.Lock()
	r.requests[req.URL.Path]++
	if req.Header.Get("Authorization") != "" {
		r.authorized[req.URL.Path]++
	}
	data, overridden := r.override[req.URL.Path]
	failing := r.failing
	wantUser, wantPass := r.user, r.pass
	r.mu.Unlock()
	user, pass, ok := req.BasicAuth()
	switch {
	case wantUser != "" && (!ok || user != wantUser || pass != wantPass):
		w.Header().Set("WWW-Authenticate", `Basic realm="charts"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	case failing:
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
	case overridden:
		_, _ = w.Write(data)
	case strings.Contains(req.URL.Path, ".."):
		http.NotFound(w, req)
	default:
		http.ServeFile(w, req, filepath.Join(r.dir, filepath.FromSlash(strings.TrimPrefix(req.URL.Path, "/"))))
	}
}
