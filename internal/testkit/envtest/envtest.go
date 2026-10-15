//go:build unix

// Package envtest starts a real Kubernetes control plane for a test: etcd and kube-apiserver from
// envtest's release store, pinned by version and checksum (make envtest-bin keeps them in
// .cache/envtest, D-64), run by controller-runtime's envtest, with a cluster-admin's kubeconfig the
// engine connects with. No kubelet, scheduler or controller manager runs: pods stay Pending, logs,
// exec and port forwards have nothing to reach, and nothing is garbage-collected. What it gives
// that testkit's fake API server does not: the real API server's code path (discovery, RBAC,
// streaming lists, watches and their bookmarks, admission) behind the real client-go stack, and a
// process the test can stop (FreezeAPIServer) without Docker.
//
// It is for tests only (depguard's no-testkit-in-product keeps it out of both binaries), and only
// tests tagged envtest start it: make test-envtest, which downloads the binaries first and names
// their folder in COGPIT_ENVTEST_ASSETS. Without that variable Start skips the test. Unix only:
// controller-runtime v0.25.2's envtest does not compile for Windows (its process package declares
// signalProcess twice there), and FreezeAPIServer needs SIGSTOP; on Windows the package has no
// files, which go build ./... and go vet ./... pass over.
package envtest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crenvtest "sigs.k8s.io/controller-runtime/pkg/envtest"
	crlog "sigs.k8s.io/controller-runtime/pkg/log"
)

// AssetsEnv names the folder that holds the etcd and kube-apiserver binaries (make test-envtest
// sets it to .cache/envtest/<version>).
const AssetsEnv = "COGPIT_ENVTEST_ASSETS"

// Context is the name of the one context, cluster and user of the control plane's kubeconfig.
const Context = "envtest"

// ControlPlane is a running etcd and kube-apiserver. It stops with the test.
type ControlPlane struct {
	// Config is a cluster-admin's REST config (a client certificate in system:masters).
	Config *rest.Config
	// Clientset is a client with Config, for the test's own reads and writes.
	Clientset kubernetes.Interface
	// Kubeconfig is that user's kubeconfig, its one context named Context.
	Kubeconfig []byte

	env     *crenvtest.Environment
	pidFile string
	logs    *tail

	mu     sync.Mutex
	pid    int
	frozen bool
}

// Start starts etcd and kube-apiserver from the binaries in COGPIT_ENVTEST_ASSETS and waits until
// the API server serves. It skips the test when the variable is not set and fails it when the
// binaries are missing.
func Start(t testing.TB) *ControlPlane {
	t.Helper()
	dir := os.Getenv(AssetsEnv)
	if dir == "" {
		t.Skipf("%s is not set: run make test-envtest", AssetsEnv)
	}
	for _, bin := range []string{"etcd", "kube-apiserver"} {
		if _, err := os.Stat(filepath.Join(dir, bin)); err != nil {
			t.Fatalf("%s names %s, which holds no %s: run make envtest-bin (%v)", AssetsEnv, dir, bin, err)
		}
	}
	// envtest logs through controller-runtime's global logger, which warns once when nothing set it.
	crlog.SetLogger(logr.Discard())

	tmp := t.TempDir()
	cp := &ControlPlane{pidFile: filepath.Join(tmp, "kube-apiserver.pid"), logs: &tail{max: 64 << 10}}
	apiServer := &crenvtest.APIServer{
		// A wrapper that records the server's process id and execs it (process_unix.go): envtest
		// keeps the process to itself, and FreezeAPIServer signals it.
		Path: apiServerCommand(t, filepath.Join(dir, "kube-apiserver"), cp.pidFile, tmp),
		Out:  cp.logs,
		Err:  cp.logs,
	}
	cp.env = &crenvtest.Environment{
		BinaryAssetsDirectory:    dir,
		ControlPlane:             crenvtest.ControlPlane{APIServer: apiServer, Etcd: &crenvtest.Etcd{Out: cp.logs, Err: cp.logs}},
		ControlPlaneStartTimeout: time.Minute,
		ControlPlaneStopTimeout:  30 * time.Second,
	}
	cfg, err := cp.env.Start()
	if err != nil {
		t.Fatalf("envtest: %v\n%s", err, cp.logs.String())
	}
	t.Cleanup(func() {
		if err := cp.thaw(); err != nil { // a stopped process would not see the SIGTERM that ends it
			t.Errorf("SIGCONT kube-apiserver (pid %d): %v", cp.pid, err)
		}
		if t.Failed() {
			t.Logf("etcd and kube-apiserver, their last output:\n%s", cp.logs.String())
		}
		if err := cp.env.Stop(); err != nil {
			t.Errorf("envtest stop: %v", err)
		}
	})
	cp.Config = cfg
	cp.Kubeconfig = cp.env.KubeConfig
	if cp.Clientset, err = kubernetes.NewForConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cp.pid = readPID(t, cp.pidFile)
	return cp
}

// WriteKubeconfig writes the control plane's kubeconfig to a file of the test's own (0600) and
// returns its path.
func (cp *ControlPlane) WriteKubeconfig(t testing.TB) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, cp.Kubeconfig, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// FreezeAPIServer stops the kube-apiserver process (SIGSTOP): a peer gone silent, as a cluster is
// for a computer that slept or lost its route (D-58). The kernel still accepts connections on its
// port and takes what clients send, but nothing answers, neither on the open connections nor on
// new ones. etcd runs on. ThawAPIServer resumes it; the test's cleanup does if the test does not.
func (cp *ControlPlane) FreezeAPIServer(t testing.TB) {
	t.Helper()
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.frozen {
		return
	}
	if err := stopProcess(cp.pid); err != nil {
		t.Fatalf("SIGSTOP kube-apiserver (pid %d): %v", cp.pid, err)
	}
	cp.frozen = true
}

// ThawAPIServer resumes the stopped kube-apiserver (SIGCONT).
func (cp *ControlPlane) ThawAPIServer(t testing.TB) {
	t.Helper()
	if err := cp.thaw(); err != nil {
		t.Fatalf("SIGCONT kube-apiserver (pid %d): %v", cp.pid, err)
	}
}

func (cp *ControlPlane) thaw() error {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if !cp.frozen {
		return nil
	}
	if err := continueProcess(cp.pid); err != nil {
		return err
	}
	cp.frozen = false
	return nil
}

// readPID reads the process id the wrapper wrote before it exec'd the server.
func readPID(t testing.TB, path string) int {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("the kube-apiserver wrapper wrote no process id: %v", err)
	}
	pid := 0
	for _, c := range strings.TrimSpace(string(raw)) {
		if c < '0' || c > '9' {
			t.Fatalf("the kube-apiserver wrapper wrote %q, not a process id", raw)
		}
		pid = pid*10 + int(c-'0')
	}
	if pid <= 1 {
		t.Fatalf("the kube-apiserver wrapper wrote %q, not a process id", raw)
	}
	return pid
}

// tail keeps the last max bytes written to it: the control plane's output, shown when a test fails.
type tail struct {
	mu  sync.Mutex
	max int
	buf []byte
}

func (w *tail) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buf = append(w.buf, p...)
	if over := len(w.buf) - w.max; over > 0 {
		w.buf = w.buf[over:]
		if i := bytes.IndexByte(w.buf, '\n'); i >= 0 {
			w.buf = w.buf[i+1:]
		}
	}
	return len(p), nil
}

func (w *tail) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return string(w.buf)
}
