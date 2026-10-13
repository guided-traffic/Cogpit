package model_test

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

var certNow = time.Date(2026, 10, 13, 12, 0, 0, 0, time.UTC)

// testChain makes a root (RSA 2048, self-signed), an intermediate (Ed25519, path length 0) and a
// leaf (ECDSA P-256) with subject alternative names of every kind.
func testChain(t *testing.T) (leaf, inter, root *testkit.TestCert) {
	t.Helper()
	root = testkit.NewCert(t, testkit.CertSpec{
		CommonName: "Example Root CA", Organization: []string{"Example"}, Serial: 1, CA: true,
		NotBefore: certNow.AddDate(-5, 0, 0), NotAfter: certNow.AddDate(10, 0, 0), Key: testkit.RSAKey(t, 2048),
	})
	inter = testkit.NewCert(t, testkit.CertSpec{
		CommonName: "Example Intermediate CA", Organization: []string{"Example"}, Serial: 2, CA: true, PathLenZero: true,
		NotBefore: certNow.AddDate(-1, 0, 0), NotAfter: certNow.AddDate(3, 0, 0), Key: testkit.Ed25519Key(t), Parent: root,
	})
	leaf = testkit.NewCert(t, testkit.CertSpec{
		CommonName: "storefront.example.com", Organization: []string{"Example Shop"}, Serial: 0x0a1b2c,
		NotBefore: certNow.AddDate(0, -2, 0), NotAfter: certNow.AddDate(0, 0, 90),
		DNSNames:    []string{"storefront.example.com", "*.storefront.example.com"},
		IPAddresses: []string{"10.0.0.7", "2001:db8::7"}, URIs: []string{"spiffe://example.com/ns/shop/sa/storefront"},
		Emails:   []string{"ops@example.com"},
		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		Parent: inter,
	})
	return leaf, inter, root
}

// A chain decodes in its order, each certificate with its public fields: names, serial, times in
// UTC, every kind of subject alternative name, the key's algorithm and size or curve, the
// signature algorithm, CA and path length, the usages, self-signed, fingerprint and key ids.
func TestParseCertificatesChain(t *testing.T) {
	leaf, inter, root := testChain(t)
	certs, omitted := model.ParseCertificates(testkit.Chain(leaf, inter, root), 0)
	if len(certs) != 3 || omitted != 0 {
		t.Fatalf("%d certificates, %d omitted", len(certs), omitted)
	}
	l, i, r := certs[0], certs[1], certs[2]
	if l.SubjectCN != "storefront.example.com" || l.Subject != "CN=storefront.example.com,O=Example Shop" ||
		l.IssuerCN != "Example Intermediate CA" || l.Issuer != "CN=Example Intermediate CA,O=Example" {
		t.Errorf("leaf names %q %q %q %q", l.SubjectCN, l.Subject, l.IssuerCN, l.Issuer)
	}
	if l.SerialNumber != "0A:1B:2C" || r.SerialNumber != "01" {
		t.Errorf("serials %q %q", l.SerialNumber, r.SerialNumber)
	}
	if !l.NotAfter.Equal(certNow.AddDate(0, 0, 90)) || l.NotAfter.Location() != time.UTC {
		t.Errorf("not after %v", l.NotAfter)
	}
	if strings.Join(l.DNSNames, ",") != "storefront.example.com,*.storefront.example.com" ||
		strings.Join(l.IPAddresses, ",") != "10.0.0.7,2001:db8::7" ||
		strings.Join(l.URIs, ",") != "spiffe://example.com/ns/shop/sa/storefront" ||
		strings.Join(l.EmailAddresses, ",") != "ops@example.com" {
		t.Errorf("SANs %v %v %v %v", l.DNSNames, l.IPAddresses, l.URIs, l.EmailAddresses)
	}
	if l.PublicKey != "ECDSA P-256" || i.PublicKey != "Ed25519" || r.PublicKey != "RSA 2048" {
		t.Errorf("public keys %q %q %q", l.PublicKey, i.PublicKey, r.PublicKey)
	}
	if l.SignatureAlgorithm != "Ed25519" || i.SignatureAlgorithm != "SHA256-RSA" || r.SignatureAlgorithm != "SHA256-RSA" {
		t.Errorf("signature algorithms %q %q %q", l.SignatureAlgorithm, i.SignatureAlgorithm, r.SignatureAlgorithm)
	}
	if l.IsCA || !i.IsCA || !r.IsCA || i.MaxPathLen == nil || *i.MaxPathLen != 0 || r.MaxPathLen != nil {
		t.Errorf("CA: leaf %v, intermediate %v %v, root %v %v", l.IsCA, i.IsCA, i.MaxPathLen, r.IsCA, r.MaxPathLen)
	}
	if strings.Join(l.KeyUsage, ", ") != "Digital Signature, Key Encipherment" ||
		strings.Join(l.ExtKeyUsage, ", ") != "Server Authentication, Client Authentication" ||
		strings.Join(r.KeyUsage, ", ") != "Certificate Sign, CRL Sign" {
		t.Errorf("usages %v %v %v", l.KeyUsage, l.ExtKeyUsage, r.KeyUsage)
	}
	if l.SelfSigned || i.SelfSigned || !r.SelfSigned {
		t.Errorf("self-signed: %v %v %v", l.SelfSigned, i.SelfSigned, r.SelfSigned)
	}
	sum := sha256.Sum256(leaf.Cert.Raw)
	if want := strings.ToUpper(hex.EncodeToString(sum[:])); strings.ReplaceAll(l.SHA256Fingerprint, ":", "") != want || len(l.SHA256Fingerprint) != 95 {
		t.Errorf("fingerprint %q, want %s", l.SHA256Fingerprint, want)
	}
	if l.AuthorityKeyID == "" || l.AuthorityKeyID != i.SubjectKeyID || r.SubjectKeyID == "" || i.AuthorityKeyID != r.SubjectKeyID {
		t.Errorf("key ids: leaf %q, intermediate %q %q, root %q", l.AuthorityKeyID, i.SubjectKeyID, i.AuthorityKeyID, r.SubjectKeyID)
	}
}

