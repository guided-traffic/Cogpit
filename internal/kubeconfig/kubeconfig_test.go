package kubeconfig

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// secret is a credential value planted in test documents; it must never appear in an error.
const secret = "s3cr3t-T0KEN-value-0123456789"

const twoContexts = `apiVersion: v1
kind: Config
current-context: kind-b
clusters:
- name: cluster-a
  cluster:
    server: https://a.example:6443
- name: cluster-b
  cluster:
    server: https://admin:hunter2@b.example:6443
users:
- name: user-a
  user:
    token: ` + secret + `
- name: user-b
  user:
    client-certificate-data: Zm9v
    client-key-data: YmFy
contexts:
- name: kind-b
  context:
    cluster: cluster-b
    user: user-b
- name: a-context
  context:
    cluster: cluster-a
    user: user-a
    namespace: team-a
- name: dangling
  context:
    cluster: missing
    user: user-a
`

func TestContexts(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []Context
	}{
		{name: "empty input", in: "", want: []Context{}},
		{name: "empty config", in: "apiVersion: v1\nkind: Config\n", want: []Context{}},
		{
			name: "sorted, server, current, namespace, userinfo stripped, dangling cluster",
			in:   twoContexts,
			want: []Context{
				{Name: "a-context", Cluster: "cluster-a", User: "user-a", Namespace: "team-a", Server: "https://a.example:6443"},
				{Name: "dangling", Cluster: "missing", User: "user-a"},
				// "foo" is no certificate: no date, and a fixed text for the cluster layer's log (D-75).
				{Name: "kind-b", Cluster: "cluster-b", User: "user-b", Server: "https://b.example:6443", Current: true,
					ClientCertProblem: "client-certificate-data: no PEM CERTIFICATE block"},
			},
		},
		{
			name: "json document",
			in:   `{"apiVersion":"v1","kind":"Config","contexts":[{"name":"x","context":{"cluster":"c","user":"u"}}],"clusters":[{"name":"c","cluster":{"server":"https://c"}}]}`,
			want: []Context{{Name: "x", Cluster: "c", User: "u", Server: "https://c"}},
		},
		{
			name: "current-context that does not exist",
			in:   "current-context: nope\ncontexts:\n- name: x\n  context: {cluster: c, user: u}\n",
			want: []Context{{Name: "x", Cluster: "c", User: "u"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Contexts([]byte(tt.in))
			if err != nil {
				t.Fatalf("Contexts: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestContextsMalformedNeverEchoesCredentials(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{"not yaml", "\t\t" + secret + ": :\n"},
		{"token wrong type", "users:\n- name: u\n  user:\n    token: [" + secret + "]\n"},
		{"bad base64", "users:\n- name: u\n  user:\n    client-key-data: " + secret + "!!\n"},
		{"user is a string", "users:\n- name: u\n  user: " + secret + "\n"},
		{"exec is a string", "users:\n- name: u\n  user:\n    exec: " + secret + "\n"},
		{"unterminated string", "users:\n- name: u\n  user:\n    token: \"" + secret + "\n"},
		// client-go's conversion error prints the whole list, credentials included.
		{"duplicate user names", "users:\n- name: u\n  user:\n    token: " + secret + "\n- name: u\n  user:\n    password: " + secret + "\n"},
		{"duplicate cluster names", "clusters:\n- name: c\n  cluster:\n    server: https://x\n    certificate-authority-data: Zm9v\n- name: c\n  cluster:\n    server: https://" + secret + "\n"},
		{"wrong kind", "apiVersion: v1\nkind: Secret\ndata:\n  token: " + secret + "\n"},
		{"wrong version", "apiVersion: v2\nkind: Config\nusers:\n- name: u\n  user:\n    token: " + secret + "\n"},
		{"bare scalar", secret},
		{"unknown anchor", "users:\n- name: u\n  user:\n    token: &a " + secret + "\n    password: *b\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Contexts([]byte(tt.in))
			if err == nil {
				t.Fatal("no error for malformed input")
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), secret[:7]) {
				t.Errorf("error echoes the credential: %v", err)
			}
		})
	}
}

func TestSafeError(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"plain", "plain"},
		{`duplicate name "u" in list: [{u {tok [] {x y}}} {u {}}]`, `duplicate name "u" in list`},
		{`a in list: [x] and b in list: [[y] z] end`, `a in list and b in list end`},
		{`unbalanced in list: [secret`, `unbalanced in list`},
		{"line 3: cannot unmarshal !!str `s3cr3t-...` into int", "line 3: cannot unmarshal !!str `…` into int"},
	}
	for _, tt := range tests {
		if got := SafeError(errors.New(tt.in)).Error(); got != tt.want {
			t.Errorf("SafeError(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
	if SafeError(nil) != nil {
		t.Error("SafeError(nil) != nil")
	}
	long := SafeError(errors.New(strings.Repeat("x", 2*maxErrorLen))).Error()
	if len(long) > maxErrorLen+len("…") {
		t.Errorf("not truncated: %d bytes", len(long))
	}
}

func TestStripUserInfo(t *testing.T) {
	tests := map[string]string{
		"https://a.example:6443":            "https://a.example:6443",
		"https://u:p@a.example:6443/prefix": "https://a.example:6443/prefix",
		"https://u@a.example":               "https://a.example",
		"https://u:p w@a.example%zz":        "https://a.example%zz",
		"":                                  "",
	}
	for in, want := range tests {
		if got := StripUserInfo(in); got != want {
			t.Errorf("StripUserInfo(%q) = %q, want %q", in, got, want)
		}
	}
}

func FuzzContexts(f *testing.F) {
	for _, seed := range []string{
		"",
		twoContexts,
		"apiVersion: v1\nkind: Config\n",
		"contexts:\n- name: a\n  context: {cluster: c}\n- name: a\n  context: {cluster: d}\n",
		`{"contexts":[{"name":"x","context":null}],"clusters":[{"name":"c","cluster":null}]}`,
		"users:\n- name: u\n  user:\n    token: &a x\n    password: *a\n",
		"current-context: [a]\n",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		got, err := Contexts(data) // must not panic
		if err != nil {
			if len(err.Error()) > maxErrorLen+200 {
				t.Errorf("error text too long: %d bytes", len(err.Error()))
			}
			return
		}
		for i := 1; i < len(got); i++ {
			if got[i-1].Name > got[i].Name {
				t.Fatalf("not sorted: %q before %q", got[i-1].Name, got[i].Name)
			}
		}
		current := 0
		for _, c := range got {
			if c.Current {
				current++
			}
		}
		if current > 1 {
			t.Fatalf("%d current contexts", current)
		}
	})
}
