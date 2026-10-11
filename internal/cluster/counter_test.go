package cluster

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		method, url  string
		verb, object string
	}{
		// Discovery.
		{"GET", "/api", VerbDiscovery, ""},
		{"GET", "/api/v1", VerbDiscovery, ""},
		{"GET", "/apis", VerbDiscovery, ""},
		{"GET", "/apis/apps", VerbDiscovery, ""},
		{"GET", "/apis/apps/v1", VerbDiscovery, ""},
		{"GET", "/version", VerbDiscovery, ""},
		{"GET", "/version/", VerbDiscovery, ""},
		{"GET", "/openapi/v2", VerbDiscovery, ""},
		{"GET", "/openapi/v3/apis/apps/v1", VerbDiscovery, ""},
		// Lists and gets.
		{"GET", "/api/v1/pods?limit=500&resourceVersion=0", VerbList, "pods"},
		{"GET", "/api/v1/namespaces/team-a/pods", VerbList, "pods"},
		{"GET", "/api/v1/namespaces/team-a/pods/web-0", VerbGet, "pods"},
		{"GET", "/api/v1/namespaces", VerbList, "namespaces"},
		{"GET", "/api/v1/namespaces/team-a", VerbGet, "namespaces"},
		{"GET", "/api/v1/nodes/node-1", VerbGet, "nodes"},
		{"GET", "/apis/apps/v1/deployments", VerbList, "deployments.apps"},
		{"GET", "/apis/apps/v1/namespaces/team-a/deployments/web", VerbGet, "deployments.apps"},
		{"GET", "/apis/example.com/v1alpha1/widgets", VerbList, "widgets.example.com"},
		{"HEAD", "/api/v1/namespaces/a/pods/p", VerbGet, "pods"},
		// Subresources.
		{"GET", "/api/v1/namespaces/a/pods/p/log?follow=true", VerbGet, "pods/log"},
		{"POST", "/api/v1/namespaces/a/pods/p/exec?command=sh", VerbCreate, "pods/exec"},
		{"GET", "/api/v1/namespaces/a/pods/p/exec?command=sh", VerbGet, "pods/exec"},
		{"PATCH", "/apis/apps/v1/namespaces/a/deployments/web/scale", VerbPatch, "deployments.apps/scale"},
		{"PUT", "/api/v1/namespaces/team-a/finalize", VerbUpdate, "namespaces/finalize"},
		{"GET", "/api/v1/namespaces/team-a/status", VerbGet, "namespaces/status"},
		// Watches.
		{"GET", "/api/v1/pods?watch=true&resourceVersion=12", VerbWatch, "pods"},
		{"GET", "/api/v1/namespaces/a/pods?watch=1", VerbWatch, "pods"},
		{"GET", "/api/v1/pods?watch=false", VerbList, "pods"},
		{"GET", "/api/v1/watch/pods", VerbWatch, "pods"},
		{"GET", "/api/v1/watch/namespaces/a/pods", VerbWatch, "pods"},
		{"GET", "/apis/apps/v1/deployments?allowWatchBookmarks=true&resourceVersionMatch=NotOlderThan&sendInitialEvents=true&watch=true", VerbWatchList, "deployments.apps"},
		{"GET", "/api/v1/pods?sendInitialEvents=true", VerbList, "pods"}, // not a watch: a (rejected) list
		// Writes.
		{"POST", "/api/v1/namespaces/a/pods", VerbCreate, "pods"},
		{"PUT", "/api/v1/namespaces/a/configmaps/c", VerbUpdate, "configmaps"},
		{"PATCH", "/apis/apps/v1/namespaces/a/deployments/web", VerbPatch, "deployments.apps"},
		{"DELETE", "/api/v1/namespaces/a/pods/p", VerbDelete, "pods"},
		{"DELETE", "/api/v1/namespaces/a/pods", VerbDeleteCollection, "pods"},
		{"OPTIONS", "/api/v1/pods", VerbOther, "pods"},
		// Other.
		{"GET", "/", VerbOther, ""},
		{"GET", "/healthz", VerbOther, ""},
		{"GET", "/livez/etcd", VerbOther, ""},
		{"GET", "/api/v1/watch", VerbOther, ""},
	}
	for _, tt := range tests {
		u, err := url.Parse(tt.url)
		if err != nil {
			t.Fatal(err)
		}
		verb, object := Classify(tt.method, u)
		if verb != tt.verb || object != tt.object {
			t.Errorf("Classify(%s %s) = %q %q, want %q %q", tt.method, tt.url, verb, object, tt.verb, tt.object)
		}
	}
	if verb, object := Classify("GET", nil); verb != VerbOther || object != "" {
		t.Errorf("Classify(nil) = %q %q", verb, object)
	}
}

func TestCounterRecordsThroughTheTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer srv.Close()
	rc := NewRequestCounter()
	client := &http.Client{Transport: rc.Wrap(http.DefaultTransport)}
	for _, p := range []string{"/api/v1/pods", "/api/v1/pods", "/api/v1/pods?watch=true", "/version"} {
		resp, err := client.Get(srv.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		_ = resp.Body.Close()
	}
	st := rc.Snapshot()
	if st.Total != 4 || st.ByVerb[VerbList] != 2 || st.ByVerb[VerbWatch] != 1 || st.ByVerb[VerbDiscovery] != 1 {
		t.Errorf("by verb %+v total %d", st.ByVerb, st.Total)
	}
	if st.ByVerbResource["list pods"] != 2 || st.ByVerbResource["watch pods"] != 1 || st.ByVerbResource["discovery"] != 1 {
		t.Errorf("by verb and resource %+v", st.ByVerbResource)
	}
	// Snapshot returns copies.
	st.ByVerb[VerbList] = 100
	if rc.Snapshot().ByVerb[VerbList] != 2 {
		t.Error("Snapshot shares its map with the counter")
	}
	client.CloseIdleConnections() // reaches the wrapped transport without panicking
}

func TestCounterStripsTheServerPathPrefix(t *testing.T) {
	rc := NewRequestCounter()
	rc.prefix = serverPathPrefix("https://proxy.example/k8s/clusters/c-1/")
	u, _ := url.Parse("https://proxy.example/k8s/clusters/c-1/api/v1/namespaces/a/pods")
	rc.Record("GET", u)
	u, _ = url.Parse("https://proxy.example/other/api/v1/pods")
	rc.Record("GET", u)
	st := rc.Snapshot()
	if st.ByVerbResource["list pods"] != 1 || st.ByVerb[VerbOther] != 1 {
		t.Errorf("prefix not stripped exactly: %+v", st.ByVerbResource)
	}
}

func TestCounterIsConcurrencySafe(t *testing.T) {
	rc := NewRequestCounter()
	u, _ := url.Parse("/api/v1/pods")
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 1000 {
				rc.Record("GET", u)
				_ = rc.Snapshot()
			}
		})
	}
	wg.Wait()
	if got := rc.Snapshot().ByVerbResource["list pods"]; got != 8000 {
		t.Errorf("counted %d, want 8000", got)
	}
}
