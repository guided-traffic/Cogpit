package engine

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// proxySecret is a proxy password planted in the tests; nothing but the settings file may hold it.
const proxySecret = "s3cr3t-PROXY-pw-77"

// The values a proxy may take, and the refusals of those it may not, none of which quotes the value.
func TestCheckProxyURL(t *testing.T) {
	for _, ok := range []string{"", "http://proxy:3128", "https://proxy.example.com", "socks5://127.0.0.1:1080",
		"HTTP://Proxy:8080/", "http://alice:" + proxySecret + "@proxy:3128", "http://[::1]:3128"} {
		if err := CheckProxyURL(ok); err != nil {
			t.Errorf("%q refused: %v", ok, err)
		}
	}
	for _, bad := range []string{"proxy:3128", "ftp://proxy", "http://", "http://:3128", "http://proxy:0", "http://proxy:70000",
		"http://proxy/path", "http://proxy?q=1", "http://proxy#x", "http://pro xy", "http://alice:" + proxySecret + "@proxy:99999",
		"http://alice:" + proxySecret + "@/nohost", "socks4://proxy", "http://alice:" + proxySecret + "@proxy\n:1", strings.Repeat("x", 3000)} {
		err := CheckProxyURL(bad)
		if err == nil {
			t.Errorf("%q accepted", bad)
			continue
		}
		if strings.Contains(err.Error(), proxySecret) || len(bad) > 3 && strings.Contains(err.Error(), bad) {
			t.Errorf("the refusal of %q quotes it: %v", bad, err)
		}
	}
}

// A password is shown as ••• with the user kept; a value shown that way keeps the password set
// before; one without a password before is refused.
func TestRedactAndMergeProxyPassword(t *testing.T) {
	raw := "http://alice:" + proxySecret + "@proxy.example.com:3128"
	shown := RedactProxyURL(raw)
	if shown != "http://alice:•••@proxy.example.com:3128" {
		t.Fatalf("shown %q", shown)
	}
	if RedactProxyURL("http://proxy:3128") != "http://proxy:3128" || RedactProxyURL("http://bob@proxy") != "http://bob@proxy" {
		t.Error("a URL without a password changed")
	}
	if got := RedactProxyURL("http://alice:" + proxySecret + "@proxy:99999"); strings.Contains(got, proxySecret) {
		t.Errorf("an invalid URL shown as %q", got)
	}
	next, err := MergeProxyPassword(raw, "http://alice:•••@other.example.com:8080")
	if err != nil || next != "http://alice:"+proxySecret+"@other.example.com:8080" {
		t.Errorf("merged %q, %v", next, err)
	}
	escaped := "http://alice:a%20b%40c@proxy:1"
	if next, err := MergeProxyPassword(escaped, "socks5://alice:•••@proxy:1080"); err != nil || next != "socks5://alice:a%20b%40c@proxy:1080" {
		t.Errorf("escaped password merged as %q, %v", next, err)
	}
	if _, err := MergeProxyPassword("http://proxy:1", "http://alice:•••@proxy:1"); err == nil {
		t.Error("a redacted password without one before accepted")
	}
	if next, err := MergeProxyPassword(raw, ""); err != nil || next != "" {
		t.Errorf("clearing: %q, %v", next, err)
	}
}

