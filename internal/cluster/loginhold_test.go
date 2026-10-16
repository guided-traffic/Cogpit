package cluster

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// countingPlugin is a credential plugin that notes each run in a file and hands out the
// ExecCredential the test wrote: what the plugin would do, without a browser.
type countingPlugin struct {
	t     *testing.T
	runs  string // one line per run
	cred  string // the ExecCredential it prints
	path  string // the plugin
	kconf string // a kubeconfig of context "held" running it against the fake API server
}

func newCountingPlugin(t *testing.T, srv *testkit.FakeAPIServer) *countingPlugin {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the test plugin is a shell script")
	}
	dir := t.TempDir()
	p := &countingPlugin{t: t, runs: filepath.Join(dir, "runs"), cred: filepath.Join(dir, "cred.json"), path: filepath.Join(dir, "login-plugin")}
	script := "#!/bin/sh\necho run >> \"$PLUGIN_RUNS\"\ncat \"$PLUGIN_CRED\"\n"
	if err := os.WriteFile(p.path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	p.kconf = writeFile(t, "kubeconfig", `apiVersion: v1
kind: Config
current-context: held
clusters:
- name: fake
  cluster:
    server: `+srv.URL+`
    certificate-authority-data: `+base64.StdEncoding.EncodeToString(srv.CAData())+`
users:
- name: plugin
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: `+p.path+`
      interactiveMode: Never
      env:
      - name: PLUGIN_RUNS
        value: `+p.runs+`
      - name: PLUGIN_CRED
        value: `+p.cred+`
contexts:
- name: held
  context: {cluster: fake, user: plugin}
`)
	return p
}

// give makes the plugin hand out token, expired already when expired is set (every request then
// needs a run), else without an expiry.
func (p *countingPlugin) give(token string, expired bool) {
	p.t.Helper()
	status := `"token":"` + token + `"`
	if expired {
		status += `,"expirationTimestamp":"2000-01-01T00:00:00Z"`
	}
	cred := `{"apiVersion":"client.authentication.k8s.io/v1","kind":"ExecCredential","status":{` + status + `}}`
	if err := os.WriteFile(p.cred, []byte(cred), 0o600); err != nil {
		p.t.Fatal(err)
	}
}

// count is how often the plugin ran.
func (p *countingPlugin) count() int {
	data, err := os.ReadFile(p.runs)
	if errors.Is(err, os.ErrNotExist) {
		return 0
	}
	if err != nil {
		p.t.Fatal(err)
	}
	return strings.Count(string(data), "run\n")
}

func (p *countingPlugin) connect() *Conn {
	p.t.Helper()
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{p.kconf}, ConnectTimeout: 10 * time.Second})
	if err != nil {
		p.t.Fatal(err)
	}
	p.t.Cleanup(c.Close)
	return c
}

// TestHeldPluginDoesNotRun is D-69 item 4's hold on client-go's own path: a login that expired by
// its time while the connection holds it starts no plugin process, the heartbeat and every other
// request end as ErrLoginLapsed; let go, the plugin runs once and the connection works.
func TestHeldPluginDoesNotRun(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newCountingPlugin(t, srv)
	p.give(secretToken, true)
	c := p.connect()
	if n := p.count(); n != 1 {
		t.Fatalf("the connect ran the plugin %d times, want once", n)
	}
	if !c.LoginHoldable() {
		t.Fatal("the plugin cannot be held: client-go's Authenticator changed (loginhold.go)")
	}

	c.HoldLogin(true)
	err := c.Heartbeat(context.Background())
	if !errors.Is(err, ErrLoginLapsed) {
		t.Fatalf("the held heartbeat: %v, want ErrLoginLapsed", err)
	}
	if _, err := c.Kube().CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{}); err == nil {
		t.Error("a list worked with the login held and expired")
	}
	if n := p.count(); n != 1 {
		t.Fatalf("the plugin ran %d times while held, want no run after the connect's", n-1)
	}

	c.HoldLogin(false)
	if err := c.Heartbeat(context.Background()); err != nil {
		t.Fatalf("the heartbeat after the hold: %v", err)
	}
	if n := p.count(); n != 2 {
		t.Errorf("the plugin ran %d times after the hold, want once", n-1)
	}
}

// A held connection's 401 (a revoked token that has not expired by its time) is not handed to
// the plugin: ErrLoginLapsed, no run. Let go and armed, the next 401 renews the login once, as
// D-58's renewal does.
func TestHeldLoginKeepsA401FromThePlugin(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: "first-token-0042"})
	defer srv.Close()
	p := newCountingPlugin(t, srv)
	p.give("first-token-0042", false)
	c := p.connect()
	c.HoldLogin(true)
	srv.SetToken("second-token-0042")
	if err := c.Heartbeat(context.Background()); !errors.Is(err, ErrLoginLapsed) {
		t.Fatalf("a held 401: %v, want ErrLoginLapsed", err)
	}
	if n := p.count(); n != 1 {
		t.Fatalf("the plugin ran %d times for a held 401", n-1)
	}

	p.give("second-token-0042", false)
	c.HoldLogin(false)
	c.ArmLogin()
	if err := c.Heartbeat(context.Background()); !errors.Is(err, ErrLoginRejected) {
		t.Fatalf("the first heartbeat after the hold: %v, want the 401 the plugin renews", err)
	}
	if err := c.Heartbeat(context.Background()); err != nil {
		t.Fatalf("the renewed login: %v", err)
	}
	if n := p.count(); n != 2 {
		t.Errorf("the plugin ran %d times to renew, want once", n-1)
	}
}

// Two connections whose plugin configuration is the same share client-go's Authenticator: the
// plugin is held only while both hold it, so the one that does not renews the login for both.
func TestSharedLoginIsHeldOnlyWhenEveryConnectionHoldsIt(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	p := newCountingPlugin(t, srv)
	p.give(secretToken, true)
	held, free := p.connect(), p.connect()
	base := p.count()
	held.HoldLogin(true)
	if err := held.Heartbeat(context.Background()); err != nil {
		t.Fatalf("the held connection's heartbeat while the other may renew: %v", err)
	}
	if p.count() != base+1 {
		t.Fatalf("the shared login was not renewed: %d runs", p.count()-base)
	}
	free.HoldLogin(true)
	if err := free.Heartbeat(context.Background()); !errors.Is(err, ErrLoginLapsed) {
		t.Fatalf("with both held: %v, want ErrLoginLapsed", err)
	}
	if p.count() != base+1 {
		t.Errorf("the plugin ran with both connections holding it")
	}
	free.Close() // a closed connection frees nothing and holds nothing: the other still holds it
	if err := held.Heartbeat(context.Background()); !errors.Is(err, ErrLoginLapsed) {
		t.Fatalf("after the other closed: %v, want ErrLoginLapsed", err)
	}
	if p.count() != base+1 {
		t.Errorf("the plugin ran after the free connection closed")
	}
}

// A connection without a credential plugin has nothing to hold: its rejected login stays a
// rejected one (LoginExpired, red), not a lapsed one.
func TestHoldWithoutAPluginChangesNothing(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{Token: secretToken})
	defer srv.Close()
	path := writeFile(t, "kubeconfig", string(srv.Kubeconfig("token", "")))
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	c.HoldLogin(true)
	if c.LoginHoldable() {
		t.Error("a token login is holdable")
	}
	srv.SetToken("another-token-0042")
	err = c.Heartbeat(context.Background())
	if errors.Is(err, ErrLoginLapsed) || !errors.Is(err, ErrLoginRejected) {
		t.Errorf("a rejected token with the hold: %v, want ErrLoginRejected", err)
	}
}