// Only CERTIFICATE blocks are decoded: a value with a private key holds no certificate, a private
// key next to a certificate is skipped, a certificate block that never ends or that holds another
// BEGIN line before its END is invalid and decoding goes on at the next block, and a block whose
// content is no certificate is invalid; none of them is quoted.
func TestParseCertificatesOnlyCertificateBlocks(t *testing.T) {
	leaf, _, _ := testChain(t)
	key := leaf.KeyPEM(t)
	if model.HoldsCertificate(key) {
		t.Error("a private key holds a certificate")
	}
	if certs, _ := model.ParseCertificates(key, 0); len(certs) != 0 {
		t.Errorf("a private key decoded: %+v", certs)
	}
	certs, _ := model.ParseCertificates(append(append([]byte{}, key...), leaf.PEM()...), 0)
	if len(certs) != 1 || certs[0].Invalid || certs[0].SubjectCN != "storefront.example.com" {
		t.Errorf("key then certificate: %+v", certs)
	}

	unterminated := []byte("-----BEGIN CERTIFICATE-----\nMIIB\n")
	value := bytes.Join([][]byte{unterminated, key, leaf.PEM()}, nil)
	certs, _ = model.ParseCertificates(value, 0)
	if len(certs) != 2 || !certs[0].Invalid || certs[1].Invalid {
		t.Errorf("an unterminated block, a key, a certificate: %+v", certs)
	}

	for name, block := range map[string]string{
		"not base64":      "-----BEGIN CERTIFICATE-----\nnot base64!\n-----END CERTIFICATE-----\n",
		"not DER":         "-----BEGIN CERTIFICATE-----\naGVsbG8gd29ybGQ=\n-----END CERTIFICATE-----\n",
		"unterminated":    string(unterminated),
		"mismatched ends": "-----BEGIN CERTIFICATE-----\naGVsbG8=\n-----END PRIVATE KEY-----\n",
	} {
		certs, _ := model.ParseCertificates([]byte(block), 0)
		if len(certs) != 1 || !certs[0].Invalid {
			t.Errorf("%s: %+v", name, certs)
			continue
		}
		b, _ := json.Marshal(certs[0])
		if string(b) != `{"invalid":true,"error":"not a valid certificate"}` {
			t.Errorf("%s: %s", name, b)
		}
	}
}

// At most MaxCertificatesPerKey certificates of a value are decoded; the rest are counted.
func TestParseCertificatesBounded(t *testing.T) {
	ca := testkit.NewCert(t, testkit.CertSpec{CommonName: "bundle", CA: true, Key: testkit.Ed25519Key(t)})
	var bundle []byte
	for range 20 {
		bundle = append(bundle, ca.PEM()...)
	}
	certs, omitted := model.ParseCertificates(bundle, 0)
	if len(certs) != model.MaxCertificatesPerKey || omitted != 20-model.MaxCertificatesPerKey {
		t.Errorf("%d decoded, %d omitted", len(certs), omitted)
	}
	if certs, omitted := model.ParseCertificates(bundle, 3); len(certs) != 3 || omitted != 17 {
		t.Errorf("max 3: %d decoded, %d omitted", len(certs), omitted)
	}
}

