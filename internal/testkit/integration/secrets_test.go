package integration_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestBudgetSecretsViewAndDrawer runs the Secrets view of D-13 (H-8 option b) through the engine
// on a real client-go stack against the fake API server, and counts what it costs at the HTTP
// level: the list is one metadata-only informer (no data crosses the wire; the last-applied
// annotation, which repeats the values, is dropped as the objects enter the cache), and opening a
// Secret's drawer is exactly one GET of that Secret, however often the drawer refreshes; the
// values stay masked for agents and only the GUI can reveal one.
func TestBudgetSecretsViewAndDrawer(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{
		Resources: append(append([]testkit.FakeResource(nil), testkit.DefaultFakeResources...),
			testkit.FakeResource{GVR: testkit.SecretsGVR, Kind: "Secret", Namespaced: true}),
	})
	defer srv.Close()
	db := testkit.Secret("apps", "db", map[string]string{"app": "db"}, map[string]string{"password": "hunter2"})
	db.SetAnnotations(map[string]string{cache.LastAppliedAnnotation: `{"apiVersion":"v1","kind":"Secret","stringData":{"password":"hunter2"}}`})
	srv.Set(testkit.SecretsGVR, db)
	srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "tls", nil, map[string]string{"tls.key": "key"}))

	s := engine.New(engine.Options{
		Connector:        &engine.ClusterConnector{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}},
		SnapshotInterval: 10 * time.Millisecond,
	})
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Connect(ctx, "fake"); err != nil {
		t.Fatal(err)
	}
	dispatch := func(origin engine.Origin, id string, params any) (any, error) {
		raw, _ := json.Marshal(params)
		return s.Dispatch(ctx, engine.Invocation{ID: id, Params: raw, Origin: origin})
	}
	if _, err := dispatch(engine.OriginGUI, "ui.openView", map[string]any{"view": "secrets"}); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool { ts := s.Table("secrets"); return ts != nil && ts.Synced && len(ts.Rows) == 2 }) {
		t.Fatalf("no Secrets view: %+v", s.Table("secrets"))
	}
	obj, err := s.Object(ctx, model.ObjectRef{Kind: "secrets", Namespace: "apps", Name: "db"})
	if err != nil {
		t.Fatal(err)
	}
	if !cache.IsMetadataObject(obj) || obj.Object["data"] != nil || obj.GetAnnotations()[cache.LastAppliedAnnotation] != "" {
		t.Fatalf("the cache holds %v", obj.Object)
	}

	ref := model.ObjectRef{Context: "fake", Kind: "secrets", Namespace: "apps", Name: "db"}
	if _, err := dispatch(engine.OriginGUI, "ui.select", map[string]any{"refs": []model.ObjectRef{ref}}); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool {
		m, err := s.Detail(ref)
		return err == nil && strings.Contains(sectionOf(m, "Data"), engine.MaskedValue)
	}) {
		t.Fatal("the drawer never showed the Secret's keys")
	}
	for range 5 { // the drawer refreshes: no further GET
		if _, err := s.Detail(ref); err != nil {
			t.Fatal(err)
		}
	}
	srv.Set(testkit.SecretsGVR, db) // a watch event on the open Secret
	time.Sleep(200 * time.Millisecond)

	res, err := dispatch(engine.OriginExternalAgent, "object.get", map[string]any{"ref": ref})
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := json.Marshal(res); strings.Contains(string(b), "hunter2") || strings.Contains(string(b), "aHVudGVyMg==") {
		t.Fatalf("object.get for an agent carries the value: %s", b)
	}
	if _, err := dispatch(engine.OriginExternalAgent, "secret.reveal", engine.RevealParams{Ref: ref, Key: "password"}); err == nil {
		t.Fatal("an agent revealed a value")
	}
	v, err := dispatch(engine.OriginGUI, "secret.reveal", engine.RevealParams{Ref: ref, Key: "password"})
	if err != nil || v.(engine.SecretValue).Value != "hunter2" {
		t.Fatalf("GUI reveal: %v, %v", v, err)
	}

	req := srv.Requests()
	if req["get secrets"] != 1 {
		t.Errorf("%d GETs of a Secret for one open drawer, want 1: %v", req["get secrets"], req)
	}
	if req["list secrets"]+req["watch secrets"]+req["watchlist secrets"] != 0 {
		t.Errorf("a full Secrets list or watch was sent: %v", req)
	}
	if req["list secrets (metadata)"]+req["watchlist secrets (metadata)"] == 0 {
		t.Errorf("no metadata-only Secrets request: %v", req)
	}
	t.Logf("requests: %v", req)
}

