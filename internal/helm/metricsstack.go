package helm

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"helm.sh/helm/v4/pkg/chart/loader/archive"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
)

// Cogpit Metrics (Cluster Settings › Cogpit Metrics, D-61): the metrics stack Cogpit installs into a
// cluster when the user asks for it, Cogpit's counterpart of Lens's "Lens Metrics" page (three
// switches, Prometheus, kube-state-metrics and node-exporter, and an Apply button; captured). The
// charts are prometheus-community's `prometheus` chart at a pinned version, embedded in the binary
// with its SHA-256 checked before every use: the chart itself for Prometheus (its subcharts off),
// and its kube-state-metrics and prometheus-node-exporter subcharts, the versions its Chart.lock
// pins, as charts of their own, so that each switch is one release. Nothing is downloaded: the
// chart is what the binary was built with, and the images come from the registries the charts
// name (quay.io, registry.k8s.io), pulled by the cluster.
//
// Updating the bundle: change MetricsChartVersion and MetricsChartSHA256 to the index's version and
// digest (`make metrics-bundle` downloads the archive from the repository and checks its digest
// against the index), and TestBundledMetricsChartIsPinned checks that the embedded archive is
// that version with that digest.

// The pinned chart: prometheus-community/prometheus.
const (
	// MetricsChartRepository is where the chart comes from (the index the version and digest are
	// read from when the bundle is updated).
	MetricsChartRepository = "https://prometheus-community.github.io/helm-charts"
	// MetricsChartName is the chart's name in the repository.
	MetricsChartName = "prometheus"
	// MetricsChartVersion is the bundled chart version.
	// renovate: datasource=helm depName=prometheus registryUrl=https://prometheus-community.github.io/helm-charts
	MetricsChartVersion = "29.37.0"
	// MetricsChartSHA256 is the archive's digest as the repository's index lists it.
	MetricsChartSHA256 = "8492d3a0e99f99a12bb389f1c83f3636e3fc5d68f995a09e815044773246217e"
	// MetricsBundleRepository is the repository name a bundled chart carries (Chart.Repository),
	// which the Helm journal records as <repository>/<chart>.
	MetricsBundleRepository = "cogpit-bundle"
)

// MetricsNamespace is the namespace Cogpit installs its metrics stack into: Cogpit's own, never
// Lens's lens-metrics (D-12).
const MetricsNamespace = "cogpit-metrics"

//go:embed bundle/*.tgz
var bundleFS embed.FS

// MetricsComponent is one switch of the page: one release.
type MetricsComponent string

// The components, in the page's order (Lens's: Prometheus, kube-state-metrics, node-exporter).
const (
	MetricsPrometheus       MetricsComponent = "prometheus"
	MetricsKubeStateMetrics MetricsComponent = "kube-state-metrics"
	MetricsNodeExporter     MetricsComponent = "node-exporter"
)

// MetricsComponents are the components in the page's order.
var MetricsComponents = []MetricsComponent{MetricsPrometheus, MetricsKubeStateMetrics, MetricsNodeExporter}

// Release is the component's release name in MetricsNamespace: cogpit-prometheus,
// cogpit-kube-state-metrics, cogpit-node-exporter. Prometheus's release makes its Service
// cogpit-prometheus-server (the chart's name for the server of a release that contains its
// name), which detection ranks first (metrics.CogpitPrometheusService).
func (c MetricsComponent) Release() string { return "cogpit-" + string(c) }

// Label is the component's name on the page.
func (c MetricsComponent) Label() string {
	if c == MetricsPrometheus {
		return "Prometheus"
	}
	return string(c)
}

// chartPath is where the component's chart lies in the bundled archive: the archive's root for
// Prometheus, else the subchart's folder.
func (c MetricsComponent) chartPath() string {
	switch c {
	case MetricsKubeStateMetrics:
		return "charts/kube-state-metrics/"
	case MetricsNodeExporter:
		return "charts/prometheus-node-exporter/"
	}
	return ""
}

