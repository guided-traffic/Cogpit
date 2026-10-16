//go:build kind

package integration_test

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The live Argo CD check's objects: its two AppProjects and three Applications in Argo CD's
// namespace, and the namespace they deploy into, all created and deleted by the test.
const (
	realArgoDest    = "cogpit-real-argo-dest"
	realArgoProject = "cogpit-real"
	realArgoFrozen  = "cogpit-real-frozen"
	realArgoChart   = "https://stefanprodan.github.io/podinfo" // a public Helm repository: no Git anywhere
	realArgoVersion = "6.15.0"
	argoFinalizer   = "resources-finalizer.argocd.argoproj.io"
)

// TestRealArgoCD drives Cogpit's Argo CD section against a running Argo CD controller on a real
// cluster, never in CI: it runs only when COGPIT_REAL_KUBECONFIG names a kubeconfig (its current
// context, or COGPIT_REAL_CONTEXT) and COGPIT_REAL_ARGOCD_NAMESPACE the namespace Argo CD runs in,
// whose repo server can reach the public podinfo Helm repository. It creates there two AppProjects
// (one in a deny window) and three manual Applications of podinfo's chart deploying into a
// namespace of its own, and checks through the engine with origin GUI and a confirming hook, as
// the kind tests do: the rows and their status tokens once the controller compared them; Refresh,
// normal and hard, and Refresh on a selection, each reconciled by the controller (its annotation
// gone, reconciledAt after the request); Sync refused in the deny window; Sync and Sync with prune
// on a selection run by the controller to Synced and Healthy, its history naming cogpit and its
// operation the source Cogpit confirmed; the drawer's Resources and project link; the Dashboard's
// counts against the API server's list; the Application's merged logs (A81); Open Argo CD UI, a
// forward to argocd-server that answers its health check; the settings views and the audit log
// without any credential of Argo CD's labelled Secrets (D-35). It deletes what
// it created, the Applications with their resources (Argo CD's finalizer) first.
func TestRealArgoCD(t *testing.T) {
	kubeconfig, argoNS := os.Getenv("COGPIT_REAL_KUBECONFIG"), os.Getenv("COGPIT_REAL_ARGOCD_NAMESPACE")
	if kubeconfig == "" || argoNS == "" {
		t.Skip("COGPIT_REAL_KUBECONFIG and COGPIT_REAL_ARGOCD_NAMESPACE are not set: a check against a live Argo CD, never in CI")
	}
	contextName := os.Getenv("COGPIT_REAL_CONTEXT")
	if contextName == "" {
		cfg, err := clientcmd.LoadFromFile(kubeconfig)
		if err != nil {
			t.Fatal(err)
		}
		contextName = cfg.CurrentContext
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(conn.Close) // after the objects' deletion, which uses it
	apps := conn.Dynamic().Resource(testkit.ArgoApplicationsGVR).Namespace(argoNS)
	projects := conn.Dynamic().Resource(testkit.ArgoProjectsGVR).Namespace(argoNS)
	freshNamespace(ctx, t, conn.Kube(), realArgoDest) // deleted last
	realArgoObjects(ctx, t, apps, projects)

	audit, logs := &lockedBuffer{}, &lockedBuffer{}
	s := engine.New(engine.Options{
		Connector: &engine.ClusterConnector{KubeconfigPaths: []string{kubeconfig}}, Audit: audit,
		Logger:    slog.New(slog.NewTextHandler(io.MultiWriter(logs), &slog.HandlerOptions{Level: slog.LevelDebug})),
		Confirmer: engine.ConfirmFunc(func(context.Context, engine.ConfirmRequest) (bool, error) { return true, nil }),
	})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, contextName); err != nil {
		t.Fatal(err)
	}
	waitFor(ctx, t, "Argo CD's kinds", func() bool { _, ok := s.Backend().Profile(kinds.ArgoClustersKey); return ok })
	d := dispatcher{t: t, ctx: ctx, s: s}
	ref := func(name string) model.ObjectRef {
		return model.ObjectRef{Kind: kinds.ArgoApplicationsKey, Namespace: argoNS, Name: name}
	}
	one, two, frozen := ref(realArgoProject+"-one"), ref(realArgoProject+"-two"), ref(realArgoFrozen)

	// The rows once the controller compared them: OutOfSync and Missing, a warning.
	d.must("ui.openView", map[string]any{"view": kinds.ArgoApplicationsKey}, nil)
	waitFor(ctx, t, "the controller's comparison in the rows", func() bool {
		tbl := s.Table(kinds.ArgoApplicationsKey)
		if tbl == nil || !tbl.Synced {
			return false
		}
		for _, r := range []model.ObjectRef{one, two, frozen} {
			i := tbl.IndexOf(r)
			if i < 0 || tbl.Rows[i].Cells[2].Text != kinds.ArgoOutOfSync || tbl.Rows[i].Cells[3].Text != kinds.ArgoMissing ||
				tbl.Rows[i].Status != model.StatusWarning {
				return false
			}
		}
		return true
	})

	// Sync in the deny window: refused, nothing written.
	if err := d.run("argocd.application.sync", map[string]any{"ref": frozen}, nil); err == nil || !strings.Contains(err.Error(), "deny window") {
		t.Errorf("a sync in the deny window: %v", err)
	}

	// Refresh, normal, hard and on a selection: the controller reconciles and removes the annotation.
	realRefresh(ctx, t, d, apps, []model.ObjectRef{one}, "argocd.application.refresh", map[string]any{"ref": one})
	realRefresh(ctx, t, d, apps, []model.ObjectRef{two}, "argocd.application.refresh", map[string]any{"ref": two, "hard": true})
	realRefresh(ctx, t, d, apps, []model.ObjectRef{one, two}, "argocd.application.refreshMany", map[string]any{"refs": []model.ObjectRef{one, two}})

	// Sync, then Sync with prune on a selection: run by the controller to Synced and Healthy.
	d.must("argocd.application.sync", map[string]any{"ref": one}, nil)
	realSynced(ctx, t, d, one, 1)
	d.must("argocd.application.syncMany", map[string]any{"refs": []model.ObjectRef{one, two}, "prune": true}, nil)
	realSynced(ctx, t, d, one, 2)
	realSynced(ctx, t, d, two, 1)

	realDrawer(ctx, t, s, one, argoNS)
	realDashboard(ctx, t, s, apps, argoNS)
	realLogs(ctx, t, d, s, one)
	realOpenUI(ctx, t, d, s)
	realSettingsWithoutCredentials(ctx, t, s, conn, argoNS, audit, logs)
}

