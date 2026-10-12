package kubeconfig

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// envSecret is an environment value planted in exec sections: it binds the fingerprint but must
// never be serialized or shown.
const envSecret = "env-VALUE-9d81-not-for-display"

const execConfig = `apiVersion: v1
kind: Config
clusters:
- name: eks
  cluster:
    server: https://admin:hunter2@eks.example:443
    certificate-authority-data: Zm9v
users:
- name: aws
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args: [eks, get-token, --cluster-name, prod]
      env:
      - name: AWS_PROFILE
        value: ` + envSecret + `
      - name: PATH
        value: /tmp/evil
      provideClusterInfo: true
- name: static
  user:
    token: ` + secret + `
contexts:
- name: prod
  context: {cluster: eks, user: aws}
- name: static
  context: {cluster: eks, user: static}
`

func parseExec(t *testing.T, doc string) *clientcmdapi.Config {
	t.Helper()
	cfg, err := Parse([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	for _, ai := range cfg.AuthInfos {
		ai.LocationOfOrigin = "/home/u/.kube/prod.yaml"
	}
	return cfg
}

func TestExecPluginOf(t *testing.T) {
	cfg := parseExec(t, execConfig)
	if p := ExecPluginOf(cfg, "static"); p != nil {
		t.Errorf("a token user has no plugin: %+v", p)
	}
	if ExecPluginOf(cfg, "missing") != nil || ExecPluginOf(nil, "prod") != nil {
		t.Error("a missing context or config has no plugin")
	}
	p := ExecPluginOf(cfg, "prod")
	if p == nil {
		t.Fatal("no plugin")
	}
	want := ExecPlugin{
		File: "/home/u/.kube/prod.yaml", Context: "prod", User: "aws", Server: "https://eks.example:443",
		APIVersion: "client.authentication.k8s.io/v1beta1", Command: "aws",
		Args:     []string{"eks", "get-token", "--cluster-name", "prod"},
		EnvNames: []string{"AWS_PROFILE", "PATH"}, InteractiveMode: "IfAvailable", ProvideClusterInfo: true,
		TLS: TLSView{CAEmbedded: true},
	}
	got := *p
	got.env, got.serverRaw, got.trust = nil, "", serverTrust{}
	if b1, b2 := mustJSON(t, got), mustJSON(t, want); b1 != b2 {
		t.Errorf("got  %s\nwant %s", b1, b2)
	}
	out := mustJSON(t, p)
	for _, leak := range []string{envSecret, "/tmp/evil", "hunter2"} {
		if strings.Contains(out, leak) || strings.Contains(p.String(), leak) {
			t.Errorf("the plugin's JSON or String shows %q: %s", leak, out)
		}
	}
	if got := p.RiskyEnv(); len(got) != 1 || got[0] != "PATH" {
		t.Errorf("RiskyEnv = %v, want [PATH]", got)
	}
	if !strings.HasPrefix(p.Fingerprint(), "sha256:") || len(p.Fingerprint()) != len("sha256:")+64 {
		t.Errorf("fingerprint %q", p.Fingerprint())
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// TestFingerprintBindsWhatRunsAndWhereItsCredentialGoes changes one thing at a time: everything
// that decides what runs, with which environment, and where its credential is sent changes the
// fingerprint; the install hint, which only a failing plugin shows, does not.
func TestFingerprintBindsWhatRunsAndWhereItsCredentialGoes(t *testing.T) {
	base := func() (*clientcmdapi.Cluster, *clientcmdapi.ExecConfig) {
		return &clientcmdapi.Cluster{Server: "https://a.example", CertificateAuthorityData: []byte("ca")},
			&clientcmdapi.ExecConfig{
				APIVersion: "client.authentication.k8s.io/v1", Command: "aws", Args: []string{"a", "b"},
				Env: []clientcmdapi.ExecEnvVar{{Name: "K", Value: "v"}}, InteractiveMode: clientcmdapi.NeverExecInteractiveMode,
			}
	}
	cl, ec := base()
	ref := NewExecPlugin("/f", "ctx", "u", cl, ec).Fingerprint()
	if again := NewExecPlugin("/f", "ctx", "u", cl, ec).Fingerprint(); again != ref {
		t.Fatal("the fingerprint is not deterministic")
	}
	tests := []struct {
		name   string
		change func(file, ctx, user *string, cl *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig)
		same   bool
	}{
		{"command", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) { ec.Command = "./aws" }, false},
		{"an argument", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) { ec.Args[1] = "c" }, false},
		{"argument order", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) {
			ec.Args = []string{"b", "a"}
		}, false},
		{"one argument more", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) {
			ec.Args = append(ec.Args, "")
		}, false},
		{"an env value", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) { ec.Env[0].Value = "w" }, false},
		{"an env name", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) { ec.Env[0].Name = "PATH" }, false},
		{"api version", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) {
			ec.APIVersion = "client.authentication.k8s.io/v1beta1"
		}, false},
		{"interactive mode", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) {
			ec.InteractiveMode = clientcmdapi.AlwaysExecInteractiveMode
		}, false},
		{"cluster info", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) {
			ec.ProvideClusterInfo = true
		}, false},
		{"server", func(_, _, _ *string, cl *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) {
			cl.Server = "https://b.example"
		}, false},
		{"password in the server URL", func(_, _, _ *string, cl *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) {
			cl.Server = "https://x:y@a.example"
		}, false},
		{"CA", func(_, _, _ *string, cl *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) {
			cl.CertificateAuthorityData = []byte("other")
		}, false},
		{"CA file", func(_, _, _ *string, cl *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) {
			cl.CertificateAuthority = "/ca.pem"
		}, false},
		{"insecure", func(_, _, _ *string, cl *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) {
			cl.InsecureSkipTLSVerify = true
		}, false},
		{"TLS server name", func(_, _, _ *string, cl *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) { cl.TLSServerName = "x" }, false},
		{"proxy", func(_, _, _ *string, cl *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) {
			cl.ProxyURL = "socks5://x"
		}, false},
		{"file", func(f, _, _ *string, _ *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) { *f = "/g" }, false},
		{"context", func(_, c, _ *string, _ *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) { *c = "other" }, false},
		{"user", func(_, _, u *string, _ *clientcmdapi.Cluster, _ *clientcmdapi.ExecConfig) { *u = "other" }, false},
		{"install hint", func(_, _, _ *string, _ *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) {
			ec.InstallHint = "brew install aws"
		}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, ctx, user := "/f", "ctx", "u"
			cl, ec := base()
			tt.change(&file, &ctx, &user, cl, ec)
			got := NewExecPlugin(file, ctx, user, cl, ec).Fingerprint()
			if (got == ref) != tt.same {
				t.Errorf("fingerprint same = %v, want %v", got == ref, tt.same)
			}
		})
	}
	if NewExecPlugin("/f", "c", "u", nil, nil) != nil {
		t.Error("no exec section must give no plugin")
	}
	var none *ExecPlugin
	if none.Fingerprint() != "" || none.RiskyEnv() != nil || none.String() == "" {
		t.Error("a nil plugin")
	}
}

