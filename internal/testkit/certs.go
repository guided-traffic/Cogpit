package testkit

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/url"
	"testing"
	"time"
)

// CertSpec describes a certificate for tests of how Cogpit decodes the certificates a Secret holds
// (D-30). Every field is optional: an empty spec is a self-signed leaf with a new ECDSA P-256 key,
// valid for a day around now.
type CertSpec struct {
	CommonName   string
	Organization []string
	Serial       int64 // 0: the next of the process's serials
	NotBefore    time.Time
	NotAfter     time.Time
	DNSNames     []string
	IPAddresses  []string
	URIs         []string
	Emails       []string
	CA           bool
	MaxPathLen   int  // with CA: the path length constraint; 0 with MaxPathLenZero is a limit of 0
	PathLenZero  bool // with CA: MaxPathLen 0 is a limit, not "none"
	KeyUsage     x509.KeyUsage
	ExtKeyUsage  []x509.ExtKeyUsage
	Key          crypto.Signer // the certificate's own key; nil: a new ECDSA P-256 key
	Parent       *TestCert     // the issuer; nil: self-signed
}

// TestCert is a certificate made for a test, with its key. Keys are made for the test and never
// written to disk.
type TestCert struct {
	Cert *x509.Certificate
	Key  crypto.Signer
}

// NewCert makes the certificate s describes, signed by its parent's key or, without a parent, by
// its own.
func NewCert(t testing.TB, s CertSpec) *TestCert {
	t.Helper()
	key := s.Key
	if key == nil {
		key = ECDSAKey(t)
	}
	serial := s.Serial
	if serial == 0 {
		serial = serials.Add(1)
	}
	notBefore, notAfter := s.NotBefore, s.NotAfter
	if notBefore.IsZero() {
		notBefore = time.Now().Add(-time.Hour)
	}
	if notAfter.IsZero() {
		notAfter = notBefore.Add(24 * time.Hour)
	}
	tmpl := &x509.Certificate{
		SerialNumber:   big.NewInt(serial),
		Subject:        pkix.Name{CommonName: s.CommonName, Organization: s.Organization},
		NotBefore:      notBefore,
		NotAfter:       notAfter,
		DNSNames:       s.DNSNames,
		EmailAddresses: s.Emails,
		KeyUsage:       s.KeyUsage,
		ExtKeyUsage:    s.ExtKeyUsage,
	}
	for _, ip := range s.IPAddresses {
		tmpl.IPAddresses = append(tmpl.IPAddresses, net.ParseIP(ip))
	}
	for _, u := range s.URIs {
		parsed, err := url.Parse(u)
		if err != nil {
			t.Fatalf("URI %q: %v", u, err)
		}
		tmpl.URIs = append(tmpl.URIs, parsed)
	}
	if s.CA {
		tmpl.BasicConstraintsValid, tmpl.IsCA = true, true
		tmpl.MaxPathLen, tmpl.MaxPathLenZero = s.MaxPathLen, s.PathLenZero
		if tmpl.KeyUsage == 0 {
			tmpl.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageCRLSign
		}
	}
	parent, signer := tmpl, key
	if s.Parent != nil {
		parent, signer = s.Parent.Cert, s.Parent.Key
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, key.Public(), signer)
	if err != nil {
		t.Fatalf("certificate %q: %v", s.CommonName, err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("certificate %q: %v", s.CommonName, err)
	}
	return &TestCert{Cert: cert, Key: key}
}

// PEM returns the certificate as a PEM CERTIFICATE block.
func (c *TestCert) PEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Cert.Raw})
}

// KeyPEM returns the certificate's key as a PKCS #8 PRIVATE KEY block, as a TLS Secret's tls.key
// holds it.
func (c *TestCert) KeyPEM(t testing.TB) []byte {
	t.Helper()
	der, err := x509.MarshalPKCS8PrivateKey(c.Key)
	if err != nil {
		t.Fatalf("key: %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

// Chain returns the certificates' PEM blocks one after the other, as a tls.crt with a chain holds
// them.
func Chain(certs ...*TestCert) []byte {
	var out []byte
	for _, c := range certs {
		out = append(out, c.PEM()...)
	}
	return out
}

// ECDSAKey makes an ECDSA P-256 key.
func ECDSAKey(t testing.TB) crypto.Signer {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ECDSA key: %v", err)
	}
	return k
}

// RSAKey makes an RSA key of bits bits.
func RSAKey(t testing.TB, bits int) crypto.Signer {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		t.Fatalf("RSA key: %v", err)
	}
	return k
}

// Ed25519Key makes an Ed25519 key.
func Ed25519Key(t testing.TB) crypto.Signer {
	t.Helper()
	_, k, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("Ed25519 key: %v", err)
	}
	return k
}
