package testkit

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/guided-traffic/cogpit/internal/kubectl"
	"github.com/guided-traffic/cogpit/internal/testkit/sigstoretest"
)

// KubectlReleases is a local release host for kubectl downloads: an https server that lays out
// releases as dl.k8s.io does (/release/v<version>/bin/<os>/<arch>/kubectl with .sha256, .sig and
// .cert), signed by a sigstore instance of its own (sigstoretest) for the Kubernetes release's
// identity, and answers that instance's transparency-log search under /api/v1/ as Rekor does.
// Tests download from it; nothing reaches the internet.
type KubectlReleases struct {
	srv      *httptest.Server
	sigstore *sigstoretest.Sigstore
	verifier *kubectl.Verifier

	mu    sync.Mutex
	files map[string][]byte
	asked []string
}

// NewKubectlReleases starts the release host; it stops when the test ends.
func NewKubectlReleases(t testing.TB) *KubectlReleases {
	t.Helper()
	r := &KubectlReleases{files: map[string][]byte{}, sigstore: sigstoretest.New(t)}
	v, err := kubectl.NewVerifier(r.sigstore.TrustedMaterial(), kubectl.KubernetesRelease, 0)
	if err != nil {
		t.Fatal(err)
	}
	r.verifier = v
	r.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		r.asked = append(r.asked, req.URL.Path)
		body, ok := r.files[req.URL.Path]
		r.mu.Unlock()
		if strings.HasPrefix(req.URL.Path, "/api/v1/") {
			r.sigstore.ServeHTTP(w, req)
			return
		}
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

// Downloader downloads from the host, trusting its sigstore instance and asking its log; the
// mirror is ignored.
func (r *KubectlReleases) Downloader(string) *kubectl.Downloader {
	return &kubectl.Downloader{Mirror: r.Mirror(), Client: r.srv.Client(), Verifier: r.verifier, Rekor: r.srv.URL}
}

// Publish puts release version of binary for goos and goarch on the host, with its SHA-256 and a
// signature made and logged for the Kubernetes release's identity.
func (r *KubectlReleases) Publish(t testing.TB, version, goos, goarch string, binary []byte) {
	t.Helper()
	sum := sha256.Sum256(binary)
	sig, cert := r.sigstore.Release(t, sum[:], sigstoretest.Leaf{})
	dir := "/release/v" + version + "/bin/" + goos + "/" + goarch + "/" + kubectl.BinaryName(goos)
	r.Set(dir, binary)
	r.Set(dir+".sha256", []byte(hex.EncodeToString(sum[:])))
	r.Set(dir+".sig", sig)
	r.Set(dir+".cert", cert)
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

// Requests returns the paths asked for, in order: the release files' and the log's.
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