// TestFingerprintBindsFileFactsAndTheTokenFile: the digests of files inside the folder, the
// resolved link paths and the token file (when it needs approval) are part of the fingerprint
// (version 2: R-5, D-17); a token file inside the folder is not.
func TestFingerprintBindsFileFactsAndTheTokenFile(t *testing.T) {
	cl := &clientcmdapi.Cluster{Server: "https://a.example"}
	ec := &clientcmdapi.ExecConfig{APIVersion: "client.authentication.k8s.io/v1", Command: "/f/plug"}
	mk := func(tokenFile string, facts FileFacts) string {
		p := NewLogin("/f/kc.yaml", "ctx", "u", cl, ec, tokenFile)
		p.SetFiles(facts)
		return p.Fingerprint()
	}
	ref := mk("", FileFacts{ProgramSHA256: "aa"})
	for name, other := range map[string]string{
		"program digest":   mk("", FileFacts{ProgramSHA256: "bb"}),
		"program note":     mk("", FileFacts{ProgramNote: "not a regular file"}),
		"program link":     mk("", FileFacts{ProgramSHA256: "aa", ProgramResolved: "/x"}),
		"CA digest":        mk("", FileFacts{ProgramSHA256: "aa", CAFileSHA256: "cc"}),
		"CA link":          mk("", FileFacts{ProgramSHA256: "aa", CAFileResolved: "/x"}),
		"token file":       mk("/t/token", FileFacts{ProgramSHA256: "aa"}),
		"token file other": mk("/t/other", FileFacts{ProgramSHA256: "aa"}),
	} {
		if other == ref {
			t.Errorf("%s must change the fingerprint", name)
		}
	}
	withToken := mk("/t/token", FileFacts{ProgramSHA256: "aa"})
	if again := mk("/t/token", FileFacts{ProgramSHA256: "aa"}); again != withToken {
		t.Error("not deterministic")
	}
	if mk("/f/token", FileFacts{ProgramSHA256: "aa", TokenInFolder: true}) != ref {
		t.Error("a token file inside the folder needs no approval and must not be bound")
	}

	tok := NewLogin("/f/kc.yaml", "ctx", "u", cl, nil, "/t/token")
	if tok == nil || !tok.NeedsApproval() || tok.Kind() != LoginTokenFile || tok.Command != "" {
		t.Fatalf("a token file outside the folder: %+v", tok)
	}
	tok.SetFiles(FileFacts{TokenInFolder: true})
	if tok.NeedsApproval() || tok.Kind() != "" {
		t.Error("a token file inside the folder needs no approval")
	}
	if NewLogin("/f", "c", "u", nil, nil, "") != nil {
		t.Error("neither exec nor token file gives no login")
	}
	if p := NewLogin("/f/kc.yaml", "ctx", "u", cl, ec, "/t/token"); p.Kind() != LoginBoth {
		t.Errorf("kind %q", p.Kind())
	}
}

