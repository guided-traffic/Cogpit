package model_test

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

func pemOf(typ string, der []byte, headers map[string]string) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: typ, Headers: headers, Bytes: der})
}

// A private key in every format Go reads (PKCS #1, PKCS #8 and SEC 1, in PEM or DER; RSA, ECDSA,
// Ed25519) matches its own certificate and not another one.
func TestMatchPrivateKeyPairs(t *testing.T) {
	rsaCert := testkit.NewCert(t, testkit.CertSpec{CommonName: "rsa", Key: testkit.RSAKey(t, 2048)})
	ecCert := testkit.NewCert(t, testkit.CertSpec{CommonName: "ecdsa", Key: testkit.ECDSAKey(t)})
	edCert := testkit.NewCert(t, testkit.CertSpec{CommonName: "ed25519", Key: testkit.Ed25519Key(t)})
	rsaKey, ecKey := rsaCert.Key.(*rsa.PrivateKey), ecCert.Key.(*ecdsa.PrivateKey)
	sec1, err := x509.MarshalECPrivateKey(ecKey)
	if err != nil {
		t.Fatal(err)
	}
	pkcs8 := func(c *testkit.TestCert) []byte {
		der, err := x509.MarshalPKCS8PrivateKey(c.Key)
		if err != nil {
			t.Fatal(err)
		}
		return der
	}
	ecParams := pemOf("EC PARAMETERS", []byte{0x06, 0x08, 0x2a, 0x86, 0x48, 0xce, 0x3d, 0x03, 0x01, 0x07}, nil)
	for _, c := range []struct {
		name string
		cert *testkit.TestCert
		key  []byte
	}{
		{"RSA PKCS #1", rsaCert, pemOf("RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(rsaKey), nil)},
		{"RSA PKCS #8", rsaCert, rsaCert.KeyPEM(t)},
		{"RSA PKCS #1 DER", rsaCert, x509.MarshalPKCS1PrivateKey(rsaKey)},
		{"ECDSA SEC 1", ecCert, pemOf("EC PRIVATE KEY", sec1, nil)},
		{"ECDSA SEC 1 after its parameters", ecCert, append(append([]byte{}, ecParams...), pemOf("EC PRIVATE KEY", sec1, nil)...)},
		{"ECDSA PKCS #8", ecCert, ecCert.KeyPEM(t)},
		{"ECDSA SEC 1 DER", ecCert, sec1},
		{"Ed25519 PKCS #8", edCert, edCert.KeyPEM(t)},
		{"Ed25519 PKCS #8 DER", edCert, pkcs8(edCert)},
	} {
		d := model.DecodeCertificates(c.cert.PEM(), 1)
		if m, ok := d.MatchPrivateKey(bytes.Clone(c.key)); !ok || !m.Matches || m.Reason != "" {
			t.Errorf("%s: %+v, %v", c.name, m, ok)
		}
		other := model.DecodeCertificates(testkit.NewCert(t, testkit.CertSpec{CommonName: "other"}).PEM(), 1)
		if m, ok := other.MatchPrivateKey(bytes.Clone(c.key)); !ok || m.Matches || m.Reason != "" {
			t.Errorf("%s against another certificate: %+v, %v", c.name, m, ok)
		}
	}
}

// A key that cannot be compared says why with a fixed reason: encrypted (PKCS #8 or a legacy PEM
// header), an OpenSSH key, garbage in PEM or not, a value over the bound; a value without a first
// certificate is not compared at all.
func TestMatchPrivateKeyReasons(t *testing.T) {
	cert := testkit.NewCert(t, testkit.CertSpec{CommonName: "c"})
	d := model.DecodeCertificates(cert.PEM(), 1)
	for _, c := range []struct {
		name, reason string
		key          []byte
	}{
		{"PKCS #8 encrypted", model.KeyEncrypted, pemOf("ENCRYPTED PRIVATE KEY", []byte("0123456789abcdef"), nil)},
		{"legacy encrypted", model.KeyEncrypted, pemOf("RSA PRIVATE KEY", []byte("0123456789abcdef"), map[string]string{"Proc-Type": "4,ENCRYPTED", "DEK-Info": "AES-128-CBC,00112233445566778899AABBCCDDEEFF"})},
		{"OpenSSH", model.KeyUnsupported, pemOf("OPENSSH PRIVATE KEY", []byte("openssh-key-v1"), nil)},
		{"garbage in PEM", model.KeyNotReadable, pemOf("PRIVATE KEY", []byte("not a key at all"), nil)},
		{"a PEM without a key", model.KeyNotReadable, cert.PEM()},
		{"garbage", model.KeyNotReadable, []byte("not a key at all")},
		{"empty", model.KeyNotReadable, nil},
		{"too large", model.KeyTooLarge, bytes.Repeat([]byte("A"), model.MaxPrivateKeyBytes+1)},
	} {
		if m, ok := d.MatchPrivateKey(c.key); !ok || m.Matches || m.Reason != c.reason {
			t.Errorf("%s: %+v, %v", c.name, m, ok)
		}
	}
	invalid := model.DecodeCertificates([]byte("-----BEGIN CERTIFICATE-----\naGVsbG8=\n-----END CERTIFICATE-----\n"), 1)
	if _, ok := invalid.MatchPrivateKey(cert.KeyPEM(t)); ok || invalid.HasLeaf() {
		t.Error("a key was compared with an invalid certificate")
	}
}

// Agents read keyMatches as true, false or the reason.
func TestKeyMatchJSON(t *testing.T) {
	for want, m := range map[string]model.KeyMatch{
		`true`: {Matches: true}, `false`: {}, `"encrypted"`: {Reason: model.KeyEncrypted},
	} {
		b, err := json.Marshal(m)
		if err != nil || string(b) != want {
			t.Errorf("%+v: %s, %v", m, b, err)
		}
		var back model.KeyMatch
		if err := json.Unmarshal(b, &back); err != nil || back != m {
			t.Errorf("%s read back as %+v, %v", b, back, err)
		}
	}
	var bad model.KeyMatch
	if err := json.Unmarshal([]byte(`{}`), &bad); err == nil {
		t.Error("an object read as a key match")
	}
}

// A chain is in order when each certificate is signed by the next; a reversed one breaks at the
// first; one certificate or an invalid entry is not checked.
func TestChainInOrder(t *testing.T) {
	leaf, inter, root := testChain(t)
	check := func(name string, value []byte, inOrder, checked bool, broken int) {
		t.Helper()
		d := model.DecodeCertificates(value, 16)
		if o, c, b := d.ChainInOrder(); o != inOrder || c != checked || b != broken {
			t.Errorf("%s: in order %v, checked %v, broken at %d", name, o, c, b)
		}
	}
	check("leaf, intermediate, root", testkit.Chain(leaf, inter, root), true, true, 0)
	check("leaf, intermediate", testkit.Chain(leaf, inter), true, true, 0)
	check("root, intermediate, leaf", testkit.Chain(root, inter, leaf), false, true, 1)
	check("leaf, root", testkit.Chain(leaf, root), false, true, 1)
	check("one certificate", leaf.PEM(), false, false, 0)
	check("an invalid entry", append(testkit.Chain(leaf, inter), "-----BEGIN CERTIFICATE-----\naGVsbG8=\n-----END CERTIFICATE-----\n"...), false, false, 0)
}

// DER certificates decode like PEM ones, one after the other and bounded; DER that is no
// certificate (garbage, a private key in PKCS #1, PKCS #8 or SEC 1) holds none and is not
// described; a certificate followed by bytes that are none ends in an invalid entry.
func TestDecodeDERCertificates(t *testing.T) {
	leaf, inter, root := testChain(t)
	d := model.DecodeCertificates(bytes.Clone(root.Cert.Raw), 16)
	if !d.DER || len(d.Certificates) != 1 || d.Certificates[0].SubjectCN != "Example Root CA" {
		t.Fatalf("one DER certificate: %+v", d)
	}
	chain := bytes.Join([][]byte{leaf.Cert.Raw, inter.Cert.Raw, root.Cert.Raw}, nil)
	d = model.DecodeCertificates(chain, 16)
	if len(d.Certificates) != 3 || d.Certificates[1].SubjectCN != "Example Intermediate CA" {
		t.Fatalf("a DER chain: %+v", d.Certificates)
	}
	if inOrder, checked, _ := d.ChainInOrder(); !inOrder || !checked {
		t.Error("the DER chain is not in order")
	}
	if m, ok := d.MatchPrivateKey(leaf.KeyPEM(t)); !ok || !m.Matches {
		t.Errorf("the DER leaf's key: %+v", m)
	}
	ecKey := leaf.Key.(*ecdsa.PrivateKey)
	sec1, _ := x509.MarshalECPrivateKey(ecKey)
	pkcs8, _ := x509.MarshalPKCS8PrivateKey(ecKey)
	rsaKey := testkit.RSAKey(t, 2048).(*rsa.PrivateKey)
	for name, v := range map[string][]byte{
		"garbage": []byte("\x00\x01\x02 not DER"), "an empty SEQUENCE": {0x30, 0x00}, "an INTEGER": {0x02, 0x01, 0x05},
		"PKCS #8 key": pkcs8, "SEC 1 key": sec1, "PKCS #1 key": x509.MarshalPKCS1PrivateKey(rsaKey),
	} {
		d := model.DecodeCertificates(v, 16)
		if !d.Empty() || d.DER {
			t.Errorf("%s decoded: %+v", name, d)
		}
		if b, _ := json.Marshal(d.Certificates); string(b) != "null" {
			t.Errorf("%s: %s", name, b)
		}
	}
	d = model.DecodeCertificates(append(bytes.Clone(root.Cert.Raw), "trailing"...), 16)
	if len(d.Certificates) != 2 || d.Certificates[0].Invalid || !d.Certificates[1].Invalid {
		t.Errorf("a certificate and trailing bytes: %+v", d.Certificates)
	}
	var many []byte
	for range 20 {
		many = append(many, root.Cert.Raw...)
	}
	if d := model.DecodeCertificates(many, 16); len(d.Certificates) != 16 || d.Omitted != 4 {
		t.Errorf("20 DER certificates: %d decoded, %d omitted", len(d.Certificates), d.Omitted)
	}
	if d := model.DecodeCertificates(many, 0); len(d.Certificates) != 0 || d.Omitted != 20 {
		t.Errorf("counted only: %d decoded, %d omitted", len(d.Certificates), d.Omitted)
	}
	if strings.Contains(string(chain), "BEGIN") {
		t.Fatal("the DER fixture is PEM")
	}
}
