package kubectl

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

// testCA is a certificate authority shaped like sigstore's: a root, an intermediate for code
// signing, and short-lived leaf certificates for an e-mail identity with Fulcio's issuer
// extensions.
type testCA struct {
	root, inter       *x509.Certificate
	rootKey, interKey *ecdsa.PrivateKey
	pem               []byte // root and intermediate, as NewVerifier reads them
}

func newTestCA(t *testing.T) *testCA {
	t.Helper()
	ca := &testCA{}
	ca.rootKey = newKey(t, elliptic.P384())
	root := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{Organization: []string{"test"}, CommonName: "test-root"},
		NotBefore: time.Now().Add(-48 * time.Hour), NotAfter: time.Now().Add(48 * time.Hour),
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, BasicConstraintsValid: true, IsCA: true,
	}
	ca.root = sign(t, root, root, &ca.rootKey.PublicKey, ca.rootKey)
	ca.interKey = newKey(t, elliptic.P384())
	inter := &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{Organization: []string{"test"}, CommonName: "test-intermediate"},
		NotBefore: time.Now().Add(-47 * time.Hour), NotAfter: time.Now().Add(47 * time.Hour),
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true, IsCA: true, MaxPathLenZero: true,
	}
	ca.inter = sign(t, inter, ca.root, &ca.interKey.PublicKey, ca.rootKey)
	ca.pem = append(pemOf(ca.root), pemOf(ca.inter)...)
	return ca
}

func newKey(t *testing.T, c elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func sign(t *testing.T, tmpl, parent *x509.Certificate, pub *ecdsa.PublicKey, key *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, key)
	if err != nil {
		t.Fatal(err)
	}
	c, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func pemOf(c *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
}

// leafOpts shape a signing certificate.
type leafOpts struct {
	identity, issuer string
	noIssuer         bool
	eku              x509.ExtKeyUsage
}

// leaf issues a ten-minute signing certificate that ended an hour ago, as a release's certificate
// has, and returns it as a .cert file (base64 of PEM) with its key.
func (ca *testCA) leaf(t *testing.T, o leafOpts) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	if o.identity == "" {
		o.identity = KubernetesRelease.Identity
	}
	if o.issuer == "" {
		o.issuer = KubernetesRelease.Issuer
	}
	if o.eku == 0 {
		o.eku = x509.ExtKeyUsageCodeSigning
	}
	key := newKey(t, elliptic.P256())
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), NotBefore: time.Now().Add(-70 * time.Minute),
		NotAfter: time.Now().Add(-60 * time.Minute), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{o.eku}, EmailAddresses: []string{o.identity},
	}
	if !o.noIssuer {
		v2, err := asn1.MarshalWithParams(o.issuer, "utf8")
		if err != nil {
			t.Fatal(err)
		}
		tmpl.ExtraExtensions = []pkix.Extension{{Id: oidIssuerV1, Value: []byte(o.issuer)}, {Id: oidIssuerV2, Value: v2}}
	}
	c := sign(t, tmpl, ca.inter, &key.PublicKey, ca.interKey)
	return []byte(base64.StdEncoding.EncodeToString(pemOf(c))), key
}

// sigFile signs a digest as cosign does and returns the .sig file.
func sigFile(t *testing.T, key *ecdsa.PrivateKey, digest []byte) []byte {
	t.Helper()
	sig, err := ecdsa.SignASN1(rand.Reader, key, digest)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(base64.StdEncoding.EncodeToString(sig) + "\n")
}

