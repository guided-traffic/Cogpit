//go:build nodemo

// The release build's cogpit-cli (D-67): go test -tags nodemo runs these (make check-nodemo, in
// CI's lint job).

package main

import (
	"runtime"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// testDemo: a release build has no demo cluster to size.
func testDemo() engine.DemoOptions { return engine.DemoOptions{} }

// TestNoDemoInARelease: a release's cogpit-cli neither lists nor reads a demo cluster, and its
// help does not mention one.
func TestNoDemoInARelease(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the test home's layout is a Unix one")
	}
	for _, args := range [][]string{{"help"}, {"help", "contexts"}, {"help", "get"}, {"help", "mcp"}} {
		te := newTestEnv(t)
		te.run(t, args...)
		if out := te.out.String() + te.errOut.String(); strings.Contains(strings.ToLower(out), "demo") {
			t.Errorf("cogpit-cli %s mentions the demo:\n%s", strings.Join(args, " "), out)
		}
	}

	te := newTestEnv(t)
	if code := te.run(t, "contexts"); code != exitOK {
		t.Fatalf("contexts: exit code %d; stderr: %s", code, te.errOut)
	}
	if lines := strings.Split(strings.TrimSpace(te.out.String()), "\n"); len(lines) != 1 || strings.Contains(te.out.String(), "demo") {
		t.Errorf("contexts without kubeconfigs, want the header alone:\n%s", te.out)
	}

	te = newTestEnv(t)
	if code := te.run(t, "get", "pods", "--context", engine.DemoContext, "--timeout", "10s"); code != exitError {
		t.Errorf("get pods --context demo: exit code %d, want %d; stdout: %s; stderr: %s", code, exitError, te.out, te.errOut)
	}
}
