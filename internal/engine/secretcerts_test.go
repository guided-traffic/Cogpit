package engine

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// certsNow is the moment the certificate tests evaluate at; the drawer writes times in its zone.
var certsNow = time.Date(2026, 10, 13, 12, 0, 0, 0, time.FixedZone("CEST", 2*3600))

// certFixture is a TLS Secret with a chain in tls.crt, its root in ca.crt and the leaf's key in
// tls.key, and an Opaque Secret with an expired certificate, one expiring in 10 days, one not yet
// valid, a bundle with a malformed block, a private key under a certificate's name, a value over
// 1 MiB and a password.
type certFixture struct {
	tls, opaque *unstructured.Unstructured
	leaf, inter *testkit.TestCert
	root        *testkit.TestCert
	keyPEM      []byte
	values      []string // every value of both Secrets, as the API sends them (base64) and decoded
}

func newCertFixture(t *testing.T) *certFixture {
	t.Helper()
	f := &certFixture{}
	f.root = testkit.NewCert(t, testkit.CertSpec{
		CommonName: "Example Root CA", Organization: []string{"Example"}, Serial: 1, CA: true,
		NotBefore: certsNow.AddDate(-5, 0, 0), NotAfter: certsNow.AddDate(10, 0, 0), Key: testkit.RSAKey(t, 2048),
	})
	f.inter = testkit.NewCert(t, testkit.CertSpec{
		CommonName: "Example Intermediate CA", Organization: []string{"Example"}, Serial: 2, CA: true, PathLenZero: true,
		NotBefore: certsNow.AddDate(-1, 0, 0), NotAfter: certsNow.AddDate(3, 0, 0), Key: testkit.Ed25519Key(t), Parent: f.root,
	})
	f.leaf = testkit.NewCert(t, testkit.CertSpec{
		CommonName: "storefront.example.com", Organization: []string{"Example Shop"}, Serial: 0x0a1b2c,
		NotBefore: certsNow.AddDate(0, -2, 0), NotAfter: certsNow.AddDate(0, 0, 90),
		DNSNames:    []string{"storefront.example.com", "*.storefront.example.com"},
		IPAddresses: []string{"10.0.0.7"}, URIs: []string{"spiffe://example.com/ns/shop/sa/storefront"},
		Emails:   []string{"ops@example.com"},
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		Parent: f.inter,
	})
	f.keyPEM = f.leaf.KeyPEM(t)
	tlsData := map[string]string{
		"tls.crt": string(testkit.Chain(f.leaf, f.inter, f.root)),
		"ca.crt":  string(f.root.PEM()),
		"tls.key": string(f.keyPEM),
	}
	f.tls = testkit.Secret("shop", "storefront-tls", nil, tlsData)
	f.tls.Object["type"] = tlsSecretType

	expired := testkit.NewCert(t, testkit.CertSpec{CommonName: "old.example.com", NotBefore: certsNow.AddDate(-1, 0, 0), NotAfter: certsNow.AddDate(0, 0, -3), Key: testkit.Ed25519Key(t)})
	soon := testkit.NewCert(t, testkit.CertSpec{CommonName: "soon.example.com", NotBefore: certsNow.AddDate(0, -3, 0), NotAfter: certsNow.AddDate(0, 0, 10), Key: testkit.Ed25519Key(t)})
	future := testkit.NewCert(t, testkit.CertSpec{CommonName: "future.example.com", NotBefore: certsNow.AddDate(0, 0, 5), NotAfter: certsNow.AddDate(1, 0, 0), Key: testkit.Ed25519Key(t)})
	malformed := "-----BEGIN CERTIFICATE-----\nbm90IGEgY2VydGlmaWNhdGU=\n-----END CERTIFICATE-----\n"
	opaqueData := map[string]string{
		"expired.pem": string(expired.PEM()),
		"soon.crt":    string(soon.PEM()),
		"future.crt":  string(future.PEM()),
		"bundle.pem":  malformed + string(soon.PEM()),
		"server.crt":  string(f.keyPEM), // a private key under a certificate's name
		"password":    "hunter2",
	}
	f.opaque = testkit.Secret("shop", "mixed", nil, opaqueData)
	huge := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("A", certValueBytes+1)))
	_ = unstructured.SetNestedField(f.opaque.Object, huge, "data", "huge.crt")

	for _, data := range []map[string]string{tlsData, opaqueData} {
		for _, v := range data {
			f.values = append(f.values, v, base64.StdEncoding.EncodeToString([]byte(v)))
			for _, line := range strings.Split(v, "\n") { // a PEM body's lines: base64 of the DER
				if len(line) >= 16 && !strings.HasPrefix(line, "-----") {
					f.values = append(f.values, line)
				}
			}
		}
	}
	return f
}