// The proxy is kept as typed in the owner-only settings file only: the UI state, its JSON (what
// ui_state hands an agent) and the settings view show it redacted; handing the redacted form back
// changes nothing.
func TestClusterProxyKeptPrivately(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	s := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	raw := "http://alice:" + proxySecret + "@proxy:3128"
	if _, err := s.ChangeClusterSettings(devCtx, func(cs *ClusterSettings) error { cs.Proxy = raw; return nil }); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, ClusterSettingsFile)
	data, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(data), proxySecret) {
		t.Fatalf("the file does not keep the proxy: %s, %v", data, err)
	}
	if fi, _ := os.Stat(file); runtime.GOOS != goosWindows && fi.Mode().Perm() != 0o600 {
		t.Errorf("the file is %v", fi.Mode().Perm())
	}
	u := s.UIState()
	if got := u.ClusterSettings[devCtx].Proxy; got != "http://alice:•••@proxy:3128" {
		t.Errorf("ui state shows %q", got)
	}
	b, _ := json.Marshal(u)
	v, _ := s.ClusterSettingsView(devCtx)
	vb, _ := json.Marshal(v)
	for what, out := range map[string][]byte{"ui state": b, "view": vb} {
		if strings.Contains(string(out), proxySecret) {
			t.Errorf("the %s carries the password: %s", what, out)
		}
	}
	if v.Proxy != "http://alice:•••@proxy:3128" {
		t.Errorf("view %q", v.Proxy)
	}
	if _, err := s.ChangeClusterSettings(devCtx, func(cs *ClusterSettings) error { cs.Proxy = v.Proxy; return nil }); err != nil {
		t.Fatal(err)
	}
	if cs, _ := s.ClusterSettingsOf(devCtx); cs.Proxy != raw {
		t.Errorf("the redacted form handed back replaced the proxy: %q", cs.Proxy)
	}
	if _, err := s.ChangeClusterSettings(devCtx, func(cs *ClusterSettings) error { cs.Proxy = "ftp://alice:" + proxySecret + "@x"; return nil }); err == nil ||
		strings.Contains(err.Error(), proxySecret) {
		t.Errorf("a bad proxy: %v", err)
	}
	again := New(Options{ClusterSettings: NewClusterSettingsStore(dir)})
	if got := again.UIState().ClusterSettings[devCtx].Proxy; got != "http://alice:•••@proxy:3128" {
		t.Errorf("read back into the UI state as %q", got)
	}
}

