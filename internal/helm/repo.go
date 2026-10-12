package helm

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	repo "helm.sh/helm/v4/pkg/repo/v1"
	"sigs.k8s.io/yaml"
)

// Chart repositories (G11, architecture §8): the HTTPS chart repositories the user adds by name
// and URL, their index (index.yaml) downloaded on add, on an explicit update and, while the Charts
// view or a Helm Install tab is open, when its time-to-live has passed (engine/helmrepos.go), and
// the chart archives an install takes from them. Everything here is bounded and parsed with the
// Helm SDK; nothing reads or writes Helm's own repositories.yaml or cache: the engine hands every
// path in, all of them under Cogpit's own folders. Public repositories only: no credentials, no
// client certificates, no plain HTTP (an authenticated repository is gap H-29).

// Bounds on what Cogpit downloads from a repository (Cogpit's defaults, not Helm's: Helm reads an
// index whole and bounds only a chart's decompressed size, 100 MiB).
const (
	// MaxIndexBytes bounds a repository's index.yaml. Large public indexes are a few tens of MiB.
	MaxIndexBytes int64 = 32 << 20
	// MaxChartBytes bounds a chart archive (.tgz) as downloaded.
	MaxChartBytes int64 = 16 << 20
	// fetchTimeout bounds one download, connection and body.
	fetchTimeout = 2 * time.Minute
	// maxRedirects bounds the redirects one download follows, each to https only.
	maxRedirects = 10
	// maxText bounds a chart's description and the other texts Cogpit keeps of an index entry.
	maxText = 512
	// maxListed bounds the keywords, maintainers and sources kept of an index entry.
	maxListed = 32
	// maxVersions bounds the versions kept per chart, newest first.
	maxVersions = 1000
)

// schemeHTTPS is the only scheme Cogpit downloads charts and indexes over.
const schemeHTTPS = "https"

// ErrNotHTTPS refuses a repository or chart URL that is not https: Cogpit reads public chart
// repositories over TLS only (no plain-HTTP override).
var ErrNotHTTPS = errors.New("only https URLs are allowed")

// ErrDigestMismatch refuses a chart archive whose SHA-256 differs from the digest its repository's
// index lists: the archive is not the one the index describes.
var ErrDigestMismatch = errors.New("the chart archive does not match the digest its repository's index lists")

// repoNamePattern is a repository's name as Cogpit keeps it: letters, digits, '-' and '_', no dot,
// so that "<repository>.<chart>" names a chart unambiguously, and safe as a file name.
var repoNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,62}$`)

// chartNamePattern is a chart's name in an index that Cogpit lists: Helm's chart names, without a
// path separator.
var chartNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,200}$`)

// CheckRepositoryName checks a repository's name.
func CheckRepositoryName(name string) error {
	if !repoNamePattern.MatchString(name) {
		return errors.New("a repository's name is 1 to 63 letters, digits, '-' or '_', starting with a letter or digit")
	}
	return nil
}

// CheckRepository checks a repository's name and URL as Cogpit keeps them and returns the URL
// normalised (no trailing slash). The URL is https, with a host, without credentials, a query or
// a fragment. An error never quotes the URL: a URL with credentials is refused, not echoed.
func CheckRepository(name, rawURL string) (string, error) {
	if err := CheckRepositoryName(name); err != nil {
		return "", err
	}
	u, err := url.Parse(strings.TrimSpace(rawURL))
	switch {
	case err != nil:
		return "", errors.New("the repository URL does not parse")
	case u.User != nil:
		return "", errors.New("a repository URL with a user or password is not supported: Cogpit adds public repositories only")
	case u.Scheme != schemeHTTPS:
		return "", fmt.Errorf("the repository URL: %w", ErrNotHTTPS)
	case u.Host == "" || u.Opaque != "":
		return "", errors.New("the repository URL names no host")
	case u.RawQuery != "" || u.Fragment != "" || u.ForceQuery:
		return "", errors.New("a repository URL has no query or fragment")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	return u.String(), nil
}

// Fetcher downloads repository indexes and chart archives over https, bounded. Its zero value is
// ready to use.
type Fetcher struct {
	// Client makes the requests; nil: a client with Cogpit's timeout and the environment's proxy
	// settings. Its redirects are followed to https URLs only, whatever the client allows.
	Client *http.Client
	// MaxIndex and MaxChart bound a download; zero means MaxIndexBytes and MaxChartBytes.
	MaxIndex, MaxChart int64
}

var defaultClient = &http.Client{Timeout: fetchTimeout}

func (f *Fetcher) client() *http.Client {
	base := defaultClient
	if f != nil && f.Client != nil {
		base = f.Client
	}
	c := *base
	next := base.CheckRedirect
	c.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != schemeHTTPS {
			return fmt.Errorf("a redirect: %w", ErrNotHTTPS)
		}
		if len(via) >= maxRedirects {
			return errors.New("too many redirects")
		}
		if next != nil {
			return next(req, via)
		}
		return nil
	}
	return &c
}