func (ca *testCA) verifier(t *testing.T) *Verifier {
	t.Helper()
	v, err := NewVerifier(ca.pem, KubernetesRelease)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// TestRealReleaseSignature verifies kubectl 1.36.1's published signature (darwin/arm64, read from
// dl.k8s.io on 2026-10-13) with the embedded sigstore certificate authority, offline: the release
// signs the SHA-256 it publishes, so the signature is checked without the 57 MB binary.
func TestRealReleaseSignature(t *testing.T) {
	read := func(name string) []byte {
		data, err := os.ReadFile("testdata/kubectl-v1.36.1-darwin-arm64." + name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	v := DefaultVerifier()
	cert, err := v.Certificate(read("cert"))
	if err != nil {
		t.Fatalf("the release's certificate: %v", err)
	}
	if cert.NotAfter.Sub(cert.NotBefore) > 15*time.Minute {
		t.Errorf("a signing certificate valid for %s", cert.NotAfter.Sub(cert.NotBefore))
	}
	digest, err := parseSHA256(read("sha256"))
	if err != nil {
		t.Fatal(err)
	}
	if err := v.Verify(cert, digest, read("sig")); err != nil {
		t.Fatalf("the release's signature: %v", err)
	}
	digest[0] ^= 1
	if err := v.Verify(cert, digest, read("sig")); !errors.Is(err, ErrSignature) {
		t.Errorf("a changed digest verified: %v", err)
	}
	other := *v
	other.Signer.Identity = "someone@example.com"
	if _, err := other.Certificate(read("cert")); !errors.Is(err, ErrSignature) {
		t.Errorf("another identity accepted the release's certificate: %v", err)
	}
}

// TestVerifierRefuses: every part of the check refuses on its own.
func TestVerifierRefuses(t *testing.T) {
	ca := newTestCA(t)
	v := ca.verifier(t)
	digest := sha256.Sum256([]byte("kubectl"))

	good, key := ca.leaf(t, leafOpts{})
	cert, err := v.Certificate(good)
	if err != nil {
		t.Fatalf("a good certificate: %v", err)
	}
	if err := v.Verify(cert, digest[:], sigFile(t, key, digest[:])); err != nil {
		t.Fatalf("a good signature: %v", err)
	}
	// A plain PEM .cert is read too.
	raw, _ := base64.StdEncoding.DecodeString(string(good))
	if _, err := v.Certificate(raw); err != nil {
		t.Errorf("a PEM certificate: %v", err)
	}

	other := newTestCA(t)
	foreign, _ := other.leaf(t, leafOpts{})
	for name, file := range map[string][]byte{
		"another identity":      first(ca.leaf(t, leafOpts{identity: "mallory@example.com"})),
		"another issuer":        first(ca.leaf(t, leafOpts{issuer: "https://token.actions.githubusercontent.com"})),
		"no issuer":             first(ca.leaf(t, leafOpts{noIssuer: true})),
		"not for code signing":  first(ca.leaf(t, leafOpts{eku: x509.ExtKeyUsageServerAuth})),
		"another authority":     foreign,
		"not base64 or PEM":     []byte("%%%"),
		"base64 of no PEM":      []byte(base64.StdEncoding.EncodeToString([]byte("hello"))),
		"the intermediate only": []byte(base64.StdEncoding.EncodeToString(pemOf(ca.inter))),
	} {
		if _, err := v.Certificate(file); !errors.Is(err, ErrSignature) {
			t.Errorf("%s: %v, want ErrSignature", name, err)
		}
	}

	_, otherKey := ca.leaf(t, leafOpts{})
	tampered := sha256.Sum256([]byte("kubectl, changed"))
	for name, sig := range map[string][]byte{
		"another key":       sigFile(t, otherKey, digest[:]),
		"another digest":    sigFile(t, key, tampered[:]),
		"not base64":        []byte("***"),
		"an empty sig file": nil,
	} {
		if err := v.Verify(cert, digest[:], sig); !errors.Is(err, ErrSignature) {
			t.Errorf("%s: %v, want ErrSignature", name, err)
		}
	}
	if _, err := NewVerifier(pemOf(ca.inter), KubernetesRelease); err == nil {
		t.Error("a verifier without a root")
	}
}

func first[A, B any](a A, _ B) A { return a }

// TestEmbeddedAuthority: the embedded file holds sigstore's root and intermediate, and the
// identity is the one Kubernetes documents.
func TestEmbeddedAuthority(t *testing.T) {
	v := DefaultVerifier()
	if v.Signer != KubernetesRelease || !strings.HasSuffix(v.Signer.Identity, "@k8s-releng-prod.iam.gserviceaccount.com") {
		t.Errorf("signer %+v", v.Signer)
	}
	var fps []string
	for rest := sigstoreFulcio; ; {
		var b *pem.Block
		if b, rest = pem.Decode(rest); b == nil {
			break
		}
		sum := sha256.Sum256(b.Bytes)
		fps = append(fps, hex.EncodeToString(sum[:]))
	}
	want := []string{
		"3ba7b6cc4e95469d4d334b49cb257ad8537076fa84b0ca87ff4ecfe6a54680c1", // CN=sigstore
		"15d795348226b4649f750f5802592c393bee7cc53c3b86982175b7ad087efe47", // CN=sigstore-intermediate
	}
	if strings.Join(fps, " ") != strings.Join(want, " ") {
		t.Errorf("embedded certificates %v, want %v", fps, want)
	}
}
