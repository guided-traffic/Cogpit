package engine

import (
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// certsKubeconfig is a kubeconfig for srv with a context per client certificate given (by name),
// each its own user with the certificate and its key, and "plugin", whose user runs a program.
func certsKubeconfig(t *testing.T, srv *testkit.FakeAPIServer, certs map[string]*testkit.TestCert) string {
	t.Helper()
	var users, contexts strings.Builder
	for name, c := range certs {
		users.WriteString("- name: " + name + "\n  user:\n    client-certificate-data: " + base64.StdEncoding.EncodeToString(c.PEM()) +
			"\n    client-key-data: " + base64.StdEncoding.EncodeToString(c.KeyPEM(t)) + "\n")
		contexts.WriteString("- {name: " + name + ", context: {cluster: fake, user: " + name + "}}\n")
	}
	return `apiVersion: v1
kind: Config
clusters:
- name: fake
  cluster:
    server: ` + srv.URL + `
    certificate-authority-data: ` + base64.StdEncoding.EncodeToString(srv.CAData()) + `
users:
` + users.String() + `- name: plugin
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: kubelogin
      interactiveMode: Never
contexts:
` + contexts.String() + `- {name: plugin, context: {cluster: fake, user: plugin}}
`
}

// cluster.contexts carries each client certificate's dates and its state at the time of the
// listing (D-75): valid, expiring (5 days of a year's certificate), expired; nothing for a context
// whose user runs a credential plugin. Its JSON, what agents read, holds the dates and the state,
// never the certificate or the key. Connecting the context whose certificate expired is not
// refused, asked about or delayed by Cogpit: the connect reaches the server (which, being a fake,
// does not check client certificates).
func TestContextsCarryTheClientCertificate(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{})
	defer srv.Close()
	now := time.Date(2026, 10, 17, 9, 0, 0, 0, time.UTC)
	cert := func(from, until time.Duration) *testkit.TestCert {
		return testkit.NewCert(t, testkit.CertSpec{CommonName: "kubernetes-admin", NotBefore: now.Add(from), NotAfter: now.Add(until)})
	}
	const day = 24 * time.Hour
	// The fake server's own TLS handshake checks no client certificate, so the expired one connects
	// as long as Cogpit tries; its NotBefore lies in the past of the real clock too.
	certs := map[string]*testkit.TestCert{
		"valid":    cert(-300*day, 65*day),
		"expiring": cert(-360*day, 5*day),
		"expired":  cert(-400*day, -3*day),
	}
	file := filepath.Join(t.TempDir(), "certs.yaml")
	writeUserFile(t, file, certsKubeconfig(t, srv, certs))
	cfg := filepath.Join(t.TempDir(), "Cogpit")
	if _, _, err := NewKubeconfigSources(cfg).AddFile(file); err != nil {
		t.Fatal(err)
	}
	s := New(Options{Connector: &ClusterConnector{Sources: NewKubeconfigSources(cfg)}, Now: func() time.Time { return now }})
	t.Cleanup(s.Close)

	list, err := s.Contexts()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]ContextInfo{}
	for _, c := range list {
		got[c.KubeContext] = c
	}
	for name, want := range map[string]string{"valid": kubeconfig.CertValid, "expiring": kubeconfig.CertExpiring, "expired": kubeconfig.CertExpired} {
		v := got[name].ClientCert
		if v == nil || !v.NotAfter.Equal(certs[name].Cert.NotAfter) || !v.NotBefore.Equal(certs[name].Cert.NotBefore) || v.State != want {
			t.Errorf("%s: %+v, want %s until %v", name, v, want, certs[name].Cert.NotAfter)
		}
	}
	if c := got["plugin"]; c.ClientCert != nil {
		t.Errorf("a credential plugin's context: %+v", c.ClientCert)
	}
	out, err := json.Marshal(got["expired"])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"clientCert":{"notBefore":"2025-09-12T09:00:00Z","notAfter":"2026-10-14T09:00:00Z","state":"expired"}`) {
		t.Errorf("the JSON agents read: %s", out)
	}
	pemB64 := base64.StdEncoding.EncodeToString(certs["expired"].PEM())
	for _, leak := range []string{"PRIVATE KEY", "CERTIFICATE", pemB64[:40], "kubernetes-admin"} {
		if strings.Contains(string(out), leak) {
			t.Errorf("the JSON holds %q: %s", leak, out)
		}
	}

	// A second listing a day later: the state follows the clock, not the first listing.
	later := New(Options{Connector: &ClusterConnector{Sources: NewKubeconfigSources(cfg)}, Now: func() time.Time { return now.Add(70 * day) }})
	t.Cleanup(later.Close)
	again, err := later.Contexts()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range again {
		if c.KubeContext == "valid" && (c.ClientCert == nil || c.ClientCert.State != kubeconfig.CertExpired) {
			t.Errorf("70 days later: %+v", c.ClientCert)
		}
	}

	before := srv.Requests()["discovery"]
	if err := s.Connect(t.Context(), got["expired"].Name); err != nil {
		t.Fatalf("connecting the expired certificate's context: %v", err)
	}
	if u := s.UIState(); u.Connection.State != Connected || srv.Requests()["discovery"] == before {
		t.Errorf("connection %+v, discovery requests %d -> %d", u.Connection, before, srv.Requests()["discovery"])
	}
}
