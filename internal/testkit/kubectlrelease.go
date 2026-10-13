package testkit

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
	"math/big"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/kubectl"
)

// KubectlReleases is a local release host for kubectl downloads: an https server that lays out
// releases as dl.k8s.io does (/release/v<version>/bin/<os>/<arch>/kubectl with .sha256, .sig and
// .cert), signed by a test certificate authority shaped like sigstore's for the Kubernetes
// release's identity. Tests download from it; nothing reaches the internet.
type KubectlReleases struct {
	srv      *httptest.Server
	verifier *kubectl.Verifier
	inter    *x509.Certificate
	interKey *ecdsa.PrivateKey

	mu    sync.Mutex
	files map[string][]byte
	asked []string
}

// NewKubectlReleases starts the release host; it stops when the test ends.
func NewKubectlReleases(t testing.TB) *KubectlReleases {
	t.Helper()
	r := &KubectlReleases{files: map[string][]byte{}}
	rootKey := kubectlKey(t, elliptic.P384())
	root := kubectlCert(t, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test-root"},
		NotBefore: time.Now().Add(-48 * time.Hour), NotAfter: time.Now().Add(48 * time.Hour),
		KeyUsage: x509.KeyUsageCertSign, BasicConstraintsValid: true, IsCA: true,
	}, nil, &rootKey.PublicKey, rootKey)
	r.interKey = kubectlKey(t, elliptic.P384())
	r.inter = kubectlCert(t, &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "test-intermediate"},
		NotBefore: time.Now().Add(-47 * time.Hour), NotAfter: time.Now().Add(47 * time.Hour),
		KeyUsage: x509.KeyUsageCertSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true, IsCA: true, MaxPathLenZero: true,
	}, root, &r.interKey.PublicKey, rootKey)
	v, err := kubectl.NewVerifier(append(pemCert(root), pemCert(r.inter)...), kubectl.KubernetesRelease)
	if err != nil {
		t.Fatal(err)
	}
	r.verifier = v
	r.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.asked = append(r.asked, req.URL.Path)
		body, ok := r.files[req.URL.Path]
		r.mu.Unlock()
		if !ok {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = w.Write(body)
	}))
	t.Cleanup(r.srv.Close)
	return r
}

// Mirror is the host's base URL, as the download mirror preference takes it.
func (r *KubectlReleases) Mirror() string { return r.srv.URL + "/release" }

// Downloader downloads from the host, trusting its certificate authority; the mirror is ignored.
func (r *KubectlReleases) Downloader(string) *kubectl.Downloader {
	return &kubectl.Downloader{Mirror: r.Mirror(), Client: r.srv.Client(), Verifier: r.verifier}
}

// Publish puts release version of binary for goos and goarch on the host, with its SHA-256 and a
// signature made for the Kubernetes release's identity.
func (r *KubectlReleases) Publish(t testing.TB, version, goos, goarch string, binary []byte) {
	t.Helper()
	sum := sha256.Sum256(binary)
	key := kubectlKey(t, elliptic.P256())
	v2, err := asn1.MarshalWithParams(kubectl.KubernetesRelease.Issuer, "utf8")
	if err != nil {
		t.Fatal(err)
	}
	leaf := kubectlCert(t, &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), NotBefore: time.Now().Add(-70 * time.Minute),
		NotAfter: time.Now().Add(-60 * time.Minute), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning}, EmailAddresses: []string{kubectl.KubernetesRelease.Identity},
		ExtraExtensions: []pkix.Extension{{Id: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 8}, Value: v2}},
	}, r.inter, &key.PublicKey, r.interKey)
	sig, err := ecdsa.SignASN1(rand.Reader, key, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	dir := "/release/v" + version + "/bin/" + goos + "/" + goarch + "/" + kubectl.BinaryName(goos)
	r.Set(dir, binary)
	r.Set(dir+".sha256", []byte(hex.EncodeToString(sum[:])))
	r.Set(dir+".sig", []byte(base64.StdEncoding.EncodeToString(sig)))
	r.Set(dir+".cert", []byte(base64.StdEncoding.EncodeToString(pemCert(leaf))))
}

// Set serves body at path; nil removes it.
func (r *KubectlReleases) Set(path string, body []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if body == nil {
		delete(r.files, path)
		return
	}
	r.files[path] = body
}

// Requests returns the paths asked for, in order.
func (r *KubectlReleases) Requests() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.asked...)
}

// FakeKubectl is a shell script that answers "version --client -o json" as kubectl does, with
// gitVersion, and prints its arguments otherwise.
func FakeKubectl(gitVersion string) string {
	return "#!/bin/sh\n" +
		`if [ "$1 $2 $3 $4" = "version --client -o json" ]; then` + "\n" +
		`  printf '{"clientVersion":{"major":"1","gitVersion":"` + gitVersion + `"}}\n'` + "\n" +
		"  exit 0\nfi\necho \"fake kubectl " + gitVersion + ": $*\"\n"
}

func kubectlKey(t testing.TB, c elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// kubectlCert signs tmpl with key under parent (nil: self-signed).
func kubectlCert(t testing.TB, tmpl, parent *x509.Certificate, pub *ecdsa.PublicKey, key *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()
	if parent == nil {
		parent = tmpl
	}
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

func pemCert(c *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
}
