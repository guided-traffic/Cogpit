package kubeconfig

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clientcmdlatest "k8s.io/client-go/tools/clientcmd/api/latest"
)

// decode parses data the way client-go's kubeconfig loader does.
func decode(t *testing.T, data []byte) *clientcmdapi.Config {
	t.Helper()
	obj, err := runtime.Decode(clientcmdlatest.Codec, data)
	if err != nil {
		t.Fatalf("client-go cannot decode the terminal kubeconfig: %v\n%s", err, data)
	}
	cfg, ok := obj.(*clientcmdapi.Config)
	if !ok {
		t.Fatalf("decoded %T, want *api.Config", obj)
	}
	return cfg
}

func TestTerminalConfigRoundTrips(t *testing.T) {
	cases := []Context{
		{Name: "ctx-a", Cluster: "cluster-a", User: "user-a", Namespace: "team-x"},
		{Name: "kind-cogpit", Cluster: "kind-cogpit", User: "kind-cogpit"}, // no namespace
		{Name: "arn:aws:eks:eu-central-1:123456789012:cluster/prod", Cluster: "arn:aws:eks:eu-central-1:123456789012:cluster/prod", User: "arn:aws:eks:eu-central-1:123456789012:cluster/prod", Namespace: "default"},
		// YAML 1.1 would read these unquoted as booleans, null, numbers or mappings.
		{Name: "yes", Cluster: "null", User: "123", Namespace: "on"},
		{Name: "a: b", Cluster: "#not-a-comment", User: "- dash", Namespace: "x"},
		{Name: `quote"and\backslash`, Cluster: "spaces in name", User: "ünïcödé 集群", Namespace: "y"},
		{Name: "{json}", Cluster: "[list]", User: "&anchor *alias !tag %dir @at `tick`", Namespace: "z"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			data, err := TerminalConfig(c)
			if err != nil {
				t.Fatalf("TerminalConfig: %v", err)
			}
			cfg := decode(t, data)
			if cfg.CurrentContext != c.Name {
				t.Errorf("current-context = %q, want %q", cfg.CurrentContext, c.Name)
			}
			if len(cfg.Contexts) != 1 {
				t.Fatalf("contexts = %d, want 1", len(cfg.Contexts))
			}
			got := cfg.Contexts[c.Name]
			if got == nil {
				t.Fatalf("context %q missing: %v", c.Name, cfg.Contexts)
			}
			if got.Cluster != c.Cluster || got.AuthInfo != c.User || got.Namespace != c.Namespace {
				t.Errorf("context = {cluster %q, user %q, namespace %q}, want {%q, %q, %q}",
					got.Cluster, got.AuthInfo, got.Namespace, c.Cluster, c.User, c.Namespace)
			}
			if len(cfg.Clusters) != 0 || len(cfg.AuthInfos) != 0 || len(cfg.Extensions) != 0 {
				t.Errorf("terminal kubeconfig defines clusters %v, users %v or extensions %v; it must name them only",
					cfg.Clusters, cfg.AuthInfos, cfg.Extensions)
			}
		})
	}
}

func TestTerminalConfigHoldsOnlyNames(t *testing.T) {
	data, err := TerminalConfig(Context{Name: "ctx-a", Cluster: "c", User: "u", Namespace: "team-x",
		Server: "https://example.invalid:6443", Current: true})
	if err != nil {
		t.Fatal(err)
	}
	want := `apiVersion: v1
kind: Config
current-context: "ctx-a"
contexts:
- name: "ctx-a"
  context:
    cluster: "c"
    user: "u"
    namespace: "team-x"
`
	if string(data) != want {
		t.Errorf("TerminalConfig =\n%s\nwant\n%s", data, want)
	}
	for _, key := range []string{"clusters:", "users:", "server", "token", "client-key", "client-certificate", "exec", "auth-provider", "password"} {
		if strings.Contains(string(data), key) {
			t.Errorf("terminal kubeconfig contains %q", key)
		}
	}
}

func TestTerminalConfigRejectsBadInput(t *testing.T) {
	cases := map[string]Context{
		"empty context":        {Cluster: "c", User: "u"},
		"empty cluster":        {Name: "n", User: "u"},
		"empty user":           {Name: "n", Cluster: "c"},
		"newline in name":      {Name: "n\ncurrent-context: other", Cluster: "c", User: "u"},
		"escape in cluster":    {Name: "n", Cluster: "c\x1b]52;c;AAAA\a", User: "u"},
		"invalid UTF-8 user":   {Name: "n", Cluster: "c", User: "u\xff"},
		"overlong name":        {Name: strings.Repeat("n", maxNameLen+1), Cluster: "c", User: "u"},
		"namespace upper case": {Name: "n", Cluster: "c", User: "u", Namespace: "Team"},
		"namespace with slash": {Name: "n", Cluster: "c", User: "u", Namespace: "a/b"},
		"namespace injection":  {Name: "n", Cluster: "c", User: "u", Namespace: "x\"\n    user: other"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if data, err := TerminalConfig(c); err == nil {
				t.Errorf("TerminalConfig accepted %+v:\n%s", c, data)
			}
		})
	}
}
