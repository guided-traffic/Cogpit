package cluster

import (
	"context"
	"errors"
	"net/url"
	"slices"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// proxyPassword is a proxy credential planted in the tests; no error may contain it.
const proxyPassword = "pr0xy-p@ss:SECRET-0099"

// A proxy in the options carries every request of the connection: the reachability check, the
// clients' lists, a copy of the configuration; with the user's credentials sent to the proxy, not
// the server. Without it nothing goes through the proxy.
func TestConnectGoesThroughTheProxy(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	path := srv.WriteKubeconfig(t, "fake", "")
	proxy := testkit.NewForwardProxy(t, "alice", proxyPassword)
	server, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{path}, Proxy: proxy.URLWithUser("alice", proxyPassword)})
	if err != nil {
		t.Fatalf("Connect through the proxy: %v", err)
	}
	defer c.Close()
	if n := proxy.Connects()[server.Host]; n < 1 {
		t.Fatalf("the proxy tunnelled %v, want the API server %s", proxy.Connects(), server.Host)
	}
	if users := proxy.Users(); !slices.Contains(users, "alice") {
		t.Errorf("the proxy saw the users %q, want alice", users)
	}
	if _, err := c.Kube().CoreV1().Pods("").List(context.Background(), metav1.ListOptions{}); err != nil {
		t.Fatalf("a list through the proxy: %v", err)
	}
	if srv.Requests()["list pods"] != 1 {
		t.Errorf("the server saw %v", srv.Requests())
	}
	if c.RESTConfig().Proxy == nil {
		t.Error("a copy of the configuration lost the proxy")
	}
	if c.Server() != srv.URL {
		t.Errorf("server %q", c.Server())
	}

	before := proxy.Total()
	direct, err := Connect(context.Background(), Options{KubeconfigPaths: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	defer direct.Close()
	if proxy.Total() != before {
		t.Errorf("a connection without a proxy went through it: %d requests", proxy.Total()-before)
	}
}

// A proxy that refuses the credentials fails the connect with a sentence that names the server,
// never the proxy's password; the kubeconfig's token stays out too.
func TestProxyRefusalNeverQuotesThePassword(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	path := srv.WriteKubeconfig(t, "fake", "")
	proxy := testkit.NewForwardProxy(t, "alice", "the-right-one")

	_, err := Connect(context.Background(), Options{KubeconfigPaths: []string{path}, Proxy: proxy.URLWithUser("alice", proxyPassword)})
	var ce *ConnectError
	if !errors.As(err, &ce) {
		t.Fatalf("err = %v, want a ConnectError", err)
	}
	if proxy.Refused() == 0 {
		t.Error("the proxy refused nothing")
	}
	for _, secret := range []string{proxyPassword, url.UserPassword("", proxyPassword).String()[1:], secretToken} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("the error quotes a secret: %v", err)
		}
	}
	if !strings.Contains(err.Error(), srv.URL) {
		t.Errorf("the error does not name the server: %v", err)
	}
	t.Logf("refused: %v", err)
}

// The guard replaces the password in an error that quotes it, as typed and as escaped in a URL.
func TestScrubProxySecret(t *testing.T) {
	p := &url.URL{Scheme: "http", Host: "proxy:3128", User: url.UserPassword("alice", "a b@c")}
	ce := &ConnectError{Context: "x", Problem: "dial http://alice:a%20b%40c@proxy:3128 and a b@c failed"}
	err := scrubProxySecret(ce, p)
	if strings.Contains(err.Error(), "a b@c") || strings.Contains(err.Error(), "a%20b%40c") || !strings.Contains(err.Error(), "•••") {
		t.Errorf("scrubbed: %v", err)
	}
	if !errors.As(err, new(*ConnectError)) {
		t.Error("the scrubbed error is no ConnectError")
	}
	if got := scrubProxySecret(errors.New("plain a b@c"), p); got.Error() != "plain •••" {
		t.Errorf("plain error: %v", got)
	}
	if got := scrubProxySecret(ce, &url.URL{Scheme: "http", Host: "proxy"}); got != error(ce) {
		t.Error("an error without a password to scrub changed")
	}
}