// The Preferences' HTTP proxy carries the Helm repositories' downloads: kept as typed in the
// owner-only preferences file, shown redacted in the UI state; without it the environment's proxy
// decides (stubbed here: tests never read the real environment's proxy variables).
func TestNonClusterProxyCarriesHelmDownloads(t *testing.T) {
	envCalls := 0
	defer func(old func(*http.Request) (*url.URL, error)) { envProxy = old }(envProxy)
	envProxy = func(*http.Request) (*url.URL, error) { envCalls++; return nil, nil }

	repo := testkit.NewHelmRepo(t, testkit.TestChart{Name: "web", Version: "0.1.0",
		Templates: map[string]string{"templates/cm.yaml": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n"}})
	proxy := testkit.NewForwardProxy(t, "bob", proxySecret)
	cfg := filepath.Join(t.TempDir(), "Cogpit")
	helmFile := testkit.NewHelmRepoFile(t)
	helmFile.Write(repo.Entry("plain"), repo.Entry("proxied"))
	repos := NewHelmRepos(helmFile.Path, t.TempDir(), WithChartFetcher(&helm.Fetcher{Client: repo.Client()}))
	s := New(Options{Preferences: NewPreferencesStore(cfg), HelmRepos: repos})
	t.Cleanup(s.Close)

	// No preference: the environment's proxy is asked, and nothing goes through the test proxy.
	if _, err := repos.Update(context.Background(), []string{"plain"}); err != nil {
		t.Fatal(err)
	}
	if envCalls == 0 || proxy.Total() != 0 {
		t.Fatalf("without the preference: %d environment lookups, %d proxy requests", envCalls, proxy.Total())
	}

	withUser := proxy.URLWithUser("bob", proxySecret).String()
	if _, err := s.ChangePreferences(func(p *Preferences) error { p.HTTPProxy = withUser; return nil }); err != nil {
		t.Fatal(err)
	}
	if got := s.UIState().Preferences.HTTPProxy; got != "http://bob:•••@"+proxy.URL.Host {
		t.Errorf("ui state shows %q", got)
	}
	raw, _ := os.ReadFile(filepath.Join(cfg, PreferencesFile))
	if !strings.Contains(string(raw), proxySecret) {
		t.Errorf("the preferences file does not keep the proxy: %s", raw)
	}
	if b, _ := json.Marshal(s.UIState()); strings.Contains(string(b), proxySecret) {
		t.Error("the UI state carries the password")
	}
	before := envCalls
	if _, err := repos.Update(context.Background(), []string{"proxied"}); err != nil {
		t.Fatalf("index through the proxy: %v", err)
	}
	if proxy.Total() == 0 || !slices.Contains(proxy.Users(), "bob") || envCalls != before {
		t.Errorf("with the preference: %d proxy requests from %q, %d more environment lookups", proxy.Total(), proxy.Users(), envCalls-before)
	}
	if _, _, err := repos.FetchChart(context.Background(), "proxied", "web", "0.1.0"); err != nil {
		t.Errorf("chart through the proxy: %v", err)
	}

	// Handing back the redacted form keeps it; a bad one is refused without quoting it.
	if _, err := s.ChangePreferences(func(p *Preferences) error { p.HTTPProxy = s.UIState().Preferences.HTTPProxy; return nil }); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.Preferences(); p.HTTPProxy != withUser {
		t.Errorf("after the redacted form came back: %q", p.HTTPProxy)
	}
	if _, err := s.ChangePreferences(func(p *Preferences) error { p.HTTPProxy = "gopher://bob:" + proxySecret + "@x"; return nil }); err == nil || strings.Contains(err.Error(), proxySecret) {
		t.Errorf("a bad proxy: %v", err)
	}
}

// A connect goes through the context's proxy from Cogpit's settings: the proxy tunnels to the API
// server, the connection log names it without its password; without the setting the connection
// goes straight to the server.
func TestConnectThroughTheClusterProxy(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{})
	defer srv.Close()
	kc := srv.WriteKubeconfig(t, "fake", "")
	proxy := testkit.NewForwardProxy(t, "alice", proxySecret)
	store := NewClusterSettingsStore(filepath.Join(t.TempDir(), "Cogpit"))
	s := New(Options{Connector: &ClusterConnector{KubeconfigPaths: []string{kc}, ClusterSettings: store}, ClusterSettings: store})
	t.Cleanup(s.Close)
	withUser := proxy.URLWithUser("alice", proxySecret).String()
	if _, err := s.ChangeClusterSettings("fake", func(cs *ClusterSettings) error { cs.Proxy = withUser; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(context.Background(), "fake"); err != nil {
		t.Fatalf("connect through the proxy: %v", err)
	}
	if proxy.Total() == 0 || !slices.Contains(proxy.Users(), "alice") {
		t.Fatalf("the proxy saw %d requests from %q", proxy.Total(), proxy.Users())
	}
	var lines []string
	for _, l := range s.ConnectLog("fake") {
		lines = append(lines, l.Text)
	}
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "Through the proxy http://alice:•••@"+proxy.URL.Host) || strings.Contains(joined, proxySecret) {
		t.Errorf("connection log:\n%s", joined)
	}

	// Cleared: the next connect goes straight to the server.
	if _, err := s.ChangeClusterSettings("fake", func(cs *ClusterSettings) error { cs.Proxy = ""; return nil }); err != nil {
		t.Fatal(err)
	}
	before := proxy.Total()
	if err := s.Disconnect(); err != nil { // a connect again would only show the open connection (D-69)
		t.Fatal(err)
	}
	if err := s.Connect(context.Background(), "fake"); err != nil {
		t.Fatal(err)
	}
	if proxy.Total() != before {
		t.Errorf("%d requests through a cleared proxy", proxy.Total()-before)
	}

	// A proxy that refuses: the connect fails, and neither the state's error nor the log quotes
	// the password.
	if _, err := s.ChangeClusterSettings("fake", func(cs *ClusterSettings) error {
		cs.Proxy = proxy.URLWithUser("alice", "wrong-"+proxySecret).String()
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.Disconnect(); err != nil {
		t.Fatal(err)
	}
	err := s.Connect(context.Background(), "fake")
	if err == nil || strings.Contains(err.Error(), proxySecret) || strings.Contains(s.UIState().Connection.Error, proxySecret) {
		t.Errorf("refused connect: %v; state %q", err, s.UIState().Connection.Error)
	}
	for _, l := range s.ConnectLog("fake") {
		if strings.Contains(l.Text, proxySecret) {
			t.Errorf("the connection log quotes the password: %q", l.Text)
		}
	}
}
