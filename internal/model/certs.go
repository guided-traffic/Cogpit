package model

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// The certificates a Secret holds, decoded (D-30, owner 2026-10-13). A certificate is public
// material: its fields reach the drawer and agents, while the PEM text, the value it came from and
// any private key never do. Only PEM blocks of type CERTIFICATE are decoded; the parser hands
// pem.Decode nothing but such a block, so a PRIVATE KEY block next to it is never decoded, let
// alone parsed. Pure: no I/O.

// Bounds on what is decoded (Cogpit's defaults, not the owner's).
const (
	// MaxCertificatesPerKey is how many certificates of one value are decoded; the rest are counted.
	MaxCertificatesPerKey = 16
	// CertificateWarnBefore is how long before its end a certificate's validity turns to a warning.
	CertificateWarnBefore = 30 * 24 * time.Hour
	// maxCertText caps each text taken from a certificate (a name, a SAN), in bytes.
	maxCertText = 1024
	// maxCertNames caps each list of names (DNS names, IP addresses, URIs, e-mail addresses).
	maxCertNames = 100
)

// NotACertificate is what an invalid PEM CERTIFICATE block shows; the block is never quoted.
const NotACertificate = "not a valid certificate"

const (
	pemCertBegin = "-----BEGIN CERTIFICATE-----"
	pemCertEnd   = "-----END CERTIFICATE-----"
	pemAnyBegin  = "-----BEGIN "
)

// Certificate is what Cogpit shows of one X.509 certificate: its public fields as text, its times
// in UTC and, once Evaluate ran, its validity at a moment. It holds nothing of the PEM text or the
// DER bytes. The JSON shape is part of MCP's object_get result (DetailModel.Certificates); fields
// are only ever added, and every added field is optional. An invalid block marshals as
// {"invalid": true, "error": "not a valid certificate"}.
type Certificate struct {
	// Invalid is true for a PEM CERTIFICATE block that does not decode or parse; no other field is
	// set.
	Invalid bool `json:"invalid,omitempty"`
	// SubjectCN and IssuerCN are the common names; Subject and Issuer the whole distinguished names
	// in RFC 2253 form ("CN=web,O=Example,C=DE").
	SubjectCN string `json:"subjectCommonName,omitempty"`
	Subject   string `json:"subject"`
	IssuerCN  string `json:"issuerCommonName,omitempty"`
	Issuer    string `json:"issuer"`
	// SerialNumber is hex, its bytes separated by colons ("0A:1B:2C"); a negative one starts "-".
	SerialNumber string `json:"serialNumber"`
	// NotBefore and NotAfter are UTC (RFC 3339 in JSON).
	NotBefore time.Time `json:"notBefore"`
	NotAfter  time.Time `json:"notAfter"`
	// ExpiresInDays is the whole days left until NotAfter at the moment Evaluate was given, rounded
	// down: 0 on the last day, negative once expired.
	ExpiresInDays int  `json:"expiresInDays"`
	Expired       bool `json:"expired"`
	NotYetValid   bool `json:"notYetValid,omitempty"`
	// Status is ok while valid with more than 30 days left, warning with 30 days or fewer, error
	// when expired or not yet valid (Cogpit's thresholds).
	Status Status `json:"status,omitempty"`
	// The subject alternative names, at most 100 of each (NamesOmitted counts the rest).
	DNSNames       []string `json:"dnsNames,omitempty"`
	IPAddresses    []string `json:"ipAddresses,omitempty"`
	URIs           []string `json:"uris,omitempty"`
	EmailAddresses []string `json:"emailAddresses,omitempty"`
	NamesOmitted   int      `json:"namesOmitted,omitempty"`
	// PublicKey is the key's algorithm with its size or curve: "RSA 2048", "ECDSA P-256", "Ed25519".
	PublicKey          string `json:"publicKey"`
	SignatureAlgorithm string `json:"signatureAlgorithm"`
	// IsCA is the basic constraints' CA flag; MaxPathLen a CA's path length limit, absent without.
	IsCA       bool `json:"isCA"`
	MaxPathLen *int `json:"maxPathLen,omitempty"`
	// KeyUsage and ExtKeyUsage name the usages ("Digital Signature", "Server Authentication"); an
	// extended usage Cogpit has no name for is its object identifier.
	KeyUsage    []string `json:"keyUsage,omitempty"`
	ExtKeyUsage []string `json:"extKeyUsage,omitempty"`
	// SelfSigned is true when the issuer is the subject and the certificate's own key verifies its
	// signature (for a SHA-1 signature, which Go no longer verifies: when no other key is named).
	SelfSigned bool `json:"selfSigned"`
	// SHA256Fingerprint is the SHA-256 of the DER bytes, hex with colons.
	SHA256Fingerprint string `json:"sha256Fingerprint"`
	SubjectKeyID      string `json:"subjectKeyId,omitempty"`
	AuthorityKeyID    string `json:"authorityKeyId,omitempty"`
}

