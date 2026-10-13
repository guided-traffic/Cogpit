//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
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
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// driverNamespace is the ConfigMaps driver kind test's namespace (agent HD of the night run), which
// the install creates (createNamespace) and the test deletes.
const driverNamespace = "cogpit-hd-configmap"

// TestKindHelmConfigMapDriver runs a release kept in ConfigMaps (HELM_DRIVER=configmap) through
// Cogpit's engine on the kind cluster: the preference set in the window, an agent's install from
// a chart repository served by httptest (its confirmation saying where the values go), the
// records written as ConfigMaps and no Secret, the release in the Releases view with its storage,
// the release ConfigMap's payload redacted for agents, an upgrade, a rollback and the uninstall
// through the ConfigMaps driver, and the namespace deleted.
func TestKindHelmConfigMapDriver(t *testing.T) {
	kubeconfig, contextName := kindEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kc := conn.Kube()
	_ = kc.CoreV1().Namespaces().Delete(ctx, driverNamespace, metav1.DeleteOptions{})
	if !testkit.WaitFor(ctx, func() bool {
		_, err := kc.CoreV1().Namespaces().Get(ctx, driverNamespace, metav1.GetOptions{})
		return apierrors.IsNotFound(err)
	}) {
		t.Fatalf("namespace %s is still there", driverNamespace)
	}
	t.Cleanup(func() {
		_ = kc.CoreV1().Namespaces().Delete(context.Background(), driverNamespace, metav1.DeleteOptions{})
	})

	repo := testkit.NewHelmRepo(t, testkit.TestChart{
		Name: "hd-web", Version: "0.1.0", AppVersion: "1.0", Description: "Cogpit's ConfigMaps driver test chart",
		Values: "greeting: hello\npassword: \"\"\n",
		Templates: map[string]string{
			"templates/cm.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: {{ .Release.Name }}-config\ndata:\n  greeting: {{ .Values.greeting | quote }}\n",
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
		Preferences:      engine.NewPreferencesStore(t.TempDir()),
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
	if _, err := k.dispatch(ctx, engine.OriginExternalAgent, "preferences.set", map[string]any{"helmDriver": "configmap"}); err == nil {
		t.Error("an agent chose the ConfigMaps driver")
	}
	if _, err := k.dispatch(ctx, engine.OriginGUI, "preferences.set", map[string]any{"helmDriver": "configmap"}); err != nil {
		t.Fatal(err)
	}
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.repo.add", map[string]any{"name": "hd", "url": repo.URL}); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool { _, err := k.s.Lookup(engine.HelmChartRef(contextName, "hd", "hd-web")); return err == nil }) {
		t.Fatal("the chart is not listed")
	}

	// An agent's install, confirmed by the user; the confirmation says the values go to ConfigMaps.
	if _, err := k.dispatch(ctx, engine.OriginExternalAgent, "helm.chart.install", map[string]any{
		"repository": "hd", "chart": "hd-web", "version": "0.1.0", "values": "greeting: kind\npassword: hd-s3cr3t\n",
		"namespace": driverNamespace, "name": "hd-web", "createNamespace": true}); err != nil {
		t.Fatalf("install: %v", err)
	}
	mu.Lock()
	asked := slices.Clone(k.asked)
	mu.Unlock()
	if len(asked) != 1 || !strings.Contains(asked[0].Detail, "in ConfigMaps") {
		t.Errorf("confirmations %+v", asked)
	}
	ref := engine.HelmReleaseRef(contextName, driverNamespace, "hd-web")
	waitAt := func(rev int) {
		t.Helper()
		if !testkit.WaitFor(ctx, func() bool {
			obj, err := k.s.Lookup(ref)
			return err == nil && helm.ReleaseRevision(obj) == rev && helm.ReleaseStatus(obj) == "deployed"
		}) {
			t.Fatalf("release hd-web never deployed at revision %d", rev)
		}
	}
	records := func() (secrets, configMaps []string) {
		t.Helper()
		ss, err := kc.CoreV1().Secrets(driverNamespace).List(ctx, metav1.ListOptions{LabelSelector: helm.ReleaseSelector})
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range ss.Items {
			secrets = append(secrets, s.Name)
		}
		cms, err := kc.CoreV1().ConfigMaps(driverNamespace).List(ctx, metav1.ListOptions{LabelSelector: helm.ReleaseSelector})
		if err != nil {
			t.Fatal(err)
		}
		for _, c := range cms.Items {
			configMaps = append(configMaps, c.Name)
		}
		slices.Sort(configMaps)
		return secrets, configMaps
	}
	waitAt(1)
	if s, c := records(); len(s) != 0 || strings.Join(c, ",") != "sh.helm.release.v1.hd-web.v1" {
		t.Fatalf("after the install: Secrets %v, ConfigMaps %v", s, c)
	}

	// The Releases view lists it with its storage; agents read the release ConfigMap redacted.
	raw, _ := json.Marshal(map[string]any{"view": helm.ReleaseKind})
	if _, err := k.s.Dispatch(ctx, engine.Invocation{ID: "ui.openView", Params: raw, Origin: engine.OriginGUI}); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool {
		ts := k.s.Table(helm.ReleaseKind)
		if ts == nil || !ts.Synced {
			return false
		}
		for _, r := range ts.Rows {
			if r.Ref.Name == "hd-web" && r.Ref.Namespace == driverNamespace {
				return true
			}
		}
		return false
	}) {
		t.Fatalf("the Releases view %+v", k.s.Table(helm.ReleaseKind))
	}
	m, err := k.s.DetailWait(ctx, ref)
	if err != nil || !strings.Contains(sectionOf(m, model.TitleProperties), "ConfigMaps") {
		t.Errorf("the drawer %+v %v", m, err)
	}
	res, err := k.dispatch(ctx, engine.OriginExternalAgent, "object.get", map[string]any{"ref": model.ObjectRef{
		Context: contextName, Kind: "configmaps", Namespace: driverNamespace, Name: "sh.helm.release.v1.hd-web.v1"}})
	if err != nil {
		t.Fatal(err)
	}
	cm, err := kc.CoreV1().ConfigMaps(driverNamespace).Get(ctx, "sh.helm.release.v1.hd-web.v1", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := json.Marshal(res); strings.Contains(string(b), cm.Data["release"][:32]) || !strings.Contains(string(b), "Helm release record") {
		t.Errorf("an agent reads the release ConfigMap: %s", b)
	}

	// Upgrade and rollback through the ConfigMaps driver.
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.release.upgrade", map[string]any{"ref": ref, "values": "greeting: upgraded\n", "revision": 1}); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	waitAt(2)
	if got, err := kc.CoreV1().ConfigMaps(driverNamespace).Get(ctx, "hd-web-config", metav1.GetOptions{}); err != nil || got.Data["greeting"] != "upgraded" {
		t.Errorf("the release's ConfigMap after the upgrade: %v %v", got, err)
	}
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.release.rollback", map[string]any{"ref": ref, "revision": 1}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	waitAt(3)
	if s, c := records(); len(s) != 0 || len(c) != 3 {
		t.Errorf("after upgrade and rollback: Secrets %v, ConfigMaps %v", s, c)
	}
	if list, _ := journal.List(); len(list) != 0 {
		t.Errorf("journal %+v", list)
	}
	if strings.Contains(k.audit.String(), "hd-s3cr3t") {
		t.Error("the values in the audit log")
	}

	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.release.uninstall", map[string]any{"ref": ref}); err != nil {
		t.Fatalf("uninstall: %v", err)
	}
	if !testkit.WaitFor(ctx, func() bool { _, err := k.s.Lookup(ref); return err != nil }) {
		t.Error("the release is still listed")
	}
	if s, c := records(); len(s)+len(c) != 0 {
		t.Errorf("after the uninstall: Secrets %v, ConfigMaps %v", s, c)
	}
	if err := kc.CoreV1().Namespaces().Delete(ctx, driverNamespace, metav1.DeleteOptions{}); err != nil {
		t.Errorf("deleting the namespace: %v", err)
	}
}
