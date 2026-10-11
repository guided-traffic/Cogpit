// Package kubeconfig parses kubeconfig content: the contexts it defines and the per-terminal
// kubeconfig that binds a terminal to one context.
//
// Layer L0: no I/O. Functions take bytes and return values; reading files is package cluster's
// job. Kubeconfig content is untrusted input and must never panic the parser.
package kubeconfig

import "errors"

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

// Contexts returns the contexts defined in one kubeconfig document, sorted by name.
func Contexts(data []byte) ([]Context, error) {
	_ = data
	return nil, ErrNotImplemented
}
