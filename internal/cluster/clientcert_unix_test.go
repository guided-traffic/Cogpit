//go:build unix

package cluster

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// The key is never opened when the contexts are listed (D-75): client-key names a FIFO, which
// opening for reading would block on until a writer came, and listing still returns at once with
// the certificate's dates. A client-certificate that is a FIFO is refused unopened too.
func TestListingNeverOpensTheKey(t *testing.T) {
	resetCertMemo(t, false)
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{})
	defer srv.Close()
	dir := realTempDir(t)
	now := time.Now().UTC().Truncate(time.Second)
	certPath, _, cert := writeCert(t, dir, now.Add(-time.Hour), now.Add(10*24*time.Hour))
	keyPipe := filepath.Join(dir, "key.pipe")
	if err := syscall.Mkfifo(keyPipe, 0o600); err != nil {
		t.Fatal(err)
	}
	kc := filepath.Join(dir, "config")
	if err := os.WriteFile(kc, []byte(fileCertKubeconfig(srv, certPath, keyPipe)), 0o600); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		if v := inspectedCert(t, kc); v == nil || !v.NotAfter.Equal(cert.Cert.NotAfter) {
			t.Errorf("%+v, want until %v", v, cert.Cert.NotAfter)
		}
		if _, err := ListContexts([]string{kc}); err != nil {
			t.Error(err)
		}
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("listing the contexts opened the key file (a FIFO): it waits for a writer")
	}

	certPipe := filepath.Join(dir, "cert.pipe")
	if err := syscall.Mkfifo(certPipe, 0o600); err != nil {
		t.Fatal(err)
	}
	errc := make(chan error, 1)
	go func() { _, err := readCertValidity(certPipe); errc <- err }()
	select {
	case err := <-errc:
		if err == nil || err.Error() != "not a regular file" {
			t.Errorf("a FIFO certificate: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a FIFO certificate was opened")
	}
}
