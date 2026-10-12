package kubeconfig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ExecPlugin is the exec credential plugin a kubeconfig context's user runs to get a credential
// (users[].user.exec): the program Cogpit shows the user before its first run, and what the user's
// approval of it binds (docs/security/trust-boundaries.md, H-1).
//
// It holds the names of the variables the plugin's environment adds, never their values: those
// enter the fingerprint only. The other fields are as the kubeconfig wrote them, untrusted text;
// show them through DisplayText.
type ExecPlugin struct {
	// File is the kubeconfig file the user entry comes from.
	File string `json:"file"`
	// Context and User are the context's name and its user entry's name in File.
	Context string `json:"context"`
	User    string `json:"user"`
	// Server is the API server the credential is sent to, without user information.
	Server string `json:"server,omitempty"`
	// APIVersion is the ExecCredential version the plugin speaks.
	APIVersion string `json:"apiVersion,omitempty"`
	// Command is the program as client-go runs it: a path with a separator, made absolute
	// against File's directory when relative, or a name looked up on PATH when it runs.
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	// EnvNames are the names of the variables the kubeconfig adds to the plugin's environment.
	EnvNames []string `json:"envNames,omitempty"`
	// InteractiveMode and ProvideClusterInfo are the exec section's own fields.
	InteractiveMode    string `json:"interactiveMode,omitempty"`
	ProvideClusterInfo bool   `json:"provideClusterInfo,omitempty"`

	env       []clientcmdapi.ExecEnvVar // values are bound by the fingerprint, never serialized
	serverRaw string                    // the server URL as written, bound by the fingerprint
	trust     serverTrust               // where the credential goes, bound by the fingerprint
}

// serverTrust is what decides which server receives the plugin's credential, beyond its URL.
type serverTrust struct {
	CAData     []byte `json:"caData,omitempty"`
	CAFile     string `json:"caFile,omitempty"`
	Insecure   bool   `json:"insecure,omitempty"`
	ServerName string `json:"serverName,omitempty"`
	ProxyURL   string `json:"proxyURL,omitempty"`
}

// NewExecPlugin describes the exec section ec of user entry user, used by context in file to reach
// cluster (which may be nil). It returns nil when ec is nil.
func NewExecPlugin(file, context, user string, cluster *clientcmdapi.Cluster, ec *clientcmdapi.ExecConfig) *ExecPlugin {
	if ec == nil {
		return nil
	}
	p := &ExecPlugin{
		File: file, Context: context, User: user,
		APIVersion: ec.APIVersion, Command: ec.Command, Args: append([]string(nil), ec.Args...),
		InteractiveMode: string(ec.InteractiveMode), ProvideClusterInfo: ec.ProvideClusterInfo,
		env: append([]clientcmdapi.ExecEnvVar(nil), ec.Env...),
	}
	for _, e := range ec.Env {
		p.EnvNames = append(p.EnvNames, e.Name)
	}
	if cluster != nil {
		// The fingerprint binds the URL as written, user information included (a changed password
		// in the URL is a changed destination too); only the stripped form is ever shown.
		p.serverRaw = cluster.Server
		p.Server = StripUserInfo(cluster.Server)
		p.trust = serverTrust{
			CAData: append([]byte(nil), cluster.CertificateAuthorityData...), CAFile: cluster.CertificateAuthority,
			Insecure: cluster.InsecureSkipTLSVerify, ServerName: cluster.TLSServerName, ProxyURL: cluster.ProxyURL,
		}
	}
	return p
}

// ExecPluginOf returns the exec plugin of the context named context in cfg, or nil when the
// context, its user entry or the user's exec section is missing. cfg should come from client-go's
// loader, which records each entry's file (LocationOfOrigin) and resolves relative paths, so that
// the plugin is described as it will run.
func ExecPluginOf(cfg *clientcmdapi.Config, context string) *ExecPlugin {
	if cfg == nil {
		return nil
	}
	kc := cfg.Contexts[context]
	if kc == nil {
		return nil
	}
	ai := cfg.AuthInfos[kc.AuthInfo]
	if ai == nil || ai.Exec == nil {
		return nil
	}
	return NewExecPlugin(ai.LocationOfOrigin, context, kc.AuthInfo, cfg.Clusters[kc.Cluster], ai.Exec)
}

