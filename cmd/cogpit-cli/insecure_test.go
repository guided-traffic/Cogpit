package main

import (
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// labKube is a kubeconfig for srv whose cluster skips the certificate check.
func labKube(srv *testkit.FakeAPIServer) string {
	return `apiVersion: v1
kind: Config
clusters:
- name: lab
  cluster:
    server: ` + srv.URL + `
    insecure-skip-tls-verify: true
contexts:
- name: lab
  context: {cluster: lab, user: u, namespace: team}
users:
- name: u
  user: {token: tok-SECRET-lab}
`
}

// allowLab records the user's approval of the lab context's unchecked connection, as the window's
// dialog does, in the configuration directory the CLI reads.
func allowLab(t *testing.T, te *testEnv) {
	t.Helper()
	k := te.dirs.KubeconfigSources(nil)
	f, kctx, ok, err := k.Find("lab [.kube/lab.yaml]")
	if err != nil || !ok || f.Exec[kctx] == nil {
		t.Fatalf("find lab: %v %v %+v", ok, err, f.Exec)
	}
	if err := k.Approve(f.Exec[kctx]); err != nil {
		t.Fatal(err)
	}
}

func requests(srv *testkit.FakeAPIServer) int64 {
	var n int64
	for _, c := range srv.Requests() {
		n += c
	}
	return n
}

// TestGetRefusesAnUncheckedContextUntilAllowed (A28, H-37): a context whose cluster skips the
// certificate check is refused before anything reaches its address, as an unapproved credential
// plugin is, naming the flag and the window; the CLI writes no approval. Once the user allowed it
// in the window, get reads it.
func TestGetRefusesAnUncheckedContextUntilAllowed(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{})
	defer srv.Close()
	srv.Set(testkit.PodsGVR, testkit.Pod("team", "web-0"))
	te := newTestEnv(t)
	te.writeKube(t, "lab.yaml", labKube(srv))

	if code := te.run(t, "get", "pods", "--context", "lab"); code != exitError {
		t.Fatalf("exit code %d, want %d; stdout %s", code, exitError, te.out)
	}
	errOut := te.errOut.String()
	if !strings.Contains(errOut, "insecure-skip-tls-verify") || !strings.Contains(errOut, "navigator's clusters in Cogpit's window (⌘P)") ||
		strings.Contains(errOut, "tok-SECRET") {
		t.Errorf("stderr: %s", errOut)
	}
	if n := requests(srv); n != 0 {
		t.Errorf("refused, yet %d requests reached the address", n)
	}

	allowLab(t, te)
	te.reset()
	if code := te.run(t, "get", "pods", "--context", "lab"); code != exitOK {
		t.Fatalf("allowed: exit code %d; stderr: %s", code, te.errOut)
	}
	if !strings.Contains(te.out.String(), "web-0") {
		t.Errorf("stdout:\n%s", te.out)
	}
}

// TestGetWarnsAboutAnUncheckedCertificate (A29, H-37): connected to a context that skips the
// certificate check, get writes one warning line to standard error, naming the context and the
// server, never the token; standard output stays the result (JSON parses). A checked context gets
// no such line.
func TestGetWarnsAboutAnUncheckedCertificate(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{})
	defer srv.Close()
	srv.Set(testkit.PodsGVR, testkit.Pod("team", "web-0"))
	te := newTestEnv(t)
	te.writeKube(t, "lab.yaml", labKube(srv))
	allowLab(t, te)

	got := te.listJSON(t, "pods", "--context", "lab") // fails the test unless stdout is the JSON result
	if got.Total != 1 {
		t.Errorf("pods %+v", got)
	}
	var warnings []string
	for _, l := range strings.Split(strings.TrimSpace(te.errOut.String()), "\n") {
		if strings.HasPrefix(l, "cogpit-cli: warning: TLS not checked") {
			warnings = append(warnings, l)
		}
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "lab [.kube/lab.yaml]") || !strings.Contains(warnings[0], srv.URL) ||
		!strings.Contains(warnings[0], "insecure-skip-tls-verify") {
		t.Errorf("stderr warnings %q; stderr:\n%s", warnings, te.errOut)
	} else {
		t.Logf("stderr: %s", warnings[0])
	}
	if strings.Contains(te.out.String()+te.errOut.String(), "tok-SECRET") {
		t.Error("the token reached the output")
	}

	checked := srv.WriteKubeconfig(t, "fake", "team")
	te.reset()
	if code := te.run(t, "get", "pods", "--kubeconfig", checked, "--context", "fake"); code != exitOK {
		t.Fatalf("checked: exit code %d; stderr: %s", code, te.errOut)
	}
	if strings.Contains(te.errOut.String(), "TLS not checked") {
		t.Errorf("a checked context warns:\n%s", te.errOut)
	}
}
