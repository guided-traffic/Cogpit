package testkit

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// TestCA is a certificate authority for tests of how a client trusts a server: it issues server
// certificates for the host names and addresses a test names, so a test can serve a certificate
// that chains to a CA the kubeconfig names, one that names another host than the server's URL, or
// one no client trusts. Keys are ECDSA P-256, made for the test and never written to disk.
type TestCA struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
}

// serials numbers the certificates of every TestCA of the process.
var serials atomic.Int64

// NewTestCA makes a self-signed CA certificate with the common name name, valid for a day around
// now.
func NewTestCA(t testing.TB, name string) *TestCA {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("CA key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(serials.Add(1)),
		Subject:               pkix.Name{CommonName: name},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(23 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("CA certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("CA certificate: %v", err)
	}
	return &TestCA{cert: cert, key: key}
}

// PEM returns the CA's certificate in PEM, as a kubeconfig's certificate-authority-data holds it
// (base64-encoded in the YAML) or a certificate-authority file does.
func (ca *TestCA) PEM() []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw})
}

// Issue makes a server certificate signed by the CA, valid for each host: an IP address goes into
// the certificate's IP addresses, anything else into its DNS names. The returned certificate
// carries the CA's certificate as its chain.
func (ca *TestCA) Issue(t testing.TB, hosts ...string) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("server key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(serials.Add(1)),
		Subject:      pkix.Name{CommonName: "cogpit test server"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(23 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	for _, h := range hosts {
		if ip := net.ParseIP(h); ip != nil {
			tmpl.IPAddresses = append(tmpl.IPAddresses, ip)
		} else {
			tmpl.DNSNames = append(tmpl.DNSNames, h)
		}
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("server certificate: %v", err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("server certificate: %v", err)
	}
	return tls.Certificate{Certificate: [][]byte{der, ca.cert.Raw}, PrivateKey: key, Leaf: leaf}
}

// NewFakeAPIServerWithCert starts a FakeAPIServer that presents cert (with its chain) in place of
// httptest's built-in certificate for 127.0.0.1 and example.com. Kubeconfig and CAData then name
// the leaf certificate, which Go trusts as a root of its own; a test of a CA writes its own
// kubeconfig with the CA's PEM. Close stops it as usual.
func NewFakeAPIServerWithCert(opts FakeAPIOptions, cert tls.Certificate) *FakeAPIServer {
	s := NewFakeAPIServer(opts)
	// Serve the same handler on a listener that presents cert: the server built above is replaced
	// before any client saw its URL.
	s.srv.Close()
	srv := httptest.NewUnstartedServer(http.HandlerFunc(s.serveHTTP))
	srv.EnableHTTP2 = true
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	// A client that refuses the certificate is what these tests are about; the server's log line
	// for each refused handshake is noise.
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.StartTLS()
	s.srv, s.URL = srv, srv.URL
	return s
}