// realOpenUI runs Open Argo CD UI from the GUI: a forward of the argocd-server Service's https
// port, whose URL answers Argo CD's health check; the forward stops after.
func realOpenUI(ctx context.Context, t *testing.T, d dispatcher, s *engine.Session) {
	t.Helper()
	var f engine.Forward
	d.must("argocd.openUI", map[string]any{}, &f)
	defer func() { _, _ = s.StopForward(f.Ref) }()
	if f.Spec.Port != 443 || !f.Spec.HTTPS {
		t.Errorf("the forward %+v, want argocd-server's https port", f.Spec)
	}
	waitFor(ctx, t, "the forward to listen", func() bool {
		got, err := s.LookupForward(f.Ref)
		f = got
		return err == nil && got.Listening()
	})
	// argocd-server's own self-signed certificate, reached through a forward on 127.0.0.1.
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- a test against argocd-server's self-signed certificate on a local forward
	}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, f.URL+"/healthz", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Argo CD's UI through %s: %v", f.URL, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<10))
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "ok" {
		t.Errorf("Argo CD's UI through %s: %d %q", f.URL, resp.StatusCode, body)
	}
}

// realArgoObjects creates the projects and Applications, and their deletion: the Applications with
// their resources first (Argo CD's finalizer), then the projects.
func realArgoObjects(ctx context.Context, t *testing.T, apps, projects dynamic.ResourceInterface) {
	t.Helper()
	inCluster := map[string]any{"server": "https://kubernetes.default.svc", "namespace": realArgoDest}
	for name, windows := range map[string][]any{
		realArgoProject: nil,
		realArgoFrozen:  {map[string]any{"kind": "deny", "schedule": "* * * * *", "duration": "1h", "applications": []any{"*"}}},
	} {
		spec := map[string]any{
			"description": "Cogpit's live check", "sourceRepos": []any{realArgoChart}, "destinations": []any{inCluster},
		}
		if windows != nil {
			spec["syncWindows"] = windows
		}
		p := testkit.Object("argoproj.io/v1alpha1", "AppProject", "", name, map[string]any{"spec": spec})
		p.SetUID("")
		p.SetCreationTimestamp(metav1.Time{})
		if _, err := projects.Create(ctx, p, metav1.CreateOptions{}); err != nil {
			t.Fatalf("creating the project %s: %v", name, err)
		}
		t.Cleanup(func() {
			dctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			if err := projects.Delete(dctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				t.Errorf("deleting the project %s: %v", name, err)
			}
		})
	}
	for name, project := range map[string]string{realArgoProject + "-one": realArgoProject, realArgoProject + "-two": realArgoProject, realArgoFrozen: realArgoFrozen} {
		a := testkit.Object("argoproj.io/v1alpha1", "Application", "", name, map[string]any{"spec": map[string]any{
			"project": project,
			"source": map[string]any{"repoURL": realArgoChart, "chart": "podinfo", "targetRevision": realArgoVersion,
				"helm": map[string]any{"releaseName": name, "valuesObject": map[string]any{"replicaCount": int64(1)}}},
			"destination": inCluster,
		}})
		a.SetUID("")
		a.SetCreationTimestamp(metav1.Time{})
		a.SetFinalizers([]string{argoFinalizer})
		if _, err := apps.Create(ctx, a, metav1.CreateOptions{}); err != nil {
			t.Fatalf("creating the Application %s: %v", name, err)
		}
		t.Cleanup(func() { deleteRealApp(t, apps, name) })
	}
}