// TestTLSViewHidesProxyCredentials: what the dialog shows of the proxy has no user information.
func TestTLSViewHidesProxyCredentials(t *testing.T) {
	cl := &clientcmdapi.Cluster{
		Server: "https://a.example", ProxyURL: "http://user:hunter2@proxy.example:3128", InsecureSkipTLSVerify: true,
		TLSServerName: "sni.example", CertificateAuthority: "/ca.pem",
	}
	p := NewExecPlugin("/f", "c", "u", cl, &clientcmdapi.ExecConfig{Command: "aws"})
	if p.TLS.ProxyURL != "http://proxy.example:3128" || !p.TLS.Insecure || p.TLS.ServerName != "sni.example" || p.TLS.CAFile != "/ca.pem" || p.TLS.CAEmbedded {
		t.Errorf("TLS view %+v", p.TLS)
	}
	if out := mustJSON(t, p); strings.Contains(out, "hunter2") {
		t.Errorf("the JSON shows the proxy password: %s", out)
	}
}

func TestDisplayText(t *testing.T) {
	tests := []struct {
		in   string
		max  int
		want string
	}{
		{"aws", 0, "aws"},
		{"/usr/bin/kube login", 0, "/usr/bin/kube login"},
		{"a\nb", 0, `a\u000ab`},
		{"evil\u202egnp.exe", 0, `evil\u202egnp.exe`}, // right-to-left override
		{"zero\u200bwidth", 0, `zero\u200bwidth`},
		{"bad\xffutf8", 0, `bad\xffutf8`},
		{"\x1b[31mred", 0, `\u001b[31mred`},
		{"tab\there", 0, `tab\u0009here`},
		{"Grüße/kubectl-ök \U0001F680", 0, "Grüße/kubectl-ök \U0001F680"},
		{strings.Repeat("x", 20), 10, strings.Repeat("x", 10) + "…"},
		{strings.Repeat("ü", 10), 9, strings.Repeat("ü", 4) + "…"}, // never cuts inside a character
		{strings.Repeat("\x00", 10), 13, `\u0000\u0000` + "…"},     // never cuts inside an escape
		{"short", 10, "short"},
		{"", 10, ""},
	}
	for _, tt := range tests {
		if got := DisplayText(tt.in, tt.max); got != tt.want {
			t.Errorf("DisplayText(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
		}
	}
}

func TestSyncCandidate(t *testing.T) {
	for name, want := range map[string]bool{
		"config": true, "team.yaml": true, "kind.kubeconfig": true,
		"": false, ".DS_Store": false, ".team.yaml.swp": false, "team.yaml~": false, "x.swp": false, "download.tmp": false,
	} {
		if SyncCandidate(name) != want {
			t.Errorf("SyncCandidate(%q) = %v, want %v", name, !want, want)
		}
	}
}

// FuzzExecPlugins feeds kubeconfig bytes through the parser and the plugin description of
// every context: nothing panics, the fingerprint is deterministic, the JSON never carries an
// environment value, and DisplayText never lets a control or format character through.
func FuzzExecPlugins(f *testing.F) {
	for _, seed := range []string{
		"", execConfig, twoContexts,
		"users:\n- name: u\n  user:\n    exec: {command: x, env: [{name: A, value: B}], args: null}\ncontexts:\n- name: c\n  context: {user: u}\n",
		"users:\n- name: u\n  user:\n    exec: {command: \"\\u202e/tmp/x\"}\ncontexts:\n- name: c\n  context: {user: u, cluster: k}\nclusters:\n- name: k\n  cluster: null\n",
		"contexts:\n- name: c\n  context: null\n",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		cfg, err := Parse(data)
		if err != nil {
			return
		}
		for name := range cfg.Contexts {
			p := ExecPluginOf(cfg, name)
			if p == nil {
				continue
			}
			if p.Fingerprint() != ExecPluginOf(cfg, name).Fingerprint() {
				t.Fatal("fingerprint not deterministic")
			}
			out, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range p.env {
				if len(e.Value) >= 8 && !strings.Contains(e.Name+p.Command+strings.Join(p.Args, " ")+p.File+p.Context+p.User+p.Server, e.Value) &&
					strings.Contains(string(out), e.Value) {
					t.Fatalf("the JSON carries an environment value: %s", out)
				}
			}
			for _, s := range append([]string{p.Command, p.File, p.Context}, p.Args...) {
				for _, r := range DisplayText(s, 0) {
					if !unicode.IsGraphic(r) || unicode.Is(unicode.Cf, r) {
						t.Fatalf("DisplayText(%q) lets %U through", s, r)
					}
				}
			}
		}
	})
}
