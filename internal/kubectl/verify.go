package kubectl

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	_ "embed" // the sigstore certificate authority
	"encoding/asn1"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"sync"
)

// The Kubernetes release signs every binary with sigstore's keyless signing: cosign signs the
// SHA-256 digest of the file with a key it made for that one release, and sigstore's certificate
// authority (Fulcio) certified that key for ten minutes, for the identity Google's OIDC issuer
// vouched for. dl.k8s.io publishes the signature as <file>.sig (base64 of an ASN.1 ECDSA
// signature) and the certificate as <file>.cert (base64 of a PEM certificate)
// (https://kubernetes.io/docs/tasks/administer-cluster/verify-signed-artifacts/).
//
// Verifier checks that pair with the standard library alone: the certificate chains to the
// certificate authority Cogpit embeds (sigstore_fulcio.pem) at the time it was issued, is for
// code signing, names the release's identity and issuer, and its key made the signature over the
// digest. It does not consult sigstore's transparency log (Rekor), whose entry proves that the
// signature was made while the certificate was valid, nor the certificate's signed certificate
// timestamps (docs/security/trust-boundaries.md, H-35).

// Signer is who must have signed: the identity Fulcio wrote into the certificate's subject
// alternative name, and the OIDC issuer that vouched for it.
type Signer struct {
	Identity string
	Issuer   string
}

// KubernetesRelease is the signer of Kubernetes' release binaries, as Kubernetes documents it and
// as kubectl 1.36.1's certificate names it (darwin/arm64, read 2026-10-13).
var KubernetesRelease = Signer{
	Identity: "krel-staging@k8s-releng-prod.iam.gserviceaccount.com",
	Issuer:   "https://accounts.google.com",
}

// Fulcio's certificate extensions for the OIDC issuer: 1.1 holds the raw text (deprecated), 1.8
// the same as a DER UTF8String (https://github.com/sigstore/fulcio/blob/main/docs/oid-info.md).
var (
	oidIssuerV1 = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 1}
	oidIssuerV2 = asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 57264, 1, 8}
)

// ErrSignature marks a download whose signature or signing certificate does not verify.
var ErrSignature = errors.New("the signature does not verify")

// Verifier checks a release file's signing certificate and signature.
type Verifier struct {
	Roots         *x509.CertPool // the trusted certificate authorities
	Intermediates *x509.CertPool // the authorities' intermediates; the .cert holds only the signing certificate
	Signer        Signer
}

//go:embed sigstore_fulcio.pem
var sigstoreFulcio []byte

var defaultVerifier = sync.OnceValue(func() *Verifier {
	v, err := NewVerifier(sigstoreFulcio, KubernetesRelease)
	if err != nil {
		panic("kubectl: the embedded sigstore certificate authority does not parse: " + err.Error())
	}
	return v
})

// DefaultVerifier is the Kubernetes release's signer under sigstore's public-good certificate
// authority, which Cogpit embeds.
func DefaultVerifier() *Verifier { return defaultVerifier() }

// NewVerifier returns a Verifier for signer with the certificate authorities in pemCerts: each
// self-signed certificate is a root, every other one an intermediate. Text between the PEM blocks
// is ignored.
func NewVerifier(pemCerts []byte, signer Signer) (*Verifier, error) {
	v := &Verifier{Roots: x509.NewCertPool(), Intermediates: x509.NewCertPool(), Signer: signer}
	roots := 0
	for rest := pemCerts; ; {
		var b *pem.Block
		if b, rest = pem.Decode(rest); b == nil {
			break
		}
		if b.Type != "CERTIFICATE" {
			continue
		}
		c, err := x509.ParseCertificate(b.Bytes)
		if err != nil {
			return nil, err
		}
		if !c.IsCA {
			return nil, fmt.Errorf("%s is not a certificate authority", c.Subject)
		}
		if bytes.Equal(c.RawSubject, c.RawIssuer) && c.CheckSignatureFrom(c) == nil {
			v.Roots.AddCert(c)
			roots++
		} else {
			v.Intermediates.AddCert(c)
		}
	}
	if roots == 0 {
		return nil, errors.New("no root certificate authority")
	}
	return v, nil
}

// Certificate decodes a .cert file and checks the certificate in it: it chains to the roots at
// the moment it was issued (it is valid for minutes only, and the signature was made then), is
// for code signing, and names the signer's identity and issuer.
func (v *Verifier) Certificate(data []byte) (*x509.Certificate, error) {
	data = bytes.TrimSpace(data)
	if !bytes.HasPrefix(data, []byte("-----BEGIN")) {
		dec, err := base64.StdEncoding.DecodeString(string(data))
		if err != nil {
			return nil, fmt.Errorf("%w: the certificate is neither PEM nor base64", ErrSignature)
		}
		data = dec
	}
	b, _ := pem.Decode(data)
	if b == nil || b.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("%w: no PEM certificate in the .cert file", ErrSignature)
	}
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: the certificate does not parse: %v", ErrSignature, err)
	}
	if _, err := c.Verify(x509.VerifyOptions{
		Roots: v.Roots, Intermediates: v.Intermediates, CurrentTime: c.NotBefore,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageCodeSigning},
	}); err != nil {
		return nil, fmt.Errorf("%w: the signing certificate does not chain to sigstore's certificate authority: %v", ErrSignature, err)
	}
	if !slices.Contains(c.EmailAddresses, v.Signer.Identity) {
		return nil, fmt.Errorf("%w: the certificate names %s, not %s", ErrSignature, identities(c), v.Signer.Identity)
	}
	if issuer := oidcIssuer(c); issuer != v.Signer.Issuer {
		return nil, fmt.Errorf("%w: the certificate's OIDC issuer is %q, not %s", ErrSignature, issuer, v.Signer.Issuer)
	}
	return c, nil
}

// Verify checks a .sig file, base64 of an ASN.1 ECDSA signature, against the SHA-256 digest of
// the file with the certificate's key.
func (v *Verifier) Verify(c *x509.Certificate, digest, sig []byte) error {
	raw, err := base64.StdEncoding.DecodeString(string(bytes.TrimSpace(sig)))
	if err != nil {
		return fmt.Errorf("%w: the .sig file is not base64", ErrSignature)
	}
	pub, ok := c.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("%w: the certificate's key is %T, not ECDSA", ErrSignature, c.PublicKey)
	}
	if !ecdsa.VerifyASN1(pub, digest, raw) {
		return fmt.Errorf("%w: it was not made over this file's digest by the certificate's key", ErrSignature)
	}
	return nil
}

// oidcIssuer is the OIDC issuer Fulcio recorded in the certificate; "" when it recorded none.
func oidcIssuer(c *x509.Certificate) string {
	var v1 string
	for _, e := range c.Extensions {
		switch {
		case e.Id.Equal(oidIssuerV2):
			var s string
			if rest, err := asn1.UnmarshalWithParams(e.Value, &s, "utf8"); err == nil && len(rest) == 0 {
				return s
			}
		case e.Id.Equal(oidIssuerV1):
			v1 = string(e.Value)
		}
	}
	return v1
}

// identities lists the certificate's subject alternative names for an error, quoted: they are the
// download's text, not Cogpit's.
func identities(c *x509.Certificate) string {
	ids := make([]string, 0, len(c.EmailAddresses)+len(c.URIs))
	ids = append(ids, c.EmailAddresses...)
	for _, u := range c.URIs {
		ids = append(ids, u.String())
	}
	if len(ids) == 0 {
		return "no identity"
	}
	return fmt.Sprintf("%q", ids)
}