// MetricsComponentOf reads a component's name.
func MetricsComponentOf(name string) (MetricsComponent, bool) {
	for _, c := range MetricsComponents {
		if string(c) == name {
			return c, true
		}
	}
	return "", false
}

// MetricsComponentOfRelease returns the component a release of MetricsNamespace is, by its name.
func MetricsComponentOfRelease(release string) (MetricsComponent, bool) {
	for _, c := range MetricsComponents {
		if c.Release() == release {
			return c, true
		}
	}
	return "", false
}

// ErrBundleDigest refuses a bundled archive whose digest is not the pinned one: a build that
// embedded another file than MetricsChartSHA256 names.
var ErrBundleDigest = errors.New("the bundled metrics chart does not match its pinned digest")

// bundledArchive returns the embedded archive of the pinned version, its digest checked.
func bundledArchive() ([]byte, error) {
	data, err := bundleFS.ReadFile("bundle/" + MetricsChartName + "-" + MetricsChartVersion + ".tgz")
	if err != nil {
		return nil, fmt.Errorf("the bundled metrics chart %s %s is not in this build: %w", MetricsChartName, MetricsChartVersion, err)
	}
	sum := sha256.Sum256(data)
	if hex.EncodeToString(sum[:]) != MetricsChartSHA256 {
		return nil, ErrBundleDigest
	}
	return data, nil
}

// BundledMetricsChart returns a component's chart from the embedded archive, loaded anew for each
// call (the Helm SDK changes a chart while it installs it: it drops the subcharts that are off).
// Digest is the archive's, which matched the pin (Verified).
func BundledMetricsChart(c MetricsComponent) (*Chart, error) {
	data, err := bundledArchive()
	if err != nil {
		return nil, err
	}
	var ch *Chart
	if c.chartPath() == "" {
		ch, err = LoadChartArchive(data)
	} else {
		ch, err = loadSubchart(data, c.chartPath())
	}
	if err != nil {
		return nil, fmt.Errorf("the bundled chart of %s: %w", c.Label(), err)
	}
	ch.Repository, ch.Digest, ch.Verified, ch.Size = MetricsBundleRepository, MetricsChartSHA256, true, len(data)
	return ch, nil
}

