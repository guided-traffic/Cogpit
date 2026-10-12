package helm

import (
	"context"
	"errors"
	"strings"
	"testing"

	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// upChart is the repository's chart web at a version: a ConfigMap from .Values.greeting, and from
// 0.2.0 on a second ConfigMap, so that an upgrade to it adds an object.
func upChart(v string) testkit.TestChart {
	c := testkit.TestChart{
		Name: "web", Version: v, AppVersion: "1." + v,
		Values: "greeting: hello\n",
		Templates: map[string]string{"templates/cm.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}\n" +
			"data:\n  greeting: {{ .Values.greeting | quote }}\n  version: " + v + "\n"},
	}
	if v != "0.1.0" {
		c.Templates["templates/extra.yaml"] = "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}-extra\n"
	}
	return c
}

// fetchUp downloads a version of web from the repository, checked against its index's digest.
func fetchUp(t *testing.T, r *testkit.HelmRepo, version string) *Chart {
	t.Helper()
	f := &Fetcher{Client: r.Client()}
	dir := t.TempDir()
	idx, err := f.DownloadIndex(context.Background(), r.URL, dir+"/index.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, ok := idx.Chart("web")
	if !ok {
		t.Fatal("no chart web")
	}
	v, ok := c.Version(version)
	if !ok {
		t.Fatalf("no version %s", version)
	}
	ch, err := f.FetchChart(context.Background(), "test", r.URL, "web", v)
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

// An upgrade to a chart downloaded from a repository: web 0.1.0, installed from the repository,
// goes to 0.2.0 with new values; the revision holds the new chart, its manifest the object the new
// version adds; the previous revision is superseded. Back to 0.1.0 (a downgrade) works the same
// way. A chart of another name, a library chart and a release that moved are refused before
// anything changes.
func TestUpgradeToARepositoryChart(t *testing.T) {
	r := testkit.NewHelmRepo(t, upChart("0.1.0"), upChart("0.2.0"))
	a, store := installFixture(t)
	v1, v2 := fetchUp(t, r, "0.1.0"), fetchUp(t, r, "0.2.0")
	if !v2.Verified || v2.Version != "0.2.0" {
		t.Fatalf("downloaded %+v", v2)
	}
	if _, err := a.Install(context.Background(), InstallRequest{Namespace: "apps", Name: "web", Chart: v1, Values: map[string]any{"greeting": "one"}}); err != nil {
		t.Fatal(err)
	}
	rev, err := a.UpgradeChart(context.Background(), "apps", "web", v2, 1, map[string]any{"greeting": "two"})
	if err != nil || rev != 2 {
		t.Fatalf("upgrade to 0.2.0: %d %v", rev, err)
	}
	rls := last(t, store)
	if rls.Version != 2 || rls.Info.Status != common.StatusDeployed || rls.Chart.Metadata.Version != "0.2.0" || rls.Config["greeting"] != "two" ||
		!strings.Contains(rls.Manifest, "name: web-extra") || !strings.Contains(rls.Manifest, "version: 0.2.0") {
		t.Errorf("revision %d %s chart %s config %v manifest %q", rls.Version, rls.Info.Status, rls.Chart.Metadata.Version, rls.Config, rls.Manifest)
	}
	if prev, err := store.Get("web", 1); err != nil || prev.(*rspb.Release).Info.Status != common.StatusSuperseded {
		t.Errorf("revision 1 after the upgrade: %v", err)
	}
	// The release's own chart is now 0.2.0: Upgrade without a download takes it, and refuses 0.1.0.
	if _, err := a.Upgrade(context.Background(), "apps", "web", "0.1.0", 2, nil); !errors.Is(err, ErrNoRepository) || !errors.Is(err, ErrUnchanged) {
		t.Errorf("0.1.0 without a download: %v", err)
	}
	if rev, err := a.UpgradeChart(context.Background(), "apps", "web", v1, 2, map[string]any{"greeting": "back"}); err != nil || rev != 3 ||
		last(t, store).Chart.Metadata.Version != "0.1.0" || strings.Contains(last(t, store).Manifest, "web-extra") {
		t.Errorf("downgrade to 0.1.0: %d %v", rev, err)
	}

	other := testkit.NewHelmRepo(t, testkit.TestChart{Name: "api", Version: "0.2.0", Templates: map[string]string{
		"templates/cm.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: api\n"}})
	api := fetchOther(t, other, "api", "0.2.0")
	lib := testkit.NewHelmRepo(t, testkit.TestChart{Name: "web", Version: "0.3.0", Type: "library"})
	library := fetchOther(t, lib, "web", "0.3.0")
	for _, c := range []struct {
		ch   *Chart
		from int
		want string
	}{
		{api, 3, "not the release's chart"},
		{library, 3, "library chart"},
		{v2, 2, "the release moved"},
		{nil, 3, "no chart"},
	} {
		_, err := a.UpgradeChart(context.Background(), "apps", "web", c.ch, c.from, map[string]any{"greeting": "s3cr3t-greeting"})
		if !errors.Is(err, ErrUnchanged) || !strings.Contains(err.Error(), c.want) || strings.Contains(err.Error(), "s3cr3t") {
			t.Errorf("%s: %v", c.want, err)
		}
	}
	if rls := last(t, store); rls.Version != 3 {
		t.Errorf("a refused upgrade made revision %d", rls.Version)
	}
}

// fetchOther downloads a chart version from a repository, by name.
func fetchOther(t *testing.T, r *testkit.HelmRepo, name, version string) *Chart {
	t.Helper()
	f := &Fetcher{Client: r.Client()}
	idx, err := f.DownloadIndex(context.Background(), r.URL, t.TempDir()+"/index.yaml")
	if err != nil {
		t.Fatal(err)
	}
	c, _ := idx.Chart(name)
	v, ok := c.Version(version)
	if !ok {
		t.Fatalf("no %s %s", name, version)
	}
	ch, err := f.FetchChart(context.Background(), "test", r.URL, name, v)
	if err != nil {
		t.Fatal(err)
	}
	return ch
}

// A template that fails quoting a value fails the upgrade to a repository chart unchanged, the
// value scrubbed, as for the release's own chart.
func TestUpgradeChartScrubsValues(t *testing.T) {
	failing := upChart("0.2.0")
	failing.Templates["templates/fail.yaml"] = "{{ if .Values.boom }}{{ fail (printf \"cannot use %s\" .Values.boom) }}{{ end }}"
	r := testkit.NewHelmRepo(t, upChart("0.1.0"), failing)
	a, store := installFixture(t)
	if _, err := a.Install(context.Background(), InstallRequest{Namespace: "apps", Name: "web", Chart: fetchUp(t, r, "0.1.0")}); err != nil {
		t.Fatal(err)
	}
	_, err := a.UpgradeChart(context.Background(), "apps", "web", fetchUp(t, r, "0.2.0"), 1, map[string]any{"boom": "s3cr3t-token"})
	if !errors.Is(err, ErrUnchanged) || strings.Contains(err.Error(), "s3cr3t-token") || !strings.Contains(err.Error(), "cannot use "+withheldValue) {
		t.Errorf("error %v", err)
	}
	if rls := last(t, store); rls.Version != 1 || rls.Chart.Metadata.Version != "0.1.0" {
		t.Errorf("a failed render changed the release: revision %d chart %s", rls.Version, rls.Chart.Metadata.Version)
	}
}

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
		ok   bool
	}{
		{"0.2.0", "0.1.0", 1, true},
		{"0.1.0", "0.2.0", -1, true},
		{"1.0.0", "v1.0.0", 0, true},
		{"1.0.0-rc.1", "1.0.0", -1, true},
		{"1.0.0-rc.2", "1.0.0-rc.1", 1, true},
		{"10.0.0", "9.9.9", 1, true},
		{"1.2", "1.1.9", 1, true},
		{"latest", "1.0.0", 0, false},
	} {
		got, ok := CompareVersions(c.a, c.b)
		if got != c.want || ok != c.ok {
			t.Errorf("CompareVersions(%q, %q) = %d %v, want %d %v", c.a, c.b, got, ok, c.want, c.ok)
		}
	}
}
