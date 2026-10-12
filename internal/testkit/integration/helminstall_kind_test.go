//go:build kind

package integration_test

import (
	"context"
	"maps"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// installNamespace is the Helm install kind test's namespace (agent HI of the night run), which
// the install itself creates (createNamespace) and the test deletes.
const installNamespace = "cogpit-hi-install"

// TestKindHelmInstall installs a chart from a chart repository served by httptest (the SDK runs
// in Cogpit's process, so the cluster never reaches the repository) through Cogpit's engine on the
// kind cluster: the repository added in temporary folders, the Install tab with the chart's
// values, the preview (which writes nothing), the install from the tab creating its namespace,
// with the requests counted; an agent's install with inline values, confirmed; a release name in
// use refused; then both releases uninstalled and the namespace deleted.
func TestKindHelmInstall(t *testing.T) {
	kubeconfig, contextName := kindEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kc := conn.Kube()
	_ = kc.CoreV1().Namespaces().Delete(ctx, installNamespace, metav1.DeleteOptions{})
	if !testkit.WaitFor(ctx, func() bool {
		_, err := kc.CoreV1().Namespaces().Get(ctx, installNamespace, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}) {
		t.Fatalf("namespace %s is still there", installNamespace)
	}
	t.Cleanup(func() { _ = kc.CoreV1().Namespaces().Delete(context.Background(), installNamespace, metav1.DeleteOptions{}) })

	repo := testkit.NewHelmRepo(t, testkit.TestChart{
		Name: "hi-web", Version: "0.1.0", AppVersion: "1.0", Description: "Cogpit's install test chart",
		Values: "# What the ConfigMap greets with.\ngreeting: hello\npassword: \"\"\n",
		Templates: map[string]string{
			"templates/cm.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}-config\ndata:\n  greeting: {{ .Values.greeting | quote }}\n",
			"templates/secret.yaml": "apiVersion: v1\nkind: Secret\nmetadata:\n  name: {{ .Release.Name }}-secret\n" +
				"stringData:\n  password: {{ .Values.password | quote }}\n",
		},
	})
	journal := engine.NewMemoryHelmJournal()
	k := &kindSession{audit: &auditBuffer{}, journal: journal}
	var mu sync.Mutex
	k.s = engine.New(engine.Options{
		Connector:        &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}},
		SnapshotInterval: 20 * time.Millisecond,
		Audit:            k.audit,
		Helm:             engine.NewHelmOps(journal),
		HelmRepos:        engine.NewHelmRepos(t.TempDir(), t.TempDir(), engine.WithChartFetcher(&helm.Fetcher{Client: repo.Client()})),
		Confirmer: engine.ConfirmFunc(func(_ context.Context, req engine.ConfirmRequest) (bool, error) {
			mu.Lock()
			defer mu.Unlock()
			k.asked = append(k.asked, req)
			return true, nil
		}),
	})
	defer k.s.Close()
	if err := commands.RegisterAll(k.s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := k.s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.repo.add", map[string]any{"name": "hi", "url": repo.URL}); err != nil {
		t.Fatal(err)
	}
	chart := engine.HelmChartRef(contextName, "hi", "hi-web")
	if !testkit.WaitFor(ctx, func() bool { _, err := k.s.Lookup(chart); return err == nil }) {
		t.Fatal("the chart is not listed")
	}
	if _, err := k.s.Objects(ctx, kinds.HelmChartsKey, nil); err != nil {
		t.Fatal(err)
	}

	// The GUI's path: the tab, the values edited, the preview, the install.
	res, err := k.dispatch(ctx, engine.OriginGUI, "helm.chart.installTab", map[string]any{"ref": chart})
	if err != nil {
		t.Fatal(err)
	}
	tab := res.(commands.EditResult)
	if tab.Tab.HelmInstall == nil || !tab.Tab.HelmInstall.Verified || !strings.Contains(tab.Text, "What the ConfigMap greets with") {
		t.Fatalf("tab %+v", tab)
	}
	if _, err := k.dispatch(ctx, engine.OriginGUI, "editor.setText", map[string]any{"id": tab.Tab.ID, "yaml": "greeting: kind\npassword: kind-s3cr3t\n"}); err != nil {
		t.Fatal(err)
	}
	params := map[string]any{"tab": tab.Tab.ID, "namespace": installNamespace, "name": "hi-web", "createNamespace": true}
	counter := k.s.Backend().Conn.Counter()
	before := counter.Snapshot()
	res, err = k.dispatch(ctx, engine.OriginGUI, "helm.chart.preview", params)
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	pre := res.(commands.InstallPreview)
	var objs []string
	for _, o := range pre.Objects {
		objs = append(objs, o.Kind+" "+o.Namespace+"/"+o.Name)
	}
	if strings.Join(objs, ", ") != "ConfigMap "+installNamespace+"/hi-web-config, Secret "+installNamespace+"/hi-web-secret" {
		t.Errorf("preview %v", objs)
	}
	previewRequests := delta(before, counter.Snapshot())
	for key := range previewRequests {
		if verb, _, _ := strings.Cut(key, " "); !slices.Contains([]string{"get", "list", "watch", "watchlist", "discovery", "other"}, verb) {
			t.Errorf("the preview sent %s", key)
		}
	}
	t.Logf("requests of the preview: %v", previewRequests)

	params["textRevision"] = kindTabRevision(ctx, t, k, tab.Tab.ID)
	before = counter.Snapshot()
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.chart.install", params); err != nil {
		t.Fatalf("install: %v", err)
	}
	t.Logf("requests of the install (the SDK's, and the cache's meanwhile): %v", delta(before, counter.Snapshot()))
	cm, err := kc.CoreV1().ConfigMaps(installNamespace).Get(ctx, "hi-web-config", metav1.GetOptions{})
	if err != nil || cm.Data["greeting"] != "kind" {
		t.Fatalf("the release's ConfigMap: %v %v", cm, err)
	}
	waitInstalled := func(name string) {
		t.Helper()
		ref := engine.HelmReleaseRef(contextName, installNamespace, name)
		if !testkit.WaitFor(ctx, func() bool {
			obj, err := k.s.Lookup(ref)
			return err == nil && helm.ReleaseRevision(obj) == 1 && helm.ReleaseStatus(obj) == "deployed"
		}) {
			t.Fatalf("release %s never deployed", name)
		}
	}
	waitInstalled("hi-web")
	if list, _ := journal.List(); len(list) != 0 {
		t.Errorf("journal after the install: %+v", list)
	}
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.chart.install", params); err == nil || !strings.Contains(err.Error(), "exists") {
		t.Errorf("a second install of hi-web: %v", err)
	}

	// An agent's install with inline values: confirmed, audited without them.
	if _, err := k.dispatch(ctx, engine.OriginExternalAgent, "helm.chart.install", map[string]any{
		"repository": "hi", "chart": "hi-web", "version": "0.1.0", "values": "greeting: agent\npassword: agent-s3cr3t\n",
		"namespace": installNamespace, "name": "hi-api"}); err != nil {
		t.Fatalf("an agent's install: %v", err)
	}
	waitInstalled("hi-api")
	mu.Lock()
	asked := slices.Clone(k.asked)
	mu.Unlock()
	if len(asked) != 1 || asked[0].Command != "helm.chart.install" || asked[0].Review == "" || !strings.Contains(asked[0].Detail, "hi/hi-web 0.1.0") {
		t.Errorf("confirmations %+v", asked)
	}
	for _, leak := range []string{"kind-s3cr3t", "agent-s3cr3t"} {
		if strings.Contains(k.audit.String(), leak) {
			t.Errorf("%q in the audit log", leak)
		}
	}

	for _, name := range []string{"hi-web", "hi-api"} {
		if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.release.uninstall", map[string]any{"ref": engine.HelmReleaseRef(contextName, installNamespace, name)}); err != nil {
			t.Fatalf("uninstall %s: %v", name, err)
		}
	}
	if _, err := kc.CoreV1().ConfigMaps(installNamespace).Get(ctx, "hi-web-config", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Errorf("the ConfigMap after the uninstall: %v", err)
	}
	if err := kc.CoreV1().Namespaces().Delete(ctx, installNamespace, metav1.DeleteOptions{}); err != nil {
		t.Errorf("deleting the namespace: %v", err)
	}
}

// delta returns the requests counted between two snapshots, by verb and resource.
func delta(before, after cluster.RequestStats) map[string]int64 {
	out := map[string]int64{}
	for _, key := range slices.Sorted(maps.Keys(after.ByVerbResource)) {
		if d := after.ByVerbResource[key] - before.ByVerbResource[key]; d > 0 {
			out[key] = d
		}
	}
	return out
}