// deleteRealApp deletes one of the test's Applications and waits for Argo CD to delete its
// resources and remove its finalizer; should it not within three minutes, the finalizer comes off
// (the object is the test's own) and the test says so.
func deleteRealApp(t *testing.T, apps dynamic.ResourceInterface, name string) {
	dctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	if err := apps.Delete(dctx, name, metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		t.Errorf("deleting the Application %s: %v", name, err)
		return
	}
	wctx, wcancel := context.WithTimeout(dctx, 3*time.Minute)
	defer wcancel()
	if testkit.WaitFor(wctx, func() bool { _, err := apps.Get(dctx, name, metav1.GetOptions{}); return apierrors.IsNotFound(err) }) {
		return
	}
	t.Errorf("Argo CD did not delete the Application %s in three minutes; its finalizer is removed", name)
	_, _ = apps.Patch(dctx, name, types.MergePatchType, []byte(`{"metadata":{"finalizers":null}}`), metav1.PatchOptions{})
}

// realRefresh refreshes and waits for the controller: the annotation gone and every Application
// reconciled at or after the request.
func realRefresh(ctx context.Context, t *testing.T, d dispatcher, apps dynamic.ResourceInterface, refs []model.ObjectRef, id string, params any) {
	t.Helper()
	asked := time.Now().Truncate(time.Second)
	d.must(id, params, nil)
	for _, r := range refs {
		waitFor(ctx, t, "the controller to refresh "+r.Name+" ("+id+")", func() bool {
			got, err := apps.Get(ctx, r.Name, metav1.GetOptions{})
			if err != nil {
				return false
			}
			_, pending := got.GetAnnotations()[cluster.ArgoRefreshAnnotation]
			return !pending && !model.NestedTime(got, "status", "reconciledAt").Before(asked)
		})
	}
}

