// Package sigstoretest is a sigstore instance of its own for tests: a certificate authority shaped
// like Fulcio's, and a transparency log shaped like Rekor v1 with a key of its own, whose search
// (POST /api/v1/log/entries/retrieve) it answers as Rekor's API does. Its TrustedMaterial trusts
// both, so a sigstore-go verifier built on it checks what it signs and logs as the default
// verifier checks a Kubernetes release, except for signed certificate timestamps: it has no
// certificate transparency log. Nothing reaches the internet.
//
// It does not import internal/kubectl, whose own tests use it.
package sigstoretest

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/sigstore/rekor/pkg/util"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore/pkg/signature"
)

// LogHost is the log's name in its checkpoints.
const LogHost = "rekor.localhost"

// Leaf shapes a signing certificate. Empty Identity and Issuer are the Kubernetes release's
// (krel-staging@k8s-releng-prod.iam.gserviceaccount.com, https://accounts.google.com).
type Leaf struct {
	Identity, Issuer string
	NoIssuer         bool
	EKU              x509.ExtKeyUsage // 0: code signing
}

// Sigstore is the instance. Safe for concurrent use.
type Sigstore struct {
	root, inter *x509.Certificate
	interKey    *ecdsa.PrivateKey
	logKey      *ecdsa.PrivateKey
	logID       []byte
	trusted     *root.TrustedRoot

	mu      sync.Mutex
	entries map[string]map[string]any // by lookupKey: uuid → the API's entry
	index   int64
	asked   int
}

// New makes an instance: a root and an intermediate valid from two days ago, and a log key.
func New(t testing.TB) *Sigstore {
	t.Helper()
	s := &Sigstore{entries: map[string]map[string]any{}, index: 1000}
	rootKey := key(t, elliptic.P384())
	s.root = cert(t, &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{Organization: []string{"sigstoretest"}, CommonName: "sigstoretest"},
		NotBefore: time.Now().Add(-48 * time.Hour), NotAfter: time.Now().Add(48 * time.Hour),
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, BasicConstraintsValid: true, IsCA: true,
	}, nil, &rootKey.PublicKey, rootKey)
	s.interKey = key(t, elliptic.P384())
	s.inter = cert(t, &x509.Certificate{
		SerialNumber: big.NewInt(2), Subject: pkix.Name{Organization: []string{"sigstoretest"}, CommonName: "sigstoretest-intermediate"},
		NotBefore: time.Now().Add(-47 * time.Hour), NotAfter: time.Now().Add(47 * time.Hour),
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
		BasicConstraintsValid: true, IsCA: true, MaxPathLenZero: true,
	}, s.root, &s.interKey.PublicKey, rootKey)
	s.logKey = key(t, elliptic.P256())
	der, err := x509.MarshalPKIXPublicKey(&s.logKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	id := sha256.Sum256(der)
	s.logID = id[:]
	start := time.Now().Add(-48 * time.Hour)
	s.trusted, err = root.NewTrustedRoot(root.TrustedRootMediaType01,
		[]root.CertificateAuthority{&root.FulcioCertificateAuthority{
			Root: s.root, Intermediates: []*x509.Certificate{s.inter}, ValidityPeriodStart: start, URI: "https://fulcio.localhost",
		}},
		nil, nil,
		map[string]*root.TransparencyLog{hex.EncodeToString(s.logID): {
			BaseURL: "https://" + LogHost, ID: s.logID, ValidityPeriodStart: start, HashFunc: crypto.SHA256,
			PublicKey: &s.logKey.PublicKey, SignatureHashFunc: crypto.SHA256,
		}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// TrustedMaterial trusts the instance's certificate authority and log.
func (s *Sigstore) TrustedMaterial() root.TrustedMaterial { return s.trusted }

// Issue makes a ten-minute signing certificate that ended an hour ago, as a release's has, and
// returns it as a .cert file (base64 of PEM) with its key.
func (s *Sigstore) Issue(t testing.TB, l Leaf) ([]byte, *ecdsa.PrivateKey) {
	t.Helper()
	if l.Identity == "" {
		l.Identity = "krel-staging@k8s-releng-prod.iam.gserviceaccount.com"
	}
	if l.Issuer == "" {
		l.Issuer = "https://accounts.google.com"
	}
	if l.EKU == 0 {
		l.EKU = x509.ExtKeyUsageCodeSigning
	}
	k := key(t, elliptic.P256())
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()), NotBefore: time.Now().Add(-70 * time.Minute),
		NotAfter: time.Now().Add(-60 * time.Minute), KeyUsage: x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{l.EKU}, EmailAddresses: []string{l.Identity},
	}
	if !l.NoIssuer {
		v2, err := asn1.MarshalWithParams(l.Issuer, "utf8")
		if err != nil {
			t.Fatal(err)
		}
		tmpl.ExtraExtensions = []pkix.Extension{
			{Id: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}, Value: []byte(l.Issuer)},
			{Id: asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 8}, Value: v2},
		}
	}
	c := cert(t, tmpl, s.inter, &k.PublicKey, s.interKey)
	return []byte(base64.StdEncoding.EncodeToString(pemOf(c))), k
}

