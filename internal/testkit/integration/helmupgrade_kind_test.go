//go:build kind

package integration_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"strings"
	"testing"
	"time"

	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	"helm.sh/helm/v4/pkg/kube"
	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// upgradeNamespace is the namespace of the Helm upgrade kind test (agent C of the fourth run; not
// cogpit-c, which TestKindConfigViews deletes);
// the test deletes it afterwards.
const upgradeNamespace = "cogpit-c-helm"

// TestKindHelmUpgrade installs the local test chart with the Helm SDK, then upgrades the release
// through Cogpit's engine on the kind cluster: from a Helm Upgrade tab (the GUI's path) and with
// values an agent passes (confirmed), with the requests the SDK makes counted; a value the chart
// refuses (its template's fail quotes it) leaves the release unchanged and is never shown; a
// release that moved since the tab read its values is refused; and a simulated crash of an
// upgrade (the journal written, the release left pending-upgrade) is offered by the next session,
// refused as an upgrade and finished by a rollback to the revision it started from.
func TestKindHelmUpgrade(t *testing.T) {
	kubeconfig, contextName := kindEnv(t)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	kc := conn.Kube()
	kindNamespace(ctx, t, kc, upgradeNamespace)

	const name = "c-web"
	cfg, err := helm.NewActionConfig(conn.RESTConfig(), upgradeNamespace, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	chrt, err := loader.Load("testdata/helmchart")
	if err != nil {
		t.Fatal(err)
	}
	install := action.NewInstall(cfg)
	install.ReleaseName, install.Namespace = name, upgradeNamespace
	install.WaitStrategy, install.Timeout = kube.HookOnlyStrategy, time.Minute
	if _, err := install.RunWithContext(ctx, chrt, map[string]any{"greeting": "v1", "password": "values-s3cr3t"}); err != nil {
		t.Fatalf("install: %v", err)
	}

	journal := engine.NewMemoryHelmJournal()
	k := newKindSession(ctx, t, kubeconfig, contextName, journal)
	ref := model.ObjectRef{Context: contextName, Kind: helm.ReleaseKind, Namespace: upgradeNamespace, Name: name}
	waitRelease := func(k *kindSession, rev int, status string) {
		t.Helper()
		if !testkit.WaitFor(ctx, func() bool {
			obj, err := k.s.Lookup(ref)
			return err == nil && helm.ReleaseRevision(obj) == rev && helm.ReleaseStatus(obj) == status
		}) {
			obj, err := k.s.Lookup(ref)
			t.Fatalf("release never at revision %d %s: %v %v", rev, status, obj, err)
		}
	}
	waitRelease(k, 1, "deployed")

	// The GUI's path: the tab with the user-supplied values from the cache, edited, saved.
	res, err := k.dispatch(ctx, engine.OriginGUI, helm.UpgradeTabCommand, map[string]any{"ref": ref})
	if err != nil {
		t.Fatal(err)
	}
	tab := res.(commands.EditResult)
	if tab.Text != "greeting: v1\npassword: values-s3cr3t\n" || tab.Tab.Helm == nil || tab.Tab.Helm.Chart != "cogpit-test" || tab.Tab.Helm.Version != "0.1.0" {
		t.Fatalf("tab %+v", tab)
	}
	if _, err := k.dispatch(ctx, engine.OriginGUI, "editor.setText", map[string]any{"id": tab.Tab.ID, "yaml": "greeting: v2\npassword: values-s3cr3t\n"}); err != nil {
		t.Fatal(err)
	}
	before := k.s.Backend().Conn.Counter().Snapshot()
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.release.upgrade", map[string]any{"ref": ref, "tab": tab.Tab.ID, "textRevision": kindTabRevision(ctx, t, k, tab.Tab.ID)}); err != nil {
		t.Fatalf("upgrade from the tab: %v", err)
	}
	after := k.s.Backend().Conn.Counter().Snapshot()
	sdkRequests := map[string]int64{}
	for key, n := range after.ByVerbResource {
		if d := n - before.ByVerbResource[key]; d > 0 {
			sdkRequests[key] = d
		}
	}
	t.Logf("requests of one upgrade (the SDK's, and the cache's meanwhile): %v", sdkRequests)
	waitRelease(k, 2, "deployed")
	if greetingIn(ctx, t, kc, name, upgradeNamespace) != "v2" {
		t.Error("the upgrade did not change the release's ConfigMap")
	}
	if list, _ := journal.List(); len(list) != 0 {
		t.Errorf("journal after the upgrade: %+v", list)
	}
	m := k.detail(ctx, t, engine.OriginExternalAgent, ref)
	if sectionOf(m, helm.TitleNotes) != model.Withheld {
		t.Errorf("notes for an agent: %q", sectionOf(m, helm.TitleNotes))
	}
	if gui := k.detail(ctx, t, engine.OriginCLI, ref); !strings.Contains(sectionOf(gui, helm.TitleNotes), `greets with "v2"`) {
		t.Errorf("notes: %q", sectionOf(gui, helm.TitleNotes))
	}

	// An agent's upgrade with values and the revision they are based on: confirmed, audited
	// without the values; the editor's values withheld from it.
	if st, err := k.dispatch(ctx, engine.OriginExternalAgent, "editor.get", map[string]any{"id": tab.Tab.ID}); err != nil ||
		st.(engine.EditorState).Text != model.Withheld || st.(engine.EditorState).BaseRevision != 2 {
		t.Errorf("editor.get for an agent: %+v, %v", st, err)
	}
	if _, err := k.dispatch(ctx, engine.OriginExternalAgent, "helm.release.upgrade", map[string]any{
		"ref": ref, "values": "greeting: v3\npassword: agent-s3cr3t\n", "revision": 2}); err != nil {
		t.Fatalf("agent upgrade: %v", err)
	}
	waitRelease(k, 3, "deployed")
	k.mu.Lock()
	asked := slices.Clone(k.asked)
	k.mu.Unlock()
	if len(asked) != 1 || asked[0].Danger != engine.DangerMutating || asked[0].Command != "helm.release.upgrade" {
		t.Errorf("confirmations %+v", asked)
	}

	// The tab read revision 2; the release is at 3: refused.
	if _, err := k.dispatch(ctx, engine.OriginGUI, "helm.release.upgrade", map[string]any{"ref": ref, "tab": tab.Tab.ID, "textRevision": kindTabRevision(ctx, t, k, tab.Tab.ID)}); !errors.Is(err, engine.ErrTargetsChanged) {
		t.Errorf("an upgrade from a tab whose release moved: %v", err)
	}

	// A value the chart refuses: the SDK's error quotes it, Cogpit does not; nothing changed.
	_, err = k.dispatch(ctx, engine.OriginGUI, "helm.release.upgrade", map[string]any{"ref": ref, "values": "refuse: s3cr3t-refusal\n", "revision": 3})
	if err == nil || !errors.Is(err, helm.ErrUnchanged) || strings.Contains(err.Error(), "s3cr3t-refusal") || !strings.Contains(err.Error(), "refused") {
		t.Errorf("a refused value: %v", err)
	}
	if list, _ := journal.List(); len(list) != 0 {
		t.Errorf("journal after an unchanged failure: %+v", list)
	}
	waitRelease(k, 3, "deployed")
	for _, leak := range []string{"values-s3cr3t", "agent-s3cr3t", "s3cr3t-refusal"} {
		if strings.Contains(k.audit.String(), leak) {
			t.Errorf("%q in the audit log", leak)
		}
	}
	k.s.Close()

	// A crash during an upgrade from revision 3: Cogpit recorded it, the SDK created revision 4 as
	// pending-upgrade, and the process died before the release's objects changed.
	rev3, err := cfg.Releases.Get(name, 3)
	if err != nil {
		t.Fatal(err)
	}
	pending := *rev3.(*rspb.Release)
	info := *pending.Info
	info.Status, info.Description, info.LastDeployed = common.StatusPendingUpgrade, "Preparing upgrade", time.Now()
	pending.Info, pending.Version = &info, 4
	pending.Config = map[string]any{"greeting": "v4"}
	if err := cfg.Releases.Create(&pending); err != nil {
		t.Fatal(err)
	}
	if err := journal.Put(engine.PendingHelmOp{Context: contextName, Namespace: upgradeNamespace, Release: name,
		Action: engine.HelmUpgrade, FromRevision: 3, Chart: "cogpit-test", ChartVersion: "0.1.0",
		Started: time.Now().UTC(), StartedBy: engine.OriginGUI, State: engine.HelmOpRunning}); err != nil {
		t.Fatal(err)
	}

	k2 := newKindSession(ctx, t, kubeconfig, contextName, journal)
	defer k2.s.Close()
	var offered []engine.PendingHelmOpInfo
	if !testkit.WaitFor(ctx, func() bool {
		res, err := k2.dispatch(ctx, engine.OriginExternalAgent, "helm.pendingOperations", struct{}{})
		if err != nil {
			return false
		}
		b, _ := json.Marshal(res)
		offered = nil
		_ = json.Unmarshal(b, &offered)
		return len(offered) == 1 && offered[0].ReleaseStatus == "pending-upgrade"
	}) {
		t.Fatalf("not offered: %+v", offered)
	}
	if o := offered[0]; o.Action != engine.HelmUpgrade || o.FromRevision != 3 || o.ReleaseRevision != 4 || o.Status != engine.HelmOpStatusInterrupted {
		t.Errorf("offer %+v", o)
	}
	if _, err := k2.dispatch(ctx, engine.OriginGUI, "helm.release.upgrade", map[string]any{"ref": ref, "values": "greeting: v5\n", "revision": 4}); err == nil {
		t.Error("an upgrade of a release left pending-upgrade ran")
	}
	if _, err := k2.dispatch(ctx, engine.OriginGUI, "helm.release.rollback", map[string]any{"ref": ref, "revision": offered[0].FromRevision}); err != nil {
		t.Fatalf("finishing the upgrade by a rollback: %v", err)
	}
	waitRelease(k2, 5, "deployed")
	if greetingIn(ctx, t, kc, name, upgradeNamespace) != "v3" {
		t.Error("the rollback did not restore revision 3's ConfigMap")
	}
	if list, _ := journal.List(); len(list) != 0 {
		t.Errorf("journal after finishing: %+v", list)
	}
	secrets, err := kc.CoreV1().Secrets(upgradeNamespace).List(ctx, metav1.ListOptions{LabelSelector: helm.ReleaseSelector})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("release Secrets kept: %d; SDK request kinds of one upgrade: %v", len(secrets.Items), slices.Sorted(maps.Keys(sdkRequests)))
}

// greetingIn reads the test chart's ConfigMap in a namespace.
func greetingIn(ctx context.Context, t *testing.T, kc kubernetes.Interface, release, namespace string) string {
	t.Helper()
	cm, err := kc.CoreV1().ConfigMaps(namespace).Get(ctx, release+"-config", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	return cm.Data["greeting"]
}

// kindTabRevision is a tab's text revision as editor.get reports it.
func kindTabRevision(ctx context.Context, t *testing.T, k *kindSession, id string) uint64 {
	t.Helper()
	res, err := k.dispatch(ctx, engine.OriginGUI, "editor.get", map[string]any{"id": id})
	if err != nil {
		t.Fatal(err)
	}
	return res.(engine.EditorState).Revision
}
