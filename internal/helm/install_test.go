package helm

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"helm.sh/helm/v4/pkg/action"
	chartcommon "helm.sh/helm/v4/pkg/chart/common"
	chartutil "helm.sh/helm/v4/pkg/chart/v2/util"
	kubefake "helm.sh/helm/v4/pkg/kube/fake"
	"helm.sh/helm/v4/pkg/release/common"
	"helm.sh/helm/v4/pkg/storage"
	"helm.sh/helm/v4/pkg/storage/driver"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// installChart packages a chart with the SDK and loads it as an install takes it: a ConfigMap
// and a Secret from the values, a hook, with crds a custom resource definition in crds/, and a
// template that fails on .Values.boom quoting it.
func installChart(t *testing.T, crds bool) *Chart {
	t.Helper()
	ch := testkit.BuildChart(t, testkit.TestChart{
		Name: "web", Version: "0.2.0", AppVersion: "1.2",
		Values: "greeting: hello\nimage: registry.example/web:1.2\npassword: \"\"\nboom: false\n",
		Templates: map[string]string{
			"templates/cm.yaml": "{{ if .Values.boom }}{{ fail (printf \"cannot use %s\" .Values.boom) }}{{ end }}" +
				"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}\ndata:\n  greeting: {{ .Values.greeting | quote }}\n",
			"templates/secret.yaml": "apiVersion: v1\nkind: Secret\nmetadata:\n  name: {{ .Release.Name }}\nstringData:\n  password: {{ .Values.password | quote }}\n",
			"templates/hook.yaml": "apiVersion: v1\nkind: Pod\nmetadata:\n  name: {{ .Release.Name }}-test\n  annotations:\n    helm.sh/hook: test\n" +
				"spec:\n  containers: [{name: t, image: busybox}]\n",
		},
	})
	if crds {
		ch.Files = append(ch.Files, &chartcommon.File{Name: "crds/widgets.yaml", Data: []byte(
			"apiVersion: apiextensions.k8s.io/v1\nkind: CustomResourceDefinition\nmetadata:\n  name: widgets.example.com\n")})
	}
	path, err := chartutil.Save(ch, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	c, err := LoadChartArchive(data)
	if err != nil {
		t.Fatal(err)
	}
	c.Repository = "test"
	return c
}

// installFixture is the Helm SDK on an in-memory release store and a kube client that only prints.
func installFixture(t *testing.T) (*Actions, *storage.Storage) {
	t.Helper()
	store := storage.Init(driver.NewMemory())
	cfg := &action.Configuration{Releases: store, KubeClient: &kubefake.PrintingKubeClient{Out: io.Discard},
		Capabilities: chartcommon.DefaultCapabilities}
	return &Actions{config: func(string, Driver, *slog.Logger) (*action.Configuration, error) { return cfg, nil }}, store
}

func values(t *testing.T, text string) map[string]any {
	t.Helper()
	v, err := ParseValues(text)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// The dry run names what the chart renders (objects, hooks, the CRDs of crds/) and stores nothing;
// the install stores revision 1 deployed with the values; a second install of the name is refused
// before anything changes.
func TestInstallThroughTheSDK(t *testing.T) {
	a, store := installFixture(t)
	ch := installChart(t, false)
	req := InstallRequest{Namespace: "apps", Name: "web", Chart: ch, Values: values(t, ch.DefaultValues()+"password: s3cr3t-pw\n")}
	withCRDs := req
	withCRDs.Chart = installChart(t, true)
	p, err := a.PreviewInstall(context.Background(), withCRDs)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Objects) != 2 || p.Objects[0].Kind != "ConfigMap" || p.Objects[0].Name != "web" || p.Objects[0].Namespace != "apps" ||
		p.Objects[1].Kind != "Secret" || len(p.Hooks) != 1 || p.Hooks[0].Name != "web-test" ||
		len(p.CRDs) != 1 || p.CRDs[0].Name != "widgets.example.com" || p.CRDs[0].Namespace != "" {
		t.Errorf("preview %+v", p)
	}
	if h, err := store.History("web"); err == nil && len(h) > 0 {
		t.Fatalf("the dry run stored %d revisions", len(h))
	}
	offline, offlineStore := installFixture(t)
	req.Offline = true
	if p, err := offline.PreviewInstall(context.Background(), req); err != nil || len(p.Objects) != 2 {
		t.Errorf("an offline preview: %+v %v", p, err)
	}
	req.Offline = false
	if h, err := offlineStore.History("web"); err == nil && len(h) > 0 {
		t.Fatalf("the offline dry run stored %d revisions", len(h))
	}

	rev, err := a.Install(context.Background(), req)
	if err != nil || rev != 1 {
		t.Fatalf("install: %d %v", rev, err)
	}
	rls := last(t, store)
	if rls.Version != 1 || rls.Info.Status != common.StatusDeployed || rls.Config["password"] != "s3cr3t-pw" || rls.Chart.Metadata.Version != "0.2.0" {
		t.Errorf("release %d %s %v", rls.Version, rls.Info.Status, rls.Config)
	}
	if _, err := a.Install(context.Background(), req); !errors.Is(err, ErrUnchanged) || !strings.Contains(err.Error(), "exists in namespace apps") {
		t.Errorf("a second install: %v", err)
	}
}

// What an install refuses before it stores a release is ErrUnchanged, and an error never quotes a
// value the user typed; the chart's own defaults may appear.
func TestInstallRefusalsAndScrubbing(t *testing.T) {
	a, store := installFixture(t)
	ch := installChart(t, false)
	_, err := a.Install(context.Background(), InstallRequest{Namespace: "apps", Name: "web", Chart: ch, Values: values(t, "boom: s3cr3t-boom\n")})
	if !errors.Is(err, ErrUnchanged) || strings.Contains(err.Error(), "s3cr3t-boom") || !strings.Contains(err.Error(), "cannot use") {
		t.Errorf("a template that fails: %v", err)
	}
	if _, err := a.PreviewInstall(context.Background(), InstallRequest{Namespace: "apps", Name: "web", Chart: ch, Values: values(t, "boom: s3cr3t-boom\n")}); err == nil ||
		strings.Contains(err.Error(), "s3cr3t-boom") {
		t.Errorf("a preview that fails: %v", err)
	}
	if h, err := store.History("web"); err == nil && len(h) > 0 {
		t.Errorf("a refused install stored %d revisions", len(h))
	}
	for _, c := range []struct {
		req  InstallRequest
		want string
	}{
		{InstallRequest{Namespace: "apps", Name: "Web_1", Chart: ch}, "release name"},
		{InstallRequest{Namespace: "Apps", Name: "web", Chart: ch}, "namespace"},
		{InstallRequest{Namespace: "apps", Name: strings.Repeat("a", 54), Chart: ch}, "release name"},
	} {
		if _, err := a.Install(context.Background(), c.req); !errors.Is(err, ErrUnchanged) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: %v", c.req, err)
		}
	}
	if got := valuesDiff(values(t, "greeting: hello\nimage: other\nnested: {a: 1, b: 2}\n"), values(t, "greeting: hello\nimage: x\nnested: {a: 1}\n")); len(got) != 2 ||
		got["image"] != "other" || got["nested"].(map[string]any)["b"] != float64(2) {
		t.Errorf("the user's values: %v", got)
	}
}
