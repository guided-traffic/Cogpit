package kubeconfig

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/validation"
)

// maxNameLen bounds context, cluster and user names. Real names (cloud ARNs and the like) stay
// far below it; the bound only keeps a hostile kubeconfig from producing an unbounded file.
const maxNameLen = 1024

// TerminalConfig returns the kubeconfig a terminal puts first on KUBECONFIG: only a
// current-context and one context entry that names the cluster and user entries of the user's own
// files, plus the namespace. It holds no credential (docs/security/credentials.md).
//
// The context entry carries ctx.Name, so it shadows the user's context of that name inside the
// terminal: kubeconfig merging takes a map entry from the first file that defines it, and a
// `kubectl config use-context` or `set-context` in the terminal writes to this file, not to the
// user's. An empty ctx.Namespace leaves the namespace unset (kubectl then uses "default"); callers
// that want the context's own namespace pass it.
func TerminalConfig(ctx Context) ([]byte, error) {
	for _, f := range []struct{ what, value string }{
		{"context", ctx.Name},
		{"cluster", ctx.Cluster},
		{"user", ctx.User},
	} {
		if err := validateName(f.what, f.value); err != nil {
			return nil, err
		}
	}
	if ctx.Namespace != "" {
		if errs := validation.IsDNS1123Label(ctx.Namespace); len(errs) > 0 {
			return nil, fmt.Errorf("terminal kubeconfig: namespace %q is not a valid namespace name: %s",
				ctx.Namespace, strings.Join(errs, "; "))
		}
	}

	// Hand-written rather than marshalled so that the file holds exactly these keys and nothing
	// else (a marshalled clientcmd Config adds empty clusters, users and preferences). Every value
	// is a JSON string, which is a valid YAML double-quoted scalar, so names such as "yes", "null",
	// "123" or "a: b" stay strings.
	var b strings.Builder
	b.WriteString("apiVersion: v1\n")
	b.WriteString("kind: Config\n")
	b.WriteString("current-context: " + quote(ctx.Name) + "\n")
	b.WriteString("contexts:\n")
	b.WriteString("- name: " + quote(ctx.Name) + "\n")
	b.WriteString("  context:\n")
	b.WriteString("    cluster: " + quote(ctx.Cluster) + "\n")
	b.WriteString("    user: " + quote(ctx.User) + "\n")
	if ctx.Namespace != "" {
		b.WriteString("    namespace: " + quote(ctx.Namespace) + "\n")
	}
	return []byte(b.String()), nil
}

// validateName rejects names that cannot come from a well-formed kubeconfig: empty, overlong,
// invalid UTF-8 or containing control characters.
func validateName(what, name string) error {
	switch {
	case name == "":
		return fmt.Errorf("terminal kubeconfig: %s name is empty", what)
	case len(name) > maxNameLen:
		return fmt.Errorf("terminal kubeconfig: %s name is longer than %d bytes", what, maxNameLen)
	case !utf8.ValidString(name):
		return fmt.Errorf("terminal kubeconfig: %s name is not valid UTF-8", what)
	case strings.IndexFunc(name, unicode.IsControl) >= 0:
		return fmt.Errorf("terminal kubeconfig: %s name %q contains a control character", what, name)
	}
	return nil
}

// quote returns s as a JSON string literal, which YAML reads as a double-quoted scalar.
func quote(s string) string {
	q, err := json.Marshal(s)
	if err != nil { // cannot happen for a string; keep the file well-formed regardless
		return `""`
	}
	return string(q)
}
