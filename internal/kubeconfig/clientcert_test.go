package kubeconfig

import (
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// certKubeconfig is a kubeconfig whose contexts' users log in with client certificates (data and
// file), a token, a credential plugin and an auth provider. keyData is client-key-data as written.
func certKubeconfig(certData, keyData string) string {
	return `apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: https://c.example:6443
users:
- name: data
  user:
    client-certificate-data: ` + certData + `
    client-key-data: ` + keyData + `
- name: file
  user:
    client-certificate: /etc/cogpit-test/client.crt
    client-key: /etc/cogpit-test/client.key
- name: token
  user:
    token: ` + secret + `
- name: plugin
  user:
    client-certificate-data: ` + certData + `
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: kubelogin
- name: provider
  user:
    client-certificate-data: ` + certData + `
    auth-provider:
      name: oidc
contexts:
- {name: data, context: {cluster: c, user: data}}
- {name: file, context: {cluster: c, user: file}}
- {name: token, context: {cluster: c, user: token}}
- {name: plugin, context: {cluster: c, user: plugin}}
- {name: provider, context: {cluster: c, user: provider}}
- {name: nouser, context: {cluster: c, user: missing}}
`
}

func contextsByName(t *testing.T, doc string) map[string]Context {
	t.Helper()
	list, err := Contexts([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]Context{}
	for _, c := range list {
		out[c.Name] = c
	}
	return out
}

// A context whose user logs in with client-certificate-data carries the certificate's two dates
// (D-75): a certificate valid for N more days and one that expired both. A client-certificate file
// is named for package cluster to read; a token, a credential plugin and an auth provider (whose
// credential is not known before they run), and a missing user carry nothing. The key is never
// read: a key that is garbage, or no key at all, leaves the dates as they are, and nothing of the
// key or the certificate's PEM reaches the context's JSON.
func TestClientCertificateDates(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	for _, tc := range []struct {
		name      string
		notBefore time.Time
		notAfter  time.Time
	}{
		{"valid for 23 more days", now.Add(-24 * time.Hour), now.Add(23 * 24 * time.Hour)},
		{"expired 3 days ago", now.Add(-400 * 24 * time.Hour), now.Add(-3 * 24 * time.Hour)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cert := testkit.NewCert(t, testkit.CertSpec{CommonName: "kubernetes-admin", Organization: []string{"system:masters"},
				NotBefore: tc.notBefore, NotAfter: tc.notAfter})
			certData := base64.StdEncoding.EncodeToString(cert.PEM())
			garbageKey := base64.StdEncoding.EncodeToString([]byte("-----BEGIN EC PRIVATE KEY-----\nnot a key " + secret + "\n-----END EC PRIVATE KEY-----\n"))
			realKey := base64.StdEncoding.EncodeToString(cert.KeyPEM(t))
			for _, key := range []string{realKey, garbageKey, `""`} {
				got := contextsByName(t, certKubeconfig(certData, key))
				v := got["data"].ClientCert
				if v == nil || !v.NotBefore.Equal(tc.notBefore) || !v.NotAfter.Equal(tc.notAfter) || v.State != "" {
					t.Fatalf("key %.12s…: client certificate %+v, want %v to %v", key, v, tc.notBefore, tc.notAfter)
				}
				if c := got["file"]; c.ClientCert != nil || c.ClientCertFile != "/etc/cogpit-test/client.crt" {
					t.Errorf("the file form: %+v", c)
				}
				for _, name := range []string{"token", "plugin", "provider", "nouser"} {
					if c := got[name]; c.ClientCert != nil || c.ClientCertFile != "" || c.ClientCertProblem != "" {
						t.Errorf("%s carries a certificate: %+v", name, c)
					}
				}
				out, err := json.Marshal(got["data"])
				if err != nil {
					t.Fatal(err)
				}
				for _, leak := range []string{key, certData, "PRIVATE KEY", "BEGIN CERTIFICATE", secret, "kubernetes-admin", "system:masters"} {
					if leak != `""` && strings.Contains(string(out), leak) {
						t.Errorf("the context's JSON holds %.20q: %s", leak, out)
					}
				}
			}
		})
	}
}

// Data that is no certificate gives no date and a fixed text that quotes nothing of it.
func TestUnreadableClientCertificate(t *testing.T) {
	notCert := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte(secret)})
	for _, data := range [][]byte{[]byte(secret), notCert, nil} {
		_, err := ParseCertValidity(data)
		if err == nil || strings.Contains(err.Error(), secret) {
			t.Errorf("%q: %v", data, err)
		}
	}
	got := contextsByName(t, certKubeconfig(base64.StdEncoding.EncodeToString(notCert), `""`))
	if c := got["data"]; c.ClientCert != nil || c.ClientCertProblem != "client-certificate-data: not an X.509 certificate" {
		t.Errorf("%+v", c)
	}
}

