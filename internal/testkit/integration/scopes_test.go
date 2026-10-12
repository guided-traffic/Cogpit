package integration_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestBudgetFindWithoutKindStartsNoReleaseWatch: an agent's resources.find without a kind searches
// the kinds that hold Secret values (the Secrets, Helm's releases) only when the cache holds them
// already: it must not start a cluster-wide watch of every Secret, values and all, as a side effect
// of a search by name. Once the releases are cached (their view was open), the same search finds
// them.
func TestBudgetFindWithoutKindStartsNoReleaseWatch(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{
		Resources: append(append([]testkit.FakeResource(nil), testkit.DefaultFakeResources...), testkit.FakeResource{GVR: testkit.SecretsGVR, Kind: "Secret", Namespaced: true}),
	})
	defer srv.Close()
	srv.Set(testkit.SecretsGVR, helmSecret(t, "apps", "web", 1, "deployed"))
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
	find := func() commands.FindResult {
		t.Helper()
		raw, _ := json.Marshal(map[string]any{"field": "name", "value": "web"})
		res, err := s.Dispatch(ctx, engine.Invocation{ID: "resources.find", Params: raw, Origin: engine.OriginExternalAgent})
		if err != nil {
			t.Fatal(err)
		}
		return res.(commands.FindResult)
	}
	secretRequests := func() int64 {
		req := srv.Requests()
		return req["list secrets with owner=helm"] + req["watchlist secrets with owner=helm"] +
			req["list secrets"] + req["watchlist secrets"] + req["watch secrets"]
	}

	for _, m := range find().Matches {
		if m.Ref.Kind == helm.ReleaseKind || m.Ref.Kind == "secrets" {
			t.Errorf("found an object of a kind that was not cached: %+v", m)
		}
	}
	if n := secretRequests(); n != 0 {
		t.Fatalf("resources.find without a kind started a Secrets watch: %v", srv.Requests())
	}

	if _, err := s.Objects(ctx, helm.ReleaseKind, nil); err != nil { // the Releases view opens
		t.Fatal(err)
	}
	found := false
	for _, m := range find().Matches {
		found = found || (m.Ref.Kind == helm.ReleaseKind && m.Ref.Name == "web")
	}
	if !found {
		t.Error("a search without a kind misses the cached releases")
	}
	req := srv.Requests()
	if n := req["list secrets with owner=helm"] + req["watchlist secrets with owner=helm"]; n != 1 || secretRequests() != 1 {
		t.Errorf("want the one initial request of the Releases view: %v", req)
	}
}

// TestBudgetSecretScopes proves the covering scopes of architecture §5 at the HTTP level, for a
// server that streams initial lists and for one that does not: the owner=helm scope alone is one
// label-selected request; once the full Secrets scope runs and has synced, the owner=helm scope
// reads from it and its own watch closes; when the full scope's warm period ends while an
// owner=helm handle is open, the owner=helm scope gets a watch of its own again and the full watch
// closes. A metadata-only scope is never covered by a full one: it asks for PartialObjectMetadata,
// so no value crosses the wire for it.
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
			c := cache.New(cache.DynamicSource(conn.Dynamic(), conn.Metadata()), cache.Options{WarmPeriod: 200 * time.Millisecond})
			defer c.Close()
			// initial counts the initial lists of a resource: streaming watches, or, from a server
			// that refuses them (client-go asks once per informer, then lists), lists.
			initial := func(resource string) int64 {
				if tt.watchList {
					return srv.Requests()["watchlist "+resource]
				}
				return srv.Requests()["list "+resource]
			}
			wait := func(what string, cond func() bool) {
				t.Helper()
				if !testkit.WaitFor(ctx, cond) {
					t.Fatalf("%s: requests %v, open watches %v, scopes %+v", what, srv.Requests(), srv.OpenWatchesBy(), c.Stats())
				}
			}

			// The owner=helm scope alone: one label-selected request, its own watch.
			helmScope := cache.Scope{GVR: testkit.SecretsGVR, LabelSelector: "owner=helm"}
			releases := c.Acquire(helmScope)
			defer releases.Release()
			if !releases.WaitForSync(ctx) {
				t.Fatalf("owner=helm did not sync: %v", releases.Err())
			}
			if l := releases.List(); len(l) != 1 || l[0].GetName() != "sh.helm.release.v1.web.v1" || l[0].Object["data"] == nil {
				t.Fatalf("owner=helm scope: %v", l)
			}
			wait("the owner=helm watch", func() bool { return srv.OpenWatchesBy()["secrets with owner=helm"] == 1 })
			if initial("secrets with owner=helm") != 1 || initial("secrets") != 0 {
				t.Fatalf("the owner=helm scope alone: %v", srv.Requests())
			}

			// The full scope starts: the owner=helm scope moves onto it and its watch closes.
			all := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR})
			if !all.WaitForSync(ctx) {
				t.Fatalf("the full scope did not sync: %v", all.Err())
			}
			wait("the move onto the full scope", func() bool {
				open := srv.OpenWatchesBy()
				return open["secrets with owner=helm"] == 0 && open["secrets"] == 1
			})
			if initial("secrets") != 1 || initial("secrets with owner=helm") != 1 {
				t.Errorf("one request per scope expected: %v", srv.Requests())
			}
			if len(all.List()) != 2 || len(releases.List()) != 1 {
				t.Errorf("full scope %d objects, owner=helm %d", len(all.List()), len(releases.List()))
			}

			// A watch event reaches both, each as its scope says.
			srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "sh.helm.release.v1.web.v2", map[string]string{"owner": "helm", "name": "web"}, map[string]string{"release": "p2"}))
			srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "tls", nil, map[string]string{"tls.key": "secret-key"}))
			wait("the events", func() bool { return len(all.List()) == 4 && len(releases.List()) == 2 })

			// A metadata-only scope is not covered by the full one: its own request, no data.
			meta := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, MetadataOnly: true})
			if !meta.WaitForSync(ctx) {
				t.Fatalf("metadata scope did not sync: %v", meta.Err())
			}
			for _, o := range meta.List() {
				if _, has := o.Object["data"]; has || !cache.IsMetadataObject(o) || o.GetUID() == "" {
					t.Errorf("metadata scope holds %v", o.Object)
				}
			}
			meta.Release()
			if initial("secrets (metadata)") != 1 {
				t.Errorf("the metadata scope: %v", srv.Requests())
			}

			// The full scope's warm period ends while the owner=helm handle is open: the owner=helm
			// scope lists on its own again, and the full watch closes.
			all.Release()
			wait("the handover back", func() bool {
				open := srv.OpenWatchesBy()
				return open["secrets"] == 0 && open["secrets with owner=helm"] == 1 && releases.HasSynced()
			})
			if initial("secrets with owner=helm") != 2 || initial("secrets") != 1 {
				t.Errorf("after the handover: %v", srv.Requests())
			}
			srv.Set(testkit.SecretsGVR, testkit.Secret("apps", "sh.helm.release.v1.web.v3", map[string]string{"owner": "helm", "name": "web"}, map[string]string{"release": "p3"}))
			wait("an event through the own watch", func() bool { return len(releases.List()) == 3 })
			t.Logf("requests: %v", srv.Requests())
		})
	}
}
