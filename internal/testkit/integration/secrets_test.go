package integration_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
