package agent

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// A connection's CA file (engine.ChatConnection.CAFile, the owner's decision of 2026-10-15): the
// CA certificates of an https server with a private CA, trusted for that connection beside the
// system's roots, never instead of them.
//
// How the roots are built, checked in Go 1.27.2's crypto/x509 (cert_pool.go SystemCertPool,
// root.go loadSystemRoots, verify.go Certificate.Verify): x509.SystemCertPool returns a copy of the
// system pool. On macOS (and Windows) that pool holds no certificates but is marked as the system's
// (systemPool), and Verify hands it to the platform verifier (macOS's Security framework: its trust
// store, the user's trust settings). AppendCertsFromPEM keeps the mark and adds the file's
// certificates; Verify then asks the platform verifier first and, only when it fails, verifies
// again with Go's own verifier against the added certificates alone. So a public server is still
// verified by macOS, and a server under the private CA by Go against the file. A pool from
// x509.NewCertPool would replace the system's roots instead: only the file trusted, the platform
// verifier never asked; Cogpit does not do that. On Linux the system pool is the distribution's
// bundle read from disk, and the file's certificates are added to that copy. Since Go 1.27,
// SSL_CERT_FILE or SSL_CERT_DIR in the environment make macOS load roots from those files instead
// of asking the platform verifier (unless GODEBUG=x509sslcertoverrideplatform=0), for every
// connection alike.
//
// The file is read whenever a request is sent (a turn, a connection test, the models list), so a
// changed file counts from the next; the client is kept per connection and file content, so the
// connections it opened are reused. The file may hold anything else besides certificates (only
// CERTIFICATE blocks are taken); its content is never logged or quoted in an error.

// maxCAFile bounds a CA file: a bundle of a few hundred certificates.
const maxCAFile = 1 << 20

// loadCAPool reads a CA file and returns the system's roots with its certificates added, and a
// digest of the file. Errors name the file and why, never its content.
func loadCAPool(path string) (*x509.CertPool, [sha256.Size]byte, error) {
	var sum [sha256.Size]byte
	data, err := readCAFile(path)
	if err != nil {
		return nil, sum, err
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool() // no system roots to add to: the file's alone, as Go would have none
	}
	if !pool.AppendCertsFromPEM(data) {
		return nil, sum, fmt.Errorf("the connection's CA file %s holds no PEM certificate (-----BEGIN CERTIFICATE-----)", path)
	}
	return pool, sha256.Sum256(data), nil
}

// readCAFile reads a CA file: a regular file (a symlink to one is fine) of at most maxCAFile.
func readCAFile(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("the connection's CA file %s is not an absolute path", path)
	}
	f, err := os.Open(path) // the file the user named for this connection, in the window only
	if err != nil {
		return nil, caFileError(path, err)
	}
	defer func() { _ = f.Close() }()
	fi, err := f.Stat()
	if err != nil {
		return nil, caFileError(path, err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("the connection's CA file %s is not a file", path)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxCAFile+1))
	if err != nil {
		return nil, caFileError(path, err)
	}
	if len(data) > maxCAFile {
		return nil, fmt.Errorf("the connection's CA file %s is larger than 1 MiB", path)
	}
	return data, nil
}

// caFileError names why a CA file could not be read, without repeating the path in Go's words.
func caFileError(path string, err error) error {
	var pe *fs.PathError
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("the connection's CA file %s does not exist", path)
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("the connection's CA file %s may not be read (permission denied)", path)
	case errors.As(err, &pe):
		return fmt.Errorf("the connection's CA file %s could not be read: %v", path, pe.Err)
	}
	return fmt.Errorf("the connection's CA file %s could not be read: %v", path, err)
}

// caClient is a connection's HTTP client with its CA file's certificates, and the file's digest it
// was built from.
type caClient struct {
	sum [sha256.Size]byte
	hc  *http.Client
}

// httpClientFor returns the client a connection's requests go through: the agent's own, or, for a
// connection with a CA file, one whose TLS trusts the file's certificates beside the system's
// roots. It reads the file on every call.
func (a *Agent) httpClientFor(c engine.ChatConnection) (*http.Client, error) {
	if c.CAFile == "" {
		return a.hc, nil
	}
	pool, sum, err := loadCAPool(c.CAFile)
	if err != nil {
		return nil, err
	}
	key := c.ID + "\x00" + c.CAFile
	a.mu.Lock()
	defer a.mu.Unlock()
	if old, ok := a.caClients[key]; ok {
		if bytes.Equal(old.sum[:], sum[:]) {
			return old.hc, nil
		}
		old.hc.CloseIdleConnections() // the file changed: its old trust goes with its connections
	}
	rt := a.hc.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	tr, ok := rt.(*http.Transport)
	if !ok {
		return nil, errors.New("a connection's CA file needs Cogpit's own HTTP transport")
	}
	tr = tr.Clone()
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	if tr.TLSClientConfig != nil {
		cfg = tr.TLSClientConfig.Clone()
		cfg.MinVersion = max(cfg.MinVersion, tls.VersionTLS12)
	}
	cfg.RootCAs = pool
	tr.TLSClientConfig = cfg
	hc := &http.Client{Transport: tr, CheckRedirect: a.hc.CheckRedirect, Timeout: a.hc.Timeout}
	a.caClients[key] = caClient{sum: sum, hc: hc}
	return hc, nil
}