// MarshalJSON writes an invalid certificate as {"invalid": true, "error": ...}, without the zero
// values of the fields it does not have.
func (c Certificate) MarshalJSON() ([]byte, error) {
	if c.Invalid {
		return json.Marshal(struct {
			Invalid bool   `json:"invalid"`
			Error   string `json:"error"`
		}{true, NotACertificate})
	}
	type plain Certificate
	return json.Marshal(plain(c))
}

// CertificateKey is one key of a Secret whose value holds certificates, and what was decoded of it.
type CertificateKey struct {
	Key string `json:"key"`
	// Certificates are the value's certificates in their order: a chain's leaf first, then its
	// intermediates and CA, as the value lists them.
	Certificates []Certificate `json:"certificates,omitempty"`
	// Omitted counts the certificates after the first MaxCertificatesPerKey, which are not decoded.
	Omitted int `json:"omitted,omitempty"`
	// NotDecoded says why nothing of the value was decoded (too large, or no PEM certificate where
	// one belongs); Certificates is then empty.
	NotDecoded string `json:"notDecoded,omitempty"`
}

// HoldsCertificate reports whether a value holds a PEM CERTIFICATE block, without decoding it.
func HoldsCertificate(data []byte) bool {
	return bytes.Contains(data, []byte(pemCertBegin))
}

// CountCertificates counts a value's PEM CERTIFICATE blocks, without decoding them.
func CountCertificates(data []byte) int {
	return bytes.Count(data, []byte(pemCertBegin))
}

// ParseCertificates decodes the PEM CERTIFICATE blocks of a value, in order, at most limit of
// them (MaxCertificatesPerKey when limit is 0 or more than that), and counts the blocks after
// those. Every other block (a private key, a certificate request) is skipped without being
// decoded: pem.Decode only ever sees the text from a CERTIFICATE block's BEGIN line to its END
// line, and a block that holds another BEGIN line before its END is invalid. A block that does not
// decode or parse is an Invalid entry.
func ParseCertificates(data []byte, limit int) (certs []Certificate, omitted int) {
	if limit <= 0 || limit > MaxCertificatesPerKey {
		limit = MaxCertificatesPerKey
	}
	begin, end, other := []byte(pemCertBegin), []byte(pemCertEnd), []byte(pemAnyBegin)
	rest := data
	for {
		i := bytes.Index(rest, begin)
		if i < 0 {
			return certs, omitted
		}
		rest = rest[i:]
		if len(certs) == limit {
			return certs, bytes.Count(rest, begin)
		}
		body := rest[len(begin):]
		j := bytes.Index(body, end)
		if k := bytes.Index(body, other); j < 0 || (k >= 0 && k < j) {
			// No END line, or another block begins first: not a certificate; go on from that block.
			certs = append(certs, Certificate{Invalid: true})
			if k < 0 {
				return certs, omitted
			}
			rest = body[k:]
			continue
		}
		n := len(begin) + j + len(end)
		certs = append(certs, parseBlock(rest[:n]))
		rest = rest[n:]
	}
}

// parseBlock decodes and parses one PEM CERTIFICATE block (BEGIN line to END line).
func parseBlock(block []byte) Certificate {
	b, _ := pem.Decode(block)
	if b == nil || b.Type != "CERTIFICATE" {
		return Certificate{Invalid: true}
	}
	c, err := x509.ParseCertificate(b.Bytes)
	if err != nil {
		return Certificate{Invalid: true}
	}
	return DescribeCertificate(c)
}

