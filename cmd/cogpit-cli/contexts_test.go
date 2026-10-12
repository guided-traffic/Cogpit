package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// A kubeconfig with a static token and a server URL with user information: neither may be printed.
const tokenKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: alpha
  cluster:
    server: https://admin:hunter2@127.0.0.1:6443
contexts:
- name: alpha
  context:
    cluster: alpha
    user: alpha
    namespace: team-a
- name: "evil\e[31mred"
  context:
    cluster: alpha
    user: alpha
users:
- name: alpha
  user:
    token: tok-SECRET-alpha
`

// pluginKubeconfig logs in through a program that leaves a marker file when it runs.
func pluginKubeconfig(t *testing.T, marker string) string {
	t.Helper()
	script := filepath.Join(t.TempDir(), "login.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\ntouch '"+marker+"'\necho '{}'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return `apiVersion: v1
kind: Config
clusters:
- name: cloud
  cluster:
    server: https://127.0.0.1:1
contexts:
- name: login
  context:
    cluster: cloud
    user: login
users:
- name: login
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: ` + script + `
      args: ["--client-secret=arg-SECRET"]
      interactiveMode: Never
`
}

func TestContextsListsTheSourcesWithoutSecrets(t *testing.T) {
	te := newTestEnv(t)
	te.writeKube(t, "config", tokenKubeconfig)
	marker := filepath.Join(t.TempDir(), "plugin-ran")
	te.writeKube(t, "plugins.yaml", pluginKubeconfig(t, marker))

	if code := te.run(t, "contexts"); code != exitOK {
		t.Fatalf("exit code %d; stderr: %s", code, te.errOut)
	}
	out := te.out.String()
	for _, want := range []string{
		"CONTEXT", "LOGIN",
		"alpha [.kube/config]", "team-a", "https://127.0.0.1:6443",
		"login [.kube/plugins.yaml]", "plugin required",
		"demo", "(synthetic demo cluster)",
		`evil\u001b[31mred [.kube/config]`, // an escape sequence in a context name is shown, not obeyed
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output lacks %q:\n%s", want, out)
		}
	}
	for _, leak := range []string{"tok-SECRET", "hunter2", "arg-SECRET", "\x1b"} {
		if strings.Contains(out+te.errOut.String(), leak) {
			t.Errorf("output carries %q:\n%s\n%s", leak, out, te.errOut)
		}
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("listing ran the credential plugin: %v", err)
	}
	// Listing writes nothing of Cogpit's: no sources file, no approvals, no directory.
	if _, err := os.Stat(te.dirs.Config); !os.IsNotExist(err) {
		t.Errorf("contexts created %s: %v", te.dirs.Config, err)
	}
}

func TestContextsJSONIsTheToolsResult(t *testing.T) {
	te := newTestEnv(t)
	te.writeKube(t, "config", tokenKubeconfig)
	// A file named for this run, as --kubeconfig names it: a source of its own, read alone.
	extra := filepath.Join(t.TempDir(), "beta.yaml")
	if err := os.WriteFile(extra, []byte(strings.ReplaceAll(tokenKubeconfig, "name: alpha\n  context", "name: beta\n  context")), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := te.run(t, "contexts", "-o", "json", "--kubeconfig", extra); code != exitOK {
		t.Fatalf("exit code %d; stderr: %s", code, te.errOut)
	}
	var list []engine.ContextInfo
	if err := json.Unmarshal([]byte(te.out.String()), &list); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, te.out)
	}
	byName := map[string]engine.ContextInfo{}
	for _, c := range list {
		byName[c.Name] = c
	}
	alpha, ok := byName["alpha [.kube/config]"]
	if !ok || alpha.KubeContext != "alpha" || alpha.Namespace != "team-a" || alpha.File != filepath.Join(te.home, ".kube", "config") {
		t.Errorf("alpha: %+v (all: %+v)", alpha, list)
	}
	if beta, ok := byName["beta [beta.yaml]"]; !ok || beta.File != extra {
		t.Errorf("the --kubeconfig file is not a source of its own: %+v", list)
	}
	if demo, ok := byName["demo"]; !ok || !demo.Demo {
		t.Errorf("no demo context: %+v", list)
	}
	if strings.Contains(te.out.String(), "tok-SECRET") || strings.Contains(te.out.String(), "hunter2") {
		t.Errorf("credential in the output:\n%s", te.out)
	}

	te.out.mu.Lock()
	te.out.buf.Reset()
	te.out.mu.Unlock()
	if code := te.run(t, "contexts", "--output=yaml"); code != exitOK {
		t.Fatalf("yaml: exit code %d; stderr: %s", code, te.errOut)
	}
	if !strings.Contains(te.out.String(), "- kubeContext: alpha") && !strings.Contains(te.out.String(), "kubeContext: alpha") {
		t.Errorf("yaml output:\n%s", te.out)
	}
}

func TestContextsWithoutKubeconfigsListsTheDemo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the test home's layout is a Unix one")
	}
	te := newTestEnv(t)
	if code := te.run(t, "contexts"); code != exitOK {
		t.Fatalf("exit code %d; stderr: %s", code, te.errOut)
	}
	lines := strings.Split(strings.TrimSpace(te.out.String()), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[1], "demo") {
		t.Errorf("want the header and the demo cluster:\n%s", te.out)
	}
}
