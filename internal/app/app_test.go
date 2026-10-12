package app_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/guided-traffic/cogpit/internal/app"
)

const kubeconfigOne = `apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: https://127.0.0.1:6443
contexts:
- name: one
  context:
    cluster: c
    user: u
users:
- name: u
  user: {}
`

func TestKubeconfigSourcesAreTheAppsSources(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".kube"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".kube", "config"), []byte(kubeconfigOne), 0o600); err != nil {
		t.Fatal(err)
	}
	named := filepath.Join(t.TempDir(), "run.yaml")
	if err := os.WriteFile(named, []byte(kubeconfigOne), 0o600); err != nil {
		t.Fatal(err)
	}
	d := app.Dirs{Config: filepath.Join(t.TempDir(), app.DirName), Home: home}

	ctxs, err := d.KubeconfigSources([]string{named}).Contexts()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, c := range ctxs {
		got[c.Name] = true
	}
	if !got["one [.kube/config]"] || !got["one [run.yaml]"] || len(got) != 2 {
		t.Errorf("contexts %v: want ~/.kube/config as the default folder's file and the named file as a source of its own", got)
	}
	if _, err := os.Stat(d.Config); !os.IsNotExist(err) {
		t.Errorf("listing wrote Cogpit's directory: %v", err)
	}

	// Without a home directory there is no default folder.
	ctxs, err = app.Dirs{Config: d.Config}.KubeconfigSources(nil).Contexts()
	if err != nil || len(ctxs) != 0 {
		t.Errorf("without a home: %v, %v", ctxs, err)
	}
}

func TestAuditLogIsOwnerOnlyAndAppends(t *testing.T) {
	d := app.Dirs{Config: filepath.Join(t.TempDir(), app.DirName)}
	for _, line := range []string{"one\n", "two\n"} {
		f, err := d.OpenAuditLog()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(line)); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(d.Config, app.AuditLogFile)
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "one\ntwo\n" {
		t.Errorf("audit log %q, %v", data, err)
	}
	if runtime.GOOS != "windows" {
		for p, want := range map[string]os.FileMode{path: 0o600, d.Config: 0o700} {
			fi, err := os.Stat(p)
			if err != nil {
				t.Fatal(err)
			}
			if fi.Mode().Perm() != want {
				t.Errorf("%s: mode %#o, want %#o", p, fi.Mode().Perm(), want)
			}
		}
	}
	if _, err := (app.Dirs{}).OpenAuditLog(); err == nil {
		t.Error("an audit log without a configuration directory")
	}
}