func (f *Fetcher) maxIndex() int64 {
	if f != nil && f.MaxIndex > 0 {
		return f.MaxIndex
	}
	return MaxIndexBytes
}

func (f *Fetcher) maxChart() int64 {
	if f != nil && f.MaxChart > 0 {
		return f.MaxChart
	}
	return MaxChartBytes
}

// get downloads an https URL, at most limit bytes.
func (f *Fetcher) get(ctx context.Context, rawURL string, limit int64) ([]byte, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("the URL does not parse")
	}
	if u.Scheme != schemeHTTPS {
		return nil, ErrNotHTTPS
	}
	if u.User != nil {
		return nil, errors.New("a URL with a user or password is not followed")
	}
	ctx, cancel := context.WithTimeout(ctx, fetchTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Cogpit")
	resp, err := f.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", u.Redacted(), resp.Status)
	}
	if resp.ContentLength > limit {
		return nil, fmt.Errorf("%s: %d bytes is more than the %d Cogpit downloads", u.Redacted(), resp.ContentLength, limit)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("%s: more than the %d bytes Cogpit downloads", u.Redacted(), limit)
	}
	return data, nil
}

// Index is what Cogpit keeps of a repository's index: its charts by name, each with its versions
// newest first (the Helm SDK's order).
type Index struct {
	Generated time.Time    `json:"generated"`
	Charts    []IndexChart `json:"charts"`
}

// IndexChart is one chart of an index.
type IndexChart struct {
	Name     string         `json:"name"`
	Versions []ChartVersion `json:"versions"`
}