// realSynced waits until the controller has run the n-th sync Cogpit started: Synced, Healthy,
// the operation initiated by cogpit with the source it confirmed, n entries of the history.
func realSynced(ctx context.Context, t *testing.T, d dispatcher, ref model.ObjectRef, n int) {
	t.Helper()
	d.cached(ref, func(o *unstructured.Unstructured) bool {
		return kinds.ArgoOperationPhase(o) == "Succeeded" && kinds.ArgoSyncStatus(o) == kinds.ArgoSynced &&
			kinds.ArgoHealthStatus(o) == kinds.ArgoHealthy && len(model.NestedSlice(o, "status", "history")) == n &&
			model.NestedString(o, "status", "operationState", "operation", "initiatedBy", "username") == kinds.ArgoSyncInitiator
	})
	obj, _ := d.s.Lookup(ref)
	if got := model.NestedString(obj, "status", "operationState", "operation", "sync", "source", "chart"); got != "podinfo" {
		t.Errorf("%s: the operation's source chart %q", ref.Name, got)
	}
	if got := kinds.ArgoAppActivity(obj); len(got) == 0 || !strings.Contains(got[0].Text, "by "+kinds.ArgoSyncInitiator) {
		t.Errorf("%s: activity %+v", ref.Name, got)
	}
}

// realDrawer reads the drawer as an agent: the project linked in Argo CD's namespace, the
// Resources with the Health column only when the controller writes the resources' health.
func realDrawer(ctx context.Context, t *testing.T, s *engine.Session, ref model.ObjectRef, argoNS string) {
	t.Helper()
	var m engine.DetailModel
	agentInto(ctx, t, s, "object.get", map[string]any{"ref": ref, "noYaml": true}, &m)
	obj, _ := s.Lookup(ref)
	inTree := model.NestedString(obj, "status", "resourceHealthSource") == "appTree"
	found := false
	for _, sec := range m.Sections {
		for _, it := range sec.Items {
			if it.Name == "Project" && (it.Link == nil || it.Link.Namespace != argoNS) {
				t.Errorf("project link %+v", it.Link)
			}
		}
		if sec.Title != "Resources" || sec.Table == nil {
			continue
		}
		found = true
		cols := strings.Join(sec.Table.Columns, ",")
		switch {
		case inTree && (cols != "Kind,Namespace,Name,Sync" || len(sec.Items) != 1 || sec.Items[0].Value != kinds.ArgoResourceHealthNote):
			t.Errorf("resources with the health in Argo CD's tree: %s %+v", cols, sec.Items)
		case !inTree && cols != "Kind,Namespace,Name,Sync,Health":
			t.Errorf("resources: %s", cols)
		}
		kindsSeen := map[string]bool{}
		for _, r := range sec.Table.Rows {
			kindsSeen[r[0]] = r[3] == kinds.ArgoSynced
		}
		if !kindsSeen["Deployment"] || !kindsSeen["Service"] {
			t.Errorf("resources %v", sec.Table.Rows)
		}
	}
	if !found {
		t.Errorf("no Resources section: %+v", m.Sections)
	}
}

// realDashboard compares the Dashboard's strip with the API server's list of the namespace's
// Applications, until they agree (the controller writes status meanwhile).
func realDashboard(ctx context.Context, t *testing.T, s *engine.Session, apps dynamic.ResourceInterface, argoNS string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	var last string
	ok := testkit.WaitFor(ctx, func() bool {
		list, err := apps.List(ctx, metav1.ListOptions{})
		if err != nil {
			return false
		}
		want := map[string]int{"Applications": len(list.Items)}
		for i := range list.Items {
			o := &list.Items[i]
			for label, cond := range map[string]bool{
				"Synced": kinds.ArgoSyncStatus(o) == kinds.ArgoSynced, "Out of sync": kinds.ArgoSyncStatus(o) == kinds.ArgoOutOfSync,
				"Healthy": kinds.ArgoHealthStatus(o) == kinds.ArgoHealthy, "Degraded": kinds.ArgoHealthStatus(o) == kinds.ArgoDegraded,
				"Missing": kinds.ArgoHealthStatus(o) == kinds.ArgoMissing, "Progressing": kinds.ArgoHealthStatus(o) == kinds.ArgoProgressing,
			} {
				if cond {
					want[label]++
				}
			}
		}
		m, _, err := s.ArgoCDDashboard(ctx, "", []string{argoNS}, false)
		if err != nil || !m.Served || !m.Synced {
			return false
		}
		got := map[string]int{}
		for _, tile := range m.Strip {
			if tile.Count > 0 {
				got[tile.Label] = tile.Count
			}
		}
		b, _ := json.Marshal(map[string]any{"dashboard": got, "api": want})
		last = string(b)
		return len(got) == len(want) && func() bool {
			for k, v := range want {
				if got[k] != v {
					return false
				}
			}
			return true
		}()
	})
	if !ok {
		t.Errorf("the Dashboard's strip never matched the API server's list: %s", last)
	}
}