// SignFile signs digest as cosign does and returns the .sig file.
func SignFile(t testing.TB, k *ecdsa.PrivateKey, digest []byte) []byte {
	t.Helper()
	sig, err := ecdsa.SignASN1(rand.Reader, k, digest)
	if err != nil {
		t.Fatal(err)
	}
	return []byte(base64.StdEncoding.EncodeToString(sig) + "\n")
}

// Release signs digest for l, as the release's cosign does, and logs the signature: it returns
// the .sig and .cert files.
func (s *Sigstore) Release(t testing.TB, digest []byte, l Leaf) (sigFile, certFile []byte) {
	t.Helper()
	certFile, k := s.Issue(t, l)
	sigFile = SignFile(t, k, digest)
	s.Log(t, digest, sigFile, certFile)
	return sigFile, certFile
}

// Log records the hashedrekord entry of digest, the .sig and the .cert, as cosign uploads it: the
// log's signed entry timestamp a minute into the certificate's validity, and an inclusion proof in
// a tree of two leaves whose checkpoint the log signed. It returns the entry as the API answers it.
func (s *Sigstore) Log(t testing.TB, digest, sigFile, certFile []byte) map[string]any {
	t.Helper()
	sig, c := decode(t, sigFile, certFile)
	body, err := json.Marshal(map[string]any{
		"apiVersion": "0.0.1", "kind": "hashedrekord",
		"spec": map[string]any{
			"data": map[string]any{"hash": map[string]any{"algorithm": "sha256", "value": hex.EncodeToString(digest)}},
			"signature": map[string]any{
				"content":   base64.StdEncoding.EncodeToString(sig),
				"publicKey": map[string]any{"content": base64.StdEncoding.EncodeToString(pemOf(c))},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	s.mu.Lock()
	s.index++
	index := s.index
	s.mu.Unlock()
	integrated := c.NotBefore.Add(time.Minute).Unix()
	b64 := base64.StdEncoding.EncodeToString(body)
	payload, err := json.Marshal(map[string]any{ // keys sorted: the canonical form the SET signs
		"body": b64, "integratedTime": integrated, "logID": hex.EncodeToString(s.logID), "logIndex": index,
	})
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(payload)
	set, err := ecdsa.SignASN1(rand.Reader, s.logKey, h[:])
	if err != nil {
		t.Fatal(err)
	}
	// A tree of two leaves, another entry's and this one: the proof is the other leaf's hash.
	other := leafHash([]byte(strconv.FormatInt(index, 10)))
	leaf := leafHash(body)
	rootHash := nodeHash(other, leaf)
	signer, err := signature.LoadECDSASignerVerifier(s.logKey, crypto.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	checkpoint, err := util.CreateAndSignCheckpoint(context.Background(), LogHost, 123, 2, rootHash, signer)
	if err != nil {
		t.Fatal(err)
	}
	entry := map[string]any{
		"body": b64, "integratedTime": integrated, "logID": hex.EncodeToString(s.logID), "logIndex": index,
		"verification": map[string]any{
			"signedEntryTimestamp": base64.StdEncoding.EncodeToString(set),
			"inclusionProof": map[string]any{
				"checkpoint": string(checkpoint), "hashes": []string{hex.EncodeToString(other)},
				"logIndex": 1, "rootHash": hex.EncodeToString(rootHash), "treeSize": 2,
			},
		},
	}
	uuid := hex.EncodeToString(leaf)
	s.mu.Lock()
	s.entries[lookupKey(digest, sig, c)] = map[string]any{uuid: entry}
	s.mu.Unlock()
	return entry
}

// Asked is how many searches the log answered.
func (s *Sigstore) Asked() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.asked
}

// ServeHTTP answers POST /api/v1/log/entries/retrieve as Rekor v1 does: 400 for an entry whose
// signature does not verify over its digest, else the entries it has, [] when none.
func (s *Sigstore) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || r.URL.Path != "/api/v1/log/entries/retrieve" {
		http.NotFound(w, r)
		return
	}
	s.mu.Lock()
	s.asked++
	s.mu.Unlock()
	var q struct {
		Entries []struct {
			Spec struct {
				Data struct {
					Hash struct{ Algorithm, Value string } `json:"hash"`
				} `json:"data"`
				Signature struct {
					Content   []byte `json:"content"`
					PublicKey struct {
						Content []byte `json:"content"`
					} `json:"publicKey"`
				} `json:"signature"`
			} `json:"spec"`
		} `json:"entries"`
	}
	fail := func(msg string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"code": 400, "message": msg})
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&q); err != nil {
		fail("decoding: " + err.Error())
		return
	}
	found := []map[string]any{}
	for _, e := range q.Entries {
		digest, err := hex.DecodeString(e.Spec.Data.Hash.Value)
		b, _ := pem.Decode(e.Spec.Signature.PublicKey.Content)
		if err != nil || e.Spec.Data.Hash.Algorithm != "sha256" || b == nil {
			fail("invalid entry")
			return
		}
		c, err := x509.ParseCertificate(b.Bytes)
		pub, ok := certKey(c, err)
		if !ok || !ecdsa.VerifyASN1(pub, digest, e.Spec.Signature.Content) {
			fail("verifying signature: invalid signature when validating ASN.1 encoded signature")
			return
		}
		s.mu.Lock()
		m, ok := s.entries[lookupKey(digest, e.Spec.Signature.Content, c)]
		s.mu.Unlock()
		if ok {
			found = append(found, m)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(found)
}

func certKey(c *x509.Certificate, err error) (*ecdsa.PublicKey, bool) {
	if err != nil {
		return nil, false
	}
	pub, ok := c.PublicKey.(*ecdsa.PublicKey)
	return pub, ok
}

func lookupKey(digest, sig []byte, c *x509.Certificate) string {
	fp := sha256.Sum256(c.Raw)
	return hex.EncodeToString(digest) + "|" + base64.StdEncoding.EncodeToString(sig) + "|" + hex.EncodeToString(fp[:])
}

func decode(t testing.TB, sigFile, certFile []byte) ([]byte, *x509.Certificate) {
	t.Helper()
	sig, err := base64.StdEncoding.DecodeString(string(trim(sigFile)))
	if err != nil {
		t.Fatal(err)
	}
	p, err := base64.StdEncoding.DecodeString(string(trim(certFile)))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := pem.Decode(p)
	if b == nil {
		t.Fatal("no PEM in the .cert file")
	}
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return sig, c
}

func trim(b []byte) []byte {
	for len(b) > 0 && (b[len(b)-1] == '\n' || b[len(b)-1] == ' ') {
		b = b[:len(b)-1]
	}
	return b
}

// leafHash and nodeHash are RFC 6962's.
func leafHash(data []byte) []byte {
	h := sha256.Sum256(append([]byte{0}, data...))
	return h[:]
}

func nodeHash(l, r []byte) []byte {
	h := sha256.Sum256(append(append([]byte{1}, l...), r...))
	return h[:]
}

func key(t testing.TB, c elliptic.Curve) *ecdsa.PrivateKey {
	t.Helper()
	k, err := ecdsa.GenerateKey(c, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// cert signs tmpl with k under parent (nil: self-signed).
func cert(t testing.TB, tmpl, parent *x509.Certificate, pub *ecdsa.PublicKey, k *ecdsa.PrivateKey) *x509.Certificate {
	t.Helper()
	if parent == nil {
		parent = tmpl
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, parent, pub, k)
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
