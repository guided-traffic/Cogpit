package cluster

import (
	"crypto/x509"
	"errors"
)

// How a connection trusts its API server comes from the kubeconfig's cluster entry, and client-go
// applies it by itself: clientcmd copies certificate-authority-data, certificate-authority,
// tls-server-name and insecure-skip-tls-verify into the rest.Config's TLS settings and proxy-url
// into rest.Config.Proxy (client-go 0.37.1, tools/clientcmd/client_config.go). Cogpit adds what a
// user needs around it: a connect error that says which field to check when the server's
// certificate fails the check (certificateHint), and Insecure for the warnings of a connection
// that does not check it (TestConnectTrustsTheKubeconfigCA, TestConnectWithTLSServerName,
// TestInsecureSkipTLSVerifyConnectsToAnUntrustedServer).

// Insecure reports whether the connection skips the check of the API server's certificate: the
// kubeconfig's cluster sets insecure-skip-tls-verify. Its credential then goes to whoever answers
// at the server's address (docs/security/trust-boundaries.md, H-35); the engine writes a warning
// into the connection log and the picker marks such a context. A zero Conn (tests' fakes) and a
// nil one report false.
func (c *Conn) Insecure() bool { return c != nil && c.config != nil && c.config.Insecure }

// certificateHint says what to check in the kubeconfig when the API server's certificate failed
// the check, "" for any other error. It names kubeconfig fields, never their values.
func certificateHint(err error) string {
	var unknown x509.UnknownAuthorityError
	var host x509.HostnameError
	var invalid x509.CertificateInvalidError
	switch {
	case errors.As(err, &unknown):
		return "the server's certificate is not signed by a certificate authority this context trusts: " +
			"check the cluster's certificate-authority-data or certificate-authority in the kubeconfig"
	case errors.As(err, &host):
		return "the server's certificate is not issued for the host of the server URL: check the URL, " +
			"or set the cluster's tls-server-name in the kubeconfig to a name the certificate holds"
	case errors.As(err, &invalid) && invalid.Reason == x509.Expired:
		return "the server's certificate has expired or is not valid yet: check the server's certificate " +
			"and this computer's clock"
	case errors.As(err, &invalid):
		return "the server's certificate is not valid for an API server"
	}
	return ""
}