// ChartVersion is one version of a chart as its repository's index lists it.
type ChartVersion struct {
	Version     string    `json:"version"`
	AppVersion  string    `json:"appVersion,omitempty"`
	Description string    `json:"description,omitempty"`
	Home        string    `json:"home,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	KubeVersion string    `json:"kubeVersion,omitempty"`
	Type        string    `json:"type,omitempty"`
	Keywords    []string  `json:"keywords,omitempty"`
	Maintainers []string  `json:"maintainers,omitempty"`
	Sources     []string  `json:"sources,omitempty"`
	Deprecated  bool      `json:"deprecated,omitempty"`
	Created     time.Time `json:"created,omitzero"`
	// Digest is the archive's SHA-256 as the index lists it; empty when the index lists none, and
	// the archive is then not verified.
	Digest string   `json:"digest,omitempty"`
	URLs   []string `json:"urls,omitempty"`
}

// Prerelease reports whether the version is a semantic-version prerelease ("1.2.0-rc.1"), which
// Helm's search leaves out unless asked for (--devel).
func (v ChartVersion) Prerelease() bool {
	core, _, _ := strings.Cut(v.Version, "+") // build metadata may hold a '-'
	return strings.Contains(core, "-")
}

// Latest returns the version a chart is listed with: the newest that is not a prerelease, as helm
// search shows it, else the newest.
func (c IndexChart) Latest() (ChartVersion, bool) {
	if len(c.Versions) == 0 {
		return ChartVersion{}, false
	}
	for _, v := range c.Versions {
		if !v.Prerelease() {
			return v, true
		}
	}
	return c.Versions[0], true
}

// Version returns a chart's version by its exact version string.
func (c IndexChart) Version(version string) (ChartVersion, bool) {
	i := slices.IndexFunc(c.Versions, func(v ChartVersion) bool { return v.Version == version })
	if i < 0 {
		return ChartVersion{}, false
	}
	return c.Versions[i], true
}

// Chart returns a chart of the index by name.
func (x *Index) Chart(name string) (IndexChart, bool) {
	if x == nil {
		return IndexChart{}, false
	}
	i, ok := slices.BinarySearchFunc(x.Charts, name, func(c IndexChart, n string) int { return strings.Compare(c.Name, n) })
	if !ok {
		return IndexChart{}, false
	}
	return x.Charts[i], true
}

// DownloadIndex downloads a repository's index.yaml (bounded), parses it with the Helm SDK and,
// only when it parses, replaces dest with it (0600, in a 0700 directory, atomically): a download
// that fails or does not parse leaves the previous index. It returns what Cogpit keeps of it.
func (f *Fetcher) DownloadIndex(ctx context.Context, repoURL, dest string) (*Index, error) {
	indexURL, err := repo.ResolveReferenceURL(repoURL, "index.yaml")
	if err != nil {
		return nil, errors.New("the repository URL does not parse")
	}
	data, err := f.get(ctx, indexURL, f.maxIndex())
	if err != nil {
		return nil, fmt.Errorf("downloading the index: %w", err)
	}
	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(dest)+"-*.tmp") // created 0600
	if err != nil {
		return nil, err
	}
	done := false
	defer func() {
		if !done {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		return nil, err
	}
	if err := tmp.Close(); err != nil {
		return nil, err
	}
	idx, err := loadIndex(tmp.Name(), f.maxIndex())
	if err != nil {
		return nil, err
	}
	if err := os.Rename(tmp.Name(), dest); err != nil {
		return nil, err
	}
	done = true
	return idx, nil
}

// LoadIndex reads an index Cogpit downloaded before (DownloadIndex) with the Helm SDK. A file
// larger than MaxIndexBytes is refused.
func LoadIndex(path string) (*Index, error) { return loadIndex(path, MaxIndexBytes) }

func loadIndex(path string, limit int64) (idx *Index, err error) {
	defer func() {
		if r := recover(); r != nil { // the index is untrusted input
			idx, err = nil, errors.New("the index could not be read")
		}
	}()
	fi, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, errors.New("the index is not a regular file")
	}
	if fi.Size() > limit {
		return nil, fmt.Errorf("the index has %d bytes, more than the %d Cogpit reads", fi.Size(), limit)
	}
	f, err := repo.LoadIndexFile(path)
	if err != nil {
		return nil, errors.New("the index is not a Helm repository index (index.yaml)")
	}
	return summarize(f), nil
}

// summarize keeps what the Charts view, the drawer and an install need of an index, bounded.
func summarize(f *repo.IndexFile) *Index {
	out := &Index{Generated: f.Generated}
	for name, versions := range f.Entries {
		if !chartNamePattern.MatchString(name) {
			continue
		}
		c := IndexChart{Name: name}
		for _, v := range versions {
			if v == nil || v.Metadata == nil || v.Version == "" || len(c.Versions) >= maxVersions {
				continue
			}
			md := v.Metadata
			cv := ChartVersion{
				Version: clip(md.Version), AppVersion: clip(md.AppVersion), Description: clip(md.Description),
				Home: clip(md.Home), Icon: clip(md.Icon), KubeVersion: clip(md.KubeVersion), Type: clip(md.Type),
				Keywords: clipList(md.Keywords), Sources: clipList(md.Sources), Deprecated: md.Deprecated,
				Created: v.Created, Digest: clip(v.Digest), URLs: clipList(v.URLs),
			}
			for _, m := range md.Maintainers {
				if m != nil && m.Name != "" && len(cv.Maintainers) < maxListed {
					cv.Maintainers = append(cv.Maintainers, clip(m.Name))
				}
			}
			c.Versions = append(c.Versions, cv)
		}
		if len(c.Versions) > 0 {
			out.Charts = append(out.Charts, c)
		}
	}
	slices.SortFunc(out.Charts, func(a, b IndexChart) int { return strings.Compare(a.Name, b.Name) })
	return out
}

func clip(s string) string {
	if len(s) <= maxText {
		return s
	}
	s = s[:maxText]
	for len(s) > 0 && !utf8.ValidString(s) { // a rune cut in half
		s = s[:len(s)-1]
	}
	return s + "…"
}

func clipList(in []string) []string {
	var out []string
	for _, s := range in {
		if s != "" && len(out) < maxListed {
			out = append(out, clip(s))
		}
	}
	return out
}

// Chart is a chart archive downloaded from a repository and loaded with the Helm SDK: what an
// install renders and installs.
type Chart struct {
	chart *chart.Chart

	Repository  string `json:"repository"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	AppVersion  string `json:"appVersion,omitempty"`
	Description string `json:"description,omitempty"`
	// Digest is the archive's SHA-256 (hex); Verified says it matched the digest its repository's
	// index lists, false when the index lists none.
	Digest   string `json:"digest"`
	Verified bool   `json:"verified"`
	Size     int    `json:"size"`
}

