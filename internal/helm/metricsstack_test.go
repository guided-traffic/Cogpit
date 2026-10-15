package helm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"slices"
	"strings"
	"testing"

	rspb "helm.sh/helm/v4/pkg/release/v1"
)

// The embedded archive is the pinned version with the pinned digest, and the only one: a version
// bumped without the archive (Renovate's pull request) fails here until `make metrics-bundle` puts
// the new archive in.
func TestBundledMetricsChartIsPinned(t *testing.T) {
	names, err := fs.Glob(bundleFS, "bundle/*.tgz")
	if err != nil {
		t.Fatal(err)
	}
	want := "bundle/" + MetricsChartName + "-" + MetricsChartVersion + ".tgz"
	if !slices.Equal(names, []string{want}) {
		t.Fatalf("bundled archives %v, want %s alone: run make metrics-bundle", names, want)
	}
	data, err := bundleFS.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != MetricsChartSHA256 {
		t.Fatalf("the archive's SHA-256 is %x, MetricsChartSHA256 says %s", sum, MetricsChartSHA256)
	}
	prom, err := BundledMetricsChart(MetricsPrometheus)
	if err != nil {
		t.Fatal(err)
	}
	if prom.Name != MetricsChartName || prom.Version != MetricsChartVersion || !prom.Verified || prom.Digest != MetricsChartSHA256 ||
		prom.Repository != MetricsBundleRepository {
		t.Errorf("Prometheus's chart %+v", prom)
	}
	// The subcharts are the versions the chart's Chart.lock pins.
	locked := map[string]string{}
	for _, d := range prom.chart.Lock.Dependencies {
		locked[d.Name] = d.Version
	}
	for c, name := range map[MetricsComponent]string{MetricsKubeStateMetrics: "kube-state-metrics", MetricsNodeExporter: "prometheus-node-exporter"} {
		ch, err := BundledMetricsChart(c)
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		if ch.Name != name || ch.Version != locked[name] || ch.Version == "" || !ch.Verified {
			t.Errorf("%s: chart %s %s, the lock pins %s", c, ch.Name, ch.Version, locked[name])
		}
		if err := ch.checkInstallable(); err != nil {
			t.Errorf("%s: %v", c, err)
		}
	}
	if err := prom.checkInstallable(); err != nil {
		t.Error(err)
	}
}