// TestBudgetHelmReleasesView: a cluster that serves Secrets offers the Helm releases kind; its view
// and drawer are one label-selected watch on the release Secrets, with no GET and no other Secrets
// request; the releases and their history come from it.
func TestBudgetHelmReleasesView(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{
		Resources: append(append([]testkit.FakeResource(nil), testkit.DefaultFakeResources...),
			testkit.FakeResource{GVR: testkit.SecretsGVR, Kind: "Secret", Namespaced: true}),
	})
	defer srv.Close()
	for rev, status := range []string{"superseded", "deployed"} {
		srv.Set(testkit.SecretsGVR, helmSecret(t, "apps", "web", rev+1, status))
	}
	srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", nil, map[string]string{"password": "hunter2"}))

	s := engine.New(engine.Options{
		Connector:        &engine.ClusterConnector{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}},
		SnapshotInterval: 10 * time.Millisecond,
	})
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.Connect(ctx, "fake"); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Backend().Profile("helmreleases"); !ok {
		t.Fatal("no Helm releases kind on a cluster that serves Secrets")
	}
	raw, _ := json.Marshal(map[string]any{"view": "helmreleases"})
	if _, err := s.Dispatch(ctx, engine.Invocation{ID: "ui.openView", Params: raw, Origin: engine.OriginGUI}); err != nil {
		t.Fatal(err)
	}
	if !testkit.WaitFor(ctx, func() bool { ts := s.Table("helmreleases"); return ts != nil && ts.Synced && len(ts.Rows) == 1 }) {
		t.Fatalf("no releases view: %+v", s.Table("helmreleases"))
	}
	ref := model.ObjectRef{Context: "fake", Kind: "helmreleases", Namespace: "apps", Name: "web"}
	m, err := s.DetailWait(ctx, ref)
	if err != nil || m.Missing || !strings.Contains(m.Title, "web") {
		t.Fatalf("drawer %+v, %v", m, err)
	}
	req := srv.Requests()
	if req["list secrets with owner=helm"]+req["watchlist secrets with owner=helm"] != 1 {
		t.Errorf("want one initial request for the release Secrets: %v", req)
	}
	for k, n := range req {
		if strings.Contains(k, "secrets") && !strings.Contains(k, "owner=helm") && n > 0 {
			t.Errorf("another Secrets request: %s ×%d", k, n)
		}
	}
	t.Logf("requests: %v", req)
}

// helmSecret is a release Secret as Helm's Secrets driver writes it.
func helmSecret(t *testing.T, ns, name string, rev int, status string) *unstructured.Unstructured {
	t.Helper()
	rls := map[string]any{
		"name": name, "namespace": ns, "version": rev,
		"info":  map[string]any{"status": status, "description": "revision " + strconv.Itoa(rev)},
		"chart": map[string]any{"metadata": map[string]any{"name": name, "version": "0.1.0"}},
	}
	raw, err := json.Marshal(rls)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	_, _ = zw.Write(raw)
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	payload := base64.StdEncoding.EncodeToString(buf.Bytes())
	s := testkit.Object("v1", "Secret", ns, "sh.helm.release.v1."+name+".v"+strconv.Itoa(rev), map[string]any{
		"type": "helm.sh/release.v1",
		"data": map[string]any{"release": base64.StdEncoding.EncodeToString([]byte(payload))},
	})
	s.SetLabels(map[string]string{"owner": "helm", "name": name, "version": strconv.Itoa(rev), "status": status})
	return s
}

// sectionOf joins the values of a drawer section.
func sectionOf(m *engine.DetailModel, title string) string {
	if m == nil {
		return ""
	}
	var out []string
	for _, sec := range m.Sections {
		if sec.Title == title {
			for _, it := range sec.Items {
				out = append(out, it.Value)
			}
		}
	}
	return strings.Join(out, "\n")
}