// The leaf is the first CERTIFICATE block, as crypto/tls presents it: blocks of other types before
// it are skipped, the chain's other certificates after it are not looked at.
func TestClientCertificateIsTheLeaf(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	ca := testkit.NewCert(t, testkit.CertSpec{CommonName: "ca", CA: true, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour)})
	leaf := testkit.NewCert(t, testkit.CertSpec{CommonName: "u", Parent: ca, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(90 * 24 * time.Hour)})
	data := append(pem.EncodeToMemory(&pem.Block{Type: "COMMENT", Bytes: []byte("x")}), testkit.Chain(leaf, ca)...)
	v, err := ParseCertValidity(data)
	if err != nil || !v.NotAfter.Equal(leaf.Cert.NotAfter) {
		t.Errorf("%+v, %v; want the leaf's %v", v, err, leaf.Cert.NotAfter)
	}
}

// StateAt: valid; expiring within 14 days, or within the last fifth of a shorter lifetime (a
// certificate for 12 hours is expiring in its last 2 h 24 min, not all its life); expired after
// NotAfter, as crypto/x509 checks it; not valid before NotBefore. Describe says it in words.
func TestClientCertificateState(t *testing.T) {
	at := time.Date(2026, 10, 17, 12, 0, 0, 0, time.UTC)
	year := CertValidity{NotBefore: at.Add(-300 * 24 * time.Hour), NotAfter: at.Add(65 * 24 * time.Hour)}
	day := 24 * time.Hour
	short := CertValidity{NotBefore: at, NotAfter: at.Add(12 * time.Hour)}
	for _, tc := range []struct {
		name  string
		v     CertValidity
		now   time.Time
		state string
		text  string
	}{
		{"a year's certificate, 65 days left", year, at, CertValid, "expires in 65 days"},
		{"15 days left", year, year.NotAfter.Add(-15 * day), CertValid, "expires in 15 days"},
		{"14 days left", year, year.NotAfter.Add(-14 * day), CertExpiring, "expires in 14 days"},
		{"47 hours left", year, year.NotAfter.Add(-47*time.Hour - 30*time.Minute), CertExpiring, "expires in 47 h"},
		{"5 hours left", year, year.NotAfter.Add(-5 * time.Hour), CertExpiring, "expires in 5 h"},
		{"12 minutes left", year, year.NotAfter.Add(-12 * time.Minute), CertExpiring, "expires in 12 min"},
		{"seconds left", year, year.NotAfter.Add(-10 * time.Second), CertExpiring, "expires in less than a minute"},
		{"at NotAfter", year, year.NotAfter, CertExpiring, "expires in less than a minute"},
		{"just after NotAfter", year, year.NotAfter.Add(time.Second), CertExpired, "expired just now"},
		{"3 days after", year, year.NotAfter.Add(3*day + time.Hour), CertExpired, "expired 3 days ago"},
		{"5 hours after", year, year.NotAfter.Add(5 * time.Hour), CertExpired, "expired 5 h ago"},
		{"before NotBefore", year, year.NotBefore.Add(-time.Minute), CertNotYetValid, "not valid yet"},
		{"three years left", CertValidity{NotBefore: at, NotAfter: at.Add(1095 * day)}, at, CertValid, "expires in 3 years"},
		{"a 12 h certificate, 3 h left", short, short.NotAfter.Add(-3 * time.Hour), CertValid, "expires in 3 h"},
		{"a 12 h certificate, 2 h left", short, short.NotAfter.Add(-2 * time.Hour), CertExpiring, "expires in 2 h"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.v.StateAt(tc.now); got != tc.state {
				t.Errorf("state %q, want %q", got, tc.state)
			}
			if got := tc.v.Describe(tc.now); got != tc.text {
				t.Errorf("text %q, want %q", got, tc.text)
			}
			if got := tc.v.At(tc.now); got.State != tc.state || tc.v.State != "" {
				t.Errorf("At: %+v (the original %+v)", got, tc.v)
			}
		})
	}
}