// leaks returns the first value of the fixture that b carries, or "".
func (f *certFixture) leaks(b []byte) string {
	for _, v := range f.values {
		if strings.Contains(string(b), v) {
			if len(v) > 40 {
				return v[:40] + "…"
			}
			return v
		}
	}
	for _, s := range []string{"PRIVATE KEY", "BEGIN CERTIFICATE"} {
		if strings.Contains(string(b), s) {
			return s
		}
	}
	return ""
}

func certDetail(t *testing.T, obj *unstructured.Unstructured) *DetailModel {
	t.Helper()
	p, _ := kinds.ByKey(secretsKind)
	ref := model.ObjectRef{Context: "test", Kind: secretsKind, Namespace: obj.GetNamespace(), Name: obj.GetName()}
	m, err := BuildDetail(p, ref, obj, nil, certsNow)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// certSections returns the drawer's certificate sections by group.
func certSections(m *DetailModel) map[string][]model.DetailSection {
	out := map[string][]model.DetailSection{}
	for _, s := range m.Sections {
		if strings.HasPrefix(s.Group, certGroupPrefix) {
			out[strings.TrimPrefix(s.Group, certGroupPrefix)] = append(out[strings.TrimPrefix(s.Group, certGroupPrefix)], s)
		}
	}
	return out
}

func itemOf(sec model.DetailSection, name string) model.DetailItem {
	for _, it := range sec.Items {
		if it.Name == name {
			return it
		}
	}
	return model.DetailItem{}
}

// A TLS Secret's drawer has a group per certificate key after the Data section, tls.crt first,
// then ca.crt: one section per certificate of the chain in its order, titled by the subject's
// common name, each field its own item, times in the drawer's zone; tls.key has none.
func TestSecretCertificateSections(t *testing.T) {
	f := newCertFixture(t)
	m := certDetail(t, f.tls)
	var groups []string
	for _, s := range m.Sections {
		if s.Group != "" && (len(groups) == 0 || groups[len(groups)-1] != s.Group) {
			groups = append(groups, s.Group)
		}
	}
	if strings.Join(groups, " | ") != "Certificates · tls.crt | Certificates · ca.crt" {
		t.Fatalf("groups %v", groups)
	}
	if m.Sections[1].Title != "Data" {
		t.Errorf("the Data section moved: %q", m.Sections[1].Title)
	}
	secs := certSections(m)
	chain := secs["tls.crt"]
	if len(chain) != 3 || chain[0].Title != "1 · storefront.example.com" || chain[1].Title != "2 · Example Intermediate CA" ||
		chain[2].Title != "3 · Example Root CA" || len(secs["ca.crt"]) != 1 || secs["ca.crt"][0].Title != "Example Root CA" {
		t.Fatalf("sections %+v", secs)
	}
	leaf := chain[0]
	want := map[string]string{
		"Subject":             "storefront.example.com",
		"Subject DN":          "CN=storefront.example.com,O=Example Shop",
		"Issuer":              "Example Intermediate CA",
		"Issuer DN":           "CN=Example Intermediate CA,O=Example",
		"Serial Number":       "0A:1B:2C",
		"Not Before":          "2026-08-13T12:00:00+02:00",
		"Not After":           "2027-01-11T12:00:00+02:00 (in 90 days)",
		"DNS Names":           "storefront.example.com\n*.storefront.example.com",
		"IP Addresses":        "10.0.0.7",
		"URIs":                "spiffe://example.com/ns/shop/sa/storefront",
		"E-mail Addresses":    "ops@example.com",
		"Public Key":          "ECDSA P-256",
		"Signature Algorithm": "Ed25519",
		"CA":                  "No",
		"Key Usage":           "Digital Signature",
		"Extended Key Usage":  "Server Authentication",
		"Self-Signed":         "No",
	}
	for name, v := range want {
		if got := itemOf(leaf, name).Value; got != v {
			t.Errorf("leaf %s: %q, want %q", name, got, v)
		}
	}
	if itemOf(leaf, "Not After").Status != model.StatusOK {
		t.Errorf("a certificate with 90 days left: %+v", itemOf(leaf, "Not After"))
	}
	for _, name := range []string{"SHA-256 Fingerprint", "Authority Key ID"} {
		if itemOf(leaf, name).Value == "" {
			t.Errorf("leaf has no %s", name)
		}
	}
	if got := itemOf(chain[1], "CA").Value; got != "Yes, path length 0" {
		t.Errorf("intermediate CA %q", got)
	}
	if got := itemOf(chain[2], "CA").Value + " | " + itemOf(chain[2], "Self-Signed").Value + " | " + itemOf(chain[2], "Public Key").Value; got != "Yes, no path length limit | Yes | RSA 2048" {
		t.Errorf("root %q", got)
	}
	// The same fields for agents, with times in UTC.
	if len(m.Certificates) != 2 || m.Certificates[0].Key != "tls.crt" || len(m.Certificates[0].Certificates) != 3 {
		t.Fatalf("certificates %+v", m.Certificates)
	}
	c := m.Certificates[0].Certificates[0]
	if c.NotAfter.Location() != time.UTC || c.ExpiresInDays != 90 || c.Expired || c.Status != model.StatusOK {
		t.Errorf("leaf for agents %+v", c)
	}
	b, _ := json.Marshal(m)
	if l := f.leaks(b); l != "" {
		t.Errorf("the drawer model carries %q", l)
	}
}

// Validity and bounds in the drawer: an expired certificate is an error with the days since, one
// expiring in 10 days a warning, one not yet valid an error on both times; a malformed block says
// so without quoting it; a private key under a certificate's name, a password and a key named
// .key show nothing; a value over 1 MiB is not decoded and says so.
func TestSecretCertificateStatusesAndBounds(t *testing.T) {
	f := newCertFixture(t)
	m := certDetail(t, f.opaque)
	secs := certSections(m)
	var keys []string
	for _, ck := range m.Certificates {
		keys = append(keys, ck.Key)
	}
	if strings.Join(keys, ",") != "bundle.pem,expired.pem,future.crt,huge.crt,soon.crt" {
		t.Fatalf("keys with certificates %v", keys)
	}
	if na := itemOf(secs["expired.pem"][0], "Not After"); na.Status != model.StatusError || !strings.HasSuffix(na.Value, "(expired 3 days ago)") {
		t.Errorf("expired %+v", na)
	}
	if na := itemOf(secs["soon.crt"][0], "Not After"); na.Status != model.StatusWarning || !strings.HasSuffix(na.Value, "(in 10 days)") {
		t.Errorf("expiring in 10 days %+v", na)
	}
	fut := secs["future.crt"][0]
	if nb, na := itemOf(fut, "Not Before"), itemOf(fut, "Not After"); nb.Status != model.StatusError ||
		!strings.HasSuffix(nb.Value, "(not yet valid, in 5 days)") || na.Status != model.StatusError {
		t.Errorf("not yet valid %+v %+v", nb, na)
	}
	bundle := secs["bundle.pem"]
	if len(bundle) != 2 || bundle[0].Title != "1 · Certificate" || len(bundle[0].Items) != 1 ||
		bundle[0].Items[0].Value != model.NotACertificate || bundle[0].Items[0].Status != model.StatusError ||
		bundle[1].Title != "2 · soon.example.com" {
		t.Errorf("bundle %+v", bundle)
	}
	if huge := secs["huge.crt"]; len(huge) != 1 || huge[0].Title != certNotDecodedT || huge[0].Items[0].Value != certTooLarge {
		t.Errorf("a value over 1 MiB %+v", huge)
	}
	for _, k := range []string{"server.crt", "password"} {
		if _, ok := secs[k]; ok {
			t.Errorf("%s has a certificate section", k)
		}
	}
	b, _ := json.Marshal(m)
	if l := f.leaks(b); l != "" {
		t.Errorf("the drawer model carries %q", l)
	}
	if !strings.Contains(string(b), `{"invalid":true,"error":"not a valid certificate"}`) {
		t.Errorf("agents do not read the malformed block as invalid: %s", b)
	}
}

// A TLS Secret's tls.crt that holds no PEM certificate says so; other kinds of drawers have no
// certificate sections, and a Secret's certificates are bounded per Secret.
func TestSecretCertificateEdges(t *testing.T) {
	notPEM := testkit.Secret("shop", "broken-tls", nil, map[string]string{"tls.crt": "not-a-cert", "tls.key": "not-a-key"})
	notPEM.Object["type"] = tlsSecretType
	m := certDetail(t, notPEM)
	if secs := certSections(m)["tls.crt"]; len(secs) != 1 || secs[0].Items[0].Value != certNoPEM {
		t.Errorf("a tls.crt without PEM: %+v", certSections(m))
	}
	if b, _ := json.Marshal(m); strings.Contains(string(b), "not-a-cert") || strings.Contains(string(b), "not-a-key") {
		t.Errorf("a value reached the drawer: %s", b)
	}

	ca := testkit.NewCert(t, testkit.CertSpec{CommonName: "bundle", CA: true, Key: testkit.Ed25519Key(t)})
	data := map[string]string{}
	for _, k := range []string{"a.pem", "b.pem", "c.pem", "d.pem", "e.pem"} {
		data[k] = strings.Repeat(string(ca.PEM()), 15)
	}
	m = certDetail(t, testkit.Secret("shop", "bundles", nil, data))
	decoded, omitted := 0, 0
	for _, ck := range m.Certificates {
		decoded += len(ck.Certificates)
		omitted += ck.Omitted
	}
	if decoded != certsPerSecret || omitted != 75-certsPerSecret {
		t.Errorf("%d decoded, %d omitted", decoded, omitted)
	}
	if more := certSections(m)["e.pem"]; len(more) == 0 || more[len(more)-1].Title != certMoreTitle {
		t.Errorf("the last key says nothing of the certificates not decoded: %+v", more)
	}
}