// Each component renders offline (helm template) with Cogpit's values into what the confirmation
// names: its objects in cogpit-metrics and the cluster-scoped ones. Prometheus alone has neither
// Alertmanager, the Pushgateway nor its own kube-state-metrics and node-exporter; its history goes
// to a claim of the default StorageClass or to an emptyDir; kube-state-metrics may not list
// Secrets; node-exporter runs on the host's network.
func TestBundledMetricsChartsRender(t *testing.T) {
	a, _ := installFixture(t)
	render := func(c MetricsComponent, storage MetricsStorage) InstallPreview {
		t.Helper()
		ch, err := BundledMetricsChart(c)
		if err != nil {
			t.Fatal(err)
		}
		p, err := a.PreviewInstall(context.Background(), InstallRequest{Namespace: MetricsNamespace, Name: c.Release(), Chart: ch,
			Values: MetricsValues(c, storage), CreateNamespace: true, Offline: true})
		if err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		return p
	}
	kinds := func(p InstallPreview) []string {
		var out []string
		for _, o := range p.Objects {
			ns := o.Namespace
			if ns == "" {
				ns = "-"
			}
			out = append(out, o.Kind+" "+ns+"/"+o.Name)
		}
		slices.Sort(out)
		return out
	}

	prom := kinds(render(MetricsPrometheus, MetricsStorage{StorageClass: "standard"}))
	for _, want := range []string{
		"ClusterRole -/cogpit-prometheus-server", "ClusterRoleBinding -/cogpit-prometheus-server",
		"Service cogpit-metrics/cogpit-prometheus-server", "Deployment cogpit-metrics/cogpit-prometheus-server",
		"PersistentVolumeClaim cogpit-metrics/cogpit-prometheus-server", "ConfigMap cogpit-metrics/cogpit-prometheus-server",
		"ServiceAccount cogpit-metrics/cogpit-prometheus-server",
	} {
		if !slices.Contains(prom, want) {
			t.Errorf("Prometheus renders no %s: %v", want, prom)
		}
	}
	for _, o := range prom {
		if strings.Contains(o, "alertmanager") || strings.Contains(o, "pushgateway") || strings.Contains(o, "node-exporter") || strings.Contains(o, "kube-state") {
			t.Errorf("Prometheus renders %s", o)
		}
	}
	if emptyDir := kinds(render(MetricsPrometheus, MetricsStorage{})); slices.ContainsFunc(emptyDir, func(o string) bool { return strings.HasPrefix(o, "PersistentVolumeClaim") }) {
		t.Errorf("without a default StorageClass a claim is rendered: %v", emptyDir)
	}

	ksm := render(MetricsKubeStateMetrics, MetricsStorage{})
	if objs := kinds(ksm); !slices.Contains(objs, "ClusterRole -/cogpit-kube-state-metrics") || !slices.Contains(objs, "Deployment cogpit-metrics/cogpit-kube-state-metrics") {
		t.Errorf("kube-state-metrics: %v", objs)
	}
	ne := kinds(render(MetricsNodeExporter, MetricsStorage{}))
	if !slices.Equal(ne, []string{"DaemonSet cogpit-metrics/cogpit-node-exporter", "Service cogpit-metrics/cogpit-node-exporter", "ServiceAccount cogpit-metrics/cogpit-node-exporter"}) {
		t.Errorf("node-exporter: %v", ne)
	}
	// What the objects hold, from the manifests an install stores: kube-state-metrics's role
	// without Secrets, node-exporter on the host's network and process namespace.
	for c, check := range map[MetricsComponent]func(manifest string) bool{
		MetricsKubeStateMetrics: func(m string) bool {
			return strings.Contains(m, "kind: ClusterRole") && !strings.Contains(m, "- secrets")
		},
		MetricsNodeExporter: func(m string) bool {
			return strings.Contains(m, "hostNetwork: true") && strings.Contains(m, "hostPID: true")
		},
		MetricsPrometheus: func(m string) bool { return strings.Contains(m, "--storage.tsdb.retention.time=30d") },
	} {
		ia, store := installFixture(t)
		ch, err := BundledMetricsChart(c)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ia.Install(context.Background(), InstallRequest{Namespace: MetricsNamespace, Name: c.Release(), Chart: ch, Values: MetricsValues(c, MetricsStorage{})}); err != nil {
			t.Fatalf("%s: %v", c, err)
		}
		last, err := store.Last(c.Release())
		if err != nil {
			t.Fatal(err)
		}
		rel, ok := last.(*rspb.Release)
		if !ok || !check(rel.Manifest) {
			t.Errorf("%s's manifest (%T):\n%v", c, last, last)
		}
	}
	t.Logf("Prometheus: %v", prom)
	t.Logf("kube-state-metrics: %v", kinds(ksm))
	t.Logf("node-exporter: %v", ne)
}

// The values Cogpit installs with.
func TestMetricsValues(t *testing.T) {
	v := MetricsValues(MetricsPrometheus, MetricsStorage{})
	server := v["server"].(map[string]any)
	if server["persistentVolume"].(map[string]any)["enabled"] != false || server["emptyDir"] == nil || server["retention"] != "30d" {
		t.Errorf("emptyDir: %+v", server)
	}
	server = MetricsValues(MetricsPrometheus, MetricsStorage{StorageClass: "fast"})["server"].(map[string]any)
	if server["persistentVolume"].(map[string]any)["enabled"] != true || server["retentionSize"] != "6GB" {
		t.Errorf("a volume: %+v", server)
	}
	for _, sub := range []string{"alertmanager", "prometheus-pushgateway", "kube-state-metrics", "prometheus-node-exporter"} {
		if v[sub].(map[string]any)["enabled"] != false {
			t.Errorf("%s is on in Prometheus's release", sub)
		}
	}
	if ex := MetricsValues(MetricsKubeStateMetrics, MetricsStorage{})["collectorsExclude"]; !slices.Contains(ex.([]any), any("secrets")) {
		t.Errorf("kube-state-metrics excludes %v", ex)
	}
	for _, c := range MetricsComponents {
		if got, ok := MetricsComponentOfRelease(c.Release()); !ok || got != c {
			t.Errorf("%s: release %s reads back as %q", c, c.Release(), got)
		}
	}
}