// loadSubchart loads the chart in a folder of a chart archive (charts/<name>/) as a chart of its
// own: its files with the folder's path cut, loaded as an archive's are (the SDK's path checks and
// size bound applied when the archive was read).
func loadSubchart(data []byte, dir string) (c *Chart, err error) {
	defer func() {
		if r := recover(); r != nil {
			c, err = nil, errors.New("the chart archive could not be read")
		}
	}()
	files, err := archive.LoadArchiveFiles(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var sub []*archive.BufferedFile
	for _, f := range files {
		if name, ok := strings.CutPrefix(f.Name, dir); ok && name != "" {
			sub = append(sub, &archive.BufferedFile{Name: name, ModTime: f.ModTime, Data: f.Data})
		}
	}
	if len(sub) == 0 {
		return nil, fmt.Errorf("the archive has no chart in %s", dir)
	}
	ch, err := loader.LoadFiles(sub)
	if err != nil {
		return nil, err
	}
	if ch.Metadata == nil {
		return nil, fmt.Errorf("the chart in %s has no Chart.yaml", dir)
	}
	return &Chart{chart: ch, Name: ch.Metadata.Name, Version: ch.Metadata.Version, AppVersion: ch.Metadata.AppVersion,
		Description: clip(ch.Metadata.Description)}, nil
}

// MetricsStorage is where the bundled Prometheus keeps its history: a volume of the cluster's
// default StorageClass when it has one, else the pod's emptyDir (lost when the pod goes).
type MetricsStorage struct {
	// StorageClass is the default StorageClass the volume is claimed from; "" for emptyDir.
	StorageClass string `json:"storageClass,omitempty"`
	// ReclaimPolicy is that class's: Delete (the API server's default) removes the volume with the
	// claim when the release goes; Retain keeps it, a PersistentVolume Cogpit then leaves behind.
	ReclaimPolicy string `json:"reclaimPolicy,omitempty"`
}

// Persistent reports whether the history is kept on a volume.
func (s MetricsStorage) Persistent() bool { return s.StorageClass != "" }

// The bundled Prometheus's bounds (Cogpit's defaults, the lead's for the owner to confirm, D-61):
// 30 days, the longest range Cogpit's charts offer, bounded by size so that the volume (the
// chart's default claim of 8 GiB) or the node's disk under an emptyDir does not fill.
const (
	metricsRetention          = "30d"
	metricsVolumeSize         = "8Gi"
	metricsVolumeRetention    = "6GB"
	metricsEmptyDirSize       = "2Gi"
	metricsEmptyDirRetention  = "1500MB"
	metricsKSMExcludedSecrets = "secrets"
	// valueEnabled is the charts' switch of a part (a subchart, the volume).
	valueEnabled = "enabled"
)

// Text of the storage for the page and the confirmation.
func (s MetricsStorage) String() string {
	if s.Persistent() {
		out := "a volume of " + metricsVolumeSize + " from StorageClass " + s.StorageClass + " (the cluster's default), " +
			metricsRetention + " or " + metricsVolumeRetention + " of history"
		if s.ReclaimPolicy == "Retain" {
			out += "; the class retains its volumes, so the PersistentVolume stays when Prometheus is removed"
		}
		return out
	}
	return "the pod's emptyDir of at most " + metricsEmptyDirSize + " (the cluster has no default StorageClass), lost when " +
		"the pod is replaced; " + metricsRetention + " or " + metricsEmptyDirRetention + " of history"
}

// MetricsValues are a component's values (the release's user-supplied values):
//
//   - Prometheus: the chart's server alone (Alertmanager, the Pushgateway and its own
//     kube-state-metrics and node-exporter off: those are switches of their own), its history on
//     storage, retention bounded by time and size. The chart's scrape jobs find kube-state-metrics
//     and node-exporter by their services' prometheus.io/scrape annotation, which their charts set.
//   - kube-state-metrics: the chart's defaults without its Secrets collector, so that its
//     ServiceAccount may not list the cluster's Secrets (the collector reads only metadata, but list
//     returns the data; Cogpit reads none of its series).
//   - node-exporter: the values the prometheus chart passes to it (no PodSecurityPolicy, no
//     privilege escalation).
func MetricsValues(c MetricsComponent, storage MetricsStorage) map[string]any {
	switch c {
	case MetricsKubeStateMetrics:
		return map[string]any{"collectorsExclude": []any{metricsKSMExcludedSecrets}}
	case MetricsNodeExporter:
		return map[string]any{
			"fullnameOverride":         c.Release(), // its objects named as the release, not <release>-prometheus-node-exporter
			"rbac":                     map[string]any{"pspEnabled": false},
			"containerSecurityContext": map[string]any{"allowPrivilegeEscalation": false},
		}
	}
	server := map[string]any{"retention": metricsRetention}
	if storage.Persistent() {
		server["persistentVolume"] = map[string]any{valueEnabled: true, "size": metricsVolumeSize}
		server["retentionSize"] = metricsVolumeRetention
	} else {
		server["persistentVolume"] = map[string]any{valueEnabled: false}
		server["emptyDir"] = map[string]any{"sizeLimit": metricsEmptyDirSize}
		server["retentionSize"] = metricsEmptyDirRetention
	}
	return map[string]any{
		"alertmanager":             map[string]any{valueEnabled: false},
		"prometheus-pushgateway":   map[string]any{valueEnabled: false},
		"kube-state-metrics":       map[string]any{valueEnabled: false},
		"prometheus-node-exporter": map[string]any{valueEnabled: false},
		"server":                   server,
	}
}