// Validity at a moment: the days left rounded down, expired, not yet valid, and the status with
// Cogpit's thresholds (a warning at 30 days or fewer, an error when expired or not yet valid).
func TestCertificateEvaluate(t *testing.T) {
	for _, c := range []struct {
		name                string
		notBefore, notAfter time.Time
		days                int
		expired, notYet     bool
		status              model.Status
	}{
		{"valid", certNow.AddDate(0, -1, 0), certNow.AddDate(0, 0, 90), 90, false, false, model.StatusOK},
		{"31 days", certNow.AddDate(0, -1, 0), certNow.AddDate(0, 0, 31), 31, false, false, model.StatusOK},
		{"30 days", certNow.AddDate(0, -1, 0), certNow.AddDate(0, 0, 30), 30, false, false, model.StatusWarning},
		{"10 days", certNow.AddDate(0, -1, 0), certNow.AddDate(0, 0, 10), 10, false, false, model.StatusWarning},
		{"last hours", certNow.AddDate(0, -1, 0), certNow.Add(5 * time.Hour), 0, false, false, model.StatusWarning},
		{"expired", certNow.AddDate(-1, 0, 0), certNow.AddDate(0, 0, -3), -3, true, false, model.StatusError},
		{"expired an hour ago", certNow.AddDate(-1, 0, 0), certNow.Add(-time.Hour), -1, true, false, model.StatusError},
		{"not yet valid", certNow.AddDate(0, 0, 5), certNow.AddDate(1, 0, 0), 365, false, true, model.StatusError},
	} {
		cert := model.Certificate{NotBefore: c.notBefore, NotAfter: c.notAfter}
		cert.Evaluate(certNow)
		if cert.ExpiresInDays != c.days || cert.Expired != c.expired || cert.NotYetValid != c.notYet || cert.Status != c.status {
			t.Errorf("%s: %d days, expired %v, not yet valid %v, %s", c.name, cert.ExpiresInDays, cert.Expired, cert.NotYetValid, cert.Status)
		}
	}
	invalid := model.Certificate{Invalid: true}
	invalid.Evaluate(certNow)
	if invalid.Status != model.StatusNone {
		t.Errorf("an invalid entry got a status: %+v", invalid)
	}
}

// Agents read times as RFC 3339 in UTC with the computed days and expired flag.
func TestCertificateJSON(t *testing.T) {
	leaf, _, _ := testChain(t)
	certs, _ := model.ParseCertificates(leaf.PEM(), 0)
	certs[0].Evaluate(certNow)
	b, err := json.Marshal(model.CertificateKey{Key: "tls.crt", Certificates: certs})
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf(`"notAfter":"%s"`, certNow.AddDate(0, 0, 90).Format(time.RFC3339))
	for _, s := range []string{`"key":"tls.crt"`, want, `"expiresInDays":90`, `"expired":false`, `"status":"ok"`,
		`"subjectCommonName":"storefront.example.com"`, `"publicKey":"ECDSA P-256"`, `"selfSigned":false`} {
		if !strings.Contains(string(b), s) {
			t.Errorf("no %s in %s", s, b)
		}
	}
}

// A certificate's texts reach a drawer line safely: control characters and bidirectional
// overrides are escaped, and a long text is cut.
func TestCertificateTextsAreSafe(t *testing.T) {
	cert := testkit.NewCert(t, testkit.CertSpec{
		CommonName: "evil\nCN=bank.example\u202e", DNSNames: []string{"a.example", strings.Repeat("x", 3000)},
		Key: testkit.Ed25519Key(t),
	})
	certs, _ := model.ParseCertificates(cert.PEM(), 0)
	if len(certs) != 1 || certs[0].Invalid {
		t.Fatalf("%+v", certs)
	}
	c := certs[0]
	if c.SubjectCN != "evil\\x0aCN=bank.example\\u202e" || strings.ContainsAny(c.Subject, "\n\u202e") {
		t.Errorf("subject %q, %q", c.SubjectCN, c.Subject)
	}
	if long := c.DNSNames[1]; len(long) > 1024+len("…") || !strings.HasSuffix(long, "…") {
		t.Errorf("a long name kept %d bytes", len(long))
	}
}