// DescribeCertificate returns what Cogpit shows of a parsed certificate (not yet evaluated).
func DescribeCertificate(c *x509.Certificate) Certificate {
	fp := sha256.Sum256(c.Raw)
	out := Certificate{
		SubjectCN:          certText(c.Subject.CommonName),
		Subject:            certText(distinguishedName(c.RawSubject, c.Subject)),
		IssuerCN:           certText(c.Issuer.CommonName),
		Issuer:             certText(distinguishedName(c.RawIssuer, c.Issuer)),
		SerialNumber:       serialHex(c),
		NotBefore:          c.NotBefore.UTC(),
		NotAfter:           c.NotAfter.UTC(),
		PublicKey:          publicKeyText(c),
		SignatureAlgorithm: c.SignatureAlgorithm.String(),
		IsCA:               c.BasicConstraintsValid && c.IsCA,
		KeyUsage:           keyUsageNames(c.KeyUsage),
		ExtKeyUsage:        extKeyUsageNames(c.ExtKeyUsage, c.UnknownExtKeyUsage),
		SelfSigned:         selfSigned(c),
		SHA256Fingerprint:  hexColons(fp[:]),
		SubjectKeyID:       hexColons(c.SubjectKeyId),
		AuthorityKeyID:     hexColons(c.AuthorityKeyId),
	}
	if out.IsCA && (c.MaxPathLen > 0 || (c.MaxPathLen == 0 && c.MaxPathLenZero)) {
		n := c.MaxPathLen
		out.MaxPathLen = &n
	}
	var dropped int
	names := func(in []string) []string {
		if len(in) == 0 {
			return nil
		}
		if len(in) > maxCertNames {
			dropped += len(in) - maxCertNames
			in = in[:maxCertNames]
		}
		out := make([]string, len(in))
		for i, s := range in {
			out[i] = certText(s)
		}
		return out
	}
	out.DNSNames = names(c.DNSNames)
	ips := make([]string, len(c.IPAddresses))
	for i, ip := range c.IPAddresses {
		ips[i] = ip.String()
	}
	out.IPAddresses = names(ips)
	uris := make([]string, len(c.URIs))
	for i, u := range c.URIs {
		uris[i] = u.String()
	}
	out.URIs = names(uris)
	out.EmailAddresses = names(c.EmailAddresses)
	out.NamesOmitted = dropped
	return out
}

// Evaluate sets the certificate's validity at now: the days left, expired, not yet valid and the
// status (Cogpit's thresholds: a warning at 30 days or fewer left, an error when expired or not yet
// valid). An invalid entry stays as it is.
func (c *Certificate) Evaluate(now time.Time) {
	if c.Invalid {
		return
	}
	left := c.NotAfter.Sub(now)
	c.ExpiresInDays = int(math.Floor(left.Hours() / 24))
	c.Expired = now.After(c.NotAfter)
	c.NotYetValid = now.Before(c.NotBefore)
	switch {
	case c.Expired || c.NotYetValid:
		c.Status = StatusError
	case left <= CertificateWarnBefore:
		c.Status = StatusWarning
	default:
		c.Status = StatusOK
	}
}

// distinguishedName renders a name in RFC 2253 form from its raw sequence, which keeps the order
// and the attributes the certificate has; the parsed name when the raw one does not parse.
func distinguishedName(raw []byte, parsed pkix.Name) string {
	var seq pkix.RDNSequence
	if rest, err := asn1.Unmarshal(raw, &seq); err == nil && len(rest) == 0 {
		return seq.String()
	}
	return parsed.String()
}

// serialHex renders a serial number as hex bytes separated by colons.
func serialHex(c *x509.Certificate) string {
	if c.SerialNumber == nil {
		return ""
	}
	b := c.SerialNumber.Bytes()
	if len(b) == 0 {
		b = []byte{0}
	}
	if c.SerialNumber.Sign() < 0 {
		return "-" + hexColons(b)
	}
	return hexColons(b)
}

// hexColons renders bytes as upper-case hex pairs separated by colons; "" for none.
func hexColons(b []byte) string {
	const digits = "0123456789ABCDEF"
	out := make([]byte, 0, 3*len(b))
	for i, x := range b {
		if i > 0 {
			out = append(out, ':')
		}
		out = append(out, digits[x>>4], digits[x&0x0f])
	}
	return string(out)
}

// publicKeyText names the certificate's public key: algorithm and size or curve.
func publicKeyText(c *x509.Certificate) string {
	switch k := c.PublicKey.(type) {
	case *rsa.PublicKey:
		if k.N != nil {
			return "RSA " + strconv.Itoa(k.N.BitLen())
		}
	case *ecdsa.PublicKey:
		if k.Curve != nil {
			return "ECDSA " + k.Curve.Params().Name
		}
	case ed25519.PublicKey:
		return "Ed25519"
	case *ecdh.PublicKey:
		if k.Curve() == ecdh.X25519() {
			return "X25519"
		}
	}
	return c.PublicKeyAlgorithm.String()
}

// selfSigned reports whether a certificate names itself as its issuer and its own key verifies its
// signature. Go refuses SHA-1 signatures; such a certificate counts as self-signed when it names
// no other authority key.
func selfSigned(c *x509.Certificate) bool {
	if !bytes.Equal(c.RawSubject, c.RawIssuer) {
		return false
	}
	err := c.CheckSignature(c.SignatureAlgorithm, c.RawTBSCertificate, c.Signature)
	if err == nil {
		return true
	}
	var insecure x509.InsecureAlgorithmError
	return errors.As(err, &insecure) && (len(c.AuthorityKeyId) == 0 || bytes.Equal(c.AuthorityKeyId, c.SubjectKeyId))
}