// FetchChart downloads a chart version's archive from its repository (the index's first URL,
// resolved against the repository's URL; https only, bounded), checks it against the index's
// digest when the index lists one, and loads it with the Helm SDK. The archive must hold the
// chart and version the index names.
func (f *Fetcher) FetchChart(ctx context.Context, repository, repoURL, name string, v ChartVersion) (*Chart, error) {
	if len(v.URLs) == 0 {
		return nil, fmt.Errorf("chart %s %s: the index lists no URL", name, v.Version)
	}
	u, err := repo.ResolveReferenceURL(repoURL, v.URLs[0])
	if err != nil {
		return nil, fmt.Errorf("chart %s %s: the URL does not parse", name, v.Version)
	}
	data, err := f.get(ctx, u, f.maxChart())
	if err != nil {
		return nil, fmt.Errorf("chart %s %s: %w", name, v.Version, err)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	verified := false
	if want := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(v.Digest)), "sha256:"); want != "" {
		if want != digest {
			return nil, fmt.Errorf("chart %s %s: %w", name, v.Version, ErrDigestMismatch)
		}
		verified = true
	}
	c, err := LoadChartArchive(data)
	if err != nil {
		return nil, fmt.Errorf("chart %s %s: %w", name, v.Version, err)
	}
	if c.Name != name || c.Version != v.Version {
		return nil, fmt.Errorf("chart %s %s: the archive holds chart %s %s", name, v.Version, c.Name, c.Version)
	}
	c.Repository, c.Digest, c.Verified = repository, digest, verified
	return c, nil
}

// LoadChartArchive loads a chart archive (.tgz) with the Helm SDK. The archive is untrusted input:
// the SDK bounds its decompressed size, and a loader that panics is an error.
func LoadChartArchive(data []byte) (c *Chart, err error) {
	defer func() {
		if r := recover(); r != nil {
			c, err = nil, errors.New("the chart archive could not be read")
		}
	}()
	ch, err := loader.LoadArchive(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("the chart archive could not be read: %w", err)
	}
	if ch.Metadata == nil {
		return nil, errors.New("the chart archive holds no Chart.yaml")
	}
	sum := sha256.Sum256(data)
	return &Chart{
		chart: ch, Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion,
		Description: clip(ch.Metadata.Description), Digest: hex.EncodeToString(sum[:]), Size: len(data),
	}, nil
}

// DefaultValues returns the chart's values.yaml as the chart ships it, comments included: what a
// Helm Install tab starts with. A chart without one has none; one whose file is missing from the
// archive's raw files is written from its parsed values.
func (c *Chart) DefaultValues() string {
	if c == nil || c.chart == nil {
		return ""
	}
	for _, f := range c.chart.Raw {
		if f != nil && f.Name == "values.yaml" {
			return string(f.Data)
		}
	}
	if len(c.chart.Values) == 0 {
		return ""
	}
	y, err := yaml.Marshal(c.chart.Values)
	if err != nil {
		return ""
	}
	return string(y)
}

// Defaults returns the chart's parsed default values (shared: do not modify).
func (c *Chart) Defaults() map[string]any {
	if c == nil || c.chart == nil {
		return nil
	}
	return c.chart.Values
}

// checkInstallable refuses what helm install refuses before rendering: a library chart and a
// chart whose dependencies are not packaged with it (Cogpit does not fetch dependencies).
func (c *Chart) checkInstallable() error {
	if c == nil || c.chart == nil || c.chart.Metadata == nil {
		return errors.New("no chart")
	}
	if t := c.chart.Metadata.Type; t != "" && t != "application" {
		return fmt.Errorf("chart %s is a %s chart, which cannot be installed", c.Name, t)
	}
	var missing []string
	for _, d := range c.chart.Metadata.Dependencies {
		if d == nil {
			continue
		}
		if !slices.ContainsFunc(c.chart.Dependencies(), func(sub *chart.Chart) bool { return sub.Name() == d.Name }) {
			missing = append(missing, d.Name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("chart %s lacks its dependencies %s in its charts/ folder", c.Name, strings.Join(missing, ", "))
	}
	return nil
}
