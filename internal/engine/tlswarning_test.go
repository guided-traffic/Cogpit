package engine

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// insecureKubeconfig has two contexts on the same fake API server: "checked" trusts the server's
// certificate through certificate-authority-data, "unchecked" skips the check
// (insecure-skip-tls-verify).
func insecureKubeconfig(t *testing.T, srv *testkit.FakeAPIServer) string {
	t.Helper()
	doc := `apiVersion: v1
kind: Config
current-context: checked
clusters:
- name: checked
  cluster:
    server: ` + srv.URL + `
    certificate-authority-data: ` + base64.StdEncoding.EncodeToString(srv.CAData()) + `
- name: unchecked
  cluster:
    server: ` + srv.URL + `
    insecure-skip-tls-verify: true
users:
- name: u
  user: {}
contexts:
- name: checked
  context: {cluster: checked, user: u}
- name: unchecked
  context: {cluster: unchecked, user: u}
`
	p := filepath.Join(t.TempDir(), "insecure.yaml")
	if err := os.WriteFile(p, []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// A context whose cluster skips the certificate check is marked in the context list, from the
// merged kubeconfig files and from a kubeconfig source alike, its connect log carries a WARN line
// naming the flag once the connection stands, and its connection's state is TLSUnchecked while it
// stands (A29); the context with a checked certificate has none of it. The warning is no connect
// problem.
func TestInsecureContextIsMarkedAndLogged(t *testing.T) {
	srv := testkit.NewFakeAPIServer(testkit.FakeAPIOptions{})
	defer srv.Close()
	path := insecureKubeconfig(t, srv)

	sources := NewKubeconfigSources(filepath.Join(t.TempDir(), "Cogpit"))
	if _, _, err := sources.AddFile(path); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name               string
		conn               *ClusterConnector
		checked, unchecked string
	}{
		{"merged files", &ClusterConnector{KubeconfigPaths: []string{path}}, "checked", "unchecked"},
		{"a kubeconfig source", &ClusterConnector{Sources: sources}, "checked [insecure.yaml]", "unchecked [insecure.yaml]"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			list, err := tt.conn.Contexts()
			if err != nil {
				t.Fatal(err)
			}
			marked := map[string]bool{}
			for _, c := range list {
				marked[c.Name] = c.Insecure
			}
			if !marked[tt.unchecked] || marked[tt.checked] || len(marked) != 2 {
				t.Errorf("Insecure by context: %v", marked)
			}

			s := New(Options{Connector: tt.conn})
			t.Cleanup(s.Close)
			if tt.conn.Sources != nil {
				// A source context connects once the user allowed it (A28; insecure_approval_test.go).
				approveInsecure(t, s, tt.unchecked)
			}
			if err := s.Connect(context.Background(), tt.unchecked); err != nil {
				t.Fatalf("connect %s: %v", tt.unchecked, err)
			}
			var warns []string
			for _, l := range s.ConnectLog(tt.unchecked) {
				if l.Level == LogWarn {
					warns = append(warns, l.Text)
				}
			}
			if len(warns) != 1 || !strings.Contains(warns[0], "insecure-skip-tls-verify") || !strings.Contains(warns[0], srv.URL) {
				t.Errorf("the connect log's warnings: %q", warns)
			}
			if lines := s.ConnectLog(tt.unchecked); lines[len(lines)-1].Text != "Connected to "+tt.unchecked {
				t.Errorf("the log does not end with the connect: %+v", lines)
			}
			// A29: the connection's state says so while it stands, for the status bar and agents.
			if c := s.UIState().Connection; c.State != Connected || !c.TLSUnchecked {
				t.Errorf("the unchecked connection's state: %+v", c)
			}
			ctxs, err := s.Contexts()
			if err != nil {
				t.Fatal(err)
			}
			for _, c := range ctxs {
				if c.ConnectProblem {
					t.Errorf("%s marked as a connect problem", c.Name)
				}
			}

			if err := s.Connect(context.Background(), tt.checked); err != nil {
				t.Fatalf("connect %s: %v", tt.checked, err)
			}
			if c := s.UIState().Connection; c.State != Connected || c.TLSUnchecked {
				t.Errorf("the checked connection's state: %+v", c)
			}
			for _, l := range s.ConnectLog(tt.checked) {
				if l.Level == LogWarn {
					t.Errorf("a checked connection warns: %q", l.Text)
				}
			}
		})
	}
}