// fingerprintDoc is the canonical form the fingerprint hashes; its field order is fixed.
type fingerprintDoc struct {
	Version            int         `json:"v"`
	File               string      `json:"file"`
	Context            string      `json:"context"`
	User               string      `json:"user"`
	Server             string      `json:"server"`
	Trust              serverTrust `json:"trust"`
	APIVersion         string      `json:"apiVersion"`
	Command            string      `json:"command"`
	Args               []string    `json:"args"`
	Env                [][2]string `json:"env"`
	InteractiveMode    string      `json:"interactiveMode"`
	ProvideClusterInfo bool        `json:"provideClusterInfo"`
}

// Fingerprint returns "sha256:<hex>" over everything that decides what runs and where its
// credential goes: the file, context and user, the server (with any user information in its URL)
// and how it is trusted (CA, insecure, TLS server name, proxy), the API version, the command, the
// arguments in order, the environment's names and values in order, the interactive mode and
// whether cluster information is passed. An approval is bound to it: any change asks the user
// again. The hash keeps the environment's values out of the approval store.
func (p *ExecPlugin) Fingerprint() string {
	if p == nil {
		return ""
	}
	doc := fingerprintDoc{
		Version: 1, File: p.File, Context: p.Context, User: p.User, Server: p.serverRaw, Trust: p.trust,
		APIVersion: p.APIVersion, Command: p.Command, Args: p.Args,
		InteractiveMode: p.InteractiveMode, ProvideClusterInfo: p.ProvideClusterInfo,
	}
	if doc.Args == nil {
		doc.Args = []string{}
	}
	doc.Env = make([][2]string, 0, len(p.env))
	for _, e := range p.env {
		doc.Env = append(doc.Env, [2]string{e.Name, e.Value})
	}
	data, err := json.Marshal(doc)
	if err != nil { // cannot happen for these types; an empty fingerprint never matches an approval
		return ""
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// String names the plugin without its arguments or environment: for logs and errors.
func (p *ExecPlugin) String() string {
	if p == nil {
		return "<no exec plugin>"
	}
	return fmt.Sprintf("exec plugin %s of user %s in %s", DisplayText(p.Command, 200), DisplayText(p.User, 200), DisplayText(p.File, 400))
}

// RiskyEnv returns the names among the plugin's environment that change which program runs or
// what it loads (PATH, the dynamic loader's variables), sorted: the approval dialog points them out.
func (p *ExecPlugin) RiskyEnv() []string {
	if p == nil {
		return nil
	}
	var out []string
	for _, n := range p.EnvNames {
		u := strings.ToUpper(n)
		if u == "PATH" || strings.HasPrefix(u, "LD_") || strings.HasPrefix(u, "DYLD_") {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}

// DisplayText returns untrusted text from a kubeconfig made safe to show as one line: control and
// format characters (bidirectional overrides among them, which can make one path look like
// another) and invalid UTF-8 become visible escapes, and text longer than maxBytes is cut with an
// ellipsis. maxBytes ≤ 0 means no limit.
func DisplayText(s string, maxBytes int) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		var unit string // one character or its escape: the text is cut between units only
		switch {
		case r == utf8.RuneError && size <= 1:
			unit = fmt.Sprintf(`\x%02x`, s[i])
		case !unicode.IsGraphic(r) || unicode.Is(unicode.Cf, r):
			unit = fmt.Sprintf(`\u%04x`, r)
		default:
			unit = s[i : i+size]
		}
		if maxBytes > 0 && b.Len()+len(unit) > maxBytes {
			return b.String() + "…"
		}
		b.WriteString(unit)
		i += size
	}
	return b.String()
}