// realLogs opens the Application's logs in one tab (A81) and waits for a line of its Deployment,
// named as its status.resources names it ("<workload>/<pod>/<container> …").
func realLogs(ctx context.Context, t *testing.T, d dispatcher, s *engine.Session, ref model.ObjectRef) {
	t.Helper()
	obj, err := s.Lookup(ref)
	if err != nil {
		t.Fatal(err)
	}
	var workload string
	for _, r := range model.NestedSlice(obj, "status", "resources") {
		if model.NestedString(r, "kind") == "Deployment" {
			workload = model.NestedString(r, "name")
		}
	}
	if workload == "" {
		t.Fatalf("%s manages no Deployment: %v", ref.Name, model.NestedSlice(obj, "status", "resources"))
	}
	var tab commands.DockResult
	d.must("argocd.logs", map[string]any{"ref": ref}, &tab)
	stream, err := s.OpenLogs(ctx, tab.Tab.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	lctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	collect(stream).waitLines(lctx, t, "a line of the Application's workload "+workload, func(got []string) bool {
		return slices.ContainsFunc(got, func(l string) bool { return strings.HasPrefix(l, workload+"/"+workload+"-") })
	})
}

// realSettingsWithoutCredentials reads every labelled Secret of Argo CD's namespace as its settings
// view shows it to an agent: no value of a key outside the allowlist reaches object_get,
// resources_list, the audit log or the log (D-35).
func realSettingsWithoutCredentials(ctx context.Context, t *testing.T, s *engine.Session, conn *cluster.Conn, argoNS string, audit, logs *lockedBuffer) {
	t.Helper()
	secrets, err := conn.Kube().CoreV1().Secrets(argoNS).List(ctx, metav1.ListOptions{LabelSelector: kinds.ArgoSecretTypeLabel})
	if err != nil {
		t.Fatal(err)
	}
	var forbidden []string
	var texts []string
	for _, kind := range []string{kinds.ArgoRepositoriesKey, kinds.ArgoClustersKey} {
		var list any
		agentInto(ctx, t, s, "resources.list", map[string]any{"kind": kind, "namespaces": []string{argoNS}}, &list)
		b, _ := json.Marshal(list)
		texts = append(texts, string(b))
	}
	for _, sec := range secrets.Items {
		allowed := kinds.ArgoRepositoryKeys
		kind := kinds.ArgoRepositoriesKey
		if sec.Labels[kinds.ArgoSecretTypeLabel] == "cluster" {
			allowed, kind = kinds.ArgoClusterKeys, kinds.ArgoClustersKey
		}
		for k, v := range sec.Data {
			if !slices.Contains(allowed, k) && len(v) >= 6 {
				forbidden = append(forbidden, string(v), base64.StdEncoding.EncodeToString(v))
			}
		}
		var got any
		agentInto(ctx, t, s, "object.get", map[string]any{"ref": model.ObjectRef{Kind: kind, Namespace: argoNS, Name: sec.Name}}, &got)
		b, _ := json.Marshal(got)
		texts = append(texts, string(b))
	}
	texts = append(texts, audit.String(), logs.String())
	for _, text := range texts {
		for _, f := range forbidden {
			if strings.Contains(text, f) {
				t.Errorf("a credential of Argo CD's Secrets reached an agent's read or a log (%d bytes)", len(f))
			}
		}
	}
}

// agentInto dispatches as an external agent and decodes the result through JSON.
func agentInto(ctx context.Context, t *testing.T, s *engine.Session, id string, params, out any) {
	t.Helper()
	raw, _ := json.Marshal(params)
	res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
	if err != nil {
		t.Fatalf("%s: %v", id, err)
	}
	b, _ := json.Marshal(res)
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatalf("%s: %v", id, err)
	}
}