// keyUsages are the key usage bits by their names in RFC 5280's order.
var keyUsages = []struct {
	bit  x509.KeyUsage
	name string
}{
	{x509.KeyUsageDigitalSignature, "Digital Signature"},
	{x509.KeyUsageContentCommitment, "Content Commitment"},
	{x509.KeyUsageKeyEncipherment, "Key Encipherment"},
	{x509.KeyUsageDataEncipherment, "Data Encipherment"},
	{x509.KeyUsageKeyAgreement, "Key Agreement"},
	{x509.KeyUsageCertSign, "Certificate Sign"},
	{x509.KeyUsageCRLSign, "CRL Sign"},
	{x509.KeyUsageEncipherOnly, "Encipher Only"},
	{x509.KeyUsageDecipherOnly, "Decipher Only"},
}

func keyUsageNames(u x509.KeyUsage) []string {
	var out []string
	for _, k := range keyUsages {
		if u&k.bit != 0 {
			out = append(out, k.name)
		}
	}
	return out
}

// extKeyUsages names the extended key usages crypto/x509 knows.
var extKeyUsages = map[x509.ExtKeyUsage]string{
	x509.ExtKeyUsageAny:                            "Any",
	x509.ExtKeyUsageServerAuth:                     "Server Authentication",
	x509.ExtKeyUsageClientAuth:                     "Client Authentication",
	x509.ExtKeyUsageCodeSigning:                    "Code Signing",
	x509.ExtKeyUsageEmailProtection:                "E-mail Protection",
	x509.ExtKeyUsageIPSECEndSystem:                 "IPsec End System",
	x509.ExtKeyUsageIPSECTunnel:                    "IPsec Tunnel",
	x509.ExtKeyUsageIPSECUser:                      "IPsec User",
	x509.ExtKeyUsageTimeStamping:                   "Time Stamping",
	x509.ExtKeyUsageOCSPSigning:                    "OCSP Signing",
	x509.ExtKeyUsageMicrosoftServerGatedCrypto:     "Microsoft Server Gated Crypto",
	x509.ExtKeyUsageNetscapeServerGatedCrypto:      "Netscape Server Gated Crypto",
	x509.ExtKeyUsageMicrosoftCommercialCodeSigning: "Microsoft Commercial Code Signing",
	x509.ExtKeyUsageMicrosoftKernelCodeSigning:     "Microsoft Kernel Code Signing",
}

func extKeyUsageNames(known []x509.ExtKeyUsage, unknown []asn1.ObjectIdentifier) []string {
	var out []string
	for _, u := range known {
		if name, ok := extKeyUsages[u]; ok {
			out = append(out, name)
		} else {
			out = append(out, "usage "+strconv.Itoa(int(u)))
		}
	}
	for _, oid := range unknown {
		out = append(out, oid.String())
	}
	return out
}

// certText makes a text taken from a certificate safe to show on one line: control characters
// and the bidirectional overrides become \x / \u escapes (a name with a line break would read as
// two names, an override reorders what follows), invalid UTF-8 becomes U+FFFD, and it is cut at
// 1 KiB with "…".
func certText(s string) string {
	clean := true
	for _, r := range s {
		if r == utf8.RuneError || unsafeRune(r) {
			clean = false
			break
		}
	}
	if !clean {
		var b strings.Builder
		for _, r := range strings.ToValidUTF8(s, "\uFFFD") {
			switch {
			case r < 0x80 && unsafeRune(r):
				b.WriteString(`\x`)
				b.WriteString(hex2(byte(r)))
			case unsafeRune(r):
				q := strconv.QuoteRuneToASCII(r)
				b.WriteString(q[1 : len(q)-1])
			default:
				b.WriteRune(r)
			}
		}
		s = b.String()
	}
	if len(s) > maxCertText {
		cut := maxCertText
		for cut > 0 && !utf8.RuneStart(s[cut]) {
			cut--
		}
		s = s[:cut] + "…"
	}
	return s
}

// unsafeRune reports whether a rune must not reach a drawer line as it is.
func unsafeRune(r rune) bool {
	switch {
	case unicode.IsControl(r):
		return true
	case r >= 0x202a && r <= 0x202e, r >= 0x2066 && r <= 0x2069, r == 0x200e, r == 0x200f,
		r == 0x2028, r == 0x2029:
		return true
	}
	return false
}

func hex2(b byte) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[b>>4], digits[b&0x0f]})
}
