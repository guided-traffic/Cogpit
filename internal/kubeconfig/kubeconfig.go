// Package kubeconfig parses kubeconfig content: the contexts it defines and the per-terminal
// kubeconfig that binds a terminal to one context.
//
// Layer L0: no I/O. Functions take bytes and return values; reading files is package cluster's
// job. Kubeconfig content is untrusted input and must never panic the parser.
package kubeconfig

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/tools/clientcmd/api/latest"
)

// Context is one kubeconfig context as the context picker shows it. It carries names only, never
// a credential.
type Context struct {
	Name      string `json:"name"`
	Cluster   string `json:"cluster"`
	User      string `json:"user"`
	Namespace string `json:"namespace,omitempty"`
	Server    string `json:"server,omitempty"`
	Current   bool   `json:"current,omitempty"`
}

// ErrNotImplemented marks a Phase 0 stub.
var ErrNotImplemented = errors.New("not implemented")

// maxErrorLen caps the length of a decoding error's text: a parser error is for a human, and a
// long one is more likely to carry pieces of the input.
const maxErrorLen = 300

// Contexts returns the contexts defined in one kubeconfig document, sorted by name. Empty input
// is a valid, empty kubeconfig. Errors never echo credential fields.
func Contexts(data []byte) ([]Context, error) {
	cfg, err := Parse(data)
	if err != nil {
		return nil, err
	}
	return FromConfig(cfg), nil
}

// Parse decodes one kubeconfig document the way kubectl does (clientcmd's v1 codec), without
// reading files and without resolving relative paths. Empty input is an empty config. It never
// panics, and its errors never echo credential fields: see SafeError.
func Parse(data []byte) (cfg *clientcmdapi.Config, err error) {
	// The decoder is third-party code fed with untrusted bytes; a panic in it must become an error.
	defer func() {
		if r := recover(); r != nil {
			cfg, err = nil, errors.New("kubeconfig: not a valid kubeconfig document")
		}
	}()
	out := clientcmdapi.NewConfig()
	if len(data) == 0 {
		return out, nil
	}
	decoded, _, err := latest.Codec.Decode(data, &schema.GroupVersionKind{Version: latest.Version, Kind: "Config"}, out)
	if err != nil {
		return nil, fmt.Errorf("kubeconfig: not a valid kubeconfig document: %w", SafeError(err))
	}
	c, ok := decoded.(*clientcmdapi.Config)
	if !ok || c == nil {
		return nil, errors.New("kubeconfig: not a valid kubeconfig document")
	}
	return c, nil
}

// FromConfig returns the contexts of a decoded (possibly merged) kubeconfig, sorted by name. The
// server comes from the context's cluster entry, with any user information in the URL removed.
func FromConfig(cfg *clientcmdapi.Config) []Context {
	if cfg == nil {
		return nil
	}
	out := make([]Context, 0, len(cfg.Contexts))
	for name, kc := range cfg.Contexts {
		c := Context{Name: name, Current: name == cfg.CurrentContext}
		if kc != nil {
			c.Cluster, c.User, c.Namespace = kc.Cluster, kc.AuthInfo, kc.Namespace
			if cl := cfg.Clusters[kc.Cluster]; cl != nil {
				c.Server = StripUserInfo(cl.Server)
			}
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// StripUserInfo removes a "user:password@" part from a server URL: basic-auth credentials in the
// URL are credentials, and the server address is shown in the UI and in errors.
func StripUserInfo(server string) string {
	if !strings.Contains(server, "@") {
		return server
	}
	u, err := url.Parse(server)
	if err != nil || u.User == nil {
		// Unparseable: drop everything up to the last '@' rather than guess.
		if i := strings.LastIndex(server, "@"); i >= 0 {
			scheme := ""
			if j := strings.Index(server, "://"); j >= 0 && j < i {
				scheme = server[:j+3]
			}
			return scheme + server[i+1:]
		}
		return server
	}
	u.User = nil
	return u.String()
}

var backtickSpan = regexp.MustCompile("`[^`]*`")

// SafeError returns an error with the text of err made safe to show: it cannot carry credential
// content from the kubeconfig that caused it. It deliberately does not wrap err, so the original
// text cannot be reached through errors.Unwrap and logged by accident.
//
// Two decoder behaviours make this necessary: client-go's conversion error for a duplicate name
// prints the whole list ("duplicate name "u" in list: [...]"), tokens included, and go-yaml type
// errors quote up to ten characters of the offending value in backticks.
func SafeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for {
		i := strings.Index(msg, " in list: ")
		if i < 0 {
			break
		}
		rest := msg[i+len(" in list: "):]
		// Skip the printed list: up to its matching bracket, or to the end.
		msg = msg[:i] + " in list" + rest[endOfList(rest):]
	}
	msg = backtickSpan.ReplaceAllString(msg, "`…`")
	if len(msg) > maxErrorLen {
		msg = strings.ToValidUTF8(msg[:maxErrorLen], "") + "…"
	}
	return errors.New(msg)
}

// endOfList returns the index just past the bracketed list s starts with, or len(s) when s does
// not start with one or the brackets do not balance (then everything after is dropped).
func endOfList(s string) int {
	if !strings.HasPrefix(s, "[") {
		return len(s)
	}
	depth := 0
	for k, r := range s {
		switch r {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return k + 1
			}
		}
	}
	return len(s)
}
