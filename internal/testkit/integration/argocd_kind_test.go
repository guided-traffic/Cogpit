//go:build kind

package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// argoNamespace is TestKindArgoCD's namespace.
const argoNamespace = "cogpit-argo-argocd"

// argoPassword is the fake password of the test's repository Secret; argoToken its cluster
// Secret's fake bearer token.
const (
	argoPassword = "fake-kind-repo-password"
	argoToken    = "fake-kind-cluster-token"
)

// lockedBuffer is a buffer two goroutines write to.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// TestKindArgoCD installs hand-written stand-ins for Argo CD's definitions (testdata/argocd-crds.yaml;
// no Argo CD runs), creates an Application with the status Argo CD's controller would write and
// a repository and a cluster Secret with fake credentials in a namespace of its own, and checks
// over the engine: the curated rows, the settings views without any credential (an agent's
// object_get, resources_list, the audit log and the log searched for the password and the token),
// Refresh (normal, then hard) and Sync setting what the API server then holds, a second Sync
// refused while the first is requested, and the API server's refusal of a Sync bound to another
// UID or an older resourceVersion. A cluster that already has Argo CD's definitions is left alone
// (the test skips); the definitions are removed after.
func TestKindArgoCD(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	crds := conn.Dynamic().Resource(testkit.CRDsGVR)
	for _, name := range []string{kinds.ArgoApplicationsKey, kinds.ArgoApplicationSetKey, kinds.ArgoProjectsKey} {
		if _, err := crds.Get(ctx, name, metav1.GetOptions{}); err == nil {
			t.Skip("the cluster has Argo CD's definitions of its own; the test installs its own only")
		} else if !apierrors.IsNotFound(err) {
			t.Fatal(err)
		}
	}
	defs := argoCRDsFromTestdata(t)
	t.Cleanup(func() {
		dctx, dcancel := context.WithTimeout(context.Background(), time.Minute)
		defer dcancel()
		for _, d := range defs {
			if err := crds.Delete(dctx, d.GetName(), metav1.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
				t.Errorf("deleting %s: %v", d.GetName(), err)
			}
		}
		_ = conn.Kube().CoreV1().Namespaces().Delete(dctx, argoNamespace, metav1.DeleteOptions{})
	})
	for _, d := range defs {
		if _, err := crds.Create(ctx, d, metav1.CreateOptions{}); err != nil {
			t.Fatalf("creating %s: %v", d.GetName(), err)
		}
	}
	freshNamespace(ctx, t, conn.Kube(), argoNamespace)

	apps := conn.Dynamic().Resource(testkit.ArgoApplicationsGVR).Namespace(argoNamespace)
	app := testkit.ArgoApplication(argoNamespace, "shop", "OutOfSync", "Healthy")
	app.SetUID("")
	app.SetCreationTimestamp(metav1.Time{})
	waitFor(ctx, t, "the Application definition to serve", func() bool {
		_, err := apps.Create(ctx, app, metav1.CreateOptions{})
		return err == nil || apierrors.IsAlreadyExists(err)
	})
	for _, s := range []*unstructured.Unstructured{
		testkit.ArgoRepoSecret(argoNamespace, "repo-shop", "https://git.example.com/shop.git", argoPassword),
		testkit.ArgoClusterSecret(argoNamespace, "prod", "https://prod.example.com:6443", argoToken),
	} {
		s.SetUID("")
		s.SetCreationTimestamp(metav1.Time{})
		if _, err := conn.Dynamic().Resource(testkit.SecretsGVR).Namespace(argoNamespace).Create(ctx, s, metav1.CreateOptions{}); err != nil {
			t.Fatalf("creating %s: %v", s.GetName(), err)
		}
	}

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
	ref := model.ObjectRef{Kind: kinds.ArgoApplicationsKey, Namespace: argoNamespace, Name: "shop"}
	d.must("ui.openView", map[string]any{"view": kinds.ArgoApplicationsKey}, nil)
	waitFor(ctx, t, "the Application's row", func() bool {
		tbl := s.Table(kinds.ArgoApplicationsKey)
		if tbl == nil || !tbl.Synced {
			return false
		}
		i := tbl.IndexOf(ref)
		return i >= 0 && tbl.Rows[i].Cells[2].Text == "OutOfSync" && tbl.Rows[i].Cells[3].Text == "Healthy" &&
			tbl.Rows[i].Status == model.StatusWarning
	})

	// The settings views, read by an agent: the allowlist, never a credential.
	agent := func(id string, params any) []byte {
		t.Helper()
		raw, _ := json.Marshal(params)
		res, err := s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: engine.OriginExternalAgent})
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		b, _ := json.Marshal(res)
		return b
	}
	repo := agent("object.get", map[string]any{"ref": model.ObjectRef{Kind: kinds.ArgoRepositoriesKey, Namespace: argoNamespace, Name: "repo-shop"}})
	if !strings.Contains(string(repo), "https://git.example.com/shop.git") {
		t.Errorf("the repository's object_get has no URL: %s", repo)
	}
	list := agent("resources.list", map[string]any{"kind": kinds.ArgoClustersKey, "namespaces": []string{argoNamespace}})
	if !strings.Contains(string(list), "https://prod.example.com:6443") {
		t.Errorf("the clusters' resources_list has no server: %s", list)
	}
	forbidden := []string{argoPassword, argoToken, base64.StdEncoding.EncodeToString([]byte(argoPassword)), "deploy-bot"}
	for what, text := range map[string]string{"object_get": string(repo), "resources_list": string(list)} {
		for _, f := range forbidden {
			if strings.Contains(text, f) {
				t.Errorf("%s carries %q", what, f)
			}
		}
	}

	// Refresh, normal then hard: the annotation on the API server.
	annotation := func() string {
		got, err := apps.Get(ctx, "shop", metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		return got.GetAnnotations()[cluster.ArgoRefreshAnnotation]
	}
	d.must("argocd.application.refresh", map[string]any{"ref": ref}, nil)
	if a := annotation(); a != "normal" {
		t.Errorf("after a refresh the annotation is %q", a)
	}
	d.must("argocd.application.refresh", map[string]any{"ref": ref, "hard": true}, nil)
	if a := annotation(); a != "hard" {
		t.Errorf("after a hard refresh the annotation is %q", a)
	}

	// Sync: the operation on the API server; a second one refused while it is requested.
	d.cached(ref, func(o *unstructured.Unstructured) bool {
		return o.GetAnnotations()[cluster.ArgoRefreshAnnotation] == "hard"
	})
	d.must("argocd.application.sync", map[string]any{"ref": ref, "prune": true}, nil)
	got, err := apps.Get(ctx, "shop", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if model.NestedString(got, "operation", "initiatedBy", "username") != "cogpit" || model.NestedString(got, "operation", "sync", "revision") != "main" {
		t.Errorf("operation %v", got.Object["operation"])
	}
	if prune, _ := model.NestedBool(got, "operation", "sync", "prune"); !prune {
		t.Errorf("operation without prune: %v", got.Object["operation"])
	}
	d.cached(ref, func(o *unstructured.Unstructured) bool { _, ok := o.Object["operation"]; return ok })
	if err := d.run("argocd.application.sync", map[string]any{"ref": ref}, nil); err == nil || !strings.Contains(err.Error(), "operation was requested") {
		t.Errorf("a second sync while the first is requested: %v", err)
	}

	// The API server holds a Sync to the UID and the resourceVersion it names.
	op := kinds.ArgoSyncOperation(got, false)
	if err := conn.SyncArgoApplication(ctx, "v1alpha1", argoNamespace, "shop", "00000000-0000-0000-0000-000000000000", got.GetResourceVersion(), op); !errors.Is(err, cluster.ErrReplaced) {
		t.Errorf("a sync bound to another UID: %v", err)
	}
	if _, err := apps.Patch(ctx, "shop", "application/merge-patch+json", []byte(`{"metadata":{"labels":{"touched":"yes"}}}`), metav1.PatchOptions{}); err != nil {
		t.Fatal(err)
	}
	if err := conn.SyncArgoApplication(ctx, "v1alpha1", argoNamespace, "shop", string(got.GetUID()), got.GetResourceVersion(), op); !errors.Is(err, cluster.ErrArgoChanged) {
		t.Errorf("a sync bound to an older resourceVersion: %v", err)
	}

	for what, text := range map[string]string{"the audit log": audit.String(), "the log": logs.String()} {
		for _, f := range forbidden {
			if strings.Contains(text, f) {
				t.Errorf("%s carries %q", what, f)
			}
		}
	}
}

// argoCRDsFromTestdata reads the stand-in definitions.
func argoCRDsFromTestdata(t *testing.T) []*unstructured.Unstructured {
	t.Helper()
	raw, err := os.ReadFile("testdata/argocd-crds.yaml")
	if err != nil {
		t.Fatal(err)
	}
	dec := utilyaml.NewYAMLOrJSONDecoder(bytes.NewReader(raw), 4096)
	var out []*unstructured.Unstructured
	for {
		var m map[string]any
		if err := dec.Decode(&m); errors.Is(err, io.EOF) {
			break
		} else if err != nil {
			t.Fatal(err)
		}
		if m != nil {
			out = append(out, &unstructured.Unstructured{Object: m})
		}
	}
	if len(out) != 3 {
		t.Fatalf("%d definitions in testdata", len(out))
	}
	return out
}
