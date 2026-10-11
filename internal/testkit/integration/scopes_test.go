package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestBudgetSecretScopes proves the Secrets contract of D-13 (H-8, option b) at the HTTP level:
// the metadata-only scope asks the server for PartialObjectMetadata, so no Secret value crosses
// the wire or reaches the cache; the label-selected scope for Helm's releases asks the server for
// the owner=helm Secrets only; each scope is one informer with its own initial list and watch,
// for a server that streams initial lists and for one that does not.
func TestBudgetSecretScopes(t *testing.T) {
	for _, tt := range []struct {
		name      string
		watchList bool
	}{{"watchlist", true}, {"list and watch", false}} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{
				DisableWatchList: !tt.watchList,
				Resources:        append(append([]testkit.FakeResource(nil), testkit.DefaultFakeResources...), testkit.FakeResource{GVR: testkit.SecretsGVR, Kind: "Secret", Namespaced: true}),
			})
			defer srv.Close()
			srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", map[string]string{"app": "db"}, map[string]string{"password": "hunter2"}))
			srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "sh.helm.release.v1.web.v1", map[string]string{"owner": "helm", "name": "web"}, map[string]string{"release": "payload"}))

			conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{srv.WriteKubeconfig(t, "fake", "")}})
			if err != nil {
				t.Fatalf("Connect: %v", err)
			}
			defer conn.Close()
			c := cache.New(cache.DynamicSource(conn.Dynamic(), conn.Metadata()), cache.Options{WarmPeriod: time.Minute})
			defer c.Close()

			meta := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, MetadataOnly: true})
			defer meta.Release()
			helm := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm"})
			defer helm.Release()
			helm2 := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "owner = helm"})
			defer helm2.Release()
			for _, h := range []*cache.Handle{meta, helm, helm2} {
				if !h.WaitForSync(ctx) {
					t.Fatalf("%s did not sync: %v", h.Scope(), h.Err())
				}
			}

			if got := len(meta.List()); got != 2 {
				t.Fatalf("metadata scope: %d objects", got)
			}
			for _, o := range meta.List() {
				if _, has := o.Object["data"]; has || !cache.IsMetadataObject(o) || o.GetUID() == "" {
					t.Errorf("metadata scope holds %v", o.Object)
				}
			}
			if l := helm.List(); len(l) != 1 || l[0].GetName() != "sh.helm.release.v1.web.v1" {
				t.Fatalf("owner=helm scope: %d objects", len(l))
			} else if _, has := l[0].Object["data"]; !has {
				t.Error("owner=helm scope lost the release payload")
			}

			// A watch event reaches each scope as its scope says.
			srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "tls", nil, map[string]string{"tls.key": "secret-key"}))
			if !testkit.WaitFor(ctx, func() bool { return len(meta.List()) == 3 }) {
				t.Fatal("the metadata scope missed a watch event")
			}
			for _, o := range meta.List() {
				if _, has := o.Object["data"]; has {
					t.Errorf("%s arrived with its data", o.GetName())
				}
			}
			if len(helm.List()) != 1 {
				t.Errorf("an unlabelled Secret reached the owner=helm scope")
			}

			req := srv.Requests()
			metaInitial := req["watchlist secrets (metadata)"] + req["list secrets (metadata)"]
			helmInitial := req["watchlist secrets with owner=helm"] + req["list secrets with owner=helm"]
			if metaInitial == 0 || helmInitial == 0 {
				t.Fatalf("requests: %v", req)
			}
			if req["list secrets"]+req["watch secrets"]+req["watchlist secrets"] != 0 {
				t.Errorf("a full, unselected Secrets request was sent: %v", req)
			}
			if !tt.watchList && (req["list secrets (metadata)"] != 1 || req["list secrets with owner=helm"] != 1) {
				t.Errorf("one list per scope expected (the two owner=helm handles share one): %v", req)
			}
			if tt.watchList && (req["watchlist secrets (metadata)"] != 1 || req["watchlist secrets with owner=helm"] != 1) {
				t.Errorf("one streaming watch per scope expected: %v", req)
			}
		})
	}
}
