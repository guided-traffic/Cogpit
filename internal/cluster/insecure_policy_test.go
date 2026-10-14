package cluster

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestConnectAsksThePolicyAboutAnUncheckedCertificate (A28): a context whose cluster skips the
// certificate check and whose user runs nothing reaches the policy before any request, with a
// login of kind insecure that is the one InspectFile lists, fingerprint for fingerprint; refused,
// nothing reaches the server; allowed, it connects. A checked context never reaches the policy.
func TestConnectAsksThePolicyAboutAnUncheckedCertificate(t *testing.T) {
	ca := testkit.NewTestCA(t, "cogpit test CA")
	srv := testkit.NewFakeAPIServerWithCert(testkit.FakeAPIOptions{Token: secretToken}, ca.Issue(t, "127.0.0.1"))
	defer srv.Close()
	path := writeTLSKubeconfig(t, tlsCluster{server: srv.URL, insecure: true})

	in, err := InspectFile(path)
	if err != nil {
		t.Fatal(err)
	}
	listed := in.Exec["tls"]
	if listed == nil || listed.Kind() != kubeconfig.LoginInsecure {
		t.Fatalf("InspectFile's login: %+v", listed)
	}

	refusal := errors.New("not allowed in this test")
	var seen *kubeconfig.ExecPlugin
	_, err = Connect(context.Background(), Options{KubeconfigPaths: []string{path}, Context: "tls",
		ExecPolicy: func(p *kubeconfig.ExecPlugin) error { seen = p; return refusal }})
	if !errors.Is(err, refusal) {
		t.Fatalf("Connect with a refusing policy: %v", err)
	}
	mustNotLeak(t, err)
	if seen == nil || seen.Kind() != kubeconfig.LoginInsecure || seen.Fingerprint() != listed.Fingerprint() {
		t.Fatalf("the policy saw %+v, InspectFile listed %+v", seen, listed)
	}
	for k, n := range srv.Requests() {
		if n > 0 {
			t.Errorf("refused, yet %s reached the server %d times", k, n)
		}
	}

	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{path}, Context: "tls",
		ExecPolicy: func(*kubeconfig.ExecPlugin) error { return nil }})
	if err != nil {
		t.Fatalf("Connect with an allowing policy: %v", err)
	}
	defer c.Close()
	if !c.Insecure() {
		t.Error("the connection does not say its certificate is unchecked")
	}

	checked := writeTLSKubeconfig(t, tlsCluster{server: srv.URL, caData: ca.PEM()})
	c2, err := Connect(context.Background(), Options{KubeconfigPaths: []string{checked}, Context: "tls",
		ExecPolicy: func(p *kubeconfig.ExecPlugin) error { return errors.New("asked about " + p.String()) }})
	if err != nil {
		t.Fatalf("a checked context reached the policy: %v", err)
	}
	c2.Close()
	if strings.Contains(listed.String(), secretToken) {
		t.Error("the login's text holds the token")
	}
}
