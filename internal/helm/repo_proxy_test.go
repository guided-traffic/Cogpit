package helm

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// A fetcher with a Proxy downloads the index and the chart archive through it (a CONNECT tunnel to
// the repository, the proxy's user sent to the proxy); the Proxy is asked per request, so a proxy
// that changes takes the next download; without one nothing goes through the proxy.
func TestDownloadsGoThroughTheProxy(t *testing.T) {
	r, f := testRepo(t)
	proxy := testkit.NewForwardProxy(t, "alice", "pw-for-the-proxy")
	var use *url.URL
	f.Proxy = func(*http.Request) (*url.URL, error) { return use, nil }
	repo, err := url.Parse(r.URL)
	if err != nil {
		t.Fatal(err)
	}

	use = proxy.URLWithUser("alice", "pw-for-the-proxy")
	idx, err := f.DownloadIndex(context.Background(), r.URL, filepath.Join(t.TempDir(), "web-index.yaml"))
	if err != nil {
		t.Fatalf("index through the proxy: %v", err)
	}
	web, _ := idx.Chart("web")
	latest, _ := web.Latest()
	if _, err := f.FetchChart(context.Background(), "test", r.URL, "web", latest); err != nil {
		t.Fatalf("chart through the proxy: %v", err)
	}
	if proxy.Connects()[repo.Host] < 1 || !slices.Contains(proxy.Users(), "alice") {
		t.Fatalf("the proxy tunnelled %v for %q", proxy.Connects(), proxy.Users())
	}
	if r.Requests("/index.yaml") != 1 {
		t.Errorf("the repository served the index %d times", r.Requests("/index.yaml"))
	}

	// The Proxy says none now: the next download goes straight to the repository.
	use = nil
	f.Client.CloseIdleConnections()
	if tr, ok := f.proxiedBase.(*http.Transport); ok {
		tr.CloseIdleConnections() // the tunnel stays open otherwise and carries the next request
	}
	before := proxy.Total()
	if _, err := f.DownloadIndex(context.Background(), r.URL, filepath.Join(t.TempDir(), "again.yaml")); err != nil {
		t.Fatal(err)
	}
	if proxy.Total() != before {
		t.Errorf("%d requests through the proxy after it was cleared", proxy.Total()-before)
	}

	// A proxy that refuses the credentials fails the download; the error does not quote them.
	tr, _ := f.proxiedBase.(*http.Transport)
	tr.CloseIdleConnections()
	use = proxy.URLWithUser("alice", "wrong-secret-77")
	_, err = f.DownloadIndex(context.Background(), r.URL, filepath.Join(t.TempDir(), "refused.yaml"))
	if err == nil || strings.Contains(err.Error(), "wrong-secret-77") {
		t.Errorf("refused download: %v", err)
	}

	// A Proxy that fails stops the download with its error.
	tr.CloseIdleConnections()
	f.Proxy = func(*http.Request) (*url.URL, error) { return nil, errors.New("no proxy today") }
	if _, err := f.DownloadIndex(context.Background(), r.URL, filepath.Join(t.TempDir(), "x.yaml")); err == nil || !strings.Contains(err.Error(), "no proxy today") {
		t.Errorf("a failing Proxy: %v", err)
	}
}
