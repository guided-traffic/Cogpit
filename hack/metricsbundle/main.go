// Command metricsbundle puts the chart archive of Cogpit Metrics into internal/helm/bundle (D-61):
// prometheus-community's prometheus chart at the version internal/helm/metricsstack.go pins
// (MetricsChartVersion), downloaded from the repository's index's URL and checked against the
// SHA-256 the index lists. It removes the other archives of the folder and prints the digest,
// which MetricsChartSHA256 must say; TestBundledMetricsChartIsPinned fails until both agree.
// Renovate proposes a new version of the constant (its comment), not the archive: run this on its
// pull request and read the chart's changes before it merges.
//
//	make metrics-bundle   # go run ./hack/metricsbundle
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	repo "helm.sh/helm/v4/pkg/repo/v1"
	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/helm"
)

func main() {
	dir := flag.String("dir", "internal/helm/bundle", "the folder the archive goes to")
	version := flag.String("version", helm.MetricsChartVersion, "the chart version")
	flag.Parse()
	if err := run(*dir, *version); err != nil {
		fmt.Fprintln(os.Stderr, "metricsbundle:", err)
		os.Exit(1)
	}
}

// Bounds of the downloads.
const (
	maxIndex   = 64 << 20
	maxArchive = 8 << 20
)

func run(dir, version string) error {
	client := &http.Client{Timeout: 2 * time.Minute}
	index, err := get(client, strings.TrimSuffix(helm.MetricsChartRepository, "/")+"/index.yaml", maxIndex)
	if err != nil {
		return err
	}
	var idx repo.IndexFile
	if err := yaml.Unmarshal(index, &idx); err != nil {
		return fmt.Errorf("the index: %w", err)
	}
	var entry *repo.ChartVersion
	for _, v := range idx.Entries[helm.MetricsChartName] {
		if v != nil && v.Metadata != nil && v.Version == version {
			entry = v
		}
	}
	if entry == nil || len(entry.URLs) == 0 || entry.Digest == "" {
		return fmt.Errorf("the index lists no %s %s with a URL and a digest", helm.MetricsChartName, version)
	}
	u, err := url.Parse(entry.URLs[0])
	if err != nil || u.Scheme != "https" {
		return fmt.Errorf("the archive's URL %q is not https", entry.URLs[0])
	}
	data, err := get(client, u.String(), maxArchive)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if digest != strings.TrimPrefix(entry.Digest, "sha256:") {
		return fmt.Errorf("the archive's SHA-256 %s is not the index's %s", digest, entry.Digest)
	}
	old, err := filepath.Glob(filepath.Join(dir, "*.tgz"))
	if err != nil {
		return err
	}
	name := filepath.Join(dir, helm.MetricsChartName+"-"+version+".tgz")
	if err := os.WriteFile(name, data, 0o644); err != nil { //nolint:gosec // a chart archive in the repository, readable by all
		return err
	}
	for _, f := range old {
		if f != name {
			if err := os.Remove(f); err != nil {
				return err
			}
		}
	}
	fmt.Printf("%s: %d bytes, SHA-256 %s (created %s)\n", name, len(data), digest, entry.Created.Format(time.DateOnly))
	if digest != helm.MetricsChartSHA256 || version != helm.MetricsChartVersion {
		fmt.Printf("set MetricsChartVersion = %q and MetricsChartSHA256 = %q in internal/helm/metricsstack.go\n", version, digest)
	}
	return nil
}

// get downloads a URL, at most limit bytes.
func get(client *http.Client, rawURL string, limit int64) ([]byte, error) {
	resp, err := client.Get(rawURL) //nolint:noctx // a command-line tool with the client's timeout
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %s", rawURL, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New(rawURL + ": larger than expected")
	}
	return data, nil
}
