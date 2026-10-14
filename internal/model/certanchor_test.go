package model_test

import (
	"crypto/x509"
	"encoding/pem"
	"testing"

	"golang.org/x/crypto/x509roots/fallback/bundle"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestCertAnchor (R1): a chain to a root of the public program is public, whatever the computer
// trusts; one to a root only the computer's pool holds is the computer's; any other is internal,
// named by the top certificate's issuer.
func TestCertAnchor(t *testing.T) {
	var public *x509.Certificate
	for r := range bundle.Roots() {
		if r.Constraint != nil {
			continue
		}
		if c, err := x509.ParseCertificate(r.Certificate); err == nil {
			public = c
			break
		}
	}
	if public == nil {
		t.Fatal("no public root in the bundle")
	}
	pemOf := func(der []byte) []byte { return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}) }
	a, ok := model.DecodeCertificates(pemOf(public.Raw), 10).Anchor(nil)
	if !ok || a.Kind != model.AnchorPublic || (a.Root != public.Subject.CommonName && a.Root != public.Subject.String()) {
		t.Errorf("a public root: %+v, %v", a, ok)
	}

	ca := testkit.NewCert(t, testkit.CertSpec{CommonName: "Corp Root", CA: true})
	inter := testkit.NewCert(t, testkit.CertSpec{CommonName: "Corp Issuing", CA: true, Parent: ca})
	leaf := testkit.NewCert(t, testkit.CertSpec{CommonName: "web.corp", Parent: inter})
	chain := testkit.Chain(leaf, inter)
	if a, ok := model.DecodeCertificates(chain, 10).Anchor(nil); !ok || a.Kind != model.AnchorInternal || a.Root != "Corp Root" {
		t.Errorf("an internal chain: %+v, %v", a, ok)
	}
	system := x509.NewCertPool()
	system.AddCert(ca.Cert)
	if a, ok := model.DecodeCertificates(chain, 10).Anchor(system); !ok || a.Kind != model.AnchorSystem || a.Root != "Corp Root" {
		t.Errorf("a chain to a root the computer trusts: %+v, %v", a, ok)
	}
	self := testkit.NewCert(t, testkit.CertSpec{CommonName: "self.example"})
	if a, ok := model.DecodeCertificates(testkit.Chain(self), 10).Anchor(system); !ok || a.Kind != model.AnchorInternal || a.Root != "self.example" {
		t.Errorf("a self-signed certificate: %+v, %v", a, ok)
	}
	if _, ok := model.DecodeCertificates([]byte("no certificate"), 10).Anchor(nil); ok {
		t.Error("a value without a certificate has an anchor")
	}
}
